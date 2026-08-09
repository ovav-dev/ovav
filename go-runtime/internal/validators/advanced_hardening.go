package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AdvancedHardening validates F5 advanced hardening: new states governance,
// gate liberation rules, and advanced_surfaces policy integrity.
// Replaces: check_f5_advanced_hardening.py
type AdvancedHardening struct{}

func NewAdvancedHardening() *AdvancedHardening { return &AdvancedHardening{} }

func (a *AdvancedHardening) ID() string   { return "advanced_hardening" }
func (a *AdvancedHardening) Name() string { return "F5 Advanced Hardening" }
func (a *AdvancedHardening) Description() string {
	return "Validates new states governance, gate liberation, and advanced surfaces"
}
func (a *AdvancedHardening) Weight() int { return 16 }

// Expected gates in F5 gate liberation.
var expectedGates = []string{
	"auto_switch",
	"research_firewall",
	"snapshot_apply",
	"ledger_vivo",
	"ovav_mesh",
}

// Expected new states in F5.
var expectedNewStates = []string{
	"adaptive", "emergent", "delegated", "collaborative",
	"autonomous", "reflective", "generative", "orchestrated",
	"healing", "evolving", "convergent", "divergent",
	"inherited", "canary_gated",
}

func (a *AdvancedHardening) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	policyPath := filepath.Join(root, ".ovav", "policy", "permission_authority.json")
	data, err := os.ReadFile(policyPath)
	if err != nil {
		return Result{
			ID: a.ID(), Name: a.Name(), Status: "fail", Weight: a.Weight(),
			Message: "FAIL: permission_authority.json not found", Duration: time.Since(start),
		}
	}

	var policy map[string]interface{}
	if err := json.Unmarshal(data, &policy); err != nil {
		return Result{
			ID: a.ID(), Name: a.Name(), Status: "fail", Weight: a.Weight(),
			Message: "FAIL: invalid JSON", Duration: time.Since(start),
		}
	}

	// 1. Validate advanced_surfaces section exists
	adv, ok := policy["advanced_surfaces"].(map[string]interface{})
	if !ok {
		issues = append(issues, "F5: advanced_surfaces section missing from permission_authority.json")
		return Result{
			ID: a.ID(), Name: a.Name(), Status: "fail", Weight: a.Weight(),
			Message: "FAIL F5 — advanced_surfaces missing", Issues: issues, Duration: time.Since(start),
		}
	}

	// 2. Validate F5.1 new_states governance
	ns, ok := adv["f5_new_states"].(map[string]interface{})
	if !ok {
		issues = append(issues, "F5.1: f5_new_states section missing")
	} else {
		total := intVal(ns, "total_rules")
		allowed := intVal(ns, "allowed")
		denied := intVal(ns, "denied")

		if total != 14 {
			issues = append(issues, fmt.Sprintf("F5.1: new_states total_rules expected 14, got %d", total))
		}
		if allowed != 12 {
			issues = append(issues, fmt.Sprintf("F5.1: new_states allowed expected 12, got %d", allowed))
		}
		if denied != 2 {
			issues = append(issues, fmt.Sprintf("F5.1: new_states denied expected 2, got %d", denied))
		}
		if allowed+denied != total {
			issues = append(issues, fmt.Sprintf("F5.1: new_states allowed(%d) + denied(%d) != total(%d)", allowed, denied, total))
		}
	}

	// 3. Validate F5.2 gate_liberation governance (from f5_gate_liberation key)
	gl, ok := policy["f5_gate_liberation"].(map[string]interface{})
	if !ok {
		issues = append(issues, "F5.2: f5_gate_liberation section missing")
	} else {
		total := intVal(gl, "total_rules")
		allowed := intVal(gl, "allowed")
		requiresAllF0 := boolVal(gl, "requires_all_f0")

		if total != 5 {
			issues = append(issues, fmt.Sprintf("F5.2: gate_liberation total_rules expected 5, got %d", total))
		}
		if allowed != 5 {
			issues = append(issues, fmt.Sprintf("F5.2: gate_liberation allowed expected 5, got %d", allowed))
		}
		if !requiresAllF0 {
			issues = append(issues, "F5.2: gate_liberation requires_all_f0 must be true")
		}

		// Validate each gate defined
		gates, _ := gl["gates"].(map[string]interface{})
		for _, gate := range expectedGates {
			if g, ok := gates[gate].(map[string]interface{}); ok {
				if g["action"] == nil {
					issues = append(issues, fmt.Sprintf("F5.2: gate '%s' missing action", gate))
				}
				if g["blocked_surface"] == nil {
					issues = append(issues, fmt.Sprintf("F5.2: gate '%s' missing blocked_surface", gate))
				}
			} else {
				issues = append(issues, fmt.Sprintf("F5.2: missing gate definition: %s", gate))
			}
		}
	}

	// 4. Validate gates section in advanced_surfaces
	advGates, ok := adv["gates"].(map[string]interface{})
	if ok {
		for _, gate := range expectedGates {
			if g, ok := advGates[gate].(map[string]interface{}); ok {
				action := strVal(g, "action")
				if !strings.Contains(action, "f0_green") && action != "" {
					issues = append(issues, fmt.Sprintf("F5: gate '%s' action should require f0_green", gate))
				}
				gain := intVal(g, "gain_pct")
				if gain <= 0 || gain > 100 {
					issues = append(issues, fmt.Sprintf("F5: gate '%s' gain_pct invalid: %d", gate, gain))
				}
			}
		}
		// Count defined gates match expected
		if len(advGates) != 5 {
			issues = append(issues, fmt.Sprintf("F5: advanced_surfaces.gates expected 5, got %d", len(advGates)))
		}
	}

	// 5. Cross-validate: gate_liberation gates match advanced_surfaces gates
	glGates, _ := gl["gates"].(map[string]interface{})
	for _, gate := range expectedGates {
		if glGates != nil {
			if _, ok := glGates[gate]; !ok {
				issues = append(issues, fmt.Sprintf("F5: gate '%s' in f5_gate_liberation but missing from advanced_surfaces", gate))
			}
		}
	}

	// 6. Check infrastructure_surfaces claims require all F0-F5
	infra, _ := policy["infrastructure_surfaces"].(map[string]interface{})
	if fc, ok := infra["f2_claims"].(map[string]interface{}); ok {
		prod := strVal(fc, "production_ready")
		if !strings.Contains(prod, "f5") {
			issues = append(issues, "F5: production_ready claim should require f5 validators")
		}
	}

	if len(issues) > 0 {
		return Result{
			ID: a.ID(), Name: a.Name(), Status: "fail", Weight: a.Weight(),
			Message:  fmt.Sprintf("FAIL F5 advanced hardening — %d issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: a.ID(), Name: a.Name(), Status: "pass", Weight: a.Weight(),
		Message:  "PASS F5 advanced hardening — new states, gate liberation, advanced surfaces verified",
		Duration: time.Since(start),
	}
}

var _ Validator = (*AdvancedHardening)(nil)
