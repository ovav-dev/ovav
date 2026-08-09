# TASK8 — Delegation: UX/UI 2026 Modernization, Animations, URL Optimization

| Field | Value |
|---|---|
| Handoff ID | TASK8-UX-MODERNIZATION |
| Delegate To | **Elena** (UX/UI Design) + **Dante** (Digital Product Engineering) |
| Delegated By | Thavren / Platform Engineering |
| Date | 2026-06-16 |
| Priority | P1 — Pre-Launch Polish |
| Branch | task/tasknext-ceo-task8 |

---

## 1. Task Specification

Modernize all 4 OVAV web surfaces to 2026 design standards. This includes visual refresh, animations/micro-interactions, navigation improvements, and URL structure optimization. All surfaces must feel coherent under the "Professional Development Governance" brand.

### 1.1 Surfaces to Modernize

| Surface | URL | Current Stack | Category |
|---|---|---|---|
| **Landing** | `ovav.dev` | Next.js 14 + Tailwind, static export, CF Pages | Marketing |
| **Dashboard** | `cpanel.ovav.dev` | React 18 + Vite, Go backend, Fly.io | Product |
| **Documentation** | `docs.ovav.dev` | Starlight (Astro), CF Pages | Reference |
| **Status** | `status.ovav.dev` | Better Uptime managed page | Operations |

### 1.2 Scope by Surface

**ovav.dev (Landing):**
- Visual refresh to 2026 aesthetic — dark/light mode toggle, refined typography, gradient system
- Animations: scroll-triggered reveals (Intersection Observer), profile card hover states, tier comparison transitions, hero section parallax or subtle motion
- Navigation: smooth scroll to sections, sticky header with blur-backdrop, mobile hamburger with expand animation
- URLs: already clean (`ovav.dev`, `ovav.dev/#pricing`). Ensure all internal links consistent
- Performance: current build ~112KB static. Audit for Core Web Vitals (LCP <2.5s, INP <200ms, CLS <0.1). Lighthouse >90

**cpanel.ovav.dev (Dashboard):**
- Visual refresh: professional dashboard aesthetic, consistent color system with landing, data visualization polish
- Animations: section transitions (fade/slide between tabs), loading skeletons, status pulse indicators, toast refinement
- Navigation: tab/sidebar polish, active state indicators, breadcrumbs, responsive sidebar collapse
- URLs: current SPA uses React state for section routing. Consider clean URL paths for deep-linking (`cpanel.ovav.dev/dashboard`, `/profiles`)
- Auth UX: login page polish, loading states during OAuth redirect, error state design

**docs.ovav.dev (Documentation):**
- Visual refresh: Starlight theme customization to match OVAV brand colors and typography
- Animations: minimal — smooth page transitions, sidebar expand/collapse, code block copy feedback. Respect `prefers-reduced-motion`
- Navigation: sidebar polish (current page indicator, collapsible sections), search integration (Pagefind or Starlight built-in), mobile nav refinement
- URLs: already clean Starlight URLs. Root redirect must be fixed (currently 404)
- Content: 15 pages exist (Phase 5 complete). Nadia owns content completeness

**status.ovav.dev (Status):**
- Visual refresh: custom CSS on Better Uptime page to match OVAV brand colors and logo
- Scope: minimal — managed by Better Uptime. Brand alignment only. No structural changes

### 1.3 Cross-Cutting Requirements

| Requirement | Details |
|---|---|
| **Design System** | Single design token system (colors, spacing, typography, shadows, radii) shared across all surfaces. CSS custom properties or Tailwind preset |
| **Dark/Light Mode** | All surfaces support toggle. Respect `prefers-color-scheme` by default, manual override in localStorage |
| **Motion Respect** | All animations respect `prefers-reduced-motion`. No animation-only information conveyance |
| **Mobile First** | All surfaces fully responsive. Landing and docs tested at 320px. cPanel functional at 768px+ |
| **Performance Budget** | Landing: <150KB transferred. Docs: <200KB. cPanel: code-split by section, initial load <200KB JS |
| **Brand Coherence** | Single visual identity across all 4 surfaces. User should feel they're in the same ecosystem |

---

## 2. Current State — What Platform Engineering Has Done

### 2.1 Landing Page — DEPLOYED

- **Stack**: Next.js 14 + Tailwind CSS, static HTML export
- **Content**: 6 sections (hero, pricing 3 tiers, 8 profiles, competitive moat, CTA, footer). Copy from Sofia's `landing_copy_brief.yaml`
- **Deploy**: Cloudflare Pages (`ovav-landing`), auto-deploy on main. DNS: `ovav.dev` + `www.ovav.dev` both 200 OK, SSL valid
- **Build**: ~112KB static output. Fast load time
- **What exists**: Full semantic HTML structure. Tailwind utility classes throughout. Component tree, section layout, content complete. This is a **restyle + animate** task
- **Source**: Commit `1991635 feat(landing): ovav.dev landing page`

