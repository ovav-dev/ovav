package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ── findRepoRoot ─────────────────────────────────────────────────────────────

func TestFindRepoRoot_WithGoMod(t *testing.T) {
	// OVAV mono-repo: requires go-runtime/go.mod at root (not bare go.mod)
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ovav", "service_areas"), 0755)
	os.MkdirAll(filepath.Join(root, "go-runtime"), 0755)
	os.WriteFile(filepath.Join(root, "go-runtime", "go.mod"), []byte("module test\n"), 0644)

	subdir := filepath.Join(root, "a", "b", "c")
	os.MkdirAll(subdir, 0755)

	found, err := findRepoRoot(subdir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != root {
		t.Errorf("expected root %q, got %q", root, found)
	}
}

func TestFindRepoRoot_WithGoRuntimeGoMod(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ovav"), 0755)
	os.MkdirAll(filepath.Join(root, ".ovav", "service_areas"), 0755)
	os.MkdirAll(filepath.Join(root, "go-runtime"), 0755)
	os.WriteFile(filepath.Join(root, "go-runtime", "go.mod"), []byte("module test\n"), 0644)

	found, err := findRepoRoot(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != root {
		t.Errorf("expected root %q, got %q", root, found)
	}
}

func TestFindRepoRoot_MissingOvav(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test\n"), 0644)

	_, err := findRepoRoot(root)
	if err == nil {
		t.Error("expected error without .ovav/ directory")
	}
}

func TestFindRepoRoot_MissingGoMod(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ovav"), 0755)

	_, err := findRepoRoot(root)
	if err == nil {
		t.Error("expected error without go.mod")
	}
}

func TestFindRepoRoot_NestedDeep(t *testing.T) {
	// OVAV mono-repo: requires go-runtime/go.mod at root
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ovav", "service_areas"), 0755)
	os.MkdirAll(filepath.Join(root, "go-runtime"), 0755)
	os.WriteFile(filepath.Join(root, "go-runtime", "go.mod"), []byte("module test\n"), 0644)

	deep := filepath.Join(root, "x", "y", "z", "w")
	os.MkdirAll(deep, 0755)

	found, err := findRepoRoot(deep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != root {
		t.Errorf("expected root %q, got %q", root, found)
	}
}

func TestFindRepoRoot_RootNotFound(t *testing.T) {
	tmp := t.TempDir()
	_, err := findRepoRoot(tmp)
	if err == nil {
		t.Error("expected error when repo root cannot be found")
	}
}

// ── cleanOldOutput ───────────────────────────────────────────────────────────

func TestCleanOldOutput_RemovesGeneratedFiles(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "agents")
	os.MkdirAll(agentDir, 0755)

	os.WriteFile(filepath.Join(agentDir, "area-security.md"), []byte("gen"), 0644)
	os.WriteFile(filepath.Join(agentDir, "lead-backend.md"), []byte("gen"), 0644)
	os.WriteFile(filepath.Join(agentDir, "ovav.md"), []byte("keep"), 0644)
	os.WriteFile(filepath.Join(agentDir, "notes.txt"), []byte("keep"), 0644)

	err := cleanOldOutput(agentDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(agentDir, "area-security.md")); !os.IsNotExist(err) {
		t.Error("area-security.md should have been removed")
	}
	if _, err := os.Stat(filepath.Join(agentDir, "lead-backend.md")); !os.IsNotExist(err) {
		t.Error("lead-backend.md should have been removed")
	}
	if _, err := os.Stat(filepath.Join(agentDir, "ovav.md")); err != nil {
		t.Error("ovav.md should have been preserved")
	}
	if _, err := os.Stat(filepath.Join(agentDir, "notes.txt")); err != nil {
		t.Error("notes.txt should have been preserved")
	}
}

func TestCleanOldOutput_NonexistentDir(t *testing.T) {
	dir := t.TempDir()
	err := cleanOldOutput(filepath.Join(dir, "nonexistent"))
	if err != nil {
		t.Errorf("expected nil error for nonexistent dir, got: %v", err)
	}
}

func TestCleanOldOutput_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "agents")
	os.MkdirAll(agentDir, 0755)

	err := cleanOldOutput(agentDir)
	if err != nil {
		t.Errorf("unexpected error on empty dir: %v", err)
	}
}

