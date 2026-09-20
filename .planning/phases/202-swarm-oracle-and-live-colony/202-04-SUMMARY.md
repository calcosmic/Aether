---
phase: 202-swarm-oracle-and-live-colony
plan: "04"
subsystem: live-events
tags: [event-bus, dead-code, ast-guard, live-colony]

# Dependency graph
requires:
  - phase: 202-02
    provides: "pkg/events/colony_live.go's one versioned live-colony event model and cmd/live_events.go's emitColonyLive single emission boundary -- the surviving transport this plan proves is the only one left standing."
provides:
  - "The retired NDJSON event stream (cmd/event_types.go, event_stream.go, event_bridge.go, event_reader.go, event_writer.go, events_test.go) deleted after a repository-wide, recorded caller-freedom search -- LIVE-01's 'one versioned model' is now structurally true, not a description of intent."
  - "cmd/live_model_singleton_test.go: TestOneLiveEventModelOnly, an AST guard that finds a candidate second event transport structurally (a type whose fields carry a timestamp and an event-kind shape) and fails by name and declaration position if any function both produces one and passes it to a raw file-write call."
affects: [202-06, 202-09, 202-10, 202-11, 202-15]

# Actuals (#2632)
actuals:
  tokens: 9822
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Structural second-transport detection: a guard identifies a candidate event transport by field shape (timestamp + event-kind, matched by Go field name or JSON tag, case-insensitively) rather than by a maintained type-name or file-name list, so a differently-named reintroduction is still caught. The permitted set of known-legitimate exceptions (the streaming server's wsEvent, the emission boundary, the CLI subcommand surface) is expressed as symbols (type/function names), never file paths, so renaming a file can never silently widen or narrow the guard."

key-files:
  created:
    - cmd/live_model_singleton_test.go
  modified: []

key-decisions:
  - "Caller-freedom was proven, not assumed: a repository-wide search (every exported and unexported symbol from the five files, across Go, TypeScript under .aether/ts-host, and wrapper markdown/YAML) found references only within the five files and their own test file, plus historical citations in planning docs and the Classic contract corpus's source_citations (evidentiary, not code dependencies -- classic_contract_test.go only requires those strings be non-blank, never that the path exists on disk)."
  - "The orphan allowlist (cmd/testdata/orphan_allowlist.json) is keyed by CLI subcommand name ('aether <subcommand>'), not by file or symbol; none of its 257 entries referenced the deleted files' internal helper functions, so no entry needed removal -- the acceptance criterion 'any entry it held for those files is removed rather than retained' is satisfied vacuously because none ever existed."
  - "The second-transport guard uses same-function co-occurrence (a function that both produces a candidate-shaped type -- as a parameter, receiver, or composite literal -- and calls a raw file-write primitive in its own body), not unrestricted call-graph reachability across the whole package: the cmd package has 70+ unrelated os.WriteFile call sites, and full transitive reachability would produce false positives from unrelated call chains (e.g. a candidate-shaped type's constructor happening to call a logger that eventually writes some other file). Same-function scoping still catches the retired code's own shape (WriteEvent(streamPath string, eventLine EventLine) called os.OpenFile directly in its own body) while staying provably free of false positives against the real package."
  - "wsEvent (cmd/serve.go) matches the guard's structural fingerprint (Type + Timestamp fields) exactly like the retired EventLine did, but is explicitly proven -- not merely assumed -- to go unreported: a dedicated subtest confirms the detector actually finds it as a structural candidate (so the 'zero violations' result is not vacuous), and it is additionally named on the symbol-keyed permitted-types allowlist as documented defense-in-depth, since it is never passed to a real file write anywhere in the package (it goes to a WebSocket connection)."

requirements-completed: [LIVE-01]

coverage:
  - id: D1
    description: "The retired NDJSON event stream (event_types.go, event_stream.go, event_bridge.go, event_reader.go, event_writer.go, events_test.go) is deleted, with caller-freedom proven by a repository-wide search recorded in this summary rather than assumed; eventbus.go and serve.go (the surviving bus's real callers) are untouched; the whole module builds and vets clean; the surviving event-bus tests pass unchanged."
    requirement: "LIVE-01"
    verification:
      - kind: unit
        ref: "go build ./... && go vet ./... && go test ./pkg/events -count=1 && go test ./cmd -run '^TestEventBus' -count=1 (Task 1 <verify>)"
        status: pass
    human_judgment: false
  - id: D2
    description: "A guard (TestOneLiveEventModelOnly) refuses a second event transport by name: it identifies a candidate structurally (timestamp + event-kind field shape), reports it by file and declaration position via a negative fixture, passes against the real package, and its permitted set is expressed as symbols rather than file paths."
    requirement: "LIVE-01"
    verification:
      - kind: unit
        ref: "cmd/live_model_singleton_test.go#TestOneLiveEventModelOnly (all 4 subtests: real-package-zero-violations, wsEvent-structurally-matches-but-unreported, fixture-reported-by-name-and-position, permitted-type-suppresses-violation)"
        status: pass
    human_judgment: false

duration: 25min
completed: 2026-09-11
status: complete
---

# Phase 202 Plan 04: Delete the Retired Event Stream, Refuse Its Return Summary

**Deleted the zero-caller NDJSON event stream (five source files, one test file, 882 lines) after a recorded repository-wide caller-freedom search, and added an AST guard that fails by name and declaration position the moment a structurally-shaped second event transport reaches a real file write.**

## Performance

- **Duration:** 25 min
- **Started:** 2026-09-11T09:16:00Z (approx.)
- **Completed:** 2026-09-11T09:41:15Z
- **Tasks:** 2
- **Files modified:** 7 (1 created, 6 deleted)

## Accomplishments

- Ran a repository-wide symbol search (every exported and unexported name declared in the five retired files) across Go, TypeScript under `.aether/ts-host`, and wrapper markdown/YAML sources -- confirmed the only references were within the five files themselves and their own test file, plus historical citations in planning documents and the Classic contract corpus's evidence trail (non-code, non-blocking).
- Deleted `cmd/event_types.go`, `cmd/event_stream.go`, `cmd/event_bridge.go`, `cmd/event_reader.go`, `cmd/event_writer.go`, and `cmd/events_test.go` (882 lines total) -- the complete, tested, never-wired second event transport this repository's own CLAUDE.md names as its signature failure mode.
- Left `cmd/eventbus.go` (the surviving bus's CLI subcommand surface) and `cmd/serve.go` (the streaming server) byte-for-byte untouched, confirming neither referenced the retired files.
- Confirmed `cmd/testdata/orphan_allowlist.json` (257 CLI-subcommand-keyed entries) held no entry for the deleted files or their symbols, so none required removal; `cmd/testdata/orphan_allowlist_baseline.json` remained byte-identical.
- Added `cmd/live_model_singleton_test.go`'s `TestOneLiveEventModelOnly`: an AST guard, following the existing `TestEveryLiveEventGoesThroughOneBoundary` precedent, that structurally detects a candidate second event transport (a struct whose fields carry a timestamp and an event-kind shape, matched by field name or JSON tag) and fails, naming the type and its declaration position, if any function both produces that type and calls a raw file-write primitive (`os.WriteFile`/`OpenFile`/`Create`, `ioutil.WriteFile`) in its own body.
- Proved the guard is not vacuous: a dedicated subtest confirms `wsEvent` (`cmd/serve.go`) structurally matches the candidate fingerprint (it carries the same `Type`/`Timestamp` field pair the retired `EventLine` did) yet is genuinely unreported, because it is never passed to a real file write anywhere in the package (it only ever reaches a WebSocket connection).
- Proved the guard can fail: a synthetic negative fixture (a `sneakyEventLine` type plus a `sneakyEventWriter` function calling `os.OpenFile` directly) is reported with the exact type name, function name, and a position naming the fixture file.
- Proved the permitted set is real, not decorative: a second fixture confirms explicitly allowlisting a type name by symbol suppresses its own violation.

## Task Commits

Each task was committed atomically:

1. **Task 1: Prove the retired stream caller-free, then delete it** - `b3942a2d` (fix)
2. **Task 2: Refuse a second event transport by name** - `3ccbf24a` (test)

**Plan metadata:** (this commit)

_Note: Task 2 carried `tdd="true"`, but the detector and its test are one self-contained pair (the guard function lives in the same file as `TestOneLiveEventModelOnly` and has no separate production implementation elsewhere to fail against first) -- the same no-RED-then-GREEN pattern 202-01's and 202-02's Tasks 2 already documented for static-fixture / structural-regression tests._

## Files Created/Modified

- `cmd/live_model_singleton_test.go` (created) - `TestOneLiveEventModelOnly` and its supporting structural-detection helpers (`structHasTimestampAndKindShape`, `findLiveModelSingletonCandidates`, `functionMentionsCandidate`, `functionPerformsRawFileWrite`, `findLiveModelSingletonViolations`), plus the symbol-keyed permitted-type/permitted-function allowlists.
- `cmd/event_types.go` (deleted) - The retired `EventType`/`EventLine`/`EventPayload` domain types.
- `cmd/event_stream.go` (deleted) - The retired NDJSON stream init/emit helpers (`InitEventStream`, `EmitPhaseStart`, etc.).
- `cmd/event_bridge.go` (deleted) - The retired Go/TypeScript bridge stub (`SyncEventsWithTS`, `ReadTSEvents`, `DemonstrateBridgeCompatibility`).
- `cmd/event_reader.go` (deleted) - The retired NDJSON reader/tailer (`ReadEvents`, `TailEvents`).
- `cmd/event_writer.go` (deleted) - The retired NDJSON writer (`WriteEvent`, `EventStream`, `CreateEventStream`).
- `cmd/events_test.go` (deleted) - The retired stream's own test suite, exercising only the deleted code.

## Decisions Made

See `key-decisions` in frontmatter.

## Deviations from Plan

None - plan executed exactly as written. The repository-wide search (Task 1) and the guard's field-shape/same-function design (Task 2) were both already specified by the plan's action text; no bugs, missing critical functionality, or blockers were found during execution.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- LIVE-01's "one versioned typed live-event model" is now structurally enforced, not merely described: a future attempt to reintroduce a second event transport under any file name fails `TestOneLiveEventModelOnly` by name and position before it can accumulate callers.
- `cmd/eventbus.go` and `cmd/serve.go` (the surviving bus's CLI surface and streaming server) are unchanged and remain the only paths reading/writing durable event data in package `cmd`.
- Later Phase 202 plans (202-06, 202-09, 202-10, 202-11, 202-15) that extend live-event rendering or the cockpit have one less standing distraction in the tree, and their own new event-shaped types (if any) will be caught by this guard if they ever reach a raw file write outside the surviving bus.
- No blockers.

## Self-Check: PASSED

- `cmd/live_model_singleton_test.go` — FOUND
- `cmd/event_types.go` — CONFIRMED ABSENT
- `cmd/event_stream.go` — CONFIRMED ABSENT
- `cmd/event_bridge.go` — CONFIRMED ABSENT
- `cmd/event_reader.go` — CONFIRMED ABSENT
- `cmd/event_writer.go` — CONFIRMED ABSENT
- `cmd/events_test.go` — CONFIRMED ABSENT
- `cmd/eventbus.go`, `cmd/serve.go` — unmodified (confirmed via git diff against parent commits)
- Commit `b3942a2d` — FOUND in `git log --oneline --all`
- Commit `3ccbf24a` — FOUND in `git log --oneline --all`
- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test ./pkg/events -count=1` — PASS
- `go test ./cmd -run '^TestEventBus' -count=1` — PASS
- `go test ./cmd -run '^TestOneLiveEventModelOnly$' -v -count=1` — PASS (all 4 subtests)
- `cmd/testdata/orphan_allowlist_baseline.json` — byte-identical (no diff)

---
*Phase: 202-swarm-oracle-and-live-colony*
*Completed: 2026-09-11*
