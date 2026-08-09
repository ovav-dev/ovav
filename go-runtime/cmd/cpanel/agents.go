// OVAV cPanel v5.0 — Agents & Topology handlers.
//
// GET /api/v1/agents             — Agent definitions list
// GET /api/v1/agents/topology    — Topology overview
// GET /api/v1/agents/profiles    — Service profiles
// GET /api/v1/agents/permissions — Permission authority

package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ── Agent list ────────────────────────────────────────────────────────────────

func handleAgentList(w http.ResponseWriter, r *http.Request) {
	agentsDir := filepath.Join(RepoRoot, ".opencode", "agents")
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		sendOK(w, map[string]interface{}{"agents": []interface{}{}, "total": 0})
		return
	}

	type agentInfo struct {
		Name   string
		File   string
		Mode   string
		Hidden bool
	}

	var agents []agentInfo
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		data, err := os.ReadFile(filepath.Join(agentsDir, e.Name()))
		if err != nil {
			continue
		}
		content := string(data)
		if len(content) > 500 {
			content = content[:500]
		}

		hidden := strings.Contains(content, "hidden: true") || strings.Contains(content, "hidden:true")

		mode := "subagent"
		for _, line := range strings.Split(content, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "mode:") {
				mode = strings.TrimSpace(strings.TrimPrefix(trimmed, "mode:"))
				break
			}
		}

		agents = append(agents, agentInfo{Name: name, File: e.Name(), Mode: mode, Hidden: hidden})
	}

	// Sort by name
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].Name < agents[j].Name
	})

	result := []interface{}{}
	for _, a := range agents {
		result = append(result, map[string]interface{}{
			"name":   a.Name,
			"file":   a.File,
			"mode":   a.Mode,
			"hidden": a.Hidden,
		})
	}

	sendOK(w, map[string]interface{}{"agents": result, "total": len(result)})
}

// ── Topology ──────────────────────────────────────────────────────────────────

func handleTopology(w http.ResponseWriter, r *http.Request) {
	topoDir := filepath.Join(RepoRoot, ".ovav", "topology")
	result := map[string]interface{}{
		"areas":      []interface{}{},
		"governance": nil,
	}

	entries, err := os.ReadDir(topoDir)
	if err != nil {
		sendOK(w, result)
		return
	}

	areas := []interface{}{}
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "area_") || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(topoDir, e.Name()))
		if err != nil {
			continue
		}
		// Pass YAML content as raw string for now
		areas = append(areas, map[string]interface{}{
			"file": e.Name(),
			"data": string(data),
		})
	}
	result["areas"] = areas

	// Governance rules
	govPath := filepath.Join(topoDir, "governance_rules.yaml")
	if data, err := os.ReadFile(govPath); err == nil {
		result["governance"] = string(data)
	}

	sendOK(w, result)
}

// ── Service profiles ──────────────────────────────────────────────────────────

func handleServiceProfiles(w http.ResponseWriter, r *http.Request) {
	profilesPath := filepath.Join(RepoRoot, ".ovav", "registry", "service_profiles.yaml")
	data, err := os.ReadFile(profilesPath)
	if err != nil {
		sendOK(w, map[string]interface{}{"profiles": []interface{}{}, "note": "not found"})
		return
	}

	// Parse simple YAML structure
	lines := strings.Split(string(data), "\n")
	profiles := []interface{}{}
	current := make(map[string]interface{})
	inProfile := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- id:") || strings.HasPrefix(trimmed, "- name:") {
			if inProfile && len(current) > 0 {
				profiles = append(profiles, current)
			}
			current = make(map[string]interface{})
			if strings.HasPrefix(trimmed, "- id:") {
				current["id"] = strings.TrimSpace(strings.TrimPrefix(trimmed, "- id:"))
			} else {
				current["name"] = strings.TrimSpace(strings.TrimPrefix(trimmed, "- name:"))
			}
			inProfile = true
			continue
		}
		if inProfile && strings.Contains(trimmed, ":") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				current[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}
	if inProfile && len(current) > 0 {
		profiles = append(profiles, current)
	}

	sendOK(w, map[string]interface{}{"profiles": profiles, "total": len(profiles)})
}

// ── Permissions ───────────────────────────────────────────────────────────────

func handlePermissions(w http.ResponseWriter, r *http.Request) {
	permPath := filepath.Join(RepoRoot, ".ovav", "policy", "permission_authority.json")
	data, err := os.ReadFile(permPath)
	if err != nil {
		sendOK(w, map[string]string{"note": "permission_authority.json not found"})
		return
	}

	var perms interface{}
	if err := json.Unmarshal(data, &perms); err != nil {
		sendError(w, "failed to parse permission authority", http.StatusInternalServerError)
		return
	}

	sendOK(w, perms)
}
