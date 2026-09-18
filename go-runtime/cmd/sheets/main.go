// sheets CLI — entry point for `go run -C go-runtime ./cmd/sheets`.
//
// Subcommands:
//   auth   — exchange OAuth2 code for refresh_token (interactive, 1-shot)
//   demo   — create a tab named after the current user + write a 2x2 header
//   list   — list tabs in the bound spreadsheet
//   read   — read a range
//   write  — write a range (USER_ENTERED by default)
//
// Default spreadsheet ID is taken from OVAV_SHEETS_SPREADSHEET env var,
// falling back to the allowlist first entry.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)
const (
	defaultSpreadsheetID = "1MQ3wts_cEG_4Dp5U6X4gehEn6u7F61tIl7KHzSzKw0Y"
	defaultRedirectURI    = "http://127.0.0.1:9876/callback"
)
func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	repoRoot, err := findRepoRoot()
	if err != nil {
		die(err)
	}
	// Load allowlist before doing anything — every operation gates on it.
	if err := loadAllowlist(allowlistPath(repoRoot)); err != nil {
		die(err)
	}
	switch cmd {
	case "auth":
		exitOn(runAuth(repoRoot, args))
	case "list":
		exitOn(runList(repoRoot, args))
	case "read":
		exitOn(runRead(repoRoot, args))
	case "table":
		exitOn(runReadTable(repoRoot, args))
	case "write":
		exitOn(runWrite(repoRoot, args))
	case "append":
		exitOn(runAppend(repoRoot, args))
	case "update":
		exitOn(runUpdate(repoRoot, args))
	case "create-tab":
		exitOn(runCreateTab(repoRoot, args))
	case "snapshots":
		exitOn(runSnapshots(repoRoot, args))
	case "rollback":
		exitOn(runRollback(args))
	case "xlsx-in":
		exitOn(runXlsxIn(repoRoot, args))
	case "xlsx-out":
		exitOn(runXlsxOut(repoRoot, args))
	case "demo":
		exitOn(runDemo(repoRoot, args))
	case "mcp":
		exitOn(runMCPServer(repoRoot))
	case "allowlist":
		exitOn(runAllowlistCmd(args))
	case "version":
		fmt.Println("ovav-sheets 0.2.0")
	default:
		usage()
		os.Exit(2)
	}
}
func usage() {
	fmt.Fprintln(os.Stderr, `Usage: ovav-sheets <command> [flags]
OAuth & access:
  auth         Run OAuth2 web flow, store refresh_token in vault
  allowlist    list|add|remove spreadsheets
Reads:
  list         List tabs in the bound spreadsheet
  read         Read A1 range (--range)
  table        Read tab as JSON rows (--tab)
Writes:
  write        Write A1 range (--range, --values-json)
  append       Append row to a table (--tab, --json or --k=v)
  update       Update rows where column=value (--tab, --where col=val, --set col=val)
  create-tab   Create a new tab (--title)
Safety:
  snapshots    List captured snapshots (--spreadsheet, --tab)
  rollback     Restore a snapshot (--id)
Excel:
  xlsx-in      Import a .xlsx file into a tab (--tab, --path)
  xlsx-out     Export a tab to .xlsx (--tab, --path)
Tools:
  mcp          Run MCP server on stdio (JSON-RPC 2.0)
  demo         Create a tab named after the user, write a header cell
  version      Print version`)
}
func runAllowlistCmd(args []string) error {
	if len(args) == 0 {
		return cmdAllowlistList(nil)
	}
	switch args[0] {
	case "list":
		return cmdAllowlistList(args[1:])
	case "add":
		return cmdAllowlistAdd(args[1:])
	case "remove":
		return cmdAllowlistRemove(args[1:])
	}
	return fmt.Errorf("allowlist: unknown subcommand %q", args[0])
}
func allowlistName(id string) string {
	globalAllowlist.mu.RLock()
	defer globalAllowlist.mu.RUnlock()
	if e, ok := globalAllowlist.entries[id]; ok {
		return e.Name
	}
	return "(unknown)"
}
func runAuth(repoRoot string, args []string) error {
	var clientID, clientSecret, projectID, redirectURI string
	var manualCode string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--client-id":
			i++; clientID = args[i]
		case "--client-secret":
			i++; clientSecret = args[i]
		case "--project-id":
			i++; projectID = args[i]
		case "--redirect-uri":
			i++; redirectURI = args[i]
		case "--code":
			i++; manualCode = args[i]
		case "--from-stdin":
			b, err := readJSONCreds()
			if err != nil {
				return err
			}
			clientID, clientSecret, projectID = b.ID, b.Secret, b.Project
		}
	}
	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("auth: --client-id and --client-secret required (or --from-stdin)")
	}
	if redirectURI == "" {
		redirectURI = defaultRedirectURI
	}
	stateBytes := make([]byte, 16)
	_, _ = rand.Read(stateBytes)
	state := hex.EncodeToString(stateBytes)
	authURL := AuthCodeURL(clientID, redirectURI, state)
	fmt.Println("=========================================================")
	fmt.Println(" OVAV Sheets — OAuth2 consent (1-shot)")
	fmt.Println("=========================================================")
	fmt.Println()
	fmt.Println("Opening browser to Google sign-in…")
	fmt.Println("(if it does not open, copy the URL below into any browser)")
	fmt.Println()
	fmt.Println(authURL)
	fmt.Println()
	if err := openBrowser(authURL); err != nil {
		fmt.Fprintf(os.Stderr, "(could not auto-open browser: %v)\n", err)
	}
	// Path A: auto-capture via local HTTP server on :9876.
	// The browser hits /callback?code=… and we exchange + store.
	// Path B (--code <value>): manual paste if the server cannot bind.
	var code string
	if manualCode == "" {
		captured, err := awaitCallback(redirectURI, state, 120*time.Second)
		if err != nil {
			return err
		}
		code = captured
	} else {
		code = manualCode
	}
	tok, err := ExchangeCode(clientID, clientSecret, redirectURI, strings.TrimSpace(code))
	if err != nil {
		return err
	}
	tok.ClientID = clientID
	tok.ClientSecret = clientSecret
	tok.ProjectID = projectID
	tok.RedirectURI = redirectURI
	store := NewCredStore(repoRoot)
	if err := store.Save(tok); err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("✅ refresh_token stored in:", storePath(repoRoot))
	fmt.Println("   expires_in:", "3600s", "scope:", tok.Scope)
	return nil
}
func runList(repoRoot string, args []string) error {
	id := pickSpreadsheetID(args)
	if err := assertAllowed(id); err != nil {
		return err
	}
	store := NewCredStore(repoRoot)
	creds, err := store.Load()
	if err != nil {
		return err
	}
	cl := NewClient(creds, id)
	sheets2, err := cl.ListSheets()
	if err != nil {
		return err
	}
		fmt.Printf("Spreadsheet: %s (%s)\n", id, allowlistName(id))
	fmt.Printf("Tabs (%d):\n", len(sheets2))
	for _, s := range sheets2 {
		fmt.Printf("  • %s  [sheetId=%d  %dx%d]\n", s.Title, s.SheetID, s.Rows, s.Cols)
	}
	return nil
}
func runRead(repoRoot string, args []string) error {
	id := pickSpreadsheetID(args)
	rng := flagValue(args, "--range", "A1:Z100")
	if err := assertAllowed(id); err != nil {
		return err
	}
	store := NewCredStore(repoRoot)
	creds, err := store.Load()
	if err != nil {
		return err
	}
	cl := NewClient(creds, id)
	vr, err := cl.ReadRange(rng)
	if err != nil {
		return err
	}
	fmt.Printf("Range: %s  rows=%d\n", vr.Range, len(vr.Values))
	for i, row := range vr.Values {
		fmt.Printf("  %2d: %s\n", i+1, strings.Join(row, " | "))
	}
	return nil
}
func runWrite(repoRoot string, args []string) error {
	id := pickSpreadsheetID(args)
	rng := flagValue(args, "--range", "A1")
	valuesJSON := flagValue(args, "--values-json", "")
	if valuesJSON == "" {
		return fmt.Errorf("write: --values-json required, e.g. '[[\"A\",\"B\"],[\"1\",\"2\"]]'")
	}
	values, err := parseMatrix(valuesJSON)
	if err != nil {
		return err
	}
	if err := assertAllowed(id); err != nil {
		return err
	}
	store := NewCredStore(repoRoot)
	creds, err := store.Load()
	if err != nil {
		return err
	}
	cl := NewClient(creds, id)
	ur, err := cl.WriteValues(rng, values, "USER_ENTERED")
	if err != nil {
		return err
	}
	fmt.Printf("✅ wrote %d cells (%d rows × %d cols) at %s\n",
		ur.UpdatedCells, ur.UpdatedRows, ur.UpdatedColumns, ur.UpdatedRange)
	return nil
}
// runDemo is the end-to-end proof: add a tab named after the user,
// write a header row, log every step.
func runDemo(repoRoot string, args []string) error {
	id := pickSpreadsheetID(args)
	userName := flagValue(args, "--user", "Alexander Salvador")
	tabName := flagValue(args, "--tab", fmt.Sprintf("OVAV — %s", userName))
	if err := assertAllowed(id); err != nil {
		return err
	}
	store := NewCredStore(repoRoot)
	creds, err := store.Load()
	if err != nil {
		return err
	}
	cl := NewClient(creds, id)
	fmt.Println("──────────────────────────────────────────")
	fmt.Println(" OVAV Sheets DEMO — single end-to-end run")
	fmt.Println("──────────────────────────────────────────")
	fmt.Printf("  spreadsheet : %s\n", id)
	fmt.Printf("  user        : %s\n", userName)
	fmt.Printf("  tab name    : %s\n", tabName)
	fmt.Printf("  timestamp   : %s\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Println()
	existing, err := cl.ListSheets()
	if err != nil {
		return err
	}
	fmt.Printf("📋 existing tabs (%d):\n", len(existing))
	for _, s := range existing {
		fmt.Printf("   • %s [id=%d]\n", s.Title, s.SheetID)
	}
	sheetID, err := cl.AddSheet(tabName, map[string]float64{
		"red": 0.145, "green": 0.388, "blue": 0.922, // OVAV blue
	})
	if err != nil {
		return err
	}
	fmt.Printf("\n✅ tab created: %q [sheetId=%d]\n", tabName, sheetID)
	// Header row at A1, message at A2
	header := [][]string{
		{fmt.Sprintf("OVAV · %s · %s", userName, time.Now().UTC().Format("2006-01-02 15:04 UTC"))},
		{"hello@ovav.dev · owned by OVAV Sheets Bridge"},
	}
	ur, err := cl.WriteValues(fmt.Sprintf("'%s'!A1:A2", tabName), header, "USER_ENTERED")
	if err != nil {
		return err
	}
	fmt.Printf("✅ header written: %d cells at %s\n", ur.UpdatedCells, ur.UpdatedRange)
	fmt.Println()
	fmt.Println("🎉 DEMO OK — open the spreadsheet to see the new tab.")
	return nil
}
// ── v0.2 commands ───────────────────────────────────────────────────
func runReadTable(repoRoot string, args []string) error {
	id := pickSpreadsheetID(args)
	tab := flagValue(args, "--tab", "")
	if tab == "" {
		return fmt.Errorf("table: --tab required")
	}
	if err := assertAllowed(id); err != nil {
		return err
	}
	store := NewCredStore(repoRoot)
	creds, err := store.Load()
	if err != nil {
		return err
	}
	cl := NewClient(creds, id)
	t, err := cl.LoadTable(tab)
	if err != nil {
		return err
	}
	out, _ := json.MarshalIndent(map[string]any{
		"headers": t.Header(),
		"rows":    t.AsMaps(),
	}, "", "  ")
	fmt.Println(string(out))
	return nil
}
func runAppend(repoRoot string, args []string) error {
	id := pickSpreadsheetID(args)
	tab := flagValue(args, "--tab", "")
	if tab == "" {
		return fmt.Errorf("append: --tab required")
	}
	if err := assertAllowed(id); err != nil {
		return err
	}
	store := NewCredStore(repoRoot)
	creds, err := store.Load()
	if err != nil {
		return err
	}
	cl := NewClient(creds, id)
	values, err := parseKVArgs(args, "--tab", "--spreadsheet")
	if err != nil {
		return err
	}
	if pre, err := cl.ReadRange(tab + "!A1:Z10000"); err == nil {
		if b, err := json.Marshal(pre); err == nil {
			_, _ = cl.Capture(repoRoot, "append-pre", "before append on "+tab, tab, b)
		}
	}
	t, err := cl.LoadTable(tab)
	if err != nil {
		return err
	}
	rowIdx, a1, err := t.AppendRow(values)
	if err != nil {
		return err
	}
	audit(repoRoot, id, tab, "append_row", map[string]any{"values": values}, rowIdx)
	fmt.Printf("✅ appended row %d at %s\n", rowIdx, a1)
	return nil
}
func runUpdate(repoRoot string, args []string) error {
	id := pickSpreadsheetID(args)
	tab := flagValue(args, "--tab", "")
	wCol := flagValue(args, "--where-col", "")
	wVal := flagValue(args, "--where-val", "")
	sCol := flagValue(args, "--set-col", "")
	sVal := flagValue(args, "--set-val", "")
	if tab == "" || wCol == "" || sCol == "" {
		return fmt.Errorf("update: --tab, --where-col, --where-val, --set-col, --set-val required")
	}
	if err := assertAllowed(id); err != nil {
		return err
	}
	store := NewCredStore(repoRoot)
	creds, err := store.Load()
	if err != nil {
		return err
	}
	cl := NewClient(creds, id)
	if pre, err := cl.ReadRange(tab + "!A1:Z10000"); err == nil {
		if b, err := json.Marshal(pre); err == nil {
			_, _ = cl.Capture(repoRoot, "update-pre", fmt.Sprintf("before update where %s=%s on %s", wCol, wVal, tab), tab, b)
		}
	}
	t, err := cl.LoadTable(tab)
	if err != nil {
		return err
	}
	n, err := t.UpdateWhere(wCol, wVal, sCol, sVal)
	if err != nil {
		return err
	}
	audit(repoRoot, id, tab, "update_where",
		map[string]any{"where_col": wCol, "where_val": wVal, "set_col": sCol, "set_val": sVal}, n)
	fmt.Printf("✅ updated %d row(s)\n", n)
	return nil
}
func runCreateTab(repoRoot string, args []string) error {
	id := pickSpreadsheetID(args)
	title := flagValue(args, "--title", "")
	if title == "" {
		return fmt.Errorf("create-tab: --title required")
	}
	if err := assertAllowed(id); err != nil {
		return err
	}
	store := NewCredStore(repoRoot)
	creds, err := store.Load()
	if err != nil {
		return err
	}
	cl := NewClient(creds, id)
	shID, err := cl.AddSheet(title, map[string]float64{"red": 0.145, "green": 0.388, "blue": 0.922})
	if err != nil {
		return err
	}
	audit(repoRoot, id, title, "create_tab", map[string]any{"title": title}, int(shID))
	fmt.Printf("✅ created tab %q [sheetId=%d]\n", title, shID)
	return nil
}
func runSnapshots(repoRoot string, args []string) error {
	id := flagValue(args, "--spreadsheet", "")
	tab := flagValue(args, "--tab", "")
	recs, err := ListSnapshots(repoRoot, id, tab)
	if err != nil {
		return err
	}
	fmt.Printf("Snapshots (%d):\n", len(recs))
	for _, r := range recs {
		fmt.Printf("  • %s  op=%s  sheet=%s/%s  cells=%d  %s\n",
			r.ID, r.Operation, r.SpreadsheetID, r.Tab, r.CellCount, r.CreatedAt.Format(time.RFC3339))
	}
	return nil
}
func runRollback(args []string) error {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}
	id := flagValue(args, "--id", "")
	if id == "" {
		return fmt.Errorf("rollback: --id required")
	}
	return Rollback(repoRoot, id)
}
func runXlsxIn(repoRoot string, args []string) error {
	id := pickSpreadsheetID(args)
	tab := flagValue(args, "--tab", "")
	path := flagValue(args, "--path", "")
	if tab == "" || path == "" {
		return fmt.Errorf("xlsx-in: --tab and --path required")
	}
	if err := assertAllowed(id); err != nil {
		return err
	}
	store := NewCredStore(repoRoot)
	creds, err := store.Load()
	if err != nil {
		return err
	}
	cl := NewClient(creds, id)
	sheets2, err := ReadXlsx(path)
	if err != nil {
		return err
	}
	if len(sheets2) == 0 {
		return fmt.Errorf("xlsx: empty file")
	}
	if _, err := cl.AddSheet(tab, nil); err != nil {
		_ = err
	}
	rows := XlsxToCells(sheets2[0])
	if len(rows) == 0 {
		return fmt.Errorf("xlsx: no rows")
	}
	a1 := fmt.Sprintf("%s!A1", quoted(tab))
	ur, err := cl.WriteValues(a1, rows, "USER_ENTERED")
	if err != nil {
		return err
	}
	audit(repoRoot, id, tab, "xlsx_in", map[string]any{"path": path, "rows": len(rows)}, ur.UpdatedCells)
	fmt.Printf("✅ imported %d rows (%d cells) from %s into %s\n", len(rows), ur.UpdatedCells, path, tab)
	return nil
}
func runXlsxOut(repoRoot string, args []string) error {
	id := pickSpreadsheetID(args)
	tab := flagValue(args, "--tab", "")
	path := flagValue(args, "--path", "")
	if tab == "" || path == "" {
		return fmt.Errorf("xlsx-out: --tab and --path required")
	}
	if err := assertAllowed(id); err != nil {
		return err
	}
	store := NewCredStore(repoRoot)
	creds, err := store.Load()
	if err != nil {
		return err
	}
	cl := NewClient(creds, id)
	t, err := cl.LoadTable(tab)
	if err != nil {
		return err
	}
	cells := []XlsxCell{}
	for i, h := range t.Header() {
		cells = append(cells, XlsxCell{Ref: fmt.Sprintf("%s%d", colLetter(i+1), 1), Val: h})
	}
	maxR := 1
	maxC := len(t.Header())
	for ri, r := range t.Rows() {
		for ci, v := range r {
			cells = append(cells, XlsxCell{Ref: fmt.Sprintf("%s%d", colLetter(ci+1), ri+2), Val: v})
		}
		if ri+2 > maxR {
			maxR = ri + 2
		}
	}
	data, err := WriteXlsx(tab, cells, maxR, maxC)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	audit(repoRoot, id, tab, "xlsx_out", map[string]any{"path": path, "cells": len(cells)}, len(cells))
	fmt.Printf("✅ wrote %d cells to %s\n", len(cells), path)
	return nil
}
// parseKVArgs turns --key value pairs (or --json '{...}') into a map.
// `reserved` lists flags that belong to the command (not the data row)
// and should be ignored when harvesting key/value pairs.
func parseKVArgs(args []string, reserved ...string) (map[string]string, error) {
	out := map[string]string{}
	res := map[string]bool{}
	for _, r := range reserved {
		res[r] = true
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("--json requires a value")
			}
			var m map[string]string
			if err := jsonUnmarshalString(args[i], &m); err != nil {
				return nil, err
			}
			for k, v := range m {
				out[k] = v
			}
		default:
			if res[args[i]] {
				continue
			}
			if strings.HasPrefix(args[i], "--") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				out[strings.TrimPrefix(args[i], "--")] = args[i+1]
				i++
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no data key/value pairs found (use --key value or --json '{...}')")
	}
	return out, nil
}
// ── helpers ─────────────────────────────────────────────────────────
func pickSpreadsheetID(args []string) string {
	if v := flagValue(args, "--spreadsheet", ""); v != "" {
		return v
	}
	if v := os.Getenv("OVAV_SHEETS_SPREADSHEET"); v != "" {
		return v
	}
	return defaultSpreadsheetID
}
func flagValue(args []string, name, def string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == name && i+1 < len(args) {
			return args[i+1]
		}
	}
	return def
}
func findRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(dir + "/.ovav"); err == nil {
			return dir, nil
		}
		parent := dir + "/.."
		if parent == dir {
			break
		}
		dir = parent
	}
	return cwd, nil
}
func exitOn(err error) {
	if err != nil {
		die(err)
	}
}
func die(err error) {
	fmt.Fprintln(os.Stderr, "❌", err)
	os.Exit(1)
}
func storePath(repoRoot string) string {
	return repoRoot + "/.ovav/vault/credentials/google_oauth.enc"
}
func parseMatrix(s string) ([][]string, error) {
	// Delegate to json unmarshal — safer than ad-hoc parsing.
	var m [][]string
	if err := jsonUnmarshalString(s, &m); err != nil {
		return nil, fmt.Errorf("values-json: %w", err)
	}
	if len(m) == 0 {
		return nil, fmt.Errorf("values-json: empty matrix")
	}
	return m, nil
}
// openBrowser tries to launch a browser on the host OS. Best effort.
func openBrowser(u string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", u)
	case "darwin":
		cmd = exec.Command("open", u)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	return cmd.Start()
}
// awaitCallback starts an HTTP server on the loopback address of
// redirectURI (e.g. 127.0.0.1:9876), waits up to timeout for Google to
// deliver ?code=…&state=…, validates state, and returns the code.
//
// On success it shows a friendly page in the user's browser and closes.
// On failure it returns a descriptive error.
func awaitCallback(redirectURI, expectedState string, timeout time.Duration) (string, error) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return "", fmt.Errorf("callback: bad redirect URI: %w", err)
	}
	host := u.Host
	if !strings.HasPrefix(host, "127.0.0.1") && !strings.HasPrefix(host, "localhost") {
		return "", fmt.Errorf("callback: refusing non-loopback redirect URI %q", redirectURI)
	}
	mux := http.NewServeMux()
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	mux.HandleFunc(u.Path, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if errParam := q.Get("error"); errParam != "" {
			errCh <- fmt.Errorf("google returned error=%s: %s", errParam, q.Get("error_description"))
			writeFriendlyPage(w, false, "Google denied consent: "+errParam)
			return
		}
		gotState := q.Get("state")
		if gotState != expectedState {
			errCh <- fmt.Errorf("state mismatch: got %q want %q", gotState, expectedState)
			writeFriendlyPage(w, false, "State mismatch — possible CSRF, refusing.")
			return
		}
		code := q.Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback: %s", r.URL.RawQuery)
			writeFriendlyPage(w, false, "No authorization code received.")
			return
		}
		writeFriendlyPage(w, true, "OVAV Sheets is connected. You can close this tab.")
		codeCh <- code
	})
	listener, err := net.Listen("tcp", host)
	if err != nil {
		return "", fmt.Errorf("callback: listen %s: %w", host, err)
	}
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(listener) }()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()
	fmt.Printf("⏳ waiting up to %s for Google to redirect back to %s%s …\n",
		timeout, host, u.Path)
	select {
	case code := <-codeCh:
		return code, nil
	case err := <-errCh:
		return "", err
	case <-time.After(timeout):
		return "", fmt.Errorf("callback: timeout after %s — user did not complete consent", timeout)
	}
}
func writeFriendlyPage(w http.ResponseWriter, ok bool, msg string) {
	bg := "#0f172a"
	card := "#1e293b"
	text := "#e2e8f0"
	accent := "#22c55e"
	if !ok {
		accent = "#ef4444"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!doctype html>
<html><head><meta charset="utf-8"><title>OVAV Sheets</title>
<style>
 body{margin:0;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;
     background:%s;color:%s;display:grid;place-items:center;height:100vh}
 .card{background:%s;padding:48px 64px;border-radius:16px;
       box-shadow:0 20px 60px rgba(0,0,0,.4);text-align:center;max-width:520px}
 h1{font-size:42px;margin:0 0 12px}
 p{font-size:16px;line-height:1.5;opacity:.85}
 .badge{display:inline-block;background:%s;color:#fff;padding:6px 14px;
        border-radius:999px;font-weight:600;margin-bottom:24px}
</style></head><body>
<div class="card">
 <div class="badge">%s</div>
 <h1>%s</h1>
 <p>%s</p>
</div></body></html>`,
		bg, text, card,
		accent,
		ternary(ok, "OVAV · CONNECTED", "OVAV · ERROR"),
		ternary(ok, "✓ Connected", "✗ Failed"),
		msg,
	)
}
func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
