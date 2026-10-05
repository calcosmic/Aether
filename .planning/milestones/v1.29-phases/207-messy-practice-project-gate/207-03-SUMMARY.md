---
phase: 207-messy-practice-project-gate
plan: 03
subsystem: testing
tags: [journey-gate, status-guidance, expected-red-register, go-test, owner-ruling]

# Dependency graph
requires:
  - phase: 207-messy-practice-project-gate
    provides: "207-01 -- cmd/eval_gates.go's repo-root-override and schema-versioned-manifest conventions, cmd/journey_traps.go as the direct structural template this plan's register file mirrors"
provides:
  - "cmd/journey_expected_red.go -- statusGuidanceAdvisedCommands and statusGuidanceCommandsWithoutMenuWrapper, which derive what the project's own status screen advises by calling the real guidance code (loadGuidedActions, computeWarnings), never a re-typed command list"
  - "cmd/testdata/journey/expected-red.json -- the committed, schema-versioned register holding exactly one standing case: the status card advising `aether midden-review`, a command with no menu wrapper, naming Phase 208 (UED-13) as what closes it"
  - "cmd/journey_expected_red_test.go -- seven named tests proving the sixth blocker is genuinely red today, a new unregistered gap fails by name, the register may only shrink, this phase proves five fixes not six, an empty store reports no gap, two gaps report as separate ordered rows, and the check never mutates the store"
affects: [207-04-messy-practice-project-gate, 207-05-messy-practice-project-gate, 207-06-messy-practice-project-gate, 208]

# Actuals (#2632)
actuals:
  tokens: 4924
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "expected-red register as a committed, schema-versioned file mirroring cmd/journey_traps.go and cmd/eval_gates.go's manifest conventions exactly (schema_version constant, repo-root-override variable for test isolation, plain-function loader) -- the third file in this phase to follow that shape"
    - "backtick-command extraction as the one place status-warning prose is parsed back into a runtime verb: statusAdvisedCommandRe (`(aether [^`]+)`) turns 'Run `aether midden-review` to inspect.' back into 'aether midden-review', reused identically to derive both the advised-command list and (in the test file) the register's own covered-verb set from its detail prose -- one extraction mechanism, two call sites, never two."
    - "ceiling ratchet as the mirror image of evalGateSentinelFloor: journeyExpectedRedCeiling is a maximum that may only shrink (Phase 208 removing the one case), where evalGateSentinelFloor is a minimum that may only grow -- same written-reason-to-change discipline, opposite direction, because a shrinking register is the desired outcome here rather than a regression."

key-files:
  created:
    - cmd/journey_expected_red.go
    - cmd/testdata/journey/expected-red.json
    - cmd/journey_expected_red_test.go
  modified: []

key-decisions:
  - "statusGuidanceAdvisedCommands takes the command from two live sources, never a hand-typed list: every guided action's Command/AlternativeCommand (loadGuidedActions), and every backtick-quoted `aether ...` span inside the prose computeWarnings produces. Both are called against the real cmd/status.go functions with a seeded store; nothing here re-implements or approximates that logic."
  - "The wrapper-existence check resolves repoRoot separately from the guided-action workspace root (root), so TestStatusGuidanceGapsAreReportedSeparatelyAndInOrder can point the wrapper lookup at an isolated fixture directory holding a deliberate subset of wrapper files -- producing a second, genuine gap without ever touching cmd/status.go or stubbing its guidance functions."
  - "TestSixthBlockerCheckIsStillRed checks the wrapper's existence against the real repository checkout (findTestModuleRoot(t)), not a synthetic mirror -- the whole point of a standing expected-red case is that it tracks the live repository's actual state, so a fabricated fixture that never gets updated would silently stop proving anything the moment Phase 208 lands its fix without anyone noticing."
  - "journeyExpectedRedRegisteredVerbs (test-only) derives which verbs the committed register already covers by running the same backtick-extraction regex over each case's own `detail` field, rather than adding a separate `verb` field to the schema -- keeps the register's schema exactly what Task 1 specified while still letting TestNoUnregisteredStatusGuidanceGap check coverage mechanically."
  - "journeyExpectedRedCeiling is a committed Go slice (mirroring evalGateSentinelFloor's own pattern) rather than deriving 'may only shrink' purely from a stored count, so a future case addition is named by id in the failure message, not just counted."

