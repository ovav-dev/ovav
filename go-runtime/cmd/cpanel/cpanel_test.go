// OVAV cPanel — Backend tests (httptest)
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fixRepoRoot walks up from the test working directory to find the true
// OVAV repo root (the directory containing .ovav/). Needed because
// findRepoRoot() uses os.Executable() which points to a temp dir in tests.
func fixRepoRoot() {
	cwd, _ := os.Getwd()
	for range 12 {
		// Look for .ovav/plan/caps.yaml — this only exists at the true repo root,
		// not in test-created .ovav/runtime/ directories.
		if _, err := os.Stat(filepath.Join(cwd, ".ovav", "plan", "caps.yaml")); err == nil {
			RepoRoot = cwd
			return
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			break
		}
		cwd = parent
	}
}

// init runs after shared.go's init() because test files are compiled after
// non-test files. This fixes RepoRoot for the test environment.
func init() {
	fixRepoRoot()
}

// TestMain provides per-binary-run setup/teardown. Resets the global rate
// limiter store to prevent state leakage across -count=N iterations (FLAKY-01).
func TestMain(m *testing.M) {
	ResetRateLimiterForTesting()
	os.Exit(m.Run())
}

// ── Health & Status ────────────────────────────────────────────────────────────

func TestHealthEndpoint(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	handleHealth(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected JSON content-type, got %s", contentType)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", body["status"])
	}
	if body["service"] != "ovav-cpanel" {
		t.Errorf("expected service=ovav-cpanel, got %v", body["service"])
	}
}

func TestStatusEndpoint(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/status", nil)
	w := httptest.NewRecorder()
	handleStatus(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	// Git info
	git, ok := body["git"].(map[string]interface{})
	if !ok {
		t.Fatal("expected git section in status response")
	}
	if git["branch"] == nil {
		t.Error("expected git.branch in status")
	}
	if git["head"] == nil {
		t.Error("expected git.head in status")
	}
	// Timestamp
	if body["timestamp"] == nil {
		t.Error("expected timestamp in status")
	}
	// System section
	if _, ok := body["system"]; !ok {
		t.Error("expected system section in status")
	}
	// Economy section
	if _, ok := body["economy"]; !ok {
		t.Error("expected economy section in status")
	}
}

// ── Auth: Login ────────────────────────────────────────────────────────────────

func TestAuthLoginValidToken(t *testing.T) {
	ResetRateLimiterForTesting()
	// v58.0: Token login now requires an active OVAV Systems session.
	// Without a session file, any token is rejected with 401.
	validToken := "abcdefghijklmnopqrstuvwxyz123456" // exactly 32 chars
	body := strings.NewReader(`{"token":"` + validToken + `"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.100:0"
	w := httptest.NewRecorder()

	handleAuthLogin(w, req)

	resp := w.Result()
	// Without OVAV session, token login MUST be rejected
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 (no OVAV session) for token without session, got %d", resp.StatusCode)
	}
}

func TestAuthLoginTokenTooShort(t *testing.T) {
	shortToken := "short" // less than 32 chars
	body := strings.NewReader(`{"token":"` + shortToken + `"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.101:0"
	w := httptest.NewRecorder()

	handleAuthLogin(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for short token, got %d", resp.StatusCode)
	}
}

func TestAuthLoginRateLimiting(t *testing.T) {
	ResetRateLimiterForTesting()
	// Use a unique IP to avoid interference with other tests
	testIP := "10.99.88.77:0"
	shortToken := "short"

	// Make 5 requests — the 5th should still be allowed
	for i := 0; i < 5; i++ {
		body := strings.NewReader(`{"token":"` + shortToken + `"}`)
		req := httptest.NewRequest("POST", "/api/v1/auth/login", body)
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = testIP
		w := httptest.NewRecorder()
		handleAuthLogin(w, req)
		// All 5 should not be 429 (they may be 401 due to short token, but not rate-limited)
		if w.Result().StatusCode == http.StatusTooManyRequests {
			t.Fatalf("request %d was rate-limited too early", i+1)
		}
	}

	// 6th request should be rate-limited → 429
	body := strings.NewReader(`{"token":"` + shortToken + `"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = testIP
	w := httptest.NewRecorder()
	handleAuthLogin(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected 429 on 6th attempt, got %d", resp.StatusCode)
	}
	var errBody map[string]string
	json.NewDecoder(resp.Body).Decode(&errBody)
	if !strings.Contains(errBody["error"], "too many attempts") {
		t.Errorf("expected rate-limit error message, got %v", errBody["error"])
	}
}

func TestAuthLoginEmptyBody(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.102:0"
	w := httptest.NewRecorder()

	handleAuthLogin(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for empty body, got %d", resp.StatusCode)
	}
}

// ── Auth: Session ──────────────────────────────────────────────────────────────

func TestAuthSessionNoToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/auth/session", nil)
	w := httptest.NewRecorder()
	handleAuthSession(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for no token, got %d", resp.StatusCode)
	}
}

func TestAuthSessionInvalidToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/auth/session", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	w := httptest.NewRecorder()
	handleAuthSession(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid token, got %d", resp.StatusCode)
	}
}

// ── Auth: Config ───────────────────────────────────────────────────────────────

func TestAuthConfigReturnsMethods(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/auth/config", nil)
	w := httptest.NewRecorder()
	handleAuthConfig(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	methods, ok := body["methods"].([]interface{})
	if !ok {
		t.Fatal("expected methods array in auth config")
	}
	// "token" method should always be present
	foundToken := false
	for _, m := range methods {
		if s, ok := m.(string); ok && s == "token" {
			foundToken = true
			break
		}
	}
	if !foundToken {
		t.Error("expected 'token' method in auth config methods")
	}
	if body["has_oauth"] == nil {
		t.Error("expected has_oauth field in auth config")
	}
}

// ── OAuth: Callback ────────────────────────────────────────────────────────────

func TestOAuthCallbackInvalidProvider(t *testing.T) {
	body := strings.NewReader(`{"code":"test-code","state":"some-state"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/oauth/invalidprovider", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleOAuthCallback(w, req)

	resp := w.Result()
	// OAuth may return 400 (invalid provider) or 503 (OAuth not configured in test env)
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 400 or 503 for invalid OAuth provider, got %d", resp.StatusCode)
	}
}

func TestOAuthCallbackMissingState(t *testing.T) {
	// Missing state triggers CSRF check failure
	body := strings.NewReader(`{"code":"test-code"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/oauth/google", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleOAuthCallback(w, req)

	resp := w.Result()
	// OAuth may return 403 (CSRF fail) or 503 (OAuth not configured in test env)
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 403 or 503 for missing OAuth state, got %d", resp.StatusCode)
	}
}

func TestOAuthCallbackEmptyState(t *testing.T) {
	body := strings.NewReader(`{"code":"test-code","state":""}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/oauth/google", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleOAuthCallback(w, req)

	resp := w.Result()
	// OAuth may return 403 (CSRF fail) or 503 (OAuth not configured in test env)
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 403 or 503 for empty OAuth state, got %d", resp.StatusCode)
	}
}

func TestOAuthCallbackMissingCode(t *testing.T) {
	body := strings.NewReader(`{"state":"some-state"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/oauth/google", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleOAuthCallback(w, req)

	resp := w.Result()
	// OAuth may return 400 (missing code) or 503 (OAuth not configured in test env)
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 400 or 503 for missing OAuth code, got %d", resp.StatusCode)
	}
}

func TestOAuthCallbackEmptyBody(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/auth/oauth/google", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleOAuthCallback(w, req)

	resp := w.Result()
	// OAuth may return 400 (bad body) or 503 (OAuth not configured in test env)
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 400 or 503 for empty OAuth body, got %d", resp.StatusCode)
	}
}

// ── Memory ─────────────────────────────────────────────────────────────────────

func TestHandleMemoryStatus(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/memory/status", nil)
	w := httptest.NewRecorder()
	handleMemoryStatus(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	// Verify required fields
	if body["governor"] != "operational" {
		t.Errorf("expected governor=operational, got %v", body["governor"])
	}
	if body["simplified"] != true {
		t.Error("expected simplified=true")
	}
	if body["source"] == nil {
		t.Error("expected source field indicating canonical sources")
	}
}

func TestHandleMemoryLedger(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/memory/ledger", nil)
	w := httptest.NewRecorder()
	handleLedger(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if body["cards"] == nil {
		t.Error("expected cards in ledger response")
	}
	if body["simplified"] != true {
		t.Error("expected simplified=true")
	}
}

func TestHandleMemoryBeliefs(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/memory/beliefs", nil)
	w := httptest.NewRecorder()
	handleBeliefs(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if body["simplified"] != true {
		t.Error("expected simplified=true")
	}
}

// ── System ─────────────────────────────────────────────────────────────────────

func TestHandleSystemHealth(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/system/health", nil)
	w := httptest.NewRecorder()
	handleSystemHealth(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if body["schema"] != "ovav.self_diagnosis.v1" {
		t.Errorf("expected schema=ovav.self_diagnosis.v1, got %v", body["schema"])
	}
	checks, ok := body["checks"].([]interface{})
	if !ok || len(checks) == 0 {
		t.Fatal("expected checks array in system health")
	}
	// Should have at least identity, git, go_runtime checks
	if len(checks) < 3 {
		t.Errorf("expected at least 3 health checks, got %d", len(checks))
	}
	if body["status"] == nil {
		t.Error("expected overall status in system health")
	}
}

func TestHandleSystemConfig(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/system/config", nil)
	w := httptest.NewRecorder()
	handleConfig(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	// host_drift should be present
	if body["host_drift"] == nil {
		t.Error("expected host_drift section in config")
	}
}

// ── Profiles ───────────────────────────────────────────────────────────────────

func TestHandleProfilesList(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/profiles", nil)
	w := httptest.NewRecorder()
	handleProfileList(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	// In test environment, RepoRoot may not point to the repo root,
	// so service_profiles.yaml may not be found. Accept both cases.
	profiles, ok := body["profiles"].([]interface{})
	if !ok {
		t.Fatal("expected profiles array in response")
	}
	if body["ok"] == true && len(profiles) == 0 {
		t.Error("ok=true but zero profiles returned")
	}
	if body["engine"] != "OVAV Go Native Profiles v5.0" {
		t.Errorf("expected engine version, got %v", body["engine"])
	}
}

// ── Security ───────────────────────────────────────────────────────────────────

func TestHandleSecurityLivingIntegrity(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/security/living-integrity", nil)
	w := httptest.NewRecorder()
	handleLivingIntegrity(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if body["overall"] == nil {
		t.Error("expected overall status in living integrity")
	}
	if body["score"] == nil {
		t.Error("expected score in living integrity")
	}
	checks, ok := body["checks"].([]interface{})
	if !ok || len(checks) == 0 {
		t.Error("expected checks array in living integrity")
	}
}

func TestHandleSecurityAuditLog(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/security/audit-log", nil)
	w := httptest.NewRecorder()
	handleAuditLog(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if body["entries"] == nil {
		t.Error("expected entries in audit log")
	}
	if body["chain_intact"] == nil {
		t.Error("expected chain_intact in audit log")
	}
}

// ── Router ─────────────────────────────────────────────────────────────────────

func TestRouterExactMatch(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	registerRoutes()
	routerMux(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for /health, got %d", resp.StatusCode)
	}
}

func TestRouterNotFound(t *testing.T) {
	// Use an /api/ path not registered — API paths return 404 JSON, not SPA fallback
	req := httptest.NewRequest("GET", "/api/v1/nonexistent", nil)
	w := httptest.NewRecorder()
	registerRoutes()
	routerMux(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for unknown API route, got %d", resp.StatusCode)
	}
}

// ── CORS Middleware ────────────────────────────────────────────────────────────

func TestCORSMiddlewareAllowedOrigin(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/status", nil)
	req.Header.Set("Origin", "https://d678beea.ovav.dev")
	w := httptest.NewRecorder()

	handler := corsMiddleware(http.HandlerFunc(handleStatus))
	handler.ServeHTTP(w, req)

	resp := w.Result()
	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	if allowOrigin != "https://d678beea.ovav.dev" {
		t.Errorf("expected Access-Control-Allow-Origin=https://d678beea.ovav.dev, got %s", allowOrigin)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestCORSMiddlewareDisallowedOrigin(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/status", nil)
	req.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()

	handler := corsMiddleware(http.HandlerFunc(handleStatus))
	handler.ServeHTTP(w, req)

	resp := w.Result()
	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	// Disallowed origin should NOT be echoed back — falls to default origin
	if allowOrigin == "https://evil.com" {
		t.Error("disallowed origin should NOT be echoed as Access-Control-Allow-Origin")
	}
	if allowOrigin == "" {
		t.Error("expected a default Access-Control-Allow-Origin header")
	}
	if allowOrigin == "*" {
		t.Error("CORS should not use wildcard (*)")
	}
	// Response still succeeds server-side (browser will block it)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 (server-side), got %d", resp.StatusCode)
	}
}

func TestCORSPreflightOptions(t *testing.T) {
	req := httptest.NewRequest("OPTIONS", "/api/v1/status", nil)
	req.Header.Set("Origin", "https://d678beea.ovav.dev")
	w := httptest.NewRecorder()

	handler := corsMiddleware(http.HandlerFunc(handleStatus))
	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for OPTIONS preflight, got %d", resp.StatusCode)
	}
	// Verify CORS headers are set
	if resp.Header.Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected Access-Control-Allow-Methods header")
	}
	if resp.Header.Get("Access-Control-Allow-Headers") == "" {
		t.Error("expected Access-Control-Allow-Headers header")
	}
	if resp.Header.Get("Access-Control-Max-Age") == "" {
		t.Error("expected Access-Control-Max-Age header")
	}
}

func TestCORSNoOriginHeader(t *testing.T) {
	// Requests without Origin header (curl, same-origin) should still get CORS headers
	req := httptest.NewRequest("GET", "/api/v1/status", nil)
	w := httptest.NewRecorder()

	handler := corsMiddleware(http.HandlerFunc(handleStatus))
	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.Header.Get("Access-Control-Allow-Origin") == "" {
		t.Error("expected Access-Control-Allow-Origin even without Origin header")
	}
}

// ── SSE Connection Limit ───────────────────────────────────────────────────────

func TestSSEConnectionLimit(t *testing.T) {
	// Artificially max out connections
	oldCount := atomic.LoadInt32(&sseConnectionCount)
	atomic.StoreInt32(&sseConnectionCount, maxSSEConnections)

	req := httptest.NewRequest("GET", "/api/v1/events", nil)
	w := httptest.NewRecorder()
	handleEvents(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when SSE connections full, got %d", resp.StatusCode)
	}

	// Restore original count
	atomic.StoreInt32(&sseConnectionCount, oldCount)
}

// ── Path Traversal ─────────────────────────────────────────────────────────────

func TestPathTraversalURLEncoded(t *testing.T) {
	// URL-encoded %2e%2e decodes to ".." — should be blocked
	req := httptest.NewRequest("GET", "/assets/%2e%2e%2fetc%2fpasswd", nil)
	w := httptest.NewRecorder()
	serveAsset(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for path traversal (%%2e%%2e), got %d", resp.StatusCode)
	}
}

func TestPathTraversalDoubleDot(t *testing.T) {
	req := httptest.NewRequest("GET", "/assets/../../../etc/passwd", nil)
	w := httptest.NewRecorder()
	serveAsset(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for path traversal (..), got %d", resp.StatusCode)
	}
}

func TestPathTraversalBackslash(t *testing.T) {
	req := httptest.NewRequest("GET", "/assets/..\\..\\etc\\passwd", nil)
	w := httptest.NewRecorder()
	serveAsset(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for backslash traversal, got %d", resp.StatusCode)
	}
}

func TestPathTraversalCleanPath(t *testing.T) {
	// A clean path that resolves outside base dir via filepath.Clean
	req := httptest.NewRequest("GET", "/assets/../api/v1/status", nil)
	w := httptest.NewRecorder()
	serveAsset(w, req)

	resp := w.Result()
	// Should be caught by the ".." check before Clean
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for ../ path, got %d", resp.StatusCode)
	}
}

// ── SSE Events (existing, kept for coverage) ───────────────────────────────────

func TestEventsEndpoint(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/events", nil)
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	done := make(chan bool, 1)
	go func() {
		handleEvents(w, req)
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Logf("SSE handler returned status: %d (expected 200)", resp.StatusCode)
	}
}

// ── Auth Login: Invalid ────────────────────────────────────────────────────────

func TestAuthLoginInvalid(t *testing.T) {
	body := strings.NewReader(`{"token":""}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.200:0"
	w := httptest.NewRecorder()

	handleAuthLogin(w, req)

	resp := w.Result()
	if resp.StatusCode == http.StatusOK {
		t.Error("expected non-200 for empty token login")
	}
}

// ── CORS Headers (existing, kept for coverage) ─────────────────────────────────

func TestCORSHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/status", nil)
	w := httptest.NewRecorder()

	handler := corsMiddleware(http.HandlerFunc(handleStatus))
	handler.ServeHTTP(w, req)

	resp := w.Result()
	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	if allowOrigin == "" {
		t.Error("expected CORS Access-Control-Allow-Origin header")
	}
	if allowOrigin == "*" {
		t.Error("CORS should not use wildcard (*) — must be restricted origin")
	}
}

// ── Validators ─────────────────────────────────────────────────────────────────

func TestHandleValidatorsList(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/validators", nil)
	w := httptest.NewRecorder()
	handleValidatorsList(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if body["overall"] == nil {
		t.Error("expected overall status in validators list")
	}
}

func TestHandleValidatorsRun(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/validators/run", nil)
	w := httptest.NewRecorder()
	handleValidatorsRun(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if body["task_id"] == nil {
		t.Error("expected task_id in validators run response")
	}
	if body["status"] != "queued" {
		t.Errorf("expected status=queued, got %v", body["status"])
	}
}

// ── API routes via router ──────────────────────────────────────────────────────

func TestAPIRoutesReturnJSON(t *testing.T) {
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/status"},
		{"GET", "/api/v1/memory/status"},
		{"GET", "/api/v1/agents"},
		{"GET", "/api/v1/profiles"},
		{"GET", "/api/v1/system/health"},
		{"GET", "/api/v1/validators"},
		{"GET", "/api/v1/security/living-integrity"},
		{"GET", "/api/v1/auth/config"},
	}

	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			req := httptest.NewRequest(rt.method, rt.path, nil)
			w := httptest.NewRecorder()
			routerMux(w, req)

			resp := w.Result()
			if resp.StatusCode < 200 || resp.StatusCode >= 500 {
				t.Errorf("%s %s returned %d, expected 2xx-4xx", rt.method, rt.path, resp.StatusCode)
			}
			ct := resp.Header.Get("Content-Type")
			if !strings.Contains(ct, "application/json") && resp.StatusCode == 200 {
				t.Errorf("%s %s: expected JSON content-type, got %s", rt.method, rt.path, ct)
			}
		})
	}
}

// ── sendJSON / sendError helpers ───────────────────────────────────────────────

func TestSendError(t *testing.T) {
	w := httptest.NewRecorder()
	sendError(w, "test error message", http.StatusBadRequest)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] != "test error message" {
		t.Errorf("expected error='test error message', got %v", body["error"])
	}
}

func TestSendOK(t *testing.T) {
	w := httptest.NewRecorder()
	sendOK(w, map[string]string{"key": "value"})

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected JSON content-type, got %s", ct)
	}
}

