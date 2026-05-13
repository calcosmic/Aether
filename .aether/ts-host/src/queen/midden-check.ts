/**
 * Midden threshold check for the Queen orchestrator.
 *
 * Calls the Go CLI to review recent midden entries. If the total exceeds
 * a configurable threshold, emits a REDIRECT pheromone to steer the colony
 * away from risky patterns.
 *
 * Satisfies ORC-04 (midden threshold checking).
 */

import { callGoJSON } from "../go-bridge.js";
import type { GoBridgeOptions } from "../go-bridge.js";
import type { MiddenCheckResult } from "./types.js";

// ---------------------------------------------------------------------------
// Midden review
// ---------------------------------------------------------------------------

/**
 * Check whether the midden threshold has been exceeded.
 *
 * Calls `aether midden-review` via the Go CLI to get recent failure counts.
 * If total > threshold, emits a REDIRECT pheromone via `aether pheromone-write`.
 *
 * @param opts - Go bridge options (binary path and cwd)
 * @param threshold - Maximum acceptable midden entries before steering (default: 3)
 * @returns Midden check result with exceeded flag and category breakdown
 */
export function checkMiddenThreshold(
  opts: GoBridgeOptions,
  threshold = 3
): MiddenCheckResult {
  let total = 0;
  const categories: Record<string, number> = {};

  try {
    const reviewResult = callGoJSON<{
      total?: number;
      categories?: Record<string, number>;
      entries?: unknown[];
    }>(opts, ["midden-review"]);

    total = reviewResult.total ?? 0;
    if (reviewResult.categories) {
      Object.assign(categories, reviewResult.categories);
    } else if (reviewResult.entries) {
      // Fallback: count entries by category if categories not provided
      for (const entry of reviewResult.entries) {
        const cat =
          typeof entry === "object" &&
          entry !== null &&
          "category" in entry &&
          typeof (entry as Record<string, unknown>).category === "string"
            ? (entry as Record<string, unknown>).category
            : "unknown";
        categories[cat as string] = (categories[cat as string] ?? 0) + 1;
      }
      total = reviewResult.entries.length;
    }
  } catch {
    // If midden-review fails (e.g., command not found), assume clean state
    total = 0;
  }

  const exceeded = total > threshold;

  if (exceeded) {
    try {
      callGoJSON<Record<string, unknown>>(opts, [
        "pheromone-write",
        "--type",
        "REDIRECT",
        "--content",
        `Midden threshold exceeded (${total} > ${threshold}). Colony steering toward caution.`,
        "--strength",
        "80",
      ]);
    } catch {
      // Pheromone emission failure is non-blocking
    }
  }

  return { exceeded, total, categories, threshold };
}

// ---------------------------------------------------------------------------
// Summary formatting
// ---------------------------------------------------------------------------

/**
 * Format a midden check result as a human-readable summary.
 *
 * @param result - Midden check result
 * @returns Formatted summary string
 */
export function formatMiddenSummary(result: MiddenCheckResult): string {
  const status = result.exceeded ? "THRESHOLD BREACHED" : "within limits";
  const lines: string[] = [
    `Midden: ${result.total} entries (${status}, threshold=${result.threshold})`,
  ];

  const cats = Object.entries(result.categories);
  if (cats.length > 0) {
    lines.push("Categories:");
    for (const [cat, count] of cats) {
      lines.push(`  - ${cat}: ${count}`);
    }
  }

  return lines.join("\n");
}
