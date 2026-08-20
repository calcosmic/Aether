# Phase 190 — Deferred Items

Discoveries made during execution that are real, verified findings but out of scope for the
plan that found them. Logged per the executor's scope-boundary rule rather than silently fixed
or silently ignored.

## D-190-01-A: Pheromone signals and prior-worker handoffs render twice in the plan-only wrapper flow

**RESOLVED by 190-03.** `composeBuildManifestBrief` now takes an
`includeSteeringSections` flag; `attachBuildDispatchContext` (the wrapper
plan-only flow and the `--print-brief` inspector that simulates it) passes
`false`, deferring pheromone signals and prior-worker handoffs to the
manifest-level capsule exclusively. `writeBuildWorkerBriefFiles`'s own
fallback composition (the direct/native path, which carries no capsule in its
JSON envelope) passes `true` and keeps the old self-contained behavior. See
`190-03-SUMMARY.md` for the full ownership decision and proof. The rest of
this entry is kept for historical record.

**Found during:** 190-01, Task 2 (building `duplicatedBriefSections` and proving it against real
output).

**What was verified, by reading and by test:**

- `cmd/colony_prime_context.go:571` (`resolveCodexWorkerContext()`, the manifest-level capsule
  every plan-only response carries as `dispatch_manifest.context_capsule`) emits its own
  `## Pheromone Signals` section whenever a pheromone signal is active.
- `cmd/colony_prime_context.go:695-699` (the same capsule) also emits its own
  `## Previous Worker Handoffs` section from `renderWorkerHandoffSection("build", ...)`.
- `cmd/codex_build.go`'s `composeBuildManifestBrief` (the function that composes every
  per-dispatch `brief`/`brief_path` content) independently calls `resolvePheromoneSection()` and
  embeds `dispatch.HandoffSection` (itself `renderWorkerHandoffSection("build", phase.ID,
  dispatch.Name)`) into the SAME brief.
- Since the wrapper contract (`.claude/commands/ant/build.md`) prepends the capsule ONCE ahead of
  each dispatch's brief, a wrapper-spawned worker's assembled context contains
  `## Pheromone Signals` and (when handoffs exist) `## Previous Worker Handoffs` **twice** whenever
  either is active — once from the capsule, once from the brief.
- Proven empirically, not just by reading: `TestPrintBriefCommandFailsWhenPrintWorkerBriefsFindsDuplication`
  (`cmd/build_print_brief_test.go`) seeds one active pheromone signal against the ordinary
  `basePrintBriefState()` fixture and the new `duplicatedBriefSections` check (Task 2 of this plan)
  correctly flags `Pheromone Signals` as duplicated. This is a REAL, already-shipping duplicate the
  new detector catches — not a synthetic one.

**Why this is not fixed here:** Criterion 3 of ROADMAP Phase 190 named a specific, different double
injection (the TS-host `hive_section` channel, owned by plan 190-02) and 190-CONTEXT.md's research
pass did not identify this one. Fixing it would mean making `composeBuildManifestBrief` aware of
whether a capsule will ALSO be prepended (i.e. distinguishing the plan-only caller from the direct
dispatch path, which has NO capsule and therefore genuinely needs the brief-level pheromone/handoff
sections — this is exactly what `TestWorkerBriefFileHoldsComposedBrief`, a protected test, requires
to keep passing unmodified). That is a real design decision about which composition layer owns which
section for which caller — Rule 4 territory (architectural), not a mechanical bug fix, and squarely
outside this plan's `files_modified` list.

**Recommended follow-up:** A future phase should apply the same "one canonical channel" pattern
Phase 190's own criterion 3 already applies to hive wisdom: either (a) have
`writeBuildWorkerBriefFiles`/`composeBuildManifestBrief` accept a flag mirroring `clearInlineBrief`
that skips the pheromone/handoff sections when a capsule is about to be prepended (plan-only), or
(b) stop delivering pheromones/handoffs via the capsule for build specifically and let the brief stay
the single source, mirroring what D-07 already confirmed is true for `continue`. Either fix should be
proven with the exact `TestPrintBriefCommandFailsWhenPrintWorkerBriefsFindsDuplication`-style fixture
(real active pheromone signal, real assembled context, real duplication check) this phase's Task 2
already wired.

