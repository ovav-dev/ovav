// Package main — MCP server for the OVAV Sheets bridge.
//
// Wire format: JSON-RPC 2.0 over stdio. One tool per bridge
// capability. Schema is published via tools/list so OpenCode /
// Claude / Cursor can invoke them like any other MCP tool.
//
// Run with:  go run ./cmd/sheets mcp
//
// The server is intentionally idempotent and stateless; every
// request re-loads credentials from the vault, so a long-running
// session auto-detects token rotation.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// JSON-RPC 2.0 envelope.
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

const (
	rpcErrParse          = -32700
	rpcErrInvalidRequest = -32600
	rpcErrMethodNotFound = -32601
	rpcErrInvalidParams  = -32602
	rpcErrInternal       = -32603
)

// mcpTool is the published schema of one bridge operation.
type mcpTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func allTools() []mcpTool {
	return []mcpTool{
		{
			Name:        "sheets_list_spreadsheets",
			Description: "List every spreadsheet the bridge is allowed to operate on (from .ovav/vault/sheets_allowlist.yaml).",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			Name:        "sheets_list_tabs",
			Description: "List tabs in a spreadsheet.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"spreadsheet_id"},
				"properties": map[string]any{
					"spreadsheet_id": map[string]any{"type": "string", "description": "Allowed spreadsheet ID, e.g. 1MQ3wts..."},
				},
			},
		},
		{
			Name:        "sheets_read",
			Description: "Read a range (e.g. 'Sheet1!A1:D10').",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"spreadsheet_id", "range"},
				"properties": map[string]any{
					"spreadsheet_id": map[string]any{"type": "string"},
					"range":          map[string]any{"type": "string", "description": "A1 notation, may include tab name"},
				},
			},
		},
		{
			Name:        "sheets_read_table",
			Description: "Read a tab treating row 1 as headers; returns rows as JSON objects.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"spreadsheet_id", "tab"},
				"properties": map[string]any{
					"spreadsheet_id": map[string]any{"type": "string"},
					"tab":            map[string]any{"type": "string"},
				},
			},
		},
		{
			Name:        "sheets_append_row",
			Description: "Append one row at the bottom of a tab. Keys are column headers.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"spreadsheet_id", "tab", "values"},
				"properties": map[string]any{
					"spreadsheet_id": map[string]any{"type": "string"},
					"tab":            map[string]any{"type": "string"},
					"values":         map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}},
				},
			},
		},
		{
			Name:        "sheets_update_where",
			Description: "Find rows by column=value and update another column. Returns count.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"spreadsheet_id", "tab", "where_col", "where_val", "set_col", "set_val"},
				"properties": map[string]any{
					"spreadsheet_id": map[string]any{"type": "string"},
					"tab":            map[string]any{"type": "string"},
					"where_col":      map[string]any{"type": "string"},
					"where_val":      map[string]any{"type": "string"},
					"set_col":        map[string]any{"type": "string"},
					"set_val":        map[string]any{"type": "string"},
				},
			},
		},
		{
			Name:        "sheets_create_tab",
			Description: "Add a new tab to a spreadsheet.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"spreadsheet_id", "title"},
				"properties": map[string]any{
					"spreadsheet_id": map[string]any{"type": "string"},
					"title":          map[string]any{"type": "string"},
				},
			},
		},
		{
			Name:        "sheets_snapshots",
			Description: "List captured snapshots for a spreadsheet/tab.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"spreadsheet_id"},
				"properties": map[string]any{
					"spreadsheet_id": map[string]any{"type": "string"},
					"tab":            map[string]any{"type": "string"},
				},
			},
		},
		{
			Name:        "sheets_rollback",
			Description: "Restore a previously captured snapshot.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"snapshot_id"},
				"properties": map[string]any{
					"snapshot_id": map[string]any{"type": "string"},
				},
			},
		},
		{
			Name:        "xlsx_to_sheets",
			Description: "Read a local .xlsx file and append all rows to a tab (creates tab if absent).",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"spreadsheet_id", "tab", "path"},
				"properties": map[string]any{
					"spreadsheet_id": map[string]any{"type": "string"},
					"tab":            map[string]any{"type": "string"},
					"path":           map[string]any{"type": "string"},
				},
			},
		},
		{
			Name:        "sheets_to_xlsx",
			Description: "Download a tab as .xlsx to a local path.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"spreadsheet_id", "tab", "path"},
				"properties": map[string]any{
					"spreadsheet_id": map[string]any{"type": "string"},
					"tab":            map[string]any{"type": "string"},
					"path":           map[string]any{"type": "string"},
				},
			},
		},
	}
}

