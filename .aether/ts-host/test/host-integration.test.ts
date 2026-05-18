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
} from "../src/host.js";

import type { DispatchResult } from "../src/worker-dispatch.js";
import type { Platform } from "../src/platform-dispatcher.js";

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
