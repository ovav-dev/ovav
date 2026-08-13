package permissions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPermissionAuthorityV3(t *testing.T) {
	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{name: "v3 accepted", version: "ovav.permission_authority.v3"},
		{name: "unknown rejected", version: "ovav.permission_authority.v4", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writePermissionPolicy(t, dir, tt.version, currentMaterializedTargets())

			err := NewPermissionAuthority(dir).assertPolicySafe()
			if (err != nil) != tt.wantErr {
				t.Fatalf("assertPolicySafe() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPermissionAuthorityUsesCurrentTargets(t *testing.T) {
	dir := t.TempDir()
	authority := NewPermissionAuthority(dir)

	wants := map[string]string{
		"platform": filepath.Join(dir, ".opencode", "agents", "area-platform-engineering.md"),
		"thavren":  filepath.Join(dir, ".opencode", "agents", "lead-thavren.md"),
	}
	got := map[string]string{
		"platform": authority.PlatformAgent,
		"thavren":  authority.ThavrenAlias,
	}
	for name, want := range wants {
		if got[name] != want {
			t.Errorf("%s target = %q, want %q", name, got[name], want)
		}
	}

	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, ".ovav", "policy", "permission_authority.json"))
	if err != nil {
		t.Fatalf("read canonical policy: %v", err)
	}
	var policy struct {
		MaterializedTargets []string `json:"materialized_targets"`
	}
	if err := json.Unmarshal(data, &policy); err != nil {
		t.Fatalf("parse canonical policy: %v", err)
	}
	for _, want := range currentMaterializedTargets() {
		if !containsString(policy.MaterializedTargets, want) {
			t.Errorf("canonical policy missing current target %q", want)
		}
	}
	for _, target := range policy.MaterializedTargets {
		if strings.HasPrefix(target, "clients/opencode/agents/") {
			t.Errorf("canonical policy contains retired materialized target %q", target)
		}
	}
}

func TestCanonicalPermissionsUseGoNativeCommands(t *testing.T) {
	for group, permissions := range map[string]map[string]string{
		"allows": RequiredAllows(),
		"denies": CriticalDenies(),
	} {
		for command := range permissions {
			if strings.Contains(command, "python3 tools/") {
				t.Errorf("%s contains removed Python tool command %q", group, command)
			}
		}
	}

	allows := RequiredAllows()
	for _, command := range []string{
		"go run -C go-runtime ./cmd/ovav validate*",
		"go run -C go-runtime ./internal/validators/cmd/validate*",
	} {
		if allows[command] != "allow" {
			t.Errorf("Go-native command %q = %q, want allow", command, allows[command])
		}
	}
	if allows["gh pr create*"] != "allow" {
		t.Errorf("gh pr create host permission = %q, want allow", allows["gh pr create*"])
	}
}

func TestCanonicalPermissionsRetainCriticalDenies(t *testing.T) {
	tests := []struct {
		pattern string
	}{
		{pattern: "git push*"},
		{pattern: "git push --force *"},
		{pattern: "git push --force-with-lease *"},
		{pattern: "git push -f *"},
		{pattern: "sudo *"},
		{pattern: "gh auth token*"},
		{pattern: "gh auth login*"},
		{pattern: "pip install *"},
		{pattern: "pip3 install *"},
		{pattern: "npm install *"},
		{pattern: "pnpm add *"},
		{pattern: "yarn add *"},
		{pattern: "apt install *"},
		{pattern: "apt-get install *"},
		{pattern: "go install *"},
	}

	denies := CriticalDenies()
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			if denies[tt.pattern] != "deny" {
				t.Errorf("CriticalDenies()[%q] = %q, want deny", tt.pattern, denies[tt.pattern])
			}
		})
	}
}

func TestCanonicalProtectedDeniesAreMaterialized(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	authority := NewPermissionAuthority(repoRoot)
	permissions, err := authority.expectedBashPermissions()
	if err != nil {
		t.Fatal(err)
	}
	denies, err := authority.protectedBashDenies()
	if err != nil {
		t.Fatal(err)
	}
	for _, pattern := range denies {
		if permissions[pattern] != "deny" {
			t.Errorf("canonical protected deny %q = %q, want deny", pattern, permissions[pattern])
		}
	}
	if permissions["*"] != "allow" {
		t.Errorf("routine host wildcard = %q, want allow", permissions["*"])
	}
}

func TestExternalDirectoryIsExplicitlyScoped(t *testing.T) {
	permissions := ExpectedExternalDirectory("")
	for _, forbidden := range []string{"/home/braka/*", "*"} {
		if permissions[forbidden] == "allow" {
			t.Errorf("broad external write grant remains: %s", forbidden)
		}
	}
	for _, required := range []string{
		"/home/braka/Systems/ovav/*",
		"/home/braka/.config/ovav/*",
		"/home/braka/.local/share/ovav/*",
		"/home/braka/.local/state/ovav-opencode/*",
		"/mnt/c/Users/Alexa/AppData/Roaming/wezterm/*",
	} {
		if permissions[required] != "allow" {
			t.Errorf("explicit external grant %q = %q, want allow", required, permissions[required])
		}
	}
}

func TestExternalDirectoryHasNoOVAVPathTypo(t *testing.T) {
	for path := range ExpectedExternalDirectory("") {
		if strings.Contains(path, "/home/braka/..ovav") {
			t.Errorf("external directory contains malformed path %q", path)
		}
	}
}

func TestAgentProjectionOrdersWildcardBeforeCriticalRules(t *testing.T) {
	lines := strings.Join(expectedAgentPermissionYAML("area-platform-engineering"), "\n")
	tests := []struct {
		name     string
		wildcard string
		rule     string
	}{
		{name: "bash allow before push deny", wildcard: `    "*": allow`, rule: `    "git push*": deny`},
		{name: "external deny before path allow", wildcard: `    "*": deny`, rule: `    "/home/braka/Systems/ovav/*": allow`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wildcardIndex := strings.Index(lines, tt.wildcard)
			ruleIndex := strings.Index(lines, tt.rule)
			if wildcardIndex < 0 || ruleIndex < 0 || wildcardIndex >= ruleIndex {
				t.Errorf("wildcard must precede specific rule:\n%s", lines)
			}
		})
	}
}

func currentMaterializedTargets() []string {
	return []string{
		"opencode.json",
		".opencode/agents/area-platform-engineering.md",
		".opencode/agents/lead-thavren.md",
	}
}

func writePermissionPolicy(t *testing.T, root, version string, targets []string) {
	t.Helper()
	policyDir := filepath.Join(root, ".ovav", "policy")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("create policy directory: %v", err)
	}
	data, err := json.Marshal(map[string]any{
		"schema_version":       version,
		"materialized_targets": targets,
	})
	if err != nil {
		t.Fatalf("marshal policy: %v", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "permission_authority.json"), data, 0o644); err != nil {
		t.Fatalf("write policy: %v", err)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
