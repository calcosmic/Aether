/**
 * Confidence evaluator for the TypeScript orchestration host.
 *
 * Derives a 0-100 confidence score from worker claims (test pass rates,
 * files covered, blockers) and optional finalizer gate results. The score
 * comes from actual results, not worker self-reports (ITER-02).
 *
 * Bridges to ConfidenceMetric for use by ConfidenceLoop.
 */
import { isSuccessfulWorkerStatus } from "./worker-status.js";
// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------
/** Base score before adjustments (neutral). */
const BASE_SCORE = 50;
/** Maximum bonus for test pass rate (0-30, proportional to pass rate). */
const TEST_BONUS_MAX = 30;
/** Flat bonus when any files were touched. */
const FILE_BONUS = 10;
/** Bonus when all workers completed successfully. */
const ALL_COMPLETED_BONUS = 10;
/** Penalty per blocker. */
const BLOCKER_PENALTY = 10;
/** Maximum total blocker penalty. */
const BLOCKER_PENALTY_CAP = 30;
/** Maximum score. */
const SCORE_MAX = 100;
/** Minimum score. */
const SCORE_MIN = 0;
// ---------------------------------------------------------------------------
// ConfidenceEvaluator
// ---------------------------------------------------------------------------
/**
 * Converts raw worker results into a confidence score.
 *
 * Scoring algorithm:
 * - Base: 50 (neutral)
 * - Test pass rate: +30 * (passed / total) when test data exists
 * - Files touched: +10 when any files_created or files_modified present
 * - All workers completed: +10
 * - Per blocker: -10 (max -30)
 * - Final score clamped to [0, 100]
 */
export class ConfidenceEvaluator {
    /**
     * Evaluate confidence from worker claims and optional finalizer output.
     *
     * @param input - Worker claims and optional finalizer gate result
     * @returns Evaluated confidence with score breakdown
     */
    evaluate(input) {
        const claims = input.workerClaims;
        // Aggregate test results across all workers
        let totalPassed = 0;
        let totalTests = 0;
        for (const claim of claims) {
            // WorkerClaims doesn't have test_results directly; we infer from
            // tests_written and status. But the plan interface says WorkerClaims
            // has test_results. Check for it defensively on the raw claims.
            const raw = claim;
            if (typeof raw.test_results === "object" &&
                raw.test_results !== null) {
                const tr = raw.test_results;
                if (typeof tr.passed === "number")
                    totalPassed += tr.passed;
                if (typeof tr.total === "number")
                    totalTests += tr.total;
            }
        }
        // Aggregate unique files
        const fileSet = new Set();
        for (const claim of claims) {
            if (claim.files_created) {
                for (const f of claim.files_created)
                    fileSet.add(f);
            }
            if (claim.files_modified) {
                for (const f of claim.files_modified)
                    fileSet.add(f);
            }
        }
        const filesCovered = fileSet.size;
        // Collect all blockers
        const blockers = [];
        for (const claim of claims) {
            if (claim.blockers) {
                for (const b of claim.blockers)
                    blockers.push(b);
            }
        }
        // Count completed workers
        const totalWorkers = claims.length;
        // completed_no_change counts as a completed worker (ruling D6);
        // scoring it as incomplete penalised the honest answer.
        const completedWorkers = claims.filter((c) => isSuccessfulWorkerStatus(c.status)).length;
        // Compute score
        let score = BASE_SCORE;
        // Test pass rate bonus
        if (totalTests > 0) {
            score += TEST_BONUS_MAX * (totalPassed / totalTests);
        }
        // Files bonus
        if (filesCovered > 0) {
            score += FILE_BONUS;
        }
        // All-completed bonus
        if (totalWorkers > 0 && completedWorkers === totalWorkers) {
            score += ALL_COMPLETED_BONUS;
        }
        // Blocker penalty (capped)
        const blockerPenalty = Math.min(blockers.length * BLOCKER_PENALTY, BLOCKER_PENALTY_CAP);
        score -= blockerPenalty;
        // Clamp
        score = Math.max(SCORE_MIN, Math.min(SCORE_MAX, Math.round(score)));
        // Test pass rate (0-1)
        const testPassRate = totalTests > 0 ? totalPassed / totalTests : 0;
        // Source detection
        const source = input.finalizerResult !== undefined &&
            typeof input.finalizerResult === "object" &&
            "gate_results" in input.finalizerResult
            ? "finalizer-gate"
            : "worker-claims";
        return {
            score,
            test_pass_rate: testPassRate,
            files_covered: filesCovered,
            blockers,
            source,
        };
    }
    /**
     * Convert an EvaluatedConfidence to a ConfidenceMetric for use by
     * ConfidenceLoop.
     *
     * @param evaluated - The evaluated confidence result
     * @returns ConfidenceMetric suitable for the confidence loop
     */
    toMetric(evaluated) {
        return {
            score: evaluated.score,
            test_pass_rate: evaluated.test_pass_rate,
            files_covered: evaluated.files_covered,
            blockers: evaluated.blockers,
            source: "worker-claims",
        };
    }
}
