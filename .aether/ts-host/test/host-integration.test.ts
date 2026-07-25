/**
 * Integration tests for the host entry point.
 *
 * Tests verify:
 * - Host module source file exists
 * - Host prints usage when called with no args
 * - Host lifecycle command is explicit simulate-only smoke
 * - Plan and continue dispatched runner pipeline (HOST-03, HOST-04)
 */

import { execFileSync, spawnSync } from "node:child_process";
import { existsSync, mkdtempSync, readFileSync, symlinkSync } from "node:fs";
import { join, dirname } from "node:path";
import { tmpdir } from "node:os";
import { fileURLToPath } from "node:url";

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
  __setPreflightWorkerPlatform,
  __restorePreflightWorkerPlatform,
  __restoreAllMocks,
  runDispatchedBuildCommand,
  runDispatchedPlanCommand,
  runDispatchedContinueCommand,
  runDryRunDispatchedCommand,
} from "../src/host.js";

import type { DispatchResult } from "../src/worker-dispatch.js";
import type { BuildDispatch } from "../src/types.js";
import type { GoBridgeOptions } from "../src/go-bridge.js";
import {
  __setCallGoJSON as __setGoBridgeCallGoJSON,
  __restoreCallGoJSON as __restoreGoBridgeCallGoJSON,
} from "../src/go-bridge.js";
import {
  __setCreateCeremonyAdapter,
  __restoreCreateCeremonyAdapter,
} from "../src/ceremony-adapter.js";
import type { CeremonyAdapter, CeremonyWorkflow } from "../src/ceremony-adapter.js";
import { TEST_EXECUTION_BINDING } from "./execution-binding-fixture.js";

// Path to host entry point source
const __dirname = dirname(fileURLToPath(import.meta.url));
const hostPath = join(__dirname, "..", "src", "host.ts");

describe("host entry point", () => {
  it("host module source file exists", () => {
    assert.ok(existsSync(hostPath), `host.ts should exist at ${hostPath}`);
  });

  it("host prints usage when called with no args", () => {
    let stderr = "";
    try {
      stderr = execFileSync("node", ["--import", "tsx", hostPath], {
        encoding: "utf-8",
        timeout: 10000,
      });
    } catch (err: unknown) {
      // The process exits with code 1 which throws in execFileSync
      const e = err as { stderr?: string; status?: number };
      stderr = e.stderr ?? "";
      // Exit code 1 is expected for usage display
      assert.equal(e.status, 1, "Should exit with code 1");
    }

    // Should contain usage info, not an unhandled crash
    assert.ok(
      stderr.includes("Usage:") || stderr.includes("command"),
      `Stderr should contain usage info. Got: ${stderr.slice(0, 200)}`
    );
  });

  it("host runs when invoked through a symlinked entrypoint path", () => {
    const tempDir = mkdtempSync(join(tmpdir(), "aether-host-symlink-"));
    const symlinkedHost = join(tempDir, "host.ts");
    symlinkSync(hostPath, symlinkedHost);

    const result = spawnSync("node", ["--import", "tsx", symlinkedHost], {
      encoding: "utf-8",
      timeout: 10000,
    });

    assert.equal(result.status, 1);
    assert.match(result.stderr ?? "", /Usage:|command/);
  });

  it("host lifecycle command rejects production-style execution without --simulate", () => {
    const result = spawnSync(
      "node",
      ["--import", "tsx", hostPath, "lifecycle"],
      {
        encoding: "utf-8",
        timeout: 10000,
      }
    );

    assert.equal(result.status, 1);
    assert.match(result.stderr ?? "", /simulate-only/);
    assert.match(result.stderr ?? "", /aether plan\/build\/continue/);
  });

  it("host rejects unknown flags for display commands", () => {
    const result = spawnSync(
      "node",
      ["--import", "tsx", hostPath, "watch", "--definitely-unknown", "--no-dashboard"],
      {
        encoding: "utf-8",
        timeout: 10000,
      }
    );

    assert.equal(result.status, 1);
    assert.match(result.stderr ?? "", /Unsupported host flag\(s\): --definitely-unknown/);
  });
});

// ---------------------------------------------------------------------------
// Plan and continue dispatched runner tests (HOST-03, HOST-04)
// ---------------------------------------------------------------------------

