---
title: API Reference
description: OVAV cPanel REST API reference — all endpoints, request/response schemas, and authentication.
---

The cPanel API provides programmatic access to OVAV system management.

**Base URL:** `https://cpanel.ovav.dev`

## Authentication

All `/api/v1/*` endpoints require authentication via JWT Bearer token:
```
Authorization: Bearer <token>
```

Obtain a token via:
- `POST /api/v1/auth/login` (token-based)
- `POST /api/v1/auth/oauth/{provider}` (OAuth)

## Endpoints

### Health
```
GET /health
```
Returns `{"status":"ok","version":"5.1.0"}`

### Status
```
GET /api/v1/status
```
Returns system status including git info, memory usage, uptime, and integrity mesh score.

### Authentication
```
POST /api/v1/auth/login
Content-Type: application/json
{"token": "your-32-character-token"}

Response: {"token":"jwt...","role":"admin","expires_at":1740000000}
```

```
POST /api/v1/auth/oauth/google
Content-Type: application/json
{"code":"oauth-code","state":"csrf-state-token"}
```

```
GET /api/v1/auth/session
Authorization: Bearer <token>

Response: {"token":"...","role":"admin","created_at":"...","expires_at":"..."}
```

```
GET /api/v1/auth/config
Response: {"methods":["oauth","token"],"oauth":[...],"has_oauth":true}
```

### Events (SSE)
```
GET /api/v1/events
Accept: text/event-stream

Stream:
event: connected
data: {"status":"connected","time":"2026-06-16T10:00:00Z"}

event: heartbeat
data: {"time":"2026-06-16T10:00:15Z"}
```

### Memory
```
GET /api/v1/memory
Authorization: Bearer <token>

Response: {"alloc":"15MB","sys":"32MB","num_gc":42,"goroutines":8}
```

### Profiles
```
GET /api/v1/profiles
Authorization: Bearer <token>

Response: {"profiles":[...]}
```

### Security
```
GET /api/v1/security
Authorization: Bearer <token>

Response: {"integrity_mesh":"healthy","score":100,"validators":11}
```

### System
```
GET /api/v1/system
Authorization: Bearer <token>

Response: {"go_version":"1.24","cpanel_version":"5.1.0","os":"linux","arch":"amd64"}
```

## Error Responses

All errors follow this format:
```json
{"error": "human-readable error message"}
```

HTTP status codes:
- `200` — Success
- `400` — Bad request (invalid input)
- `401` — Unauthorized (invalid/missing token)
- `403` — Forbidden (CSRF failure, rate limited)
- `429` — Too many requests (rate limit exceeded)
- `500` — Internal server error
- `503` — Service unavailable (OAuth not configured, SSE at capacity)
