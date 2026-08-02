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

import {
  researchLoopPreset,
  researchLoopOptions,
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
