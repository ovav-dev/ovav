/**
 * OVAV MEMORY v3 — CellStore
 * Filesystem-backed Cell storage with event-signature indexing.
 */
import { type Cell, type CellCreateInput } from "./cell.js";
/**
 * CellStore persists Cells to the filesystem and maintains an in-memory
 * index from eventSignature → cellId[] for fast lookups.
 */
export declare class CellStore {
    private cells;
    private index;
    private basePath;
    private loaded;
    /** When true, filesystem writes are skipped (pi.dev sandbox or permission error) */
    private memoryOnly;
    constructor(basePath: string);
    /**
     * Load all cells from the filesystem and rebuild the index.
     * Idempotent — subsequent calls are no-ops if already loaded.
     */
    load(): Promise<void>;
    /**
     * Persist a new cell to the store.
     */
    save(input: CellCreateInput, privacy?: string): Promise<Cell>;
    /**
     * Lookup cells by eventSignature.
     * Returns cells sorted by weight descending.
     */
    lookup(sig: string): Cell[];
    /**
     * Find the best injectible cell for a given signature and minimum weight.
     */
    lookupBest(sig: string, minWeight?: number): Cell | null;
    /**
     * Record the outcome of a cell injection.
     * @param cellId - cell that was injected
     * @param helped - whether the injection was useful
     */
    recordOutcome(cellId: string, helped: boolean): Promise<void>;
    /**
     * Get a cell by ID.
     */
    get(id: string): Cell | undefined;
    /**
     * Return all cells.
     */
    all(): Cell[];
    /**
     * Return all unique event signatures currently in the index.
     */
    signatures(): string[];
    private reindex;
    private writeCell;
}
