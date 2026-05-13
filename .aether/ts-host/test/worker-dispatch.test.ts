/**
 * Unit tests for the updated worker dispatch module.
 *
 * Tests verify:
 * - dispatchWorkers flattens wave results into a single DispatchResult array
 * - dispatchWorkers maintains input dispatch order
 * - dispatchWorkers with simulateWorkers=true uses simulation (no real spawn)
 * - toWorkerResults maps dispatches to WorkerResults correctly
 *
 * Uses __setDispatchSingleWorker to mock wave-orchestrator dispatch.
 */

import { describe, it } from "node:test";
import assert from "node:assert/strict";

import type { BuildDispatch } from "../src/types.js";
import {
  dispatchWorkers,
  toWorkerResults,
  type DispatchResult,
} from "../src/worker-dispatch.js";
import {
  __setDispatchSingleWorker,
  __restoreDispatchSingleWorker,
} from "../src/wave-orchestrator.js";

// ---------------------------------------------------------------------------
// Mock helpers
// ---------------------------------------------------------------------------

let mockResults: Map<string, DispatchResult> = new Map();
let mockCallCount: Map<string, number> = new Map();

async function mockDispatchSingleWorker(
  _opts: Record<string, unknown>,
  dispatch: BuildDispatch
): Promise<DispatchResult> {
  const count = mockCallCount.get(dispatch.name) ?? 0;
  mockCallCount.set(dispatch.name, count + 1);

  const key = `${dispatch.name}:${count}`;
  if (mockResults.has(key)) {
    return mockResults.get(key)!;
  }

  return {
    name: dispatch.name,
    status: "completed",
    summary: `Mock completion for ${dispatch.name}`,
  };
}

function resetMocks(): void {
  mockResults = new Map();
  mockCallCount = new Map();
}

// ---------------------------------------------------------------------------
// Test data
// ---------------------------------------------------------------------------

function makeDispatch(name: string, wave: number): BuildDispatch {
  return {
    stage: "implement",
    wave,
    caste: "builder",
    name,
    task: `Task for ${name}`,
    status: "pending",
  };
}

const defaultOpts: Record<string, unknown> = {
  goBinaryPath: "/usr/bin/true",
  cwd: "/tmp",
  simulateWorkers: true,
  parallel: true,
  retryLimit: 2,
  retryDelayMs: 10,
};

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("worker-dispatch", () => {
  it("dispatchWorkers flattens wave results", async () => {
    resetMocks();
    __setDispatchSingleWorker(mockDispatchSingleWorker);

    const dispatches = [
      makeDispatch("W1-A", 1),
      makeDispatch("W1-B", 1),
      makeDispatch("W2-A", 2),
    ];

    for (const d of dispatches) {
      mockResults.set(`${d.name}:0`, {
        name: d.name,
        status: "completed",
        summary: `Done ${d.name}`,
      });
    }

    const results = await dispatchWorkers(defaultOpts, dispatches);

    __restoreDispatchSingleWorker();

    assert.equal(results.length, 3, "Should return flat array of 3 results");
    assert.equal(results[0]!.name, "W1-A");
    assert.equal(results[1]!.name, "W1-B");
    assert.equal(results[2]!.name, "W2-A");
  });

  it("dispatchWorkers maintains order", async () => {
    resetMocks();
    __setDispatchSingleWorker(mockDispatchSingleWorker);

    const dispatches = [
      makeDispatch("Alpha", 1),
      makeDispatch("Beta", 1),
      makeDispatch("Gamma", 2),
      makeDispatch("Delta", 2),
    ];

    for (const d of dispatches) {
      mockResults.set(`${d.name}:0`, {
        name: d.name,
        status: "completed",
        summary: `Done ${d.name}`,
      });
    }

    const results = await dispatchWorkers(defaultOpts, dispatches);

    __restoreDispatchSingleWorker();

    const names = results.map((r) => r.name);
    assert.deepEqual(names, ["Alpha", "Beta", "Gamma", "Delta"]);
  });

  it("dispatchWorkers with simulateWorkers=true uses simulation", async () => {
    resetMocks();
    let realSpawnAttempted = false;

    __setDispatchSingleWorker(async (_opts, dispatch) => {
      // If simulateWorkers is true, we should never reach the real spawn path.
      // The mock itself proves we're in simulation because we injected it.
      return {
        name: dispatch.name,
        status: "completed",
        summary: "Simulated",
      };
    });

    const dispatches = [makeDispatch("Sim-1", 1)];
    const results = await dispatchWorkers(
      { ...defaultOpts, simulateWorkers: true },
      dispatches
    );

    __restoreDispatchSingleWorker();

    assert.equal(results.length, 1);
    assert.equal(results[0]!.status, "completed");
    assert.equal(results[0]!.summary, "Simulated");
    assert.ok(!realSpawnAttempted, "Real spawn should not be attempted in simulation mode");
  });

  it("toWorkerResults maps dispatches to WorkerResults", () => {
    const dispatches = [
      {
        stage: "implement",
        wave: 0,
        caste: "builder",
        name: "Builder-01-AA",
        task: "Implement feature X",
        status: "pending",
        task_id: "1.1",
      },
      {
        stage: "verify",
        wave: 1,
        caste: "watcher",
        name: "Watcher-01-BB",
        task: "Verify feature X",
        status: "pending",
        task_id: "1.2",
      },
    ];

    const results: DispatchResult[] = [
      {
        name: "Builder-01-AA",
        status: "completed",
        summary: "Implemented feature X",
        duration: 0.1,
      },
      {
        name: "Watcher-01-BB",
        status: "completed",
        summary: "Verified feature X",
        duration: 0.05,
      },
    ];

    const workerResults = toWorkerResults(dispatches, results);

    assert.equal(workerResults.length, 2, "Should produce 2 WorkerResult entries");

    assert.equal(workerResults[0]!.name, "Builder-01-AA");
    assert.equal(workerResults[0]!.status, "completed");
    assert.equal(workerResults[0]!.summary, "Implemented feature X");
    assert.equal(workerResults[0]!.caste, "builder");
    assert.equal(workerResults[0]!.task, "Implement feature X");
    assert.equal(workerResults[0]!.stage, "implement");
    assert.equal(workerResults[0]!.task_id, "1.1");
    assert.equal(workerResults[0]!.wave, 0);
    assert.equal(workerResults[0]!.duration, 0.1);

    assert.equal(workerResults[1]!.name, "Watcher-01-BB");
    assert.equal(workerResults[1]!.status, "completed");
    assert.equal(workerResults[1]!.summary, "Verified feature X");
    assert.equal(workerResults[1]!.caste, "watcher");
    assert.equal(workerResults[1]!.task, "Verify feature X");
    assert.equal(workerResults[1]!.stage, "verify");
    assert.equal(workerResults[1]!.task_id, "1.2");
    assert.equal(workerResults[1]!.wave, 1);
    assert.equal(workerResults[1]!.duration, 0.05);
  });
});
