# OVAV Landing Page — Content Brief
# =============================================================================
# Author: Sofía (Commercial & Growth Strategy)
# For: Dante (Digital Product Engineering) — copy-paste ready
# Date: 2026-06-15
# Deadline: Landing viva antes de Jul 7 2026 (Product Hunt)
# Source of truth: .ovav/plan/business_model.yaml
# =============================================================================

## 1. HERO SECTION

### Headline (pick one — A/B test both)

**Option A (benefit-driven):**
> Goberná la IA en tu flujo de desarrollo. No solo la uses.

**Option B (problem-driven):**
> Tu equipo usa 4 IAs distintas. ¿Quién gobierna qué modelo toca tu código?

### Subheadline

> OVAV es el gobernador open-core de tu estación de trabajo con IA.
> 8 perfiles profesionales, multi-modelo sin vendor lock-in, 100% local-first.
> Runtime Go nativo. 15 MB. Sin Electron. Sin cloud.

### CTA Button

**Pre-launch (ahora → Jul 6):**
```
[Join the Waitlist] → email capture form
```

**Product Hunt day (Jul 7+):**
```
[Get Early Access — Free] → direct signup / GitHub OAuth
```

### Secondary CTA
```
[Ver en GitHub] → github.com/ovav/ovav
```

---

## 2. VALUE PROPOSITION — ONE-LINER

> **Governed AI — not just suggested AI.**
>
> Mientras Copilot y Cursor te sugieren código, OVAV gobierna cómo usás la IA
> en todo tu ciclo de desarrollo: plan → build → test → deploy, con evidencia
> auditable de cada decisión.

---

## 3. COMPETITIVE MOAT — 5 BULLETS

| # | Diferenciador | Por qué importa |
|---|---|---|
| 1 | **Local-first. Tu código nunca sale de tu máquina.** | Copilot y Cursor mandan tu código a la nube. OVAV corre 100% local con Vault AES-256-GCM. |
| 2 | **Multi-modelo sin vendor lock-in.** | Usá OpenAI hoy, Claude mañana, Llama el viernes. Cambiá de modelo en caliente sin tocar tu workflow. |
| 3 | **8 perfiles profesionales por dominio.** | No sos "un developer". Sos Platform Engineer, Product Engineer, Health Scientist. Cada perfil tiene sus propias herramientas, benchmarks y flujos. |
| 4 | **Runtime Go nativo (~15 MB binario).** | Sin Electron. Sin Node runtime. Sin 2 GB de RAM. Un solo binario compilado con Go stdlib. |
| 5 | **Open-core (Apache 2.0).** | Auditable. Contribuible. Sin caja negra. El código que gobierna tu código es tan abierto como el tuyo. |

### Extra for detail section

- **SDLC completo** — plan → build → test → deploy, no solo autocompletado en el editor.
- **Evidence & benchmarks integrados** — sabés qué modelo sugirió qué, por qué, y con qué respaldo.
- **Self-hosting disponible** (Enterprise) — Docker, Kubernetes, on-prem.
- **Editor-agnóstico** — funciona con VS Code, Neovim, terminal, lo que uses.

---

## 4. PRICING TIERS — FINAL

### OVAV Free — $0/mo

*Para developers individuales que quieren probar gobernanza con IA.*

**Incluye:**
- CLI completa (`ovav plan`, `build`, `test`, `deploy`)
- 2 perfiles profesionales (Platform Engineering + 1 a elección)
- Modelos comunitarios (DeepSeek, Llama, Mistral, etc.)
- Documentación pública
- Soporte comunitario (GitHub Discussions)
- Vault encryption local (AES-256-GCM)

**Limitaciones:**
- Máximo 2 perfiles activos
- Sin SSO ni audit logs
- Sin soporte prioritario
- Sin modelos privados (OpenAI, Anthropic, Azure)
- Sin despliegue gestionado

**CTA:** `[Start Free]`

---

### OVAV Pro — $19/mo · $190/año (2 meses gratis)

*Para developers profesionales que exigen control total de su stack con IA.*

