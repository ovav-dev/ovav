// OVAV cPanel v5.0 — Memory & State handlers.
//
// GET /api/v1/memory/status   — Memory subsystem overview (from git HEAD + caps.yaml)
// GET /api/v1/memory/ledger   — Context timeline (from git log + caps.yaml)
// GET /api/v1/memory/beliefs  — Beliefs (pending rearchitecture)
// Context ledger is PERMANENTLY DEPRECATED per AGENTS.md.
// All memory/ledger/beliefs now derive from canonical sources: git HEAD + caps.yaml.

package main

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ── Memory status ─────────────────────────────────────────────────────────────

func handleMemoryStatus(w http.ResponseWriter, r *http.Request) {
	result := map[string]interface{}{
		"governor":         "operational",
		"belief_manager":   "operational",
		"feedback_loop":    "operational",
		"ledger_cards":     0,
		"active_cards":     0,
		"deprecated_cards": 0,
		"simplified":       true,
		"source":           "git HEAD + caps.yaml (ledger permanently deprecated)",
	}

	// Derive memory status from caps.yaml (canonical plan data)
	capsPath := filepath.Join(RepoRoot, ".ovav", "plan", "caps.yaml")
	if data, err := os.ReadFile(capsPath); err == nil {
		lines := strings.Split(string(data), "\n")
		capCount := 0
		activeCount := 0
		inCaps := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "caps:" {
				inCaps = true
				continue
			}
			// Stop at pending: section
			if inCaps && strings.HasPrefix(trimmed, "pending:") {
				break
			}
			// Top-level cap entry: "  C1:" style keys
			if inCaps && !strings.HasPrefix(trimmed, "#") && strings.HasSuffix(trimmed, ":") &&
				strings.Count(line, " ")-strings.Count(strings.TrimLeft(line, " "), " ") == 2 && trimmed != "caps:" {
				capCount++
				continue
			}
			if inCaps && strings.Contains(line, "status: done") {
				activeCount++
			}
		}
		result["ledger_cards"] = capCount
		result["active_cards"] = activeCount
	}

	// Recent commits from git HEAD
	cmd := exec.Command("git", "log", "--oneline", "-5")
	cmd.Dir = RepoRoot
	if out, err := cmd.Output(); err == nil {
		commits := strings.Split(strings.TrimSpace(string(out)), "\n")
		count := 0
		for _, c := range commits {
			if strings.TrimSpace(c) != "" {
				count++
			}
		}
		result["recent_commits"] = count
	}

	sendOK(w, result)
}

// ── Context ledger ────────────────────────────────────────────────────────────

func handleLedger(w http.ResponseWriter, r *http.Request) {
	// Context ledger is permanently deprecated per AGENTS.md. Derive context
	// from canonical sources: git HEAD (temporal) + caps.yaml (plan data).
	cards := []interface{}{}

	// Git log as recent context timeline
	cmd := exec.Command("git", "log", "--oneline", "-20", "--format=%h %s (%ar)")
	cmd.Dir = RepoRoot
	if out, err := cmd.Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			cards = append(cards, map[string]interface{}{
				"id":      trimmed,
				"summary": trimmed,
				"status":  "git-history",
			})
		}
	}

	// Caps from caps.yaml as plan context cards
	capsPath := filepath.Join(RepoRoot, ".ovav", "plan", "caps.yaml")
	if data, err := os.ReadFile(capsPath); err == nil {
		lines := strings.Split(string(data), "\n")
		inCaps := false
		currentCapID := ""
		currentSummary := ""
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "caps:" {
				inCaps = true
				continue
			}
			if inCaps && strings.HasPrefix(trimmed, "pending:") {
				break
			}
			if inCaps && !strings.HasPrefix(trimmed, "#") && strings.HasSuffix(trimmed, ":") &&
				strings.Count(line, " ")-strings.Count(strings.TrimLeft(line, " "), " ") == 2 && trimmed != "caps:" {
				// Save previous cap
				if currentCapID != "" {
					cards = append(cards, map[string]interface{}{
						"id":      currentCapID,
						"summary": currentSummary,
						"status":  "plan-cap",
					})
				}
				currentCapID = strings.TrimSuffix(trimmed, ":")
				currentSummary = currentCapID
				continue
			}
			// Track summary line
			if inCaps && currentCapID != "" && strings.HasPrefix(strings.TrimSpace(line), "summary:") {
				currentSummary = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "summary:"))
			}
			// Track status as we go
			if inCaps && currentCapID != "" && strings.HasPrefix(strings.TrimSpace(line), "status:") {
				status := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "status:"))
				// Update the cap status as we encounter it
				for i, c := range cards {
					if m, ok := c.(map[string]interface{}); ok && m["id"] == currentCapID {
						m["status"] = "plan-cap-" + status
						cards[i] = m
					}
				}
			}
		}
		// Save last cap
		if currentCapID != "" {
			cards = append(cards, map[string]interface{}{
				"id":      currentCapID,
				"summary": currentSummary,
				"status":  "plan-cap",
			})
		}
	}

	sendOK(w, map[string]interface{}{
		"cards":      cards,
		"total":      len(cards),
		"simplified": true,
		"note":       "Context ledger is permanently deprecated. Derived from git HEAD + caps.yaml.",
	})
}

// ── Beliefs ───────────────────────────────────────────────────────────────────

func handleBeliefs(w http.ResponseWriter, r *http.Request) {
	// Beliefs subsystem pending rearchitecture.
	// Context ledger permanently deprecated — beliefs
	// were previously stored as belief-* cards in that deprecated ledger.
	sendOK(w, map[string]interface{}{
		"beliefs":    []interface{}{},
		"total":      0,
		"simplified": true,
		"note":       "Beliefs subsystem rearchitecture pending. Memory derived from git HEAD + caps.yaml (canonical sources).",
	})
}
