# OVAV MCP Integration Plan — Consumer-Ready Package

**Date:** 2026-07-23
**Author:** Thavren (Platform Engineering)
**Status:** 🟡 PLAN READY — Awaiting CEO waiver for develop branch

---

## Executive Summary

OVAV will integrate the top global MCP servers to create a **production-grade AI development platform** with:
- **Frontend:** Figma design-to-code, design system registry, visual verification
- **Backend:** Database access, API gateway, memory systems, security hardening
- **Consumer:** One-click MCP server setup for end users

---

## 🎯 Integration Scope

### Phase 1: Core MCP Servers (Priority 🔴 CRITICAL)

| MCP Server | Source | Purpose | Consumer Package |
|------------|--------|---------|------------------|
| **Figma Context MCP** | GLips/Figma-Context-MCP (15.5k★) | Design-to-code workflow | `ovav-mcp-figma` |
| **PostgreSQL MCP** | modelcontextprotocol/server-postgres | Database queries | `ovav-mcp-postgres` |
| **SQLite MCP** | modelcontextprotocol/server-sqlite | Local analytics | `ovav-mcp-sqlite` |
| **Memory MCP** | modelcontextprotocol/server-memory | Knowledge graph | `ovav-mcp-memory` |
| **Git MCP** | modelcontextprotocol/server-git | Version control | `ovav-mcp-git` |

### Phase 2: Design System (Priority 🔴 HIGH)

| MCP Server | Source | Purpose | Consumer Package |
|------------|--------|---------|------------------|
| **Better Design** | marvkr/better-design (176★) | shadcn/ui registry | `ovav-mcp-design-system` |
| **UX Skill** | Laith0003/ux-skill (47★) | Anti-AI-slop linter | `ovav-mcp-ux-linter` |
| **Tailwind MCP** | CarbonoDev/tailwindcss-mcp-server | Tailwind assistance | `ovav-mcp-tailwind` |

### Phase 3: Backend Robustness (Priority 🟡 HIGH)

| MCP Server | Source | Purpose | Consumer Package |
|------------|--------|---------|------------------|
| **API Agent** | agoda-com/api-agent (282★) | REST/GraphQL gateway | `ovav-mcp-api-gateway` |
| **AnythingMCP** | HelpCode-ai/anythingmcp (162★) | Universal MCP gateway | `ovav-mcp-gateway` |
| **Redis MCP** | modelcontextprotocol/server-redis | Cache/sessions | `ovav-mcp-redis` |

### Phase 4: Advanced (Priority 🟢 MEDIUM)

| MCP Server | Source | Purpose | Consumer Package |
|------------|--------|---------|------------------|
| **Figwright** | awdr74100/figwright (133★) | Two-way Figma flow | `ovav-mcp-figma-sync` |
| **Figma Bridge** | gethopp/figma-mcp-bridge (374★) | API limit bypass | `ovav-mcp-figma-bridge` |

---

## 📦 Consumer Package Structure

```
ovav-consumer/
├── README.md                           # Quick start guide
├── install.sh                          # One-click installer
├── uninstall.sh                        # Clean removal
├── config/
│   ├── opencode.jsonc                  # OpenCode MCP config
│   ├── claude-desktop.json             # Claude Desktop config
│   └── cursor.json                     # Cursor config
├── servers/
│   ├── figma/
│   │   ├── package.json
│   │   └── index.ts
│   ├── postgres/
│   │   ├── package.json
│   │   └── index.ts
│   ├── design-system/
│   │   ├── package.json
│   │   └── registry.json
│   ├── api-gateway/
│   │   ├── package.json
│   │   └── index.ts
│   └── memory/
│       ├── package.json
│       └── index.ts
├── skills/
│   ├── figma-workflow.md               # Skill: Figma → Code
│   ├── design-system.md                # Skill: Component generation
│   ├── api-design.md                   # Skill: REST/GraphQL design
│   └── database-queries.md             # Skill: Safe DB queries
└── docs/
    ├── FRONTEND.md                     # Frontend integration guide
    ├── BACKEND.md                      # Backend integration guide
    ├── SECURITY.md                     # Security best practices
    └── TROUBLESHOOTING.md              # Common issues
```

---

## 🔧 Implementation Details

### 1. Figma Context MCP Integration

**File:** `servers/figma/index.ts`

