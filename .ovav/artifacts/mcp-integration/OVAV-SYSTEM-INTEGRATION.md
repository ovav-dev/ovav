# OVAV System MCP Integration — Ready for Implementation

**Date:** 2026-07-23
**Status:** 🟡 READY — Awaiting CEO waiver for develop branch

---

## 📋 Integration Summary

### What's Been Done

1. ✅ **Research Complete** — Top 11 MCP servers identified globally
2. ✅ **Consumer Package Created** — One-click installer with pnpm
3. ✅ **OVAV Config Updated** — New MCP servers added to config.yaml
4. ✅ **OpenCode Config Updated** — Consumer servers configured
5. ✅ **Skills Created** — 4 agent skills for MCP usage
6. ✅ **Security Docs Complete** — OWASP compliant

### What's Pending (Requires Waiver)

1. ⏳ **Install MCP Servers** — `pnpm install` blocked by policy
2. ⏳ **Test Integration** — Verify servers work
3. ⏳ **Update Skill Registry** — Add MCP skills to OVAV
4. ⏳ **Publish Package** — npm/Docker registry

---

## 🎯 MCP Servers to Install

### Core (Official)
```bash
pnpm add -g @modelcontextprotocol/server-postgres
pnpm add -g @modelcontextprotocol/server-sqlite
pnpm add -g @modelcontextprotocol/server-memory
pnpm add -g @modelcontextprotocol/server-git
pnpm add -g @modelcontextprotocol/server-fetch
```

### OVAV Custom
```bash
pnpm add -g ovav-mcp-figma
pnpm add -g ovav-mcp-design-system
pnpm add -g ovav-mcp-api-gateway
pnpm add -g ovav-mcp-ux-linter
```

---

## 📝 Config Changes Made

### OVAV Source Config (`.ovav/source/opencode/config.yaml`)

Added 6 new MCP servers:

```yaml
# MCP Consumer Servers (pnpm)
ovav-figma:
  type: local
  command: [pnpm, exec, ovav-mcp-figma]
  enabled: true
  env:
    FIGMA_ACCESS_TOKEN: "${FIGMA_TOKEN}"

ovav-postgres:
  type: local
  command: [pnpm, exec, "@modelcontextprotocol/server-postgres", "${DATABASE_URL}"]
  enabled: false  # Enable when DATABASE_URL is set

ovav-sqlite:
  type: local
  command: [pnpm, exec, "@modelcontextprotocol/server-sqlite", "${SQLITE_PATH:-./data.db}"]
  enabled: false  # Enable when SQLite is needed

ovav-memory:
  type: local
  command: [pnpm, exec, "@modelcontextprotocol/server-memory"]
  enabled: true

ovav-design-system:
  type: local
  command: [pnpm, exec, ovav-mcp-design-system]
  enabled: true

ovav-api-gateway:
  type: local
  command: [pnpm, exec, ovav-mcp-api-gateway]
  enabled: false  # Enable when API integration is needed
```

### OpenCode Config (`~/.config/opencode/opencode.jsonc`)

```json
{
  "mcp": {
    "ovav-figma": {
      "command": "pnpm",
      "args": ["exec", "ovav-mcp-figma"],
      "env": { "FIGMA_ACCESS_TOKEN": "${FIGMA_TOKEN}" }
    },
    "ovav-memory": {
      "command": "pnpm",
      "args": ["exec", "@modelcontextprotocol/server-memory"]
    },
    "ovav-design-system": {
      "command": "pnpm",
      "args": ["exec", "ovav-mcp-design-system"]
    }
  }
}
```

---

## 🚀 Implementation Steps (Post-Waiver)

### Step 1: Install MCP Servers
```bash
cd /home/braka/Systems/OVAV/tools/mcp/consumer
pnpm install
```

### Step 2: Test Servers
```bash
# Test Figma MCP
pnpm exec ovav-mcp-figma --help

# Test Memory MCP
pnpm exec @modelcontextprotocol/server-memory --help

# Test Design System MCP
pnpm exec ovav-mcp-design-system --help
```

