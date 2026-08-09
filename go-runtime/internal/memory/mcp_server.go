// Package memory provides the OVAV Agent Memory system v2.
//
// This file implements the MCP server interface for OVAV Memory,
// exposing memory tools as an always-on MCP server that can be queried
// at any time during an agent session — not just at session start.
//
// MCP tools exposed:
//   - memory_recall   Query memory cards by text, tags, or topic
//   - memory_store    Store a new memory card
//   - memory_stats    Get memory statistics (counts by agent, tag)
//   - memory_recent   Get N most recent memory cards
//   - memory_verify   Verify authenticity of memory cards
//   - memory_search_decisions  Search governance decisions
//   - memory_search_errors     Search error recovery cards
//
// Run as: memory-mcp (stdio-based MCP server)
package memory

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// ── MCP Server ────────────────────────────────────────────────────────────────

// MCPServer is a stdio-based MCP server that exposes OVAV Memory tools.
// It runs as a long-lived background process and can be queried at any
// time during an agent session — making memory truly autonomous and
// persistent, not just loaded once at session start.
type MCPServer struct {
	am   *AgentMemory
	root string
}

// NewMCPServer creates a new OVAV Memory MCP server.
func NewMCPServer(am *AgentMemory, root string) *MCPServer {
	return &MCPServer{am: am, root: root}
}

// ── MCP Tool Definitions ──────────────────────────────────────────────────────

// mcpTools defines the tools exposed by the OVAV Memory MCP server.
// These mirror the CLI commands but are callable via MCP JSON-RPC at
// any time during the session.
var mcpTools = []map[string]interface{}{
	{
		"name":        "memory_recall",
		"description": "Query OVAV SYSTEM memory. Use when user asks about past decisions, errors, rules, or knowledge that OVAV has stored. Searches topic, summary, operational rule, tags, and card ID. Returns matching cards with authenticity evidence.",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Text to search across topic, summary, rule, tags. E.g. 'governance decision', 'error recovery', 'agent memory'",
				},
				"tags": map[string]interface{}{
					"type":        "string",
					"description": "Comma-separated tags to filter by. E.g. 'governance,decision'",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Max cards to return (default: 10)",
				},
				"format": map[string]interface{}{
					"type":        "string",
					"description": "Output format: 'compact' (default, human-readable) or 'json' (machine-readable)",
				},
			},
		},
	},
	{
		"name":        "memory_store",
		"description": "Store a new memory card in OVAV SYSTEM persistent memory. Use to record decisions, error recoveries, governance rules, or important knowledge. The card is permanently stored and verified with SHA-256 evidence hash.",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"topic":    map[string]interface{}{"type": "string", "description": "Topic/subject of the memory (e.g. 'governance: push gate policy')"},
				"summary":  map[string]interface{}{"type": "string", "description": "Brief summary of what to remember"},
				"rule":     map[string]interface{}{"type": "string", "description": "Operational rule or guidance derived from this knowledge"},
				"agent":    map[string]interface{}{"type": "string", "description": "Agent ID creating this card (e.g. 'thavren', 'ovav-system')"},
				"tags":     map[string]interface{}{"type": "string", "description": "Comma-separated tags (e.g. 'governance,rule,push')"},
				"priority": map[string]interface{}{"type": "string", "description": "Priority: HIGH, NORMAL, LOW (default: NORMAL)"},
			},
			"required": []string{"topic", "summary"},
		},
	},
	{
		"name":        "memory_stats",
		"description": "Get OVAV SYSTEM memory statistics. Returns total/active card counts, breakdown by agent and by tag. Use to understand what knowledge OVAV has stored.",
		"inputSchema": map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	},
	{
		"name":        "memory_recent",
		"description": "Get the N most recent memory cards from OVAV SYSTEM. Use to see latest recorded knowledge, decisions, and system activity.",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Number of recent cards to return (default: 10, max: 50)",
				},
			},
		},
	},
	{
		"name":        "memory_verify",
		"description": "Verify the authenticity of OVAV memory cards using SHA-256 evidence hashes and git HEAD anchors. Detects tampered or corrupted cards. Returns verification report per card.",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"card_id": map[string]interface{}{
					"type":        "string",
					"description": "Specific card ID to verify. If omitted, verifies all recent cards.",
				},
			},
		},
	},
	{
		"name":        "memory_search_decisions",
		"description": "Search OVAV SYSTEM memory specifically for governance decisions. Use when user asks about policies, rules, or decisions that govern OVAV behavior.",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Decision topic to search for (e.g. 'push gate', 'permission', 'memory')",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Max results (default: 5)",
				},
			},
		},
	},
	{
		"name":        "memory_search_errors",
		"description": "Search OVAV SYSTEM memory for error recovery knowledge. Use when user encounters an error or wants to avoid repeating past failures.",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Error context or technology to search for (e.g. 'git push', 'permission denied')",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Max results (default: 5)",
				},
			},
		},
	},
}

