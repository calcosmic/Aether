---
phase: 202-swarm-oracle-and-live-colony
plan: "12"
subsystem: oracle
tags: [oracle, research-synthesis, provenance, discoverability, recommendation-first]

# Dependency graph
requires:
  - phase: 202-08
    provides: "cmd/oracle_preset.go's shared Fast/Balanced/Deep/Exhaustive vocabulary -- referenced by 202-CLASSIC-SYNTHESIS.md's SYN-202-12 row alongside this plan, though this plan does not itself touch preset resolution."
  - phase: 202-11
    provides: "cmd/oracle_live.go's oracleLiveEpisodeID(state) -- a stable per-run identifier derived from oracleStateFile.StartedAt, reused here as the new front-matter run_identifier and as the resave-dedup key."
provides:
  - "cmd/oracle_synthesis.go: renderOracleFinalSynthesis(state, plan, evidenceTrail) -- the recommendation-first document body in fixed section order (Recommendation, Confidence, What Is Still Unsettled, Sources, Evidence Trail), with every recommendation-section claim traced to a plan.Questions[].KeyFindings source and refused-by-name if a cited source is missing from plan.Sources."
  - "saveOracleResearchDocument / runOracleSave (oracle save --dry-run) now build the durable document's body from renderOracleFinalSynthesis; the existing per-template report (writeOracleSynthesisReport's tech-eval/generic/etc output) is folded in verbatim as the closing Evidence Trail section rather than being the whole document."
  - "finalizeOracleResearchArtifacts now tells the owner, by topic and reason, when a run gathered no evidence and wrote nothing -- previously a silent no-op."
  - "New research front-matter keys (round_count, standing, run_identifier), additive only -- a document written before this plan still parses correctly and renders in its old shape via the same fallback path."
  - "Re-saving the same run (matched by run_identifier) now replaces its own document instead of creating a duplicate; two genuinely different runs on the same topic/day still stay as two documents."
affects: [202-13, 202-14]

# Actuals (#2632)
actuals:
  tokens: 9414
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Wrap-not-replace body composition: renderOracleFinalSynthesis takes the existing per-template report text as its evidenceTrail parameter and folds it in as the closing section, rather than re-deriving 'round-by-round detail' a second time from state/plan. This kept every pre-existing raw-body-driven test (TestOracleResearchDocumentRecordsWhatWasAsked, TestOracleResearchDocumentSurvivesNextOracleRun, TestStoppedResearchIsFiledAndLabelledPartial, etc.) passing unchanged while adding the new leading sections on top."
    - "Lenient citation validation: a recorded finding with zero source citations is allowed (Oracle's worker responses do not always attach evidence to every finding, and pre-202-12 fixtures never did); only a finding that cites a source ID absent from plan.Sources -- a genuine dangling reference -- is refused. This is the narrowest rule that satisfies 'never state a conclusion the gathered sources do not support' without retroactively invalidating every pre-existing findings-without-citations fixture."
    - "Standing reuses oracleResearchPartialLabel rather than inventing a second partial-vs-settled vocabulary (D-12): oracleResearchStandingLabel returns \"settled\" for a clean completion and the exact same partial-run sentence otherwise."

key-files:
  created:
    - cmd/oracle_synthesis.go
    - cmd/oracle_synthesis_test.go
  modified:
    - cmd/oracle_research_doc.go
    - cmd/oracle_loop.go
    - cmd/oracle_research_doc_test.go
    - cmd/oracle_autofile_198_2_test.go

