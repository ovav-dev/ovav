// OVAV cPanel v5.0 — Status & Health handlers.
//
// GET /health          — Basic health check
// GET /api/v1/status   — Full system status (git, economy, session, agents)

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ── Status cache ──────────────────────────────────────────────────────────────

var (
	statusCache     map[string]interface{}
	statusCacheLock sync.RWMutex
	statusCacheTime time.Time
	statusCacheTTL  = 15 * time.Second
)

func getCachedStatus() map[string]interface{} {
	statusCacheLock.RLock()
	if time.Since(statusCacheTime) < statusCacheTTL && statusCache != nil {
		defer statusCacheLock.RUnlock()
		return statusCache
	}
	statusCacheLock.RUnlock()

	statusCacheLock.Lock()
	defer statusCacheLock.Unlock()

	if time.Since(statusCacheTime) < statusCacheTTL && statusCache != nil {
		return statusCache
	}

	statusCache = collectStatus()
	statusCacheTime = time.Now()
	return statusCache
}

// ── Status collection ─────────────────────────────────────────────────────────

func collectStatus() map[string]interface{} {
	data := make(map[string]interface{})

	// Timestamp
	data["timestamp"] = time.Now().Format("2006-01-02 15:04:05")

	// Git info — use build-time injected vars as fallback when git unavailable
	branch := gitCmd("branch", "--show-current")
	if branch == "?" {
		branch = GitBranch
	}
	head := gitCmd("rev-parse", "--short", "HEAD")
	if head == "?" {
		head = GitSHA
	}
	data["git"] = map[string]interface{}{
		"branch":      branch,
		"head":        head,
		"commits":     gitCmd("rev-list", "--count", "HEAD"),
		"dirty":       gitDirty(),
		"last_commit": gitCmd("log", "--oneline", "-1"),
	}

	// System info
	data["system"] = map[string]interface{}{
		"agents": countAgents(),
	}

	// Economy
	data["economy"] = readEconomy()

	// Session
	data["session"] = readSession()

	return data
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// handleHealth responds to GET /health.
func handleHealth(w http.ResponseWriter, r *http.Request) {
	sendOK(w, map[string]string{
		"status":  "ok",
		"service": "ovav-cpanel",
		"runtime": "Go stdlib",
		"version": Version,
	})
}

// handleHealthz responds to GET /api/v1/healthz — deep public health check.
// Does NOT require Cloudflare Access auth. Checks internal subsystems.
// This is the endpoint to monitor from outside when Access might be down.
func handleHealthz(w http.ResponseWriter, r *http.Request) {
	type subCheck struct {
		Status string `json:"status"`
		Detail string `json:"detail,omitempty"`
	}
	checks := map[string]subCheck{
		"backend": {Status: "ok"},
		"vault":   {Status: "ok"},
		"auth":    {Status: "ok"},
		"jwt":     {Status: "ok"},
		"version": {Status: "ok", Detail: Version},
	}

	healthy := true

	// 1. JWT subsystem initialized?
	jwtKeyLock.RLock()
	key := jwtPrivateKey
	jwtKeyLock.RUnlock()
	if key == nil {
		checks["jwt"] = subCheck{Status: "FAIL", Detail: "jwtPrivateKey is nil — RSA key not loaded"}
		checks["auth"] = subCheck{Status: "DEGRADED", Detail: "JWT subsystem uninitialized"}
		healthy = false
	}

	// 2. Auth session map exists?
	if jwtSessions == nil {
		checks["auth"] = subCheck{Status: "FAIL", Detail: "jwtSessions map is nil"}
		healthy = false
	}

	// 3. Vault store accessible?
	store := getUserStore()
	if store == nil {
		checks["vault"] = subCheck{Status: "DEGRADED", Detail: "user store not initialized"}
	}

	// 4. Version set?
	if Version == "" || Version == "unknown" {
		checks["version"] = subCheck{Status: "DEGRADED", Detail: "version not set"}
	}

	code := http.StatusOK
	if !healthy {
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    map[bool]string{true: "ok", false: "degraded"}[healthy],
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"req_id":    ReqIDFrom(r.Context()),
		"checks":    checks,
	})
}

// handleStatus responds to GET /api/v1/status.
func handleStatus(w http.ResponseWriter, r *http.Request) {
	status := getCachedStatus()
	sendOK(w, status)
}

// ── Git helpers ───────────────────────────────────────────────────────────────

func gitCmd(args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = RepoRoot
	out, err := cmd.Output()
	if err != nil {
		return "?"
	}
	return strings.TrimSpace(string(out))
}

func gitDirty() string {
	cmd := exec.Command("git", "status", "--short")
	cmd.Dir = RepoRoot
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return "clean"
	}
	return "dirty"
}

// ── Agent count ───────────────────────────────────────────────────────────────

func countAgents() int {
	agentsDir := filepath.Join(RepoRoot, ".opencode", "agents")
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			count++
		}
	}
	return count
}

// ── Economy reader ────────────────────────────────────────────────────────────

func readEconomy() map[string]interface{} {
	def := map[string]interface{}{
		"session_cost": 0, "session_pct": 0,
		"monthly_cost": 0, "monthly_pct": 0,
		"model": "?",
	}

	dashPath := filepath.Join(RepoRoot, ".ovav", "economy", "dashboard.json")
	data, err := os.ReadFile(dashPath)
	if err != nil {
		return def
	}

	var dash map[string]interface{}
	if err := json.Unmarshal(data, &dash); err != nil {
		return def
	}

	if s, ok := dash["session"].(map[string]interface{}); ok {
		if v, ok := s["cost_usd"]; ok {
			def["session_cost"] = v
		}
		if v, ok := s["percent"]; ok {
			def["session_pct"] = v
		}
	}
	if m, ok := dash["monthly"].(map[string]interface{}); ok {
		if v, ok := m["cost_usd"]; ok {
			def["monthly_cost"] = v
		}
		if v, ok := m["percent"]; ok {
			def["monthly_pct"] = v
		}
	}
	if m, ok := dash["current_model"]; ok {
		def["model"] = m
	}

	return def
}

// ── Session reader ────────────────────────────────────────────────────────────

func readSession() map[string]interface{} {
	def := map[string]interface{}{
		"uptime":        "?",
		"canary_alarms": 0,
	}

	// Uptime from session marker
	markerPath := filepath.Join(RepoRoot, ".ovav", "runtime", ".session_marker")
	if data, err := os.ReadFile(markerPath); err == nil {
		ts := strings.TrimSpace(string(data))
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			delta := time.Since(t)
			h := int(delta.Hours())
			m := int(delta.Minutes()) % 60
			def["uptime"] = fmt.Sprintf("%dh %dm", h, m)
		}
	}

	// Canary alarms
	canaryPath := filepath.Join(RepoRoot, ".ovav", "security", "canary_state.json")
	if data, err := os.ReadFile(canaryPath); err == nil {
		var canary map[string]interface{}
		if json.Unmarshal(data, &canary) == nil {
			if v, ok := canary["alarm_count"]; ok {
				switch n := v.(type) {
				case float64:
					def["canary_alarms"] = int(n)
				case int:
					def["canary_alarms"] = n
				}
			}
		}
	}

	return def
}
