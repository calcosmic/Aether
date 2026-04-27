# Phase 59: Gate Failure Recovery - Context

**Gathered:** 2026-04-27
**Status:** Ready for planning

<domain>
## Phase Boundary

When verification gates fail during `/ant-continue`, the user gets clear, actionable recovery instructions and can fix and re-check only what failed. Three requirements: GATE-01 (actionable recovery instructions), GATE-02 (watcher veto confirmation), GATE-03 (incremental gate checking).

This phase modifies the continue verification flow (continue-verify.md, continue-gates.md playbooks) and the Go runtime (`cmd/gate.go`, `cmd/codex_continue.go`). It does NOT change build-time behavior or add new gate types.

</domain>

<decisions>
## Implementation Decisions

### Recovery Instructions (GATE-01)
- **D-01:** Per-gate recovery templates — each gate type has a hardcoded recovery template with specific steps: what to check, which command to run, and what success looks like. Not generic "fix it and re-run."
- **D-02:** Multiple gate failures shown together — all failed gates displayed with their individual recovery templates. User fixes everything in one pass, then re-runs `/ant-continue` once.

### Watcher Veto Confirmation (GATE-02)
- **D-03:** Three choices on Watcher Veto: (1) Stash changes, (2) Keep working (leave phase blocked, user fixes manually), (3) Force advance (accept the risk, advance anyway). No auto-stash without asking.
- **D-04:** Veto reason shown first (quality score + critical issues list) before presenting the three choices. User makes an informed decision.
- **D-05:** "Stash changes" runs `git stash push` + creates blocker flag (same as today's auto-behavior, but now user-triggered). "Keep working" does nothing — phase stays blocked. "Force advance" creates a FEEDBACK pheromone noting the override and proceeds to Step 2.

### Gate State Tracking (GATE-03 storage)
- **D-06:** Gate results stored in COLONY_STATE.json as a new `gate_results` field — follows the existing pattern of all colony state living in one file.
- **D-07:** Flat list schema: `[{name: "tests_pass", passed: true, timestamp: "2026-04-27T14:00:00Z", detail: "all tests passed"}, ...]`. Simple to read, easy to filter by `passed: false`.
- **D-08:** Results cleaned up when phase advances — `gate_results` gets cleared on state advance so no stale data carries over.

### Incremental Recheck UX (GATE-03 behavior)
- **D-09:** Skip summary displayed at the top ("Skipping 8 passed gates — re-checking 3 failures") then only failed gates re-run. Clean, fast output.
- **D-10:** Tests always re-run regardless of prior pass status — cheapest hard truth check and catches any code changes. Other passed gates (spawn, anti-pattern, complexity, gatekeeper, auditor, TDD, runtime, flags, veto, medic) are skipped if they passed last time.
- **D-11:** Gate results are keyed by gate name — on re-run, the planner checks for existing `passed: true` entries and skips them (except tests). Failed gates get re-run and their entries updated in-place.

### Claude's Discretion
- Exact recovery template wording for each gate type
- Exact AskUserQuestion format for the veto confirmation (labels, descriptions)
- Whether `gate_results` lives at top level of COLONY_STATE.json or nested under current phase
- How to handle the edge case where gate_results exists but the phase changed (user inserted a new phase)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements
- `.planning/REQUIREMENTS.md` — GATE-01, GATE-02, GATE-03 requirements with acceptance criteria
- `.planning/ROADMAP.md` — Phase 59 success criteria (lines 161-168)

### Gate System (existing)
- `cmd/gate.go` — `gateCheck`, `gateResult` structs, `checkTaskComplete`, `checkPhaseAdvance`, `runPreContinueGates`, individual check functions
- `.aether/docs/command-playbooks/continue-verify.md` — 6-phase verification loop, gate decision logic, "VERIFICATION FAILED" output
- `.aether/docs/command-playbooks/continue-gates.md` — Steps 1.6-1.14: Spawn gate, Anti-pattern gate, Complexity gate, Gatekeeper, Auditor, TDD gate, Runtime gate, Flags gate, Watcher Veto gate, Medic gate

### Continue Flow (existing)
- `cmd/codex_continue.go` — Continue command, review specs, dispatch logic
- `.aether/docs/command-playbooks/continue-advance.md` — State advancement, learning extraction, pheromone auto-emission
- `.aether/docs/command-playbooks/continue-finalize.md` — Continue completion ceremony

### Review Depth (Phase 58 — must maintain compatibility)
- `.planning/phases/58-smart-review-depth/58-CONTEXT.md` — Light/heavy review mode decisions that affect which gates run
- `cmd/codex_continue.go` — `codexContinueReviewSpecs`, `plannedContinueReviewDispatches` — filtered by review depth

### Colony State (storage)
- `cmd/state_cmds.go` — `state-mutate` pattern for atomic COLONY_STATE.json updates
- `pkg/colony/` — ColonyState struct definition (where `gate_results` field gets added)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `gateCheck` struct in `cmd/gate.go` — already has Name, Passed, Detail fields. Recovery templates can be attached to the gate name.
- `runPreContinueGates()` in `cmd/gate.go` — already runs pre-continue checks. Can be extended to read/write gate_results.
- `state-mutate` pattern — atomic COLONY_STATE.json updates with jq expressions. Use this for gate_results writes.
- Continue playbooks already separate gates into distinct steps — each step can read gate_results and skip if passed.

### Established Patterns
- Gate checks return structured results (pass/fail + detail). Recovery templates are a natural extension of the detail field.
- Continue playbooks run gates sequentially with hard-stop on failure. Adding "skip if passed" logic is straightforward.
- AskUserQuestion is already used in the runtime verification gate (Step 1.11) — same pattern for veto confirmation.

### Integration Points
- Continue-verify.md: The "Gate Decision" section (Step 3) writes gate_results on failure and reads them on re-run
- Continue-gates.md: Each gate step (1.6 through 1.14) checks gate_results before running. If `passed: true` (and gate name != "tests_pass"), skip with brief log.
- `cmd/gate.go`: Add `gateResultsWrite` and `gateResultsRead` helper functions
- COLONY_STATE.json: Add `gate_results` field (array of {name, passed, timestamp, detail})

</code_context>

<specifics>
## Specific Ideas

- The 14 gates in continue-gates.md that could fail: verification loop (build/types/lint/test/secrets/diff), spawn gate, anti-pattern gate, complexity gate, gatekeeper, auditor, TDD gate, runtime gate, flags gate, watcher veto, medic gate
- Each needs a 2-3 line recovery template: what failed, what to do, what command to run
- Example: "Spawn gate failed → Prime Worker completed without specialists. Run `/ant-build {phase}` again and ensure the Prime Worker spawns at least 1 specialist (Builder or Watcher). Re-run `/ant-continue` after."
- The veto confirmation replaces the auto-stash at continue-gates.md Step 1.13 with an AskUserQuestion offering 3 choices
- Gate results are written AFTER each gate runs (not batched at end) so partial results survive crashes

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 59-gate-failure-recovery*
*Context gathered: 2026-04-27*
