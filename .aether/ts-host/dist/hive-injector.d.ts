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
import { type GoBridgeOptions } from "./go-bridge.js";
import type { HiveWisdomEntry } from "./types.js";
export interface HiveInjectorOptions extends GoBridgeOptions {
    /** Domain tags for the current repo (e.g., ["go", "typescript", "cli"]). */
    domains: string[];
    /** Minimum confidence threshold (inclusive). */
    minConfidence: number;
    /** Maximum number of entries to include. */
    maxEntries: number;
    /** Character budget for the formatted section (including header). */
    budgetChars: number;
}
/**
 * Resolve domain tags for the current repository from the colony registry.
 *
 * Calls `aether registry-list` and matches the colony by repo path.
 * Best-effort: returns empty array on any failure.
 */
export declare function resolveDomainTags(opts: GoBridgeOptions): Promise<string[]>;
/**
 * Read hive wisdom from the Go runtime and format it for worker prompts.
 *
 * Calls `aether hive-read` without `--domain` (fetch all entries), applies
 * relevance scoring and token budget, and returns a formatted markdown section.
 *
 * Graceful degradation: on any error returns empty string and logs a warning.
 */
export declare function readHiveWisdom(opts: HiveInjectorOptions): Promise<string>;
/**
 * Format wisdom entries as a markdown section.
 *
 * Returns empty string if entries is empty.
 * Header: `## HIVE WISDOM (Cross-Colony Patterns)\n\n`
 * Each line: `({domain}, {confidence.toFixed(2)}) {text}`
 */
export declare function formatHiveWisdomSection(entries: HiveWisdomEntry[]): string;
