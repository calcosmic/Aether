/**
 * Unit tests for the swarm display module.
 *
 * Tests verify:
 * - runSwarmDisplay with dashboard disabled returns success and manifest
 * - runSwarmDisplay handles empty manifest gracefully
 * - renderSwarmText produces non-empty string containing swarm_id and wave groups
 * - renderSwarmFrame produces non-empty compact frame
 */

import { describe, it, beforeEach, afterEach } from "node:test";
import assert from "node:assert/strict";

import {
  runSwarmDisplay,
  renderSwarmText,
  renderSwarmFrame,
  __setCallGoJSON,
  __restoreCallGoJSON,
} from "../src/swarm-display.js";
import type { SwarmManifest, SwarmWorkerPlan } from "../src/types.js";

// ---------------------------------------------------------------------------
// Mock helpers
// ---------------------------------------------------------------------------

function makeMockManifest(): SwarmManifest {
  const dispatches: SwarmWorkerPlan[] = [
    {
      stage: "investigation",
      wave: 1,
      name: "Scout-01",
      caste: "scout",
      role: "scout",
      task: "Map codebase structure",
      agent_name: "aether-scout",
      brief: "Survey the repository for relevant files",
    },
    {
      stage: "investigation",
      wave: 1,
      name: "Tracker-02",
      caste: "tracker",
      role: "tracker",
      task: "Identify root cause",
      agent_name: "aether-tracker",
      brief: "Trace the bug to its origin",
    },
    {
      stage: "fix",
      wave: 2,
      name: "Builder-03",
      caste: "builder",
      role: "builder",
      task: "Implement fix",
      agent_name: "aether-builder",
      brief: "Write the minimal fix",
    },
    {
      stage: "verification",
      wave: 2,
      name: "Watcher-04",
      caste: "watcher",
      role: "watcher",
      task: "Verify fix with tests",
      agent_name: "aether-watcher",
      brief: "Run tests to confirm the fix",
    },
  ];

  return {
    workflow: "swarm",
    dispatch_mode: "plan-only",
    requires_finalizer: true,
    generated_at: new Date().toISOString(),
    root: "/tmp/test-repo",
    swarm_id: "swarm-128-02-test",
    target: "Fix navigation bug in host.ts",
    wave_count: 2,
    worker_count: 4,
    run_timeout_seconds: 300,
    worker_timeout_seconds: 120,
    dispatch_contract: {},
    dispatches,
    execution_plan: [],
    finalizer_command: "aether continue",
  };
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("swarm-display", () => {
  beforeEach(() => {
    __restoreCallGoJSON();
  });

  afterEach(() => {
    __restoreCallGoJSON();
  });

  it("runSwarmDisplay with dashboard disabled returns success and manifest", async () => {
    __setCallGoJSON(<T>(_opts: unknown, args: string[]): T => {
      if (args[0] === "swarm") {
        return makeMockManifest() as unknown as T;
      }
      throw new Error(`Unexpected command: ${args[0]}`);
    });

    const result = await runSwarmDisplay({
      goBinaryPath: "/usr/bin/true",
      cwd: "/tmp",
      dashboard: false,
      planOnly: true,
    });

    assert.equal(result.success, true, "Should succeed");
    assert.ok(result.manifest, "Should have manifest");
    assert.equal(result.manifest!.swarm_id, "swarm-128-02-test", "Should capture swarm_id");
    assert.equal(result.dispatches_count, 4, "Should count dispatches correctly");
  });

  it("runSwarmDisplay handles empty manifest gracefully", async () => {
    const emptyManifest: SwarmManifest = {
      workflow: "swarm",
      dispatch_mode: "plan-only",
      requires_finalizer: false,
      generated_at: new Date().toISOString(),
      root: "/tmp/test-repo",
      swarm_id: "swarm-empty",
      target: "",
      wave_count: 0,
      worker_count: 0,
      run_timeout_seconds: 0,
      worker_timeout_seconds: 0,
      dispatch_contract: {},
      dispatches: [],
      execution_plan: [],
      finalizer_command: "",
    };

    __setCallGoJSON(<T>(_opts: unknown, args: string[]): T => {
      if (args[0] === "swarm") {
        return emptyManifest as unknown as T;
      }
      throw new Error(`Unexpected command: ${args[0]}`);
    });

    const result = await runSwarmDisplay({
      goBinaryPath: "/usr/bin/true",
      cwd: "/tmp",
      dashboard: false,
      planOnly: true,
    });

    assert.equal(result.success, false, "Should fail for empty manifest");
    assert.ok(result.error, "Should have error message");
    assert.ok(result.error!.includes("no dispatches"), "Error should mention no dispatches");
    assert.equal(result.dispatches_count, 0, "Should report zero dispatches");
  });

  it("renderSwarmText produces non-empty string containing swarm_id and wave groups", () => {
    const manifest = makeMockManifest();
    const text = renderSwarmText(manifest);

    assert.ok(text.length > 0, "Text should not be empty");
    assert.ok(text.includes("swarm-128-02-test"), "Text should include swarm_id");
    assert.ok(text.includes("Fix navigation bug in host.ts"), "Text should include target");
    assert.ok(text.includes("Wave 1"), "Text should include wave 1 header");
    assert.ok(text.includes("Wave 2"), "Text should include wave 2 header");
    assert.ok(text.includes("Scout-01"), "Text should include Scout-01 worker");
    assert.ok(text.includes("Builder-03"), "Text should include Builder-03 worker");
    assert.ok(text.includes("aether continue"), "Text should include finalizer command");
  });

  it("renderSwarmFrame produces non-empty compact frame", () => {
    const manifest = makeMockManifest();
    const frame = renderSwarmFrame(manifest);

    assert.ok(frame.length > 0, "Frame should not be empty");
    assert.ok(frame.includes("Aether Swarm"), "Frame should include header");
    assert.ok(frame.includes("swarm-128-02-test"), "Frame should include swarm_id");
    assert.ok(frame.includes("Wave 1:"), "Frame should include wave 1 count");
    assert.ok(frame.includes("Wave 2:"), "Frame should include wave 2 count");
    assert.ok(frame.includes("2 worker(s)"), "Frame should show worker count per wave");
  });

  it("renderSwarmText handles missing target gracefully", () => {
    const manifest = makeMockManifest();
    manifest.target = "";
    const text = renderSwarmText(manifest);
    assert.ok(text.includes("(none)"), "Empty target should render as (none)");
  });

  it("renderSwarmFrame handles long target gracefully", () => {
    const manifest = makeMockManifest();
    manifest.target = "a".repeat(100);
    const frame = renderSwarmFrame(manifest);
    assert.ok(frame.includes("Aether Swarm"), "Frame should still have header");
    assert.ok(!frame.includes("a".repeat(100)), "Long target should be truncated");
  });
});
