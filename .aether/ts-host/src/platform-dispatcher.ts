/**
 * Legacy TypeScript provider launcher retained for regression tests only.
 *
 * Production host code must not import this module. Real provider selection,
 * preflight, launch, and result parsing are owned by Go's hidden
 * `internal-worker-adapter` command.
 *
 * Detects available platform CLIs (claude, opencode, codex), checks
 * authentication, and spawns real worker subprocesses with the correct
 * CLI arguments per platform.
 *
 * Uses spawn (not execFileSync) for async subprocess invocation.
 * Default timeout: 10 minutes via AbortController.
 *
 * @deprecated Test/forensic compatibility only; not a production launcher.
 */

import { spawn } from "node:child_process";
import { existsSync, readFileSync, writeFileSync, mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

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

const canonicalWorkerPlatformOrder: Platform[] = ["codex", "claude", "opencode"];

// ---------------------------------------------------------------------------
// Platform detection
// ---------------------------------------------------------------------------

/**
 * Detect which platform CLIs are available on this machine.
 *
 * Checks PATH via `which` and AETHER_*_PATH environment variables.
 *
 * @returns Array of available platform names
 */
// Mutable reference for test injection.
let _detectAvailablePlatformsRef = _realDetectAvailablePlatforms;

async function _realDetectAvailablePlatforms(): Promise<Platform[]> {
  const available: Platform[] = [];

  for (const platform of ["claude", "opencode", "codex"] as Platform[]) {
    if (await isPlatformAvailable(platform)) {
      available.push(platform);
    }
  }

  return available;
}

export async function detectAvailablePlatforms(): Promise<Platform[]> {
  return _detectAvailablePlatformsRef();
}

/**
 * Select the worker platform for this host run.
 *
 * AETHER_WORKER_PLATFORM is a hard override. Otherwise, prefer the active host
 * platform when known, then fall back to the runtime's canonical provider
 * order. This keeps Codex-invoked host runs on Codex instead of drifting to
 * Claude just because Claude is also installed.
 */
export function selectWorkerPlatform(
  available: readonly Platform[],
  env: NodeJS.ProcessEnv = process.env
): Platform | undefined {
  const override = normalizePlatform(env["AETHER_WORKER_PLATFORM"]);
  if (env["AETHER_WORKER_PLATFORM"]?.trim()) {
    if (!override) return undefined;
    return available.includes(override) ? override : undefined;
  }

  const active =
    normalizePlatform(env["AETHER_ACTIVE_PLATFORM"]) ??
    detectActivePlatformFromEnv(env);
  const preferred = uniquePlatforms([
    ...(active ? [active] : []),
    ...canonicalWorkerPlatformOrder,
  ]);

  for (const platform of preferred) {
    if (available.includes(platform)) {
      return platform;
    }
  }
  return undefined;
}

export function formatWorkerPlatformSelectionMessage(
  available: readonly Platform[],
  env: NodeJS.ProcessEnv = process.env
): string {
  const rawOverride = env["AETHER_WORKER_PLATFORM"]?.trim();
  if (rawOverride) {
    const normalized = normalizePlatform(rawOverride);
    if (!normalized) {
      return `Unsupported AETHER_WORKER_PLATFORM "${rawOverride}". Set it to codex, claude, or opencode.`;
    }
    return `AETHER_WORKER_PLATFORM is set to ${normalized}, but that provider is not available. Available providers: ${available.join(", ") || "none"}.`;
  }
  return `No selectable worker platform is available. Available providers: ${available.join(", ") || "none"}.`;
}

/**
 * Run a tiny worker-provider check before dispatching expensive workers.
 * This catches account/model/provider configuration failures before the host
 * creates worker completion state or spawns expensive worker waves.
 */
export async function preflightWorkerPlatform(
  platform: Platform,
  cwd = process.cwd()
): Promise<void> {
  const binary = resolveBinaryName(platform);
  const args = preflightArgs(platform);
  const result = await runPreflight(binary, args, cwd, "", 20_000);
  if (result.exitCode !== 0) {
    const diagnostic = sanitizeDiagnostic(`${result.stdout}\n${result.stderr}`);
    throw new Error(
      `${providerDisplayName(platform)} provider/model preflight failed before worker dispatch: ${diagnostic || `${platform} exited with status ${result.exitCode ?? "unknown"}`}`
    );
  }
}

function preflightArgs(platform: Platform): string[] {
  switch (platform) {
    case "claude":
      return [
        "-p",
        "--output-format", "json",
        "--permission-mode", "bypassPermissions",
        "Return exactly OK.",
      ];
    case "opencode":
      return [
        "run",
        "--agent", process.env["AETHER_OPENCODE_PRIMARY_AGENT"]?.trim() || "build",
        "--format", "json",
        "Return exactly OK.",
      ];
    case "codex":
      return [
        "--sandbox", "workspace-write",
        "--ask-for-approval", "never",
        "exec",
        "--json",
        "--ephemeral",
        "--skip-git-repo-check",
        "Return exactly OK.",
      ];
  }
}

function providerDisplayName(platform: Platform): string {
  switch (platform) {
    case "claude":
      return "Claude Code";
    case "opencode":
      return "OpenCode";
    case "codex":
      return "Codex";
  }
}

/**
 * Return the TS-host-facing unavailable-provider message.
 *
 * Detailed provider classification is owned by Go's AvailabilityStatus
 * contract. The TS host keeps only the legacy boolean preflight here, then
 * tells users how to fetch the Go-owned diagnostic instead of inventing a
 * second auth vocabulary.
 */
export function formatPlatformUnavailableMessage(
  context = "worker dispatch",
  providerDiagnostics?: string
): string {
  // When Go provides diagnostics, use them as the primary message (D-03).
  if (providerDiagnostics && providerDiagnostics.trim()) {
    return providerDiagnostics.trim();
  }
  // Generic fallback when no Go diagnostics are available.
  return (
    `Worker dispatch cannot start for ${context}; the TypeScript host did not receive ` +
    "a Go provider_diagnostics value. Detailed provider availability and fallback " +
    "diagnostics are owned by the Go AvailabilityStatus contract; use the Go result " +
    "or run `aether status` to inspect the runtime-owned diagnostic."
  );
}

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
export function formatPlatformDiagnosticMessage(platform: Platform): string {
  switch (platform) {
    case "claude":
      return "Claude Code is not installed or not available on your PATH. Install it from claude.ai/code and try again.";
    case "opencode":
      return "OpenCode is not installed or not available on your PATH. Install it from opencode.ai and try again.";
    case "codex":
      return "Codex CLI is not installed or not available on your PATH. Install it from github.com/openai/codex and try again.";
  }
}

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
export function classifyPlatformError(
  _platform: string,
  error: unknown
): PlatformErrorClass {
  const message =
    error instanceof Error ? error.message.toLowerCase() : String(error).toLowerCase();

  if (/auth|\bcredentials\b|\b(log\s*in|logged\s*in)\b|\bpermission\b/.test(message)) {
    return "auth";
  }
  if (/\b(timeout|timed?\s*out|etimedout|abort_err)\b/.test(message)) {
    return "timeout";
  }
  if (/\b(enoent|not found|which)\b/.test(message)) {
    return "missing";
  }
  return "unknown";
}

/** Test-only: inject a mock detectAvailablePlatforms. */
export function __setDetectAvailablePlatforms(fn: typeof detectAvailablePlatforms): void {
  _detectAvailablePlatformsRef = fn;
}

/** Test-only: restore the real detectAvailablePlatforms. */
export function __restoreDetectAvailablePlatforms(): void {
  _detectAvailablePlatformsRef = _realDetectAvailablePlatforms;
}

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
export async function isPlatformAvailable(platform: Platform): Promise<boolean> {
  const binary = resolveBinaryName(platform);

  // Check binary exists on PATH
  try {
    const { spawnSync } = await import("node:child_process");
    const result = spawnSync("which", [binary], { encoding: "utf-8", timeout: 5000 });
    if (result.status !== 0 || !result.stdout.trim()) {
      return false;
    }
  } catch {
    return false;
  }

  // Check authentication
  try {
    switch (platform) {
      case "claude": {
        const result = await runProbe(binary, ["auth", "status", "--json"]);
        const parsed = JSON.parse(result) as { loggedIn?: boolean };
        return parsed.loggedIn === true;
      }
      case "opencode": {
        const result = await runProbe(binary, ["auth", "list"]);
        return countOpenCodeCredentials(result) > 0;
      }
      case "codex": {
        const result = await runProbe(binary, ["login", "status"]);
        return codexLoginStatusIsActive(result);
      }
    }
  } catch {
    return false;
  }
}

// ---------------------------------------------------------------------------
// Worker spawning
// ---------------------------------------------------------------------------

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
export async function spawnWorker(config: WorkerConfig): Promise<SpawnResult> {
  const binary = resolveBinaryName(config.platform);
  const start = Date.now();

  const args = buildArgs(config);
  const abortController = new AbortController();
  const timeoutMs = 10 * 60 * 1000; // 10 minutes
  const timeoutId = setTimeout(() => abortController.abort(), timeoutMs);

  return new Promise<SpawnResult>((resolve) => {
    const stdoutChunks: Buffer[] = [];
    const stderrChunks: Buffer[] = [];

    const child = spawn(binary, args, {
      cwd: config.root,
      signal: abortController.signal,
      env: { ...process.env },
    });

    child.stdout?.on("data", (chunk: Buffer) => {
      stdoutChunks.push(chunk);
    });

    child.stderr?.on("data", (chunk: Buffer) => {
      stderrChunks.push(chunk);
    });

    child.on("error", (err: Error) => {
      clearTimeout(timeoutId);
      const duration = Date.now() - start;
      if ((err as NodeJS.ErrnoException).code === "ABORT_ERR") {
        resolve({
          exitCode: null,
          stdout: Buffer.concat(stdoutChunks).toString("utf-8"),
          stderr: Buffer.concat(stderrChunks).toString("utf-8"),
          duration,
        });
        return;
      }
      resolve({
        exitCode: null,
        stdout: Buffer.concat(stdoutChunks).toString("utf-8"),
        stderr: `${Buffer.concat(stderrChunks).toString("utf-8")}\n${err.message}`,
        duration,
      });
    });

    child.on("close", (exitCode) => {
      clearTimeout(timeoutId);
      const duration = Date.now() - start;
      resolve({
        exitCode,
        stdout: Buffer.concat(stdoutChunks).toString("utf-8"),
        stderr: Buffer.concat(stderrChunks).toString("utf-8"),
        duration,
      });
    });
  });
}

/**
 * Create a platform-specific dispatcher object.
 *
 * @param platform - Target platform
 * @returns Object with spawnWorker method bound to the platform
 */
export function createPlatformDispatcher(platform: Platform) {
  return {
    platform,
    spawnWorker: (config: WorkerConfig) => spawnWorker({ ...config, platform }),
  };
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

/** Resolve binary name from env var or default. */
function resolveBinaryName(platform: Platform): string {
  switch (platform) {
    case "claude":
      return process.env["AETHER_CLAUDE_PATH"]?.trim() || "claude";
    case "opencode":
      return process.env["AETHER_OPENCODE_PATH"]?.trim() || "opencode";
    case "codex":
      return process.env["AETHER_CODEX_PATH"]?.trim() || "codex";
  }
}

function normalizePlatform(raw: string | undefined): Platform | undefined {
  switch ((raw ?? "").trim().toLowerCase()) {
    case "codex":
    case "codex-cli":
      return "codex";
    case "claude":
    case "claude-code":
      return "claude";
    case "opencode":
    case "open-code":
      return "opencode";
    default:
      return undefined;
  }
}

function detectActivePlatformFromEnv(env: NodeJS.ProcessEnv): Platform | undefined {
  if (hasAnyEnv(env, ["CODEX_THREAD_ID", "CODEX_SESSION_ID", "CODEX_CI"])) {
    return "codex";
  }
  if (
    hasEnvPrefix(env, "CLAUDE_CODE_") ||
    hasAnyEnv(env, ["CLAUDECODE", "CLAUDECODE_PROJECT_DIR", "CLAUDE_PROJECT_DIR", "CLAUDE_CODE_SIMPLE"])
  ) {
    return "claude";
  }
  if (hasEnvPrefix(env, "OPENCODE_")) {
    return "opencode";
  }
  return undefined;
}

function hasAnyEnv(env: NodeJS.ProcessEnv, keys: readonly string[]): boolean {
  return keys.some((key) => {
    const value = env[key];
    return typeof value === "string" && value.trim() !== "" && value !== "0";
  });
}

function hasEnvPrefix(env: NodeJS.ProcessEnv, prefix: string): boolean {
  for (const [key, value] of Object.entries(env)) {
    if (key.startsWith(prefix) && typeof value === "string" && value.trim() !== "" && value !== "0") {
      return true;
    }
  }
  return false;
}

function uniquePlatforms(platforms: readonly Platform[]): Platform[] {
  const seen = new Set<Platform>();
  const out: Platform[] = [];
  for (const platform of platforms) {
    if (!seen.has(platform)) {
      seen.add(platform);
      out.push(platform);
    }
  }
  return out;
}

/**
 * Build CLI arguments per platform.
 * @internal — exported for testing only
 */
export function buildArgs(config: WorkerConfig): string[] {
  switch (config.platform) {
    case "claude": {
      const schemaJSON = JSON.stringify(workerClaimsSchema());
      return [
        "-p",
        "--output-format", "json",
        "--json-schema", schemaJSON,
        "--agent", config.agentName,
        "--permission-mode", "bypassPermissions",
        config.prompt,
      ];
    }
    case "opencode": {
      return [
        "run",
        "--agent", config.agentName,
        "--format", "json",
        config.prompt,
      ];
    }
    case "codex": {
      // Write schema to a temp file (not .aether/data/)
      const schemaDir = mkdtempSync(join(tmpdir(), "aether-codex-schema-"));
      const schemaPath = join(schemaDir, "schema.json");
      writeFileSync(schemaPath, JSON.stringify(workerClaimsSchema(), null, 2), "utf-8");

      return [
        "--sandbox", "workspace-write",
        "--ask-for-approval", "never",
        "exec",
        "--json",
        "--ephemeral",
        "--skip-git-repo-check",
        "--output-schema", schemaPath,
        config.prompt,
      ];
    }
  }
}

/** Run a short-lived probe command and return combined output. */
async function runProbe(binary: string, args: string[]): Promise<string> {
  return new Promise((resolve, reject) => {
    const child = spawn(binary, args, { timeout: 5000 });
    const stdout: Buffer[] = [];
    const stderr: Buffer[] = [];

    child.stdout?.on("data", (chunk: Buffer) => stdout.push(chunk));
    child.stderr?.on("data", (chunk: Buffer) => stderr.push(chunk));

    child.on("error", reject);
    child.on("close", (exitCode) => {
      const output =
        Buffer.concat(stdout).toString("utf-8") +
        Buffer.concat(stderr).toString("utf-8");
      if (exitCode === 0) {
        resolve(output);
        return;
      }
      reject(new Error(`probe exited with status ${exitCode ?? "unknown"}`));
    });
  });
}

async function runPreflight(
  binary: string,
  args: string[],
  cwd: string,
  stdin: string,
  timeoutMs: number
): Promise<SpawnResult> {
  const start = Date.now();
  const abortController = new AbortController();
  const timeoutId = setTimeout(() => abortController.abort(), timeoutMs);

  return new Promise((resolve) => {
    const stdout: Buffer[] = [];
    const stderr: Buffer[] = [];
    const child = spawn(binary, args, {
      cwd,
      signal: abortController.signal,
      env: { ...process.env },
    });
    child.stdout?.on("data", (chunk: Buffer) => stdout.push(chunk));
    child.stderr?.on("data", (chunk: Buffer) => stderr.push(chunk));
    child.on("error", (err: Error) => {
      clearTimeout(timeoutId);
      resolve({
        exitCode: null,
        stdout: Buffer.concat(stdout).toString("utf-8"),
        stderr: `${Buffer.concat(stderr).toString("utf-8")}\n${err.message}`,
        duration: Date.now() - start,
      });
    });
    child.on("close", (exitCode) => {
      clearTimeout(timeoutId);
      resolve({
        exitCode,
        stdout: Buffer.concat(stdout).toString("utf-8"),
        stderr: Buffer.concat(stderr).toString("utf-8"),
        duration: Date.now() - start,
      });
    });
    child.stdin?.end(stdin);
  });
}

function sanitizeDiagnostic(raw: string): string {
  let text = raw.replace(/sk-[A-Za-z0-9_-]+/g, "[redacted]");
  text = text.replace(/gh[pousr]_[A-Za-z0-9_]+/g, "[redacted]");
  text = text.replace(/\s+/g, " ").trim();
  if (text.length > 400) {
    text = `${text.slice(0, 397)}...`;
  }
  return text;
}

/** Count OpenCode credential entries from auth list output. */
function countOpenCodeCredentials(output: string): number {
  let count = 0;
  for (const line of output.split("\n")) {
    if (line.trim().startsWith("●")) {
      count++;
    }
  }
  return count;
}

/** Return true only when Codex explicitly reports an active login. */
function codexLoginStatusIsActive(output: string): boolean {
  const cleaned = output.toLowerCase();
  if (!cleaned.includes("logged in")) {
    return false;
  }
  if (/\b(not|no|never|without)\b.{0,40}\blogged in\b|\bnot authenticated\b|\bunauthenticated\b|\bno authenticated session\b/.test(cleaned)) {
    return false;
  }
  return /\blogged in\b/.test(cleaned);
}

/** Build the worker claims JSON schema used for --json-schema args. */
function workerClaimsSchema(): Record<string, unknown> {
  const stringArray = {
    type: "array",
    items: { type: "string" },
  };
  return {
    type: "object",
    additionalProperties: false,
    required: [
      "ant_name",
      "caste",
      "task_id",
      "status",
      "summary",
      "files_created",
      "files_modified",
      "tests_written",
      "tool_count",
      "blockers",
      "spawns",
      "handoff",
    ],
    properties: {
      ant_name: { type: "string" },
      caste: { type: "string" },
      task_id: { type: "string" },
      status: { type: "string", enum: ["completed", "code_written", "failed", "blocked"] },
      summary: { type: "string" },
      files_created: stringArray,
      files_modified: stringArray,
      tests_written: stringArray,
      tool_count: { type: "integer", minimum: 0 },
      blockers: stringArray,
      spawns: stringArray,
      handoff: {
        type: "object",
        additionalProperties: false,
        required: [
          "changed_files",
          "commands_run",
          "verification_status",
          "known_failures",
          "open_decisions",
          "assumptions",
          "next_worker_instructions",
          "do_not_repeat",
          "freshness",
        ],
        properties: {
          changed_files: stringArray,
          commands_run: stringArray,
          verification_status: { type: "string", enum: ["pass", "fail", "partial", "not_run", "unknown"] },
          known_failures: stringArray,
          open_decisions: stringArray,
          assumptions: stringArray,
          next_worker_instructions: stringArray,
          do_not_repeat: stringArray,
          freshness: { type: "string" },
        },
      },
    },
  };
}
