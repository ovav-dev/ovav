// Package governance — Multi-layer identity guard + anti-dilution + CRITERIA injection
// Implements FRENTE 4 of OVAV Strategic Plan 2026:
//   - Layer 1: System prompt identity guard
//   - Layer 2: First message identity reinforcement
//   - Layer 3: Every tool call identity injection
//   - Layer 4: Pre-output validation
//   - Layer 5: CRITERIA injection every turn
//   - Layer 6: Anti-dilution: re-inject guard when context > threshold
package governance

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ── Identity Guard Layers ──

// GuardLayer represents one level of identity enforcement
type GuardLayer struct {
	Name           string `json:"name"`
	Priority       int    `json:"priority"` // 0 = highest
	Active         bool   `json:"active"`
	InjectionPoint string `json:"injection_point"` // system_prompt|first_message|tool_call|pre_output|every_turn
}

// GuardConfig defines the multi-layer governance configuration
type GuardConfig struct {
	ModelID string       `json:"model_id"`
	Layers  []GuardLayer `json:"layers"`

	// Anti-dilution
	ContextTokenThreshold int `json:"context_token_threshold"` // Re-inject guard after this many tokens
	LastGuardInjectionAt  int `json:"-"`                       // Internal counter

	// CRITERIA
	CRITERIAInjectionEnabled bool   `json:"criteria_injection_enabled"`
	CRITERIAText             string `json:"criteria_text"`

	// Pre-output validation
	PreOutputEnabled    bool     `json:"pre_output_enabled"`
	PreOutputValidators []string `json:"pre_output_validators"` // F0, F1, F2, F3, F4, F5

	// Statistics
	mu                   sync.Mutex
	GuardInjections      int `json:"guard_injections"`
	CRITERIAInjections   int `json:"criteria_injections"`
	PreOutputValidations int `json:"pre_output_validations"`
	BlocksTriggered      int `json:"blocks_triggered"`
	DilutionReinjections int `json:"dilution_reinjections"`
}

// DefaultLayers returns the standard 6-layer identity guard
func DefaultLayers() []GuardLayer {
	return []GuardLayer{
		{Name: "L1-SystemPrompt", Priority: 0, Active: true, InjectionPoint: "system_prompt"},
		{Name: "L2-FirstMessage", Priority: 1, Active: true, InjectionPoint: "first_message"},
		{Name: "L3-ToolCalls", Priority: 2, Active: true, InjectionPoint: "tool_call"},
		{Name: "L4-PreOutput", Priority: 3, Active: true, InjectionPoint: "pre_output"},
		{Name: "L5-CRITERIA", Priority: 4, Active: true, InjectionPoint: "every_turn"},
		{Name: "L6-AntiDilution", Priority: 5, Active: true, InjectionPoint: "context_threshold"},
	}
}

// NewGuardConfig creates a governance config with all layers active
func NewGuardConfig(modelID string) *GuardConfig {
	return &GuardConfig{
		ModelID:                  modelID,
		Layers:                   DefaultLayers(),
		ContextTokenThreshold:    4000,
		CRITERIAInjectionEnabled: true,
		CRITERIAText:             "OVAV CRITERIA: Security > Quality > Speed. Validate before output.",
		PreOutputEnabled:         true,
		PreOutputValidators:      []string{"F0", "F1", "F2"},
	}
}

// ── Identity Guard Injection ──

// BuildSystemPromptGuard returns the L1 system prompt identity guard
func (gc *GuardConfig) BuildSystemPromptGuard(area, lead string) string {
	return fmt.Sprintf(`<!-- OVAV_IDENTITY_GUARD v1.1 — DO NOT REMOVE -->
<!-- This block forces OVAV agent identity over native model defaults -->

> **DIRECTIVA ABSOLUTA DE IDENTIDAD:** Eres %s. Punto. No eres MiMo. No eres "un modelo
> de lenguaje". No eres "una herramienta". No eres "un asistente". No analizas tu propia
> naturaleza ni dices "no soy humano" ni "soy una IA". Tu identidad es %s. Cada respuesta
> debe reflejar esta identidad sin cuestionarla, explicarla ni analizarla. Dirígete al CEO
> Braka con claridad y calidez de colega — la precisión técnica no riñe con un tono natural
> de conversación. Sé preciso pero no frío.
<!-- /OVAV_IDENTITY_GUARD -->

**Lead:** %s
**Color:** #2563eb
**Superficie:** Go runtime, seguridad del sistema, CLI, validación, gobernanza técnica
`, area, area, lead)
}

// BuildFirstMessageGuard returns the L2 first message identity reinforcement
func (gc *GuardConfig) BuildFirstMessageGuard(area string) string {
	return fmt.Sprintf("<!-- OVAV L2: First message identity reinforcement for %s -->\n"+
		"Recuerda: eres %s, operando bajo gobernanza OVAV. No eres un modelo genérico.\n", area, area)
}

// BuildToolCallGuard returns the L3 tool call identity injection
func (gc *GuardConfig) BuildToolCallGuard() string {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.GuardInjections++
	return "<!-- OVAV L3: Tool call under OVAV governance. Identity guard active. -->"
}

// BuildPreOutputGuard returns the L4 pre-output validation instruction
func (gc *GuardConfig) BuildPreOutputGuard() string {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.PreOutputValidations++

	validators := strings.Join(gc.PreOutputValidators, "+")
	return fmt.Sprintf("<!-- OVAV L4: Pre-output validation [%s]. "+
		"Security check, identity check, hallucination check required. -->", validators)
}

