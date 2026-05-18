/**
 * Spawn orchestrator tests.
 *
 * Verifies budget enforcement (SPAWN-03, SPAWN-06), depth limits (SPAWN-02),
 * child dispatch synthesis, rejection logging, and edge cases.
 */

import { describe, it } from "node:test";
import assert from "node:assert/strict";

import {
  createSpawnOrchestrator,
  synthesizeChildDispatch,
  type SpawnOrchestrator,
  type SpawnProcessingResult,
} from "../src/spawn-orchestrator.js";
import type { SpawnClaim } from "../src/types.js";

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

/** Convenience: create an orchestrator with specific budget/consumed values. */
function makeOrchestrator(
  totalBudget: number,
  consumedBudget = 0
): SpawnOrchestrator {
  return createSpawnOrchestrator({
    goBinaryPath: "/usr/local/bin/aether",
    cwd: "/tmp/test",
    totalBudget,
    consumedBudget,
    currentDepth: 1,
  });
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("budget enforcement (SPAWN-03, SPAWN-06)", () => {
  it("accepts spawns within remaining budget", () => {
    const orch = makeOrchestrator(5, 3);
    const claims: SpawnClaim[] = [
      { caste: "scout", task: "Research X" },
      { caste: "builder", task: "Build Y" },
    ];
    const result = orch.processClaims("Builder-01", 1, claims);
    assert.equal(result.accepted.length, 2, "both claims should be accepted");
    assert.equal(result.rejected.length, 0, "no rejections expected");
    assert.equal(orch.remainingBudget, 0, "budget should be fully consumed");
  });

  it("rejects spawns exceeding budget", () => {
    const orch = makeOrchestrator(5, 4);
    const claims: SpawnClaim[] = [
      { caste: "scout", task: "Research A" },
      { caste: "builder", task: "Build B" },
      { caste: "watcher", task: "Test C" },
    ];
    const result = orch.processClaims("Builder-01", 1, claims);
    assert.equal(result.accepted.length, 1, "only 1 claim should fit in budget");
    assert.equal(result.rejected.length, 2, "2 claims should be rejected");
    assert.equal(
      result.rejected[0]!.reason,
      "spawn budget exhausted",
      "first rejection reason"
    );
    assert.equal(
      result.rejected[1]!.reason,
      "spawn budget exhausted",
      "second rejection reason"
    );
  });

  it("tracks consumed budget across multiple processClaims calls", () => {
    const orch = makeOrchestrator(5, 2);

    // First call: 2 claims accepted (budget remaining: 5 - 2 - 2 = 1)
    const result1 = orch.processClaims("Builder-01", 1, [
      { caste: "scout", task: "Research A" },
      { caste: "scout", task: "Research B" },
    ]);
    assert.equal(result1.accepted.length, 2);
    assert.equal(orch.remainingBudget, 1);

    // Second call: 3 claims, but only 1 slot left
    const result2 = orch.processClaims("Builder-02", 1, [
      { caste: "builder", task: "Build C" },
      { caste: "builder", task: "Build D" },
      { caste: "builder", task: "Build E" },
    ]);
    assert.equal(result2.accepted.length, 1, "only 1 more fits");
    assert.equal(result2.rejected.length, 2, "2 rejected on second call");
  });

  it("zero budget rejects all spawns", () => {
    const orch = makeOrchestrator(0, 0);
    const claims: SpawnClaim[] = [
      { caste: "scout", task: "Research X" },
    ];
    const result = orch.processClaims("Builder-01", 1, claims);
    assert.equal(result.accepted.length, 0, "nothing accepted with zero budget");
    assert.equal(result.rejected.length, 1);
    assert.equal(result.rejected[0]!.reason, "spawn budget exhausted");
  });
});

describe("depth limits (SPAWN-02)", () => {
  it("allows depth-1 workers to spawn depth-2 children", () => {
    const orch = makeOrchestrator(20, 0);
    const result = orch.processClaims("Builder-01", 1, [
      { caste: "scout", task: "Research dependencies" },
    ]);
    assert.equal(result.accepted.length, 1);
    assert.equal(result.accepted[0]!.depth, 2, "child should be at depth 2");
    assert.equal(result.accepted[0]!.parent, "Builder-01");
  });

  it("rejects depth-2 workers from spawning depth-3 grandchildren", () => {
    const orch = makeOrchestrator(20, 0);
    const result = orch.processClaims("Builder-01-spawn-0", 2, [
      { caste: "watcher", task: "Test edge cases" },
    ]);
    assert.equal(result.accepted.length, 0, "depth-2 parent cannot spawn");
    assert.equal(result.rejected.length, 1);
    assert.equal(
      result.rejected[0]!.reason,
      "max spawn depth exceeded (limit: 2)"
    );
  });

  it("manifest workers (depth=1) can spawn, their children (depth=2) cannot", () => {
    const orch = makeOrchestrator(20, 0);

    // First call: manifest worker at depth 1
    const result1 = orch.processClaims("Builder-01", 1, [
      { caste: "scout", task: "Research" },
    ]);
    assert.equal(result1.accepted.length, 1);
    assert.equal(result1.accepted[0]!.depth, 2);

    // Second call: spawned child at depth 2 tries to spawn
    const result2 = orch.processClaims("Builder-01-spawn-0", 2, [
      { caste: "builder", task: "Build more" },
    ]);
    assert.equal(result2.accepted.length, 0, "depth-2 worker cannot spawn");
    assert.equal(result2.rejected.length, 1);
  });
});

describe("child dispatch synthesis", () => {
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
  it("logs rejected spawns to stderr", () => {
    const orch = makeOrchestrator(1, 1);
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
    const orch = makeOrchestrator(20, 0);
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
  it("empty claims array returns empty accepted and rejected", () => {
    const orch = makeOrchestrator(20, 0);
    const result = orch.processClaims("Builder-01", 1, []);
    assert.equal(result.accepted.length, 0);
    assert.equal(result.rejected.length, 0);
  });

  it("handles undefined claims gracefully", () => {
    const orch = makeOrchestrator(20, 0);
    // Explicitly pass undefined to test the guard
    const result = orch.processClaims(
      "Builder-01",
      1,
      undefined as unknown as SpawnClaim[]
    );
    assert.equal(result.accepted.length, 0);
    assert.equal(result.rejected.length, 0);
  });

  it("remainingBudget reflects defaults when no options provided", () => {
    const orch = createSpawnOrchestrator();
    assert.equal(orch.remainingBudget, 20, "default budget is 20");
    assert.equal(orch.totalBudget, 20);
    assert.equal(orch.consumedBudget, 0);
  });
});