**Impact if left unfixed:** Every real `/ant-build` interactive session with an active FOCUS,
REDIRECT, or FEEDBACK signal (or a non-empty prior-worker handoff history) will now trip
`aether build <phase> --print-brief`'s new duplication check. This is not a false positive — it is
the tool correctly reporting a real duplicate — but it means `--print-brief` will not be "clean" on a
large share of real colonies until this is fixed. Flagging prominently for the phase that picks this
up.

---

## D-190-03-A: The native/direct build dispatch path independently double-delivers pheromone signals through a separate, unfixed channel

**RESOLVED by 190-05.** Audited all ~8 `codex.WorkerDispatch`/`codex.WorkerConfig` construction
sites named below (build native, continue review, continue watcher, plan, colonize, seal, swarm,
quick, oracle) individually: build/continue×2/plan/colonize/seal/swarm all set `ContextCapsule` to
`resolveCodexWorkerContext()`, which unconditionally embeds `## Pheromone Signals` whenever a
signal is active — those seven call sites now leave `PheromoneSection` unset entirely (the
`resolvePheromoneSection()` call was removed at each). quick and oracle build their own lightweight,
custom capsules (`renderQuickContextCapsule`, `renderOracleContextCapsule`) that do NOT render
pheromones at all — `PheromoneSection` remains their sole channel, deliberately unchanged.
`codexWorkerDispatchesForRecovery` (`cmd/codex_build_finalize.go`, build's retry-instruction
builder) also had the same redundant-but-inert pattern; fixed for consistency even though its
`WorkerDispatch` values are never passed through `AssemblePrompt` today. See `190-05-SUMMARY.md`
for the full per-caller audit table and the fail-then-pass proof (`TestNativeDispatchPheromoneStaysExactlyOnceViaCapsule`,
`TestEightCommandsDeliverPheromoneExactlyOnce`). The rest of this entry is kept for historical record.

**Found during:** 190-03, while tracing "both delivery paths" per this plan's own architectural-decision
constraint 2 (the trap: a fix must not zero out a section on a path it didn't intend to touch).

**What was verified, by reading and by test:** `executeCodexBuildDispatches` (`cmd/codex_build.go:1682`,
the function that actually spawns a native/hosted worker subprocess for `aether build <phase>`, no
`--plan-only`) computes `capsule := resolveCodexWorkerContext()` (line 1694) and
`pheromoneSection := resolvePheromoneSection()` (line 1697), then passes BOTH into
`codex.WorkerDispatch` as separate fields for every dispatch: `ContextCapsule: capsule` (line 1715) and
`PheromoneSection: pheromoneSection` (line 1720). `capsule` already renders its own
`## Pheromone Signals` section whenever a signal is active (`cmd/colony_prime_context.go:571`, inside
`buildColonyPrimeOutput`, unconditionally, independent of anything this plan touched).
`AssemblePrompt`/`AssembleHostedPrompt` (`pkg/codex/prompt.go:58-87`) then join `contextCapsule` and
`pheromoneSection` as two SEPARATE, both-included prompt parts (`parts := []promptPart{..., {name:
"context", content: contextCapsule}, ..., {name: "pheromone", content: pheromoneSection}, ...}`,
joined with `"\n\n"` between non-empty parts) — never deduplicated against each other.

Proven empirically, not just by reading: a throwaway probe (deleted after use) seeded one active
FOCUS signal and called `resolveCodexWorkerContext()` and `resolvePheromoneSection()` directly (the
exact two values `executeCodexBuildDispatches` computes) — both returned non-empty, and both contained
the identical signal text (`capsule` under `"## Pheromone Signals"`, `pheromoneSection` under its own
`"### Active Pheromone Signals"` heading). A native-dispatched worker's assembled prompt therefore
carries the same steering text twice, under two different headings.

**Why this is not fixed here:** Out of 190-03's scope on two counts. First, the plan's objective and
D-190-01-A both name "the plan-only wrapper flow" specifically — `composeBuildManifestBrief`'s brief
text (what this plan's fix touches) is never read by `executeCodexBuildDispatches` at all (it builds
`TaskBrief` from `renderCodexBuildWorkerBrief`, the raw, un-composed render); this is a structurally
independent duplication mechanism this plan's fix does not reach, for better or worse. Second, fixing
it means deciding whether `codex.WorkerDispatch.PheromoneSection`/`.HandoffSection` should exist at all
once `ContextCapsule` already carries this content for every one of that struct's many callers (build,
continue, plan, colonize, quick, oracle, seal, swarm all set these fields via the same
`renderWorkerHandoffSection`/`resolvePheromoneSection` pattern) — a genuine architectural question
(Rule 4 territory) about a widely-shared struct this plan's `files_modified` list does not cover, not a
mechanical fix.

