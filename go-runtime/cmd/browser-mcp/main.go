// Command browser-mcp is the Browser MCP server for OVAV agents.
//
// Usage:
//
//	browser-mcp                     # stdio mode (for MCP clients)
//	browser-mcp --http :18922       # rejected: obsolete insecure HTTP mode
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
	"flag"
	"fmt"
	"os"

	"github.com/ovav/ovav/internal/browser"
)

func main() {
	httpAddr := flag.String("http", "", "HTTP daemon listen address (e.g. :18922) — persistent process")
	autoConnect := flag.Bool("connect", false, "Auto-connect to Chrome on CDP port")
	port := flag.Int("port", 9222, "Chrome CDP port")
	endpoint := flag.String("endpoint", "", "CDP endpoint URL (e.g. http://172.22.80.1:9222)")
	flag.Parse()
	if err := validateLegacyHTTPMode(*httpAddr); err != nil {
		fmt.Fprintf(os.Stderr, "browser-mcp: %v\n", err)
		return
	}

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
			fmt.Sprintf("http://127.0.0.1:%d", *port),
		)
		firewall := browser.NewURLFirewall(currentRoot())

		connected := false
		for _, ep := range endpoints {
			fmt.Fprintf(os.Stderr, "browser-mcp: trying %s ...\n", ep)
			if err := firewall.Validate(ep); err != nil {
				fmt.Fprintf(os.Stderr, "browser-mcp: endpoint denied: %v\n", err)
				continue
			}
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

	server.Run()
}

func currentRoot() string {
	root, err := os.Getwd()
	if err != nil {
		return "."
	}
	return root
}

func validateLegacyHTTPMode(addr string) error {
	if addr != "" {
		return fmt.Errorf("obsolete HTTP mode is disabled; use cmd/browser_server with loopback bind and bearer authentication")
	}
	return nil
}
