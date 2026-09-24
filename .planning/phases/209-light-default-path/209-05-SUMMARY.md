---
phase: 209-light-default-path
plan: 05
subsystem: cli
tags: [measurement, documentation, golden-fixtures, ci-hygiene]

requires:
  - phase: 209-light-default-path (plans 01-04)
    provides: "/ant-go's small route, big route, D-03 self-escalation, the owner-ruled default menu, and the derived-specification planning handoff -- everything this plan measures and documents"
provides:
  - "209-TIMING.md -- four real claude -p sessions (small/medium jobs, each once through /ant-go and once through plain Claude), timed and costed from each session's own final cost-state record, with an honest verdict"
  - "CLAUDE.md's 'A Light Default Path (v1.29, Phase 209)' section, every claim naming a live, verified test"
  - "the /ant-go row and the advanced-commands note landed byte-identically in .aether/rules/aether-colony.md and .claude/rules/aether-colony.md"
  - "four phase-209-introduced (plans 01/02), not this plan's own, gaps closed: a missing command-guide entry for go, a missing D-01 severity classification for go and advanced-commands, a local directory skip-list in go_route.go replaced with the canonical codegraph.ShouldSkipDir, and three stale golden fixtures refreshed"
affects: [210, any later phase reading this phase's own CLAUDE.md section or timing report]

actuals:
  tokens: 9700
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "One canonical skip-list, never a local one: cmd/go_route.go's gatherJobSizeFacts now calls pkg/codegraph.ShouldSkipDir instead of maintaining its own jobSizeSkipDirNames map -- the same rule TestSkipListDivergence enforces structurally across all of cmd/."
    - "A reviewed enrichment/gate judgement, not a default: /ant-go and advanced-commands were added to knownEnrichmentSubcommands with a one-line rationale each, per command_call_audit_test.go's own review-not-inherited contract (T-160-23)."

key-files:
  created:
    - .planning/phases/209-light-default-path/209-TIMING.md
  modified:
    - CLAUDE.md
    - .aether/rules/aether-colony.md
    - .claude/rules/aether-colony.md
    - cmd/command_guide.go
    - cmd/command_call_audit_test.go
    - cmd/go_route.go
    - cmd/testdata/command_catalog.json
    - cmd/testdata/parity_snapshot.json
    - cmd/testdata/regression_snapshot.json
    - .planning/WINDOWS.md

key-decisions:
  - "The medium job was deliberately built to route through /ant-go's planning (big) route -- a new package name with zero matched existing paths -- because 209-CONTEXT.md's own checkpoint reasoning says the planning route is where the real gap is most likely to be found. It was: the first /ant-go call on the medium job stopped and asked how to proceed instead of dispatching the planning helpers itself, contradicting the phase's own 'zero further owner-typed steps' claim. Recorded exactly as observed in 209-TIMING.md and filed as an open finding, not patched -- fixing the wrapper's own planning hand-off text is out of this plan's scope."
  - "The practice-project fixture's own aether install/update --force steps were found, live, to sync Aether's shared slash commands to this machine's real, global ~/.claude and ~/.codex by default -- contradicting the fixture script's own isolation comment. Worked around at the call site (an isolated HOME for the one setup step only, never for the timed chat sessions, which need the owner's real login) rather than edited in the script itself. Filed to WINDOWS.md as entry 59; confirmed before and after that no real ~/.claude or ~/.codex content was actually written by any of the four measured runs."
  - "The seven full-suite failures beyond the recorded 30-failure baseline were all traced to phase 209's own earlier plans (01/02 adding the /ant-go command) never having refreshed three golden fixtures, added a command-guide entry, classified /ant-go's and advanced-commands' D-01 severity, or reused the canonical directory skip-list -- not to this plan's own doc-only changes. Per this plan's own Task 3 instruction ('fix anything this phase caused'), all four were fixed here rather than left open, then the full suite was re-run to confirm exactly the same 30 pre-existing failures and zero new ones."

requirements-completed: [UED-16, UED-17, UED-18]

coverage:
  - id: D1
    description: "The same small job and the same medium job were each run once through /ant-go and once through plain Claude, with wall time and cost read from each session's own recorded cost-state figures and an honest verdict written up for the owner in 209-TIMING.md -- a slower or less-autonomous result reported first, not hidden."
    requirement: UED-16
    verification: []
    human_judgment: true
    rationale: "This is a real-money (subscription-allowance), real-chat-session measurement, not something an automated test can substitute for. A human should read 209-TIMING.md itself to judge whether the report is honest and matches what actually happened; the underlying session transcripts (session IDs recorded in the report) are the durable evidence."
  - id: D2
    description: "CLAUDE.md's new 'A Light Default Path (v1.29, Phase 209)' section describes the single entry command, the two-valued route decision, the self-escalation, the owner-ruled default menu, and the zero-extra-steps planning handoff, with every runtime claim naming a test that exists and passes."
    requirement: UED-16
    verification:
      - kind: unit
        ref: "cmd/go_route_test.go#TestGoRouteIsComputedNotConstant"
        status: pass
      - kind: unit
        ref: "cmd/go_route_test.go#TestGoRouteIgnoresHowTheSentenceIsWorded"
        status: pass
      - kind: unit
        ref: "cmd/go_route_test.go#TestGoRouteHasOneAuthority"
        status: pass
      - kind: unit
        ref: "cmd/go_escalation_test.go#TestGoEscalatesWhenTheSmallAttemptProvesBigger"
        status: pass
      - kind: unit
        ref: "cmd/go_escalation_test.go#TestGoNeverMovesAJobBackDown"
        status: pass
      - kind: unit
        ref: "cmd/go_escalation_test.go#TestGoEscalationNeverAsksAndNeverBlocks"
        status: pass
      - kind: unit
        ref: "cmd/front_door_209_test.go#TestDefaultMenuShowsOnlyTheApprovedSet"
        status: pass
      - kind: unit
        ref: "cmd/front_door_209_test.go#TestHiddenCommandsStillRun"
        status: pass
      - kind: unit
        ref: "cmd/go_default_path_test.go#TestGoalReachesBuiltWorkWithNoExtraSteps"
        status: pass
      - kind: unit
        ref: "cmd/go_default_path_test.go#TestDiscussAndSpecStillBehaveExactlyAsBefore"
        status: pass
    human_judgment: false
  - id: D3
    description: "The /ant-go row and the short advanced-commands note landed in both .aether/rules/aether-colony.md and .claude/rules/aether-colony.md, and the two files are byte-identical."
    requirement: UED-18
    verification:
      - kind: other
        ref: "diff .aether/rules/aether-colony.md .claude/rules/aether-colony.md"
        status: pass
    human_judgment: false
  - id: D4
    description: "The full suite (go test ./... -count=1 -timeout 90m) ran to completion twice, discovered equal executed both times, with exactly the same 30 pre-existing failures both times and zero failures newly introduced by this plan."
    verification:
      - kind: other
        ref: "go test ./... -count=1 -timeout 90m (FULL-SUITE headline: discovered=6057 executed=6057, both runs)"
        status: pass
    human_judgment: false

duration: 1h 59min
completed: 2026-09-24
status: complete
---

# Phase 209 Plan 05: Timed Against Plain Claude, and the Phase Documented Summary

**Four real `claude -p` sessions timed `/ant-go` against plain Claude on identical jobs — the medium job's real planning handoff stopped and asked instead of finishing unattended, a genuine gap recorded honestly rather than hidden — then CLAUDE.md and the distributed rules files were updated with test-backed claims, and the full suite was run to completion twice with zero new failures.**

## Performance

- **Duration:** 1h 59min
- **Started:** 2026-09-24T16:43:56Z (approx, first commit after 209-04)
- **Completed:** 2026-09-24T18:42:29Z
- **Tasks:** 2 (Task 1 was the checkpoint, already answered by the owner's 2026-09-24 ruling — see below)
- **Files modified:** 9 (1 created)

## Owner ruling carried into this execution

Task 1's `checkpoint:decision` was answered by the owner on 2026-09-24, before this plan was
dispatched: **measure-now** — run all four real sessions. His one condition ("as long as it's
not real money") was checked and confirmed met before Task 2 ran: this machine has no
`ANTHROPIC_API_KEY`/`ANTHROPIC_AUTH_TOKEN` set, and the `claude` CLI (`2.1.281`) authenticates
through his existing subscription, so all four sessions drew on already-paid allowance rather
than a separate metered bill. Recorded as the authority for Task 2 in
`.planning/phases/209-light-default-path/209-TIMING.md` itself.

## Accomplishments

- **Task 2 — the timing measurement.** Built four fresh, identical throwaway practice projects
  with `scripts/build-messy-practice-project.sh`, ran a small job and a medium job each once
  through `/ant-go` and once through plain Claude, and read wall time and cost from each
  session's own final `cost-state` transcript record (matching `207-JOURNEY-RUN.md`'s own
  method). Small job: `/ant-go` took ~2x as long (36.4s vs 17.0s) and a few cents more ($0.44
  vs $0.40); both landed correctly. Medium job: `/ant-go`'s first call stopped after 45s/$0.52
  and asked how to proceed instead of finishing on its own — contradicting this phase's own
  "zero further owner-typed steps" promise — and only after one nudge did the whole session
  (833.4s / ~13.9 min, $6.44) actually build and verify the real greeting package. Full honest
  verdict, both job sentences verbatim, and the new-finding writeup are in `209-TIMING.md`.
- **A live isolation gap found and worked around, filed rather than fixed.** The practice-project
  script's own `aether install`/`aether update --force` steps sync Aether's global slash
  commands to this machine's real `~/.claude` and `~/.codex` by default, contradicting the
  script's own isolation comment. Confirmed no real content was written during this measurement
  (checked before and after); worked around at the call site with an isolated `HOME` for setup
  only. Filed as `.planning/WINDOWS.md` entry 59.
- **Task 3 — documentation.** Added "A Light Default Path (v1.29, Phase 209)" to `CLAUDE.md`,
  citing ten real, individually-confirmed-passing tests (`TestGoRouteIsComputedNotConstant`,
  `TestGoRouteIgnoresHowTheSentenceIsWorded`, `TestGoRouteHasOneAuthority`,
  `TestGoEscalatesWhenTheSmallAttemptProvesBigger`, `TestGoNeverMovesAJobBackDown`,
  `TestGoEscalationNeverAsksAndNeverBlocks`, `TestDefaultMenuShowsOnlyTheApprovedSet`,
  `TestHiddenCommandsStillRun`, `TestGoalReachesBuiltWorkWithNoExtraSteps`,
  `TestDiscussAndSpecStillBehaveExactlyAsBefore`), plus an honest "one limit, measured rather
  than assumed" paragraph pointing at the Task 2 finding above, and the "for dummies" close the
  house style requires. Added the identical `/ant-go` row and advanced-commands note to both
  `.aether/rules/aether-colony.md` and `.claude/rules/aether-colony.md` — confirmed byte-identical
  afterward.
- **Four phase-209-caused gaps closed, found only by running the full suite.** Plans 01/02 added
  `/ant-go` across the wrapper surfaces but never: gave it a command-guide entry
  (`TestAllYamlHaveWrappersAndGuide`, `TestCommandGuideCoversAllYamlCommands` — fixed by adding
  `"go"` to `commandGuideLiteralCommands()`); classified its (or `advanced-commands`'s) D-01
  severity (`TestDocumentedSubcommandsAreSeverityClassified` — fixed by adding both, with
  rationale, to `knownEnrichmentSubcommands`); or refreshed three golden fixtures that the new
  command shifted (`TestRegressionSnapshot`, `TestAuditCatalogGolden`, `TestPlatformParityGolden`
  — fixed via each test's own `-update-golden` path). Separately, `cmd/go_route.go`'s
  `gatherJobSizeFacts` kept its own local `jobSizeSkipDirNames` map instead of the one canonical
  skip-list every tree-walk in this repo must use (`TestSkipListDivergence` — fixed by switching
  to `codegraph.ShouldSkipDir`, which also let a now-redundant explicit `.aether/data` skip be
  removed). None of these four were caused by this plan's own doc-only changes; all four are
  phase-209 regressions this plan's own "fix anything this phase caused" instruction requires
  closing.
- **The full suite ran to completion twice**, `go test ./... -count=1 -timeout 90m`, both times
  with `FULL-SUITE ... discovered=6057 executed=6057` (equal — not truncated). Before the four
  fixes above: 37 unique failing test names (30 pre-existing plus the 7 phase-209 gaps). After:
  exactly 30 unique failing test names both runs, an exact match against
  `.planning/WINDOWS.md` entry 56's own recorded 30-failure baseline (2026-09-23, commit
  `aaab3dfa`) and this repo's own known-red rules — zero new, zero newly-passing.

## Task Commits

1. **Task 2: the timing measurement** — `3ab83659` (docs) — `209-TIMING.md` created,
   `.planning/WINDOWS.md` entry 59 added.
2. **Task 3: documentation + full-suite gap closure** — `f84f4b93` (docs) — `CLAUDE.md`, both
   rules files, `cmd/command_guide.go`, `cmd/command_call_audit_test.go`, `cmd/go_route.go`,
   and three `cmd/testdata/*.json` golden fixtures.

**Plan metadata:** (this commit, following this SUMMARY)

## Files Created/Modified

- `.planning/phases/209-light-default-path/209-TIMING.md` — the four real runs, verdict, and new finding
- `CLAUDE.md` — new "A Light Default Path (v1.29, Phase 209)" section
- `.aether/rules/aether-colony.md`, `.claude/rules/aether-colony.md` — `/ant-go` row + advanced-commands note, byte-identical
- `cmd/command_guide.go` — `"go"` added to `commandGuideLiteralCommands()`
- `cmd/command_call_audit_test.go` — `"go"` and `"advanced-commands"` added to `knownEnrichmentSubcommands`, each with a one-line rationale
- `cmd/go_route.go` — `gatherJobSizeFacts` now calls `codegraph.ShouldSkipDir` instead of the local `jobSizeSkipDirNames` map; the now-redundant explicit `.aether/data` skip removed
- `cmd/testdata/command_catalog.json`, `cmd/testdata/parity_snapshot.json`, `cmd/testdata/regression_snapshot.json` — refreshed golden fixtures (each via its own test's `-update-golden` path)
- `.planning/WINDOWS.md` — entry 59 (the practice-project fixture's own real-HOME sync gap)

## Decisions Made

See `key-decisions` in frontmatter above.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Command allowlist restriction blocked the medium-Aether run's own subagent dispatch tool**
- **Found during:** Task 2, first attempt at the medium-Aether run
- **Issue:** An initial `claude -p` invocation restricted `--allowedTools` to `Bash,Edit,Write,Read,Glob,Grep`, excluding the subagent-dispatch tool. The session correctly reported it could not actually dispatch planning helpers and stopped to ask, which looked at first like a tool-permission artifact of the test harness rather than a real finding.
- **Fix:** Rebuilt all four practice projects fresh and reran all four sessions with no tool allowlist restriction (`--permission-mode bypassPermissions`, no `--allowedTools`), matching how the owner would actually run Claude Code with nothing artificially withheld. The medium-Aether run's behavior was unchanged: it still stopped and asked on the first call. This confirmed the stall is a real property of the `/ant-go` wrapper's own instructions, not a test-harness artifact — see the key-decisions entry above.
- **Files modified:** none (test harness only, all in `/tmp`)
- **Verification:** rerun with unrestricted tools reproduced the identical stall; documented in `209-TIMING.md`.
- **Committed in:** n/a (the timing report itself, `3ab83659`, records the corrected methodology)

**2. [Rule 1/3 - Bug/Blocking] Four phase-209-introduced full-suite failures**
- **Found during:** Task 3's own full-suite run requirement
- **Issue:** Plans 01/02 added the `/ant-go` command across wrapper surfaces without a command-guide entry, a D-01 severity classification, or refreshing three golden fixtures that its addition shifted; `go_route.go` also carried its own local directory skip-list rather than the one canonical list every other tree-walk in this repo uses.
- **Fix:** See "Four phase-209-caused gaps closed" above.
- **Files modified:** `cmd/command_guide.go`, `cmd/command_call_audit_test.go`, `cmd/go_route.go`, `cmd/testdata/command_catalog.json`, `cmd/testdata/parity_snapshot.json`, `cmd/testdata/regression_snapshot.json`
- **Verification:** all seven previously-failing tests pass individually and together; the full suite re-run afterward reproduces exactly the 30-failure pre-existing baseline, no more, no less.
- **Committed in:** `f84f4b93`

---

**Total deviations:** 2 (1 test-methodology correction with no production files touched, 1 set of four Rule 1/3 fixes for phase-209-caused, not this plan's own, gaps).
**Impact on plan:** Both were necessary to produce an honest measurement and a genuinely clean full-suite run. No scope creep beyond what Task 2's own honesty requirement and Task 3's own "fix anything this phase caused" instruction required.

## Issues Encountered

None beyond the two deviations above, both resolved during execution.

Pre-existing, unrelated full-suite failures (unchanged before and after this plan's own fixes,
30 in both runs): `TestBuildStartLegacyHelpersRetired200`, `TestCheapModelWorkerNeedsNoReasonOnTheCard`,
`TestCodexBuildPlanOnlySpawnBudgetSeparatesCasteBudgetFromWorkerCount`,
`TestCodexNativeCancellationRefusalEvidence`, `TestCodexNativeCancellationRefusalReplay`,
`TestCodexNativeEvidenceReceiptSchemaDispatch`, `TestCodexNativeEvidenceRejects`,
`TestCodexNativeFourthReviewLegacyReplayInventory`, `TestCodexNativeGapRecovery`,
`TestCodexNativePhaseEvidence`, `TestCodexNativeThirdReviewReplayInventory`,
`TestCompletionPacketSchemaMatchesStructs`, `TestContinueCreditsTasksProvenInAnEarlierAttempt`,
`TestCurrentVocabulary199`, `TestDefaultOracleInvokerAvoidsOpenCodeInsideOpenCodeAgent`,
`TestFailedCheckSendsExactlyOneBuilderFixAttempt`, `TestFixAttemptIsCountedSeparately`,
`TestFixAttemptNeverOverwritesTheFirstResult`, `TestGoldenBuildVisualOutput`,
`TestGoldenContinueVisualOutput`, `TestGoSourceHintsMatchCobraContracts`,
`TestHumanFacingOutputGoesThroughWriteVisualOutput`, `TestNoRegisteredSubcommandIsUnreferenced`,
`TestNoSecondAutomaticFixAttempt`, `TestPartialRedispatchRecoveryNeverNamesAlreadyProvenWork`,
`TestPhase199GateReceipt`, `TestPlanningAdversarial200`, `TestPlanningPublicPaths200`,
`TestResolveTestCommand_GoProject`, `TestSeededBankIsReproducible`. All 30 exactly match
`.planning/WINDOWS.md` entry 56 (2026-09-23, commit `aaab3dfa`) and this repo's own CLAUDE.md
repo-specific rules (the four named there — `TestGoSourceHintsMatchCobraContracts`,
`TestGoldenBuildVisualOutput`, `TestGoldenContinueVisualOutput`,
`TestNoRegisteredSubcommandIsUnreferenced` — are all present in this list). None touched, none
caused by this plan, per the deviation rules' scope boundary.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 209 ("A Light Default Path") is now fully executed: all five plans complete.
- ROADMAP success criterion 4 is satisfied per A-01/A-02's own definition — a written measurement,
  not a new harness — with the honest result that `/ant-go` was not faster than plain Claude on
  either job measured, and the medium job's planning handoff needed an owner nudge.
- A genuine, freshly-found gap is now on record for a future phase to close: `/ant-go`'s wrapper
  text does not itself instruct a live assistant to dispatch the real planning helpers the way
  `/ant-plan`'s own wrapper does, so an unattended run can stall at that handoff. Not fixed here
  (out of this plan's own scope — a wrapper-text change, not a documentation or measurement one).
- `.planning/WINDOWS.md` entry 59 (the practice-project fixture's own real-HOME platform sync)
  and the reopened medium-Aether finding are both new, both recorded, neither fixed here.

## Known Stubs

None. Every change in this plan is either a real documentation update backed by a passing test,
a real measurement with real session transcripts as evidence, or a real, verified bug fix.

---
*Phase: 209-light-default-path*
*Completed: 2026-09-24*

## Self-Check: PASSED
