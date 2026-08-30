package main

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/ovav/ovav/internal/install"
	"github.com/ovav/ovav/internal/validators"
)

type pushTestValidator struct {
	id     string
	status string
	calls  *[]string
}

func (v pushTestValidator) ID() string          { return v.id }
func (v pushTestValidator) Name() string        { return v.id }
func (v pushTestValidator) Description() string { return v.id }
func (v pushTestValidator) Weight() int         { return 1 }
func (v pushTestValidator) Validate(context.Context, string) validators.Result {
	*v.calls = append(*v.calls, v.id)
	return validators.Result{ID: v.id, Name: v.id, Status: v.status}
}

func TestGovernedPushValidatorsIncludeGateModeIntegrity(t *testing.T) {
	validatorsList := governedPushValidators()
	var supply, integrity bool
	for _, validator := range validatorsList {
		switch typed := validator.(type) {
		case *validators.SupplyChain:
			supply = typed.Mode() == validators.ValidationGate
		case *validators.RuntimeIntegrity:
			integrity = typed.Mode() == validators.ValidationGate
		}
	}
	if !supply || !integrity {
		t.Fatalf("governed push validators missing gate mode: supply=%v integrity=%v", supply, integrity)
	}
}

func TestRunPushPreflightBlocksFailure(t *testing.T) {
	var calls []string
	results, ok := runPushPreflight(context.Background(), t.TempDir(), []validators.Validator{
		pushTestValidator{id: "existing", status: "pass", calls: &calls},
		pushTestValidator{id: "supply", status: "fail", calls: &calls},
		pushTestValidator{id: "integrity", status: "pass", calls: &calls},
	})
	if ok || len(results) != 3 || strings.Join(calls, ",") != "existing,supply,integrity" {
		t.Fatalf("runPushPreflight() ok=%v results=%d calls=%v", ok, len(results), calls)
	}
}

func TestParseIntegrityArgsDefaultsToPlanAndRequiresExplicitWrite(t *testing.T) {
	// parseIntegrityArgs now operates on flags only — the `baseline`/`gate`
	// subcommand prefix is stripped by cmdIntegrity's dispatcher.
	options, err := parseIntegrityArgs([]string{})
	if err != nil || options.write {
		t.Fatalf("default integrity options=%+v err=%v", options, err)
	}
	options, err = parseIntegrityArgs([]string{"--write"})
	if err != nil || !options.write {
		t.Fatalf("write integrity options=%+v err=%v", options, err)
	}
	if _, err := parseIntegrityArgs([]string{"--apply"}); err == nil {
		t.Fatal("unknown integrity write option accepted")
	}
}

func TestParseWindowsTerminalPlanArgsIsDryRunOnly(t *testing.T) {
	options, err := parseWindowsTerminalPlanArgs([]string{"windows", "plan", "--settings", "settings.json", "--fragment", "ovav.fragment.json"})
	if err != nil || options.settings != "settings.json" || options.fragment != "ovav.fragment.json" {
		t.Fatalf("terminal plan options=%+v err=%v", options, err)
	}
	if _, err := parseWindowsTerminalPlanArgs([]string{"windows", "plan", "--settings", "settings.json", "--fragment", "fragment.json", "--apply"}); err == nil {
		t.Fatal("terminal planner accepted a global apply")
	}
}

func TestParseValidateArgsSupportsExplicitGateMode(t *testing.T) {
	mode, target, err := parseValidateArgs([]string{"--gate", "runtime_integrity"}, validators.ValidationDeveloper)
	if err != nil || mode != validators.ValidationGate || target != "runtime_integrity" {
		t.Fatalf("parseValidateArgs() mode=%s target=%q err=%v", mode, target, err)
	}
	if _, _, err := parseValidateArgs([]string{"--gate", "all", "extra"}, validators.ValidationDeveloper); err == nil {
		t.Fatal("extra validate argument accepted")
	}
}

