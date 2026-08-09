package validators

import (
	"context"
	"time"
)

// LedgerWritePath — permanently deprecated no-op validator.
// active_context_ledger.yaml was permanently deprecated 2026-06-15 per AGENTS.md.
// Operational memory derives from git HEAD + caps.yaml exclusively.
// This validator always passes — it exists only to prevent breakage in
// downstream systems that may reference its ID.
type LedgerWritePath struct{}

func NewLedgerWritePath() *LedgerWritePath { return &LedgerWritePath{} }

func (l *LedgerWritePath) ID() string   { return "ledger_write_path" }
func (l *LedgerWritePath) Name() string { return "Ledger Write Path (deprecated)" }
func (l *LedgerWritePath) Description() string {
	return "Permanently deprecated. active_context_ledger.yaml must not exist."
}
func (l *LedgerWritePath) Weight() int { return 0 }

func (l *LedgerWritePath) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	msg := "PASS (deprecated) — active_context_ledger.yaml permanently deprecated. Operational memory: git HEAD + caps.yaml."
	return Result{
		ID: l.ID(), Name: l.Name(), Status: "pass", Weight: l.Weight(),
		Message:  msg,
		Duration: time.Since(start),
	}
}

var _ Validator = (*LedgerWritePath)(nil)
