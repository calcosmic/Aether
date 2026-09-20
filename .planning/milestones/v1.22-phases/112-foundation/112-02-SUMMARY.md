---
phase: 112
plan: 02
status: complete
completed: "2026-05-13"
---

# Plan 112-02 Summary: Event Bridge, Caste Config, and Boundary Enforcement

## What Was Built

Built the runtime core that downstream phases (113-118) depend on: the event bridge consuming Go ceremony events, the shared YAML caste config loader, and hardened boundary enforcement.

### Changes

1. **Event bridge** (`.aether/ts-host/src/event-bridge.ts`)
   - `startEventBridge(opts)` replays historical events via `aether event-bus-replay`, then spawns `aether event-bus-subscribe --stream` for live NDJSON consumption
   - Uses `readline` to parse NDJSON lines from subprocess stdout
   - Deduplicates replay-to-stream handoff via a `Set` of seen event IDs with LRU eviction at 10,000 entries
   - Runtime boundary guard rejects any write-mode attempt on `.aether/data/` paths
   - Exports `startEventBridge`, `stopEventBridge`, `EventBridgeOptions`, `EventBridgeController`

2. **Caste config loader** (`.aether/ts-host/src/caste-config.ts`)
   - `loadCeremonyConfig(cwd)` reads `.aether/config/ceremony.yaml` with `js-yaml`
   - Validates minimum required keys (`castes`, `stage_separator`)
   - Falls back to inline `DEFAULT_CEREMONY_CONFIG` (mirroring Go hardcoded defaults) if YAML is missing
   - Typed accessors: `getCasteConfig`, `getCasteEmoji`, `getCasteColor`, `getCasteLabel`

3. **Extended boundary reference** (`.aether/ts-host/src/boundary-reference.ts`)
   - Added `ALLOWED_READ_PATHS` with `.aether/data/event-bus.jsonl` as explicit read-only exception
   - Added `isReadOnlyAllowed(path)` predicate
   - Added `assertNoWriteToData(path, mode?)` that throws `BoundaryViolationError` for any write-mode attempt under `.aether/data/`
   - Exported `BoundaryViolationError` class

4. **Tests**
   - `boundary-contract.test.ts` — 8 tests covering write rejection, read allowance, GO_OWNED_PATHS coverage
   - `event-bridge.test.ts` — 4 tests covering replay + stream, deduplication, stop, and error shape
   - `caste-config.test.ts` — 9 tests covering load, fallback, accessors, and unknown-caste fallbacks

## Self-Check

- [x] `npm test` passes with zero failures (26 tests across 5 suites)
- [x] `npm run typecheck` passes
- [x] `npm run build` produces dist/ with no errors
- [x] Go `pkg/events` tests pass
- [x] All acceptance criteria met

## Key Files Created/Modified

- `.aether/ts-host/src/event-bridge.ts`
- `.aether/ts-host/src/caste-config.ts`
- `.aether/ts-host/src/boundary-reference.ts`
- `.aether/ts-host/test/boundary-contract.test.ts`
- `.aether/ts-host/test/event-bridge.test.ts`
- `.aether/ts-host/test/caste-config.test.ts`
- `.aether/ts-host/package.json` (added @types/js-yaml)
