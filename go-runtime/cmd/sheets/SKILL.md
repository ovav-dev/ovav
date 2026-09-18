# OVAV Sheets Bridge

Power-tool bridge between OVAV and Google Sheets / Excel — stdlib-only Go, AES-256-GCM credentials, MCP-ready, snapshot-safe.

## What it does

- **Reads** any tab in any approved spreadsheet, with auto header detection.
- **Writes** new tabs, values, rows by header-keyed maps, and conditional updates (`UPDATE WHERE col=val SET col=val`).
- **xlsx ↔ Sheets** bidirectional sync (pure Go reader + writer, no third-party deps).
- **Snapshots** every write; `rollback <id>` restores prior state.
- **Audit log** JSONL with hash chain at `.ovav/registry/audit/sheets/YYYY-MM-DD.jsonl`.
- **MCP server** (`go run ./cmd/sheets mcp`) speaks JSON-RPC 2.0 on stdio so OpenCode / Claude / Cursor can invoke every operation as a native tool.

## Setup (one-time)

```bash
cd go-runtime
go run ./cmd/sheets auth --client-id <ID> --client-secret <SECRET> --project-id <PID>
```

This opens your browser, captures the OAuth2 consent, encrypts the refresh token in `.ovav/vault/credentials/google_oauth.enc`, and registers an AES-256-GCM key at `.ovav/vault/vault.key` (0600).

## Allowlist

`.ovav/vault/sheets_allowlist.yaml` lists every spreadsheet the bridge is allowed to touch. The bridge REFUSES anything not listed:

```bash
go run ./cmd/sheets allowlist list
go run ./cmd/sheets allowlist add --id <SPREADSHEET_ID> --name "OVAV tasks" --tag work --default
go run ./cmd/sheets allowlist remove --id <SPREADSHEET_ID>
```

## CLI quick reference

| Command | Purpose |
|---|---|
| `auth` | OAuth2 web flow → encrypted refresh_token |
| `allowlist list/add/remove` | Manage approved spreadsheets |
| `list` | List tabs in the bound spreadsheet |
| `read --range A1:D10` | Read a range |
| `table --tab MyTab` | Read a tab as JSON rows keyed by header |
| `write --range A1:D1 --values-json '[[...]]'` | Write a range (USER_ENTERED) |
| `append --tab MyTab --sku A1 --qty 5` | Append a row by header → value |
| `update --tab MyTab --where-col sku --where-val A1 --set-col qty --set-val 99` | Update by condition |
| `create-tab --title NewTab` | Add a new tab |
| `snapshots [--tab MyTab]` | List captured snapshots |
| `rollback --id <snapshot-id>` | Restore a snapshot |
| `xlsx-in --tab MyTab --path ./file.xlsx` | Import Excel file |
| `xlsx-out --tab MyTab --path ./file.xlsx` | Export tab to Excel |
| `mcp` | Run MCP stdio server |

## MCP tools (auto-published via `tools/list`)

| Tool | Args |
|---|---|
| `sheets_list_spreadsheets` | (none) |
| `sheets_list_tabs` | `spreadsheet_id` |
| `sheets_read` | `spreadsheet_id`, `range` |
| `sheets_read_table` | `spreadsheet_id`, `tab` |
| `sheets_append_row` | `spreadsheet_id`, `tab`, `values` (object) |
| `sheets_update_where` | `spreadsheet_id`, `tab`, `where_col`, `where_val`, `set_col`, `set_val` |
| `sheets_create_tab` | `spreadsheet_id`, `title` |
| `sheets_snapshots` | `spreadsheet_id`, `tab?` |
| `sheets_rollback` | `snapshot_id` |
| `xlsx_to_sheets` | `spreadsheet_id`, `tab`, `path` |
| `sheets_to_xlsx` | `spreadsheet_id`, `tab`, `path` |

## Wire to OpenCode

Already wired in `.ovav/source/opencode/config.yaml`:

```yaml
mcp:
  ovav-sheets:
    type: local
    command: ["./bin/ovav-sheets-mcp"]
    enabled: true
```

Launch the Cockpit or any agent — `ovav-sheets` will appear as native tool family.

## Security model

| Layer | Where | Notes |
|---|---|---|
| Credentials | `.ovav/vault/credentials/google_oauth.enc` | AES-256-GCM, vault.key 0600 |
| Allowlist | `.ovav/vault/sheets_allowlist.yaml` | Default-deny on every operation |
| Snapshots | `.ovav/vault/snapshots/<sid>/<tab>/<ts>.json.gz` | Encrypted at rest with vault.key |
| Audit | `.ovav/registry/audit/sheets/YYYY-MM-DD.jsonl` | Append-only, hash-chained |
| Loopback only | OAuth2 callback at `127.0.0.1:9876` | Refuses non-loopback redirect URI |

## Tests

```bash
go test ./cmd/sheets -count=1 -v
```

Covers: column letter math, allowlist round-trip, xlsx round-trip, snapshot encrypt/decrypt, audit append, sanitization.

## Files

| File | Purpose |
|---|---|
| `main.go` | CLI entry (auth, list, read, write, append, update, mcp, …) |
| `sheets.go` | OAuth2 + token transport |
| `sheets_crypto.go` | Vault-aware credential storage |
| `client.go` | Sheets API v4 client |
| `tables.go` | Header-aware table ops (append, update where, find) |
| `update_cells.go` | GridRange-based batch update |
| `xlsx.go` | Pure-stdlib xlsx read + write |
| `snapshot.go` | Snapshot capture, list, rollback |
| `allowlist.go` | YAML-backed spreadsheet allowlist |
| `audit.go` | Append-only audit log with hash chain |
| `mcp.go` | JSON-RPC 2.0 MCP server |
| `util.go` | JSON / stdin helpers |
