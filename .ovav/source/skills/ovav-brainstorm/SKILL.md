---
name: ovav-brainstorm
description: Use when OVAV detects a NEW IDEA (no existing plan) and needs to gather requirements before any code is written. This is the official HARD GATE protocol that forces area-specific questions one-by-one before generating DESIGN.md. Activates on new project mentions, scope deviations, or first-time external system integration.
license: Apache-2.0
metadata:
  author: thavren (OVAV)
  version: "1.1"
owner_profile: ovav_systems_architect
owner_lane: runtime_governance
status: active
memory_write: scoped
memory_write_scope: runtime_governance_only
risk_level: low
last_updated: 2026-07-30
---

# OVAV Brainstorm Protocol — Mandatory Questions Per Area

This skill is the OFFICIAL HARD GATE that runs BEFORE any `DESIGN.md` is generated.
It enforces that the correct LEAD asks the questions that ONLY THEIR AREA can answer.

## Why This Skill Exists

**ROOT PROBLEM**: In previous sessions, PL-0 received "execute this project at 100% autonomy"
and skipped directly to code, generating DESIGN.md without asking the questions that would
have prevented the result from feeling like a "basic mockup".

**FIX**: This protocol is **NON-NEGOTIABLE**. Even at 100% autonomy, the agent MUST ask
at least 3-5 area-specific questions and wait for CEO answers before DESIGN.md.

**Pattern Source**: Kimi Code `plan` subagent (read-only + one-by-one questions).
**Improvement over Kimi**: OVAV questions are AREA-SPECIFIC — only Elena asks UX,
only Thavren asks architecture, only Dante asks frontend implementation.

## Detection — When This Skill Activates

| Trigger | Example |
|---|---|
| New project mention | "quiero hacer una app de...", "construyamos un..." |
| New feature request | "agregar X", "implementar Y" |
| Scope deviation | "en vez de X hagamos Y" |
| First external system mention | "integrar con Stripe", "conectar a AWS" |
| Goal redefinition | "el objetivo es...", "lo que necesito es..." |

If detected → this skill fires BEFORE any code generation.

## Protocol — 5 Steps

### Step 1: Identify the AREA
From the user's request, classify into ONE primary area:

| Request Signal | Primary Area | Lead |
|---|---|---|
| web app, React, UI, components, frontend | Digital Product | Dante |
| design, UX, accessibility, wireframes, design system | UX Design | Elena |
| Go backend, CLI, runtime, security, install | Platform Engineering | Thavren |
| research, benchmark, comparison, evidence | Research Intelligence | Eidren |
| business, monetization, growth, pricing | Commercial & Growth | Sofía |
| deploy, infra, cloud, Docker, K8s | DevOps | Uriel |
| nutrition, fitness, health metrics | Health & Performance | Renata |
| contract, legal, GDPR, compliance | Legal | Camila |
| security testing, penetration | Adversarial | Kenji |
| learning, curriculum, course | Education | Valeria |

---

## Pre-Project Mandatory Questions (Q1-Q10) — ASK BEFORE ANY AREA QUESTIONS

> **CRITICAL**: These 10 questions are PROJECT-STRUCTURE questions that apply to ALL projects.
> Ask them FIRST before any area-specific questions. Answers determine entire architecture.

### PPQ1: Repository Structure
"¿Monorepo o polyrepo? ¿Cuántos paquetes/espacios de trabajo planeas?"
- Monorepo (pnpm workspaces, Turborepo, Nx)
- Polyrepo (repo separada por servicio)
- Mezcla (algunos servicios monorepo, otros separados)

Why: Cambia toda la estructura de carpetas, CI/CD, y cómo se comparten tipos entre backend/frontend.

### PPQ2: Form Factor & Platform
"¿Web-only, PWA, mobile native (iOS/Android), desktop, o híbrido? ¿Necesidad de App Store?"
- Web / PWA / Native (React Native, Flutter, Swift/Kotlin) / Desktop (Tauri, Electron) / Híbrido
- Si mobile: ¿PWA-first + Capacitor wrap para stores?

Why: Cambia todo el frontend (React SPA vs Native vs Tauri). Si es PWA-first, usar vite-plugin-pwa.

