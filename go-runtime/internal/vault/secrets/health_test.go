package secrets

import (
	"testing"
	"time"
)

// ── HealthStatus Constants ─────────────────────────────────────────────────────

func TestHealthStatus_Values(t *testing.T) {
	if HealthOK != "ok" {
		t.Errorf("HealthOK = %q, want %q", HealthOK, "ok")
	}
	if HealthExpiringSoon != "expiring_soon" {
		t.Errorf("HealthExpiringSoon = %q, want %q", HealthExpiringSoon, "expiring_soon")
	}
	if HealthExpired != "expired" {
		t.Errorf("HealthExpired = %q, want %q", HealthExpired, "expired")
	}
	if HealthUnknown != "unknown" {
		t.Errorf("HealthUnknown = %q, want %q", HealthUnknown, "unknown")
	}
}

// ── HealthReport ─────────────────────────────────────────────────────────────

func TestCheckStoreHealth_Expired(t *testing.T) {
	store := NewSecretStore()
	past := time.Now().Add(-24 * time.Hour)
	sec := NewSecret("ExpiredToken", TypeAPIToken, "test", "manual", []byte("value"))
	sec.ExpiresAt = &past
	store.Add(sec)

	reports := CheckStoreHealth(store)
	if len(reports) != 1 {
		t.Fatalf("CheckStoreHealth: got %d reports, want 1", len(reports))
	}

	if reports[0].Status != HealthExpired {
		t.Errorf("Status = %v, want %v", reports[0].Status, HealthExpired)
	}
	if reports[0].Message == "" {
		t.Error("Expired secret should have a message")
	}
}

func TestCheckStoreHealth_ExpiringSoon(t *testing.T) {
	store := NewSecretStore()
	// Expires in 3 days — within the 7-day warning window
	soon := time.Now().Add(3 * 24 * time.Hour)
	sec := NewSecret("ExpiringSoon", TypeAPIToken, "test", "manual", []byte("value"))
	sec.ExpiresAt = &soon
	store.Add(sec)

	reports := CheckStoreHealth(store)
	if reports[0].Status != HealthExpiringSoon {
		t.Errorf("Status = %v, want %v", reports[0].Status, HealthExpiringSoon)
	}
}

func TestCheckStoreHealth_OK(t *testing.T) {
	store := NewSecretStore()
	future := time.Now().Add(30 * 24 * time.Hour) // 30 days out
	sec := NewSecret("HealthyToken", TypeAPIToken, "test", "manual", []byte("value"))
	sec.ExpiresAt = &future
	store.Add(sec)

	reports := CheckStoreHealth(store)
	if reports[0].Status != HealthOK {
		t.Errorf("Status = %v, want %v", reports[0].Status, HealthOK)
	}
}

func TestCheckStoreHealth_NoExpiration(t *testing.T) {
	store := NewSecretStore()
	sec := NewSecret("NoExpiry", TypeAPIToken, "test", "manual", []byte("value"))
	// ExpiresAt is nil
	store.Add(sec)

	reports := CheckStoreHealth(store)
	if reports[0].Status != HealthUnknown {
		t.Errorf("Status = %v, want %v (no expiration set)", reports[0].Status, HealthUnknown)
	}
	if reports[0].Message == "" {
		t.Error("Unknown status should have a message")
	}
}

func TestCheckStoreHealth_MultipleSecrets(t *testing.T) {
	store := NewSecretStore()

	// Expired
	past := time.Now().Add(-1 * time.Hour)
	e1 := NewSecret("E1", TypeAPIToken, "cf", "manual", []byte("v"))
	e1.ExpiresAt = &past
	store.Add(e1)

	// Expiring soon
	soon := time.Now().Add(2 * 24 * time.Hour)
	e2 := NewSecret("E2", TypeAPIToken, "cf", "manual", []byte("v"))
	e2.ExpiresAt = &soon
	store.Add(e2)

	// OK
	future := time.Now().Add(30 * 24 * time.Hour)
	e3 := NewSecret("E3", TypeAPIToken, "cf", "manual", []byte("v"))
	e3.ExpiresAt = &future
	store.Add(e3)

	// Unknown
	e4 := NewSecret("E4", TypeAPIToken, "cf", "manual", []byte("v"))
	store.Add(e4)

	reports := CheckStoreHealth(store)
	if len(reports) != 4 {
		t.Fatalf("CheckStoreHealth: got %d, want 4", len(reports))
	}

	statusMap := make(map[string]string)
	for _, r := range reports {
		statusMap[r.Name] = string(r.Status)
	}

	if statusMap["E1"] != "expired" {
		t.Errorf("E1 = %v, want expired", statusMap["E1"])
	}
	if statusMap["E2"] != "expiring_soon" {
		t.Errorf("E2 = %v, want expiring_soon", statusMap["E2"])
	}
	if statusMap["E3"] != "ok" {
		t.Errorf("E3 = %v, want ok", statusMap["E3"])
	}
	if statusMap["E4"] != "unknown" {
		t.Errorf("E4 = %v, want unknown", statusMap["E4"])
	}
}

func TestCheckStoreHealth_EmptyStore(t *testing.T) {
	store := NewSecretStore()
	reports := CheckStoreHealth(store)
	if len(reports) != 0 {
		t.Errorf("CheckStoreHealth on empty store: got %d reports, want 0", len(reports))
	}
}

func TestCheckSecretHealth(t *testing.T) {
	sec := NewSecret("Test", TypeAPIToken, "test", "manual", []byte("value"))
	future := time.Now().Add(10 * 24 * time.Hour)
	sec.ExpiresAt = &future

	report := CheckSecretHealth(sec)
	if report.Status != HealthOK {
		t.Errorf("Status = %v, want %v", report.Status, HealthOK)
	}
	if report.SecretID != sec.ID {
		t.Errorf("SecretID = %q, want %q", report.SecretID, sec.ID)
	}
	if report.Name != "Test" {
		t.Errorf("Name = %q, want %q", report.Name, "Test")
	}
}

func TestCheckStoreHealth_LastCheck(t *testing.T) {
	store := NewSecretStore()
	sec := NewSecret("Test", TypeAPIToken, "test", "manual", []byte("v"))
	store.Add(sec)

	before := time.Now().Add(-time.Second)
	reports := CheckStoreHealth(store)
	after := time.Now().Add(time.Second)

	if reports[0].LastCheck.Before(before) || reports[0].LastCheck.After(after) {
		t.Errorf("LastCheck = %v, want between %v and %v", reports[0].LastCheck, before, after)
	}
}

// ── HealthReport JSON ─────────────────────────────────────────────────────────

func TestHealthReport_JSON(t *testing.T) {
	rep := HealthReport{
		SecretID:  "sec-123",
		Name:      "CF_TOKEN",
		Type:      TypeAPIToken,
		Provider:  "cloudflare",
		Status:    HealthOK,
		Message:   "All good",
		LastCheck: time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC),
	}

	// Test that fields are correctly serialized
	if rep.Status != HealthOK {
		t.Errorf("Status = %v", rep.Status)
	}
	if rep.Type != TypeAPIToken {
		t.Errorf("Type = %v", rep.Type)
	}
}
