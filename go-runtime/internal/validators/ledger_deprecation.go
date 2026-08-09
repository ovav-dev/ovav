package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LedgerDeprecation enforces that active_context_ledger.yaml is permanently
// absent. The ledger was deprecated on 2026-06-15 and operational memory is
// derived exclusively from git HEAD + caps.yaml.
// Replaces: check_ledger_deprecation.py
type LedgerDeprecation struct{}

func NewLedgerDeprecation() *LedgerDeprecation { return &LedgerDeprecation{} }

func (l *LedgerDeprecation) ID() string   { return "ledger_deprecation" }
func (l *LedgerDeprecation) Name() string { return "Ledger Deprecation Enforcer" }
func (l *LedgerDeprecation) Description() string {
	return "Fails if active_context_ledger.yaml exists — it is permanently deprecated"
}
func (l *LedgerDeprecation) Weight() int { return 5 }

const ledgerPath = ".ovav/registry/active_context_ledger.yaml"

func (l *LedgerDeprecation) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	target := filepath.Join(root, ledgerPath)
	if _, err := os.Stat(target); err == nil {
		issues = append(issues, fmt.Sprintf(
			"BLOCKED: %s exists — permanently deprecated since 2026-06-15. Delete it. Canonical sources: git HEAD + caps.yaml",
			ledgerPath,
		))
		return Result{
			ID: l.ID(), Name: l.Name(), Status: "fail", Weight: l.Weight(),
			Message:  fmt.Sprintf("FAIL %s — deprecated ledger file detected", ledgerPath),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}

	return Result{
		ID: l.ID(), Name: l.Name(), Status: "pass", Weight: l.Weight(),
		Message:  fmt.Sprintf("PASS %s not present — operational memory from git HEAD + caps.yaml", ledgerPath),
		Duration: time.Since(start),
	}
}

var _ Validator = (*LedgerDeprecation)(nil)
