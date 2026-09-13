---
phase: 203-biological-runtime
plan: "12"
subsystem: recruitment
tags: [credit, cec-07, agency-receipt, decision-join, biological-runtime]

# Dependency graph
requires:
  - phase: 203-biological-runtime
    provides: "203-10's trophallaxisPacket, packTrophallaxisPacket/acknowledgeTrophallaxisPacket/recordTrophallaxisDecision, and its recorded colony.LifecycleDecision -- this plan's public-path tests and its production resolver both drive/read that exact chain"
  - phase: 203-biological-runtime
    provides: "203-07's bound recruitmentResult (Receipt-guarded exactly-once binding) that packTrophallaxisPacket itself requires"
provides:
  - "recruitmentCreditRecord / recruitmentCreditOutcomes (helpful, neutral, harmful, pending) / recruitmentContributionKinds (recruitment result, note, memory item, specialist contribution) and recordRecruitmentCredit -- the one function in cmd/ that writes credit/records.json, replay-safe, closed-vocabulary, both-facts-gated for any earned (non-pending) outcome"
  - "recruitmentCreditForDecision / recruitmentCreditAll / recruitmentCreditForContribution -- deterministic query surface (timestamp desc, record id asc tie-break) and renderRecruitmentCreditOutcome, the three-way rendering (uncredited / pending / earned)"
  - "agencyEvidenceFromTrophallaxisDecision (cmd/agency_contract.go) -- the read-only resolver populating AgencyReceiptEvidence.ChangedDecision/EffectEvidence from real trophallaxis-decision + recruitment-credit data, finally feeding BuildAgencySignalResult's already-correct both-or-neither join"
  - "SwarmPhase202Limitation deleted; BuildSwarmInterventionContract's Limitation field now states its actual, present-tense capability gap instead of a phase-numbered claim that would go stale the moment that phase shipped"
affects: [203-13, 203-14, 203-15]

# Actuals (#2632)
actuals:
  tokens: 11300
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A closed outcome vocabulary with an explicit 'pending' member distinct from both 'no record' and any earned outcome, so a decision-changed-but-unverified contribution is neither silently credited nor indistinguishable from a contribution nobody ever measured"
    - "A single credit-record writer whose OWN signature carries both facts a caller must supply (changed-decision id, effect-evidence id), gated by an AST test asserting no second write path can drop either parameter -- mirrors cmd/recruitment_result_test.go's TestEveryResultWriteGoesThroughTheBinding chokepoint style"
    - "A read-only resolver (agencyEvidenceFromTrophallaxisDecision) that joins two independently-written stores (recruitment/packets.json, credit/records.json) purely by reading, never by mutating either -- the join lives entirely in the read path"
    - "A trophallaxis packet's own PacketID doubles as the effect-evidence identifier, since recordTrophallaxisDecision (203-10) already guarantees PacketID is always present in the decision's own EvidenceIDs -- this satisfies the both-or-neither invariant's evidence-linkage check with zero changes to cmd/trophallaxis.go"

key-files:
  created:
    - cmd/recruitment_credit.go
    - cmd/recruitment_credit_test.go
    - cmd/agency_contract_test.go
  modified:
    - cmd/agency_contract.go
    - cmd/swarm_scope_199_test.go