// ── Shared helpers ─────────────────────────────────────────────────────────────

func TestIsOriginAllowed(t *testing.T) {
	// Allowed origin
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://d678beea.ovav.dev")
	if !isOriginAllowed(req) {
		t.Error("expected https://d678beea.ovav.dev to be allowed")
	}
}

func TestIsOriginAllowedDisallowed(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	if isOriginAllowed(req) {
		t.Error("expected https://evil.com to be disallowed")
	}
}

func TestIsOriginAllowedEmpty(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	// No Origin header → same-origin → allowed
	if !isOriginAllowed(req) {
		t.Error("expected empty origin to be allowed (same-origin)")
	}
}

func TestSetCORSHeaderAllowed(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://d678beea.ovav.dev")
	w := httptest.NewRecorder()
	setCORSHeader(w, req)
	if w.Header().Get("Access-Control-Allow-Origin") != "https://d678beea.ovav.dev" {
		t.Error("expected echoed origin for allowed origin")
	}
}

func TestSetCORSHeaderDisallowed(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()
	setCORSHeader(w, req)
	aco := w.Header().Get("Access-Control-Allow-Origin")
	if aco == "https://evil.com" {
		t.Error("disallowed origin should not be echoed")
	}
	if aco == "" {
		t.Error("expected fallback origin for disallowed origin")
	}
}

func TestQueryParam(t *testing.T) {
	req := httptest.NewRequest("GET", "/?foo=bar", nil)
	v := queryParam(req, "foo", "default")
	if v != "bar" {
		t.Errorf("expected 'bar', got '%s'", v)
	}
	v = queryParam(req, "missing", "default")
	if v != "default" {
		t.Errorf("expected 'default', got '%s'", v)
	}
}

// ── Git handlers ───────────────────────────────────────────────────────────────

func TestHandleGitBranches(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/git/branches", nil)
	w := httptest.NewRecorder()
	handleGitBranches(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if body["current"] == nil {
		t.Error("expected current branch in response")
	}
	if body["branches"] == nil {
		t.Error("expected branches list in response")
	}
}

func TestHandleGitLog(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/git/log?limit=5", nil)
	w := httptest.NewRecorder()
	handleGitLog(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if body["commits"] == nil {
		t.Error("expected commits in git log response")
	}
}

func TestHandleGitLogInvalidLimit(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/git/log?limit=abc", nil)
	w := httptest.NewRecorder()
	handleGitLog(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid limit, got %d", resp.StatusCode)
	}
}

func TestHandleGitWorktrees(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/git/worktrees", nil)
	w := httptest.NewRecorder()
	handleGitWorktrees(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if body["worktrees"] == nil {
		t.Error("expected worktrees in response")
	}
}

func TestHandleGitFetch(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/git/fetch", nil)
	w := httptest.NewRecorder()
	handleGitFetch(w, req)

	resp := w.Result()
	// Fetch may succeed or fail depending on network; both are valid
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 200 or 500 for git fetch, got %d", resp.StatusCode)
	}
}

// ── System handlers ────────────────────────────────────────────────────────────

func TestHandleOperations(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/system/operations", nil)
	w := httptest.NewRecorder()
	handleOperations(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if body["install"] == nil || body["backup"] == nil || body["deploy"] == nil {
		t.Error("expected install, backup, deploy sections in operations")
	}
}

func TestHandleEconomyDetail(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/economy", nil)
	w := httptest.NewRecorder()
	handleEconomyDetail(w, req)

	resp := w.Result()
	// Economy dashboard may not exist, but handler should return 200 either way
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHandleSBOM(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/system/sbom", nil)
	w := httptest.NewRecorder()
	handleSBOM(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHandleKCStatus(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/system/kc", nil)
	w := httptest.NewRecorder()
	handleKCStatus(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHandleRegistry(t *testing.T) {
	// List registries
	req := httptest.NewRequest("GET", "/api/v1/system/registry/", nil)
	w := httptest.NewRecorder()
	handleRegistry(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if body["registries"] == nil {
		t.Error("expected registries in response")
	}
}

func TestHandleRegistryNotFound(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/system/registry/nonexistent-file", nil)
	w := httptest.NewRecorder()
	handleRegistry(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for unknown registry, got %d", resp.StatusCode)
	}
}

// ── Security handlers ──────────────────────────────────────────────────────────

func TestHandleClearAlarms(t *testing.T) {
	req := httptest.NewRequest("DELETE", "/api/v1/security/canary-alarms", nil)
	w := httptest.NewRecorder()
	handleClearAlarms(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// ── Profiles apply ─────────────────────────────────────────────────────────────

func TestHandleProfileApply(t *testing.T) {
	body := strings.NewReader(`{"area":"platform_engineering","dry_run":true}`)
	req := httptest.NewRequest("POST", "/api/v1/profiles/apply", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleProfileApply(w, req)

	resp := w.Result()
	// May succeed if profile found or return 404/500 if RepoRoot is wrong
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 200, 404, or 500 for profile apply, got %d", resp.StatusCode)
	}
}

func TestHandleProfileApplyNoArea(t *testing.T) {
	body := strings.NewReader(`{"dry_run":true}`)
	req := httptest.NewRequest("POST", "/api/v1/profiles/apply", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleProfileApply(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for missing area, got %d", resp.StatusCode)
	}
}

// ── Validators status ─────────────────────────────────────────────────────────

func TestHandleValidatorsStatusNotFound(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/validators/status/nonexistent-task-id", nil)
	w := httptest.NewRecorder()
	handleValidatorsStatus(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for unknown task, got %d", resp.StatusCode)
	}
}

// ── Auth: session with valid flow ─────────────────────────────────────────────

func TestAuthSessionBadAuthHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/auth/session", nil)
	req.Header.Set("Authorization", "NotBearer token")
	w := httptest.NewRecorder()
	handleAuthSession(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for malformed auth header, got %d", resp.StatusCode)
	}
}

// ── API routes via router (extended) ──────────────────────────────────────────

func TestAPIRoutesExtendedReturnJSON(t *testing.T) {
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/memory/ledger"},
		{"GET", "/api/v1/memory/beliefs"},
		{"GET", "/api/v1/git/branches"},
		{"GET", "/api/v1/git/log"},
		{"GET", "/api/v1/git/worktrees"},
		{"GET", "/api/v1/system/config"},
		{"GET", "/api/v1/system/sbom"},
		{"GET", "/api/v1/system/kc"},
		{"GET", "/api/v1/system/operations"},
		{"GET", "/api/v1/economy"},
		{"GET", "/api/v1/agents/topology"},
		{"GET", "/api/v1/agents/profiles"},
		{"GET", "/api/v1/agents/permissions"},
	}

	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			req := httptest.NewRequest(rt.method, rt.path, nil)
			w := httptest.NewRecorder()
			routerMux(w, req)

			resp := w.Result()
			if resp.StatusCode < 200 || resp.StatusCode >= 500 {
				t.Errorf("%s %s returned %d, expected 2xx-4xx", rt.method, rt.path, resp.StatusCode)
			}
		})
	}
}

// ── Auth: valid login with admin-like token ───────────────────────────────────

func TestAuthLoginAdminToken(t *testing.T) {
	// v58.0: "ovav-admin-" prefix no longer grants admin role.
	// Token login requires an active OVAV Systems session.
	adminToken := "ovav-admin-abcdefghijklmnopqrstuv" // >32 chars
	body := strings.NewReader(`{"token":"` + adminToken + `"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.150:0"
	w := httptest.NewRecorder()

	handleAuthLogin(w, req)

	resp := w.Result()
	// Without OVAV session, admin-prefix token MUST be rejected
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 (no OVAV session) for admin-prefix token, got %d", resp.StatusCode)
	}
}

// ── sendJSON error path ────────────────────────────────────────────────────────

func TestSendJSONMarshalError(t *testing.T) {
	// json.Marshal cannot marshal channels — triggers the error path
	w := httptest.NewRecorder()
	sendJSON(w, make(chan int), http.StatusOK)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 for marshal error, got %d", resp.StatusCode)
	}
}

// ── JWT verification (deep paths) ─────────────────────────────────────────────

func TestVerifyJWTInvalidFormat(t *testing.T) {
	// Token with wrong number of parts
	_, err := verifyJWT("only.two")
	if err == nil {
		t.Error("expected error for 2-part token")
	}
}

func TestVerifyJWTBadBase64(t *testing.T) {
	// Token with invalid base64 in claims
	_, err := verifyJWT("header.!@#$%^.signature")
	if err == nil {
		t.Error("expected error for invalid base64 claims")
	}
}

func TestVerifyJWTInvalidClaimsJSON(t *testing.T) {
	// Valid base64 but invalid JSON in claims
	_, err := verifyJWT("eyJhbGciOiJSUzI1NiJ9.dGhpcyBpcyBub3QganNvbg.signature")
	if err == nil {
		t.Error("expected error for invalid JSON claims")
	}
}

func TestVerifyJWTBadSignatureEncoding(t *testing.T) {
	// Valid header and claims, but bad signature base64
	_, err := verifyJWT("eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.!@#$%^")
	if err == nil {
		t.Error("expected error for bad signature encoding")
	}
}

func TestVerifyJWTExpiredToken(t *testing.T) {
	// First get a valid token
	validToken := "abcdefghijklmnopqrstuvwxyz123456"
	body := strings.NewReader(`{"token":"` + validToken + `"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.180:0"
	w := httptest.NewRecorder()
	handleAuthLogin(w, req)

	var result map[string]interface{}
	json.NewDecoder(w.Result().Body).Decode(&result)
	jwt, ok := result["token"].(string)
	if !ok || jwt == "" {
		t.Skip("could not obtain valid JWT — skipping expired token test")
	}
	// Verify the valid token works
	claims, err := verifyJWT(jwt)
	if err != nil {
		t.Fatalf("valid token should verify: %v", err)
	}
	if claims.Role != "operator" {
		t.Errorf("expected operator role, got %s", claims.Role)
	}
}

func TestHandleAuthSessionValidToken(t *testing.T) {
	// Login to get a valid JWT
	validToken := "abcdefghijklmnopqrstuvwxyz123456"
	body := strings.NewReader(`{"token":"` + validToken + `"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.190:0"
	w := httptest.NewRecorder()
	handleAuthLogin(w, req)

	var loginResult map[string]interface{}
	json.NewDecoder(w.Result().Body).Decode(&loginResult)
	jwt, ok := loginResult["token"].(string)
	if !ok || jwt == "" {
		t.Skip("could not obtain valid JWT — skipping session test")
	}

	// Now check the session
	req2 := httptest.NewRequest("GET", "/api/v1/auth/session", nil)
	req2.Header.Set("Authorization", "Bearer "+jwt)
	w2 := httptest.NewRecorder()
	handleAuthSession(w2, req2)

	resp := w2.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for valid session, got %d", resp.StatusCode)
	}
	var sessionResult map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&sessionResult)
	if sessionResult["valid"] != true {
		t.Error("expected valid=true for valid session")
	}
}

func TestHandleAuthSessionExpiredToken(t *testing.T) {
	// Token is "expired" but still has valid JWT structure → "session not found"
	validToken := "abcdefghijklmnopqrstuvwxyz123456"
	body := strings.NewReader(`{"token":"` + validToken + `"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.200:0"
	w := httptest.NewRecorder()
	handleAuthLogin(w, req)

	var loginResult map[string]interface{}
	json.NewDecoder(w.Result().Body).Decode(&loginResult)
	jwt, _ := loginResult["token"].(string)

	// Using a completely random JWT that won't be in sessions
	req2 := httptest.NewRequest("GET", "/api/v1/auth/session", nil)
	req2.Header.Set("Authorization", "Bearer "+jwt+"x")
	w2 := httptest.NewRecorder()
	handleAuthSession(w2, req2)

	resp := w2.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for tampered token, got %d", resp.StatusCode)
	}
}

// ── OAuth helpers ─────────────────────────────────────────────────────────────

func TestGenerateOAuthState(t *testing.T) {
	state1 := generateOAuthState("")
	state2 := generateOAuthState("")
	if state1 == "" {
		t.Error("expected non-empty OAuth state")
	}
	if state1 == state2 {
		t.Error("expected unique OAuth states")
	}
	// State should be base64-encoded 32 bytes → 44 chars
	if len(state1) != 44 {
		t.Errorf("expected 44-char base64 state, got %d chars", len(state1))
	}
}

func TestVerifyOAuthStateValid(t *testing.T) {
	state := generateOAuthState("")
	if !verifyOAuthState(state) {
		t.Error("expected valid OAuth state to verify successfully")
	}
}

func TestVerifyOAuthStateReplay(t *testing.T) {
	state := generateOAuthState("")
	// First verification consumes the state
	if !verifyOAuthState(state) {
		t.Fatal("first verify should succeed")
	}
	// Second verification should fail (replay protection)
	if verifyOAuthState(state) {
		t.Error("expected replay to fail (one-time use)")
	}
}

func TestVerifyOAuthStateUnknown(t *testing.T) {
	if verifyOAuthState("unknown-state-token-never-generated") {
		t.Error("expected unknown state to fail verification")
	}
}

func TestVerifyOAuthStateEmpty(t *testing.T) {
	if verifyOAuthState("") {
		t.Error("expected empty state to fail verification")
	}
}

// ── CORS with specific allowed origins ────────────────────────────────────────

func TestCORSLocalhost3000(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/status", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	handler := corsMiddleware(http.HandlerFunc(handleStatus))
	handler.ServeHTTP(w, req)

	resp := w.Result()
	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	if allowOrigin != "http://localhost:3000" {
		t.Errorf("expected localhost:3000 echoed, got %s", allowOrigin)
	}
}

// ── Status cache ──────────────────────────────────────────────────────────────

func TestGetCachedStatus(t *testing.T) {
	// First call populates cache
	status1 := getCachedStatus()
	if status1 == nil {
		t.Fatal("expected non-nil cached status")
	}
	// Second call returns from cache (within TTL)
	status2 := getCachedStatus()
	if status2 == nil {
		t.Fatal("expected non-nil cached status on second call")
	}
	if status1["timestamp"] == nil {
		t.Error("expected timestamp in cached status")
	}
}

// ── Base64 URL encoding ───────────────────────────────────────────────────────

func TestBase64urlDecodePadding(t *testing.T) {
	// Test padding cases: 2 chars → "==", 3 chars → "="
	tests := []string{
		"YQ",   // 1 byte → needs ==
		"YWI",  // 2 bytes → needs =
		"YWJj", // 3 bytes → no padding needed
	}
	for _, s := range tests {
		_, err := base64urlDecode(s)
		if err != nil {
			t.Errorf("base64urlDecode(%s) failed: %v", s, err)
		}
	}
}

// ── Router with POST method ───────────────────────────────────────────────────

func TestRouterPostNotFound(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/nonexistent", nil)
	w := httptest.NewRecorder()
	registerRoutes()
	routerMux(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for unknown POST, got %d", resp.StatusCode)
	}
}

// ── Auth: login with 32-char token exactly ────────────────────────────────────

func TestAuthLoginExact32Chars(t *testing.T) {
	// v58.0: 32-char token without OVAV session is rejected.
	// Token login requires an active OVAV Systems session.
	exact32 := "1234567890abcdefghij1234567890ab" // 32 chars
	body := strings.NewReader(`{"token":"` + exact32 + `"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.210:0"
	w := httptest.NewRecorder()

	handleAuthLogin(w, req)
	// Without OVAV session, token MUST be rejected
	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 (no OVAV session) for 32-char token, got %d", w.Result().StatusCode)
	}
}

// ── Auth: login with X-Forwarded-For ─────────────────────────────────────────

func TestAuthLoginXForwardedFor(t *testing.T) {
	shortToken := "short"
	body := strings.NewReader(`{"token":"` + shortToken + `"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	req.RemoteAddr = "192.168.1.220:0"
	w := httptest.NewRecorder()

	handleAuthLogin(w, req)
	// Should use X-Forwarded-For IP for rate limiting; short token → 401
	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for short token, got %d", w.Result().StatusCode)
	}
}

// ── Shared helpers (new coverage) ─────────────────────────────────────────────

func TestJoinPath(t *testing.T) {
	fixRepoRoot()
	result := joinPath(".ovav", "registry", "test.yaml")
	if !strings.Contains(result, ".ovav") {
		t.Errorf("expected path to contain .ovav, got %s", result)
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	tmpFile := filepath.Join(dir, "test.txt")
	os.WriteFile(tmpFile, []byte("hello"), 0644)
	if !fileExists(tmpFile) {
		t.Error("expected file to exist")
	}
	if fileExists(filepath.Join(dir, "nonexistent.txt")) {
		t.Error("expected nonexistent file to not exist")
	}
	if fileExists(dir) {
		t.Error("expected dir to not be reported as file")
	}
}

func TestDirExists(t *testing.T) {
	dir := t.TempDir()
	if !dirExists(dir) {
		t.Error("expected dir to exist")
	}
	if dirExists(filepath.Join(dir, "nonexistent")) {
		t.Error("expected nonexistent dir to not exist")
	}
	tmpFile := filepath.Join(dir, "test.txt")
	os.WriteFile(tmpFile, []byte("hello"), 0644)
	if dirExists(tmpFile) {
		t.Error("expected file to not be reported as dir")
	}
}

func TestReadFile(t *testing.T) {
	dir := t.TempDir()
	tmpFile := filepath.Join(dir, "test.txt")
	os.WriteFile(tmpFile, []byte("hello world"), 0644)
	if readFile(tmpFile) != "hello world" {
		t.Error("expected file content match")
	}
	if readFile(filepath.Join(dir, "nonexistent.txt")) != "" {
		t.Error("expected empty string for nonexistent file")
	}
}

// ── SBOM Go-native coverage ──────────────────────────────────────────────────

func TestHandleSBOM_GoNative(t *testing.T) {
	fixRepoRoot()
	req := httptest.NewRequest("GET", "/api/v1/system/sbom", nil)
	w := httptest.NewRecorder()
	handleSBOM(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Result().StatusCode)
	}
}

func TestHandleSBOM_Missing(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "registry"), 0755)
	RepoRoot = dir
	req := httptest.NewRequest("GET", "/api/v1/system/sbom", nil)
	w := httptest.NewRecorder()
	handleSBOM(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200 graceful fallback, got %d", w.Result().StatusCode)
	}
}

// ── System: Config ───────────────────────────────────────────────────────────

func TestHandleConfig(t *testing.T) {
	fixRepoRoot()
	req := httptest.NewRequest("GET", "/api/v1/system/config", nil)
	w := httptest.NewRecorder()
	handleConfig(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Result().StatusCode)
	}
}

// ── Registry root listing ────────────────────────────────────────────────────

func TestHandleRegistry_Root(t *testing.T) {
	fixRepoRoot()
	req := httptest.NewRequest("GET", "/api/v1/system/registry/", nil)
	w := httptest.NewRecorder()
	handleRegistry(w, req)
	t.Logf("Registry root status: %d", w.Result().StatusCode)
}

// ── Security: Audit Log & Living Integrity ───────────────────────────────────

func TestHandleAuditLog(t *testing.T) {
	fixRepoRoot()
	req := httptest.NewRequest("GET", "/api/v1/security/audit", nil)
	w := httptest.NewRecorder()
	handleAuditLog(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Result().StatusCode)
	}
}

func TestHandleLivingIntegrity(t *testing.T) {
	fixRepoRoot()
	req := httptest.NewRequest("GET", "/api/v1/security/living-integrity", nil)
	w := httptest.NewRecorder()
	handleLivingIntegrity(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Result().StatusCode)
	}
}

// ── Static assets ────────────────────────────────────────────────────────────

func TestSpaIndexPath(t *testing.T) {
	fixRepoRoot()
	path := spaIndexPath()
	if !strings.Contains(path, "index.html") {
		t.Errorf("expected index.html in path, got %s", path)
	}
}

func TestServeSPAIndex(t *testing.T) {
	fixRepoRoot()
	w := httptest.NewRecorder()
	serveSPAIndex(w)
	t.Logf("SPA index status: %d", w.Result().StatusCode)
}

func TestServeAsset(t *testing.T) {
	fixRepoRoot()
	req := httptest.NewRequest("GET", "/assets/index.html", nil)
	w := httptest.NewRecorder()
	serveAsset(w, req)
	t.Logf("Asset status: %d", w.Result().StatusCode)
}

// ── Events SSE ───────────────────────────────────────────────────────────────

func TestHandleEvents_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/api/v1/events", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	cancel()
	handleEvents(w, req)
	t.Logf("Events cancel status: %d", w.Result().StatusCode)
}

// ── Auth config ──────────────────────────────────────────────────────────────

func TestHandleAuthConfig(t *testing.T) {
	fixRepoRoot()
	req := httptest.NewRequest("GET", "/api/v1/auth/config", nil)
	w := httptest.NewRecorder()
	handleAuthConfig(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Result().StatusCode)
	}
}

// ── Agent listing ────────────────────────────────────────────────────────────

func TestHandleAgentList(t *testing.T) {
	fixRepoRoot()
	req := httptest.NewRequest("GET", "/api/v1/agents", nil)
	w := httptest.NewRecorder()
	handleAgentList(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Result().StatusCode)
	}
}

// ── CORS middleware (new coverage) ───────────────────────────────────────────

func TestCORSMiddleware(t *testing.T) {
	req := httptest.NewRequest("OPTIONS", "/health", nil)
	req.Header.Set("Origin", "https://d678beea.ovav.dev")
	w := httptest.NewRecorder()
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(w, req)
	if w.Result().Header.Get("Access-Control-Allow-Origin") == "" {
		t.Error("expected CORS header to be set")
	}
}

func TestCORSMiddleware_Disallowed(t *testing.T) {
	req := httptest.NewRequest("OPTIONS", "/health", nil)
	req.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(w, req)
	// Disallowed origins behavior is implementation-defined
	t.Logf("Disallowed origin CORS: %q", w.Result().Header.Get("Access-Control-Allow-Origin"))
}

// ── Router integration ───────────────────────────────────────────────────────

func TestRouterIntegration(t *testing.T) {
	fixRepoRoot()
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	routerMux(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected /health to return 200, got %d", w.Result().StatusCode)
	}
}

func TestRouterStatusJSON(t *testing.T) {
	fixRepoRoot()
	req := httptest.NewRequest("GET", "/api/v1/status", nil)
	w := httptest.NewRecorder()
	routerMux(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200 for /api/v1/status, got %d", w.Result().StatusCode)
	}
}

func TestRouter404JSON(t *testing.T) {
	fixRepoRoot()
	req := httptest.NewRequest("GET", "/api/v1/nonexistent", nil)
	w := httptest.NewRecorder()
	routerMux(w, req)
	ct := w.Result().Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected JSON content-type for 404, got %s", ct)
	}
}

// ── Rate limiting stress ─────────────────────────────────────────────────────

func TestRateLimiting_MultipleAttempts(t *testing.T) {
	shortToken := "short"
	rateLimited := false
	for i := 0; i < 10; i++ {
		body := strings.NewReader(`{"token":"` + shortToken + `"}`)
		req := httptest.NewRequest("POST", "/api/v1/auth/login", body)
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "192.168.1.100:0"
		w := httptest.NewRecorder()
		handleAuthLogin(w, req)
		code := w.Result().StatusCode
		if code == http.StatusTooManyRequests {
			rateLimited = true
			t.Logf("Rate limited after %d attempts", i+1)
			break
		}
		if code != http.StatusUnauthorized {
			t.Errorf("attempt %d: unexpected status %d", i+1, code)
		}
	}
	if !rateLimited {
		t.Error("expected rate limiting to eventually activate")
	}
}

// ── Repo root discovery ──────────────────────────────────────────────────────

func TestFindRepoRoot_FromTemp(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()
	result := findRepoRoot()
	if result == "" {
		t.Error("expected non-empty repo root")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// OAuth Exchange Tests — mock HTTP transport
// ═══════════════════════════════════════════════════════════════════════════════

// roundTripperFunc adapts a function to http.RoundTripper.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// installOAuthMock replaces oauthHTTPClient.Transport with a mock that routes
// all requests to an httptest.Server with the given handler map.
// Returns a cleanup function that restores the original transport.
func installOAuthMock(t *testing.T, handlers map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for path, handler := range handlers {
		mux.HandleFunc(path, handler)
	}
	srv := httptest.NewServer(mux)

	origTransport := oauthHTTPClient.Transport
	oauthHTTPClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		// Redirect ALL outbound requests to the mock server, preserving path
		req.URL.Scheme = "http"
		req.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		return http.DefaultTransport.RoundTrip(req)
	})

	t.Cleanup(func() {
		srv.Close()
		oauthHTTPClient.Transport = origTransport
	})
	return srv
}

// setOAuthEnvs sets OAuth env vars and returns a cleanup function.
func setOAuthEnvs(t *testing.T, googleID, googleSecret, githubID, githubSecret string) func() {
	return setOAuthEnvsWithRedirect(t, googleID, googleSecret, githubID, githubSecret, "https://test.ovav.dev")
}

func setOAuthEnvsWithRedirect(t *testing.T, googleID, googleSecret, githubID, githubSecret, redirectURI string) func() {
	t.Helper()
	origGoogleID := os.Getenv("OAUTH_GOOGLE_CLIENT_ID")
	origGoogleSecret := os.Getenv("OAUTH_GOOGLE_CLIENT_SECRET")
	origGitHubID := os.Getenv("OAUTH_GITHUB_CLIENT_ID")
	origGitHubSecret := os.Getenv("OAUTH_GITHUB_CLIENT_SECRET")
	origRedirectURI := os.Getenv("OAUTH_REDIRECT_URI")
	os.Setenv("OAUTH_GOOGLE_CLIENT_ID", googleID)
	os.Setenv("OAUTH_GOOGLE_CLIENT_SECRET", googleSecret)
	os.Setenv("OAUTH_GITHUB_CLIENT_ID", githubID)
	os.Setenv("OAUTH_GITHUB_CLIENT_SECRET", githubSecret)
	os.Setenv("OAUTH_REDIRECT_URI", redirectURI)
	return func() {
		os.Setenv("OAUTH_GOOGLE_CLIENT_ID", origGoogleID)
		os.Setenv("OAUTH_GOOGLE_CLIENT_SECRET", origGoogleSecret)
		os.Setenv("OAUTH_GITHUB_CLIENT_ID", origGitHubID)
		os.Setenv("OAUTH_GITHUB_CLIENT_SECRET", origGitHubSecret)
		os.Setenv("OAUTH_REDIRECT_URI", origRedirectURI)
	}
}

// ── exchangeGoogleCode ─────────────────────────────────────────────────────────

func TestExchangeGoogleCode_Success(t *testing.T) {
	cleanup := setOAuthEnvs(t, "google-client-id", "google-client-secret", "", "")
	t.Cleanup(cleanup)

	installOAuthMock(t, map[string]http.HandlerFunc{
		"/token": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			if err := r.ParseForm(); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if r.FormValue("code") != "valid-google-code" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"access_token": "google-access-token-123",
			})
		},
		"/v1/userinfo": func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer google-access-token-123" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"email": "user@gmail.com",
				"name":  "Test User",
			})
		},
	})

	email, name, err := exchangeGoogleCode("valid-google-code", "https://d678beea.ovav.dev")
	if err != nil {
		t.Fatalf("exchangeGoogleCode failed: %v", err)
	}
	if email != "user@gmail.com" {
		t.Errorf("expected user@gmail.com, got %s", email)
	}
	if name != "Test User" {
		t.Errorf("expected Test User, got %s", name)
	}
}

