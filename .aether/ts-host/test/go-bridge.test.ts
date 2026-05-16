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

  it("callGoJSON redacts failed subprocess provider output", () => {
    const tempDir = mkdtempSync(join(tmpdir(), "ts-host-bridge-failure-"));
    const fakeGo = join(tempDir, "aether-fake");
    writeFileSync(
      fakeGo,
      "#!/bin/sh\nprintf 'stdout sk-proj-secret-123\\n'\nprintf 'stderr ghp_secret_123\\n' >&2\nexit 42\n",
      { encoding: "utf-8", mode: 0o755 }
    );

    try {
      assert.throws(
        () => callGoJSON({ goBinaryPath: fakeGo, cwd: tempDir }, ["build", "2", "--plan-only"]),
        (err: unknown) => {
          assert.ok(err instanceof Error);
          assert.ok(err.message.includes("exit status 42"), err.message);
          for (const forbidden of ["sk-proj-secret-123", "ghp_secret_123", "stdout sk", "stderr ghp"]) {
            assert.ok(!err.message.includes(forbidden), `leaked ${forbidden}: ${err.message}`);
          }
          return true;
        }
      );
    } finally {
      rmSync(tempDir, { recursive: true, force: true });
    }
  });

  it("callGoJSON redacts Go error envelopes", () => {
    const tempDir = mkdtempSync(join(tmpdir(), "ts-host-bridge-envelope-"));
    const fakeGo = join(tempDir, "aether-fake");
    writeFileSync(
      fakeGo,
      "#!/bin/sh\nprintf '{\"ok\":false,\"error\":\"auth failed token sk-proj-secret-456 stderr: ghp_secret_456\",\"code\":2}\\n' >&2\nexit 2\n",
      { encoding: "utf-8", mode: 0o755 }
    );

    try {
      assert.throws(
        () => callGoJSON({ goBinaryPath: fakeGo, cwd: tempDir }, ["build", "2", "--plan-only"]),
        (err: unknown) => {
          assert.ok(err instanceof Error);
          assert.ok(err.message.includes("Go command failed"), err.message);
          for (const forbidden of ["sk-proj-secret-456", "ghp_secret_456"]) {
            assert.ok(!err.message.includes(forbidden), `leaked ${forbidden}: ${err.message}`);
          }
          assert.ok(err.message.includes("[redacted]") || err.message.includes("[omitted]"), err.message);
          return true;
        }
      );
    } finally {
      rmSync(tempDir, { recursive: true, force: true });
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
