---
phase: 194-the-queen-decides-the-team
reviewed: 2026-08-23T00:00:00Z
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

# Phase 194: Code Review Report (Iteration 2)

**Reviewed:** 2026-08-23T00:00:00Z
**Depth:** standard
**Files Reviewed:** 58
**Status:** issues_found

## Summary

This is the iteration-2 re-review after the fixer landed five commits
(`0836296c`, `ab940c24`, `7bef82d2`, `cb6f2a10`, `4cd7333d`) on top of
`06693a34` in response to `194-REVIEW.iter2.md`'s CR-01, WR-01, WR-02, WR-03,
IN-01, and IN-03. I re-read every changed file in the fix diff
(`git diff 06693a34..HEAD`), ran the full targeted test set for this area
(all pass), and confirmed the build is clean (`go build ./...`, no errors).

**All six named findings from iteration 1 are genuinely fixed, correctly
tested, and I found no regression in the surrounding code:**

- **WR-02 / WR-03 (path-pattern boundary + bare "auth" pattern):**
  `matchesPathPatternAtBoundary` replaces the old plain `strings.Contains`
  with the same left/right boundary discipline `matchesPhraseAtWordBoundary`
  already applies to prose, and `"auth/"` was replaced with a bare `"auth"`
  matched the same boundary-safe way as `"session"`/`"credential"`/
  `"secrets"`. `TestPathPatternMatchingRequiresAWordBoundary` proves the
  exact false positive named in iteration 1
  (`repossession_handler.go` no longer matches `"session"`) while a genuine
  `session.go` still forces the reviewer; `TestBareAuthPathPatternCatchesAuthNamedFiles`
  table-tests `auth.go`, `auth_service.go`, `auth-config.yaml`, and
  `auth/handler.go` all now hit, while `authorization.go` correctly does not
  (a fused identifier with no boundary, the same accepted trade-off the
  phrase table already documents for "token" vs "tokenizer").
- **WR-01 (file detector trusted only self-reported changed files):**
  `phaseChangedFilesForRiskSignals` now unions
  `phaseChangedFilesFromHandoffs` with an independent `git diff --name-only
  ... HEAD` (`discoverChangedFilesFromGit`, the same call `codex_build_finalize.go`
  already uses), and both continue lanes
  (`plannedContinueReviewDispatches`, `plannedExternalContinueDispatches`)
  were switched to it identically, preserving the two-lane parity discipline.
  `TestGitDiffCatchesAFileTheHandoffOmitted` proves a git-tracked file no
  handoff ever mentions still forces the reviewer.
- **IN-01 (untrimmed `pattern` vs matched `p` in the longest-match compare):**
  fixed in the same edit as WR-02 — `best = p` now, consistent with the
  phrase-matching function.
- **IN-03 (raw path pattern with trailing slash shown to the owner):**
  `forcedReviewerReasonPathLabel` trims the trailing `/` before the sentence
  reaches the owner-facing reason; `TestForcedReviewerReasonNeverShowsATrailingSlash`
  and the updated assertion in `TestChangedFilesCanOnlyAddAForcedReviewer`
  both check the rendered clause, not just the underlying pattern.
- **CR-01, forgery half (the original vulnerability: anyone able to invoke
  `aether decision-answer` could compute the deterministic waiver question
  from public information alone and waive a reviewer with no owner ever
  having seen it):** this is genuinely closed. The check-in card's render
  step now writes a pending, *unresolved* decision row the moment (and only
  the moment) it shows a live forced reviewer
  (`ensureForcedReviewerWaiverPendingDecision`), and `decision-answer` now
  refuses to record a forced-reviewer-shaped answer unless it resolves that
  exact runtime-created row (`resolveForcedReviewerWaiverPendingDecision`) —
  it can no longer manufacture a fresh resolved entry for this class of
  question the way the general clarification path still can for everything
  else. `TestDecisionAnswerCannotForgeAForcedReviewerWaiver` goes through the
  real CLI command (not just the Go function) and proves a `--question` with
  no prior card render leaves the reviewer in the dispatch list.

**One Critical finding remains — a residual of CR-01 that the fix narrows
but does not close, and that iteration 1 did not test for.** The pending row
the card creates has no expiry and nothing closes it when the build actually
proceeds. See CR-01 below (renumbered from the prior iteration since the
original vulnerability it named is fixed; this is the follow-on gap the fix
introduced/left open).

I did not re-raise the "no marker distinguishes a worker's Bash tool from
the orchestrating session's Bash tool" limitation on its own — the fixer's
documented reasoning (no reliable platform-supplied identity for an
arbitrary CLI call; a self-set env var is decorative) is sound and I found
no concrete, enforceable mechanism in this codebase or platform that would
close it. It is, however, the exact mechanism that makes the CR-01 residual
below exploitable, so it is cited as supporting evidence there rather than
raised again as an independent finding.

## Critical Issues