func TestCleanOldOutput_ReadDirError(t *testing.T) {
	tmp := t.TempDir()
	fakeFile := filepath.Join(tmp, "not_a_dir")
	os.WriteFile(fakeFile, []byte("x"), 0644)

	err := cleanOldOutput(fakeFile)
	if err == nil {
		t.Error("expected error when ReadDir is called on a file")
	}
}

func TestCleanOldOutput_RemoveError(t *testing.T) {
	tmp := t.TempDir()
	agentDir := filepath.Join(tmp, "agents")
	os.MkdirAll(agentDir, 0755)
	os.WriteFile(filepath.Join(agentDir, "area-x.md"), []byte("gen"), 0644)

	if err := os.Chmod(agentDir, 0555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer os.Chmod(agentDir, 0755)

	err := cleanOldOutput(agentDir)
	if err == nil {
		t.Error("expected error when os.Remove fails on read-only dir")
	}
}

// ── countGenerated ───────────────────────────────────────────────────────────

func TestCountGenerated(t *testing.T) {
	tests := []struct {
		name                            string
		files                           []string
		wantAreas, wantLeads, wantTeams int
	}{
		{
			name:      "empty",
			files:     nil,
			wantAreas: 0, wantLeads: 0, wantTeams: 0,
		},
		{
			name:      "mixed agents",
			files:     []string{"area-security.md", "lead-backend.md", "team-alpha.md", "ovav.md", "notes.txt"},
			wantAreas: 1, wantLeads: 1, wantTeams: 1,
		},
		{
			name:      "multiple areas",
			files:     []string{"area-security.md", "area-infra.md", "area-data.md"},
			wantAreas: 3, wantLeads: 0, wantTeams: 0,
		},
		{
			name:      "short names ignored",
			files:     []string{"a-.md", "l-.md", "t-.md", "area.md"},
			wantAreas: 0, wantLeads: 0, wantTeams: 0,
		},
		{
			name:      "all types multiple",
			files:     []string{"area-a.md", "area-b.md", "lead-x.md", "lead-y.md", "lead-z.md", "team-1.md"},
			wantAreas: 2, wantLeads: 3, wantTeams: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tt.files {
				os.WriteFile(filepath.Join(dir, f), []byte("x"), 0644)
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatalf("ReadDir: %v", err)
			}

			areas, leads, teams := countGenerated(entries)
			if areas != tt.wantAreas {
				t.Errorf("areas: got %d, want %d", areas, tt.wantAreas)
			}
			if leads != tt.wantLeads {
				t.Errorf("leads: got %d, want %d", leads, tt.wantLeads)
			}
			if teams != tt.wantTeams {
				t.Errorf("teams: got %d, want %d", teams, tt.wantTeams)
			}
		})
	}
}

// ── generateTarget ───────────────────────────────────────────────────────────

func createCanonicalDir(t *testing.T, root string) {
	t.Helper()
	// Canonical agents are now at go-runtime/internal/agents (mono-repo restructure)
	areas := filepath.Join(root, "go-runtime", "internal", "agents", "areas")
	leads := filepath.Join(root, "go-runtime", "internal", "agents", "leads")
	teams := filepath.Join(root, "go-runtime", "internal", "agents", "teams")
	os.MkdirAll(areas, 0755)
	os.MkdirAll(leads, 0755)
	os.MkdirAll(teams, 0755)

	areaYAML := `id: test-area
name: Test Area
description: A test area for unit tests
lead: test-lead
color: '#ff0000'
surface: Testing surface
functions:
- 'Test function one.'
limitations:
- 'No limitation.'
hard_stop: '🚫 HARD STOP'
`
	os.WriteFile(filepath.Join(areas, "area-test-area.yaml"), []byte(areaYAML), 0644)

	leadYAML := `id: test-lead
name: Test Lead
display_name: Test Lead Display
area: test_area
origin: Test
color: '#0000ff'
authority: test
functions:
- 'Lead function one.'
limitations:
- 'No limitation.'
hard_stop: '🚫 HARD STOP'
squad: []
`
	os.WriteFile(filepath.Join(leads, "lead-test-lead.yaml"), []byte(leadYAML), 0644)

	teamYAML := `id: test-team
name: Test Team
description: A test team member
area: test_area
lead: test-lead
country: Test
function: Testing
actions:
- 'Action one.'
hard_stop: '🚫 HARD STOP'
`
	os.WriteFile(filepath.Join(teams, "team-test-team.yaml"), []byte(teamYAML), 0644)
}

