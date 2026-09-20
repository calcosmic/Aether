/**
 * One place for the worker-status vocabulary on the host lane.
 *
 * The Go runtime owns the truth (cmd/codex_build_finalize.go:
 * isTerminalExternalBuildStatus / isSuccessfulExternalBuildStatus). These
 * sets mirror it. They live in one module because the previous shape —
 * each call site naming statuses inline — silently dropped
 * completed_no_change from every judgement the moment the vocabulary grew.
 */
/** Terminal: the worker stopped and its record is final. */
export const TERMINAL_WORKER_STATUSES = new Set([
    "completed",
    // Honest no-change success (owner ruling D6) and resumable quota
    // interruption (D7). Both sides must agree or the host lane rejects
    // results the Go finalizer accepts.
    "completed_no_change",
    "interrupted",
    "failed",
    "blocked",
    "timeout",
    "manually-reconciled",
    "code_written",
]);
/**
 * Successful: the work SUCCEEDED. `interrupted` is deliberately absent —
 * it is terminal but the work is unfinished, so counting it as success
 * would credit work nobody did.
 */
export const SUCCESSFUL_WORKER_STATUSES = new Set([
    "completed",
    "code_written",
    "completed_no_change",
    "manually-reconciled",
]);
function normalizeWorkerStatus(status) {
    return typeof status === "string" ? status.trim().toLowerCase() : "";
}
export function isTerminalWorkerStatus(status) {
    return TERMINAL_WORKER_STATUSES.has(normalizeWorkerStatus(status));
}
export function isSuccessfulWorkerStatus(status) {
    return SUCCESSFUL_WORKER_STATUSES.has(normalizeWorkerStatus(status));
}
