package main

import (
	"os"
	"path/filepath"
	"testing"
)

// ── Benchmark tests for hot paths ──────────────────────────────────────────

// BenchmarkAtomicWriteLive measures the atomic write performance.
func BenchmarkAtomicWriteLive_Small(b *testing.B) {
	dir := b.TempDir()
	live := filepath.Join(dir, "live.txt")
	content := []byte("hello world, this is a typical keybinding line\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := atomicWriteLive(live, content); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAtomicWriteLive_Large measures performance with larger content.
func BenchmarkAtomicWriteLive_Large(b *testing.B) {
	dir := b.TempDir()
	live := filepath.Join(dir, "live.txt")
	content := make([]byte, 64*1024) // 64 KB
	for i := range content {
		content[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := atomicWriteLive(live, content); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkHashBytes measures hash performance.
func BenchmarkHashBytes(b *testing.B) {
	data := make([]byte, 4096)
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hashBytes(data)
	}
}

// BenchmarkCompareITKeybindings measures IT keybinding comparison.
func BenchmarkCompareITKeybindings(b *testing.B) {
	fragment := []byte(`{
		"keybindings": [
			{"keys":"ctrl+v","id":"Terminal.PasteFromClipboard"},
			{"keys":"ctrl+c","id":"Terminal.CopyToClipboard"},
			{"keys":"ctrl+t","id":"Terminal.OpenNewTab"},
			{"keys":"ctrl+w","id":"Terminal.CloseTab"},
			{"keys":"ctrl+shift+t","id":"Terminal.ReopenClosedTab"},
			{"keys":"ctrl+shift+c","id":"Terminal.CopyToClipboard"},
			{"keys":"ctrl+shift+v","id":"Terminal.PasteFromClipboard"},
			{"keys":"alt+1","id":"Terminal.SwitchToTab1"},
			{"keys":"alt+2","id":"Terminal.SwitchToTab2"},
			{"keys":"alt+3","id":"Terminal.SwitchToTab3"}
		]
	}`)
	live := fragment

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compareITKeybindings(fragment, live)
	}
}

// BenchmarkCompareBashInputrc measures bash inputrc comparison.
func BenchmarkCompareBashInputrc(b *testing.B) {
	fragment := []byte(`# OVAV inputrc
set show-all-if-ambiguous on
set completion-ignore-case on
set editing-mode vi
set mark-symlinked-directories on
set match-hidden-files on
set page-completions off
set print-completions-horizontally off
set show-all-if-ambiguous on
set visible-stats off
"\\C-x\\C-e": edit-and-execute-command
"\\C-x\\C-r": re-read-init-file
`)
	live := fragment

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compareBashInputrc(fragment, live)
	}
}

// BenchmarkBuildDriftReport measures end-to-end drift report generation.
func BenchmarkBuildDriftReport(b *testing.B) {
	root := b.TempDir()
	// Create all needed directories + a sample fragment
	os.MkdirAll(filepath.Join(root, ".ovav", "plan"), 0o755)
	os.WriteFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"),
		[]byte("# test\ncanonical: test\n"), 0o644)
	os.MkdirAll(filepath.Join(root, "workstation", "configs", "inputrc"), 0o755)
	os.WriteFile(filepath.Join(root, "workstation", "configs", "inputrc", "ovav.inputrc"),
		[]byte("test\n"), 0o644)
	os.MkdirAll(filepath.Join(root, ".ovav", "integrity_backups"), 0o755)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = buildDriftReport(root, "")
	}
}
