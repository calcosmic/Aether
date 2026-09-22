---
phase: 208-never-a-dead-end
plan: 07
subsystem: cli
tags: [go, journey, refusal, error-handling, cli]

# Dependency graph
requires:
  - phase: 208-01
    provides: "The typed refusal contract (refuse(), refusalRegistry, renderRefusal -- both exit lanes) this plan's extractor keys on"
  - phase: 208-06
    provides: "27 registry rows and the behavioural proof that the table and the program's real behaviour agree, which this plan's extractor fixtures render for real"
provides:
  - "journeyPrintedRefusals(transcriptPath) -- finds every real renderRefusal block a step's own transcript printed, keyed on the stable `Next: \`cmd\`` marker"
  - "printedCommandToRuntimeCommand(printed) -- reverses the /ant-<verb> menu mapping through the same wrapperCommandNames table the forward mapping (platformCommandName) reads"
  - "journeyRunPrintedNextCommands -- runs every printed next command for real, as a subprocess in the practice project, after a step's own on-disk fact check passes; fails the step on four named conditions (unmapped, timed out, unknown command/flag, empty-next refusal)"
  - "journeyStepResult.RefusalsPrinted / .NextCommandsRun, carried into journeyReportSummary so a clean run says it met none rather than reading as a silent pass"
affects: [208-08]

# Actuals (#2632)
actuals:
  tokens: 6874
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A refusal's stable printed marker is its exact rendered text (\"Next: `cmd`\" on its own line), not a banner or any other structural cue -- proven by a mutation that drops the label requirement and watches the ordinary-output test fail"
    - "A reverse mapping reads the exact table the forward mapping reads (wrapperCommandNames) rather than re-deriving its own copy, so the two can never drift apart"
    - "Pure classification logic (journeyNextCommandFailureReason) lives in the untagged cmd/journey.go, not the //go:build journey harness file, specifically so it can be unit-tested with captured output in the ordinary suite -- the live-only file only orchestrates the real subprocess call"

key-files:
  created:
    - cmd/refusal_printed_test.go
  modified:
    - cmd/journey.go
    - cmd/journey_live_test.go
    - cmd/journey_test.go

key-decisions:
  - "The extractor's marker is exactly renderRefusal's own \"Next: `<command>`\" line (matched with `(?m)^Next: `...`), not the banner -- the plan's own action text names the Next: line's label plus the backticked command as the stable marker, and the mutation proof (dropping the label requirement) confirms it is load-bearing."
  - "journeyNextCommandFailureReason and its supporting regex/constant were placed in the untagged cmd/journey.go rather than the //go:build journey cmd/journey_live_test.go, so the plan's own required unit-level test over the four failure conditions runs in the ordinary suite (no live chat, no build tag) -- journeyRunPrintedNextCommands, which needs *testing.T and a real subprocess, stays in the live-only file and calls the shared classifier."
  - "journeyRunPrintedNextCommands takes caps journeyCaps as an explicit parameter (beyond the plan's artifact-table listing) because the per-command timeout must come from the step's own journeyCaps.WallClockSecs, not a new constant -- the artifact table names the essential parameters, not a literal final signature."
  - "NextCommandsRun counts only commands actually executed as a subprocess (mapped=true); an unmapped printed command is a same-step failure before any subprocess runs, so it is not counted as \"run\"."

requirements-completed: [UED-10]

