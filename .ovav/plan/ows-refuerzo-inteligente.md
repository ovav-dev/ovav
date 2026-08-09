# OVAV Worktree System — Plan de Refuerzo Inteligente

**Fecha**: 2026-07-16
**Owner**: Thavren (Platform Engineering)
**Status**: ANÁLISIS + PLAN
**Trigger**: Braka detectó regresión — cada comando perdió su lógica advanced original

---

## 1. Análisis Histórico — qué tenía el OWS original

**Evidencia encontrada:**
- `docs/worktree/OWS_USER_GUIDE.md` (commiteado en HEAD, blob `46394b7`) — fuente canónica del design original
- `go-runtime/internal/worktree/` (referenciado por OWS_USER_GUIDE) — código Go del runtime
- 4 handoffs `OWS-review-{dante,eidren,kenji,uriel}-*.md` — reviews comprehensivos de múltiples especialistas

**Comandos según USER GUIDE** (11 total) — capacidades que tenía el original:

| Cmd | Funcionalidad original |
|---|---|
| `owc` | 12 perfiles (feature/fix/hotfix/release/docs/refactor/spike/research/migration/enterprise/emergency/patch) + conflict prediction via `git merge-tree` + auto-cd via fish alias + hooks + git rerere |
| `owd` | 3 modos (auto/branch/path) + 4 niveles compliance (quick/standard/strict/maximum) + 6 stages pipeline: conflict-pred / secrets-sweep / forbidden-files / go-vet+fmt+test / hygiene / gpg-signatures + reviewer required (strict+) + merge local→develop + cleanup + push |
| `owl` | list con conflict predictions vs develop + `--history` audit trail + `--json` format |
| `owv` | go vet + gofmt + go test + hygiene + validators |
| `ows` | fetch + maintenance + prune + `--rebase` + `--full` modes |
| `owclean` | huérfanos + spike TTL (48h) auto-cleanup |
| `owx` | route changes + `--target` flag (no cherry-pick simple) |
| `owa` | abort con cleanup |
| `owr` | rescue broken worktrees |
| `owlk` | lock con razón auditada |
| `owm` | move con validaciones |

**Convención ubicación**: SIEMPRE `.ovav/worktrees/<nombre>` (NO separación por tipo) — workspace aislado por branch-name completo.

---

## 2. Diagnóstico de Regresión — qué tengo vs qué tenía

**Estado actual (`bin/ovav-ow-runtime.sh` v2.0)**:

| Cmd | Features actuales | Features originales (user guide) | GAP |
|---|---|---|---|
| `owc` | detect 10 tipos, carpeta `../worktrees/<type>/<name>` | 12 perfiles, conflict prediction, `.ovav/worktrees/<name>`, hooks, git rerere | **MEDIO** (falta perfil spike/research/enterprise/emergency, conflict prediction, hooks, fixer ubicación) |
| `owd` | imprime hint "git push" | 6 stages pipeline + 4 compliance levels + 3 modes + reviewer | **CRÍTICO** (falta stages, compliance, modes, reviewer) |
| `owl` | list simple | list + conflict predictions + `--history` + `--json` | **MEDIO** |
| `owv` | llama `ovav status` | go vet + fmt + test + hygiene + validators específicos | **BAJO** |
| `ows` | prune + list | fetch + maintenance + prune + `--rebase` + `--full` | **MEDIO** |
| `owclean` | prune | 48h TTL spike cleanup + prunable detect | **BAJO** |
| `owx` | cherry-pick simple | route changes | **BAJO** |
| `owa/owr/owlk/owm` | básico stubs | equivalente | **OK** |
| `owprep/owsuggest` | nuevo (cache project analysis) | no existía → agregado | **NUEVO ✓** |

**Resumen del GAP:**
- **CRÍTICO**: `owd` perdió todo el validation pipeline (6 stages)
- **MEDIO**: `owc` carece de conflict prediction; `owd` carece de compliance levels
- **BAJO**: few polish items (history/JSON flags)
- **NUEVO ✓**: análisis de proyecto (`owprep`) — feature mejorada sobre original

---

## 3. Causa raíz de la regresión

**Reconstrucción cronológica:**
1. **2026-07-16 02:33-03:17** (incidente side-channel injection): bt-sys-react ejecutó 4 scripts que crearon:
   - `bin/ovav-{owa,c,d,l,m,s,v,x,owclean}` (root-owned, lógica limitada)
   - `clients/setup-ows-consumer.sh` (versión inyección)
2. **2026-07-16 ~16:00** (cleanup): revertí OVAV, pero LOS SCRIPTS INYECTADOS FUERON ELIMINADOS — los OWS avanzados NUNCA se commiteaban (vivían solo en filesystem)
3. **2026-07-16 ~16:10** (instalación nueva): `bin/ovav-consumer install` regeneró stubs BÁSICOS con un solo caso por comando
4. **2026-07-16 ~17:00** (mejora): agregué branch type detection + smart folder + conventional commits — pero aún muy lejos del original