func TestParsePushArgsRejectsGovernanceBypasses(t *testing.T) {
	for _, arg := range []string{"--skip-validate", "--no-validate", "--force", "-f", "--force-with-lease"} {
		t.Run(arg, func(t *testing.T) {
			_, err := parsePushArgs([]string{arg})
			if err == nil || !strings.Contains(err.Error(), "prohibited") {
				t.Fatalf("parsePushArgs(%q) error = %v, want prohibited", arg, err)
			}
		})
	}
}

func TestParsePushArgsAcceptsGovernedOptionsOnly(t *testing.T) {
	opts, err := parsePushArgs([]string{"--dry-run", "--remote=upstream"})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.dryRun || opts.remote != "upstream" {
		t.Fatalf("parsePushArgs() = %+v", opts)
	}
}

func TestValidationModeForProtectedAndWorktreeBranches(t *testing.T) {
	for _, test := range []struct {
		branch string
		want   validators.ValidationMode
	}{
		{branch: "main", want: validators.ValidationGate},
		{branch: "release/v1", want: validators.ValidationGate},
		{branch: "feature/test", want: validators.ValidationDeveloper},
	} {
		t.Run(test.branch, func(t *testing.T) {
			root := t.TempDir()
			cmd := exec.Command("git", "init", "-q", "-b", test.branch)
			cmd.Dir = root
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("git init: %v: %s", err, output)
			}
			if got := validationModeForRepo(root); got != test.want {
				t.Errorf("validationModeForRepo() = %s, want %s", got, test.want)
			}
		})
	}
}

func TestParseSyncArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantStep    string
		wantPlan    bool
		wantDry     bool
		wantHost    string
		wantRoll    string
		wantWin     string
		wantApply   bool
		wantApprove bool
		wantErr     bool
	}{
		{name: "plan json", args: []string{"--plan-json"}, wantPlan: true},
		{name: "step plan", args: []string{"--skills", "--plan-json"}, wantStep: "skills", wantPlan: true},
		{name: "dry run", args: []string{"--dry-run"}, wantDry: true},
		{name: "unknown flag", args: []string{"--typo"}, wantErr: true},
		{name: "unknown positional", args: []string{"agents"}, wantErr: true},
		{name: "conflicting steps", args: []string{"--agents", "--skills"}, wantErr: true},
		{name: "host plan", args: []string{"--host-profile", "opencode-bootstrap"}, wantHost: "opencode-bootstrap"},
		{name: "windows host apply", args: []string{"--host-profile", "wsl2-resource-policy", "--windows-home", "/tmp/windows", "--apply", "--approve-host-write"}, wantHost: "wsl2-resource-policy", wantWin: "/tmp/windows", wantApply: true, wantApprove: true},
		{name: "rollback", args: []string{"--rollback-journal", "/tmp/home/.local/state/ovav/host-projection/a.journal.json", "--approve-host-write"}, wantRoll: "/tmp/home/.local/state/ovav/host-projection/a.journal.json", wantApprove: true},
		{name: "apply lacks approval", args: []string{"--host-profile", "opencode-bootstrap", "--apply"}, wantErr: true},
		{name: "rollback lacks approval", args: []string{"--rollback-journal", "/tmp/journal"}, wantErr: true},
		{name: "unknown host profile", args: []string{"--host-profile", "other"}, wantErr: true},
		{name: "host and rollback conflict", args: []string{"--host-profile", "opencode-bootstrap", "--rollback-journal", "/tmp/journal", "--approve-host-write"}, wantErr: true},
		{name: "host and legacy conflict", args: []string{"--host-profile", "opencode-bootstrap", "--agents"}, wantErr: true},
		{name: "windows home on non-windows profile", args: []string{"--host-profile", "opencode-bootstrap", "--windows-home", "/tmp/windows"}, wantErr: true},
		{name: "windows profile lacks home", args: []string{"--host-profile", "warp-wsl-tab"}, wantErr: true},
		{name: "missing host profile value", args: []string{"--host-profile"}, wantErr: true},
		{name: "empty host profile value", args: []string{"--host-profile", ""}, wantErr: true},
		{name: "duplicate host profile", args: []string{"--host-profile", "opencode-bootstrap", "--host-profile", "opencode-bootstrap"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := parseSyncArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseSyncArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if opts.step != tt.wantStep || opts.planJSON != tt.wantPlan || opts.dryRun != tt.wantDry ||
				opts.hostProfile != tt.wantHost || opts.rollbackJournal != tt.wantRoll || opts.windowsHome != tt.wantWin ||
				opts.apply != tt.wantApply || opts.approveHostWrite != tt.wantApprove {
				t.Errorf("parseSyncArgs() = %+v", opts)
			}
		})
	}
}

