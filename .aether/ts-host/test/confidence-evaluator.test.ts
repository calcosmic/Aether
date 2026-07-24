/**
 * ConfidenceEvaluator tests.
 *
 * Verifies scoring algorithm, test pass rate, file coverage, blocker impact,
 * source detection, and toMetric conversion.
 */

import { describe, it } from "node:test";
import assert from "node:assert/strict";

import {
  ConfidenceEvaluator,
  type ConfidenceInput,
  type EvaluatedConfidence,
} from "../src/confidence-evaluator.js";
import type { WorkerClaims } from "../src/claims-parser.js";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const evaluator = new ConfidenceEvaluator();

/** Build a claim with test_results (which is accessed via unknown cast). */
function claimWithTests(
  status: string,
  passed: number,
  total: number,
  extra?: Partial<WorkerClaims> & { test_results?: { passed: number; failed: number; total: number } }
): WorkerClaims & { test_results?: { passed: number; failed: number; total: number } } {
  return {
    status,
    ...(extra ?? {}),
    // test_results is not on the WorkerClaims interface but is accessed
    // defensively via unknown cast in the evaluator.
  } as WorkerClaims & { test_results?: { passed: number; failed: number; total: number } };
}

// ---------------------------------------------------------------------------
// 1. Basic scoring
// ---------------------------------------------------------------------------

describe("basic scoring", () => {
  it("returns 50 for empty claims (neutral baseline)", () => {
    const result = evaluator.evaluate({ workerClaims: [] });
    assert.equal(result.score, 50, "empty claims should give neutral baseline");
    assert.equal(result.test_pass_rate, 0);
    assert.equal(result.files_covered, 0);
    assert.deepEqual(result.blockers, []);
  });

  it("returns high score for all-completed workers with tests", () => {
    const claims: WorkerClaims[] = [
      { status: "completed", files_created: ["a.ts"], files_modified: ["b.ts"] },
      { status: "completed", files_created: ["c.ts"] },
      { status: "completed", files_modified: ["d.ts"] },
    ];
    // Attach test_results via extended object
    const extended = claims.map((c, i) => ({
      ...c,
      test_results: { passed: 4, failed: 0, total: 4 },
    })) as WorkerClaims[];

    const result = evaluator.evaluate({ workerClaims: extended });
    // 50 base + 30 (tests 12/12=1.0) + 10 (files) + 10 (all completed) = 100
    assert.ok(
      result.score >= 90,
      `score should be >= 90, got ${result.score}`
    );
  });

  it("returns low score for all-failed workers", () => {
    const claims: WorkerClaims[] = [
      { status: "failed" },
      { status: "failed" },
    ];
    const result = evaluator.evaluate({ workerClaims: claims });
    // 50 base, no bonuses, no all-completed bonus = 50
    assert.ok(
      result.score < 60,
      `score should be below 60 for failed workers with no data, got ${result.score}`
    );
  });
});

// ---------------------------------------------------------------------------
// 2. Test pass rate scoring
// ---------------------------------------------------------------------------

describe("test pass rate scoring", () => {
  it("partial test pass reduces score", () => {
    const claims = [
      {
        status: "completed",
        test_results: { passed: 5, failed: 5, total: 10 },
      },
    ] as unknown as WorkerClaims[];
    const result = evaluator.evaluate({ workerClaims: claims });
    // 50 base + 30 * 0.5 = 65, + 10 (completed) = 75, round
    assert.ok(
      result.score >= 60 && result.score <= 80,
      `score should be 60-80 for 50% pass rate, got ${result.score}`
    );
  });

  it("no test data gives 0 pass rate", () => {
    const claims: WorkerClaims[] = [
      { status: "completed" },
    ];
    const result = evaluator.evaluate({ workerClaims: claims });
    assert.equal(result.test_pass_rate, 0, "no test data should give 0 pass rate");
  });

  it("aggregates tests across multiple workers", () => {
    const claims = [
      { status: "completed", test_results: { passed: 5, failed: 0, total: 5 } },
      { status: "completed", test_results: { passed: 5, failed: 0, total: 5 } },
    ] as unknown as WorkerClaims[];
    const result = evaluator.evaluate({ workerClaims: claims });
    // 10/10 = 1.0 pass rate
    assert.equal(result.test_pass_rate, 1, "should aggregate to 10 passed of 10 total");
  });
});

// ---------------------------------------------------------------------------
// 3. Blocker impact
// ---------------------------------------------------------------------------

