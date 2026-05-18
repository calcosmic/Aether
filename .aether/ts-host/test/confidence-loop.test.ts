/**
 * ConfidenceLoop tests.
 *
 * Verifies hard cap (ITER-01), diminishing returns (ITER-03), confidence
 * target, cumulative budget (ITER-05), ceremony output (ITER-06), and reset.
 */

import { describe, it } from "node:test";
import assert from "node:assert/strict";

import {
  ConfidenceLoop,
  type ConfidenceLoopOptions,
  type ConfidenceResult,
} from "../src/confidence-loop.js";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Run evaluate repeatedly with fixed confidence and worker counts. */
function runLoop(
  opts: ConfidenceLoopOptions,
  confidences: number[],
  workersPerIteration: number
): ConfidenceResult[] {
  const loop = new ConfidenceLoop(opts);
  const results: ConfidenceResult[] = [];
  for (const confidence of confidences) {
    const result = loop.evaluate(confidence, workersPerIteration);
    results.push(result);
    if (!result.shouldContinue) break;
  }
  return results;
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// 1. Hard cap (ITER-01)
// ---------------------------------------------------------------------------

describe("hard cap (ITER-01)", () => {
  it("stops after 3 iterations max", () => {
    // Always return confidence=50 so no target is met
    const results = runLoop({ maxIterations: 3, confidenceTarget: 100 }, [50, 50, 50, 50], 1);
    assert.equal(results.length, 3, "should stop after 3 iterations");
    assert.equal(results[2]!.stopReason, "max_iterations_met");
    assert.equal(results[2]!.iterationCount, 3);
  });

  it("stops after custom max iterations", () => {
    // Vary confidence enough to avoid diminishing returns (deltas must stay >= 5%)
    const results = runLoop({ maxIterations: 5, confidenceTarget: 100 }, [10, 20, 30, 40, 50, 60], 1);
    assert.equal(results.length, 5, "should stop after 5 iterations");
    assert.equal(results[4]!.stopReason, "max_iterations_met");
    assert.equal(results[4]!.iterationCount, 5);
  });
});

// ---------------------------------------------------------------------------
// 2. Confidence target
// ---------------------------------------------------------------------------

describe("confidence target", () => {
  it("stops when confidence meets target", () => {
    const results = runLoop({ maxIterations: 10, confidenceTarget: 80 }, [50, 70, 85, 90], 1);
    assert.equal(results.length, 3, "should stop when target met at iteration 3");
    assert.equal(results[2]!.stopReason, "confidence_target_met");
    assert.equal(results[2]!.currentConfidence, 85);
  });

  it("stops immediately if first iteration meets target", () => {
    const results = runLoop({ maxIterations: 10, confidenceTarget: 50 }, [60], 1);
    assert.equal(results.length, 1, "should stop after first iteration");
    assert.equal(results[0]!.stopReason, "confidence_target_met");
    assert.equal(results[0]!.iterationCount, 1);
  });
});

// ---------------------------------------------------------------------------
// 3. Diminishing returns (ITER-03)
// ---------------------------------------------------------------------------

describe("diminishing returns (ITER-03)", () => {
  it("stops when delta < 5% for 2 consecutive iterations", () => {
    // confidence 50 -> 70 (delta +20) -> 73 (delta +3) -> 74 (delta +1)
    // Last 2 deltas: +3, +1, both < 5%
    const loop = new ConfidenceLoop({ maxIterations: 10, confidenceTarget: 100, totalBudget: 100 });
    const r1 = loop.evaluate(50, 1);
    assert.equal(r1.shouldContinue, true, "iteration 1 should continue");
    const r2 = loop.evaluate(70, 1);
    assert.equal(r2.shouldContinue, true, "iteration 2 should continue");
    const r3 = loop.evaluate(73, 1);
    assert.equal(r3.shouldContinue, true, "iteration 3: only 1 below threshold so far");
    const r4 = loop.evaluate(74, 1);
    assert.equal(r4.shouldContinue, false, "iteration 4: 2 consecutive below threshold");
    assert.equal(r4.stopReason, "diminishing_returns");
  });

  it("continues when only 1 iteration below threshold", () => {
    // confidence 50 -> 70 (delta +20) -> 73 (delta +3) -> 80 (delta +7)
    // Deltas: +20, +3, +7. Only +3 is below 5%. Not 2 consecutive.
    const loop = new ConfidenceLoop({ maxIterations: 10, confidenceTarget: 100, totalBudget: 100 });
    loop.evaluate(50, 1);
    loop.evaluate(70, 1);
    loop.evaluate(73, 1);
    const r4 = loop.evaluate(80, 1);
    assert.equal(r4.shouldContinue, true, "should continue — only 1 consecutive low delta");
    assert.equal(r4.iterationCount, 4);
  });

  it("respects custom threshold and window", () => {
    // threshold=3, window=2. confidence 50 -> 60 -> 62 -> 63
    // Deltas: +10, +2, +1. Last 2 deltas: +2 and +1, both < 3.
    const loop = new ConfidenceLoop({
      maxIterations: 10,
      confidenceTarget: 100,
      diminishingReturnsThreshold: 3,
      diminishingReturnsWindow: 2,
      totalBudget: 100,
    });
    loop.evaluate(50, 1);
    loop.evaluate(60, 1);
    loop.evaluate(62, 1);
    const r4 = loop.evaluate(63, 1);
    assert.equal(r4.shouldContinue, false, "custom threshold: should stop");
    assert.equal(r4.stopReason, "diminishing_returns");
  });
});

// ---------------------------------------------------------------------------
// 4. Cumulative budget (ITER-05)
// ---------------------------------------------------------------------------

describe("cumulative budget (ITER-05)", () => {
  it("tracks budget across iterations", () => {
    const loop = new ConfidenceLoop({ maxIterations: 10, confidenceTarget: 100, totalBudget: 10 });
    const r1 = loop.evaluate(50, 4);
    assert.equal(r1.budgetRemaining, 6, "after 4 of 10, 6 remaining");
    const r2 = loop.evaluate(55, 4);
    assert.equal(r2.budgetRemaining, 2, "after 8 of 10, 2 remaining");
    assert.equal(r2.shouldContinue, true, "still has budget");
  });

  it("stops when budget exhausted", () => {
    const loop = new ConfidenceLoop({ maxIterations: 10, confidenceTarget: 100, totalBudget: 5 });
    const r1 = loop.evaluate(50, 3);
    assert.equal(r1.budgetRemaining, 2, "after 3 of 5, 2 remaining");
    assert.equal(r1.shouldContinue, true);
    const r2 = loop.evaluate(55, 3);
    assert.equal(r2.budgetRemaining, -1, "over budget");
    assert.equal(r2.shouldContinue, false, "should stop — budget exhausted");
    assert.equal(r2.stopReason, "budget_exhausted");
  });

  it("budget of 0 means no budget tracking", () => {
    const loop = new ConfidenceLoop({ maxIterations: 5, confidenceTarget: 100, totalBudget: 0 });
    // Use many workers per iteration — with totalBudget=0, budget is unlimited
    const r1 = loop.evaluate(50, 999);
    assert.equal(r1.budgetRemaining, 0, "budget tracking disabled");
    assert.equal(r1.shouldContinue, true, "should continue regardless of workers used");
    const r2 = loop.evaluate(55, 999);
    assert.equal(r2.budgetRemaining, 0, "still 0 — no tracking");
    assert.equal(r2.shouldContinue, true, "budget does not stop the loop");
  });
});

// ---------------------------------------------------------------------------
// 5. Ceremony output (ITER-06)
// ---------------------------------------------------------------------------

describe("ceremony output (ITER-06)", () => {
  it("renders iteration number, confidence, delta, budget", () => {
    const loop = new ConfidenceLoop({ maxIterations: 5, confidenceTarget: 100, totalBudget: 20 });
    const result = loop.evaluate(75, 5);
    const ceremony = loop.renderIterationCeremony(result);
    assert.ok(ceremony.includes("Iteration 1"), `should contain iteration number: ${ceremony}`);
    assert.ok(ceremony.includes("75%"), `should contain confidence: ${ceremony}`);
    assert.ok(ceremony.includes("15 workers remaining"), `should contain budget: ${ceremony}`);
  });

  it("renders stop reason when stopped", () => {
    const loop = new ConfidenceLoop({ maxIterations: 2, confidenceTarget: 100, totalBudget: 20 });
    loop.evaluate(50, 1);
    const result = loop.evaluate(55, 1);
    const ceremony = loop.renderIterationCeremony(result);
    assert.ok(ceremony.includes("max_iterations_met"), `should contain stop reason: ${ceremony}`);
  });

  it("renders delta with sign", () => {
    const loop = new ConfidenceLoop({ maxIterations: 5, confidenceTarget: 100, totalBudget: 20 });
    loop.evaluate(50, 1);
    const result = loop.evaluate(70, 1);
    const ceremony = loop.renderIterationCeremony(result);
    assert.ok(ceremony.includes("+20%"), `positive delta should have +: ${ceremony}`);
  });
});

// ---------------------------------------------------------------------------
// 6. Reset
// ---------------------------------------------------------------------------

describe("reset", () => {
  it("reset clears iteration state", () => {
    const loop = new ConfidenceLoop({ maxIterations: 5, confidenceTarget: 100, totalBudget: 20 });

    // Run 2 iterations
    loop.evaluate(50, 3);
    loop.evaluate(60, 2);
    const state1 = loop.getState();
    assert.equal(state1.iterationCount, 2);
    assert.deepEqual(state1.confidenceHistory, [50, 60]);
    assert.equal(state1.budgetConsumed, 5);

    // Reset
    loop.reset();
    const state2 = loop.getState();
    assert.equal(state2.iterationCount, 0, "iterationCount should reset");
    assert.deepEqual(state2.confidenceHistory, [], "history should clear");
    assert.equal(state2.budgetConsumed, 0, "budget should reset");

    // Run again — should start fresh
    const result = loop.evaluate(40, 1);
    assert.equal(result.iterationCount, 1, "starts fresh after reset");
    assert.equal(result.delta, 0, "delta is 0 for first iteration after reset");
  });
});

// ---------------------------------------------------------------------------
// 7. getState
// ---------------------------------------------------------------------------

describe("getState", () => {
  it("returns correct state snapshot", () => {
    const loop = new ConfidenceLoop({ maxIterations: 5, confidenceTarget: 100, totalBudget: 10 });
    loop.evaluate(30, 2);
    loop.evaluate(45, 3);

    const state = loop.getState();
    assert.equal(state.iterationCount, 2);
    assert.deepEqual(state.confidenceHistory, [30, 45]);
    assert.equal(state.budgetConsumed, 5);
    assert.equal(state.budgetRemaining, 5);
  });

  it("returns defensive copy of confidenceHistory", () => {
    const loop = new ConfidenceLoop({ maxIterations: 5, confidenceTarget: 100, totalBudget: 10 });
    loop.evaluate(30, 1);

    const state = loop.getState();
    state.confidenceHistory.push(999);
    const state2 = loop.getState();
    assert.equal(state2.confidenceHistory.length, 1, "mutation of snapshot should not affect loop");
  });
});
