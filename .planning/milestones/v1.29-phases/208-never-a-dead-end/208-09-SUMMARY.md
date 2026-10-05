---
phase: 208-never-a-dead-end
plan: 09
subsystem: refusal-lifecycle
tags: [go, refusal-contract, journey-harness, colonize-wrapper, cli]

# Dependency graph
requires:
  - phase: 208-never-a-dead-end (plans 01-08)
    provides: the typed refusal contract (cmd/refusal.go), the printed-refusal
      journey extractor (cmd/journey.go, cmd/journey_live_test.go), and the
      one real journey walk's own transcript and findings (208-JOURNEY-RUN.md,
      WINDOWS rows 53/55)
provides:
  - "sessionHasNoOneToAsk (cmd/unattended_session.go): the one opt-in
    AETHER_UNATTENDED=1 fact both refusal lanes read"
  - "The shared act-when-alone guidance sentence on Error()'s one-line
    host-relayed form and renderRefusal's drawn block, gated on
    ProtectsWork AND a next command AND the fact"
  - "The journey harness setting AETHER_UNATTENDED=1 on every claude child
    it spawns, in journeyRunClaudeWithRetry"
  - "D-01's act-when-alone / ask-when-present rule stated identically in
    all four hand-kept colonize command sources (yaml + 3 wrapper copies)"
  - "journeyPrintedRefusals finding the host-relayed one-line refusal form,
    not just the drawn block form, deduplicated across both"
  - "WINDOWS.md row 55 closed with the owner's ruling recorded as reason"
affects: [208-10-PLAN.md]

# Actuals (#2632)
actuals:
  tokens: 12962
  tasks: 3
  commits: 5

tech-stack:
  added: []
  patterns:
    - "Fail-safe opt-in environment fact (AETHER_UNATTENDED=1), never inferred from terminal/signal state"
    - "One shared guidance-sentence constant read by both the plain-text and drawn-screen refusal lanes"
    - "Positional merge-then-dedupe extraction across two regex forms describing the same underlying event"

key-files:
  created:
    - cmd/unattended_session.go
    - cmd/unattended_refusal_test.go
    - cmd/testdata/refusals/host-relay-refusal.txt
  modified:
    - cmd/refusal.go
    - cmd/journey.go
    - cmd/journey_live_test.go
    - cmd/refusal_printed_test.go
    - cmd/lifecycle_wrapper_contract_test.go
    - .aether/commands/colonize.yaml
    - .claude/commands/ant/colonize.md
    - .claude/commands/ant-colonize.md
    - .opencode/commands/ant/colonize.md
    - .planning/WINDOWS.md

key-decisions:
  - "The is-anyone-here fact is deliberately opt-in (AETHER_UNATTENDED=1), never inferred from a terminal check or an unobserved Claude Code signal -- D-01's own reasoning, restated in sessionHasNoOneToAsk's doc comment."
  - "The guidance sentence is gated on ProtectsWork AND a non-empty next command AND the fact, on both refusal lanes, from one shared string constant -- never two copies that could drift."
  - "The relayed one-line extractor accepts a match only when the text after the marker is itself a runtime invocation or menu command (printedCommandToRuntimeCommand); an unmappable tail is treated as ordinary prose, not a hard failure -- deliberately different from the block form's stricter contract."
  - "colonize.md is a FOUR-way hand-kept copy, not three: canonical nested Claude, the flat installed-consumer mirror (.claude/commands/ant-colonize.md), OpenCode, and the YAML source of truth. This plan's Task 2 only updated three; the fourth was caught by the full suite and fixed in the same wave (commit 5590843f), and TestColonizeWrapperCarriesTheActWhenAloneRule now also names the mirror."

patterns-established:
  - "A wrapper-copy count test's own missing check first: before asserting parity, grep for every test file that enumerates 'wrapper copies' or 'mirror' for a given verb, since a hand-kept N-way set can silently be N+1."

requirements-completed: [UED-10]

