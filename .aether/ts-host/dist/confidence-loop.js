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
// ---------------------------------------------------------------------------
// Defaults
// ---------------------------------------------------------------------------
const DEFAULT_MAX_ITERATIONS = 3;
const DEFAULT_CONFIDENCE_TARGET = 80;
const DEFAULT_DIMINISHING_THRESHOLD = 5;
const DEFAULT_DIMINISHING_WINDOW = 2;
const DEFAULT_TOTAL_BUDGET = 20;
// ---------------------------------------------------------------------------
// ConfidenceLoop
// ---------------------------------------------------------------------------
/**
 * Reusable iteration driver used by build and continue pipelines.
 *
 * Call `evaluate()` after each iteration's finalize step. The method records
 * the confidence score, tracks budget, checks stop conditions, and returns
 * a result telling the caller whether to iterate again.
 *
 * Ceremony output is available via `renderIterationCeremony()`.
 */
export class ConfidenceLoop {
    maxIterations;
    confidenceTarget;
    diminishingThreshold;
    diminishingWindow;
    totalBudget;
    confidenceHistory = [];
    budgetConsumed = 0;
    constructor(opts = {}) {
        this.maxIterations = opts.maxIterations ?? DEFAULT_MAX_ITERATIONS;
        this.confidenceTarget = opts.confidenceTarget ?? DEFAULT_CONFIDENCE_TARGET;
        this.diminishingThreshold =
            opts.diminishingReturnsThreshold ?? DEFAULT_DIMINISHING_THRESHOLD;
        this.diminishingWindow =
            opts.diminishingReturnsWindow ?? DEFAULT_DIMINISHING_WINDOW;
        this.totalBudget = opts.totalBudget ?? DEFAULT_TOTAL_BUDGET;
    }
    // -----------------------------------------------------------------------
    // Core API
    // -----------------------------------------------------------------------
    /**
     * Record an iteration result and decide whether the loop should continue.
     *
     * @param currentConfidence - Confidence score (0-100) from this iteration.
     * @param workersUsed - Number of workers consumed during this iteration.
     * @returns A ConfidenceResult with the continue decision and metadata.
     */
    evaluate(currentConfidence, workersUsed) {
        // Track budget (only when budget tracking is enabled)
        if (this.totalBudget > 0) {
            this.budgetConsumed += workersUsed;
        }
        const budgetRemaining = this.totalBudget > 0
            ? this.totalBudget - this.budgetConsumed
            : 0;
        // Record confidence
        this.confidenceHistory.push(currentConfidence);
        const iterationCount = this.confidenceHistory.length;
        // Compute delta from previous iteration (0 on first iteration)
        const delta = iterationCount >= 2
            ? currentConfidence - this.confidenceHistory[iterationCount - 2]
            : 0;
        // Check stop conditions in priority order
        const stopReason = this.checkStopConditions(currentConfidence, iterationCount, budgetRemaining);
        return {
            shouldContinue: stopReason === "",
            stopReason,
            currentConfidence,
            delta,
            iterationCount,
            budgetRemaining,
        };
    }
    /**
     * Render a ceremony marker string for the current iteration state.
     *
     * Format: `── Iteration N: confidence XX% (delta YY%, budget ZZ workers remaining) ──`
     *
     * @param result - The ConfidenceResult from the last evaluate call.
     * @param stopReason - Optional human-readable stop reason for the final iteration.
     */
    renderIterationCeremony(result) {
        const parts = [
            `Iteration ${result.iterationCount}:`,
            `confidence ${result.currentConfidence}%`,
            `(delta ${result.delta >= 0 ? "+" : ""}${result.delta}%,`,
            `budget ${result.budgetRemaining} workers remaining)`,
        ];
        if (result.stopReason) {
            parts.push(`[${result.stopReason}]`);
        }
        return `\u2500\u2500 ${parts.join(" ")} \u2500\u2500`;
    }
    /**
     * Return a snapshot of the current loop state.
     */
    getState() {
        return {
            iterationCount: this.confidenceHistory.length,
            confidenceHistory: [...this.confidenceHistory],
            budgetConsumed: this.budgetConsumed,
            budgetRemaining: this.totalBudget - this.budgetConsumed,
        };
    }
    /**
     * Reset the loop to its initial state so the instance can be reused.
     */
    reset() {
        this.confidenceHistory = [];
        this.budgetConsumed = 0;
    }
    // -----------------------------------------------------------------------
    // Internals
    // -----------------------------------------------------------------------
    /**
     * Evaluate all stop conditions. Returns the reason string if any condition
     * is met, or "" if the loop should continue.
     */
    checkStopConditions(currentConfidence, iterationCount, budgetRemaining) {
        // 1. Hard cap on iterations (ITER-01)
        if (iterationCount >= this.maxIterations) {
            return "max_iterations_met";
        }
        // 2. Confidence target met
        if (currentConfidence >= this.confidenceTarget) {
            return "confidence_target_met";
        }
        // 3. Budget exhausted (ITER-05) — only when budget tracking is active
        if (this.totalBudget > 0 && budgetRemaining <= 0) {
            return "budget_exhausted";
        }
        // 4. Diminishing returns (ITER-03)
        if (this.isDiminishingReturns()) {
            return "diminishing_returns";
        }
        return "";
    }
    /**
     * Detect diminishing returns: true when the last `window` deltas are all
     * below the configured threshold.
     */
    isDiminishingReturns() {
        const history = this.confidenceHistory;
        if (history.length < this.diminishingWindow + 1) {
            // Need at least window+1 data points to produce `window` deltas.
            return false;
        }
        // Compute the last `window` deltas.
        const deltas = [];
        const startIdx = history.length - this.diminishingWindow;
        for (let i = startIdx; i < history.length; i++) {
            deltas.push(Math.abs(history[i] - history[i - 1]));
        }
        // All deltas must be strictly below the threshold.
        return deltas.every((d) => d < this.diminishingThreshold);
    }
}
