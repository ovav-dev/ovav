#!/usr/bin/env bash
# OVAV MEMORY v3 — pi-extension build script
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist"

echo "Building OVAV MEMORY v3 TypeScript extension..."

cd "$ROOT_DIR"

# Check dependencies
if ! command -v tsc &> /dev/null; then
  echo "Error: TypeScript compiler (tsc) not found."
  echo "Install with: npm install -g typescript"
  exit 1
fi

# Clean dist
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

# Compile TypeScript
tsc --project "$SCRIPT_DIR/tsconfig.json"

# Copy extension manifest
cp "$SCRIPT_DIR/extension.json" "$DIST_DIR/"

echo "Build complete → $DIST_DIR"