### CR-01: The runtime-created waiver row stays open and resolvable for the whole build, so a decline the owner never gave can still be recorded

**File:** `cmd/ceremony_team_checkin.go:150-152`, `cmd/forced_reviewer_waiver.go:177-208`, `cmd/queen_risk_signals.go:488-505`

**Issue:** The fix correctly requires a waiver to resolve a row the runtime
itself created (`ensureForcedReviewerWaiverPendingDecision`). But that row is
written the instant the check-in card renders a live forced reviewer —
*before* the owner has answered "Proceed with this team" — and nothing in
the build, worker-spawn, or continue path ever closes, expires, or
time-boxes it. Per `.claude/commands/ant/build.md`'s own Team Check-In
stage, "Proceed with this team" is the recommended, default answer and
involves no call to `decision-answer` at all; the pending row is left
exactly as open after the owner proceeds as it was before they answered.

`resolveForcedReviewerWaiverPendingDecision` (`forced_reviewer_waiver.go:230-266`)
matches on question text and phase alone — it has no concept of "before
workers were dispatched" — and `queenForcedContinueReviewers`
(`queen_risk_signals.go:488`), the one function both continue lanes go
through, calls `applyForcedReviewerWaivers` → `forcedReviewerWaiver` at
**continue time**, i.e. after the build's workers have already run. So the
live attack window is exactly the build itself: once the card has rendered
(which happens for any phase this feature is meant to protect — logins,
payments, deletions, migrations, release sign-off), anything able to invoke
the `aether` binary during that build — a Builder's own Bash tool, a
misfiring script, a prompt-injected instruction in phase content — can run
the exact command the card legitimately displayed:

```
aether decision-answer --question "Phase 7: a security reviewer is being added because this touches logins and passwords. Waive it?" --answer "auto" --phase 7
```

and this now *succeeds* (the row exists, was runtime-created, and is still
unresolved), silently dropping the reviewer at the continue step that runs
after the build — even though the owner's actual answer was "proceed",
never "decline". I confirmed this with an executable proof-of-concept
(temporary test, not committed): rendering the check-in card, then calling
`decision-answer` with the card's exact question text as a distinct step
*after* the render (simulating a mid-build call, with no simulated owner
decline), still waives the reviewer at the subsequent
`queenForcedContinueReviewers` check — going from 1 forced reviewer to 0.

This is a narrower window than the original CR-01 (the forger must wait for
a live card render first, so it can't work on a phase that never triggers a
signal), but it covers exactly the phases the feature exists to protect, and
it defeats the owner's explicit "keep" decision, not just an unanswered
one — which iteration 1's tests did not check, since they only tested "no
answer was ever given" (`TestOnlyTheOwnerCanWaiveAForcedReviewer`,
`TestAutopilotNeverWaives`) and "a forged question with no prior render"
(`TestDecisionAnswerCannotForgeAForcedReviewerWaiver`), not "a genuinely
rendered row, answered by something other than the owner, after the owner
was shown the card and chose to keep the reviewer."

**Fix:** Close or time-box the row at the point the build actually proceeds,
not just at the point the card renders. Concretely, either:
- Have `build-finalize` (or the point where `Worker Spawning` begins, once
  the owner's check-in answer is known) resolve any still-pending
  forced-reviewer row for that phase as "kept" (not waived) before workers
  are dispatched, so a later `decision-answer` call with the same question
  text finds no pending row to match — the same refusal path CR-01 already
  built for the no-row case; or
- Have `queenForcedContinueReviewers` (or `forcedReviewerWaiver`) refuse a
  resolution whose `ResolvedAt` falls after the build's own recorded
  dispatch timestamp for that phase (a timestamp already available on the
  manifest/build-attempt record), so a waiver answered during or after
  worker execution is never honored, only one answered at the check-in
  pause itself.
Either way, add a test that renders the card, does **not** call
`decision-answer` at all (the "proceed" path), then calls
`decision-answer` with the card's own question text as a separate step
after that — and asserts the reviewer is still forced, because the owner's
answer was "proceed", not "decline".

## Info

### IN-01: `TestCasteRelevanceDoc_SpawnBudgetNumbersMatch`'s doc-consistency check is a loose numeric substring match (unchanged, pre-existing, out of scope)

**File:** `cmd/caste_relevance_doc_test.go:204-213`

**Issue:** Carried forward from iteration 1 (there numbered IN-02) —
unchanged by this fix round and explicitly marked as not requiring action in
this phase. `strings.Contains(content, strconv.Itoa(got))` only proves the
digit sequence appears somewhere in the doc, not that it appears as the
documented number for that specific flow/depth row, so this half of the test
would not fail if the doc's claim for a given flow drifted from the code as
long as the digit appeared anywhere else on the page.

**Fix (optional, future work):** Anchor the check to the specific table
row/line for that flow, or drop the doc-content half in favor of the
pre-existing `docstest`-style structured extraction other doc tests in this
repo already use.

---

_Reviewed: 2026-08-23T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
