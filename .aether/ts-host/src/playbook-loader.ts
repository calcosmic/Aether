/**
 * Playbook loader: reads playbook markdown files from repo-local, hub, and
 * absolute paths for injection into worker briefs.
 *
 * Resolution order mirrors Go's `buildPlaybookCandidates` in
 * `cmd/build_playbook_context.go`:
 *   1. Absolute path (if input is absolute)
 *   2. root + playbook path
 *   3. hub + "system" + trimmed path (stripping ".aether/" prefix)
 *   4. hub + trimmed path
 *   5. bare path (fallback)
 *
 * Budget constants match Go:
 *   buildWorkerBriefPlaybookBudgetChars  = 7000
 *   buildWorkerBriefPlaybookPerFileChars = 2800
 */

import { readFileSync, existsSync } from "node:fs";
import { join, isAbsolute } from "node:path";
import { homedir } from "node:os";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface Playbook {
  /** Playbook file name (e.g., "build-wave.md"). */
  name: string;
  /** Absolute path to the resolved playbook file on disk. */
  path: string;
  /** Full markdown content of the playbook. */
  content: string;
}

// ---------------------------------------------------------------------------
// Constants (matching Go)
// ---------------------------------------------------------------------------

const DEFAULT_BUDGET = 7000;
const DEFAULT_PER_FILE = 2800;

// ---------------------------------------------------------------------------
// Workflow -> playbook file mappings
// ---------------------------------------------------------------------------

const WORKFLOW_PLAYBOOKS: Record<string, string[]> = {
  build: [
    "build-prep.md",
    "build-context.md",
    "build-wave.md",
    "build-verify.md",
    "build-complete.md",
  ],
  plan: ["plan-prep.md", "plan-dispatch.md"],
  continue: [
    "continue-verify.md",
    "continue-gates.md",
    "continue-advance.md",
    "continue-finalize.md",
  ],
};

// ---------------------------------------------------------------------------
// Resolution
// ---------------------------------------------------------------------------

/**
 * Resolve candidate file paths for a playbook name, mirroring Go's
 * `buildPlaybookCandidates`.
 *
 * @param root - Repository root directory.
 * @param playbook - Playbook file name (relative or absolute).
 * @returns Ordered array of candidate paths. Empty for empty/whitespace input.
 */
export function resolvePlaybookCandidates(
  root: string,
  playbook: string
): string[] {
  const trimmed = playbook.trim();
  if (trimmed === "") {
    return [];
  }

  const candidates: string[] = [];

  // 1. Absolute path first (if input is absolute)
  if (isAbsolute(trimmed)) {
    candidates.push(trimmed);
  }

  // 2. root + playbook path (for full relative paths like ".aether/docs/command-playbooks/build-prep.md")
  if (root.trim() !== "") {
    candidates.push(join(root, trimmed));
  }

  // 2b. root + .aether/docs/command-playbooks/ + name (for bare names like "build-prep.md")
  if (root.trim() !== "" && !trimmed.includes("/")) {
    candidates.push(join(root, ".aether", "docs", "command-playbooks", trimmed));
  }

  // 3 & 4. Hub paths
  const hubDir = join(homedir(), ".aether");
  const hubSystem = join(hubDir, "system");
  const stripped = trimmed.replace(/^\.aether\//, "");

  candidates.push(join(hubSystem, "docs", "command-playbooks", stripped));
  candidates.push(join(hubDir, stripped));

  // 5. Bare path (fallback)
  candidates.push(trimmed);

  // Deduplicate while preserving order
  const seen = new Set<string>();
  return candidates.filter((c: string) => {
    if (seen.has(c)) return false;
    seen.add(c);
    return true;
  });
}

// ---------------------------------------------------------------------------
// Loading
// ---------------------------------------------------------------------------

/**
 * Load a single playbook by name, trying candidates in resolution order.
 *
 * @param root - Repository root directory.
 * @param name - Playbook file name.
 * @returns Playbook object if found, null otherwise.
 */
export function loadPlaybook(root: string, name: string): Playbook | null {
  const candidates = resolvePlaybookCandidates(root, name);
  for (const candidate of candidates) {
    if (existsSync(candidate)) {
      const content = readFileSync(candidate, "utf-8");
      return { name, path: candidate, content };
    }
  }
  return null;
}

// ---------------------------------------------------------------------------
// Workflow mapping
// ---------------------------------------------------------------------------

/**
 * Load all playbooks for a given workflow, filtering out ones not found on disk.
 *
 * @param root - Repository root directory.
 * @param workflow - Workflow name ("build", "plan", or "continue").
 * @returns Array of loaded Playbook objects.
 */
export function loadPlaybooksForWorkflow(
  root: string,
  workflow: "build" | "plan" | "continue"
): Playbook[] {
  const names = WORKFLOW_PLAYBOOKS[workflow] ?? [];
  const results: Playbook[] = [];
  for (const name of names) {
    const playbook = loadPlaybook(root, name);
    if (playbook !== null) {
      results.push(playbook);
    }
  }
  return results;
}

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

/**
 * Render playbook content as a document section for injection into worker briefs.
 *
 * Mirrors Go's `renderBuildPlaybookContext`: builds a header, iterates playbooks,
 * truncates each to fit within budget, and adds "[playbook truncated]" markers
 * when content exceeds limits.
 *
 * @param playbooks - Array of Playbook objects to render.
 * @param maxBudget - Maximum total character budget. Defaults to 7000.
 * @param maxPerFile - Maximum characters per playbook file. Defaults to 2800.
 * @returns Rendered markdown string, or empty string if no playbooks provided.
 */
export function renderPlaybookContext(
  playbooks: Playbook[],
  maxBudget: number = DEFAULT_BUDGET,
  maxPerFile: number = DEFAULT_PER_FILE
): string {
  if (playbooks.length === 0) {
    return "";
  }

  const header = "## Relevant Playbooks\n\n";
  let remaining = maxBudget - header.length;
  if (remaining <= 0) {
    return header.trim();
  }

  const parts: string[] = [header];

  for (const playbook of playbooks) {
    if (remaining <= 0) {
      break;
    }

    const fileHeader = `### ${playbook.name}\n\n`;
    const fileBudget = Math.min(maxPerFile, remaining - fileHeader.length);

    if (fileBudget <= 0) {
      // Not enough room for even a file header
      break;
    }

    let snippet = playbook.content;
    const truncateMarker = "\n\n[playbook truncated]";

    if (snippet.length > fileBudget) {
      const available = fileBudget - truncateMarker.length;
      if (available > 0) {
        snippet = snippet.slice(0, available) + truncateMarker;
      } else {
        // Not enough room for content + marker, skip
        break;
      }
    }

    const entry = fileHeader + snippet + "\n\n";
    parts.push(entry);
    remaining -= entry.length;
  }

  return parts.join("").trim();
}
