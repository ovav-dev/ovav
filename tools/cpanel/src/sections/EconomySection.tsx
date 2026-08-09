import { useState, useEffect, useCallback } from 'react';
import { API_BASE } from '../config';
import type { ToastType } from '../types';

export default function EconomySection({ toast }: { toast: (msg: string, type?: ToastType) => void }) {
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try { const r = await fetch(`${API_BASE}/api/v1/economy`); setData(await r.json()); }
    catch { /* */ } finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);
  useEffect(() => { const h = () => load(); window.addEventListener('cpanel-refresh', h); return () => window.removeEventListener('cpanel-refresh', h); }, [load]);

  if (loading) return <div className="card skeleton-card"><div className="skeleton skeleton-title" /><div className="skeleton skeleton-block" /></div>;

  const session = data?.session || {};
  const monthly = data?.monthly || {};
  const model = data?.current_model || '?';
  const sp = session.percent || 0;
  const mp = monthly.percent || 0;

  const pctColor = (p: number) => p >= 90 ? 'var(--danger)' : p >= 70 ? 'var(--warning)' : 'var(--success)';

  return (
    <>
      <div className="grid">
        <div className="stat-card">
          <div className="stat-icon">💵</div>
          <div className="stat-value">${(session.cost_usd || 0).toFixed(2)}</div>
          <div className="stat-label">Session Cost</div>
          <div className="stat-sub"><span style={{color:pctColor(sp)}}>{sp}% of budget</span></div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">📊</div>
          <div className="stat-value">${(monthly.cost_usd || 0).toFixed(2)}</div>
          <div className="stat-label">Monthly Cost</div>
          <div className="stat-sub"><span style={{color:pctColor(mp)}}>{mp}% of budget</span></div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">🧠</div>
          <div className="stat-value" style={{fontSize:20}}>{model}</div>
          <div className="stat-label">Current Model</div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">📅</div>
          <div className="stat-value" style={{fontSize:20}}>{session.date || '?'}</div>
          <div className="stat-label">Last Updated</div>
        </div>
      </div>
      <div className="card mt-16">
        <h3>📋 Raw Economy Data</h3>
        <pre style={{maxHeight:400}}>{JSON.stringify(data, null, 2)}</pre>
        <div className="flex-row mt-8 gap-8"><button className="btn btn-sm" onClick={load}>↻ Refresh</button></div>
      </div>
    </>
  );
}
