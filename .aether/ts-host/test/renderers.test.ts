/**
 * Renderer unit tests.
 *
 * Verifies visual, markdown, and json renderer output.
 */

import { describe, it } from "node:test";
import assert from "node:assert/strict";

import { visualRenderer } from "../src/renderers/visual.js";
import { markdownRenderer } from "../src/renderers/markdown.js";
import { jsonRenderer } from "../src/renderers/json.js";
import { loadCeremonyConfig } from "../src/caste-config.js";
import type { CeremonyPayload } from "../src/types.js";

const REPO_ROOT = "/Users/callumcowie/repos/Aether";
const config = loadCeremonyConfig(REPO_ROOT);

// ---------------------------------------------------------------------------
// Visual renderer
// ---------------------------------------------------------------------------

describe("visualRenderer", () => {
  it("renderBanner returns multi-line figlet string", () => {
    const result = visualRenderer.renderBanner("BUILD");
    const lines = result.split("\n");
    assert.ok(lines.length > 1, "Should produce multi-line figlet output");
    // Figlet ASCII art uses block characters like |, _, \, /, (, )
    const hasArt = lines.some((l) => /[|_\\/()]/.test(l));
    assert.ok(hasArt, "Should contain figlet ASCII art characters");
  });

  it("renderSpawnFrame includes emoji and colored label", () => {
    const payload: CeremonyPayload = {
      caste: "builder",
      name: "Mason-67",
      task: "Task 1",
    };
    const result = visualRenderer.renderSpawnFrame(payload, config);
    assert.ok(result.includes("🔨"), "Should include builder emoji");
    assert.ok(result.includes("Builder"), "Should include builder label");
    assert.ok(result.includes("Mason-67"), "Should include name");
    assert.ok(result.includes("Task 1"), "Should include task");
  });

  it("renderStageSeparator uses prefix and suffix", () => {
    const result = visualRenderer.renderStageSeparator("Build", config);
    assert.ok(result.startsWith("── "), "Should start with prefix");
    assert.ok(result.includes("Build"), "Should include stage name");
    assert.ok(result.endsWith(" ──\n"), "Should end with suffix and newline");
  });

  it("renderBox returns boxen-framed string", () => {
    const result = visualRenderer.renderBox("Hello");
    // Boxen "round" style uses ╭, ╮, ╰, ╯
    assert.ok(result.includes("╭"), "Should have top-left rounded border");
    assert.ok(result.includes("╰"), "Should have bottom-left rounded border");
    assert.ok(result.includes("Hello"), "Should include content");
  });
});

// ---------------------------------------------------------------------------
// Markdown renderer
// ---------------------------------------------------------------------------

describe("markdownRenderer", () => {
  it("strips ANSI from visual output", () => {
    const visualBanner = visualRenderer.renderBanner("TEST");
    const mdBanner = markdownRenderer.renderBanner("TEST");
    // Markdown must never contain ANSI escape sequences
    assert.ok(!mdBanner.includes("\x1b["), "Markdown should strip ANSI codes");
    // Both should contain the same figlet art structure (sans color)
    assert.ok(mdBanner.includes("TEST") || /[|_\\/()]/.test(mdBanner), "Should preserve figlet structure");
  });

  it("preserves emojis after stripping ANSI", () => {
    const payload: CeremonyPayload = {
      caste: "builder",
      name: "Mason-67",
      task: "Task 1",
    };
    const result = markdownRenderer.renderSpawnFrame(payload, config);
    assert.ok(!result.includes("\x1b["), "Should not contain ANSI codes");
    assert.ok(result.includes("🔨"), "Should preserve emoji");
    assert.ok(result.includes("Builder"), "Should preserve label");
  });
});

// ---------------------------------------------------------------------------
// JSON renderer
// ---------------------------------------------------------------------------

describe("jsonRenderer", () => {
  it("returns empty string for all methods", () => {
    const payload: CeremonyPayload = {
      caste: "builder",
      name: "Mason-67",
      task: "Task 1",
    };

    assert.equal(jsonRenderer.renderBanner("TEST"), "");
    assert.equal(jsonRenderer.renderSpawnFrame(payload, config), "");
    assert.equal(jsonRenderer.renderStageSeparator("Build", config), "");
    assert.equal(jsonRenderer.renderBox("Hello"), "");
  });
});
