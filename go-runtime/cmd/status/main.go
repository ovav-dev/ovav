// OVAV Status — compact auth state snapshot for scripts and dashboards.
//
// Output format:
//
//	{
//	  "status": "ok",
//	  "auth": {
//	    "mode": "LOGIN",
//	    "level": "CEO",
//	    "ttl_remaining": "47m",
//	    "gates": ["git_push","git_merge","protected_branch",...]
//	  }
//	}
//
// Falls back to NO_LOGIN if the session cannot be loaded.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/auth"
	"github.com/ovav/ovav/internal/ceo"
)

// StatusOutput is the JSON output schema for `ovav status`.
type StatusOutput struct {
	Status string   `json:"status"`
	Auth   AuthInfo `json:"auth"`
}

// AuthInfo carries the Login System v2 auth block.
type AuthInfo struct {
	Mode         string   `json:"mode"`
	Level        string   `json:"level"`
	TTLRemaining string   `json:"ttl_remaining"`
	Gates        []string `json:"gates"`
}

func main() {
	jsonFlag := flag.Bool("json", false, "JSON output (default)")
	flag.Parse()

	repoRoot, err := detectRepoRoot()
	if err != nil {
		emitJSON(StatusOutput{
			Status: "error",
			Auth:   noLoginAuth(),
		})
		return
	}

	output := buildStatus(repoRoot)

	if *jsonFlag || !atty() {
		emitJSON(output)
		return
	}

	// Human-readable fallback
	printHuman(output)
}

func emitJSON(o StatusOutput) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	_ = enc.Encode(o)
}

func atty() bool {
	return false // always JSON unless explicitly suppressed
}

func buildStatus(repoRoot string) StatusOutput {
	authInfo := buildAuthInfo(repoRoot)

	status := "ok"
	if authInfo.Mode == "NO_LOGIN" {
		status = "degraded"
	}

	return StatusOutput{
		Status: status,
		Auth:   authInfo,
	}
}

func buildAuthInfo(repoRoot string) AuthInfo {
	sess, err := ceo.Load(repoRoot)
	if err != nil || sess == nil || !sess.Valid() {
		return noLoginAuth()
	}

	expiry, err := time.Parse(time.RFC3339, sess.ExpiresAt)
	if err != nil {
		return noLoginAuth()
	}
	remaining := time.Until(expiry)
	if remaining < 0 {
		remaining = 0
	}

	level := "CEO"
	if sess.Operator != "" && sess.Operator != "ceo-alexander" {
		level = "USER"
	}

	// Build gate list using auth package
	authState := auth.NewAuthState()
	if sess.Valid() {
		authState.Inject(auth.ModeLOGIN)
	} else {
		authState.Inject(auth.ModeNOLOGIN)
	}
	gates := authState.Injector().GetAllGates()

	return AuthInfo{
		Mode:         "LOGIN",
		Level:        level,
		TTLRemaining: formatDuration(remaining),
		Gates:        gates,
	}
}

func noLoginAuth() AuthInfo {
	authState := auth.NewAuthState()
	authState.Inject(auth.ModeNOLOGIN)
	return AuthInfo{
		Mode:         "NO_LOGIN",
		Level:        "ANONYMOUS",
		TTLRemaining: "0m",
		Gates:        authState.Injector().GetAllGates(),
	}
}

func printHuman(o StatusOutput) {
	a := o.Auth
	if a.Mode == "LOGIN" {
		fmt.Printf("🔓 LOGIN  | %s | TTL %s\n", a.Level, a.TTLRemaining)
	} else {
		fmt.Printf("🔒 NO_LOGIN | %s | Operaciones guiadas\n", a.Level)
	}
	fmt.Printf("Gates: %s\n", strings.Join(a.Gates, ", "))
}

func detectRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		gitPath := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitPath); err == nil {
			if info.IsDir() {
				return dir, nil
			}
			if data, err := os.ReadFile(gitPath); err == nil {
				line := strings.TrimSpace(string(data))
				if strings.HasPrefix(line, "gitdir: ") {
					return dir, nil
				}
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return cwd, fmt.Errorf("not in a git repository")
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "0m"
	}
	d = d.Round(time.Minute)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 && m > 0 {
		return fmt.Sprintf("%dh%dm", h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dm", m)
}
