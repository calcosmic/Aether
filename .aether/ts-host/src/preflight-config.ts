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
export const PREFLIGHT_DEFAULT_TIMEOUT_MS = 45_000;

/**
 * Parse a Go-style duration string into milliseconds.
 *
 * Accepts one or more `<decimal><unit>` tokens with units `ms`, `s`, `m`, or
 * `h`, summed exactly like Go's time.ParseDuration does for compound values
 * (e.g. "90s", "1500ms", "1m30s", "1.5h"). Returns undefined for anything
 * else — empty strings, bare numbers with no unit, and the exotic Go units
 * (`ns`, `us`, `µs`) this knob has no meaningful use for.
 */
export function parseGoDurationMs(raw: string): number | undefined {
  if (!/^(\d+(?:\.\d+)?(ms|s|m|h))+$/.test(raw)) return undefined;

  let totalMs = 0;
  for (const match of raw.matchAll(/(\d+(?:\.\d+)?)(ms|s|m|h)/g)) {
    const value = Number.parseFloat(match[1]!);
    if (!Number.isFinite(value)) return undefined;
    const unit = match[2];
    const multiplier =
      unit === "ms" ? 1 : unit === "s" ? 1000 : unit === "m" ? 60_000 : 3_600_000;
    totalMs += value * multiplier;
  }
  return totalMs;
}

// One warning per distinct bad value per process: the resolver is called on
// every budget computation, and repeating the same line would drown the
// dispatch output it is trying to protect.
const warnedTimeoutValues = new Set<string>();

/** Test-only: reset the once-per-value warning dedupe. */
export function __resetPreflightTimeoutWarnings(): void {
  warnedTimeoutValues.clear();
}

/**
 * Resolve the preflight probe timeout from AETHER_PREFLIGHT_TIMEOUT.
 *
 * Accepts Go duration syntax with `ms`, `s`, `m`, and `h` units, including
 * compound values ("90s", "1500ms", "1m30s", "1.5h") — the same forms the Go
 * runtime accepts via time.ParseDuration, so one knob means one value on
 * both hosts (WR-01). Any other value — empty, unparseable, zero, negative,
 * or a bare number with no unit — falls back to PREFLIGHT_DEFAULT_TIMEOUT_MS
 * so a broken env var never bricks dispatch, and prints one loud stderr
 * warning so the divergence is never silent.
 */
export function resolvePreflightTimeoutMs(): number {
  const raw = process.env["AETHER_PREFLIGHT_TIMEOUT"]?.trim();
  if (!raw) return PREFLIGHT_DEFAULT_TIMEOUT_MS;

  const parsedMs = parseGoDurationMs(raw);
  if (parsedMs !== undefined && parsedMs > 0) return parsedMs;

  if (!warnedTimeoutValues.has(raw)) {
    warnedTimeoutValues.add(raw);
    process.stderr.write(
      `Warning: AETHER_PREFLIGHT_TIMEOUT="${raw}" is not a positive duration (expected e.g. "90s", "1500ms", "1m30s", "1.5h") — falling back to ${PREFLIGHT_DEFAULT_TIMEOUT_MS}ms\n`
    );
  }
  return PREFLIGHT_DEFAULT_TIMEOUT_MS;
}
