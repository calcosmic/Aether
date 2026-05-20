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
// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------
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
export function parseWorkerClaims(stdout) {
    const trimmed = stdout.trim();
    if (trimmed === "") {
        throw new Error("Failed to parse worker claims from output: empty output");
    }
    // Strategy 1: direct JSON.parse
    try {
        const parsed = JSON.parse(trimmed);
        return normalizeClaimsSpawns(validateWorkerClaims(parsed));
    }
    catch {
        // Continue to next strategy
    }
    // Strategy 2: strip code fences
    try {
        const stripped = stripCodeFences(trimmed);
        if (stripped !== trimmed) {
            const parsed = JSON.parse(stripped);
            return normalizeClaimsSpawns(validateWorkerClaims(parsed));
        }
    }
    catch {
        // Continue to next strategy
    }
    // Strategy 3: extract trailing JSON block
    try {
        const extracted = extractJSONBlock(trimmed);
        const parsed = JSON.parse(extracted);
        return normalizeClaimsSpawns(validateWorkerClaims(parsed));
    }
    catch {
        // Continue to error
    }
    throw new Error("Failed to parse worker claims from output");
}
/**
 * Strip markdown code fences from text.
 *
 * Handles ```json ... ``` and ``` ... ``` blocks.
 *
 * @param text - Text that may contain code fences
 * @returns Inner content with fences removed
 */
export function stripCodeFences(text) {
    text = text.trim();
    if (!text.startsWith("```")) {
        return text;
    }
    const lines = text.split("\n");
    if (lines.length < 3) {
        return text;
    }
    // Remove opening fence line (may include "json" language tag)
    lines.shift();
    // Remove closing fence line if present
    const lastLine = lines[lines.length - 1];
    if (lastLine && lastLine.trim().startsWith("```")) {
        lines.pop();
    }
    return lines.join("\n").trim();
}
/**
 * Extract the last JSON object from text by walking backward from the last `}`.
 *
 * Finds the matching `{` by tracking brace depth.
 *
 * @param text - Text that may contain a trailing JSON block
 * @returns Extracted JSON string
 * @throws Error if no JSON block can be found
 */
export function extractJSONBlock(text) {
    const trimmed = text.trim();
    const lastBrace = trimmed.lastIndexOf("}");
    if (lastBrace === -1) {
        throw new Error("No JSON block found: no closing brace");
    }
    let depth = 0;
    let startIdx = -1;
    for (let i = lastBrace; i >= 0; i--) {
        const ch = trimmed[i];
        if (ch === "}") {
            depth++;
        }
        else if (ch === "{") {
            depth--;
            if (depth === 0) {
                startIdx = i;
                break;
            }
        }
    }
    if (startIdx === -1) {
        throw new Error("No JSON block found: no matching opening brace");
    }
    return trimmed.slice(startIdx, lastBrace + 1);
}
/**
 * Validate that an unknown value is a valid WorkerClaims object.
 *
 * Checks required fields and normalizes optional arrays.
 *
 * @param claims - Unknown value to validate
 * @returns Validated WorkerClaims
 * @throws Error if required fields are missing or types are wrong
 */
export function validateWorkerClaims(claims) {
    if (typeof claims !== "object" || claims === null) {
        throw new Error("Worker claims must be an object");
    }
    const obj = claims;
    // Required field: status
    if (typeof obj.status !== "string" || obj.status.trim() === "") {
        throw new Error("Worker claims missing required field: status");
    }
    const result = {
        status: obj.status,
    };
    if (typeof obj.ant_name === "string")
        result.ant_name = obj.ant_name;
    if (typeof obj.caste === "string")
        result.caste = obj.caste;
    if (typeof obj.task_id === "string")
        result.task_id = obj.task_id;
    if (typeof obj.summary === "string")
        result.summary = obj.summary;
    if (typeof obj.tool_count === "number")
        result.tool_count = obj.tool_count;
    if (Array.isArray(obj.files_created)) {
        result.files_created = obj.files_created.filter((v) => typeof v === "string");
    }
    if (Array.isArray(obj.files_modified)) {
        result.files_modified = obj.files_modified.filter((v) => typeof v === "string");
    }
    if (Array.isArray(obj.tests_written)) {
        result.tests_written = obj.tests_written.filter((v) => typeof v === "string");
    }
    if (Array.isArray(obj.blockers)) {
        result.blockers = obj.blockers.filter((v) => typeof v === "string");
    }
    if (Array.isArray(obj.spawns)) {
        // Cap spawn claims at 50 entries to prevent DoS (T-138-02).
        const rawSpawns = obj.spawns.slice(0, 50);
        const validSpawns = [];
        for (let i = 0; i < rawSpawns.length; i++) {
            const entry = rawSpawns[i];
            if (typeof entry === "string") {
                // Backward compatibility: accept string spawns
                validSpawns.push(entry);
            }
            else if (typeof entry === "object" && entry !== null) {
                const obj2 = entry;
                if (typeof obj2.caste === "string" && typeof obj2.task === "string") {
                    const claim = { caste: obj2.caste, task: obj2.task };
                    if (typeof obj2.reason === "string")
                        claim.reason = obj2.reason;
                    validSpawns.push(claim);
                }
                else {
                    process.stderr.write(`Warning: invalid spawn claim at index ${i}, skipping\n`);
                }
            }
            else {
                process.stderr.write(`Warning: invalid spawn claim at index ${i}, skipping\n`);
            }
        }
        if (validSpawns.length > 0) {
            result.spawns = validSpawns;
        }
        if (obj.spawns.length > 50) {
            process.stderr.write(`Warning: spawn claims truncated from ${obj.spawns.length} to 50\n`);
        }
    }
    if (typeof obj.handoff === "object" && obj.handoff !== null) {
        result.handoff = obj.handoff;
    }
    return result;
}
/**
 * Normalize spawns on a WorkerClaims object to SpawnClaim[].
 *
 * If the claims have spawns, runs them through normalizeSpawnClaims.
 * Otherwise returns claims unchanged.
 */
function normalizeClaimsSpawns(claims) {
    if (claims.spawns !== undefined && claims.spawns.length > 0) {
        return { ...claims, spawns: normalizeSpawnClaims(claims.spawns) };
    }
    return claims;
}
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
export function normalizeSpawnClaims(spawns) {
    return spawns.map((entry) => {
        if (typeof entry === "string") {
            return {
                caste: "builder",
                task: entry,
                reason: "auto-converted from string spawn",
            };
        }
        return entry;
    });
}
