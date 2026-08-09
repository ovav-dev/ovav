// Package tools implements the OVAV tool catalog.
//
// Provides a discoverable, named registry of all OVAV tools
// (Go, Python, shell). Every tool has a unique ID, clear name,
// description, language, and status.
//
// Commands:
//
//	ovav tools list            List all tools
//	ovav tools search <name>   Search by name or keyword
//	ovav tools show <id>       Show tool details
//	ovav tools go              Show Go-native tools only
//
// C9.6: Tool catalog. Zero external dependencies.
package tools

import (
	"fmt"
	"sort"
	"strings"
)

// ── Tool registry ────────────────────────────────────────────────────────────
// Every OVAV tool gets a unique ID, clear name, and description.
// This is the SINGLE SOURCE OF TRUTH for tool discovery.
// When a new tool is created, register it here.
// When a tool is deprecated, update its status.

// Status represents the lifecycle state of a tool.
type Status string

const (
	StatusActive     Status = "active"
	StatusFrozen     Status = "frozen"     // Python: only bugfixes
	StatusDeprecated Status = "deprecated" // Will be removed
	StatusPlanned    Status = "planned"    // Go: not yet implemented
)

// Language represents the implementation language.
type Language string

const (
	LangGo     Language = "Go"
	LangPython Language = "Python"
	LangShell  Language = "Shell"
	LangTS     Language = "TypeScript"
)

// Tool represents a registered OVAV tool.
type Tool struct {
	ID          string   `json:"id"`          // Unique identifier (e.g., "agent-runtime")
	Name        string   `json:"name"`        // Human-readable name
	Description string   `json:"description"` // What it does, in one sentence
	Path        string   `json:"path"`        // Filesystem path (relative to repo root)
	Language    Language `json:"language"`    // Go, Python, Shell, TypeScript
	Status      Status   `json:"status"`      // active, frozen, deprecated, planned
	Category    string   `json:"category"`    // runtime, security, cli, web, quality, dev
	Commands    []string `json:"commands"`    // Key commands or entry points
	GoBinary    string   `json:"go_binary"`   // Go binary name (if applicable)
}

