// Package browser implements a Browser MCP server for OVAV agents.
//
// Exposes browser automation tools via MCP JSON-RPC 2.0 over stdio.
// Supports both headless mode and remote connection to user's Chrome.
//
// Transport: stdio (default) or HTTP (--http :9222)
package browser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const browserEvaluateEnv = "OVAV_BROWSER_EVALUATE"

// MCPRequest represents a JSON-RPC 2.0 request.
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// MCPResponse represents a JSON-RPC 2.0 response.
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents a JSON-RPC 2.0 error.
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCPServer wraps a Controller and exposes MCP tools.
type MCPServer struct {
	ctrl        *Controller
	urlFirewall *URLFirewall
}

// NewMCPServer creates a new MCP server.
func NewMCPServer(ctrl *Controller) *MCPServer {
	root, _ := os.Getwd()
	return &MCPServer{ctrl: ctrl, urlFirewall: NewURLFirewall(root)}
}

// ToolCount returns the number of registered tools.
func (s *MCPServer) ToolCount() int {
	return len(availableTools())
}

func availableTools() []map[string]interface{} {
	if os.Getenv(browserEvaluateEnv) == "1" {
		return tools
	}
	filtered := make([]map[string]interface{}, 0, len(tools)-1)
	for _, tool := range tools {
		if tool["name"] != "browser_evaluate" {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}

// tools lists all available MCP tools.
var tools = []map[string]interface{}{
	{
		"name":        "browser_connect",
		"description": "Connect to user's existing Chrome browser via CDP. Chrome must be running with --remote-debugging-port=9222. Uses the user's real cookies, sessions, and extensions.",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"endpoint": map[string]interface{}{
					"type":        "string",
					"description": "CDP endpoint URL (default: http://127.0.0.1:9222)",
				},
			},
		},
	},
	{
		"name":        "browser_start",
		"description": "Start a headless Chromium browser (for server-side automation, no user cookies)",
		"inputSchema": map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	},
	{
		"name":        "browser_navigate",
		"description": "Navigate to a URL and return page HTML",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "URL to navigate to",
				},
			},
			"required": []string{"url"},
		},
	},
	{
		"name":        "browser_screenshot",
		"description": "Capture a screenshot of the page or a specific element",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"selector": map[string]interface{}{
					"type":        "string",
					"description": "CSS selector (empty for full page)",
				},
			},
		},
	},
	{
		"name":        "browser_click",
		"description": "Click an element by CSS selector",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"selector": map[string]interface{}{
					"type":        "string",
					"description": "CSS selector",
				},
			},
			"required": []string{"selector"},
		},
	},
	{
		"name":        "browser_type",
		"description": "Type text into an input element",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"selector": map[string]interface{}{
					"type":        "string",
					"description": "CSS selector of input element",
				},
				"text": map[string]interface{}{
					"type":        "string",
					"description": "Text to type",
				},
			},
			"required": []string{"selector", "text"},
		},
	},
	{
		"name":        "browser_get_html",
		"description": "Get the HTML of the page or a specific element",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"selector": map[string]interface{}{
					"type":        "string",
					"description": "CSS selector (empty for full page)",
				},
			},
		},
	},
	{
		"name":        "browser_evaluate",
		"description": "Execute arbitrary JavaScript in the page and return the result",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"js": map[string]interface{}{
					"type":        "string",
					"description": "JavaScript code to execute",
				},
			},
			"required": []string{"js"},
		},
	},
	// === DOM MODIFICATION TOOLS ===
	{
		"name":        "browser_inject_css",
		"description": "Inject CSS styles into an element (live frontend editing)",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"selector": map[string]interface{}{
					"type":        "string",
					"description": "CSS selector of target element",
				},
				"css": map[string]interface{}{
					"type":        "string",
					"description": "CSS properties (e.g. 'background: red; font-size: 20px')",
				},
			},
			"required": []string{"selector", "css"},
		},
	},
	{
		"name":        "browser_set_html",
		"description": "Replace inner HTML of an element (live DOM editing)",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"selector": map[string]interface{}{
					"type":        "string",
					"description": "CSS selector of target element",
				},
				"html": map[string]interface{}{
					"type":        "string",
					"description": "New HTML content",
				},
			},
			"required": []string{"selector", "html"},
		},
	},
	{
		"name":        "browser_set_attribute",
		"description": "Set an attribute on an element (e.g. src, href, class, data-*)",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"selector": map[string]interface{}{
					"type":        "string",
					"description": "CSS selector",
				},
				"attribute": map[string]interface{}{
					"type":        "string",
					"description": "Attribute name",
				},
				"value": map[string]interface{}{
					"type":        "string",
					"description": "Attribute value",
				},
			},
			"required": []string{"selector", "attribute", "value"},
		},
	},
	{
		"name":        "browser_insert_html",
		"description": "Insert HTML at a position relative to an element",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"selector": map[string]interface{}{
					"type":        "string",
					"description": "CSS selector",
				},
				"position": map[string]interface{}{
					"type":        "string",
					"description": "Insert position: beforebegin, afterbegin, beforeend, afterend",
					"enum":        []string{"beforebegin", "afterbegin", "beforeend", "afterend"},
				},
				"html": map[string]interface{}{
					"type":        "string",
					"description": "HTML to insert",
				},
			},
			"required": []string{"selector", "position", "html"},
		},
	},
	{
		"name":        "browser_remove_element",
		"description": "Remove an element from the DOM",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"selector": map[string]interface{}{
					"type":        "string",
					"description": "CSS selector of element to remove",
				},
			},
			"required": []string{"selector"},
		},
	},
	{
		"name":        "browser_reload",
		"description": "Reload the current page",
		"inputSchema": map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	},
	{
		"name":        "browser_status",
		"description": "Get browser connection status",
		"inputSchema": map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	},
	// React-aware tools — use nativeInputValueSetter for controlled components
	{
		"name":        "browser_type_react",
		"description": "Type text into a React controlled input (uses nativeInputValueSetter to trigger React state updates)",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"selector": map[string]interface{}{
					"type":        "string",
					"description": "CSS selector of input element",
				},
				"text": map[string]interface{}{
					"type":        "string",
					"description": "Text to type",
				},
			},
			"required": []string{"selector", "text"},
		},
	},
	{
		"name":        "browser_click_react",
		"description": "Click an element using React-compatible method (dispatchEvent with bubbles)",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"selector": map[string]interface{}{
					"type":        "string",
					"description": "CSS selector",
				},
			},
			"required": []string{"selector"},
		},
	},
}

