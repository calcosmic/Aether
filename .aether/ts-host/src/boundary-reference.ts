/**
 * Runtime boundary contract reference for the TypeScript orchestration host.
 *
 * This file imports the contract path constant and defines the Go-owned paths
 * that the TS host MUST NOT write to. All state mutation goes through Go
 * finalizers (build-finalize, plan-finalize, continue-finalize).
 *
 * Contract document: .aether/references/contracts/runtime-boundary-contract.md
 * Enforcement test: cmd/boundary_contract_test.go
 */

/**
 * Path to the runtime boundary contract document.
 * The contract defines ownership between Go, TypeScript, editable assets, and Bash.
 */
export const RUNTIME_BOUNDARY_CONTRACT_PATH =
  ".aether/references/contracts/runtime-boundary-contract.md" as const;

/**
 * Paths owned exclusively by the Go runtime.
 * The TS host MUST NOT write to these paths directly.
 * All mutations to these files must go through Go finalizer commands.
 *
 * Enforced by: TestBoundaryContract_NoStateWritesDuringOrchestration
 * in cmd/boundary_contract_test.go
 */
export const GO_OWNED_PATHS = [
  ".aether/data/COLONY_STATE.json",
  ".aether/data/session.json",
  ".aether/data/pheromones.json",
  ".aether/data/constraints.json",
  ".aether/data/handoffs/",
  ".aether/data/midden/",
] as const;

/**
 * Paths that the TS host is explicitly allowed to read.
 * These are exceptions to the write-only boundary contract.
 */
export const ALLOWED_READ_PATHS = [
  ".aether/data/event-bus.jsonl",
] as const;

/**
 * TS host ownership classification (per Classic v5.4.0 comparison):
 * - Restore in TS: spawn-logger, logger, errors
 * - Keep in Go: state-guard, caste-colors, event-types, file-lock, banner,
 *   colors, init, binary-downloader, update-transaction, version-gate
 * - Obsolete: state-sync, interactive-setup, nestmate-loader
 * - Reject as unsafe: direct state writes, visual parsing, wrapper recovery menus
 */
export const CLASSIFICATION_SCHEMA = {
  restoreInTS: "Behaviors that should live in the TypeScript host",
  keepInGo: "Safety behaviors correctly owned by the Go runtime",
  obsolete: "Behaviors no longer relevant in the hybrid architecture",
  rejectAsUnsafe: "Behaviors that violated safety boundaries",
} as const;

// ---------------------------------------------------------------------------
// Boundary enforcement helpers
// ---------------------------------------------------------------------------

/**
 * Error thrown when the TS host attempts to violate the runtime boundary.
 */
export class BoundaryViolationError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "BoundaryViolationError";
  }
}

/**
 * Check whether a path is in the explicit read-only allowlist.
 */
export function isReadOnlyAllowed(path: string): boolean {
  const normalized = path.replace(/\\/g, "/");
  return ALLOWED_READ_PATHS.some((allowed) => normalized === allowed);
}

/**
 * Assert that a path is not being opened in write mode under `.aether/data/`.
 *
 * - If `mode` is `"read"`, the path is allowed (provided it is in ALLOWED_READ_PATHS).
 * - If `mode` is omitted or anything other than `"read"`, any path under
 *   `.aether/data/` is rejected.
 * - Paths outside `.aether/data/` are always allowed.
 */
export function assertNoWriteToData(
  path: string,
  mode?: string
): void {
  const normalized = path.replace(/\\/g, "/");

  // Not under .aether/data/ — no concern
  if (!normalized.startsWith(".aether/data/")) return;

  // Read mode on allowlisted path — permitted
  if (mode === "read" && isReadOnlyAllowed(normalized)) return;

  // Everything else under .aether/data/ is rejected
  throw new BoundaryViolationError(
    `TS host attempted to write to Go-owned path: ${path}`
  );
}
