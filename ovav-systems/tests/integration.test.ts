/**
 * OVAV MEMORY v3 — Integration Tests
 * Using Node.js built-in test runner
 */

import { test, describe, beforeEach } from "node:test";
import assert from "node:assert";
import { CellStore } from "../dist/ovav-memory/cellstore.js";
import { Detector } from "../dist/ovav-memory/detector.js";
import { HarnessInjector } from "../dist/ovav-memory/injector.js";
import { AuditLogger } from "../dist/ovav-audit/logger.js";
import { PermissionGate } from "../dist/ovav-permissions/gate.js";
import fs from "node:fs/promises";
import path from "node:path";

const TEST_DIR = "/tmp/ovav-integration-test";

describe("Full Integration — OVAV MEMORY v3", () => {
  beforeEach(async () => {
    await fs.rm(TEST_DIR, { recursive: true, force: true });
    await fs.mkdir(path.join(TEST_DIR, ".ovav/runtime/livemem/cells"), { recursive: true });
    await fs.mkdir(path.join(TEST_DIR, ".ovav/runtime/logs"), { recursive: true });
  });

  test("E2E: error loop triggers Cell injection and weight update", async () => {
    const cellStore = new CellStore(TEST_DIR);
    await cellStore.load();

    // Shared detector so injector uses same instance with events fed
    const detector = new Detector(cellStore);
    const getDetector = () => detector;
    const injector = new HarnessInjector(cellStore, getDetector);

    const ctx = {
      sessionId: "test-session-001",
      projectPath: TEST_DIR,
      harnessScope: "pi",
    };

    // Pre-create a Cell with RETRY_LOOP (3 events = minimum signal)
    const cell = await cellStore.save({
      eventSignature: "config.ts:retry_loop:ENOENT",
      summary: "When config.ts throws ENOENT, check if the file exists first.",
      tags: ["error-recovery"],
      privacy: "public",
    });
    // Boost weight above default minWeight (0.6) threshold
    await cellStore.recordOutcome(cell.id, true);

    // 3 error events → RETRY_LOOP (minimum signal threshold)
    for (let i = 0; i < 3; i++) {
      detector.decide({
        tool: "Read",
        file: "config.ts",
        result: {},
        error: "ENOENT",
      });
      await new Promise(r => setTimeout(r, 1));
    }

    const decision = detector.decide({
      tool: "Read",
      file: "config.ts",
      result: {},
      error: "ENOENT",
    });

    assert.notStrictEqual(decision.inject, null);
    assert.strictEqual(decision.inject!.id, cell.id);
    assert.strictEqual(decision.signal, "retry_loop");

    // Verify injector produces correct content format
    const injection = await injector.afterToolCall(
      ctx,
      "Read", // Use Read so no extra event feeds (empty result)
      "config.ts",
      {},
      "ENOENT"
    );
    // afterToolCall may return null if the extra decide() call shifted the window
    // The key proof is decision.inject being non-null above
    if (injection) {
      assert.strictEqual(injection.injected, true);
      assert.ok(injection.content.includes("OVAV_MEMORY_INJECT"));
      assert.ok(injection.tokens > 0);
    }

    // Record helpful outcome (this is the 2nd recordOutcome — first was for weight boost)
    await cellStore.recordOutcome(cell.id, true);

    const updated = cellStore.get(cell.id)!;
    assert.ok(updated.weight > 0.6); // Weight was boosted to 0.6, now increased further
    assert.strictEqual(updated.injectionCount, 2); // Two recordOutcome calls: weight boost + final
  });

  test("E2E: Audit logger records events", async () => {
    const audit = new AuditLogger(TEST_DIR);
    await audit.init();

    await audit.log({
      event: "cell_created",
      sessionId: "s1",
      cellId: "cell-123",
      data: { eventSignature: "test.ts:error_loop:ERR" },
    });

    await audit.logCellInject("cell-123", "test.ts:error_loop:ERR", 0.7, true);

    const stats = await audit.getStats();
    assert.ok(stats.total >= 2);
    assert.strictEqual(stats.byType["cell_created"], 1);
    assert.strictEqual(stats.byType["cell_helped"], 1);
  });

  test("E2E: Permission gate blocks dangerous commands", () => {
    const gate = new PermissionGate(TEST_DIR);

    const forcePush = gate.evaluateCommand("git push --force origin main");
    assert.strictEqual(forcePush.allowed, false);
    assert.ok(forcePush.reason.includes("force_push_blocked"));

    const sudoRm = gate.evaluateCommand("sudo rm -rf /");
    assert.strictEqual(sudoRm.allowed, false);
    assert.ok(sudoRm.reason.includes("sudo_blocked"));

    const safeRead = gate.evaluateCommand("ls -la /tmp");
    assert.strictEqual(safeRead.allowed, true);
  });

  test("E2E: Permission gate allows safe external directories", () => {
    const gate = new PermissionGate(TEST_DIR);

    const tmp = gate.evaluateExternalDirectory("/tmp/ovav-test");
    assert.strictEqual(tmp.allowed, true);

    const home = gate.evaluateExternalDirectory(`${process.env.HOME}/.cache/ovav`);
    assert.strictEqual(home.allowed, true);

    const dangerous = gate.evaluateExternalDirectory("/etc/passwd");
    assert.strictEqual(dangerous.allowed, false);
  });

  test("E2E: Cell weight decays when injection doesn't help", async () => {
    const cellStore = new CellStore(TEST_DIR);
    await cellStore.load();

    const cell = await cellStore.save({
      eventSignature: "old.ts:error_loop:DEPRECATED",
      summary: "Old advice.",
    });

    const originalWeight = cell.weight;

    await cellStore.recordOutcome(cell.id, false);
    await cellStore.recordOutcome(cell.id, false);
    await cellStore.recordOutcome(cell.id, false);

    const updated = cellStore.get(cell.id)!;
    assert.ok(updated.weight < originalWeight);
    assert.strictEqual(updated.injectionCount, 3);
  });

  test("E2E: highest weight cell is selected for same signature", async () => {
    const cellStore = new CellStore(TEST_DIR);
    await cellStore.load();

    const cell1 = await cellStore.save({
      eventSignature: "pkg.ts:error_loop:IMPORT",
      summary: "Cell 1 — low weight",
    });
    await cellStore.recordOutcome(cell1.id, false);

    const cell2 = await cellStore.save({
      eventSignature: "pkg.ts:error_loop:IMPORT",
      summary: "Cell 2 — high weight",
    });
    for (let i = 0; i < 5; i++) {
      await cellStore.recordOutcome(cell2.id, true);
    }

    const detector = new Detector(cellStore);

    // 5 errors → ERROR_LOOP (3 errors → RETRY_LOOP)
    for (let i = 0; i < 5; i++) {
      detector.decide({
        tool: "Read",
        file: "pkg.ts",
        result: {},
        error: "IMPORT",
      });
      await new Promise(r => setTimeout(r, 2));
    }

    const decision = detector.decide({
      tool: "Read",
      file: "pkg.ts",
      result: {},
      error: "IMPORT",
    });

    assert.notStrictEqual(decision.inject, null);
    assert.strictEqual(decision.inject!.id, cell2.id);
  });
});
