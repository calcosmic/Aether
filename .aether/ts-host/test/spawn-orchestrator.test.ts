/**
 * Spawn orchestrator tests.
 *
 * 203-09: the orchestrator no longer computes admission itself. Every test
 * here stubs the Go bridge (`__setCallGoJSON`) and asserts processClaims
 * bridges to it correctly -- one call per claim, the gate's own detail
 * returned verbatim, and every bridge-failure shape denying fail-closed.
 * Local budget/depth arithmetic tests are gone with the arithmetic itself.
 */

import { describe, it, afterEach } from "node:test";
import assert from "node:assert/strict";

import {
  createSpawnOrchestrator,
  synthesizeChildDispatch,
  type SpawnOrchestrator,
} from "../src/spawn-orchestrator.js";
import type { SpawnClaim } from "../src/types.js";
import { __setCallGoJSON, __restoreCallGoJSON, type GoBridgeOptions } from "../src/go-bridge.js";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Capture stderr output during a callback. */
function captureStderr(fn: () => void): string {
  const chunks: string[] = [];
  const originalWrite = process.stderr.write.bind(process.stderr);
  process.stderr.write = (chunk: unknown, ...args: unknown[]) => {
    if (typeof chunk === "string") {
      chunks.push(chunk);
    }
    if (args.length === 0) {
      return originalWrite(chunk as string);
    }
    if (typeof args[0] === "function") {
      return originalWrite(chunk as string, args[0] as (...rest: unknown[]) => void);
    }
    return originalWrite(
      chunk as string,
      args[0] as BufferEncoding,
      ...(args.slice(1) as [((...rest: unknown[]) => void)?])
    );
  };
  try {
    fn();
  } finally {
    process.stderr.write = originalWrite;
  }
  return chunks.join("");
}

function makeOrchestrator(): SpawnOrchestrator {
  return createSpawnOrchestrator({
    goBinaryPath: "/usr/local/bin/aether",
    cwd: "/tmp/test",
  });
}

