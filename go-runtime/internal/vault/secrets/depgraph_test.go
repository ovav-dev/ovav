package secrets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ── DependencyGraph Tests ───────────────────────────────────────────────────────

func TestDependencyGraph_AddRef(t *testing.T) {
	g := &DependencyGraph{refs: make(map[string][]SecretRef)}

	// Add a reference
	err := g.AddRef("sec-1", SystemGitHubActions, "GitHub Actions: ovav-dev/ovav", "CF_API_TOKEN")
	if err != nil {
		t.Fatalf("AddRef: %v", err)
	}

	refs := g.GetRefs("sec-1")
	if len(refs) != 1 {
		t.Fatalf("GetRefs: got %d refs, want 1", len(refs))
	}

	if refs[0].System != SystemGitHubActions {
		t.Errorf("System = %v, want %v", refs[0].System, SystemGitHubActions)
	}
	if refs[0].EnvVar != "CF_API_TOKEN" {
		t.Errorf("EnvVar = %q, want %q", refs[0].EnvVar, "CF_API_TOKEN")
	}
}

func TestDependencyGraph_AddRef_Deduplication(t *testing.T) {
	g := &DependencyGraph{refs: make(map[string][]SecretRef)}

	// Same ref added twice should be deduplicated
	g.AddRef("sec-1", SystemGitHubActions, "GitHub Actions: ovav-dev/ovav", "TOKEN")
	g.AddRef("sec-1", SystemGitHubActions, "GitHub Actions: ovav-dev/ovav", "TOKEN")

	refs := g.GetRefs("sec-1")
	if len(refs) != 1 {
		t.Errorf("Duplicate ref: got %d refs, want 1 (deduped)", len(refs))
	}
}

func TestDependencyGraph_RemoveRef(t *testing.T) {
	g := &DependencyGraph{refs: make(map[string][]SecretRef)}
	g.AddRef("sec-1", SystemGitHubActions, "GitHub Actions: ovav-dev/ovav", "TOKEN")

	err := g.RemoveRef("sec-1", string(SystemGitHubActions), "GitHub Actions: ovav-dev/ovav")
	if err != nil {
		t.Fatalf("RemoveRef: %v", err)
	}

	refs := g.GetRefs("sec-1")
	if len(refs) != 0 {
		t.Errorf("After RemoveRef: got %d refs, want 0", len(refs))
	}
}

func TestDependencyGraph_IsOrphan(t *testing.T) {
	g := &DependencyGraph{refs: make(map[string][]SecretRef)}

	if !g.IsOrphan("sec-nonexistent") {
		t.Error("IsOrphan for nonexistent: got false, want true")
	}

	g.AddRef("sec-1", SystemGitHubActions, "GitHub Actions: ovav-dev/ovav", "TOKEN")
	if g.IsOrphan("sec-1") {
		t.Error("IsOrphan for existing: got true, want false")
	}
}

func TestDependencyGraph_OrphanReport(t *testing.T) {
	g := &DependencyGraph{refs: make(map[string][]SecretRef)}

	// sec-1 has a ref; sec-2 has refs; sec-3 has no refs
	g.AddRef("sec-1", SystemGitHubActions, "GitHub Actions: o/r", "T1")
	g.AddRef("sec-2", SystemGitHubActions, "GitHub Actions: o/r", "T2")
	g.AddRef("sec-2", SystemFlyIO, "Fly.io app: myapp", "T2")
	// sec-3 has no refs — it is an orphan

	ids := []string{"sec-1", "sec-2", "sec-3", "sec-none"}
	orphans := g.OrphanReport(ids)

	// Only sec-3 and sec-none (unknown) should be orphans
	if len(orphans) != 2 {
		t.Errorf("OrphanReport: got %d orphans, want 2", len(orphans))
	}

	found := make(map[string]bool)
	for _, o := range orphans {
		found[o] = true
	}
	if !found["sec-3"] {
		t.Error("OrphanReport: missing sec-3 (no refs)")
	}
	if !found["sec-none"] {
		t.Error("OrphanReport: missing sec-none (unknown id)")
	}
	if found["sec-1"] {
		t.Error("OrphanReport: sec-1 should NOT be orphan (has refs)")
	}
	if found["sec-2"] {
		t.Error("OrphanReport: sec-2 should NOT be orphan (has refs)")
	}
}

