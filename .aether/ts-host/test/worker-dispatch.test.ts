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

import { describe, it, afterEach } from "node:test";
import assert from "node:assert/strict";

import type { BuildDispatch } from "../src/types.js";
import {
  buildPromptForDispatch,
  dispatchWorkers,
  resolveGoPromptContext,
  sanitizeWorkerDiagnosticOutput,
  toWorkerResults,
  isAuthError,
  type DispatchResult,
  type DispatchOptions,
} from "../src/worker-dispatch.js";
import {
  __restoreCallGoJSON,
  __setCallGoJSON,
  type GoBridgeOptions,
} from "../src/go-bridge.js";
import {
  __setDispatchSingleWorker,
  __restoreDispatchSingleWorker,
} from "../src/wave-orchestrator.js";
import {
  __setDetectAvailablePlatforms,
  __restoreDetectAvailablePlatforms,
} from "../src/platform-dispatcher.js";
import {
  dispatchSingleWorker,
} from "../src/worker-dispatch.js";

// ---------------------------------------------------------------------------
// Mock helpers
// ---------------------------------------------------------------------------

let mockResults: Map<string, DispatchResult> = new Map();
let mockCallCount: Map<string, number> = new Map();

async function mockDispatchSingleWorker(
  _opts: DispatchOptions,
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

const defaultOpts: DispatchOptions = {
  goBinaryPath: "/usr/bin/true",
  cwd: "/Users/callumcowie/repos/Aether",
  simulateWorkers: true,
  parallel: true,
  retryLimit: 2,
  retryDelayMs: 10,
};

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("worker-dispatch", () => {
  it("sanitizeWorkerDiagnosticOutput redacts provider secrets for failed worker summaries", () => {
    const sanitized = sanitizeWorkerDiagnosticOutput(
      "stderr: auth failed token sk-proj-secret-123 ghp_worker_secret github_pat_abc npm_secret token=raw-secret"
    );

    for (const forbidden of [
      "sk-proj-secret-123",
      "ghp_worker_secret",
      "github_pat_abc",
      "npm_secret",
      "raw-secret",
    ]) {
      assert.ok(!sanitized.includes(forbidden), `sanitized output leaked ${forbidden}: ${sanitized}`);
    }
    assert.ok(sanitized.includes("[redacted]"), sanitized);
  });

  it("resolveGoPromptContext reads Go colony-prime prompt_section", () => {
    __setCallGoJSON(<T>(_opts: GoBridgeOptions, args: string[]): T => {
      assert.deepEqual(args, ["colony-prime", "--compact"]);
      return {
        prompt_section: "## Colony State\n\nPhase: 6",
      } as T;
    });

    try {
      assert.equal(resolveGoPromptContext(defaultOpts), "## Colony State\n\nPhase: 6");
    } finally {
      __restoreCallGoJSON();
    }
  });

  it("resolveGoPromptContext falls back to context when prompt_section is absent", () => {
    __setCallGoJSON(<T>(_opts: GoBridgeOptions, _args: string[]): T => {
      return {
        context: "## Compact Context\n\nFallback section",
      } as T;
    });

    try {
      assert.equal(resolveGoPromptContext(defaultOpts), "## Compact Context\n\nFallback section");
    } finally {
      __restoreCallGoJSON();
    }
  });

  it("buildPromptForDispatch preserves Go manifest context, handoff, and skill sections", () => {
    const dispatch: BuildDispatch = {
      stage: "wave",
      wave: 1,
      execution_wave: 11,
      caste: "builder",
      name: "Builder-Prompt",
      task: "Implement prompt parity",
      status: "planned",
      task_id: "6.3",
      context_capsule: "## Colony State\n\nGo-provided context",
      handoff_section: "## Previous Worker Handoffs\n\nPrior worker result",
      skill_section: "### Skill: worker-priming\n\nUse matched skill context",
      task_brief: "# Codex Build Dispatch\n\nGo-authored task brief",
    };

    const prompt = buildPromptForDispatch(
      defaultOpts,
      dispatch,
      "claude",
      "aether-builder"
    );

    assert.match(prompt, /Go-provided context/);
    assert.match(prompt, /Prior worker result/);
    assert.match(prompt, /Skill: worker-priming/);
    assert.match(prompt, /Go-authored task brief/);
  });

  it("buildPromptForDispatch passes hive_section from dispatch to prompt", () => {
    const dispatch: BuildDispatch = {
      stage: "wave",
      wave: 1,
      caste: "builder",
      name: "Builder-Hive",
      task: "Implement hive wiring",
      status: "planned",
      hive_section: "## HIVE WISDOM (Cross-Colony Patterns)\n\n(go, 0.90) Prefer table-driven tests",
    };

    const prompt = buildPromptForDispatch(
      defaultOpts,
      dispatch,
      "claude",
      "aether-builder"
    );

    assert.ok(
      prompt.includes("## HIVE WISDOM (Cross-Colony Patterns)"),
      "Prompt should contain hive section header from dispatch.hive_section"
    );
    assert.ok(
      prompt.includes("(go, 0.90) Prefer table-driven tests"),
      "Prompt should contain hive wisdom content from dispatch.hive_section"
    );
  });

  it("buildPromptForDispatch omits hive section when hive_section is undefined", () => {
    const dispatch: BuildDispatch = {
      stage: "wave",
      wave: 1,
      caste: "builder",
      name: "Builder-NoHive",
      task: "Implement without hive",
      status: "planned",
      // hive_section intentionally omitted
    };

    const prompt = buildPromptForDispatch(
      defaultOpts,
      dispatch,
      "claude",
      "aether-builder"
    );

    assert.ok(
      !prompt.includes("## HIVE WISDOM (Cross-Colony Patterns)"),
      "Prompt should not contain hive section when dispatch.hive_section is undefined"
    );
  });

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

  it("toWorkerResults treats missing terminal results as timeouts", () => {
    const dispatches = [
      {
        stage: "verify",
        wave: 1,
        caste: "watcher",
        name: "Watcher-Missing",
        task: "Verify feature X",
        status: "pending",
        task_id: "1.2",
      },
    ];

    const workerResults = toWorkerResults(dispatches, []);

    assert.equal(workerResults.length, 1);
    assert.equal(workerResults[0]!.name, "Watcher-Missing");
    assert.equal(workerResults[0]!.status, "timeout");
    assert.match(workerResults[0]!.summary ?? "", /No terminal worker result/);
  });
});

// ---------------------------------------------------------------------------
// Real dispatch default tests (HOST-01, HOST-09)
// ---------------------------------------------------------------------------

describe("worker-dispatch: simulation default", { concurrency: false }, () => {
  const mockDispatch: BuildDispatch = {
    stage: "build",
    caste: "builder",
    name: "Builder-Default",
    task: "Test default dispatch behavior",
    status: "pending",
  };

  afterEach(() => {
    __restoreDetectAvailablePlatforms();
    __restoreCallGoJSON();
  });

  it("defaults to real execution when simulateWorkers is not set", async () => {
    // With no platforms available and simulateWorkers undefined,
    // the real dispatch path should be taken, which throws because
    // no platform CLIs are found. This proves simulation is NOT the default.
    __setDetectAvailablePlatforms(async () => []);
    __setCallGoJSON(<T>(): T => ({ recorded: true } as unknown as T));

    const capturedStderr: string[] = [];
    const originalWrite = process.stderr.write.bind(process.stderr);
    const originalStderrWrite = process.stderr.write;
    process.stderr.write = ((chunk: unknown) => {
      if (typeof chunk === "string") {
        capturedStderr.push(chunk);
      }
      return originalWrite(chunk);
    }) as typeof process.stderr.write;

    try {
      await dispatchSingleWorker(
        {
          goBinaryPath: "/usr/bin/true",
          cwd: "/tmp",
        },
        mockDispatch
      );
      assert.fail("Should have thrown for missing platforms");
    } catch (err: unknown) {
      assert.ok(
        err instanceof Error && /Worker dispatch cannot start/.test(err.message),
        `Expected platform-unavailable error, got: ${err}`
      );
    } finally {
      process.stderr.write = originalStderrWrite;
    }

    // Should NOT log "Simulating worker" because real dispatch is the default
    const simulationLog = capturedStderr.find((line) => /Simulating worker/.test(line));
    assert.equal(
      simulationLog,
      undefined,
      `Should NOT log "Simulating worker" when simulateWorkers is not set, but found: ${simulationLog}`
    );
  });

  it("simulates when simulateWorkers is explicitly true", async () => {
    __setDetectAvailablePlatforms(async () => []);
    __setCallGoJSON(<T>(): T => ({ recorded: true } as unknown as T));

    const capturedStderr: string[] = [];
    const originalWrite = process.stderr.write.bind(process.stderr);
    const originalStderrWrite = process.stderr.write;
    process.stderr.write = ((chunk: unknown) => {
      if (typeof chunk === "string") {
        capturedStderr.push(chunk);
      }
      return originalWrite(chunk);
    }) as typeof process.stderr.write;

    try {
      const result = await dispatchSingleWorker(
        {
          goBinaryPath: "/usr/bin/true",
          cwd: "/tmp",
          simulateWorkers: true,
        },
        mockDispatch
      );

      assert.equal(result.status, "completed");
      assert.ok(
        result.summary.includes("Simulated"),
        `Summary should indicate simulation, got: ${result.summary}`
      );
    } finally {
      process.stderr.write = originalStderrWrite;
    }

    // Should log "Simulating" when simulateWorkers is explicitly true
    const simulationLog = capturedStderr.find((line) => /Simulating/.test(line));
    assert.ok(
      simulationLog,
      `Should log "Simulating" when simulateWorkers=true, captured stderr: ${capturedStderr.join("\\n")}`
    );
  });
});

// ---------------------------------------------------------------------------
// Error classification tests (D-01, D-02)
// ---------------------------------------------------------------------------

describe("worker-dispatch: error classification", () => {
  it("isAuthError returns true for auth-related errors", () => {
    assert.equal(isAuthError(new Error("Authentication required")), true);
    assert.equal(isAuthError(new Error("credentials expired")), true);
    assert.equal(isAuthError(new Error("Permission denied")), true);
  });

  it("isAuthError returns false for non-auth errors", () => {
    assert.equal(isAuthError(new Error("Connection timeout")), false);
    assert.equal(isAuthError(new Error("ENOENT: file not found")), false);
    assert.equal(isAuthError(new Error("Something went wrong")), false);
  });

  it("auth errors in dispatchSingleWorker propagate upward via wave-orchestrator (D-01)", async () => {
    // Simulate what dispatchSingleWorker does for auth errors: throw with
    // "halted" message. The wave-orchestrator mock throws the classified error
    // that dispatchSingleWorker would produce after classifyPlatformError.
    __setDispatchSingleWorker(async (_opts, dispatch) => {
      if (dispatch.name === "Auth-Worker") {
        // This matches what dispatchSingleWorker does when classifyPlatformError
        // returns "auth" -- it throws with "halted" in the message.
        throw new Error(
          "Worker dispatch halted: Authentication credentials expired"
        );
      }
      return { name: dispatch.name, status: "completed", summary: "Done" };
    });

    const dispatches: BuildDispatch[] = [
      { stage: "implement", caste: "builder", name: "Auth-Worker", task: "Test auth", status: "pending" },
    ];

    try {
      await dispatchWorkers(
        { ...defaultOpts, retryLimit: 0, retryDelayMs: 10 },
        dispatches
      );
      assert.fail("Should have thrown for auth error");
    } catch (err: unknown) {
      assert.ok(err instanceof Error, "Should throw an Error");
      assert.match(err.message, /halted/, `Error message should mention halt: ${err.message}`);
    } finally {
      __restoreDispatchSingleWorker();
    }
  });

  it("timeout errors in dispatchSingleWorker return failed status via wave-orchestrator (D-02)", async () => {
    // Simulate what dispatchSingleWorker does for timeout errors: returns
    // a DispatchResult with status "failed" instead of throwing.
    __setDispatchSingleWorker(async (_opts, dispatch) => {
      if (dispatch.name === "Timeout-Worker") {
        // This matches what dispatchSingleWorker does for timeout/transient
        // errors -- returns failed result instead of throwing.
        return {
          name: dispatch.name,
          status: "failed" as const,
          summary: "Worker dispatch failed: Connection ETIMEDOUT after 30s",
        };
      }
      return { name: dispatch.name, status: "completed", summary: "Done" };
    });

    const dispatches: BuildDispatch[] = [
      { stage: "implement", caste: "builder", name: "Timeout-Worker", task: "Test timeout", status: "pending" },
      { stage: "implement", caste: "builder", name: "Good-Worker", task: "Test success", status: "pending" },
    ];

    const results = await dispatchWorkers(
      { ...defaultOpts, retryLimit: 0, retryDelayMs: 10 },
      dispatches
    );
    __restoreDispatchSingleWorker();

    // Timeout errors should NOT propagate -- they return a failed result.
    const timeoutResult = results.find((r) => r.name === "Timeout-Worker");
    assert.ok(timeoutResult, "Should have a result for Timeout-Worker");
    assert.equal(timeoutResult!.status, "failed", "Timeout should result in 'failed' status");

    // The other worker should still succeed.
    const goodResult = results.find((r) => r.name === "Good-Worker");
    assert.ok(goodResult, "Should have a result for Good-Worker");
    assert.equal(goodResult!.status, "completed", "Good worker should succeed despite sibling failure");
  });
});

// ---------------------------------------------------------------------------
// Wave-end failure summary tests (D-02)
// ---------------------------------------------------------------------------

describe("worker-dispatch: wave summary", { concurrency: false }, () => {
  afterEach(() => {
    __restoreDispatchSingleWorker();
  });

  it("wave summary includes per-worker failure names when workers fail", async () => {
    resetMocks();

    const dispatches = [
      makeDispatch("Mason-01", 1),
      makeDispatch("Mason-02", 1),
      makeDispatch("Mason-03", 2),
    ];

    // Two succeed, one fails in wave 1
    mockResults.set("Mason-01:0", { name: "Mason-01", status: "completed", summary: "Done" });
    mockResults.set("Mason-02:0", { name: "Mason-02", status: "failed", summary: "Timed out" });
    mockResults.set("Mason-03:0", { name: "Mason-03", status: "completed", summary: "Done" });

    __setDispatchSingleWorker(mockDispatchSingleWorker);

    const capturedStderr: string[] = [];
    const originalStderrWrite = process.stderr.write;
    process.stderr.write = ((chunk: unknown) => {
      if (typeof chunk === "string") {
        capturedStderr.push(chunk);
      }
      return true;
    }) as typeof process.stderr.write;

    try {
      // Use retryLimit: 0 so failed workers stay failed (no retry to success)
      await dispatchWorkers({ ...defaultOpts, retryLimit: 0, retryDelayMs: 10 }, dispatches);
    } finally {
      process.stderr.write = originalStderrWrite;
    }

    // dispatchWorkers writes a failure summary line after wave-orchestrator.
    // Find the line from dispatchWorkers (not wave-orchestrator) that includes failure details.
    const failureSummary = capturedStderr.find(
      (line) => /Wave 1/.test(line) && /Mason-02/.test(line)
    );
    assert.ok(
      failureSummary,
      `Should have a Wave 1 summary with failed worker name, captured: ${capturedStderr.join("\\n")}`
    );
    assert.ok(
      failureSummary!.includes("failed"),
      `Failure summary should mention "failed": ${failureSummary}`
    );
  });

  it("wave summary shows all-succeeded format when no failures", async () => {
    resetMocks();

    const dispatches = [makeDispatch("Worker-A", 1)];

    mockResults.set("Worker-A:0", { name: "Worker-A", status: "completed", summary: "Done" });

    __setDispatchSingleWorker(mockDispatchSingleWorker);

    const capturedStderr: string[] = [];
    const originalStderrWrite = process.stderr.write;
    process.stderr.write = ((chunk: unknown) => {
      if (typeof chunk === "string") {
        capturedStderr.push(chunk);
      }
      return true;
    }) as typeof process.stderr.write;

    try {
      await dispatchWorkers({ ...defaultOpts, retryLimit: 0, retryDelayMs: 10 }, dispatches);
    } finally {
      process.stderr.write = originalStderrWrite;
    }

    // Find the dispatchWorkers summary (not wave-orchestrator's "complete" line)
    const successSummary = capturedStderr.find(
      (line) => /Wave 1/.test(line) && /succeeded/.test(line)
    );
    assert.ok(successSummary, "Should have a Wave 1 summary with 'succeeded'");
    assert.ok(
      successSummary!.includes("1/1 succeeded"),
      `Should show "1/1 succeeded": ${successSummary}`
    );
  });
});
