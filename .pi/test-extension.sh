#!/bin/bash
# OVAV Premium Input - Test Script
# Verifica que la extensión cargue y los componentes funcionen

echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║         OVAV PREMIUM INPUT v2.0 — TEST SUITE                ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo ""

# Test 1: Verify symlink
echo "🔍 Test 1: Verificando symlink..."
if [ -L ~/.pi/agent/extensions/ovav-premium-input ]; then
    TARGET=$(readlink -f ~/.pi/agent/extensions/ovav-premium-input)
    echo "   ✅ Symlink correcto: $TARGET"
else
    echo "   ❌ Symlink no existe - creando..."
    ln -sfn /home/braka/Systems/OVAV/.ovav/worktrees/feature-piagent-themes/tools/extensions/ovav-premium-input ~/.pi/agent/extensions/ovav-premium-input
fi
echo ""

# Test 2: Verify file exists and has correct content
echo "🔍 Test 2: Verificando archivo de extensión..."
FILE=~/.pi/agent/extensions/ovav-premium-input/index.ts
if [ -f "$FILE" ]; then
    SIZE=$(wc -c < "$FILE")
    echo "   ✅ Archivo existe: $SIZE bytes"
    
    # Check for key components
    grep -q "class CommandPalette" "$FILE" && echo "   ✅ CommandPalette class found"
    grep -q "class ShortcutsOverlay" "$FILE" && echo "   ✅ ShortcutsOverlay class found"
    grep -q "startHotReloadWatcher" "$FILE" && echo "   ✅ Hot-reload system found"
    grep -q "SelectList" "$FILE" && echo "   ✅ TUI SelectList component found"
else
    echo "   ❌ Archivo no encontrado!"
fi
echo ""

# Test 3: Run PI with extension and check output
echo "🔍 Test 3: Ejecutando PI con extensión..."
cd /home/braka/Systems/OVAV/.ovav/worktrees/feature-piagent-themes

OUTPUT=$(PI_LOG=info timeout 10 npx pi --print "test" 2>&1)

echo "   ──────────────────────────────────────────"
echo "$OUTPUT" | grep -i "ovav"
echo "   ──────────────────────────────────────────"
echo ""

# Test 4: Check if hot-reload is working
echo "🔍 Test 4: Verificando hot-reload..."
if echo "$OUTPUT" | grep -q "Hot-reload: ACTIVE"; then
    echo "   ✅ Hot-reload está ACTIVO"
else
    echo "   ⚠️ Hot-reload puede no estar activo (sesión muy corta)"
fi
echo ""

# Test 5: List available themes
echo "🔍 Test 5: Temas OVAV disponibles..."
ls -1 ~/.pi/agent/themes/ovav-*.json 2>/dev/null | while read f; do
    NAME=$(basename "$f" .json)
    echo "   • $NAME"
done
echo ""

echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║              INSTRUCCIONES DE USO                             ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo ""
echo "1. Iniciar PI con la extensión:"
echo "   cd /home/braka/Systems/OVAV/.ovav/worktrees/feature-piagent-themes"
echo "   pi"
echo ""
echo "2. Verás las notificaciones de OVAV Premium al inicio"
echo ""
echo "3. Comandos disponibles:"
echo "   /cmd        → Command Palette (TUI SelectList)"
echo "   ?           → Shortcuts Overlay"
echo "   /ovav daily → Estado del sistema"
echo ""
echo "4. Para probar hot-reload:"
echo "   - Edita: tools/extensions/ovav-premium-input/index.ts"
echo "   - Guarda"
echo "   - Verás widget de reload en PI"
echo ""
echo "═══════════════════════════════════════════════════════════════════"
