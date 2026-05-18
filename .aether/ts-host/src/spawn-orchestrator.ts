/**
 * Spawn orchestrator for the TypeScript orchestration host.
 *
 * Validates worker spawn claims against budget and depth limits, synthesizes
 * child worker dispatch objects for the wave orchestrator, and logs rejected
 * spawns to stderr. This is the policy engine that decides which spawn requests
 * are allowed and which are rejected.
 *
 * Satisfies SPAWN-02 (depth=2 max), SPAWN-03 (budget tracking), and
 * SPAWN-06 (fail-closed: over-budget spawns are skipped, never queued).
 */

import type {
  SpawnClaim,
  SpawnedWorker,
  BuildDispatch,
} from "./types.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** Options for creating a spawn orchestrator instance. */
export interface SpawnOrchestratorOptions {
  /** Path to the Go binary. Unused by the orchestrator itself but kept for future use. */
  goBinaryPath?: string;
  /** Working directory. Unused by the orchestrator itself but kept for future use. */
  cwd?: string;
  /** Maximum total workers allowed (max_workers from manifest QueenSpawnBudget). Default: 20. */
  totalBudget?: number;
  /** Workers already dispatched before this orchestrator was created. Default: 0. */
  consumedBudget?: number;
  /** Depth of the workers currently being dispatched. Default: 1. */
  currentDepth?: number;
}

/** Result of processing a batch of spawn claims. */
export interface SpawnProcessingResult {
  /** Claims that passed budget and depth checks, now SpawnedWorker entries. */
  accepted: SpawnedWorker[];
  /** Claims that were rejected, with a reason string. */
  rejected: { claim: SpawnClaim; reason: string }[];
}

/** Spawn orchestrator interface. */
export interface SpawnOrchestrator {
  /** Read-only: how many worker slots remain in the budget. */
  readonly remainingBudget: number;
  /** Read-only: total budget capacity. */
  readonly totalBudget: number;
  /** Read-only: how many workers have been consumed so far. */
  readonly consumedBudget: number;
  /**
   * Process a batch of spawn claims from a parent worker.
   *
   * Validates each claim against depth and budget limits. Accepted claims
   * are converted to SpawnedWorker entries. Rejected claims are logged to
   * stderr with a generic reason.
   *
   * @param parentName - Name of the parent worker issuing the claims
   * @param parentDepth - Depth of the parent worker
   * @param claims - Array of spawn claims to process
   * @returns Accepted and rejected claims
   */
  processClaims(
    parentName: string,
    parentDepth: number,
    claims: SpawnClaim[]
  ): SpawnProcessingResult;
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

/** Maximum spawn depth. Workers at depth >= 2 cannot spawn children. */
const MAX_SPAWN_DEPTH = 2;

/** Default total budget when not specified. */
const DEFAULT_TOTAL_BUDGET = 20;

// ---------------------------------------------------------------------------
// Factory
// ---------------------------------------------------------------------------

/**
 * Create a new SpawnOrchestrator with the given options.
 *
 * The orchestrator is stateful: each call to processClaims consumes budget
 * from the remaining pool. Callers should create one orchestrator per build
 * and pass it through the wave execution pipeline.
 *
 * @param opts - Orchestrator configuration
 * @returns A SpawnOrchestrator instance
 */
export function createSpawnOrchestrator(
  opts: SpawnOrchestratorOptions = {}
): SpawnOrchestrator {
  const totalBudget = opts.totalBudget ?? DEFAULT_TOTAL_BUDGET;
  let consumedBudget = opts.consumedBudget ?? 0;

  return {
    get remainingBudget(): number {
      return totalBudget - consumedBudget;
    },
    get totalBudget(): number {
      return totalBudget;
    },
    get consumedBudget(): number {
      return consumedBudget;
    },

    processClaims(
      parentName: string,
      parentDepth: number,
      claims: SpawnClaim[]
    ): SpawnProcessingResult {
      // Guard against undefined/null claims
      const safeClaims = Array.isArray(claims) ? claims : [];

      const accepted: SpawnedWorker[] = [];
      const rejected: { claim: SpawnClaim; reason: string }[] = [];

      for (let i = 0; i < safeClaims.length; i++) {
        const claim = safeClaims[i]!;

        // SPAWN-02: depth check
        if (parentDepth >= MAX_SPAWN_DEPTH) {
          const reason = "max spawn depth exceeded (limit: 2)";
          rejected.push({ claim, reason });
          process.stderr.write(
            `Spawn rejected for ${parentName}: ${reason}\n`
          );
          continue;
        }

        // SPAWN-06: budget check (fail-closed)
        if (accepted.length >= (totalBudget - consumedBudget)) {
          const reason = "spawn budget exhausted";
          rejected.push({ claim, reason });
          process.stderr.write(
            `Spawn rejected for ${parentName}: ${reason}\n`
          );
          continue;
        }

        // Both checks passed: synthesize the child worker
        const childDepth = parentDepth + 1;
        const childName = `${parentName}-spawn-${accepted.length}`;
        const spawnedWorker: SpawnedWorker = {
          claim,
          name: childName,
          parent: parentName,
          depth: childDepth,
          status: "pending",
        };
        accepted.push(spawnedWorker);
      }

      // Update consumed budget by accepted count
      consumedBudget += accepted.length;

      return { accepted, rejected };
    },
  };
}

// ---------------------------------------------------------------------------
// Child dispatch synthesis
// ---------------------------------------------------------------------------

/**
 * Synthesize a BuildDispatch object from a spawn claim for child worker execution.
 *
 * The synthesized dispatch can be passed to the wave orchestrator for actual
 * worker dispatch. The child's name follows the pattern
 * `${parentName}-spawn-${index}` (e.g., "Builder-01-spawn-0").
 *
 * @param claim - The validated spawn claim
 * @param parentName - Parent worker name
 * @param depth - Spawn depth of the child worker
 * @param index - Index of this child among siblings
 * @returns A BuildDispatch-like object for the child worker
 */
export function synthesizeChildDispatch(
  claim: SpawnClaim,
  parentName: string,
  depth: number,
  index: number
): BuildDispatch {
  return {
    name: `${parentName}-spawn-${index}`,
    caste: claim.caste,
    task: claim.task,
    status: "pending",
    stage: "spawned",
    // BuildDispatch doesn't have depth/parent fields natively, but the
    // dispatch pipeline can read them from the SpawnedWorker that wraps it.
    // We store contextual info in stage for traceability.
    wave: depth,
  };
}
