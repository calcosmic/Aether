---
phase: 188-one-truth-for-failures-and-advances
plan: 01
subsystem: infra
tags: [go, storage, midden, colony-state, observability]

requires: []
provides:
  - "cmd/midden_shared.go: loadMiddenFile (canonical read) and appendMiddenEntry (canonical, atomic write) — the one path every midden reader and future writer converges on"
  - "Five production consumers (autopilot pause-check, /ant-resume dashboard, pr-context's context-capsule section, memory-health summary, immune-auto-scar) repointed from the dead nested midden/midden.json to the real, written midden.json"
  - "immune-auto-scar's independent field-name bug fixed (entry[\"description\"] never existed on MiddenEntry; the real field is Message)"
  - "Bonus: aether patrol/medic's data-file health scan (scanDataFiles) now validates the real midden.json instead of a file nothing writes"
  - "cmd/midden_unification_test.go: TestOneMiddenEntryReachesAllFourConsumers, a single committed proof that one appendMiddenEntry call is visible through all four ROADMAP-named consumers"
affects: [188-05, autopilot, colony-prime, immune, memory-health, medic]

tech-stack:
  added: []
  patterns:
    - "One canonical path constant (middenCanonicalPath) plus one load/append helper pair, used by every reader and (optionally, going forward) every writer — copies the existing subcommand_reachability_ratchet_test.go idea of a single source of truth rather than five independently-hardcoded path strings"
    - "appendMiddenEntry uses store.UpdateJSONAtomically (read-mutate-write under one lock), stricter than the two existing production writers' plain Load-then-Save pair"

key-files:
  created:
    - cmd/midden_shared.go
    - cmd/midden_shared_test.go
    - cmd/midden_unification_test.go
    - .planning/phases/188-one-truth-for-failures-and-advances/deferred-items.md
  modified:
    - cmd/autopilot.go
    - cmd/context.go
    - cmd/memory_health.go
    - cmd/immune.go
    - cmd/medic_scanner.go
    - cmd/medic_scanner_test.go
    - cmd/run_autopilot_test.go
    - cmd/context_test.go

key-decisions:
  - "Canonical path is the flat midden.json (D-01), matching what the two real production writers already write — not the nested midden/midden.json every consumer had been reading"
  - "appendMiddenEntry is required for this plan's own proof test and for 188-05's retry-exhaustion record; migrating the two pre-existing writers (midden-write, spawn_budget.go) onto it is left as discretionary, unstarted here (D-02)"
  - "Corrected the plan's own interface citation: buildContextCapsuleOutput (cmd/context.go ~434-667) has no midden section at all -- the real broken-path code lived in the sibling pr-context command's RunE, not the function named in 188-01-PLAN.md's <interfaces> block. Fixed the real code; retargeted Task 3's proof test at the real command"
  - "The plan's proposed fail-then-pass method for Task 3 (flip middenCanonicalPath, expect reader/writer disagreement) does not produce a failure once appendMiddenEntry and every reader share one constant -- that is the intended outcome of unification, not a test gap. Substituted a fail-then-pass demonstration that reverts one consumer's call site at a time instead, which does fail as expected"

patterns-established:
  - "Canonical-path-constant-plus-helper-pair: one unexported path constant, one load function, one atomic append function, all readers call the load function instead of hardcoding the path"

requirements-completed: []

duration: ~45min
completed: 2026-08-19
---

# Phase 188 Plan 01: One Real Midden Summary

**One canonical `midden.json` path plus atomic `loadMiddenFile`/`appendMiddenEntry` helpers, wired into all five production readers that had silently been reading a file nothing ever wrote — proven end-to-end by one committed test that fails when any single reader regresses.**

## Performance