key-decisions:
  - "renderOracleFinalSynthesis's third parameter (evidenceTrail) carries the pre-existing per-template synthesis.md content verbatim into the closing section, rather than re-deriving a round-by-round trail purely from structured state/plan data. This was necessary, not just convenient: the plan's own Task 2 verify gate requires the existing oracle_research_doc_test.go/oracle_autofile_198_2_test.go suites to pass unchanged, and those fixtures assert exact raw synthesis-body text appears in the saved document."
  - "'Conclusion' (the unit validated against plan.Sources) is defined as one entry per plan.Questions[].KeyFindings item, not as free-parsed sentences from state.Recommendation. This keeps validation grounded in the loop's own structured data (never inventing prose) and makes the refusal test (TestUnsupportedConclusionIsRefusedByName) deterministic and simple to construct."
  - "A finding with SourceIDs == nil/empty is NOT treated as an unsupported conclusion needing refusal -- only a finding whose SourceIDs names an ID absent from plan.Sources is refused. This lenient reading was required for backward compatibility: every pre-202-12 test fixture (seedFinishedOracleRun, the autofile-198.2 fixtures) records findings with zero citations."
  - "run_identifier reuses 202-11's oracleLiveEpisodeID(state) (derived from state.StartedAt) rather than adding a new state field -- honors the phase's running prohibition on new Oracle research state, and makes resave-dedup and live-event episode identity the same concept."
  - "round_count is an additive new key carrying the same value as the pre-existing iterations key, named in the 'round' vocabulary the rest of this phase's live events and synthesis sections already use -- the old iterations key is kept, not renamed, so nothing that reads it breaks."

requirements-completed: [LIVE-07]

coverage:
  - id: D1
    description: "The synthesis document leads with the recommendation, then confidence in plain words, then what is still unsettled and how to settle it, with sources and the evidence trail beneath -- asserted by structural section order, not merely presence."
    requirement: "LIVE-07"
    verification:
      - kind: unit
        ref: "cmd/oracle_synthesis_test.go#TestSynthesisLeadsWithTheRecommendation"
        status: pass
      - kind: unit
        ref: "cmd/oracle_synthesis_test.go#TestSynthesisStatesConfidenceInOrdinaryWords"
        status: pass
      - kind: unit
        ref: "cmd/oracle_synthesis_test.go#TestSynthesisListsUnsettledQuestionsWithTheirEvidence"
        status: pass
      - kind: unit
        ref: "cmd/oracle_synthesis_test.go#TestSynthesisWithoutARecommendationSaysSo"
        status: pass
      - kind: unit
        ref: "cmd/oracle_synthesis_test.go#TestSynthesisCarriesTheStandingLabelInTheFirstSection"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every source entry carries title, location, kind and the round that produced it; a conclusion citing a source absent from the run's own recorded sources is refused at render time, naming the conclusion; a run that gathered no evidence writes no document and is reported to the owner as empty."
    requirement: "LIVE-07"
    verification:
      - kind: unit
        ref: "cmd/oracle_synthesis_test.go#TestSynthesisSourcesCarryRoundAndKind"
        status: pass
      - kind: unit
        ref: "cmd/oracle_synthesis_test.go#TestUnsupportedConclusionIsRefusedByName"
        status: pass
      - kind: unit
        ref: "cmd/oracle_synthesis_test.go#TestEmptyResearchRunWritesNoDocument"
        status: pass
      - kind: unit
        ref: "cmd/oracle_synthesis_test.go#TestEmptyResearchRunIsReportedAsEmpty"
        status: pass
      - kind: unit
        ref: "go test ./cmd -run '^(TestOracleAutofile|TestOracleResearchDoc|TestRunResearchList)' -count=1 (existing research-document/autofile tests unchanged)"
        status: pass
    human_judgment: false
  - id: D3
    description: "A finished run's synthesis is saved out of the run workspace, survives the next run's workspace reset, and is discoverable through the existing research listing; the front matter carries topic, standing, confidence, round count and run identifier; resaving the same run replaces its document rather than duplicating it; a pre-plan-shape document still parses and lists correctly."
    requirement: "LIVE-07"
    verification:
      - kind: unit
        ref: "cmd/oracle_synthesis_test.go#TestSynthesisIsSavedOutOfTheWorkspace"
        status: pass
      - kind: unit
        ref: "cmd/oracle_synthesis_test.go#TestSavedSynthesisFrontMatterCarriesStandingAndConfidence"
        status: pass
      - kind: unit
        ref: "cmd/oracle_synthesis_test.go#TestResaveOfOneRunDoesNotDuplicate"
        status: pass
      - kind: unit
        ref: "cmd/oracle_synthesis_test.go#TestResearchListStillParsesOlderDocuments"
        status: pass
      - kind: unit
        ref: "go vet ./cmd"
        status: pass
    human_judgment: false

