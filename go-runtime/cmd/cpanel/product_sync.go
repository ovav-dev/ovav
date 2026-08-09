// OVAV cPanel — Product Sync Handlers (GOV-009)
//
// Protected endpoints for managing the sync queue between
// OVAV Systems and OVAV Product. Auth-protected + rate-limited.

package main

import (
	"encoding/json"
	"net/http"
	"time"

	ovavsync "github.com/ovav/ovav/internal/sync"
)

// syncProtect wraps a sync handler with rate limiting.
func syncProtect(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if !checkRateLimit(ip) {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{
				"error": "rate limit exceeded — slow down",
			})
			return
		}
		handler(w, r)
	}
}

// ── GET /api/v1/product/sync/status ─────────────────────────────────
//
// Returns the current sync queue: what's staged, pending, and synced.
// Protected: requires valid session token via token auth.

func handleProductSyncStatus(w http.ResponseWriter, r *http.Request) {
	root := ovavRoot()

	manifest, err := ovavsync.DetectChanges(root)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "sync detection failed: " + err.Error(),
		})
		return
	}

	// Also include queue status
	queue, _ := ovavsync.GetQueueStatus(root)

	type syncStatusResponse struct {
		Detected  *ovavsync.SyncManifest `json:"detected"`
		Queue     *ovavsync.SyncManifest `json:"queue"`
		CheckedAt string                 `json:"checked_at"`
		CPanelURL string                 `json:"cpanel_url"`
	}

	writeJSON(w, http.StatusOK, syncStatusResponse{
		Detected:  manifest,
		Queue:     queue,
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
		CPanelURL: "d678beea.ovav.dev",
	})
}

// ── POST /api/v1/product/sync/stage ─────────────────────────────────
//
// Stages selected items for product distribution. Accepts JSON body
// with item IDs to stage.
//
// Body: {"items": ["id1", "id2", ...]}

func handleProductSyncStage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}

	var req struct {
		Items []string `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON: " + err.Error()})
		return
	}

	root := ovavRoot()
	result, err := ovavsync.StageChanges(root, req.Items)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Push SSE event notifying connected Cockpit clients
	PushEvent("sync:staged", map[string]interface{}{
		"items_staged": result.ItemsStaged,
		"errors":       result.Errors,
		"manifest":     result.Manifest,
	})

	writeJSON(w, http.StatusOK, result)
}

// ── POST /api/v1/product/sync/push ──────────────────────────────────
//
// Pushes staged changes to the product distribution queue.
// This creates a queue file that OVAV Product polls.

func handleProductSyncPush(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}

	root := ovavRoot()

	// QueueForProduct auto-stages all pending items
	result, err := ovavsync.QueueForProduct(root)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Mark as synced
	if result.ItemsStaged > 0 {
		ovavsync.ApplySync(root)
	}

	// Broadcast to all connected Cockpit clients
	PushEvent("sync:pushed", map[string]interface{}{
		"items_pushed": result.ItemsStaged,
		"manifest":     result.Manifest,
		"timestamp":    result.CompletedAt.Format(time.RFC3339),
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":       "pushed",
		"items_pushed": result.ItemsStaged,
		"manifest":     result.Manifest,
		"message":      "Sync package queued for OVAV Product distribution",
	})
}

// ── POST /api/v1/product/sync/apply ─────────────────────────────────
//
// Applies a sync — marks staged items as synced. Called by cPanel
// after Product confirms receipt.

func handleProductSyncApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}

	root := ovavRoot()
	result, err := ovavsync.ApplySync(root)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	PushEvent("sync:applied", map[string]interface{}{
		"items_synced": result.ItemsSynced,
		"errors":       result.Errors,
	})

	writeJSON(w, http.StatusOK, result)
}
