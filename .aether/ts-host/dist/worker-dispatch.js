/**
 * Worker dispatch module for the TypeScript orchestration host.
 *
 * Iterates over Go manifest dispatches, records spawn-log before each worker,
 * dispatches the worker (simulated or real), and records spawn-complete
 * after. Restores the visible worker activity lost in the Bash-to-Go migration.
 *
 * When simulateWorkers is false, delegates real worker execution to Go's
 * typed platform adapter boundary. The host coordinates waves but never
 * selects or launches provider CLIs itself.
 *
 * Satisfies HOST-03 (visible dispatch from manifest) and HOST-06 (spawn
 * lifecycle events via Go CLI).
 */
import { callGoJSON, callGoJSONAsync, cleanupCompletionDir, writeCompletionFile, } from "./go-bridge.js";
import { normalizeSpawnClaims } from "./claims-parser.js";
import { dispatchWaves } from "./wave-orchestrator.js";
import { resolvePreflightTimeoutMs } from "./preflight-config.js";
// ---------------------------------------------------------------------------
// Single worker dispatch
// ---------------------------------------------------------------------------
/**
 * Dispatch a single worker from a manifest dispatch entry.
 *
 * Lifecycle:
 * 1. Call `aether spawn-log` to record the spawn before dispatch.
 * 2. Execute the worker (simulated or real).
 * 3. Call `aether spawn-complete` to record the outcome.
 *
 * Spawn-log failure does not block dispatch. Spawn-complete is always
 * attempted, even on dispatch error.
 *
 * @param opts - Dispatch options including Go binary path and cwd
 * @param dispatch - Build dispatch entry from the Go manifest
 * @returns Dispatch result with name, status, and summary
 */
export async function dispatchSingleWorker(opts, dispatch) {
    // NOTE: This function is only called from dispatchWaves with manifest
    // dispatches (from the Go build manifest). spawn-log/spawn-complete
    // therefore only record manifest workers, never internal/system workers.
    // SPAWN-05: Child spawns record actual parent worker name in spawn tree.
    // Manifest workers (no parent field) log parent="Queen", depth=1.
    // Spawned workers (have parent field) log parent="Builder-01", depth=2.
    const parent = dispatch.parent || "Queen";
    const spawnDepth = String(dispatch.depth || 1);
    // Step 1: Record spawn-log before dispatch.
    // Spawn-log failure is logged but does not block dispatch.
    try {
        const logResult = callGoJSON(opts, [
            "spawn-log",
            "--parent", parent,
            "--caste", dispatch.caste,
            "--name", dispatch.name,
            "--task", dispatch.task,
            "--depth", spawnDepth,
        ]);
        if (!logResult.recorded) {
            process.stderr.write(`Warning: spawn-log for ${dispatch.name} returned recorded=false\n`);
        }
    }
    catch (err) {
        const msg = err instanceof Error ? err.message : String(err);
        process.stderr.write(`Warning: spawn-log failed for ${dispatch.name}: ${msg}\n`);
    }
    // Step 2: Execute the worker.
    let result;
    // Default to real execution; simulation only with explicit opt-in.
    const simulate = opts.simulateWorkers === true;
    if (simulate) {
        process.stderr.write(`ℹ️  Simulating worker ${dispatch.name} (simulateWorkers=true)\n`);
    }
    try {
        if (simulate) {
            // Simulated worker execution: brief delay to mimic work.
            await new Promise((resolve) => setTimeout(resolve, 100));
            // Use configurable simulated file claims, or empty if not provided.
            // The Go build-finalizer validates that all file claims exist in the
            // repo, so simulated claims must reference real files.
            const simClaims = opts.simulatedFileClaims ?? [];
            result = {
                name: dispatch.name,
                status: "completed",
                summary: `Simulated worker completion for ${dispatch.name}`,
                duration: 0.1,
            };
            if (simClaims.length > 0) {
                result.files_modified = [simClaims[0]];
            }
            if (simClaims.length > 1) {
                result.tests_written = [simClaims[1]];
            }
        }
        else {
            // Real worker dispatch through the Go-owned platform adapter.
            result = await dispatchRealWorker(opts, dispatch);
        }
    }
    catch (dispatchErr) {
        const errMsg = dispatchErr instanceof Error ? dispatchErr.message : String(dispatchErr);
        // Auth/config errors halt the build immediately (D-01).
        if (isAuthError(dispatchErr)) {
            throw new Error(`Worker dispatch halted: ${sanitizeWorkerDiagnosticOutput(errMsg)}`);
        }
        // Timeout/transient errors mark the worker as failed and continue (D-02).
        result = {
            name: dispatch.name,
            status: "failed",
            summary: `Worker dispatch failed: ${sanitizeWorkerDiagnosticOutput(errMsg)}`,
        };
    }
    // Step 3: Record spawn-complete after dispatch.
    // Always attempt spawn-complete, even if the worker failed.
    try {
        const completeResult = callGoJSON(opts, [
            "spawn-complete",
            "--name", result.name,
            "--status", result.status,
            "--summary", result.summary,
        ]);
        if (!completeResult.completed) {
            process.stderr.write(`Warning: spawn-complete for ${result.name} returned completed=false\n`);
        }
    }
    catch (err) {
        const msg = err instanceof Error ? err.message : String(err);
        process.stderr.write(`Warning: spawn-complete failed for ${result.name}: ${msg}\n`);
    }
    return result;
}
// ---------------------------------------------------------------------------
// Real worker dispatch
// ---------------------------------------------------------------------------
/**
 * Dispatch a real worker through Go's typed provider adapter.
 *
 * Go owns platform selection, agent validation, provider invocation, claims
 * parsing, and the versioned platform capability contract. TypeScript owns
 * only the temporary request file and mapping the typed response into the
 * host's wave result.
 *
 * @param opts - Dispatch options
 * @param dispatch - Build dispatch entry
 * @returns Dispatch result from real worker claims
 */
