/**
 * Spawn orchestrator for the TypeScript orchestration host.
 *
 * Bridges worker spawn claims to the ONE Go admission chokepoint
 * (spawnCanSpawnDecision, exposed via `aether spawn-can-spawn`) instead of
 * deciding depth/budget itself. Before 203-09 this file held its own budget
 * total, consumed count, and depth constant -- a second, uncoordinated
 * admission authority running alongside the Go safety kernel on the
 * autopilot/host lane. That arithmetic is deleted here, not merely bypassed
 * (SYN-203-02): every claim is decided by the same Go ledger the interactive
 * `aether recruit` lane already consults.
 *
 * CR-01 fix (203-REVIEW.md): every call also passes `--recruitment`, so the
 * Go side decides under spawnOriginRecruit -- the SAME origin, and the SAME
 * five extra admission dimensions (parent authority, permission, path,
 * cost, duplicate) the in-repo build lane and `aether recruit` already
 * apply -- instead of the bare depth/budget/ancestor-cycle check
 * `spawn-can-spawn` applies without that flag. Before this flag existed, a
 * host-lane recruitment and a native-lane recruitment against the same
 * ledger state could disagree: a read-only caste requesting a write
 * workspace was refused via `aether recruit` and silently admitted here.
 * Now both lanes decide through recruitmentClaimAdmission's one code path
 * (cmd/recruitment_admission.go), so a host-lane recruitment and a
 * native-lane recruitment against the same spawn-tree and
 * recruitment-intent state produce the same allow/deny answer, with the
 * same reason vocabulary.
 *
 * A bridge call that fails, times out, or returns an unparseable envelope
 * denies the claim and names the bridge failure -- it never falls back to
 * local admission arithmetic (the same fail-closed discipline the Go side's
 * own unreadable-state branches already follow).
 */
import type { SpawnClaim, SpawnedWorker, BuildDispatch } from "./types.js";
/** Options for creating a spawn orchestrator instance. */
export interface SpawnOrchestratorOptions {
    /** Path to the Go binary this orchestrator asks for every admission decision. */
    goBinaryPath: string;
    /** Working directory for the Go subprocess (also used as the declared workspace). */
    cwd: string;
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
    /**
     * Process a batch of spawn claims from a parent worker.
     *
     * Calls the Go admission gate (`aether spawn-can-spawn --recruitment`)
     * once per claim, passing the parent name, the parent depth, the
     * requested caste, the objective and the declared workspace.
     * `--recruitment` (CR-01 fix, 203-REVIEW.md) asks the gate to decide
     * under spawnOriginRecruit -- the same origin and the same five
     * admission dimensions `aether recruit` and the in-repo build lane
     * already apply, not the bare depth/budget check this command applies
     * without the flag. The gate's own `allowed` (as `can_spawn`), `reason`
     * and `detail` fields are returned unchanged -- this orchestrator
     * computes no admission decision of its own.
     *
     * @param parentName - Name of the parent worker issuing the claims
     * @param parentDepth - Depth of the parent worker
     * @param claims - Array of spawn claims to process
     * @returns Accepted and rejected claims
     */
    processClaims(parentName: string, parentDepth: number, claims: SpawnClaim[]): SpawnProcessingResult;
}
/**
 * Create a new SpawnOrchestrator bridging every admission decision to the
 * Go binary's `spawn-can-spawn` command -- the same chokepoint
 * (spawnCanSpawnDecision) the interactive `aether recruit` lane and every
 * ordinary spawn already use. This orchestrator holds no budget total, no
 * consumed count and no depth constant of its own.
 *
 * @param opts - Orchestrator configuration (Go binary path and cwd)
 * @returns A SpawnOrchestrator instance
 */
export declare function createSpawnOrchestrator(opts: SpawnOrchestratorOptions): SpawnOrchestrator;
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
