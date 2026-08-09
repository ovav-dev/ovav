package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SurfaceDrift detects drift between plan-declared UNLOCKED surfaces and runtime-blocked surfaces.
// Replaces: check_surface_drift.py
type SurfaceDrift struct{}

func NewSurfaceDrift() *SurfaceDrift { return &SurfaceDrift{} }

func (s *SurfaceDrift) ID() string   { return "surface_drift" }
func (s *SurfaceDrift) Name() string { return "Surface Drift Detector" }
func (s *SurfaceDrift) Description() string {
	return "Detects drift between plan-unlocked surfaces and runtime-blocked surfaces"
}
func (s *SurfaceDrift) Weight() int { return 12 }

// Normalized surface keys to canonical names.
var surfaceNormalize = map[string]string{
	"mcp":                    "MCP/A2A",
	"a2a":                    "MCP/A2A",
	"mcp_a2a":                "MCP/A2A",
	"external_service":       "external_service",
	"global_opencode_config": "global_opencode_config",
	"global_config_writes":   "global_opencode_config",
	"plugin_install":         "plugin_install",
	"global_install":         "install",
}

func (s *SurfaceDrift) scanPlanUnlocked(root string) map[string]bool {
	unlocked := make(map[string]bool)
	capsPath := filepath.Join(root, ".ovav", "plan", "caps.yaml")
	data, err := os.ReadFile(capsPath)
	if err != nil {
		return unlocked
	}
	text := strings.ToLower(string(data))

	patterns := []struct {
		re  *regexp.Regexp
		key string
	}{
		{regexp.MustCompile(`mcp.*desbloqueado|rag.*desbloqueado`), "MCP/A2A"},
		{regexp.MustCompile(`multi\.target.*completado|e4.*completado|proyección.*completado`), "external_service"},
		{regexp.MustCompile(`external.*service|servicio.*externo`), "external_service"},
		{regexp.MustCompile(`global.*config.*write|opencode.*json.*update|model.*body.*switch`), "global_opencode_config"},
	}

	for _, p := range patterns {
		if p.re.MatchString(text) {
			unlocked[p.key] = true
		}
	}
	return unlocked
}

func (s *SurfaceDrift) getRuntimeBlocked(root string) map[string]bool {
	blocked := make(map[string]bool)
	paPath := filepath.Join(root, ".ovav", "policy", "permission_authority.json")
	data, err := os.ReadFile(paPath)
	if err != nil {
		return blocked
	}
	var permData struct {
		BlockedSurfaces []string `json:"blocked_surfaces"`
	}
	if err := json.Unmarshal(data, &permData); err != nil {
		return blocked
	}
	for _, b := range permData.BlockedSurfaces {
		// Extract surface name before any parentheses
		clean := strings.Split(b, "(")[0]
		clean = strings.TrimSpace(clean)
		clean = strings.ReplaceAll(strings.ToLower(clean), " ", "_")
		if canonical, ok := surfaceNormalize[clean]; ok {
			blocked[canonical] = true
		} else {
			blocked[clean] = true
		}
	}
	return blocked
}

func (s *SurfaceDrift) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	planUnlocked := s.scanPlanUnlocked(root)
	runtimeBlocked := s.getRuntimeBlocked(root)

	// Detect drift: plan says unlocked, runtime still blocks
	drift := false
	for surface := range planUnlocked {
		if runtimeBlocked[surface] {
			drift = true
			issues = append(issues, fmt.Sprintf("SURFACE DRIFT: plan unlocked '%s' but permission_authority still blocks it", surface))
		}
	}

	if len(planUnlocked) == 0 {
		issues = append(issues, "INFO: no unlocked surfaces found in caps.yaml plan")
	}

	if drift {
		return Result{
			ID: s.ID(), Name: s.Name(), Status: "fail", Weight: s.Weight(),
			Message:  fmt.Sprintf("FAIL surface drift — %d drift(s) detected between plan and runtime", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: s.ID(), Name: s.Name(), Status: "pass", Weight: s.Weight(),
		Message:  "PASS surface drift — plan and runtime aligned",
		Duration: time.Since(start),
	}
}

var _ Validator = (*SurfaceDrift)(nil)