// ── MCP Request/Response Types ────────────────────────────────────────────────

// mcpRequest represents a JSON-RPC 2.0 request.
type mcpRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// mcpResponse represents a JSON-RPC 2.0 response.
type mcpResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *mcpError   `json:"error,omitempty"`
}

// mcpError represents a JSON-RPC 2.0 error.
type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ── Server Lifecycle ─────────────────────────────────────────────────────────

// Run starts the OVAV Memory MCP server on stdio.
// It listens for JSON-RPC requests and responds until EOF.
func (s *MCPServer) Run() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req mcpRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.sendError(nil, -32700, "Parse error: "+err.Error())
			continue
		}

		resp := s.handleRequest(req)
		if resp != nil {
			data, _ := json.Marshal(resp)
			fmt.Println(string(data))
		}
	}
}

// handleRequest processes a single MCP JSON-RPC request.
func (s *MCPServer) handleRequest(req mcpRequest) *mcpResponse {
	switch req.Method {
	case "initialize":
		return &mcpResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
				"serverInfo": map[string]interface{}{
					"name":    "ovav-memory",
					"version": "2.0.0",
				},
			},
		}

	case "tools/list":
		return &mcpResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"tools": mcpTools,
			},
		}

	case "tools/call":
		return s.handleToolCall(req)

	default:
		return &mcpResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &mcpError{Code: -32601, Message: "Method not found: " + req.Method},
		}
	}
}

func (s *MCPServer) sendError(id interface{}, code int, msg string) {
	resp := mcpResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &mcpError{Code: code, Message: msg},
	}
	data, _ := json.Marshal(resp)
	fmt.Println(string(data))
}

// ── Tool Call Handler ─────────────────────────────────────────────────────────

func (s *MCPServer) handleToolCall(req mcpRequest) *mcpResponse {
	var params map[string]interface{}
	if req.Params != nil {
		if m, ok := req.Params.(map[string]interface{}); ok {
			params = m
		}
	}
	if params == nil {
		params = make(map[string]interface{})
	}

	toolName, _ := params["name"].(string)

	var result interface{}
	var errMsg string

	switch toolName {
	case "memory_recall":
		result = s.mcpRecall(params)
	case "memory_store":
		result, errMsg = s.mcpStore(params)
	case "memory_stats":
		result = s.mcpStats()
	case "memory_recent":
		result = s.mcpRecent(params)
	case "memory_verify":
		result = s.mcpVerify(params)
	case "memory_search_decisions":
		result = s.mcpSearchDecisions(params)
	case "memory_search_errors":
		result = s.mcpSearchErrors(params)
	default:
		return &mcpResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &mcpError{Code: -32602, Message: "Unknown tool: " + toolName},
		}
	}

	if errMsg != "" {
		return &mcpResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &mcpError{Code: -32603, Message: errMsg},
		}
	}

	return &mcpResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": result,
				},
			},
		},
	}
}

// ── Tool Implementations ──────────────────────────────────────────────────────