- **Duration:** ~45 min (includes reading 5 source files, 3 planning docs, and waiting on 3 full/background test runs — commit-to-commit span was ~16 min)
- **Started:** ~2026-08-19T21:05:00Z (research and worktree setup, before first commit)
- **Completed:** 2026-08-19T21:42:00Z
- **Tasks:** 3 (Task 1 was TDD: RED + GREEN)
- **Files modified:** 8 modified, 4 created (3 Go files + this plan's deferred-items.md)

## Accomplishments

- One canonical midden path (`cmd/midden_shared.go`'s `middenCanonicalPath = "midden.json"`) and one load/append helper pair, TDD'd (RED confirmed as a build failure, then GREEN)
- Five real production consumers repointed: `checkAutopilotPauseConditions` (autopilot), `buildResumeDashboardResult` (`/ant-resume`), `pr-context`'s context-capsule section (the real "colony-prime context capsule" code — see Deviations), `loadMemoryHealthSummary` (memory-health), `immune-auto-scar`
- `immune-auto-scar`'s independent field-name bug fixed: it read `entry["description"]`, a key that has never existed on `MiddenEntry` (the real JSON tag is `message`) — even with the path fixed, this bug alone would have kept every scar's error text empty forever
- Bonus D-03 fix: `aether patrol`/`aether medic`'s health scanner (`scanDataFiles`) now checks the real `midden.json`, not the dead nested path
- One committed proof test (`TestOneMiddenEntryReachesAllFourConsumers`) that calls the real `appendMiddenEntry` once and exercises all four ROADMAP-named consumers directly, with per-consumer failure messages naming exactly which one regressed
- Demonstrated, by actually running the tests (not by reading code), that `TestGoldenAutopilotPauseConditions`'s `critical_chaos_findings` case and `TestScanDataFilesCorrupted` were previously passing only because their fixtures and their readers agreed on the same wrong path — see the exact failure output quoted below

## Task Commits

Each task was committed atomically:

1. **Task 1a (RED): failing test for midden helpers** — `7a0a3ddd` (test)
2. **Task 1b (GREEN): canonical midden load/append helpers** — `3afe9642` (feat)
3. **Task 2: repoint five consumers, fix immune's field bug, fix two stale fixtures, plus the necessary `context_test.go` fixture fix** — `ee4bd6f6` (fix)
4. **Task 3: fail-then-pass proof that one write reaches all four consumers** — `5bbff68c` (test)

**Plan metadata:** (this commit, made after this SUMMARY)

## The demonstration the Definition of Done requires

Quoting the actual failure, captured by running the tests *before* fixing the five readers (fixtures moved to the flat path first, readers still pointed at the dead nested path):

```
=== RUN   TestScanDataFilesCorrupted
    medic_scanner_test.go:449: corrupted data file should produce critical issue
--- FAIL: TestScanDataFilesCorrupted (0.00s)
=== RUN   TestGoldenAutopilotPauseConditions
=== RUN   TestGoldenAutopilotPauseConditions/active_blockers
=== RUN   TestGoldenAutopilotPauseConditions/critical_chaos_findings
    run_autopilot_test.go:283: stopped_reason = "blocked", want "paused:critical_chaos_findings"
=== RUN   TestGoldenAutopilotPauseConditions/uncommitted_changes
--- FAIL: TestGoldenAutopilotPauseConditions (1.53s)
    --- PASS: TestGoldenAutopilotPauseConditions/active_blockers (0.27s)
    --- FAIL: TestGoldenAutopilotPauseConditions/critical_chaos_findings (1.00s)
    --- PASS: TestGoldenAutopilotPauseConditions/uncommitted_changes (0.26s)
FAIL
FAIL	github.com/calcosmic/Aether/cmd	2.337s
FAIL
```

This proves the earlier passing state was coincidental self-consistency (fixture and reader agreeing on the same wrong path), not real coverage of the production writer. After repointing the five readers at the canonical path, both pass — see the Task 2 commit message for the full command and clean rerun.

A second, independent fail-then-pass demonstration was run for Task 3's own proof test, reverting one production consumer at a time back to its old hardcoded path (see Deviations below for why the plan's originally-proposed method — flipping the shared path constant — does not produce a failure, and why that is correct).

## Files Created/Modified

- `cmd/midden_shared.go` — new: `middenCanonicalPath`, `loadMiddenFile`, `appendMiddenEntry`
- `cmd/midden_shared_test.go` — new: 5 tests covering missing-file, populated-file, fresh-append, twice-append, and nil-tags round-trip behavior
- `cmd/midden_unification_test.go` — new: `TestOneMiddenEntryReachesAllFourConsumers`
- `cmd/autopilot.go` — `checkAutopilotPauseConditions` now calls `loadMiddenFile(store)`
- `cmd/context.go` — `buildResumeDashboardResult` and `pr-context`'s RunE (its "9. midden" section) now call `loadMiddenFile(store)`
- `cmd/memory_health.go` — `loadMemoryHealthSummary` now calls `loadMiddenFile(s)`
- `cmd/immune.go` — `immuneAutoScarCmd` now calls `loadMiddenFile(store)` and reads `entry.Category`/`entry.Message` as real struct fields instead of `map[string]interface{}` key lookups
- `cmd/medic_scanner.go` — `scanDataFiles`'s `structuredFiles` list now checks `midden.json`
- `cmd/medic_scanner_test.go` — `TestScanDataFilesCorrupted`'s and the healthy-scan integration fixture's seed paths moved to the flat path
- `cmd/run_autopilot_test.go` — `TestGoldenAutopilotPauseConditions`'s `critical_chaos_findings` case seed moved to the flat path, field name corrected from `description` to `message`
- `cmd/context_test.go` — three fixtures (`TestResumeDashboardWithMemory`, `TestPRContext`, `TestPRContextWithMidden`) moved off the dead nested path — not in the plan's `files_modified`, required by this task's own change (see Deviations)
- `.planning/phases/188-one-truth-for-failures-and-advances/deferred-items.md` — new: two out-of-scope discoveries logged, not fixed

## Decisions Made

- Canonical path is the flat `midden.json`, matching both existing production writers (`midden-write`, `spawn_budget.go`) — this was evidence, not a judgment call (see 188-CONTEXT.md D-01)
- `appendMiddenEntry` uses `store.UpdateJSONAtomically`, stricter than either existing writer's plain Load-then-Save pair, per D-02
- Did not migrate the two pre-existing writers onto `appendMiddenEntry` — explicitly discretionary per D-02, left for a future pass or 188-05 to decide
- Did not add a synthetic midden section to `buildContextCapsuleOutput` itself (it structurally has none) — the real, live "context capsule" delivery path for midden data is `pr-context`, already fixed, and this repo's own pre-existing test comment (`TestColonyPrimeAAC005Audit`) already documents this as the accepted architecture ("goes through the context capsule path, not colony-prime")

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Three `context_test.go` fixtures would have regressed from this task's own reader fix**
- **Found during:** Task 2, verification pass
- **Issue:** `TestResumeDashboardWithMemory`, `TestPRContext`, and `TestPRContextWithMidden` each seed midden data at the dead nested path and assert on counts (`recent_failures`, `midden.count`) returned by `buildResumeDashboardResult`/`pr-context` — exactly the two functions this task repoints. Left unfixed, all three would fail after Task 2's reader change.
- **Fix:** Moved each seed's `SaveJSON` call to the flat `midden.json` path; removed the now-unnecessary `os.MkdirAll(.../midden)` calls, mirroring the same treatment the plan already specifies for `run_autopilot_test.go`.
- **Files modified:** `cmd/context_test.go`
- **Verification:** Full `go test ./cmd/...` run (350s) after the fix: all packages pass, no regressions.
- **Committed in:** `ee4bd6f6` (Task 2 commit)

**2. [Rule 1 - Bug, plan-research inaccuracy] Plan's own `<interfaces>` block misattributed the context-capsule midden read to the wrong function**
- **Found during:** Task 2, while reading `cmd/context.go` before editing
- **Issue:** 188-01-PLAN.md's `<interfaces>` section and `must_haves.key_links` cite `cmd/context.go:435 func buildContextCapsuleOutput(...)` as the site of the broken nested-path read at line 859. Structural inspection (`grep -n "^func \|^type "`) shows `buildContextCapsuleOutput` actually spans lines 434-667 only, ending well before line 859; it has no `Midden` field on its `ContextCapsuleOutput` return type and none of its five assembled sections (state, signals, decisions, risks, recent_narrative) ever reads midden data in any form. The actual broken-path code — the real "9. midden" numbered section — lives in the sibling `pr-context` command (`prContextCmd`'s RunE, starting at line 692), a separate function entirely. This repo's own pre-existing test comment in `cmd/colony_prime_audit_test.go`'s `TestColonyPrimeAAC005Audit` already documents this exact fact ("midden... goes through the context capsule path, not colony-prime"), so this is a known, pre-existing architecture — the plan's citation was simply imprecise about which function name to write down.
- **Fix:** Fixed the real code (in `prContextCmd`'s RunE) at the correct line, exactly as originally planned in every respect except the enclosing function's name. Retargeted Task 3's proof test to invoke the real `pr-context` command instead of calling the misattributed `buildContextCapsuleOutput` (which cannot satisfy the plan's own described assertion — it has no midden output to assert on at all; following the plan literally here would not have compiled).
- **Files modified:** `cmd/context.go` (Task 2, already correct); `cmd/midden_unification_test.go` (Task 3, retargeted)
- **Verification:** `TestOneMiddenEntryReachesAllFourConsumers` passes against the real `pr-context` command. Note for whoever runs an automated check against this plan's `must_haves.key_links` pattern (`loadMiddenFile\(store\)` appearing somewhere in `cmd/context.go`): that plain-text pattern still matches, twice — once in `buildResumeDashboardResult` (correctly attributed) and once in `prContextCmd`'s RunE (the code the plan's `via` text meant but mis-named) — so a source-grep-based check would pass either way; this note exists so a human reviewer knows why, rather than trusting the grep alone.
- **Committed in:** `ee4bd6f6` (Task 2), `5bbff68c` (Task 3)

**3. [Rule 1 - Bug] Task 1's nil-tags test asserted an impossible post-round-trip state**
- **Found during:** Task 1, GREEN step
- **Issue:** The first version of `TestAppendMiddenEntryNilTagsBecomesEmptyNonNilSlice` asserted that `Tags` reads back as a non-nil empty slice after a round trip through disk. `colony.MiddenEntry.Tags` is JSON-tagged `omitempty` (a field this plan is explicitly told not to modify), so an empty slice is omitted from the written JSON and legitimately unmarshals back as `nil` — this is true of the real, pre-existing `midden-write` writer too, not a defect `appendMiddenEntry` introduces.
- **Fix:** Rewrote the test to assert the actual observable contract: length zero after reload, and no `"tags"` key written to the on-disk JSON for an empty slice (renamed to `TestAppendMiddenEntryNilTagsRoundTripsEmpty`).
- **Files modified:** `cmd/midden_shared_test.go`
- **Verification:** `go test ./cmd/ -run 'TestLoadMiddenFile|TestAppendMiddenEntry' -v -count=1`: 5/5 pass
- **Committed in:** `3afe9642` (Task 1 GREEN commit)

**4. [Rule 1 - Bug] Task 3's plan-specified fail-then-pass method doesn't apply to a correctly-unified implementation**
- **Found during:** Task 3, acceptance-criteria verification
- **Issue:** The plan's acceptance criteria say to temporarily change `middenCanonicalPath` back to the nested path and confirm the proof test fails. Since `appendMiddenEntry` and every reader now dereference the identical constant (the entire point of D-02's unification), flipping it moves the write path and every read path together — they stay in agreement, and the test keeps passing. This is the correct, intended behavior of a genuinely unified implementation, not a test defect.
- **Fix:** Substituted a fail-then-pass demonstration that actually exercises the regression class this phase closes: reverted `checkAutopilotPauseConditions` (then, separately, `loadMemoryHealthSummary`) back to its own old hardcoded `store.LoadJSON("midden/midden.json", ...)` call, one at a time, confirmed each produces a named test failure, then reverted back (`git diff` against the parent commit is empty except for the new test file — confirmed).
- **Files modified:** none permanently (temporary, reverted); `cmd/midden_unification_test.go` documents both findings in its own doc comment
- **Verification:** two isolated failing runs quoted in the Task 3 commit message; final `git diff` clean; `go build`/`go vet`/full `go test ./cmd/...` all clean afterward
- **Committed in:** `5bbff68c` (Task 3 commit message documents both fail-then-pass runs)

