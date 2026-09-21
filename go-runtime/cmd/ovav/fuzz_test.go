package main

import (
	"strings"
	"testing"
)

// ── Fuzz tests for drift compare functions ─────────────────────────────────

// FuzzCompareITKeybindings ensures the comparator never panics on arbitrary JSON input.
func FuzzCompareITKeybindings(f *testing.F) {
	// Seed corpus
	f.Add(`{"keybindings":[{"keys":"ctrl+c","id":"Terminal.CopyToClipboard"}]}`, `{"keybindings":[{"keys":"ctrl+c","id":"Terminal.CopyToClipboard"}]}`)
	f.Add(`{"keybindings":[]}`, `{"keybindings":[]}`)
	f.Add(`{"keybindings":null}`, `{"keybindings":null}`)
	f.Add(`{}`, `{}`)
	f.Add(``, ``)
	f.Add(`not json`, `also not json`)
	f.Add(`{"keybindings":[{"keys":"ctrl+c","id":null}]}`, `{"keybindings":[{"keys":"ctrl+c","id":""}]}`)
	f.Add(`{"keybindings":[{"keys":123,"id":"x"}]}`, `{"keybindings":[{"keys":true,"id":"x"}]}`)

	f.Fuzz(func(t *testing.T, frag, live string) {
		// Must not panic
		items, _ := compareITKeybindings([]byte(frag), []byte(live))
		// Sanity: items is a slice (could be empty or have items)
		_ = items
	})
}

// FuzzCompareBashInputrc ensures the bash inputrc comparator handles arbitrary input.
func FuzzCompareBashInputrc(f *testing.F) {
	// Seed corpus
	f.Add("set show-all-if-ambiguous on\n", "set show-all-if-ambiguous on\n")
	f.Add("# comment only\n", "actual line\n")
	f.Add("", "")
	f.Add("# multi\n# line\n# comment\n", "\n\n")
	f.Add("\x00\x01\x02binary\xff", "normal text\n")

	f.Fuzz(func(t *testing.T, frag, live string) {
		// Must not panic
		items, _ := compareBashInputrc([]byte(frag), []byte(live))
		_ = items
	})
}

// FuzzComparePinnedBaseline ensures pinned baseline comparison is robust.
func FuzzComparePinnedBaseline(f *testing.F) {
	// Seed corpus
	f.Add(`{"files":{"a":"x"}}`, `{"files":{"a":"x"}}`)
	f.Add(`{"files":{}}`, `{"files":{}}`)
	f.Add(`{"files":{"a":"x","b":"y"}}`, `{"files":{"a":"x","c":"z"}}`)
	f.Add(``, ``)
	f.Add(`{"files":{"a":""}}`, `{"files":{"a":""}}`)
	f.Add(`{"files":null}`, `{"files":null}`)

	f.Fuzz(func(t *testing.T, pinned, current string) {
		// Must not panic
		items, _ := comparePinnedBaseline([]byte(pinned), []byte(current))
		_ = items
	})
}

// ── Fuzz tests for atomic write ─────────────────────────────────────────────

// FuzzAtomicWriteLive ensures the atomic write helper handles any input safely.
func FuzzAtomicWriteLive(f *testing.F) {
	f.Add("hello world")
	f.Add("")
	f.Add("\x00\x01\x02")
	f.Add(strings.Repeat("a", 10000))
	f.Add("multi\nline\ncontent\n")

	f.Fuzz(func(t *testing.T, content string) {
		dir := t.TempDir()
		live := dir + "/live.txt"
		// Must not panic
		_ = atomicWriteLive(live, []byte(content))
	})
}

// FuzzHashBytes ensures hash function handles arbitrary bytes.
func FuzzHashBytes(f *testing.F) {
	f.Add([]byte(""))
	f.Add([]byte("hello"))
	f.Add([]byte{0x00, 0xFF, 0x80})

	f.Fuzz(func(t *testing.T, data []byte) {
		h := hashBytes(data)
		// SHA-256 hex is always 64 chars
		if len(h) != 64 {
			t.Errorf("hash length %d for %d bytes", len(h), len(data))
		}
	})
}
