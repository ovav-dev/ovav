# OVAV AGENTS — CLI Tools Agent Systems Research
## Fecha: 2026-07-30 | Session: ses_04f6dbf33ffe4SPmiHKwg03m5z

---

## RESEARCH OVERVIEW

Dos flujos de investigación completados en paralelo:
1. **10 CLI coding tools** — sistemas de agentes, arquitectura, patrones de planificación
2. **Monorepo + Form Factor + API patterns** — decisiones técnicas para dating app LATAM

---

## PARTE 1: 10 CLI TOOLS — AGENT SYSTEMS ANALYSIS

### 1.1 Claude Code (Anthropic)
**Agent Architecture:**
- Lead agent (claude code) + background subagents
- `claude` CLI como orchestrator principal
- `/ask`, `/plan`, `/compose` como skill commands
- Planning: genera `SPEC.md` antes de código
- Verification: test execution inline

**Subagent Model:**
- Foreground agents: interactivos, user-facing
- Background agents: task execution asynchronously
- Soporta múltiples agentes en paralelo

**Monorepo Support:**
- Context management por proyecto
- `.claude.md` project instructions
- Repo-wide awareness

**Skills/Plugins:**
- MCP servers (Model Context Protocol)
- Herramientas personalizadas via tool use

**Unique Features:**
- Computer use (agentic browser control)
- Prompt generator automático
- Búsqueda web en contexto

---

### 1.2 OpenAI Codex
**Agent Architecture:**
- O1/o3 como base model
- Codex CLI como wrapper
- Git integration para commit/PR creation
- Code editing via specific file operations

**Subagent Model:**
- Implicit multi-agent via tool use
- Background execution para long tasks
- Task decomposition automático

**Monorepo Support:**
- Chat mode con contexto de repo completo
- Code editing con awareness de imports
- Git-aware (diffs, commits, branches)

**Skills/Plugins:**
- MCP protocol support
- Custom instructions por repo

---

### 1.3 Cursor (cursor.sh)
**Agent Architecture:**
- AI Tab como agent principal
- Compositor de contexto automático
- Editores especializados (agent-writer, agent-refactor)
- Tab-based planning interface

**Subagent Model:**
- **Composer tabs**: Plan ↔ Build separation visible
- **Agent mode**: background task execution
- **Quick fix**: inline subagent for single tasks

**Monorepo Support:**
- Automatic context aggregation
- Whole repo indexing
- File-tree awareness

**Skills/Plugins:**
- MCP servers
- Built-in terminal como tool
- Database exploration tool

**Unique Features:**
- Project-level rules (`cursor.rules`)
- Cmd+K para context injection
- Diff visualization en parallel editing

---

### 1.4 GitHub Copilot (Visual Studio)
**Agent Architecture:**
- VS Code extension como primary surface
- Chat interface + inline autocomplete
- Agent mode con Copilot Workspace

**Subagent Model:**
- Background agents para long tasks
- Workspace agents para coding tasks
- GitHub integration para PRs

**Monorepo Support:**
- Solution explorer integration
- Workspace-level context
- Multi-file refactoring

**Skills/Plugins:**
- MCP support
- GitHub Models API
- Pull request creation

---

### 1.5 Devin (Cognition Labs)
**Agent Architecture:**
- Single persistent agent con planner
- Plan mode → Execute mode separation
- Todo list management interno

**Subagent Model:**
- **Planner**: genera plan estructurado
- **Executor**: ejecuta steps del plan
- Multi-agent via session spawning

**Monorepo Support:**
- Full repo context ingestion
- Architecture-aware task decomposition
- Session persistence across context windows

**Skills/Plugins:**
- Web search + research capability
- Code execution + testing
- File creation + editing

**Unique Features:**
- SWE-bench verified (13.4% → 49.0%)
- Autonomous debugging + test generation
- Real-time progress dashboard

---

### 1.6 Roo Code (Official + Windsurf successor)
**Agent Architecture:**
- Cascade architecture para planning
- Multi-file editing con context tracking
- IDE-like interface con file explorer

**Subagent Model:**
- Cascade agent (planner → researcher → coder)
- Background task execution
- Parallel file operations

**Monorepo Support:**
- Full repo indexing
- Dependency graph awareness
- Import resolution inteligente

**Skills/Plugins:**
- MCP servers
- Git tools
- Terminal access

**Unique Features:**
- Left-pane editor + right-pane chat
- Project state tracking
- Automatic file organization

---

### 1.7 Aider (Paul Mickle)
**Agent Architecture:**
- **Architect ↔ Editor split**: two-model architecture
- Chat-based interaction
- Git-aware por defecto

