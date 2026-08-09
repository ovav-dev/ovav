---
# ISSUE-2026-07-29: Output Guard No Mixed-Script Detection

## Date: 2026-07-29
## Severity: MEDIUM (security/quality)
## Status: OPEN

## Problem
Output Guard (`cmd/output_guard/main.go`) has NO character encoding validation. Cyrillic characters ("остальное") appeared in a Spanish-language response ("🔴 остальное PENDIENTE") without being blocked. The guard only checks: not empty, no secrets, no governor override, reasonable length.

## Root Cause
No `railMixedScripts` or Unicode script detection exists in the content rails. The guard uses regex-based detection for pre-defined signals but has no general Unicode script boundary checking.

## Evidence
- Cyrillic word "остальное" (Russian for "remaining") appeared in Spanish output
- Mixed-script output could indicate: prompt injection, model hallucination, or encoding corruption
- Response was <300 words → passed through without detailed review (Output Guard skips reviews for short responses)

## Fix Specification
Add `railNoMixedScripts` to content rails in `cmd/output_guard/main.go`:
- Detect when non-Latin scripts (Cyrillic, CJK, Arabic, Devanagari, etc.) appear in text predominantly in Latin script
- Use Unicode script categories or character range checks
- Threshold: if >20% of non-ASCII characters are from a different script than the dominant one, flag as suspicious
- Allowlist: proper nouns, technical terms, valid Unicode symbols (emoji, arrows →)
- This is a quality AND security gate — mixed-script output can indicate injection attacks

## Prevention
Add mixed-script detection to the OVAV Response Contract (`ovav-response-contract` skill) as a required check before calling Output Guard.

## Owner: Platform Engineering (thavren)
---