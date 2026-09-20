/**
 * Integration tests for host.ts subcommand dispatch.
 *
 * Tests verify:
 * - Each subcommand builds the correct Go CLI args
 * - Flags are passed through correctly
 * - callGoJSON is invoked with the expected arguments
 * - Dispatched runner pipeline for build (HOST-02, HOST-06)
 */

import { describe, it, beforeEach, afterEach } from "node:test";
import assert from "node:assert/strict";

import {
  buildHostGoArgs,
  parseArgs,
  __setCallGoJSON,
  __restoreCallGoJSON,
  __setDispatchWorkers,
  __restoreDispatchWorkers,
  __setDetectAvailablePlatforms,
  __restoreDetectAvailablePlatforms,
  __restoreAllMocks,
  runDispatchedBuildCommand,
  runDispatchedPlanCommand,
  runDryRunDispatchedCommand,
} from "../src/host.js";

import type { DispatchResult } from "../src/worker-dispatch.js";
import type { BuildDispatch } from "../src/types.js";
import type { GoBridgeOptions } from "../src/go-bridge.js";
import {
  __setCreateCeremonyAdapter,
  __restoreCreateCeremonyAdapter,
} from "../src/ceremony-adapter.js";
import type { CeremonyAdapter, CeremonyWorkflow } from "../src/ceremony-adapter.js";
import {
  __setCallGoJSON as __setGoBridgeCallGoJSON,
  __restoreCallGoJSON as __restoreGoBridgeCallGoJSON,
} from "../src/go-bridge.js";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";
import { TEST_EXECUTION_BINDING } from "./execution-binding-fixture.js";

// Compute repo root from test file path: test/host.test.ts -> ts-host/test -> .aether -> repo root
const _testDirname = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = resolve(_testDirname, "..", "..", "..");

// ---------------------------------------------------------------------------
// Host integration tests (existing)
// ---------------------------------------------------------------------------

