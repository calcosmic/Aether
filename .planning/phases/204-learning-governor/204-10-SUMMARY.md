---
phase: 204-learning-governor
plan: "10"
subsystem: learning
tags: [improvement-report, episode-ledger, hard-gates, source-proposal, propose-only-boundary, LEARN-08]

# Dependency graph
requires:
  - phase: 204-01
    provides: "204-CLASSIC-SYNTHESIS.md's LEARN-08 mechanism rows -- two separate honest figures, non-gameable hard failures, propose-only source boundary"
  - phase: 204-04
    provides: "cmd/episode_ledger.go's durable append-only episode/outcome ledger (recordEpisodeOutcome, readEpisodeLedger, episodeLedgerTerminalRecord, episodeLedgerEpisodeIDs) this plan's report reads its two figures from"
  - phase: 204-07
    provides: "cmd/eval_gates.go's evalGateSentinel type and cmd/testdata/eval-gates/sentinels.json, the sentinel list this plan's hard-failure list enriches its consequence text from"
provides:
  - "cmd/improvement_report.go: buildImprovementReport/renderImprovementReport -- two independent count-and-total figures (VerifiedUsefulSuccess, PreventableInterventions) with no combined score field, ever, and a hard-failure list assembled by a function with no parameter capable of suppressing an entry"
  - "cmd/source_proposal.go: proposeSourceImprovement -- opens one isolated branch, applies a change set, records a durable proposal, and does nothing else; recordIndependentVerification -- the ONLY function that ever marks a proposal verified, refusing a verifier identity equal to the proposal's own creator"
  - "sourceProposalForbiddenOperations -- the named, source-parsed list cmd/source_proposal_test.go's TestSourceProposalCannotMergePublishOrDeploy reads to prove, by walking the real call graph across cmd/ and pkg/, that nothing reachable from a proposal can merge, push, publish or deploy"
affects: [204-11]

# Actuals (#2632) -- chars/4 over the realized diff (942fbffd..6b2e5224), never a harness token count.
actuals:
  tokens: 20496
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Two-figure, no-combined-score report type enforced structurally: TestTwoFiguresAreNeverCombined reflects over improvementReport's field list against an explicit allow-list, so a future field cannot silently reintroduce a blended score"
    - "Cross-multiplied-integer threshold comparison (never a rounded Percent()) -- every classification compares count*thresholdTotal against thresholdCount*total, proven never violated by an AST scan (TestPercentagesAreNeverCompared) that also proves itself non-vacuous against a synthetic violation"
    - "Package-directory-scoped call-graph resolution for a structural reachability check: an unqualified call resolves within the calling function's own package directory, a qualified call resolves through the calling file's own import-alias table -- never bare-name-only, which this plan's own development caught producing a real false positive (exec.Cmd.Run() colliding with an unrelated pkg/memory function also named Run)"
    - "Propose-only branch boundary: proposeSourceImprovement never touches the branch it doesn't create, restores the original branch on both success and failure, and records verification state through a single, self-verification-refusing gate function -- mirroring cmd/recruitment_credit.go's one-writer, deterministic-identity, replay-safe discipline"

key-files:
  created:
    - cmd/improvement_report.go
    - cmd/improvement_report_test.go
    - cmd/source_proposal.go
    - cmd/source_proposal_test.go
  modified: []

