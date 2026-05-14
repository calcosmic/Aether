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

import type {
  BuildDispatch,
  WorkerResult,
  TerminalWorkerStatus,
} from "../types.js";
import type { BuilderProbeLockResult } from "./types.js";

// ---------------------------------------------------------------------------
// Lock application
// ---------------------------------------------------------------------------

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
export function applyBuilderProbeLock(
  results: WorkerResult[],
  dispatches: BuildDispatch[]
): BuilderProbeLockResult {
  const dispatchByName = new Map<string, BuildDispatch>();
  for (const d of dispatches) {
    dispatchByName.set(d.name, d);
  }

  const hasProbe = hasProbeVerification(results, dispatches);
  let downgraded = false;

  const updated = results.map((r): WorkerResult => {
    const dispatch = dispatchByName.get(r.name);
    const isBuilder = dispatch?.caste === "builder";
    const wasCompleted = r.status === "completed";

    if (isBuilder && wasCompleted && !hasProbe) {
      downgraded = true;
      return {
        ...r,
        status: "code_written" as TerminalWorkerStatus,
        summary: `${r.summary ?? ""} [Builder-Probe Lock: downgraded to code_written — no probe verification]`.trim(),
      };
    }

    return r;
  });

  const summary = downgraded
    ? `Builder-Probe Lock applied: ${updated.filter((r) => r.status === "code_written").length} builder(s) downgraded to code_written (no probe verification)`
    : "Builder-Probe Lock satisfied: no downgrade needed";

  return { results: updated, downgraded, summary };
}

// ---------------------------------------------------------------------------
// Probe detection
// ---------------------------------------------------------------------------

/**
 * Determine whether any probe (or watcher) completed in the results.
 *
 * Probes are identified by caste: "probe" or "watcher".
 *
 * @param results - Worker results
 * @param dispatches - Original build dispatches (for caste lookup)
 * @returns True if at least one probe or watcher completed
 */
export function hasProbeVerification(
  results: WorkerResult[],
  dispatches: BuildDispatch[]
): boolean {
  const dispatchByName = new Map<string, BuildDispatch>();
  for (const d of dispatches) {
    dispatchByName.set(d.name, d);
  }

  return results.some((r) => {
    const dispatch = dispatchByName.get(r.name);
    const isProbe = dispatch?.caste === "probe" || dispatch?.caste === "watcher";
    return isProbe && r.status === "completed";
  });
}

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
export function isBuilderProbeLockSatisfied(
  results: WorkerResult[],
  dispatches: BuildDispatch[]
): boolean {
  const hasBuilder = dispatches.some((d) => d.caste === "builder");
  const hasProbe = dispatches.some(
    (d) => d.caste === "probe" || d.caste === "watcher"
  );

  if (!hasBuilder || !hasProbe) {
    return true;
  }

  return hasProbeVerification(results, dispatches);
}