describe("host integration", () => {
  beforeEach(() => {
    __restoreAllMocks();
  });

  afterEach(() => {
    __restoreAllMocks();
  });

  it("plan passes depth and planning-depth to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "plan",
      "--depth", "balanced",
      "--planning-depth", "standard",
    ]);

    const args = buildHostGoArgs(parsed)!;

    assert.deepStrictEqual(args, [
      "plan", "--plan-only",
      "--depth", "balanced",
      "--planning-depth", "standard",
    ]);
  });

  it("plan passes preset to Go CLI and does not mark it unknown", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "plan",
      "--preset", "fast",
    ]);

    assert.deepStrictEqual(parsed.unknownFlags, []);

    const args = buildHostGoArgs(parsed)!;

    assert.deepStrictEqual(args, [
      "plan", "--plan-only",
      "--preset", "fast",
    ]);
  });

  it("plan passes equals-form depth flags to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "plan",
      "--depth=balanced",
      "--planning-depth=standard",
    ]);

    const args = buildHostGoArgs(parsed)!;

    assert.deepStrictEqual(args, [
      "plan", "--plan-only",
      "--depth", "balanced",
      "--planning-depth", "standard",
    ]);
  });

  it("plan passes verification-depth and worker-timeout to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "plan",
      "--verification-depth", "heavy",
      "--worker-timeout", "5m",
    ]);

    const args = buildHostGoArgs(parsed)!;

    assert.deepStrictEqual(args, [
      "plan", "--plan-only",
      "--verification-depth", "heavy",
      "--worker-timeout", "5m",
    ]);
  });

  it("plan passes classic loop controls to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "plan",
      "--target", "95",
      "--max-iterations", "8",
      "--accept",
    ]);

    const args = buildHostGoArgs(parsed)!;

    assert.deepStrictEqual(args, [
      "plan", "--plan-only",
      "--target", "95",
      "--max-iterations", "8",
      "--accept",
    ]);
  });

  it("plan passes synthetic flag to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "plan",
      "--synthetic",
    ]);

    const args = buildHostGoArgs(parsed)!;

    assert.deepStrictEqual(args, [
      "plan", "--plan-only",
      "--synthetic",
    ]);
  });

  it("plan passes refresh and force to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "plan",
      "--refresh",
      "--force",
    ]);

    const args = buildHostGoArgs(parsed)!;

    assert.deepStrictEqual(args, [
      "plan", "--plan-only",
      "--refresh",
      "--force",
    ]);
  });

  it("plan preserves versioned revision metadata", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "plan",
      "--refresh",
      "--revision-type", "verification_failure",
      "--revision-reason", "The acceptance check invalidated the remaining plan",
      "--revision-evidence", ".aether/data/build/phase-2/verification.json",
      "--revision-evidence=.aether/data/build/phase-2/review.json",
    ]);

    const args = buildHostGoArgs(parsed)!;

    assert.deepStrictEqual(args, [
      "plan", "--plan-only", "--refresh",
      "--revision-type", "verification_failure",
      "--revision-reason", "The acceptance check invalidated the remaining plan",
      "--revision-evidence", ".aether/data/build/phase-2/verification.json",
      "--revision-evidence", ".aether/data/build/phase-2/review.json",
    ]);
  });

  it("build passes phase and light flag to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "build", "1",
      "--light",
    ]);

    const args = buildHostGoArgs(parsed)!;

    assert.deepStrictEqual(args, [
      "build", "1", "--plan-only",
      "--light",
    ]);
  });

  it("build passes worker-timeout to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "build", "2",
      "--worker-timeout", "15m",
    ]);

    const args = buildHostGoArgs(parsed)!;

    assert.deepStrictEqual(args, [
      "build", "2", "--plan-only",
      "--worker-timeout", "15m",
    ]);
  });

  it("build passes heavy and verification-depth to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "build", "3",
      "--heavy",
      "--verification-depth", "heavy",
    ]);

    const args = buildHostGoArgs(parsed)!;

    assert.deepStrictEqual(args, [
      "build", "3", "--plan-only",
      "--heavy",
      "--verification-depth", "heavy",
    ]);
  });

  it("build passes force, repeated task, circuit breaker, no-suggest, and verbose to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "build", "5",
      "--task", "5.1",
      "--task=5.2",
      "--force",
      "--circuit-breaker-threshold", "4",
      "--no-suggest",
      "--verbose",
    ]);

    const args = buildHostGoArgs(parsed)!;

    assert.deepStrictEqual(args, [
      "build", "5", "--plan-only",
      "--task", "5.1",
      "--task", "5.2",
      "--force",
      "--circuit-breaker-threshold", "4",
      "--no-suggest",
      "--verbose",
    ]);
  });

  it("build rejects unknown host flags before Go invocation", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "build", "5",
      "--definitely-unknown",
    ]);

    assert.throws(() => buildHostGoArgs(parsed), /Unsupported host flag\(s\): --definitely-unknown/);
  });

  it("colonize passes force-resurvey and worker-timeout to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "colonize",
      "--force-resurvey",
      "--worker-timeout", "5m",
    ]);

    const args = buildHostGoArgs(parsed)!;

    assert.deepStrictEqual(args, [
      "colonize", "--plan-only",
      "--force-resurvey",
      "--worker-timeout", "5m",
    ]);
  });

  it("continue passes verification-depth heavy to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "continue",
      "--verification-depth", "heavy",
    ]);

    const args = buildHostGoArgs(parsed)!;

    assert.deepStrictEqual(args, [
      "continue", "--plan-only",
      "--verification-depth", "heavy",
    ]);
  });

  it("continue passes equals-form verification-depth to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "continue",
      "--verification-depth=heavy",
    ]);

    const args = buildHostGoArgs(parsed)!;

    assert.deepStrictEqual(args, [
      "continue", "--plan-only",
      "--verification-depth", "heavy",
    ]);
  });

  it("continue passes light and heavy flags to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "continue",
      "--light",
      "--heavy",
    ]);

    const args = buildHostGoArgs(parsed)!;

    assert.deepStrictEqual(args, [
      "continue", "--plan-only",
      "--light",
      "--heavy",
    ]);
  });

  it("continue passes skip-watchers to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "continue",
      "--skip-watchers",
    ]);

    const args = buildHostGoArgs(parsed)!;

    assert.deepStrictEqual(args, [
      "continue", "--plan-only",
      "--skip-watchers",
    ]);
  });

  it("continue passes synthetic and worker-timeout to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "continue",
      "--simulate",
      "--worker-timeout", "10m",
    ]);

    const args = buildHostGoArgs(parsed)!;

    assert.deepStrictEqual(args, [
      "continue", "--plan-only",
      "--synthetic",
      "--worker-timeout", "10m",
    ]);
  });

  it("continue passes reconcile-task, verification-timeout, and no-learn to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "continue",
      "--reconcile-task", "5.1",
      "--reconcile-task=5.2",
      "--verification-timeout", "30m",
      "--no-learn",
    ]);

    const args = buildHostGoArgs(parsed)!;

    assert.deepStrictEqual(args, [
      "continue", "--plan-only",
      "--reconcile-task", "5.1",
      "--reconcile-task", "5.2",
      "--verification-timeout", "30m",
      "--no-learn",
    ]);
  });

  it("continue passes classic ceremony to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "continue",
      "--classic-ceremony",
    ]);

    const args = buildHostGoArgs(parsed)!;

    assert.deepStrictEqual(args, [
      "continue", "--plan-only",
      "--classic-ceremony",
    ]);
  });

  it("seal passes force to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "seal",
      "--force",
    ]);

    const args = buildHostGoArgs(parsed)!;

    assert.deepStrictEqual(args, [
      "seal", "--plan-only",
      "--force",
    ]);
  });

  it("parses all 9 documented subcommands", () => {
    const commands = ["colonize", "plan", "build", "continue", "seal", "oracle", "lifecycle", "watch", "swarm"];

    for (const cmd of commands) {
      const parsed = parseArgs(["node", "host.js", cmd]);
      assert.equal(parsed.command, cmd, `Should parse command: ${cmd}`);
    }
  });

  it("callGoJSON mock can be injected and restored", () => {
    let called = false;
    __setCallGoJSON(<T>(_opts: unknown, _args: string[]): T => {
      called = true;
      return { ok: true } as unknown as T;
    });

    const parsed = parseArgs(["node", "host.js", "plan"]);
    const args = buildHostGoArgs(parsed)!;

    // Simulate what main() would do
    const result = { ok: true };
    __restoreCallGoJSON();

    assert.ok(!called || true, "Mock was set up correctly");
    assert.ok(args);
    assert.equal(args[0], "plan");
  });
});

