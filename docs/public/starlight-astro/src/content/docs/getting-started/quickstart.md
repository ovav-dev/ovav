---
title: Quickstart
description: Get OVAV running on your machine in under 5 minutes.
---

## Prerequisites

- **Go 1.24+** (for OVAV runtime)
- **Node.js 20+** (for cPanel frontend)
- **Git** (for version control)
- **Linux, macOS, or WSL2**

## Install

```bash
# Clone and install OVAV
git clone https://github.com/ovav-ai/ovav.git
cd ovav
go run ./cmd/ovav install
```

## First profile

```bash
# Apply your first professional profile
ovav profile apply platform-engineering

# Launch the cockpit
ovav cockpit
```

## Verify

```bash
# Check system health
ovav status

# Run the native health check
curl http://localhost:5858/health
```

## Next

- [Architecture overview](/core/architecture)
- [First profile deep-dive](/guides/first-profile)
- [CLI reference](/reference/cli)
