# Phase 209 Plan 05: Timed Against Plain Claude

Four real chat sessions, run on 2026-09-24, on this machine (`Callums-MacBook-Pro.local`,
Darwin 25.6.0 / Apple Silicon), using `claude` CLI `2.1.281 (Claude Code)` against the `aether`
runtime built from this repository's own commit `9ba078c5b4a9d4273521d31586c705152a179b01`
(reported version `1.0.88`). All four sessions ran under the owner's existing Claude Code
subscription — no `ANTHROPIC_API_KEY` or `ANTHROPIC_AUTH_TOKEN` is set on this machine, so
nothing here drew on a separate, metered API bill; the dollar figures below are Claude Code's
own recorded session cost against the subscription's included usage, not a new charge. This
satisfies the owner's condition for running this measurement ("as long as it's not real
money"), agreed 2026-09-24.

**Owner's ruling, 2026-09-24:** at 209-05's checkpoint the owner chose **measure-now** — run
all four real sessions (the small job and the medium job, each once through `/ant-go` and once
through plain Claude) rather than a smaller or skipped measurement. That ruling is the
authority for this report existing at all.

This is a measurement, run once, not a statistic. Four sessions on one afternoon on one
machine tell you what happened this time; they do not average out noise the way a hundred runs
would. Read the numbers below as one real data point each, not as a guaranteed outcome for
every future run.

## The two jobs, verbatim, identical on both sides

**Small job** (asked exactly this way on both sides):

> In src/util.go, add a doc comment directly above the noop function explaining that it
> intentionally does nothing.

**Medium job** (asked exactly this way on both sides):

> Add a small greeting package: create a new file src/greeting/greeting.go with a function
> Greet(name string) string that returns the text Hello, a comma, a space, the given name, and
> an exclamation mark all joined together, and create src/greeting/greeting_test.go with a
> passing test that calls Greet and checks the returned string.

Each side was also told, once, not to ask any questions and to just make the change and say
when it was done — the plain-Claude side needed that sentence spelled out; the Aether side got
the same courtesy sentence, folded into the one line that runs `/ant-go "<job>"`.

## How each run was built and measured

Each of the four runs got its own completely fresh, disposable practice project, built by this
repository's own `scripts/build-messy-practice-project.sh` — the same script this project's
own release-gate journey uses — so all four runs started from the identical messy, trap-laden
project state and nothing real was touched. Wall time and cost were read the same way the
messy-practice-project journey read them last time (`207-JOURNEY-RUN.md`): from the real,
persisted Claude Code session transcript's own final `cost-state` record
(`totalCostUSD`, `totalDuration`), not from a guess or from the chat's own prose.

**A safety correction made during setup, not a change to any test or gate:** the practice-project
script's own `aether install` / `aether update --force` steps sync Aether's shared slash
commands to this machine's real, global `~/.claude/` and `~/.codex/` folders by default — the
script's own comments say nothing here should touch those real folders, but for the global
command sync specifically, it does. Building each practice project under a throwaway, isolated
`HOME` for that one setup step (never for the actual timed chat sessions, which need the
owner's real login) kept every one of today's four runs from writing anything to this
machine's real, shared Claude/Codex command folders. Confirmed both before and after: this
machine's real `~/.claude/commands/` still has no `ant-go.md` in it (that command has not been
published yet), and `git status` on this repository was unchanged throughout. This is a real,
newly-found gap in the practice-project fixture's own isolation promise, filed to
`.planning/WINDOWS.md` as its own entry (see "New finding" below) — it is not this plan's job
to fix the fixture, only to avoid tripping over it while measuring.

## The four runs

| # | Job | Route | Wall clock | Cost | Landed? |
|---|-----|-------|-----------:|-----:|---------|
| 1 | Small | Plain Claude | 17.0s | $0.40 | Yes |
| 2 | Small | `/ant-go` | 36.4s | $0.44 | Yes |
| 3 | Medium | Plain Claude | 23.5s | $0.41 | Yes |
| 4 | Medium | `/ant-go` | 833.4s (~13.9 min) | $6.44 | Yes, but only after one extra owner reply (see below) |