### PPQ3: API Pattern & Real-Time
"¿REST simple, GraphQL, tRPC, o gRPC? ¿Necesita real-time (WebSocket, SSE)?"
- REST / GraphQL / tRPC / gRPC
- Real-time: WebSocket / Server-Sent Events / Polling

Why: Dating apps necesitan real-time para match notifications y chat. REST es insuficiente.

### PPQ4: Database & Persistence
"¿SQLite, PostgreSQL, MySQL, MongoDB? ¿Managed (RDS/Supabase) o self-hosted?"
- SQLite (MVP) / PostgreSQL 16+ / MySQL / MongoDB / Otro
- Managed vs Self-hosted

Why: Schema design y connection pooling dependen de esta decisión.

### PPQ5: Authentication Strategy
"¿JWT (RS256/HS256), session cookies, OAuth2 (Google/Apple), magic links, SMS?"
- JWT / Sessions / OAuth / Magic Links / SMS
- KYC requerido (DNI, passport, biometric)?

Why: Dating app requiere identidad verificable. bcrypt + JWT RS256 es mínimo.

### PPQ6: Deployment Target
"¿Vercel/Railway/Fly.io, AWS/GCP, self-hosted, o multi-cloud?"
- Vercel / Railway / Fly.io / AWS / GCP / Self-hosted / Multi-cloud

Why: Secret management, CI/CD, y observabilidad dependen del hosting.

### PPQ7: Team Size & CI Strategy
"¿Equipo de 1 persona o 10+? ¿Skills del equipo (Go, TypeScript, Python)?"
- 1 persona / 2-5 personas / 5-10 personas / 10+ personas
- Skills dominantes

Why: Tamaño del equipo determina complejidad del plan. 1 persona = plan simple. 10+ = monorepo con límites de módulos.

### PPQ8: Scalability Expectations
"¿Cuántos usuarios simultáneos esperados? ¿<10K, 10K-100K, 100K-1M, 1M+?"
- <10K usuarios (MVP) / 10K-100K / 100K-1M / 1M+

Why: <10K = SQLite OK. >100K = PostgreSQL + Redis cache. >1M = microservicios.

### PPQ9: Security & Compliance
"¿LGPD/GDPR, age verification (18+), moderation (content/AI)? ¿Auditoría?"
- LGPD / GDPR / Age verification / Content moderation / AI moderation / Auditoría

Why: Dating app requiere KYC (DNI) y age verification. Compliance afecta arquitectura.

### PPQ10: Budget & Timeline
"¿MVP rápido (2-4 semanas) o producto escalable desde día 1? ¿Budget para servicios terceros?"
- MVP rápido / Escalable desde día 1 / Con budget para servicios (Auth0, Supabase, etc.)

Why: MVP rápido = SQLite + placeholder JWT. Escalable = PostgreSQL + JWT RS256 real.

---

### Step 2: Load the AREA's Mandatory Questions
Each area has its own question set. Load from `references/area-questions-{area}.md`.

**Minimum 3 questions, maximum 5.** ONE AT A TIME (not all at once).

### Step 3: Ask Questions SEQUENTIALLY
Ask the first question. WAIT for CEO response. Then ask the next. NEVER batch.

**Example — Elena (UX Design) on "build me a dating app":**
```
Elena: "¿Cuál es el público objetivo principal? (edad, género, contexto LATAM/global)"
→ CEO: "Peruanos 25-35, urbano"
Elena: "¿WCAG 2.1 AA o AAA?"
→ CEO: "AA"
Elena: "¿Necesitas design system nuevo o usar uno existente (Tailwind UI / shadcn)?"
→ CEO: "shadcn"
```

### Step 4: Generate DESIGN.md Only After All Questions Answered
After minimum 3 questions answered, generate `plans/<project>/DESIGN.md` using the **DESIGN.md Template** below.

### Step 5: Wait for CEO Approval of DESIGN.md
Even at 100% autonomy — wait. Generate PLAN.md only after CEO approves DESIGN.
Only then invoke `ovav-build` to execute.

---

## DESIGN.md Template

Every OVAV DESIGN.md follows this exact structure:

