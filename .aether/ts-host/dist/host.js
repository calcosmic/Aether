/**
 * TypeScript orchestration host entry point.
 *
 * Invoked as: node .aether/ts-host/dist/host.js <command> [options]
 *
 * Commands:
 *   colonize   -- Call `aether colonize --plan-only` and print JSON manifest
 *   plan       -- Call `aether plan --plan-only` and print JSON manifest
 *   build <N>  -- Call `aether build N --plan-only` and print JSON manifest
 *   continue   -- Call `aether continue --plan-only` and print JSON manifest
 *   seal       -- Call `aether seal --plan-only` and print JSON manifest
 *   oracle     -- Run the Oracle lifecycle loop
 *   lifecycle  -- Run the experimental simulate-only lifecycle smoke harness
 *   watch      -- Show colony status through the host display surface
 *   swarm      -- Show or plan swarm activity through the host display surface
 *
 * Options:
 *   --cwd <path>  Working directory (default: process.cwd())
 */
import { callGoJSON, discoverGoBinary, writeCompletionFile, approvedCompletionDirPrefix, cleanupCompletionDir } from "./go-bridge.js";
import { buildHostGoArgs, getHostCommandDefinition, listHostCommandDefinitions, } from "./command-registry.js";
import { runGoJSONCommand } from "./go-command.js";
import { dispatchWorkers, toWorkerResults } from "./worker-dispatch.js";
import { detectAvailablePlatforms, formatPlatformUnavailableMessage } from "./platform-dispatcher.js";
import { createSpawnOrchestrator } from "./spawn-orchestrator.js";
import { createCeremonyAdapter, renderDryRunBadge, } from "./ceremony-adapter.js";
import { ConfidenceLoop } from "./confidence-loop.js";
import { ConfidenceEvaluator } from "./confidence-evaluator.js";
import { loadPlaybooksForWorkflow, renderPlaybookContext } from "./playbook-loader.js";
export { buildHostGoArgs } from "./command-registry.js";
// Mutable reference for test injection.
let _callGoJSONRef = callGoJSON;
/** Test-only: inject a mock callGoJSON. */
export function __setCallGoJSON(fn) {
    _callGoJSONRef = fn;
}
/** Test-only: restore the real callGoJSON. */
export function __restoreCallGoJSON() {
    _callGoJSONRef = callGoJSON;
}
// Mutable references for dispatch worker injection (testing).
let _dispatchWorkersRef = dispatchWorkers;
let _detectAvailablePlatformsRef = detectAvailablePlatforms;
/** Test-only: inject a mock dispatchWorkers. */
export function __setDispatchWorkers(fn) {
    _dispatchWorkersRef = fn;
}
/** Test-only: restore the real dispatchWorkers. */
export function __restoreDispatchWorkers() {
    _dispatchWorkersRef = dispatchWorkers;
}
/** Test-only: inject a mock detectAvailablePlatforms. */
export function __setDetectAvailablePlatforms(fn) {
    _detectAvailablePlatformsRef = fn;
}
/** Test-only: restore the real detectAvailablePlatforms. */
export function __restoreDetectAvailablePlatforms() {
    _detectAvailablePlatformsRef = detectAvailablePlatforms;
}
/** Restore all test mocks at once. */
export function __restoreAllMocks() {
    __restoreCallGoJSON();
    __restoreDispatchWorkers();
    __restoreDetectAvailablePlatforms();
}
// Test-only: exported runner functions for integration testing.
export { runDispatchedBuildCommand, runDispatchedPlanCommand, runDispatchedContinueCommand, runDryRunDispatchedCommand };
import { runLifecycle } from "./lifecycle.js";
import { runOracleLifecycle } from "./oracle-lifecycle.js";
import { runWatchDisplay } from "./watch-display.js";
import { runSwarmDisplay } from "./swarm-display.js";
import { createNarrator } from "./narrator.js";
import { startEventBridge } from "./event-bridge.js";
import { readHiveWisdom, resolveDomainTags } from "./hive-injector.js";
/** Parse command-line arguments for the TS host. */
export function parseArgs(argv) {
    const args = argv.slice(2); // skip node and script path
    let command = "";
    let cwd = process.cwd();
    let simulate = false;
    let dryRun = false;
    let synthetic = false;
    let noDashboard = false;
    let skipMiddenCheck = false;
    let skipWatchers = false;
    let refresh = false;
    let force = false;
    let forceResurvey = false;
    const tasks = [];
    let depth = undefined;
    let planningDepth = undefined;
    let verificationDepth = undefined;
    let targetConfidence = undefined;
    let maxIterations = undefined;
    let accept = false;
    let verificationTimeout = undefined;
    let light = false;
    let heavy = false;
    let workerTimeout = undefined;
    let circuitBreakerThreshold = undefined;
    let noSuggest = false;
    let verbose = false;
    const reconcileTasks = [];
    let noLearn = false;
    let classicCeremony = false;
    let help = false;
    const positional = [];
    const unknownFlags = [];
    for (let i = 0; i < args.length; i++) {
        let arg = args[i];
        const rawArg = arg;
        let inlineValue;
        if (arg.startsWith("--")) {
            const eq = arg.indexOf("=");
            if (eq > 2) {
                inlineValue = arg.slice(eq + 1);
                arg = arg.slice(0, eq);
            }
        }
        const nextValue = () => {
            if (inlineValue !== undefined)
                return inlineValue;
            if (i + 1 < args.length)
                return args[++i];
            return undefined;
        };
        const readValue = (flag) => {
            const value = nextValue();
            if (value === undefined) {
                unknownFlags.push(`${flag} requires a value`);
            }
            return value;
        };
        if (arg === "--cwd") {
            cwd = readValue(arg) ?? cwd;
        }
        else if (arg === "--simulate") {
            simulate = true;
        }
        else if (arg === "--dry-run") {
            dryRun = true;
        }
        else if (arg === "--synthetic") {
            synthetic = true;
        }
        else if (arg === "--no-dashboard") {
            noDashboard = true;
        }
        else if (arg === "--skip-midden-check") {
            skipMiddenCheck = true;
        }
        else if (arg === "--skip-watchers") {
            skipWatchers = true;
        }
        else if (arg === "--refresh") {
            refresh = true;
        }
        else if (arg === "--force") {
            force = true;
        }
        else if (arg === "--force-resurvey") {
            forceResurvey = true;
        }
        else if (arg === "--task") {
            const value = readValue(arg);
            if (value !== undefined)
                tasks.push(value);
        }
        else if (arg === "--depth") {
            depth = readValue(arg);
        }
        else if (arg === "--planning-depth") {
            planningDepth = readValue(arg);
        }
        else if (arg === "--verification-depth") {
            verificationDepth = readValue(arg);
        }
        else if (arg === "--target") {
            targetConfidence = readValue(arg);
        }
        else if (arg === "--max-iterations") {
            maxIterations = readValue(arg);
        }
        else if (arg === "--accept") {
            accept = true;
        }
        else if (arg === "--verification-timeout") {
            verificationTimeout = readValue(arg);
        }
        else if (arg === "--light") {
            light = true;
        }
        else if (arg === "--heavy") {
            heavy = true;
        }
        else if (arg === "--worker-timeout") {
            workerTimeout = readValue(arg);
        }
        else if (arg === "--circuit-breaker-threshold") {
            circuitBreakerThreshold = readValue(arg);
        }
        else if (arg === "--no-suggest") {
            noSuggest = true;
        }
        else if (arg === "--verbose") {
            verbose = true;
        }
        else if (arg === "--reconcile-task") {
            const value = readValue(arg);
            if (value !== undefined)
                reconcileTasks.push(value);
        }
        else if (arg === "--no-learn") {
            noLearn = true;
        }
        else if (arg === "--classic-ceremony") {
            classicCeremony = true;
        }
        else if (arg === "--help" || arg === "-h") {
            help = true;
        }
        else if (arg.startsWith("-")) {
            unknownFlags.push(rawArg);
        }
        else if (!command) {
            command = rawArg;
        }
        else {
            positional.push(rawArg);
        }
    }
    return {
        command,
        cwd,
        simulate,
        dryRun,
        synthetic,
        noDashboard,
        skipMiddenCheck,
        skipWatchers,
        refresh,
        force,
        forceResurvey,
        tasks,
        depth,
        planningDepth,
        verificationDepth,
        targetConfidence,
        maxIterations,
        accept,
        verificationTimeout,
        light,
        heavy,
        workerTimeout,
        circuitBreakerThreshold,
        noSuggest,
        verbose,
        reconcileTasks,
        noLearn,
        classicCeremony,
        help,
        positional,
        unknownFlags,
    };
}
function printUsage() {
    const commandLines = listHostCommandDefinitions()
        .map((definition) => `  ${definition.usage.padEnd(14)} ${definition.description}`)
        .join("\n");
    process.stderr.write("Usage: host <command> [options]\n\n" +
        "Commands:\n" +
        commandLines + "\n\n" +
        "Options:\n" +
        "  --cwd <path>           Working directory\n" +
        "  --simulate             Run in simulation mode (no real worker spawning)\n" +
        "  --dry-run              Preview ceremony output without spawning workers\n" +
        "  --synthetic            Forward Go synthetic mode for plan/build/continue\n" +
        "  --no-dashboard         Disable live dashboard, use plain text output\n" +
        "  --skip-midden-check    Skip pre-build midden threshold check\n" +
        "  --skip-watchers        Skip continue watcher workers when Go allows it\n" +
        "  --refresh              Refresh an existing plan\n" +
        "  --force                Forward Go force aliases for plan/build\n" +
        "  --force-resurvey       Refresh colonize survey artifacts\n" +
        "  --task <id>            Limit build dispatch to a task id (repeatable)\n" +
        "  --depth <level>        fast | balanced | deep | exhaustive\n" +
        "  --planning-depth <lvl> light | standard | deep\n" +
        "  --verification-depth <lvl> light | standard | heavy\n" +
        "  --target <n>           Planning confidence target 70-99\n" +
        "  --max-iterations <n>   Planning iteration budget 2-12\n" +
        "  --accept               Accept current best plan below target\n" +
        "  --verification-timeout <dur> Override continue verification timeout\n" +
        "  --light                Force light review\n" +
        "  --heavy                Force heavy review\n" +
        "  --worker-timeout <dur> Override per-worker timeout (e.g. 5m, 15m)\n" +
        "  --circuit-breaker-threshold <n> Forward build circuit-breaker threshold\n" +
        "  --no-suggest           Skip build suggestion analysis\n" +
        "  --verbose              Forward verbose build output mode\n" +
        "  --reconcile-task <id>  Mark continue task reconciliation (repeatable)\n" +
        "  --no-learn             Disable continue learning capture when supported\n" +
        "  --classic-ceremony     Use continue's heavy visible review manifest\n");
}
function emitCeremonyOutput(output) {
    if (output.trim() === "")
        return;
    process.stderr.write(output.endsWith("\n") ? output : `${output}\n`);
}
function ceremonyExecutionWaves(dispatches) {
    const waves = new Set();
    for (const dispatch of dispatches) {
        const wave = dispatch.execution_wave ?? dispatch.wave ?? 0;
        if (wave > 0)
            waves.add(wave);
    }
    return [...waves].sort((a, b) => a - b);
}
function renderManifestCeremony(ceremony, workflow, manifestEnvelope, dispatches) {
    emitCeremonyOutput(ceremony.renderSpawnPlan(workflow, manifestEnvelope));
    for (const executionWave of ceremonyExecutionWaves(dispatches)) {
        emitCeremonyOutput(ceremony.renderWaveStart(workflow, manifestEnvelope, executionWave));
    }
}
function renderWorkerCeremony(ceremony, workflow, workers) {
    for (const worker of workers) {
        emitCeremonyOutput(ceremony.renderWorkerComplete(workflow, worker));
    }
}
function toWorkerDispatches(dispatches) {
    return dispatches.map((dispatch) => {
        const workerDispatch = {
            stage: dispatch.stage ?? "dispatch",
            caste: dispatch.caste,
            name: dispatch.name,
            task: dispatch.task,
            status: dispatch.status,
        };
        if (dispatch.wave !== undefined)
            workerDispatch.wave = dispatch.wave;
        if (dispatch.execution_wave !== undefined) {
            workerDispatch.execution_wave = dispatch.execution_wave;
        }
        if (dispatch.task_id !== undefined)
            workerDispatch.task_id = dispatch.task_id;
        if (dispatch.summary !== undefined)
            workerDispatch.summary = dispatch.summary;
        if (dispatch.blockers !== undefined)
            workerDispatch.blockers = dispatch.blockers;
        if (dispatch.duration !== undefined)
            workerDispatch.duration = dispatch.duration;
        if (dispatch.skill_section !== undefined) {
            workerDispatch.skill_section = dispatch.skill_section;
        }
        if (dispatch.task_brief !== undefined)
            workerDispatch.task_brief = dispatch.task_brief;
        if (dispatch.hive_section !== undefined) {
            workerDispatch.hive_section = dispatch.hive_section;
        }
        if (dispatch.matched_skills !== undefined) {
            workerDispatch.matched_skills = dispatch.matched_skills;
        }
        if (dispatch.skill_count !== undefined) {
            workerDispatch.skill_count = dispatch.skill_count;
        }
        if (dispatch.colony_skill_count !== undefined) {
            workerDispatch.colony_skill_count = dispatch.colony_skill_count;
        }
        if (dispatch.domain_skill_count !== undefined) {
            workerDispatch.domain_skill_count = dispatch.domain_skill_count;
        }
        return workerDispatch;
    });
}
// ---------------------------------------------------------------------------
// Dispatched runner: real dispatch pipeline for build/plan/continue
// ---------------------------------------------------------------------------
/** Build a skill injection summary line (D-07) */
function emitSkillSummary(dispatches) {
    const skillCount = dispatches.filter((d) => typeof d.skill_section === "string" && d.skill_section.trim() !== "").length;
    if (skillCount > 0) {
        process.stderr.write(`Injecting ${skillCount} skills into worker prompts.\n`);
    }
}
/** Build a hive wisdom injection summary line */
function emitHiveSummary(dispatches) {
    const hiveCount = dispatches.filter((d) => typeof d.hive_section === "string" && d.hive_section.trim() !== "").length;
    if (hiveCount > 0) {
        process.stderr.write(`Injecting hive wisdom into ${hiveCount} worker prompts.\n`);
    }
}
/**
 * Prepare the hive wisdom section for a dispatch pipeline.
 * Resolves domain tags, reads hive wisdom, and returns a formatted section.
 * Graceful degradation: any failure logs a warning and returns empty string.
 */
