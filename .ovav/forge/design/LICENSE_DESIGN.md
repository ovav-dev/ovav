# OVAV License — Diseño de Membresía

> Fase C4 · Diseño · 2026-06-04
> Estado: 📐 design — no implementado

---

## Propósito

Sistema de feature-gating por membresía para OVAV. Sin licencia activa → degradar features visuales y de proyección. El core de gobernanza (validadores, seguridad, runtime) siempre funciona.

---

## Principios

1. **Core siempre activo** — validadores, seguridad y runtime no dependen de licencia
2. **Features visuales degradables** — tema, dashboard avanzado, proyección multi-target
3. **Licencia local** — archivo firmado en `.ovav/license.key`, no validación externa
4. **Período de gracia** — 7 días tras expiración antes de degradar

---

## Tiers propuestos

| Tier | Features | Core |
|------|----------|------|
| **Free** | Tema básico, 1 target (OpenCode), dashboard compacto | ✅ Completo |
| **Pro** | Tema completo, 4 targets, dashboard avanzado, SNV Dashboard | ✅ Completo |
| **Enterprise** | Pro + staging pipeline externo, personnel multi-lead, priority support | ✅ Completo |

---

## Mecanismo

```
.ovav/license.key  ← archivo firmado (JWT o HMAC)
     ↓
tools/license/validator.py  ← verifica firma, expiración, tier
     ↓
Feature gating en:
  - tools/visual/project_opencode_visual.py  (tema)
  - .ovav/forge/pipeline.py                  (targets)
  - tools/ovav_dashboard.py                  (dashboard)
```

---

## Reglas

- Sin licencia: core funciona, features visuales usan defaults
- Licencia expirada: 7 días de gracia, luego degradación
- Licencia inválida: mismo comportamiento que sin licencia
- Nunca bloquear el core de gobernanza

---

## Dependencias

- C5 (staging pipeline) — para distribuir licencias
- I-001 (Sistema Visual) — para feature-gating visual
- Post-Fase C

---

*Diseño. No implementar hasta que el staging pipeline (C5) esté validado por el CEO.*
