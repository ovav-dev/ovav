---
title: First Profile Setup
description: Step-by-step guide to applying your first professional profile in OVAV.
---

This guide walks you through applying your first professional profile in OVAV. We recommend **Platform Engineering** as a starting point — it gives you the full runtime, security, and CLI experience.

## Prerequisites

- OVAV installed (`ovav version` should show version info)
- Go 1.24+ runtime available
- Git repository initialized

## Step 1: Check Your Current State

```bash
ovav status
```

This shows your current system posture, including:

- Go runtime version and platform
- Git branch and commit
- Repository root detection
- Current profile status

## Step 2: List Available Profiles

```bash
ovav profile list
```

You should see all 8 professional profiles:

| Profile | Description |
|---|---|
| `platform-engineering` | Runtime, security, terminal, CLI |
| `digital-product-engineering` | Full-stack, frontend, backend |
| `evidence-decision-intelligence` | Research, benchmarks, source verification |
| `education-career-development` | Learning paths, skill assessment |
| `health-performance-science` | Nutrition, fitness, medical research |
| `commercial-growth-strategy` | Business, pricing, go-to-market |
| `devops-infrastructure` | CI/CD, cloud, SRE, monitoring |
| `ui-ux-design` | Design system, accessibility, prototyping |

## Step 3: Apply Platform Engineering

```bash
ovav profile apply platform-engineering
```

This configures:

- **Runtime** — Go toolchain, validators, install gateway
- **Security** — Vault encryption, audit logging
- **Terminal** — Shell configuration, session management
- **CLI** — Full `ovav` command access

The profile is **additive** — you can apply multiple profiles and they merge intelligently.

## Step 4: Verify the Profile

```bash
ovav profile list
```

The applied profile should show as active. You can also run:

```bash
ovav doctor
```

This runs a full system diagnostic and confirms the profile is correctly configured.

## Step 5: Use the Tailor Composer

The Tailor Composer lets you fine-tune your workstation setup:

```bash
ovav tailor
```

This shows your current plan and available tools:

```
Plan: Core | 2 tools · 1 role
  [✓] OpenCode     governed AI workspace
  [✓] Git          safe versioning
  [ ] Neovim       technical editing
  [ ] Zellij       live sessions
  [ ] Fish         productive shell
  [✓] Platform Engineering    systems, CLI and runtime
```

### Plan Levels

| Plan | Unlocks |
|---|---|
| **Nucleo** (Core) | OpenCode, Git, Platform Engineering |
| **Studio** | Neovim, Zellij, Fish, Research Intelligence |
| **Command** | Security Architecture, advanced operations |

Select a higher plan to unlock more tools:

```bash
ovav tailor select studio
```

## Step 6: Launch the Cockpit

The Cockpit is a terminal dashboard for monitoring OVAV:

```bash
ovav cockpit
```

The Cockpit provides 8 views:

| View | Purpose |
|---|---|
| Welcome | System overview |
| Root | Repository status |
| Dashboard | Active tools and profiles |
| Health | System diagnostics |
| Install | Installation management |
| Tailor | Workstation composer |
| Detail | Component details |
| Quit | Exit |

## Next Steps

- [Tailor Composer Guide](/guides/tailor) — Create custom development plans
- [Vault Encryption](/guides/vault) — Secure your sensitive assets
- [cPanel Administration](/guides/cpanel) — Web-based management
- [CLI Reference](/reference/cli) — Full command documentation
