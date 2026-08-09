// OVAV cPanel v5.0 — Static file serving.
//
// Serves the Vite-built React SPA from tools/cpanel/static/dist/.
// Falls back to index.html for SPA client-side routing.

package main

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// spaIndexPath returns the full path to the SPA index.html.
func spaIndexPath() string {
	return filepath.Join(RepoRoot, "tools/cpanel/static/dist/index.html")
}

// spaIngress handles the root path — serves the SPA.
func spaIngress(w http.ResponseWriter, r *http.Request) {
	// Only serve index.html for GET /
	if r.URL.Path != "/" {
		// Fall through to static asset serving
		serveAsset(w, r)
		return
	}
	serveSPAIndex(w)
}

// serveSPAIndex serves the admin dashboard after successful login.
func serveSPAIndex(w http.ResponseWriter) {
	path := spaIndexPath()
	data, err := os.ReadFile(path)
	if err != nil {
		// GOV-010: Serve admin dashboard when SPA is not built
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(adminDashboardHTML))
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

const adminDashboardHTML = `<!DOCTYPE html>
<html lang="es">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>OVAV cPanel · Dashboard</title>
  <style>
    * { margin:0; padding:0; box-sizing:border-box; }
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', system-ui, sans-serif; background: #0a0e17; color: #e2e8f0; padding: 2rem; }
    .header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 2rem; }
    .header h1 { font-size: 1.5rem; background: linear-gradient(135deg,#2563eb,#7c3aed); -webkit-background-clip:text; -webkit-text-fill-color:transparent; background-clip:text; }
    .status { display: inline-flex; align-items: center; gap: 0.5rem; padding: 0.25rem 0.75rem; background: #064e3b; border: 1px solid #065f46; border-radius: 20px; font-size: 0.8rem; color: #6ee7b7; }
    .status::before { content: ''; width: 8px; height: 8px; background: #34d399; border-radius: 50%; }
    .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 1rem; }
    .card { background: #111827; border: 1px solid #1e293b; border-radius: 12px; padding: 1.5rem; }
    .card h3 { font-size: 0.9rem; color: #64748b; margin-bottom: 1rem; text-transform: uppercase; letter-spacing: 0.05em; }
    .card .value { font-size: 2rem; font-weight: 700; }
    .card .label { font-size: 0.85rem; color: #475569; margin-top: 0.25rem; }
    .green { color: #34d399; }
    .blue { color: #60a5fa; }
    .purple { color: #a78bfa; }
    .links { margin-top: 2rem; display: flex; gap: 1rem; flex-wrap: wrap; }
    .links a { padding: 0.5rem 1rem; background: #1e293b; border: 1px solid #334155; border-radius: 8px; color: #94a3b8; text-decoration: none; font-size: 0.85rem; transition: all 0.2s; }
    .links a:hover { border-color: #2563eb; color: #e2e8f0; }
    .footer { margin-top: 3rem; text-align: center; color: #1e293b; font-size: 0.7rem; }
  </style>
</head>
<body>
  <div class="header">
    <h1>OVAV cPanel</h1>
    <div class="status">Authenticated · CEO</div>
  </div>
  <div class="grid">
    <div class="card">
      <h3>Sync Engine</h3>
      <div class="value green">179</div>
      <div class="label">items ready for distribution</div>
    </div>
    <div class="card">
      <h3>Product Version</h3>
      <div class="value blue">v1.0.0</div>
      <div class="label">current · update_ready: false</div>
    </div>
    <div class="card">
      <h3>Tunnel Status</h3>
      <div class="value purple">Active</div>
      <div class="label">d678beea.ovav.dev · Cloudflare</div>
    </div>
  </div>
  <div class="links">
    <a href="/login">← Login Page</a>
    <a href="/api/v1/product/sync/status">Sync Status (API)</a>
    <a href="/api/v1/product/version">Product Version (API)</a>
    <a href="/api/v1/health">Health Check</a>
  </div>
  <div class="footer">OVAV Systems · GOV-010 · Internal Infrastructure</div>
</body>
</html>`

// serveAsset handles /assets/* and /css/* static file requests.
func serveAsset(w http.ResponseWriter, r *http.Request) {
	rel := strings.TrimPrefix(r.URL.Path, "/")

	// URL-decode to catch encoded path traversal attempts (%2e%2e, %252e%252e, etc.)
	decoded, err := url.PathUnescape(rel)
	if err != nil {
		decoded = rel
	}

	// Defense-in-depth: prevent path traversal (raw and URL-decoded)
	if strings.Contains(rel, "..") || strings.Contains(decoded, "..") {
		sendError(w, "invalid path", http.StatusBadRequest)
		return
	}
	// Also check for backslash-based traversal (Windows compatibility)
	if strings.Contains(rel, "\\") || strings.Contains(decoded, "\\") {
		sendError(w, "invalid path", http.StatusBadRequest)
		return
	}

	// Try dist/ first
	baseDir := filepath.Join(RepoRoot, "tools", "cpanel", "static", "dist")
	filePath := filepath.Join(baseDir, rel)

	// Verify path is within the expected directory
	if !strings.HasPrefix(filepath.Clean(filePath), filepath.Clean(baseDir)) {
		sendError(w, "forbidden", http.StatusForbidden)
		return
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		// Try static/ root (for css/design-tokens.css etc.)
		baseDir = filepath.Join(RepoRoot, "tools", "cpanel", "static")
		filePath = filepath.Join(baseDir, rel)
		if !strings.HasPrefix(filepath.Clean(filePath), filepath.Clean(baseDir)) {
			sendError(w, "forbidden", http.StatusForbidden)
			return
		}
		data, err = os.ReadFile(filePath)
	}

	if err != nil {
		// SPA fallback ONLY for non-API paths (client-side routing).
		// API paths must return proper JSON errors, never HTML.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			sendError(w, "endpoint not found: "+r.URL.Path, http.StatusNotFound)
			return
		}
		serveSPAIndex(w)
		return
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	ct, ok := mimeTypes[ext]
	if !ok {
		ct = "application/octet-stream"
	}

	w.Header().Set("Content-Type", ct)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// handleLoginPortal serves a standalone login page for CLI web-login flow.
// This page is served BY the Go backend, bypassing the React SPA and CF Access.
// Pattern: CLI opens this page, user authenticates, CLI polls for completion.
func handleLoginPortal(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	challenge := r.URL.Query().Get("challenge")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.WriteHeader(http.StatusOK)

	// Challenge embedded in page so JS can use it
	challengeEsc := escapeHTML(challenge)
	w.Write([]byte(`<!DOCTYPE html>
<html lang="es">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>OVAV — Iniciar Sesión</title>
<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
body {
  background: #09090b;
  color: #fafafa;
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
}
.container {
  width: 100%;
  max-width: 420px;
  padding: 2rem;
}
.card {
  background: #18181b;
  border: 1px solid #27272a;
  border-radius: 16px;
  padding: 2.5rem;
  box-shadow: 0 0 60px rgba(124, 58, 237, 0.1);
}
.logo {
  text-align: center;
  margin-bottom: 2rem;
}
.logo-text {
  font-size: 1.5rem;
  font-weight: 700;
  background: linear-gradient(135deg, #7c3aed, #a78bfa);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}
h1 {
  font-size: 1.25rem;
  font-weight: 600;
  text-align: center;
  margin-bottom: 0.25rem;
}
.subtitle {
  color: #71717a;
  text-align: center;
  margin-bottom: 2rem;
  font-size: 0.875rem;
}
.field {
  margin-bottom: 1.25rem;
}
label {
  display: block;
  font-size: 0.875rem;
  font-weight: 500;
  margin-bottom: 0.5rem;
  color: #a1a1aa;
}
input {
  width: 100%;
  padding: 0.625rem 0.875rem;
  background: #09090b;
  border: 1px solid #3f3f46;
  border-radius: 8px;
  color: #fafafa;
  font-size: 0.9375rem;
  transition: border-color 0.2s, box-shadow 0.2s;
  outline: none;
}
input:focus {
  border-color: #7c3aed;
  box-shadow: 0 0 0 3px rgba(124, 58, 237, 0.2);
}
input::placeholder { color: #52525b; }
.btn {
  width: 100%;
  padding: 0.75rem;
  background: #7c3aed;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 0.9375rem;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s, transform 0.1s;
}
.btn:hover { background: #6d28d9; }
.btn:active { transform: scale(0.98); }
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
.divider {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin: 1.25rem 0;
  color: #52525b;
  font-size: 0.8125rem;
}
.divider::before, .divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: #27272a;
}
.oauth-btn {
  width: 100%;
  padding: 0.625rem;
  background: #fff;
  color: #18181b;
  border: 1px solid #3f3f46;
  border-radius: 8px;
  font-size: 0.9375rem;
  font-weight: 500;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.625rem;
  transition: background 0.2s, border-color 0.2s;
  text-decoration: none;
}
.oauth-btn:hover { background: #f4f4f5; border-color: #52525b; }
.oauth-btn svg { flex-shrink: 0; }
.error {
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid #ef4444;
  border-radius: 8px;
  padding: 0.75rem;
  margin-bottom: 1rem;
  color: #fca5a5;
  font-size: 0.875rem;
  display: none;
}
.success {
  background: rgba(34, 197, 94, 0.1);
  border: 1px solid #22c55e;
  border-radius: 8px;
  padding: 0.75rem;
  margin-bottom: 1rem;
  color: #86efac;
  font-size: 0.875rem;
  text-align: center;
  display: none;
}
.spinner {
  display: none;
  text-align: center;
  margin: 1rem 0;
}
.spinner::after {
  content: '';
  width: 24px;
  height: 24px;
  border: 2px solid #27272a;
  border-top-color: #7c3aed;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  display: inline-block;
}
@keyframes spin { to { transform: rotate(360deg); } }
</style>
</head>
<body>
<div class="container">
  <div class="card">
    <div class="logo">
      <div class="logo-text">OVAV</div>
    </div>
    <h1>Iniciar Sesión</h1>
    <p class="subtitle">Accede a tu vault desde cualquier dispositivo</p>
    <div class="error" id="error"></div>
    <div class="success" id="success">
      ✓ Sesión verificada — podés cerrar esta pestaña
    </div>
    <form id="loginForm" onsubmit="handleLogin(event)">
      <div class="field">
        <label for="email">Email</label>
        <input type="email" id="email" placeholder="tu@email.com" required autocomplete="email">
      </div>
      <div class="field">
        <label for="password">Contraseña</label>
        <input type="password" id="password" placeholder="••••••••" required autocomplete="current-password">
      </div>
      <button type="submit" class="btn" id="submitBtn">Iniciar Sesión</button>
      <div class="spinner" id="spinner"></div>
      <div class="divider">o</div>
      <a href="/api/v1/auth/oauth-url?provider=google&amp;redirect=1&amp;challenge=` + challengeEsc + `" class="oauth-btn" id="googleBtn">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
          <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" fill="#4285F4"/>
          <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
          <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
          <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
        </svg>
        Continuar con Google
      </a>
    </form>
  </div>
</div>
<script>
const challenge = "` + challengeEsc + `';

function el(id) { return document.getElementById(id); }

function showError(msg) {
  const e = el('error');
  e.textContent = msg;
  e.style.display = 'block';
}
function hideError() { el('error').style.display = 'none'; }
function setLoading(on) {
  el('submitBtn').disabled = on;
  el('spinner').style.display = on ? 'block' : 'none';
}

async function handleLogin(e) {
  e.preventDefault();
  hideError();
  const email = el('email').value.trim();
  const password = el('password').value;
  if (!email || !password) { showError('Completá email y contraseña'); return; }

  setLoading(true);
  try {
    const resp = await fetch('/api/v1/auth/user-login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password, challenge }),
    });
    const data = await resp.json().catch(() => ({}));
    if (!resp.ok) {
      showError(data.error || 'Login falló — verificá tus credenciales');
    } else {
      el('loginForm').style.display = 'none';
      el('success').style.display = 'block';
    }
  } catch(err) {
    showError('Error de conexión: ' + err.message);
  } finally {
    setLoading(false);
  }
}
</script>
</body>
</html>`))
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
