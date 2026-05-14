/**
 * Queen orchestrator for the TypeScript orchestration host.
 *
 * Drives the build phase by:
 * 1. Checking midden thresholds before build
 * 2. Deriving the workflow pattern from dispatch composition
 * 3. Dispatching workers via wave orchestrator
 * 4. Applying the Builder-Probe Lock
 * 5. Handling wave failures with recovery actions
 * 6. Returning a structured result with pattern, recommendation, and actions
 *
 * Satisfies ORC-06 (Queen orchestrator) and integrates ORC-01 through ORC-05.
 */

import type { BuildManifest, BuildDispatch, WorkerResult } from "../types.js";
import type { GoBridgeOptions } from "../go-bridge.js";
import { callGoJSON } from "../go-bridge.js";
import { dispatchWaves, type WaveResult } from "../wave-orchestrator.js";
import { toWorkerResults, type DispatchResult } from "../worker-dispatch.js";
import {
  deriveWorkflowPattern,
  formatQueenRecommendation,
  mapVerificationDepth,
} from "./workflow-patterns.js";
import {
  applyBuilderProbeLock,
  isBuilderProbeLockSatisfied,
} from "./builder-probe-lock.js";
import { checkMiddenThreshold, formatMiddenSummary } from "./midden-check.js";
import { handleWaveFailures, formatRecoveryActions } from "./escalation.js";
import type {
  QueenOrchestratorOptions,
  QueenOrchestratorResult,
  QueenOrchestrator,
  QueenRecommendation,
  MiddenCheckResult,
  RecoveryAction,
} from "./types.js";

// ---------------------------------------------------------------------------
// Orchestrator factory
// ---------------------------------------------------------------------------

/**
 * Create a Queen orchestrator for the given options.
 *
 * @param opts - Queen orchestrator options
 * @returns Queen orchestrator with runBuild method
 */
export function createQueenOrchestrator(
  opts: QueenOrchestratorOptions
): QueenOrchestrator {
  return {
    async runBuild(manifest: {
      dispatches?: BuildDispatch[];
    }): Promise<QueenOrchestratorResult> {
      return runBuild(opts, manifest);
    },
  };
}

// ---------------------------------------------------------------------------
// Build execution
// ---------------------------------------------------------------------------

/**
 * Run the build phase under Queen orchestration.
 *
 * Steps:
 * 1. Pre-build midden check (if not skipped)
 * 2. Derive workflow pattern from manifest dispatches
 * 3. Dispatch workers via dispatchWaves
 * 4. Apply Builder-Probe Lock
 * 5. Handle wave failures
 * 6. Build and return result
 *
 * @param opts - Queen orchestrator options
 * @param manifest - Build manifest with dispatches
 * @returns Queen orchestrator result
 */
export async function runBuild(
  opts: QueenOrchestratorOptions,
  manifest: { dispatches?: BuildDispatch[] }
): Promise<QueenOrchestratorResult> {
  const dispatches = manifest.dispatches ?? [];

  // ── Step 1: Pre-build midden check ───────────────────────────────────────
  let middenResult: MiddenCheckResult | undefined;
  if (!opts.skipMiddenCheck) {
    middenResult = checkMiddenThreshold(opts, 3);
    process.stderr.write(formatMiddenSummary(middenResult) + "\n");
  }

  // ── Step 2: Derive workflow pattern ──────────────────────────────────────
  const pattern = deriveWorkflowPattern(dispatches);
  process.stderr.write(`Queen: workflow pattern = ${pattern}\n`);

  // Build a recommendation from the pattern
  const recommendation = buildRecommendation(pattern, dispatches);
  process.stderr.write(
    `Queen: recommendation = ${formatQueenRecommendation(recommendation)}\n`
  );

  // ── Step 3: Dispatch workers via waves ───────────────────────────────────
  // Forward simulatedFileClaims from lifecycle placeholder creation so
  // that simulated workers produce file claims the Go finalizer accepts.
  const waveOpts = opts.simulatedFileClaims
    ? { ...opts, simulatedFileClaims: opts.simulatedFileClaims }
    : opts;
  const waveResults = await dispatchWaves(waveOpts, dispatches);

  // Flatten wave results to dispatch results
  const dispatchResults: DispatchResult[] = [];
  for (const wr of waveResults) {
    dispatchResults.push(...wr.results);
  }

  // Convert to WorkerResult format
  let workerResults = toWorkerResults(dispatches, dispatchResults);

  // ── Step 4: Apply Builder-Probe Lock ─────────────────────────────────────
  const lockResult = applyBuilderProbeLock(workerResults, dispatches);
  workerResults = lockResult.results;
  process.stderr.write(`Queen: ${lockResult.summary}\n`);

  // ── Step 5: Handle wave failures ─────────────────────────────────────────
  const failures: DispatchResult[] = [];
  for (const wr of waveResults) {
    failures.push(...wr.failures);
  }

  let recoveryActions: RecoveryAction[] | undefined;
  if (failures.length > 0) {
    recoveryActions = handleWaveFailures(opts, failures);
    process.stderr.write(formatRecoveryActions(recoveryActions) + "\n");
  }

  // ── Step 6: Build result ─────────────────────────────────────────────────
  const success =
    failures.length === 0 && isBuilderProbeLockSatisfied(workerResults, dispatches);

  return {
    success,
    workerResults,
    pattern,
    recommendation,
    middenResult,
    recoveryActions,
    error: success
      ? undefined
      : `Build incomplete: ${failures.length} failure(s), lock satisfied=${isBuilderProbeLockSatisfied(workerResults, dispatches)}`,
  };
}

// ---------------------------------------------------------------------------
// Recommendation builder
// ---------------------------------------------------------------------------

/**
 * Build a Queen recommendation from the derived pattern and dispatches.
 *
 * @param pattern - Derived workflow pattern
 * @param dispatches - Build dispatches
 * @returns Queen recommendation
 */
function buildRecommendation(
  pattern: ReturnType<typeof deriveWorkflowPattern>,
  dispatches: BuildDispatch[]
): QueenRecommendation {
  const castes = new Set(dispatches.map((d) => d.caste));

  switch (pattern) {
    case "Deep Research":
      return {
        review_depth: "standard",
        reason: "Oracle-led research requires thorough documentation review",
      };
    case "Investigate-Fix":
      return {
        review_depth: "final-review",
        reason: "Bug investigation demands heavy verification before merge",
      };
    case "Refactor":
      return {
        review_depth: "standard",
        reason: "Refactoring needs careful regression testing",
      };
    case "Compliance":
      return {
        review_depth: "final-review",
        reason: "Security and compliance work requires full gate review",
      };
    case "Documentation Sprint":
      return {
        review_depth: "fast",
        reason: "Documentation changes are low-risk and fast to verify",
      };
    case "SPBV":
    default: {
      const hasBuilder = castes.has("builder");
      const hasWatcher = castes.has("watcher");
      if (hasBuilder && hasWatcher) {
        return {
          review_depth: "standard",
          reason: "Standard build with builder and watcher coverage",
        };
      }
      return {
        review_depth: "fast",
        reason: "Light build with minimal worker coverage",
      };
    }
  }
}
