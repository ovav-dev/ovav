package browser

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestMCPServerDoesNotAdvertiseEvaluateWithoutGrant(t *testing.T) {
	t.Setenv(browserEvaluateEnv, "")
	server := NewMCPServer(New())
	response := server.HandleRequest(MCPRequest{JSONRPC: "2.0", ID: 1, Method: "tools/list"})
	result, ok := response.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("tools/list result type = %T", response.Result)
	}
	listed, ok := result["tools"].([]map[string]interface{})
	if !ok {
		t.Fatalf("tools type = %T", result["tools"])
	}
	for _, tool := range listed {
		if tool["name"] == "browser_evaluate" {
			t.Fatal("browser_evaluate advertised without capability grant")
		}
	}

	t.Setenv(browserEvaluateEnv, "1")
	if server.ToolCount() != len(tools) {
		t.Fatalf("granted tool count = %d, want %d", server.ToolCount(), len(tools))
	}
}

func TestURLFirewall(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".ovav", "registry", "network_allowlist.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("domains:\n  - pattern: example.com\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	firewall := NewURLFirewall(root)
	firewall.lookup = func(context.Context, string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("93.184.216.34")}, nil
	}
	for _, target := range []string{"https://example.com/docs", "http://localhost:9222/json"} {
		if err := firewall.Validate(target); err != nil {
			t.Errorf("Validate(%q) error = %v", target, err)
		}
	}
	for _, target := range []string{"file:///etc/passwd", "https://u:p@example.com", "http://169.254.169.254", "https://denied.invalid"} {
		if err := firewall.Validate(target); err == nil {
			t.Errorf("Validate(%q) accepted unsafe URL", target)
		}
	}
	firewall.lookup = func(context.Context, string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("10.0.0.1")}, nil
	}
	if err := firewall.Validate("https://example.com"); err == nil {
		t.Fatal("allowlisted hostname resolving private was accepted")
	}
}
