// Package theme — OVAV Unified Theme Engine
//
// Canonical source: .ovav/visual/theme/theme.yaml
//
// This package is the SINGLE SOURCE OF TRUTH for all OVAV visual colors.
// All consumers (cockpit, OpenCode generators, WezTerm, Windows Terminal)
// MUST import from here. No more hardcoded colors anywhere.
//
// Day/night toggle: theme.SetMode("dark"|"light")
//
// Architecture:
//
//   .ovav/visual/theme/theme.yaml
//           |
//           v
//   internal/theme/theme.go  (engine + getters)
//           |
//           +-- cockpit/styles/theme.go  (lipgloss styles)
//           +-- project/sync.go          (OpenCode JSON generator)
//           +-- GenerateWezTermLua()     (WezTerm palette)
//           +-- GenerateWindowsTerminal() (Windows Terminal JSON)
package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Mode represents the active color mode.
type Mode string

const (
	ModeDark  Mode = "dark"
	ModeLight Mode = "light"
)

// ── Theme Data ────────────────────────────────────────────────────────────────

// Theme is the canonical parsed form of theme.yaml.
type Theme struct {
	Schema   string
	Name     string
	Version  string
	Brand    map[string]string
	Semantic map[string]string
	Surfaces map[string]map[string]string // surfaces.dark, surfaces.light
	Syntax   map[string]string
	Diff     map[string]string
	Agents   map[string]AgentColor
	Status   map[string]string
}

// AgentColor represents an agent's visual identity.
type AgentColor struct {
	Color string `yaml:"color"`
	Icon  string `yaml:"icon"`
	Label string `yaml:"label"`
}

// ── Engine ───────────────────────────────────────────────────────────────────

var (
	theme     *Theme
	themeMode Mode = ModeDark
	themeOnce sync.Once
	themeErr  error
)

// Load reads and parses .ovav/visual/theme/theme.yaml exactly once.
// It is called automatically on first access.
func Load(ovavRoot string) (*Theme, error) {
	themeOnce.Do(func() {
		theme, themeErr = loadTheme(ovavRoot)
	})
	return theme, themeErr
}

func loadTheme(ovavRoot string) (*Theme, error) {
	path := filepath.Join(ovavRoot, ".ovav", "visual", "theme", "theme.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("theme: read %s: %w", path, err)
	}

	var t Theme
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("theme: parse %s: %w", path, err)
	}

	if t.Schema != "ovav.visual.theme.v1" {
		return nil, fmt.Errorf("theme: unknown schema %q (expected ovav.visual.theme.v1)", t.Schema)
	}

	return &t, nil
}

// GetMode returns the current color mode (dark or light).
func GetMode() Mode { return themeMode }

// SetMode switches the active color mode. Only "dark" and "light" are valid.
func SetMode(m string) error {
	switch Mode(m) {
	case ModeDark, ModeLight:
		themeMode = Mode(m)
		return nil
	default:
		return fmt.Errorf("theme: unknown mode %q (use 'dark' or 'light')", m)
	}
}

// Toggle switches between dark and light mode.
func Toggle() {
	if themeMode == ModeDark {
		themeMode = ModeLight
	} else {
		themeMode = ModeDark
	}
}

// surfaces returns the surface map for the current mode.
func (t *Theme) surfaces() map[string]string {
	if themeMode == ModeLight {
		if s, ok := t.Surfaces["light"]; ok {
			return s
		}
	}
	return t.Surfaces["dark"]
}

// Brand returns a brand color by key.
func (t *Theme) BrandColor(key string) string {
	if t == nil {
		return "#000000"
	}
	if v, ok := t.Brand[key]; ok {
		return v
	}
	return "#000000"
}

// Semantic returns a semantic color by key.
func (t *Theme) SemanticColor(key string) string {
	if t == nil {
		return "#000000"
	}
	if v, ok := t.Semantic[key]; ok {
		return v
	}
	return "#000000"
}

// Surface returns a surface color by key for the current mode.
func (t *Theme) Surface(key string) string {
	if t == nil {
		return "#000000"
	}
	s := t.surfaces()
	if v, ok := s[key]; ok {
		return v
	}
	return "#000000"
}

// SyntaxColor returns a syntax color by key.
func (t *Theme) SyntaxColor(key string) string {
	if t == nil {
		return "#000000"
	}
	if v, ok := t.Syntax[key]; ok {
		return v
	}
	return "#000000"
}

// DiffColor returns a diff color by key.
func (t *Theme) DiffColor(key string) string {
	if t == nil {
		return "#000000"
	}
	if v, ok := t.Diff[key]; ok {
		return v
	}
	return "#000000"
}

// StatusColor returns a status color by key.
func (t *Theme) StatusColor(key string) string {
	if t == nil {
		return "#000000"
	}
	if v, ok := t.Status[key]; ok {
		return v
	}
	return "#000000"
}

// Agent returns the visual identity for an agent.
func (t *Theme) Agent(key string) AgentColor {
	if t == nil {
		return AgentColor{Color: "#000000", Icon: "?", Label: key}
	}
	if v, ok := t.Agents[key]; ok {
		return v
	}
	return AgentColor{Color: "#000000", Icon: "?", Label: key}
}

// ── Convenience: OVAV Brand Palette ───────────────────────────────────────────

// OVAV brand colors (softened for long sessions).
func (t *Theme) Thavren() string { return t.BrandColor("thavren") }    // teal — Platform Engineering
func (t *Theme) Eidren() string  { return t.BrandColor("eidren") }    // olive green — Research
func (t *Theme) Core() string    { return t.BrandColor("ovav_core") }  // blue-gray — core
func (t *Theme) Accent() string  { return t.BrandColor("ovav_accent") } // rose — accent

// ── Convenience: Semantic Palette ──────────────────────────────────────────────

func (t *Theme) Success() string  { return t.SemanticColor("success") }
func (t *Theme) Error() string    { return t.SemanticColor("error") }
func (t *Theme) Warning() string  { return t.SemanticColor("warning") }
func (t *Theme) Info() string     { return t.SemanticColor("info") }
func (t *Theme) Highlight() string { return t.SemanticColor("highlight") }

// ── Convenience: Surface Palette (current mode) ───────────────────────────────

func (t *Theme) BGRoot()     string { return t.Surface("bg_root") }
func (t *Theme) BGPanel()   string { return t.Surface("bg_panel") }
func (t *Theme) BGElement() string { return t.Surface("bg_element") }
func (t *Theme) BGHover()   string { return t.Surface("bg_hover") }
func (t *Theme) BGSelected() string { return t.Surface("bg_selected") }
func (t *Theme) Border()     string { return t.Surface("border") }
func (t *Theme) BorderActive() string {
	if themeMode == ModeLight {
		return t.Surface("border_active")
	}
	return t.Surface("border_active")
}
func (t *Theme) BorderFocus() string { return t.Surface("border_focus") }
func (t *Theme) TextPrimary()   string { return t.Surface("text_primary") }
func (t *Theme) TextSecondary() string { return t.Surface("text_secondary") }
func (t *Theme) TextMuted()    string { return t.Surface("text_muted") }
func (t *Theme) TextInverse()   string { return t.Surface("text_inverse") }
