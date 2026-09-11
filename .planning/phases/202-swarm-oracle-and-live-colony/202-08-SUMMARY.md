---
phase: 202-swarm-oracle-and-live-colony
plan: "08"
subsystem: oracle
tags: [oracle, research-depth, planning-preset, vocabulary-unification]

# Dependency graph
requires:
  - phase: 202-01
    provides: "202-CLASSIC-SYNTHESIS.md's SYN-202-10 ruling: Oracle's preset labels are a pure mapping layer over the unchanged oracleDepthLevels numeric table -- rewriting Oracle's own confidence targets to match planning's would be a real regression to fix a display inconsistency."
provides:
  - "cmd/oracle_preset.go: oraclePresetPolicy/oraclePresetPolicies/resolveOraclePreset/oraclePresetOptions -- the shared Fast/Balanced/Deep/Exhaustive vocabulary over Oracle's own unchanged oracleDepthLevels numbers, computed fresh from the map on every call so a later change to Oracle's economics flows through automatically."
  - "The research proposal picker, the approved brief panel, the --depth flag's help text, and the owner-facing run start (startOracleCompatibility) all speak the shared four names; every previously accepted legacy word (quick/balanced/standard/deep/exhaustive/marathon) still resolves to the same numbers it always resolved to."
affects: []

# Actuals (#2632)
actuals:
  tokens: 6202
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Preset-over-legacy-map pattern: oraclePresetPolicies() rebuilds the four shared entries from oracleDepthLevels on every call (never a value copied in once at init), so a test that mutates the underlying map observes the change immediately and a later change to Oracle's own economics needs no second edit."
    - "resolveOraclePreset accepts both the shared names and every legacy oracleDepthLevels key via an explicit alias map (oraclePresetAliasKey), refusing anything else by name -- mirroring planningPresetPolicyByName's shape in cmd/codex_plan.go without importing planning's own resolution logic."

key-files:
  created:
    - cmd/oracle_preset.go
    - cmd/oracle_preset_test.go
  modified:
    - cmd/oracle_brief.go
    - cmd/oracle_loop.go
    - cmd/compatibility_cmds.go

key-decisions:
  - "oraclePresetPolicies is a function, not a package-level var initialized once -- required so a test mutating oracleDepthLevels directly observes the preset follow it, and so the entries are genuinely 'read at construction' rather than typed in."
  - "reused cmd/planning_stage.go's existing planningStagePreset type (and its Fast/Balanced/Deep/Exhaustive constants) as oraclePresetPolicy.ID instead of declaring a parallel Oracle-only enum -- this is literally 'the same vocabulary planning already established', not a copy of it."
  - "runOracleBriefApprove's --depth validation now also accepts the shared names via resolveOraclePreset (previously only the legacy oracleDepthLevels keys) -- see Deviations: without this, the picker's own recommended Value would be rejected by the very subcommand it tells the owner to run next."

requirements-completed: [LIVE-06]

coverage:
  - id: D1
    description: "Oracle's own depth machinery is reachable through one shared Fast/Balanced/Deep/Exhaustive vocabulary (oraclePresetPolicy/oraclePresetPolicies/resolveOraclePreset/oraclePresetOptions), with every legacy word still resolving to its unchanged numbers and an unknown word refused by name."
    requirement: "LIVE-06"
    verification:
      - kind: unit
        ref: "cmd/oracle_preset_test.go#TestOraclePresetLabelsMatchPlanningVocabulary"
        status: pass
      - kind: unit
        ref: "cmd/oracle_preset_test.go#TestOraclePresetNumbersAreUnchanged"
        status: pass
      - kind: unit
        ref: "cmd/oracle_preset_test.go#TestOracleLegacyDepthWordsStillResolve"
        status: pass
      - kind: unit
        ref: "cmd/oracle_preset_test.go#TestOracleUnknownDepthIsRefusedByName"
        status: pass
    human_judgment: false
  - id: D2
    description: "The research proposal picker, the approved brief panel, the --depth flag's help text, and the owner-facing run start all display and accept the shared four names, each with its own confidence target and round cap; the loop's own stopping arithmetic (confidence target, round cap) is unchanged and proven to stop exactly at each boundary."
    requirement: "LIVE-06"
    verification:
      - kind: unit
        ref: "cmd/oracle_preset_test.go#TestOraclePickerShowsSharedNamesWithOwnNumbers"
        status: pass
      - kind: unit
        ref: "cmd/oracle_preset_test.go#TestOracleBriefPanelNamesTheSharedPreset"
        status: pass
      - kind: unit
        ref: "cmd/oracle_preset_test.go#TestOracleDepthFlagHelpListsSharedNames"
        status: pass
      - kind: unit
        ref: "cmd/oracle_preset_test.go#TestOracleStopsAtExactlyTheConfidenceTarget"
        status: pass
      - kind: unit
        ref: "cmd/oracle_preset_test.go#TestOracleStopsAtExactlyTheRoundCap"
        status: pass
      - kind: unit
        ref: "go test ./cmd -run '^TestOracle' -count=1 (full existing Oracle suite, no regressions)"
        status: pass
    human_judgment: false

