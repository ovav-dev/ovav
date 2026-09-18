// Package main — xlsx ↔ Sheets sync using only stdlib.
//
// Reading: xlsx is a zip of XML files (sheet1.xml, sharedStrings.xml,
// styles.xml). We walk sheet1.xml, parse each <row>/<c>/<v> and
// resolve shared-string references. No styles/formatting — values
// only. Good enough for the "real work" use case (numeric + text data).
//
// Writing: we emit a minimal valid xlsx with shared strings. Layout
// follows the OOXML spec strictly enough to open in Excel, LibreOffice
// and Google Sheets.
package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// XlsxCell is the normalized form of one cell.
type XlsxCell struct {
	Ref  string // e.g. "A1"
	Val  string // resolved (type-aware) value
	Type string // "n" number, "s" string, "b" bool, "" inline
}

// XlsxSheet is one tab.
type XlsxSheet struct {
	Name           string
	Cells          []XlsxCell
	MaxRow, MaxCol int
}

// ReadXlsx parses a .xlsx file. The first (and only, for now) sheet
// is returned. Use SheetN to pick a specific sheet if needed.
func ReadXlsx(path string) ([]XlsxSheet, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("xlsx: open zip: %w", err)
	}
	defer r.Close()

	shared := map[string]string{}
	if f, err := openZipFile(r, "xl/sharedStrings.xml"); err == nil {
		_ = decodeSharedStrings(f, shared)
		f.Close()
	}
	styles := map[string]string{}
	if f, err := openZipFile(r, "xl/styles.xml"); err == nil {
		_ = decodeStyles(f, styles)
		f.Close()
	}
	// Find sheets via xl/workbook.xml
	workbook, err := openZipFile(r, "xl/workbook.xml")
	if err != nil {
		return nil, fmt.Errorf("xlsx: workbook.xml missing: %w", err)
	}
	sheetNames := decodeSheetNames(workbook)
	workbook.Close()

	sheets := []XlsxSheet{}
	for i, name := range sheetNames {
		entry := fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1)
		f, err := openZipFile(r, entry)
		if err != nil {
			// skip missing
			continue
		}
		cells, maxR, maxC, err := decodeSheet(f, shared)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("xlsx: decode %s: %w", entry, err)
		}
		sheets = append(sheets, XlsxSheet{
			Name: name, Cells: cells, MaxRow: maxR, MaxCol: maxC,
		})
	}
	if len(sheets) == 0 {
		return nil, fmt.Errorf("xlsx: no sheets found")
	}
	return sheets, nil
}

func openZipFile(r *zip.ReadCloser, name string) (io.ReadCloser, error) {
	for _, f := range r.File {
		if f.Name == name {
			return f.Open()
		}
	}
	return nil, os.ErrNotExist
}

type sharedString struct {
	XMLName xml.Name `xml:"si"`
	T       string   `xml:"t"`
	R       []struct {
		T string `xml:"t"`
	} `xml:"r"`
}

type sharedStringsXML struct {
	Items []sharedString `xml:"si"`
}

func decodeSharedStrings(r io.Reader, out map[string]string) error {
	var ss sharedStringsXML
	if err := xml.NewDecoder(r).Decode(&ss); err != nil {
		return err
	}
	for i, item := range ss.Items {
		if len(item.R) > 0 {
			var b strings.Builder
			for _, run := range item.R {
				b.WriteString(run.T)
			}
			out[strconv.Itoa(i)] = b.String()
		} else {
			out[strconv.Itoa(i)] = item.T
		}
	}
	return nil
}

type stylesXML struct {
	CellXfs struct {
		Xf []struct {
			NumFmtID string `xml:"numFmtId,attr"`
		} `xml:"xf"`
	} `xml:"cellXfs"`
	NumFmts struct {
		NumFmt []struct {
			ID   string `xml:"numFmtId,attr"`
			Code string `xml:"formatCode,attr"`
		} `xml:"numFmt"`
	} `xml:"numFmts"`
}

func decodeStyles(r io.Reader, out map[string]string) error {
	var s stylesXML
	if err := xml.NewDecoder(r).Decode(&s); err != nil {
		return err
	}
	for _, xf := range s.CellXfs.Xf {
		id := xf.NumFmtID
		// built-in number formats
		switch id {
		case "0":
			out["general"] = ""
		case "1":
			out[id] = "0"
		case "2":
			out[id] = "0.00"
		case "14":
			out[id] = "m/d/yyyy"
		default:
			for _, nf := range s.NumFmts.NumFmt {
				if nf.ID == id {
					out[id] = nf.Code
				}
			}
		}
	}
	return nil
}

