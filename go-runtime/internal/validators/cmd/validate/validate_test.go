package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ovav/ovav/internal/validators"
)

// mockValidator implements validators.Validator for testing
type mockValidator struct {
	id     string
	name   string
	weight int
}

func (m *mockValidator) ID() string          { return m.id }
func (m *mockValidator) Name() string        { return m.name }
func (m *mockValidator) Description() string { return "mock validator" }
func (m *mockValidator) Weight() int         { return m.weight }
func (m *mockValidator) Validate(ctx context.Context, root string) validators.Result {
	return validators.Result{ID: m.id, Status: "pass"}
}

// configurableMock is a mock validator with configurable status and weight.
type configurableMock struct {
	id     string
	name   string
	weight int
	status string
	issues []string
}

func (m *configurableMock) ID() string          { return m.id }
func (m *configurableMock) Name() string        { return m.name }
func (m *configurableMock) Description() string { return "configurable mock" }
func (m *configurableMock) Weight() int         { return m.weight }
func (m *configurableMock) Validate(ctx context.Context, root string) validators.Result {
	return validators.Result{
		ID: m.id, Name: m.name, Status: m.status,
		Weight: m.weight, Issues: m.issues, Message: m.name + " result",
	}
}

// setupFakeRepo creates a temp directory that passes isOVAVRepo checks.
func setupFakeRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav"), 0755)
	os.MkdirAll(filepath.Join(dir, ".ovav", "plan"), 0755)
	os.WriteFile(filepath.Join(dir, ".ovav", "plan", "caps.yaml"), []byte("canonical: true\n"), 0644)
	os.MkdirAll(filepath.Join(dir, "go-runtime"), 0755)
	os.WriteFile(filepath.Join(dir, "go-runtime", "go.mod"), []byte("module x\n"), 0644)
	return dir
}

func TestIsOVAVRepo(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		expected bool
	}{
		{
			name: "valid repo with both markers",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				os.MkdirAll(filepath.Join(dir, ".ovav"), 0755)
				os.MkdirAll(filepath.Join(dir, ".ovav", "plan"), 0755)
				os.WriteFile(filepath.Join(dir, ".ovav", "plan", "caps.yaml"), []byte("canonical: true\n"), 0644)
				os.MkdirAll(filepath.Join(dir, "go-runtime"), 0755)
				os.WriteFile(filepath.Join(dir, "go-runtime", "go.mod"), []byte("module x\n"), 0644)
				return dir
			},
			expected: true,
		},
		{
			name: "missing .ovav directory",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				os.MkdirAll(filepath.Join(dir, "go-runtime"), 0755)
				os.WriteFile(filepath.Join(dir, "go-runtime", "go.mod"), []byte("module x\n"), 0644)
				return dir
			},
			expected: false,
		},
		{
			name: "missing go-runtime/go.mod",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				os.MkdirAll(filepath.Join(dir, ".ovav"), 0755)
				return dir
			},
			expected: false,
		},
		{
			name: "empty directory",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			expected: false,
		},
		{
			name:     "non-existent path",
			setup:    func(t *testing.T) string { return "/nonexistent/path/xyz" },
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)
			result := isOVAVRepo(dir)
			if result != tt.expected {
				t.Errorf("isOVAVRepo(%q) = %v, expected %v", dir, result, tt.expected)
			}
		})
	}
}