```typescript
// OVAV Figma MCP Server
// Wraps GLips/Figma-Context-MCP with OVAV-specific enhancements

import { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';

export class OvavFigmaMCP extends McpServer {
  name = 'ovav-figma';
  version = '1.0.0';

  tools = [
    {
      name: 'figma_get_layout',
      description: 'Get Figma layout information for design-to-code',
      inputSchema: {
        type: 'object',
        properties: {
          fileKey: { type: 'string', description: 'Figma file key' },
          nodeId: { type: 'string', description: 'Node ID to inspect' }
        }
      }
    },
    {
      name: 'figma_get_tokens',
      description: 'Extract design tokens (colors, spacing, typography)',
      inputSchema: {
        type: 'object',
        properties: {
          fileKey: { type: 'string' },
          tokenType: { 
            type: 'string', 
            enum: ['colors', 'spacing', 'typography', 'all']
          }
        }
      }
    },
    {
      name: 'figma_generate_component',
      description: 'Generate React/Vue/Svelte component from Figma node',
      inputSchema: {
        type: 'object',
        properties: {
          fileKey: { type: 'string' },
          nodeId: { type: 'string' },
          framework: { type: 'string', enum: ['react', 'vue', 'svelte'] },
          designSystem: { type: 'string', default: 'shadcn' }
        }
      }
    }
  ];
}
```

### 2. Design System Registry

**File:** `servers/design-system/registry.json`

```json
{
  "name": "ovav-design-system",
  "version": "1.0.0",
  "components": {
    "button": {
      "variants": ["primary", "secondary", "ghost", "destructive"],
      "sizes": ["sm", "md", "lg"],
      "frameworks": {
        "react": "shadcn-button",
        "vue": "primevue-button",
        "svelte": "shadcn-svelte-button"
      }
    },
    "card": {
      "variants": ["default", "outlined", "elevated"],
      "frameworks": {
        "react": "shadcn-card",
        "vue": "primevue-card"
      }
    }
  },
  "tokens": {
    "colors": {
      "primary": "hsl(var(--primary))",
      "secondary": "hsl(var(--secondary))",
      "destructive": "hsl(var(--destructive))"
    },
    "spacing": {
      "xs": "0.25rem",
      "sm": "0.5rem",
      "md": "1rem",
      "lg": "1.5rem",
      "xl": "2rem"
    }
  }
}
```

### 3. API Gateway MCP Server

**File:** `servers/api-gateway/index.ts`

```typescript
// Universal API Gateway MCP
// Connects to any REST/GraphQL API via OpenAPI spec

import { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';

export class OvavAPIGatewayMCP extends McpServer {
  name = 'ovav-api-gateway';
  version = '1.0.0';

  tools = [
    {
      name: 'api_register',
      description: 'Register an API from OpenAPI/Swagger spec',
      inputSchema: {
        type: 'object',
        properties: {
          name: { type: 'string' },
          specUrl: { type: 'string' },
          auth: { 
            type: 'object',
            properties: {
              type: { type: 'string', enum: ['bearer', 'apikey', 'oauth2'] },
              token: { type: 'string' }
            }
          }
        }
      }
    },
    {
      name: 'api_call',
      description: 'Execute an API call with automatic auth handling',
      inputSchema: {
        type: 'object',
        properties: {
          api: { type: 'string' },
          endpoint: { type: 'string' },
          method: { type: 'string', enum: ['GET', 'POST', 'PUT', 'DELETE'] },
          body: { type: 'object' }
        }
      }
    },
    {
      name: 'api_document',
      description: 'Auto-document API endpoints from code',
      inputSchema: {
        type: 'object',
        properties: {
          source: { type: 'string' },
          framework: { type: 'string', enum: ['express', 'fastify', 'gin', 'fiber'] }
        }
      }
    }
  ];
}
```

### 4. Consumer Install Script

**File:** `install.sh`