async function prepareHiveSection(bridge) {
    try {
        const domains = await resolveDomainTags(bridge);
        return await readHiveWisdom({
            goBinaryPath: bridge.goBinaryPath,
            cwd: bridge.cwd,
            domains,
            minConfidence: 0.5,
            maxEntries: 10,
            budgetChars: 2500,
        });
    }
    catch (err) {
        const msg = err instanceof Error ? err.message : String(err);
        process.stderr.write(`Warning: hive-read failed: ${msg}\n`);
        return "";
    }
}
// ---------------------------------------------------------------------------
// Dry-run ceremony preview (HOST-07, D-06)
// ---------------------------------------------------------------------------
/**
 * Run the dry-run path for any dispatched command: fetch the manifest,
 * render ceremony, show DRY RUN badge, exit without dispatching workers.
 */
async function runDryRunDispatchedCommand(bridge, parsed, definition) {
    const ceremony = createCeremonyAdapter(bridge);
    const goArgs = buildHostGoArgs(parsed);
    const workflow = (definition.ceremonyWorkflow ?? "build");
    // Fetch manifest (read-only)
    const manifestResult = _callGoJSONRef(bridge, goArgs);
    // Extract dispatches from manifest for ceremony rendering
    const manifestEnvelope = manifestResult;
    const topLevelDispatches = manifestResult?.dispatches;
    const dispatchManifest = manifestResult?.dispatch_manifest;
    const planManifest = manifestResult?.plan_manifest;
    const planningManifest = manifestResult?.planning_manifest;
    const continueManifest = manifestResult?.continue_manifest;
    const dispatches = topLevelDispatches
        ?? dispatchManifest?.dispatches
        ?? planManifest?.dispatches
        ?? planningManifest?.dispatches
        ?? continueManifest?.dispatches
        ?? [];
    const manifestObj = dispatchManifest ?? planManifest ?? planningManifest ?? continueManifest;
    // Attach hive wisdom to each dispatch (dry-run also fetches wisdom per RESEARCH.md Q4)
    const hiveSection = await prepareHiveSection(bridge);
    for (const d of dispatches) {
        d.hive_section = hiveSection;
    }
    // Load playbooks for the active workflow and inject context into worker briefs (CEREMONY-06)
    const dryRunPlaybooks = loadPlaybooksForWorkflow(bridge.cwd, workflow);
    const dryRunPlaybookContext = renderPlaybookContext(dryRunPlaybooks);
    if (dryRunPlaybookContext) {
        for (const d of dispatches) {
            d.task_brief = (d.task_brief ?? d.task ?? "") + "\n\n" + dryRunPlaybookContext;
        }
    }
    if (manifestObj) {
        const envelope = { ...manifestEnvelope, [`${workflow}_manifest`]: manifestObj };
        renderManifestCeremony(ceremony, workflow, envelope, dispatches);
    }
    else {
        renderManifestCeremony(ceremony, workflow, manifestEnvelope, dispatches);
    }
    // DRY RUN badge (D-06)
    renderDryRunBadge();
    // Skill injection summary
    emitSkillSummary(dispatches);
    // Hive wisdom injection summary
    emitHiveSummary(dispatches);
    // Output manifest JSON to stdout
    process.stdout.write(JSON.stringify({ ok: true, dry_run: true, manifest: manifestResult }, null, 2) + "\n");
}
// ---------------------------------------------------------------------------
// Iteration ceremony helpers (ITER-06)
// ---------------------------------------------------------------------------
/** Render an iteration marker between dispatch waves. */
function renderIterationCeremony(result) {
    const parts = [
        `Iteration ${result.iterationCount}:`,
        `confidence ${result.currentConfidence}%`,
        `(delta ${result.delta >= 0 ? "+" : ""}${result.delta}%,`,
        `budget ${result.budgetRemaining} workers remaining)`,
    ];
    if (result.stopReason) {
        parts.push(`[${result.stopReason}]`);
    }
    emitCeremonyOutput(`\u2500\u2500 ${parts.join(" ")} \u2500\u2500`);
}
/** Render the final iteration stop reason. */
function renderIterationComplete(stopReason) {
    emitCeremonyOutput(`\u2500\u2500 Iteration complete: ${stopReason} \u2500\u2500`);
}
/**
 * Dispatch a single build wave: dispatch workers, render ceremony,
 * write completion file, call finalizer.
 *
 * @param bridge - Go bridge options
 * @param parsed - Parsed host arguments
 * @param ceremony - Ceremony adapter for rendering
 * @param buildManifest - The dispatch manifest
 * @param dispatches - The dispatches for this wave
 * @param spawnOrchestrator - Spawn budget orchestrator (carries cumulative budget)
 * @param iterationFeedback - Optional feedback from a previous iteration to inject into task briefs
 * @returns Wave result with mapped workers, completion path, and claims
 */
