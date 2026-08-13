package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/ovav/ovav/internal/browser"
)

type fakeBrowserController struct {
	mu         sync.Mutex
	startCalls int
	stopCalls  int
	startErr   error
	navigated  string
	currentURL string
	evaluated  int
}

func (f *fakeBrowserController) Start() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.startCalls++
	return f.startErr
}

func (f *fakeBrowserController) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopCalls++
	return nil
}

func (f *fakeBrowserController) Navigate(target string) (string, error) {
	f.navigated = target
	if f.currentURL == "" {
		f.currentURL = target
	}
	return "<html></html>", nil
}
func (f *fakeBrowserController) Screenshot(string) (string, error) { return "png", nil }
func (f *fakeBrowserController) Click(string) error                { return nil }
func (f *fakeBrowserController) Type(string, string) error         { return nil }
func (f *fakeBrowserController) GetHTML(string) (string, error)    { return "<html></html>", nil }
func (f *fakeBrowserController) GetComputedStyles(string) ([]browser.ComputedStyle, error) {
	return nil, nil
}
func (f *fakeBrowserController) GetURL() (string, error) { return f.currentURL, nil }

func TestHTTPHandlerRequiresBearerAndLimitsBody(t *testing.T) {
	server := newMCPServer(&fakeBrowserController{})
	handler := newHTTPHandler(server, "test-token", 128)

	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("missing token status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	health := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthResponse := httptest.NewRecorder()
	handler.ServeHTTP(healthResponse, health)
	if healthResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated health status = %d, want %d", healthResponse.Code, http.StatusUnauthorized)
	}

	request = httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBuffer(make([]byte, 129)))
	request.Header.Set("Authorization", "Bearer test-token")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("large body status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestNormalizeHTTPBindRejectsNonLoopback(t *testing.T) {
	for _, addr := range []string{"0.0.0.0:9222", "[::]:9222", "192.168.1.2:9222", "example.com:9222"} {
		if _, err := normalizeHTTPBind(addr); err == nil {
			t.Errorf("normalizeHTTPBind(%q) accepted non-loopback bind", addr)
		}
	}
	for _, addr := range []string{"", ":9222", "127.0.0.1:9222", "localhost:9222", "[::1]:9222"} {
		if _, err := normalizeHTTPBind(addr); err != nil {
			t.Errorf("normalizeHTTPBind(%q) error = %v", addr, err)
		}
	}
}

func TestBrowserNavigateUsesURLFirewall(t *testing.T) {
	root := t.TempDir()
	writeBrowserFile(t, root, ".ovav/registry/network_allowlist.yaml", "domains:\n  - pattern: example.com\n")
	ctrl := &fakeBrowserController{}
	server := newMCPServer(ctrl)
	server.root = root
	server.validateURL = browser.NewURLFirewall(root).Validate

	for _, target := range []string{
		"file:///etc/passwd",
		"https://user:pass@example.com/",
		"http://169.254.169.254/latest/meta-data",
		"https://unapproved.invalid/",
	} {
		result, rpcErr := server.callTool(CallToolParams{Name: "browser_navigate", Arguments: map[string]interface{}{"url": target}})
		if rpcErr != nil || result == nil || !result.IsError {
			t.Errorf("navigate %q = %#v, %v; want tool error", target, result, rpcErr)
		}
	}
	result, rpcErr := server.callTool(CallToolParams{Name: "browser_navigate", Arguments: map[string]interface{}{"url": "https://example.com/docs"}})
	if rpcErr != nil || result == nil || result.IsError || ctrl.navigated != "https://example.com/docs" {
		t.Fatalf("allowlisted navigate = %#v, %v, target %q", result, rpcErr, ctrl.navigated)
	}
}

func writeBrowserFile(t *testing.T, root, path, content string) {
	t.Helper()
	abs := root + "/" + path
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
func (f *fakeBrowserController) Evaluate(string) (string, error) {
	f.evaluated++
	return "result", nil
}

func TestBrowserEvaluateRequiresExplicitCapabilityGrant(t *testing.T) {
	t.Setenv(browserEvaluateEnv, "")
	for _, tool := range availableTools() {
		if tool.Name == "browser_evaluate" {
			t.Fatal("browser_evaluate advertised without capability grant")
		}
	}
	ctrl := &fakeBrowserController{}
	server := newMCPServer(ctrl)
	result, rpcErr := server.callTool(CallToolParams{Name: "browser_evaluate", Arguments: map[string]interface{}{"js": "fetch('https://example.com')"}})
	if rpcErr != nil || result == nil || !result.IsError || ctrl.evaluated != 0 {
		t.Fatalf("evaluate result=%#v rpcErr=%v calls=%d; want denied without execution", result, rpcErr, ctrl.evaluated)
	}

	t.Setenv(browserEvaluateEnv, "1")
	found := false
	for _, tool := range availableTools() {
		found = found || tool.Name == "browser_evaluate"
	}
	if !found {
		t.Fatal("browser_evaluate not advertised with capability grant")
	}
	result, rpcErr = server.callTool(CallToolParams{Name: "browser_evaluate", Arguments: map[string]interface{}{"js": "document.title"}})
	if rpcErr != nil || result == nil || result.IsError || ctrl.evaluated != 1 {
		t.Fatalf("granted evaluate result=%#v rpcErr=%v calls=%d", result, rpcErr, ctrl.evaluated)
	}
}

func TestBrowserNavigateRejectsForbiddenRedirectTarget(t *testing.T) {
	root := t.TempDir()
	writeBrowserFile(t, root, ".ovav/registry/network_allowlist.yaml", "domains:\n  - pattern: example.com\n")
	ctrl := &fakeBrowserController{currentURL: "http://169.254.169.254/latest/meta-data"}
	server := newMCPServer(ctrl)
	server.validateURL = func(target string) error {
		if target == "https://example.com/start" {
			return nil
		}
		return errors.New("redirect target denied")
	}
	result, rpcErr := server.callTool(CallToolParams{Name: "browser_navigate", Arguments: map[string]interface{}{"url": "https://example.com/start"}})
	if rpcErr != nil || result == nil || !result.IsError || !strings.Contains(result.Content[0].Text, "redirect") {
		t.Fatalf("redirect result=%#v rpcErr=%v, want denial", result, rpcErr)
	}
}

func (f *fakeBrowserController) calls() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.startCalls, f.stopCalls
}

