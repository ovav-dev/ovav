package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ovav/ovav/internal/validators"
)

// cmdDocs dispatches ovav docs subcommands.
//
// Usage: ovav docs <subcommand>
// Subcommands:
//   generate              — Generate all auto-generated docs
//   generate --target=X   — Generate specific doc (validators | commands | drift-targets | auto-fix)
//   check                 — Verify docs are up-to-date (CI mode)
func cmdDocs(args []string) int {
	if len(args) == 0 {
		printDocsHelp()
		return 0
	}
	switch args[0] {
	case "generate":
		return runDocsGenerate(args[1:])
	case "check":
		return runDocsCheck(args[1:])
	case "help", "--help", "-h":
		printDocsHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "OVAV docs: unknown subcommand %q\n", args[0])
		printDocsHelp()
		return 2
	}
}

func printDocsHelp() {
	fmt.Println(`OVAV docs — auto-generated documentation

Usage:
  ovav docs generate                    # Generate all docs
  ovav docs generate --target=validators # Specific target
  ovav docs check                       # Verify docs are up-to-date (CI)

Targets:
  validators     — Registry of all validators
  commands       — CLI command reference
  drift-targets  — Drift detection target registry
  auto-fix       — SAFE_FIX whitelist

Output: docs/auto-generated/*.md
CI: pre-commit hook refuses if stale`)
}

// runDocsGenerate generates docs (one or all targets).
func runDocsGenerate(args []string) int {
	root, err := cliFindRepoRootSafe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV docs: %v\n", err)
		return 1
	}

	// Parse target filter
	target := ""
	dryRun := false
	for _, a := range args {
		switch a {
		case "--dry-run":
			dryRun = true
		case "--target=validators":
			target = "validators"
		case "--target=commands":
			target = "commands"
		case "--target=drift-targets":
			target = "drift-targets"
		case "--target=auto-fix":
			target = "auto-fix"
		case "--help", "-h":
			printDocsHelp()
			return 0
		}
	}

	outDir := filepath.Join(root, "docs", "auto-generated")
	if !dryRun {
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "OVAV docs: mkdir: %v\n", err)
			return 1
		}
	}

	count := 0
	if target == "" || target == "validators" {
		if err := generateValidatorsDoc(root, outDir, dryRun); err != nil {
			fmt.Fprintf(os.Stderr, "OVAV docs (validators): %v\n", err)
			return 1
		}
		count++
	}
	if target == "" || target == "commands" {
		if err := generateCommandsDoc(root, outDir, dryRun); err != nil {
			fmt.Fprintf(os.Stderr, "OVAV docs (commands): %v\n", err)
			return 1
		}
		count++
	}
	if target == "" || target == "drift-targets" {
		if err := generateDriftTargetsDoc(root, outDir, dryRun); err != nil {
			fmt.Fprintf(os.Stderr, "OVAV docs (drift-targets): %v\n", err)
			return 1
		}
		count++
	}
	if target == "" || target == "auto-fix" {
		if err := generateAutoFixDoc(root, outDir, dryRun); err != nil {
			fmt.Fprintf(os.Stderr, "OVAV docs (auto-fix): %v\n", err)
			return 1
		}
		count++
	}

	if dryRun {
		fmt.Printf("OVAV docs (dry-run): would generate %d files\n", count)
	} else {
		fmt.Printf("OVAV docs: generated %d files in %s\n", count, outDir)
	}
	return 0
}

// runDocsCheck verifies docs are up-to-date (CI mode).
// Returns 0 if current, 1 if stale.
func runDocsCheck(args []string) int {
	root, err := cliFindRepoRootSafe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV docs: %v\n", err)
		return 1
	}

	outDir := filepath.Join(root, "docs", "auto-generated")

	// Generate in-memory, compare to disk
	docs := map[string]string{}
	for _, gen := range []struct {
		path string
		fn   func(string) (string, error)
	}{
		{"validators.md", func(r string) (string, error) { return generateValidatorsContent(r) }},
		{"commands.md", func(r string) (string, error) { return generateCommandsContent(r) }},
		{"drift-targets.md", func(r string) (string, error) { return generateDriftTargetsContent(r) }},
		{"auto-fix.md", func(r string) (string, error) { return generateAutoFixContent(r) }},
	} {
		content, err := gen.fn(root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "OVAV docs check: %s: %v\n", gen.path, err)
			return 1
		}
		docs[gen.path] = content
	}

	stale := false
	for name, expected := range docs {
		path := filepath.Join(outDir, name)
		actual, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("❌ %s: MISSING\n", name)
			stale = true
			continue
		}
		if string(actual) != expected {
			fmt.Printf("❌ %s: STALE (run 'ovav docs generate')\n", name)
			stale = true
			continue
		}
		fmt.Printf("✅ %s: current\n", name)
	}

	if stale {
		return 1
	}
	return 0
}