coverage:
  - id: D1
    description: "An unattended chat is told, by the program itself, to run a work-protecting stop's named next command and carry on; an attended chat sees byte-identical output to before this plan."
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/unattended_refusal_test.go#TestUnattendedRefusalNamesTheWayPastOnBothLanes"
        status: pass
      - kind: unit
        ref: "cmd/unattended_refusal_test.go#TestAttendedRefusalTextIsUnchanged"
        status: pass
      - kind: unit
        ref: "cmd/unattended_refusal_test.go#TestOnlyAWorkProtectingStopCarriesTheGuidance"
        status: pass
    human_judgment: false
  - id: D2
    description: "The is-anyone-here fact has exactly one reader in the codebase, and the journey harness is the one place that sets it on every claude child process it spawns."
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/unattended_refusal_test.go#TestTheIsAnyoneHereFactHasOneReader"
        status: pass
      - kind: unit
        ref: "cmd/unattended_refusal_test.go#TestUnattendedFactIsSetByTheJourneyHarness"
        status: pass
    human_judgment: false
  - id: D3
    description: "The act-when-alone / ask-when-present rule is stated in all four hand-kept colonize command sources, with a test that fails naming whichever one drops it."
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/unattended_refusal_test.go#TestColonizeWrapperCarriesTheActWhenAloneRule"
        status: pass
    human_judgment: false
  - id: D4
    description: "The journey's printed-refusal extractor finds a refusal relayed as a single line by the TypeScript host, not only the drawn block form, proven against a real captured fixture and deduplicated across both forms."
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/refusal_printed_test.go#TestPrintedRefusalExtractorFindsAHostRelayedRefusal"
        status: pass
      - kind: unit
        ref: "cmd/refusal_printed_test.go#TestPrintedRefusalExtractorIgnoresRelayedProse"
        status: pass
      - kind: unit
        ref: "cmd/refusal_printed_test.go#TestPrintedRefusalsAreDeduplicatedWithinOneTranscript"
        status: pass
    human_judgment: false
  - id: D5
    description: "WINDOWS.md row 55 is closed with the owner's D-01 ruling recorded as its reason, surviving a fresh ledger read."
    verification:
      - kind: other
        ref: "node ~/.claude/gsd-core/bin/gsd-tools.cjs query windows status (row 55 status=fixed, non-empty reason, confirmed after a later windows-touching commit)"
        status: pass
    human_judgment: false

duration: 1h 41m (dominated by an ~79-minute full ./cmd suite run under heavy host contention from a concurrent session)
completed: 2026-09-23
status: complete
---

# Phase 208 Plan 09: Act When Alone, Ask When You're There Summary

**An opt-in `AETHER_UNATTENDED=1` fact now tells a work-protecting refusal's own text — on both the plain-text and drawn-screen lanes — to name its way past and carry on rather than wait for an owner who isn't there; the journey harness sets that fact, all four hand-kept colonize command copies state the rule, and the printed-refusal extractor now finds a refusal relayed as one line as well as a drawn block.**

## Performance

- **Duration:** 1h 41m (14:09-15:50 CEST); the code and tests themselves took under 15 minutes across three task commits — the remainder was one full, unscoped `go test ./cmd -count=1 -timeout 90m` run (4749.86s / ~79 minutes), slowed by CPU/I/O contention from another concurrent session's own `go build`/`go test ./...` activity in this same checkout, plus two small post-suite fixes.
- **Started:** 2026-09-23T14:09:13+02:00
- **Completed:** 2026-09-23T15:49:55+02:00
- **Tasks:** 3 planned, all completed
- **Commits:** 5 (3 task commits + 2 fixes surfaced by the full suite, all tagged `208-09`)
- **Files modified:** 13

## Accomplishments

- `cmd/unattended_session.go` adds `sessionHasNoOneToAsk()`, reading the one opt-in `AETHER_UNATTENDED` fact (accepted only at exactly `"1"`, fail-safe toward "someone is present" on any other value including unset) and the one shared guidance-sentence constant both refusal lanes read.
- `cmd/refusal.go`'s `Error()` (the TS-host-relayed one-line form) and `renderRefusal` (the drawn block) both carry the guidance sentence only when the fact is set AND the refusal protects work AND names a next command — attended output (the ordinary, default case) is proven byte-for-byte unchanged.
- `cmd/journey_live_test.go`'s `journeyRunClaudeWithRetry` — the one place both the session-establishing call and every driven step build their child-process env — now sets `AETHER_UNATTENDED=1` on every `claude` invocation the harness makes.
- D-01's act-when-alone / ask-when-present rule replaces the old ambiguous "follow the runtime recovery guidance" sentence in all four hand-kept colonize command sources: `.aether/commands/colonize.yaml`, `.claude/commands/ant/colonize.md`, `.claude/commands/ant-colonize.md` (the flat installed-consumer mirror — see Deviations), and `.opencode/commands/ant/colonize.md`.
- `cmd/journey.go`'s `journeyPrintedRefusals` now also matches `refusal.Error()`'s relayed one-line marker (`— next: <command>`) alongside the existing drawn-block `Next: \`...\`` form, in the same user-role `tool_result` trust boundary; an unmappable relayed tail is treated as ordinary prose, never a hard failure; distinct commands dedupe across both forms in first-seen order.
- `.planning/WINDOWS.md` row 55 is marked fixed with the owner's D-01 ruling recorded as the reason, written through the ledger's own parse/render functions (never a hand-edited table cell) and confirmed to survive a later `windows`-touching commit.

