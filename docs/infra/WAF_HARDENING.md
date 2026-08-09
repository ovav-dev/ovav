# OVAV cPanel — WAF Hardening Guide

> **Auditado por:** Nora · API & Security Engineer  
> **Para:** CEO · Configuración en Cloudflare Dashboard  
> **Tiempo:** 10 minutos

---

## 1. Security Level Settings

```
Cloudflare Dashboard → ovav.dev → Security → Settings
```

| Setting | Value | Why |
|---------|-------|-----|
| Security Level | **High** | cPanel es admin-only, no público |
| Challenge Passage | 30 min | Reduce fricción para operadores legítimos |
| Browser Integrity Check | **On** | Bloquea bots que inyectan headers maliciosos |
| Privacy Pass | On | Reduce CAPTCHAs para usuarios recurrentes |

---

## 2. DDoS Protection

```
Cloudflare Dashboard → ovav.dev → Security → DDoS
```

| Setting | Value |
|---------|-------|
| HTTP DDoS managed ruleset | **On**, Sensitivity: **High** |

> L3/L4 es automático en el edge de Cloudflare. Con Tunnel ni siquiera hay IP de origen expuesta.

---

## 3. Rate Limiting Rules

```
Cloudflare Dashboard → ovav.dev → Security → WAF → Rate limiting rules
```

### Rule 1: Auth endpoints (brute force protection)

| Field | Value |
|-------|-------|
| Rule name | `cPanel — Auth rate limit` |
| Field | URI Path |
| Operator | contains |
| Value | `/api/v1/auth/` |
| Requests | 10 |
| Period | 1 minute |
| Action | **Block** |
| Response type | 429 |

### Rule 2: General API (anti-abuse)

| Field | Value |
|-------|-------|
| Rule name | `cPanel — API rate limit` |
| Field | URI Path |
| Operator | starts with |
| Value | `/api/v1/` |
| Requests | 60 |
| Period | 1 minute |
| Action | **Managed Challenge** |

### Rule 3: Health endpoint (anti-recon)

| Field | Value |
|-------|-------|
| Rule name | `cPanel — Health rate limit` |
| Field | URI Path |
| Operator | contains |
| Value | `/health` |
| Requests | 30 |
| Period | 1 minute |
| Action | Managed Challenge |

> ⚠️ `/api/v1/events` (SSE) tiene timeouts separados y NO debe pasar por rate limiting estricto. La regla #2 con Managed Challenge es suficiente (no Block).

---

## 4. Geo-Blocking

```
Cloudflare Dashboard → ovav.dev → Security → WAF → Custom rules → Create rule
```

| Field | Value |
|-------|-------|
| Rule name | `cPanel — Geo allowlist` |
| Expression | `(not ip.geoip.country in {"PE" "US" "CL" "CO"})` |
| Action | **Managed Challenge** |

> Perú (donde operás) + US (OAuth providers: Google, GitHub) + vecinos opcionales.  
> Managed Challenge en vez de Block: si viajás, pasás el challenge y entrás.

---

## 5. Managed Rulesets (OWASP + Cloudflare)

```
Cloudflare Dashboard → ovav.dev → Security → WAF → Managed rulesets
```

| Ruleset | Action | Mode |
|---------|--------|------|
| **Cloudflare Managed Ruleset** | ✅ Deploy | Default |
| **OWASP Core Ruleset** | ✅ Deploy | Anomaly scoring: High |
| **OWASP XSS** | Deploy (si está separado) | Block |
| **OWASP SQL Injection** | Deploy (si está separado) | Block |

---

## 6. Bot Fight Mode

```
Cloudflare Dashboard → ovav.dev → Security → Bots
```

| Setting | Value |
|---------|-------|
| Bot Fight Mode | **On** |
| Definitely Automated | Block |
| Likely Automated | Managed Challenge |

---

## 7. Custom Rule: Block Known Attack Patterns

```
Cloudflare Dashboard → ovav.dev → Security → WAF → Custom rules → Create rule
```

| Field | Value |
|-------|-------|
| Rule name | `cPanel — Block malicious patterns` |
| Expression | `(http.request.uri.path contains \"../\") or (http.request.uri contains \"%2e%2e\") or (http.user_agent eq \"\")` |
| Action | **Block** |

---

## Verificación post-configuración

```bash
# 1. Rate limiting funciona
for i in $(seq 1 15); do
  curl -s -o /dev/null -w "%{http_code}\n" https://TU-SUBDOMINIO.ovav.dev/api/v1/auth/login
done
# Después de 10 requests en 1 min → 429

# 2. Geo-blocking (desde fuera de Perú/US)
# Usar VPN a Europa → visitar URL → CAPTCHA

# 3. Path traversal bloqueado
curl -s https://TU-SUBDOMINIO.ovav.dev/../../../etc/passwd
# → Cloudflare Block (no llega al backend)
```
