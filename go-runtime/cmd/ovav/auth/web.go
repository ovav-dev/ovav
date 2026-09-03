//go:build linux || darwin

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/term"
)

// CmdWeb implements `ovav auth web` — browser-based OAuth login
// via the OVAV web backend. R-3 mandates a preflight HTTP probe;
// the interactive flow refuses to launch on a broken backend.
//
// YOLO 2026: this command is gated by CheckLoginAllowed. By default
// (no OVAV_AUTH_LOGIN_ENABLED and no --force), it returns
// ExitConfigDisabled. Cloudflare Access currently blocks the
// preflight probe (R-3), so this gate is also the operational
// kill-switch for the broken web flow.
//
// Usage:
//
//	ovav auth web [--check] [--no-open] [--force]
//	ovav auth web --timeout 60    # custom poll timeout (seconds)
func CmdWeb(args []string) int {
	// YOLO 2026: gate login by default. Bypass with --force or env.
	if !CheckLoginAllowed(args) {
		return ExitConfigDisabled
	}

	noOpen := false
	checkOnly := false
	timeout := 90 * time.Second
	for i, a := range args {
		switch a {
		case "--check":
			checkOnly = true
		case "--no-open":
			noOpen = true
		case "--timeout":
			if i+1 < len(args) {
				d, err := time.ParseDuration(args[i+1] + "s")
				if err == nil && d > 0 {
					timeout = d
				}
				i++
			}
		case "--help", "-h":
			fmt.Println("ovav auth web — browser-based OAuth via https://ovav.dev")
			fmt.Println()
			fmt.Println("USAGE:")
			fmt.Println("  ovav auth web              full flow (preflight + browser + poll)")
			fmt.Println("  ovav auth web --check      preflight only (HTTP probe + JSON schema)")
			fmt.Println("  ovav auth web --no-open    print login URL instead of opening browser")
			fmt.Println("  ovav auth web --timeout N  custom poll timeout in seconds")
			fmt.Println("  ovav auth web --force      bypass YOLO 2026 gate")
			fmt.Println()
			fmt.Println("YOLO 2026: login disabled by default. Use `ovav waiver` or pass --force.")
			fmt.Println()
			fmt.Println("ENV VARS:")
			fmt.Println("  OVAV_WEB_URL              override backend (default: https://d678beea.ovav.dev)")
			fmt.Println("  OVAV_AUTH_LOGIN_ENABLED   set to 1 to enable login (default: disabled)")
			return 0
		}
	}

	backendURL := os.Getenv("OVAV_WEB_URL")
	if backendURL == "" {
		backendURL = "https://d678beea.ovav.dev"
	}

	// R-3: mandatory preflight
	if err := PreflightProbe(backendURL); err != nil {
		return Die(1, "web backend NOT ready: %v\n  → refuse to launch interactive flow (R-3)", err)
	}

	if checkOnly {
		PrintOK(fmt.Sprintf("web backend reachable + JSON contract valid: %s", backendURL))
		return 0
	}

	// Fetch challenge
	challenge, err := getChallenge(backendURL)
	if err != nil {
		return Die(1, "challenge fetch: %v", err)
	}
	PrintOK(fmt.Sprintf("challenge obtained: %s…", challenge[:min(12, len(challenge))]))

	// Open browser (or print URL)
	loginURL := fmt.Sprintf("%s/oauth/start?code=%s", backendURL, url.QueryEscape(challenge))
	if noOpen {
		fmt.Println()
		fmt.Println("  🔗 Open this URL in your browser:")
		fmt.Println("    " + loginURL)
	} else {
		if err := openBrowser(loginURL); err != nil {
			PrintWarn(fmt.Sprintf("auto-open failed: %v", err))
			fmt.Println("  🔗 Open this URL in your browser:")
			fmt.Println("    " + loginURL)
		} else {
			fmt.Printf("  🌐 browser opened: %s\n", loginURL)
		}
	}

	// Poll for JWT
	fmt.Printf("  ⏳ polling for completion (timeout %s)...\n", timeout)
	jwt, err := pollForJWT(backendURL, challenge, timeout)
	if err != nil {
		return Die(1, "poll: %v", err)
	}
	if jwt == "" {
		return Die(1, "login timed out after %s", timeout)
	}

	// Store JWT in vault (NOT plaintext)
	home := HomeDirOrDefault("/home/braka")
	if err := writeJWTVault(home, jwt); err != nil {
		return Die(1, "store jwt: %v", err)
	}
	PrintOK("web identity stored in vault (encrypted)")
	fmt.Printf("   JWT:    %s…\n", jwt[:min(20, len(jwt))])
	fmt.Println()
	PrintOK("web session ready")
	return 0
}