### 2.2 cPanel Dashboard — DEPLOYED

- **Frontend**: React 18 + Vite + TypeScript. `tools/cpanel/src/`. 10 sections (Dashboard, Profiles, Validators, Security, Economy, Memory, Git, Agents, System, Operations). Login with OAuth (Google/GitHub). Toast notifications. ErrorBoundary
- **Backend**: Go runtime `go-runtime/cmd/cpanel/` with 17 route handlers. RS256 JWT auth. OAuth. SSE events. CORS, CSRF, rate limiting, path traversal (all hardened)
- **Tests**: 107 tests, 65.5% coverage, 0 data races
- **Deploy**: Fly.io (DFW, 2 machines). Dockerfile.cpanel
- **What exists**: Full dashboard functionality. All API integrations work. Skeleton of every section built. **Visual polish + navigation** task
- **Key files**: `tools/cpanel/src/App.tsx` (routing), `tools/cpanel/src/components/Login.tsx` (auth UI), `tools/cpanel/src/sections/*.tsx` (10 sections)

### 2.3 Documentation Site — DEPLOYED

- **Stack**: Starlight (Astro) + Cloudflare Pages
- **Content**: 15 pages (intro, quickstart, install, architecture, profiles, governance, multi-model, first-profile, tailor, vault, cpanel, cli, api, security, configuration). Logo SVG created
- **Deploy**: CF Pages to `docs.ovav.dev`. Custom domain connected. Auto-deploy on main
- **What exists**: Full documentation structure. Starlight theme with custom CSS (`src/assets/ovav.css`). Sidebar configured. All pages have content
- **Known issues**: Root redirect 404. No search integration. Starlight theme not yet brand-aligned
- **Key files**: `docs-site/astro.config.mjs`, `docs-site/src/assets/ovav.css`, `docs-site/src/content/docs/`

### 2.4 Status Page — DEPLOYED

- **Provider**: Better Uptime free tier
- **Monitors**: 4/4 LIVE (ovav.dev, cpanel.ovav.dev API, docs.ovav.dev, cPanel health check)
- **Status page**: `status.ovav.dev` active. Email alerts confirmed
- **What exists**: Functional status page with Better Uptime default theme
- **Scope**: Minimal — brand CSS only

### 2.5 Design Assets Already Created

- **Brand**: "Professional Development Governance" positioning (CEO-approved reframing)
- **Logo**: SVG logo exists (created in Phase 5 docs-site work)
- **Copy**: All landing copy is canonical in `landing_copy_brief.yaml`
- **Colors**: No formal design system yet — this is Elena's first task
- **Typography**: Not specified — Elena's call

---

## 3. What is NOT Done (Your Scope)

| Gap | Details |
|---|---|
| **Design system** | No shared design tokens. Each surface has independent styles |
| **Dark/light mode** | None of the surfaces have theme toggle |
| **Animations** | Zero animations across all surfaces. Static renders only |
| **cPanel navigation UX** | Tab switching is functional but basic. No transitions, no deep-linking |
| **Dashboard visual polish** | Sections render data but lack visual hierarchy, card design, data viz |
| **Landing visual refresh** | Clean but generic Tailwind. Needs distinctive 2026 identity |
| **Docs brand alignment** | Starlight default theme. CSS file exists but is minimal |
| **Status brand alignment** | Better Uptime default. No OVAV branding |
| **Performance audit** | No Lighthouse/CWV measurement for any surface |
| **Responsive polish** | Functional at breakpoints but not polished |

---

## 4. Architecture Guidance (Non-Binding)

**Design System (suggested approach):**
```
shared/
  design-tokens.css       (CSS custom properties: --ovav-color-primary, etc.)
  tailwind-preset.js      (Tailwind preset extending with OVAV tokens)
  animations.css          (Shared keyframe animations)

Landing: Tailwind preset + shared tokens
cPanel:   CSS custom properties injected in Vite (or Tailwind if adopted)
Docs:     Starlight custom CSS overriding with shared tokens
Status:   Custom CSS injection via Better Uptime settings
```

