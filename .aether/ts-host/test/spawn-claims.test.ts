/**
 * Spawn claims parsing tests.
 *
 * Verifies structured spawn claims parsing, backward compatibility with
 * string spawns, validation of malformed entries, and normalization.
 */

import { describe, it, beforeEach } from "node:test";
import assert from "node:assert/strict";

import {
  parseWorkerClaims,
  validateWorkerClaims,
  normalizeSpawnClaims,
} from "../src/claims-parser.js";
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
    // Call original with correct types
    if (args.length === 0) {
      return originalWrite(chunk as string);
    }
    if (typeof args[0] === "function") {
      return originalWrite(chunk as string, args[0] as (...rest: unknown[]) => void);
    }
    return originalWrite(chunk as string, args[0] as BufferEncoding, ...(args.slice(1) as [((...rest: unknown[]) => void)?]));
  };
  try {
    fn();
  } finally {
    process.stderr.write = originalWrite;
  }
  return chunks.join("");
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("structured spawn claims parsing", () => {
  it("parses SpawnClaim array from direct JSON", () => {
    const stdout = JSON.stringify({
      status: "completed",
      spawns: [
        { caste: "scout", task: "Research API", reason: "Need context" },
        { caste: "builder", task: "Implement feature" },
      ],
    });
    const claims = parseWorkerClaims(stdout);
    assert.ok(claims.spawns, "spawns should be present");
    assert.equal(claims.spawns!.length, 2);

    const spawn0 = claims.spawns![0] as SpawnClaim;
    assert.equal(spawn0.caste, "scout");
    assert.equal(spawn0.task, "Research API");
    assert.equal(spawn0.reason, "Need context");

    const spawn1 = claims.spawns![1] as SpawnClaim;
    assert.equal(spawn1.caste, "builder");
    assert.equal(spawn1.task, "Implement feature");
    assert.equal(spawn1.reason, undefined);
  });

  it("parses spawns from code-fenced output", () => {
    const json = JSON.stringify({
      status: "completed",
      spawns: [{ caste: "watcher", task: "Run integration tests" }],
    });
    const stdout = "```json\n" + json + "\n```";
    const claims = parseWorkerClaims(stdout);
    assert.ok(claims.spawns, "spawns should be present");
    assert.equal(claims.spawns!.length, 1);
    const spawn = claims.spawns![0] as SpawnClaim;
    assert.equal(spawn.caste, "watcher");
    assert.equal(spawn.task, "Run integration tests");
  });

  it("parses spawns from trailing JSON block", () => {
    const json = JSON.stringify({
      status: "completed",
      spawns: [{ caste: "scout", task: "Investigate dependencies" }],
    });
    const stdout = "Some worker output text\nMore output\n" + json;
    const claims = parseWorkerClaims(stdout);
    assert.ok(claims.spawns, "spawns should be present");
    assert.equal(claims.spawns!.length, 1);
    const spawn = claims.spawns![0] as SpawnClaim;
    assert.equal(spawn.caste, "scout");
    assert.equal(spawn.task, "Investigate dependencies");
  });
});

describe("backward compatibility", () => {
  it("accepts string[] spawns and normalizes them", () => {
    const stdout = JSON.stringify({
      status: "completed",
      spawns: ["Need a Scout"],
    });
    const claims = parseWorkerClaims(stdout);
    assert.ok(claims.spawns, "spawns should be present");
    assert.equal(claims.spawns!.length, 1);

    const spawn = claims.spawns![0] as SpawnClaim;
    assert.equal(spawn.caste, "builder", "string spawns default to builder caste");
    assert.equal(spawn.task, "Need a Scout");
    assert.equal(spawn.reason, "auto-converted from string spawn");
  });

  it("accepts mixed string and object spawns", () => {
    const stdout = JSON.stringify({
      status: "completed",
      spawns: [
        "Need a Scout",
        { caste: "watcher", task: "Run tests" },
      ],
    });
    const claims = parseWorkerClaims(stdout);
    assert.ok(claims.spawns, "spawns should be present");
    assert.equal(claims.spawns!.length, 2);

    // First spawn: string normalized
    const spawn0 = claims.spawns![0] as SpawnClaim;
    assert.equal(spawn0.caste, "builder");
    assert.equal(spawn0.task, "Need a Scout");
    assert.equal(spawn0.reason, "auto-converted from string spawn");

    // Second spawn: object passed through
    const spawn1 = claims.spawns![1] as SpawnClaim;
    assert.equal(spawn1.caste, "watcher");
    assert.equal(spawn1.task, "Run tests");
    assert.equal(spawn1.reason, undefined);
  });
});

describe("validation", () => {
  it("filters invalid spawn claims and logs warning", () => {
    const stderr = captureStderr(() => {
      const stdout = JSON.stringify({
        status: "completed",
        spawns: [
          { caste: "scout", task: "Valid task" },
          { caste: "builder" },  // missing task field
          { task: "No caste" },  // missing caste field
          42,                     // not string or object
        ],
      });
      const claims = parseWorkerClaims(stdout);
      assert.ok(claims.spawns, "valid spawns should remain");
      assert.equal(claims.spawns!.length, 1, "only valid spawn should survive");
      const spawn = claims.spawns![0] as SpawnClaim;
      assert.equal(spawn.caste, "scout");
      assert.equal(spawn.task, "Valid task");
    });
    assert.ok(
      stderr.includes("Warning: invalid spawn claim at index 1"),
      "should warn about missing task"
    );
    assert.ok(
      stderr.includes("Warning: invalid spawn claim at index 2"),
      "should warn about missing caste"
    );
    assert.ok(
      stderr.includes("Warning: invalid spawn claim at index 3"),
      "should warn about non-object/string entry"
    );
  });

  it("handles non-array spawns gracefully", () => {
    // Non-array spawns field should not crash the parser
    const stdout = JSON.stringify({
      status: "completed",
      spawns: "not-an-array",
    });
    const claims = parseWorkerClaims(stdout);
    assert.equal(claims.spawns, undefined, "non-array spawns should be ignored");
  });
});

describe("normalizeSpawnClaims", () => {
  it("converts strings to SpawnClaim with builder default", () => {
    const result = normalizeSpawnClaims(["Research X", "Build Y"]);
    assert.equal(result.length, 2);
    assert.equal(result[0]!.caste, "builder");
    assert.equal(result[0]!.task, "Research X");
    assert.equal(result[0]!.reason, "auto-converted from string spawn");
    assert.equal(result[1]!.caste, "builder");
    assert.equal(result[1]!.task, "Build Y");
  });

  it("passes through existing SpawnClaim objects unchanged", () => {
    const claim: SpawnClaim = { caste: "scout", task: "Investigate", reason: "needed" };
    const result = normalizeSpawnClaims([claim]);
    assert.equal(result.length, 1);
    assert.equal(result[0]!.caste, "scout");
    assert.equal(result[0]!.task, "Investigate");
    assert.equal(result[0]!.reason, "needed");
  });
});
