/**
 * Playbook loader unit tests.
 *
 * Verifies candidate resolution, playbook loading, workflow mapping,
 * and document context rendering with budget truncation.
 */

import { describe, it, beforeEach, afterEach } from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, writeFileSync, mkdirSync, rmSync } from "node:fs";
import { join } from "node:path";
import os from "node:os";

import {
  resolvePlaybookCandidates,
  loadPlaybook,
  loadPlaybooksForWorkflow,
  renderPlaybookContext,
  Playbook,
} from "../src/playbook-loader.js";

// ---------------------------------------------------------------------------
// Test fixtures
// ---------------------------------------------------------------------------

let tmpDir: string;

beforeEach(() => {
  tmpDir = mkdtempSync(join(os.tmpdir(), "aether-playbook-test-"));
});

afterEach(() => {
  rmSync(tmpDir, { recursive: true, force: true });
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("playbook-loader", () => {
  // -- resolvePlaybookCandidates --

  describe("resolvePlaybookCandidates", () => {
    it("returns repo-local, hub system, hub root, and bare paths for relative names", () => {
      const root = "/project";
      const candidates = resolvePlaybookCandidates(root, "build-prep.md");
      // Should include: repo-local, repo-join, hub system, hub root, bare
      assert.ok(candidates.length >= 3, `expected >=3 candidates, got ${candidates.length}`);
      // repo-local path should be present
      assert.ok(candidates.some((c: string) => c.includes(".aether/docs/command-playbooks/build-prep.md")));
      // hub system path should be present
      assert.ok(candidates.some((c: string) => c.includes(".aether/system/docs/command-playbooks/build-prep.md")));
      // bare path should be last
      assert.equal(candidates[candidates.length - 1], "build-prep.md");
    });

    it("returns absolute path first when input is absolute", () => {
      const candidates = resolvePlaybookCandidates("/project", "/absolute/path/playbook.md");
      assert.equal(candidates[0], "/absolute/path/playbook.md");
    });

    it("returns empty array for empty string input", () => {
      assert.deepEqual(resolvePlaybookCandidates("/project", ""), []);
    });

    it("returns empty array for whitespace-only input", () => {
      assert.deepEqual(resolvePlaybookCandidates("/project", "   "), []);
    });

    it("deduplicates candidates", () => {
      // If root equals homedir + something that produces duplicate paths
      const candidates = resolvePlaybookCandidates("/project", "build-prep.md");
      const unique = new Set(candidates);
      assert.equal(candidates.length, unique.size, "candidates should be deduplicated");
    });
  });

  // -- loadPlaybook --

  describe("loadPlaybook", () => {
    it("returns null when no candidates exist on disk", () => {
      const result = loadPlaybook("/nonexistent", "ghost-playbook.md");
      assert.equal(result, null);
    });

    it("returns Playbook with name, path, content when first candidate exists", () => {
      const playbookDir = join(tmpDir, ".aether/docs/command-playbooks");
      mkdirSync(playbookDir, { recursive: true });
      const playbookPath = join(playbookDir, "test-playbook.md");
      writeFileSync(playbookPath, "# Test Playbook\n\nSome content here.");

      const result = loadPlaybook(tmpDir, "test-playbook.md");
      assert.ok(result !== null);
      assert.equal(result!.name, "test-playbook.md");
      assert.equal(result!.path, playbookPath);
      assert.ok(result!.content.includes("# Test Playbook"));
    });

    it("skips non-existent candidates and loads from later candidate", () => {
      // Only create the bare-path file
      const barePath = join(tmpDir, "fallback-playbook.md");
      writeFileSync(barePath, "# Fallback\n\nFallback content.");

      // Use a root that doesn't have the playbook in .aether/ dir
      const result = loadPlaybook(tmpDir, "fallback-playbook.md");
      assert.ok(result !== null);
      assert.equal(result!.name, "fallback-playbook.md");
      assert.ok(result!.content.includes("# Fallback"));
    });
  });

  // -- loadPlaybooksForWorkflow --

  describe("loadPlaybooksForWorkflow", () => {
    it('returns build playbooks matching the 5-file list (only ones that exist on disk)', () => {
      // Create some build playbooks in the test fixture
      const playbookDir = join(tmpDir, ".aether/docs/command-playbooks");
      mkdirSync(playbookDir, { recursive: true });
      writeFileSync(join(playbookDir, "build-prep.md"), "# Build Prep");
      writeFileSync(join(playbookDir, "build-wave.md"), "# Build Wave");
      // Don't create the other 3 -- only 2 should be returned

      const result = loadPlaybooksForWorkflow(tmpDir, "build");
      assert.equal(result.length, 2);
      assert.ok(result.some((p: Playbook) => p.name === "build-prep.md"));
      assert.ok(result.some((p: Playbook) => p.name === "build-wave.md"));
    });

    it('returns plan playbooks when plan-prep and plan-dispatch exist', () => {
      const playbookDir = join(tmpDir, ".aether/docs/command-playbooks");
      mkdirSync(playbookDir, { recursive: true });
      writeFileSync(join(playbookDir, "plan-prep.md"), "# Plan Prep");
      writeFileSync(join(playbookDir, "plan-dispatch.md"), "# Plan Dispatch");

      const result = loadPlaybooksForWorkflow(tmpDir, "plan");
      assert.equal(result.length, 2);
      assert.ok(result.some((p: Playbook) => p.name === "plan-prep.md"));
      assert.ok(result.some((p: Playbook) => p.name === "plan-dispatch.md"));
    });

    it('returns continue playbooks matching the 4-file list (only ones that exist)', () => {
      const playbookDir = join(tmpDir, ".aether/docs/command-playbooks");
      mkdirSync(playbookDir, { recursive: true });
      writeFileSync(join(playbookDir, "continue-verify.md"), "# Continue Verify");
      writeFileSync(join(playbookDir, "continue-advance.md"), "# Continue Advance");
      writeFileSync(join(playbookDir, "continue-finalize.md"), "# Continue Finalize");
      // Don't create continue-gates -- 3 should be returned

      const result = loadPlaybooksForWorkflow(tmpDir, "continue");
      assert.equal(result.length, 3);
      assert.ok(result.some((p: Playbook) => p.name === "continue-verify.md"));
      assert.ok(result.some((p: Playbook) => p.name === "continue-advance.md"));
      assert.ok(result.some((p: Playbook) => p.name === "continue-finalize.md"));
    });

    it("returns empty array for unknown workflow", () => {
      const result = loadPlaybooksForWorkflow(tmpDir, "unknown" as "build");
      assert.deepEqual(result, []);
    });
  });

  // -- renderPlaybookContext --

  describe("renderPlaybookContext", () => {
    it("returns empty string for empty playbooks array", () => {
      assert.equal(renderPlaybookContext([]), "");
    });

    it('returns "## Relevant Playbooks" header with playbook content', () => {
      const playbooks: Playbook[] = [
        { name: "test.md", path: "/tmp/test.md", content: "# Hello\n\nWorld." },
      ];
      const result = renderPlaybookContext(playbooks);
      assert.ok(result.startsWith("## Relevant Playbooks"));
      assert.ok(result.includes("# Hello"));
      assert.ok(result.includes("World."));
    });

    it("truncates individual files to maxPerFile chars with marker", () => {
      const longContent = "A".repeat(5000);
      const playbooks: Playbook[] = [
        { name: "long.md", path: "/tmp/long.md", content: longContent },
      ];
      const result = renderPlaybookContext(playbooks, 7000, 1000);
      assert.ok(result.includes("[playbook truncated]"), "should include truncation marker");
      // The snippet should be <= 1000 chars plus the header and marker
      const afterHeader = result.split("### long.md\n\n")[1] ?? "";
      assert.ok(afterHeader.length <= 1200, `snippet too long: ${afterHeader.length}`);
    });

    it("stops adding playbooks when maxBudget chars are exhausted", () => {
      const bigPlaybooks: Playbook[] = [
        { name: "first.md", path: "/tmp/first.md", content: "A".repeat(4000) },
        { name: "second.md", path: "/tmp/second.md", content: "B".repeat(4000) },
      ];
      const result = renderPlaybookContext(bigPlaybooks, 5000, 4000);
      assert.ok(result.includes("### first.md"), "first playbook should be present");
      assert.ok(!result.includes("### second.md"), "second playbook should be omitted (budget exceeded)");
    });

    it("uses default budget constants (7000 total, 2800 per file)", () => {
      const playbooks: Playbook[] = [
        { name: "default-test.md", path: "/tmp/default.md", content: "X".repeat(3000) },
      ];
      const result = renderPlaybookContext(playbooks);
      // With default maxPerFile=2800, 3000 chars should be truncated
      assert.ok(result.includes("[playbook truncated]"), "should truncate with default 2800 budget");
    });

    it("includes playbook name as section header", () => {
      const playbooks: Playbook[] = [
        { name: "build-wave.md", path: "/tmp/build-wave.md", content: "# Build Wave\n\nSteps here." },
      ];
      const result = renderPlaybookContext(playbooks);
      assert.ok(result.includes("### build-wave.md"));
    });
  });
});
