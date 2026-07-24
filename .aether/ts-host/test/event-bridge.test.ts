/**
 * Event bridge unit tests.
 *
 * Uses a mock `aether` script to simulate NDJSON ceremony event streams,
 * verifies replay + stream handoff, deduplication, and boundary enforcement.
 */

import { after, before, describe, it } from "node:test";
import assert from "node:assert/strict";
import { chmodSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

import {
  startEventBridge,
  stopEventBridge,
  BoundaryViolationError,
} from "../src/event-bridge.js";
import type { CeremonyEvent } from "../src/types.js";

let mockDir = "";
let MOCK_AETHER = "";
const REPO_ROOT = "/Users/callumcowie/repos/Aether";

before(() => {
  mockDir = mkdtempSync(join(tmpdir(), "aether-event-bridge-"));
  MOCK_AETHER = join(mockDir, "mock-aether");
  writeFileSync(
    MOCK_AETHER,
    `#!/usr/bin/env node
function event(id) {
  return {
    id,
    topic: "ceremony.build.spawn",
    payload: {},
    source: "test",
    timestamp: "2026-05-18T00:00:00Z",
    ttl_days: 1,
    expires_at: "2026-05-19T00:00:00Z"
  };
}

const command = process.argv[2];
if (command === "event-bus-replay") {
  console.log(JSON.stringify({ ok: true, result: { events: [event("evt-1")] } }));
  process.exit(0);
}

if (command === "event-bus-subscribe") {
  for (const id of ["evt-1", "evt-2", "evt-3"]) {
    console.log(JSON.stringify(event(id)));
  }
  setTimeout(() => {}, 1000);
}
`,
    "utf-8"
  );
  chmodSync(MOCK_AETHER, 0o755);
});

after(() => {
  if (mockDir) {
    rmSync(mockDir, { recursive: true, force: true });
  }
});

async function waitForEventCount(
  events: CeremonyEvent[],
  expectedCount: number,
  timeoutMs = 1000
): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  while (events.length < expectedCount && Date.now() < deadline) {
    await new Promise((resolve) => setTimeout(resolve, 10));
  }
}

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

    try {
      // evt-1 from replay + evt-2 and evt-3 from stream (evt-1 deduplicated)
      await waitForEventCount(events, 3);
      assert.equal(events.length, 3, "Should receive replay + 2 new events");
      assert.equal(events[0]!.id, "evt-1");
      assert.equal(events[1]!.id, "evt-2");
      assert.equal(events[2]!.id, "evt-3");
    } finally {
      await stopEventBridge(controller);
    }
  });

  it("deduplicates events by id", async () => {
    const events: CeremonyEvent[] = [];
    const controller = await startEventBridge({
      goBinaryPath: MOCK_AETHER,
      cwd: REPO_ROOT,
      onEvent: (evt) => events.push(evt),
    });

    try {
      await waitForEventCount(events, 3);
      // evt-1 appears in replay AND stream, but should only be emitted once
      const evt1Count = events.filter((e) => e.id === "evt-1").length;
      assert.equal(evt1Count, 1, "Duplicate evt-1 should be deduplicated");
    } finally {
      await stopEventBridge(controller);
    }
  });

  it("stopEventBridge kills the subprocess", async () => {
    const controller = await startEventBridge({
      goBinaryPath: MOCK_AETHER,
      cwd: REPO_ROOT,
      onEvent: () => {},
    });

    // Should not throw
    await assert.doesNotReject(stopEventBridge(controller));
  });

  it("BoundaryViolationError is exported and has correct shape", () => {
    const err = new BoundaryViolationError(
      "TS host attempted to write to Go-owned path: .aether/data/event-bus.jsonl"
    );
    assert.equal(err.name, "BoundaryViolationError");
    assert.ok(err.message.includes("Go-owned path"));
  });
});
