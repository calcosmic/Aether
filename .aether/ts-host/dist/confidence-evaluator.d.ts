/**
 * Confidence evaluator for the TypeScript orchestration host.
 *
 * Derives a 0-100 confidence score from worker claims (test pass rates,
 * files covered, blockers) and optional finalizer gate results. The score
 * comes from actual results, not worker self-reports (ITER-02).
 *
 * Bridges to ConfidenceMetric for use by ConfidenceLoop.
 */
import type { WorkerClaims } from "./claims-parser.js";
import type { ConfidenceMetric } from "./types.js";
/** Input to the confidence evaluator. */
export interface ConfidenceInput {
    /** Worker claims extracted from build/continue results. */
    workerClaims: WorkerClaims[];
    /** Reserved for future Go gate results from the finalizer. */
    finalizerResult?: Record<string, unknown>;
}
/** Evaluated confidence with detailed breakdown. */
export interface EvaluatedConfidence {
    /** Overall confidence score (0-100). */
    score: number;
    /** Fraction of tests that passed (0-1). 0 when no test data. */
    test_pass_rate: number;
    /** Number of unique files touched (created + modified, deduplicated). */
    files_covered: number;
    /** Aggregated blockers from all workers. */
    blockers: string[];
    /** Where this evaluation came from. */
    source: "worker-claims" | "finalizer-gate";
}
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
export declare class ConfidenceEvaluator {
    /**
     * Evaluate confidence from worker claims and optional finalizer output.
     *
     * @param input - Worker claims and optional finalizer gate result
     * @returns Evaluated confidence with score breakdown
     */
    evaluate(input: ConfidenceInput): EvaluatedConfidence;
    /**
     * Convert an EvaluatedConfidence to a ConfidenceMetric for use by
     * ConfidenceLoop.
     *
     * @param evaluated - The evaluated confidence result
     * @returns ConfidenceMetric suitable for the confidence loop
     */
    toMetric(evaluated: EvaluatedConfidence): ConfidenceMetric;
}
