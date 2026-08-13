// OVAV cPanel — Product Update Handlers
//
// GOV-007: Product version check, update trigger, and Cockpit notification endpoints.
// Routes:
//   GET  /api/v1/product/version  — current product version + available update
//   POST /api/v1/product/update   — trigger sync + broadcast update to Cockpit clients

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var runProductSyncCommand = func(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

// ProductVersionResponse is the API response for version queries.
type ProductVersionResponse struct {
	Current     string `json:"current"`
	Available   string `json:"available,omitempty"`
	UpdateReady bool   `json:"update_ready"`
	Channel     string `json:"channel"`
	CheckedAt   string `json:"checked_at"`
}

// ProductUpdateResponse is the API response for update triggers.
type ProductUpdateResponse struct {
	Status    string `json:"status"`
	FromVer   string `json:"from_version"`
	ToVer     string `json:"to_version,omitempty"`
	Message   string `json:"message"`
	Notified  int    `json:"clients_notified"`
	StartedAt string `json:"started_at"`
}

// ── GET /api/v1/product/version ────────────────────────────────────────────────

func handleProductVersion(w http.ResponseWriter, r *http.Request) {
	resp := ProductVersionResponse{
		Channel:   "stable",
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Read VERSION from OVAV root
	verPath := filepath.Join(ovavRoot(), "VERSION")
	if data, err := os.ReadFile(verPath); err == nil {
		resp.Current = strings.TrimSpace(string(data))
	}

	// Check git for latest tag (available version)
	cmd := exec.Command("git", "-C", ovavRoot(), "describe", "--tags", "--abbrev=0")
	if out, err := cmd.Output(); err == nil {
		latest := strings.TrimSpace(string(out))
		if latest != "" && latest != resp.Current {
			resp.Available = latest
			resp.UpdateReady = true
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// ── POST /api/v1/product/update ───────────────────────────────────────────────

func handleProductUpdate(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now().UTC().Format(time.RFC3339)
	fromVer := ""

	verPath := filepath.Join(ovavRoot(), "VERSION")
	if data, err := os.ReadFile(verPath); err == nil {
		fromVer = strings.TrimSpace(string(data))
	}

	// 1. Run sync to project updates to product surfaces
	syncErr := runProductSync()

	// 2. Determine target version
	toVer := ""
	cmd := exec.Command("git", "-C", ovavRoot(), "describe", "--tags", "--abbrev=0")
	if out, err := cmd.Output(); err == nil {
		toVer = strings.TrimSpace(string(out))
	}

	// 3. Broadcast update event to all connected Cockpit clients
	clientCount := ClientCount()
	PushEvent("product_update", map[string]interface{}{
		"type":         "update_available",
		"from_version": fromVer,
		"to_version":   toVer,
		"message":      "New OVAV Product version available",
		"timestamp":    startTime,
	})

	// 4. Response
	resp := ProductUpdateResponse{
		FromVer:   fromVer,
		ToVer:     toVer,
		Notified:  clientCount,
		StartedAt: startTime,
	}

	if syncErr != nil {
		resp.Status = "partial"
		resp.Message = fmt.Sprintf("Update dispatched but sync had issues: %v", syncErr)
		writeJSON(w, http.StatusMultiStatus, resp)
		return
	}

	resp.Status = "dispatched"
	resp.Message = fmt.Sprintf("Update from %s to %s dispatched. %d clients notified.", fromVer, toVer, clientCount)
	writeJSON(w, http.StatusOK, resp)
}

// runProductSync syncs agents and reinstalls OVAV Product.
func runProductSync() error {
	root := ovavRoot()
	goRuntime := filepath.Join(root, "go-runtime")

	// Step 1: Sync agents to runtimes/mimocode/agents/
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if out, err := runProductSyncCommand(ctx, goRuntime, "run", "./cmd/ovav", "sync", "--agents"); err != nil {
		return fmt.Errorf("sync agents: %v — %s", err, string(out))
	}

	// Step 2: Reinstall OVAV Product from runtimes/mimocode/agents/
	if out, err := runProductSyncCommand(ctx, goRuntime, "run", "./cmd/ovav", "product", "install"); err != nil {
		return fmt.Errorf("product install: %v — %s", err, string(out))
	}

	return nil
}

// ovavRoot returns the OVAV Systems root directory.
func ovavRoot() string {
	// Derive from binary location or env var
	if root := os.Getenv("OVAV_ROOT"); root != "" {
		return root
	}
	// Default: parent of go-runtime (where the cPanel binary is typically invoked)
	cwd, _ := os.Getwd()
	// Walk up to find go-runtime/ directory
	for range 5 {
		if _, err := os.Stat(filepath.Join(cwd, "go-runtime")); err == nil {
			return cwd
		}
		cwd = filepath.Dir(cwd)
	}
	return "."
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
