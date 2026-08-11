# 🔧 OVAV Keybinding Repair Plan — Emergency Fix

**Date:** 2026-08-10  
**Priority:** CRITICAL (BLOCKER)  
**Status:** IN PROGRESS  

---

## 🚨 Problema Identificado

Los keybindings habituales están rotos en WezTerm + Fish shell. Causa raíz:

### **Root Cause:**
Archivo `~/.config/fish/conf.d/30-ovav-runtime-tools.fish` tenía bindings conflictivos:

```fish
bind \ee edit_command_buffer      # ← OK, pero sobrescribe sin verificar
bind \e\[1\;3D backward-word      # ← CONFLICTO con WezTerm Alt+Left
bind \e\[1\;3C forward-word       # ← CONFLICTO con WezTerm Alt+Right
```

Estos bindings de Fish interferían con los keybindings nativos de WezTerm Windows:
- `Alt+←` / `Alt+→` para navegación de palabras
- `Alt+h/j/k/l` para pane navigation
- Otros atajos del terminal

---

## ✅ Fix Aplicado

### **Cambios Realizados:**

1. **Removidos bindings conflictivos** (`backward-word`, `forward-word`)
2. **Agregada verificación de seguridad** antes de bind:
   ```fish
   bind --query \ee >/dev/null 2>&1; or bind \ee edit_command_buffer
   ```
3. **Sincronizado canonical source** en OVAV repo

### **Archivos Modificados:**
- `~/.config/fish/conf.d/30-ovav-runtime-tools.fish` (live config)
- `/home/braka/Systems/OVAV/config/fish/30-ovav-runtime-tools.fish` (canonical)

---

## 🎯 Plan de Ataque OVAV CONVERT

### **OVAV CONVERT — Intelligent Configuration Reconstruction System**

OVAV CONVERT es el subsistema encargado de **detección avanzada y reconstrucción automática** de configuraciones rotas. Funciona así:

#### **Arquitectura:**

```
┌─────────────────────────────────────────┐
│        OVAV CONVERT ENGINE              │
├─────────────────────────────────────────┤
│                                         │
│  1. DETECTION LAYER                     │
│     ├── Config file integrity check     │
│     ├── Syntax validation               │
│     ├── Dependency verification         │
│     └── Conflict detection              │
│                                         │
│  2. ANALYSIS LAYER                      │
│     ├── Root cause identification       │
│     ├── Impact assessment               │
│     ├── Stable version lookup           │
│     └── Risk scoring                    │
│                                         │
│  3. RECONSTRUCTION LAYER                │
│     ├── Backup current state            │
│     ├── Restore from stable baseline    │
│     ├── Apply safe patches              │
│     └── Validation & rollback if needed │
│                                         │
│  4. VERIFICATION LAYER                  │
│     ├── Functional testing              │
│     ├── Integration checks              │
│     ├── Performance validation          │
│     └── User acceptance confirmation    │
│                                         │
└─────────────────────────────────────────┘
```

---

## 📋 Implementation Plan

### **Phase 1: Detection Engine (Day 1)**

**Location:** `go-runtime/internal/convert/detector.go`

```go
type ConfigDetector struct {
    configPaths []string
    validators  map[string]ConfigValidator
}

func (d *ConfigDetector) Scan() ([]ConfigIssue, error) {
    // Scan all OVAV-managed configs
    // Check syntax, permissions, dependencies
    // Detect conflicts between layers (WezTerm → Fish → OVAV)
}
```

**Features:**
- Recursive scan de directorios de config
- Validación de sintaxis (YAML, JSON, Lua, Fish)
- Detección de conflictos entre sistemas
- Integrity checksums vs stable baseline

---

### **Phase 2: Analysis Engine (Day 2)**

**Location:** `go-runtime/internal/convert/analyzer.go`

```go
type ConfigAnalyzer struct {
    stableVersions map[string]string  // Known-good versions
    impactMatrix   map[string][]string // Dependencies
}

func (a *ConfigAnalyzer) Analyze(issue ConfigIssue) (AnalysisResult, error) {
    // Determine root cause
    // Assess blast radius
    // Find stable baseline version
    // Calculate risk score
}
```

**Features:**
- Root cause analysis con ML-like pattern matching
- Blast radius calculation (qué más se rompe)
- Lookup de versiones estables conocidas
- Risk scoring (0.0-1.0)

