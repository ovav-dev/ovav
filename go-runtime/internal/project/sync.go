// Package project implements OVAV source → OpenCode projection.
//
// Architecture principle: OVAV works natively with well-organized configs
// (in .ovav/source/, .ovav/topology/, .ovav/connector_bus/), then converts
// and projects for the OpenCode CLI surface (clients/opencode/).
//
// The source is canonical and irreplaceable. The projection is derived
// and regenerable. This package provides the sync bridge.
//
// v2.0: Pure Go — zero Python subprocess calls. All projection logic
// is implemented natively using YAML parsing (gopkg.in/yaml.v3) and
// file operations.
package project

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ovav/ovav/internal/convert"
	"gopkg.in/yaml.v3"
)

// Sync runs all projection tools to synchronize the OpenCode surface
// with the OVAV canonical source.
func Sync(root string, verbose bool) error {
	// HARD GATE: prevent Systems/Product cross-contamination.
	if err := ValidateTarget(root); err != nil {
		return err
	}

	totalFailed := 0

	// 1. Agent projection — copy .ovav/source/agents/ → clients/opencode/agents/
	cleaned, created, err := projectAgentsSimple(root, verbose)
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "  ✗ agents: FAILED — %v\n", err)
		}
		totalFailed++
	} else if verbose {
		fmt.Printf("  ✓ agents: %d cleaned, %d projected\n", cleaned, created)
	}

	// 2. ConnectorBus-based skills + personnel projection
	skillsSynced, agentsSynced, err := projectFromConnectorBus(root, verbose)
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "  ✗ connector_bus: FAILED — %v\n", err)
		}
		totalFailed++
	} else if verbose {
		fmt.Printf("  ✓ connector_bus: %d skills synced, %d agents synced\n", skillsSynced, agentsSynced)
	}

	// 3. Visual projection — theme JSON, monitor plugin, TUI, WezTerm
	visualCount, err := projectVisual(root, verbose)
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "  ✗ visual: FAILED — %v\n", err)
		}
		totalFailed++
	} else if verbose {
		fmt.Printf("  ✓ visual: %d artifacts projected\n", visualCount)
	}

	// 4. MiMo Code projection — skills + plugins + workflows → .mimocode/
	mimocodeCount, err := projectToMimocode(root, verbose)
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "  ✗ mimocode: FAILED — %v\n", err)
		}
		totalFailed++
	} else if verbose {
		fmt.Printf("  ✓ mimocode: %d artifacts projected\n", mimocodeCount)
	}

	// 5. Harness AGENTS projection — opencode_AGENTS.md + mimocode_AGENTS.md
	agentsCount, err := projectHarnessAgents(root, verbose)
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "  ✗ harness_agents: FAILED — %v\n", err)
		}
		totalFailed++
	} else if verbose {
		fmt.Printf("  ✓ harness_agents: %d harness AGENTS files projected\n", agentsCount)
	}

	// 6. OpenCode config generation — canonical YAML → opencode.json (OVERWRITE)
	//    OVAV es la fuente autoritativa. Genera el config completo desde cero.
	if err := convert.GenerateOpenCodeConfig(root); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "  ✗ config: FAILED — %v\n", err)
		}
		totalFailed++
	} else if verbose {
		fmt.Println("  ✓ config: generated from .ovav/source/opencode/config.yaml")
	}

	// 7. Validate generated config
	configIssues, configErr := convert.ValidateOpenCodeConfig(root)
	if configErr != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "  ⚠ config validation: %v\n", configErr)
		}
	} else {
		criticals := 0
		for _, issue := range configIssues {
			if issue.Severity == "critical" {
				criticals++
				fmt.Fprintf(os.Stderr, "  ✗ config: %s — %s\n", issue.Field, issue.Message)
			}
		}
		if criticals > 0 {
			totalFailed++
		} else if verbose {
			fmt.Println("  ✓ config: valid")
		}
	}

	if totalFailed > 0 {
		return fmt.Errorf("%d projector(s) failed", totalFailed)
	}
	return nil
}

// ── Projector 1: Convert Engine Agent Generation ────────────────────────────

