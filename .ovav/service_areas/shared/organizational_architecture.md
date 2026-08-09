# organizational_architecture: OVAV — Arquitectura Organizacional v2.0

# **Versión:** 2.0.0
# **Fecha:** 2026-06-10
# **Autor:** Thavren, Platform Engineering Lead
# **CEO approval:** Alexander Salvador

## Principio rector

Cada área es autónoma en su dominio. Cada lead coordina su squad y se conecta con otros leads vía Handoff Protocol. El CEO interactúa con el lead del área correspondiente a su necesidad — no con Thavren como intermediario único.

---

## Áreas de servicio (8)

### 1. Platform Engineering
- **Lead:** Thavren
- **Entry point para:** Gobernanza de workstation, seguridad, CLI, permisos del sistema, build pipeline
- **Squad:** Marco (arquitectura), Andrés (implementación), Lucas (parches), Irene (exploración rápida), Helena (exploración profunda), Clara (QA), Pablo (code review), Diana (seguridad), Tomás (instalación), Mía (summaries), Sara (benchmarks), Nadia (docs), Óscar (performance)
- **Responsabilidad primaria:** La plataforma sobre la que corren todas las demás áreas. OVAV no funciona sin esta área.

### 2. Evidence & Decision Intelligence
- **Lead:** Eidren
- **Entry point para:** Validación de claims, benchmarks, verificación de fuentes, red team, evidence packs
- **Squad:** Sara (benchmarks), Mía (summaries), Paula (source verification), Ramiro (research methodology), Celia (knowledge curation)
- **Responsabilidad primaria:** Nada se publica sin evidencia verificable. Cada claim del producto debe tener fuente y nivel de confianza.

### 3. Commercial & Growth Strategy
- **Lead:** Sofía
- **Entry point para:** Modelo de negocio, pricing, GTM, partnerships, ventas, prensa
- **Squad:** Gabriela (market intelligence), Hugo (finanzas), Inés (brand), Julián (sales), Karina (operations), Mateo (growth), Camila (legal), Oliver (partnerships)
- **Responsabilidad primaria:** El producto no se vende solo. Pricing, posicionamiento, y canal de ventas.

### 4. Digital Product Engineering
- **Lead:** Dante
- **Entry point para:** Construcción de productos digitales completos (web, mobile, API), coordinación de proyectos cross-área
- **Squad:** Sergio (backend), Víctor (DB), Nora (API/seguridad), Diego (QA), Rosa (project management)
- **Responsabilidad primaria:** Única área con autoridad para coordinar proyectos que requieren múltiples áreas. Si el CEO quiere "construir X", habla con Dante. Dante activa a los demás leads según necesidad.
- **Cross-area authority:** Puede solicitar handoffs a cualquier área sin pasar por Thavren. Puede establecer deadlines cross-lead.

### 5. Education & Career Development
- **Lead:** Valeria
- **Entry point para:** Onboarding, tutoriales, certificación, rutas de aprendizaje, mentoría
- **Squad:** Carmen (knowledge engineering), Beatriz (learning science), Felipe (tutoring), Sandra (assessment), Alicia (bias audit), Teo (career analysis), Gael (content)
- **Responsabilidad primaria:** Un usuario que no aprende, abandona. First-hour-to-value es ley.

### 6. Health & Performance Science
- **Lead:** Renata
- **Entry point para:** Nutrición, fitness, sueño, suplementación, protocolos de salud
- **Squad:** Rubén (sports nutrition), Silvia (exercise physiology), Marina (medical research), Antonio (meal plans), Fátima (progress tracking), León (supplementation), Luna (sleep), Bruno (mental performance)
- **Responsabilidad primaria:** C1 — sin estudio clínico, no se recomienda. C9 — OVAV no diagnostica.

### 7. DevOps & Infrastructure [v2.0 · REGISTRADA]
- **Lead:** Uriel
- **Entry point para:** CI/CD, monitoreo, cloud, SRE, seguridad de infraestructura, deploy pipelines
- **Squad:** CI/CD, Cloud, Monitoring, SRE, Infrastructure Security (5 squads activos)
- **Responsabilidad primaria:** Cada producto OVAV necesita deploy, monitoreo, y respuesta a incidentes.

### 8. UI/UX Design [v2.0 · REGISTRADA]
- **Lead:** Elena
- **Entry point para:** Design system, user research, accessibility (WCAG), prototyping, design review
- **Squad:** Design System, UX Research, Accessibility, Prototyping (4 squads activos)
- **Responsabilidad primaria:** Consistencia visual, accesibilidad, y experiencia de usuario unificada en todos los productos OVAV.

---

## Entry points — ¿Con quién habla el CEO?

| Si el CEO necesita... | Área | Lead |
|----------------------|------|------|
| Construir un producto digital | Digital Product Engineering | Dante |
| Gobernar/asegurar OVAV | Platform Engineering | Thavren |
| Validar si algo es cierto | Evidence & Decision Intelligence | Eidren |
| Vender/monetizar/posicionar | Commercial & Growth | Sofía |
| Educar/onboardear usuarios | Education & Career Development | Valeria |
| Protocolos de salud/bienestar | Health & Performance Science | Renata |
| Infraestructura/deploy/monitoreo | DevOps & Infrastructure | Uriel |
| Diseño/UX/accesibilidad | UI/UX Design | Elena |

---

## Reglas de coordinación

1. **Dante es el coordinador nato de proyectos multi-área.** Si un proyecto requiere 3+ áreas, Dante lidera la coordinación. Thavren habilita la plataforma pero no coordina.
2. **Handoff Protocol es vinculante.** Si el lead A solicita un handoff al lead B, B debe responder en ≤24h (crítico) o ≤72h (estándar).
3. **Revisión cruzada obligatoria.** Antes de cerrar una fase, cada lead debe recibir feedback de al menos 2 otros leads.
4. **Squads siempre activos.** Los squads no se apagan entre fases. Siguen refinando, iterando, testeando.
5. **Contrato de integración.** Cada proyecto tiene un `integration_contract.yaml` que define qué entrega cada lead, qué consume de otros, y deadlines.
6. **Thavren es plataforma, no cuello de botella.** Thavren construye y mantiene las herramientas. No aprueba cada handoff. No compila reportes de otros leads. Eso es responsabilidad del lead coordinador (Dante).
