// OVAV Vault Secrets — Secrets Subsystem CLI.
//
// Part of the OVAV Vault 2026 plan.
// See: plans/OVAV-VAULT-2026/PLAN.md
//
// Usage:
//
//	ovav-vault-secrets add --type api_token --name "CF Production" --value $CF_API_TOKEN
//	ovav-vault-secrets list [--type <type>] [--json]
//	ovav-vault-secrets get --id <uuid>
//	ovav-vault-secrets remove --id <uuid>
//	ovav-vault-secrets discover
//	ovav-vault-secrets health
//
// Key loading:
//   - OVAV_SECRETS_KEY env var (hex-encoded 32-byte key)
//   - ~/.local/share/ovav/vault_key_export (raw 32-byte key)
//   - --key <path> flag (hex-encoded or raw)
//
// Stack: Go 1.25+, stdlib only. Zero external dependencies.
package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/license"
	"github.com/ovav/ovav/internal/vault/secrets"
)

// ═══════════════════════════════════════════════════════════════════════════
// GLOBAL FLAGS (re-parsed per command via flag.CommandLine)
// ═══════════════════════════════════════════════════════════════════════════

var (
	flagJSON       = flag.Bool("json", false, "Output as JSON")
	flagKeyPath    = flag.String("key", "", "Path to vault key file")
	flagSecretType = flag.String("type", "", "Secret type")
	flagName       = flag.String("name", "", "Human-readable name")
	flagValue      = flag.String("value", "", "Secret value (or @filename)")
	flagID         = flag.String("id", "", "Secret ID (UUID)")
	flagShow       = flag.Bool("show", false, "Show secret value")
	flagProvider   = flag.String("provider", "", "Provider (openai, anthropic, openrouter, azure)")
	flagKeyName    = flag.String("key-name", "", "Display name for the API key")
	flagLimit      = flag.Int("limit", 0, "Monthly spend limit in cents (e.g. 50000 = $500)")
)

const usage = `OVAV Vault Secrets — Secrets Subsystem CLI

Usage:
  ovav-vault-secrets add --type <type> --name <name> --value <value>
  ovav-vault-secrets list [--type <type>] [--json]
  ovav-vault-secrets get --id <uuid> [--show]
  ovav-vault-secrets remove --id <uuid>
  ovav-vault-secrets discover
  ovav-vault-secrets health
  ovav-vault-secrets backup [--key <path>]
  ovav-vault-secrets restore --key <path>
  ovav-vault-secrets connect add --provider <openai|anthropic|openrouter|azure> [--key-name <name>] [--value <key>]
  ovav-vault-secrets connect list
  ovav-vault-secrets connect status
  ovav-vault-secrets connect track --provider <provider>
  ovav-vault-secrets connect sync-spend --provider <provider>
  ovav-vault-secrets connect detect
  ovav-vault-secrets sync

Secret types:
  api_token       Third-party API tokens (CF, FLY, Resend)
  oauth_creds     OAuth client_id + client_secret
  db_credential  Database connection credentials
  cloud_key      Cloud provider keys (AWS, GCP)
  encryption_key Symmetric keys (HMAC, JWT)
  user_secret    End-user secrets (Firebase, API keys)
  tunnel_token   Tunnel credentials (Cloudflare Tunnel)

OVAV CONNECT providers:
  openai        OpenAI API (OPENAI_API_KEY)
  anthropic     Anthropic API (ANTHROPIC_API_KEY)
  openrouter    OpenRouter API (OPENROUTER_API_KEY)
  azure         Azure OpenAI (AZURE_OPENAI_KEY)

Key loading (in order of priority):
  1. OVAV_SECRETS_KEY          Hex-encoded 32-byte key
  2. ~/.local/share/ovav/vault_key_export (raw 32-byte key)
  3. --key <path>              Hex-encoded or raw key file

Examples:
  ovav-vault-secrets add --type api_token --name "CF prod" --value $CF_API_TOKEN
  ovav-vault-secrets list --type api_token --json
  ovav-vault-secrets get --id 550e8400-e29b-41d4-a716-446655440000 --show
  ovav-vault-secrets remove --id 550e8400-e29b-41d4-a716-446655440000
`

// ═══════════════════════════════════════════════════════════════════════════
// KEY LOADING
// ═══════════════════════════════════════════════════════════════════════════

func loadKey() ([]byte, error) {
	// 1. OVAV_SECRETS_KEY env
	if envKey := os.Getenv("OVAV_SECRETS_KEY"); envKey != "" {
		return hex.DecodeString(strings.TrimSpace(envKey))
	}

	// 2. --key flag
	if *flagKeyPath != "" {
		data, err := os.ReadFile(*flagKeyPath)
		if err != nil {
			return nil, fmt.Errorf("reading key file %s: %w", *flagKeyPath, err)
		}
		trimmed := strings.TrimSpace(string(data))
		if len(trimmed) == 64 {
			return hex.DecodeString(trimmed)
		}
		if len(data) == 32 {
			return data, nil
		}
		return nil, fmt.Errorf("key file must be 32 raw bytes or 64 hex chars, got %d bytes", len(data))
	}

	// 3. Fallback: vault_key_export
	exportPath := filepath.Join(os.Getenv("HOME"), ".local", "share", "ovav", "vault_key_export")
	if data, err := os.ReadFile(exportPath); err == nil {
		if len(data) == 32 {
			return data, nil
		}
		if len(strings.TrimSpace(string(data))) == 64 {
			return hex.DecodeString(strings.TrimSpace(string(data)))
		}
	}

	return nil, fmt.Errorf("no key found — set OVAV_SECRETS_KEY env or use --key <path>")
}

