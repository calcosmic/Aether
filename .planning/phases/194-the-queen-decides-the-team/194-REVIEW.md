---
phase: 194-the-queen-decides-the-team
reviewed: 2026-08-26T19:55:08Z
depth: standard
files_reviewed: 56
files_reviewed_list:
  - .aether/docs/command-playbooks/caste-relevance-reference.md
  - .aether/docs/retired-tests-ledger.md
  - .aether/schemas/completion-packet.schema.json
  - .claude/commands/ant-build.md
  - .claude/commands/ant-continue.md
  - .claude/commands/ant/build.md
  - .claude/commands/ant/continue.md
  - .opencode/commands/ant/build.md
  - .opencode/commands/ant/continue.md
  - cmd/boundary_double_dispatch_test.go
  - cmd/build_attempt_external_test.go
  - cmd/caste_relevance.go
  - cmd/caste_relevance_doc_test.go
  - cmd/caste_relevance_test.go
  - cmd/ceremony_cmd_test.go
  - cmd/ceremony_team_checkin.go
  - cmd/ceremony_team_checkin_test.go
  - cmd/claudemd_verification_depth_test.go
  - cmd/codex_build.go
  - cmd/codex_build_finalize_test.go
  - cmd/codex_build_test.go
  - cmd/codex_continue.go
  - cmd/codex_continue_plan.go
  - cmd/codex_continue_plan_test.go
  - cmd/codex_continue_test.go
  - cmd/codex_dispatch_contract.go
  - cmd/codex_visuals_test.go
  - cmd/codex_workflow_cmds.go
  - cmd/continue_depth_spawn_test.go
  - cmd/continue_fastpath_castes_test.go
  - cmd/floor_unskippable_test.go
  - cmd/forced_reviewer_waiver.go
  - cmd/forced_reviewer_waiver_test.go
  - cmd/golden_workflow_test.go
  - cmd/one_task_bug_fix_test.go
  - cmd/owner_dials_test.go
  - cmd/phase_verified_once_test.go
  - cmd/queen_fallback_team_test.go
  - cmd/queen_forced_reviewer_test.go
  - cmd/queen_judgement.go
  - cmd/queen_judgement_test.go
  - cmd/queen_orchestration_regression_test.go
  - cmd/queen_probe_gating_test.go
  - cmd/queen_relevance_floor_test.go
  - cmd/queen_risk_signals.go
  - cmd/queen_spawn_budget.go
  - cmd/queen_team_choice_test.go
  - cmd/queen_worker_reason_test.go
  - cmd/reclaim_wiring_test.go
  - cmd/review_depth.go
  - cmd/review_depth_test.go
  - cmd/spawn_budget_test.go
  - cmd/testdata/command_catalog.json
  - cmd/testdata/golden_build.txt
  - cmd/testdata/golden_continue.txt
  - cmd/testdata/golden_plan.txt
findings:
  critical: 5
  warning: 2
  info: 0
  total: 7
status: issues_found
---

# Phase 194: Code Review Report

**Reviewed:** 2026-08-26T19:55:08Z
**Depth:** standard
**Files Reviewed:** 56
**Status:** issues_found

## Summary

The retry fix in `bff1ce8a` closes the previously reported stale-window defect on the intended interactive retry path: a fresh plan-only build now clears the phase's old dispatch marker before starting the new attempt, while an already resolved phase/signal waiver remains durable. That specific Critical is closed.

The current implementation is still not shippable. Five independently traceable correctness/security defects remain: a recorded waiver does not actually remove a plan-wording reviewer from the real continue team; normal CLI execution discards the Queen's explicit team flags; the wrapper's trim flow loses the reasons required to retain the selected workers; the retry reset can be driven by a worker to forge an owner-only decision; and the supposedly independent changed-file detector misses untracked sensitive files. The authoritative relevance reference and eleven Codex visual assertions are also stale.

For dummies: the retry button now resets the old lock correctly, but several wires after that button are connected to the wrong places. The UI can say a reviewer was declined or a worker was kept while the runtime does the opposite, and a newly created sensitive file can still slip past the safety check.

## Narrative Findings (AI reviewer)

## Critical Issues

### CR-01 [BLOCKER]: A genuine owner waiver does not remove a plan-wording reviewer from the dispatched continue team

**Files:** `cmd/queen_spawn_budget.go:121-140`; `cmd/caste_relevance.go:218-257`; `cmd/codex_continue.go:1257-1278,1300-1321`; `cmd/queen_risk_signals.go:488-508`

