package validators

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ovav/ovav/internal/sbom"
	"github.com/ovav/ovav/internal/truststore"
)

func TestModelPolicyUsesCanonicalModelGroups(t *testing.T) {
	tests := []struct {
		name   string
		model  string
		status string
	}{
		{name: "OpenAI allowed", model: "openai/gpt-5.6-luna", status: "pass"},
		{name: "MiniMax allowed", model: "minimax-coding-plan/MiniMax-M3", status: "pass"},
		{name: "unknown denied", model: "unknown/retired-model", status: "fail"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := tempRepoWithFiles(t, map[string]string{
				".ovav/policy/permission_authority.json": `{"model_groups":{"standard":{"models":["openai/gpt-5.6-luna","minimax-coding-plan/MiniMax-M3"]}}}`,
				"opencode.json":                          `{"model":"` + test.model + `","small_model":"` + test.model + `","agent":{"worker":{"model":"` + test.model + `"}}}`,
			})

			result := NewModelPolicy().Validate(context.Background(), root)
			if result.Status != test.status {
				t.Fatalf("expected %s, got %s: %v", test.status, result.Status, result.Issues)
			}
		})
	}
}

func TestFeedbackLoopUsesFunctionalGoImplementations(t *testing.T) {
	root := tempRepoWithFiles(t, map[string]string{
		"go-runtime/internal/agents/belief.go": `package agents
type BeliefManager struct{}
func (b *BeliefManager) AddBelief() {}
func (b *BeliefManager) DeprecateBelief() {}
func (b *BeliefManager) AddBeliefWithState() error { return nil }
func (b *BeliefManager) DeprecateStaleEmergent() {}
func (b *BeliefManager) DeprecateStaleEmergentAt() { delete(map[string]int{}, "stale") }
`,
		"go-runtime/internal/memory/governor.go": `package memory
type Governor struct{}
func (g *Governor) Write() error { g.classifier.Classify(card); g.ledger.UpsertCard(card); return g.ledger.Save() }
func (g *Governor) SessionPack() {}
`,
	})

	result := NewFeedbackLoop().Validate(context.Background(), root)
	if result.Status != "pass" {
		t.Fatalf("expected Go feedback loop to pass, got %s: %v", result.Status, result.Issues)
	}
	assertNoIssueContains(t, result.Issues, ".py")
}

func TestSingleAuthorityGeneratedHandoffClaimIsWarning(t *testing.T) {
	root := tempRepoWithFiles(t, map[string]string{
		".ovav/plan/caps.yaml":             "# canonical source\n",
		".ovav/context/CURRENT_HANDOFF.md": "GENERADO DESDE git HEAD\nfuente canónica del sistema\n",
	})

	result := NewSingleAuthority().Validate(context.Background(), root)
	if result.Status != "warn" {
		t.Fatalf("expected generated handoff warning, got %s: %v", result.Status, result.Issues)
	}
}

func TestSupplyChainValidationIsPureAndClassifiesWorkingTreeDrift(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	writeTestFile(t, root, "go-runtime/go.mod", "module test\n")
	writeTestFile(t, root, "go-runtime/go.sum", "sum\n")
	runGitTest(t, root, "add", "go-runtime/go.mod", "go-runtime/go.sum")
	runGitTest(t, root, "commit", "-m", "add module")

	baseline, err := sbom.Generate(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := baseline.Save(root); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(root, sbom.SBOMRegistry))
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, "go-runtime/go.mod", "module changed\n")
	writeTestFile(t, root, "go-runtime/new.go", "package changed\n")

	result := NewSupplyChain().Validate(context.Background(), root)
	if result.Status != "warn" {
		t.Fatalf("expected working tree drift warning, got %s: %v", result.Status, result.Issues)
	}
	assertIssueContains(t, result.Issues, "working_tree_drift")
	after, err := os.ReadFile(filepath.Join(root, sbom.SBOMRegistry))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("validation mutated SBOM baseline")
	}
}

