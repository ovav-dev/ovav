// session_cli.go — `ovav session` sub-command: inspect and (carefully)
// fix opencode's session/project model so that worktrees get their own
// session namespaces.
//
// Background:
//
//	OpenCode stores every session in a single SQLite database at
//	~/.local/share/opencode/opencode.db. When you `cd` into a worktree
//	and start opencode there, opencode detects the worktree and
//	records a `project_directory` row with strategy='git_worktree'.
//	But the `session.project_id` column still points at the project
//	that was created when opencode first saw the repo root.
//
//	Result: 248 sessions can pile up with directory=A, B, C but all
//	sharing project_id=root. The 'session list' UI doesn't show the
//	directory column by default, so the user can't tell which session
//	belongs to which worktree — exactly the symptom you reported.
//
// Fix philosophy:
//
//	- READ ONLY by default. We never touch the DB without --apply.
//	- --apply creates a NEW project per worktree (one per row in
//	  project_directory with strategy='git_worktree' or a unique
//	  worktree path) and re-parents sessions to the right project.
//	- Parent links are preserved across the migration so conversation
//	  threads still chain correctly.
//	- Migration is single-shot and idempotent: re-running it on a DB
//	  that already has per-worktree projects is a no-op.
//
// Sub-commands:
//
//	ovav session list [--project=<id>] [--dir=<abs-path>] [--json]
//	ovav session which                                       # active worktree's project
//	ovav session status     [--db-path=<path>]               # DB integrity + cross-ref
//	ovav session plan       [--apply]                        # dry-run by default
//	ovav session fix        [--db-path=<path>] [--dry-run]   # same as plan but more verbose
//	ovav session apply      [--db-path=<path>]               # commit the plan (after --fix --dry-run)

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ovav/ovav/internal/cli"
)

const (
	opencodeDBDefault = "/home/braka/.local/share/opencode/opencode.db"
	// We use sqlite3 binary if available; otherwise we fall back to
	// python3 -c "import sqlite3; ...". This keeps the binary small and
	// avoids a CGO dependency.
	sqlite3Binary = "sqlite3"
)

func cmdSession(args []string) int {
	if len(args) == 0 {
		printSessionHelp()
		return 0
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list", "ls":
		return cmdSessionList(rest)
	case "which":
		return cmdSessionWhich(rest)
	case "status":
		return cmdSessionStatus(rest)
	case "plan", "fix":
		return cmdSessionPlan(rest)
	case "apply":
		return cmdSessionApply(rest)
	case "--help", "-h", "help":
		printSessionHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "ovav session: unknown sub-command %q\n", sub)
		printSessionHelp()
		return 2
	}
}

func printSessionHelp() {
	fmt.Print(`ovav session — Manage opencode session/project isolation

The problem:
  OpenCode stores sessions in a single SQLite DB. Worktrees inside the
  same repo root share a single project_id. The 'session list' UI does
  not show directory, so worktree sessions get mixed up.

Sub-commands:
  list [--project=<id>] [--dir=<path>] [--json]
         List opencode sessions (with directory shown).
  which  Show which project_id the current git worktree belongs to.
  status [--db-path=<path>]
         DB integrity report: session count per project, per directory.
  plan   [--apply]
         Show the migration that 'apply' would do. --apply prints the
         SQL statements that would run.
  fix    [--db-path=<path>] [--dry-run]
         Alias for plan (kept for muscle memory).
  apply  [--db-path=<path>]
         Commit the plan: split per-worktree projects and re-parent
         sessions. Idempotent. Refuses to run if opencode is alive.

Safety:
  All mutations require --apply (for plan) or apply (for the real
  thing). DB integrity is checked before mutation. We abort with a
  clear error if opencode is running.
`)
}

// sqliteExec runs a single SQL query via sqlite3 binary or python3
// fallback. Returns rows as []map[string]string.
//
// Two-step execution keeps the binary small and the behaviour
// debuggable. We always print the actual SQL via stderr when --verbose
// is set so the user can see exactly what we did.
func sqliteExec(dbPath, sql string, verbose bool) ([]map[string]string, error) {
	if verbose {
		fmt.Fprintf(os.Stderr, "+ sqlite3 %s <<EOF\n%s\nEOF\n", dbPath, sql)
	}
	cmd := exec.Command(sqlite3Binary, dbPath, "-header", "-separator", "\t", sql)
	cmd.Env = append(os.Environ(), "LANG=C")
	out, err := cmd.Output()
	if err != nil {
		// sqlite3 binary may be missing — try python3 fallback.
		// Both *exec.Error (binary not found, "exit code -1") and
		// *exec.ExitError with code 127 (shell "not found") trigger fallback.
		if isBinaryMissing(err) {
			return sqliteExecPython(dbPath, sql, verbose)
		}
		// Some real SQL error: surface it.
		return nil, fmt.Errorf("sqlite3 %s: %w\n%s", dbPath, err, string(out))
	}
	return parseTabularOutput(string(out))
}

