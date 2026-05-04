# Phase 96: Auto-Recovery Orchestrator - Context

**Gathered:** 2026-05-03
**Status:** Ready for planning

<domain>
## Phase Boundary

The queen automatically recovers from worker and gate failures: workers are retried within a per-wave budget, tasks are redistributed to same-caste peers, and the Fixer agent is dispatched as a recovery strategy in the retry/reassign/fixer sequence. All actions are logged to the existing recovery-log-{N}.json files.

This phase wires together infrastructure from Phases 93-95 into an active recovery loop. It does NOT add new gates, new failure classifications, or new Fixer capabilities — it orchestrates what already exists.

The phase adds: (1) classification-dependent recovery sequence (transient → retry then peer, systemic → escalate, requires-attempt → one retry), (2) per-wave retry budget tracking in the recovery-log file, (3) Fixer dispatch as a recovery strategy (not just a gate outcome), (4) shared recovery decision logic called from both build and continue.

</domain>

<decisions>
## Implementation Decisions

### Recovery Sequence Order
- **D-01:** Recovery sequence is classification-dependent: transient failures retry first (1 retry), then try peer reassignment. Systemic failures (blocking) skip retry entirely and escalate immediately. Requires-attempt failures get 1 retry then escalate. The existing 3-tier classification from Phase 94 provides enough control — no finer tiers needed.
- **D-02:** For transient failures: retry same worker once. If it fails again, immediately try peer reassignment. If peer also fails, retry peer once. This balances giving transient issues a chance to self-resolve with fast escalation to healthy workers.
- **D-03:** For requires-attempt failures: try once. If it succeeds, continue. If it fails, escalate to user. No peer reassignment — the ambiguity that makes it requires-attempt means the task itself may be problematic, not the worker.
- **D-04:** For blocking failures: immediate escalation. No retry, no reassignment. Per Phase 94 D-04.

### Build vs Continue Split
- **D-05:** Shared recovery decision logic (classify → choose action) is called from separate points. The build dispatch loop calls it for worker failures. The continue finalize flow calls it for gate failures. Both call into the same core decision function but with different failure sources.
- **D-06:** The orchestrator is a function (or small set of functions), not a long-running process. It runs, makes a decision, logs it, and returns. Per COORD-03 (queen is single-invocation).

### RECV-04: Fixer as Recovery Strategy
- **D-07:** Fixer dispatch is integrated into the auto-recovery sequence as a third strategy after retry and peer reassignment. For recoverable failures: retry → peer reassign → Fixer dispatch. For requires-attempt: retry → Fixer dispatch. For blocking: escalate (no Fixer).
- **D-08:** The Fixer is dispatched with context about the recovery sequence so far (original failure, retry attempts, peer reassignment outcome). This gives the Fixer more context than Phase 95's gate-only dispatch.
- **D-09:** Phase 95's auto-resolve for soft_block gates in continue finalize remains unchanged. Phase 96 adds Fixer-as-recovery-strategy as a new trigger path (build-time worker failure → retry → peer → Fixer), distinct from Phase 95's gate-outcome trigger path (continue → soft_block gate failed → auto-resolve failed → Fixer).

### Budget Tracking
- **D-10:** Per-wave retry budget (default 3). Budget resets when a new wave starts, matching the circuit breaker's per-wave reset (D-06 from Phase 94 context). Each wave gets its own retry budget.
- **D-11:** Budget tracking lives in the recovery-log file via a `recovery_budget` object: total_budget, retries_used, reassigns_used, fixer_dispatches_used. Co-located with the actions it governs. Does not bloat COLONY_STATE.json.
- **D-12:** Budget is consumed by recovery actions: each retry, peer reassignment, or Fixer dispatch decrements the remaining budget. When budget is exhausted, the remaining failures escalate to the user.

### Claude's Discretion
- Exact function signatures and Go struct names for the orchestrator
- How the shared recovery decision function is structured (single function with parameters, or small decision tree)
- Whether the orchestrator function is a new file or extends an existing one
- Exact format of the recovery_budget object in the recovery-log file
- How Fixer context is enriched with recovery sequence history
- Escalation message format when all recovery attempts fail
- Test structure and coverage approach
- Whether to add a CLI command for inspecting recovery budget status

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Recovery Data Model (Phase 94 — consumed by this phase)
- `cmd/recovery_classify.go` — FailureClassification, FailureType, classifyWorkerFailure(), FailureRecord, RecoveryLogEntry, RecoveryLogFile, recoveryLogWritePhase(), recoveryLogReadPhase()
- `cmd/recovery_classify_test.go` — Classification test patterns
- `.planning/phases/94-recovery-data-model/94-CONTEXT.md` — Failure classification decisions, retry boundaries, recovery visibility

