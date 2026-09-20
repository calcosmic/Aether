---
phase: 194-the-queen-decides-the-team
reviewed: 2026-08-26T21:33:08Z
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
  critical: 6
  warning: 1
  info: 0
  total: 7
status: issues_found
---

# Phase 194: Code Review Report

**Reviewed:** 2026-08-26T21:33:08Z
**Depth:** standard
**Files Reviewed:** 56
**Status:** issues_found

## Summary

The post-fix implementation is not ready to ship. The normal build/continue CLI options, retained-wrapper reasons, untracked-file detector, and continue-floor table now work on their covered paths, but six blocker-level defects remain around the real dispatch and waiver boundaries and the Codex blocked-flow UI. The reference document also still describes several policies that Phase 194 deleted.

For dummies: the new safety locks exist, but there are still side doors around them. A generic command can approve a protected waiver without its secret token, some valid team choices disappear at the last step, and Codex can tell a blocked user to run commands Codex does not support.

Targeted Phase 194 tests passed (13 tests), and `go vet ./cmd` passed. The repository-wide run reached 6,533 passes with 8 skips, but the `cmd` package exceeded Go's 10-minute package timeout; this timeout is not counted as a finding because the scoped Phase 194 tests completed cleanly and performance is outside this review's scope.

## Narrative Findings (AI reviewer)

### Critical Issues

#### CR-01: The public generic resolver bypasses the owner waiver capability

**Classification:** BLOCKER  
**Severity:** Critical  
**Files:** `cmd/forced_reviewer_waiver.go:481-560`; `cmd/pending_decision.go:101-152,157-214`; `cmd/testdata/command_catalog.json:4959-5010`

**Issue:** `resolveForcedReviewerWaiverPendingDecision` is documented as the only resolver for a forced-reviewer row and correctly validates the active attempt, phase, source, capability hash, and dispatch window. However, the separately registered public commands expose every active row (including its ID) through `pending-decision-list`, then let `pending-decision-resolve --id ... --resolution ...` mark that row resolved without any capability, source, attempt, or dispatch-window check. That generic resolver preserves the authentic row's source, attempt ID, phase, and capability hash, so `forcedReviewerWaiver` accepts it as an owner decline. Any process able to invoke `aether` can therefore bypass the security control before dispatch.

**Fix:** Make protected decision sources unresolvable through the generic command and route them through the capability-aware resolver. Ideally also redact protected implementation fields from the generic list response.

```go
if d.Source == "forced-reviewer-waiver" {
    return fmt.Errorf("forced reviewer decisions must be answered with decision-answer and the displayed capability")
}
```

Add an end-to-end test that renders a card, obtains the row ID through `pending-decision-list`, calls `pending-decision-resolve`, and proves the reviewer remains forced.

#### CR-02: Dispatch proceeds when persistence fails to close the waiver window

**Classification:** BLOCKER  
**Severity:** Critical  
**Files:** `cmd/forced_reviewer_waiver.go:91-121`; `cmd/spawn.go:120-136`

**Issue:** The dispatch-start marker is the security boundary that prevents a worker from using a previously displayed decline command. Both writes in `closeForcedReviewerWaiverWindowForPhase` discard their errors, the function returns no status, and `spawn-log` reports success after calling it. If the window file cannot be saved, dispatch still proceeds while `phaseDispatchStartedAt` reports no start and the pending capability remains usable. This is an explicit fail-open authorization check: a transient disk/lock/permission failure restores the exact post-dispatch waiver attack the marker was added to prevent.

**Fix:** Make the authoritative dispatch-window write return an error and make `spawn-log` fail before the wrapper spawns the worker unless that write is durable. Pending-row deletion can remain defense in depth once the durable marker is guaranteed.

```go
func closeForcedReviewerWaiverWindowForPhase(phaseID int, at time.Time) error {
    // Atomically persist the earliest start marker; return any failure.
}

if err := closeForcedReviewerWaiverWindowForPhase(phase, time.Now().UTC()); err != nil {
    return fmt.Errorf("cannot safely begin dispatch: %w", err)
}
```

Add a fault-injection test whose store rejects the marker write and assert that `spawn-log` fails and no worker may be launched.

#### CR-03: Re-rendering a check-in invalidates the decline command already shown

**Classification:** BLOCKER  
**Severity:** Critical  
**Files:** `cmd/forced_reviewer_waiver.go:362-440`; `cmd/ceremony_team_checkin.go:11-23,137-155`; `.claude/commands/ant-build.md:260-270`

**Issue:** The function promises that a repeated render of the same pending question is an idempotent no-op, but it generates a fresh random capability and replaces the matching unresolved row on every call. The build wrapper invokes the issuing render twice in succession—first visual, then JSON—so the second call immediately invalidates the exact decline command displayed by the first. Any later status/re-render similarly makes a saved command fail without explaining why. This breaks the owner action the check-in card presents.

**Fix:** Issue one capability per attempt/phase/signal and reuse it for every representation of that same card, or introduce a single runtime call that produces both the visual card and machine data from one issuance. Do not rotate a live token merely because the view was rendered again.

