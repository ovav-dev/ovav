// fish_cli_test.go — unit tests for the ovav fish sub-command.
//
// These run against a temporary fixture so the production ~/.config/fish
// is never touched. The fixture directory mimics /home/braka/.config/fish
// with a fake conf.d + control files.

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureFish sets up a fake OVAV repo + fake ~/.config/fish and writes
// a few canonical fish files. Returns (canonicalRoot, userConfigDir,
// cleanup). It also writes a marker ovav.fish so fishCanonicalRoot can
// discover the repo by walking up from cwd.
func fixtureFish(t *testing.T, names []string) (string, string, func()) {
	t.Helper()

	tmp := t.TempDir()
	canonicalRoot := filepath.Join(tmp, "repo")
	userConfigDir := filepath.Join(tmp, "home", ".config", "fish")
	fishDir := filepath.Join(canonicalRoot, "config", "fish")
	if err := os.MkdirAll(fishDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(userConfigDir, "conf.d"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Always include the marker ovav.fish so fishCanonicalRoot finds this
	// repo. It is created BEFORE the requested names so callers can rely
	// on its presence and content.
	if err := os.WriteFile(filepath.Join(fishDir, "ovav.fish"),
		[]byte("# canonical fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, n := range names {
		path := filepath.Join(fishDir, n)
		body := "# fixture " + n + "\n"
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return canonicalRoot, userConfigDir, func() {}
}

// chdir changes cwd to dir for the duration of fn.
func chdir(t *testing.T, dir string, fn func()) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(old) }()
	fn()
}

// ─── Tests ───

func TestFishCanonicalRoot_FindsRepoByCwd(t *testing.T) {
	canonical, _, cleanup := fixtureFish(t, []string{"05-a.fish"})
	defer cleanup()
	chdir(t, canonical, func() {
		got, err := fishCanonicalRoot()
		if err != nil {
			t.Fatalf("fishCanonicalRoot: %v", err)
		}
		// Strip trailing slash for cross-platform safety.
		got = strings.TrimRight(got, string(os.PathSeparator))
		if got != canonical {
			t.Errorf("got %q, want %q", got, canonical)
		}
	})
}

func TestFishCanonicalRoot_EnvOverride(t *testing.T) {
	canonical, _, cleanup := fixtureFish(t, []string{"05-a.fish"})
	defer cleanup()
	t.Setenv("OVAV_FISH_ROOT", canonical)
	got, err := fishCanonicalRoot()
	if err != nil {
		t.Fatalf("fishCanonicalRoot: %v", err)
	}
	if got != canonical {
		t.Errorf("got %q, want %q (env override)", got, canonical)
	}
}

func TestListFishFiles_FiltersExtras(t *testing.T) {
	canonical, _, cleanup := fixtureFish(t, []string{
		"05-a.fish", "20-b.fish", "99-c.fish", "config.fish",
		"fish_prompt.fish", "ovav.fish",
	})
	defer cleanup()
	files, err := listFishFiles(canonical)
	if err != nil {
		t.Fatalf("listFishFiles: %v", err)
	}
	// All *.fish (top-level) are deployable. README/metadata excluded by extension.
	if len(files) != 6 {
		t.Errorf("expected 6 deployable files, got %d: %v", len(files), files)
	}
	for _, f := range files {
		if filepath.Ext(f) != ".fish" {
			t.Errorf("non-fish in result: %q", f)
		}
	}
}

func TestListFishFiles_ExcludesNonFishFiles(t *testing.T) {
	canonical, _, cleanup := fixtureFish(t, []string{"05-a.fish"})
	defer cleanup()
	// Add README.md, metadata.yaml, tests/ — none should appear.
	if err := os.WriteFile(filepath.Join(canonical, "config", "fish", "README.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(canonical, "config", "fish", "metadata.yaml"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(canonical, "config", "fish", "tests"), 0o755); err != nil {
		t.Fatal(err)
	}
	files, err := listFishFiles(canonical)
	if err != nil {
		t.Fatalf("listFishFiles: %v", err)
	}
	// fixtureFish injects ovav.fish as a canonical marker; expect 2 (.fish
	// entries) — only the .fish files. README, metadata, and tests/ excluded.
	if len(files) != 2 {
		t.Errorf("expected 2 deployable files (05-a.fish + ovav.fish), got %d: %v", len(files), files)
	}
	names := make(map[string]bool, len(files))
	for _, f := range files {
		names[f] = true
	}
	if !names["05-a.fish"] {
		t.Errorf("05-a.fish missing from result: %v", files)
	}
	if !names["ovav.fish"] {
		t.Errorf("ovav.fish missing from result: %v", files)
	}
}

func TestInspectFishFiles_AllMissing(t *testing.T) {
	canonical, userConfig, cleanup := fixtureFish(t, []string{
		"05-a.fish", "20-b.fish",
	})
	defer cleanup()
	states, err := inspectFishFiles(canonical, userConfig)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	// fixtureFish injects ovav.fish + the two we asked for = 3 states.
	if len(states) != 3 {
		t.Fatalf("expected 3 inspect states, got %d", len(states))
	}
	for _, s := range states {
		if s.State != "missing" {
			t.Errorf("file %s: expected missing, got %s", filepath.Base(s.Src), s.State)
		}
		if s.DstSHA != "" {
			t.Errorf("missing file should have empty DstSHA: %s", s.DstSHA)
		}
	}
}

func TestInspectFishFiles_Match(t *testing.T) {
	canonical, userConfig, cleanup := fixtureFish(t, []string{"05-a.fish"})
	defer cleanup()
	// Copy 05-a.fish into the user side so hashes match.
	src := filepath.Join(canonical, "config", "fish", "05-a.fish")
	dst := filepath.Join(userConfig, "conf.d", "05-a.fish")
	if err := fishCopyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	states, err := inspectFishFiles(canonical, userConfig)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	// Find the 05-a.fish state and verify it's match.
	found := false
	for _, s := range states {
		if filepath.Base(s.Src) == "05-a.fish" {
			found = true
			if s.State != "match" {
				t.Errorf("05-a.fish: expected match, got %s", s.State)
			}
		}
	}
	if !found {
		t.Errorf("did not find 05-a.fish in states: %+v", states)
	}
}

func TestInspectFishFiles_Drift(t *testing.T) {
	canonical, userConfig, cleanup := fixtureFish(t, []string{"05-a.fish"})
	defer cleanup()
	// Write a different file to dst so hashes differ.
	dst := filepath.Join(userConfig, "conf.d", "05-a.fish")
	if err := os.WriteFile(dst, []byte("different content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	states, err := inspectFishFiles(canonical, userConfig)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	found := false
	for _, s := range states {
		if filepath.Base(s.Src) == "05-a.fish" {
			found = true
			if s.State != "drift" {
				t.Errorf("05-a.fish: expected drift, got %s", s.State)
			}
		}
	}
	if !found {
		t.Errorf("did not find 05-a.fish in states: %+v", states)
	}
}

func TestFishTargetFor_TopLevelVsConfD(t *testing.T) {
	userConfigDir := "/home/test/.config/fish"
	cases := map[string]string{
		"config.fish":                "/home/test/.config/fish/config.fish",
		"fish_prompt.fish":           "/home/test/.config/fish/fish_prompt.fish",
		"ovav.fish":                  "/home/test/.config/fish/ovav.fish",
		"05-ovav-tmux-session.fish":  "/home/test/.config/fish/conf.d/05-ovav-tmux-session.fish",
		"90-ovav-terminal-auto.fish": "/home/test/.config/fish/conf.d/90-ovav-terminal-auto.fish",
	}
	for in, want := range cases {
		got := fishTargetFor(userConfigDir, in)
		if got != want {
			t.Errorf("fishTargetFor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFishCopyFile_Basic(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.txt")
	dst := filepath.Join(tmp, "dst.txt")
	content := "hello fish copy\n"
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fishCopyFile(src, dst); err != nil {
		t.Fatalf("copy: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != content {
		t.Errorf("dst content mismatch: got %q want %q", got, content)
	}
}

func TestShortHashFile_Deterministic(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "file.txt")
	if err := os.WriteFile(p, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}
	h1, n1, err := shortHashFile(p)
	if err != nil {
		t.Fatal(err)
	}
	h2, n2, err := shortHashFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 || h1 == "" || len(h1) != 16 {
		t.Errorf("hashes not deterministic / wrong length: h1=%q h2=%q len=%d", h1, h2, len(h1))
	}
	if n1 != n2 || n1 != 11 {
		t.Errorf("byte counts differ: %d vs %d", n1, n2)
	}
}

func TestShortHashFile_MissingFileEmpty(t *testing.T) {
	h, n, err := shortHashFile("/nonexistent/file.fish")
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if h != "" || n != 0 {
		t.Errorf("expected empty hash + zero bytes, got h=%q n=%d", h, n)
	}
}

func TestFishUserConfigDir_UsesXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/custom/xdg")
	t.Setenv("HOME", "/home/whatever")
	got := fishUserConfigDir()
	want := "/custom/xdg/fish"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestFishUserConfigDir_FallsBackToHOME(t *testing.T) {
	// Unset XDG_CONFIG_HOME so the function uses $HOME/.config.
	oldXdg := os.Getenv("XDG_CONFIG_HOME")
	os.Unsetenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", oldXdg)
	t.Setenv("HOME", "/home/tester")
	got := fishUserConfigDir()
	want := "/home/tester/.config/fish"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestCmdFish_HelpExitZero(t *testing.T) {
	if code := cmdFish([]string{"--help"}); code != 0 {
		t.Errorf("help exit code = %d, want 0", code)
	}
}

func TestCmdFish_UnknownSubReturnsTwo(t *testing.T) {
	if code := cmdFish([]string{"bogus-sub"}); code != 2 {
		t.Errorf("unknown sub exit code = %d, want 2", code)
	}
}
