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
import type { SpawnClaim, SpawnedWorker, BuildDispatch } from "./types.js";
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
    rejected: {
        claim: SpawnClaim;
        reason: string;
    }[];
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
    processClaims(parentName: string, parentDepth: number, claims: SpawnClaim[]): SpawnProcessingResult;
}
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
export declare function createSpawnOrchestrator(opts?: SpawnOrchestratorOptions): SpawnOrchestrator;
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
export declare function synthesizeChildDispatch(claim: SpawnClaim, parentName: string, depth: number, index: number): BuildDispatch;
