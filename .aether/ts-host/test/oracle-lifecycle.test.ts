/**
 * Unit tests for the Oracle lifecycle orchestrator.
 *
 * Tests verify:
 * - runOracleLifecycle loops until max iterations
 * - runOracleLifecycle stops at confidence target
 * - runOracleLifecycle handles worker failure gracefully
 * - runOracleLifecycle never writes to Go-owned paths
 * - buildOracleWorkerResponse builds correct response from dispatch result
 * - checkStopConditions respects max iterations and confidence target
 */

import { describe, it, beforeEach, afterEach } from "node:test";
import assert from "node:assert/strict";

import type { BuildDispatch } from "../src/types.js";
import type { DispatchResult } from "../src/worker-dispatch.js";
import type { OracleIterationManifest, OracleIterationCompletion } from "../src/types.js";
import {
  runOracleLifecycle,
  buildOracleWorkerResponse,
  checkStopConditions,
  __setCallGoJSON,
  __restoreCallGoJSON,
  __setWriteCompletionFile,
  __restoreWriteCompletionFile,
  __setDispatchSingleWorker,
  __restoreDispatchSingleWorker,
} from "../src/oracle-lifecycle.js";
import type { OracleLifecycleOptions } from "../src/oracle-lifecycle.js";

// ---------------------------------------------------------------------------
// Mock helpers
// ---------------------------------------------------------------------------

let iterateCallCount = 0;
let finalizeCallCount = 0;
let dispatchCallCount = 0;

function resetMocks(): void {
  iterateCallCount = 0;
  finalizeCallCount = 0;
  dispatchCallCount = 0;
}

function makeMockManifest(iteration: number, maxIterations: number, confidenceTarget: number): OracleIterationManifest {
  return {
    ok: true,
    iteration_manifest: {
      topic: "test-topic",
      depth: "balanced",
      max_iterations: maxIterations,
      confidence_target: confidenceTarget,
      current_iteration: iteration,
      workers: [
        {
          name: `Oracle-${String(iteration).padStart(2, "0")}`,
          caste: "oracle",
          task: `Research iteration ${iteration}`,
          brief: `Conduct research iteration ${iteration}/${maxIterations}`,
        },
      ],
    },
  };
}

function makeMockFinalize(shouldContinue: boolean, currentConfidence: number): OracleIterationCompletion {
  return {
    ok: true,
    state_path: ".aether/data/oracle/state.json",
    current_confidence: currentConfidence,
    confidence_target: 85,
    should_continue: shouldContinue,
    next_command: "aether oracle-iterate --plan-only --topic test-topic",
  };
}