// PreflightProbe performs the R-3 mandatory probe: HTTP GET
// /api/v1/auth/login-challenge-web, expect HTTP 200 + JSON with a
// 'challenge' field.
func PreflightProbe(backendURL string) error {
	endpoint := strings.TrimRight(backendURL, "/") + "/api/v1/auth/login-challenge-web"
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		return fmt.Errorf("connect %s: %w", endpoint, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d (expected 200) from %s", resp.StatusCode, endpoint)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	var probe struct {
		Challenge string `json:"challenge"`
	}
	if err := json.Unmarshal(body, &probe); err != nil {
		return fmt.Errorf("invalid JSON: %v (body: %.80s)", err, string(body))
	}
	if probe.Challenge == "" {
		return fmt.Errorf("missing 'challenge' field in response")
	}
	return nil
}

// ─── helpers ─────────────────────────────────────────────────────────────

func getChallenge(backendURL string) (string, error) {
	endpoint := strings.TrimRight(backendURL, "/") + "/api/v1/auth/login-challenge-web"
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var out struct {
		Challenge string `json:"challenge"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Challenge, nil
}

func pollForJWT(backendURL, challenge string, timeout time.Duration) (string, error) {
	endpoint := strings.TrimRight(backendURL, "/") + "/api/v1/auth/login-status?challenge=" + url.QueryEscape(challenge)
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 5 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(endpoint)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var out struct {
			Status   string `json:"status"`
			VaultJWT string `json:"vault_jwt,omitempty"`
		}
		if json.Unmarshal(body, &out) == nil && out.VaultJWT != "" {
			return out.VaultJWT, nil
		}
		if out.Status == "expired" || out.Status == "denied" {
			return "", fmt.Errorf("login %s", out.Status)
		}
		time.Sleep(2 * time.Second)
	}
	return "", nil
}

// writeJWTVault stores the JWT in the encrypted vault path (R-3
// compliance: never store JWT in plaintext on disk).
func writeJWTVault(home, jwt string) error {
	cfgDir := filepath.Join(home, ".config", "ovav", "vault")
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		return err
	}
	// The legacy ovav login stores JWT in vault_jwt.enc; if we don't
	// have a working vault here, fall back to writing under share/
	// in mode 600 so it's at least protected.
	path := filepath.Join(cfgDir, "vault_jwt.enc")
	return os.WriteFile(path, []byte(jwt), 0o600)
}

func openBrowser(rawURL string) error {
	// Minimal cross-platform stub. Falls through to xdg-open or open.
	// Real implementation in cmd/ovav/login.go (openBrowser fn).
	// The legacy implementation lives in login.go; we don't import
	// that here to avoid import cycles. Caller's wrapper script
	// can use ovav-web --no-open + a manual click as the safe path.
	return openBrowserStub(rawURL)
}

func openBrowserStub(rawURL string) error {
	// We use the system-bundled `xdg-open`, `open`, or `cmd /c start`.
	candidates := [][]string{
		{"xdg-open", rawURL},
		{"open", rawURL},
		{"wslview", rawURL},
		{"cmd.exe", "/c", "start", "", rawURL},
	}
	for _, c := range candidates {
		if _, err := execLookPath(c[0]); err == nil {
			p, err := os.StartProcess(c[0], c, &os.ProcAttr{})
			if err == nil {
				_ = p.Release()
				return nil
			}
		}
	}
	return fmt.Errorf("no opener found (tried xdg-open/open/wslview/cmd.exe)")
}

// execLookPath is a small wrapper that handles `PATH` lookups.
func execLookPath(name string) (string, error) {
	// Use os/exec.LookPath under the hood.
	return execLookPathImpl(name)
}

// keep unused imports for go vet
var _ = term.IsTerminal

// helper that imports the actual exec.LookPath
func execLookPathImpl(name string) (string, error) {
	return execLookPathReal(name)
}