coverage:
  - id: D1
    description: "journeyPrintedRefusals finds every real renderRefusal block a transcript printed (one, two in order, none in ordinary prose), keyed on the stable Next: line marker rather than any backticked command"
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/refusal_printed_test.go#TestPrintedRefusalExtractorFindsARealRefusal"
        status: pass
      - kind: unit
        ref: "cmd/refusal_printed_test.go#TestPrintedRefusalExtractorIgnoresOrdinaryOutput"
        status: pass
    human_judgment: false
  - id: D2
    description: "printedCommandToRuntimeCommand reverses a menu-form /ant-<verb> command back to aether <verb>, arguments preserved, through the same wrapperCommandNames table the forward mapping reads; an already-runtime-form command survives unchanged; a verb in neither form returns not-ok"
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/refusal_printed_test.go#TestMenuFormNextCommandMapsBackToTheRuntimeCommand"
        status: pass
    human_judgment: false
  - id: D3
    description: "During a real journey step, every refusal the step's own transcript printed has its mapped next command executed as a real subprocess in the practice project, after the step's own on-disk fact check has already passed; the step fails, naming the command and reason, on any of four conditions (unmapped, timed out, unknown command/flag, empty-next refusal)"
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/journey_test.go#TestJourneyNextCommandFailureReason"
        status: pass
    human_judgment: true
    rationale: "The classification helper the step's pass/fail decision is built on is fully unit-tested against captured output; the live orchestration itself (journeyRunPrintedNextCommands actually spawning `aether <verb>` inside a real practice project, wired into journeyDriveStep after the fact check) can only be proven end to end by a real, money-spending journey run, which is explicitly out of scope for this plan (208-08's job) -- CLAUDE.md's own instruction for this plan."
  - id: D4
    description: "journeyStepResult.RefusalsPrinted / NextCommandsRun are carried into the report, and journeyReportSummary names both totals per trial in plain words -- a run that met no refusals says exactly that"
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/journey_test.go#TestJourneyReportNamesHowManyNextCommandsRan"
        status: pass
      - kind: unit
        ref: "cmd/journey_test.go#TestJourneyReportSummaryNamesZeroRefusalsHonestly"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-09-22
status: complete
---

# Phase 208 Plan 07: A Refusal's Way Out Is Executed, Not Just Printed Summary

**The journey no longer trusts that a refusal's printed next command actually works -- it now finds every refusal a real chat's transcript printed and runs the command it named, for real, in the same practice project, failing the step by name when that command turns out to be unmapped, unknown, timed out, or itself a dead end.**

For the owner, in plain English: when Aether tells you "run this command next" after refusing to do something, this plan makes the practice-project test actually TYPE that command and check it works -- instead of just trusting that the words printed on the screen were correct. If the suggested command doesn't exist, is misspelled, takes too long, or itself leads nowhere, the test now catches that and says so, naming the exact command and what went wrong. No money was spent proving this -- the actual live run happens in the next piece of work.

## Performance

- **Duration:** 55 min
- **Started:** 2026-09-22T20:12:00Z (approx, first file read)
- **Completed:** 2026-09-22T21:07:00Z
- **Tasks:** 2
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments

