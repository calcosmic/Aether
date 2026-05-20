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
export interface GoBridgeOptions {
    /** Absolute path to the aether Go binary. */
    goBinaryPath: string;
    /** Working directory for Go subprocess (typically the repo root). */
    cwd: string;
}
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
export declare function discoverGoBinary(): string;
export declare function callGoJSON<T>(opts: GoBridgeOptions, args: string[]): T;
/** Test-only: inject a mock callGoJSON. */
export declare function __setCallGoJSON(fn: typeof callGoJSON): void;
/** Test-only: restore the real callGoJSON. */
export declare function __restoreCallGoJSON(): void;
/**
 * Assert that the given file path does not fall within Go-owned directories.
 * Throws if the path starts with any entry in GO_OWNED_PATHS.
 *
 * This is the HOST-05 enforcement mechanism ensuring the TS host never
 * writes to .aether/data/ directly.
 */
export declare function assertNoDirectDataWrites(filePath: string): void;
export declare function approvedCompletionDirPrefix(workflow: string): string;
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
export declare function writeCompletionFile(dir: string, filename: string, data: unknown): string;
/**
 * Remove a completion file's temp directory after the Go finalizer reads it.
 *
 * Safe to call with a non-existent path (no-op). Only removes directories
 * that match the expected aether temp dir pattern.
 */
export declare function cleanupCompletionDir(completionPath: string): void;
