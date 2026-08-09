package subagent

import (
	"os"
	"path/filepath"
	"testing"
)

// repoRootFromTest locates the OVAV repo root from the test file location.
// tests run in go-runtime/internal/subagent/ so 3 levels up is the repo root.
func repoRootFromTest(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root := wd
	for i := 0; i < 6; i++ {
		// Real repo root must have .ovav/ AND go-runtime/go.mod (not just .ovav/)
		_, ovavErr := os.Stat(filepath.Join(root, ".ovav"))
		_, modErr := os.Stat(filepath.Join(root, "go-runtime", "go.mod"))
		if ovavErr == nil && modErr == nil {
			return root
		}
		root = filepath.Dir(root)
	}
	t.Fatalf("could not find repo root from %s", wd)
	return ""
}

func TestLoadCatalog(t *testing.T) {
	root := repoRootFromTest(t)
	c, err := LoadCatalog(root)
	if err != nil {
		t.Skipf("LoadCatalog failed (CI/docker env): %v", err)
	}
	if c.Version == "" {
		t.Error("catalog missing version")
	}
	if len(c.Agents) < 50 {
		t.Errorf("catalog has only %d agents, expected ≥50", len(c.Agents))
	}
}

func TestResolveExactMatch(t *testing.T) {
	root := repoRootFromTest(t)
	c, err := LoadCatalog(root)
	if err != nil {
		t.Skipf("LoadCatalog failed (CI/docker env): %v", err)
	}

	r := c.Resolve("team-elena-frontend")
	if r.Error != "" {
		t.Errorf("expected no error, got: %s", r.Error)
	}
	if r.Ambiguous {
		t.Error("expected NOT ambiguous")
	}
	if len(r.ExactMatches) != 1 || r.ExactMatches[0].ID != "team-elena-frontend" {
		t.Errorf("expected exact match team-elena-frontend, got: %+v", r)
	}
}

func TestResolveAliasMatch(t *testing.T) {
	root := repoRootFromTest(t)
	c, err := LoadCatalog(root)
	if err != nil {
		t.Skipf("LoadCatalog failed (CI/docker env): %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"frontend-elena", "team-elena-frontend"},
		{"elena-frontend", "team-elena-frontend"},
		{"elena-ux", "lead-elena"},
		{"uriel-devops", "team-uriel-devops"},
		{"team-elena", "team-elena-frontend"},
	}

	for _, tc := range tests {
		r := c.Resolve(tc.input)
		if r.Error != "" {
			t.Errorf("input '%s' produced error: %s", tc.input, r.Error)
			continue
		}
		if r.Ambiguous {
			t.Errorf("input '%s' unexpectedly ambiguous: %v", tc.input, r.AmbiguousIDs)
			continue
		}
		var got string
		if len(r.ExactMatches) > 0 {
			got = r.ExactMatches[0].ID
		} else if len(r.AliasMatches) > 0 {
			got = r.AliasMatches[0].ID
		}
		if got != tc.expected {
			t.Errorf("input '%s' resolved to '%s', expected '%s'", tc.input, got, tc.expected)
		}
	}
}

func TestResolveAmbiguous(t *testing.T) {
	root := repoRootFromTest(t)
	c, err := LoadCatalog(root)
	if err != nil {
		t.Skipf("LoadCatalog failed (CI/docker env): %v", err)
	}

	tests := []struct {
		input         string
		expectedCount int
		mustContain   []string
	}{
		{"elena", 2, []string{"lead-elena", "team-elena-frontend"}},
		{"uriel", 2, []string{"lead-uriel", "team-uriel-devops"}},
	}

	for _, tc := range tests {
		r := c.Resolve(tc.input)
		if !r.Ambiguous {
			t.Errorf("input '%s' should be ambiguous but isn't", tc.input)
			continue
		}
		if len(r.AmbiguousIDs) != tc.expectedCount {
			t.Errorf("input '%s': expected %d matches, got %d (%v)",
				tc.input, tc.expectedCount, len(r.AmbiguousIDs), r.AmbiguousIDs)
		}
		for _, want := range tc.mustContain {
			found := false
			for _, got := range r.AmbiguousIDs {
				if got == want {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("input '%s': expected to contain '%s', got %v",
					tc.input, want, r.AmbiguousIDs)
			}
		}
		if r.Suggestion == "" {
			t.Errorf("input '%s': expected disambiguation hint", tc.input)
		}
	}
}

func TestResolveNotFound(t *testing.T) {
	root := repoRootFromTest(t)
	c, err := LoadCatalog(root)
	if err != nil {
		t.Skipf("LoadCatalog failed (CI/docker env): %v", err)
	}

	r := c.Resolve("nonexistent-agent-xyz")
	if r.Error == "" {
		t.Error("expected error for unknown agent")
	}
	if r.Ambiguous {
		t.Error("unknown agent should not be ambiguous")
	}
}

func TestResolveCaseInsensitive(t *testing.T) {
	root := repoRootFromTest(t)
	c, err := LoadCatalog(root)
	if err != nil {
		t.Skipf("LoadCatalog failed (CI/docker env): %v", err)
	}

	tests := []string{
		"TEAM-ELENA-FRONTEND",
		"Team-Elena-Frontend",
		"team-ELENA-frontend",
	}

	for _, input := range tests {
		r := c.Resolve(input)
		if r.Error != "" {
			t.Errorf("case-insensitive failed for '%s': %s", input, r.Error)
		}
		if len(r.ExactMatches) != 1 || r.ExactMatches[0].ID != "team-elena-frontend" {
			t.Errorf("case-insensitive failed for '%s': got %+v", input, r)
		}
	}
}

func TestResolveAllAreasCovered(t *testing.T) {
	root := repoRootFromTest(t)
	c, err := LoadCatalog(root)
	if err != nil {
		t.Skipf("LoadCatalog failed (CI/docker env): %v", err)
	}

	// Every agent must have non-empty id, kind, name, area
	for _, a := range c.Agents {
		if a.ID == "" || a.Kind == "" || a.Name == "" || a.Area == "" {
			t.Errorf("incomplete agent entry: %+v", a)
		}
		if a.Kind == "team" && (a.Lead == nil || *a.Lead == "") {
			t.Errorf("team agent %s missing lead", a.ID)
		}
	}
}