afterEach(() => {
  __restoreCallGoJSON();
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("bridges every claim to the Go admission gate (SYN-203-02)", () => {
  it("calls the Go binary once per claim with parent/depth/caste/task/workspace", () => {
    const calls: { args: string[]; opts: GoBridgeOptions }[] = [];
    __setCallGoJSON((opts: GoBridgeOptions, args: string[]) => {
      calls.push({ args, opts });
      return { can_spawn: true } as never;
    });

    const orch = makeOrchestrator();
    const claims: SpawnClaim[] = [
      { caste: "scout", task: "Research X" },
      { caste: "builder", task: "Build Y" },
    ];
    const result = orch.processClaims("Builder-01", 1, claims);

    assert.equal(calls.length, 2, "one Go call per claim");
    assert.equal(result.accepted.length, 2);
    assert.equal(result.rejected.length, 0);

    const [first] = calls;
    assert.equal(first!.args[0], "spawn-can-spawn");
    assert.ok(first!.args.includes("--name"));
    assert.ok(first!.args.includes("Builder-01"));
    assert.ok(first!.args.includes("--depth"));
    assert.ok(first!.args.includes("1"));
    assert.ok(first!.args.includes("--caste"));
    assert.ok(first!.args.includes("scout"));
    assert.ok(first!.args.includes("--task"));
    assert.ok(first!.args.includes("Research X"));
    assert.ok(first!.args.includes("--workspace"));
  });

  it("returns the gate's detail verbatim on a deny", () => {
    __setCallGoJSON(() => ({
      can_spawn: false,
      reason: "depth",
      detail: "Builder-01 is at depth 2; a helper spawned from here would be depth 3, past the cap of 2",
    } as never));

    const orch = makeOrchestrator();
    const result = orch.processClaims("Builder-01", 2, [
      { caste: "watcher", task: "Test edge cases" },
    ]);

    assert.equal(result.accepted.length, 0);
    assert.equal(result.rejected.length, 1);
    assert.equal(
      result.rejected[0]!.reason,
      "Builder-01 is at depth 2; a helper spawned from here would be depth 3, past the cap of 2",
      "rejection reason must equal the gate's own detail sentence exactly"
    );
  });

  it("falls back to the gate's reason when no detail is present", () => {
    __setCallGoJSON(() => ({ can_spawn: false, reason: "budget" } as never));

    const orch = makeOrchestrator();
    const result = orch.processClaims("Builder-01", 1, [
      { caste: "scout", task: "Research" },
    ]);

    assert.equal(result.rejected.length, 1);
    assert.equal(result.rejected[0]!.reason, "budget");
  });

  it("no exported or module-level value holds a budget total, consumed count or depth limit", async () => {
    const mod: Record<string, unknown> = await import("../src/spawn-orchestrator.js");
    assert.equal(mod.DEFAULT_TOTAL_BUDGET, undefined);
    assert.equal(mod.MAX_SPAWN_DEPTH, undefined);
    const orch = makeOrchestrator();
    assert.equal((orch as unknown as Record<string, unknown>).totalBudget, undefined);
    assert.equal((orch as unknown as Record<string, unknown>).consumedBudget, undefined);
    assert.equal((orch as unknown as Record<string, unknown>).remainingBudget, undefined);
  });
});

describe("bridge failure shapes all deny (fail-closed, never local recomputation)", () => {
  it("denies on a non-zero exit", () => {
    __setCallGoJSON(() => {
      throw new Error("Go command failed: spawn-can-spawn: exit status 1");
    });
    const orch = makeOrchestrator();
    const result = orch.processClaims("Builder-01", 1, [
      { caste: "scout", task: "Research" },
    ]);
    assert.equal(result.accepted.length, 0);
    assert.equal(result.rejected.length, 1);
    assert.match(result.rejected[0]!.reason, /spawn admission gate unreachable/);
  });

  it("denies on a timeout", () => {
    __setCallGoJSON(() => {
      throw new Error("Go subprocess failed for spawn-can-spawn: signal SIGTERM; subprocess output omitted");
    });
    const orch = makeOrchestrator();
    const result = orch.processClaims("Builder-01", 1, [
      { caste: "scout", task: "Research" },
    ]);
    assert.equal(result.accepted.length, 0);
    assert.equal(result.rejected.length, 1);
    assert.match(result.rejected[0]!.reason, /spawn admission gate unreachable/);
  });

  it("denies on a malformed envelope", () => {
    __setCallGoJSON(() => {
      throw new Error("Go subprocess returned invalid JSON for spawn-can-spawn; subprocess output omitted");
    });
    const orch = makeOrchestrator();
    const result = orch.processClaims("Builder-01", 1, [
      { caste: "scout", task: "Research" },
    ]);
    assert.equal(result.accepted.length, 0);
    assert.equal(result.rejected.length, 1);
    assert.match(result.rejected[0]!.reason, /spawn admission gate unreachable/);
  });

  it("denies on a missing binary", () => {
    __setCallGoJSON(() => {
      throw new Error("Go subprocess failed for spawn-can-spawn: binary not found; subprocess output omitted");
    });
    const orch = makeOrchestrator();
    const result = orch.processClaims("Builder-01", 1, [
      { caste: "scout", task: "Research" },
    ]);
    assert.equal(result.accepted.length, 0);
    assert.equal(result.rejected.length, 1);
    assert.match(result.rejected[0]!.reason, /spawn admission gate unreachable/);
  });
});

describe("child dispatch synthesis (unchanged)", () => {
  it("synthesizes BuildDispatch with correct name, caste, task", () => {
    const claim: SpawnClaim = { caste: "scout", task: "Research" };
    const dispatch = synthesizeChildDispatch(claim, "Builder-01", 2, 0);
    assert.equal(dispatch.name, "Builder-01-spawn-0");
    assert.equal(dispatch.caste, "scout");
    assert.equal(dispatch.task, "Research");
    assert.equal(dispatch.status, "pending");
  });

  it("increments index for multiple spawns from same parent", () => {
    const claims: SpawnClaim[] = [
      { caste: "scout", task: "Research A" },
      { caste: "builder", task: "Build B" },
      { caste: "watcher", task: "Test C" },
    ];
    const dispatches = claims.map((claim, i) =>
      synthesizeChildDispatch(claim, "Builder-01", 2, i)
    );
    assert.equal(dispatches[0]!.name, "Builder-01-spawn-0");
    assert.equal(dispatches[1]!.name, "Builder-01-spawn-1");
    assert.equal(dispatches[2]!.name, "Builder-01-spawn-2");
  });
});

describe("rejection logging", () => {
  it("logs rejected spawns to stderr with the gate's own detail", () => {
    __setCallGoJSON(() => ({
      can_spawn: false,
      reason: "budget",
      detail: "spawn budget exhausted",
    } as never));
    const orch = makeOrchestrator();
    const stderr = captureStderr(() => {
      orch.processClaims("Builder-01", 1, [
        { caste: "scout", task: "Research X" },
      ]);
    });
    assert.ok(
      stderr.includes("Spawn rejected"),
      `stderr should contain "Spawn rejected", got: ${stderr}`
    );
    assert.ok(
      stderr.includes("spawn budget exhausted"),
      `stderr should contain rejection reason, got: ${stderr}`
    );
  });

  it("logs depth rejection to stderr", () => {
    __setCallGoJSON(() => ({
      can_spawn: false,
      reason: "depth",
      detail: "max spawn depth exceeded (limit: 2)",
    } as never));
    const orch = makeOrchestrator();
    const stderr = captureStderr(() => {
      orch.processClaims("Builder-01-spawn-0", 2, [
        { caste: "scout", task: "Research X" },
      ]);
    });
    assert.ok(
      stderr.includes("Spawn rejected"),
      `stderr should contain "Spawn rejected", got: ${stderr}`
    );
    assert.ok(
      stderr.includes("max spawn depth exceeded"),
      `stderr should contain depth reason, got: ${stderr}`
    );
  });
});

describe("edge cases", () => {
  it("empty claims array returns empty accepted and rejected, with zero Go calls", () => {
    let callCount = 0;
    __setCallGoJSON(() => {
      callCount++;
      return { can_spawn: true } as never;
    });
    const orch = makeOrchestrator();
    const result = orch.processClaims("Builder-01", 1, []);
    assert.equal(result.accepted.length, 0);
    assert.equal(result.rejected.length, 0);
    assert.equal(callCount, 0, "no admission call for an empty claim list");
  });

  it("handles undefined claims gracefully", () => {
    __setCallGoJSON(() => ({ can_spawn: true } as never));
    const orch = makeOrchestrator();
    // Explicitly pass undefined to test the guard
    const result = orch.processClaims(
      "Builder-01",
      1,
      undefined as unknown as SpawnClaim[]
    );
    assert.equal(result.accepted.length, 0);
    assert.equal(result.rejected.length, 0);
  });
});
