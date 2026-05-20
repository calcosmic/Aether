/**
 * End-to-end tests for the full confidence-driven iteration cycle.
 *
 * Proves the loop, evaluator, ceremony, and feedback injection work together
 * through the real host.ts build dispatch path with mocked Go bridge.
 *
 * Covers: ITER-01 (max iterations), ITER-02 (confidence from results),
 * ITER-03 (diminishing returns), ITER-04 (feedback injection),
 * ITER-05 (budget tracking), ITER-06 (ceremony output).
 */

import { describe, it, beforeEach, afterEach } from "node:test";
import assert from "node:assert/strict";

import {
  parseArgs,
  __setCallGoJSON,
  __restoreCallGoJSON,
  __setDispatchWorkers,
  __restoreDispatchWorkers,
  __setDetectAvailablePlatforms,
  __restoreDetectAvailablePlatforms,
  __restoreAllMocks,
  runDispatchedBuildCommand,
} from "../src/host.js";

import {
  __setCallGoJSON as __setGoBridgeCallGoJSON,
  __restoreCallGoJSON as __restoreGoBridgeCallGoJSON,
} from "../src/go-bridge.js";

import {
  __setCreateCeremonyAdapter,
  __restoreCreateCeremonyAdapter,
} from "../src/ceremony-adapter.js";

import type { CeremonyAdapter, CeremonyWorkflow } from "../src/ceremony-adapter.js";
import type { GoBridgeOptions } from "../src/go-bridge.js";
import type { BuildDispatch } from "../src/types.js";

// ---------------------------------------------------------------------------
// Shared mock helpers
// ---------------------------------------------------------------------------

function createMockCeremonyAdapter(): CeremonyAdapter {
  return {
    renderSpawnPlan: (_workflow: CeremonyWorkflow, _manifest: unknown) => "",
    renderWaveStart: (_workflow: CeremonyWorkflow, _manifest: unknown, _wave: number) => "",
    renderWorkerComplete: (_workflow: CeremonyWorkflow, _worker: unknown) => "",
    renderCloseout: (_workflow: CeremonyWorkflow, _completionPath: string) => "",
  };
}

/** Tracks state shared across a test (go calls, dispatches, stderr). */
interface TestHarness {
  goCalls: string[][];
  dispatchCallCount: number;
  capturedAllDispatches: unknown[][];
  stderrOutput: string;
}

function createTestHarness(): TestHarness {
  return {
    goCalls: [],
    dispatchCallCount: 0,
    capturedAllDispatches: [],
    stderrOutput: "",
  };
}

/**
 * Build a mock callGoJSON for the iteration E2E path.
 * Returns build manifests with configurable per-iteration worker results.
 */
function createIterationMockGo(
  harness: TestHarness,
  opts?: {
    dispatchCount?: number;
    spawnBudget?: number;
    workerResults?: (callNum: number) => Array<Record<string, unknown>>;
  },
) {
  const defaultWorkerResults = () => [
    {
      name: "Builder-01",
      status: "completed",
      summary: "Done",
      duration: 5,
      files_created: ["src/module.ts"],
      files_modified: ["src/helper.ts"],
      tests_written: ["test/module.test.ts"],
    },
  ];
  const workerResultsFn = opts?.workerResults ?? defaultWorkerResults;
  const dispatchCount = opts?.dispatchCount ?? 1;
  const spawnBudget = opts?.spawnBudget;

  return <T>(_bridgeOpts: unknown, args: string[]): T => {
    harness.goCalls.push(args);
    const cmd = args[0];
    if (cmd === "hive-read") {
      return { entries: null, total: 0 } as unknown as T;
    }
    if (cmd === "registry-list") {
      return {
        colonies: [
          {
            repo_path: process.cwd(),
            domains: ["ts"],
            active: true,
            registered_at: "2026-05-18T00:00:00Z",
          },
        ],
      } as unknown as T;
    }
    if (cmd === "build") {
      const dispatches: Record<string, unknown>[] = [];
      for (let i = 0; i < dispatchCount; i++) {
        dispatches.push({
          name: `Builder-${String(i + 1).padStart(2, "0")}`,
          caste: "builder",
          task: "Build",
          wave: 1,
          execution_wave: 1,
          skill_section: "",
        });
      }
      const manifest: Record<string, unknown> = {
        dispatch_manifest: { dispatches },
      };
      if (spawnBudget !== undefined) {
        (manifest.dispatch_manifest as Record<string, unknown>).queen_execution_policy = {
          spawn_budget: { max_workers: spawnBudget },
        };
      }
      return manifest as unknown as T;
    }
    if (cmd === "build-finalize") {
      return { ok: true } as unknown as T;
    }
    return { ok: true } as unknown as T;
  };
}

