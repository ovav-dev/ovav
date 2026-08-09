# ISSUE-2026-0802-OWS-BIG-FILES-GAP

**Severity**: MEDIUM (workflow blocker)
**Status**: FIXED
**Date**: 2026-08-02
**Detected during**: owd merge of feature/ciclo-completado
**Lead**: Thavren (Platform Engineering)

---

## Problema

OWD Stage 3 (Forbidden Files) tenía un gap: el check BIG_FILES escaneaba TODOS los archivos del worktree, no solo los del branch actual.

### Error
```
[S3/6] Forbidden files:
  FAIL: FORBIDDEN: 0 BIG_FILES: 1
BLOCKED: stages failed (S 3)
```

### Root Cause
`git ls-files --cached --others --exclude-standard` lista TODOS los archivos heredados del parent.

### Solución
`ow_forbidden_files()` ahora usa `git diff parent..HEAD` para escanear solo archivos del branch.

---

## Archivos modificados

| Archivo | Cambio |
|---|---|
| bin/ovav-owlib.sh | ow_forbidden_files() reescrito — solo diff del branch |

---

## Commit

`44721d43` fix(ows): ow_forbidden_files — scan branch diff only

---

## Lección

OWS debe escanear solo el diff del branch, nunca el worktree completo.
