/**
 * OVAV MEMORY v3 — CellStore Unit Tests
 * Using Node.js built-in test runner
 */

import { test, describe, beforeEach } from "node:test";
import assert from "node:assert";
import { CellStore } from "../dist/ovav-memory/cellstore.js";
import { PrivacyTag } from "../dist/ovav-memory/cell.js";
import fs from "node:fs/promises";
import path from "node:path";

const TEST_DIR = "/tmp/ovav-cellstore-test";

describe("CellStore", () => {
  beforeEach(async () => {
    await fs.rm(TEST_DIR, { recursive: true, force: true });
    await fs.mkdir(path.join(TEST_DIR, ".ovav/runtime/livemem/cells"), { recursive: true });
  });

  test("saves and retrieves a cell", async () => {
    const store = new CellStore(TEST_DIR);
    await store.load();

    const cell = await store.save({
      eventSignature: "readme.md:error_loop:ENOENT",
      summary: "When readme.md is missing, create it from template.",
      tags: ["error-recovery"],
    });

    assert.ok(cell.id);
    assert.strictEqual(cell.weight, 0.5);
    assert.strictEqual(cell.eventSignature, "readme.md:error_loop:ENOENT");

    const found = store.get(cell.id);
    assert.notStrictEqual(found, undefined);
    assert.strictEqual(
      found!.summary,
      "When readme.md is missing, create it from template."
    );
  });

  test("indexes cells by event signature", async () => {
    const store = new CellStore(TEST_DIR);
    await store.load();

    await store.save({ eventSignature: "editor.ts:error_loop:ECONF", summary: "Cell 1" });
    await store.save({ eventSignature: "editor.ts:error_loop:ECONF", summary: "Cell 2" });
    await store.save({ eventSignature: "editor.ts:edit_burst:MAX", summary: "Cell 3" });

    const results = store.lookup("editor.ts:error_loop:ECONF");
    assert.strictEqual(results.length, 2);

    const empty = store.lookup("nonexistent:signal:detail");
    assert.strictEqual(empty.length, 0);
  });

  test("lookupBest returns highest weight cell above threshold", async () => {
    const store = new CellStore(TEST_DIR);
    await store.load();

    const lowCell = await store.save({
      eventSignature: "test.ts:error_loop:ERR",
      summary: "Low weight",
    });
    await store.recordOutcome(lowCell.id, false);
    await store.recordOutcome(lowCell.id, false);

    const highCell = await store.save({
      eventSignature: "test.ts:error_loop:ERR",
      summary: "High weight",
    });
    await store.recordOutcome(highCell.id, true);
    await store.recordOutcome(highCell.id, true);

    const best = store.lookupBest("test.ts:error_loop:ERR", 0.6);
    assert.notStrictEqual(best, null);
    assert.strictEqual(best!.id, highCell.id);
  });

  test("recordOutcome increases weight when helped=true", async () => {
    const store = new CellStore(TEST_DIR);
    await store.load();

    const cell = await store.save({
      eventSignature: "test.ts:stuck:LOOP",
      summary: "Test",
    });
    const weightBefore = cell.weight; // Capture before mutation

    await store.recordOutcome(cell.id, true);

    const updated = store.get(cell.id)!;
    assert.ok(updated.weight > weightBefore);
    assert.strictEqual(updated.injectionCount, 1);
  });

  test("recordOutcome decreases weight when helped=false", async () => {
    const store = new CellStore(TEST_DIR);
    await store.load();

    const cell = await store.save({
      eventSignature: "test.ts:stuck:LOOP",
      summary: "Test",
    });

    const weightBefore = cell.weight; // Capture before mutation
    await store.recordOutcome(cell.id, false);

    const updated = store.get(cell.id)!;
    assert.ok(updated.weight < weightBefore);
    assert.strictEqual(updated.injectionCount, 1);
  });

  test("load rebuilds index from filesystem", async () => {
    const store1 = new CellStore(TEST_DIR);
    await store1.load();

    await store1.save({ eventSignature: "a:b:c", summary: "Persisted" });

    // New instance loads from same directory
    const store2 = new CellStore(TEST_DIR);
    await store2.load();

    const cells = store2.all();
    assert.ok(cells.length >= 1);
  });

  test("truncates summary to 500 chars", async () => {
    const store = new CellStore(TEST_DIR);
    await store.load();

    const longSummary = "x".repeat(600);
    const cell = await store.save({
      eventSignature: "test.ts:x:y",
      summary: longSummary,
    });

    assert.ok(cell.summary.length <= 500);
  });

  test("all() returns every cell", async () => {
    const store = new CellStore(TEST_DIR);
    await store.load();

    await store.save({ eventSignature: "a:b:c", summary: "1" });
    await store.save({ eventSignature: "d:e:f", summary: "2" });

    const all = store.all();
    assert.ok(all.length >= 2);
  });

  test("signatures() returns unique event signatures", async () => {
    const store = new CellStore(TEST_DIR);
    await store.load();

    await store.save({ eventSignature: "a:b:c", summary: "1" });
    await store.save({ eventSignature: "a:b:c", summary: "2" });
    await store.save({ eventSignature: "x:y:z", summary: "3" });

    const sigs = store.signatures();
    assert.ok(sigs.includes("a:b:c"));
    assert.ok(sigs.includes("x:y:z"));
    assert.strictEqual(sigs.length, 2);
  });

  test("saves cells in memory when filesystem is unavailable (pi.dev sandbox)", async () => {
    // Use a path that will fail all filesystem operations
    const sandboxStore = new CellStore("/proc/0/nonexistent");
    await sandboxStore.load(); // Should not throw

    // save() should not throw even when filesystem is blocked
    const cell = await sandboxStore.save({
      eventSignature: "test:sandbox:SANDBOX",
      summary: "This cell lives only in memory",
    });

    assert.strictEqual(cell.eventSignature, "test:sandbox:SANDBOX");
    assert.strictEqual(cell.weight, 0.5);
    assert.ok((sandboxStore as any).memoryOnly === true);

    // lookup should still work
    const found = sandboxStore.lookup("test:sandbox:SANDBOX");
    assert.strictEqual(found.length, 1);
    assert.strictEqual(found[0].id, cell.id);
  });
});