**Issue:** The waiver is applied only in `queenForcedContinueReviewers`, but the no-proposal continue path has already inserted every unwaived *phase-wording* reviewer through `queenRequiredCastesForBudget -> queenForcedReviewersForPhase -> queenFallbackTeam`. `unionForcedContinueReviewers` is append/overwrite-only: when the waiver-filtered reviewer list is empty, it returns those existing dispatches unchanged. Therefore the check-in card can promise that Gatekeeper or Auditor “will NOT run,” persist a valid owner waiver, and still dispatch that caste during continue. The current waiver tests stop at the filtering helper and do not assert the final dispatch list.

**Fix:** Make the waiver-filtered reviewer set authoritative at the dispatch boundary. One safe shape is to remove named-risk reviewers from `queenRequiredCastesForBudget` and add them only through `unionForcedContinueReviewers`; alternatively reconcile signal-derived entries before returning rather than only unioning them. Preserve a reviewer independently required by heavy depth or explicitly proposed by the Queen.

```go
// Derive once, after waivers, at the actual continue boundary.
reviewers := queenForcedContinueReviewers(phase, forced, changedFiles)
dispatches = reconcileForcedReviewers(dispatches, reviewers, phase, reviewDepth)
```

Add an integration test that records a real waiver and asserts both continue lanes' final dispatches omit only that signal's reviewer, including the case where a second live signal still requires the same caste.

### CR-02 [BLOCKER]: Normal `build` and `continue` silently discard `--castes`, `--caste-reason`, and `--caste-why`

**Files:** `cmd/codex_workflow_cmds.go:126-138,158-167,184-192,237-261,271-280`; `cmd/codex_continue.go:143-184,243-290`

**Issue:** Both commands parse all three Queen-team flags and forward them in the `--plan-only` branch, but omit them from the options passed to the normal execution branch. This breaks the explicit owner/Queen control surface on the direct Codex lifecycle path and contradicts the Phase 194 requirement that `--castes` keep working on build and continue, including the fast path. The continue report's option snapshot and loop comparison also omit these fields, so even after forwarding is restored, a changed team proposal can be mistaken for a repeat of the previous parameters.

**Fix:** Forward the parsed fields in both non-plan-only option literals, persist them in `codexContinueOptionsJSON`, and compare normalized values in `continueOptionsMatchCurrent`.

```go
QueenCastes:      queenCastes,       // or continueCastes
QueenCasteReason: queenCasteReason,  // or continueCasteReason
QueenCasteWhy:    queenCasteWhy,     // or continueCasteWhy
```

Add command-level tests that execute each normal Cobra path with these flags and inspect the resulting manifest/dispatches; calling `plannedContinueReviewDispatches` directly does not exercise this wiring.

### CR-03 [BLOCKER]: The wrapper's “Trim optional workers” action silently refuses every kept optional worker

**Files:** `.claude/commands/ant/build.md:181-210,266-273`; `.claude/commands/ant-build.md:181-210,266-273`; `.opencode/commands/ant/build.md:181-210,266-273`; `cmd/queen_judgement.go:202-213`

**Issue:** The wrappers correctly state that every optional proposal needs its own `--caste-why` and that the team-level `--caste-reason` never satisfies that requirement. The trim recipe then re-fetches the manifest with only `--castes <kept optional castes>` and a team reason. Runtime judgement consequently classifies every kept optional caste as `refusedNoReason`. An owner choosing to keep selected workers gets the required floor only, the opposite of the recorded choice. All three build wrappers contain the same recipe.

**Fix:** Replay each retained caste's original reason from `result.reasons` as a separate `--caste-why caste=reason` argument on trim, and preserve the same proposal/reasons when re-fetching after a reviewer decline.

```text
aether build ... --plan-only --castes <kept> \
  --caste-why <caste1>="<result.reasons[caste1]>" \
  --caste-why <caste2>="<result.reasons[caste2]>" \
  --caste-reason "owner check-in trim"
```

Add wrapper contract tests that trace a multi-worker trim through `queenApplyJudgement` and assert the retained optionals remain in the new manifest.

### CR-04 [BLOCKER]: A worker can reopen the decline window and manufacture the owner-only waiver state

**Files:** `cmd/codex_build.go:466-474`; `cmd/forced_reviewer_waiver.go:119-177,365-390,455-488`; `cmd/ceremony_team_checkin.go:137-152`; `cmd/codex_workflow_cmds.go:126-167`

