/**
 * Oracle RALF lifecycle orchestrator for the TypeScript host.
 *
 * Drives the Oracle research iteration loop by:
 * 1. Calling `aether oracle-iterate --plan-only` to get the iteration manifest
 * 2. Coordinating Oracle workers through the Go-owned provider adapter
 * 3. Building completion files with worker results
 * 4. Calling `aether oracle-iterate-finalize --completion-file` to commit state
 * 5. Repeating until stop conditions are met
 *
 * Contract: The TS host never writes to .aether/data/oracle/ directly. All state
 * mutations go through Go finalizer commands.
 *
 * Satisfies TOL-01 (manifest-driven iteration), TOL-02 (worker dispatch),
 * TOL-03 (finalize via Go), and HOST-05 (boundary enforcement).
 */
import type { GoBridgeOptions } from "./go-bridge.js";
import { callGoJSON, writeCompletionFile } from "./go-bridge.js";
import type { OracleIterationState, OracleWorkerResponse, BuildDispatch } from "./types.js";
import { type DispatchOptions, type DispatchResult } from "./worker-dispatch.js";
import { type CeremonyAdapter } from "./ceremony-adapter.js";
/** Test-only: inject a mock callGoJSON. */
export declare function __setCallGoJSON(fn: typeof callGoJSON): void;
/** Test-only: restore the real callGoJSON. */
export declare function __restoreCallGoJSON(): void;
/** Test-only: inject a mock writeCompletionFile. */
export declare function __setWriteCompletionFile(fn: typeof writeCompletionFile): void;
/** Test-only: restore the real writeCompletionFile. */
export declare function __restoreWriteCompletionFile(): void;
/** Test-only: inject a mock dispatchSingleWorker. */
export declare function __setDispatchSingleWorker(fn: (opts: DispatchOptions, dispatch: BuildDispatch) => Promise<DispatchResult>): void;
/** Test-only: restore the real dispatchSingleWorker. */
export declare function __restoreDispatchSingleWorker(): void;
/** Test-only: inject a mock createCeremonyAdapter. */
export declare function __setCreateCeremonyAdapter(factory: (opts: GoBridgeOptions) => CeremonyAdapter): void;
/** Test-only: restore the real createCeremonyAdapter. */
export declare function __restoreCreateCeremonyAdapter(): void;
/** Options for the Oracle lifecycle orchestrator. */
export interface OracleLifecycleOptions extends DispatchOptions {
    /** Research topic (default: "auto"). */
    topic?: string;
    /** When true (default), show the live dashboard during dispatch. */
    dashboard?: boolean;
    /** Override max iterations (otherwise from Go manifest). */
    maxIterations?: number;
    /** Override confidence target (otherwise from Go manifest). */
    confidenceTarget?: number;
}
/** Result of the Oracle lifecycle orchestration. */
export interface OracleLifecycleResult {
    /** Whether the Oracle loop completed successfully. */
    success: boolean;
    /** Number of iterations completed. */
    iterations_completed: number;
    /** Final confidence percentage. */
    final_confidence: number;
    /** Target confidence percentage. */
    confidence_target: number;
    /** Reason the loop stopped. */
    stop_reason: string;
    /** Steps completed in order. */
    steps_completed: string[];
    /** Error message if the loop failed. */
    error?: string;
}
/**
 * Run the Oracle RALF iteration loop through the Go CLI.
 *
 * Each iteration:
 * 1. Calls `aether oracle-iterate --plan-only` to get a manifest (no state mutation)
 * 2. Checks stop conditions from the manifest
 * 3. Dispatches Oracle workers via `dispatchSingleWorker`
 * 4. Builds a completion file with worker results
 * 5. Calls `aether oracle-iterate-finalize` to commit state atomically
 * 6. Loops back if `should_continue` is true
 *
 * Error handling: Per HOST-07, if the loop cannot complete, the error
 * message documents the exact blocker.
 *
 * @param opts - Oracle lifecycle options including Go binary path and cwd
 * @returns Oracle lifecycle result with success status and completed iterations
 */
export declare function runOracleLifecycle(opts: OracleLifecycleOptions): Promise<OracleLifecycleResult>;
/**
 * Build an OracleWorkerResponse from a DispatchResult.
 *
 * @param result - Dispatch result from dispatchSingleWorker
 * @param questionId - Question or iteration identifier
 * @returns OracleWorkerResponse suitable for the completion file
 */
export declare function buildOracleWorkerResponse(result: import("./worker-dispatch.js").DispatchResult, questionId: string): OracleWorkerResponse;
/**
 * Check stop conditions for the Oracle loop.
 *
 * Computes stop conditions from the manifest state and options.
 * Returns stop=true with a reason if any condition is met.
 *
 * @param state - Current Oracle iteration state from the manifest
 * @param opts - Oracle lifecycle options (for overrides)
 * @returns Stop check result with boolean and reason
 */
export declare function checkStopConditions(state: OracleIterationState, opts: OracleLifecycleOptions): {
    stop: boolean;
    reason: string;
};