func (s *MCPServer) mcpRecall(params map[string]interface{}) string {
	// MCP tools/call passes arguments under "arguments" key
	args, _ := params["arguments"].(map[string]interface{})
	if args == nil {
		args = params
	}
	query, _ := args["query"].(string)
	tagsStr, _ := args["tags"].(string)
	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	var tags []string
	if tagsStr != "" {
		for _, t := range strings.Split(tagsStr, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	opts := RecallOptions{
		Query: query,
		Tags:  tags,
		Limit: limit,
	}
	result := s.am.Recall(opts)

	var lines []string
	if len(result.Cards) == 0 {
		return "No memory cards found matching your query. You may need to store this knowledge first with memory_store."
	}

	lines = append(lines, fmt.Sprintf("OVAV Memory — %d card(s) found:", len(result.Cards)))
	for _, c := range result.Cards {
		lines = append(lines, fmt.Sprintf("\n[%s] %s | %s", c.Priority, c.Topic, c.Summary))
		if c.OperationalRule != "" {
			lines = append(lines, fmt.Sprintf("   Rule: %s", c.OperationalRule))
		}
		if c.Commit != "" {
			lines = append(lines, fmt.Sprintf("   Commit: %s", c.Commit[:7]))
		}
	}

	return strings.Join(lines, "")
}

func (s *MCPServer) mcpStore(params map[string]interface{}) (string, string) {
	args, _ := params["arguments"].(map[string]interface{})
	if args == nil {
		args = params
	}
	topic, _ := args["topic"].(string)
	summary, _ := args["summary"].(string)
	rule, _ := args["rule"].(string)
	agentID, _ := args["agent"].(string)
	tagsStr, _ := args["tags"].(string)
	priority, _ := args["priority"].(string)

	if topic == "" || summary == "" {
		return "", "topic and summary are required"
	}

	var tags []string
	if tagsStr != "" {
		for _, t := range strings.Split(tagsStr, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	card := Card{
		Topic:           topic,
		Summary:         summary,
		OperationalRule: rule,
		Tags:            tags,
		Priority:        priority,
	}

	if agentID == "" {
		agentID = "ovav-system"
	}
	if priority == "" {
		card.Priority = "NORMAL"
	}

	commit := guessGitHead(s.root)
	record, err := s.am.Store(card, StoreOptions{
		AgentID: agentID,
		Commit:  commit,
	})
	if err != nil {
		return "", fmt.Sprintf("store failed: %v", err)
	}

	// Verify the stored card
	v := s.am.Verify([]Card{record.Card})
	auth := "✅"
	if v.Verified < 1 {
		auth = "⚠️"
	}

	return fmt.Sprintf("%s Memory card stored and verified: %s\n  Topic: %s\n  ID: %s", auth, summary, topic, record.Card.ID), ""
}

func (s *MCPServer) mcpStats() string {
	total, active, byAgent, byTag := s.am.Stats()

	var lines []string
	lines = append(lines, "OVAV Memory Statistics")
	lines = append(lines, "═══════════════════════════════════")
	lines = append(lines, fmt.Sprintf("  Total cards:   %d", total["total"]))
	lines = append(lines, fmt.Sprintf("  Active cards:  %d", active["active"]))

	if len(byAgent) > 0 {
		lines = append(lines, "\n  By agent:")
		for agent, count := range byAgent {
			lines = append(lines, fmt.Sprintf("    %s: %d", agent, count))
		}
	}

	if len(byTag) > 0 {
		lines = append(lines, "\n  By tag:")
		// Sort tags by count descending
		type tagCount struct {
			tag   string
			count int
		}
		var sorted []tagCount
		for t, c := range byTag {
			sorted = append(sorted, tagCount{t, c})
		}
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].count > sorted[j].count
		})
		for _, tc := range sorted {
			lines = append(lines, fmt.Sprintf("    %s: %d", tc.tag, tc.count))
		}
	}

	return strings.Join(lines, "\n")
}

func (s *MCPServer) mcpRecent(params map[string]interface{}) string {
	args, _ := params["arguments"].(map[string]interface{})
	if args == nil {
		args = params
	}
	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}
	if limit > 50 {
		limit = 50
	}

	cards := s.am.Recent(limit)

	var lines []string
	lines = append(lines, fmt.Sprintf("OVAV Memory — %d most recent card(s):", len(cards)))
	for _, c := range cards {
		lines = append(lines, fmt.Sprintf("\n[%s] %s | %s", c.Priority, c.Topic, c.Summary))
		if c.OperationalRule != "" {
			rule := c.OperationalRule
			if len(rule) > 80 {
				rule = rule[:80] + "..."
			}
			lines = append(lines, fmt.Sprintf("   → %s", rule))
		}
	}

	return strings.Join(lines, "")
}

