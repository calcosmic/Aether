/**
 * Runtime ceremony adapter for the TypeScript host.
 *
 * The TS host may decide when ceremony surfaces should be shown, but the Go
 * runtime owns the visual rendering contract. This adapter writes temporary
 * JSON packets and calls `aether ceremony ...` with visual output enabled.
 */
import { execFileSync } from "node:child_process";
import { isTerminalWorkerStatus } from "./worker-status.js";
import { writeCompletionFile } from "./go-bridge.js";
function runGoCeremonyCommand(opts, args) {
    try {
        return execFileSync(opts.goBinaryPath, args, {
            cwd: opts.cwd,
            env: {
                ...process.env,
                AETHER_OUTPUT_MODE: "visual",
                AETHER_FORCE_COLOR: "1",
            },
            encoding: "utf-8",
            maxBuffer: 10 * 1024 * 1024,
            stdio: ["ignore", "pipe", "pipe"],
        });
    }
    catch (err) {
        const typed = err;
        const detail = typeof typed.status === "number"
            ? `exit status ${typed.status}`
            : typeof typed.code === "string"
                ? typed.code
                : typeof typed.signal === "string"
                    ? `signal ${typed.signal}`
                    : "execution failed";
        throw new Error(`Go ceremony command failed for ${args.slice(0, 3).join(" ")}: ${detail}; subprocess output omitted`);
    }
}
function isTerminalWorkerResult(data) {
    if (typeof data !== "object" || data === null || Array.isArray(data)) {
        return false;
    }
    return isTerminalWorkerStatus(data["status"]);
}
export class GoCeremonyAdapter {
    opts;
    runner;
    constructor(opts, runner = runGoCeremonyCommand) {
        this.opts = opts;
        this.runner = runner;
    }
    renderSpawnPlan(workflow, manifestEnvelope) {
        const manifestFile = this.writePacket(workflow, "manifest", manifestEnvelope);
        return this.runner(this.opts, [
            "ceremony",
            "spawn-plan",
            "--workflow",
            workflow,
            "--manifest-file",
            manifestFile,
        ]);
    }
    renderWaveStart(workflow, manifestEnvelope, executionWave) {
        const manifestFile = this.writePacket(workflow, "manifest", manifestEnvelope);
        return this.runner(this.opts, [
            "ceremony",
            "wave-start",
            "--workflow",
            workflow,
            "--manifest-file",
            manifestFile,
            "--execution-wave",
            String(executionWave),
        ]);
    }
    renderWorkerComplete(workflow, workerResult) {
        if (!isTerminalWorkerResult(workerResult)) {
            return "";
        }
        const workerFile = this.writePacket(workflow, "worker", workerResult);
        return this.runner(this.opts, [
            "ceremony",
            "worker-complete",
            "--workflow",
            workflow,
            "--worker-file",
            workerFile,
        ]);
    }
    renderCloseout(workflow, completionFile) {
        return this.runner(this.opts, [
            "ceremony",
            "closeout",
            "--workflow",
            workflow,
            "--completion-file",
            completionFile,
        ]);
    }
    writePacket(workflow, kind, data) {
        return writeCompletionFile("aether-ceremony", `${workflow}-${kind}.json`, data);
    }
}
let _createCeremonyAdapterRef = (opts) => new GoCeremonyAdapter(opts);
export function createCeremonyAdapter(opts) {
    return _createCeremonyAdapterRef(opts);
}
export function __setCreateCeremonyAdapter(factory) {
    _createCeremonyAdapterRef = factory;
}
export function __restoreCreateCeremonyAdapter() {
    _createCeremonyAdapterRef = (opts) => new GoCeremonyAdapter(opts);
}
// ---------------------------------------------------------------------------
// DRY RUN badge rendering (D-06)
// ---------------------------------------------------------------------------
/**
 * Render a visible DRY RUN badge to stderr.
 * Per D-06, the badge is shown before and after ceremony output
 * so the user knows it is a preview and no workers will be dispatched.
 */
export function renderDryRunBadge() {
    process.stderr.write("--- DRY RUN: No workers dispatched. Showing ceremony preview only. ---\n");
}
