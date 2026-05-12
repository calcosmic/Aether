/**
 * Integration tests for the host entry point.
 *
 * Tests verify:
 * - Host module source file exists
 * - Host prints usage when called with no args
 * - Host lifecycle command gives not-yet-implemented message
 */

import { execFileSync, spawnSync } from "node:child_process";
import { existsSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

import { describe, it } from "node:test";
import assert from "node:assert/strict";

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

  it("host lifecycle command prints not-yet-implemented message", () => {
    const result = spawnSync(
      "node",
      ["--import", "tsx", hostPath, "lifecycle"],
      {
        encoding: "utf-8",
        timeout: 10000,
      }
    );

    const stderr = result.stderr ?? "";
    assert.ok(
      stderr.includes("not yet implemented") ||
        stderr.includes("Lifecycle"),
      `Stderr should mention not yet implemented. Got: ${stderr.slice(0, 200)}`
    );
  });
});