// projectAgentsSimple generates CLI agent files from canonical OVAV YAML
// using the convert engine — pure Go, no raw file copies.
//
//	Canonical source: go-runtime/internal/agents/{areas,leads,teams}/*.yaml
//	CLI targets:      go-runtime/internal/runtimes/{opencode,claude-code,cursor}/agents/*.md
//
// Replaces: tools/agents/project_opencode.py (removed python3 dependency)
func projectAgentsSimple(root string, verbose bool) (cleaned int, created int, err error) {
	canonicalRoot := filepath.Join(root, "go-runtime", "internal", "agents")
	targets := convert.AvailableTargets()

	// Clean old generated files across all target output directories
	for _, target := range targets {
		converter, convErr := convert.GetConverter(target)
		if convErr != nil {
			return 0, 0, fmt.Errorf("get converter for %s: %w", target, convErr)
		}
		outputDir := filepath.Join(root, converter.OutputDir())
		ext := converter.FileExtension()
		for _, prefix := range []string{"area-", "lead-", "team-"} {
			pattern := filepath.Join(outputDir, prefix+"*"+ext)
			matches, _ := filepath.Glob(pattern)
			for _, m := range matches {
				if rmErr := os.Remove(m); rmErr != nil {
					return 0, 0, fmt.Errorf("clean %s: %w", m, rmErr)
				}
				cleaned++
			}
		}
	}

	// Generate for all targets (OpenCode, Claude Code, Cursor)
	for _, target := range targets {
		if genErr := convert.GenerateAll(canonicalRoot, target, root); genErr != nil {
			return 0, 0, fmt.Errorf("generate %s: %w", target, genErr)
		}
	}

	// Count created agents from canonical source
	areas, leads, teams, loadErr := convert.LoadCanonicalAgents(canonicalRoot)
	if loadErr != nil {
		return cleaned, 0, fmt.Errorf("count agents: %w", loadErr)
	}
	created = (len(areas) + len(leads) + len(teams)) * len(targets)

	return cleaned, created, nil
}

// ── Projector 2: ConnectorBus Skills + Personnel ─────────────────────────

// connectorsSkillsConfig mirrors connectors/skills.yaml structure.
type connectorsSkillsConfig struct {
	Version    string                    `yaml:"version"`
	SlotType   string                    `yaml:"slot_type"`
	Components map[string]skillComponent `yaml:"components"`
}

type skillComponent struct {
	SourceDir    string   `yaml:"source_dir"`
	OwnerProfile string   `yaml:"owner_profile"`
	RiskLevel    string   `yaml:"risk_level"`
	Labels       []string `yaml:"labels"`
}

// connectorsPersonnelConfig mirrors connectors/personnel.yaml structure.
type connectorsPersonnelConfig struct {
	Version    string                        `yaml:"version"`
	SlotType   string                        `yaml:"slot_type"`
	Components map[string]personnelComponent `yaml:"components"`
}

type personnelComponent struct {
	Role        string   `yaml:"role"`
	Area        string   `yaml:"area"`
	Type        string   `yaml:"type"`
	Artifacts   []string `yaml:"artifacts"`
	Permissions string   `yaml:"permissions"`
	Active      bool     `yaml:"active"`
	Labels      []string `yaml:"labels"`
}

