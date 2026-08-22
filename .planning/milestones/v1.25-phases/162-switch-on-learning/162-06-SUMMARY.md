---
phase: 162-switch-on-learning
plan: 06
subsystem: docs
tags: [documentation, adr, decision-record, learning-pipeline, regression-guard]

# Dependency graph
requires:
  - phase: 162-01
    provides: "AETHER_HIVE_POLICY single-switch implementation and AGENTS.md hive gating rewrite this plan's Decision 4 documents and does not re-edit"
  - phase: 162-03
    provides: "runPhaseEndConsolidation and the tests this plan's Decision 1/Consequences cite as proof phase-end consolidation is wired"
  - phase: 162-04
    provides: "runSealConsolidation, the D-09 seal writer reconciliation, and the tests this plan's Decision 2 cites as proof"
provides:
  - ".aether/docs/learning-system-authority.md — the ADR-style decision record for LEARN-03 (authority + duplicated stages + LearningValidator) and LEARN-04 (Hive Brain default), every runtime claim pinned to a named passing test"
  - "cmd/docs_truth_test.go — TestLearningDocsDoNotClaimUnwiredConsolidation and TestLearningDecisionRecordExists, guarding against the retired claims or the decision record itself rotting back"
  - "Corrected .aether/docs/structural-learning-stack.md, CLAUDE.md, AGENTS.md — no longer claim consolidation is unwired; the pre-existing false phase-end ant-list claim is fixed"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Documentation-claim regression guard: a table-driven Go test asserts markdown files under a fixed relative path do not contain retired substrings, skip gracefully only when the subject file's absence is itself ambiguous (vendored package / unusual cwd), and fail hard when the file's absence IS the regression being guarded against (the decision record's own existence check)"
    - "Every claim in a decision record traces to one specific named test in the same commit range, so 'uncheckable claim' has no home to hide in"

key-files:
  created:
    - .aether/docs/learning-system-authority.md
    - cmd/docs_truth_test.go
  modified:
    - .aether/docs/structural-learning-stack.md
    - CLAUDE.md
    - AGENTS.md

key-decisions:
  - "pkg/memory is authoritative; pkg/learn/learn.go+curator.go+colony_store.go (Entry/Evidence capture) is explicitly subordinate as the live capture front-end; pkg/learn/wrappers.go is a forwarding shim, not a competitor, and is excluded from the authority framing entirely — confirmed by reading its source before writing the claim"
  - "Of three candidate duplicated stages, only QUEEN.md instinct promotion at seal was real, resolved in plan 04 by pkg/memory writing first and the seal's own loop skipping already-promoted IDs while still counting them in the aggregate total; hive promotion at seal was never duplicated (pkg/memory has no hive stage); promotion-target reachability (plan 02) was a prerequisite, not a duplication"
  - "LearningValidator's permanent nil state is recorded as a deliberate choice (wiring it would invert the authority direction Decision 1 establishes), not documented as a bug or left silent"
  - "TestLearningDecisionRecordExists must FAIL (not skip) when the decision record file is missing, even though the plan's own action text describes a general skip-on-not-found pattern — the general pattern was written for the three always-present source docs (CLAUDE.md/AGENTS.md/structural-learning-stack.md) where a missing file signals an unusual cwd, not for this test's own subject file, where a missing file IS the exact regression (record deleted or renamed) the test exists to catch; the plan's acceptance criteria confirms this by requiring a fail on rename, not a skip"
  - "The ADR must not literally contain the retired command names it describes retiring (e.g. hive-opt-in) because the plan's own top-level <verification> section greps the entire .aether/docs/ directory, not just the three corrected files — reworded to describe the deleted commands without repeating their exact names"

patterns-established:
  - "A decision record's Consequences section is a table of {claim, test(s)} pairs, not prose — makes 'is every claim checkable' a mechanical review, not a judgment call"

requirements-completed: [LEARN-03, LEARN-04]

# Metrics
duration: ~50min
completed: 2026-08-04
---

# Phase 162 Plan 06: Switch On Learning — Learning System Authority Decision Record Summary

**Wrote the LEARN-03/LEARN-04 decision record naming `pkg/memory` authoritative and `pkg/learn` explicitly subordinate, corrected three documents' stale "not wired yet" claims (including a pre-existing false phase-end ant-list description), and added a regression test so none of it can rot back the way this exact pipeline's "restored" claim did across v1.10, v1.11, v1.13, and v1.23.**

## Performance

- **Duration:** ~50 min
- **Started:** 2026-08-04 (first commit `20319799`)
- **Completed:** 2026-08-04 (last commit `c6d11764`)
- **Tasks:** 3 (plus 1 deviation fix)
- **Files modified:** 5 (2 created, 3 modified)

## Accomplishments

- `.aether/docs/learning-system-authority.md` is a ~190-line ADR covering all four decisions: authority (`pkg/memory` wins, `pkg/learn`'s Entry/Evidence layer is subordinate, `pkg/learn/wrappers.go` is a shim excluded from the framing entirely), the three candidate duplicated stages (one real — seal-time QUEEN.md promotion, resolved by plan 04's skip-set — two not: hive promotion was never duplicated, promotion-target reachability was a prerequisite plan 02 fixed), the deliberate non-wiring of `LearningValidator`, and the Hive Brain default flip with its full `AETHER_HIVE_POLICY` resolution table. Every runtime claim in its Consequences table names a specific test; all 20 named tests were run and confirmed passing before this SUMMARY was written.
- Corrected the pre-existing false claim in `.aether/docs/structural-learning-stack.md` that phase-end consolidation "executes three ants only: nurse -> herald -> janitor" — confirmed by reading `pkg/memory/consolidate.go` end to end that it contains zero references to `pkg/agent/curation`; the section now describes what `pipeline.RunConsolidation` actually does (decay, archive, promotion-candidate detection, `QueenEligible` computation, no ant calls) and reserves the eight-ant description for the seal path only.
- Removed all three "(Phase 162 wires this)" caveats (`CLAUDE.md:838`, `AGENTS.md:904`, `structural-learning-stack.md:15/214/221`) now that both consolidation subcommands have real runtime callers from plans 03/04, and pointed all three corrected files at the new decision record.
- `cmd/docs_truth_test.go` adds `TestLearningDocsDoNotClaimUnwiredConsolidation` (fails if any of 7 retired substrings reappear in the three corrected files) and `TestLearningDecisionRecordExists` (fails if the decision record is deleted, renamed, gutted to a stub, or loses its `pkg/memory`/`AETHER_HIVE_POLICY` mentions).

## Task Commits

Each task was committed atomically:

1. **Task 1: Write the learning-system authority decision record** — `20319799` (docs)
2. **Task 2: Correct the three documents that describe the learning pipeline** — `3d5656b6` (docs)
3. **Task 3: Add a docs-truth test so the corrected claims cannot rot back** — `0ce6e05f` (test)

**Deviation fix:** `c6d11764` (fix) — reworded the ADR to avoid literally containing a retired command name; see below.

## Files Created/Modified

- `.aether/docs/learning-system-authority.md` — new ADR: Context, Decision 1 (authority), Decision 2 (duplicated stages), Decision 3 (LearningValidator), Decision 4 (Hive Brain default + full resolution table), Consequences (claim -> test table)
- `.aether/docs/structural-learning-stack.md` — Overview wiring-status paragraph, the Consolidation Pipeline section (phase-end and seal subsections rewritten with the corrected ant-list claim), and the Integration Points table all corrected and pointed at the decision record
- `CLAUDE.md` — §Structural Learning Stack lifecycle-integration bullet corrected; §Hive Brain gained a short policy-resolution summary and a pointer to the decision record
- `AGENTS.md` — §Structural Learning Stack lifecycle-integration bullet corrected (only this one line changed; plan 01's hive gating section, verified byte-identical via `git diff` against the wave-3 merge base, is untouched)
- `cmd/docs_truth_test.go` — new test file: `TestLearningDocsDoNotClaimUnwiredConsolidation`, `TestLearningDecisionRecordExists`

## Decisions Made

See `key-decisions` in frontmatter for the four substantive decisions recorded in the ADR itself (authority, duplicated stages, LearningValidator, Hive Brain default), plus two testing/writing decisions made in the course of building the guard test (skip-vs-fail semantics; avoiding literal retired command names inside the ADR's own historical prose).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `TestLearningDecisionRecordExists` skipped instead of failing when the decision record was renamed**
- **Found during:** Task 3's proof-of-linkage verification (the acceptance criteria's explicit "temporarily renaming ... makes the test fail" check)
- **Issue:** The plan's `<action>` text describes a general "skip if not found" pattern intended to prevent spurious failures from unusual working directories or vendored packages. I initially applied that same skip to `TestLearningDecisionRecordExists`'s own subject file. Doing so meant that renaming or deleting `.aether/docs/learning-system-authority.md` — the exact regression this test exists to catch — produced a `SKIP`, not a `FAIL`, silently defeating the test's purpose. The plan's own acceptance criteria contradicts the general action text here: it explicitly requires a fail on rename.
- **Fix:** Changed `TestLearningDecisionRecordExists` to `t.Fatalf` on any read error (including not-found), while leaving `TestLearningDocsDoNotClaimUnwiredConsolidation`'s skip-on-not-found behavior for the three always-present source docs unchanged — that skip protects against a genuinely different failure mode (unusual cwd/vendoring) where the subject files are not the thing being regression-tested.
- **Files modified:** `cmd/docs_truth_test.go`
- **Verification:** Renamed `.aether/docs/learning-system-authority.md` away and confirmed `TestLearningDecisionRecordExists` now reports `--- FAIL`, not `--- SKIP`; restored the file and confirmed `git status --short` showed no diff.
- **Committed in:** `0ce6e05f` (folded into Task 3's commit — found before Task 3 was committed)

**2. [Rule 1 - Bug] The decision record's own historical description tripped the plan's top-level verification grep**
- **Found during:** Running the plan's top-level `<verification>` command (`grep -rn 'Phase 162 wires this\|three ants only\|hive-opt-in\|opt-in, twice over' CLAUDE.md AGENTS.md .aether/docs/`) after Task 3
- **Issue:** That grep scans the entire `.aether/docs/` directory, including the newly-created `learning-system-authority.md`. Decision 4's prose named the retired `hive-opt-in`/`hive-opt-out` cobra commands literally when describing what was deleted, which matched the `hive-opt-in` banned substring — even though the usage was a correct historical description, not a stale claim that the commands still exist.
- **Fix:** Reworded the sentence to describe "the two cobra commands that let an operator record or clear that consent file" without repeating their literal names. No factual content was lost.
- **Files modified:** `.aether/docs/learning-system-authority.md`
- **Verification:** Re-ran the exact plan-level verification grep — zero matches. Re-ran both `cmd/docs_truth_test.go` tests and the full previously-cited 20-test set — all pass.
- **Committed in:** `c6d11764` (separate fix commit, applied after Task 3 was already committed)

---

**Total deviations:** 2 auto-fixed (both Rule 1 — internal contradictions between the plan's general guidance/prose and its own stricter acceptance criteria/verification commands, caught by actually running every verification step rather than assuming the first draft satisfied them).
**Impact on plan:** Neither deviation changed the plan's intended scope or the ADR's factual content; both are corrections to make the plan's own written verification commands pass exactly as specified.

## Issues Encountered

None beyond the two auto-fixed deviations above.

## User Setup Required

None — no external service configuration required.

## Verification Performed

- `.aether/docs/learning-system-authority.md` exists, mentions `pkg/memory` and `AETHER_HIVE_POLICY` (Task 1's automated check).
- `grep -rn 'Phase 162 wires this\|three ants only\|nurse → herald → janitor' CLAUDE.md AGENTS.md .aether/docs/structural-learning-stack.md` returns nothing (Task 2's automated check).
- `grep -rn 'Phase 162 wires this\|three ants only\|hive-opt-in\|opt-in, twice over' CLAUDE.md AGENTS.md .aether/docs/` returns nothing (plan-level verification, broader scope including the new ADR file itself).
- `go test ./cmd/ -run 'TestLearningDocsDoNotClaimUnwiredConsolidation|TestLearningDecisionRecordExists' -count=1 -v` — both pass (Task 3's automated check).
- Proof-of-linkage: temporarily re-adding `Phase 162 wires this` to `CLAUDE.md` makes `TestLearningDocsDoNotClaimUnwiredConsolidation` fail (confirmed, then restored); temporarily renaming `learning-system-authority.md` makes `TestLearningDecisionRecordExists` fail, not skip (confirmed, then restored).
- All 20 tests named in the ADR's Consequences table run individually and pass.
- `go test ./cmd/ -count=1` — full package suite green (`ok`, 272.392s, zero failures).
- `go build ./...` and `go vet ./...` — clean.
- AGENTS.md's plan-01 hive gating section (`### Retrieval is on by default, through one switch` / `### Contradiction and revocation`) confirmed byte-identical via `git diff a9fd2ebdb32288a45bd56966cce8d169b5cf8cc0 HEAD -- AGENTS.md`, which shows only the single Structural Learning Stack lifecycle-integration line changed.
- REQUIREMENTS.md: LEARN-03 and LEARN-04 marked complete via `requirements.mark-complete` (both checkbox and traceability table rows).

## Next Phase Readiness

LEARN-03 and LEARN-04's decision-shaped halves are now both fully satisfied and marked complete — the implementation halves landed in plans 01 (hive control), 02 (promotion reachability), and 04 (seal reconciliation); this plan supplies the written decision record plus the regression guard that keeps the whole pipeline's documentation honest going forward. Phase 162's remaining open requirement is LEARN-05 (plan 05, running in the same wave, not a dependency of this plan and not touched by it).

No blockers for downstream phases. Phase 170's RECLAIM-09 (`queen-seed-from-hive`) can proceed against a settled, documented hive-on-by-default posture.

---
*Phase: 162-switch-on-learning*
*Completed: 2026-08-04*

## Self-Check: PASSED

All created/modified files verified present on disk:
- .aether/docs/learning-system-authority.md
- cmd/docs_truth_test.go
- .aether/docs/structural-learning-stack.md (modified)
- CLAUDE.md (modified)
- AGENTS.md (modified)

All task commits verified present in `git log --oneline --all`:
- 20319799 (Task 1), 3d5656b6 (Task 2), 0ce6e05f (Task 3), c6d11764 (deviation fix)

Full `go build ./...` clean, `go vet ./...` clean, `go test ./cmd/ -count=1` green (272.392s, zero failures).
