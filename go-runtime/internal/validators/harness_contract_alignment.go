package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// HarnessContractAlignment validates that the single authority source (.ovav/plan/caps.yaml) exists.
// Replaces: check_harness_contract_alignment.py
type HarnessContractAlignment struct{}

func NewHarnessContractAlignment() *HarnessContractAlignment { return &HarnessContractAlignment{} }

func (h *HarnessContractAlignment) ID() string   { return "harness_contract_alignment" }
func (h *HarnessContractAlignment) Name() string { return "Harness Contract Alignment" }
func (h *HarnessContractAlignment) Description() string {
	return "Validates single authority source (.ovav/plan/caps.yaml) exists"
}
func (h *HarnessContractAlignment) Weight() int { return 10 }

func (h *HarnessContractAlignment) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	capsPath := filepath.Join(root, ".ovav", "plan", "caps.yaml")
	if _, err := os.Stat(capsPath); os.IsNotExist(err) {
		return Result{
			ID: h.ID(), Name: h.Name(), Status: "fail", Weight: h.Weight(),
			Message:  "FAIL — missing .ovav/plan/caps.yaml (single authority source)",
			Issues:   []string{fmt.Sprintf("MISSING: %s", capsPath)},
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: h.ID(), Name: h.Name(), Status: "pass", Weight: h.Weight(),
		Message:  "PASS — .ovav/plan/caps.yaml exists (single authority source)",
		Duration: time.Since(start),
	}
}

var _ Validator = (*HarnessContractAlignment)(nil)
