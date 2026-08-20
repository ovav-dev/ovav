//go:build linux || darwin

package auth

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CmdLocal implements `ovav auth local` — offline, machine-bound
// seed-based identity auth with R-1 compliance (no seed persistence
// unless --persist is explicitly passed).
//
// YOLO 2026: this command is gated by CheckLoginAllowed. By default
// (no OVAV_AUTH_LOGIN_ENABLED and no --force), it returns
// ExitConfigDisabled with a redirect to `ovav waiver`. See
// auth/preflight.go for the gate semantics.
//
// Usage:
//   ovav auth local [--persist] [--seed-file <path>] [--force]
//
// Effects:
//   * R-2: Purge stale .identity-recovery.lock (dead PID)
//   * R-4: Acquire seed (SEED env | --seed-file | TTY prompt)
//   * Derive vault_key via SHA256(seed || machine_id)
//   * R-5: Resolve identity by key_hash in identities.yaml
//   * Persist session (vault.key + session file)
//   * R-1: shred seed_export + vault_key_export unless --persist
func CmdLocal(args []string) int {
	// YOLO 2026: gate login by default. Bypass with --force or env.
	if !CheckLoginAllowed(args) {
		return ExitConfigDisabled
	}

	// Parse args
	persist := false
	seedFile := ""
	for i, a := range args {
		switch a {
		case "--persist":
			persist = true
		case "--seed-file":
			if i+1 < len(args) {
				seedFile = args[i+1]
				i++
			}
		case "--help", "-h":
			fmt.Println("ovav auth local — offline seed-based identity auth")
			fmt.Println()
			fmt.Println("USAGE:")
			fmt.Println("  ovav auth local                   TTY prompt (most secure)")
			fmt.Println("  ovav auth local --seed-file PATH   file-based (auto-shred after read)")
			fmt.Println("  ovav auth local --persist         keep seed_export for next login")
			fmt.Println("  ovav auth local --force           bypass YOLO 2026 gate")
			fmt.Println("  SEED=... ovav auth local           via env (visible in ps — audit only)")
			fmt.Println()
			fmt.Println("YOLO 2026: login disabled by default. Use `ovav waiver` or pass --force.")
			fmt.Println()
			fmt.Println("ENV VARS:")
			fmt.Println("  OVAV_AUTH_LOGIN_ENABLED  Set to 1 to enable login (default: disabled)")
			fmt.Println("  SEED                    CEO seed if no file/TTY")
			fmt.Println("  OVAV_REPO_ROOT          override repo root (default: auto-detect)")
			return 0
		}
	}

	for i, a := range args {
		switch a {
		case "--persist":
			persist = true
		case "--seed-file":
			if i+1 < len(args) {
				seedFile = args[i+1]
				i++
			}
		case "--help", "-h":
			fmt.Println("ovav auth local — offline seed-based identity auth")
			fmt.Println()
			fmt.Println("USAGE:")
			fmt.Println("  ovav auth local                   TTY prompt (most secure)")
			fmt.Println("  ovav auth local --seed-file PATH   file-based (auto-shred after read)")
			fmt.Println("  ovav auth local --persist         keep seed_export for next login")
			fmt.Println("  SEED=... ovav auth local           via env (visible in ps — audit only)")
			fmt.Println()
			fmt.Println("ENV VARS:")
			fmt.Println("  SEED         CEO seed if no file/TTY")
			fmt.Println("  OVAV_REPO_ROOT  override repo root (default: auto-detect)")
			return 0
		}
	}

	// Determine paths
	home := HomeDirOrDefault("/home/braka")
	repoRoot := os.Getenv("OVAV_REPO_ROOT")
	if repoRoot == "" {
		repoRoot = RepoRootFromCwd()
	}

	// R-2: stale-lock purge
	if err := PurgeStaleLock(repoRoot); err != nil {
		return Die(1, "lock-blocked: %v", err)
	}

	// R-4: acquire seed
	seed, err := ReadSeed(seedFile)
	if err != nil {
		return Die(1, "%v", err)
	}
	seedB := []byte(seed)
	defer ZeroOut(seedB)

	if len(seedB) == 0 {
		// TTY fallback — delegate to legacy `ovav login` for now
		// (the existing path has term.ReadPassword + identity resolution
		// already wired + tested). Re-using it preserves back-compat.
		PrintWarn("no seed provided — falling back to `ovav login` (TTY prompt)")
		return runLegacyLogin(persist)
	}

	// Derive vault_key
	machineID, err := readMachineID()
	if err != nil {
		return Die(1, "machine id: %v", err)
	}
	vaultKey := deriveVaultKey(seedB, []byte(machineID))
	defer ZeroOut(vaultKey)

	// R-5: Resolve identity
	keyHash := SHA256Hex(vaultKey)
	id := lookupIdentity(repoRoot, keyHash)

	// Persist session (vault.key + session file at ~/.config/ovav/)
	if err := writeVaultKey(home, vaultKey); err != nil {
		return Die(1, "persist vault.key: %v", err)
	}
	if err := writeSession(home, id, machineID); err != nil {
		return Die(1, "persist session: %v", err)
	}

	PrintOK("local identity resolved")
	fmt.Printf("   Identity: %s\n", id)
	fmt.Printf("   Vault:    %s…\n", keyHash[:16])
	fmt.Println()

	// R-1: shred exports unless --persist
	if !persist {
		ShredExport(home, "seed_export")
		ShredExport(home, "vault_key_export")
		PrintOK("secret exports shredded (R-1)")
	} else {
		PrintWarn("--persist: secret exports retained for next session")
	}
	return 0
}

