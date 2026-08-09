package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/cli"
	"github.com/ovav/ovav/internal/identity"
)

// ── Waiver configuration — canonical, not magic numbers ──────────────────────

const (
	waiverSchema     = "ovav.protected_branch_waiver.v2"
	maxWaiverMinutes = 60
	waiverIDPrefix   = "waiver-"
	grantedRole      = "ceo"
	grantedMinLevel  = 10
)

// ── Subcommand names — single source of truth ────────────────────────────────

const (
	waiverCmdStatus = "status"
	waiverCmdRevoke = "revoke"
	waiverCmdCreate = "create"
)

// ── Flag names — no magic strings in argument parsing ────────────────────────

const (
	flagReason  = "--reason"
	flagBranch  = "--branch"
	flagMins    = "--mins"
	flagMinutes = "--minutes"
)

// ── Audit actions ────────────────────────────────────────────────────────────

const (
	auditActionCreated = "waiver_created"
	auditActionRevoked = "waiver_revoked"
)

// ── File layout ──────────────────────────────────────────────────────────────

const (
	waiverRelDir    = ".ovav"
	waiverRelSubDir = "runtime"
	waiverFileName  = "protected_branch_waiver.yaml"
	auditFileName   = "waiver_audit.jsonl"
)

// ── Help strings — a single block is easier to translate later ───────────────

const (
	helpLong  = "help"
	helpShort = "--help"
	helpDash  = "-h"
)

// ── waiverRecord — sealed evidence that the CEO authorised a branch ──────────

type waiverRecord struct {
	Schema          string `json:"schema"`
	ID              string `json:"id"`
	Active          bool   `json:"active"`
	Branch          string `json:"branch"`
	Reason          string `json:"reason"`
	IdentityID      string `json:"identity_id"`
	IdentityName    string `json:"identity_name"`
	IdentityRole    string `json:"identity_role"`
	IdentityLevel   int    `json:"identity_level"`
	MachineID       string `json:"machine_id"`
	SessionCreated  string `json:"session_created_at"`
	GrantedAt       string `json:"granted_at"`
	ExpiresAt       string `json:"expires_at"`
	DurationMinutes int    `json:"duration_minutes"`
	Nonce           string `json:"nonce"`
	Signature       string `json:"signature"`
}

// ── Dispatcher — every unrecognised token is a motivo, never rejected ────────

func cmdWaiver(args []string) int {
	if len(args) == 0 || isHelpArg(args[0]) {
		printWaiverHelp()
		return 0
	}

	switch args[0] {
	case waiverCmdStatus:
		return waiverStatus()
	case waiverCmdRevoke:
		return waiverRevoke()
	case waiverCmdCreate:
		// Explicit "create" subcommand — strip it and treat the rest as motivo.
		return waiverCreate(args[1:])
	default:
		// Anything the user types is a motivo.  No static validation.
		return waiverCreate(args)
	}
}

func printWaiverHelp() {
	fmt.Println("ovav waiver — Waiver inteligente de seguridad")
	fmt.Println()
	fmt.Printf("  ovav waiver <motivo>                  Crea un waiver por %d minutos\n", maxWaiverMinutes)
	fmt.Printf("  ovav waiver <motivo> %s <%d-%d>  Ajusta la duración\n", flagMins, 1, maxWaiverMinutes)
	fmt.Printf("  ovav waiver %s                    Verifica identidad, firma y vigencia\n", waiverCmdStatus)
	fmt.Printf("  ovav waiver %s                    Revoca el waiver activo\n", waiverCmdRevoke)
	fmt.Println()
	fmt.Printf("Requiere una sesión OVAV autenticada con rol %s y nivel %d.\n", grantedRole, grantedMinLevel)
}

// ── Create ────────────────────────────────────────────────────────────────────

