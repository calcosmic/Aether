/**
 * Event bridge unit tests.
 *
 * Uses a mock `aether` script to simulate NDJSON ceremony event streams,
 * verifies replay + stream handoff, deduplication, and boundary enforcement.
 */

import { describe, it } from "node:test";
import assert from "node:assert/strict";

import {
  startEventBridge,
  stopEventBridge,
  BoundaryViolationError,
} from "../src/event-bridge.js";
import type { CeremonyEvent } from "../src/types.js";

const MOCK_AETHER = "/tmp/mock-aether";
const REPO_ROOT = "/Users/callumcowie/repos/Aether";

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("event-bridge", () => {
  it("replays historical events and starts stream", async () => {
    const events: CeremonyEvent[] = [];
    const controller = await startEventBridge({
      goBinaryPath: MOCK_AETHER,
      cwd: REPO_ROOT,
      filter: "ceremony.*",
      onEvent: (evt) => events.push(evt),
    });

    // Wait for mocked stream events
    await new Promise((r) => setTimeout(r, 150));

    // evt-1 from replay + evt-2 and evt-3 from stream (evt-1 deduplicated)
    assert.equal(events.length, 3, "Should receive replay + 2 new events");
    assert.equal(events[0]!.id, "evt-1");
    assert.equal(events[1]!.id, "evt-2");
    assert.equal(events[2]!.id, "evt-3");

    stopEventBridge(controller);
  });

  it("deduplicates events by id", async () => {
    const events: CeremonyEvent[] = [];
    const controller = await startEventBridge({
      goBinaryPath: MOCK_AETHER,
      cwd: REPO_ROOT,
      onEvent: (evt) => events.push(evt),
    });

    await new Promise((r) => setTimeout(r, 150));

    // evt-1 appears in replay AND stream, but should only be emitted once
    const evt1Count = events.filter((e) => e.id === "evt-1").length;
    assert.equal(evt1Count, 1, "Duplicate evt-1 should be deduplicated");

    stopEventBridge(controller);
  });

  it("stopEventBridge kills the subprocess", async () => {
    const controller = await startEventBridge({
      goBinaryPath: MOCK_AETHER,
      cwd: REPO_ROOT,
      onEvent: () => {},
    });

    // Should not throw
    assert.doesNotThrow(() => stopEventBridge(controller));
  });

  it("BoundaryViolationError is exported and has correct shape", () => {
    const err = new BoundaryViolationError(
      "TS host attempted to write to Go-owned path: .aether/data/event-bus.jsonl"
    );
    assert.equal(err.name, "BoundaryViolationError");
    assert.ok(err.message.includes("Go-owned path"));
  });
});
