import { useState, useEffect, useCallback } from 'react';
import { API_BASE } from '../config';
import ResultPanel from '../components/ResultPanel';
import type { ToastType } from '../types';

interface SectionProps { toast: (msg: string, type?: ToastType) => void; view: string; }

export default function SystemSection({ toast, view }: SectionProps) {
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [registryList, setRegistryList] = useState<string[]>([]);
  const [selectedRegistry, setSelectedRegistry] = useState('');
  const [actionStatus, setActionStatus] = useState<'idle' | 'running' | 'success' | 'error'>('idle');
  const [actionOutput, setActionOutput] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      if (view === 'registry' && !selectedRegistry) {
        const r = await fetch(`${API_BASE}/api/v1/system/registry/`);
        const d = await r.json();
        setRegistryList(d.registries || []);
        setData(null);
      } else if (view === 'registry' && selectedRegistry) {
        const r = await fetch(`${API_BASE}/api/v1/system/registry/${selectedRegistry}`);
        setData(await r.json());
      } else if (view !== 'health') {
        // config, sbom, kc — load automatically
        const endpoints: Record<string, string> = {
          config: `${API_BASE}/api/v1/system/config`, sbom: `${API_BASE}/api/v1/system/sbom`, kc: `${API_BASE}/api/v1/system/kc`,
        };
        const r = await fetch(endpoints[view] || `${API_BASE}/api/v1/system/config`);
        setData(await r.json());
      }
      // health is manual — don't auto-load
    } catch { /* */ } finally { setLoading(false); }
  }, [view, selectedRegistry]);

  useEffect(() => { load(); }, [load]);

  const runHealth = async () => {
    setActionStatus('running'); setActionOutput(null);
    try {
      const r = await fetch(`${API_BASE}/api/v1/system/health`);
      const d = await r.json();
      setData(d);
      // Go backend: {status: "ok"/"warning", checks: {...}, issues: [...]}
      const hasIssues = d.issues && d.issues.length > 0;
      setActionStatus(hasIssues ? 'error' : 'success');
      setActionOutput(d.output || JSON.stringify(d, null, 2));
    } catch (e) {
      setActionStatus('error');
      setActionOutput(`Health check failed: ${(e as Error).message}`);
    }
  };

  if (loading && view !== 'health') return <div className="card skeleton-card"><div className="skeleton skeleton-title" /><div className="skeleton skeleton-text" /></div>;

  // Health view — on-demand button
  if (view === 'health') {
    return (
      <>
        <div className="card">
          <h3>🏥 System Diagnosis</h3>
          <p style={{ fontSize: 12, color: 'var(--text-secondary)', marginBottom: 16 }}>
            Runs Go native health check (doctor). Checks all OVAV subsystems.
          </p>
          <button className="btn btn-primary" onClick={runHealth} disabled={actionStatus === 'running'}>
            {actionStatus === 'running' ? '⏳ Running Diagnosis...' : '▶ Run System Diagnosis'}
          </button>
        </div>
        <ResultPanel status={actionStatus} title="System Diagnosis" output={actionOutput}
          onRunAgain={runHealth}
          onClear={() => { setActionStatus('idle'); setActionOutput(null); setData(null); }} />
      </>
    );
  }

  // Other views
  const renderContent = () => {
    if (!data) return <div className="empty-state"><div className="empty-icon">⚙</div><div className="empty-title">No data</div></div>;
    if (view === 'config') return <pre style={{ maxHeight: 500 }}>{JSON.stringify(data, null, 2).substring(0, 5000)}</pre>;
    if (view === 'registry') {
      if (!selectedRegistry) {
        return (
          <div className="flex-col gap-4">
            {registryList.map(r => (
              <div key={r} className="btn btn-sm" onClick={() => setSelectedRegistry(r)} style={{ textAlign: 'left' }}>📄 {r}</div>
            ))}
            {registryList.length === 0 && <div className="dim">No registries found.</div>}
          </div>
        );
      }
      return <pre style={{ maxHeight: 500 }}>{JSON.stringify(data?.data || data, null, 2).substring(0, 5000)}</pre>;
    }
    if (view === 'sbom') return <pre style={{ maxHeight: 500 }}>{JSON.stringify(data, null, 2).substring(0, 5000)}</pre>;
    if (view === 'kc') return <pre style={{ maxHeight: 500 }}>{JSON.stringify(data, null, 2).substring(0, 3000)}</pre>;
    return <pre>{JSON.stringify(data, null, 2)}</pre>;
  };

  return (
    <div className="card">
      <h3>{view === 'config' ? '⚙ Configuration' : view === 'registry' ? (selectedRegistry ? `📄 ${selectedRegistry}` : '📂 Registries') : view === 'sbom' ? '📦 SBOM' : '🧬 Knowledge Compiler'}</h3>
      {view === 'registry' && selectedRegistry && <button className="btn btn-sm mb-8" onClick={() => setSelectedRegistry('')}>← Back to list</button>}
      {renderContent()}
      <div className="flex-row mt-8 gap-8"><button className="btn btn-sm" onClick={load}>↻ Refresh</button></div>
    </div>
  );
}