func waiverCreate(args []string) int {
	reason, branch, mins, err := parseWaiverCreateArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		fmt.Fprintf(os.Stderr, "Uso: ovav waiver <motivo> [%s <rama>] [%s <%d-%d>]\n",
			flagBranch, flagMins, 1, maxWaiverMinutes)
		return 2
	}

	repoRoot := cli.MustFindRepoRoot()
	sess, id, err := authenticatedWaiverIdentity(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "🔒 Waiver denegado: %v\n", err)
		return 1
	}

	if branch == "" {
		branch, _, _ = cli.GitInfo()
		if branch == "" || branch == "unknown" {
			fmt.Fprintf(os.Stderr, "❌ No se pudo detectar la rama actual; use %s.\n", flagBranch)
			return 1
		}
	}

	now := time.Now().UTC()
	nonce, err := waiverNonce()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ No se pudo generar el nonce: %v\n", err)
		return 1
	}

	record := waiverRecord{
		Schema:          waiverSchema,
		ID:              fmt.Sprintf("%s%d", waiverIDPrefix, now.UnixNano()),
		Active:          true,
		Branch:          branch,
		Reason:          reason,
		IdentityID:      id.ID,
		IdentityName:    id.Name,
		IdentityRole:    id.Role,
		IdentityLevel:   id.Level,
		MachineID:       sess.MachineID,
		SessionCreated:  sess.CreatedAt,
		GrantedAt:       now.Format(time.RFC3339Nano),
		ExpiresAt:       now.Add(time.Duration(mins) * time.Minute).Format(time.RFC3339Nano),
		DurationMinutes: mins,
		Nonce:           nonce,
	}
	if record.Signature, err = signWaiverRecord(record, sess.VaultKeyHash); err != nil {
		fmt.Fprintf(os.Stderr, "❌ No se pudo firmar el waiver: %v\n", err)
		return 1
	}

	path := waiverPath(repoRoot)
	if err := writeWaiverRecord(path, record); err != nil {
		fmt.Fprintf(os.Stderr, "❌ No se pudo escribir el waiver: %v\n", err)
		return 1
	}
	if err := appendWaiverAudit(repoRoot, auditActionCreated, record); err != nil {
		_ = os.Remove(path)
		fmt.Fprintf(os.Stderr, "❌ Auditoría falló; waiver revertido: %v\n", err)
		return 1
	}

	fmt.Printf("🟢 Waiver creado para '%s' (%d min)\n", branch, mins)
	fmt.Printf("   Motivo:    %s\n", reason)
	fmt.Printf("   Identidad: %s [%s · Level %d]\n", id.Name, strings.ToUpper(id.Role), id.Level)
	fmt.Printf("   ID:        %s\n", record.ID)
	fmt.Printf("   Expira:    %s\n", record.ExpiresAt)
	return 0
}

// ── Status ────────────────────────────────────────────────────────────────────

func waiverStatus() int {
	repoRoot := cli.MustFindRepoRoot()
	record, err := readWaiverRecord(waiverPath(repoRoot))
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No hay waiver activo.")
			return 0
		}
		fmt.Fprintf(os.Stderr, "❌ Waiver inválido: %v\n", err)
		return 1
	}

	sess, id, authErr := authenticatedWaiverIdentity(repoRoot)
	if validErr := validateWaiverRecord(record, sess, id, authErr); validErr != nil {
		fmt.Printf("🔴 Waiver INACTIVO: %v\n", validErr)
		printWaiverRecord(record)
		return 1
	}

	fmt.Println("🟢 Waiver ACTIVO y verificado")
	printWaiverRecord(record)
	return 0
}

func printWaiverRecord(record waiverRecord) {
	fmt.Printf("   Rama:      %s\n", record.Branch)
	fmt.Printf("   Motivo:    %s\n", record.Reason)
	fmt.Printf("   Identidad: %s [%s · Level %d]\n", record.IdentityName,
		strings.ToUpper(record.IdentityRole), record.IdentityLevel)
	fmt.Printf("   ID:        %s\n", record.ID)
	fmt.Printf("   Expira:    %s\n", record.ExpiresAt)
}

// ── Revoke ────────────────────────────────────────────────────────────────────

