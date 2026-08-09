package economy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── Load ─────────────────────────────────────────────────────────────────────

func TestLoad_NoFile(t *testing.T) {
	dir := t.TempDir()
	bs, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bs != nil {
		t.Error("expected nil when file missing")
	}
}

func TestLoad_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	econDir := filepath.Join(dir, ".ovav", "economy")
	os.MkdirAll(econDir, 0755)
	os.WriteFile(filepath.Join(econDir, "budget_status.json"), []byte("not json"), 0644)

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestLoad_Success(t *testing.T) {
	dir := t.TempDir()
	econDir := filepath.Join(dir, ".ovav", "economy")
	os.MkdirAll(econDir, 0755)
	data := `{"model":"deepseek-v4-pro","session_cost_usd":0.5,"alert_level":"green","session_turns":3}`
	os.WriteFile(filepath.Join(econDir, "budget_status.json"), []byte(data), 0644)

	bs, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bs == nil {
		t.Fatal("expected non-nil result")
	}
	if bs.Model != "deepseek-v4-pro" {
		t.Errorf("expected deepseek-v4-pro, got %s", bs.Model)
	}
	if bs.SessionCostUSD != 0.5 {
		t.Errorf("expected 0.5, got %f", bs.SessionCostUSD)
	}
}

func TestLoad_ReadError(t *testing.T) {
	dir := t.TempDir()
	econDir := filepath.Join(dir, ".ovav", "economy")
	os.MkdirAll(econDir, 0755)
	// Create a file we can't read
	path := filepath.Join(econDir, "budget_status.json")
	os.WriteFile(path, []byte("{}"), 0000)

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected read error")
	}
}

// ── ColoredAlert ─────────────────────────────────────────────────────────────

func TestColoredAlert_Green(t *testing.T) {
	result := ColoredAlert("green")
	if !strings.Contains(result, "green") {
		t.Error("result should contain 'green'")
	}
	if !strings.HasPrefix(result, "\033[") {
		t.Error("result should have ANSI prefix")
	}
}

func TestColoredAlert_Yellow(t *testing.T) {
	result := ColoredAlert("yellow")
	if !strings.Contains(result, "yellow") {
		t.Error("result should contain 'yellow'")
	}
}

func TestColoredAlert_Red(t *testing.T) {
	result := ColoredAlert("red")
	if !strings.Contains(result, "red") {
		t.Error("result should contain 'red'")
	}
}

func TestColoredAlert_Default(t *testing.T) {
	result := ColoredAlert("orange")
	if result != "orange" {
		t.Errorf("expected plain 'orange', got %q", result)
	}
}

// ── FormatHuman ──────────────────────────────────────────────────────────────

func TestFormatHuman_Nil(t *testing.T) {
	result := FormatHuman(nil)
	if result != "" {
		t.Errorf("expected empty string for nil, got %q", result)
	}
}

func TestFormatHuman_WithData(t *testing.T) {
	bs := &BudgetStatus{
		Model:               "deepseek-v4-pro",
		SessionCostUSD:      0.1234,
		SessionRemainingUSD: 9.8766,
		AlertLevel:          "green",
		SessionTurns:        5,
		SessionPct:          12.3,
		MonthlyPct:          2.1,
	}
	result := FormatHuman(bs)
	if !strings.Contains(result, "deepseek-v4-pro") {
		t.Error("result should contain model name")
	}
	if !strings.Contains(result, "0.1234") {
		t.Error("result should contain session cost")
	}
	if !strings.Contains(result, "5") {
		t.Error("result should contain turns")
	}
}

// ── defaultBudgetConfig ─────────────────────────────────────────────────────

func TestDefaultBudgetConfig(t *testing.T) {
	cfg := defaultBudgetConfig()
	if cfg.MonthlyBudgetUSD != 200.0 {
		t.Errorf("expected 200.0, got %f", cfg.MonthlyBudgetUSD)
	}
	if cfg.PerSessionLimitUSD != 10.0 {
		t.Errorf("expected 10.0, got %f", cfg.PerSessionLimitUSD)
	}
	if cfg.AutoStopThreshold != 1.0 {
		t.Errorf("expected 1.0, got %f", cfg.AutoStopThreshold)
	}
	if len(cfg.AlertThresholds) != 3 {
		t.Errorf("expected 3 thresholds, got %d", len(cfg.AlertThresholds))
	}
}

// ── loadBudgetConfig paths ──────────────────────────────────────────────────

