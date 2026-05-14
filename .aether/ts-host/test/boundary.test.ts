/**
 * Boundary enforcement tests for the TypeScript orchestration host.
 *
 * Tests verify HOST-05: No TypeScript code writes to .aether/data/ directly.
 * All state mutations must go through Go finalizer commands.
 *
 * These tests validate both the runtime boundary enforcement functions and
 * the static code analysis proving no .aether/data/ writes exist in src/.
 */

import { existsSync, readdirSync, readFileSync, rmSync, statSync } from "node:fs";
import { join, dirname } from "node:path";
import { tmpdir } from "node:os";
import { fileURLToPath } from "node:url";

import { describe, it } from "node:test";
import assert from "node:assert/strict";

import {
  assertNoDirectDataWrites,
  writeCompletionFile,
} from "../src/go-bridge.js";
import { GO_OWNED_PATHS } from "../src/boundary-reference.js";

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

const __dirname = dirname(fileURLToPath(import.meta.url));
const srcDir = join(__dirname, "..", "src");

/**
 * Recursively read all .ts files in a directory.
 */
function readAllTsFiles(dir: string): Map<string, string> {
  const files = new Map<string, string>();

  for (const entry of readdirSync(dir)) {
    const fullPath = join(dir, entry);
    const stat = statSync(fullPath);
    if (stat.isDirectory()) {
      for (const [path, content] of readAllTsFiles(fullPath)) {
        files.set(path, content);
      }
    } else if (entry.endsWith(".ts")) {
      files.set(fullPath, readFileSync(fullPath, "utf-8"));
    }
  }

  return files;
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("boundary-enforcement", () => {
  it("assertNoDirectDataWrites rejects .aether/data/COLONY_STATE.json", () => {
    assert.throws(
      () => assertNoDirectDataWrites(".aether/data/COLONY_STATE.json"),
      /Boundary violation/,
      "Should throw for .aether/data/COLONY_STATE.json"
    );
  });

  it("assertNoDirectDataWrites rejects .aether/data/session.json", () => {
    assert.throws(
      () => assertNoDirectDataWrites(".aether/data/session.json"),
      /Boundary violation/,
      "Should throw for .aether/data/session.json"
    );
  });

  it("assertNoDirectDataWrites rejects .aether/data/pheromones.json", () => {
    assert.throws(
      () => assertNoDirectDataWrites(".aether/data/pheromones.json"),
      /Boundary violation/,
      "Should throw for .aether/data/pheromones.json"
    );
  });

  it("assertNoDirectDataWrites rejects nested .aether/data paths", () => {
    assert.throws(
      () => assertNoDirectDataWrites(".aether/data/midden/midden.json"),
      /Boundary violation/,
      "Should throw for nested .aether/data/ paths"
    );

    assert.throws(
      () => assertNoDirectDataWrites(".aether/data/handoffs/worker-handoffs.json"),
      /Boundary violation/,
      "Should throw for .aether/data/handoffs/ paths"
    );
  });

  it("assertNoDirectDataWrites allows tmpdir paths", () => {
    assert.doesNotThrow(
      () => assertNoDirectDataWrites("/tmp/completion.json"),
      "Should not throw for /tmp path"
    );

    assert.doesNotThrow(
      () => assertNoDirectDataWrites(join(tmpdir(), "aether-completions", "build.json")),
      "Should not throw for tmpdir paths"
    );
  });

  it("assertNoDirectDataWrites allows arbitrary non-data paths", () => {
    assert.doesNotThrow(
      () => assertNoDirectDataWrites("/var/folders/output.json"),
      "Should not throw for other temp paths"
    );

    assert.doesNotThrow(
      () => assertNoDirectDataWrites("/Users/test/repo/src/file.ts"),
      "Should not throw for source file paths"
    );
  });

  it("writeCompletionFile writes to tmpdir", () => {
    const path = writeCompletionFile(
      "aether-boundary-test",
      "test-completion.json",
      { test: true, dispatches: [] }
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
    const content = JSON.parse(readFileSync(path, "utf-8"));
    assert.ok(content.test === true, "File should contain the test data");

    // Cleanup
    rmSync(join(tmpdir(), "aether-boundary-test"), {
      recursive: true,
      force: true,
    });
  });

  it("writeCompletionFile always uses tmpdir regardless of input", () => {
    // Even if we try to pass a data-like subdirectory name, it should still
    // end up in tmpdir, not in .aether/data/
    const path = writeCompletionFile(
      "completions",
      "build-completion.json",
      { manifest: {}, dispatches: [] }
    );

    assert.ok(
      path.startsWith(tmpdir()),
      `Path should still be in tmpdir: ${path}`
    );
    assert.ok(
      !path.includes(".aether/data"),
      `Path should never contain .aether/data: ${path}`
    );

    // Cleanup
    rmSync(join(tmpdir(), "completions"), {
      recursive: true,
      force: true,
    });
  });

  it("no TypeScript source file writes to .aether/data", () => {
    // Scan all .ts files in src/ for patterns that would write to .aether/data
    const srcFiles = readAllTsFiles(srcDir);

    // Patterns that indicate a direct write to .aether/data
    const forbiddenPatterns = [
      /writeFile.*\.aether\/data/,
      /writeFileSync.*\.aether\/data/,
      /appendFile.*\.aether\/data/,
      /appendFileSync.*\.aether\/data/,
      /mkdir.*\.aether\/data/,
      /mkdirSync.*\.aether\/data/,
      /createWriteStream.*\.aether\/data/,
    ];

    const violations: string[] = [];

    for (const [filePath, content] of srcFiles) {
      const lines = content.split("\n");
      for (let i = 0; i < lines.length; i++) {
        const line = lines[i]!;
        for (const pattern of forbiddenPatterns) {
          if (pattern.test(line)) {
            violations.push(
              `${filePath}:${i + 1}: ${line.trim()}`
            );
          }
        }
      }
    }

    assert.equal(
      violations.length,
      0,
      `No TypeScript source should write to .aether/data. Violations:\n${violations.join("\n")}`
    );
  });

  it("GO_OWNED_PATHS covers all critical state files", () => {
    // Verify GO_OWNED_PATHS includes the critical state files
    const criticalPaths = [
      ".aether/data/COLONY_STATE.json",
      ".aether/data/session.json",
      ".aether/data/pheromones.json",
    ];

    for (const criticalPath of criticalPaths) {
      assert.ok(
        GO_OWNED_PATHS.includes(criticalPath as typeof GO_OWNED_PATHS[number]),
        `GO_OWNED_PATHS should include ${criticalPath}. ` +
          `Current paths: ${GO_OWNED_PATHS.join(", ")}`
      );
    }
  });

  it("GO_OWNED_PATHS covers handoff and midden directories", () => {
    const dirPaths = [
      ".aether/data/handoffs/",
      ".aether/data/midden/",
    ];

    for (const dirPath of dirPaths) {
      assert.ok(
        GO_OWNED_PATHS.includes(dirPath as typeof GO_OWNED_PATHS[number]),
        `GO_OWNED_PATHS should include ${dirPath}`
      );
    }
  });
});
