// Package economy provides budget/usage data from .ovav/economy/budget_status.json.
//
// CAPA 9.x: Economy data integration for ovav status.
package economy

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/cli"
	yaml "gopkg.in/yaml.v3"
)

// BudgetStatus mirrors the flat JSON structure in .ovav/economy/budget_status.json.
type BudgetStatus struct {
	UpdatedAt           string  `json:"updated_at"`
	Model               string  `json:"model"`
	SessionCostUSD      float64 `json:"session_cost_usd"`
	MonthlyCostUSD      float64 `json:"monthly_cost_usd"`
	SessionBudgetUSD    float64 `json:"session_budget_usd"`
	MonthlyBudgetUSD    float64 `json:"monthly_budget_usd"`
	SessionPct          float64 `json:"session_pct"`
	MonthlyPct          float64 `json:"monthly_pct"`
	SessionRemainingUSD float64 `json:"session_remaining_usd"`
	AlertLevel          string  `json:"alert_level"`
	SessionTurns        int     `json:"session_turns"`
}

// Load reads budget_status.json from the OVAV root.
// Returns nil if the file doesn't exist (graceful degradation).
func Load(repoRoot string) (*BudgetStatus, error) {
	path := filepath.Join(repoRoot, ".ovav", "economy", "budget_status.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no data, not an error
		}
		return nil, fmt.Errorf("reading budget status: %w", err)
	}

	var bs BudgetStatus
	if err := json.Unmarshal(data, &bs); err != nil {
		return nil, fmt.Errorf("parsing budget status: %w", err)
	}
	return &bs, nil
}

// ANSI color codes for alert_level display.
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
)

// ColoredAlert returns alert_level with ANSI color.
func ColoredAlert(level string) string {
	switch level {
	case "green":
		return colorGreen + level + colorReset
	case "yellow":
		return colorYellow + level + colorReset
	case "red":
		return colorRed + level + colorReset
	default:
		return level
	}
}

// FormatHuman returns a human-readable economy section string.
func FormatHuman(bs *BudgetStatus) string {
	if bs == nil {
		return ""
	}

	return fmt.Sprintf(`  Model:        %s
  Session cost: $%.4f  |  Remaining: $%.2f
  Alert level:  %s
  Session turns: %d
  Budget used:  %.1f%% session  |  %.1f%% monthly`,
		bs.Model,
		bs.SessionCostUSD, bs.SessionRemainingUSD,
		ColoredAlert(bs.AlertLevel),
		bs.SessionTurns,
		bs.SessionPct, bs.MonthlyPct,
	)
}

// ── Ledger entry (cost_ledger.jsonl) ────────────────────────────────────────

// LedgerEntry represents a single line in cost_ledger.jsonl.
type LedgerEntry struct {
	Timestamp       string  `json:"timestamp"`
	UnixTS          int64   `json:"unix_ts"`
	Model           string  `json:"model"`
	Provider        string  `json:"provider"`
	TokensIn        int     `json:"tokens_in"`
	TokensOut       int     `json:"tokens_out"`
	TotalTokens     int     `json:"total_tokens"`
	CacheHit        bool    `json:"cache_hit"`
	CostUSD         float64 `json:"cost_usd"`
	CacheSavingsUSD float64 `json:"cache_savings_usd"`
	TaskType        string  `json:"task_type"`
	Area            string  `json:"area"`
	SessionID       string  `json:"session_id"`
	Lead            string  `json:"lead"`
	Project         string  `json:"project"`
	Result          string  `json:"result"`
	Tier            string  `json:"tier"`
}

// SessionStats mirrors cost_tracker.get_session_stats() — 24h window.
type SessionStats struct {
	Period               string             `json:"period"`
	Turns                int                `json:"turns"`
	TotalCostUSD         float64            `json:"total_cost_usd"`
	TotalTokens          int                `json:"total_tokens"`
	CacheSavingsUSD      float64            `json:"cache_savings_usd"`
	CacheHitRate         float64            `json:"cache_hit_rate"`
	ByModel              map[string]float64 `json:"by_model"`
	ByArea               map[string]float64 `json:"by_area"`
	EstimatedMonthlyCost float64            `json:"estimated_monthly_cost"`
}