func TestGenerateTarget_Opencode(t *testing.T) {
	root := t.TempDir()
	createCanonicalDir(t, root)

	areas, leads, teams, err := generateTarget(
		filepath.Join(root, "go-runtime", "internal", "agents"),
		root,
		"opencode",
		"",
	)
	if err != nil {
		t.Fatalf("generateTarget failed: %v", err)
	}
	if areas == 0 {
		t.Error("expected at least 1 area generated")
	}
	// opencode now generates full hierarchy (AreasOnly=false) for GAP-1 converter fix
	if leads == 0 {
		t.Errorf("expected leads for opencode (AreasOnly=false), got 0")
	}
	if teams == 0 {
		t.Errorf("expected teams for opencode (AreasOnly=false), got 0")
	}
	if areas+leads+teams < 3 {
		t.Errorf("expected at least 3 files (1 area + 1 lead + 1 team), got %d", areas+leads+teams)
	}
}

func TestGenerateTarget_ClaudeCode(t *testing.T) {
	root := t.TempDir()
	createCanonicalDir(t, root)

	areas, leads, teams, err := generateTarget(
		filepath.Join(root, "go-runtime", "internal", "agents"),
		root,
		"claude-code",
		"",
	)
	if err != nil {
		t.Fatalf("generateTarget failed: %v", err)
	}
	if areas == 0 {
		t.Error("expected at least 1 area generated")
	}
	if leads == 0 {
		t.Error("expected at least 1 lead generated for claude-code")
	}
	if teams == 0 {
		t.Error("expected at least 1 team generated for claude-code")
	}
}

func TestGenerateTarget_Mimocode(t *testing.T) {
	root := t.TempDir()
	createCanonicalDir(t, root)

	areas, leads, teams, err := generateTarget(
		filepath.Join(root, "go-runtime", "internal", "agents"),
		root,
		"mimocode",
		"",
	)
	if err != nil {
		t.Fatalf("generateTarget failed: %v", err)
	}
	if areas == 0 {
		t.Error("expected at least 1 area generated")
	}
	if leads != 0 {
		t.Errorf("expected 0 leads for mimocode (AreasOnly), got %d", leads)
	}
	if teams != 0 {
		t.Errorf("expected 0 teams for mimocode (AreasOnly), got %d", teams)
	}
}

func TestGenerateTarget_Cursor(t *testing.T) {
	root := t.TempDir()
	createCanonicalDir(t, root)

	areas, leads, teams, err := generateTarget(
		filepath.Join(root, "go-runtime", "internal", "agents"),
		root,
		"cursor",
		"",
	)
	if err != nil {
		t.Fatalf("generateTarget failed: %v", err)
	}
	if areas == 0 {
		t.Error("expected at least 1 area generated")
	}
	if leads == 0 {
		t.Error("expected at least 1 lead generated for cursor")
	}
	if teams == 0 {
		t.Error("expected at least 1 team generated for cursor")
	}
}

func TestGenerateTarget_WithLevelsAll(t *testing.T) {
	root := t.TempDir()
	createCanonicalDir(t, root)

	areas, leads, teams, err := generateTarget(
		filepath.Join(root, "go-runtime", "internal", "agents"),
		root,
		"opencode",
		"all",
	)
	if err != nil {
		t.Fatalf("generateTarget failed: %v", err)
	}
	if areas == 0 {
		t.Error("expected at least 1 area")
	}
	if leads == 0 {
		t.Error("expected leads with --levels all")
	}
	if teams == 0 {
		t.Error("expected teams with --levels all")
	}
}

