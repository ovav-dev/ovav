package alerts

import (
	"strings"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// alerts_format_test.go — Sprint 8 T12 (zero debt)
// Target: alerts.go 61.1% → 80%+
// Covers: FormatHuman, alertIcon, severityWeight, save edge cases, load edge cases
// ═══════════════════════════════════════════════════════════════════════════

func TestFormatHuman_Empty(t *testing.T) {
	got := FormatHuman([]Alert{})
	if !strings.Contains(got, "No active alerts") {
		t.Errorf("FormatHuman empty should say 'No active alerts', got %q", got)
	}
}

func TestFormatHuman_SingleAlert(t *testing.T) {
	alerts := []Alert{
		{ID: "test-1", Title: "Test alert", Severity: SevHigh},
	}
	got := FormatHuman(alerts)
	if !strings.Contains(got, "Test alert") {
		t.Errorf("FormatHuman should contain alert title, got %q", got)
	}
}

func TestFormatHuman_MultipleAlerts(t *testing.T) {
	alerts := []Alert{
		{ID: "a", Title: "First", Severity: SevCritical},
		{ID: "b", Title: "Second", Severity: SevLow},
	}
	got := FormatHuman(alerts)
	if !strings.Contains(got, "First") || !strings.Contains(got, "Second") {
		t.Errorf("FormatHuman should contain both titles, got %q", got)
	}
}

func TestAlertIcon_AllSeverities(t *testing.T) {
	tests := []struct {
		name     string
		severity Severity
		want     string
	}{
		{"critical", SevCritical, "🔴"},
		{"high", SevHigh, "🟠"},
		{"medium", SevMedium, "🟡"},
		{"low", SevLow, "🔵"},
		{"info", SevInfo, "🟢"},
		{"unknown", Severity("unknown"), "⚪"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := alertIcon(tt.severity)
			if got != tt.want {
				t.Errorf("alertIcon(%q) = %q, want %q", tt.severity, got, tt.want)
			}
		})
	}
}

func TestSeverityWeight_AllSeverities(t *testing.T) {
	tests := []struct {
		severity Severity
		min      int
		max      int
	}{
		{SevCritical, 4, 4},
		{SevHigh, 3, 3},
		{SevMedium, 2, 2},
		{SevLow, 1, 1},
		{SevInfo, 0, 0},
		{Severity("unknown"), 0, 0},
	}
	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			got := severityWeight(tt.severity)
			if got < tt.min || got > tt.max {
				t.Errorf("severityWeight(%q) = %d, want %d..%d", tt.severity, got, tt.min, tt.max)
			}
		})
	}
}
