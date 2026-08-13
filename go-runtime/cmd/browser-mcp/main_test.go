package main

import "testing"

func TestLegacyHTTPModeIsDisabled(t *testing.T) {
	if err := validateLegacyHTTPMode(":18922"); err == nil {
		t.Fatal("legacy browser-mcp HTTP mode must be disabled")
	}
	if err := validateLegacyHTTPMode(""); err != nil {
		t.Fatalf("stdio mode rejected: %v", err)
	}
}
