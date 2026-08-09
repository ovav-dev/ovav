// OVAV cPanel v5.1 — HTTP router with JWT auth middleware.
//
// CRITICAL FIX (C2): All API endpoints now require JWT Bearer token
// except public paths (/health, /api/v1/auth/*, static assets, SPA ingress).
// Previously: routerMux had NO auth check — any caller could access any endpoint.
//
// Stdlib net/http. Exact + prefix route matching.
// Server lifecycle managed by main.go.

package main

import (
	"fmt"
	"net/http"
	"strings"
)

// ── Route table ───────────────────────────────────────────────────────────────

type route struct {
	method  string
	prefix  string
	handler http.HandlerFunc
}

var routes []route

func addRoute(method, path string, handler http.HandlerFunc) {
	routes = append(routes, route{method: method, prefix: path, handler: handler})
}

func addPrefix(method, prefix string, handler http.HandlerFunc) {
	routes = append(routes, route{method: method, prefix: prefix, handler: handler})
}

// ── Public paths (no JWT required) ────────────────────────────────────────────

var publicPaths = []string{
	"/health",
	"/api/v1/health",
	"/api/v1/healthz", // deep health — bypasses Cloudflare Access
	"/api/v1/auth/",
	"/api/v1/auth/login",
	"/api/v1/auth/session",
	"/api/v1/auth/config",
	"/api/v1/auth/cli-verify",
	"/api/v1/events", // SSE — token in query param
	"/assets/",
	"/css/",
	"/login", // OVAV admin login page (GOV-010)
	// Vault Sync (Phase 6.2) — auth endpoint is public, blob ops require JWT
	"/api/v1/vault/health",
	"/api/v1/vault/auth",
}

// publicExactPaths are product endpoints accessible without auth (GOV-007: Product polling).
var publicExactPaths = map[string]bool{
	"/api/v1/product/version": true,
	"/api/v1/product/update":  true,
	"/api/v1/vault/health":    true, // public health check
	"/api/v1/vault/auth":      true, // public auth endpoint
	"/login/verify":           true, // GOV-010: TOTP verification page (pre-auth)
}

func isPublicPath(path string) bool {
	// Check exact matches first
	if publicExactPaths[path] {
		return true
	}
	for _, p := range publicPaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// ── Auth middleware ────────────────────────────────────────────────────────────

// authMiddleware wraps the router with JWT verification for protected endpoints.
// Public paths bypass auth. All other paths require a valid Bearer token.
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// OPTIONS preflight — always allowed (CORS handles auth separately)
		if r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// P0-C: Rate limit on all paths — per IP, per path category
		ip := helperIP(r)
		if allowed, retryAfter := checkPathRateLimit(ip, r.URL.Path); !allowed {
			w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
			sendError(w, fmt.Sprintf("rate limit exceeded — try again in %ds", retryAfter), http.StatusTooManyRequests)
			return
		}

		// Public paths — no auth required
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// SSE events — allow token in query param (?token=...)
		if strings.HasPrefix(r.URL.Path, "/api/v1/events") {
			next.ServeHTTP(w, r)
			return
		}

		// Protected paths — require Bearer token or cookie
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")

		// Fall back to cookie if no Authorization header
		if token == "" || token == authHeader {
			if cookie, err := r.Cookie("ovav_token"); err == nil {
				token = cookie.Value
			}
		}

		// GOV-010: root shows login if unauthenticated, dashboard if valid token
		if r.URL.Path == "/" {
			if token != "" {
				if _, err := verifyJWT(token); err == nil {
					next.ServeHTTP(w, r) // valid → dashboard
					return
				}
			}
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		if token == "" {
			AuditAuth(r.Context(), "anonymous", "", helperIP(r),
				"auth.missing_token", r.URL.Path, "denied", http.StatusUnauthorized, "")
			sendError(w, "authentication required — provide Bearer token", http.StatusUnauthorized)
			return
		}

		claims, err := verifyJWT(token)
		if err != nil {
			AuditAuth(r.Context(), "anonymous", "", helperIP(r),
				"auth.invalid_token", r.URL.Path, "denied", http.StatusUnauthorized, err.Error())
			sendError(w, fmt.Sprintf("invalid token: %v", err), http.StatusUnauthorized)
			return
		}

		// Audit successful authenticated access
		AuditAuth(r.Context(), claims.UserID, claims.Email, helperIP(r),
			"auth.success", r.URL.Path, "ok", http.StatusOK, "")

		// Token valid — inject claims into context for role-based access
		_ = claims // claims available for future role-based access control

		next.ServeHTTP(w, r)
	})
}

// ── Router ────────────────────────────────────────────────────────────────────

func routerMux(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	method := r.Method

	// Exact match first
	for _, rt := range routes {
		if rt.method == method && rt.prefix == path && !strings.HasSuffix(rt.prefix, "/") {
			rt.handler(w, r)
			return
		}
	}

	// Prefix match (routes ending with "/")
	for _, rt := range routes {
		if rt.method == method && strings.HasSuffix(rt.prefix, "/") && strings.HasPrefix(path, rt.prefix) {
			rt.handler(w, r)
			return
		}
	}

	sendError(w, fmt.Sprintf("not found: %s %s", method, path), http.StatusNotFound)
}