func TestBuildSyncPlan(t *testing.T) {
	tests := []struct {
		name      string
		step      string
		wantSteps int
	}{
		{name: "all", wantSteps: 6},
		{name: "agents", step: "agents", wantSteps: 2},
		{name: "skills", step: "skills", wantSteps: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := buildSyncPlan("/tmp/repo", tt.step)
			if plan["writes_performed"] != false {
				t.Error("sync plan must report no writes")
			}
			steps, ok := plan["steps"].([]string)
			if !ok || len(steps) != tt.wantSteps {
				t.Fatalf("steps = %#v, want %d", plan["steps"], tt.wantSteps)
			}
		})
	}
}

func TestSecurityAliasAcceptsStatusFlags(t *testing.T) {
	if code := cmdDefend([]string{"--json"}); code != 0 {
		t.Fatalf("cmdDefend(--json) = %d, want 0", code)
	}
}

func TestParseRecoveryArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantMode install.Mode
		wantPlan bool
		wantErr  bool
	}{
		{name: "default dry run", wantMode: install.ModeDryRun},
		{name: "explicit plan", args: []string{"--plan"}, wantMode: install.ModeDryRun, wantPlan: true},
		{name: "plan json", args: []string{"--plan", "--json"}, wantMode: install.ModeDryRun, wantPlan: true},
		{name: "apply", args: []string{"--apply"}, wantMode: install.ModeSourceLocalApply},
		{name: "mode sandbox", args: []string{"--mode", "sandbox"}, wantMode: install.ModeSandbox},
		{name: "plan apply conflict", args: []string{"--plan", "--apply"}, wantErr: true},
		{name: "unknown flag", args: []string{"--typo"}, wantErr: true},
		{name: "missing mode value", args: []string{"--mode"}, wantErr: true},
		{name: "unknown positional", args: []string{"now"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := parseRecoveryArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseRecoveryArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if opts.mode != tt.wantMode || opts.plan != tt.wantPlan {
				t.Errorf("parseRecoveryArgs() = %+v", opts)
			}
		})
	}
}

func TestRequiredGateExitCode(t *testing.T) {
	tests := []struct {
		name   string
		result map[string]interface{}
		field  string
		want   int
	}{
		{name: "passed", result: map[string]interface{}{"passed": true}, field: "passed", want: 0},
		{name: "failed", result: map[string]interface{}{"passed": false}, field: "passed", want: 1},
		{name: "missing field", result: map[string]interface{}{}, field: "passed", want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := requiredGateExitCode(tt.result, tt.field); got != tt.want {
				t.Errorf("requiredGateExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParseSmokeArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantTimeout time.Duration
		wantErr     bool
	}{
		{name: "default", wantTimeout: defaultSourceSmokeTimeout},
		{name: "timeout", args: []string{"--timeout", "5s"}, wantTimeout: 5 * time.Second},
		{name: "timeout equals", args: []string{"--timeout=2m"}, wantTimeout: 2 * time.Minute},
		{name: "unknown", args: []string{"--typo"}, wantErr: true},
		{name: "invalid timeout", args: []string{"--timeout", "forever"}, wantErr: true},
		{name: "unbounded timeout", args: []string{"--timeout", "0s"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := parseSmokeArgs(tt.args, false)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseSmokeArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && opts.timeout != tt.wantTimeout {
				t.Errorf("timeout = %s, want %s", opts.timeout, tt.wantTimeout)
			}
		})
	}
}

func TestCEORequiredAliasesAreKnown(t *testing.T) {
	for _, command := range []string{"security", "smoke"} {
		t.Run(command, func(t *testing.T) {
			if !isKnownCommand(command) {
				t.Errorf("%q must be a known command", command)
			}
		})
	}
}
