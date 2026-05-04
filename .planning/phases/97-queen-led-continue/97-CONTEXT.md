# Phase 97: Queen-Led Continue - Context

**Gathered:** 2026-05-03
**Status:** Ready for planning

<domain>
## Phase Boundary

The continue command's plan-only mode produces a queen decision list — a JSON array of gate recommendations with recovery actions, rationale, and budget state. The finalize mode auto-executes based on live re-evaluation using the plan-only decisions as advisory context. The queen is a single-invocation function (not a daemon) that wraps existing Phase 93 gate classification + Phase 95 auto-resolve + Phase 96 orchestrator into a unified decision output.

This phase does NOT add new gates, new classification logic, new recovery strategies, or new Fixer capabilities — it adds the queen decision layer that wraps what already exists into a structured, persisted, and logged decision format.

The existing plan-only/finalize split already works. This phase enriches both sides: plan-only gets richer output (decision list + budget), finalize gets advisory context (queen-state file) and circuit breaker escalation logging.

</domain>

<decisions>
## Implementation Decisions

### Decision List Content
- **D-01:** Plan-only output includes a JSON array of gate recommendations. Each entry has: gate name, status (pass/fail/block), classification tier (hard_block/soft_block/advisory), queen_recommendation (auto-resolve / dispatch-fixer / escalate / pass), rationale text, auto_resolve_eligible boolean.
- **D-02:** Decision list includes recovery recommendations for failed gates — what the Phase 96 orchestrator would do (retry/peer/fixer/escalate) if finalize runs.
- **D-03:** Decision list includes recovery budget state: how many retries, reassigns, and fixer dispatches remain for the current wave.
- **D-04:** Recovery preview included for ALL gates (even passing ones) — shows what would happen IF that gate fails during finalize. Gives the wrapper full contingency visibility.

### Queen Decision Scope
- **D-05:** The queen wraps existing decisions — she does NOT make new ones. She takes Phase 93 gate classifications, Phase 95 auto-resolve results, and Phase 96 orchestrator output, and packages them into a single structured decision list.
- **D-06:** The queen adds rationale text to each recommendation — why she recommends auto-resolve vs fixer vs escalate. Based on classification tier, threshold comparison, and circuit breaker state.

### Finalize Approval Flow
- **D-07:** Finalize auto-executes plan-only decisions. No human approval gate between plan-only and finalize. The wrapper runs plan-only, sees the decisions, and automatically runs finalize.
- **D-08:** The decision list is embedded in the plan-only manifest JSON (not a separate file). Finalize reads the manifest, extracts decisions as advisory context.
- **D-09:** Finalize re-evaluates gates against live results. Plan-only decisions are advisory context, not authoritative commands. This prevents stale decisions from executing if the codebase changed between plan-only and finalize runs.

### Single-Invocation Contract
- **D-10:** The queen is a function call per invocation — not a goroutine, daemon, or background process. She runs once in plan-only (produces decision list), runs once in finalize (re-evaluates and executes). State persists via files between calls.
- **D-11:** New state file: `queen-state-{phase}.json` — stores the decision list from plan-only, plus budget, recovery history, and escalation log. Finalize reads it for advisory context. This is a NEW file, distinct from the existing manifest.
- **D-12:** When the circuit breaker trips during finalize, the queen logs the escalation event to queen-state-{N}.json with: breaker state, which workers are tripped, and the escalation action taken. She never overrides or resets breaker state (per COORD-04).

### Claude's Discretion
- Exact struct names for QueenDecision, QueenStateFile, etc.
- Field names in the JSON output
- Whether the queen decision function is in a new file (cmd/queen_decision.go) or extends an existing one
- Exact format of the escalation log entry in queen-state-{N}.json
- Test structure and coverage approach
- Whether a CLI command for inspecting queen state is needed in this phase
- How to handle the case where plan-only decisions conflict with live finalize re-evaluation

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Continue Flow (where the queen wires in)
- `cmd/codex_continue.go` -- Continue command, runCodexContinue() function, full verification/review/advance flow
- `cmd/codex_continue_finalize.go` -- runCodexContinueFinalize() function, gate evaluation, auto-resolve, orchestrator integration (line 367), finalizeBlockedExternalContinue
- `cmd/codex_continue_plan.go` -- runCodexContinuePlanOnly() function, plan-only manifest creation

