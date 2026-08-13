// Command browser_server implements an MCP JSON-RPC 2.0 server that exposes
// browser automation tools via chromedp. Designed for OVAV agent consumption.
//
// Usage:
//
//	go run ./go-runtime/cmd/browser_server [--show|--no-headless] [--http :9222]
//
// Protocol: MCP JSON-RPC 2.0 (stdio or HTTP — auto-detects --http flag)
// Tools: browser_navigate, browser_screenshot, browser_click, browser_type,
//
//	browser_get_html, browser_get_styles, browser_evaluate
//
// HTTP mode (default recommended for OVAV agent invocation): listening on
// 127.0.0.1:9222, accepts POST requests to /mcp with JSON-RPC body.
// OVAV agents can call with: curl -X POST http://127.0.0.1:9222/mcp -d '{...}'
package main

import (
	"bufio"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ovav/ovav/internal/browser"
)

const (
	protocolVersion    = "2024-11-05"
	serverName         = "ovav-browser"
	serverVersion      = "0.1.0"
	defaultHTTPAddr    = "127.0.0.1:9222"
	maxHTTPBody        = int64(1 << 20)
	httpTokenEnv       = "OVAV_BROWSER_MCP_TOKEN"
	browserEvaluateEnv = "OVAV_BROWSER_EVALUATE"
)