// ── Generators ──────────────────────────────────────────────────────────────

func generateValidatorsDoc(root, outDir string, dryRun bool) error {
	content, err := generateValidatorsContent(root)
	if err != nil {
		return err
	}
	if dryRun {
		return nil
	}
	return os.WriteFile(filepath.Join(outDir, "validators.md"), []byte(content), 0o644)
}

func generateValidatorsContent(root string) (string, error) {
	registry := validators.DefaultRegistry()
	all := registry.All()

	// Sort validators by ID for deterministic output
	sortedAll := make([]validators.Validator, len(all))
	copy(sortedAll, all)
	sort.Slice(sortedAll, func(i, j int) bool {
		return sortedAll[i].ID() < sortedAll[j].ID()
	})

	var b strings.Builder
	b.WriteString("# OVAV Validator Registry\n\n")
	b.WriteString("> **Auto-generated** from `internal/validators/*.go`. DO NOT EDIT MANUALLY.\n")
	b.WriteString("> Run `ovav docs generate` to refresh.\n\n")
	b.WriteString(fmt.Sprintf("**Total validators**: %d\n\n", len(sortedAll)))
	b.WriteString("| ID | Name | Weight | Description |\n")
	b.WriteString("|----|------|--------|-------------|\n")

	for _, v := range sortedAll {
		b.WriteString(fmt.Sprintf("| `%s` | %s | %d | %s |\n",
			v.ID(), v.Name(), v.Weight(), v.Description()))
	}

	b.WriteString("\n## Categories\n\n")
	categories := map[string][]string{}
	for _, v := range sortedAll {
		cat := categoryFromID(v.ID())
		categories[cat] = append(categories[cat], v.ID())
	}
	// Sort category names for deterministic output
	catNames := make([]string, 0, len(categories))
	for cat := range categories {
		catNames = append(catNames, cat)
	}
	sort.Strings(catNames)
	for _, cat := range catNames {
		ids := categories[cat]
		sort.Strings(ids)
		b.WriteString(fmt.Sprintf("### %s (%d)\n\n", cat, len(ids)))
		for _, id := range ids {
			b.WriteString(fmt.Sprintf("- `%s`\n", id))
		}
		b.WriteString("\n")
	}

	return b.String(), nil
}

func categoryFromID(id string) string {
	if strings.Contains(id, "permission") || strings.Contains(id, "agent") {
		return "Agent Governance"
	}
	if strings.Contains(id, "deploy") || strings.Contains(id, "install") || strings.Contains(id, "license") {
		return "Deployment"
	}
	if strings.Contains(id, "sbom") || strings.Contains(id, "supply") || strings.Contains(id, "integrity") {
		return "Supply Chain"
	}
	if strings.Contains(id, "caps") || strings.Contains(id, "capability") || strings.Contains(id, "trigger") {
		return "Capability Registry"
	}
	if strings.Contains(id, "worktree") || strings.Contains(id, "worktree") {
		return "Worktree System"
	}
	if strings.Contains(id, "firewall") || strings.Contains(id, "context") || strings.Contains(id, "economy") {
		return "Context Economy"
	}
	if strings.Contains(id, "keybindings") || strings.Contains(id, "bash") || strings.Contains(id, "readline") {
		return "Workstation"
	}
	if strings.Contains(id, "zero_trust") || strings.Contains(id, "security") || strings.Contains(id, "hardening") || strings.Contains(id, "exfiltration") {
		return "Security"
	}
	if strings.Contains(id, "loop") || strings.Contains(id, "project") || strings.Contains(id, "subagent") {
		return "Orchestration"
	}
	return "General"
}

func generateCommandsDoc(root, outDir string, dryRun bool) error {
	content, err := generateCommandsContent(root)
	if err != nil {
		return err
	}
	if dryRun {
		return nil
	}
	return os.WriteFile(filepath.Join(outDir, "commands.md"), []byte(content), 0o644)
}

