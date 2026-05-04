# Phase 93: Gate Classification Infrastructure - Context

**Gathered:** 2026-05-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Every gate in the Aether continue flow has a deterministic classification (hard_block, soft_block, advisory) and every auto-resolution preserves the original finding in an audit trail. This is pure infrastructure — no behavioral changes to existing gates, just a classification layer that downstream phases (95: Smart Gate Pipeline) will consume.

The phase adds: (1) a classification registry mapping each gate to its classification with rationale, (2) an audit trail mechanism that annotates gate findings with queen decisions while preserving originals, (3) a CLI command to inspect classifications.

</domain>

<decisions>
## Implementation Decisions

### Gate Classification Mapping
- **D-01:** Three tiers — hard_block, soft_block, advisory — following GATE-01 requirements
- **D-02:** hard_block gates (3): gatekeeper, watcher_veto, flags — security findings and explicit human signals that must never auto-resolve. GATE-02 locks gatekeeper and watcher_veto; flags are hard_block because they represent intentional human escalation
- **D-03:** soft_block gates (6): auditor, complexity, tdd_evidence, anti_pattern, verification_loop, spawn_gate — findings that the queen can auto-resolve when verified non-critical (Phase 95 consumes this classification)
- **D-04:** advisory gates (2): medic, runtime — diagnostic/logging only, never block advancement
- **D-05:** tests_pass and no_critical_flags are pre-check gates (not continue gates) — tests_pass is hard_block (broken build), no_critical_flags is hard_block (critical errors exist)
- **D-06:** Gatekeeper and watcher_veto classifications are hardcoded — no configuration flag can change them from hard_block. This is a compile-time guarantee, not a runtime setting

### Audit Trail Preservation
- **D-07:** Extend the existing `GateCheckResult` struct with a `queen_annotation` field containing decision, rationale, and timestamp — the original finding detail, fix_hint, and recovery_options fields are never modified or deleted
- **D-08:** Audit trail lives in the existing `gate-results-{N}.json` per-phase files — no new file format. The queen_annotation is appended alongside the original data
- **D-09:** The `queen_annotation` struct contains: decision (string: "auto-resolved", "escalated", "skipped"), rationale (string: why), timestamp (RFC3339), and queen_version (string: for traceability)

### Command Surface
- **D-10:** `aether gate-classify` uses the existing OutputWorkflow pattern — human-readable table (gate name, classification, one-line rationale) plus JSON output for agent consumption
- **D-11:** `aether gate-classify --json` for structured output. Default is the table view
- **D-12:** Classification data is stored as a Go map constant in the runtime — not in a config file or COLONY_STATE.json. Classifications are deterministic and code-level, not user-configurable

### Claude's Discretion
- Exact struct field names and Go types for queen_annotation
- Table formatting for gate-classify output
- Whether to add a gate-classify subcommand or extend an existing command
- Test structure and coverage approach
- How to document the classification rationale in code comments

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Gate System (existing code to extend)
- `cmd/gate.go` — Core gate types (gateCheck, gateResult, GateCheckResult), gate check functions, recovery templates map (12 entries), alwaysRunGates map, shouldSkipGate logic
- `cmd/gate_results.go` — Gate results validation (validGateStatuses), format validation
- `cmd/gate_test.go` — Gate test patterns
- `cmd/gate_incremental_test.go` — Incremental gate result accumulation tests

### Gate Persistence (existing infrastructure)
- `pkg/colony/colony.go` — ColonyState struct with GateResults field (GateResultEntry)
- `cmd/gate.go` — gateResultsWrite() and gateResultsRead() for COLONY_STATE.json persistence
- `cmd/gate.go` — gateResultsWritePhase() and gateResultsReadPhase() for per-phase files

### Continue Flow (where gates are consumed)
- `cmd/codex_continue.go` — Continue command that runs gates
- `cmd/codex_continue_finalize.go` — Finalize phase with gate evaluation
- `cmd/fixer_dispatch.go` — Fixer agent dispatch on gate failure

### Recovery Templates (gate behavior documentation)
- `cmd/gate.go` § gateRecoveryTemplates — Maps all 12 gate names to 3-step recovery instructions. The classification rationale should align with these recovery patterns.

### Requirements
- `.planning/REQUIREMENTS.md` — GATE-01 (classify all gates), GATE-02 (hardcode security + watcher), GATE-05 (preserve original findings)
- `.planning/ROADMAP.md` § Phase 93 — Success criteria and goal definition
- `.planning/phases/89-gate-self-healing-smart-planning/89-CONTEXT.md` — Prior phase that established gate-results persistence, per-gate skip logic

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `GateCheckResult` struct in `cmd/gate.go` — Already has Name, Status, Detail, FixHint, RecoveryOptions, Timestamp, RetryCount. Adding queen_annotation field is a natural extension.
- `gateRecoveryTemplates` map — 12 entries defining each gate's recovery behavior. The classification should be consistent with the severity implied by these templates.
- `alwaysRunGates` map — 4 gates that always run regardless of prior results (tests_pass, flags, watcher_veto, no_critical_flags). These align with hard_block classification.
- OutputWorkflow pattern — `outputOK()` for JSON+visual, `outputError()` for errors. `aether gate-classify` follows this.

### Established Patterns
- Gate results persistence: upsert by Name into COLONY_STATE.json + per-phase gate-results-{N}.json files
- Cobra CLI subcommands: flags for input, JSON output via outputOK()
- Atomic JSON writes via `store.UpdateJSONAtomically()` for safe concurrent access
- `shouldSkipGate()` already differentiates gate behavior — classification extends this concept

### Integration Points
- Phase 95 (Smart Gate Pipeline) will consume the classification to decide auto-resolve vs escalate
- `gateResultsWrite()` needs to handle the new queen_annotation field in GateCheckResult
- Continue flow in `codex_continue.go` reads gate results — must handle new fields gracefully
- `gateRecoveryTemplate()` is called for failed gates — audit trail annotation happens after this

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

*Phase: 93-Gate Classification Infrastructure*
*Context gathered: 2026-05-03*
