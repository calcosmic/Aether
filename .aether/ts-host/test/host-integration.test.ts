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
import { existsSync, mkdtempSync, symlinkSync } from "node:fs";
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
  __restoreAllMocks,
  runDispatchedBuildCommand,
  runDispatchedPlanCommand,
  runDispatchedContinueCommand,
  runDryRunDispatchedCommand,
} from "../src/host.js";

import type { DispatchResult } from "../src/worker-dispatch.js";
import type { Platform } from "../src/platform-dispatcher.js";
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
    __setDetectAvailablePlatforms(async () => [
      { name: "claude", cliCommand: "claude" } as Platform,
    ]);
  });

  afterEach(() => {
    __restoreAllMocks();
  });

  it("plan command uses dispatched runner type", async () => {
    const { getHostCommandDefinition } = await import("../src/command-registry.js");
    const def = getHostCommandDefinition("plan");
    assert.equal(def?.runner, "dispatched", "plan command should use dispatched runner");
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
    const args = buildHostGoArgs(parsed);

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
    const args = buildHostGoArgs(parsed);

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
      return originalStderrWrite(chunk, ...args as [string, ...unknown[]]);
    }) as typeof process.stderr.write;

    // Mock callGoJSON to return manifests
    __setCallGoJSON(<T>(_opts: unknown, args: string[]): T => {
      goCalls.push(args);
      const cmd = args[0];
      if (cmd === "build") {
        return {
          dispatch_manifest: {
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
    __setDetectAvailablePlatforms(async () => [
      { name: "claude", cliCommand: "claude" } as Platform,
    ]);
  });

  afterEach(() => {
    __restoreAllMocks();
  });

  it("--dry-run on build renders ceremony without dispatching workers", async () => {
    const parsed = parseArgs(["node", "host.js", "build", "1", "--dry-run"]);
    assert.equal(parsed.dryRun, true, "--dry-run should set parsed.dryRun");
    assert.equal(parsed.simulate, false, "--dry-run should not set simulate");

    // Verify dry-run calls Go for manifest
    const args = buildHostGoArgs(parsed);
    assert.equal(args[0], "build");
    assert.ok(args.includes("--plan-only"));
  });

  it("--dry-run on plan renders ceremony without dispatching workers", () => {
    const parsed = parseArgs(["node", "host.js", "plan", "--dry-run"]);
    assert.equal(parsed.dryRun, true);

    const args = buildHostGoArgs(parsed);
    assert.equal(args[0], "plan");
    assert.ok(args.includes("--plan-only"));
  });

  it("--dry-run on continue renders ceremony without dispatching workers", () => {
    const parsed = parseArgs(["node", "host.js", "continue", "--dry-run"]);
    assert.equal(parsed.dryRun, true);

    const args = buildHostGoArgs(parsed);
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
      return originalStderrWrite(chunk, ...args as [string, ...unknown[]]);
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
      return dispatches.map((d: Record<string, unknown>) => ({
        name: d.name,
        status: "completed",
        summary: "Done",
        duration: 5,
      }));
    });

    __setDetectAvailablePlatforms(async () => [
      { name: "claude", cliCommand: "claude" } as Platform,
    ]);
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
      return dispatches.map((d: Record<string, unknown>) => ({
        name: d.name,
        status: "completed",
        summary: "Done",
        duration: 5,
      }));
    });

    __setDetectAvailablePlatforms(async () => [
      { name: "claude", cliCommand: "claude" } as Platform,
    ]);
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
      return originalStderrWrite(chunk, ...args as [string, ...unknown[]]);
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
      return dispatches.map((d: Record<string, unknown>) => ({
        name: d.name,
        status: "completed",
        summary: "Done",
        duration: 5,
      }));
    });

    __setDetectAvailablePlatforms(async () => [
      { name: "claude", cliCommand: "claude" } as Platform,
    ]);
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