key-decisions:
  - "recordRecruitmentCredit is the sole writer of credit/records.json, and 'both facts present' is enforced structurally, not just at the call site: a changed decision with no effect evidence is stored as pending (a real, persisted, distinct state); effect evidence with no changed decision is refused outright; only when both are present can the outcome be one of the three earnable, declared values (helpful/neutral/harmful) -- pending itself is refused in that branch, so an earned outcome can never appear without both facts."
  - "The effect-evidence identifier a credit record names is the trophallaxis packet's own PacketID, not a newly invented evidence object. Since recordTrophallaxisDecision (203-10) already always includes PacketID in the recorded decision's EvidenceIDs, this makes the both-or-neither invariant's own 'evidence must be linked to the decision' check (cmd/agency_contract.go's agencyContainsID) satisfied by construction, using only 203-10's existing, unmodified guarantee -- no change to cmd/trophallaxis.go was needed or made."
  - "The stale SwarmPhase202Limitation sentence was not simply deleted -- its ONE call site (BuildSwarmInterventionContract) was traced to confirm swarmInterventionPreflight (cmd/swarm_cmd.go) never actually supplies CheckpointEvidenceID today, so the genuine, present-tense replacement names that real gap ('no typed checkpoint evidence is supplied to this preflight') rather than inventing an equally vague substitute."
  - "TestAgencyNoLimitationNamesAFuturePhase builds the identifier it scans for by string concatenation, mirroring cmd/trophallaxis_test.go's own precedent for its retired command names -- a literal occurrence of the deleted identifier in this test's own source would otherwise trip the very grep the plan's acceptance criteria run against the whole of cmd/."

patterns-established:
  - "Pending-vs-credited-vs-uncredited is a three-state, explicitly renderable vocabulary (renderRecruitmentCreditOutcome), never collapsed to a boolean -- a future consumer rendering 'did this help' can distinguish 'nothing recorded yet', 'we know it changed something but not yet whether it helped', and 'we know exactly what it did, including if it hurt'."

requirements-completed: []
# CEC-07 is intentionally NOT marked complete here: it is also declared by
# sibling plan 203-13 (.planning/phases/203-biological-runtime/203-13-PLAN.md),
# which has no SUMMARY yet at the time this plan finished. Per the phase's
# shared-ID gate, CEC-07 flips to Complete only once every plan declaring it
# has its own SUMMARY -- marking it here would let this plan's finish flip a
# requirement 203-13 has not yet delivered its own half of.

coverage:
  - id: D1
    description: "A credit record can say a contribution was helpful, neutral, or harmful -- and a changed decision with no verified outcome yet is recorded as pending, never silently promoted to helpful. Two different contributions changing the same decision each get their own record; the same contribution+decision replayed twice writes nothing."
    requirement: "CEC-07"
    verification:
      - kind: unit
        ref: "cmd/recruitment_credit_test.go#TestRecruitmentCreditOutcomeVocabulary"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_credit_test.go#TestRecruitmentCreditRecordsAHarmfulOutcomeDirectlyOnTheFile"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_credit_test.go#TestRecruitmentCreditReplayIsByteIdentical"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_credit_test.go#TestRecruitmentCreditTwoContributionsOneDecision"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_credit_test.go#TestRecruitmentCreditSortOrderIsStableAcrossRepeatedReads"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_credit_test.go#TestRecruitmentCreditUncreditedRendersExistingNoMeasuredEffectWording"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_credit_test.go#TestRecruitmentCreditNeitherFactRecordsNothing"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_credit_test.go#TestRecruitmentCreditEffectEvidenceWithoutDecisionIsRefused"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_credit_test.go#TestRecruitmentCreditDecisionOnlyIsPendingNotHelpful"
        status: pass
    human_judgment: false
  - id: D2
    description: "The pre-existing, already-correct both-or-neither join in AgencyReceiptEvidence (cmd/agency_contract.go) is now populated from real recorded evidence -- a trophallaxis packet's own decision plus its recruitment credit record -- instead of remaining permanently starved. The stale SwarmPhase202Limitation sentence, which claimed this machinery still awaited an already-shipped phase, is deleted, and a source-scan test enforces it cannot silently return."
    requirement: "CEC-07"
    verification:
      - kind: unit
        ref: "cmd/agency_contract_test.go#TestAgencyEvidenceFromTrophallaxisDecision"
        status: pass
      - kind: unit
        ref: "cmd/agency_contract_test.go#TestAgencyEvidenceFromTrophallaxisDecisionSkipsPending"
        status: pass
      - kind: unit
        ref: "cmd/agency_contract_test.go#TestAgencyNoLimitationNamesAFuturePhase"
        status: pass
      - kind: unit
        ref: "cmd/swarm_scope_199_test.go#TestSwarmScope199"
        status: pass
    human_judgment: false
  - id: D3
    description: "Credit cannot be inferred: a note delivered into a real resolved worker brief followed by a real passing verification step produces zero credit records; no function in cmd/ can write a non-pending credit record without both a changed-decision and an effect-evidence identifier (an AST guard, proven against a synthetic violation); a harmful outcome is reachable end to end through the real pack/acknowledge/decide/credit path, not only a unit fixture."
    requirement: "CEC-07"
    verification:
      - kind: unit
        ref: "cmd/recruitment_credit_test.go#TestDeliveredPlusPassingIsNotCredit"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_credit_test.go#TestCreditRequiresBothFacts"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_credit_test.go#TestHarmfulOutcomeIsReachable"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-09-13
