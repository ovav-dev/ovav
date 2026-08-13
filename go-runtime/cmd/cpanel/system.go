// OVAV cPanel v5.0 — System & Economy handlers.
//
// GET /api/v1/system/health     — Self-diagnosis (native Go)
// GET /api/v1/system/registry/   — List/fetch registries
// GET /api/v1/system/config      — Config overview (native Go)
// GET /api/v1/system/sbom        — SBOM summary
// GET /api/v1/system/kc          — Knowledge Compiler status
// GET /api/v1/system/operations  — Operations status
// GET /api/v1/economy            — Economy detail
//
// 🔒 OVAV GOVERNED: Zero Python bridge. All logic in native Go.
//    Python exec removed per GO-MIGRATION strategy (2026-06-15).

package main

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/sbom"
)

// ── System health (native Go) ─────────────────────────────────────────────────

type healthCheck struct {
	Name   string   `json:"name"`
	Status string   `json:"status"`
	Issues []string `json:"issues"`
}

func handleSystemHealth(w http.ResponseWriter, r *http.Request) {
	checks := []healthCheck{}

	// ── Identity layers ──────────────────────────────────────────────────────
	identityOk := true
	identityIssues := []string{}
	agentFiles := []string{
		".opencode/agents/lead-thavren.md",
		".opencode/agents/lead-eidren.md",
	}
	for _, f := range agentFiles {
		if _, err := os.Stat(filepath.Join(RepoRoot, f)); os.IsNotExist(err) {
			identityOk = false
			identityIssues = append(identityIssues, "Missing: "+f)
		}
	}
	status := "pass"
	if !identityOk {
		status = "fail"
	}
	checks = append(checks, healthCheck{Name: "identity_layers", Status: status, Issues: identityIssues})

	// ── Git state ────────────────────────────────────────────────────────────
	gitOk := true
	gitIssues := []string{}
	gitHead := filepath.Join(RepoRoot, ".git", "HEAD")
	if _, err := os.Stat(gitHead); os.IsNotExist(err) {
		gitOk = false
		gitIssues = append(gitIssues, "Git HEAD missing")
	}
	// Check if repo root is valid
	if _, err := os.Stat(filepath.Join(RepoRoot, ".git")); os.IsNotExist(err) {
		gitOk = false
		gitIssues = append(gitIssues, ".git directory missing — not a git repo")
	}
	gitStatus := "pass"
	if !gitOk {
		gitStatus = "fail"
	}
	checks = append(checks, healthCheck{Name: "git_state", Status: gitStatus, Issues: gitIssues})

	// ── Go runtime health ────────────────────────────────────────────────────
	goOk := true
	goIssues := []string{}
	goMod := filepath.Join(RepoRoot, "go-runtime", "go.mod")
	if _, err := os.Stat(goMod); os.IsNotExist(err) {
		goOk = false
		goIssues = append(goIssues, "go-runtime/go.mod missing")
	}
	goMain := filepath.Join(RepoRoot, "go-runtime", "cmd", "cpanel", "main.go")
	if _, err := os.Stat(goMain); os.IsNotExist(err) {
		goOk = false
		goIssues = append(goIssues, "cpanel main.go missing")
	}
	goStatus := "pass"
	if !goOk {
		goStatus = "fail"
	}
	checks = append(checks, healthCheck{Name: "go_runtime", Status: goStatus, Issues: goIssues})

	// ── Semantic drift ───────────────────────────────────────────────────────
	driftOk := true
	driftIssues := []string{}
	// Check for host config drift
	homeOpenCode := filepath.Join(os.Getenv("HOME"), ".config", "opencode", "opencode.jsonc")
	if _, err := os.Stat(homeOpenCode); err == nil {
		driftOk = false
		driftIssues = append(driftIssues, "External opencode.jsonc detected in ~/.config/opencode/ — may override OVAV config")
	}
	driftStatus := "pass"
	if !driftOk {
		driftStatus = "fail"
	}
	checks = append(checks, healthCheck{Name: "semantic_drift", Status: driftStatus, Issues: driftIssues})

	// ── Overall status ───────────────────────────────────────────────────────
	overallStatus := "pass"
	for _, c := range checks {
		if c.Status == "fail" {
			overallStatus = "fail"
			break
		}
	}

	sendOK(w, map[string]interface{}{
		"schema":       "ovav.self_diagnosis.v1",
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"status":       overallStatus,
		"checks":       checks,
		"engine":       "OVAV Go Native Health v5.0",
		"note":         "Zero Python bridge. Full self_diagnosis port tracked in GO-SYSTEM cap.",
	})
}