func generateCommandsContent(root string) (string, error) {
	var b strings.Builder
	b.WriteString("# OVAV CLI Commands\n\n")
	b.WriteString("> **Auto-generated** from `cmd/ovav/*.go`. DO NOT EDIT MANUALLY.\n")
	b.WriteString("> Run `ovav docs generate` to refresh.\n\n")

	commands := []struct {
		name        string
		description string
	}{
		{"validate", "Run validation gates"},
		{"validate --fix", "Auto-remediate SAFE_FIX validators (ADR-011)"},
		{"drift", "Fragment vs live drift detection (ADR-007)"},
		{"deploy", "Auto-deploy fragments to live (ADR-008)"},
		{"ci", "CI runner commands (ADR-009)"},
		{"hooks", "Git hook management"},
		{"it", "Intelligent Terminal ops (ADR-010)"},
		{"memory", "Agent memory queries"},
		{"worktree", "OWS worktree lifecycle"},
		{"status", "System status"},
		{"sbom", "SBOM generation/verification"},
		{"integrity", "Runtime integrity baseline (ADR-006)"},
		{"drift show", "Visual diff fragment vs live"},
		{"drift catalog", "Drift history"},
		{"drift targets", "List registered drift targets"},
		{"deploy run", "Execute deploy pipeline"},
		{"deploy status", "Last deploy summary"},
		{"deploy list", "All recent deploys"},
		{"deploy rollback", "Restore from snapshot"},
		{"ci drift-check", "CI-friendly drift check"},
		{"hooks install-pre-commit", "Install baseline freshness hook"},
		{"hooks install-pre-push", "Install drift gate hook"},
		{"hooks install-all", "Install all OVAV hooks"},
		{"hooks status", "Show all hook states"},
		{"it reload", "IT reload via Win32 API"},
		{"it status", "Check if IT is running"},
		{"docs generate", "Generate auto-generated docs"},
		{"docs check", "Verify docs are up-to-date"},
	}

	b.WriteString("| Command | Description |\n")
	b.WriteString("|---------|-------------|\n")
	for _, c := range commands {
		b.WriteString(fmt.Sprintf("| `ovav %s` | %s |\n", c.name, c.description))
	}

	return b.String(), nil
}

func generateDriftTargetsDoc(root, outDir string, dryRun bool) error {
	content, err := generateDriftTargetsContent(root)
	if err != nil {
		return err
	}
	if dryRun {
		return nil
	}
	return os.WriteFile(filepath.Join(outDir, "drift-targets.md"), []byte(content), 0o644)
}

func generateDriftTargetsContent(root string) (string, error) {
	var b strings.Builder
	b.WriteString("# Drift Detection Targets\n\n")
	b.WriteString("> **Auto-generated** from `cmd/ovav/drift_targets.go`. DO NOT EDIT MANUALLY.\n")
	b.WriteString("> Run `ovav docs generate` to refresh.\n\n")
	b.WriteString("Per ADR-007 (drift detection). Each target is a (fragment, live) pair\n")
	b.WriteString("with a comparison function.\n\n")

	b.WriteString("| ID | Fragment | Live | Auto-fixable |\n")
	b.WriteString("|----|----------|------|--------------|\n")
	b.WriteString("| `it-keybindings` | `workstation/configs/intelligent-terminal/settings-fragment.json` | `/mnt/c/.../LocalState/settings.json` | ✅ |\n")
	b.WriteString("| `bash-inputrc` | `workstation/configs/inputrc/ovav.inputrc` | `~/.inputrc` | ✅ |\n")
	b.WriteString("| `runtime-baseline` | `.ovav/integrity_backups/baseline.json` | (file hashes) | ✅ |\n")
	b.WriteString("| `pinned-baseline` | `.ovav/integrity_backups/baseline.pinned.json` | (pinned vs current) | ⚠️ CEO approval |\n")
	b.WriteString("| `tool-configs` | `.ovav/registry/tool_configs.yaml` | `bin/ovav` | ❌ rebuild required |\n")

	return b.String(), nil
}

func generateAutoFixDoc(root, outDir string, dryRun bool) error {
	content, err := generateAutoFixContent(root)
	if err != nil {
		return err
	}
	if dryRun {
		return nil
	}
	return os.WriteFile(filepath.Join(outDir, "auto-fix.md"), []byte(content), 0o644)
}

func generateAutoFixContent(root string) (string, error) {
	entries := validators.GetSafeFixRegistry()

	var b strings.Builder
	b.WriteString("# Auto-Fix SAFE_FIX Registry\n\n")
	b.WriteString("> **Auto-generated** from `internal/validators/auto_fix_registry.go`. DO NOT EDIT MANUALLY.\n")
	b.WriteString("> Run `ovav docs generate` to refresh.\n\n")
	b.WriteString("Per ADR-011 (auto-remediation). Each entry is a validator that\n")
	b.WriteString("opt-in to `ovav validate --fix`.\n\n")
	b.WriteString(fmt.Sprintf("**Total entries**: %d\n\n", len(entries)))
	b.WriteString("| Validator ID | Description | Risk |\n")
	b.WriteString("|--------------|-------------|------|\n")
	for _, e := range entries {
		b.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n", e.ValidatorID, e.Description, e.RiskLevel))
	}

	b.WriteString("\n## Safety guards\n\n")
	b.WriteString("1. Snapshot before any fix\n")
	b.WriteString("2. Rollback on regression (fix introduces new issues)\n")
	b.WriteString("3. Max 10 fixes per run\n")
	b.WriteString("4. No fix on protected files (CEO waiver required)\n")
	b.WriteString("5. JSONL history with operator + timestamp + outcome\n")

	return b.String(), nil
}