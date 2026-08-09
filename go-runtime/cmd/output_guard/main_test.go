package main

import (
	"testing"
)

func TestRailNoMixedScripts_PureLatin(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{"pure english", "Hello, this is a normal Spanish response.", "PASS"},
		{"pure spanish", "Hola, ¿cómo estás? Esto es una respuesta normal.", "PASS"},
		{"numbers and punctuation", "2024 es el año 3.14159 — that's it!", "PASS"},
		{"empty", "", "PASS"},
		{"whitespace only", "   \n\t  ", "PASS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := railNoMixedScripts(tt.text)
			if got.Status != tt.want {
				t.Errorf("railNoMixedScripts() = %v, want %v", got.Status, tt.want)
			}
		})
	}
}

func TestRailNoMixedScripts_Cyrillic(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		// Any Cyrillic character → DETECTED (threshold=1, >= operator)
		{"cyrillic word 4 chars", "Привет world", "DETECTED"},
		{"cyrillic word 5 chars", "остальное in Spanish", "DETECTED"},
		{"cyrillic mixed", "Hello остальное world", "DETECTED"},
		{"cyrillic 3 chars", "abcдеf", "DETECTED"}, // 1 char ≥ 1
		{"cyrillic 1 char", "abcdeф", "DETECTED"},  // 1 char ≥ 1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := railNoMixedScripts(tt.text)
			if got.Status != tt.want {
				t.Errorf("railNoMixedScripts(%q) = %v, want %v", tt.text, got.Status, tt.want)
			}
		})
	}
}

func TestRailNoMixedScripts_CJK(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		// Any CJK character → DETECTED (threshold=1, >= operator)
		{"cjk 4 chars", "Hello 世界world世界", "DETECTED"}, // 4 CJK ≥ 1
		{"cjk 5 chars", "你好世界", "DETECTED"},            // 5 CJK ≥ 1
		{"cjk 1 char", "Hello 中", "DETECTED"},          // 1 CJK ≥ 1
		{"cjk 2 chars", "中日", "DETECTED"},              // 2 CJK ≥ 1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := railNoMixedScripts(tt.text)
			if got.Status != tt.want {
				t.Errorf("railNoMixedScripts(%q) = %v, want %v", tt.text, got.Status, tt.want)
			}
		})
	}
}

func TestRailNoMixedScripts_Arabic(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		// Any Arabic character → DETECTED (threshold=1, >= operator)
		{"arabic 4 chars", "Hello مرحبا world", "DETECTED"},
		{"arabic 5 chars", "العربية", "DETECTED"},
		{"arabic 1 char", "abcدdef", "DETECTED"}, // 1 char ≥ 1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := railNoMixedScripts(tt.text)
			if got.Status != tt.want {
				t.Errorf("railNoMixedScripts(%q) = %v, want %v", tt.text, got.Status, tt.want)
			}
		})
	}
}

func TestRailNoMixedScripts_Greek(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		// Any Greek character → DETECTED (threshold=1, >= operator)
		{"greek 4 chars", "Hello κόσμος world", "DETECTED"},
		{"greek 5 chars", "καλημέρα", "DETECTED"},
		{"greek 1 char", "αbcdef", "DETECTED"}, // 1 char ≥ 1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := railNoMixedScripts(tt.text)
			if got.Status != tt.want {
				t.Errorf("railNoMixedScripts(%q) = %v, want %v", tt.text, got.Status, tt.want)
			}
		})
	}
}

func TestRailNoMixedScripts_Hebrew(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		// Any Hebrew character → DETECTED (threshold=1, >= operator)
		{"hebrew 4 chars", "Hello שלום world", "DETECTED"},
		{"hebrew 5 chars", "שלום עולם", "DETECTED"},
		{"hebrew 1 char", "abcשdef", "DETECTED"}, // 1 char ≥ 1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := railNoMixedScripts(tt.text)
			if got.Status != tt.want {
				t.Errorf("railNoMixedScripts(%q) = %v, want %v", tt.text, got.Status, tt.want)
			}
		})
	}
}

func TestRailNoMixedScripts_Devanagari(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		// Any Devanagari character → DETECTED (threshold=1, >= operator)
		{"devanagari 4 chars", "Hello नमस्ते world", "DETECTED"},
		{"devanagari 5 chars", "नमस्ते", "DETECTED"},
		{"devanagari 1 char", "abcहdef", "DETECTED"}, // 1 char ≥ 1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := railNoMixedScripts(tt.text)
			if got.Status != tt.want {
				t.Errorf("railNoMixedScripts(%q) = %v, want %v", tt.text, got.Status, tt.want)
			}
		})
	}
}

