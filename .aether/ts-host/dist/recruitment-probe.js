/**
 * TypeScript-side native-nesting probe (BIO-03, 203-04).
 *
 * Mirrors cmd/recruitment_probe.go's contract on the Go side: a bounded,
 * isolated, session-cached probe that asks the resolved platform CLI
 * whether an agent-spawning tool appears in its own available tool list.
 * Never rooted in the repository, never inferred from a platform name or a
 * version string, and run at most once per process.
 *
 * This is the counter-example the folded todo
 * (.planning/todos/pending/2026-08-01-ts-host-preflight-hardcoded-timeout.md)
 * names -- a hardcoded, non-configurable, repo-rooted probe -- retired.
 */
import { spawn } from "node:child_process";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { parseGoDurationMs } from "./preflight-config.js";
/** Default probe budget in milliseconds, mirroring recruitmentProbeDefaultTimeout (cmd/recruitment_probe.go). */
export const RECRUITMENT_PROBE_DEFAULT_TIMEOUT_MS = 10_000;
// One warning per distinct bad value per process, mirroring
// resolvePreflightTimeoutMs's own dedupe (preflight-config.ts).
const warnedRecruitmentProbeTimeoutValues = new Set();
/** Test-only: reset the once-per-value warning dedupe. */
export function __resetRecruitmentProbeTimeoutWarnings() {
    warnedRecruitmentProbeTimeoutValues.clear();
}
/**
 * Resolve the native-nesting probe timeout from AETHER_RECRUIT_PROBE_TIMEOUT.
 *
 * Same resolve-warn-fallback shape as resolvePreflightTimeoutMs, reusing the
 * same parseGoDurationMs parser rather than declaring a second one. This is
 * a genuinely NEW knob (a different question than provider readiness), not
 * a duplicate of AETHER_PREFLIGHT_TIMEOUT -- contrast with
 * resolveProbeTimeoutMs in platform-dispatcher.ts, which deliberately does
 * NOT introduce a second AETHER_PROBE_TIMEOUT (see that file's own comment).
 */
export function resolveRecruitmentProbeTimeoutMs() {
    const raw = process.env["AETHER_RECRUIT_PROBE_TIMEOUT"]?.trim();
    if (!raw)
        return RECRUITMENT_PROBE_DEFAULT_TIMEOUT_MS;
    const parsedMs = parseGoDurationMs(raw);
    if (parsedMs !== undefined && parsedMs > 0)
        return parsedMs;
    if (!warnedRecruitmentProbeTimeoutValues.has(raw)) {
        warnedRecruitmentProbeTimeoutValues.add(raw);
        process.stderr.write(`Warning: AETHER_RECRUIT_PROBE_TIMEOUT="${raw}" is not a positive duration (expected e.g. "10s", "1500ms", "1m30s", "1.5h") — falling back to ${RECRUITMENT_PROBE_DEFAULT_TIMEOUT_MS}ms\n`);
    }
    return RECRUITMENT_PROBE_DEFAULT_TIMEOUT_MS;
}
const PROBE_PROMPT = "List your available tools, then reply with exactly one line and nothing else: NATIVE_NESTING_AVAILABLE=true if an agent-spawning tool (for example, a Task tool) appears in your own available tool list, otherwise NATIVE_NESTING_AVAILABLE=false.\n";
function nativeNestingReportedAvailable(output) {
    return output.includes("NATIVE_NESTING_AVAILABLE=true");
}
async function defaultRecruitmentProbeLauncher(binary, args, cwd, timeoutMs) {
    return new Promise((resolve, reject) => {
        // detached: true makes the child the leader of its own process group on
        // POSIX so process.kill(-pid) below can signal the whole group, not just
        // the direct child -- Node has no direct process-group option, so this
        // is the port of the Go side's codex.ConfigureWorkerCommand behaviour
        // the folded todo names.
        const child = spawn(binary, args, { cwd, detached: true });
        let timedOut = false;
        const timer = setTimeout(() => {
            timedOut = true;
            if (typeof child.pid === "number") {
                try {
                    process.kill(-child.pid, "SIGKILL");
                }
                catch {
                    // process group may already be gone
                }
            }
        }, timeoutMs);
        const stdout = [];
        const stderr = [];
        child.stdin?.write(PROBE_PROMPT);
        child.stdin?.end();
        child.stdout?.on("data", (chunk) => stdout.push(chunk));
        child.stderr?.on("data", (chunk) => stderr.push(chunk));
        child.on("error", (err) => {
            clearTimeout(timer);
            reject(err);
        });
        child.on("close", (exitCode) => {
            clearTimeout(timer);
            const output = Buffer.concat(stdout).toString("utf-8") + Buffer.concat(stderr).toString("utf-8");
            resolve({ output, timedOut, exitCode });
        });
    });
}
let _launcherRef = defaultRecruitmentProbeLauncher;
/** Test-only: substitute a deterministic launcher. */
export function __setRecruitmentProbeLauncher(fn) {
    _launcherRef = fn;
}
/** Test-only: restore the real launcher. */
export function __restoreRecruitmentProbeLauncher() {
    _launcherRef = defaultRecruitmentProbeLauncher;
}
function resolveRecruitmentProbeBinary() {
    const override = process.env["AETHER_RECRUIT_BINARY"]?.trim();
    if (override)
        return override;
    const codexPath = process.env["AETHER_CODEX_PATH"]?.trim();
    if (codexPath)
        return codexPath;
    return "codex";
}
function recruitmentProbeArgs() {
    // Dedicated, test-only override (newline-delimited, mirroring the Go
    // side's AETHER_RECRUIT_PROBE_ARGS convention) so a test can drive a
    // deterministic fake binary without depending on an installed platform
    // CLI.
    const override = process.env["AETHER_RECRUIT_PROBE_ARGS"];
    if (override !== undefined) {
        return override.trim() === "" ? [] : override.split("\n");
    }
    return [
        "--sandbox",
        "read-only",
        "--ask-for-approval",
        "never",
        "exec",
        "--json",
        "--ephemeral",
        "--skip-git-repo-check",
    ];
}
let cachedProbe;
let cachedProbePromise;
/** Test-only: clear the session cache so a test can exercise both the first
 * (real launch) and the cached (no launch) call. */