status: complete
---

# Phase 203 Plan 12: Recruitment Credit and the Real Decision-Effect Join Summary

**A closed-vocabulary credit record (helpful/neutral/harmful/pending) that only a fully-facted write can earn, joined into the pre-existing (and previously starved) `AgencyReceiptEvidence` both-or-neither invariant with real trophallaxis-decision evidence, and the stale "awaits Phase 202" limitation sentence deleted for good.**

## Performance

- **Duration:** 55 min
- **Completed:** 2026-09-13T15:12Z
- **Tasks:** 3 completed
- **Files modified:** 5 (3 created, 2 modified)

## Accomplishments

- `cmd/recruitment_credit.go` (new): `recruitmentCreditRecord`, the closed 4-member outcome vocabulary (`helpful`/`neutral`/`harmful`/`pending`) with `recruitmentCreditOutcomeNames()`, the four declared contribution kinds (recruitment result, note, memory item, specialist contribution) with `recruitmentContributionKinds()`, and `recordRecruitmentCredit` -- the one function in `cmd/` that writes `credit/records.json`. Neither fact present earns no record at all; a changed decision alone is recorded as pending, never promoted to an earned outcome; effect evidence with no decision is refused; a replay of the same (contribution, decision) pair is byte-identical.
- `recruitmentCreditForDecision`/`recruitmentCreditAll`/`recruitmentCreditForContribution` give a deterministic, byte-identical-on-repeat query surface (timestamp descending, record id ascending tie-break); `renderRecruitmentCreditOutcome` renders three distinguishable states -- the existing no-measured-effect wording for uncredited, a new pending phrase for decision-without-outcome, and the outcome name itself once earned.
- `cmd/agency_contract.go`: added `agencyEvidenceFromTrophallaxisDecision`, a read-only resolver joining a trophallaxis packet's own recorded `colony.LifecycleDecision` (203-10) with the recruitment credit record for it, so `BuildAgencySignalResult`'s already-correct both-or-neither join finally receives genuine recorded data. Deleted `SwarmPhase202Limitation` and its one call site (`BuildSwarmInterventionContract`), replacing it with a present-tense description of the actual, currently-missing evidence (`swarmInterventionPreflight` never supplies `CheckpointEvidenceID` today) -- never a phase number.
- `cmd/agency_contract_test.go` (new): the end-to-end public-path proof (pack -> acknowledge -> decide -> credit -> resolve -> feed into `BuildAgencySignalResult`), a pending-skip proof, and a source-scan test enforcing the deleted limitation identifier and any "awaits Phase N" sentence never return to `cmd/`.
- `cmd/swarm_scope_199_test.go`: updated the pre-existing assertions that checked for the literal "Phase 202" limitation text, since that text no longer exists (see Deviations).

## Task Commits

Each task was committed atomically:

1. **Task 1: A credit record that can say helped, did nothing, or made it worse** - `c12e3d13` (feat)
2. **Task 2: Populate the existing join from real recorded evidence and delete the stale limitation** - `b436e7b9` (feat)
3. **Task 3: Prove credit cannot be inferred** - `f9589ea2` (test)

**Plan metadata:** this commit (docs: complete plan)

## Files Created/Modified

