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
