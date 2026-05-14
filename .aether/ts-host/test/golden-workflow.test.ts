/**
 * Golden workflow test — captures the full ceremony output of a lifecycle run.
 *
 * Runs plan -> build -> continue with simulated workers, captures all rendered
 * output, and compares against a stored baseline snapshot.
 *
 * To update the snapshot after intentional ceremony changes:
 *   AETHER_UPDATE_SNAPSHOTS=1 npx tsx --test test/golden-workflow.test.ts
 */

import { mkdtempSync, mkdirSync, writeFileSync, readFileSync, rmSync, existsSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { describe, it, beforeEach, afterEach } from "node:test";
import assert from "node:assert/strict";

import { discoverGoBinary, callGoJSON } from "../src/go-bridge.js";
import { runLifecycle } from "../src/lifecycle.js";

import { readFileSync as readSnapshot, writeFileSync as writeSnapshot, existsSync as snapshotExists, mkdirSync as mkdirSnapshot } from "node:fs";
import { dirname } from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const SNAPSHOT_DIR = join(__dirname, "__snapshots__");
const UPDATE_SNAPSHOTS = process.env.AETHER_UPDATE_SNAPSHOTS === "1";

// ---------------------------------------------------------------------------
// Snapshot helpers
// ---------------------------------------------------------------------------

function goldenSnapshotPath(): string {
  return join(SNAPSHOT_DIR, "golden-workflow-spawn-tree.txt");
}

function loadGoldenSnapshot(): string | null {
  const path = goldenSnapshotPath();
  if (!snapshotExists(path)) return null;
  return readSnapshot(path, "utf-8");
}

function saveGoldenSnapshot(content: string): void {
  if (!snapshotExists(SNAPSHOT_DIR)) {
    mkdirSnapshot(SNAPSHOT_DIR, { recursive: true });
  }
  writeSnapshot(goldenSnapshotPath(), content, "utf-8");
}

function assertGoldenSnapshot(actual: string): void {
  const expected = loadGoldenSnapshot();

  if (expected === null || UPDATE_SNAPSHOTS) {
    saveGoldenSnapshot(actual);
    if (expected === null) {
      console.warn("[snapshot] Created missing golden workflow snapshot");
    } else {
      console.warn("[snapshot] Updated golden workflow snapshot");
    }
    return;
  }

  if (actual !== expected) {
    const preview = actual.slice(0, 300).replace(/\n/g, "\\n");
    const expPreview = expected.slice(0, 300).replace(/\n/g, "\\n");
    assert.fail(
      `Golden workflow snapshot mismatch\n` +
        `  Actual (first 300 chars):   ${preview}\n` +
        `  Expected (first 300 chars): ${expPreview}\n` +
        `  Run with AETHER_UPDATE_SNAPSHOTS=1 to regenerate.`
    );
  }
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

function setupTestColony(): { tempDir: string; cleanup: () => void } {
  const tempDir = mkdtempSync(join(tmpdir(), "ts-host-golden-test-"));
  const dataDir = join(tempDir, ".aether", "data");
  const plansDir = join(tempDir, ".aether", "plans");

  mkdirSync(dataDir, { recursive: true });
  mkdirSync(plansDir, { recursive: true });

  const colonyState = {
    version: "3.0",
    goal: "Golden workflow test colony",
    state: "READY",
    plan: { phases: [] },
    current_phase: 0,
  };
  writeFileSync(
    join(dataDir, "COLONY_STATE.json"),
    JSON.stringify(colonyState, null, 2),
    "utf-8"
  );
  writeFileSync(join(dataDir, "pheromones.json"), "[]", "utf-8");
  writeFileSync(join(dataDir, "constraints.json"), "[]", "utf-8");
  writeFileSync(join(dataDir, "session.json"), "{}", "utf-8");

  return {
    tempDir,
    cleanup: () => {
      rmSync(tempDir, { recursive: true, force: true });
    },
  };
}

function normalizeOutput(output: string): string {
  return output
    .replace(/\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z?/g, "<TIMESTAMP>")
    .replace(/\/var\/folders\/[^\/]+\/[^\/]+\/T\/[^\s\n]+/g, "<TMPDIR>")
    .replace(/\/tmp\/[^\s\n]+/g, "<TMPDIR>")
    .replace(/ts-host-golden-test-\w+/g, "<TMPDIR>")
    .replace(/ts-host-lifecycle-test-\w+/g, "<TMPDIR>")
    .replace(/process \d+/g, "process <PID>")
    .replace(/\d+\.\d+(ms|s)/g, "<DURATION>")
    .trim();
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("golden-workflow", () => {
  let context: ReturnType<typeof setupTestColony> | null = null;
  const originalWrite = process.stdout.write;
  let capturedOutput: string[] = [];

  beforeEach(() => {
    context = setupTestColony();
    capturedOutput = [];
    process.stdout.write = ((chunk: string | Buffer, encoding?: BufferEncoding | ((err?: Error) => void), cb?: (err?: Error) => void) => {
      const str = typeof chunk === "string" ? chunk : chunk.toString(encoding as BufferEncoding ?? "utf-8");
      capturedOutput.push(str);
      if (typeof cb === "function") cb();
      return true;
    }) as typeof process.stdout.write;
  });

  afterEach(() => {
    process.stdout.write = originalWrite;
    if (context) {
      context.cleanup();
    }
  });

  it("runLifecycle produces deterministic golden output", async () => {
    const goBinaryPath = discoverGoBinary();
    const result = await runLifecycle({
      goBinaryPath,
      cwd: context!.tempDir,
      simulateWorkers: true,
      phase: 1,
    });

    assert.equal(result.success, true, "Lifecycle should succeed");
    assert.deepEqual(result.steps_completed, ["plan", "build", "continue"]);

    // Combine stdout captures into normalized golden output
    const rawOutput = capturedOutput.join("");
    const normalized = normalizeOutput(rawOutput);

    assertGoldenSnapshot(normalized);
  });
});
