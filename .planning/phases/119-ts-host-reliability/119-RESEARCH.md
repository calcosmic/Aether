# Phase 119: TS Host Reliability - Research

**Gathered:** 2026-05-14
**Status:** Ready for planning

## Known Issues from Codex Analysis

### 1. TypeScript Typecheck Failures
- `npm --prefix .aether/ts-host run typecheck` fails
- Root cause: TS test mocks violate strict optional typing (`exactOptionalPropertyTypes`)
- Files likely affected: test files using `__setCreateQueenOrchestrator` and other mock injection patterns

### 2. Full TS Test Suite Hangs
- `npm test` hangs in `event-bridge.test.ts`
- Root cause: event bridge stops the child process but does not await full subprocess/readline cleanup
- Location: `.aether/ts-host/src/event-bridge.ts:171`
- Impact: prevents CI from running full suite

### 3. Fixed Temp Completion File Paths
- Completion files written to `/tmp/aether-lifecycle/plan-completion.json`
- Location: `.aether/ts-host/src/go-bridge.ts:177`
- Risk: race conditions and cross-test contamination when multiple test runs overlap

### 4. Event Bridge Teardown
- `stopEventBridge` may not properly await `readline` interface close and subprocess exit
- Need to verify: `controller.subprocess.kill()` vs graceful shutdown with `readline.close()`

## Existing Test Infrastructure

- 18 test files in `.aether/ts-host/test/`
- `node:test` + `node:assert/strict` pattern
- `setupTestColony()` helper for temp directories
- `process.stdout.write` monkey-patching for output capture

## Phase Boundary

This phase is **fix-only** — no new features. Focus on making existing tests reliable and typecheck clean.
