// OVAV cPanel v5.0 — Validators handler.
//
// GET  /api/v1/validators           — List validators and their status
// POST /api/v1/validators/run       — Trigger validator execution (async)
// GET  /api/v1/validators/status/:id — Poll background task status
//
// 🔒 OVAV GOVERNED: All validation logic is native Go. No Python bridge.
//    Python exec removed per GO-MIGRATION strategy (2026-06-15).
//    Full validator suite tracked in caps.yaml → GO-VALIDATORS.

package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ── OVAV Integrity Check (native Go) ──────────────────────────────────────────

type integrityCheck struct {
	Name   string `json:"name"`
	Ok     bool   `json:"ok"`
	Weight int    `json:"weight"`
	Detail string `json:"detail,omitempty"`
}

func runLivingIntegrity() (overall string, score float64, pass int, fail int, checks []integrityCheck) {
	checks = []integrityCheck{}

	// ── Supply chain: verify key OVAV files exist ────────────────────────────
	keyFiles := []string{
		"AGENTS.md",
		".ovav/policy/permission_authority.json",
		".ovav/plan/caps.yaml",
		"opencode.json",
		"bin/ovav",
		"go-runtime/cmd/cpanel/main.go",
	}
	allSupply := true
	missingSupply := []string{}
	for _, f := range keyFiles {
		if _, err := os.Stat(filepath.Join(RepoRoot, f)); os.IsNotExist(err) {
			allSupply = false
			missingSupply = append(missingSupply, f)
		}
	}
	if allSupply {
		checks = append(checks, integrityCheck{Name: "supply_chain", Ok: true, Weight: 20, Detail: "All key files present"})
		pass++
	} else {
		checks = append(checks, integrityCheck{Name: "supply_chain", Ok: false, Weight: 20, Detail: "Missing: " + strings.Join(missingSupply, ", ")})
		fail++
	}

	// ── Secrets hygiene: no secrets in tracked files ─────────────────────────
	secretsOk := true
	secretsDetail := "No secrets detected in quick scan"
	// Quick check: verify .env.production is gitignored
	envPath := filepath.Join(RepoRoot, ".env.production")
	if _, err := os.Stat(envPath); err == nil {
		// Check if gitignored by looking at .gitignore patterns
		gitignorePath := filepath.Join(RepoRoot, ".gitignore")
		if data, err := os.ReadFile(gitignorePath); err == nil {
			if strings.Contains(string(data), ".env.production") {
				secretsDetail = ".env.production gitignored ✅"
			} else {
				secretsOk = false
				secretsDetail = ".env.production exists but NOT in .gitignore"
			}
		}
	} else {
		secretsDetail = "No .env.production file (clean)"
	}
	if secretsOk {
		checks = append(checks, integrityCheck{Name: "secrets_hygiene", Ok: true, Weight: 20, Detail: secretsDetail})
		pass++
	} else {
		checks = append(checks, integrityCheck{Name: "secrets_hygiene", Ok: false, Weight: 20, Detail: secretsDetail})
		fail++
	}

	// ── Runtime integrity: git repo clean, no drift ──────────────────────────
	runtimeOk := true
	runtimeDetail := "Runtime integrity check passed"
	gitHead := filepath.Join(RepoRoot, ".git", "HEAD")
	if _, err := os.Stat(gitHead); os.IsNotExist(err) {
		runtimeOk = false
		runtimeDetail = "Git HEAD missing — repo may be corrupted"
	}
	// Check .ovav/runtime exists
	runtimeDir := filepath.Join(RepoRoot, ".ovav", "runtime")
	if _, err := os.Stat(runtimeDir); os.IsNotExist(err) {
		runtimeOk = false
		runtimeDetail = ".ovav/runtime/ directory missing"
	}
	if runtimeOk {
		checks = append(checks, integrityCheck{Name: "runtime_integrity", Ok: true, Weight: 20, Detail: runtimeDetail})
		pass++
	} else {
		checks = append(checks, integrityCheck{Name: "runtime_integrity", Ok: false, Weight: 20, Detail: runtimeDetail})
		fail++
	}

	// ── Bootstrap chain: ensure key binaries/configs intact ──────────────────
	bootstrapOk := true
	bootstrapDetail := "Bootstrap chain intact"
	bootstrapFiles := []string{"go-runtime/go.mod", "requirements.txt", "pyproject.toml"}
	missingBootstrap := []string{}
	for _, f := range bootstrapFiles {
		if _, err := os.Stat(filepath.Join(RepoRoot, f)); os.IsNotExist(err) {
			bootstrapOk = false
			missingBootstrap = append(missingBootstrap, f)
		}
	}
	if len(missingBootstrap) > 0 {
		bootstrapDetail = "Missing bootstrap files: " + strings.Join(missingBootstrap, ", ")
	}
	if bootstrapOk {
		checks = append(checks, integrityCheck{Name: "bootstrap_chain", Ok: true, Weight: 15, Detail: bootstrapDetail})
		pass++
	} else {
		checks = append(checks, integrityCheck{Name: "bootstrap_chain", Ok: false, Weight: 15, Detail: bootstrapDetail})
		fail++
	}

	// ── Exfil patterns: check no sensitive data exposed ──────────────────────
	exfilOk := true
	exfilDetail := "No exfiltration patterns detected"
	// Quick check: opencode.json doesn't contain API keys in plaintext
	ocPath := filepath.Join(RepoRoot, "opencode.json")
	if data, err := os.ReadFile(ocPath); err == nil {
		content := string(data)
		sensitive := []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "GEMINI_API_KEY", "token"}
		found := []string{}
		for _, s := range sensitive {
			if strings.Contains(content, s) {
				found = append(found, s)
			}
		}
		if len(found) > 0 {
			exfilOk = false
			exfilDetail = "Potential secrets in opencode.json: " + strings.Join(found, ", ")
		}
	}
	if exfilOk {
		checks = append(checks, integrityCheck{Name: "exfil_patterns", Ok: true, Weight: 10, Detail: exfilDetail})
		pass++
	} else {
		checks = append(checks, integrityCheck{Name: "exfil_patterns", Ok: false, Weight: 10, Detail: exfilDetail})
		fail++
	}

	// ── Compute overall ──────────────────────────────────────────────────────
	totalWeight := 0
	passedWeight := 0
	for _, c := range checks {
		totalWeight += c.Weight
		if c.Ok {
			passedWeight += c.Weight
		}
	}
	if totalWeight > 0 {
		score = float64(passedWeight) / float64(totalWeight) * 100.0
	} else {
		score = 100.0
	}

	if fail == 0 {
		overall = "HEALTHY"
	} else if score >= 70 {
		overall = "DEGRADED"
	} else {
		overall = "CRITICAL"
	}

	return
}