func TestSupplyChainGateFailsSensitiveCandidateButDeveloperModeWarns(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	writeTestFile(t, root, "go-runtime/go.mod", "module test\n")
	writeTestFile(t, root, "go-runtime/go.sum", "sum\n")
	writeTestFile(t, root, ".ovav/policy/permission_authority.json", "{}\n")
	runGitTest(t, root, "add", "go-runtime/go.mod", "go-runtime/go.sum", ".ovav/policy/permission_authority.json")
	runGitTest(t, root, "commit", "-m", "baseline")
	baseline, err := sbom.Generate(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := baseline.Save(root); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, ".ovav/policy/permission_authority.json", "{\"changed\":true}\n")

	if got := NewSupplyChain(ValidationDeveloper).Validate(context.Background(), root); got.Status != "warn" {
		t.Fatalf("developer status = %s, want warn: %v", got.Status, got.Issues)
	}
	if got := NewSupplyChain(ValidationGate).Validate(context.Background(), root); got.Status != "fail" {
		t.Fatalf("gate status = %s, want fail: %v", got.Status, got.Issues)
	}
}

func TestSensitiveCandidateDriftIncludesRootOpenCodeConfig(t *testing.T) {
	if !IsSensitiveCandidateDrift("MODIFIED: opencode.json") {
		t.Fatal("root opencode.json drift must be sensitive in gate mode")
	}
}

func TestSupplyChainFailsStaleBaselineEntryStillInScope(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	writeTestFile(t, root, "go-runtime/go.mod", "module test\n")
	writeTestFile(t, root, "go-runtime/go.sum", "sum\n")
	baseline := &sbom.SBOM{CoreFiles: map[string]string{
		"go-runtime/removed.go": strings.Repeat("0", 64),
	}}
	if err := baseline.Save(root); err != nil {
		t.Fatal(err)
	}

	result := NewSupplyChain().Validate(context.Background(), root)
	if result.Status != "fail" {
		t.Fatalf("expected invalid baseline failure, got %s: %v", result.Status, result.Issues)
	}
	assertIssueContains(t, result.Issues, "baseline_invalid")
}

func TestSupplyChainTreatsCurrentGoSumDeletionAsWorktreeDrift(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	writeTestFile(t, root, "go-runtime/go.mod", "module test\n")
	writeTestFile(t, root, "go-runtime/go.sum", "sum\n")
	runGitTest(t, root, "add", "go-runtime/go.mod", "go-runtime/go.sum")
	runGitTest(t, root, "commit", "-m", "add module")
	baseline, err := sbom.Generate(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := baseline.Save(root); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "go-runtime", "go.sum")); err != nil {
		t.Fatal(err)
	}

	result := NewSupplyChain().Validate(context.Background(), root)
	if result.Status != "warn" {
		t.Fatalf("expected worktree warning, got %s: %v", result.Status, result.Issues)
	}
	assertIssueContains(t, result.Issues, "WORKTREE_DELETED: go-runtime/go.sum")
}

func TestGitPushRecognizesGoNativeWiringAndWarnsOnDirtyWorktree(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	runGitTest(t, root, "checkout", "-b", "feature/test")
	runGitTest(t, root, "remote", "add", "origin", "https://github.com/ovav/test.git")
	writeTestFile(t, root, "clients/opencode/agents/area-platform-engineering.md", "raw git push prohibited; force push prohibited\n")
	writeTestFile(t, root, "opencode.json", `{"permission":{"bash":{"git push*":"deny","git push --force *":"deny"}}}`)
	writeTestFile(t, root, "go-runtime/cmd/ovav/push_cli.go", "package main\nfunc cmdPush() { gitflow.Push() }\n")
	writeTestFile(t, root, "go-runtime/cmd/ovav/dispatch.go", "package main\nfunc dispatch() { casePush := \"push\"; _ = casePush; cmdPush() }\n")

	result := NewGitPush().Validate(context.Background(), root)
	if result.Status != "warn" {
		t.Fatalf("expected dirty worktree warning, got %s: %v", result.Status, result.Issues)
	}
	assertNoIssueContains(t, result.Issues, "wiring")
	assertIssueContains(t, result.Issues, "Uncommitted changes")
}