### Step 3: Update Skill Registry
Add to `.opencode/skills/ovav-skill-registry/SKILL.md`:

```markdown
### 21. ovav-mcp-frontend
**Trigger:** Figma design, UI components, design tokens, visual development.
**In scope:** MCP servers for frontend design workflow.
**Rules:**
- Use ovav-figma for design-to-code (15.5k★ GLips/Figma-Context-MCP)
- Use ovav-design-system for shadcn/ui components
- Use ovav-ux-linter for anti-AI-slop quality gate
- Always extract design tokens before generating components

### 22. ovav-mcp-backend
**Trigger:** Database queries, API design, backend integration.
**In scope:** MCP servers for backend robustness.
**Rules:**
- Use ovav-postgres for PostgreSQL queries (parameterized only)
- Use ovav-sqlite for local analytics
- Use ovav-api-gateway for REST/GraphQL integration
- Always use transactions for write operations
```

### Step 4: Update Agent Instructions
Add to relevant agent AGENTS.md files:

```markdown
## MCP Tools Available

When working with frontend or backend tasks, you have access to MCP servers:

### Frontend
- `ovav_figma_get_layout` — Get Figma layout info
- `ovav_figma_generate_component` — Generate React/Vue/Svelte components
- `ovav_design_system_get_component` — Get UI components from registry

### Backend
- `ovav_postgres_query` — Execute PostgreSQL queries
- `ovav_api_call` — Execute API calls
- `ovav_memory_store` — Store data in knowledge graph
```

### Step 5: Publish Consumer Package
```bash
# Build and publish to npm
cd .ovav/artifacts/mcp-integration/consumer-package
pnpm publish

# Build Docker image
docker build -t ovav-mcp-consumer:latest .
docker push registry.ovav.dev/ovav-mcp-consumer:latest
```

---

## 📊 Impact on OVAV System

### Agent Capabilities Enhanced

| Agent | New Capabilities |
|-------|------------------|
| **Dante** (Digital Product) | Figma → Code, Design System, API Gateway |
| **Thavren** (Platform) | Database queries, Memory system, Git MCP |
| **Elena** (UX Design) | Figma integration, Design tokens, Component generation |
| **Clara** (QA) | Database testing, API testing, Visual regression |

### Consumer Experience

| Feature | Before | After |
|---------|--------|-------|
| **Frontend Dev** | Manual coding | Figma → Auto-generated |
| **Backend Dev** | Manual DB queries | Natural language queries |
| **API Integration** | Manual OpenAPI | Auto-discovery + execution |
| **Component Reuse** | Copy-paste | Design system registry |

---

## 🎯 Success Criteria

| Metric | Target | Measurement |
|--------|--------|-------------|
| MCP Servers Installed | 11 | `pnpm list -g` |
| Config Updated | 3 files | Git diff |
| Skills Added | 2 new | Skill registry |
| Agent Capabilities | 10+ new tools | Tool count |
| Consumer Package | Published | npm registry |

---

## 📚 Files Reference

| File | Purpose |
|------|---------|
| `PLAN.md` | Full integration roadmap |
| `CONSUMER-READY.md` | Consumer package summary |
| `consumer-package/install.sh` | One-click installer (pnpm) |
| `consumer-package/package.json` | Dependencies (pnpm) |
| `consumer-package/README.md` | Consumer documentation |
| `consumer-package/skills/*.md` | Agent skill files |
| `consumer-package/docs/SECURITY.md` | Security best practices |

---

## ✅ Verification Checklist

- [x] Research complete (11 MCP servers)
- [x] Consumer package created
- [x] Install script updated (pnpm)
- [x] OVAV config updated
- [x] OpenCode config updated
- [x] Skills created (4 skills)
- [x] Security docs complete
- [ ] **Pending:** CEO waiver
- [ ] **Pending:** pnpm install
- [ ] **Pending:** Test integration
- [ ] **Pending:** Update skill registry
- [ ] **Pending:** Publish package

---

**Status:** 🟡 READY FOR IMPLEMENTATION — All artifacts complete, awaiting waiver