Note: `HandoffSection` does NOT independently duplicate on this same path today — for BUILD's native
dispatch specifically, `dispatches[i].HandoffSection` is only ever set by `attachBuildDispatchContext`
(`cmd/codex_build.go:3253`), which is never called on the direct/native path (confirmed:
`writeCodexBuildArtifacts` never calls it), so the field stays empty and `AssemblePrompt`'s "handoff"
part is omitted there — only the capsule's own `## Previous Worker Handoffs` section delivers it, once.
The finding above is pheromone-specific.

**Recommended follow-up:** A future phase should decide, for `codex.WorkerDispatch`/`WorkerConfig`'s
`ContextCapsule` + `PheromoneSection` (+ `HandoffSection`, for whichever of its 8 call sites populates
both non-trivially) fields, the same kind of ownership question 190-03 just answered for the wrapper
brief: either stop populating `PheromoneSection`/`HandoffSection` when `ContextCapsule` already carries
the same content (and confirm every one of the 8 `codex.WorkerDispatch` construction sites — build,
continue, plan, colonize, quick, oracle, seal, swarm — after the change), or make `buildColonyPrimeOutput`
omit its own pheromone/handoff sections when the caller signals a dedicated channel will be delivered
alongside it. Prove with the same "seed one active signal, assert count == 1 in the assembled native
prompt" pattern this plan established for the wrapper flow.

**Impact if left unfixed:** Every native-dispatched worker (autopilot `/ant-run` builds, and any other
caller of `executeCodexBuildDispatches`) with an active FOCUS, REDIRECT, or FEEDBACK signal receives
that steering text twice in its assembled prompt, under two different headings. This does not trip
`--print-brief`'s duplication detector (which only inspects the wrapper-facing capsule + composed brief
+ skill section, never `executeCodexBuildDispatches`'s live `AssemblePrompt` output), so it is currently
invisible to the one tool built to catch exactly this class of bug.

---

## D-190-05-A: Continue's native review and watcher dispatches independently double-deliver prior-worker handoffs through a separate, unfixed channel (HandoffSection, not pheromones)

**Found during:** 190-05, while auditing all ~8 `codex.WorkerDispatch`/`codex.WorkerConfig`
construction sites per this plan's own per-caller discipline (required to establish, for each
caller, whether its capsule already embeds the pheromone section before touching
`PheromoneSection` anywhere). D-190-03-A's own text asserted "`HandoffSection` does NOT
independently duplicate on this same path today" — true for BUILD's native dispatch specifically,
but that assertion does not hold for CONTINUE's native dispatch functions, which this plan's audit
newly inspected.