- `cmd/recruitment_credit.go` - The credit record, its outcome/contribution vocabularies, and the sole writer/query surface
- `cmd/recruitment_credit_test.go` - RED/GREEN-proven tests for all three tasks, including the AST guard and the delivered-plus-passing negative proof
- `cmd/agency_contract.go` - The new resolver, and the stale limitation's deletion
- `cmd/agency_contract_test.go` - The end-to-end public-path proof and the source-scan enforcement test
- `cmd/swarm_scope_199_test.go` - Assertions updated to match the deleted limitation's present-tense replacement (deviation, see below)

## Decisions Made

See `key-decisions` in frontmatter. In brief: `recordRecruitmentCredit` is the one writer, structurally gated so an earned (non-pending) outcome always requires both facts; the trophallaxis packet's own `PacketID` doubles as the effect-evidence identifier so the both-or-neither invariant is satisfied using 203-10's existing guarantee with zero changes to `cmd/trophallaxis.go`; the deleted limitation's replacement names the real, present-tense capability gap rather than an equally vague substitute; and the source-scan test builds its target identifier by concatenation so it does not trip its own grep.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated `cmd/swarm_scope_199_test.go`'s stale assertions**
- **Found during:** Task 2 (deleting `SwarmPhase202Limitation`)
- **Issue:** This pre-existing test asserted `strings.Contains(contract.Limitation, "Phase 202")` and the same literal string in the rendered visual output -- exactly the stale sentence Task 2's own acceptance criteria required deleting. Deleting the sentence without updating this test would have left it red as a direct, foreseeable consequence of an in-scope change.
- **Fix:** Updated the two assertions to check for the new present-tense wording ("checkpoint evidence") and added an explicit assertion that the limitation sentence never names a phase number again.
- **Files modified:** `cmd/swarm_scope_199_test.go`
- **Verification:** `go test ./cmd -run '^TestSwarmScope199$' -count=1` -- PASS
- **Committed in:** `b436e7b9` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug, a direct and foreseeable consequence of Task 2's own required deletion)
**Impact on plan:** No scope creep -- the fix lives entirely in a test file whose only broken assertions were the ones this plan's own acceptance criteria required to break. Recorded in the cross-phase defect ledger (`.planning/WINDOWS.md`, entry 20) per the broken-windows discipline even though it was fixed in the same commit, not left open.

## Inherited Loose End (203-10)

203-10's `recordTrophallaxisDecision` was flagged as not yet wired into `cmd/codex_verify_advance.go`'s `runContinueAcceptVerifyAdvance` -- the single phase-level accept/verify/advance boundary. This plan's own objective is explicit that the join CEC-07 actually needs is `AgencyReceiptEvidence.ChangedDecision`/`EffectEvidence` (cmd/agency_contract.go), which is a **different** boundary from `runContinueAcceptVerifyAdvance`: reading `cmd/codex_verify_advance.go` this session confirms it is a phase-level decision body (gates, review, waves) with no per-recruitment-packet join point at all, and it remains outside this plan's declared file scope (`cmd/recruitment_credit.go`, `cmd/agency_contract.go`, `cmd/agency_contract_test.go`, `cmd/recruitment_credit_test.go`).

**This plan closed the CEC-07 join it was actually asked to close** (`AgencyReceiptEvidence`, populated from real trophallaxis-decision + credit data) -- proven end to end by `TestAgencyEvidenceFromTrophallaxisDecision`. **The separate question of whether a trophallaxis decision should also reach `runContinueAcceptVerifyAdvance` remains open and unresolved**, named explicitly here (and recorded in `.planning/WINDOWS.md`, entry 21) rather than silently absorbed or dropped, per this plan's own instruction: a follow-up plan (203-14/203-15) should confirm whether that additional wiring is actually required or whether the `AgencyReceiptEvidence` join alone satisfies BIO-05/CEC-07's full intent.

## Issues Encountered

