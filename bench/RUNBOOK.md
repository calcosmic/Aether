# Operator runbook: the benchmark runs

> **RESCOPED 2026-08-18 — read this before anything else.**
>
> This runbook was written for twelve runs. **Only one is now required:**
>
> ```
> bench/run.sh --lane aether-interactive --category 01-bug-fix
> ```
>
> The head-to-head comparison against GSD was cut by owner decision. The
> repair work it was meant to inform (phases 187–191) turned out not to need
> it — each of those has its own pass/fail test. What was kept is a single
> honest run of Aether on a real open-source project it has never seen,
> fixing a small deliberately-planted bug. It answers one question: can
> Aether finish a small real job without someone nudging it? That is a smoke
> alarm, not a scoreboard.
>
> **Everything below still applies to that one run** — the setup checks, the
> permitted-inputs discipline, the recording of the terminal session. Skip
> the "The order to run them in" section entirely; there is no ordering
> question with one cell. The other eleven cells still work and can be run
> later with the same command if anyone wants them.
>
> The full comparison lives in Phase 192, at the end of the programme.

This is the procedure a human follows to produce all twelve benchmark runs
(three ways of running the tools under test, times four kinds of coding
task). It is written for someone who is sitting at the keyboard right now
and does not want to stop and reason about how the underlying scripts work
while doing it — every repo-specific word is explained the first time it
appears.

Two systems are being compared here: **Aether** (this repository's own
tool, run two different ways — "interactive" and "autopilot") and **GSD**
("get-shit-done," a separate development tool with no connection to
Aether). "The harness" below means the scripts under `bench/` that drive
both systems and record what happened; "a cell" means one single
combination of one way-of-running-the-tool ("lane") and one task
("category") — twelve cells in total.

## Before you start

Complete these one-time checks before running any cell:

1. **The hermetic smoke passes.** "Hermetic" means sealed off and isolated
   — each run happens inside a brand-new, throwaway home directory so
   neither system can see the other's memory or the operator's own
   machine setup. Run:
   ```
   bench/smoke-hermetic.sh
   ```
   This must exit 0 before anything else in this runbook is trusted. See
   `bench/README.md`'s "The first gate: hermetic smoke" section if it
   fails.

