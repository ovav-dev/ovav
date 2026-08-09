/**
 * OVAV MEMORY v3 — Cell type
 * Lightweight event-signature lookup cell for live memory injection.
 */

export const PrivacyTag = {
  PUBLIC: "public",
  PROJECT: "project",
  SENSITIVE: "sensitive",
  SECRET: "secret" // NEVER inject — excluded from all lookups
} as const;

export type PrivacyTag = (typeof PrivacyTag)[keyof typeof PrivacyTag];

export interface Cell {
  id: string;               // uuid v4
  eventSignature: string;   // "filepath:signalType:detail" e.g. "editor.ts:error_loop:ECONF"
  summary: string;          // ≤500 chars — injected into agent context
  detailRef: string;        // absolute path to full cell .md (on-demand load)
  weight: number;           // 0.0–1.0 — governs injection priority
  tags: string[];           // ["error-recovery", "loop", "typescript"]
  harnessScope: string[];   // ["pi", "opencode", "mimocode", "claude", "cursor"]
  privacy: PrivacyTag;      // controls where cell can be injected
  createdAt: string;        // ISO 8601
  lastHelpedAt: string;     // ISO 8601 — updated when injection helps
  injectionCount: number;   // total times this cell was injected
  lastInjectedAt: string;   // ISO 8601
}

export interface CellCreateInput {
  eventSignature: string;
  summary: string;
  detailRef?: string;
  tags?: string[];
  harnessScope?: string[];
  privacy?: PrivacyTag;
}

export function isInjectable(cell: Cell): boolean {
  return cell.privacy !== PrivacyTag.SECRET && cell.weight > 0;
}

export function createCellId(): string {
  return crypto.randomUUID();
}

export function buildEventSignature(
  file: string,
  signal: string,
  detail: string
): string {
  return `${file}:${signal}:${detail}`;
}
