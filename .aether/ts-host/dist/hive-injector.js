/**
 * Hive wisdom injection module for the TypeScript orchestration host.
 *
 * Calls Go's `hive-read` and `registry-list` commands to fetch cross-colony
 * wisdom, formats entries for worker prompts, and enforces token budget and
 * relevance scoring.
 *
 * Hive wisdom is injected after the skill section. The current prompt order
 * (skills before pheromones) is preserved; hive is added after skills.
 */
import { resolve } from "node:path";
import { callGoJSON } from "./go-bridge.js";
// ---------------------------------------------------------------------------
// Domain tag resolution
// ---------------------------------------------------------------------------
/**
 * Resolve domain tags for the current repository from the colony registry.
 *
 * Calls `aether registry-list` and matches the colony by repo path.
 * Best-effort: returns empty array on any failure.
 */
export async function resolveDomainTags(opts) {
    try {
        const result = callGoJSON(opts, ["registry-list"]);
        const cwdResolved = resolve(opts.cwd);
        for (const colony of result.colonies) {
            if (resolve(colony.repo_path) === cwdResolved) {
                return colony.domains;
            }
        }
        return [];
    }
    catch {
        return [];
    }
}
// ---------------------------------------------------------------------------
// Hive wisdom reading
// ---------------------------------------------------------------------------
/**
 * Read hive wisdom from the Go runtime and format it for worker prompts.
 *
 * Calls `aether hive-read` without `--domain` (fetch all entries), applies
 * relevance scoring and token budget, and returns a formatted markdown section.
 *
 * Graceful degradation: on any error returns empty string and logs a warning.
 */
export async function readHiveWisdom(opts) {
    let result;
    try {
        result = callGoJSON(opts, [
            "hive-read",
            "--for-worker",
            "--min-confidence",
            String(opts.minConfidence),
        ]);
    }
    catch (err) {
        const msg = err instanceof Error ? err.message : String(err);
        process.stderr.write(`Warning: hive-read failed: ${msg}\n`);
        return "";
    }
    if (!result.entries || result.entries.length === 0) {
        return "";
    }
    // Relevance scoring: exact domain match = full confidence; partial = * 0.5
    const domainSet = new Set(opts.domains.map((d) => d.toLowerCase()));
    const scored = result.entries.map((entry) => {
        const isExact = domainSet.has(entry.domain.toLowerCase());
        const effectiveConfidence = isExact ? entry.confidence : entry.confidence * 0.5;
        return { entry, effectiveConfidence };
    });
    // Sort by discounted confidence descending, then take top N
    scored.sort((a, b) => b.effectiveConfidence - a.effectiveConfidence);
    const top = scored.slice(0, opts.maxEntries).map((s) => s.entry);
    let section = formatHiveWisdomSection(top);
    // Enforce budget: drop oldest entries until it fits
    while (section.length > opts.budgetChars && top.length > 1) {
        // Remove oldest by created_at
        top.sort((a, b) => +new Date(a.created_at) - +new Date(b.created_at));
        top.shift();
        section = formatHiveWisdomSection(top);
    }
    // If even a single entry exceeds budget, truncate its text
    if (section.length > opts.budgetChars && top.length === 1) {
        const entry = top[0];
        const header = "## HIVE WISDOM (Cross-Colony Patterns)\n\n";
        const prefix = `(${entry.domain}, ${entry.confidence.toFixed(2)}) `;
        const maxTextLen = opts.budgetChars - header.length - prefix.length;
        if (maxTextLen > 3) {
            const truncatedText = entry.text.slice(0, maxTextLen - 3) + "...";
            section = header + prefix + truncatedText;
        }
        else {
            // Budget is too small even for header + prefix; return empty
            section = "";
        }
    }
    return section.length <= opts.budgetChars ? section : "";
}
// ---------------------------------------------------------------------------
// Formatting
// ---------------------------------------------------------------------------
/**
 * Format wisdom entries as a markdown section.
 *
 * Returns empty string if entries is empty.
 * Header: `## HIVE WISDOM (Cross-Colony Patterns)\n\n`
 * Each line: `({domain}, {confidence.toFixed(2)}) {text}`
 */
export function formatHiveWisdomSection(entries) {
    if (entries.length === 0) {
        return "";
    }
    const lines = ["## HIVE WISDOM (Cross-Colony Patterns)"];
    for (const entry of entries) {
        lines.push(`(${entry.domain}, ${entry.confidence.toFixed(2)}) ${entry.text}`);
    }
    return lines.join("\n\n");
}
