//go:build linux

package hostsync

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ovav/ovav/internal/hostprojection"
)

const v9fsModeDegradation = "destination mode enforcement unsupported on v9fs"

const (
	validOpenCode = "{\n  \"$schema\": \"https://opencode.ai/config.json\"\n}\n"
	validWSL      = "# Activation requires a natural full WSL stop before reopening.\n# OVAV MUST NOT automatically run wsl --shutdown or wsl --terminate.\n[wsl2]\nmemory=4GB\nprocessors=4\nswap=4GB\nnetworkingMode=mirrored\ndnsTunneling=true\nautoProxy=true\nfirewall=true\n\n[experimental]\nautoMemoryReclaim=dropCache\n"
	validWarp     = "name = \"OVAV WSL\"\ntitle = \"OVAV WSL\"\ncolor = \"blue\"\n\n[[panes]]\nid = \"main\"\ntype = \"terminal\"\ndirectory = \"~\"\nis_focused = true\n"
)

func TestProfilesAreExact(t *testing.T) {
	want := []Profile{
		{Name: "opencode-bootstrap", SourceRelative: "ops/host-projections/opencode-bootstrap.json"},
		{Name: "wsl2-resource-policy", SourceRelative: "ops/host-projections/wsl2/.wslconfig", Windows: true},
		{Name: "warp-wsl-tab", SourceRelative: "ops/host-projections/warp/ovav_wsl.toml", Windows: true},
	}
	if got := Profiles(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Profiles() = %#v, want %#v", got, want)
	}
	if IsProfile("opencode-bootstrap/../warp-wsl-tab") {
		t.Fatal("traversal-like profile name accepted")
	}
}

func TestCanonicalProfileSourcesValidate(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller could not locate test source")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	for _, definition := range profileRegistry {
		t.Run(definition.profile.Name, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(definition.profile.SourceRelative)))
			if err != nil {
				t.Fatalf("read canonical source: %v", err)
			}
			if err := definition.validate(content); err != nil {
				t.Fatalf("canonical source validation: %v", err)
			}
		})
	}
}

func TestPlanIsNoWriteAndApprovalIsRequired(t *testing.T) {
	fixture := newHostFixture(t, "opencode-bootstrap")
	result, err := Run(Request{
		RepoRoot: fixture.repo, Home: fixture.home, Profile: fixture.profile, Now: time.Unix(1, 0),
	})
	if err != nil {
		t.Fatalf("Run(plan): %v", err)
	}
	if result.Mode != "plan" || result.WritesPerformed || result.JournalPath == "" {
		t.Fatalf("Run(plan) = %+v", result)
	}
	assertFileContent(t, fixture.destination, "original")
	if _, err := os.Lstat(result.JournalPath); !os.IsNotExist(err) {
		t.Fatalf("plan journal exists: %v", err)
	}
	if _, err := Run(Request{
		RepoRoot: fixture.repo, Home: fixture.home, Profile: fixture.profile, Apply: true,
	}); err == nil || !strings.Contains(err.Error(), "approval") {
		t.Fatalf("unapproved apply error = %v", err)
	}
	assertFileContent(t, fixture.destination, "original")
}

func TestDurabilityDegradationPropagatesFromProjection(t *testing.T) {
	t.Parallel()
	profile := resolvedProfile{definition: profileDefinition{profile: Profile{Name: "test"}}}
	result := resultFromPreview(profile, hostprojection.Preview{
		Durability:       hostprojection.DurabilityDegraded,
		DurabilityDetail: v9fsModeDegradation,
	})
	if result.Durability != hostprojection.DurabilityDegraded || result.DurabilityDetail != v9fsModeDegradation {
		t.Fatalf("preview degradation = %+v", result)
	}
	result.mergeMutation(hostprojection.Result{
		Durability:       hostprojection.DurabilityDegraded,
		DurabilityDetail: v9fsModeDegradation,
	})
	if result.Durability != hostprojection.DurabilityDegraded || result.DurabilityDetail != v9fsModeDegradation {
		t.Fatalf("mutation degradation = %+v", result)
	}
}

