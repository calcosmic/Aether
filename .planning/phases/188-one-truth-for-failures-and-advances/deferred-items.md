# Deferred Items — Phase 188 Plan 01

Out-of-scope discoveries logged per the executor's deviation rules (scope boundary: only
auto-fix issues directly caused by the current task's changes; log everything else here
instead of fixing it).

## 1. `cmd/context_bench_test.go:159` seeds the dead nested midden path

`setupBenchmarkStore` (used only by `BenchmarkPRContext`, `BenchmarkColonyPrime`,
`BenchmarkProof`) seeds `midden/midden.json` via `s.SaveJSON(...)`. After this plan's fix,
`pr-context` (and every other consumer) reads the canonical flat `midden.json`, so these
benchmarks now measure the "midden empty" code path instead of "midden populated" —
benchmark realism only, not correctness. No `TestX` function reads this fixture, so
`go test ./cmd/...` is unaffected; benchmarks only run under `-bench`, which is not part of
this plan's or the repo's standard verification commands.

**Recommendation:** move the seed to the flat path in a future benchmark-maintenance pass.
Not fixed here — zero required verification command depends on it, and it is not a
regression this task's changes directly caused (the benchmark fixture was already
disconnected from correctness, only from performance realism).

## 2. `cmd/colony_prime_audit_test.go` and `cmd/medic_repair_test.go` seed the nested path too

Both were read in full and confirmed **not broken** by this plan's changes — not logged as
issues, just noted here for whoever next touches midden plumbing:

- `colony_prime_audit_test.go` (lines 139/234, 369): the read-back at line 234 uses
  `s.LoadJSON("midden/midden.json", ...)` directly (not `loadMiddenFile`), round-tripping
  against its own seed — self-consistent regardless of this plan's changes. The seed at 369
  (`TestColonyPrimeSectionsPresent`) is never asserted on (`buildColonyPrimeOutput`'s expected
  section list has no "midden" entry). Both are dead weight, not defects.
- `medic_repair_test.go` (lines 32/45, 663): line 32/45 exercise `createBackup`, which copies
  the whole data directory generically and does not consult `scanDataFiles`'s file list at
  all. Line 663 seeds `midden/midden.json` for `TestPerformRepairsIntegration`; after this
  plan's fix, `scanDataFiles` now additionally reports a non-fixable "midden.json not found"
  info-level issue there, which does not affect either `Healthy` (critical-only) or the
  `fixableCount`/`remainingFixable` comparison the test actually asserts on.

# Deferred Items — Phase 188 Plan 06 (code review fixes: CR-01, CR-02, CR-03, WR-01)

## 3. `cmd/codex_build.go`'s `rollbackCodexBuildFailure` discards its own fresh read too

While extending the atomicity ratchet for WR-01 (188-REVIEW.md), the new
`TestNoUpdateJSONAtomicallyDiscardsFreshRead` check — scanned across all of `cmd/*.go`, not
scoped to any one of the four items this plan was dispatched to fix — found a fourth live
instance of the same defect class, outside this plan's explicit scope (CR-01:
`cmd/codex_continue_finalize.go`, CR-02: `cmd/codex_build_finalize.go`, CR-03:
`cmd/state_cmds.go`):

`cmd/codex_build.go`'s `rollbackCodexBuildFailure` (the function that reverts colony state
after a build's worker dispatch fails) does:
```go
var current colony.ColonyState
if err := store.UpdateJSONAtomically("COLONY_STATE.json", &current, func() error {
    if err := validateRuntimeStateStillCurrent(current, phaseNum, &startedAt, colony.StateEXECUTING, colony.StateBUILT); err != nil {
        return err
    }
    rollback.Worktrees = mergeBuildFailureWorktrees(rollback.Worktrees, current.Worktrees)
    current = rollback
    return nil
}); err != nil {
```
It DOES call `validateRuntimeStateStillCurrent` first (so a concurrent pause, phase change, or
state-machine drift is caught and refused), but then discards the fresh read's OTHER fields —
anything not covered by that specific check — by reassigning `current = rollback`, where
`rollback` is built from a `previous` parameter captured before this call. Narrower than CR-02
(a real guard exists here, unlike CR-01/CR-02's original unguarded overwrites), but
structurally the same clobber shape the ratchet now polices.

**Not fixed here** — `cmd/codex_build.go` is not one of this plan's four assigned findings, and
understanding this function's full rollback contract (worktree merge-failure recovery,
interaction with `beginBuildAttempt`/attempt journaling) is out of the context budget for a
plan already touching four other files. Per the executor's scope-boundary rule, out-of-scope
discoveries are logged, not fixed.

**Handled without silently hiding it:** added to a small, explicit, shrink-only
`colonyStateDiscardedReadAllowlist` in `cmd/colony_state_atomicity_ratchet_test.go` (NOT the
JSON-file-backed baseline `testdata/colony_state_write_allowlist.json` — a separate, in-file
list scoped to exactly this one pending item), with a doc comment pointing back here. The
allowlist is shrink-only in both directions: a stale entry (once this site is fixed) fails the
test just as loudly as an unlisted new one would.

**Recommendation:** a future plan should apply the same fix shape used here for CR-01/CR-02 —
mutate `current`'s own fields (or a small, explicit list of fields `rollback` actually needs to
apply) instead of reassigning `current` wholesale from `rollback`, then remove the
`colonyStateDiscardedReadAllowlist` entry once fixed.

## 4. `cmd/codex_build.go` has a pre-existing `gofmt` struct-alignment drift

`gofmt -l cmd/*.go` flags `cmd/codex_build.go` (a `codexBuildDispatch` struct field
alignment gap around line 23). Confirmed pre-existing and unrelated: `git diff --stat
cmd/codex_build.go` shows zero changes from this plan's work, and the drift is a pure
whitespace/alignment issue with no behavioral effect. Not fixed here (out of scope, unrelated
file) — a future formatting pass (`gofmt -w cmd/codex_build.go`) can pick it up in one line.