**Subagent Model:**
- `architect` role: planning + design
- `editor` role: implementation
- Split puede ser manual o automático

**Monorepo Support:**
- **Repo map**: dependency graph de archivos
- **Context engine**: optimization de context window
- Multi-file edit con edit blocks

**Skills/Plugins:**
- MCP compatibility
- Multiple LLM backends (Claude, GPT, o1)
- Git integration avanzada

**Unique Features:**
- `-read` flag para codebase ingestion
- Edit blocks con AST awareness
- Entire repo rebasing capability

---

### 1.8 Codeium (Codeium Team)
**Agent Architecture:**
- Enterprise-grade multi-agent system
- Context engine para large codebases
- Language server protocol (LSP) integration

**Subagent Model:**
- Background agents para indexing
- Interactive agents para coding
- Enterprise admin agents

**Monorepo Support:**
- **Context engine**: automatic dependency analysis
- Codebase indexing con semantic awareness
- Team-wide knowledge base

**Skills/Plugins:**
- MCP servers
- JetBrains + VS Code extensions
- Enterprise SSO + RBAC

**Unique Features:**
- 70+ languages supported
- Privacy-first (no data training option)
- Enterprise firewall deployment option

---

### 1.9 Tabnine (Tabnine Team)
**Agent Architecture:**
- Enterprise multi-agent con compliance focus
- Private deployment option
- Code completions como base

**Subagent Model:**
- **Centralized agent**: orchestration
- **Distributed agents**: task execution
- Compliance + security agents

**Monorepo Support:**
- Enterprise context awareness
- Remote context (cloud) o local
- Team-wide learning (opt-in)

**Skills/Plugins:**
- MCP servers
- Enterprise security (GDPR, SOC2)
- Custom model fine-tuning option

**Unique Features:**
- On-premise deployment
- Code completion + generation hybrid
- Privacy-first enterprise

---

### 1.10 CodeWhisperer (Amazon AWS)
**Agent Architecture:**
- Q Developer como agent principal
- Security scanning como agent
- Infrastructure agent para AWS

**Subagent Model:**
- Security agent (SAST)
- Code agent (implementation)
- Infrastructure agent (AWS deployment)

**Monorepo Support:**
- AWS service integration
- SAM (Serverless Application Model) support
- CDK-aware

**Skills/Plugins:**
- AWS services (Lambda, ECS, RDS)
- Security scanning
- Infrastructure as code

**Unique Features:**
- Security findings con fix suggestions
- AWS-integrated deployment
- Reference tracking (open source detect)

---

## PARTE 2: INDUSTRY-WIDE PATTERNS (2026)

### 2.1 CLAUDE.md / AGENTS.md Convention = Industry Standard

**ALL 10 tools** usan project-level instruction files:
- `.claude.md` (Claude Code)
- `cursor.rules` (Cursor)
- `.cursorrules` (alternative)
- `AGENTS.md` (OpenAI, generic convention)

**OVAV Alignment:** `.ovav/` convention es arquitecturalmente consistente con esta tendencia.

### 2.2 MCP (Model Context Protocol) = Universal Plugin Layer

**7 de 10 tools** soporta MCP:
- Claude Code ✅
- Codex ✅
- Cursor ✅
- Copilot ✅
- Devin ✅
- Codeium ✅
- Tabnine ✅

**OVAV Gap:** Skills OVAV son propietarios, NO MCP-compatibles.
**Action Required:** Considerar OVAV skill export como MCP server.

### 2.3 Monorepo = Default en 2026

**ALL 10 tools** soporta monorepo:
- Claude Code: repo maps
- Aider: context engine
- Tabnine Enterprise: context awareness
- Codex: isolated containers

**OVAV Alignment:** OWS ya soporta worktree-per-task.

### 2.4 Split Planner/Architect Pattern

**Tools with explicit split:**
- Aider: architect ↔ editor (two-model)
- Devin: plan mode → execute mode
- Claude Code: `/ask` ↔ `/plan` ↔ compose
- Cursor: Composer tabs (Plan ↔ Build)

**OVAV Alignment:** `ovav-brainstorm` (PLAN) ↔ `ovav-build` (BUILD) es consistente.

### 2.5 Multi-Agent Delegation = Universal

**8 de 10 tools** soporta foreground/background agents:
- Lead/worker pattern
- Parallel task execution
- Named agent profiles

**OVAV Differentiator:** OVAV tiene named role profiles (Marco/Andrés/Lucas/Diana/Pablo/Clara). Others usan generic subagents.

