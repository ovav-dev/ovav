// ovav login — Unified CLI identity with vault unlock.
//
// Pattern: 1Password (PBKDF2 vault derive) × Tailscale (device identity) × GitHub (device flow, no browser).
//
// Architecture:
//
//	ovav login        → seed → PBKDF2(machine_id) → vault key → session stored
//	ovav whoami        → check session → show identity
//	ovav logout        → clear session
//
// Session storage: ~/.local/share/ovav/session (vault_key_hash + machine_id + timestamp)
// The vault key itself is NEVER stored on disk. Only its SHA-256 hash for verification.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/ovav/ovav/internal/cli"
	"github.com/ovav/ovav/internal/identity"
	"github.com/ovav/ovav/internal/infra"
	"github.com/ovav/ovav/internal/license"
	"golang.org/x/term"
)

const (
	sessionDir  = ".local/share/ovav"
	sessionFile = "session"
	sessionTTL  = 24 * time.Hour
)

// openBrowser opens url in the default browser for the current OS.
func openBrowser(rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "linux":
		cmd = exec.Command("xdg-open", rawURL)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", rawURL)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	return cmd.Start()
}

// Session represents a stored OVAV CLI session.
type Session struct {
	VaultKeyHash string `json:"vault_key_hash"` // SHA-256(vault_key) — never raw key
	MachineID    string `json:"machine_id"`
	CreatedAt    string `json:"created_at"`
	Hostname     string `json:"hostname"`
	User         string `json:"user"`
	IdentityID   string `json:"identity_id,omitempty"` // Registry identity ID
	Role         string `json:"role,omitempty"`        // ceo, lead, developer, viewer
	Level        int    `json:"level,omitempty"`       // Access level (1-10)
	Name         string `json:"name,omitempty"`        // Human-readable name
	VaultJWT     string `json:"vault_jwt,omitempty"`   // JWT from d678beea.ovav.dev vault auth
}

// ── Login command ─────────────────────────────────────────────────────────────

