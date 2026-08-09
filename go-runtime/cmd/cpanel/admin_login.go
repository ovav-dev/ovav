// OVAV cPanel — Admin Login Page (GOV-010)
//
// Serves a branded OVAV login page at /login.
// Replaces Cloudflare Access with cPanel's own authentication.
// Allowlist: CEO only (valid OVAV session required).
//
// GET  /login          → Login page (HTML, public)
// POST /api/v1/auth/admin-login → Validate credential, issue JWT
//
// Security layers:
//   1. Allowlist: only identities with valid OVAV vault key
//   2. Rate limiting: max 5 attempts per minute per IP
//   3. JWT token: 1h expiry, RS256 signed
//   4. Session binding: token tied to machine ID
//
// OVAV Signature: cmd/cpanel/admin_login.go — stabilized 2026-08-02

package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

// ── Admin allowlist ──────────────────────────────────────────────────

var adminAllowlist = map[string]bool{
	"alexander@ovav.dev": true, // CEO — Braka
}

var (
	adminLoginAttempts   = make(map[string][]time.Time)
	adminLoginAttemptsMu sync.Mutex
)

const maxAdminAttempts = 5
const adminAttemptWindow = time.Minute

func checkAdminRateLimit(ip string) bool {
	adminLoginAttemptsMu.Lock()
	defer adminLoginAttemptsMu.Unlock()

	now := time.Now()
	window := now.Add(-adminAttemptWindow)

	attempts := adminLoginAttempts[ip]
	var recent []time.Time
	for _, t := range attempts {
		if t.After(window) {
			recent = append(recent, t)
		}
	}
	adminLoginAttempts[ip] = append(recent, now)
	return len(recent) < maxAdminAttempts
}

// ── Login page (HTML) ────────────────────────────────────────────────

