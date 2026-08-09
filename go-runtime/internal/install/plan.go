package install

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// InstallPack represents the top-level structure of install_packs.yaml.
type InstallPack struct {
	Modes   []string `yaml:"modes"`
	Targets []string `yaml:"targets"`
	Notes   string   `yaml:"notes"`
}

// installPacksFile holds the YAML structure.
type installPacksFile struct {
	InstallPacks map[string]InstallPack `yaml:"install_packs"`
}

// LoadInstallPacks reads and parses the install packs registry.
func LoadInstallPacks(repoRoot string) (map[string]InstallPack, error) {
	packsPath := filepath.Join(repoRoot, ".ovav", "registry", "install_packs.yaml")
	data, err := os.ReadFile(packsPath)
	if err != nil {
		return nil, fmt.Errorf("install: read install_packs.yaml: %w", err)
	}
	var file installPacksFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("install: parse install_packs.yaml: %w", err)
	}
	return file.InstallPacks, nil
}

// BuildPlan generates a deterministic install plan for a given pack and mode.
func BuildPlan(packID string, mode Mode, repoRoot string) Plan {
	root, err := filepath.Abs(repoRoot)
	if err != nil {
		return Plan{
			Status: "fail",
			PackID: packID,
			Error:  fmt.Sprintf("invalid_repo_root: %v", err),
		}
	}

	packs, err := LoadInstallPacks(root)
	if err != nil {
		return Plan{
			Status: "fail",
			PackID: packID,
			Error:  fmt.Sprintf("load_packs_failed: %v", err),
		}
	}

	pack, ok := packs[packID]
	if !ok {
		available := make([]string, 0, len(packs))
		for k := range packs {
			available = append(available, k)
		}
		sort.Strings(available)
		errMsg := fmt.Sprintf("unknown_pack: %s", packID)
		if packID == "default" {
			errMsg = fmt.Sprintf("unknown_pack: %s. Use --pack-id=<name> or --full. Available packs: %v", packID, available)
		}
		return Plan{
			Status: "fail",
			PackID: packID,
			Error:  errMsg,
		}
	}

	// Check if mode is allowed for this pack
	modeAllowed := false
	modeStr := string(mode)
	for _, m := range pack.Modes {
		if m == modeStr {
			modeAllowed = true
			break
		}
	}
	if !modeAllowed {
		return Plan{
			Status:       "blocked",
			PackID:       packID,
			Mode:         mode,
			Reason:       fmt.Sprintf("mode %q not in allowed modes: %v", mode, pack.Modes),
			AllowedModes: pack.Modes,
		}
	}

	// Build entries
	entries := make([]PlanEntry, 0, len(pack.Targets))
	for _, targetID := range pack.Targets {
		targetPath := resolveTarget(targetID, mode, root)
		sourcePath := resolveSource(targetID, root)
		risk := classifyRisk(targetPath)

		entries = append(entries, PlanEntry{
			TargetID:     targetID,
			Target:       targetPath,
			Source:       sourcePath,
			TargetRisk:   risk,
			Mode:         mode,
			WriteEnabled: mode == ModeSandbox || mode == ModeSourceLocalApply,
		})
	}

	return Plan{
		Status:       "pass",
		PackID:       packID,
		PackNotes:    pack.Notes,
		Mode:         mode,
		RealApply:    mode == ModeSourceLocalApply,
		DryRunOnly:   mode == ModeDryRun,
		SandboxOnly:  mode == ModeSandbox,
		Entries:      entries,
		EntryCount:   len(entries),
		AllowedModes: pack.Modes,
	}
}

// resolveTarget maps a target ID to a concrete filesystem path.
func resolveTarget(targetID string, mode Mode, root string) string {
	targetMap := map[string]string{
		"ovav_artifacts":     filepath.Join(root, ".ovav", "artifacts"),
		"registry":           filepath.Join(root, ".ovav", "registry"),
		"source_runtime":     filepath.Join(root, "tools"),
		"opencode_agents":    filepath.Join(root, ".opencode", "agents"),
		"opencode_skills":    filepath.Join(root, ".opencode", "skills"),
		"opencode_commands":  filepath.Join(root, ".opencode", "commands"),
		"research_artifacts": filepath.Join(root, ".ovav", "artifacts", "research"),
		"source_maps":        filepath.Join(root, ".ovav", "artifacts", "source_maps"),
		"reports":            filepath.Join(root, ".ovav", "reports"),
	}

	if mode == ModeSandbox {
		sandboxBase := filepath.Join(root, ".ovav", "artifacts", "S86", "evidence", "sandbox")
		return filepath.Join(sandboxBase, targetID)
	}

	if path, ok := targetMap[targetID]; ok {
		return path
	}
	return filepath.Join(root, targetID)
}

// resolveSource maps a target ID to its source path. Returns empty string for synthetic targets.
func resolveSource(targetID string, root string) string {
	sourceMap := map[string]string{
		"registry":          filepath.Join(root, ".ovav", "registry"),
		"source_runtime":    filepath.Join(root, "tools"),
		"opencode_agents":   filepath.Join(root, ".opencode", "agents"),
		"opencode_skills":   filepath.Join(root, ".opencode", "skills"),
		"opencode_commands": filepath.Join(root, ".opencode", "commands"),
	}
	if src, ok := sourceMap[targetID]; ok {
		return src
	}
	return ""
}

// classifyRisk classifies a target path's risk level based on heuristics.
func classifyRisk(target string) string {
	t := strings.ToLower(target)
	switch {
	case strings.Contains(target, "~/") || strings.Contains(t, "/home/") || strings.Contains(t, "/users/"):
		if strings.Contains(t, ".config") {
			return "user-config-risk"
		}
		if strings.Contains(t, ".local") {
			return "user-local-risk"
		}
		return "user-home-risk"
	case strings.HasPrefix(t, "/etc") || strings.HasPrefix(t, "/usr") || strings.HasPrefix(t, "/opt"):
		return "global-risk"
	case strings.Contains(t, ".opencode/") && (strings.Contains(t, "global") || strings.Contains(t, "plugin")):
		return "global-risk"
	case strings.Contains(t, "sandbox") || strings.Contains(t, "artifacts") || strings.Contains(t, "evidence"):
		return "sandbox"
	default:
		return "repo-local"
	}
}
