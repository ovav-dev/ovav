// fish_cli.go — `ovav fish` sub-command: manage OVAV fish config deployment.
//
// Sub-commands:
//
//	ovav fish status   show drift between repo canonical and ~/~/.config/fish
//	ovav fish sync     deploy config/fish/*.fish to ~/.config/fish/conf.d/
//	ovav fish diff     print per-file drift report (no writes)
//	ovav fish list     list canonical modules + their deployment state
//
// All paths are auto-discovered from repoRoot. Designed to run from a
// worktree (uses FindRepoRoot to walk up until .ovav/ is found) or from
// the main repo path.

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ovav/ovav/internal/cli"
)

// fishFileState describes one canonical fish file and its deployment state.
type fishFileState struct {
	Src      string `json:"src"`       // absolute path to canonical source
	SrcSHA   string `json:"src_sha"`   // sha256(short) of source
	SrcBytes int64  `json:"src_bytes"` // size of source
	Dst      string `json:"dst"`       // absolute path at target
	DstSHA   string `json:"dst_sha"`   // sha256(short) of deployed, or empty
	State    string `json:"state"`     // "match" | "drift" | "missing" | "extra"
}

func cmdFish(args []string) int {
	if len(args) == 0 {
		printFishHelp()
		return 0
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "status":
		return cmdFishStatus(rest)
	case "sync", "deploy":
		return cmdFishSync(rest)
	case "diff":
		return cmdFishDiff(rest)
	case "list", "ls":
		return cmdFishList(rest)
	case "--help", "-h", "help":
		printFishHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "ovav fish: unknown sub-command %q\n", sub)
		printFishHelp()
		return 2
	}
}

func printFishHelp() {
	fmt.Print(`ovav fish — Manage OVAV fish shell config deployment

Usage:
  ovav fish status   Show drift count (no writes)
  ovav fish diff     Show per-file drift report (no writes)
  ovav fish list     List canonical modules + deployment state
  ovav fish sync     Deploy config/fish/*.fish → ~/.config/fish/conf.d/

Flags (sync):
  --dry-run   print what would happen, do nothing
  --force     overwrite even existing user files (default: backup first)

Why this exists:
  Without it, every new feature in config/fish/*.fish is invisible to the
  user until they manually copy. The reload command also won't appear until
  the config file is in the user's conf.d/.

Examples:
  ovav fish status        # quick drift count
  ovav fish sync          # deploy everything
  ovav fish sync --dry-run  # preview only
`)
}

