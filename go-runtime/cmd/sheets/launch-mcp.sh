#!/usr/bin/env bash
# ovav-sheets/launch-mcp.sh — launcher for the OVAV Sheets MCP server.
#
# Detects the repo root (parent of go-runtime/), then runs
# `go run ./cmd/sheets mcp` from the go-runtime/ subdirectory.
# Designed to be referenced from .ovav/source/opencode/config.yaml as a
# local stdio MCP server.
#
# Usage:
#   bash launch-mcp.sh                  # starts server on stdin/stdout
set -euo pipefail
BIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GO_RUNTIME="$BIN_DIR"
REPO_ROOT="$(cd "$GO_RUNTIME/../.." && pwd)"
if [[ ! -d "$GO_RUNTIME" ]]; then
  echo "❌ go-runtime not found at $GO_RUNTIME" >&2
  exit 1
fi
exec go run -C "$GO_RUNTIME" ./cmd/sheets mcp

