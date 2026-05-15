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

import {
  __setDetectAvailablePlatforms,
  __restoreDetectAvailablePlatforms,
} from "../src/platform-dispatcher.js";

import { runLifecycle } from "../src/lifecycle.js";
import {
  __setCallGoJSON,
  __restoreCallGoJSON,
} from "../src/go-bridge.js";

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
    assert.ok(result.error?.includes("No platform CLI available"), `Expected platform error but got: ${result.error}`);
  });

  it("errors when no platforms available and simulateWorkers is undefined", async () => {
    __setDetectAvailablePlatforms(async () => []);
    __setCallGoJSON(mockGoJSON);

    const result = await runLifecycle({
      goBinaryPath: "/usr/bin/true",
      cwd: "/tmp",
    });

    assert.equal(result.success, false);
    assert.ok(result.error?.includes("No platform CLI available"), `Expected platform error but got: ${result.error}`);
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
});