**What was verified, by reading and by test:** `plannedContinueReviewDispatches`
(`cmd/codex_continue.go`) and `plannedContinueWatcherDispatch` (`cmd/codex_continue.go`) both set
`ContextCapsule: resolveCodexWorkerContext()` (which unconditionally renders
`## Previous Worker Handoffs` whenever a handoff record matches, `cmd/colony_prime_context.go:695`)
AND `HandoffSection: renderWorkerHandoffSection("continue", phase.ID, workerName)` on every
dispatch — the same "capsule + dedicated field" shape D-190-03-A found for pheromones, but for
handoffs, and specific to continue's native path.

Proven empirically, not just by reading: a throwaway probe (deleted after use) seeded one stored
handoff with `Workflow: "continue"` (the earlier probe attempt using the base test fixture's
`Workflow: "build"` handoff produced a false negative — `renderWorkerHandoffSection` filters by
workflow, so a build-workflow handoff never populates a continue-scoped `HandoffSection`, which is
why this went unnoticed) and called `plannedContinueReviewDispatches` directly. The assembled
prompt via `codex.AssembleHostedPrompt` carried `## Previous Worker Handoffs` **twice**: once from
`ContextCapsule`, once from the independently-populated `HandoffSection`.

**Why this is not fixed here:** Out of 190-05's scope, which is pheromone-specific (per this plan's
own objective: "an active pheromone signal reaches each worker TWICE"). Fixing it means deciding
whether `plannedContinueReviewDispatches`/`plannedContinueWatcherDispatch` should stop populating
`HandoffSection` (mirroring what this plan just did for `PheromoneSection` at these same two call
sites) — a decision this plan's own objective and success criteria do not name, and touching it
would silently expand this plan's declared scope past pheromones. Note
`plannedContinueWatcherDispatch`'s own comment ("The watcher... was the only one dispatched without
the relay... Nothing in the design justified the asymmetry; it was omitted") shows `HandoffSection`
was deliberately ADDED to these two call sites by a prior change, apparently without checking
whether the capsule already delivered the same content — the same oversight D-190-03-A named for
pheromones, recurring for handoffs on a different command.

**Recommended follow-up:** A future phase should apply the exact fix this plan (190-05) just applied
to `PheromoneSection` at these two call sites, to `HandoffSection` instead: remove
`HandoffSection: renderWorkerHandoffSection("continue", ...)` from both
`plannedContinueReviewDispatches` and `plannedContinueWatcherDispatch`, since `ContextCapsule`
already delivers the same content. Prove with a `Workflow: "continue"`-scoped handoff fixture (the
base `seedHandoffColonyForBriefTests` fixture's `Workflow: "build"` handoff will NOT exercise this
bug — reuse the probe pattern in this entry, not that fixture alone).

**Impact if left unfixed:** Every native continue-review or continue-watcher dispatch (`aether
continue`, no `--plan-only`) with a stored handoff matching `Workflow: "continue"` for the current
or preceding phase receives that handoff's content twice in its assembled prompt, under the same
heading. Not currently caught by any test in the suite (the existing 190 locks target pheromones
and the build native path, or the continue plan-only wrapper manifest, not continue's native
review/watcher dispatch functions).

---


