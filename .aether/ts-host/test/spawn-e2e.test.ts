/**
 * End-to-end spawn integration tests.
 *
 * Proves the full spawn pipeline works: a manifest worker requests spawns,
 * the orchestrator validates them, child workers dispatch and complete,
 * results attach to parent handoffs, and the spawn tree records correct
 * parent references.
 *
 * Covers: SPAWN-01, SPAWN-02, SPAWN-03, SPAWN-04, SPAWN-05, SPAWN-06.
 */

import { describe, it, beforeEach, afterEach } from "node:test";
import assert from "node:assert/strict";

import type { BuildDispatch, SpawnClaim } from "../src/types.js";
import type { DispatchResult } from "../src/worker-dispatch.js";
import {
  dispatchWaves,
  __setDispatchSingleWorker,
  __restoreDispatchSingleWorker,
  type WaveOrchestratorOptions,
} from "../src/wave-orchestrator.js";
import {
  createSpawnOrchestrator,
  type SpawnOrchestrator,
} from "../src/spawn-orchestrator.js";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Create a standard manifest dispatch entry. */
function makeDispatch(name: string, wave: number, caste = "builder"): BuildDispatch {
  return {
    stage: "implement",
    wave,
    caste,
    name,
    task: `Task for ${name}`,
    status: "pending",
  };
}

/** Default wave orchestrator options for E2E tests. */
const defaultOpts: WaveOrchestratorOptions = {
  goBinaryPath: "/usr/bin/true",
  cwd: "/tmp",
  simulateWorkers: true,
  parallel: false, // Sequential for deterministic test ordering
  retryLimit: 1,
  retryDelayMs: 10,
};

/** Capture stderr output during an async callback, restoring on completion. */
async function captureStderrAsync<T>(fn: () => Promise<T>): Promise<{ result: T; stderr: string }> {
  const chunks: string[] = [];
  const originalWrite = process.stderr.write.bind(process.stderr);
  process.stderr.write = ((chunk: unknown, ...args: unknown[]) => {
    if (typeof chunk === "string") {
      chunks.push(chunk);
    }
    return originalWrite(chunk, ...args as [string, ...unknown[]]);
  }) as typeof process.stderr.write;

  try {
    const result = await fn();
    return { result, stderr: chunks.join("") };
  } finally {
    process.stderr.write = originalWrite;
  }
}

// ---------------------------------------------------------------------------
// E2E: Full spawn pipeline (SPAWN-01)
// ---------------------------------------------------------------------------

