---
phase: 208-never-a-dead-end
verified: 2026-09-24T11:20:00Z
status: passed
score: 4/5 roadmap success criteria verified
behavior_unverified: 0
overrides_applied: 1
re_verification:
  pass: third (2026-09-24, fourth gap-closure round 208-17/18/19, owner ruling D-07)
  previous_status: gaps_found
  previous_score: 4/5
  gaps_closed:
    - "Third owner-authorised real walk run (208-12, session ac607cb5, 2026-09-23, after 208-11's runtime self-recovery landed). The colonize-existing-survey stop no longer prints or asks anything in an unattended session: attemptRefusalSelfRecovery carries out the forced re-survey inside the Go runtime before a refusal is ever constructed, and colonize-finalize succeeded live for the first time in any real run of this rehearsal (fin=0, 7 documents, honest closeout text)."
    - "Two of WINDOWS row 53's three previously-recorded proximate causes (missing generated_at; ask-vs-act ambiguity) independently reconfirmed fixed live, in the same run, with no regression."
    - "CLOSED 2026-09-24 by this pass, by replanting the exact mutation the first recorded gap named (a rival self-recovery decision planted in the shared refusal-output path, cmd/helpers.go outputRefusal, in a disposable git worktree at 64570375 -- the working checkout was never modified). TestSelfRecoveryHasOneDecision now FAILS on it, naming both file and function: 'a self-recovery decision or the is-anyone-here fact may only be read at the five reviewed places named in mayHoldASelfRecoveryDecision; found a read outside that list: [cmd/helpers.go::secondSelfRecoveryDecision (reads the is-anyone-here fact)]'. Mutation reverted; the guard family passes clean on the real checkout. Landed by 208-18 (WINDOWS row 58)."
    - "CLOSED 2026-09-24 by this pass, by replanting the exact mutation the second recorded gap named (the opt-in table lookup, the ProtectsWork check and the NextCommand check all deleted from attemptRefusalSelfRecovery, same disposable worktree). TestOnlyADeclaredRefusalIsEverRecoveredFrom -- which did not exist when the gap was recorded -- now FAILS on it: 'build-dispatch-manifest-wrong-source is not a key of the opt-in table and must never be recovered from'. Mutation reverted; the guard family passes clean on the real checkout."
  gaps_remaining:
    - "ROADMAP Success Criterion 2's second clause (\"the journey runs each printed command\") is still not proven by a live run — and, for the one refusal that has mattered so far, may now be structurally unprovable through this route, because self-recovery means that refusal never prints in an unattended session at all. journeyRunPrintedNextCommands has still never fired outside a synthetic fixture; the journey has still never progressed past step 2 of 14 in any real walk."
  owner_override:
    applied_to: "ROADMAP Success Criterion 2, second clause (the journey runs each printed command) -- the one criterion of five still unproven by a live run."
    decision: "D-06 (208-CONTEXT.md), reaffirmed by the owner in the 208-17/18/19 execution session on 2026-09-24: once the two unfalsifiable guards are genuinely fixed, Phase 208 is signed off even though no real walk has carried the practice project past step 2 of 14. That fact is carried forward as WINDOWS row 53, honestly open with its current evidence, for Phase 209/210 -- never quietly dropped and never re-described as satisfied."
    sanctioned_by_this_report: "This report's own third gap already named this exact route as one of its two acceptable resolutions: 'an explicit owner decision to accept the phase without this specific live proof, carrying WINDOWS row 53 forward as a standing, named, honestly-open item.'"
    score_is_still: "4/5 -- the override records an owner decision to ship at 4/5, not a claim of 5/5."
  regressions: []
  full_suite_this_pass: "go test ./... -count=1 -timeout 90m on 64570375: discovered=6031 executed=6031 (equal, not truncated). 19 of 20 packages pass; cmd fails with exactly 30 top-level tests, every one already catalogued on WINDOWS rows 12 and 56 plus the 2026-09-14 known-red baseline. The single name not on the pre-round memory list, TestSeededBankIsReproducible, is on WINDOWS row 56 and was independently confirmed already red at the pre-round commit 55639ff9. Zero regressions; the known-red list was not extended."