// ── Registry ──────────────────────────────────────────────────────────────────

func handleRegistry(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/system/registry")
	path = strings.TrimPrefix(path, "/")

	regDir := filepath.Join(RepoRoot, ".ovav", "registry")

	if path == "" || path == "registry" {
		entries, err := os.ReadDir(regDir)
		if err != nil {
			sendOK(w, map[string]interface{}{"registries": []string{}})
			return
		}
		files := []string{}
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".yaml") || strings.HasSuffix(e.Name(), ".json") {
				files = append(files, e.Name())
			}
		}
		sort.Strings(files)
		sendOK(w, map[string]interface{}{"registries": files})
		return
	}

	var regPath string
	candidates := []string{
		filepath.Join(regDir, path),
		filepath.Join(regDir, path+".yaml"),
		filepath.Join(regDir, path+".json"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			regPath = p
			break
		}
	}

	if regPath == "" {
		sendError(w, "registry '"+path+"' not found", http.StatusNotFound)
		return
	}

	data, err := os.ReadFile(regPath)
	if err != nil {
		sendError(w, "failed to read registry", http.StatusInternalServerError)
		return
	}

	var parsed interface{}
	if strings.HasSuffix(regPath, ".json") {
		json.Unmarshal(data, &parsed)
	} else {
		parsed = string(data)
	}

	sendOK(w, map[string]interface{}{
		"file": filepath.Base(regPath),
		"data": parsed,
	})
}

// ── Config (native Go) ────────────────────────────────────────────────────────

func handleConfig(w http.ResponseWriter, r *http.Request) {
	config := make(map[string]interface{})

	// opencode.json
	ocPath := filepath.Join(RepoRoot, "opencode.json")
	if data, err := os.ReadFile(ocPath); err == nil {
		var oc interface{}
		if json.Unmarshal(data, &oc) == nil {
			config["opencode"] = oc
		}
	}

	// Host drift check (native Go — no Python bridge)
	driftClean := true
	driftOutput := "Host config clean — no external interference detected."
	driftIssues := []string{}

	// Check for external opencode configs that could override OVAV
	homeConfigs := []string{
		filepath.Join(os.Getenv("HOME"), ".config", "opencode", "opencode.jsonc"),
		filepath.Join(os.Getenv("HOME"), ".config", "opencode", "AGENTS.md"),
		filepath.Join(os.Getenv("HOME"), ".config", "opencode", "agents"),
	}
	for _, p := range homeConfigs {
		if info, err := os.Stat(p); err == nil {
			driftClean = false
			if info.IsDir() {
				driftIssues = append(driftIssues, "External directory: "+p)
			} else {
				driftIssues = append(driftIssues, "External file: "+p)
			}
		}
	}

	if !driftClean {
		driftOutput = "EXTERNAL INTERFERENCE DETECTED:\n" + strings.Join(driftIssues, "\n") +
			"\n\nTo restore OVAV, remove these files and run: go run go-runtime/internal/validators/host_config_drift.go"
	}

	config["host_drift"] = map[string]interface{}{
		"clean":  driftClean,
		"output": driftOutput,
		"issues": driftIssues,
		"engine": "OVAV Go Native Drift Check v5.0",
	}

	sendOK(w, config)
}

// ── SBOM ──────────────────────────────────────────────────────────────────────