### Gate Classification (Phase 93 — consumed for decision list)
- `cmd/gate.go` -- GateClassificationTier, gateClassify(), gateClassifications map, isHardBlockGate(), GateCheckResult with QueenAnnotation
- `cmd/gate.go` -- gateResultsWritePhase() / gateResultsReadPhase() for per-phase persistence

### Smart Gate Pipeline (Phase 95 — auto-resolve decisions)
- `cmd/codex_continue_finalize.go` -- autoResolveSoftBlockGates(), depth-based threshold evaluation
- `cmd/gate.go` -- soft_block thresholds, depth multiplier integration

### Auto-Recovery Orchestrator (Phase 96 — recovery recommendations)
- `cmd/recovery_orchestrator.go` -- orchestrateRecovery(), RecoveryContext, RecoveryOutcome, RecoveryAction, RecoveryBudget
- `cmd/recovery_orchestrator.go` -- newRecoveryBudget(), budgetFromRecoveryLog(), persistBudgetToRecoveryLog()
- `cmd/recovery_classify.go` -- FailureClassification, classifyWorkerFailure(), RecoveryLogEntry, RecoveryLogFile

### Circuit Breaker (Phase 94 — escalation logging)
- `cmd/circuit_breaker.go` -- CircuitBreaker struct, Allow(), RecordFailure(), Reset(), findSameCastePeer()
- `cmd/circuit_breaker.go` -- globalCircuitBreaker usage pattern

### Prior Phase Context
- `.planning/phases/95-smart-gate-pipeline/95-CONTEXT.md` -- Auto-resolve decisions, threshold configuration, Fixer dispatch
- `.planning/phases/96-auto-recovery-orchestrator/96-CONTEXT.md` -- Recovery sequence, budget tracking, build/continue wiring
- `.planning/phases/96-auto-recovery-orchestrator/96-01-SUMMARY.md` -- What Phase 96 built (orchestrator function)
- `.planning/phases/96-auto-recovery-orchestrator/96-02-SUMMARY.md` -- What Phase 96 built (build/continue wiring)

### Requirements
- `.planning/REQUIREMENTS.md` -- COORD-02 (plan-only/finalize split), COORD-03 (single-invocation), COORD-04 (circuit breaker respect)
- `.planning/ROADMAP.md` Phase 97 -- Goal and success criteria

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `runCodexContinuePlanOnly()` already produces a manifest — the queen decision list extends this output, doesn't replace it
- `gateClassify()` returns tier for any gate name — the queen wraps this into a recommendation
- `orchestrateRecovery()` returns RecoveryOutcome with action + rationale — the queen wraps this into the recovery preview
- `RecoveryBudget` already tracks retries/reassigns/fixer counts — the queen includes this in the decision list
- `QueenAnnotation` struct from Phase 93 — may be reused for the queen's rationale text
- `gateResultsWritePhase()` / `gateResultsReadPhase()` — pattern for per-phase JSON persistence (queen-state follows this pattern)

### Established Patterns
- Classification as Go map constants (gateClassifications, failureClassifications)
- Per-phase persistence via store.SaveJSON/LoadJSON with phase-scoped files
- Cobra CLI subcommands with --json flag for structured output
- OutputWorkflow pattern (outputOK, outputError)
- Plan-only manifest with DispatchMode: "plan-only" + RequiresFinalizer: true
- Result map construction in finalize (result["key"] = value pattern)

### Integration Points
- Plan-only output: add "queen_decisions" array to the result map from runCodexContinuePlanOnly()
- Finalize input: read queen-state-{N}.json for advisory context during runCodexContinueFinalize()
- Circuit breaker: queen logs escalation events to queen-state when breaker trips
- Recovery budget: queen reads current budget from recovery-log and includes in decision list

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches following established patterns.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

---

*Phase: 97-Queen-Led Continue*
*Context gathered: 2026-05-03*
