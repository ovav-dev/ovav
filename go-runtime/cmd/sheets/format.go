// Package main — formatting, validation, charts, named ranges,
// protected ranges and image insertion via Sheets API v4.
//
// All implemented via batchUpdate requests — no manual editor work.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// ── Conditional formatting ────────────────────────────────────────────

// ConditionalFormatRule is a thin wrapper around the rich
// sheets API request shape. Two convenience constructors cover 95%
// of real-world rules: CONDITION_FORMATTING_HIGHLIGHT and
// BOOLEAN_RULE.
type ConditionalFormatRule struct {
	Range    string // A1 notation, e.g. "CIMA!A5:B5"
	When     string // "TEXT_EQ" | "TEXT_CONTAINS" | "CUSTOM_FORMULA" | "NUMBER_LESS" | ...
	Value    string // comparison value (or formula if When == CUSTOM_FORMULA)
	BGColor  string // hex without #, e.g. "FFEB9C" (yellow)
	FGColor  string // hex without #
	Bold     bool
}

func (cl *Client) AddConditionalRule(r ConditionalFormatRule) error {
	if err := EnsureFresh(cl.c); err != nil {
		return err
	}
	gridRange, err := cl.a1ToGrid(r.Range)
	if err != nil {
		return err
	}
	rule := buildConditionalRule(r, gridRange)
	body, _ := json.Marshal(map[string]any{
		"requests": []map[string]any{
			{"addConditionalFormatRule": rule},
		},
	})
	return cl.batchUpdate(body)
}

// buildConditionalRule constructs the Sheets API v4
// addConditionalFormatRule request. Sheets v4 wraps `type` and
// `values` inside a `condition` object under `booleanRule`.
func buildConditionalRule(r ConditionalFormatRule, gridRange map[string]any) map[string]any {
	format := map[string]any{}
	if r.BGColor != "" {
		format["backgroundColor"] = map[string]any{
			"red":   hexToFloat(r.BGColor, 0),
			"green": hexToFloat(r.BGColor, 1),
			"blue":  hexToFloat(r.BGColor, 2),
		}
	}
	if r.FGColor != "" {
		format["textFormat"] = map[string]any{
			"foregroundColor": map[string]any{
				"red":   hexToFloat(r.FGColor, 0),
				"green": hexToFloat(r.FGColor, 1),
				"blue":  hexToFloat(r.FGColor, 2),
			},
		}
		if r.Bold {
			format["textFormat"].(map[string]any)["bold"] = true
		}
	} else if r.Bold {
		format["textFormat"] = map[string]any{"bold": true}
	}

	var booleanRule map[string]any
	switch r.When {
	case "TEXT_EQ":
		booleanRule = map[string]any{
			"type": "TEXT_EQ",
			"values": []map[string]any{{"userEnteredValue": r.Value}},
		}
	case "TEXT_CONTAINS":
		booleanRule = map[string]any{
			"type": "TEXT_CONTAINS",
			"values": []map[string]any{{"userEnteredValue": r.Value}},
		}
	case "NUMBER_LESS":
		booleanRule = map[string]any{
			"type": "NUMBER_LESS",
			"values": []map[string]any{{"userEnteredValue": r.Value}},
		}
	case "NUMBER_GREATER":
		booleanRule = map[string]any{
			"type": "NUMBER_GREATER",
			"values": []map[string]any{{"userEnteredValue": r.Value}},
		}
	case "CUSTOM_FORMULA":
		booleanRule = map[string]any{
			"type": "CUSTOM_FORMULA",
			"values": []map[string]any{{"userEnteredValue": r.Value}},
		}
	default:
		booleanRule = map[string]any{
			"type": "TEXT_CONTAINS",
			"values": []map[string]any{{"userEnteredValue": r.Value}},
		}
	}

	return map[string]any{
		"rule": map[string]any{
			"ranges": []map[string]any{gridRange},
			"booleanRule": map[string]any{
				"condition": booleanRule,
				"format":    format,
			},
		},
		"index": 0,
	}
}

// ── Data validation ──────────────────────────────────────────────────

