import type { Toast as ToastType } from '../hooks/useApi';

export default function ToastContainer({ toasts }: { toasts: ToastType[] }) {
  if (!toasts.length) return null;
  return (
    <div className="toast-container">
      {toasts.map((t) => (
        <div key={t.id} className={`toast toast-${t.type === 'ok' ? 'ok' : t.type === 'err' ? 'err' : 'info'}`}>
          {t.message}
        </div>
      ))}
    </div>
  );
}
