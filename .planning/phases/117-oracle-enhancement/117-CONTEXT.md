# Phase 117: Oracle Enhancement - Context

**Gathered:** 2026-05-13
**Status:** Ready for planning

## Phase Boundary

Enhance the Go Oracle RALF (Research-Analyze-Learn-Formulate) loop with three capabilities: phase-aware prompt directives, diminishing-returns detection via novelty-delta tracking, and template-specific synthesis sections. The Oracle is implemented entirely in the Go runtime (`cmd/oracle_loop.go`). The TS host receives ceremony events but does not own Oracle logic.

## Implementation Decisions

### D-01: Go-Only Implementation
- **Decision:** All three enhancements (phase-aware prompts, diminishing returns, template synthesis) are implemented in Go (`cmd/oracle_loop.go`), not in the TS host.
- **Why:** The Oracle owns state, iteration control, and worker dispatch. Building a parallel Oracle in TS would violate the boundary contract.

### D-02: Ceremony Events for Visibility
- **Decision:** Go emits `ceremony.oracle.phase_transition` and `ceremony.oracle.iteration` events so the TS host narrator and dashboard can render Oracle progress.
- **Why:** Keeps Go as the source of truth while letting the TS host present Oracle state to the user.

### D-03: Novelty Delta for Diminishing Returns
- **Decision:** Track keyword overlap between consecutive iterations. If novelty drops below 15% for 3 consecutive iterations, the Oracle stops with a `ceremony.loop.break` event.
- **Why:** Simple, deterministic, and does not require LLM-based semantic comparison.

### D-04: Template Branching in Synthesis
- **Decision:** `writeOracleSynthesisReport` branches on `state.Template` to select the appropriate synthesis structure (tech evaluation, architecture review, bug investigation, etc.).
- **Why:** Reference templates already exist in `.aether/references/templates/`.

## Canonical References

- `.planning/phases/117-oracle-enhancement/117-RESEARCH.md` — Full research with Go code references
- `cmd/oracle_loop.go` — Oracle controller loop (~3,200 lines)
- `cmd/oracle_cmd.go` — Oracle CLI commands
- `.aether/utils/oracle/oracle.md` — Oracle instructions
- `.aether/references/templates/oracle-tech-evaluation.md` — Tech evaluation template
- `.aether/references/templates/architecture-review-template.md` — Architecture review template
- `.aether/references/templates/bug-investigation-template.md` — Bug investigation template
- `pkg/events/ceremony.go` — Ceremony event bus

## Existing Code Insights

### Reusable Assets
- **Oracle loop** — `runOracleLoop`, `nextOraclePhase`, `buildOracleWorkerConfig`, `writeOracleSynthesisReport` all exist.
- **Ceremony emitter** — `emitLoopBreakEvent` already exists; new events can follow the same pattern.
- **Agent definition** — `.claude/agents/ant/aether-oracle.md` defines the Oracle agent.

### Integration Points
- **TS host narrator** — already handles `ceremony.loop.break`. New `ceremony.oracle.*` topics need handler entries.
- **TS host dashboard** — can display Oracle iteration counts and phase names if events are emitted.
