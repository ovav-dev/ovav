---
title: Security
description: OVAV security architecture — vault encryption, audit trails, and governance model.
---

## Vault Encryption

OVAV vault uses **AES-256-GCM** with PBKDF2 key derivation.

```bash
ovav vault scan      # Discover sensitive assets
ovav vault encrypt   # Encrypt all discovered assets
ovav vault decrypt   # Decrypt for access
ovav vault gen-key   # Generate new master key
```

All encryption is **local-first** — keys never leave your machine.

## Authentication

- **OAuth 2.0**: Google and GitHub (cPanel)
- **JWT RS256**: Session tokens with role-based claims
- **Token auth**: Development mode access

## Audit trail

Every governed action is logged:
- Profile application/removal
- Tool execution with parameters
- Model switches and configuration changes
- Security events (canary alarms)

## Supply chain

OVAV is **stdlib-only** on the Go side:
- Zero third-party Go dependencies
- Static binary compilation
- Reproducible builds via Go modules

## Reporting

Found a security issue? Email `security@ovav.dev`. See our [security policy](https://github.com/ovav-ai/ovav/security) for details.
