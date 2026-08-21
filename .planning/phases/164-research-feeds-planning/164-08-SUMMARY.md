---
phase: 164-research-feeds-planning
plan: 08
subsystem: research-iteration (Go verb + ts-host confidence loop)
tags: [oracle, escalation, permission-profile, confidence-loop, D-04, RESEARCH-07]

# Dependency graph
requires:
  - phase: 164-05
    provides: "computePhaseResearchProposalFields, approval-gated plannedPhaseResearchDispatches, and the phaseResearchStage/phaseResearchCandidate types this plan's dispatch builder mirrors"
  - phase: 164-06
    provides: "runResearchConfidenceLoop -- the per-phase ConfidenceLoop construction site this plan adds the escalation branch to, plus renderIterationCeremony's prefix parameter"
provides:
  - "pkg/codex/permission_profile.go: an explicit \"oracle\" case in behavioralRestrictionsForCaste scoping writes to .aether/oracle and .aether/data/phase-research (previously fell through to an unrestricted default)"
  - "cmd/phase_research_escalate.go: phaseResearchEscalationDispatch (the escalation dispatch builder) and the `aether plan-research-escalate` cobra verb"
  - ".aether/ts-host/src/host.ts: the escalation branch inside runResearchConfidenceLoop -- stall at deep/exhaustive depth triggers one plan-research-escalate call, dispatched once outside any ConfidenceLoop"
affects: [164-09, autopilot-run, oracle-lifecycle]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Escalation-round-after-the-loop: rather than escalating inline inside the per-phase while loop, finished phases needing escalation are collected into escalationCandidates and dispatched together in one extra round after the main loop drains -- keeps 'exactly one dispatch, no second loop' structurally obvious rather than threaded through iteration state"
    - "Dispatch-builder mirroring: phaseResearchEscalationDispatch copies the plannedPhaseResearchDispatches struct literal field-by-field, changing only caste/agent_name/name/task_id, so the two dispatch shapes cannot silently diverge on Stage/Wave/Outputs (the fields D-07's in-place overwrite depends on)"

key-files:
  created:
    - cmd/phase_research_escalate.go
    - cmd/phase_research_escalate_test.go
  modified:
    - pkg/codex/permission_profile.go
    - .aether/ts-host/src/host.ts
    - .aether/ts-host/test/research-confidence-loop.test.ts
    - .aether/ts-host/dist/host.js
    - .aether/ts-host/dist/host.d.ts
    - cmd/testdata/command_catalog.json
    - cmd/testdata/parity_snapshot.json
    - cmd/testdata/regression_snapshot.json

key-decisions:
  - "The oracle behavioral restriction is a literal relative-path string (\".aether/oracle\" and \".aether/data/phase-research\"), not a value computed by calling oracleWorkspacePaths(root) at runtime -- pkg/codex cannot import cmd (would be circular), so the plan's \"read it, do not guess\" instruction was satisfied by copying the exact relative subpath from oracleWorkspacePaths's source rather than deriving it dynamically"
  - "Escalation is collected during the main while loop but dispatched in a separate round after the loop drains, not inline when a phase finishes -- this keeps the invariant 'the escalated dispatch is not wrapped in a new ConfidenceLoop' structurally true by construction: the escalation dispatch never enters the loop/active map at all"
  - "plan-research-escalate loads the current colony state via loadActiveColonyState (same helper the plan-only path uses) rather than accepting phase name/description over flags -- this guarantees the escalated Oracle's brief describes the same phase content the original Scout brief did, with zero risk of the TS host and Go verb disagreeing on phase metadata"

patterns-established:
  - "Escalation ceremony line format: \"Oracle escalation: phase N research stalled at X% against a Y% target -- escalating Scout to Oracle\" -- always states phase, stalled confidence, target, and the caste swap in one sentence (D-02 applied to this sub-decision)"

requirements-completed: [RESEARCH-07]

# Metrics
duration: ~70min
completed: 2026-08-02
---

# Phase 164 Plan 08: Oracle Escalation on Stalled Research Summary

**A phase's research that stalls below its confidence target at deep or exhaustive depth escalates from Scout to Oracle exactly once, with a stated reason and a write promise no broader than the Scout it replaces — closing D-04, the last unbuilt piece of Phase 164's research iteration story.**

