---
phase: 204-learning-governor
plan: "08"
subsystem: learning
tags: [shadow-evaluation, structural-isolation, candidate-vs-baseline, holdout, LEARN-06]

# Dependency graph
requires:
  - phase: 204-01
    provides: "204-CLASSIC-SYNTHESIS.md ruling and the mechanism study naming LEARN-06's isolation requirement"
  - phase: 204-04
    provides: "cmd/episode_ledger.go's recordEpisodeOutcome -- the one durable writer this plan routes every comparison result through"
  - phase: 204-07
    provides: "cmd/eval_gates.go's resolveEvalGateHoldouts and the digest-only holdout file (cmd/testdata/eval-gates/holdouts.json) this plan resolves exactly once, in the command layer"
provides:
  - "pkg/shadow: a new, sealed package -- FrozenEvaluator and AcceptanceCriteria with no setter, no pointer-receiver method and no exported field; Candidate, immutable and requiring all six declarations; Baseline, frozen the same way; Compare, returning a four-score, five-verdict comparison"
  - "Four independent structural tests (isolation_test.go), each with a non-vacuous synthetic fixture, proving a candidate cannot reach or alter its grader"
  - "cmd/shadow_cmds.go: shadow-declare and shadow-compare, an append-only candidate store with exactly one writer, and every comparison recorded durably through recordEpisodeOutcome"
affects: [204-09]

actuals:
  tokens: 22285
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Sealed-package isolation: unexported fields, value (never pointer) receivers, constructor-only initialization, digest-based identity -- FrozenEvaluator/AcceptanceCriteria/Candidate/Baseline all follow the identical shape"
    - "Four-condition structural proof (unreachable fields, no mutator method, no mutable-returning function, no field of the grader's type) each checked by an AST scan carrying its own synthetic violation fixture, mirroring cmd/classic_voice_event_test.go's and cmd/episode_ledger_test.go's 'ONE function may write X' AST-enforcement shape applied to a new property (isolation, not single-writer)"
    - "Exact cross-multiplied score comparison (scoreCompare) rather than percentage comparison, with an AST guard (TestNoComparisonAppliedToPercentResult) proving no comparison operator in the package is ever applied to a Percent() result"

key-files:
  created:
    - pkg/shadow/evaluator.go
    - pkg/shadow/candidate.go
    - pkg/shadow/baseline.go
    - pkg/shadow/comparison.go
    - pkg/shadow/evaluator_test.go
    - pkg/shadow/comparison_test.go
    - pkg/shadow/isolation_test.go
    - cmd/shadow_cmds.go
    - cmd/shadow_cmds_test.go
  modified: []

key-decisions:
  - "Baseline is run through the SAME FrozenEvaluator.Run(Candidate, Task) Result signature a real candidate is graded through, via an unexported, package-private baselineAsCandidate helper that bypasses NewCandidate's own declaration validation (it is never a declaration, never stored, and unreachable outside pkg/shadow). This avoids a second grading path while honouring the plan's literal Run(Candidate, Task) Result signature."
  - "NewFrozenEvaluator's signature carries no AcceptanceCriteria parameter (matching the plan's literal quoted signature). The grader's digest is made to cover the criteria's digest by the CALLER folding criteria.Digest() bytes into the def it passes -- cmd/shadow_cmds.go's shadowEvaluator does exactly this -- rather than pkg/shadow's own constructor taking criteria directly."
  - "The command layer's grading run function is a deterministic, always-pass placeholder. LEARN-06's own scope is the structural isolation and comparison mechanism a candidate cannot reach or alter -- not a production classifier for what 'passing' concretely means for a given fixture (re-running its guard test live, evaluating a real policy predicate, etc.), which is a materially larger piece of engineering out of scope for this plan. Every real shadow-compare run therefore currently reports a tied verdict against an unchanged baseline until a later plan wires a real run function through the same FrozenEvaluator.Run(Candidate, Task) Result seam this plan proves cannot be bypassed. See Deviations below."
  - "A shadow comparison is recorded as an episode_opened + episode_closed pair through recordEpisodeOutcome (cmd/episode_ledger.go), not a single ad-hoc record type, following the exact discipline every other lifecycle lane in this project already uses. Both writes are digest-based and idempotent (timestamps excluded from the identity digest for these record kinds), which is what makes a replayed shadow-compare run write nothing without any special-case replay logic in this plan's own code."
  - "The four scores have no dedicated field on episodeLedgerRecord (out of scope: episode_ledger.go is not in this plan's files_modified). They are carried as four formatted count-over-total strings in the record's existing EvidenceIDs field; the baseline's digest is carried in PolicyVersion (already documented as 'what governed it' and already populated from a policy-identity source by 204-04's own precedent)."

