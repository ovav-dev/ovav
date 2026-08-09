<p align="center">
  <img src="assets/readme/ovav-hero.png" width="100%" alt="OVAV — governed AI workstation" />
</p>

<h1 align="center">OVAV</h1>

<p align="center">
  <strong>Governed Collective Intelligence System.</strong><br/>
  OVAV-first architecture · gated writes · verifiable runtime · knowledge-aware.
</p>

<p align="center">
  <a href="docs/QUICKSTART.md">Quick Start</a> ·
  <a href=".ovav/plan/caps.yaml">Strategic Plan (caps.yaml)</a> ·
  <a href="docs/lab/ARCHITECTURE_LAB.md">Architecture Lab</a> ·
  <a href="CHANGELOG.md">Changelog</a>
</p>

<p align="center">
  <img alt="Status" src="https://img.shields.io/badge/status-Integrity%20Mesh%20Active-7c3aed" />
  <img alt="Mode" src="https://img.shields.io/badge/mode-source--local-111827" />
  <img alt="Platform" src="https://img.shields.io/badge/platform-Linux%20%7C%20WSL2-0f766e" />
  <img alt="Python" src="https://img.shields.io/badge/python-3.11%2B-f59e0b?logo=python" />
  <img alt="Safety" src="https://img.shields.io/badge/safety-gated%20writes%20%2B%20auto--sync-22c55e" />
</p>

---

## What OVAV does

OVAV turns a repository into a governed AI workstation cockpit and is evolving into a source-local Collective Intelligence System.

It is not a random dotfiles pack and it is not just a chatbot wrapper. OVAV provides a source-local control plane for AI-assisted work: command navigation, OpenCode surfaces, validation gates, plan artifacts, source-local backup/rollback and publication checks. The current route adds a **Knowledge Compiler** that will compile handoffs, ledger entries, commits and artifacts into operational knowledge.

```txt
source repo -> cockpit -> plan -> gate -> backup -> verify -> publish
```

---

## Why it exists

| Before | With OVAV |
|---|---|
| AI tools drift across scattered configs. | Work starts from a governed source-local baseline. |
| Commands become manual memory work. | The cockpit recommends the next safe action. |
| Context leaks between roles. | OpenCode surfaces are checked and routed. |
| Writes are hard to audit. | Apply paths require consent and risk acceptance. |
| Release prep is fragile. | Publish, smoke and fresh-archive checks are first-class gates. |

---

## Core capabilities

| Capability | Current behavior |
|---|---|
| **CLI mother cockpit** | `bin/ovav` opens the guided source-local cockpit. |
| **OpenCode-first workflow** | Checks repo-local agents, commands and skills before use. |
| **Plan artifacts** | `setup`, `sync`, `security`, `recovery` and `update` can emit JSON plans. |
| **Execution gateway** | `--apply` requires explicit `--consent --accept-risk`. |
| **Managed backup/rollback** | Source-local managed files can be backed up and restored safely. |
| **Surface manager** | Audits OpenCode, CLI tools, validators and release docs. |
| **Public export gate** | Blocks secret-like payloads, generated runtime files and unsafe release state. |
| **Fresh clone smoke** | Tests a clean source archive as a user would receive it. |

---

## Quick start

```sh
curl -fsSL https://raw.githubusercontent.com/ovav-dev/ovav-systems/main/install.sh | sh
```

After install, use the single entrypoint:

```sh
ovav
```

The cockpit handles setup, configuration, updates, recovery and diagnostics through navigation. Advanced commands remain available, but they are not the primary user experience.

---

## CLI cockpit

```txt
ovav                     # guided cockpit
ovav security            # secret and payload posture
ovav surfaces            # OpenCode / CLI / docs surface status
ovav sync --plan-json    # source-local sync plan
ovav backup --plan       # managed backup plan
ovav backup --create --consent
ovav rollback --plan
ovav release-check       # RC package verification
ovav publish-check       # public export gate
ovav fresh-smoke         # fresh archive user journey
ovav smoke               # practical sandbox smoke
```

---

## How the system is governed

<p align="center">
  <img src="assets/readme/ovav-flow.svg" width="100%" alt="OVAV governance flow" />
</p>

<details>
<summary><strong>Safety model</strong></summary>

