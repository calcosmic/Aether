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
export declare const TERMINAL_WORKER_STATUSES: ReadonlySet<string>;
/**
 * Successful: the work SUCCEEDED. `interrupted` is deliberately absent —
 * it is terminal but the work is unfinished, so counting it as success
 * would credit work nobody did.
 */
export declare const SUCCESSFUL_WORKER_STATUSES: ReadonlySet<string>;
export declare function isTerminalWorkerStatus(status: unknown): boolean;
export declare function isSuccessfulWorkerStatus(status: unknown): boolean;