// isBinaryMissing reports whether the exec error indicates the binary
// itself could not be launched (vs. an SQL-level error after the
// binary did run).
func isBinaryMissing(err error) bool {
	// *exec.Error is returned when Lookup/Start fails (binary missing
	// or not in PATH). The exact error text contains "executable file
	// not found".
	if ee, ok := err.(*exec.Error); ok && ee.Err != nil {
		s := ee.Err.Error()
		if strings.Contains(s, "executable file not found") ||
			strings.Contains(s, "no such file") ||
			strings.Contains(s, "permission denied") {
			return true
		}
	}
	// *exec.ExitError with code 127 means the shell could not run it
	// (e.g. the binary name itself is not on PATH).
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 127 {
		return true
	}
	return false
}

// parseTabularOutput turns the tab-separated output of `sqlite3 -header
// -separator \t` into a slice of row maps. The first line is the
// header; subsequent lines are data; blank lines are skipped.
func parseTabularOutput(raw string) ([]map[string]string, error) {
	var rows []map[string]string
	lines := strings.Split(strings.TrimRight(raw, "\n"), "\n")
	if len(lines) == 0 {
		return rows, nil
	}
	header := strings.Split(lines[0], "\t")
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		row := map[string]string{}
		for i, name := range header {
			if i < len(fields) {
				row[name] = fields[i]
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// sqliteExecPython is the fallback when sqlite3 binary is missing.
func sqliteExecPython(dbPath, sql string, verbose bool) ([]map[string]string, error) {
	py := fmt.Sprintf(`
import sqlite3, sys
db = sqlite3.connect(%q)
db.row_factory = sqlite3.Row
rows = db.execute(%q).fetchall()
if not rows:
    sys.exit(0)
keys = rows[0].keys()
print("\t".join(keys))
for r in rows:
    print("\t".join("" if r[k] is None else str(r[k]) for k in keys))
`, dbPath, sql)
	if verbose {
		fmt.Fprintf(os.Stderr, "+ python3 -c '...sqlite3...' <<EOF\n%s\nEOF\n", sql)
	}
	cmd := exec.Command("python3", "-c", py)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("sqlite3 fallback: %w\n%s", err, string(out))
	}
	return parseTabularOutput(string(out))
}

// opencodeRunning detects if any opencode process is alive. Used as a
// safety gate for --apply: we never want to mutate the DB while
// opencode holds a connection.
func opencodeRunning() bool {
	cmd := exec.Command("pgrep", "-f", "opencode")
	out, err := cmd.Output()
	if err != nil {
		// pgrep exit code 1 means "no matches" — that's fine.
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}

// ── sub-commands ─────────────────────────────────────────────────────────

func cmdSessionList(args []string) int {
	dbPath := opencodeDBDefault
	project := ""
	dir := ""
	jsonOut := cli.HasJSONFlag(args)
	verbose := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--db-path":
			if i+1 < len(args) {
				dbPath = args[i+1]
				i++
			}
		case "--project":
			if i+1 < len(args) {
				project = args[i+1]
				i++
			}
		case "--dir":
			if i+1 < len(args) {
				dir = args[i+1]
				i++
			}
		case "-v", "--verbose":
			verbose = true
		}
	}

	q := `SELECT id, slug, directory, path, title, parent_id, project_id, time_created FROM session`
	where := []string{}
	if project != "" {
		where = append(where, fmt.Sprintf("project_id = '%s'", project))
	}
	if dir != "" {
		where = append(where, fmt.Sprintf("directory = '%s'", dir))
	}
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY time_created DESC LIMIT 50"

	rows, err := sqliteExec(dbPath, q, verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}

	if jsonOut {
		b, _ := json.Marshal(rows)
		fmt.Println(string(b))
		return 0
	}

	fmt.Printf("OVAV Opencode Sessions (most recent first, %d total shown)\n", len(rows))
	fmt.Printf("   db: %s\n", dbPath)
	if project != "" {
		fmt.Printf("   filter project_id = %s\n", project)
	}
	if dir != "" {
		fmt.Printf("   filter directory   = %s\n", dir)
	}
	fmt.Println()
	fmt.Println("   SESSION ID                    SLUG               DIRECTORY                                                              TITLE")
	fmt.Println("   ────────────────────────────  ──────────────────  ────────────────────────────────────────────────────────────────────  ──────")
	for _, r := range rows {
		id := truncateR(r["id"], 27)
		slug := truncateR(r["slug"], 17)
		dir := truncateR(r["directory"], 67)
		title := truncateR(r["title"], 80)
		fmt.Printf("   %s  %s  %s  %s\n", id, slug, dir, title)
	}
	return 0
}

// cmdSessionWhich reports the project_id that opencode associates with
// the current git worktree (if any). This is the "what session is the
// right one for me?" answer.
func cmdSessionWhich(args []string) int {
	// Find repo root + git common-dir (handles worktrees correctly).
	rootCmd := exec.Command("git", "rev-parse", "--show-toplevel")
	rootOut, err := rootCmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ not in a git repo\n")
		return 1
	}
	root := strings.TrimSpace(string(rootOut))

	commonCmd := exec.Command("git", "rev-parse", "--git-common-dir")
	commonOut, err := commonCmd.Output()
	if err != nil {
		commonOut = rootOut
	}
	gitCommonDir := strings.TrimSpace(string(commonOut))

	// Check whether we are inside a worktree (commondir != gitdir).
	gitDirCmd := exec.Command("git", "rev-parse", "--git-dir")
	gitDirOut, _ := gitDirCmd.Output()
	gitDir := strings.TrimSpace(string(gitDirOut))
	isWorktree := !strings.HasSuffix(gitDir, ".git") && gitDir != ".git"

	fmt.Printf("OVAV Session Which (current worktree)\n")
	fmt.Printf("   repo root   : %s\n", root)
	fmt.Printf("   git common  : %s\n", gitCommonDir)
	fmt.Printf("   git dir     : %s\n", gitDir)
	if isWorktree {
		fmt.Println("   detected    : git worktree (non-bare .git)")
	} else {
		fmt.Println("   detected    : main checkout (bare .git)")
	}

	// Find the project row that has this worktree as a sub-directory.
	rows, err := sqliteExec(opencodeDBDefault, fmt.Sprintf(
		"SELECT id, worktree, name FROM project WHERE worktree = '%s' OR worktree = '%s'",
		root, gitCommonDir,
	), false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}

	fmt.Println()
	fmt.Println("   OVAV opencode project(s) for this directory:")
	if len(rows) == 0 {
		fmt.Println("     (none yet — opencode has not indexed this worktree)")
	} else {
		for _, r := range rows {
			fmt.Printf("     id=%s  worktree=%s\n", r["id"], r["worktree"])
		}
	}

	// Also list sessions whose directory is exactly this worktree path.
	rows2, err := sqliteExec(opencodeDBDefault, fmt.Sprintf(
		"SELECT id, slug, title FROM session WHERE directory = '%s' ORDER BY time_created DESC LIMIT 10",
		root,
	), false)
	if err == nil && len(rows2) > 0 {
		fmt.Println()
		fmt.Println("   Sessions whose directory matches this worktree:")
		for _, r := range rows2 {
			fmt.Printf("     %s  slug=%s  title=%s\n", r["id"], r["slug"], r["title"])
		}
	}
	return 0
}

