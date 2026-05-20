/**
 * Worker dispatch module for the TypeScript orchestration host.
 *
 * Iterates over Go manifest dispatches, records spawn-log before each worker,
 * dispatches the worker (simulated or real), and records spawn-complete
 * after. Restores the visible worker activity lost in the Bash-to-Go migration.
 *
 * When simulateWorkers is false, dispatches real workers via platform CLI
 * subprocess using platform-dispatcher.ts, prompt-assembler.ts, and
 * claims-parser.ts.
 *
 * Satisfies HOST-03 (visible dispatch from manifest) and HOST-06 (spawn
 * lifecycle events via Go CLI).
 */
import type { GoBridgeOptions } from "./go-bridge.js";
import type { BuildDispatch, WorkerResult, TerminalWorkerStatus, SpawnClaim, WorkerHandoff } from "./types.js";
import { type Platform } from "./platform-dispatcher.js";
/** Result of dispatching a single worker. */
export interface DispatchResult {
    /** Worker name from the manifest dispatch. */
    name: string;
    /** Terminal status after dispatch attempt. */
    status: TerminalWorkerStatus;
    /** Summary of what the worker did (or why it failed). */
    summary: string;
    /** Approximate duration in seconds. */
    duration?: number;
    /** Files modified by the worker (simulated or real). */
    files_modified?: string[];
    /** Files created by the worker (simulated or real). */
    files_created?: string[];
    /** Tests written by the worker (simulated or real). */
    tests_written?: string[];
    /** Detected platform for debugging. */
    detectedPlatform?: string;
    /** Sub-workers requested by this worker via structured spawn claims. */
    spawns?: SpawnClaim[];
    /** Worker handoff data, including child_results for spawned children (SPAWN-04). */
    handoff?: WorkerHandoff;
}
/** Options for worker dispatch, extending Go bridge options. */
export interface DispatchOptions extends GoBridgeOptions {
    /**
     * When true (default), simulate worker execution instead of spawning
     * a real platform CLI. The prototype uses simulation.
     */
    simulateWorkers?: boolean;
    /**
     * Spawn orchestrator for processing worker spawn claims after wave completion.
     * Passed through to wave-orchestrator for child worker dispatch (SPAWN-01).
     */
    spawnOrchestrator?: import("./spawn-orchestrator.js").SpawnOrchestrator;
    /**
     * File paths that actually exist in the repo, used as simulated worker
     * file claims. Must be real repo-relative paths that exist on disk,
     * because the Go build-finalizer validates all file claims.
     */
    simulatedFileClaims?: string[];
    /**
     * When true (default), workers within the same wave are dispatched
     * concurrently via Promise.all. When false, they run sequentially.
     */
    parallel?: boolean;
    /**
     * Maximum number of retry attempts for a failed worker.
     * Default: 2
     */
    retryLimit?: number;
    /**
     * Base delay between retry attempts in milliseconds.
     * Default: 5000
     */
    retryDelayMs?: number;
    /**
     * Timeout for each worker dispatch in milliseconds.
     * Default: 600000 (10 minutes)
     */
    timeoutMs?: number;
}
/**
 * Dispatch a single worker from a manifest dispatch entry.
 *
 * Lifecycle:
 * 1. Call `aether spawn-log` to record the spawn before dispatch.
 * 2. Execute the worker (simulated or real).
 * 3. Call `aether spawn-complete` to record the outcome.
 *
 * Spawn-log failure does not block dispatch. Spawn-complete is always
 * attempted, even on dispatch error.
 *
 * @param opts - Dispatch options including Go binary path and cwd
 * @param dispatch - Build dispatch entry from the Go manifest
 * @returns Dispatch result with name, status, and summary
 */
export declare function dispatchSingleWorker(opts: DispatchOptions, dispatch: BuildDispatch): Promise<DispatchResult>;
export declare function buildPromptForDispatch(opts: GoBridgeOptions, dispatch: BuildDispatch, platform: Platform, agentName?: string): string;
export interface GoPromptContextResult {
    prompt_section?: string;
    context?: string;
}
export declare function resolveGoPromptContext(opts: GoBridgeOptions): string;
export declare function sanitizeWorkerDiagnosticOutput(value: string): string;
/**
 * Check if an error is classified as an authentication error.
 *
 * Auth errors should halt the build immediately (D-01).
 * The build pipeline (Plan 03+) uses this to decide whether to halt.
 *
 * @param error - The error to check
 * @returns true if the error is an auth/config error
 */
export declare function isAuthError(error: unknown): boolean;
/**
 * Dispatch multiple workers from a manifest, grouped by wave.
 *
 * Waves are processed sequentially. Within each wave, workers are
 * dispatched in parallel by default (parallel=true). Delegates to
 * wave-orchestrator.ts for wave grouping and parallel dispatch.
 *
 * @param opts - Dispatch options including Go binary path and cwd
 * @param dispatches - Array of build dispatch entries from the manifest
 * @returns Array of dispatch results, one per input dispatch
 */
export declare function dispatchWorkers(opts: DispatchOptions, dispatches: BuildDispatch[]): Promise<DispatchResult[]>;
/**
 * Map dispatch entries and their results to WorkerResult objects for
 * the Go build finalizer.
 *
 * Matches results to dispatches by name (not by index) to handle
 * wave-grouped re-ordering from dispatchWorkers.
 *
 * Preserves manifest fields (caste, task_id, stage, wave) alongside
 * dispatch outcomes (status, summary).
 *
 * @param dispatches - Original build dispatch entries from the manifest
 * @param results - Dispatch results from dispatchWorkers
 * @returns WorkerResult array suitable for the Go finalizer completion file
 */
export declare function toWorkerResults(dispatches: BuildDispatch[], results: DispatchResult[]): WorkerResult[];