func TestDependencyGraph_RotationImpact(t *testing.T) {
	g := &DependencyGraph{refs: make(map[string][]SecretRef)}

	g.AddRef("sec-1", SystemGitHubActions, "GitHub Actions: ovav-dev/ovav", "CF_TOKEN")
	g.AddRef("sec-1", SystemFlyIO, "Fly.io app: myapp", "CF_TOKEN")

	impact := g.RotationImpact("sec-1")
	if len(impact) != 2 {
		t.Fatalf("RotationImpact: got %d entries, want 2", len(impact))
	}

	// Both should be auto-rotatable (GitHub Actions and Fly.io)
	for _, entry := range impact {
		if !strings.Contains(entry, "rotatable") && !strings.Contains(entry, "auto-rotatable") {
			// The format is "system: path (envvar)"
			if !strings.Contains(entry, "GitHub") && !strings.Contains(entry, "Fly.io") {
				t.Errorf("Unexpected impact entry: %s", entry)
			}
		}
	}

	// Non-existent secret
	if g.RotationImpact("sec-nonexistent") != nil {
		t.Error("RotationImpact for nonexistent: got non-nil, want nil")
	}
}

func TestDependencyGraph_GetSecretsForSystem(t *testing.T) {
	g := &DependencyGraph{refs: make(map[string][]SecretRef)}

	g.AddRef("sec-1", SystemGitHubActions, "GitHub Actions: o/r", "T1")
	g.AddRef("sec-2", SystemGitHubActions, "GitHub Actions: o2/r2", "T2")
	g.AddRef("sec-3", SystemFlyIO, "Fly.io app: app", "T3")

	ghIDs := g.GetSecretsForSystem(SystemGitHubActions)
	if len(ghIDs) != 2 {
		t.Errorf("GetSecretsForSystem(GitHubActions): got %d, want 2", len(ghIDs))
	}

	flyIDs := g.GetSecretsForSystem(SystemFlyIO)
	if len(flyIDs) != 1 {
		t.Errorf("GetSecretsForSystem(FlyIO): got %d, want 1", len(flyIDs))
	}
}

func TestDependencyGraph_SaveLoad(t *testing.T) {
	tmpDir := t.TempDir()

	// Set HOME so depsGraphPath() writes to our temp dir
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	g := &DependencyGraph{refs: make(map[string][]SecretRef)}
	g.AddRef("sec-1", SystemGitHubActions, "GitHub Actions: o/r", "TOKEN")
	g.AddRef("sec-2", SystemFlyIO, "Fly.io app: app", "SECRET")

	err := g.Save()
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Load into a fresh graph
	g2, err := LoadDependencyGraph()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(g2.GetRefs("sec-1")) != 1 {
		t.Error("After save/load: sec-1 ref missing")
	}
	if len(g2.GetRefs("sec-2")) != 1 {
		t.Error("After save/load: sec-2 ref missing")
	}
}

func TestDependencyGraph_AutoRotatable(t *testing.T) {
	g := &DependencyGraph{refs: make(map[string][]SecretRef)}

	tests := []struct {
		system System
		want   bool
	}{
		{SystemGitHubActions, true},
		{SystemFlyIO, true},
		{SystemGitLabCI, true},
		{SystemCICD, true},
		{SystemLocalEnv, false},
		{SystemUnknown, false},
	}

	for _, tc := range tests {
		g.AddRef("sec-test", tc.system, "path", "VAR")
		refs := g.GetRefs("sec-test")
		if len(refs) == 0 {
			t.Fatalf("No refs for system %s", tc.system)
		}
		if refs[len(refs)-1].AutoRotatable != tc.want {
			t.Errorf("canAutoRotate(%s) = %v, want %v", tc.system, refs[len(refs)-1].AutoRotatable, tc.want)
		}
	}
}