// SetDataValidation adds a dropdown or checkbox rule.
// type: "ONE_OF_LIST" | "ONE_OF_RANGE" | "CHECKBOX" | "NUMBER_BETWEEN"
// values: list items (string for ONE_OF_LIST) or {"min","max"} (for BETWEEN)
func (cl *Client) SetDataValidation(a1Range, ruleType string, values any, strict bool) error {
	if err := EnsureFresh(cl.c); err != nil {
		return err
	}
	grid, err := cl.a1ToGrid(a1Range)
	if err != nil {
		return err
	}
	rule := map[string]any{}
	switch ruleType {
	case "ONE_OF_LIST":
		list, _ := values.([]string)
		asAny := make([]any, 0, len(list))
		for _, s := range list {
			asAny = append(asAny, s)
		}
		rule["condition"] = map[string]any{
			"type":   "ONE_OF_LIST",
			"values": toUserEnteredValues(asAny),
		}
		if len(list) > 0 {
			rule["showCustomUi"] = false
		}
	case "ONE_OF_RANGE":
		rule["condition"] = map[string]any{
			"type":   "ONE_OF_RANGE",
			"values": []map[string]any{{"userEnteredValue": fmt.Sprint(values)}},
		}
	case "CHECKBOX":
		rule["condition"] = map[string]any{"type": "BOOLEAN"}
	case "NUMBER_BETWEEN":
		mm, ok := values.(map[string]any)
		if !ok {
			return fmt.Errorf("sheets: NUMBER_BETWEEN requires values {min,max}")
		}
		rule["condition"] = map[string]any{
			"type":   "NUMBER_BETWEEN",
			"values": toUserEnteredValues([]any{mm["min"], mm["max"]}),
		}
	default:
		return fmt.Errorf("sheets: unknown validation type %q", ruleType)
	}
	if strict {
		rule["strict"] = true
	}
	req := map[string]any{
		"range": grid,
		"rule":  rule,
	}
	body, _ := json.Marshal(map[string]any{
		"requests": []map[string]any{
			{"setDataValidation": req},
		},
	})
	return cl.batchUpdate(body)
}

func toUserEnteredValues(items []any) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, x := range items {
		out = append(out, map[string]any{"userEnteredValue": fmt.Sprint(x)})
	}
	return out
}

// ── Protected ranges ─────────────────────────────────────────────────

// AddProtectedRange locks a range behind an editors-only warning.
// Editors (the email list) can edit without a prompt; everyone else
// gets a warning before editing.
func (cl *Client) AddProtectedRange(a1Range, description string, editors []string, warningOnly bool) error {
	if err := EnsureFresh(cl.c); err != nil {
		return err
	}
	grid, err := cl.a1ToGrid(a1Range)
	if err != nil {
		return err
	}
	protected := map[string]any{
		"description": description,
		"range":       grid,
		"warningOnly": warningOnly,
	}
	if len(editors) > 0 {
		asAny := make([]any, 0, len(editors))
		for _, e := range editors {
			asAny = append(asAny, e)
		}
		protected["editors"] = map[string]any{"users": toUserEnteredValues(asAny)}
	}
	body, _ := json.Marshal(map[string]any{
		"requests": []map[string]any{
			{"addProtectedRange": map[string]any{"protectedRange": protected}},
		},
	})
	return cl.batchUpdate(body)
}

// ── Named ranges ─────────────────────────────────────────────────────

func (cl *Client) AddNamedRange(name, a1Range string) error {
	if err := EnsureFresh(cl.c); err != nil {
		return err
	}
	grid, err := cl.a1ToGrid(a1Range)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]any{
		"requests": []map[string]any{
			{"addNamedRange": map[string]any{
				"namedRange": map[string]any{
					"name":  name,
					"range": grid,
				},
			}},
		},
	})
	return cl.batchUpdate(body)
}

// ── Charts ───────────────────────────────────────────────────────────

// AddChart inserts a basic chart of the given type over a1Range.
// Supported types: BAR, LINE, PIE, AREA, COLUMN, SCATTER.
func (cl *Client) AddChart(tabName, a1Range, chartType, title string) error {
	if err := EnsureFresh(cl.c); err != nil {
		return err
	}
	// Resolve sheetId from tab name.
	tab, err := cl.resolveSheetID(tabName)
	if err != nil {
		return err
	}
	spec := buildChartSpec(chartType, title, a1Range)
	body, _ := json.Marshal(map[string]any{
		"requests": []map[string]any{
			{"addChart": map[string]any{
				"chart": map[string]any{
					"spec":   spec,
					"position": map[string]any{
						"overlayPosition": map[string]any{
							"anchorCell": map[string]any{"sheetId": tab, "rowIndex": 1, "columnIndex": 8},
							"widthPixels":  600,
							"heightPixels": 360,
						},
					},
				},
			}},
		},
	})
	return cl.batchUpdate(body)
}

func buildChartSpec(chartType, title, a1Range string) map[string]any {
	var basicChartType string
	switch chartType {
	case "PIE":
		return map[string]any{
			"title": title,
			"pieChart": map[string]any{
				"legendPosition": "RIGHT_LEGEND",
				"domain":        chartSource(a1Range, 0)["domain"],
				"series":        chartSource(a1Range, 1)["domain"],
			},
		}
	case "LINE":
		basicChartType = "LINE"
	case "AREA":
		basicChartType = "AREA"
	case "SCATTER":
		basicChartType = "SCATTER"
	default:
		basicChartType = chartType // BAR or COLUMN
	}
	return map[string]any{
		"title": title,
		"basicChart": map[string]any{
			"chartType":      basicChartType,
			"domains":        []map[string]any{chartSource(a1Range, 0)},
			"series":         []map[string]any{chartSource(a1Range, 1)},
			"legendPosition": "RIGHT_LEGEND",
			"axis": map[string]any{
				"position": "BOTTOM_AXIS",
				"title":    "",
			},
		},
	}
}

