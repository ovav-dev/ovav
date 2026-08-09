// Package cli — envdetect.go: OVAV environment detector.
//
// Go migration of tools/cli/env_detector.py (126 LOC).
// Detects: ovav_dev, ovav_project, external.
package cli

import (
	"os"
	"path/filepath"
)

// EnvType classifies the OVAV environment.
type EnvType string

const (
	EnvOVAVDev     EnvType = "ovav_dev"
	EnvOVAVProject EnvType = "ovav_project"
	EnvExternal    EnvType = "external"
)

// EnvResult holds the environment detection result.
type EnvResult struct {
	Env               EnvType  `json:"env"`
	Root              string   `json:"root,omitempty"`
	HasOVAV           bool     `json:"has_ovav"`
	HasDevTools       bool     `json:"has_dev_tools"`
	CommandsAvailable []string `json:"commands_available"`
	OVAVDir           string   `json:"ovav_dir,omitempty"`
	Suggestion        string   `json:"suggestion,omitempty"`
}

// DetectEnv detects the OVAV environment from startPath upward.
// Replaces env_detector.py detect_env().
func DetectEnv(startPath string) EnvResult {
	if startPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return externalResult()
		}
		startPath = cwd
	}

	current, err := filepath.Abs(startPath)
	if err != nil {
		return externalResult()
	}

	// Walk up to find repo root (.git)
	repoRoot := findGitRoot(current)
	if repoRoot == "" {
		return externalResult()
	}

	ovavDir := filepath.Join(repoRoot, ".ovav")
	hasOVAV := pathExists(ovavDir)
	hasDevTools := pathExists(filepath.Join(repoRoot, "tools", "harnesses"))

	commands := []string{}
	var envType EnvType

	if hasDevTools && hasOVAV {
		envType = EnvOVAVDev
		commands = []string{"status", "update", "config", "help"}
	} else if hasOVAV {
		envType = EnvOVAVProject
		commands = []string{"status", "update", "config", "help"}
	} else {
		envType = EnvExternal
	}

	result := EnvResult{
		Env:               envType,
		Root:              repoRoot,
		HasOVAV:           hasOVAV,
		HasDevTools:       hasDevTools,
		CommandsAvailable: commands,
	}

	if hasOVAV {
		result.OVAVDir = ovavDir
	}

	switch envType {
	case EnvExternal:
		result.Suggestion = "ovav install — inicializar OVAV en este proyecto"
	case EnvOVAVProject:
		result.Suggestion = "ovav status — ver estado del sistema"
	case EnvOVAVDev:
		result.Suggestion = "ovav status — ver salud del sistema de desarrollo"
	}

	return result
}

// IsOVAVDev returns true if we're inside the OVAV development repo.
func IsOVAVDev(startPath string) bool {
	return DetectEnv(startPath).Env == EnvOVAVDev
}

// IsOVAVProject returns true if we're in a user project with OVAV.
func IsOVAVProject(startPath string) bool {
	return DetectEnv(startPath).Env == EnvOVAVProject
}

// GetRepoRoot returns the git repo root, or empty string if not found.
func GetRepoRoot(startPath string) string {
	return DetectEnv(startPath).Root
}

// EffectiveTier delegates to surface_gate for three-tier access control.
// Returns 'public', 'internal', or 'governor'.
func EffectiveTier() string {
	if os.Getenv("OVAV_DEV") == "1" {
		return "internal"
	}
	return "public"
}

// IsInternalTier returns true if internal or governor tier.
func IsInternalTier() bool {
	tier := EffectiveTier()
	return tier == "internal" || tier == "governor"
}

// findGitRoot walks up from dir to find a .git directory or file.
func findGitRoot(dir string) string {
	for {
		gitPath := filepath.Join(dir, ".git")
		if pathExists(gitPath) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func externalResult() EnvResult {
	return EnvResult{
		Env:               EnvExternal,
		HasOVAV:           false,
		HasDevTools:       false,
		CommandsAvailable: []string{},
		Suggestion:        "ovav install — inicializar OVAV en este proyecto",
	}
}