key-decisions:
  - "isVerifiedUsefulSuccess treats an EMPTY HardGateResults map as vacuously passing (the plan's own literal wording: 'whose hard-gate results are all passing'), not as a missing-evidence failure. Combined with WINDOWS.md entry 38 (no production lane populates HardGateResults, EvidenceIDs or Interventions today), this means a real production episode is classified as verified-success only once EvidenceIDs is populated too -- the report is honestly unclassified for every real episode until that wiring lands, never falsely perfect."
  - "The plan's action text names 'the categories come from the intervention vocabulary the ledger already declares,' but cmd/episode_ledger.go declares no such closed vocabulary -- episodeLedgerRecord.Interventions is a free-form []string written by emitColonyLiveInterventionRecorded. collectPreventableInterventions treats each non-empty string in that field as one category, without inventing a fabricated closed vocabulary the runtime does not actually enforce."
  - "sourceProposal has no explicit 'creator' field per the plan's own field list. recordIndependentVerification refuses self-verification by comparing verifierID against the proposal's own CandidateID -- the identity of what proposed it -- rather than adding an unlisted field."
  - "Task 3's action text describes building the call graph 'reachable from proposeSourceImprovement and from the command that invokes it,' but no cobra command exists yet (Task 2's own files_modified never included one). The structural check's entry points are proposeSourceImprovement and recordIndependentVerification -- the whole public surface this plan's file exposes -- documented inline as the only sense in which 'the command that invokes it' currently exists."
  - "cmd/promotion_gate.go, named in Task 1's read_first list, does not exist in this codebase (closest analog: cmd/oracle_promote.go). Read what exists instead; this did not block any acceptance criterion."
  - "sourceProposalForbiddenOperations' doc comment cites .planning/WINDOWS.md entry 18 (the recruitment delegation-depth check trusting a short, fixed list of coordinator names on their own word alone) as the closest existing defect-register entry recording the same class of gap Assumption V names for the execution sandbox -- no entry describing the sandbox fact itself existed to cite, and fabricating one would have been worse than naming the real, closely analogous entry that does exist."

patterns-established:
  - "A structural reachability check over a large, multi-package Go tree must resolve calls by package-directory scope, not bare function name -- a lesson this plan's own TestSourceProposalCannotMergePublishOrDeploy proved by first producing, then eliminating, a real false positive from an unrelated same-named method."

requirements-completed: [LEARN-08]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "The improvement report's two independent figures (VerifiedUsefulSuccess, PreventableInterventions) with no combined score field, ever, and exact-arithmetic threshold classification (cross-multiplied integers, never a compared percentage)."
    requirement: "LEARN-08"
    verification:
      - kind: unit
        ref: "cmd/improvement_report_test.go#TestTwoFiguresAreNeverCombined"
        status: pass
      - kind: unit
        ref: "cmd/improvement_report_test.go#TestThresholdBoundaries"
        status: pass
      - kind: unit
        ref: "cmd/improvement_report_test.go#TestEmptyWindowReportsZeroNotPerfect"
        status: pass
      - kind: unit
        ref: "cmd/improvement_report_test.go#TestReportOrderingIsTotalAndStable"
        status: pass
      - kind: unit
        ref: "cmd/improvement_report_test.go#TestOneEpisodeCanCountInBothFigures"
        status: pass
      - kind: unit
        ref: "cmd/improvement_report_test.go#TestPercentagesAreNeverCompared"
        status: pass
      - kind: unit
        ref: "cmd/improvement_report_test.go#TestPercentRoundingIsHalfToEven"
        status: pass
    human_judgment: false
  - id: D2
    description: "Hard-gate failures are read from the ledger's own recorded results and never recomputed, excluded, downgraded or offset by a success recorded elsewhere in the same window."
    requirement: "LEARN-08"
    verification:
      - kind: unit
        ref: "cmd/improvement_report_test.go#TestHardFailureCannotBeExcludedOrOffset"
        status: pass
    human_judgment: false
  - id: D3
    description: "proposeSourceImprovement opens one isolated branch, applies a change set, and records a durable proposal -- refusing a dirty tree and a branch-name collision by name, replaying safely on identical input -- and recordIndependentVerification is the only function that ever marks a proposal verified, refusing self-verification by the proposal's own creator."
    requirement: "LEARN-08"
    verification:
      - kind: unit
        ref: "cmd/source_proposal_test.go#TestProposalCreatesABranchAndNothingElse"
        status: pass
      - kind: unit
        ref: "cmd/source_proposal_test.go#TestDirtyTreeProposalIsRefusedByName"
        status: pass
      - kind: unit
        ref: "cmd/source_proposal_test.go#TestBranchNameCollisionIsRefused"
        status: pass
      - kind: unit
        ref: "cmd/source_proposal_test.go#TestProposalReplayCreatesNoSecondBranch"
        status: pass
      - kind: unit
        ref: "cmd/source_proposal_test.go#TestUnverifiedProposalIsReportedNotReady"
        status: pass
      - kind: unit
        ref: "cmd/source_proposal_test.go#TestProposalCannotVerifyItself"
        status: pass
      - kind: unit
        ref: "cmd/source_proposal_test.go#TestProposalRecordNamesItsCandidateAndEvidence"
        status: pass
    human_judgment: false
  - id: D4
    description: "A structural, source-parsed proof that no function reachable from proposing a source improvement can merge, push, publish or deploy it -- covering direct calls and subprocess-literal arguments, proven non-vacuous by four synthetic fixtures, one per forbidden family."
    requirement: "LEARN-08"
    verification:
      - kind: unit
        ref: "cmd/source_proposal_test.go#TestSourceProposalCannotMergePublishOrDeploy"
        status: pass
      - kind: unit
        ref: "cmd/source_proposal_test.go#TestForbiddenOperationListIsReadFromTheSource"
        status: pass
    human_judgment: false

