---
phase: 194-the-queen-decides-the-team
reviewed: 2026-08-23T18:42:21Z
depth: standard
files_reviewed: 58
files_reviewed_list:
  - .aether/docs/command-playbooks/caste-relevance-reference.md
  - .aether/docs/retired-tests-ledger.md
  - .aether/schemas/completion-packet.schema.json
  - .claude/commands/ant-build.md
  - .claude/commands/ant-continue.md
  - .claude/commands/ant/build.md
  - .claude/commands/ant/continue.md
  - .gitignore
  - .opencode/commands/ant/build.md
  - .opencode/commands/ant/continue.md
  - CLAUDE.md
  - cmd/boundary_double_dispatch_test.go
  - cmd/build_attempt_external_test.go
  - cmd/caste_relevance_doc_test.go
  - cmd/caste_relevance_test.go
  - cmd/caste_relevance.go
  - cmd/ceremony_cmd_test.go
  - cmd/ceremony_team_checkin_test.go
  - cmd/ceremony_team_checkin.go
  - cmd/claudemd_verification_depth_test.go
  - cmd/codex_build_finalize_test.go
  - cmd/codex_build_test.go
  - cmd/codex_build.go
  - cmd/codex_continue_plan_test.go
  - cmd/codex_continue_plan.go
  - cmd/codex_continue_test.go
  - cmd/codex_continue.go
  - cmd/codex_dispatch_contract.go
  - cmd/codex_visuals_test.go
  - cmd/codex_workflow_cmds.go
  - cmd/continue_depth_spawn_test.go
  - cmd/continue_fastpath_castes_test.go
  - cmd/floor_unskippable_test.go
  - cmd/forced_reviewer_waiver_test.go
  - cmd/forced_reviewer_waiver.go
  - cmd/golden_workflow_test.go
  - cmd/handoff_decisions_cmd.go
  - cmd/one_task_bug_fix_test.go
  - cmd/owner_dials_test.go
  - cmd/phase_verified_once_test.go
  - cmd/queen_fallback_team_test.go
  - cmd/queen_forced_reviewer_test.go
  - cmd/queen_judgement_test.go
  - cmd/queen_judgement.go
  - cmd/queen_orchestration_regression_test.go
  - cmd/queen_probe_gating_test.go
  - cmd/queen_relevance_floor_test.go
  - cmd/queen_risk_signals.go
  - cmd/queen_spawn_budget.go
  - cmd/queen_team_choice_test.go
  - cmd/queen_worker_reason_test.go
  - cmd/reclaim_wiring_test.go
  - cmd/review_depth_test.go
  - cmd/review_depth.go
  - cmd/spawn.go
  - cmd/spawn_budget_test.go
  - cmd/testdata/command_catalog.json
  - cmd/testdata/golden_build.txt
  - cmd/testdata/golden_continue.txt
  - cmd/testdata/golden_plan.txt
findings:
  critical: 1
  warning: 0
  info: 1
  total: 2
status: issues_found
---

# Phase 194: Code Review Report

**Reviewed:** 2026-08-23T18:42:21Z
**Depth:** standard
**Files Reviewed:** 58
**Status:** issues_found

## Summary

This is iteration 3 (final) of the review-fix loop for phase 194. It focuses
on `git diff 4cd7333d..HEAD` (`cmd/spawn.go`, `cmd/forced_reviewer_waiver.go`,
`cmd/forced_reviewer_waiver_test.go`, and the three wrapper mirrors), the
commit `bce085ce` fix for iteration 2's residual CR-01: the runtime-created
"waive it?" row staying open and resolvable for the whole build.