func chartSource(a1Range string, columnIndex int) map[string]any {
	return map[string]any{
		"domain": map[string]any{
			"sourceRange": map[string]any{
				"sources": []map[string]any{
					{"a1Range": a1Range},
				},
			},
		},
	}
}

// ── Images ───────────────────────────────────────────────────────────

// InsertImage adds an image from a URL.
func (cl *Client) InsertImage(tabName, anchorCell, imageURL string) error {
	if err := EnsureFresh(cl.c); err != nil {
		return err
	}
	tab, err := cl.resolveSheetID(tabName)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]any{
		"requests": []map[string]any{
			{"insertImage": map[string]any{
				"uri":    imageURL,
				"overlayPosition": map[string]any{
					"anchorCell": map[string]any{"sheetId": tab, "rowIndex": 0, "columnIndex": 0},
					"offsetXPixels": 0,
					"offsetYPixels": 0,
					"widthPixels":    300,
					"heightPixels":   200,
				},
			}},
		},
	})
	return cl.batchUpdate(body)
}

// ── helpers ──────────────────────────────────────────────────────────

func (cl *Client) batchUpdate(body []byte) error {
	u := fmt.Sprintf("%s/%s:batchUpdate", sheetsAPIBase, cl.spreadsheetID)
	req, _ := http.NewRequest(http.MethodPost, u, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := cl.http.Do(req)
	if err != nil {
		return fmt.Errorf("sheets: batchUpdate: %w", err)
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sheets: batchUpdate HTTP %d: %s", resp.StatusCode, truncate(string(rb), 320))
	}
	return nil
}

func (cl *Client) resolveSheetID(tabName string) (int64, error) {
	sheets, err := cl.ListSheets()
	if err != nil {
		return 0, err
	}
	for _, s := range sheets {
		if s.Title == tabName {
			return s.SheetID, nil
		}
	}
	return 0, fmt.Errorf("sheets: tab %q not found", tabName)
}

func hexToFloat(hex string, idx int) float64 {
	if len(hex) < 6 {
		return 0
	}
	hex = strings.ToUpper(hex)
	if hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) >= 2 {
		var v uint8
		_, err := fmt.Sscanf(hex[2*idx:2*idx+2], "%x", &v)
		if err == nil {
			return float64(v) / 255.0
		}
	}
	return 0
}

// a1ToGrid parses a tab-qualified A1 like 'PERSONAL!A4:P4' into a
// GridRange (sheetId + numeric indices). Used by named ranges.
func (cl *Client) a1ToGrid(a1Range string) (map[string]any, error) {
	parts := strings.SplitN(a1Range, "!", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("a1ToGrid: range %q must be tab-qualified", a1Range)
	}
	tab := strings.Trim(parts[0], "'")
	rng := parts[1]
	span := strings.SplitN(rng, ":", 2)
	startA1, endA1 := span[0], span[0]
	if len(span) == 2 {
		endA1 = span[1]
	}
	startCol, startRow, err := parseA1Cell(startA1)
	if err != nil {
		return nil, err
	}
	endCol, endRow, err := parseA1Cell(endA1)
	if err != nil {
		return nil, err
	}
	sheets, err := cl.ListSheets()
	if err != nil {
		return nil, err
	}
	var sheetID int64
	for _, s := range sheets {
		if s.Title == tab {
			sheetID = s.SheetID
			break
		}
	}
	if sheetID == 0 {
		return nil, fmt.Errorf("a1ToGrid: tab %q not found", tab)
	}
	return map[string]any{
		"sheetId":          sheetID,
		"startRowIndex":    startRow - 1,
		"endRowIndex":      endRow,
		"startColumnIndex": startCol - 1,
		"endColumnIndex":   endCol,
	}, nil
}

func parseA1Cell(cell string) (col, row int, err error) {
	i := 0
	for i < len(cell) && cell[i] >= 'A' && cell[i] <= 'Z' {
		i++
	}
	if i == 0 || i == len(cell) {
		return 0, 0, fmt.Errorf("bad cell %q", cell)
	}
	col = colLetterInverse(cell[:i])
	row, err = strconv.Atoi(cell[i:])
	if err != nil {
		return 0, 0, fmt.Errorf("bad row in %q: %w", cell, err)
	}
	return col, row, nil
}
