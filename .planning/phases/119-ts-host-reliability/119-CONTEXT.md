# Phase 119: TS Host Reliability - Context

**Gathered:** 2026-05-14
**Status:** Ready for planning

## Phase Boundary

Fix TypeScript typecheck failures, test suite hangs, and temp file races in the TS host. No new features — only stabilization of existing v1.17 code.

## Implementation Decisions

- `exactOptionalPropertyTypes: true` in tsconfig means mock objects must match exact types; `undefined` is not assignable to optional properties
- Event bridge teardown must await both subprocess exit AND readline interface close
- Completion files should use unique temp directories per `runLifecycle` call (e.g., `mkdtempSync`)
- Test suite hangs suggest async cleanup not awaited — may need test timeout or explicit `afterEach` cleanup

## Known Issues

| Issue | Location | Severity |
|-------|----------|----------|
| Typecheck: `caste: string \| undefined` not assignable to `string` | `test/lifecycle.test.ts:445,497,547` | Blocking CI |
| Test hang: event bridge subprocess not fully cleaned up | `src/event-bridge.ts:171-178` | Blocking full suite |
| Race condition: fixed temp path `/tmp/aether-lifecycle/` | `src/lifecycle.ts:237,378,437` | Cross-test contamination |

## Files to Modify

- `test/lifecycle.test.ts` — Fix mock types for `exactOptionalPropertyTypes`
- `src/event-bridge.ts` — Make `stop()` async, await subprocess exit + readline close
- `src/lifecycle.ts` — Pass unique temp dir to `writeCompletionFile` per run
- `src/go-bridge.ts` — May need `writeCompletionFile` to accept or generate unique dirs