---

**Total deviations:** 4 auto-fixed (3 bugs the plan's own text would have caused if followed literally, 1 additional regression this task's own reader change would have caused in an unlisted file)
**Impact on plan:** All four were necessary for correctness or for the plan's own stated acceptance criteria to be achievable at all. No scope creep — the underlying goal (canonical midden path, five consumers repointed, one proof test) is delivered exactly as scoped; only citations and one already-broken test assumption were corrected.

## Issues Encountered

None beyond the four deviations above, which were self-contained and resolved within the task they were found in.

## Next Phase Readiness

- `appendMiddenEntry` is ready for 188-05's autopilot retry-exhaustion record (D-13) — it is the exact function 188-05's plan is specified to call
- Zero production references to the nested `midden/midden.json` path remain in `cmd/*.go` (verified by repo-wide grep)
- Full-repo `go build ./...`, `go vet ./...`, and `go test ./... -count=1` all clean — every package passes, including `cmd` (382.5s) and every `pkg/*` package (agent, agent/curation, cache, codegraph, codex, colony, downloader, events, exchange, graph, learn, llm, memory, smoke, storage, terminal, trace)
- No blockers for 188-02/03 (different files, ran in parallel) or 188-05 (depends on this plan's `appendMiddenEntry`, which now exists and is tested)

## Known Stubs

None.

## Self-Check

Files:
- FOUND: `cmd/midden_shared.go`
- FOUND: `cmd/midden_shared_test.go`
- FOUND: `cmd/midden_unification_test.go`
- FOUND: `.planning/phases/188-one-truth-for-failures-and-advances/deferred-items.md`
- FOUND: `.planning/phases/188-one-truth-for-failures-and-advances/188-01-SUMMARY.md`

Commits (`git log --oneline --all | grep <hash>`):
- FOUND: `7a0a3ddd` (test: RED)
- FOUND: `3afe9642` (feat: GREEN)
- FOUND: `ee4bd6f6` (fix: repoint five consumers)
- FOUND: `5bbff68c` (test: unification proof)

Repo-wide verification (required by this executor's `<parallel_execution_note>`):
- `go build ./...`: clean, zero output
- `go vet ./...`: clean, zero output
- `go test ./... -count=1`: **ok**, every package passes —
  `cmd` (382.490s), `pkg/agent` (9.428s), `pkg/agent/curation` (1.769s),
  `pkg/cache` (1.455s), `pkg/codegraph` (1.138s), `pkg/codex` (25.222s),
  `pkg/colony` (0.376s), `pkg/downloader` (2.974s), `pkg/events` (2.519s),
  `pkg/exchange` (2.650s), `pkg/graph` (3.145s), `pkg/learn` (4.328s),
  `pkg/llm` (3.085s), `pkg/memory` (3.392s), `pkg/smoke` (2.464s),
  `pkg/storage` (3.021s), `pkg/terminal` (2.790s), `pkg/trace` (2.376s).
  Exit code 0.

## Self-Check: PASSED

---
*Phase: 188-one-truth-for-failures-and-advances*
*Completed: 2026-08-19*
