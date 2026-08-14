#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────
#  OVAV Workstation Benchmark
#  Rule #34: measure shell startup, prompt render, tab open.
# ─────────────────────────────────────────────────────────────
set -euo pipefail

OVAV_ROOT="${OVAV_ROOT:-/home/braka/Systems/ovav}"

echo "═══════════════════════════════════════════════════════════"
echo "  OVAV WORKSTATION BENCHMARK"
echo "═══════════════════════════════════════════════════════════"
echo ""

# ─── 1. Clean Bash startup (no integrations) ────────────────
echo "▸ Clean Bash startup (no .bashrc, no integrations)"
echo "  → use 'env -i bash --noprofile --norc -c exit' as baseline"
time env -i HOME="$HOME" PATH="/usr/local/bin:/usr/bin:/bin" bash --noprofile --norc -c 'exit' 2>/dev/null
echo ""

# ─── 2. Interactive Bash startup (with current bashrc) ──────
echo "▸ Interactive Bash startup (with current bashrc)"
echo "  → use 'bash -i -c exit' to measure .bashrc + integrations"
time bash -i -c 'exit' 2>/dev/null || true
echo ""

# ─── 3. Prompt render (Starship) ────────────────────────────
if command -v starship >/dev/null 2>&1; then
  echo "▸ Starship prompt render"
  echo "  → measure time to render one prompt"
  for i in 1 2 3; do
    time starship prompt 2>/dev/null > /dev/null
  done
else
  echo "⚠ Starship not installed"
fi
echo ""

# ─── 4. Atuin startup ───────────────────────────────────────
if command -v atuin >/dev/null 2>&1; then
  echo "▸ Atuin init (bash startup overhead)"
  time bash -c 'eval "$(atuin init bash --disable-up-arrow 2>/dev/null)"' 2>/dev/null
else
  echo "⚠ Atuin not installed"
fi
echo ""

# ─── 5. fzf load ───────────────────────────────────────────
if command -v fzf >/dev/null 2>&1; then
  echo "▸ fzf load"
  time bash -c 'eval "$(fzf --bash 2>/dev/null)"' 2>/dev/null
else
  echo "⚠ fzf not installed"
fi
echo ""

# ─── 6. zoxide load ────────────────────────────────────────
if command -v zoxide >/dev/null 2>&1; then
  echo "▸ zoxide load"
  time bash -c 'eval "$(zoxide init bash 2>/dev/null)"' 2>/dev/null
else
  echo "⚠ zoxide not installed"
fi
echo ""

# ─── 7. OVAV CLI startup ───────────────────────────────────
if [ -x "$HOME/.local/bin/ovav" ]; then
  echo "▸ OVAV CLI startup"
  time "$HOME/.local/bin/ovav" status > /dev/null 2>&1
else
  echo "⚠ OVAV CLI not installed"
fi
echo ""

# ─── 8. OpenCode TUI version ───────────────────────────────
if [ -x "$HOME/.opencode/bin/opencode" ]; then
  echo "▸ OpenCode version"
  "$HOME/.opencode/bin/opencode" --version 2>&1 | head -1
else
  echo "⚠ OpenCode not installed"
fi
echo ""

# ─── Summary ───────────────────────────────────────────────
echo "═══════════════════════════════════════════════════════════"
echo "  Budget targets:"
echo "    Clean Bash startup:    <50ms"
echo "    Interactive Bash:      <500ms"
echo "    Prompt render:         <100ms"
echo "    OVAV CLI startup:      <200ms"
echo ""
echo "  Run again after install to compare before/after."
echo "═══════════════════════════════════════════════════════════"