/**
 * Midden threshold check for the Queen orchestrator.
 *
 * Calls the Go CLI to review recent midden entries. If the total exceeds
 * a configurable threshold, emits a REDIRECT pheromone to steer the colony
 * away from risky patterns.
 *
 * Satisfies ORC-04 (midden threshold checking).
 */
import type { GoBridgeOptions } from "../go-bridge.js";
import type { MiddenCheckResult } from "./types.js";
/**
 * Check whether the midden threshold has been exceeded.
 *
 * Calls `aether midden-review` via the Go CLI to get recent failure counts.
 * If total > threshold, emits a REDIRECT pheromone via `aether pheromone-write`.
 *
 * @param opts - Go bridge options (binary path and cwd)
 * @param threshold - Maximum acceptable midden entries before steering (default: 3)
 * @returns Midden check result with exceeded flag and category breakdown
 */
export declare function checkMiddenThreshold(opts: GoBridgeOptions, threshold?: number): MiddenCheckResult;
/**
 * Format a midden check result as a human-readable summary.
 *
 * @param result - Midden check result
 * @returns Formatted summary string
 */
export declare function formatMiddenSummary(result: MiddenCheckResult): string;
