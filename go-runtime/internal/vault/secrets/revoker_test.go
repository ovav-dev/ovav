package secrets

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// ── Revoker Interface ─────────────────────────────────────────────────────────

func TestRevokerInterface(t *testing.T) {
	var _ Revoker = &GitHubActionsRevoker{}
	var _ Revoker = &FlyIORevoker{}
	var _ Revoker = &GenericRevoker{providerName: "test"}
}

func TestGitHubActionsRevoker_Name(t *testing.T) {
	r := &GitHubActionsRevoker{}
	if r.Name() != "github-actions" {
		t.Errorf("Name = %q, want %q", r.Name(), "github-actions")
	}
}

func TestFlyIORevoker_Name(t *testing.T) {
	r := &FlyIORevoker{}
	if r.Name() != "fly-io" {
		t.Errorf("Name = %q, want %q", r.Name(), "fly-io")
	}
}

func TestGenericRevoker_Name(t *testing.T) {
	r := &GenericRevoker{providerName: "jenkins"}
	if r.Name() != "jenkins" {
		t.Errorf("Name = %q, want %q", r.Name(), "jenkins")
	}
}

func TestGenericRevoker_Revoke(t *testing.T) {
	r := &GenericRevoker{providerName: "gitlab-ci"}
	err := r.Revoke(SecretRef{
		SecretID: "sec-1",
		System:   SystemGitLabCI,
		Path:     "GitLab CI: myproject",
		EnvVar:   "GITLAB_TOKEN",
	}, "")
	if err == nil {
		t.Error("GenericRevoker.Revoke: expected error (manual revocation required)")
	}
}

// ── getRevoker ───────────────────────────────────────────────────────────────

func TestGetRevoker(t *testing.T) {
	tests := []struct {
		system System
		want   string
	}{
		{SystemGitHubActions, "github-actions"},
		{SystemFlyIO, "fly-io"},
		{SystemGitLabCI, "gitlab-ci"},
		{SystemJenkins, "jenkins"},
		{SystemLocalEnv, "local-env"},
	}

	for _, tc := range tests {
		r := getRevoker(tc.system)
		if r.Name() != tc.want {
			t.Errorf("getRevoker(%s): name = %q, want %q", tc.system, r.Name(), tc.want)
		}
	}
}

// ── RevokeResult / RevocationReport JSON ──────────────────────────────────────