// runMCPServer is the stdio loop. It reads newline-delimited JSON
// requests and writes one response per line.
func runMCPServer(repoRoot string) error {
	store := NewCredStore(repoRoot)
	if _, err := store.Load(); err != nil {
		return err
	}
	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()
	fmt.Fprintln(os.Stderr, "ovav-sheets MCP server ready (stdio)")

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		line = bytesTrim(line)
		if len(line) == 0 {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			_ = writeRPC(writer, rpcResponse{
				JSONRPC: "2.0", Error: &rpcError{Code: rpcErrParse, Message: err.Error()},
			})
			continue
		}
		resp := handleRequest(repoRoot, &req)
		_ = writeRPC(writer, resp)
	}
}

func writeRPC(w *bufio.Writer, resp rpcResponse) error {
	b, _ := json.Marshal(resp)
	if _, err := w.Write(append(b, '\n')); err != nil {
		return err
	}
	return w.Flush()
}

func bytesTrim(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r' || b[len(b)-1] == ' ' || b[len(b)-1] == '\t') {
		b = b[:len(b)-1]
	}
	return b
}

func handleRequest(repoRoot string, req *rpcRequest) rpcResponse {
	switch req.Method {
	case "initialize":
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
			"protocolVersion": "2024-11-05",
			"serverInfo":      map[string]any{"name": "ovav-sheets", "version": "0.2.0"},
			"capabilities":    map[string]any{"tools": map[string]any{}},
		}}
	case "tools/list":
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
			"tools": allTools(),
		}}
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{
				Code: rpcErrInvalidParams, Message: "params: " + err.Error(),
			}}
		}
		res, err := dispatchTool(repoRoot, p.Name, p.Arguments)
		if err != nil {
			return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{
				Code: rpcErrInternal, Message: err.Error(),
			}}
		}
		// MCP expects {content: [{type: "text", text: "<json>"}]}
		text, _ := json.MarshalIndent(res, "", "  ")
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
			"content": []map[string]any{{"type": "text", "text": string(text)}},
		}}
	default:
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{
			Code: rpcErrMethodNotFound, Message: "method not found: " + req.Method,
		}}
	}
}

