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
import { callGoJSON } from "./go-bridge.js";
// ---------------------------------------------------------------------------
// Factory
// ---------------------------------------------------------------------------
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
export function createSpawnOrchestrator(opts) {
    const bridge = { goBinaryPath: opts.goBinaryPath, cwd: opts.cwd };
    return {
        processClaims(parentName, parentDepth, claims) {
            // Guard against undefined/null claims
            const safeClaims = Array.isArray(claims) ? claims : [];
            const accepted = [];
            const rejected = [];
            for (let i = 0; i < safeClaims.length; i++) {
                const claim = safeClaims[i];
                let decision;
                try {
                    decision = callGoJSON(bridge, [
                        "spawn-can-spawn",
                        "--recruitment",
                        "--name", parentName,
                        "--depth", String(parentDepth),
                        "--caste", claim.caste,
                        "--task", claim.task,
                        "--workspace", opts.cwd,
                    ]);
                }
                catch (err) {
                    // Fail-closed: a bridge call that fails, times out, returns a
                    // non-zero exit, or returns an unparseable envelope denies the
                    // claim and names the bridge failure -- never a local
                    // recomputation of depth or budget.
                    const msg = err instanceof Error ? err.message : String(err);
                    const reason = `spawn admission gate unreachable: ${msg}`;
                    rejected.push({ claim, reason });
                    process.stderr.write(`Spawn rejected for ${parentName}: ${reason}\n`);
                    continue;
                }
                if (!decision || decision.can_spawn !== true) {
                    // The gate's own detail sentence, verbatim -- never a locally
                    // composed rejection string.
                    const reason = (decision && (decision.detail || decision.reason)) ||
                        "spawn denied by the admission gate";
                    rejected.push({ claim, reason });
                    process.stderr.write(`Spawn rejected for ${parentName}: ${reason}\n`);
                    continue;
                }
                // The gate admitted this claim: synthesize the child worker.
                const childDepth = parentDepth + 1;
                const childName = `${parentName}-spawn-${accepted.length}`;
                const spawnedWorker = {
                    claim,
                    name: childName,
                    parent: parentName,
                    depth: childDepth,
                    status: "pending",
                };
                accepted.push(spawnedWorker);
            }
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
export function synthesizeChildDispatch(claim, parentName, depth, index) {
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
