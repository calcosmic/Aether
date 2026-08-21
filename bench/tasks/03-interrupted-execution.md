# Task 03: Interrupted execution

## Substrate

`gorilla/mux` (Go), pinned commit `db9d1d0073d27a0a2d9a8c1bc52aa0af4374d265`
(same substrate as Task 01; see `bench/tasks/substrate.md`).

## The task

**This prompt text is identical to Task 01's, handed to all three lanes
verbatim, written the way a real developer would be briefed, and containing
none of either system's internal vocabulary.**

> One of the tests in this repository is failing:
> `TestCleanPathPreservesNonSlashInput`. Fix the bug it caught so that test
> passes. Do not change the test itself — the test is correct as written;
> the bug is in the code it is testing.

This task category reuses Task 01's exact prompt and exact seeded defect
(see `bench/tasks/01-bug-fix.md`'s "Setup" section for the precise patch)
on purpose — the only variable this category tests is what happens when the
run is killed partway through and then resumed, not whether the underlying
fix is harder or easier.

## Setup

Identical to Task 01's setup: the harness applies the seeded defect to
`mux.go` and the seeded failing test to `mux_test.go` on a fresh clone,
before the system under test ever sees the repository.

**The kill rule (identical across all three lanes, by design):**

The harness watches the substrate clone's filesystem. At the moment of the
very first file write inside that clone — the first time any file in the
working copy changes on disk after the run starts — the harness starts a
120-second timer. When that timer expires, the harness sends `SIGKILL` to
the running system's entire process group (not just its top-level process,
so no child process survives to keep working unsupervised).

The operator then issues **exactly one** documented resume command and
nothing else:

| Lane | Resume command |
|---|---|
| GSD | `/gsd-resume-work` |
| Aether interactive | `/ant-resume`, then `/ant-recover` |
| Aether autopilot | `/ant-resume`, then `/ant-recover` |

The two Aether lanes use the same two-command sequence because both are the
same underlying system reached through two different entry points; only the
GSD lane's command differs, and only in name — every lane gets exactly one
kill, exactly one scripted resume action, and no other operator input.

The `/ant-resume` and `/ant-recover` commands used above are real, existing
commands in this repository, confirmed present at `.claude/commands/ant/resume.md`
and `.claude/commands/ant/recover.md` — not assumed or invented for this
spec.

**Any operator input beyond the one documented resume command/sequence for
that lane counts as an unscripted intervention** and is logged as such; it
does not disqualify the run, but it must never be silently absorbed into a
"the system resumed successfully" result.

## Done means

- The seeded test, `TestCleanPathPreservesNonSlashInput`, passes:
  `go test -run TestCleanPathPreservesNonSlashInput -v ./...` exits 0.
- The full suite, `go test ./...`, exits 0.
- `mux_test.go` is unchanged from what the setup step wrote.
- Exactly one resume action was issued per the table above, and any
  operator input beyond it is present in the run's logged operator input,
  not hidden.
- The harness's own kill-and-resume timing is recorded: the timestamp of
  the first file write, the timestamp `SIGKILL` was sent (120 seconds
  later), and the timestamp the resume command was issued.

## Allowlist

`bench/tasks/allowlists/03-interrupted-execution.txt`

## Per-lane invocation

- **GSD:** `/gsd-execute-phase` against a plan produced for this bug fix;
  killed 120 seconds after its first file write; resumed with
  `/gsd-resume-work`; followed, once complete, by `/gsd-verify-work`.
- **Aether interactive:** `/ant-build <phase>` against a phase describing
  this fix; killed 120 seconds after its first file write; resumed with
  `/ant-resume` then `/ant-recover`; followed, once complete, by
  `/ant-continue`.
- **Aether autopilot:** `/ant-run` against a colony initialized with this
  task as its goal; killed 120 seconds after its first file write; resumed
  with `/ant-resume` then `/ant-recover`.