type sheetXML struct {
	Rows []struct {
		R     int `xml:"r,attr"`
		Cells []struct {
			R  string `xml:"r,attr"`
			T  string `xml:"t,attr"` // type ("s","str","n","b",...)
			V  string `xml:"v"`
			Is struct {
				T string `xml:"t"`
			} `xml:"is>t"`
		} `xml:"c"`
	} `xml:"sheetData>row"`
}

func decodeSheet(r io.Reader, shared map[string]string) ([]XlsxCell, int, int, error) {
	var sh sheetXML
	if err := xml.NewDecoder(r).Decode(&sh); err != nil {
		return nil, 0, 0, err
	}
	out := make([]XlsxCell, 0)
	maxR, maxC := 0, 0
	for _, row := range sh.Rows {
		for _, c := range row.Cells {
			val := c.V
			switch c.T {
			case "s":
				val = shared[c.V]
			case "b":
				if c.V == "1" {
					val = "TRUE"
				} else {
					val = "FALSE"
				}
			case "str", "inlineStr":
				val = c.Is.T
			}
			out = append(out, XlsxCell{Ref: c.R, Val: val, Type: c.T})
			// parse col index from cell ref "A12"
			col := 0
			for i := 0; i < len(c.R); i++ {
				ch := c.R[i]
				if ch >= '0' && ch <= '9' {
					col = colLetterInverse(c.R[:i])
					break
				}
			}
			if row.R > maxR {
				maxR = row.R
			}
			if col > maxC {
				maxC = col
			}
		}
	}
	return out, maxR, maxC, nil
}

type workbookXML struct {
	Sheets []struct {
		Name string `xml:"name,attr"`
		ID   string `xml:"sheetId,attr"`
	} `xml:"sheets>sheet"`
}

func decodeSheetNames(r io.Reader) []string {
	var wb workbookXML
	if err := xml.NewDecoder(r).Decode(&wb); err != nil {
		return nil
	}
	out := make([]string, 0, len(wb.Sheets))
	for _, s := range wb.Sheets {
		out = append(out, s.Name)
	}
	return out
}

// ── Writing ─────────────────────────────────────────────────────────

// WriteXlsx writes a single-sheet .xlsx with values + shared strings.
// Returns the file as a byte slice (so callers can choose to write
// to disk, stream over HTTP, etc).
func WriteXlsx(sheetName string, cells []XlsxCell, maxRow, maxCol int) ([]byte, error) {
	if sheetName == "" {
		sheetName = "Sheet1"
	}
	if maxRow <= 0 {
		for _, c := range cells {
			row := 0
			for i := 0; i < len(c.Ref); i++ {
				ch := c.Ref[i]
				if ch >= '0' && ch <= '9' {
					row, _ = strconv.Atoi(c.Ref[i:])
					break
				}
			}
			if row > maxRow {
				maxRow = row
			}
		}
	}
	if maxCol <= 0 {
		for _, c := range cells {
			col := 0
			for i := 0; i < len(c.Ref); i++ {
				ch := c.Ref[i]
				if ch >= '0' && ch <= '9' {
					col = colLetterInverse(c.Ref[:i])
					break
				}
			}
			if col > maxCol {
				maxCol = col
			}
		}
	}

	// Group shared strings.
	ssMap := map[string]int{}
	var ssList []string
	addSS := func(v string) string {
		if idx, ok := ssMap[v]; ok {
			return strconv.Itoa(idx)
		}
		ssMap[v] = len(ssList)
		ssList = append(ssList, v)
		return strconv.Itoa(len(ssList) - 1)
	}

	// Build sheet XML by row.
	rowMap := map[int][]XlsxCell{}
	for _, c := range cells {
		row := 0
		for i := 0; i < len(c.Ref); i++ {
			ch := c.Ref[i]
			if ch >= '0' && ch <= '9' {
				row, _ = strconv.Atoi(c.Ref[i:])
				break
			}
		}
		rowMap[row] = append(rowMap[row], c)
	}
	var sheetXML strings.Builder
	sheetXML.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	for r := 1; r <= maxRow; r++ {
		rowCells := rowMap[r]
		sheetXML.WriteString(fmt.Sprintf(`<row r="%d">`, r))
		for _, c := range rowCells {
			// Decide type. If parseable as float → inline number, else shared string.
			if _, err := strconv.ParseFloat(c.Val, 64); err == nil && c.Val != "" {
				sheetXML.WriteString(fmt.Sprintf(`<c r="%s"><v>%s</v></c>`, c.Ref, xmlEscape(c.Val)))
			} else if c.Val == "TRUE" || c.Val == "FALSE" {
				bt := "0"
				if c.Val == "TRUE" {
					bt = "1"
				}
				sheetXML.WriteString(fmt.Sprintf(`<c r="%s" t="b"><v>%s</v></c>`, c.Ref, bt))
			} else {
				idx := addSS(c.Val)
				sheetXML.WriteString(fmt.Sprintf(`<c r="%s" t="s"><v>%s</v></c>`, c.Ref, idx))
			}
		}
		sheetXML.WriteString(`</row>`)
	}
	sheetXML.WriteString(`</sheetData></worksheet>`)

	var ssXML strings.Builder
	ssXML.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="` +
		strconv.Itoa(len(ssList)) + `" uniqueCount="` + strconv.Itoa(len(ssList)) + `">`)
	for _, s := range ssList {
		ssXML.WriteString(`<si><t xml:space="preserve">` + xmlEscape(s) + `</t></si>`)
	}
	ssXML.WriteString(`</sst>`)

	workbookXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"
  xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
 <sheets><sheet name="` + xmlEscape(sheetName) + `" sheetId="1" r:id="rId1"/></sheets>
