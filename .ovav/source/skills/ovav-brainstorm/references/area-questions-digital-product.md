# Digital Product — Mandatory Questions

> Loaded by `ovav-brainstorm` skill when frontend/web app request detected.
> Ask 3-5 questions ONE AT A TIME. Wait for CEO response before next.

## Digital Product Questions

### DPQ1: State Management
"¿Cómo manejan estado global del frontend?"
- Zustand (minimal, TypeScript-first, boilerplate bajo)
- Jotai (atomic, granular reactivity)
- Redux Toolkit (enterprise, verbose pero predecible)
- TanStack Query (server state, cache + sync)
- Context API (estado simple, sin biblioteca extra)
- No estado global (props drilling OK)

Why: Determina cómo componentes comparten datos, estructura de archivos, y curva de aprendizaje del equipo.

### DPQ2: Form Handling
"¿Qué biblioteca de formularios?"
- React Hook Form + Zod (performance, validation schema)
- Formik + Yup (tradicional, más boilerplate)
- Native HTML forms (mínimo, sin dependencias)
- Conform (nuevo, blazing fast, aún no mainstream)

Why: Forms son 40%+ del código frontend. La elección impacta UX (validación en tiempo real) y DX.

### DPQ3: Routing
"¿Navegación SPA o MPA?"
- React Router v6 (estándar SPA)
- TanStack Router (type-safe, nuevo, competitivo con Remix)
- Next.js App Router (SSR/SSG si se requiere SEO)
- MPA tradicional (sin SPA, página completa)

Why: SPA = velocidad post-carga inicial. SSR = SEO + first contentful paint rápido.

### DPQ4: API Client Layer
"¿Cómo se comunican con el backend?"
- Axios (interceptors, cancelación, testing fácil)
- Fetch API nativo (menos dependencias, pero menos features)
- TanStack Query (cache + sync + fetch, reemplaza axios)
- tRPC (end-to-end types, si backend es TypeScript)
- SWR (stale-while-revalidate, similar a TanStack Query)

Why: Determina cómo se manejan errores, loading states, y retry logic.

### DPQ5: Testing Strategy
"¿Qué nivel de tests?"
- Unit tests (Vitest/Jest) + Component tests (Testing Library)
- E2E con Playwright/Playwright CLI (GUI visual + headless)
- Snapshot tests (UI consistency)
- No tests (velocidad inicial, deuda técnica)

Why: Testing es insurance. Playwright E2E detecta regressions visuales que unit tests no ven.

## Deliverable
After all questions answered → Dante adds `## 3. Layout`, `## 4. Features`, and `## 6. Technical Approach` sections to DESIGN.md.