func TestDetectSystemFromName(t *testing.T) {
	tests := []struct {
		name   string
		expect System
	}{
		{"GITHUB_TOKEN", SystemGitHubActions},
		{"FLY_API_TOKEN", SystemFlyIO},
		{"GITLAB_CI_TOKEN", SystemGitLabCI},
		{"JENKINS_API_KEY", SystemJenkins},
		{"CF_API_KEY", SystemOVAVAgent},
		{"RANDOM_KEY", SystemUnknown},
	}

	for _, tc := range tests {
		got := DetectSystemFromName(tc.name)
		if got != tc.expect {
			t.Errorf("DetectSystemFromName(%q) = %v, want %v", tc.name, got, tc.expect)
		}
	}
}

func TestDependencyGraph_DiscoverFromSecrets(t *testing.T) {
	g := &DependencyGraph{refs: make(map[string][]SecretRef)}
	store := NewSecretStore()

	sec1 := NewSecret("GH_TOKEN", TypeAPIToken, "github", "github", []byte("value"))
	sec2 := NewSecret("FLY_TOKEN", TypeAPIToken, "fly.io", "fly", []byte("value"))
	store.Add(sec1)
	store.Add(sec2)

	err := g.DiscoverFromSecrets(store)
	if err != nil {
		t.Fatalf("DiscoverFromSecrets: %v", err)
	}

	refs1 := g.GetRefs(sec1.ID)
	if len(refs1) == 0 {
		t.Error("DiscoverFromSecrets: expected refs for github-sourced secret")
	}
	refs2 := g.GetRefs(sec2.ID)
	if len(refs2) == 0 {
		t.Error("DiscoverFromSecrets: expected refs for fly-sourced secret")
	}
}

func TestQuerySecrets_Orphan(t *testing.T) {
	store := NewSecretStore()
	graph := &DependencyGraph{refs: make(map[string][]SecretRef)}

	sec := NewSecret("Orphaned", TypeAPIToken, "cloudflare", "manual", []byte("val"))
	store.Add(sec)
	// No refs added — it's an orphan

	results := QuerySecrets(store, graph, "orphan")
	if len(results) != 1 {
		t.Fatalf("QuerySecrets(orphan): got %d results, want 1", len(results))
	}
	if results[0].Name != "Orphaned" {
		t.Errorf("Name = %q, want %q", results[0].Name, "Orphaned")
	}
}

func TestQuerySecrets_NameMatch(t *testing.T) {
	store := NewSecretStore()
	graph := &DependencyGraph{refs: make(map[string][]SecretRef)}

	sec := NewSecret("CF_API_TOKEN", TypeAPIToken, "cloudflare", "manual", []byte("val"))
	store.Add(sec)

	results := QuerySecrets(store, graph, "cf")
	if len(results) == 0 {
		t.Fatal("QuerySecrets(name match): got 0 results, want at least 1")
	}
	if results[0].Name != "CF_API_TOKEN" {
		t.Errorf("Name = %q, want %q", results[0].Name, "CF_API_TOKEN")
	}
}

func TestQuerySecrets_GitHub(t *testing.T) {
	store := NewSecretStore()
	graph := &DependencyGraph{refs: make(map[string][]SecretRef)}

	sec := NewSecret("GH_TOKEN", TypeAPIToken, "github", "github", []byte("val"))
	store.Add(sec)

	results := QuerySecrets(store, graph, "github")
	if len(results) == 0 {
		t.Fatal("QuerySecrets(github): got 0 results, want 1")
	}
	if results[0].Name != "GH_TOKEN" {
		t.Errorf("Name = %q, want %q", results[0].Name, "GH_TOKEN")
	}
}

func TestRefID_Deterministic(t *testing.T) {
	id1 := refID("sec-1", "github-actions", "path/to/secret")
	id2 := refID("sec-1", "github-actions", "path/to/secret")
	id3 := refID("sec-1", "github-actions", "different/path")

	if id1 != id2 {
		t.Error("refID: same inputs should produce same output")
	}
	if id1 == id3 {
		t.Error("refID: different inputs should produce different output")
	}

	// Should be 32 hex chars (16 bytes → 32 hex chars)
	if len(id1) != 32 {
		t.Errorf("refID: len = %d, want 32", len(id1))
	}
}