2. **The acceptance-ordering checker passes.** "Acceptance" is the
   automatic judge that decides, after a run finishes, whether the task
   was actually done — one independent judge script per task, written and
   committed before any run happened, specifically so no judge could be
   quietly shaped around a result it already knew about. Run:
   ```
   bench/acceptance/verify-predates-runs.sh
   ```
   `bench/run.sh` (the single command used to run a cell — see "Running
   one cell" below) already re-checks this automatically and refuses to
   run if it fails, but confirm it here too before starting, so a broken
   ordering claim is caught before any time is spent on a run.

3. **The model is chosen and exported as `BENCH_MODEL`.** Both systems
   must use the exact same underlying AI model for the comparison to be
   fair — this is called model parity. Export it once per terminal
   session before running any cell:
   ```
   export BENCH_MODEL=<the chosen model id>
   ```
   `bench/run.sh` refuses to run without this set.

4. **The terminal is set up to save a full transcript.** After each cell,
   the harness needs a plain text copy of everything the system under test
   printed to the terminal, saved to a specific file, so it can check
   whether the system *claimed* success (a fact recorded from that saved
   text, never trusted on its own — see "After each cell" below). Use the
   `script` command (built into macOS and Linux) to record the session:
   ```
   script -q /path/to/claimed-success-capture.txt
   ```
   Run the lane's commands inside that recording, then exit the recording
   (`exit` or Ctrl-D) once the run is done. The exact file path convention
   the harness's cell runner (`bench/harness/run-cell.sh`) expects for
   this is `$WORK/claimed-success.txt`, where `$WORK` is the cell runner's
   own private throwaway working directory (printed at the start of each
   cell's output) — copy your saved transcript there before the cell
   finishes writing its evidence.

## The order to run them in

All twelve cells, listed in the order they are run:

1. `gsd` / `01-bug-fix`
2. `gsd` / `02-brownfield-feature`
3. `gsd` / `03-interrupted-execution`
4. `gsd` / `04-fresh-repo-lifecycle`
5. `aether-interactive` / `01-bug-fix`
6. `aether-interactive` / `02-brownfield-feature`
7. `aether-interactive` / `03-interrupted-execution`
8. `aether-interactive` / `04-fresh-repo-lifecycle`
9. `aether-autopilot` / `01-bug-fix`
10. `aether-autopilot` / `02-brownfield-feature`
11. `aether-autopilot` / `03-interrupted-execution`
12. `aether-autopilot` / `04-fresh-repo-lifecycle`

**Why this order:** complete all four tasks for one lane (one way of
running one tool) before moving to the next lane. This means each lane's
one-time setup — installing that lane's system into its own isolated home
directory, learning that lane's specific command sequence — is only paid
for once per lane, not re-paid every time the task category changes.

**The counter-note that matters:** this exact ordering — `gsd` first, then
`aether-interactive`, then `aether-autopilot`, and the four task categories
in the same 01→02→03→04 sequence within each lane — must be used
identically for every lane. If one lane were run in a different task order
than another, the operator's growing familiarity with the harness itself
(not with either system) could make the later-run lane look artificially
faster or smoother. Running the same sequence for all three lanes cancels
that effect out.

## Running one cell

The exact command for any one cell:
```
bench/run.sh --lane <lane> --category <category>
```
Example: `bench/run.sh --lane gsd --category 01-bug-fix`.

What the harness prints and where it pauses, in order:

1. A fresh clone of the task's real-world test project (or, for
   `04-fresh-repo-lifecycle`, a brand-new empty project) is created, and
   — for `01-bug-fix` and `03-interrupted-execution` — the one seeded bug
   and its matching failing test are written into it.
2. The lane's isolated home directory is set up and that lane's system
   (Aether or GSD) is installed fresh into it, with the pinned model
   applied.
3. The list of inputs the operator is allowed to give for this specific
   cell is printed to the terminal — this is also written down in advance
   in `bench/harness/permitted-inputs.md`; see "What you may type" below.
4. For `03-interrupted-execution` only: a background watcher starts,
   silently waiting for the first file to be written.
5. **The harness pauses and hands control to the operator.** This is where
   the operator runs the lane's own commands (see the task's own file
   under `bench/tasks/` for the exact command sequence for that lane) —
   the harness does not drive the interactive session itself; it prints
   the commands, waits, and resumes only when told to.
6. **Signal completion by pressing Enter** in the terminal running
   `bench/run.sh` once the lane's own work is finished (or once the run
   was killed and resumed, for the interrupted-execution category — see
   below).
7. The harness resumes, measures everything by script (never by asking
   either system how it did), judges the result against a second, fresh
   clone, and writes the evidence.

## What you may type

The complete, cell-by-cell list of what the operator is allowed to type or
click during each run lives in `bench/harness/permitted-inputs.md` — read
the section for the specific lane and category being run before starting
it.

**The rule, restated plainly:** anything typed that is not on that list for
that specific cell counts as an unscripted intervention. It gets logged as
one automatically. **Logging an intervention is not a failure of the
operator — it is a measurement, and the honest number is the whole point of
this benchmark.** If a system genuinely needed help beyond what was
scripted, the correct thing to do is help it, note that it happened, and
let the number reflect that. Suppressing an intervention — quietly not
mentioning that something extra was typed — to make one lane look more
autonomous than it really was destroys the entire comparison; the result
would no longer mean what the results table says it means.

## The interrupted-execution runs

This applies to category `03-interrupted-execution`, run once per lane
(cells 3, 7, and 11 in the order above).

**What it looks like when it happens:** partway through the run, without
warning, the process running the system under test is killed outright
(`SIGKILL` — an immediate, unstoppable stop, not a request the system could
catch and clean up after). This is expected and is the entire point of
this task category: it tests whether the system can pick back up correctly
after being interrupted mid-task, the same way a real developer's laptop
might lose power or a terminal might get closed by accident.