**Animation philosophy (Elena's domain, non-binding):**
- Prefer CSS-only animations (GPU-accelerated, no JS overhead)
- Use Intersection Observer for scroll reveals (landing)
- Use CSS transitions for UI state changes (cPanel tabs, docs sidebar)
- Use Framer Motion or similar only if complex orchestration needed
- Gate behind `prefers-reduced-motion`
- No animation should exceed 300ms (accessibility + perceived performance)

**Key decisions Elena + Dante own:**
- Design system tooling (CSS vars vs Tailwind preset vs both)
- Color palette and typography
- Animation library (or CSS-only)
- Whether to unify cPanel onto Next.js or keep React+Vite
- Dark/light mode implementation strategy
- Motion intensity levels (subtle vs expressive)

---

## 5. Acceptance Criteria

### Landing (ovav.dev)
1. Dark/light mode toggle in header, respects system preference, persists in localStorage
2. Scroll-triggered section reveals with smooth animations
3. Profile cards have hover micro-interactions
4. Pricing tier cards have selection/hover states with transitions
5. Sticky header with backdrop-blur on scroll
6. Mobile hamburger menu with expand/collapse animation
7. Lighthouse score >90 (Performance, Accessibility, Best Practices, SEO)
8. LCP < 2.5s, INP < 200ms, CLS < 0.1
9. Total transferred < 150KB
10. All animations disabled when `prefers-reduced-motion: reduce`

### cPanel (cpanel.ovav.dev)
11. Consistent color system with landing page
12. Section transitions (fade/slide) when switching between dashboard tabs
13. Loading skeletons while API calls are in-flight
14. Status indicators (pulse animation for active checks, color-coded states)
15. Deep-linkable sections via URL path or query params
16. Login page redesign aligned with new brand identity
17. Toast notifications with enter/exit animations
18. Responsive sidebar (collapse to icons on narrow screens)
19. Mobile-accessible (768px+ functional, 1024px+ optimal)

### Documentation (docs.ovav.dev)
20. Starlight theme matches OVAV brand colors and typography
21. Sidebar polish: active page indicator, smooth expand/collapse
22. Code blocks have copy button with feedback animation
23. Root redirect (`docs.ovav.dev/` to first doc page)
24. Mobile nav drawer with backdrop
25. Dark/light mode synced with OS preference or manual toggle

### Status (status.ovav.dev)
26. Custom CSS injects OVAV brand colors and logo into Better Uptime page

### Cross-Cutting
27. Single design token source used by all 4 surfaces
28. No surface uses hardcoded colors — all from design tokens
29. `prefers-reduced-motion` respected on all surfaces
30. Brand coherence: visual identity feels like one product across all domains

---

## 6. Context References

| Artifact | Path | Relevant For |
|---|---|---|
| Business Model (brand positioning) | `.ovav/plan/business_model.yaml` | Brand strategy, "Professional Development Governance" framing |
| Landing Copy Brief | `.ovav/plan/landing_copy_brief.yaml` | Exact copy for landing sections, visual notes from Sofia |
| Landing Source | Committed in repo (Next.js 14 + Tailwind) | Component tree, existing section structure |
| cPanel Frontend | `tools/cpanel/src/` | 10 section components, App.tsx routing, Login.tsx |
| cPanel Backend | `go-runtime/cmd/cpanel/` | API endpoints, auth, SSE |
| Docs Site Source | `docs-site/` | Astro config, Starlight sidebar, existing CSS |
| Docs Site Content | `docs-site/src/content/docs/` | 15 MDX/MD pages |
| Logo SVG | `docs-site/src/assets/ovav-logo.svg` | OVAV logo |
| Competitive Analysis | `docs/research/competitive_analysis_2026.md` | Visual references from competitor surfaces |
| DNS Config | `docs/infra/DNS_CONFIG_cpanel_split.md` | Domain mapping |

---

## 7. Notes for Delegates

- **Elena**: No hay sistema de diseño. Las 4 superficies usan estilos independientes. Tu primera decisión debe ser el sistema de tokens (CSS custom properties, Tailwind preset, o ambos). Las superficies ya están deployadas y funcionales — tu trabajo es darles identidad visual coherente. El reframing a "Professional Development Governance" debe leerse en el diseño. Las animaciones deben sentirse profesionales, no juguetonas.
- **Dante**: El código ya existe y funciona. Landing (Next.js 14 + Tailwind) y cPanel (React 18 + Vite) son codebases separadas. Evaluá si unificarlas o mantenerlas separadas. La integración de design tokens debe ser lo primero técnicamente. No hay tests de frontend visual — considerá Storybook o Chromatic si el presupuesto de animaciones lo justifica.
- **Ambos**: Product Hunt Jul 7. La landing page es la primera impresión. El cPanel es donde los usuarios viven. La coherencia visual entre ambas es clave para conversión y retención.
