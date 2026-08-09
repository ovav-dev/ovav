# UX Design — Mandatory Questions

> Loaded by `ovav-brainstorm` skill when UX/UI request detected.
> Ask 3-5 questions ONE AT A TIME. Wait for CEO response before next.

## UX Design Questions

### UXQ1: Accessibility Standard
"¿Cuál es el estándar de accesibilidad?"
- WCAG 2.1 AA (estándar general)
- WCAG 2.2 AAA (máxima accesibilidad)
- WCAG 2.1 A (mínimo, solo si no hay opción)

Why: Determina contraste mínimo, focus states, screen reader support, y puede requerir ARIA labels.

### UXQ2: Design System Strategy
"¿Design system propio o existente?"
- shadcn/ui (React, Tailwind, copy-paste, no library dependency)
- MUI / Ant Design / Chakra (component library)
- Diseño propio desde cero (Figma → código custom)
- Headless UI + Tailwind (libertad total)

Why: shadcn = máximo control + velocidad. MUI = rapidez inicial + deuda técnica. Propio = identidad única + más costo.

### UXQ3: Mobile-First o Desktop-First
"¿Cuál es el factor de forma primario?"
- Mobile-first (孝顺手机, PWA para desktop)
- Desktop-first (desktop como base, mobile secundario)
- Responsivo puro (mismo código, breakpoints)
- App nativa wrapper (Capacitor/Cordova)

Why: Determina breakpoints, touch targets (mínimo 44x44px), y estrategia de компоненты.

### UXQ4: Design Tokens
"¿Tokens de diseño (colores, spacing, typography)?"
- Figma Tokens plugin → código (Style Dictionary)
- Manual: CSS variables / Tailwind config
- No aplica (diseño hardcoded)

Why: Tokens permiten theming day/night y coherencia entre Figma y código sin duplicación.

### UXQ5: Animación y Motion
"¿Nivel de animación esperado?"
- Micro-interacciones mínimas (150ms hover, focus)
- Animaciones ricas (page transitions, skeleton loaders, motion design)
- Motion primer (Framer Motion / GSAP para interfaces complejas)
- Sin animaciones (accesibilidad, performance)

Why: Animaciones añaden percepción de velocidad pero aumentan bundle size y complejidad.

## Deliverable
After all questions answered → Elena adds `## 4. Design Language` and `## 5. Component Inventory` sections to DESIGN.md using answers above.