### Gate Classification (Phase 93 — consumed for gate-aware decisions)
- `cmd/gate.go` — GateClassificationTier, gateClassifications, gateClassify(), isHardBlockGate(), GateCheckResult with QueenAnnotation
- `.planning/phases/93-gate-classification-infrastructure/93-CONTEXT.md` — Gate classification tiers, audit trail patterns

### Smart Gate Pipeline (Phase 95 — existing Fixer dispatch to build on)
- `cmd/codex_continue_finalize.go` — Auto-resolve flow for soft_block gates, existing Fixer dispatch on gate failure
- `.planning/phases/95-smart-gate-pipeline/95-CONTEXT.md` — Auto-resolve decisions, threshold configuration, Fixer dispatch flow

### Circuit Breaker and Peer Redistribution
- `cmd/circuit_breaker.go` — CircuitBreaker struct, findSameCastePeer(), NewCircuitBreaker(), per-wave Reset(), event emission
- `cmd/circuit_breaker_event_test.go` — Event emission test patterns

### Fixer Dispatch
- `cmd/fixer_dispatch.go` — dispatchFixer(), resolveFixedGates(), recordFixerFailure(), gateRetryKey(), readGateResultsPhase(), incrementUnblockAttempts(), checkAttemptCap()
- `cmd/fixer_dispatch_test.go` — Fixer dispatch test patterns
- `cmd/unblock_cmd.go` — /ant-unblock command that triggers Fixer dispatch

### Build and Continue Flows
- `cmd/codex_continue.go` — Continue command, gate check loop
- `cmd/codex_continue_finalize.go` — Finalize phase with gate evaluation and auto-recovery

### Worker Dispatch
- `pkg/codex/` — WorkerDispatch struct, dispatch types used by findSameCastePeer()

### Requirements
- `.planning/REQUIREMENTS.md` — RECV-02 (auto-retry budget), RECV-03 (peer reassignment), RECV-04 (Fixer on gate failure)
- `.planning/ROADMAP.md` Phase 96 — Goal, success criteria, dependencies

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `classifyWorkerFailure()` in recovery_classify.go — Already classifies failures into recoverable/requires-attempt/blocking. The orchestrator calls this first.
- `findSameCastePeer()` in circuit_breaker.go — Already finds a non-tripped peer of the same caste. Ready for peer reassignment.
- `CircuitBreaker` — Per-worker failure tracking, per-wave Reset(), Allow()/RecordFailure()/RecordSuccess(). The orchestrator uses this to check if a worker is still healthy enough for retry.
- `dispatchFixer()` in fixer_dispatch.go — Already handles Fixer dispatch with circuit breaker check, attempt cap, and telemetry. The orchestrator calls this as the last recovery strategy.
- `recoveryLogWritePhase()` / `recoveryLogReadPhase()` — Persistence for recovery actions. The orchestrator logs every action here.
- `RecoveryLogEntry` — Structured log entry with failure record, action taken, outcome, attempt number. Ready for budget tracking extension.

### Established Patterns
- Classification as Go map constants (failureClassifications, gateClassifications)
- Per-phase persistence via store.SaveJSON/LoadJSON with phase-scoped files
- Cobra CLI subcommands with --json flag for structured output
- OutputWorkflow pattern (outputOK, outputError)
- Atomic JSON writes via store.UpdateJSONAtomically()
- omitempty on all new struct fields for backward compatibility
- Ceremony event emission for build lifecycle visibility

### Integration Points
- Build dispatch loop — add orchestrator call after worker failure detection
- Continue finalize — add orchestrator call for Fixer-as-recovery-strategy (distinct from Phase 95's auto-resolve Fixer dispatch)
- Recovery log file — add recovery_budget tracking object
- Circuit breaker — orchestrator checks Allow() before retry, calls RecordFailure() on failure, calls Reset() at wave start

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches following established patterns.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 96-Auto-Recovery Orchestrator*
*Context gathered: 2026-05-03*
