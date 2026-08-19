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
