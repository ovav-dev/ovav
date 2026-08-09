import { useState, useEffect, useCallback } from 'react';
import type { ToastType } from '../types';

interface SectionProps { toast: (msg: string, type?: ToastType) => void; view: string; }

export default function AgentsSection({ toast, view }: SectionProps) {
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  const endpoints: Record<string, string> = {
    list: '/api/v1/agents',
    topology: '/api/v1/agents/topology',
    permissions: '/api/v1/agents/permissions',
  };

  const load = useCallback(async () => {
    setLoading(true);
    try { const r = await fetch(endpoints[view] || '/api/v1/agents'); setData(await r.json()); }
    catch { /* */ } finally { setLoading(false); }
  }, [view]);

  useEffect(() => { load(); }, [load]);

  if (loading) return <div className="card skeleton-card"><div className="skeleton skeleton-title" /><div className="skeleton skeleton-text" /></div>;

  if (view === 'list') {
    const agents = data?.agents || [];
    return (
      <div className="card">
        <h3>🤖 Agents · {data?.total || 0} total</h3>
        <table className="log-table">
          <thead><tr><th>Name</th><th>File</th><th>Mode</th><th>Visible</th></tr></thead>
          <tbody>{agents.map((a: any, i: number) => (
            <tr key={i}><td className="row-v">{a.name}</td><td style={{fontSize:11}}>{a.file}</td>
              <td><span className="tag tag-ok">{a.mode}</span></td>
              <td><span className={`tag ${a.hidden?'tag-warn':'tag-ok'}`}>{a.hidden ? 'hidden' : 'visible'}</span></td></tr>
          ))}</tbody></table>
        <div className="flex-row mt-8 gap-8"><button className="btn btn-sm" onClick={load}>↻ Refresh</button></div>
      </div>
    );
  }

  if (view === 'topology') {
    const areas = data?.areas || [];
    return (
      <div className="card">
        <h3>🗺 Topology</h3>
        {areas.map((a: any, i: number) => (
          <div key={i} className="card mt-8" style={{background:'var(--bg-primary)'}}>
            <div style={{fontSize:11,color:'var(--text-muted)',marginBottom:4}}>{a.file}</div>
            <pre style={{fontSize:11,maxHeight:200}}>{JSON.stringify(a.data, null, 2).substring(0,1000)}</pre>
          </div>
        ))}
        {areas.length === 0 && <div className="empty-state"><div className="empty-icon">🗺</div><div className="empty-title">No topology data</div></div>}
      </div>
    );
  }

  // permissions
  return (
    <div className="card">
      <h3>🔐 Permission Authority</h3>
      {data ? <pre style={{fontSize:11,maxHeight:500}}>{JSON.stringify(data, null, 2).substring(0,3000)}</pre>
       : <div className="empty-state"><div className="empty-icon">🔐</div><div className="empty-title">No permissions data</div></div>}
    </div>
  );
}
