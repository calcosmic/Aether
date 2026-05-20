/**
 * Wave orchestrator for the TypeScript orchestration host.
 *
 * Groups workers by wave, dispatches them in parallel within each wave,
 * and retries failed workers with exponential backoff.
 *
 * Satisfies TS-02 (concurrent wave dispatch) and TS-03 (retry, timeout,
 * graceful error handling).
 */
import { dispatchSingleWorker, } from "./worker-dispatch.js";
// Mutable reference for test injection.
let _dispatchSingleWorker = dispatchSingleWorker;
// ---------------------------------------------------------------------------
// Retry dispatch
// ---------------------------------------------------------------------------
/**
 * Dispatch a single worker with retry logic and exponential backoff.
 *
 * @param opts - Wave orchestrator options
 * @param dispatch - Build dispatch entry
 * @param attempt - Current attempt number (starts at 1)
 * @returns Dispatch result after retries exhausted or success
 */
export async function retryDispatch(opts, dispatch, attempt = 1) {
    const result = await _dispatchSingleWorker(opts, dispatch);
    const retryLimit = opts.retryLimit ?? 1;
    if (result.status === "failed" && attempt < retryLimit) {
        const delay = (opts.retryDelayMs ?? 5000) * attempt;
        process.stderr.write(`Retrying ${dispatch.name} (attempt ${attempt + 1}/${retryLimit}) after ${delay}ms\n`);
        await new Promise((resolve) => setTimeout(resolve, delay));
        return retryDispatch(opts, dispatch, attempt + 1);
    }
    return result;
}
/** Internal helper used by dispatchWave to retry a single dispatch. */
async function runRetryLoop(opts, dispatch) {
    let result = await _dispatchSingleWorker(opts, dispatch);
    let retried = 0;
    const retryLimit = opts.retryLimit ?? 1;
    while (result.status === "failed" && retried < retryLimit - 1) {
        retried++;
        const delay = (opts.retryDelayMs ?? 5000) * retried;
        process.stderr.write(`Retrying ${dispatch.name} (attempt ${retried + 1}/${retryLimit}) after ${delay}ms\n`);
        await new Promise((resolve) => setTimeout(resolve, delay));
        result = await _dispatchSingleWorker(opts, dispatch);
    }
    return { result, retried };
}
// ---------------------------------------------------------------------------
// Wave dispatch
// ---------------------------------------------------------------------------
/**
 * Dispatch all workers in a single wave.
 *
 * Workers are dispatched in parallel when opts.parallel is true and there
 * are multiple dispatches. Otherwise they run sequentially.
 *
 * Each worker is wrapped with retry logic via retryDispatch.
 *
 * @param opts - Wave orchestrator options
 * @param dispatches - Array of build dispatch entries for this wave
 * @returns Wave result with results, failures, and retry count
 */
export async function dispatchWave(opts, dispatches) {
    const wave = dispatches[0]?.wave ?? dispatches[0]?.execution_wave ?? 0;
    process.stderr.write(`Dispatching wave ${wave}: ${dispatches.length} workers\n`);
    let retried = 0;
    const runWithRetry = async (d) => {
        const { result, retried: r } = await runRetryLoop(opts, d);
        retried += r;
        return result;
    };
    let results;
    if (opts.parallel !== false && dispatches.length > 1) {
        results = await Promise.all(dispatches.map((d) => runWithRetry(d)));
    }
    else {
        results = [];
        for (const d of dispatches) {
            results.push(await runWithRetry(d));
        }
    }
    const failures = results.filter((r) => r.status === "failed");
    process.stderr.write(`Wave ${wave} complete (${results.length - failures.length}/${results.length} succeeded, ${retried} retries)\n`);
    return { wave, results, failures, retried };
}
// ---------------------------------------------------------------------------
// Multi-wave dispatch
// ---------------------------------------------------------------------------
/** Test-only: inject a mock dispatchSingleWorker. */
export function __setDispatchSingleWorker(fn) {
    _dispatchSingleWorker = fn;
}
/** Test-only: restore the real dispatchSingleWorker. */
export function __restoreDispatchSingleWorker() {
    _dispatchSingleWorker = dispatchSingleWorker;
}
// ---------------------------------------------------------------------------
// Post-wave spawn processing (SPAWN-01, SPAWN-04)
// ---------------------------------------------------------------------------
/**
 * Process spawn claims from a completed wave.
 *
 * After a wave of workers completes, collect any spawn claims from their
 * results, validate them through the spawn orchestrator, dispatch accepted
 * child workers in a follow-up spawn wave, and attach child results to the
 * parent workers' handoffs.
 *
 * Child workers dispatched via spawn waves do not themselves trigger further
 * spawn processing in this version (prevents recursive complexity). The
 * spawnOrchestrator depth limit already prevents grandchildren.
 *
 * @param opts - Wave orchestrator options (must include spawnOrchestrator)
 * @param waveResult - The completed wave result to process spawns from
 * @returns Child wave result if children were dispatched, undefined otherwise
 */