// ═══════════════════════════════════════════════════════════════════════════
// STORE HELPERS
// ═══════════════════════════════════════════════════════════════════════════

func openStore(key []byte) (*secrets.SecretStore, error) {
	store, err := secrets.Load(key)
	if err == secrets.ErrNotFound {
		return secrets.NewSecretStore(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("loading vault: %w", err)
	}
	return store, nil
}

func persistStore(store *secrets.SecretStore, key []byte) error {
	if err := store.Save(key); err != nil {
		return fmt.Errorf("saving vault: %w", err)
	}
	return nil
}

// auditAppend appends an audit log entry.
// Errors are logged to stderr but do not fail the command.
func auditAppend(key []byte, action secrets.AuditAction, secretID, secretName string) {
	log, err := secrets.NewAuditLog(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  audit log unavailable: %v\n", err)
		return
	}
	// MachineID — use hostname as proxy
	hostname, _ := os.Hostname()
	entry := secrets.LogEntry{
		SecretID:   secretID,
		SecretName: secretName,
		Action:     action,
		Source:     "cli",
		MachineID:  hostname,
	}
	if err := log.Append(entry); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  audit append failed: %v\n", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// COMMANDS
// ═══════════════════════════════════════════════════════════════════════════

func cmdAdd(key []byte) error {
	if *flagName == "" {
		return fmt.Errorf("--name is required")
	}
	if *flagSecretType == "" {
		return fmt.Errorf("--type is required")
	}
	if *flagValue == "" {
		return fmt.Errorf("--value is required")
	}

	value := *flagValue
	if strings.HasPrefix(value, "@") {
		data, err := os.ReadFile(value[1:])
		if err != nil {
			return fmt.Errorf("reading value file %s: %w", value[1:], err)
		}
		value = strings.TrimSpace(string(data))
	}

	st := secrets.SecretType(*flagSecretType)
	valid := false
	for _, t := range secrets.AllTypes() {
		if string(t) == *flagSecretType {
			valid = true
			st = t
			break
		}
	}
	if !valid {
		return fmt.Errorf("unknown type %q — valid: api_token, oauth_creds, db_credential, cloud_key, encryption_key, user_secret, tunnel_token", *flagSecretType)
	}

	provider := strings.Split(*flagName, " ")[0]

	store, err := openStore(key)
	if err != nil {
		return err
	}

	sec := secrets.NewSecret(*flagName, st, provider, "manual", []byte(value))
	if err := store.Add(sec); err != nil {
		return fmt.Errorf("adding secret: %w", err)
	}
	if err := persistStore(store, key); err != nil {
		return err
	}

	auditAppend(key, secrets.AuditAdd, sec.ID, sec.Name)
	fmt.Printf("✅ Secret added: %s (%s)\n", sec.Name, sec.ID)
	return nil
}

func cmdList(key []byte) error {
	store, err := openStore(key)
	if err != nil {
		return err
	}

	list := store.List(secrets.SecretType(*flagSecretType))
	auditAppend(key, secrets.AuditList, "", "")

	if *flagJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(list)
	}

	if len(list) == 0 {
		fmt.Println("No secrets found.")
		return nil
	}

	fmt.Printf("%-38s %-20s %-15s %s\n", "ID", "NAME", "TYPE", "PROVIDER")
	fmt.Println(strings.Repeat("-", 90))
	for _, sec := range list {
		id := sec.ID
		if len(id) > 38 {
			id = id[:38]
		}
		name := sec.Name
		if len(name) > 20 {
			name = name[:17] + "..."
		}
		fmt.Printf("%-38s %-20s %-15s %s\n", id, name, sec.Type, sec.Provider)
	}
	fmt.Printf("\nTotal: %d secret(s)\n", len(list))
	return nil
}

func cmdGet(key []byte) error {
	if *flagID == "" {
		return fmt.Errorf("--id is required")
	}

	store, err := openStore(key)
	if err != nil {
		return err
	}

	sec := store.Get(*flagID)
	if sec == nil {
		return fmt.Errorf("secret not found: %s", *flagID)
	}

	// Update last used
	store.UpdateUsage(*flagID)
	persistStore(store, key)
	auditAppend(key, secrets.AuditGet, sec.ID, sec.Name)

	if *flagShow {
		fmt.Println("⚠️  Showing secret — treat as sensitive!")
	}

	if *flagJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(sec)
	}

	fmt.Printf("ID:        %s\n", sec.ID)
	fmt.Printf("Name:      %s\n", sec.Name)
	fmt.Printf("Type:      %s\n", sec.Type)
	fmt.Printf("Provider:  %s\n", sec.Provider)
	fmt.Printf("Source:    %s\n", sec.Source)
	fmt.Printf("Hash:      %s\n", sec.Hash)
	fmt.Printf("Created:   %s\n", sec.CreatedAt.Format(time.RFC3339))
	if sec.ExpiresAt != nil {
		fmt.Printf("Expires:   %s\n", sec.ExpiresAt.Format(time.RFC3339))
	}
	if sec.LastUsed != nil {
		fmt.Printf("Last used: %s\n", sec.LastUsed.Format(time.RFC3339))
	}
	fmt.Printf("Rotatable: %v\n", sec.Rotatable)
	if len(sec.Tags) > 0 {
		fmt.Printf("Tags:      %s\n", strings.Join(sec.Tags, ", "))
	}
	if len(sec.Metadata) > 0 {
		fmt.Printf("Metadata:  %v\n", sec.Metadata)
	}
	return nil
}

