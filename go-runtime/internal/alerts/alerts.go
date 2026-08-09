// Package alerts provides the OVAV alert system for security and integrity issues.
//
// Alerts are persistent YAML files stored in .ovav/alerts/ that survive worktree
// cleanup. Each alert represents a security issue that must be resolved before
// merge (owd). The owd command checks for active alerts and blocks the merge.
//
// Alert lifecycle:
//   - CREATED → when a validator detects a critical issue (e.g., secret in code)
//   - ACKNOWLEDGED → when the issue is reviewed (optional state)
//   - RESOLVED → when the issue is fixed (alert file deleted)
//   - EXPIRED → automatic cleanup after 90 days if unresolved (configurable)
//
// Integration:
//   - secrets_hygiene validator creates SECRETS alerts
//   - protected_branch validator creates WAIVER alerts
//   - owd checks for active alerts before merge
//   - cockpit TUI shows active alert count as banner
package alerts

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ── Types ──────────────────────────────────────────────────────────────────────

// Severity defines the urgency level of an alert.
type Severity string

const (
	SevCritical Severity = "CRITICAL" // blocks merge, requires immediate action
	SevHigh     Severity = "HIGH"     // should be resolved before merge
	SevMedium   Severity = "MEDIUM"   // should be resolved soon
	SevLow      Severity = "LOW"      // informational, non-blocking
	SevInfo     Severity = "INFO"     // trace-level signal, no action needed
)

// Category defines the type of alert.
type Category string

const (
	CatSecrets     Category = "SECRETS"      // plaintext secret detected
	CatIntegrity   Category = "INTEGRITY"    // file integrity violation
	CatWaiver      Category = "WAIVER"       // missing/existing waiver
	CatBranch      Category = "BRANCH"       // branch protection violation
	CatWorkspace   Category = "WORKSPACE"    // workspace safety issue
	CatSupplyChain Category = "SUPPLY_CHAIN" // supply chain issue
	CatConfig      Category = "CONFIG"       // configuration drift
)

// Alert represents a single security or integrity issue that needs attention.
type Alert struct {
	ID        string   `yaml:"id" json:"id"`
	Category  Category `yaml:"category" json:"category"`
	Severity  Severity `yaml:"severity" json:"severity"`
	Title     string   `yaml:"title" json:"title"`
	Detail    string   `yaml:"detail" json:"detail"`
	File      string   `yaml:"file,omitempty" json:"file,omitempty"`
	Line      int      `yaml:"line,omitempty" json:"line,omitempty"`
	CreatedAt string   `yaml:"created_at" json:"created_at"`
	CreatedBy string   `yaml:"created_by" json:"created_by"`
	Resolved  bool     `yaml:"resolved" json:"resolved"`
}

// ── Manager ────────────────────────────────────────────────────────────────────

// Manager manages the OVAV alert system.
type Manager struct {
	RepoRoot string
}

// NewManager creates an alert manager for the given repository.
func NewManager(repoRoot string) *Manager {
	return &Manager{RepoRoot: repoRoot}
}

// alertsDir returns the path to .ovav/alerts/.
func (m *Manager) alertsDir() string {
	return filepath.Join(m.RepoRoot, ".ovav", "alerts")
}

// ── Create / Resolve ───────────────────────────────────────────────────────────

// Create creates a new alert and persists it to .ovav/alerts/<id>.yaml.
// Returns the alert ID for reference.
func (m *Manager) Create(category Category, severity Severity, title, detail, file string, line int) (*Alert, error) {
	now := time.Now().UTC()
	id := fmt.Sprintf("%s_%s_%d", category, now.Format("2006-01-02T150405"), now.Nanosecond())
	// Sanitize ID for filesystem
	id = strings.ReplaceAll(id, ":", "")
	id = strings.ReplaceAll(id, ".", "_")

	alert := &Alert{
		ID:        id,
		Category:  category,
		Severity:  severity,
		Title:     title,
		Detail:    detail,
		File:      file,
		Line:      line,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		CreatedBy: "ovav-hook", // or "ovav-user" for manual creation
	}

	if err := m.save(alert); err != nil {
		return nil, fmt.Errorf("create alert: %w", err)
	}

	return alert, nil
}