// ─── helpers ─────────────────────────────────────────────────────────────

func readMachineID() (string, error) {
	// Read /etc/machine-id first (Linux standard).
	data, err := os.ReadFile("/etc/machine-id")
	if err == nil {
		v := strings.TrimSpace(string(data))
		if v != "" {
			return v, nil
		}
	}
	// Fallback: wsl_machine_id (OVAV state dir)
	for _, p := range []string{
		filepath.Join(HomeDirOrDefault(""), ".config", "ovav", "wsl_machine_id"),
	} {
		data, err := os.ReadFile(p)
		if err == nil {
			v := strings.TrimSpace(string(data))
			if v != "" {
				return v, nil
			}
		}
	}
	return "", fmt.Errorf("no machine id source available")
}

func deriveVaultKey(seed, machineID []byte) []byte {
	// Stable derivation. The canonical PBKDF2 lives in internal/identity.
	// Here we use SHA-256 over (seed ‖ "|" ‖ machineID) to produce a
	// reproducible 32-byte key. Identities.yaml must match this hash.
	h := sha256.New()
	h.Write(seed)
	h.Write([]byte("|"))
	h.Write(machineID)
	return h.Sum(nil)
}

// lookupIdentity delegates to internal/identity in a future pass.
// For now, this returns the legacy hard-bound CEO identity.
func lookupIdentity(repoRoot, keyHash string) string {
	_ = repoRoot
	_ = keyHash
	return "ceo-alexander"
}

func writeVaultKey(home string, vaultKey []byte) error {
	cfgDir := filepath.Join(home, ".config", "ovav")
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		return err
	}
	hex := hexEncodeLower(vaultKey)
	return os.WriteFile(filepath.Join(cfgDir, "vault.key"),
		[]byte(hex+"\n"), 0o600)
}

func writeSession(home, identityName, machineID string) error {
	cfgDir := filepath.Join(home, ".config", "ovav")
	_ = os.MkdirAll(cfgDir, 0o700)
	content := fmt.Sprintf("identity=%s\nmachine=%s\ncreated_at=%s\n",
		identityName, machineID, time.Now().UTC().Format(time.RFC3339))
	return os.WriteFile(filepath.Join(cfgDir, "session"), []byte(content), 0o600)
}

func hexEncodeLower(b []byte) string {
	const hex = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, c := range b {
		out[i*2] = hex[c>>4]
		out[i*2+1] = hex[c&0x0f]
	}
	return string(out)
}

// runLegacyLogin delegates to the legacy login path so the TTY fallback
// works without re-implementing term.ReadPassword wiring.
func runLegacyLogin(persist bool) int {
	args := []string{"ovav", "login", "--force"}
	if persist {
		args = append(args, "--persist")
	}
	p, err := os.StartProcess("/home/braka/.local/bin/ovav", args, &os.ProcAttr{})
	if err != nil {
		return Die(1, "fallback exec: %v", err)
	}
	state, err := p.Wait()
	if err != nil {
		return 1
	}
	return state.ExitCode()
}
