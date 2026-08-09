package main

import (
	"reflect"
	"testing"
)

// TestResolvePackID_ExplicitPackID: --pack-id wins over --full.
func TestResolvePackID_ExplicitPackID(t *testing.T) {
	args := []string{"--pack-id=custom_pack", "--full"}
	got := resolvePackID(args)
	if got != "custom_pack" {
		t.Errorf("expected custom_pack, got %s", got)
	}
}

// TestResolvePackID_OnlyFull: --full maps to build8_source_local_apply_pack.
func TestResolvePackID_OnlyFull(t *testing.T) {
	args := []string{"--full"}
	got := resolvePackID(args)
	if got != "build8_source_local_apply_pack" {
		t.Errorf("expected build8_source_local_apply_pack, got %s", got)
	}
}

// TestResolvePackID_PackIDDashSpace: --pack-id <value> (space-separated).
func TestResolvePackID_PackIDDashSpace(t *testing.T) {
	args := []string{"--pack-id", "space_pack"}
	got := resolvePackID(args)
	if got != "space_pack" {
		t.Errorf("expected space_pack, got %s", got)
	}
}

// TestResolvePackID_NoFlag: empty args → default (which is "default" sentinel).
// Caller is expected to handle "default" upstream (now with helpful error).
func TestResolvePackID_NoFlag(t *testing.T) {
	args := []string{"install"} // subcommand itself
	got := resolvePackID(args)
	if got != "default" {
		t.Errorf("expected default, got %s", got)
	}
}

// TestResolvePackID_AllFlagShapes: order-independent.
func TestResolvePackID_AllFlagShapes(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want string
	}{
		{"explicit_equals", []string{"--pack-id=foo"}, "foo"},
		{"explicit_space", []string{"--pack-id", "foo"}, "foo"},
		{"full_only", []string{"--full"}, "build8_source_local_apply_pack"},
		{"full_and_explicit_wins", []string{"--full", "--pack-id=winner"}, "winner"},
		{"empty", []string{}, "default"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolvePackID(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("input=%v: got %s, want %s", tc.in, got, tc.want)
			}
		})
	}
}