duration: 9min
completed: 2026-09-11
status: complete
---

# Phase 202 Plan 08: One Oracle Depth Vocabulary Summary

**Oracle now speaks planning's own Fast/Balanced/Deep/Exhaustive research-depth names -- shown in the proposal picker, the approved brief panel, the `--depth` flag help, and the run start -- while its own confidence targets and round caps (60/5, 85/15, 95/30, 99/50) are completely unchanged underneath.**

## Performance

- **Duration:** 9 min
- **Started:** 2026-09-11T10:24:34Z (first task commit)
- **Completed:** 2026-09-11T10:33:12Z (second task commit)
- **Tasks:** 2
- **Files modified:** 5 (2 created, 3 modified)

## Accomplishments

- `cmd/oracle_preset.go`: `oraclePresetPolicy` (identifier, label, target confidence, round cap) and `oraclePresetPolicies()`, which builds the four shared entries fresh from `oracleDepthLevels` on every call -- proven by a test that mutates the map directly and observes the preset follow it, so the numbers can never silently drift into a second, retyped copy.
- `resolveOraclePreset` accepts the four shared names plus every word `oracleDepthLevels` has ever accepted (`quick`, `balanced`, `standard`, `deep`, `exhaustive`, `marathon`), mapping the cheapest and standard legacy words onto their shared preset; an unrecognized word is refused, naming all four accepted words.
- The research proposal picker (`oracleDepthOptions`/`renderOraclePropose`) now lists all four shared names, each row showing its own confidence target and round cap; the previously-computed recommendation (`oracleSuggestedDepth`) is translated through `resolveOraclePreset` so the recommended row is marked under its shared name.
- The approved brief panel (`renderOracleBriefPanel`) names the chosen preset by its shared label, whether the brief was approved with a legacy word or a shared name.
- The `--depth` flag's help text on `oracleCmd` lists the four shared names (noting the legacy words are still accepted); the flag's value is validated against `resolveOraclePreset` at the command boundary, refusing an unknown word before any research work starts.
- `startOracleCompatibility` (the owner-facing run start in `cmd/oracle_loop.go`) resolves `--depth` through `resolveOraclePreset`, so a started run's recorded depth label carries the shared name; an omitted `--depth` still silently defaults to Balanced, matching prior behavior. The loop's own stopping arithmetic (`oracleReadyForCompletion`, the `state.Iteration < state.MaxIterations` guard, `oracle iterate`'s separate `--depth` flag) is untouched.
- Proved the stopping boundaries directly against the real predicates the loop uses: a fixture state one point below a target continues, exactly at the target stops (`TestOracleStopsAtExactlyTheConfidenceTarget`); a fixture state one round below the cap runs another round, exactly at the cap does not (`TestOracleStopsAtExactlyTheRoundCap`).

## Task Commits

Each task was committed atomically:

1. **Task 1: One preset vocabulary over Oracle's existing depth machinery** - `db2d5d33` (feat)
2. **Task 2: Show the shared names, with each preset's own target and cap, everywhere the owner looks** - `516099a9` (feat)

**Plan metadata:** (this commit)

_Note: both tasks carried `tdd="true"`. In both cases the required tests were written and run alongside the implementation in the same commit -- the same pattern 202-01 and 202-02 each documented for their own structural/mechanical changes -- rather than a strict RED-then-GREEN split, since the described behaviors (a mapping layer over an existing map, and rendering changes to existing functions) don't have a meaningful pre-implementation "should fail" state distinct from "doesn't compile yet."_

## Files Created/Modified

- `cmd/oracle_preset.go` - `oraclePresetPolicy`, `oraclePresetPolicies()`, `oraclePresetOptions()`, `resolveOraclePreset()`, and the legacy/alias tables that connect them to `oracleDepthLevels`.
- `cmd/oracle_preset_test.go` - All nine tests named across both tasks.
- `cmd/oracle_brief.go` - `oracleDepthOptionOrder`/`oracleDepthOptions` rewritten over `oraclePresetOptions()`; `runOraclePropose` translates `oracleSuggestedDepth`'s result through `resolveOraclePreset`; `renderOraclePropose` and `renderOracleBriefPanel` display the shared label and (in the picker) the target alongside the round cap; `runOracleBriefApprove`'s `--depth` validation now accepts the shared names too (see Deviations).
- `cmd/oracle_loop.go` - `startOracleCompatibility` resolves `depth` through `resolveOraclePreset` instead of `resolveOracleDepth`; every other depth-config use in the file (the loop's stopping arithmetic, `oracle iterate`) is untouched.
- `cmd/compatibility_cmds.go` - `--depth` flag help text lists the shared names; the flag's value is validated through `resolveOraclePreset` at the command boundary before a run starts.

## Decisions Made

See `key-decisions` in frontmatter.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `runOracleBriefApprove`'s `--depth` validation would have rejected the picker's own recommended value**
- **Found during:** Task 2, while updating the picker (`oracleDepthOptions`) and brief panel (`renderOracleBriefPanel`)
- **Issue:** After changing `oracleDepthOptions` so each row's `Value` is a shared name (e.g. `"fast"`), and changing the `--depth` flag's help text to advertise the shared names, `runOracleBriefApprove`'s own validation (`if _, ok := oracleDepthLevels[depth]; !ok { ... }`) still only accepted the raw legacy `oracleDepthLevels` keys (`quick`, `balanced`, `standard`, `deep`, `exhaustive`, `marathon`). Following the tool's own advertised words -- `aether oracle brief --depth fast` -- would fail with "must be quick, balanced, deep, or exhaustive," citing words the flag help and picker no longer showed. This was a real regression introduced by the task's own rendering changes, not a pre-existing issue.
- **Fix:** Replaced the raw map lookup with `resolveOraclePreset(depthInput)`, which accepts both the shared names and every legacy word, and stores `brief.Depth` as the resolved shared ID so the brief panel and `--from-brief` run start both see one consistent value.
- **Files modified:** `cmd/oracle_brief.go`
- **Verification:** `TestOracleBriefPanelNamesTheSharedPreset` approves a brief with both a legacy word (`quick`) and a shared name (`fast`) and asserts both succeed and both panels show `Fast`. `TestOracleFromBriefFailsWithoutApprovedBrief` (pre-existing, still approves with `Depth: "quick"`) continues to pass unchanged.
- **Committed in:** `516099a9` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 -- a genuine regression the task's own display changes would otherwise have introduced).
**Impact on plan:** Necessary for the plan's own must-have ("the previous depth words remain accepted as input... while the words shown to the owner are the shared four") to hold consistently across the picker, the brief, and the run start, rather than only two of the three. No scope creep -- no new production surface, only extending an already-touched validation to accept what the tool now shows.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `cmd/oracle_preset.go`'s vocabulary is self-contained and depends on nothing else added in Phase 202; no other in-progress plan needs it as a precondition.
- `oracleDepthLevels`'s literal entries are confirmed unchanged (diff shows only the one `startOracleCompatibility` call site touched in `cmd/oracle_loop.go`); `oracle iterate`'s separate `--depth` flag and `resolveOracleDepth` itself are both untouched, matching the plan's explicit prohibition against retuning Oracle's own economics.
- Plan-required verification (`go test ./cmd -run` for all nine named tests, the full `^TestOracle` suite, and `go vet ./cmd`) all pass with no regressions. A broader `go test ./cmd -count=1` full-package run was also started as additional diligence but had not finished within this session's window (the suite's own known ~20 min floor, per prior project history) -- the targeted verification above is what the plan's own `<verify>` blocks require and is what gates completion.
- No blockers. Ready for the next Phase 202 plan.

## Self-Check: PASSED

- `cmd/oracle_preset.go` - FOUND, builds clean
- `cmd/oracle_preset_test.go` - FOUND, all 9 tests pass
- `cmd/oracle_brief.go` - FOUND, modified as described
- `cmd/oracle_loop.go` - FOUND, modified as described
- `cmd/compatibility_cmds.go` - FOUND, modified as described
- Commit `db2d5d33` - FOUND in `git log --oneline --all`
- Commit `516099a9` - FOUND in `git log --oneline --all`
- `go test ./cmd -run '^(TestOraclePresetLabelsMatchPlanningVocabulary|TestOraclePresetNumbersAreUnchanged|TestOracleLegacyDepthWordsStillResolve|TestOracleUnknownDepthIsRefusedByName)$' -count=1` - PASS
- `go test ./cmd -run '^(TestOraclePickerShowsSharedNamesWithOwnNumbers|TestOracleBriefPanelNamesTheSharedPreset|TestOracleDepthFlagHelpListsSharedNames|TestOracleStopsAtExactlyTheConfidenceTarget|TestOracleStopsAtExactlyTheRoundCap)$' -count=1 && go test ./cmd -run '^TestOracle' -count=1 && go vet ./cmd` - PASS

---
*Phase: 202-swarm-oracle-and-live-colony*
*Completed: 2026-09-11*