func handleSBOM(w http.ResponseWriter, r *http.Request) {
	// Try Go-native SBOM first (sbom.json)
	s, err := sbom.Load(RepoRoot)
	if err != nil {
		// Fallback: try legacy Python sbom.yaml
		sbomPath := filepath.Join(RepoRoot, ".ovav", "registry", "sbom.yaml")
		if data, readErr := os.ReadFile(sbomPath); readErr == nil {
			sendOK(w, map[string]interface{}{
				"source":       "legacy-python",
				"dependencies": 0,
				"core_files":   0,
				"data":         string(data),
			})
			return
		}
		sendOK(w, map[string]interface{}{
			"source":    "go-native",
			"generated": false,
			"action":    "Run 'ovav sbom generate' to create SBOM baseline.",
		})
		return
	}

	goDeps := make([]map[string]string, 0, len(s.Dependencies.Go))
	for _, d := range s.Dependencies.Go {
		goDeps = append(goDeps, map[string]string{
			"name":    d.Name,
			"version": d.Version,
		})
	}

	sendOK(w, map[string]interface{}{
		"source":       "go-native",
		"schema":       s.SchemaVersion,
		"generated_at": s.GeneratedAt,
		"git_identity": s.Metadata.GitIdentity,
		"dependencies": map[string]interface{}{
			"go_count":     len(s.Dependencies.Go),
			"python_count": len(s.Dependencies.Python),
			"go":           goDeps,
		},
		"core_files":     len(s.CoreFiles),
		"hash_algorithm": s.HashAlgorithm,
	})
}

// ── Knowledge Compiler ────────────────────────────────────────────────────────

func handleKCStatus(w http.ResponseWriter, r *http.Request) {
	kcPath := filepath.Join(RepoRoot, ".ovav", "registry", "auto_triggers.yaml")
	data, err := os.ReadFile(kcPath)
	if err != nil {
		sendOK(w, map[string]string{"status": "not found"})
		return
	}

	triggerCount := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "- trigger:") ||
			strings.HasPrefix(strings.TrimSpace(line), "- id:") {
			triggerCount++
		}
	}

	sendOK(w, map[string]interface{}{
		"triggers_count": triggerCount,
		"status":         "operational",
		"data":           string(data),
	})
}

// ── Operations ────────────────────────────────────────────────────────────────

func handleOperations(w http.ResponseWriter, r *http.Request) {
	ops := map[string]interface{}{
		"install":  map[string]interface{}{"status": "available", "detail": "Go runtime operational — migration pending"},
		"backup":   map[string]interface{}{"status": "available", "detail": "Backup manager ready (Python, pending Go port)"},
		"deploy":   map[string]interface{}{"status": "clean", "detail": "Workspace safety gate available"},
		"sync":     map[string]interface{}{"status": "clean", "changes": 0},
		"qa":       map[string]interface{}{"status": "available", "checks": map[string]string{"smoke": "available", "export_gate": "available"}},
		"surfaces": map[string]interface{}{"status": "available", "detail": "Surface manager ready"},
		"segment":  map[string]interface{}{"status": "active", "current": gitCmd("branch", "--show-current")},
	}

	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = RepoRoot
	out, err := cmd.Output()
	if err == nil {
		changes := strings.Split(strings.TrimSpace(string(out)), "\n")
		changeCount := 0
		for _, c := range changes {
			if strings.TrimSpace(c) != "" {
				changeCount++
			}
		}
		if changeCount == 0 {
			ops["sync"] = map[string]interface{}{"status": "clean", "changes": 0}
		} else {
			ops["sync"] = map[string]interface{}{"status": "dirty", "changes": changeCount}
		}
	}

	sendOK(w, ops)
}

// ── Economy detail ────────────────────────────────────────────────────────────

func handleEconomyDetail(w http.ResponseWriter, r *http.Request) {
	dashPath := filepath.Join(RepoRoot, ".ovav", "economy", "dashboard.json")
	data, err := os.ReadFile(dashPath)
	if err != nil {
		sendOK(w, map[string]interface{}{
			"session": map[string]interface{}{},
			"monthly": map[string]interface{}{},
			"note":    "dashboard.json not found",
		})
		return
	}

	var dash interface{}
	if err := json.Unmarshal(data, &dash); err != nil {
		sendError(w, "failed to parse economy data", http.StatusInternalServerError)
		return
	}

	sendOK(w, dash)
}
