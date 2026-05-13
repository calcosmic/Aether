/**
 * Unit tests for the wave orchestrator.
 *
 * Tests verify:
 * - Parallel dispatch within a wave (Promise.all)
 * - Sequential dispatch when parallel=false
 * - Retry logic with exponential backoff
 * - Retry limit enforcement
 * - Wave grouping and sequential wave execution
 * - Failure tracking in wave results
 *
 * Uses __setDispatchSingleWorker to inject a mock.
 */

import { describe, it } from "node:test";
import assert from "node:assert/strict";

import type { BuildDispatch } from "../src/types.js";
import type { DispatchResult } from "../src/worker-dispatch.js";
import {
  dispatchWave,
  retryDispatch,
  dispatchWaves,
  __setDispatchSingleWorker,
  __restoreDispatchSingleWorker,
  type WaveOrchestratorOptions,
} from "../src/wave-orchestrator.js";

// ---------------------------------------------------------------------------
// Mock helpers
// ---------------------------------------------------------------------------

let mockResults: Map<string, DispatchResult> = new Map();
let mockCallCount: Map<string, number> = new Map();

async function mockDispatchSingleWorker(
  _opts: WaveOrchestratorOptions,
  dispatch: BuildDispatch
): Promise<DispatchResult> {
  const count = mockCallCount.get(dispatch.name) ?? 0;
  mockCallCount.set(dispatch.name, count + 1);

  const key = `${dispatch.name}:${count}`;
  if (mockResults.has(key)) {
    return mockResults.get(key)!;
  }

  // Default: succeed
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

const defaultOpts: WaveOrchestratorOptions = {
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

describe("wave-orchestrator", () => {
  it("dispatchWave runs workers in parallel when parallel=true", async () => {
    resetMocks();
    const dispatches = [
      makeDispatch("W1-A", 1),
      makeDispatch("W1-B", 1),
      makeDispatch("W1-C", 1),
    ];

    // Mock each worker to take 50ms
    const delayedMock = async (
      opts: WaveOrchestratorOptions,
      dispatch: BuildDispatch
    ): Promise<DispatchResult> => {
      await new Promise<void>((resolve) => setTimeout(resolve, 50));
      return mockDispatchSingleWorker(opts, dispatch);
    };
    __setDispatchSingleWorker(delayedMock);

    const start = Date.now();
    const result = await dispatchWave({ ...defaultOpts, parallel: true }, dispatches);
    const elapsed = Date.now() - start;

    __restoreDispatchSingleWorker();

    assert.equal(result.wave, 1);
    assert.equal(result.results.length, 3);
    assert.equal(result.failures.length, 0);
    // Parallel: all 3 should finish in ~50-120ms, not ~150ms+
    assert.ok(elapsed < 150, `Expected parallel dispatch < 150ms, got ${elapsed}ms`);
  });

  it("dispatchWave runs workers sequentially when parallel=false", async () => {
    resetMocks();
    const dispatches = [
      makeDispatch("W1-A", 1),
      makeDispatch("W1-B", 1),
      makeDispatch("W1-C", 1),
    ];

    const delayedMock = async (
      opts: WaveOrchestratorOptions,
      dispatch: BuildDispatch
    ): Promise<DispatchResult> => {
      await new Promise<void>((resolve) => setTimeout(resolve, 50));
      return mockDispatchSingleWorker(opts, dispatch);
    };
    __setDispatchSingleWorker(delayedMock);

    const start = Date.now();
    const result = await dispatchWave({ ...defaultOpts, parallel: false }, dispatches);
    const elapsed = Date.now() - start;

    __restoreDispatchSingleWorker();

    assert.equal(result.wave, 1);
    assert.equal(result.results.length, 3);
    assert.equal(result.failures.length, 0);
    // Sequential: 3 workers * 50ms = ~150ms minimum
    assert.ok(elapsed >= 150, `Expected sequential dispatch >= 150ms, got ${elapsed}ms`);
  });

  it("retryDispatch retries failed workers", async () => {
    resetMocks();
    __setDispatchSingleWorker(mockDispatchSingleWorker);
    const dispatch = makeDispatch("Retry-1", 1);

    // First call fails, second succeeds
    mockResults.set("Retry-1:0", {
      name: "Retry-1",
      status: "failed",
      summary: "First attempt failed",
    });
    mockResults.set("Retry-1:1", {
      name: "Retry-1",
      status: "completed",
      summary: "Second attempt succeeded",
    });

    const result = await retryDispatch(
      { ...defaultOpts, retryLimit: 2, retryDelayMs: 10 },
      dispatch
    );

    __restoreDispatchSingleWorker();

    assert.equal(result.status, "completed");
    assert.equal(result.summary, "Second attempt succeeded");
    assert.equal(mockCallCount.get("Retry-1"), 2);
  });

  it("retryDispatch respects retry limit", async () => {
    resetMocks();
    __setDispatchSingleWorker(mockDispatchSingleWorker);
    const dispatch = makeDispatch("Retry-Fail", 1);

    // All calls fail
    mockResults.set("Retry-Fail:0", {
      name: "Retry-Fail",
      status: "failed",
      summary: "Attempt 1 failed",
    });
    mockResults.set("Retry-Fail:1", {
      name: "Retry-Fail",
      status: "failed",
      summary: "Attempt 2 failed",
    });

    const result = await retryDispatch(
      { ...defaultOpts, retryLimit: 2, retryDelayMs: 10 },
      dispatch
    );

    __restoreDispatchSingleWorker();

    assert.equal(result.status, "failed");
    // retryLimit=2 means 2 attempts total (initial + 1 retry)
    assert.equal(mockCallCount.get("Retry-Fail"), 2);
  });

  it("dispatchWaves groups by wave and runs waves sequentially", async () => {
    resetMocks();
    __setDispatchSingleWorker(mockDispatchSingleWorker);
    const dispatches = [
      makeDispatch("W2-A", 2),
      makeDispatch("W1-A", 1),
      makeDispatch("W2-B", 2),
      makeDispatch("W1-B", 1),
    ];

    for (const d of dispatches) {
      mockResults.set(`${d.name}:0`, {
        name: d.name,
        status: "completed",
        summary: `Done ${d.name}`,
      });
    }

    const results = await dispatchWaves(defaultOpts, dispatches);

    __restoreDispatchSingleWorker();

    assert.equal(results.length, 2);
    // Waves should be sorted ascending
    assert.equal(results[0]!.wave, 1);
    assert.equal(results[0]!.results.length, 2);
    assert.equal(results[1]!.wave, 2);
    assert.equal(results[1]!.results.length, 2);
  });

  it("dispatchWave returns failures array", async () => {
    resetMocks();
    __setDispatchSingleWorker(mockDispatchSingleWorker);
    const dispatches = [
      makeDispatch("OK-1", 1),
      makeDispatch("FAIL-1", 1),
      makeDispatch("OK-2", 1),
    ];

    mockResults.set("OK-1:0", { name: "OK-1", status: "completed", summary: "OK" });
    mockResults.set("FAIL-1:0", { name: "FAIL-1", status: "failed", summary: "Failed" });
    mockResults.set("FAIL-1:1", { name: "FAIL-1", status: "failed", summary: "Failed again" });
    mockResults.set("OK-2:0", { name: "OK-2", status: "completed", summary: "OK" });

    const result = await dispatchWave(defaultOpts, dispatches);

    __restoreDispatchSingleWorker();

    assert.equal(result.results.length, 3);
    assert.equal(result.failures.length, 1);
    assert.equal(result.failures[0]!.name, "FAIL-1");
  });
});