func TestSecretRef_JSON(t *testing.T) {
	ref := SecretRef{
		ID:            "abc123",
		SecretID:      "sec-1",
		System:        SystemGitHubActions,
		Path:          "GitHub Actions: owner/repo",
		EnvVar:        "SECRET_TOKEN",
		AddedAt:       time.Now().UTC(),
		AutoRotatable: true,
	}

	data, err := json.Marshal(ref)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var ref2 SecretRef
	if err := json.Unmarshal(data, &ref2); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if ref2.ID != ref.ID {
		t.Errorf("ID = %q, want %q", ref2.ID, ref.ID)
	}
	if ref2.System != ref.System {
		t.Errorf("System = %v, want %v", ref2.System, ref.System)
	}
	if ref2.EnvVar != ref.EnvVar {
		t.Errorf("EnvVar = %q, want %q", ref2.EnvVar, ref.EnvVar)
	}
	if ref2.AutoRotatable != ref.AutoRotatable {
		t.Errorf("AutoRotatable = %v, want %v", ref2.AutoRotatable, ref.AutoRotatable)
	}
}

func TestDependencyGraph_GetSystemsUsing(t *testing.T) {
	g := &DependencyGraph{refs: make(map[string][]SecretRef)}

	g.AddRef("sec-1", SystemGitHubActions, "GitHub Actions: o/r", "T1")
	g.AddRef("sec-1", SystemFlyIO, "Fly.io app: myapp", "T1")
	g.AddRef("sec-1", SystemGitHubActions, "GitHub Actions: o2/r2", "T2") // duplicate system

	systems := g.GetSystemsUsing("sec-1")
	if len(systems) != 2 {
		t.Errorf("GetSystemsUsing: got %d systems, want 2", len(systems))
	}
}

func TestDependencyGraph_RefsForSecretByName(t *testing.T) {
	g := &DependencyGraph{refs: make(map[string][]SecretRef)}
	store := NewSecretStore()

	sec := NewSecret("TestSecret", TypeAPIToken, "github", "github", []byte("val"))
	store.Add(sec)
	g.AddRef(sec.ID, SystemGitHubActions, "GitHub Actions: o/r", "TEST_SECRET")

	refs := g.GetRefsForSecretByName(store, "TestSecret")
	if len(refs) != 1 {
		t.Fatalf("GetRefsForSecretByName: got %d refs, want 1", len(refs))
	}

	// Non-existent
	refs = g.GetRefsForSecretByName(store, "DoesNotExist")
	if refs != nil {
		t.Error("GetRefsForSecretByName for nonexistent: got non-nil, want nil")
	}
}

// ── QueryResult Tests ─────────────────────────────────────────────────────────

func TestQueryResult_JSON(t *testing.T) {
	qr := QueryResult{
		Icon:   "🔑",
		Name:   "CF_TOKEN",
		Type:   "api_token",
		Detail: "cloudflare production",
	}

	data, err := json.Marshal(qr)
	if err != nil {
		t.Fatalf("Marshal QueryResult: %v", err)
	}

	var qr2 QueryResult
	if err := json.Unmarshal(data, &qr2); err != nil {
		t.Fatalf("Unmarshal QueryResult: %v", err)
	}

	if qr2.Icon != qr.Icon || qr2.Name != qr.Name || qr2.Type != qr.Type {
		t.Error("QueryResult fields mismatch after round-trip")
	}
}

// ── Exported path override helper for tests ─────────────────────────────────────

func TestDepsGraphPath_EnvOverride(t *testing.T) {
	// Save and restore env
	orig := os.Getenv("HOME")
	defer os.Setenv("HOME", orig)

	os.Setenv("HOME", "/test/home")
	path := depsGraphPath()
	expected := filepath.Join("/test/home", ".local", "share", "ovav", "deps.graph")
	if path != expected {
		t.Errorf("depsGraphPath = %q, want %q", path, expected)
	}
}