requirements-completed: [UED-09]

coverage:
  - id: D1
    description: "The set of commands the status card actually advises is derived from the runtime's own code (loadGuidedActions + computeWarnings), the wrapper-existence check exists, and the one known gap (status card advising `aether midden-review`, no menu wrapper) is written down in a committed register naming Phase 208 (UED-13) as what closes it"
    requirement: "UED-09"
    verification:
      - kind: unit
        ref: "bash -c 'go build ./cmd/aether; go vet ./cmd/...; jq -e schema/count/closed_by checks on cmd/testdata/journey/expected-red.json; grep -c rootCmd.AddCommand cmd/journey_expected_red.go' (Task 1 <verify>)"
        status: pass
      - kind: unit
        ref: "jq -r '.cases[0].blocker' cmd/testdata/journey/expected-red.json byte-identical to the decision record's own phrase in .planning/decisions/2026-09-21-v1.29-use-it-every-day.md (verified via a Python whitespace-normalized substring check)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The sixth blocker's check runs for real against the runtime's own guidance code, is red today against the live repository, is recorded as red with what closes it, and cannot become a silent pass, an unnoticed new gap, or a claim of six -- proven by seven named tests"
    requirement: "UED-09"
    verification:
      - kind: unit
        ref: "go test -run 'TestSixthBlockerCheckIsStillRed|TestNoUnregisteredStatusGuidanceGap|TestExpectedRedRegisterOnlyShrinks|TestPhaseProvesFiveFixesNotSix|TestStatusGuidanceGapCheckHandlesAnEmptyStore|TestStatusGuidanceGapsAreReportedSeparatelyAndInOrder|TestStatusGuidanceGapCheckDoesNotMutate' -count=1 -timeout 300s ./cmd"
        status: pass
      - kind: manual_procedural
        ref: "hand-run scratch test (not committed): adding a second case to a scratch copy of the register makes the ceiling-ratchet logic fail, naming the added case id; adding a real midden-review.md wrapper to an isolated fixture root closes the gap -- proving TestSixthBlockerCheckIsStillRed is genuinely failable, not tautological"
        status: pass
    human_judgment: true
    rationale: "The two scratch-fixture demonstrations (second case rejected by name; wrapper presence closes the gap) prove the test suite's own genuineness -- that TestSixthBlockerCheckIsStillRed and TestExpectedRedRegisterOnlyShrinks can actually fail, not just that they currently pass. This is exactly the property CLAUDE.md's Definition of Done requires ('a test must be able to fail') but by its nature cannot be re-asserted as a permanent committed test without either mutating the real repository checkout mid-suite or paying the cost of a second fixture-repo build on every run; a human (or a future session) re-running the same two scratch checks is the honest way to re-confirm it."

duration: 45min
completed: 2026-09-22
status: complete
---

# Phase 207 Plan 03: The Sixth Blocker's Check, Built and Honestly Red Summary

**`cmd/journey_expected_red.go` derives what `aether status` actually advises by calling the real guidance code (never a re-typed command list), proves the status card's advice to run `aether midden-review` has no menu wrapper, and records that one gap in a committed register naming Phase 208 as what closes it -- backed by seven tests that keep it from ever becoming a silent pass, an unnoticed new gap, or a claim of six.**

## Performance

- **Duration:** 45 min
- **Tasks:** 2
- **Files created:** 3
- **Files modified:** 0

## Accomplishments

