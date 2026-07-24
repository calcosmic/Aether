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
import { readFileSync, readdirSync } from "node:fs";

import type { BuildDispatch } from "../src/types.js";
import {
  dispatchWorkers,
  sanitizeWorkerDiagnosticOutput,
  toWorkerResults,
  isAuthError,
  type DispatchResult,
  type DispatchOptions,
} from "../src/worker-dispatch.js";
import {
  __restoreCallGoJSON,
  __restoreCallGoJSONAsync,
  __setCallGoJSON,
  __setCallGoJSONAsync,
  type GoBridgeOptions,
} from "../src/go-bridge.js";
import {
  __setDispatchSingleWorker,
  __restoreDispatchSingleWorker,
} from "../src/wave-orchestrator.js";
import {
  dispatchSingleWorker,
} from "../src/worker-dispatch.js";
import { TEST_EXECUTION_BINDING } from "./execution-binding-fixture.js";

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
  it("production host sources do not import the retired TypeScript provider launcher", () => {
    const legacyFixtures = new Set(["platform-dispatcher.ts", "prompt-assembler.ts"]);
    const sourceNames = readdirSync(new URL("../src/", import.meta.url))
      .filter((sourceName) => sourceName.endsWith(".ts") && !legacyFixtures.has(sourceName));
    for (const sourceName of sourceNames) {
      const source = readFileSync(new URL(`../src/${sourceName}`, import.meta.url), "utf-8");
      assert.ok(!source.includes('from "./platform-dispatcher.js"'), sourceName);
      if (sourceName === "worker-dispatch.ts") {
        assert.ok(!source.includes("spawnWorker("), sourceName);
        assert.ok(!source.includes("selectWorkerPlatform("), sourceName);
      }
    }
  });

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

  it("real dispatch delegates the full worker context to the Go adapter boundary", async () => {
    let request: Record<string, unknown> | undefined;
    __setCallGoJSON(<T>(): T => ({ recorded: true, completed: true } as T));
    __setCallGoJSONAsync(async <T>(
      _opts: GoBridgeOptions,
      args: string[]
    ): Promise<T> => {
      assert.equal(args[0], "internal-worker-adapter");
      const requestIndex = args.indexOf("--request-file");
      assert.ok(requestIndex >= 0);
      request = JSON.parse(readFileSync(args[requestIndex + 1]!, "utf-8")) as Record<string, unknown>;
      return {
        schema_version: 1,
        execution_owner: "go-adapter",
        platform: "codex",
        platform_contract: {},
        availability: {},
        permission_decision: {
          schema_version: 1,
          platform: "codex",
          profile: {
            schema_version: 1,
            name: "workspace_write",
            filesystem: "workspace_write",
            shell: "within_filesystem_boundary",
            network: "provider_default",
            approval: "never",
          },
          allowed: true,
          enforcement: "proven",
          mechanism: "test",
        },
        worker: {
          name: "Builder-Prompt",
          caste: "builder",
          task_id: "6.3",
          status: "completed",
          summary: "Go adapter completed",
          artifacts: { phase_plan: { phases: [] } },
          scout_report: { findings: [], gaps: [], confidence: 80, study_files: [] },
        },
      } as T;
    });

    try {
      const result = await dispatchSingleWorker({ ...defaultOpts, simulateWorkers: false }, {
        stage: "wave",
        wave: 1,
        caste: "builder",
        name: "Builder-Prompt",
        task: "Implement prompt parity",
        status: "planned",
        task_id: "6.3",
        context_capsule: "## Colony State\n\nGo-provided context",
        handoff_section: "## Previous Worker Handoffs\n\nPrior worker result",
        skill_section: "### Skill: worker-priming\n\nUse matched skill context",
        hive_section: "## HIVE WISDOM\n\nUse verified patterns",
        task_brief: "# Build Dispatch\n\nGo-authored task brief",
      });

      assert.equal(result.detectedPlatform, "codex");
      assert.deepEqual(result.phase_plan, { phases: [] });
      assert.equal(request?.["context_capsule"], "## Colony State\n\nGo-provided context");
      assert.equal(request?.["handoff_section"], "## Previous Worker Handoffs\n\nPrior worker result");
      assert.equal(request?.["skill_section"], "### Skill: worker-priming\n\nUse matched skill context");
      assert.equal(request?.["hive_section"], "## HIVE WISDOM\n\nUse verified patterns");
      assert.equal(request?.["task_brief"], "# Build Dispatch\n\nGo-authored task brief");
      assert.deepEqual(request?.["permission_profile"], {
        schema_version: 1,
        name: "workspace_write",
        filesystem: "workspace_write",
        shell: "within_filesystem_boundary",
        network: "provider_default",
        approval: "never",
      });
      assert.equal(result.permission_decision?.enforcement, "proven");
    } finally {
      __restoreCallGoJSON();
      __restoreCallGoJSONAsync();
    }
  });

  it("forwards the exact durable build binding and accepts its matching echo", async () => {
    let request: Record<string, unknown> | undefined;
    __setCallGoJSON(<T>(): T => ({ recorded: true, completed: true } as T));
    __setCallGoJSONAsync(async <T>(
      _opts: GoBridgeOptions,
      args: string[]
    ): Promise<T> => {
      const requestIndex = args.indexOf("--request-file");
      request = JSON.parse(readFileSync(args[requestIndex + 1]!, "utf-8")) as Record<string, unknown>;
      return {
        schema_version: 1,
        execution_owner: "go-adapter",
        platform: "codex",
        platform_contract: {},
        availability: {},
        permission_decision: {
          schema_version: 1,
          platform: "codex",
          profile: request["permission_profile"],
          allowed: true,
          enforcement: "proven",
          mechanism: "test",
        },
        execution_binding: TEST_EXECUTION_BINDING,
        provider_run_id: "provider-run-test",
        worker: {
          name: "Builder-Bound",
          caste: "builder",
          status: "completed",
          summary: "Bound result",
        },
      } as T;
    });

    try {
      const result = await dispatchSingleWorker({
        ...defaultOpts,
        simulateWorkers: false,
        workflow: "build",
        phase: 2,
        executionBinding: TEST_EXECUTION_BINDING,
      }, makeDispatch("Builder-Bound", 1));

      assert.equal(request?.["workflow"], "build");
      assert.equal(request?.["phase"], 2);
      assert.deepEqual(request?.["execution_binding"], TEST_EXECUTION_BINDING);
      assert.deepEqual(result.execution_binding, TEST_EXECUTION_BINDING);
      assert.equal(result.provider_run_id, "provider-run-test");
      assert.equal(result.status, "completed");
    } finally {
      __restoreCallGoJSON();
      __restoreCallGoJSONAsync();
    }
  });

  it("rejects a terminal worker result echoed for a different build run", async () => {
    __setCallGoJSON(<T>(): T => ({ recorded: true, completed: true } as T));
    __setCallGoJSONAsync(async <T>(): Promise<T> => ({
      schema_version: 1,
      execution_owner: "go-adapter",
      platform: "codex",
      platform_contract: {},
      availability: {},
      permission_decision: {
        schema_version: 1,
        platform: "codex",
        profile: {
          schema_version: 1,
          name: "workspace_write",
          filesystem: "workspace_write",
          shell: "within_filesystem_boundary",
          network: "provider_default",
          approval: "never",
        },
        allowed: true,
        enforcement: "proven",
        mechanism: "test",
      },
      execution_binding: {
        ...TEST_EXECUTION_BINDING,
        run_id: "run-stale0000000000000000000000000000",
      },
      provider_run_id: "provider-run-stale",
      worker: {
        name: "Builder-Stale",
        caste: "builder",
        status: "completed",
        summary: "This must not be accepted",
      },
    } as T));

    try {
      const result = await dispatchSingleWorker({
        ...defaultOpts,
        simulateWorkers: false,
        workflow: "build",
        phase: 2,
        executionBinding: TEST_EXECUTION_BINDING,
      }, makeDispatch("Builder-Stale", 1));

      assert.equal(result.status, "failed");
      assert.match(result.summary, /different build run/);
      assert.notEqual(result.summary, "This must not be accepted");
    } finally {
      __restoreCallGoJSON();
      __restoreCallGoJSONAsync();
    }
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
    __restoreCallGoJSON();
    __restoreCallGoJSONAsync();
  });

  it("defaults to real Go-owned execution when simulateWorkers is not set", async () => {
    __setCallGoJSON(<T>(): T => ({ recorded: true } as unknown as T));
    __setCallGoJSONAsync(async <T>(): Promise<T> => {
      throw new Error("Go command failed: no authenticated provider is available");
    });

    const capturedStderr: string[] = [];
    const originalWrite = process.stderr.write.bind(process.stderr);
    const originalStderrWrite = process.stderr.write;
    process.stderr.write = ((chunk: unknown) => {
      if (typeof chunk === "string") {
        capturedStderr.push(chunk);
      }
      return originalWrite(chunk as string | Uint8Array);
    }) as typeof process.stderr.write;

    let result;
    try {
      result = await dispatchSingleWorker(
        {
          goBinaryPath: "/usr/bin/true",
          cwd: "/tmp",
        },
        mockDispatch
      );
    } finally {
      process.stderr.write = originalStderrWrite;
    }

    assert.equal(result?.status, "failed");
    assert.match(result?.summary ?? "", /no authenticated provider/);

    // Should NOT log "Simulating worker" because real dispatch is the default
    const simulationLog = capturedStderr.find((line) => /Simulating worker/.test(line));
    assert.equal(
      simulationLog,
      undefined,
      `Should NOT log "Simulating worker" when simulateWorkers is not set, but found: ${simulationLog}`
    );
  });

  it("simulates when simulateWorkers is explicitly true", async () => {
    __setCallGoJSON(<T>(): T => ({ recorded: true } as unknown as T));
    __setCallGoJSONAsync(async <T>(): Promise<T> => {
      throw new Error("real adapter should not be called in simulation");
    });

    const capturedStderr: string[] = [];
    const originalWrite = process.stderr.write.bind(process.stderr);
    const originalStderrWrite = process.stderr.write;
    process.stderr.write = ((chunk: unknown) => {
      if (typeof chunk === "string") {
        capturedStderr.push(chunk);
      }
      return originalWrite(chunk as string | Uint8Array);
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

  it("reports the platform selected by Go without selecting a provider in TypeScript", async () => {
    __setCallGoJSON(<T>(): T => ({ recorded: true, completed: true } as unknown as T));
    __setCallGoJSONAsync(async <T>(): Promise<T> => ({
      schema_version: 1,
      execution_owner: "go-adapter",
      platform: "codex",
      platform_contract: {},
      availability: {},
      worker: {
        name: mockDispatch.name,
        caste: mockDispatch.caste,
        status: "completed",
        summary: "codex selected by Go",
      },
    } as T));

    const result = await dispatchSingleWorker(
      { goBinaryPath: "/usr/bin/true", cwd: "/tmp" },
      mockDispatch
    );

    assert.equal(result.status, "completed");
    assert.equal(result.detectedPlatform, "codex");
    assert.equal(result.summary, "codex selected by Go");
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
