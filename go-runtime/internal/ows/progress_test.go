package ows

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunNodeJSVerificationSkipsUndefinedChecks(t *testing.T) {
	dir := t.TempDir()
	packageJSON := `{"name":"root-orchestrator","private":true,"scripts":{"dev":"true"}}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSON), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	emptyPath := t.TempDir()
	t.Setenv("PATH", emptyPath)

	results := runNodeJSVerification(dir, 2)
	if len(results) != 2 {
		t.Fatalf("result count = %d, want 2", len(results))
	}
	for _, result := range results {
		if !result.Pass {
			t.Errorf("%s should pass when undefined: %v", result.Name, result.Issues)
		}
		if len(result.Issues) != 1 || !strings.HasPrefix(result.Issues[0], "skipped:") {
			t.Errorf("%s issues = %v, want one skipped reason", result.Name, result.Issues)
		}
	}
}

func TestRunNodeJSVerificationRejectsInvalidManifest(t *testing.T) {
	dir := t.TempDir()
	writeOWSTestFile(t, filepath.Join(dir, "package.json"), "{")

	results := runNodeJSVerification(dir, 2)
	if len(results) != 1 || results[0].Pass || results[0].Name != "Node manifest" {
		t.Fatalf("expected invalid manifest failure, got %#v", results)
	}
}

func TestRunNodeJSVerificationUsesDeclaredPackageManager(t *testing.T) {
	dir := t.TempDir()
	binDir := t.TempDir()
	argsFile := filepath.Join(t.TempDir(), "args")
	writeOWSExecutable(t, filepath.Join(binDir, "pnpm"), "#!/bin/sh\nprintf '%s' \"$*\" > \"$ARGS_FILE\"\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ARGS_FILE", argsFile)
	writeOWSTestFile(t, filepath.Join(dir, "package.json"), `{"packageManager":"pnpm@9.0.0"}`)
	writeOWSTestFile(t, filepath.Join(dir, "tsconfig.base.json"), `{}`)

	results := runNodeJSVerification(dir, 2)
	if len(results) != 2 || !results[0].Pass || results[0].Name != "tsc" {
		t.Fatalf("expected pnpm TypeScript pass and skipped tests, got %#v", results)
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(args); got != "exec tsc --noEmit" {
		t.Fatalf("pnpm args = %q", got)
	}
}

func TestRunNodeJSVerificationConfiguredFailuresBlock(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*testing.T, string)
		executable string
		wantPhase  string
	}{
		{
			name: "biome",
			setup: func(t *testing.T, dir string) {
				writeOWSTestFile(t, filepath.Join(dir, "package.json"), `{}`)
				writeOWSTestFile(t, filepath.Join(dir, "biome.json"), `{}`)
			},
			executable: "npx",
			wantPhase:  "biome check",
		},
		{
			name: "typescript",
			setup: func(t *testing.T, dir string) {
				writeOWSTestFile(t, filepath.Join(dir, "package.json"), `{}`)
				writeOWSTestFile(t, filepath.Join(dir, "tsconfig.json"), `{}`)
			},
			executable: "npx",
			wantPhase:  "tsc",
		},
		{
			name: "test script",
			setup: func(t *testing.T, dir string) {
				writeOWSTestFile(t, filepath.Join(dir, "package.json"), `{"scripts":{"test":"vitest"}}`)
			},
			executable: "npm",
			wantPhase:  "node test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			binDir := t.TempDir()
			writeOWSExecutable(t, filepath.Join(binDir, tt.executable), "#!/bin/sh\necho configured-check-failed >&2\nexit 7\n")
			t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			tt.setup(t, dir)

			results := runNodeJSVerification(dir, 2)
			var matched *PhaseResult
			for i := range results {
				if results[i].Name == tt.wantPhase {
					matched = &results[i]
					break
				}
			}
			if matched == nil || matched.Pass {
				t.Fatalf("expected %s failure, got %#v", tt.wantPhase, results)
			}
			if !strings.Contains(strings.Join(matched.Issues, "\n"), "configured-check-failed") {
				t.Fatalf("missing command output in %#v", matched.Issues)
			}
		})
	}
}

func writeOWSTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeOWSExecutable(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o700); err != nil {
		t.Fatal(err)
	}
}

func TestHygieneWarningsDoNotBlockVerification(t *testing.T) {
	result := hygienePhaseResult(&HygieneResult{
		Clean:         false,
		WarningIssues: 2,
		TotalIssues:   2,
	})
	if !result.Pass {
		t.Fatalf("warnings-only hygiene result blocked verification: %v", result.Issues)
	}
	if len(result.Issues) != 1 || !strings.Contains(result.Issues[0], "2 warning") {
		t.Fatalf("issues = %v, want warning count", result.Issues)
	}
}

func TestHygieneBlockingIssuesFailVerification(t *testing.T) {
	result := hygienePhaseResult(&HygieneResult{
		Clean:          false,
		BlockingIssues: 1,
		WarningIssues:  2,
		TotalIssues:    3,
	})
	if result.Pass {
		t.Fatal("blocking hygiene issue passed verification")
	}
	if len(result.Issues) != 2 {
		t.Fatalf("issues = %v, want blocking and warning summaries", result.Issues)
	}
}
