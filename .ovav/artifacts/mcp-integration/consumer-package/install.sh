#!/bin/bash
# OVAV Consumer MCP Installer (pnpm)
# One-click setup for all MCP servers

set -e

echo "🔧 OVAV MCP Consumer Installer (pnpm)"
echo "====================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Detect platform
detect_platform() {
  if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    echo "linux"
  elif [[ "$OSTYPE" == "darwin"* ]]; then
    echo "macos"
  elif [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" ]]; then
    echo "windows"
  else
    echo "unknown"
  fi
}

# Check prerequisites
check_prerequisites() {
  echo "📋 Checking prerequisites..."
  
  if ! command -v node &> /dev/null; then
    echo -e "${RED}❌ Node.js is required. Install from https://nodejs.org${NC}"
    exit 1
  fi
  
  if ! command -v pnpm &> /dev/null; then
    echo -e "${YELLOW}⚠️  pnpm not found. Installing...${NC}"
    npm install -g pnpm
    if ! command -v pnpm &> /dev/null; then
      echo -e "${RED}❌ Failed to install pnpm. Please install manually: npm install -g pnpm${NC}"
      exit 1
    fi
  fi
  
  NODE_VERSION=$(node -v | cut -d'v' -f2 | cut -d'.' -f1)
  if [ "$NODE_VERSION" -lt 18 ]; then
    echo -e "${YELLOW}⚠️  Node.js 18+ recommended (current: $(node -v))${NC}"
  fi
  
  PNPM_VERSION=$(pnpm --version)
  echo -e "${GREEN}✅ Prerequisites met (pnpm v${PNPM_VERSION})${NC}"
  echo ""
}

# Install MCP servers
install_servers() {
  echo "📦 Installing MCP servers with pnpm..."
  echo ""
  
  # Core MCP servers (official)
  echo "  Installing core servers..."
  pnpm add -g @modelcontextprotocol/server-postgres 2>/dev/null || true
  pnpm add -g @modelcontextprotocol/server-sqlite 2>/dev/null || true
  pnpm add -g @modelcontextprotocol/server-memory 2>/dev/null || true
  pnpm add -g @modelcontextprotocol/server-git 2>/dev/null || true
  pnpm add -g @modelcontextprotocol/server-fetch 2>/dev/null || true
  
  # OVAV custom servers
  echo "  Installing OVAV servers..."
  pnpm add -g ovav-mcp-figma 2>/dev/null || true
  pnpm add -g ovav-mcp-design-system 2>/dev/null || true
  pnpm add -g ovav-mcp-api-gateway 2>/dev/null || true
  pnpm add -g ovav-mcp-ux-linter 2>/dev/null || true
  
  echo -e "${GREEN}✅ All servers installed${NC}"
  echo ""
}

# Generate configuration for editor
generate_config() {
  local editor=$1
  local config_dir=$2
  local config_file=$3
  
  echo "  Configuring for ${BLUE}$editor${NC}..."
  mkdir -p "$config_dir"
  
  # Check if config exists and backup
  if [ -f "$config_dir/$config_file" ]; then
    cp "$config_dir/$config_file" "$config_dir/$config_file.bak"
    echo "    Backed up existing config"
  fi
  
  # Generate config with pnpm exec
  cat > "$config_dir/$config_file" << 'EOF'
{
  "mcp": {
    "ovav-figma": {
      "command": "pnpm",
      "args": ["exec", "ovav-mcp-figma"],
      "env": {
        "FIGMA_ACCESS_TOKEN": "${FIGMA_TOKEN}"
      }
    },
    "ovav-postgres": {
      "command": "pnpm",
      "args": ["exec", "@modelcontextprotocol/server-postgres", "${DATABASE_URL}"]
    },
    "ovav-sqlite": {
      "command": "pnpm",
      "args": ["exec", "@modelcontextprotocol/server-sqlite", "${SQLITE_PATH:-./data.db}"]
    },
    "ovav-memory": {
      "command": "pnpm",
      "args": ["exec", "@modelcontextprotocol/server-memory"]
    },
    "ovav-git": {
      "command": "pnpm",
      "args": ["exec", "@modelcontextprotocol/server-git", "--repository", "${REPO_PATH:-.}"]
    },
    "ovav-design-system": {
      "command": "pnpm",
      "args": ["exec", "ovav-mcp-design-system"]
    },
    "ovav-ux-linter": {
      "command": "pnpm",
      "args": ["exec", "ovav-mcp-ux-linter"]
    }
  }
}
EOF
  
  echo -e "    ${GREEN}✅ Config generated${NC}"
}

# Configure all detected editors
configure_editors() {
  echo "⚙️  Configuring editors..."
  echo ""
  
  # OpenCode
  if [ -d "$HOME/.config/opencode" ]; then
    generate_config "OpenCode" "$HOME/.config/opencode" "opencode.jsonc"
  fi
  
  # Claude Desktop (macOS)
  if [ -d "$HOME/Library/Application Support/Claude" ]; then
    generate_config "Claude Desktop" "$HOME/Library/Application Support/Claude" "claude_desktop_config.json"
  fi
  
  # Claude Desktop (Linux)
  if [ -d "$HOME/.config/claude" ]; then
    generate_config "Claude Desktop" "$HOME/.config/claude" "claude_desktop_config.json"
  fi
  
  # Cursor
  if [ -d "$HOME/.cursor" ]; then
    generate_config "Cursor" "$HOME/.cursor" "mcp.json"
  fi
  
  # VS Code (via settings)
  if [ -d "$HOME/.config/Code" ]; then
    echo "    ${YELLOW}ℹ️  VS Code: Install 'MCP' extension and add config manually${NC}"
  fi
  
  echo ""
}

# Set up environment variables
setup_env() {
  echo "🔐 Setting up environment..."
  echo ""
  
  # Create .env file if it doesn't exist
  if [ ! -f "$HOME/.ovav-env" ]; then
    cat > "$HOME/.ovav-env" << 'EOF'
# OVAV MCP Environment Variables
# Uncomment and set the values you need

# Figma API Token (get from https://www.figma.com/developers/api#access-tokens)
# export FIGMA_TOKEN="your_figma_token_here"

# Database URL (PostgreSQL)
# export DATABASE_URL="postgresql://user:password@localhost:5432/dbname"

# SQLite path (optional, defaults to ./data.db)
# export SQLITE_PATH="./data.db"

# Repository path for Git MCP (optional, defaults to current directory)
# export REPO_PATH="."
EOF
    
    echo -e "  ${GREEN}✅ Created ~/.ovav-env${NC}"
    echo -e "  ${YELLOW}ℹ️  Edit ~/.ovav-env to set your credentials${NC}"
  else
    echo -e "  ${YELLOW}ℹ️  ~/.ovav-env already exists, skipping${NC}"
  fi
  
  # Source env in shell profile
  if [ -f "$HOME/.bashrc" ] && ! grep -q "source ~/.ovav-env" "$HOME/.bashrc"; then
    echo "" >> "$HOME/.bashrc"
    echo "# OVAV MCP Environment" >> "$HOME/.bashrc"
    echo "source ~/.ovav-env 2>/dev/null || true" >> "$HOME/.bashrc"
    echo -e "  ${GREEN}✅ Added to ~/.bashrc${NC}"
  fi
  
  if [ -f "$HOME/.zshrc" ] && ! grep -q "source ~/.ovav-env" "$HOME/.zshrc"; then
    echo "" >> "$HOME/.zshrc"
    echo "# OVAV MCP Environment" >> "$HOME/.zshrc"
    echo "source ~/.ovav-env 2>/dev/null || true" >> "$HOME/.zshrc"
    echo -e "  ${GREEN}✅ Added to ~/.zshrc${NC}"
  fi
  
  echo ""
}

# Verify installation
verify_installation() {
  echo "🔍 Verifying installation..."
  echo ""
  
  local servers=(
    "ovav-mcp-figma"
    "ovav-mcp-design-system"
    "@modelcontextprotocol/server-postgres"
    "@modelcontextprotocol/server-sqlite"
    "@modelcontextprotocol/server-memory"
    "@modelcontextprotocol/server-git"
  )
  
  local installed=0
  local total=${#servers[@]}
  
  for server in "${servers[@]}"; do
    if pnpm list -g "$server" &> /dev/null; then
      echo -e "  ${GREEN}✅ $server${NC}"
      ((installed++))
    else
      echo -e "  ${RED}❌ $server${NC}"
    fi
  done
  
  echo ""
  echo "Installed: $installed/$total"
  echo ""
}

# Print summary
print_summary() {
  echo "🎉 Installation Complete!"
  echo "========================"
  echo ""
  echo "📋 Next Steps:"
  echo "  1. Edit ~/.ovav-env to set your credentials"
  echo "  2. Restart your terminal (or run: source ~/.ovav-env)"
  echo "  3. Restart your editor (OpenCode/Claude/Cursor)"
  echo "  4. MCP servers are ready to use!"
  echo ""
  echo "📚 Documentation:"
  echo "  • Frontend: https://docs.ovav.dev/frontend"
  echo "  • Backend: https://docs.ovav.dev/backend"
  echo "  • Security: https://docs.ovav.dev/security"
  echo ""
  echo "🤝 Support:"
  echo "  • Issues: https://github.com/ovav/ovav-consumer/issues"
  echo "  • Discord: https://discord.gg/ovav"
  echo ""
}

# Main
main() {
  local PLATFORM=$(detect_platform)
  echo "📍 Platform: $PLATFORM"
  echo ""
  
  check_prerequisites
  install_servers
  configure_editors
  setup_env
  verify_installation
  print_summary
}

# Run main
main "$@"