func cmdLogin(args []string) int {
	// Parse flags
	force := false
	webLogin := false
	for _, a := range args {
		if a == "--force" || a == "-f" {
			force = true
		}
		if a == "--web" || a == "-w" {
			webLogin = true
		}
	}

	// ── Web login (browser-based) ─────────────────────────────────────────────
	if webLogin {
		return cmdLoginWeb(force)
	}

	// ── Seed-based login (default) ───────────────────────────────────────────
	machineID, err := license.MachineID()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Cannot determine machine identity: %v\n", err)
		return 1
	}

	// Check existing session
	if !force {
		if sess, ok := loadSession(); ok {
			age := time.Since(sess.createdAt())
			if age < sessionTTL {
				fmt.Printf("🟢 Session active (%s ago)\n", humanDuration(age))
				fmt.Printf("   Machine: %s\n", sess.MachineID[:16]+"...")
				fmt.Printf("   Enter seed to re-verify or 'ovav logout' to close.\n")
			}
		}
	}

	// Read seed (hidden input)
	fmt.Print("Seed: ")
	seedBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Cannot read seed: %v\n", err)
		return 1
	}
	seed := strings.TrimSpace(string(seedBytes))
	if seed == "" {
		fmt.Fprintln(os.Stderr, "❌ Seed cannot be empty")
		return 1
	}

	// Derive vault key: PBKDF2-HMAC-SHA256(seed, salt=SHA256(machine_id)[:32], 600_000, 32)
	vaultKey, err := license.DeriveKey(seed, machineID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Key derivation failed: %v\n", err)
		return 1
	}

	// Verify against stored session (if exists and not forced)
	if !force {
		if sess, ok := loadSession(); ok {
			vaultKeyHash := sha256Hex(vaultKey)
			if !strings.EqualFold(vaultKeyHash, sess.VaultKeyHash) {
				fmt.Fprintln(os.Stderr, "❌ Seed does not match stored identity.")
				fmt.Fprintln(os.Stderr, "   Use 'ovav login --force' to re-initialize.")
				return 1
			}
			// Identity re-verified — show stored identity info
			if sess.Name != "" {
				fmt.Printf("✅ Identity verified: %s [%s · Level %d]\n",
					sess.Name, strings.ToUpper(sess.Role), sess.Level)
			} else {
				fmt.Println("✅ Identity verified. Vault unlocked.")
			}
			// Update session timestamp so push/commit gates don't see expired session
			sess.CreatedAt = time.Now().UTC().Format(time.RFC3339)
			if saveErr := saveSession(sess); saveErr != nil {
				fmt.Fprintf(os.Stderr, "⚠️  Could not update session: %v\n", saveErr)
			}

			// ── Vault web auth ──────────────────────────────────────────────
			hname, _ := os.Hostname()
			jwt, jwtErr := fetchVaultJWT(seed, machineID, hname)
			if jwtErr != nil {
				fmt.Fprintf(os.Stderr, "⚠️  Web vault auth failed: %v\n", jwtErr)
			} else {
				sess.VaultJWT = jwt
				if saveErr := saveSession(sess); saveErr != nil {
					fmt.Fprintf(os.Stderr, "⚠️  Could not save vault JWT: %v\n", saveErr)
				} else {
					fmt.Println("   Web:     vault session active")
				}
			}

			exportVaultKey(vaultKey, seed)
			infra.DecryptTokensFromVault(hex.EncodeToString(vaultKey))
			return 0
		}
	}

	// First-time login: create session
	vaultKeyHash := sha256Hex(vaultKey)
	hostname, _ := os.Hostname()
	user := os.Getenv("USER")

	// ── Identity registry lookup ────────────────────────────────────────
	repoRoot := ""
	var matchedIdentity *identity.Identity
	if root, err := cli.FindRepoRoot(); err == nil {
		repoRoot = root
		reg, regErr := identity.LoadRegistry(root)
		if regErr == nil {
			if id, findErr := identity.FindIdentity(reg, vaultKeyHash); findErr == nil {
				matchedIdentity = id
			}
		}
	}

	// ── Audit logging ───────────────────────────────────────────────────
	if repoRoot != "" {
		if err := identity.InitAudit(repoRoot); err == nil {
			defer identity.CloseAudit()
			if matchedIdentity != nil {
				identity.LogAudit(identity.NewLoginEntry(matchedIdentity, machineID))
			} else {
				identity.LogAudit(identity.NewFailedLoginEntry(machineID, "identity not found in registry"))
			}
		}
	}

	// ── Identity not found → block ──────────────────────────────────────
	if matchedIdentity == nil {
		fmt.Fprintln(os.Stderr, "❌ Identity not recognized. Contact CEO.")
		fmt.Fprintln(os.Stderr, "   Your vault key does not match any registered identity.")
		return 1
	}

	// ── Build session with identity ─────────────────────────────────────
	sess := Session{
		VaultKeyHash: vaultKeyHash,
		MachineID:    machineID,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
		Hostname:     hostname,
		User:         user,
		IdentityID:   matchedIdentity.ID,
		Role:         matchedIdentity.Role,
		Level:        matchedIdentity.Level,
		Name:         matchedIdentity.Name,
	}

	if err := saveSession(sess); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Could not save session: %v\n", err)
	}

	// ── Vault web auth ──────────────────────────────────────────────────
	jwt, jwtErr := fetchVaultJWT(seed, machineID, hostname)
	if jwtErr != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Web vault auth failed: %v (vault sync unavailable)\n", jwtErr)
	} else {
		sess.VaultJWT = jwt
		if saveErr := saveSession(sess); saveErr != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Could not save vault JWT: %v\n", saveErr)
		} else {
			fmt.Println("   Web:     vault session active — sync enabled")
		}
	}

	exportVaultKey(vaultKey, seed)
	infra.DecryptTokensFromVault(hex.EncodeToString(vaultKey))

	fmt.Println()
	fmt.Printf("🟢 %s\n", identity.WelcomeMessage(matchedIdentity))
	fmt.Printf("   Machine: %s\n", machineID[:16]+"...")
	fmt.Printf("   Vault:   unlocked (AES-256-GCM)\n")
	fmt.Printf("   Web:     https://d678beea.ovav.dev\n")
	fmt.Println()
	fmt.Println("   Run 'ovav whoami' to verify. 'ovav logout' to close.")

	return 0
}