func cmdRemove(key []byte) error {
	if *flagID == "" {
		return fmt.Errorf("--id is required")
	}

	store, err := openStore(key)
	if err != nil {
		return err
	}

	sec := store.Get(*flagID)
	if sec == nil {
		return fmt.Errorf("secret not found: %s", *flagID)
	}

	if err := store.Remove(*flagID); err != nil {
		return fmt.Errorf("removing secret: %w", err)
	}
	if err := persistStore(store, key); err != nil {
		return err
	}

	auditAppend(key, secrets.AuditRemove, sec.ID, sec.Name)
	fmt.Printf("✅ Secret removed: %s (%s)\n", sec.Name, sec.ID)
	return nil
}

func cmdDiscover(key []byte) error {
	store, err := openStore(key)
	if err != nil {
		return err
	}

	cfg := secrets.DiscoveryConfig{
		GitHubOrg: "ovav-dev",
		SearchPaths: []string{
			filepath.Join(os.Getenv("HOME"), "Systems"),
			filepath.Join(os.Getenv("HOME"), ".config"),
		},
	}

	rep, err := secrets.Discover(cfg)
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	// Print report
	fmt.Println("🔍 Discovery Report")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()
	fmt.Println(rep.Summary())
	fmt.Println()

	// Compare against vault — show missing secrets
	fmt.Println("Secrets NOT yet in vault:")
	fmt.Println(strings.Repeat("-", 60))

	missingCount := 0

	// Fly.io secrets
	for app, fl := range rep.Fly {
		for _, f := range fl {
			if store.GetByName(f.Name) == nil {
				missingCount++
				fmt.Printf("  [%s] %s (%s) — %s\n", f.Source, f.Name, f.Type, app)
			}
		}
	}

	// GitHub secrets
	for repo, gl := range rep.GitHub {
		for _, g := range gl {
			if store.GetByName(g.Name) == nil {
				missingCount++
				fmt.Printf("  [%s] %s (%s) — %s\n", g.Source, g.Name, g.Type, repo)
			}
		}
	}

	// Filesystem secrets
	for _, fs := range rep.Files {
		if store.GetByName(fs.Name) == nil {
			missingCount++
			fmt.Printf("  [%s] %s (%s) — %s\n", fs.Source, fs.Name, fs.Type, fs.Path)
		}
	}

	if missingCount == 0 {
		fmt.Println("  ✅ All discovered secrets are already in the vault!")
	} else {
		fmt.Printf("\nTotal missing: %d secret(s)\n", missingCount)
		fmt.Println("Use 'add' to ingest them, or 'health' to check vault status.")
	}

	auditAppend(key, secrets.AuditDiscover, "", "")
	return nil
}

func cmdHealth(key []byte) error {
	store, err := openStore(key)
	if err != nil {
		return err
	}

	reports := secrets.CheckStoreHealth(store)
	auditAppend(key, secrets.AuditHealth, "", "")

	if *flagJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(reports)
	}

	secrets.PrintHealthReport(reports)
	return nil
}

func cmdBackup(key []byte) error {
	store, err := openStore(key)
	if err != nil {
		return err
	}

	backupPath := *flagKeyPath
	if backupPath == "" {
		backupPath = filepath.Join(os.Getenv("HOME"), ".local", "share", "ovav",
			fmt.Sprintf("secrets-backup-%s.enc", time.Now().Format("20060102-150405")))
	}

	if err := secrets.Backup(store, key, backupPath); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	fmt.Printf("✅ Backup created: %s (%d secret(s))\n", backupPath, store.Count())
	return nil
}

func cmdRestore(key []byte) error {
	restorePath := *flagKeyPath
	if restorePath == "" {
		return fmt.Errorf("--key is required for restore (use --key <backup_path>)")
	}

	info, err := secrets.BackupInfo(restorePath)
	if err != nil {
		return fmt.Errorf("backup info: %w", err)
	}

	fmt.Printf("Backup info:\n")
	fmt.Printf("  Created:    %s\n", info.CreatedAt.Format(time.RFC3339))
	fmt.Printf("  Machine:    %s\n", info.MachineID)
	fmt.Printf("  Secrets:    %d\n", info.SecretCount)
	fmt.Printf("  Version:    %d\n", info.Version)
	fmt.Println()

	confirm := ""
	fmt.Print("⚠️  This will REPLACE your current vault with this backup. Continue? (yes/no): ")
	fmt.Scanln(&confirm)
	if confirm != "yes" {
		fmt.Println("Cancelled.")
		return nil
	}

	store, err := secrets.Restore(restorePath, key)
	if err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	if err := store.Save(key); err != nil {
		return fmt.Errorf("saving restored vault: %w", err)
	}

	fmt.Printf("✅ Vault restored from backup: %d secret(s)\n", store.Count())
	return nil
}

