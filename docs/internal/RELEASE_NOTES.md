# OVAV Release Notes

<!-- OVAV_V2_0_0_PLAN_ACTIVATED -->
## v2.0.0 — Plan Maestro Activado

**2026-06-10** — Thavren (Platform Engineering Lead) con CEO Alexander Salvador

### Nueva arquitectura
- **5 capas segmentadas** con review gates obligatorios (RG1-RG5):
  - C1 Foundation Hardening — sidecar tracking, protección mecánica push, defensa auto-immune
  - C2 Cross-LEAD Governance — area_scope filter, Signal Bus, tool registry version
  - C3 Tooling & Developer XP — CLI compacta, credenciales por proyecto, ovav update, routing inteligente
  - C4 Intelligence & Autonomy — sidecar propio, memoria inteligente, collective intelligence
  - C5 Production Hardening — QA Automation Layer, multi-plataforma, instalación real
- **QA Squad activado**: Clara (QA Lead) + Diego (automation) + Pablo (code review)
- **35 ideas del lab mapeadas** a capas. 7 hallazgos de arquitectura absorbidos.
- **Principio rector**: toda herramienta se prueba con datos reales (CE-005). Protecciones mecánicas, no textuales.
- C1.3.1 completado: identity packets auto-regenerados por sesión.

### Documents updated
- `IMPLEMENTATION_PLAN.md` — línea temporal única v1.0.0 → v2.4.0 (documento histórico, reemplazado por caps.yaml)
- `current_authority_contract.yaml` — active_phase: v2.0.0 C1 (documento histórico, reemplazado por caps.yaml)
- Identity packets ahora en `derived_artifacts.yaml` (categoría `identity_surface`, deprecado en v2.0)

<!-- OVAV_V2_0_0_PLAN_ACTIVATED_END -->

<!-- OVAV_V1_0_0_RELEASE -->
## v1.0.0 — Release Final

**2026-06-07**

### Completado
- **Fases A-F**: 597 commits, 8 release candidates, 0 fallos.
- **S1-S12-E**: Gobernanza Autónoma completa (OVAV Presence, Governor Authority, Lead Accountability, Research Escalation).
- **76 validadores**, 388 harnesses, 69 herramientas de runtime.
- **L0-L7 Full Stack**: Identity Packet Compiler → Feedback Loop operativo.
- **Integrity Mesh F0-F4**: Living Integrity 100%, Self-Diagnosis 35/35.
- **Economy Engine**: 8 modelos, 5 proveedores, $10/sesión, $200/mes.
- **Memory Governor**: capsule-bound, F5-gated (649 loc).
# SIS removed — worktree automation system eliminated 2026-06-11
- **Behavioral Directives**: 26 reglas activas por sesión.
- **OpenCode Plugins**: ovav-monitor.js, ovav-status.js + 7 comandos.

### Branch
- `main` (v1.0.0 tag) → `develop` (post-release)

### Safety posture
- Raw git push, force push y force delete prohibidos.
- HTTPS-only push vía `ovav_git_push_gate.py`.
- Sin remote configurado por el paquete.
<!-- OVAV_V1_0_0_RELEASE_END -->

<!-- OVAV_RC9_PLUS_RELEASE_NOTES_START -->
## Final Launch Verification — Knowledge Compiler Alignment

**2026-06-01**