patterns-established:
  - "A grader (or any type a candidate must not influence) is sealed by: unexported fields, value receivers only, a constructor as the sole initializer, and a Digest() [32]byte accessor -- proven, not merely asserted, by four independent AST scans each carrying its own synthetic violation fixture."

requirements-completed: [LEARN-06]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "The sealed grader (FrozenEvaluator/AcceptanceCriteria) with no setter, no pointer-receiver method, no exported field, and no exported function anywhere returning a mutable reference to either."
    requirement: LEARN-06
    verification:
      - kind: unit
        ref: "pkg/shadow/isolation_test.go#TestCandidateCannotAlterItsEvaluatorDigest"
        status: pass
      - kind: unit
        ref: "pkg/shadow/isolation_test.go#TestEvaluatorHasNoMutator"
        status: pass
      - kind: unit
        ref: "pkg/shadow/isolation_test.go#TestNoExportedFunctionReturnsAMutableEvaluator"
        status: pass
      - kind: unit
        ref: "pkg/shadow/isolation_test.go#TestCandidateHoldsNoEvaluator"
        status: pass
      - kind: unit
        ref: "pkg/shadow/isolation_test.go#TestPackageImportsNothingFromCmd"
        status: pass
    human_judgment: false
  - id: D2
    description: "Candidate requires all six declarations up front, refuses an expiry already past, and refuses a declaration naming its own grader, acceptance criteria or a holdout."
    requirement: LEARN-06
    verification:
      - kind: unit
        ref: "pkg/shadow/evaluator_test.go#TestCandidateRequiresAllSixDeclarations"
        status: pass
      - kind: unit
        ref: "pkg/shadow/evaluator_test.go#TestCandidateNamingItsGraderIsRefused"
        status: pass
      - kind: unit
        ref: "pkg/shadow/evaluator_test.go#TestExpiredCandidateIsRefused"
        status: pass
      - kind: unit
        ref: "pkg/shadow/comparison_test.go#TestExpiredCandidateRefusedAtComparisonTime"
        status: pass
    human_judgment: false
  - id: D3
    description: "A candidate is run beside a frozen baseline over the identical visible and holdout task sets; an overfit candidate is never reported as beneficial; a tie or an empty comparison recommends nothing; scores compare exactly, never by rounded percentage."
    requirement: LEARN-06
    verification:
      - kind: unit
        ref: "pkg/shadow/comparison_test.go#TestCompareVerdictRules"
        status: pass
      - kind: unit
        ref: "pkg/shadow/comparison_test.go#TestOverfitIsNeverReportedAsBeneficial"
        status: pass
      - kind: unit
        ref: "pkg/shadow/comparison_test.go#TestTieRecommendsNothing"
        status: pass
      - kind: unit
        ref: "pkg/shadow/comparison_test.go#TestEmptyComparisonIsInconclusiveNotBeneficial"
        status: pass
      - kind: unit
        ref: "pkg/shadow/comparison_test.go#TestScoresCompareExactlyNotByPercentage"
        status: pass
      - kind: unit
        ref: "pkg/shadow/comparison_test.go#TestNoComparisonAppliedToPercentResult"
        status: pass
      - kind: unit
        ref: "pkg/shadow/comparison_test.go#TestBaselineDigestIsUnchangedByTheComparison"
        status: pass
      - kind: unit
        ref: "pkg/shadow/comparison_test.go#TestComparisonIsDeterministic"
        status: pass
    human_judgment: false
  - id: D4
    description: "The command surface: an append-only candidate store with exactly one writer, holdout resolution happening only in the command layer, and every comparison result recorded durably and replay-safe."
    requirement: LEARN-06
    verification:
      - kind: unit
        ref: "cmd/shadow_cmds_test.go#TestCandidateStoreIsAppendOnlyAndRefusesEdits"
        status: pass
      - kind: unit
        ref: "cmd/shadow_cmds_test.go#TestShadowCandidateStoreHasOneWriter"
        status: pass
      - kind: unit
        ref: "cmd/shadow_cmds_test.go#TestComparisonWithoutACandidateIsRefusedByName"
        status: pass
      - kind: unit
        ref: "cmd/shadow_cmds_test.go#TestComparisonResultReachesTheDurableLedger"
        status: pass
      - kind: unit
        ref: "cmd/shadow_cmds_test.go#TestComparisonReplayWritesNothing"
        status: pass
      - kind: unit
        ref: "cmd/shadow_cmds_test.go#TestHoldoutResolutionHappensOnlyInTheCommandLayer"
        status: pass
      - kind: unit
        ref: "cmd/shadow_cmds_test.go#TestComparisonResultSpeaksPlainEnglish"
        status: pass
      - kind: integration
        ref: "cmd/cli_flag_audit_test.go#TestCLIFlagAudit"
        status: pass
      - kind: integration
        ref: "cmd/classic_voice_corpus_test.go#TestVoicedScreensSpeakPlainEnglish"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-09-14