func cmdDeps(key []byte) error {
	store, err := openStore(key)
	if err != nil {
		return err
	}

	g, err := secrets.LoadDependencyGraph()
	if err != nil {
		// If no graph file, create empty one
		g = &secrets.DependencyGraph{}
	}

	// Auto-discover from current secrets
	g.DiscoverFromSecrets(store)

	// Parse subcommand
	subCmd := ""
	if len(os.Args) >= 3 {
		subCmd = os.Args[2]
	}

	switch subCmd {
	case "list", "":
		return depsList(store, g, false)
	case "impact":
		return depsImpact(store, g)
	case "orphans":
		return depsOrphans(store, g)
	default:
		fmt.Println("OVAV VAULT — Dependency Graph")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  ovav-vault-secrets deps list              # List all secret dependencies")
		fmt.Println("  ovav-vault-secrets deps impact <name>     # Show what rotates if this secret changes")
		fmt.Println("  ovav-vault-secrets deps orphans            # Show secrets with no tracked dependencies")
		return nil
	}
}

func depsList(store *secrets.SecretStore, g *secrets.DependencyGraph, listOrphans bool) error {
	allSecrets := store.List("")
	if len(allSecrets) == 0 {
		fmt.Println("No secrets in vault.")
		return nil
	}

	if *flagJSON {
		type secretWithDeps struct {
			ID     string              `json:"id"`
			Name   string              `json:"name"`
			Type   string              `json:"type"`
			Source string              `json:"source"`
			Refs   []secrets.SecretRef `json:"refs"`
		}
		var out []secretWithDeps
		for _, s := range allSecrets {
			out = append(out, secretWithDeps{
				ID:     s.ID,
				Name:   s.Name,
				Type:   string(s.Type),
				Source: s.Source,
				Refs:   g.GetRefs(s.ID),
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(out)
		return nil
	}

	fmt.Println()
	fmt.Println("  Secret Dependency Graph")
	fmt.Println("  ══════════════════════════════════════")
	fmt.Printf("  %-30s %-15s %-15s %s\n", "SECRET", "TYPE", "SOURCE", "SYSTEMS")
	fmt.Println()

	printed := 0
	for _, s := range allSecrets {
		refs := g.GetRefs(s.ID)
		systems := g.GetSystemsUsing(s.ID)
		if len(refs) == 0 && listOrphans {
			continue
		}

		sysStr := "—"
		if len(systems) > 0 {
			strs := make([]string, len(systems))
			for i, sys := range systems {
				strs[i] = string(sys)
			}
			sysStr = strings.Join(strs, ", ")
		}

		orphan := ""
		if len(refs) == 0 {
			orphan = " (orphan)"
		}

		fmt.Printf("  %-30s %-15s %-15s %s%s\n",
			s.Name, s.Type, s.Source, sysStr, orphan)
		printed++
	}

	if printed == 0 {
		fmt.Println("  (no secrets with dependencies tracked)")
	}
	fmt.Println()
	return nil
}

func depsImpact(store *secrets.SecretStore, g *secrets.DependencyGraph) error {
	// Get secret name from args
	name := ""
	if len(os.Args) >= 3 {
		name = os.Args[2]
	}
	if name == "" || name == "impact" {
		return fmt.Errorf("specify a secret name: ovav-vault-secrets deps impact <name>")
	}

	sec := store.GetByName(name)
	if sec == nil {
		return fmt.Errorf("secret %q not found in vault", name)
	}

	refs := g.GetRefs(sec.ID)
	if len(refs) == 0 {
		fmt.Printf("  No dependencies tracked for %s.\n", name)
		fmt.Printf("  Secret discovered from: %s\n", sec.Source)
		return nil
	}

	fmt.Println()
	fmt.Printf("  Rotation Impact: %s\n", name)
	fmt.Println("  ══════════════════════════════════════")
	fmt.Println()

	rotatable := make([]secrets.SecretRef, 0, len(refs))
	manual := make([]secrets.SecretRef, 0, len(refs))

	for _, r := range refs {
		if r.AutoRotatable {
			rotatable = append(rotatable, r)
		} else {
			manual = append(manual, r)
		}
	}

	if len(rotatable) > 0 {
		fmt.Println("  🔄 AUTO-ROTATABLE (cPanel can rotate automatically):")
		for _, r := range rotatable {
			fmt.Printf("    • %s: %s → %s\n", r.System, r.Path, r.EnvVar)
		}
		fmt.Println()
	}

	if len(manual) > 0 {
		fmt.Println("  🔒 MANUAL (requires human intervention):")
		for _, r := range manual {
			fmt.Printf("    • %s: %s → %s\n", r.System, r.Path, r.EnvVar)
		}
		fmt.Println()
	}

	return nil
}

func depsOrphans(store *secrets.SecretStore, g *secrets.DependencyGraph) error {
	allSecrets := store.List("")
	orphanedIDs := g.OrphanReport(secretIDs(allSecrets))

	if len(orphanedIDs) == 0 {
		fmt.Println("✅ No orphan secrets — all vaulted secrets are referenced by systems.")
		return nil
	}

	fmt.Println()
	fmt.Println("  Orphaned Secrets (no tracked system dependencies)")
	fmt.Println("  ═════════════════════════════════════════════════")
	for _, id := range orphanedIDs {
		for _, s := range store.List("") {
			if s.ID == id {
				fmt.Printf("  ⚠️  %s (%s) — never referenced by any system\n", s.Name, s.Type)
				break
			}
		}
	}
	fmt.Println()
	fmt.Println("  These secrets may be:")
	fmt.Println("    • Stale — previously used, now abandoned")
	fmt.Println("    • Manual-only — used but not tracked in the graph")
	fmt.Println("    • Backup — kept for disaster recovery")
	return nil
}

func secretIDs(list []*secrets.Secret) []string {
	ids := make([]string, len(list))
	for i, s := range list {
		ids[i] = s.ID
	}
	return ids
}

// parseConnectFlags manually parses flags after the sub-subcommand
// since Go's flag parser stops at the first positional arg.
func parseConnectFlags() string {
	subCmd := ""
	if len(os.Args) >= 3 {
		subCmd = os.Args[2]
	}

	// os.Args[3:] = flags after "ovav-vault-secrets connect <subcmd> ..."
	args := os.Args[3:]
	i := 0
	for i < len(args) {
		arg := args[i]
		if arg == "--provider" || arg == "-p" {
			if i+1 < len(args) {
				*flagProvider = args[i+1]
				i += 2
			} else {
				i++
			}
		} else if arg == "--key-name" || arg == "-n" {
			if i+1 < len(args) {
				*flagKeyName = args[i+1]
				i += 2
			} else {
				i++
			}
		} else if arg == "--value" || arg == "-v" {
			if i+1 < len(args) {
				*flagValue = args[i+1]
				i += 2
			} else {
				i++
			}
		} else if arg == "--limit" || arg == "-l" {
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", flagLimit)
				i += 2
			} else {
				i++
			}
		} else if arg == "--json" || arg == "-j" {
			*flagJSON = true
			i++
		} else if arg == "--show" || arg == "-s" {
			*flagShow = true
			i++
		} else if arg == "--name" || arg == "--id" || arg == "--type" || arg == "--key" {
			// Skip unknown-for-connect flags
			i += 2
		} else {
			// Unknown arg, skip
			i++
		}
	}

	return subCmd
}

func cmdConnect(key []byte, subCmd string) error {
	store, err := openStore(key)
	if err != nil {
		return err
	}
	cs := secrets.NewConnectStore(store, key)

	switch subCmd {
	case "add":
		err = connectAdd(cs, key)
	case "list", "":
		err = connectList(cs)
	case "status":
		err = connectStatus(cs)
	case "track":
		err = connectTrack(cs)
	case "sync-spend":
		err = connectSyncSpend(cs)
	case "detect":
		err = connectDetect(cs)
	default:
		fmt.Println("OVAV CONNECT — AI Provider Key Management")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  ovav-vault-secrets connect add --provider openai --key-name 'GPT-4o Prod'")
		fmt.Println("  ovav-vault-secrets connect list")
		fmt.Println("  ovav-vault-secrets connect status")
		fmt.Println("  ovav-vault-secrets connect track --provider openai")
		fmt.Println("  ovav-vault-secrets connect sync-spend --provider openrouter")
		fmt.Println("  ovav-vault-secrets connect detect")
		fmt.Println()
		fmt.Println("Providers: openai, anthropic, openrouter, azure")
		return nil
	}

	// Persist any in-memory changes (add, track operations)
	if err == nil {
		store.Save(key)
	}
	return err
}

func connectAdd(cs *secrets.ConnectStore, key []byte) error {
	provider := *flagProvider
	if provider == "" {
		return fmt.Errorf("--provider is required (openai, anthropic, openrouter, azure)")
	}

	known := map[string]string{
		"openai":     "OPENAI_API_KEY",
		"anthropic":  "ANTHROPIC_API_KEY",
		"openrouter": "OPENROUTER_API_KEY",
		"azure":      "AZURE_OPENAI_KEY",
	}
	envVar, ok := known[provider]
	if !ok {
		return fmt.Errorf("unknown provider %q — valid: openai, anthropic, openrouter, azure", provider)
	}

	// Get key value
	rawKey := *flagValue
	if rawKey == "" {
		rawKey = os.Getenv(envVar)
	}
	if rawKey == "" {
		return fmt.Errorf("--value required or set %s env var", envVar)
	}

	keyName := *flagKeyName
	if keyName == "" {
		keyName = provider + " API Key"
	}

	if err := cs.AddConnectKey(provider, keyName, rawKey, envVar); err != nil {
		return fmt.Errorf("adding connect key: %w", err)
	}

	auditAppend(key, "connect_add", "", keyName)
	fmt.Printf("✅ CONNECT key added: %s (%s)\n", keyName, provider)
	return nil
}

func connectList(cs *secrets.ConnectStore) error {
	keys, err := cs.ListConnectKeys()
	if err != nil {
		return fmt.Errorf("listing connect keys: %w", err)
	}

	if len(keys) == 0 {
		fmt.Println("No OVAV CONNECT keys found. Run 'connect detect' to auto-vault env keys.")
		return nil
	}

	if *flagJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(keys)
	}

	fmt.Println()
	fmt.Println("  OVAV CONNECT — AI Provider Keys")
	fmt.Println("  ══════════════════════════════════")
	for _, ck := range keys {
		statusIcon := "✅"
		if ck.Status == "expired" || ck.Status == "quota_exceeded" {
			statusIcon = "⚠️"
		}
		fmt.Printf("  %s %-12s %-30s  [%s]\n", statusIcon, ck.Provider, ck.Name, ck.EnvVar)
	}
	fmt.Println()
	return nil
}

func connectStatus(cs *secrets.ConnectStore) error {
	report, err := cs.SpendReport()
	if err != nil {
		return fmt.Errorf("spend report: %w", err)
	}

	if len(report) == 0 {
		fmt.Println("No OVAV CONNECT keys found.")
		return nil
	}

	if *flagJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	fmt.Println()
	fmt.Println("  OVAV CONNECT — Spend Report")
	fmt.Println("  ══════════════════════════════════")
	fmt.Printf("  %-12s %-25s %-10s %-12s %s\n", "PROVIDER", "NAME", "STATUS", "SPEND", "LIMIT")
	fmt.Println()

	for _, r := range report {
		warning := ""
		if r.Warning {
			warning = " ⚠️"
		}
		bar := spendBar(r.UsagePct)
		fmt.Printf("  %-12s %-25s %-10s %-12s [%s]%s\n",
			r.Provider, r.Name, r.Status, r.CurrentSpendFmt, bar, warning)
	}
	fmt.Println()
	return nil
}

func spendBar(pct float64) string {
	const width = 20
	filled := int(pct / 100 * width)
	if filled > width {
		filled = width
	}
	empty := width - filled
	return strings.Repeat("█", filled) + strings.Repeat("░", empty)
}

func connectTrack(cs *secrets.ConnectStore) error {
	provider := *flagProvider
	if provider == "" {
		return fmt.Errorf("--provider required")
	}
	if err := cs.TrackUsage(provider); err != nil {
		return err
	}
	fmt.Printf("✅ Usage tracked for %s\n", provider)
	return nil
}

func connectSyncSpend(cs *secrets.ConnectStore) error {
	provider := *flagProvider
	if provider == "" {
		return fmt.Errorf("--provider required")
	}
	if err := cs.SyncSpend(provider); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Sync failed: %v\n", err)
		return err
	}
	fmt.Printf("✅ Spend synced for %s\n", provider)
	return nil
}