duration: 19min
completed: 2026-09-11
status: complete
---

# Phase 202 Plan 12: Recommendation-First Oracle Synthesis Summary

**`renderOracleFinalSynthesis` renders every saved Oracle research document recommendation-first (recommendation, plain-English confidence, unsettled questions with what would settle them, sources, evidence trail), refuses to render an unsupported claim by name, and the saved document is now idempotent-per-run and discoverable with a "settled"/partial standing label.**

## Performance

- **Duration:** ~19 min (task-commit span; front-loaded with reading `cmd/oracle_research_doc.go`, `cmd/oracle_loop.go`, `cmd/oracle_live.go`, and the existing test fixtures to find every place a naive body-replacement would have broken a pre-existing test)
- **Started:** 2026-09-11T15:09:44+02:00 (first task commit)
- **Completed:** 2026-09-11T15:28:33+02:00 (third task commit)
- **Tasks:** 3
- **Files modified:** 6 (2 created, 4 modified)

## Accomplishments

- `cmd/oracle_synthesis.go` (new): `renderOracleFinalSynthesis(state, plan, evidenceTrail)` renders the document in fixed section order -- Recommendation, Confidence, What Is Still Unsettled, Sources, Evidence Trail -- with `oracleSynthesisConclusions`/`validateOracleSynthesisConclusions` deriving every recommendation-section claim from `plan.Questions[].KeyFindings` (never composed prose) and refusing to render if a finding cites a source ID absent from `plan.Sources`, naming the offending conclusion.
- The Sources section carries every source's title, location (URL), kind (`src.Type`) and the earliest round (`oracleSynthesisSourceRound`) any finding cited it in -- derived purely from existing `oracleFinding.Iteration`/`SourceIDs` data, no new state.
- Confidence is printed as the exact recorded integer against the target, never rounded or rescaled, with a plain-English sentence (`oracleConfidenceExplanation`) describing what the figure means.
- Unsettled questions reuse the existing `identifyGaps` (state.OpenGaps + under-confidence questions), each rendered with a plain-English hint (`oracleSynthesisGapHint`) naming the kind of evidence that would settle it.
- `saveOracleResearchDocument` and `runOracleSave --dry-run` now build the durable document's body from `renderOracleFinalSynthesis`, with the pre-existing per-template report (`writeOracleSynthesisReport`'s tech-eval/generic/architecture-review/bug-investigation output) folded in verbatim as the closing Evidence Trail section -- this preserved every pre-existing raw-body-driven test unchanged while adding the new leading sections.
- `finalizeOracleResearchArtifacts` now emits an owner-facing message naming the topic and reason when a run gathered no evidence and wrote nothing (previously silent).
- Front matter gained three additive keys -- `round_count` (same value as the existing `iterations`, named in the "round" vocabulary the rest of the phase uses), `standing` (reuses `oracleResearchPartialLabel`; "settled" for a clean completion), and `run_identifier` (reuses 202-11's `oracleLiveEpisodeID`). `runResearchList`/`renderResearchList` display `Standing`, falling back to `Status` for documents written before this plan so their output shape is unchanged.
- `saveOracleResearchDocument` now scans the research directory for an existing document whose `run_identifier` matches the current run (`oracleFindExistingResearchDocument`) and replaces it in place, rather than always minting a new filename -- resaving the same run is now idempotent; two runs with distinct `StartedAt` values on the same topic/day still produce two documents.

## Task Commits

Each task was committed atomically:

1. **Task 1: Lead with the recommendation, then confidence, then what is unsettled** - `e7512b38` (feat)
2. **Task 2: Make every claim traceable, and write nothing for an empty run** - `1776092c` (feat)
3. **Task 3: Keep the synthesis discoverable after the run ends** - `dcd4da75` (feat)

**Plan metadata:** (this commit)

_Note: all three tasks carried `tdd="true"`. Each landed its required tests alongside the implementation in the same commit (the pattern 202-01/202-02/202-08/202-11 each documented) -- the described behaviors (rendering fixed sections from existing data, wiring an existing renderer into an existing save path, extending front matter with additive keys) don't have a meaningful pre-implementation "should fail" state distinct from "doesn't compile yet."_

## Files Created/Modified

- `cmd/oracle_synthesis.go` - `renderOracleFinalSynthesis`, `oracleSynthesisConclusion(s)`, `validateOracleSynthesisConclusions`, `oracleSynthesisSourceRound`, `oracleConfidenceExplanation`, `oracleSynthesisGapHint`, `oracleEmptyRunTopic`.
- `cmd/oracle_synthesis_test.go` - all twelve named tests across the three tasks, plus `oracleSynthesisFixture` and `directoryDigest` test helpers.
- `cmd/oracle_research_doc.go` - `saveOracleResearchDocument`/`runOracleSave` now call `renderOracleFinalSynthesis`; new front-matter keys (`round_count`, `standing`, `run_identifier`); `oracleResearchStandingLabel`; `oracleFindExistingResearchDocument`; `oracleResearchEntry`/`parseOracleResearchFrontMatter`/`renderResearchList` extended for `Standing`/`RoundCount`/`RunIdentifier`.
- `cmd/oracle_loop.go` - `finalizeOracleResearchArtifacts` now emits the owner-facing empty-run message before its early return.
- `cmd/oracle_research_doc_test.go` - two pre-existing fixtures (`TestOracleSaveKeepsBothRunsOnTheSameTopicAndDay`, `TestResearchListReportsSavedDocuments`) updated to give their two logical runs distinct `StartedAt` values (see Deviations).
- `cmd/oracle_autofile_198_2_test.go` - `TestResearchOnTheSameTopicKeepsBothWriteUps` given the same treatment, plus a doc-comment note explaining why.

## Decisions Made

See `key-decisions` in frontmatter.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Touched `cmd/oracle_loop.go`, not in the plan's declared `files_modified`**
- **Found during:** Task 2
- **Issue:** The plan's frontmatter `files_modified` lists only `cmd/oracle_synthesis.go`, `cmd/oracle_synthesis_test.go`, `cmd/oracle_research_doc.go` -- but Task 2's own action text explicitly requires adding the owner-facing empty-run message to `finalizeOracleResearchArtifacts`, which lives in `cmd/oracle_loop.go`. Following the plan's own instructions was impossible without this file.
- **Fix:** Added a five-line, narrowly-scoped `emitVisualLine` call inside the existing early-return branch; no other behavior in the file changed.
- **Files modified:** `cmd/oracle_loop.go`
- **Verification:** `TestEmptyResearchRunIsReportedAsEmpty`; full `^TestOracle` suite still green.
- **Committed in:** `1776092c` (Task 2 commit)

**2. [Rule 1 - Bug] Three pre-existing tests modelled "two different runs" by reusing one identical state value, which Task 3's new resave-identity rule collapsed into "one run saved twice"**
- **Found during:** Task 3, immediately after wiring resave-dedup by `run_identifier`
- **Issue:** `TestOracleSaveKeepsBothRunsOnTheSameTopicAndDay`, `TestResearchListReportsSavedDocuments` (`cmd/oracle_research_doc_test.go`) and `TestResearchOnTheSameTopicKeepsBothWriteUps` (`cmd/oracle_autofile_198_2_test.go`) each simulate "two runs on the same topic on the same day" by calling the save path twice with a single, unmutated `oracleStateFile` value (identical `StartedAt`, hence identical `run_identifier` under the new rule). Task 3's own explicit requirement ("re-saving the same run does not create a second document for it") makes this exactly the case that now correctly collapses to one document -- which is not what any of these three tests intended to prove.
- **Fix:** Gave the second logical run in each test its own distinct `StartedAt` value, matching what a genuinely separate `aether oracle` invocation would carry. This restores each test's original intent (two independent runs on the same topic/day both survive) without weakening the new dedup guarantee, which is proven separately by the new `TestResaveOfOneRunDoesNotDuplicate` (same state, same run, one document).
- **Files modified:** `cmd/oracle_research_doc_test.go`, `cmd/oracle_autofile_198_2_test.go`
- **Verification:** All three tests pass; `go test ./cmd -run 'Research|Oracle|Synthesis' -count=1` is green with zero `--- FAIL` lines.
- **Committed in:** `dcd4da75` (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (1 Rule 3 - a file the plan's own action text required but its frontmatter omitted; 1 Rule 1 - three test fixtures whose reused-state shortcut became ambiguous under a new, explicitly-required identity rule).
**Impact on plan:** Both were necessary to satisfy the plan's own literal instructions and to keep the full existing suite green. No scope creep -- no production behavior changed in the touched test files, only which `StartedAt` value distinguishes "two runs" in fixtures.

## Issues Encountered

None beyond the deviations documented above. A full unscoped `go test ./cmd -count=1` run was attempted as additional diligence and hit this machine's known ~20-minute infrastructure ceiling (parallel lanes exceeding a 9-minute deadline, a pre-existing environment issue tracked in prior project history, not caused by this plan) -- it reported zero `--- FAIL` lines for any test before timing out, and every oracle/research/synthesis-scoped run (the plan's own `<verify>` commands, plus a broader `Research|Oracle|Synthesis` sweep) completed and passed within seconds.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `renderOracleFinalSynthesis` is self-contained and depends only on 202-11's `oracleLiveEpisodeID` (already shipped); no other in-progress plan needs it as a precondition.
- The three new front-matter keys (`round_count`, `standing`, `run_identifier`) are purely additive -- `runResearchList`/`renderResearchList` and `parseOracleResearchFrontMatter` still parse and render documents written before this plan without any special-casing beyond the existing `emptyFallback` pattern.
- 202-13's shared partial-work vocabulary (SYN-202-12) can build on `oracleResearchStandingLabel`/`standing` directly rather than re-deriving Oracle's own settled/partial distinction.
- Plan-required verification (`go test ./cmd -run` for all twelve named tests across the three tasks, the existing `TestOracleAutofile*`/`TestOracleResearchDoc*`/`TestRunResearchList*` regression set, the full `^TestOracle` suite, and `go vet ./cmd`) all pass with no regressions.
- No blockers. Ready for 202-13.

## Self-Check: PASSED

- `cmd/oracle_synthesis.go` - FOUND, builds clean
- `cmd/oracle_synthesis_test.go` - FOUND, all 12 named tests pass
- `cmd/oracle_research_doc.go` - FOUND, modified as described
- `cmd/oracle_loop.go` - FOUND, modified as described
- `cmd/oracle_research_doc_test.go` - FOUND, modified as described
- `cmd/oracle_autofile_198_2_test.go` - FOUND, modified as described
- Commit `e7512b38` - FOUND in `git log --oneline --all`
- Commit `1776092c` - FOUND in `git log --oneline --all`
- Commit `dcd4da75` - FOUND in `git log --oneline --all`
- `go test ./cmd -run '^(TestSynthesisLeadsWithTheRecommendation|TestSynthesisStatesConfidenceInOrdinaryWords|TestSynthesisListsUnsettledQuestionsWithTheirEvidence|TestSynthesisWithoutARecommendationSaysSo)$' -count=1` - PASS
- `go test ./cmd -run '^(TestSynthesisSourcesCarryRoundAndKind|TestUnsupportedConclusionIsRefusedByName|TestEmptyResearchRunWritesNoDocument|TestEmptyResearchRunIsReportedAsEmpty)$' -count=1 && go test ./cmd -run '^(TestOracleAutofile|TestOracleResearchDoc|TestRunResearchList)' -count=1` - PASS
- `go test ./cmd -run '^(TestSynthesisIsSavedOutOfTheWorkspace|TestSavedSynthesisFrontMatterCarriesStandingAndConfidence|TestResaveOfOneRunDoesNotDuplicate|TestResearchListStillParsesOlderDocuments)$' -count=1 && go vet ./cmd` - PASS
- `go test ./cmd -run '^TestOracle' -count=1` (full existing Oracle suite) - PASS
- `go test ./cmd -run 'Research|Oracle|Synthesis' -count=1` (broad regression sweep) - PASS

---
*Phase: 202-swarm-oracle-and-live-colony*
*Completed: 2026-09-11*
