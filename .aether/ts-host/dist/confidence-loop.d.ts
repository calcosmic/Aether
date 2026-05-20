/**
 * Confidence-driven iteration loop for the TypeScript orchestration host.
 *
 * Drives build/continue iteration by tracking confidence across iterations,
 * detecting diminishing returns, enforcing a hard iteration cap and cumulative
 * budget, and emitting ceremony progress markers.
 *
 * Satisfies ITER-01 (max 3 iterations), ITER-03 (diminishing returns),
 * ITER-05 (cumulative budget tracking), and ITER-06 (ceremony output).
 */
/** Options that control the confidence loop behaviour. */
export interface ConfidenceLoopOptions {
    /** Hard cap on iterations. Default: 3. */
    maxIterations?: number;
    /** Confidence target (0-100). Loop stops when met. Default: 80. */
    confidenceTarget?: number;
    /** Delta-per-iteration percentage below which progress is "diminishing". Default: 5. */
    diminishingReturnsThreshold?: number;
    /** How many consecutive diminishing iterations trigger a stop. Default: 2. */
    diminishingReturnsWindow?: number;
    /** Total worker budget (0 = unlimited). Default: 20. */
    totalBudget?: number;
}
/** Snapshot of the loop's internal state after an evaluate call. */
export interface IterationState {
    /** How many iterations have completed. */
    iterationCount: number;
    /** Confidence score recorded after each iteration. */
    confidenceHistory: number[];
    /** Workers dispatched so far across all iterations. */
    budgetConsumed: number;
    /** Workers remaining in the budget. */
    budgetRemaining: number;
}
/** Result returned by a single evaluate call. */
export interface ConfidenceResult {
    /** Whether the loop should continue into another iteration. */
    shouldContinue: boolean;
    /** Machine-readable stop reason (empty string while continuing). */
    stopReason: string;
    /** Confidence score passed to this evaluate call. */
    currentConfidence: number;
    /** Confidence change from the previous iteration (0 on the first). */
    delta: number;
    /** Iterations completed so far (including this one). */
    iterationCount: number;
    /** Workers remaining after this iteration consumed its budget. */
    budgetRemaining: number;
}
/**
 * Reusable iteration driver used by build and continue pipelines.
 *
 * Call `evaluate()` after each iteration's finalize step. The method records
 * the confidence score, tracks budget, checks stop conditions, and returns
 * a result telling the caller whether to iterate again.
 *
 * Ceremony output is available via `renderIterationCeremony()`.
 */
export declare class ConfidenceLoop {
    private readonly maxIterations;
    private readonly confidenceTarget;
    private readonly diminishingThreshold;
    private readonly diminishingWindow;
    private readonly totalBudget;
    private confidenceHistory;
    private budgetConsumed;
    constructor(opts?: ConfidenceLoopOptions);
    /**
     * Record an iteration result and decide whether the loop should continue.
     *
     * @param currentConfidence - Confidence score (0-100) from this iteration.
     * @param workersUsed - Number of workers consumed during this iteration.
     * @returns A ConfidenceResult with the continue decision and metadata.
     */
    evaluate(currentConfidence: number, workersUsed: number): ConfidenceResult;
    /**
     * Render a ceremony marker string for the current iteration state.
     *
     * Format: `── Iteration N: confidence XX% (delta YY%, budget ZZ workers remaining) ──`
     *
     * @param result - The ConfidenceResult from the last evaluate call.
     * @param stopReason - Optional human-readable stop reason for the final iteration.
     */
    renderIterationCeremony(result: ConfidenceResult): string;
    /**
     * Return a snapshot of the current loop state.
     */
    getState(): IterationState;
    /**
     * Reset the loop to its initial state so the instance can be reused.
     */
    reset(): void;
    /**
     * Evaluate all stop conditions. Returns the reason string if any condition
     * is met, or "" if the loop should continue.
     */
    private checkStopConditions;
    /**
     * Detect diminishing returns: true when the last `window` deltas are all
     * below the configured threshold.
     */
    private isDiminishingReturns;
}
