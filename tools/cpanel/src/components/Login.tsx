import { useState, useEffect } from 'react';
import { API_BASE } from '../config';

interface OAuthProvider {
  provider: string;
  client_id: string;
  redirect_uri: string;
}

interface AuthConfig {
  methods: string[];
  oauth: OAuthProvider[];
  has_oauth: boolean;
}

interface LoginProps {
  onLogin: (token: string, role: string, email: string) => void;
}

export default function Login({ onLogin }: LoginProps) {
  const [token, setToken] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [regName, setRegName] = useState('');
  const [regEmail, setRegEmail] = useState('');
  const [regPassword, setRegPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [oauthConfig, setOauthConfig] = useState<AuthConfig | null>(null);
  const [activeTab, setActiveTab] = useState<'oauth' | 'password' | 'register' | 'token'>('password');

  useEffect(() => {
    // Fetch auth config to see which OAuth providers are available
    fetch(`${API_BASE}/api/v1/auth/config`)
      .then(r => r.json())
      .then((cfg: AuthConfig) => {
        setOauthConfig(cfg);
        if (!cfg.has_oauth) setActiveTab('token');
      })
      .catch(() => setActiveTab('token'));
  }, []);

  const handleTokenSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (token.trim().length < 4) {
      setError('Token must be at least 32 characters');
      return;
    }
    setLoading(true);
    try {
      const r = await fetch(`${API_BASE}/api/v1/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: token.trim() }),
      });
      if (!r.ok) { setError('Invalid token'); return; }
      const data = await r.json();
      localStorage.setItem('ovav_cpanel_token', data.token);
      localStorage.setItem('ovav_cpanel_role', data.role);
      localStorage.setItem('ovav_cpanel_email', data.email || 'admin@localhost');
      onLogin(data.token, data.role, data.email || 'admin@localhost');
    } catch {
      setError('Connection failed');
    } finally {
      setLoading(false);
    }
  };

  const handlePasswordSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email.trim() || !password) {
      setError('Email and password are required');
      return;
    }
    setLoading(true);
    try {
      const r = await fetch(`${API_BASE}/api/v1/auth/user-login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: email.trim(), password }),
      });
      if (!r.ok) {
        const d = await r.json().catch(() => ({}));
        setError(d.error || 'Invalid email or password');
        return;
      }
      const data = await r.json();
      localStorage.setItem('ovav_cpanel_token', data.token);
      localStorage.setItem('ovav_cpanel_role', 'user');
      localStorage.setItem('ovav_cpanel_email', data.email);
      localStorage.setItem('ovav_cpanel_user_id', data.user_id);
      localStorage.setItem('ovav_cpanel_tier', data.tier);
      onLogin(data.token, 'user', data.email);
    } catch {
      setError('Connection failed');
    } finally {
      setLoading(false);
    }
  };

  const handleRegisterSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!regEmail.trim() || !regPassword || !regName.trim()) {
      setError('Name, email, and password are required');
      return;
    }
    if (regPassword.length < 8) {
      setError('Password must be at least 8 characters');
      return;
    }
    setLoading(true);
    try {
      const r = await fetch(`${API_BASE}/api/v1/auth/register`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: regEmail.trim(), password: regPassword, name: regName.trim() }),
      });
      if (!r.ok) {
        const d = await r.json().catch(() => ({}));
        setError(d.error || 'Registration failed');
        return;
      }
      const data = await r.json();
      localStorage.setItem('ovav_cpanel_token', data.token);
      localStorage.setItem('ovav_cpanel_role', 'user');
      localStorage.setItem('ovav_cpanel_email', data.email);
      localStorage.setItem('ovav_cpanel_user_id', data.user_id);
      localStorage.setItem('ovav_cpanel_tier', data.tier);
      onLogin(data.token, 'user', data.email);
    } catch {
      setError('Connection failed');
    } finally {
      setLoading(false);
    }
  };

  const handleDevSkip = () => {
    localStorage.setItem('ovav_cpanel_token', 'dev');
    localStorage.setItem('ovav_cpanel_role', 'viewer');
    localStorage.setItem('ovav_cpanel_email', 'dev@localhost');
    onLogin('dev', 'viewer', 'dev@localhost');
  };

  const handleOAuth = (provider: string) => {
    const prov = oauthConfig?.oauth.find(p => p.provider === provider);
    if (!prov) return;

    // Generate CSRF state token (SPA-side, verified on callback)
    const state = crypto.randomUUID();
    localStorage.setItem('ovav_oauth_provider', provider);
    localStorage.setItem('ovav_oauth_state', state);

    // Build OAuth URL
    let authUrl = '';
    if (provider === 'google') {
      const params = new URLSearchParams({
        client_id: prov.client_id,
        redirect_uri: prov.redirect_uri,
        response_type: 'code',
        scope: 'openid email profile',
        access_type: 'offline',
        prompt: 'consent',
        state,
      });
      authUrl = `https://accounts.google.com/o/oauth2/v2/auth?${params}`;
    } else if (provider === 'github') {
      const params = new URLSearchParams({
        client_id: prov.client_id,
        redirect_uri: prov.redirect_uri,
        scope: 'read:user user:email',
        state,
      });
      authUrl = `https://github.com/login/oauth/authorize?${params}`;
    }

    if (authUrl) {
      window.location.href = authUrl;
    }
  };

  // Handle OAuth callback (when redirected back)
  useEffect(() => {
    const urlParams = new URLSearchParams(window.location.search);
    const code = urlParams.get('code');
    const state = urlParams.get('state');
    const provider = localStorage.getItem('ovav_oauth_provider');
    const savedState = localStorage.getItem('ovav_oauth_state');

    // CSRF: verify state matches what we sent
    if (code && provider && state && state === savedState) {
      localStorage.removeItem('ovav_oauth_provider');
      localStorage.removeItem('ovav_oauth_state');
      setLoading(true);
      // Clean URL
      window.history.replaceState({}, '', window.location.pathname);

      fetch(`${API_BASE}/api/v1/auth/oauth/${provider}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ code }),
      })
        .then(r => r.json())
        .then(data => {
          if (data.token) {
            localStorage.setItem('ovav_cpanel_token', data.token);
            localStorage.setItem('ovav_cpanel_role', data.role);
            localStorage.setItem('ovav_cpanel_email', data.email);
            onLogin(data.token, data.role, data.email);
          } else {
            setError(data.error || 'OAuth failed');
          }
        })
        .catch(() => setError('OAuth connection failed'))
        .finally(() => setLoading(false));
    }
  }, []);

  const hasOAuth = oauthConfig?.has_oauth ?? false;

  return (
    <div className="login-container">
      <div className="login-card">
        <div className="login-logo">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 128 128" width="56" height="56" style={{ display: 'block', margin: '0 auto 16px' }}>
            <defs>
              <linearGradient id="lg" x1="0%" y1="0%" x2="100%" y2="100%">
                <stop offset="0%" stopColor="#7c3aed"/>
                <stop offset="100%" stopColor="#8b5cf6"/>
              </linearGradient>
            </defs>
            <rect width="128" height="128" rx="24" fill="url(#lg)"/>
            <text x="64" y="82" fontFamily="system-ui,sans-serif" fontSize="56" fontWeight="700" fill="white" textAnchor="middle" letterSpacing="-2">OV</text>
            <circle cx="96" cy="40" r="8" fill="#f59e0b"/>
          </svg>
          <h1 style={{ fontSize: 24, fontWeight: 800, background: 'linear-gradient(135deg, #7c3aed, #8b5cf6)', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent', marginBottom: 4 }}>
            OVAV Systems
          </h1>
          <div className="login-ver">Control Plane · d678beea.ovav.dev</div>
        </div>

        {loading ? (
          <div style={{ padding: '20px 0', textAlign: 'center' }}>
            <div className="spinner" /> <span style={{ color: 'var(--text-secondary)', fontSize: 14 }}>Authenticating...</span>
          </div>
        ) : (
          <>
            {/* Auth Tabs */}
            <div className="auth-tabs">
              <button
                className={`auth-tab ${activeTab === 'password' ? 'active' : ''}`}
                onClick={() => setActiveTab('password')}
              >Account</button>
              <button
                className={`auth-tab ${activeTab === 'register' ? 'active' : ''}`}
                onClick={() => setActiveTab('register')}
              >Register</button>
              {hasOAuth && (
                <button
                  className={`auth-tab ${activeTab === 'oauth' ? 'active' : ''}`}
                  onClick={() => setActiveTab('oauth')}
                >OAuth</button>
              )}
              <button
                className={`auth-tab ${activeTab === 'token' ? 'active' : ''}`}
                onClick={() => setActiveTab('token')}
              >Token</button>
            </div>

            {/* Password login */}
            {activeTab === 'password' && (
              <form onSubmit={handlePasswordSubmit} className="login-form">
                <label htmlFor="email">Email</label>
                <input
                  id="email"
                  type="email"
                  value={email}
                  onChange={e => { setEmail(e.target.value); setError(''); }}
                  placeholder="you@example.com"
                  autoFocus
                />
                <label htmlFor="password" style={{ marginTop: 8 }}>Password</label>
                <input
                  id="password"
                  type="password"
                  value={password}
                  onChange={e => { setPassword(e.target.value); setError(''); }}
                  placeholder="Your password"
                />
                {error && <div className="login-error">{error}</div>}
                <button type="submit" className="btn-primary" disabled={loading}>
                  {loading ? 'Signing in...' : '🔐 Sign In'}
                </button>
              </form>
            )}

            {/* Registration */}
            {activeTab === 'register' && (
              <form onSubmit={handleRegisterSubmit} className="login-form">
                <label htmlFor="reg-name">Name</label>
                <input
                  id="reg-name"
                  type="text"
                  value={regName}
                  onChange={e => { setRegName(e.target.value); setError(''); }}
                  placeholder="Your name"
                  autoFocus
                />
                <label htmlFor="reg-email" style={{ marginTop: 8 }}>Email</label>
                <input
                  id="reg-email"
                  type="email"
                  value={regEmail}
                  onChange={e => { setRegEmail(e.target.value); setError(''); }}
                  placeholder="you@example.com"
                />
                <label htmlFor="reg-password" style={{ marginTop: 8 }}>Password</label>
                <input
                  id="reg-password"
                  type="password"
                  value={regPassword}
                  onChange={e => { setRegPassword(e.target.value); setError(''); }}
                  placeholder="Min 8 characters"
                />
                {error && <div className="login-error">{error}</div>}
                <button type="submit" className="btn-primary" disabled={loading}>
                  {loading ? 'Creating account...' : '✨ Create Account'}
                </button>
                <div style={{ fontSize: 11, color: 'var(--text-secondary)', marginTop: 8 }}>
                  Free tier: 5 secrets. Premium: 50 secrets + revoke/rotate.
                </div>
              </form>
            )}

            {/* OAuth */}
            {activeTab === 'oauth' && hasOAuth && (
              <div className="oauth-buttons">
                {oauthConfig?.oauth.map(prov => (
                  <button
                    key={prov.provider}
                    className={`oauth-btn oauth-${prov.provider}`}
                    onClick={() => handleOAuth(prov.provider)}
                  >
                    <span className="oauth-icon">
                      {prov.provider === 'google' ? 'G' : '\u2328'}
                    </span>
                    Sign in with {prov.provider === 'google' ? 'Google' : 'GitHub'}
                  </button>
                ))}
              </div>
            )}

            {/* Token auth (admin/CLI) */}
            {activeTab === 'token' && (
              <form onSubmit={handleTokenSubmit} className="login-form">
                <label htmlFor="token">Access Token</label>
                <input
                  id="token"
                  type="password"
                  value={token}
                  onChange={e => { setToken(e.target.value); setError(''); }}
                  placeholder="Admin access token..."
                />
                {error && <div className="login-error">{error}</div>}
                <button type="submit" className="btn-primary" disabled={loading}>
                  {loading ? '...' : 'Access cPanel (Admin)'}
                </button>
              </form>
            )}
          </>
        )}

        <div className="login-footer">
          <button onClick={handleDevSkip} className="btn-link">Skip (dev mode)</button>
          <span className="text-muted">d678beea.ovav.dev</span>
        </div>
      </div>
    </div>
  );
}