**The exact rule (identical for every lane):** the harness waits for the
very first file inside the project to actually change on disk, then waits
120 more seconds, then kills the process. The operator does not need to
watch for this or trigger it — the harness's background watcher does it
automatically.

**The single resume command per lane** — issue exactly this, and nothing
else, once the kill has happened:

| Lane | Resume command |
|---|---|
| gsd | `/gsd-resume-work` |
| aether-interactive | `/ant-resume`, then `/ant-recover` |
| aether-autopilot | `/ant-resume`, then `/ant-recover` |

Any operator input beyond the single documented resume command/sequence for
that lane is an unscripted intervention (see "What you may type" above) —
it does not disqualify the run, but it must be logged, not silently
absorbed into "the system resumed successfully."

## After each cell

Before moving on to the next cell, confirm:

1. **`run.json` was written.** This is the one file per cell that holds
   every measured number for that run — it should exist at
   `bench/results/<today's date>/<lane>__<category>/run.json` once the
   cell finishes.
2. **The raw evidence was copied out.** Alongside `run.json`, the same
   folder should contain the saved session transcripts, the operator input
   log, and the acceptance judge's detailed output — all written by
   script, not typed by the operator.
3. **Do not edit anything by hand.** Not `run.json`, not the transcripts,
   not the results table (generated later, see "After all twelve" below).
   Every number in this benchmark must trace back to something a script
   wrote from watching the actual run happen — a hand-edited number breaks
   that chain and makes the whole result untrustworthy.

## After all twelve

Once every cell has a `run.json`:

1. **Generate the table:**
   ```
   bench/results/generate-table.sh bench/results/<today's date>
   ```
   This reads every `run.json` under that dated folder and produces the
   final markdown results table — never hand-typed.
2. **Commit the dated results directory** (`bench/results/<today's
   date>/`) so the raw evidence and the generated table are both saved in
   the project's history, not just on one machine.
3. **Re-run the ordering checker** (`bench/acceptance/verify-predates-runs.sh`)
   to confirm it still passes now that real results exist — this is the
   check that proves every judge script really was written and committed
   before it ever saw a result, and it is worth re-confirming with real
   data in place, not just trusting the earlier vacuous pass from "Before
   you start."

## If something goes wrong

**Authentication expires mid-benchmark.** Both systems eventually need to
re-confirm they are allowed to talk to the AI provider. If a cell's run
stops because of an authentication error partway through, do not try to
patch it live inside the isolated home directory — stop that cell, fix
authentication in the harness's provisioning step (see
`bench/README.md`'s "Known environment risks" section for the documented
fix), and start that cell over as a fresh, complete run.

**A substrate test suite goes red for a reason that has nothing to do with
the system under test.** ("Substrate" is the real, unrelated open-source
project each task is run against — see `bench/tasks/substrate.md`.) If the
project's own pre-existing tests fail before either system has touched
anything (for example, a flaky test unrelated to the task, or a dependency
that failed to install), that is an environment problem, not a result.
Record what happened, fix the environment issue, and start that cell over
as a fresh run — do not let a broken starting point silently become part
of either system's score.

**The harness pauses somewhere the operator did not expect.** If it is
unclear whether the harness is waiting on the operator or on the system
under test, check the terminal output for the most recent message — the
harness always states plainly when it is handing control over and always
states what it is waiting for. If it still is not clear, it is safer to
wait than to type something outside the permitted list.

**The one rule that overrides all of the above:** a cell that is abandoned
partway through — for any reason — is recorded as abandoned, with the
reason written down, and it is never silently re-run in its place without
that record. Quietly re-running a cell that had a real problem and
reporting only the second, cleaner attempt turns one bad result into a
better-looking one without saying so — which is exactly the kind of hidden
thumb-on-the-scale this whole harness exists to prevent.