func TestFindRepoRootFrom(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		validate func(t *testing.T, result string)
	}{
		{
			name: "finds root at current level",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				os.MkdirAll(filepath.Join(dir, ".ovav"), 0755)
				os.MkdirAll(filepath.Join(dir, ".ovav", "plan"), 0755)
				os.WriteFile(filepath.Join(dir, ".ovav", "plan", "caps.yaml"), []byte("canonical: true\n"), 0644)
				os.MkdirAll(filepath.Join(dir, "go-runtime"), 0755)
				os.WriteFile(filepath.Join(dir, "go-runtime", "go.mod"), []byte("module x\n"), 0644)
				return dir
			},
			validate: func(t *testing.T, result string) {
				if result == "." {
					t.Error("expected to find root, got '.'")
				}
				if !isOVAVRepo(result) {
					t.Errorf("result %q is not a valid OVAV repo", result)
				}
			},
		},
		{
			name: "finds root from nested directory",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				os.MkdirAll(filepath.Join(dir, ".ovav"), 0755)
				os.MkdirAll(filepath.Join(dir, ".ovav", "plan"), 0755)
				os.WriteFile(filepath.Join(dir, ".ovav", "plan", "caps.yaml"), []byte("canonical: true\n"), 0644)
				os.MkdirAll(filepath.Join(dir, "go-runtime"), 0755)
				os.WriteFile(filepath.Join(dir, "go-runtime", "go.mod"), []byte("module x\n"), 0644)
				nested := filepath.Join(dir, "a", "b", "c")
				os.MkdirAll(nested, 0755)
				return nested
			},
			validate: func(t *testing.T, result string) {
				if result == "." {
					t.Error("expected to find root from nested dir, got '.'")
				}
				if !isOVAVRepo(result) {
					t.Errorf("result %q is not a valid OVAV repo", result)
				}
			},
		},
		{
			name: "returns dot when no root found",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			validate: func(t *testing.T, result string) {
				if result != "." {
					t.Errorf("expected '.', got %q", result)
				}
			},
		},
		{
			name: "stops after 10 levels",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				deep := dir
				for i := 0; i < 15; i++ {
					deep = filepath.Join(deep, "level")
				}
				os.MkdirAll(deep, 0755)
				return deep
			},
			validate: func(t *testing.T, result string) {
				if result != "." {
					t.Errorf("expected '.' after 10+ levels, got %q", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startDir := tt.setup(t)
			result := findRepoRootFrom(startDir)
			tt.validate(t, result)
		})
	}
}

func TestFilterByID(t *testing.T) {
	v1 := &mockValidator{id: "validator-1", name: "First", weight: 10}
	v2 := &mockValidator{id: "validator-2", name: "Second", weight: 20}
	v3 := &mockValidator{id: "validator-3", name: "Third", weight: 15}
	all := []validators.Validator{v1, v2, v3}

	tests := []struct {
		name     string
		id       string
		input    []validators.Validator
		expected int
		wantID   string
	}{
		{
			name:     "empty id returns all",
			id:       "",
			input:    all,
			expected: 3,
		},
		{
			name:     "finds existing validator",
			id:       "validator-2",
			input:    all,
			expected: 1,
			wantID:   "validator-2",
		},
		{
			name:     "returns nil for non-existent id",
			id:       "non-existent",
			input:    all,
			expected: 0,
		},
		{
			name:     "handles empty input",
			id:       "validator-1",
			input:    []validators.Validator{},
			expected: 0,
		},
		{
			name:     "finds first validator",
			id:       "validator-1",
			input:    all,
			expected: 1,
			wantID:   "validator-1",
		},
		{
			name:     "finds last validator",
			id:       "validator-3",
			input:    all,
			expected: 1,
			wantID:   "validator-3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterByID(tt.input, tt.id)

			if len(result) != tt.expected {
				t.Errorf("expected %d validators, got %d", tt.expected, len(result))
			}

			if tt.wantID != "" && len(result) > 0 {
				if result[0].ID() != tt.wantID {
					t.Errorf("expected validator with ID %q, got %q", tt.wantID, result[0].ID())
				}
			}
		})
	}
}

