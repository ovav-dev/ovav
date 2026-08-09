import { useState, useEffect, useCallback, useMemo } from 'react';
import { API_BASE } from '../config';
import ResultPanel from '../components/ResultPanel';
import type { ToastType } from '../types';

interface SectionProps { toast: (msg: string, type?: ToastType) => void; view: string; }

interface Card {
  id?: string;
  topic?: string;
  status?: string;
  summary?: string;
  tags?: string[];
  operational_rule?: string;
  evidence?: string[];
  last_confirmed?: string;
  confidence?: number;
  deprecation_reason?: string;
}

export default function MemorySection({ toast, view }: SectionProps) {
  const [cards, setCards] = useState<Card[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const endpoints: Record<string, string> = {
    ledger: `${API_BASE}/api/v1/memory/ledger`,
    beliefs: `${API_BASE}/api/v1/memory/beliefs`,
    capsules: `${API_BASE}/api/v1/memory/capsules`,
  };

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const r = await fetch(endpoints[view] || `${API_BASE}/api/v1/memory/status`);
      const d = await r.json();
      setCards(d.cards || d.beliefs || d.capsules || []);
      setTotal(d.total || 0);
    } catch { /* */ }
    finally { setLoading(false); }
  }, [view]);

  useEffect(() => { load(); }, [load]);
  useEffect(() => { const h = () => load(); window.addEventListener('cpanel-refresh', h); return () => window.removeEventListener('cpanel-refresh', h); }, [load]);

  // Filter + search
  const filtered = useMemo(() => {
    let result = cards;
    if (search) {
      const q = search.toLowerCase();
      result = result.filter(c =>
        (c.id || '').toLowerCase().includes(q) ||
        (c.topic || '').toLowerCase().includes(q) ||
        (c.summary || '').toLowerCase().includes(q) ||
        (c.tags || []).some(t => t.toLowerCase().includes(q))
      );
    }
    if (statusFilter) {
      result = result.filter(c => c.status === statusFilter);
    }
    return result;
  }, [cards, search, statusFilter]);

  // Stats
  const stats = useMemo(() => {
    const byStatus: Record<string, number> = {};
    const byTag: Record<string, number> = {};
    cards.forEach(c => {
      const s = c.status || 'unknown';
      byStatus[s] = (byStatus[s] || 0) + 1;
      (c.tags || []).forEach(t => {
        byTag[t] = (byTag[t] || 0) + 1;
      });
    });
    return { byStatus, byTag };
  }, [cards]);

  const statuses = [...new Set(cards.map(c => c.status || 'unknown'))];

  if (loading) return <div className="card skeleton-card"><div className="skeleton skeleton-title" /><div className="skeleton skeleton-text" /><div className="skeleton skeleton-text short" /></div>;

  const renderCard = (c: Card, i: number) => {
    const id = c.id || c.topic || `card-${i}`;
    const isExpanded = expandedId === id;
    return (
      <div key={i} className="card mt-8" style={{ background: 'var(--bg-primary)', border: '1px solid var(--border-default)', cursor: 'pointer' }}
        onClick={() => setExpandedId(isExpanded ? null : id)}>
        <div className="row">
          <span className="row-v ok" style={{ fontSize: 12 }}>{id.length > 50 ? id.substring(0, 50) + '...' : id}</span>
          <div className="flex-row gap-4">
            <span className={`tag ${c.status === 'active' ? 'tag-ok' : c.status === 'deprecated' ? 'tag-warn' : 'tag-fail'}`}>
              {c.status || '?'}
            </span>
            {c.confidence !== undefined && (
              <span className="tag tag-ok" style={{ fontSize: 10 }}>
                {(c.confidence * 100).toFixed(0)}%
              </span>
            )}
          </div>
        </div>
        <div style={{ fontSize: 12, color: 'var(--text-secondary)', marginTop: 4 }}>
          {c.summary || 'No summary'}
        </div>
        {c.tags && c.tags.length > 0 && (
          <div className="flex-row mt-4 gap-4" style={{ flexWrap: 'wrap' }}>
            {c.tags.map(t => (
              <span key={t} className="tag tag-ok" style={{ fontSize: 10 }}>{t}</span>
            ))}
          </div>
        )}

        {isExpanded && (
          <div className="mt-8" style={{ borderTop: '1px solid var(--border-default)', paddingTop: 8 }}>
            {c.operational_rule && (
              <div className="mb-8">
                <div style={{ fontSize: 10, color: 'var(--text-muted)', marginBottom: 2 }}>OPERATIONAL RULE</div>
                <pre style={{ fontSize: 11, padding: 8, maxHeight: 150 }}>{c.operational_rule}</pre>
              </div>
            )}
            {c.evidence && c.evidence.length > 0 && (
              <div className="mb-8">
                <div style={{ fontSize: 10, color: 'var(--text-muted)', marginBottom: 2 }}>EVIDENCE ({c.evidence.length} files)</div>
                <div style={{ fontSize: 10, color: 'var(--text-secondary)', maxHeight: 100, overflow: 'auto' }}>
                  {c.evidence.map((e, j) => <div key={j}>• {e}</div>)}
                </div>
              </div>
            )}
            {c.last_confirmed && (
              <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>Last confirmed: {c.last_confirmed}</div>
            )}
            {c.deprecation_reason && (
              <div className="mt-4" style={{ fontSize: 11, color: 'var(--warning)' }}>
                ⚠ Deprecated: {c.deprecation_reason}
              </div>
            )}
          </div>
        )}
      </div>
    );
  };

  if (view === 'ledger') {
    return (
      <>
        {/* Stats bar */}
        {Object.keys(stats.byStatus).length > 0 && (
          <div className="card mb-12">
            <h3>📊 Ledger Stats</h3>
            <div className="flex-row gap-8" style={{ flexWrap: 'wrap' }}>
              {Object.entries(stats.byStatus).map(([k, v]) => (
                <span key={k} className={`tag ${k === 'active' ? 'tag-ok' : k === 'deprecated' ? 'tag-warn' : 'tag-fail'}`}>
                  {k}: {v}
                </span>
              ))}
            </div>
            {Object.keys(stats.byTag).length > 0 && (
              <div className="flex-row mt-8 gap-4" style={{ flexWrap: 'wrap' }}>
                {Object.entries(stats.byTag).slice(0, 10).map(([k, v]) => (
                  <span key={k} className="tag tag-ok" style={{ fontSize: 10 }}>{k}:{v}</span>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Search + filter */}
        <div className="flex-row mb-12 gap-8">
          <input
            type="text"
            placeholder="Search cards by ID, topic, summary, or tag..."
            value={search}
            onChange={e => setSearch(e.target.value)}
            style={{ flex: 1 }}
          />
          <select value={statusFilter} onChange={e => setStatusFilter(e.target.value)}>
            <option value="">All statuses</option>
            {statuses.map(s => <option key={s} value={s}>{s}</option>)}
          </select>
          <button className="btn btn-sm" onClick={load}>↻</button>
        </div>

        {/* Cards */}
        <div className="card">
          <h3>🧠 Context Ledger · {filtered.length}/{total} cards</h3>
          {filtered.length ? (
            filtered.map((c, i) => renderCard(c, i))
          ) : (
            <div className="empty-state">
              <div className="empty-icon">{search ? '🔍' : '📭'}</div>
              <div className="empty-title">{search ? 'No matching cards' : 'No cards in ledger'}</div>
              <div className="empty-desc">{search ? 'Try a different search term.' : 'The context ledger will populate as OVAV learns.'}</div>
            </div>
          )}
        </div>
      </>
    );
  }

  if (view === 'beliefs') {
    return (
      <div className="card">
        <h3>💭 Beliefs · {total} active</h3>
        {filtered.length ? filtered.map((c, i) => renderCard(c, i))
          : <div className="empty-state"><div className="empty-icon">💭</div><div className="empty-title">No active beliefs</div></div>}
        <div className="flex-row mt-8 gap-8"><button className="btn btn-sm" onClick={load}>↻ Refresh</button></div>
      </div>
    );
  }

  // capsules
  return (
    <div className="card">
      <h3>📦 Session Capsules · {total} recent</h3>
      {filtered.length ? filtered.map((c: any, i: number) => (
        <div key={i} className="card mt-8" style={{ background: 'var(--bg-primary)' }}>
          <div className="row"><span className="row-v" style={{ fontSize: 11 }}>{c.id}</span><span className="tag tag-ok">{c.branch}</span></div>
          {c.created && <div style={{ fontSize: 10, color: 'var(--text-muted)', marginTop: 2 }}>{c.created}</div>}
        </div>
      )) : <div className="empty-state"><div className="empty-icon">📦</div><div className="empty-title">No capsules</div></div>}
      <div className="flex-row mt-8 gap-8"><button className="btn btn-sm" onClick={load}>↻ Refresh</button></div>
    </div>
  );
}
