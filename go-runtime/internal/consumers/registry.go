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
	"os/exec"
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

// AuthorityRoot returns the registered consumer root. For an external
// worktree this deliberately returns the registered checkout, not the
// temporary worktree path, so central state remains shared by repository
// identity rather than by a mutable filesystem location.
func (p Profile) AuthorityRoot() string {
	if p.Consumer == nil {
		return ""
	}
	return canonicalPath(p.Consumer.RootPath)
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
	identity, identityErr := gitIdentity(root)
	for i := range registry.Consumers {
		consumer := &registry.Consumers[i]
		registeredRoot := canonicalPath(consumer.RootPath)
		registeredIdentity, registeredErr := gitIdentity(registeredRoot)
		identityMatch := identityErr == nil && registeredErr == nil && identity.CommonDir == registeredIdentity.CommonDir
		if !identityMatch {
			continue
		}
		profile.Consumer = consumer
		profile.Registered = strings.EqualFold(strings.TrimSpace(consumer.State), "registered")
		if !profile.Registered {
			profile.Err = fmt.Errorf("consumer %q is not active (state=%q)", consumer.ID, consumer.State)
		}
		return profile
	}
	if identityErr != nil {
		profile.Err = fmt.Errorf("repository is not registered as an OVAV consumer: cannot resolve Git identity: %w", identityErr)
		return profile
	}
	profile.Err = fmt.Errorf("repository is not registered as an OVAV consumer")
	return profile
}

type gitRepositoryIdentity struct {
	CommonDir string
}

// gitIdentity is the only identity used to associate an external worktree
// with a registered consumer. It never reads the candidate's .ovav files.
// The input must be the Git toplevel, preventing a path that merely happens
// to be below a registered checkout from being treated as a repository root.
func gitIdentity(root string) (gitRepositoryIdentity, error) {
	if root == "" {
		return gitRepositoryIdentity{}, fmt.Errorf("empty repository path")
	}

	toplevel, err := gitRevParse(root, "--show-toplevel")
	if err != nil {
		return gitRepositoryIdentity{}, fmt.Errorf("not a Git repository: %w", err)
	}
	if canonicalPath(toplevel) != root {
		return gitRepositoryIdentity{}, fmt.Errorf("path is not the Git toplevel")
	}

	commonDir, err := gitRevParse(root, "--git-common-dir")
	if err != nil {
		return gitRepositoryIdentity{}, fmt.Errorf("read Git common directory: %w", err)
	}
	if !filepath.IsAbs(commonDir) {
		commonDir = filepath.Join(root, commonDir)
	}
	commonDir = canonicalPath(commonDir)
	if commonDir == "" {
		return gitRepositoryIdentity{}, fmt.Errorf("empty Git common directory")
	}
	return gitRepositoryIdentity{CommonDir: commonDir}, nil
}

func gitRevParse(root string, arg string) (string, error) {
	cmd := exec.Command("git", "-C", root, "rev-parse", arg)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(string(out))
	if value == "" {
		return "", fmt.Errorf("git rev-parse %s returned empty output", arg)
	}
	return value, nil
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