func TestLoadBudgetConfig_NoFile(t *testing.T) {
	dir := t.TempDir()
	cfg := loadBudgetConfig(dir)
	if cfg.MonthlyBudgetUSD != 200.0 {
		t.Errorf("expected default 200.0, got %f", cfg.MonthlyBudgetUSD)
	}
	if cfg.PerSessionLimitUSD != 10.0 {
		t.Errorf("expected default 10.0, got %f", cfg.PerSessionLimitUSD)
	}
}

func TestLoadBudgetConfig_BadYAML(t *testing.T) {
	dir := t.TempDir()
	writeProviderPrices(t, dir, "not: [valid: yaml: {{}}}}")
	cfg := loadBudgetConfig(dir)
	if cfg.MonthlyBudgetUSD != 200.0 {
		t.Errorf("expected default fallback, got %f", cfg.MonthlyBudgetUSD)
	}
}

func TestLoadBudgetConfig_ZeroFields(t *testing.T) {
	dir := t.TempDir()
	writeProviderPrices(t, dir, `budget_defaults:
  monthly_budget_usd: 0
  per_session_limit_usd: 0
  alert_thresholds: []
  auto_stop_threshold: 0
`)
	cfg := loadBudgetConfig(dir)
	// Zero fields should be replaced with defaults
	if cfg.MonthlyBudgetUSD != 200.0 {
		t.Errorf("expected 200.0 fallback, got %f", cfg.MonthlyBudgetUSD)
	}
	if cfg.PerSessionLimitUSD != 10.0 {
		t.Errorf("expected 10.0 fallback, got %f", cfg.PerSessionLimitUSD)
	}
	if len(cfg.AlertThresholds) != 3 {
		t.Errorf("expected 3 default thresholds, got %d", len(cfg.AlertThresholds))
	}
}

// ── parseTimestamp edge cases ────────────────────────────────────────────────

func TestParseTimestamp_NoColonTimezone(t *testing.T) {
	// Format: "2006-01-02T15:04:05-0700"
	ts, err := parseTimestamp("2026-06-18T12:30:00-0500")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts.Hour() != 12 {
		t.Errorf("expected hour 12, got %d", ts.Hour())
	}
}

func TestParseTimestamp_RFC3339(t *testing.T) {
	ts, err := parseTimestamp("2026-06-18T12:30:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts.Hour() != 12 {
		t.Errorf("expected hour 12, got %d", ts.Hour())
	}
}

func TestParseTimestamp_Invalid(t *testing.T) {
	_, err := parseTimestamp("not-a-timestamp")
	if err == nil {
		t.Fatal("expected error for invalid timestamp")
	}
}

// ── readLedger permission error ─────────────────────────────────────────────

func TestReadLedger_PermissionError(t *testing.T) {
	dir := t.TempDir()
	econDir := filepath.Join(dir, ".ovav", "economy")
	os.MkdirAll(econDir, 0755)
	path := filepath.Join(econDir, "cost_ledger.jsonl")
	os.WriteFile(path, []byte("data"), 0000) // no read permissions

	_, err := readLedger(dir)
	if err == nil {
		t.Fatal("expected permission error")
	}
}

// ── GetSessionStats with cache savings ──────────────────────────────────────

