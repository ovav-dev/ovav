# OVAV — Area / Persona Switching

> Canonical reference for context switches between leads (areas)
> and the persistent identifiers each one carries.

---

## Identity Anchors

Each lead is anchored by **4 persistent identifiers**:

| Anchor | Description | Example |
|--------|-------------|---------|
| **Name** | First-person identity used in commits, agent prompts, signing identity | `Thavren`, `Dante` |
| **Color** | Visual identity in TUI/UI | `#2563eb` (Thavren) · `#ea580c` (Dante) · `#7c3aed` (Eidren) · `#db2777` (Elena) · … |
| **SSH signing key** | Cryptographic fingerprint per area | `SHA256:UDKEEUCSvDX71yk9aO2S/6dm6K58brq3E99kyRejaF0` (Thavren) |
| **Email** | Author/committer email | `<lead>@ovav.worktree` |
| **OVAV_IDENTITY_GUARD** | Identity-binding directive at top of agent prompt | `<!-- OVAV_IDENTITY_GUARD v1.1 — DO NOT REMOVE -->` |

These 4 are non-negotiable. **Switching persona = all 4 shift simultaneously.**

---

## How to switch (canonical)

```bash
# 1. Switch signing key (cryptographic anchor)
ovav worktree/scripts/ovav-area-signing/ovav-area-signing.sh switch thavren

# 2. Verify all 4 anchors are aligned
ovav-area-signing.sh status
# Expected output:
#   active lead     = thavren
#   active area     = platform_engineering
#   user.name       = Thavren
#   user.email      = thavren@ovav.worktree
#   user.signingkey = /home/braka/.ssh/ovav_signing/ovav_thavren

# 3. (Agent prompt context) — reload lead agent prompt
# When switching contexts in OVAV AGENTS, load the agent file:
#   ~/.config/opencode/agents/lead-dante.md (or current)
# The prompt carries: identity, criteria, personality, knowledge domain.
```

---

## The 10 leads (canonical)

| Lead | Area | Color | Key | When to switch to |
|------|------|-------|-----|-------------------|
| `thavren` | platform_engineering | `#2563eb` blue | ovav_thavren | **System**: OVAV, Windows 11, WSL2, Git, signing infra, runtime, validators |
| `dante` | digital_product | `#ea580c` orange | ovav_dante | **Web app**: frontend React/Vue, full-stack, APIs, dbs, productos digitales |
| `eidren` | research_intelligence | `#7c3aed` purple | ovav_eidren | Investigation, benchmark, evidence collection, academic work |
| `elena` | ux_design | `#db2777` pink | ovav_elena | UX/UI design, accessibility, prototypes, design system |
| `camila` | legal_compliance | `#9333ea` purple | ovav_camila | Compliance, contracts, GDPR, regulatory, legal docs |
| `kenji` | adversarial_intelligence | `#dc2626` red | ovav_kenji | Security audits, red team, penetration, threat modeling |
| `renata` | health_performance | `#10b981` green | ovav_renata | Health/wellness, performance, metrics, nutrition, fitness |
| `sofia` | commercial_growth | `#f59e0b` amber | ovav_sofia | Sales, pricing, growth, customer, revenue, business |
| `uriel` | devops_infrastructure | `#0891b2` cyan | ovav_uriel | CI/CD, Docker, cloud, SRE, infrastructure |
| `valeria` | education_career | `#ec4899` pink | ovav_valeria | Curriculum, tutoring, education, career paths |

---

## Hard rules

1. **Persona = commit author = SSH key = email.** Never mix. A commit authored
   by "Thavren" but signed by dante's key is rejected by `verify-commit`.
2. **No impersonation.** Switching persona is allowed; pretending to be a
   different lead without switching anchors is a governance violation.
3. **Each area has its own criteria + brains + personality.** Even when a
   commit is co-authored by squad members, the **lead's criteria govern**.
4. **Default persona = current lead's primary focus.**
   - OVAV system work → Thavren (platform_engineering)
   - Web product work → Dante (digital_product)
   - UX work → Elena (ux_design)
   - etc.
5. **Switching back is mandatory.** When the task changes, switch back to
   the original persona's anchors before continuing.

---

## Validation

```bash
# After switching, ALWAYS run:
ovav-area-signing.sh status
git log --show-signature -1
# Confirm: Author == Lead · Email matches · Signature is from that key
```

A signature mismatch means the **commit was not authored by the active persona**.
Reset the worktree's persona and amend before pushing.

---

## Co-authored commits

Multi-area commits (rare) use `Co-authored-by:` trailers:

```
feat(observability): add metrics dashboard

Co-authored-by: Thavren <thavren@ovav.worktree>   # platform lead (signs)
Co-authored-by: Uriel <uriel@ovav.worktree>        # infra co-author
Area: platform_engineering                          # primary area
Signed-off-by: Thavren <thavren@ovav.worktree>     # signature lead
```

The **first Co-authored-by** is the signing lead. Subsequent are contributing
squad members. `Signed-off-by:` is the canonical DCO trailer.

---

## Files implementing this protocol

| File | Role |
|------|------|
| `workstation/scripts/ovav-area-signing/ovav-area-signing.sh` | Active persona + signing key |
| `go-runtime/internal/runtimes/opencode/agents/lead-<lead>.md` | Persona prompt (criteria, personality) |
| `.ovav/allowed_signers_full` | Trust store mapping email → key |
| `~/.ssh/ovav_signing/ovav_<lead>` | Per-lead SSH ed25519 private keys |
| `.ovav/source/docs/area-context-switching.md` | This document |

---

*Canonical reference — version 2026.08.18.4*
*Maintained by Platform Engineering — Thavren*
