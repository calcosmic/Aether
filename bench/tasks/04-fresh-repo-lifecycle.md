# Task 04: Fresh-repo full lifecycle

## Substrate

None — this task starts from a brand-new, completely empty git repository
(`git init` into an empty directory), not a clone of either substrate repo
named in `bench/tasks/substrate.md`. There is nothing to pin a commit SHA
to; the starting point is deliberately nothing.

## The task

**This prompt text is identical across all three lanes — the same product
goal, handed to a human developer, containing none of either system's
internal vocabulary. Only the sequence of commands used to hand it to each
system differs, recorded separately below in "Per-lane invocation".**

> Build a small, self-contained command-line program that converts a
> temperature given in Celsius to Fahrenheit and back. It should accept a
> number and a unit letter (`C` or `F`) as input and print the converted
> value in the other unit. Include automated tests that check the
> conversion math is correct in both directions, including at least one
> negative-number case and the two fixed points (0°C = 32°F and
> 100°C = 212°F). The program should have its own short instructions for
> how to run it.

## Setup

None beyond `git init` in a fresh, otherwise-empty directory. Because each
lane's full lifecycle genuinely differs (one installs and initializes a
colony, the other sets up a new project and runs a phase cycle), this task
category's "Setup" is the same nothing-to-prepare starting point for every
lane; everything else about how a lane gets from nothing to done is
recorded in "Per-lane invocation" below, not here.

## Done means

- The empty starting directory now contains a working command-line program
  matching the product goal above, plus its automated tests.
- Running the project's own test command (as documented in whatever the
  system wrote for "how to run it") exits 0.
- Manually running the resulting program confirms both fixed points:
  converting `0 C` prints `32` (Fahrenheit) and converting `100 C` prints
  `212`; converting back (`32 F` to Celsius, `212 F` to Celsius) reproduces
  the original values.
- **The extra assertion this category specifically requires:** after an
  Aether-lane run finishes, running `git status --porcelain` inside the
  Aether source repository itself (not the fresh project directory the
  lane just built) produces no output — completely empty. This proves
  Aether did not modify its own source repository while it was working on
  someone else's project. This check does not apply to the GSD lane, which
  has no equivalent "the tool's own repository" to protect in this
  benchmark's setup.

## Allowlist

`bench/tasks/allowlists/04-fresh-repo-lifecycle.txt`

## Per-lane invocation

- **GSD:** `/gsd-new-project` to set up the fresh directory, followed by
  the standard phase cycle for this one small goal — `/gsd-discuss-phase`,
  `/gsd-plan-phase`, `/gsd-execute-phase`, `/gsd-verify-work` — through to
  completion.
- **Aether interactive:** install the `aether` CLI into the fresh
  directory, then `/ant-init` with this task's goal, `/ant-plan`,
  `/ant-build`, `/ant-continue` through to completion, then `/ant-seal`.
- **Aether autopilot:** install the `aether` CLI into the fresh directory,
  then `/ant-init` with this task's goal, `/ant-plan`, then `/ant-run`
  through to completion, then `/ant-seal`.

After each Aether-lane run completes, run `git status --porcelain` in the
Aether source repository itself (the repository this harness lives in) and
record whether it is empty, per the "Done means" assertion above.
