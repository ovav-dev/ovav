// discovery.go — Secrets auto-discovery from multiple sources.
//
// Phase 2 of OVAV Vault 2026 plan.
// Scans: GitHub Secrets, Fly.io secrets, filesystem .env files.
// Results are compared against the vault to report missing/extra secrets.
package secrets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ovav/ovav/internal/vault/secrets/providers"
)

// DiscoveryConfig controls which sources are scanned.
type DiscoveryConfig struct {
	GitHubOrg   string   // GitHub org to scan (e.g. "ovav-dev")
	GitHubRepos []string // specific repos, or empty for all org repos
	FlyApps     []string // Fly.io apps, or empty for all accessible apps
	SearchPaths []string // filesystem paths to scan for .env files
	ExcludeDirs []string // directories to exclude from filesystem scan
}

// DiscoveryReport contains all discovered secrets organized by source.
type DiscoveryReport struct {
	GitHub map[string][]providers.GitHubDiscoveryResult // repo -> secrets
	Fly    map[string][]providers.FlyDiscoveryResult    // app -> secrets
	Files  []FilesystemSecret                           // .env files
}

// FilesystemSecret represents a secret found in a .env file.
type FilesystemSecret struct {
	Path   string
	Name   string
	Type   SecretType
	Value  []byte
	Source string // always "filesystem"
}

// Discover scans all configured sources and returns a full report.
// Each source is scanned independently; failures are logged but not fatal.
func Discover(cfg DiscoveryConfig) (*DiscoveryReport, error) {
	rep := &DiscoveryReport{
		GitHub: make(map[string][]providers.GitHubDiscoveryResult),
		Fly:    make(map[string][]providers.FlyDiscoveryResult),
		Files:  make([]FilesystemSecret, 0),
	}

	// ── GitHub ─────────────────────────────────────────────────────────────────
	if cfg.GitHubOrg != "" {
		if len(cfg.GitHubRepos) > 0 {
			for _, repo := range cfg.GitHubRepos {
				results, gherr := providers.DiscoverGitHubSecrets(cfg.GitHubOrg, repo)
				if gherr != nil {
					fmt.Fprintf(os.Stderr, "⚠️  GitHub %s/%s: %v\n", cfg.GitHubOrg, repo, gherr)
					continue
				}
				if len(results) > 0 {
					rep.GitHub[repo] = results
				}
			}
		} else {
			all, gherr := providers.DiscoverGitHubOrg(cfg.GitHubOrg)
			if gherr != nil {
				fmt.Fprintf(os.Stderr, "⚠️  GitHub org %s: %v\n", cfg.GitHubOrg, gherr)
			} else {
				rep.GitHub = all
			}
		}
	}

	// ── Fly.io ──────────────────────────────────────────────────────────────────
	if len(cfg.FlyApps) > 0 {
		for _, app := range cfg.FlyApps {
			results, flyerr := providers.DiscoverFlySecrets(app)
			if flyerr != nil {
				fmt.Fprintf(os.Stderr, "⚠️  Fly.io %s: %v\n", app, flyerr)
				continue
			}
			if len(results) > 0 {
				rep.Fly[app] = results
			}
		}
	} else {
		all, flyerr := providers.DiscoverAllFlyApps()
		if flyerr != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Fly.io: %v\n", flyerr)
		} else {
			rep.Fly = all
		}
	}

	// ── Filesystem ──────────────────────────────────────────────────────────────
	if len(cfg.SearchPaths) == 0 {
		cfg.SearchPaths = []string{
			filepath.Join(os.Getenv("HOME"), "Systems"),
			filepath.Join(os.Getenv("HOME"), ".config"),
		}
	}
	if len(cfg.ExcludeDirs) == 0 {
		cfg.ExcludeDirs = []string{
			"node_modules", ".git", ".venv", "venv", "vendor",
			".next", ".cache", "__pycache__",
		}
	}

	fsSecrets, fserr := discoverFilesystem(cfg.SearchPaths, cfg.ExcludeDirs)
	if fserr != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Filesystem: %v\n", fserr)
	} else {
		rep.Files = fsSecrets
	}

	return rep, nil
}

// discoverFilesystem scans cfg.SearchPaths for .env files.
func discoverFilesystem(paths, excludeDirs []string) ([]FilesystemSecret, error) {
	var results []FilesystemSecret

	exclude := make(map[string]bool)
	for _, d := range excludeDirs {
		exclude[d] = true
	}

	for _, root := range paths {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // skip inaccessible
			}
			if info.IsDir() {
				if exclude[info.Name()] {
					return filepath.SkipDir
				}
				return nil
			}
			name := info.Name()
			if name == ".env" || name == ".env.local" ||
				strings.HasPrefix(name, ".env.") ||
				name == ".env.production" ||
				name == ".env.production.local" {
				secrets, err := parseEnvFile(path)
				if err != nil {
					return nil
				}
				for _, s := range secrets {
					s.Path = path
					results = append(results, s)
				}
			}
			return nil
		})
		if err != nil {
			continue
		}
	}

	return results, nil
}

// parseEnvFile reads a .env file and extracts key=value pairs.
func parseEnvFile(path string) ([]FilesystemSecret, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var results []FilesystemSecret
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// KEY=value or KEY="value" or KEY='value'
		idx := strings.Index(line, "=")
		if idx == -1 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])

		// Strip surrounding quotes
		if (strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"")) ||
			(strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'")) {
			val = val[1 : len(val)-1]
		}

		// Skip empty values or obvious non-secrets
		if val == "" || val == "your_"+strings.ToLower(key)+"_here" {
			continue
		}

		// Only include likely secrets
		upper := strings.ToUpper(key)
		isSecret := false
		for _, kw := range []string{"TOKEN", "SECRET", "PASSWORD", "KEY", "CREDENTIAL", "PRIVATE", "API"} {
			if strings.Contains(upper, kw) {
				isSecret = true
				break
			}
		}
		if !isSecret {
			continue
		}

		results = append(results, FilesystemSecret{
			Name:   key,
			Type:   InferType(key),
			Value:  []byte(val),
			Source: "filesystem",
		})
	}

	return results, nil
}

// Summary returns a human-readable summary of the discovery report.
func (r *DiscoveryReport) Summary() string {
	var lines []string
	ghTotal := 0
	for _, ss := range r.GitHub {
		ghTotal += len(ss)
	}
	flyTotal := 0
	for _, ss := range r.Fly {
		flyTotal += len(ss)
	}

	lines = append(lines, fmt.Sprintf("GitHub Secrets:  %d secret(s) in %d repo(s)", ghTotal, len(r.GitHub)))
	lines = append(lines, fmt.Sprintf("Fly.io Secrets:   %d secret(s) in %d app(s)", flyTotal, len(r.Fly)))
	lines = append(lines, fmt.Sprintf("Filesystem:       %d secret(s) in %d file(s)", len(r.Files), countEnvFiles(r.Files)))
	return strings.Join(lines, "\n")
}

func countEnvFiles(fs []FilesystemSecret) int {
	m := make(map[string]bool)
	for _, s := range fs {
		m[s.Path] = true
	}
	return len(m)
}