async function dispatchBuildWave(bridge, parsed, ceremony, buildManifest, dispatches, spawnOrchestrator, iterationFeedback) {
    const phase = parsed.positional[0] ?? "1";
    // Inject iteration feedback into task briefs if provided
    if (iterationFeedback) {
        for (const d of dispatches) {
            const existingBrief = d.task_brief ?? d.task ?? "";
            d.task_brief = existingBrief
                ? `${existingBrief}\n\nPrevious iteration feedback:\n${iterationFeedback}`
                : `Previous iteration feedback:\n${iterationFeedback}`;
        }
    }
    // Skill injection summary (D-07)
    emitSkillSummary(dispatches);
    // Hive wisdom injection summary
    emitHiveSummary(dispatches);
    // Dispatch workers
    const dispatchOpts = {
        goBinaryPath: bridge.goBinaryPath,
        cwd: bridge.cwd,
        simulateWorkers: parsed.simulate,
        spawnOrchestrator,
    };
    const workerResults = await _dispatchWorkersRef(dispatchOpts, dispatches);
    const mappedResults = toWorkerResults(dispatches, workerResults);
    // Render worker-complete ceremony
    renderWorkerCeremony(ceremony, "build", mappedResults);
    // Write completion file and call finalizer
    const completion = {
        dispatch_manifest: buildManifest,
        dispatches: mappedResults,
    };
    const completionPath = writeCompletionFile(approvedCompletionDirPrefix("build"), "build-completion.json", { result: completion });
    _callGoJSONRef(bridge, [
        "build-finalize", phase,
        "--completion-file", completionPath,
    ]);
    cleanupCompletionDir(completionPath);
    // Build worker claims from raw dispatch results for confidence evaluation.
    // Raw results carry extra fields (blockers, test_results) that toWorkerResults drops.
    const workerClaims = workerResults.map((w) => {
        const claim = {
            status: w.status ?? "completed",
        };
        const blockers = w.blockers;
        if (Array.isArray(blockers))
            claim.blockers = blockers;
        const filesCreated = w.files_created;
        if (Array.isArray(filesCreated))
            claim.files_created = filesCreated;
        const filesModified = w.files_modified;
        if (Array.isArray(filesModified))
            claim.files_modified = filesModified;
        const testsWritten = w.tests_written;
        if (Array.isArray(testsWritten))
            claim.tests_written = testsWritten;
        // Preserve test_results for ConfidenceEvaluator raw cast
        const testResults = w.test_results;
        if (typeof testResults === "object" && testResults !== null) {
            claim.test_results = testResults;
        }
        return claim;
    });
    return {
        mappedResults,
        completionPath,
        buildManifest,
        workerCount: dispatches.length,
        workerClaims,
    };
}
/**
 * Run the dispatched build pipeline: fetch manifest, dispatch workers,
 * write completion file, call finalizer, render ceremony.
 *
 * Now iteration-aware: after each dispatch wave, evaluate confidence.
 * If confidence is too low and budget/iterations remain, re-dispatch
 * with failure feedback injected into worker task briefs.
 */
