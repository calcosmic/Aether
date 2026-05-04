# Phase 95: Smart Gate Pipeline - Context

**Gathered:** 2026-05-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Soft_block gates auto-resolve when the queen verifies the finding is non-critical, with configurable severity thresholds and documented safe defaults. Hard_block gates (gatekeeper, watcher_veto, flags, tests_pass, no_critical_flags) are never auto-resolved — they always require user intervention. Advisory gates log but never block.

This phase wires the gate classification from Phase 93 into the continue flow. When a gate fails during continue, soft_block gates are evaluated against per-gate thresholds. If the finding is below threshold, the queen auto-resolves it with an audit annotation. If above threshold, the Fixer agent is dispatched automatically. All annotations are written to the existing gate-results-{N}.json files.

The phase does NOT change gate check logic, add new gates, or modify hard_block/advisory behavior.

</domain>

<decisions>
## Implementation Decisions

### Auto-Resolve Criteria
- **D-01:** Auto-resolve is threshold-based: each soft_block gate has a numeric threshold. If the gate's score/detail is below threshold, auto-resolve. No LLM judgment, no history-based resolution.
- **D-02:** Auto-resolve runs inside the existing continue command — no new commands for the resolve flow itself. The continue command's gate check loop gains an auto-resolve step after gate failure.
- **D-03:** When auto-resolve fails (gate finding is too severe), the Fixer agent is dispatched automatically. The Fixer's outcome is logged as a separate annotation.
- **D-04:** Auto-resolve applies ONLY to soft_block gates. Hard_block gates always block. Advisory gates never block.

### Threshold Configuration
- **D-05:** Per-gate thresholds live in a Go map constant (like gateClassifications from Phase 93). Each soft_block gate has its own threshold value appropriate to its scale. Hardcoded constants, not configurable per-colony.
- **D-06:** Auto-resolve behavior ties into the existing verification depth system: light depth = more aggressive auto-resolve, heavy depth = more conservative. No new flags needed. The depth model already controls how aggressive verification is.
- **D-07:** A new `aether gate-auto-resolve` CLI command shows current thresholds and their rationale (mirrors `gate-classify` pattern from Phase 93). Useful for debugging and documentation.

### Queen Annotation Flow
- **D-08:** Reuse the existing QueenAnnotation struct from Phase 93 (decision, rationale, timestamp, queen_version). Auto-resolve writes "auto-resolved" or "fixer-dispatched" as the decision field.
- **D-09:** Annotations are written to the same gate-results-{N}.json file alongside the original finding. The existing gateResultsWritePhase() function handles persistence. No new file format.
- **D-10:** Auto-resolve annotation happens in-place: read gate results, annotate failed soft_block gates, write back. The original finding (detail, fix_hint, recovery_options) is never modified — only the queen_annotation field is added/updated.

### Claude's Discretion
- Exact threshold values for each soft_block gate (auditor, complexity, tdd_evidence, anti_pattern, verification_loop, spawn_gate)
- How verification depth maps to auto-resolve aggressiveness (which thresholds relax at light vs heavy depth)
- Table formatting for gate-auto-resolve CLI command output
- Whether gate-auto-resolve also accepts --json flag (likely yes, following pattern)
- Fixer dispatch flow details (how long to wait, how to handle Fixer failure)
- Test structure and coverage approach
- How to handle the case where auto-resolve + Fixer both fail (escalate to user)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Gate Classification (Phase 93 output — consumed by this phase)
- `cmd/gate.go` -- GateClassificationTier type, gateClassifications map, gateClassify() function, isHardBlockGate(), GateCheckResult struct with QueenAnnotation field
- `cmd/gate.go` -- gateResultsWritePhase() / gateResultsReadPhase() for per-phase persistence
- `cmd/gate.go` -- gateClassifyCmd Cobra command pattern and renderGateClassifyTable() go-pretty pattern

### Continue Flow (where auto-resolve wires in)
- `cmd/codex_continue.go` -- Continue command that runs gates, reads prior gate results, evaluates pass/fail
- `cmd/codex_continue_finalize.go` -- Finalize phase with gate evaluation
- `cmd/codex_continue_plan.go` -- Plan-only continue mode

### Fixer Dispatch (auto-triggered on auto-resolve failure)
- `cmd/fixer_dispatch.go` -- Fixer agent dispatch on gate failure
- `cmd/fixer_dispatch_test.go` -- Fixer dispatch test patterns

### Recovery Data Model (Phase 94 — parallel context)
- `cmd/recovery_classify.go` -- FailureClassification, classifyWorkerFailure(), recovery log persistence patterns

### Prior Phase Context
- `.planning/phases/93-gate-classification-infrastructure/93-CONTEXT.md` -- Gate classification decisions, audit trail preservation, command surface
- `.planning/phases/93-gate-classification-infrastructure/93-01-SUMMARY.md` -- What Phase 93 actually built
- `.planning/phases/94-recovery-data-model/94-CONTEXT.md` -- Recovery data model context (parallel infrastructure)

### Requirements
- `.planning/REQUIREMENTS.md` -- GATE-03 (soft_block auto-resolve), GATE-04 (configurable thresholds), GATE-05 (audit trail preservation)
- `.planning/ROADMAP.md` Section Phase 95 -- Goal and success criteria

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `GateCheckResult` struct with `QueenAnnotation` field already added by Phase 93 — ready for auto-resolve annotations
- `gateClassify()` function returns the tier for any gate name — auto-resolve logic simply checks `tier == softBlock`
- `gateResultsWritePhase()` / `gateResultsReadPhase()` — atomic JSON read-modify-write for gate results
- `gateClassifyCmd` pattern — same Cobra + go-pretty table pattern for the new `gate-auto-resolve` command
- `isHardBlockGate()` — quick check to skip auto-resolve for hard_block gates

### Established Patterns
- Classification as Go map constants (gateClassifications, failureClassifications)
- Per-phase persistence via store.SaveJSON/LoadJSON with gate-results-{N}.json
- Cobra CLI subcommands with --json flag for structured output
- OutputWorkflow pattern (outputOK, outputError)
- Verification depth model already controls gate check behavior in continue

### Integration Points
- Continue command's gate check loop (codex_continue.go) — add auto-resolve step after gate failure
- Gate results files (gate-results-{N}.json) — auto-resolve annotations written here
- Fixer dispatch (fixer_dispatch.go) — triggered when auto-resolve threshold not met
- Verification depth system — auto-resolve aggressiveness ties to depth level

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

*Phase: 95-Smart Gate Pipeline*
*Context gathered: 2026-05-03*