- `cmd/journey_expected_red.go`: `statusGuidanceAdvisedCommands` calls `loadGuidedActions` and `computeWarnings` directly against a seeded store -- never parsing `cmd/status.go`'s source or re-typing a command list -- and extracts every `aether ...` command those two real functions produce, in the exact order `aether status` itself would show them. `statusGuidanceCommandsWithoutMenuWrapper` checks each advised verb against `.claude/commands/ant/<verb>.md` (the same path convention `TestDeclaredAliasesHaveWrappersOnEveryPlatform` uses for the Claude surface) and returns, in order, the verbs with no wrapper.
- `cmd/testdata/journey/expected-red.json`: the schema-versioned register (`journey-expected-red/v1`) holding exactly one case -- `status-card-advises-a-menu-command-that-does-not-exist` -- whose `blocker` field is byte-identical to the owner's own decision-record wording, and whose `closed_by` names `Phase 208 (UED-13)`.
- `cmd/journey_expected_red_test.go`: seven tests. `TestSixthBlockerCheckIsStillRed` proves the gap is genuinely red against the real, live repository checkout today (with an explicit in-code note that the correct response to it going green is deleting the register entry, never loosening the test). `TestNoUnregisteredStatusGuidanceGap` fails by name the day any advised command loses its wrapper without a matching register case. `TestExpectedRedRegisterOnlyShrinks` is the ceiling-ratchet mirror of `evalGateSentinelFloor` -- the register may never gain a case. `TestPhaseProvesFiveFixesNotSix` asserts exactly one case, naming Phase 208. `TestStatusGuidanceGapCheckHandlesAnEmptyStore`, `TestStatusGuidanceGapsAreReportedSeparatelyAndInOrder` (two genuine gaps via an isolated fixture wrapper directory, never by stubbing `cmd/status.go`), and `TestStatusGuidanceGapCheckDoesNotMutate` round out the backstop truths from the plan's `must_haves`.
- Hand-verified (not committed): a scratch copy of the register with a second case added makes the ceiling ratchet fail, naming the added id; a scratch fixture root carrying the real `midden-review.md` wrapper closes the gap entirely -- proving the committed tests are genuinely failable in both directions, not tautological.

## Task Commits

1. **Task 1: Derive what the status card advises, and record the one known gap** - `ca679d31` (feat)
2. **Task 2: The sixth case is red, is recorded as red, and cannot quietly stop being either** - `d6afa269` (test)

_No separate plan-metadata commit yet -- STATE.md/ROADMAP.md/REQUIREMENTS.md are updated and committed after this file is written, per the executor's atomic close-out order._

## Files Created/Modified

- `cmd/journey_expected_red.go` - `statusGuidanceAdvisedCommands`, `statusGuidanceCommandsWithoutMenuWrapper`, `journeyExpectedRedPath`/`SchemaVersion`/`RepoRootOverride`, `journeyExpectedRedCase`/`File`, `loadJourneyExpectedRed`
- `cmd/testdata/journey/expected-red.json` - the one committed standing expected-red case, naming Phase 208
- `cmd/journey_expected_red_test.go` - the seven named tests plus `journeyExpectedRedCeiling` and `journeyExpectedRedRegisteredVerbs`

## Decisions Made

See `key-decisions` in the frontmatter above -- deriving advised commands from two live sources rather than a typed list, separating `repoRoot` from the guided-action workspace `root` so the second-gap test never touches `cmd/status.go`, checking `TestSixthBlockerCheckIsStillRed` against the real repository rather than a synthetic mirror, deriving registered-verb coverage from the register's own `detail` prose, and the ceiling-ratchet shape mirroring `evalGateSentinelFloor` in the opposite direction.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The sixth blocker's check exists, runs, and is honestly red against the live repository -- ready for Plan 05's revert-and-rerun proof of the five landed fixes, and ready for Phase 208 to close this one case.
- UED-09 is shared with plans 207-05 and 207-06 (per this session's own instructions); `gsd-tools.cjs query requirements.ready-ids` reports it `blocked`, not `ready` -- this plan's close-out does NOT mark UED-09 complete in REQUIREMENTS.md.

---
*Phase: 207-messy-practice-project-gate*
*Completed: 2026-09-22*

## Self-Check: PASSED