// fishCanonicalRoot discovers the canonical fish config root.
// Priority:
//  1. Walk up from cwd looking for a /config/fish/ovav.fish marker (in-repo case)
//  2. cli.FindRepoRoot (handles worktrees correctly via .ovav/ discovery)
//  3. $OVAV_FISH_ROOT env override (for tooling that runs from elsewhere)
func fishCanonicalRoot() (string, error) {
	if v := os.Getenv("OVAV_FISH_ROOT"); v != "" {
		if _, err := os.Stat(filepath.Join(v, "config", "fish", "ovav.fish")); err == nil {
			return v, nil
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		dir := cwd
		for i := 0; i < 8; i++ {
			if _, err := os.Stat(filepath.Join(dir, "config", "fish", "ovav.fish")); err == nil {
				return dir, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	if root, err := cli.FindRepoRoot(); err == nil {
		return root, nil
	}
	return "", fmt.Errorf("cannot locate canonical fish config; set OVAV_FISH_ROOT or run from inside the OVAV repo")
}

// fishUserConfD returns the user's fish conf.d directory.
func fishUserConfD() string {
	return filepath.Join(fishUserConfigDir(), "conf.d")
}

// fishUserConfigDir returns the user's ~/.config/fish directory.
func fishUserConfigDir() string {
	confHome := os.Getenv("XDG_CONFIG_HOME")
	if confHome == "" {
		confHome = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(confHome, "fish")
}

// fishTargetFor returns the destination path for a canonical fish file.
// Files that are top-level (config.fish, fish_prompt.fish, ovav.fish) go
// directly to ~/.config/fish/; everything else goes to ~/.config/fish/conf.d/.
func fishTargetFor(userConfigDir, fname string) string {
	switch fname {
	case "config.fish", "fish_prompt.fish", "ovav.fish":
		return filepath.Join(userConfigDir, fname)
	default:
		return filepath.Join(userConfigDir, "conf.d", fname)
	}
}

func listFishFiles(canonicalRoot string) ([]string, error) {
	fishDir := filepath.Join(canonicalRoot, "config", "fish")
	entries, err := os.ReadDir(fishDir)
	if err != nil {
		return nil, fmt.Errorf("ovav fish: read %s: %w", fishDir, err)
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Deploy only what belongs in conf.d.
		// Excluded:
		//   - config.fish          (it lives at ~/.config/fish/config.fish, not in conf.d/)
		//   - ovav.fish            (root aggregator, already source via config.fish)
		//   - fish_prompt.fish     (lives at ~/.config/fish/, not conf.d/)
		//   - *.example            (template, not for live config)
		//   - *.yaml / *.md / *.txt (docs, not fish source)
		//   - tests/ subdirectory  (already handled by IsDir above)
		if name != "config.fish" &&
			name != "ovav.fish" &&
			name != "fish_prompt.fish" &&
			filepath.Ext(name) != ".fish" {
			continue
		}
		if filepath.Ext(name) == ".example" {
			continue
		}
		files = append(files, name)
	}
	return files, nil
}

func shortHashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", 0, nil
		}
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", n, err
	}
	return hex.EncodeToString(h.Sum(nil))[:16], n, nil
}

func inspectFishFiles(canonicalRoot, userConfigDir string) ([]fishFileState, error) {
	files, err := listFishFiles(canonicalRoot)
	if err != nil {
		return nil, err
	}
	out := make([]fishFileState, 0, len(files))
	for _, fname := range files {
		src := filepath.Join(canonicalRoot, "config", "fish", fname)
		dst := fishTargetFor(userConfigDir, fname)
		sSHA, sBytes, err := shortHashFile(src)
		if err != nil {
			return nil, fmt.Errorf("hash source %s: %w", src, err)
		}
		dSHA, _, err := shortHashFile(dst)
		if err != nil {
			return nil, fmt.Errorf("hash dest %s: %w", dst, err)
		}
		state := "match"
		if dSHA == "" {
			state = "missing"
		} else if dSHA != sSHA {
			state = "drift"
		}
		out = append(out, fishFileState{
			Src:      src,
			SrcSHA:   sSHA,
			SrcBytes: sBytes,
			Dst:      dst,
			DstSHA:   dSHA,
			State:    state,
		})
	}
	return out, nil
}

func cmdFishStatus(args []string) int {
	canonicalRoot, err := fishCanonicalRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ ovav fish: cannot locate canonical repo: %v\n", err)
		return 1
	}
	userConfigDir := fishUserConfigDir()
	states, err := inspectFishFiles(canonicalRoot, userConfigDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ ovav fish: %v\n", err)
		return 1
	}

	drift := 0
	missing := 0
	match := 0
	for _, s := range states {
		switch s.State {
		case "drift":
			drift++
		case "missing":
			missing++
		case "match":
			match++
		}
	}

	jsonOut := cli.HasJSONFlag(args)
	if jsonOut {
		result := map[string]any{
			"canonical_root": canonicalRoot,
			"target":         userConfigDir,
			"total":          len(states),
			"match":          match,
			"drift":          drift,
			"missing":        missing,
			"files":          states,
		}
		// minimal json output: just write the map
		b, _ := jsonMarshal(result)
		fmt.Println(string(b))
		return 0
	}

	fmt.Println("OVAV Fish Drift Report")
	fmt.Printf("   canonical : %s\n", canonicalRoot)
	fmt.Printf("   target    : %s\n", userConfigDir)
	fmt.Printf("   modules   : %d   match=%d   drift=%d   missing=%d\n",
		len(states), match, drift, missing)
	if drift+missing > 0 {
		fmt.Println()
		fmt.Println("   Run `ovav fish sync` to deploy.")
	}
	return 0
}

func cmdFishDiff(args []string) int {
	canonicalRoot, err := fishCanonicalRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ ovav fish: %v\n", err)
		return 1
	}
	userConfigDir := fishUserConfigDir()
	states, err := inspectFishFiles(canonicalRoot, userConfigDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ ovav fish: %v\n", err)
		return 1
	}
	for _, s := range states {
		switch s.State {
		case "match":
			fmt.Printf("  [ok]      %-32s  %s\n", filepath.Base(s.Src), s.SrcSHA[:12])
		case "drift":
			fmt.Printf("  [drift]   %-32s  src=%s dst=%s\n",
				filepath.Base(s.Src), s.SrcSHA[:12], s.DstSHA[:12])
		case "missing":
			fmt.Printf("  [miss]    %-32s  (not in %s)\n",
				filepath.Base(s.Src), userConfigDir)
		}
	}
	return 0
}

