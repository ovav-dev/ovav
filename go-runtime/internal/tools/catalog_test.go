package tools

import (
	"strings"
	"testing"
)

func TestCatalogNotEmpty(t *testing.T) {
	catalog := Catalog()
	if len(catalog) == 0 {
		t.Fatal("catalog is empty — expected at least 30 tools")
	}
	if len(catalog) < 30 {
		t.Errorf("catalog has %d tools — expected at least 30", len(catalog))
	}
}

func TestCatalogIsSortedCopy(t *testing.T) {
	original := Catalog()
	// Modify the copy — should not affect builtinCatalog
	if len(original) > 0 {
		original[0].Name = "MODIFIED"
	}
	// builtinCatalog should be unchanged
	if builtinCatalog[0].Name == "MODIFIED" {
		t.Error("Catalog() returned reference to builtinCatalog — expected copy")
	}
}

func TestByID(t *testing.T) {
	tests := []struct {
		id       string
		expected string // expected Name
		wantNil  bool
	}{
		{"ovav-cli", "OVAV CLI (Go)", false},
		{"vault-go", "Vault (Go)", false},
		{"pipeline", "Quality Pipeline (harnesses)", false},
		{"nonexistent", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		tool := ByID(tt.id)
		if tt.wantNil {
			if tool != nil {
				t.Errorf("ByID(%q) = %v, want nil", tt.id, tool)
			}
		} else {
			if tool == nil {
				t.Errorf("ByID(%q) = nil, want %q", tt.id, tt.expected)
			} else if tool.Name != tt.expected {
				t.Errorf("ByID(%q).Name = %q, want %q", tt.id, tool.Name, tt.expected)
			}
		}
	}
}

func TestAllToolsHaveRequiredFields(t *testing.T) {
	for _, tool := range builtinCatalog {
		if tool.ID == "" {
			t.Error("found tool with empty ID")
		}
		if tool.Name == "" {
			t.Errorf("tool %q has empty Name", tool.ID)
		}
		if tool.Description == "" {
			t.Errorf("tool %q has empty Description", tool.ID)
		}
		if tool.Path == "" {
			t.Errorf("tool %q has empty Path", tool.ID)
		}
		if tool.Category == "" {
			t.Errorf("tool %q has empty Category", tool.ID)
		}
		if tool.Language == "" {
			t.Errorf("tool %q has empty Language", tool.ID)
		}
		if tool.Status == "" {
			t.Errorf("tool %q has empty Status", tool.ID)
		}
	}
}

func TestNoDuplicateIDs(t *testing.T) {
	seen := make(map[string]bool)
	for _, tool := range builtinCatalog {
		if seen[tool.ID] {
			t.Errorf("duplicate tool ID: %q", tool.ID)
		}
		seen[tool.ID] = true
	}
}

func TestSearch(t *testing.T) {
	tests := []struct {
		keyword  string
		minCount int
	}{
		{"vault", 2},      // Python vault + Go vault
		{"go", 3},         // Go tools in name
		{"security", 5},   // All security tools
		{"cockpit", 1},    // Cockpit TUI
		{"xyzzy_none", 0}, // Nonexistent
		{"runtime", 3},    // Runtime tools
	}

	for _, tt := range tests {
		results := Search(tt.keyword)
		if len(results) < tt.minCount {
			t.Errorf("Search(%q) = %d results, want >= %d", tt.keyword, len(results), tt.minCount)
		}
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	lower := Search("vault")
	upper := Search("VAULT")
	if len(lower) != len(upper) {
		t.Errorf("Search case-insensitive: 'vault'=%d, 'VAULT'=%d", len(lower), len(upper))
	}
}

func TestByCategory(t *testing.T) {
	cats := Categories()
	for _, cat := range cats {
		tools := ByCategory(cat)
		if len(tools) == 0 {
			t.Errorf("ByCategory(%q) returned 0 tools", cat)
		}
		for _, tool := range tools {
			if !strings.EqualFold(tool.Category, cat) {
				t.Errorf("tool %q has category %q, expected %q", tool.ID, tool.Category, cat)
			}
		}
	}
}

func TestByLanguage(t *testing.T) {
	goTools := ByLanguage(LangGo)
	if len(goTools) == 0 {
		t.Error("expected at least 1 Go tool")
	}
	for _, tool := range goTools {
		if tool.Language != LangGo {
			t.Errorf("tool %q has language %q, expected Go", tool.ID, tool.Language)
		}
	}

	pyTools := ByLanguage(LangPython)
	if len(pyTools) == 0 {
		t.Error("expected at least 1 Python tool")
	}

	shellTools := ByLanguage(LangShell)
	if len(shellTools) == 0 {
		t.Error("expected at least 1 Shell tool")
	}

	tsTools := ByLanguage(LangTS)
	if len(tsTools) == 0 {
		t.Error("expected at least 1 TypeScript tool")
	}
}

func TestByStatus(t *testing.T) {
	active := ByStatus(StatusActive)
	if len(active) < 30 {
		t.Errorf("expected >=30 active tools, got %d", len(active))
	}

	frozen := ByStatus(StatusFrozen)
	if len(frozen) < 2 {
		t.Errorf("expected >=2 frozen tools, got %d", len(frozen))
	}

	planned := ByStatus(StatusPlanned)
	// Zero planned tools is valid — everything is either active or frozen
	_ = planned
}

func TestCategoriesNotEmpty(t *testing.T) {
	cats := Categories()
	if len(cats) == 0 {
		t.Fatal("categories list is empty")
	}
	if len(cats) < 4 {
		t.Errorf("expected at least 4 categories, got %d", len(cats))
	}
}

func TestCategoriesSorted(t *testing.T) {
	cats := Categories()
	for i := 1; i < len(cats); i++ {
		if cats[i] < cats[i-1] {
			t.Errorf("categories not sorted: %q < %q", cats[i], cats[i-1])
		}
	}
}

func TestFormatList(t *testing.T) {
	all := Catalog()
	output := FormatList(all, false)
	if !strings.Contains(output, "OVAV Tool Catalog") {
		t.Error("FormatList should contain catalog header")
	}
	if !strings.Contains(output, "ovav tools show") {
		t.Error("FormatList should include usage hint")
	}

	// Detailed mode
	detailed := FormatList(all[:3], true)
	if !strings.Contains(detailed, "id:") {
		t.Error("Detailed FormatList should show IDs")
	}
	if !strings.Contains(detailed, "run:") {
		t.Error("Detailed FormatList should show commands")
	}
}

func TestFormatListEmpty(t *testing.T) {
	output := FormatList(nil, false)
	if !strings.Contains(output, "No tools found") {
		t.Errorf("empty list: got %q", output)
	}
}

func TestFormatTool(t *testing.T) {
	tool := ByID("ovav-cli")
	if tool == nil {
		t.Fatal("ovav-cli not found")
	}
	output := FormatTool(tool)
	checks := []string{
		"OVAV CLI (Go)",
		"ovav-cli",
		"Go",
		"cli",
		"ovav status",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("FormatTool missing %q in output", check)
		}
	}
}

func TestFormatToolNil(t *testing.T) {
	output := FormatTool(nil)
	if output != "Tool not found." {
		t.Errorf("nil tool: got %q", output)
	}
}

func TestFormatCategories(t *testing.T) {
	output := FormatCategories()
	if !strings.Contains(output, "Tool Categories") {
		t.Error("FormatCategories missing header")
	}
	for _, cat := range Categories() {
		if !strings.Contains(output, cat) {
			t.Errorf("FormatCategories missing category %q", cat)
		}
	}
}

func TestGoToolsHaveGoBinary(t *testing.T) {
	goTools := ByLanguage(LangGo)
	binaryTools := 0
	for _, tool := range goTools {
		if tool.GoBinary != "" {
			binaryTools++
		}
	}
	if binaryTools < 2 {
		t.Errorf("expected at least 2 Go tools with GoBinary field, got %d", binaryTools)
	}
}

func TestFrozenToolsArePython(t *testing.T) {
	frozen := ByStatus(StatusFrozen)
	for _, tool := range frozen {
		if tool.Language != LangPython {
			t.Errorf("frozen tool %q has language %q — expected Python (frozen = Python deprecated)", tool.ID, tool.Language)
		}
	}
}
