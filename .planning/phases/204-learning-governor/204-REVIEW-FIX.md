---
phase: 204-learning-governor
fixed_at: 2026-09-14T16:39:29Z
review_path: .planning/phases/204-learning-governor/204-REVIEW.md
iteration: 1
findings_in_scope: 5
fixed: 5
skipped: 0
status: all_fixed
---

# Phase 204: Code Review Fix Report

**Fixed at:** 2026-09-14T16:39:29Z
**Source review:** .planning/phases/204-learning-governor/204-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 5 (2 critical, 3 warning; fix_scope=critical_warning)
- Fixed: 5
- Skipped: 0

**Verification environment:** all fixes and tests ran directly in the main
checkout (branch `oracle-reinstate`) per this run's explicit instruction —
no isolated worktree was created for this pass. `go build ./...` and
`go vet ./cmd/... ./pkg/...` were run clean after every fix; targeted
`go test ./cmd -run '<names>'` was run per finding (never the full `cmd`
package, per this repo's 21-minute/12-shard TestMain warning). Every new
test was proven able to fail by running it against the pre-fix version of
its source file (via `git stash` on that one file) before restoring the fix
and re-running — recorded per finding below.

## Fixed Issues

### CR-01: Path traversal in the source-proposal file writer

**Files modified:** `cmd/source_proposal.go`, `cmd/source_proposal_test.go`
**Commit:** `9928e786`
**Applied fix:** `sourceProposalApplyChangeSet` now resolves the proposal
root to an absolute path and refuses, by name, any change-set file path that
is itself absolute, or that `filepath.Join`-resolves outside that root (a
`..`-escaping path). Matches the review's suggested fix essentially as
written, adapted to also refuse an absolute path explicitly (the original
code's `filepath.Join(root, relPath)` would otherwise silently accept an
absolute `relPath`, since `Join` special-cases absolute second arguments on
some inputs) — the reviewer's own text names this exact risk ("Anything a
worker or a wrapper can influence ... treat it as untrusted input").
**Proof:** Added `TestPathTraversalChangeSetIsRefusedByName` (a `../` escape)
and `TestAbsolutePathChangeSetIsRefusedByName` (an absolute path). Both
assert the refusal names the escape/absolute path, that `created=false`,
that no file lands outside the repo root, that no proposal branch is left
behind, and that the original branch is still checked out. Ran both against
the pre-fix `source_proposal.go` (via `git stash`): both failed — the escape
test failed because `git add` (not the application code) was the first thing
to refuse the outside-repo path, with a different error message than the
one this fix produces, and the absolute-path test failed with `git add`
similarly refusing after the file had already been written to disk via
`os.WriteFile` — confirming the write-before-refuse gap was real. Restored
the fix: both pass, plus the full existing `source_proposal_test.go` suite
(`TestProposalCreatesABranchAndNothingElse`, `TestDirtyTreeProposalIsRefusedByName`,
`TestBranchNameCollisionIsRefused`, `TestProposalReplayCreatesNoSecondBranch`, etc.).

### CR-02: `rollbackCanary` can revert an already-completed canary

**Files modified:** `cmd/rollback.go`, `cmd/rollback_test.go`
**Commit:** `a0a43f18`
**Applied fix:** `rollbackCanary` now re-reads the current stored canary run
(`loadCanaryRun`, not the caller's possibly-stale `run` argument) immediately
after the existing replay-receipt check, and refuses by name when its status
is already `completed` — leaving the stored run record, the durable
episode-ledger terminal result, and the (post-completion) working-tree state
untouched. Matches the review's suggested fix essentially as written.
**Proof:** Added `TestRollbackRefusesAnAlreadyCompletedCanary`: starts a
canary, completes it, applies a post-completion file change (simulating a
delayed re-evaluation), then calls `rollbackCanary` and asserts (a) an error
naming "already completed", (b) `credited=false`, (c) the post-completion
file change survives untouched (i.e. no restore happened), (d) the stored
run status is still `completed`, and (e) the episode ledger's terminal
result is still `"completed"`, not `"rolled_back"`. Ran against the pre-fix
`rollback.go` (via `git stash`): failed with "expected rollbackCanary to
refuse an already-completed canary, got nil error" — confirming the silent
revert was real. Restored the fix: passes, plus the full existing
`rollback_test.go` suite (`TestCompleteCanaryMarksTheChangeAsKept`,
`TestRegressionRestoresAndQuarantinesAtomically`,
`TestHoldoutRegressionAlsoRollsBack`, `TestRollbackReplayReturnsTheFirstReceipt`,
`TestCanaryEventsAreInlineAndDurable`, `TestQuarantineThresholdBoundaries`,
`TestNoAutomaticPathReleasesAQuarantine`, `TestReleaseCanaryQuarantineThroughApprovedPath`).

### WR-01: `instinct-apply --success` collapses "did not help" into "harmful"

**Files modified:** `cmd/internal_cmds.go`, `cmd/internal_cmds_test.go`
**Commit:** `80fc3164`
**Applied fix:** Added an explicit `--outcome helpful|neutral|harmful` flag
to `instinct-apply`. `--success` is kept as a backward-compatible alias
(`true` → `helpful`, the same as before), but `--success=false` now maps to
`neutral`, never `harmful` — an owner must pass `--outcome=harmful`
explicitly to record the stronger claim. Passing both `--success` and
`--outcome` (via `cmd.Flags().Changed(...)`, so this is detected even though
`--success` has a `true` default) is refused by name before any write. The
legacy `COLONY_STATE.json` fallback path (an instinct not yet migrated into
`instincts.json`) was updated to match: `neutral` increments neither
`Successes` nor `Failures` (that schema has no neutral counter), rather than
being forced into the `Failures` bucket. This is the shape the review's own
Fix section named as "the least surprising": a new explicit flag, backward
compatible, with the ambiguous case moved to `neutral` and the two flags
mutually exclusive.
**Proof:** Added four tests: `TestInstinctApplySuccessFalseRecordsNeutralNotHarmful`
(the core WR-01 assertion — `--success=false` now records `"neutral"`),
`TestInstinctApplySuccessTrueRecordsHelpful` (default/backward-compat
unchanged), `TestInstinctApplyOutcomeFlagRecordsHarmful` (the new explicit
path to the stronger claim), and `TestInstinctApplyRefusesSuccessAndOutcomeTogether`
(both flags together is refused, and writes nothing). Ran the first three
against the pre-fix `internal_cmds.go` (via `git stash`): the neutral test
failed with `--success=false recorded outcome "harmful", want "neutral"`
(exactly the defect), and the other two failed with `unknown flag: --outcome`
(the flag did not exist pre-fix) — confirming all three exercise genuinely
new behavior. Restored the fix: all four pass, plus the pre-existing
`TestInstinctApplyUpdatesStandaloneStore` (which only asserts on
`application_count`/history length, unaffected by the outcome-word change).
Checked `.claude/commands`, `.opencode/commands`, `.aether/commands`, and
`.aether/workers.md` for any documented `instinct-apply` usage that could be
broken by the new flag — found none, so `TestCLIFlagAudit`'s markdown-corpus
check is unaffected (confirmed green by direct run).

### WR-02: Episode ledger has no guard against a second `episode_closed` record

**Files modified:** `cmd/episode_ledger.go`, `cmd/episode_ledger_test.go`
**Commit:** `8462b72e`
**Applied fix:** `recordEpisodeOutcome` now refuses, by name, a SECOND,
genuinely different `episode_closed` write for an episode that already has
one — checked after the existing identical-digest replay check, so an exact
replay of the same close (differing only in an excluded-from-identity field
like `EndedAt`) still collapses silently as before; only a close carrying a
different payload (e.g. a different `TerminalResult`) for an already-closed
episode is now refused. Matches the review's suggested fix in effect, placed
after the existing recordID/replay lookup (rather than before, as the
review's snippet showed) specifically so it does not fire on identical
replays — which the review's own text requires ("an identical replay of the
identical close still collapses silently via the existing digest path; only
a DIFFERENT close is refused").
**Proof:** Added `TestEpisodeSecondDifferentCloseIsRefusedByName`: opens an
episode, closes it once with `TerminalResult: "completed"`, then attempts a
second close with `TerminalResult: "failed"` and asserts (a) an error naming
"already closed", (b) `credited=false`, (c) the FIRST close's terminal
result (`"completed"`) survives untouched and is still returned by
`episodeLedgerTerminalRecord`, and (d) exactly one `episode_closed` record
exists for the episode. Ran against the pre-fix `episode_ledger.go` (via
`git stash`): failed with "expected an error on a second, different close
for an already-closed episode" — confirming the silent second-record
append was real. Restored the fix: passes, plus the full existing
`episode_ledger_test.go` suite, notably `TestEqualDigestsCollapseAndDifferentTimestampsDoNot`
(proves the identical-replay-still-collapses distinction this fix must
preserve), `TestEpisodeLedgerReplayWritesNothing`, and
`TestEpisodeLedgerHasOneWriter`. Also re-ran the canary/rollback and
shadow-compare tests that write through this same function
(`TestRegressionRestoresAndQuarantinesAtomically`, `TestCompleteCanaryMarksTheChangeAsKept`,
`TestComparisonResultReachesTheDurableLedger`, `TestComparisonReplayWritesNothing`) —
all still pass, since CR-02's fix already prevents `rollbackCanary` from
reaching a second close on an already-completed run in practice, and this
guard is defense-in-depth for any other caller.

### WR-03: Credit ledger crosses every delivered instinct against every decision delta

**Files modified:** `cmd/application_evidence.go`, `cmd/application_evidence_test.go`
**Commit:** `3ce57ce4`
**Applied fix:** `recordPhaseApplicationCredit` now writes exactly ONE
credit record per delivered contribution per phase, against a combined
decision identifier that joins every decision-kind knowledge delta the
phase's latest attempt recorded (`strings.Join(decisionIDs, ",")`), instead
of the previous nested loop that wrote one record per (contribution,
decision) pair. For the common single-decision case this combined
identifier is byte-identical to the single decision id used before this fix
(`strings.Join` of a one-element slice returns that element unchanged), so
`recruitmentCreditRecordID`'s existing `(contributionID, changedDecisionID)`
identity and every reader keyed on it are unaffected for that case — which
is exactly why the four named tests below needed no behavioral change to
stay green. **Design-decision check performed first, as instructed:** I
read `TestPhaseApplicationCreditTracerEndToEnd`, `TestPhaseApplicationCreditIsReachedFromBothCheckLanes`,
`TestPhaseApplicationCreditIsReplaySafe`, and `TestTunerSeesGenuinelyProducedCredit`
before making any change. All four exercise `newApplicationCreditFixture`,
which always delivers exactly one instinct against exactly one decision
delta — none of them assert on, or depend on, the cross-product shape, so
none of them "encode the cross product as intended behaviour." The fix was
therefore applied directly rather than being deferred as a design question.
**Proof:** Ran all four required tests against the fix — all pass unchanged.
Added a new test, `TestPhaseApplicationCreditRecordsOncePerContributionNotPerDecision`,
built from the same real production writers (`commitTestBuildStart`,
`attachBuildFreeCheckReport`, `attachBuildKnowledgeDeltas`,
`promoteRealInstinct`, `recordInstinctDeliveries`) with 2 delivered
instincts and 2 decision-kind deltas on one attempt, asserting
`summary.Recorded == 2` (not 4), exactly 2 credit records total, and that
each surviving record's `ChangedDecisionID` names both decisions (the
combined id). Ran this new test against the pre-fix `application_evidence.go`
(via `git stash`): failed with `summary.Recorded = 4, want 2 ... {Recorded:4
Helpful:4}` — the exact 2×2 cross product the review described. Restored
the fix: passes, plus the four required tests, the full
`application_evidence_test.go` suite (guidance-state, corroboration,
promotion tests — none of which touch `recordPhaseApplicationCredit`'s
internals), and `TestCreditRequiresBothFacts`/`TestAgencyEvidence*` in the
sibling `recruitment_credit`/`agency_contract` files (which read credit
records through `recruitmentCreditForDecision`/`recruitmentCreditForContribution`
on an entirely separate decision-ID namespace — trophallaxis packet
decisions — unaffected by this change).

## Skipped Issues

None — all 5 in-scope findings were fixed.

---

_Fixed: 2026-09-14T16:39:29Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