**El diseño original de Braka vivía solo en su filesystem local, sin commit.** El USER GUIDE en git era documentación; el código real de los OWS shells nunca llegó al repo.

**Eliminación accidental:** Al limpiar el side-channel injection, borré archivos que parecían "injection" pero que en realidad eran la versión local (mínima) de los OWS originales de Braka.

---

## 4. Plan de Refuerzo Inteligente (PRIORIZADO)

### FASE A — Restaurar fidelidad al USER GUIDE original (ALTA PRIORIDAD)
**Esfuerzo**: ~3 días
**Entregable**: `bin/ovav-ow-runtime.sh` v3.0 con comportamiento idéntico al original

1. **owc v3.0**:
   - Carpeta FIJA `.ovav/worktrees/<safe-name>` (no por tipo)
   - 12 perfiles + conflict prediction (`git merge-tree --write-tree <local> <remote> HEAD`)
   - Branch base auto-detectada (develop / main según tipo)
   - Spike TTL config (48h)
   - Post-create hooks desde `.ovav/hooks/post-create`
   - Branch NUNCA pusheada al remote (zero noise)
   - Audit log estructurado con branch_type + base_branch + conflict_status

2. **owd v3.0 (EL CRÍTICO)**:
   - 3 modos: `owd` (auto) / `owd <branch>` / `owd <path>`
   - 4 compliance levels: `--compliance quick|standard|strict|maximum`
   - 6 stages pipeline:
     - **S1**: `git merge-tree` conflict prediction
     - **S2**: secrets sweep (27 regex patterns — copiar del original)
     - **S3**: forbidden files (.env, .pem, .key, binarios grandes >10MB)
     - **S4**: go validation (vet + fmt + test) OR multi-stack validation (TS/Python/Rust según stack)
     - **S5**: hygiene scan (.DS_Store, archivos >5MB, git config)
     - **S6**: GPG signatures (solo strict+, requiere en maximum)
   - Reviewer `--reviewer "<name>"` (strict+/maximum)
   - Output gate: `STAGES FAILED: S2, S4` con detail por stage
   - Solo si PASS todos → merge local + push develop + cleanup
   - Audit log con cada stage (pass/fail + count + duration)

3. **owl v1.1**:
   - Predicción de conflictos vs develop integrada al output
   - `--history` flag
   - `--json` flag para tooling

4. **ows v1.1**:
   - `git fetch origin` + rebase (opcional `--rebase`) + full sync (`--full`)
   - Stack-aware commands
   - Maintenance cycles (git gc, fsck)

5. **owclean v1.1**:
   - Spike TTL cleanup (48h desde creación)
   - Prunable detection + force

### FASE B — Nuevos comandos advance (MEDIA PRIORIDAD)
**Esfuerzo**: ~2 días

- `owprep` / `owsuggest` — análisis de proyecto (ya existen, refinar)
- `owx --target <T>` — route change tracking (no cherry-pick)
- Audit dashboard: `ovav-consumer audit <id> --tail` con filter por command

### FASE C — Integración con Worktree Skill + Hooks (BAJA PRIORIDAD)
**Esfuerzo**: ~2 días

- Publicar `.ovav/hooks/post-create` con stack-aware setup
- Fish alias en `bin/setup-fish.sh`
- Documentar todo en `.ovav/docs/OWS_REINFORCEMENT.md`

### FASE D — Verificación end-to-end
**Esfuerzo**: ~1 día

- Tests por perfil (12 perfiles)
- Tests por compliance level (4 levels)
- Test edge case: merge conflict detectado y abort
- Test edge case: spike TTL expirado cleanup

---

## 5. Estimación total

| Fase | Esfuerzo | Valor |
|---|---|---|
| A — Restaurar fidelidad | 3 días | ALTO — recupera intent original |
| B — Nuevos commands | 2 días | MEDIO — agrega capabilities |
| C — Integración | 2 días | BAJO — UX polish |
| D — Tests | 1 día | REQUERIDO — gates |
| **TOTAL** | **8 días** | |

---

## 6. Decisión CEO

Braka — ¿qué priorizo?

| Opción | Entregable |
|---|---|
| **α) Solo `owc v3.0` + `owd v3.0`** (Fase A) | Restaurar los 2 comandos más críticos con conflict prediction + 6 stages validation |
| **β) Fase A completa** | Los 5 comandos críticos (owc/owd/owl/ows/owclean) v3.0 |
| **γ) Fase A + B** | Todo + `owx --target` + audit dashboard |
| **δ) Full 8 días** | A+B+C+D completo |

Mi recomendación: **β)** — Fase A completa porque los 5 comandos están interrelacionados (`owd` invoca `owl` para conflict predictions; `owclean` lo dispara `owd`; etc.).

¿Confirmás β (Fase A completa) o priorizás diferente?
