/**
 * Wave orchestrator for the TypeScript orchestration host.
 *
 * Groups workers by wave, dispatches them in parallel within each wave,
 * and retries failed workers with exponential backoff.
 *
 * Satisfies TS-02 (concurrent wave dispatch) and TS-03 (retry, timeout,
 * graceful error handling).
 */
import type { BuildDispatch } from "./types.js";
import { dispatchSingleWorker, type DispatchOptions, type DispatchResult } from "./worker-dispatch.js";
import { type SpawnOrchestrator } from "./spawn-orchestrator.js";
/** Options for the wave orchestrator, extending DispatchOptions. */
export interface WaveOrchestratorOptions extends DispatchOptions {
    /**
     * When true (default), workers within the same wave are dispatched
     * concurrently via Promise.all. When false, they run sequentially.
     */
    parallel?: boolean;
    /**
     * Maximum number of retry attempts for a failed worker.
     * Default: 1 (delegate deeper recovery to Go orchestrateRecovery)
     */
    retryLimit?: number;
    /**
     * Base delay between retry attempts in milliseconds.
     * The actual delay is retryDelayMs * attempt (exponential backoff).
     * Default: 5000
     */
    retryDelayMs?: number;
    /**
     * Timeout for each worker dispatch in milliseconds.
     * Default: 600000 (10 minutes)
     */
    timeoutMs?: number;
    /**
     * Spawn orchestrator for processing worker spawn claims after wave completion.
     * When provided, completed workers' spawn claims are validated, accepted children
     * are dispatched in a follow-up spawn wave, and child results are attached to
     * parent handoffs (SPAWN-04).
     */
    spawnOrchestrator?: SpawnOrchestrator;
}
/** Result of dispatching a single wave. */
export interface WaveResult {
    /** Wave number. */
    wave: number;
    /** All dispatch results for this wave (including retried successes). */
    results: DispatchResult[];
    /** Dispatch results that ultimately failed after all retries. */
    failures: DispatchResult[];
    /** Total number of retry attempts made across all workers in this wave. */
    retried: number;
}
/**
 * Dispatch a single worker with retry logic and exponential backoff.
 *
 * @param opts - Wave orchestrator options
 * @param dispatch - Build dispatch entry
 * @param attempt - Current attempt number (starts at 1)
 * @returns Dispatch result after retries exhausted or success
 */
export declare function retryDispatch(opts: WaveOrchestratorOptions, dispatch: BuildDispatch, attempt?: number): Promise<DispatchResult>;
/**
 * Dispatch all workers in a single wave.
 *
 * Workers are dispatched in parallel when opts.parallel is true and there
 * are multiple dispatches. Otherwise they run sequentially.
 *
 * Each worker is wrapped with retry logic via retryDispatch.
 *
 * @param opts - Wave orchestrator options
 * @param dispatches - Array of build dispatch entries for this wave
 * @returns Wave result with results, failures, and retry count
 */
export declare function dispatchWave(opts: WaveOrchestratorOptions, dispatches: BuildDispatch[]): Promise<WaveResult>;
/** Test-only: inject a mock dispatchSingleWorker. */
export declare function __setDispatchSingleWorker(fn: typeof dispatchSingleWorker): void;
/** Test-only: restore the real dispatchSingleWorker. */
export declare function __restoreDispatchSingleWorker(): void;
/**
 * Dispatch multiple workers grouped by wave.
 *
 * Waves run sequentially (wave 1 must complete before wave 2 starts).
 * Within each wave, workers run in parallel by default.
 *
 * When a spawnOrchestrator is provided, after each wave completes, spawn
 * claims from completed workers are processed and accepted children are
 * dispatched in a follow-up spawn wave.
 *
 * @param opts - Wave orchestrator options
 * @param dispatches - Array of build dispatch entries from the manifest
 * @returns Array of wave results, one per wave (including spawn waves)
 */
export declare function dispatchWaves(opts: WaveOrchestratorOptions, dispatches: BuildDispatch[]): Promise<WaveResult[]>;