function makeMockDispatchResult(status: string): DispatchResult {
  return {
    name: "Oracle-01",
    status: status as "completed" | "failed" | "blocked" | "timeout" | "manually-reconciled" | "code_written",
    summary: `Simulated ${status} result`,
  };
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("oracle-lifecycle", () => {
  beforeEach(() => {
    resetMocks();
  });

  afterEach(() => {
    __restoreCallGoJSON();
    __restoreWriteCompletionFile();
    __restoreDispatchSingleWorker();
  });

  it("runs until max iterations reached", async () => {
    __setCallGoJSON(<T>(_opts: unknown, args: string[]): T => {
      if (args[0] === "oracle-iterate") {
        iterateCallCount++;
        // Stop after iteration 3 (max_iterations=3)
        const manifest = makeMockManifest(iterateCallCount, 3, 85);
        return manifest as unknown as T;
      }
      if (args[0] === "oracle-iterate-finalize") {
        finalizeCallCount++;
        const finalize = makeMockFinalize(iterateCallCount < 3, 50);
        return finalize as unknown as T;
      }
      throw new Error(`Unexpected command: ${args[0]}`);
    });

    __setWriteCompletionFile(() => "/tmp/fake-completion.json");
    __setDispatchSingleWorker(async () => makeMockDispatchResult("completed"));

    const result = await runOracleLifecycle({
      goBinaryPath: "/usr/bin/true",
      cwd: "/tmp",
      topic: "test-topic",
      simulateWorkers: true,
    });

    assert.equal(result.success, true, "Should succeed");
    assert.equal(result.iterations_completed, 3, "Should complete 3 iterations");
    assert.equal(result.stop_reason, "max_iterations_met", "Should stop at max iterations");
    assert.ok(result.steps_completed.includes("iteration-3-manifest"), "Should have iteration 3 manifest step");
  });

  it("stops when confidence target is reached", async () => {
    __setCallGoJSON(<T>(_opts: unknown, args: string[]): T => {
      if (args[0] === "oracle-iterate") {
        iterateCallCount++;
        const manifest = makeMockManifest(iterateCallCount, 10, 85);
        return manifest as unknown as T;
      }
      if (args[0] === "oracle-iterate-finalize") {
        finalizeCallCount++;
        // After iteration 2, confidence reaches target (90 >= 85)
        const confidence = iterateCallCount === 2 ? 90 : 50;
        const shouldContinue = iterateCallCount < 2;
        const finalize = makeMockFinalize(shouldContinue, confidence);
        return finalize as unknown as T;
      }
      throw new Error(`Unexpected command: ${args[0]}`);
    });

    __setWriteCompletionFile(() => "/tmp/fake-completion.json");
    __setDispatchSingleWorker(async () => makeMockDispatchResult("completed"));

    const result = await runOracleLifecycle({
      goBinaryPath: "/usr/bin/true",
      cwd: "/tmp",
      topic: "test-topic",
      simulateWorkers: true,
    });

    assert.equal(result.success, true, "Should succeed");
    assert.equal(iterateCallCount, 2, "Should call iterate twice");
    assert.equal(finalizeCallCount, 2, "Should call finalize twice");
    assert.equal(result.stop_reason, "finalize:should_continue=false", "Should stop when finalize says so");
    assert.equal(result.final_confidence, 90, "Should record final confidence");
  });

  it("handles worker failure gracefully", async () => {
    __setCallGoJSON(<T>(_opts: unknown, args: string[]): T => {
      if (args[0] === "oracle-iterate") {
        iterateCallCount++;
        const manifest = makeMockManifest(iterateCallCount, 5, 85);
        return manifest as unknown as T;
      }
      if (args[0] === "oracle-iterate-finalize") {
        finalizeCallCount++;
        const finalize = makeMockFinalize(iterateCallCount < 5, 30);
        return finalize as unknown as T;
      }
      throw new Error(`Unexpected command: ${args[0]}`);
    });

    __setWriteCompletionFile(() => "/tmp/fake-completion.json");
    __setDispatchSingleWorker(async () => makeMockDispatchResult("failed"));

    const result = await runOracleLifecycle({
      goBinaryPath: "/usr/bin/true",
      cwd: "/tmp",
      topic: "test-topic",
      simulateWorkers: true,
    });

    assert.equal(result.success, true, "Should succeed even with failed workers");
    assert.ok(iterateCallCount > 0, "Should have attempted at least one iteration");
  });

  it("never writes to Go-owned paths", async () => {
    const writtenPaths: string[] = [];

    __setCallGoJSON(<T>(_opts: unknown, args: string[]): T => {
      if (args[0] === "oracle-iterate") {
        iterateCallCount++;
        const manifest = makeMockManifest(iterateCallCount, 1, 85);
        return manifest as unknown as T;
      }
      if (args[0] === "oracle-iterate-finalize") {
        finalizeCallCount++;
        const finalize = makeMockFinalize(false, 70);
        return finalize as unknown as T;
      }
      throw new Error(`Unexpected command: ${args[0]}`);
    });

    __setWriteCompletionFile((_dir: string, _filename: string, _data: unknown) => {
      const path = `/tmp/aether-oracle-${Date.now()}.json`;
      writtenPaths.push(path);
      return path;
    });

    __setDispatchSingleWorker(async () => makeMockDispatchResult("completed"));

    await runOracleLifecycle({
      goBinaryPath: "/usr/bin/true",
      cwd: "/tmp",
      topic: "test-topic",
      simulateWorkers: true,
    });

    for (const path of writtenPaths) {
      assert.ok(
        !path.includes(".aether/data/oracle"),
        `Should not write to .aether/data/oracle: ${path}`
      );
      assert.ok(
        !path.includes(".aether/oracle"),
        `Should not write to .aether/oracle: ${path}`
      );
      assert.ok(
        path.startsWith("/tmp"),
        `Should write to tmpdir: ${path}`
      );
    }
  });

  it("buildOracleWorkerResponse maps completed dispatch correctly", () => {
    const result: DispatchResult = {
      name: "Oracle-01",
      status: "completed",
      summary: "Found patterns in codebase",
      files_modified: ["src/foo.ts"],
    };

    const response = buildOracleWorkerResponse(result, "q-1");

    assert.equal(response.question_id, "q-1");
    assert.equal(response.status, "completed");
    assert.equal(response.confidence, undefined, "TS must not invent Oracle confidence");
    assert.equal(response.summary, "Found patterns in codebase");
    assert.ok(response.findings, "Should have findings");
    const findings = response.findings!;
    assert.equal(findings.length, 1);
    assert.ok(findings[0], "Should have first finding");
    assert.ok(findings[0].evidence, "Should have evidence");
    assert.ok(findings[0].evidence![0], "Should have first evidence item");
    assert.equal(findings[0].evidence![0].title, "src/foo.ts");
  });

  it("buildOracleWorkerResponse maps failed dispatch correctly", () => {
    const result: DispatchResult = {
      name: "Oracle-01",
      status: "failed",
      summary: "Worker timed out",
    };

    const response = buildOracleWorkerResponse(result, "q-2");

    assert.equal(response.question_id, "q-2");
    assert.equal(response.status, "failed");
    assert.equal(response.confidence, undefined, "TS must not invent Oracle confidence");
    assert.equal(response.summary, "Worker timed out");
    assert.equal(response.findings, undefined, "Failed worker should have no findings");
  });

  it("checkStopConditions stops at max iterations", () => {
    const state = makeMockManifest(5, 5, 85).iteration_manifest;
    const check = checkStopConditions(state, { goBinaryPath: "/usr/bin/true", cwd: "/tmp" });

    assert.equal(check.stop, true);
    assert.equal(check.reason, "max_iterations_met");
  });

  it("checkStopConditions allows iteration when below max", () => {
    const state = makeMockManifest(2, 5, 85).iteration_manifest;
    const check = checkStopConditions(state, { goBinaryPath: "/usr/bin/true", cwd: "/tmp" });

    assert.equal(check.stop, false);
    assert.equal(check.reason, "");
  });

  it("checkStopConditions respects maxIterations override", () => {
    const state = makeMockManifest(3, 10, 85).iteration_manifest;
    const check = checkStopConditions(state, { goBinaryPath: "/usr/bin/true", cwd: "/tmp", maxIterations: 3 });

    assert.equal(check.stop, true);
    assert.equal(check.reason, "max_iterations_met");
  });

  it("continues loop when Go manifest indicates resuming", async () => {
    __setCallGoJSON(<T>(_opts: unknown, args: string[]): T => {
      if (args[0] === "oracle-iterate") {
        iterateCallCount++;
        // First call returns resuming=true (interrupted iteration 2)
        const manifest = makeMockManifest(iterateCallCount + 1, 5, 85);
        if (iterateCallCount === 1) {
          manifest.iteration_manifest.resuming = true;
        }
        return manifest as unknown as T;
      }
      if (args[0] === "oracle-iterate-finalize") {
        finalizeCallCount++;
        const finalize = makeMockFinalize(finalizeCallCount < 2, 50);
        return finalize as unknown as T;
      }
      throw new Error(`Unexpected command: ${args[0]}`);
    });

    __setWriteCompletionFile(() => "/tmp/fake-completion.json");
    __setDispatchSingleWorker(async () => makeMockDispatchResult("completed"));

    const result = await runOracleLifecycle({
      goBinaryPath: "/usr/bin/true",
      cwd: "/tmp",
      topic: "test-topic",
      simulateWorkers: true,
    });

    assert.equal(result.success, true, "Should succeed after resume");
    assert.equal(result.iterations_completed, 3, "Should complete resumed iteration and advance");
    assert.equal(iterateCallCount, 2, "Should call iterate twice");
    assert.equal(finalizeCallCount, 2, "Should call finalize twice");
    assert.equal(result.stop_reason, "finalize:should_continue=false", "Should stop when finalize says so");
  });

  it("does not synthesize current confidence in completion payload", async () => {
    let capturedCompletion: Record<string, unknown> | undefined;

    __setCallGoJSON(<T>(_opts: unknown, args: string[]): T => {
      if (args[0] === "oracle-iterate") {
        iterateCallCount++;
        return makeMockManifest(iterateCallCount, 2, 85) as unknown as T;
      }
      if (args[0] === "oracle-iterate-finalize") {
        finalizeCallCount++;
        return makeMockFinalize(false, 42) as unknown as T;
      }
      throw new Error(`Unexpected command: ${args[0]}`);
    });

    __setWriteCompletionFile((_dir: string, _filename: string, data: unknown) => {
      capturedCompletion = data as Record<string, unknown>;
      return "/tmp/fake-oracle-completion.json";
    });
    __setDispatchSingleWorker(async () => makeMockDispatchResult("completed"));

    const result = await runOracleLifecycle({
      goBinaryPath: "/usr/bin/true",
      cwd: "/tmp",
      topic: "test-topic",
      simulateWorkers: true,
    });

    assert.equal(result.success, true, "Should let Go finalizer own confidence");
    assert.equal(result.final_confidence, 42, "Should use Go finalizer confidence");
    assert.ok(capturedCompletion, "Should capture completion payload");
    assert.equal(
      Object.hasOwn(capturedCompletion!, "current_confidence"),
      false,
      "TS completion payload must not include wrapper-synthesized current_confidence"
    );
    const dispatches = capturedCompletion!.dispatches as Array<Record<string, unknown>>;
    assert.equal(
      Object.hasOwn(dispatches[0]!, "confidence_delta"),
      false,
      "TS completion dispatch must not include wrapper-synthesized confidence_delta"
    );
  });

  it("returns a clear blocker when Oracle dispatch throws", async () => {
    __setCallGoJSON(<T>(_opts: unknown, args: string[]): T => {
      if (args[0] === "oracle-iterate") {
        iterateCallCount++;
        return makeMockManifest(iterateCallCount, 2, 85) as unknown as T;
      }
      throw new Error(`Unexpected command: ${args[0]}`);
    });

    __setWriteCompletionFile(() => "/tmp/fake-oracle-completion.json");
    __setDispatchSingleWorker(async () => {
      throw new Error("worker timeout after 1ms");
    });

    const result = await runOracleLifecycle({
      goBinaryPath: "/usr/bin/true",
      cwd: "/tmp",
      topic: "test-topic",
      simulateWorkers: false,
    });

    assert.equal(result.success, false, "Should fail visibly when dispatch fails");
    assert.match(result.error ?? "", /worker timeout after 1ms/);
    assert.equal(result.stop_reason, "error");
  });
});
