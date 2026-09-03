# Cloudflare Access blocks /api/v1/auth/* — 2026-08-19 diagnosis

## TL;DR

The endpoint `https://d678beea.ovav.dev/api/v1/auth/login-challenge-web`
is **protected by Cloudflare Access (Zero Trust)**. Any request without
proper Cloudflare Access auth gets a 302 redirect to
`https://ovav.cloudflareaccess.com/cdn-cgi/access/login/...` — NOT the
JSON contract the OVAV web auth flow expects.

This is the root cause of:
- `ovav login --web` → "Web vault auth failed: invalid character '<' looking for beginning of value"
- `ovav auth web --check` → "web backend NOT ready" (R-3 refusal — correct behavior)

## Reproduction

```bash
$ curl -sS --max-time 8 -i -H "Accept: application/json" \
    "https://d678beea.ovav.dev/api/v1/auth/login-challenge-web"
HTTP/2 302
date: Wed, 19 Aug 2026 20:47:14 GMT
content-type: text/html; charset=UTF-8
location: https://ovav.cloudflareaccess.com/cdn-cgi/access/login/d678beea.ovav.dev?...
www-authenticate: Cloudflare-Access resource_metadata="https://d678beea.ovav.dev/.well-known/cloudflare-access-protected-resource/api/v1/auth/login-challenge-web"
server: cloudflare

<html>
<head><title>302 Found</title></title></head>
<body><center><h1>302 Found</h1></center>...
```

Headers confirm:
- `server: cloudflare`
- `cf-ray: a2dc03264c21ac95-MIA`
- `www-authenticate: Cloudflare-Access ...`

## Why this blocks `ovav auth web`

The OVAV auth flow expects:

```
GET /api/v1/auth/login-challenge-web
→ 200 OK
→ {"challenge": "abc123...", "expires_in": 600, "login_url": "..."}
```

Reality:
```
GET /api/v1/auth/login-challenge-web
→ 302 Found → ovav.cloudflareaccess.com
→ {"error": "unauthorized"}  (after following redirect, which we don't)
```

The `cmdLoginWeb` (legacy) and `auth/web.go` (new) code path doesn't
follow the auth redirect — it tries to parse HTML as JSON, fails with
`invalid character '<'`.

## Server-side fixes (choose one)

### Option A — Exempt `/api/v1/auth/*` from Cloudflare Access (preferred)

In the Cloudflare Zero Trust dashboard for the `d678beea` app:

1. **Access → Applications → d678beea.ovav.dev → Policies**
2. Edit the policy that protects `api/v1/auth/*`
3. Add an **Allow** rule for path `api/v1/auth/*` with no identity check
   (or with a Service Token requirement, see Option B)
4. Save + deploy

If you have the Cloudflare CLI:

```bash
# Pre-flight
wrangler tail --format=pretty  # watch live traffic during fix

# Apply (replace $APP_ID with your Access app UUID)
curl -X PATCH \
  "https://api.cloudflare.com/client/v4/access/apps/$APP_ID" \
  -H "Authorization: Bearer $CF_API_TOKEN" \
  -H "Content-Type: application/json" \
  --data '{
    "policies": [
      {
        "name": "ovav-public-auth",
        "decision": "allow",
        "include": [{ "everyone": {} }],
        "resource_types": ["api"],
        "paths": ["/api/v1/auth/*"]
      },
      {
        "name": "ovav-default-deny",
        "decision": "deny",
        "include": [{ "everyone": {} }],
        "resource_types": ["api"],
        "paths": ["*"]
      }
    ]
  }'
```

Verify:
```bash
$ curl -sS -H "Accept: application/json" \
    "https://d678beea.ovav.dev/api/v1/auth/login-challenge-web" | jq .
{
  "challenge": "...",
  "expires_in": 600,
  "login_url": "..."
}
```

### Option B — Service Tokens (zero changes to Access policy)

1. Cloudflare dashboard → **Access → Service Auth → Create Token**
2. Generate Client ID + Client Secret
3. Pass them to ovav on each call:

```bash
export CF_ACCESS_CLIENT_ID="..."
export CF_ACCESS_CLIENT_SECRET="..."
ovav auth web       # new binary picks these up automatically
ovav login --web    # legacy path also picks them up (one-line patch in cmdLoginWeb)
```

The CLI would send `Cf-Access-Client-Id` and `Cf-Access-Client-Secret`
headers on every request.

### Option C — Move the API out of Cloudflare Access entirely

If the backend is on Fly.io behind a Cloudflare Tunnel, you can split
the tunnel into two apps:

| Subdomain | Tunnel | Access |
|---|---|---|
| `d678beea.ovav.dev` | App A (existing) | Protected (admin UI, etc.) |
| `api.d678beea.ovav.dev` | App B (new) | Public |

Use `api.d678beea.ovav.dev` for the auth flow. Update
`OVAV_WEB_URL` env var accordingly.

## Decision matrix

| Option | Effort | Risk | Recommended for |
|---|---|---|---|
| A — exempt path | 5 min | low (narrow allowlist) | Production users who self-host the backend |
| B — service tokens | 15 min + secret rotation | medium (token leak) | Short-term until Cloudflare rule fixed |
| C — separate tunnel | 30 min | low | Long-term clean architecture |

## What needs to change in source

After picking an option, two of the auth paths need a tweak:

**`cmd/ovav/auth/web.go`** (new path):
- Add CF_ACCESS_CLIENT_ID/SECRET support via env vars
- If `OVAV_WEB_URL` differs from `https://ovav.dev` (no Cloudflare), skip

**`cmd/ovav/login.go`** (legacy `cmdLoginWeb`):
- Same CF_ACCESS_CLIENT_ID support — keep back-compat

## Quick triage command (for future incidents)

```bash
# Wrap `ovav auth web --check` with diagnostics
ovav-auth-diagnostic() {
  echo "── Backend DNS ──"
  getent hosts d678beea.ovav.dev
  echo "── TCP connect ──"
  timeout 5 bash -c 'cat </dev/tcp/d678beea.ovav.dev/443' 2>&1 | head -2
  echo "── TLS handshake ──"
  echo | timeout 5 openssl s_client -connect d678beea.ovav.dev:443 -servername d678beea.ovav.dev 2>&1 | grep -E "subject=|issuer=" | head -2
  echo "── HTTP probe ──"
  curl -sS -H "Accept: application/json" -o /dev/null \
       -w "  status=%{http_code}\n  location=%{redirect_url}\n  server=%{header_server}\n" \
       "https://d678beea.ovav.dev/api/v1/health"
}
```