```markdown
# <Project Name> — Design Document

> **Versión:** 1.0 | **Fecha:** YYYY-MM-DD | **Lead:** <Lead Name> (+ contributors)
> **Status:** Borrador | Pendiente CEO approval

---

## 1. Concept & Vision

**¿Qué es [nombre]?**
[2-3 sentences describing what it is]

**¿Qué problema resuelve?**
[1-2 sentences on the specific problem it solves]

**Tono y feeling:**
[Adjectives describing user experience: warm/safe, fast/dynamic, etc.]

---

## 2. Design Language

### Color Palette
| Rol | Color | Hex | Uso |
|---|---|---|---|
| Primary | [Name] | `#XXXXXX` | CTAs principales |
| Secondary | [Name] | `#XXXXXX` | Secondary actions |
| Background | [Name] | `#XXXXXX` | Base background |
| Text Primary | [Name] | `#XXXXXX` | Main text |
| Text Secondary | [Name] | `#XXXXXX` | Subtitles |
| Success | [Name] | `#XXXXXX` | Confirmations |
| Error | [Name] | `#XXXXXX` | Errors |

### Typography
| Elemento | Font | Size | Weight |
|---|---|---|---|
| H1 | Inter | 32px | 700 |
| H2 | Inter | 24px | 600 |
| Body | Inter | 16px | 400 |
| Caption | Inter | 14px | 400 |

### Spacing System
- Base unit: **4px**
- Scale: 4, 8, 12, 16, 24, 32, 48, 64, 96

### Motion
- Micro-interactions: **150ms ease-out**
- Page transitions: **300ms ease-in-out**
- Loading: skeleton pulse **1.5s infinite**

---

## 3. Layout & Structure

### Page Structure
```
/
├── /page1              — [description]
├── /page2              — [description]
└── /app                — [description]
    ├── /app/feature1  — [description]
    └── /app/feature2  — [description]
```

### Responsive Breakpoints
| Breakpoint | Width | Layout |
|---|---|---|
| Mobile | `< 640px` | Single column |
| Tablet | `640px - 1024px` | Two column |
| Desktop | `> 1024px` | Max-width centered |

---

## 4. Features & Interactions

### F1: [Feature Name]
**Concepto:** [1 sentence]

**UI:**
- [Bullet points describing UI elements]

**Estados:**
- Default: [description]
- Loading: [description]
- Error: [description]
- Empty: [description]

---

## 5. Component Inventory

### Atoms
| Component | States | Notes |
|---|---|---|
| Button | default, hover, active, disabled, loading | Variants: primary, secondary, ghost |
| Input | default, focus, error, disabled | Con label y error message |
| Avatar | with-image, fallback, loading | Sizes: sm, md, lg |

### Molecules
| Component | States | Notes |
|---|---|---|
| SwipeCard | default, dragging, liked, nope | Con gesture handler |
| MatchCard | default, unread | Avatar + name + preview |

### Organisms
| Component | States | Notes |
|---|---|---|
| BottomNav | 4 tabs | Mobile only |
| DiscoveryStack | loading, cards, empty | Swipe cards stack |

---

## 6. Technical Approach

### Frontend Architecture
```
frontend/src/
├── api/               # Axios client + endpoint functions
├── components/
│   ├── atoms/        # Button, Input, Avatar, Badge
│   ├── molecules/    # SwipeCard, MatchCard
│   ├── organisms/    # BottomNav, DiscoveryStack
│   └── templates/   # AppLayout, AuthLayout
├── hooks/            # useAuth, useSwipe
├── stores/           # Zustand stores
├── pages/            # Page components
├── types/            # TypeScript interfaces
└── utils/           # Formatters, validators
```

### Backend Architecture
```
backend/
├── cmd/server/main.go    # Entry point
├── internal/
│   ├── api/
│   │   ├── handlers/    # auth, users, discovery, swipes, matches, chats
│   │   ├── middleware/  # auth, cors, ratelimit
│   │   ├── router.go
│   │   └── server.go
│   ├── db/              # db.go + migrations/
│   ├── models/          # user, swipe, match, chat, message
│   └── services/        # auth, discovery, match, chat
└── workers/             # Background workers
```