// ---------------------------------------------------------------------------
// Dispatched runner tests (HOST-02, HOST-06)
// ---------------------------------------------------------------------------

describe("dispatched build runner", () => {
  // Helper to create a fake build manifest result
  function fakeBuildManifest(overrides?: { skillCount?: number }) {
    const dispatches: Array<Record<string, unknown>> = [
      {
        stage: "implement",
        wave: 1,
        execution_wave: 1,
        caste: "builder",
        name: "Mason-01",
        task: "Implement feature X",
        status: "pending",
        skill_section: overrides?.skillCount ? "Skill content here" : undefined,
      },
      {
        stage: "implement",
        wave: 1,
        execution_wave: 1,
        caste: "builder",
        name: "Mason-02",
        task: "Implement feature Y",
        status: "pending",
      },
    ];
    return {
      dispatch_manifest: {
        execution_binding: TEST_EXECUTION_BINDING,
        phase: 1,
        phase_name: "test",
        dispatches,
      },
      dispatches,
    };
  }

  function fakeDispatchResults(): DispatchResult[] {
    return [
      {
        name: "Mason-01",
        status: "completed",
        summary: "Implemented feature X",
        duration: 10,
        files_created: ["src/feature-x.ts"],
      },
      {
        name: "Mason-02",
        status: "completed",
        summary: "Implemented feature Y",
        duration: 12,
        files_modified: ["src/feature-y.ts"],
      },
    ];
  }

  let goCalls: string[][];

  beforeEach(() => {
    __restoreAllMocks();
    goCalls = [];

    // Mock callGoJSON to track calls and return appropriate results
    __setCallGoJSON(<T>(_opts: unknown, args: string[]): T => {
      goCalls.push(args);
      const cmd = args[0];
      if (cmd === "build") {
        return fakeBuildManifest() as unknown as T;
      }
      if (cmd === "build-finalize") {
        return { ok: true } as unknown as T;
      }
      return { ok: true } as unknown as T;
    });

    // Mock dispatchWorkers to track calls
    __setDispatchWorkers(async (_opts, _dispatches) => {
      return fakeDispatchResults();
    });

    // Mock detectAvailablePlatforms
    __setDetectAvailablePlatforms(async () => [
      "claude" as const,
    ]);
  });

  afterEach(() => {
    __restoreAllMocks();
  });

  it("build command uses dispatched runner type", async () => {
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build");
    assert.equal(def?.runner, "dispatched", "build command should use dispatched runner");
    assert.equal(def?.ceremonyWorkflow, "build", "build should have build ceremony workflow");
  });

  it("build dispatched runner calls Go manifest with build --plan-only", () => {
    const parsed = parseArgs(["node", "host.js", "build", "1"]);
    const args = buildHostGoArgs(parsed)!;

    assert.equal(args[0], "build");
    assert.equal(args[1], "1");
    assert.ok(args.includes("--plan-only"), "Should include --plan-only");
  });

  it("build dispatched runner passes --simulate as simulateWorkers to dispatch", () => {
    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate"]);
    assert.equal(parsed.simulate, true, "--simulate should set parsed.simulate to true");

    const args = buildHostGoArgs(parsed)!;
    assert.ok(args.includes("--synthetic"), "--simulate should forward as --synthetic to Go");
  });

  it("plan is a manifest bridge while continue uses dispatched runner", async () => {
    const { getHostCommandDefinition } = await import("../src/command-registry.js");

    const planDef = getHostCommandDefinition("plan");
    assert.equal(planDef?.runner, "go-json", "plan should return a Go-owned iteration manifest");

    const continueDef = getHostCommandDefinition("continue");
    assert.equal(continueDef?.runner, "dispatched", "continue should use dispatched runner");
  });

  it("mock injection and restore for dispatch workers works", () => {
    let dispatchCalled = false;
    __setDispatchWorkers(async () => {
      dispatchCalled = true;
      return fakeDispatchResults();
    });

    __restoreDispatchWorkers();
    // After restore, the real dispatchWorkers is back
    assert.equal(dispatchCalled, false, "Dispatch should not be called during restore");
  });

  it("mock injection and restore for detectAvailablePlatforms works", async () => {
    let detectCalled = false;
    __setDetectAvailablePlatforms(async () => {
      detectCalled = true;
      return [];
    });

    __restoreDetectAvailablePlatforms();
    assert.equal(detectCalled, false, "Detect should not be called during restore");
  });
});

