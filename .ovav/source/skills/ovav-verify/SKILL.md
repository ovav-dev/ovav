# OVAV Verify Mode — Skill

> **Versión:** 1.0 | **Fecha:** 2026-07-30 | **Lead:** Thavren

---

## Nombre
`ovav-verify`

## Descripción
Verification formal antes de cualquier claim de completitud. Evidence before assertions — siempre.

---

## Iron Law

```
NINGÚN CLAIM SIN FRESH VERIFICATION EVIDENCE
```

Si no ejecutaste el comando de verificación en ESTE mensaje → no puedes claim que pasa.

---

## Gate Function

Para CUALQUIER claim:

1. **IDENTIFY:** ¿Qué comando prueba este claim?
2. **RUN:** Ejecutar el comando COMPLETO (fresh)
3. **READ:** Leer output completo + exit code
4. **VERIFY:** ¿El output confirma el claim?
   - NO → Estado real con evidencia
   - YES → Claim WITH evidence
5. **ONLY THEN:** Hacer el claim

---

## Claims y Verifications

| Claim | Command | Required Evidence |
|---|---|---|
| "Tests pass" | `go test ./...` | Full output: 0 failures + exit 0 |
| "Tests pass" | `pnpm test` | Full output: 0 failures + exit 0 |
| "Build succeeds" | `go build ./...` | Exit 0 |
| "Build succeeds" | `pnpm build` | Exit 0 + dist/ generated |
| "Linter clean" | `go vet ./...` | 0 errors |
| "Linter clean" | `pnpm lint` | 0 errors |
| "Type check OK" | `go build` / `tsc --noEmit` | Exit 0 |
| "E2E passes" | `playwright test` | Full report |
| "Coverage ≥80%" | Coverage report | % por package |
| "Auth works" | `curl POST /auth/login` | 200 + token |
| "Auth rejects bad" | `curl POST /auth/login` (wrong pass) | 401 + error |

---

## Red Flags — STOP

- Usar "should", "probably", "seems to"
- Expresar satisfacción antes de verification ("Great!", "Perfect!", "Done!")
- Commit/push/PR sin verification
- Confiar en reports de agente
- Partial verification
- "Solo esta vez"

---

## Pre-Merge Checklist

```
[ ] go test ./... → 0 failures, exit 0
[ ] go build ./... → exit 0
[ ] go vet ./... → 0 errors
[ ] pnpm test → 0 failures
[ ] pnpm build → exit 0 + dist/
[ ] pnpm lint → 0 errors
[ ] playwright test → all pass
[ ] Coverage ≥80% en backend
[ ] Coverage ≥80% en frontend stores
[ ] No console errors en browser
[ ] Auth flow: register → login → logout
[ ] API: todos los endpoints responden 200/401/404 correctamente
```

---

## Metadata

- **Ubicación:** `.ovav/source/skills/ovav-verify/SKILL.md`
- **Skill ID:** `ovav-verify`
- **Trigger:** Antes de commit, antes de merge, antes de claim "completado"
- **Predecesor:** `ovav-build`
- **Sucesor:** `owd` (merge)