// projectFromConnectorBus reads connectors/*.yaml and projects skills
// and personnel (leads + team members) to the OpenCode surface.
//
// Replaces: tools/agent_runtime/projection_engine.py
func projectFromConnectorBus(root string, verbose bool) (skillsSynced int, agentsSynced int, err error) {
	// ── Skills ──────────────────────────────────────────────────────────
	skillsPath := filepath.Join(root, ".ovav", "connector_bus.legacy", "connectors", "skills.yaml")
	skillsData, err := os.ReadFile(skillsPath)
	if err != nil && !os.IsNotExist(err) {
		return 0, 0, fmt.Errorf("read skills.yaml: %w", err)
	}

	if err == nil {
		var skillsCfg connectorsSkillsConfig
		if err := yaml.Unmarshal(skillsData, &skillsCfg); err != nil {
			return 0, 0, fmt.Errorf("parse skills.yaml: %w", err)
		}

		skillsTarget := filepath.Join(root, ".opencode", "skills")
		for _, comp := range skillsCfg.Components {
			if comp.SourceDir == "" {
				continue
			}
			srcDir := filepath.Join(root, comp.SourceDir)
			skillFile := filepath.Join(srcDir, "SKILL.md")
			refsDir := filepath.Join(srcDir, "references")

			// Normalize: use source_dir basename as target dir name
			normalizedName := filepath.Base(comp.SourceDir)

			targetDir := filepath.Join(skillsTarget, normalizedName)
			targetSkill := filepath.Join(targetDir, "SKILL.md")

			// Skip if content unchanged
			if filesEqual(skillFile, targetSkill) {
				if verbose {
					fmt.Printf("    · %s: up to date\n", normalizedName)
				}
				continue
			}

			if err := os.MkdirAll(targetDir, 0755); err != nil {
				return 0, 0, fmt.Errorf("mkdir skill %s: %w", normalizedName, err)
			}
			if err := copyFile(skillFile, targetSkill); err != nil {
				return 0, 0, fmt.Errorf("copy skill %s: %w", normalizedName, err)
			}

			// Sync references/ if exists
			if info, err := os.Stat(refsDir); err == nil && info.IsDir() {
				targetRefs := filepath.Join(targetDir, "references")
				if err := os.RemoveAll(targetRefs); err != nil {
					return 0, 0, fmt.Errorf("rm refs %s: %w", normalizedName, err)
				}
				if err := copyDir(refsDir, targetRefs); err != nil {
					return 0, 0, fmt.Errorf("copy refs %s: %w", normalizedName, err)
				}
			}

			skillsSynced++
			if verbose {
				fmt.Printf("    ✅ %s: synced\n", normalizedName)
			}
		}
	}

	// ── Personnel (leads + team) ────────────────────────────────────────
	personnelPath := filepath.Join(root, ".ovav", "connector_bus.legacy", "connectors", "personnel.yaml")
	personnelData, err := os.ReadFile(personnelPath)
	if err != nil && !os.IsNotExist(err) {
		return skillsSynced, 0, fmt.Errorf("read personnel.yaml: %w", err)
	}

	if err == nil {
		var persCfg connectorsPersonnelConfig
		if err := yaml.Unmarshal(personnelData, &persCfg); err != nil {
			return skillsSynced, 0, fmt.Errorf("parse personnel.yaml: %w", err)
		}

		agentsTarget := filepath.Join(root, ".opencode", "agents")
		seen := make(map[string]bool) // dedup targets

		for name, comp := range persCfg.Components {
			if !comp.Active {
				continue
			}
			for _, artifact := range comp.Artifacts {
				srcPath := filepath.Join(root, artifact)
				if _, err := os.Stat(srcPath); os.IsNotExist(err) {
					continue // source artifact may not be checked out, skip
				}

				// Derive target filename from artifact path
				var targetName string
				switch {
				case strings.Contains(artifact, "/leads/"):
					targetName = "lead-" + name + ".md"
				case strings.Contains(artifact, "/teams/"):
					targetName = "team-" + name + ".md"
				default:
					targetName = filepath.Base(artifact)
				}

				if seen[targetName] {
					continue
				}
				seen[targetName] = true

				targetPath := filepath.Join(agentsTarget, targetName)

				// Skip if content unchanged
				if filesEqual(srcPath, targetPath) {
					if verbose {
						fmt.Printf("    · %s (%s): up to date\n", name, targetName)
					}
					continue
				}

				if err := os.MkdirAll(agentsTarget, 0755); err != nil {
					return skillsSynced, agentsSynced, fmt.Errorf("mkdir agents: %w", err)
				}
				if err := copyFile(srcPath, targetPath); err != nil {
					return skillsSynced, agentsSynced, fmt.Errorf("copy agent %s: %w", name, err)
				}
				agentsSynced++
				if verbose {
					fmt.Printf("    ✅ %s: synced → %s\n", name, targetName)
				}
			}
		}

		// ── Area files ──────────────────────────────────────────────────
		areasSource := filepath.Join(root, ".ovav", "source", "agents", "areas")
		if info, err := os.Stat(areasSource); err == nil && info.IsDir() {
			entries, err := os.ReadDir(areasSource)
			if err != nil {
				return skillsSynced, agentsSynced, fmt.Errorf("read areas: %w", err)
			}
			for _, entry := range entries {
				if entry.IsDir() || !strings.HasPrefix(entry.Name(), "area-") || !strings.HasSuffix(entry.Name(), ".md") {
					continue
				}
				srcPath := filepath.Join(areasSource, entry.Name())
				targetPath := filepath.Join(agentsTarget, entry.Name())

				if filesEqual(srcPath, targetPath) {
					continue
				}
				if err := os.MkdirAll(agentsTarget, 0755); err != nil {
					return skillsSynced, agentsSynced, fmt.Errorf("mkdir agents: %w", err)
				}
				if err := copyFile(srcPath, targetPath); err != nil {
					return skillsSynced, agentsSynced, fmt.Errorf("copy area %s: %w", entry.Name(), err)
				}
				agentsSynced++
				if verbose {
					fmt.Printf("    ✅ %s: synced\n", entry.Name())
				}
			}
		}
	}

	return skillsSynced, agentsSynced, nil
}

// ── Projector 3: Visual Assets ────────────────────────────────────────────

// themeRaw mirrors the structure of .ovav/visual/theme/theme.yaml
// for generating .opencode/themes/ovav.json.
type themeRaw struct {
	Schema   string                       `yaml:"schema"`
	Name     string                       `yaml:"name"`
	Version  string                       `yaml:"version"`
	Brand    map[string]string            `yaml:"brand"`
	Semantic map[string]string            `yaml:"semantic"`
	Surfaces map[string]map[string]string `yaml:"surfaces"`
	Syntax   map[string]string            `yaml:"syntax"`
	Diff     map[string]string            `yaml:"diff"`
	Agents   map[string]agentVisual       `yaml:"agents"`
	Status   map[string]string            `yaml:"status"`
	Budget   struct {
		Thresholds map[string]struct {
			Max   float64 `yaml:"max"`
			Color string  `yaml:"color"`
			Icon  string  `yaml:"icon"`
		} `yaml:"thresholds"`
	} `yaml:"budget"`
}

type agentVisual struct {
	Color string `yaml:"color"`
	Icon  string `yaml:"icon"`
	Label string `yaml:"label"`
}

// monitoringRaw mirrors .ovav/visual/monitoring/monitoring.yaml structure.
type monitoringRaw struct {
	Schema      string                  `yaml:"schema"`
	Version     string                  `yaml:"version"`
	Description string                  `yaml:"description"`
	Watchers    map[string]watcherEntry `yaml:"watchers"`
	Alerts      map[string]alertEntry   `yaml:"alerts"`
}

type watcherEntry struct {
	Description string         `yaml:"description"`
	Source      string         `yaml:"source"`
	Display     watcherDisplay `yaml:"display"`
}