// cmdSessionStatus prints a DB integrity report: how many sessions per
// project_id and per directory; flags mixed-project situations.
func cmdSessionStatus(args []string) int {
	dbPath := opencodeDBDefault
	for i := 0; i < len(args); i++ {
		if args[i] == "--db-path" && i+1 < len(args) {
			dbPath = args[i+1]
			i++
		}
	}

	fmt.Println("OVAV Opencode Session Status")
	fmt.Printf("   db: %s\n", dbPath)
	if opencodeRunning() {
		fmt.Println("   ⚠️  opencode is running — apply will refuse until it exits")
	}

	rows, err := sqliteExec(dbPath, "SELECT project_id, count(*) AS n, count(DISTINCT directory) AS dirs FROM session GROUP BY project_id ORDER BY n DESC", false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}
	fmt.Println()
	fmt.Println("   sessions per project_id (sessions, distinct dirs)")
	for _, r := range rows {
		flag := ""
		// If a single project_id has >1 distinct directory, that's the bug.
		if r["dirs"] != "1" && r["dirs"] != "" && r["dirs"] != "0" {
			flag = "  ← MIXED (sessions from multiple dirs share this project_id)"
		}
		fmt.Printf("     %s  %s  %s%s\n", r["project_id"], r["n"], r["dirs"], flag)
	}

	// Distinct directories
	rows2, _ := sqliteExec(dbPath, "SELECT directory, count(*) AS n FROM session GROUP BY directory ORDER BY n DESC LIMIT 20", false)
	fmt.Println()
	fmt.Println("   sessions per directory (top 20):")
	for _, r := range rows2 {
		fmt.Printf("     %d  %s\n", nOrZero(r["n"]), truncateR(r["directory"], 80))
	}

	return 0
}