func TestRuntimeIntegrityMissingBaselineWarnsWithoutWriting(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	for _, rel := range coreFiles {
		writeTestFile(t, root, rel, rel+"\n")
	}
	runGitTest(t, root, "add", "AGENTS.md", "opencode.json", ".ovav/policy/permission_authority.json", ".ovav/plan/caps.yaml", "go-runtime/go.mod", "go-runtime/internal/validators/cmd/validate/main.go")
	runGitTest(t, root, "commit", "-m", "add core")
	writeTestFile(t, root, "opencode.json", "feature work\n")

	result := NewRuntimeIntegrity().Validate(context.Background(), root)
	if result.Status != "warn" {
		t.Fatalf("expected missing baseline/feature drift warning, got %s: %v", result.Status, result.Issues)
	}
	if _, err := os.Stat(baselinePath(root)); !os.IsNotExist(err) {
		t.Fatal("validation created an integrity baseline")
	}
}

func TestRuntimeIntegrityGateFailsMissingBaselineAndDrift(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	for _, rel := range coreFiles {
		writeTestFile(t, root, rel, rel+"\n")
	}
	runGitTest(t, root, "add", "-A")
	runGitTest(t, root, "commit", "-m", "core")
	if result := NewRuntimeIntegrity(ValidationGate).Validate(context.Background(), root); result.Status != "fail" {
		t.Fatalf("missing baseline gate status = %s, want fail", result.Status)
	}
	createIntegrityBaseline(t, root)
	writeTestFile(t, root, "opencode.json", "drift\n")
	if result := NewRuntimeIntegrity(ValidationGate).Validate(context.Background(), root); result.Status != "fail" {
		t.Fatalf("drift gate status = %s, want fail: %v", result.Status, result.Issues)
	}
}