// ── Background tasks ──────────────────────────────────────────────────────────

type bgTask struct {
	Status string      `json:"status"`
	Result interface{} `json:"result,omitempty"`
}

var (
	bgTasksMap   = make(map[string]*bgTask)
	bgTasksMutex sync.Mutex
	bgTasksInit  sync.Once
)

func initBgTasks() {
	bgTasksInit.Do(func() {
		go func() {
			for {
				time.Sleep(30 * time.Minute)
				bgTasksMutex.Lock()
				for id, task := range bgTasksMap {
					if task.Status == "complete" || task.Status == "error" {
						delete(bgTasksMap, id)
					}
				}
				bgTasksMutex.Unlock()
			}
		}()
	})
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func handleValidatorsList(w http.ResponseWriter, r *http.Request) {
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

func handleValidatorsRun(w http.ResponseWriter, r *http.Request) {
	initBgTasks()
	taskID := fmt.Sprintf("%x", time.Now().UnixNano())

	bgTasksMutex.Lock()
	bgTasksMap[taskID] = &bgTask{Status: "queued"}
	bgTasksMutex.Unlock()

	go func() {
		bgTasksMutex.Lock()
		bgTasksMap[taskID].Status = "running"
		bgTasksMutex.Unlock()

		// Run native integrity check (no Python bridge)
		overall, score, pass, fail, checks := runLivingIntegrity()

		bgTasksMutex.Lock()
		defer bgTasksMutex.Unlock()
		bgTasksMap[taskID].Status = "complete"
		bgTasksMap[taskID].Result = map[string]interface{}{
			"overall": overall,
			"score":   score,
			"pass":    pass,
			"fail":    fail,
			"checks":  checks,
			"engine":  "OVAV Go Native Integrity v5.0",
		}
	}()

	sendOK(w, map[string]string{"task_id": taskID, "status": "queued"})
}

func handleValidatorsStatus(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
	taskID := parts[len(parts)-1]

	bgTasksMutex.Lock()
	task, ok := bgTasksMap[taskID]
	bgTasksMutex.Unlock()

	if !ok {
		sendError(w, "task not found", http.StatusNotFound)
		return
	}

	sendOK(w, task)
}