async function dispatchRealWorker(opts, dispatch) {
    const timeoutMs = opts.timeoutMs ?? 600_000;
    const request = {
        schema_version: 1,
        caste: dispatch.caste,
        worker_name: dispatch.name,
        task_id: dispatch.task_id ?? dispatch.name,
        task: dispatch.task,
        task_brief: dispatch.task_brief ?? dispatch.task,
        timeout_ms: timeoutMs,
        permission_profile: dispatch.permission_profile ?? permissionProfileForCaste(dispatch.caste),
    };
    if (opts.workflow !== undefined)
        request.workflow = opts.workflow;
    if (opts.phase !== undefined)
        request.phase = opts.phase;
    if (opts.executionBinding !== undefined)
        request.execution_binding = opts.executionBinding;
    if (dispatch.context_capsule !== undefined)
        request.context_capsule = dispatch.context_capsule;
    if (dispatch.skill_section !== undefined)
        request.skill_section = dispatch.skill_section;
    if (dispatch.pheromone_section !== undefined)
        request.pheromone_section = dispatch.pheromone_section;
    if (dispatch.handoff_section !== undefined)
        request.handoff_section = dispatch.handoff_section;
    const requestPath = writeCompletionFile("aether-worker-request", "worker-request.json", request);
    let response;
    try {
        response = await callGoJSONAsync(opts, ["internal-worker-adapter", "--request-file", requestPath], timeoutMs + 30_000);
    }
    finally {
        cleanupCompletionDir(requestPath);
    }
    if (response.execution_owner !== "go-adapter" || response.schema_version !== 1) {
        throw new Error("Go worker adapter returned an unsupported ownership contract");
    }
    if (opts.workflow === "build") {
        if (!opts.executionBinding || !response.execution_binding) {
            throw new Error("Go worker adapter returned no durable build execution binding");
        }
        if (!sameExecutionBinding(opts.executionBinding, response.execution_binding)) {
            throw new Error("Go worker adapter returned a result for a different build run");
        }
    }
    const worker = response.worker;
    if (!worker) {
        throw new Error("Go worker adapter returned no terminal worker result");
    }
    if (worker.name !== dispatch.name || worker.caste.toLowerCase() !== dispatch.caste.toLowerCase()) {
        throw new Error("Go worker adapter returned a result for a different worker");
    }
    const status = normalizeTerminalStatus(worker.status);
    const result = {
        name: dispatch.name,
        status,
        summary: worker.summary?.trim() ||
            worker.error?.trim() ||
            `Worker ${dispatch.name} returned ${status}`,
        detectedPlatform: response.platform,
        permission_decision: response.permission_decision,
    };
    if (response.execution_binding !== undefined)
        result.execution_binding = response.execution_binding;
    if (response.provider_run_id !== undefined)
        result.provider_run_id = response.provider_run_id;
    if (worker.duration !== undefined)
        result.duration = worker.duration;
    if (worker.files_created !== undefined) {
        result.files_created = worker.files_created;
    }
    if (worker.files_modified !== undefined) {
        result.files_modified = worker.files_modified;
    }
    if (worker.tests_written !== undefined) {
        result.tests_written = worker.tests_written;
    }
    if (worker.spawns !== undefined && worker.spawns.length > 0) {
        result.spawns = normalizeSpawnClaims(worker.spawns);
    }
    if (worker.handoff !== undefined)
        result.handoff = worker.handoff;
    if (worker.artifacts !== undefined) {
        result.artifacts = worker.artifacts;
        if (worker.artifacts["phase_plan"] !== undefined) {
            result.phase_plan = worker.artifacts["phase_plan"];
        }
        if (worker.scout_report === undefined && worker.artifacts["scout_report"] !== undefined) {
            result.scout_report = worker.artifacts["scout_report"];
        }
    }
    if (worker.scout_report !== undefined)
        result.scout_report = worker.scout_report;
    if (worker.tool_count !== undefined)
        result.tool_count = worker.tool_count;
    if (worker.blockers !== undefined)
        result.blockers = worker.blockers;
    return result;
}
// Go remains authoritative and rejects stale or broadened values. This fallback
// covers dynamically spawned children that are not present in a Go manifest.
// The read-only predicate mirrors `repositoryReadOnlyCastes` in
// pkg/codex/permission_profile.go exactly (currently just "includer") --
// Go's ResolvePermissionProfile rejects any mismatch outright rather than
// downgrading, so this fallback must never grant a broader or narrower
// profile than the canonical Go map. Exported so
// test/dispatch-field-fidelity.test.ts can assert against it directly and
// catch drift the next time Go's read-only set changes.
export function permissionProfileForCaste(caste) {
    const normalized = caste.trim().toLowerCase().replace(/^aether-/, "").replaceAll("-", "_");
    const readOnly = normalized === "includer";
    const filesystem = readOnly ? "repository_read_only" : "workspace_write";
    return {
        schema_version: 1,
        name: filesystem,
        filesystem,
        shell: "within_filesystem_boundary",
        network: "provider_default",
        approval: "never",
    };
}
function sameExecutionBinding(left, right) {
    return left.schema_version === right.schema_version &&
        left.run_id === right.run_id &&
        left.attempt_id === right.attempt_id &&
        left.manifest_sha256 === right.manifest_sha256 &&
        left.workspace_fingerprint === right.workspace_fingerprint &&
        left.execution_owner === right.execution_owner;
}
function normalizeTerminalStatus(value) {
    const status = value.trim().toLowerCase();
    switch (status) {
        case "completed":
        case "failed":
        case "blocked":
        case "timeout":
            return status;
        default:
            throw new Error(`Go worker adapter returned invalid terminal status "${value}"`);
    }
}
/**
 * Mirror of hostedPreflightAttempts in pkg/codex/platform_dispatch.go.
 * Pinned by TestHostsAgreeOnPreflightRetryAttempts in
 * cmd/preflight_docs_test.go — change one side without the other and that
 * test fails.
 */