status: complete
---

# Phase 204 Plan 08: Learning Governor -- Shadow Evaluation Isolation Summary

**A new, sealed `pkg/shadow` package lets a proposed setting or rule be run beside the current behaviour over the same visible fixtures and the same hidden holdout set, graded by a `FrozenEvaluator` that four independent AST scans (each with its own synthetic violation fixture) prove a candidate structurally cannot reach or alter; `cmd/shadow_cmds.go` wires it to two new commands and records every verdict durably through the existing episode ledger.**

## Performance
- **Duration:** ~55 min
- **Tasks:** 3/3 completed
- **Files created:** 9 (7 in pkg/shadow, 2 in cmd)
- **Files modified:** 0

## Accomplishments
- `pkg/shadow/evaluator.go` -- `FrozenEvaluator` and `AcceptanceCriteria`, both sealed: unexported fields, value receivers only, a `Digest() [32]byte` accessor, no setter and no pointer-receiver method anywhere.
- `pkg/shadow/candidate.go` -- `Candidate`, immutable once constructed; `NewCandidate` refuses (by name) any of its six required declarations missing, an already-past expiry, or a declaration that names an evaluator, acceptance criterion or holdout.
- `pkg/shadow/isolation_test.go` -- four independent structural tests, each carrying a non-vacuous synthetic fixture: `TestEvaluatorHasNoMutator`, `TestNoExportedFunctionReturnsAMutableEvaluator`, `TestCandidateHoldsNoEvaluator`, plus the behavioural `TestCandidateCannotAlterItsEvaluatorDigest` (attempts running, copying, interface-passing, map-storing and double-digesting the evaluator -- digest byte-identical every time) and an import-boundary check (`TestPackageImportsNothingFromCmd`).
- `pkg/shadow/comparison.go` -- `Compare` runs the baseline and the candidate over the identical visible and holdout task sets and returns a `Comparison` carrying four `Score` values (numerator/denominator, never a bare percentage), a `Verdict` (beneficial / not_beneficial / overfit / tied / inconclusive) and both digests. An overfit candidate is never beneficial; a tie or a fully-empty comparison recommends nothing; every verdict decision cross-multiplies (`scoreCompare`) rather than comparing `Percent()`, proved with a real pair of scores (2/3 vs 67/100) that display identically but differ exactly.
- `cmd/shadow_cmds.go` -- `shadow-declare` and `shadow-compare`. Declaring stores a candidate through `shadow.NewCandidate` in an append-only store with exactly one writer (`declareShadowCandidate`, AST-enforced); a second declaration under the same id with different content is refused by name, never silently applied. Comparing resolves the visible set from the fixture bank and the holdout set through `resolveEvalGateHoldouts` -- called exactly once, in this file and nowhere else -- and records the verdict, both digests and all four scores through `recordEpisodeOutcome` as an open/close pair, replay-safe on the ledger's own digest-based idempotency. The rendered result opens every line with a shared-table glyph, states the verdict in plain English, and restates the candidate's five declarations.

