/**
 * Prompt assembler unit tests.
 *
 * Verifies agent definition loading, prompt assembly, response contract
 * rendering, and caste-to-agent name mapping.
 */

import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { mkdirSync, mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

import {
  loadAgentDefinition,
  assemblePrompt,
  renderContextCapsule,
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

  it("loadAgentDefinition falls back to hub Claude agents when repo-local agents are absent", () => {
    const root = mkdtempSync(join(tmpdir(), "aether-prompt-consumer-"));
    const home = mkdtempSync(join(tmpdir(), "aether-prompt-home-"));
    const hub = join(home, ".aether");
    const agentDir = join(hub, "system", "agents-claude");
    mkdirSync(agentDir, { recursive: true });
    writeFileSync(
      join(agentDir, "aether-builder.md"),
      "# Hub Claude Builder\n\nPublished from the Aether hub.",
      "utf-8"
    );

    const originalHub = process.env["AETHER_HUB_DIR"];
    process.env["AETHER_HUB_DIR"] = hub;
    try {
      const content = loadAgentDefinition(root, "claude", "aether-builder", home);
      assert.match(content, /Hub Claude Builder/);
    } finally {
      restoreEnv("AETHER_HUB_DIR", originalHub);
    }
  });

  it("loadAgentDefinition falls back to hub Codex agents when repo-local agents are absent", () => {
    const root = mkdtempSync(join(tmpdir(), "aether-prompt-consumer-"));
    const home = mkdtempSync(join(tmpdir(), "aether-prompt-home-"));
    const hub = join(home, ".aether");
    const agentDir = join(hub, "system", "codex");
    mkdirSync(agentDir, { recursive: true });
    writeFileSync(
      join(agentDir, "aether-builder.toml"),
      'name = "aether-builder"\ndescription = "Hub Codex Builder"\n',
      "utf-8"
    );

    const originalHub = process.env["AETHER_HUB_DIR"];
    process.env["AETHER_HUB_DIR"] = hub;
    try {
      const content = loadAgentDefinition(root, "codex", "aether-builder", home);
      assert.match(content, /Hub Codex Builder/);
    } finally {
      restoreEnv("AETHER_HUB_DIR", originalHub);
    }
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

  it("assemblePrompt includes Go-provided context, handoff, and skill sections", () => {
    const prompt = assemblePrompt({
      cwd: REPO_ROOT,
      caste: "builder",
      name: "Mason-67",
      task: "Implement feature X",
      platform: "claude",
      agentName: "aether-builder",
      contextCapsule: "## Colony State\n\nPhase: 6",
      handoffSection: "## Previous Worker Handoffs\n\n- Builder completed setup",
      skillSection: "### Skill: test-skill\n\nUse the shared skill section.",
      pheromoneSection: "## Pheromone Signals\n\n- [FOCUS] prompt context",
      taskBrief: "# Codex Build Dispatch\n\nImplement feature X from the Go manifest.",
    });

    assert.ok(prompt.includes("## Colony State"), "Should include Go context");
    assert.ok(prompt.includes("Previous Worker Handoffs"), "Should include handoffs");
    assert.ok(prompt.includes("### Skill: test-skill"), "Should include skills");
    assert.ok(prompt.includes("[FOCUS] prompt context"), "Should include pheromones");
    assert.ok(prompt.includes("# Codex Build Dispatch"), "Should include Go task brief");
    assert.ok(!prompt.includes("## Goal\n\nImplement feature X"), "Should not add fallback task brief when Go task brief is present");
  });

  it("renderContextCapsule loads repo Queen before global Queen without CommonJS require", () => {
    const root = mkdtempSync(join(tmpdir(), "aether-prompt-root-"));
    const home = mkdtempSync(join(tmpdir(), "aether-prompt-home-"));
    mkdirSync(join(root, ".aether"), { recursive: true });
    mkdirSync(join(home, ".aether"), { recursive: true });
    writeFileSync(join(root, ".aether", "QUEEN.md"), "# Local Queen\n\nRepo-only wisdom", "utf-8");
    writeFileSync(join(home, ".aether", "QUEEN.md"), "# Global Queen\n\nUser-level wisdom", "utf-8");

    const capsule = renderContextCapsule({
      cwd: root,
      caste: "builder",
      name: "Mason-67",
      task: "Implement feature X",
      platform: "claude",
      agentName: "aether-builder",
      homeDir: home,
    });

    assert.match(capsule, /Repo Queen Wisdom/);
    assert.match(capsule, /Local Queen/);
    assert.match(capsule, /Global Queen Wisdom/);
    assert.match(capsule, /Global Queen/);
    assert.ok(
      capsule.indexOf("Repo Queen Wisdom") < capsule.indexOf("Global Queen Wisdom"),
      "Repo Queen should appear before global Queen"
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

  // ---------------------------------------------------------------------------
  // Skill section injection tests (SKILL-01, SKILL-02, SKILL-03)
  // ---------------------------------------------------------------------------

  it("skill section appears in prompt when skillSection is a non-empty string (SKILL-01)", () => {
    const prompt = assemblePrompt({
      cwd: REPO_ROOT,
      caste: "builder",
      name: "Mason-67",
      task: "Implement feature X",
      platform: "claude",
      agentName: "aether-builder",
      skillSection: "## Skills\n- Go testing patterns\n- Error handling",
    });

    assert.ok(
      prompt.includes("## Skills"),
      "Prompt should contain the skill section header when skillSection is provided"
    );
    assert.ok(
      prompt.includes("Go testing patterns"),
      "Prompt should contain the skill content text"
    );
  });

  it("skill section is absent when skillSection is undefined (SKILL-02)", () => {
    const prompt = assemblePrompt({
      cwd: REPO_ROOT,
      caste: "builder",
      name: "Mason-67",
      task: "Implement feature X",
      platform: "claude",
      agentName: "aether-builder",
      // skillSection intentionally omitted
    });

    // The prompt should not contain any skill-related headers
    // Note: "skill" lowercase might appear in other sections (e.g., agent definition),
    // so we check for the specific header format
    assert.ok(
      !prompt.includes("## Skills\n"),
      "Prompt should not contain a ## Skills header when skillSection is undefined"
    );
    assert.ok(
      !prompt.includes("### Skill:"),
      "Prompt should not contain skill subsections when skillSection is undefined"
    );
  });

  it("skill section handles non-string skillSection gracefully - number (SKILL-03)", () => {
    // compactSection returns "" for non-strings, so the prompt should not crash
    const prompt = assemblePrompt({
      cwd: REPO_ROOT,
      caste: "builder",
      name: "Mason-67",
      task: "Implement feature X",
      platform: "claude",
      agentName: "aether-builder",
      skillSection: 42 as unknown as string,
    });

    assert.ok(
      typeof prompt === "string",
      "assemblePrompt should return a string even with non-string skillSection"
    );
    assert.ok(
      !prompt.includes("## Skills\n"),
      "Prompt should not contain a ## Skills header for non-string skillSection"
    );
  });

  it("skill section handles null skillSection gracefully (SKILL-03)", () => {
    const prompt = assemblePrompt({
      cwd: REPO_ROOT,
      caste: "builder",
      name: "Mason-67",
      task: "Implement feature X",
      platform: "claude",
      agentName: "aether-builder",
      skillSection: null as unknown as string,
    });

    assert.ok(
      typeof prompt === "string",
      "assemblePrompt should return a string even with null skillSection"
    );
    assert.ok(
      !prompt.includes("## Skills\n"),
      "Prompt should not contain a ## Skills header for null skillSection"
    );
  });

  it("skill section handles empty string skillSection gracefully (SKILL-03)", () => {
    const prompt = assemblePrompt({
      cwd: REPO_ROOT,
      caste: "builder",
      name: "Mason-67",
      task: "Implement feature X",
      platform: "claude",
      agentName: "aether-builder",
      skillSection: "",
    });

    assert.ok(
      typeof prompt === "string",
      "assemblePrompt should return a string even with empty skillSection"
    );
    // Empty string after compactSection trim is "", which is falsy, so it should not be pushed
    // Check that there are no empty section gaps (consecutive double newlines with nothing between)
    assert.ok(
      !/\n\n\n\n/.test(prompt),
      "Prompt should not have empty section gaps from empty skillSection"
    );
  });

  // ---------------------------------------------------------------------------
  // Hive section injection tests (HIVE-03)
  // ---------------------------------------------------------------------------

  it("hive section appears in prompt when hiveSection is a non-empty string", () => {
    const prompt = assemblePrompt({
      cwd: REPO_ROOT,
      caste: "builder",
      name: "Mason-67",
      task: "Implement feature X",
      platform: "claude",
      agentName: "aether-builder",
      hiveSection: "## HIVE WISDOM (Cross-Colony Patterns)\n\n(go, 0.90) Prefer table-driven tests",
    });

    assert.ok(
      prompt.includes("## HIVE WISDOM (Cross-Colony Patterns)"),
      "Prompt should contain the hive section header when hiveSection is provided"
    );
    assert.ok(
      prompt.includes("(go, 0.90) Prefer table-driven tests"),
      "Prompt should contain the hive wisdom content text"
    );
  });

  it("hive section is absent when hiveSection is undefined (HIVE-03)", () => {
    const prompt = assemblePrompt({
      cwd: REPO_ROOT,
      caste: "builder",
      name: "Mason-67",
      task: "Implement feature X",
      platform: "claude",
      agentName: "aether-builder",
      // hiveSection intentionally omitted
    });

    assert.ok(
      !prompt.includes("## HIVE WISDOM (Cross-Colony Patterns)"),
      "Prompt should not contain a hive section header when hiveSection is undefined"
    );
  });

  it("hive section is absent when hiveSection is empty string", () => {
    const prompt = assemblePrompt({
      cwd: REPO_ROOT,
      caste: "builder",
      name: "Mason-67",
      task: "Implement feature X",
      platform: "claude",
      agentName: "aether-builder",
      hiveSection: "",
    });

    assert.ok(
      !prompt.includes("## HIVE WISDOM (Cross-Colony Patterns)"),
      "Prompt should not contain a hive section header when hiveSection is empty"
    );
    assert.ok(
      !/\n\n\n\n/.test(prompt),
      "Prompt should not have empty section gaps from empty hiveSection"
    );
  });

  it("hive section appears after skill section when both present", () => {
    const prompt = assemblePrompt({
      cwd: REPO_ROOT,
      caste: "builder",
      name: "Mason-67",
      task: "Implement feature X",
      platform: "claude",
      agentName: "aether-builder",
      skillSection: "## Skills\n- Go testing patterns",
      hiveSection: "## HIVE WISDOM (Cross-Colony Patterns)\n\n(go, 0.90) Prefer table-driven tests",
    });

    const skillIndex = prompt.indexOf("## Skills");
    const hiveIndex = prompt.indexOf("## HIVE WISDOM (Cross-Colony Patterns)");
    assert.ok(skillIndex > 0, "Prompt should contain skill section");
    assert.ok(hiveIndex > 0, "Prompt should contain hive section");
    assert.ok(
      skillIndex < hiveIndex,
      "Hive section should appear after skill section in prompt"
    );
  });

  it("hive section handles non-string hiveSection gracefully", () => {
    const prompt = assemblePrompt({
      cwd: REPO_ROOT,
      caste: "builder",
      name: "Mason-67",
      task: "Implement feature X",
      platform: "claude",
      agentName: "aether-builder",
      hiveSection: 42 as unknown as string,
    });

    assert.ok(
      typeof prompt === "string",
      "assemblePrompt should return a string even with non-string hiveSection"
    );
    assert.ok(
      !prompt.includes("## HIVE WISDOM (Cross-Colony Patterns)"),
      "Prompt should not contain a hive section header for non-string hiveSection"
    );
  });
});

function restoreEnv(key: string, value: string | undefined): void {
  if (value === undefined) {
    delete process.env[key];
    return;
  }
  process.env[key] = value;
}
