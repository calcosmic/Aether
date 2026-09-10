---
phase: 200-iterative-planning
plan: 55
status: partial
completed: 2026-09-10
deviation: Task 3's 11-minute wall-clock truth is machine-infeasible for the current corpus; evidence below
tasks_completed: 2
tasks_total: 3
---

# Plan 200-55 Summary: Bounded Complete Suite Harness

## What was delivered

**Task 1 — exact-once partition and failure propagation: DONE.**
An unfiltered default `go test ./cmd` run enters a controller in `TestMain`
that discovers every top-level test from the compiled binary, partitions the
complete set exactly once into lanes, and re-executes the *current*
(race-instrumented when applicable) binary per lane via a recursion sentinel.
Focused `-run`/bench/fuzz/profile/explicit-parallel invocations bypass the
controller entirely. Any child nonzero exit, panic, or timeout fails the
parent with the child's full output replayed deterministically. Locked by
`TestFullSuiteRuntime200(PartitionExactOnce|PreservesFocusedRun|PropagatesFailure|UsesCurrentBinary)`.

**Task 2 — isolation and serialization: DONE.**
Each child receives its own `AETHER_HUB_DIR` with parent store/root scrubbed
(probes tolerate empty-string residue, which the runtime treats as unset).
An explicit, evidence-backed serial inventory (11 entries) runs alone before
any parallel lane; `TestMain` pins `AETHER_PLATFORM=codex` unless the caller
set one, so test outcomes no longer depend on which terminal launches the
suite. Locked by
`TestFullSuiteRuntime200(IsolatesChildEnvironment|SerializesSharedResources|AdversarialOrder|ReportsCompleteAccounting)`.

**Regressions the complete suite exposed — all fixed and committed:**
the suite had never run to completion, so a backlog of real defects surfaced
as coverage extended. Fixed here:

1. `pending-decisions.json` schema union — the strict lifecycle decoder
   rejected the blocker-flag fields that legitimately share the file
   (broke overnight autopilot).
2. `spec --repair-projection` (and discuss settlement) opened no planning
   mutation session, so every drift repair failed.
3. Whole-failure `build-finalize` was impossible to commit: two guards
   demanded a partial-retry plan that zero-credit dispatches can never have
   (D-10: retry follows accepted credit).
4. Acceptance's derived-authority agreement was incompatible with the
   declaration-wrapper card hashes for phases/tasks and root aggregates;
   second-round planning journeys (revise an already-built plan) could not
   be accepted, erased completion credit when they were, and then wedged
   build authority as permanently "unreconciled". Three symmetric fixes in
   `plan_impact.go`/`plan_revision.go`; both CLI blackbox revision journeys
   now pass.
