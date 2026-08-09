import { type ReactNode } from 'react';

interface Props {
  status: 'idle' | 'running' | 'success' | 'error';
  title?: string;
  output?: string | null;
  error?: string | null;
  children?: ReactNode;
  onRunAgain?: () => void;
  onClear?: () => void;
}

export default function ResultPanel({ status, title, output, error, children, onRunAgain, onClear }: Props) {
  if (status === 'idle' && !output) return null;

  const borderColor = status === 'running' ? 'var(--accent)'
    : status === 'success' ? 'var(--success)'
    : status === 'error' ? 'var(--danger)'
    : 'var(--border-default)';

  const dotColor = status === 'running' ? 'var(--accent)'
    : status === 'success' ? 'var(--success)'
    : 'var(--danger)';

  return (
    <div className="card mt-12" style={{ borderColor, borderLeftWidth: 3 }}>
      <div className="flex-row mb-8" style={{ justifyContent: 'space-between' }}>
        <div className="flex-row gap-8">
          {status === 'running' && <span className="spinner" style={{ width: 14, height: 14 }} />}
          {status !== 'running' && <span className="status-dot" style={{ background: dotColor, boxShadow: `0 0 6px ${dotColor}` }} />}
          <strong style={{ fontSize: 13, color: dotColor }}>
            {title || (status === 'running' ? 'Running...' : status === 'success' ? 'Completed' : 'Error')}
          </strong>
        </div>
        <div className="flex-row gap-4">
          {onRunAgain && status !== 'running' && (
            <button className="btn btn-sm" onClick={onRunAgain}>↻ Run Again</button>
          )}
          {onClear && (
            <button className="btn btn-sm" onClick={onClear}>✕ Clear</button>
          )}
        </div>
      </div>

      {status === 'running' && !output && !error && (
        <div style={{ padding: '20px 0', textAlign: 'center', color: 'var(--text-secondary)' }}>
          <span className="spinner" /> Executing...
        </div>
      )}

      {error && (
        <pre style={{ borderColor: 'var(--danger)', color: 'var(--danger)', fontSize: 12 }}>
          {error}
        </pre>
      )}

      {output && (
        <pre style={{ fontSize: 12, maxHeight: 400, background: '#0a0e14', borderColor }}>
          {output}
        </pre>
      )}

      {children}
    </div>
  );
}
