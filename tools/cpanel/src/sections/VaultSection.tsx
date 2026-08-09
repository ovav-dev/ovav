/* ═══════════ OVAV cPanel v5.3 — Vault Section ═══════════ */
import { useState, useEffect, useCallback } from 'react';
import { API_BASE } from '../config';
import ResultPanel from '../components/ResultPanel';

interface VaultUser {
  user_id: string;
  email: string;
  tier: string;
  slots: number;
  token: string;
}

interface VaultSecret {
  id: string;
  name: string;
  type: string;
  metadata?: Record<string, string>;
  source: string;
  source_path?: string;
  created_at: string;
  updated_at: string;
}

interface VaultHealth {
  status: 'ok' | 'warning' | 'error';
  online: boolean;
  issues: string[];
}

interface SecretForm {
  name: string;
  value: string;
  type: string;
  source: string;
}

const SECRET_TYPES = [
  { value: 'api_token', label: 'API Token' },
  { value: 'oauth_creds', label: 'OAuth Credentials' },
  { value: 'db_credential', label: 'Database Credential' },
  { value: 'cloud_key', label: 'Cloud Key' },
  { value: 'encryption_key', label: 'Encryption Key' },
  { value: 'user_secret', label: 'User Secret' },
  { value: 'tunnel_token', label: 'Tunnel Token' },
];

function getToken(): string | null {
  return localStorage.getItem('ovav_cpanel_token');
}

function getVaultUser(): VaultUser | null {
  const token = getToken();
  const email = localStorage.getItem('ovav_cpanel_email');
  const userId = localStorage.getItem('ovav_cpanel_user_id');
  const tier = localStorage.getItem('ovav_cpanel_tier');
  if (!token || !email || !userId) return null;
  return { user_id: userId, email, tier: tier || 'free', slots: 0, token };
}

