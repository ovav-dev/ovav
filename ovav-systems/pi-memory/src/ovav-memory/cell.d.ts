/**
 * OVAV MEMORY v3 — Cell type
 * Lightweight event-signature lookup cell for live memory injection.
 */
export declare const PrivacyTag: {
    readonly PUBLIC: "public";
    readonly PROJECT: "project";
    readonly SENSITIVE: "sensitive";
    readonly SECRET: "secret";
};
export type PrivacyTag = (typeof PrivacyTag)[keyof typeof PrivacyTag];
export interface Cell {
    id: string;
    eventSignature: string;
    summary: string;
    detailRef: string;
    weight: number;
    tags: string[];
    harnessScope: string[];
    privacy: PrivacyTag;
    createdAt: string;
    lastHelpedAt: string;
    injectionCount: number;
    lastInjectedAt: string;
}
export interface CellCreateInput {
    eventSignature: string;
    summary: string;
    detailRef?: string;
    tags?: string[];
    harnessScope?: string[];
    privacy?: PrivacyTag;
}
export declare function isInjectable(cell: Cell): boolean;
export declare function createCellId(): string;
export declare function buildEventSignature(file: string, signal: string, detail: string): string;
