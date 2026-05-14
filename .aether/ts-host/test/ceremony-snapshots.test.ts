/**
 * Ceremony snapshot tests.
 *
 * Captures renderer output for all ceremony templates and compares against
 * stored baseline snapshots in test/__snapshots__/*.txt.
 *
 * Snapshot update: AETHER_UPDATE_SNAPSHOTS=1 npm test
 */

import { readFileSync, writeFileSync, existsSync, mkdirSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

import { describe, it } from "node:test";
import assert from "node:assert/strict";

import { visualRenderer } from "../src/renderers/visual.js";
import { markdownRenderer } from "../src/renderers/markdown.js";
import { jsonRenderer } from "../src/renderers/json.js";
import { loadCeremonyConfig } from "../src/caste-config.js";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const REPO_ROOT = "/Users/callumcowie/repos/Aether";
const config = loadCeremonyConfig(REPO_ROOT);

const SNAPSHOT_DIR = join(__dirname, "__snapshots__");
const UPDATE_SNAPSHOTS = process.env["AETHER_UPDATE_SNAPSHOTS"] === "1";

// ---------------------------------------------------------------------------
// Snapshot helpers
// ---------------------------------------------------------------------------

function loadSnapshot(name: string): string | undefined {
  const path = join(SNAPSHOT_DIR, `${name}.txt`);
  if (!existsSync(path)) return undefined;
  return readFileSync(path, "utf-8");
}

function saveSnapshot(name: string, content: string): void {
  if (!existsSync(SNAPSHOT_DIR)) {
    mkdirSync(SNAPSHOT_DIR, { recursive: true });
  }
  const path = join(SNAPSHOT_DIR, `${name}.txt`);
  writeFileSync(path, content, "utf-8");
}

function assertSnapshot(name: string, actual: string): void {
  if (UPDATE_SNAPSHOTS) {
    saveSnapshot(name, actual);
    return;
  }

  const expected = loadSnapshot(name);
  if (expected === undefined) {
    saveSnapshot(name, actual);
    process.stderr.write(
      `Warning: created missing snapshot ${name}.txt (run with AETHER_UPDATE_SNAPSHOTS=1 to regenerate)\n`
    );
    return;
  }

  if (actual !== expected) {
    const lines = actual.split("\n");
    const expectedLines = expected.split("\n");
    const maxLen = Math.max(lines.length, expectedLines.length);
    const diff: string[] = [];
    for (let i = 0; i < maxLen; i++) {
      const a = lines[i] ?? "(missing)";
      const e = expectedLines[i] ?? "(missing)";
      if (a !== e) {
        diff.push(`  line ${i + 1}:`);
        diff.push(`    expected: ${JSON.stringify(e)}`);
        diff.push(`    actual:   ${JSON.stringify(a)}`);
      }
    }
    throw new Error(
      `Snapshot mismatch for ${name}.txt\n${diff.join("\n")}\n\n` +
        `Run AETHER_UPDATE_SNAPSHOTS=1 to regenerate snapshots.`
    );
  }
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("ceremony snapshots", () => {
  it("renderBanner(BUILD) matches snapshot", () => {
    const result = visualRenderer.renderBanner("BUILD");
    assertSnapshot("banner-build-start", result);
  });

  it("renderBanner(CROWNED ANTHILL) matches snapshot", () => {
    const result = visualRenderer.renderBanner("CROWNED ANTHILL");
    assertSnapshot("banner-seal-complete", result);
  });

  it("renderSpawnFrame(builder) matches snapshot", () => {
    const result = visualRenderer.renderSpawnFrame(
      { caste: "builder", name: "Mason-67", task: "Build the wall" },
      config
    );
    assertSnapshot("spawn-frame-builder", result);
  });

  it("renderSpawnFrame(oracle) matches snapshot", () => {
    const result = visualRenderer.renderSpawnFrame(
      { caste: "oracle", name: "Seer-42", task: "Research patterns" },
      config
    );
    assertSnapshot("spawn-frame-oracle", result);
  });

  it("renderStageSeparator(Build) matches snapshot", () => {
    const result = visualRenderer.renderStageSeparator("Build", config);
    assertSnapshot("stage-separator-build", result);
  });

  it("renderStageSeparator(Continue) matches snapshot", () => {
    const result = visualRenderer.renderStageSeparator("Continue", config);
    assertSnapshot("stage-separator-continue", result);
  });

  it("renderBox(build-summary) matches snapshot", () => {
    const result = visualRenderer.renderBox(
      "Completed: 3/3\nTools: 12\nTokens: 4200",
      { borderStyle: "round", borderColor: "green" }
    );
    assertSnapshot("build-summary", result);
  });

  it("renderBox(closeout-ritual) matches snapshot", () => {
    const result = visualRenderer.renderBox(
      "Phase 1\nStatus: Crowned Anthill",
      { borderStyle: "double", borderColor: "cyan" }
    );
    assertSnapshot("closeout-ritual", result);
  });

  it("markdownRenderer strips ANSI while preserving structure", () => {
    const result = markdownRenderer.renderBanner("TEST");
    assert.ok(!result.includes("\x1b["), "Markdown should not contain ANSI codes");
    // Should still contain figlet art characters
    assert.ok(/[|_\\/()]/.test(result), "Should preserve figlet structure");
  });

  it("jsonRenderer returns empty strings for all methods", () => {
    assert.equal(jsonRenderer.renderBanner("TEST"), "");
    assert.equal(
      jsonRenderer.renderSpawnFrame(
        { caste: "builder", name: "Mason-67", task: "Build the wall" },
        config
      ),
      ""
    );
    assert.equal(jsonRenderer.renderStageSeparator("Build", config), "");
    assert.equal(jsonRenderer.renderBox("Hello"), "");
  });
});