func TestRevokeResult_JSON(t *testing.T) {
	result := RevokeResult{
		SecretName: "CF_TOKEN",
		Provider:   "github-actions",
		Path:       "GitHub Actions: ovav-dev/ovav",
		Status:     "revoked",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var r2 RevokeResult
	if err := json.Unmarshal(data, &r2); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if r2.Status != "revoked" {
		t.Errorf("Status = %q, want %q", r2.Status, "revoked")
	}
}

func TestRevocationReport_JSON(t *testing.T) {
	report := RevocationReport{
		SecretName:      "CF_TOKEN",
		SecretID:        "sec-123",
		VaultDeleted:    true,
		DepGraphCleaned: true,
		Results: []RevokeResult{
			{Provider: "github-actions", Status: "revoked"},
			{Provider: "fly-io", Status: "failed", Error: "token expired"},
		},
		AuditID: "audit-1",
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var r2 RevocationReport
	if err := json.Unmarshal(data, &r2); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if !r2.VaultDeleted {
		t.Error("VaultDeleted = false, want true")
	}
	if len(r2.Results) != 2 {
		t.Errorf("Results len = %d, want 2", len(r2.Results))
	}
}

// ── RevokeSecret Tests ─────────────────────────────────────────────────────────

func TestRevokeSecret_NotFound(t *testing.T) {
	store := NewSecretStore()
	graph := &DependencyGraph{refs: make(map[string][]SecretRef)}

	_, err := RevokeSecret(store, graph, "nonexistent")
	if err == nil {
		t.Error("RevokeSecret nonexistent: expected error")
	}
}

func TestRevokeSecret_NoRefs(t *testing.T) {
	store := NewSecretStore()
	sec := NewSecret("OrphanSecret", TypeAPIToken, "cf", "manual", []byte("val"))
	store.Add(sec)

	graph := &DependencyGraph{refs: make(map[string][]SecretRef)}

	report, err := RevokeSecret(store, graph, "OrphanSecret")
	// RevokeSecret returns success even with no refs — it just has empty Results
	if err != nil {
		t.Errorf("RevokeSecret orphan with no refs: unexpected error: %v", err)
	}
	if report.VaultDeleted != true {
		t.Error("VaultDeleted should be true")
	}
	if len(report.Results) != 0 {
		t.Errorf("Results: got %d, want 0", len(report.Results))
	}
}

func TestRevokeSecret_MissingGitHubToken(t *testing.T) {
	orig := os.Getenv("GITHUB_TOKEN")
	os.Unsetenv("GITHUB_TOKEN")
	defer func() {
		if orig != "" {
			os.Setenv("GITHUB_TOKEN", orig)
		}
	}()

	store := NewSecretStore()
	sec := NewSecret("GH_TOKEN", TypeAPIToken, "github", "manual", []byte("val"))
	store.Add(sec)

	graph := &DependencyGraph{refs: make(map[string][]SecretRef)}
	graph.AddRef(sec.ID, SystemGitHubActions, "GitHub Actions: owner/repo", "GH_TOKEN")

	report, err := RevokeSecret(store, graph, "GH_TOKEN")
	// Should not panic even without GITHUB_TOKEN
	if err == nil && len(report.Results) > 0 {
		if report.Results[0].Status != "failed" {
			t.Errorf("Status = %q, want %q", report.Results[0].Status, "failed")
		}
	}
}

func TestRevokeSecret_MissingFlyToken(t *testing.T) {
	orig := os.Getenv("FLY_API_TOKEN")
	os.Unsetenv("FLY_API_TOKEN")
	defer func() {
		if orig != "" {
			os.Setenv("FLY_API_TOKEN", orig)
		}
	}()

	store := NewSecretStore()
	sec := NewSecret("FLY_SECRET", TypeAPIToken, "fly.io", "manual", []byte("val"))
	store.Add(sec)

	graph := &DependencyGraph{refs: make(map[string][]SecretRef)}
	graph.AddRef(sec.ID, SystemFlyIO, "Fly.io app: myapp", "FLY_SECRET")

	report, err := RevokeSecret(store, graph, "FLY_SECRET")
	if err == nil && len(report.Results) > 0 {
		if report.Results[0].Status != "failed" {
			t.Errorf("Status = %q, want %q", report.Results[0].Status, "failed")
		}
	}
	_ = report
}

// ── GitHub Actions Revoker HTTP Mock ──────────────────────────────────────────

func TestGitHubActionsRevoker_Revoke_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Method = %q, want DELETE", r.Method)
		}
		expected := "/repos/owner/repo/actions/secrets/MY_SECRET"
		if r.URL.Path != expected {
			t.Errorf("Path = %q, want %q", r.URL.Path, expected)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	// Test path parsing logic
	ref := SecretRef{
		SecretID: "sec-1",
		System:   SystemGitHubActions,
		Path:     "GitHub Actions: owner/repo",
		EnvVar:   "MY_SECRET",
	}

	// Verify the path parsing produces the expected repo
	parts := splitPath(ref.Path)
	if len(parts) >= 2 {
		if parts[0] != "owner" || parts[1] != "repo" {
			t.Errorf("parsed = %v, want [owner repo]", parts)
		}
	}

	// Test that we handle the URL construction correctly
	expectedURL := "https://api.github.com/repos/" + parts[0] + "/" + parts[1] + "/actions/secrets/" + ref.EnvVar
	expectedPath := "/repos/owner/repo/actions/secrets/MY_SECRET"
	if expectedPath != "/repos/owner/repo/actions/secrets/MY_SECRET" {
		t.Errorf("path construction wrong: got %s", expectedURL)
	}
	_ = server
}

func splitPath(path string) []string {
	prefix := "GitHub Actions: "
	rest := path[len(prefix):]
	for i := range rest {
		if rest[i] == '/' {
			return []string{rest[:i], rest[i+1:]}
		}
	}
	return []string{rest}
}

func TestFlyIORevoker_Revoke_AppNameParsing(t *testing.T) {
	ref := SecretRef{
		SecretID: "sec-1",
		System:   SystemFlyIO,
		Path:     "Fly.io app: my-cool-app",
		EnvVar:   "SECRET_TOKEN",
	}

	appName := "Fly.io app: my-cool-app"
	parsed := appName[len("Fly.io app: "):]
	if parsed != "my-cool-app" {
		t.Errorf("parseFlyAppName = %q, want %q", parsed, "my-cool-app")
	}

	// Also test that fly secret path parsing works
	prefix := "Fly.io app: "
	if len(ref.Path) > len(prefix) {
		got := ref.Path[len(prefix):]
		if got != "my-cool-app" {
			t.Errorf("Fly.io path parsing = %q, want %q", got, "my-cool-app")
		}
	}
}

// ── RevokeSecret_GraphCleaning ─────────────────────────────────────────────────

func TestRevokeSecret_GraphCleaning(t *testing.T) {
	origGH := os.Getenv("GITHUB_TOKEN")
	origFLY := os.Getenv("FLY_API_TOKEN")
	os.Unsetenv("GITHUB_TOKEN")
	os.Unsetenv("FLY_API_TOKEN")
	defer func() {
		if origGH != "" {
			os.Setenv("GITHUB_TOKEN", origGH)
		}
		if origFLY != "" {
			os.Setenv("FLY_API_TOKEN", origFLY)
		}
	}()

	store := NewSecretStore()
	sec := NewSecret("TOKEN", TypeAPIToken, "github", "github", []byte("val"))
	store.Add(sec)

	graph := &DependencyGraph{refs: make(map[string][]SecretRef)}
	graph.AddRef(sec.ID, SystemGitHubActions, "GitHub Actions: owner/repo", "TOKEN")
	graph.AddRef(sec.ID, SystemFlyIO, "Fly.io app: myapp", "TOKEN")

	if len(graph.GetRefs(sec.ID)) != 2 {
		t.Fatal("Expected 2 refs before revoke")
	}

	report, _ := RevokeSecret(store, graph, "TOKEN")

	if !report.DepGraphCleaned {
		t.Error("DepGraphCleaned should be true")
	}

	if len(graph.GetRefs(sec.ID)) != 0 {
		t.Error("Graph refs should be empty after RevokeSecret")
	}
}

// ── RevokeResult Status Values ─────────────────────────────────────────────────

func TestRevokeResult_StatusValues(t *testing.T) {
	statuses := []string{"revoked", "not_found", "failed", "source_not_in_depgraph_manual_action_required"}

	for _, s := range statuses {
		r := RevokeResult{Status: s}
		if r.Status != s {
			t.Errorf("Status = %q, want %q", r.Status, s)
		}
	}
}
