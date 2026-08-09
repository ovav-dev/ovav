---
title: Architecture
description: OVAV system architecture — Go runtime, TypeScript frontend, Python governance.
---

OVAV runs on a three-layer architecture optimized for performance, security, and developer experience.

## Layer 1: Go Runtime (Product)

All user-facing product code runs in **Go 1.24+**, compiled to static binaries with zero dependencies.

| Component | Purpose | Size |
|---|---|---|
| **Cockpit TUI** | Terminal dashboard (Bubble Tea) | 18 files, 8 views |
| **cPanel Backend** | Admin API (29 endpoints, JWT RS256) | 14 files, 2125 loc |
| **Vault** | AES-256-GCM encryption | 8 files, 14 tests |
| **Tailor** | Development plan composer | 6 files, 32 tests |
| **Install Gateway** | Safe install, backup, rollback | 12 files, 81 tests |
| **CLI** | Unified dispatch (bin/ovav) | 11 handlers |

**Why Go?** Static binaries (~3MB), no runtime dependencies, stdlib-only cryptography, cross-compile to 5 platforms.

## Layer 2: TypeScript Frontend (Browser)

The **cPanel web interface** is a React 18 + Vite SPA served by the Go backend. OAuth authentication (Google, GitHub) with JWT sessions.

**Why TypeScript?** Browser runtime requires JavaScript. React 18 for component model, Vite for fast builds. Static export for CF Pages (landing).

## Layer 3: Python Governance (Operational)

Internal governance tools that keep the system running.

| Category | Files | Role |
|---|---|---|
| Validators | 73 | Runtime checks, integrity, pre-commit |
| Harnesses | 332 | Safety gates, security checks |
| Agent Runtime | 64 | Session management, chronos |
| Governor | 27 | Output guard, visual formatter |

**Python is NOT the product.** It is the operating system of OVAV — never exposed to end users.

## Infrastructure

```
ovav.dev        → Cloudflare Pages (landing, Next.js static)
cpanel.ovav.dev → Fly.io (Go backend + React SPA)
docs.ovav.dev   → Cloudflare Pages (Starlight, Astro)
```

See [Deployment guide](/reference/configuration) for details.
