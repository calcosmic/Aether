/**
 * Milestone audit test for v1.21 requirement coverage.
 *
 * Tests verify:
 * - All 31 v1.21 requirement IDs have at least one test file covering them
 * - Each referenced test file exists on disk
 * - REQUIREMENTS.md contains all 31 requirement entries
 * - No requirement ID is missing from the coverage map
 */

import { existsSync, readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

import { describe, it } from "node:test";
import assert from "node:assert/strict";

const __dirname = dirname(fileURLToPath(import.meta.url));
const testDir = __dirname;
const repoRoot = join(testDir, "..", "..", "..");
const requirementsPath = join(repoRoot, ".planning", "REQUIREMENTS.md");

/**
 * Coverage map: each v1.21 requirement ID mapped to the test file(s)
 * that verify it. This is the single source of truth for milestone coverage.
 */
const REQUIREMENT_COVERAGE_MAP: Array<{
  id: string;
  testFiles: string[];
  description: string;
}> = [
  // HOST-01 through HOST-09
  { id: "HOST-01", testFiles: ["host-integration.test.ts"], description: "Real platform dispatch default" },
  { id: "HOST-02", testFiles: ["host-integration.test.ts", "wave-orchestrator.test.ts"], description: "Parallel waves with dependency ordering" },
  { id: "HOST-03", testFiles: ["host-integration.test.ts"], description: "Plan workers with plan-finalizer" },
  { id: "HOST-04", testFiles: ["host-integration.test.ts"], description: "Continue verification with real dispatch" },
  { id: "HOST-05", testFiles: ["host-integration.test.ts", "oracle-lifecycle.test.ts"], description: "Oracle lifecycle real workers" },
  { id: "HOST-06", testFiles: ["host-integration.test.ts", "ceremony-adapter.test.ts", "ceremony-snapshots.test.ts"], description: "Production ceremony rendering" },
  { id: "HOST-07", testFiles: ["host-integration.test.ts", "host-flags.test.ts"], description: "Dry-run flag on all host commands" },
  { id: "HOST-08", testFiles: ["host-integration.test.ts"], description: "Missing platform clear error" },
  { id: "HOST-09", testFiles: ["host-integration.test.ts", "lifecycle.test.ts"], description: "Simulate flag retains simulation" },

  // SPAWN-01 through SPAWN-06
  { id: "SPAWN-01", testFiles: ["spawn-e2e.test.ts", "spawn-orchestrator.test.ts", "spawn-claims.test.ts", "host-integration.test.ts"], description: "Workers return spawns field" },
  { id: "SPAWN-02", testFiles: ["spawn-orchestrator.test.ts", "spawn-e2e.test.ts"], description: "Maximum spawn depth 2" },
  { id: "SPAWN-03", testFiles: ["spawn-orchestrator.test.ts", "spawn-e2e.test.ts", "host-integration.test.ts"], description: "Child spawns count against budget" },
  { id: "SPAWN-04", testFiles: ["spawn-e2e.test.ts", "spawn-claims.test.ts"], description: "Child spawn results attach to parent handoff" },
  { id: "SPAWN-05", testFiles: ["spawn-e2e.test.ts", "spawn-orchestrator.test.ts"], description: "Go spawn-log records child entries" },
  { id: "SPAWN-06", testFiles: ["spawn-orchestrator.test.ts", "spawn-e2e.test.ts"], description: "Budget exceeded spawns skipped" },

  // ITER-01 through ITER-06
  { id: "ITER-01", testFiles: ["confidence-loop-e2e.test.ts", "confidence-loop.test.ts", "host-integration.test.ts"], description: "Build loops up to 3 iterations" },
  { id: "ITER-02", testFiles: ["confidence-evaluator.test.ts", "confidence-loop.test.ts", "confidence-loop-e2e.test.ts"], description: "Confidence from Go finalizer gates" },
  { id: "ITER-03", testFiles: ["confidence-loop.test.ts", "confidence-loop-e2e.test.ts"], description: "Diminishing returns detection" },
  { id: "ITER-04", testFiles: ["confidence-loop.test.ts", "confidence-loop-e2e.test.ts"], description: "Reusable ConfidenceLoop class" },
  { id: "ITER-05", testFiles: ["confidence-loop.test.ts", "confidence-loop-e2e.test.ts", "host-integration.test.ts"], description: "Cumulative budget across iterations" },
  { id: "ITER-06", testFiles: ["confidence-loop.test.ts", "confidence-loop-e2e.test.ts"], description: "Iteration progress in ceremony" },

  // HIVE-01 through HIVE-07
  { id: "HIVE-01", testFiles: ["hive-injector.test.ts", "host-integration.test.ts"], description: "Hive injector calls hive-read" },
  { id: "HIVE-02", testFiles: ["hive-injector.test.ts", "host-integration.test.ts"], description: "Colony-prime reads hive wisdom" },
  { id: "HIVE-03", testFiles: ["hive-injector.test.ts", "host-integration.test.ts"], description: "Graceful degradation without hive" },
  { id: "HIVE-04", testFiles: ["hive-injector.test.ts", "host-integration.test.ts"], description: "Hive-read failure does not block" },
  { id: "HIVE-05", testFiles: ["hive-injector.test.ts"], description: "Colony B benefits from colony A wisdom" },
  { id: "HIVE-06", testFiles: ["hive-injector.test.ts"], description: "Domain tags include tech stack" },
  { id: "HIVE-07", testFiles: ["hive-injector.test.ts"], description: "Partial domain match discount" },

  // SKILL-01 through SKILL-03
  { id: "SKILL-01", testFiles: ["host-integration.test.ts", "classic-command-parity.test.ts", "cross-platform-parity.test.ts"], description: "Skill section present when dispatch has skill_section" },
  { id: "SKILL-02", testFiles: ["host-integration.test.ts", "cross-platform-parity.test.ts"], description: "Skill section omitted when no skill_section" },
  { id: "SKILL-03", testFiles: ["host-integration.test.ts", "cross-platform-parity.test.ts"], description: "Malformed skill section handled gracefully" },
];

const EXPECTED_TOTAL_REQUIREMENTS = 31;

const REQUIREMENT_ID_PATTERN = /\b(HOST-\d+|SPAWN-\d+|ITER-\d+|HIVE-\d+|SKILL-\d+)\b/g;

describe("milestone audit: v1.21 requirement coverage", () => {
  it("all 31 v1.21 requirements have test coverage", () => {
    const missing: string[] = [];
    for (const entry of REQUIREMENT_COVERAGE_MAP) {
      assert.ok(
        entry.testFiles.length >= 1,
        `${entry.id} should have at least one test file covering it`,
      );
      if (entry.testFiles.length === 0) {
        missing.push(entry.id);
      }
    }

    assert.strictEqual(
      REQUIREMENT_COVERAGE_MAP.length,
      EXPECTED_TOTAL_REQUIREMENTS,
      `Coverage map should have exactly ${EXPECTED_TOTAL_REQUIREMENTS} entries, got ${REQUIREMENT_COVERAGE_MAP.length}`,
    );

    assert.deepStrictEqual(
      missing,
      [],
      "No requirement should be missing coverage",
    );
  });

  it("coverage map test files exist on disk", () => {
    const allTestFiles = new Set<string>();
    for (const entry of REQUIREMENT_COVERAGE_MAP) {
      for (const f of entry.testFiles) {
        allTestFiles.add(f);
      }
    }

    const missing: string[] = [];
    for (const testFile of allTestFiles) {
      const fullPath = join(testDir, testFile);
      if (!existsSync(fullPath)) {
        missing.push(testFile);
      }
    }

    assert.deepStrictEqual(
      missing,
      [],
      `All ${allTestFiles.size} referenced test files should exist on disk`,
    );
  });

  it("REQUIREMENTS.md contains all 31 requirement IDs", () => {
    assert.ok(
      existsSync(requirementsPath),
      `REQUIREMENTS.md should exist at ${requirementsPath}`,
    );

    const content = readFileSync(requirementsPath, "utf-8");
    const matches = content.matchAll(REQUIREMENT_ID_PATTERN);
    const idsFound = new Set<string>();
    for (const match of matches) {
      idsFound.add(match[1]);
    }

    assert.strictEqual(
      idsFound.size,
      EXPECTED_TOTAL_REQUIREMENTS,
      `REQUIREMENTS.md should contain exactly ${EXPECTED_TOTAL_REQUIREMENTS} unique requirement IDs, found ${idsFound.size}`,
    );

    // Verify every ID in the coverage map appears in REQUIREMENTS.md
    const missingFromRequirements: string[] = [];
    for (const entry of REQUIREMENT_COVERAGE_MAP) {
      if (!idsFound.has(entry.id)) {
        missingFromRequirements.push(entry.id);
      }
    }

    assert.deepStrictEqual(
      missingFromRequirements,
      [],
      "All coverage map IDs should appear in REQUIREMENTS.md",
    );
  });

  it("no requirement ID missing from coverage map", () => {
    const content = readFileSync(requirementsPath, "utf-8");
    const matches = content.matchAll(REQUIREMENT_ID_PATTERN);
    const idsFound = new Set<string>();
    for (const match of matches) {
      idsFound.add(match[1]);
    }

    const coverageIds = new Set(REQUIREMENT_COVERAGE_MAP.map((e) => e.id));
    const missingFromCoverage: string[] = [];
    for (const id of idsFound) {
      if (!coverageIds.has(id)) {
        missingFromCoverage.push(id);
      }
    }

    assert.deepStrictEqual(
      missingFromCoverage,
      [],
      `All ${idsFound.size} requirement IDs in REQUIREMENTS.md should appear in the coverage map`,
    );
  });
});
