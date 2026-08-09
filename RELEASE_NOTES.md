# OVAV Release Notes

## v2.0.0 — Go Runtime Production (2026-06-16)

**1,020+ commits. Go runtime 100% producto. Python gobernanza.**

### What's New

- **Go Runtime**: 14 paquetes, 89 archivos, 17,796 LOC. CLI, API, vault, install, validators, cockpit, tailor.
- **Validadores Go**: 11 validadores migrados (secrets, exfil, supply chain, protected branch, workspace safety, git push, permissions, runtime integrity, contracts, install verification, security policy).
- **API Hardening**: CORS restringido, CSRF state, rate limiting, SSE limits, path traversal URL-encoded, license HMAC-SHA256.
- **Security Policy**: 10 reglas estrictas NIST/CIS/OWASP-aligned.
- **Vault**: AES-256-GCM real en Go (Python XOR deprecado).
- **Docs-site**: 15 páginas, Starlight + Cloudflare Pages.

### Migration Progress

| Capa | Estado |
|------|--------|
| CLI | ✅ Go |
| API/cPanel | ✅ Go |
| Vault | ✅ Go (AES-256-GCM) |
| Install Pipeline | ✅ Go |
| Cockpit TUI | ✅ Go |
| Tailor | ✅ Go |
| Validators | 🟡 12.3% migrados |
| Harnesses | 🟡 Go install verification |
| Docs-site | ✅ 15 páginas |

### Security

- 0 DATA RACES (todos los tests pasan con -race)
- 0 secretos expuestos en plaintext
- 0 imports rotos
- Integrity Mesh VERDE 100%
- Defense Gate: PASS
- go vet: clean

### Known Limits

- Plugin installation: permanently denied for non-Thavren roles
- New public profiles: require creator authorization
- Force push/delete: permanently blocked on all surfaces
- Raw git push: prohibited on all surfaces
- Protected branches: CEO waiver required for writes

### Previous: v1.0.0 (2026-06-07)

Initial release. 597 commits. 66 validators. ConnectorBus. All phases A-F + S1-S12-E complete.