func TestIntegrityBaselinePlanIsDeterministicAndDoesNotWrite(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	for _, rel := range coreFiles {
		writeTestFile(t, root, rel, rel+"\n")
	}
	runGitTest(t, root, "add", "-A")
	runGitTest(t, root, "commit", "-m", "core")

	first, err := PlanIntegrityBaseline(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := PlanIntegrityBaseline(root)
	if err != nil {
		t.Fatal(err)
	}
	if string(first.JSON) != string(second.JSON) {
		t.Fatalf("baseline plan is not deterministic:\n%s\n%s", first.JSON, second.JSON)
	}
	if first.Schema != IntegrityBaselineSchema || first.Algorithm != "sha256" || len(first.Files) != len(coreFiles) {
		t.Fatalf("unexpected baseline plan: %#v", first)
	}
	if _, err := os.Stat(baselinePath(root)); !os.IsNotExist(err) {
		t.Fatal("baseline plan wrote a file")
	}
}

func TestWriteIntegrityBaselineRequiresFeatureWorktree(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	for _, rel := range coreFiles {
		writeTestFile(t, root, rel, rel+"\n")
	}
	runGitTest(t, root, "add", "-A")
	runGitTest(t, root, "commit", "-m", "core")

	if _, err := WriteIntegrityBaseline(root); err == nil || !strings.Contains(err.Error(), "worktree") {
		t.Fatalf("main workspace write error = %v, want worktree rejection", err)
	}
	if _, err := os.Stat(baselinePath(root)); !os.IsNotExist(err) {
		t.Fatal("unsafe baseline write created a file")
	}
}

func TestWriteIntegrityBaselineFromCleanFeatureWorktree(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	for _, rel := range coreFiles {
		writeTestFile(t, root, rel, rel+"\n")
	}
	runGitTest(t, root, "add", "-A")
	runGitTest(t, root, "commit", "-m", "core")

	worktree := filepath.Join(t.TempDir(), "feature-baseline")
	runGitTest(t, root, "worktree", "add", "-b", "feature/baseline", worktree)
	writeTestFile(t, worktree, "candidate.txt", "committed candidate\n")
	runGitTest(t, worktree, "add", "--", "candidate.txt")
	runGitTest(t, worktree, "commit", "-m", "candidate")
	baseline, err := WriteIntegrityBaseline(worktree)
	if err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(baselinePath(worktree))
	if err != nil {
		t.Fatal(err)
	}
	if string(written) != string(baseline.JSON) {
		t.Fatalf("written baseline differs from plan:\n%s\n%s", written, baseline.JSON)
	}
}

func TestWriteIntegrityBaselineRejectsFeatureBranchWithoutCandidate(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	for _, rel := range coreFiles {
		writeTestFile(t, root, rel, rel+"\n")
	}
	runGitTest(t, root, "add", "-A")
	runGitTest(t, root, "commit", "-m", "core")

	worktree := filepath.Join(t.TempDir(), "feature-no-candidate")
	runGitTest(t, root, "worktree", "add", "-b", "feature/no-candidate", worktree)
	if _, err := WriteIntegrityBaseline(worktree); err == nil || !strings.Contains(err.Error(), "candidate") {
		t.Fatalf("empty feature branch error = %v, want candidate rejection", err)
	}
}

func TestWriteIntegrityBaselineFromStagedFeatureCandidate(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	for _, rel := range coreFiles {
		writeTestFile(t, root, rel, rel+"\n")
	}
	runGitTest(t, root, "add", "-A")
	runGitTest(t, root, "commit", "-m", "core")

	worktree := filepath.Join(t.TempDir(), "feature-staged-baseline")
	runGitTest(t, root, "worktree", "add", "-b", "feature/staged-baseline", worktree)
	writeTestFile(t, worktree, coreFiles[0], "staged candidate\n")
	runGitTest(t, worktree, "add", "--", coreFiles[0])

	baseline, err := WriteIntegrityBaseline(worktree)
	if err != nil {
		t.Fatal(err)
	}
	if baseline.Files[coreFiles[0]] != digest([]byte("staged candidate\n")) {
		t.Fatal("baseline did not attest the staged candidate")
	}
}

func TestWriteIntegrityBaselineRejectsUnstagedFeatureChanges(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	for _, rel := range coreFiles {
		writeTestFile(t, root, rel, rel+"\n")
	}
	runGitTest(t, root, "add", "-A")
	runGitTest(t, root, "commit", "-m", "core")

	worktree := filepath.Join(t.TempDir(), "feature-unstaged-baseline")
	runGitTest(t, root, "worktree", "add", "-b", "feature/unstaged-baseline", worktree)
	writeTestFile(t, worktree, coreFiles[0], "unstaged candidate\n")

	if _, err := WriteIntegrityBaseline(worktree); err == nil || !strings.Contains(err.Error(), "unstaged") {
		t.Fatalf("unstaged candidate error = %v, want rejection", err)
	}
	if _, err := os.Stat(baselinePath(worktree)); !os.IsNotExist(err) {
		t.Fatal("unsafe baseline write created a file")
	}
}

func createIntegrityBaseline(t *testing.T, root string) {
	t.Helper()
	baseline := IntegrityBaseline{Schema: IntegrityBaselineSchema, Algorithm: "sha256", Files: map[string]string{}}
	for _, rel := range coreFiles {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		baseline.Files[rel] = digest(data)
	}
	data, err := json.Marshal(baseline)
	if err != nil {
		t.Fatal(err)
	}
	path := baselinePath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGateSelfProtectionWarnsForTrackedFeatureChangeWithoutUpdatingHash(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	writeTestFile(t, root, gateFile, "package validators\n")
	runGitTest(t, root, "add", gateFile)
	runGitTest(t, root, "commit", "-m", "add gate")
	state := truststore.GateState{GateSHA256: fileDigest([]byte("package validators\n"))}
	stateData, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, ".ovav/runtime/gate_state.json", string(stateData))
	writeTestFile(t, root, gateFile, "package validators\n// scoped feature change\n")

	result := NewGateSelfProtection().Validate(context.Background(), root)
	if result.Status != "warn" {
		t.Fatalf("expected tracked feature change warning, got %s: %v", result.Status, result.Issues)
	}
	after := truststore.ReadGateState(root)
	if after.GateSHA256 != state.GateSHA256 {
		t.Fatal("validation updated gate hash")
	}
}

func TestGateSelfProtectionFailsUntrackedGate(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	writeTestFile(t, root, gateFile, "package validators\n")

	result := NewGateSelfProtection().Validate(context.Background(), root)
	if result.Status != "fail" {
		t.Fatalf("expected untracked protected gate failure, got %s: %v", result.Status, result.Issues)
	}
}

func runGitTest(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func assertIssueContains(t *testing.T, issues []string, want string) {
	t.Helper()
	for _, issue := range issues {
		if strings.Contains(issue, want) {
			return
		}
	}
	t.Fatalf("expected issue containing %q, got %v", want, issues)
}

func fileDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