- `cmd/journey.go`: `journeyPrintedRefusals(transcriptPath)` walks a real transcript (assistant text blocks and user-role tool_result blocks, the same two shapes `journeyMenuCommandNames` and `owedScreenFrom` already read) and finds every real `renderRefusal` block by its stable `Next: \`cmd\`` marker -- proven never to be fooled by an ordinary backticked command in prose.
- `cmd/journey.go`: `printedCommandToRuntimeCommand(printed)` reverses the `/ant-<verb>` menu mapping back to `aether <verb>`, arguments preserved byte for byte, reading `wrapperCommandNames` directly -- the same table `platformCommandName` reads to produce the menu form in the first place, so the two mappings can never drift apart.
- `cmd/journey_live_test.go`: `journeyDriveStep` now calls `journeyRunPrintedNextCommands` immediately after a step's own on-disk fact check passes. Every printed refusal's mapped next command is run as a real `aether <verb>` subprocess inside the practice project (`binDir` first on `PATH`, `AETHER_OUTPUT_MODE=visual` set, a timeout sized from the step's own `journeyCaps.WallClockSecs`), and the step fails -- naming the command verbatim and the reason -- on any of four conditions: the printed command could not be mapped; the subprocess timed out; it reported an unknown command or flag; or its own output was itself a refusal with nothing to run next. A non-zero exit matching none of those four is left alone, since a recovery command may legitimately report more work outstanding.
- `cmd/journey.go`: `journeyStepResult` gains `RefusalsPrinted` / `NextCommandsRun`; `journeyReportSummary` gains one line per trial naming both totals -- a trial that met no refusals reads "0 printed refusal(s) found, 0 next command(s) run", never a silent gap.
- `cmd/journey_test.go` / `cmd/refusal_printed_test.go`: the offline suite (no build tag, no live chat) covers the extractor, the reverse mapping, and -- critically -- the pure `journeyNextCommandFailureReason` classification helper against all four named conditions plus the "not a failure" case, using captured/rendered output rather than a live run.

## Task Commits

Each task was committed atomically:

1. **Task 1: Find the refusals a real chat printed, and map a menu-form next command back to the command that runs** - `d09906e9` (feat)
2. **Task 2: Run every printed next command for real, in the practice project, as part of the step** - `9c8744be` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified

- `cmd/journey.go` - `journeyPrintedRefusal`, `journeyPrintedRefusalNextLineRe`, `journeyPrintedRefusals`, `printedCommandToRuntimeCommand`, `journeyPrintedRefusalEmptyNextCommandRe`, `journeyNextCommandTimeoutFraction`, `journeyNextCommandFailureReason`, `journeyStepResult.RefusalsPrinted`/`.NextCommandsRun`, `journeyReportSummary`'s new per-trial totals line, `journeyContentBlock.Content` (extends the existing reader to carry a `tool_result` block's own payload, read via the existing `toolResultText` helper from `cmd/hook_cmds.go`)
- `cmd/journey_live_test.go` - `journeyDriveStep` wired to call `journeyRunPrintedNextCommands` after the on-disk fact check; `journeyRunPrintedNextCommands` itself (the live subprocess runner)
- `cmd/journey_test.go` - `TestJourneyReportNamesHowManyNextCommandsRan`, `TestJourneyReportSummaryNamesZeroRefusalsHonestly`, `TestJourneyNextCommandFailureReason`
- `cmd/refusal_printed_test.go` (new) - `journeyWriteRefusalTranscript` (derives a transcript fixture from the real captured session, varying only the tool_result's own content to a real `renderRefusal(...)` call's output), `TestPrintedRefusalExtractorFindsARealRefusal`, `TestPrintedRefusalExtractorIgnoresOrdinaryOutput`, `TestMenuFormNextCommandMapsBackToTheRuntimeCommand`

## Decisions Made

