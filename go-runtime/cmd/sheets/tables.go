package main

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Table wraps a sheet treating the first row as headers, returning
// rows as maps[string]string keyed by header. This is what every
// real workflow actually wants: column-aware access without writing
// A1 addresses.
//
// All operations are stateless on the server side — we always
// Read first, then issue a targeted Update. Safe for concurrent
// agents when callers respect the contract.
type Table struct {
	cl      *Client
	tabName string
	headers []string
	rows    [][]string
}

// LoadTable reads a tab (or any range that starts with the header
// row), normalizes blank cells to "", and returns a Table handle.
func (cl *Client) LoadTable(tabName string) (*Table, error) {
	// First peek at A1:Z1 to know how wide the table is.
	header, err := cl.ReadRange(quoted(tabName) + "!A1:Z1")
	if err != nil {
		return nil, err
	}
	headers := make([]string, 0)
	if len(header.Values) > 0 {
		for _, c := range header.Values[0] {
			headers = append(headers, strings.TrimSpace(c))
		}
	}
	if len(headers) == 0 {
		return nil, fmt.Errorf("sheets: tab %q has no header row at A1:Z1", tabName)
	}
	// Now read the data block. Cap at 10k rows by default.
	data, err := cl.ReadRange(quoted(tabName) + fmt.Sprintf("!A2:Z%d", 10000))
	if err != nil {
		return nil, err
	}
	return &Table{
		cl: cl, tabName: tabName,
		headers: headers, rows: data.Values,
	}, nil
}

// Header returns the column names.
func (t *Table) Header() []string { return t.headers }

// Rows returns every row as a slice of slices.
func (t *Table) Rows() [][]string { return t.rows }

// AsMaps returns rows as map[header]value. Missing cells are "".
func (t *Table) AsMaps() []map[string]string {
	out := make([]map[string]string, 0, len(t.rows))
	for _, r := range t.rows {
		m := map[string]string{}
		for i, h := range t.headers {
			if i < len(r) {
				m[h] = r[i]
			} else {
				m[h] = ""
			}
		}
		out = append(out, m)
	}
	return out
}

// Find returns the indices of rows where column == value. Empty if none.
func (t *Table) Find(column, value string) []int {
	col := indexOf(t.headers, column)
	if col < 0 {
		return nil
	}
	var out []int
	for i, r := range t.rows {
		var cell string
		if col < len(r) {
			cell = r[col]
		}
		if cell == value {
			out = append(out, i)
		}
	}
	return out
}

// AppendRow adds one row at the bottom of the table. cells is a map
// keyed by header — only provided columns are written; missing ones
// are left blank. Returns the new row index (1-based, including the
// header) and the A1 range that was written.
func (t *Table) AppendRow(cells map[string]string) (rowIdx int, a1 string, err error) {
	rowIdx = len(t.rows) + 2 // +1 for header, +1 for 1-based
	a1 = fmt.Sprintf("%s!A%d:%s%d", quoted(t.tabName), rowIdx,
		colLetter(len(t.headers)), rowIdx)
	values := make([][]string, 1)
	values[0] = make([]string, len(t.headers))
	for i, h := range t.headers {
		values[0][i] = cells[h]
	}
	ur, err := t.cl.WriteValues(a1, values, "USER_ENTERED")
	if err != nil {
		return 0, "", err
	}
	return rowIdx, ur.UpdatedRange, nil
}

// UpdateWhere updates every row where column==oldValue, replacing
// the value of updateColumn with newValue. Returns the number of
// rows affected. Empty updateColumn means "replace the matching
// column itself" (alias-rename semantics).
func (t *Table) UpdateWhere(column, oldValue, updateColumn, newValue string) (int, error) {
	colIdx := indexOf(t.headers, column)
	if colIdx < 0 {
		return 0, fmt.Errorf("sheets: column %q not in headers", column)
	}
	if updateColumn == "" {
		updateColumn = column
	}
	updIdx := indexOf(t.headers, updateColumn)
	if updIdx < 0 {
		return 0, fmt.Errorf("sheets: column %q not in headers", updateColumn)
	}
	matches := t.Find(column, oldValue)
	if len(matches) == 0 {
		return 0, nil
	}
	// Collect ranges so we can issue a single batchUpdate.
	// m is 0-based index into t.rows; the header occupies sheet
	// row 1, so data row m sits at sheet row m+2 (1-based).
	// We emit A1 using the 1-based number; update_cells converts
	// to 0-based for the Sheets API.
	type patch struct{ r, c int }
	var patches []patch
	for _, m := range matches {
		patches = append(patches, patch{r: m + 2, c: updIdx})
	}
	// Build a single values body.
	values := make([][]any, len(patches))
	for i := range patches {
		row := make([]any, 1)
		row[0] = newValue
		values[i] = row
	}
	// Use UpdateCells (batchUpdate with updateCells request).
	ranges := make([]string, 0, len(patches))
	for _, p := range patches {
		ranges = append(ranges, fmt.Sprintf("%s!%s%d",
			quoted(t.tabName), colLetter(p.c+1), p.r))
	}
	if err := t.cl.updateCells(ranges, values); err != nil {
		return 0, err
	}
	return len(patches), nil
}

// context import retained for future timeout support.
var _ = context.Background

func quoted(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

func indexOf(haystack []string, needle string) int {
	for i, h := range haystack {
		if h == needle {
			return i
		}
	}
	return -1
}

// colLetter converts 1-based column index to spreadsheet letter(s).
// 1 -> A, 26 -> Z, 27 -> AA, 28 -> AB, ...
func colLetter(n int) string {
	if n <= 0 {
		return "A"
	}
	var out []byte
	for n > 0 {
		n--
		out = append([]byte{byte('A' + n%26)}, out...)
		n /= 26
	}
	return string(out)
}

// colLetterInverse is used in tests / future round-trips.
func colLetterInverse(letters string) int {
	n := 0
	for _, c := range letters {
		n = n*26 + int(c-'A'+1)
	}
	return n
}

// url.Values import marker — keep import surface minimal but available.
var _ = url.Values{}