type watcherDisplay struct {
	Format          string `yaml:"format"`
	UpdateFrequency string `yaml:"update_frequency"`
}

type alertEntry struct {
	Trigger    string `yaml:"trigger"`
	Severity   string `yaml:"severity"`
	Toast      string `yaml:"toast"`
	DurationMs int    `yaml:"duration_ms"`
}

// projectVisual generates all visual artifacts for OpenCode:
// theme JSON, monitor JS plugin, TUI config, WezTerm sync, plugin registry.
//
// Replaces: tools/visual/project_opencode_visual.py
func projectVisual(root string, verbose bool) (count int, err error) {
	// ── 1. Theme JSON ──────────────────────────────────────────────────
	themePath := filepath.Join(root, ".ovav", "visual", "theme", "theme.yaml")
	themeData, themeErr := os.ReadFile(themePath)
	if themeErr != nil {
		return 0, fmt.Errorf("read theme.yaml: %w", themeErr)
	}

	var theme themeRaw
	if err := yaml.Unmarshal(themeData, &theme); err != nil {
		return 0, fmt.Errorf("parse theme.yaml: %w", err)
	}

	themeJSON := generateOpenCodeTheme(&theme)
	themeTarget := filepath.Join(root, ".opencode", "themes", "ovav.json")
	if err := os.MkdirAll(filepath.Dir(themeTarget), 0755); err != nil {
		return 0, fmt.Errorf("mkdir themes: %w", err)
	}
	themeBytes, _ := json.MarshalIndent(themeJSON, "", "  ")
	if err := os.WriteFile(themeTarget, append(themeBytes, '\n'), 0644); err != nil {
		return 0, fmt.Errorf("write ovav.json: %w", err)
	}
	count++

	// ── 2. Monitor Plugin JS ───────────────────────────────────────────
	monPath := filepath.Join(root, ".ovav", "visual", "monitoring", "monitoring.yaml")
	monData, monErr := os.ReadFile(monPath)
	if monErr != nil {
		return count, fmt.Errorf("read monitoring.yaml: %w", monErr)
	}

	var mon monitoringRaw
	if err := yaml.Unmarshal(monData, &mon); err != nil {
		return count, fmt.Errorf("parse monitoring.yaml: %w", err)
	}

	pluginJS := generateOpenCodePlugin(&mon)
	pluginTarget := filepath.Join(root, ".opencode", "plugins", "ovav-monitor.js")
	if err := os.MkdirAll(filepath.Dir(pluginTarget), 0755); err != nil {
		return count, fmt.Errorf("mkdir plugins: %w", err)
	}
	if err := os.WriteFile(pluginTarget, []byte(pluginJS), 0644); err != nil {
		return count, fmt.Errorf("write plugin.js: %w", err)
	}
	count++

	// ── 3. TUI Config ──────────────────────────────────────────────────
	tuiConfig := map[string]string{
		"$schema": "https://opencode.ai/tui.json",
		"theme":   "ovav",
	}
	tuiPath := filepath.Join(root, "tui.json")
	tuiBytes, _ := json.MarshalIndent(tuiConfig, "", "  ")
	if err := os.WriteFile(tuiPath, append(tuiBytes, '\n'), 0644); err != nil {
		return count, fmt.Errorf("write tui.json: %w", err)
	}
	count++

	// Plugin registry now handled by convert.GenerateOpenCodeConfig()
	// (canonical source: .ovav/source/opencode/config.yaml → opencode.json)

	// ── 5. WezTerm — validate canonical config and sync deploy target ──
	wezCanonical := filepath.Join(root, ".ovav", "visual", "wezterm", "config.lua")
	wezDeploy := filepath.Join(root, "config", "wezterm", "wezterm.lua")
	if _, err := os.Stat(wezCanonical); err == nil {
		if err := os.MkdirAll(filepath.Dir(wezDeploy), 0755); err != nil {
			return count, fmt.Errorf("mkdir wezterm deploy: %w", err)
		}
		if !filesEqual(wezCanonical, wezDeploy) {
			if err := copyFile(wezCanonical, wezDeploy); err != nil {
				return count, fmt.Errorf("copy wezterm config: %w", err)
			}
			if verbose {
				fmt.Println("    ✅ wezterm: deploy synced")
			}
		}
		count++
	}

	if verbose {
		fmt.Printf("    ✅ theme: %s → .opencode/themes/ovav.json\n", themePath)
		fmt.Printf("    ✅ monitor: %s → .opencode/plugins/ovav-monitor.js\n", monPath)
		fmt.Printf("    ✅ tui: tui.json\n")
	}

	return count, nil
}