func cmdFishList(args []string) int {
	canonicalRoot, err := fishCanonicalRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ ovav fish: %v\n", err)
		return 1
	}
	userConfigDir := fishUserConfigDir()
	states, err := inspectFishFiles(canonicalRoot, userConfigDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ ovav fish: %v\n", err)
		return 1
	}
	fmt.Printf("OVAV Fish Canonical Modules (%d)\n", len(states))
	fmt.Printf("   location: %s\n", filepath.Join(canonicalRoot, "config", "fish"))
	fmt.Printf("   target  : %s\n", userConfigDir)
	fmt.Println()
	for _, s := range states {
		fmt.Printf("  %-32s  %5d B   [%s]\n",
			filepath.Base(s.Src), s.SrcBytes, s.State)
	}
	return 0
}

func cmdFishSync(args []string) int {
	dryRun := false
	force := false
	for _, a := range args {
		switch a {
		case "--dry-run":
			dryRun = true
		case "--force":
			force = true
		}
	}
	canonicalRoot, err := fishCanonicalRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ ovav fish: %v\n", err)
		return 1
	}
	userConfigDir := fishUserConfigDir()
	if err := os.MkdirAll(filepath.Join(userConfigDir, "conf.d"), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "❌ ovav fish: mkdir %s: %v\n", userConfigDir, err)
		return 1
	}
	states, err := inspectFishFiles(canonicalRoot, userConfigDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ ovav fish: %v\n", err)
		return 1
	}
	deployed := 0
	driftKept := 0
	planned := 0
	for _, s := range states {
		switch s.State {
		case "match":
			continue // already ok
		case "missing", "drift":
			if dryRun {
				fmt.Printf("  [plan]   %s (%s)\n", filepath.Base(s.Src), s.State)
				planned++
				continue
			}
			// Back up any existing dst before overwriting (unless --force).
			if s.State == "drift" && !force {
				bak := s.Dst + ".bak-" + s.SrcSHA[:7]
				if err := fishCopyFile(s.Dst, bak); err != nil {
					fmt.Fprintf(os.Stderr, "⚠️  backup failed %s: %v\n", s.Dst, err)
				} else {
					fmt.Printf("  [bak]    %s → %s\n", s.Dst, bak)
				}
			}
			if err := fishCopyFile(s.Src, s.Dst); err != nil {
				fmt.Fprintf(os.Stderr, "❌ deploy %s: %v\n", s.Src, err)
				driftKept++
				continue
			}
			fmt.Printf("  [ok]     %s → %s\n", s.Src, s.Dst)
			deployed++
		}
	}
	fmt.Println()
	if dryRun {
		fmt.Printf("plan: %d would deploy, %d match already, %d drift (no writes)\n", planned, len(states)-planned, driftKept)
	} else {
		fmt.Printf("done: %d deployed, %d failed, %d matched\n", deployed, driftKept, len(states)-deployed-driftKept)
		fmt.Println()
		fmt.Println("Reload your shell to activate: fish -c 'reload'")
		fmt.Println("   (the 'reload' function itself lives in 45-ovav-reload.fish)")
	}
	if driftKept > 0 {
		return 1
	}
	return 0
}

func fishCopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// jsonMarshal is a minimal wrapper so we don't import encoding/json in
// the simple hot path. Avoids bloating the binary when CLI isn't used.
func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}
