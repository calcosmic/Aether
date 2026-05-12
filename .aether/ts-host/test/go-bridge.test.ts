/**
 * Integration tests for the Go bridge module.
 *
 * Tests verify:
 * - discoverGoBinary finds the aether CLI
 * - callGoJSON invokes Go commands with AETHER_OUTPUT_MODE=json
 * - assertNoDirectDataWrites enforces GO_OWNED_PATHS boundary
 * - writeCompletionFile writes to tmpdir, not .aether/data/
 */

import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { tmpdir } from "node:os";
import { fileURLToPath } from "node:url";

import { describe, it } from "node:test";
import assert from "node:assert/strict";

import {
  discoverGoBinary,
  callGoJSON,
  assertNoDirectDataWrites,
  writeCompletionFile,
} from "../src/go-bridge.js";
import type { GoBridgeOptions } from "../src/go-bridge.js";

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

/**
 * Set up a minimal test colony in a temp directory.
 * Creates .aether/data/ with a valid COLONY_STATE.json so that
 * `aether plan --plan-only` can run without errors.
 */
function setupTestColony(): {
  tempDir: string;
  cleanup: () => void;
  bridge: GoBridgeOptions;
} {
  const tempDir = mkdtempSync(join(tmpdir(), "ts-host-test-"));
  const dataDir = join(tempDir, ".aether", "data");

  mkdirSync(dataDir, { recursive: true });

  // Write a minimal valid colony state
  const colonyState = {
    version: "3.0",
    goal: "Test colony for TS host integration tests",
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

  const goBinaryPath = discoverGoBinary();
  const bridge: GoBridgeOptions = { goBinaryPath, cwd: tempDir };

  return {
    tempDir,
    cleanup: () => {
      rmSync(tempDir, { recursive: true, force: true });
    },
    bridge,
  };
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("go-bridge", () => {
  it("discoverGoBinary returns a non-empty path", () => {
    const path = discoverGoBinary();
    assert.ok(path, "discoverGoBinary should return a non-empty string");
    assert.ok(
      path.includes("aether"),
      `Path should contain "aether": ${path}`
    );
  });

  it("discoverGoBinary returns an executable path", () => {
    const path = discoverGoBinary();
    // Verify the binary actually runs
    const version = execFileSync(path, ["version"], { encoding: "utf-8" });
    assert.ok(version, "Binary should produce output for version command");
  });

  it("callGoJSON calls plan --plan-only and returns parsed JSON", () => {
    const { bridge, cleanup } = setupTestColony();
    try {
      const result = callGoJSON(bridge, [
        "plan",
        "--plan-only",
        "--depth",
        "fast",
      ]);

      // The plan-only result should be an object (the manifest)
      assert.ok(result, "callGoJSON should return a result");
      assert.equal(typeof result, "object", "Result should be an object");
    } finally {
      cleanup();
    }
  });

  it("assertNoDirectDataWrites throws for .aether/data/ paths", () => {
    assert.throws(
      () => assertNoDirectDataWrites(".aether/data/COLONY_STATE.json"),
      /Boundary violation/,
      "Should throw for .aether/data/ path"
    );

    assert.throws(
      () => assertNoDirectDataWrites(".aether/data/session.json"),
      /Boundary violation/,
      "Should throw for .aether/data/session.json"
    );

    assert.throws(
      () => assertNoDirectDataWrites(".aether/data/midden/midden.json"),
      /Boundary violation/,
      "Should throw for .aether/data/midden/ path"
    );
  });

  it("assertNoDirectDataWrites does NOT throw for safe paths", () => {
    assert.doesNotThrow(
      () => assertNoDirectDataWrites("/tmp/completion.json"),
      "Should not throw for /tmp path"
    );

    assert.doesNotThrow(
      () => assertNoDirectDataWrites("/var/folders/something/output.json"),
      "Should not throw for other temp paths"
    );
  });

  it("writeCompletionFile writes to tmpdir not .aether/data", () => {
    const path = writeCompletionFile(
      "aether-test-completions",
      "test-completion.json",
      { test: true, workers: [] }
    );

    // Path should be in tmpdir
    assert.ok(
      path.startsWith(tmpdir()),
      `Path should be in tmpdir: ${path}`
    );

    // Path should NOT contain .aether/data
    assert.ok(
      !path.includes(".aether/data"),
      `Path should not contain .aether/data: ${path}`
    );

    // File should exist and contain valid JSON
    assert.ok(existsSync(path), `File should exist: ${path}`);

    // Cleanup
    rmSync(join(tmpdir(), "aether-test-completions"), {
      recursive: true,
      force: true,
    });
  });
});
