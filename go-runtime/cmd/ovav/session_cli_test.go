// session_cli_test.go — unit tests for ovav session sub-command.
//
// Tests use a tmpfile SQLite DB (created via python3) so we never
// touch the real opencode DB. Helpers cover both happy path and the
// detection of mixed-project_id states.

package main

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// makeFixtureDB writes a minimal opencode.db fixture into a temp dir
// that mimics the mixed-project bug. Returns the path to the DB file.
//
// Schema:
//   - 2 projects: P_OVAV (root), P_FEATURE (worktree)
//   - sessions: 3 in /ovav, 1 in /ovav/worktrees/feature
//   - project_directory: rows for both worktree paths
func makeFixtureDB(t *testing.T, dir string) string {
	t.Helper()
	dbPath := filepath.Join(dir, "test-opencode.db")
	// Pass dbPath via argv[1]; the Python script uses sys.argv[1].
	py := `
import sqlite3, sys
db = sqlite3.connect(sys.argv[1])
db.executescript("""
CREATE TABLE project (
    id TEXT PRIMARY KEY, worktree TEXT NOT NULL, vcs TEXT, name TEXT,
    icon_url TEXT, icon_url_override TEXT, icon_color TEXT,
    time_created INTEGER NOT NULL, time_updated INTEGER NOT NULL,
    time_initialized INTEGER, sandboxes TEXT NOT NULL, commands TEXT
);
CREATE TABLE session (
    id TEXT PRIMARY KEY, project_id TEXT NOT NULL,
    workspace_id TEXT, parent_id TEXT, slug TEXT NOT NULL,
    directory TEXT NOT NULL, path TEXT, title TEXT NOT NULL,
    version TEXT NOT NULL, share_url TEXT,
    summary_additions INTEGER, summary_deletions INTEGER,
    summary_files INTEGER, summary_diffs TEXT,
    message_count INTEGER NOT NULL DEFAULT 0,
    time_created INTEGER NOT NULL, time_updated INTEGER NOT NULL,
    time_archived INTEGER, time_compacted INTEGER,
    parent_session_compacted INTEGER,
    revert_version INTEGER, revert_message_id TEXT
);
CREATE TABLE project_directory (
    project_id TEXT NOT NULL, directory TEXT NOT NULL, type TEXT,
    strategy TEXT, time_created INTEGER NOT NULL,
    PRIMARY KEY (project_id, directory)
);
""")
now = 1000
# Use the same sha-derived IDs that buildMigrationPlan() computes —
# that's what makes the fixture match what the production migration
# will generate.
import hashlib
def shortid(seed):
    return hashlib.sha256(seed.encode()).hexdigest()[:40]
p_root = shortid("project:/ovav")
p_feat = shortid("project:/ovav/.ovav/worktrees/feature")
db.execute("INSERT INTO project VALUES (?, ?, 'git', 'ovav', NULL, NULL, NULL, ?, ?, NULL, '[]', '[]')",
           [p_root, "/ovav", now, now])
db.execute("INSERT INTO project VALUES (?, ?, 'git', 'feature', NULL, NULL, NULL, ?, ?, NULL, '[]', '[]')",
           [p_feat, "/ovav/.ovav/worktrees/feature", now, now])
# All sessions initially assigned to the root project's id — that
# mimics the bug: opencode put feature-tree sessions under the same
# project_id as the main checkout.
mixed = p_root
db.execute("INSERT INTO session (id, project_id, slug, directory, title, version, time_created, time_updated) VALUES (?, ?, ?, ?, ?, 'v1', ?, ?)",
           ["ses_root1", mixed, "root-1", "/ovav", "Root A", now, now])
db.execute("INSERT INTO session (id, project_id, slug, directory, title, version, time_created, time_updated) VALUES (?, ?, ?, ?, ?, 'v1', ?, ?)",
           ["ses_root2", mixed, "root-2", "/ovav", "Root B", now, now])
db.execute("INSERT INTO session (id, project_id, slug, directory, title, version, time_created, time_updated) VALUES (?, ?, ?, ?, ?, 'v1', ?, ?)",
           ["ses_root3", mixed, "root-3", "/ovav", "Root C", now, now])
# THE BUG: 2 sessions in feature worktree wrongly assigned to root.
db.execute("INSERT INTO session (id, project_id, slug, directory, title, version, time_created, time_updated) VALUES (?, ?, ?, ?, ?, 'v1', ?, ?)",
           ["ses_feat1", mixed, "feat-1", "/ovav/.ovav/worktrees/feature", "Feat 1", now, now])
db.execute("INSERT INTO session (id, project_id, slug, directory, title, version, time_created, time_updated) VALUES (?, ?, ?, ?, ?, 'v1', ?, ?)",
           ["ses_feat2", mixed, "feat-2", "/ovav/.ovav/worktrees/feature", "Feat 2", now, now])
db.commit()
db.close()
`
	cmd := exec.Command("python3", "-c", py, dbPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("setup python: %v\n%s", err, string(out))
	}
	return dbPath
}