- The full, unscoped `go test ./cmd -count=1` run takes on the order of minutes on this machine and was still running in the background when this plan's own targeted verification completed; every test this plan's changes could plausibly affect (`Trophallaxis|Recruitment|SwarmScope199|Agency`, plus the named ratchets `TestColonyStateWriteAllowlistOnlyShrinks`/`TestNextActionNeverHardcoded`, plus `TestPlatformParityGolden`/`TestRegressionSnapshot`/`TestOrphanAllowlistOnlyShrinks`) was run directly and passes. `TestNoRegisteredSubcommandIsUnreferenced` fails exactly as documented in this plan's own project-specific warnings (`aether recruit is registered but nothing calls it`, expected-red and owned by 203-15) -- confirmed pre-existing and unrelated to this plan's changes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `recordRecruitmentCredit`, `recruitmentCreditForDecision`/`recruitmentCreditAll`, and `agencyEvidenceFromTrophallaxisDecision` are all real, tested, and reachable within the `cmd` package -- ready for a future plan to wire a real dispatch/decision site into them, or for 203-13's outcome-weighted strengthen/weaken work to read from `credit/records.json` directly.
- **Blocker/concern for a human or a follow-up plan:** see "Inherited Loose End" above -- whether `recordTrophallaxisDecision`'s decision should also reach `runContinueAcceptVerifyAdvance` is unresolved and named for 203-14/203-15.
- CEC-07 is shared with sibling plan 203-13 (still unfinished at the time this plan completed) -- this plan intentionally does NOT mark CEC-07 complete in REQUIREMENTS.md, per the phase's shared-ID gate; it will flip once 203-13 also finishes.

---
*Phase: 203-biological-runtime*
*Completed: 2026-09-13*

## Self-Check: PASSED

- FOUND: `cmd/recruitment_credit.go` (created)
- FOUND: `cmd/recruitment_credit_test.go` (created)
- FOUND: `cmd/agency_contract_test.go` (created)
- FOUND: `cmd/agency_contract.go` (modified, `SwarmPhase202Limitation` deleted, `agencyEvidenceFromTrophallaxisDecision` added)
- FOUND: `cmd/swarm_scope_199_test.go` (modified, stale assertions updated)
- FOUND: commit `c12e3d13` (feat(203-12): a credit record that can say helped, did nothing, or made it worse) in `git log --oneline`
- FOUND: commit `b436e7b9` (feat(203-12): populate the existing decision-effect join from real evidence) in `git log --oneline`
- FOUND: commit `f9589ea2` (test(203-12): prove credit cannot be inferred from delivery plus a pass) in `git log --oneline`
- Re-ran plan-level `<verify>` commands individually:
  - Task 1: `go test ./cmd -run '^TestRecruitmentCredit' -count=1` -- PASS
  - Task 2: `go test ./cmd -run '^TestAgency' -count=1 && ! grep -rqE 'awaits Phase [0-9]+' cmd/ && ! grep -rq 'SwarmPhase202Limitation' cmd/` -- PASS
  - Task 3: `go test ./cmd -run '^(TestDeliveredPlusPassingIsNotCredit|TestCreditRequiresBothFacts|TestHarmfulOutcomeIsReachable)$' -count=1` -- PASS
- Re-ran `go build ./...` and `go vet ./...` -- clean
- Re-ran the two named regression ratchets from this plan's warnings: `go test ./cmd -run 'TestColonyStateWriteAllowlistOnlyShrinks|TestNextActionNeverHardcoded' -count=1` -- PASS
- Re-ran `go test ./cmd -run 'TestPlatformParityGolden|TestRegressionSnapshot|TestOrphanAllowlistOnlyShrinks' -count=1` -- PASS; `TestNoRegisteredSubcommandIsUnreferenced` fails exactly as this plan's own warnings document (pre-existing, owned by 203-15) and was NOT allowlisted
- Re-ran acceptance-criteria negative proofs by breaking and reverting: `TestCreditRequiresBothFacts`'s own synthetic-fixture subtest proves the AST guard fires on a single-identifier write; `TestRecruitmentCreditOutcomeVocabulary` proves an undeclared outcome value is refused by name