**Issue:** The new reset is keyed only by phase and is reached by the public `aether build <phase> --plan-only --force` path. A running worker with Bash access can force a fresh plan-only attempt, render the public `ceremony team-checkin` command to create a pending row, and submit the deterministic waiver answer before any new `spawn-log` closes the phase marker. There is no owner capability, nonce, or attempt binding in that chain. In addition, the resolver accepts any unresolved same-phase row with matching text without checking `Source == "forced-reviewer-waiver"`, so a look-alike pending question is treated as runtime-authentic. This restores the forgery class the dispatch window was intended to prevent; it merely requires starting a forced retry first.

**Fix:** Bind a decline to an immutable build-attempt ID and an unguessable, single-use capability emitted only in the owner-facing card. Validate the attempt, capability, phase, signal, and `Source` atomically when resolving it. Superseding an attempt must invalidate its capability, and unauthenticated plan-only/card-render commands must not create a live owner approval channel while worker execution is active.

```go
if d.Source != "forced-reviewer-waiver" || d.AttemptID != activeAttempt.ID ||
   subtle.ConstantTimeCompare([]byte(d.Nonce), []byte(answerNonce)) != 1 {
    continue
}
```

Add an adversarial test that begins dispatch, invokes the force/replan/card-render sequence as a worker, and proves `decision-answer` cannot create a waiver without the owner capability.

### CR-05 [BLOCKER]: The changed-file safety net misses brand-new untracked sensitive files

**Files:** `cmd/queen_risk_signals.go:511-549`; `cmd/queen_forced_reviewer_test.go:503-540`

**Issue:** `phaseChangedFilesForRiskSignals` claims its Git union prevents an incomplete or dishonest handoff from defeating the reviewer safety net, but it delegates to `discoverChangedFilesFromGit`, whose `git diff ... HEAD` queries do not report untracked files. The regression test stages its new migration with `git add` before checking it, masking the common case. A worker can create `migrations/0099.sql`, omit it from the handoff, and leave it untracked; neither input sees it, so Auditor is not forced. The same bypass applies to other named sensitive-path signals.

**Fix:** Union ignored-aware untracked files into the independent detector, for example with `git ls-files --others --exclude-standard`, then normalize/deduplicate them with created and modified paths.

```go
out, err := exec.Command("git", "ls-files", "--others", "--exclude-standard").Output()
if err == nil {
    created = append(created, parseGitNameOutput(out)...)
}
```

Add a regression test that deliberately leaves the omitted sensitive file untracked and asserts the appropriate forced reviewer is still selected.

## Warnings

### WR-01 [WARNING]: The authoritative caste reference documents obsolete continue floors

**Files:** `.aether/docs/command-playbooks/caste-relevance-reference.md:50-69`; `cmd/caste_relevance.go:526-556`

**Issue:** The reference still says light requires Watcher, standard requires Watcher plus Probe, and heavy requires Watcher plus the full panel. Current code requires no unconditional caste at light/standard and requires Gatekeeper, Auditor, and a conditional Probe at heavy; Watcher is not a floor. This file labels the table as sourced from `isAlwaysRequired`, so operators and future maintainers receive the inverse of the shipped policy.

**Fix:** Update the table to match the current policy and strengthen the documentation test to compare exact caste names per depth rather than checking only loose numeric fragments.

### WR-02 [WARNING]: Codex visual tests deterministically assert retired slash-command guidance

**File:** `cmd/codex_visuals_test.go:291,379,438,560,870,932,1028,1151,1219,1356,1491`

**Issue:** Eleven scoped visual tests require legacy `/ant-*` strings even though the Codex runtime correctly renders native `aether ...` lifecycle commands under the repository's platform policy. The failures reproduce when these tests are isolated, so they are not suite-order contamination. As a result, `go test ./...` fails despite the corresponding production output following the supported Codex interface; this makes the regression gate unreliable and can hide a genuine visual regression among known-red assertions.

**Fix:** Replace each legacy expectation with the native Codex command currently required by `AGENTS.md` (for example `aether build 1`, `aether continue`, and `aether seal`). If any case intentionally tests a Claude/OpenCode renderer, explicitly select that platform in its fixture instead of using the Codex visual path. Keep the expected command surface internally consistent within each test.

---

_Reviewed: 2026-08-26T19:55:08Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
