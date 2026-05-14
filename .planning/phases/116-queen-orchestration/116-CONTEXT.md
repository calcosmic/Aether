# Phase 116: Queen Orchestration - Context

**Gathered:** 2026-05-13
**Status:** Ready for planning

## Phase Boundary

Port Queen intelligence into the TS host: workflow pattern display, Builder-Probe Lock enforcement, tiered escalation delegation to Go, and midden threshold checks. The TS host delegates heavy logic to Go CLI commands rather than reimplementing 474 lines of recovery orchestration.

## Implementation Decisions

### D-01: Hybrid Delegation Model
- **Decision:** TS host calls Go `--plan-only` for Queen recommendations, Go CLI for recovery/midden decisions, but enforces Builder-Probe Lock in TS before writing completion files.
- **Why:** Go owns the complex logic (recovery, midden, caste scoring). TS host owns the dispatch boundary where locks are enforced.

### D-02: Builder-Probe Lock in TS Host
- **Decision:** The TS host downgrades builder `completed` to `code_written` in the completion file unless a probe worker has also completed. Go finalizer normalizes `code_written` -> `completed`.
- **Why:** The finalizer already normalizes; preserving `code_written` in the completion file ensures the lock is respected.

### D-03: TS Retry Limit = 1 (Delegate to Go)
- **Decision:** Wave orchestrator retry limit defaults to 1. Deeper recovery (peer reassignment, fixer dispatch) is delegated to Go's recovery orchestrator.
- **Why:** Prevents recovery budget desync between TS and Go.

### D-04: Workflow Patterns are Display-Only
- **Decision:** Pattern names (SPBV, Investigate-Fix, etc.) are derived from manifest dispatches for display, not used to modify dispatch plans.
- **Why:** Go's `queenOrchestrate` already selects castes. Pattern names are ceremony sugar.

## Canonical References

- `.planning/phases/116-queen-orchestration/116-RESEARCH.md` — Full research with Go code references
- `.planning/phases/114-real-worker-dispatch/114-02-SUMMARY.md` — Wave orchestrator and lifecycle
- `.aether/ts-host/src/lifecycle.ts` — Lifecycle orchestrator
- `.aether/ts-host/src/wave-orchestrator.ts` — Wave dispatch with retry
- `.aether/ts-host/src/worker-dispatch.ts` — Single worker dispatch
- `.aether/ts-host/src/go-bridge.ts` — Go CLI invocation
- `cmd/codex_build.go` — Manifest generation with queen recommendations
- `cmd/codex_build_finalize.go` — Status normalization
- `cmd/recovery_orchestrator.go` — Recovery logic
- `cmd/midden_cmds.go` — Midden CLI

## Existing Code Insights

### Reusable Assets
- **lifecycle.ts** — already orchestrates plan/build/continue. Queen orchestrator wraps the build step.
- **wave-orchestrator.ts** — retry logic exists; just need to change default retryLimit from 2 to 1.
- **go-bridge.ts** — `callGoJSON` already used for spawn-log/complete/finalize.
- **worker-dispatch.ts** — dispatchSingleWorker returns results that the lock inspects.

### Integration Points
- **Build manifest** — already contains `queen_recommendation` and `queen_execution_policy` fields.
- **Completion file** — written by `writeCompletionFile` in tmpdir, passed to `build-finalize`.
- **Midden CLI** — `aether midden-review` returns grouped unacknowledged entries.
- **Failure classify** — `aether failure-classify` classifies worker failures.