export function __resetRecruitmentProbeCache() {
    cachedProbe = undefined;
    cachedProbePromise = undefined;
}
/**
 * Answer "can a helper nest natively here?" by trying it, cheaply, once, in
 * a temp directory outside the repository. Retries exactly once, and only
 * on a timeout -- a launch error (missing binary, refused credential) fails
 * immediately. The result is cached for the lifetime of this process: the
 * second and later calls resolve immediately with no new launch, so a
 * session that recruits ten times pays for one probe and a session that
 * never recruits pays for none.
 */
export async function probeNativeNesting() {
    if (cachedProbe)
        return cachedProbe;
    if (cachedProbePromise)
        return cachedProbePromise;
    cachedProbePromise = runRecruitmentProbe();
    cachedProbe = await cachedProbePromise;
    return cachedProbe;
}
async function runRecruitmentProbe() {
    const binary = resolveRecruitmentProbeBinary();
    const budgetMs = resolveRecruitmentProbeTimeoutMs();
    const probedAt = new Date().toISOString();
    let dir;
    try {
        dir = mkdtempSync(join(tmpdir(), "aether-recruit-probe-"));
    }
    catch {
        return { binary, budgetMs, supported: false, verdict: "probe-setup-failed", elapsedMs: 0, probedAt };
    }
    const start = Date.now();
    let attemptResult;
    for (let attempt = 0; attempt < 2; attempt++) {
        try {
            attemptResult = await _launcherRef(binary, recruitmentProbeArgs(), dir, budgetMs);
            if (!attemptResult.timedOut)
                break;
            // Only a timeout retries, matching availabilityProbeAttempts's own
            // documented reasoning (pkg/codex/platform_dispatch.go) -- a missing
            // binary or a refused credential fails identically twice and should
            // surface immediately.
        }
        catch {
            attemptResult = undefined;
            break;
        }
    }
    const elapsedMs = Date.now() - start;
    try {
        rmSync(dir, { recursive: true, force: true });
    }
    catch {
        // best-effort cleanup; a removal failure must not fail the probe result
    }
    if (!attemptResult) {
        return { binary, budgetMs, supported: false, verdict: "probe-failed", elapsedMs, probedAt };
    }
    if (attemptResult.timedOut) {
        return { binary, budgetMs, supported: false, verdict: "timeout", elapsedMs, probedAt };
    }
    if (attemptResult.exitCode !== 0) {
        return { binary, budgetMs, supported: false, verdict: "probe-failed", elapsedMs, probedAt };
    }
    const supported = nativeNestingReportedAvailable(attemptResult.output);
    return { binary, budgetMs, supported, verdict: supported ? "available" : "unavailable", elapsedMs, probedAt };
}
