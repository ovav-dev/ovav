// OVAV cPanel v5.0 — Auth handlers (JWT RS256).
//
// POST /api/v1/auth/login   — Authenticate, return JWT
// GET  /api/v1/auth/session  — Return active session info
//
// RS256 JWT using crypto/rsa + crypto/sha256. Stdlib only.
// Keys auto-generated on first run. Stored in .ovav/runtime/cpanel_jwt.*

package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ── OVAV Session integration ──────────────────────────────────────────────────

// OVAVSession mirrors the CLI session struct for token validation.
type OVAVSession struct {
	VaultKeyHash string `json:"vault_key_hash"`
	MachineID    string `json:"machine_id"`
	CreatedAt    string `json:"created_at"`
	IdentityID   string `json:"identity_id,omitempty"`
	Role         string `json:"role,omitempty"`
	Level        int    `json:"level,omitempty"`
	Name         string `json:"name,omitempty"`
}

// loadOVAVSession reads the active OVAV Systems session from disk.
func loadOVAVSession() (OVAVSession, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return OVAVSession{}, false
	}
	sessionPath := filepath.Join(home, ".local", "share", "ovav", "session")
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		return OVAVSession{}, false
	}
	var s OVAVSession
	if err := json.Unmarshal(data, &s); err != nil {
		return OVAVSession{}, false
	}
	// Verify session is not expired (24h TTL)
	if s.CreatedAt != "" {
		created, err := time.Parse(time.RFC3339, s.CreatedAt)
		if err == nil && time.Since(created) > 24*time.Hour {
			return OVAVSession{}, false
		}
	}
	return s, true
}

// ── JWT key management ────────────────────────────────────────────────────────

var (
	jwtPrivateKey *rsa.PrivateKey
	jwtKeyLock    sync.RWMutex
	jwtSessions   = make(map[string]sessionInfo)
	jwtSessLock   sync.RWMutex

	// pendingWebLogins tracks in-flight web login flows (challenge → result).
	pendingWebLogins = make(map[string]*webLoginResult)
	webLoginLock     sync.RWMutex
)

type webLoginResult struct {
	mu       sync.RWMutex
	complete bool
	jwt      string
	role     string
	exp      int64
	err      string
}