// builtinCatalog is the complete OVAV tool registry.
// Sorted by category for readability.
var builtinCatalog = []Tool{
	// ── Runtime & Core ───────────────────────────────────────────────────
	{
		ID: "agent-runtime", Name: "Agent Runtime",
		Description: "Núcleo del runtime de agentes: sesiones, ruteo, firewalls, chronos, identidad",
		Path:        "tools/agent_runtime/", Language: LangPython, Status: StatusFrozen,
		Category: "runtime", Commands: []string{"session_greeting.py", "context_firewall_v2.py"},
	},
	{
		ID: "governor", Name: "OVAV Governor",
		Description: "Núcleo del governor: firma, integridad, memorias, voces, trust",
		Path:        "tools/governor/", Language: LangPython, Status: StatusActive,
		Category: "runtime", Commands: []string{"output_guard.py"},
	},
	{
		ID: "memory", Name: "Memory Orchestrator",
		Description: "Orquestador de memoria con sub-sistemas: core, gateway, probe, signals",
		Path:        "tools/memory/", Language: LangPython, Status: StatusActive,
		Category: "runtime", Commands: []string{"memory_governor.py"},
	},

	// ── CLI & User-facing ────────────────────────────────────────────────
	{
		ID: "ovav-cli", Name: "OVAV CLI (Go)",
		Description: "CLI pública OVAV — binario nativo Go. status, profile, config, waiver, version, tools",
		Path:        "go-runtime/cmd/ovav/", Language: LangGo, Status: StatusActive,
		Category: "cli", Commands: []string{"ovav status", "ovav profile list", "ovav tools list", "ovav waiver <motivo>"},
		GoBinary: "ovav",
	},
	{
		ID: "ovav-cli-py", Name: "OVAV CLI (Python — deprecated)",
		Description: "CLI Python original. Deprecada Jun 2026. Reemplazada por Go Cockpit TUI + ovav Go CLI",
		Path:        "bin/ovav-shell", Language: LangPython, Status: StatusDeprecated,
		Category: "cli", Commands: []string{"bin/ovav-shell (frozen)"},
	},
	{
		ID: "cockpit-tui", Name: "Cockpit TUI (Go) — DEFAULT",
		Description: "TUI interactiva Go nativa — Bubble Tea, Elm Architecture, mouse, multiplataforma. Lanzada por defecto con 'ovav'",
		Path:        "go-runtime/cmd/cockpit/", Language: LangGo, Status: StatusActive,
		Category: "cli", Commands: []string{"ovav"},
		GoBinary: "cockpit",
	},

	// ── Control Panel ────────────────────────────────────────────────────
	{
		ID: "cpanel-backend", Name: "cPanel Backend (Go)",
		Description: "Servidor HTTP del panel de control. Auth OAuth, profiles API, health checks nativos. Desplegado en Fly.io",
		Path:        "go-runtime/cmd/cpanel/", Language: LangGo, Status: StatusActive,
		Category: "web", Commands: []string{"cpanel serve"},
		GoBinary: "cpanel",
	},
	{
		ID: "cpanel-frontend", Name: "cPanel Frontend (React)",
		Description: "Frontend React 18 + TypeScript + Vite. Login OAuth, vistas por rol, launcher multi-modo",
		Path:        "tools/cpanel/static/", Language: LangTS, Status: StatusActive,
		Category: "web", Commands: []string{"npm run build"},
	},

	// ── Security ─────────────────────────────────────────────────────────
	{
		ID: "security-vault", Name: "Security Vault",
		Description: "Credential vault, exfiltration detection, integrity checks, quarantine system",
		Path:        "tools/security/", Language: LangPython, Status: StatusActive,
		Category: "security", Commands: []string{"credential_vault.py", "living_integrity.py"},
	},
	{
		ID: "vault-go", Name: "Vault (Go)",
		Description: "AES-256-GCM encrypt/decrypt nativo. Key derivation via PBKDF2. Stdlib only",
		Path:        "go-runtime/internal/vault/", Language: LangGo, Status: StatusActive,
		Category: "security", Commands: []string{"vault.Encrypt()", "vault.Decrypt()"},
		GoBinary: "ovav",
	},
	{
		ID: "license-go", Name: "License Binding (Go)",
		Description: "License key binding via PBKDF2-600K + machine ID. Verification with constant-time compare",
		Path:        "go-runtime/internal/license/", Language: LangGo, Status: StatusActive,
		Category: "security", Commands: []string{"license.Bind()", "license.Verify()"},
		GoBinary: "ovav",
	},
	{
		ID: "permissions", Name: "Permission Authority",
		Description: "Autorización, rego policies, claims, sandbox, gobierno de permisos",
		Path:        "tools/permissions/", Language: LangPython, Status: StatusActive,
		Category: "security", Commands: []string{"permission_authority.py", "materialize.py"},
	},
	{
		ID: "hooks", Name: "Git Security Hooks",
		Description: "Pre-push gate (waiver + marker), post-checkout integrity. Enforcement mecánico",
		Path:        "tools/hooks/", Language: LangPython, Status: StatusActive,
		Category: "security", Commands: []string{"pre-push", "post-checkout"},
	},

	// ── Quality & Validation ─────────────────────────────────────────────
	{
		ID: "validators", Name: "Validators (70+)",
		Description: "~70 validadores: integridad, arquitectura, seguridad, gates, drift, cobertura",
		Path:        "tools/validators/", Language: LangPython, Status: StatusActive,
		Category: "quality", Commands: []string{"protected_branch.go", "host_config_drift.go", "agent_surface_hierarchy.go"},
	},
	{
		ID: "pipeline", Name: "Quality Pipeline (harnesses)",
		Description: "240 scripts de verificación, evaluación, testing. Corazón del pipeline de calidad OVAV",
		Path:        "tools/harnesses/", Language: LangPython, Status: StatusActive,
		Category: "quality", Commands: []string{"evaluation_pipeline_runner.py", "workspace_safety_gate.py", "ovav worktree create", "ovav worktree done"},
	},
	{
		ID: "model-integrity", Name: "Model Integrity",
		Description: "Capa de integridad de modelos: claim parser, verification engine, output rails",
		Path:        "tools/model_integrity/", Language: LangPython, Status: StatusActive,
		Category: "quality", Commands: []string{"output_rails.py", "verification_engine.py"},
	},

	// ── Install & Deploy ─────────────────────────────────────────────────
	{
		ID: "install-gateway", Name: "Install Gateway",
		Description: "Gateway de instalación: apply, rollback, backup, plan, manifiesto",
		Path:        "tools/install_gateway/", Language: LangPython, Status: StatusActive,
		Category: "devops", Commands: []string{"install_apply.py", "install_rollback.py"},
	},
	{
		ID: "build-system", Name: "Build System",
		Description: "Empaquetado del producto OVAV para distribución. Makefile + cross-compile 5 targets",
		Path:        "go-runtime/Makefile", Language: LangGo, Status: StatusActive,
		Category: "devops", Commands: []string{"make build", "make test", "make release"},
		GoBinary: "ovav",
	},

	// ── Web & Research ───────────────────────────────────────────────────
	{
		ID: "web-tools", Name: "Web Tools",
		Description: "Fetching HTTP, búsqueda multi-engine (Brave, Tavily, DuckDuckGo), caché de research",
		Path:        "tools/web/", Language: LangPython, Status: StatusActive,
		Category: "web", Commands: []string{"search_gateway.py", "web_fetch.py"},
	},
	{
		ID: "research-engine", Name: "Research Engine",
		Description: "Motor de research: scoring de fuentes, verificación, briefs, benchmarks",
		Path:        "tools/research/", Language: LangPython, Status: StatusActive,
		Category: "web", Commands: []string{"source_scorer.py", "evidence_verifier.py"},
	},

	// ── Developer Tools ──────────────────────────────────────────────────
	{
		ID: "profile-compiler", Name: "Profile Compiler (Go)",
		Description: "Compilador de perfiles profesionales. List, apply, remove. 8 perfiles built-in",
		Path:        "go-runtime/internal/profile/", Language: LangGo, Status: StatusActive,
		Category: "dev", Commands: []string{"ovav profile list", "ovav profile apply <area>"},
		GoBinary: "ovav",
	},
	{
		ID: "profile-compiler-py", Name: "Profile Compiler (Python — frozen)",
		Description: "Compilador Python original. Congelado. Reemplazado por Go",
		Path:        "tools/profile/", Language: LangPython, Status: StatusFrozen,
		Category: "dev", Commands: []string{"ovav_profile.py"},
	},
	{
		ID: "prompt-architect", Name: "Prompt Architect",
		Description: "Arquitecto de prompts del sistema",
		Path:        "tools/prompt/", Language: LangPython, Status: StatusActive,
		Category: "dev", Commands: []string{"prompt_architect.py"},
	},
	{
		ID: "knowledge", Name: "Knowledge Compiler",
		Description: "Compilador y seed de base de conocimiento",
		Path:        "tools/knowledge/", Language: LangPython, Status: StatusActive,
		Category: "dev", Commands: []string{"knowledge_compiler.py"},
	},
	{
		ID: "skills", Name: "Skills Manager",
		Description: "Descubrimiento, scoring y registro de skills",
		Path:        "tools/skills/", Language: LangPython, Status: StatusActive,
		Category: "dev", Commands: []string{"skill_manager.py", "skill_scoring.py"},
	},

	// ── Economics & Monitoring ───────────────────────────────────────────
	{
		ID: "economy", Name: "Economy Dashboard",
		Description: "Costos, presupuesto, dashboard económico del runtime",
		Path:        "tools/economy/", Language: LangPython, Status: StatusActive,
		Category: "dev", Commands: []string{"economy_dashboard.py"},
	},
	{
		ID: "ovav-status", Name: "OVAV Status",
		Description: "Dashboard de estado: memoria, tokens, engine — ELIMINADO Python, reemplazado por Go native",
		Path:        "tools/ovav_status/", Language: LangPython, Status: StatusDeprecated,
		Category: "dev", Commands: []string{"ovav_status.py (REMOVED — use 'ovav status --write-markers' Go native)"},
	},
	{
		ID: "ovav-status-go", Name: "OVAV Status (Go native)",
		Description: "Status engine Go nativo — 0 Python. Governor, integrity, branch, tokens. Escribe ovav_status.json",
		Path:        "go-runtime/internal/status/", Language: LangGo, Status: StatusActive,
		Category: "dev", Commands: []string{"ovav status --write-markers"},
	},
	{
		ID: "visual", Name: "Visual Tools",
		Description: "Temas visuales, monitoreo, pipeline de release, paletas WezTerm",
		Path:        "tools/visual/", Language: LangPython, Status: StatusActive,
		Category: "dev", Commands: []string{"theme_manager.py", "wezterm_palette.py"},
	},

	// ── Workflow & Session ───────────────────────────────────────────────
	{
		ID: "snapshot", Name: "Session Snapshot",
		Description: "Snapshots de sesión, handoff, continuidad entre worktrees",
		Path:        "tools/snapshot/", Language: LangPython, Status: StatusActive,
		Category: "runtime", Commands: []string{"session_handoff.py", "snapshot_writer.py"},
	},
	{
		ID: "context", Name: "Context Control",
		Description: "Control de presupuesto de contexto y ruteo por profundidad",
		Path:        "tools/context/", Language: LangPython, Status: StatusActive,
		Category: "runtime", Commands: []string{"context_budget.py", "task_router.py"},
	},
	{
		ID: "work-session", Name: "Work Session Queue",
		Description: "Cola de sesiones de trabajo",
		Path:        "tools/work_session/", Language: LangPython, Status: StatusActive,
		Category: "runtime", Commands: []string{"work_session_queue.py"},
	},

	// ── Platform & Protocols ─────────────────────────────────────────────
	{
		ID: "mcp-server", Name: "MCP Server",
		Description: "Servidor MCP (Model Context Protocol) de OVAV",
		Path:        "tools/mcp/", Language: LangPython, Status: StatusActive,
		Category: "dev", Commands: []string{"mcp_server.py"},
	},
	{
		ID: "protocols", Name: "Communication Protocols",
		Description: "Protocolos de comunicación: gateway, auditoría, whitelist",
		Path:        "tools/protocols/", Language: LangPython, Status: StatusActive,
		Category: "runtime", Commands: []string{"protocol_gateway.py", "audit_log.py"},
	},
	{
		ID: "github-gates", Name: "GitHub Gates",
		Description: "Gates para issues y pushes de GitHub. Push gate + issue gate",
		Path:        "tools/github/", Language: LangPython, Status: StatusActive,
		Category: "devops", Commands: []string{"ovav_git_push_gate.py", "ovav_gh_issue_gate.py"},
	},
	{
		ID: "platform", Name: "Platform Resolver",
		Description: "Resolución de paths cross-platform (Linux, macOS, Windows)",
		Path:        "tools/platform/", Language: LangPython, Status: StatusActive,
		Category: "dev", Commands: []string{"platform_resolver.py"},
	},

	// ── Maintenance ─────────────────────────────────────────────────────
	{
		ID: "maintenance", Name: "Maintenance Tools",
		Description: "Limpieza del ledger de trabajo",
		Path:        "tools/maintenance/", Language: LangPython, Status: StatusActive,
		Category: "dev", Commands: []string{"clean_work_ledger.py"},
	},
	{
		ID: "logging", Name: "System Logging",
		Description: "Log de operaciones del sistema",
		Path:        "tools/logging/", Language: LangPython, Status: StatusActive,
		Category: "runtime", Commands: []string{"system_logger.py"},
	},
	{
		ID: "install-tools", Name: "Install/Uninstall",
		Description: "Desinstalación limpia de OVAV",
		Path:        "tools/install/", Language: LangPython, Status: StatusActive,
		Category: "devops", Commands: []string{"uninstall_ovav.py"},
	},
	{
		ID: "workstation", Name: "Workstation Access",
		Description: "Acceso y workspace de WezTerm",
		Path:        "tools/workstation/", Language: LangPython, Status: StatusActive,
		Category: "dev", Commands: []string{"wezterm_workspace.py"},
	},
	{
		ID: "dev-sandbox", Name: "Dev Sandbox",
		Description: "Herramientas de desarrollo internas (sandbox visual)",
		Path:        "tools/dev/", Language: LangPython, Status: StatusActive,
		Category: "dev", Commands: []string{"visual_sandbox.py"},
	},
	{
		ID: "agents-config", Name: "Agent Config (OpenCode)",
		Description: "Gestión de agentes de proyecto (project_opencode.py)",
		Path:        "tools/agents/", Language: LangPython, Status: StatusActive,
		Category: "dev", Commands: []string{"project_opencode.py"},
	},
	{
		ID: "common-utils", Name: "Common Utilities",
		Description: "Utilidades compartidas (formato CLI)",
		Path:        "tools/common/", Language: LangPython, Status: StatusActive,
		Category: "dev", Commands: []string{"cli_format.py"},
	},
	{
		ID: "repair-script", Name: "Runtime Repair",
		Description: "Script de reparación del runtime OVAV",
		Path:        "tools/repair_ovav_runtime.sh", Language: LangShell, Status: StatusActive,
		Category: "devops", Commands: []string{"bash repair_ovav_runtime.sh"},
	},
}

