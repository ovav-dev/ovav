package chronos

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// ── Test helpers ────────────────────────────────────────────────────────────

// initTestRepo creates a temporary git repository with a few commits.
// Returns the repo root path and a cleanup function.
func initTestRepo(t *testing.T) (string, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "ovav-chronos-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(dir)
	}

	// Initialize git repo
	cmds := [][]string{
		{"init", dir},
	}
	for _, args := range cmds {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %s\n%v", args, out, err)
		}
	}

	// Set git user for commits
	setGitUser(t, dir)

	// Create commits
	for i, msg := range []string{"initial commit", "second commit", "third commit"} {
		createCommit(t, dir, msg, i)
	}

	return dir, cleanup
}

func setGitUser(t *testing.T, dir string) {
	t.Helper()
	for _, kv := range [][2]string{
		{"user.name", "Test User"},
		{"user.email", "test@ovav.local"},
	} {
		cmd := exec.Command("git", "config", kv[0], kv[1])
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git config %s failed: %s\n%v", kv[0], out, err)
		}
	}
}

func createCommit(t *testing.T, dir, msg string, id int) {
	t.Helper()

	fname := filepath.Join(dir, "test.txt")
	f, err := os.OpenFile(fname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("failed to open test file: %v", err)
	}
	f.WriteString(msg + "\n")
	f.Close()

	cmd := exec.Command("git", "add", "test.txt")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %s\n%v", out, err)
	}

	// Use a past date to ensure commit age is always positive.
	commitDate := time.Now().Add(-24 * time.Hour).Format("2006-01-02T15:04:05-0700")
	cmd = exec.Command("git", "commit", "-m", msg)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_DATE="+commitDate,
		"GIT_COMMITTER_DATE="+commitDate,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %s\n%v", out, err)
	}
}

// ── Tests ───────────────────────────────────────────────────────────────────

func TestBuildNowBlock(t *testing.T) {
	now := time.Date(2026, 6, 18, 10, 30, 45, 0, limaLocation)
	block := buildNowBlock(now)

	if block.Year != 2026 {
		t.Errorf("expected year 2026, got %d", block.Year)
	}
	if block.Month != 6 {
		t.Errorf("expected month 6, got %d", block.Month)
	}
	if block.Day != 18 {
		t.Errorf("expected day 18, got %d", block.Day)
	}
	if block.Hour != 10 {
		t.Errorf("expected hour 10, got %d", block.Hour)
	}
	if block.Minute != 30 {
		t.Errorf("expected minute 30, got %d", block.Minute)
	}
	if block.Second != 45 {
		t.Errorf("expected second 45, got %d", block.Second)
	}
	if block.Timezone != limaTZ {
		t.Errorf("expected timezone %s, got %s", limaTZ, block.Timezone)
	}
	if !block.Format24h {
		t.Error("expected format_24h to be true")
	}
	if block.Weekday == "" {
		t.Error("expected non-empty weekday")
	}
	if block.UTC == "" {
		t.Error("expected non-empty UTC")
	}
	if block.Lima == "" {
		t.Error("expected non-empty Lima")
	}
	if block.ISO == "" {
		t.Error("expected non-empty ISO")
	}
	if block.Lima != block.ISO {
		t.Errorf("Lima and ISO should be equal, got Lima=%q ISO=%q", block.Lima, block.ISO)
	}
}

func TestBuildHeadBlock(t *testing.T) {
	repoRoot, cleanup := initTestRepo(t)
	defer cleanup()

	now := time.Now()
	block := buildHeadBlock(repoRoot, now)

	if block.Hash == "" {
		t.Error("expected non-empty hash")
	}
	if block.HashShort == "" {
		t.Error("expected non-empty hash_short")
	}
	if len(block.HashShort) != 7 {
		t.Errorf("expected hash_short of length 7, got %q (len=%d)", block.HashShort, len(block.HashShort))
	}
	if block.Message == "" {
		t.Error("expected non-empty message")
	}
	if block.AgeSeconds < 0 {
		t.Errorf("expected age_seconds >= 0, got %d", block.AgeSeconds)
	}
	if block.AgeHuman == "" || block.AgeHuman == "desconocido" {
		t.Errorf("expected valid age_human, got %q", block.AgeHuman)
	}
}

