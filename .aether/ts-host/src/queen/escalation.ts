/**
 * Escalation and recovery action handling for the Queen orchestrator.
 *
 * Classifies worker failures, maps them to recovery actions, and formats
 * human-readable recovery summaries. Delegates failure classification to
 * the Go CLI when available, with fallback heuristics.
 *
 * Satisfies ORC-05 (escalation delegation).
 */

import { callGoJSON } from "../go-bridge.js";
import type { GoBridgeOptions } from "../go-bridge.js";
import type {
  RecoveryAction,
  FailureClassification,
} from "./types.js";
import type { DispatchResult } from "../worker-dispatch.js";

// ---------------------------------------------------------------------------
// Failure classification
// ---------------------------------------------------------------------------

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
export function classifyFailure(
  opts: GoBridgeOptions,
  status: string,
  summary: string
): FailureClassification {
  try {
    const result = callGoJSON<{
      classification?: FailureClassification;
    }>(opts, [
      "failure-classify",
      "--status",
      status,
      "--summary",
      summary,
    ]);
    if (result.classification) {
      return result.classification;
    }
  } catch {
    // Go command unavailable — fall through to heuristic
  }

  // Fallback heuristic
  switch (status) {
    case "failed":
      return "recoverable";
    case "blocked":
      return "blocking";
    case "timeout":
      return "requires-attempt";
    default:
      return "unknown";
  }
}

// ---------------------------------------------------------------------------
// Recovery action mapping
// ---------------------------------------------------------------------------

/**
 * Map a failure classification to a recovery action type.
 *
 * - "recoverable" → "retry"
 * - "blocking" → "escalate"
 * - "requires-attempt" → "fixer_dispatch"
 * - "unknown" → "peer_reassign"
 *
 * @param classification - Failure classification
 * @returns Recovery action type
 */
function mapClassificationToAction(
  classification: FailureClassification
): RecoveryAction["type"] {
  switch (classification) {
    case "recoverable":
      return "retry";
    case "blocking":
      return "escalate";
    case "requires-attempt":
      return "fixer_dispatch";
    case "unknown":
    default:
      return "peer_reassign";
  }
}

/**
 * Handle wave failures by classifying each and mapping to recovery actions.
 *
 * @param opts - Go bridge options
 * @param failures - Failed dispatch results from wave dispatch
 * @returns Array of recovery actions
 */
export function handleWaveFailures(
  opts: GoBridgeOptions,
  failures: DispatchResult[]
): RecoveryAction[] {
  return failures.map((f): RecoveryAction => {
    const classification = classifyFailure(opts, f.status, f.summary);
    const type = mapClassificationToAction(classification);
    const reason = `${classification}: ${f.summary}`;
    return { type, worker: f.name, reason };
  });
}

// ---------------------------------------------------------------------------
// Summary formatting
// ---------------------------------------------------------------------------

/**
 * Format recovery actions as a human-readable summary.
 *
 * @param actions - Recovery actions
 * @returns Formatted summary string
 */
export function formatRecoveryActions(actions: RecoveryAction[]): string {
  if (actions.length === 0) {
    return "No recovery actions needed.";
  }

  const lines: string[] = [
    `Recovery actions (${actions.length}):`,
  ];

  for (const action of actions) {
    const reason = action.reason ? ` — ${action.reason}` : "";
    lines.push(`  - ${action.type} for ${action.worker}${reason}`);
  }

  return lines.join("\n");
}