func waiverRevoke() int {
	repoRoot := cli.MustFindRepoRoot()
	sess, id, err := authenticatedWaiverIdentity(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "🔒 Revocación denegada: %v\n", err)
		return 1
	}
	path := waiverPath(repoRoot)
	record, err := readWaiverRecord(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No hay waiver activo para revocar.")
			return 0
		}
		fmt.Fprintf(os.Stderr, "❌ No se pudo leer el waiver: %v\n", err)
		return 1
	}
	if record.IdentityID != id.ID || record.MachineID != sess.MachineID {
		fmt.Fprintln(os.Stderr, "🔒 Revocación denegada: la identidad o máquina no coincide con el otorgante.")
		return 1
	}
	if err := appendWaiverAudit(repoRoot, auditActionRevoked, record); err != nil {
		fmt.Fprintf(os.Stderr, "❌ No se pudo auditar la revocación: %v\n", err)
		return 1
	}
	if err := os.Remove(path); err != nil {
		fmt.Fprintf(os.Stderr, "❌ No se pudo revocar el waiver: %v\n", err)
		return 1
	}
	fmt.Printf("✅ Waiver %s revocado por %s.\n", record.ID, id.Name)
	return 0
}

// ── Argument parser — flag names come from constants, motivo is fully free ────

func parseWaiverCreateArgs(args []string) (reason, branch string, mins int, err error) {
	mins = maxWaiverMinutes
	var reasonParts []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == flagReason:
			if i+1 >= len(args) {
				return "", "", 0, fmt.Errorf("%s requiere un valor", flagReason)
			}
			reasonParts = append(reasonParts, args[i+1])
			i++
		case a == flagBranch:
			if i+1 >= len(args) {
				return "", "", 0, fmt.Errorf("%s requiere un valor", flagBranch)
			}
			branch = args[i+1]
			i++
		case a == flagMins || a == flagMinutes:
			if i+1 >= len(args) {
				return "", "", 0, fmt.Errorf("%s requiere un valor", a)
			}
			mins, err = strconv.Atoi(args[i+1])
			if err != nil {
				return "", "", 0, fmt.Errorf("duración inválida %q", args[i+1])
			}
			i++
		case strings.HasPrefix(a, "--"):
			return "", "", 0, fmt.Errorf("argumento desconocido %q", a)
		default:
			reasonParts = append(reasonParts, a)
		}
	}
	reason = strings.TrimSpace(strings.Join(reasonParts, " "))
	if reason == "" {
		return "", "", 0, fmt.Errorf("el motivo es obligatorio")
	}
	if mins < 1 || mins > maxWaiverMinutes {
		return "", "", 0, fmt.Errorf("la duración debe estar entre 1 y %d minutos", maxWaiverMinutes)
	}
	return reason, branch, mins, nil
}

// ── Identity gate — reads live session, validates against registry ───────────

func authenticatedWaiverIdentity(repoRoot string) (Session, *identity.Identity, error) {
	sess, ok := loadSession()
	if !ok || time.Since(sess.createdAt()) > sessionTTL {
		return Session{}, nil, fmt.Errorf("se requiere una sesión activa; ejecute `ovav login`")
	}
	reg, err := identity.LoadRegistry(repoRoot)
	if err != nil {
		return Session{}, nil, fmt.Errorf("registro de identidades no disponible: %w", err)
	}
	id, err := identity.FindIdentity(reg, sess.VaultKeyHash)
	if err != nil {
		return Session{}, nil, fmt.Errorf("la sesión no coincide con una identidad activa: %w", err)
	}
	if sess.IdentityID != id.ID || sess.Name != id.Name || sess.Role != id.Role || sess.Level != id.Level {
		return Session{}, nil, fmt.Errorf("la sesión no coincide con el registro canónico")
	}
	if !strings.EqualFold(id.Role, grantedRole) || id.Level < grantedMinLevel {
		return Session{}, nil, fmt.Errorf("solo una identidad %s nivel %d puede emitir waivers", grantedRole, grantedMinLevel)
	}
	if sess.MachineID == "" || sess.VaultKeyHash == "" {
		return Session{}, nil, fmt.Errorf("la sesión carece de trazabilidad criptográfica")
	}
	return sess, id, nil
}

