/**
 * Oracle RALF lifecycle orchestrator for the TypeScript host.
 *
 * Drives the Oracle research iteration loop by:
 * 1. Calling `aether oracle-iterate --plan-only` to get the iteration manifest
 * 2. Dispatching Oracle workers via the platform dispatcher
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
import { callGoJSON, writeCompletionFile } from "./go-bridge.js";
import { dispatchSingleWorker } from "./worker-dispatch.js";
import { createCeremonyAdapter, } from "./ceremony-adapter.js";
// Mutable references for test injection.
let _callGoJSONRef = callGoJSON;
let _writeCompletionFileRef = writeCompletionFile;
let _dispatchSingleWorkerRef = dispatchSingleWorker;
/** Test-only: inject a mock callGoJSON. */
export function __setCallGoJSON(fn) {
    _callGoJSONRef = fn;
}
/** Test-only: restore the real callGoJSON. */
export function __restoreCallGoJSON() {
    _callGoJSONRef = callGoJSON;
}
/** Test-only: inject a mock writeCompletionFile. */
export function __setWriteCompletionFile(fn) {
    _writeCompletionFileRef = fn;
}
/** Test-only: restore the real writeCompletionFile. */
export function __restoreWriteCompletionFile() {
    _writeCompletionFileRef = writeCompletionFile;
}
/** Test-only: inject a mock dispatchSingleWorker. */
export function __setDispatchSingleWorker(fn) {
    _dispatchSingleWorkerRef = fn;
}
/** Test-only: restore the real dispatchSingleWorker. */
export function __restoreDispatchSingleWorker() {
    _dispatchSingleWorkerRef = dispatchSingleWorker;
}
// Mutable reference for ceremony adapter injection (testing).
let _createCeremonyAdapterRef = createCeremonyAdapter;
/** Test-only: inject a mock createCeremonyAdapter. */
export function __setCreateCeremonyAdapter(factory) {
    _createCeremonyAdapterRef = factory;
}
/** Test-only: restore the real createCeremonyAdapter. */
export function __restoreCreateCeremonyAdapter() {
    _createCeremonyAdapterRef = createCeremonyAdapter;
}
function callGoJSONRef(opts, args) {
    return _callGoJSONRef(opts, args);
}
function writeCompletionFileRef(dir, filename, data) {
    return _writeCompletionFileRef(dir, filename, data);
}
function dispatchSingleWorkerRef(opts, dispatch) {
    return _dispatchSingleWorkerRef(opts, dispatch);
}
function emitCeremonyOutput(output) {
    if (output.trim() === "")
        return;
    process.stderr.write(output.endsWith("\n") ? output : `${output}\n`);
}
// ---------------------------------------------------------------------------
// Oracle lifecycle orchestrator
// ---------------------------------------------------------------------------
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
export async function runOracleLifecycle(opts) {
    const stepsCompleted = [];
    const topic = opts.topic ?? "auto";
    let iterationsCompleted = 0;
    let finalConfidence = 0;
    let confidenceTarget = 0;
    let stopReason = "unknown";
    try {
        const ceremony = _createCeremonyAdapterRef(opts);
        // eslint-disable-next-line no-constant-condition
        while (true) {
            // ── Step 1: Get iteration manifest from Go ───────────────────────────
            const manifest = callGoJSONRef(opts, [
                "oracle-iterate",
                "--plan-only",
                "--topic",
                topic,
            ]);
            if (!manifest.iteration_manifest) {
                throw new Error("oracle-iterate --plan-only returned no iteration_manifest");
            }
            const state = manifest.iteration_manifest;
            confidenceTarget = state.confidence_target;
            iterationsCompleted = state.current_iteration;
            stepsCompleted.push(`iteration-${state.current_iteration}-manifest`);
            // ── Step 2: Check stop conditions ────────────────────────────────────
            const stopCheck = checkStopConditions(state, opts);
            if (stopCheck.stop) {
                stopReason = stopCheck.reason;
                stepsCompleted.push(`stopped:${stopReason}`);
                break;
            }
            // ── Step 3: Dispatch Oracle workers ──────────────────────────────────
            const workers = state.workers;
            if (workers.length === 0) {
                throw new Error("Oracle manifest contains no workers");
            }
            // Build a BuildDispatch from the first Oracle worker
            const oracleWorker = workers[0];
            const dispatch = {
                stage: "oracle",
                caste: oracleWorker.caste,
                name: oracleWorker.name,
                task: oracleWorker.task,
                status: "pending",
                summary: oracleWorker.brief,
                task_brief: oracleWorker.brief,
            };
            // Ceremony: render spawn-plan and wave-start before dispatch
            const ceremonyEnvelope = { iteration_manifest: state };
            emitCeremonyOutput(ceremony.renderSpawnPlan("build", ceremonyEnvelope));
            emitCeremonyOutput(ceremony.renderWaveStart("build", ceremonyEnvelope, 1));
            const dispatchResult = await dispatchSingleWorkerRef(opts, dispatch);
            stepsCompleted.push(`iteration-${state.current_iteration}-dispatch`);
            // Ceremony: render worker-complete after dispatch
            emitCeremonyOutput(ceremony.renderWorkerComplete("build", dispatchResult));
            // ── Step 4: Build completion file ────────────────────────────────────
            const workerResponse = buildOracleWorkerResponse(dispatchResult, String(state.current_iteration));
            const completionData = {
                iteration_manifest: state,
                dispatches: [
                    {
                        worker: dispatchResult.name,
                        status: dispatchResult.status,
                        summary: dispatchResult.summary,
                    },
                ],
                current_iteration: state.current_iteration + 1,
                should_continue: true,
                worker_response: workerResponse,
            };
            const completionPath = writeCompletionFileRef("aether-oracle", `oracle-completion-${state.current_iteration}.json`, completionData);
            stepsCompleted.push(`iteration-${state.current_iteration}-completion`);
            // ── Step 5: Finalize via Go ──────────────────────────────────────────
            const finalizeResult = callGoJSONRef(opts, [
                "oracle-iterate-finalize",
                "--completion-file",
                completionPath,
            ]);
            finalConfidence = finalizeResult.current_confidence;
            confidenceTarget = finalizeResult.confidence_target;
            iterationsCompleted = state.current_iteration;
            stepsCompleted.push(`iteration-${state.current_iteration}-finalize`);
            // Ceremony: render closeout after finalize
            emitCeremonyOutput(ceremony.renderCloseout("build", completionPath));
            // ── Step 6: Check if we should continue ──────────────────────────────
            if (!finalizeResult.should_continue) {
                stopReason = "finalize:should_continue=false";
                stepsCompleted.push(`stopped:${stopReason}`);
                break;
            }
            // Safety: also check max iterations as a hard ceiling
            if (state.current_iteration >= state.max_iterations) {
                stopReason = "max_iterations_met";
                stepsCompleted.push(`stopped:${stopReason}`);
                break;
            }
        }
        return {
            success: true,
            iterations_completed: iterationsCompleted,
            final_confidence: finalConfidence,
            confidence_target: confidenceTarget,
            stop_reason: stopReason,
            steps_completed: stepsCompleted,
        };
    }
    catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        const errorMessage = `Oracle lifecycle failed: ${message}`;
        process.stderr.write(`Oracle lifecycle error: ${errorMessage}\n`);
        return {
            success: false,
            iterations_completed: iterationsCompleted,
            final_confidence: finalConfidence,
            confidence_target: confidenceTarget,
            stop_reason: "error",
            steps_completed: stepsCompleted,
            error: errorMessage,
        };
    }
}
// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------
/**
 * Build an OracleWorkerResponse from a DispatchResult.
 *
 * @param result - Dispatch result from dispatchSingleWorker
 * @param questionId - Question or iteration identifier
 * @returns OracleWorkerResponse suitable for the completion file
 */
export function buildOracleWorkerResponse(result, questionId) {
    const response = {
        question_id: questionId,
        status: result.status,
        summary: result.summary,
    };
    if (result.files_modified && result.files_modified.length > 0) {
        response.findings = [
            {
                text: `Worker ${result.name} completed with file modifications`,
                evidence: result.files_modified.map((f) => ({
                    title: f,
                    location: f,
                    type: "code",
                })),
            },
        ];
    }
    return response;
}
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
export function checkStopConditions(state, opts) {
    const maxIterations = opts.maxIterations ?? state.max_iterations;
    const confidenceTarget = opts.confidenceTarget ?? state.confidence_target;
    // max_iterations_met
    if (state.current_iteration >= maxIterations) {
        return { stop: true, reason: "max_iterations_met" };
    }
    // confidence_met (if we had current confidence in the manifest)
    // The Go manifest does not include current_confidence yet, but we check
    // the finalize result after each iteration. This is a pre-dispatch check.
    // no_progress: could be computed from history if available
    return { stop: false, reason: "" };
}
