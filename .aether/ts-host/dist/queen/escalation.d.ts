/**
 * Escalation and recovery action handling for the Queen orchestrator.
 *
 * Classifies worker failures, maps them to recovery actions, and formats
 * human-readable recovery summaries. Delegates failure classification to
 * the Go CLI when available, with fallback heuristics.
 *
 * Satisfies ORC-05 (escalation delegation).
 */
import type { GoBridgeOptions } from "../go-bridge.js";
import type { RecoveryAction, FailureClassification } from "./types.js";
import type { DispatchResult } from "../worker-dispatch.js";
/**
 * Classify a worker failure using Go CLI or fallback heuristics.
 *
 * Attempts `aether failure-classify` via callGoJSON. If that fails,
 * falls back to heuristic rules:
 * - "failed" → "recoverable"
 * - "blocked" → "blocking"
 * - "timeout" → "requires-attempt"
 * - anything else → "unknown"
 *
 * @param opts - Go bridge options
 * @param status - Terminal worker status
 * @param summary - Worker summary text
 * @returns Failure classification
 */
export declare function classifyFailure(opts: GoBridgeOptions, status: string, summary: string): FailureClassification;
/**
 * Handle wave failures by classifying each and mapping to recovery actions.
 *
 * @param opts - Go bridge options
 * @param failures - Failed dispatch results from wave dispatch
 * @returns Array of recovery actions
 */
export declare function handleWaveFailures(opts: GoBridgeOptions, failures: DispatchResult[]): RecoveryAction[];
/**
 * Format recovery actions as a human-readable summary.
 *
 * @param actions - Recovery actions
 * @returns Formatted summary string
 */
export declare function formatRecoveryActions(actions: RecoveryAction[]): string;