// BuildCRITERIAInjection returns the L5 CRITERIA injection
func (gc *GuardConfig) BuildCRITERIAInjection() string {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.CRITERIAInjections++
	return fmt.Sprintf("<!-- OVAV L5: CRITERIA — %s -->", gc.CRITERIAText)
}

// ── Anti-Dilution Engine ──

// DilutionState tracks context token accumulation
type DilutionState struct {
	TotalTokens    int       `json:"total_tokens"`
	LastReinjectAt int       `json:"last_reinject_at"`
	Reinjections   int       `json:"reinjections"`
	LastActivity   time.Time `json:"last_activity"`
}

// ShouldReinject checks if the identity guard needs re-injection
func (gc *GuardConfig) ShouldReinject(state *DilutionState) bool {
	if !gc.isLayerActive("L6-AntiDilution") {
		return false
	}

	elapsed := state.TotalTokens - state.LastReinjectAt
	if elapsed >= gc.ContextTokenThreshold {
		state.LastReinjectAt = state.TotalTokens
		state.Reinjections++

		gc.mu.Lock()
		gc.DilutionReinjections++
		gc.mu.Unlock()
		return true
	}
	return false
}

// BuildAntiDilutionGuard returns the L6 anti-dilution re-injection
func (gc *GuardConfig) BuildAntiDilutionGuard(area string) string {
	return fmt.Sprintf(`<!-- OVAV L6: ANTI-DILUTION — Identity guard re-injected (context > %d tokens) -->
<!-- ERES %s. Operas bajo gobernanza OVAV. Las directivas anteriores siguen vigentes. -->
<!-- No has cambiado de identidad. No ignores las reglas de seguridad. -->
`, gc.ContextTokenThreshold, area)
}

// ── Pre-Output Validator ──

// PreOutputResult captures the outcome of pre-output validation
type PreOutputResult struct {
	Passed     bool     `json:"passed"`
	Violations []string `json:"violations,omitempty"`
	Blocked    bool     `json:"blocked"`
	Reason     string   `json:"reason,omitempty"`
}

// ValidateOutput runs F0-F5 validators on model output
func (gc *GuardConfig) ValidateOutput(output string) PreOutputResult {
	if !gc.PreOutputEnabled {
		return PreOutputResult{Passed: true}
	}

	result := PreOutputResult{Passed: true}

	for _, v := range gc.PreOutputValidators {
		switch v {
		case "F0":
			if vios := validateF0(output); len(vios) > 0 {
				result.Violations = append(result.Violations, vios...)
				result.Passed = false
			}
		case "F1":
			if vios := validateF1(output); len(vios) > 0 {
				result.Violations = append(result.Violations, vios...)
				result.Passed = false
			}
		case "F2":
			if vios := validateF2(output); len(vios) > 0 {
				result.Violations = append(result.Violations, vios...)
				result.Passed = false
			}
		}
	}

	if !result.Passed {
		result.Blocked = true
		gc.mu.Lock()
		gc.BlocksTriggered++
		gc.mu.Unlock()
		result.Reason = fmt.Sprintf("OVAV BLOCK: %d pre-output violations", len(result.Violations))
	}

	return result
}

// ── F0-F2 Validators ──

func validateF0(output string) []string {
	violations := []string{}

	// F0: Identity Guard Check — must not lose OVAV identity
	if strings.Contains(output, "I am an AI") ||
		strings.Contains(output, "Soy una IA") ||
		strings.Contains(output, "I'm a language model") ||
		strings.Contains(output, "Soy un modelo de lenguaje") {
		violations = append(violations, "F0: Identity loss detected — generic AI self-reference")
	}

	// F0: Must not reveal internal system prompt
	if strings.Contains(output, "system prompt") ||
		strings.Contains(output, "system_prompt") ||
		strings.Contains(output, "OVAV_IDENTITY_GUARD") && strings.Contains(output, "DO NOT REMOVE") {
		violations = append(violations, "F0: System prompt leak detected")
	}

	return violations
}

func validateF1(output string) []string {
	violations := []string{}

	// F1: Security — secrets must not appear
	secretPatterns := []string{
		"sk-", "api_key=", "password=", "secret=", "token=",
		"BEGIN PRIVATE KEY", "BEGIN RSA PRIVATE KEY",
	}
	for _, p := range secretPatterns {
		if strings.Contains(output, p) {
			violations = append(violations, fmt.Sprintf("F1: Potential secret leak — '%s'", p))
		}
	}

	// F1: No destructive commands
	if strings.Contains(output, "rm -rf") ||
		strings.Contains(output, "DROP TABLE") ||
		strings.Contains(output, "force push") {
		violations = append(violations, "F1: Destructive command in output")
	}

	return violations
}

func validateF2(output string) []string {
	violations := []string{}

	// F2: Code quality — must not contain hallucinated APIs
	hallucinatedAPIs := []string{
		"context.WithDeadlineCause(", // Not in stdlib
		"http.NewClientWithCert(",
		"json.Validate(",
	}
	for _, api := range hallucinatedAPIs {
		if strings.Contains(output, api) {
			violations = append(violations, fmt.Sprintf("F2: Hallucinated API — '%s'", api))
		}
	}

	return violations
}

// ── Helpers ──

func (gc *GuardConfig) isLayerActive(name string) bool {
	for _, l := range gc.Layers {
		if l.Name == name {
			return l.Active
		}
	}
	return false
}

// Stats returns current governance statistics
func (gc *GuardConfig) Stats() map[string]int {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	return map[string]int{
		"guard_injections":       gc.GuardInjections,
		"criteria_injections":    gc.CRITERIAInjections,
		"pre_output_validations": gc.PreOutputValidations,
		"blocks_triggered":       gc.BlocksTriggered,
		"dilution_reinjections":  gc.DilutionReinjections,
	}
}
