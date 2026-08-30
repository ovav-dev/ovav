package hostsync

import (
	"fmt"
	"path/filepath"

	"github.com/ovav/ovav/internal/hostprojection"
)

type profileDefinition struct {
	profile  Profile
	resolve  func(roots roots) (destination, allowedRoot string)
	validate hostprojection.SourceValidator
}

type roots struct {
	repoRoot    string
	home        string
	windowsHome string
}

var profileRegistry = []profileDefinition{
	{
		profile: Profile{Name: "opencode-bootstrap", SourceRelative: "ops/host-projections/opencode-bootstrap.json"},
		resolve: func(roots roots) (string, string) {
			root := filepath.Join(roots.home, ".config", "opencode")
			return filepath.Join(root, "opencode.json"), root
		},
		validate: validateOpenCodeBootstrap,
	},
	{
		profile: Profile{Name: "wsl2-resource-policy", SourceRelative: "ops/host-projections/wsl2/.wslconfig", Windows: true},
		resolve: func(roots roots) (string, string) {
			return filepath.Join(roots.windowsHome, ".wslconfig"), roots.windowsHome
		},
		validate: validateWSLResourcePolicy,
	},
	{
		profile: Profile{Name: "warp-wsl-tab", SourceRelative: "ops/host-projections/warp/ovav_wsl.toml", Windows: true},
		resolve: func(roots roots) (string, string) {
			root := filepath.Join(roots.windowsHome, "AppData", "Roaming", "warp", "Warp", "data", "tab_configs")
			return filepath.Join(root, "ovav_wsl.toml"), root
		},
		validate: validateWarpWSLTab,
	},
}

// Profiles returns a copy of the exact allowlisted profile registry.
func Profiles() []Profile {
	profiles := make([]Profile, 0, len(profileRegistry))
	for _, definition := range profileRegistry {
		profiles = append(profiles, definition.profile)
	}
	return profiles
}

// IsProfile reports whether name exactly matches an allowlisted profile.
func IsProfile(name string) bool {
	_, ok := profileByName(name)
	return ok
}

// RequiresWindowsHome reports whether an exact profile requires Windows home.
func RequiresWindowsHome(name string) (bool, bool) {
	definition, ok := profileByName(name)
	return definition.profile.Windows, ok
}

func profileByName(name string) (profileDefinition, bool) {
	for _, definition := range profileRegistry {
		if definition.profile.Name == name {
			return definition, true
		}
	}
	return profileDefinition{}, false
}

func resolveDefinition(definition profileDefinition, resolved roots) (resolvedProfile, error) {
	if definition.profile.Windows && resolved.windowsHome == "" {
		return resolvedProfile{}, fmt.Errorf("profile %q requires an absolute Windows home", definition.profile.Name)
	}
	if !definition.profile.Windows && resolved.windowsHome != "" {
		return resolvedProfile{}, fmt.Errorf("--windows-home is not valid for profile %q", definition.profile.Name)
	}
	destination, allowedRoot := definition.resolve(resolved)
	return resolvedProfile{
		definition:  definition,
		source:      filepath.Join(resolved.repoRoot, filepath.FromSlash(definition.profile.SourceRelative)),
		destination: destination,
		allowedRoot: allowedRoot,
		backupRoot:  filepath.Join(resolved.home, ".local", "state", "ovav", "host-projection"),
	}, nil
}

type resolvedProfile struct {
	definition  profileDefinition
	source      string
	destination string
	allowedRoot string
	backupRoot  string
}
