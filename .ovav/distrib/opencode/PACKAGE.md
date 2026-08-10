# OVAV → OpenCode Distribution Package

**Status:** Ready for Distribution  
**Surface:** OpenCode (opencode.ai)  
**OVAV Version:** 3.4.0 — CAPA 9 — Go Runtime

---

## What Is This?

OVAV for OpenCode is a **zero-dependency governance layer** that gives any OpenCode user access to:

- **10 OVAV professional agents** (Platform Engineering, Research Intelligence, UX Design, etc.)
- **32 OVAV skills** (context-pack, brainstorm, security-gates, runtime-gates, squad-delegation, etc.)
- **6 MCP servers** (git, memory, budget, browser, figma, design-system)
- **OVAV monitor plugin** (session tracking, budget alerts, agent switch notifications)
- **OVAV governance** (git push gate, workspace safety, permission authority, context economy)

---

## Two-Line Install

```bash
# From any directory — first time only
git clone https://github.com/earendil-works/ovav.git && cd ovav

# Install OVAV governance to your OpenCode project
ovav install --opencode --target ~/Projects/myproject
```

That's it. OVAV will:
1. Copy `.opencode/` (agents, skills, plugins, themes) to your project
2. Copy `opencode.json` to your project (merges with existing config)
3. Create a backup at `~/.ovav/backup/<timestamp>/`

---

## Manual Install (No ovav CLI Required)

If you don't have the OVAV CLI yet:

```bash
# 1. Clone OVAV
git clone https://github.com/earendil-works/ovav.git
cd ovav

# 2. Copy to your OpenCode project
cp -r .opencode/ ~/Projects/myproject/
cp opencode.json ~/Projects/myproject/

# 3. Done — opencode will auto-discover agents from .opencode/agents/
cd ~/Projects/myproject
opencode
```

---

## Directory Structure

```
your-project/
├── .opencode/               ← OVAV governance surface
│   ├── agents/              ← 81 agent .md files
│   │   ├── area-platform-engineering.md
│   │   ├── lead-thavren.md
│   │   └── team-*.md        (60 team agents)
│   ├── skills/              ← 32 OVAV skills
│   │   ├── ovav-context-pack/
│   │   ├── ovav-security-gates/
│   │   └── ... (30 more)
│   ├── plugins/
│   │   └── ovav-monitor.js  ← session monitor + budget alerts
│   └── themes/
│       ├── ovav-dark.json
│       └── ovav-light.json
└── opencode.json            ← OVAV config (model, MCP, permissions)
```

---

## OpenCode Agent Discovery

OpenCode discovers agents from `CWD/.opencode/agents/*.md`. No registry, no package manager — pure directory-based enumeration.

**Agent types:**
| Type | Example ID | Visibility |
|---|---|---|
| Areas (10) | `area-platform-engineering` | TAB selector |
| Leads (10) | `lead-thavren`, `lead-eidren` | By name only |
| Teams (60) | `team-marco`, `team-clara` | Via Task tool |

**Usage:** Press `TAB` in OpenCode to see all 10 OVAV area agents.

---

## OVAV Default Configuration

```json
{
  "default_agent": "Platform Engineering",
  "model": "opencode-go/deepseek-v4-pro",
  "small_model": "opencode-go/deepseek-v4-flash",
  "instructions": ["opencode_AGENTS.md"]
}
```

---

## OVAV MCP Servers

| Server | Purpose | Status |
|---|---|---|
| `ovav-git` | Git operations, PR review, issue management | ✅ Enabled |
| `ovav-memory` | OVAV persistent memory (261 cards) | ✅ Enabled |
| `ovav-budget` | Token budget tracking and alerts | ✅ Enabled |
| `ovav-browser` | Headless browser for web research | ✅ Enabled |
| `ovav-figma` | Figma design token sync | ✅ Enabled |
| `ovav-design-system` | Design system enforcement | ✅ Enabled |
| `ovav-sqlite` | Project local database | ⏸ Disabled |

---

## Governance Built-In

Every OVAV agent inherits:

- **Git push gate** — blocks `git push` (use `gh pr create`), denies force push
- **Workspace safety** — validates paths before write operations
- **Permission authority** — deny `sudo`, `pip install`, `npm install` by default
- **Context economy** — T1-T5 tiered loading, no unnecessary context
- **Safe stop** — graceful session closure with checkpoint

---

## Updating OVAV

```bash
# Pull latest OVAV
cd /path/to/ovav
git pull

# Regenerate all artifacts
go run -C go-runtime ./cmd/ovav/ sync

# Reinstall to your projects
ovav install --opencode --target ~/Projects/myproject
```

---

## Requirements

- **OpenCode** 1.17+ (opencode.ai)
- **Go** 1.21+ (for OVAV CLI and MCP servers)
- **Git** (for OVAV clone and updates)

---

## OVAV License

OVAV is a commercial AI workstation governor. See `LICENSE` for terms.

---

*OVAV 3.4.0 — CAPA 9 — Go Runtime — 70/70 validators passing*
