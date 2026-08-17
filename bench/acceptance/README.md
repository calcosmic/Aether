# Acceptance scripts

## What these are

This folder holds an independent judge for each of the four benchmark tasks
in `bench/tasks/`. Each judge is a small program that decides, by running
the actual code a system produced, whether that task was really done — not
by reading a report the system wrote about itself, and not by anyone's
opinion. Every judge either exits with a success code (the task passed) or
a failure code (it didn't), and every failure is printed in plain language
naming exactly what it expected and what it actually found.

There is one judge per task:

- `01-bug-fix.sh` — judges the small bug-fix task
- `02-brownfield-feature.sh` — judges the multi-file feature task
- `03-interrupted-execution.sh` — judges the "killed partway through, then
  resumed" task
- `04-fresh-repo-lifecycle.sh` — judges the build-something-from-nothing task

Each one is called with a single argument: the path to a fresh copy of the
project a run produced. It then runs the project's real tests, runs the
actual program, and checks the actual files on disk. Nothing about the
verdict depends on which of the three systems under comparison produced
that project, or on what that system claims about its own success.

## Why they were written before any run happened

If a judge is written after seeing how a run turned out, there's a strong
and very human temptation to quietly shape the judge around what actually
happened — loosen a check that a favored system almost passed, or tighten
one a disfavored system barely failed. Writing the judge first, before any
run exists to be tempted by, removes that possibility entirely: the rules
were fixed before anyone knew the outcome.

## How to check that claim yourself

Run this one command:

```
bench/acceptance/verify-predates-runs.sh
```

It reads this project's own history log and proves two things about every
judge script in this folder: it was first saved (committed) before any run
result existed, and it was never edited again after the first run result
appeared. If either of those isn't true, the command fails loudly and names
exactly which judge and which run result broke the rule, with real dates —
not hidden, not summarized away. If no run has happened yet, the command
still succeeds, but says plainly that the check hasn't really been put to
the test yet (there's nothing to compare against), rather than pretending
that's a real pass.

## The vocabulary rule

None of the judges in this folder use the internal words either compared
system uses to describe its own checking or its own files — the exact
banned words are: `must_have`, `must-have`, `criterion` or `criteria` (as
a system term), `phase`, `colony`, `worker`, `pheromone`, `instinct`,
`wave`, `plan` (as a system artifact), `SUMMARY.md`, `PLAN.md`, and
`COLONY_STATE`. This matters because letting a judge lean on either
system's own vocabulary for deciding pass or fail would quietly hand that
system's own conventions the final word on its own grade — the judge has
to stay a genuinely outside observer.

## Ordering baseline

The commit below is the fixed point every later benchmark result is
compared against — it is the last commit that touched anything in this
folder before any run had happened. At the moment this commit landed,
`bench/results/` did not exist anywhere in this project's history (verified
with `git log --oneline -- bench/results`, which returned nothing) and
`bench/acceptance/` was not excluded from version control (verified with
`git check-ignore -v bench/acceptance`, which also returned nothing).

- **Commit:** `763572bad64fea86245eeec448f88ca0c0335807`
- **Committer date:** 2026-08-17T22:34:32+02:00
- **Verified by:** running `bench/acceptance/verify-predates-runs.sh`
  immediately after this commit, which reported the vacuous pass — no
  results existed yet — and named 5 acceptance scripts checked (the four
  task judges plus this ordering checker itself).
