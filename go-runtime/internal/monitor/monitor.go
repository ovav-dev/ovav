// Package monitor implements OVAV's 2026 Monitoring & Auto-Remediation System (OMARS).
// Replaces the legacy validator system with intelligent monitors that alert,
// don't block, and auto-fix when possible.
//
// Architecture:
//
//	Monitor → Alert → Dispatcher → Queue → [AutoFix | Human | Archive]
package monitor
