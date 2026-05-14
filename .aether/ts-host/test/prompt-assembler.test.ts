/**
 * Prompt assembler unit tests.
 *
 * Verifies agent definition loading, prompt assembly, response contract
 * rendering, and caste-to-agent name mapping.
 */

import { describe, it } from "node:test";
import assert from "node:assert/strict";

import {
  loadAgentDefinition,
  assemblePrompt,
  renderResponseContract,
  getAgentNameForCaste,
} from "../src/prompt-assembler.js";

const REPO_ROOT = "/Users/callumcowie/repos/Aether";

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("prompt-assembler", () => {
  it("loadAgentDefinition reads aether-builder.md for builder caste", () => {
    const content = loadAgentDefinition(REPO_ROOT, "claude", "aether-builder");
    assert.ok(content.length > 0, "Should return non-empty content");
    assert.ok(
      content.includes("Builder"),
      "Should include 'Builder' in the agent definition"
    );
  });

  it("assemblePrompt includes agent definition and task brief", () => {
    const prompt = assemblePrompt({
      cwd: REPO_ROOT,
      caste: "builder",
      name: "Mason-67",
      task: "Implement feature X",
      platform: "claude",
      agentName: "aether-builder",
    });

    assert.ok(prompt.includes("Builder"), "Should include agent definition content");
    assert.ok(
      prompt.includes("Implement feature X"),
      "Should include task brief"
    );
    assert.ok(
      prompt.includes("Final Response Contract"),
      "Should include response contract"
    );
  });

  it("renderResponseContract includes required JSON fields", () => {
    const contract = renderResponseContract({
      cwd: REPO_ROOT,
      caste: "builder",
      name: "Mason-67",
      task: "Implement feature X",
      platform: "claude",
      agentName: "aether-builder",
    });

    assert.ok(contract.includes("ant_name"), "Should mention ant_name");
    assert.ok(contract.includes("status"), "Should mention status");
    assert.ok(contract.includes("summary"), "Should mention summary");
    assert.ok(contract.includes("files_created"), "Should mention files_created");
    assert.ok(contract.includes("handoff"), "Should mention handoff");
  });

  it("getAgentNameForCaste maps builder to aether-builder", () => {
    assert.equal(getAgentNameForCaste("builder"), "aether-builder");
    assert.equal(getAgentNameForCaste("watcher"), "aether-watcher");
    assert.equal(getAgentNameForCaste("scout"), "aether-scout");
    assert.equal(
      getAgentNameForCaste("unknown-caste"),
      "aether-unknown-caste",
      "Unknown castes should fallback to aether-<caste>"
    );
  });
});
