// OVAV cPanel v5.0 — Shared types and helpers.
//
// Package main, stdlib-only. No external dependencies.
// Repo root detection, JSON response helpers, CORS, MIME types.

package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ── Build metadata (injected at compile time via -ldflags) ────────────────────

var (
	Version   = "3.4.0"
	Build     = "cPanel v2.1 — Go Runtime"
	GitBranch = "unknown"
	GitSHA    = "unknown"
	BuildTime = "unknown"
)

// ── Repo root ─────────────────────────────────────────────────────────────────

// RepoRoot is the absolute path to the OVAV repository root.
// Detected at startup by walking up from the binary's directory.
var RepoRoot string

func init() {
	RepoRoot = findRepoRoot()
}

func findRepoRoot() string {
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		for range 10 {
			if _, err := os.Stat(filepath.Join(dir, ".ovav")); err == nil {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	cwd, _ := os.Getwd()
	return cwd
}

// ── MIME types for static file serving ────────────────────────────────────────

var mimeTypes = map[string]string{
	".js":    "application/javascript",
	".css":   "text/css",
	".html":  "text/html; charset=utf-8",
	".json":  "application/json",
	".png":   "image/png",
	".svg":   "image/svg+xml",
	".ico":   "image/x-icon",
	".woff2": "font/woff2",
	".map":   "application/json",
}

// ── JSON response helpers ─────────────────────────────────────────────────────

func sendJSON(w http.ResponseWriter, data interface{}, code int) {
	body, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"json marshal failed"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	w.WriteHeader(code)
	w.Write(body)
}

func sendOK(w http.ResponseWriter, data interface{}) {
	sendJSON(w, data, http.StatusOK)
}

func sendError(w http.ResponseWriter, msg string, code int) {
	sendJSON(w, map[string]string{"error": msg}, code)
}

// ── CORS — Production hardening: restricted origins, not wildcard ──────────────

// allowedOrigins lists origins permitted to access the cPanel API.
// In production, this should be set via CPANEL_ALLOWED_ORIGINS env var
// (comma-separated). Default: localhost + ovav.dev domains.
var allowedOrigins map[string]bool

func init() {
	allowedOrigins = map[string]bool{
		"http://localhost:3000":     true,
		"http://localhost:5173":     true,
		"https://d678beea.ovav.dev": true,
		"https://docs.ovav.dev":     true,
		"https://status.ovav.dev":   true,
		"https://ovav.dev":          true,
	}
	if extra := os.Getenv("CPANEL_ALLOWED_ORIGINS"); extra != "" {
		for _, origin := range strings.Split(extra, ",") {
			origin = strings.TrimSpace(origin)
			if origin != "" {
				allowedOrigins[origin] = true
			}
		}
	}
}

// isOriginAllowed checks if the request Origin header matches an allowed origin.
func isOriginAllowed(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // same-origin requests, browser extensions, curl
	}
	return allowedOrigins[origin]
}

// setCORSHeader sets the appropriate Access-Control-Allow-Origin header.
// Uses the request's Origin if allowed; falls back to the first configured origin.
func setCORSHeader(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin != "" && allowedOrigins[origin] {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	} else {
		w.Header().Set("Access-Control-Allow-Origin", "https://d678beea.ovav.dev")
	}
}
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setCORSHeader(w, r)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ── Security headers — production hardening ─────────────────────────────────────

// securityHeadersMiddleware sets OWASP-recommended security headers on every response.
// Admin CPanel should NEVER be embeddable in iframes — X-Frame-Options: DENY is critical.
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME type sniffing — browsers won't execute non-script as script
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Clickjacking protection — admin panel must never be iframeable
		w.Header().Set("X-Frame-Options", "DENY")
		// HTTPS enforcement — Fly.io handles the redirect, this is a belt-and-suspenders
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		// No referrer leak to external sites on navigation away from cPanel
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// Disable browser features the admin panel doesn't need
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), cross-origin-isolated=()")
		// Content Security Policy — strict, no inline, no external scripts
		w.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		// Suppress server version fingerprinting
		w.Header().Set("Server", "OVAV-cPanel/2.3")
		next.ServeHTTP(w, r)
	})
}

// ── Request tracing (P0-B) ──────────────────────────────────────────────────────

// reqContextKey is a type-safe key for request-scoped values.
type reqContextKey string

const requestIDKey reqContextKey = "requestID"