func TestRunRejectsUnknownTraversalAndInvalidSource(t *testing.T) {
	fixture := newHostFixture(t, "wsl2-resource-policy")
	tests := []struct {
		name string
		req  Request
	}{
		{name: "unknown profile", req: Request{RepoRoot: fixture.repo, Home: fixture.home, Profile: "unknown"}},
		{name: "repo traversal", req: Request{RepoRoot: fixture.repo + "/child/..", Home: fixture.home, WindowsHome: fixture.windowsHome, Profile: fixture.profile}},
		{name: "windows traversal", req: Request{RepoRoot: fixture.repo, Home: fixture.home, WindowsHome: fixture.windowsHome + "/child/..", Profile: fixture.profile}},
		{name: "relative rollback", req: Request{RepoRoot: fixture.repo, Home: fixture.home, RollbackJournal: "journal.json", ApproveHostWrite: true}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Run(test.req); err == nil {
				t.Fatal("Run() error = nil, want rejection")
			}
		})
	}

	if err := os.WriteFile(fixture.source, []byte(strings.Replace(validWSL, "memory=4GB", "memory=8GB", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(Request{RepoRoot: fixture.repo, Home: fixture.home, WindowsHome: fixture.windowsHome, Profile: fixture.profile}); err == nil {
		t.Fatal("invalid WSL source accepted")
	}
}

func TestProfileContentValidationRejectsWidening(t *testing.T) {
	tests := []struct {
		name     string
		validate func([]byte) error
		content  string
	}{
		{name: "OpenCode agent", validate: validateOpenCodeBootstrap, content: `{"$schema":"https://opencode.ai/config.json","agent":{}}`},
		{name: "OpenCode provider", validate: validateOpenCodeBootstrap, content: `{"$schema":"https://opencode.ai/config.json","provider":{}}`},
		{name: "OpenCode permission", validate: validateOpenCodeBootstrap, content: `{"$schema":"https://opencode.ai/config.json","permission":"allow"}`},
		{name: "WSL memory", validate: validateWSLResourcePolicy, content: strings.Replace(validWSL, "memory=4GB", "memory=8GB", 1)},
		{name: "Warp shell", validate: validateWarpWSLTab, content: validWarp + "shell = \"bash\"\n"},
		{name: "Warp commands", validate: validateWarpWSLTab, content: validWarp + "commands = []\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.validate([]byte(test.content)); err == nil {
				t.Fatal("widened source content accepted")
			}
		})
	}
	for name, test := range map[string]struct {
		validate func([]byte) error
		content  string
	}{
		"OpenCode": {validateOpenCodeBootstrap, validOpenCode},
		"WSL":      {validateWSLResourcePolicy, validWSL},
		"Warp":     {validateWarpWSLTab, validWarp},
	} {
		t.Run(name+" valid", func(t *testing.T) {
			if err := test.validate([]byte(test.content)); err != nil {
				t.Fatalf("canonical content rejected: %v", err)
			}
		})
	}
}

func TestApplyAndRollbackUseTemporaryHomes(t *testing.T) {
	for _, profile := range []string{"opencode-bootstrap", "wsl2-resource-policy", "warp-wsl-tab"} {
		t.Run(profile, func(t *testing.T) {
			fixture := newHostFixture(t, profile)
			request := Request{
				RepoRoot: fixture.repo, Home: fixture.home,
				Profile: profile, Apply: true, ApproveHostWrite: true, Now: time.Unix(2, 0),
			}
			if profile != "opencode-bootstrap" {
				request.WindowsHome = fixture.windowsHome
			}
			applied, err := Run(request)
			if err != nil || !applied.Applied || !applied.WritesPerformed || applied.JournalPath == "" {
				t.Fatalf("Run(apply) = %+v, %v", applied, err)
			}
			assertFileContent(t, fixture.destination, fixture.sourceContent)
			backupRoot := filepath.Join(fixture.home, ".local", "state", "ovav", "host-projection")
			if info, err := os.Stat(backupRoot); err != nil || info.Mode().Perm() != 0o700 {
				t.Fatalf("backup root mode = %v, %v", info, err)
			}

			rollbackRequest := Request{
				RepoRoot: fixture.repo, Home: fixture.home,
				RollbackJournal: applied.JournalPath, ApproveHostWrite: true,
			}
			if profile != "opencode-bootstrap" {
				rollbackRequest.WindowsHome = fixture.windowsHome
			}
			rolled, err := Run(rollbackRequest)
			if err != nil || !rolled.RolledBack || rolled.Profile != profile || !rolled.WritesPerformed {
				t.Fatalf("Run(rollback) = %+v, %v", rolled, err)
			}
			assertFileContent(t, fixture.destination, "original")
		})
	}
}

