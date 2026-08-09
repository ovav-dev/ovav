package adapters

import "testing"

// ── ModelAdapter tests ─────────────────────────────────────────────

func TestModelAdapter_BuildGovernanceBlock(t *testing.T) {
	adapter := DeepSeekV4Adapter()
	block := adapter.BuildGovernanceBlock()
	if block == "" {
		t.Fatal("block should not be empty")
	}
	if !contains(block, "OVAV_IDENTITY_GUARD") {
		t.Error("block should contain OVAV_IDENTITY_GUARD")
	}
	if !contains(block, "deepseek-v4") {
		t.Error("block should contain model ID")
	}
}

func TestModelAdapter_BuildPreOutputInstruction(t *testing.T) {
	tests := []struct {
		validation  string
		wantContain string
	}{
		{"F0", "OVAV F0"},
		{"F2", "OVAV F2"},
		{"F5", "OVAV F5"},
		{"unknown", "OVAV: Pre-output"},
	}

	for _, tt := range tests {
		adapter := &ModelAdapter{PreOutputValidation: tt.validation}
		block := adapter.BuildPreOutputInstruction()
		if !contains(block, tt.wantContain) {
			t.Errorf("BuildPreOutputInstruction(%q) = %q, want contains %q", tt.validation, block, tt.wantContain)
		}
	}
}

func TestDeepSeekV4Adapter(t *testing.T) {
	a := DeepSeekV4Adapter()
	if a.ModelID != "deepseek-v4" {
		t.Errorf("ModelID = %q, want deepseek-v4", a.ModelID)
	}
	if a.IdentityGuardRepeatTokens != 2000 {
		t.Errorf("IdentityGuardRepeatTokens = %d, want 2000", a.IdentityGuardRepeatTokens)
	}
	if !a.IdentityGuardInToolCalls {
		t.Error("IdentityGuardInToolCalls should be true")
	}
	if a.PreOutputValidation != "F2" {
		t.Errorf("PreOutputValidation = %q, want F2", a.PreOutputValidation)
	}
	if len(a.HardStopPatterns) == 0 {
		t.Error("HardStopPatterns should not be empty")
	}
}

func TestClaudeAdapter(t *testing.T) {
	a := ClaudeAdapter()
	if a.ModelID != "claude" {
		t.Errorf("ModelID = %q, want claude", a.ModelID)
	}
	if !a.PreAuthRequired {
		t.Error("PreAuthRequired should be true for Claude")
	}
	if a.PreOutputValidation != "F5" {
		t.Errorf("PreOutputValidation = %q, want F5", a.PreOutputValidation)
	}
}

func TestGPT5Adapter(t *testing.T) {
	a := GPT5Adapter()
	if a.ModelID != "gpt-5" {
		t.Errorf("ModelID = %q, want gpt-5", a.ModelID)
	}
	if a.IdentityGuardRepeatTokens != 3000 {
		t.Errorf("IdentityGuardRepeatTokens = %d, want 3000", a.IdentityGuardRepeatTokens)
	}
}

func TestQwenAdapter(t *testing.T) {
	a := QwenAdapter()
	if a.ModelID != "qwen" {
		t.Errorf("ModelID = %q, want qwen", a.ModelID)
	}
	if !a.BilingualGuard {
		t.Error("BilingualGuard should be true for Qwen")
	}
	if a.SecondLanguage != "zh-CN" {
		t.Errorf("SecondLanguage = %q, want zh-CN", a.SecondLanguage)
	}
}

func TestKimiK7Adapter(t *testing.T) {
	a := KimiK7Adapter()
	if a.ModelID != "kimi-k7" {
		t.Errorf("ModelID = %q, want kimi-k7", a.ModelID)
	}
	if !a.BilingualGuard {
		t.Error("BilingualGuard should be true for Kimi")
	}
	if !a.PreAuthRequired {
		t.Error("PreAuthRequired should be true for Kimi")
	}
}

func TestRegistry(t *testing.T) {
	r := Registry()
	if len(r) != 5 {
		t.Errorf("Registry len = %d, want 5", len(r))
	}
	expected := []string{"deepseek-v4", "claude", "gpt-5", "qwen", "kimi-k7"}
	for _, id := range expected {
		if _, ok := r[id]; !ok {
			t.Errorf("Registry missing model %q", id)
		}
	}
}

// ── Helper ────────────────────────────────────────────────────────

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
