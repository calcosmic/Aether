---
phase: 202-swarm-oracle-and-live-colony
reviewed: 2026-09-11T00:00:00Z
depth: standard
files_reviewed: 13
files_reviewed_list:
  - cmd/live_events.go
  - cmd/live_projection.go
  - cmd/live_recovery_episode_test.go
  - cmd/oracle_live.go
  - cmd/oracle_live_test.go
  - cmd/oracle_loop.go
  - cmd/recovery_orchestrator.go
  - cmd/swarm_cmd.go
  - cmd/swarm_lens.go
  - cmd/watch_live.go
  - cmd/watch_live_test.go
  - cmd/watch_replay.go
  - pkg/codex/dispatch.go
findings:
  critical: 0
  warning: 2
  info: 1
  total: 3
status: issues_found
---

# Phase 202: Code Review Report (Gap-Closure Round, plans 202-16/202-17)

**Reviewed:** 2026-09-11
**Depth:** standard
**Files Reviewed:** 13 (CLAUDE.md excluded from findings scope — doc-only diff, reviewed for factual accuracy against the code)
**Status:** issues_found (warnings only — no blockers)

## Summary

This round closes two gaps left by the earlier 202 review: (1) a recovery
decision fired mid-build/check used to always mint its own synthetic
"recovery-phase-N" episode, hijacking `aether watch` away from the real
running episode (CR-02); (2) Oracle's research loop never opened a
`live.episode.*` boundary at all, so a genuinely-in-progress research round
could never resolve `watchModeLive` (CR-01). Both are now fixed with narrowly
scoped, well-commented changes, and both fixes are proven by tests that
exercise `resolveWatchMode` end-to-end rather than only the helper functions
in isolation (`TestRecoveryDecisionKeepsTheBuildEpisodeLive`,
`TestOracleRoundIsLiveWhileItRuns`, `TestAbandonedOracleRoundIsNotLive`,
`TestWatchFollowsTheMostRecentlyStartedOpenEpisode`). I traced the
`foldColonyLiveEvents` open/close-balance refactor line-by-line against its
pre-diff form and confirmed the extracted `colonyLiveBoundaryDelta` /
`openColonyLiveEpisodeIDs` helpers preserve the exact floor-at-zero semantics
of the original `openBalance++`/`if openBalance>0 { openBalance-- }` code.
`go build ./...` is clean, and I ran the full `Live|Watch|Oracle|Swarm|Recovery`
test slice (`go test ./cmd/...`) plus the newly added tests individually — all
pass.

I found no blockers. Two warnings are latent-fragility notes about the
narrowness of the fixes (they solve the two named lanes but leave the same
class of bug reachable if a third lane is ever wired the same way), and one
info-level note about a benign double-emission edge case that is harmless
today only because of the pre-existing floor-at-zero balance rule.

## Warnings

### WR-01: Recovery-episode routing only knows about build and continue carriers

**File:** `cmd/live_events.go:363-371` (`currentLiveRecoveryEpisode`), consumed by `cmd/recovery_orchestrator.go:218-219`

**Issue:** `currentLiveRecoveryEpisode` decides where to record a recovery
decision by checking exactly two carriers — `currentLiveBuildEpisode()` and
`currentLiveContinueEpisode()` — and falls back to a synthetic
`recovery-phase-N` episode of its own whenever neither is set. Today that is
correct because `orchestrateRecovery` is only ever called from the build and
continue finalize paths (`cmd/codex_build_finalize.go`,
`cmd/codex_continue_finalize.go`, `cmd/queen_wave_lifecycle.go`) — confirmed
by grepping every `orchestrateRecovery(` call site. But nothing enforces that
invariant: Swarm (`cmd/swarm_cmd.go`, `cmd/swarm_lens.go`) and Oracle both run
their own dispatch loops and neither sets an "active recovery carrier," so if
recovery orchestration is ever extended to either lane (a plausible future
step given Swarm already has its own multi-attempt repair loop per
`TestThirdStrikeRendersAnArchitecturalCase` in CLAUDE.md), a recovery decision
fired mid-Swarm-run would silently regress into exactly the CR-02 bug this
plan just fixed: it would mint its own standalone episode and the live view
would not show the recovery detail layered on the real running Swarm episode.
The comment on `currentLiveRecoveryEpisode` documents the current mutual
exclusivity of the two carriers it does check, but says nothing about the
carriers it doesn't.