**What the fix gets right, verified directly:** I traced the new
`closeForcedReviewerWaiverWindowForPhase` call chain end to end and reran the
three new tests plus the two carried-forward forgery tests
(`go test ./cmd/... -run 'TestForcedReviewerDeclineWindowClosesWhenDispatchBegins|TestOwnerDeclineBeforeDispatchStaysHonoredOnceDispatchBegins|TestSpawnLogWithNoCardRenderedIsANoOpAndReviewerStaysForced|TestDecisionAnswerCannotForgeAForcedReviewerWaiver|TestDecisionAnswerResolvesARuntimeCreatedWaiverRow'`) — all pass. I confirmed in `.claude/commands/ant/build.md` (and its two byte-identical mirrors, `.opencode/commands/ant/build.md` and `.claude/commands/ant-build.md`) that `spawn-log --phase $ARGUMENTS` is step 2 of the per-worker loop and the platform Task-tool spawn is step 3 — the window genuinely closes before that worker's own shell could run, for every wave, not just the first. I confirmed the TS host lane (`.aether/ts-host/src/worker-dispatch.ts`) never renders the check-in card, so its not carrying `--phase` is inert, not a gap, as disclosed. `--no-checkin`/autopilot: no card, no pending row, `spawn-log --phase` is a proven no-op, reviewer stays forced. The owner's real decline-before-dispatch path: proven to still resolve and stay honored after dispatch begins. The `ResolvedAt` (second-precision `RFC3339`) vs. dispatch-start (`RFC3339Nano`) time comparison in `forcedReviewerWaiver` is sound — truncation only ever makes a genuine prior resolution look *earlier*, never later, so it cannot flip a legitimate decline into a rejection.

**What's still broken, found by tracing a scenario the fix report's own proof list doesn't cover:** rebuilding the *same phase number* a second time. I wrote a throwaway test reproducing exactly this (run, confirmed it fails against the current code, then deleted it per instructions — it is not part of the repo) and it confirms a genuine regression in the owner's decline path for the single most common operational pattern in this codebase's own stated methodology: build, fail or get reviewed, rebuild the same phase. See CR-01 below.

The one deliberately-carried Info item (loose numeric doc-consistency assertion) is unchanged from iteration 2 and stays Info per prior instruction not to re-raise its severity.

## Critical Issues

### CR-01: A phase's forced-reviewer decline window never reopens on a retried build — the owner's genuine decline is silently refused