/** Count how many times a Go command was called. */
function countGoCommandCalls(calls: string[][], command: string): number {
  return calls.filter((args) => args[0] === command).length;
}

/** Set up mocks common to every test. Returns cleanup function. */
function setupMocks(harness: TestHarness): () => void {
  __restoreAllMocks();
  __restoreGoBridgeCallGoJSON();
  __restoreCreateCeremonyAdapter();

  __setCreateCeremonyAdapter(() => createMockCeremonyAdapter());

  const originalStderrWrite = process.stderr.write.bind(process.stderr);
  process.stderr.write = ((chunk: unknown, ...args: unknown[]) => {
    if (typeof chunk === "string") harness.stderrOutput += chunk;
    return originalStderrWrite(chunk as string | Uint8Array, ...args as [BufferEncoding]);
  }) as typeof process.stderr.write;

  __setDetectAvailablePlatforms(async () => [
    "claude" as const,
  ]);

  return () => {
    __restoreAllMocks();
    __restoreGoBridgeCallGoJSON();
    __restoreCreateCeremonyAdapter();
  };
}

// ---------------------------------------------------------------------------
// 1. E2E: full iteration lifecycle
// ---------------------------------------------------------------------------

describe("E2E: full iteration lifecycle", () => {
  let harness: TestHarness;
  let cleanup: () => void;

  beforeEach(() => {
    harness = createTestHarness();
    cleanup = setupMocks(harness);
  });

  afterEach(() => {
    cleanup();
  });

  it("low confidence triggers second iteration which succeeds", async () => {
    const mockGo = createIterationMockGo(harness, {
      workerResults: (callNum) => {
        if (callNum === 0) {
          // Iteration 1: partial tests, blockers -> low confidence
          return [
            {
              name: "Builder-01",
              status: "failed",
              summary: "Build failed",
              duration: 5,
              blockers: ["missing import"],
            },
          ];
        }
        // Iteration 2: all pass -> high confidence
        return [
          {
            name: "Builder-01",
            status: "completed",
            summary: "Done",
            duration: 5,
            files_created: ["src/module.ts"],
            files_modified: ["src/helper.ts"],
            test_results: { passed: 8, total: 8 },
          },
        ];
      },
    });

    __setCallGoJSON(mockGo);
    __setGoBridgeCallGoJSON(mockGo);

    let dispatchNum = 0;
    __setDispatchWorkers(async (_opts, dispatches) => {
      dispatchNum++;
      harness.dispatchCallCount++;
      harness.capturedAllDispatches.push(dispatches);
      if (dispatchNum === 1) {
        return [
          {
            name: "Builder-01",
            status: "failed",
            summary: "Build failed",
            duration: 5,
            blockers: ["missing import"],
          },
        ];
      }
      return [
        {
          name: "Builder-01",
          status: "completed",
          summary: "Done",
          duration: 5,
          files_created: ["src/module.ts"],
          files_modified: ["src/helper.ts"],
          test_results: { passed: 8, total: 8 },
        },
      ];
    });

    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate"]);
    const bridge: GoBridgeOptions = {
      goBinaryPath: "/usr/bin/aether",
      cwd: process.cwd(),
    };

    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    await runDispatchedBuildCommand(bridge, parsed, def);

    // Verify 2 iterations (2 build-finalize calls)
    const finalizeCount = countGoCommandCalls(harness.goCalls, "build-finalize");
    assert.equal(finalizeCount, 2, "Should run 2 iterations: low then high confidence");
    assert.equal(harness.dispatchCallCount, 2, "Should dispatch 2 waves");

    // Verify ceremony shows both iterations
    assert.match(
      harness.stderrOutput,
      /Iteration 1/,
      "Ceremony should show Iteration 1",
    );
    assert.match(
      harness.stderrOutput,
      /Iteration 2/,
      "Ceremony should show Iteration 2",
    );

    // Verify stop reason is confidence_target_met
    assert.match(
      harness.stderrOutput,
      /confidence_target_met/,
      "Stop reason should be confidence_target_met",
    );
  });

  it("diminishing returns stops iteration before hard cap", async () => {
    // Confidence sequence: 40% -> 42% -> 43%
    // Deltas: +2, +1 -- both below 5% threshold (default diminishing returns)
    // With maxIterations=5, the loop should stop at iteration 3 because
    // diminishing returns triggers BEFORE the hard cap of 5.
    // (Using maxIterations > 3 so the hard cap doesn't preempt diminishing returns.)
    const mockGo = createIterationMockGo(harness, {
      workerResults: (callNum) => {
        // Build results that produce specific confidence scores:
        // Call 0: base=50, no tests, no files, 1 blocker (-10) = 40
        // Call 1: base=50, no tests, files (+10), completed (+10), 1 blocker (-10) = 60... too high
        // Instead, we need to carefully craft results to get 40, 42, 43.
        // Simplest: use test_results to control the score precisely.
        const scores = [
          { passed: 0, total: 10 },   // test_bonus = 0 -> score ~40 (base 50 - blocker 10)
          { passed: 1, total: 10 },   // test_bonus = 3 -> score ~43 (+ file 10 + completed 10 - blocker 10)... hard to get exact
        ];
        // Let's use a different approach: no blockers so we get predictable scores
        // base=50, file_bonus=10, all_completed=10 = 70. With test_bonus:
        // 0/10 tests: 70 -> 70
        // Actually let's just use the raw approach:
        // Iteration 1: status=failed, blockers -> score will be low
        // Iteration 2: partial improvement
        // Iteration 3: tiny improvement

        // For diminishing returns, the key is small deltas between iterations.
        // We craft worker results to produce small score changes.

        // Using: base=50 + test_bonus(30 * passed/total) + file_bonus(10) + all_completed(10) - blocker_penalty
        // Iter 1: 50 + 0 + 0 + 0 - 10 = 40 (status failed -> not all completed, no files, 1 blocker)
        // Iter 2: 50 + 0 + 10 + 0 - 10 = 50 -> wait, let's do it differently

        // Simplest approach for diminishing returns:
        // Iter 1: no tests, no files, 1 blocker, failed -> base=50 -10 = 40
        // Iter 2: same as iter 1 but with 1 file (+10), still 1 blocker -> 50
        // That's +10 delta, not diminishing...

        // Better approach for exact control: use a custom maxIterations and
        // carefully crafted results. Actually, the ConfidenceLoop checkStopConditions
        // runs BEFORE the next iteration, so with maxIterations=3:
        // Iteration 1: confidence=40, stopReason="" (continue)
        // Iteration 2: confidence=42, delta=+2, stopReason="" (continue, need 2 consecutive)
        // Iteration 3: confidence=43, delta=+1, check: last 2 deltas = +2, +1, both < 5 -> diminishing_returns

        // For iter 1: 50 + 0 + 0 + 0 - 10 = 40 (failed, 1 blocker)
        // For iter 2: 50 + 0 + 0 + 0 - 8 = 42 (failed, no... penalty is per blocker at -10 each)
        // Hmm, penalties are 10 each. Can't get 42 exactly with integer penalties.

        // Let's use test_results to fine-tune:
        // Score = 50 + 30*(passed/total) + 10(files) + 10(completed) - 10*blockers
        // For 42: 50 + 30*x + 0 + 0 - 10 = 42 -> 30*x = 2 -> x = 2/30 -> 2 passed out of 30
        // For 43: 50 + 30*y + 0 + 0 - 10 = 43 -> 30*y = 3 -> y = 3/30 -> 3 passed out of 30

        // Wait, we also need iteration 1 at 40.
        // Iter 1: 50 + 0 + 0 + 0 - 10 = 40 (failed, 1 blocker)
        // But to get diminishing we need the confidence history [40, 42, 43]
        // which means iter 2 evaluates to 42 and iter 3 to 43.
        // But iter 2 workers would need to produce 42...

        // Actually the simplest way: use confidenceLoop directly.
        // But the plan says "exercises full build dispatch loop via mocked bridge".
        // So let's create results that produce diminishing deltas.

        // Use: base=50, 1 blocker(-10), no tests, no files, status=failed
        // Iter 1: score = 40
        // Iter 2: base=50, 1 blocker(-10), test_bonus = 30*1/15 = 2, no files, failed -> 50+2-10=42
        // Iter 3: base=50, 1 blocker(-10), test_bonus = 30*1/10 = 3, no files, failed -> 50+3-10=43
        if (callNum === 0) {
          return [{
            name: "Builder-01", status: "failed", summary: "Partial", duration: 5,
            blockers: ["config error"],
          }];
        }
        if (callNum === 1) {
          return [{
            name: "Builder-01", status: "failed", summary: "Partial", duration: 5,
            blockers: ["config error"],
            test_results: { passed: 1, total: 15 },
          }];
        }
        return [{
          name: "Builder-01", status: "failed", summary: "Partial", duration: 5,
          blockers: ["config error"],
          test_results: { passed: 1, total: 10 },
        }];
      },
    });

    __setCallGoJSON(mockGo);
    __setGoBridgeCallGoJSON(mockGo);

    let dispatchNum = 0;
    __setDispatchWorkers(async (_opts, dispatches) => {
      dispatchNum++;
      harness.dispatchCallCount++;
      harness.capturedAllDispatches.push(dispatches);
      if (dispatchNum === 1) {
        return [{
          name: "Builder-01", status: "failed", summary: "Partial", duration: 5,
          blockers: ["config error"],
        }];
      }
      if (dispatchNum === 2) {
        return [{
          name: "Builder-01", status: "failed", summary: "Partial", duration: 5,
          blockers: ["config error"],
          test_results: { passed: 1, total: 15 },
        }];
      }
      return [{
        name: "Builder-01", status: "failed", summary: "Partial", duration: 5,
        blockers: ["config error"],
        test_results: { passed: 1, total: 10 },
      }];
    });

    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate", "--max-iterations", "5"]);
    const bridge: GoBridgeOptions = {
      goBinaryPath: "/usr/bin/aether",
      cwd: process.cwd(),
    };

    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    await runDispatchedBuildCommand(bridge, parsed, def);

    // Verify 3 iterations completed (all three dispatched)
    const finalizeCount = countGoCommandCalls(harness.goCalls, "build-finalize");
    assert.equal(finalizeCount, 3, "Should run 3 iterations before stopping");
    assert.equal(harness.dispatchCallCount, 3, "Should dispatch 3 waves");

    // Verify stop reason is diminishing_returns, NOT max_iterations_met
    assert.match(
      harness.stderrOutput,
      /diminishing_returns/,
      "Stop reason should be diminishing_returns",
    );
    assert.doesNotMatch(
      harness.stderrOutput,
      /max_iterations_met/,
      "Stop reason should NOT be max_iterations_met",
    );
  });

  it("budget exhaustion prevents third iteration", async () => {
    // totalBudget=5, each iteration uses 3 workers (3 per dispatch).
    // Iteration 1: 3 workers consumed, 2 remaining -> continue
    // Iteration 2: 3 more consumed = 6 total > 5 budget -> budget exhausted
    // But ConfidenceLoop.evaluate tracks budget per-iteration, so:
    // Iter 1: budgetRemaining = 5-3 = 2, shouldContinue = true (confidence low, budget > 0)
    // Iter 2: budgetRemaining = 5-6 = -1 -> budget_exhausted
    const mockGo = createIterationMockGo(harness, {
      dispatchCount: 3,
      spawnBudget: 5,
      workerResults: () => [
        {
          name: "Builder-01", status: "failed", summary: "Failed", duration: 5,
          blockers: ["compile error"],
        },
        {
          name: "Builder-02", status: "failed", summary: "Failed", duration: 5,
          blockers: ["compile error"],
        },
        {
          name: "Builder-03", status: "failed", summary: "Failed", duration: 5,
          blockers: ["compile error"],
        },
      ],
    });

    __setCallGoJSON(mockGo);
    __setGoBridgeCallGoJSON(mockGo);

    __setDispatchWorkers(async (_opts, dispatches) => {
      harness.dispatchCallCount++;
      harness.capturedAllDispatches.push(dispatches);
      return dispatches.map((d: BuildDispatch) => ({
        name: d.name,
        status: "failed" as const,
        summary: "Failed",
        duration: 5,
        blockers: ["compile error"],
      }));
    });

    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate"]);
    const bridge: GoBridgeOptions = {
      goBinaryPath: "/usr/bin/aether",
      cwd: process.cwd(),
    };

    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    await runDispatchedBuildCommand(bridge, parsed, def);

    // Should stop at iteration 2 because budget (5) is exhausted by 3+3=6 workers
    const finalizeCount = countGoCommandCalls(harness.goCalls, "build-finalize");
    assert.equal(finalizeCount, 2, "Should stop at 2 iterations due to budget exhaustion");
    assert.equal(harness.dispatchCallCount, 2, "Should dispatch 2 waves before budget exhausted");

    // Verify stop reason includes budget
    assert.match(
      harness.stderrOutput,
      /budget/,
      "Ceremony should mention budget",
    );
  });

  it("happy path: single iteration when confidence is already high", async () => {
    const mockGo = createIterationMockGo(harness, {
      workerResults: () => [
        {
          name: "Builder-01",
          status: "completed",
          summary: "Done",
          duration: 5,
          files_created: ["src/module.ts"],
          files_modified: ["src/helper.ts"],
          tests_written: ["test/module.test.ts"],
          test_results: { passed: 10, total: 10 },
        },
      ],
    });

    __setCallGoJSON(mockGo);
    __setGoBridgeCallGoJSON(mockGo);

    __setDispatchWorkers(async (_opts, dispatches) => {
      harness.dispatchCallCount++;
      harness.capturedAllDispatches.push(dispatches);
      return [
        {
          name: "Builder-01",
          status: "completed",
          summary: "Done",
          duration: 5,
          files_created: ["src/module.ts"],
          files_modified: ["src/helper.ts"],
          tests_written: ["test/module.test.ts"],
          test_results: { passed: 10, total: 10 },
        },
      ];
    });

    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate"]);
    const bridge: GoBridgeOptions = {
      goBinaryPath: "/usr/bin/aether",
      cwd: process.cwd(),
    };

    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    await runDispatchedBuildCommand(bridge, parsed, def);

    // Exactly 1 iteration
    const finalizeCount = countGoCommandCalls(harness.goCalls, "build-finalize");
    assert.equal(finalizeCount, 1, "Should run exactly 1 iteration when confidence is high");
    assert.equal(harness.dispatchCallCount, 1, "Should dispatch exactly 1 wave");

    // Stop reason should be confidence_target_met
    assert.match(
      harness.stderrOutput,
      /confidence_target_met/,
      "Stop reason should be confidence_target_met",
    );

    // Should NOT show Iteration 2
    assert.doesNotMatch(
      harness.stderrOutput,
      /Iteration 2/,
      "Should not show Iteration 2 for single-iteration happy path",
    );
  });
});