// ── Catalog query functions ─────────────────────────────────────────────────

// Catalog returns a copy of the full tool catalog.
func Catalog() []Tool {
	result := make([]Tool, len(builtinCatalog))
	copy(result, builtinCatalog)
	return result
}

// ByID finds a tool by its unique ID.
func ByID(id string) *Tool {
	for i := range builtinCatalog {
		if builtinCatalog[i].ID == id {
			return &builtinCatalog[i]
		}
	}
	return nil
}

// Search finds tools matching a keyword in name, description, or ID.
func Search(keyword string) []Tool {
	kw := strings.ToLower(keyword)
	var results []Tool
	for _, t := range builtinCatalog {
		if strings.Contains(strings.ToLower(t.ID), kw) ||
			strings.Contains(strings.ToLower(t.Name), kw) ||
			strings.Contains(strings.ToLower(t.Description), kw) ||
			strings.Contains(strings.ToLower(t.Category), kw) {
			results = append(results, t)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})
	return results
}

// ByCategory returns tools in a specific category.
func ByCategory(cat string) []Tool {
	var results []Tool
	for _, t := range builtinCatalog {
		if strings.EqualFold(t.Category, cat) {
			results = append(results, t)
		}
	}
	return results
}

// ByLanguage returns tools in a specific language.
func ByLanguage(lang Language) []Tool {
	var results []Tool
	for _, t := range builtinCatalog {
		if t.Language == lang {
			results = append(results, t)
		}
	}
	return results
}

