import { useState, useEffect, useCallback, useRef } from 'react';
import { api } from '../services/api';
import ResultPanel from '../components/ResultPanel';
import type { ToastType, ValidatorListResponse } from '../types';

export default function ValidatorsSection({ toast }: { toast: (msg: string, type?: ToastType) => void }) {
  const [list, setList] = useState<ValidatorListResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [status, setStatus] = useState<'idle' | 'running' | 'success' | 'error'>('idle');
  const [output, setOutput] = useState<string | null>(null);
  const [progress, setProgress] = useState('');
  const pollRef = useRef<ReturnType<typeof setInterval>>();

  const load = useCallback(async () => {
    try { setList(await api.getValidators()); } catch { /* */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);
  useEffect(() => { const h = () => load(); window.addEventListener('cpanel-refresh', h); return () => window.removeEventListener('cpanel-refresh', h); }, [load]);
  useEffect(() => () => clearInterval(pollRef.current), []);

  const run = async () => {
    setStatus('running'); setOutput(null); setProgress('Starting validators...');
    try {
      const { task_id } = await api.runValidators();
      pollRef.current = setInterval(async () => {
        try {
          const data = await api.getValidatorStatus(task_id);
          setProgress(data.status === 'running' ? 'Running validators...' : data.status);
          if (data.status === 'complete') {
            setStatus('success');
            setOutput(data.result?.output ?? 'No output');
            clearInterval(pollRef.current);
            load();
          } else if (data.status === 'error') {
            setStatus('error');
            setOutput(`Error: ${data.result?.error || 'unknown'}`);
            clearInterval(pollRef.current);
          }
        } catch { /* keep polling */ }
      }, 1500);
    } catch {
      setStatus('error');
      setOutput('Failed to start validator task.');
    }
  };

  const passRate = list ? Math.round((list.pass / (list.pass + list.fail || 1)) * 100) : 0;

  return (
    <>
      <div className="flex-row mb-12 gap-8">
        <button className="btn btn-primary" onClick={run} disabled={status === 'running'}>
          {status === 'running' ? '⏳ Running...' : '▶ Run All Validators'}
        </button>
        <button className="btn" onClick={load}>↻ Refresh</button>
        {status === 'running' && <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>{progress}</span>}
      </div>

      {loading ? (
        <div className="card skeleton-card"><div className="skeleton skeleton-title" /><div className="skeleton skeleton-text" /></div>
      ) : list ? (
        <div className="card mb-12">
          <h3>System Integrity</h3>
          <div style={{ display: 'flex', gap: 24, alignItems: 'center', flexWrap: 'wrap' }}>
            <div style={{ textAlign: 'center', minWidth: 80 }}>
              <div className="gauge-ring" style={{ borderTopColor: passRate >= 90 ? 'var(--success)' : passRate >= 70 ? 'var(--warning)' : 'var(--danger)' }}>
                <div className="gauge-value">{passRate}%</div>
              </div>
              <div style={{ fontSize: 10, color: 'var(--text-muted)', marginTop: 6 }}>Pass Rate</div>
            </div>
            <div>
              <div style={{ fontSize: 28, fontWeight: 700 }} className={list.overall === 'PASS' ? 'ok' : 'fail'}>{list.overall}</div>
              <div style={{ fontSize: 12, color: 'var(--text-secondary)' }}>
                Score: <strong>{list.score}%</strong> · <span className="ok">{list.pass} pass</span> · <span className={list.fail > 0 ? 'fail' : 'dim'}>{list.fail} fail</span>
              </div>
            </div>
          </div>
          {list.checks.length > 0 && (
            <table className="log-table mt-16">
              <thead><tr><th>Check</th><th>Status</th></tr></thead>
              <tbody>{list.checks.map((c, i) => (
                <tr key={i}><td>{c.name}</td><td><span className={`tag ${c.ok ? 'tag-ok' : 'tag-fail'}`}>{c.ok ? 'PASS' : 'FAIL'}</span></td></tr>
              ))}</tbody>
            </table>
          )}
        </div>
      ) : (
        <div className="empty-state">
          <div className="empty-icon">✓</div>
          <div className="empty-title">No validator data</div>
          <div className="empty-desc">Click "Run All Validators" to check system integrity.</div>
        </div>
      )}

      <ResultPanel
        status={status}
        title="Validator Execution"
        output={output}
        onRunAgain={run}
        onClear={() => { setStatus('idle'); setOutput(null); }}
      />
    </>
  );
}