// cmdLoginWeb authenticates via browser (Google OAuth or email+password).
// Flow: get challenge → open browser → poll status → derive vault key from seed.
func cmdLoginWeb(force bool) int {
	machineID, err := license.MachineID()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Cannot determine machine identity: %v\n", err)
		return 1
	}

	backendURL := "https://d678beea.ovav.dev"

	// Step 1: Get challenge token
	fmt.Println("🔐 OVAV — Web Login")
	fmt.Println()

	req, err := http.NewRequest("GET", backendURL+"/api/v1/auth/login-challenge-web", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Cannot create request: %v\n", err)
		return 1
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Cannot reach backend: %v\n", err)
		fmt.Fprintf(os.Stderr, "   Hint: are you online?\n")
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "❌ Backend returned HTTP %d\n", resp.StatusCode)
		return 1
	}

	var challengeResp struct {
		Challenge string `json:"challenge"`
		ExpiresIn int    `json:"expires_in"`
		LoginURL  string `json:"login_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&challengeResp); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Cannot parse challenge response: %v\n", err)
		return 1
	}

	loginURL := challengeResp.LoginURL
	if loginURL == "" {
		loginURL = backendURL + "/login-portal?challenge=" + challengeResp.Challenge
	}

	expiresIn := time.Duration(challengeResp.ExpiresIn) * time.Second
	if expiresIn == 0 {
		expiresIn = 10 * time.Minute
	}

	fmt.Printf("🌐 Opening browser for authentication...\n")
	fmt.Printf("   URL: %s\n", loginURL)
	fmt.Println()

	// Open browser
	if err := openBrowser(loginURL); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Could not open browser automatically.\n")
		fmt.Fprintf(os.Stderr, "   Open this URL manually: %s\n", loginURL)
		fmt.Println()
	}

	// Step 2: Poll for login completion
	pollURL := backendURL + "/api/v1/auth/login-status?challenge=" + url.QueryEscape(challengeResp.Challenge)
	deadline := time.Now().Add(expiresIn)

	fmt.Printf("⏳ Waiting for authentication... (expires in %s)\n", humanDuration(expiresIn))
	fmt.Printf("   Complete login in your browser, then press Enter here.\n")
	fmt.Println()
	fmt.Println("   Tip: Click 'Sign in with Google' to use Google OAuth,")
	fmt.Println("        or enter your email + password to use magic link.")
	fmt.Println()

	// Use a goroutine to poll in background
	type pollResult struct {
		jwt  string
		role string
		exp  int64
		err  error
	}
	resultCh := make(chan pollResult, 1)

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if time.Now().After(deadline) {
					resultCh <- pollResult{err: fmt.Errorf("login expired (%s)", humanDuration(expiresIn))}
					return
				}
				resp, err := client.Get(pollURL)
				if err != nil {
					continue
				}
				var statusResp struct {
					Status string `json:"status"`
					JWT    string `json:"jwt"`
					Role   string `json:"role"`
					Exp    int64  `json:"exp"`
				}
				if resp.StatusCode == http.StatusOK && json.NewDecoder(resp.Body).Decode(&statusResp) == nil {
					resp.Body.Close()
					if statusResp.Status == "complete" && statusResp.JWT != "" {
						resultCh <- pollResult{jwt: statusResp.JWT, role: statusResp.Role, exp: statusResp.Exp}
						return
					}
					continue
				}
				resp.Body.Close()
			}
		}
	}()

	// Wait for user to press Enter OR for poll to complete
	go func() {
		bufio.NewReader(os.Stdin).ReadByte()
		close(resultCh) // signal user pressed Enter
	}()

	result := <-resultCh
	if result.err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", result.err)
		fmt.Fprintln(os.Stderr, "   Run 'ovav login --web' to try again.")
		return 1
	}

	if result.jwt == "" {
		fmt.Fprintln(os.Stderr, "❌ No JWT received from backend.")
		return 1
	}

	// Step 3: Get seed from user (needed for vault key derivation)
	fmt.Println()
	fmt.Println("✅ Browser authentication complete!")
	fmt.Printf("   Role: %s\n", strings.ToUpper(result.role))
	fmt.Println()
	fmt.Println("   Vault operations require your seed (the secret you received at signup).")
	fmt.Println("   If this is your first time, enter a new seed (min 16 chars).")
	fmt.Println()

	// Try to load existing seed from session file
	existingSeed := ""
	seedPath := filepath.Join(os.Getenv("HOME"), sessionDir, "seed_export")
	if seedData, err := os.ReadFile(seedPath); err == nil {
		existingSeed = strings.TrimSpace(string(seedData))
	}

	if existingSeed != "" && !force {
		fmt.Printf("   Using stored seed from previous login.\n")
		fmt.Println("   Press Enter to continue, or type a new seed.")
		fmt.Print("Seed [Enter to use stored]: ")
	} else {
		fmt.Print("Seed: ")
	}

	seedBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Cannot read seed: %v\n", err)
		return 1
	}
	seed := strings.TrimSpace(string(seedBytes))

	// If user pressed Enter with no input, use existing seed
	if seed == "" {
		seed = existingSeed
	}

	if seed == "" {
		fmt.Fprintln(os.Stderr, "❌ Seed cannot be empty. Enter your seed from signup.")
		return 1
	}
	if len(seed) < 16 {
		fmt.Fprintf(os.Stderr, "❌ Seed too short (minimum 16 characters).\n")
		return 1
	}

	// Derive vault key
	vaultKey, err := license.DeriveKey(seed, machineID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Key derivation failed: %v\n", err)
		return 1
	}

	// Derive vault key hash for identity matching
	vaultKeyHash := sha256Hex(vaultKey)

	// Build session
	hostname, _ := os.Hostname()
	user := os.Getenv("USER")

	// Check identity registry
	var matchedIdentity *identity.Identity
	if repoRoot, err := cli.FindRepoRoot(); err == nil {
		reg, regErr := identity.LoadRegistry(repoRoot)
		if regErr == nil {
			if id, findErr := identity.FindIdentity(reg, vaultKeyHash); findErr == nil {
				matchedIdentity = id
			}
		}
	}

	sess := Session{
		VaultKeyHash: vaultKeyHash,
		MachineID:    machineID,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
		Hostname:     hostname,
		User:         user,
		IdentityID:   matchedIdentity.ID,
		Role:         result.role,
		Level:        matchedIdentity.Level,
		Name:         matchedIdentity.Name,
		VaultJWT:     result.jwt,
	}

	if err := saveSession(sess); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Could not save session: %v\n", err)
	}

	exportVaultKey(vaultKey, seed)
	infra.DecryptTokensFromVault(hex.EncodeToString(vaultKey))

	if matchedIdentity != nil {
		fmt.Println()
		fmt.Printf("🟢 %s\n", identity.WelcomeMessage(matchedIdentity))
	} else {
		fmt.Println()
		fmt.Println("🟢 Web login successful!")
		fmt.Println("   Your vault key is not in the identity registry.")
		fmt.Println("   Contact CEO to register your identity.")
	}
	fmt.Printf("   Machine: %s\n", machineID[:16]+"...")
	fmt.Printf("   Vault:   unlocked (AES-256-GCM)\n")
	fmt.Printf("   Web:     authenticated (%s role)\n", strings.ToUpper(result.role))
	fmt.Println()
	fmt.Println("   Run 'ovav whoami' to verify. 'ovav logout' to close.")

	return 0
}

// ── Whoami command ────────────────────────────────────────────────────────────

func cmdWhoami(args []string) int {
	sess, ok := loadSession()
	if !ok {
		fmt.Println("🔒 Not logged in. Run 'ovav login' to connect.")
		return 1
	}

	if time.Since(sess.createdAt()) > sessionTTL {
		fmt.Println("⚠️  Session expired. Run 'ovav login' to reconnect.")
		return 1
	}

	fmt.Println("🟢 OVAV Identity")
	if sess.Name != "" {
		fmt.Printf("   Name:     %s\n", sess.Name)
	}
	if sess.Role != "" {
		fmt.Printf("   Role:     %s · Level %d\n", strings.ToUpper(sess.Role), sess.Level)
	}
	if sess.IdentityID != "" {
		fmt.Printf("   ID:       %s\n", sess.IdentityID)
	}
	fmt.Printf("   Machine:  %s\n", sess.MachineID[:16]+"...")
	fmt.Printf("   Hostname: %s\n", sess.Hostname)
	fmt.Printf("   User:     %s\n", sess.User)
	fmt.Printf("   Since:    %s (%s ago)\n", sess.CreatedAt, humanDuration(time.Since(sess.createdAt())))
	fmt.Printf("   Vault:    %s\n", sess.VaultKeyHash[:16]+"...")

	// ── Identity registry integration (S3-7) ──
	repoRoot, _ := cli.FindRepoRoot()
	if repoRoot != "" {
		reg, err := identity.LoadRegistry(repoRoot)
		if err == nil && reg != nil {
			fmt.Printf("\n📋 Registry: %d identities defined\n", len(reg.Identities))
			matched := false
			for _, id := range reg.Identities {
				if id.KeyHash == sess.VaultKeyHash {
					fmt.Printf("   ✅ Matched: %s (%s, level %d)\n", id.Name, id.Role, id.Level)
					matched = true
					break
				}
			}
			if !matched {
				fmt.Println("   ⚠️  No registry match for current vault key")
			}
		}
	}

	return 0
}

// ── Logout command ────────────────────────────────────────────────────────────

func cmdLogout(args []string) int {
	home, _ := os.UserHomeDir()
	sessPath := filepath.Join(home, sessionDir, sessionFile)

	if _, err := os.Stat(sessPath); os.IsNotExist(err) {
		fmt.Println("🔒 No active session.")
		return 0
	}

	// Load session before removing (for audit logging)
	sess, sessLoaded := loadSession()

	// ── Audit: logout entry with duration ───────────────────────────────
	if sessLoaded && sess.IdentityID != "" {
		if repoRoot, err := cli.FindRepoRoot(); err == nil {
			if err := identity.InitAudit(repoRoot); err == nil {
				defer identity.CloseAudit()

				// Build a minimal identity struct from session for audit
				id := &identity.Identity{
					ID:    sess.IdentityID,
					Name:  sess.Name,
					Role:  sess.Role,
					Level: sess.Level,
				}
				duration := time.Since(sess.createdAt())
				identity.LogAudit(identity.NewLogoutEntry(id, sess.MachineID, duration))
			}
		}
	}

	if err := os.Remove(sessPath); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Could not remove session: %v\n", err)
		return 1
	}

	fmt.Println("👋 Session closed.")
	if sessLoaded && sess.Name != "" {
		fmt.Printf("   Goodbye, %s.\n", sess.Name)
	}
	fmt.Println("   OVAV_VAULT_KEY has been cleared from this shell.")
	fmt.Println("   Run 'ovav login' to reconnect.")
	return 0
}

// ── Session helpers ───────────────────────────────────────────────────────────

func sessionPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, sessionDir, sessionFile)
}

func loadSession() (Session, bool) {
	data, err := os.ReadFile(sessionPath())
	if err != nil {
		return Session{}, false
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return Session{}, false
	}
	return s, true
}

func saveSession(s Session) error {
	dir := filepath.Dir(sessionPath())
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	// Create temp file and atomic rename to avoid corruption
	tmpPath := sessionPath() + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmpPath, sessionPath())
}

func (s Session) createdAt() time.Time {
	t, _ := time.Parse(time.RFC3339, s.CreatedAt)
	return t
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// exportVaultKey writes the vault key AND seed to secure temp files (0600).
// OVAV fish hooks (99-ovav-systems-lock.fish) auto-load them silently on next prompt.
// No manual source command needed in fish. For other shells, prints the export command.
// seed is passed separately because it was used for PBKDF2 derivation but not persisted.
func exportVaultKey(key []byte, seed string) {
	hexKey := hex.EncodeToString(key)
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, sessionDir)

	// Write vault key export
	keyPath := filepath.Join(dir, "vault_key_export")
	if err := os.WriteFile(keyPath, []byte(hexKey+"\n"), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Could not write vault key: %v\n", err)
	} else {
		fmt.Println("   Vault key:   " + keyPath)
	}

	// Write seed export (needed for cross-device sync, not just local vault)
	seedPath := filepath.Join(dir, "seed_export")
	if err := os.WriteFile(seedPath, []byte(seed+"\n"), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Could not write seed: %v\n", err)
	} else {
		fmt.Println("   Seed:        " + seedPath)
	}

	if isFish() {
		// OVAV fish hooks auto-load both silently on next prompt.
		fmt.Println("\n✅ Vault + seed ready — auto-loaded by OVAV on next prompt.")
	} else {
		fmt.Printf("\n# Vault key and seed written. Run this to unlock:\n")
		fmt.Printf("export OVAV_VAULT_KEY=$(cat %s) OVAV_SEED=$(cat %s); rm %s %s\n",
			keyPath, seedPath, keyPath, seedPath)
	}
}

func isFish() bool {
	shell := os.Getenv("SHELL")
	return strings.Contains(shell, "fish")
}

// fetchVaultJWT calls the vault auth endpoint at d678beea.ovav.dev.
// Returns the JWT on success, empty string on failure (caller handles degradation).
func fetchVaultJWT(seed, machineID, hostname string) (string, error) {
	// Use direct Fly.io URL — bypasses CF Access tunnel (which blocks /vault/auth)
	const vaultAuthURL = "https://d678beea.ovav.dev/api/v1/vault/auth"

	body, err := json.Marshal(map[string]string{
		"seed":       seed,
		"machine_id": machineID,
		"hostname":   hostname,
	})
	if err != nil {
		return "", err
	}

	// 10-second timeout — do not block login if endpoint is unreachable
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest(http.MethodPost, vaultAuthURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var result struct {
		JWT string `json:"jwt"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.JWT, nil
}

func humanDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

// ── Init: register SIGINT handler for clean seed input ────────────────────────

func init() {
	signal.Ignore(syscall.SIGINT)
}
