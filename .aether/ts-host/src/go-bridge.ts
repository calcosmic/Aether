/**
 * Go subprocess invocation bridge for the TypeScript orchestration host.
 *
 * Provides helpers to call the Go CLI binary with AETHER_OUTPUT_MODE=json
 * and parse the structured JSON output. All Go CLI communication goes through
 * this module.
 *
 * Contract: cmd/helpers.go outputOK produces {"ok":true,"result":<data>}
 * and outputError produces {"ok":false,"error":"msg","code":N}.
 */

import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, mkdtempSync, writeFileSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";
import { tmpdir } from "node:os";

import { GO_OWNED_PATHS } from "./boundary-reference.js";
import type { GoOutput } from "./types.js";

// ---------------------------------------------------------------------------
// Options
// ---------------------------------------------------------------------------

export interface GoBridgeOptions {
  /** Absolute path to the aether Go binary. */
  goBinaryPath: string;
  /** Working directory for Go subprocess (typically the repo root). */
  cwd: string;
}

// ---------------------------------------------------------------------------
// Binary discovery
// ---------------------------------------------------------------------------

/**
 * Discover the aether Go binary path.
 *
 * Search order:
 * 1. AETHER_BINARY_PATH environment variable
 * 2. `which aether` (PATH lookup)
 * 3. $HOME/.local/bin/aether (default install location)
 *
 * @throws Error if no binary found
 */
export function discoverGoBinary(): string {
  // 1. Explicit env var
  const envPath = process.env["AETHER_BINARY_PATH"];
  if (envPath) {
    return envPath;
  }

  // 2. PATH lookup via which
  try {
    const whichResult = execFileSync("which", ["aether"], {
      encoding: "utf-8",
      timeout: 5000,
    }).trim();
    if (whichResult) {
      return whichResult;
    }
  } catch {
    // which failed, continue to fallback
  }

  // 3. Default install location
  const defaultPath = join(homedir(), ".local", "bin", "aether");
  if (existsSync(defaultPath)) {
    return defaultPath;
  }

  throw new Error(
    "Cannot locate aether binary. Set AETHER_BINARY_PATH, add aether to PATH, " +
      "or install to $HOME/.local/bin/aether"
  );
}

// ---------------------------------------------------------------------------
// Go JSON invocation
// ---------------------------------------------------------------------------

/**
 * Call a Go CLI command with AETHER_OUTPUT_MODE=json and return the parsed result.
 *
 * Uses execFileSync (no shell interpolation) for security. The Go binary
 * validates all string inputs server-side.
 *
 * @param opts - Bridge options (binary path and cwd)
 * @param args - CLI arguments (e.g., ["build", "1", "--plan-only"])
 * @returns Parsed result from Go's {"ok":true,"result":<data>} envelope
 * @throws Error if Go returns an error envelope or subprocess fails
 */
export function callGoJSON<T>(opts: GoBridgeOptions, args: string[]): T {
  let raw: string;
  try {
    raw = execFileSync(opts.goBinaryPath, args, {
      cwd: opts.cwd,
      env: { ...process.env, AETHER_OUTPUT_MODE: "json" },
      encoding: "utf-8",
      maxBuffer: 10 * 1024 * 1024, // 10 MB safety limit
    });
  } catch (err: unknown) {
    // Surface ENOENT and spawn errors clearly
    const message =
      err instanceof Error ? err.message : String(err);
    throw new Error(
      `Go subprocess failed for ${args.join(" ")}: ${message}`
    );
  }

  let parsed: GoOutput<T>;
  try {
    parsed = JSON.parse(raw) as GoOutput<T>;
  } catch {
    throw new Error(
      `Go subprocess returned invalid JSON for ${args.join(" ")}: ` +
        raw.slice(0, 200)
    );
  }

  // Go outputError writes {"ok":false,"error":"msg","code":N} to stderr,
  // but some commands write errors to stdout too. Check both ok flag and
  // error field.
  if (!parsed.ok || parsed.error) {
    throw new Error(
      `Go command failed: ${args.join(" ")}: ${parsed.error ?? "unknown error (ok=false)"}`
    );
  }

  return parsed.result as T;
}

// ---------------------------------------------------------------------------
// Boundary enforcement (HOST-05)
// ---------------------------------------------------------------------------

/**
 * Assert that the given file path does not fall within Go-owned directories.
 * Throws if the path starts with any entry in GO_OWNED_PATHS.
 *
 * This is the HOST-05 enforcement mechanism ensuring the TS host never
 * writes to .aether/data/ directly.
 */
export function assertNoDirectDataWrites(filePath: string): void {
  const normalized = filePath.replace(/\\/g, "/");
  // Resolve .. segments to prevent traversal bypass
  const resolved = normalized.split("/").reduce<string[]>((acc, segment) => {
    if (segment === "..") acc.pop();
    else if (segment !== ".") acc.push(segment);
    return acc;
  }, []).join("/");
  for (const goPath of GO_OWNED_PATHS) {
    if (resolved.startsWith(goPath) || resolved.includes("/" + goPath)) {
      throw new Error(
        `Boundary violation: TS host must not write to Go-owned path "${goPath}". ` +
          `Path attempted: "${filePath}". Use Go finalizer commands instead.`
      );
    }
  }
}

// ---------------------------------------------------------------------------
// Completion file helper
// ---------------------------------------------------------------------------

/**
 * Write a completion JSON file to the system temp directory (NOT .aether/data/).
 *
 * The Go finalizer reads this file, validates provenance, and commits state
 * atomically. The TS host never writes completion files to Go-owned paths.
 *
 * @param dir - Subdirectory within tmpdir (e.g., "aether-completions")
 * @param filename - File name (e.g., "build-completion.json")
 * @param data - Serializable data to write as JSON
 * @returns Absolute path to the written file
 */
export function writeCompletionFile(
  dir: string,
  filename: string,
  data: unknown
): string {
  // Validate prefix does not escape tmpdir before creating the unique directory
  const prefix = join(tmpdir(), `${dir}-`);
  const resolvedPrefix = prefix.replace(/\\/g, "/");
  const resolvedTmpdir = tmpdir().replace(/\\/g, "/");
  if (!resolvedPrefix.startsWith(resolvedTmpdir)) {
    throw new Error(
      `Completion file path escapes tmpdir: ${prefix}`
    );
  }

  // Create a unique temp directory per run to avoid collisions
  const targetDir = mkdtempSync(prefix);
  const targetPath = join(targetDir, filename);

  // Boundary enforcement: reject Go-owned paths
  assertNoDirectDataWrites(targetPath);

  writeFileSync(targetPath, JSON.stringify(data, null, 2), "utf-8");
  return targetPath;
}