---

### **Phase 3: Reconstruction Engine (Day 3)**

**Location:** `go-runtime/internal/convert/reconstructor.go`

```go
type ConfigReconstructor struct {
    backupDir   string
    baselineDir string
    patchEngine *PatchEngine
}

func (r *ConfigReconstructor) Reconstruct(issue ConfigIssue) (ReconstructionPlan, error) {
    // Create backup of current state
    // Load stable baseline
    // Generate minimal patch
    // Validate patch safety
    // Apply with rollback capability
}
```

**Features:**
- Atomic backups antes de cambios
- Reconstrucción desde baseline estable
- Patch generation inteligente (mínimos cambios)
- Rollback automático si falla validación

---

### **Phase 4: CLI Integration (Day 4)**

**Location:** `go-runtime/cmd/ovav/convert_cli.go`

```bash
# Usage examples:
ovav convert scan                    # Detect issues
ovav convert analyze <issue-id>      # Deep analysis
ovav convert fix <issue-id> --auto   # Auto-reconstruct
ovav convert verify                  # Post-fix validation
ovav convert rollback <issue-id>     # Undo last fix
```

**Features:**
- Interactive mode con confirmación
- Dry-run mode (preview changes)
- Batch mode (fix all critical issues)
- Detailed reporting

---

## 🎯 Immediate Actions (This Session)

### **1. Verify Fix Applied:**

```bash
# Reload Fish config to test
fish -c "source ~/.config/fish/conf.d/30-ovav-runtime-tools.fish"

# Test keybindings manually:
# - Alt+Left/Right should move by word
# - Alt+h/j/k/l should navigate panes
# - Ctrl+E should edit command buffer
```

### **2. Commit Fix:**

```bash
cd /home/braka/Systems/OVAV
git add config/fish/30-ovav-runtime-tools.fish
git commit -m "$(cat <<'EOF'
fix(keybindings): remove conflicting Fish bindings that broke WezTerm shortcuts

Remove backward-word and forward-word bindings that conflicted with
WezTerm's native Alt+Left/Right navigation. Add safe binding check
for edit_command_buffer to avoid overwriting existing bindings.

This restores standard terminal navigation while keeping OVAV enhancements.

Fixes: Keybinding conflicts in WezTerm + Fish integration


💘 Generated with Crush



Assisted-by: Crush:qwen3.7-plus

EOF
)"
```

### **3. Deploy to All Environments:**

```bash
# Ensure sync across all machines
rsync -av /home/braka/Systems/OVAV/config/fish/30-ovav-runtime-tools.fish \
       ~/.config/fish/conf.d/30-ovav-runtime-tools.fish

# Or use OVAV deploy mechanism when ready
```

---

## 📊 Success Criteria

| Metric | Target | Current | Status |
|--------|--------|---------|--------|
| Alt+Left/Right works | ✅ Yes | ❌ Broken | 🔄 Fixing |
| Alt+h/j/k/l pane nav | ✅ Yes | ❌ Broken | 🔄 Fixing |
| Ctrl+E edit buffer | ✅ Yes | ⚠️ Overwritten | 🔄 Fixing |
| No duplicate binds | ✅ Zero | ⚠️ Multiple | 🔄 Fixing |
| Config loads clean | ✅ No errors | ⚠️ Warnings | 🔄 Fixing |

---

## 🚀 Next Steps After This Fix

1. **Implement OVAV CONVERT Phase 1** (Detection Engine)
   - Build automated scanner
   - Integrate with `ovav status`
   - Alert on config drift

2. **Add Monitoring**
   - Track config file changes
   - Alert on syntax errors
   - Log binding conflicts

3. **Create Baseline Repository**
   - Store known-good configs
   - Version each stable state
   - Enable instant rollback

4. **User Documentation**
   - Document all OVAV keybindings
   - Troubleshooting guide
   - FAQ for common issues

---

## 💡 Prevention Strategy

Para evitar que esto vuelva a pasar:

1. **Pre-commit hooks** que validan configs antes de commitear
2. **CI pipeline** que testa keybindings en sandbox
3. **Config diff alerts** cuando algo cambia
4. **Automated rollback** si tests fallan post-deploy
5. **User feedback loop** para reportar issues rápido

---

*Generated by OVAV Autonomous Potentiation System*  
*Timestamp: 2026-08-10T23:05:00-05:00*
