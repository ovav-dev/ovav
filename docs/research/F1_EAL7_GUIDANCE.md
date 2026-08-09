# Common Criteria EAL7 — Guía de Rigor para OVAV

> F1.6 — Arquitectura de Seguridad
> Estándar: Common Criteria Evaluation Assurance Level 7 (EAL7)
> Adopción: Metodología aspiracional, sin certificación externa requerida
> **Material de laboratorio. La ruta real está en `IMPLEMENTATION_PLAN.md`.**

---

## Qué es EAL7

EAL7 es el nivel más alto de garantía en Common Criteria (ISO/IEC 15408). Requiere:

1. **Diseño formalmente verificado**: el modelo de seguridad se expresa en notación matemática
2. **Pruebas de penetración exhaustivas**: el sistema resiste ataques de un adversario con alto potencial
3. **Desarrollo estructurado y modular**: cada componente es verificable de forma independiente
4. **Análisis de canales encubiertos**: se identifican y mitigan fugas de información laterales
5. **Gestión de configuración estricta**: toda modificación al sistema es trazable y autorizable

---

## Cómo OVAV adopta EAL7

OVAV no busca certificación EAL7 externa. Adopta su **metodología** como guía de rigor:

| Principio EAL7 | Implementación OVAV |
|---|---|
| Diseño formalmente verificado | `tools/permissions/verify.py` — 5 propiedades matemáticas verificadas |
| Desarrollo modular | 6 capas F0 independientes + F1-F5 acopladas solo por contratos |
| Pruebas de penetración | `tools/validators/check_*.py` — 20+ validadores automáticos |
| Canales encubiertos | `tools/security/exfil_detector.py` — anti-exfiltración activa |
| Gestión de configuración | `tools/security/bootstrap_verifier.py` — hash chain verificable |
| Supply chain integrity | `tools/security/sbom.py` — SBOM con verificación de hashes |
| Secrets management | `tools/security/secrets_vault.py` — encrypted at rest, zeroed in memory |

---

## Métricas de Rigor

OVAV mide su proximidad a EAL7 con estas métricas:

| Métrica | Objetivo EAL7 | Estado OVAV |
|---|---|---|
| Cobertura de verificación formal | 100% de propiedades críticas | 5/5 verificadas ✅ |
| Cobertura de validadores automáticos | Todas las superficies | 26 validadores ✅ |
| Tiempo de detección de drift | < 5 minutos | Inmediato (on push) ✅ |
| Tiempo de respuesta a compromiso | < 1 minuto | Self-healing + lockdown ✅ |
| Secrets en disco | 0 en texto plano | Vault + sanitize ✅ |
| Bootstrap chain | Verificable y no repudiable | Hash chain ✅ |

---

## Propiedades Verificadas (F1.3)

1. **Consistencia**: evaluación determinista — mismas entradas → misma decisión
2. **Invariancia de seguridad**: operaciones críticas (sudo, path traversal, force-push) nunca permitidas
3. **No-circunvención**: `explicit_grant` no puede saltarse denies críticos
4. **Completitud**: 100% de combinaciones acción × operador × scope tienen resultado definido
5. **Aislamiento de operadores**: Eidren no puede escalar a permisos de Thavren

---

## Próximos pasos hacia EAL7

- [ ] Verificación formal de la cadena de bootstrap completa (F0.5)
- [ ] Análisis de timing side-channels en network_guard (F0.4)
- [ ] Pruebas de fuzzing en el motor Rego (F1.1)
- [ ] Model checking del flujo de delegación (F5)
- [ ] Auditoría de cobertura de anti-exfiltración (F0.6)

---

> **Principio**: EAL7 no es un checklist — es una disciplina de diseño. Cada componente de OVAV se pregunta: ¿puede esto ser verificado formalmente? Si la respuesta es no, se rediseña hasta que sea sí.