async function runDispatchedBuildCommand(bridge, parsed, definition) {
    const ceremony = createCeremonyAdapter(bridge);
    const goArgs = buildHostGoArgs(parsed);
    const phase = parsed.positional[0] ?? "1";
    // Step 1: Fetch manifest
    const buildResult = _callGoJSONRef(bridge, goArgs);
    const buildManifest = buildResult.dispatch_manifest;
    if (!buildManifest) {
        throw new Error("Build --plan-only returned no dispatch_manifest. Check colony state and try again.");
    }
    let dispatches = buildManifest.dispatches ?? buildResult.dispatches ?? [];
    if (dispatches.length === 0) {
        throw new Error("Build manifest contains no dispatches. Nothing to build.");
    }
    // Step 1b: Resolve hive wisdom and attach to each dispatch
    const hiveSection = await prepareHiveSection(bridge);
    for (const d of dispatches) {
        d.hive_section = hiveSection;
    }
    // Step 1c: Load build playbooks and inject context into worker briefs (CEREMONY-06)
    const buildPlaybooks = loadPlaybooksForWorkflow(bridge.cwd, "build");
    const buildPlaybookContext = renderPlaybookContext(buildPlaybooks);
    if (buildPlaybookContext) {
        for (const d of dispatches) {
            d.task_brief = (d.task_brief ?? d.task ?? "") + "\n\n" + buildPlaybookContext;
        }
    }
    // Step 2: Render spawn-plan and wave-start ceremony
    const ceremonyEnvelope = { dispatch_manifest: buildManifest };
    renderManifestCeremony(ceremony, "build", ceremonyEnvelope, dispatches);
    // Step 3: Check available platforms (unless simulating)
    if (!parsed.simulate) {
        const available = await _detectAvailablePlatformsRef();
        if (available.length === 0) {
            const diagnostic = buildResult.provider_diagnostics;
            const msg = diagnostic
                ? `No platform workers available. ${diagnostic}`
                : formatPlatformUnavailableMessage(`build phase ${phase}`);
            throw new Error(msg);
        }
    }
    // Step 5: Initialize spawn budget from manifest QueenSpawnBudget.max_workers (SPAWN-03)
    const spawnBudget = buildManifest.queen_execution_policy?.spawn_budget?.max_workers ?? 20;
    const spawnOrchestrator = createSpawnOrchestrator({
        goBinaryPath: bridge.goBinaryPath,
        cwd: bridge.cwd,
        totalBudget: spawnBudget,
        consumedBudget: dispatches.length,
        currentDepth: 1,
    });
    // Step 6: Initialize ConfidenceLoop and ConfidenceEvaluator
    const loopOpts = {
        totalBudget: spawnBudget,
    };
    if (parsed.maxIterations) {
        loopOpts.maxIterations = parseInt(parsed.maxIterations, 10);
    }
    if (parsed.targetConfidence) {
        loopOpts.confidenceTarget = parseInt(parsed.targetConfidence, 10);
    }
    const confidenceLoop = new ConfidenceLoop(loopOpts);
    const confidenceEvaluator = new ConfidenceEvaluator();
    // Step 7: Iteration loop
    let lastWaveResult;
    let iterationCount = 0;
    while (true) {
        iterationCount++;
        // Re-attach hive wisdom for iterations after the first (dispatches are re-fetched)
        if (iterationCount > 1) {
            for (const d of dispatches) {
                d.hive_section = hiveSection;
            }
        }
        // Build iteration feedback from previous iteration's blockers
        let iterationFeedback;
        if (lastWaveResult && lastWaveResult.workerClaims.length > 0) {
            const allBlockers = [];
            for (const claim of lastWaveResult.workerClaims) {
                if (claim.blockers) {
                    for (const b of claim.blockers)
                        allBlockers.push(b);
                }
            }
            if (allBlockers.length > 0) {
                iterationFeedback = `Blockers from previous iteration:\n${allBlockers.map((b) => `- ${b}`).join("\n")}`;
            }
        }
        // Dispatch the wave
        lastWaveResult = await dispatchBuildWave(bridge, parsed, ceremony, buildManifest, dispatches, spawnOrchestrator, iterationFeedback);
        // Evaluate confidence from worker claims
        const evaluated = confidenceEvaluator.evaluate({
            workerClaims: lastWaveResult.workerClaims,
        });
        // Feed into confidence loop
        const loopResult = confidenceLoop.evaluate(evaluated.score, lastWaveResult.workerCount);
        // Render iteration ceremony marker
        renderIterationCeremony(loopResult);
        // Check if loop should continue
        if (!loopResult.shouldContinue) {
            renderIterationComplete(loopResult.stopReason);
            break;
        }
        // Re-fetch manifest for next iteration (Go may adjust dispatches)
        const reFetchResult = _callGoJSONRef(bridge, goArgs);
        const reFetchManifest = reFetchResult.dispatch_manifest;
        if (!reFetchManifest) {
            // Manifest re-fetch failed; stop iterating
            renderIterationComplete("manifest_re_fetch_failed");
            break;
        }
        const newDispatches = reFetchManifest.dispatches ?? reFetchResult.dispatches ?? [];
        if (newDispatches.length === 0) {
            renderIterationComplete("no_dispatches");
            break;
        }
        // Update dispatches and spawn orchestrator consumed budget for new dispatches
        dispatches = newDispatches;
        // Re-inject playbook context for re-fetched dispatches (playbook content
        // is lost when manifest is re-fetched from Go on subsequent iterations)
        if (buildPlaybookContext) {
            for (const d of dispatches) {
                d.task_brief = (d.task_brief ?? d.task ?? "") + "\n\n" + buildPlaybookContext;
            }
        }
        // Re-create spawn orchestrator with updated budget for the new dispatches
        // The confidence loop already tracks cumulative budget internally
    }
    // Step 8: Render closeout with final completion path
    const finalCompletionPath = lastWaveResult?.completionPath ?? "";
    emitCeremonyOutput(ceremony.renderCloseout("build", finalCompletionPath));
    // Output result JSON to stdout with iteration summary
    const loopState = confidenceLoop.getState();
    const lastResult = lastWaveResult;
    // Determine stop reason from the loop state
    let stopReason = "unknown";
    if (loopState.confidenceHistory.length > 0) {
        // Re-evaluate the stop condition from the last recorded confidence
        const lastConfidence = loopState.confidenceHistory[loopState.confidenceHistory.length - 1];
        if (lastConfidence >= (loopOpts.confidenceTarget ?? 80)) {
            stopReason = "confidence_target_met";
        }
        else if (loopState.iterationCount >= (loopOpts.maxIterations ?? 3)) {
            stopReason = "max_iterations_met";
        }
        else if (loopOpts.totalBudget && loopOpts.totalBudget > 0 && loopState.budgetRemaining <= 0) {
            stopReason = "budget_exhausted";
        }
        else {
            stopReason = "diminishing_returns";
        }
    }
    process.stdout.write(JSON.stringify({
        ok: true,
        completion_file: finalCompletionPath,
        iterations: {
            count: loopState.iterationCount,
            final_confidence: loopState.confidenceHistory[loopState.confidenceHistory.length - 1] ?? 0,
            stop_reason: stopReason,
        },
    }, null, 2) + "\n");
}
/**
 * Run the dispatched plan pipeline: fetch plan manifest, dispatch planning
 * workers (Scout/Route-Setter), write completion file, call plan-finalizer.
 */
