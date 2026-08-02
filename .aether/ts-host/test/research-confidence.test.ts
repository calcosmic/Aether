/**
 * research-confidence tests.
 *
 * Pins the four RESEARCH-08 depth tiers (researchLoopPreset,
 * researchLoopOptions) and the ResearchConfidenceEvaluator's evidence-based
 * scoring behaviour (sections, citations, files, self-check blend,
 * reproducibility).
 */

import { describe, it } from "node:test";
import assert from "node:assert/strict";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";

import {
  researchLoopPreset,
  researchLoopOptions,
  ResearchConfidenceEvaluator,
  type ResearchLoopPreset,
} from "../src/research-confidence.js";

// ---------------------------------------------------------------------------
// researchLoopPreset / researchLoopOptions (RESEARCH-08 / D-12)
// ---------------------------------------------------------------------------

describe("researchLoopPreset (RESEARCH-08)", () => {
  it("resolves fast to 80% target / 4 iterations", () => {
    const preset: ResearchLoopPreset = researchLoopPreset("fast");
    assert.deepEqual(preset, { confidenceTarget: 80, maxIterations: 4 });
  });

  it("resolves balanced to 90% target / 6 iterations", () => {
    const preset = researchLoopPreset("balanced");
    assert.deepEqual(preset, { confidenceTarget: 90, maxIterations: 6 });
  });

  it("resolves deep to 95% target / 8 iterations", () => {
    const preset = researchLoopPreset("deep");
    assert.deepEqual(preset, { confidenceTarget: 95, maxIterations: 8 });
  });

  it("resolves exhaustive to 99% target / 12 iterations", () => {
    const preset = researchLoopPreset("exhaustive");
    assert.deepEqual(preset, { confidenceTarget: 99, maxIterations: 12 });
  });

  it("defaults an unknown depth to the balanced pair", () => {
    assert.deepEqual(researchLoopPreset("bogus"), {
      confidenceTarget: 90,
      maxIterations: 6,
    });
  });

  it("defaults an empty depth to the balanced pair", () => {
    assert.deepEqual(researchLoopPreset(""), {
      confidenceTarget: 90,
      maxIterations: 6,
    });
  });

  it("normalises mixed-case and whitespace-padded depth", () => {
    assert.deepEqual(researchLoopPreset("  Fast  "), {
      confidenceTarget: 80,
      maxIterations: 4,
    });
    assert.deepEqual(researchLoopPreset("DEEP"), {
      confidenceTarget: 95,
      maxIterations: 8,
    });
  });
});

describe("researchLoopOptions (RESEARCH-08 / D-12)", () => {
  it("packages the preset pair with the supplied totalBudget", () => {
    const options = researchLoopOptions("deep", 20);
    assert.deepEqual(options, {
      confidenceTarget: 95,
      maxIterations: 8,
      totalBudget: 20,
    });
  });

  it("packages the balanced default with a custom budget", () => {
    const options = researchLoopOptions("unknown-depth", 5);
    assert.deepEqual(options, {
      confidenceTarget: 90,
      maxIterations: 6,
      totalBudget: 5,
    });
  });
});

// ---------------------------------------------------------------------------
// ResearchConfidenceEvaluator (D-11)
// ---------------------------------------------------------------------------

/** Create a temp repo root with two real files for Files-to-Study checks. */
function makeFixtureRepo(): string {
  const repoRoot = fs.mkdtempSync(
    path.join(os.tmpdir(), "research-confidence-")
  );
  fs.writeFileSync(path.join(repoRoot, "confidence-loop.ts"), "// fixture\n");
  fs.writeFileSync(
    path.join(repoRoot, "confidence-evaluator.ts"),
    "// fixture\n"
  );
  return repoRoot;
}

/**
 * A fully-evidenced research artifact: all six sections filled, every
 * Key Patterns / Gotchas bullet cited with a real-looking path or URL, and
 * two Files to Study bullets naming files that exist in the fixture repo.
 */
const FULL_MARKDOWN = `# Phase 999 Research: Test Phase

**Generated:** 2026-01-01T00:00:00Z
**Phase:** 999 - Test Phase
**Research scope:** testing the evidence-based scorer

## Hive Wisdom (Pre-existing Knowledge)
No relevant hive wisdom found for this phase, but a similar pattern applied successfully in an earlier phase.

## Key Patterns
- **Evidence scoring:** confidence-evaluator.ts already implements a base/bonus/penalty/clamp shape worth mirroring (Source: .aether/ts-host/src/confidence-evaluator.ts)
- **Depth binding:** planningLoopPreset already encodes the four target/iteration tiers this phase needs to copy (Source: cmd/codex_plan.go)

## External Context
No external research needed for this phase.

## Gotchas
- **Loop conflation:** three separate confidence loops exist in this codebase and mixing them up would wire the wrong one (Source: .planning/phases/164-research-feeds-planning/164-RESEARCH.md)
- **Reproducibility:** using Date.now or Math.random would break the deterministic scoring requirement (Source: https://example.com/definition-of-done)

## Recommended Approach
Treat this as wiring existing pieces together rather than building new subsystems, mirroring the iteration pattern already proven elsewhere in the host.

## Files to Study
- confidence-loop.ts
- confidence-evaluator.ts
`;

/** Same shape as FULL_MARKDOWN but with citations stripped from every bullet. */
const UNCITED_MARKDOWN = FULL_MARKDOWN.replace(
  /\s*\(Source:[^)]*\)/g,
  ""
);

