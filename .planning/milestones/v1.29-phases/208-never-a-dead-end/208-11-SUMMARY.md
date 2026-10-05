---
phase: 208-never-a-dead-end
plan: 11
subsystem: refusal-lifecycle
tags: [go, refusal-contract, colonize, unattended-session, cli]

# Dependency graph
requires:
  - phase: 208-never-a-dead-end (plans 01-10)
    provides: the typed refusal contract (cmd/refusal.go), sessionHasNoOneToAsk
      and the shared guidance sentence (cmd/unattended_session.go, plan 09),
      and the second gap-closure round's owner ruling D-03 that a chat
      reading correct guidance still is not a mechanism (208-CONTEXT.md)
provides:
  - "attemptRefusalSelfRecovery (cmd/refusal_self_recovery.go): the one
    decision Aether uses to carry out a refusal's own recovery itself when
    sessionHasNoOneToAsk() is true, the refusal is opted in, protects work,
    and names a next command"
  - "refusalSelfRecoveryTable: the checked-in opt-in list, holding exactly
    one entry this round (colonize-existing-survey-found)"
  - "Both colonize existing-survey call sites (runCodexColonizePlanOnly and
    runCodexColonizeWithOptions, cmd/codex_colonize.go) route through the
    one decision instead of unconditionally returning the refusal"
  - "refusalLogEntry's additive Recovered field (cmd/refusal_log.go),
    written by appendRecoveredRefusalToLog, read correctly by pre-existing
    log records with no schema-version bump"
  - "Three guards, each proved by breaking the thing it guards:
    TestAttendedColonizeStillStopsAndAsks (attended behaviour unchanged),
    TestSelfRecoveryHasOneDecision (one decision, structurally enforced),
    TestOnlyASafeRefusalCanRecoverItself (opt-in contract)"
affects: []

# Actuals (#2632)
actuals:
  tokens: 7191
  tasks: 3
  commits: 2

tech-stack:
  added: []
  patterns:
    - "A refusal offers itself to one shared self-recovery decision before
      returning; the decision announces (visual-mode only) and records
      (refusal log) before ever returning true, so a caller that receives
      true has nothing left to announce or record itself"
    - "An additive, omitempty log field read by every pre-existing record as
      its honest default, with no schema-version bump"

key-files:
  created:
    - cmd/refusal_self_recovery.go
    - cmd/refusal_self_recovery_test.go
  modified:
    - cmd/codex_colonize.go
    - cmd/refusal_log.go

key-decisions:
  - "The finalize sibling (colonize-finalize-existing-survey-found)
    deliberately stays out of refusalSelfRecoveryTable: its own next command
    is work a chat performs (dispatching workers, posting a completion
    packet), not work this runtime process can perform for itself at the
    moment that refusal fires -- documented in the map's own doc comment so
    a later reader does not add it by symmetry (per the plan's own explicit
    instruction)."
  - "The four self-recovery conditions are deliberately not joined by any
    freshness/staleness check of their own -- the refusal itself consults no
    such classification, so a second condition here would make the same
    refusal behave two different ways for reasons the refusal never states."
  - "The notice is drawn through emitVisualProgress (silent in
    machine-output mode), never warnAndCarryOn's unconditional stdout write
    -- the link the plan named explicitly as load-bearing for keeping a
    host's JSON parse of the plan-only manifest intact."
  - "Task 1 and Task 2 landed as two separate commits despite touching the
    same file (cmd/codex_colonize.go) in two different functions -- the
    direct-lane edit was temporarily reverted, staged/committed for Task 1
    without it, then re-applied and committed alone for Task 2, so each
    commit's diff matches its task's own declared scope exactly."

patterns-established:
  - "A per-refusal opt-in table plus one decision function is the shape a
    future recoverable refusal follows -- add an entry to the map, never a
    second call site re-deriving the four conditions."

requirements-completed: [UED-10]