func TestBuildHeadBlockNoRepo(t *testing.T) {
	block := buildHeadBlock("/nonexistent/path", time.Now())

	if block.Hash != "" {
		t.Error("expected empty hash for non-existent repo")
	}
	if block.Error == "" {
		t.Error("expected error message for non-existent repo")
	}
}

func TestBuildTimeline(t *testing.T) {
	repoRoot, cleanup := initTestRepo(t)
	defer cleanup()

	now := time.Now()
	timeline := buildTimeline(repoRoot, 3, now)

	if len(timeline) == 0 {
		t.Error("expected non-empty timeline")
	}
	if len(timeline) > 3 {
		t.Errorf("expected at most 3 entries, got %d", len(timeline))
	}

	for i, entry := range timeline {
		if entry.Hash == "" {
			t.Errorf("timeline[%d]: empty hash", i)
		}
		if entry.HashShort == "" {
			t.Errorf("timeline[%d]: empty hash_short", i)
		}
		if len(entry.HashShort) != 7 {
			t.Errorf("timeline[%d]: expected hash_short of length 7, got %q", i, entry.HashShort)
		}
		if entry.Message == "" {
			t.Errorf("timeline[%d]: empty message", i)
		}
		if entry.AgeHuman == "" {
			t.Errorf("timeline[%d]: empty age_human", i)
		}
	}
}

func TestBuildTimelineLimit(t *testing.T) {
	repoRoot, cleanup := initTestRepo(t)
	defer cleanup()

	now := time.Now()
	timeline := buildTimeline(repoRoot, 1, now)

	if len(timeline) != 1 {
		t.Errorf("expected exactly 1 entry, got %d", len(timeline))
	}
}

func TestBuildChronosOutput(t *testing.T) {
	repoRoot, cleanup := initTestRepo(t)
	defer cleanup()

	output := BuildChronosOutput(repoRoot, 5, 120)

	if output.Schema != schemaVersion {
		t.Errorf("expected schema %q, got %q", schemaVersion, output.Schema)
	}
	if output.GeneratedAt == "" {
		t.Error("expected non-empty generated_at")
	}

	// Now block checks
	if output.Now.Year == 0 {
		t.Error("now.year is zero")
	}
	if output.Now.Weekday == "" {
		t.Error("now.weekday is empty")
	}

	// Head block checks
	if output.Head.Hash == "" {
		t.Error("head.hash is empty")
	}

	// Timeline checks
	if len(output.Timeline) == 0 {
		t.Error("timeline is empty")
	}

	// Session block checks
	if output.Session.Source != "git_reflog" {
		t.Errorf("expected session source 'git_reflog', got %q", output.Session.Source)
	}

	// System block checks
	if output.System.Hostname == "" {
		t.Error("system.hostname is empty")
	}
	if output.System.GoVersion == "" {
		t.Error("system.go_version is empty")
	}
	if output.System.GitVersion == "" {
		t.Error("system.git_version is empty")
	}
}

func TestChronosOutputToJSON(t *testing.T) {
	repoRoot, cleanup := initTestRepo(t)
	defer cleanup()

	output := BuildChronosOutput(repoRoot, 3, 120)
	jsonBytes, err := output.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Verify it's valid JSON and can be deserialized
	var decoded map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("JSON parse failed: %v", err)
	}

	// Check key blocks exist
	requiredBlocks := []string{"schema", "generated_at", "now", "head", "timeline", "session", "system"}
	for _, block := range requiredBlocks {
		if _, ok := decoded[block]; !ok {
			t.Errorf("missing required block: %q", block)
		}
	}

	// Verify schema value
	if schema, ok := decoded["schema"].(string); !ok || schema != schemaVersion {
		t.Errorf("expected schema %q, got %v", schemaVersion, decoded["schema"])
	}
}

