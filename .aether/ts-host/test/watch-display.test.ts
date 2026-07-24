/**
 * Unit tests for the watch display module.
 *
 * Tests verify:
 * - runWatchDisplay with dashboard disabled returns success and status
 * - runWatchDisplay handles Go errors gracefully
 * - renderStatusText produces non-empty string containing the goal
 * - renderStatusFrame produces non-empty string with progress bar
 */

import { describe, it, beforeEach, afterEach } from "node:test";
import assert from "node:assert/strict";

import {
  runWatchDisplay,
  renderStatusText,
  renderStatusFrame,
  __setCallGoJSON,
  __restoreCallGoJSON,
} from "../src/watch-display.js";
import type { StatusResult } from "../src/types.js";

// ---------------------------------------------------------------------------
// Mock helpers
// ---------------------------------------------------------------------------

function makeMockStatus(): StatusResult {
  return {
    goal: "Test colony goal",
    state: "executing",
    current_phase: 2,
    total_phases: 5,
    phases_completed: 1,
    phase_name: "Swarm/Watch Host Bridge",
    tasks_completed: 3,
    tasks_total: 8,
    colony_mode: "standard",
    warnings: ["Low pheromone signal strength"],
  };
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("watch-display", () => {
  beforeEach(() => {
    __restoreCallGoJSON();
  });

  afterEach(() => {
    __restoreCallGoJSON();
  });

  it("runWatchDisplay with dashboard disabled returns success and status", async () => {
    let capturedArgs: string[] | undefined;
    __setCallGoJSON(<T>(_opts: unknown, args: string[]): T => {
      capturedArgs = [...args];
      if (args[0] === "status") {
        return makeMockStatus() as unknown as T;
      }
      throw new Error(`Unexpected command: ${args[0]}`);
    });

    const result = await runWatchDisplay({
      goBinaryPath: "/usr/bin/true",
      cwd: "/tmp",
      dashboard: false,
    });

    assert.equal(result.success, true, "Should succeed");
    assert.ok(result.final_status, "Should have final_status");
    assert.equal(result.final_status!.goal, "Test colony goal", "Should capture goal");
    assert.equal(result.snapshots_count, 1, "Should take one snapshot in plain-text mode");
    assert.deepEqual(capturedArgs, ["status"], "Watch display must consume Go status JSON directly");
  });

  it("runWatchDisplay handles Go errors gracefully", async () => {
    __setCallGoJSON(() => {
      throw new Error("Go status command failed: no colony initialized");
    });

    const result = await runWatchDisplay({
      goBinaryPath: "/usr/bin/true",
      cwd: "/tmp",
      dashboard: false,
    });

    assert.equal(result.success, false, "Should fail when Go errors");
    assert.ok(result.error, "Should have error message");
    assert.ok(result.error!.includes("Go status command failed"), "Error should mention Go failure");
    assert.equal(result.snapshots_count, 0, "Should have zero snapshots on error");
  });

  it("renderStatusText produces non-empty string containing the goal", () => {
    const status = makeMockStatus();
    const text = renderStatusText(status);

    assert.ok(text.length > 0, "Text should not be empty");
    assert.ok(text.includes("Test colony goal"), "Text should include goal");
    assert.ok(text.includes("executing"), "Text should include state");
    assert.ok(text.includes("Swarm/Watch Host Bridge"), "Text should include phase name");
    assert.ok(text.includes("3/8"), "Text should include task progress");
    assert.ok(text.includes("Low pheromone signal strength"), "Text should include warnings");
  });

  it("renderStatusFrame produces non-empty string with progress bar", () => {
    const status = makeMockStatus();
    const frame = renderStatusFrame(status);

    assert.ok(frame.length > 0, "Frame should not be empty");
    assert.ok(frame.includes("Aether Watch"), "Frame should include header");
    assert.ok(frame.includes("█"), "Frame should include filled progress bar chars");
    assert.ok(frame.includes("░"), "Frame should include empty progress bar chars");
    assert.ok(frame.includes("executing"), "Frame should include state");
    assert.ok(frame.includes("Test colony goal"), "Frame should include goal");
    assert.ok(frame.includes("1 warning(s)"), "Frame should include warning count");
  });

  it("renderStatusText handles empty goal gracefully", () => {
    const status: StatusResult = {
      goal: "",
      state: "planning",
      current_phase: 0,
      total_phases: 0,
      phases_completed: 0,
      phase_name: "",
      tasks_completed: 0,
      tasks_total: 0,
    };

    const text = renderStatusText(status);
    assert.ok(text.includes("(none)"), "Empty goal should render as (none)");
    assert.ok(text.includes("planning"), "State should still render");
  });

  it("renderStatusFrame handles zero totals gracefully", () => {
    const status: StatusResult = {
      goal: "",
      state: "planning",
      current_phase: 0,
      total_phases: 0,
      phases_completed: 0,
      phase_name: "",
      tasks_completed: 0,
      tasks_total: 0,
    };

    const frame = renderStatusFrame(status);
    assert.ok(frame.includes("Aether Watch"), "Frame should still have header");
    assert.ok(frame.includes("planning"), "State should still render");
  });
});
