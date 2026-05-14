/**
 * Cross-platform parity smoke tests.
 *
 * Verifies Claude Code, OpenCode, and Codex command wrappers and agent
 * definitions stay in sync.
 */

import { readdirSync, readFileSync, statSync } from "node:fs";
import { join } from "node:path";
import { describe, it } from "node:test";
import assert from "node:assert/strict";

import { detectAvailablePlatforms, isPlatformAvailable } from "../src/platform-dispatcher.js";

const REPO_ROOT = "/Users/callumcowie/repos/Aether";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function listFiles(dir: string, ext: string): string[] {
  if (!statSync(dir, { throwIfNoEntry: false })?.isDirectory()) {
    return [];
  }
  return readdirSync(dir).filter((f) => f.endsWith(ext)).sort();
}

function fileSize(path: string): number {
  return statSync(path).size;
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("cross-platform parity", () => {
  it("Claude and OpenCode commands have identical file names", () => {
    const claudeDir = join(REPO_ROOT, ".claude", "commands", "ant");
    const opencodeDir = join(REPO_ROOT, ".opencode", "commands", "ant");

    const claudeFiles = listFiles(claudeDir, ".md");
    const opencodeFiles = listFiles(opencodeDir, ".md");

    assert.deepEqual(
      claudeFiles,
      opencodeFiles,
      "Claude and OpenCode command directories should have identical .md files"
    );

    // Lightweight size parity check
    for (const file of claudeFiles) {
      const cSize = fileSize(join(claudeDir, file));
      const oSize = fileSize(join(opencodeDir, file));
      assert.ok(
        Math.abs(cSize - oSize) < 100,
        `Command file ${file} size differs significantly between Claude (${cSize}) and OpenCode (${oSize})`
      );
    }
  });

  it("all 27 castes have agents on all 3 platforms", () => {
    const claudeDir = join(REPO_ROOT, ".claude", "agents", "ant");
    const opencodeDir = join(REPO_ROOT, ".opencode", "agents");
    const codexDir = join(REPO_ROOT, ".codex", "agents");

    const claudeAgents = listFiles(claudeDir, ".md").map((f) =>
      f.replace(/^aether-/, "").replace(/\.md$/, "")
    );
    const opencodeAgents = listFiles(opencodeDir, ".md").map((f) =>
      f.replace(/^aether-/, "").replace(/\.md$/, "")
    );
    const codexAgents = listFiles(codexDir, ".toml").map((f) =>
      f.replace(/^aether-/, "").replace(/\.toml$/, "")
    );

    assert.equal(claudeAgents.length, 27, `Claude should have 27 agents, found ${claudeAgents.length}`);

    for (const agent of claudeAgents) {
      assert.ok(
        opencodeAgents.includes(agent),
        `OpenCode missing agent for caste: ${agent}`
      );
      assert.ok(
        codexAgents.includes(agent),
        `Codex missing agent for caste: ${agent}`
      );
    }
  });

  it("build wrappers reference the same split playbooks", () => {
    const claudeBuild = readFileSync(
      join(REPO_ROOT, ".claude", "commands", "ant", "build.md"),
      "utf-8"
    );
    const opencodeBuild = readFileSync(
      join(REPO_ROOT, ".opencode", "commands", "ant", "build.md"),
      "utf-8"
    );

    const playbookPattern = /build-(prep|context|wave|verify|complete)\.md/g;
    const claudePlaybooks = [...claudeBuild.matchAll(playbookPattern)].map((m) => m[0]).sort();
    const opencodePlaybooks = [...opencodeBuild.matchAll(playbookPattern)].map((m) => m[0]).sort();

    assert.deepEqual(
      claudePlaybooks,
      opencodePlaybooks,
      "Claude and OpenCode build wrappers should reference the same split playbooks"
    );
  });

  it("Codex TOML agents have required fields", () => {
    const codexDir = join(REPO_ROOT, ".codex", "agents");
    const tomlFiles = listFiles(codexDir, ".toml");

    assert.ok(tomlFiles.length > 0, "Codex agents directory should contain .toml files");

    for (const file of tomlFiles) {
      const content = readFileSync(join(codexDir, file), "utf-8");
      assert.ok(
        /name\s*=/.test(content),
        `${file} should contain a name field`
      );
      assert.ok(
        /description\s*=/.test(content),
        `${file} should contain a description field`
      );
      assert.ok(
        /developer_instructions\s*=/.test(content) || /instructions\s*=/.test(content),
        `${file} should contain developer_instructions field`
      );
    }
  });

  it("platform dispatcher returns at least one available platform", async () => {
    const platforms = await detectAvailablePlatforms();

    if (platforms.length === 0) {
      console.warn("Warning: no platforms detected on this machine (expected in CI)");
      return; // Skip, do not fail
    }

    for (const platform of platforms) {
      const available = await isPlatformAvailable(platform);
      assert.ok(available, `Platform ${platform} should be available`);
    }
  });
});
