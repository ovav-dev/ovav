// Command memory-mcp is the OVAV Memory MCP server.
//
// This is a stdio-based MCP server that exposes OVAV's persistent memory
// as always-on MCP tools. Unlike session_greeting (which only loads memory
// once at session start), memory-mcp runs as a background server and can
// be queried at any time during an agent session.
//
// Usage:
//
//	go run ./cmd/memory-mcp/
//
// The server reads JSON-RPC requests from stdin and writes responses to stdout.
// It is designed to be registered as an MCP server in opencode.json or
// similar agent configuration.
//
// MCP tools exposed:
//
//	memory_recall           — Query memory cards
//	memory_store            — Store a new memory card
//	memory_stats            — Get memory statistics
//	memory_recent           — Get recent memory cards
//	memory_verify           — Verify card authenticity
//	memory_search_decisions — Search governance decisions
//	memory_search_errors    — Search error recovery knowledge
package main

import (
	"log"

	"github.com/ovav/ovav/internal/cli"
	"github.com/ovav/ovav/internal/memory"
)

func main() {
	repoRoot, err := cli.FindRepoRoot()
	if err != nil {
		log.Fatalf("memory-mcp: find repo root: %v", err)
	}

	am, err := memory.NewAgentMemory(repoRoot, true)
	if err != nil {
		log.Fatalf("memory-mcp: init agent memory: %v", err)
	}

	server := memory.NewMCPServer(am, repoRoot)
	server.Run()
}