// generateOpenCodeTheme converts OVAV theme.yaml to one adaptive OpenCode theme.
func generateOpenCodeTheme(t *themeRaw) map[string]interface{} {
	dark := t.Surfaces["dark"]
	light := t.Surfaces["light"]
	syntax := t.Syntax
	pair := func(darkValue, lightValue string) map[string]string {
		return map[string]string{"dark": darkValue, "light": lightValue}
	}

	return map[string]interface{}{
		"$schema": "https://opencode.ai/theme.json",
		"defs": map[string]string{
			"syntax_keyword": syntax["keyword"],
			"syntax_string":  syntax["string"],
			"syntax_number":  syntax["number"],
		},
		"theme": map[string]interface{}{
			"primary":              pair(dark["primary"], light["primary"]),
			"secondary":            pair(dark["secondary"], light["secondary"]),
			"accent":               pair(dark["accent"], light["accent"]),
			"error":                pair(dark["error"], light["error"]),
			"warning":              pair(dark["warning"], light["warning"]),
			"success":              pair(dark["success"], light["success"]),
			"info":                 pair(dark["info"], light["info"]),
			"text":                 pair(dark["text_primary"], light["text_primary"]),
			"textMuted":            pair(dark["text_muted"], light["text_muted"]),
			"textSecondary":        pair(dark["text_secondary"], light["text_secondary"]),
			"background":           pair(dark["bg_root"], light["bg_root"]),
			"backgroundPanel":      pair(dark["bg_panel"], light["bg_panel"]),
			"backgroundElement":    pair(dark["bg_element"], light["bg_element"]),
			"border":               pair(dark["border"], light["border"]),
			"borderActive":         pair(dark["border_active"], light["border_active"]),
			"borderSubtle":         pair(dark["bg_element"], light["bg_element"]),
			"diffAdded":            pair(dark["diff_added"], light["diff_added"]),
			"diffRemoved":          pair(dark["diff_removed"], light["diff_removed"]),
			"diffContext":          pair(dark["diff_context"], light["diff_context"]),
			"diffHunkHeader":       pair(dark["diff_hunk"], light["diff_hunk"]),
			"diffHighlightAdded":   pair(dark["diff_added"], light["diff_added"]),
			"diffHighlightRemoved": pair(dark["diff_removed"], light["diff_removed"]),
			"diffAddedBg":          pair(dark["diff_added_bg"], light["diff_added_bg"]),
			"diffRemovedBg":        pair(dark["diff_removed_bg"], light["diff_removed_bg"]),
			"diffContextBg":        pair(dark["bg_panel"], light["bg_panel"]),
			"markdownText":         pair(dark["text_primary"], light["text_primary"]),
			"markdownHeading":      pair(dark["primary"], light["primary"]),
			"markdownLink":         pair(dark["info"], light["info"]),
			"markdownCode":         pair(dark["success"], light["success"]),
			"syntaxComment":        pair(dark["text_muted"], light["text_muted"]),
			"syntaxKeyword":        pair(dark["accent"], light["accent"]),
			"syntaxFunction":       pair(dark["primary"], light["primary"]),
			"syntaxVariable":       pair(dark["text_primary"], light["text_primary"]),
			"syntaxString":         pair(dark["secondary"], light["secondary"]),
			"syntaxNumber":         pair(dark["accent"], light["accent"]),
		},
	}
}

