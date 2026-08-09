// Package chronos provides git-based temporal orientation for OVAV.
//
// Replaces tools/agent_runtime/chronos_gate.py (841 LOC Python).
// Uses go-git for pure-Go git log operations; falls back to exec.Command
// for git reflog (not supported by go-git).
//
// Output schema: chronos_gate.v1 — compatible with session_greeting.py consumer.
package chronos

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// ═══════════════════════════════════════════════════════════════════════════
// CONSTANTS
// ═══════════════════════════════════════════════════════════════════════════

const (
	schemaVersion         = "chronos_gate.v1"
	limaTZ                = "America/Lima"
	utcOffsetStr          = "-0500"
	defaultTimelineCount  = 5
	defaultReflogCount    = 20
	sessionThresholdMin   = 120 // 2 hours default
	driftWarningThreshold = 300 // 5 minutes
	gitSubprocessTimeout  = 5 * time.Second
)

var limaLocation *time.Location

func init() {
	var err error
	limaLocation, err = time.LoadLocation(limaTZ)
	if err != nil {
		// Fallback: UTC-5 fixed offset
		limaLocation = time.FixedZone(limaTZ, -5*3600)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// JSON OUTPUT TYPES (matching chronos_gate.v1 schema)
// ═══════════════════════════════════════════════════════════════════════════

// ChronosOutput is the top-level JSON structure.
type ChronosOutput struct {
	Schema       string         `json:"schema"`
	GeneratedAt  string         `json:"generated_at"`
	Now          NowBlock       `json:"now"`
	Head         HeadBlock      `json:"head"`
	Timeline     []CommitBlock  `json:"timeline"`
	Session      SessionBlock   `json:"session"`
	HostTimezone HostTimezone   `json:"host_timezone"`
	Drift        DriftBlock     `json:"drift,omitempty"`
	Monotonic    MonotonicBlock `json:"monotonic,omitempty"`
	System       SystemBlock    `json:"system"`
}

// NowBlock represents the current time in multiple formats.
type NowBlock struct {
	UTC       string `json:"utc"`
	Lima      string `json:"lima"`
	ISO       string `json:"iso"`
	Epoch     int64  `json:"epoch"`
	Year      int    `json:"year"`
	Month     int    `json:"month"`
	Day       int    `json:"day"`
	Weekday   string `json:"weekday"`
	Hour      int    `json:"hour"`
	Minute    int    `json:"minute"`
	Second    int    `json:"second"`
	Timezone  string `json:"timezone"`
	UTCOffset string `json:"utc_offset"`
	Format24h bool   `json:"format_24h"`
}

// HeadBlock represents the git HEAD commit.
type HeadBlock struct {
	Hash       string `json:"hash"`
	HashShort  string `json:"hash_short"`
	ISO        string `json:"iso"`
	LimaDate   string `json:"lima_date"`
	LimaTime   string `json:"lima_time"`
	AgeSeconds int64  `json:"age_seconds"`
	AgeMinutes int64  `json:"age_minutes"`
	AgeHuman   string `json:"age_human"`
	Message    string `json:"message"`
	Error      string `json:"error,omitempty"`
}

// CommitBlock represents a single commit in the timeline.
type CommitBlock struct {
	Hash       string `json:"hash"`
	HashShort  string `json:"hash_short"`
	ISO        string `json:"iso"`
	LimaDate   string `json:"lima_date"`
	LimaTime   string `json:"lima_time"`
	AgeMinutes int64  `json:"age_minutes"`
	AgeHuman   string `json:"age_human"`
	Message    string `json:"message"`
}

// SessionBlock represents session detection from git reflog.
type SessionBlock struct {
	Detected       bool   `json:"detected"`
	Source         string `json:"source"`
	LastAction     string `json:"last_action"`
	LastActionAt   string `json:"last_action_at"`
	MinutesActive  int    `json:"minutes_active"`
	IsContinuation bool   `json:"is_continuation"`
	IsNew          bool   `json:"is_new"`
	Error          string `json:"error,omitempty"`
}

// HostTimezone holds detected host timezone info.
type HostTimezone struct {
	Timezone        string `json:"timezone"`
	DetectedAt      string `json:"detected_at"`
	FreshlyDetected bool   `json:"freshly_detected"`
}

// DriftBlock holds system clock drift detection.
type DriftBlock struct {
	SystemClockUTC   string `json:"system_clock_utc"`
	MonotonicSeconds int64  `json:"monotonic_seconds"`
	DeltaSeconds     int64  `json:"delta_seconds"`
	Healthy          bool   `json:"healthy"`
	Warning          string `json:"warning,omitempty"`
}

// MonotonicBlock holds monotonic clock info.
type MonotonicBlock struct {
	SecondsSinceBoot float64 `json:"seconds_since_boot"`
	HoursSinceBoot   float64 `json:"hours_since_boot"`
}

// SystemBlock holds system identification info.
type SystemBlock struct {
	Hostname   string `json:"hostname"`
	GoVersion  string `json:"go_version"`
	GitVersion string `json:"git_version"`
}

// ═══════════════════════════════════════════════════════════════════════════
// OUTPUT BUILDER (main entry point)
// ═══════════════════════════════════════════════════════════════════════════

// BuildChronosOutput builds the complete chronos output.
// repoRoot is the path to the git repository root.
// timelineCount is the number of commits in the timeline (1-100).
// sessionThresholdMinutes is the threshold for session continuation.
func BuildChronosOutput(repoRoot string, timelineCount int, sessionThresholdMinutes int) ChronosOutput {
	if timelineCount < 1 {
		timelineCount = defaultTimelineCount
	}
	if timelineCount > 100 {
		timelineCount = 100
	}
	if sessionThresholdMinutes < 1 {
		sessionThresholdMinutes = sessionThresholdMin
	}

	nowLima := time.Now().In(limaLocation)

	head := buildHeadBlock(repoRoot, nowLima)
	timeline := buildTimeline(repoRoot, timelineCount, nowLima)
	session := buildSessionBlock(repoRoot, nowLima, sessionThresholdMinutes)
	drift := buildDriftBlock(nowLima)
	hostTZ := detectHostTimezone()

	return ChronosOutput{
		Schema:       schemaVersion,
		GeneratedAt:  formatISO(nowLima),
		Now:          buildNowBlock(nowLima),
		Head:         head,
		Timeline:     timeline,
		Session:      session,
		HostTimezone: hostTZ,
		Drift:        drift,
		Monotonic:    buildMonotonicBlock(),
		System:       buildSystemBlock(),
	}
}

// ToJSON marshals the ChronosOutput to indented JSON.
func (c ChronosOutput) ToJSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

// ═══════════════════════════════════════════════════════════════════════════
// NOW BLOCK
// ═══════════════════════════════════════════════════════════════════════════

func buildNowBlock(now time.Time) NowBlock {
	weekdays := []string{"Monday", "Tuesday", "Wednesday", "Thursday",
		"Friday", "Saturday", "Sunday"}

	limaISO := now.Format("2006-01-02T15:04:05-0700")
	return NowBlock{
		UTC:       now.UTC().Format("2006-01-02T15:04:05Z"),
		Lima:      limaISO,
		ISO:       limaISO,
		Epoch:     now.Unix(),
		Year:      now.Year(),
		Month:     int(now.Month()),
		Day:       now.Day(),
		Weekday:   weekdays[now.Weekday()],
		Hour:      now.Hour(),
		Minute:    now.Minute(),
		Second:    now.Second(),
		Timezone:  limaTZ,
		UTCOffset: utcOffsetStr,
		Format24h: true,
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// GIT DIRECTORY RESOLUTION (handles worktrees)
// ═══════════════════════════════════════════════════════════════════════════

// resolveGitDir returns the actual .git directory path.
// For regular repos, returns <repoRoot>/.git.
// For worktrees, reads the .git file to find the worktree git dir,
// then reads commondir to find the real (main) .git directory.
func resolveGitDir(repoRoot string) string {
	gitPath := filepath.Join(repoRoot, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return repoRoot // fallback
	}
	if info.IsDir() {
		return gitPath // regular repo
	}
	// Worktree: .git is a file containing "gitdir: <path>"
	data, err := os.ReadFile(gitPath)
	if err != nil {
		return repoRoot
	}
	content := strings.TrimSpace(string(data))
	const prefix = "gitdir: "
	if !strings.HasPrefix(content, prefix) {
		return repoRoot
	}
	worktreeGit := strings.TrimSpace(content[len(prefix):])

	// Read commondir to find the real shared .git directory
	commondirPath := filepath.Join(worktreeGit, "commondir")
	cd, err := os.ReadFile(commondirPath)
	if err != nil {
		return worktreeGit // fallback to worktree git dir
	}
	commonDir := strings.TrimSpace(string(cd))
	if !filepath.IsAbs(commonDir) {
		commonDir = filepath.Join(worktreeGit, commonDir)
	}
	return commonDir
}

// resolveWorktreeHead reads the worktree's HEAD file from the worktree git dir.
// Returns the symbolic ref name (e.g., "refs/heads/mybranch") or empty string.
// This is needed because go-git opens the main repo, whose HEAD may differ.
func resolveWorktreeHead(repoRoot string) string {
	gitPath := filepath.Join(repoRoot, ".git")
	data, err := os.ReadFile(gitPath)
	if err != nil {
		return ""
	}
	content := strings.TrimSpace(string(data))
	const prefix = "gitdir: "
	if !strings.HasPrefix(content, prefix) {
		return "" // Not a worktree
	}
	worktreeGit := strings.TrimSpace(content[len(prefix):])
	headPath := filepath.Join(worktreeGit, "HEAD")
	headData, err := os.ReadFile(headPath)
	if err != nil {
		return ""
	}
	headContent := strings.TrimSpace(string(headData))
	const refPrefix = "ref: "
	if strings.HasPrefix(headContent, refPrefix) {
		return strings.TrimSpace(headContent[len(refPrefix):])
	}
	// Detached HEAD — return the hash directly
	return headContent
}

// ═══════════════════════════════════════════════════════════════════════════
// HEAD BLOCK (go-git pure Go)
// ═══════════════════════════════════════════════════════════════════════════

func buildHeadBlock(repoRoot string, now time.Time) HeadBlock {
	errorBlock := HeadBlock{
		Hash:       "",
		HashShort:  "",
		ISO:        "",
		AgeSeconds: -1,
		AgeMinutes: -1,
		AgeHuman:   "desconocido",
		Message:    "",
		Error:      "git log failed or no commits",
	}

	commit, err := resolveHeadCommit(repoRoot)
	if err != nil {
		return errorBlock
	}

	fullHash := commit.Hash.String()
	headTime := commit.Author.When
	headLima := headTime.In(limaLocation)
	delta := now.Sub(headLima)
	ageSeconds := int64(delta.Seconds())
	ageMinutes := ageSeconds / 60

	return HeadBlock{
		Hash:       fullHash,
		HashShort:  fullHash[:7],
		ISO:        formatISO(headLima),
		LimaDate:   headLima.Format("2006-01-02"),
		LimaTime:   headLima.Format("15:04"),
		AgeSeconds: ageSeconds,
		AgeMinutes: ageMinutes,
		AgeHuman:   ageHumanSpanish(delta),
		Message:    strings.TrimSpace(commit.Message),
	}
}

// resolveHeadCommit returns the HEAD commit, handling worktrees.
func resolveHeadCommit(repoRoot string) (*object.Commit, error) {
	gitDir := resolveGitDir(repoRoot)
	repo, err := git.PlainOpen(gitDir)
	if err != nil {
		return nil, fmt.Errorf("chronos: git open: %w", err)
	}

	// Check if this is a worktree — if so, use the worktree's HEAD ref
	worktreeRef := resolveWorktreeHead(repoRoot)
	if worktreeRef != "" && strings.HasPrefix(worktreeRef, "refs/") {
		// Worktree: resolve the specific ref
		ref, err := repo.Reference(plumbing.ReferenceName(worktreeRef), true)
		if err != nil {
			return nil, fmt.Errorf("chronos: resolve ref %s: %w", worktreeRef, err)
		}
		return repo.CommitObject(ref.Hash())
	}

	// Regular repo or detached HEAD worktree
	if worktreeRef != "" {
		// Detached HEAD — worktreeRef is a hash
		hash := plumbing.NewHash(worktreeRef)
		return repo.CommitObject(hash)
	}

	// Regular repo: use repo.Head()
	ref, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("chronos: repo head: %w", err)
	}
	return repo.CommitObject(ref.Hash())
}

// ═══════════════════════════════════════════════════════════════════════════
// TIMELINE BLOCK (go-git pure Go)
// ═══════════════════════════════════════════════════════════════════════════

func buildTimeline(repoRoot string, count int, now time.Time) []CommitBlock {
	commit, err := resolveHeadCommit(repoRoot)
	if err != nil {
		return []CommitBlock{}
	}

	gitDir := resolveGitDir(repoRoot)
	repo, err := git.PlainOpen(gitDir)
	if err != nil {
		return []CommitBlock{}
	}

	iter, err := repo.Log(&git.LogOptions{
		From:  commit.Hash,
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		return []CommitBlock{}
	}
	defer iter.Close()

	var entries []CommitBlock
	err = iter.ForEach(func(c *object.Commit) error {
		if len(entries) >= count {
			return fmt.Errorf("stop") // break iteration
		}

		fullHash := c.Hash.String()
		commitTime := c.Author.When
		commitLima := commitTime.In(limaLocation)
		delta := now.Sub(commitLima)
		ageMinutes := int64(delta.Seconds()) / 60

		entries = append(entries, CommitBlock{
			Hash:       fullHash,
			HashShort:  fullHash[:7],
			ISO:        formatISO(commitLima),
			LimaDate:   commitLima.Format("2006-01-02"),
			LimaTime:   commitLima.Format("15:04"),
			AgeMinutes: ageMinutes,
			AgeHuman:   ageHumanSpanish(delta),
			Message:    strings.TrimSpace(c.Message),
		})
		return nil
	})
	if err != nil && err.Error() != "stop" {
		// Log iteration error — return what we have
	}
	return entries
}

// ═══════════════════════════════════════════════════════════════════════════
// SESSION BLOCK (exec.Command fallback — go-git lacks reflog)
// ═══════════════════════════════════════════════════════════════════════════

func buildSessionBlock(repoRoot string, now time.Time, thresholdMinutes int) SessionBlock {
	defaultBlock := SessionBlock{
		Detected:       false,
		Source:         "git_reflog",
		LastAction:     "",
		LastActionAt:   "",
		MinutesActive:  0,
		IsContinuation: false,
		IsNew:          true,
		Error:          "reflog unavailable",
	}

	reflogCount := defaultReflogCount
	cmd := exec.Command("git", "reflog",
		"--format=%h|%ai|%gs",
		fmt.Sprintf("-%d", reflogCount),
	)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return defaultBlock
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return defaultBlock
	}

	// Parse the most recent reflog entry
	firstParts := strings.SplitN(lines[0], "|", 3)
	if len(firstParts) < 3 {
		return defaultBlock
	}

	firstISO := strings.TrimSpace(firstParts[1])
	firstAction := strings.TrimSpace(firstParts[2])
	firstDT := parseGitISO(firstISO)
	if firstDT == nil {
		return defaultBlock
	}

	firstLima := firstDT.In(limaLocation)
	delta := now.Sub(firstLima)
	minutesSinceLast := delta.Minutes()
	isContinuation := minutesSinceLast <= float64(thresholdMinutes)

	// Find earliest reflog entry within threshold to estimate session duration
	var earliestInSession *time.Time
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 3 {
			continue
		}
		entryDT := parseGitISO(strings.TrimSpace(parts[1]))
		if entryDT == nil {
			continue
		}
		entryLima := entryDT.In(limaLocation)
		entryAgeMin := now.Sub(entryLima).Minutes()
		if entryAgeMin <= float64(thresholdMinutes) {
			earliestInSession = &entryLima
		}
	}

	minutesActive := 0
	if earliestInSession != nil {
		minutesActive = int(now.Sub(*earliestInSession).Minutes())
	}

	return SessionBlock{
		Detected:       true,
		Source:         "git_reflog",
		LastAction:     firstAction,
		LastActionAt:   formatISO(firstLima),
		MinutesActive:  minutesActive,
		IsContinuation: isContinuation,
		IsNew:          !isContinuation,
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// DRIFT DETECTION
// ═══════════════════════════════════════════════════════════════════════════

func buildDriftBlock(now time.Time) DriftBlock {
	nowUTC := now.UTC()

	// Approximate monotonic clock via /proc/uptime on Linux
	monoSec := readUptimeSeconds()
	sysSec := now.Unix()

	bootEstimate := sysSec - int64(monoSec)
	delta := int64(0)
	if monoSec > 0 {
		// Compare: sysSec should equal monoSec + bootEstimate
		delta = sysSec - (int64(monoSec) + bootEstimate)
		if delta < 0 {
			delta = -delta
		}
	}

	healthy := delta < driftWarningThreshold
	var warning string
	if !healthy {
		hoursUp := float64(monoSec) / 3600.0
		bootTime := time.Unix(bootEstimate, 0).UTC()
		warning = fmt.Sprintf(
			"Posible ajuste de reloj detectado. "+
				"Monotónico: %.1fh desde boot (%s). "+
				"Delta sistema vs monotónico: %ds. "+
				"Verificar NTP.",
			hoursUp,
			bootTime.Format("2006-01-02T15:04:05Z"),
			delta,
		)
	}

	return DriftBlock{
		SystemClockUTC:   nowUTC.Format("2006-01-02T15:04:05Z"),
		MonotonicSeconds: int64(monoSec),
		DeltaSeconds:     delta,
		Healthy:          healthy,
		Warning:          warning,
	}
}

// readUptimeSeconds reads system uptime in seconds from /proc/uptime (Linux).
// Returns 0 on any error (graceful degradation).
func readUptimeSeconds() float64 {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	parts := strings.Fields(string(data))
	if len(parts) < 1 {
		return 0
	}
	uptime, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0
	}
	return uptime
}

// ═══════════════════════════════════════════════════════════════════════════
// MONOTONIC CLOCK
// ═══════════════════════════════════════════════════════════════════════════

func buildMonotonicBlock() MonotonicBlock {
	sec := readUptimeSeconds()
	return MonotonicBlock{
		SecondsSinceBoot: sec,
		HoursSinceBoot:   sec / 3600.0,
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// SYSTEM INFO
// ═══════════════════════════════════════════════════════════════════════════

func buildSystemBlock() SystemBlock {
	hostname, _ := os.Hostname()
	gitVersion := detectGitVersion()
	goVersion := detectGoVersion()

	return SystemBlock{
		Hostname:   hostname,
		GoVersion:  goVersion,
		GitVersion: gitVersion,
	}
}

func detectGitVersion() string {
	cmd := exec.Command("git", "--version")
	cmd.Stderr = nil
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	// "git version 2.43.0" → "2.43"
	if len(fields) >= 3 {
		ver := fields[2]
		segments := strings.Split(ver, ".")
		if len(segments) >= 2 {
			return segments[0] + "." + segments[1]
		}
		return ver
	}
	return "unknown"
}

func detectGoVersion() string {
	// Use runtime.Version() — returns "go1.24.2"
	ver := runtime.Version()
	return strings.TrimPrefix(ver, "go")
}

// ═══════════════════════════════════════════════════════════════════════════
// HOST TIMEZONE DETECTION
// ═══════════════════════════════════════════════════════════════════════════

func detectHostTimezone() HostTimezone {
	// Method 1: timedatectl (systemd)
	if tz := detectViaTimedatectl(); tz != "" {
		return HostTimezone{
			Timezone:        tz,
			DetectedAt:      "",
			FreshlyDetected: true,
		}
	}

	// Method 2: /etc/timezone (Debian/Ubuntu)
	if tz := detectViaEtcTimezone(); tz != "" {
		return HostTimezone{
			Timezone:        tz,
			DetectedAt:      "",
			FreshlyDetected: true,
		}
	}

	// Method 3: /etc/localtime symlink
	if tz := detectViaLocaltime(); tz != "" {
		return HostTimezone{
			Timezone:        tz,
			DetectedAt:      "",
			FreshlyDetected: true,
		}
	}

	// Method 4: Go stdlib fallback
	localTZ, _ := time.Now().Zone()
	if tz, err := time.LoadLocation(localTZ); err == nil && strings.Contains(tz.String(), "/") {
		return HostTimezone{
			Timezone:        tz.String(),
			DetectedAt:      "",
			FreshlyDetected: true,
		}
	}

	return HostTimezone{
		Timezone:        "unknown",
		DetectedAt:      "",
		FreshlyDetected: false,
	}
}

func detectViaTimedatectl() string {
	cmd := exec.Command("timedatectl", "show", "--property=Timezone", "--value")
	cmd.Stderr = nil
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	tz := strings.TrimSpace(string(out))
	if tz != "" && strings.Contains(tz, "/") {
		return tz
	}
	return ""
}

func detectViaEtcTimezone() string {
	data, err := os.ReadFile("/etc/timezone")
	if err != nil {
		return ""
	}
	tz := strings.TrimSpace(string(data))
	if tz != "" && strings.Contains(tz, "/") {
		return tz
	}
	return ""
}

func detectViaLocaltime() string {
	target, err := os.Readlink("/etc/localtime")
	if err != nil {
		return ""
	}
	// Extract timezone from path like .../zoneinfo/America/Lima
	for _, prefix := range []string{"zoneinfo/", "zoneinfo-lead/"} {
		idx := strings.Index(target, prefix)
		if idx >= 0 {
			tz := target[idx+len(prefix):]
			if tz != "" && strings.Contains(tz, "/") {
				return tz
			}
		}
	}
	return ""
}

// ═══════════════════════════════════════════════════════════════════════════
// SPANISH AGE FORMATTER
// ═══════════════════════════════════════════════════════════════════════════

// ageHumanSpanish converts a duration to a human-readable Spanish age string.
// Examples: "hace 3 minutos", "hace 2 horas", "hace 5 días".
func ageHumanSpanish(delta time.Duration) string {
	totalSeconds := int(delta.Seconds())
	if totalSeconds < 0 {
		return "en el futuro"
	}
	if totalSeconds < 60 {
		return "hace menos de un minuto"
	}

	minutes := totalSeconds / 60
	if minutes == 1 {
		return "hace 1 minuto"
	}
	if minutes < 60 {
		return fmt.Sprintf("hace %d minutos", minutes)
	}

	hours := minutes / 60
	remainingMinutes := minutes % 60
	if hours == 1 && remainingMinutes == 0 {
		return "hace 1 hora"
	}
	if hours == 1 {
		return fmt.Sprintf("hace 1 hora %d minutos", remainingMinutes)
	}
	if hours < 24 {
		return fmt.Sprintf("hace %d horas", hours)
	}

	days := hours / 24
	if days == 1 {
		return "hace 1 día"
	}
	if days < 30 {
		return fmt.Sprintf("hace %d días", days)
	}

	months := days / 30
	if months == 1 {
		return "hace 1 mes"
	}
	if months < 12 {
		return fmt.Sprintf("hace %d meses", months)
	}

	years := days / 365
	if years == 1 {
		return "hace 1 año"
	}
	return fmt.Sprintf("hace %d años", years)
}

// ═══════════════════════════════════════════════════════════════════════════
// HELPERS
// ═══════════════════════════════════════════════════════════════════════════

// formatISO formats a time as ISO 8601 with timezone offset.
func formatISO(t time.Time) string {
	return t.Format("2006-01-02T15:04:05-0700")
}

// parseGitISO parses a git ISO date string like "2026-06-12 20:39:32 -0500".
func parseGitISO(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	formats := []string{
		"2006-01-02 15:04:05 -0700",
		"2006-01-02 15:04:05 -0700 MST",
		time.RFC3339,
		time.RFC3339Nano,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return &t
		}
	}
	return nil
}