See `key-decisions` in the frontmatter above -- summarised: the extractor's marker is exactly the `Next:` line, not the banner; the pure classification helper lives in the untagged file specifically so it is unit-testable without a live chat; `journeyRunPrintedNextCommands` takes an explicit `caps` parameter beyond the plan's artifact-table listing, because the per-command timeout must come from the step's own caps; `NextCommandsRun` counts only commands that actually ran as a subprocess.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] The classification helper was first written into the `//go:build journey` file, where the plan's own required offline unit test could never reach it**
- **Found during:** Task 2, before running the plan's own verification command
- **Issue:** My first draft placed `journeyNextCommandFailureReason` and its supporting regex/constant inside `cmd/journey_live_test.go` (`//go:build journey`). The plan's Task 2 acceptance criteria explicitly require "The four failure conditions are each covered by a unit-level test over the classification helper, driven with captured subprocess output rather than a live chat" -- but a test for a symbol that only exists under the `journey` build tag cannot run in the plan's own verification command (`go test ./cmd -run '...' -count=1 -timeout 8m`, which never passes `-tags=journey`), and the plan's own harness instructions never permit running the live-tagged suite (that spends money).
- **Fix:** Moved `journeyPrintedRefusalEmptyNextCommandRe`, `journeyNextCommandTimeoutFraction`, and `journeyNextCommandFailureReason` into the untagged `cmd/journey.go`, alongside the rest of the printed-refusal machinery. `journeyRunPrintedNextCommands` (which genuinely needs `*testing.T` and a real subprocess, and is only ever called from the live harness) stays in `cmd/journey_live_test.go` and calls the shared classifier.
- **Files modified:** cmd/journey.go, cmd/journey_live_test.go
- **Verification:** `TestJourneyNextCommandFailureReason` (cmd/journey_test.go, no build tag) covers all four named conditions plus the not-a-failure case and passes in the ordinary suite; `go test -tags=journey -run=NoSuchTest -count=1 ./cmd` still compiles the live harness.
- **Committed in:** 9c8744be (folded into the task's single commit; caught before the first commit for this task, so no separate revert was needed)

---

**Total deviations:** 1 auto-fixed (1 Rule 1 placement bug, caught during my own verification before committing). **Impact on plan:** No scope or architecture change -- the fix only relocates two pieces of pure logic to the file where the plan's own required test can actually reach them; behaviour is identical either way.

## Mutation Proof

Performed manually during Task 1 verification (temporary edit -> confirm the named test fails -> revert -> confirm the test passes again). The mutation was never committed.

**Dropped the `Next:` label requirement from the extractor's marker regex:** temporarily changed `journeyPrintedRefusalNextLineRe` from `` (?m)^Next: `([^`\n]+)` `` to `` `([^`\n]+)` `` (matching ANY backticked command anywhere, not just one preceded by the `Next:` label). Result: `TestPrintedRefusalExtractorIgnoresOrdinaryOutput` failed --
`journeyPrintedRefusals(...) = [{NextCommand:aether status Raw:\`aether status\`}], want zero -- a backticked command in ordinary prose is not a refusal`.
Reverted (`cp` from a pre-mutation backup); rebuilt and re-ran the full targeted test set (`TestPrintedRefusalExtractor*`, `TestMenuFormNextCommandMapsBackToTheRuntimeCommand`, `TestJourney*`) -- all pass again.

## Issues Encountered

None beyond the deviation above, caught and fixed before committing.

**This plan spends no money.** Per the harness's own explicit instruction, no live journey run (`make eval-gate-journey`, `TestJourney` with a real `claude -p`, or anything that spends money) was executed. Every test run in this plan's own verification is offline: the offline `cmd/journey_test.go` / `cmd/refusal_printed_test.go` suite (real fixtures derived from a captured transcript and real `renderRefusal(...)` calls, but no subprocess spawn and no chat call), plus a compile-only check of the `//go:build journey` live harness (`go test -tags=journey -run=NoSuchTest`). The real, money-spending journey run that actually exercises `journeyRunPrintedNextCommands` against a live chat is 208-08's job.

## Known Stubs

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

`journeyPrintedRefusals`, `printedCommandToRuntimeCommand`, and `journeyRunPrintedNextCommands` are all in place, wired into `journeyDriveStep`, and proven offline against real rendered fixtures. Phase 208 plan 08 (per this plan's own `affects` list) can now run the real, three-trial, money-spending journey and expect this machinery to genuinely execute every refusal's printed way out inside it -- no further scaffolding is needed from this plan.

No blockers.

---
*Phase: 208-never-a-dead-end*
*Completed: 2026-09-22*

## Self-Check: PASSED

- The 1 file listed under key-files.created (`cmd/refusal_printed_test.go`) verified present on disk (`[ -f ]`).
- Both task commit hashes (d09906e9, 9c8744be) verified present in `git log --oneline --all`.
- Re-ran acceptance-criteria tests: `TestPrintedRefusalExtractorFindsARealRefusal`, `TestPrintedRefusalExtractorIgnoresOrdinaryOutput`, `TestMenuFormNextCommandMapsBackToTheRuntimeCommand`, `TestJourneyReportNamesHowManyNextCommandsRan`, `TestJourneyReportSummaryNamesZeroRefusalsHonestly`, `TestJourneyNextCommandFailureReason` -- all pass.
- Re-ran plan-level `<verification>`: `go build ./cmd/aether` and `go vet ./cmd` both clean; `go test ./cmd -run 'TestJourney|TestPrintedRefusal|TestMenuFormNextCommand' -count=1 -timeout 8m` passes; `go test -tags=journey -run=NoSuchTest -count=1 ./cmd` compiles the live harness without running it; no paid chat run occurred.
- Wider regression sweep also run and passing: `go test ./cmd -run 'TestRefusal|TestFriendlyError|TestBehaviourMatchesTheRefusalTable|TestNoRefusalBothWarnsAndStops|TestEveryRefusal' -count=1 -timeout 8m`.