coverage:
  - id: D1
    description: "An unattended colonize --plan-only meeting an existing survey carries out the forced re-survey itself instead of stopping, reports it in machine-readable output with nothing leaked into it, names the exact command in visual mode, and records a recovered refusal-log entry; an old-shaped log record still reads as not recovered."
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/refusal_self_recovery_test.go#TestNoOneHereMeansAetherRefreshesTheMapItself"
        status: pass
      - kind: unit
        ref: "cmd/refusal_self_recovery_test.go#TestOldShapedRefusalLogRecordStillReadsAsNotRecovered"
        status: pass
    human_judgment: false
  - id: D2
    description: "The direct (non-plan-only) colonize lane routes through the same one decision as the plan-only lane; an attended session on either lane still stops and names its next command byte-identically to before; the decision is structurally confined to one file; the opt-in contract (registered, stop, protects work, real next command) is enforced and provably fails on a downgraded row."
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/refusal_self_recovery_test.go#TestUnattendedDirectColonizeRefreshesTheMapItself"
        status: pass
      - kind: unit
        ref: "cmd/refusal_self_recovery_test.go#TestAttendedColonizeStillStopsAndAsks"
        status: pass
      - kind: unit
        ref: "cmd/refusal_self_recovery_test.go#TestSelfRecoveryHasOneDecision"
        status: pass
      - kind: unit
        ref: "cmd/refusal_self_recovery_test.go#TestOnlyASafeRefusalCanRecoverItself"
        status: pass
    human_judgment: false
  - id: D3
    description: "The full, unscoped cmd package suite ran to completion (discovered==executed) with every failure matching this project's recorded known-red list; no new failure was introduced."
    verification:
      - kind: other
        ref: "go test ./cmd -count=1 -timeout 90m (FULL-SUITE FAIL discovered=6024 executed=6024 lanes=59; 30 top-level failures, all 30 names matching .planning/WINDOWS.md row 56's catalogued list exactly)"
        status: pass
    human_judgment: false

duration: ~52min (dominated by the required full, unscoped ./cmd suite run: 29m25s)
completed: 2026-09-23
status: complete
---

# Phase 208 Plan 11: Aether Refreshes the Map Itself When Nobody Is There Summary

**`attemptRefusalSelfRecovery` (cmd/refusal_self_recovery.go) makes an unattended `aether colonize` (both the plan-only and direct lanes) carry out the existing-survey refusal's own forced-resurvey recovery itself — rather than printing an instruction and hoping a chat follows it — while an attended session stays byte-identical.**

## Performance

