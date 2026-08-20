# Phase 190 — Deferred Items

Discoveries made during execution that are real, verified findings but out of scope for the
plan that found them. Logged per the executor's scope-boundary rule rather than silently fixed
or silently ignored.

## D-190-01-A: Pheromone signals and prior-worker handoffs render twice in the plan-only wrapper flow

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