async function runDispatchedPlanCommand(bridge, parsed) {
    const ceremony = createCeremonyAdapter(bridge);
    const goArgs = buildHostGoArgs(parsed);
    // Step 1: Fetch plan manifest
    const planResult = _callGoJSONRef(bridge, goArgs);
    const planManifest = planResult.plan_manifest ?? planResult.planning_manifest;
    if (!planManifest) {
        throw new Error("Plan --plan-only returned no plan manifest. Check colony state and try again.");
    }
    const dispatches = planResult.dispatches ?? [];
    if (dispatches.length === 0) {
        throw new Error("Plan manifest contains no dispatches. Nothing to plan.");
    }
    // Step 1b: Resolve hive wisdom and attach to each dispatch
    const hiveSection = await prepareHiveSection(bridge);
    for (const d of dispatches) {
        d.hive_section = hiveSection;
    }
    // Step 1c: Load plan playbooks and inject context into worker briefs (CEREMONY-06)
    const planPlaybooks = loadPlaybooksForWorkflow(bridge.cwd, "plan");
    const planPlaybookContext = renderPlaybookContext(planPlaybooks);
    if (planPlaybookContext) {
        for (const d of dispatches) {
            d.task_brief = (d.task_brief ?? d.task ?? "") + "\n\n" + planPlaybookContext;
        }
    }
    // Step 2: Render spawn-plan and wave-start ceremony
    const ceremonyEnvelope = { plan_manifest: planManifest, dispatches };
    renderManifestCeremony(ceremony, "plan", ceremonyEnvelope, dispatches);
    // Step 3: Check platforms (unless simulating)
    if (!parsed.simulate) {
        const available = await _detectAvailablePlatformsRef();
        if (available.length === 0) {
            throw new Error(formatPlatformUnavailableMessage("plan"));
        }
    }
    // Step 3b: Hive wisdom injection summary
    emitHiveSummary(dispatches);
    // Step 4: Dispatch planning workers
    const dispatchOpts = {
        goBinaryPath: bridge.goBinaryPath,
        cwd: bridge.cwd,
        simulateWorkers: parsed.simulate,
    };
    const buildDispatches = toWorkerDispatches(dispatches);
    const workerResults = await _dispatchWorkersRef(dispatchOpts, buildDispatches);
    const mappedResults = toWorkerResults(buildDispatches, workerResults);
    // Step 5: Render worker-complete ceremony
    renderWorkerCeremony(ceremony, "plan", mappedResults);
    // Step 6: Write completion file and call finalizer
    const completion = {
        plan_manifest: planManifest,
        dispatches: mappedResults,
    };
    const completionPath = writeCompletionFile(approvedCompletionDirPrefix("plan"), "plan-completion.json", { result: completion });
    _callGoJSONRef(bridge, [
        "plan-finalize",
        "--completion-file", completionPath,
    ]);
    cleanupCompletionDir(completionPath);
    // Step 7: Render closeout
    emitCeremonyOutput(ceremony.renderCloseout("plan", completionPath));
    process.stdout.write(JSON.stringify({ ok: true, completion_file: completionPath }, null, 2) + "\n");
}
/**
 * Run the dispatched continue pipeline: fetch continue manifest, dispatch
 * review workers, write completion file, call continue-finalizer.
 */