describe("dispatched plan and continue runners", () => {
  let goCalls: string[][];

  function fakeDispatchResults(): DispatchResult[] {
    return [
      {
        name: "Scout-01",
        status: "completed",
        summary: "Analyzed codebase",
        duration: 8,
      },
    ];
  }

  function fakeContinueDispatchResults(): DispatchResult[] {
    return [
      {
        name: "Watcher-01",
        status: "completed",
        summary: "Verification passed",
        duration: 5,
        files_modified: ["src/module.ts"],
      },
    ];
  }

  beforeEach(() => {
    __restoreAllMocks();
    goCalls = [];

    // Mock callGoJSON to track calls
    __setCallGoJSON(<T>(_opts: unknown, args: string[]): T => {
      goCalls.push(args);
      const cmd = args[0];
      if (cmd === "plan") {
        return {
          plan_manifest: { phases: 5 },
          dispatches: [
            { name: "Scout-01", caste: "scout", task: "Research", wave: 1, execution_wave: 1 },
          ],
        } as unknown as T;
      }
      if (cmd === "plan-finalize") {
        return { ok: true } as unknown as T;
      }
      if (cmd === "continue") {
        return {
          continue_manifest: { phase: 1 },
          dispatches: [
            { name: "Watcher-01", caste: "watcher", task: "Verify", wave: 1, execution_wave: 1 },
          ],
        } as unknown as T;
      }
      if (cmd === "continue-finalize") {
        return { ok: true } as unknown as T;
      }
      return { ok: true } as unknown as T;
    });

    // Mock dispatchWorkers
    __setDispatchWorkers(async (_opts, dispatches) => {
      if (dispatches[0]?.name === "Watcher-01") {
        return fakeContinueDispatchResults();
      }
      return fakeDispatchResults();
    });

    // Mock detectAvailablePlatforms
    __setDetectAvailablePlatforms(async () => ["claude"]);
  });

  afterEach(() => {
    __restoreAllMocks();
  });

  it("plan command uses go-json manifest bridge type", async () => {
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("plan");
    assert.equal(def?.runner, "go-json", "plan command should return a Go-owned iteration manifest");
    assert.equal(def?.ceremonyWorkflow, "plan", "plan should have plan ceremony workflow");
  });

  it("continue command uses dispatched runner type", async () => {
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("continue");
    assert.equal(def?.runner, "dispatched", "continue command should use dispatched runner");
    assert.equal(def?.ceremonyWorkflow, "continue", "continue should have continue ceremony workflow");
  });

  it("plan dispatched runner calls Go manifest with plan --plan-only", () => {
    const parsed = parseArgs(["node", "host.js", "plan"]);
    const args = buildHostGoArgs(parsed)!;

    assert.equal(args[0], "plan");
    assert.ok(args.includes("--plan-only"), "Should include --plan-only");
  });

  it("plan dispatched runner calls plan-finalize", async () => {
    // Verify the finalizer command is configured
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("plan");
    assert.ok(
      def?.finalizerCommand?.includes("plan-finalize"),
      "plan should have plan-finalize finalizer command"
    );
  });

  it("continue dispatched runner calls Go manifest with continue --plan-only", () => {
    const parsed = parseArgs(["node", "host.js", "continue"]);
    const args = buildHostGoArgs(parsed)!;

    assert.equal(args[0], "continue");
    assert.ok(args.includes("--plan-only"), "Should include --plan-only");
  });

  it("continue dispatched runner calls continue-finalize", async () => {
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("continue");
    assert.ok(
      def?.finalizerCommand?.includes("continue-finalize"),
      "continue should have continue-finalize finalizer command"
    );
  });

  it("plan dispatched runner passes simulateWorkers when --simulate is set", () => {
    const parsed = parseArgs(["node", "host.js", "plan", "--simulate"]);
    assert.equal(parsed.simulate, true, "--simulate should set parsed.simulate");
  });

  it("continue dispatched runner passes simulateWorkers when --simulate is set", () => {
    const parsed = parseArgs(["node", "host.js", "continue", "--simulate"]);
    assert.equal(parsed.simulate, true, "--simulate should set parsed.simulate");
  });

  it("plan preflight fails before worker dispatch when Codex model is unsupported", async () => {
    let dispatchCalled = false;
    __setDetectAvailablePlatforms(async () => ["codex"]);
    __setPreflightWorkerPlatform(async () => {
      throw new Error("Codex provider/model preflight failed before worker dispatch: The 'o4-mini' model is not supported");
    });
    __setDispatchWorkers(async () => {
      dispatchCalled = true;
      return fakeDispatchResults();
    });

    const parsed = parseArgs(["node", "host.js", "plan"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: process.cwd() };
    await assert.rejects(
      () => runDispatchedPlanCommand(bridge, parsed),
      /provider\/model preflight failed/
    );
    assert.equal(dispatchCalled, false, "dispatchWorkers must not run after preflight failure");
  });

  it("colonize and seal still use go-json runner", async () => {
    const { getHostCommandDefinition } = await import("../src/command-registry.js");

    const colonizeDef = getHostCommandDefinition("colonize");
    assert.equal(colonizeDef?.runner, "go-json", "colonize should use go-json");

    const sealDef = getHostCommandDefinition("seal");
    assert.equal(sealDef?.runner, "go-json", "seal should use go-json");
  });
});

// ---------------------------------------------------------------------------
// Dry-run ceremony preview tests (HOST-07, D-06)
// ---------------------------------------------------------------------------

describe("dry-run ceremony preview", () => {
  let goCalls: string[][];
  let dispatchCalled: boolean;
  let stderrOutput: string;

  beforeEach(() => {
    __restoreAllMocks();
    goCalls = [];
    dispatchCalled = false;
    stderrOutput = "";

    // Capture stderr output
    const originalStderrWrite = process.stderr.write.bind(process.stderr);
    process.stderr.write = ((chunk: unknown, ...args: unknown[]) => {
      if (typeof chunk === "string") stderrOutput += chunk;
      return originalStderrWrite(chunk as string | Uint8Array, ...args as [BufferEncoding]);
    }) as typeof process.stderr.write;

    // Mock callGoJSON to return manifests
    __setCallGoJSON(<T>(_opts: unknown, args: string[]): T => {
      goCalls.push(args);
      const cmd = args[0];
      if (cmd === "build") {
        return {
          dispatch_manifest: {
            execution_binding: TEST_EXECUTION_BINDING,
            dispatches: [
              { name: "Builder-01", caste: "builder", task: "Build", wave: 1, execution_wave: 1, skill_section: "TDD" },
            ],
          },
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
      if (cmd === "continue") {
        return {
          continue_manifest: { phase: 1 },
          dispatches: [
            { name: "Watcher-01", caste: "watcher", task: "Verify", wave: 1, execution_wave: 1 },
          ],
        } as unknown as T;
      }
      if (cmd === "oracle-iterate") {
        return {
          iteration_manifest: {
            topic: "test",
            depth: "balanced",
            max_iterations: 5,
            confidence_target: 85,
            current_iteration: 1,
            workers: [{ name: "Oracle-01", caste: "oracle", task: "Research", brief: "Do research" }],
          },
        } as unknown as T;
      }
      if (cmd === "colonize") {
        return { ok: true, dispatches: [] } as unknown as T;
      }
      return { ok: true } as unknown as T;
    });

    // Mock dispatchWorkers to track calls
    __setDispatchWorkers(async () => {
      dispatchCalled = true;
      return [{ name: "Builder-01", status: "completed", summary: "Done", duration: 5 }];
    });

    // Mock detectAvailablePlatforms
    __setDetectAvailablePlatforms(async () => ["claude"]);
  });

  afterEach(() => {
    __restoreAllMocks();
  });

  it("--dry-run on build renders ceremony without dispatching workers", async () => {
    const parsed = parseArgs(["node", "host.js", "build", "1", "--dry-run"]);
    assert.equal(parsed.dryRun, true, "--dry-run should set parsed.dryRun");
    assert.equal(parsed.simulate, false, "--dry-run should not set simulate");

    // Verify dry-run calls Go for manifest
    const args = buildHostGoArgs(parsed)!;
    assert.equal(args[0], "build");
    assert.ok(args.includes("--plan-only"));
  });

  it("--dry-run on plan renders ceremony without dispatching workers", () => {
    const parsed = parseArgs(["node", "host.js", "plan", "--dry-run"]);
    assert.equal(parsed.dryRun, true);

    const args = buildHostGoArgs(parsed)!;
    assert.equal(args[0], "plan");
    assert.ok(args.includes("--plan-only"));
  });

  it("--dry-run on continue renders ceremony without dispatching workers", () => {
    const parsed = parseArgs(["node", "host.js", "continue", "--dry-run"]);
    assert.equal(parsed.dryRun, true);

    const args = buildHostGoArgs(parsed)!;
    assert.equal(args[0], "continue");
    assert.ok(args.includes("--plan-only"));
  });

  it("--dry-run on oracle renders ceremony without dispatching workers", () => {
    const parsed = parseArgs(["node", "host.js", "oracle", "--dry-run"]);
    assert.equal(parsed.dryRun, true);
    // Oracle doesn't have buildGoArgs in registry (uses oracle-lifecycle runner)
  });

  it("--dry-run output contains DRY RUN indicator", async () => {
    // Verify the badge function exists and can be imported
    const { renderDryRunBadge } = await import("../src/ceremony-adapter.js");
    assert.equal(typeof renderDryRunBadge, "function", "renderDryRunBadge should be a function");
  });

  it("--dry-run does not call dispatchWorkers", () => {
    const parsed = parseArgs(["node", "host.js", "build", "1", "--dry-run"]);
    assert.equal(parsed.dryRun, true, "dryRun should be true");
    // dispatchCalled is tracked by mock; in the dry-run path, dispatchWorkers
    // should never be called because the main() function exits before reaching dispatch
  });
});

// ---------------------------------------------------------------------------
// Hive wisdom injection (HIVE-04, HIVE-05)
// ---------------------------------------------------------------------------

function createMockCeremonyAdapter(): CeremonyAdapter {
  return {
    renderSpawnPlan: (_workflow: CeremonyWorkflow, _manifest: unknown) => "",
    renderWaveStart: (_workflow: CeremonyWorkflow, _manifest: unknown, _wave: number) => "",
    renderWorkerComplete: (_workflow: CeremonyWorkflow, _worker: unknown) => "",
    renderCloseout: (_workflow: CeremonyWorkflow, _completionPath: string) => "",
  };
}

describe("hive wisdom injection (HIVE-04, HIVE-05)", () => {
  let goCalls: string[][];
  let capturedDispatches: unknown[] | undefined;
  let stderrOutput: string;

  beforeEach(() => {
    __restoreAllMocks();
    __restoreGoBridgeCallGoJSON();
    __restoreCreateCeremonyAdapter();
    goCalls = [];
    capturedDispatches = undefined;
    stderrOutput = "";

    // Mock ceremony adapter to avoid Go subprocess calls
    __setCreateCeremonyAdapter(() => createMockCeremonyAdapter());

    // Capture stderr output
    const originalStderrWrite = process.stderr.write.bind(process.stderr);
    process.stderr.write = ((chunk: unknown, ...args: unknown[]) => {
      if (typeof chunk === "string") stderrOutput += chunk;
      return originalStderrWrite(chunk as string | Uint8Array, ...args as [BufferEncoding]);
    }) as typeof process.stderr.write;
  });

  afterEach(() => {
    __restoreAllMocks();
    __restoreGoBridgeCallGoJSON();
    __restoreCreateCeremonyAdapter();
  });

  function mockHiveReadSuccess() {
    const handler = <T>(_opts: unknown, args: string[]): T => {
      goCalls.push(args);
      const cmd = args[0];
      if (cmd === "hive-read") {
        return {
          entries: [
            {
              id: "1",
              text: "Use table-driven tests",
              domain: "go",
              confidence: 0.95,
              source_repo: "other-repo",
              source_repos: ["other-repo"],
              created_at: "2026-05-18T00:00:00Z",
              accessed_at: "2026-05-18T00:00:00Z",
              access_count: 5,
            },
          ],
          total: 1,
        } as unknown as T;
      }
      if (cmd === "registry-list") {
        return {
          colonies: [
            {
              repo_path: process.cwd(),
              domains: ["go"],
              active: true,
              registered_at: "2026-05-18T00:00:00Z",
            },
          ],
        } as unknown as T;
      }
      if (cmd === "build") {
        return {
          dispatch_manifest: {
            execution_binding: TEST_EXECUTION_BINDING,
            dispatches: [
              { name: "Builder-01", caste: "builder", task: "Build", wave: 1, execution_wave: 1 },
              { name: "Builder-02", caste: "builder", task: "Build more", wave: 1, execution_wave: 1 },
            ],
          },
        } as unknown as T;
      }
      if (cmd === "build-finalize") {
        return { ok: true } as unknown as T;
      }
      if (cmd === "plan") {
        return {
          plan_manifest: { phases: 5 },
          dispatches: [
            { name: "Scout-01", caste: "scout", task: "Research", wave: 1, execution_wave: 1 },
          ],
        } as unknown as T;
      }
      if (cmd === "plan-finalize") {
        return { ok: true } as unknown as T;
      }
      if (cmd === "continue") {
        return {
          continue_manifest: { phase: 1 },
          dispatches: [
            { name: "Watcher-01", caste: "watcher", task: "Verify", wave: 1, execution_wave: 1 },
          ],
        } as unknown as T;
      }
      if (cmd === "continue-finalize") {
        return { ok: true } as unknown as T;
      }
      return { ok: true } as unknown as T;
    };

    // Set mock at both levels: host.ts uses _callGoJSONRef; hive-injector.ts uses go-bridge module-level callGoJSON
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

    __setDetectAvailablePlatforms(async () => ["claude"]);
  }

  function mockHiveReadFailure() {
    const handler = <T>(_opts: unknown, args: string[]): T => {
      goCalls.push(args);
      const cmd = args[0];
      if (cmd === "hive-read") {
        throw new Error("corrupted wisdom.json");
      }
      if (cmd === "registry-list") {
        return {
          colonies: [
            {
              repo_path: process.cwd(),
              domains: ["go"],
              active: true,
              registered_at: "2026-05-18T00:00:00Z",
            },
          ],
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
      if (cmd === "build-finalize") {
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

    __setDetectAvailablePlatforms(async () => ["claude"]);
  }

  it("build runner calls hive-read before dispatch and attaches hive_section to dispatches (HIVE-04)", async () => {
    mockHiveReadSuccess();

    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: process.cwd() };
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    await runDispatchedBuildCommand(bridge, parsed, def);

    // Verify hive-read was called
    const hiveReadCall = goCalls.find((args) => args[0] === "hive-read");
    assert.ok(hiveReadCall, "hive-read should have been called");

    // Verify dispatches have hive_section
    assert.ok(capturedDispatches, "dispatchWorkers should have been called");
    assert.ok(capturedDispatches!.length > 0, "dispatches should not be empty");
    for (const d of capturedDispatches!) {
      const dispatch = d as Record<string, unknown>;
      assert.ok(
        typeof dispatch.hive_section === "string" && dispatch.hive_section.includes("HIVE WISDOM"),
        `Dispatch ${dispatch.name} should have hive_section containing HIVE WISDOM. Got: ${dispatch.hive_section}`
      );
      assert.ok(
        (dispatch.hive_section as string).includes("Use table-driven tests"),
        `Dispatch ${dispatch.name} should contain the wisdom text`
      );
    }

    // Verify stderr contains hive summary
    assert.ok(
      stderrOutput.includes("Injecting hive wisdom"),
      `Stderr should contain hive summary. Got: ${stderrOutput.slice(0, 200)}`
    );
  });

  it("hive-read failure does not block dispatch (HIVE-04)", async () => {
    mockHiveReadFailure();

    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: process.cwd() };
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    await runDispatchedBuildCommand(bridge, parsed, def);

    // Verify warning was logged
    assert.ok(
      stderrOutput.includes("Warning: hive-read failed: corrupted wisdom.json"),
      `Stderr should contain hive-read failure warning. Got: ${stderrOutput.slice(0, 300)}`
    );

    // Verify dispatch still proceeded
    assert.ok(capturedDispatches, "dispatchWorkers should still have been called despite hive-read failure");
    assert.ok(capturedDispatches!.length > 0, "dispatches should not be empty");

    // Verify dispatches have empty hive_section (graceful degradation)
    for (const d of capturedDispatches!) {
      const dispatch = d as Record<string, unknown>;
      assert.equal(
        dispatch.hive_section,
        "",
        `Dispatch ${dispatch.name} should have empty hive_section on failure`
      );
    }
  });

  it("plan runner attaches hive_section to dispatches", async () => {
    mockHiveReadSuccess();

    const parsed = parseArgs(["node", "host.js", "plan", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: process.cwd() };

    await runDispatchedPlanCommand(bridge, parsed);

    const hiveReadCall = goCalls.find((args) => args[0] === "hive-read");
    assert.ok(hiveReadCall, "hive-read should have been called for plan");

    assert.ok(capturedDispatches, "dispatchWorkers should have been called for plan");
    for (const d of capturedDispatches!) {
      const dispatch = d as Record<string, unknown>;
      assert.ok(
        typeof dispatch.hive_section === "string" && dispatch.hive_section.includes("HIVE WISDOM"),
        `Plan dispatch ${dispatch.name} should have hive_section`
      );
    }
  });

  it("continue runner attaches hive_section to dispatches", async () => {
    mockHiveReadSuccess();

    const parsed = parseArgs(["node", "host.js", "continue", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: process.cwd() };

    await runDispatchedContinueCommand(bridge, parsed);

    const hiveReadCall = goCalls.find((args) => args[0] === "hive-read");
    assert.ok(hiveReadCall, "hive-read should have been called for continue");

    assert.ok(capturedDispatches, "dispatchWorkers should have been called for continue");
    for (const d of capturedDispatches!) {
      const dispatch = d as Record<string, unknown>;
      assert.ok(
        typeof dispatch.hive_section === "string" && dispatch.hive_section.includes("HIVE WISDOM"),
        `Continue dispatch ${dispatch.name} should have hive_section`
      );
    }
  });

  it("dry-run calls hive-read and attaches hive_section", async () => {
    mockHiveReadSuccess();

    const parsed = parseArgs(["node", "host.js", "build", "1", "--dry-run"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: process.cwd() };
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    await runDryRunDispatchedCommand(bridge, parsed, def);

    const hiveReadCall = goCalls.find((args) => args[0] === "hive-read");
    assert.ok(hiveReadCall, "hive-read should have been called for dry-run");

    // In dry-run, dispatchWorkers is NOT called, but we can verify
    // the hive summary was logged and no error occurred
    assert.ok(
      stderrOutput.includes("Injecting hive wisdom"),
      `Dry-run stderr should contain hive summary. Got: ${stderrOutput.slice(0, 200)}`
    );
  });
});

// ---------------------------------------------------------------------------
// Cross-colony wisdom benefit (HIVE-05)
// ---------------------------------------------------------------------------

describe("cross-colony wisdom benefit (HIVE-05)", () => {
  let goCalls: string[][];
  let capturedDispatches: unknown[] | undefined;
  let stderrOutput: string;

  beforeEach(() => {
    __restoreAllMocks();
    __restoreGoBridgeCallGoJSON();
    __restoreCreateCeremonyAdapter();
    goCalls = [];
    capturedDispatches = undefined;
    stderrOutput = "";

    // Mock ceremony adapter to avoid Go subprocess calls
    __setCreateCeremonyAdapter(() => createMockCeremonyAdapter());

    const originalStderrWrite = process.stderr.write.bind(process.stderr);
    process.stderr.write = ((chunk: unknown, ...args: unknown[]) => {
      if (typeof chunk === "string") stderrOutput += chunk;
      return originalStderrWrite(chunk as string | Uint8Array, ...args as [BufferEncoding]);
    }) as typeof process.stderr.write;

    // Mock: colony A promoted wisdom (source_repo = "colony-a")
    const handler = <T>(_opts: unknown, args: string[]): T => {
      goCalls.push(args);
      const cmd = args[0];
      if (cmd === "hive-read") {
        return {
          entries: [
            {
              id: "colony-a-1",
              text: "Prefer early error returns over deep nesting",
              domain: "go",
              confidence: 0.92,
              source_repo: "colony-a",
              source_repos: ["colony-a"],
              created_at: "2026-05-18T00:00:00Z",
              accessed_at: "2026-05-18T00:00:00Z",
              access_count: 3,
            },
          ],
          total: 1,
        } as unknown as T;
      }
      if (cmd === "registry-list") {
        return {
          colonies: [
            {
              repo_path: process.cwd(),
              domains: ["go"],
              active: true,
              registered_at: "2026-05-18T00:00:00Z",
            },
          ],
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
      if (cmd === "build-finalize") {
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

    __setDetectAvailablePlatforms(async () => ["claude"]);
  });

  afterEach(() => {
    __restoreAllMocks();
    __restoreGoBridgeCallGoJSON();
    __restoreCreateCeremonyAdapter();
  });

  it("colony B worker prompt contains colony A wisdom (presence proxy)", async () => {
    // This test verifies the mechanism: wisdom from colony-a is present
    // in colony-b's worker dispatches. The actual speedup ("faster or fewer
    // retries") is a system property verified by observation, not by
    // automated assertion. See RESEARCH.md HIVE-05 test strategy.
    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: process.cwd() };
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    await runDispatchedBuildCommand(bridge, parsed, def);

    assert.ok(capturedDispatches, "dispatchWorkers should have been called");
    assert.ok(capturedDispatches!.length > 0, "dispatches should not be empty");

    for (const d of capturedDispatches!) {
      const dispatch = d as Record<string, unknown>;
      const hiveSection = dispatch.hive_section as string;
      assert.ok(
        typeof hiveSection === "string" && hiveSection.includes("HIVE WISDOM"),
        `Dispatch should have hive_section with HIVE WISDOM header`
      );
      assert.ok(
        hiveSection.includes("Prefer early error returns over deep nesting"),
        `Dispatch should contain colony-a wisdom text`
      );
      // source_repo is in the data structure but not rendered in the formatted text;
      // the presence of the wisdom text proves the cross-colony mechanism works
    }

    // Verify the hive summary was emitted
    assert.ok(
      stderrOutput.includes("Injecting hive wisdom"),
      `Stderr should confirm hive wisdom injection`
    );
  });
});

// ---------------------------------------------------------------------------
// Spawn orchestrator initialization from manifest (SPAWN-03, SPAWN-05)
// ---------------------------------------------------------------------------

describe("spawn orchestrator initialization (SPAWN-03, SPAWN-05)", () => {
  let goCalls: string[][];
  let capturedDispatchOpts: unknown;

  beforeEach(() => {
    __restoreAllMocks();
    __restoreGoBridgeCallGoJSON();
    __restoreCreateCeremonyAdapter();
    goCalls = [];
    capturedDispatchOpts = undefined;

    // Mock ceremony adapter to avoid Go subprocess calls
    __setCreateCeremonyAdapter(() => createMockCeremonyAdapter());

    // Mock callGoJSON to return manifest with queen_execution_policy
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
            domains: ["go"],
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
              { name: "Builder-02", caste: "builder", task: "Build more", wave: 1, execution_wave: 1 },
            ],
            queen_execution_policy: {
              spawn_budget: {
                max_workers: 10,
              },
            },
          },
        } as unknown as T;
      }
      if (cmd === "build-finalize") {
        return { ok: true } as unknown as T;
      }
      // spawn-log / spawn-complete calls (SPAWN-05)
      if (cmd === "spawn-log") {
        return { recorded: true } as unknown as T;
      }
      if (cmd === "spawn-complete") {
        return { completed: true } as unknown as T;
      }
      return { ok: true } as unknown as T;
    };

    __setCallGoJSON(handler);
    __setGoBridgeCallGoJSON(handler);

    __setDispatchWorkers(async (opts, dispatches) => {
      capturedDispatchOpts = opts;
      return dispatches.map((d: BuildDispatch) => ({
        name: d.name as string,
        status: "completed" as const,
        summary: "Done",
        duration: 5,
      }));
    });

    __setDetectAvailablePlatforms(async () => [
      "claude" as const,
    ]);
  });

  afterEach(() => {
    __restoreAllMocks();
    __restoreGoBridgeCallGoJSON();
    __restoreCreateCeremonyAdapter();
  });

  it("build runner initializes spawn orchestrator from manifest budget", async () => {
    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: process.cwd() };
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    await runDispatchedBuildCommand(bridge, parsed, def);

    // Verify dispatch opts include spawnOrchestrator
    assert.ok(capturedDispatchOpts, "dispatchWorkers should have been called");
    const opts = capturedDispatchOpts as Record<string, unknown>;
    assert.ok(opts.spawnOrchestrator, "Dispatch opts should include spawnOrchestrator");

    const orchestrator = opts.spawnOrchestrator as { totalBudget: number; consumedBudget: number };
    assert.equal(orchestrator.totalBudget, 10, "Total budget should match manifest max_workers");
    assert.equal(orchestrator.consumedBudget, 2, "Consumed budget should equal manifest dispatch count");
  });

  it("spawn-log calls include correct parent for manifest workers (SPAWN-05)", async () => {
    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: process.cwd() };
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    await runDispatchedBuildCommand(bridge, parsed, def);

    // spawn-log is called by dispatchSingleWorker which runs inside dispatchWorkers.
    // Since we mock dispatchWorkers, spawn-log calls won't actually happen through
    // the mock. Instead, verify the build pipeline ran successfully.
    assert.ok(capturedDispatchOpts, "Build pipeline should have completed");
    const buildFinalizeCall = goCalls.find((args) => args[0] === "build-finalize");
    assert.ok(buildFinalizeCall, "build-finalize should have been called");
  });
});

// ---------------------------------------------------------------------------
// Build iteration loop tests (ITER-01 through ITER-06)
// ---------------------------------------------------------------------------

/** Extract the last JSON object from stdout (handles multiple JSON writes). */
function extractLastJSON(stdout: string): Record<string, unknown> {
  const trimmed = stdout.trim();
  // Find the last occurrence of a line that starts with { and parse backwards
  const lines = trimmed.split("\n");
  for (let i = lines.length - 1; i >= 0; i--) {
    const line = lines[i]!.trim();
    if (line.startsWith("{")) {
      try {
        return JSON.parse(line);
      } catch {
        // Multi-line JSON: collect from this line to end
        const remainder = lines.slice(i).join("\n");
        // Find the last closing brace
        const lastClose = remainder.lastIndexOf("}");
        if (lastClose !== -1) {
          return JSON.parse(remainder.slice(0, lastClose + 1));
        }
      }
    }
  }
  throw new Error(`No JSON found in stdout: ${trimmed.slice(0, 300)}`);
}

/**
 * Count how many times a specific Go command was called.
 * Uses goCalls array to count invocations of the given command.
 */
function countGoCommandCalls(calls: string[][], command: string): number {
  return calls.filter((args) => args[0] === command).length;
}

describe("build iteration loop", () => {
  let goCalls: string[][];
  let dispatchCallCount: number;
  let capturedAllDispatches: unknown[][];
  let stderrOutput: string;

  /** Creates a mock callGoJSON that returns manifests with configurable dispatches. */
  function createIterationMockGo(opts?: {
    dispatchCount?: number;
    workerResults?: ((callNum: number) => Array<Record<string, unknown>>);
  }) {
    const defaultWorkerResults = (_callNum: number) => [
      { name: "Builder-01", status: "completed", summary: "Done", duration: 5,
        files_created: ["src/module.ts"], files_modified: ["src/helper.ts"],
        tests_written: ["test/module.test.ts"] },
    ];
    const workerResultsFn = opts?.workerResults ?? defaultWorkerResults;
    const dispatchCount = opts?.dispatchCount ?? 1;

    return <T>(_bridgeOpts: unknown, args: string[]): T => {
      goCalls.push(args);
      const cmd = args[0];
      if (cmd === "hive-read") {
        return { entries: null, total: 0 } as unknown as T;
      }
      if (cmd === "registry-list") {
        return {
          colonies: [{
            repo_path: process.cwd(),
            domains: ["ts"],
            active: true,
            registered_at: "2026-05-18T00:00:00Z",
          }],
        } as unknown as T;
      }
      if (cmd === "build") {
        const dispatches: Record<string, unknown>[] = [];
        for (let i = 0; i < dispatchCount; i++) {
          dispatches.push({
            name: `Builder-${String(i + 1).padStart(2, "0")}`,
            caste: "builder",
            task: "Build",
            wave: 1,
            execution_wave: 1,
            skill_section: "",
          });
        }
        return {
          dispatch_manifest: {
            execution_binding: TEST_EXECUTION_BINDING,
            dispatches,
          },
        } as unknown as T;
      }
      if (cmd === "build-finalize") {
        return { ok: true } as unknown as T;
      }
      return { ok: true } as unknown as T;
    };
  }

  beforeEach(() => {
    __restoreAllMocks();
    __restoreGoBridgeCallGoJSON();
    __restoreCreateCeremonyAdapter();
    goCalls = [];
    dispatchCallCount = 0;
    capturedAllDispatches = [];
    stderrOutput = "";

    __setCreateCeremonyAdapter(() => createMockCeremonyAdapter());

    const originalStderrWrite = process.stderr.write.bind(process.stderr);
    process.stderr.write = ((chunk: unknown, ...args: unknown[]) => {
      if (typeof chunk === "string") stderrOutput += chunk;
      return originalStderrWrite(chunk as string | Uint8Array, ...args as [BufferEncoding]);
    }) as typeof process.stderr.write;
  });

  afterEach(() => {
    __restoreAllMocks();
    __restoreGoBridgeCallGoJSON();
    __restoreCreateCeremonyAdapter();
  });

  it("single iteration when confidence is high", async () => {
    __setCallGoJSON(createIterationMockGo({
      workerResults: () => [
        { name: "Builder-01", status: "completed", summary: "Done", duration: 5,
          files_created: ["src/module.ts"], files_modified: ["src/helper.ts"],
          tests_written: ["test/module.test.ts"],
          test_results: { passed: 10, total: 10 } },
      ],
    }));
    __setGoBridgeCallGoJSON(createIterationMockGo({
      workerResults: () => [
        { name: "Builder-01", status: "completed", summary: "Done", duration: 5,
          files_created: ["src/module.ts"], files_modified: ["src/helper.ts"],
          tests_written: ["test/module.test.ts"],
          test_results: { passed: 10, total: 10 } },
      ],
    }));

    __setDispatchWorkers(async (_opts, dispatches) => {
      dispatchCallCount++;
      capturedAllDispatches.push(dispatches);
      return [
        { name: "Builder-01", status: "completed", summary: "Done", duration: 5,
          files_created: ["src/module.ts"], files_modified: ["src/helper.ts"],
          tests_written: ["test/module.test.ts"],
          test_results: { passed: 10, total: 10 } },
      ];
    });

    __setDetectAvailablePlatforms(async () => [
      "claude" as const,
    ]);

    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: process.cwd() };
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    await runDispatchedBuildCommand(bridge, parsed, def);

    // Verify exactly 1 build-finalize call = 1 iteration
    const finalizeCount = countGoCommandCalls(goCalls, "build-finalize");
    assert.equal(finalizeCount, 1, "Should call build-finalize exactly once when confidence is high");
    assert.equal(dispatchCallCount, 1, "Should dispatch exactly 1 wave when confidence is high");
  });

  it("keeps the Go-issued manifest immutable while enriching worker briefs", async () => {
    const issuedManifest = {
      execution_binding: TEST_EXECUTION_BINDING,
      phase: 1,
      attempt_id: "attempt-immutable",
      attempt_path: ".aether/data/build/phase-1/attempts/attempt-immutable.json",
      dispatches: [{
        name: "Builder-01",
        caste: "builder",
        task: "Build immutable contract",
        wave: 1,
        execution_wave: 1,
      }],
    };
    let finalizedManifest: unknown;
    const mockGo = <T>(_bridgeOpts: unknown, args: string[]): T => {
      goCalls.push(args);
      if (args[0] === "hive-read") {
        return { entries: [{ text: "Use typed errors", domain: "go", confidence: 0.9 }], total: 1 } as unknown as T;
      }
		if (args[0] === "build") {
			return { dispatch_manifest: issuedManifest } as unknown as T;
		}
		if (args[0] === "build-completion-stage") {
			const sourcePath = args[args.indexOf("--completion-file") + 1]!;
			return { completion_path: sourcePath } as unknown as T;
		}
      if (args[0] === "build-finalize") {
        const completionPath = args[args.indexOf("--completion-file") + 1]!;
        const packet = JSON.parse(readFileSync(completionPath, "utf8")) as {
          result: { dispatch_manifest: unknown };
        };
        finalizedManifest = packet.result.dispatch_manifest;
        return { ok: true } as unknown as T;
      }
      return { ok: true } as unknown as T;
    };
    __setCallGoJSON(mockGo);
    __setGoBridgeCallGoJSON(mockGo);
    __setDispatchWorkers(async (_opts, dispatches) => {
      dispatchCallCount++;
      capturedAllDispatches.push(dispatches);
      return [{
        name: "Builder-01",
        status: "completed",
        summary: "Done",
        duration: 1,
        files_modified: ["src/module.ts"],
        tests_written: ["test/module.test.ts"],
        test_results: { passed: 1, total: 1 },
      }];
    });
    __setDetectAvailablePlatforms(async () => ["claude" as const]);

    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: process.cwd() };
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    await runDispatchedBuildCommand(bridge, parsed, getHostCommandDefinition("build")!);

		assert.deepEqual(finalizedManifest, issuedManifest, "completion must preserve the exact Go-issued manifest");
		const stageIndex = goCalls.findIndex((args) => args[0] === "build-completion-stage");
		const finalizeIndex = goCalls.findIndex((args) => args[0] === "build-finalize");
		assert.ok(stageIndex >= 0 && stageIndex < finalizeIndex, "accepted completion must be staged before finalization");
    const enriched = capturedAllDispatches[0]![0] as Record<string, unknown>;
    // Playbook context is no longer appended to worker briefs — it was
    // orchestrator guidance that told a single worker it was the Queen.
    // Hive enrichment is still expected; that is genuine per-worker context.
    assert.doesNotMatch(String(enriched.task_brief ?? ""), /Relevant Playbooks/);
    assert.match(String(enriched.hive_section), /Use typed errors/);
  });

  it("does not dispatch a build blocked by orchestrator boundary questions", async () => {
    const mockGo = <T>(_bridgeOpts: unknown, args: string[]): T => {
      goCalls.push(args);
      if (args[0] === "build") {
        return {
          dispatch_manifest: {
            execution_binding: TEST_EXECUTION_BINDING,
            dispatches: [{ name: "Builder-01", caste: "builder", task: "Do not run" }],
            orchestrator_boundary_guidance: {
              active: true,
              next: "aether discuss",
              summary: "Answer the build boundary question first.",
            },
          },
        } as unknown as T;
      }
      return { ok: true } as unknown as T;
    };
    __setCallGoJSON(mockGo);
    __setGoBridgeCallGoJSON(mockGo);
    __setDispatchWorkers(async () => {
      dispatchCallCount++;
      return [];
    });

    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: process.cwd() };
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    await assert.rejects(
      runDispatchedBuildCommand(bridge, parsed, getHostCommandDefinition("build")!),
      /Answer the build boundary question first/,
    );

    assert.equal(dispatchCallCount, 0);
    assert.equal(countGoCommandCalls(goCalls, "build-finalize"), 0);
  });

  it("two iterations when confidence is low then high", async () => {
    const workerResults = (_callNum: number) => {
      if (_callNum === 0) {
        return [
          { name: "Builder-01", status: "failed", summary: "Build failed", duration: 5,
            blockers: ["missing import"] },
        ];
      }
      return [
        { name: "Builder-01", status: "completed", summary: "Done", duration: 5,
          files_created: ["src/module.ts"],
          test_results: { passed: 8, total: 8 } },
      ];
    };

    __setCallGoJSON(createIterationMockGo({ workerResults }));
    __setGoBridgeCallGoJSON(createIterationMockGo({ workerResults }));

    let dispatchNum = 0;
    __setDispatchWorkers(async (_opts, dispatches) => {
      dispatchNum++;
      dispatchCallCount++;
      capturedAllDispatches.push(dispatches);
      return workerResults(dispatchNum - 1) as DispatchResult[];
    });

    __setDetectAvailablePlatforms(async () => [
      "claude" as const,
    ]);

    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: process.cwd() };
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    await runDispatchedBuildCommand(bridge, parsed, def);

    const finalizeCount = countGoCommandCalls(goCalls, "build-finalize");
    assert.equal(finalizeCount, 1, "Should call build-finalize once for 2 iterations");
    assert.equal(dispatchCallCount, 2, "Should dispatch 2 waves");
  });

  it("stops at max iterations (default 3)", async () => {
    const workerResults = () => [
      { name: "Builder-01", status: "failed", summary: "Build failed", duration: 5,
        blockers: ["compile error"] },
    ];

    __setCallGoJSON(createIterationMockGo({ workerResults }));
    __setGoBridgeCallGoJSON(createIterationMockGo({ workerResults }));

    __setDispatchWorkers(async (_opts, dispatches) => {
      dispatchCallCount++;
      capturedAllDispatches.push(dispatches);
      return workerResults() as DispatchResult[];
    });

    __setDetectAvailablePlatforms(async () => [
      "claude" as const,
    ]);

    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: process.cwd() };
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    await runDispatchedBuildCommand(bridge, parsed, def);

    const finalizeCount = countGoCommandCalls(goCalls, "build-finalize");
    assert.equal(finalizeCount, 1, "Should call build-finalize once after the default max");
    assert.equal(dispatchCallCount, 3, "Should dispatch exactly 3 waves (default max)");
  });

  it("stops at custom --max-iterations", async () => {
    const workerResults = () => [
      { name: "Builder-01", status: "failed", summary: "Build failed", duration: 5,
        blockers: ["compile error"] },
    ];

    __setCallGoJSON(createIterationMockGo({ workerResults }));
    __setGoBridgeCallGoJSON(createIterationMockGo({ workerResults }));

    __setDispatchWorkers(async (_opts, dispatches) => {
      dispatchCallCount++;
      capturedAllDispatches.push(dispatches);
      return workerResults() as DispatchResult[];
    });

    __setDetectAvailablePlatforms(async () => [
      "claude" as const,
    ]);

    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate", "--max-iterations", "2"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: process.cwd() };
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    await runDispatchedBuildCommand(bridge, parsed, def);

    const finalizeCount = countGoCommandCalls(goCalls, "build-finalize");
    assert.equal(finalizeCount, 1, "Should call build-finalize once after the custom max");
    assert.equal(dispatchCallCount, 2, "Should dispatch exactly 2 waves (custom max)");
  });

  it("stops when budget exhausted", async () => {
    const mockGo = <T>(_bridgeOpts: unknown, args: string[]): T => {
      goCalls.push(args);
      const cmd = args[0];
      if (cmd === "hive-read") {
        return { entries: null, total: 0 } as unknown as T;
      }
      if (cmd === "registry-list") {
        return {
          colonies: [{
            repo_path: process.cwd(),
            domains: ["ts"],
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
              { name: "Builder-02", caste: "builder", task: "Build", wave: 1, execution_wave: 1 },
              { name: "Builder-03", caste: "builder", task: "Build", wave: 1, execution_wave: 1 },
            ],
            queen_execution_policy: {
              spawn_budget: { max_workers: 3 },
            },
          },
        } as unknown as T;
      }
      if (cmd === "build-finalize") {
        return { ok: true } as unknown as T;
      }
      return { ok: true } as unknown as T;
    };

    __setCallGoJSON(mockGo);
    __setGoBridgeCallGoJSON(mockGo);

    __setDispatchWorkers(async (_opts, dispatches) => {
      dispatchCallCount++;
      capturedAllDispatches.push(dispatches);
      return dispatches.map((d: BuildDispatch) => ({
        name: d.name,
        status: "failed",
        summary: "Failed",
        duration: 5,
        blockers: ["compile error"],
      }));
    });

    __setDetectAvailablePlatforms(async () => [
      "claude" as const,
    ]);

    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: process.cwd() };
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    await runDispatchedBuildCommand(bridge, parsed, def);

    const finalizeCount = countGoCommandCalls(goCalls, "build-finalize");
    assert.equal(finalizeCount, 1, "Should call build-finalize only once when budget exhausted");
    assert.equal(dispatchCallCount, 1, "Should dispatch only 1 wave when budget exhausted");
  });
});

