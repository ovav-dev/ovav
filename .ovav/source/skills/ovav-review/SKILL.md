# OVAV Code Review — Skill

> **Versión:** 1.0 | **Fecha:** 2026-07-30 | **Lead:** Thavren

---

## Nombre
`ovav-review`

## Descripción
Code review con SHA-based diff targeting. Reviewer recibe contexto preciso — nunca el history de la sesión completa.

---

## Cuando Request Review

**Obligatorio:**
- Después de cada task en subagent-driven development
- Después de completar major feature
- Antes de merge a main

**Opcional pero valioso:**
- Cuando estás atascado (fresh perspective)
- Antes de refactoring (baseline check)
- Después de fix complejo

---

## Two-Stage Review Gate

### Stage 1: Spec Compliance
El diff se compara contra EL SPEC, no contra opiniones.

Preguntas:
- ¿El código hace lo que el spec dice?
- ¿Hay silent omissions (cosas que el spec dice que se hacen pero no)?
- ¿Los tests cubren los casos del spec?

### Stage 2: Code Quality
- ¿Está tipado correctamente?
- ¿Sigue los patterns del codebase?
- ¿Tiene tests?
- ¿Es el código mantenible?
- ¿Hay security issues?

---

## Como Request Review

### 1. Get git SHAs
```bash
BASE_SHA=$(git rev-parse HEAD~1)  # or origin/main
HEAD_SHA=$(git rev-parse HEAD)
```

### 2. Dispatch reviewer subagent
El reviewer recibe:
- SHA range (BASE → HEAD)
- Spec sections relevantes
- Scope boundary

### 3. Act on feedback
- **Critical:** Fix inmediatamente
- **Important:** Fix antes de proceed
- **Minor:** Note para después
- **Wrong reviewer:** Push back con razón

---

## Review Checklist

### Spec Compliance
- [ ] Cada feature del spec está implementada
- [ ] Cada endpoint del API spec está conectado
- [ ] Cada test case del spec tiene test
- [ ] No hay silent omissions

### Code Quality
- [ ] Tipos correctos (TypeScript / Go)
- [ ] No `any` en TypeScript
- [ ] Error handling completo
- [ ] Tests para edge cases
- [ ] No hardcoded values (config en env)
- [ ] Security: input validation, SQL injection prevention
- [ ] Performance: no N+1 queries, no blocking operations

### Testing
- [ ] Unit tests para lógica de negocio
- [ ] Integration tests para API handlers
- [ ] E2E tests para user flows
- [ ] Coverage ≥80%

---

## Metadata

- **Ubicación:** `.ovav/source/skills/ovav-review/SKILL.md`
- **Skill ID:** `ovav-review`
- **Trigger:** Después de cada task, antes de merge
- **Predecesor:** `ovav-build`
- **Sucesor:** `ovav-verify`