func TestMetadataRequestsDoNotStartBrowser(t *testing.T) {
	ctrl := &fakeBrowserController{}
	server := newMCPServer(ctrl)

	tests := []struct {
		name   string
		method string
	}{
		{name: "initialize", method: "initialize"},
		{name: "tools list", method: "tools/list"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := server.handleRequest(JSONRPCRequest{JSONRPC: "2.0", ID: json.RawMessage("1"), Method: tt.method})
			if resp == nil || resp.Error != nil {
				t.Fatalf("handleRequest() = %#v, want successful response", resp)
			}
		})
	}

	if starts, _ := ctrl.calls(); starts != 0 {
		t.Fatalf("Start() calls = %d, want 0", starts)
	}
}

func TestConcurrentToolCallsStartBrowserOnceAndStopCleanly(t *testing.T) {
	ctrl := &fakeBrowserController{}
	server := newMCPServer(ctrl)
	params := json.RawMessage(`{"name":"browser_get_html","arguments":{}}`)

	const calls = 20
	var wg sync.WaitGroup
	for i := 0; i < calls; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp := server.handleRequest(JSONRPCRequest{JSONRPC: "2.0", ID: json.RawMessage("1"), Method: "tools/call", Params: params})
			if resp == nil || resp.Error != nil {
				t.Errorf("handleRequest() = %#v, want successful response", resp)
			}
		}()
	}
	wg.Wait()

	if err := server.stop(); err != nil {
		t.Fatalf("stop() error = %v", err)
	}
	if err := server.stop(); err != nil {
		t.Fatalf("second stop() error = %v", err)
	}
	starts, stops := ctrl.calls()
	if starts != 1 || stops != 1 {
		t.Fatalf("Start() calls = %d, Stop() calls = %d; want 1, 1", starts, stops)
	}
}

func TestBrowserStartupFailureIsToolError(t *testing.T) {
	ctrl := &fakeBrowserController{startErr: errors.New("chrome unavailable")}
	server := newMCPServer(ctrl)
	params := json.RawMessage(`{"name":"browser_navigate","arguments":{"url":"http://localhost"}}`)

	resp := server.handleRequest(JSONRPCRequest{JSONRPC: "2.0", ID: json.RawMessage("7"), Method: "tools/call", Params: params})
	if resp == nil || resp.Error != nil {
		t.Fatalf("handleRequest() = %#v, want MCP tool result", resp)
	}
	result, ok := resp.Result.(*CallToolResult)
	if !ok {
		t.Fatalf("result type = %T, want *CallToolResult", resp.Result)
	}
	if !result.IsError || len(result.Content) != 1 || result.Content[0].Text != "chrome unavailable" {
		t.Fatalf("tool result = %#v, want startup error content", result)
	}

	metadata := server.handleRequest(JSONRPCRequest{JSONRPC: "2.0", ID: json.RawMessage("8"), Method: "tools/list"})
	if metadata == nil || metadata.Error != nil {
		t.Fatalf("server stopped after startup failure: %#v", metadata)
	}
	if err := server.stop(); err != nil {
		t.Fatalf("stop() error = %v", err)
	}
	starts, stops := ctrl.calls()
	if starts != 1 || stops != 0 {
		t.Fatalf("Start() calls = %d, Stop() calls = %d; want 1, 0", starts, stops)
	}
}

func TestInitializedNotificationWithoutIDHasNoResponse(t *testing.T) {
	ctrl := &fakeBrowserController{}
	server := newMCPServer(ctrl)

	resp := server.handleRequest(JSONRPCRequest{JSONRPC: "2.0", Method: "notifications/initialized"})
	if resp != nil {
		t.Fatalf("handleRequest() = %#v, want nil for notification", resp)
	}
	if starts, _ := ctrl.calls(); starts != 0 {
		t.Fatalf("Start() calls = %d, want 0", starts)
	}
}
