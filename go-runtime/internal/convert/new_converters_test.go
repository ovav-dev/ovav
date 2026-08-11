package convert

import (
	"testing"
)

func TestNewConverters_Registered(t *testing.T) {
	newTargets := []Target{
		TargetWindsurf,
		TargetCopilot,
		TargetContinue,
		TargetAider,
		TargetGoose,
	}
	
	for _, target := range newTargets {
		t.Run(string(target), func(t *testing.T) {
			conv, err := GetConverter(target)
			if err != nil {
				t.Fatalf("GetConverter(%s) failed: %v", target, err)
			}
			if conv == nil {
				t.Fatal("converter is nil")
			}
			if conv.OutputDir() == "" {
				t.Error("OutputDir is empty")
			}
			if conv.FileExtension() == "" {
				t.Error("FileExtension is empty")
			}
		})
	}
}

func TestNewConverters_AreasOnly(t *testing.T) {
	// All new converters should use AreasOnly mode
	targets := []Target{
		TargetWindsurf, TargetCopilot, TargetContinue,
		TargetAider, TargetGoose,
	}
	
	for _, target := range targets {
		conv, err := GetConverter(target)
		if err != nil {
			t.Fatalf("GetConverter(%s) failed: %v", target, err)
		}
		if !conv.AreasOnly() {
			t.Errorf("%s should use AreasOnly mode", target)
		}
	}
}

func TestNewConverters_OutputDirs(t *testing.T) {
	expected := map[Target]string{
		TargetWindsurf:  "runtimes/windsurf/agents",
		TargetCopilot:   ".github/agents",
		TargetContinue:  ".continue/agents",
		TargetAider:     "runtimes/aider/agents",
		TargetGoose:     "runtimes/goose",
		TargetClaude:    "go-runtime/internal/runtimes/claude-code/agents",
		TargetCursor:    "runtimes/cursor",
		TargetOpenCode:  "go-runtime/internal/runtimes/opencode/agents",
	}
	
	for target, expectedDir := range expected {
		conv, err := GetConverter(target)
		if err != nil {
			t.Fatalf("GetConverter(%s) failed: %v", target, err)
		}
		if conv.OutputDir() != expectedDir {
			t.Errorf("%s OutputDir: got %q, want %q", target, conv.OutputDir(), expectedDir)
		}
	}
}