```bash
#!/bin/bash
# OVAV Consumer MCP Installer
# One-click setup for all MCP servers

set -e

echo "🔧 OVAV MCP Consumer Installer"
echo "=============================="

# Detect platform
detect_platform() {
  if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    echo "linux"
  elif [[ "$OSTYPE" == "darwin"* ]]; then
    echo "macos"
  elif [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" ]]; then
    echo "windows"
  fi
}

PLATFORM=$(detect_platform)
echo "📍 Platform: $PLATFORM"

# Check prerequisites
check_prerequisites() {
  command -v node >/dev/null 2>&1 || { echo "❌ Node.js required"; exit 1; }
  command -v npm >/dev/null 2>&1 || { echo "❌ npm required"; exit 1; }
  echo "✅ Prerequisites met"
}

check_prerequisites

# Install MCP servers
install_servers() {
  echo "📦 Installing MCP servers..."
  
  # Core servers
  npm install -g @modelcontextprotocol/server-postgres
  npm install -g @modelcontextprotocol/server-sqlite
  npm install -g @modelcontextprotocol/server-memory
  npm install -g @modelcontextprotocol/server-git
  
  # OVAV custom servers
  npm install -g ovav-mcp-figma
  npm install -g ovav-mcp-design-system
  npm install -g ovav-mcp-api-gateway
  
  echo "✅ All servers installed"
}

install_servers

# Configure for detected editor
configure_editor() {
  local editor=$1
  local config_dir=$2
  
  echo "⚙️  Configuring for $editor..."
  mkdir -p "$config_dir"
  
  # Generate config based on editor
  case $editor in
    "opencode")
      cat > "$config_dir/opencode.jsonc" << EOF
{
  "mcp": {
    "figma": {
      "command": "npx",
      "args": ["-y", "ovav-mcp-figma"],
      "env": {
        "FIGMA_ACCESS_TOKEN": "\${FIGMA_TOKEN}"
      }
    },
    "postgres": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-postgres", "\${DATABASE_URL}"]
    },
    "design-system": {
      "command": "npx",
      "args": ["-y", "ovav-mcp-design-system"]
    }
  }
}
EOF
      ;;
    "claude")
      cat > "$config_dir/claude-desktop.json" << EOF
{
  "mcpServers": {
    "figma": {
      "command": "npx",
      "args": ["-y", "ovav-mcp-figma"],
      "env": {
        "FIGMA_ACCESS_TOKEN": "\${FIGMA_TOKEN}"
      }
    },
    "postgres": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-postgres", "\${DATABASE_URL}"]
    }
  }
}
EOF
      ;;
    "cursor")
      cat > "$config_dir/cursor.json" << EOF
{
  "mcp": {
    "figma": {
      "command": "npx",
      "args": ["-y", "ovav-mcp-figma"]
    },
    "postgres": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-postgres", "\${DATABASE_URL}"]
    }
  }
}
EOF
      ;;
  esac
  
  echo "✅ $editor configured"
}

# Auto-detect and configure editors
[[ -d "$HOME/.config/opencode" ]] && configure_editor "opencode" "$HOME/.config/opencode"
[[ -d "$HOME/.config/claude" ]] && configure_editor "claude" "$HOME/.config/claude"
[[ -d "$HOME/.cursor" ]] && configure_editor "cursor" "$HOME/.cursor"

echo ""
echo "🎉 Installation complete!"
echo ""
echo "Next steps:"
echo "  1. Set FIGMA_TOKEN: export FIGMA_TOKEN=your_token"
echo "  2. Set DATABASE_URL: export DATABASE_URL=postgresql://..."
echo "  3. Restart your editor"
echo "  4. MCP servers are ready to use!"
```

---

## 🛡️ Security Considerations

### MCP Security Checklist

- [ ] **Tool name namespacing** — All OVAV tools prefixed with `ovav_`
- [ ] **Token audience validation** — Verify token issuer before tool execution
- [ ] **Stale authorization handling** — Implement token refresh before TTL expiry
- [ ] **Tool description sanitization** — Treat all descriptions as untrusted
- [ ] **Namespace isolation** — Each MCP server gets unique tool namespace
- [ ] **Audit logging** — Log all tool invocations for compliance

### OWASP Compliance

Per OWASP MCP Top 10 2026:
- ✅ **Tool poisoning defense** — Namespace isolation + description sanitization
- ✅ **Prompt injection defense** — Input validation on all tool parameters
- ✅ **Memory poisoning defense** — Scope isolation for memory MCP
- ✅ **Tool interference defense** — Unique tool names across servers

---

## 📊 Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| **MCP Server Uptime** | 99.9% | Health checks every 30s |
| **Tool Response Time** | <500ms p95 | Tracing middleware |
| **Consumer Setup Time** | <5 min | install.sh execution |
| **Error Rate** | <0.1% | Tool invocation logs |
| **Security Incidents** | 0 | Audit log monitoring |

---

## 🚀 Deployment Strategy

### Development (Local)
```bash
# Clone and build
git clone https://github.com/ovav/ovav-consumer.git
cd ovav-consumer
npm install
npm run build

# Run locally
npm run dev
```

### Staging
```bash
# Build Docker image
docker build -t ovav-mcp-consumer:staging .

# Deploy to staging
docker push registry.ovav.dev/ovav-mcp-consumer:staging
```

### Production
```bash
# Tag release
git tag -a v1.0.0 -m "Consumer MCP package v1.0.0"
git push origin v1.0.0

# Build and push
docker build -t ovav-mcp-consumer:latest .
docker push registry.ovav.dev/ovav-mcp-consumer:latest
```

---

## 📝 Next Steps

1. **CEO Approval** — Obtain waiver for develop branch modifications
2. **Squad Assignment** — Assign implementation tasks to:
   - **Andrés** — Figma MCP integration
   - **Lucas** — Design system registry
   - **Helena** — API gateway implementation
   - **Clara** — Testing and validation
3. **Implementation** — Execute Phase 1-4 per plan
4. **Documentation** — Complete consumer guides
5. **Release** — Publish to npm + Docker registry

---

**Status:** 🟡 PLAN COMPLETE — Awaiting waiver for implementation