func dispatchTool(repoRoot, name string, args json.RawMessage) (any, error) {
	switch name {
	case "sheets_list_spreadsheets":
		return globalAllowlist.entries, nil
	case "sheets_list_tabs":
		var p struct{ SpreadsheetID string `json:"spreadsheet_id"` }
		_ = json.Unmarshal(args, &p)
		return mcpListTabs(repoRoot, p.SpreadsheetID)
	case "sheets_read":
		var p struct {
			SpreadsheetID string `json:"spreadsheet_id"`
			Range         string `json:"range"`
		}
		_ = json.Unmarshal(args, &p)
		return mcpRead(repoRoot, p.SpreadsheetID, p.Range)
	case "sheets_read_table":
		var p struct {
			SpreadsheetID string `json:"spreadsheet_id"`
			Tab           string `json:"tab"`
		}
		_ = json.Unmarshal(args, &p)
		return mcpReadTable(repoRoot, p.SpreadsheetID, p.Tab)
	case "sheets_append_row":
		var p struct {
			SpreadsheetID string            `json:"spreadsheet_id"`
			Tab           string            `json:"tab"`
			Values        map[string]string `json:"values"`
		}
		_ = json.Unmarshal(args, &p)
		return mcpAppendRow(repoRoot, p.SpreadsheetID, p.Tab, p.Values)
	case "sheets_update_where":
		var p struct {
			SpreadsheetID string `json:"spreadsheet_id"`
			Tab           string `json:"tab"`
			WhereCol      string `json:"where_col"`
			WhereVal      string `json:"where_val"`
			SetCol        string `json:"set_col"`
			SetVal        string `json:"set_val"`
		}
		_ = json.Unmarshal(args, &p)
		return mcpUpdateWhere(repoRoot, p.SpreadsheetID, p.Tab, p.WhereCol, p.WhereVal, p.SetCol, p.SetVal)
	case "sheets_create_tab":
		var p struct {
			SpreadsheetID string `json:"spreadsheet_id"`
			Title         string `json:"title"`
		}
		_ = json.Unmarshal(args, &p)
		return mcpCreateTab(repoRoot, p.SpreadsheetID, p.Title)
	case "sheets_snapshots":
		var p struct {
			SpreadsheetID string `json:"spreadsheet_id"`
			Tab           string `json:"tab"`
		}
		_ = json.Unmarshal(args, &p)
		return ListSnapshots(repoRoot, p.SpreadsheetID, p.Tab)
	case "sheets_rollback":
		var p struct {
			SnapshotID string `json:"snapshot_id"`
		}
		_ = json.Unmarshal(args, &p)
		return nil, Rollback(repoRoot, p.SnapshotID)
	case "xlsx_to_sheets":
		var p struct {
			SpreadsheetID string `json:"spreadsheet_id"`
			Tab           string `json:"tab"`
			Path          string `json:"path"`
		}
		_ = json.Unmarshal(args, &p)
		return mcpXlsxToSheets(repoRoot, p.SpreadsheetID, p.Tab, p.Path)
	case "sheets_to_xlsx":
		var p struct {
			SpreadsheetID string `json:"spreadsheet_id"`
			Tab           string `json:"tab"`
			Path          string `json:"path"`
		}
		_ = json.Unmarshal(args, &p)
		return nil, mcpSheetsToXlsx(repoRoot, p.SpreadsheetID, p.Tab, p.Path)
	}
	return nil, fmt.Errorf("unknown tool: %s", name)
}

// ── MCP tool implementations ────────────────────────────────────────

func mcpListTabs(repoRoot, sid string) (any, error) {
	if err := assertAllowed(sid); err != nil {
		return nil, err
	}
	cl, err := newClientFor(repoRoot, sid)
	if err != nil {
		return nil, err
	}
	return cl.ListSheets()
}

func mcpRead(repoRoot, sid, rng string) (any, error) {
	if err := assertAllowed(sid); err != nil {
		return nil, err
	}
	cl, err := newClientFor(repoRoot, sid)
	if err != nil {
		return nil, err
	}
	return cl.ReadRange(rng)
}

func mcpReadTable(repoRoot, sid, tab string) (any, error) {
	if err := assertAllowed(sid); err != nil {
		return nil, err
	}
	cl, err := newClientFor(repoRoot, sid)
	if err != nil {
		return nil, err
	}
	t, err := cl.LoadTable(tab)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"headers": t.Header(),
		"rows":    t.AsMaps(),
	}, nil
}

func mcpAppendRow(repoRoot, sid, tab string, values map[string]string) (any, error) {
	if err := assertAllowed(sid); err != nil {
		return nil, err
	}
	cl, err := newClientFor(repoRoot, sid)
	if err != nil {
		return nil, err
	}
	// snapshot pre-state
	if pre, err := cl.ReadRange(tab + "!A1:Z10000"); err == nil {
		if b, err := json.Marshal(pre); err == nil {
			_, _ = cl.Capture(repoRoot, "append-pre", "before append on "+tab, tab, b)
		}
	}
	t, err := cl.LoadTable(tab)
	if err != nil {
		return nil, err
	}
	rowIdx, a1, err := t.AppendRow(values)
	if err != nil {
		return nil, err
	}
	audit(repoRoot, sid, tab, "append_row", map[string]any{"values": values}, rowIdx)
	return map[string]any{"row": rowIdx, "range": a1}, nil
}