// ── Validation — every field checked against live state ──────────────────────

func validateWaiverRecord(record waiverRecord, sess Session, id *identity.Identity, authErr error) error {
	if authErr != nil {
		return authErr
	}
	if record.Schema != waiverSchema || !record.Active {
		return fmt.Errorf("schema o estado inválido")
	}
	granted, err := time.Parse(time.RFC3339Nano, record.GrantedAt)
	if err != nil {
		return fmt.Errorf("fecha de emisión inválida")
	}
	expires, err := time.Parse(time.RFC3339Nano, record.ExpiresAt)
	if err != nil {
		return fmt.Errorf("fecha de expiración inválida")
	}
	if record.DurationMinutes < 1 || record.DurationMinutes > maxWaiverMinutes || expires.Sub(granted) > time.Hour {
		return fmt.Errorf("TTL fuera del límite de %d minutos", maxWaiverMinutes)
	}
	if time.Now().UTC().After(expires) {
		return fmt.Errorf("expiró en %s", record.ExpiresAt)
	}
	if record.IdentityID != id.ID || record.MachineID != sess.MachineID || record.SessionCreated != sess.CreatedAt {
		return fmt.Errorf("waiver no vinculado a la sesión autenticada actual")
	}
	expected, err := signWaiverRecord(record, sess.VaultKeyHash)
	if err != nil || !hmac.Equal([]byte(record.Signature), []byte(expected)) {
		return fmt.Errorf("firma inválida o artefacto manipulado")
	}
	branch, _, _ := cli.GitInfo()
	if branch != record.Branch {
		return fmt.Errorf("waiver emitido para %s, rama actual %s", record.Branch, branch)
	}
	return nil
}

// ── HMAC signer — binds every field with the session's vault-key hash ────────

func signWaiverRecord(record waiverRecord, keyHash string) (string, error) {
	key, err := hex.DecodeString(keyHash)
	if err != nil || len(key) != sha256.Size {
		return "", fmt.Errorf("hash de identidad inválido")
	}
	payload := strings.Join([]string{
		record.Schema, record.ID, record.Branch, record.Reason, record.IdentityID,
		record.IdentityName, record.IdentityRole, strconv.Itoa(record.IdentityLevel),
		record.MachineID, record.SessionCreated, record.GrantedAt, record.ExpiresAt,
		strconv.Itoa(record.DurationMinutes), record.Nonce,
	}, "\x00")
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// ── I/O helpers — atomic write, append-only audit ────────────────────────────

func waiverNonce() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func waiverPath(repoRoot string) string {
	return filepath.Join(repoRoot, waiverRelDir, waiverRelSubDir, waiverFileName)
}

func writeWaiverRecord(path string, record waiverRecord) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readWaiverRecord(path string) (waiverRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return waiverRecord{}, err
	}
	var record waiverRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return waiverRecord{}, fmt.Errorf("formato legado o corrupto: %w", err)
	}
	return record, nil
}

func appendWaiverAudit(repoRoot, action string, record waiverRecord) error {
	path := filepath.Join(repoRoot, waiverRelDir, waiverRelSubDir, auditFileName)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	entry := map[string]interface{}{
		"timestamp":      time.Now().UTC().Format(time.RFC3339Nano),
		"action":         action,
		"waiver_id":      record.ID,
		"branch":         record.Branch,
		"reason":         record.Reason,
		"identity_id":    record.IdentityID,
		"identity_name":  record.IdentityName,
		"identity_role":  record.IdentityRole,
		"identity_level": record.IdentityLevel,
		"machine_id":     record.MachineID,
		"expires_at":     record.ExpiresAt,
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

func isHelpArg(arg string) bool {
	return arg == helpLong || arg == helpShort || arg == helpDash
}
