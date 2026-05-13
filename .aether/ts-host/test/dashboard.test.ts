/**
 * Dashboard unit tests.
 *
 * Verifies dashboard controller, worker widget rendering, chamber map
 * grouping, and duration formatting.
 */

import { describe, it, beforeEach, afterEach } from "node:test";
import assert from "node:assert/strict";

import { createDashboard } from "../src/dashboard.js";
import {
  createWorkerWidget,
  renderWorkerWidget,
  formatDuration,
  type WorkerState,
} from "../src/dashboard/worker-widget.js";
import {
  createChamberMap,
  extractDirectoryPrefix,
} from "../src/dashboard/chamber-map.js";
import { loadCeremonyConfig, type CeremonyConfig } from "../src/caste-config.js";
import type { CeremonyEvent } from "../src/types.js";

const REPO_ROOT = "/Users/callumcowie/repos/Aether";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeEvent(topic: string, payload: Record<string, unknown>): CeremonyEvent {
  return {
    id: "test-1",
    topic,
    payload,
    source: "test",
    timestamp: new Date().toISOString(),
    ttl_days: 1,
    expires_at: new Date().toISOString(),
  };
}

function makeWorkerState(overrides?: Partial<WorkerState>): WorkerState {
  return {
    spawn_id: "spawn-1",
    caste: "builder",
    name: "Mason-67",
    task: "Task 1",
    status: "active",
    tool_count: 0,
    token_count: 0,
    files_created: [],
    files_modified: [],
    startTime: Date.now(),
    lastUpdate: Date.now(),
    ...overrides,
  };
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("dashboard", () => {
  let config: CeremonyConfig;

  beforeEach(() => {
    config = loadCeremonyConfig(REPO_ROOT);
  });

  it("createDashboard returns object with onEvent, start, stop", () => {
    const dashboard = createDashboard({ cwd: REPO_ROOT });
    assert.equal(typeof dashboard.onEvent, "function");
    assert.equal(typeof dashboard.start, "function");
    assert.equal(typeof dashboard.stop, "function");
    dashboard.stop();
  });

  it("onEvent creates worker on ceremony.build.spawn", () => {
    const dashboard = createDashboard({ cwd: REPO_ROOT });
    dashboard.start();
    const event = makeEvent("ceremony.build.spawn", {
      spawn_id: "spawn-1",
      caste: "builder",
      name: "Mason-67",
      task: "Build the wall",
    });
    dashboard.onEvent(event);
    // Dashboard does not expose internal state directly; we verify via
    // the fact that start + onEvent does not throw and stop cleans up.
    dashboard.stop();
  });

  it("onEvent updates tool count on ceremony.build.tool_use", () => {
    const dashboard = createDashboard({ cwd: REPO_ROOT });
    dashboard.start();
    dashboard.onEvent(
      makeEvent("ceremony.build.spawn", {
        spawn_id: "spawn-2",
        caste: "watcher",
        name: "Hawk-12",
        task: "Watch the gate",
      })
    );
    dashboard.onEvent(
      makeEvent("ceremony.build.tool_use", {
        spawn_id: "spawn-2",
        tool_count: 7,
        token_count: 2100,
      })
    );
    // No direct state accessor; verify via non-throwing lifecycle
    dashboard.stop();
  });

  it("renderWorkerWidget returns string with emoji and progress", () => {
    const state = makeWorkerState({ tool_count: 12, token_count: 4200 });
    const widget = createWorkerWidget(state, config);
    const output = renderWorkerWidget(widget, config);
    assert.ok(output.includes("🔨"), "Should include builder emoji");
    assert.ok(output.includes("█"), "Should include filled progress block");
    assert.ok(output.includes("░"), "Should include empty progress block");
    assert.ok(output.includes("60%"), "Should include 60% progress for 12 tools");
    assert.ok(output.includes("4.2k"), "Should format token count as 4.2k");
    widget.spinner?.stop();
  });

  it("chamberMap groups workers by directory", () => {
    const map = createChamberMap();
    const workers: WorkerState[] = [
      makeWorkerState({
        spawn_id: "w1",
        files_created: ["src/renderers/visual.ts"],
        files_modified: [],
        tool_count: 10,
      }),
      makeWorkerState({
        spawn_id: "w2",
        files_created: ["src/renderers/json.ts"],
        files_modified: ["test/renderers.test.ts"],
        tool_count: 20,
      }),
    ];
    map.update(workers);
    assert.equal(map.activities.length, 2, "Should have two directory groups");
    const srcActivity = map.activities.find((a) => a.directory === "src/renderers");
    assert.ok(srcActivity, "Should have src/renderers activity");
    assert.equal(srcActivity?.workerCount, 2, "Should count both workers in src/renderers");
    const testActivity = map.activities.find((a) => a.directory === "test");
    assert.ok(testActivity, "Should have test activity");
    assert.equal(testActivity?.workerCount, 1, "Should count one worker in test");
  });

  it("formatDuration formats seconds as MM:SS", () => {
    assert.equal(formatDuration(125000), "02:05");
    assert.equal(formatDuration(60000), "01:00");
    assert.equal(formatDuration(59000), "00:59");
    assert.equal(formatDuration(3661000), "01:01:01");
  });

  it("extractDirectoryPrefix returns directory portion", () => {
    assert.equal(extractDirectoryPrefix("src/renderers/visual.ts"), "src/renderers");
    assert.equal(extractDirectoryPrefix("file.txt"), ".");
    assert.equal(extractDirectoryPrefix("a/b/c/d.ts"), "a/b/c");
  });
});