## Task Commits

Each task was committed atomically, plus two fixes the full suite surfaced:

1. **Task 1: An unattended chat is told to run the way out itself** - `1f7db85c` (feat)
2. **Task 2: State the rule where the chat reads it, close the register row** - `3b1bf496` (docs)
3. **Task 3: The journey can find a refusal that arrives as one relayed line** - `e47a96c1` (feat)
4. **Fix: carry the rule into the flat colonize mirror** - `5590843f` (fix)
5. **Fix: update the CMD-04 hash fence for colonize.md's new text** - `8a5e072f` (fix)

## Files Created/Modified

- `cmd/unattended_session.go` - `sessionHasNoOneToAsk()` and the shared guidance-sentence constant
- `cmd/unattended_refusal_test.go` - the six named tests for Tasks 1-2 (five is-anyone-here tests plus the colonize-wrapper rule test)
- `cmd/testdata/refusals/host-relay-refusal.txt` - real captured host-relay text, copied verbatim from the 208-08 journey walk's own on-disk transcript
- `cmd/refusal.go` - guidance-sentence insertion on both refusal lanes, gated on the fact
- `cmd/journey.go` - the relayed-line pattern, positional merge with the block-form pattern, cross-form dedup
- `cmd/journey_live_test.go` - `AETHER_UNATTENDED=1` set in `journeyRunClaudeWithRetry`
- `cmd/refusal_printed_test.go` - three new tests for the relayed-form extractor
- `cmd/lifecycle_wrapper_contract_test.go` - updated CMD-04 frozen hash for colonize.md's new content (both platform copies), with a stated reason
- `.aether/commands/colonize.yaml` - the rule inserted after orchestration step 1
- `.claude/commands/ant/colonize.md`, `.claude/commands/ant-colonize.md`, `.opencode/commands/ant/colonize.md` - the existing-survey sentence replaced with D-01's ruling, identically in all three
- `.planning/WINDOWS.md` - row 55 marked fixed with the owner's ruling as reason

## Decisions Made

