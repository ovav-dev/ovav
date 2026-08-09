// OVAV cPanel v5.0 — OAuth handlers (Google + GitHub).
//
// GET  /api/v1/auth/config              — Returns available OAuth providers
// POST /api/v1/auth/oauth/google        — Google OAuth callback (code exchange)
// POST /api/v1/auth/oauth/github        — GitHub OAuth callback (code exchange)
//
// Environment variables:
//   OAUTH_GOOGLE_CLIENT_ID, OAUTH_GOOGLE_CLIENT_SECRET
//   OAUTH_GITHUB_CLIENT_ID, OAUTH_GITHUB_CLIENT_SECRET
//   OAUTH_REDIRECT_URI  (required — must be explicitly set)
//
// OVAV Signature: cmd/cpanel/oauth.go — stabilized 2026-08-02
// Security: S-03 OAuth redirect URI fix — no hardcoded defaults
//
// Security:
//   - redirect_uri from env var only (no request-body override)
//   - Generic error messages (no internal leakage)
//   - 10s HTTP client timeout on provider requests
//   - CSRF handled by SPA (state param verified client-side)
//
// Stdlib-only. No external dependencies.

package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// oauthHTTPClient is a dedicated client with timeout for provider calls.
var oauthHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		DialContext:         (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
		TLSHandshakeTimeout: 5 * time.Second,
	},
}

// ── CSRF state protection ─────────────────────────────────────────────────────

var (
	oauthStateStore   = make(map[string]time.Time)
	oauthStateStoreMu sync.Mutex
	// oauthChallengeStore maps OAuth state token → CLI web-login challenge
	oauthChallengeStore = make(map[string]string)
)

func init() {
	// Cleanup expired states every 5 minutes
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			oauthStateStoreMu.Lock()
			now := time.Now()
			for state, expiry := range oauthStateStore {
				if now.After(expiry) {
					delete(oauthStateStore, state)
				}
			}
			oauthStateStoreMu.Unlock()
		}
	}()
}

// generateOAuthState creates a cryptographically random state token valid for 10 minutes.
// If challenge is non-empty, it is stored alongside the state for CLI web-login flows.
// CRITICAL FIX (C4): crypto/rand.Read error now verified. Previously, the error was
// silently ignored — if rand.Read failed, b would be [32]byte{0,0,...} producing
// a predictable state token: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=" (CSRF bypass).
func generateOAuthState(challenge string) string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failure is catastrophic — return empty state that will fail verification
		return ""
	}
	state := base64.URLEncoding.EncodeToString(b)
	oauthStateStoreMu.Lock()
	oauthStateStore[state] = time.Now().Add(10 * time.Minute)
	if challenge != "" {
		oauthChallengeStore[state] = challenge
	}
	oauthStateStoreMu.Unlock()
	return state
}

// verifyOAuthState validates the CSRF state token and consumes it (one-time use).
// Returns true if valid. The challenge (if any) is retrieved separately via getOAuthChallenge.
func verifyOAuthState(state string) bool {
	if state == "" {
		return false
	}
	oauthStateStoreMu.Lock()
	defer oauthStateStoreMu.Unlock()
	expiry, ok := oauthStateStore[state]
	if !ok {
		return false
	}
	delete(oauthStateStore, state) // one-time use
	return time.Now().Before(expiry)
}

// getOAuthChallenge retrieves and clears the CLI web-login challenge for a given OAuth state.
// Must be called AFTER verifyOAuthState succeeds (or within it) to prevent race on deletion.
func getOAuthChallenge(state string) string {
	oauthStateStoreMu.Lock()
	defer oauthStateStoreMu.Unlock()
	challenge := oauthChallengeStore[state]
	delete(oauthChallengeStore, state)
	return challenge
}

type oauthProvider struct {
	Provider    string `json:"provider"`
	ClientID    string `json:"client_id"`
	RedirectURI string `json:"redirect_uri"`
}

var oauthProviders []oauthProvider
var oauthConfigured bool

func initOAuth() {
	redirectURI := os.Getenv("OAUTH_REDIRECT_URI")
	// SECURITY: OAUTH_REDIRECT_URI is required. No hardcoded fallback —
	// OAuth providers enforce exact URI matching, so a wrong default would
	// cause immediate auth failure anyway.
	if redirectURI == "" {
		oauthProviders = nil
		oauthConfigured = false
		return
	}

	googleID := os.Getenv("OAUTH_GOOGLE_CLIENT_ID")
	googleSecret := os.Getenv("OAUTH_GOOGLE_CLIENT_SECRET")
	githubID := os.Getenv("OAUTH_GITHUB_CLIENT_ID")
	githubSecret := os.Getenv("OAUTH_GITHUB_CLIENT_SECRET")

	oauthProviders = nil

	if googleID != "" && googleSecret != "" {
		oauthProviders = append(oauthProviders, oauthProvider{
			Provider:    "google",
			ClientID:    googleID,
			RedirectURI: redirectURI,
		})
	}

	if githubID != "" && githubSecret != "" {
		oauthProviders = append(oauthProviders, oauthProvider{
			Provider:    "github",
			ClientID:    githubID,
			RedirectURI: redirectURI,
		})
	}

	oauthConfigured = len(oauthProviders) > 0
}

