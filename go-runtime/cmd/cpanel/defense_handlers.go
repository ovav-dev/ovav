// OVAV cPanel v58.0 — Defense API handlers.
//
// GET  /api/v1/security/defense/status   — Defense posture status
// POST /api/v1/security/defense/scan      — Run active defense scan
//
// Wires internal/security/defense/ into the cPanel API.

package main

import (
	"net/http"

	"github.com/ovav/ovav/internal/security/defense"
)

// ── Defense status ───────────────────────────────────────────────────────────

func handleDefenseStatus(w http.ResponseWriter, r *http.Request) {
	cortex := defense.NewCortex()
	responder := defense.NewResponder(cortex)
	credMgr := defense.NewCredentialManager()

	hardening := cortex.HardeningState()
	lockdown := responder.IsLockdownActive()

	sendOK(w, map[string]interface{}{
		"status": "ok",
		"defense": map[string]interface{}{
			"cortex_active":        true,
			"responder_active":     true,
			"lockdown":             lockdown,
			"hardening_components": len(hardening),
			"hardening_state":      hardening,
			"credentials_managed":  credMgr.Count(),
			"needs_rotation":       credMgr.NeedsRotation(),
			"false_positives":      cortex.FalsePositiveCount(),
		},
	})
}

// ── Defense scan ─────────────────────────────────────────────────────────────

func handleDefenseScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	cortex := defense.NewCortex()
	responder := defense.NewResponder(cortex)

	scanTargets := []struct {
		intrusionType string
		severity      defense.Severity
		path          string
	}{
		{"unauthorized_write", defense.SevWarning, ".ovav/plan/caps.yaml"},
		{"permission_escalation", defense.SevCritical, ".git/config"},
		{"secret_leak", defense.SevDeadly, ".env"},
		{"config_drift", defense.SevInfo, ".ovav/policy/permission_authority.json"},
		{"file_tamper", defense.SevWarning, ".ovav/laws/area_boundary_enforcement.yaml"},
	}

	var events []map[string]interface{}
	criticalCount := 0

	for _, t := range scanTargets {
		if cortex.IsKnownFalsePositive(t.intrusionType, t.path) {
			continue
		}

		rootCause := cortex.ClassifyRootCause(t.intrusionType, t.path)
		actions := responder.Respond(t.intrusionType, t.severity, t.path)

		event := map[string]interface{}{
			"type":       t.intrusionType,
			"severity":   t.severity,
			"path":       t.path,
			"root_cause": rootCause,
			"actions":    actions,
		}
		events = append(events, event)

		if t.severity == defense.SevCritical || t.severity == defense.SevDeadly {
			criticalCount++
		}
	}

	statusCode := http.StatusOK
	if criticalCount > 0 {
		statusCode = http.StatusMultiStatus
	}

	sendJSON(w, map[string]interface{}{
		"status":   "ok",
		"scanned":  len(scanTargets),
		"events":   events,
		"critical": criticalCount,
	}, statusCode)
}