**File:** `cmd/forced_reviewer_waiver.go:80-107` (`closeForcedReviewerWaiverWindowForPhase`), `cmd/forced_reviewer_waiver.go:376-410` (`resolveForcedReviewerWaiverPendingDecision`), `cmd/build_attempt.go:185` (`beginBuildAttempt`, the natural reset point that doesn't exist)

**Issue:** `phase-dispatch-started.json` (`phaseDispatchWindowFileName`) records, per phase number, the *first ever* moment any worker was dispatched for that phase — and nothing in this diff, nor anywhere else in `cmd/`, ever deletes or resets an entry in it. `beginBuildAttempt` (`cmd/build_attempt.go`), which already runs once per build attempt for a given phase number (including retries) and writes a fresh `attempt-<timestamp>-<pid>.json` record, does not touch this file.

Consequence: once a phase has had even one worker spawned in *any* prior attempt — the phase fails partway through, gets rebuilt after `/ant-continue` finds problems, or (as in this exact review-fix loop) goes through a second build-review-fix cycle — `phaseDispatchStartedAt(phaseID)` returns `true` forever after, for that phase number, regardless of how many fresh, unrelated build attempts follow. `resolveForcedReviewerWaiverPendingDecision` (`cmd/forced_reviewer_waiver.go:394-403`) checks only `phaseDispatchStartedAt(phaseID)` — not whether *this* attempt's dispatch has begun — and refuses immediately:

```go
if phaseID > 0 {
    if _, started := phaseDispatchStartedAt(phaseID); started {
        return PendingDecision{}, false, nil
    }
}
```

So on a genuine retry: the check-in card renders again (correctly, since `ensureForcedReviewerWaiverPendingDecision` writes a fresh pending row because the prior attempt's row was already deleted), the card legitimately prints "To decline, run: ...", the owner runs that exact command *before any worker of the new attempt has been dispatched* — and it is refused with: *"Nothing was recorded. Phase N is not currently waiting on an answer for that reviewer -- the program only accepts a decline for a reviewer it has actually shown on the check-in card."* That message is false in this case: the card just showed it, in this very attempt. The owner has no way to tell this apart from a real forgery attempt, and no way to actually decline the reviewer for the retried build short of manually deleting `phase-dispatch-started.json`.

I confirmed this concretely with a throwaway test (not committed, deleted after running): render the check-in card for phase 1, run `spawn-log --phase 1` once (simulating attempt 1 dispatching and failing), then render the card again for phase 1 (attempt 2) and immediately run `decision-answer` with the card's own decline command, with **no** attempt-2 `spawn-log` call yet. Expected: waived. Actual: `forcedReviewerWaiver(1, ...)` returns `false` — the decline silently did nothing, and the test's failure message read: `BUG CONFIRMED: owner's decline in attempt 2, made before any attempt-2 worker dispatched, was refused because attempt 1's stale dispatch-start record never expired`.

This is a correctness regression in the exact feature this iteration's fix was meant to complete: the decline window now closes correctly *within* one attempt, but it also — as an unintended side effect — never reopens for any later attempt of the same phase. Given this repo's own build→continue→(review→fix)→rebuild loop is the normal path, not an edge case, this will reproduce routinely, not rarely.

**Fix:** Reset (delete) `phase-dispatch-started.json`'s entry for `phaseNum` at the start of each new build attempt, not just once ever per phase. `beginBuildAttempt` (`cmd/build_attempt.go:185`) is the natural site — it already runs exactly once per attempt, before the check-in card renders, and already receives `phaseNum`:

```go
// at the top of beginBuildAttempt, before writing the new attempt record
clearPhaseDispatchWindow(phaseNum) // new: delete windowFile.Phases[strconv.Itoa(phaseNum)]
```

Add a small `clearPhaseDispatchWindow(phaseID int)` alongside `closeForcedReviewerWaiverWindowForPhase` in `cmd/forced_reviewer_waiver.go` that loads `phase-dispatch-started.json`, deletes the key for `phaseID` if present, and saves. This keeps the within-attempt guarantee (the window still closes the moment the *current* attempt's first worker spawns) while letting a genuinely new attempt's check-in card and decline command work again. Add a test mirroring the throwaway one above (render card for phase N, `spawn-log --phase N`, render card again for phase N simulating a retry, `decision-answer` before any new `spawn-log` call, assert waived) as the permanent regression proof — this is exactly the shape of test this repo's Definition of Done requires (a command that fails when the requirement is unmet), and the one gap in the current test list (`TestForcedReviewerDeclineWindowClosesWhenDispatchBegins`, `TestOwnerDeclineBeforeDispatchStaysHonoredOnceDispatchBegins`, `TestSpawnLogWithNoCardRenderedIsANoOpAndReviewerStaysForced`) is that none of them exercise a second attempt of the same phase number.

## Info

### IN-01: `TestCasteRelevanceDoc_SpawnBudgetNumbersMatch`'s doc-consistency check is a loose numeric substring match

**File:** `cmd/caste_relevance_doc_test.go:204-213`
**Issue:** `strings.Contains(content, strconv.Itoa(got))` only proves the digit sequence appears somewhere in the doc, not that it appears as the documented number for that specific flow/depth row — unchanged since iteration 1 (there numbered IN-02) and iteration 2, both of which left it as pre-existing/out of scope for this phase.
**Fix:** Anchor the check to the specific table row/line (or switch to the repo's existing `docstest`-style structured extraction) rather than a bare substring search. Carried forward as Info per explicit instruction not to re-raise it above that severity; still not in scope for this phase's own deliverable.

---

_Reviewed: 2026-08-23T18:42:21Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
