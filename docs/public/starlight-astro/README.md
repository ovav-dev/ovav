# OVAV Documentation Site

Starlight (Astro) + Cloudflare Pages.

## Setup

```bash
pnpm install
pnpm dev     # http://localhost:4321
pnpm build   # static export to dist/
```

## Deploy

```bash
# Option A: Cloudflare Pages git integration (recommended)
# 1. Push to main branch of ovav-docs repo
# 2. Cloudflare Pages auto-deploys

# Option B: Manual deploy via wrangler
pnpm build
npx wrangler pages deploy dist/ --project-name ovav-docs --branch main
```

## Content structure

```
src/content/docs/
  getting-started/
    intro.md          ← Landing
    quickstart.md     ← 5-min setup
    installation.md   ← Full install
  core/
    architecture.md   ← System design
    profiles.md       ← 8 professional profiles
    governance.md     ← Governance model
    multi-model.md    ← Multi-model runtime
  guides/
    first-profile.md  ← Profile setup
    tailor.md         ← Tailor composer
    vault.md          ← Encryption
    cpanel.md         ← Admin panel
  reference/
    cli.md            ← CLI commands
    api.md            ← API reference
    security.md       ← Security docs
    configuration.md  ← Config reference
```

## Pages to complete

- [x] Introduction
- [x] Quickstart
- [x] Installation
- [x] Architecture
- [x] Profiles
- [x] Security
- [ ] Governance model
- [ ] Multi-model runtime
- [ ] First profile guide
- [ ] Tailor composer guide
- [ ] Vault encryption guide
- [ ] cPanel administration guide
- [ ] CLI reference
- [ ] API reference
- [ ] Configuration reference