func connectDetect(cs *secrets.ConnectStore) error {
	added, err := cs.DetectAndVaultEnvKeys()
	if err != nil {
		return fmt.Errorf("detect: %w", err)
	}
	if len(added) == 0 {
		fmt.Println("No new env keys found — all providers already vaulted.")
		return nil
	}
	fmt.Printf("✅ Vaulted %d new connect key(s):\n", len(added))
	for _, ck := range added {
		fmt.Printf("   %s (%s)\n", ck.Name, ck.Provider)
	}
	return nil
}

func cmdSync(key []byte) error {
	// Sync requires the SEED (not just the vault key) to derive the SyncWrapKey.
	// The vault key = PBKDF2(seed, machineID). SyncWrapKey = PBKDF2(seed, "ovav-sync-v1").
	// We need the seed for the sync key; the vault key is used for local vault ops.
	//
	// Auth strategy (in priority order):
	// 1. OVAV_SEED env var → used to obtain a fresh JWT via /vault/auth
	// 2. VaultJWT from session file → used directly if seed is not available
	// 3. Neither available → offline mode
	seed := os.Getenv("OVAV_SEED")

	store, err := openStore(key)
	if err != nil {
		return err
	}

	// Get machine ID for sync
	machineID := ""
	if seed != "" {
		if mid, err := license.MachineID(); err == nil {
			machineID = mid
		}
	}

	result, err := secrets.FullSync(store, seed, machineID)
	if err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	if !result.Online {
		fmt.Println("🔴 Offline — sync unavailable (cPanel unreachable)")
		fmt.Printf("   Pending operations queued: %d\n", result.PendingOps)
		return nil
	}

	fmt.Println("🟢 Online — sync with cPanel complete")
	if result.Uploaded {
		fmt.Println("   ✅ Uploaded vault to cPanel")
	}
	if result.Downloaded && result.Merged > 0 {
		fmt.Printf("   ✅ Merged %d secret(s) from other devices\n", result.Merged)
	} else if result.Downloaded {
		fmt.Println("   ✅ Vault already up-to-date")
	}
	if result.Errors != nil {
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "   ⚠️  %s\n", e)
		}
	}
	return nil
}