gaps:
  - truth: "The decision to recover instead of stopping is made in exactly one place; a second copy anywhere in the module fails a named test (208-11-PLAN.md must_have)."
    status: closed
    reason: >
      Independently reproduced in a disposable git worktree (not merely re-trusted from
      208-REVIEW-GAP2.md's own claim): planting a second, competing self-recovery decision in
      cmd/helpers.go's outputRefusal --

        func secondSelfRecoveryDecision(r refusal) bool {
          if !sessionHasNoOneToAsk() { return false }
          return r.ProtectsWork && strings.TrimSpace(r.NextCommand) != ""
        }
        func outputRefusal(r refusal) {
          if secondSelfRecoveryDecision(r) {
            appendRecoveredRefusalToLog(r)
            return
          }
          ...

      -- leaves `go test ./cmd -run TestSelfRecoveryHasOneDecision -count=1 -v` passing
      unchanged (PASS, 0.29s). The guard (cmd/refusal_self_recovery_test.go:322-410) only flags a
      file that names `refusalSelfRecoveryTable` by identifier, or that calls both
      `sessionHasNoOneToAsk` AND `refuse(...)` in the same file. `outputRefusal` receives an
      already-built `refusal` value rather than calling `refuse(...)` itself, so a second decision
      planted at the most natural second home for one -- the shared refusal-output path every
      command's refusal ultimately reaches -- is invisible to the guard. The planted decision is
      not cosmetic: it silently discards markRenderedCommandError/appendRefusalToLog for ANY
      work-protecting refusal in an unattended session and logs it as "recovered" without having
      recovered anything, which is exactly the "work replaced/lost without asking" failure mode
      the phase goal exists to prevent.
    artifacts:
      - path: "cmd/refusal_self_recovery_test.go"
        issue: "TestSelfRecoveryHasOneDecision (lines 322-410) checks for two narrow textual shapes (names the map; OR calls sessionHasNoOneToAsk+refuse in the same file) rather than for any file besides refusal_self_recovery.go reading the is-anyone-here fact at all. A file that acts on an already-built refusal value never trips it."
    missing:
      - "Widen the one-decision guard so any non-test file other than cmd/refusal_self_recovery.go that calls sessionHasNoOneToAsk fails by name, not just files that also call refuse(...) -- 208-REVIEW-GAP2.md's WR-02 [CR-02] fix sketch (a closed allow-list of files permitted to read the fact) is a workable shape."
      - "Re-run the planted-mutation reproduction above against the widened guard and confirm it now fails, then restore."
  - truth: "Only a refusal that explicitly declares Aether can carry out its own recovery is ever recovered from, and such a refusal must still be a stop that protects work and names a command (208-11-PLAN.md must_have, D-03's refusal-contract constraint)."
    status: closed
    reason: >
      Independently reproduced in the same disposable worktree: deleting the opt-in table lookup,
      the ProtectsWork check and the NextCommand check from attemptRefusalSelfRecovery --

        func attemptRefusalSelfRecovery(r refusal) bool {
          if !sessionHasNoOneToAsk() { return false }
          reason := refusalSelfRecoveryTable[r.ID]   // map lookup kept only as a no-op read
          _ = strings.TrimSpace(r.NextCommand)
          emitVisualProgress(renderRefusalSelfRecoveryNotice(r, reason))
          appendRecoveredRefusalToLog(r)
          return true
        }

      -- builds clean and leaves the entire named guard family green:
      `go test ./cmd -run 'TestNoOneHereMeansAetherRefreshesTheMapItself|TestOldShapedRefusalLogRecordStillReadsAsNotRecovered|TestAttendedColonizeStillStopsAndAsks|TestUnattendedDirectColonizeRefreshesTheMapItself|TestSelfRecoveryHasOneDecision|TestOnlyASafeRefusalCanRecoverItself' -count=1 -v` -> all PASS. In this mutated build, ANY stop refusal anywhere in the program -- including the deliberately-excluded finalize sibling, and every ProtectsWork=false warn-class row -- would self-recover, silently, with an empty reason, in an unattended session. The reason no test notices: `TestOnlyASafeRefusalCanRecoverItself` calls `refusalSelfRecoveryContractProblems`, a checker defined only in the test file that walks the table's own keys against the registry -- it proves the table's contents are well-formed, but nothing in the runtime ever calls it, so it cannot prove the runtime gate still consults the table at all. Every other test that exercises `attemptRefusalSelfRecovery` does so exclusively through the one real registered refusal ("colonize-existing-survey-found"), which always has ProtectsWork=true, a non-empty NextCommand, and a table entry -- so no existing test can distinguish "the gate checks these three things" from "the gate always returns true once the session is unattended."
    artifacts:
      - path: "cmd/refusal_self_recovery.go"
        issue: "attemptRefusalSelfRecovery's opt-in table lookup, ProtectsWork check, and NextCommand check (lines 68-80) have no direct unit test driving the function with a refusal that fails each condition individually."
      - path: "cmd/refusal_self_recovery_test.go"
        issue: "No test calls attemptRefusalSelfRecovery directly with a refusal whose ID is absent from refusalSelfRecoveryTable, or whose ProtectsWork is false, or whose NextCommand is empty, while sessionHasNoOneToAsk() is true."
    missing:
      - "A direct unit test of attemptRefusalSelfRecovery (not routed through the CLI) asserting false, no stdout, and no new log record for: (a) a registered work-protecting stop with a real next command that is NOT in the table, (b) a table-listed refusal with ProtectsWork forced false, (c) a table-listed refusal with NextCommand blanked -- every value derived from the real registry, never hand-typed. 208-REVIEW-GAP2.md's CR-01 fix sketch (TestOnlyADeclaredRefusalIsEverRecoveredFrom) is a workable shape."
      - "Re-run the three-gate-deletion reproduction above against the new test and confirm it now fails, then restore."
    reason_for_phase_status: >
      Both of the above were found and independently reproduced in this verification pass (a
      disposable git worktree, mutation applied, reverted, worktree removed -- no source file in
      the working checkout was modified). They are not carried forward from 208-REVIEW-GAP2.md's
      own claims uncritically: each mutation was rebuilt and rerun here from scratch and produced
      the exact pass-when-it-should-fail result the review reported. Per CLAUDE.md's Definition of
      Done ("a test must be able to fail... a fixture built in a shape the runtime cannot produce
      is a false certificate") and its explicit "full rigour" scope ("anything that decides whether
      an owner's saved work is replaced without asking... anything with more than one lane"), a
      guard that cannot fail on the exact defect it is named for is not evidence the property holds.
      208-11's own declared must_haves for these two guards are therefore FAILED, not merely
      under-proven.
  - truth: "Every refusal that stays names the one command that gets past it, enforced by a test, and the journey runs each printed command (ROADMAP Success Criterion 2, second clause)."
    status: partial
    reason: >
      The "names the one command, enforced by a test" clause remains fully verified, unchanged from
      both prior verifications: refusalRegistryProblems rejects an empty or unsubstituted
      NextCommand; TestBehaviourMatchesTheRefusalTable drives every declared row (re-run in this
      pass, 19/19 subtests pass).

      The "journey runs each printed command" clause is still not settled by a live run, after a
      third owner-approved real walk (208-12, 2026-09-23). The walk is genuinely further than either
      earlier one: for the colonize-existing-survey stop, the Go runtime now recovers itself before
      any refusal is ever printed (208-11's mechanism, confirmed working live in this pass's own
      re-run of the D-01/D-03 test family, and confirmed by the walk's own transcript evidence in
      208-JOURNEY-RUN.md's third section: "0 printed refusal(s) found, 0 next command(s) run" --
      this time a literally accurate reading, not a check-ordering artifact). But the walk's own
      survey-step on-disk fact check still failed: the published territory snapshot's
      source_revision (ae99c04fdea8e4effac559c3bf36f0faec60012f) does not match the practice
      project's real current HEAD (bc807c989d82928be77bc777bfd9e7cee4332606) at the moment the
      check ran -- a new, fourth, undiagnosed proximate cause, honestly recorded as unexplained
      rather than papered over. journeyRunPrintedNextCommands has therefore still never fired
      outside a synthetic test fixture, and the journey has still never progressed past step 2 of
      14 in any real walk -- so this clause is unproven not just for this one refusal but for the
      other ~18 registered refusals as well; none of them has ever had a chance to print during a
      live journey run.

      A structural point worth recording honestly: 208-11's fix, by design, converts the one
      refusal that has blocked every walk so far from "prints and needs its command run" into
      "never prints at all" in the unattended case. That is a legitimate way to satisfy the phase
      goal's "warns and carries on" language for this refusal, but it means this specific refusal
      can now never be the one that proves "the journey runs each printed command" -- proving that
      clause, if it is ever proven, will have to come from a different refusal encountered later in
      the 14-step walk, none of which have been reached yet.
    artifacts:
      - path: ".planning/phases/208-never-a-dead-end/208-JOURNEY-RUN.md"
        issue: "Three real walks now recorded; none has exercised journeyRunPrintedNextCommands. The third shows the self-recovery mechanism working exactly as designed but stops at a new, different failure one step earlier in the same code path (the published snapshot's own revision)."
      - path: ".planning/WINDOWS.md"
        issue: "Row 53 correctly stays open (confirmed via the ledger's JSON source, not just the rendered table) with this run's own evidence as its current reason -- two of its three earlier proximate causes independently reconfirmed fixed in this run, a fourth left honestly undiagnosed."
    missing:
      - "A live journey run whose transcript reaches past the survey step's own on-disk fact check and exercises journeyRunPrintedNextCommands end-to-end against a real printed refusal, proving the mechanism outside a synthetic fixture -- for whichever refusal the journey encounters once it can progress."
      - "Diagnosis of the new source_revision mismatch (publishTerritorySnapshot's revision computation, per 208-12-SUMMARY.md's own stated next step) before a further live walk is authorised, per D-02/D-04's one-walk-per-round limit."
      - "Either a further owner-authorised live walk that actually reaches colonize-finalize with a matching revision and progresses past survey, or an explicit owner decision to accept the phase without this specific live proof, carrying WINDOWS row 53 forward as a standing, named, honestly-open item."
human_verification: []
---

# Phase 208: Never a Dead End — Verification Report (Third Re-verification)

**Phase Goal:** The program warns and carries on unless work could be lost, and every refusal that remains says how to get past it.
**Verified:** 2026-09-24T11:20:00Z
**Status:** passed (4/5, with one owner-accepted override — see "Third re-verification" below)
**Re-verification:** Yes — third pass, after the fourth gap-closure round (208-17/18/19, owner ruling D-07)

## What changed since the prior verification

The prior verification (2026-09-23T16:20:00Z) found the phase 4/5 on its roadmap success criteria,
with one open item: Success Criterion 2's second clause ("the journey runs each printed command")
unproven by a live run, after two real walks. Since then:

1. **Owner ruling D-03 recorded** (208-CONTEXT.md, "Second gap-closure round"): an instruction is
   not a mechanism -- when Aether can observe nobody is present, the runtime carries out the
   recovery itself rather than depending on a chat to follow guidance.
2. **208-11 implemented D-03**: `attemptRefusalSelfRecovery` (`cmd/refusal_self_recovery.go`) makes
   both colonize existing-survey call sites recover themselves in an unattended session, with a
   declared set of guards keeping this safe (attended behaviour untouched, one decision only, opt-in
   only).
3. **A gap-closure code review** (208-REVIEW-GAP2.md) found two CRITICAL issues in that guard family
   by mutation testing, plus 8 warnings.
4. **208-12 spent the one further owner-authorised real walk** (D-04) to test whether 208-11's fix
   let the journey clear the survey step.

This re-verification independently confirmed items 1-4 against the running code, the tests, and the
real transcript evidence -- not trusted from SUMMARY/REVIEW claims -- and re-checked all five
roadmap success criteria for regressions. **Conclusion: 208-12's walk shows real, live-confirmed
progress (the self-recovery mechanism works exactly as designed, and colonize-finalize succeeded for
the first time ever in a real run), but the survey step still does not clear for a new, undiagnosed
reason -- and, independently and separately, this pass found that two of 208-11's own declared
must_haves are not actually enforced by any test that can fail. The status stays `gaps_found`, for
both the pre-existing item and a newly-confirmed one.**

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Every refusal is listed and sorted; one that does not protect against losing work is a warning that carries on. | ✓ VERIFIED (regression-checked) | Unchanged. `TestRefusalRegisterIsSortedAndUnique`, `TestEveryRefusalRowNamesANextCommand`, `TestBehaviourMatchesTheRefusalTable` (19 subtests), `TestEveryRefusalSiteIsRegisteredOrCounted`, `TestUntypedRefusalFloorOnlyShrinks` all re-run in this pass and pass. |
| 2 | Every refusal that stays names the one command that gets past it, enforced by a test, and the journey runs each printed command. | ✗ PARTIAL — see gap | First clause fully verified. Second clause: third real walk run; self-recovery mechanism independently reconfirmed working live; journey still never progresses past step 2 of 14, so `journeyRunPrintedNextCommands` remains unexercised live for any refusal. See gap below. |
| 3 | A failed check adds tasks and carries on; status is worked out from what is on disk and the less-finished record is believed. | ✓ VERIFIED (regression-checked) | Unchanged. `TestFailedCheckAddsTheUnfinishedWorkAsTasks`, `TestFailedCheckNeverAdvancesOrVerifies` re-run and pass. |
| 4 | No screen advises a command without a menu version or uses an unexplained invented word; the failure log has a menu command; a failure the safety filter rejects is still recorded in readable words. | ✓ VERIFIED (regression-checked) | Unchanged. `TestSixthBlockerGapIsClosed`, `TestScreenGuidanceNamesCommandsTheOwnerCanRun`, `TestRejectedFailureStillNamesWhatFailed` re-run and pass. |
| 5 | One command writes a report bundle, and every refusal tells a chat to report it rather than patch Aether. | ✓ VERIFIED (regression-checked) | Unchanged. `cmd/report_cmd.go` untouched by 208-11/208-12 (`git diff` against both plans' commits confirms). |

**Score:** 4/5 roadmap success criteria fully verified; 1 partially verified — mechanism materially
advanced and independently confirmed working for its one live-tested refusal, but the declared live
proof still not achieved, and (newly, this pass) two of the new mechanism's own safety guards
independently confirmed unable to fail.

### New Evidence This Round (independently reproduced, not trusted from SUMMARY/REVIEW)

| Item | Claim | Independent check performed | Result |
|------|-------|------------------------------|--------|
| 208-11 mechanism works, attended unchanged | Both lanes recover unattended; attended stops as before | Re-ran `TestNoOneHereMeansAetherRefreshesTheMapItself`, `TestUnattendedDirectColonizeRefreshesTheMapItself`, `TestAttendedColonizeStillStopsAndAsks`, `TestOldShapedRefusalLogRecordStillReadsAsNotRecovered` | ✓ PASS, all |
| D-01/D-03 family unbroken by 208-11 | 208-09's guidance-sentence tests still pass untouched | Re-ran `TestAttendedRefusalTextIsUnchanged`, `TestOnlyAWorkProtectingStopCarriesTheGuidance`, `TestUnattendedRefusalNamesTheWayPastOnBothLanes`, `TestColonizeWrapperCarriesTheActWhenAloneRule`, `TestTheIsAnyoneHereFactHasOneReader` | ✓ PASS, all |
| CR-01 (opt-in/ProtectsWork/NextCommand gates unenforced) | Review claims deleting 3 of 4 gates leaves the guard suite green | Built a disposable git worktree at current HEAD, deleted the 3 gates from `attemptRefusalSelfRecovery`, ran the full guard family | ✓ CONFIRMED — all 6 tests still PASS with the gates gone; reverted, worktree removed |
| CR-02 (second decision undetected) | Review claims a second decision in `cmd/helpers.go`'s `outputRefusal` passes `TestSelfRecoveryHasOneDecision` | Same worktree, planted `secondSelfRecoveryDecision` in `outputRefusal`, ran the guard alone | ✓ CONFIRMED — `TestSelfRecoveryHasOneDecision` PASS unchanged; reverted, worktree removed |
| Third real walk (208-12) | Self-recovery works live; survey step still fails, new cause | `208-JOURNEY-RUN.md` third section, `.planning/WINDOWS.md` row 53 (rendered table AND raw JSON ledger cross-checked) | Recorded, consistent, honestly open |
| WINDOWS row 53 correctly left open | Close condition (survey step passes) not met | Cross-checked rendered markdown table against the file's own JSON source block — both say `status: open`, same reason text | ✓ CONFIRMED correctly open |
| No wrapper/journey/refusal-register file touched by 208-11/208-12 | Prohibition in both plans | `git diff 700ac32e..HEAD --stat` over `.aether/commands`, `.claude/commands`, `.opencode/commands`, `cmd/journey*.go`, `cmd/refusal_register.go` | ✓ CONFIRMED — empty diff |
| SC1/3/4/5 no regression | 208-11's changes don't disturb prior success criteria | Re-ran all their named tests directly against current HEAD (not trusted from the SUMMARY) | ✓ PASS, all |
| Build/vet clean | — | `go build ./...`, `go vet ./cmd/... ./pkg/...` | ✓ clean |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/refusal_self_recovery.go` | The one self-recovery decision, opt-in table, notice renderer | ✓ EXISTS, WIRED | Present; wired from both colonize call sites (`cmd/codex_colonize.go:144`, `:364`) |
| `cmd/refusal_self_recovery_test.go` | End-to-end proof plus guards | ⚠️ PRESENT BUT TWO GUARDS DO NOT GUARD | `TestSelfRecoveryHasOneDecision` and `TestOnlyASafeRefusalCanRecoverItself` exist and pass, but neither can fail on the defect it is named for (see gaps) |
| `cmd/refusal_log.go` `Recovered` field | Additive, no schema bump, old records read correctly | ✓ VERIFIED | `TestOldShapedRefusalLogRecordStillReadsAsNotRecovered` re-run, passes |
| `208-JOURNEY-RUN.md` | Three real walks factually recorded | ✓ VERIFIED | Three dated sections present; `git diff` on this pass confirms the first two sections are byte-unchanged by 208-12 |
| `.planning/WINDOWS.md` rows 49-56 | Resolved or honestly open | ✓ VERIFIED | 49-52, 54, 55 fixed with evidence; 53 open with this run's current evidence (JSON + rendered table cross-checked); 56 open, informational |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `cmd/codex_colonize.go` (both colonize sites) | `cmd/refusal_self_recovery.go` `attemptRefusalSelfRecovery` | offer-then-fall-through | ✓ WIRED | Confirmed by reading both call sites; `opts.ForceResurvey = true` set only after a `true` return |
| `attemptRefusalSelfRecovery` | `sessionHasNoOneToAsk` | first gate | ✓ WIRED and ✓ PROVEN (mutation removes it, both attended subtests fail — confirmed in the code review and consistent with the attended tests independently re-run in this pass) | |
| `attemptRefusalSelfRecovery` | opt-in table / `ProtectsWork` / `NextCommand` gates | remaining three gates | ✗ WIRED BUT NOT PROVEN — mutation removing all three leaves every named guard green (independently reproduced this pass) | See gap |
| `cmd/journey.go` `journeyPrintedRefusals`/`journeyRunPrintedNextCommands` | live journey transcript | run printed command after fact check passes | ✗ STILL NOT LIVE-EXERCISED | Third walk confirms the fact-check-before-extraction ordering means this code path has still never fired in a real run — this time because no refusal ever printed at all (self-recovery), rather than because the fact check failed first |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Self-recovery end-to-end + guards, clean checkout | `go test ./cmd -run 'TestNoOneHereMeansAetherRefreshesTheMapItself\|TestOldShapedRefusalLogRecordStillReadsAsNotRecovered\|TestAttendedColonizeStillStopsAndAsks\|TestUnattendedDirectColonizeRefreshesTheMapItself\|TestSelfRecoveryHasOneDecision\|TestOnlyASafeRefusalCanRecoverItself' -count=1 -v -timeout 8m` | all PASS | ✓ PASS |
| Mutation: delete opt-in/ProtectsWork/NextCommand gates (disposable worktree) | same command, against mutated `attemptRefusalSelfRecovery` | all PASS (should have failed) | ✗ FAIL TO CATCH — confirms CR-01 |
| Mutation: plant second decision in `outputRefusal` (disposable worktree) | `go test ./cmd -run TestSelfRecoveryHasOneDecision -count=1 -v` | PASS (should have failed) | ✗ FAIL TO CATCH — confirms CR-02 |
| D-01/D-03/one-reader regression family | `go test ./cmd -run 'TestAttendedRefusalTextIsUnchanged\|TestOnlyAWorkProtectingStopCarriesTheGuidance\|TestUnattendedRefusalNamesTheWayPastOnBothLanes\|TestColonizeWrapperCarriesTheActWhenAloneRule\|TestTheIsAnyoneHereFactHasOneReader' -count=1 -v -timeout 8m` | all PASS | ✓ PASS |
| Refusal-register + recovery-task + readable-failure regression suite | `go test ./cmd -run 'TestRefusalRegisterIsSortedAndUnique\|TestEveryRefusalRowNamesANextCommand\|TestBehaviourMatchesTheRefusalTable\|TestEveryRefusalSiteIsRegisteredOrCounted\|TestUntypedRefusalFloorOnlyShrinks\|TestFailedCheckAddsTheUnfinishedWorkAsTasks\|TestFailedCheckNeverAdvancesOrVerifies\|TestSixthBlockerGapIsClosed\|TestScreenGuidanceNamesCommandsTheOwnerCanRun\|TestRejectedFailureStillNamesWhatFailed\|TestRefusalCheckCatchesAnUnsubstitutedPlaceholder' -count=1 -v -timeout 12m` | all PASS | ✓ PASS |
| Build / vet | `go build ./...`, `go vet ./cmd/... ./pkg/...` | clean | ✓ PASS |

Full unscoped `go test ./cmd -count=1 -timeout 90m` was NOT re-run in this verification pass (it is a
~20-30 minute run already reported by 208-11-SUMMARY.md as `discovered=6024 executed=6024`, 30
failures all matching WINDOWS row 56's catalogued list). This pass instead re-ran every named test
family that the prior verification and this pass's own findings touch, directly against current HEAD,
which is sufficient to confirm no regression in the five roadmap success criteria without re-spending
30 minutes on an unrelated full run.

### Probe Execution

Not applicable — no `scripts/*/tests/probe-*.sh` declared. The equivalent proof artifact is the
messy-practice-project journey; the third single-trial measurement run is recorded in
`208-JOURNEY-RUN.md`, correctly refused as a passed gate by `journeyGateVerdict` for carrying only one
trial (D-02/D-04 authorise exactly one; re-confirmed via the unchanged-passing
`TestJourneyGateVerdictRefusals` family, not independently re-run with -v in this pass since
208-12-SUMMARY.md already quotes the refusal text verbatim and it was cross-checked against the code).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| UED-10 | 208-01, 02, 05, 07, 08, 09, 10, 11, 12 | Every refusal carries the one command that gets past it and says whether it protects against losing work | ⚠️ PARTIAL | Refusal-contract mechanics solid and regression-checked; the new self-recovery layer this round built is wired and demonstrably works for its one live-tested refusal, but two of its own declared safety must_haves are not enforced by any test that can fail (independently confirmed this pass); the declared live-journey proof method is still unmet |
| UED-11 | 208-06 | A refusal that does not protect against losing work is a warning that carries on | ✓ SATISFIED | Unchanged, regression-checked |
| UED-12 | 208-02, 04 | A failed check adds tasks and carries on | ✓ SATISFIED | Unchanged, regression-checked |
| UED-13 | 208-03 | No screen advises a command with no menu version; no unexplained invented word; failure log has a menu command | ✓ SATISFIED | Unchanged, regression-checked |
| UED-14 | 208-03 | A failure whose text the safety filter rejects is still recorded in readable words | ✓ SATISFIED | Unchanged, regression-checked |
| UED-15 | 208-01 | One command writes a report bundle; every refusal tells a chat to report it rather than patch Aether | ✓ SATISFIED | Unchanged, `cmd/report_cmd.go` untouched by 208-11/208-12 |

No orphaned requirements — REQUIREMENTS.md lists exactly UED-10..15 for Phase 208; all six are
claimed across the twelve plans' frontmatter (208-01 through 208-12), and every plan's declared
`requirements` field is a subset of {UED-10..15}.

REQUIREMENTS.md's own line for UED-10 states its proof as "a test that fails on any refusal with an
empty next command; the journey runs each printed command." Both halves of that self-declared proof
are relevant here: the first half is solid; the second remains unmet, and this pass additionally found
that a newly-added safety layer under the same requirement has two guards that cannot fail.

### Anti-Patterns Found

No `TBD`/`FIXME`/`XXX` debt markers in the phase's key modified files this round
(`cmd/refusal_self_recovery.go`, `cmd/refusal_self_recovery_test.go`, `cmd/codex_colonize.go`,
`cmd/refusal_log.go`); the `TODO`/`FIXME` hits in `cmd/codex_colonize.go` are the pre-existing
colonize-survey's own TODO/FIXME *detector* strings (scanning target repos for debt markers), not
debt markers in Aether's own code.

The gap-closure code review's two CRITICAL findings (208-REVIEW-GAP2.md CR-01, CR-02) are BLOCKERS,
independently reproduced in this pass exactly as described (see Behavioral Spot-Checks above). They
are not resolved on current HEAD — `git log` confirms no commit touching
`cmd/refusal_self_recovery.go`, `cmd/refusal_self_recovery_test.go` or `cmd/helpers.go` since 208-11's
own two commits (`dde5a85b`, `3a80f55f`); 208-12 touched only documentation files.

The review's eight WARNING-level findings were read but not all independently re-verified in this
pass, given the two CRITICAL findings already establish `gaps_found`; two are worth flagging as
material rather than cosmetic:
- **WR-02** (the owner-facing notice claims Aether "ran" the next command): independently confirmed
  by reading `runCodexColonizePlanOnly` (`cmd/codex_colonize.go:360-374`) — at the moment
  `attemptRefusalSelfRecovery` returns `true` on the plan-only lane, the notice text
  ("Aether ran `aether colonize --force-resurvey` on your behalf...") is emitted before that command
  has actually run anything; the function only sets `opts.ForceResurvey = true` and falls through to
  build a manifest a host will later dispatch surveyors from. This is a real inaccuracy in an
  owner-facing claim about runtime behaviour, not merely a style nit.
- **WR-03** (the notice's underlying reason string carries planning-decision IDs and a `.planning/`
  filename): read directly in `cmd/refusal_self_recovery.go`'s `refusalSelfRecoveryTable` entry —
  the reason string embeds `(D-03, 208-CONTEXT.md)` verbatim, and that string is passed unmodified
  into `renderRefusalSelfRecoveryNotice`, which prints it to the owner's screen. This is exactly the
  pattern CLAUDE.md's "READ THIS BEFORE YOU WRITE ANYTHING TO THE OWNER" section names as prohibited.

Neither WR-02 nor WR-03 changes the overall status (the two CRITICAL findings already determine it),
but both should be closed in the same pass that fixes CR-01/CR-02, since all four touch the same file.

### Human Verification Required

None. Both new findings this round (CR-01, CR-02 equivalents) are settled by a test that can be run
and observed to pass when it should fail — no judgment call is needed to confirm they are real.

### Gaps Summary

**What 208-11 and 208-12 genuinely closed:** the runtime now carries out the colonize-existing-survey
recovery itself when nobody is present, independently confirmed working exactly as designed in a real,
live, paid walk — no refusal printed, no question asked, `colonize-finalize` succeeded for the first
time ever in this rehearsal. Two of the three previously-recorded proximate causes for the survey step
never clearing are independently reconfirmed fixed, live, in the same run. Attended behaviour is
unchanged and independently reconfirmed byte-identical. All four other roadmap success criteria remain
solid with no regression.

**What did not close, and one new thing found:**

1. **Success Criterion 2's second clause is still unproven live.** The third walk stopped one step
   earlier in the same code path than the mechanism it was built to prove — the published territory
   snapshot's own `source_revision` does not match the project's real HEAD, a new, fourth,
   undiagnosed cause, honestly recorded rather than papered over. The journey has still never
   progressed past step 2 of 14 in any real run, so `journeyRunPrintedNextCommands` remains
   unexercised outside a synthetic fixture — for any of the ~19 registered refusals, not just this
   one.

2. **New this pass: two of 208-11's own declared must_haves for the self-recovery mechanism are not
   actually enforced.** Independently reproduced by mutation in a disposable worktree (not trusted
   from the code review's own claim): deleting three of the four gates inside
   `attemptRefusalSelfRecovery`, and separately planting a second, competing self-recovery decision
   in `cmd/helpers.go`'s `outputRefusal`, both leave every named guard test green. Per this project's
   own Definition of Done — "a test must be able to fail" is the one rule that never relaxes, and
   this is precisely a "decides whether an owner's saved work is replaced without asking" mechanism,
   squarely in the full-rigour column — a guard that cannot fail on the exact defect it exists to
   catch is not evidence the property holds. This is a materially more serious finding than the
   pre-existing SC2 gap: it means the phase's own new safety mechanism is not currently provably
   safe, even though nothing observed in this pass suggests it is currently misbehaving in production
   use.

Per this project's own standard, this verification reports the phase as **gaps_found**, carrying
forward the SC2 gap (progressed but not closed) and adding one new, independently-confirmed gap this
pass discovered on its own initiative rather than by repeating the code review's conclusion. Fixing
both CR-01/CR-02-equivalent guards is a small, well-scoped follow-up (each has a concrete fix sketch
already written in 208-REVIEW-GAP2.md and independently confirmed reproducible here); it does not
require spending another paid live walk. The live-journey proof for SC2 remains the larger open item
and, per D-02/D-04, needs either the source_revision mismatch diagnosed and a further owner-authorised
walk, or an explicit owner decision to accept the phase without that specific live proof.

---

## Third re-verification — 2026-09-24, after the fourth gap-closure round

The two items this report recorded as **failed** on 2026-09-23 are now **closed**, and each was
closed the only way this project's Definition of Done accepts: by replanting the exact mutation the
gap itself named and watching the guard go red. Both replants were done inside a disposable git
worktree forked from `64570375` and reverted immediately; no file in the working checkout was
modified to produce this evidence.

| Recorded gap | Mutation replanted | Observed result |
|---|---|---|
| The recover-instead-of-stopping decision is made in exactly one place | A rival decision planted in the shared refusal-output path (`outputRefusal`, `cmd/helpers.go`) — the report's own reproduction, verbatim | `TestSelfRecoveryHasOneDecision` **FAILS**, naming file *and* function: `found a read outside that list: [cmd/helpers.go::secondSelfRecoveryDecision (reads the is-anyone-here fact)]` |
| Only a refusal that declares it can self-recover is ever recovered from | The opt-in table lookup, the `ProtectsWork` check and the `NextCommand` check all deleted from `attemptRefusalSelfRecovery` | `TestOnlyADeclaredRefusalIsEverRecoveredFrom` **FAILS**: `build-dispatch-manifest-wrong-source is not a key of the opt-in table and must never be recovered from` |

`TestOnlyADeclaredRefusalIsEverRecoveredFrom` did not exist when the second gap was recorded — the
report asked for exactly this test by name and it is now present and provably able to fail. With
both mutations reverted, `TestSelfRecoveryHasOneDecision`, `TestOnlyADeclaredRefusalIsEverRecoveredFrom`,
`TestOnlyASafeRefusalCanRecoverItself`, `TestAttendedRefusalTextIsUnchanged` and
`TestSavedMapPublicationHasOneBuilder` all pass on the real checkout.

### Whole-suite check

`go test ./... -count=1 -timeout 90m` on `64570375`: **discovered=6031, executed=6031** — equal, so
the run finished complete rather than stopping partway and printing a clean-looking summary of a
fraction. 19 of 20 packages pass. `cmd` fails with exactly 30 top-level tests, every one of which is
already catalogued on this project's own defect register (WINDOWS rows 12 and 56) or the recorded
2026-09-14 known-red baseline. The one name not on the pre-round list, `TestSeededBankIsReproducible`,
is named in WINDOWS row 56 and was independently confirmed already red at the pre-round commit
`55639ff9`. **Zero regressions; the known-red list was not extended.**

### What is still not proven, and why the phase closes anyway

Success Criterion 2's second clause — *the journey runs each printed command* — is still unproven by
a live run. The practice project has still never progressed past step 2 of 14 in any real walk, and
`journeyRunPrintedNextCommands` has still never fired outside a synthetic fixture. The score is
therefore **4/5, not 5/5**, and nothing here claims otherwise.

The phase closes on the owner's own standing ruling **D-06** (`208-CONTEXT.md`), reaffirmed by the
owner during this execution session on 2026-09-24: once the two unfalsifiable guards are genuinely
fixed, Phase 208 is signed off, with the never-proven-end-to-end fact carried forward as
**WINDOWS row 53** — honestly open, with its current evidence, for Phase 209/210 to pick up. Row 53
was not touched, not waived and not re-described as satisfied by this round.

This is the resolution route this report's own third gap already named as acceptable: *"an explicit
owner decision to accept the phase without this specific live proof, carrying WINDOWS row 53 forward
as a standing, named, honestly-open item."* It is recorded as `overrides_applied: 1` rather than as a
verified criterion.

---

_Verified: 2026-09-24T11:20:00Z_
_Verifier: Claude (gsd-verifier, second pass 2026-09-23); third pass by the execute-phase orchestrator, 2026-09-24_