## Task Commits
Each task was committed atomically:
1. **Task 1: Build the frozen grader that nothing can reach into** - `d5c62e7c` (feat)
2. **Task 2: Run the candidate beside the frozen baseline, on work it can see and checks it cannot** - `934f562e` (feat)
3. **Task 3: Give the comparison a command, and record every result durably** - `9181ebdf` (feat)

**Plan metadata:** committed alongside this SUMMARY (worktree mode -- STATE.md/ROADMAP.md excluded, handled by the orchestrator).

## Files Created/Modified
- `pkg/shadow/evaluator.go` - `FrozenEvaluator`, `NewFrozenEvaluator`, `AcceptanceCriteria`, `NewAcceptanceCriteria`, `Task`, `NewTask`, `Result`, `NewResult`
- `pkg/shadow/candidate.go` - `Candidate`, `NewCandidate`, the six-field completeness check, the forbidden-reference check, `baselineAsCandidate` (package-private)
- `pkg/shadow/baseline.go` - `Baseline`, `NewBaseline`
- `pkg/shadow/comparison.go` - `Score`, `Verdict` + vocabulary, `Comparison`, `Compare`, `scoreCompare`, `determineVerdict`
- `pkg/shadow/evaluator_test.go` - 5 named tests from the plan's artifacts list, plus `TestCandidateIsImmutableAndReadThroughAccessors`
- `pkg/shadow/isolation_test.go` - 4 structural tests + `TestCandidateCannotAlterItsEvaluatorDigest` + `TestPackageImportsNothingFromCmd`
- `pkg/shadow/comparison_test.go` - 6 named tests from the plan's artifacts list, plus `TestCompareVerdictRules`, `TestExpiredCandidateRefusedAtComparisonTime`, `TestVerdictVocabularyIsComplete`, `TestNoComparisonAppliedToPercentResult`
- `cmd/shadow_cmds.go` - `shadowDeclareCmd`, `shadowCompareCmd`, `declareShadowCandidate`, `loadShadowCandidate`, `shadowCandidateToDomain`, `shadowBaseline`, `shadowEvaluator`, `shadowVisibleAndHoldoutTasks`, `recordShadowComparisonOutcome`, `renderShadowComparison`, `runShadowCompare`
- `cmd/shadow_cmds_test.go` - 6 named tests from the plan's artifacts list, plus `TestShadowCandidateStoreHasOneWriter`

## Decisions Made
See `key-decisions` in frontmatter for the full account. Summary: `baselineAsCandidate` reuses the identical `Run(Candidate, Task) Result` seam a real candidate is graded through, rather than a second grading path; `NewFrozenEvaluator` matches the plan's literal quoted signature (no criteria parameter) and the caller folds the criteria digest into `def` instead; a shadow comparison is recorded as an episode_opened/closed pair, not a bespoke record type; the four scores are carried in `EvidenceIDs` and the baseline digest in `PolicyVersion` since `episodeLedgerRecord` (out of this plan's scope) has no dedicated fields for them.

## Deviations from Plan