</workbook>`

	rootRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
 <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`

	wbRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
 <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
 <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
 <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/>
</Relationships>`

	stylesXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
 <fonts count="1"><font><sz val="11"/><name val="Calibri"/></font></fonts>
 <fills count="1"><fill><patternFill patternType="none"/></fill></fills>
 <borders count="1"><border/></borders>
 <cellStyleXfs count="1"><xf/></cellStyleXfs>
 <cellXfs count="1"><xf/></cellXfs>
</styleSheet>`

	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
 <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
 <Default Extension="xml" ContentType="application/xml"/>
 <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
 <Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
 <Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>
 <Override PartName="/xl/sharedStrings.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"/>
</Types>`

	// zip everything
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	files := map[string][]byte{
		"[Content_Types].xml":        []byte(contentTypes),
		"_rels/.rels":                []byte(rootRels),
		"xl/workbook.xml":            []byte(workbookXML),
		"xl/_rels/workbook.xml.rels": []byte(wbRels),
		"xl/styles.xml":              []byte(stylesXML),
		"xl/sharedStrings.xml":       []byte(ssXML.String()),
		"xl/worksheets/sheet1.xml":   []byte(sheetXML.String()),
	}
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(body); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func xmlEscape(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 8)
	for _, r := range s {
		switch r {
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '&':
			b.WriteString("&amp;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&apos;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// XlsxToCells flattens an XlsxSheet into [][]string rows for the
// Sheets writer. maxRow/maxCol allow padding to keep alignment.
func XlsxToCells(sh XlsxSheet) [][]string {
	if sh.MaxRow == 0 || sh.MaxCol == 0 {
		return nil
	}
	grid := make([][]string, sh.MaxRow)
	for i := range grid {
		grid[i] = make([]string, sh.MaxCol)
	}
	for _, c := range sh.Cells {
		row := 0
		col := 0
		for i := 0; i < len(c.Ref); i++ {
			ch := c.Ref[i]
			if ch >= '0' && ch <= '9' {
				row, _ = strconv.Atoi(c.Ref[i:])
				col = colLetterInverse(c.Ref[:i])
				break
			}
		}
		if row >= 1 && row <= sh.MaxRow && col >= 1 && col <= sh.MaxCol {
			grid[row-1][col-1] = c.Val
		}
	}
	// Trim trailing empty rows
	for len(grid) > 0 && allEmpty(grid[len(grid)-1]) {
		grid = grid[:len(grid)-1]
	}
	return grid
}

func allEmpty(row []string) bool {
	for _, v := range row {
		if v != "" {
			return false
		}
	}
	return true
}