// ── REVOKE ─────────────────────────────────────────────────────────────────

func cmdRevoke(key []byte) error {
	if len(os.Args) < 3 {
		return fmt.Errorf("specify secret name: ovav vault revoke <name>")
	}
	secretName := os.Args[2]

	store, err := openStore(key)
	if err != nil {
		return err
	}

	graph, err := secrets.LoadDependencyGraph()
	if err != nil {
		graph = &secrets.DependencyGraph{}
	}

	// Get impact preview first
	refs := graph.GetRefsForSecretByName(store, secretName)
	if len(refs) > 0 {
		fmt.Printf("⚠️  This secret is used by %d system(s):\n", len(refs))
		for _, r := range refs {
			autoTag := ""
			if r.AutoRotatable {
				autoTag = " [AUTO-REVOCABLE]"
			}
			fmt.Printf("  • %s: %s → %s%s\n", r.System, r.Path, r.EnvVar, autoTag)
		}
		fmt.Println()
		fmt.Printf("Run 'ovav vault revoke %s' to confirm revocation.\n", secretName)
		return nil
	}

	// Dry run: show what would happen
	fmt.Printf("🔴 Revoke: %s\n", secretName)
	fmt.Println("═══════════════════════════════════════════════")

	report, err := secrets.RevokeSecret(store, graph, secretName)
	if err != nil {
		return fmt.Errorf("revoke: %w", err)
	}

	// Print results
	ok := 0
	failed := 0
	for _, r := range report.Results {
		icon := "✅"
		if r.Status == "failed" || r.Status == "failed_vault_delete" {
			icon = "❌"
			failed++
		} else {
			ok++
		}
		fmt.Printf("  %s %-15s %-30s %s\n", icon, r.Provider, r.Path, r.Error)
	}

	fmt.Println()
	if report.VaultDeleted {
		fmt.Printf("✅ Deleted from vault: %s\n", secretName)
	} else {
		fmt.Printf("❌ NOT deleted from vault\n")
	}

	if report.DepGraphCleaned {
		fmt.Printf("✅ Dependency graph cleaned\n")
	}

	if failed > 0 {
		fmt.Printf("\n⚠️  %d provider(s) failed — manual action may be required\n", failed)
	}

	return nil
}