// ---------------------------------------------------------------------------
// 2. E2E: feedback injection across iterations
// ---------------------------------------------------------------------------

describe("E2E: feedback injection across iterations", () => {
  let harness: TestHarness;
  let cleanup: () => void;

  beforeEach(() => {
    harness = createTestHarness();
    cleanup = setupMocks(harness);
  });

  afterEach(() => {
    cleanup();
  });

  it("blockers from iteration 1 appear in iteration 2 task briefs", async () => {
    const mockGo = createIterationMockGo(harness, {
      workerResults: (callNum) => {
        if (callNum === 0) {
          return [{
            name: "Builder-01", status: "failed", summary: "Build failed", duration: 5,
            blockers: ["module X not found"],
          }];
        }
        return [{
          name: "Builder-01", status: "completed", summary: "Done", duration: 5,
          files_created: ["src/module.ts"],
          test_results: { passed: 5, total: 5 },
        }];
      },
    });

    __setCallGoJSON(mockGo);
    __setGoBridgeCallGoJSON(mockGo);

    let dispatchNum = 0;
    __setDispatchWorkers(async (_opts, dispatches) => {
      dispatchNum++;
      harness.dispatchCallCount++;
      harness.capturedAllDispatches.push(dispatches);
      if (dispatchNum === 1) {
        return [{
          name: "Builder-01", status: "failed", summary: "Build failed", duration: 5,
          blockers: ["module X not found"],
        }];
      }
      return [{
        name: "Builder-01", status: "completed", summary: "Done", duration: 5,
        files_created: ["src/module.ts"],
        test_results: { passed: 5, total: 5 },
      }];
    });

    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate"]);
    const bridge: GoBridgeOptions = {
      goBinaryPath: "/usr/bin/aether",
      cwd: process.cwd(),
    };

    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    await runDispatchedBuildCommand(bridge, parsed, def);

    // Verify 2 dispatches occurred
    const finalizeCount = countGoCommandCalls(harness.goCalls, "build-finalize");
    assert.equal(finalizeCount, 2, "Should call build-finalize twice");
    assert.equal(harness.dispatchCallCount, 2, "Should dispatch 2 waves");

    // Verify second dispatch has feedback from first iteration's blockers
    assert.ok(
      harness.capturedAllDispatches.length >= 2,
      "Should have captured at least 2 dispatch calls",
    );
    const secondDispatch = harness.capturedAllDispatches[1]!;
    assert.ok(secondDispatch.length > 0, "Second dispatch should have dispatches");

    const dispatch = secondDispatch[0] as Record<string, unknown>;
    assert.ok(
      typeof dispatch.task_brief === "string" && dispatch.task_brief.includes("module X not found"),
      `Second iteration task_brief should contain blocker text. Got: ${dispatch.task_brief}`,
    );
    assert.ok(
      typeof dispatch.task_brief === "string" && dispatch.task_brief.includes("Previous iteration feedback"),
      `Second iteration task_brief should contain feedback header. Got: ${dispatch.task_brief}`,
    );
  });
});