// Run starts the MCP server on stdio.
func (s *MCPServer) Run() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req MCPRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.sendError(nil, -32700, "Parse error")
			continue
		}

		resp := s.HandleRequest(req)
		if resp != nil {
			data, _ := json.Marshal(resp)
			fmt.Println(string(data))
		}
	}
}

// HandleRequest processes a single MCP JSON-RPC request and returns the response.
func (s *MCPServer) HandleRequest(req MCPRequest) *MCPResponse {
	switch req.Method {
	case "initialize":
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
				"serverInfo": map[string]interface{}{
					"name":    "ovav-browser",
					"version": "2.0.0",
				},
			},
		}

	case "notifications/initialized":
		return nil // notification, no response

	case "tools/list":
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"tools": availableTools(),
			},
		}

	case "tools/call":
		return s.handleToolCall(req)

	default:
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: fmt.Sprintf("Unknown method: %s", req.Method),
			},
		}
	}
}

func (s *MCPServer) handleToolCall(req MCPRequest) *MCPResponse {
	params, ok := req.Params.(map[string]interface{})
	if !ok {
		return s.sendError(req.ID, -32602, "Invalid params")
	}

	name, _ := params["name"].(string)
	args, _ := params["arguments"].(map[string]interface{})

	var result interface{}
	var err error

	switch name {
	case "browser_connect":
		endpoint, _ := args["endpoint"].(string)
		if endpoint == "" {
			endpoint = "http://127.0.0.1:9222"
		}
		if err = s.urlFirewall.Validate(endpoint); err != nil {
			break
		}
		err = s.ctrl.ConnectRemote(endpoint)

	case "browser_start":
		err = s.ctrl.Start()

	case "browser_navigate":
		target, _ := args["url"].(string)
		if err = s.urlFirewall.Validate(target); err != nil {
			break
		}
		result, err = s.ctrl.Navigate(target)
		if err == nil {
			var finalURL string
			finalURL, err = s.ctrl.GetURL()
			if err == nil {
				err = s.urlFirewall.Validate(finalURL)
			}
			if err != nil {
				_ = s.ctrl.Stop()
				err = fmt.Errorf("browser redirect target denied: %w", err)
			}
		}

	case "browser_screenshot":
		selector, _ := args["selector"].(string)
		result, err = s.ctrl.Screenshot(selector)

	case "browser_click":
		selector, _ := args["selector"].(string)
		err = s.ctrl.Click(selector)

	case "browser_type":
		selector, _ := args["selector"].(string)
		text, _ := args["text"].(string)
		err = s.ctrl.Type(selector, text)

	case "browser_get_html":
		selector, _ := args["selector"].(string)
		result, err = s.ctrl.GetHTML(selector)

	case "browser_evaluate":
		if os.Getenv(browserEvaluateEnv) != "1" {
			err = fmt.Errorf("browser_evaluate disabled; set %s=1 for an explicit capability grant", browserEvaluateEnv)
			break
		}
		js, _ := args["js"].(string)
		result, err = s.ctrl.Evaluate(js)

	case "browser_inject_css":
		selector, _ := args["selector"].(string)
		css, _ := args["css"].(string)
		err = s.ctrl.InjectCSS(selector, css)

	case "browser_set_html":
		selector, _ := args["selector"].(string)
		html, _ := args["html"].(string)
		err = s.ctrl.SetInnerHTML(selector, html)

	case "browser_set_attribute":
		selector, _ := args["selector"].(string)
		attr, _ := args["attribute"].(string)
		value, _ := args["value"].(string)
		err = s.ctrl.SetAttribute(selector, attr, value)

	case "browser_insert_html":
		selector, _ := args["selector"].(string)
		position, _ := args["position"].(string)
		html, _ := args["html"].(string)
		err = s.ctrl.InsertAdjacentHTML(selector, position, html)

	case "browser_remove_element":
		selector, _ := args["selector"].(string)
		err = s.ctrl.RemoveElement(selector)

	case "browser_reload":
		err = s.ctrl.Reload()

	case "browser_status":
		result = s.ctrl.Status()

	// React-aware tools
	case "browser_type_react":
		selector, _ := args["selector"].(string)
		text, _ := args["text"].(string)
		js := fmt.Sprintf(`(() => {
			var el = document.querySelector(%q);
			if (!el) return 'element not found';
			var setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set;
			setter.call(el, %q);
			el.dispatchEvent(new Event("input", {bubbles: true}));
			el.dispatchEvent(new Event("change", {bubbles: true}));
			return "ok: " + el.value;
		})()`, selector, text)
		result, err = s.ctrl.Evaluate(js)

	case "browser_click_react":
		selector, _ := args["selector"].(string)
		js := fmt.Sprintf(`(() => {
			var el = document.querySelector(%q);
			if (!el) return 'element not found';
			el.dispatchEvent(new MouseEvent("click", {bubbles: true, cancelable: true}));
			return "clicked: " + el.tagName;
		})()`, selector)
		result, err = s.ctrl.Evaluate(js)

	default:
		return s.sendError(req.ID, -32601, fmt.Sprintf("Unknown tool: %s", name))
	}

	if err != nil {
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"isError": true,
				"content": []map[string]interface{}{
					{"type": "text", "text": err.Error()},
				},
			},
		}
	}

	textResult := fmt.Sprintf("%v", result)
	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": textResult},
			},
		},
	}
}

func (s *MCPServer) sendError(id interface{}, code int, msg string) *MCPResponse {
	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: msg,
		},
	}
}
