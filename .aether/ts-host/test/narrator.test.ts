/**
 * Narrator unit tests.
 *
 * Verifies event dispatch, renderer selection, and output mode behavior.
 */

import { describe, it, beforeEach, afterEach } from "node:test";
import assert from "node:assert/strict";

import { createNarrator } from "../src/narrator.js";
import type { CeremonyEvent } from "../src/types.js";

const REPO_ROOT = "/Users/callumcowie/repos/Aether";

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

function makeEvent(topic: string, payload: Record<string, unknown>): CeremonyEvent {
  return {
    id: "test-1",
    topic,
    payload,
    source: "test",
    timestamp: new Date().toISOString(),
    ttl_days: 1,
    expires_at: new Date().toISOString(),
  };
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("narrator", () => {
  let originalWrite: typeof process.stdout.write;
  let writes: string[];

  beforeEach(() => {
    originalWrite = process.stdout.write.bind(process.stdout);
    writes = [];
    process.stdout.write = ((chunk: string | Uint8Array, ...args: unknown[]): boolean => {
      writes.push(typeof chunk === "string" ? chunk : Buffer.from(chunk).toString("utf-8"));
      return true;
    }) as typeof process.stdout.write;
  });

  afterEach(() => {
    process.stdout.write = originalWrite;
  });

  it("createNarrator returns an object with onEvent and stop", () => {
    const narrator = createNarrator({ cwd: REPO_ROOT });
    assert.equal(typeof narrator.onEvent, "function");
    assert.equal(typeof narrator.stop, "function");
    narrator.stop();
  });

  it("onEvent writes to stdout for known topic", () => {
    const narrator = createNarrator({ cwd: REPO_ROOT });
    const event = makeEvent("ceremony.build.spawn", {
      caste: "builder",
      name: "Mason-67",
      task: "Task 1",
    });
    narrator.onEvent(event);
    const output = writes.join("");
    assert.ok(output.includes("🔨"), "Should include builder emoji");
    assert.ok(output.includes("Builder"), "Should include builder label");
    narrator.stop();
  });

  it("onEvent ignores unknown topics", () => {
    const narrator = createNarrator({ cwd: REPO_ROOT });
    const event = makeEvent("ceremony.unknown.topic", { message: "hello" });
    narrator.onEvent(event);
    assert.equal(writes.length, 0, "Should not write for unknown topics");
    narrator.stop();
  });

  it("json mode does not write to stdout", () => {
    const narrator = createNarrator({ cwd: REPO_ROOT, outputMode: "json" });
    const event = makeEvent("ceremony.build.spawn", {
      caste: "builder",
      name: "Mason-67",
      task: "Task 1",
    });
    narrator.onEvent(event);
    assert.equal(writes.length, 0, "Should not write in json mode");
    narrator.stop();
  });

  it("markdown mode strips ANSI", () => {
    const narrator = createNarrator({ cwd: REPO_ROOT, outputMode: "markdown" });
    const event = makeEvent("ceremony.build.spawn", {
      caste: "builder",
      name: "Mason-67",
      task: "Task 1",
    });
    narrator.onEvent(event);
    const output = writes.join("");
    assert.ok(!output.includes("\x1b["), "Should not contain ANSI codes");
    assert.ok(output.includes("🔨"), "Should preserve emoji");
    narrator.stop();
  });

  it("narrator suppresses stdout when suppressOutput is true", () => {
    const narrator = createNarrator({ cwd: REPO_ROOT, suppressOutput: true });
    const event = makeEvent("ceremony.build.spawn", {
      caste: "builder",
      name: "Mason-67",
      task: "Task 1",
    });
    narrator.onEvent(event);
    assert.equal(writes.length, 0, "Should not write to stdout when suppressOutput is true");
    narrator.stop();
  });

  it("narrator writes stdout when suppressOutput is false", () => {
    const narrator = createNarrator({ cwd: REPO_ROOT, suppressOutput: false });
    const event = makeEvent("ceremony.build.spawn", {
      caste: "builder",
      name: "Mason-67",
      task: "Task 1",
    });
    narrator.onEvent(event);
    assert.ok(writes.length > 0, "Should write to stdout when suppressOutput is false");
    narrator.stop();
  });
});
