package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// updateCells sends a batchUpdate with multiple updateCells requests.
// Each range gets the matching row from values, preserving row order.
// Note: the Sheets API requires GridRange with numeric indices + sheetId,
// not a1Range. We compute those from the tab name by looking up the
// sheetId once via ListSheets and caching it for the client's lifetime.
func (cl *Client) updateCells(ranges []string, values [][]any) error {
	if err := EnsureFresh(cl.c); err != nil {
		return err
	}
	if len(ranges) != len(values) {
		return fmt.Errorf("sheets: updateCells range/value count mismatch")
	}
	// Resolve each range into (sheetId, row, col).
	type loc struct {
		sheetID  int64
		row, col int
	}
	locs := make([]loc, 0, len(ranges))
	for _, r := range ranges {
		sheetID, row, col, err := cl.resolveA1(r)
		if err != nil {
			return err
		}
		locs = append(locs, loc{sheetID, row, col})
	}
	reqs := make([]map[string]any, 0, len(ranges))
	for i, l := range locs {
		rowValues := values[i]
		extVals := make([]map[string]any, 0, len(rowValues))
		for _, v := range rowValues {
			switch x := v.(type) {
			case string:
				extVals = append(extVals, map[string]any{
					"userEnteredValue": map[string]any{"stringValue": x},
				})
			default:
				b, _ := json.Marshal(x)
				extVals = append(extVals, map[string]any{
					"userEnteredValue": map[string]any{"stringValue": string(b)},
				})
			}
		}
		reqs = append(reqs, map[string]any{
			"updateCells": map[string]any{
				"range": map[string]any{
					"sheetId":          l.sheetID,
					"startRowIndex":    l.row,
					"endRowIndex":      l.row + 1,
					"startColumnIndex": l.col,
					"endColumnIndex":   l.col + 1,
				},
				"rows":   []map[string]any{{"values": extVals}},
				"fields": "userEnteredValue",
			},
		})
	}
	body, err := json.Marshal(map[string]any{"requests": reqs})
	if err != nil {
		return err
	}
	u := fmt.Sprintf("%s/%s:batchUpdate", sheetsAPIBase, cl.spreadsheetID)
	req, _ := http.NewRequest(http.MethodPost, u, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := cl.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return readHTTPError(resp)
	}
	return nil
}

// sheetIDCache memoizes the sheetId lookup per spreadsheet + tab name.
var sheetIDCache sync.Map // map[spreadsheetID+"::"+tab]SheetMeta

func (cl *Client) resolveA1(a1 string) (sheetID int64, row, col int, err error) {
	// Parse "<tab>!<colLetter><row>" or just "<colLetter><row>".
	tab := ""
	if idx := strings.Index(a1, "!"); idx >= 0 {
		tab = strings.Trim(a1[:idx], "'")
		a1 = a1[idx+1:]
		if c := strings.Index(a1, ":"); c >= 0 {
			a1 = a1[:c] // only need first cell for update
		}
	}
	if tab == "" {
		return 0, 0, 0, fmt.Errorf("sheets: updateCells needs tab-qualified range, got %q", a1)
	}
	// Split letters / digits.
	i := 0
	for i < len(a1) && a1[i] >= 'A' && a1[i] <= 'Z' {
		i++
	}
	if i == 0 {
		return 0, 0, 0, fmt.Errorf("sheets: bad A1 %q", a1)
	}
	col = colLetterInverse(a1[:i]) - 1 // colLetterInverse returns 1-based; sheets API is 0-based
	rowStr := a1[i:]
	r, err := strconv.Atoi(rowStr)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("sheets: bad A1 row %q", a1)
	}
	row = r - 1 // sheets uses 0-based indices

	cacheKey := cl.spreadsheetID + "::" + tab
	if cached, ok := sheetIDCache.Load(cacheKey); ok {
		meta := cached.(SheetMeta)
		return meta.SheetID, row, col, nil
	}
	sheets, err := cl.ListSheets()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("sheets: listSheets for resolve: %w", err)
	}
	for _, s := range sheets {
		if s.Title == tab {
			sheetIDCache.Store(cacheKey, s)
			return s.SheetID, row, col, nil
		}
	}
	return 0, 0, 0, fmt.Errorf("sheets: tab %q not found for resolveA1", tab)
}

// readHTTPError extracts a readable error from a non-2xx response.
func readHTTPError(resp *http.Response) error {
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	return fmt.Errorf("sheets: HTTP %d: %s", resp.StatusCode, truncate(buf.String(), 320))
}