func (s *MCPServer) mcpVerify(params map[string]interface{}) string {
	args, _ := params["arguments"].(map[string]interface{})
	if args == nil {
		args = params
	}
	cardID, _ := args["card_id"].(string)

	if cardID != "" {
		cards := s.am.Recall(RecallOptions{Limit: 100})
		var found Card
		for _, c := range cards.Cards {
			if c.ID == cardID {
				found = c
				break
			}
		}
		if found.ID == "" {
			return fmt.Sprintf("Card %s not found", cardID)
		}
		v := s.am.Verify([]Card{found})
		status := "✅ AUTHENTIC"
		if v.Verified == 0 {
			status = "❌ UNAUTHENTIC"
		}
		summary := fmt.Sprintf("%s | %s", status, found.Summary)
		if found.OperationalRule != "" {
			summary += fmt.Sprintf("\n   Rule: %s", found.OperationalRule)
		}
		summary += fmt.Sprintf("\n   Report: verified=%d stale=%d no_source=%d conflicts=%d",
			v.Verified, v.Stale, v.NoSource, v.Conflicts)
		return summary
	}

	// Verify all recent cards
	cards := s.am.Recent(20)
	v := s.am.Verify(cards)

	var lines []string
	lines = append(lines, fmt.Sprintf("OVAV Memory Verification Report — %d card(s) checked", len(cards)))
	lines = append(lines, "═══════════════════════════════════")
	lines = append(lines, fmt.Sprintf("✅ Verified (valid hash): %d", v.Verified))
	lines = append(lines, fmt.Sprintf("⚠️  Stale (git commit changed): %d", v.Stale))
	lines = append(lines, fmt.Sprintf("⚠️  No source chain: %d", v.NoSource))
	lines = append(lines, fmt.Sprintf("🔴 Conflicts detected: %d", v.Conflicts))

	if len(v.Issues) > 0 {
		lines = append(lines, "\nIssues:")
		for _, issue := range v.Issues {
			lines = append(lines, fmt.Sprintf("  • %s", issue))
		}
	}

	return strings.Join(lines, "\n")
}

func (s *MCPServer) mcpSearchDecisions(params map[string]interface{}) string {
	args, _ := params["arguments"].(map[string]interface{})
	if args == nil {
		args = params
	}
	query, _ := args["query"].(string)
	limit := 5
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	result := s.am.Recall(RecallOptions{
		Query: query,
		Tags:  []string{"governance", "decision", "rule"},
		Limit: limit,
	})

	var lines []string
	if len(result.Cards) == 0 {
		return "No governance decisions found. Store a decision first with memory_store."
	}

	lines = append(lines, "OVAV Governance Decisions:")
	for _, c := range result.Cards {
		lines = append(lines, fmt.Sprintf("\n📋 %s | %s", c.Topic, c.Summary))
		if c.OperationalRule != "" {
			lines = append(lines, fmt.Sprintf("   Rule: %s", c.OperationalRule))
		}
	}

	return strings.Join(lines, "")
}

func (s *MCPServer) mcpSearchErrors(params map[string]interface{}) string {
	args, _ := params["arguments"].(map[string]interface{})
	if args == nil {
		args = params
	}
	query, _ := args["query"].(string)
	limit := 5
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	result := s.am.Recall(RecallOptions{
		Query: query,
		Tags:  []string{"error", "error_recovery"},
		Limit: limit,
	})

	var lines []string
	if len(result.Cards) == 0 {
		return "No error recovery cards found matching your query."
	}

	lines = append(lines, "OVAV Error Recovery Knowledge:")
	for _, c := range result.Cards {
		lines = append(lines, fmt.Sprintf("\n🔴 %s | %s", c.Topic, c.Summary))
		if c.OperationalRule != "" {
			lines = append(lines, fmt.Sprintf("   Recovery: %s", c.OperationalRule))
		}
	}

	return strings.Join(lines, "")
}
