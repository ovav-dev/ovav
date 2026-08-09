import { useState, useEffect } from 'react';
import { API_BASE } from '../config';
import type { ToastFn } from '../types';

interface OpStatus {
  status: string;
  detail?: string;
  plans?: number;
  files?: string[];
  changes?: number;
  checks?: Record<string, string>;
  managed?: number;
  current?: string;
}

interface OperationsResponse {
  install: OpStatus;
  backup: OpStatus;
  deploy: OpStatus;
  sync: OpStatus;
  qa: OpStatus;
  surfaces: OpStatus;
  segment: OpStatus;
}

const statusBadge = (s: string) => {
  const map: Record<string, { cls: string; label: string }> = {
    available: { cls: 'badge-ok', label: 'READY' },
    clean: { cls: 'badge-ok', label: 'CLEAN' },
    dirty: { cls: 'badge-warn', label: 'DIRTY' },
    active: { cls: 'badge-ok', label: 'ACTIVE' },
    not_available: { cls: 'badge-muted', label: 'N/A' },
    not_configured: { cls: 'badge-muted', label: 'NONE' },
    unknown: { cls: 'badge-muted', label: '?' },
    error: { cls: 'badge-fail', label: 'ERROR' },
    has_warnings: { cls: 'badge-warn', label: 'WARN' },
  };
  const m = map[s] || { cls: 'badge-muted', label: s };
  return <span className={`badge ${m.cls}`}>{m.label}</span>;
};

export default function OperationsSection({ toast }: { toast: ToastFn }) {
  const [ops, setOps] = useState<OperationsResponse | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let active = true;
    const load = async () => {
      try {
        const r = await globalThis.fetch(`${API_BASE}/api/v1/system/operations`);
        if (r.ok && active) {
          const data = await r.json();
          setOps(data);
        }
      } catch { /* silent */ }
      if (active) setLoading(false);
    };
    load();
    const h = () => { load(); };
    window.addEventListener('cpanel-refresh', h);
    return () => { active = false; window.removeEventListener('cpanel-refresh', h); };
  }, []);

  if (loading) return <div className="loading">Loading operations…</div>;
  if (!ops) return <div className="empty-state">Operations data unavailable</div>;

  const sections: { key: keyof OperationsResponse; icon: string; label: string }[] = [
    { key: 'install', icon: '\u{1F4E6}', label: 'Install Pipeline' },
    { key: 'backup', icon: '\u{1F4BE}', label: 'Backup & Restore' },
    { key: 'deploy', icon: '\u{1F680}', label: 'Deploy' },
    { key: 'sync', icon: '\u{1F504}', label: 'Sync Status' },
    { key: 'qa', icon: '\u{1F9EA}', label: 'QA Gates' },
    { key: 'surfaces', icon: '\u{1F3A8}', label: 'Surfaces' },
    { key: 'segment', icon: '\u{1F333}', label: 'Current Segment' },
  ];

  return (
    <div className="ops-grid">
      {sections.map(({ key, icon, label }) => {
        const op = ops[key];
        return (
          <div key={key} className="ops-card">
            <div className="ops-card-header">
              <span className="ops-icon">{icon}</span>
              <span className="ops-label">{label}</span>
              {statusBadge(op.status)}
            </div>
            <div className="ops-card-body">
              {key === 'install' && op.plans != null && (
                <div className="ops-detail">Plans: {op.plans}</div>
              )}
              {key === 'sync' && op.changes != null && (
                <div className={`ops-detail ${op.changes > 0 ? 'text-warn' : 'text-ok'}`}>
                  Changes: {op.changes}
                </div>
              )}
              {key === 'qa' && op.checks && (
                <div className="ops-detail">
                  {Object.entries(op.checks).map(([k, v]) => (
                    <div key={k} className="flex-row" style={{ gap: 8 }}>
                      <span className="text-muted">{k}:</span>
                      <span className={v === 'available' ? 'text-ok' : 'text-muted'}>{v === 'available' ? '\u2713' : '\u2717'}</span>
                    </div>
                  ))}
                </div>
              )}
              {key === 'segment' && op.current && (
                <div className="ops-detail">{op.current}</div>
              )}
              {op.detail && <div className="ops-detail text-muted">{op.detail}</div>}
            </div>
          </div>
        );
      })}
    </div>
  );
}