// MonthlyStats mirrors cost_tracker.get_monthly_stats() — current month.
type MonthlyStats struct {
	Period       string  `json:"period"`
	Month        string  `json:"month"`
	Turns        int     `json:"turns"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	TotalTokens  int     `json:"total_tokens"`
}

// BudgetConsumption extends budget data with remaining amounts and alert thresholds.
// Returned by GetBudgetConsumption.
type BudgetConsumption struct {
	SessionCostUSD      float64   `json:"session_cost_usd"`
	MonthlyCostUSD      float64   `json:"monthly_cost_usd"`
	SessionBudgetUSD    float64   `json:"session_budget_usd"`
	MonthlyBudgetUSD    float64   `json:"monthly_budget_usd"`
	SessionPct          float64   `json:"session_pct"`
	MonthlyPct          float64   `json:"monthly_pct"`
	SessionRemainingUSD float64   `json:"session_remaining_usd"`
	MonthlyRemainingUSD float64   `json:"monthly_remaining_usd"`
	AlertLevel          string    `json:"alert_level"`
	AlertThresholds     []float64 `json:"alert_thresholds"`
	SessionTurns        int       `json:"session_turns"`
	UpdatedAt           string    `json:"updated_at"`
}

// ── YAML config for budget defaults ──────────────────────────────────────────

// budgetDefaultsSection mirrors the budget_defaults block in provider_prices.yaml.
type budgetDefaultsSection struct {
	MonthlyBudgetUSD   float64   `yaml:"monthly_budget_usd"`
	PerSessionLimitUSD float64   `yaml:"per_session_limit_usd"`
	AlertThresholds    []float64 `yaml:"alert_thresholds"`
	AutoStopThreshold  float64   `yaml:"auto_stop_threshold"`
}

type providerPricesFile struct {
	BudgetDefaults budgetDefaultsSection `yaml:"budget_defaults"`
}

// ── Rounding helpers ─────────────────────────────────────────────────────────

func round4(v float64) float64 { return math.Round(v*10000) / 10000 }
func round2(v float64) float64 { return math.Round(v*100) / 100 }
func round1(v float64) float64 { return math.Round(v*10) / 10 }

// ── Ledger reading ───────────────────────────────────────────────────────────

// readLedger reads all entries from cost_ledger.jsonl.
// Returns empty slice if the file does not exist.
func readLedger(repoRoot string) ([]LedgerEntry, error) {
	path := filepath.Join(repoRoot, ".ovav", "economy", "cost_ledger.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading cost ledger: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	entries := make([]LedgerEntry, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry LedgerEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // skip malformed lines (graceful)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// parseTimestamp tries common ISO 8601 variants.
func parseTimestamp(s string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,           // "2006-01-02T15:04:05.999999999Z07:00"
		time.RFC3339,               // "2006-01-02T15:04:05Z07:00"
		"2006-01-02T15:04:05-0700", // no colon in tz
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse timestamp: %s", s)
}

// ── Statistics (migrated from cost_tracker.py) ───────────────────────────────

// GetSessionStats returns cost statistics for the last 24 hours.
func GetSessionStats(repoRoot string) (*SessionStats, error) {
	entries, err := readLedger(repoRoot)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	cutoff := now.Add(-24 * time.Hour)

	stats := &SessionStats{
		Period:  "24h",
		ByModel: make(map[string]float64),
		ByArea:  make(map[string]float64),
	}

	cacheHits := 0
	for _, e := range entries {
		ts, err := parseTimestamp(e.Timestamp)
		if err != nil || !ts.After(cutoff) {
			continue
		}
		stats.TotalCostUSD += e.CostUSD
		stats.TotalTokens += e.TotalTokens
		stats.CacheSavingsUSD += e.CacheSavingsUSD
		stats.Turns++
		stats.ByModel[e.Model] += e.CostUSD
		stats.ByArea[e.Area] += e.CostUSD
		if e.CacheHit {
			cacheHits++
		}
	}

	if stats.Turns > 0 {
		stats.CacheHitRate = float64(cacheHits) / float64(stats.Turns)
	}
	stats.TotalCostUSD = round4(stats.TotalCostUSD)
	stats.CacheSavingsUSD = round4(stats.CacheSavingsUSD)
	stats.CacheHitRate = round2(stats.CacheHitRate)
	for k, v := range stats.ByModel {
		stats.ByModel[k] = round4(v)
	}
	for k, v := range stats.ByArea {
		stats.ByArea[k] = round4(v)
	}
	stats.EstimatedMonthlyCost = round2(stats.TotalCostUSD * 30)

	return stats, nil
}

// GetMonthlyStats returns cost statistics for the current month.
func GetMonthlyStats(repoRoot string) (*MonthlyStats, error) {
	entries, err := readLedger(repoRoot)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	stats := &MonthlyStats{
		Period: "month",
		Month:  now.Format("2006-01"),
	}

	for _, e := range entries {
		ts, err := parseTimestamp(e.Timestamp)
		if err != nil || ts.Before(monthStart) {
			continue
		}
		stats.TotalCostUSD += e.CostUSD
		stats.TotalTokens += e.TotalTokens
		stats.Turns++
	}

	stats.TotalCostUSD = round4(stats.TotalCostUSD)

	return stats, nil
}

// ── Budget config (migrated from budget_governor.py) ─────────────────────────

// loadBudgetConfig reads budget defaults from provider_prices.yaml.
// Falls back to hardcoded defaults if the file is missing or unreadable.
func loadBudgetConfig(repoRoot string) budgetDefaultsSection {
	path := filepath.Join(repoRoot, ".ovav", "economy", "provider_prices.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultBudgetConfig()
	}

	var pf providerPricesFile
	if err := yaml.Unmarshal(data, &pf); err != nil {
		return defaultBudgetConfig()
	}

	cfg := pf.BudgetDefaults
	// Validate: if critical fields are zero, use defaults
	if cfg.MonthlyBudgetUSD == 0 {
		cfg.MonthlyBudgetUSD = 200.0
	}
	if cfg.PerSessionLimitUSD == 0 {
		cfg.PerSessionLimitUSD = 10.0
	}
	if len(cfg.AlertThresholds) == 0 {
		cfg.AlertThresholds = []float64{0.70, 0.85, 0.95}
	}
	return cfg
}

func defaultBudgetConfig() budgetDefaultsSection {
	return budgetDefaultsSection{
		MonthlyBudgetUSD:   200.0,
		PerSessionLimitUSD: 10.0,
		AlertThresholds:    []float64{0.70, 0.85, 0.95},
		AutoStopThreshold:  1.0,
	}
}

// ── Budget consumption (migrated from budget_governor.get_budget_consumption)

// GetBudgetConsumption calculates % consumption vs budget and alert level.
// Reads from cost_ledger.jsonl and provider_prices.yaml.
func GetBudgetConsumption(repoRoot string) (*BudgetConsumption, error) {
	config := loadBudgetConfig(repoRoot)

	session, err := GetSessionStats(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("session stats: %w", err)
	}

	monthly, err := GetMonthlyStats(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("monthly stats: %w", err)
	}

	sessionLimit := config.PerSessionLimitUSD
	monthlyLimit := config.MonthlyBudgetUSD
	thresholds := config.AlertThresholds

	sessionPct := 0.0
	if sessionLimit > 0 {
		sessionPct = (session.TotalCostUSD / sessionLimit) * 100
	}
	monthlyPct := 0.0
	if monthlyLimit > 0 {
		monthlyPct = (monthly.TotalCostUSD / monthlyLimit) * 100
	}

	// Determine alert level (worst of session/monthly).
	maxRatio := sessionPct / 100
	if monthlyPct/100 > maxRatio {
		maxRatio = monthlyPct / 100
	}

	alertLevel := "green"
	if len(thresholds) >= 3 && maxRatio >= thresholds[2] {
		alertLevel = "red"
	} else if len(thresholds) >= 2 && maxRatio >= thresholds[1] {
		alertLevel = "orange"
	} else if len(thresholds) >= 1 && maxRatio >= thresholds[0] {
		alertLevel = "yellow"
	}

	return &BudgetConsumption{
		SessionCostUSD:      round4(session.TotalCostUSD),
		MonthlyCostUSD:      round4(monthly.TotalCostUSD),
		SessionBudgetUSD:    sessionLimit,
		MonthlyBudgetUSD:    monthlyLimit,
		SessionPct:          round1(sessionPct),
		MonthlyPct:          round1(monthlyPct),
		SessionRemainingUSD: round4(max(0, sessionLimit-session.TotalCostUSD)),
		MonthlyRemainingUSD: round4(max(0, monthlyLimit-monthly.TotalCostUSD)),
		AlertLevel:          alertLevel,
		AlertThresholds:     thresholds,
		SessionTurns:        session.Turns,
		UpdatedAt:           time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// ── Budget status writer (migrated from economy_runtime.update_budget_status)

// UpdateBudgetStatus writes budget_status.json from current ledger data.
// model is the active model ID (e.g. "opencode-go/deepseek-v4-pro").
// If model is empty, "unknown" is used.
func UpdateBudgetStatus(repoRoot string, model string) error {
	if model == "" {
		model = "unknown"
	}

	consumption, err := GetBudgetConsumption(repoRoot)
	if err != nil {
		// Fallback: write zero-status
		consumption = &BudgetConsumption{
			SessionBudgetUSD:    10.0,
			MonthlyBudgetUSD:    200.0,
			SessionRemainingUSD: 10.0,
			AlertLevel:          "green",
			UpdatedAt:           time.Now().UTC().Format(time.RFC3339),
		}
	}

	status := BudgetStatus{
		UpdatedAt:           consumption.UpdatedAt,
		Model:               model,
		SessionCostUSD:      consumption.SessionCostUSD,
		MonthlyCostUSD:      consumption.MonthlyCostUSD,
		SessionBudgetUSD:    consumption.SessionBudgetUSD,
		MonthlyBudgetUSD:    consumption.MonthlyBudgetUSD,
		SessionPct:          consumption.SessionPct,
		MonthlyPct:          consumption.MonthlyPct,
		SessionRemainingUSD: consumption.SessionRemainingUSD,
		AlertLevel:          consumption.AlertLevel,
		SessionTurns:        consumption.SessionTurns,
	}

	path := filepath.Join(repoRoot, ".ovav", "economy", "budget_status.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating economy dir: %w", err)
	}

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling budget status: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing budget status: %w", err)
	}

	return nil
}

// MustFindRepoRoot is a convenience wrapper for cli.MustFindRepoRoot.
func MustFindRepoRoot() string {
	return cli.MustFindRepoRoot()
}
