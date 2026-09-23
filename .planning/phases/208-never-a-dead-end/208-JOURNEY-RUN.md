# The Real Run: One Walk, Approved, Measured

This is a factual record of one real run made on this machine, with a real `claude -p` chat and
real money. Nothing below is projected or estimated — every figure is read from a log, a session
transcript, or a process timestamp captured during the run itself.

**In plain words first:** the walk took about a minute and a half end to end and cost under a
dollar (the chat's own recorded total was $0.93). It did **not** get all the way through — it
stopped at step 2 of 14 ("survey", the code-map step), the same step Phase 207's run always
stopped at, but this time for a different reason. Aether correctly told the assistant "there's
already a survey here, are you sure you want to replace it?" and named the exact command to run.
The assistant chose to ask the owner for permission instead of just running that command — which
would be the right, cautious thing to do in a normal conversation, but there was no owner
available to answer inside this automated test, so the walk stopped there. The three-walk
minimum this gate requires was deliberately not run (the owner approved one walk only), so this
is a measurement, not a passed release gate.

## Owner's decision this plan executed against

Checkpoint "How much of the full walkthrough to run for real" (208-08-PLAN.md, Task 1):
**one-then-decide** — run ONE full walk now (~£12–25 / 20–40 min approved), measure it, and do
not run the other two without asking again. Recorded verbatim per the executor's own brief.

**No second paid run was started.** The failure below is not clearly attributable to a bug this
phase's own commits introduced (see "Why it stopped" below); per the checkpoint's own
instruction, a second paid run is not something this plan starts on its own — it needs the
owner's agreement first.

## Run identity

- **Date:** 2026-09-23
- **`claude` CLI version:** `2.1.280 (Claude Code)`
- **`aether` binary version:** `1.0.88` (the version string checked in at the commit this run was
  made at; the journey harness builds its own temporary binary from the same source tree)
- **Repository commit the run was made at:** `d8f38b40a02f3ab26b2fd7d50cef4c9ac7950d9a`
  (`fix(208-08): size the journey gate provisionally for a full 14-step walk`)
- **Working tree before the run:** clean except the pre-existing, unrelated `.aether/CONTEXT.md`
  edit from another session (present before this plan started, untouched by it).
- **Working tree after the run:** unchanged by the run itself — `git status --short` shows the
  same pre-existing `.aether/CONTEXT.md` line and this plan's own subsequent edits to
  `.planning/WINDOWS.md`; nothing under `cmd/journey*.go` was touched during or after the run.

## Caps in force per step

Unchanged from 207-04's own measured, safety-margined figures (`cmd/journey_live_test.go`):

| Step | Max turns | Wall-clock cap | Budget cap |
|---|---|---|---|
| start | 4 | 120s | $0.50 |
| survey | 20 | 600s | $4.00 |
| discuss | 12 | 400s | $2.00 |
| specification | 15 | 400s | $2.00 |
| plan-first | 25 | 700s | $4.00 |
| plan-second | 25 | 700s | $4.00 |
| build | 40 | 1200s | $6.00 |
| check | 40 | 1200s | $6.00 |
| status | 4 | 120s | $0.50 |
| pause | 4 | 120s | $0.50 |
| resume | 4 | 120s | $0.50 |
| finish | 12 | 400s | $2.00 |
| archive | 12 | 400s | $2.00 |
| start-again | 4 | 120s | $0.50 |

---

## The single-trial run (`AETHER_JOURNEY_TRIALS=1 make eval-gate-journey`)

**Command:** `AETHER_JOURNEY_TRIALS=1 go test -tags=journey -run=TestJourney -count=1
-timeout=6662s ./...` (via `AETHER_JOURNEY_TRIALS=1 make eval-gate-journey`), started in the
background at `2026-09-23T08:53:35Z`, polled to completion at `2026-09-23T08:55:05Z` — **90
seconds of total wall clock** for the whole `make eval-gate-journey` invocation (npm install for
`aether host`'s node wrapper, building the test binary, running every `./...` package under the
`journey` tag, and the one live trial itself).

**Result: FAILED, as expected for a below-minimum trial count.** `discovered=17 executed=17` —
the Go suite ran everything it was supposed to; the failure is `TestJourney` itself.
`journeyGateVerdict` correctly refused the report: *"journey report carries 1 trial(s), want at
least 3 — a reduced trial count can never satisfy the release gate."* This is the expected,
honest outcome of running below the gate's own three-trial minimum on purpose, per the owner's
decision above — **this run is a measurement, not a passed gate.**

### The one trial

| Trial | Session id | Steps reached | Outcome | `TestJourney` subtest wall clock | Cost (from the session's own `cost-state` total) |
|---|---|---|---|---|---|
| 0 | `06a5379a-8663-48dd-9d4f-68865e2d8879` | start (pass, 8.75s), survey (fail, 27.73s) | stopped at survey — real failure | 58.08s (`TestJourney`), of which start=8.75s + survey=27.73s=36.48s is subtest time | $0.9333466000000001 (~$0.93) |

Cost is read from the session's own final `cost-state` record in the real, persisted Claude Code
transcript (`~/.claude/projects/-private-var-folders-*-TestJourney*-repo/06a5379a-8663-48dd-9d4f-68865e2d8879.jsonl`,
line 92) — the same on-disk transcript `journeyFindSessionTranscript` reads, not the chat's own
prose. The transcript records three successive `cost-state` snapshots, one per driven step:
$0.3714948 after the session-establishing "ready" call, $0.8009504 after `/ant-init` (the "start"
step), $0.9333466 after `/ant-colonize` (the "survey" step, where the trial stopped).

### Why it stopped: the same dead end, a different proximate cause

**The "start" step passed correctly, by design.** The driven prompt for "start" is the bare
`/ant-init` command with no goal argument — deliberately, because `scripts/build-messy-practice-project.sh`
already ran a real `aether init` to seed this practice project before the chat ever starts. The
real `/ant-init` wrapper, seeing an already-active colony, correctly declined and named `/ant-plan`
as the next step, without running any Bash command — exactly the documented, correct behaviour
for this step (`cmd/journey_live_test.go`'s own comment on `bashCalls < 1`).

**The "survey" step (`/ant-colonize`) stopped at the same on-disk check Phase 207 never got past,
for a different reason.** The transcript shows:

1. The assistant ran `aether host colonize`, which exited 1 with: *"Fatal: Go command failed:
   colonize --plan-only: A territory survey already exists for this project. — next: aether
   colonize --force-resurvey."* This is the typed refusal `colonize-existing-survey-found`
   (`cmd/refusal_register.go`), `Disposition: "stop"`, `ProtectsWork: true` — a **pre-existing**
   dead-end check (its underlying bare-error text predates Phase 208 by git blame; 208-06 only
   wrapped it in the typed-refusal envelope and confirmed its `NextCommand`).
2. The assistant investigated — listed the existing survey files, found the practice project's
   deliberately-seeded placeholder content (`journey trap: out-of-date-code-map placeholder
   artifact`), checked `git log` (29 commits since the survey was written), and correctly
   concluded the map was stale.
3. **Rather than running the named `aether colonize --force-resurvey` itself, the assistant asked
   the owner:** *"My pick: redo the survey... To go ahead, reply 'resurvey'. To keep the current
   map and go straight to planning, type `/ant-plan`."*
4. The journey harness sends exactly one driven prompt per step (the bare menu command, per
   `journeyStepPrompt`) and never a follow-up answering a mid-step question — there is no owner
   present in an automated `-p` chain to reply "resurvey". The session ended there, and the
   step's own on-disk fact check failed:

   ```
   territory snapshot source_revision is a2bed0c8dc2d49f3741fd9d74292764b84b1de14,
   want current HEAD 2cf2c6f54cc3c69ce879586fedb521346cdf904e
   ```

**This is the same net dead end WINDOWS.md row 53 describes (the survey step never refreshes the
stale territory snapshot), but a genuinely different proximate cause than the one Phase 207 found
and this phase's 208-01/208-07 fixed.** Phase 207's three trials all *did* run `--force-resurvey`
autonomously, dispatched four real surveyor subagents, and then hit a *different* wall further
along (colonize-finalize refusing on a missing `generated_at`, silently reported as "completed
clean"). That specific defect's code fix shipped in this phase (208-01: recover `generated_at`
from Aether's own receipt; 208-07: run every printed refusal's `NextCommand` for real, after the
step's on-disk fact check) — but **this run never reached that code path at all**, because this
trial's own assistant chose the more cautious "ask first" interpretation at the earlier
existing-survey stop, rather than the "just run the named command" interpretation Phase 207's
trials happened to take. The refusal fired exactly as designed (named the one way past, protected
existing work) — the dead end here is that an automated, unattended `-p` chain cannot answer a
question a real owner could, and the assistant's own judgement call this run erred toward asking.

**Filed as a new, distinct finding — WINDOWS.md row 55** (not folded into row 53, since it is a
different proximate cause with a different owner-facing decision needed): whether a
`ProtectsWork: true` "stop" refusal's `NextCommand` should be auto-run when there is no realistic
owner to ask (an automated `-p` chain) versus always deferred to the owner in an interactive
session — a product decision this plan's own declared files (`Makefile`,
`cmd/testdata/eval-gates/gates.json`, `cmd/eval_gates_test.go`, this report, `WINDOWS.md`) do not
include the file (`.aether/commands/colonize.yaml`) that would need to change to resolve it.

### Not proven by this run

- **The survey step did not pass.** The must-have truth this plan's own frontmatter names ("the
  out-of-date code map is refreshed, closing the dead end the Phase 207 run found three times out
  of three") is **not met** by this run. The dead end is still real, live, and reproducible — for
  a narrower, now-precisely-identified reason.
- **The generated_at-recovery code path (208-01, 208-07) was never exercised live.** It is
  covered by unit tests (`cmd/criterion_binding_dead_end_test.go`, `cmd/refusal_printed_test.go`,
  `cmd/refusal_behaviour_test.go`) but this run's own transcript never reached
  `colonize-finalize` at all.
- **The three-trial minimum was not met.** By design (the owner's own decision) — this run
  measures one walk, not the release gate.
- **No steps past "survey" were exercised** — discuss, specification, plan-first, plan-second,
  build, check, status (including the sixth-blocker `midden-review` check), pause, resume,
  finish, archive, start-again are all unmeasured by this run.

---

## What the gate's budget was set from

`cmd/testdata/eval-gates/gates.json`'s `journey` gate `budget_seconds` and the Makefile
`eval-gate-journey` target's `-timeout=` were both set, **before** this run, to a **provisional**
value derived exactly as this plan's own action text specifies: the sum of all fourteen steps'
own per-step wall-clock caps (`journeyStepWallClockSecs`, `cmd/journey_live_test.go`:
120+600+400+400+700+700+1200+1200+120+120+120+400+400+120 = 6600s) for one trial (the
owner-approved single walk), plus 207-06's own measured per-trial build/setup headroom (61.93s),
rounded up = **6662 seconds**. Committed as `d8f38b40` before the run started.

**No final re-measurement was made after the run, and none is claimed.** This plan's action text
also calls for a *final* re-measurement — "the slowest trial's own wall clock times the trial
count, plus this run's own measured build and setup headroom, rounded up" — but that formula
needs a completed trial's real wall clock for the full fourteen-step chain, and this run's one
trial stopped at step 2 of 14. Inventing a number for the eleven steps that never ran (survey's
own subagent dispatch, and especially build/check at up to 1200s wall-clock cap each) would be
exactly the projection this plan and CLAUDE.md's Definition of Done forbid. **The provisional
6662s value stays in force**, honestly labelled provisional in both `Makefile` and `gates.json`,
until a run completes the full chain and a real final figure can be measured from it.

---

## WINDOWS.md rows 49–53, resolved from this phase's own evidence

| Row | Resolution | Evidence |
|---|---|---|
| 49 | **Fixed** | `e70620fa` (208-05): `validatePhaseCriterionEvidenceAgainstDisk` refuses a directory (or symlink to one) bound as a criterion artifact at build time, by name, before any worker is dispatched. Proven by `cmd/criterion_binding_dead_end_test.go`. |
| 50 | **Fixed** (already resolved by 208-02, 2026-09-22T16:47:34Z) | Per-task `Verified` now comes from that task's own bound criteria (`taskVerifiedFromCriteria`); one failing criterion no longer marks every task `implemented_unverified`. |
| 51 | **Fixed** (already resolved by 208-02, 2026-09-22T16:47:36Z) | A `cd <dir> &&`-prefixed verification-commands line is now parsed, classified, and run in that directory (`splitVerificationCommandDirectoryPrefix`, `runVerificationStepInDir`). |
| 52 | **Fixed** | `213eddbc` + `c2dd8955` (208-05): `criterionBindingIsUnsatisfiable` routes an already-bound directory criterion to the existing owner-confirmation exit (`aether decision-answer`); `aether skip-phase --force` now names that route first. |
| 53 | **Left open — partially addressed, not proven live.** | Both named code fixes shipped this phase (`cebe676d` 208-01: recover `generated_at` from Aether's own colonize receipt; `9c8744be`+`d09906e9` 208-07: run every printed refusal's `NextCommand` for real). This run's own single trial never reached `colonize-finalize` — it stopped one step earlier, at the pre-existing `colonize-existing-survey-found` refusal, where the assistant chose to ask the owner rather than run the named `--force-resurvey` itself (see "Why it stopped" above, and the new row 55). The row's own proposed close names a live behaviour (the survey step passing); that has not happened, so it stays open. |

**New finding recorded as row 55** (`.planning/WINDOWS.md`): the `ProtectsWork: true` stop
refusal's ask-vs-act ambiguity described above, distinct from row 53's original cause.

---

## Combined totals for this plan's own run

- **Wall clock:** 90s (the whole `make eval-gate-journey` invocation, `AETHER_JOURNEY_TRIALS=1`).
- **Cost:** $0.9333466000000001 (~$0.93), read from the one session's final `cost-state` total.

Both figures are far under the owner's approved range for one walk (~£12–25, 20–40 minutes) —
the run was cheap and fast precisely because it stopped after two of fourteen steps, not because
the full chain is actually this inexpensive.
