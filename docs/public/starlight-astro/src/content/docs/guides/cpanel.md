---
title: cPanel Administration
description: Guide to administering OVAV cPanel — the Go-powered control panel API.
---

OVAV cPanel is the central administration API for managing OVAV profiles, agents, vault, and system status.

## Overview

cPanel is built in **Go** (stdlib-only) and runs on Fly.io. It provides:

- **JWT RS256 authentication** with OAuth support (Google, GitHub)
- **REST API** with 29+ endpoints
- **Server-Sent Events** for real-time system monitoring
- **Static file serving** for the React SPA frontend

## Quick Start

```bash
# Start cPanel server locally
cd go-runtime && go run ./cmd/cpanel/

# Server starts on :8080 by default
# Health check: http://localhost:8080/health
```

## Authentication

cPanel supports two authentication methods:

### Token-based auth
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"token": "your-32-char-or-longer-token"}'
```

### OAuth (Google/GitHub)
Configure environment variables:
```bash
export OAUTH_GOOGLE_CLIENT_ID=your-client-id
export OAUTH_GOOGLE_CLIENT_SECRET=your-client-secret
export OAUTH_REDIRECT_URI=https://cpanel.ovav.dev
```

## Security

- **Rate limiting**: 5 attempts/minute per IP on auth endpoints
- **CSRF protection**: State parameter verification on OAuth callbacks
- **CORS**: Restricted to authorized origins (no wildcard)
- **SSE limits**: Maximum 100 concurrent connections
- **Path traversal**: URL-encoded variant detection
- **License**: HMAC-SHA256 signed, forgery-resistant

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/api/v1/status` | System status (git, memory, uptime) |
| POST | `/api/v1/auth/login` | Token authentication |
| POST | `/api/v1/auth/oauth/{provider}` | OAuth callback |
| GET | `/api/v1/auth/session` | Active session info |
| GET | `/api/v1/auth/config` | Available auth methods |
| GET | `/api/v1/events` | SSE event stream |
| GET | `/api/v1/memory` | Memory stats |
| GET | `/api/v1/profiles` | Profile management |
| GET | `/api/v1/security` | Security status |
| GET | `/api/v1/system` | System diagnostics |
