/**
 * Builder-Probe Lock enforcement for the Queen orchestrator.
 *
 * The Builder-Probe Lock ensures that builder workers are not marked as
 * "completed" unless a probe (or watcher) has verified their work. If a
 * builder completed but no probe verified, the builder status is downgraded
 * to "code_written".
 *
 * Satisfies ORC-03 (Builder-Probe Lock).
 */
import type { BuildDispatch, WorkerResult } from "../types.js";
import type { BuilderProbeLockResult } from "./types.js";
/**
 * Apply the Builder-Probe Lock to worker results.
 *
 * For each builder worker that completed: if no probe completed in the
 * results, downgrade the builder's status to "code_written". Otherwise
 * preserve the original status.
 *
 * @param results - Worker results after wave dispatch
 * @param dispatches - Original build dispatches (for caste lookup)
 * @returns Lock result with possibly downgraded results
 */
export declare function applyBuilderProbeLock(results: WorkerResult[], dispatches: BuildDispatch[]): BuilderProbeLockResult;
/**
 * Determine whether any probe (or watcher) completed in the results.
 *
 * Probes are identified by caste: "probe" or "watcher".
 *
 * @param results - Worker results
 * @param dispatches - Original build dispatches (for caste lookup)
 * @returns True if at least one probe or watcher completed
 */
export declare function hasProbeVerification(results: WorkerResult[], dispatches: BuildDispatch[]): boolean;
/**
 * Check whether the Builder-Probe Lock is satisfied.
 *
 * The lock is satisfied when:
 * - There are no builders in the dispatches, OR
 * - There are no probes/watchers in the dispatches, OR
 * - At least one probe/watcher completed
 *
 * @param results - Worker results
 * @param dispatches - Original build dispatches
 * @returns True if the lock is satisfied
 */
export declare function isBuilderProbeLockSatisfied(results: WorkerResult[], dispatches: BuildDispatch[]): boolean;