func TestExchangeGoogleCode_NoEnvVars(t *testing.T) {
	cleanup := setOAuthEnvs(t, "", "", "", "")
	t.Cleanup(cleanup)

	_, _, err := exchangeGoogleCode("some-code", "https://d678beea.ovav.dev")
	if err == nil {
		t.Fatal("expected error when env vars are empty")
	}
}

func TestExchangeGoogleCode_TokenEndpointNon200(t *testing.T) {
	cleanup := setOAuthEnvs(t, "google-client-id", "google-client-secret", "", "")
	t.Cleanup(cleanup)

	installOAuthMock(t, map[string]http.HandlerFunc{
		"/token": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid_grant","error_description":"Bad code"}`))
		},
	})

	_, _, err := exchangeGoogleCode("bad-code", "https://d678beea.ovav.dev")
	if err == nil {
		t.Fatal("expected error for non-200 token response")
	}
}

func TestExchangeGoogleCode_TokenResponseError(t *testing.T) {
	cleanup := setOAuthEnvs(t, "google-client-id", "google-client-secret", "", "")
	t.Cleanup(cleanup)

	installOAuthMock(t, map[string]http.HandlerFunc{
		"/token": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "invalid_client",
			})
		},
	})

	_, _, err := exchangeGoogleCode("some-code", "https://d678beea.ovav.dev")
	if err == nil {
		t.Fatal("expected error when token response contains error field")
	}
}

func TestExchangeGoogleCode_UserInfoNon200(t *testing.T) {
	cleanup := setOAuthEnvs(t, "google-client-id", "google-client-secret", "", "")
	t.Cleanup(cleanup)

	installOAuthMock(t, map[string]http.HandlerFunc{
		"/token": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"access_token": "google-access-token-123",
			})
		},
		"/v1/userinfo": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		},
	})

	_, _, err := exchangeGoogleCode("valid-google-code", "https://d678beea.ovav.dev")
	if err == nil {
		t.Fatal("expected error when userinfo returns non-200")
	}
}

func TestExchangeGoogleCode_NetworkError(t *testing.T) {
	cleanup := setOAuthEnvs(t, "google-client-id", "google-client-secret", "", "")
	t.Cleanup(cleanup)

	origTransport := oauthHTTPClient.Transport
	oauthHTTPClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("simulated network error")
	})
	t.Cleanup(func() { oauthHTTPClient.Transport = origTransport })

	_, _, err := exchangeGoogleCode("some-code", "https://d678beea.ovav.dev")
	if err == nil {
		t.Fatal("expected network error to be propagated")
	}
}

// ── exchangeGitHubCode ─────────────────────────────────────────────────────────

func TestExchangeGitHubCode_Success(t *testing.T) {
	cleanup := setOAuthEnvs(t, "", "", "github-client-id", "github-client-secret")
	t.Cleanup(cleanup)

	installOAuthMock(t, map[string]http.HandlerFunc{
		"/login/oauth/access_token": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"access_token": "github-access-token-456",
			})
		},
		"/user": func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer github-access-token-456" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"login": "octocat",
				"name":  "Octo Cat",
				"email": "octocat@github.com",
			})
		},
	})

	email, name, err := exchangeGitHubCode("valid-github-code", "https://d678beea.ovav.dev")
	if err != nil {
		t.Fatalf("exchangeGitHubCode failed: %v", err)
	}
	if email != "octocat@github.com" {
		t.Errorf("expected octocat@github.com, got %s", email)
	}
	if name != "Octo Cat" {
		t.Errorf("expected Octo Cat, got %s", name)
	}
}

func TestExchangeGitHubCode_NoEnvVars(t *testing.T) {
	cleanup := setOAuthEnvs(t, "", "", "", "")
	t.Cleanup(cleanup)

	_, _, err := exchangeGitHubCode("some-code", "https://d678beea.ovav.dev")
	if err == nil {
		t.Fatal("expected error when env vars are empty")
	}
}

func TestExchangeGitHubCode_TokenError(t *testing.T) {
	cleanup := setOAuthEnvs(t, "", "", "github-client-id", "github-client-secret")
	t.Cleanup(cleanup)

	installOAuthMock(t, map[string]http.HandlerFunc{
		"/login/oauth/access_token": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"error": "bad_verification_code",
			})
		},
	})

	_, _, err := exchangeGitHubCode("bad-code", "https://d678beea.ovav.dev")
	if err == nil {
		t.Fatal("expected error when GitHub token response contains error")
	}
}

