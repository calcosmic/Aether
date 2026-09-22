# The Real Run: Three Trials, Five Reverts, Measured

This is a factual record of two real runs made on this machine, with real `claude -p` chats
and real money. Nothing below is projected or estimated -- every figure is read from a log,
a session transcript, or a process timestamp captured during the run itself.

## Run identity

- **Date:** 2026-09-22
- **`claude` CLI version:** `2.1.278 (Claude Code)`
- **`aether` binary version:** `1.0.88`
- **Repository commit the run was made at:** `d721f4983385a80e5e24abb6e336999545e5820a`
- **Working tree:** the owner's own checkout for the three-trial run (`make eval-gate-journey`
  drives a real chat against a freshly built practice project each trial, but never mutates
  this checkout itself); five separate, disposable `git worktree` copies for the five-revert
  proof, each removed immediately after its own entry finished. `git status --short` on the
  owner's checkout was unchanged by either run (only the pre-existing, unrelated edits to
  `CLAUDE.md` and `.aether/CONTEXT.md` from another session remained, exactly as before).

## Caps in force per step

Every step's own `claude -p` call carries three caps: a turn limit, a wall-clock limit, and a
dollar limit. These are the numbers 207-04 measured and sized with a safety margin; this run
used them unchanged.

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

A separate warm-up call establishes each trial's session before the first real step; it carries
its own cap (3 turns, $2.00) to pay the one-time cost of building this repository's own large
prompt cache from cold.

---

## The three-trial run (`make eval-gate-journey`)

**Command:** `go test -tags=journey -run=TestJourney -count=1 -timeout=5400s ./...` (via
`make eval-gate-journey`), no `AETHER_JOURNEY_STEP` or `AETHER_JOURNEY_TRIALS` override, so the
harness ran its real default: three trials, whole chain.

