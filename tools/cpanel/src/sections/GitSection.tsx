import { useState, useEffect, useCallback } from 'react';
import { API_BASE } from '../config';
import ResultPanel from '../components/ResultPanel';
import type { ToastType } from '../types';

interface SectionProps { toast: (msg: string, type?: ToastType) => void; view: string; }

export default function GitSection({ toast, view }: SectionProps) {
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [actionStatus, setActionStatus] = useState<'idle' | 'running' | 'success' | 'error'>('idle');
  const [actionOutput, setActionOutput] = useState<string | null>(null);

  const endpoints: Record<string, string> = {
    branches: `${API_BASE}/api/v1/git/branches`,
    commits: `${API_BASE}/api/v1/git/log`,
    worktrees: `${API_BASE}/api/v1/git/worktrees`,
  };

  const load = useCallback(async () => {
    setLoading(true);
    try { const r = await fetch(endpoints[view] || `${API_BASE}/api/v1/git/branches`); setData(await r.json()); }
    catch { /* */ } finally { setLoading(false); }
  }, [view]);

  useEffect(() => { load(); }, [load]);
  useEffect(() => { const h = () => load(); window.addEventListener('cpanel-refresh', h); return () => window.removeEventListener('cpanel-refresh', h); }, [load]);

  const handleFetch = async () => {
    setActionStatus('running'); setActionOutput(null);
    try {
      const r = await fetch(`${API_BASE}/api/v1/git/fetch`, { method: 'POST' });
      const d = await r.json();
      setActionStatus(d.status === 'ok' ? 'success' : 'error');
      setActionOutput(d.output || d.error || 'Fetch completed.');
      if (d.status === 'ok') { load(); toast('Fetch completed', 'ok'); }
    } catch (e) {
      setActionStatus('error');
      setActionOutput(`Fetch failed: ${(e as Error).message}`);
    }
  };

  if (loading) return <div className="card skeleton-card"><div className="skeleton skeleton-title" /><div className="skeleton skeleton-text" /></div>;

  if (view === 'branches') {
    const branches: string[] = data?.branches || [];
    const current = data?.current || '?';
    return (
      <>
        <div className="card">
          <h3>🔀 Branches · {data?.total || 0} total</h3>
          <div className="flex-row mb-12 gap-8">
            <span className="tag tag-ok">Current: {current}</span>
            <button className="btn btn-sm btn-success" onClick={handleFetch} disabled={actionStatus === 'running'}>
              {actionStatus === 'running' ? '⏳ Fetching...' : '⬇ Fetch Origin'}
            </button>
          </div>
          {branches.length ? branches.map((b: string, i: number) => (
            <div key={i} className="row" style={{ padding: '4px 0' }}>
              <span style={{ fontSize: 12, fontFamily: 'var(--font-mono)' }} className={b === current ? 'ok' : ''}>
                {b === current ? '★ ' : ''}{b}
              </span>
            </div>
          )) : <div className="empty-state"><div className="empty-icon">🔀</div><div className="empty-title">No branches</div></div>}
          <div className="flex-row mt-8 gap-8"><button className="btn btn-sm" onClick={load}>↻ Refresh</button></div>
        </div>
        <ResultPanel status={actionStatus} title="Git Fetch" output={actionOutput}
          onRunAgain={handleFetch}
          onClear={() => { setActionStatus('idle'); setActionOutput(null); }} />
      </>
    );
  }

  if (view === 'commits') {
    const commits: string[] = data?.commits || [];
    return (
      <div className="card">
        <h3>📜 Recent Commits · {data?.total || 0}</h3>
        <div className="flex-row mb-8 gap-8">
          <button className="btn btn-sm btn-success" onClick={handleFetch} disabled={actionStatus === 'running'}>
            {actionStatus === 'running' ? '⏳' : '⬇ Fetch'}
          </button>
        </div>
        {commits.length ? commits.map((c: string, i: number) => (
          <div key={i} style={{ padding: '4px 0', fontSize: 12, fontFamily: 'var(--font-mono)', borderBottom: '1px solid var(--border-default)' }}>{c}</div>
        )) : <div className="empty-state"><div className="empty-icon">📜</div><div className="empty-title">No commits</div></div>}
        <div className="flex-row mt-8 gap-8"><button className="btn btn-sm" onClick={load}>↻ Refresh</button></div>
        <ResultPanel status={actionStatus} title="Git Fetch" output={actionOutput}
          onClear={() => { setActionStatus('idle'); setActionOutput(null); }} />
      </div>
    );
  }

  const worktrees: string[] = data?.worktrees || [];
  return (
    <div className="card">
      <h3>📂 Worktrees · {data?.total || 0}</h3>
      {worktrees.length ? worktrees.map((w: string, i: number) => (
        <div key={i} style={{ padding: '4px 0', fontSize: 11, fontFamily: 'var(--font-mono)', borderBottom: '1px solid var(--border-default)' }}>{w}</div>
      )) : <div className="empty-state"><div className="empty-icon">📂</div><div className="empty-title">No worktrees</div></div>}
      <div className="flex-row mt-8 gap-8"><button className="btn btn-sm" onClick={load}>↻ Refresh</button></div>
    </div>
  );
}