// ── MCP Types ─────────────────────────────────────────────────────────────────

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
	Capabilities    Capabilities `json:"capabilities"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Capabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type CallToolResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type ContentItem struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Data     string `json:"data,omitempty"`
}

// ── Tool Definitions ──────────────────────────────────────────────────────────

var tools = []Tool{
	{
		Name:        "browser_navigate",
		Description: "Navigate browser to a URL and return page HTML.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"url": {Type: "string", Description: "Full URL to navigate to"},
			},
			Required: []string{"url"},
		},
	},
	{
		Name:        "browser_screenshot",
		Description: "Capture a screenshot of the page or a specific element. Returns base64 PNG.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"selector": {Type: "string", Description: "CSS selector of element to screenshot (empty = full page)"},
			},
		},
	},
	{
		Name:        "browser_click",
		Description: "Click an element by CSS selector.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"selector": {Type: "string", Description: "CSS selector of element to click"},
			},
			Required: []string{"selector"},
		},
	},
	{
		Name:        "browser_type",
		Description: "Type text into an input element (clears first).",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"selector": {Type: "string", Description: "CSS selector of input element"},
				"text":     {Type: "string", Description: "Text to type"},
			},
			Required: []string{"selector", "text"},
		},
	},
	{
		Name:        "browser_get_html",
		Description: "Get HTML of page or specific element.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"selector": {Type: "string", Description: "CSS selector (empty = full page HTML)"},
			},
		},
	},
	{
		Name:        "browser_get_styles",
		Description: "Get computed CSS styles for an element.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"selector": {Type: "string", Description: "CSS selector of element"},
			},
			Required: []string{"selector"},
		},
	},
	{
		Name:        "browser_evaluate",
		Description: "Execute JavaScript in the page and return the result.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"js": {Type: "string", Description: "JavaScript code to evaluate"},
			},
			Required: []string{"js"},
		},
	},
}

// ── Main ──────────────────────────────────────────────────────────────────────

type browserController interface {
	Start() error
	Stop() error
	Navigate(string) (string, error)
	Screenshot(string) (string, error)
	Click(string) error
	Type(string, string) error
	GetHTML(string) (string, error)
	GetComputedStyles(string) ([]browser.ComputedStyle, error)
	Evaluate(string) (string, error)
	GetURL() (string, error)
}

type mcpServer struct {
	mu          sync.Mutex
	ctrl        browserController
	started     bool
	root        string
	validateURL func(string) error
}

func newMCPServer(ctrl browserController) *mcpServer {
	root, _ := os.Getwd()
	return &mcpServer{ctrl: ctrl, root: root, validateURL: browser.NewURLFirewall(root).Validate}
}

func main() {
	headless := true
	httpAddr := ""
	for _, a := range os.Args[1:] {
		switch a {
		case "--show", "--no-headless":
			headless = false
		default:
			if strings.HasPrefix(a, "--http=") || strings.HasPrefix(a, "--http") {
				httpAddr = strings.TrimPrefix(strings.TrimPrefix(a, "--http="), "--http")
				if httpAddr == "" {
					httpAddr = defaultHTTPAddr
				}
			}
		}
	}

	server := newMCPServer(browser.New(browser.WithHeadless(headless)))
	if root, err := browserRepoRoot(); err == nil {
		server.root = root
		server.validateURL = browser.NewURLFirewall(root).Validate
	}
	defer func() {
		if err := server.stop(); err != nil {
			fmt.Fprintf(os.Stderr, "MCP ERROR: browser stop failed: %v\n", err)
		}
	}()

	fmt.Fprintf(os.Stderr, "OVAV Browser MCP v%s started (headless=%v)\n", serverVersion, headless)

	if httpAddr != "" {
		addr, err := normalizeHTTPBind(httpAddr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "HTTP ERROR: %v\n", err)
			return
		}
		token := os.Getenv(httpTokenEnv)
		if token == "" {
			fmt.Fprintf(os.Stderr, "HTTP ERROR: %s is required for HTTP mode\n", httpTokenEnv)
			return
		}
		runHTTP(addr, token, server)
	} else {
		runStdio(server)
	}
}

func browserRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".ovav", "registry", "network_allowlist.yaml")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("OVAV repository root not found")
		}
		dir = parent
	}
}

func runStdio(server *mcpServer) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for screenshots

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			respond(&JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      nil,
				Error:   &RPCError{Code: -32700, Message: "Parse error", Data: err.Error()},
			})
			continue
		}

		if resp := server.handleRequest(req); resp != nil {
			respond(resp)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "MCP FATAL: %v\n", err)
		os.Exit(1)
	}
}

func runHTTP(addr, token string, server *mcpServer) {
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           newHTTPHandler(server, token, maxHTTPBody),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	fmt.Fprintf(os.Stderr, "HTTP server listening on http://%s/mcp\n", addr)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(os.Stderr, "HTTP ERROR: %v\n", err)
	}
}

func newHTTPHandler(server *mcpServer, token string, maxBody int64) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBody))
		if err != nil {
			status := http.StatusBadRequest
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				status = http.StatusRequestEntityTooLarge
			}
			http.Error(w, http.StatusText(status), status)
			return
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(body, &req); err != nil {
			writeHTTPResponse(w, JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &RPCError{Code: -32700, Message: "Parse error", Data: err.Error()}})
			return
		}

		resp := server.handleRequest(req)
		if resp == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeHTTPResponse(w, *resp)
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func normalizeHTTPBind(addr string) (string, error) {
	if addr == "" {
		addr = defaultHTTPAddr
	} else if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil || port == "" {
		return "", fmt.Errorf("invalid HTTP bind %q", addr)
	}
	if strings.EqualFold(host, "localhost") {
		return net.JoinHostPort(host, port), nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return "", fmt.Errorf("HTTP mode requires a loopback bind, got %q", host)
	}
	return net.JoinHostPort(host, port), nil
}

func writeHTTPResponse(w http.ResponseWriter, resp JSONRPCResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *mcpServer) handleRequest(req JSONRPCRequest) *JSONRPCResponse {
	var result interface{}
	var rpcErr *RPCError

	switch req.Method {
	case "initialize":
		result = InitializeResult{
			ProtocolVersion: protocolVersion,
			ServerInfo:      ServerInfo{Name: serverName, Version: serverVersion},
			Capabilities:    Capabilities{Tools: &ToolsCapability{ListChanged: false}},
		}

	case "initialized", "notifications/initialized":
		result = map[string]string{}

	case "tools/list":
		result = map[string]interface{}{"tools": availableTools()}

	case "tools/call":
		var p CallToolParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			rpcErr = &RPCError{Code: -32602, Message: "Invalid params", Data: err.Error()}
		} else {
			result, rpcErr = s.callTool(p)
		}

	default:
		rpcErr = &RPCError{Code: -32601, Message: fmt.Sprintf("Method not found: %s", req.Method)}
	}

	if req.ID == nil {
		return nil
	}
	return &JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: result, Error: rpcErr}
}

func availableTools() []Tool {
	if os.Getenv(browserEvaluateEnv) == "1" {
		return tools
	}
	filtered := make([]Tool, 0, len(tools)-1)
	for _, tool := range tools {
		if tool.Name != "browser_evaluate" {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}

func (s *mcpServer) callTool(p CallToolParams) (*CallToolResult, *RPCError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !isBrowserTool(p.Name) {
		return nil, &RPCError{Code: -32601, Message: fmt.Sprintf("Tool not found: %s", p.Name)}
	}
	if p.Name == "browser_evaluate" && os.Getenv(browserEvaluateEnv) != "1" {
		return toolError(fmt.Errorf("browser_evaluate disabled; set %s=1 for an explicit capability grant", browserEvaluateEnv)), nil
	}
	if !s.started {
		if err := s.ctrl.Start(); err != nil {
			return toolError(err), nil
		}
		s.started = true
	}

	switch p.Name {
	case "browser_navigate":
		target, _ := p.Arguments["url"].(string)
		if err := s.validateURL(target); err != nil {
			return toolError(err), nil
		}
		html, err := s.ctrl.Navigate(target)
		if err != nil {
			return toolError(err), nil
		}
		finalURL, err := s.ctrl.GetURL()
		if err != nil {
			return toolError(fmt.Errorf("verify navigation target: %w", err)), nil
		}
		if err := s.validateURL(finalURL); err != nil {
			_ = s.ctrl.Stop()
			s.started = false
			return toolError(fmt.Errorf("browser redirect target denied: %w", err)), nil
		}
		return &CallToolResult{Content: []ContentItem{{Type: "text", Text: truncate(html, 50000)}}}, nil

	case "browser_screenshot":
		sel, _ := p.Arguments["selector"].(string)
		png, err := s.ctrl.Screenshot(sel)
		if err != nil {
			return toolError(err), nil
		}
		return &CallToolResult{Content: []ContentItem{
			{Type: "image", Data: png, MimeType: "image/png"},
			{Type: "text", Text: fmt.Sprintf("Screenshot captured (%s bytes base64)", humanSize(len(png)))},
		}}, nil

	case "browser_click":
		sel, _ := p.Arguments["selector"].(string)
		err := s.ctrl.Click(sel)
		if err != nil {
			return toolError(err), nil
		}
		return &CallToolResult{Content: []ContentItem{{Type: "text", Text: fmt.Sprintf("Clicked: %s", sel)}}}, nil

	case "browser_type":
		sel, _ := p.Arguments["selector"].(string)
		text, _ := p.Arguments["text"].(string)
		err := s.ctrl.Type(sel, text)
		if err != nil {
			return toolError(err), nil
		}
		return &CallToolResult{Content: []ContentItem{{Type: "text", Text: fmt.Sprintf("Typed %q into %s", text, sel)}}}, nil

	case "browser_get_html":
		sel, _ := p.Arguments["selector"].(string)
		html, err := s.ctrl.GetHTML(sel)
		if err != nil {
			return toolError(err), nil
		}
		return &CallToolResult{Content: []ContentItem{{Type: "text", Text: truncate(html, 50000)}}}, nil

	case "browser_get_styles":
		sel, _ := p.Arguments["selector"].(string)
		styles, err := s.ctrl.GetComputedStyles(sel)
		if err != nil {
			return toolError(err), nil
		}
		var sb strings.Builder
		for _, s := range styles {
			sb.WriteString(fmt.Sprintf("%s: %s\n", s.Property, s.Value))
		}
		return &CallToolResult{Content: []ContentItem{{Type: "text", Text: sb.String()}}}, nil

	case "browser_evaluate":
		js, _ := p.Arguments["js"].(string)
		result, err := s.ctrl.Evaluate(js)
		if err != nil {
			return toolError(err), nil
		}
		return &CallToolResult{Content: []ContentItem{{Type: "text", Text: result}}}, nil

	default:
		return nil, &RPCError{Code: -32601, Message: fmt.Sprintf("Tool not found: %s", p.Name)}
	}
}

func isBrowserTool(name string) bool {
	switch name {
	case "browser_navigate", "browser_screenshot", "browser_click", "browser_type", "browser_get_html", "browser_get_styles", "browser_evaluate":
		return true
	default:
		return false
	}
}

func toolError(err error) *CallToolResult {
	return &CallToolResult{IsError: true, Content: []ContentItem{{Type: "text", Text: err.Error()}}}
}

func (s *mcpServer) stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started {
		return nil
	}
	s.started = false
	return s.ctrl.Stop()
}

func respond(resp *JSONRPCResponse) {
	out, _ := json.Marshal(resp)
	fmt.Println(string(out))
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "... [truncated]"
}

func humanSize(n int) string {
	if n < 1024 {
		return fmt.Sprintf("%dB", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%dKB", n/1024)
	}
	return fmt.Sprintf("%dMB", n/(1024*1024))
}
