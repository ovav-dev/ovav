/**
 * OVAV MEMORY v3 — Privacy Classifier Unit Tests
 * Using Node.js built-in test runner
 */

import { test, describe } from "node:test";
import assert from "node:assert";
import { classifyContent, canInject, filterInjectable } from "../dist/ovav-memory/privacy.js";
import { PrivacyTag } from "../dist/ovav-memory/cell.js";
import type { Cell } from "../dist/ovav-memory/cell.js";

describe("Privacy Classifier", () => {
  test("classifyContent returns SECRET for api_key patterns", () => {
    assert.strictEqual(
      classifyContent("api_key = 'sk-1234567890abcdef'"),
      PrivacyTag.SECRET
    );
    assert.strictEqual(
      classifyContent("const API_KEY = 'ghp_xxxxxxxxxxxx'"),
      PrivacyTag.SECRET
    );
  });

  test("classifyContent returns SECRET for private_key patterns", () => {
    assert.strictEqual(
      classifyContent("-----BEGIN RSA PRIVATE KEY-----"),
      PrivacyTag.SECRET
    );
  });

  test("classifyContent returns SENSITIVE for SSN patterns", () => {
    assert.strictEqual(classifyContent("SSN: 123-45-6789"), PrivacyTag.SENSITIVE);
  });

  test("classifyContent returns SENSITIVE for credit card patterns", () => {
    assert.strictEqual(classifyContent("Card: 4111111111111111"), PrivacyTag.SENSITIVE);
  });

  test("classifyContent returns PROJECT for internal patterns", () => {
    assert.strictEqual(classifyContent("internal endpoint"), PrivacyTag.PROJECT);
  });

  test("classifyContent returns PUBLIC for normal content", () => {
    assert.strictEqual(classifyContent("This is a normal code file"), PrivacyTag.PUBLIC);
    assert.strictEqual(classifyContent("TODO: fix the error handler"), PrivacyTag.PUBLIC);
  });

  test("canInject blocks SECRET cells", () => {
    const cell = { privacy: PrivacyTag.SECRET } as Cell;
    const decision = canInject(cell, "pi");
    assert.strictEqual(decision.allowed, false);
    assert.strictEqual(decision.tag, PrivacyTag.SECRET);
  });

  test("canInject blocks SENSITIVE cells in pi scope", () => {
    const cell = { privacy: PrivacyTag.SENSITIVE } as Cell;
    const decision = canInject(cell, "pi");
    assert.strictEqual(decision.allowed, false);
  });

  test("canInject allows PUBLIC cells in any scope", () => {
    const cell = { privacy: PrivacyTag.PUBLIC } as Cell;
    const decision = canInject(cell, "pi");
    assert.strictEqual(decision.allowed, true);
  });

  test("canInject blocks PROJECT cells in non-project scope", () => {
    const cell = { privacy: PrivacyTag.PROJECT } as Cell;
    const decision = canInject(cell, "unknown");
    assert.strictEqual(decision.allowed, false);
  });

  test("filterInjectable returns only allowed cells", () => {
    const cells: Cell[] = [
      { id: "1", privacy: PrivacyTag.PUBLIC } as Cell,
      { id: "2", privacy: PrivacyTag.SECRET } as Cell,
      { id: "3", privacy: PrivacyTag.SENSITIVE } as Cell,
      { id: "4", privacy: PrivacyTag.PUBLIC } as Cell,
    ];

    const result = filterInjectable(cells, "pi");
    assert.strictEqual(result.length, 2);
    assert.deepStrictEqual(result.map(c => c.id), ["1", "4"]);
  });
});