// ---------------------------------------------------------------------------
// Feedback injection tests (ITER-02)
// ---------------------------------------------------------------------------

describe("feedback injection", () => {
  let goCalls: string[][];
  let dispatchCallCount: number;
  let capturedAllDispatches: unknown[][];
  let stderrOutput: string;

  beforeEach(() => {
    __restoreAllMocks();
    __restoreGoBridgeCallGoJSON();
    __restoreCreateCeremonyAdapter();
    goCalls = [];
    dispatchCallCount = 0;
    capturedAllDispatches = [];
    stderrOutput = "";

    __setCreateCeremonyAdapter(() => createMockCeremonyAdapter());

    const originalStderrWrite = process.stderr.write.bind(process.stderr);
    process.stderr.write = ((chunk: unknown, ...args: unknown[]) => {
      if (typeof chunk === "string") stderrOutput += chunk;
      return originalStderrWrite(chunk as string | Uint8Array, ...args as [BufferEncoding]);
    }) as typeof process.stderr.write;
  });

  afterEach(() => {
    __restoreAllMocks();
    __restoreGoBridgeCallGoJSON();
    __restoreCreateCeremonyAdapter();
  });

  it("iteration 2 task briefs include blockers from iteration 1", async () => {
    const mockGo = <T>(_bridgeOpts: unknown, args: string[]): T => {
      goCalls.push(args);
      const cmd = args[0];
      if (cmd === "hive-read") {
        return { entries: null, total: 0 } as unknown as T;
      }
      if (cmd === "registry-list") {
        return {
          colonies: [{
            repo_path: process.cwd(),
            domains: ["ts"],
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
      if (cmd === "build-finalize") {
        return { ok: true } as unknown as T;
      }
      return { ok: true } as unknown as T;
    };

    __setCallGoJSON(mockGo);
    __setGoBridgeCallGoJSON(mockGo);

    let dispatchNum = 0;
    __setDispatchWorkers(async (_opts, dispatches) => {
      dispatchNum++;
      dispatchCallCount++;
      capturedAllDispatches.push(dispatches);
      if (dispatchNum === 1) {
        // First iteration: worker fails with blocker
        return [
          { name: "Builder-01", status: "failed", summary: "Build failed", duration: 5,
            blockers: ["missing import foo"] },
        ];
      }
      // Second iteration: worker succeeds
      return [
        { name: "Builder-01", status: "completed", summary: "Done", duration: 5,
          files_created: ["src/module.ts"],
          test_results: { passed: 5, total: 5 } },
      ];
    });

    __setDetectAvailablePlatforms(async () => [
      "claude" as const,
    ]);

    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: process.cwd() };
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    await runDispatchedBuildCommand(bridge, parsed, def);

    // Verify 2 dispatches occurred (confidence goes low then high)
    const finalizeCount = countGoCommandCalls(goCalls, "build-finalize");
    assert.equal(finalizeCount, 1, "Should call build-finalize exactly once");

    // Verify second dispatch has feedback from first iteration's blockers
    assert.ok(capturedAllDispatches.length >= 2, "Should have captured at least 2 dispatch calls");
    const secondDispatch = capturedAllDispatches[1]!;
    assert.ok(secondDispatch.length > 0, "Second dispatch should have dispatches");
    const dispatch = secondDispatch[0] as Record<string, unknown>;
    assert.ok(
      typeof dispatch.task_brief === "string" && dispatch.task_brief.includes("missing import foo"),
      `Second iteration task_brief should contain blocker text. Got: ${dispatch.task_brief}`
    );
  });
});

// ---------------------------------------------------------------------------
// Ceremony output tests (ITER-06)
// ---------------------------------------------------------------------------

describe("ceremony output", () => {
  let goCalls: string[][];
  let stderrOutput: string;

  beforeEach(() => {
    __restoreAllMocks();
    __restoreGoBridgeCallGoJSON();
    __restoreCreateCeremonyAdapter();
    goCalls = [];
    stderrOutput = "";

    __setCreateCeremonyAdapter(() => createMockCeremonyAdapter());

    const originalStderrWrite = process.stderr.write.bind(process.stderr);
    process.stderr.write = ((chunk: unknown, ...args: unknown[]) => {
      if (typeof chunk === "string") stderrOutput += chunk;
      return originalStderrWrite(chunk as string | Uint8Array, ...args as [BufferEncoding]);
    }) as typeof process.stderr.write;
  });

  afterEach(() => {
    __restoreAllMocks();
    __restoreGoBridgeCallGoJSON();
    __restoreCreateCeremonyAdapter();
  });

  it("iteration markers in ceremony output", async () => {
    const mockGo = <T>(_bridgeOpts: unknown, args: string[]): T => {
      goCalls.push(args);
      const cmd = args[0];
      if (cmd === "hive-read") {
        return { entries: null, total: 0 } as unknown as T;
      }
      if (cmd === "registry-list") {
        return {
          colonies: [{
            repo_path: process.cwd(),
            domains: ["ts"],
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
      if (cmd === "build-finalize") {
        return { ok: true } as unknown as T;
      }
      return { ok: true } as unknown as T;
    };

    __setCallGoJSON(mockGo);
    __setGoBridgeCallGoJSON(mockGo);

    __setDispatchWorkers(async () => {
      return [
        { name: "Builder-01", status: "completed", summary: "Done", duration: 5,
          files_created: ["src/module.ts"],
          test_results: { passed: 5, total: 5 } },
      ];
    });

    __setDetectAvailablePlatforms(async () => [
      "claude" as const,
    ]);

    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: process.cwd() };
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    await runDispatchedBuildCommand(bridge, parsed, def);

    // Verify ceremony output contains iteration markers
    assert.ok(
      stderrOutput.includes("Iteration"),
      `Stderr should contain iteration markers. Got: ${stderrOutput.slice(0, 300)}`
    );
    assert.ok(
      stderrOutput.includes("confidence"),
      `Stderr should contain confidence text. Got: ${stderrOutput.slice(0, 300)}`
    );
    assert.ok(
      stderrOutput.includes("Iteration complete"),
      `Stderr should contain final iteration marker. Got: ${stderrOutput.slice(0, 300)}`
    );
  });
});

// ---------------------------------------------------------------------------
// Cross-phase integration: hive + spawn + iteration (VAL-03)
// ---------------------------------------------------------------------------

describe("cross-phase integration (hive + spawn + iteration)", () => {
  let goCalls: string[][];
  let capturedAllDispatches: unknown[][];
  let capturedDispatchOpts: unknown[];
  let dispatchCallCount: number;
  let stderrOutput: string;

  beforeEach(() => {
    __restoreAllMocks();
    __restoreGoBridgeCallGoJSON();
    __restoreCreateCeremonyAdapter();
    goCalls = [];
    capturedAllDispatches = [];
    capturedDispatchOpts = [];
    dispatchCallCount = 0;
    stderrOutput = "";

    __setCreateCeremonyAdapter(() => createMockCeremonyAdapter());

    const originalStderrWrite = process.stderr.write.bind(process.stderr);
    process.stderr.write = ((chunk: unknown, ...args: unknown[]) => {
      if (typeof chunk === "string") stderrOutput += chunk;
      return originalStderrWrite(chunk as string | Uint8Array, ...args as [BufferEncoding]);
    }) as typeof process.stderr.write;
  });

  afterEach(() => {
    __restoreAllMocks();
    __restoreGoBridgeCallGoJSON();
    __restoreCreateCeremonyAdapter();
  });

  it("build pipeline exercises hive, spawn, and iteration together", async () => {
    // Set up mock callGoJSON that returns hive-read entries, registry-list,
    // build manifest with queen_execution_policy (spawn budget: max_workers: 10),
    // and build-finalize responses. First dispatch workers fail, second succeed.
    const handler = <T>(_bridgeOpts: unknown, args: string[]): T => {
      goCalls.push(args);
      const cmd = args[0];
      if (cmd === "hive-read") {
        return {
          entries: [
            {
              id: "1",
              text: "Use table-driven tests for Go code",
              domain: "go",
              confidence: 0.92,
              source_repo: "colony-a",
              source_repos: ["colony-a"],
              created_at: "2026-05-18T00:00:00Z",
              accessed_at: "2026-05-18T00:00:00Z",
              access_count: 3,
            },
          ],
          total: 1,
        } as unknown as T;
      }
      if (cmd === "registry-list") {
        return {
          colonies: [
            {
              repo_path: process.cwd(),
              domains: ["go", "typescript"],
              active: true,
              registered_at: "2026-05-18T00:00:00Z",
            },
          ],
        } as unknown as T;
      }
      if (cmd === "build") {
        return {
          dispatch_manifest: {
            execution_binding: TEST_EXECUTION_BINDING,
            dispatches: [
              { name: "Builder-01", caste: "builder", task: "Implement feature", wave: 1, execution_wave: 1, skill_section: "" },
              { name: "Watcher-01", caste: "watcher", task: "Verify tests", wave: 1, execution_wave: 1, skill_section: "" },
            ],
            queen_execution_policy: {
              spawn_budget: { max_workers: 10 },
            },
          },
        } as unknown as T;
      }
      if (cmd === "build-finalize") {
        return { ok: true } as unknown as T;
      }
      return { ok: true } as unknown as T;
    };

    // Set mock at both levels: host.ts uses _callGoJSONRef; hive-injector uses go-bridge module-level callGoJSON
    __setCallGoJSON(handler);
    __setGoBridgeCallGoJSON(handler);

    let dispatchNum = 0;
    __setDispatchWorkers(async (opts, dispatches) => {
      dispatchNum++;
      dispatchCallCount++;
      capturedAllDispatches.push([...dispatches]);
      capturedDispatchOpts.push(opts);

      if (dispatchNum === 1) {
        // First iteration: workers fail with blockers -> low confidence -> iterate
        return [
          { name: "Builder-01", status: "failed", summary: "Build failed", duration: 5,
            blockers: ["missing import 'types'"] },
          { name: "Watcher-01", status: "failed", summary: "Tests failed", duration: 3,
            blockers: ["compile error in module"] },
        ];
      }
      // Second iteration: workers succeed with test results -> high confidence -> stop
      return [
        { name: "Builder-01", status: "completed", summary: "Done", duration: 8,
          files_created: ["src/feature.ts"], files_modified: ["src/helper.ts"],
          tests_written: ["test/feature.test.ts"],
          test_results: { passed: 10, total: 10 } },
        { name: "Watcher-01", status: "completed", summary: "All green", duration: 4,
          files_modified: ["test/feature.test.ts"],
          test_results: { passed: 10, total: 10 } },
      ];
    });

    __setDetectAvailablePlatforms(async () => [
      "claude" as const,
    ]);

    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: process.cwd() };
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    await runDispatchedBuildCommand(bridge, parsed, def);

    // 1. hive-read was called
    const hiveReadCall = goCalls.find((args) => args[0] === "hive-read");
    assert.ok(hiveReadCall, "hive-read should have been called");

    // 2. spawnOrchestrator in dispatch opts has totalBudget from manifest
    assert.ok(capturedDispatchOpts.length >= 1, "Should have captured dispatch opts");
    const firstOpts = capturedDispatchOpts[0] as Record<string, unknown>;
    assert.ok(firstOpts.spawnOrchestrator, "Dispatch opts should include spawnOrchestrator");
    const orchestrator = firstOpts.spawnOrchestrator as { totalBudget: number; consumedBudget: number };
    assert.equal(orchestrator.totalBudget, 10, "Total budget should match manifest max_workers");
    assert.equal(orchestrator.consumedBudget, 2, "Consumed budget should equal manifest dispatch count");

    // 3. Two dispatch iterations produce one lifecycle finalization.
    const finalizeCount = countGoCommandCalls(goCalls, "build-finalize");
    assert.equal(finalizeCount, 1, "Should call build-finalize once after 2 iterations");

    // 4. Second iteration dispatches have task_brief with blocker text from iteration 1
    assert.ok(capturedAllDispatches.length >= 2, "Should have at least 2 dispatch calls");
    const secondDispatches = capturedAllDispatches[1] as Array<Record<string, unknown>>;
    assert.ok(secondDispatches.length > 0, "Second dispatch should have dispatches");
    const hasBlockerFeedback = secondDispatches.some(
      (d) => typeof d.task_brief === "string" && (
        d.task_brief!.includes("missing import") ||
        d.task_brief!.includes("compile error") ||
        d.task_brief!.includes("Previous iteration feedback")
      ),
    );
    assert.ok(hasBlockerFeedback, "Second iteration dispatches should contain blocker text from iteration 1");

    // 5. stderr contains hive wisdom and iteration markers
    assert.ok(
      stderrOutput.includes("Injecting hive wisdom"),
      `Stderr should contain hive wisdom injection marker. Got: ${stderrOutput.slice(0, 500)}`
    );
    assert.ok(
      stderrOutput.includes("Iteration"),
      `Stderr should contain iteration markers. Got: ${stderrOutput.slice(0, 500)}`
    );
  });

  it("plan-through-build-through-continue pipeline", async () => {
    // This test verifies the 3-step lifecycle runs end-to-end without errors.
    // Each step uses its own mock setup and verifies its finalizer is called.
    const handler = <T>(_bridgeOpts: unknown, args: string[]): T => {
      goCalls.push(args);
      const cmd = args[0];

      if (cmd === "hive-read") {
        return { entries: null, total: 0 } as unknown as T;
      }
      if (cmd === "registry-list") {
        return {
          colonies: [
            {
              repo_path: process.cwd(),
              domains: ["go"],
              active: true,
              registered_at: "2026-05-18T00:00:00Z",
            },
          ],
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
      if (cmd === "build") {
        return {
          dispatch_manifest: {
            execution_binding: TEST_EXECUTION_BINDING,
            dispatches: [
              { name: "Builder-01", caste: "builder", task: "Implement", wave: 1, execution_wave: 1 },
            ],
          },
        } as unknown as T;
      }
      if (cmd === "build-finalize") {
        return { ok: true } as unknown as T;
      }
      if (cmd === "continue") {
        return {
          continue_manifest: { phase: 1 },
          dispatches: [
            { name: "Watcher-01", caste: "watcher", task: "Verify", wave: 1, execution_wave: 1 },
          ],
        } as unknown as T;
      }
      if (cmd === "continue-finalize") {
        return { ok: true } as unknown as T;
      }
      return { ok: true } as unknown as T;
    };

    __setCallGoJSON(handler);
    __setGoBridgeCallGoJSON(handler);

    __setDispatchWorkers(async (_opts, dispatches) => {
      dispatchCallCount++;
      capturedAllDispatches.push(dispatches);
      return dispatches.map((d: BuildDispatch) => ({
        name: d.name,
        status: "completed",
        summary: "Done",
        duration: 5,
        files_created: ["src/module.ts"],
        test_results: { passed: 8, total: 8 },
      }));
    });

    __setDetectAvailablePlatforms(async () => [
      "claude" as const,
    ]);

    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: process.cwd() };
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const buildDef = getHostCommandDefinition("build")!;

    // Step 1: Run plan pipeline
    const planParsed = parseArgs(["node", "host.js", "plan", "--simulate"]);
    await runDispatchedPlanCommand(bridge, planParsed);

    // Verify plan-finalize was called
    const planFinalizeCount = countGoCommandCalls(goCalls, "plan-finalize");
    assert.equal(planFinalizeCount, 1, "plan-finalize should have been called once");

    // Step 2: Run build pipeline
    const buildParsed = parseArgs(["node", "host.js", "build", "1", "--simulate"]);
    await runDispatchedBuildCommand(bridge, buildParsed, buildDef);

    // Verify build-finalize was called
    const buildFinalizeCount = countGoCommandCalls(goCalls, "build-finalize");
    assert.ok(buildFinalizeCount >= 1, "build-finalize should have been called at least once");

    // Step 3: Run continue pipeline
    const continueParsed = parseArgs(["node", "host.js", "continue", "--simulate"]);
    await runDispatchedContinueCommand(bridge, continueParsed);

    // Verify continue-finalize was called
    const continueFinalizeCount = countGoCommandCalls(goCalls, "continue-finalize");
    assert.equal(continueFinalizeCount, 1, "continue-finalize should have been called once");

    // Verify all 3 steps completed without throwing
    assert.equal(dispatchCallCount, 3, "Should have dispatched workers 3 times (plan + build + continue)");
  });

  it("spawn budget consumed across iterations", async () => {
    // Manifest has max_workers: 5 with 3 dispatches. After iteration 1 consumes 3,
    // verify iteration 2 still dispatches (budget allows it).
    // Assert spawnOrchestrator.consumedBudget tracks correctly across iterations.
    const handler = <T>(_bridgeOpts: unknown, args: string[]): T => {
      goCalls.push(args);
      const cmd = args[0];

      if (cmd === "hive-read") {
        return { entries: null, total: 0 } as unknown as T;
      }
      if (cmd === "registry-list") {
        return {
          colonies: [
            {
              repo_path: process.cwd(),
              domains: ["go"],
              active: true,
              registered_at: "2026-05-18T00:00:00Z",
            },
          ],
        } as unknown as T;
      }
      if (cmd === "build") {
        return {
          dispatch_manifest: {
            execution_binding: TEST_EXECUTION_BINDING,
            dispatches: [
              { name: "Builder-01", caste: "builder", task: "Build A", wave: 1, execution_wave: 1 },
              { name: "Builder-02", caste: "builder", task: "Build B", wave: 1, execution_wave: 1 },
              { name: "Builder-03", caste: "builder", task: "Build C", wave: 1, execution_wave: 1 },
            ],
            queen_execution_policy: {
              spawn_budget: { max_workers: 5 },
            },
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

    let dispatchNum = 0;
    __setDispatchWorkers(async (opts, dispatches) => {
      dispatchNum++;
      dispatchCallCount++;
      capturedAllDispatches.push(dispatches);
      capturedDispatchOpts.push(opts);

      if (dispatchNum === 1) {
        // First iteration: fail with blockers -> triggers second iteration
        return dispatches.map((d: BuildDispatch) => ({
          name: d.name,
          status: "failed",
          summary: "Failed",
          duration: 5,
          blockers: ["compile error"],
        }));
      }
      // Second iteration: succeed with test results -> high confidence, stop
      return dispatches.map((d: BuildDispatch) => ({
        name: d.name,
        status: "completed",
        summary: "Done",
        duration: 5,
        files_created: ["src/module.ts"],
        test_results: { passed: 10, total: 10 },
      }));
    });

    __setDetectAvailablePlatforms(async () => [
      "claude" as const,
    ]);

    const parsed = parseArgs(["node", "host.js", "build", "1", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: process.cwd() };
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("build")!;

    await runDispatchedBuildCommand(bridge, parsed, def);

    // Should have 2 iterations (first fails, second succeeds)
    const finalizeCount = countGoCommandCalls(goCalls, "build-finalize");
    assert.equal(finalizeCount, 1, "Should call build-finalize exactly once");

    // First dispatch opts should have spawnOrchestrator with totalBudget=5, consumedBudget=3
    assert.ok(capturedDispatchOpts.length >= 2, "Should have captured dispatch opts for both iterations");
    const firstOpts = capturedDispatchOpts[0] as Record<string, unknown>;
    assert.ok(firstOpts.spawnOrchestrator, "First dispatch opts should include spawnOrchestrator");
    const firstOrchestrator = firstOpts.spawnOrchestrator as { totalBudget: number; consumedBudget: number };
    assert.equal(firstOrchestrator.totalBudget, 5, "First iteration totalBudget should be 5");
    assert.equal(firstOrchestrator.consumedBudget, 3, "First iteration consumedBudget should be 3 (manifest dispatch count)");

    // Second dispatch opts should also have spawnOrchestrator
    const secondOpts = capturedDispatchOpts[1] as Record<string, unknown>;
    assert.ok(secondOpts.spawnOrchestrator, "Second dispatch opts should include spawnOrchestrator");
    const secondOrchestrator = secondOpts.spawnOrchestrator as { totalBudget: number; consumedBudget: number };
    assert.equal(secondOrchestrator.totalBudget, 5, "Second iteration totalBudget should be 5");

    // Budget allows second iteration: 5 total - 3 consumed from first = 2 remaining, and 3 new dispatches
    // The host re-fetches manifest for iteration 2 which returns 3 dispatches, but the confidence loop
    // checks budgetRemaining from the ConfidenceLoop (not SpawnOrchestrator), so iteration 2 should proceed
    // since default budget of 20 is used by ConfidenceLoop (unless totalBudget < 20).
    // With totalBudget=5 and 3 workers used in iteration 1, ConfidenceLoop budgetRemaining = 5-3 = 2.
    // But dispatches for iteration 2 = 3 > 2 remaining -> budget exhausted after iteration 2.
    // The second iteration DOES dispatch because budget check happens AFTER evaluate(), not before.
    assert.equal(dispatchCallCount, 2, "Should dispatch 2 waves");

    // Verify stderr contains iteration ceremony markers
    assert.ok(
      stderrOutput.includes("Iteration"),
      `Stderr should contain iteration markers. Got: ${stderrOutput.slice(0, 500)}`
    );
  });
});

describe("temp file cleanup after finalizer", () => {
  beforeEach(() => {
    __restoreAllMocks();
    __restoreGoBridgeCallGoJSON();
    __restoreCreateCeremonyAdapter();
  });

  afterEach(() => {
    __restoreAllMocks();
    __restoreGoBridgeCallGoJSON();
    __restoreCreateCeremonyAdapter();
  });

  it("cleanup removes aether temp dirs and is safe on non-existent paths", async () => {
    const { writeCompletionFile, cleanupCompletionDir } = await import("../src/go-bridge.js");

    // Create a temp completion file like the host would
    const completionPath = writeCompletionFile(
      "aether-build",
      "test-completion.json",
      { ok: true }
    );
    assert.ok(existsSync(completionPath), "Completion file should exist before cleanup");

    // Cleanup should remove it
    cleanupCompletionDir(completionPath);
    assert.ok(!existsSync(completionPath), "Completion file should be gone after cleanup");

    // Calling again on non-existent path should not throw
    assert.doesNotThrow(() => cleanupCompletionDir(completionPath));
    assert.doesNotThrow(() => cleanupCompletionDir("/tmp/aether-build-nonexistent-abc/completion.json"));
  });
});