func TestGenerateTarget_CleanOldOutputError(t *testing.T) {
	root := t.TempDir()
	createCanonicalDir(t, root)

	outputDir := filepath.Join(root, "go-runtime", "internal", "runtimes", "opencode", "agents")
	os.MkdirAll(outputDir, 0755)
	os.WriteFile(filepath.Join(outputDir, "stale-area.md"), []byte("old"), 0644)
	if err := os.Chmod(outputDir, 0555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer os.Chmod(outputDir, 0755)

	_, _, _, err := generateTarget(
		filepath.Join(root, "go-runtime", "internal", "agents"),
		root,
		"opencode",
		"",
	)
	if err == nil {
		t.Error("expected error when cleanOldOutput fails")
		return
	}
	if !strings.Contains(err.Error(), "cleaning old output") {
		t.Errorf("expected 'cleaning old output' in error, got: %v", err)
	}
}

func TestGenerateTarget_InvalidTarget(t *testing.T) {
	root := t.TempDir()

	_, _, _, err := generateTarget(
		filepath.Join(root, "go-runtime", "internal", "agents"),
		root,
		"nonexistent-runtime",
		"",
	)
	if err == nil {
		t.Error("expected error for invalid target")
	}
}

func TestGenerateTarget_InvalidLevelsOverride(t *testing.T) {
	root := t.TempDir()
	createCanonicalDir(t, root)

	_, _, _, err := generateTarget(
		filepath.Join(root, "go-runtime", "internal", "agents"),
		root,
		"opencode",
		"bogus",
	)
	if err == nil {
		t.Error("expected error for invalid --levels override")
	}
}

// ── main (in-process via os.Chdir to fake repo) ──────────────────────────────

// createFakeRepo builds a minimal repo in dir so findRepoRoot succeeds when
// called from that directory.
func createFakeRepo(t *testing.T, dir string) {
	t.Helper()
	// OVAV mono-repo structure: go-runtime/go.mod at root, .ovav/service_areas/
	os.MkdirAll(filepath.Join(dir, ".ovav", "service_areas"), 0755)
	os.MkdirAll(filepath.Join(dir, "go-runtime"), 0755)
	os.WriteFile(filepath.Join(dir, "go-runtime", "go.mod"), []byte("module test\n"), 0644)
	createCanonicalDir(t, dir)
}

// runMainInProcess calls main() in-process by chdir-ing to a fake repo.
// It captures stdout/stderr and restores os state afterwards.
func runMainInProcess(t *testing.T, args []string) (stdout, stderr string) {
	t.Helper()

	fakeRepo := t.TempDir()
	createFakeRepo(t, fakeRepo)

	// Save state
	origDir, _ := os.Getwd()
	origArgs := os.Args
	origStdout := os.Stdout
	origStderr := os.Stderr

	// Create pipes to capture output
	stdoutR, stdoutW, _ := os.Pipe()
	stderrR, stderrW, _ := os.Pipe()
	os.Stdout = stdoutW
	os.Stderr = stderrW

	// Change to fake repo
	if err := os.Chdir(fakeRepo); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	os.Args = append([]string{"convert_agents"}, args...)

	// Run main (should NOT call os.Exit since findRepoRoot will succeed)
	main()

	// Restore state
	os.Chdir(origDir)
	os.Args = origArgs
	os.Stdout = origStdout
	os.Stderr = origStderr
	stdoutW.Close()
	stderrW.Close()

	var outBuf, errBuf [4096]byte
	n, _ := stdoutR.Read(outBuf[:])
	stdout = string(outBuf[:n])
	n, _ = stderrR.Read(errBuf[:])
	stderr = string(errBuf[:n])
	return
}

func TestMain_InProcess_NoArgs(t *testing.T) {
	stdout, _ := runMainInProcess(t, nil)
	if !strings.Contains(stdout, "Done.") {
		t.Errorf("expected 'Done.' in stdout, got: %s", stdout)
	}
}

func TestMain_InProcess_AllTargets(t *testing.T) {
	stdout, _ := runMainInProcess(t, nil)
	if !strings.Contains(stdout, "mimocode") {
		t.Error("expected 'mimocode' in output")
	}
	if !strings.Contains(stdout, "opencode") {
		t.Error("expected 'opencode' in output")
	}
	if !strings.Contains(stdout, "claude-code") {
		t.Error("expected 'claude-code' in output")
	}
	if !strings.Contains(stdout, "cursor") {
		t.Error("expected 'cursor' in output")
	}
}

func TestMain_InProcess_WithTarget(t *testing.T) {
	stdout, _ := runMainInProcess(t, []string{"--target", "opencode"})
	if !strings.Contains(stdout, "opencode") {
		t.Errorf("expected 'opencode' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Done.") {
		t.Errorf("expected 'Done.' in output, got: %s", stdout)
	}
}

func TestMain_InProcess_LevelsAll(t *testing.T) {
	stdout, _ := runMainInProcess(t, []string{"--target", "opencode", "--levels", "all"})
	if !strings.Contains(stdout, "forced all") {
		t.Errorf("expected 'forced all' in output, got: %s", stdout)
	}
}

func TestMain_InProcess_LevelsAreas(t *testing.T) {
	stdout, _ := runMainInProcess(t, []string{"--target", "mimocode", "--levels", "areas"})
	if !strings.Contains(stdout, "forced areas") {
		t.Errorf("expected 'forced areas' in output, got: %s", stdout)
	}
}

func TestMain_InProcess_NonexistentTarget(t *testing.T) {
	stdout, _ := runMainInProcess(t, []string{"--target", "nonexistent-runtime"})
	if !strings.Contains(stdout, "Done.") {
		t.Errorf("expected 'Done.' in output even with bad target, got: %s", stdout)
	}
}

func TestMain_InProcess_GenerateTargetError(t *testing.T) {
	// Set up a fake repo and pre-create a read-only output dir so
	// cleanOldOutput fails inside generateTarget, hitting the error
	// branch in main()'s loop (line 136-138).
	fakeRepo := t.TempDir()
	// OVAV mono-repo structure: .ovav/service_areas/ + go-runtime/go.mod
	os.MkdirAll(filepath.Join(fakeRepo, ".ovav", "service_areas"), 0755)
	os.MkdirAll(filepath.Join(fakeRepo, "go-runtime"), 0755)
	os.WriteFile(filepath.Join(fakeRepo, "go-runtime", "go.mod"), []byte("module test\n"), 0644)
	createCanonicalDir(t, fakeRepo)

	// Pre-create the mimocode output dir with a stale .md file and make it read-only.
	outDir := filepath.Join(fakeRepo, "runtimes", "mimocode", "agents")
	os.MkdirAll(outDir, 0755)
	os.WriteFile(filepath.Join(outDir, "stale.md"), []byte("old"), 0644)
	os.Chmod(outDir, 0555)
	defer os.Chmod(outDir, 0755)

	origDir, _ := os.Getwd()
	origArgs := os.Args
	origStdout := os.Stdout
	origStderr := os.Stderr
	stdoutR, stdoutW, _ := os.Pipe()
	stderrR, stderrW, _ := os.Pipe()
	os.Stdout = stdoutW
	os.Stderr = stderrW

	os.Chdir(fakeRepo)
	// Run all targets; mimocode will fail on cleanOldOutput, others should succeed
	os.Args = []string{"convert_agents"}
	main()

	os.Chdir(origDir)
	os.Args = origArgs
	os.Stdout = origStdout
	os.Stderr = origStderr
	stdoutW.Close()
	stderrW.Close()

	var outBuf, errBuf [4096]byte
	n, _ := stdoutR.Read(outBuf[:])
	stdout := string(outBuf[:n])
	n, _ = stderrR.Read(errBuf[:])
	stderr := string(errBuf[:n])

	if !strings.Contains(stdout, "Done.") {
		t.Errorf("expected 'Done.' in stdout, got: %s", stdout)
	}
	// The error branch should have printed to stderr
	if !strings.Contains(stderr, "mimocode") {
		t.Errorf("expected 'mimocode' error in stderr, got: %s", stderr)
	}
}

// ── main (subprocess — for os.Exit(1) error path) ────────────────────────────

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "convert_agents")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = filepath.Join(mustFindRepoRoot(), "go-runtime", "cmd", "convert_agents")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
	return bin
}

func mustFindRepoRoot() string {
	wd, _ := os.Getwd()
	dir := wd
	for {
		// OVAV mono-repo: must have .ovav/service_areas/ AND go-runtime/go.mod
		if _, err := os.Stat(filepath.Join(dir, ".ovav", "service_areas")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "go-runtime", "go.mod")); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "/home/braka/Systems/OVAV"
		}
		dir = parent
	}
}

func TestMain_Subprocess_NoRepoRoot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test")
	}
	bin := buildBinary(t)

	// The workspace contains a nested .ovav/ directory; placing the subprocess
	// outside it proves root discovery fails without touching real projections.
	tmp := filepath.Join(t.TempDir(), "outside-workspace")
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin)
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("expected non-zero exit when run outside repo root")
	}
	if !strings.Contains(string(out), "ERROR") {
		t.Errorf("expected ERROR in output, got: %s", out)
	}
}