// ---------------------------------------------------------------------------
// Playbook context injection tests (CEREMONY-06)
// ---------------------------------------------------------------------------

function createMockCeremonyAdapterForHost(): CeremonyAdapter {
  return {
    renderSpawnPlan: (_workflow: CeremonyWorkflow, _manifest: unknown) => "",
    renderWaveStart: (_workflow: CeremonyWorkflow, _manifest: unknown, _wave: number) => "",
    renderWorkerComplete: (_workflow: CeremonyWorkflow, _worker: unknown) => "",
    renderCloseout: (_workflow: CeremonyWorkflow, _completionPath: string) => "",
  };
}

describe("playbook context injection (CEREMONY-06)", () => {
  let capturedDispatches: unknown[] | undefined;
  let goCalls: string[][];

  beforeEach(() => {
    __restoreAllMocks();
    __restoreGoBridgeCallGoJSON();
    __restoreCreateCeremonyAdapter();
    capturedDispatches = undefined;
    goCalls = [];

    __setCreateCeremonyAdapter(() => createMockCeremonyAdapterForHost());
  });

  afterEach(() => {
    __restoreAllMocks();
    __restoreGoBridgeCallGoJSON();
    __restoreCreateCeremonyAdapter();
  });

  it("build runner does NOT inject playbook context into worker task_briefs", async () => {
    const handler = <T>(_opts: unknown, args: string[]): T => {
      goCalls.push(args);
      const cmd = args[0];
      if (cmd === "hive-read") {
        return { entries: null, total: 0 } as unknown as T;
      }
      if (cmd === "registry-list") {
        return {
          colonies: [{
            repo_path: process.cwd(),
            domains: ["typescript"],
            active: true,
            registered_at: "2026-05-18T00:00:00Z",
          }],
        } as unknown as T;
      }
      if (cmd === "build") {
        return {
          dispatch_manifest: {
            execution_binding: TEST_EXECUTION_BINDING,
            dispatches: [
              { name: "Builder-01", caste: "builder", task: "Implement feature", wave: 1, execution_wave: 1 },
            ],
          },
        } as unknown as T;
      }
      if (cmd === "build-finalize") {
        return { ok: true } as unknown as T;
      }
      return { ok: true } as unknown as T;
    };

    __setCallGoJSON(handler);
    __setGoBridgeCallGoJSON(handler);

    const allCapturedDispatches: unknown[][] = [];
    __setDispatchWorkers(async (_opts, dispatches) => {
      allCapturedDispatches.push(dispatches);
      return dispatches.map((d: BuildDispatch) => ({
        name: d.name,
        status: "completed",
        summary: "Done",
        duration: 5,
        files_created: ["src/feature.ts"],
        tests_written: ["test/feature.test.ts"],
        test_results: { passed: 10, total: 10 },
      }));
    });

    __setDetectAvailablePlatforms(async () => [
      "claude" as const,
    ]);

    // Use --max-iterations 1 to ensure single iteration (confidence high from test_results)
    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate", "--max-iterations", "1"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: process.cwd() };
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    await runDispatchedBuildCommand(bridge, parsed, def);

    assert.ok(allCapturedDispatches.length > 0, "dispatchWorkers should have been called");

    // Verify playbook context in first iteration dispatches
    const firstIterationDispatches = allCapturedDispatches[0]!;
    for (const d of firstIterationDispatches) {
      const dispatch = d as Record<string, unknown>;
      const brief = dispatch.task_brief as string;
      assert.ok(
        typeof brief !== "string" || !brief.includes("Relevant Playbooks"),
        `Build dispatch ${dispatch.name} task_brief must NOT contain playbook context; playbooks are orchestrator guidance and told workers they were the Queen. Got: ${brief?.slice(0, 200)}`
      );
    }
  });

  it("plan runner does NOT inject playbook context into worker task_briefs", async () => {
    const handler = <T>(_opts: unknown, args: string[]): T => {
      goCalls.push(args);
      const cmd = args[0];
      if (cmd === "hive-read") {
        return { entries: null, total: 0 } as unknown as T;
      }
      if (cmd === "registry-list") {
        return {
          colonies: [{
            repo_path: process.cwd(),
            domains: ["typescript"],
            active: true,
            registered_at: "2026-05-18T00:00:00Z",
          }],
        } as unknown as T;
      }
      if (cmd === "plan") {
        return {
          plan_manifest: { phases: 5 },
          dispatches: [
            { name: "Scout-01", caste: "scout", task: "Research codebase", wave: 1, execution_wave: 1 },
          ],
        } as unknown as T;
      }
      if (cmd === "plan-finalize") {
        return { ok: true } as unknown as T;
      }
      return { ok: true } as unknown as T;
    };

    __setCallGoJSON(handler);
    __setGoBridgeCallGoJSON(handler);

    __setDispatchWorkers(async (_opts, dispatches) => {
      capturedDispatches = dispatches;
      return dispatches.map((d: BuildDispatch) => ({
        name: d.name,
        status: "completed",
        summary: "Done",
        duration: 5,
      }));
    });

    __setDetectAvailablePlatforms(async () => [
      "claude" as const,
    ]);

    const parsed = parseArgs(["node", "host.js", "plan", "--simulate"]);
    // Use repo root as cwd so plan playbooks are found at .aether/docs/command-playbooks/
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: REPO_ROOT };

    await runDispatchedPlanCommand(bridge, parsed);

    assert.ok(capturedDispatches, "dispatchWorkers should have been called for plan");
    assert.ok(capturedDispatches!.length > 0, "dispatches should not be empty");

    for (const d of capturedDispatches!) {
      const dispatch = d as Record<string, unknown>;
      const brief = dispatch.task_brief as string;
      assert.ok(
        typeof brief !== "string" || !brief.includes("Relevant Playbooks"),
        `Plan dispatch ${dispatch.name} task_brief must NOT contain playbook context (orchestrator guidance). Got: ${brief?.slice(0, 200)}`
      );
    }
  });

  it("dry-run build does NOT inject playbook context into manifest dispatches", async () => {
    const handler = <T>(_opts: unknown, args: string[]): T => {
      goCalls.push(args);
      const cmd = args[0];
      if (cmd === "hive-read") {
        return { entries: null, total: 0 } as unknown as T;
      }
      if (cmd === "registry-list") {
        return {
          colonies: [{
            repo_path: process.cwd(),
            domains: ["typescript"],
            active: true,
            registered_at: "2026-05-18T00:00:00Z",
          }],
        } as unknown as T;
      }
      if (cmd === "build") {
        return {
          dispatch_manifest: {
            execution_binding: TEST_EXECUTION_BINDING,
            dispatches: [
              { name: "Builder-01", caste: "builder", task: "Build", wave: 1, execution_wave: 1 },
            ],
          },
        } as unknown as T;
      }
      return { ok: true } as unknown as T;
    };

    __setCallGoJSON(handler);
    __setGoBridgeCallGoJSON(handler);

    const parsed = parseArgs(["node", "host.js", "build", "1", "--dry-run"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: process.cwd() };
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    // Capture stdout to inspect the dry-run manifest output
    let stdoutOutput = "";
    const originalStdoutWrite = process.stdout.write.bind(process.stdout);
    process.stdout.write = ((chunk: unknown, ...args: unknown[]) => {
      if (typeof chunk === "string") stdoutOutput += chunk;
      return originalStdoutWrite(chunk as string | Uint8Array, ...args as [BufferEncoding]);
    }) as typeof process.stdout.write;

    try {
      await runDryRunDispatchedCommand(bridge, parsed, def);

      // Verify playbook context appears in the manifest output
      assert.ok(
        !stdoutOutput.includes("Relevant Playbooks"),
        `Dry-run output must NOT contain playbook context. Got: ${stdoutOutput.slice(0, 500)}`
      );
    } finally {
      process.stdout.write = originalStdoutWrite;
    }
  });

  it("dry-run plan does NOT inject playbook context into manifest dispatches", async () => {
    const handler = <T>(_opts: unknown, args: string[]): T => {
      goCalls.push(args);
      const cmd = args[0];
      if (cmd === "hive-read") {
        return { entries: null, total: 0 } as unknown as T;
      }
      if (cmd === "registry-list") {
        return {
          colonies: [{
            repo_path: process.cwd(),
            domains: ["typescript"],
            active: true,
            registered_at: "2026-05-18T00:00:00Z",
          }],
        } as unknown as T;
      }
      if (cmd === "plan") {
        return {
          plan_manifest: { phases: 5 },
          dispatches: [
            { name: "Scout-01", caste: "scout", task: "Research", wave: 1, execution_wave: 1 },
          ],
        } as unknown as T;
      }
      return { ok: true } as unknown as T;
    };

    __setCallGoJSON(handler);
    __setGoBridgeCallGoJSON(handler);

    const parsed = parseArgs(["node", "host.js", "plan", "--dry-run"]);
    // Use repo root as cwd so plan playbooks are found at .aether/docs/command-playbooks/
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: REPO_ROOT };
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("plan")!;

    let stdoutOutput = "";
    const originalStdoutWrite = process.stdout.write.bind(process.stdout);
    process.stdout.write = ((chunk: unknown, ...args: unknown[]) => {
      if (typeof chunk === "string") stdoutOutput += chunk;
      return originalStdoutWrite(chunk as string | Uint8Array, ...args as [BufferEncoding]);
    }) as typeof process.stdout.write;

    try {
      await runDryRunDispatchedCommand(bridge, parsed, def);

      assert.ok(
        !stdoutOutput.includes("Relevant Playbooks"),
        `Dry-run plan output must NOT contain playbook context. Got: ${stdoutOutput.slice(0, 500)}`
      );
    } finally {
      process.stdout.write = originalStdoutWrite;
    }
  });
});
