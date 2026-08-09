import { useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';
import ResultPanel from '../components/ResultPanel';
import type { ToastType } from '../types';

interface ProfileItem {
  id: string;
  name: string;
  area: string;
  description?: string;
}

export default function ProfilesSection({ toast }: { toast: (msg: string, type?: ToastType) => void }) {
  const [profiles, setProfiles] = useState<ProfileItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [actionStatus, setActionStatus] = useState<'idle' | 'running' | 'success' | 'error'>('idle');
  const [actionOutput, setActionOutput] = useState<string | null>(null);
  const [actionTitle, setActionTitle] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.getProfiles();
      // Go backend returns: { status: "ok", profiles: [{id, name, area, ...}] }
      if (data.ok && Array.isArray((data as any).profiles)) {
        setProfiles((data as any).profiles);
      } else if (data.ok && data.output) {
        // Fallback: Python-style string output — parse line by line
        const ids = data.output.split('\n').slice(2).filter((l: string) => l.trim())
          .map((l: string) => {
            const parts = l.trim().split(/\s+/);
            return { id: parts[0] || '', name: parts[1] || parts[0] || '', area: parts[0] || '' } as ProfileItem;
          }).filter((p: ProfileItem) => p.id);
        setProfiles(ids);
      } else {
        setProfiles([]);
      }
    } catch {
      setProfiles([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);
  useEffect(() => {
    const h = () => load();
    window.addEventListener('cpanel-refresh', h);
    return () => window.removeEventListener('cpanel-refresh', h);
  }, [load]);

  const handleApply = async (area: string) => {
    setActionStatus('running');
    setActionTitle(`Applying: ${area}`);
    setActionOutput(null);
    try {
      const data = await api.applyProfile(area);
      if (data.ok) {
        setActionStatus('success');
        setActionOutput(data.output || `Profile "${area}" applied successfully.`);
        toast(`${area} applied`, 'ok');
        load();
      } else {
        setActionStatus('error');
        setActionOutput(data.error || data.output || `Failed to apply "${area}".`);
      }
    } catch (e) {
      setActionStatus('error');
      setActionOutput(`Error: ${(e as Error).message}`);
    }
  };

  const handlePreview = async (area: string) => {
    setActionStatus('running');
    setActionTitle(`Preview: ${area} (dry-run)`);
    setActionOutput(null);
    try {
      const data = await api.applyProfile(area, '.', true);
      if (data.ok || data.output) {
        setActionStatus('success');
        setActionOutput(data.output || data.error || 'No changes detected.');
        toast(`${area} preview ready`, 'info');
      } else {
        setActionStatus('error');
        setActionOutput(data.error || 'Preview failed.');
      }
    } catch (e) {
      setActionStatus('error');
      setActionOutput(`Error: ${(e as Error).message}`);
    }
  };

  if (loading) {
    return <div className="card skeleton-card"><div className="skeleton skeleton-title" /><div className="skeleton skeleton-block" /></div>;
  }

  return (
    <>
      <div className="card">
        <h3>📦 Service Profiles {profiles.length > 0 && `(${profiles.length})`}</h3>
        {profiles.length > 0 ? (
          <div className="flex-col gap-4" style={{ marginBottom: 16 }}>
            {profiles.map((p) => (
              <div key={p.id} className="row" style={{ padding: '8px 0', borderBottom: '1px solid var(--border-default)' }}>
                <div>
                  <span className="row-v ok" style={{ fontWeight: 600 }}>{p.name || p.id}</span>
                  <span className="tag tag-ok ml-8">{p.area || p.id}</span>
                  {p.description && <div style={{ fontSize: 11, color: 'var(--text-secondary)', marginTop: 2 }}>{p.description}</div>}
                </div>
                <div className="flex-row gap-4">
                  <button
                    className="btn btn-primary btn-sm"
                    onClick={() => handleApply(p.id)}
                    disabled={actionStatus === 'running'}
                  >
                    {actionStatus === 'running' && actionTitle.includes(p.id) ? '⏳' : 'Apply'}
                  </button>
                  <button
                    className="btn btn-sm"
                    onClick={() => handlePreview(p.id)}
                    disabled={actionStatus === 'running'}
                  >
                    {actionStatus === 'running' && actionTitle.includes(p.id) ? '⏳' : 'Preview'}
                  </button>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="empty-state">
            <div className="empty-icon">📦</div>
            <div className="empty-title">No profiles available</div>
            <div className="empty-desc">Install service area profiles to manage them from here.</div>
          </div>
        )}
        <div className="flex-row mt-8 gap-8">
          <button className="btn btn-sm" onClick={load}>↻ Refresh</button>
        </div>
      </div>

      <ResultPanel
        status={actionStatus}
        title={actionTitle}
        output={actionOutput}
        onRunAgain={() => setActionStatus('idle')}
        onClear={() => { setActionStatus('idle'); setActionOutput(null); }}
      />
    </>
  );
}
