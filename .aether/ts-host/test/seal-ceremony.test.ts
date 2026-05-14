/**
 * Seal ceremony end-to-end test.
 *
 * Simulates the full seal ritual: Sage wisdom review, Chronicler documentation
 * update, wisdom promotion, and Crowned Anthill closeout.
 */

import { describe, it } from "node:test";
import assert from "node:assert/strict";

import { visualRenderer } from "../src/renderers/visual.js";
import { DEFAULT_CEREMONY_CONFIG } from "../src/caste-config.js";
import { readFileSync as readSnapshot, writeFileSync as writeSnapshot, existsSync as snapshotExists, mkdirSync as mkdirSnapshot } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const SNAPSHOT_DIR = join(__dirname, "__snapshots__");
const UPDATE_SNAPSHOTS = process.env.AETHER_UPDATE_SNAPSHOTS === "1";

// ---------------------------------------------------------------------------
// Snapshot helpers
// ---------------------------------------------------------------------------

function sealSnapshotPath(): string {
  return join(SNAPSHOT_DIR, "seal-ceremony.txt");
}

function loadSealSnapshot(): string | null {
  const path = sealSnapshotPath();
  if (!snapshotExists(path)) return null;
  return readSnapshot(path, "utf-8");
}

function saveSealSnapshot(content: string): void {
  if (!snapshotExists(SNAPSHOT_DIR)) {
    mkdirSnapshot(SNAPSHOT_DIR, { recursive: true });
  }
  writeSnapshot(sealSnapshotPath(), content, "utf-8");
}

function assertSealSnapshot(actual: string): void {
  const expected = loadSealSnapshot();

  if (expected === null || UPDATE_SNAPSHOTS) {
    saveSealSnapshot(actual);
    if (expected === null) {
      console.warn("[snapshot] Created missing seal ceremony snapshot");
    } else {
      console.warn("[snapshot] Updated seal ceremony snapshot");
    }
    return;
  }

  if (actual !== expected) {
    const preview = actual.slice(0, 300).replace(/\n/g, "\\n");
    const expPreview = expected.slice(0, 300).replace(/\n/g, "\\n");
    assert.fail(
      `Seal ceremony snapshot mismatch\n` +
        `  Actual (first 300 chars):   ${preview}\n` +
        `  Expected (first 300 chars): ${expPreview}\n` +
        `  Run with AETHER_UPDATE_SNAPSHOTS=1 to regenerate.`
    );
  }
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("seal-ceremony", () => {
  const config = DEFAULT_CEREMONY_CONFIG;

  it("produces deterministic seal ceremony output", async () => {
    // Simulate seal ceremony event sequence using renderer directly
    const parts: string[] = [];

    // Seal wave start
    parts.push(visualRenderer.renderStageSeparator("Seal", config));

    // Sage spawn
    parts.push(
      visualRenderer.renderSpawnFrame(
        { caste: "sage", name: "Sage-1", task: "Review wisdom" },
        config
      )
    );

    // Chronicler spawn
    parts.push(
      visualRenderer.renderSpawnFrame(
        { caste: "chronicler", name: "Chronicler-1", task: "Update docs" },
        config
      )
    );

    // Build summary
    parts.push(
      visualRenderer.renderBox(
        "Phase: 1\nWorkers: 2\nTools: 8\nTokens: 15000",
        { borderStyle: "round", borderColor: "green" }
      )
    );

    // Crowned Anthill banner
    parts.push(visualRenderer.renderBanner("CROWNED ANTHILL"));

    // Closeout ritual
    parts.push(
      visualRenderer.renderBox(
        "Phase 1\nStatus: Crowned Anthill\nNext: Start new colony",
        { borderStyle: "double", borderColor: "cyan" }
      )
    );

    const rawOutput = parts.join("");

    // Key content assertions
    assert.ok(rawOutput.includes("Sage"), "Output should contain Sage spawn frame");
    assert.ok(rawOutput.includes("Chronicler"), "Output should contain Chronicler spawn frame");
    assert.ok(rawOutput.includes("Crowned Anthill"), "Output should contain closeout ritual content");

    // Compare normalized snapshot (strip ANSI for readable git diffs)
    const { default: stripAnsi } = await import("strip-ansi");
    const normalized = stripAnsi(rawOutput).trim();
    assertSealSnapshot(normalized);
  });
});
