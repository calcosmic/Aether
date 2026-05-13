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

import { spawn } from "node:child_process";
import { existsSync, readFileSync, writeFileSync, mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** Supported platform names. */
export type Platform = "claude" | "opencode" | "codex";

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
export async function detectAvailablePlatforms(): Promise<Platform[]> {
  const available: Platform[] = [];

  for (const platform of ["claude", "opencode", "codex"] as Platform[]) {
    if (await isPlatformAvailable(platform)) {
      available.push(platform);
    }
  }

  return available;
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
        return result.toLowerCase().includes("logged in");
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

/** Build CLI arguments per platform. */
function buildArgs(config: WorkerConfig): string[] {
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
    child.on("close", () => {
      resolve(
        Buffer.concat(stdout).toString("utf-8") +
        Buffer.concat(stderr).toString("utf-8")
      );
    });
  });
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
