# Phase 97: Queen-Led Continue - Discussion Log

**Date:** 2026-05-03
**Mode:** Default (interactive)

## Areas Discussed

### 1. Decision List Content
- **Q1:** What should the plan-only decision list contain?
  - Options: Gate recommendations / Gate status only / Gate status + contingency plan
  - **Selected:** Gate recommendations — each gate gets a queen recommendation (auto-resolve / dispatch-fixer / escalate / pass)
- **Q2:** How should the decision list be formatted?
  - Options: JSON array / Text table / Both JSON + text
  - **Selected:** JSON array — structured and easy for wrapper to parse
- **Q3:** Should the decision list include recovery recommendations for failed gates?
  - Options: Include recovery / Skip recovery
  - **Selected:** Include recovery — what the orchestrator would do if finalize runs
- **Q4:** Should the plan-only output include the current recovery budget state?
  - Options: Yes, include budget / No budget
  - **Selected:** Yes, include budget — wrapper knows recovery capacity before deciding

### 2. Queen Decision Scope
- **Q1:** Does the queen wrap existing decisions or make new ones?
  - Options: Wrap existing / New decision layer
  - **Selected:** Wrap existing — packages Phase 93/95/96 results, no new decision logic
- **Q2:** Should the queen add rationale text to each recommendation?
  - Options: Include rationale / No rationale
  - **Selected:** Include rationale — why she recommends each action
- **Q3:** For passing gates: should the queen show what recovery action would apply IF they fail?
  - Options: Recovery preview / Skip preview
  - **Selected:** Recovery preview — contingency visibility for all gates

### 3. Finalize Approval Flow
- **Q1:** How does finalize decide what to execute from the plan-only decision list?
  - Options: Auto-execute / Human approval gate / Gate-type dependent
  - **Selected:** Auto-execute — no human approval between plan-only and finalize
- **Q2:** How does the decision list flow from plan-only to finalize?
  - Options: Embed in manifest / Separate decisions file
  - **Selected:** Embed in manifest — decisions in the existing plan-only manifest JSON
- **Q3:** Does finalize re-evaluate gates against live results, or trust the plan-only decisions?
  - Options: Re-evaluate live / Trust plan-only
  - **Selected:** Re-evaluate live — plan-only decisions are advisory context, not commands

### 4. Single-Invocation Contract
- **Q1:** What does 'single-invocation' mean in practice?
  - Options: Function call per invocation / CLI command per invocation
  - **Selected:** Function call per invocation — no goroutine, daemon, or background process
- **Q2:** How does queen state persist between plan-only and finalize?
  - Options: Use existing files / New queen state file
  - **Selected:** New queen state file — distinct from existing manifest
- **Q3:** What should the new state file contain?
  - Options: queen-decisions-{N}.json (focused) / queen-state-{N}.json (broader)
  - **Selected:** queen-state-{N}.json — decisions + budget + recovery history + escalation log
- **Q4:** Per COORD-04: when the circuit breaker trips, should the queen log the escalation event?
  - Options: Log escalation / Existing behavior is fine
  - **Selected:** Log escalation to queen-state-{N}.json — breaker state, tripped workers, action taken

---

*Discussion completed: 2026-05-03*