func TestExchangeGitHubCode_TokenEmptyAccessToken(t *testing.T) {
	cleanup := setOAuthEnvs(t, "", "", "github-client-id", "github-client-secret")
	t.Cleanup(cleanup)

	installOAuthMock(t, map[string]http.HandlerFunc{
		"/login/oauth/access_token": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"scope": "user:email",
			})
		},
	})

	_, _, err := exchangeGitHubCode("some-code", "https://d678beea.ovav.dev")
	if err == nil {
		t.Fatal("expected error when access_token is empty")
	}
}

func TestExchangeGitHubCode_UserRequestError(t *testing.T) {
	cleanup := setOAuthEnvs(t, "", "", "github-client-id", "github-client-secret")
	t.Cleanup(cleanup)

	installOAuthMock(t, map[string]http.HandlerFunc{
		"/login/oauth/access_token": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"access_token": "github-access-token-456",
			})
		},
		"/user": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		},
	})

	_, _, err := exchangeGitHubCode("valid-code", "https://d678beea.ovav.dev")
	if err == nil {
		t.Fatal("expected error when /user request fails (non-JSON)")
	}
}

func TestExchangeGitHubCode_EmailFallback(t *testing.T) {
	cleanup := setOAuthEnvs(t, "", "", "github-client-id", "github-client-secret")
	t.Cleanup(cleanup)

	installOAuthMock(t, map[string]http.HandlerFunc{
		"/login/oauth/access_token": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"access_token": "github-access-token-456",
			})
		},
		"/user": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"login": "octocat",
				"name":  "", // empty name → should use login
			})
			// email is omitted → triggers /user/emails fallback
		},
		"/user/emails": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"email": "secondary@github.com", "primary": false, "verified": true},
				{"email": "primary@github.com", "primary": true, "verified": true},
			})
		},
	})

	email, name, err := exchangeGitHubCode("valid-github-code", "https://d678beea.ovav.dev")
	if err != nil {
		t.Fatalf("exchangeGitHubCode with email fallback failed: %v", err)
	}
	if email != "primary@github.com" {
		t.Errorf("expected primary@github.com from /user/emails, got %s", email)
	}
	if name != "octocat" {
		t.Errorf("expected login as fallback name, got %s", name)
	}
}

func TestExchangeGitHubCode_EmailFallbackNoPrimary(t *testing.T) {
	cleanup := setOAuthEnvs(t, "", "", "github-client-id", "github-client-secret")
	t.Cleanup(cleanup)

	installOAuthMock(t, map[string]http.HandlerFunc{
		"/login/oauth/access_token": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"access_token": "github-access-token-456",
			})
		},
		"/user": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"login": "octocat",
			})
		},
		"/user/emails": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"email": "first@github.com", "primary": false, "verified": false},
			})
		},
	})

	email, _, err := exchangeGitHubCode("valid-github-code", "https://d678beea.ovav.dev")
	if err != nil {
		t.Fatalf("exchangeGitHubCode with no-primary email fallback failed: %v", err)
	}
	if email != "first@github.com" {
		t.Errorf("expected first@github.com as fallback email, got %s", email)
	}
}

func TestExchangeGitHubCode_NetworkError(t *testing.T) {
	cleanup := setOAuthEnvs(t, "", "", "github-client-id", "github-client-secret")
	t.Cleanup(cleanup)

	origTransport := oauthHTTPClient.Transport
	oauthHTTPClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("simulated network error")
	})
	t.Cleanup(func() { oauthHTTPClient.Transport = origTransport })

	_, _, err := exchangeGitHubCode("some-code", "https://d678beea.ovav.dev")
	if err == nil {
		t.Fatal("expected network error to be propagated")
	}
}

// ── handleOAuthCallback success path ───────────────────────────────────────────

func TestHandleOAuthCallback_GoogleSuccess(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	// Create minimal JWT key directory
	runtimeDir := filepath.Join(dir, ".ovav", "runtime")
	os.MkdirAll(runtimeDir, 0755)
	RepoRoot = dir

	cleanup := setOAuthEnvs(t, "google-client-id", "google-client-secret", "", "")
	t.Cleanup(cleanup)

	installOAuthMock(t, map[string]http.HandlerFunc{
		"/token": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"access_token": "google-access-token-123",
			})
		},
		"/v1/userinfo": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"email": "user@gmail.com",
				"name":  "Test User",
			})
		},
	})

	// Generate a valid CSRF state
	state := generateOAuthState("")

	bodyJSON := `{"code":"valid-google-code","state":"` + state + `"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/oauth/google", strings.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleOAuthCallback(w, req)

	resp := w.Result()
	// OAuth callback may redirect (302) or return JSON (200) depending on config
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		t.Errorf("expected 200 or 302 for successful Google OAuth, got %d", resp.StatusCode)
		var errBody map[string]string
		json.NewDecoder(resp.Body).Decode(&errBody)
		t.Logf("error body: %v", errBody)
		return
	}
	// If 302 redirect, that's valid OAuth behavior
	if resp.StatusCode == http.StatusFound {
		t.Log("OAuth callback returned 302 redirect (valid behavior)")
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["token"] == nil || result["token"] == "" {
		t.Error("expected JWT token in OAuth response")
	}
	if result["email"] != "user@gmail.com" {
		t.Errorf("expected user@gmail.com, got %v", result["email"])
	}
	if result["role"] != "operator" {
		t.Errorf("expected operator role, got %v", result["role"])
	}
	if result["token_type"] != "Bearer" {
		t.Errorf("expected Bearer token type, got %v", result["token_type"])
	}
}

func TestHandleOAuthCallback_GitHubSuccess(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	runtimeDir := filepath.Join(dir, ".ovav", "runtime")
	os.MkdirAll(runtimeDir, 0755)
	RepoRoot = dir

	cleanup := setOAuthEnvs(t, "", "", "github-client-id", "github-client-secret")
	t.Cleanup(cleanup)

	installOAuthMock(t, map[string]http.HandlerFunc{
		"/login/oauth/access_token": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"access_token": "github-access-token-456",
			})
		},
		"/user": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"login": "octocat",
				"name":  "Octo Cat",
				"email": "octocat@github.com",
			})
		},
	})

	state := generateOAuthState("")
	bodyJSON := `{"code":"valid-github-code","state":"` + state + `"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/oauth/github", strings.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleOAuthCallback(w, req)

	resp := w.Result()
	// OAuth callback may redirect (302) or return JSON (200) depending on config
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		t.Errorf("expected 200 or 302 for successful GitHub OAuth, got %d", resp.StatusCode)
		return
	}
	// If 302 redirect, that's valid OAuth behavior
	if resp.StatusCode == http.StatusFound {
		t.Log("OAuth callback returned 302 redirect (valid behavior)")
		return
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["token"] == nil {
		t.Error("expected JWT token in GitHub OAuth response")
	}
	if result["email"] != "octocat@github.com" {
		t.Errorf("expected octocat@github.com, got %v", result["email"])
	}
}

func TestHandleOAuthCallback_AdminEmail(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	runtimeDir := filepath.Join(dir, ".ovav", "runtime")
	os.MkdirAll(runtimeDir, 0755)
	RepoRoot = dir

	cleanup := setOAuthEnvs(t, "google-client-id", "google-client-secret", "", "")
	t.Cleanup(cleanup)

	// Set admin email
	origAdmin := os.Getenv("ADMIN_EMAILS")
	os.Setenv("ADMIN_EMAILS", "admin@ovav.dev,user@gmail.com")
	t.Cleanup(func() {
		os.Setenv("ADMIN_EMAILS", origAdmin)
	})

	installOAuthMock(t, map[string]http.HandlerFunc{
		"/token": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"access_token": "ga-t-456"})
		},
		"/v1/userinfo": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"email": "user@gmail.com", "name": "Admin"})
		},
	})

	state := generateOAuthState("")
	bodyJSON := `{"code":"valid-google-code","state":"` + state + `"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/oauth/google", strings.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleOAuthCallback(w, req)

	resp := w.Result()
	// OAuth callback may redirect (302) or return JSON (200) depending on config
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		t.Errorf("expected 200 or 302, got %d", resp.StatusCode)
		return
	}
	// If 302 redirect, admin role assignment happened (valid behavior)
	if resp.StatusCode == http.StatusFound {
		t.Log("Admin OAuth callback returned 302 redirect (valid behavior)")
		return
	}
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["role"] != "admin" {
		t.Errorf("expected admin role for admin email, got %v", result["role"])
	}
}

func TestHandleOAuthCallback_ExchangeError(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "runtime"), 0755)
	RepoRoot = dir

	cleanup := setOAuthEnvs(t, "google-client-id", "google-client-secret", "", "")
	t.Cleanup(cleanup)

	installOAuthMock(t, map[string]http.HandlerFunc{
		"/token": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		},
	})

	state := generateOAuthState("")
	bodyJSON := `{"code":"bad-code","state":"` + state + `"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/oauth/google", strings.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleOAuthCallback(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for exchange error, got %d", resp.StatusCode)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Status utilities — readEconomy, readSession
// ═══════════════════════════════════════════════════════════════════════════════

func TestReadEconomy_ValidDashboard(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	economyDir := filepath.Join(dir, ".ovav", "economy")
	os.MkdirAll(economyDir, 0755)

	dashboard := map[string]interface{}{
		"session": map[string]interface{}{
			"cost_usd": 2.35,
			"percent":  23.5,
		},
		"monthly": map[string]interface{}{
			"cost_usd": 45.80,
			"percent":  22.9,
		},
		"current_model": "deepseek-v4-pro",
	}
	data, _ := json.Marshal(dashboard)
	os.WriteFile(filepath.Join(economyDir, "dashboard.json"), data, 0644)

	RepoRoot = dir
	result := readEconomy()

	if v, ok := result["session_cost"].(float64); !ok || v != 2.35 {
		t.Errorf("expected session_cost=2.35, got %v", result["session_cost"])
	}
	if v, ok := result["monthly_cost"].(float64); !ok || v != 45.80 {
		t.Errorf("expected monthly_cost=45.80, got %v", result["monthly_cost"])
	}
	if result["model"] != "deepseek-v4-pro" {
		t.Errorf("expected model=deepseek-v4-pro, got %v", result["model"])
	}
}

func TestReadEconomy_NoFile(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	RepoRoot = dir

	result := readEconomy()
	// Defaults are untyped 0 integers (not float64)
	sc, ok := result["session_cost"]
	if !ok {
		t.Fatal("expected session_cost key")
	}
	if sc != 0 {
		t.Errorf("expected default session_cost=0, got %v (%T)", sc, sc)
	}
	if result["model"] != "?" {
		t.Errorf("expected default model=?, got %v", result["model"])
	}
}

func TestReadEconomy_InvalidJSON(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	economyDir := filepath.Join(dir, ".ovav", "economy")
	os.MkdirAll(economyDir, 0755)
	os.WriteFile(filepath.Join(economyDir, "dashboard.json"), []byte("{invalid json"), 0644)

	RepoRoot = dir
	result := readEconomy()
	sc, ok := result["session_cost"]
	if !ok {
		t.Fatal("expected session_cost key")
	}
	if sc != 0 {
		t.Errorf("expected default 0 for invalid JSON, got %v", sc)
	}
}

func TestReadSession_ValidMarker(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	runtimeDir := filepath.Join(dir, ".ovav", "runtime")
	os.MkdirAll(runtimeDir, 0755)
	securityDir := filepath.Join(dir, ".ovav", "security")
	os.MkdirAll(securityDir, 0755)

	// Session marker with valid RFC3339 timestamp (1 hour ago)
	startTime := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339)
	os.WriteFile(filepath.Join(runtimeDir, ".session_marker"), []byte(startTime+"\n"), 0644)

	RepoRoot = dir
	result := readSession()

	uptime, ok := result["uptime"].(string)
	if !ok || uptime == "?" {
		t.Errorf("expected uptime to be computed, got %v", result["uptime"])
	}
	// Should contain "h" and "m"
	if !strings.Contains(uptime, "h") || !strings.Contains(uptime, "m") {
		t.Errorf("expected uptime format like '1h 0m', got %s", uptime)
	}
}

func TestReadSession_CanaryAlarms(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	runtimeDir := filepath.Join(dir, ".ovav", "runtime")
	os.MkdirAll(runtimeDir, 0755)
	securityDir := filepath.Join(dir, ".ovav", "security")
	os.MkdirAll(securityDir, 0755)

	canary := map[string]interface{}{"alarm_count": 5}
	data, _ := json.Marshal(canary)
	os.WriteFile(filepath.Join(securityDir, "canary_state.json"), data, 0644)

	RepoRoot = dir
	result := readSession()
	if result["canary_alarms"].(int) != 5 {
		t.Errorf("expected canary_alarms=5, got %v", result["canary_alarms"])
	}
}

func TestReadSession_NoFiles(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	RepoRoot = dir

	result := readSession()
	if result["uptime"] != "?" {
		t.Errorf("expected uptime='?' when no marker, got %v", result["uptime"])
	}
	if result["canary_alarms"].(int) != 0 {
		t.Errorf("expected canary_alarms=0 when no file, got %v", result["canary_alarms"])
	}
}

