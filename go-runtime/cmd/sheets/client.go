// Package sheets — Sheets API v4 client (stdlib only).
//
// Wraps only the operations OVAV needs:
//   - read range
//   - write values
//   - add new sheet (tab) inside an existing spreadsheet
//
// Keeps the surface small on purpose — every method is auditable.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const sheetsAPIBase = "https://sheets.googleapis.com/v4/spreadsheets"

// Client is a thin wrapper around Sheets API v4 with one spreadsheet bound.
type Client struct {
	c            *Creds
	spreadsheetID string
	http         *http.Client
}

// NewClient binds a spreadsheet ID. spreadsheetID must be in the
// allowlist — production callers should verify before constructing.
func NewClient(c *Creds, spreadsheetID string) *Client {
	return &Client{
		c:             c,
		spreadsheetID: spreadsheetID,
		http:          AuthorizedClient(c),
	}
}

// legacyAllowlist is preserved only for backwards compatibility with
// the v0.1 demo binary. The dynamic loader (allowlist.go) is the
// authoritative source as of v0.2 — see loadAllowlist / assertAllowed.
var legacyAllowlist = map[string]string{
	"1MQ3wts_cEG_4Dp5U6X4gehEn6u7F61tIl7KHzSzKw0Y": "OVAV demo workbook",
}

// legacyAssertAllowed is the v0.1 fallback gate. New code calls
// assertAllowed (lowercase) from allowlist.go which reads the YAML.
func legacyAssertAllowed(spreadsheetID string) error {
	if _, ok := legacyAllowlist[spreadsheetID]; ok {
		return nil
	}
	return fmt.Errorf("sheets: spreadsheet %q not in allowlist (v0.1 legacy)", spreadsheetID)
}

// SheetMeta describes one tab in the spreadsheet.
type SheetMeta struct {
	SheetID int64  `json:"sheetId"`
	Title   string `json:"title"`
	Index   int    `json:"index"`
	Rows    int    `json:"gridProperties.rowCount"`
	Cols    int    `json:"gridProperties.columnCount"`
}

type spreadsheetResp struct {
	Sheets []struct {
		Properties SheetMeta `json:"properties"`
	} `json:"sheets"`
}

// ListSheets returns metadata for every tab in the bound spreadsheet.
func (cl *Client) ListSheets() ([]SheetMeta, error) {
	if err := EnsureFresh(cl.c); err != nil {
		return nil, err
	}
	u := fmt.Sprintf("%s/%s?fields=sheets(properties)", sheetsAPIBase, cl.spreadsheetID)
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	resp, err := cl.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sheets: list: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sheets: list HTTP %d: %s", resp.StatusCode, truncate(string(body), 240))
	}
	var sr spreadsheetResp
	if err := json.Unmarshal(body, &sr); err != nil {
		return nil, fmt.Errorf("sheets: parse list: %w", err)
	}
	out := make([]SheetMeta, 0, len(sr.Sheets))
	for _, s := range sr.Sheets {
		out = append(out, s.Properties)
	}
	return out, nil
}

// ValueRange is the shape used by values.get / values.update.
type ValueRange struct {
	Range  string     `json:"range"`
	Values [][]string `json:"values"`
}

// ReadRange returns the A1-notated range. Pass empty majorDimension to
// default to ROWS.
func (cl *Client) ReadRange(a1Range string) (*ValueRange, error) {
	if err := EnsureFresh(cl.c); err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("valueRenderOption", "FORMATTED_VALUE")
	u := fmt.Sprintf("%s/%s/values/%s?%s", sheetsAPIBase, cl.spreadsheetID, url.PathEscape(a1Range), q.Encode())
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	resp, err := cl.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sheets: read: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sheets: read HTTP %d: %s", resp.StatusCode, truncate(string(body), 240))
	}
	var vr ValueRange
	if err := json.Unmarshal(body, &vr); err != nil {
		return nil, fmt.Errorf("sheets: parse read: %w", err)
	}
	return &vr, nil
}

// WriteValues replaces values in a range. values must be [][]string,
// row-major. valueInputOption "USER_ENTERED" parses formulas and types.
func (cl *Client) WriteValues(a1Range string, values [][]string, valueInputOption string) (*UpdateResponse, error) {
	if err := EnsureFresh(cl.c); err != nil {
		return nil, err
	}
	if valueInputOption == "" {
		valueInputOption = "USER_ENTERED"
	}
	q := url.Values{}
	q.Set("valueInputOption", valueInputOption)
	u := fmt.Sprintf("%s/%s/values/%s?%s", sheetsAPIBase, cl.spreadsheetID, url.PathEscape(a1Range), q.Encode())

	body, err := json.Marshal(ValueRange{Range: a1Range, Values: values})
	if err != nil {
		return nil, fmt.Errorf("sheets: marshal write: %w", err)
	}
	req, _ := http.NewRequest(http.MethodPut, u, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := cl.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sheets: write: %w", err)
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sheets: write HTTP %d: %s", resp.StatusCode, truncate(string(rb), 240))
	}
	var ur UpdateResponse
	if err := json.Unmarshal(rb, &ur); err != nil {
		return nil, fmt.Errorf("sheets: parse write: %w", err)
	}
	return &ur, nil
}