See `key-decisions` in frontmatter. In brief: the is-anyone-here fact is opt-in and fail-safe by construction (D-01); the guidance sentence is a single shared string read by both lanes; the relayed-form extractor is deliberately more forgiving than the block-form extractor (silence, not failure, on an unmappable tail, because the relayed form has no delimiter promising a command was ever there); and colonize.md turned out to be a four-way hand-kept copy, not three.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue, caught by the plan's own full-suite verification] Task 2's wrapper edit landed in only three of colonize.md's four hand-kept copies**
- **Found during:** the plan-level full `go test ./cmd` run (not during Task 2 itself — the scoped Task 2 verification command only exercises the three files the plan's own `files_modified` list named)
- **Issue:** `TestLifecycleFlatMirrorsMatchCanonical` and `TestPlanAndColonizeWrappersAreByteIdentical` both failed. `colonizeWrapperTripletPaths` (`cmd/lifecycle_wrapper_contract_test.go`) names a fourth copy neither the plan's `files_modified` nor its `read_first` list mentioned: `.claude/commands/ant-colonize.md`, the flat installed-consumer mirror `aether install`/`aether update` write from the canonical nested source. Task 2 updated the canonical nested Claude copy, the OpenCode copy, and the YAML source — never this flat mirror — so it still carried the old ambiguous sentence.
- **Fix:** the flat mirror was synced byte-for-byte with the other three, and `TestColonizeWrapperCarriesTheActWhenAloneRule` was extended to name the mirror as a fourth file, so a future two-copy edit is caught by name rather than by a separate, differently-worded parity test.
- **Files modified:** `.claude/commands/ant-colonize.md`, `cmd/unattended_refusal_test.go`
- **Verification:** `TestColonizeWrapperCarriesTheActWhenAloneRule` now runs a fourth subtest (`claude-flat-mirror`), shown able to fail by name when the rule was removed from that file alone; `TestLifecycleFlatMirrorsMatchCanonical` and `TestPlanAndColonizeWrappersAreByteIdentical` both re-verified green in isolation.
- **Committed in:** `5590843f`

**2. [Rule 1 - Bug, caught by the plan's own full-suite verification] The CMD-04 content-hash fence for colonize.md went stale**
- **Found during:** the plan-level full `go test ./cmd` run
- **Issue:** `TestSpecialistCommandSurfacesUnchanged` pins seventeen specialist command surfaces (including both platform copies of colonize.md) by SHA-256, deliberately brittle by design ("CMD-04 requires this command to keep working unchanged through milestone v1.25... a legitimate future edit updates the recorded hash in the same commit with a stated reason"). Task 2's wording change to colonize.md was intentional, so this hash needed updating — the test's own failure message names the exact mechanism.
- **Fix:** recomputed the SHA-256 of the new `.claude/commands/ant/colonize.md` / `.opencode/commands/ant/colonize.md` content (`7c8b3918aeace20d166691154e7d1991cceabe054aecad0b2971d59db0f05ce9`, identical for both since they are byte-identical), updated both map entries in `specialistCommandSurfaceHashes`, and added a dated, stated-reason comment following the ledger's own convention.
- **Files modified:** `cmd/lifecycle_wrapper_contract_test.go`
- **Verification:** `TestSpecialistCommandSurfacesUnchanged` green, including its `command_guide_reachability` subtest.
- **Committed in:** `8a5e072f`

---

**Total deviations:** 2 auto-fixed (Rule 3 — a missed fourth hand-kept copy; Rule 1 — a stale content-hash fence, both direct, expected consequences of Task 2's own intentional content change, both caught only by the plan's own full-suite verification step and fixed in the same execution wave).
**Impact on plan:** None on scope. Both fixes are exactly what the plan's own Task 2 acceptance criteria call for ("the two wrappers... must land identically", CMD-04's "a legitimate future edit updates the recorded hash... with a stated reason") applied to a fourth copy the plan's own file list under-counted. No runtime behavior outside colonize.md's own wording changed.

## Mutation Proof: Which Test Caught Which Reverted Change

Per CLAUDE.md's Definition of Done, each of the five Task 1 tests was shown able to fail by temporarily reverting its own change, confirming the failure names the test, then restoring:

| Reverted change | Test that failed | Failure text (abbreviated) |
|---|---|---|
| `Error()`'s guidance-sentence insertion removed | `TestUnattendedRefusalNamesTheWayPastOnBothLanes` | "Error() with the fact set on a work-protecting row does not carry the shared guidance sentence" |
| `ProtectsWork` gate removed from both lanes (guidance always inserted when the fact is set, regardless of disposition) | `TestOnlyAWorkProtectingStopCarriesTheGuidance` | "a non-work-protecting row's Error() must never carry the guidance sentence, even with the fact set" |
| `sessionHasNoOneToAsk()` hard-coded to return `true` | `TestAttendedRefusalTextIsUnchanged` | "sessionHasNoOneToAsk() is true with the variable unset -- fail-safe direction violated" |
| A second `os.Getenv("AETHER_UNATTENDED")` reader added in a scratch file | `TestTheIsAnyoneHereFactHasOneReader` | "found it also in: [zz_temp_second_reader.go]" (named the exact offending file) |
| `unattendedEnvVar+"=1"` removed from `journeyRunClaudeWithRetry`'s env | `TestUnattendedFactIsSetByTheJourneyHarness` | "journey_live_test.go no longer sets unattendedEnvVar+\"=1\" on the claude child's env" |

Task 2's `TestColonizeWrapperCarriesTheActWhenAloneRule` was shown able to fail by deleting the rule from `.opencode/commands/ant/colonize.md` alone: the failure named that exact path and listed all four missing phrase fragments, then (after Deviation 1 above) was re-proven against the fourth `claude-flat-mirror` file the same way.

Task 3's `TestPrintedRefusalExtractorFindsAHostRelayedRefusal` was shown able to fail by removing the relayed-pattern matching block from `journeyPrintedRefusals` entirely: the failure reported "0 refusal(s), want 1" for the real captured fixture.

## Issues Encountered

**The full, unscoped `go test ./cmd -count=1 -timeout 90m` run took ~79 minutes**, well over the usual ~20-minute figure CLAUDE.md documents, because of sustained CPU/I/O contention from another concurrent Claude Code session actively running its own `go build ./...` / `go test ./...` / `go test -list=...` sweeps in this same checkout for the entire duration (confirmed via `ps`/`lsof`/`sample` — nested subprocesses of this run, including a real `npm install`/`aether install`/`aether update` chain inside `TestPackedNPMReleaseCandidateContract`, were genuinely making progress throughout, just slowly). This is an environmental condition, not a defect in this plan's changes; per this project's own "Concurrent sessions in one repo" precedent, the run was not restarted or abandoned, only genuinely waited out (after one initial kill-and-restart to rule out Go telemetry-counter lock contention as a separate, addressable cause, which turned out not to be the dominant factor).

**FULL-SUITE headline:** `FULL-SUITE FAIL discovered=6017 executed=6017 lanes=59` — the two counts are equal, confirming the run was not silently truncated (`cmd/testing_main_test.go`'s own `TestMain` partitions the package into 59 lanes, each independently tracking planned vs. executed, and the overall report only ever counts a genuinely completed lane). The `FAIL` status reflects 37 distinct top-level test failures across the run.

**Every failure was checked against WINDOWS rows 12 and 56 and the 2026-09-14/2026-09-20 known-red baselines before being called anything else:**

- **30 of 37** match the existing known-red union exactly (14 still-active entries from WINDOWS row 12's original 20 minus the 6 it records as now passing; the 15 tests WINDOWS row 56 newly catalogued from 208-08's own full run; and `TestResolveTestCommand_GoProject`, the separately-recorded 2026-09-14 baseline entry). None of these are addressed here — out of scope per the deviation rules' scope boundary, and already tracked.
- **4 of 37** (`TestBuildWorktreeModeRejectsOverlappingUntrackedPaths`, `TestColonyPrimeMdDeletionProducesByteIdenticalOutput`, `TestOracleCompatibilityWritesHeartbeatWhileRunning`, `TestTerritoryWrapperAuthority199`) are new to this run but **pass cleanly when re-run in isolation** (individually verified after the full run completed) — consistent with flakiness under the exceptional parallel/host-contention conditions this run experienced, not a regression this plan's changes introduced. Not filed as a new WINDOWS row since they are not reproducible in isolation; if they recur in a future full-suite run under normal load, they warrant their own row.
- **3 of 37** (`TestLifecycleFlatMirrorsMatchCanonical`, `TestPlanAndColonizeWrappersAreByteIdentical`, `TestSpecialistCommandSurfacesUnchanged`) were genuine, direct, expected consequences of Task 2's own intentional content change to colonize.md, caught by exactly the mechanisms designed to catch them (a byte-parity fence and a stated-reason content-hash fence) — both fixed within this same execution wave; see Deviations above.

None of this plan's own new tests (`TestUnattendedRefusalNamesTheWayPastOnBothLanes`, `TestAttendedRefusalTextIsUnchanged`, `TestOnlyAWorkProtectingStopCarriesTheGuidance`, `TestTheIsAnyoneHereFactHasOneReader`, `TestUnattendedFactIsSetByTheJourneyHarness`, `TestColonizeWrapperCarriesTheActWhenAloneRule`, `TestPrintedRefusalExtractorFindsAHostRelayedRefusal`, `TestPrintedRefusalExtractorIgnoresRelayedProse`, `TestPrintedRefusalsAreDeduplicatedWithinOneTranscript`) appear anywhere in the 37-name failure list.

## Fixture Provenance

`cmd/testdata/refusals/host-relay-refusal.txt` was captured by the **first** of the plan's two named routes: the one real journey walk's own transcript was still on disk at `~/.claude/projects/-private-var-folders-pj-fn0nrs6s1zj-pm7s486lnnz40000gn-T-TestJourney2625118349-001-repo/06a5379a-8663-48dd-9d4f-68865e2d8879.jsonl` (the session id and directory recorded in `208-JOURNEY-RUN.md`). Line 65 of that file is a user-role `tool_result` block whose `content` string is exactly the fixture's contents: a real `npm install` warning followed by `Fatal: Go command failed: colonize --plan-only: A territory survey already exists for this project. — next: aether colonize --force-resurvey` then `Error: exit status 1` — copied byte-for-byte, never typed by hand or retyped from the JOURNEY-RUN.md report's own paraphrase.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

All three of this plan's tasks are complete and verified, plus the two content-consequence fixes the full suite surfaced. `208-10-PLAN.md` (one more real, owner-approved journey walk) can now proceed: an unattended `-p` chain hitting the `colonize-existing-survey-found` refusal will run `aether colonize --force-resurvey` itself and carry on, rather than asking an owner who isn't there — the exact dead end 208-08's single trial hit. WINDOWS row 53 (the underlying survey-freshness finding) stays open until a live run actually exercises that path and the survey step passes; this plan only removes the specific obstacle that stopped 208-08's trial one step earlier.

---
*Phase: 208-never-a-dead-end*
*Completed: 2026-09-23*