// ---------------------------------------------------------------------------
// 3. E2E: ceremony output
// ---------------------------------------------------------------------------

describe("E2E: ceremony output", () => {
  let harness: TestHarness;
  let cleanup: () => void;

  beforeEach(() => {
    harness = createTestHarness();
    cleanup = setupMocks(harness);
  });

  afterEach(() => {
    cleanup();
  });

  it("ceremony shows iteration markers with correct data", async () => {
    const mockGo = createIterationMockGo(harness, {
      workerResults: (callNum) => {
        if (callNum === 0) {
          return [{
            name: "Builder-01", status: "failed", summary: "Build failed", duration: 5,
            blockers: ["compile error"],
          }];
        }
        return [{
          name: "Builder-01", status: "completed", summary: "Done", duration: 5,
          files_created: ["src/module.ts"],
          test_results: { passed: 5, total: 5 },
        }];
      },
    });

    __setCallGoJSON(mockGo);
    __setGoBridgeCallGoJSON(mockGo);

    let dispatchNum = 0;
    __setDispatchWorkers(async (_opts, dispatches) => {
      dispatchNum++;
      harness.dispatchCallCount++;
      harness.capturedAllDispatches.push(dispatches);
      if (dispatchNum === 1) {
        return [{
          name: "Builder-01", status: "failed", summary: "Build failed", duration: 5,
          blockers: ["compile error"],
        }];
      }
      return [{
        name: "Builder-01", status: "completed", summary: "Done", duration: 5,
        files_created: ["src/module.ts"],
        test_results: { passed: 5, total: 5 },
      }];
    });

    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate"]);
    const bridge: GoBridgeOptions = {
      goBinaryPath: "/usr/bin/aether",
      cwd: process.cwd(),
    };

    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    await runDispatchedBuildCommand(bridge, parsed, def);

    // Verify both iteration markers present
    assert.match(
      harness.stderrOutput,
      /Iteration 1/,
      "Ceremony should show Iteration 1",
    );
    assert.match(
      harness.stderrOutput,
      /Iteration 2/,
      "Ceremony should show Iteration 2",
    );

    // Verify confidence values shown
    assert.match(
      harness.stderrOutput,
      /confidence \d+%/,
      "Ceremony should show confidence percentage",
    );

    // Verify delta shown
    assert.match(
      harness.stderrOutput,
      /delta [+-]?\d+%/,
      "Ceremony should show delta percentage",
    );
  });

  it("final ceremony shows stop reason", async () => {
    const mockGo = createIterationMockGo(harness, {
      workerResults: () => [
        {
          name: "Builder-01", status: "completed", summary: "Done", duration: 5,
          files_created: ["src/module.ts"],
          test_results: { passed: 10, total: 10 },
        },
      ],
    });

    __setCallGoJSON(mockGo);
    __setGoBridgeCallGoJSON(mockGo);

    __setDispatchWorkers(async (_opts, dispatches) => {
      harness.dispatchCallCount++;
      harness.capturedAllDispatches.push(dispatches);
      return [
        {
          name: "Builder-01", status: "completed", summary: "Done", duration: 5,
          files_created: ["src/module.ts"],
          test_results: { passed: 10, total: 10 },
        },
      ];
    });

    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate"]);
    const bridge: GoBridgeOptions = {
      goBinaryPath: "/usr/bin/aether",
      cwd: process.cwd(),
    };

    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    await runDispatchedBuildCommand(bridge, parsed, def);

    // Verify stop reason is in ceremony
    assert.match(
      harness.stderrOutput,
      /Iteration complete.*confidence_target_met/,
      "Closeout ceremony should contain stop reason",
    );
  });
});