func TestRollbackRejectsJournalSelectedAuthority(t *testing.T) {
	fixture := newHostFixture(t, "opencode-bootstrap")
	applied, err := Run(Request{
		RepoRoot: fixture.repo, Home: fixture.home, Profile: fixture.profile,
		Apply: true, ApproveHostWrite: true, Now: time.Unix(3, 0),
	})
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(applied.JournalPath)
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(filepath.Dir(fixture.home), "outside")
	tampered := strings.Replace(string(content), fixture.destination, outside, 1)
	if tampered == string(content) {
		t.Fatal("journal destination was not found for tamper fixture")
	}
	if err := os.WriteFile(applied.JournalPath, []byte(tampered), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(Request{
		RepoRoot: fixture.repo, Home: fixture.home,
		RollbackJournal: applied.JournalPath, ApproveHostWrite: true,
	}); err == nil {
		t.Fatal("rollback accepted journal-selected destination")
	}
	assertFileContent(t, fixture.destination, fixture.sourceContent)
	if _, err := os.Lstat(outside); !os.IsNotExist(err) {
		t.Fatalf("journal-selected path was touched: %v", err)
	}
}

type hostFixture struct {
	repo, home, windowsHome, profile, source, destination, sourceContent string
}

func newHostFixture(t *testing.T, profile string) hostFixture {
	t.Helper()
	base := t.TempDir()
	fixture := hostFixture{
		repo: filepath.Join(base, "repo"), home: filepath.Join(base, "home"),
		windowsHome: filepath.Join(base, "windows-home"), profile: profile,
	}
	for _, directory := range []string{fixture.repo, fixture.home, fixture.windowsHome} {
		mustMkdirAll(t, directory, 0o755)
	}
	switch profile {
	case "opencode-bootstrap":
		fixture.source = filepath.Join(fixture.repo, "ops", "host-projections", "opencode-bootstrap.json")
		fixture.destination = filepath.Join(fixture.home, ".config", "opencode", "opencode.json")
		fixture.sourceContent = validOpenCode
	case "wsl2-resource-policy":
		fixture.source = filepath.Join(fixture.repo, "ops", "host-projections", "wsl2", ".wslconfig")
		fixture.destination = filepath.Join(fixture.windowsHome, ".wslconfig")
		fixture.sourceContent = validWSL
	case "warp-wsl-tab":
		fixture.source = filepath.Join(fixture.repo, "ops", "host-projections", "warp", "ovav_wsl.toml")
		fixture.destination = filepath.Join(fixture.windowsHome, "AppData", "Roaming", "warp", "Warp", "data", "tab_configs", "ovav_wsl.toml")
		fixture.sourceContent = validWarp
	default:
		t.Fatalf("unknown fixture profile %q", profile)
	}
	mustMkdirAll(t, filepath.Dir(fixture.source), 0o755)
	mustMkdirAll(t, filepath.Dir(fixture.destination), 0o755)
	if err := os.WriteFile(fixture.source, []byte(fixture.sourceContent), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixture.destination, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func mustMkdirAll(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(path, mode); err != nil {
		t.Fatal(err)
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil || string(got) != want {
		t.Fatalf("%s = %q, %v; want %q", path, got, err, want)
	}
}