// ── ROTATE ──────────────────────────────────────────────────────────────────

func cmdRotate(key []byte) error {
	if len(os.Args) < 3 {
		return fmt.Errorf("specify secret name: ovav vault rotate <name>")
	}
	secretName := os.Args[2]

	store, err := openStore(key)
	if err != nil {
		return err
	}

	sec := store.GetByName(secretName)
	if sec == nil {
		return fmt.Errorf("secret %q not found", secretName)
	}

	fmt.Printf("🔄 Rotating: %s (%s)\n", secretName, sec.Type)
	fmt.Println("═══════════════════════════════════════════════")

	// TODO: Phase 6.9 — generate new value, push to providers, update vault
	// For now, just show what would happen
	refs := secrets.GetRefsForSecretByNameStatic(store, secretName)
	fmt.Printf("  Used by %d system(s):\n", len(refs))
	for _, r := range refs {
		autoTag := ""
		if r.AutoRotatable {
			autoTag = " [AUTO-ROTATABLE]"
		}
		fmt.Printf("    • %s: %s → %s%s\n", r.System, r.Path, r.EnvVar, autoTag)
	}

	fmt.Println()
	fmt.Println("⚠️  Rotation engine (Phase 6.9) not yet implemented.")
	fmt.Println("   Use 'ovav vault add' to add a new secret with a fresh value.")
	return nil
}

// ── QUERY (Natural Language) ───────────────────────────────────────────────────

func cmdQuery(key []byte) error {
	store, err := openStore(key)
	if err != nil {
		return err
	}

	graph, err := secrets.LoadDependencyGraph()
	if err != nil {
		graph = &secrets.DependencyGraph{}
	}

	if len(os.Args) < 3 {
		fmt.Println("OVAV VAULT — Natural Language Query")
		fmt.Println()
		fmt.Println("Usage: ovav vault query \"<question>\"")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  \"what secrets expire next week\"")
		fmt.Println("  \"show me all github tokens\"")
		fmt.Println("  \"which secrets are used in production\"")
		fmt.Println("  \"who used CLOUDFLARE_API_KEY last month\"")
		fmt.Println("  \"list all orphaned secrets\"")
		return nil
	}

	query := strings.Join(os.Args[2:], " ")
	fmt.Printf("🔍 Query: %s\n\n", query)

	// Simple pattern matching — Phase 7 will use NLP
	results := secrets.QuerySecrets(store, graph, query)

	if len(results) == 0 {
		fmt.Println("  No results found.")
		return nil
	}

	for _, r := range results {
		fmt.Printf("  %s  %-30s %s (%s)\n", r.Icon, r.Name, r.Type, r.Detail)
	}

	return nil
}

// ── Airgap Export ─────────────────────────────────────────────────────────────

// cmdAirgapExport exports the vault to an encrypted .airgap file.
func cmdAirgapExport(key []byte) error {
	store, err := openStore(key)
	if err != nil {
		return err
	}
	graph, _ := secrets.LoadDependencyGraph()
	if graph == nil {
		graph = &secrets.DependencyGraph{}
	}

	if len(os.Args) < 3 {
		fmt.Println("OVAV VAULT — Air-Gap Export")
		fmt.Println()
		fmt.Println("Usage: ovav vault export <output.airgap> [flags]")
		fmt.Println()
		fmt.Println("Flags:")
		fmt.Println("  --password <pw>   Additional password protection")
		fmt.Println("  --expires <time>  Expiration time (RFC3339 or duration, e.g. 24h, 7d)")
		fmt.Println()
		fmt.Println("Example:")
		os.Stdout.WriteString("  ovav vault export backup-$(date +%Y%m%d).airgap --expires 30d\n")
		return nil
	}

	outputPath := os.Args[2]
	password := flagGet("--password")
	expStr := flagGet("--expires")

	var expiration time.Time
	if expStr != "" {
		if d, err := time.ParseDuration(expStr); err == nil {
			expiration = time.Now().UTC().Add(d)
		} else if t, err := time.Parse(time.RFC3339, expStr); err == nil {
			expiration = t
		} else {
			return fmt.Errorf("invalid expiration: %s (use 24h, 7d, or RFC3339)", expStr)
		}
	}

	seed := os.Getenv("OVAV_SEED")
	opts := &secrets.ExportOptions{
		Password:   password,
		Expiration: expiration,
	}

	data, err := secrets.ExportToAirgap(store, graph, seed, opts)
	if err != nil {
		return fmt.Errorf("export: %w", err)
	}

	if err := secrets.WriteAirgapFile(outputPath, data, 0600); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	fmt.Printf("✅ Exported to %s (%d bytes)\n", outputPath, len(data))
	if !expiration.IsZero() {
		fmt.Printf("   Expires: %s\n", expiration.Format(time.RFC3339))
	}
	fmt.Printf("   HMAC: BLAKE3-based, seed-derived\n")
	return nil
}