# Metrics
duration: ~50min
completed: 2026-09-14
status: complete
---

# Phase 204 Plan 10: Learning Governor -- Honest Improvement Report and Propose-Only Source Boundary Summary

**Two figures the system can never blend into one score (verified-useful-success, preventable-owner-intervention), with hard-gate failures the code structurally cannot exclude or offset -- plus a propose-only source-change boundary proven, by walking the real call graph, to have no route to merge, push, publish or deploy anything it suggests.**

## Performance
- **Duration:** ~50 min (exact start time not captured; estimated from the plan's own scope and the sibling wave-3 plans' recorded durations)
- **Tasks:** 3/3 completed
- **Files created:** 4 (cmd/improvement_report.go, cmd/improvement_report_test.go, cmd/source_proposal.go, cmd/source_proposal_test.go)
- **Files modified:** 0

## Accomplishments
- `cmd/improvement_report.go`'s `improvementReport` declares exactly two independent count-and-total figures -- `VerifiedUsefulSuccess` and `PreventableInterventions` -- and no third, combined field; `TestTwoFiguresAreNeverCombined` reflects over the struct's own field list against an explicit allow-list, so a future addition cannot silently reintroduce a blended score.
- `reportRatio.Percent()` implements half-to-even rounding explicitly (never Go's default `%.1f` half-away-from-zero behaviour), and `TestPercentagesAreNeverCompared` is an AST scan proving no comparison operator in the file is ever applied to a `Percent()` result -- proven non-vacuous against a synthetic violation fixture.
- `assembleHardFailureList` takes only the recorded ledger results and the sentinel list; it has no parameter, flag or branch capable of excluding, downgrading or offsetting an entry. `TestHardFailureCannotBeExcludedOrOffset` drives nine verified successes against one hard failure and proves the failure still appears.
- `cmd/source_proposal.go`'s `proposeSourceImprovement` opens one isolated branch from the current HEAD, applies a change set, restores the original branch on both success and failure, and records a durable, replay-safe proposal -- refusing a dirty working tree and a branch-name collision by name, and returning the first proposal (creating no second branch) on an identical replay.
- `recordIndependentVerification` is the only function in the file that ever sets a proposal's verification state to `verified`, and refuses a verifier identity equal to the proposal's own `CandidateID` -- the same contribution cannot both propose a change and independently verify it.
- `TestSourceProposalCannotMergePublishOrDeploy` parses every non-test file in `cmd/` and `pkg/`, builds the real call graph reachable from `proposeSourceImprovement` and `recordIndependentVerification`, and fails naming any reachable direct call or subprocess-literal argument invoking a merge, push, publish or deploy operation. Four synthetic fixtures (one per family, including one wrapped in a subprocess construction) prove the detector is not vacuous. `TestForbiddenOperationListIsReadFromTheSource` proves the checker's own list is parsed from `sourceProposalForbiddenOperations`' source declaration, never hand-typed, so the two cannot drift.
- Building the real, whole-tree call-graph check live during development caught a genuine false positive: bare function-name matching resolved `exec.Cmd.Run()` (called inside `sourceProposalBranchExists`) to an unrelated `Run` function in `pkg/memory/consolidate.go` that happens to share the name and itself calls something named `Publish`. The check was redesigned to resolve calls by package-directory scope (an unqualified call resolves within the calling function's own directory; a qualified call resolves through the calling file's own import-alias table) before expanding into a function's body -- eliminating the false positive while still checking every call site, resolvable or not, for a direct name match.

## Task Commits
Each task was committed atomically:
1. **Task 1: Report the two figures separately, with exact arithmetic** - `2330a278` (feat)
2. **Task 2: Let a source improvement be proposed, and only proposed** - `5a2867ce` (feat)
3. **Task 3: Prove there is no path from proposing a change to accepting one** - `6b2e5224` (test)

**Plan metadata:** committed alongside this SUMMARY (worktree mode -- STATE.md/ROADMAP.md excluded, handled by the orchestrator).

_TDD note: all three tasks were `tdd="true"`. As with 204-05/204-07's own precedent, each task's declarative/structural shape (a report type plus its pure builder, a propose-only writer with real-git-repo fixtures, a source-parsed AST reachability check) meant the genuine RED state was a synthetic broken fixture rather than a naturally red first commit, so each task was committed as a single `feat`/`test` carrying implementation and its tests together. Every test's fail-capability was proven live during development -- most concretely by Task 3's real false-positive-then-fix cycle described above._

## Files Created/Modified
- `cmd/improvement_report.go` - `reportWindow`, `reportRatio`/`ratioOf`/`Percent`/`roundHalfToEven`, `improvementReportSuccessClassification`/`improvementReportInterventionClassification` with named threshold constants, `improvementReportHardFailure`/`assembleHardFailureList`, `isVerifiedUsefulSuccess`, `improvementReportInterventionEntry`/`collectPreventableInterventions`, `improvementReportRow`, `improvementReport`, `recordsInWindow`, `buildImprovementReport`, `renderImprovementReport`
- `cmd/improvement_report_test.go` - all 8 named tests from the plan's artifacts list, plus the AST scan helpers `TestPercentagesAreNeverCompared` needs
- `cmd/source_proposal.go` - `sourceProposalVerificationState` vocabulary, `sourceChangeSetFile`/`sourceChangeSet`, `sourceProposal`/`sourceProposalFile`, `sourceProposalID`, `sourceProposalBranchName`, `writeSourceProposal`/`readSourceProposal`, `sourceProposalWorkingTreeIsDirty`, `sourceProposalBranchExists`, `sourceProposalApplyChangeSet`, `proposeSourceImprovement`, `recordIndependentVerification`, `renderSourceProposalAnnouncement`/`renderSourceProposalReadiness`, `sourceProposalForbiddenOperations`
- `cmd/source_proposal_test.go` - all 7 named Task 2 tests, plus `TestSourceProposalCannotMergePublishOrDeploy` and `TestForbiddenOperationListIsReadFromTheSource` (Task 3), plus the shared call-graph index/walk helpers both depend on

## Decisions Made
See `key-decisions` in the frontmatter for the full account: the vacuous-empty-HardGateResults reading (and its honest interaction with WINDOWS.md entry 38), the free-form (not closed-vocabulary) intervention categories, the self-verification refusal via `CandidateID` rather than an unlisted field, the two-function entry-point set standing in for "the command that invokes it," the nonexistent `cmd/promotion_gate.go` read_first target, and the WINDOWS.md entry-18 citation for the sandbox-as-second-layer doc comment.

## Deviations from Plan

### Judgment calls (no bug, no missing critical functionality -- plan ambiguity resolved and documented)

**1. [Judgment call] No closed "intervention vocabulary" exists in the ledger for the plan's own wording to point to**
- **Found during:** Task 1, designing `collectPreventableInterventions`
- **Issue:** The plan's action text says "the categories come from the intervention vocabulary the ledger already declares," but `cmd/episode_ledger.go`'s `Interventions []string` field is free-form text written by `emitColonyLiveInterventionRecorded`, with no declared closed vocabulary anywhere in the codebase.
- **Fix:** Treated each non-empty string in `Interventions` as one category, rather than inventing a closed vocabulary the runtime does not enforce.
- **Files modified:** `cmd/improvement_report.go`
- **Verification:** `TestOneEpisodeCanCountInBothFigures` exercises this path against a real ledger fixture.
- **Committed in:** `2330a278`

**2. [Judgment call] `cmd/promotion_gate.go`, named in Task 1's `read_first`, does not exist**
- **Found during:** Task 1, initial required-reading pass
- **Issue:** The file is not present anywhere in this checkout; the closest existing analog is `cmd/oracle_promote.go`.
- **Fix:** Read what exists instead and proceeded; this did not block any acceptance criterion, since none of Task 1's checkable criteria reference this file's content directly.
- **Files modified:** none
- **Verification:** n/a (a reading-list gap, not a code defect)
- **Committed in:** n/a

**3. [Judgment call] Task 3's own text assumes a CLI command exists that invokes `proposeSourceImprovement`; none does**
- **Found during:** Task 3, designing the structural check's entry points
- **Issue:** "builds the call graph reachable from `proposeSourceImprovement` and from the command that invokes it" presumes a wired cobra command. Task 2's `files_modified` never included one, and none was added.
- **Fix:** The structural check's entry points are `proposeSourceImprovement` and `recordIndependentVerification` -- the whole public surface this plan's file exposes -- documented inline in the test file as the only sense in which "the command that invokes it" currently exists. A later plan wiring a CLI command should add it to `sourceProposalReachabilityEntryPoints`.
- **Files modified:** `cmd/source_proposal_test.go`
- **Verification:** `TestSourceProposalCannotMergePublishOrDeploy` passes with this entry-point set.
- **Committed in:** `6b2e5224`

### Auto-fixed Issues

**4. [Rule 1 - Bug] Bare-name call-graph resolution produced a real false positive**
- **Found during:** Task 3, first run of `TestSourceProposalCannotMergePublishOrDeploy` against the real `cmd/`+`pkg/` tree
- **Issue:** Resolving a called function purely by its bare name (ignoring which package it is declared in) caused the walk to follow `sourceProposalBranchExists`'s own `cmd.Run()` (a call on a local `*exec.Cmd` value) into an unrelated function literally named `Run` in `pkg/memory/consolidate.go`, which in turn called something named `publishConsolidationEvent` -> `.Publish()` -- a genuine false positive from name collision, not a real reachable path.
- **Fix:** Redesigned the index and walk to resolve calls by package-directory scope: an unqualified call resolves only within the calling function's own directory, and a qualified call resolves only through the calling file's own import-alias table (mapped against this module's own `go.mod` path) into another indexed directory. Every call site, resolvable or not, is still checked for a direct forbidden-name match; only the decision to *expand into a function's body* now requires a real resolved declaration.
- **Files modified:** `cmd/source_proposal_test.go`
- **Verification:** `TestSourceProposalCannotMergePublishOrDeploy` passes clean against the real tree with this fix; the four synthetic fixtures (including the `publish` fixture, which is itself a same-named-method call) still all catch their violation, proving the fix did not weaken direct-match detection.
- **Committed in:** `6b2e5224`

---
**Total deviations:** 3 judgment calls (plan-text ambiguity resolved and documented) + 1 auto-fixed bug (Rule 1, the false-positive call-graph resolution). **Impact on plan:** Every machine-checkable `<acceptance_criteria>` line and every task's `<verify>` command passes. No deviation weakened any guarantee; the Rule 1 fix strengthened the structural proof's precision.

## Issues Encountered
None beyond the deviations documented above. `go build ./...`, `go vet ./cmd/... ./pkg/...` are clean; `gofmt -l` reports nothing for any created file.

## User Setup Required
None -- no external service configuration required.

## Threat Flags

| Flag | File | Description |
|------|------|--------------|
| threat_flag: unwired-capability | cmd/source_proposal.go | `proposeSourceImprovement` writes files and creates git branches from within the running process; no cobra command currently calls it, so it carries no reachable attack surface today. When a future plan wires a CLI entry point (or an automated caller) to this function, that wiring is the point requiring a security review -- who can trigger it, with what candidate/evidence/change-set input, and under what admission check -- not this file's own logic, which is already structurally proven to reach no merge/push/publish/deploy path. |

## Next Phase Readiness
- `buildImprovementReport`/`renderImprovementReport` and `proposeSourceImprovement`/`recordIndependentVerification` are now established, tested symbols. **Honest limitation, stated plainly:** per `.planning/WINDOWS.md` entry 38, no production lifecycle lane yet populates `EvidenceIDs`, `HardGateResults` or `Interventions` on a real episode -- the boundary callers that would (`emitColonyLiveOutcomeRecorded`, `emitColonyLiveInterventionRecorded`) exist but have no caller. Until that gap closes, a real window of production episodes will report every episode as unclassified (zero verified successes, zero preventable interventions, all unclassified), honestly, rather than a fabricated figure -- this report's correctness is proven against real ledger fixtures written through `recordEpisodeOutcome`, but its usefulness on a live colony is gated on the same wiring gap 204-04's own SUMMARY already named. Closing that gap is exactly the "production wiring" work WINDOWS #38 describes, not something this plan could close within its own declared file scope.
- No CLI command yet calls `proposeSourceImprovement`. A future plan wiring one should extend `sourceProposalReachabilityEntryPoints` in `cmd/source_proposal_test.go` and re-run `TestSourceProposalCannotMergePublishOrDeploy` to keep the structural proof covering the new entry point.
- No blockers.

## Self-Check: PASSED

- All 4 key files confirmed present via `git ls-files` (4 created, 0 modified).
- All 3 task commits (`2330a278`, `5a2867ce`, `6b2e5224`) confirmed present via `git log --oneline`.
- Acceptance-criteria greps re-run clean: `func renderImprovementReport(` in `cmd/improvement_report.go`; `func proposeSourceImprovement(`, `func recordIndependentVerification(`, `sourceProposalForbiddenOperations` all in `cmd/source_proposal.go`.
- Full named test set for all three tasks passes together: `go test ./cmd -run '^(TestTwoFiguresAreNeverCombined|TestThresholdBoundaries|TestEmptyWindowReportsZeroNotPerfect|TestReportOrderingIsTotalAndStable|TestOneEpisodeCanCountInBothFigures|TestHardFailureCannotBeExcludedOrOffset|TestPercentagesAreNeverCompared|TestPercentRoundingIsHalfToEven|TestProposalCreatesABranchAndNothingElse|TestDirtyTreeProposalIsRefusedByName|TestBranchNameCollisionIsRefused|TestProposalReplayCreatesNoSecondBranch|TestUnverifiedProposalIsReportedNotReady|TestProposalCannotVerifyItself|TestProposalRecordNamesItsCandidateAndEvidence|TestSourceProposalCannotMergePublishOrDeploy|TestForbiddenOperationListIsReadFromTheSource|TestVoicedScreensSpeakPlainEnglish|TestVoicedScreensCarryNoRawStateToken)$' -count=1 -timeout 5m` -> `ok`.
- `go build ./...` and `go vet ./cmd/... ./pkg/...` both clean; `gofmt -l` reports nothing for any file this plan created.
- `LEARN-08` confirmed as this plan's sole owner in this phase and ready to mark complete in `REQUIREMENTS.md` (the orchestrator owns that write in worktree mode).

---
*Phase: 204-learning-governor*
*Completed: 2026-09-14*