**Todo lo de Free, más:**
- ✅ 8 perfiles profesionales completos
- ✅ Modelos ilimitados — OpenAI, Anthropic, Google, Azure, open-source
- ✅ Evidence & Decision Intelligence (benchmarks, research briefs)
- ✅ Tailor Composer — planes de desarrollo personalizados por perfil
- ✅ Soporte prioritario (email, respuesta < 24h hábiles)
- ✅ Vault encryption con backup manual
- ✅ Acceso anticipado a features nuevas (beta channel)

**Limitaciones:**
- Sin SSO/SAML
- Sin audit logs
- Sin SLA
- 1 usuario por licencia

**Ideal para:** Developers independientes, tech leads, founders técnicos.

**CTA:** `[Start Pro — $19/mo]`

---

### OVAV Enterprise — $49/usuario/mes

*Para equipos de ingeniería que necesitan gobernanza, compliance y escala.*

**Todo lo de Pro, más:**
- ✅ SSO (SAML/OIDC) — Google Workspace, GitHub, Azure AD
- ✅ Audit logs completos (quién, qué, cuándo, con qué modelo)
- ✅ Self-hosting (Docker, Kubernetes, on-prem)
- ✅ Custom profiles específicos de tu empresa
- ✅ SLA 99.5% uptime (cloud) o acuerdo personalizado (self-hosted)
- ✅ Soporte dedicado — Slack compartido + account manager
- ✅ Onboarding guiado (2 sesiones de 90 min con Platform Engineer OVAV)
- ✅ Vault encryption con backup automatizado a S3/GCS
- ✅ Custom model routing rules (ej. compliance models para datos sensibles)

**Mínimo 10 seats.** Para 500+ seats → `enterprise@ovav.dev`

**Ideal para:** Equipos de 10-500+ devs. Fintech, healthtech, empresas con compliance.

**CTA:** `[Contact Sales]` o `[Start Enterprise Trial]`

---

## 5. PRICING TABLE — WEB-FRIENDLY (copy-paste a HTML/React)

```
┌─────────────────┬──────────────────┬──────────────────┬──────────────────────┐
│                 │     FREE         │      PRO         │    ENTERPRISE        │
│                 │     $0/mo        │    $19/mo        │  $49/user/mo         │
├─────────────────┼──────────────────┼──────────────────┼──────────────────────┤
│ Perfiles        │ 2                │ 8                │ 8 + custom           │
│ Modelos         │ Comunitarios     │ Ilimitados       │ Ilimitados           │
│ CLI completa    │ ✓                │ ✓                │ ✓                    │
│ Vault encrypt   │ Local            │ Local + backup   │ Auto-backup S3/GCS   │
│ Evidence        │ —                │ ✓                │ ✓                    │
│ Tailor Composer │ —                │ ✓                │ ✓                    │
│ SSO             │ —                │ —                │ ✓                    │
│ Audit logs      │ —                │ —                │ ✓                    │
│ Self-hosting    │ —                │ —                │ ✓                    │
│ SLA             │ —                │ —                │ 99.5%                │
│ Soporte         │ Comunidad        │ Email <24h       │ Slack + AM           │
│ Onboarding      │ —                │ —                │ 2 sesiones guiadas   │
├─────────────────┼──────────────────┼──────────────────┼──────────────────────┤
│ CTA             │ Start Free       │ Start Pro        │ Contact Sales        │
└─────────────────┴──────────────────┴──────────────────┴──────────────────────┘
```

---

## 6. SOCIAL PROOF — PRE-LAUNCH

Estamos en **pre-launch**. No tenemos números de usuarios. Lo que SÍ podemos mostrar:

### Sección: "Built Different"

```
344+ tests   ·   15K+ LOC Go   ·   0 dependencias   ·   8 perfiles   ·   1 binario
```

### Sección: "Open-core, desde el día 1"

```
Apache 2.0. El código que gobierna tu IA es tan abierto como tu proyecto.
```

### Sección opcional: "El equipo"

```
Construido por un equipo distribuido de ingenieros en 6 países,
liderado por Alexander Salvador desde Buenos Aires.
```

### NO pongas (todavía):

- ❌ "Trusted by X,000 developers" (no tenemos números)
- ❌ Testimonials falsos (integridad > vanity metrics)
- ❌ "Used by companies like..." (no tenemos enterprise clients aún)