export default function VaultSection() {
  const [vaultUser, setVaultUser] = useState<VaultUser | null>(getVaultUser);
  const [loginEmail, setLoginEmail] = useState('');
  const [loginPassword, setLoginPassword] = useState('');
  const [secrets, setSecrets] = useState<VaultSecret[]>([]);
  const [health, setHealth] = useState<VaultHealth | null>(null);
  const [loading, setLoading] = useState(false);
  const [authLoading, setAuthLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedSecret, setSelectedSecret] = useState<VaultSecret | null>(null);
  const [showAddForm, setShowAddForm] = useState(false);
  const [addForm, setAddForm] = useState<SecretForm>({ name: '', value: '', type: 'api_token', source: 'manual' });
  const [addError, setAddError] = useState<string | null>(null);
  const [addLoading, setAddLoading] = useState(false);
  const [deleteLoading, setDeleteLoading] = useState<string | null>(null);

  const token = getToken();

  // Load health
  const loadHealth = useCallback(async () => {
    try {
      const r = await fetch(`${API_BASE}/api/v1/vault/health`);
      if (r.ok) setHealth(await r.json());
    } catch { /* offline */ }
  }, []);

  // Load secrets
  const loadSecrets = useCallback(async (t: string) => {
    setLoading(true);
    try {
      const r = await fetch(`${API_BASE}/api/v1/vault/secrets`, {
        headers: { Authorization: `Bearer ${t}` },
      });
      if (r.ok) {
        const d = await r.json();
        setSecrets(d.secrets || []);
      } else if (r.status === 401) {
        // Token invalid — clear vault session
        localStorage.removeItem('ovav_cpanel_user_id');
        localStorage.removeItem('ovav_cpanel_tier');
        setVaultUser(null);
        setSecrets([]);
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadHealth();
    if (token) loadSecrets(token);
  }, [loadHealth, loadSecrets, token]);

  // Vault login
  const handleVaultLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setAuthLoading(true);
    setError(null);
    try {
      const r = await fetch(`${API_BASE}/api/v1/auth/user-login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: loginEmail.trim(), password: loginPassword }),
      });
      if (!r.ok) {
        const d = await r.json().catch(() => ({}));
        setError(d.error || 'Login failed');
        return;
      }
      const data = await r.json();
      const vu: VaultUser = {
        user_id: data.user_id,
        email: data.email,
        tier: data.tier,
        slots: data.slots,
        token: data.token,
      };
      // Update cPanel tokens
      localStorage.setItem('ovav_cpanel_user_id', data.user_id);
      localStorage.setItem('ovav_cpanel_tier', data.tier);
      setVaultUser(vu);
      setLoginEmail('');
      setLoginPassword('');
      loadSecrets(data.token);
    } catch {
      setError('Connection failed');
    } finally {
      setAuthLoading(false);
    }
  };

  // Add secret
  const handleAddSecret = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!addForm.name || !addForm.value) {
      setAddError('Name and value are required');
      return;
    }
    setAddLoading(true);
    setAddError(null);
    try {
      const r = await fetch(`${API_BASE}/api/v1/vault/secrets`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          name: addForm.name.toUpperCase().replace(/[^A-Z0-9_]/g, '_'),
          value: addForm.value,
          type: addForm.type,
          source: addForm.source,
        }),
      });
      if (!r.ok) {
        const d = await r.json().catch(() => ({}));
        setAddError(d.error || 'Failed to add secret');
        return;
      }
      setAddForm({ name: '', value: '', type: 'api_token', source: 'manual' });
      setShowAddForm(false);
      loadSecrets(token!);
    } catch {
      setAddError('Connection failed');
    } finally {
      setAddLoading(false);
    }
  };

  // Delete secret
  const handleDeleteSecret = async (name: string) => {
    if (!confirm(`Delete secret "${name}"? This cannot be undone.`)) return;
    setDeleteLoading(name);
    try {
      const r = await fetch(`${API_BASE}/api/v1/vault/secrets/${encodeURIComponent(name)}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      });
      if (r.ok) {
        setSecrets(prev => prev.filter(s => s.name !== name));
        if (selectedSecret?.name === name) setSelectedSecret(null);
      }
    } finally {
      setDeleteLoading(null);
    }
  };

  // View secret detail
  const handleViewSecret = async (sec: VaultSecret) => {
    setSelectedSecret(sec);
  };

  const refresh = () => {
    loadHealth();
    if (token) loadSecrets(token);
  };

  // ── Auth gate ─────────────────────────────────────────────────────────────
  if (!vaultUser) {
    return (
      <div className="vault-auth-gate">
        <div className="card">
          <h3>🔐 OVAV Vault — Per-User Secret Storage</h3>
          <p style={{ fontSize: 12, color: 'var(--text-secondary)', marginBottom: 20 }}>
            Secrets are stored <strong>encrypted on the server</strong>. Sign in with your
            OVAV account to access your vault.
          </p>
          <form onSubmit={handleVaultLogin}>
            <div style={{ marginBottom: 12 }}>
              <label style={{ fontSize: 12, display: 'block', marginBottom: 4 }}>Email</label>
              <input
                type="email"
                value={loginEmail}
                onChange={e => { setLoginEmail(e.target.value); setError(null); }}
                placeholder="you@example.com"
                style={{
                  width: '100%', padding: '8px 12px',
                  background: 'var(--bg-secondary)', border: '1px solid var(--border)',
                  borderRadius: 6, color: 'var(--text-primary)', fontSize: 13,
                  boxSizing: 'border-box',
                }}
                autoFocus
              />
            </div>
            <div style={{ marginBottom: 12 }}>
              <label style={{ fontSize: 12, display: 'block', marginBottom: 4 }}>Password</label>
              <input
                type="password"
                value={loginPassword}
                onChange={e => { setLoginPassword(e.target.value); setError(null); }}
                placeholder="Your password"
                style={{
                  width: '100%', padding: '8px 12px',
                  background: 'var(--bg-secondary)', border: '1px solid var(--border)',
                  borderRadius: 6, color: 'var(--text-primary)', fontSize: 13,
                  boxSizing: 'border-box',
                }}
              />
            </div>
            {error && <div style={{ color: 'var(--color-err)', fontSize: 12, marginBottom: 12 }}>❌ {error}</div>}
            <button
              type="submit"
              className="btn btn-primary"
              disabled={authLoading || !loginEmail || !loginPassword}
            >
              {authLoading ? '⏳ Signing in...' : '🔐 Sign In to Vault'}
            </button>
          </form>
          <div style={{ marginTop: 16, fontSize: 11, color: 'var(--text-secondary)' }}>
            No account? Use the <strong>Register</strong> tab above to create one.
            Free tier includes 5 secrets.
          </div>
        </div>
      </div>
    );
  }

  // ── Vault dashboard ─────────────────────────────────────────────────────────
  const count = secrets.length;
  const limit = vaultUser.slots || (vaultUser.tier === 'premium' ? 50 : vaultUser.tier === 'enterprise' ? 999999 : 5);

  return (
    <div className="vault-dashboard">
      {/* Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <div>
          <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
            {vaultUser.email} · <strong>{vaultUser.tier}</strong> tier
          </span>
        </div>
        <div style={{ display: 'flex', gap: 8 }}>
          <button className="btn btn-secondary" onClick={refresh}>🔄 Refresh</button>
          <button className="btn btn-primary" onClick={() => setShowAddForm(true)} disabled={count >= limit}>
            + Add Secret
          </button>
        </div>
      </div>

      {/* Health */}
      {health && (
        <div className="card" style={{ marginBottom: 16 }}>
          <div style={{ display: 'flex', gap: 24, flexWrap: 'wrap' }}>
            <div>
              <span style={{ fontSize: 11, color: 'var(--text-secondary)' }}>Vault Server</span>
              <div style={{ fontSize: 20, fontWeight: 600, color: health.online ? 'var(--color-ok)' : 'var(--color-err)' }}>
                {health.online ? '🌐 Online' : '📴 Offline'}
              </div>
            </div>
            <div>
              <span style={{ fontSize: 11, color: 'var(--text-secondary)' }}>Secrets</span>
              <div style={{ fontSize: 20, fontWeight: 600 }}>{count} / {limit === 999999 ? '∞' : limit}</div>
            </div>
            <div>
              <span style={{ fontSize: 11, color: 'var(--text-secondary)' }}>Tier</span>
              <div style={{ fontSize: 20, fontWeight: 600, textTransform: 'capitalize' }}>{vaultUser.tier}</div>
            </div>
          </div>
          {health.issues && health.issues.length > 0 && (
            <div style={{ marginTop: 12, padding: '8px 12px', background: 'var(--bg-secondary)', borderRadius: 6, fontSize: 12 }}>
              ⚠️ {health.issues.join(' · ')}
            </div>
          )}
        </div>
      )}

      {/* Add form */}
      {showAddForm && (
        <div className="card" style={{ marginBottom: 16, border: '1px solid var(--color-ok)' }}>
          <h3 style={{ fontSize: 13, marginBottom: 12 }}>➕ Add New Secret</h3>
          <form onSubmit={handleAddSecret}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8, marginBottom: 8 }}>
              <div>
                <label style={{ fontSize: 11, color: 'var(--text-secondary)', display: 'block', marginBottom: 2 }}>Name</label>
                <input
                  value={addForm.name}
                  onChange={e => setAddForm(f => ({ ...f, name: e.target.value.toUpperCase() }))}
                  placeholder="MY_API_KEY"
                  style={{ width: '100%', padding: '6px 8px', background: 'var(--bg-secondary)', border: '1px solid var(--border)', borderRadius: 4, color: 'var(--text-primary)', fontSize: 12, boxSizing: 'border-box' }}
                  autoFocus
                />
              </div>
              <div>
                <label style={{ fontSize: 11, color: 'var(--text-secondary)', display: 'block', marginBottom: 2 }}>Type</label>
                <select
                  value={addForm.type}
                  onChange={e => setAddForm(f => ({ ...f, type: e.target.value }))}
                  style={{ width: '100%', padding: '6px 8px', background: 'var(--bg-secondary)', border: '1px solid var(--border)', borderRadius: 4, color: 'var(--text-primary)', fontSize: 12, boxSizing: 'border-box' }}
                >
                  {SECRET_TYPES.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
                </select>
              </div>
            </div>
            <div style={{ marginBottom: 8 }}>
              <label style={{ fontSize: 11, color: 'var(--text-secondary)', display: 'block', marginBottom: 2 }}>Value</label>
              <input
                type="password"
                value={addForm.value}
                onChange={e => setAddForm(f => ({ ...f, value: e.target.value }))}
                placeholder="The secret value..."
                style={{ width: '100%', padding: '6px 8px', background: 'var(--bg-secondary)', border: '1px solid var(--border)', borderRadius: 4, color: 'var(--text-primary)', fontSize: 12, boxSizing: 'border-box' }}
              />
            </div>
            {addError && <div style={{ color: 'var(--color-err)', fontSize: 12, marginBottom: 8 }}>❌ {addError}</div>}
            <div style={{ display: 'flex', gap: 8 }}>
              <button type="submit" className="btn btn-primary" disabled={addLoading}>➕ Add</button>
              <button type="button" className="btn btn-secondary" onClick={() => setShowAddForm(false)}>Cancel</button>
            </div>
          </form>
        </div>
      )}

      {/* Secrets list */}
      <div className="card">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
          <h3>🔑 Secrets ({count})</h3>
          <span style={{ fontSize: 11, color: 'var(--text-secondary)' }}>
            {count >= limit ? '⚠️ Slot limit reached' : `${limit - count} slots remaining`}
          </span>
        </div>

        {loading ? (
          <div style={{ textAlign: 'center', padding: 20, color: 'var(--text-secondary)' }}>Loading...</div>
        ) : secrets.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 32, color: 'var(--text-secondary)', fontSize: 13 }}>
            No secrets yet.{' '}
            <button className="btn-link" onClick={() => setShowAddForm(true)}>Add your first secret</button>
          </div>
        ) : (
          <div style={{ display: 'grid', gap: 4 }}>
            {secrets.map((sec) => (
              <div
                key={sec.id}
                style={{
                  display: 'flex', alignItems: 'center', gap: 8,
                  padding: '8px 12px',
                  background: selectedSecret?.id === sec.id ? 'var(--bg-secondary)' : 'transparent',
                  borderRadius: 6, cursor: 'pointer',
                  border: '1px solid var(--border)',
                }}
                onClick={() => handleViewSecret(sec)}
              >
                <span style={{ fontSize: 14 }}>🔑</span>
                <span style={{ flex: 1, fontFamily: 'monospace', fontSize: 13 }}>{sec.name}</span>
                <span style={{ fontSize: 11, color: 'var(--text-secondary)', background: 'var(--bg-secondary)', padding: '2px 6px', borderRadius: 4 }}>
                  {sec.type}
                </span>
                <span style={{ fontSize: 11, color: 'var(--text-secondary)' }}>{sec.source}</span>
                <span style={{ fontSize: 11, color: 'var(--text-secondary)' }}>{formatAge(sec.updated_at)}</span>
                <button
                  className="btn-link"
                  style={{ color: 'var(--color-err)', fontSize: 11 }}
                  onClick={(e) => { e.stopPropagation(); handleDeleteSecret(sec.name); }}
                  disabled={deleteLoading === sec.name}
                >
                  {deleteLoading === sec.name ? '...' : '✕'}
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Help */}
      <div className="card" style={{ marginTop: 16 }}>
        <h3 style={{ fontSize: 13, marginBottom: 8 }}>💡 CLI Access</h3>
        <div style={{ fontSize: 12, color: 'var(--text-secondary)', display: 'grid', gap: 6 }}>
          <div><code style={{ color: 'var(--text-primary)' }}>ovav vault login</code> — Authenticate via browser</div>
          <div><code style={{ color: 'var(--text-primary)' }}>ovav vault secrets list</code> — List secrets</div>
          <div><code style={{ color: 'var(--text-primary)' }}>ovav vault secrets get NAME</code> — Get secret value</div>
          <div><code style={{ color: 'var(--text-primary)' }}>ovav vault secrets add NAME --value "..."</code> — Add secret</div>
        </div>
      </div>
    </div>
  );
}

function formatAge(iso: string): string {
  try {
    const diff = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
    if (diff < 60) return `${diff}s ago`;
    if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
    if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
    return `${Math.floor(diff / 86400)}d ago`;
  } catch {
    return iso;
  }
}