// UpdateResponse is the Sheets API v4 response for values.update.
type UpdateResponse struct {
	SpreadsheetID  string `json:"spreadsheetId"`
	UpdatedRange   string `json:"updatedRange"`
	UpdatedRows    int    `json:"updatedRows"`
	UpdatedColumns int    `json:"updatedColumns"`
	UpdatedCells   int    `json:"updatedCells"`
}

// AddSheetRequest is the body for spreadsheets.batchUpdate to add a tab.
type AddSheetRequest struct {
	Requests []map[string]any `json:"requests"`
}

// AddSheet appends a new tab with the given title. Returns the new sheetId.
// Color via RGB {red,green,blue} with values in 0.0-1.0; pass nil for default.
func (cl *Client) AddSheet(title string, rgb map[string]float64) (int64, error) {
	props := map[string]any{"title": title}
	if rgb != nil {
		props["tabColor"] = map[string]any{
			"red":   rgb["red"],
			"green": rgb["green"],
			"blue":  rgb["blue"],
		}
	}
	body, err := json.Marshal(AddSheetRequest{
		Requests: []map[string]any{
			{"addSheet": map[string]any{"properties": props}},
		},
	})
	if err != nil {
		return 0, fmt.Errorf("sheets: marshal addSheet: %w", err)
	}
	u := fmt.Sprintf("%s/%s:batchUpdate", sheetsAPIBase, cl.spreadsheetID)
	req, _ := http.NewRequest(http.MethodPost, u, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := cl.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("sheets: addSheet: %w", err)
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("sheets: addSheet HTTP %d: %s", resp.StatusCode, truncate(string(rb), 240))
	}
	// Sheets API returns replies as [ { addSheet: { properties: {...} } } ].
	// Decode defensively into a generic shape so nested parsing quirks don't bite us.
	var br struct {
		Replies []json.RawMessage `json:"replies"`
	}
	if err := json.Unmarshal(rb, &br); err != nil {
		return 0, fmt.Errorf("sheets: parse addSheet: %w", err)
	}
	if len(br.Replies) == 0 {
		return 0, fmt.Errorf("sheets: addSheet returned no replies: %s", truncate(string(rb), 240))
	}
	// Walk the nested map manually.
	var replyMap map[string]json.RawMessage
	if err := json.Unmarshal(br.Replies[0], &replyMap); err != nil {
		return 0, fmt.Errorf("sheets: parse addSheet reply: %w", err)
	}
	addSheetRaw, ok := replyMap["addSheet"]
	if !ok {
		return 0, fmt.Errorf("sheets: addSheet key missing: %s", string(br.Replies[0]))
	}
	var addSheet struct {
		Properties struct {
			SheetID int64  `json:"sheetId"`
			Title   string `json:"title"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(addSheetRaw, &addSheet); err != nil {
		return 0, fmt.Errorf("sheets: parse addSheet.properties: %w", err)
	}
	return addSheet.Properties.SheetID, nil
}

// DeleteSheet removes a tab by sheetId. The operation is irreversible
// from the API side; callers should snapshot first if there's any
// doubt. We require the caller to pass the sheetId explicitly so a
// typo can never wipe the wrong tab.
func (cl *Client) DeleteSheet(sheetID int64) error {
	if err := EnsureFresh(cl.c); err != nil {
		return err
	}
	body, err := json.Marshal(map[string]any{
		"requests": []map[string]any{
			{"deleteSheet": map[string]any{"sheetId": sheetID}},
		},
	})
	if err != nil {
		return fmt.Errorf("sheets: marshal deleteSheet: %w", err)
	}
	u := fmt.Sprintf("%s/%s:batchUpdate", sheetsAPIBase, cl.spreadsheetID)
	req, _ := http.NewRequest(http.MethodPost, u, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := cl.http.Do(req)
	if err != nil {
		return fmt.Errorf("sheets: deleteSheet: %w", err)
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sheets: deleteSheet HTTP %d: %s", resp.StatusCode, truncate(string(rb), 240))
	}
	return nil
}