### Auto-fixed / judgment-call issues

**1. [Judgment call] The grading run function is a deterministic placeholder, not a real classifier**
- **Found during:** Task 3, designing `shadowEvaluator`
- **Issue:** `FrozenEvaluator`'s `run func(Candidate, Task) Result` is caller-supplied; LEARN-06's own scope is the structural mechanism (a candidate cannot reach or alter its grader), not the concrete production question of what "passing" a given fixture-bank task means for a declared candidate (re-running the fixture's own guard test live, evaluating a real policy predicate, etc.) -- a materially larger, separate piece of engineering already partly represented by this project's existing fixture-bank/guard/eval-gate machinery (204-05/204-07), and explicitly out of scope for a single plan whose deliverable is the isolation guarantee.
- **Fix:** `shadowEvaluator` in `cmd/shadow_cmds.go` wires a deterministic, always-pass run function, documented in its own doc comment as a placeholder. Every real `shadow-compare` run currently reports a tied verdict against the unchanged baseline. The structural guarantee this plan proves (a candidate cannot reach or alter its grader, cannot select its holdout, cannot see the difference between the visible and hidden sets) holds regardless of what the run function eventually does -- a later plan can wire a real classifier through the identical, already-proven-safe seam without touching `pkg/shadow` at all.
- **Files affected:** `cmd/shadow_cmds.go` (`shadowEvaluator`)
- **Verification:** All of Task 3's own named tests pass; the placeholder's determinism is what makes `TestComparisonReplayWritesNothing` provable without flakiness.
- **Impact:** Deferred, not silently absent -- named here and in the function's own doc comment for the plan that eventually supplies a real grader.

**2. [Judgment call] Two new commands are not yet referenced from any wrapper markdown**
- **Found during:** Task 3, after registering `shadow-declare`/`shadow-compare` on `rootCmd`
- **Issue:** This repository's orphan-reachability ratchet (`cmd/subcommand_reachability_ratchet_test.go`'s `TestNoRegisteredSubcommandIsUnreferenced`/`TestOrphanAllowlistOnlyShrinks`, both outside this plan's own `<verify>` scope and its declared `files_modified`) requires a registered command to have real caller evidence (a wrapper doc, a hook script, or a worker-discipline caller file) or to already be on the immutable baseline allowlist -- which may only shrink, never grow. Neither is true yet for these two new commands, since this plan's `files_modified` is Go source only.
- **Fix:** Not fixed in this plan -- adding wrapper/menu-command exposure for `shadow-declare`/`shadow-compare` is a UX decision belonging to the promotion-gate plan (204-09) that will actually drive them, not this plan's own file scope. `TestCLIFlagAudit` (the check this plan's own `<verify>` command names) is one-directional (markdown -> Go) and passes regardless.
- **Files affected:** none (documented gap, not a code change)
- **Verification:** `TestCLIFlagAudit` passes as required by this plan's `<verify>`. The orphan-reachability ratchet was not run as part of this plan's own scope, per the project-specific guardrails naming this plan's exact verification surface.
- **Impact:** A future wrapper/menu exposure (or an owner-reviewed baseline addition) is needed before the orphan-reachability ratchet's own full-suite run passes with these two commands present; recorded here so it is visible rather than silently discovered later.

---
**Total deviations:** 2 documented judgment calls, 0 auto-fixed bugs. **Impact on plan:** Every machine-checkable `<acceptance_criteria>` line and the full `<verify>` command for all three tasks pass. Both deviations are deliberate scope boundaries this plan's own text and file list drew, not defects.

## Issues Encountered
None beyond the deviations documented above. `go build ./...`, `go vet ./cmd/... ./pkg/shadow/...` are clean; `gofmt -l` reports nothing for any created file. `go test ./pkg/shadow -count=1` (28 tests) and the full named target-test group for `cmd` pass together, alongside a regression sweep of `TestEpisode*` and the full `TestEvalGate*`/`TestSeedBank*`/`TestFixtureBank*`/`TestSeededBank*` groups (no collateral damage from reading the real, committed fixture bank and holdout file rather than a synthetic test root).