**Fix:** Either generalize the carrier lookup to a small ordered list of
`(kind, getter)` pairs that every live-emitting lane registers into (so a new
lane's carrier is automatically consulted), or add a code comment / structural
test (mirroring `TestEveryLiveEventGoesThroughOneBoundary`'s AST-walk idiom)
that fails by name if `orchestrateRecovery` ever gets called from a call site
outside the two known-safe finalize paths, forcing a deliberate update to
`currentLiveRecoveryEpisode` at the same time.

### WR-02: A killed background Oracle run can emit two `episode.ended` events for the same episode

**File:** `cmd/oracle_loop.go:567-608` (`stopOracleCompatibility`) vs. `cmd/oracle_loop.go:934-942` (`runOracleLoop`)

**Issue:** `aether oracle stop` (run from a separate CLI invocation) computes
the same deterministic `liveEpisodeID` from the on-disk `StartedAt` and
unconditionally emits `emitColonyLiveEpisodeEnded` for it after writing
`state.Status = "stopped"` and killing the controller process tree via
`terminateOracleProcessTree`. If the controller process is mid-loop and
receives `SIGTERM` (not `SIGKILL`) before the kill escalates, `runOracleLoop`'s
own `context.Done()` branch inside `runOracleLoopRounds` will itself reach
`finalizeOracleLoop(..., "stopped", ...)` and return normally, so the deferred
`closeEpisode(oracleLiveTerminalStatus(result, err))` in `runOracleLoop` will
*also* fire an `episode.ended` for the identical episode ID. Both writers race
independently against the event bus with no coordination between them. This
is currently harmless only because `foldColonyLiveEvents`'s balance floor
never lets `openBalance` go negative (a second `Ended` after the balance is
already 0 is a no-op) — but it is still two redundant writes racing on the
same file, and it is exactly the kind of "belt and suspenders masks a real
double-write" situation that is worth a comment or a de-duplication guard
before a future refactor of the balance logic removes the floor without
realizing this path depends on it.

**Fix:** Either have `stopOracleCompatibility` skip the explicit
`emitColonyLiveEpisodeEnded` call when it can detect the controller already
exited cleanly through its own boundary (e.g. check whether the state file's
`Status` was already `"stopped"`/terminal *before* this call's own write), or
add a one-line comment at the `emitColonyLiveEpisodeEnded` call in
`stopOracleCompatibility` noting that a second `Ended` for the same episode is
an expected, harmless possibility that depends on the balance floor in
`foldColonyLiveEvents` — so a future change to that floor doesn't
unknowingly break this path.

## Info

### IN-01: `openColonyLiveEpisodeIDs` duplicates `foldColonyLiveEvents`'s schema-skip and boundary logic by hand rather than by delegation

**File:** `cmd/live_projection.go:338-370`

**Issue:** `openColonyLiveEpisodeIDs` re-implements the same
"skip on unrecognized schema version, accumulate via
`colonyLiveBoundaryDelta`" logic that `foldColonyLiveEvents` also implements
inline (lines 391-397 and 410-416). The two are proven consistent today by
`TestOneOpenBalanceRule`, and the shared `colonyLiveBoundaryDelta` function
does remove the risk of the increment/decrement rule itself drifting apart —
but the schema-version skip condition (`payload.SchemaVersion != "" &&
payload.SchemaVersion != events.ColonyLiveSchemaVersion`) is still copy-pasted
verbatim in both places rather than factored into one shared predicate. A
future edit to the version-skip rule in one of the two places (e.g. to handle
a new "deprecated but still counted" version) could silently miss the other.

**Fix:** Extract a small `colonyLiveEntryRecognizedSchema(payload) bool`
helper both `foldColonyLiveEvents` and `openColonyLiveEpisodeIDs` call, so
there is exactly one place that decides what "recognized schema version"
means, matching the pattern this same diff already used for the boundary
delta rule.

---

_Reviewed: 2026-09-11_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
