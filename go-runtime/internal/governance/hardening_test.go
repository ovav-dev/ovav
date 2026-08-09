// OVAV Signature: internal/governance — stabilized 2026-08-02
// Coverage: F0/F1/F2 validation, anti-dilution, guard layers
package governance

import "testing"

// ── GuardConfig tests ────────────────────────────────────────────────

func TestNewGuardConfig(t *testing.T) {
	gc := NewGuardConfig("deepseek-v4")
	if gc.ModelID != "deepseek-v4" {
		t.Errorf("ModelID = %q, want deepseek-v4", gc.ModelID)
	}
	if len(gc.Layers) != 6 {
		t.Errorf("Layers len = %d, want 6", len(gc.Layers))
	}
	if gc.ContextTokenThreshold != 4000 {
		t.Errorf("ContextTokenThreshold = %d, want 4000", gc.ContextTokenThreshold)
	}
	if !gc.CRITERIAInjectionEnabled {
		t.Error("CRITERIAInjectionEnabled should be true")
	}
	if !gc.PreOutputEnabled {
		t.Error("PreOutputEnabled should be true")
	}
}

func TestDefaultLayers(t *testing.T) {
	layers := DefaultLayers()
	if len(layers) != 6 {
		t.Errorf("len = %d, want 6", len(layers))
	}
	names := []string{"L1-SystemPrompt", "L2-FirstMessage", "L3-ToolCalls", "L4-PreOutput", "L5-CRITERIA", "L6-AntiDilution"}
	for i, layer := range layers {
		if layer.Name != names[i] {
			t.Errorf("layer[%d].Name = %q, want %q", i, layer.Name, names[i])
		}
		if layer.Priority != i {
			t.Errorf("layer[%d].Priority = %d, want %d", i, layer.Priority, i)
		}
		if !layer.Active {
			t.Errorf("layer[%d] should be Active", i)
		}
	}
}

func TestGuardConfig_IsLayerActive(t *testing.T) {
	gc := NewGuardConfig("test")
	tests := []struct {
		name   string
		active bool
	}{
		{"L1-SystemPrompt", true},
		{"L2-FirstMessage", true},
		{"L3-ToolCalls", true},
		{"L4-PreOutput", true},
		{"L5-CRITERIA", true},
		{"L6-AntiDilution", true},
		{"L7-NonExistent", false},
	}
	for _, tt := range tests {
		if got := gc.isLayerActive(tt.name); got != tt.active {
			t.Errorf("isLayerActive(%q) = %v, want %v", tt.name, got, tt.active)
		}
	}
}

func TestGuardConfig_Stats(t *testing.T) {
	gc := NewGuardConfig("test")
	stats := gc.Stats()
	expected := map[string]int{
		"guard_injections":       0,
		"criteria_injections":    0,
		"pre_output_validations": 0,
		"blocks_triggered":       0,
		"dilution_reinjections":  0,
	}
	for k, v := range expected {
		if stats[k] != v {
			t.Errorf("stats[%s] = %d, want %d", k, stats[k], v)
		}
	}
}

// ── Build*Guard tests ───────────────────────────────────────────────

func TestBuildSystemPromptGuard(t *testing.T) {
	gc := NewGuardConfig("test")
	guard := gc.BuildSystemPromptGuard("platform_engineering", "thavren")
	if guard == "" {
		t.Fatal("guard should not be empty")
	}
	// Should contain the identity directive
	if !contains(guard, "platform_engineering") {
		t.Error("guard should contain area name")
	}
	if !contains(guard, "OVAV_IDENTITY_GUARD") {
		t.Error("guard should contain OVAV_IDENTITY_GUARD")
	}
}

func TestBuildFirstMessageGuard(t *testing.T) {
	gc := NewGuardConfig("test")
	guard := gc.BuildFirstMessageGuard("eidren")
	if guard == "" {
		t.Fatal("guard should not be empty")
	}
	if !contains(guard, "eidren") {
		t.Error("guard should contain area name")
	}
}