type sessionInfo struct {
	Token     string    `json:"token"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func initJWT() error {
	jwtKeyLock.Lock()
	defer jwtKeyLock.Unlock()

	if jwtPrivateKey != nil {
		return nil
	}

	keyPath := filepath.Join(RepoRoot, ".ovav", "runtime", "cpanel_jwt.key")

	// Try to load existing key
	if keyData, err := os.ReadFile(keyPath); err == nil {
		block, _ := pem.Decode(keyData)
		if block != nil {
			key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err == nil {
				if rsaKey, ok := key.(*rsa.PrivateKey); ok {
					jwtPrivateKey = rsaKey
					startSessionCleanup()
					return nil
				}
			}
		}
	}

	// Generate new key pair
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	// Save private key
	privBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privBytes,
	})
	os.MkdirAll(filepath.Dir(keyPath), 0700)
	if err := os.WriteFile(keyPath, privPEM, 0600); err != nil {
		return err
	}

	// Save public key
	pubPath := filepath.Join(RepoRoot, ".ovav", "runtime", "cpanel_jwt.pub")
	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return err
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})
	os.WriteFile(pubPath, pubPEM, 0644)

	jwtPrivateKey = key
	startSessionCleanup()
	return nil
}

// startSessionCleanup launches a background goroutine that removes expired
// sessions every hour, preventing unbounded memory growth.
func startSessionCleanup() {
	go func() {
		for {
			time.Sleep(1 * time.Hour)
			jwtSessLock.Lock()
			now := time.Now()
			for token, sess := range jwtSessions {
				if now.After(sess.ExpiresAt) {
					delete(jwtSessions, token)
				}
			}
			jwtSessLock.Unlock()
		}
	}()
}

// ── JWT encode/decode ─────────────────────────────────────────────────────────

type jwtClaims struct {
	Sub    string `json:"sub"` // user ID or "cpanel-user"
	Role   string `json:"role"`
	UserID string `json:"user_id,omitempty"` // for user account JWTs
	Email  string `json:"email,omitempty"`   // for user account JWTs
	Iat    int64  `json:"iat"`
	Exp    int64  `json:"exp"`
}

func base64urlEncode(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

func base64urlDecode(s string) ([]byte, error) {
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}

func signJWT(claims jwtClaims) (string, error) {
	header := map[string]string{"alg": "RS256", "typ": "JWT"}
	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	headerB64 := base64urlEncode(headerJSON)
	claimsB64 := base64urlEncode(claimsJSON)

	signingInput := headerB64 + "." + claimsB64
	hashed := sha256.Sum256([]byte(signingInput))

	jwtKeyLock.RLock()
	key := jwtPrivateKey
	jwtKeyLock.RUnlock()

	if key == nil {
		return "", &authError{"JWT not initialized"}
	}

	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hashed[:])
	if err != nil {
		return "", err
	}

	sigB64 := base64urlEncode(signature)
	return signingInput + "." + sigB64, nil
}

func verifyJWT(token string) (*jwtClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, &authError{"invalid token format"}
	}

	claimsJSON, err := base64urlDecode(parts[1])
	if err != nil {
		return nil, &authError{"invalid token encoding"}
	}
	var claims jwtClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, &authError{"invalid token claims"}
	}

	if time.Now().Unix() > claims.Exp {
		return nil, &authError{"token expired"}
	}

	signingInput := parts[0] + "." + parts[1]
	hashed := sha256.Sum256([]byte(signingInput))
	signature, err := base64urlDecode(parts[2])
	if err != nil {
		return nil, &authError{"invalid signature encoding"}
	}

	jwtKeyLock.RLock()
	key := jwtPrivateKey
	jwtKeyLock.RUnlock()

	if key == nil {
		return nil, &authError{"JWT not initialized"}
	}

	pubKey := &key.PublicKey

	if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hashed[:], signature); err != nil {
		return nil, &authError{"invalid signature"}
	}

	return &claims, nil
}

type authError struct{ msg string }

func (e *authError) Error() string { return e.msg }

// ── Rate limiting ─────────────────────────────────────────────────────────────

var (
	rateLimitStore   = make(map[string]*rateLimiter)
	rateLimitStoreMu sync.Mutex
)

type rateLimiter struct {
	attempts int
	resetAt  time.Time
}

// checkRateLimit returns true if the request should be allowed.
// Allows 5 attempts per minute per IP.
func checkRateLimit(ip string) bool {
	rateLimitStoreMu.Lock()
	defer rateLimitStoreMu.Unlock()

	now := time.Now()
	rl, ok := rateLimitStore[ip]
	if !ok || now.After(rl.resetAt) {
		rateLimitStore[ip] = &rateLimiter{attempts: 1, resetAt: now.Add(time.Minute)}
		return true
	}
	if rl.attempts >= 5 {
		return false
	}
	rl.attempts++
	return true
}

// ResetRateLimiterForTesting clears the rate limiter store.
// Must be called between test runs when using -count=N to prevent
// state leakage across iterations (FLAKY-01).
func ResetRateLimiterForTesting() {
	rateLimitStoreMu.Lock()
	defer rateLimitStoreMu.Unlock()
	rateLimitStore = make(map[string]*rateLimiter)
}

func handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	// Rate limiting: 5 attempts per minute per IP
	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = strings.Split(fwd, ",")[0]
	}
	if !checkRateLimit(strings.TrimSpace(ip)) {
		sendError(w, "too many attempts — try again in 1 minute", http.StatusTooManyRequests)
		return
	}

	if err := initJWT(); err != nil {
		sendError(w, "JWT init failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Enforce minimum token length for brute-force resistance.
	if len(body.Token) < 32 {
		sendError(w, "token must be at least 32 characters", http.StatusUnauthorized)
		return
	}

	// Validate token against OVAV Systems session.
	// Token must match the vault_key_hash from an active session.
	sess, ok := loadOVAVSession()
	if !ok {
		sendError(w, "no active OVAV session — run 'ovav login' first", http.StatusUnauthorized)
		return
	}
	if !strings.EqualFold(body.Token, sess.VaultKeyHash) {
		sendError(w, "invalid token", http.StatusUnauthorized)
		return
	}

	// Role from session — never from token prefix
	role := sess.Role
	if role == "" {
		role = "operator"
	}

	now := time.Now()
	claims := jwtClaims{
		Sub:  "cpanel-user",
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

	sendOK(w, map[string]interface{}{
		"token":      token,
		"role":       role,
		"expires_at": claims.Exp,
		"token_type": "Bearer",
	})
}

func handleAuthSession(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")

	if token == "" || token == authHeader {
		sendError(w, "no token provided", http.StatusUnauthorized)
		return
	}

	claims, err := verifyJWT(token)
	if err != nil {
		sendError(w, err.Error(), http.StatusUnauthorized)
		return
	}

	jwtSessLock.RLock()
	_, ok := jwtSessions[token]
	jwtSessLock.RUnlock()

	if !ok {
		sendError(w, "session not found", http.StatusUnauthorized)
		return
	}

	sendOK(w, map[string]interface{}{
		"valid":      true,
		"role":       claims.Role,
		"sub":        claims.Sub,
		"user_id":    claims.UserID,
		"email":      claims.Email,
		"expires_at": claims.Exp,
	})
}

// ── User account auth ─────────────────────────────────────────────────────────

// handleRegister creates a new user account.
// POST /api/v1/auth/register { email, password, name }
func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	body.Email = strings.TrimSpace(body.Email)
	if body.Email == "" || body.Password == "" {
		sendError(w, "email and password are required", http.StatusBadRequest)
		return
	}

	store := getUserStore()
	user, err := store.Register(body.Email, body.Password, body.Name)
	if err != nil {
		sendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Issue JWT for the new user
	if err := initJWT(); err != nil {
		sendError(w, "JWT init failed", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	claims := jwtClaims{
		Sub:    user.ID,
		Role:   "user",
		UserID: user.ID,
		Email:  user.Email,
		Iat:    now.Unix(),
		Exp:    now.Add(24 * time.Hour).Unix(),
	}

	token, err := signJWT(claims)
	if err != nil {
		sendError(w, "JWT signing failed", http.StatusInternalServerError)
		return
	}

	jwtSessLock.Lock()
	jwtSessions[token] = sessionInfo{
		Token:     token,
		Role:      "user",
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}
	jwtSessLock.Unlock()

	sendOK(w, map[string]interface{}{
		"token":      token,
		"user_id":    user.ID,
		"email":      user.Email,
		"name":       user.Name,
		"tier":       user.Tier,
		"slots":      TierSlots[user.Tier],
		"expires_at": claims.Exp,
		"token_type": "Bearer",
	})
}

// handleUserLogin authenticates a user with email + password.
// POST /api/v1/auth/user-login { email, password, challenge? }
// P1-B: Detects login anomalies (new IP, impossible travel, brute force).
// P2-A: Emergency bypass code when CF Access is completely unavailable.
// If "challenge" is provided, the result is stored for CLI polling.
func handleUserLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Real client IP (CF passes this in header)
	ip := helperIP(r)
	if !checkRateLimitLogin(strings.TrimSpace(ip)) {
		sendError(w, "too many login attempts — try again in 1 minute", http.StatusTooManyRequests)
		return
	}

	var body struct {
		Email         string `json:"email"`
		Password      string `json:"password"`
		TOTPCode      string `json:"totp_code"`      // step-up if anomaly detected
		EmergencyCode string `json:"emergency_code"` // P2-A emergency bypass
		Challenge     string `json:"challenge"`      // web login challenge token
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if body.Email == "" || body.Password == "" {
		sendError(w, "email and password are required", http.StatusUnauthorized)
		return
	}

	// ── P2-A: Emergency bypass code ────────────────────────────────────────────
	// If env var EMERGENCY_BYPASS_CODE is set, using it bypasses password auth.
	// This is for when Cloudflare Access is DOWN and normal login is impossible.
	// The code itself is stored as a Fly.io secret, never in source code.
	emergencyBypass := os.Getenv("EMERGENCY_BYPASS_CODE")
	usedEmergency := emergencyBypass != "" && body.EmergencyCode == emergencyBypass

	store := getUserStore()
	user, err := store.Authenticate(body.Email, body.Password)
	if err != nil && !usedEmergency {
		// Failed normal auth — register failure for anomaly detection
		RegisterLoginAttempt("", ip, "", "", "", false)
		AuditAuth(r.Context(), "", body.Email, ip,
			"auth.user_login", "/api/v1/auth/login", "denied", http.StatusUnauthorized, "invalid credentials")
		sendError(w, "invalid email or password", http.StatusUnauthorized)
		return
	}

	// ── P1-B: Login anomaly detection ──────────────────────────────────────────
	userAgent := r.Header.Get("User-Agent")
	risk, riskDesc := DetectLoginAnomaly("", ip, "", userAgent)

	// If user authenticated normally (not emergency), apply anomaly rules
	if !usedEmergency && risk > 70 && body.TOTPCode == "" {
		// High risk + no TOTP = require step-up auth
		RegisterLoginAttempt("", ip, "", "", userAgent, false)
		AuditAuth(r.Context(), "", body.Email, ip,
			"auth.user_login.anomaly", "/api/v1/auth/login", "step_up", http.StatusForbidden,
			fmt.Sprintf("risk=%d (%s)", risk, riskDesc))
		sendError(w, fmt.Sprintf("step-up authentication required — risk score %d: %s. Provide totp_code.", risk, riskDesc), http.StatusForbidden)
		return
	}

	// Step-up TOTP verification for elevated risk
	if risk > 30 && !usedEmergency {
		if body.TOTPCode == "" {
			RegisterLoginAttempt("", ip, "", "", userAgent, false)
			AuditAuth(r.Context(), "", body.Email, ip,
				"auth.user_login.anomaly", "/api/v1/auth/login", "step_up_required", http.StatusForbidden,
				fmt.Sprintf("risk=%d — TOTP required", risk))
			sendError(w, fmt.Sprintf("step-up auth required (risk=%d). Provide totp_code.", risk), http.StatusForbidden)
			return
		}
		// Verify TOTP (stub — implement with github.com/pquerna/otp in production)
		if !verifyTOTP(user.TOTP, body.TOTPCode) {
			RegisterLoginAttempt("", ip, "", "", userAgent, false)
			AuditAuth(r.Context(), user.ID, body.Email, ip,
				"auth.user_login.totp_fail", "/api/v1/auth/login", "denied", http.StatusUnauthorized, "invalid TOTP")
			sendError(w, "invalid TOTP code", http.StatusUnauthorized)
			return
		}
	}

	if err := initJWT(); err != nil {
		sendError(w, "JWT init failed", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	if user == nil && usedEmergency {
		// Emergency bypass: create a minimal placeholder user record
		user = &User{ID: "emergency-" + ip, Email: body.Email, Role: "operator"}
	}

	claims := jwtClaims{
		Sub:    user.ID,
		Role:   user.Role,
		UserID: user.ID,
		Email:  user.Email,
		Iat:    now.Unix(),
		Exp:    now.Add(24 * time.Hour).Unix(),
	}

	token, err := signJWT(claims)
	if err != nil {
		sendError(w, "JWT signing failed", http.StatusInternalServerError)
		return
	}

	jwtSessLock.Lock()
	jwtSessions[token] = sessionInfo{
		Token:     token,
		Role:      user.Role,
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}
	jwtSessLock.Unlock()

	// If this login is for a CLI web-login flow, notify the pending entry.
	if body.Challenge != "" {
		notifyWebLogin(body.Challenge, token, user.Role, claims.Exp)
	}

	// Register successful login
	RegisterLoginAttempt(user.ID, ip, "", "", userAgent, true)

	loginType := "normal"
	if usedEmergency {
		loginType = "emergency_bypass"
	} else if risk > 30 {
		loginType = "step_up_auth"
	}

	AuditAuth(r.Context(), user.ID, user.Email, ip,
		"auth.user_login."+loginType, "/api/v1/auth/login", "ok", http.StatusOK,
		fmt.Sprintf("risk=%d", risk))

	w.Header().Set("X-OVAV-Risk-Score", fmt.Sprintf("%d", risk))
	sendOK(w, map[string]interface{}{
		"token":      token,
		"user_id":    user.ID,
		"email":      user.Email,
		"name":       user.Name,
		"tier":       user.Tier,
		"slots":      TierSlots[user.Tier],
		"expires_at": claims.Exp,
		"token_type": "Bearer",
		"login_type": loginType,
		"risk_score": risk,
	})
}

// verifyTOTP verifies a TOTP code against a user's TOTP secret.
// Stub implementation — in production use github.com/pquerna/otp.
// Returns true for any 6-digit code when TOTP secret is non-empty (demo mode).
func verifyTOTP(totpSecret, code string) bool {
	if totpSecret == "" {
		return true // No TOTP configured — skip verification
	}
	// TODO: implement real TOTP with github.com/pquerna/otp
	// Real implementation: otp totp := totp.New(totpSecret); return totp.Validate(code)
	return len(code) == 6 && code >= "000000" && code <= "999999"
}

// handleMe returns the authenticated user's info.
// GET /api/v1/auth/me
func handleMe(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" || token == authHeader {
		sendError(w, "no token", http.StatusUnauthorized)
		return
	}

	claims, err := verifyJWT(token)
	if err != nil {
		sendError(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if claims.UserID == "" {
		sendError(w, "not a user session", http.StatusUnauthorized)
		return
	}

	store := getUserStore()
	user := store.GetByID(claims.UserID)
	if user == nil {
		sendError(w, "user not found", http.StatusNotFound)
		return
	}

	sendOK(w, map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
		"name":    user.Name,
		"tier":    user.Tier,
		"plan":    user.Plan,
		"slots":   TierSlots[user.Tier],
	})
}

// ── Web Login Flow (CLI browser-based auth) ────────────────────────────────

// notifyWebLogin marks a pending web login as completed with the given JWT.
func notifyWebLogin(challenge, token, role string, exp int64) {
	webLoginLock.Lock()
	defer webLoginLock.Unlock()
	if result, ok := pendingWebLogins[challenge]; ok {
		result.mu.Lock()
		result.complete = true
		result.jwt = token
		result.role = role
		result.exp = exp
		result.mu.Unlock()
	}
}

// registerPendingLogin creates a pending web login entry for a challenge token.
// Returns true if registered, false if the challenge is invalid or expired.
func registerPendingLogin(challenge string) bool {
	// Verify the challenge token is valid (but we don't enforce its 60s expiry
	// for the login flow — we use a separate 10min window for the web login)
	claims, err := verifyJWT(challenge)
	if err != nil {
		return false
	}
	if claims.Role != "challenge" {
		return false
	}

	webLoginLock.Lock()
	defer webLoginLock.Unlock()
	pendingWebLogins[challenge] = &webLoginResult{}
	// Expire stale entries every time we add a new one (simple cleanup)
	go func() {
		time.Sleep(10 * time.Minute)
		webLoginLock.Lock()
		delete(pendingWebLogins, challenge)
		webLoginLock.Unlock()
	}()
	return true
}

// resolveWebLogin returns the current state of a pending web login.
func resolveWebLogin(challenge string) (complete bool, jwt, role string, exp int64, errMsg string) {
	webLoginLock.RLock()
	result, ok := pendingWebLogins[challenge]
	webLoginLock.RUnlock()
	if !ok {
		return false, "", "", 0, "challenge not found or expired"
	}
	result.mu.RLock()
	defer result.mu.RUnlock()
	return result.complete, result.jwt, result.role, result.exp, result.err
}

// GET /api/v1/auth/login-status?challenge=TOKEN
// Polls the status of a pending web login initiated by 'ovav login'.
// Returns: {status: "pending"|"complete"|"error", jwt?, role?, expires_at?, error?}.
func handleLoginStatus(w http.ResponseWriter, r *http.Request) {
	challenge := r.URL.Query().Get("challenge")
	if challenge == "" {
		sendError(w, "challenge parameter required", http.StatusBadRequest)
		return
	}

	complete, jwt, role, exp, errMsg := resolveWebLogin(challenge)
	if errMsg != "" && !complete {
		sendError(w, errMsg, http.StatusNotFound)
		return
	}

	status := "pending"
	if complete {
		status = "complete"
	}
	resp := map[string]interface{}{
		"status": status,
	}
	if complete {
		resp["jwt"] = jwt
		resp["role"] = role
		resp["expires_at"] = exp
	}
	if errMsg != "" {
		resp["error"] = errMsg
	}
	sendOK(w, resp)
}

// GET /api/v1/auth/login-challenge — creates a pending web login challenge.
// CLI calls this to get a challenge token before opening the browser.
func handleLoginChallengeWeb(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Reuse the existing challenge logic but register it for polling.
	// We delegate to the existing handleLoginChallenge to get the token.
	// Since handleLoginChallenge writes its own JSON response, we need to
	// capture its output and then register the pending login.
	//
	// Instead, inline the challenge creation here.
	if err := initJWT(); err != nil {
		sendError(w, "JWT not initialized", http.StatusInternalServerError)
		return
	}

	nonceBytes := make([]byte, 16)
	rand.Read(nonceBytes)
	nonce := base64.URLEncoding.EncodeToString(nonceBytes)
	ip := r.RemoteAddr
	if fwd := r.Header.Get("CF-Connecting-IP"); fwd != "" {
		ip = fwd
	}

	claims := jwtClaims{
		Sub:  fmt.Sprintf("web-login:%s:%s", ip, nonce),
		Role: "challenge",
		Iat:  time.Now().Unix(),
		Exp:  time.Now().Add(60 * time.Second).Unix(),
	}
	token, err := signJWT(claims)
	if err != nil {
		sendError(w, "challenge generation failed", http.StatusInternalServerError)
		return
	}

	if !registerPendingLogin(token) {
		sendError(w, "invalid challenge token", http.StatusBadRequest)
		return
	}

	sendOK(w, map[string]interface{}{
		"challenge":  token,
		"expires_in": 600, // 10 minutes for web login flow
		"login_url":  "https://d678beea.ovav.dev/login-portal?challenge=" + token,
	})
}

// checkRateLimitLogin is a separate rate limiter for login attempts (10 per minute).
var loginAttempts = make(map[string]*rateLimiter)
var loginAttemptsMu sync.Mutex

func checkRateLimitLogin(ip string) bool {
	loginAttemptsMu.Lock()
	defer loginAttemptsMu.Unlock()
	now := time.Now()
	rl, ok := loginAttempts[ip]
	if !ok || now.After(rl.resetAt) {
		loginAttempts[ip] = &rateLimiter{attempts: 1, resetAt: now.Add(time.Minute)}
		return true
	}
	if rl.attempts >= 10 {
		return false
	}
	rl.attempts++
	return true
}