// ByStatus returns tools with a specific status.
func ByStatus(s Status) []Tool {
	var results []Tool
	for _, t := range builtinCatalog {
		if t.Status == s {
			results = append(results, t)
		}
	}
	return results
}

// Categories returns all unique categories.
func Categories() []string {
	seen := make(map[string]bool)
	var cats []string
	for _, t := range builtinCatalog {
		if !seen[t.Category] {
			seen[t.Category] = true
			cats = append(cats, t.Category)
		}
	}
	sort.Strings(cats)
	return cats
}

// ── Formatting ──────────────────────────────────────────────────────────────

// FormatList returns a human-readable table of tools.
func FormatList(tools []Tool, showAll bool) string {
	if len(tools) == 0 {
		return "No tools found."
	}

	statusIcon := map[Status]string{
		StatusActive:     "🟢",
		StatusFrozen:     "🔒",
		StatusDeprecated: "🔴",
		StatusPlanned:    "⬜",
	}

	langIcon := map[Language]string{
		LangGo:     "🐹",
		LangPython: "🐍",
		LangShell:  "🐚",
		LangTS:     "🟦",
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n  OVAV Tool Catalog — %d tools\n", len(tools)))
	b.WriteString("  ───────────────────────────────────────────────────────────────────────────────\n")

	for _, t := range tools {
		si := statusIcon[t.Status]
		li := langIcon[t.Language]
		cat := t.Category

		b.WriteString(fmt.Sprintf("  %s %s %-25s %s  %s\n",
			si, li, t.Name, cat, t.Description))

		if showAll {
			b.WriteString(fmt.Sprintf("      id: %s | path: %s | lang: %s\n",
				t.ID, t.Path, t.Language))
			if len(t.Commands) > 0 {
				b.WriteString(fmt.Sprintf("      run: %s\n", strings.Join(t.Commands, ", ")))
			}
		}
	}

	if !showAll {
		b.WriteString("\n  🟢 active  🔒 frozen  🔴 deprecated  ⬜ planned")
		b.WriteString("\n  🐹 Go  🐍 Python  🐚 Shell  🟦 TypeScript")
		b.WriteString("\n\n  Use 'ovav tools show <id>' for details.")
		b.WriteString("\n  Use 'ovav tools search <word>' to find by keyword.")
	}
	b.WriteString("\n")
	return b.String()
}

// FormatTool returns a detailed view of a single tool.
func FormatTool(t *Tool) string {
	if t == nil {
		return "Tool not found."
	}

	statusIcon := map[Status]string{
		StatusActive: "🟢 active", StatusFrozen: "🔒 frozen",
		StatusDeprecated: "🔴 deprecated", StatusPlanned: "⬜ planned",
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n  %s\n", t.Name))
	b.WriteString(fmt.Sprintf("  ──────────────────────────────────────────────\n"))
	b.WriteString(fmt.Sprintf("  ID:          %s\n", t.ID))
	b.WriteString(fmt.Sprintf("  Status:      %s\n", statusIcon[t.Status]))
	b.WriteString(fmt.Sprintf("  Language:    %s\n", t.Language))
	b.WriteString(fmt.Sprintf("  Category:    %s\n", t.Category))
	b.WriteString(fmt.Sprintf("  Path:        %s\n", t.Path))
	if t.GoBinary != "" {
		b.WriteString(fmt.Sprintf("  Go Binary:   %s\n", t.GoBinary))
	}
	b.WriteString(fmt.Sprintf("  Description: %s\n", t.Description))
	if len(t.Commands) > 0 {
		b.WriteString(fmt.Sprintf("  Commands:    %s\n", strings.Join(t.Commands, ", ")))
	}
	b.WriteString("\n")
	return b.String()
}

// FormatCategories returns a list of categories with tool counts.
func FormatCategories() string {
	var b strings.Builder
	b.WriteString("\n  Tool Categories\n")
	b.WriteString("  ──────────────────────────────────────────────\n")
	for _, cat := range Categories() {
		tools := ByCategory(cat)
		b.WriteString(fmt.Sprintf("  %-15s %d tools\n", cat, len(tools)))
	}
	b.WriteString("\n")
	return b.String()
}