// tracingMiddleware assigns or propagates a unique request ID on every request.
// Priority: 1) CF-Ray (Cloudflare's existing ID), 2) X-Request-ID header, 3) generated.
// The ID is injected into the request context and written to response headers.
func tracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("CF-Ray")
		if reqID == "" {
			reqID = r.Header.Get("X-Request-ID")
		}
		if reqID == "" {
			b := make([]byte, 8)
			rand.Read(b)
			reqID = hex.EncodeToString(b)
		}
		w.Header().Set("X-Request-ID", reqID)
		ctx := context.WithValue(r.Context(), requestIDKey, reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ReqIDFrom returns the request ID from context, or empty string if none.
func ReqIDFrom(ctx context.Context) string {
	if v := ctx.Value(requestIDKey); v != nil {
		return v.(string)
	}
	return ""
}

// ── Structured audit logger (P0-B) ─────────────────────────────────────────────

// AuditEvent es el formato estructurado para cada evento de auditoría.
// Written to .ovav/security/audit_log.jsonl (JSON Lines — append-only, no rotation risk).
type AuditEvent struct {
	Timestamp string `json:"timestamp"`
	ReqID     string `json:"req_id"`
	UserID    string `json:"user_id"` // "anonymous" si no hay auth
	Email     string `json:"email,omitempty"`
	IP        string `json:"ip"`         // CF-IP-Address
	Country   string `json:"cf_country"` // CF-IPCountry
	RayID     string `json:"cf_ray"`     // Cloudflare trace ID
	Action    string `json:"action"`     // e.g. "vault.secret.read", "auth.login", "auth.fail"
	Resource  string `json:"resource"`   // e.g. "/api/v1/vault/secrets/prod-key"
	Method    string `json:"method"`
	Status    string `json:"status"` // "ok", "denied", "error"
	Code      int    `json:"code"`   // HTTP status code
	UserAgent string `json:"user_agent"`
	Detail    string `json:"detail,omitempty"`
}

var (
	auditLogger     *auditWriter
	auditLoggerOnce sync.Once
)

func getAuditLogger() *auditWriter {
	auditLoggerOnce.Do(func() {
		auditLogger = &auditWriter{path: ".ovav/security/audit_log.jsonl"}
	})
	return auditLogger
}

type auditWriter struct {
	path string
	mu   sync.Mutex
}

func (w *auditWriter) log(event AuditEvent) {
	if event.Timestamp == "" {
		event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	f, err := os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	f.Write(append(data, '\n'))
	f.Close()
}

// ── Webhook dispatcher for audit events (P2-B) ─────────────────────────────────

// webhookConfig holds the webhook destination for audit events.
// Loaded from FLY.IO SECRETS: AUDIT_WEBHOOK_URL.
var webhookConfig struct {
	URL      string
	mu       sync.RWMutex
	initDone bool
	enabled  bool
}

// initWebhook loads webhook URL from FLY.IO secret or env.
// Supports: HTTP/HTTPS endpoints, Grafana Loki (/loki/api/v1/push),
// Datadog (HTTPS forwarder), or any SIEM HTTP endpoint.
func initWebhook(url string) {
	webhookConfig.mu.Lock()
	defer webhookConfig.mu.Unlock()
	webhookConfig.URL = strings.TrimSpace(url)
	webhookConfig.enabled = webhookConfig.URL != ""
	webhookConfig.initDone = true
}

func isWebhookEnabled() bool {
	webhookConfig.mu.RLock()
	defer webhookConfig.mu.RUnlock()
	return webhookConfig.enabled
}

func getWebhookURL() string {
	webhookConfig.mu.RLock()
	defer webhookConfig.mu.RUnlock()
	return webhookConfig.URL
}

// dispatchWebhook sends event to external log aggregator asynchronously.
// Non-blocking — webhook failures are logged to stderr, never block the request.
// Implements exponential backoff: fires immediately, logs on failure.
func dispatchWebhook(event AuditEvent) {
	if !isWebhookEnabled() {
		return
	}
	go func() {
		body, err := json.Marshal(event)
		if err != nil {
			return
		}
		client := &http.Client{Timeout: 5 * time.Second}
		req, err := http.NewRequest("POST", getWebhookURL(), bytes.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-OVAV-Audit", "1")
		req.Header.Set("X-Request-ID", event.ReqID)
		resp, err := client.Do(req)
		if err != nil {
			// Log locally on webhook failure — don't lose the event
			fmt.Fprintf(os.Stderr, "[webhook] delivery failed for req_id=%s action=%s: %v\n",
				event.ReqID, event.Action, err)
			return
		}
		resp.Body.Close()
		if resp.StatusCode >= 400 {
			fmt.Fprintf(os.Stderr, "[webhook] HTTP %d for req_id=%s action=%s\n",
				resp.StatusCode, event.ReqID, event.Action)
		}
	}()
}

// parseRiskScore extracts numeric risk score from AuditEvent.Detail string
// formats like "risk=85 (new IP)" or "risk=30, totp_step_up".
func parseRiskScore(detail string) int {
	if detail == "" {
		return 0
	}
	var score int
	for i := 0; i < len(detail); i++ {
		if detail[i] == '=' && i+1 < len(detail) {
			n := 0
			for j := i + 1; j < len(detail) && detail[j] >= '0' && detail[j] <= '9'; j++ {
				n = n*10 + int(detail[j]-'0')
				if j-i > 3 {
					break
				}
			}
			if n > 0 {
				score = n
			}
		}
	}
	return score
}

// EmitAudit writes a structured audit event to the append-only JSONL log,
// persists to SQLite analytics DB, and dispatches to webhook if enabled.
func EmitAudit(ctx context.Context, event AuditEvent) {
	event.ReqID = ReqIDFrom(ctx)
	logger := getAuditLogger()
	logger.log(event)
	dispatchWebhook(event)
	RecordAuthEvent(event, parseRiskScore(event.Detail))
}

// AuditAuth emits an auth event. Call from auth middleware on every auth attempt.
// Also registers the attempt with the anomaly detector (P1-B).
func AuditAuth(ctx context.Context, userID, email, ip, action, resource, status string, code int, detail string) {
	event := AuditEvent{
		UserID:    userID,
		Email:     email,
		IP:        ip,
		Country:   helperCountryFromIP(ip),
		RayID:     "",
		Action:    action,
		Resource:  resource,
		Method:    "",
		Status:    status,
		Code:      code,
		UserAgent: "",
		Detail:    detail,
	}
	EmitAudit(ctx, event)
	// Track for anomaly detection
	success := status == "ok" && code < 400
	RegisterLoginAttempt(userID, ip, "", "", "", success)
}

// helperCountryFromIP returns a country hint based on IP range heuristics.
// For accurate geo, integrate with a GeoIP service (MaxMind, IPinfo).
// This is a rough heuristic for anomaly detection.
func helperCountryFromIP(ip string) string {
	// Very rough heuristics — replace with real GeoIP in production
	if strings.HasPrefix(ip, "190.") || strings.HasPrefix(ip, "181.") ||
		strings.HasPrefix(ip, "186.") || strings.HasPrefix(ip, "201.") {
		return "LATAM"
	}
	if strings.HasPrefix(ip, "198.41.") || strings.HasPrefix(ip, "104.") {
		return "US" // Cloudflare exits often
	}
	return ""
}

// ── Login anomaly detector (P1-B — advanced alternative to IP allowlist) ───────

// LoginAnomaly tracks login patterns per user for anomaly detection.
// In production this would use Redis; here we use an in-memory ring buffer
// with a 24-hour sliding window per user (accurate enough for admin panel).
type LoginAttempt struct {
	Timestamp time.Time
	IP        string
	Country   string
	City      string
	UserAgent string
	Success   bool
}

var (
	anomalyStore = make(map[string][]*LoginAttempt) // userID → recent attempts
	anomalyMu    sync.RWMutex
	maxAttempts  = 20 // ring buffer size per user
)

// RegisterLoginAttempt records a login attempt for anomaly detection.
func RegisterLoginAttempt(userID, ip, country, city, userAgent string, success bool) {
	anomalyMu.Lock()
	defer anomalyMu.Unlock()
	cutoff := time.Now().Add(-24 * time.Hour)
	userKey := userID
	if userKey == "" {
		userKey = "anonymous:" + ip
	}
	attempts := anomalyStore[userKey]
	// Prune old entries
	kept := make([]*LoginAttempt, 0)
	for _, a := range attempts {
		if a.Timestamp.After(cutoff) {
			kept = append(kept, a)
		}
	}
	kept = append(kept, &LoginAttempt{
		Timestamp: time.Now(),
		IP:        ip,
		Country:   country,
		City:      city,
		UserAgent: userAgent,
		Success:   success,
	})
	// Ring buffer: keep last maxAttempts
	if len(kept) > maxAttempts {
		kept = kept[len(kept)-maxAttempts:]
	}
	anomalyStore[userKey] = kept
}

// DetectLoginAnomaly returns a risk score 0-100 and a description.
// 0 = normal, 100 = definitely suspicious.
func DetectLoginAnomaly(userID, ip, country, userAgent string) (risk int, description string) {
	anomalyMu.RLock()
	defer anomalyMu.RUnlock()
	userKey := userID
	if userKey == "" {
		userKey = "anonymous:" + ip
	}
	attempts := anomalyStore[userKey]
	if len(attempts) == 0 {
		return 0, "first login from this device/location"
	}

	// Count recent failures
	failures := 0
	foreignIPs := 0
	lastSuccessIP := ""
	for _, a := range attempts {
		if !a.Success {
			failures++
		} else {
			lastSuccessIP = a.IP
		}
		if a.IP != ip {
			foreignIPs++
		}
	}

	risk = 0
	var reasons []string

	// Rule 1: Many recent failures → brute force
	if failures >= 5 {
		risk += 40
		reasons = append(reasons, fmt.Sprintf("%d failures in 24h", failures))
	} else if failures >= 3 {
		risk += 20
		reasons = append(reasons, fmt.Sprintf("%d failures in 24h", failures))
	}

	// Rule 2: New IP + new country → travel anomaly
	if lastSuccessIP != "" && lastSuccessIP != ip {
		risk += 30
		reasons = append(reasons, "new IP address")
	}

	// Rule 3: Multiple countries in 24h → impossible travel
	countries := make(map[string]bool)
	for _, a := range attempts {
		if a.Timestamp.After(time.Now().Add(-24 * time.Hour)) {
			countries[a.Country] = true
		}
	}
	if len(countries) > 2 {
		risk += 30
		reasons = append(reasons, fmt.Sprintf("%d countries in 24h", len(countries)))
	}

	if risk > 0 {
		description = strings.Join(reasons, "; ")
	}
	return risk, description
}

// helperIP extracts the real client IP from request (handles CF headers).
func helperIP(r *http.Request) string {
	if ip := r.Header.Get("CF-IPAddress"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

// ── Rate limiting by path category (P0-C) ──────────────────────────────────────

// rateLimitConfig defines requests-per-minute per path category.
var rateLimits = map[string]int{
	"/api/v1/auth/login":      5, // brute force protection
	"/api/v1/auth/user-login": 5,
	"/api/v1/vault/secrets":   60, // vault operations — authenticated
	"/api/v1/vault/blobs":     30,
	"/api/v1/vault/":          60,
	"/api/v1/governor/tasks":  30, // governor ops — prevent spam
	"/api/v1/governor/":       30,
	"/api/v1/security/":       15, // sensitive ops — slower
	"/api/v1/memory/":         30,
	"/api/v1/git/fetch":       10, // git ops
}

type pathRateEntry struct {
	hits    int
	resetAt time.Time
}

var (
	pathRateMu    sync.Mutex
	pathRateStore = make(map[string]map[string]*pathRateEntry) // key: "ip:pathCategory"
)

// checkPathRateLimit returns true if allowed, false if rate limit exceeded.
// Key is IP + path category prefix. Sliding window per minute.
func checkPathRateLimit(ip string, path string) (allowed bool, retryAfter int) {
	rateLimitMu.Lock()
	defer rateLimitMu.Unlock()

	// Find matching category (longest prefix wins)
	category := ""
	for prefix := range rateLimits {
		if strings.HasPrefix(path, prefix) && len(prefix) > len(category) {
			category = prefix
		}
	}
	if category == "" {
		return true, 0 // no rate limit for this path
	}

	now := time.Now()

	if pathRateStore[ip] == nil {
		pathRateStore[ip] = make(map[string]*pathRateEntry)
	}
	entry, ok := pathRateStore[ip][category]
	if !ok || now.After(entry.resetAt) {
		pathRateStore[ip][category] = &pathRateEntry{hits: 1, resetAt: now.Add(time.Minute)}
		return true, 0
	}

	limit := rateLimits[category]
	if entry.hits >= limit {
		retry := int(entry.resetAt.Sub(now).Seconds())
		if retry < 0 {
			retry = 60
		}
		return false, retry
	}

	entry.hits++
	return true, 0
}

var rateLimitMu sync.Mutex

// helperCountry extracts country from CF header.
func helperCountry(r *http.Request) string {
	return r.Header.Get("CF-IPCountry")
}

// helperRayID extracts CF Ray ID.
func helperRayID(r *http.Request) string {
	return r.Header.Get("CF-Ray")
}

// ── Query helper ──────────────────────────────────────────────────────────────

func queryParam(r *http.Request, key, def string) string {
	if v := r.URL.Query().Get(key); v != "" {
		return v
	}
	return def
}

// ── Path helpers ──────────────────────────────────────────────────────────────

// joinPath joins repo root with relative path.
func joinPath(parts ...string) string {
	return filepath.Join(append([]string{RepoRoot}, parts...)...)
}

// fileExists checks if a path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// dirExists checks if a path exists and is a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// readFile reads a file as string, returns empty on error.
func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}