describe("end-to-end spawn pipeline (SPAWN-01)", () => {
  afterEach(() => {
    __restoreDispatchSingleWorker();
  });

  it("manifest worker spawns child, child completes, result in parent handoff", async () => {
    const spawnClaim: SpawnClaim = {
      caste: "scout",
      task: "Research dependency patterns",
      reason: "Need deeper analysis",
    };

    // Mock: parent returns a spawn claim, child completes normally
    __setDispatchSingleWorker(
      async (_opts, dispatch): Promise<DispatchResult> => {
        if (dispatch.name === "Builder-01") {
          return {
            name: "Builder-01",
            status: "completed",
            summary: "Built feature",
            spawns: [spawnClaim],
          };
        }
        // Child worker (dispatched in spawn wave)
        return {
          name: dispatch.name,
          status: "completed",
          summary: `Researched dependencies successfully`,
          duration: 2.5,
        };
      }
    );

    const orchestrator = createSpawnOrchestrator({
      totalBudget: 10,
      consumedBudget: 1, // Builder-01 already consumed
    });

    const { result: waveResults, stderr } = await captureStderrAsync(() =>
      dispatchWaves(
        { ...defaultOpts, spawnOrchestrator: orchestrator },
        [makeDispatch("Builder-01", 1)]
      )
    );

    // Two wave results: manifest wave + spawn wave
    assert.equal(waveResults.length, 2, "Should have manifest wave + spawn wave");

    // Manifest wave has the parent worker
    const manifestWave = waveResults[0]!;
    assert.equal(manifestWave.results.length, 1);
    assert.equal(manifestWave.results[0]!.name, "Builder-01");
    assert.equal(manifestWave.results[0]!.status, "completed");

    // Spawn wave has the child worker
    const spawnWave = waveResults[1]!;
    assert.equal(spawnWave.results.length, 1, "Spawn wave should have 1 child worker");
    assert.equal(
      spawnWave.results[0]!.name,
      "Builder-01-spawn-0",
      "Child name should follow parent-spawn-index pattern"
    );
    assert.equal(spawnWave.results[0]!.status, "completed");

    // Parent handoff contains child results (SPAWN-04)
    const parent = manifestWave.results[0]!;
    assert.ok(parent.handoff, "Parent should have a handoff object");
    assert.ok(
      Array.isArray(parent.handoff!.child_results),
      "Parent handoff should have child_results array"
    );
    assert.equal(parent.handoff!.child_results!.length, 1, "Should have exactly 1 child result");
    assert.equal(parent.handoff!.child_results![0]!.name, "Builder-01-spawn-0");
    assert.equal(parent.handoff!.child_results![0]!.status, "completed");
    assert.equal(
      parent.handoff!.child_results![0]!.summary,
      "Researched dependencies successfully"
    );

    // Ceremony output confirms spawn wave was dispatched
    assert.ok(
      stderr.includes("Dispatching spawn wave"),
      `Stderr should mention spawn wave dispatch. Got: ${stderr.slice(0, 300)}`
    );
  });

  it("multiple manifest workers each spawn children", async () => {
    const claim: SpawnClaim = { caste: "scout", task: "Research", reason: "Need context" };

    __setDispatchSingleWorker(
      async (_opts, dispatch): Promise<DispatchResult> => {
        // Both manifest workers return spawn claims
        if (dispatch.name === "Builder-01" || dispatch.name === "Builder-02") {
          return {
            name: dispatch.name,
            status: "completed",
            summary: `Built with ${dispatch.name}`,
            spawns: [claim],
          };
        }
        return {
          name: dispatch.name,
          status: "completed",
          summary: `Child done: ${dispatch.name}`,
        };
      }
    );

    const orchestrator = createSpawnOrchestrator({
      totalBudget: 10,
      consumedBudget: 2, // Two manifest workers
    });

    const { result: waveResults } = await captureStderrAsync(() =>
      dispatchWaves(
        { ...defaultOpts, spawnOrchestrator: orchestrator },
        [makeDispatch("Builder-01", 1), makeDispatch("Builder-02", 1)]
      )
    );

    // Manifest wave + spawn wave
    assert.equal(waveResults.length, 2, "Should have manifest + spawn wave");

    // Both parents should have child results
    const manifestWave = waveResults[0]!;
    assert.equal(manifestWave.results.length, 2);

    for (const parent of manifestWave.results) {
      assert.ok(parent.handoff, `${parent.name} should have handoff`);
      assert.equal(
        parent.handoff!.child_results!.length,
        1,
        `${parent.name} should have 1 child result`
      );
    }
  });
});

// ---------------------------------------------------------------------------
// E2E: Budget enforcement (SPAWN-03, SPAWN-06)
// ---------------------------------------------------------------------------

describe("budget enforcement end-to-end (SPAWN-03, SPAWN-06)", () => {
  afterEach(() => {
    __restoreDispatchSingleWorker();
  });

  it("total workers (manifest + spawned) never exceed budget", async () => {
    // Budget=3, 2 manifest workers, each tries to spawn 2 children
    // Remaining = 3 - 2 = 1 slot. Only 1 child total can be accepted.
    const claims: SpawnClaim[] = [
      { caste: "scout", task: "Research A" },
      { caste: "scout", task: "Research B" },
    ];

    __setDispatchSingleWorker(
      async (_opts, dispatch): Promise<DispatchResult> => {
        if (dispatch.name === "Builder-01" || dispatch.name === "Builder-02") {
          return {
            name: dispatch.name,
            status: "completed",
            summary: "Done",
            spawns: claims,
          };
        }
        return { name: dispatch.name, status: "completed", summary: "Child done" };
      }
    );

    const orchestrator = createSpawnOrchestrator({
      totalBudget: 3,
      consumedBudget: 2, // 2 manifest workers consumed
    });

    const { result: waveResults, stderr } = await captureStderrAsync(() =>
      dispatchWaves(
        { ...defaultOpts, spawnOrchestrator: orchestrator },
        [makeDispatch("Builder-01", 1), makeDispatch("Builder-02", 1)]
      )
    );

    // Manifest wave + spawn wave (if any children accepted)
    assert.ok(waveResults.length >= 1, "Should have at least the manifest wave");

    // The spawn wave should have at most 1 child (budget only allows 1)
    if (waveResults.length > 1) {
      const spawnWave = waveResults[1]!;
      assert.ok(
        spawnWave.results.length <= 1,
        `Spawn wave should have at most 1 child (budget only has 1 slot), got ${spawnWave.results.length}`
      );
    }

    // Verify budget is fully consumed (no remaining slots)
    assert.equal(
      orchestrator.remainingBudget,
      0,
      "Budget should be fully consumed after accepting at most 1 spawn"
    );

    // Rejected spawns should be logged
    assert.ok(
      stderr.includes("Spawn rejected") || stderr.includes("spawn budget exhausted"),
      `Stderr should mention budget rejection. Got: ${stderr.slice(0, 300)}`
    );
  });

  it("zero remaining budget rejects all spawn claims", async () => {
    const claim: SpawnClaim = { caste: "scout", task: "Research" };

    __setDispatchSingleWorker(
      async (_opts, dispatch): Promise<DispatchResult> => {
        if (dispatch.name === "Builder-01") {
          return {
            name: dispatch.name,
            status: "completed",
            summary: "Done",
            spawns: [claim],
          };
        }
        return { name: dispatch.name, status: "completed", summary: "Done" };
      }
    );

    // Budget fully consumed
    const orchestrator = createSpawnOrchestrator({
      totalBudget: 1,
      consumedBudget: 1,
    });

    const { result: waveResults } = await captureStderrAsync(() =>
      dispatchWaves(
        { ...defaultOpts, spawnOrchestrator: orchestrator },
        [makeDispatch("Builder-01", 1)]
      )
    );

    // Only manifest wave (no spawn wave because budget was zero)
    assert.equal(
      waveResults.length,
      1,
      "Should have only manifest wave (no spawn wave with zero budget)"
    );
  });
});

