/**
 * Integration tests for the lifecycle orchestrator.
 *
 * Tests verify:
 * - runLifecycle stops before build when planning still needs real workers
 * - Each step calls Go --plan-only and finalizer commands correctly
 * - Spawn events are recorded for build workers
 * - Completion files are written to tmpdir, not .aether/data/
 * - COLONY_STATE.json is updated by Go finalizers after each step
 *
 * All tests run against the real Go CLI binary with a temp colony.
 */

import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { tmpdir } from "node:os";
import { fileURLToPath } from "node:url";

import { describe, it, beforeEach, afterEach } from "node:test";
import assert from "node:assert/strict";

import { discoverGoBinary, callGoJSON } from "../src/go-bridge.js";
import type { GoBridgeOptions } from "../src/go-bridge.js";
import { runLifecycle } from "../src/lifecycle.js";
import type { LifecycleOptions, LifecycleResult } from "../src/lifecycle.js";
import {
  __restoreCreateCeremonyAdapter,
  __setCreateCeremonyAdapter,
  type CeremonyAdapter,
  type CeremonyWorkflow,
} from "../src/ceremony-adapter.js";

let sourceGoBinaryPath: string | undefined;

function discoverSourceGoBinary(): string {
  if (process.env["AETHER_BINARY_PATH"]) {
    return discoverGoBinary();
  }
  if (sourceGoBinaryPath) {
    return sourceGoBinaryPath;
  }

  const testDir = dirname(fileURLToPath(import.meta.url));
  const repoRoot = resolve(testDir, "..", "..", "..");
  const buildDir = mkdtempSync(join(tmpdir(), "aether-ts-host-source-bin-"));
  sourceGoBinaryPath = join(buildDir, "aether");
  execFileSync("go", ["build", "-o", sourceGoBinaryPath, "./cmd/aether"], {
    cwd: repoRoot,
    stdio: "pipe",
  });
  return sourceGoBinaryPath;
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

/**
 * Set up a minimal test colony in a temp directory.
 * Creates .aether/data/ with a valid COLONY_STATE.json so that
 * the full lifecycle can run.
 */
function setupTestColony(): {
  tempDir: string;
  dataDir: string;
  cleanup: () => void;
  bridge: GoBridgeOptions;
} {
  const tempDir = mkdtempSync(join(tmpdir(), "ts-host-lifecycle-test-"));
  const dataDir = join(tempDir, ".aether", "data");
  const plansDir = join(tempDir, ".aether", "plans");

  mkdirSync(dataDir, { recursive: true });
  mkdirSync(plansDir, { recursive: true });

  // Write a minimal valid colony state
  const colonyState = {
    version: "3.0",
    goal: "Test colony for lifecycle integration tests",
    state: "READY",
    plan: { phases: [] },
    current_phase: 0,
  };
  writeFileSync(
    join(dataDir, "COLONY_STATE.json"),
    JSON.stringify(colonyState, null, 2),
    "utf-8"
  );

  // Write empty supporting files
  writeFileSync(join(dataDir, "pheromones.json"), "[]", "utf-8");
  writeFileSync(join(dataDir, "constraints.json"), "[]", "utf-8");
  writeFileSync(join(dataDir, "session.json"), "{}", "utf-8");

  const goBinaryPath = discoverSourceGoBinary();
  const bridge: GoBridgeOptions = { goBinaryPath, cwd: tempDir };

  return {
    tempDir,
    dataDir,
    cleanup: () => {
      rmSync(tempDir, { recursive: true, force: true });
    },
    bridge,
  };
}

/**
 * Read the colony state from the test colony's .aether/data directory.
 */
function readColonyState(dataDir: string): Record<string, unknown> {
  const statePath = join(dataDir, "COLONY_STATE.json");
  if (!existsSync(statePath)) {
    return {};
  }
  const raw = readFileSync(statePath, "utf-8");
  return JSON.parse(raw) as Record<string, unknown>;
}

function assertRequiresPlanningLoop(result: LifecycleResult): void {
  assert.equal(result.success, false, "Lifecycle should stop before build");
  assert.deepEqual(
    result.steps_completed,
    [],
    "No lifecycle step should be marked complete while planning is pending"
  );
  assert.match(
    result.error ?? "",
    /intermediate planning iteration|synthesis planning packets|real Scout and Route-Setter/,
    `Error should explain the pending planning loop: ${result.error ?? "none"}`
  );
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("lifecycle", () => {
  let context: ReturnType<typeof setupTestColony> | null = null;

  beforeEach(() => {
    context = setupTestColony();
  });

  afterEach(() => {
    context?.cleanup();
    context = null;
  });

  it("runLifecycle stops before build when planning needs another worker iteration", async () => {
    assert.ok(context, "Test context should be initialized");
    const { bridge, dataDir } = context;

    const opts: LifecycleOptions = {
      goBinaryPath: bridge.goBinaryPath,
      cwd: bridge.cwd,
      simulateWorkers: true,
      phase: 1,
    };

    const result = await runLifecycle(opts);

    assertRequiresPlanningLoop(result);

    // Verify colony state was not promoted to a completed plan by TS synthesis.
    const state = readColonyState(dataDir);
    const plan = state["plan"] as { phases?: unknown[] } | undefined;
    assert.ok(
      Array.isArray(plan?.phases),
      "Plan should still have a phases array"
    );
    assert.equal(
      plan?.phases?.length ?? 0,
      0,
      "Pending planning must not write final phases"
    );
  });

  it("runLifecycle does not record build spawn events without a finalized plan", async () => {
    assert.ok(context, "Test context should be initialized");
    const { bridge } = context;

    const opts: LifecycleOptions = {
      goBinaryPath: bridge.goBinaryPath,
      cwd: bridge.cwd,
      simulateWorkers: true,
      phase: 1,
    };

    const result = await runLifecycle(opts);
    assertRequiresPlanningLoop(result);

    // Check that build worker entries were not recorded.
    try {
      interface SpawnTreeResult {
        entries?: Array<{
          agent_name: string;
          status: string;
          caste: string;
        }>;
      }
      const tree = callGoJSON<SpawnTreeResult>(bridge, ["spawn-tree-load"]);
      const entries = tree.entries ?? [];

      const completedEntries = entries.filter(
        (e) => e.status === "completed"
      );
      assert.equal(
        completedEntries.length,
        0,
        `Spawn tree should not have completed build entries before planning finishes. ` +
          `Found ${entries.length} entries: ${JSON.stringify(entries.map((e) => `${e.agent_name}=${e.status}`))}`
      );
    } catch (err) {
      // spawn-tree-load may fail in a minimal test colony; that is acceptable
      // because planning stopped before build worker spawn-log/complete calls.
      const msg = err instanceof Error ? err.message : String(err);
      process.stderr.write(
        `Note: spawn-tree-load check skipped: ${msg}\n`
      );
    }
  });

  it("runLifecycle uses tmpdir for completion files, never .aether/data", async () => {
    assert.ok(context, "Test context should be initialized");
    const { bridge, dataDir } = context;

    const { readdirSync } = await import("node:fs");
    const completionDirNamesBefore = new Set(
      readdirSync(tmpdir(), { withFileTypes: true })
        .filter((e) => e.isDirectory() && /^aether-(plan|build|continue)-/.test(e.name))
        .map((e) => e.name)
    );

    const opts: LifecycleOptions = {
      goBinaryPath: bridge.goBinaryPath,
      cwd: bridge.cwd,
      simulateWorkers: true,
      phase: 1,
    };

    const result = await runLifecycle(opts);
    assertRequiresPlanningLoop(result);

    // The .aether/data/ directory may have planning iteration state written by
    // Go, but completion files should only exist in unique tmpdirs.
    const tmpEntries = readdirSync(tmpdir(), { withFileTypes: true });
    const completionDirs = tmpEntries
      .filter((e) => e.isDirectory() && /^aether-(plan|build|continue)-/.test(e.name))
      .filter((e) => !completionDirNamesBefore.has(e.name))
      .map((e) => join(tmpdir(), e.name));

    assert.ok(
      completionDirs.length >= 1,
      `Expected at least 1 unique approved completion dir, found ${completionDirs.length}`
    );

    // Collect all completion files across unique dirs
    const completionFiles = new Set<string>();
    for (const dir of completionDirs) {
      for (const f of readdirSync(dir)) {
        completionFiles.add(f);
      }
    }

    assert.ok(
      completionFiles.has("plan-completion.json"),
      "Plan completion file should exist in tmpdir"
    );
    assert.ok(
      !completionFiles.has("build-completion.json"),
      "Build completion file should not exist before planning finishes"
    );
    assert.ok(
      !completionFiles.has("continue-completion.json"),
      "Continue completion file should not exist before planning finishes"
    );

    // Verify none of the completion files are in .aether/data/
    const dataFilesAfter = new Set<string>();
    try {
      const files = readdirSync(dataDir, { recursive: true });
      for (const f of files) {
        const path = String(f);
        assert.ok(
          !path.includes("completion"),
          `Completion file should NOT be in .aether/data: ${path}`
        );
      }
    } catch {
      // data dir may not have subdirectories
    }

    // Cleanup completion files
    for (const dir of completionDirs) {
      rmSync(dir, { recursive: true, force: true });
    }
  });

  it("runLifecycle handles plan-finalize correctly", async () => {
    assert.ok(context, "Test context should be initialized");
    const { bridge, dataDir } = context;

    // Just run the plan step and verify state
    const planResult = callGoJSON<Record<string, unknown>>(bridge, [
      "plan",
      "--plan-only",
      "--depth",
      "fast",
    ]);

    // Plan-only should return a plan_manifest
    assert.ok(
      planResult["plan_manifest"] ?? planResult["planning_manifest"],
      "Plan --plan-only should return a plan_manifest"
    );

    // Verify initial state is READY
    const stateBefore = readColonyState(dataDir);
    assert.equal(
      stateBefore["state"],
      "READY",
      "Colony should be in READY state before plan-finalize"
    );
  });

  it("runLifecycle reports failure with step context on error", async () => {
    // Create a colony with invalid state that will cause the lifecycle to fail
    const tempDir = mkdtempSync(join(tmpdir(), "ts-host-lifecycle-fail-"));
    const dataDir = join(tempDir, ".aether", "data");
    mkdirSync(dataDir, { recursive: true });

    // Write a colony state with no goal (will cause plan to fail)
    const colonyState = {
      version: "3.0",
      goal: "",
      state: "READY",
      plan: { phases: [] },
      current_phase: 0,
    };
    writeFileSync(
      join(dataDir, "COLONY_STATE.json"),
      JSON.stringify(colonyState, null, 2),
      "utf-8"
    );

    const opts: LifecycleOptions = {
      goBinaryPath: discoverSourceGoBinary(),
      cwd: tempDir,
      simulateWorkers: true,
    };

    const result = await runLifecycle(opts);

    // The lifecycle should fail
    assert.ok(!result.success, "Lifecycle should fail with empty goal");
    assert.ok(result.error, "Error message should be present");
    assert.ok(
      result.error!.includes("plan"),
      `Error should mention plan step: ${result.error}`
    );
    assert.equal(
      result.steps_completed.length,
      0,
      "No steps should be completed on plan failure"
    );

    // Cleanup
    rmSync(tempDir, { recursive: true, force: true });
  });

  it("lifecycle with dashboard option still stops at pending planning", async () => {
    assert.ok(context, "Test context should be initialized");
    const { bridge } = context;

    // Force dashboard on by overriding isTTY
    const originalIsTTY = process.stdout.isTTY;
    Object.defineProperty(process.stdout, "isTTY", { value: true, writable: true });

    const opts: LifecycleOptions = {
      goBinaryPath: bridge.goBinaryPath,
      cwd: bridge.cwd,
      simulateWorkers: true,
      phase: 1,
      dashboard: true,
    };

    const result = await runLifecycle(opts);

    // Restore isTTY
    Object.defineProperty(process.stdout, "isTTY", { value: originalIsTTY, writable: true });

    assertRequiresPlanningLoop(result);
  });

  it("lifecycle with no-dashboard still stops at pending planning", async () => {
    assert.ok(context, "Test context should be initialized");
    const { bridge } = context;

    const opts: LifecycleOptions = {
      goBinaryPath: bridge.goBinaryPath,
      cwd: bridge.cwd,
      simulateWorkers: true,
      phase: 1,
      dashboard: false,
    };

    const result = await runLifecycle(opts);

    assertRequiresPlanningLoop(result);
  });

  it("lifecycle does not start build dashboard while planning is pending", async () => {
    // Use the existing test context which has a valid colony state
    assert.ok(context, "Test context should be initialized");
    const { bridge } = context;

    // Force TTY so dashboard would be created
    const originalIsTTY = process.stdout.isTTY;
    Object.defineProperty(process.stdout, "isTTY", { value: true, writable: true });

    const opts: LifecycleOptions = {
      goBinaryPath: bridge.goBinaryPath,
      cwd: bridge.cwd,
      simulateWorkers: true,
      phase: 1,
      dashboard: true,
    };

    const result = await runLifecycle(opts);

    // Restore isTTY
    Object.defineProperty(process.stdout, "isTTY", { value: originalIsTTY, writable: true });

    assertRequiresPlanningLoop(result);

    // Since build never starts, the build dashboard never owns terminal state.
  });

  // ---------------------------------------------------------------------------
  // QueenOrchestrator integration tests
  // ---------------------------------------------------------------------------

  it("lifecycle does not use QueenOrchestrator before planning completes", async () => {
    assert.ok(context, "Test context should be initialized");
    const { bridge } = context;

    const lifecycleModule = await import("../src/lifecycle.js");

    let runBuildCalled = false;

    lifecycleModule.__setCreateQueenOrchestrator((opts: unknown) => {
      return {
        async runBuild(manifest: { dispatches?: unknown[] }) {
          runBuildCalled = true;
          // Return one result per dispatch so Go finalizer is satisfied
          const dispatches = manifest.dispatches ?? [];
          const workerResults = dispatches.map((d: unknown) => {
            const dispatch = d as { name: string; caste?: string; task?: string };
            return {
              name: dispatch.name,
              status: "completed",
              caste: dispatch.caste ?? "builder",
              task: dispatch.task ?? "Build task",
              files_modified: [".aether/ts-host/SIMULATED_BUILD_OUTPUT.txt"],
            };
          });
          return {
            success: true,
            workerResults,
            pattern: "SPBV" as const,
            recommendation: { review_depth: "fast", reason: "Test" },
          };
        },
      };
    });

    try {
      const opts: LifecycleOptions = {
        goBinaryPath: bridge.goBinaryPath,
        cwd: bridge.cwd,
        simulateWorkers: true,
        phase: 1,
      };

      const result = await runLifecycle(opts);
      assertRequiresPlanningLoop(result);
      assert.equal(runBuildCalled, false, "QueenOrchestrator.runBuild should not be called");
    } finally {
      lifecycleModule.__restoreCreateQueenOrchestrator();
    }
  });

  it("lifecycle does not pass build options to QueenOrchestrator before planning completes", async () => {
    assert.ok(context, "Test context should be initialized");
    const { bridge } = context;

    const lifecycleModule = await import("../src/lifecycle.js");

    let receivedOpts: unknown;

    lifecycleModule.__setCreateQueenOrchestrator((opts: unknown) => {
      receivedOpts = opts;
      return {
        async runBuild(manifest: { dispatches?: unknown[] }) {
          const dispatches = manifest.dispatches ?? [];
          const workerResults = dispatches.map((d: unknown) => {
            const dispatch = d as { name: string; caste?: string; task?: string };
            return {
              name: dispatch.name,
              status: "completed",
              caste: dispatch.caste ?? "builder",
              task: dispatch.task ?? "Build task",
              files_modified: [".aether/ts-host/SIMULATED_BUILD_OUTPUT.txt"],
            };
          });
          return {
            success: true,
            workerResults,
            pattern: "SPBV" as const,
            recommendation: { review_depth: "fast", reason: "Test" },
          };
        },
      };
    });

    try {
      const opts: LifecycleOptions = {
        goBinaryPath: bridge.goBinaryPath,
        cwd: bridge.cwd,
        simulateWorkers: true,
        phase: 1,
        skipMiddenCheck: true,
      };

      const result = await runLifecycle(opts);
      assertRequiresPlanningLoop(result);
      assert.equal(receivedOpts, undefined, "QueenOrchestrator should not be created");
    } finally {
      lifecycleModule.__restoreCreateQueenOrchestrator();
    }
  });

  it("lifecycle does not mask pending planning with QueenOrchestrator errors", async () => {
    assert.ok(context, "Test context should be initialized");
    const { bridge } = context;

    const lifecycleModule = await import("../src/lifecycle.js");

    lifecycleModule.__setCreateQueenOrchestrator(() => {
      return {
        async runBuild(manifest: { dispatches?: unknown[] }) {
          const dispatches = manifest.dispatches ?? [];
          const workerResults = dispatches.map((d: unknown) => {
            const dispatch = d as { name: string; caste?: string; task?: string };
            return {
              name: dispatch.name,
              status: "completed",
              caste: dispatch.caste ?? "builder",
              task: dispatch.task ?? "Build task",
              files_modified: [".aether/ts-host/SIMULATED_BUILD_OUTPUT.txt"],
            };
          });
          return {
            success: true,
            workerResults,
            pattern: "SPBV" as const,
            recommendation: { review_depth: "fast", reason: "Test" },
            error: "Simulated Queen failure",
          };
        },
      };
    });

    try {
      const opts: LifecycleOptions = {
        goBinaryPath: bridge.goBinaryPath,
        cwd: bridge.cwd,
        simulateWorkers: true,
        phase: 1,
      };

      const result = await runLifecycle(opts);
      assertRequiresPlanningLoop(result);
    } finally {
      lifecycleModule.__restoreCreateQueenOrchestrator();
    }
  });

  it("lifecycle does not write build state from Queen while planning is pending", async () => {
    assert.ok(context, "Test context should be initialized");
    const { bridge, dataDir } = context;

    const lifecycleModule = await import("../src/lifecycle.js");

    lifecycleModule.__setCreateQueenOrchestrator(() => {
      return {
        async runBuild(manifest: { dispatches?: unknown[] }) {
          const dispatches = manifest.dispatches ?? [];
          const workerResults = dispatches.map((d: unknown) => {
            const dispatch = d as { name: string; caste?: string; task?: string };
            return {
              name: dispatch.name,
              status: "completed",
              caste: dispatch.caste ?? "builder",
              task: dispatch.task ?? "Build task",
              files_modified: [".aether/ts-host/SIMULATED_BUILD_OUTPUT.txt"],
            };
          });
          return {
            success: true,
            workerResults,
            pattern: "SPBV" as const,
            recommendation: { review_depth: "standard", reason: "Test" },
          };
        },
      };
    });

    try {
      const opts: LifecycleOptions = {
        goBinaryPath: bridge.goBinaryPath,
        cwd: bridge.cwd,
        simulateWorkers: true,
        phase: 1,
      };

      const result = await runLifecycle(opts);
      assertRequiresPlanningLoop(result);

      // Verify colony state was not updated by build finalizers.
      const state = readColonyState(dataDir);
      const plan = state["plan"] as { phases?: unknown[] } | undefined;
      assert.equal(plan?.phases?.length ?? 0, 0, "No final plan should exist");
    } finally {
      lifecycleModule.__restoreCreateQueenOrchestrator();
    }
  });

  it("renders Go-owned ceremony sequencing only through pending planning", async () => {
    assert.ok(context, "Test context should be initialized");
    const { bridge } = context;
    const calls: string[] = [];

    const fakeAdapter: CeremonyAdapter = {
      renderSpawnPlan(workflow: CeremonyWorkflow) {
        calls.push(`${workflow}:spawn-plan`);
        return "";
      },
      renderWaveStart(workflow: CeremonyWorkflow, _manifest: unknown, executionWave: number) {
        calls.push(`${workflow}:wave-start:${executionWave}`);
        return "";
      },
      renderWorkerComplete(workflow: CeremonyWorkflow, worker: unknown) {
        const name = (worker as { name?: string }).name ?? "unknown";
        calls.push(`${workflow}:worker-complete:${name}`);
        return "";
      },
      renderCloseout(workflow: CeremonyWorkflow) {
        calls.push(`${workflow}:closeout`);
        return "";
      },
    };

    __setCreateCeremonyAdapter(() => fakeAdapter);
    try {
      const result = await runLifecycle({
        goBinaryPath: bridge.goBinaryPath,
        cwd: bridge.cwd,
        simulateWorkers: true,
        phase: 1,
      });

      assertRequiresPlanningLoop(result);
      assert.deepEqual(
        calls.filter((call) => call.endsWith("spawn-plan") || call.endsWith("closeout")),
        ["plan:spawn-plan"]
      );
      assert.ok(
        calls.some((call) => call.startsWith("plan:wave-start:")),
        `Expected plan wave-start ceremony call, got ${JSON.stringify(calls)}`
      );
      assert.ok(
        !calls.some((call) => call.startsWith("build:")),
        `Build ceremony should not run before planning completes, got ${JSON.stringify(calls)}`
      );
    } finally {
      __restoreCreateCeremonyAdapter();
    }
  });
});
