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
/** Default probe budget in milliseconds, mirroring recruitmentProbeDefaultTimeout (cmd/recruitment_probe.go). */
export declare const RECRUITMENT_PROBE_DEFAULT_TIMEOUT_MS = 10000;
/** Test-only: reset the once-per-value warning dedupe. */
export declare function __resetRecruitmentProbeTimeoutWarnings(): void;
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
export declare function resolveRecruitmentProbeTimeoutMs(): number;
/** One native-nesting probe run's outcome. Verdict is decided only from the
 * probed child's own reported answer -- never the platform name, a version
 * string, or a hardcoded table. */
export interface RecruitmentProbeResult {
    binary: string;
    budgetMs: number;
    supported: boolean;
    verdict: string;
    elapsedMs: number;
    probedAt: string;
}
/** One probe launch's raw outcome, before verdict classification. */
export interface RecruitmentProbeLaunchResult {
    output: string;
    timedOut: boolean;
    exitCode: number | null;
}
/** Test-only injection point for the process launcher, mirroring
 * cmd/recruitment_probe.go's recruitmentProbeRunner var -- lets a test
 * substitute a deterministic, counting fake without depending on an
 * installed platform CLI or a real model round-trip. */
export type RecruitmentProbeLauncher = (binary: string, args: string[], cwd: string, timeoutMs: number) => Promise<RecruitmentProbeLaunchResult>;
/** Test-only: substitute a deterministic launcher. */
export declare function __setRecruitmentProbeLauncher(fn: RecruitmentProbeLauncher): void;
/** Test-only: restore the real launcher. */
export declare function __restoreRecruitmentProbeLauncher(): void;
/** Test-only: clear the session cache so a test can exercise both the first
 * (real launch) and the cached (no launch) call. */
export declare function __resetRecruitmentProbeCache(): void;
/**
 * Answer "can a helper nest natively here?" by trying it, cheaply, once, in
 * a temp directory outside the repository. Retries exactly once, and only
 * on a timeout -- a launch error (missing binary, refused credential) fails
 * immediately. The result is cached for the lifetime of this process: the
 * second and later calls resolve immediately with no new launch, so a
 * session that recruits ten times pays for one probe and a session that
 * never recruits pays for none.
 */
export declare function probeNativeNesting(): Promise<RecruitmentProbeResult>;
