# Research Brief: WezTerm Nightly 2026 Compatibility Issues

## Research question
What breaking changes, deprecations, and compatibility issues exist in WezTerm nightly builds from 2026?

## Scope
**In scope:** WebGPU front_end availability, enable_csi_u_key_encoding behavior changes, fancy_tab_bar rendering changes, config.color_schemes schema changes, deprecated APIs in recent WezTerm versions, and potential conflicts between enable_tab_bar and use_fancy_tab_bar.

**Out of scope:** General WezTerm configuration tutorials, stable release notes only (unless they illuminate 2026 nightly behavior).

## Assumptions
- User is running/configuring WezTerm in 2026 and needs to know what's changed in nightlies.
- Primary source: https://wezfurlong.org/wezterm/ changelog and docs.
- Date: 2026-08-05.

## Depth mode: standard

## Angles
1. WebGPU front_end availability in WezTerm nightly 2026 builds
2. enable_csi_u_key_encoding behavior in 2026 nightly vs stable
3. fancy_tab_bar rendering changes in recent WezTerm nightlies
4. config.color_schemes schema changes (validation/structure)
5. Deprecated/removed APIs in WezTerm 2025-2026
6. Conflict between config.enable_tab_bar and config.use_fancy_tab_bar