Out-of-scope discoveries found during execution. Not fixed here per the executor's
scope-boundary rule (only auto-fix issues directly caused by the current task's changes).

## From 190-02 (TS-host hive double-injection removal)

### Stale ceremony-adapter snapshot fixtures (pre-existing, unrelated to hive_section)

**Found during:** Task 3 verification (`npm test --prefix .aether/ts-host`)

**Symptom:** Three tests in `.aether/ts-host/test/ceremony-snapshots.test.ts` fail:
- `Go renderSpawnPlan(build) matches snapshot`
- `Go renderWaveStart(build) matches snapshot`
- `Go renderWaveStart(continue) matches snapshot`

**Cause:** The Go runtime's caste-identity rendering (`cmd/codex_visuals.go`) now emits the
house style described in `CLAUDE.md`'s "Caste Identity System" section — emoji + ant glyph
(e.g. `🔨🐜`) plus a model tag (e.g. `[sonnet]`) — but the committed snapshot fixtures under
`.aether/ts-host/test/__snapshots__/` (`spawn-frame-builder.txt`, `stage-separator-build.txt`,
`stage-separator-continue.txt`) still expect the older single-emoji, no-model-tag format
(e.g. `🔨 Builder Bolt-69` instead of `🔨🐜 Builder [sonnet] Bolt-69`).

**Proof this predates Plan 190-02:** `git log` shows the snapshot fixture
(`.aether/ts-host/test/__snapshots__/spawn-frame-builder.txt`) was last touched in commit
`9f40bf48`, while `cmd/codex_visuals.go` was last touched in `a81cf6aa` (2026-08-18, Phase 187
work) — a later commit that both predate this plan's base commit (`b60a0075`). Nothing in
Plan 190-02's diff touches ceremony rendering, caste emoji, model tags, or these snapshot
files.

**Not fixed here because:** Out of this plan's scope (hive_section removal only). The fix is
almost certainly `AETHER_UPDATE_SNAPSHOTS=1 npm test --prefix .aether/ts-host` to regenerate
the three stale fixtures, but that is a distinct, unrelated correctness gap for a future plan
(or a quick standalone fix) to pick up.

### Inert `hive-read` mock branches left in unrelated host-integration.test.ts fixtures

**Found during:** Task 3 test rewrites

**Symptom:** Several unrelated `describe` blocks in `host-integration.test.ts` (e.g. "spawn
orchestrator initialization") have mock `callGoJSON` handlers with a `if (cmd === "hive-read")`
branch returning canned data. Since the TS host no longer calls `hive-read` anywhere (Phase
190), these branches are now dead code — never reached, harmless.

**Not fixed here because:** Scrubbing every last unreachable mock branch across ~8 unrelated
tests is cosmetic cleanup, not required for correctness or for proving the hive_section removal
(the branches don't pair with any assertion about hive content). Left as a minor, low-priority
cleanup opportunity for whoever next touches those tests.

---

## Raised by 190-REVIEW (2026-08-20)

### D-190-R-A: Cross-domain hive wisdom is now excluded outright, not discounted

**Found during:** 190-REVIEW WR-03

**Symptom:** Plan 190-02 was recorded as removing a duplicate delivery of "the SAME hive
wisdom". At the heading level that is true — every worker prompt carried
`## HIVE WISDOM (Cross-Colony Patterns)` twice and now carries it once. At the *entry* level it
is not. The deleted TS channel (`.aether/ts-host/src/hive-injector.ts`, see
`git show b60a0075:.aether/ts-host/src/hive-injector.ts`) called `hive-read --for-worker
--min-confidence 0.5` with **no** `--domain` filter and then ranked the results itself, keeping
entries from other domains at a **0.5x confidence discount**. The surviving Go channel
(`filterHiveWisdomEntriesByDomain`, `cmd/context_weighting.go:78-97`, untouched by this phase)
**excludes** any entry whose domain is not an exact match, or `general`/empty, as soon as the
repo has registered domain tags.

So for a colony whose registered tags do not span every domain present in
`~/.aether/hive/wisdom.json`, the deleted channel was the only one surfacing cross-domain wisdom
at all. Removing it narrowed coverage for that case; it was not pure deduplication.

**Corrected in this pass:** the misleading comment in
`.aether/ts-host/test/milestone-audit.test.ts` now states the selection-semantics difference
explicitly instead of implying the two channels chose the same entries.

**Deferred:** whether hard exclusion is the *right* final behaviour is a product decision about
how much cross-domain wisdom a worker should see, not a defect to patch quietly. If discounted
cross-domain inclusion is wanted, port it into `filterHiveWisdomEntriesByDomain` as its own
change with its own test. The record is now honest either way, which is what the review asked
for.
