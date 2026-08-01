/**
 * Shared preflight timing configuration for the TypeScript host.
 *
 * This module is production code: it holds the AETHER_PREFLIGHT_TIMEOUT
 * resolver both the retired legacy launcher (platform-dispatcher.ts, which
 * re-exports it for compatibility) and the production Go-adapter wrapper
 * (worker-dispatch.ts) size their budgets from. It exists as its own module
 * so production sources never import the deprecated launcher.
 */
/**
 * Default preflight probe budget in milliseconds. Mirrors
 * `hostedPreflightTimeout` in pkg/codex/platform_dispatch.go so both hosts
 * agree on the same fallback when AETHER_PREFLIGHT_TIMEOUT is unset or
 * unparseable. Pinned by TestHostsAgreeOnPreflightDefaultBudget in
 * cmd/preflight_docs_test.go.
 */
export declare const PREFLIGHT_DEFAULT_TIMEOUT_MS = 45000;
/**
 * Resolve the preflight probe timeout from AETHER_PREFLIGHT_TIMEOUT.
 *
 * Accepts the Go duration subset the knob is actually used with: a positive
 * decimal followed by `ms`, `s`, or `m` (e.g. "90s", "1500ms", "2m"). Any
 * other value — empty, unparseable, zero, negative, or a bare number with no
 * unit — falls back to PREFLIGHT_DEFAULT_TIMEOUT_MS so a broken env var never
 * bricks dispatch.
 */
export declare function resolvePreflightTimeoutMs(): number;
