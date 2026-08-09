import { useState, useEffect, useCallback, useRef } from 'react';
import { api } from '../services/api';
import type { StatusResponse, ToastType } from '../types';

function pctColor(pct: number) { return pct >= 90 ? 'var(--danger)' : pct >= 70 ? 'var(--warning)' : 'var(--success)'; }

function AnimatedValue({ value, suffix = '' }: { value: string | number; suffix?: string }) {
  const prevRef = useRef(0);
  const [display, setDisplay] = useState(0);
  useEffect(() => {
    const target = Number(value) || 0;
    const start = prevRef.current || 0;
    prevRef.current = target;
    const duration = 600, st = performance.now();
    const animate = (now: number) => {
      const p = Math.min((now - st) / duration, 1);
      setDisplay(Math.round(start + (target - start) * (1 - Math.pow(1 - p, 3))));
      if (p < 1) requestAnimationFrame(animate);
    };
    requestAnimationFrame(animate);
  }, [value]);
  return <>{display.toLocaleString()}{suffix}</>;
}

export default function DashboardSection({ toast }: { toast: (msg: string, type?: ToastType) => void }) {
  const [status, setStatus] = useState<StatusResponse | null>(null);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    try { setStatus(await api.getStatus()); } catch { /* */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { refresh(); const t = setInterval(refresh, 15000); return () => clearInterval(t); }, [refresh]);
  useEffect(() => { const h = () => refresh(); window.addEventListener('cpanel-refresh', h); return () => window.removeEventListener('cpanel-refresh', h); }, [refresh]);

  if (loading) return <div className="grid">{['','','',''].map((_,i) => <div key={i} className="skeleton-card card"><div className="skeleton skeleton-title" /><div className="skeleton skeleton-block" /></div>)}</div>;
  if (!status) return <div className="card"><h3>⚠ Connection Error</h3><button className="btn btn-primary mt-8" onClick={refresh}>Retry</button></div>;

  const git = status.git || { branch: '?', head: '?', commits: '0', dirty: '?', last_commit: '' };
  const system = status.system || { agents: '0', python: '?' };
  const economy = status.economy || { session_cost: 0, session_pct: 0, monthly_cost: 0, monthly_pct: 0, model: '?' };
  const ses = status.session || { uptime: '?', canary_alarms: 0 };
  const sp = economy.session_pct || 0, mp = economy.monthly_pct || 0, alarms = ses.canary_alarms ?? 0;
  const commits = parseInt(git.commits, 10) || 0, agents = parseInt(system.agents, 10) || 0;

  return (
    <>
      <div className="grid">
        <div className="stat-card"><div className="stat-icon">{alarms === 0 ? '✅' : '🚨'}</div><div className={`stat-value ${alarms===0?'ok':'fail'}`}><AnimatedValue value={alarms} /></div><div className="stat-label">Canary Alarms</div><div className="stat-sub">{alarms===0?'All Clear':'Action Required'}</div></div>
        <div className="stat-card"><div className="stat-icon">📈</div><div className="stat-value"><AnimatedValue value={economy.session_cost.toFixed(0)} suffix="¢" /></div><div className="stat-label">Session Cost</div><div className="stat-sub"><span style={{color:pctColor(sp)}}>{sp}% budget</span></div></div>
        <div className="stat-card"><div className="stat-icon">🗂</div><div className="stat-value"><AnimatedValue value={commits} /></div><div className="stat-label">Total Commits</div><div className="stat-sub dim">{git.branch}</div></div>
        <div className="stat-card"><div className="stat-icon">🤖</div><div className="stat-value"><AnimatedValue value={agents} /></div><div className="stat-label">Agents</div><div className="stat-sub dim">Python {system.python}</div></div>
      </div>

      <div className="grid mt-16">
        <div className="card"><h3>💰 Budget</h3>
          <div className="row mb-8"><span className="row-l">Session</span><span className="row-v">${economy.session_cost.toFixed(2)} · {sp}%</span></div>
          <div className="bar" style={{width:'100%',height:10,marginBottom:12}}><div className="bar-fill" style={{width:`${sp}%`,background:pctColor(sp)}} /></div>
          <div className="row mb-8"><span className="row-l">Monthly</span><span className="row-v">${economy.monthly_cost.toFixed(2)} · {mp}%</span></div>
          <div className="bar" style={{width:'100%',height:10}}><div className="bar-fill" style={{width:`${mp}%`,background:pctColor(mp)}} /></div>
        </div>
        <div className="card"><h3>🔀 Repository</h3>
          <div className="row"><span className="row-l">Branch</span><span className="row-v">{git.branch}</span></div>
          <div className="row"><span className="row-l">HEAD</span><span className="row-v" style={{fontSize:11}}>{git.head}</span></div>
          <div className="row"><span className="row-l">State</span><span className={`row-v ${git.dirty==='clean'?'ok':'warn'}`}>{git.dirty}</span></div>
          <div className="row"><span className="row-l">Uptime</span><span className="row-v">{ses.uptime}</span></div>
          <div className="row"><span className="row-l">Model</span><span className="row-v" style={{fontSize:12}}>{economy.model}</span></div>
          <div className="mt-8"><span style={{fontSize:10,color:'var(--text-muted)'}}>{status.timestamp}</span></div>
        </div>
      </div>
      <div className="flex-row mt-16 gap-8"><button className="btn" onClick={refresh}>↻ Refresh <span className="kbd">R</span></button></div>
    </>
  );
}