## Performance

- **Duration:** ~70 min
- **Tasks:** 2/2 completed
- **Files modified:** 10 (2 created, 8 modified)

## Accomplishments

- `pkg/codex/permission_profile.go` now has an explicit `"oracle"` case in `behavioralRestrictionsForCaste`, scoping writes to `.aether/oracle` and `.aether/data/phase-research`. Before this change, `"oracle"` fell through to `default: return nil` — an escalated Oracle would have carried zero write restriction, silently broadening access beyond the Scout it replaced (T-164-22, closed in the same change that introduces escalation, not after)
- `cmd/phase_research_escalate.go` adds `phaseResearchEscalationDispatch` (mirrors the Scout dispatch literal, changing only caste/agent_name/name/task_id) and the `aether plan-research-escalate --phase N --confidence X --target Y` cobra verb, which loads the current colony state, matches the phase, and prints the dispatch as a JSON envelope — or a clean `outputError` for an unknown phase ID, never a panic
- `runResearchConfidenceLoop` in `.aether/ts-host/src/host.ts` now evaluates the escalation trigger after each phase's own `ConfidenceLoop` stops: `stopReason === "diminishing_returns"` AND `currentConfidence < confidenceTarget` AND depth is `deep` or `exhaustive`. All three must hold; `confidence_target_met` and `max_iterations_met` never escalate regardless of depth or confidence
- Escalation is collected during the per-phase loop but dispatched in one dedicated round *after* the loop drains — the escalated Oracle dispatch is never wrapped in a second `ConfidenceLoop`; Oracle's own RALF loop (`runOracleLoop`, `cmd/oracle_loop.go`) drives the rest. The `new ConfidenceLoop(` construction-site count in `host.ts` stays at 2 (build path + research path), now locked by an exact-equality test
- Every escalation is announced through the existing ceremony output channel: `"Oracle escalation: phase N research stalled at X% against a Y% target — escalating Scout to Oracle"` — never silent, matching D-02's philosophy applied to this sub-decision

## Task Commits

Each task was committed atomically:

1. **Task 1: Scoped oracle permissions and the escalation dispatch builder** - `9fbca3ff` (feat)
2. **Task 2: The TS trigger — stall at deep or exhaustive escalates once** - `34991072` (feat)

_No architectural deviations; both tasks landed as specified._

## Files Created/Modified

- `pkg/codex/permission_profile.go` - Added `case "oracle":` to `behavioralRestrictionsForCaste`, closing the elevation-of-privilege gap (T-164-22)
- `cmd/phase_research_escalate.go` - New: `phaseResearchEscalationDispatch` (dispatch builder) and `plan-research-escalate` cobra command (`--phase`, `--confidence`, `--target`)
- `cmd/phase_research_escalate_test.go` - New: `TestOracleEscalationKeepsScopedWriteAccess` (permission scope) and `TestOracleEscalationDispatchNamesTheStall` (dispatch shape, name collision avoidance, brief content, CLI success and unknown-phase-error paths)
- `.aether/ts-host/src/host.ts` - `runResearchConfidenceLoop` gained the escalation branch (`isEscalationEligibleDepth`, `escalationCandidates` collection, escalation dispatch round); `ResearchLoopSummary.escalations` docstring updated to describe the real contract instead of "Plan 08 populates it"
- `.aether/ts-host/test/research-confidence-loop.test.ts` - New `describe("Oracle escalation (D-04, Plan 08)")` block: 5 behavioural tests (deep-depth stall escalates once, balanced/fast never escalate, confidence_target_met never escalates, max_iterations_met never escalates, escalation announcement content) plus a second construction-site invariant test asserting the `new ConfidenceLoop(` count is exactly 2 (not just `>= 2`)
- `.aether/ts-host/dist/host.js`, `dist/host.d.ts` - Rebuilt `go:embed` bundle
- `cmd/testdata/command_catalog.json`, `parity_snapshot.json`, `regression_snapshot.json` - Golden snapshots refreshed (`-update-golden`) for the new `plan-research-escalate` command (401 total commands, up from 400)

## Decisions Made