---

## 7. FAQ — PARA LA LANDING (sección recomendada)

**¿OVAV reemplaza a Copilot o Cursor?**
No. OVAV es la capa de gobernanza por encima de tus herramientas. Podés seguir usando tu editor favorito. OVAV gobierna qué modelo usa cada tarea y mantiene evidencia auditable de cada decisión.

**¿Mi código sale de mi máquina?**
No. OVAV es local-first. Todo corre en tu máquina. Si elegís usar modelos cloud (OpenAI, Anthropic), solo se envían los prompts que vos configurás — nunca tu código fuente completo.

**¿Qué es un "perfil profesional"?**
Cada perfil es un set de herramientas, benchmarks y flujos diseñados para un dominio específico. Platform Engineering tiene deploy y seguridad. Product Engineering tiene diseño y UX. Education tiene curriculum y evaluación. No sos "un developer" — tenés 8 especialidades.

**¿Puedo usar mis propios modelos?**
Sí. Pro y Enterprise soportan cualquier modelo compatible con API OpenAI-style. Traé tu propio endpoint.

**¿Qué pasa con mis datos si cancelo?**
Tu Vault es local. Tus datos son tuyos. Podés exportar todo en cualquier momento. Enterprise tiene backup automatizado a tu propio S3/GCS.

**¿Hay descuento para estudiantes?**
OVAV Free es permanente. Si sos estudiante, usalo gratis. Cuando te gradúes y empieces a trabajar, upgradear a Pro va a ser natural.

**¿Cuándo es el launch público?**
Product Hunt: **Julio 7, 2026**. Unite al waitlist para early access.

---

## 8. LINKS Y ASSETS QUE DANTE NECESITA

| Asset | Estado | Ubicación |
|---|---|---|
| Logo OVAV | ❓ Preguntar a CEO/Inés | — |
| Demo video (2 min) | ❌ No existe aún | Necesita producción antes de Jul 7 |
| Screenshots Cockpit TUI | ❌ No existen aún | Capturar del Go Cockpit (8 vistas) |
| GitHub repo | ✅ `github.com/ovav/ovav` | Público |
| Docs site | 🔶 En progreso (Nadia, 25%) | `docs.ovav.dev` |
| Twitter/X handle | ❓ Preguntar a CEO | — |
| Discord server | ❌ No existe aún | Crear antes de PH |

---

## 9. DECISIONES PENDIENTES (CEO)

Estos puntos requieren aprobación de Alexander Salvador antes del launch:

- [ ] **Pricing final aprobado** — $19/mo Pro, $49/user/mo Enterprise
- [ ] **Product Hunt date** — Jul 7 2026 confirmado
- [ ] **Free tier permanente** — ¿confirmamos que no hay time limit?
- [ ] **Enterprise contact email** — `enterprise@ovav.dev` ¿ya existe?
- [ ] **Logo + branding** — ¿Inés tiene assets?

---

## 10. NOTA PARA DANTE

Dante, este brief es tu source of truth para la landing page. Todo lo que está acá salió del business model que ya revisé con Gabriela y Julián.

**Lo que necesito de vos:**
- Hero con headline + subheadline + CTA (waitlist form pre-PH)
- Pricing section con la tabla de 3 tiers (Free/Pro/Enterprise) — diseño limpio, mobile-first
- Competitive moat section con los 5 bullets + íconos
- FAQ section con las 6 preguntas
- Footer con links (GitHub, docs, Twitter, Discord)
- Formulario de waitlist (email → guardar en algún lado, aunque sea Google Sheets al principio)

**Lo que NO necesito de vos (no es tu área):**
- Demo video (lo coordino con Inés y el CEO)
- Screenshots del Cockpit (Thavren los genera)
- Blog posts (Gabriela los escribe)

**Timeline:**
- Landing funcional: **Jun 22** (2 semanas antes de PH)
- Landing con todo el contenido: **Jun 29** (1 semana antes de PH)
- Product Hunt: **Jul 7**

Si algo no cierra o necesitás ajustar el copy por espacio/formato, avisame y lo refinamos juntos. No hagas cambios a pricing o VP sin consultarme — son decisiones comerciales, no de diseño.

— Sofía
