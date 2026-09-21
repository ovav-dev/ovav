// Package consumers resolves the central governance profile for repositories
// that are not part of the OVAV monorepo.
//
// The registry is deliberately outside the governed repository.  An external
// project may keep its own .ovav directory, but that directory is never an
// authority for OVAV policy, baselines, or grants.
package consumers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const RegistrySchema = "ovav.consumer_registry.v1"

type Registry struct {
	Schema    string     `yaml:"schema"`
	Consumers []Consumer `yaml:"consumers"`
}

type Consumer struct {
	ID         string `yaml:"id"`
	RootPath   string `yaml:"root_path"`
	State      string `yaml:"state"`
	Contact    string `yaml:"contact"`
	WaiverPath string `yaml:"waiver_path"`
}

// Profile is the resolved, read-only governance context for a repository.
type Profile struct {
	External     bool
	Registered   bool
	Consumer     *Consumer
	RegistryPath string
	RegistryRoot string
	Err          error
}

func (p Profile) Active() bool { return !p.External || p.Registered }

// StateDir is central OVAV state for this consumer. It is never inside the
// consumer repository and is derived from the authoritative registry path.
func (p Profile) StateDir() string {
	if p.Consumer == nil || p.RegistryRoot == "" {
		return ""
	}
	return filepath.Join(p.RegistryRoot, ".ovav", "registry", "consumers", p.Consumer.ID)
}

// Resolve classifies root without trusting any project-local OVAV files.
func Resolve(root string) Profile {
	root = canonicalPath(root)
	if isOVAVRoot(root) {
		return Profile{}
	}

	profile := Profile{External: true}
	registryPath := findRegistry()
	if registryPath == "" {
		profile.Err = fmt.Errorf("central consumer registry not found")
		return profile
	}
	profile.RegistryPath = registryPath
	profile.RegistryRoot = registryRoot(registryPath)

	data, err := os.ReadFile(registryPath)
	if err != nil {
		profile.Err = fmt.Errorf("read central consumer registry: %w", err)
		return profile
	}
	var registry Registry
	if err := yaml.Unmarshal(data, &registry); err != nil {
		profile.Err = fmt.Errorf("parse central consumer registry: %w", err)
		return profile
	}
	if registry.Schema != "" && registry.Schema != RegistrySchema {
		profile.Err = fmt.Errorf("unsupported consumer registry schema %q", registry.Schema)
		return profile
	}
	for i := range registry.Consumers {
		consumer := &registry.Consumers[i]
		if canonicalPath(consumer.RootPath) != root {
			continue
		}
		profile.Consumer = consumer
		profile.Registered = strings.EqualFold(strings.TrimSpace(consumer.State), "registered")
		if !profile.Registered {
			profile.Err = fmt.Errorf("consumer %q is not active (state=%q)", consumer.ID, consumer.State)
		}
		return profile
	}
	profile.Err = fmt.Errorf("repository is not registered as an OVAV consumer")
	return profile
}

func IsOVAVRoot(root string) bool { return isOVAVRoot(canonicalPath(root)) }

func isOVAVRoot(root string) bool {
	if root == "" {
		return false
	}
	for _, rel := range []string{".ovav/plan/caps.yaml", "go-runtime/go.mod"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			return false
		}
	}
	return true
}

func findRegistry() string {
	candidates := []string{}
	if value := strings.TrimSpace(os.Getenv("OVAV_CONSUMER_REGISTRY")); value != "" {
		candidates = append(candidates, value)
	}
	for _, envName := range []string{"OVAV_ROOT", "OVAV_SYSTEMS_ROOT"} {
		if value := strings.TrimSpace(os.Getenv(envName)); value != "" {
			// A project-local .ovav directory is not an authority. Only a
			// canonical OVAV checkout may be selected through this shortcut.
			if isOVAVRoot(canonicalPath(value)) {
				candidates = append(candidates, filepath.Join(value, ".ovav", "registry", "consumers.yaml"))
			}
		}
	}
	if executable, err := os.Executable(); err == nil {
		for dir := canonicalPath(filepath.Dir(executable)); dir != "/"; dir = filepath.Dir(dir) {
			candidates = append(candidates, filepath.Join(dir, ".ovav", "registry", "consumers.yaml"))
		}
	}
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		if home, err := os.UserHomeDir(); err == nil {
			configHome = filepath.Join(home, ".config")
		}
	}
	candidates = append(candidates, filepath.Join(configHome, "ovav", "consumers.yaml"))

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return canonicalPath(candidate)
		}
	}
	return ""
}

func registryRoot(path string) string {
	// <root>/.ovav/registry/consumers.yaml
	return filepath.Dir(filepath.Dir(filepath.Dir(path)))
}

func canonicalPath(path string) string {
	if path == "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err == nil {
		path = abs
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	return filepath.Clean(path)
}