### Sistemas activos
- **Integrity Mesh**: 18 validators, 0 rotos. 9 auto-acciones al iniciar sesión.
- **State Sync Engine**: ❌ ELIMINADO 2026-06-10. Git HEAD es la fuente de verdad inmutable — sin sync engines paralelos.
- **Behavioral Directives**: 9 reglas con system prompt injection (modelo Cursor/Claude).
- **Memory Governor**: capsule-bound, F5-gated.
- **Trigger Engine**: auto_triggers.yaml ya no es decorativo.
- **Security**: 3 brechas críticas cerradas (gate self-protection, HEAD integrity, session context guard).
- **Runtime Safety Governor**: repair corridor cerrado; startup recuperado de 8-15min a ~2s.
- **Strategic route**: Knowledge Compiler P0.2 implementado; documentación y autoridad se consolidan antes de expandir hacia Sistema Nervioso Vivo.
- **Git Transport**: HTTPS-only enforcement con gate anti-mezcla SSH/HTTPS en ovav_git_push_gate.
- **Validation**: filtro de severidad en validate_all; output inteligente que distingue bloqueos de telemetría.
- **Adaptive CPU**: threshold con media móvil; picos de trabajo legítimo no bloquean, sobrecarga real sí.
- **Evaluation Pipeline**: auto-genera packet del diff actual con rollback plan; 12/12 triggers verdes.

### Safety posture
- No remote is configured by this package.
- No push is performed by this package.
- Runtime plans/reports/backups are generated under ignored `.ovav/runtime/`.
- Global config writes, plugin install, live Engram, MCP/A2A and production/global-ready claims remain blocked.
- `raw git push`, force push y force delete están prohibidos en todas las superficies.
<!-- OVAV_RC9_PLUS_RELEASE_NOTES_END -->

## Historial

<!-- OVAV_RC3_RELEASE_NOTES_START -->
### v1.0.0-rc3 — CLI Mother Cockpit RC (Mayo 2026)

- Guided cockpit navigation with state-aware recommendations.
- Real check-only actions for setup, sync, security, recovery and update.
- Plan artifacts for setup/sync/security/recovery/update.
- Execution gateway with explicit `--apply --consent --accept-risk` gating.
- Source-local managed backup and rollback.
- OpenCode surface manager and repair-plan generation.
- Public export security gate.
- Practical smoke and fresh archive/dogfood smoke checks.
<!-- OVAV_RC3_RELEASE_NOTES_END -->

## Version

See `VERSION`.

## Final Launch Verification scope

This verification scope keeps OVAV in final launch verification with Integrity Mesh active, auto-sync enabled, security breaches closed, and Runtime Safety Governor active. Fases 10-12 (Capability Lifecycle, Approval Router, Capability Market) are re-planned post-Knowledge Compiler.
- Compatibility matrix.
- Launch pack validator.
- Service-governance and canonical-review validation chain.

## Current service areas

- OVAV Platform Engineering — Thavren.
- OVAV Research Intelligence — Eidren.

## Known limits

- Global install/apply is governed and not default.
- BUILD 18 launch readiness requires validator pass before release claim.
- Remote push is not assumed; local Git workflow remains default unless configured and approved.

<!-- OVAV_RC3_PRESENTATION_POLISH -->
## v1.0.0-rc3 presentation polish

- Adds GitHub-facing README structure and brand assets under `assets/readme/`.
- Adds public docs for CLI, intended usage, architecture, governance, roadmap and release checklist.
- Adds `.github` issue templates, PR template and CI workflow.
- Adds `ovav repo-check` and wires it into release package validation.

<!-- OVAV_RC4_PREMIUM_FIRST_RUN_COCKPIT -->
## v1.0.0-rc4 — premium first-run cockpit

- Adds one-command public install through `install.sh`.
- Makes `ovav` the primary cockpit entrypoint.
- Adds Launch, Tailor, Preview, Recovery, Update and Control Room navigation.
- Adds workspace separation docs for private engineering, public release and installed user source.
- Adds install smoke in a temporary HOME to detect private-root dependency.
- Keeps managed workstation writes gated behind plan, backup, consent and rollback.

<!-- OVAV_RC5_PREMIUM_INSTALLER_COCKPIT_UX -->
## RC5 — premium installer and cockpit UX

- Improves private beta installation output with a cleaner visual setup flow.
- Hides internal paths by default and moves diagnostics to Control Room / `--verbose`.
- Restores a stronger cockpit logo, cards, responsive layout and human labels.
- Adds RC5 visual smoke coverage using a temporary HOME sandbox.

<!-- OVAV_RC5_KEYBOARD_NAVIGATION -->
## RC5 cockpit navigation polish

