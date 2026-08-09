// OVAV cPanel v5.0 — Profiles handler.
//
// GET  /api/v1/profiles        — List available profiles (native Go)
// POST /api/v1/profiles/apply  — Apply a profile (native Go)
//
// 🔒 OVAV GOVERNED: Reads service_profiles.yaml natively. Zero Python bridge.
//    Profile apply is read-only preview — full apply logic tracked in GO-PROFILES cap.

package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ── Profile data structures ───────────────────────────────────────────────────

type serviceProfile struct {
	DisplayName             string   `json:"display_name" yaml:"display_name"`
	ProfessionalServiceArea string   `json:"professional_service_area" yaml:"professional_service_area"`
	VisibleServiceCategory  string   `json:"visible_service_category" yaml:"visible_service_category"`
	LeadOperator            string   `json:"lead_operator" yaml:"lead_operator"`
	CustomerVisible         bool     `json:"customer_visible" yaml:"customer_visible"`
	P0                      bool     `json:"p0" yaml:"p0"`
	Purpose                 string   `json:"purpose" yaml:"purpose"`
	Lanes                   []string `json:"lanes" yaml:"lanes"`
}

// ── Profile list (native Go — reads service_profiles.yaml) ────────────────────

func handleProfileList(w http.ResponseWriter, r *http.Request) {
	profilesPath := filepath.Join(RepoRoot, ".ovav", "registry", "service_profiles.yaml")

	data, err := os.ReadFile(profilesPath)
	if err != nil {
		sendOK(w, map[string]interface{}{
			"ok":       false,
			"error":    "service_profiles.yaml not found",
			"profiles": []interface{}{},
			"engine":   "OVAV Go Native Profiles v5.0",
		})
		return
	}

	// Parse service_profiles YAML structure:
	// service_profiles:
	//   profile_id:
	//     display_name: "..."
	//     professional_service_area: "..."
	//     lead_operator: "..."
	//     ...
	profiles := []interface{}{}
	inProfiles := false
	currentID := ""
	current := make(map[string]interface{})

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Detect service_profiles: section start
		if !inProfiles && trimmed == "service_profiles:" {
			inProfiles = true
			continue
		}

		if !inProfiles {
			continue
		}

		// Detect if we've left the service_profiles section
		// (line with no indent = new top-level key)
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			break
		}

		// Profile ID: indented with 2 spaces, ends with ":"
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") && strings.HasSuffix(trimmed, ":") {
			// Save previous profile
			if currentID != "" && len(current) > 0 {
				current["id"] = currentID
				profiles = append(profiles, current)
			}
			currentID = strings.TrimSuffix(trimmed, ":")
			current = make(map[string]interface{})
			continue
		}

		// Profile fields: indented with 4 spaces, key: value
		if currentID != "" && strings.HasPrefix(line, "    ") && strings.Contains(trimmed, ":") {
			parts := strings.SplitN(trimmed, ":", 2)
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			value = strings.Trim(value, "\"'")

			switch key {
			case "display_name":
				current["name"] = value
			case "visible_service_category":
				current["area"] = value
			case "lead_operator":
				current["lead"] = value
			case "purpose":
				current["description"] = value
			case "customer_visible":
				current["visible"] = value == "true"
			case "p0":
				current["priority"] = value == "true"
			case "professional_service_area":
				current["service_area"] = value
			}
		}
	}

	// Save last profile
	if currentID != "" && len(current) > 0 {
		current["id"] = currentID
		profiles = append(profiles, current)
	}

	sendOK(w, map[string]interface{}{
		"ok":       true,
		"profiles": profiles,
		"total":    len(profiles),
		"engine":   "OVAV Go Native Profiles v5.0",
		"note":     "Full profile apply logic tracked in GO-PROFILES cap",
	})
}

// ── Profile apply (read-only preview — Go migration pending) ──────────────────

func handleProfileApply(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Area   string `json:"area"`
		Target string `json:"target"`
		DryRun bool   `json:"dry_run"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if body.Area == "" {
		sendError(w, "area required", http.StatusBadRequest)
		return
	}

	// Verify profile exists in service_profiles.yaml
	profilesPath := filepath.Join(RepoRoot, ".ovav", "registry", "service_profiles.yaml")
	data, err := os.ReadFile(profilesPath)
	if err != nil {
		sendError(w, "service_profiles.yaml not found", http.StatusInternalServerError)
		return
	}

	profileExists := strings.Contains(string(data), body.Area)

	if !profileExists {
		sendError(w, "profile '"+body.Area+"' not found in service_profiles.yaml", http.StatusNotFound)
		return
	}

	sendOK(w, map[string]interface{}{
		"ok":      true,
		"area":    body.Area,
		"dry_run": body.DryRun,
		"status":  "preview",
		"message": "Profile '" + body.Area + "' found. Full apply logic tracked in GO-PROFILES cap.",
		"engine":  "OVAV Go Native Profiles v5.0",
	})
}