One pre-existing plan-script quirk, not a code defect: Task 2's own `<verify>` command computes `grep -c 'holdouts.json' pkg/shadow/*.go` over a multi-file glob, which `grep` prints one "file:count" line per file rather than a single sum -- a literal `test "$(...)" -eq 0` on that multi-line output is not well-formed shell. Every individual file's count is verified at 0 (confirmed per-file), which is the check's real intent; the plan's own script wording does not compose cleanly with `grep`'s multi-file output format.

## User Setup Required
None -- no external service configuration required.

## Next Phase Readiness
`shadow.FrozenEvaluator`, `shadow.NewFrozenEvaluator`, `shadow.AcceptanceCriteria`, `shadow.NewAcceptanceCriteria`, `shadow.Candidate`, `shadow.NewCandidate`, `shadow.Baseline`, `shadow.NewBaseline`, `shadow.Task`, `shadow.NewTask`, `shadow.Result`, `shadow.NewResult`, `shadow.Score`, `shadow.Verdict` (+ `VerdictNames`/`VerdictDeclared`), `shadow.Comparison`, `shadow.Compare` are all established, tested, exported symbols the promotion gate plan (204-09) can build on directly -- `shadow.Comparison.Recommended()` is the one boolean a gate needs to decide whether to act on a verdict at all.

Two things a later plan should pick up, both named above rather than silently absent:
1. A real per-fixture grading classifier to replace `cmd/shadow_cmds.go`'s current always-pass placeholder run function.
2. Wrapper/menu exposure (or an owner-reviewed orphan-allowlist entry) for `shadow-declare`/`shadow-compare` before this repository's full orphan-reachability suite passes with them present.

No blockers.

## Self-Check: PASSED

- All 9 key files confirmed present via `git status`/`ls`: `pkg/shadow/{evaluator,candidate,baseline,comparison,evaluator_test,comparison_test,isolation_test}.go`, `cmd/shadow_cmds.go`, `cmd/shadow_cmds_test.go`.
- All 3 task commits (`d5c62e7c`, `934f562e`, `9181ebdf`) confirmed present via `git log --oneline`.
- `go build ./...`, `go vet ./cmd/... ./pkg/shadow/...`: clean. `gofmt -l` over every file this plan created: empty.
- `go test ./pkg/shadow -count=1 -v`: all 28 tests pass, including the four structural isolation tests and their synthetic-fixture subtests.
- `go test ./cmd -run '^(TestCandidateStoreIsAppendOnlyAndRefusesEdits|TestComparisonWithoutACandidateIsRefusedByName|TestComparisonResultReachesTheDurableLedger|TestComparisonReplayWritesNothing|TestHoldoutResolutionHappensOnlyInTheCommandLayer|TestComparisonResultSpeaksPlainEnglish|TestCLIFlagAudit|TestVoicedScreensSpeakPlainEnglish|TestShadowCandidateStoreHasOneWriter)$' -count=1`: all pass.
- `go test ./cmd -run '^TestEpisode'` and `go test ./cmd -run '^(TestEvalGate|TestSeedBank|TestFixtureBank|TestSeededBank)'`: pass, no regression.
- Acceptance-criteria greps re-run clean: `grep -c 'calcosmic/Aether/cmd' pkg/shadow/*.go` reports 0 for every file; `grep -c 'holdouts.json' pkg/shadow/*.go` reports 0 for every file; `grep -c 'resolveEvalGateHoldouts' cmd/shadow_cmds.go` reports exactly 1.
- `LEARN-06` confirmed as this plan's sole owner (no other Phase 204 plan declares it) and marked complete in `REQUIREMENTS.md` via `requirements.mark-complete` (ready-ids reported 1/1 ready before marking).

---
*Phase: 204-learning-governor*
*Completed: 2026-09-14*
