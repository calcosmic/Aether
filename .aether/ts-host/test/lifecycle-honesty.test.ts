/**
 * Lifecycle honesty tests — verify simulation is opt-in only.
 *
 * Tests verify:
 * - Default behavior is real execution (simulateWorkers=false)
 * - Simulation only happens with explicit simulateWorkers=true
 * - Missing platforms without --simulate produces an error
 * - Placeholder file creation is gated behind --simulate
 */

import { describe, it, beforeEach, afterEach } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { tmpdir } from "node:os";

import {
  __setDetectAvailablePlatforms,
  __restoreDetectAvailablePlatforms,
} from "../src/platform-dispatcher.js";

import { runLifecycle } from "../src/lifecycle.js";
import {
  dispatchSingleWorker,
} from "../src/worker-dispatch.js";
import type { BuildDispatch } from "../src/types.js";
import {
  __setCallGoJSON,
  __restoreCallGoJSON,
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
    __restoreDetectAvailablePlatforms();
    __restoreCallGoJSON();
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
    if (args[0] === "plan-finalize" || args[0] === "build-finalize" || args[0] === "continue-finalize") {
      return { ok: true } as unknown as T;
    }
    throw new Error(`Unexpected: ${args[0]}`);
  };

  it("errors when no platforms available and simulateWorkers is false", async () => {
    __setDetectAvailablePlatforms(async () => []);
    __setCallGoJSON(mockGoJSON);

    const result = await runLifecycle({
      goBinaryPath: "/usr/bin/true",
      cwd: "/tmp",
      simulateWorkers: false,
    });

    assert.equal(result.success, false);
    assert.ok(result.error?.includes("Worker dispatch cannot start"), `Expected platform error but got: ${result.error}`);
    assert.ok(result.error?.includes("Go AvailabilityStatus contract"), `Expected Go diagnostic delegation but got: ${result.error}`);
  });

  it("errors when no platforms available and simulateWorkers is undefined", async () => {
    __setDetectAvailablePlatforms(async () => []);
    __setCallGoJSON(mockGoJSON);

    const result = await runLifecycle({
      goBinaryPath: "/usr/bin/true",
      cwd: "/tmp",
    });

    assert.equal(result.success, false);
    assert.ok(result.error?.includes("Worker dispatch cannot start"), `Expected platform error but got: ${result.error}`);
    assert.ok(result.error?.includes("provider_diagnostics"), `Expected Go diagnostic field guidance but got: ${result.error}`);
  });

  it("uses Go provider diagnostics from the build result when platform preflight fails", async () => {
    __setDetectAvailablePlatforms(async () => []);
    __setCallGoJSON(<T>(opts: unknown, args: string[]): T => {
      if (args[0] === "build") {
        return {
          ok: true,
          provider_diagnostics: "GO-OWNED: codex provider runtime-owned detail. Next: sign in with codex.",
          dispatch_manifest: {
            phase: 1,
            provider_diagnostics: "GO-OWNED: manifest diagnostic should be secondary.",
            dispatches: [
              { name: "Builder-01", caste: "builder", task: "Build task 1", wave: 1 },
            ],
          },
        } as unknown as T;
      }
      return mockGoJSON<T>(opts, args);
    });

    const result = await runLifecycle({
      goBinaryPath: "/usr/bin/true",
      cwd: "/tmp",
      simulateWorkers: false,
    });

    assert.equal(result.success, false);
    assert.ok(
      result.error?.includes("GO-OWNED: codex provider runtime-owned detail"),
      `Expected Go-owned diagnostic, got: ${result.error}`
    );
    assert.ok(
      !result.error?.includes("No platform CLI available"),
      `TS host should not invent provider diagnostics: ${result.error}`
    );
    assert.ok(
      !result.error?.includes("Install or authenticate"),
      `TS host should not invent provider next actions: ${result.error}`
    );
  });

  it("succeeds with simulateWorkers=true even when no platforms available", async () => {
    __setDetectAvailablePlatforms(async () => []);
    __setCallGoJSON(mockGoJSON);

    const result = await runLifecycle({
      goBinaryPath: "/usr/bin/true",
      cwd: "/tmp",
      simulateWorkers: true,
    });

    assert.equal(result.success, true, `Expected success but got error: ${result.error ?? "unknown"}`);
  });

  it("defaults to real execution when platforms are available", async () => {
    __setDetectAvailablePlatforms(async () => ["claude"]);
    let buildCalled = false;
    __setCallGoJSON(<T>(_opts: unknown, args: string[]): T => {
      if (args[0] === "build") {
        buildCalled = true;
      }
      return mockGoJSON<T>(_opts, args);
    });

    // Should reach the build step (proving real execution was attempted)
    const result = await runLifecycle({
      goBinaryPath: "/usr/bin/true",
      cwd: "/tmp",
    });

    assert.equal(buildCalled, true, "Build step should have been reached");
    assert.equal(result.success, true, "Should succeed with platforms available");
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
    __restoreDetectAvailablePlatforms();
    __restoreCallGoJSON();
  });

  const mockDispatch: BuildDispatch = {
    stage: "build",
    caste: "builder",
    name: "Builder-01",
    task: "Implement feature",
    status: "pending",
  };

  it("defaults to real execution when simulateWorkers is undefined", async () => {
    __setDetectAvailablePlatforms(async () => []);
    __setCallGoJSON(<T>(): T => ({ recorded: true } as unknown as T));

    await assert.rejects(
      async () =>
        dispatchSingleWorker(
          {
            goBinaryPath: "/usr/bin/true",
            cwd: "/tmp",
          } as import("../src/worker-dispatch.js").DispatchOptions,
          mockDispatch
        ),
      /Worker dispatch cannot start/
    );
  });

  it("delegates detailed provider diagnostics to Go when no platforms are available", async () => {
    __setDetectAvailablePlatforms(async () => []);
    __setCallGoJSON(<T>(): T => ({ recorded: true } as unknown as T));

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
    __setDetectAvailablePlatforms(async () => []);
    __setCallGoJSON(<T>(): T => ({ recorded: true } as unknown as T));

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

  it("throws error with explicit simulateWorkers=false and no platforms", async () => {
    __setDetectAvailablePlatforms(async () => []);
    __setCallGoJSON(<T>(): T => ({ recorded: true } as unknown as T));

    await assert.rejects(
      async () =>
        dispatchSingleWorker(
          {
            goBinaryPath: "/usr/bin/true",
            cwd: "/tmp",
            simulateWorkers: false,
          },
          mockDispatch
        ),
      /Worker dispatch cannot start/
    );
  });
});
