/**
 * OVAV MEMORY v3 — Detector + Profiler Unit Tests
 * Using Node.js built-in test runner
 */

import { test, describe, beforeEach } from "node:test";
import assert from "node:assert";
import { LiveProfiler, SignalType } from "../dist/ovav-memory/profiler.js";
import { Detector } from "../dist/ovav-memory/detector.js";
import { CellStore } from "../dist/ovav-memory/cellstore.js";
import fs from "node:fs/promises";
import path from "node:path";

const TEST_DIR = "/tmp/ovav-detector-test";

describe("LiveProfiler", () => {
  test("classifies RETRY_LOOP after 3-4 error events", async () => {
    const profiler = new LiveProfiler(5); // 5ms window
    for (let i = 0; i < 3; i++) {
      profiler.feed("editor.ts", "error", { tool: "Write", error: "ECONF" });
      await new Promise(r => setTimeout(r, 2));
    }
    // 3-4 errors → RETRY_LOOP (5+ errors needed for ERROR_LOOP)
    assert.strictEqual(profiler.classify(), SignalType.RETRY_LOOP);
  });

  test("classifies ERROR_LOOP after 5+ error events", async () => {
    const profiler = new LiveProfiler(5); // 5ms window
    for (let i = 0; i < 5; i++) {
      profiler.feed("editor.ts", "error", { tool: "Write", error: "ECONF" });
      await new Promise(r => setTimeout(r, 2));
    }
    assert.strictEqual(profiler.classify(), SignalType.ERROR_LOOP);
  });

  test("classifies EDIT_BURST after 4+ write events", async () => {
    const profiler = new LiveProfiler(5);
    for (let i = 0; i < 4; i++) {
      profiler.feed("editor.ts", "write", {});
      await new Promise(r => setTimeout(r, 2));
    }
    assert.strictEqual(profiler.classify(), SignalType.EDIT_BURST);
  });

  test("returns null when window is empty", () => {
    const profiler = new LiveProfiler(5);
    assert.strictEqual(profiler.classify(), null);
  });

  test("prunes events outside sliding window", async () => {
    const profiler = new LiveProfiler(1); // 1ms window
    profiler.feed("a.ts", "error", {});
    await new Promise(r => setTimeout(r, 3));
    assert.strictEqual(profiler.classify(), null);
  });

  test("classifyAll returns all active signals", async () => {
    const profiler = new LiveProfiler(10);
    for (let i = 0; i < 3; i++) {
      profiler.feed("a.ts", "error", {});
      await new Promise(r => setTimeout(r, 1));
    }
    for (let i = 0; i < 4; i++) {
      profiler.feed("b.ts", "write", {});
      await new Promise(r => setTimeout(r, 1));
    }
    const signals = profiler.classifyAll();
    const types = signals.map(s => s.signal);
    // 3 errors → RETRY_LOOP (5+ needed for ERROR_LOOP)
    assert.ok(types.includes(SignalType.RETRY_LOOP));
    assert.ok(types.includes(SignalType.EDIT_BURST));
  });

  test("reset clears all events", async () => {
    const profiler = new LiveProfiler(5);
    for (let i = 0; i < 3; i++) {
      profiler.feed("a.ts", "error", {});
      await new Promise(r => setTimeout(r, 1));
    }
    profiler.reset();
    assert.strictEqual(profiler.classify(), null);
  });
});

describe("Detector", () => {
  beforeEach(async () => {
    await fs.rm(TEST_DIR, { recursive: true, force: true });
    await fs.mkdir(path.join(TEST_DIR, ".ovav/runtime/livemem/cells"), { recursive: true });
  });

  test("returns null when no signal is detected", () => {
    const cellStore = new CellStore(TEST_DIR);
    const detector = new Detector(cellStore);

    const decision = detector.decide({
      tool: "Read",
      file: "readme.md",
      result: { content: "hello" },
    });

    assert.strictEqual(decision.inject, null);
    assert.strictEqual(decision.signal, null);
  });

  test("triggers injection when cell matches signal", async () => {
    const cellStore = new CellStore(TEST_DIR);
    await cellStore.load();

    // Pre-create a cell with RETRY_LOOP signal (3 events = minimum signal)
    const cell = await cellStore.save({
      eventSignature: "editor.ts:retry_loop:ECONF",
      summary: "When you see ECONF errors repeatedly, try restarting.",
      tags: ["error-recovery"],
    });
    // Boost weight above default minWeight (0.6) threshold
    await cellStore.recordOutcome(cell.id, true);

    const detector = new Detector(cellStore, 0.6);

    // 3 error events → RETRY_LOOP (3 is minimum for any signal)
    for (let i = 0; i < 3; i++) {
      detector.decide({
        tool: "Write",
        file: "editor.ts",
        result: {},
        error: "ECONF",
      });
      await new Promise(r => setTimeout(r, 1));
    }

    const decision = detector.decide({
      tool: "Write",
      file: "editor.ts",
      result: {},
      error: "ECONF",
    });

    assert.notStrictEqual(decision.inject, null);
    assert.strictEqual(decision.signal, SignalType.RETRY_LOOP);
  });

  test("triggers ERROR_LOOP injection with 5+ rapid errors", async () => {
    const cellStore = new CellStore(TEST_DIR);
    await cellStore.load();

    const cell = await cellStore.save({
      eventSignature: "editor.ts:error_loop:ECONF",
      summary: "When you see 5+ ECONF errors, the LSP server is stuck.",
      tags: ["error-recovery"],
    });
    await cellStore.recordOutcome(cell.id, true);

    const detector = new Detector(cellStore, 0.6);

    // Feed 6 errors synchronously to bypass async timing issues
    const profiler = (detector as any).profiler;
    for (let i = 0; i < 6; i++) {
      profiler.feed("editor.ts", "error", { tool: "Write", error: "ECONF" });
    }

    assert.strictEqual(profiler.classify(), SignalType.ERROR_LOOP);

    const decision = detector.decide({
      tool: "Write",
      file: "editor.ts",
      result: {},
      error: "ECONF",
    });

    assert.notStrictEqual(decision.inject, null);
    assert.strictEqual(decision.signal, SignalType.ERROR_LOOP);
  });

  test("skips cells with weight below 0.6", async () => {
    const cellStore = new CellStore(TEST_DIR);
    await cellStore.load();

    const lowCell = await cellStore.save({
      eventSignature: "editor.ts:error_loop:ECONF",
      summary: "Low weight cell",
    });
    // Drop weight below threshold
    await cellStore.recordOutcome(lowCell.id, false);
    await cellStore.recordOutcome(lowCell.id, false);
    await cellStore.recordOutcome(lowCell.id, false);

    const detector = new Detector(cellStore);

    for (let i = 0; i < 3; i++) {
      detector.decide({
        tool: "Write",
        file: "editor.ts",
        result: {},
        error: "ECONF",
      });
      await new Promise(r => setTimeout(r, 2));
    }

    const decision = detector.decide({
      tool: "Write",
      file: "editor.ts",
      result: {},
      error: "ECONF",
    });

    assert.strictEqual(decision.inject, null);
    assert.strictEqual(decision.reason, "no_matching_cell");
  });
});