/** Same shape as FULL_MARKDOWN but Files to Study names nonexistent paths. */
const MISSING_FILES_MARKDOWN = FULL_MARKDOWN.replace(
  "## Files to Study\n- confidence-loop.ts\n- confidence-evaluator.ts\n",
  "## Files to Study\n- does-not-exist-1.ts\n- does-not-exist-2.ts\n"
);

/** A template-placeholder artifact matching renderPhaseResearchBrief's scaffold. */
const TEMPLATE_MARKDOWN = `# Phase 999 Research: Test Phase

## Hive Wisdom (Pre-existing Knowledge)
{relevant prior wisdom, or "No relevant hive wisdom found"}

## Key Patterns
{**pattern:** relevance (Source: path or URL)}

## External Context
{**topic:** finding (Source: URL), or "No external research needed for this phase"}

## Gotchas
{**issue:** prevention (Source: evidence)}

## Recommended Approach
{one synthesis paragraph}

## Files to Study
{bullet list of file paths}
`;

describe("ResearchConfidenceEvaluator (D-11)", () => {
  it("scores an empty research file below 40", () => {
    const evaluator = new ResearchConfidenceEvaluator();
    const result = evaluator.evaluate({ markdown: "", repoRoot: "/tmp" });
    assert.ok(result.score < 40, `expected < 40, got ${result.score}`);
  });

  it("scores a template-placeholder research file below 40", () => {
    const evaluator = new ResearchConfidenceEvaluator();
    const result = evaluator.evaluate({
      markdown: TEMPLATE_MARKDOWN,
      repoRoot: "/tmp",
    });
    assert.ok(result.score < 40, `expected < 40, got ${result.score}`);
  });

  it("scores a fully-evidenced fixture at exactly 100 (reproducible formula)", () => {
    const repoRoot = makeFixtureRepo();
    const evaluator = new ResearchConfidenceEvaluator();
    const result = evaluator.evaluate({ markdown: FULL_MARKDOWN, repoRoot });
    assert.equal(result.score, 100, `expected exact score 100, got ${result.score}`);
    assert.ok(result.score >= 90, `expected >= 90, got ${result.score}`);
  });

  it("scoring the identical input twice returns the identical number", () => {
    const repoRoot = makeFixtureRepo();
    const evaluator = new ResearchConfidenceEvaluator();
    const input = { markdown: FULL_MARKDOWN, repoRoot };
    const first = evaluator.evaluate(input);
    const second = evaluator.evaluate(input);
    assert.equal(first.score, second.score);
    assert.deepEqual(first, second);
  });

  it("replacing verified citations with uncited bullets lowers the score", () => {
    const repoRoot = makeFixtureRepo();
    const evaluator = new ResearchConfidenceEvaluator();
    const cited = evaluator.evaluate({ markdown: FULL_MARKDOWN, repoRoot });
    const uncited = evaluator.evaluate({
      markdown: UNCITED_MARKDOWN,
      repoRoot,
    });
    assert.ok(
      uncited.score < cited.score,
      `expected uncited (${uncited.score}) < cited (${cited.score})`
    );
    assert.equal(uncited.citationsVerified, 0);
  });

  it("listing Files to Study paths that do not exist lowers the score versus paths that do", () => {
    const repoRoot = makeFixtureRepo();
    const evaluator = new ResearchConfidenceEvaluator();
    const withRealFiles = evaluator.evaluate({
      markdown: FULL_MARKDOWN,
      repoRoot,
    });
    const withMissingFiles = evaluator.evaluate({
      markdown: MISSING_FILES_MARKDOWN,
      repoRoot,
    });
    assert.ok(
      withMissingFiles.score < withRealFiles.score,
      `expected missing (${withMissingFiles.score}) < real (${withRealFiles.score})`
    );
    assert.equal(withMissingFiles.filesVerified, 0);
  });

  it("each self-reported gap lowers the score by a fixed amount up to a cap", () => {
    const repoRoot = makeFixtureRepo();
    const evaluator = new ResearchConfidenceEvaluator();
    const noGaps = evaluator.evaluate({
      markdown: FULL_MARKDOWN,
      repoRoot,
      selfAssessedGaps: 0,
    });
    const oneGap = evaluator.evaluate({
      markdown: FULL_MARKDOWN,
      repoRoot,
      selfAssessedGaps: 1,
    });
    const capReachedAtFour = evaluator.evaluate({
      markdown: FULL_MARKDOWN,
      repoRoot,
      selfAssessedGaps: 4,
    });
    const wayOverCap = evaluator.evaluate({
      markdown: FULL_MARKDOWN,
      repoRoot,
      selfAssessedGaps: 10,
    });
    assert.ok(
      oneGap.score < noGaps.score,
      `expected 1 gap (${oneGap.score}) < no gaps (${noGaps.score})`
    );
    assert.equal(
      capReachedAtFour.score,
      wayOverCap.score,
      "penalty should be capped — 4 gaps and 10 gaps should score identically"
    );
  });

  it("returns a diagnosable breakdown (sectionsFilled, citationsVerified, filesVerified, gaps)", () => {
    const repoRoot = makeFixtureRepo();
    const evaluator = new ResearchConfidenceEvaluator();
    const result = evaluator.evaluate({
      markdown: FULL_MARKDOWN,
      repoRoot,
      selfAssessedGaps: 2,
    });
    assert.equal(result.sectionsFilled, 6);
    assert.equal(result.sectionsTotal, 6);
    assert.equal(result.citationsVerified, 4);
    assert.equal(result.citationsTotal, 4);
    assert.equal(result.filesVerified, 2);
    assert.equal(result.filesTotal, 2);
    assert.equal(result.gaps, 2);
    assert.equal(result.source, "research-evidence");
  });
});