```go
// Pseudocode: lookup-or-create must return the same live issuance.
issuance, err := waiverCapabilities.GetOrCreate(activeAttempt.ID, phaseID, signalName)
```

Add a regression test that renders the same card twice and successfully resolves with the command from the first render.

#### CR-04: Valid aliased or comma-separated reviewer proposals are removed at the final continue boundary

**Classification:** BLOCKER  
**Severity:** Critical  
**File:** `cmd/codex_continue.go:1302-1323,1345-1378`

**Issue:** `queenApplyJudgement` deliberately accepts aliases (`security` becomes `gatekeeper`), separator variants, and comma-packed `--castes` values. The final waiver reconciliation does not use that normalization: it lowercases each raw proposal string and compares it directly with the canonical dispatch caste. On a phase whose credentials signal was waived, an explicit proposal such as `--castes security --caste-why security=...` produces a Gatekeeper dispatch in judgement, but `unionForcedContinueReviewers` sees only the raw key `security`, classifies Gatekeeper as waived-signal-only, and deletes the explicitly requested reviewer. `--castes gatekeeper,auditor` has the same problem.

**Fix:** Build the preservation set from the same canonical proposal used by judgement.

```go
normalized, _ := normalizeProposedCastes(proposed)
proposedCastes := stringSet(normalized)
```

Add final-dispatch tests for an alias and a comma-packed proposal after waiving the phase signal.

#### CR-05: Wrapper options can prevent the dispatch window from ever closing

**Classification:** BLOCKER  
**Severity:** Critical  
**Files:** `.claude/commands/ant-build.md:30-38,303-310`; `.claude/commands/ant/build.md:303-310`; `.opencode/commands/ant/build.md:303-310`

**Issue:** All three build wrappers pass raw `$ARGUMENTS` as the value of `spawn-log --phase`. `$ARGUMENTS` is also passed to `aether build`, where valid invocations may contain options such as `1 --verification-depth heavy` or `1 --force`. In the new spawn command those extra tokens are parsed as `spawn-log` options; unsupported ones make the command fail, so the phase marker is never written. If orchestration continues, the reviewer decline window remains open during worker execution. The wrapper already carries the parsed `phase_id`, but does not use it here.

**Fix:** Pass only the trusted numeric phase from the manifest/cross-stage state.

```text
AETHER_OUTPUT_MODE=json aether spawn-log ... --depth 1 --phase <phase_id>
```

Add wrapper-contract coverage using a build invocation with at least one valid additional option and assert the generated `spawn-log` command still contains exactly one numeric phase argument.

#### CR-06: Blocked Codex output still recommends unsupported slash commands

**Classification:** BLOCKER  
**Severity:** Critical  
**Files:** `cmd/codex_continue.go:3405-3450,3475-3480,3533-3555`; `cmd/codex_visuals.go:435-458,569-582,2072-2102`

**Issue:** The visual-test fix changed selected expectations to native `aether ...` names, but real gate recovery strings still contain `/ant-continue`, `/ant-unblock`, and `/ant-flags`. `writeVisualOutput` only translates native `aether` names into slash wrappers for Claude/OpenCode; on Codex it deliberately returns the text unchanged. Consequently the most important UI state—the way forward after a blocked continue—tells Codex users to run commands that do not exist on Codex. The current blocked visual test checks the headline and one native retry sentence but never rejects the stale slash commands also present in the output.

**Fix:** Store recovery guidance in canonical native CLI form (`aether continue`, `aether unblock --dispatch`, `aether flag-resolve ...`) and let the existing exit translator convert it for wrapper platforms. Change the hardcoded fallback in `renderBlockedWayForward` the same way.

```go
RecoveryOptions: []string{
    "Fix manually and run aether continue",
    "Run aether unblock --dispatch for guided recovery",
}
```

Add a Codex-platform assertion that blocked visual output contains no `/ant-` substring, plus Claude/OpenCode assertions that canonical native hints still translate to their wrappers.

### Warnings

#### WR-01: The source-of-truth relevance reference still documents deleted policies

**Classification:** WARNING  
**Severity:** Warning  
**Files:** `.aether/docs/command-playbooks/caste-relevance-reference.md:17-18,25,36-43,55-62`; `cmd/caste_relevance.go:45-57,170-257,418-463`

**Issue:** The corrected continue-floor table now matches code, but nearby authoritative text is stale in several material ways. It says Auditor is production-only even though that condition was removed; lists `documentation` instead of the actual Chronicler keyword `document`; says Gatekeeper/high-risk and Auditor/production are still auto-included even though those branches were deleted; says build/continue threshold scoring selects workers and the Queen filters later even though both flows now use the gated required-only fallback; and says Watcher/Probe can still appear at build through relevance scoring, which the build gate prevents. Maintainers following this document will implement or debug the wrong dispatch policy.

**Fix:** Update the registry rows and scoring prose from the live table, explicitly state that build/continue scores are refusal diagnostics rather than no-proposal selection, and say Watcher/Probe require an explicit proposal at build. Extend the doc test beyond the floor table so it detects deleted conditions/special rules and the gated-flow semantics.

---

_Reviewed: 2026-08-26T21:33:08Z_  
_Reviewer: the agent (gsd-code-reviewer)_  
_Depth: standard_
