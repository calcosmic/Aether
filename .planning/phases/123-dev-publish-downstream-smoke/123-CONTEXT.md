# Phase 123: Dev Publish + Downstream Smoke - Context

**Gathered:** 2026-05-14
**Status:** Ready for planning

## Phase Boundary

Publish current v1.0.38 to the dev channel and verify the full colony lifecycle in a clean downstream repo. This is the final gate before v1.18 milestone completion.

## Implementation Decisions

- Use `aether publish --channel dev` for isolated testing
- Target `~/repos/Formica` as the downstream smoke test repo
- Smoke test covers: update, init, plan, build, continue, oracle
- Record any blockers found during smoke test

## Threat Model

| Threat | Mitigation |
|--------|-----------|
| Dev publish overwrites stable runtime | Dev channel uses separate `~/.aether-dev/` and `aether-dev` binary |
| Downstream repo has uncommitted changes | Check git status first; abort if dirty |
| Smoke test leaves colony state behind | Clean up after test or use temp dir |
| Missing `aether-dev` binary not on PATH | Verify binary exists and is executable before downstream test |

## Test Strategy
- Publish succeeds with version verification
- Downstream update pulls latest dev channel files
- Full lifecycle commands execute without errors
- Any failures are recorded as blockers
