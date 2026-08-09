/* ═══════════ OVAV API Service ═══════════ */
import type { StatusResponse, ValidatorListResponse, TaskStatus, AuditLogResponse, ProfileResult } from '../types';
import { API_BASE } from '../config';

const BASE = `${API_BASE}/api/v1`;

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const r = await fetch(`${BASE}${path}`, options);
  if (!r.ok) throw new Error(`API ${r.status}: ${path}`);
  return r.json();
}

export const api = {
  getStatus: () =>
    request<StatusResponse>('/status'),

  getValidators: () =>
    request<ValidatorListResponse>('/validators'),

  runValidators: () =>
    request<{ task_id: string; status: string }>('/validators/run', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: '{}',
    }),

  getValidatorStatus: (taskId: string) =>
    request<TaskStatus>(`/validators/status/${taskId}`),

  getAuditLog: (opts: { limit?: number; category?: string } = {}) => {
    const params = new URLSearchParams();
    if (opts.limit) params.set('limit', String(opts.limit));
    if (opts.category) params.set('category', opts.category);
    return request<AuditLogResponse>(`/security/audit-log?${params}`);
  },

  clearAlarms: () =>
    request<{ status: string; message: string }>('/security/canary-alarms', {
      method: 'DELETE',
    }),

  getProfiles: () =>
    request<ProfileResult>('/profiles'),

  applyProfile: (area: string, target = '.', dryRun = false) =>
    request<ProfileResult>('/profiles/apply', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ area, target, dry_run: dryRun }),
    }),

  getOperations: () =>
    request<Record<string, unknown>>('/system/operations'),

  // ── Auth Analytics (P2-B native) ───────────────────────────────
  getAuthAnalytics: (days = 7) =>
    request<any>(`/auth/analytics?days=${days}`),

  getAuthAuditLog: (opts: { email?: string; ip?: string; action?: string; status?: string; limit?: number; offset?: number } = {}) => {
    const params = new URLSearchParams();
    if (opts.email) params.set('email', opts.email);
    if (opts.ip) params.set('ip', opts.ip);
    if (opts.action) params.set('action', opts.action);
    if (opts.status) params.set('status', opts.status);
    if (opts.limit) params.set('limit', String(opts.limit));
    if (opts.offset) params.set('offset', String(opts.offset));
    return request<any>(`/auth/audit-log?${params}`);
  },
};
