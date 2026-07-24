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
import { execFile, execFileSync } from "node:child_process";
import { existsSync, mkdtempSync, writeFileSync, rmSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { GO_OWNED_PATHS } from "./boundary-reference.js";
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
export function discoverGoBinary() {
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
    }
    catch {
        // which failed, continue to fallback
    }
    // 3. Default install location
    const defaultPath = join(homedir(), ".local", "bin", "aether");
    if (existsSync(defaultPath)) {
        return defaultPath;
    }
    throw new Error("Cannot locate aether binary. Set AETHER_BINARY_PATH, add aether to PATH, " +
        "or install to $HOME/.local/bin/aether");
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
// Mutable reference for test injection.
let _callGoJSONRef = _realCallGoJSON;
function _realCallGoJSON(opts, args) {
    let raw;
    try {
        raw = execFileSync(opts.goBinaryPath, args, {
            cwd: opts.cwd,
            env: { ...process.env, AETHER_OUTPUT_MODE: "json" },
            encoding: "utf-8",
            maxBuffer: 10 * 1024 * 1024, // 10 MB safety limit
            stdio: ["ignore", "pipe", "pipe"],
        });
    }
    catch (err) {
        const envelope = parseGoErrorEnvelope(err);
        if (envelope?.error) {
            throw new Error(`Go command failed: ${formatGoCommand(args)}: ${sanitizeBridgeMessage(envelope.error)}`);
        }
        throw new Error(`Go subprocess failed for ${formatGoCommand(args)}: ${subprocessFailureDetail(err)}; subprocess output omitted`);
    }
    let parsed;
    try {
        parsed = JSON.parse(raw);
    }
    catch {
        throw new Error(`Go subprocess returned invalid JSON for ${formatGoCommand(args)}; subprocess output omitted`);
    }
    // Go outputError writes {"ok":false,"error":"msg","code":N} to stderr,
    // but some commands write errors to stdout too. Check both ok flag and
    // error field.
    if (!parsed.ok || parsed.error) {
        throw new Error(`Go command failed: ${formatGoCommand(args)}: ${sanitizeBridgeMessage(parsed.error ?? "unknown error (ok=false)")}`);
    }
    return parsed.result;
}
function formatGoCommand(args) {
    const command = args.slice(0, 3).join(" ").trim();
    return command || "aether";
}
function parseGoErrorEnvelope(err) {
    const output = [
        bufferLikeToString(err?.stdout),
        bufferLikeToString(err?.stderr),
    ].find((value) => value.trim().startsWith("{"));
    if (!output) {
        return undefined;
    }
    try {
        return JSON.parse(output);
    }
    catch {
        return undefined;
    }
}
function bufferLikeToString(value) {
    if (typeof value === "string") {
        return value;
    }
    if (Buffer.isBuffer(value)) {
        return value.toString("utf-8");
    }
    return "";
}
function subprocessFailureDetail(err) {
    const typed = err;
    if (typeof typed.code === "string" && typed.code.trim() !== "") {
        return typed.code === "ENOENT" ? "binary not found" : typed.code;
    }
    if (typeof typed.status === "number") {
        return `exit status ${typed.status}`;
    }
    if (typeof typed.signal === "string" && typed.signal.trim() !== "") {
        return `signal ${typed.signal}`;
    }
    return "execution failed";
}
function sanitizeBridgeMessage(message) {
    return message
        .replace(/sk-[A-Za-z0-9_-]+/g, "[redacted]")
        .replace(/ghp_[A-Za-z0-9_]+/g, "[redacted]")
        .replace(/\btoken\s+[A-Za-z0-9._-]+/gi, "token [redacted]")
        .replace(/\b(stdout|stderr):\s*[^\n;]+/gi, "$1: [omitted]");
}
export function callGoJSON(opts, args) {
    return _callGoJSONRef(opts, args);
}
/**
 * Asynchronously call a Go CLI command and parse its JSON envelope.
 *
 * Worker dispatch must use the async bridge so Promise-based wave concurrency
 * is real. A synchronous child process here would serialize every worker even
 * when the wave orchestrator uses Promise.all.
 */
let _callGoJSONAsyncRef = _realCallGoJSONAsync;
function _realCallGoJSONAsync(opts, args, timeoutMs = 0) {
    return new Promise((resolve, reject) => {
        execFile(opts.goBinaryPath, args, {
            cwd: opts.cwd,
            env: { ...process.env, AETHER_OUTPUT_MODE: "json" },
            encoding: "utf-8",
            maxBuffer: 10 * 1024 * 1024,
            timeout: timeoutMs > 0 ? timeoutMs : undefined,
            killSignal: "SIGTERM",
        }, (err, stdout, stderr) => {
            if (err) {
                const envelope = parseGoErrorText(stdout, stderr);
                if (envelope?.error) {
                    reject(new Error(`Go command failed: ${formatGoCommand(args)}: ${sanitizeBridgeMessage(envelope.error)}`));
                    return;
                }
                reject(new Error(`Go subprocess failed for ${formatGoCommand(args)}: ${subprocessFailureDetail(err)}; subprocess output omitted`));
                return;
            }
            let parsed;
            try {
                parsed = JSON.parse(stdout);
            }
            catch {
                reject(new Error(`Go subprocess returned invalid JSON for ${formatGoCommand(args)}; subprocess output omitted`));
                return;
            }
            if (!parsed.ok || parsed.error) {
                reject(new Error(`Go command failed: ${formatGoCommand(args)}: ${sanitizeBridgeMessage(parsed.error ?? "unknown error (ok=false)")}`));
                return;
            }
            resolve(parsed.result);
        });
    });
}
function parseGoErrorText(stdout, stderr) {
    const output = [stdout, stderr].find((value) => value.trim().startsWith("{"));
    if (!output)
        return undefined;
    try {
        return JSON.parse(output);
    }
    catch {
        return undefined;
    }
}
export function callGoJSONAsync(opts, args, timeoutMs = 0) {
    return _callGoJSONAsyncRef(opts, args, timeoutMs);
}
/** Test-only: inject a mock callGoJSON. */
export function __setCallGoJSON(fn) {
    _callGoJSONRef = fn;
}
/** Test-only: restore the real callGoJSON. */
export function __restoreCallGoJSON() {
    _callGoJSONRef = _realCallGoJSON;
}
/** Test-only: inject a mock asynchronous Go bridge. */
export function __setCallGoJSONAsync(fn) {
    _callGoJSONAsyncRef = fn;
}
/** Test-only: restore the real asynchronous Go bridge. */
export function __restoreCallGoJSONAsync() {
    _callGoJSONAsyncRef = _realCallGoJSONAsync;
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
export function assertNoDirectDataWrites(filePath) {
    const normalized = filePath.replace(/\\/g, "/");
    // Resolve .. segments to prevent traversal bypass
    const resolved = normalized.split("/").reduce((acc, segment) => {
        if (segment === "..")
            acc.pop();
        else if (segment !== ".")
            acc.push(segment);
        return acc;
    }, []).join("/");
    for (const goPath of GO_OWNED_PATHS) {
        if (resolved.startsWith(goPath) || resolved.includes("/" + goPath)) {
            throw new Error(`Boundary violation: TS host must not write to Go-owned path "${goPath}". ` +
                `Path attempted: "${filePath}". Use Go finalizer commands instead.`);
        }
    }
}
// ---------------------------------------------------------------------------
// Completion file helper
// ---------------------------------------------------------------------------
export function approvedCompletionDirPrefix(workflow) {
    const normalized = workflow.trim().toLowerCase().replace(/[^a-z0-9_-]+/g, "-");
    if (normalized === "") {
        throw new Error("workflow is required for completion file path");
    }
    return `aether-${normalized}`;
}
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
export function writeCompletionFile(dir, filename, data) {
    // Validate prefix does not escape tmpdir before creating the unique directory
    const prefix = join(tmpdir(), `${dir}-`);
    const resolvedPrefix = prefix.replace(/\\/g, "/");
    const resolvedTmpdir = tmpdir().replace(/\\/g, "/");
    if (!resolvedPrefix.startsWith(resolvedTmpdir)) {
        throw new Error(`Completion file path escapes tmpdir: ${prefix}`);
    }
    // Create a unique temp directory per run to avoid collisions
    const targetDir = mkdtempSync(prefix);
    const targetPath = join(targetDir, filename);
    // Boundary enforcement: reject Go-owned paths
    assertNoDirectDataWrites(targetPath);
    writeFileSync(targetPath, JSON.stringify(data, null, 2), "utf-8");
    return targetPath;
}
/**
 * Remove a completion file's temp directory after the Go finalizer reads it.
 *
 * Safe to call with a non-existent path (no-op). Only removes directories
 * that match the expected aether temp dir pattern.
 */
export function cleanupCompletionDir(completionPath) {
    if (!existsSync(completionPath))
        return;
    const dir = join(completionPath, "..");
    const resolved = dir === ".." ? "" : dir;
    const base = resolved.split("/").pop() ?? "";
    if (base.startsWith("aether-")) {
        rmSync(resolved, { recursive: true, force: true });
    }
}