async function runDispatchedContinueCommand(bridge, parsed) {
    const ceremony = createCeremonyAdapter(bridge);
    const goArgs = buildHostGoArgs(parsed);
    // Step 1: Fetch continue manifest
    const continueResult = _callGoJSONRef(bridge, goArgs);
    const continueManifest = continueResult.continue_manifest;
    if (!continueManifest) {
        throw new Error("Continue --plan-only returned no continue manifest. Check colony state and try again.");
    }
    const dispatches = continueResult.dispatches ?? [];
    if (dispatches.length === 0) {
        throw new Error("Continue manifest contains no dispatches. Nothing to continue.");
    }
    // Step 1b: Resolve hive wisdom and attach to each dispatch
    const hiveSection = await prepareHiveSection(bridge);
    for (const d of dispatches) {
        d.hive_section = hiveSection;
    }
    // Step 2: Render spawn-plan and wave-start ceremony
    const ceremonyEnvelope = { continue_manifest: continueManifest, dispatches };
    renderManifestCeremony(ceremony, "continue", ceremonyEnvelope, dispatches);
    // Step 3: Check platforms (unless simulating)
    if (!parsed.simulate) {
        const available = await _detectAvailablePlatformsRef();
        if (available.length === 0) {
            throw new Error(formatPlatformUnavailableMessage("continue"));
        }
    }
    // Step 3b: Hive wisdom injection summary
    emitHiveSummary(dispatches);
    // Step 4: Dispatch review workers
    const dispatchOpts = {
        goBinaryPath: bridge.goBinaryPath,
        cwd: bridge.cwd,
        simulateWorkers: parsed.simulate,
    };
    const buildDispatches = toWorkerDispatches(dispatches);
    const workerResults = await _dispatchWorkersRef(dispatchOpts, buildDispatches);
    const mappedResults = toWorkerResults(buildDispatches, workerResults);
    // Step 5: Render worker-complete ceremony
    renderWorkerCeremony(ceremony, "continue", mappedResults);
    // Step 6: Write completion file and call finalizer
    const completion = {
        continue_manifest: continueManifest,
        dispatches: mappedResults,
    };
    const completionPath = writeCompletionFile(approvedCompletionDirPrefix("continue"), "continue-completion.json", { result: completion });
    _callGoJSONRef(bridge, [
        "continue-finalize",
        "--completion-file", completionPath,
    ]);
    cleanupCompletionDir(completionPath);
    // Step 7: Render closeout
    emitCeremonyOutput(ceremony.renderCloseout("continue", completionPath));
    process.stdout.write(JSON.stringify({ ok: true, completion_file: completionPath }, null, 2) + "\n");
}
async function main() {
    const parsed = parseArgs(process.argv);
    const { command, cwd, simulate, noDashboard, skipMiddenCheck, help, positional } = parsed;
    if (help || !command) {
        printUsage();
        process.exit(help ? 0 : 1);
    }
    const goBinaryPath = discoverGoBinary();
    const bridge = { goBinaryPath, cwd };
    const definition = getHostCommandDefinition(command);
    if (!definition) {
        process.stderr.write(`Unknown command: ${command}\n`);
        printUsage();
        process.exit(1);
    }
    if (parsed.unknownFlags.length > 0) {
        process.stderr.write(`Error: Unsupported host flag(s): ${parsed.unknownFlags.join(", ")}\n`);
        process.exit(1);
    }
    switch (definition.runner) {
        case "dispatched": {
            const workflow = definition.ceremonyWorkflow ?? "build";
            try {
                if (parsed.dryRun) {
                    await runDryRunDispatchedCommand(bridge, parsed, definition);
                }
                else if (workflow === "build") {
                    await runDispatchedBuildCommand(bridge, parsed, definition);
                }
                else if (workflow === "plan") {
                    await runDispatchedPlanCommand(bridge, parsed);
                }
                else if (workflow === "continue") {
                    await runDispatchedContinueCommand(bridge, parsed);
                }
                else {
                    // Fallback for colonize/seal/swarm -- go-json passthrough
                    let args;
                    try {
                        args = buildHostGoArgs(parsed);
                    }
                    catch (err) {
                        const message = err instanceof Error ? err.message : String(err);
                        process.stderr.write(`Error: ${message}\n`);
                        process.exit(1);
                    }
                    const result = runGoJSONCommand(bridge, args, _callGoJSONRef);
                    process.stdout.write(JSON.stringify(result, null, 2) + "\n");
                }
            }
            catch (err) {
                const message = err instanceof Error ? err.message : String(err);
                process.stderr.write(`Error: ${message}\n`);
                process.exit(1);
            }
            break;
        }
        case "go-json": {
            let args;
            try {
                args = buildHostGoArgs(parsed);
            }
            catch (err) {
                const message = err instanceof Error ? err.message : String(err);
                process.stderr.write(`Error: ${message}\n`);
                process.exit(1);
            }
            const result = runGoJSONCommand(bridge, args, _callGoJSONRef);
            if (parsed.dryRun) {
                renderDryRunBadge();
                process.stdout.write(JSON.stringify({ ok: true, dry_run: true, manifest: result }, null, 2) + "\n");
            }
            else {
                process.stdout.write(JSON.stringify(result, null, 2) + "\n");
            }
            break;
        }
        case "oracle-lifecycle": {
            const topic = positional[0] || "auto";
            if (parsed.dryRun) {
                // Dry-run: fetch first iteration manifest, render ceremony, show badge
                const ceremony = createCeremonyAdapter(bridge);
                const manifestResult = _callGoJSONRef(bridge, [
                    "oracle-iterate",
                    "--plan-only",
                    "--topic",
                    topic,
                ]);
                const state = manifestResult?.iteration_manifest;
                const ceremonyEnvelope = state ? { iteration_manifest: state } : manifestResult;
                emitCeremonyOutput(ceremony.renderSpawnPlan("build", ceremonyEnvelope));
                emitCeremonyOutput(ceremony.renderWaveStart("build", ceremonyEnvelope, 1));
                renderDryRunBadge();
                process.stdout.write(JSON.stringify({ ok: true, dry_run: true, manifest: manifestResult }, null, 2) + "\n");
                break;
            }
            const oracleOpts = {
                goBinaryPath,
                cwd,
                topic,
                simulateWorkers: simulate,
                dashboard: !noDashboard,
            };
            const result = await runOracleLifecycle(oracleOpts);
            process.stdout.write(JSON.stringify(result, null, 2) + "\n");
            break;
        }
        case "lifecycle": {
            const phaseArg = positional[0];
            const oracleTopicArg = positional[1];
            if (phaseArg && isNaN(parseInt(phaseArg, 10))) {
                process.stderr.write("Error: lifecycle phase must be a number\n");
                process.exit(1);
            }
            if (!simulate) {
                process.stderr.write("Error: lifecycle is experimental and simulate-only. Use --simulate for the smoke harness, or run aether plan/build/continue for real orchestration.\n");
                process.exit(1);
            }
            if (simulate) {
                process.stderr.write("Running in simulation mode\n");
            }
            const lifecycleOpts = {
                goBinaryPath,
                cwd,
                simulateWorkers: simulate,
                dashboard: !noDashboard,
                skipMiddenCheck,
            };
            if (phaseArg) {
                lifecycleOpts.phase = parseInt(phaseArg, 10);
            }
            if (oracleTopicArg) {
                lifecycleOpts.runOracle = true;
                lifecycleOpts.oracleTopic = oracleTopicArg;
            }
            const narrator = createNarrator({
                cwd,
                outputMode: process.env["AETHER_OUTPUT_MODE"],
                suppressOutput: !noDashboard && process.stdout.isTTY,
            });
            const eventBridge = await startEventBridge({
                goBinaryPath,
                cwd,
                onEvent: (evt) => {
                    narrator.onEvent(evt);
                },
            });
            const result = await runLifecycle(lifecycleOpts);
            await eventBridge.stop();
            narrator.stop();
            process.stdout.write(JSON.stringify(result, null, 2) + "\n");
            break;
        }
        case "watch-display": {
            const watchOpts = {
                goBinaryPath,
                cwd,
                dashboard: !noDashboard,
            };
            const result = await runWatchDisplay(watchOpts);
            process.stdout.write(JSON.stringify(result, null, 2) + "\n");
            break;
        }
        case "swarm-display": {
            const target = positional[0] || "";
            const swarmOpts = {
                goBinaryPath,
                cwd,
                target,
                dashboard: !noDashboard,
                planOnly: true,
            };
            const result = await runSwarmDisplay(swarmOpts);
            process.stdout.write(JSON.stringify(result, null, 2) + "\n");
            break;
        }
    }
}
import { fileURLToPath } from "node:url";
import { resolve } from "node:path";
import { realpathSync } from "node:fs";
function normalizeEntrypointPath(path) {
    if (!path)
        return "";
    try {
        return realpathSync(path);
    }
    catch {
        return resolve(path);
    }
}
const isMainModule = normalizeEntrypointPath(fileURLToPath(import.meta.url)) ===
    normalizeEntrypointPath(process.argv[1]);
if (isMainModule) {
    main().catch((err) => {
        const message = err instanceof Error ? err.message : String(err);
        process.stderr.write(`Fatal: ${message}\n`);
        process.exit(1);
    });
}
