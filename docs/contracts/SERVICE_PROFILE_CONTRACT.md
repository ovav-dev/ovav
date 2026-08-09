# Service Profile Contract

**Version:** 1.0.0 | **Owner:** Thavren | **Area:** Platform Engineering & DX | **Review:** 30 days

## Purpose & Scope

A Service Profile is a customer-visible professional area in OVAV. It bundles one lead operator,
their squad, service lanes, skills, harnesses, permissions, and memory scope into a single
discoverable and governable unit. This contract defines the canonical structure.

Applies to all profiles registered in `.ovav/registry/service_profiles.yaml`.

## Profile Schema

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique profile identifier (e.g., `platform-engineering`). |
| `display_name` | string | Customer-facing name (e.g., "Platform Engineering & DX"). |
| `professional_service_area` | string | Internal area ID (e.g., `platform_engineering`). |
| `visible_service_category` | string | Grouping for UI (e.g., "Infrastructure & DevOps"). |
| `lead_operator` | string | Name of the lead agent (e.g., `Thavren`). |
| `purpose` | string (≤500 chars) | What this profile does for the customer. |
| `lanes` | list of strings | Service lanes (e.g., `[security, infrastructure, cli, runtime]`). |
| `squad` | list of objects | Team members: `{name, role, location}`. |
| `skills` | list of strings | Skill IDs available to this profile. |
| `harnesses` | list of strings | Harness IDs assigned to this profile. |
| `permissions` | object | Permissions scoped to this profile (from `permission_authority.json`). |
| `memory_scope` | string | Memory partition (e.g., `platform_engineering`). |
| `customer_visible` | boolean | Whether this profile appears on the public landing page. |
| `p0` | boolean | Priority 0: critical path profile that cannot be disabled. |

### Forbidden
- Two profiles with the same `lead_operator` (one lead per area).
- A profile with lanes that cross into another area's scope.
- A `p0` profile with `customer_visible: false`.

## Enforcement Mechanism

| Validator | File | Trigger |
|-----------|------|---------|
| Service Profile Validator | `tools/validators/validate_service_profiles.py` | Every commit |
| Lead Scope Validator | `tools/validators/check_lead_scope.py` | Every commit |
| Registry Drift | `tools/validators/check_registry_drift.py` | Every 6 hours |

Validators check that every profile has all 14 required fields, that lead-to-profile mapping
is 1:1, and that no lane overlaps exist across areas.

## Breach Consequences

| Severity | Trigger | Consequence |
|----------|---------|-------------|
| **LOW** | Optional metadata field missing | Warning. Profile still served. |
| **MEDIUM** | Required field missing or malformed | Profile removed from registry until fixed. Not customer-visible. |
| **HIGH** | Lead assigned to multiple profiles | All affected profiles suspended. Lead must choose one area. |
| **CRITICAL** | P0 profile disabled | System-wide hard stop. P0 profiles are non-negotiable. |

## Review Cycle

Every 30 days:
1. Verify lead-to-area mapping is still 1:1.
2. Audit lane definitions for overlap across areas.
3. Review `customer_visible` flags against actual landing page content.
4. Update squad rosters for any personnel changes.
