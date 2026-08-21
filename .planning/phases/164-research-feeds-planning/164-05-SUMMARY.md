---
phase: 164-research-feeds-planning
plan: 05
subsystem: cmd (Go runtime — plan-time research decision wiring)
tags: [research, queen-decision, pending-decision, dispatch-gating, cobra]

requires:
  - phase: 164-01
    provides: "survey context threaded into the research brief and the reresearch=opts.Refresh&&iteration==1 replan rule"
  - phase: 164-02
    provides: "computePhaseResearchProposal, phaseResearchProposal/Recommendation types, renderPhaseResearchProposalBlock, newPhaseResearchDecision/phaseResearchDecisionResolution/resolvePhaseResearchDecisions"
provides:
  - "cmd/phase_research_decision_cmd.go: `aether plan-research-approve` (--approve-all, --flip, --auto) resolving research-decisions in pending-decisions.json"
  - "cmd/codex_plan.go: computePhaseResearchProposalFields — proposal computation, decision persistence (supersede-not-accumulate), and the four research_* result/manifest fields, wired into both the existing-plan and fresh-plan branches of runCodexPlanPlanOnly"
  - "cmd/phase_research.go: plannedPhaseResearchDispatches gated on an approved map[int]bool as its first (and now only) skip condition ahead of the re-research and candidate-availability gates"
affects: [164-09, autopilot-run]

tech-stack:
  added: []
  patterns:
    - "Recommendation records reconstructed from PendingDecision.Description text (fixed '<recommend> phase <id> (<name>): <reason>' shape) rather than a second stored field — keeps RESEARCH-06's no-new-store constraint literal"
    - "Manifest-fields-as-struct-method pattern (phaseResearchProposalResult.resultFields()) to keep the four json keys byte-identical between the result map and the typed manifest struct"
    - "Supersede-not-accumulate persistence: unresolved decisions of a given type+scope are dropped and rebuilt from the fresh proposal on every persisting call; resolved decisions are always left untouched"

key-files:
  created:
    - cmd/phase_research_decision_cmd.go
  modified:
    - cmd/codex_plan.go
    - cmd/phase_research.go
    - cmd/phase_research_dispatch_test.go
    - cmd/testdata/command_catalog.json
    - cmd/testdata/parity_snapshot.json
    - cmd/testdata/regression_snapshot.json

key-decisions:
  - "T-164-13: --flip tokens are strconv.Atoi-parsed and matched only against phase IDs present in the current unresolved research-decision set; unparsable/unknown tokens are reported back in invalid_flips but never mutate a record"
  - "The early-return (existing-plan, no-refresh) branch computes the same four research_* fields read-only (persist=false) rather than writing fresh decisions on every quick re-check, avoiding decision churn on a path that isn't actually re-planning"
  - "planDepth stays in plannedPhaseResearchDispatches's signature even though the fast-depth branch is gone from its body — the depth is now read entirely inside computePhaseResearchProposal's recommendation logic (Plan 02), and removing the parameter would be unrelated signature churn"

requirements-completed: [RESEARCH-01, RESEARCH-02, RESEARCH-06]

duration: ~55min
completed: 2026-08-02
---

# Phase 164 Plan 05: Wire the Queen's Research Decision to Dispatch Summary

**`aether plan-research-approve` answers the whole per-phase research batch in one call; the plan manifest proposes, persists, and warns; dispatch now spawns a Scout only for phases the user (or autopilot) actually approved.**

## Performance

- **Duration:** ~55 min
- **Tasks:** 3 completed
- **Files modified:** 6 (1 created, 5 modified)

## Accomplishments

- `aether plan-research-approve` exists and answers the entire research batch (`--approve-all`, `--flip 2,4`, or `--auto`) in one call, with the card's default action being acceptance when no flag is given
- The plan-only result map and `codexPlanManifest` both carry `research_proposal`, `research_proposal_card`, `research_awaiting_approval`, and `research_warning` in every branch (existing-plan early-return and fresh-plan), and a re-run supersedes rather than duplicates unresolved decisions for the same phase (D-06)
- An unanswered batch produces a loud, non-blocking `research_warning` — `runCodexPlanPlanOnly` never errors on it (D-08 / Phase 160 enrichment classification)
- `plannedPhaseResearchDispatches` now dispatches a Scout only for phases present and `true` in an `approved map[int]bool`; the old unconditional `planDepth == "fast"` skip is gone — a phase flipped on during a fast run still runs its research at the fast preset's budget (D-15)

## Task Commits

Each task was committed atomically:

1. **Task 1: The plan-research-approve command** - `0178b36e` (feat)
2. **Task 2: The manifest proposes and records; unanswered batches warn loudly** - `a5900925` (feat)
3. **Task 3: Dispatch only the approved phases** - `0e55dcbb` (feat, includes golden snapshot refresh for the new command)

## Files Created/Modified

- `cmd/phase_research_decision_cmd.go` - New `plan-research-approve` cobra command: parses `--flip`, reconstructs recommendations from stored `PendingDecision.Description`, resolves decisions, and reports `approved`/`skipped`/`overrides`/`log_line`
- `cmd/codex_plan.go` - `phaseResearchProposalResult` + `computePhaseResearchProposalFields` (proposal, card, awaiting-approval, warning, approved map); wired into both branches of `runCodexPlanPlanOnly`; four new fields on `codexPlanManifest`
- `cmd/phase_research.go` - `plannedPhaseResearchDispatches` gained an `approved map[int]bool` parameter; fast-depth unconditional skip removed in favor of the approval gate alone; doc comment restates the three gates in order
- `cmd/phase_research_dispatch_test.go` - `TestPlanResearchApproveRecordsDecisions` (6 sub-behaviours), `TestPlanManifestCarriesResearchProposal` + `TestPlanManifestWarnsWhenResearchBatchUnanswered` (6 sub-behaviours), `TestResearchDispatchGatedOnApproval` (6 sub-behaviours), and all pre-existing `plannedPhaseResearchDispatches` call sites updated for the new signature; `TestPlanFastPresetSkipsPhaseResearch` renamed to `TestFastPresetSkipsUnlessUserFlipsResearchOn`
- `cmd/testdata/command_catalog.json`, `cmd/testdata/parity_snapshot.json`, `cmd/testdata/regression_snapshot.json` - Golden snapshots refreshed (`-update-golden`) to include the new `plan-research-approve` command (400 total commands, up from 399)

## Decisions Made

- Recommendation reconstruction parses `PendingDecision.Description`'s fixed shape (`"<recommend> phase <id> (<name>): <reason>"`) via manual string splitting rather than regex, avoiding greedy-match ambiguity when reason text itself could contain `"): "`
- The early-return branch of `runCodexPlanPlanOnly` calls `computePhaseResearchProposalFields` with `persist=false` — it reads existing decisions to compute `research_awaiting_approval`/`research_warning` but does not write fresh unresolved decisions, since that branch is a quick re-check (no refresh, no new iteration), not an actual re-plan
- `approved` slices in `plan-research-approve`'s JSON result are always initialized as `[]int{}` (never left `nil`) so the envelope always serializes to a JSON array, not `null`

## Deviations from Plan

None — plan executed exactly as written. One necessary addition beyond the plan's explicit action text: Task 3's golden-snapshot refresh (`cmd/testdata/command_catalog.json`, `parity_snapshot.json`, `regression_snapshot.json`) was not called out in the plan's action steps but was required — adding the `plan-research-approve` cobra command in Task 1 shifted `TestRegressionSnapshot`'s command count from 399 to 400 and drifted the catalog/parity goldens; all three were regenerated with `-update-golden` and diffed to confirm only the expected new-command entries changed (Rule 3 — blocking test failure caused directly by this plan's own Task 1 addition).

## Issues Encountered

None beyond the golden-snapshot drift noted above, which was expected and mechanical (three tests exist specifically to catch total-command-count drift and self-document the `-update-golden` fix).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `aether plan-research-approve`, the manifest's four `research_*` fields, and approval-gated dispatch are all live and tested; ready for Plan 09's wrapper to surface the proposal card and warning to the user
- `go test ./cmd/... -count=1`, `go build ./cmd/aether`, `go vet ./cmd/...`, and `go run ./cmd/aether plan-research-approve --help` all pass
- No blockers

---
*Phase: 164-research-feeds-planning*
*Completed: 2026-08-02*

## Self-Check: PASSED

- FOUND: cmd/phase_research_decision_cmd.go
- FOUND: cmd/codex_plan.go (modified)
- FOUND: cmd/phase_research.go (modified)
- FOUND: cmd/phase_research_dispatch_test.go (modified)
- FOUND: cmd/testdata/command_catalog.json (modified)
- FOUND: cmd/testdata/parity_snapshot.json (modified)
- FOUND: cmd/testdata/regression_snapshot.json (modified)
- FOUND commit 0178b36e: feat(164-05): add plan-research-approve command
- FOUND commit a5900925: feat(164-05): manifest proposes research and warns when unanswered
- FOUND commit 0e55dcbb: feat(164-05): dispatch research Scouts only for approved phases
