# Phase 172 — Stop Rule (agreed 2026-08-12, round 4)

## Why this file exists

Phase 172 has run three build-verify rounds. Each round closed real defects and each
round's review found new ones. The cause is not the work: **two of the four success
criteria are written as absolutes** — *the release gate cannot be silently switched
off*, *the allowlist may only shrink*. An absolute over an adversarial surface
(GitHub Actions workflow semantics) has no defined edge, so review can always find
one more layer. Left unbounded, this phase does not terminate.

Round 4 is therefore the **last build round** under the criteria as currently written.

## The rule

Round 4 closes the five defects listed below and nothing else.

After round 4's code review and verification run:

- **If the only findings are the five known defects, unclosed or partially closed** —
  fix and re-verify is permitted within round 4. This is ordinary iteration.
- **If the review finds a NEW class of bypass** (any defect not in the list below) —
  **STOP. Do not plan a round 5.** Instead:
  1. Rewrite ROADMAP success criteria 1 and 4 to state exactly what is proven, with
     the unproven residue named explicitly rather than implied.
  2. Mark Phase 172 complete against the narrowed criteria.
  3. Open a tracked follow-up phase carrying the residue, so later phases that depend
     on this ratchet inherit a documented limit rather than a false guarantee.

A narrower true claim beats a broader unproven one. See CLAUDE.md's Definition of
Done: this repo has 18 of 25 milestones framed around restoring something previously
marked done, which is what unbounded "it's wired now" claims produce.

## The five defects round 4 may close

All five were independently reproduced by construction on 2026-08-12 — by mutating
the live workflow or testdata and observing the guard suite stay green. None is
inferred from reading code.

| ID | Defect | Scope |
|----|--------|-------|
| CR-01 | `jobs.<id>.continue-on-error: true` disables the release gate. `stepCanFailTheBuild` scans only inside a step's indentation block, so a job-level key is invisible to it. | job-level |
| CR-02 | `jobs.<id>.env.GOFLAGS: -run=<nothing>` makes the gate run zero tests and exit 0. The run line stays byte-identical to the pin, nothing reads `env:`, and the execution harness inherits the local environment rather than the workflow's — so neither half of the proof sees it. | job-level |
| CR-03 | `paths-ignore: ['**']` under the triggers stops the workflow firing. The `on:` check is a substring presence test for `pull_request:`/`push:`, both of which remain present. | trigger-level |
| CR-04 | `TestPathMigrationDidNotWidenTolerance` compares **bare leaf names**, and 14 tolerated leaves are generic single words (`get`, `set`, `setup`, `export`, …). A brand-new unwired command added to both the live list and the baseline passes every guard, including `TestNoRegisteredSubcommandIsUnreferenced`. Confirmed with a real registered command. | allowlist |
| CR-05 | `cmd/testdata/orphan_allowlist_baseline_pre_path_migration.json` — the frozen anchor every shrink-only claim rests on — has no integrity pin. A hand-added entry passed all 30 named guard tests. | anchor file |

## Direction, not just a defect list

CR-01/02/03 must NOT be closed by adding three more string searches. That is the
blocklist shape this phase has already rejected twice, and it is why an unbounded
tail exists at all.

The generalising move is a **whitelist of workflow shape**: assert the workflow's and
job's keys are exactly a known, reviewed set, so any unrecognised key — including one
nobody has thought of — fails by default and must be deliberately added. A blocklist
must anticipate the attack; a whitelist does not. This is the same reasoning that made
round 3's execution harness work where two text checks failed.

CR-04 is a straightforward key-format fix (compare full command paths, not leaves).
CR-05 is a straightforward integrity pin (a checksum constant in Go source).

## What is already proven and must not regress

Round 4 must not weaken any of these. All were confirmed by live mutation:

- Appending `; true`, `|| exit 0`, `| cat`, or a second `-run` filter to the gate
  command → guard goes red.
- `if ! CMD; then true; fi` and `(CMD) ; echo done` — mutations in no plan's list →
  guard goes red. This is the evidence the execution harness checks a property rather
  than a blocklist.
- Commenting out the gate step, `continue-on-error` on the **step**, removing the
  `pull_request:` trigger → guard goes red.
- Removing `aether colonize` / `aether closeout` from the live allowlist → the ratchet
  fails and names both, by full path.
- Success criteria 2 and 3 are verified and closed.