const loginPageHTML = `<!DOCTYPE html>
<html lang="es">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>OVAV · cPanel Access</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', system-ui, sans-serif;
      background: #0a0e17; color: #e2e8f0;
      display: flex; align-items: center; justify-content: center; min-height: 100vh;
    }
    .container { max-width: 420px; width: 90%; }
    .logo { text-align: center; margin-bottom: 2.5rem; }
    .logo h1 {
      font-size: 2rem; font-weight: 700; letter-spacing: -0.02em;
      background: linear-gradient(135deg, #2563eb, #7c3aed);
      -webkit-background-clip: text; -webkit-text-fill-color: transparent;
      background-clip: text;
    }
    .logo p { color: #475569; font-size: 0.85rem; margin-top: 0.3rem; }
    .card {
      background: #111827; border: 1px solid #1e293b;
      border-radius: 12px; padding: 2rem;
    }
    .card h2 { font-size: 1rem; margin-bottom: 1.5rem; color: #64748b; text-align: center; }
    .google-btn {
      display: flex; align-items: center; justify-content: center; gap: 0.75rem;
      width: 100%; padding: 0.85rem; background: #fff; color: #1e293b;
      border: none; border-radius: 8px; font-size: 1rem; font-weight: 600;
      cursor: pointer; transition: all 0.2s;
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', system-ui, sans-serif;
    }
    .google-btn:hover { background: #f1f5f9; transform: translateY(-1px); box-shadow: 0 4px 12px rgba(37,99,235,0.3); }
    .google-btn:disabled { background: #334155; color: #64748b; cursor: not-allowed; transform: none; box-shadow: none; }
    .divider {
      display: flex; align-items: center; margin: 1.5rem 0; color: #334155; font-size: 0.75rem;
    }
    .divider::before, .divider::after { content: ''; flex: 1; border-top: 1px solid #1e293b; }
    .divider span { padding: 0 0.75rem; }
    .vault-section { display: none; }
    .vault-section.show { display: block; }
    label { display: block; font-size: 0.85rem; color: #64748b; margin-bottom: 0.35rem; }
    input {
      width: 100%; padding: 0.75rem 1rem;
      background: #0a0e17; border: 1px solid #1e293b;
      border-radius: 8px; color: #e2e8f0; font-size: 1rem; outline: none;
    }
    input:focus { border-color: #2563eb; }
    .small-btn {
      width: 100%; padding: 0.65rem; margin-top: 0.75rem;
      background: transparent; color: #64748b; border: 1px solid #1e293b;
      border-radius: 8px; font-size: 0.85rem; cursor: pointer;
    }
    .small-btn:hover { color: #94a3b8; border-color: #334155; }
    .error {
      background: #450a0a; border: 1px solid #7f1d1d; color: #fca5a5;
      padding: 0.75rem; border-radius: 8px; margin-bottom: 1rem; font-size: 0.85rem; display: none;
    }
    .error.show { display: block; }
    .footer { text-align: center; margin-top: 1.5rem; color: #334155; font-size: 0.7rem; }
    .shield {
      display: flex; align-items: center; justify-content: center; gap: 0.5rem;
      margin-top: 1rem; color: #1e293b; font-size: 0.7rem;
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="logo">
      <h1>OVAV cPanel</h1>
      <p>Blind Access · CEO Authorization Required</p>
    </div>
    <div class="card">
      <h2>Secure Authentication</h2>
      <div class="error" id="error"></div>
      <a href="/api/v1/auth/oauth-url?provider=google&amp;redirect=1" style="text-decoration:none;">
        <button class="google-btn" id="googleBtn">
          <svg width="20" height="20" viewBox="0 0 24 24"><path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z"/><path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/><path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/><path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/></svg>
          Sign in with Google
        </button>
      </a>
      <div class="divider"><span>or</span></div>
      <button class="small-btn" onclick="toggleVault()">Vault Key Access</button>
      <div class="vault-section" id="vaultSection">
        <form onsubmit="handleVaultLogin(event)" style="margin-top: 1rem;">
          <input type="password" id="token" placeholder="Paste vault key..." autocomplete="off">
          <button type="submit" class="small-btn" style="margin-top:0.5rem;">Access</button>
        </form>
      </div>
    </div>
    <div class="footer">
      OVAV Systems · Internal Infrastructure
    </div>
    <div class="shield">
      <span>🔒</span> Allowlist: CEO Only <span>🔒</span>
    </div>
  </div>
  <script>
    function toggleVault() {
      document.getElementById('vaultSection').classList.toggle('show');
    }
    
    async function handleVaultLogin(e) {
      e.preventDefault();
      const token = document.getElementById('token').value;
      if (!token) return;
      try {
        const resp = await fetch('/api/v1/auth/admin-login', {
          method: 'POST',
          headers: {'Content-Type': 'application/json'},
          body: JSON.stringify({vault_key: token})
        });
        const data = await resp.json();
        if (resp.ok && data.token) {
          localStorage.setItem('ovav_cpanel_token', data.token);
          window.location.href = '/';
        } else {
          showError(data.error || 'Invalid credentials');
        }
      } catch(e) {
        showError('Connection failed');
      }
    }
    
    function showError(msg) {
      const el = document.getElementById('error');
      el.textContent = msg;
      el.classList.add('show');
    }
  </script>
</body>
</html>`

// handleLoginPage serves the login HTML.
func handleLoginPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, loginPageHTML)
}

// ── TOTP Verification Page ──────────────────────────────────────────