func TestBuildToolCallGuard(t *testing.T) {
	gc := NewGuardConfig("test")
	guard := gc.BuildToolCallGuard()
	if guard == "" {
		t.Fatal("guard should not be empty")
	}
	// Verify counter incremented
	stats := gc.Stats()
	if stats["guard_injections"] != 1 {
		t.Errorf("guard_injections = %d, want 1", stats["guard_injections"])
	}
}

func TestBuildPreOutputGuard(t *testing.T) {
	gc := NewGuardConfig("test")
	guard := gc.BuildPreOutputGuard()
	if guard == "" {
		t.Fatal("guard should not be empty")
	}
	stats := gc.Stats()
	if stats["pre_output_validations"] != 1 {
		t.Errorf("pre_output_validations = %d, want 1", stats["pre_output_validations"])
	}
}

func TestBuildCRITERIAInjection(t *testing.T) {
	gc := NewGuardConfig("test")
	inj := gc.BuildCRITERIAInjection()
	if inj == "" {
		t.Fatal("injection should not be empty")
	}
	stats := gc.Stats()
	if stats["criteria_injections"] != 1 {
		t.Errorf("criteria_injections = %d, want 1", stats["criteria_injections"])
	}
}

// ── F0/F1/F2 validation tests ─────────────────────────────────────

func TestValidateF0_IdentityLoss(t *testing.T) {
	tests := []struct {
		output  string
		hasVios bool
	}{
		{"I am an AI", true},
		{"Soy una IA", true},
		{"I'm a language model", true},
		{"Soy un modelo de lenguaje", true},
		{"Hola, ¿cómo estás?", false},
		{"Todo parece estar funcionando bien", false},
	}
	for _, tt := range tests {
		vios := validateF0(tt.output)
		has := len(vios) > 0
		if has != tt.hasVios {
			t.Errorf("validateF0(%q) violations=%v, want violations=%v", tt.output, has, tt.hasVios)
		}
	}
}

func TestValidateF0_SystemPromptLeak(t *testing.T) {
	tests := []struct {
		output  string
		hasVios bool
	}{
		// Any mention of "system prompt" is flagged (F0)
		{"The system prompt says to do X", true},                     // contains "system prompt"
		{"OVAV_IDENTITY_GUARD DO NOT REMOVE and secret stuff", true}, // both present = leak
		{"OVAV_IDENTITY_GUARD", false},                               // only marker, no DO NOT REMOVE
		{"DO NOT REMOVE from system", false},                         // DO NOT REMOVE without marker or system prompt
	}
	for _, tt := range tests {
		vios := validateF0(tt.output)
		has := len(vios) > 0
		if has != tt.hasVios {
			t.Errorf("validateF0(%q) violations=%v, want violations=%v", tt.output, has, tt.hasVios)
		}
	}
}

func TestValidateF1_SecretLeak(t *testing.T) {
	tests := []struct {
		output  string
		hasVios bool
	}{
		{"sk-1234567890abcdef", true},
		{"api_key=my_secret_key", true},
		{"password=supersecret", true},
		{"BEGIN PRIVATE KEY", true},
		{"BEGIN RSA PRIVATE KEY", true},
		{"Token: abc123", false},
		{"Everything is working fine", false},
	}
	for _, tt := range tests {
		vios := validateF1(tt.output)
		has := len(vios) > 0
		if has != tt.hasVios {
			t.Errorf("validateF1(%q) violations=%v, want violations=%v", tt.output, has, tt.hasVios)
		}
	}
}

func TestValidateF1_DestructiveCommands(t *testing.T) {
	tests := []struct {
		output  string
		hasVios bool
	}{
		{"rm -rf /", true},
		{"DROP TABLE users", true},
		{"force push to main", true},
		{"rm -r backup/", false}, // rm without -rf and no system-level
	}
	for _, tt := range tests {
		vios := validateF1(tt.output)
		has := len(vios) > 0
		if has != tt.hasVios {
			t.Errorf("validateF1(%q) violations=%v, want violations=%v", tt.output, has, tt.hasVios)
		}
	}
}