- **Duration:** ~52 min (three tasks' own code/test work took well under 10 minutes combined; the remainder was the plan-mandated full, unscoped `go test ./cmd -count=1 -timeout 90m` run, which took 29m25s)
- **Tasks:** 3 planned, all completed (Task 3 ran checks only — no files changed, no commit)
- **Commits:** 2 (Task 1, Task 2)
- **Files modified:** 4 (2 created, 2 modified)

## Accomplishments

- `cmd/refusal_self_recovery.go` adds `refusalSelfRecoveryTable` (one opt-in entry: `colonize-existing-survey-found`, with its own recorded reason and an explicit note that the finalize sibling deliberately stays out) and `attemptRefusalSelfRecovery` — the ONE place this repository decides whether to carry out a refusal's own recovery, gated on all four of: `sessionHasNoOneToAsk()`, the refusal's id being opted in, `ProtectsWork`, and a non-empty `NextCommand`.
- Both colonize existing-survey call sites — `runCodexColonizePlanOnly` and `runCodexColonizeWithOptions` (`cmd/codex_colonize.go`) — now offer the refusal to that one decision before returning it, and fall through with `ForceResurvey` set when the decision says yes. Nothing downstream needed to change: the manifest, its receipt, and the result map already read that option.
- `cmd/refusal_log.go` gets an additive `Recovered` field (`omitempty`, no schema-version bump) and a shared `appendRefusalLogEntry(r, recovered)` writer; `appendRefusalToLog` and the new `appendRecoveredRefusalToLog` both call it.
- `renderRefusalSelfRecoveryNotice` draws the person-facing notice through `renderBanner`/`voiceLine` (the shared glyph table), emitted via `emitVisualProgress` — silent in machine-output mode, which is what keeps a host's JSON parse of the plan-only manifest intact.
- `cmd/refusal_self_recovery_test.go` proves the slice end to end through the real `colonize`/`colonize --plan-only` commands (not internal functions in isolation), and adds three guards, each shown able to fail by breaking the thing it guards.

## Task Commits

1. **Task 1: End to end — nobody is here, so Aether refreshes the map itself** - `dde5a85b` (feat) — `refusal_self_recovery.go`, the `refusal_log.go` additive field, the plan-only call site, and `TestNoOneHereMeansAetherRefreshesTheMapItself` / `TestOldShapedRefusalLogRecordStillReadsAsNotRecovered`.
2. **Task 2: The second colonize lane, and the guards that keep this honest** - `3a80f55f` (feat) — the direct-lane call site plus `TestAttendedColonizeStillStopsAndAsks`, `TestUnattendedDirectColonizeRefreshesTheMapItself`, `TestSelfRecoveryHasOneDecision`, `TestOnlyASafeRefusalCanRecoverItself`.
3. **Task 3: Prove nothing else moved, and leave the tree ready for the one authorised walk** — no files changed; ran the checks below and recorded the results here. No commit (nothing to stage).

_Task 1 and Task 2 both touch `cmd/codex_colonize.go` (two different functions, two different existing-survey sites) — they were split cleanly rather than landed together: the direct-lane edit was temporarily reverted before staging Task 1, then re-applied and staged alone for Task 2, so each commit's diff matches only its own task's declared scope._

## Files Created/Modified

- `cmd/refusal_self_recovery.go` (new) — the opt-in table, the one decision function, the notice renderer
- `cmd/refusal_self_recovery_test.go` (new) — the end-to-end proof plus every guard
- `cmd/codex_colonize.go` — both existing-survey call sites route through the one decision
- `cmd/refusal_log.go` — additive `Recovered` field, shared writer, `appendRecoveredRefusalToLog`

## Decisions Made

See `key-decisions` in frontmatter. In brief: the finalize sibling refusal is deliberately excluded from the opt-in table (its recovery is chat work, not runtime work, at the moment it fires); the four conditions carry no freshness check of their own because the refusal itself has none; the notice goes through `emitVisualProgress` specifically (not `warnAndCarryOn`'s unconditional write) to keep JSON-mode output clean; and Task 1/Task 2 landed as two separate commits by temporarily reverting/re-applying the second call site rather than accepting one commit spanning both tasks' scope.

## Deviations from Plan

None — plan executed exactly as written. Both call sites, the log field, and every named guard match the plan's own description; no Rule 1-4 auto-fix was needed.

### Mutation Proof (per CLAUDE.md's Definition of Done — a test must be able to fail)

| Reverted change | Test that failed | Failure text (abbreviated) |
|---|---|---|
| Plan-only call site reverted to unconditional `return nil, refuse(...)` | `TestNoOneHereMeansAetherRefreshesTheMapItself` (both subtests) | "machine-readable output did not parse as JSON... output: " (empty — the refusal error went to stderr, not stdout) and "visual notice does not name the refusal's own next command" |
| Direct-lane call site reverted to unconditional `return nil, refuse(...)` | `TestUnattendedDirectColonizeRefreshesTheMapItself` | "machine-readable output did not parse as JSON: unexpected end of JSON input\noutput: " |
| A second file (`cmd/zz_temp_second_self_recovery_reader.go`) planted, calling both `sessionHasNoOneToAsk()` and `refuse(...)` | `TestSelfRecoveryHasOneDecision` | "only cmd/refusal_self_recovery.go may hold a self-recovery decision; found a second one also in: [cmd/zz_temp_second_self_recovery_reader.go (both calls sessionHasNoOneToAsk and refuse(...) in the same file)]" |
| A second file naming `refusalSelfRecoveryTable` alone (no call pair) | `TestSelfRecoveryHasOneDecision` | "...found a second one also in: [cmd/zz_temp_names_the_map.go (names refusalSelfRecoveryTable)]" |
| `TestOnlyASafeRefusalCanRecoverItself`'s own internal negative case (a locally built table naming a non-work-protecting row, never the real registry) | (embedded in the test itself) | reports `colonize-finalize-timestamp-in-future: does not protect work` by name |

All reverted changes were restored and re-verified green before proceeding to the next step.

## Issues Encountered

**The visual-mode assertion initially failed for an environment reason, not a code defect.** `writeVisualOutput` translates an `aether <verb>` mention through `translateHintCommandsForPlatform`/`detectPlatform()`, and this test machine's `detectPlatform()` resolved to `"codex"` (rendering `$ant-colonize --force-resurvey`) rather than the raw CLI form the test first asserted against literally. Fixed by pinning `AETHER_PLATFORM=claude` in that subtest and deriving the expected string via the same `translateHintCommandsForPlatform` call the runtime uses — still read from the refusal's own registered `NextCommand`, never a hand-typed literal, per the plan's own acceptance criterion.

## Verification Record

- `go build ./...` — clean.
- `go vet ./cmd ./pkg/...` — clean.
- `go test ./cmd -run 'Refusal|Colonize|Unattended|TestJourneyGateVerdict|TestEvalGate|TestLifecycleFlatMirrorsMatchCanonical|TestPlanAndColonizeWrappersAreByteIdentical' -count=1 -timeout 20m` — 2 failures: `TestCodexNativeCancellationRefusalReplay`, `TestCodexNativeCancellationRefusalEvidence` (matched by the `Refusal` keyword incidentally; both are unrelated codex-native-cancellation tests, both already catalogued in `.planning/WINDOWS.md` row 56 from 208-08's own full-suite run — not a regression from this plan; this plan touches none of their files).
- `go test ./cmd -run 'TestNoOneHere...|TestAttended...|TestSelfRecovery...|TestOnlyASafe...|TestUnattendedDirect...|TestOldShaped...|TestAttendedRefusalTextIsUnchanged|TestOnlyAWorkProtectingStopCarriesTheGuidance|TestUnattendedRefusalNamesTheWayPastOnBothLanes|TestTheIsAnyoneHereFactHasOneReader|TestBehaviourMatchesTheRefusalTable|TestEveryRefusalRowNamesANextCommand|TestRefusalRegisterIsSortedAndUnique' -count=1 -timeout 12m` — all pass; every 208-09 test that must stay untouched passed with no edit.
- `go test ./cmd -run 'TestRefusalLog|TestColonizeWrapperCarriesTheActWhenAloneRule|TestLifecycleFlatMirrorsMatchCanonical|TestPlanAndColonizeWrappersAreByteIdentical|TestSpecialistCommandSurfacesUnchanged'` — all pass (confirms `cmd/refusal_log.go`'s change didn't disturb its own existing test family, and no wrapper/mirror file drifted).
- **Full, unscoped suite:** `go test ./cmd -count=1 -timeout 90m` run in the background (started, polled to completion, never run in the foreground per this repo's own documented auto-background risk). `FULL-SUITE FAIL discovered=6024 executed=6024 lanes=59` — the two counts are equal, confirming the run completed rather than being silently truncated. 30 top-level failures; every one of the 30 names matches `.planning/WINDOWS.md` row 56's catalogued 30-name list from 208-08's own full-suite run **exactly** (name-for-name, zero additions, zero omissions) — the strongest possible evidence that this plan introduced no regression anywhere in the `cmd` package. None of this plan's own new tests appear in the failure list.
- **Scope-safety confirmed:** `git status --short` and `git diff HEAD~2 --stat -- 'cmd/journey*.go' cmd/refusal_register.go .aether/commands .claude/commands .opencode/commands` both show nothing — no wrapper file, no journey file, and no refusal-register row was touched by either task commit.

**Plain-English readiness statement:** the tree is ready for the next authorised walk — every check this plan owns passed, and the one full-suite run this plan is required to produce found nothing new to fix.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

Both colonize existing-survey call sites now recover themselves when `AETHER_UNATTENDED=1` is set, which is exactly the fact the journey harness (`journeyRunClaudeWithRetry`, plan 09) already sets on every `claude` child it spawns. This plan's own scope was deliberately narrow — the one refusal that has stopped every real rehearsal so far — and it is committed, so a future owner-authorised journey walk measures this code. WINDOWS.md row 53 (the underlying survey-freshness finding) is unchanged by this plan and stays open until a live walk actually exercises this new path past the survey step; that walk was explicitly not run here (D-02/D-04 govern spending the owner's one remaining authorised walk, and this plan's own scope does not include running it).

---
*Phase: 208-never-a-dead-end*
*Completed: 2026-09-23*
