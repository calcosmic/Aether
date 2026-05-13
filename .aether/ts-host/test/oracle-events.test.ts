/**
 * Oracle ceremony event tests.
 *
 * Verifies narrator rendering for Oracle-specific ceremony topics.
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

describe("oracle ceremony events", () => {
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

  it("narrator renders phase transition for ceremony.oracle.phase_transition", () => {
    const narrator = createNarrator({ cwd: REPO_ROOT });
    const event = makeEvent("ceremony.oracle.phase_transition", {
      status: "survey → verify",
      message: "Oracle phase transition at iteration 3",
      phase_name: "survey",
    });
    narrator.onEvent(event);
    const output = writes.join("");
    assert.ok(output.includes("survey"), "Should include 'survey' in output");
    assert.ok(output.includes("verify"), "Should include 'verify' in output");
    narrator.stop();
  });

  it("narrator renders iteration frame for ceremony.oracle.iteration", () => {
    const narrator = createNarrator({ cwd: REPO_ROOT });
    const event = makeEvent("ceremony.oracle.iteration", {
      wave: 5,
      task: "What is the best approach?",
      status: "investigate",
      message: "Oracle iteration 5: What is the best approach?",
    });
    narrator.onEvent(event);
    const output = writes.join("");
    assert.ok(output.includes("Oracle"), "Should include 'Oracle' in output");
    assert.ok(output.includes("Researching"), "Should include 'Researching' in output");
    narrator.stop();
  });
});