See `key-decisions` in frontmatter above:
- Oracle's behavioral restriction is a literal relative path copied from `oracleWorkspacePaths`'s source (no cross-package call, since `pkg/codex` cannot import `cmd`)
- Escalation dispatch happens in a dedicated round after the main confidence loop drains, not inline — makes "no second ConfidenceLoop" true by construction
- `plan-research-escalate` re-derives phase metadata from colony state rather than accepting it over flags, guaranteeing the escalated brief matches the original Scout brief's phase content

## Deviations from Plan

None — plan executed exactly as written, including both threat-model mitigations (T-164-22 permission scoping, T-164-23 phase-ID validation, T-164-24 at-most-once escalation gating).

## Issues Encountered

None. The golden-snapshot refresh for the new `plan-research-escalate` command followed the same mechanical pattern established by Plan 05 (three tests exist specifically to catch total-command-count drift and self-document the `-update-golden` fix).

## User Setup Required

None — no external service configuration required.

## Verification

- `go test ./cmd/... ./pkg/codex/... -run 'TestOracleEscalation|TestPermissionProfile' -count=1` — all pass
- `go run ./cmd/aether plan-research-escalate --help` — exits 0
- `go test ./... -count=1` — all 20 packages pass (full repo, not just the two touched)
- `npm --prefix .aether/ts-host exec -- tsx --test .aether/ts-host/test/research-confidence-loop.test.ts` — 13/13 pass
- `npm --prefix .aether/ts-host run test:all` — 548/548 pass (no regressions from Plan 06's 542)
- `npm --prefix .aether/ts-host run typecheck` — exits 0
- `npm --prefix .aether/ts-host run build` — exits 0, `dist/` regenerated and committed
- `go build ./cmd/aether` — exits 0 (the `//go:embed` bundle resolves)
- `git diff --stat .aether/ts-host/src/confidence-loop.ts` — empty (untouched, per RESEARCH-07's "do not reimplement the loop" constraint)
- `grep -c 'new ConfidenceLoop(' .aether/ts-host/src/host.ts` — 2 (unchanged from Plan 06)

## Threat Flags

None. All three threats in this plan's own `<threat_model>` were implemented as specified:
- T-164-22 (Elevation of Privilege, scout->oracle broadening write access): closed by the new `"oracle"` case in `behavioralRestrictionsForCaste`, verified by `TestOracleEscalationKeepsScopedWriteAccess` asserting the restriction is non-empty and mentions `.aether/data/phase-research`
- T-164-23 (Tampering, phase ID crossing host->Go boundary): `plan-research-escalate --phase` is parsed as an int and matched against candidates derived from colony state; an unmatched ID returns `outputError`, never a panic — verified by the `unknown_phase_returns_clean_error_not_panic` subtest
- T-164-24 (Denial of Service, repeated escalation of the same phase): a phase is added to `escalationCandidates` at most once (only when it finishes and is deleted from `active`), and the dispatch round runs once after the loop drains — verified by the deep-depth test asserting exactly one `plan-research-escalate` call and one extra dispatch round

## Next Phase Readiness

- `PermissionProfileForCaste("oracle")`, `aether plan-research-escalate`, and the TS host's escalation branch are all live and tested; D-04 ("Scout by default, Oracle on escalation") is now fully wired end to end
- `go test ./... -count=1`, `npm --prefix .aether/ts-host run test:all`, `go build ./cmd/aether`, and `go run ./cmd/aether plan-research-escalate --help` all pass
- No blockers

---
*Phase: 164-research-feeds-planning*
*Completed: 2026-08-02*

## Self-Check: PASSED

- FOUND: cmd/phase_research_escalate.go
- FOUND: cmd/phase_research_escalate_test.go
- FOUND: pkg/codex/permission_profile.go (modified)
- FOUND: .aether/ts-host/src/host.ts (modified)
- FOUND: .aether/ts-host/test/research-confidence-loop.test.ts (modified)
- FOUND: .aether/ts-host/dist/host.js (modified)
- FOUND: .aether/ts-host/dist/host.d.ts (modified)
- FOUND: cmd/testdata/command_catalog.json (modified)
- FOUND: cmd/testdata/parity_snapshot.json (modified)
- FOUND: cmd/testdata/regression_snapshot.json (modified)
- FOUND commit 9fbca3ff: feat(164-08): scoped oracle permissions and escalation dispatch builder
- FOUND commit 34991072: feat(164-08): escalate stalled deep/exhaustive research to Oracle
