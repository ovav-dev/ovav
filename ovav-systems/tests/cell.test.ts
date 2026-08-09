/**
 * OVAV MEMORY v3 — Cell Unit Tests
 * Using Node.js built-in test runner
 */

import { test, describe } from "node:test";
import assert from "node:assert";
import {
  PrivacyTag,
  isInjectable,
  createCellId,
  buildEventSignature,
  type Cell,
} from "../dist/ovav-memory/cell.js";

describe("Cell", () => {
  test("PrivacyTag exports all required privacy tags", () => {
    assert.strictEqual(PrivacyTag.PUBLIC, "public");
    assert.strictEqual(PrivacyTag.PROJECT, "project");
    assert.strictEqual(PrivacyTag.SENSITIVE, "sensitive");
    assert.strictEqual(PrivacyTag.SECRET, "secret");
  });

  test("isInjectable returns false for SECRET cells", () => {
    const cell: Cell = {
      id: "test-id",
      eventSignature: "test:error_loop:ECONF",
      summary: "Test summary",
      detailRef: "",
      weight: 0.8,
      tags: [],
      harnessScope: ["pi"],
      privacy: PrivacyTag.SECRET,
      createdAt: new Date().toISOString(),
      lastHelpedAt: new Date().toISOString(),
      injectionCount: 0,
      lastInjectedAt: new Date().toISOString(),
    };
    assert.strictEqual(isInjectable(cell), false);
  });

  test("isInjectable returns false for zero-weight cells", () => {
    const cell: Cell = {
      id: "test-id",
      eventSignature: "test:error_loop:ECONF",
      summary: "Test summary",
      detailRef: "",
      weight: 0.0,
      tags: [],
      harnessScope: ["pi"],
      privacy: PrivacyTag.PUBLIC,
      createdAt: new Date().toISOString(),
      lastHelpedAt: new Date().toISOString(),
      injectionCount: 0,
      lastInjectedAt: new Date().toISOString(),
    };
    assert.strictEqual(isInjectable(cell), false);
  });

  test("isInjectable returns true for PUBLIC cells with positive weight", () => {
    const cell: Cell = {
      id: "test-id",
      eventSignature: "test:error_loop:ECONF",
      summary: "Test summary",
      detailRef: "",
      weight: 0.6,
      tags: [],
      harnessScope: ["pi"],
      privacy: PrivacyTag.PUBLIC,
      createdAt: new Date().toISOString(),
      lastHelpedAt: new Date().toISOString(),
      injectionCount: 0,
      lastInjectedAt: new Date().toISOString(),
    };
    assert.strictEqual(isInjectable(cell), true);
  });

  test("createCellId returns a valid uuid v4", () => {
    const id = createCellId();
    assert.match(id, /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i);
  });

  test("createCellId returns unique ids", () => {
    const ids = new Set(Array.from({ length: 100 }, () => createCellId()));
    assert.strictEqual(ids.size, 100);
  });

  test("buildEventSignature formats correctly", () => {
    const sig = buildEventSignature("editor.ts", "error_loop", "ECONF");
    assert.strictEqual(sig, "editor.ts:error_loop:ECONF");
  });

  test("buildEventSignature handles empty detail", () => {
    const sig = buildEventSignature("readme.md", "stuck", "");
    assert.strictEqual(sig, "readme.md:stuck:");
  });
});