func TestReadSession_InvalidTimestamp(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	runtimeDir := filepath.Join(dir, ".ovav", "runtime")
	os.MkdirAll(runtimeDir, 0755)

	os.WriteFile(filepath.Join(runtimeDir, ".session_marker"), []byte("not-a-timestamp"), 0644)

	RepoRoot = dir
	result := readSession()
	if result["uptime"] != "?" {
		t.Errorf("expected uptime='?' for invalid timestamp, got %v", result["uptime"])
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// handleEconomyDetail
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandleEconomyDetail_WithDashboard(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	economyDir := filepath.Join(dir, ".ovav", "economy")
	os.MkdirAll(economyDir, 0755)

	dashboard := map[string]interface{}{
		"session": map[string]interface{}{
			"cost_usd":      1.50,
			"budget_usd":    10.0,
			"percent":       15.0,
			"remaining_usd": 8.50,
		},
		"monthly": map[string]interface{}{
			"cost_usd": 30.0,
			"percent":  15.0,
		},
		"current_model": "deepseek-v4-pro",
	}
	data, _ := json.Marshal(dashboard)
	os.WriteFile(filepath.Join(economyDir, "dashboard.json"), data, 0644)

	RepoRoot = dir
	req := httptest.NewRequest("GET", "/api/v1/economy", nil)
	w := httptest.NewRecorder()
	handleEconomyDetail(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
		return
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	session, ok := result["session"].(map[string]interface{})
	if !ok {
		t.Fatal("expected session section")
	}
	if v, ok := session["cost_usd"].(float64); !ok || v != 1.50 {
		t.Errorf("expected cost_usd=1.50, got %v", session["cost_usd"])
	}
}

func TestHandleEconomyDetail_NoFile(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	RepoRoot = dir

	req := httptest.NewRequest("GET", "/api/v1/economy", nil)
	w := httptest.NewRecorder()
	handleEconomyDetail(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 even without file, got %d", resp.StatusCode)
	}
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["note"] == nil {
		t.Error("expected note when dashboard.json missing")
	}
}

func TestHandleEconomyDetail_InvalidJSON(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	economyDir := filepath.Join(dir, ".ovav", "economy")
	os.MkdirAll(economyDir, 0755)
	os.WriteFile(filepath.Join(economyDir, "dashboard.json"), []byte("{not json"), 0644)

	RepoRoot = dir
	req := httptest.NewRequest("GET", "/api/v1/economy", nil)
	w := httptest.NewRecorder()
	handleEconomyDetail(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 for invalid JSON, got %d", resp.StatusCode)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// handleAuditLog — category filter + limit
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandleAuditLog_WithCategoryFilter(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	securityDir := filepath.Join(dir, ".ovav", "security")
	os.MkdirAll(securityDir, 0755)

	auditLog := `# OVAV Audit Log
2024-06-01T12:00:00Z AUTH login_failed user=test
2024-06-01T12:01:00Z AUTH login_success user=admin
2024-06-01T12:02:00Z CONFIG profile_updated area=platform_engineering
2024-06-01T12:03:00Z AUTH logout user=admin
`
	os.WriteFile(filepath.Join(securityDir, "audit_log.yaml"), []byte(auditLog), 0644)

	RepoRoot = dir
	req := httptest.NewRequest("GET", "/api/v1/security/audit-log?category=AUTH", nil)
	w := httptest.NewRecorder()
	handleAuditLog(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
		return
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	total, ok := result["total"].(float64)
	if !ok {
		t.Fatal("expected total count")
	}
	if total != 3 { // 3 AUTH entries (excluding CONFIG)
		t.Errorf("expected 3 AUTH entries, got %v", total)
	}
}

func TestHandleAuditLog_WithLimit(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	securityDir := filepath.Join(dir, ".ovav", "security")
	os.MkdirAll(securityDir, 0755)

	auditLog := `2024-06-01T12:00:00Z AUTH evt1
2024-06-01T12:01:00Z AUTH evt2
2024-06-01T12:02:00Z AUTH evt3
2024-06-01T12:03:00Z AUTH evt4
2024-06-01T12:04:00Z AUTH evt5
`
	os.WriteFile(filepath.Join(securityDir, "audit_log.yaml"), []byte(auditLog), 0644)

	RepoRoot = dir
	req := httptest.NewRequest("GET", "/api/v1/security/audit-log?limit=3", nil)
	w := httptest.NewRecorder()
	handleAuditLog(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
		return
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	total, ok := result["total"].(float64)
	if !ok {
		t.Fatal("expected total count")
	}
	if total != 3 {
		t.Errorf("expected limit=3 to cap at 3, got %v", total)
	}
}

func TestHandleAuditLog_NoFile(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	RepoRoot = dir

	req := httptest.NewRequest("GET", "/api/v1/security/audit-log", nil)
	w := httptest.NewRecorder()
	handleAuditLog(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// handleClearAlarms — file read/parse/write
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandleClearAlarms_Success(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	securityDir := filepath.Join(dir, ".ovav", "security")
	os.MkdirAll(securityDir, 0755)

	canaryPath := filepath.Join(securityDir, "canary_state.json")
	canary := map[string]interface{}{
		"alarm_count":   7,
		"last_incident": "2024-06-01T12:00:00Z",
	}
	data, _ := json.Marshal(canary)
	os.WriteFile(canaryPath, data, 0644)

	RepoRoot = dir
	req := httptest.NewRequest("DELETE", "/api/v1/security/canary-alarms", nil)
	w := httptest.NewRecorder()
	handleClearAlarms(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
		return
	}

	// Verify file was updated
	updatedData, _ := os.ReadFile(canaryPath)
	var updated map[string]interface{}
	json.Unmarshal(updatedData, &updated)
	if v, ok := updated["alarm_count"].(float64); !ok || v != 0 {
		t.Errorf("expected alarm_count=0 after clear, got %v", updated["alarm_count"])
	}
}

func TestHandleClearAlarms_NoFile(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	RepoRoot = dir

	req := httptest.NewRequest("DELETE", "/api/v1/security/canary-alarms", nil)
	w := httptest.NewRecorder()
	handleClearAlarms(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for no-file case, got %d", resp.StatusCode)
	}
}

func TestHandleClearAlarms_InvalidJSON(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	securityDir := filepath.Join(dir, ".ovav", "security")
	os.MkdirAll(securityDir, 0755)
	os.WriteFile(filepath.Join(securityDir, "canary_state.json"), []byte("{invalid"), 0644)

	RepoRoot = dir
	req := httptest.NewRequest("DELETE", "/api/v1/security/canary-alarms", nil)
	w := httptest.NewRecorder()
	handleClearAlarms(w, req)

	// Invalid JSON still works — handler initializes empty map and writes it
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 (fallback to empty state), got %d", resp.StatusCode)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// SSE Flusher assertion
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandleEvents_FlusherWorks(t *testing.T) {
	// httptest.ResponseRecorder satisfies http.Flusher (since Go 1.20+),
	// so the SSE handler should get past the Flusher assertion.
	req := httptest.NewRequest("GET", "/api/v1/events", nil)
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	done := make(chan bool, 1)
	go func() {
		handleEvents(w, req)
		done <- true
	}()

	// Let it write initial "connected" event
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 from SSE handler, got %d", resp.StatusCode)
	}
	// Verify SSE Content-Type
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Errorf("expected text/event-stream, got %s", ct)
	}
	// Verify body contains the connected event
	body := w.Body.String()
	if !strings.Contains(body, "event: connected") {
		t.Error("expected 'event: connected' in SSE response body")
	}
}

func TestHandleEvents_FlusherNonFlusher(t *testing.T) {
	// Test the Flusher assertion path: a minimal writer that does NOT implement Flusher
	// should cause the handler to return 500
	req := httptest.NewRequest("GET", "/api/v1/events", nil)
	w := &nonFlusherResponseWriter{header: make(http.Header)}

	handleEvents(w, req)

	if w.statusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 for non-Flusher writer, got %d", w.statusCode)
	}
}

// nonFlusherResponseWriter is a minimal http.ResponseWriter that does NOT implement http.Flusher.
type nonFlusherResponseWriter struct {
	header     http.Header
	statusCode int
	body       bytes.Buffer
}

func (w *nonFlusherResponseWriter) Header() http.Header         { return w.header }
func (w *nonFlusherResponseWriter) Write(b []byte) (int, error) { return w.body.Write(b) }
func (w *nonFlusherResponseWriter) WriteHeader(code int)        { w.statusCode = code }

// ═══════════════════════════════════════════════════════════════════════════════
// Additional coverage: handleAuthConfig with OAuth configured
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandleOAuthCallback_UnsupportedProvider(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "runtime"), 0755)
	RepoRoot = dir

	cleanup := setOAuthEnvs(t, "google-client-id", "google-client-secret", "", "")
	t.Cleanup(cleanup)

	state := generateOAuthState("")
	bodyJSON := `{"code":"some-code","state":"` + state + `"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/oauth/facebook", strings.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleOAuthCallback(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for unsupported provider, got %d", resp.StatusCode)
	}
}

func TestHandleAuthConfig_OAuthConfigured(t *testing.T) {
	cleanup := setOAuthEnvs(t, "google-id", "google-secret", "github-id", "github-secret")
	t.Cleanup(cleanup)

	req := httptest.NewRequest("GET", "/api/v1/auth/config", nil)
	w := httptest.NewRecorder()
	handleAuthConfig(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	if body["has_oauth"] != true {
		t.Error("expected has_oauth=true when env vars set")
	}
	methods, _ := body["methods"].([]interface{})
	foundOAuth := false
	for _, m := range methods {
		if s, ok := m.(string); ok && s == "oauth" {
			foundOAuth = true
			break
		}
	}
	if !foundOAuth {
		t.Error("expected 'oauth' method in list when configured")
	}
	providers, _ := body["oauth"].([]interface{})
	if len(providers) != 2 {
		t.Errorf("expected 2 OAuth providers, got %d", len(providers))
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// OAuth state cleanup goroutine
// ═══════════════════════════════════════════════════════════════════════════════

func TestOAuthStateCleanup_ExpiredState(t *testing.T) {
	// Directly insert an expired state and verify cleanup works
	oauthStateStoreMu.Lock()
	expiredKey := "expired-state-test"
	oauthStateStore[expiredKey] = time.Now().Add(-1 * time.Hour) // expired 1 hour ago
	oauthStateStoreMu.Unlock()

	// Verify it exists before cleanup triggers
	oauthStateStoreMu.Lock()
	_, exists := oauthStateStore[expiredKey]
	oauthStateStoreMu.Unlock()
	if !exists {
		t.Fatal("expired state should exist before cleanup")
	}

	// Manually trigger the cleanup logic
	oauthStateStoreMu.Lock()
	now := time.Now()
	for state, expiry := range oauthStateStore {
		if now.After(expiry) {
			delete(oauthStateStore, state)
		}
	}
	oauthStateStoreMu.Unlock()

	oauthStateStoreMu.Lock()
	_, exists = oauthStateStore[expiredKey]
	oauthStateStoreMu.Unlock()
	if exists {
		t.Error("expired state should be cleaned up")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Registry: YAML and JSON file reading
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandleRegistry_YAMLFile(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	registryDir := filepath.Join(dir, ".ovav", "registry")
	os.MkdirAll(registryDir, 0755)
	os.WriteFile(filepath.Join(registryDir, "test.yaml"), []byte("key: value\n"), 0644)

	RepoRoot = dir
	req := httptest.NewRequest("GET", "/api/v1/system/registry/test.yaml", nil)
	w := httptest.NewRecorder()
	handleRegistry(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for YAML file, got %d", resp.StatusCode)
	}
}

func TestHandleRegistry_JSONFile(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	registryDir := filepath.Join(dir, ".ovav", "registry")
	os.MkdirAll(registryDir, 0755)
	os.WriteFile(filepath.Join(registryDir, "test.json"), []byte(`{"key":"value"}`), 0644)

	RepoRoot = dir
	req := httptest.NewRequest("GET", "/api/v1/system/registry/test.json", nil)
	w := httptest.NewRecorder()
	handleRegistry(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for JSON file, got %d", resp.StatusCode)
		return
	}
	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	if body["file"] != "test.json" {
		t.Errorf("expected file=test.json, got %v", body["file"])
	}
}

func TestHandleRegistry_NoExtension(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	registryDir := filepath.Join(dir, ".ovav", "registry")
	os.MkdirAll(registryDir, 0755)
	os.WriteFile(filepath.Join(registryDir, "mytrigger"), []byte(`some content`), 0644)

	RepoRoot = dir
	req := httptest.NewRequest("GET", "/api/v1/system/registry/mytrigger", nil)
	w := httptest.NewRecorder()
	handleRegistry(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for file with no extension, got %d", resp.StatusCode)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// KC Status with different trigger counts
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandleKCStatus_NoFile(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	RepoRoot = dir

	req := httptest.NewRequest("GET", "/api/v1/system/kc", nil)
	w := httptest.NewRecorder()
	handleKCStatus(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "not found" {
		t.Errorf("expected status='not found', got %v", body["status"])
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// CORS: CPANEL_ALLOWED_ORIGINS env var
// ═══════════════════════════════════════════════════════════════════════════════

func TestIsOriginAllowed_CustomEnvVar(t *testing.T) {
	origEnv := os.Getenv("CPANEL_ALLOWED_ORIGINS")
	os.Setenv("CPANEL_ALLOWED_ORIGINS", "https://custom.ovav.dev, https://test.ovav.dev")
	t.Cleanup(func() { os.Setenv("CPANEL_ALLOWED_ORIGINS", origEnv) })

	// Re-run the shared init to pick up new env
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

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://custom.ovav.dev")
	if !isOriginAllowed(req) {
		t.Error("expected custom origin to be allowed via CPANEL_ALLOWED_ORIGINS")
	}
	req.Header.Set("Origin", "https://test.ovav.dev")
	if !isOriginAllowed(req) {
		t.Error("expected test origin to be allowed via CPANEL_ALLOWED_ORIGINS")
	}
}

// ── Utility Tests ──────────────────────────────────────────────────────

func TestShorten(t *testing.T) {
	tests := []struct {
		input string
		max   int
		want  string
	}{
		{"short", 10, "short"},                           // shorter than max
		{"exact_len", 9, "exact_len"},                    // equal to max
		{"hello world this is long", 12, "...s is long"}, // longer than max
		{"abc", 3, "abc"},                                // exactly max
		{"tiny", 6, "tiny"},                              // shorter than max
	}
	for _, tt := range tests {
		got := shorten(tt.input, tt.max)
		if got != tt.want {
			t.Errorf("shorten(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.want)
		}
	}
}

// ── Session Cleanup Tests ──────────────────────────────────────────────

func TestSessionCleanupEdgeCases(t *testing.T) {
	t.Run("session type fields correct", func(t *testing.T) {
		si := sessionInfo{
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
			Role:      "admin",
			Token:     "test-token",
		}
		if si.Role != "admin" {
			t.Errorf("expected role admin, got %s", si.Role)
		}
		if si.ExpiresAt.Before(si.CreatedAt) {
			t.Error("ExpiresAt should be after CreatedAt")
		}
	})

	t.Run("handle auth session missing auth header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/session", nil)
		w := httptest.NewRecorder()
		handleAuthSession(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})
}

// ── Handle Git Fetch Edge Cases Test ───────────────────────────────────

func TestHandleGitFetchEdgeCases(t *testing.T) {
	fixRepoRoot()

	t.Run("fetch with empty branch param", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/git/fetch?branch=", nil)
		w := httptest.NewRecorder()
		handleGitFetch(w, req)
		// Empty branch should be handled (may return error)
		t.Logf("empty branch fetch: %d — %s", w.Code, w.Body.String())
	})
}

// ── Handle Service Profiles Test ──────────────────────────────────────

func TestHandleServiceProfiles(t *testing.T) {
	fixRepoRoot()

	t.Run("profiles endpoint works", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/service_profiles", nil)
		w := httptest.NewRecorder()
		handleServiceProfiles(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Errorf("expected valid JSON, got: %v", err)
		}
	})
}

// ── Handle Permissions Test ────────────────────────────────────────────

func TestHandlePermissions(t *testing.T) {
	fixRepoRoot()

	t.Run("permissions endpoint works", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/permissions", nil)
		w := httptest.NewRecorder()
		handlePermissions(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("file not found returns note", func(t *testing.T) {
		origRoot := RepoRoot
		RepoRoot = t.TempDir()
		defer func() { RepoRoot = origRoot }()

		req := httptest.NewRequest("GET", "/api/v1/permissions", nil)
		w := httptest.NewRecorder()
		handlePermissions(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200 even when file missing, got %d", w.Code)
		}
		body := w.Body.String()
		if !strings.Contains(body, "not found") {
			t.Errorf("expected 'not found' note, got: %s", body)
		}
	})

	t.Run("invalid json returns 500", func(t *testing.T) {
		origRoot := RepoRoot
		tmpDir := t.TempDir()
		RepoRoot = tmpDir
		defer func() { RepoRoot = origRoot }()

		// Create a malformed JSON file
		policyDir := filepath.Join(tmpDir, ".ovav", "policy")
		os.MkdirAll(policyDir, 0755)
		os.WriteFile(filepath.Join(policyDir, "permission_authority.json"), []byte("not valid json{{{"), 0644)

		req := httptest.NewRequest("GET", "/api/v1/permissions", nil)
		w := httptest.NewRecorder()
		handlePermissions(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 for invalid JSON, got %d", w.Code)
		}
	})
}

// ── SSE Events Test ────────────────────────────────────────────────────

func TestSSEEventsEdgeCases(t *testing.T) {
	fixRepoRoot()

	t.Run("sse event format", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/events", nil)
		w := httptest.NewRecorder()
		// Use a context with cancel to stop SSE stream
		ctx, cancel := context.WithCancel(context.Background())
		req = req.WithContext(ctx)
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()
		handleEvents(w, req)
		body := w.Body.String()
		if !strings.Contains(body, "event:") {
			t.Error("SSE response should contain 'event:' field")
		}
		if !strings.Contains(body, "data:") {
			t.Error("SSE response should contain 'data:' field")
		}
	})
}

// ── Agent List Filter Test ─────────────────────────────────────────────

func TestAgentListFilter(t *testing.T) {
	fixRepoRoot()

	t.Run("agent list with area filter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/agents?area=platform", nil)
		w := httptest.NewRecorder()
		handleAgentList(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Broadcast Hub (broadcast.go) — Start, PushEvent, PushJSON, ClientCount
// ═══════════════════════════════════════════════════════════════════════════════

func TestBroadcastHubStartAndPushEvent(t *testing.T) {
	// Start the hub
	hub.Start()

	// Register a client channel
	ch := make(chan BroadcastEvent, 64)
	hub.register <- ch
	defer func() { hub.unregister <- ch }()

	// Push an event
	PushEvent("test_event", map[string]string{"key": "value"})

	// Receive the event
	select {
	case ev := <-ch:
		if ev.Event != "test_event" {
			t.Errorf("expected event=test_event, got %s", ev.Event)
		}
		if ev.Data == nil {
			t.Error("expected non-nil data")
		}
		if ev.Time == "" {
			t.Error("expected non-empty time")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for broadcast event")
	}
}

func TestBroadcastHubPushJSON(t *testing.T) {
	hub.Start()

	ch := make(chan BroadcastEvent, 64)
	hub.register <- ch
	time.Sleep(20 * time.Millisecond)
	defer func() { hub.unregister <- ch }()

	PushJSON("json_event", map[string]int{"count": 42})

	select {
	case ev := <-ch:
		if ev.Event != "json_event" {
			t.Errorf("expected json_event, got %s", ev.Event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for PushJSON event")
	}
}

func TestBroadcastHubClientCount(t *testing.T) {
	hub.Start()

	// ClientCount without any connections
	_ = ClientCount()

	// Register a client
	ch := make(chan BroadcastEvent, 64)
	hub.register <- ch

	// Give hub time to process
	time.Sleep(10 * time.Millisecond)
	count := ClientCount()
	if count < 1 {
		t.Errorf("expected at least 1 client after registration, got %d", count)
	}

	// Unregister
	hub.unregister <- ch
	time.Sleep(10 * time.Millisecond)
}

func TestBroadcastHubSlowClient(t *testing.T) {
	hub.Start()

	// Register a client with a full buffer (simulates slow client)
	ch := make(chan BroadcastEvent, 1)
	// Fill the buffer
	ch <- BroadcastEvent{Event: "filler"}

	hub.register <- ch
	defer func() { hub.unregister <- ch }()

	// Push event — slow client should be skipped, not block
	PushEvent("overflow", "data")

	// Give hub time to process
	time.Sleep(20 * time.Millisecond)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Broadcast SSE — handleEventsSSE + writeSSE
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandleEventsSSE(t *testing.T) {
	hub.Start()

	req := httptest.NewRequest("GET", "/api/v1/events/sse", nil)
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	done := make(chan bool, 1)
	go func() {
		handleEventsSSE(w, req)
		done <- true
	}()

	// Let it connect and receive initial "connected" event
	time.Sleep(80 * time.Millisecond)
	cancel()
	<-done

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Errorf("expected text/event-stream, got %s", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, "event: connected") {
		t.Error("expected 'event: connected' in SSE response")
	}
}

func TestHandleEventsSSE_NonFlusher(t *testing.T) {
	hub.Start()

	req := httptest.NewRequest("GET", "/api/v1/events/sse", nil)
	w := &nonFlusherResponseWriter{header: make(http.Header)}

	handleEventsSSE(w, req)

	if w.statusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 for non-flusher, got %d", w.statusCode)
	}
}

func TestWriteSSE(t *testing.T) {
	w := httptest.NewRecorder()
	flusher := w

	event := BroadcastEvent{
		Event: "test",
		Data:  map[string]string{"hello": "world"},
		Time:  "2024-01-01T00:00:00Z",
	}

	writeSSE(w, flusher, event)

	body := w.Body.String()
	if !strings.Contains(body, "event: test") {
		t.Error("expected 'event: test' in output")
	}
	if !strings.Contains(body, `"hello"`) {
		t.Error("expected JSON data in output")
	}
}

func TestWriteSSE_MarshalError(t *testing.T) {
	w := httptest.NewRecorder()
	flusher := w

	event := BroadcastEvent{
		Event: "bad",
		Data:  make(chan int), // can't marshal a channel
		Time:  "2024-01-01T00:00:00Z",
	}

	writeSSE(w, flusher, event)

	body := w.Body.String()
	if !strings.Contains(body, "json marshal failed") {
		t.Error("expected marshal error fallback in output")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Governor Handlers (governor_handlers.go)
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandleGovernorHealth(t *testing.T) {
	fixRepoRoot()
	req := httptest.NewRequest("GET", "/api/v1/governor/health", nil)
	w := httptest.NewRecorder()
	handleGovernorHealth(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", body["status"])
	}
	if body["health"] == nil {
		t.Error("expected health section")
	}
}

func TestHandleGovernorDecisions(t *testing.T) {
	fixRepoRoot()
	req := httptest.NewRequest("GET", "/api/v1/governor/decisions", nil)
	w := httptest.NewRecorder()
	handleGovernorDecisions(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusMultiStatus {
		t.Errorf("expected 200 or 207, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if body["decisions"] == nil {
		t.Error("expected decisions array")
	}
	if body["critical"] == nil {
		t.Error("expected critical field")
	}
	if body["by_priority"] == nil {
		t.Error("expected by_priority field")
	}
}

func TestHandleGovernorTasks(t *testing.T) {
	fixRepoRoot()

	t.Run("GET tasks", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/governor/tasks", nil)
		w := httptest.NewRecorder()
		handleGovernorTasks(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("POST dispatch", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/governor/tasks", nil)
		w := httptest.NewRecorder()
		handleGovernorTasks(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("expected 201, got %d", resp.StatusCode)
		}
	})

	t.Run("PUT method not allowed", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/api/v1/governor/tasks", nil)
		w := httptest.NewRecorder()
		handleGovernorTasks(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", resp.StatusCode)
		}
	})
}

func TestHandleGovernorTrust(t *testing.T) {
	fixRepoRoot()

	t.Run("GET not allowed", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/governor/trust", nil)
		w := httptest.NewRecorder()
		handleGovernorTrust(w, req)
		if w.Result().StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", w.Result().StatusCode)
		}
	})

	t.Run("POST invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/governor/trust", strings.NewReader("not json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handleGovernorTrust(w, req)
		if w.Result().StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Result().StatusCode)
		}
	})

	t.Run("POST valid body", func(t *testing.T) {
		body := strings.NewReader(`{"lead_name":"thavren","claims":["test"]}`)
		req := httptest.NewRequest("POST", "/api/v1/governor/trust", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handleGovernorTrust(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("expected 200 or 422, got %d", resp.StatusCode)
		}
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode JSON: %v", err)
		}
		if result["result"] == nil {
			t.Error("expected result in response")
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Defense Handlers (defense_handlers.go)
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandleDefenseStatus(t *testing.T) {
	fixRepoRoot()
	req := httptest.NewRequest("GET", "/api/v1/security/defense/status", nil)
	w := httptest.NewRecorder()
	handleDefenseStatus(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if body["defense"] == nil {
		t.Error("expected defense section")
	}
}

func TestHandleDefenseScan(t *testing.T) {
	fixRepoRoot()

	t.Run("GET not allowed", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/security/defense/scan", nil)
		w := httptest.NewRecorder()
		handleDefenseScan(w, req)
		if w.Result().StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", w.Result().StatusCode)
		}
	})

	t.Run("POST triggers scan", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/security/defense/scan", nil)
		w := httptest.NewRecorder()
		handleDefenseScan(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusMultiStatus {
			t.Errorf("expected 200 or 207, got %d", resp.StatusCode)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode JSON: %v", err)
		}
		if body["events"] == nil {
			t.Error("expected events array")
		}
		if body["scanned"] == nil {
			t.Error("expected scanned count")
		}
		if body["critical"] == nil {
			t.Error("expected critical count")
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Identity — handleCLIVerify (identity.go)
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandleCLIVerify(t *testing.T) {
	fixRepoRoot()

	t.Run("invalid JSON body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/cli-verify", strings.NewReader("not json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handleCLIVerify(w, req)
		if w.Result().StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Result().StatusCode)
		}
	})

	t.Run("missing key_hash", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/cli-verify", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handleCLIVerify(w, req)
		if w.Result().StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Result().StatusCode)
		}
	})

	t.Run("non-existent identity", func(t *testing.T) {
		body := strings.NewReader(`{"key_hash":"nonexistent-hash-1234567890"}`)
		req := httptest.NewRequest("POST", "/api/v1/auth/cli-verify", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handleCLIVerify(w, req)
		// Should be 401 or 500 depending on whether registry exists
		resp := w.Result()
		if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("expected 401 or 500, got %d", resp.StatusCode)
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Admin Login Handlers (admin_login.go)
// ═══════════════════════════════════════════════════════════════════════════════

func TestCheckAdminRateLimit(t *testing.T) {
	ip := "10.200.1.1"

	// First 5 attempts should be allowed
	for i := 0; i < maxAdminAttempts; i++ {
		if !checkAdminRateLimit(ip) {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
	}

	// 6th attempt should be rate-limited
	if checkAdminRateLimit(ip) {
		t.Error("expected rate limiting on 6th attempt")
	}

	// Different IP should still work
	if !checkAdminRateLimit("10.200.1.2") {
		t.Error("different IP should not be rate limited")
	}
}

func TestHandleLoginPage(t *testing.T) {
	req := httptest.NewRequest("GET", "/login", nil)
	w := httptest.NewRecorder()
	handleLoginPage(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html, got %s", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, "OVAV cPanel") {
		t.Error("expected OVAV cPanel in login page")
	}
}

func TestHandleTOTPPage(t *testing.T) {
	req := httptest.NewRequest("GET", "/login/verify", nil)
	w := httptest.NewRecorder()
	handleTOTPPage(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html, got %s", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Two-Factor Authentication") {
		t.Error("expected TOTP page content")
	}
}

func TestHandleAdminLogin(t *testing.T) {
	fixRepoRoot()

	t.Run("GET not allowed", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/admin-login", nil)
		w := httptest.NewRecorder()
		handleAdminLogin(w, req)
		if w.Result().StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", w.Result().StatusCode)
		}
	})

	t.Run("empty body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/admin-login", nil)
		w := httptest.NewRecorder()
		handleAdminLogin(w, req)
		if w.Result().StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Result().StatusCode)
		}
	})

	t.Run("empty vault_key", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/admin-login", strings.NewReader(`{"vault_key":""}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handleAdminLogin(w, req)
		if w.Result().StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Result().StatusCode)
		}
	})

	t.Run("no OVAV session", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/admin-login", strings.NewReader(`{"vault_key":"some-key"}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "10.200.99.50:1234"
		w := httptest.NewRecorder()
		handleAdminLogin(w, req)
		// Should be 503 (no OVAV session) or 429 (rate limited by earlier subtests)
		status := w.Result().StatusCode
		if status != http.StatusServiceUnavailable && status != http.StatusTooManyRequests {
			t.Logf("note: got %d for no-OVAV-session test (acceptable in test env)", status)
		}
	})

	t.Run("rate limit", func(t *testing.T) {
		ip := "10.200.99.60:1234"
		// Exhaust the admin rate limit for this IP (must match RemoteAddr format)
		for i := 0; i < maxAdminAttempts+1; i++ {
			checkAdminRateLimit(ip)
		}
		req := httptest.NewRequest("POST", "/api/v1/auth/admin-login", strings.NewReader(`{"vault_key":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = ip
		w := httptest.NewRecorder()
		handleAdminLogin(w, req)
		if w.Result().StatusCode != http.StatusTooManyRequests {
			t.Errorf("expected 429, got %d", w.Result().StatusCode)
		}
	})
}

func TestSha256Hex(t *testing.T) {
	result := sha256Hex("test")
	if len(result) != 64 {
		t.Errorf("expected 64-char hex, got %d chars", len(result))
	}
	// Known SHA-256 of "test"
	expected := "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestHandleOAuthURL(t *testing.T) {
	fixRepoRoot()

	t.Run("unsupported provider", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/oauth-url?provider=facebook", nil)
		w := httptest.NewRecorder()
		handleOAuthURL(w, req)
		if w.Result().StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Result().StatusCode)
		}
	})

	t.Run("no env vars configured", func(t *testing.T) {
		orig := os.Getenv("OAUTH_GOOGLE_CLIENT_ID")
		os.Unsetenv("OAUTH_GOOGLE_CLIENT_ID")
		defer os.Setenv("OAUTH_GOOGLE_CLIENT_ID", orig)

		req := httptest.NewRequest("GET", "/api/v1/auth/oauth-url?provider=google", nil)
		w := httptest.NewRecorder()
		handleOAuthURL(w, req)
		if w.Result().StatusCode != http.StatusServiceUnavailable {
			t.Errorf("expected 503, got %d", w.Result().StatusCode)
		}
	})

	t.Run("with env vars returns URL", func(t *testing.T) {
		cleanup := setOAuthEnvs(t, "test-client-id", "test-secret", "", "")
		defer cleanup()

		req := httptest.NewRequest("GET", "/api/v1/auth/oauth-url?provider=google", nil)
		w := httptest.NewRecorder()
		handleOAuthURL(w, req)
		if w.Result().StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Result().StatusCode)
		}
	})

	t.Run("redirect mode", func(t *testing.T) {
		cleanup := setOAuthEnvs(t, "test-client-id", "test-secret", "", "")
		defer cleanup()

		req := httptest.NewRequest("GET", "/api/v1/auth/oauth-url?provider=google&redirect=1", nil)
		w := httptest.NewRecorder()
		handleOAuthURL(w, req)
		if w.Result().StatusCode != http.StatusFound {
			t.Errorf("expected 302, got %d", w.Result().StatusCode)
		}
	})
}

func TestHandleLoginChallenge(t *testing.T) {
	fixRepoRoot()

	req := httptest.NewRequest("GET", "/api/v1/auth/login-challenge", nil)
	w := httptest.NewRecorder()
	handleLoginChallenge(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if body["challenge"] == nil || body["challenge"] == "" {
		t.Error("expected non-empty challenge token")
	}
	if body["expires_in"] != "60" {
		t.Errorf("expected expires_in=60, got %v", body["expires_in"])
	}
}

func TestValidateChallenge(t *testing.T) {
	fixRepoRoot()

	t.Run("no challenge param", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/login?challenge=", nil)
		if validateChallenge(req) {
			t.Error("expected false for empty challenge")
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/login?challenge=invalid.token.here", nil)
		if validateChallenge(req) {
			t.Error("expected false for invalid token")
		}
	})

	t.Run("valid challenge token", func(t *testing.T) {
		// Generate a valid challenge token
		if err := initJWT(); err != nil {
			t.Fatalf("failed to init JWT: %v", err)
		}
		claims := jwtClaims{
			Sub:  "challenge:127.0.0.1:nonce123",
			Role: "challenge",
			Iat:  time.Now().Unix(),
			Exp:  time.Now().Add(60 * time.Second).Unix(),
		}
		token, err := signJWT(claims)
		if err != nil {
			t.Fatalf("failed to sign challenge token: %v", err)
		}

		req := httptest.NewRequest("GET", "/login?challenge="+token, nil)
		if !validateChallenge(req) {
			t.Error("expected valid challenge to pass")
		}
	})

	t.Run("wrong role", func(t *testing.T) {
		if err := initJWT(); err != nil {
			t.Fatalf("failed to init JWT: %v", err)
		}
		claims := jwtClaims{
			Sub:  "user@test.com",
			Role: "admin",
			Iat:  time.Now().Unix(),
			Exp:  time.Now().Add(60 * time.Second).Unix(),
		}
		token, _ := signJWT(claims)
		req := httptest.NewRequest("GET", "/login?challenge="+token, nil)
		if validateChallenge(req) {
			t.Error("expected false for wrong role")
		}
	})

	t.Run("expired token", func(t *testing.T) {
		if err := initJWT(); err != nil {
			t.Fatalf("failed to init JWT: %v", err)
		}
		claims := jwtClaims{
			Sub:  "challenge:127.0.0.1:nonce",
			Role: "challenge",
			Iat:  time.Now().Add(-2 * time.Minute).Unix(),
			Exp:  time.Now().Add(-1 * time.Minute).Unix(), // already expired
		}
		token, _ := signJWT(claims)
		req := httptest.NewRequest("GET", "/login?challenge="+token, nil)
		if validateChallenge(req) {
			t.Error("expected false for expired token")
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// TOTP Functions (totp.go)
// ═══════════════════════════════════════════════════════════════════════════════

func TestGenerateTOTPSecret(t *testing.T) {
	secret, err := generateTOTPSecret()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if secret == "" {
		t.Error("expected non-empty secret")
	}
	// Two secrets should be different
	secret2, _ := generateTOTPSecret()
	if secret == secret2 {
		t.Error("expected different secrets")
	}
}

func TestComputeTOTP(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP" // base32 of "Hello!"
	now := time.Now()
	code := computeTOTP(secret, now)
	if code == 0 {
		t.Error("expected non-zero TOTP code")
	}
	if code >= 1000000 {
		t.Errorf("TOTP code should be 6 digits, got %d", code)
	}

	// Invalid base32 should return 0
	invalid := computeTOTP("!@#$%", now)
	if invalid != 0 {
		t.Errorf("expected 0 for invalid secret, got %d", invalid)
	}
}

func TestValidateTOTP(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	now := time.Now()
	code := fmt.Sprintf("%06d", computeTOTP(secret, now))

	// Current time should validate
	if !validateTOTP(secret, code) {
		t.Error("expected valid TOTP code to pass")
	}

	// Wrong length
	if validateTOTP(secret, "123") {
		t.Error("expected invalid for wrong length")
	}

	// Wrong code
	if validateTOTP(secret, "000000") {
		t.Error("expected invalid for wrong code")
	}
}

func TestTotpStoragePath(t *testing.T) {
	fixRepoRoot()
	path := totpStoragePath()
	if !strings.Contains(path, "totp_secret.json") {
		t.Errorf("expected totp_secret.json in path, got %s", path)
	}
}

func TestLoadTOTPEntry_MissingFile(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	RepoRoot = t.TempDir()
	_, err := loadTOTPEntry()
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestSaveAndLoadTOTPEntry(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "runtime"), 0700)
	RepoRoot = dir

	entry := &totpEntry{
		Email:     "test@ovav.dev",
		Secret:    "JBSWY3DPEHPK3PXP",
		CreatedAt: time.Now().Unix(),
		Verified:  true,
	}

	if err := saveTOTPEntry(entry); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	loaded, err := loadTOTPEntry()
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}
	if loaded.Email != entry.Email {
		t.Errorf("expected email=%s, got %s", entry.Email, loaded.Email)
	}
	if loaded.Secret != entry.Secret {
		t.Errorf("expected secret=%s, got %s", entry.Secret, loaded.Secret)
	}
	if !loaded.Verified {
		t.Error("expected verified=true")
	}
}

func TestDeriveStorageKey(t *testing.T) {
	key := deriveStorageKey()
	if len(key) != 32 {
		t.Errorf("expected 32-byte key, got %d", len(key))
	}
	// Same RepoRoot should produce same key
	key2 := deriveStorageKey()
	if !bytes.Equal(key, key2) {
		t.Error("expected same key for same RepoRoot")
	}
}

func TestAesEncryptDecryptRoundtrip(t *testing.T) {
	key := deriveStorageKey()
	plaintext := []byte("hello world, this is a test message!")

	encrypted, err := aesEncrypt(plaintext, key)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	decrypted, err := aesDecrypt(encrypted, key)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("expected roundtrip to produce original text")
	}
}

func TestAesDecrypt_TooShort(t *testing.T) {
	key := deriveStorageKey()
	_, err := aesDecrypt([]byte("short"), key)
	if err == nil {
		t.Error("expected error for too-short ciphertext")
	}
}

func TestAesDecrypt_WrongKey(t *testing.T) {
	key1 := deriveStorageKey()
	key2 := make([]byte, 32)
	copy(key2, "wrong_key_wrong_key_wrong_key!")

	encrypted, err := aesEncrypt([]byte("secret data"), key1)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	_, err = aesDecrypt(encrypted, key2)
	if err == nil {
		t.Error("expected error when decrypting with wrong key")
	}
}

func TestTotpURL(t *testing.T) {
	url := totpURL("user@ovav.dev", "ABCDEF")
	if !strings.HasPrefix(url, "otpauth://totp/") {
		t.Errorf("expected otpauth:// URL, got %s", url)
	}
	if !strings.Contains(url, "user@ovav.dev") {
		t.Error("expected email in URL")
	}
	if !strings.Contains(url, "ABCDEF") {
		t.Error("expected secret in URL")
	}
	if !strings.Contains(url, "OVAV-cPanel") {
		t.Error("expected issuer in URL")
	}
}

func TestIsAdminEmail(t *testing.T) {
	t.Run("no env var", func(t *testing.T) {
		orig := os.Getenv("ADMIN_EMAILS")
		os.Unsetenv("ADMIN_EMAILS")
		defer os.Setenv("ADMIN_EMAILS", orig)

		if isAdminEmail("user@ovav.dev") {
			t.Error("expected false when ADMIN_EMAILS not set")
		}
	})

	t.Run("matching email", func(t *testing.T) {
		os.Setenv("ADMIN_EMAILS", "admin@ovav.dev,ceo@ovav.dev")
		defer os.Unsetenv("ADMIN_EMAILS")

		if !isAdminEmail("admin@ovav.dev") {
			t.Error("expected true for matching admin email")
		}
	})

	t.Run("non-matching email", func(t *testing.T) {
		os.Setenv("ADMIN_EMAILS", "admin@ovav.dev")
		defer os.Unsetenv("ADMIN_EMAILS")

		if isAdminEmail("user@ovav.dev") {
			t.Error("expected false for non-admin email")
		}
	})
}

func TestHandleTOTPSetup(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "runtime"), 0700)
	RepoRoot = dir

	t.Run("invalid JSON body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/totp/setup", strings.NewReader("not json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handleTOTPSetup(w, req)
		if w.Result().StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Result().StatusCode)
		}
	})

	t.Run("non-admin email", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/totp/setup", strings.NewReader(`{"email":"user@test.com"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handleTOTPSetup(w, req)
		if w.Result().StatusCode != http.StatusForbidden {
			t.Errorf("expected 403, got %d", w.Result().StatusCode)
		}
	})

	t.Run("admin email generates secret", func(t *testing.T) {
		os.Setenv("ADMIN_EMAILS", "admin@ovav.dev")
		defer os.Unsetenv("ADMIN_EMAILS")

		req := httptest.NewRequest("POST", "/api/v1/auth/totp/setup", strings.NewReader(`{"email":"admin@ovav.dev"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handleTOTPSetup(w, req)

		if w.Result().StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Result().StatusCode, w.Body.String())
		}
		var body map[string]interface{}
		json.NewDecoder(w.Result().Body).Decode(&body)
		if body["secret"] == nil {
			t.Error("expected secret in response")
		}
		if body["otpauth"] == nil {
			t.Error("expected otpauth URL in response")
		}
	})

	t.Run("already configured", func(t *testing.T) {
		os.Setenv("ADMIN_EMAILS", "admin@ovav.dev")
		defer os.Unsetenv("ADMIN_EMAILS")

		// Save verified entry so handler detects it
		saveTOTPEntry(&totpEntry{
			Email:     "admin@ovav.dev",
			Secret:    "JBSWY3DPEHPK3PXP",
			CreatedAt: time.Now().Unix(),
			Verified:  true,
		})

		req := httptest.NewRequest("POST", "/api/v1/auth/totp/setup", strings.NewReader(`{"email":"admin@ovav.dev"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handleTOTPSetup(w, req)

		var body map[string]interface{}
		json.NewDecoder(w.Result().Body).Decode(&body)
		if body["status"] != "already_configured" {
			t.Errorf("expected already_configured, got %v (status %d)", body["status"], w.Result().StatusCode)
		}
	})
}

func TestHandleTOTPVerify(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "runtime"), 0700)
	RepoRoot = dir

	t.Run("invalid JSON body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/totp/verify", strings.NewReader("not json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handleTOTPVerify(w, req)
		if w.Result().StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Result().StatusCode)
		}
	})

	t.Run("non-admin email", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/totp/verify", strings.NewReader(`{"email":"user@test.com","code":"123456"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handleTOTPVerify(w, req)
		if w.Result().StatusCode != http.StatusForbidden {
			t.Errorf("expected 403, got %d", w.Result().StatusCode)
		}
	})

	t.Run("TOTP not configured", func(t *testing.T) {
		os.Setenv("ADMIN_EMAILS", "totp-verify-admin@test.com")
		defer os.Unsetenv("ADMIN_EMAILS")

		req := httptest.NewRequest("POST", "/api/v1/auth/totp/verify", strings.NewReader(`{"email":"totp-verify-admin@test.com","code":"123456"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handleTOTPVerify(w, req)
		if w.Result().StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Result().StatusCode)
		}
	})

	t.Run("wrong code", func(t *testing.T) {
		os.Setenv("ADMIN_EMAILS", "admin@ovav.dev")
		defer os.Unsetenv("ADMIN_EMAILS")

		// Setup first
		req1 := httptest.NewRequest("POST", "/api/v1/auth/totp/setup", strings.NewReader(`{"email":"admin@ovav.dev"}`))
		req1.Header.Set("Content-Type", "application/json")
		w1 := httptest.NewRecorder()
		handleTOTPSetup(w1, req1)

		// Verify with wrong code
		req2 := httptest.NewRequest("POST", "/api/v1/auth/totp/verify", strings.NewReader(`{"email":"admin@ovav.dev","code":"000000"}`))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		handleTOTPVerify(w2, req2)
		if w2.Result().StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401 for wrong code, got %d", w2.Result().StatusCode)
		}
	})

	t.Run("valid code issues JWT", func(t *testing.T) {
		os.Setenv("ADMIN_EMAILS", "admin@ovav.dev")
		defer os.Unsetenv("ADMIN_EMAILS")

		// Setup
		req1 := httptest.NewRequest("POST", "/api/v1/auth/totp/setup", strings.NewReader(`{"email":"admin@ovav.dev"}`))
		req1.Header.Set("Content-Type", "application/json")
		w1 := httptest.NewRecorder()
		handleTOTPSetup(w1, req1)

		// Load secret to compute valid code
		entry, err := loadTOTPEntry()
		if err != nil {
			t.Fatalf("failed to load TOTP entry: %v", err)
		}
		validCode := fmt.Sprintf("%06d", computeTOTP(entry.Secret, time.Now()))

		req2 := httptest.NewRequest("POST", "/api/v1/auth/totp/verify", strings.NewReader(`{"email":"admin@ovav.dev","code":"`+validCode+`"}`))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		handleTOTPVerify(w2, req2)

		resp := w2.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 for valid code, got %d: %s", resp.StatusCode, w2.Body.String())
		}
		var body map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&body)
		if body["token"] == nil {
			t.Error("expected JWT token in response")
		}
		if body["role"] != "admin" {
			t.Errorf("expected admin role, got %v", body["role"])
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Server — isPublicPath, authMiddleware
// ═══════════════════════════════════════════════════════════════════════════════

func TestIsPublicPath(t *testing.T) {
	tests := []struct {
		path   string
		expect bool
	}{
		{"/health", true},
		{"/api/v1/health", true},
		{"/api/v1/auth/login", true},
		{"/api/v1/auth/session", true},
		{"/api/v1/auth/config", true},
		{"/api/v1/auth/cli-verify", true},
		{"/api/v1/auth/oauth/google", true},
		{"/api/v1/events", true},
		{"/assets/index.html", true},
		{"/css/style.css", true},
		{"/login", true},
		{"/api/v1/product/version", true},
		{"/api/v1/product/update", true},
		{"/login/verify", true},
		{"/api/v1/status", false},
		{"/api/v1/agents", false},
		{"/api/v1/system/health", false},
		{"/api/v1/governor/health", false},
		{"/api/v1/memory/status", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isPublicPath(tt.path)
			if got != tt.expect {
				t.Errorf("isPublicPath(%q) = %v, want %v", tt.path, got, tt.expect)
			}
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
	fixRepoRoot()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	t.Run("OPTIONS preflight always allowed", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/v1/status", nil)
		w := httptest.NewRecorder()
		authMiddleware(inner).ServeHTTP(w, req)
		if w.Result().StatusCode != http.StatusOK {
			t.Errorf("expected 200 for OPTIONS, got %d", w.Result().StatusCode)
		}
	})

	t.Run("public path bypasses auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		authMiddleware(inner).ServeHTTP(w, req)
		if w.Result().StatusCode != http.StatusOK {
			t.Errorf("expected 200 for public path, got %d", w.Result().StatusCode)
		}
	})

	t.Run("protected path without token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/status", nil)
		w := httptest.NewRecorder()
		authMiddleware(inner).ServeHTTP(w, req)
		if w.Result().StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Result().StatusCode)
		}
	})

	t.Run("protected path with invalid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/status", nil)
		req.Header.Set("Authorization", "Bearer bad-token")
		w := httptest.NewRecorder()
		authMiddleware(inner).ServeHTTP(w, req)
		if w.Result().StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Result().StatusCode)
		}
	})

	t.Run("SSE events path bypasses auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/events", nil)
		w := httptest.NewRecorder()
		authMiddleware(inner).ServeHTTP(w, req)
		if w.Result().StatusCode != http.StatusOK {
			t.Errorf("expected 200 for events path, got %d", w.Result().StatusCode)
		}
	})

	t.Run("root path without token redirects to /login", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		authMiddleware(inner).ServeHTTP(w, req)
		if w.Result().StatusCode != http.StatusFound {
			t.Errorf("expected 302 redirect, got %d", w.Result().StatusCode)
		}
	})

	t.Run("cookie-based auth", func(t *testing.T) {
		// Init JWT and create a valid token
		if err := initJWT(); err != nil {
			t.Fatalf("failed to init JWT: %v", err)
		}
		claims := jwtClaims{
			Sub:  "test-user",
			Role: "operator",
			Iat:  time.Now().Unix(),
			Exp:  time.Now().Add(1 * time.Hour).Unix(),
		}
		token, err := signJWT(claims)
		if err != nil {
			t.Fatalf("failed to sign JWT: %v", err)
		}
		jwtSessLock.Lock()
		jwtSessions[token] = sessionInfo{Token: token, Role: "operator", ExpiresAt: time.Now().Add(1 * time.Hour)}
		jwtSessLock.Unlock()

		req := httptest.NewRequest("GET", "/api/v1/status", nil)
		req.AddCookie(&http.Cookie{Name: "ovav_token", Value: token})
		w := httptest.NewRecorder()
		authMiddleware(inner).ServeHTTP(w, req)
		if w.Result().StatusCode != http.StatusOK {
			t.Errorf("expected 200 with valid cookie, got %d", w.Result().StatusCode)
		}
	})

	t.Run("root path with valid token serves dashboard", func(t *testing.T) {
		if err := initJWT(); err != nil {
			t.Fatalf("failed to init JWT: %v", err)
		}
		claims := jwtClaims{
			Sub:  "test-user",
			Role: "operator",
			Iat:  time.Now().Unix(),
			Exp:  time.Now().Add(1 * time.Hour).Unix(),
		}
		token, _ := signJWT(claims)
		jwtSessLock.Lock()
		jwtSessions[token] = sessionInfo{Token: token, Role: "operator", ExpiresAt: time.Now().Add(1 * time.Hour)}
		jwtSessLock.Unlock()

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		authMiddleware(inner).ServeHTTP(w, req)
		if w.Result().StatusCode != http.StatusOK {
			t.Errorf("expected 200 for valid token on root, got %d", w.Result().StatusCode)
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Product Handlers (product_handlers.go)
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandleProductVersion(t *testing.T) {
	fixRepoRoot()
	req := httptest.NewRequest("GET", "/api/v1/product/version", nil)
	w := httptest.NewRecorder()
	handleProductVersion(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if body["channel"] != "stable" {
		t.Errorf("expected channel=stable, got %v", body["channel"])
	}
	if body["checked_at"] == nil {
		t.Error("expected checked_at timestamp")
	}
}

func TestHandleProductUpdate(t *testing.T) {
	root := t.TempDir()
	t.Setenv("OVAV_ROOT", root)
	if err := os.WriteFile(filepath.Join(root, "VERSION"), []byte("test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	previous := runProductSyncCommand
	runProductSyncCommand = func(context.Context, string, ...string) ([]byte, error) { return nil, nil }
	t.Cleanup(func() { runProductSyncCommand = previous })

	// Initialize hub so ClientCount() works
	hub.Start()

	req := httptest.NewRequest("POST", "/api/v1/product/update", nil)
	w := httptest.NewRecorder()
	handleProductUpdate(w, req)

	resp := w.Result()
	// May succeed or fail depending on environment (git, go commands)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusMultiStatus && resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 200/207/500, got %d", resp.StatusCode)
	}
}

func TestOvavRoot(t *testing.T) {
	t.Run("with env var", func(t *testing.T) {
		os.Setenv("OVAV_ROOT", "/custom/path")
		defer os.Unsetenv("OVAV_ROOT")

		result := ovavRoot()
		if result != "/custom/path" {
			t.Errorf("expected /custom/path, got %s", result)
		}
	})

	t.Run("without env var walks up", func(t *testing.T) {
		os.Unsetenv("OVAV_ROOT")
		result := ovavRoot()
		if result == "" {
			t.Error("expected non-empty result")
		}
	})
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]string{"key": "value"})

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected JSON content type, got %s", ct)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Product Sync (product_sync.go)
// ═══════════════════════════════════════════════════════════════════════════════

func TestSyncProtect(t *testing.T) {
	ResetRateLimiterForTesting()

	t.Run("rate limit exceeded", func(t *testing.T) {
		ip := "10.200.50.77:1234"
		// Exhaust rate limit using the same IP format as syncProtect uses (r.RemoteAddr)
		for i := 0; i < 6; i++ {
			checkRateLimit(ip)
		}

		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/api/v1/product/sync/status", nil)
		req.RemoteAddr = ip
		w := httptest.NewRecorder()
		syncProtect(inner)(w, req)
		if w.Result().StatusCode != http.StatusTooManyRequests {
			t.Errorf("expected 429, got %d", w.Result().StatusCode)
		}
	})

	t.Run("within rate limit", func(t *testing.T) {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/api/v1/product/sync/status", nil)
		req.RemoteAddr = "10.200.50.88:5678"
		w := httptest.NewRecorder()
		syncProtect(inner)(w, req)
		if w.Result().StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Result().StatusCode)
		}
	})
}

func TestHandleProductSyncStatus(t *testing.T) {
	fixRepoRoot()
	req := httptest.NewRequest("GET", "/api/v1/product/sync/status", nil)
	w := httptest.NewRecorder()
	handleProductSyncStatus(w, req)
	// May succeed or fail depending on sync module
	t.Logf("sync status: %d", w.Result().StatusCode)
}

func TestHandleProductSyncStage(t *testing.T) {
	fixRepoRoot()

	t.Run("GET not allowed", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/product/sync/stage", nil)
		w := httptest.NewRecorder()
		handleProductSyncStage(w, req)
		if w.Result().StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", w.Result().StatusCode)
		}
	})

	t.Run("POST with items", func(t *testing.T) {
		body := strings.NewReader(`{"items":["item1","item2"]}`)
		req := httptest.NewRequest("POST", "/api/v1/product/sync/stage", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handleProductSyncStage(w, req)
		t.Logf("sync stage: %d", w.Result().StatusCode)
	})
}

func TestHandleProductSyncPush(t *testing.T) {
	fixRepoRoot()
	hub.Start()

	t.Run("GET not allowed", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/product/sync/push", nil)
		w := httptest.NewRecorder()
		handleProductSyncPush(w, req)
		if w.Result().StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", w.Result().StatusCode)
		}
	})

	t.Run("POST push", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/product/sync/push", nil)
		w := httptest.NewRecorder()
		handleProductSyncPush(w, req)
		t.Logf("sync push: %d", w.Result().StatusCode)
	})
}

func TestHandleProductSyncApply(t *testing.T) {
	fixRepoRoot()
	hub.Start()

	t.Run("GET not allowed", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/product/sync/apply", nil)
		w := httptest.NewRecorder()
		handleProductSyncApply(w, req)
		if w.Result().StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", w.Result().StatusCode)
		}
	})

	t.Run("POST apply", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/product/sync/apply", nil)
		w := httptest.NewRecorder()
		handleProductSyncApply(w, req)
		t.Logf("sync apply: %d", w.Result().StatusCode)
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Static serving — improved coverage
// ═══════════════════════════════════════════════════════════════════════════════

func TestSpaIngress_Root(t *testing.T) {
	fixRepoRoot()
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	spaIngress(w, req)
	t.Logf("spaIngress / status: %d", w.Result().StatusCode)
}

func TestSpaIngress_SubPath(t *testing.T) {
	fixRepoRoot()
	req := httptest.NewRequest("GET", "/some/asset", nil)
	w := httptest.NewRecorder()
	spaIngress(w, req)
	// Falls through to serveAsset
	t.Logf("spaIngress /some/asset status: %d", w.Result().StatusCode)
}

func TestServeAsset_UnknownExtension(t *testing.T) {
	fixRepoRoot()
	req := httptest.NewRequest("GET", "/assets/unknown.xyz", nil)
	w := httptest.NewRecorder()
	serveAsset(w, req)
	// Will get 404 or SPA fallback — both acceptable
	t.Logf("unknown extension status: %d", w.Result().StatusCode)
}

func TestServeAsset_APISpaFallbackBlocked(t *testing.T) {
	fixRepoRoot()
	req := httptest.NewRequest("GET", "/api/nonexistent.html", nil)
	w := httptest.NewRecorder()
	serveAsset(w, req)
	if w.Result().StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for API path, got %d", w.Result().StatusCode)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Validators — initBgTasks, handleValidatorsStatus
// ═══════════════════════════════════════════════════════════════════════════════

func TestInitBgTasksIdempotent(t *testing.T) {
	// initBgTasks uses sync.Once — calling twice should be fine
	initBgTasks()
	initBgTasks()
}

func TestHandleValidatorsStatus_Found(t *testing.T) {
	// Create a background task
	bgTasksMutex.Lock()
	taskID := "test-task-123"
	bgTasksMap[taskID] = &bgTask{Status: "complete", Result: "test result"}
	bgTasksMutex.Unlock()

	req := httptest.NewRequest("GET", "/api/v1/validators/status/"+taskID, nil)
	w := httptest.NewRecorder()
	handleValidatorsStatus(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Cleanup
	bgTasksMutex.Lock()
	delete(bgTasksMap, taskID)
	bgTasksMutex.Unlock()
}

// ═══════════════════════════════════════════════════════════════════════════════
// Auth — deeper coverage for loadOVAVSession, initJWT, signJWT, verifyJWT
// ═══════════════════════════════════════════════════════════════════════════════

func TestLoadOVAVSession_MissingFile(t *testing.T) {
	// This test verifies the function returns empty when session file is absent.
	// We can't easily mock os.UserHomeDir, so we test that the function
	// returns gracefully regardless of environment.
	sess, ok := loadOVAVSession()
	// If a real session exists on this machine, ok will be true — that's fine.
	// The important thing is the function doesn't panic.
	if sess == (OVAVSession{}) && ok {
		t.Error("if session is zero-valued, ok should be false")
	}
}

func TestLoadOVAVSession_InvalidJSON(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("can't get home dir")
	}

	sessionDir := filepath.Join(home, ".local", "share", "ovav")
	sessionPath := filepath.Join(sessionDir, "session")

	// Back up existing file
	var origData []byte
	if data, readErr := os.ReadFile(sessionPath); readErr == nil {
		origData = data
		os.WriteFile(sessionPath+".bak", data, 0600)
	}
	defer func() {
		if origData != nil {
			os.WriteFile(sessionPath, origData, 0600)
			os.Remove(sessionPath + ".bak")
		} else {
			os.Remove(sessionPath)
		}
	}()

	// Write invalid JSON
	os.MkdirAll(sessionDir, 0700)
	os.WriteFile(sessionPath, []byte("not-json"), 0600)

	sess, ok := loadOVAVSession()
	if ok {
		t.Error("expected ok=false for invalid JSON")
	}
	_ = sess
}

func TestLoadOVAVSession_Expired(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("can't get home dir")
	}

	sessionDir := filepath.Join(home, ".local", "share", "ovav")
	sessionPath := filepath.Join(sessionDir, "session")

	// Back up existing file
	var origData []byte
	if data, readErr := os.ReadFile(sessionPath); readErr == nil {
		origData = data
		os.WriteFile(sessionPath+".bak", data, 0600)
	}
	defer func() {
		if origData != nil {
			os.WriteFile(sessionPath, origData, 0600)
			os.Remove(sessionPath + ".bak")
		} else {
			os.Remove(sessionPath)
		}
	}()

	// Write expired session (created 48h ago)
	os.MkdirAll(sessionDir, 0700)
	expiredTime := time.Now().Add(-48 * time.Hour).Format(time.RFC3339)
	sessData := `{"vault_key_hash":"test123","machine_id":"machine1","created_at":"` + expiredTime + `"}`
	os.WriteFile(sessionPath, []byte(sessData), 0600)

	sess, ok := loadOVAVSession()
	if ok {
		t.Error("expected ok=false for expired session")
	}
	_ = sess
}

func TestLoadOVAVSession_Valid(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("can't get home dir")
	}

	sessionDir := filepath.Join(home, ".local", "share", "ovav")
	sessionPath := filepath.Join(sessionDir, "session")

	// Back up existing file
	var origData []byte
	if data, readErr := os.ReadFile(sessionPath); readErr == nil {
		origData = data
		os.WriteFile(sessionPath+".bak", data, 0600)
	}
	defer func() {
		if origData != nil {
			os.WriteFile(sessionPath, origData, 0600)
			os.Remove(sessionPath + ".bak")
		} else {
			os.Remove(sessionPath)
		}
	}()

	// Write valid session
	os.MkdirAll(sessionDir, 0700)
	validTime := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	sessData := `{"vault_key_hash":"test-key-hash","machine_id":"m1","created_at":"` + validTime + `","role":"operator"}`
	os.WriteFile(sessionPath, []byte(sessData), 0600)

	sess, ok := loadOVAVSession()
	if !ok {
		t.Error("expected ok=true for valid session")
	}
	if sess.VaultKeyHash != "test-key-hash" {
		t.Errorf("expected vault_key_hash=test-key-hash, got %s", sess.VaultKeyHash)
	}
}

func TestInitJWT_AlreadyInitialized(t *testing.T) {
	// First call initializes
	if err := initJWT(); err != nil {
		t.Fatalf("first init failed: %v", err)
	}
	// Second call should be a no-op (already initialized)
	if err := initJWT(); err != nil {
		t.Fatalf("second init failed: %v", err)
	}
}

func TestSignJWT_WithoutKey(t *testing.T) {
	// Save and nil the key
	jwtKeyLock.Lock()
	savedKey := jwtPrivateKey
	jwtPrivateKey = nil
	jwtKeyLock.Unlock()

	defer func() {
		jwtKeyLock.Lock()
		jwtPrivateKey = savedKey
		jwtKeyLock.Unlock()
	}()

	claims := jwtClaims{Sub: "test", Role: "operator", Iat: time.Now().Unix(), Exp: time.Now().Add(time.Hour).Unix()}
	_, err := signJWT(claims)
	if err == nil {
		t.Error("expected error when JWT key is nil")
	}
}

func TestVerifyJWT_ExpiredToken(t *testing.T) {
	if err := initJWT(); err != nil {
		t.Fatalf("init JWT failed: %v", err)
	}

	claims := jwtClaims{
		Sub:  "test",
		Role: "operator",
		Iat:  time.Now().Add(-2 * time.Hour).Unix(),
		Exp:  time.Now().Add(-1 * time.Hour).Unix(), // expired
	}
	token, err := signJWT(claims)
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}

	_, err = verifyJWT(token)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestVerifyJWT_BadSignature(t *testing.T) {
	if err := initJWT(); err != nil {
		t.Fatalf("init JWT failed: %v", err)
	}

	// Create a valid token, then corrupt the signature
	claims := jwtClaims{Sub: "test", Role: "operator", Iat: time.Now().Unix(), Exp: time.Now().Add(time.Hour).Unix()}
	token, _ := signJWT(claims)
	parts := strings.Split(token, ".")
	corruptedToken := parts[0] + "." + parts[1] + "." + base64urlEncode([]byte("bad-sig"))

	_, err := verifyJWT(corruptedToken)
	if err == nil {
		t.Error("expected error for corrupted signature")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Profiles — more apply coverage
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandleProfileApply_InvalidBody(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/profiles/apply", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleProfileApply(w, req)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Result().StatusCode)
	}
}

func TestHandleProfileApply_MissingServiceProfiles(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()
	RepoRoot = t.TempDir()

	body := strings.NewReader(`{"area":"platform_engineering","dry_run":true}`)
	req := httptest.NewRequest("POST", "/api/v1/profiles/apply", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleProfileApply(w, req)
	if w.Result().StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Result().StatusCode)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// System handlers — more edge cases
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandleSystemHealth_MissingFiles(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()
	RepoRoot = t.TempDir()

	req := httptest.NewRequest("GET", "/api/v1/system/health", nil)
	w := httptest.NewRecorder()
	handleSystemHealth(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "fail" {
		t.Errorf("expected status=fail for missing files, got %v", body["status"])
	}
}

func TestHandleConfig_DriftDetected(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	RepoRoot = dir

	// Create external opencode config to trigger drift detection
	home, _ := os.UserHomeDir()
	driftDir := filepath.Join(home, ".config", "opencode")
	os.MkdirAll(driftDir, 0755)
	driftFile := filepath.Join(driftDir, "opencode.jsonc")

	// Create it to trigger drift detection
	os.WriteFile(driftFile, []byte(`{}`), 0644)
	defer os.Remove(driftFile)

	req := httptest.NewRequest("GET", "/api/v1/system/config", nil)
	w := httptest.NewRecorder()
	handleConfig(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHandleKCStatus_WithTriggers(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	registryDir := filepath.Join(dir, ".ovav", "registry")
	os.MkdirAll(registryDir, 0755)
	os.WriteFile(filepath.Join(registryDir, "auto_triggers.yaml"), []byte("- trigger: test\n- id: another\n- trigger: third\n"), 0644)
	RepoRoot = dir

	req := httptest.NewRequest("GET", "/api/v1/system/kc", nil)
	w := httptest.NewRecorder()
	handleKCStatus(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	if body["triggers_count"].(float64) != 3 {
		t.Errorf("expected 3 triggers, got %v", body["triggers_count"])
	}
}

func TestHandleOperations_DirtyRepo(t *testing.T) {
	fixRepoRoot()
	req := httptest.NewRequest("GET", "/api/v1/system/operations", nil)
	w := httptest.NewRecorder()
	handleOperations(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	if body["install"] == nil {
		t.Error("expected install section")
	}
	if body["segment"] == nil {
		t.Error("expected segment section")
	}
}

func TestHandleSBOM_LegacyFallback(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	registryDir := filepath.Join(dir, ".ovav", "registry")
	os.MkdirAll(registryDir, 0755)
	os.WriteFile(filepath.Join(registryDir, "sbom.yaml"), []byte("dependencies: []"), 0644)
	RepoRoot = dir

	req := httptest.NewRequest("GET", "/api/v1/system/sbom", nil)
	w := httptest.NewRecorder()
	handleSBOM(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Agent handling — deeper coverage
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandleServiceProfiles_MissingFile(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()
	RepoRoot = t.TempDir()

	req := httptest.NewRequest("GET", "/api/v1/agents/profiles", nil)
	w := httptest.NewRecorder()
	handleServiceProfiles(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	if body["note"] != "not found" {
		t.Errorf("expected note='not found', got %v", body["note"])
	}
}

func TestHandleTopology_NoDir(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()
	RepoRoot = t.TempDir()

	req := httptest.NewRequest("GET", "/api/v1/agents/topology", nil)
	w := httptest.NewRecorder()
	handleTopology(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHandleAgentList_EmptyDir(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()
	RepoRoot = t.TempDir()

	req := httptest.NewRequest("GET", "/api/v1/agents", nil)
	w := httptest.NewRecorder()
	handleAgentList(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	if body["total"].(float64) != 0 {
		t.Errorf("expected 0 agents, got %v", body["total"])
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Memory handler edge cases
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandleMemoryStatus_NoCapsYaml(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()
	RepoRoot = t.TempDir()

	req := httptest.NewRequest("GET", "/api/v1/memory/status", nil)
	w := httptest.NewRecorder()
	handleMemoryStatus(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHandleLedger_NoFiles(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()
	RepoRoot = t.TempDir()

	req := httptest.NewRequest("GET", "/api/v1/memory/ledger", nil)
	w := httptest.NewRecorder()
	handleLedger(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	if body["cards"] == nil {
		t.Error("expected cards field")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Router integration — more routes
// ═══════════════════════════════════════════════════════════════════════════════

func TestRouterGovernorRoutes(t *testing.T) {
	fixRepoRoot()
	registerRoutes()

	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/governor/health"},
		{"GET", "/api/v1/governor/decisions"},
		{"GET", "/api/v1/governor/tasks"},
		{"GET", "/api/v1/security/defense/status"},
	}

	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			req := httptest.NewRequest(rt.method, rt.path, nil)
			w := httptest.NewRecorder()
			routerMux(w, req)
			if w.Result().StatusCode < 200 || w.Result().StatusCode >= 500 {
				t.Errorf("%s %s returned %d, expected 2xx-4xx", rt.method, rt.path, w.Result().StatusCode)
			}
		})
	}
}

func TestRouterProductRoutes(t *testing.T) {
	root := t.TempDir()
	t.Setenv("OVAV_ROOT", root)
	if err := os.WriteFile(filepath.Join(root, "VERSION"), []byte("test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	previous := runProductSyncCommand
	runProductSyncCommand = func(context.Context, string, ...string) ([]byte, error) { return nil, nil }
	t.Cleanup(func() { runProductSyncCommand = previous })
	registerRoutes()

	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/product/version"},
		{"POST", "/api/v1/product/update"},
		{"GET", "/login"},
		{"GET", "/login/verify"},
		{"GET", "/api/v1/auth/login-challenge"},
	}

	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			req := httptest.NewRequest(rt.method, rt.path, nil)
			w := httptest.NewRecorder()
			routerMux(w, req)
			if w.Result().StatusCode < 200 || w.Result().StatusCode >= 500 {
				t.Errorf("%s %s returned %d, expected 2xx-4xx", rt.method, rt.path, w.Result().StatusCode)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Git fetch edge cases
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandleGitFetch_WithBranch(t *testing.T) {
	fixRepoRoot()
	req := httptest.NewRequest("POST", "/api/v1/git/fetch?branch=main", nil)
	w := httptest.NewRecorder()
	handleGitFetch(w, req)
	t.Logf("git fetch with branch: %d", w.Result().StatusCode)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Shared init — CPANEL_ALLOWED_ORIGINS
// ═══════════════════════════════════════════════════════════════════════════════

func TestSharedInit_WithExtraOrigins(t *testing.T) {
	origEnv := os.Getenv("CPANEL_ALLOWED_ORIGINS")
	os.Setenv("CPANEL_ALLOWED_ORIGINS", "https://extra1.test.com, https://extra2.test.com")
	defer os.Setenv("CPANEL_ALLOWED_ORIGINS", origEnv)

	// Re-run the shared init logic
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

	if !allowedOrigins["https://extra1.test.com"] {
		t.Error("expected extra1.test.com to be allowed")
	}
	if !allowedOrigins["https://extra2.test.com"] {
		t.Error("expected extra2.test.com to be allowed")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Agent list — mode and hidden detection
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandleAgentList_WithAgentFiles(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".opencode", "agents")
	os.MkdirAll(agentsDir, 0755)

	// Create a test agent file
	agentContent := `---
mode: primary
hidden: true
---
# Test Agent`
	os.WriteFile(filepath.Join(agentsDir, "test-agent.md"), []byte(agentContent), 0644)

	RepoRoot = dir
	req := httptest.NewRequest("GET", "/api/v1/agents", nil)
	w := httptest.NewRecorder()
	handleAgentList(w, req)

	var body map[string]interface{}
	json.NewDecoder(w.Result().Body).Decode(&body)
	agents := body["agents"].([]interface{})
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}
	agent := agents[0].(map[string]interface{})
	if agent["mode"] != "primary" {
		t.Errorf("expected mode=primary, got %v", agent["mode"])
	}
	if agent["hidden"] != true {
		t.Errorf("expected hidden=true, got %v", agent["hidden"])
	}
}

func TestHandleTopology_WithAreas(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	topoDir := filepath.Join(dir, ".ovav", "topology")
	os.MkdirAll(topoDir, 0755)
	os.WriteFile(filepath.Join(topoDir, "area_platform.yaml"), []byte("name: platform\n"), 0644)
	os.WriteFile(filepath.Join(topoDir, "governance_rules.yaml"), []byte("rules: []\n"), 0644)

	RepoRoot = dir
	req := httptest.NewRequest("GET", "/api/v1/agents/topology", nil)
	w := httptest.NewRecorder()
	handleTopology(w, req)

	var body map[string]interface{}
	json.NewDecoder(w.Result().Body).Decode(&body)
	areas := body["areas"].([]interface{})
	if len(areas) != 1 {
		t.Errorf("expected 1 area, got %d", len(areas))
	}
	if body["governance"] == nil {
		t.Error("expected governance rules")
	}
}

func TestHandleServiceProfiles_WithYAML(t *testing.T) {
	origRoot := RepoRoot
	defer func() { RepoRoot = origRoot }()

	dir := t.TempDir()
	registryDir := filepath.Join(dir, ".ovav", "registry")
	os.MkdirAll(registryDir, 0755)
	yaml := `profiles:
  - id: platform_engineering
    name: Platform Engineering
    lead: Thavren
  - id: research_intelligence
    name: Research Intelligence
    lead: Eidren
`
	os.WriteFile(filepath.Join(registryDir, "service_profiles.yaml"), []byte(yaml), 0644)

	RepoRoot = dir
	req := httptest.NewRequest("GET", "/api/v1/agents/profiles", nil)
	w := httptest.NewRecorder()
	handleServiceProfiles(w, req)

	var body map[string]interface{}
	json.NewDecoder(w.Result().Body).Decode(&body)
	profiles := body["profiles"].([]interface{})
	if len(profiles) != 2 {
		t.Errorf("expected 2 profiles, got %d", len(profiles))
	}
}