### Real Code Architecture (REQUIRED — not tables)

**Go Structs:**
```go
// backend/internal/models/user.go
type User struct {
    ID           int64     `json:"id"`
    Email        string    `json:"email"`           // plaintext — login identifier
    PasswordHash string    `json:"-"`               // bcrypt — NEVER exposed
    DNIHash      string    `json:"-"`               // FNV-64a — privacy
    Bio          string    `json:"bio"`
    PhotoURL     string    `json:"photo_url"`
    Gender       string    `json:"gender"`          // libre — cualquier identidad
    BirthDate    string    `json:"birth_date"`      // YYYY-MM-DD
    TrustScore   float64   `json:"trust_score"`     // 0.0-1.0
    Mode         string    `json:"mode"`            // "normal" | "anonimo"
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}
```

**TypeScript Interfaces (from shared package):**
```typescript
// packages/shared/src/types.ts
export interface User {
  id: number
  email: string
  real_name: string
  bio?: string
  photo_url?: string
  gender: string
  birth_date: string
  trust_score: number
  mode: 'normal' | 'anonimo'
  created_at: string
}
```

**Zod Schemas (shared validation):**
```typescript
// packages/shared/src/schemas.ts
import { z } from 'zod'

export const RegisterSchema = z.object({
  email: z.string().email(),
  password: z.string().min(8).regex(/[A-Z]/, 'mayúscula requerida'),
  real_name: z.string().min(2).max(50),
  dni: z.string().regex(/^\d{8}$/, 'DNI peruano 8 dígitos'),
  gender: z.string().min(1), // libre — cualquier identidad
  birth_date: z.string().regex(/^\d{4}-\d{2}-\d{2}$/)
})
```

**Function Signatures:**
```typescript
// frontend/src/stores/authStore.ts
interface AuthState {
  user: User | null
  token: string | null
  isAuthenticated: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => void
  register: (data: RegisterInput) => Promise<void>
}
```

### API Contract (GraphQL or REST)
```
POST /api/v1/[endpoint]
Body: { ... }
Response: { ... }
```

---

## 7. Data Model

### Schema
```sql
CREATE TABLE [name] (
    [columns]
);
```

---

## 8. Security Model
| Concern | Mitigación |
|---|---|
| [ Concern ] | [ Mitigation ] |

---

## 9. Folder Structure — Vista Completa

```
<project>/
├── frontend/
│   └── src/
├── backend/
│   └── internal/
├── plans/
│   └── <project>/
│       ├── DESIGN.md
│       └── PLAN.md
└── SPEC.md
```

---

## 10. Open Questions

| # | Pregunta | Opciones |
|---|---|---|
| 1 | [Pregunta] | A: [Option A] / B: [Option B] |

---

**¿Aprobás este DESIGN.md para proceder al PLAN.md?**
```

---

## PLAN.md Template

Every OVAV PLAN.md follows this exact structure:

```markdown
# <Project Name> — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `ovav-build` to execute this plan task-by-task.
> Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** [One sentence describing what this builds]

**Architecture:** [2-3 sentences about approach]

**Tech Stack:** [Key technologies/libraries]

---

## Global Constraints

[Project-wide requirements that bind EVERY task — version floors, naming rules, platform requirements. One line each.]

---

## File Structure

[EXACT file structure mapping before defining tasks — each file on its own line with 1-line responsibility.]

```
frontend/src/
├── api/
│   └── client.ts          # [1-line responsibility]
├── components/
│   └── ...
backend/
├── cmd/
└── ...
```

---

## Phases

### Phase 1: [Name] (Días X-Y)
- [ ] Phase intro

### Phase 2: [Name] (Días X-Y)
- [ ] Phase intro

---

## Tasks

### Phase 1.1: [Task Group]

**Tarea 1.1.1:** [Task name]

**Covers:** [spec section anchors — e.g., §3, §7]

**Files:**
- Create: `exact/path/to/file.ts` or `exact/path/to/file.go`
- Modify: `existing/file.go:123-145`
- Test: `tests/exact/path/test.ts`