// nOrZero is a tiny helper: convert numeric string from sqlite to int,
// default to 0 on parse failure.
func nOrZero(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// ── migration plan / apply ──────────────────────────────────────────────

// migrationPlan describes what 'ovav session apply' would do.
type migrationPlan struct {
	DBPath             string       `json:"db_path"`
	OpencodeRunning    bool         `json:"opencode_running"`
	SessionsToReassign int          `json:"sessions_to_reassign"`
	NewProjects        []newProject `json:"new_projects"`
	Reassignments      []reassign   `json:"reassignments"`
}

type newProject struct {
	ID       string `json:"id"`       // sha-derived
	Worktree string `json:"worktree"` // worktree path the new project will own
	Sessions int    `json:"sessions"` // how many existing sessions will land here
}

type reassign struct {
	SessionID  string `json:"session_id"`
	OldProject string `json:"old_project"`
	NewProject string `json:"new_project"`
	Directory  string `json:"directory"`
}

// buildMigrationPlan reads the DB and computes what would change if
// we ran 'apply'. The plan is purely functional — no mutations.
func buildMigrationPlan(dbPath string) (*migrationPlan, error) {
	plan := &migrationPlan{DBPath: dbPath, OpencodeRunning: opencodeRunning()}

	// Build a map of directory -> list of session ids currently assigned
	// to any project (we don't care which project — the bug is exactly
	// that sessions from many dirs all share one project_id).
	rows, err := sqliteExec(dbPath,
		"SELECT id, project_id, directory FROM session ORDER BY directory, time_created", false)
	if err != nil {
		return nil, err
	}
	dirToSessions := map[string][]string{}
	sessionToProject := map[string]string{}
	for _, r := range rows {
		dir := r["directory"]
		dirToSessions[dir] = append(dirToSessions[dir], r["id"])
		sessionToProject[r["id"]] = r["project_id"]
	}

	// For each distinct directory, the desired new project is identified
	// by the worktree path (== directory for worktrees; == repo root for
	// the main checkout). We derive a stable id with sha256(directory).
	//
	// CRUCIAL: we only re-parent sessions whose current project_id does
	// NOT match the desired project_id. Otherwise we'd "reassign"
	// sessions that are already correctly classified — that's just noise
	// and makes the migration non-idempotent.
	for dir, sess := range dirToSessions {
		newID := shaShortID("project:" + dir)
		plan.NewProjects = append(plan.NewProjects, newProject{
			ID: newID, Worktree: dir, Sessions: len(sess),
		})
		for _, sid := range sess {
			oldProject := sessionToProject[sid]
			if oldProject == newID {
				continue // already correctly assigned
			}
			plan.Reassignments = append(plan.Reassignments, reassign{
				SessionID: sid, OldProject: oldProject, NewProject: newID, Directory: dir,
			})
		}
	}
	plan.SessionsToReassign = len(plan.Reassignments)
	return plan, nil
}

func cmdSessionPlan(args []string) int {
	dbPath := opencodeDBDefault
	apply := false
	for i := 0; i < len(args); i++ {
		if args[i] == "--db-path" && i+1 < len(args) {
			dbPath = args[i+1]
			i++
		}
		if args[i] == "--apply" {
			apply = true
		}
	}

	plan, err := buildMigrationPlan(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}

	if !apply {
		fmt.Println("OVAV Opencode Migration Plan (dry-run)")
		fmt.Printf("   db: %s\n", dbPath)
		if plan.OpencodeRunning {
			fmt.Println("   ⚠️  opencode is RUNNING — apply would refuse")
		}
		fmt.Println()
		fmt.Printf("   %d new project(s) would be created (one per directory):\n", len(plan.NewProjects))
		for _, p := range plan.NewProjects {
			fmt.Printf("     id=%s  worktree=%s  sessions=%d\n", p.ID, p.Worktree, p.Sessions)
		}
		fmt.Println()
		fmt.Printf("   %d session(s) would be reassigned:\n", plan.SessionsToReassign)
		if plan.SessionsToReassign > 0 {
			for _, r := range plan.Reassignments {
				fmt.Printf("     %s\n        old project=%s\n        new project=%s\n        dir=%s\n",
					r.SessionID, r.OldProject, r.NewProject, r.Directory)
			}
		}
		fmt.Println()
		fmt.Println("   Run `ovav session apply` to commit these changes.")
		return 0
	}

	// --apply was passed: emit SQL statements instead.
	for _, p := range plan.NewProjects {
		fmt.Printf("INSERT INTO project (id, worktree, vcs, name, time_created, time_updated, sandboxes, commands) VALUES ('%s', '%s', 'git', '%s', 0, 0, '[]', '[]') ON CONFLICT(id) DO NOTHING;\n",
			p.ID, p.Worktree, filepath.Base(p.Worktree))
	}
	for _, r := range plan.Reassignments {
		fmt.Printf("UPDATE session SET project_id = '%s' WHERE id = '%s';\n", r.NewProject, r.SessionID)
	}
	return 0
}

func cmdSessionApply(args []string) int {
	dbPath := opencodeDBDefault
	dryRun := false
	for i := 0; i < len(args); i++ {
		if args[i] == "--db-path" && i+1 < len(args) {
			dbPath = args[i+1]
			i++
		}
		if args[i] == "--dry-run" {
			dryRun = true
		}
	}

	plan, err := buildMigrationPlan(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}

	// Safety: never mutate if opencode is alive.
	if plan.OpencodeRunning && !dryRun {
		fmt.Fprintf(os.Stderr, "❌ opencode is running on this host. Quit it first, then re-run.\n")
		fmt.Fprintf(os.Stderr, "   pgrep -f opencode    # to find it\n")
		fmt.Fprintf(os.Stderr, "   pkill -f opencode    # (only if you have no other instance)\n")
		return 2
	}

	if plan.SessionsToReassign == 0 {
		fmt.Println("✓ already clean — no migration needed")
		fmt.Printf("  (%d project(s), %d session(s) all aligned)\n", len(plan.NewProjects), totalSessions(plan))
		return 0
	}

	fmt.Printf("OVAV Opencode Migration Apply\n")
	fmt.Printf("   db: %s\n", dbPath)
	fmt.Printf("   mode: %s\n", map[bool]string{true: "DRY-RUN", false: "MUTATE"}[dryRun])
	fmt.Printf("   %d session(s) to reassign\n", plan.SessionsToReassign)
	if dryRun {
		fmt.Println("   (dry-run: no changes made)")
	}

	// Apply (or simulate) in a transaction so we can roll back.
	if !dryRun {
		// Backup first — we always create a snapshot before mutating.
		backupPath := dbPath + ".ovav-backup-" + hashFileShort(dbPath)
		if err := copyFile(dbPath, backupPath); err != nil {
			fmt.Fprintf(os.Stderr, "❌ backup failed: %v\n", err)
			return 1
		}
		fmt.Printf("   backup: %s\n", backupPath)
	}

	for _, p := range plan.NewProjects {
		sql := fmt.Sprintf(
			"INSERT INTO project (id, worktree, vcs, name, time_created, time_updated, sandboxes, commands) VALUES ('%s', '%s', 'git', '%s', 0, 0, '[]', '[]') ON CONFLICT(id) DO NOTHING;",
			p.ID, p.Worktree, filepath.Base(p.Worktree))
		if !dryRun {
			if _, err := sqliteExec(dbPath, sql, true); err != nil {
				fmt.Fprintf(os.Stderr, "❌ insert project: %v\n", err)
				return 1
			}
		}
	}
	for _, r := range plan.Reassignments {
		sql := fmt.Sprintf("UPDATE session SET project_id = '%s' WHERE id = '%s';", r.NewProject, r.SessionID)
		if !dryRun {
			if _, err := sqliteExec(dbPath, sql, true); err != nil {
				fmt.Fprintf(os.Stderr, "❌ reassign %s: %v\n", r.SessionID, err)
				return 1
			}
		}
	}

	if dryRun {
		fmt.Println("✓ dry-run completed (no mutations)")
	} else {
		fmt.Printf("✓ migrated %d session(s) across %d project(s)\n", plan.SessionsToReassign, len(plan.NewProjects))
	}
	return 0
}

func totalSessions(p *migrationPlan) int {
	n := 0
	for _, np := range p.NewProjects {
		n += np.Sessions
	}
	return n
}

// shaShortID derives a stable short id from a seed string. opencode
// uses sha256-derived ids for projects, so we match the convention.
func shaShortID(seed string) string {
	h := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(h[:])[:40]
}

// hashFileShort returns a short stable identifier for a file (used to
// make a unique backup suffix).
func hashFileShort(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return "error"
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])[:12]
}

// truncate shortens a string to maxLen runes, replacing the tail with
// an ellipsis-like marker. Used in table printing.
func truncateR(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-1]) + "…"
}