// ---------------------------------------------------------------------------
// E2E: Depth enforcement (SPAWN-02)
// ---------------------------------------------------------------------------

describe("depth enforcement end-to-end (SPAWN-02)", () => {
  afterEach(() => {
    __restoreDispatchSingleWorker();
  });

  it("grandchild spawns are rejected", async () => {
    // Simulate a spawned child worker (depth=2) that tries to spawn another.
    // In the real pipeline, processWaveSpawns only processes manifest worker results,
    // so spawned children at depth 2 would never have their claims processed.
    // This test verifies at the orchestrator level that depth=2 claims are rejected.

    const orchestrator = createSpawnOrchestrator({
      totalBudget: 10,
      consumedBudget: 0,
    });

    // A child worker at depth 2 tries to spawn a grandchild
    const grandchildClaim: SpawnClaim = {
      caste: "builder",
      task: "Build sub-component",
      reason: "Need more workers",
    };

    const result = orchestrator.processClaims("Builder-01-spawn-0", 2, [grandchildClaim]);

    assert.equal(
      result.accepted.length,
      0,
      "Grandchild spawn should be rejected (depth 2 is max)"
    );
    assert.equal(result.rejected.length, 1, "Should have 1 rejection");
    assert.ok(
      result.rejected[0]!.reason.includes("depth"),
      `Rejection reason should mention depth: ${result.rejected[0]!.reason}`
    );
  });

  it("manifest workers at depth 1 can spawn children at depth 2", async () => {
    const orchestrator = createSpawnOrchestrator({
      totalBudget: 10,
      consumedBudget: 0,
    });

    const childClaim: SpawnClaim = {
      caste: "scout",
      task: "Research",
    };

    const result = orchestrator.processClaims("Builder-01", 1, [childClaim]);

    assert.equal(result.accepted.length, 1, "Manifest worker should be able to spawn");
    assert.equal(
      result.accepted[0]!.depth,
      2,
      "Child should be at depth 2"
    );
    assert.equal(
      result.accepted[0]!.parent,
      "Builder-01",
      "Child parent should be the manifest worker"
    );
  });
});

// ---------------------------------------------------------------------------
// E2E: Spawn tree parent references (SPAWN-05)
// ---------------------------------------------------------------------------