func mcpUpdateWhere(repoRoot, sid, tab, wCol, wVal, sCol, sVal string) (any, error) {
	if err := assertAllowed(sid); err != nil {
		return nil, err
	}
	cl, err := newClientFor(repoRoot, sid)
	if err != nil {
		return nil, err
	}
	if pre, err := cl.ReadRange(tab + "!A1:Z10000"); err == nil {
		if b, err := json.Marshal(pre); err == nil {
			_, _ = cl.Capture(repoRoot, "update-pre", fmt.Sprintf("before update where %s=%s on %s", wCol, wVal, tab), tab, b)
		}
	}
	t, err := cl.LoadTable(tab)
	if err != nil {
		return nil, err
	}
	n, err := t.UpdateWhere(wCol, wVal, sCol, sVal)
	if err != nil {
		return nil, err
	}
	audit(repoRoot, sid, tab, "update_where",
		map[string]any{"where_col": wCol, "where_val": wVal, "set_col": sCol, "set_val": sVal},
		n)
	return map[string]any{"rows_updated": n}, nil
}

func mcpCreateTab(repoRoot, sid, title string) (any, error) {
	if err := assertAllowed(sid); err != nil {
		return nil, err
	}
	cl, err := newClientFor(repoRoot, sid)
	if err != nil {
		return nil, err
	}
	id, err := cl.AddSheet(title, map[string]float64{"red": 0.145, "green": 0.388, "blue": 0.922})
	if err != nil {
		return nil, err
	}
	audit(repoRoot, sid, title, "create_tab", map[string]any{"title": title}, int(id))
	return map[string]any{"title": title, "sheetId": id}, nil
}

func mcpXlsxToSheets(repoRoot, sid, tab, path string) (any, error) {
	if err := assertAllowed(sid); err != nil {
		return nil, err
	}
	cl, err := newClientFor(repoRoot, sid)
	if err != nil {
		return nil, err
	}
	sheetsList, err := ReadXlsx(path)
	if err != nil {
		return nil, err
	}
	if len(sheetsList) == 0 {
		return nil, fmt.Errorf("xlsx: no sheets")
	}
	sh := sheetsList[0]
	// Make sure tab exists.
	if _, err := cl.AddSheet(tab, nil); err != nil {
		// ignore "already exists" — Sheets returns that anyway
		_ = err
	}
	rows := XlsxToCells(sh)
	if len(rows) == 0 {
		return nil, fmt.Errorf("xlsx: empty sheet")
	}
	// Use first row as headers.
	a1 := fmt.Sprintf("%s!A1", quoted(tab))
	ur, err := cl.WriteValues(a1, rows, "USER_ENTERED")
	if err != nil {
		return nil, err
	}
	audit(repoRoot, sid, tab, "xlsx_to_sheets",
		map[string]any{"path": path, "rows": len(rows)},
		ur.UpdatedCells)
	return map[string]any{
		"rows_written": len(rows),
		"range":        ur.UpdatedRange,
		"cells":        ur.UpdatedCells,
	}, nil
}

func mcpSheetsToXlsx(repoRoot, sid, tab, path string) error {
	if err := assertAllowed(sid); err != nil {
		return err
	}
	cl, err := newClientFor(repoRoot, sid)
	if err != nil {
		return err
	}
	t, err := cl.LoadTable(tab)
	if err != nil {
		return err
	}
	// Build XlsxCells including header
	cells := []XlsxCell{}
	maxR := 0
	maxC := len(t.headers)
	for i, h := range t.headers {
		cells = append(cells, XlsxCell{Ref: fmt.Sprintf("%s%d", colLetter(i+1), 1), Val: h})
	}
	maxR = 1
	for ri, r := range t.rows {
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	audit(repoRoot, sid, tab, "sheets_to_xlsx", map[string]any{"path": path}, len(cells))
	fmt.Printf("✅ wrote %d cells to %s\n", len(cells), path)
	return nil
}

func newClientFor(repoRoot, sid string) (*Client, error) {
	store := NewCredStore(repoRoot)
	creds, err := store.Load()
	if err != nil {
		return nil, err
	}
	return NewClient(creds, sid), nil
}

var _ = strings.ToLower
