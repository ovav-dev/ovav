// OVAV cPanel v5.2 — Backend API server.
//
// Reemplaza tools/cpanel/server.py (Python) con Go stdlib.
// Puerto 5858, contrato API REST idéntico.
// Stack: Go 1.22+, solo stdlib. Sin dependencias externas.
//
// v5.2: GOV-007 broadcast hub + product update endpoints
//
// Usage:
//
//	go run ./cmd/cpanel/              # Start on :5858 (localhost)
//	go run ./cmd/cpanel/ --port 9000  # Custom port
//	PORT=5858 ./cpanel                # Production mode (0.0.0.0:$PORT)
//	go build -o cpanel ./cmd/cpanel/  # Build binary

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const defaultPort = 5858
const shutdownTimeout = 10 * time.Second

func main() {
	port := flag.Int("port", 0, "HTTP server port (0 = use $PORT or default)")
	flag.Parse()

	// Resolve port: flag > env > default
	if *port == 0 {
		if envPort := os.Getenv("PORT"); envPort != "" {
			if p, err := strconv.Atoi(envPort); err == nil {
				*port = p
			}
		}
	}
	if *port == 0 {
		*port = defaultPort
	}

	// Bind address: $OVAV_LISTEN_ADDR controls binding.
	// Container/tunnel mode: OVAV_LISTEN_ADDR=127.0.0.1 (default — no public exposure)
	// Direct mode: OVAV_LISTEN_ADDR=0.0.0.0 (local dev only)
	host := "127.0.0.1"
	if addr := os.Getenv("OVAV_LISTEN_ADDR"); addr != "" {
		host = addr
	}

	printBanner(host, *port)

	// GOV-007: Start the SSE broadcast hub before registering routes
	hub.Start()

	// P2-B: Initialize audit webhook from Fly.io secret if set
	if webhookURL := os.Getenv("AUDIT_WEBHOOK_URL"); webhookURL != "" {
		initWebhook(webhookURL)
		fmt.Println("  Audit webhook: enabled")
	} else {
		fmt.Println("  Audit webhook: disabled (set AUDIT_WEBHOOK_URL to enable)")
	}

	// P2-A: Emergency bypass code warning (never log the code itself)
	if bypass := os.Getenv("EMERGENCY_BYPASS_CODE"); bypass != "" {
		fmt.Println("  Emergency bypass: ENABLED — keep code secret")
	}

	// CRITICAL FIX (C2): authMiddleware wraps routerMux to enforce JWT on all
	// protected endpoints. Previously: routerMux had no auth check.
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", host, *port),
		Handler:      tracingMiddleware(securityHeadersMiddleware(corsMiddleware(authMiddleware(http.HandlerFunc(routerMux))))),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	registerRoutes()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		protocol := "http"
		displayHost := host
		if host == "0.0.0.0" {
			displayHost = "localhost"
		}
		fmt.Printf("  Listening on %s://%s:%d\n", protocol, displayHost, *port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "cPanel error: %v\n", err)
			os.Exit(1)
		}
	}()

	<-stop
	fmt.Println("\n  cPanel shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "  Shutdown error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("  cPanel stopped.")
}

func printBanner(host string, port int) {
	displayHost := host
	if host == "0.0.0.0" {
		displayHost = "0.0.0.0 (production)"
	}
	fmt.Printf(`
╔══════════════════════════════════════════════════════════╗
║       OVAV cPanel v5.1 — Enterprise Control Panel        ║
╠══════════════════════════════════════════════════════════╣
║  URL:      http://localhost:%-5d                        ║
║  API:      http://localhost:%-5d/api/v1/status          ║
║  Bind:     %-46s║
╠══════════════════════════════════════════════════════════╣
║  Runtime:  Go (stdlib-only, zero dependencies)           ║
║  Repo:     %-46s║
║  Version:  %-46s║
║  Quit:     Ctrl+C                                       ║
╚══════════════════════════════════════════════════════════╝
`, port, port, displayHost, shorten(RepoRoot, 46), Version)
}

func shorten(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return "..." + s[len(s)-max+3:]
}
