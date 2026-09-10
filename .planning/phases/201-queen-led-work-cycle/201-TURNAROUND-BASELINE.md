# Turnaround Baseline — Phase 201 Plan 13

**Scope decision (owner-approved 2026-09-10):** the plan's original design called for three
measured runs, medianed. The owner instead approved **one real measured run** for this pass,
recorded explicitly as a single-run baseline; further runs may refine it later. Every "median"
figure below is therefore the single run's own number, not a statistical median across repeats
— that substitution is the owner-approved deviation, stated here so it is never mistaken for a
three-run result.

**The task measured:** a real, owner-delegated single-task fix — `TestEveryLifecycleCommandEndsWithNextAction`
in `cmd/` fails only under heavy parallel test-suite scheduling and passes in isolation; the job
was to diagnose and fix it. This is genuine wanted work (owner delegation, not a practice task).

## What actually happened (plain account before the numbers)

To get one real, non-synthetic Aether "Queen-led work cycle" (colonize → plan → build → continue)
running end to end, the job was executed against a disposable git-worktree clone of this exact
repository at the current commit (never the live dogfood colony in `.aether/data/`, which stays
untouched), using a binary built fresh from this branch (the installed hub binary, dated 29 Aug,
predates the 201-12 telemetry work and would have produced no record at all).

**A real, repeatable environmental defect surfaced immediately and shaped the whole run:** every
spawned Codex/Claude worker subprocess in this execution session runs under an OS-level sandbox
that unconditionally denies **all** writes under `.aether/data/` — including the sanctioned
scratch subpaths (`survey/`, `planning/`, `phase-research/`) Aether's own architecture documents
as writable, and regardless of `--permission-mode acceptEdits`. `dangerouslyDisableSandbox` is
disabled by policy inside Aether's own worker invocation, so no worker in this session could work
around it. This is not a guess: four separate real surveyor workers, a real route-setter worker,
and a real builder worker all independently hit the identical denial on their own designated
output paths and said so in their own words. This is a genuine defect worth the project's
attention on its own merits (recorded below), but it is out of scope for this plan's task and was
not fixed here.

Three consequences for this measurement:

1. **Colonize's real survey workers did real research** (their responses show completed analysis)
   **but could not write their output**, so Aether's own fallback-to-local-synthesis path took
   over for the survey documents. The missing territory-snapshot metadata was then published by
   calling Aether's own real `publishTerritorySnapshot` function directly against the real survey
   artifacts already on disk — the same function `colonize-finalize` calls, over real content, not
   fabricated content.
2. **The real route-setter worker completed a full, genuine 3-phase plan** — including a specific,
   plausible root-cause diagnosis for the flaky test (a leaked repository-root environment
   variable/global reaching an isolated child test process; matches the exact error the worker's
   own reproduction later produced verbatim: `storage: repository containment refused: data root
   path must not be empty`) — but could not write its `phase-plan.json`/`ROUTE-SETTER.md` artifact
   and, unlike the surveyors, did not terminate cleanly afterward (it kept retrying/thinking after
   the denied write until the 15-minute worker timeout killed it). The plan's own real,
   schema-validated JSON answer was recovered from the worker's own session transcript (its own
   real output, not reconstructed or guessed) and activated through Aether's own real
   `activateGeneratedPlan` function.