func TestFilterQuick(t *testing.T) {
	v1 := &mockValidator{id: "low-1", weight: 5}
	v2 := &mockValidator{id: "low-2", weight: 10}
	v3 := &mockValidator{id: "high-1", weight: 15}
	v4 := &mockValidator{id: "high-2", weight: 20}
	v5 := &mockValidator{id: "high-3", weight: 25}
	all := []validators.Validator{v1, v2, v3, v4, v5}

	tests := []struct {
		name     string
		input    []validators.Validator
		expected int
	}{
		{
			name:     "filters to high-weight validators",
			input:    all,
			expected: 3, // v3, v4, v5
		},
		{
			name:     "handles empty input",
			input:    []validators.Validator{},
			expected: 0,
		},
		{
			name:     "filters all when none meet threshold",
			input:    []validators.Validator{v1, v2},
			expected: 0,
		},
		{
			name:     "keeps all when all meet threshold",
			input:    []validators.Validator{v3, v4, v5},
			expected: 3,
		},
		{
			name:     "boundary case: exactly 15",
			input:    []validators.Validator{v3},
			expected: 1,
		},
		{
			name:     "boundary case: exactly 14",
			input:    []validators.Validator{&mockValidator{weight: 14}},
			expected: 0,
		},
		{
			name:     "boundary case: exactly 16",
			input:    []validators.Validator{&mockValidator{weight: 16}},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterQuick(tt.input)

			if len(result) != tt.expected {
				t.Errorf("expected %d validators, got %d", tt.expected, len(result))
			}

			// Verify all returned validators have weight >= 15
			for _, v := range result {
				if v.Weight() < 15 {
					t.Errorf("validator %s has weight %d, expected >= 15", v.ID(), v.Weight())
				}
			}
		})
	}
}

func TestFormatResultIcon(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"pass", "✅"},
		{"fail", "❌"},
		{"error", "❌"},
		{"warn", "⚠️"},
		{"skip", "✅"},
		{"unknown", "✅"},
		{"", "✅"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := formatResultIcon(tt.status)
			if result != tt.expected {
				t.Errorf("formatResultIcon(%q) = %q, expected %q", tt.status, result, tt.expected)
			}
		})
	}
}

func TestFindRepoRoot(t *testing.T) {
	// Create a valid OVAV repo in a temp dir and chdir into it
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav"), 0755)
	os.MkdirAll(filepath.Join(dir, ".ovav", "plan"), 0755)
	os.WriteFile(filepath.Join(dir, ".ovav", "plan", "caps.yaml"), []byte("canonical: true\n"), 0644)
	os.MkdirAll(filepath.Join(dir, "go-runtime"), 0755)
	os.WriteFile(filepath.Join(dir, "go-runtime", "go.mod"), []byte("module x\n"), 0644)

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	result := findRepoRoot()
	if result == "." {
		t.Error("findRepoRoot() returned '.', expected real path")
	}
	if !isOVAVRepo(result) {
		t.Errorf("findRepoRoot() = %q, not a valid OVAV repo", result)
	}
}

func TestFindRepoRoot_NotFound(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	result := findRepoRoot()
	if result != "." {
		t.Errorf("findRepoRoot() in empty dir = %q, want '.'", result)
	}
}

// ── runValidation tests ─────────────────────────────────────────────

func TestRunValidation_AllPass(t *testing.T) {
	root := setupFakeRepo(t)
	reg := validators.NewRegistry(
		&configurableMock{id: "v1", name: "Check1", weight: 10, status: "pass"},
		&configurableMock{id: "v2", name: "Check2", weight: 20, status: "pass"},
		&configurableMock{id: "v3", name: "Check3", weight: 5, status: "pass"},
	)

	var buf bytes.Buffer
	passed, failed, err := runValidation(context.Background(), reg, validationOpts{root: root}, &buf)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if passed != 3 {
		t.Errorf("passed = %d, want 3", passed)
	}
	if failed != 0 {
		t.Errorf("failed = %d, want 0", failed)
	}
	output := buf.String()
	if !strings.Contains(output, "3 passed") {
		t.Errorf("output missing '3 passed': %s", output)
	}
}

func TestRunValidation_WithFailures(t *testing.T) {
	root := setupFakeRepo(t)
	reg := validators.NewRegistry(
		&configurableMock{id: "v1", name: "Pass1", weight: 10, status: "pass"},
		&configurableMock{id: "v2", name: "Fail1", weight: 20, status: "fail", issues: []string{"bad thing"}},
		&configurableMock{id: "v3", name: "Err1", weight: 15, status: "error"},
	)

	var buf bytes.Buffer
	passed, failed, err := runValidation(context.Background(), reg, validationOpts{root: root}, &buf)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if passed != 1 {
		t.Errorf("passed = %d, want 1", passed)
	}
	if failed != 2 {
		t.Errorf("failed = %d, want 2", failed)
	}
	output := buf.String()
	if !strings.Contains(output, "bad thing") {
		t.Errorf("output missing issue detail: %s", output)
	}
}

