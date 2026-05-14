/**
 * State safety integration tests.
 *
 * Proves all writes go through Go finalizers; no direct TS host writes to .aether/data/.
 */

import { readdirSync, readFileSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { describe, it } from "node:test";
import assert from "node:assert/strict";

import { assertNoDirectDataWrites, writeCompletionFile } from "../src/go-bridge.js";
import { GO_OWNED_PATHS } from "../src/boundary-reference.js";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function scanSourceForViolations(srcDir: string): string[] {
  const violations: string[] = [];
  const writeFunctions = [
    "writeFile",
    "writeFileSync",
    "appendFile",
    "appendFileSync",
    "mkdir",
    "mkdirSync",
    "createWriteStream",
    "rename",
    "copyFile",
  ];
  const forbiddenPattern = new RegExp(
    `(${writeFunctions.join("|")}).*\\.aether/data/`,
    "i"
  );

  function scanDir(dir: string): void {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
      const path = join(dir, entry.name);
      if (entry.isDirectory() && entry.name !== "test") {
        scanDir(path);
      } else if (entry.isFile() && entry.name.endsWith(".ts")) {
        const content = readFileSync(path, "utf-8");
        const lines = content.split("\n");
        for (let i = 0; i < lines.length; i++) {
          const line = lines[i]!;
          if (forbiddenPattern.test(line)) {
            violations.push(`${path}:${i + 1}: ${line.trim()}`);
          }
        }
      }
    }
  }

  scanDir(srcDir);
  return violations;
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("state-safety-integration", () => {
  it("rejects traversal attacks on Go-owned paths", () => {
    assert.throws(
      () => assertNoDirectDataWrites(".aether/data/../data/COLONY_STATE.json"),
      /Boundary violation/
    );
    assert.throws(
      () => assertNoDirectDataWrites("foo/.aether/data/session.json"),
      /Boundary violation/
    );
    assert.throws(
      () => assertNoDirectDataWrites("./.aether/data/pheromones.json"),
      /Boundary violation/
    );
  });

  it("writeCompletionFile rejects paths that escape tmpdir", () => {
    assert.throws(
      () => writeCompletionFile("../../../etc", "passwd.json", {}),
      /escapes tmpdir|Boundary violation/
    );
  });

  it("writeCompletionFile writes only within tmpdir", () => {
    const path = writeCompletionFile("aether-lifecycle", "test.json", { test: true });
    const resolved = path.replace(/\\/g, "/");
    const tmp = tmpdir().replace(/\\/g, "/");
    assert.ok(
      resolved.startsWith(tmp),
      `Completion file path ${resolved} should start with tmpdir ${tmp}`
    );
  });

  it("static analysis finds zero violations in src/", () => {
    const srcDir = join(import.meta.dirname, "..", "src");
    const violations = scanSourceForViolations(srcDir);

    if (violations.length > 0) {
      assert.fail(
        `Found ${violations.length} potential direct-write violations in src/:\n` +
          violations.join("\n")
      );
    }
  });

  it("GO_OWNED_PATHS covers all critical state files", () => {
    const expectedPaths = [
      ".aether/data/COLONY_STATE.json",
      ".aether/data/session.json",
      ".aether/data/pheromones.json",
      ".aether/data/constraints.json",
    ];

    for (const path of expectedPaths) {
      assert.ok(
        GO_OWNED_PATHS.some((p) => path.startsWith(p) || p.includes(path)),
        `GO_OWNED_PATHS should cover ${path}`
      );
    }
  });

  it("GO_OWNED_PATHS covers handoff and midden directories", () => {
    assert.ok(
      GO_OWNED_PATHS.some((p) => p.includes("handoffs")),
      "GO_OWNED_PATHS should include handoffs directory"
    );
    assert.ok(
      GO_OWNED_PATHS.some((p) => p.includes("midden")),
      "GO_OWNED_PATHS should include midden directory"
    );
  });

  it("boundary violation includes path in error message", () => {
    for (const path of GO_OWNED_PATHS) {
      try {
        assertNoDirectDataWrites(path);
        assert.fail(`Expected BoundaryViolationError for ${path}`);
      } catch (err) {
        assert.ok(err instanceof Error);
        assert.ok(
          err.message.includes("Boundary violation"),
          `Error message should include 'Boundary violation' for ${path}`
        );
        assert.ok(
          err.message.includes(path) || err.message.includes(".aether/data"),
          `Error message should reference the path for ${path}`
        );
      }
    }
  });
});