// ── Airgap Import ─────────────────────────────────────────────────────────────

// cmdAirgapImport imports secrets from an encrypted .airgap file.
func cmdAirgapImport(key []byte) error {
	store, err := openStore(key)
	if err != nil {
		return err
	}

	if len(os.Args) < 3 {
		fmt.Println("OVAV VAULT — Air-Gap Import")
		fmt.Println()
		fmt.Println("Usage: ovav vault import <input.airgap> [flags]")
		fmt.Println()
		fmt.Println("Flags:")
		fmt.Println("  --password <pw>   Password (if export was protected)")
		fmt.Println()
		fmt.Println("Example:")
		fmt.Println("  ovav vault import backup-20260803.airgap")
		return nil
	}

	inputPath := os.Args[2]
	password := flagGet("--password")

	seed := os.Getenv("OVAV_SEED")

	// Verify HMAC before attempting import
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	valid, err := secrets.VerifyHMAC(data, seed)
	if err != nil || !valid {
		return fmt.Errorf("HMAC verification failed: invalid seed or corrupted file")
	}

	// Import using AirgapHandle
	h := secrets.NewAirgapHandle(inputPath)
	result, err := h.Import(store, seed, password)
	if err != nil {
		return fmt.Errorf("import: %w", err)
	}

	af, _ := secrets.ReadAirgapFile(inputPath)
	if af != nil && af.Parsed != nil && af.Parsed.ExpiresAt != "" {
		if result.Expired {
			fmt.Printf("⚠️  Package expired on %s\n", af.Parsed.ExpiresAt)
		}
	}

	fmt.Printf("✅ Imported %d secrets (%d skipped, %d errors)\n",
		result.SecretsImported, result.SecretsSkipped, len(result.Errors))
	fmt.Printf("   Origin device: %s\n", result.OriginDevice)
	fmt.Printf("   Imported at:   %s\n", result.ImportedAt)
	if result.HadExpiration {
		fmt.Printf("   Expired:       %v\n", result.Expired)
	}
	for _, e := range result.Errors {
		fmt.Printf("   ❌ %s\n", e)
	}
	return nil
}

// flagGet gets a flag value from os.Args.
func flagGet(name string) string {
	for i, arg := range os.Args {
		if arg == name && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
		if strings.HasPrefix(arg, name+"=") {
			return strings.TrimPrefix(arg, name+"=")
		}
	}
	return ""
}

// ═══════════════════════════════════════════════════════════════════════════
// MAIN
// ═══════════════════════════════════════════════════════════════════════════

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(0)
	}

	cmd := os.Args[1]
	if cmd == "help" || cmd == "--help" {
		fmt.Print(usage)
		os.Exit(0)
	}

	// Parse global flags (before command)
	flag.CommandLine.Parse(os.Args[2:])
	// After this, *flagName, *flagSecretType, etc. are set from os.Args[2:]

	key, err := loadKey()
	if err != nil {
		// For "add", we can show error after flag check
		if cmd != "add" || *flagName == "" {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}
		key = nil
	}

	var exitCode int
	switch cmd {
	case "add":
		if err := cmdAdd(key); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			exitCode = 1
		}
	case "list":
		if err := cmdList(key); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			exitCode = 1
		}
	case "get":
		if err := cmdGet(key); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			exitCode = 1
		}
	case "remove", "delete":
		if err := cmdRemove(key); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			exitCode = 1
		}
	case "discover":
		if err := cmdDiscover(key); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			exitCode = 1
		}
	case "health":
		if err := cmdHealth(key); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			exitCode = 1
		}
	case "backup":
		if err := cmdBackup(key); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			exitCode = 1
		}
	case "restore":
		if err := cmdRestore(key); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			exitCode = 1
		}
	case "connect":
		// Manually parse flags after sub-subcommand since Go flag parser
		// stops at first positional arg (the sub-subcommand "add", "list", etc.)
		subCmd := parseConnectFlags()
		if err := cmdConnect(key, subCmd); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			exitCode = 1
		}
	case "sync":
		if err := cmdSync(key); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			exitCode = 1
		}
	case "deps":
		if err := cmdDeps(key); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			exitCode = 1
		}
	case "revoke":
		if err := cmdRevoke(key); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			exitCode = 1
		}
	case "rotate":
		if err := cmdRotate(key); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			exitCode = 1
		}
	case "query":
		if err := cmdQuery(key); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			exitCode = 1
		}
	case "export":
		if err := cmdAirgapExport(key); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			exitCode = 1
		}
	case "import":
		if err := cmdAirgapImport(key); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			exitCode = 1
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n%s", cmd, usage)
		exitCode = 1
	}

	os.Exit(exitCode)
}
