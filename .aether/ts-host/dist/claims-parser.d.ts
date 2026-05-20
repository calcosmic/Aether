/**
 * Claims parser for the TypeScript orchestration host.
 *
 * Extracts structured worker claims JSON from CLI stdout for all three
 * platform output formats:
 * - Direct JSON (Codex --json output)
 * - Code-fenced JSON (Claude markdown-wrapped output)
 * - Trailing JSON block (OpenCode or mixed output)
 *
 * Satisfies TS-01 (real worker dispatch).
 */
import type { WorkerHandoff, SpawnClaim } from "./types.js";
/**
 * Worker claims extracted from CLI stdout.
 *
 * Matches the WorkerResult fields used by the Go finalizer.
 */
export interface WorkerClaims {
    /** Worker's assigned name. */
    ant_name?: string;
    /** Worker caste. */
    caste?: string;
    /** Task identifier. */
    task_id?: string;
    /** Terminal status: completed, failed, blocked, timeout. */
    status: string;
    /** Worker's self-reported summary. */
    summary?: string;
    /** Files the worker claims to have created. */
    files_created?: string[];
    /** Files the worker claims to have modified. */
    files_modified?: string[];
    /** Test files the worker created. */
    tests_written?: string[];
    /** Number of tool calls reported. */
    tool_count?: number;
    /** Blocking issues reported. */
    blockers?: string[];
    /**
     * Sub-workers spawned.
     * Accepts both string[] (deprecated) and SpawnClaim[] for backward compatibility.
     * String spawns are deprecated; workers should return structured SpawnClaim objects.
     */
    spawns?: (string | SpawnClaim)[];
    /** Worker handoff relay data. */
    handoff?: WorkerHandoff;
}
/**
 * Parse worker claims from CLI stdout.
 *
 * Tries three strategies in order:
 * 1. Direct JSON.parse on the full stdout
 * 2. Strip markdown code fences and JSON.parse
 * 3. Extract the last JSON object by walking backward from the last `}`
 *
 * @param stdout - Raw stdout from the worker subprocess
 * @returns Parsed and validated WorkerClaims
 * @throws Error if no valid claims JSON can be extracted
 */
export declare function parseWorkerClaims(stdout: string): WorkerClaims;
/**
 * Strip markdown code fences from text.
 *
 * Handles ```json ... ``` and ``` ... ``` blocks.
 *
 * @param text - Text that may contain code fences
 * @returns Inner content with fences removed
 */
export declare function stripCodeFences(text: string): string;
/**
 * Extract the last JSON object from text by walking backward from the last `}`.
 *
 * Finds the matching `{` by tracking brace depth.
 *
 * @param text - Text that may contain a trailing JSON block
 * @returns Extracted JSON string
 * @throws Error if no JSON block can be found
 */
export declare function extractJSONBlock(text: string): string;
/**
 * Validate that an unknown value is a valid WorkerClaims object.
 *
 * Checks required fields and normalizes optional arrays.
 *
 * @param claims - Unknown value to validate
 * @returns Validated WorkerClaims
 * @throws Error if required fields are missing or types are wrong
 */
export declare function validateWorkerClaims(claims: unknown): WorkerClaims;
/**
 * Normalize mixed spawn claims to a uniform SpawnClaim array.
 *
 * Converts legacy string entries to SpawnClaim objects with caste="builder"
 * (default), task=the original string, and an auto-conversion reason.
 * Existing SpawnClaim objects pass through unchanged.
 *
 * @param spawns - Array of string or SpawnClaim entries
 * @returns Normalized array of SpawnClaim objects
 */
export declare function normalizeSpawnClaims(spawns: (string | SpawnClaim)[]): SpawnClaim[];