// ── GET /api/v1/auth/config ───────────────────────────────────────────────────

func handleAuthConfig(w http.ResponseWriter, r *http.Request) {
	initOAuth()

	methods := []string{"token"}
	if oauthConfigured {
		methods = append([]string{"oauth"}, methods...)
	}

	sendOK(w, map[string]interface{}{
		"methods":   methods,
		"oauth":     oauthProviders,
		"has_oauth": oauthConfigured,
	})
}

// ── POST /api/v1/auth/oauth/{provider} ────────────────────────────────────────

func handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	initOAuth()

	// Extract provider from URL path: /api/v1/auth/oauth/{provider}
	provider := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/oauth/")
	if provider == "" || provider == r.URL.Path || strings.Contains(provider, "/") {
		sendError(w, "invalid provider", http.StatusBadRequest)
		return
	}

	if !oauthConfigured {
		sendError(w, "OAuth not configured", http.StatusServiceUnavailable)
		return
	}

	var body struct {
		Code  string `json:"code"`
		State string `json:"state"`
	}

	// GOV-010: GET from Google redirect uses query params, not JSON body
	if r.Method == "GET" {
		body.Code = r.URL.Query().Get("code")
		body.State = r.URL.Query().Get("state")
	} else {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			sendError(w, "invalid request body", http.StatusBadRequest)
			return
		}
	}

	if body.Code == "" {
		sendError(w, "missing authorization code", http.StatusBadRequest)
		return
	}

	// Verify CSRF state parameter (prevents login CSRF)
	if !verifyOAuthState(body.State) {
		sendError(w, "invalid or expired state parameter", http.StatusForbidden)
		return
	}

	// Retrieve CLI web-login challenge (if any) attached to this OAuth state
	cliChallenge := getOAuthChallenge(body.State)

	// SECURITY: redirect_uri from environment ONLY — never from request body.
	redirectURI := os.Getenv("OAUTH_REDIRECT_URI")
	if redirectURI == "" {
		// No hardcoded fallback — OAuth requires exact URI match.
		// If this fires, OAUTH_REDIRECT_URI is not configured.
		sendError(w, "OAuth redirect URI not configured (OAUTH_REDIRECT_URI env var missing)", http.StatusServiceUnavailable)
		return
	}

	var email string
	var err error

	switch provider {
	case "google":
		email, _, err = exchangeGoogleCode(body.Code, redirectURI+"/api/v1/auth/oauth/google")
	case "github":
		email, _, err = exchangeGitHubCode(body.Code, redirectURI+"/api/v1/auth/oauth/github")
	default:
		sendError(w, "unsupported provider", http.StatusBadRequest)
		return
	}

	if err != nil {
		// SECURITY: generic error message — never leak provider internals.
		sendError(w, "authentication failed", http.StatusUnauthorized)
		return
	}

	// Create JWT session
	if err := initJWT(); err != nil {
		sendError(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Determine role by email domain
	role := "operator"
	adminEmails := os.Getenv("ADMIN_EMAILS")
	if adminEmails != "" {
		for _, admin := range strings.Split(adminEmails, ",") {
			if strings.TrimSpace(admin) == email {
				role = "admin"
				break
			}
		}
	}

	now := time.Now()
	claims := jwtClaims{
		Sub:  email,
		Role: role,
		Iat:  now.Unix(),
		Exp:  now.Add(24 * time.Hour).Unix(),
	}

	token, err := signJWT(claims)
	if err != nil {
		sendError(w, "JWT signing failed", http.StatusInternalServerError)
		return
	}

	jwtSessLock.Lock()
	jwtSessions[token] = sessionInfo{
		Token:     token,
		Role:      role,
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}
	jwtSessLock.Unlock()

	// GOV-010: If admin, redirect to TOTP verification instead of direct JWT
	if role == "admin" {
		// Generate a temp auth token (5 min) that allows access to TOTP verify page
		tempClaims := jwtClaims{
			Sub:  email,
			Role: "totp_pending",
			Iat:  now.Unix(),
			Exp:  now.Add(5 * time.Minute).Unix(),
		}
		tempToken, err := signJWT(tempClaims)
		if err == nil {
			http.SetCookie(w, &http.Cookie{
				Name:     "ovav_preauth",
				Value:    tempToken,
				Path:     "/",
				MaxAge:   300,
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
			})
		}
		http.Redirect(w, r, "/login/verify?email="+url.QueryEscape(email), http.StatusFound)
		return
	}

	// CLI web-login via OAuth: notify polling endpoint and serve success page
	if cliChallenge != "" {
		notifyWebLogin(cliChallenge, token, role, claims.Exp)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<!DOCTYPE html>
<html lang="es">
<head><meta charset="UTF-8"><title>OVAV — Sesión verificada</title>
<style>
body{background:#09090b;color:#fafafa;font-family:'Inter',sans-serif;min-height:100vh;display:flex;align-items:center;justify-content:center;}
.box{background:#18181b;border:1px solid #27272a;border-radius:16px;padding:2.5rem;text-align:center;max-width:420px;}
h1{font-size:1.25rem;font-weight:600;margin-bottom:.5rem;}
p{color:#a1a1aa;font-size:.875rem;}
.ok{color:#22c55e;font-size:3rem;margin-bottom:1rem;}
</style>
</head>
<body>
<div class="box">
  <div class="ok">✓</div>
  <h1>Sesión verificada</h1>
  <p>Ya podés cerrar esta pestaña y volver a la terminal.<br>Tu CLI OVAV está autenticado.</p>
</div>
</body>
</html>`))
		return
	}

	// Non-admin: direct JWT
	http.SetCookie(w, &http.Cookie{
		Name:     "ovav_token",
		Value:    token,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

// ── Google OAuth exchange ─────────────────────────────────────────────────────

func exchangeGoogleCode(code, redirectURI string) (email, name string, err error) {
	clientID := os.Getenv("OAUTH_GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("OAUTH_GOOGLE_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		return "", "", fmt.Errorf("Google OAuth not configured")
	}

	// Exchange code for access token
	data := url.Values{
		"code":          {code},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	}

	resp, err := oauthHTTPClient.PostForm("https://oauth2.googleapis.com/token", data)
	if err != nil {
		return "", "", fmt.Errorf("token exchange failed")
	}
	defer resp.Body.Close()

	// Limit response body to 1MB to prevent memory exhaustion
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("token exchange failed")
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", "", fmt.Errorf("parse token response: %w", err)
	}
	if tokenResp.Error != "" {
		return "", "", fmt.Errorf("token error: %s", tokenResp.Error)
	}

	// Fetch user info
	req, _ := http.NewRequest("GET", "https://openidconnect.googleapis.com/v1/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)

	userResp, err := oauthHTTPClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("userinfo request failed")
	}
	defer userResp.Body.Close()

	userBody, _ := io.ReadAll(io.LimitReader(userResp.Body, 1<<20))
	if userResp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("userinfo request failed")
	}

	var userInfo struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(userBody, &userInfo); err != nil {
		return "", "", fmt.Errorf("userinfo parse failed")
	}

	return userInfo.Email, userInfo.Name, nil
}

// ── GitHub OAuth exchange ─────────────────────────────────────────────────────

func exchangeGitHubCode(code, redirectURI string) (email, name string, err error) {
	clientID := os.Getenv("OAUTH_GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("OAUTH_GITHUB_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		return "", "", fmt.Errorf("GitHub OAuth not configured")
	}

	// Exchange code for access token
	data := url.Values{
		"code":          {code},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"redirect_uri":  {redirectURI},
	}

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := oauthHTTPClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("token exchange failed")
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", "", fmt.Errorf("token parse failed")
	}
	if tokenResp.Error != "" {
		return "", "", fmt.Errorf("token exchange failed")
	}
	if tokenResp.AccessToken == "" {
		return "", "", fmt.Errorf("token exchange failed")
	}

	// Fetch user info
	userReq, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	userReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
	userReq.Header.Set("Accept", "application/vnd.github.v3+json")

	userResp, err := oauthHTTPClient.Do(userReq)
	if err != nil {
		return "", "", fmt.Errorf("user request failed")
	}
	defer userResp.Body.Close()

	userBody, _ := io.ReadAll(userResp.Body)
	var userInfo struct {
		Login string `json:"login"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(userBody, &userInfo); err != nil {
		return "", "", fmt.Errorf("user parse failed")
	}

	name = userInfo.Name
	if name == "" {
		name = userInfo.Login
	}

	// GitHub may not return email in /user if it's private — fetch from /user/emails
	if userInfo.Email == "" {
		emailReq, _ := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
		emailReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
		emailReq.Header.Set("Accept", "application/vnd.github.v3+json")

		emailResp, err := oauthHTTPClient.Do(emailReq)
		if err != nil {
			return "", "", fmt.Errorf("email request failed")
		}
		defer emailResp.Body.Close()

		emailBody, _ := io.ReadAll(emailResp.Body)
		var emails []struct {
			Email    string `json:"email"`
			Primary  bool   `json:"primary"`
			Verified bool   `json:"verified"`
		}
		if err := json.Unmarshal(emailBody, &emails); err != nil {
			return "", "", fmt.Errorf("email parse failed")
		}
		for _, e := range emails {
			if e.Primary && e.Verified {
				userInfo.Email = e.Email
				break
			}
		}
		if userInfo.Email == "" && len(emails) > 0 {
			userInfo.Email = emails[0].Email
		}
	}

	return userInfo.Email, name, nil
}