// ---------------------------------------------------------------------------
// 4. E2E: edge cases
// ---------------------------------------------------------------------------

describe("E2E: edge cases", () => {
  let harness: TestHarness;
  let cleanup: () => void;

  beforeEach(() => {
    harness = createTestHarness();
    cleanup = setupMocks(harness);
  });

  afterEach(() => {
    cleanup();
  });

  it("zero workers in manifest throws error", async () => {
    const mockGo = <T>(_bridgeOpts: unknown, args: string[]): T => {
      harness.goCalls.push(args);
      const cmd = args[0];
      if (cmd === "hive-read") {
        return { entries: null, total: 0 } as unknown as T;
      }
      if (cmd === "registry-list") {
        return {
          colonies: [{
            repo_path: process.cwd(),
            domains: ["ts"],
            active: true,
            registered_at: "2026-05-18T00:00:00Z",
          }],
        } as unknown as T;
      }
      if (cmd === "build") {
        // Empty dispatches
        return {
          dispatch_manifest: { dispatches: [] },
        } as unknown as T;
      }
      return { ok: true } as unknown as T;
    };

    __setCallGoJSON(mockGo);
    __setGoBridgeCallGoJSON(mockGo);

    __setDispatchWorkers(async (_opts, _dispatches) => {
      harness.dispatchCallCount++;
      return [];
    });

    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate"]);
    const bridge: GoBridgeOptions = {
      goBinaryPath: "/usr/bin/aether",
      cwd: process.cwd(),
    };

    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    await assert.rejects(
      async () => {
        await runDispatchedBuildCommand(bridge, parsed, def);
      },
      (err: unknown) => {
        const message = err instanceof Error ? err.message : String(err);
        return message.includes("no dispatches") || message.includes("Nothing to build");
      },
      "Should throw error when manifest has no dispatches",
    );

    // Verify no iterations ran
    assert.equal(harness.dispatchCallCount, 0, "Should not dispatch any workers");
  });

  it("all workers fail on every iteration", async () => {
    const mockGo = createIterationMockGo(harness, {
      workerResults: () => [
        {
          name: "Builder-01", status: "failed", summary: "Build failed", duration: 5,
          blockers: ["compile error"],
        },
      ],
    });

    __setCallGoJSON(mockGo);
    __setGoBridgeCallGoJSON(mockGo);

    __setDispatchWorkers(async (_opts, dispatches) => {
      harness.dispatchCallCount++;
      harness.capturedAllDispatches.push(dispatches);
      return dispatches.map((d: BuildDispatch) => ({
        name: d.name,
        status: "failed" as const,
        summary: "Failed",
        duration: 5,
        blockers: ["compile error"],
      }));
    });

    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate"]);
    const bridge: GoBridgeOptions = {
      goBinaryPath: "/usr/bin/aether",
      cwd: process.cwd(),
    };

    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    await runDispatchedBuildCommand(bridge, parsed, def);

    // Should stop at max iterations (default 3)
    const finalizeCount = countGoCommandCalls(harness.goCalls, "build-finalize");
    assert.equal(finalizeCount, 3, "Should run exactly 3 iterations (default max)");
    assert.equal(harness.dispatchCallCount, 3, "Should dispatch 3 waves");

    // Verify stop reason
    assert.match(
      harness.stderrOutput,
      /Iteration complete/,
      "Ceremony should show iteration complete",
    );
  });
});