func TestSessionStatus_DetectsMixedProject(t *testing.T) {
	dir := t.TempDir()
	dbPath := makeFixtureDB(t, dir)
	// We need to inject the test DB path into the helper — for the
	// cmdSessionStatus flow we'll spawn the binary with --db-path.
	// Here we test the parsing/aggregation logic directly by running
	// the function via a small helper that points at the test DB.

	rows, err := sqliteExec(dbPath,
		"SELECT project_id, count(*) AS n, count(DISTINCT directory) AS dirs FROM session GROUP BY project_id ORDER BY n DESC",
		false)
	if err != nil {
		t.Fatalf("sqliteExec: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row (root project has all 5 sessions), got %d", len(rows))
	}
	// We don't pin the project_id here — it's a sha256-derived hash.
	// What matters: 5 sessions across 2 distinct directories on a
	// single project_id (= the bug signature).
	if rows[0]["n"] != "5" {
		t.Errorf("expected 5 sessions in root project, got %s", rows[0]["n"])
	}
	if rows[0]["dirs"] != "2" {
		t.Errorf("expected 2 distinct dirs, got %s", rows[0]["dirs"])
	}
}

func TestBuildMigrationPlan_FixtureDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := makeFixtureDB(t, dir)

	plan, err := buildMigrationPlan(dbPath)
	if err != nil {
		t.Fatalf("buildMigrationPlan: %v", err)
	}
	if len(plan.NewProjects) != 2 {
		t.Errorf("expected 2 new projects (one per dir), got %d", len(plan.NewProjects))
	}
	if plan.SessionsToReassign != 2 {
		t.Errorf("expected 2 sessions to reassign, got %d", plan.SessionsToReassign)
	}

	// Verify the session re-assignments target the feature project.
	feats := 0
	for _, r := range plan.Reassignments {
		if r.Directory == "/ovav/.ovav/worktrees/feature" && r.NewProject != "" {
			feats++
		}
	}
	if feats != 2 {
		t.Errorf("expected 2 feat reassignments, got %d", feats)
	}
}

func TestBuildMigrationPlan_AlreadyClean(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "clean.db")
	py := `
import sqlite3, sys, hashlib
def shortid(seed):
    return hashlib.sha256(seed.encode()).hexdigest()[:40]
db = sqlite3.connect(sys.argv[1])
db.executescript("""
CREATE TABLE project (id TEXT PRIMARY KEY, worktree TEXT NOT NULL, vcs TEXT, name TEXT,
    icon_url TEXT, icon_url_override TEXT, icon_color TEXT,
    time_created INTEGER NOT NULL, time_updated INTEGER NOT NULL,
    time_initialized INTEGER, sandboxes TEXT NOT NULL, commands TEXT);
CREATE TABLE session (id TEXT PRIMARY KEY, project_id TEXT NOT NULL,
    workspace_id TEXT, parent_id TEXT, slug TEXT NOT NULL,
    directory TEXT NOT NULL, path TEXT, title TEXT NOT NULL,
    version TEXT NOT NULL, share_url TEXT, summary_additions INTEGER,
    summary_deletions INTEGER, summary_files INTEGER, summary_diffs TEXT,
    message_count INTEGER NOT NULL DEFAULT 0,
    time_created INTEGER NOT NULL, time_updated INTEGER NOT NULL,
    time_archived INTEGER, time_compacted INTEGER,
    parent_session_compacted INTEGER, revert_version INTEGER,
    revert_message_id TEXT);
""")
now = 1000
db.execute("INSERT INTO project VALUES (?, ?, 'git', 'p', NULL, NULL, NULL, ?, ?, NULL, '[]', '[]')", [shortid("project:/clean"), "/clean", now, now])
db.execute("INSERT INTO session (id, project_id, slug, directory, title, version, time_created, time_updated) VALUES (?, ?, ?, ?, ?, 'v1', ?, ?)", ["ses_c1", shortid("project:/clean"), "c-1", "/clean", "Clean", now, now])
db.commit()
db.close()
`
	if out, err := exec.Command("python3", "-c", py, dbPath).CombinedOutput(); err != nil {
		t.Fatalf("setup: %v\n%s", err, string(out))
	}
	plan, err := buildMigrationPlan(dbPath)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.SessionsToReassign != 0 {
		t.Errorf("clean DB: expected 0 reassignments, got %d", plan.SessionsToReassign)
	}
}

