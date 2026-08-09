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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/ovav/ovav/internal/browser"
)

const (
	protocolVersion = "2024-11-05"
	serverName      = "ovav-browser"
	serverVersion   = "0.1.0"
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

var bc *browser.Controller

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
					httpAddr = ":9222"
				}
			}
		}
	}

	bc = browser.New(browser.WithHeadless(headless))
	if err := bc.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "MCP ERROR: %v\n", err)
		os.Exit(1)
	}
	defer bc.Stop()

	fmt.Fprintf(os.Stderr, "OVAV Browser MCP v%s started (headless=%v)\n", serverVersion, headless)

	if httpAddr != "" {
		runHTTP(httpAddr)
	} else {
		runStdio()
	}
}

func runStdio() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for screenshots

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			respond(req.ID, nil, &RPCError{Code: -32700, Message: "Parse error", Data: err.Error()})
			continue
		}

		result, rpcErr := handleMethod(req.Method, req.Params)
		respond(req.ID, result, rpcErr)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "MCP FATAL: %v\n", err)
		os.Exit(1)
	}
}

func runHTTP(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(body, &req); err != nil {
			writeHTTPResponse(w, JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &RPCError{Code: -32700, Message: "Parse error", Data: err.Error()}})
			return
		}

		result, rpcErr := handleMethod(req.Method, req.Params)
		writeHTTPResponse(w, JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: result, Error: rpcErr})
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	fmt.Fprintf(os.Stderr, "HTTP server listening on http://127.0.0.1%s/mcp\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "HTTP ERROR: %v\n", err)
		os.Exit(1)
	}
}

func writeHTTPResponse(w http.ResponseWriter, resp JSONRPCResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleMethod(method string, params json.RawMessage) (interface{}, *RPCError) {
	switch method {
	case "initialize":
		return InitializeResult{
			ProtocolVersion: protocolVersion,
			ServerInfo:      ServerInfo{Name: serverName, Version: serverVersion},
			Capabilities:    Capabilities{Tools: &ToolsCapability{ListChanged: false}},
		}, nil

	case "initialized":
		return map[string]string{}, nil

	case "tools/list":
		return map[string]interface{}{"tools": tools}, nil

	case "tools/call":
		var p CallToolParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: -32602, Message: "Invalid params", Data: err.Error()}
		}
		return callTool(p)

	default:
		return nil, &RPCError{Code: -32601, Message: fmt.Sprintf("Method not found: %s", method)}
	}
}

func callTool(p CallToolParams) (*CallToolResult, *RPCError) {
	switch p.Name {
	case "browser_navigate":
		url, _ := p.Arguments["url"].(string)
		html, err := bc.Navigate(url)
		if err != nil {
			return &CallToolResult{IsError: true, Content: []ContentItem{{Type: "text", Text: err.Error()}}}, nil
		}
		return &CallToolResult{Content: []ContentItem{{Type: "text", Text: truncate(html, 50000)}}}, nil

	case "browser_screenshot":
		sel, _ := p.Arguments["selector"].(string)
		png, err := bc.Screenshot(sel)
		if err != nil {
			return &CallToolResult{IsError: true, Content: []ContentItem{{Type: "text", Text: err.Error()}}}, nil
		}
		return &CallToolResult{Content: []ContentItem{
			{Type: "image", Data: png, MimeType: "image/png"},
			{Type: "text", Text: fmt.Sprintf("Screenshot captured (%s bytes base64)", humanSize(len(png)))},
		}}, nil

	case "browser_click":
		sel, _ := p.Arguments["selector"].(string)
		err := bc.Click(sel)
		if err != nil {
			return &CallToolResult{IsError: true, Content: []ContentItem{{Type: "text", Text: err.Error()}}}, nil
		}
		return &CallToolResult{Content: []ContentItem{{Type: "text", Text: fmt.Sprintf("Clicked: %s", sel)}}}, nil

	case "browser_type":
		sel, _ := p.Arguments["selector"].(string)
		text, _ := p.Arguments["text"].(string)
		err := bc.Type(sel, text)
		if err != nil {
			return &CallToolResult{IsError: true, Content: []ContentItem{{Type: "text", Text: err.Error()}}}, nil
		}
		return &CallToolResult{Content: []ContentItem{{Type: "text", Text: fmt.Sprintf("Typed %q into %s", text, sel)}}}, nil

	case "browser_get_html":
		sel, _ := p.Arguments["selector"].(string)
		html, err := bc.GetHTML(sel)
		if err != nil {
			return &CallToolResult{IsError: true, Content: []ContentItem{{Type: "text", Text: err.Error()}}}, nil
		}
		return &CallToolResult{Content: []ContentItem{{Type: "text", Text: truncate(html, 50000)}}}, nil

	case "browser_get_styles":
		sel, _ := p.Arguments["selector"].(string)
		styles, err := bc.GetComputedStyles(sel)
		if err != nil {
			return &CallToolResult{IsError: true, Content: []ContentItem{{Type: "text", Text: err.Error()}}}, nil
		}
		var sb strings.Builder
		for _, s := range styles {
			sb.WriteString(fmt.Sprintf("%s: %s\n", s.Property, s.Value))
		}
		return &CallToolResult{Content: []ContentItem{{Type: "text", Text: sb.String()}}}, nil

	case "browser_evaluate":
		js, _ := p.Arguments["js"].(string)
		result, err := bc.Evaluate(js)
		if err != nil {
			return &CallToolResult{IsError: true, Content: []ContentItem{{Type: "text", Text: err.Error()}}}, nil
		}
		return &CallToolResult{Content: []ContentItem{{Type: "text", Text: result}}}, nil

	default:
		return nil, &RPCError{Code: -32601, Message: fmt.Sprintf("Tool not found: %s", p.Name)}
	}
}

func respond(id interface{}, result interface{}, rpcErr *RPCError) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
		Error:   rpcErr,
	}
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
