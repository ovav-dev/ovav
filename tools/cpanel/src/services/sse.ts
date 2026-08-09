/* ═══════════ OVAV SSE Service ═══════════ */
import { API_BASE } from '../config';

type EventHandler = (data: unknown) => void;

class SSEService {
  private source: EventSource | null = null;
  private handlers = new Map<string, Set<EventHandler>>();
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;

  connect() {
    this.disconnect();
    this.source = new EventSource(`${API_BASE}/api/v1/events`);

    this.source.addEventListener('connected', (e) => this.emit('connected', JSON.parse(e.data)));
    this.source.addEventListener('heartbeat', (e) => this.emit('heartbeat', JSON.parse(e.data)));
    this.source.addEventListener('validator.progress', (e) => this.emit('validator.progress', JSON.parse(e.data)));

    this.source.onerror = () => {
      this.emit('error', { message: 'SSE connection lost' });
      this.reconnectTimer = setTimeout(() => this.connect(), 5000);
    };
  }

  on(event: string, fn: EventHandler): () => void {
    if (!this.handlers.has(event)) this.handlers.set(event, new Set());
    this.handlers.get(event)!.add(fn);
    return () => this.handlers.get(event)?.delete(fn);
  }

  private emit(event: string, data: unknown) {
    this.handlers.get(event)?.forEach((fn) => fn(data));
  }

  disconnect() {
    if (this.reconnectTimer) { clearTimeout(this.reconnectTimer); this.reconnectTimer = null; }
    if (this.source) { this.source.close(); this.source = null; }
  }
}

export const sse = new SSEService();
// test hook
// test2
// test3
