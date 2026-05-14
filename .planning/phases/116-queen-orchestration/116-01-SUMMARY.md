---
phase: "116"
plan: "01"
subsystem: "TypeScript Orchestration Host"
tags: ["queen", "orchestrator", "workflow-patterns", "builder-probe-lock", "midden", "escalation"]
dependency_graph:
  requires: ["109-03"]
  provides: ["116-02"]
  affects: [".aether/ts-host/src/queen/*", ".aether/ts-host/test/queen.test.ts"]
tech_stack:
  added: []
  patterns: ["ESM NodeNext", "node:test + node:assert/strict", "Go CLI delegation via callGoJSON"]
key_files:
  created:
    - ".aether/ts-host/src/queen/types.ts"
    - ".aether/ts-host/src/queen/workflow-patterns.ts"
    - ".aether/ts-host/src/queen/builder-probe-lock.ts"
    - ".aether/ts-host/src/queen/midden-check.ts"
    - ".aether/ts-host/src/queen/escalation.ts"
    - ".aether/ts-host/src/queen/orchestrator.ts"
    - ".aether/ts-host/src/queen/index.ts"
    - ".aether/ts-host/test/queen.test.ts"
  modified: []
decisions:
  - "Used 'code_written' as a TerminalWorkerStatus downgrade value for Builder-Probe Lock, even though it is not in the canonical TerminalWorkerStatus union. This is intentional: the Go finalizer may need to accept it, or the type may need expansion in a future plan."
  - "Mocked Go CLI calls in tests by using a no-op binary path (/usr/bin/true) and letting heuristics fall back, rather than injecting callGoJSON. This keeps tests fast and free of subprocess mocking."
  - "Kept QueenOrchestratorOptions as a flat extension of GoBridgeOptions (no nesting) to match existing host option patterns."
metrics:
  duration: "~10 minutes"
  completed_date: "2026-05-13"
---

# Phase 116 Plan 01: Queen Orchestration — Wave 1 Summary

**One-liner:** Built the Queen orchestrator module that reads manifest recommendations, derives workflow patterns, enforces the Builder-Probe Lock, checks midden thresholds, and maps failures to recovery actions.

## What Was Built

The Queen orchestrator is the brain of the TypeScript host's build phase. It decides what kind of work the colony is doing, ensures builders don't claim completion without verification, checks if the colony's failure log is too full, and figures out what to do when workers fail.

### Files Created

| File | Purpose |
|------|---------|
| `src/queen/types.ts` | Type definitions: recommendations, workflow patterns, lock results, midden results, recovery actions, orchestrator options/results |
| `src/queen/workflow-patterns.ts` | Derives workflow pattern from dispatch castes; maps verification depths; formats recommendations |
| `src/queen/builder-probe-lock.ts` | Enforces the Builder-Probe Lock: downgrades builder status to `code_written` if no probe verified |
| `src/queen/midden-check.ts` | Calls `aether midden-review` via Go CLI; emits REDIRECT pheromone if threshold exceeded |
| `src/queen/escalation.ts` | Classifies failures (via Go CLI or heuristic fallback); maps them to retry/escalate/fixer/peer-reassign actions |
| `src/queen/orchestrator.ts` | `createQueenOrchestrator` factory and `runBuild` pipeline: midden check -> pattern -> dispatch -> lock -> failure handling |
| `src/queen/index.ts` | Barrel exports for the entire queen module |
| `test/queen.test.ts` | 21 tests covering pattern derivation, lock behavior, midden formatting, and escalation mapping |

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

| File | Line | Reason |
|------|------|--------|
| `src/queen/builder-probe-lock.ts` | `status: "code_written" as TerminalWorkerStatus` | `code_written` is not in the current `TerminalWorkerStatus` union (`completed`, `failed`, `blocked`, `timeout`, `manually-reconciled`). This is a deliberate extension for the lock mechanism. The Go finalizer may need to accept it, or the union may need expansion. |

## Threat Flags

No new security-relevant surface introduced. All Go CLI calls use existing `callGoJSON` with `execFileSync` (no shell interpolation). Pheromone emission in `midden-check.ts` uses hardcoded `--type REDIRECT` and sanitized numeric strength.

## Self-Check: PASSED

- [x] All 8 planned source files created
- [x] Test file created with 21 tests, all passing
- [x] Commit `cdbd18b2` exists and contains only intended files
- [x] No unexpected file deletions