func TestRailNoMixedScripts_Thai(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		// Any Thai character → DETECTED (threshold=1, >= operator)
		{"thai 4 chars", "Hello สวัสดี world", "DETECTED"},
		{"thai 5 chars", "สวัสดี", "DETECTED"},
		{"thai 1 char", "abcสdef", "DETECTED"}, // 1 char ≥ 1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := railNoMixedScripts(tt.text)
			if got.Status != tt.want {
				t.Errorf("railNoMixedScripts(%q) = %v, want %v", tt.text, got.Status, tt.want)
			}
		})
	}
}

func TestRailNoMixedScripts_Korean(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		// Any Korean character → DETECTED (threshold=1, >= operator)
		{"korean 4 chars", "Hello 안녕하세요 world", "DETECTED"},
		{"korean 5 chars", "안녕하세요", "DETECTED"},
		{"korean 1 char", "abc안def", "DETECTED"}, // 1 char ≥ 1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := railNoMixedScripts(tt.text)
			if got.Status != tt.want {
				t.Errorf("railNoMixedScripts(%q) = %v, want %v", tt.text, got.Status, tt.want)
			}
		})
	}
}

func TestRailNoMixedScripts_MultipleForeignScripts(t *testing.T) {
	// Any foreign script at threshold=1: any single foreign char → DETECTED
	// Greek: 4 chars → DETECTED (4 ≥ 1)
	text := "αβγδxyz" // 4 Greek chars → DETECTED
	got := railNoMixedScripts(text)
	if got.Status != "DETECTED" {
		t.Errorf("railNoMixedScripts(%q) = %v, want DETECTED (threshold=1)", text, got.Status)
	}

	// Mixed scripts: 2 Cyrillic + 2 Greek = 4 total → DETECTED (≥ 1)
	text2 := "абγδxyz" // 2 Cyrillic + 2 Greek = 4 foreign → DETECTED
	got2 := railNoMixedScripts(text2)
	if got2.Status != "DETECTED" {
		t.Errorf("railNoMixedScripts(%q) = %v, want DETECTED (threshold=1)", text2, got2.Status)
	}
}

func TestIsLatin(t *testing.T) {
	tests := []struct {
		r    rune
		want bool
	}{
		{'a', true},
		{'Z', true},
		{'0', true},
		{' ', true},
		{0x0080, true},  // Latin-1 Supplement boundary
		{0x00FF, true},  // Latin-1 Supplement end
		{0x0100, false}, // Latin Extended-A
		{0x0400, false}, // Cyrillic
		{0x4E00, false}, // CJK
		{0x0590, false}, // Hebrew
	}

	for _, tt := range tests {
		got := isLatin(tt.r)
		if got != tt.want {
			t.Errorf("isLatin(%U) = %v, want %v", tt.r, got, tt.want)
		}
	}
}

func TestScriptNameFor(t *testing.T) {
	tests := []struct {
		r    rune
		want string
	}{
		{'a', ""},
		{0x0410, "Cyrillic"},   // А Cyrillic
		{0x4E2D, "CJK"},        // 中
		{0x0627, "Arabic"},     // ا Arabic
		{0x0905, "Devanagari"}, // अ Devanagari
		{0x0370, "Greek"},      // Greek
		{0x05D0, "Hebrew"},     // א Hebrew
		{0x0E01, "Thai"},       // ก Thai
		{0xAC00, "Korean"},     // Korean
		{0x0530, "Armenian"},   // Armenian
		{0x10A0, "Georgian"},   // Georgian
	}

	for _, tt := range tests {
		got := scriptNameFor(tt.r)
		if got != tt.want {
			t.Errorf("scriptNameFor(%U) = %q, want %q", tt.r, got, tt.want)
		}
	}
}

func TestMixedScriptThreshold_IsConfigurable(t *testing.T) {
	// Save original and restore after test
	orig := mixedScriptThreshold
	defer func() { mixedScriptThreshold = orig }()

	mixedScriptThreshold = 0
	// With threshold=0 and >= operator: any foreign char (count >= 0) → DETECTED
	got := railNoMixedScripts("abcде")
	if got.Status != "DETECTED" {
		t.Errorf("with threshold=0, railNoMixedScripts(%q) = %v, want DETECTED", "abcде", got.Status)
	}
}

func TestMixedScriptThreshold_HighValue(t *testing.T) {
	orig := mixedScriptThreshold
	defer func() { mixedScriptThreshold = orig }()

	mixedScriptThreshold = 100
	// With threshold=100, 10 foreign chars (10 < 100) → PASS
	got := railNoMixedScripts("Hello мир world")
	if got.Status != "PASS" {
		t.Errorf("with threshold=100, railNoMixedScripts(%q) = %v, want PASS", "Hello мир world", got.Status)
	}
}
