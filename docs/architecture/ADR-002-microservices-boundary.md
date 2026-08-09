# ADR-002: Microservices Deployment Boundary

**Date:** 2026-06-17  
**Status:** Accepted  
**Decider:** Thavren + Uriel

## Context

OVAV serves multiple surfaces: landing page (ovav.dev), cPanel (cpanel.ovav.dev), documentation (docs.ovav.dev), and status page (status.ovav.dev).

## Decision

| Surface | Platform | Runtime |
|---------|----------|---------|
| ovav.dev | Cloudflare Pages | Next.js static export |
| cpanel.ovav.dev | Fly.io | Go stdlib HTTP server |
| docs.ovav.dev | Cloudflare Pages | Starlight (Astro) |
| status.ovav.dev | Better Uptime | Managed |

## Consequences

- Each surface independently deployable
- cPanel in USA (DFW region) via Fly.io
- Static sites on Cloudflare global CDN
