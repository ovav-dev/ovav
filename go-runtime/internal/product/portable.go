// Package product implements OVAV Product installation.
//
// OVAV Product is the sellable product that end users install. It provides
// agents, skills, and identity for use with MiMo Code (and later OpenCode,
// Claude Code, OpenAI Codex).
//
// Architecture:
//
//	ovav product install    → copies assets to ~/.local/share/ovav/
//	ovav product launch     → bootstraps CWD/.mimocode/ + launches mimo
//	ovav product bootstrap  → just bootstraps CWD (no launch)
//
// MiMo Code resolves agents from CWD/.mimocode/agents/, NOT from a global
// config directory. The bootstrap step creates symlinks from CWD to the
// canonical install location.
//
// OVAV Systems (internal)          OVAV Product (user-facing)
// ─────────────────────            ──────────────────────────
// Gobernanza canónica              Agents + Skills + Identity
// Governor autónomo                Cockpit TUI
// 80+ validadores                  OWS workflow
// Security defense suite           Gitflow gobernado
// cPanel sync server               Vault, Economy, Doctor
// Install gateway                  Profile, License, Chronos
// Permissions engine               Config, Tools, Status
package product

import (
	"os"
	"path/filepath"
)

// ── Canonical install location ───────────────────────────────────────────────

// ProductDir returns ~/.local/share/ovav/ — the canonical location where
// OVAV Product assets are installed. This is XDG-compliant and separate
// from MiMo Code's config directories.
func ProductDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "ovav"), nil
}

// ── Portable surface definition ──────────────────────────────────────────────

// PortableAsset defines a single asset that gets installed.
type PortableAsset struct {
	Source      string // relative path within OVAV Systems repo
	Target      string // relative path within ProductDir
	Category    string // for manifest grouping
	Description string // human-readable for dry-run
}

// PortableAssets returns the canonical list of assets that go to OVAV Product.
// Agents are now SELECTIVE — only relevant agents are copied per project stack.
// All agents are SANITIZED — zero OVAV Systems memory/context.
func PortableAssets() []PortableAsset {
	return []PortableAsset{
		{
			Source:      "runtimes/mimocode/agents/",
			Target:      "agents/",
			Category:    "agents",
			Description: "Agent definitions — SELECTIVE per project stack, SANITIZED (zero memory)",
		},
		{
			Source:      ".ovav/source/skills/",
			Target:      "skills/",
			Category:    "skills",
			Description: "Canonical skills with workflows — SELECTIVE per project",
		},
		{
			Source:      "OVAV_IDENTITY.md",
			Target:      "OVAV_IDENTITY.md",
			Category:    "identity",
			Description: "Agent identity override — generic for product users",
		},
		{
			Source:      ".ovav/source/configs/model_routing.json",
			Target:      "model_routing.json",
			Category:    "config",
			Description: "Smart model distribution config",
		},
		{
			Source:      "", // generated: compiled from go-runtime/cmd/product_cockpit
			Target:      "product-cockpit",
			Category:    "cli",
			Description: "Product Cockpit TUI — update check, sync alerts, CWD bootstrap",
		},
	}
}

// RestrictedAssets returns assets that are NOT portable (internal only).
func RestrictedAssets() []string {
	return []string{
		"MCP servers (ovav-budget, ovav-evidence, ovav-context)",
		"Validators (80+ F0-F5 governance validators)",
		"Governor (autonomous governance cycle)",
		"Security defense suite (threat detection, response)",
		"Install gateway (plan→manifest→safety→backup→apply)",
		"Permissions engine (OPA/Rego RBAC)",
		"HMAC signing key (verify only in product)",
		"caps.yaml canonical data",
		"cPanel sync server",
		"Git hooks (pre-commit, pre-push, etc.)",
		"external_directory permissions",
		"Convert pipeline (YAML → runtime transform)",
		"OWS policy engine (compliance gates)",
		"SBOM (supply chain verification)",
		"Alerts (security alert lifecycle)",
		"Infrastructure (Cloudflare, OAuth server, bootstrap)",
	}
}

// IsPortable checks if a relative path falls within the portable surface.
func IsPortable(relPath string) bool {
	for _, asset := range PortableAssets() {
		if asset.Source == "" {
			continue
		}
		if relPath == asset.Source || hasPrefix(relPath, asset.Source) {
			return true
		}
	}
	return false
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