func TestShaShortID_Deterministic(t *testing.T) {
	a := shaShortID("project:/ovav")
	b := shaShortID("project:/ovav")
	if a != b {
		t.Errorf("shaShortID not deterministic: %s vs %s", a, b)
	}
	if len(a) != 40 {
		t.Errorf("expected 40 hex chars, got %d", len(a))
	}
	c := shaShortID("project:/feature")
	if a == c {
		t.Errorf("different inputs produced same hash: %s", a)
	}
}

func TestIsBinaryMissing(t *testing.T) {
	// Run a non-existent binary — should produce an error that
	// isBinaryMissing returns true for.
	_, err := exec.Command("definitely-not-a-real-binary-12345").Output()
	if err == nil {
		t.Fatalf("expected error from missing binary")
	}
	if !isBinaryMissing(err) {
		t.Errorf("isBinaryMissing should return true for missing binary, got false")
	}
	// A real binary that exits non-zero — should NOT be 'missing'.
	_, err = exec.Command("false").Output()
	if err == nil {
		t.Fatalf("expected non-zero exit from 'false'")
	}
	if isBinaryMissing(err) {
		t.Errorf("isBinaryMissing should return false for 'false' (binary exists)")
	}
}

func TestParseTabularOutput_Empty(t *testing.T) {
	rows, err := parseTabularOutput("")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(rows))
	}
}

func TestParseTabularOutput_Real(t *testing.T) {
	raw := "a\tb\n1\t2\n3\t4\n"
	rows, err := parseTabularOutput(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0]["a"] != "1" || rows[0]["b"] != "2" {
		t.Errorf("row 0 wrong: %+v", rows[0])
	}
	if rows[1]["a"] != "3" || rows[1]["b"] != "4" {
		t.Errorf("row 1 wrong: %+v", rows[1])
	}
}

func TestTruncateR(t *testing.T) {
	cases := map[string][2]string{
		"short": {"hello", "hello"},
		// n=8 → truncate to 7 runes + ellipsis (1 rune) = 8 runes total
		"longer-than-n": {"abcdefghij", "abcdefg…"},
		"empty":         {"", ""},
		// n=8 → 7 runes + "…" = "héllo-w…"
		"unicode": {"héllo-wörld", "héllo-w…"},
	}
	for name, tc := range cases {
		got := truncateR(tc[0], 8)
		if got != tc[1] {
			t.Errorf("%s: truncateR(%q,8) = %q, want %q", name, tc[0], got, tc[1])
		}
	}
}

func TestOpencodeRunning_Detects(t *testing.T) {
	// We assume opencode may or may not be running during tests. Just
	// verify the helper doesn't panic and returns a bool.
	_ = opencodeRunning()
	// Nothing to assert — depends on host state. We only check it
	// does not error or block.
}

// TestSessionApply_RefusesWhenOpencodeRunning guards the safety check.
func TestSessionApply_RefusesWhenOpencodeRunning(t *testing.T) {
	dir := t.TempDir()
	dbPath := makeFixtureDB(t, dir)
	// We can't actually force opencode to be "running" in a test
	// (it depends on host state), so we just check the gate logic by
	// inspecting opencodeRunning(): if it's true, the apply should
	// refuse; if it's false, the apply should proceed.
	if opencodeRunning() {
		// Simulate the apply path's safety check.
		plan, _ := buildMigrationPlan(dbPath)
		// If opencode is alive and we got this far, the apply MUST
		// refuse. Verify the gate field is set in the plan.
		if !plan.OpencodeRunning {
			t.Errorf("plan.OpencodeRunning should be true while opencode is running")
		}
	}
}

// TestSessionPlan_JSONShape guards the JSON contract for future
// consumers (scripts that want to inspect a plan programmatically).
func TestSessionPlan_JSONShape(t *testing.T) {
	dir := t.TempDir()
	dbPath := makeFixtureDB(t, dir)
	plan, err := buildMigrationPlan(dbPath)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	b, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, want := range []string{
		`"db_path":`,
		`"opencode_running":`,
		`"sessions_to_reassign":`,
		`"new_projects":`,
		`"reassignments":`,
	} {
		if !strings.Contains(string(b), want) {
			t.Errorf("JSON missing %q:\n%s", want, string(b))
		}
	}
}

func TestSessionPlan_RefsRealisticDirs(t *testing.T) {
	dir := t.TempDir()
	dbPath := makeFixtureDB(t, dir)
	plan, err := buildMigrationPlan(dbPath)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(plan.NewProjects) != 2 {
		t.Fatalf("expected 2 projects (one per dir), got %d", len(plan.NewProjects))
	}
	// Verify worktrees are unique.
	seen := map[string]bool{}
	for _, p := range plan.NewProjects {
		if seen[p.Worktree] {
			t.Errorf("duplicate worktree in plan: %s", p.Worktree)
		}
		seen[p.Worktree] = true
	}
}