const totpPageHTML = `<!DOCTYPE html>
<html lang="es">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>OVAV · TOTP Verification</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', system-ui, sans-serif;
      background: #0a0e17; color: #e2e8f0;
      display: flex; align-items: center; justify-content: center; min-height: 100vh;
    }
    .container { max-width: 420px; width: 90%; }
    .logo { text-align: center; margin-bottom: 2rem; }
    .logo h1 {
      font-size: 1.8rem; font-weight: 700;
      background: linear-gradient(135deg, #2563eb, #7c3aed);
      -webkit-background-clip: text; -webkit-text-fill-color: transparent;
      background-clip: text;
    }
    .logo p { color: #475569; font-size: 0.85rem; margin-top: 0.3rem; }
    .card {
      background: #111827; border: 1px solid #1e293b;
      border-radius: 12px; padding: 2rem;
    }
    .card h2 { font-size: 1rem; margin-bottom: 1rem; color: #94a3b8; text-align: center; }
    .step { text-align: center; color: #64748b; font-size: 0.8rem; margin-bottom: 1.5rem; }
    .step span { color: #2563eb; font-weight: 600; }
    input {
      width: 100%; padding: 0.85rem 1rem; text-align: center; letter-spacing: 0.5em; font-size: 1.5rem;
      background: #0a0e17; border: 1px solid #1e293b; border-radius: 8px; color: #e2e8f0;
      outline: none;
    }
    input:focus { border-color: #2563eb; }
    button {
      width: 100%; padding: 0.75rem; margin-top: 1rem;
      background: #2563eb; color: #fff; border: none; border-radius: 8px;
      font-size: 1rem; font-weight: 600; cursor: pointer;
    }
    button:hover { background: #1d4ed8; }
    button:disabled { background: #1e293b; color: #475569; }
    .error { background: #450a0a; border: 1px solid #7f1d1d; color: #fca5a5;
      padding: 0.75rem; border-radius: 8px; margin-bottom: 1rem; font-size: 0.85rem; display: none; }
    .error.show { display: block; }
    .setup { display: none; margin-top: 1.5rem; padding-top: 1.5rem; border-top: 1px solid #1e293b; }
    .setup.show { display: block; }
    #qr { display: block; margin: 0 auto 1rem; padding: 0.5rem; background: #fff; border-radius: 8px; }
    .secret { color: #64748b; font-size: 0.7rem; text-align: center; word-break: break-all; margin-top: 0.5rem; }
    .footer { text-align: center; margin-top: 1.5rem; color: #334155; font-size: 0.7rem; }
  </style>
</head>
<body>
  <div class="container">
    <div class="logo">
      <h1>OVAV cPanel</h1>
      <p>Two-Factor Authentication</p>
    </div>
    <div class="card">
      <h2>Enter Authenticator Code</h2>
      <div class="step">Step <span>2</span> of 2 — Open Google Authenticator and enter the 6-digit code</div>
      <div class="error" id="error"></div>
      <input type="text" id="code" name="code" placeholder="000000" maxlength="6" inputmode="numeric" autocomplete="one-time-code" autofocus>
      <button onclick="verify()">Verify</button>
      <div class="setup" id="setup">
        <p style="text-align:center;color:#64748b;margin-bottom:0.75rem;">First time? Scan this QR code:</p>
        <canvas id="qr" width="200" height="200"></canvas>
        <button onclick="setupTOTP()" style="background:transparent;border:1px solid #1e293b;color:#64748b;">Generate QR Code</button>
        <p class="secret" id="secretText"></p>
      </div>
    </div>
    <div class="footer">OVAV Systems · Internal Infrastructure</div>
  </div>
  <script>
    const email = new URLSearchParams(window.location.search).get('email') || '';
    
    async function verify() {
      const code = document.getElementById('code').value;
      if (code.length !== 6) return;
      const btn = document.querySelector('button');
      btn.disabled = true; btn.textContent = 'Verifying...';
      try {
        const resp = await fetch('/api/v1/auth/totp/verify', {
          method: 'POST',
          headers: {'Content-Type': 'application/json'},
          body: JSON.stringify({email, code})
        });
        const data = await resp.json();
        if (resp.ok && data.token) {
          window.location.href = '/';
        } else {
          showError(data.error || 'Invalid code');
        }
      } catch(e) { showError('Connection failed'); }
      finally { btn.disabled = false; btn.textContent = 'Verify'; }
    }
    
    async function setupTOTP() {
      try {
        const resp = await fetch('/api/v1/auth/totp/setup', {
          method: 'POST',
          headers: {'Content-Type': 'application/json'},
          body: JSON.stringify({email})
        });
        const data = await resp.json();
        if (data.otpauth) {
          document.getElementById('setup').classList.add('show');
          document.getElementById('secretText').textContent = 'Secret: ' + data.secret;
          // Generate QR code as data URL
          const qrData = 'https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=' + encodeURIComponent(data.otpauth);
          const img = document.createElement('img');
          img.src = qrData; img.width = 200; img.height = 200;
          img.onload = () => {
            document.getElementById('qr').replaceWith(img);
          };
          showError(''); // clear
        } else if (data.status === 'already_configured') {
          showError('TOTP already configured — enter code from your authenticator app');
        }
      } catch(e) { showError('Setup failed'); }
    }
    
    function showError(msg) {
      const el = document.getElementById('error');
      el.textContent = msg;
      el.className = 'error' + (msg ? ' show' : '');
    }
    
    // Auto-try setup on page load
    setupTOTP();
  </script>
</body>
</html>`

func handleTOTPPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, totpPageHTML)
}

// handleAdminLogin validates vault key against the active OVAV session.
func handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}

	// Rate limit
	ip := r.RemoteAddr
	if !checkAdminRateLimit(ip) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{
			"error": "Too many attempts — wait 1 minute",
		})
		return
	}

	var req struct {
		VaultKey string `json:"vault_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	if req.VaultKey == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Vault key required"})
		return
	}

	// Validate against active OVAV session
	session, ok := loadOVAVSession()
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "No active OVAV session — run 'ovav login' first",
		})
		return
	}

	// Verify vault key hash
	keyHash := sha256Hex(req.VaultKey)
	if keyHash != session.VaultKeyHash {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid vault key"})
		return
	}

	// Generate JWT
	claims := jwtClaims{
		Sub:  session.MachineID,
		Role: session.Role,
		Iat:  time.Now().Unix(),
		Exp:  time.Now().Add(1 * time.Hour).Unix(),
	}
	token, err := signJWT(claims)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Token generation failed"})
		return
	}

	// Set JWT as httpOnly cookie so browser auto-authenticates
	http.SetCookie(w, &http.Cookie{
		Name:     "ovav_token",
		Value:    token,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token":      token,
		"role":       session.Role,
		"expires_in": 3600,
	})
}

// sha256Hex returns the hex-encoded SHA-256 hash of data.
func sha256Hex(data string) string {
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

// ── OAuth URL Generator ──────────────────────────────────────────────

// handleOAuthURL generates the Google OAuth authorization URL with CSRF state.
// GET /api/v1/auth/oauth-url?provider=google[&challenge=TOKEN]
// If ?redirect=1 is set, redirects to Google directly instead of returning JSON.
// The challenge param is stored alongside the OAuth state for CLI web-login flows.
func handleOAuthURL(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	if provider != "google" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported provider"})
		return
	}

	clientID := os.Getenv("OAUTH_GOOGLE_CLIENT_ID")
	if clientID == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "OAuth not configured"})
		return
	}

	redirectURI := os.Getenv("OAUTH_REDIRECT_URI")
	if redirectURI == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "OAuth redirect URI not configured (OAUTH_REDIRECT_URI env var missing)"})
		return
	}

	// CLI web-login challenge passed through from login-portal page
	challenge := r.URL.Query().Get("challenge")

	// Generate CSRF state with optional challenge attachment
	state := generateOAuthState(challenge)

	// Build Google OAuth URL
	authURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s/api/v1/auth/oauth/google&response_type=code&scope=openid+email+profile&state=%s&access_type=offline&prompt=select_account",
		clientID, redirectURI, url.QueryEscape(state),
	)

	// GOV-010: Direct redirect mode for browser buttons (avoids JS redirect issues)
	if r.URL.Query().Get("redirect") == "1" {
		http.Redirect(w, r, authURL, http.StatusFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"url": authURL})
}

// ── Challenge Token (Cloudflare-style breadcrumb security) ──────────

// loginChallenge generates a short-lived JWT that must be presented
// to access the login page. Pattern: Cloudflare Access meta JWT in URL.
// GET /api/v1/auth/login-challenge
func handleLoginChallenge(w http.ResponseWriter, r *http.Request) {
	// Ensure JWT system is initialized
	if err := initJWT(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "JWT not initialized"})
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
		Sub:  fmt.Sprintf("challenge:%s:%s", ip, nonce),
		Role: "challenge",
		Iat:  time.Now().Unix(),
		Exp:  time.Now().Add(60 * time.Second).Unix(),
	}

	token, err := signJWT(claims)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "challenge generation failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"challenge":  token,
		"expires_in": "60",
	})
}

// validateChallenge verifies the challenge JWT before serving login page.
func validateChallenge(r *http.Request) bool {
	challenge := r.URL.Query().Get("challenge")
	if challenge == "" {
		return false
	}

	claims, err := verifyJWT(challenge)
	if err != nil || claims == nil {
		return false
	}

	if claims.Role != "challenge" {
		return false
	}

	if time.Now().Unix() > claims.Exp {
		return false
	}

	return true
}