// generateOpenCodePlugin generates the OpenCode monitor plugin JS from
// OVAV monitoring.yaml rules. Mirrors monitor_engine.py:generate_opencode_plugin.
func generateOpenCodePlugin(m *monitoringRaw) string {
	var b strings.Builder
	// bp appends a raw string when no args; with args, it formats (avoiding vet
	// "non-constant format string" errors when the string contains literal '%'
	// characters from JS templates).
	bp := func(s string, args ...interface{}) {
		if len(args) > 0 {
			fmt.Fprintf(&b, s, args...)
		} else {
			b.WriteString(s)
		}
	}
	// raw appends a literal string (no format interpretation) — use this when
	// the content contains '%' characters from JS template literals.
	raw := func(s string) { b.WriteString(s) }

	bp("/*\n")
	bp(" * OVAV Monitor Plugin — Generated by go-runtime/internal/project/sync.go\n")
	bp(" * DO NOT EDIT DIRECTLY. Source: .ovav/visual/monitoring/monitoring.yaml\n")
	bp(" */\n\n")

	bp("export const OVAVMonitor = async ({ client, $, project, directory }) => {\n")
	bp("  // State\n")
	bp("  const state = {\n")
	bp("    currentAgent: null,\n")
	bp("    currentModel: null,\n")
	bp("    totalTokens: 0,\n")
	bp("    budgetPercent: 0,\n")
	bp(`    budgetRemaining: "0",` + "\n")
	bp("    ovavMemoryActive: false,\n")
	bp("    sessionActive: false,\n")
	bp("    sessionStart: null,\n")
	bp("    idleSince: null,\n")
	bp("  };\n\n")

	bp("  const toast = (message, duration) => {\n")
	bp("    try { client.tui.toast.show({ message, duration }); } catch (e) {}\n")
	bp("  };\n\n")

	// Budget helper
	bp("  const checkBudget = () => {\n")
	bp("    const percent = Math.min(100, Math.round((state.totalTokens / 1000000) * 100)) / 100;\n")
	bp("    state.budgetPercent = Math.round(percent);\n")
	bp("    if (percent > 90 && !state._warnedCritical) {\n")
	bp("      state._warnedCritical = true;\n")
	raw("      toast(" + jsTemplate("🔴 PRESUPUESTO CRÍTICO: ${state.budgetPercent}%") + ", 8000);\n")
	bp("    } else if (percent > 75 && !state._warnedWarning) {\n")
	bp("      state._warnedWarning = true;\n")
	raw("      toast(" + jsTemplate("⚠️ Presupuesto al ${state.budgetPercent}%") + ", 5000);\n")
	bp("    } else if (percent <= 75) {\n")
	bp("      state._warnedWarning = false;\n")
	bp("      state._warnedCritical = false;\n")
	bp("    }\n")
	bp("  };\n\n")

	bp("  // [OVAV] Monitor plugin initialized — silent (see .ovav/logs/monitor.log)\n")

	// Main event handler
	bp("  return {\n")
	bp("    event: async ({ event }) => {\n")
	bp("      switch (event.type) {\n")
	bp("        case 'session.status': {\n")
	bp("          const status = event.properties;\n")
	bp("          if (!status) break;\n\n")
	bp("          if (status.agent && status.agent !== state.currentAgent) {\n")
	bp("            const prev = state.currentAgent;\n")
	bp("            state.currentAgent = status.agent;\n")
	bp("            if (prev) {\n")
	raw(buildAlertSwitchJS("agent_switch", m.Alerts))
	bp("            }\n")
	bp("          }\n\n")
	bp("          if (status.model && status.model !== state.currentModel) {\n")
	bp("            const prev = state.currentModel;\n")
	bp("            state.currentModel = status.model;\n")
	bp("            if (prev) {\n")
	raw(buildAlertSwitchJS("model_switch", m.Alerts))
	bp("            }\n")
	bp("          }\n\n")
	bp("          if (status.usage) {\n")
	bp("            state.totalTokens = status.usage.total || status.usage.input + status.usage.output || 0;\n")
	bp("            checkBudget();\n")
	bp("          }\n")
	bp("          break;\n")
	bp("        }\n")
	bp("        case 'session.idle':\n")
	bp("          state.sessionActive = false;\n")
	bp("          state.idleSince = Date.now();\n")
	bp("          break;\n")
	bp("        case 'session.created':\n")
	bp("          state.sessionActive = true;\n")
	bp("          state.sessionStart = Date.now();\n")
	bp("          state.idleSince = null;\n")
	bp("          break;\n")
	bp("        case 'todo.updated':\n")
	bp("          break;\n")
	bp("      }\n")
	bp("    },\n")
	bp("    tool: {\n")
	bp("      ovav_monitor: {\n")
	bp("        description: 'Get OVAV monitor status',\n")
	bp("        execute: async () => {\n")
	bp("          const elapsed = state.sessionStart ? Math.floor((Date.now() - state.sessionStart) / 1000) : 0;\n")
	bp("          const minutes = Math.floor(elapsed / 60);\n")
	raw("          const seconds = elapsed % 60;\n")
	bp("          return JSON.stringify({\n")
	bp("            agent: state.currentAgent || 'unknown',\n")
	bp("            model: state.currentModel || 'unknown',\n")
	bp("            tokens: state.totalTokens,\n")
	bp("            budgetPercent: state.budgetPercent,\n")
	bp("            budgetRemaining: state.budgetRemaining,\n")
	bp("            memoryActive: state.ovavMemoryActive,\n")
	bp("            sessionElapsed: `$'{'minutes}m $'{'seconds}s`,\n")
	bp("            sessionActive: state.sessionActive,\n")
	bp("          }, null, 2);\n")
	bp("        },\n")
	bp("      },\n")
	bp("    },\n")
	bp("  };\n")
	bp("};\n")

	return b.String()
}

// jsTemplate wraps a string in JavaScript backtick template literal syntax.
// Avoids raw Go backticks that would conflict with JS template literals.
func jsTemplate(s string) string {
	return "`" + s + "`"
}

// buildAlertSwitchJS generates the toast() call for a specific alert.
func buildAlertSwitchJS(name string, alerts map[string]alertEntry) string {
	alert, ok := alerts[name]
	if !ok {
		return "              // " + name + ": no config\n"
	}
	msg := alert.Toast
	msg = strings.ReplaceAll(msg, "{from}", "${prev}")
	if strings.Contains(name, "agent") {
		msg = strings.ReplaceAll(msg, "{to}", "${state.currentAgent}")
	} else {
		msg = strings.ReplaceAll(msg, "{to}", "${state.currentModel}")
	}
	duration := alert.DurationMs
	if duration <= 0 {
		duration = 3000
	}
	return fmt.Sprintf("              toast(%s, %d);\n", jsTemplate(msg), duration)
}

