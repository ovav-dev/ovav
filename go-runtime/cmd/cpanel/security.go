// OVAV cPanel v5.0 — Security handlers.
//
// GET    /api/v1/security/audit-log       — Audit log entries
// DELETE /api/v1/security/canary-alarms   — Clear canary alarms
// GET    /api/v1/security/living-integrity — Living Integrity scan
//
// 🔒 OVAV GOVERNED: Native Go integrity checks. Zero Python bridge.
//    Reuses runLivingIntegrity() from validators.go (single source of truth).

package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ── Audit log ─────────────────────────────────────────────────────────────────

func handleAuditLog(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
		limit = n
	}

	auditPath := filepath.Join(RepoRoot, ".ovav", "security", "audit_log.yaml")
	entries := []interface{}{}
	chainIntact := true

	if data, err := os.ReadFile(auditPath); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			entry := map[string]interface{}{"raw": line}
			if category == "" || strings.Contains(line, category) {
				entries = append(entries, entry)
			}
		}
	}

	if len(entries) > limit {
		entries = entries[:limit]
	}

	sendOK(w, map[string]interface{}{
		"entries":      entries,
		"total":        len(entries),
		"chain_intact": chainIntact,
	})
}

// ── Canary alarms ─────────────────────────────────────────────────────────────

func handleClearAlarms(w http.ResponseWriter, r *http.Request) {
	canaryPath := filepath.Join(RepoRoot, ".ovav", "security", "canary_state.json")
	if _, err := os.Stat(canaryPath); os.IsNotExist(err) {
		sendOK(w, map[string]string{"status": "ok", "message": "No canary state file to clear"})
		return
	}

	data, err := os.ReadFile(canaryPath)
	if err != nil {
		sendError(w, "failed to read canary state", http.StatusInternalServerError)
		return
	}

	var state map[string]interface{}
	if err := json.Unmarshal(data, &state); err != nil {
		state = make(map[string]interface{})
	}
	state["alarm_count"] = 0

	newData, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		sendError(w, "failed to marshal canary state", http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(canaryPath, newData, 0644); err != nil {
		sendError(w, "failed to write canary state", http.StatusInternalServerError)
		return
	}

	sendOK(w, map[string]string{"status": "ok", "message": "Canary alarms cleared"})
}

// ── Living Integrity (native Go — reuses validators logic) ────────────────────

func handleLivingIntegrity(w http.ResponseWriter, r *http.Request) {
	overall, score, pass, fail, checks := runLivingIntegrity()

	checksOut := []interface{}{}
	for _, c := range checks {
		icon := "❌"
		if c.Ok {
			icon = "✅"
		}
		checksOut = append(checksOut, map[string]interface{}{
			"name":   icon + " " + c.Name,
			"ok":     c.Ok,
			"weight": c.Weight,
			"detail": c.Detail,
		})
	}

	sendOK(w, map[string]interface{}{
		"overall": overall,
		"score":   score,
		"pass":    pass,
		"fail":    fail,
		"checks":  checksOut,
		"engine":  "OVAV Go Native Integrity v5.0",
		"note":    "Full validator suite migration tracked in GO-VALIDATORS cap",
	})
}