5. Shelf commands failed on a fresh repository (missing-file detection did
   not match storage's error wording).
6. Recovery-orchestrator fixtures predated the canonical build-start
   contract (root/owner/mode); the preflight source-order pin predated
   cleanup's move into `commitBuildStart` (200-34).
7. Platform-dependent Phase-199 tests (vocabulary routes, init escape
   hatch) pinned explicitly.
8. The Phase-199 gate receipt validator broke when `.planning/config.json`
   was committed; the commit was undone to preserve the recorded
   pre-existing-dirt fingerprint.

**Task 3 — both exact commands under the 11-minute ceiling: NOT ACHIEVABLE
ON THIS MACHINE, with measurement, not conjecture.**

## The evidence

Six configurations measured (uncached, wall-clock, this machine — M-series,
10 cores):

| Configuration | Wall | Executed | Outcome |
|---|---|---|---|
| Classic single process, `-parallel=10 -timeout=25m` | 25:02 | timed out incomplete | the pre-controller mode; was 7:20 on 2026-09-06 before the 200-series serial heavies added ~20 min |
| 5 workers, 8 lanes | 10:04 | 1,449 / 4,794 | internal 10m cap |
| 12 workers, 48 lanes | 10:03 | 3,453 | best throughput observed (~345 tests/min) |
| 24 workers, 120 lanes | 10:03 | 3,251 | spawn storm: 7× more kernel than user time; heavy lanes slowed into their own timeouts |
| 12 workers, 8 fat children (10-wide) | 10:35 | 2,073 | concurrent children drop to ~41 tests/min each (vs ~190 alone) |
| 12 workers, 48 cost-packed lanes, uncapped | 18:03 | 4,473 | still incomplete at 18 min; zero real failures |

Two binding constraints, both now identified exactly:

1. **The "external 11-minute ceiling" is the `go` tool itself.** The bare
   Plan-40 commands carry Go's default `-timeout=10m`; after that deadline
   plus its kill grace, the `go test` tool delivers SIGQUIT to the test
   process from outside (observed at 11:04 wall on a bare run). No
   `TestMain` arrangement escapes it — the ceiling is immovable for the
   exact commands.
2. **Kernel-side process-creation serialization** caps the machine near
   ~345 tests/min regardless of worker count: the suite's expensive tests
   each spawn real `aether`/`git` subprocesses, and concurrent children
   drop from ~190 tests/min alone to ~41 tests/min each at four-wide.
   4,794 tests therefore floor near ~14–19 minutes complete. The slowest
   single test (`TestClassicContractPhase200CausalExecution`) runs 4:33
   alone and ~8:45 under any concurrent load — under `-race` this one test
   approaches the ceiling by itself.

## The owning fix (out of this plan's scope)

Per this plan's own scope guard ("tune only bounded concurrency/grouping,
never test selection"), the remaining gap belongs to the tests themselves:
~700 tests ≥2s (238 load-minutes combined) are dominated by serial
subprocess spawning and real timeout waits. Reducing their cost (shared
compiled binaries, in-process command invocation instead of spawning, faster
poll intervals) is a corpus-wide repair — a distinct plan, not a scheduling
knob.

## Final state

The harness ships in its complete-truthful configuration: 12 workers,
6 heavy workers, 48 cost-balanced lanes (707 harvested weights in
`cmd/full_suite_costs_generated_test.go`). Its internal overall ceiling is
set just under the go tool's SIGQUIT deadline, so a bare
`go test ./... -count=1` produces an ORDERLY bounded result — complete
per-lane accounting of exactly what ran and what could not, discovered vs
executed counts printed, failures replayed — instead of a signal-killed
corpse with no accounting.

Given an explicit generous `-timeout`, the same command runs the ENTIRE
suite — nothing skipped, nothing hidden, exact-once accounting. After the
work below, the definitive consolidated receipt run (2026-09-10, revision
`68a6fd16`) executed **all 4,794 discovered tests GREEN, both normal
(18:56) and under the race detector (19:00), zero failures, zero data
races** — the first complete, green, race-clean execution of this suite in
the project's recorded history.

Reaching a *reliable* green took two further pieces beyond the raw
scheduler. First, the controller now derives its overall ceiling from the
caller's `-test.timeout` and stops orderly (full per-lane accounting) just
under the go tool's SIGQUIT deadline, so a bare command fails tidily at
~9m15s while a generous explicit budget runs the whole corpus. Second, a
family of timing/ordering-sensitive tests (byte-identical golden output,
live heartbeat intervals, exact registries) failed only under peak CPU
contention. Rather than chase them one per 19-minute run, the root cause was
removed: light-child parallelism dropped from 8-wide to 3-wide (GOMAXPROCS
3), halving peak concurrency on a 10-core machine, which calmed the whole
class at once while the heavy lanes (~9m) stayed the wall-time pole so total
time was unchanged. A handful of confirmed load-flakes also joined the
serial lane, and three `pkg/codex` preflight retry tests (outside the cmd
controller) were made load-robust by widening a 500ms-1.5s probe budget to
8s — proving the identical retry semantics (the hang path is `sleep 30`)
while surviving a starved CPU. **Zero deterministic test failures remain in
the whole corpus.**

Plan 40's receipt rerun cannot truthfully record a sub-11-minute full/race
gate on this machine; the phase carries this as an open item for the
owner's decision.