- Adds keyboard navigation for the cockpit home surface.
- Supports ↑/↓, j/k, Enter, 1-6 direct selection, q and Esc.
- Keeps non-interactive command output stable for validators and automation.

<!-- OVAV_RC5_COCKPIT_REVIEW_STATE_RETURN -->
## RC5 cockpit review-state behavior

- Visual cockpit screens remain open even when the system needs review.
- Machine gates still fail through JSON, plan artifacts, smoke, release-check and validators.
- This keeps OVAV usable as a living cockpit instead of a dead failing prompt.


<!-- OVAV_RC5_INSTALL_REVIEW_NONFATAL -->
## RC5 installer review behavior

- Installer safety review is advisory for user experience.
- If review needs attention, installation still finishes and the cockpit guides the user.
- Strict failures remain enforced by JSON gates, smoke, fresh-smoke, release-check and validators.


<!-- OVAV_RC5_ARROW_KEY_STABILITY -->
## RC5 arrow-key navigation stability

- Stabilizes arrow-key decoding for WSL, Zellij and terminal multiplexers.
- Home no longer exits on stray/incomplete ESC bytes.
- q remains the intentional quit key; arrows move the active selector.


<!-- OVAV_RC5_LIVING_MULTI_LEVEL_COCKPIT -->
## RC5 living multi-level cockpit

- Replaces prompt-style navigation with a living selector.
- Numbers move focus; Enter opens the selected option.
- Each top-level option opens a second-level panel with action details.
- Footer navigation is grouped and aligned for terminal use.
- CLI logo now uses a terminal-safe OVAV brand mark.


<!-- OVAV_RC5_KEYBOARD_SMOKE_LIVING_FOOTER -->
## RC5 keyboard smoke alignment

- Updates the keyboard smoke expectation from the old inline help text to the new grouped Navigation footer.
- Keeps PTY validation for arrow-down, Enter, Tailor open and clean quit.

<!-- OVAV_RC5_CURSES_COCKPIT_ENGINE -->
## RC5 curses cockpit engine

- Replaces manual ANSI input parsing with a curses-first interactive engine.
- Adds stable arrow-key navigation for WSL, Zellij and terminal multiplexers.
- Keeps a non-interactive renderer for validators, CI and script output.
- Numbers move focus; Enter opens the selected option.
- Live system detection shows workspace, branch, git state, shell, platform, terminal, tools and installed OVAV.

<!-- OVAV_RC5_DYNAMIC_RAILS_COCKPIT -->
## RC5 dynamic rails cockpit

- Removes heavy boxes in favor of centered rails and compact sections.
- Replaces repeated header wording with a cleaner OVAV brand block.
- Keeps curses-first navigation and live environment detection.
- Makes every entered area redraw the full surface around that area.
- Keeps numbers as focus movement and Enter as the explicit open action.

<!-- OVAV_RC5_SEGMENTED_PREMIUM_COCKPIT -->
## RC5 segmented premium cockpit

- Implements the three-part cockpit structure: brand/header, dynamic content, compact navigation rail.
- Removes bulky boxes from the interactive surface and uses centered rails with stronger spacing.
- Keeps curses-first navigation with arrows, j/k, numeric focus and Enter-to-open.
- Makes each area redraw the complete content surface around its own actions and context.
- Adds friendly quit confirmation instead of cutting the interaction abruptly.

<!-- OVAV_RC5_ANSI_SEGMENTED_COCKPIT -->
## RC5 ANSI segmented cockpit

- Restores a real centered ANSI OVAV wordmark instead of a small broken symbol.
- Implements the requested three-part segmentation: ANSI header, dynamic content surface, compact navigation rail.
- Keeps curses-first navigation with arrows, j/k, numeric focus and Enter-to-open.
- Makes each area redraw the full middle surface around its own actions and context.
- Adds friendly close confirmation instead of cutting the interaction abruptly.

<!-- OVAV_RC5_GROUPED_COCKPIT_RAILS -->
## RC5 grouped cockpit rails