func TestAgeHumanSpanish(t *testing.T) {
	tests := []struct {
		seconds  int
		expected string
	}{
		{-10, "en el futuro"},
		{0, "hace menos de un minuto"},
		{30, "hace menos de un minuto"},
		{60, "hace 1 minuto"},
		{61, "hace 1 minuto"},
		{119, "hace 1 minuto"},
		{120, "hace 2 minutos"},
		{300, "hace 5 minutos"},
		{3540, "hace 59 minutos"},
		{3600, "hace 1 hora"},
		{3660, "hace 1 hora 1 minutos"},
		{7200, "hace 2 horas"},
		{86400, "hace 1 día"},
		{172800, "hace 2 días"},
		{2592000, "hace 1 mes"},
		{5184000, "hace 2 meses"},
		{31536000, "hace 1 año"},
		{63072000, "hace 2 años"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			delta := time.Duration(tt.seconds) * time.Second
			got := ageHumanSpanish(delta)
			if got != tt.expected {
				t.Errorf("ageHumanSpanish(%ds) = %q, want %q", tt.seconds, got, tt.expected)
			}
		})
	}
}

func TestParseGitISO(t *testing.T) {
	tests := []struct {
		input   string
		wantNil bool
	}{
		{"2026-06-18 10:30:00 -0500", false},
		{"", true},
		{"   ", true},
		{"not-a-date", true},
		{"2026-06-18T10:30:00-05:00", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseGitISO(tt.input)
			if tt.wantNil && result != nil {
				t.Errorf("expected nil for %q, got %v", tt.input, result)
			}
			if !tt.wantNil && result == nil {
				t.Errorf("expected non-nil for %q", tt.input)
			}
		})
	}
}

func TestDetectGitVersion(t *testing.T) {
	ver := detectGitVersion()
	if ver == "" || ver == "unknown" {
		t.Skip("git not available — skipping version check")
	}
	// Should be something like "2.43"
	if len(ver) < 3 {
		t.Errorf("expected git version like '2.43', got %q", ver)
	}
}

func TestDetectGoVersion(t *testing.T) {
	ver := detectGoVersion()
	if ver == "" || ver == "unknown" {
		t.Error("expected valid go version")
	}
	// Should be something like "1.24.2"
	if len(ver) < 5 {
		t.Errorf("expected go version like '1.24.2', got %q", ver)
	}
}

func TestHostTimezoneDetection(t *testing.T) {
	tz := detectHostTimezone()
	if tz.Timezone == "" {
		t.Error("expected non-empty timezone")
	}
	t.Logf("Detected timezone: %s (freshly: %v)", tz.Timezone, tz.FreshlyDetected)
}

func TestBuildDriftBlock(t *testing.T) {
	now := time.Now()
	drift := buildDriftBlock(now)

	if drift.SystemClockUTC == "" {
		t.Error("expected non-empty system_clock_utc")
	}

	// On Linux, monotonic_seconds should be > 0
	if runtime.GOOS == "linux" && drift.MonotonicSeconds <= 0 {
		t.Log("warning: monotonic_seconds is 0 on Linux (expected >0)")
	}

	if !drift.Healthy && drift.Warning == "" {
		t.Error("expected warning message when drift is unhealthy")
	}
}

func TestBuildMonotonicBlock(t *testing.T) {
	m := buildMonotonicBlock()
	if m.SecondsSinceBoot < 0 {
		t.Errorf("expected seconds_since_boot >= 0, got %f", m.SecondsSinceBoot)
	}
	if m.HoursSinceBoot < 0 {
		t.Errorf("expected hours_since_boot >= 0, got %f", m.HoursSinceBoot)
	}
}

// Test JSON field names match schema expectations
func TestJSONFieldNames(t *testing.T) {
	// Verify struct tags produce correct JSON field names
	output := ChronosOutput{
		Schema: schemaVersion,
		Now: NowBlock{
			Format24h: true,
			Timezone:  limaTZ,
		},
	}

	jsonBytes, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	jsonStr := string(jsonBytes)
	// Check snake_case field names used by chronos_gate.v1
	expectedFields := []string{
		`"schema"`,
		`"generated_at"`,
		`"format_24h"`,
		`"utc_offset"`,
		`"hash_short"`,
		`"age_human"`,
		`"age_seconds"`,
		`"age_minutes"`,
		`"is_continuation"`,
		`"is_new"`,
		`"minutes_active"`,
		`"last_action"`,
		`"last_action_at"`,
		`"system_clock_utc"`,
		`"monotonic_seconds"`,
		`"delta_seconds"`,
		`"seconds_since_boot"`,
		`"hours_since_boot"`,
		`"go_version"`,
		`"git_version"`,
		`"host_timezone"`,
		`"freshly_detected"`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("expected JSON field %s not found in output", field)
		}
	}
}
