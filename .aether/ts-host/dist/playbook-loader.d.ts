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
export interface Playbook {
    /** Playbook file name (e.g., "build-wave.md"). */
    name: string;
    /** Absolute path to the resolved playbook file on disk. */
    path: string;
    /** Full markdown content of the playbook. */
    content: string;
}
/**
 * Resolve candidate file paths for a playbook name, mirroring Go's
 * `buildPlaybookCandidates`.
 *
 * @param root - Repository root directory.
 * @param playbook - Playbook file name (relative or absolute).
 * @returns Ordered array of candidate paths. Empty for empty/whitespace input.
 */
export declare function resolvePlaybookCandidates(root: string, playbook: string): string[];
/**
 * Load a single playbook by name, trying candidates in resolution order.
 *
 * @param root - Repository root directory.
 * @param name - Playbook file name.
 * @returns Playbook object if found, null otherwise.
 */
export declare function loadPlaybook(root: string, name: string): Playbook | null;
/**
 * Load all playbooks for a given workflow, filtering out ones not found on disk.
 *
 * @param root - Repository root directory.
 * @param workflow - Workflow name ("build", "plan", or "continue").
 * @returns Array of loaded Playbook objects.
 */
export declare function loadPlaybooksForWorkflow(root: string, workflow: "build" | "plan" | "continue"): Playbook[];
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
export declare function renderPlaybookContext(playbooks: Playbook[], maxBudget?: number, maxPerFile?: number): string;
