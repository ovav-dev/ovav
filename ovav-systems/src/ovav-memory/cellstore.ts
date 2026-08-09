/**
 * OVAV MEMORY v3 — CellStore
 * Filesystem-backed Cell storage with event-signature indexing.
 */

import * as fs from "node:fs/promises";
import * as path from "node:path";
import { createCellId, type Cell, type CellCreateInput } from "./cell.js";

const CELLS_DIR = ".ovav/runtime/livemem/cells";

/**
 * CellStore persists Cells to the filesystem and maintains an in-memory
 * index from eventSignature → cellId[] for fast lookups.
 */
export class CellStore {
  private cells: Map<string, Cell> = new Map();
  private index: Map<string, string[]> = new Map(); // sig → [cellId, ...]
  private basePath: string;
  private loaded = false;
  /** When true, filesystem writes are skipped (pi.dev sandbox or permission error) */
  private memoryOnly = false;

  constructor(basePath: string) {
    this.basePath = basePath;
  }

  /**
   * Load all cells from the filesystem and rebuild the index.
   * Idempotent — subsequent calls are no-ops if already loaded.
   */
  async load(): Promise<void> {
    if (this.loaded) return;

    const cellsDir = path.join(this.basePath, CELLS_DIR);

    let entries: string[];
    try {
      entries = await fs.readdir(cellsDir);
    } catch {
      // Directory doesn't exist yet OR sandbox blocks filesystem — empty store
      this.loaded = true;
      this.memoryOnly = true;
      return;
    }

    const jsonFiles = entries.filter(f => f.endsWith(".json"));

    await Promise.all(
      jsonFiles.map(async file => {
        try {
          const content = await fs.readFile(path.join(cellsDir, file), "utf-8");
          const cell: Cell = JSON.parse(content);
          this.cells.set(cell.id, cell);
          this.reindex(cell);
        } catch {
          // Skip corrupted files
        }
      })
    );

    this.loaded = true;
  }

  /**
   * Persist a new cell to the store.
   */
  async save(input: CellCreateInput, privacy = "public"): Promise<Cell> {
    const cell: Cell = {
      id: createCellId(),
      eventSignature: input.eventSignature,
      summary: input.summary.slice(0, 500),
      detailRef: input.detailRef ?? "",
      weight: 0.5, // start at neutral weight
      tags: input.tags ?? [],
      harnessScope: input.harnessScope ?? ["pi"],
      privacy: privacy as Cell["privacy"],
      createdAt: new Date().toISOString(),
      lastHelpedAt: new Date().toISOString(),
      injectionCount: 0,
      lastInjectedAt: new Date().toISOString()
    };

    this.cells.set(cell.id, cell);
    this.reindex(cell);

    await this.writeCell(cell);
    return cell;
  }

  /**
   * Lookup cells by eventSignature.
   * Returns cells sorted by weight descending.
   */
  lookup(sig: string): Cell[] {
    const ids = this.index.get(sig) ?? [];
    const results: Cell[] = [];

    for (const id of ids) {
      const cell = this.cells.get(id);
      if (cell) results.push(cell);
    }

    return results.sort((a, b) => b.weight - a.weight);
  }

  /**
   * Find the best injectible cell for a given signature and minimum weight.
   */
  lookupBest(sig: string, minWeight = 0.6): Cell | null {
    const cells = this.lookup(sig);
    return cells.find(c => c.weight >= minWeight) ?? null;
  }

  /**
   * Record the outcome of a cell injection.
   * @param cellId - cell that was injected
   * @param helped - whether the injection was useful
   */
  async recordOutcome(cellId: string, helped: boolean): Promise<void> {
    const cell = this.cells.get(cellId);
    if (!cell) return;

    cell.injectionCount++;
    cell.lastInjectedAt = new Date().toISOString();

    if (helped) {
      cell.weight = Math.min(1.0, cell.weight + 0.1);
      cell.lastHelpedAt = new Date().toISOString();
    } else {
      cell.weight = Math.max(0.0, cell.weight - 0.05);
    }

    await this.writeCell(cell);
  }

  /**
   * Get a cell by ID.
   */
  get(id: string): Cell | undefined {
    return this.cells.get(id);
  }

  /**
   * Return all cells.
   */
  all(): Cell[] {
    return Array.from(this.cells.values());
  }

  /**
   * Return all unique event signatures currently in the index.
   */
  signatures(): string[] {
    return Array.from(this.index.keys());
  }

  private reindex(cell: Cell): void {
    const existing = this.index.get(cell.eventSignature) ?? [];
    if (!existing.includes(cell.id)) {
      existing.push(cell.id);
      this.index.set(cell.eventSignature, existing);
    }
  }

  private async writeCell(cell: Cell): Promise<void> {
    if (this.memoryOnly) return;

    const cellsDir = path.join(this.basePath, CELLS_DIR);

    try {
      await fs.mkdir(cellsDir, { recursive: true });
      const filePath = path.join(cellsDir, `${cell.id}.json`);
      await fs.writeFile(filePath, JSON.stringify(cell, null, 2), "utf-8");
    } catch (err) {
      // pi.dev sandbox or permission error — fall back to memory-only
      this.memoryOnly = true;
      // Cell stays in memory; no crash
    }
  }
}
