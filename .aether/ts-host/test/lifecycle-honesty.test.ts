/**
 * Lifecycle honesty tests — verify lifecycle is a simulate-only smoke harness.
 *
 * Tests verify:
 * - Default behavior rejects production-style execution
 * - Simulation only happens with explicit simulateWorkers=true
 * - Placeholder file creation is gated behind --simulate
 */

import { describe, it, beforeEach, afterEach } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { tmpdir } from "node:os";

import { runLifecycle } from "../src/lifecycle.js";
import {
  dispatchSingleWorker,
} from "../src/worker-dispatch.js";
import type { BuildDispatch } from "../src/types.js";
import {
  __setCallGoJSON,
  __restoreCallGoJSON,
  __setCallGoJSONAsync,
  __restoreCallGoJSONAsync,
} from "../src/go-bridge.js";

function completionFileArg(args: string[]): string {
  const completionFileIndex = args.indexOf("--completion-file");
  assert.notEqual(
    completionFileIndex,
    -1,
    `Expected --completion-file in args: ${args.join(" ")}`
  );
  const completionPath = args[completionFileIndex + 1];
  if (typeof completionPath !== "string" || completionPath.trim() === "") {
    throw new Error(`Missing completion file path in args: ${args.join(" ")}`);
  }
  return completionPath;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function readCompletionResult(path: string): Record<string, unknown> {
  const envelope = JSON.parse(readFileSync(path, "utf-8")) as unknown;
  assert.ok(isRecord(envelope), "Completion file should contain an object");
  const result = envelope["result"];
  assert.ok(isRecord(result), "Completion file should wrap the finalizer result");
  return result;
}

describe("lifecycle honesty", { concurrency: false }, () => {
  afterEach(() => {
    __restoreCallGoJSON();
    __restoreCallGoJSONAsync();
  });

  const mockGoJSON = <T>(_opts: unknown, args: string[]): T => {
    if (args[0] === "plan") {
      return {
        ok: true,
        plan_manifest: {
          phase: 1,
          name: "Test Phase",
          tasks: [{ id: "t1", name: "Task 1", status: "pending" }],
        },
      } as unknown as T;
    }
    if (args[0] === "build") {
      return {
        ok: true,
        dispatch_manifest: {
          phase: 1,
          dispatches: [
            { name: "Builder-01", caste: "builder", task: "Build task 1", wave: 1 },
          ],
        },
      } as unknown as T;
    }
    if (args[0] === "continue") {
      return {
        ok: true,
        continue_manifest: {
          phase: 1,
          status: "completed",
        },
      } as unknown as T;
    }
    if (args[0] === "plan-finalize") {
      return {
        ok: true,
        planned: false,
        iteration_completed: true,
        requires_next_iteration: true,
      } as unknown as T;
    }
    if (args[0] === "build-finalize" || args[0] === "continue-finalize") {
      return { ok: true } as unknown as T;
    }
    throw new Error(`Unexpected: ${args[0]}`);
  };

  it("rejects simulateWorkers=false before any production-style lifecycle work", async () => {
    let goCalled = false;
    __setCallGoJSON(<T>(opts: unknown, args: string[]): T => {
      goCalled = true;
      return mockGoJSON<T>(opts, args);
    });

    const result = await runLifecycle({
      goBinaryPath: "/usr/bin/true",
      cwd: "/tmp",
      simulateWorkers: false,
    });

    assert.equal(result.success, false);
    assert.ok(result.error?.includes("simulate-only"), `Expected simulate-only error but got: ${result.error}`);
    assert.equal(goCalled, false, "Lifecycle rejection should not request Go manifests");
  });

  it("rejects simulateWorkers undefined before any production-style lifecycle work", async () => {
    __setCallGoJSON(mockGoJSON);

    const result = await runLifecycle({
      goBinaryPath: "/usr/bin/true",
      cwd: "/tmp",
    });

    assert.equal(result.success, false);
    assert.ok(result.error?.includes("simulate-only"), `Expected simulate-only error but got: ${result.error}`);
  });

  it("stops with simulateWorkers=true when planning still needs real worker loop", async () => {
    let buildCalled = false;
    __setCallGoJSON(<T>(opts: unknown, args: string[]): T => {
      if (args[0] === "build") {
        buildCalled = true;
      }
      return mockGoJSON<T>(opts, args);
    });

    const result = await runLifecycle({
      goBinaryPath: "/usr/bin/true",
      cwd: "/tmp",
      simulateWorkers: true,
    });

    assert.equal(buildCalled, false, "Build step should not be reached before planning finishes");
    assert.equal(result.success, false);
    assert.ok(
      /intermediate planning iteration|synthesis planning packets|real Scout and Route-Setter/.test(result.error ?? ""),
      `Expected pending planning error but got: ${result.error ?? "unknown"}`
    );
  });

  it("rejects lifecycle without explicit simulation before provider preflight", async () => {
    let buildCalled = false;
    __setCallGoJSON(<T>(_opts: unknown, args: string[]): T => {
      if (args[0] === "build") {
        buildCalled = true;
      }
      return mockGoJSON<T>(_opts, args);
    });

    const result = await runLifecycle({
      goBinaryPath: "/usr/bin/true",
      cwd: "/tmp",
    });

    assert.equal(buildCalled, false, "Build step should not be reached");
    assert.equal(result.success, false);
    assert.ok(result.error?.includes("simulate-only"), `Expected simulate-only error but got: ${result.error}`);
  });

  it("does not send completed plan dispatches to plan-finalize without real worker results", async () => {
    let capturedCompletionPath = "";
    let capturedCompletion: Record<string, unknown> | undefined;

    __setCallGoJSON(<T>(_opts: unknown, args: string[]): T => {
      if (args[0] === "plan") {
        return {
          plan_manifest: {
            phase: 0,
            goal: "Test honest planning",
          },
          dispatches: [
            {
              name: "Route-Setter-01",
              caste: "route-setter",
              stage: "plan",
              task: "Create the phase plan",
              task_id: "plan-1",
              wave: 1,
              execution_wave: 1,
            },
          ],
        } as unknown as T;
      }
      if (args[0] === "plan-finalize") {
        capturedCompletionPath = completionFileArg(args);
        capturedCompletion = readCompletionResult(capturedCompletionPath);
        throw new Error("stop after plan-finalize capture");
      }
      throw new Error(`Unexpected command before plan-finalize: ${args[0]}`);
    });

    const result = await runLifecycle({
      goBinaryPath: "/usr/bin/true",
      cwd: "/tmp",
      simulateWorkers: true,
      dashboard: false,
    });

    assert.equal(result.success, false);
    assert.ok(
      result.error?.includes("stop after plan-finalize capture"),
      `Expected capture sentinel, got: ${result.error}`
    );
    assert.ok(
      capturedCompletionPath.startsWith(tmpdir()),
      `Plan completion file should be temporary: ${capturedCompletionPath}`
    );
    assert.ok(
      !capturedCompletionPath.includes(".aether/data"),
      `Plan completion file should not be under .aether/data: ${capturedCompletionPath}`
    );
    assert.ok(capturedCompletion, "Expected to capture plan-finalize completion");

    const dispatches = capturedCompletion["dispatches"];
    assert.ok(Array.isArray(dispatches), "Plan completion should include dispatches");
    const completedDispatches = dispatches.filter(
      (dispatch): dispatch is Record<string, unknown> =>
        isRecord(dispatch) && dispatch["status"] === "completed"
    );

    assert.deepEqual(
      completedDispatches,
      [],
      "runLifecycle must not claim plan dispatches completed without real worker results"
    );
  });

  it("labels host-synthesized planning artifacts instead of relying on ambiguous top-level phase_plan evidence", async () => {
    let capturedCompletion: Record<string, unknown> | undefined;

    __setCallGoJSON(<T>(_opts: unknown, args: string[]): T => {
      if (args[0] === "plan") {
        return {
          plan_manifest: {
            phase: 0,
            goal: "Test explicit synthesis",
          },
          dispatches: [
            {
              name: "Route-Setter-02",
              caste: "route-setter",
              stage: "plan",
              task: "Create the phase plan",
            },
          ],
        } as unknown as T;
      }
      if (args[0] === "plan-finalize") {
        capturedCompletion = readCompletionResult(completionFileArg(args));
        throw new Error("stop after plan-finalize capture");
      }
      throw new Error(`Unexpected command before plan-finalize: ${args[0]}`);
    });

    const result = await runLifecycle({
      goBinaryPath: "/usr/bin/true",
      cwd: "/tmp",
      simulateWorkers: true,
      dashboard: false,
    });

    assert.equal(result.success, false);
    assert.ok(
      result.error?.includes("stop after plan-finalize capture"),
      `Expected capture sentinel, got: ${result.error}`
    );
    assert.ok(capturedCompletion, "Expected to capture plan-finalize completion");

    const synthesis = capturedCompletion["synthesis"];
    assert.ok(
      isRecord(synthesis),
      "Host-created phase plans must be identified with an explicit synthesis envelope"
    );
    assert.equal(
      synthesis["source"],
      "ts-host",
      "Host-created phase plans must identify ts-host as the synthesis source"
    );
    assert.equal(
      typeof synthesis["reason"],
      "string",
      "Host-created phase plans must explain why synthesis was used"
    );
  });
});

describe("worker-dispatch honesty", { concurrency: false }, () => {
  afterEach(() => {
    __restoreCallGoJSON();
    __restoreCallGoJSONAsync();
  });

  const mockDispatch: BuildDispatch = {
    stage: "build",
    caste: "builder",
    name: "Builder-01",
    task: "Implement feature",
    status: "pending",
  };

  it("defaults to real execution when simulateWorkers is undefined", async () => {
    __setCallGoJSON(<T>(): T => ({ recorded: true } as unknown as T));
    __setCallGoJSONAsync(async <T>(): Promise<T> => {
      throw new Error("no worker provider available");
    });

    const result = await dispatchSingleWorker(
      {
        goBinaryPath: "/usr/bin/true",
        cwd: "/tmp",
      } as import("../src/worker-dispatch.js").DispatchOptions,
      mockDispatch
    );
    assert.equal(result.status, "failed");
    assert.match(result.summary, /no worker provider available/);
  });

  it("delegates detailed provider diagnostics to Go when no platforms are available", async () => {
    __setCallGoJSON(<T>(): T => ({ recorded: true } as unknown as T));
    __setCallGoJSONAsync(async <T>(): Promise<T> => {
      throw new Error("Go AvailabilityStatus contract: credentials missing");
    });

    await assert.rejects(
      async () =>
        dispatchSingleWorker(
          {
            goBinaryPath: "/usr/bin/true",
            cwd: "/tmp",
          } as import("../src/worker-dispatch.js").DispatchOptions,
          mockDispatch
        ),
      /Go AvailabilityStatus contract/
    );
  });

  it("simulates only with explicit simulateWorkers=true", async () => {
    __setCallGoJSON(<T>(): T => ({ recorded: true } as unknown as T));
    __setCallGoJSONAsync(async <T>(): Promise<T> => {
      throw new Error("real adapter must not be called");
    });

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
  });

  it("returns a failed result with explicit simulateWorkers=false and no platforms", async () => {
    __setCallGoJSON(<T>(): T => ({ recorded: true } as unknown as T));
    __setCallGoJSONAsync(async <T>(): Promise<T> => {
      throw new Error("no worker provider available");
    });

    const result = await dispatchSingleWorker(
      {
        goBinaryPath: "/usr/bin/true",
        cwd: "/tmp",
        simulateWorkers: false,
      },
      mockDispatch
    );
    assert.equal(result.status, "failed");
    assert.match(result.summary, /no worker provider available/);
  });
});