export const PREFLIGHT_GO_ATTEMPTS = 2;
/** Startup slack added on top of the Go side's worst-case retry budget. */
const PREFLIGHT_ADAPTER_SLACK_MS = 30_000;
/**
 * Node-side kill budget for the `internal-worker-adapter --preflight` call.
 *
 * Must exceed the Go side's full preflight budget — the resolved
 * AETHER_PREFLIGHT_TIMEOUT (not the 45s constant: the knob is configurable)
 * times hostedPreflightAttempts — plus startup slack. At a hardcoded 30s
 * Node SIGTERM'd the adapter before the Go retry could ever fire (27 July
 * incident); a hardcoded 120s reintroduced the same failure for any
 * AETHER_PREFLIGHT_TIMEOUT above ~45s, killing the adapter mid-retry.
 * Floored at 120s so the wrapper never gets tighter than the old constant.
 */
export function resolvePreflightAdapterBudgetMs() {
    return Math.max(120_000, resolvePreflightTimeoutMs() * PREFLIGHT_GO_ATTEMPTS + PREFLIGHT_ADAPTER_SLACK_MS);
}
/** Ask the Go-owned adapter layer to select and preflight the worker provider. */
export async function preflightGoWorkerProvider(opts, context) {
    try {
        const response = await callGoJSONAsync(opts, ["internal-worker-adapter", "--preflight"], resolvePreflightAdapterBudgetMs());
        if (response.preflight?.notice && response.preflight.notice.trim()) {
            process.stderr.write(`${response.preflight.notice}\n`);
        }
        return response;
    }
    catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        throw new Error(`${context} cannot start: ${sanitizeWorkerDiagnosticOutput(message)}`);
    }
}
export function sanitizeWorkerDiagnosticOutput(value) {
    let output = stripAnsi(value.trim());
    if (!output) {
        return "";
    }
    output = output.replace(/\b(token|api[_-]?key|secret|password|authorization)\b\s*[:=]\s*["']?[^"'\s;]+/gi, (_match, key) => `${key}=[redacted]`);
    return output
        .replace(/sk-[A-Za-z0-9_-]+/g, "[redacted]")
        .replace(/github_pat_[A-Za-z0-9_]+/g, "[redacted]")
        .replace(/gh[pousr]_[A-Za-z0-9_]+/g, "[redacted]")
        .replace(/npm_[A-Za-z0-9_]+/g, "[redacted]")
        .replace(/\b[A-Za-z0-9._-]*secret[A-Za-z0-9._-]*\b/gi, "[redacted]");
}
/**
 * Check if an error is classified as an authentication error.
 *
 * Auth errors should halt the build immediately (D-01).
 * The build pipeline (Plan 03+) uses this to decide whether to halt.
 *
 * @param error - The error to check
 * @returns true if the error is an auth/config error
 */
export function isAuthError(error) {
    const message = error instanceof Error ? error.message : String(error);
    return /\b(auth(?:entication|orization)?|credentials?|login|permission denied|api[_ -]?key)\b/i.test(message);
}
function stripAnsi(value) {
    return value.replace(/\x1b\[[0-9;]*m/g, "");
}
// ---------------------------------------------------------------------------
// Multi-worker dispatch
// ---------------------------------------------------------------------------
/**
 * Dispatch multiple workers from a manifest, grouped by wave.
 *
 * Waves are processed sequentially. Within each wave, workers are
 * dispatched in parallel by default (parallel=true). Delegates to
 * wave-orchestrator.ts for wave grouping and parallel dispatch.
 *
 * @param opts - Dispatch options including Go binary path and cwd
 * @param dispatches - Array of build dispatch entries from the manifest
 * @returns Array of dispatch results, one per input dispatch
 */
export async function dispatchWorkers(opts, dispatches) {
    const waveResults = await dispatchWaves(opts, dispatches);
    // Log wave summaries to stderr with per-worker failure details (D-02).
    for (const wr of waveResults) {
        const succeeded = wr.results.length - wr.failures.length;
        const failed = wr.failures.length;
        if (failed > 0) {
            const failureNames = wr.failures
                .map((f) => `${f.name}: ${f.status}`)
                .join(", ");
            process.stderr.write(`Wave ${wr.wave}: ${succeeded} succeeded, ${failed} failed (${failureNames})\n`);
        }
        else {
            process.stderr.write(`Wave ${wr.wave}: ${succeeded}/${wr.results.length} succeeded, ${wr.retried} retries\n`);
        }
    }
    // Flatten WaveResult array back to DispatchResult array
    const results = [];
    for (const wr of waveResults) {
        results.push(...wr.results);
    }
    return results;
}
// ---------------------------------------------------------------------------
// Result mapping
// ---------------------------------------------------------------------------
/**
 * Map dispatch entries and their results to WorkerResult objects for
 * the Go build finalizer.
 *
 * Matches results to dispatches by name (not by index) to handle
 * wave-grouped re-ordering from dispatchWorkers.
 *
 * Preserves manifest fields (caste, task_id, stage, wave) alongside
 * dispatch outcomes (status, summary).
 *
 * @param dispatches - Original build dispatch entries from the manifest
 * @param results - Dispatch results from dispatchWorkers
 * @returns WorkerResult array suitable for the Go finalizer completion file
 */
export function toWorkerResults(dispatches, results) {
    // Build a name -> result lookup for O(1) matching
    const resultByName = new Map();
    for (const r of results) {
        resultByName.set(r.name, r);
    }
    return dispatches.map((dispatch) => {
        const result = resultByName.get(dispatch.name);
        // Build result object; omit optional fields when undefined for
        // exactOptionalPropertyTypes compatibility.
        const status = result?.status ?? "timeout";
        const summary = result?.summary ??
            `No terminal worker result was provided for ${dispatch.name}; treating as timeout.`;
        const worker = {
            name: dispatch.name,
            status,
            summary,
            caste: dispatch.caste,
            task: dispatch.task,
            stage: dispatch.stage,
        };
        if (dispatch.task_id !== undefined)
            worker.task_id = dispatch.task_id;
        if (dispatch.wave !== undefined)
            worker.wave = dispatch.wave;
        if (dispatch.execution_wave !== undefined)
            worker.execution_wave = dispatch.execution_wave;
        if (result?.duration !== undefined)
            worker.duration = result.duration;
        if (result?.files_modified !== undefined)
            worker.files_modified = result.files_modified;
        if (result?.files_created !== undefined)
            worker.files_created = result.files_created;
        if (result?.tests_written !== undefined)
            worker.tests_written = result.tests_written;
        if (result?.blockers !== undefined)
            worker.blockers = result.blockers;
        if (result?.tool_count !== undefined)
            worker.tool_count = result.tool_count;
        if (result?.spawns !== undefined)
            worker.spawns = result.spawns;
        if (result?.handoff !== undefined)
            worker.handoff = result.handoff;
        if (result?.artifacts !== undefined)
            worker.artifacts = result.artifacts;
        if (result?.scout_report !== undefined) {
            worker.scout_report = result.scout_report;
        }
        if (result?.phase_plan !== undefined) {
            worker.phase_plan = result.phase_plan;
        }
        return worker;
    });
}
