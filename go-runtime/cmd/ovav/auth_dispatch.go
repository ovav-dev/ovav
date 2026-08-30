package main

import (
	"fmt"
	"os"

	"github.com/ovav/ovav/cmd/ovav/auth"
)

// cmdAuth implements `ovav auth <sub>` — the new 2-flow auth surface.
//
// Usage:
//
//	ovav auth                       # help
//	ovav auth local [--persist]    # R-1: no seed persistence by default
//	ovav auth local --seed-file    # file-based (auto-shred)
//	ovav auth web [--check]        # R-3: mandatory preflight
//	ovav auth status                # both local + web state
//	ovav auth signout               # clear both
//
// The legacy `ovav login --web`, `--recover-ceo`, `--force` flags
// continue to work under `ovav login` (back-compat). New code should
// prefer `ovav auth local|web`.
func cmdAuth(args []string) int {
	if len(args) == 0 {
		printAuthHelp()
		return 0
	}
	sub := args[0]
	subArgs := args[1:]
	switch sub {
	case "local":
		return auth.CmdLocal(subArgs)
	case "web":
		return auth.CmdWeb(subArgs)
	case "status":
		return cmdAuthStatus(subArgs)
	case "signout":
		return cmdAuthSignout(subArgs)
	case "logout":
		// alias
		return cmdAuthSignout(subArgs)
	case "--help", "-h", "help":
		printAuthHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Unknown auth subcommand: %s\n", sub)
		printAuthHelp()
		return 2
	}
}

func printAuthHelp() {
	fmt.Println("ovav auth — OVAV identity authentication (2-flow model)")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  ovav auth local              Offline seed-based login (machine-bound)")
	fmt.Println("  ovav auth web                Browser-based OAuth via ovav.dev")
	fmt.Println("  ovav auth status             Show both local + web state")
	fmt.Println("  ovav auth signout            Clear both local and web sessions")
	fmt.Println()
	fmt.Println("OPTIONS for `ovav auth local`:")
	fmt.Println("  --persist         Keep seed_export (R-1 default: NEVER)")
	fmt.Println("  --seed-file PATH  Read seed from file (auto-shred after use)")
	fmt.Println()
	fmt.Println("OPTIONS for `ovav auth web`:")
	fmt.Println("  --check           Preflight only (HTTP probe + JSON schema)")
	fmt.Println("  --no-open         Print URL instead of opening browser")
	fmt.Println("  --timeout N       Custom poll timeout in seconds (default: 90)")
	fmt.Println()
	fmt.Println("DESIGN RULES (see .ovav/plan/auth-reconstruction.md):")
	fmt.Println("  R-1  No plaintext seed on disk unless --persist")
	fmt.Println("  R-2  Stale locks auto-purged (PID-liveness)")
	fmt.Println("  R-3  Web refuses to launch on broken backend")
	fmt.Println("  R-4  TTY | --seed-file | SEED env as seed input")
	fmt.Println("  R-5  Single canonical identity source (vault.key + registry)")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  ovav auth local                       # TTY prompt")
	fmt.Println("  SEED=... ovav auth local              # via env (audit-only)")
	fmt.Println("  ovav auth web --check                 # probe backend only")
	fmt.Println("  ovav auth status                      # what's active")
}

func cmdAuthStatus(args []string) int {
	fmt.Println("🔐 OVAV auth status — both flows")
	fmt.Println()
	fmt.Println("── local (offline, machine-bound) ──")
	if _, err := os.Stat("/home/braka/.config/ovav/vault.key"); err == nil {
		fmt.Println("   ✅ vault.key present")
	} else {
		fmt.Println("   ⚠️  no vault.key — run `ovav auth local`")
	}
	if _, err := os.Stat("/home/braka/.config/ovav/session"); err == nil {
		fmt.Println("   ✅ session file present")
	} else {
		fmt.Println("   ⚠️  no session — run `ovav auth local`")
	}
	fmt.Println()
	fmt.Println("── web (browser-based OAuth) ──")
	if _, err := os.Stat("/home/braka/.config/ovav/vault/vault_jwt.enc"); err == nil {
		fmt.Println("   ✅ vault_jwt.enc present (web JWT stored encrypted)")
	} else {
		fmt.Println("   ⚠️  no JWT — run `ovav auth web`")
	}
	fmt.Println()
	fmt.Println("── identity registry match ──")
	fmt.Println("   Run `ovav whoami` to see resolved identity.")
	return 0
}

func cmdAuthSignout(args []string) int {
	home := os.Getenv("HOME")
	if home == "" {
		home = "/home/braka"
	}
	targets := []string{
		"/home/braka/.config/ovav/vault.key",
		"/home/braka/.config/ovav/session",
		"/home/braka/.config/ovav/vault/vault_jwt.enc",
		"/home/braka/.local/share/ovav/seed_export",
		"/home/braka/.local/share/ovav/vault_key_export",
	}
	removed := 0
	for _, p := range targets {
		if err := os.Remove(p); err == nil {
			fmt.Printf("   ✅ removed %s\n", p)
			removed++
		}
	}
	if removed == 0 {
		fmt.Println("   (nothing to remove)")
	} else {
		fmt.Printf("\n🟢 %d session artifacts removed\n", removed)
	}
	return 0
}