func TestGetSessionStats_CacheSavings(t *testing.T) {
	dir := t.TempDir()
	e1 := nowEntry(0.010, 1)
	e1.CacheHit = true
	e1.CacheSavingsUSD = 0.005
	e2 := nowEntry(0.020, 2)
	e2.CacheHit = false
	writeLedger(t, dir, []LedgerEntry{e1, e2})

	stats, err := GetSessionStats(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.CacheSavingsUSD != 0.005 {
		t.Errorf("expected cache savings 0.005, got %f", stats.CacheSavingsUSD)
	}
	if stats.CacheHitRate != 0.5 {
		t.Errorf("expected cache hit rate 0.5, got %f", stats.CacheHitRate)
	}
}

// ── GetMonthlyStats empty ───────────────────────────────────────────────────

func TestGetMonthlyStats_Empty(t *testing.T) {
	dir := t.TempDir()
	stats, err := GetMonthlyStats(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Turns != 0 {
		t.Errorf("expected 0 turns, got %d", stats.Turns)
	}
	if stats.TotalCostUSD != 0 {
		t.Errorf("expected 0 cost, got %f", stats.TotalCostUSD)
	}
}

// ── GetBudgetConsumption edge cases ─────────────────────────────────────────

func TestGetBudgetConsumption_NoThresholds(t *testing.T) {
	dir := t.TempDir()
	writeProviderPrices(t, dir, `budget_defaults:
  monthly_budget_usd: 200.0
  per_session_limit_usd: 10.0
  alert_thresholds: []
  auto_stop_threshold: 1.0
`)
	consumption, err := GetBudgetConsumption(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// With no thresholds, alert should be green
	if consumption.AlertLevel != "green" {
		t.Errorf("expected green with no thresholds, got %s", consumption.AlertLevel)
	}
}

func TestGetBudgetConsumption_MonthlyHigher(t *testing.T) {
	dir := t.TempDir()
	writeProviderPrices(t, dir, `budget_defaults:
  monthly_budget_usd: 10.0
  per_session_limit_usd: 200.0
  alert_thresholds: [0.70, 0.85, 0.95]
  auto_stop_threshold: 1.0
`)
	// Consume 96% of monthly budget
	writeLedger(t, dir, []LedgerEntry{nowEntry(9.60, 1)})

	consumption, err := GetBudgetConsumption(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Monthly pct > session pct, so monthly drives the alert
	if consumption.AlertLevel != "red" {
		t.Errorf("expected red from monthly, got %s (monthlyPct: %.1f)", consumption.AlertLevel, consumption.MonthlyPct)
	}
}

func TestGetBudgetConsumption_OrangeAlert(t *testing.T) {
	dir := t.TempDir()
	writeProviderPrices(t, dir, `budget_defaults:
  monthly_budget_usd: 200.0
  per_session_limit_usd: 10.0
  alert_thresholds: [0.70, 0.85, 0.95]
  auto_stop_threshold: 1.0
`)
	// 88% of session budget
	writeLedger(t, dir, []LedgerEntry{nowEntry(8.80, 1)})

	consumption, err := GetBudgetConsumption(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if consumption.AlertLevel != "orange" {
		t.Errorf("expected orange at 88%%, got %s (pct: %.1f)", consumption.AlertLevel, consumption.SessionPct)
	}
}

// ── UpdateBudgetStatus edge cases ───────────────────────────────────────────

func TestUpdateBudgetStatus_EmptyModel(t *testing.T) {
	dir := t.TempDir()
	writeProviderPrices(t, dir, `budget_defaults:
  monthly_budget_usd: 200.0
  per_session_limit_usd: 10.0
  alert_thresholds: [0.70, 0.85, 0.95]
  auto_stop_threshold: 1.0
`)
	if err := UpdateBudgetStatus(dir, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := filepath.Join(dir, ".ovav", "economy", "budget_status.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file not written: %v", err)
	}

	var status BudgetStatus
	json.Unmarshal(data, &status)
	if status.Model != "unknown" {
		t.Errorf("expected 'unknown', got %s", status.Model)
	}
}

func TestUpdateBudgetStatus_FallbackOnReadError(t *testing.T) {
	dir := t.TempDir()
	// No provider_prices.yaml → loadBudgetConfig uses defaults, no ledger → stats are 0
	// This should still write a valid status file
	if err := UpdateBudgetStatus(dir, "test-model"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := filepath.Join(dir, ".ovav", "economy", "budget_status.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file not written: %v", err)
	}

	var status BudgetStatus
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if status.Model != "test-model" {
		t.Errorf("expected test-model, got %s", status.Model)
	}
}

func TestUpdateBudgetStatus_InvalidDir(t *testing.T) {
	// Write to a path that doesn't exist and can't be created (e.g., under /proc)
	err := UpdateBudgetStatus("/proc/fake", "model")
	if err == nil {
		t.Fatal("expected error for invalid dir")
	}
}

// ── BudgetConsumption negative remaining ────────────────────────────────────

func TestGetBudgetConsumption_NegativeRemaining(t *testing.T) {
	dir := t.TempDir()
	writeProviderPrices(t, dir, `budget_defaults:
  monthly_budget_usd: 0.1
  per_session_limit_usd: 0.1
  alert_thresholds: [0.70, 0.85, 0.95]
  auto_stop_threshold: 1.0
`)
	// Overspend
	writeLedger(t, dir, []LedgerEntry{nowEntry(1.0, 1)})

	consumption, err := GetBudgetConsumption(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Remaining should be clamped at 0
	if consumption.SessionRemainingUSD < 0 {
		t.Errorf("session remaining should be >= 0, got %f", consumption.SessionRemainingUSD)
	}
	if consumption.MonthlyRemainingUSD < 0 {
		t.Errorf("monthly remaining should be >= 0, got %f", consumption.MonthlyRemainingUSD)
	}
}
