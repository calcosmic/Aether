---
created: 2026-08-27T00:00:00Z
title: Worker turnaround is too slow — a one-plan job costs ~30 minutes
area: orchestration/performance
source: Owner feedback, 2026-08-27, during Phase 195 wave 5
resolves_phase: ""  # not yet scheduled — owner named this a priority; route via /gsd-discuss-phase at the next planning pass
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
