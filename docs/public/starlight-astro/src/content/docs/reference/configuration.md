---
title: Configuration
description: OVAV configuration reference — environment variables, config files, and deployment settings.
---

OVAV uses a combination of environment variables and YAML/JSON config files.

## Environment Variables

### cPanel Server

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server listen port | `8080` |
| `CPANEL_ALLOWED_ORIGINS` | Comma-separated CORS origins | `localhost:3000,ovav.dev` |
| `OAUTH_GOOGLE_CLIENT_ID` | Google OAuth client ID | — |
| `OAUTH_GOOGLE_CLIENT_SECRET` | Google OAuth client secret | — |
| `OAUTH_GITHUB_CLIENT_ID` | GitHub OAuth client ID | — |
| `OAUTH_GITHUB_CLIENT_SECRET` | GitHub OAuth client secret | — |
| `OAUTH_REDIRECT_URI` | OAuth callback URL | `https://cpanel.ovav.dev` |
| `ADMIN_EMAILS` | Comma-separated admin emails | — |

### Security

| Variable | Description | Default |
|----------|-------------|---------|
| `OVAV_LICENSE_HMAC_KEY` | HMAC key for license signing | Built-in default |

## Config Files

### `.ovav/policy/permission_authority.json`
Canonical permission authority. Defines operator permissions, F0-F5 security layers, and protected denies.

### `.ovav/plan/caps.yaml`
Implementation plan — canonical source of truth for OVAV progress tracking.

### `.ovav/registry/auto_triggers.yaml`
80+ auto-triggers for validation pipeline automation.

### `opencode.json`
OpenCode client configuration — agent routing, skill mappings, theme settings.

### `wrangler.toml`
Cloudflare Pages deployment config for docs-site.

### `fly.toml`
Fly.io deployment config for cPanel server.

## Fish Shell Config

Located in `config/fish/`:
- `20-ovav-wezterm-osc7.fish` — WezTerm OSC7 integration
- `25-ovav-wezterm-git.fish` — Git status in prompt
- `30-ovav-runtime-tools.fish` — Runtime tool aliases
- `90-ovav-terminal-auto.fish` — Terminal auto-detection

## WezTerm Config

- `config/wezterm/wezterm.lua` — Full WezTerm config
- `config/wezterm/wezterm-fallback-minimal.lua` — Minimal fallback

## Git Config

- `config/git/aliases.gitconfig` — OVAV git aliases (`owc`, `owd`, etc.)