func TestRunValidation_JSONOutput(t *testing.T) {
	root := setupFakeRepo(t)
	reg := validators.NewRegistry(
		&configurableMock{id: "v1", name: "Check1", weight: 10, status: "pass"},
		&configurableMock{id: "v2", name: "Check2", weight: 20, status: "fail"},
	)

	var buf bytes.Buffer
	_, _, err := runValidation(context.Background(), reg, validationOpts{root: root, jsonOut: true}, &buf)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var results []validators.Result
	if err := json.Unmarshal(buf.Bytes(), &results); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}
	if len(results) != 2 {
		t.Errorf("JSON results count = %d, want 2", len(results))
	}
	if results[0].ID != "v1" || results[1].ID != "v2" {
		t.Errorf("unexpected result IDs: %+v", results)
	}
}

func TestRunValidation_QuickFilter(t *testing.T) {
	root := setupFakeRepo(t)
	reg := validators.NewRegistry(
		&configurableMock{id: "low1", name: "Low1", weight: 5, status: "pass"},
		&configurableMock{id: "low2", name: "Low2", weight: 10, status: "pass"},
		&configurableMock{id: "high1", name: "High1", weight: 15, status: "pass"},
		&configurableMock{id: "high2", name: "High2", weight: 25, status: "pass"},
	)

	var buf bytes.Buffer
	passed, _, err := runValidation(context.Background(), reg, validationOpts{root: root, quick: true}, &buf)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if passed != 2 {
		t.Errorf("passed = %d, want 2 (only high-weight)", passed)
	}
	output := buf.String()
	if strings.Contains(output, "low1") || strings.Contains(output, "low2") {
		t.Errorf("quick mode should skip low-weight validators, output: %s", output)
	}
}

func TestRunValidation_IDFilter(t *testing.T) {
	root := setupFakeRepo(t)
	reg := validators.NewRegistry(
		&configurableMock{id: "alpha", name: "Alpha", weight: 10, status: "pass"},
		&configurableMock{id: "beta", name: "Beta", weight: 20, status: "fail"},
		&configurableMock{id: "gamma", name: "Gamma", weight: 15, status: "pass"},
	)

	var buf bytes.Buffer
	passed, failed, err := runValidation(context.Background(), reg, validationOpts{root: root, id: "beta"}, &buf)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if passed != 0 {
		t.Errorf("passed = %d, want 0", passed)
	}
	if failed != 1 {
		t.Errorf("failed = %d, want 1", failed)
	}
	output := buf.String()
	if !strings.Contains(output, "Beta") {
		t.Errorf("output should contain 'Beta': %s", output)
	}
	if strings.Contains(output, "Alpha") || strings.Contains(output, "Gamma") {
		t.Errorf("ID filter should only run 'beta', output: %s", output)
	}
}

func TestRunValidation_IDFilter_NotFound(t *testing.T) {
	root := setupFakeRepo(t)
	reg := validators.NewRegistry(
		&configurableMock{id: "alpha", name: "Alpha", weight: 10, status: "pass"},
	)

	var buf bytes.Buffer
	_, _, err := runValidation(context.Background(), reg, validationOpts{root: root, id: "nonexistent"}, &buf)

	if err == nil {
		t.Fatal("expected error for unknown validator ID, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found': %v", err)
	}
}

func TestRunValidation_InvalidRepo(t *testing.T) {
	dir := t.TempDir() // no .ovav/ or go-runtime/go.mod
	reg := validators.NewRegistry(
		&configurableMock{id: "v1", name: "Check1", weight: 10, status: "pass"},
	)

	var buf bytes.Buffer
	_, _, err := runValidation(context.Background(), reg, validationOpts{root: dir}, &buf)

	if err == nil {
		t.Fatal("expected error for invalid repo, got nil")
	}
	if !strings.Contains(err.Error(), "not a valid OVAV repository") {
		t.Errorf("error should mention invalid repo: %v", err)
	}
}
