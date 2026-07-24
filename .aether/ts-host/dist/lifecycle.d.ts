/**
 * Experimental lifecycle smoke harness for the TypeScript orchestration host.
 *
 * Drives the plan -> build 1 -> continue lifecycle in explicit simulation mode by:
 * 1. Calling Go --plan-only commands to obtain JSON manifests
 * 2. Building completion files with worker results
 * 3. Calling Go finalizer commands to commit state changes
 * 4. Dispatching build workers with spawn-log/complete via Go CLI
 *
 * Contract: The TS host never writes to .aether/data/ directly. All state
 * mutations go through Go finalizer commands.
 *
 * This is not production orchestration. Real colony work should use the
 * dedicated plan/build/continue host commands so every worker result comes from
 * the active wrapper layer.
 *
 * Satisfies HOST-04 (finalizers called) and HOST-07 (end-to-end lifecycle).
 */
import { type DispatchOptions } from "./worker-dispatch.js";
import { createQueenOrchestrator as _createQueenOrchestrator } from "./queen/orchestrator.js";
/** Test-only: inject a mock createQueenOrchestrator. */
export declare function __setCreateQueenOrchestrator(fn: typeof _createQueenOrchestrator): void;
/** Test-only: restore the real createQueenOrchestrator. */
export declare function __restoreCreateQueenOrchestrator(): void;
/** Options for the full lifecycle orchestrator. */
export interface LifecycleOptions extends DispatchOptions {
    /** Phase number to build (default: 1). */
    phase?: number;
    /**
     * When true (default), workers within the same wave are dispatched
     * concurrently via Promise.all.
     */
    parallel?: boolean;
    /**
     * When true (default when TTY), show the live dashboard during build.
     * When false, use plain text narrator output.
     */
    dashboard?: boolean;
    /** When true, skip the pre-build midden threshold check. */
    skipMiddenCheck?: boolean;
    /** When true, run Oracle RALF research before planning. */
    runOracle?: boolean;
    /** Research topic for the Oracle step (default: "auto"). */
    oracleTopic?: string;
}
/** Oracle result summary embedded in LifecycleResult. */
export interface LifecycleOracleResult {
    /** Number of Oracle iterations completed. */
    iterations_completed: number;
    /** Final confidence percentage after Oracle loop. */
    final_confidence: number;
    /** Reason the Oracle loop stopped. */
    stop_reason: string;
}
/** Result of the lifecycle orchestration. */
export interface LifecycleResult {
    /** Whether the full lifecycle completed successfully. */
    success: boolean;
    /** Steps completed in order. */
    steps_completed: string[];
    /** Oracle result if the Oracle step ran. */
    oracle_result?: LifecycleOracleResult;
    /** Error message if the lifecycle failed. */
    error?: string;
}
/**
 * Run the full plan -> build -> continue lifecycle through the Go CLI.
 *
 * Each step:
 * 1. Calls Go --plan-only to get a manifest (no state mutation)
 * 2. Builds a completion file with worker results
 * 3. Calls the Go finalizer to commit state atomically
 *
 * Error handling: Per HOST-07, if the lifecycle cannot complete, the error
 * message documents the exact blocker.
 *
 * @param opts - Lifecycle options including Go binary path and cwd
 * @returns Lifecycle result with success status and completed steps
 */
export declare function runLifecycle(opts: LifecycleOptions): Promise<LifecycleResult>;