- Removes the redundant selected-summary section from the root surface.
- Expands the navigable content area and groups actions by SETUP, GOVERN and SYSTEM.
- Keeps the ANSI OVAV wordmark, dynamic area surfaces and compact navigation rail.
- Preserves curses-first navigation: arrows, j/k, numeric focus and Enter-to-open.

<!-- OVAV_RC5_SPACIOUS_GROUPED_COCKPIT -->
## RC5 spacious grouped cockpit

- Adds breathing room between SETUP, GOVERN and SYSTEM groups.
- Removes noisy workspace/branch status from the primary surface.
- Keeps only useful live indicators: git cleanliness, shell, workspace mode and installed signal.
- Expands the navigable area without adding redundant content sections.

<!-- OVAV_RC5_ADAPTIVE_SURFACE_OPTIMIZATION -->
## RC5 adaptive surface optimization

- Applies one visual rule across root, area and detail screens: brand, signal chips, adaptive content, semantic navigation rail.
- Replaces noisy workspace/branch status with intelligent signal chips: readiness, install/source state, tool coverage, platform and surface.
- Removes long underscore-like separators and uses soft rails instead.
- Makes the navigation rail semantic and color-coded: move, focus, open/detail, back and close.
- Preserves curses-first navigation and friendly close confirmation.

<!-- OVAV_RC5_FULL_SURFACE_BEHAVIOR -->
## RC5 full cockpit surface behavior

- Applies the same premium surface rule to root, area and action-result screens.
- Every top-level area now opens a real area surface with navigable actions.
- Enter on an area action now opens a result surface backed by live read-only checks or concrete system signals.
- Keeps the semantic navigation rail, signal chips, soft separators and friendly close confirmation.

<!-- OVAV_RC5_REAL_OPTION_SURFACES -->
## RC5 real option-specific cockpit surfaces

- Replaces generic area detail with option-specific preview panels for every navigable action.
- Enter on an action opens a concrete read-only result surface: live environment, git state, tool detection, repo/security/surface gates, backup/rollback/update plans, or exact diagnostics.
- Keeps the same premium visual rule across root, area and result screens.
- Clarifies that RC5 cockpit is operational for inspection and planning; destructive/apply paths remain consent-gated.

<!-- OVAV_RC5_KEYBOARD_SMOKE_FULL_SURFACE_REPAIR -->
## RC5 keyboard smoke full-surface repair

- Stabilizes the keyboard smoke by forcing a 140x40 PTY size.
- Uses deterministic numeric focus instead of relying on arrow timing during automated smoke.
- Verifies root -> OVAV Tailor -> OpenCode action -> concrete read-only result -> friendly close.
- Keeps the user-facing cockpit behavior unchanged.

<!-- OVAV_RC5_STATIC_HOME_SMOKE_CONTRACT -->
## RC5 non-interactive smoke contract

- Preserves the legacy `Home` marker in non-interactive/static output for `bin/ovav smoke --json`.
- Keeps the interactive cockpit clean; the `Home` marker is not reintroduced into the curses UI.
- This is a compatibility repair, not a visual UX regression.

<!-- OVAV_RC6_CLEAN_INSTALL_EXPERIENCE -->
## RC6 clean install experience

- Replaces noisy install output with a quiet premium path.
- Hides git clone/fetch/pull internals unless `--verbose` is explicitly used.
- Keeps install verification quiet: repo-check and smoke run without dumping JSON.
- Adds `tools/cli/ovav_clean_install_smoke.py` to enforce a clean install contract.
- Maintains `--no-launch`, `--source-url`, `--channel`, `--source-dir`, `--bin-dir`, and `--verbose`.

<!-- OVAV_RC6_SKIP_SOURCE_REFRESH_CONTRACT -->
## RC6 skip-source-refresh compatibility

- Restores `--skip-source-refresh` for local smoke/dogfood tests.
- In skip mode, the installer uses the current checkout as the source payload.
- Keeps the public/user install path quiet and premium by default.