| Layer | Rule |
|---|---|
| Planning | Preview first; JSON plans must declare write posture. |
| Apply | Requires explicit consent and risk acceptance. |
| Backup | Managed source-local backups are created before risky restore paths. |
| Validation | Release checks run before promotion or remote publication. |
| Publication | Runtime outputs, secrets, archives, caches and private paths are blocked. |
| Global surfaces | Global config, plugin install, live Engram, MCP/A2A and production readiness claims remain gated. |

</details>

---

## Repository layout

```txt
OVAV/
├── bin/                         # public CLI entrypoints
├── assets/readme/               # GitHub visual presentation
├── docs/                        # public docs and architecture notes
├── .opencode/                   # repo-local OpenCode surfaces
├── .ovav/                       # governance artifacts and policies
├── registry/                    # harness/eval/artifact registry
├── runtime/                     # source runtime references
├── tests/                       # fixtures, golden checks and evaluations
├── tools/cli/                   # public CLI support modules
├── tools/validators/            # release and authority validators
└── README.md                    # repo entrypoint
```

---

## Documentation

| Topic | File |
|---|---|
| Start here | [QUICKSTART.md](docs/QUICKSTART.md) |
| CLI cockpit | [docs/CLI.md](docs/CLI.md) |
| Intended usage | [docs/INTENDED_USAGE.md](docs/INTENDED_USAGE.md) |
| Architecture | [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) |
| Governance | [docs/GOVERNANCE.md](docs/GOVERNANCE.md) |
| Release checklist | [docs/RELEASE_CHECKLIST.md](docs/RELEASE_CHECKLIST.md) |
| Brand assets | [docs/BRAND_ASSETS.md](docs/BRAND_ASSETS.md) |
| Roadmap | [docs/ROADMAP.md](docs/ROADMAP.md) |
| Security | [SECURITY.md](docs/SECURITY.md) |
| Privacy | [PRIVACY.md](docs/PRIVACY.md) |
| Support | [SUPPORT.md](docs/SUPPORT.md) |

---

## Current release posture

| Item | State |
|---|---|
| Version posture | `Final Launch Verification` |
| Primary mode | Source-local |
| Remote publication | Prepared, gated |
| Global install | Gated |
| Plugin install | Blocked |
| Live Engram | Blocked |
| MCP/A2A | Blocked |
| Production/global readiness claim | Blocked until a later release gate |

## Current strategic route

**P0 completado: Knowledge Compiler consolidation.** Próximo: **surface evolution** — CLI RC10 con NerveBus, MCP/RAG sobre KC+SNV, Dashboard↔Lockdown Authority.

## Build Status (live)

| System | State |
|---|---|
| Integrity Mesh (F0-F4) | ✅ 62 validators, 0 rotos |
| Behavioral Directives | ✅ 19 reglas session-persistent con confidence scoring |
| Memory Governor | ✅ capsule-bound, F5-gated, classify→redact→poison→decide |
| State Sync Engine | ❌ ELIMINADO — git HEAD es la fuente de verdad inmutable |
| Session Capsule | ✅ auto-activate + auto-diagnosis |
| L0-L7 Intelligence Stack | ✅ 8 capas completas: Identity→Capsule→Router→Model→Observe→Firewall→Security→Feedback |
| Knowledge Compiler P0.2 | ✅ 4 motores: Pattern Detector, Alignment Engine, Transition Detector, Criterion Compiler |
| Sistema Nervioso Vivo | ✅ 7 módulos: NerveBus, KnowledgeGraph, HebbianWeights, TemporalCortex, PatternLearner, SNV Bridge, Dashboard Interactivo |
| Living Intelligence 10-12 | ✅ Capas de inteligencia cableadas sobre KC+SNV |
| Credential Vault F1 | ✅ Bóveda encriptada AES-256, separación personal/producto, auditoría |
| Git Transport | ✅ HTTPS-only; gate anti-mezcla SSH/HTTPS |
| Evaluation Pipeline | ✅ auto-generates current diff packet; rollback-aware |
| Trigger Engine | ✅ 12/12 triggers pass |
| Total módulos | 623 Python modules, 51 validators, 367 harnesses, 0 rotos |
| Próximo | Surface evolution — CLI RC10, MCP/RAG, SNV hardening |

---

## What OVAV is not

OVAV is intentionally opinionated.

It is not:

- a generic AI marketplace
- a loose prompt collection
- a support-every-workflow framework
- a production/global installer in current release
- a replacement for human release judgment

OVAV is a governed workstation system with a strict release discipline.
