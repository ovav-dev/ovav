package ows

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestParseWorktreeEntries(t *testing.T) {
	out := strings.Join([]string{
		"worktree /repo",
		"HEAD abc123",
		"branch refs/heads/develop",
		"",
		"worktree /repo/feature",
		"HEAD def456",
		"branch refs/heads/feature/demo",
		"locked reason: active review",
		"",
		"worktree /repo/detached",
		"HEAD 789abc",
		"detached",
		"prunable stale metadata",
	}, "\n")

	entries := parseWorktreeEntries(out)
	if len(entries) != 3 {
		t.Fatalf("parseWorktreeEntries() returned %d entries, want 3", len(entries))
	}
	if entries[0].Branch != "develop" || entries[0].Head != "abc123" {
		t.Fatalf("main entry = %+v", entries[0])
	}
	if !entries[1].Locked || entries[1].LockReason != "reason: active review" {
		t.Fatalf("locked entry = %+v", entries[1])
	}
	if !entries[2].Detached || !entries[2].Prunable {
		t.Fatalf("detached entry = %+v", entries[2])
	}
}

func TestFilterWorktreeEntriesMineAndStale(t *testing.T) {
	t.Setenv("OVAV_ACTIVE_LEAD", "thavren")
	entries := []worktreeListEntry{
		{Path: "/repo", Branch: "develop", Current: true},
		{Path: "/repo/mine", Branch: "feature/mine", Owner: "thavren", Stale: true},
		{Path: "/repo/other", Branch: "feature/other", Owner: "dante", Stale: true},
	}

	mine := filterWorktreeEntries(entries, "/repo", true, false)
	if len(mine) != 2 {
		t.Fatalf("mine filter returned %d entries, want 2", len(mine))
	}
	stale := filterWorktreeEntries(entries, "/repo", false, true)
	if len(stale) != 2 {
		t.Fatalf("stale filter returned %d entries, want 2", len(stale))
	}
}

func TestEncodeWorktreeEntriesJSON(t *testing.T) {
	entries := []worktreeListEntry{{Path: "/repo", Branch: "develop", State: "ACTIVE"}}
	old := os.Stdout
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = write
	err = encodeWorktreeEntries(entries)
	_ = write.Close()
	os.Stdout = old
	if err != nil {
		t.Fatal(err)
	}
	var decoded []worktreeListEntry
	if err := json.NewDecoder(read).Decode(&decoded); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if len(decoded) != 1 || decoded[0].Branch != "develop" {
		t.Fatalf("decoded = %+v", decoded)
	}
}
