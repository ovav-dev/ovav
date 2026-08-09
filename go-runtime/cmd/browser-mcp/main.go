// Command browser-mcp is the Browser MCP server for OVAV agents.
//
// Usage:
//
//	browser-mcp                     # stdio mode (for MCP clients)
//	browser-mcp --http :18922       # HTTP daemon mode (persistent, no restart per call)
//	browser-mcp --connect           # Auto-connect to Chrome on CDP port
//	browser-mcp --endpoint http://HOST:9222  # Connect to specific endpoint
//
// Environment:
//
//	BROWSER_CDP_PORT=9222           # Chrome debugging port (default 9222)
//	BROWSER_CDP_ENDPOINT=http://HOST:9222  # Override endpoint
//	BROWSER_CONNECT=true            # Auto-connect to existing Chrome
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/ovav/ovav/internal/browser"
)

func main() {
	httpAddr := flag.String("http", "", "HTTP daemon listen address (e.g. :18922) — persistent process")
	autoConnect := flag.Bool("connect", false, "Auto-connect to Chrome on CDP port")
	port := flag.Int("port", 9222, "Chrome CDP port")
	endpoint := flag.String("endpoint", "", "CDP endpoint URL (e.g. http://172.22.80.1:9222)")
	flag.Parse()

	// Override from env
	if p := os.Getenv("BROWSER_CDP_PORT"); p != "" {
		fmt.Sscanf(p, "%d", port)
	}
	if e := os.Getenv("BROWSER_CDP_ENDPOINT"); e != "" {
		*endpoint = e
	}
	if os.Getenv("BROWSER_CONNECT") == "true" {
		*autoConnect = true
	}

	ctrl := browser.New(browser.WithPort(*port))

	// Auto-connect to existing Chrome
	if *autoConnect {
		endpoints := []string{}
		if *endpoint != "" {
			endpoints = append(endpoints, *endpoint)
		}
		endpoints = append(endpoints,
			fmt.Sprintf("http://172.22.80.1:%d", *port),
			fmt.Sprintf("http://127.0.0.1:%d", *port),
		)

		connected := false
		for _, ep := range endpoints {
			fmt.Fprintf(os.Stderr, "browser-mcp: trying %s ...\n", ep)
			if err := ctrl.ConnectRemote(ep); err == nil {
				fmt.Fprintf(os.Stderr, "browser-mcp: connected to Chrome at %s\n", ep)
				connected = true
				break
			}
		}

		if !connected {
			fmt.Fprintf(os.Stderr, "browser-mcp: Chrome not found on any endpoint\n")
			fmt.Fprintf(os.Stderr, "  Start Chrome with: chrome --remote-debugging-port=%d\n", *port)
			fmt.Fprintf(os.Stderr, "  Falling back to headless mode\n")
			if err := ctrl.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "browser-mcp: headless start failed: %v\n", err)
				os.Exit(1)
			}
		}
	}

	server := browser.NewMCPServer(ctrl)

	if *httpAddr != "" {
		// HTTP DAEMON MODE — persistent process, survives across MCP calls
		// No restart per call = no new tabs = stable connection
		http.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "POST required", http.StatusMethodNotAllowed)
				return
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "read error", http.StatusBadRequest)
				return
			}
			defer r.Body.Close()

			// Handle newline-delimited JSON (multiple requests in one body)
			lines := strings.Split(strings.TrimSpace(string(body)), "\n")
			var lastResponse string
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				var req browser.MCPRequest
				if err := json.Unmarshal([]byte(line), &req); err != nil {
					continue
				}
				resp := server.HandleRequest(req)
				if resp != nil {
					data, _ := json.Marshal(resp)
					lastResponse = string(data)
				}
			}

			w.Header().Set("Content-Type", "application/json")
			if lastResponse != "" {
				fmt.Fprint(w, lastResponse)
			} else {
				fmt.Fprint(w, `{"jsonrpc":"2.0","result":{}}`)
			}
		})

		// Health check endpoint
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			status := ctrl.Status()
			data, _ := json.Marshal(status)
			fmt.Fprint(w, string(data))
		})

		// Bind explicitly to TCP4 to avoid IPv6/IPv4 dual-stack "address in use" on Windows
		addr := *httpAddr
		if !strings.Contains(addr, "://") {
			if !strings.HasPrefix(addr, ":") {
				addr = ":" + addr
			}
		}
		listener, err := net.Listen("tcp4", addr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "browser-mcp: listen failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "browser-mcp: daemon listening on %s (tcp4)\n", addr)
		if err := http.Serve(listener, nil); err != nil {
			fmt.Fprintf(os.Stderr, "browser-mcp: http serve failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		// STDIO MODE — traditional MCP, process per session
		server.Run()
	}
}
