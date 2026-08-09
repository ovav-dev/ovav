---
title: Multi-Model Runtime
description: How OVAV orchestrates multiple AI models with vendor independence and intelligent model switching.
---

OVAV does not sell AI models. It **governs** them. You bring your own models — OpenAI, Anthropic, Google, open-source — and OVAV orchestrates them across your workflow.

## Why Multi-Model?

Single-vendor lock-in is a risk. Models have different strengths:

| Strength | Best Models |
|---|---|
| Code generation | Claude, GPT-4 |
| Reasoning | o1, Claude |
| Speed | GPT-4o-mini, Gemini Flash |
| Cost efficiency | Open-source (Llama, Qwen) |
| Long context | Claude, Gemini |

OVAV lets you use the right model for each task without changing your workflow.

## Model Orchestration

OVAV routes work to models based on:

1. **Task type** — code, research, design, governance
2. **Complexity** — simple queries vs. multi-step reasoning
3. **Cost budget** — economy tier controls spending
4. **Availability** — fallback if a model is degraded

### Routing Flow

```
Task → Profile → Service Lane → Model Selection → Execution → Validation
```

Each professional profile defines which models are appropriate for its domain. The Governor enforces these choices.

## Vendor Independence

OVAV's architecture is **model-agnostic** at every layer:

- **CLI** — `ovav` dispatch works regardless of backend model
- **cPanel** — API endpoints return data, not model-specific output
- **Cockpit TUI** — terminal dashboard shows results, not model internals
- **Profiles** — define workflows, not model bindings

This means you can switch your entire model stack without reconfiguring OVAV.

## Model Switching

Model switching is controlled by the permission authority:

| Role | Can Switch? |
|---|---|
| CEO | Yes — full control |
| Thavren (Lead) | Yes — governed switching |
| Other agents | No — model assigned by profile |

Switching is session-bound and logged to the audit trail.

## Economy Integration

OVAV tracks model costs in real-time:

```bash
ovav status          # Shows current posture including economy data
```

The economy dashboard provides:

- **Cost per model** — spending breakdown by vendor
- **Token usage** — input/output token counts
- **Budget alerts** — warnings when approaching limits
- **Model efficiency** — cost vs. output quality metrics

## Current Stack (v5.0)

| Layer | Technology | Models |
|---|---|---|
| Product (Go) | stdlib-only, static binary | Model-agnostic |
| Frontend (TS) | React 18 + Vite | Model-agnostic |
| Governance (Python) | Validators, harnesses, governor | Routes to configured models |

The Go runtime has **zero third-party dependencies** — all model communication goes through standard HTTP interfaces.

## Adding a New Model

To add a new model provider:

1. Configure the model endpoint in your OVAV config
2. Set the model in your profile's tool configuration
3. The Governor validates the model is authorized for your role
4. Tasks route to the new model automatically

No code changes required — OVAV's model layer is designed for extension.