async function processWaveSpawns(opts, waveResult) {
    const orchestrator = opts.spawnOrchestrator;
    if (!orchestrator)
        return undefined;
    // Collect spawn claims from all completed workers in the wave
    const allClaims = [];
    for (const result of waveResult.results) {
        if (result.status === "completed" && result.spawns && result.spawns.length > 0) {
            allClaims.push({
                parent: result.name,
                depth: 1, // Manifest workers are depth 1
                claims: result.spawns,
            });
        }
    }
    if (allClaims.length === 0)
        return undefined;
    // Process claims through orchestrator
    const allAccepted = [];
    const allRejected = [];
    for (const { parent, depth, claims } of allClaims) {
        const { accepted, rejected } = orchestrator.processClaims(parent, depth, claims);
        allAccepted.push(...accepted);
        allRejected.push(...rejected);
    }
    if (allAccepted.length === 0)
        return undefined;
    // Synthesize child dispatches
    const childDispatches = allAccepted.map((sw) => ({
        stage: "spawned",
        caste: sw.claim.caste,
        name: sw.name,
        task: sw.claim.task,
        status: "pending",
        // Mark as spawned worker with parent context for spawn-log (SPAWN-05)
        parent: sw.parent,
        depth: sw.depth,
    }));
    // Dispatch child workers as a spawn wave
    process.stderr.write(`Dispatching spawn wave: ${childDispatches.length} child workers\n`);
    const childWaveResult = await dispatchWave(opts, childDispatches);
    // Attach child results to parent handoffs (SPAWN-04)
    for (const childResult of childWaveResult.results) {
        const parentName = childResult.name.split("-spawn-")[0];
        if (!parentName)
            continue;
        const parentWorker = waveResult.results.find((r) => r.name === parentName);
        if (parentWorker) {
            if (!parentWorker.handoff)
                parentWorker.handoff = {};
            if (!parentWorker.handoff.child_results)
                parentWorker.handoff.child_results = [];
            parentWorker.handoff.child_results.push({
                name: childResult.name,
                status: childResult.status,
                summary: childResult.summary,
            });
        }
    }
    return childWaveResult;
}
/**
 * Dispatch multiple workers grouped by wave.
 *
 * Waves run sequentially (wave 1 must complete before wave 2 starts).
 * Within each wave, workers run in parallel by default.
 *
 * When a spawnOrchestrator is provided, after each wave completes, spawn
 * claims from completed workers are processed and accepted children are
 * dispatched in a follow-up spawn wave.
 *
 * @param opts - Wave orchestrator options
 * @param dispatches - Array of build dispatch entries from the manifest
 * @returns Array of wave results, one per wave (including spawn waves)
 */
export async function dispatchWaves(opts, dispatches) {
    // Group dispatches by wave number.
    const waveMap = new Map();
    for (const d of dispatches) {
        const wave = d.wave ?? d.execution_wave ?? 0;
        const existing = waveMap.get(wave);
        if (existing) {
            existing.push(d);
        }
        else {
            waveMap.set(wave, [d]);
        }
    }
    // Sort waves ascending.
    const sortedWaves = Array.from(waveMap.keys()).sort((a, b) => a - b);
    const waveResults = [];
    for (const waveNum of sortedWaves) {
        const waveDispatches = waveMap.get(waveNum);
        const result = await dispatchWave(opts, waveDispatches);
        waveResults.push(result);
        // Process spawns from this wave (SPAWN-01, SPAWN-04)
        const spawnResult = await processWaveSpawns(opts, result);
        if (spawnResult) {
            waveResults.push(spawnResult);
        }
    }
    return waveResults;
}