**Result: FAILED.** `discovered=12 executed=12` -- the Go suite itself ran everything it was
supposed to; the failure is `TestJourney` itself, not a truncated run. `journeyGateVerdict`
correctly **refused** the resulting report ("trial 0 declares 14 steps but only executed 2
(stopped at 'survey') -- a trial that stopped part way must not read as clean"). This is the
expected outcome described in 207-06-PLAN.md's own assumptions, and it is the correct, honest
result of this run: WINDOWS.md entry 53, found live during 207-04's own reduced run, is a real
dead end this journey catches every time it is driven for real.

### Per trial

| Trial | Session id | Steps reached | Outcome | Wall clock | Cost (measured from the session's own `cost-state` total) |
|---|---|---|---|---|---|
| 0 | `99ae68e4-244d-4caa-9f4d-89a2fc232a12` | start (pass, 12.24s), survey (fail, 332.71s) | stopped at survey -- real failure | 344.95s | $3.00 |
| 1 | `01e17a2e-67da-419b-95b7-bac539b125a8` | start (pass, 11.56s), survey (fail, 315.12s) | stopped at survey -- real failure | 326.68s | $3.09 |
| 2 | `8cd98dc1-220f-44b9-acc0-5924d8a60892` | start (pass, 11.74s), survey (fail, 319.09s) | stopped at survey -- real failure | 330.83s | $2.91 |

Cost is read from each session's own final `cost-state` record in the real, persisted Claude
Code transcript (`~/.claude/projects/<encoded-cwd>/<session-id>.jsonl`) -- the same on-disk
transcript 207-01 established as the source of truth, not the chat's own prose. Precise
figures: trial 0 `$2.9997030000000007`, trial 1 `$3.0868871999999987`, trial 2 `$2.9099253`.

**Every trial failed at the same step, for the same reason.** In every trial, `/ant-colonize`
ran to real completion -- four real surveyor subagents dispatched, seven survey documents
written -- and the step's own on-disk fact check then found the territory snapshot's
`source_revision` had not moved to the practice project's current HEAD (the out-of-date-code-map
trap's snapshot was never refreshed):

- trial 0: snapshot pinned at `bd1f0118e65d6640fd6cb0941660247de4d9f2ec`, HEAD was
  `5e0d29bbe93734ac6e9d45853fd304965a382273`
- trial 1: snapshot pinned at `27961ee49a9f35acb5d0b47e7ab34f5355f4a0d1`, HEAD was
  `cf99b9cd5337e749250ced15d257fc94d28fc616`
- trial 2: snapshot pinned at `ef9c4e04b4797b15ceda5b2386fc42293581b143`, HEAD was
  `9ff58aa22e36a409d61ca1724a2641adff71590a`

**Classification: real, not flaky.** The same step failed the same way in all three trials,
each against its own freshly built practice project and its own freshly dispatched surveyor
subagents -- this is not noise from one bad run, it is a deterministic gap in when Aether
refreshes the territory snapshot. It is `WINDOWS.md` entry 53, opened live by 207-04's own
reduced run and reproduced here at full scale, three times over. **This belongs to Phase 208,
not this phase.** Per this plan's own explicit instruction, the check was not weakened and
Aether's colonize/territory-refresh logic was not touched to get past it.

**The sixth blocker's own check did not run in this pass.** The sixth 2026-09-21 problem's
check (207-03's `statusGuidanceCommandsWithoutMenuWrapper`, folded into the report by
`journeyEvaluateExpectedRed`) lives on the "status" step, which is step 9 of 14 -- every trial
stopped at step 2 ("survey") before reaching it. That check is still real, still built, and
still proven live and still-red on its own (207-03-SUMMARY.md), but this specific three-trial
run never reached far enough to re-evaluate it. Recorded honestly rather than implied.

### Measured totals, three-trial run

- **Total wall clock (the `TestJourney` subtest itself):** 1064.39s (~17.7 minutes) -- this
  includes the three trials' own step time (1002.46s) plus each trial's own fresh binary build
  and fresh practice-project construction (61.93s combined).
- **Total cost:** $8.9965155 (~$9.00), summed across all three trials' own final `cost-state`
  totals.

---

## The five-revert proof (`make prove-journey-fix-reverts`)

**Command:** `./scripts/prove-journey-catches-the-2026-09-21-fixes.sh`, over all five declared
entries in `cmd/testdata/journey/fix-reverts.json`, in their declared order. Each entry creates
its own throwaway `git worktree` from the owner's checkout's own HEAD, applies its exact
single-line revert, drives only the journey step that fix protects, and asserts the step fails.
Every worktree was removed immediately after its own entry finished (the script's own `trap
cleanup EXIT`); `git status --short` at the repository root was unchanged before and after.

**Result: PASS.** Every one of the five reverted fixes made the journey fail at exactly the
step it protects -- five for five, none missed, none needing a retry.

### The five rows, in the table's declared order

| # | The everyday problem (owner's own words) | Journey step it protects | Caught? |
|---|---|---|---|
| 1 | A folder missing from a write allowlist -- the program refused to write somewhere it legitimately needed to | survey | **Yes** -- the journey's "survey" step failed as expected: the territory refresh's write to `.aether/data/territory-candidates/` was blocked by the runtime's own protected-path hook, because the reverted allowlist entry no longer matched the real path. |
| 2 | Two archive names differing only by letter case -- filing away a finished project could silently overwrite another | archive | **Yes** -- the journey's "archive" step failed as expected: `aether entomb` collided `.aether/HANDOFF.md` and `.aether/data/handoff.md` into the same archive name because the comparison was no longer case-insensitive. |
| 3 | A fingerprint no helper can compute -- a research helper was asked to produce a cryptographic hash it has no way to genuinely calculate, so it could never report new findings | plan-second | **Yes** -- the journey's second "/ant-plan" pass failed as expected: it refused the Scout's new evidence as not content-addressed, because the reverted code trusted it (and hash-validated it) as an already-hashed existing record instead of deriving the hash fresh from disk. |
| 4 | A planning run pinned to a superseded specification -- correcting the project's own description mid-planning didn't stick, so planning kept working from the old, wrong version | plan-second | **Yes** -- the journey's second "/ant-plan" pass failed as expected: it resumed the parked run against the specification the owner had already corrected and re-approved, instead of refusing it as history. |
| 5 | A pause fingerprint that followed folder shortcuts -- saving your place broke on an ordinary shortcut (symlink) to a folder | pause | **Yes** -- the journey's "pause" step failed as expected: `aether pause` failed with "is a directory" on an uncommitted shortcut to a folder, because masking off the symlink/directory bits forced every path through the raw file-read the fix had removed. |

### The sixth case

The sixth 2026-09-21 problem -- a status screen advising the owner to run `aether
midden-review`, a command with no menu wrapper (no `/ant-...` command exists for it) -- has
**no fix yet**. Its own check is built (207-03) and stays honestly red on purpose; it is
Phase 208's job to close it, per the owner's own ruling (D-01, `207-CONTEXT.md`): *"the journey
proves it catches the five fixed blockers... The check for the sixth is built too and stays
honestly red until Phase 208 lands the fix... UED-09 is satisfied for the five; the sixth check
is a standing, named, expected-red case -- never a silent pass and never a claim of six."*

**Fixes proven caught this run: five. Never six.**

### Per-entry cost and wall clock

| Entry | Session id | Wall clock (this entry's own driven step + worktree/build overhead) | Cost (this entry's own session total) |
|---|---|---|---|
| write-allowlist | `eb51300e-28d1-41b2-bd11-917c2170a0ce` | 448s (includes the full survey-step subagent dispatch, 429.8s of it inside the chat itself) | $3.48 |
| archive-name-case | `8bcda3c9-94cb-4f1c-9928-3fd09843a794` | 73s (28.25s inside the chat) | $0.60 |
| helper-fingerprint | `fa8713ea-3d11-459c-8bc4-b016290e6f26` | 96s (53.78s inside the chat) | $0.87 |
| superseded-specification | `8b1724cc-e023-496c-a068-00e208e30a15` | 90s (48.94s inside the chat) | $0.85 |
| pause-follows-shortcuts | `0437a07f-0952-45e6-a61b-99d0609630bb` | 72s (29.85s inside the chat) | $0.62 |

Precise cost figures: `$3.4766199999999996`, `$0.6046055`, `$0.8698805000000001`, `$0.845083`,
`$0.6210389999999999`.

### Measured totals, five-revert run

- **Total wall clock (the whole script, all five entries, including five separate `git
  worktree` creations, five `go build` sanity gates, and five removals):** 780s (13 minutes),
  measured from process start to the script's own summary line.
- **Total cost:** $6.417228 (~$6.42), summed across all five entries' own session totals.

---

## Combined totals for this plan's own run

- **Wall clock, both runs together:** 1064.39s + 780s = 1844.39s (~30.7 minutes).
- **Cost, both runs together:** $8.9965155 + $6.417228 = $15.4137435 (~$15.41).

## What the gate's budget was set from

`cmd/testdata/eval-gates/gates.json`'s `journey` gate `budget_seconds` was set from this run's
own measured wall clock, using the formula this plan specified: the slowest of the three trials
(trial 0, 344.95s) multiplied by three (1034.85s), plus this run's own measured headroom for
building a fresh binary and a fresh practice project per trial (61.93s, the difference between
the whole `TestJourney` subtest's 1064.39s and the three trials' own step time of 1002.46s) =
1096.78s, rounded up to **1100 seconds** for a clean number. This reflects what was actually
measured in this run -- three trials that each reached only the "survey" step -- not a
projection of what a completed fourteen-step trial would cost. If Phase 208 closes the
territory-refresh gap and a future run completes the full chain, this budget will need
re-measuring from that run, honestly, the same way this one was.

---

## Owner's answer (Task 3)

*(recorded below once the owner has answered; see the checkpoint in 207-06-PLAN.md Task 3)*