describe("blocker impact", () => {
  it("each blocker reduces score by 10", () => {
    const base: WorkerClaims[] = [{ status: "completed" }];

    const r0 = evaluator.evaluate({ workerClaims: base });
    const r1 = evaluator.evaluate({
      workerClaims: [{ status: "completed", blockers: ["b1"] }],
    });
    const r2 = evaluator.evaluate({
      workerClaims: [{ status: "completed", blockers: ["b1", "b2"] }],
    });
    const r3 = evaluator.evaluate({
      workerClaims: [{ status: "completed", blockers: ["b1", "b2", "b3"] }],
    });
    // Penalty capped at -30, so 4 blockers should also be -30
    const r4 = evaluator.evaluate({
      workerClaims: [{ status: "completed", blockers: ["b1", "b2", "b3", "b4", "b5"] }],
    });

    assert.equal(r0.score - r1.score, 10, "1 blocker should reduce by 10");
    assert.equal(r0.score - r2.score, 20, "2 blockers should reduce by 20");
    assert.equal(r0.score - r3.score, 30, "3 blockers should reduce by 30");
    assert.equal(r0.score - r4.score, 30, "5 blockers should be capped at -30");
  });

  it("blockers from multiple workers are aggregated", () => {
    const result = evaluator.evaluate({
      workerClaims: [
        { status: "completed", blockers: ["blocker-a"] },
        { status: "completed", blockers: ["blocker-b"] },
      ],
    });
    assert.equal(result.blockers.length, 2, "should aggregate blockers from all workers");
    assert.ok(result.blockers.includes("blocker-a"));
    assert.ok(result.blockers.includes("blocker-b"));
  });

  it("score never goes below 0", () => {
    const result = evaluator.evaluate({
      workerClaims: [
        {
          status: "failed",
          blockers: ["b1", "b2", "b3", "b4", "b5"],
        },
      ],
    });
    assert.ok(result.score >= 0, `score should never go below 0, got ${result.score}`);
  });
});

// ---------------------------------------------------------------------------
// 4. Files covered
// ---------------------------------------------------------------------------

describe("files covered", () => {
  it("counts files from claims", () => {
    const result = evaluator.evaluate({
      workerClaims: [
        {
          status: "completed",
          files_created: ["a.ts", "b.ts"],
          files_modified: ["c.ts"],
        },
      ],
    });
    assert.ok(
      result.files_covered >= 3,
      `should count at least 3 files, got ${result.files_covered}`
    );
  });

  it("deduplicates files across workers", () => {
    const result = evaluator.evaluate({
      workerClaims: [
        { status: "completed", files_created: ["a.ts"] },
        { status: "completed", files_modified: ["a.ts"] },
      ],
    });
    assert.equal(result.files_covered, 1, "same file across workers should count once");
  });
});

// ---------------------------------------------------------------------------
// 5. Source detection
// ---------------------------------------------------------------------------

describe("source detection", () => {
  it("defaults to worker-claims source", () => {
    const result = evaluator.evaluate({ workerClaims: [] });
    assert.equal(result.source, "worker-claims");
  });

  it("uses finalizer-gate when gate_results present", () => {
    const result = evaluator.evaluate({
      workerClaims: [],
      finalizerResult: { gate_results: { passed: true } },
    });
    assert.equal(result.source, "finalizer-gate");
  });
});

// ---------------------------------------------------------------------------
// 6. toMetric conversion
// ---------------------------------------------------------------------------

describe("toMetric conversion", () => {
  it("converts EvaluatedConfidence to ConfidenceMetric", () => {
    const evaluated: EvaluatedConfidence = {
      score: 75,
      test_pass_rate: 0.8,
      files_covered: 5,
      blockers: ["slow test"],
      source: "worker-claims",
    };
    const metric = evaluator.toMetric(evaluated);
    assert.equal(metric.score, 75);
    assert.equal(metric.test_pass_rate, 0.8);
    assert.equal(metric.files_covered, 5);
    assert.deepEqual(metric.blockers, ["slow test"]);
    assert.equal(metric.source, "worker-claims");
  });

  it("toMetric maps finalizer-gate source to worker-claims", () => {
    const evaluated: EvaluatedConfidence = {
      score: 90,
      test_pass_rate: 1,
      files_covered: 3,
      blockers: [],
      source: "finalizer-gate",
    };
    const metric = evaluator.toMetric(evaluated);
    // toMetric always maps to "worker-claims" for ConfidenceMetric compatibility
    assert.equal(metric.source, "worker-claims");
  });
});
