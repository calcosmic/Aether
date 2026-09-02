---
created: 2026-08-27T00:00:00Z
title: Worker turnaround is too slow — a one-plan job costs ~30 minutes
area: orchestration/performance
source: Owner feedback, 2026-08-27, during Phase 195 wave 5
resolves_phase: 201
priority: high
---

## The owner's observation (verbatim intent)

Using Aether, the helpers "seem to be running forever", where a plain
plan-and-execute session in another tool "moves a lot quicker". The owner called
this annoying and named fixing it a priority for Aether itself.

## Measured, not guessed (Phase 195, 2026-08-27)

Four executors on this machine, same repo, same day:

| Plan | Wall clock | Tool calls | Context |
|------|-----------|-----------|---------|
| 195-04 | 27 min | 143 | ~392k |
| 195-05 | 29 min | 174 | ~393k |
| 195-06 | 33 min | 177 | ~402k |
| 195-07 | 24 min | 152 | ~456k |

One full `go test ./cmd` run: **~12 minutes** (698s and 700s on two separate
clean runs).

## The four stacked causes

1. **The `cmd` package test suite takes ~12 minutes.** This is the single
   largest term and it is this repo's own debt, not the framework's. Already
   tracked in `.planning/phases/195-coherent-jobs/deferred-items.md`.
2. **The TDD contract multiplies test runs.** RED, GREEN, then a regression
   sweep, per task, two tasks per plan. That discipline is *why* this project
   stopped shipping unwired features — it is not waste — but it multiplies (1).
3. **The dispatched brief is enormous.** Roughly a dozen required-reading files
   including CLAUDE.md and multi-thousand-line planning documents, read before
   any work starts. ~150-180 tool calls and ~400k context per executor, much of
   it re-reading what the previous executor already read.
4. **Executors run on the cheaper model** (`executor_model: sonnet`), which
   needs more iterations to converge on intricate Go changes.

## Why the comparison is not apples-to-apples

The faster tool was running fewer gates: no per-plan write-up, no
prove-it-fails-first cycle, no full-suite check between waves. Some of that gap
is genuine waste (3 and 4). Some of it is precisely the discipline that this
project's own audits show it cannot safely drop — see CLAUDE.md's Definition of
Done and the 18-of-25-milestones-were-repairs finding.

**The goal is therefore to cut 3 and 4 and attack 1 — not to weaken 2.**

## New: the slow suite now has a second, harder cost (2026-08-27)

Phase 195 shipped the owner's ruling that a "nothing needed changing" claim is
only credited when the runtime RE-RUNS the check the worker named and sees it
pass. That re-run carries a 5-minute total budget per finalize pass; a check that
overruns is reported as *unavailable* and refused, because a check with no result
confirms nothing.

**This repo's own `cmd` suite takes 10-12 minutes.** So a worker whose honest
no-change check is `go test ./cmd` loses its credit here, on Aether itself. The
refusal is the safe direction and the work is not lost — the task goes into the
recovery job and is credited on a clean run — but it is a real cost that lands
directly on this repo, and it is caused by cause (1) below, not by the ruling.

Two levers, and the choice is the owner's once there is usage data: raise the
budget (trading build wall-clock), or fix the suite (which this item already
argues for on its own merits). Fixing the suite fixes both.

## PROGRESS 2026-08-28 — cause (1) attacked, suite cut 41% by one change

**Measured before:** `go test ./cmd` 482-586s; per-test profile showed the top 15
tests at ~62% of total, each 17-27s.

**Root cause found, and it was not "too many tests".** `newCLIBlackBox` in
`cmd/blackbox_harness_test.go` built the CLI **and** the deterministic adapter
per test, into a per-test `GOCACHE` under `t.TempDir()`. A fresh GOCACHE means a
cold compile every time. Measured on this machine: a cold build of `./cmd/aether`
takes **16.4s** against **1.1s** warm. Fifteen black-box tests were therefore
spending roughly four of the package's eight-plus minutes compiling the same
program fifteen times from scratch — re-testing the Go toolchain, not Aether.

**Fix (commit `a047e307`):** build both binaries ONCE per package behind a
`sync.Once` and hand every test the same paths. Only the compiled artifact is
shared — every test still gets its own home, repo, tmp and environment from
`t.TempDir()`, so no isolation a test depends on was weakened.

**Measured after:** the ten worst tests went 219s → 37s (6x). Whole `cmd`
package **482s → 284s (-41%)**. All eighteen packages green, zero failures.

**The profile is now flat**, which is the real signal: top 12 tests are 27% of
total (was: top 15 at 62%). The remaining ~268s is ~3,890 tests at roughly 50ms
each — ordinary test time, not waste. One legitimate outlier remains,
`TestPackedNPMReleaseCandidateContract` at 16.8s, which really does pack, install
and run the npm release; that is work, not waste.

**What this bought, beyond the clock:** a full-suite run happened ELEVEN times
during the 2026-08-27/28 session (between waves, after each fix pass). At ~200s
saved per run that is roughly 35 minutes of pure waiting per session of this
shape. It also widens the margin against the 5-minute no-change re-check budget
noted below.

**Cause (1) is not finished, but the cheap half is done.** What remains is
parallelism, and it is a real project rather than a one-liner: **zero of the 450
`cmd` test files call `t.Parallel()`** (3,926 top-level tests), and they cannot
simply be marked parallel — `saveGlobals` in `cmd/testing_main_test.go` shows the
suite swaps package-level globals (`store`, `stdout`, `stderr`, the flag vars)
and restores them, so two tests running at once would corrupt each other. The
work is removing that shared mutable state, not adding a line to 3,926 tests.
Three files elsewhere (`pkg/exchange`, `pkg/learn`) already use `t.Parallel`, so
this is not a project-wide policy — this package never got it.

## Candidate work

- **Fix (1).** `t.Parallel()` coverage, splitting `cmd`, or a documented
  long-run lane. Biggest single win and it compounds with everything else.
- **Fix (3).** Stop re-sending the whole planning corpus to every executor.
  The worker-handoff mechanism already exists for exactly this — carry forward
  a short relay note instead of making each worker rediscover the phase.
- **Fix (4).** Reconsider `executor_model` per plan complexity; a faster model
  that converges in half the turns can be cheaper in wall-clock AND tokens.
- **Measure it.** There is no turnaround metric today. The numbers above were
  gathered by hand from agent notifications. A per-plan duration and tool-call
  count belongs in the phase performance table automatically.

## Interim mitigations already applied (2026-08-27, this session)

Not a fix — recorded so a later reader knows the baseline shifted:

- Executor briefs trimmed to the files a plan actually needs.
- Executors told not to run broad test sweeps; the orchestrator owns the full
  gate.
- Executors moved off the cheap model for the remainder of Phase 195.
- Per-wave full-suite gates dropped for single-plan waves, where no parallel
  merge exists for that gate to catch. The phase-level gate still runs.