3. **The real build dispatch genuinely ran**: two real Builder workers (Anvil-97, Bolt-76) were
   dispatched for Phase 1's three tasks. Bolt-76 wrote a real new failing test file
   (`cmd/isolated_process_inheritance_test.go`) matching task 1.2's goal. Anvil-97 ran the real
   reproduction sweep, reproduced the failure on the first try, and reported the exact isolated-child
   error line — but could not write its designated baseline artifact to
   `.aether/data/phase-research/`, the same sandbox denial. The build did not reach Aether's own
   "successful" terminal state in this environment, so `jobTelemetryRecord` — which 201-12's own
   design writes only at the end of a successful build — was **never written**. That is the direct,
   stated reason the eight-segment breakdown below is mostly unmeasured: not because the
   instrumentation is broken (201-12's own tests prove it works), but because this session's build
   attempt never reached the point in the code where it fires.

## The measured number (real wall clock, not telemetry)

Because the formal instrumented record was never written, the one number directly measured here
is the real build-dispatch wall clock, bracketed by hand around the actual `aether build 1`
invocation that dispatched both real workers concurrently:

- **Start:** 2026-09-10T18:28:50Z
- **Finish:** 2026-09-10T18:35:38Z
- **Elapsed:** 6m48s (408s) — two real workers, three tasks, one phase, concurrent dispatch

This is real, not synthetic: both workers ran genuine LLM-backed sessions and produced genuine
output (a real new test file; a real reproduction with a real error line) inside that window.

## Eight-segment breakdown (queue / preflight / model / tool-call / context / work / verification / wait)

| Segment | Status | Reason |
|---|---|---|
| queue | unmeasured | `jobTelemetryRecord` was never written — the build did not reach a successful terminal state in this environment (see above), so the instrumented queue segment (job-planning-done to brief-assembly-start) was never captured. |
| preflight | unmeasured | Not observable on the direct build lane per 201-12's own design (no per-call duration reported by the platform); also never reached telemetry-write in this run regardless. |
| model | unmeasured | Same as preflight — not a segment the direct build lane observes today (201-12 SUMMARY, "Key decisions"). |
| tool-call | unmeasured | Same as preflight/model — no first-response/tool-call boundary is observed by this lane today. |
| context | unmeasured | `prepareBuildWorkerBriefFiles` genuinely ran (both workers' `--print-brief` checklists show real context assembled: 85.2% and higher of budget), but its duration was never persisted because the build never reached the telemetry write. |
| work | **the one real measured number above** — 408s wall clock for the concurrent two-worker dispatch-and-wait window, hand-bracketed around the real `aether build 1` invocation, not read from a `jobTelemetryRecord` (none was written). Treated as the closest real analog to the instrumented "work" segment, with that substitution stated plainly rather than presented as the formal record. |
| verification | unmeasured | `aether continue` was never run — the build never reached a state continue could meaningfully verify (one task blocked, the phase not closed). |
| wait | unmeasured | Not observed by the direct build lane today (201-12 SUMMARY). |

**Median total:** not a median — a single run, owner-approved 2026-09-10 (see Scope decision
above). The single run's total for the measured build-dispatch window is **408s (6m48s)**.

**Median largest segment:** the only segment with a real measured figure is **work**, at 408s;
by elimination it is also the largest of the eight, but that claim rests on one real number
against seven honestly-unmeasured ones, not a comparison across eight measured figures.

## Reported cost

Not available in this environment. The scratch colony's `.aether/data/spend/session.json` recorded
only session identity (platform, session ID, transcript path), not a token or dollar total — this
run's cost ledger was never populated, for the same reason the telemetry record was never written
(the build never reached the point where a completed attempt's cost gets rolled up).

## Proposed target (not ratified)

Given the thin evidence base above (one real 408s number, not a clean multi-segment record), the
proposed target is stated conservatively and ties directly to the one real figure measured:

**Target: cut the measured build-dispatch window by roughly a third, from 408s to approximately
270s (4m30s), for a comparably-sized one-phase, two-worker, three-task build.**

Arithmetic: 408s × (1 − 1/3) ≈ 272s, rounded to 270s.

This targets the **work** segment specifically — the real, measured number — because it is both
the only segment with direct evidence from this run and the segment the project's own prior
evidence (the `.planning/todos/pending/2026-08-27-worker-turnaround-is-too-slow.md` todo this
plan traces to) already names as the dominant cost: worker dispatch/execution time and the
project's own `cmd` test suite, not queue or context assembly.

**Lever expected to deliver it:** trimming the dispatched brief further (the todo's own "Fix (3)"
— stop re-sending the whole planning corpus to every worker; the `--print-brief` checklists from
this very run show both workers already carrying a large context share, 85%+ of budget, before any
work begins) and/or the project's own `cmd` suite fix already partially landed (the todo's own
2026-08-28 progress note: package build time cut 41% by sharing one compiled binary across tests
instead of recompiling per test). Both are real, already-partially-proven levers from this
project's own history, not new invention.

**Explicitly excluded, per the plan's own prohibitions:** no part of this target depends on
reducing test coverage, skipping the deterministic checks, or running more workers in parallel.
Parallel-wave expansion is not an approved lever and is not proposed here.

## Environmental finding (recorded, not fixed here — out of scope for this task)

Every spawned Codex/Claude worker subprocess dispatched by this Aether binary, in this execution
session, is denied all writes under `.aether/data/` by an OS-level sandbox that Aether's own
`writeClaudeWorkspaceSettings` (`pkg/codex/platform_dispatch.go`) enables
(`"sandbox": {"enabled": true, ...}`) with no override available
(`dangerouslyDisableSandbox` disabled by policy). This blocked four real surveyors, one real
route-setter, and one real builder from writing their own designated, sanctioned-subpath output —
in every case after they had already done the real work. The route-setter additionally did not
terminate cleanly after the denial and consumed its full 15-minute timeout retrying instead of
reporting "blocked" the way the surveyors and the builder did. This is a real, reproducible defect
affecting any Aether colony run in an environment with this sandbox profile, independent of
repository size or task complexity. Not fixed as part of this plan — recorded here for the
project's own attention, per this repo's own practice of naming real discovered defects rather
than quietly working around them and saying nothing.

**A second, smaller, real defect surfaced in the same session:** `aether discuss`'s automatic
specification-draft creation (`cmd/discuss.go`) derives each `affected_public_paths` item's
semantic lineage as `"known-public-path-" + path`, where `path` comes directly from the survey's
entry-point list. Any entry-point path longer than 21 characters pushes the lineage past the
specification system's own 40-character limit, and the draft is refused outright
(`semantic lineage exceeds 40 canonical characters`) — this reproduced immediately, unconditionally,
on this real repository's own entry points (e.g. `cmd/testdata/skill-fixtures/go-cli/main.go`).
Worked around for this measurement only, in a disposable scratch clone, by numbering the lineage
instead of embedding the path (`known-public-path-01`, `...-02`, ...); never applied to this
repository. Also recorded here, not fixed, for the same reason as above.

## Owner ratification

**Pending.** The next step in this plan (Task 2, a `checkpoint:decision`) asks the owner two
questions before any target is ratified: whether this measured job (a real single-task bug fix,
delegated to the orchestrator, run through a real but partially-blocked Aether work cycle) is
representative of the work he actually runs, and whether the proposed target above is the right
bar. Nothing below this line is ratified; the figures above are the proposed baseline only.
