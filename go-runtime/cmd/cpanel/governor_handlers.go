// OVAV cPanel v58.0 — Governor API handlers.
//
// GET  /api/v1/governor/health      — Unified health scores
// GET  /api/v1/governor/decisions   — Active governance decisions
// GET  /api/v1/governor/tasks       — Delegation task queue
// POST /api/v1/governor/tasks       — Dispatch tasks from decisions
// GET  /api/v1/governor/counts      — Quick counts: alerts, delegations, decisions
// POST /api/v1/governor/trust       — Trust verification
//
// Wires internal/governor/ into the cPanel API.
// OVAV Signature: cmd/cpanel/governor_handlers.go — stabilized 2026-08-02

package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ovav/ovav/internal/governor"
)

// ── Governor health ──────────────────────────────────────────────────────────

func handleGovernorHealth(w http.ResponseWriter, r *http.Request) {
	p, f, t, _ := governor.QuickIntegrityMesh()
	integrity := governor.IntegrityMeshHealth(p, f, t)

	o, wC, c, _ := governor.QuickSelfDiagnosis() // wC to avoid shadowing http.ResponseWriter w
	sd := governor.SelfDiagnosisHealth(o, wC, c)

	avg, max, events, escalation := governor.QuickPainScorer()
	pain := governor.PainScorerHealth(avg, max, events, escalation)

	health := governor.ComputeUnifiedHealth(integrity, sd, pain)

	sendOK(w, map[string]interface{}{
		"status": "ok",
		"health": health,
	})
}

// ── Governor decisions ───────────────────────────────────────────────────────

func handleGovernorDecisions(w http.ResponseWriter, r *http.Request) {
	state := governor.GatherSystemState()
	decisions := governor.Decide(state)

	response := map[string]interface{}{
		"status":      "ok",
		"decisions":   decisions,
		"count":       len(decisions),
		"critical":    governor.HasCriticalDecisions(decisions),
		"by_priority": governor.CountByPriority(decisions),
	}

	statusCode := http.StatusOK
	if governor.HasCriticalDecisions(decisions) {
		statusCode = http.StatusMultiStatus // 207 — attention required
	}
	sendJSON(w, response, statusCode)
}

// ── Governor tasks (delegation queue) ────────────────────────────────────────

var globalTaskQueue = governor.NewTaskQueue()

func handleGovernorTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tasks := globalTaskQueue.GetAll()
		counts := globalTaskQueue.CountByStatus()
		sendOK(w, map[string]interface{}{
			"status": "ok",
			"tasks":  tasks,
			"counts": counts,
			"total":  globalTaskQueue.Count(),
		})

	case http.MethodPost:
		state := governor.GatherSystemState()
		decisions := governor.Decide(state)
		tasks := globalTaskQueue.DispatchFromDecisions(decisions)
		syncDelegateQueue() // persist queue state for CLI bridge queries

		sendJSON(w, map[string]interface{}{
			"status":        "ok",
			"dispatched":    len(tasks),
			"tasks":         tasks,
			"total_pending": globalTaskQueue.Count(),
		}, http.StatusCreated)

	default:
		sendError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// ── Governor trust verification ──────────────────────────────────────────────

func handleGovernorTrust(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		LeadName string   `json:"lead_name"`
		Claims   []string `json:"claims"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		sendError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	result := governor.VerifyTrust(governor.TrustInput{
		LeadName:     input.LeadName,
		OutputClaims: input.Claims,
	})

	statusCode := http.StatusOK
	if !result.Passed {
		statusCode = http.StatusUnprocessableEntity
	}

	sendJSON(w, map[string]interface{}{
		"status": "ok",
		"result": result,
	}, statusCode)
}

// ── Governor quick counts (bridged from CLI) ─────────────────────────────────

// delegateQueueState mirrors the type in bridge.go for JSON serialization.
// OVAV: delegation queue persistence — written by cPanel, read by CLI bridge.
type delegateQueueState struct {
	Total   int `json:"total"`
	Pending int `json:"pending"`
}

// syncDelegateQueue writes the current task queue state to disk so that
// the CLI bridge function CountPendingDelegationsQuick can read it.
// This decouples the in-memory cPanel task queue from the read-only
// CLI context that runs in a different process.
func syncDelegateQueue() {
	path := filepath.Join(RepoRoot, ".ovav", "runtime", "delegate_queue.json")
	state := delegateQueueState{
		Total:   globalTaskQueue.Count(),
		Pending: len(globalTaskQueue.GetByStatus(governor.TaskPending)),
	}
	data, err := json.Marshal(state)
	if err != nil {
		return
	}
	// #nosec G304 — path is constructed from RepoRoot constant, not user input
	os.WriteFile(path, data, 0644) //nolint:errcheck // best-effort sync
}

// handleGovernorCounts returns lightweight counts for dashboard banners.
// Exposes: active alerts, pending delegations, outstanding decisions.
//
// GET /api/v1/governor/counts
//
// OVAV: real-time subsystem counts — fused with alerts, delegate, decision engines.
func handleGovernorCounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Sync before reading so we get latest delegation count
	syncDelegateQueue()

	alerts := governor.CountActiveAlertsQuick()
	delegations := governor.CountPendingDelegationsQuick()
	decisions := governor.CountOutstandingDecisionsQuick()

	sendOK(w, map[string]interface{}{
		"status":       "ok",
		"alerts":       alerts,
		"delegations":  delegations,
		"decisions":    decisions,
		"has_alerts":   alerts > 0,
		"has_delays":   delegations > 0,
		"needs_action": decisions > 0,
	})
}