// Resolve marks an alert as resolved and removes the alert file.
func (m *Manager) Resolve(id string) error {
	alertPath := filepath.Join(m.alertsDir(), id+".yaml")
	if err := os.Remove(alertPath); err != nil {
		if os.IsNotExist(err) {
			return nil // already resolved
		}
		return fmt.Errorf("resolve alert: %w", err)
	}
	return nil
}

// ── Query ──────────────────────────────────────────────────────────────────────

// Active returns all unresolved alerts sorted by severity (critical first).
func (m *Manager) Active() ([]Alert, error) {
	dir := m.alertsDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read alerts dir: %w", err)
	}

	var alerts []Alert
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		alertPath := filepath.Join(dir, entry.Name())
		alert, err := m.load(alertPath)
		if err != nil {
			continue
		}
		if !alert.Resolved {
			alerts = append(alerts, *alert)
		}
	}

	// Sort by severity: CRITICAL > HIGH > MEDIUM > LOW
	sort.Slice(alerts, func(i, j int) bool {
		return severityWeight(alerts[i].Severity) > severityWeight(alerts[j].Severity)
	})

	return alerts, nil
}

// Count returns the count of active alerts grouped by severity.
func (m *Manager) Count() (map[Severity]int, error) {
	alerts, err := m.Active()
	if err != nil {
		return nil, err
	}
	counts := make(map[Severity]int)
	for _, a := range alerts {
		counts[a.Severity]++
	}
	return counts, nil
}

// HasBlocking returns true if there are any CRITICAL or HIGH severity alerts
// that should block a merge.
func (m *Manager) HasBlocking() (bool, error) {
	alerts, err := m.Active()
	if err != nil {
		return false, err
	}
	for _, a := range alerts {
		if a.Severity == SevCritical || a.Severity == SevHigh {
			return true, nil
		}
	}
	return false, nil
}

// ── Persistence ────────────────────────────────────────────────────────────────

func (m *Manager) save(alert *Alert) error {
	dir := m.alertsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create alerts dir: %w", err)
	}

	alertPath := filepath.Join(dir, alert.ID+".yaml")
	data, err := yaml.Marshal(alert)
	if err != nil {
		return fmt.Errorf("marshal alert: %w", err)
	}

	if err := os.WriteFile(alertPath, data, 0644); err != nil {
		return fmt.Errorf("write alert: %w", err)
	}

	return nil
}

func (m *Manager) load(path string) (*Alert, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var alert Alert
	if err := yaml.Unmarshal(data, &alert); err != nil {
		return nil, err
	}
	return &alert, nil
}

// ── Helpers ────────────────────────────────────────────────────────────────────

func severityWeight(s Severity) int {
	switch s {
	case SevCritical:
		return 4
	case SevHigh:
		return 3
	case SevMedium:
		return 2
	case SevLow:
		return 1
	case SevInfo:
		return 0
	default:
		return 0
	}
}

// FormatHuman returns a human-readable summary of active alerts.
func FormatHuman(alerts []Alert) string {
	if len(alerts) == 0 {
		return "✅ No active alerts."
	}
	var b strings.Builder
	critical := 0
	high := 0
	for _, a := range alerts {
		if a.Severity == SevCritical {
			critical++
		} else if a.Severity == SevHigh {
			high++
		}
	}
	icon := "🔴"
	if critical == 0 {
		icon = "🟡"
	}
	b.WriteString(fmt.Sprintf("%s %d active alert(s) — %d critical, %d high\n", icon, len(alerts), critical, high))
	for _, a := range alerts {
		b.WriteString(fmt.Sprintf("  %s [%s] %s — %s:%d\n", alertIcon(a.Severity), a.Category, a.Title, a.File, a.Line))
	}
	return b.String()
}

func alertIcon(s Severity) string {
	switch s {
	case SevCritical:
		return "🔴"
	case SevHigh:
		return "🟠"
	case SevMedium:
		return "🟡"
	case SevLow:
		return "🔵"
	case SevInfo:
		return "🟢"
	default:
		return "⚪"
	}
}