// syncPluginRegistry ensures opencode.json plugin[] references local OVAV plugins.
func syncPluginRegistry(root string) (int, error) {
	opencodeJSON := filepath.Join(root, "opencode.json")
	data, err := os.ReadFile(opencodeJSON)
	if err != nil {
		return 0, err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return 0, err
	}

	ovavPlugins := []string{
		// MiMo Code built-in plugins (ovav_monitor, ovav_status) are NOT listed here.
		// They are provided by MiMo Code natively. Only list plugins that ADD value.
	}

	// Verify plugin files exist
	var existing []string
	for _, p := range ovavPlugins {
		if _, err := os.Stat(filepath.Join(root, p)); err == nil {
			existing = append(existing, p)
		}
	}

	pluginsRaw, ok := config["plugin"]
	if !ok {
		if len(existing) > 0 {
			config["plugin"] = existing
		}
	} else {
		plugins, ok := pluginsRaw.([]interface{})
		if !ok {
			return 0, fmt.Errorf("plugin field is not an array")
		}

		// Filter out @ovav/ npm references and non-local plugins
		var cleaned []interface{}
		for _, p := range plugins {
			pStr, ok := p.(string)
			if !ok {
				cleaned = append(cleaned, p)
				continue
			}
			if strings.HasPrefix(pStr, "@ovav/") || pStr == "@ovav/opencode-tui" {
				continue
			}
			cleaned = append(cleaned, p)
		}

		// Add local OVAV plugins if not present
		for _, p := range existing {
			found := false
			for _, c := range cleaned {
				if s, ok := c.(string); ok && s == p {
					found = true
					break
				}
			}
			if !found {
				cleaned = append(cleaned, p)
			}
		}
		config["plugin"] = cleaned
	}

	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return 0, err
	}
	if err := os.WriteFile(opencodeJSON, append(out, '\n'), 0644); err != nil {
		return 0, err
	}
	return len(existing), nil
}

// ── Projector 4b: Harness AGENTS Projection ──────────────────────────────

// projectHarnessAgents projects harness-specific AGENTS.md files from OVAV
// canonical sources to the repository root.
//
// opencode_AGENTS.md:  Source → .ovav/source/opencode/AGENTS.md
//
//	Target → {root}/opencode_AGENTS.md
//
// mimocode_AGENTS.md:  Source → .ovav/source/mimocode/AGENTS.md
//
//	Target → {root}/mimocode_AGENTS.md
//
// crush_AGENTS.md:     Source → .ovav/source/crush/AGENTS.md
//
//	Target → {root}/crush_AGENTS.md
//
// These are the harness-specific instruction overlays that supersede the
// generic AGENTS.md for OpenCode, MiMoCode, and Crush users respectively.
func projectHarnessAgents(root string, verbose bool) (count int, err error) {
	agents := []struct {
		sourceRel string // relative to .ovav/source/
		target    string // absolute target path
	}{
		{filepath.Join("opencode", "AGENTS.md"), filepath.Join(root, "opencode_AGENTS.md")},
		{filepath.Join("mimocode", "AGENTS.md"), filepath.Join(root, "mimocode_AGENTS.md")},
		{filepath.Join("crush", "AGENTS.md"), filepath.Join(root, "crush_AGENTS.md")},
	}

	for _, a := range agents {
		src := filepath.Join(root, ".ovav", "source", a.sourceRel)
		if _, statErr := os.Stat(src); os.IsNotExist(statErr) {
			if verbose {
				fmt.Printf("    · %s: source not found, skipping\n", filepath.Base(a.target))
			}
			continue
		}
		if filesEqual(src, a.target) {
			if verbose {
				fmt.Printf("    · %s: up to date\n", filepath.Base(a.target))
			}
			continue
		}
		if copyErr := copyFile(src, a.target); copyErr != nil {
			return count, fmt.Errorf("copy %s: %w", a.target, copyErr)
		}
		count++
		if verbose {
			fmt.Printf("    ✅ %s: projected\n", filepath.Base(a.target))
		}
	}
	return count, nil
}

// ── Projector 4: MiMo Code Projection ────────────────────────────────────