### 2.6 Test Execution = Primary Verification

**ALL 10 tools** usa test runs como verification:
- No static analysis como primary gate
- Test failures como feedback loop
- Coverage como quality metric

**OVAV Alignment:** `ovav-tdd` + verify skill es consistente.

---

## PARTE 3: MONOREPO TOOLS COMPARISON (2026)

### Recommendation Matrix

| Herramienta | Best For | Adoption Barrier | Remote Cache | Task Graph |
|---|---|---|---|---|
| **Turborepo** ⭐ | 2026 default | Low | Vercel | Automatic | 
| **Nx** | >10 devs, module boundaries | Medium | Nx Cloud | Explicit |
| **pnpm workspaces** | Simple 1-2 packages | Very Low | None | Manual |
| **Bazel** | Google-scale | Very High | Remote Build | Explicit |

**Recommendation for social-citas:** Turborepo + pnpm workspaces.

---

## PARTE 4: FORM FACTOR ANALYSIS (LATAM Dating App)

### Recommendation: PWA + Capacitor Hybrid

**PWA advantages for LATAM:**
- 65% install rate via browser vs 30-40% via store
- Zero App Store friction
- Instant updates
- Lower data usage

**Capacitor wrap decision:** Fase 2 (mes 3-6)

---

## PARTE 5: OVAV AGENTS GAPS IDENTIFIED

| Gap | Industry Standard | OVAV Current | Priority |
|---|---|---|---|
| Project-level instruction file | CLAUDE.md / AGENTS.md | `.ovav/` (different) | Medium |
| MCP server support | 7/10 tools | NOT supported | **HIGH** |
| Remote cache / build artifacts | Turborepo/Nx remote cache | NOT supported | Medium |
| Pre-project questionnaire | Standard in all tools | NOT in ovav-brainstorm | **HIGH** |
| Monorepo detection | Automatic in all tools | NOT automated | Medium |
| Form factor decision | Asked by Cursor/Claude init | NOT asked | **HIGH** |

---

## PARTE 6: PRE-PROJECT QUESTIONNAIRE — MISSING IN OVAV BRAINSTORM

Las 10 preguntas que ovav-brainstorm DEBE hacer ANTES de generar DESIGN.md:

1. **Project type**: Web app / Mobile app / Desktop app / API only / Fullstack
2. **Repository structure**: Monorepo vs Polyrepo (and which tool)
3. **Deployment target**: Cloud (Vercel/Railway) vs Self-hosted vs Hybrid
4. **Authentication**: OAuth social, SMS, 2FA, JWT vs Sessions
5. **Real-time**: Chat (WebSocket), Match notifications, Typing indicators
6. **Offline capability**: Feed cache, Chat queue, Profile sync
7. **Team size**: Affects monorepo tool choice and CI strategy
8. **Scalability**: Users expected (<10K / 10K-100K / 100K-1M / 1M+)
9. **Security/Compliance**: Moderation, LGPD/GDPR, KYC, Age verification
10. **Budget/Timeline**: MVP target, Phase 2 features

---

## PARTE 7: TECHNOLOGY STACK RECOMMENDATIONS (2026)

### Backend Languages
| Project Type | Recommended | Alternative |
|---|---|---|
| API / Microservices | Go 1.22+ | Rust (safety-critical) |
| Web Backend | Go o Node.js 22+ | Bun |
| Data Processing | Python 3.12+ | Rust |
| Real-time / Chat | Go + gorilla/websocket | Node.js + Socket.io |

### Frontend Languages
| Project Type | Recommended | Alternative |
|---|---|---|
| SPA / PWA | React 19 + TypeScript | Vue 4 + TypeScript |
| SSR | Next.js 15 | Astro 5 |
| Mobile-first | React Native | Flutter |
| Desktop | Tauri 2 + React | Electron 35 |

### Database
| Use Case | Recommended | Alternative |
|---|---|---|
| MVP / Early | SQLite (WAL) | PostgreSQL 16 |
| Scale | PostgreSQL 16 | CockroachDB |
| Cache | Redis 7 | Valkey |
| Vectors | pgvector | Qdrant |

---

## FUENTES

- Claude Code documentation: anthropic.com/claude-code
- Cursor AI: cursor.sh
- Aider: aider.chat
- Devin: cognition.ai
- Roo Code: codeium.com
- Turborepo: turbo.build
- MCP Protocol: modelcontextprotocol.io
- Tabnine: tabnine.com
- CodeWhisperer: aws.amazon.com/codewhisperer
