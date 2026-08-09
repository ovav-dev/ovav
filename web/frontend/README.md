# OVAV Product — Frontend

Plataforma web pública de OVAV. Next.js 14 (App Router) + TypeScript + TailwindCSS.

## Requisitos

- **pnpm** 9.x (npm PROHIBIDO por política de seguridad)
- Node.js 20+
- Cuenta de Stripe (test mode para desarrollo)
- Cuenta de Resend (para emails transaccionales)

## Instalación

```bash
pnpm install
cp .env.example .env.local
# Editar .env.local con credenciales
pnpm dev
```

## Estructura

```
src/
├── app/           # Next.js App Router
│   ├── auth/      # Login, registro, magic link
│   ├── checkout/  # Flujo de compra
│   ├── dashboard/ # Portal de cliente (protegido)
│   ├── docs/      # Documentación pública
│   └── api/       # API routes (proxies al backend)
├── components/    # Componentes reutilizables
├── lib/           # Utilidades, tipos, config
└── styles/        # Estilos globales
```

## Seguridad

- **npm PROHIBIDO.** Usar pnpm install --frozen-lockfile en CI.
- Monitoreo continuo: GitHub Advisories, Socket.dev, Snyk (revisión semanal).
