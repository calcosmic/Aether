/**
 * Platform dispatcher for the TypeScript orchestration host.
 *
 * Detects available platform CLIs (claude, opencode, codex), checks
 * authentication, and spawns real worker subprocesses with the correct
 * CLI arguments per platform.
 *
 * Uses spawn (not execFileSync) for async subprocess invocation.
 * Default timeout: 10 minutes via AbortController.
 *
 * Satisfies TS-01 (real worker dispatch).
 */
/** Supported platform names. */
export type Platform = "claude" | "opencode" | "codex";
/** Error classification for platform dispatch failures. */
export type PlatformErrorClass = "auth" | "missing" | "timeout" | "unknown";
/** Configuration for spawning a single worker. */
export interface WorkerConfig {
    /** Target platform. */
    platform: Platform;
    /** Agent name (e.g. "aether-builder"). */
    agentName: string;
    /** Worker caste (builder, watcher, etc.). */
    caste: string;
    /** Deterministic worker name. */
    name: string;
    /** Task description. */
    task: string;
    /** Repository root (working directory for subprocess). */
    root: string;
    /** Fully assembled worker prompt. */
    prompt: string;
}
/** Result of a spawned worker subprocess. */
export interface SpawnResult {
    /** Process exit code (null if killed/timed out). */
    exitCode: number | null;
    /** Combined stdout from the subprocess. */
    stdout: string;
    /** Combined stderr from the subprocess. */
    stderr: string;
    /** Wall-clock duration in milliseconds. */
    duration: number;
}
export declare function detectAvailablePlatforms(): Promise<Platform[]>;
/**
 * Return the TS-host-facing unavailable-provider message.
 *
 * Detailed provider classification is owned by Go's AvailabilityStatus
 * contract. The TS host keeps only the legacy boolean preflight here, then
 * tells users how to fetch the Go-owned diagnostic instead of inventing a
 * second auth vocabulary.
 */
export declare function formatPlatformUnavailableMessage(context?: string, providerDiagnostics?: string): string;
/**
 * Produce a per-platform plain English error message when a platform CLI
 * is not installed or not available on PATH.
 *
 * Per D-03, messages must be plain English -- no error codes, file paths,
 * or jargon. Go runtime's platform-diagnostic is used behind the scenes,
 * but the user never sees raw diagnostic output.
 *
 * @param platform - The platform that is missing
 * @returns Plain English error message
 */
export declare function formatPlatformDiagnosticMessage(platform: Platform): string;
/**
 * Classify a platform dispatch error into categories that drive error handling.
 *
 * - "auth" errors halt the build immediately (D-01)
 * - "timeout" and "unknown" errors mark the worker as failed and continue (D-02)
 * - "missing" errors indicate the CLI is not installed
 *
 * @param _platform - The platform name (reserved for platform-specific heuristics)
 * @param error - The error to classify
 * @returns Error classification
 */
export declare function classifyPlatformError(_platform: string, error: unknown): PlatformErrorClass;
/** Test-only: inject a mock detectAvailablePlatforms. */
export declare function __setDetectAvailablePlatforms(fn: typeof detectAvailablePlatforms): void;
/** Test-only: restore the real detectAvailablePlatforms. */
export declare function __restoreDetectAvailablePlatforms(): void;
/**
 * Check whether a specific platform is available and authenticated.
 *
 * - Claude: `claude auth status --json` must report loggedIn=true
 * - OpenCode: `opencode auth list` must report non-empty credentials
 * - Codex: `codex login status` must exit 0 and contain "logged in"
 *
 * @param platform - Platform to check
 * @returns true if the platform binary exists and is authenticated
 */
export declare function isPlatformAvailable(platform: Platform): Promise<boolean>;
/**
 * Spawn a worker subprocess for the given platform.
 *
 * Builds platform-specific CLI arguments, collects stdout/stderr,
 * measures duration, and enforces a default 10-minute timeout via
 * AbortController.
 *
 * @param config - Worker configuration
 * @returns Spawn result with exit code, output, and duration
 */
export declare function spawnWorker(config: WorkerConfig): Promise<SpawnResult>;
/**
 * Create a platform-specific dispatcher object.
 *
 * @param platform - Target platform
 * @returns Object with spawnWorker method bound to the platform
 */
export declare function createPlatformDispatcher(platform: Platform): {
    platform: Platform;
    spawnWorker: (config: WorkerConfig) => Promise<SpawnResult>;
};
/**
 * Build CLI arguments per platform.
 * @internal — exported for testing only
 */
export declare function buildArgs(config: WorkerConfig): string[];