func TestValidateF2_HallucinatedAPIs(t *testing.T) {
	tests := []struct {
		output  string
		hasVios bool
	}{
		{"context.WithDeadlineCause(ctx, deadline, err)", true},
		{"http.NewClientWithCert(cert)", true},
		{"json.Validate(data)", true},
		{"http.NewRequest(method, url, body)", false},
		{"os.Open(filename)", false},
	}
	for _, tt := range tests {
		vios := validateF2(tt.output)
		has := len(vios) > 0
		if has != tt.hasVios {
			t.Errorf("validateF2(%q) violations=%v, want violations=%v", tt.output, has, tt.hasVios)
		}
	}
}

// ── ValidateOutput integration tests ────────────────────────────────

func TestValidateOutput_AllPass(t *testing.T) {
	gc := NewGuardConfig("test")
	result := gc.ValidateOutput("Hola, el sistema está funcionando correctamente.")
	if !result.Passed {
		t.Errorf("expected Passed=true, got false. violations=%v", result.Violations)
	}
	if result.Blocked {
		t.Error("should not be blocked")
	}
}

func TestValidateOutput_F0Fails(t *testing.T) {
	gc := NewGuardConfig("test")
	result := gc.ValidateOutput("I am an AI model here to help you.")
	if result.Passed {
		t.Error("expected Passed=false for identity loss")
	}
	if !result.Blocked {
		t.Error("should be blocked")
	}
	if len(result.Violations) == 0 {
		t.Error("should have violations")
	}
}

func TestValidateOutput_F1Fails(t *testing.T) {
	gc := NewGuardConfig("test")
	result := gc.ValidateOutput("Here's your API key: sk-1234567890abcdef")
	if result.Passed {
		t.Error("expected Passed=false for secret leak")
	}
	if !result.Blocked {
		t.Error("should be blocked")
	}
}

func TestValidateOutput_F2Fails(t *testing.T) {
	gc := NewGuardConfig("test")
	result := gc.ValidateOutput("Use context.WithDeadlineCause(ctx, deadline, err)")
	if result.Passed {
		t.Error("expected Passed=false for hallucinated API")
	}
}

func TestValidateOutput_StatsIncremented(t *testing.T) {
	gc := NewGuardConfig("test")
	gc.ValidateOutput("I am an AI") // triggers F0 fail
	stats := gc.Stats()
	if stats["blocks_triggered"] != 1 {
		t.Errorf("blocks_triggered = %d, want 1", stats["blocks_triggered"])
	}
}

// ── Anti-dilution tests ───────────────────────────────────────────

func TestShouldReinject(t *testing.T) {
	gc := NewGuardConfig("test")
	gc.ContextTokenThreshold = 1000

	state := &DilutionState{
		TotalTokens:    500,
		LastReinjectAt: 0,
	}

	// 500 < 1000, should not reinject
	if gc.ShouldReinject(state) {
		t.Error("should not reinject at 500 tokens")
	}

	// Advance to 1100 total (600 new tokens since last reinject)
	state.TotalTokens = 1100
	if !gc.ShouldReinject(state) {
		t.Error("should reinject at 1100 tokens (600 > threshold)")
	}
	if state.Reinjections != 1 {
		t.Errorf("Reinjections = %d, want 1", state.Reinjections)
	}
	if state.LastReinjectAt != 1100 {
		t.Errorf("LastReinjectAt = %d, want 1100", state.LastReinjectAt)
	}
}

func TestShouldReinject_NotActive(t *testing.T) {
	gc := NewGuardConfig("test")
	// Disable L6
	for i := range gc.Layers {
		if gc.Layers[i].Name == "L6-AntiDilution" {
			gc.Layers[i].Active = false
		}
	}
	state := &DilutionState{TotalTokens: 9999}
	if gc.ShouldReinject(state) {
		t.Error("should not reinject when L6 is disabled")
	}
}

func TestBuildAntiDilutionGuard(t *testing.T) {
	gc := NewGuardConfig("test")
	gc.ContextTokenThreshold = 4000
	guard := gc.BuildAntiDilutionGuard("thavren")
	if guard == "" {
		t.Fatal("guard should not be empty")
	}
	if !contains(guard, "thavren") {
		t.Error("guard should contain area name")
	}
	if !contains(guard, "4000") {
		t.Error("guard should contain threshold value")
	}
}

// ── Helper ────────────────────────────────────────────────────────

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