**Interfaces:**
- Consumes: [what this task uses from earlier tasks — exact signatures, types]
- Produces: [what later tasks rely on — exact function names, parameter and return types]

- [ ] **Step 1: Write the failing test**

```typescript
// tests/exact/path/to/test.ts
describe('[Component]', () => {
  it('should [expected behavior]', async () => {
    const result = await someFunction(input)
    expect(result).toEqual(expected)
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm test tests/exact/path/to/test.ts --run` or `go test ./...`
Expected: FAIL with "[error description]"

- [ ] **Step 3: Write minimal implementation**

```typescript
// exact/path/to/file.ts
export async function someFunction(input: InputType): Promise<OutputType> {
  return expected // minimal to pass test
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pnpm test tests/exact/path/to/test.ts --run` or `go test ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add tests/exact/path/to/test.ts exact/path/to/file.ts
git commit -m "feat: add [specific feature]"
```

---

**Tarea 1.1.2:** [Task name]
[Same structure — every task MUST have code in every step, no TODOs, no placeholders]

---

## Squad Assignments

| Lead | Phase | Responsabilidad |
|---|---|---|
| [Lead Name] | Phase 1 | [Responsibility] |
| [Lead Name] | Phase 2 | [Responsibility] |

---

## Definition of Done

- [ ] Todos los tests pasan
- [ ] Build succeeds
- [ ] Coverage ≥80%
- [ ] No console errors
- [ ] Responsive tested
```

## Area Question Sets

Each area's mandatory questions live in `references/area-questions-{area}.md`.
Loaded by the lead agent when this skill activates.

| Area | Questions File |
|---|---|
| UX Design | `references/area-questions-ux-design.md` |
| Digital Product | `references/area-questions-digital-product.md` |
| Platform Engineering | `references/area-questions-platform-engineering.md` |
| Research Intelligence | `references/area-questions-research.md` |
| Commercial & Growth | `references/area-questions-commercial.md` |
| DevOps | `references/area-questions-devops.md` |
| Health | `references/area-questions-health.md` |
| Legal | `references/area-questions-legal.md` |
| Adversarial | `references/area-questions-adversarial.md` |
| Education | `references/area-questions-education.md` |

## Cross-Area Projects

If a project spans multiple areas (e.g., dating app = UX + Product + Backend):

1. Identify PRIMARY area (the one whose questions are most critical)
2. Run that area's questions FIRST
3. After DESIGN.md generated, SECONDARY area questions are derived
4. Each area lead adds their section to DESIGN.md as a contributor

**Example — social-citas (multi-area):**
- PRIMARY: UX Design (Elena) → 3 questions about design system, accessibility, mobile-first
- SECONDARY: Digital Product (Dante) → 2 questions about component library, state management
- TERTIARY: Platform Engineering (Thavren) → 3 questions about DB, API, deployment

## Hard Rules

1. **NEVER generate DESIGN.md before answering PPQ1-PPQ10 (pre-project questions).**
2. **NEVER generate DESIGN.md before answering 3+ questions per area.**
3. **NEVER batch questions** — ask one at a time.
4. **NEVER skip questions** even under CEO "100% autonomy" directive.
5. **NEVER pretend an answer** — if CEO says "you decide", document it as `[AUTO-DECIDED]` and explain why.
6. **ALWAYS cite the area's CRITERIA.yaml** when answering on behalf of CEO.
7. **DESIGN.md must contain REAL CODE ARCHITECTURE** — Go structs, TypeScript interfaces, function signatures — not just file trees.
8. **PLAN.md must contain TDD steps** — RED-GREEN-REFACTOR per task, with actual test code and implementation code.

## Anti-Patterns (What This Skill Prevents)

| Anti-pattern | Symptom | Fix |
|---|---|---|
| "Skip questions, just code" | Basic mockup result | This protocol forces questions |
| "All questions at once" | CEO overwhelmed, low quality answers | ONE BY ONE |
| "Same questions for every area" | Wrong questions asked | Area-specific sets |
| "Skip multi-area coordination" | UX done without backend coordination | Cross-area handoff |
| "GENERAL TASK fallback" | Wrong lead name shown | Use `workflow + agent(team-*)` not `actor.run` |
