# OVAV MCP Consumer Package

> **One-click MCP server setup for AI-powered development (pnpm)**

## 🚀 Quick Start

```bash
# Install all MCP servers
pnpm install

# Or run the installer
bash install.sh
```

## 📦 What's Included

### Frontend Design
- **Figma Context MCP** — Design-to-code workflow
- **Design System Registry** — shadcn/ui components
- **UX Linter** — Anti-AI-slop quality gate
- **Tailwind MCP** — Utility class assistance

### Backend Robustness
- **PostgreSQL MCP** — Database queries
- **SQLite MCP** — Local analytics
- **API Gateway** — REST/GraphQL integration
- **Memory MCP** — Knowledge graph

## ⚙️ Configuration

### OpenCode
```json
// ~/.config/opencode/opencode.jsonc
{
  "mcp": {
    "ovav-figma": {
      "command": "pnpm",
      "args": ["exec", "ovav-mcp-figma"],
      "env": { "FIGMA_TOKEN": "${FIGMA_TOKEN}" }
    },
    "ovav-postgres": {
      "command": "pnpm",
      "args": ["exec", "@modelcontextprotocol/server-postgres", "${DATABASE_URL}"]
    }
  }
}
```

### Claude Desktop
```json
// ~/Library/Application Support/Claude/claude_desktop_config.json
{
  "mcpServers": {
    "ovav-figma": {
      "command": "pnpm",
      "args": ["exec", "ovav-mcp-figma"],
      "env": { "FIGMA_TOKEN": "${FIGMA_TOKEN}" }
    }
  }
}
```

### Cursor
```json
// ~/.cursor/mcp.json
{
  "mcp": {
    "ovav-figma": {
      "command": "pnpm",
      "args": ["exec", "ovav-mcp-figma"]
    }
  }
}
```

## 🎯 Usage Examples

### Figma → Code
```
Agent: "Generate a login form from this Figma design"
→ figma_get_layout(fileKey="abc123", nodeId="1:234")
→ figma_generate_component(framework="react", designSystem="shadcn")
→ Returns: Login.tsx with proper styling
```

### Database Query
```
Agent: "Show me active users from the last 7 days"
→ postgres_query(sql="SELECT * FROM users WHERE created_at > NOW() - INTERVAL '7 days'")
→ Returns: User records with proper formatting
```

### API Integration
```
Agent: "Fetch product catalog from our API"
→ api_register(name="products", specUrl="https://api.example.com/openapi.json")
→ api_call(api="products", endpoint="/catalog", method="GET")
→ Returns: Product data
```

## 📚 Documentation

- [Frontend Guide](docs/FRONTEND.md)
- [Backend Guide](docs/BACKEND.md)
- [Security Best Practices](docs/SECURITY.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)

## 🤝 Support

- **Issues:** https://github.com/ovav/ovav-consumer/issues
- **Discord:** https://discord.gg/ovav
- **Docs:** https://docs.ovav.dev

## 📄 License

MIT License — See [LICENSE](LICENSE) for details