describe("spawn tree parent references (SPAWN-05)", () => {
  afterEach(() => {
    __restoreDispatchSingleWorker();
  });

  it("child spawn records correct parent in spawn tree", async () => {
    const claim: SpawnClaim = { caste: "scout", task: "Investigate" };

    // Track which dispatches were sent, including their parent/depth fields
    const dispatchedWorkers: { name: string; parent?: string; depth?: number }[] = [];

    __setDispatchSingleWorker(
      async (_opts, dispatch): Promise<DispatchResult> => {
        dispatchedWorkers.push({
          name: dispatch.name,
          parent: (dispatch as unknown as Record<string, unknown>).parent as string | undefined,
          depth: (dispatch as unknown as Record<string, unknown>).depth as number | undefined,
        });

        if (dispatch.name === "Builder-01") {
          return {
            name: "Builder-01",
            status: "completed",
            summary: "Built",
            spawns: [claim],
          };
        }
        return { name: dispatch.name, status: "completed", summary: "Child done" };
      }
    );

    const orchestrator = createSpawnOrchestrator({
      totalBudget: 10,
      consumedBudget: 1,
    });

    await dispatchWaves(
      { ...defaultOpts, spawnOrchestrator: orchestrator },
      [makeDispatch("Builder-01", 1)]
    );

    // Verify manifest worker dispatch
    const manifestDispatch = dispatchedWorkers.find((w) => w.name === "Builder-01");
    assert.ok(manifestDispatch, "Manifest worker should have been dispatched");

    // Verify child worker dispatch has parent reference
    const childDispatch = dispatchedWorkers.find((w) => w.name === "Builder-01-spawn-0");
    assert.ok(childDispatch, "Child worker should have been dispatched");
    assert.equal(
      childDispatch!.parent,
      "Builder-01",
      "Child should record manifest worker as parent"
    );
    assert.equal(
      childDispatch!.depth,
      2,
      "Child should be at depth 2"
    );
  });

  it("manifest worker has no parent field (Queen is implicit parent)", async () => {
    const dispatchedWorkers: { name: string; parent?: string; depth?: number }[] = [];

    __setDispatchSingleWorker(
      async (_opts, dispatch): Promise<DispatchResult> => {
        dispatchedWorkers.push({
          name: dispatch.name,
          parent: (dispatch as unknown as Record<string, unknown>).parent as string | undefined,
          depth: (dispatch as unknown as Record<string, unknown>).depth as number | undefined,
        });
        return { name: dispatch.name, status: "completed", summary: "Done" };
      }
    );

    await dispatchWaves(defaultOpts, [makeDispatch("Builder-01", 1)]);

    const manifestDispatch = dispatchedWorkers.find((w) => w.name === "Builder-01");
    assert.ok(manifestDispatch, "Manifest worker should have been dispatched");
    // Manifest workers have no parent/depth fields set in their dispatch
    assert.equal(
      manifestDispatch!.parent,
      undefined,
      "Manifest worker should have no explicit parent (Queen is implicit)"
    );
    assert.equal(
      manifestDispatch!.depth,
      undefined,
      "Manifest worker should have no explicit depth field in dispatch"
    );
  });
});

// ---------------------------------------------------------------------------
// E2E: Ceremony output
// ---------------------------------------------------------------------------

describe("ceremony output", () => {
  afterEach(() => {
    __restoreDispatchSingleWorker();
  });

  it("spawn wave appears in ceremony output", async () => {
    const claim: SpawnClaim = { caste: "scout", task: "Explore" };

    __setDispatchSingleWorker(
      async (_opts, dispatch): Promise<DispatchResult> => {
        if (dispatch.name === "Builder-01") {
          return {
            name: dispatch.name,
            status: "completed",
            summary: "Built",
            spawns: [claim],
          };
        }
        return { name: dispatch.name, status: "completed", summary: "Child done" };
      }
    );

    const orchestrator = createSpawnOrchestrator({
      totalBudget: 10,
      consumedBudget: 1,
    });

    const { stderr } = await captureStderrAsync(() =>
      dispatchWaves(
        { ...defaultOpts, spawnOrchestrator: orchestrator },
        [makeDispatch("Builder-01", 1)]
      )
    );

    assert.ok(
      stderr.includes("Dispatching spawn wave"),
      `Stderr should contain "Dispatching spawn wave". Got: ${stderr.slice(0, 400)}`
    );

    // Child count should be mentioned
    assert.ok(
      stderr.includes("1 child workers") || stderr.includes("child workers"),
      `Stderr should mention child worker count. Got: ${stderr.slice(0, 400)}`
    );
  });

  it("no spawn wave output when no spawn claims", async () => {
    __setDispatchSingleWorker(
      async (_opts, dispatch): Promise<DispatchResult> => ({
        name: dispatch.name,
        status: "completed",
        summary: "Done",
      })
    );

    const orchestrator = createSpawnOrchestrator({
      totalBudget: 10,
      consumedBudget: 1,
    });

    const { stderr } = await captureStderrAsync(() =>
      dispatchWaves(
        { ...defaultOpts, spawnOrchestrator: orchestrator },
        [makeDispatch("Builder-01", 1)]
      )
    );

    assert.ok(
      !stderr.includes("Dispatching spawn wave"),
      `Stderr should NOT contain "Dispatching spawn wave" when no claims. Got: ${stderr.slice(0, 300)}`
    );
  });
});