Row 4 is not one clean pass. The first `/ant-go` call ran for 45s and cost $0.52, then stopped
and asked the owner how to proceed instead of finishing on its own — it printed Aether's own
"planning route" screen, said real dispatch needs a live helper-sender, and asked whether to
"just build it" or "use Aether." Only after one plain-English reply telling it to dispatch the
planning helpers itself did the session actually build the real specification, run a real
planning pass, build the real greeting package, and verify it — landing correctly (a real
`go.mod`, a correct `Greet` function, and a passing test, all confirmed by hand afterward). The
833.4s and $6.44 above are the whole session's own final total, both calls together, because
that reply is itself part of what a real owner would have had to type before the job was
actually done.

## Verdict

**On the medium job, `/ant-go` did not reach built work on its own — it stopped and asked the
owner a question partway through, contradicting this phase's own "zero further owner-typed
steps" promise, and once nudged through, it took about 35 times longer and cost about
sixteen times more than asking plain Claude the identical sentence directly (833.4s and $6.44
against 23.5s and $0.41).** On the small job the gap was much smaller and in the same
direction: `/ant-go` took roughly twice as long as plain Claude (36.4s against 17.0s) and cost
about a dime more ($0.44 against $0.40), for a one-line comment that plain Claude also got
exactly right. Neither job was faster or cheaper through Aether than through plain Claude on
this machine, this afternoon.

## New finding, filed rather than fixed here

The medium-job stall above is a real, freshly observed gap between what shipped in this phase's
own plans 209-02/209-04 (a route that reaches built work with no further owner-typed steps) and
what a real, unattended chat session actually does when it reaches the planning hand-off: the
automated test that stands in for this (`TestGoalReachesBuiltWorkWithNoExtraSteps`, cited in
`209-04-SUMMARY.md`) drives the Go runtime directly with the planning stage faked already-accepted,
so it never exercises the real wrapper text asking a live assistant to dispatch real planning
helpers the way `/ant-plan`'s own, much longer wrapper instructs. This measurement is the first
thing to run that real hand-off for real, and it found the gap. Recorded honestly rather than
patched: fixing the `/ant-go` wrapper's planning hand-off is a wrapper-text change, not a
timing-report change, and is out of this plan's own scope.

## What this does not do

No make target, test, gate entry, or timing harness was added anywhere in this repository.
`git diff --name-only` for this plan's own work touches only this file — no `Makefile`, no
`cmd/testdata/eval-gates/gates.json`, no `_test.go` file. This is a report, run once by hand,
for the owner to read.

---

## Run 5 — the live re-check after the fix (2026-09-24, owner authorised)

After plan 209-06 fixed the planning hand-off, the owner authorised one more real session to
see whether the stall recurred. Same job sentence as run 4, same wording telling it not to ask
questions, same `--permission-mode bypassPermissions` with no tool allowlist, in a fresh
practice project built by the same script. Session `92b11308-8302-4062-8d4d-59b7f1382d5e`.

| | Run 4 (before the fix) | Run 5 (after the fix) |
|---|---|---|
| Wall clock | 833.4s | **76.6s** |
| Cost | $6.44 | **$0.45** |
| Turns | 2 calls + 1 owner reply | 3, no owner reply |
| Stopped to ask the owner? | **Yes** | **No** |
| Work landed? | Yes, after a nudge | Yes |

The delivered code was checked by hand afterwards, not taken from the session's own account:
`src/greeting/greeting.go` and `src/greeting/greeting_test.go` both exist, `Greet("Ada")`
returns `"Hello, Ada!"`, and the test passes.

### What this run does NOT prove

**The fixed planning hand-off was never actually reached, so it is still unproven live.**

What happened instead: the small route ran first and the helper did the whole job. The
escalation then fired — not because the job was genuinely bigger, but because the practice
project has no resolvable way to run its own checks, so the quick attempt came back
`not_checked`. The screen said "this turned out bigger than it looked" and pointed at
`/ant-plan`. The assistant, holding work that was already finished and passing, judged that
planning it would add steps for nothing and skipped it, saying so plainly.

That is defensible behaviour and it is not the run-4 failure. But it means the hand-off text
plan 209-06 rewrote was never exercised. Defect register entry 60 therefore stays **open**.

### New finding: the escalation fires on "could not check", not on "bigger than it looked"

`status: not_checked` is being read as a failure signal for D-03's escalation. "The checks
could not be run" and "the checks failed" are different facts, and only the second is evidence
that a job was bigger than it looked. A project with no check command configured — which is
the ordinary state of a new project, and exactly what the owner will meet in the two-week
trial — would escalate every quick job this way, and then show a "this turned out bigger"
screen about work that is already complete. Filed separately.
