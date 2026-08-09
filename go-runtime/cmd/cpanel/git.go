// OVAV cPanel v5.0 — Git handlers.
//
// GET  /api/v1/git/branches  — List branches
// GET  /api/v1/git/log       — Recent commits
// GET  /api/v1/git/worktrees — List worktrees
// POST /api/v1/git/fetch     — Fetch from origin

package main

import (
	"net/http"
	"os/exec"
	"strconv"
	"strings"
)

// ── Git branches ──────────────────────────────────────────────────────────────

func handleGitBranches(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("git", "branch", "-a")
	cmd.Dir = RepoRoot
	out, err := cmd.Output()
	if err != nil {
		sendError(w, "git branch failed", http.StatusInternalServerError)
		return
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	branches := []string{}
	current := "?"

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "* ") {
			current = strings.TrimPrefix(trimmed, "* ")
			branches = append(branches, current)
		} else {
			branches = append(branches, trimmed)
		}
	}

	sendOK(w, map[string]interface{}{
		"current":  current,
		"branches": branches,
		"total":    len(branches),
	})
}

// ── Git log ───────────────────────────────────────────────────────────────────

func handleGitLog(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		limitStr = "20"
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		sendError(w, "invalid 'limit' parameter: must be an integer", http.StatusBadRequest)
		return
	}
	if limit <= 0 || limit > 1000 {
		limit = 20
	}

	cmd := exec.Command("git", "log", "--oneline", "-"+strconv.Itoa(limit), "--format=%h %s (%ar)")
	cmd.Dir = RepoRoot
	out, err := cmd.Output()
	if err != nil {
		sendError(w, "git log failed", http.StatusInternalServerError)
		return
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	commits := []string{}
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			commits = append(commits, trimmed)
		}
	}

	sendOK(w, map[string]interface{}{
		"commits": commits,
		"total":   len(commits),
	})
}

// ── Git worktrees ─────────────────────────────────────────────────────────────

func handleGitWorktrees(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("git", "worktree", "list")
	cmd.Dir = RepoRoot
	out, err := cmd.Output()
	if err != nil {
		sendError(w, "git worktree failed", http.StatusInternalServerError)
		return
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	worktrees := []string{}
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			worktrees = append(worktrees, trimmed)
		}
	}

	sendOK(w, map[string]interface{}{
		"worktrees": worktrees,
		"total":     len(worktrees),
	})
}

// ── Git fetch ─────────────────────────────────────────────────────────────────

func handleGitFetch(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("git", "fetch", "origin")
	cmd.Dir = RepoRoot
	out, err := cmd.Output()

	result := map[string]interface{}{
		"status":     "ok",
		"output":     strings.TrimSpace(string(out)),
		"returncode": 0,
	}
	if err != nil {
		result["status"] = "error"
		if exitErr, ok := err.(*exec.ExitError); ok {
			result["output"] = strings.TrimSpace(string(out) + "\n" + string(exitErr.Stderr))
			result["returncode"] = exitErr.ExitCode()
		} else {
			result["output"] = err.Error()
			result["returncode"] = -1
		}
		sendJSON(w, result, http.StatusInternalServerError)
		return
	}

	sendOK(w, result)
}
