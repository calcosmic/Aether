---
phase: "139"
plan_id: "139-02"
subsystem: "ts-host/confidence"
tags: ["confidence-evaluator", "scoring", "ITER-02", "ITER-04"]
dependency_graph:
  requires: ["139-01 (ConfidenceMetric type, ConfidenceLoop)"]
  provides: ["ConfidenceEvaluator class", "EvaluatedConfidence type", "ConfidenceInput type"]
  affects: ["confidence-loop.ts (consumes toMetric output)"]
tech_stack:
  added: []
  patterns: ["scoring-from-results", "defensive-unknown-cast"]
key_files:
  created:
    - ".aether/ts-host/src/confidence-evaluator.ts"
    - ".aether/ts-host/test/confidence-evaluator.test.ts"
  modified: []
decisions:
  - "Score uses WorkerClaims fields (test_results, files, blockers) accessed via unknown cast since test_results is not on the interface"
  - "toMetric always maps source to worker-claims for ConfidenceMetric compatibility"
metrics:
  duration: "4m"
  completed: "2026-05-18"
  tasks: 2
  files: 2
  tests: 15
---

# Phase 139 Plan 2: Confidence Evaluator Summary

ConfidenceEvaluator derives 0-100 confidence scores from worker claims using test pass rates, file coverage, and blocker counts -- the "scoring eye" that tells ConfidenceLoop whether to keep iterating.

## What Was Built

**confidence-evaluator.ts** -- A `ConfidenceEvaluator` class with two methods:
- `evaluate(input)` -- Takes an array of `WorkerClaims` and optional `finalizerResult`, computes a 0-100 score using: 50 base + up to 30 for test pass rate + 10 for files touched + 10 when all workers completed, minus 10 per blocker (capped at -30). Handles missing data gracefully (empty claims = neutral 50).
- `toMetric(evaluated)` -- Bridges `EvaluatedConfidence` to `ConfidenceMetric` for `ConfidenceLoop` consumption.

**confidence-evaluator.test.ts** -- 15 tests across 6 suites covering basic scoring, test pass rate, blocker impact (including cap and floor), file deduplication, source detection, and toMetric conversion.

## Commits

| Hash | Message |
|------|---------|
| `73f7b11a` | feat(139-02): create ConfidenceEvaluator module |
| `6a350648` | test(139-02): add ConfidenceEvaluator tests (15 passing) |

## Deviations from Plan

None -- plan executed exactly as written.

## Known Stubs

| File | Line | Stub | Reason |
|------|------|------|--------|
| confidence-evaluator.ts | `toMetric` | Source hardcoded to "worker-claims" | ConfidenceMetric.source union does not include "finalizer-gate"; maps conservatively |

## Self-Check: PASSED

- [x] confidence-evaluator.ts exists
- [x] confidence-evaluator.test.ts exists
- [x] Commit 73f7b11a found
- [x] Commit 6a350648 found
- [x] 15 tests pass
- [x] TypeScript compiles (no new errors introduced)