// projectToMimocode projects skills, plugins, and workflows from OVAV canonical
// sources to .mimocode/ — the directory MiMo Code reads from.
//
// Skills:    .ovav/source/skills/*/SKILL.md → .mimocode/skills/*/SKILL.md
// Plugins:   .ovav/source/plugins/mimocode/*/*.js → .mimocode/plugins/*.js
// Workflows: .ovav/source/workflows/*.js → .mimocode/workflows/*.js
//
// This mirrors the OpenCode projection but targets MiMo Code's directory layout.
func projectToMimocode(root string, verbose bool) (count int, err error) {
	mimocodeDir := filepath.Join(root, ".mimocode")

	// ── Skills ──────────────────────────────────────────────────────────
	skillsSource := filepath.Join(root, ".ovav", "source", "skills")
	skillsTarget := filepath.Join(mimocodeDir, "skills")
	if info, err := os.Stat(skillsSource); err == nil && info.IsDir() {
		entries, err := os.ReadDir(skillsSource)
		if err != nil {
			return count, fmt.Errorf("read mimocode skills source: %w", err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			skillFile := filepath.Join(skillsSource, entry.Name(), "SKILL.md")
			if _, err := os.Stat(skillFile); os.IsNotExist(err) {
				continue
			}
			targetDir := filepath.Join(skillsTarget, entry.Name())
			targetSkill := filepath.Join(targetDir, "SKILL.md")

			if filesEqual(skillFile, targetSkill) {
				continue
			}
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				return count, fmt.Errorf("mkdir mimocode skill %s: %w", entry.Name(), err)
			}
			if err := copyFile(skillFile, targetSkill); err != nil {
				return count, fmt.Errorf("copy mimocode skill %s: %w", entry.Name(), err)
			}

			// Sync references/ if exists
			refsDir := filepath.Join(skillsSource, entry.Name(), "references")
			if info, err := os.Stat(refsDir); err == nil && info.IsDir() {
				targetRefs := filepath.Join(targetDir, "references")
				os.RemoveAll(targetRefs)
				if err := copyDir(refsDir, targetRefs); err != nil {
					return count, fmt.Errorf("copy mimocode refs %s: %w", entry.Name(), err)
				}
			}
			count++
			if verbose {
				fmt.Printf("    ✅ mimocode skill: %s\n", entry.Name())
			}
		}
	}

	// ── Plugins (MiMo Code native) ────────────────────────────────────
	pluginsSource := filepath.Join(root, ".ovav", "source", "plugins", "mimocode")
	pluginsTarget := filepath.Join(mimocodeDir, "plugins")
	if info, err := os.Stat(pluginsSource); err == nil && info.IsDir() {
		entries, err := os.ReadDir(pluginsSource)
		if err != nil {
			return count, fmt.Errorf("read mimocode plugins source: %w", err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			// Each subdirectory contains a .js plugin file
			pluginFiles, _ := filepath.Glob(filepath.Join(pluginsSource, entry.Name(), "*.js"))
			for _, pluginFile := range pluginFiles {
				targetFile := filepath.Join(pluginsTarget, filepath.Base(pluginFile))
				if filesEqual(pluginFile, targetFile) {
					continue
				}
				if err := os.MkdirAll(pluginsTarget, 0755); err != nil {
					return count, fmt.Errorf("mkdir mimocode plugins: %w", err)
				}
				if err := copyFile(pluginFile, targetFile); err != nil {
					return count, fmt.Errorf("copy mimocode plugin %s: %w", filepath.Base(pluginFile), err)
				}
				count++
				if verbose {
					fmt.Printf("    ✅ mimocode plugin: %s\n", filepath.Base(pluginFile))
				}
			}
		}
	}

	// ── Workflows ──────────────────────────────────────────────────────
	wfSource := filepath.Join(root, ".ovav", "source", "workflows")
	wfTarget := filepath.Join(mimocodeDir, "workflows")
	if info, err := os.Stat(wfSource); err == nil && info.IsDir() {
		entries, err := os.ReadDir(wfSource)
		if err != nil {
			return count, fmt.Errorf("read mimocode workflows source: %w", err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".js") {
				continue
			}
			srcFile := filepath.Join(wfSource, entry.Name())
			targetFile := filepath.Join(wfTarget, entry.Name())
			if filesEqual(srcFile, targetFile) {
				continue
			}
			if err := os.MkdirAll(wfTarget, 0755); err != nil {
				return count, fmt.Errorf("mkdir mimocode workflows: %w", err)
			}
			if err := copyFile(srcFile, targetFile); err != nil {
				return count, fmt.Errorf("copy mimocode workflow %s: %w", entry.Name(), err)
			}
			count++
			if verbose {
				fmt.Printf("    ✅ mimocode workflow: %s\n", entry.Name())
			}
		}
	}

	return count, nil
}

// ── Helpers ──────────────────────────────────────────────────────────────

// copyFile copies a file from src to dst, preserving permissions.
func copyFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	info, err := s.Stat()
	if err != nil {
		return err
	}

	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()

	if _, err := io.Copy(d, s); err != nil {
		return err
	}
	return os.Chmod(dst, info.Mode())
}

// copyDir recursively copies a directory from src to dst.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target)
	})
}

// filesEqual compares two files by content. Returns true if both exist
// and have identical content.
func filesEqual(a, b string) bool {
	da, err := os.ReadFile(a)
	if err != nil {
		return false
	}
	db, err := os.ReadFile(b)
	if err != nil {
		return false
	}
	return string(da) == string(db)
}

// ── Exported Wrappers for Cockpit TUI ─────────────────────────────

// SyncAgents runs only the agent projection step.
func SyncAgents(root string, verbose bool) (cleaned int, created int, err error) {
	return projectAgentsSimple(root, verbose)
}

// SyncConnectorBus runs the connector_bus skills + personnel projection.
func SyncConnectorBus(root string, verbose bool) (skillsSynced int, agentsSynced int, err error) {
	return projectFromConnectorBus(root, verbose)
}

// SyncVisual runs the visual projection step.
func SyncVisual(root string, verbose bool) (count int, err error) {
	return projectVisual(root, verbose)
}

// SyncMiMoCode runs the MiMo Code projection step.
func SyncMiMoCode(root string, verbose bool) (count int, err error) {
	return projectToMimocode(root, verbose)
}

// SyncHarnessAgents runs the harness AGENTS projection step.
func SyncHarnessAgents(root string, verbose bool) (count int, err error) {
	return projectHarnessAgents(root, verbose)
}
