import { useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';
import { API_BASE } from '../config';
import ResultPanel from '../components/ResultPanel';
import type { ToastType } from '../types';

interface SectionProps { toast: (msg: string, type?: ToastType) => void; view: string; }

export default function SecuritySection({ toast, view }: SectionProps) {
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [canaryAlarms, setCanaryAlarms] = useState<number | null>(null);
  const [actionStatus, setActionStatus] = useState<'idle' | 'running' | 'success' | 'error'>('idle');
  const [actionOutput, setActionOutput] = useState<string | null>(null);
  const [actionTitle, setActionTitle] = useState('');

  // Auth analytics view (P2-B native)
  const [analyticsData, setAnalyticsData] = useState<any>(null);
  const [auditLog, setAuditLog] = useState<any>(null);
  const [days, setDays] = useState(7);

  const loadAnalytics = useCallback(async () => {
    try {
      const [an, ev] = await Promise.all([
        api.getAuthAnalytics(days),
        api.getAuthAuditLog({ limit: 100 }),
      ]);
      setAnalyticsData(an);
      setAuditLog(ev);
    } catch { /* */ }
  }, [days]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      if (view === 'audit') {
        const [log, status] = await Promise.all([api.getAuditLog({ limit: 50 }), api.getStatus()]);
        setData(log); setCanaryAlarms(status?.session?.canary_alarms ?? 0);
      } else if (view === 'alarms') {
        const status = await api.getStatus();
        setCanaryAlarms(status?.session?.canary_alarms ?? 0);
      } else if (view === 'integrity') {
        setActionStatus('running'); setActionTitle('Living Integrity Scan');
        try {
          const r = await fetch(`${API_BASE}/api/v1/security/living-integrity`);
          const d = await r.json();
          setData(d);
          setActionStatus(d.overall === 'PASS' ? 'success' : 'error');
          setActionOutput(d.output || JSON.stringify(d.checks, null, 2));
        } catch (e) {
          setActionStatus('error');
          setActionOutput(`Scan failed: ${(e as Error).message}`);
        }
      }
    } catch { /* */ }
    finally { setLoading(false); }
  }, [view]);

  useEffect(() => { load(); }, [load]);
  useEffect(() => { const h = () => load(); window.addEventListener('cpanel-refresh', h); return () => window.removeEventListener('cpanel-refresh', h); }, [load]);
  useEffect(() => { if (view === 'auth-analytics') loadAnalytics(); }, [view, loadAnalytics]);

  const handleClearAlarms = async () => {
    setActionStatus('running'); setActionTitle('Clear Canary Alarms');
    try {
      await api.clearAlarms();
      setCanaryAlarms(0);
      setActionStatus('success');
      setActionOutput('All canary alarms cleared. Counter reset to 0.');
      toast('Alarms cleared', 'ok');
    } catch (e) {
      setActionStatus('error');
      setActionOutput(`Failed: ${(e as Error).message}`);
    }
  };

  // ── Auth Analytics View ─────────────────────────────────────────────
  if (view === 'auth-analytics') {
    const stats = analyticsData?.daily_stats || [];
    const topRisks = analyticsData?.top_risk || [];
    const bruteForce = analyticsData?.brute_force_ips || [];
    return (
      <>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginBottom: 16 }}>
          <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>Period:</span>
          {[7, 14, 30].map(d => (
            <button key={d} className={`btn btn-sm ${days === d ? 'btn-primary' : ''}`}
              onClick={() => { setDays(d); }}>{d}d</button>
          ))}
          <button className="btn btn-sm" onClick={loadAnalytics}>↻ Refresh</button>
        </div>

        <div className="grid">
          <div className="stat-card">
            <div className="stat-icon">👥</div>
            <div className="stat-value" style={{ color: 'var(--accent)' }}>{analyticsData?.unique_users ?? 0}</div>
            <div className="stat-label">Unique Users</div>
          </div>
          <div className="stat-card">
            <div className="stat-icon">🌍</div>
            <div className="stat-value" style={{ color: 'var(--info)' }}>{analyticsData?.countries ?? 0}</div>
            <div className="stat-label">Countries</div>
          </div>
          <div className="stat-card">
            <div className="stat-icon">📊</div>
            <div className="stat-value">{analyticsData?.total_events ?? 0}</div>
            <div className="stat-label">Auth Events</div>
          </div>
          <div className="stat-card">
            <div className="stat-icon">🚨</div>
            <div className="stat-value" style={{ color: bruteForce.length > 0 ? 'var(--danger)' : 'var(--success)' }}>
              {bruteForce.length}
            </div>
            <div className="stat-label">Suspicious IPs</div>
          </div>
        </div>

        {stats.length > 0 && (
          <div className="card mt-16">
            <h3>📈 Login Activity</h3>
            <table className="log-table">
              <thead><tr><th>Date</th><th>Total</th><th>Allowed</th><th>Denied</th></tr></thead>
              <tbody>
                {stats.map((s: any) => (
                  <tr key={s.date}>
                    <td>{s.date}</td>
                    <td><span className="badge badge-muted">{s.count}</span></td>
                    <td><span className="badge badge-ok">{s.allowed}</span></td>
                    <td><span className={s.failed > 0 ? 'badge badge-fail' : 'badge badge-muted'}>{s.failed}</span></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {topRisks.length > 0 && (
          <div className="card mt-16">
            <h3>⚠️ Top Risk Events</h3>
            <table className="log-table">
              <thead><tr><th>Time</th><th>Email</th><th>IP</th><th>Country</th><th>Risk</th><th>Status</th></tr></thead>
              <tbody>
                {topRisks.slice(0, 20).map((r: any, i: number) => (
                  <tr key={i}>
                    <td style={{ fontSize: 10 }}>{r.timestamp?.substring(11, 19)}</td>
                    <td style={{ fontSize: 11 }}>{r.email}</td>
                    <td style={{ fontSize: 11 }}>{r.ip}</td>
                    <td><span className="badge badge-muted">{r.country || '?'}</span></td>
                    <td><span className={`badge ${r.risk_score > 70 ? 'badge-fail' : r.risk_score > 30 ? 'badge-warn' : 'badge-ok'}`}>{r.risk_score}</span></td>
                    <td><span className={`badge ${r.status === 'ok' ? 'badge-ok' : 'badge-fail'}`}>{r.status}</span></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {bruteForce.length > 0 && (
          <div className="card mt-16">
            <h3>🔴 Brute Force Detected</h3>
            <table className="log-table">
              <thead><tr><th>IP</th><th>Failed Attempts</th><th>Emails Targeted</th></tr></thead>
              <tbody>
                {bruteForce.map((b: any, i: number) => (
                  <tr key={i}>
                    <td><span className="badge badge-fail">{b.ip}</span></td>
                    <td><span className="badge badge-fail">{b.count}</span></td>
                    <td style={{ fontSize: 11, color: 'var(--text-secondary)' }}>{b.emails}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {(!stats.length && !topRisks.length && !bruteForce.length) && (
          <div className="card mt-16">
            <div className="empty-state">
              <div className="empty-icon">🔐</div>
              <div className="empty-title">No auth events yet</div>
              <div className="empty-desc">Login events will appear here once users authenticate via CF Access or the portal.</div>
            </div>
          </div>
        )}
      </>
    );
  }

  if (loading && view !== 'integrity') return <div className="card skeleton-card"><div className="skeleton skeleton-title" /><div className="skeleton skeleton-block" /></div>;

  if (view === 'audit') {
    const entries = data?.entries || [];
    return (
      <div className="card">
        <h3>📋 Audit Log · <span className={data?.chain_intact ? 'ok' : 'fail'}>{data?.chain_intact ? 'Chain INTACT' : 'Chain BROKEN'}</span> · {data?.total || 0} entries</h3>
        {entries.length ? (
          <table className="log-table"><thead><tr><th>Time</th><th>Category</th><th>Tier</th><th>Details</th></tr></thead>
            <tbody>{entries.slice(0, 50).map((e: any, i: number) => (
              <tr key={i}><td style={{ whiteSpace: 'nowrap', fontSize: 11 }}>{(e.iso_time || '').substring(11, 19)}</td>
                <td><span className={`tag ${e.category === 'canary_trigger' ? 'tag-fail' : 'tag-ok'}`}>{e.category}</span></td>
                <td style={{ fontSize: 11 }}>{e.tier}</td>
                <td style={{ fontSize: 10, maxWidth: 300, overflow: 'hidden', textOverflow: 'ellipsis' }}>{JSON.stringify(e.details || {}).substring(0, 100)}</td></tr>
            ))}</tbody></table>
        ) : <div className="empty-state"><div className="empty-icon">🔍</div><div className="empty-title">No audit entries</div></div>}
        <div className="flex-row mt-8 gap-8"><button className="btn btn-sm" onClick={load}>↻ Refresh</button></div>
      </div>
    );
  }

  if (view === 'alarms') {
    return (
      <>
        <div className="card">
          <h3>🚨 Canary Alarms</h3>
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <span style={{ fontSize: 48, fontWeight: 700 }} className={canaryAlarms === 0 ? 'ok' : 'fail'}>{canaryAlarms ?? '?'}</span>
            <div>
              <div style={{ fontSize: 14, color: canaryAlarms === 0 ? 'var(--success)' : 'var(--danger)' }}>
                {canaryAlarms === 0 ? 'System Secure' : `${canaryAlarms} Active Alarm${canaryAlarms !== 1 ? 's' : ''}`}
              </div>
            </div>
          </div>
          <div className="flex-row mt-12 gap-8">
            <button className="btn btn-danger btn-sm" onClick={handleClearAlarms} disabled={canaryAlarms === 0 || actionStatus === 'running'}>
              {actionStatus === 'running' ? '⏳ Clearing...' : 'Clear Alarms'}
            </button>
            <button className="btn btn-sm" onClick={load}>↻ Refresh</button>
          </div>
        </div>
        <ResultPanel status={actionStatus} title={actionTitle} output={actionOutput}
          onRunAgain={() => { if (canaryAlarms !== 0) handleClearAlarms(); }}
          onClear={() => { setActionStatus('idle'); setActionOutput(null); }} />
      </>
    );
  }

  // integrity
  const checks = data?.checks || [];
  return (
    <>
      <div className="card">
        <h3>🔍 Living Integrity Scan · <span className={data?.overall === 'PASS' ? 'ok' : 'fail'}>{data?.overall || 'Scanning...'}</span></h3>
        {checks.length > 0 && (
          <table className="log-table mt-8"><thead><tr><th>Check</th><th>Status</th></tr></thead>
            <tbody>{checks.map((c: any, i: number) => <tr key={i}><td>{c.name}</td><td><span className={`tag ${c.ok ? 'tag-ok' : 'tag-fail'}`}>{c.ok ? 'PASS' : 'FAIL'}</span></td></tr>)}</tbody></table>
        )}
        <div className="flex-row mt-8 gap-8"><button className="btn btn-sm" onClick={load}>↻ Re-scan</button></div>
      </div>
      <ResultPanel status={actionStatus} title={actionTitle} output={actionOutput}
        onRunAgain={load}
        onClear={() => { setActionStatus('idle'); setActionOutput(null); }} />
    </>
  );
}
