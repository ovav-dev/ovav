/* ═══════════ OVAV cPanel v4 — Types ═══════════ */

export interface GitInfo {
  branch: string;
  head: string;
  commits: string;
  dirty: string;
  last_commit: string;
}

export interface SystemInfo {
  agents: string;
  python: string;
}

export interface EconomyInfo {
  session_cost: number;
  session_pct: number;
  monthly_cost: number;
  monthly_pct: number;
  model: string;
}

export interface SessionInfo {
  uptime: string;
  canary_alarms: number;
}

export interface StatusResponse {
  timestamp: string;
  git: GitInfo;
  system: SystemInfo;
  economy: EconomyInfo;
  session: SessionInfo;
}

export interface ValidatorCheck {
  name: string;
  ok: boolean;
}

export interface ValidatorListResponse {
  overall: string;
  score: number;
  pass: number;
  fail: number;
  checks: ValidatorCheck[];
}

export interface TaskStatus {
  status: 'queued' | 'running' | 'complete' | 'error';
  progress?: number;
  result?: {
    returncode?: number;
    output?: string;
    error?: string;
  };
}

export interface AuditEntry {
  iso_time: string;
  category: string;
  tier: string;
  details: Record<string, unknown>;
}

export interface AuditLogResponse {
  entries: AuditEntry[];
  total: number;
  chain_intact: boolean;
}

export interface ProfileResult {
  ok: boolean;
  output?: string;
  error?: string;
}

export type ToastType = 'info' | 'ok' | 'err';
export type ToastFn = (msg: string, type?: ToastType) => void;

export type ViewId = 'dashboard' | 'validators' | 'security' | 'profiles' | 'logs';

export interface NavItem {
  id: ViewId;
  icon: string;
  label: string;
}
