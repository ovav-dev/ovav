// sheets_test.go — minimal end-to-end coverage for the bridge.
//
// We test the pure-GO parts (xlsx round-trip, table helpers, snapshot
// index, allowlist load) with no network. The Sheets-API paths are
// covered by hand in the smoke-test section; full integration tests
// run only when GOOGLE_SHEETS_TOKEN is set in env.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestColLetterRoundTrip(t *testing.T) {
	pairs := map[int]string{
		1: "A", 2: "B", 26: "Z", 27: "AA", 28: "AB", 52: "AZ", 53: "BA",
	}
	for n, want := range pairs {
		if g := colLetter(n); g != want {
			t.Fatalf("colLetter(%d)=%q want %q", n, g, want)
		}
		if g := colLetterInverse(want); g != n {
			t.Fatalf("colLetterInverse(%q)=%d want %d", want, g, n)
		}
	}
}

func TestIndexOf(t *testing.T) {
	h := []string{"a", "b", "c"}
	if g := indexOf(h, "b"); g != 1 {
		t.Fatalf("indexOf(b)=%d", g)
	}
	if g := indexOf(h, "z"); g != -1 {
		t.Fatalf("indexOf(z)=%d", g)
	}
}

func TestAllowlistLoadAndRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sheets_allowlist.yaml")
	if err := loadAllowlist(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var file AllowlistFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		t.Fatal(err)
	}
	if len(file.Spreadsheets) != 1 || file.Spreadsheets[0].ID == "" {
		t.Fatalf("seeded allowlist wrong: %+v", file)
	}
	// add
	if err := cmdAllowlistAdd([]string{"--id", "TEST-ID-1", "--name", "test"}); err != nil {
		t.Fatal(err)
	}
	if err := loadAllowlist(path); err != nil {
		t.Fatal(err)
	}
	if _, ok := globalAllowlist.entries["TEST-ID-1"]; !ok {
		t.Fatal("added id not present")
	}
	// remove
	if err := cmdAllowlistRemove([]string{"--id", "TEST-ID-1"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := globalAllowlist.entries["TEST-ID-1"]; ok {
		t.Fatal("removed id still present")
	}
}

func TestXlsxRoundTrip(t *testing.T) {
	// First just dump what's actually inside.
	cells := []XlsxCell{
		{Ref: "A1", Val: "Name"}, {Ref: "B1", Val: "Age"},
		{Ref: "A2", Val: "Ada"}, {Ref: "B2", Val: "36"},
		{Ref: "A3", Val: "Linus"}, {Ref: "B3", Val: "55"},
	}
	out, err := WriteXlsx("people", cells, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	tmp := filepath.Join(t.TempDir(), "rt.xlsx")
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		t.Fatal(err)
	}
	// Dump for visibility.
	_ = os.WriteFile(filepath.Join(t.TempDir(), "rt.xlsx"), out, 0o644)
	t.Logf("wrote %d bytes to %s", len(out), tmp)
	t.Logf("xmlEscape('people') = %q", xmlEscape("people"))

	sheets, err := ReadXlsx(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if len(sheets) != 1 {
		t.Fatalf("want 1 sheet got %d", len(sheets))
	}
	if sheets[0].Name != "people" {
		t.Fatalf("name=%q", sheets[0].Name)
	}
	rows := XlsxToCells(sheets[0])
	if len(rows) != 3 {
		t.Fatalf("rows=%d want 3", len(rows))
	}
	if rows[0][0] != "Name" || rows[1][0] != "Ada" || rows[2][1] != "55" {
		t.Fatalf("round-trip mismatch: %+v", rows)
	}
}

func TestSnapshotIndexRoundTrip(t *testing.T) {
	dir := t.TempDir()
	repoRoot := dir
	// create vault.key so encrypt works
	if _, err := loadVaultKey(repoRoot); err != nil {
		t.Fatal(err)
	}
	// fake client capture
	cl := &Client{spreadsheetID: "TEST-SID", c: &Creds{}}
	payload := []byte(`{"range":"A1","values":[["x"]]}`)
	rec, err := cl.Capture(repoRoot, "test-op", "summary", "test-tab", payload)
	if err != nil {
		t.Fatal(err)
	}
	if rec.SpreadsheetID != "TEST-SID" {
		t.Fatalf("rec=%+v", rec)
	}
	// index readable
	recs, err := ListSnapshots(repoRoot, "TEST-SID", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) == 0 || recs[0].ID != rec.ID {
		t.Fatalf("list mismatch: %+v", recs)
	}
	// decryption round-trip
	pt, err := LoadSnapshot(repoRoot, &recs[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(pt) != string(payload) {
		t.Fatalf("decrypted mismatch: %s", pt)
	}
}

func TestSanitize(t *testing.T) {
	if s := sanitize("foo bar/baz?x=1"); s != "foo_bar_baz_x_1" {
		t.Fatalf("sanitize=%q", s)
	}
}

func TestAuditAppend(t *testing.T) {
	dir := t.TempDir()
	audit(dir, "SID", "Tab", "test_op", map[string]any{"k": "v"}, 5)
	files, err := os.ReadDir(filepath.Join(dir, ".ovav", "registry", "audit", "sheets"))
	if err != nil || len(files) == 0 {
		t.Fatalf("audit dir missing/empty: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, ".ovav", "registry", "audit", "sheets", files[0].Name()))
	var ev AuditEvent
	if err := json.Unmarshal(extractLine(data), &ev); err != nil {
		t.Fatal(err)
	}
	if ev.Operation != "test_op" || ev.Affected != 5 {
		t.Fatalf("event=%+v", ev)
	}
}

func extractLine(b []byte) []byte {
	for i, ch := range b {
		if ch == '\n' {
			return b[:i]
		}
	}
	return b
}
