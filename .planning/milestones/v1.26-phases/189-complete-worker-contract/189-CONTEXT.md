# Phase 189: Complete Worker Contract - Context

**Gathered:** 2026-08-20
**Status:** Ready for planning
**Source:** Direct codebase reconnaissance by the planner (no `/gsd-discuss-phase` session exists
for this phase — the owner was asleep when this was requested). Every decision below was reached
by reading the live source, not by inference from documentation, and is marked **auto-decided
(owner asleep) — revisit if wrong** per the planning brief's instruction. None of these are open
questions blocking execution; each is a concrete, conservative choice with its reasoning stated.

The orchestrator's own scouting map (grep-verified before dispatch) was correct on criteria 1 and
2's existence and largely correct on criterion 3 — but reading the full call graph turned up two
corrections to that map (D-06/D-09 below) and one design hazard the scouting could not have seen
without reading deep into `pkg/codex` (D-04/D-11 below: adding the handoff schema to the wrong
function would have silently duplicated it for every native-Codex worker). Both are recorded as
decisions, not silently fixed, so the correction is visible.

<domain>
## Phase Boundary

This phase closes three independent gaps, all confirmed by reading (not assuming) the current
code:

1. **Merged-dispatch task coverage (criterion 1).** `renderCodexBuildWorkerBrief`
   (`cmd/codex_build.go:2667`) renders Constraints/Hints/Task Success Criteria from exactly one
   `colony.Task`, resolved via `findDispatchTask(phase, dispatch)` — which looks up only
   `dispatch.TaskID`. `mergeDispatchInto` (`cmd/codex_build.go:1077`), which folds a chain of
   dependent steps into one worker, appends every merged step's ID to `dispatch.CoveredTaskIDs` but
   never changes `dispatch.TaskID` — it stays the *first* step's ID forever. Confirmed by reading
   both functions directly: for a 3-task merged dispatch, the worker's "## Assignment" section
   already lists all 3 tasks' goal text (via `dispatch.Task`, correctly built by `mergeDispatchInto`
   as a numbered list) but its "## Constraints"/"## Hints"/"## Task Success Criteria" sections show
   only task 1's. Tasks 2 and 3's constraints and success criteria never reach the worker the
   finalizer later judges against all three.

2. **The handoff/return schema the finalizer enforces is undocumented in the wrapper-facing
   brief (criterion 2).** `pkg/codex/worker.go`'s `renderResponseContract` (unexported) already
   renders the exact handoff field list — `changed_files, commands_run, verification_status,
   known_failures, open_decisions, assumptions, next_worker_instructions, do_not_repeat, freshness`
   — in one sentence, matching `WorkerHandoff`'s own JSON tags (`pkg/codex/handoff.go`) field for
   field. But it is called only from `AssembleHostedPrompt`/`AssemblePrompt`, the native-Codex
   dispatch composition path (`pkg/codex/platform_dispatch.go:892,947`, `pkg/codex/worker.go:381`).
   `renderCodexBuildWorkerBrief` — the function whose output becomes `dispatch.Brief`, the ONLY
   prompt content a Claude-Code- or OpenCode-wrapper-spawned worker ever sees (confirmed:
   `.claude/commands/ant/build.md:272`, "worker's prompt = context_capsule + brief + skill_section
   ... Nothing else, nothing invented") — never mentions it. The finalizer
   (`cmd/codex_build_finalize.go:1080`, `codex.ValidateWorkerHandoff(result.Handoff)`) rejects a
   worker whose handoff doesn't conform, but a wrapper-spawned worker was never told the shape.
   `cmd/codex_continue_finalize.go:643` (`mergeExternalContinueResults`) enforces the identical
   validation on continue's external reviewers — same asymmetry, second location.

3. **Continue's external (wrapper-mediated) dispatches carry no context capsule and no pheromone
   content at all (criterion 3).** `codexContinuePlanManifest` and `codexContinueExternalDispatch`
   (`cmd/codex_continue_plan.go:14-67`) — the structs serialized into the JSON manifest Claude Code
   and OpenCode's "Heavy External Review" flow reads — have neither a capsule field nor a pheromone
   field anywhere, confirmed by reading the full struct definitions and by grep (`resolvePheromoneSection`
   is never called anywhere in `codex_continue_plan.go`). `SkillSection` exists per-dispatch and is
   populated (`plannedExternalContinueDispatches`, line 307/320), but no wrapper instructs its
   delivery either — `.claude/commands/ant/continue.md`'s entire delivery instruction is "Pass each
   dispatch's runtime-provided `brief` verbatim" (lines 168, 176), with no mention of capsule, skill,
   or pheromone content at all. Compare `build.md`'s explicit "context_capsule (read once) + brief +
   skill_section" formula (`build.md:272`) — continue has no equivalent instruction whatsoever.

**Out of scope, explicitly:**
- Continue's INTERNAL/native dispatch path (`plannedContinueReviewDispatches`,
  `plannedContinueWatcherDispatch`, `cmd/codex_continue.go:1387,1781`) already carries
  `ContextCapsule`, `SkillSection`, `PheromoneSection`, `HandoffSection` — confirmed by reading; not
  touched by this phase because it is not broken.
- Phase 190's own scope (removing `--print-brief` duplication, `brief_path`-only manifests, hive
  double-injection). This phase is written to avoid *creating* new duplication for 190 to remove —
  see D-04/D-06/D-11 — but does not perform 190's own cleanup.
- The MVP-mode/TDD-mode toggles: this repo's `workflow.tdd_mode` default is off; TDD is applied
  opportunistically per task, per the standard heuristic (can `expect(fn(input)).toBe(output)` be
  written before the fix — yes, for both brief-rendering changes below).

</domain>

<decisions>
## Implementation Decisions

### Criterion 1 — merged-dispatch task coverage

- **D-01 (auto-decided):** Replace `findDispatchTask` (singular, resolves `dispatch.TaskID` only)
  with `findDispatchTasks` (plural), which resolves every ID returned by the existing
  `dispatchCoveredTaskIDs(dispatch)` (`cmd/codex_build.go:2170` — already correctly merge-aware,
  used today by `completedBuildTaskIDs`) against `phase.Tasks`. `renderCodexBuildWorkerBrief` loops
  over the result and renders EVERY covered task's Constraints/Hints/Task Success Criteria, not just
  the first. For the ordinary case (exactly one covered task — the overwhelming majority of
  dispatches), output is byte-identical to today: no "Task 1:" label, same headings, same bullets.
  Only when more than one task is covered does each task's block get a `**Task N:**` sub-label
  (matching the numbering convention `mergeDispatchInto` already uses for `dispatch.Task` itself —
  `"1. ...\n2. ..."`), so a worker reading a merged brief can tell which constraint belongs to which
  step.

- **D-02 (auto-decided, bonus — not named in the ROADMAP criterion):** `codegraphTextPartsForBuildBrief`
  (`cmd/codegraph_context.go:105`) also calls `findDispatchTask` (singular) to gather
  Goal/Constraints/Hints/SuccessCriteria text used to select which codebase-graph files are
  relevant to inject. Same root cause, same function, same read pass that found D-01 — for a merged
  dispatch, codegraph relevance is currently scored against task 1's text only, so tasks 2..N's
  referenced files are less likely to surface. Migrated to `findDispatchTasks` in the same change
  (Phase 188's D-03 "bonus, flagged, folded into the same plan" precedent). A repo-wide grep
  confirms `findDispatchTask` (singular) has exactly these two call sites
  (`cmd/codex_build.go:2708`, `cmd/codegraph_context.go:113`); once both migrate, the singular
  function is deleted outright — no orphaned function left behind for a future reachability ratchet
  to flag.

### Criterion 2 — the handoff schema in the brief

- **D-03 (auto-decided):** Extract `renderResponseContract`'s existing field-list sentence
  (`pkg/codex/worker.go:831`) into a new EXPORTED constant, `codex.HandoffFieldsSummary`
  (`pkg/codex/handoff.go`, beside `WorkerHandoff`/`ValidateWorkerHandoff` — the schema it describes).
  `renderResponseContract` is refactored to reference the constant instead of a literal, producing
  byte-identical output (verified by the two existing tests that already assert the
  "Final Response Contract" substring is present: `pkg/codex/worker_test.go:698,1135`). Both
  `cmd`-side additions below (build and continue) reference this ONE constant — never a
  third hand-copied sentence. This is the same "all readers converge on one canonical helper"
  idiom Phase 188 established for `loadMiddenFile`/`continueSupersededResult` (D-02/D-04 there).

- **D-04 (auto-decided — this is the correction the orchestrator's scouting map could not see
  without reading `pkg/codex`, and it changes WHERE the fix goes):** The handoff-schema note must
  NOT be added inside `renderCodexBuildWorkerBrief` itself. That function's output becomes
  `TaskBrief` for BOTH the wrapper-external path (`composeBuildManifestBrief`, which becomes
  `dispatch.Brief` in the JSON manifest) AND the native-Codex path (`executeCodexBuildDispatches`,
  `cmd/codex_build.go:1705`: `TaskBrief: renderCodexBuildWorkerBrief(...)`, confirmed by reading).
  The native-Codex path ALREADY appends `renderResponseContract(config)` — and therefore D-03's
  schema sentence — as a SEPARATE, later section via `AssembleHostedPrompt`/`AssemblePrompt`
  (`pkg/codex/platform_dispatch.go:892,947`). Adding the same content inside
  `renderCodexBuildWorkerBrief` would make every native-Codex worker see the schema TWICE. This
  exact class of mistake is already named and guarded against in this file's own doc comment on
  `composeBuildManifestBrief` (`cmd/codex_build.go:3108-3117`): *"The Go subprocess path
  deliberately does NOT use this composition — it delivers PheromoneSection and HandoffSection
  separately through WorkerConfig ..., so embedding them in the shared renderer would duplicate
  them there."* The handoff schema note is exactly this class of content. It is added in
  `composeBuildManifestBrief` (the wrapper-external-ONLY composition layer, already the home of the
  pheromone-signals append for the identical reason), referencing `codex.HandoffFieldsSummary`.
  Consequence: `TestBuildWorkerBriefIsMostlyTask` (which calls `renderCodexBuildWorkerBrief`
  directly, never `composeBuildManifestBrief`) is UNAFFECTED by this criterion's build-side change —
  see the scaffolding-ratio note below.

- **D-05 (auto-decided):** Criterion 2 applies to continue's external reviewer/watcher briefs too,
  not build alone. The phase's own ROADMAP goal states "reviewers get the same context on every
  path," and continue's finalizer independently enforces the identical schema:
  `mergeExternalContinueResults` (`cmd/codex_continue_finalize.go:643`) calls the same
  `codex.ValidateWorkerHandoff(result.Handoff)` build's finalizer calls, confirmed by reading.
  Scoping the fix to build alone would leave continue's external reviewers facing the identical
  undocumented-schema problem. Delivered in 189-02 (reviewer parity), reusing D-03's constant.

- **D-06 (auto-decided — same duplication hazard as D-04, second location):** By the same
  reasoning as D-04, the note must NOT be added inside `renderCodexContinueReviewBrief` /
  `renderCodexContinueWatcherBrief` (`cmd/codex_continue.go`) — their output becomes `TaskBrief` for
  BOTH continue's native/internal dispatch (`plannedContinueReviewDispatches`,
  `plannedContinueWatcherDispatch`, which already flow through the same
  `AssembleHostedPrompt`+`renderResponseContract` composition as build's native path, since both use
  the same `pkg/codex` worker-invocation machinery) AND continue's external dispatch
  (`plannedExternalContinueDispatches`, `cmd/codex_continue_plan.go:290`, which is EXCLUSIVELY the
  external/plan-only path — confirmed it is called only from `runCodexContinuePlanOnly`). The note
  is appended at the `plannedExternalContinueDispatches` call sites, after calling the (unmodified)
  brief renderers, mirroring D-04's placement choice exactly.

  This correction is why the plan set does not simply "add the schema everywhere the word handoff
  appears" — the fix location depends on which of two composition layers a given brief-render
  function feeds, and gets that answer wrong for either build or continue reintroduces the exact
  byte-duplication Phase 190 (`depends_on: 189`) exists to remove. Its own criterion 2 is
  "`--print-brief` asserts zero duplicated sections (pheromones and handoffs get one home each)" —
  seeding that phase with a fresh duplication for it to clean up would be a direct contradiction of
  this phase's own instruction not to do that.

### The scaffolding-ratio tension (resolved — the design fix above dissolves most of it, not a
  test change)

- **D-07 (auto-decided, resolves the orchestrator brief's flagged tension):**
  `TestBuildWorkerBriefIsMostlyTask` (`cmd/codex_build_test.go:3649`) asserts task-relevant content
  is >= 40% of `renderCodexBuildWorkerBrief`'s OWN output — it does not measure
  `composeBuildManifestBrief`'s output at all. Because D-04 places the handoff-schema note in
  `composeBuildManifestBrief`, criterion 2's build-side change cannot move this test's ratio by even
  one character. Criterion 1's change (D-01) is the only one of the two that touches
  `renderCodexBuildWorkerBrief`, and it only ever ADDS content that the test's own switch already
  counts as task content (`"Task Success Criteria"`, `"Hints"`, and — once D-08 below lands —
  `"Task Constraints"`), growing numerator and denominator together. For
  `TestBuildWorkerBriefIsMostlyTask`'s own fixture specifically (a dispatch with no `TaskID` and a
  phase with no `Tasks`), `dispatchCoveredTaskIDs` returns nil and D-01's new loop renders zero
  additional sections — so this specific test's measured brief is untouched by either criterion.
  **The executor must still run `TestBuildWorkerBriefIsMostlyTask` after both changes and record the
  before/after share in the plan SUMMARY** — this paragraph is reasoning, not a substitute for
  execution — but the design is set up so the test should not need touching. If real usage still
  shows insufficient headroom for actual merged, constraint-heavy dispatches, the only permitted
  next move is trimming brief prose elsewhere; the test's 40% floor and named-section list are never
  weakened (`planner_authority_limits`, CLAUDE.md's Definition of Done).

- **D-08 (auto-decided, bonus — a genuine pre-existing bug this phase's own reconnaissance found,
  directly adjacent to D-01's edit):** `renderCodexBuildWorkerBrief` emits the literal heading
  `"## Constraints"` (`cmd/codex_build.go:2711`), but `TestBuildWorkerBriefIsMostlyTask`'s
  counted-as-task-content switch (`cmd/codex_build_test.go:3682-3684`) checks for the name
  `"Task Constraints"` — a name `briefOwnedSections` already recognizes
  (`cmd/build_print_brief.go:197`, alongside `"Constraints"` at line 198) but that no render path has
  ever actually emitted. This is a latent naming-mismatch bug, not a policy choice: the test's own
  comment states the philosophy plainly ("the bar for adding a name here is that a worker could not
  complete this task correctly without it... Colony state, skills, pheromones and survey pointers do
  not meet it and stay on the scaffolding side" — by omission, task-declared Constraints ARE meant
  to be task content; they simply never matched the heading name the switch checks for). Renamed to
  `"## Task Constraints"` in the same change as D-01 (same function, same edit). This is a bugfix,
  not a test change — it makes the switch's own pre-existing, already-committed entry finally
  reachable, for both the pre-existing single-task case (whenever a task declares Constraints) and
  the new multi-task case D-01 adds. `renderBriefChecklist`'s parallel switch
  (`build_print_brief.go:363-365`) already lists both names; the now-permanently-unreachable
  `"Constraints"` case may be dropped there for hygiene (Claude's Discretion, not required).

### Criterion 3 — continue's external context delivery

- **D-09 (auto-decided — corrects the orchestrator's scouting map):** The three files that must
  change identically for criterion 3 are NOT `.claude/commands/ant/continue.md` +
  `.opencode/commands/ant/continue.md` + `.codex/CODEX.md`, as the scouting map listed. Confirmed by
  reading and by `diff` (both exit 0, byte-identical today): the real triplet is
  `.claude/commands/ant/continue.md` (canonical), `.claude/commands/ant-continue.md` (the flat
  installed-consumer mirror `aether install`/`aether update` write from the canonical source —
  enforced byte-identical by `TestLifecycleFlatMirrorsMatchCanonical`,
  `cmd/lifecycle_wrapper_contract_test.go:135`), and `.opencode/commands/ant/continue.md` (structural
  parity with the Claude canonical via `TestContinueWrapperCeremonyContract` and
  `TestContinueWrapperStageSkeletonAndParity`, and in practice byte-identical too). `.codex/CODEX.md`
  is not part of either test. Reading it confirms why: Codex's `aether continue` runs the
  ALREADY-FULLY-WIRED internal dispatch path (`plannedContinueReviewDispatches`/
  `plannedContinueWatcherDispatch`, confirmed carrying `ContextCapsule`/`SkillSection`/
  `PheromoneSection`/`HandoffSection` today) — it never goes through the external
  JSON-manifest-plus-markdown-ceremony flow the other two platforms use, so it needs no structural
  wrapper fix for this criterion's Go-side gap. All three byte-identical files land in ONE plan
  (189-02), written once and copied verbatim to the other two paths — never edited independently
  (the orchestrator's own instruction to keep them in one plan is honored; the file list is
  corrected).

- **D-10 (auto-decided — a small, separate, real doc-accuracy fix):** `.codex/CODEX.md`'s existing
  "Skills" and "Pheromone Signals" sections both state injection happens "during `build`, `colonize`,
  and `plan` dispatches" (lines ~190-191, ~201-202) — omitting `continue`, even though continue's
  reviewers already receive skill injection today on Codex's native path (confirmed:
  `SkillSection: resolveSkillSectionForWorkflow("continue", ...)`) and, after 189-02, reliably
  receive pheromone content on every path (native already did; external now will too). This is a
  pre-existing doc-accuracy gap this phase's own reconnaissance surfaced — directly serving the
  phase's "reviewers get the same context on every path" goal and CLAUDE.md's Definition of Done (a
  doc claim must be testable/accurate, and this one was quietly incomplete). Fix: add `continue` to
  both sentences' dispatch lists. A two-line prose change, not a restructure; `.codex/CODEX.md` is
  NOT held to D-09's byte-identical requirement with the other two files.

- **D-11 (auto-decided, mirrors D-04/D-06's reasoning for capsule/pheromone placement):**
  `ContextCapsule` and `PheromoneSection` are added as MANIFEST-LEVEL fields on
  `codexContinuePlanManifest` (`cmd/codex_continue_plan.go:43`), resolved ONCE via
  `resolveCodexWorkerContext()`/`resolvePheromoneSection()` inside `runCodexContinuePlanOnly` — NOT
  as per-dispatch fields on `codexContinueExternalDispatch`. Both are colony-wide and identical for
  every worker in the same continue run, exactly matching how build already treats capsule
  (`codexBuildManifest.ContextCapsule`, resolved once, `build.md:259`: "it is not per-dispatch data,
  reuse the same value for every worker"). Repeating either N times across N dispatches in the JSON
  manifest would be exactly the byte-duplication Phase 190 (`depends_on: 189`) exists to remove —
  choosing the single-home design now, rather than a per-dispatch duplicate 190 would then have to
  delete, is the direct instruction in this phase's own planning brief ("do not create duplication
  here that 190 must then remove"). `SkillSection` stays per-dispatch on
  `codexContinueExternalDispatch` (already present, genuinely varies per caste/task — matches
  build's existing per-dispatch skill treatment, which build.md's wrapper instruction already
  reflects: "append `dispatch.skill_section` when present").

- **D-12 (auto-decided):** New JSON field names mirror the closest existing convention rather than
  inventing new ones: `context_capsule` (matches `codexBuildManifest`'s own field,
  `cmd/codex_build.go:114`) and `pheromone_section` (matches `codexContinueExternalDispatch`'s own
  `SkillSection` field's `skill_section` tag convention one field over,
  `cmd/codex_continue_plan.go:35`).

- **D-13 (auto-decided):** The wrapper instruction added to all three D-09 files mirrors
  `build.md`'s exact phrasing pattern (`build.md:259,272`) rather than inventing new wording: read
  `continue_manifest.context_capsule` and `continue_manifest.pheromone_section` ONCE (not
  per-dispatch), prepend both verbatim ahead of each dispatch's `brief`, then append
  `dispatch.skill_section` when present. This replaces the current "Pass each dispatch's
  runtime-provided `brief` verbatim" instruction (`continue.md:168,176`) and extends the "Reads:"
  line (`continue.md:133-134`).

### Claude's Discretion

- Exact wording of the `**Task N:**` sub-label prefix for merged-dispatch Constraints/Hints/Task
  Success Criteria (D-01), provided it is unambiguous, consistent across all three sections, and
  matches the Assignment section's own `"1. / 2. / 3."` numbering in spirit.
- Whether `renderBriefChecklist`'s now-permanently-unreachable `"Constraints"` switch case (D-08) is
  removed or left as harmless dead code.
- Exact phrasing of the handoff-schema sentence appended in `composeBuildManifestBrief` and at
  `plannedExternalContinueDispatches`'s call sites (D-04/D-06), provided each references
  `codex.HandoffFieldsSummary` by symbol (not by hand-copy) and states plainly that an empty handoff
  is rejected.
- Mechanics of the new Go tests proving criteria 1 and 3 (exact fixture construction), provided each
  is a genuine invariant/proportion check per the Definition of Done (e.g., "every one of N covered
  tasks' constraints is present," not a single hardcoded string), not a named-section-exists check.
- Whether `midden`-style helper naming conventions are followed exactly for any new small function
  names (e.g. `continueExternalBriefWithHandoffSchema` is a suggested name, not mandated).

### Folded Todos

Not applicable. This CONTEXT.md was authored directly by the planner because no `/gsd-discuss-phase`
session exists for this phase (owner asleep) — there is no todo backlog to score against it.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase governance
- `.planning/ROADMAP.md` § "Phase 189" (line ~963) — the goal and three success criteria this phase
  is judged against; "Plan as ≥2 plans (build-side contract; reviewer parity)"
- `.planning/ROADMAP.md` § "Phase 190" (line ~968) — the very next phase, `depends_on: 189`; its own
  criterion 2 ("no context section delivered twice... pheromones and handoffs get one home each") is
  the direct reason D-04/D-06/D-11 place new content in the wrapper-external-only composition layer
  rather than the shared brief renderers
- `CLAUDE.md` § "Definition of Done" — a requirement is satisfied only when a command exists that
  fails when the requirement is unmet; prefer invariant/proportion tests over named-section checks
- `CLAUDE.md` § "UX Architecture" / wrapper-triplet convention — confirmed empirically this phase:
  `.claude/commands/ant/continue.md`, `.claude/commands/ant-continue.md`, and
  `.opencode/commands/ant/continue.md` are byte-identical today (`diff` exit 0 both pairs)

### Criterion 1 — merged-dispatch task coverage
- `cmd/codex_build.go:22-69` — `codexBuildDispatch` struct, incl. `CoveredTaskIDs` (line 45) and its
  doc comment
- `cmd/codex_build.go:1077-1091` — `mergeDispatchInto`: appends to `CoveredTaskIDs`, numbers
  `target.Task` as `"1. ...\n2. ..."`, NEVER changes `target.TaskID`
- `cmd/codex_build.go:2667-2814` — `renderCodexBuildWorkerBrief`, full function; the single-task
  lookup is at line 2708 (`relatedTask := findDispatchTask(phase, dispatch)`), the three sections it
  gates are Constraints (2710-2721), Hints (2722-2733), Task Success Criteria (2734-2745)
- `cmd/codex_build.go:2829-2839` — `findDispatchTask` (singular), to be replaced
- `cmd/codex_build.go:2170-2184` — `dispatchCoveredTaskIDs`, ALREADY correctly merge-aware (used by
  `completedBuildTaskIDs`); reuse this, do not reinvent the "get every covered ID" logic
- `cmd/codex_build.go:1666-1671` — `buildTaskID(task colony.Task, idx int) string`: task's own `ID`
  if set, else `"task-%d"` fallback — needed to construct matching fixture IDs in the new test
- `cmd/codegraph_context.go:105-120` — `codegraphTextPartsForBuildBrief`, the second (bonus, D-02)
  call site of `findDispatchTask`
- `cmd/codex_build_test.go:3649-3697` — `TestBuildWorkerBriefIsMostlyTask`, full function including
  its counted-section switch and its own comment explaining the philosophy
- `cmd/build_print_brief.go:185-211` — `briefOwnedSections`, the full map (both `"Constraints"` and
  `"Task Constraints"` already present)
- `cmd/build_print_brief.go:356-369` — `renderBriefChecklist`'s parallel switch (already lists both
  names)
- No existing test named `TestMergeDispatchInto`/`TestCoveredTaskIDs`/similar was found by grep —
  this merge logic has zero dedicated test coverage today; the new test is genuinely new coverage,
  not a duplicate (Nyquist rule: MISSING today)

### Criterion 2 — the handoff schema
- `pkg/codex/handoff.go:1-52` — `WorkerHandoff` struct (the 9 JSON-tagged fields), `IsEmptyWorkerHandoff`,
  `ValidateWorkerHandoff` — the schema being described; `HandoffFieldsSummary` (D-03) belongs in this
  file
- `pkg/codex/worker.go:807-836` — `renderResponseContract`, full function; the sentence to extract is
  the `Include handoff with ...` line inside the `fmt.Sprintf` (currently line ~831)
- `pkg/codex/worker.go:381` — `AssemblePrompt` call site (native/internal dispatch path,
  `WorkerConfig.TaskBrief` fed in directly)
- `pkg/codex/platform_dispatch.go:892,947` — the two `AssembleHostedPrompt(...) + ... +
  renderResponseContract(config)` compositions (hosted/native dispatch)
- `pkg/codex/worker_test.go:698,1135` — existing tests asserting the `"Final Response Contract"`
  substring is present; these must still pass unchanged after D-03's refactor (byte-identical output)
- `cmd/codex_build.go:1698-1714` — `executeCodexBuildDispatches`'s `codex.WorkerDispatch`
  construction; line 1705 (`TaskBrief: renderCodexBuildWorkerBrief(...)`) is the proof that the
  native path already feeds off the SAME renderer the wrapper-external path uses — the reason D-04
  exists
- `cmd/codex_build.go:3108-3150` — `composeBuildManifestBrief`, full function, incl. its own doc
  comment (3108-3117) already explaining the no-duplicate-for-native-path rule for pheromone/handoff
  content — the precedent D-04's placement follows
- `cmd/codex_build_finalize.go:1080-1090` — the build-finalize handoff validation call
  (`codex.ValidateWorkerHandoff(result.Handoff)`) this criterion's asymmetry exists to close
- `cmd/codex_continue_finalize.go:602-646` — `mergeExternalContinueResults`, full function; line 643
  is continue's OWN identical handoff validation call (the evidence for D-05)
- `cmd/codex_continue.go:1387-1415` — `plannedContinueReviewDispatches` (continue's native/internal
  review dispatch path — confirms it ALSO flows through `AssemblePrompt`/`renderResponseContract` via
  the shared `pkg/codex` worker-invocation machinery, the reason D-06 exists)
- `cmd/codex_continue.go:1781-1803` — `plannedContinueWatcherDispatch` (continue's native/internal
  watcher dispatch path, same reasoning)
- `cmd/codex_continue_plan.go:290-340` — `plannedExternalContinueDispatches`, full function — this IS
  the external-ONLY path (confirmed its only caller is `runCodexContinuePlanOnly`, line 141); this is
  where D-06's append belongs, wrapping the (unmodified) `renderCodexContinueWatcherBrief` (line 306)
  and `renderCodexContinueReviewBrief` (line 331) call results

### Criterion 3 — continue's external context delivery
- `cmd/codex_continue_plan.go:14-41` — `codexContinueExternalDispatch` struct, full definition (no
  capsule field, no pheromone field, `SkillSection` present at line 35)
- `cmd/codex_continue_plan.go:43-67` — `codexContinuePlanManifest` struct, full definition (no
  capsule field, no pheromone field anywhere)
- `cmd/codex_continue_plan.go:69-210` — `runCodexContinuePlanOnly`, full function; the `plan :=
  codexContinuePlanManifest{...}` literal is at line 142-161 — this is where the two new fields get
  set
- `cmd/codex_build.go:79-114` — `codexBuildManifest` struct, incl. `ContextCapsule` field (line 114)
  — the pattern D-11/D-12 mirror
- `cmd/codex_build.go:1685,1688` — `resolveCodexWorkerContext()`/`resolvePheromoneSection()` called
  once each, at manifest-assembly time — the "resolve once" pattern to copy
- `.claude/commands/ant/build.md:249,259,272` — the "Reads:" line and the exact "context_capsule
  (read once) + brief + skill_section" phrasing to mirror for continue's equivalent instruction
- `.claude/commands/ant/continue.md:68-198` — the full "Heavy External Review" section; the
  instruction to replace is step 5 (lines 168, 176: "Pass each dispatch's runtime-provided `brief`
  verbatim") and the "Reads:" line (133-134)
- `cmd/continue_wrapper_ceremony_test.go:13-101` — `TestContinueWrapperCeremonyContract`, the
  required/forbidden/in-order substring check against the two canonical paths — model for the new
  test proving delivery is instructed
- `cmd/continue_wrapper_ceremony_test.go:195-260` — `TestContinueWrapperStageSkeletonAndParity`,
  incl. `canonicalPaths := canonicalWrapperPaths(repoRoot, "continue")` (only the two `ant/` paths,
  NOT `.codex/CODEX.md`) and the `ordered_heading_parity` check between them
- `cmd/lifecycle_wrapper_contract_test.go:22-27` — `canonicalWrapperPaths`, confirms exactly two
  paths per verb
- `cmd/lifecycle_wrapper_contract_test.go:129-184` — `TestLifecycleFlatMirrorsMatchCanonical`, the
  byte-equality enforcement between `.claude/commands/ant/continue.md` and
  `.claude/commands/ant-continue.md` — this test will fail if the flat mirror is forgotten
- `.codex/CODEX.md:187-218` — the "Skills" and "Pheromone Signals" sections, the two sentences D-10
  extends with `continue`
- `cmd/codex_continue_plan_test.go` (full file, 162 lines) — existing plan-only test fixture pattern
  (`setupIntermediateContinueState`, `runCodexContinuePlanOnly(root, codexContinueOptions{...})`,
  reading `planResult["continue_manifest"].(codexContinuePlanManifest)`) — model for the new test
- `cmd/codex_continue_test.go:3010` — `setupIntermediateContinueState(t, phaseName string) (string,
  string, string, string)`, the simplest fixture builder for a mid-phase continue state
- `cmd/codex_continue_test.go:1157-1171` — `writeTestPheromones(t, dataDir,
  colony.PheromoneFile{Signals: []colony.PheromoneSignal{...}})`, the exact seeding pattern for a
  test pheromone signal (ID, Type, Priority, Source, CreatedAt, Active, Strength, Content as
  `json.RawMessage`)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `dispatchCoveredTaskIDs(dispatch)` (`cmd/codex_build.go:2170`) — already exists, already correct;
  D-01's `findDispatchTasks` is a thin wrapper that resolves these IDs against `phase.Tasks`, not a
  reimplementation of "which tasks does this dispatch cover."
- `renderResponseContract`'s existing sentence (`pkg/codex/worker.go:831`) — the schema text already
  exists and already matches `WorkerHandoff`'s JSON tags exactly; D-03 extracts it, never
  re-authors it.
- `composeBuildManifestBrief`'s existing pheromone-append pattern (`cmd/codex_build.go:3122-3132`) —
  the exact structural precedent for D-04's handoff-schema append (same function, same
  "wrapper-external-only content" rationale, same doc comment already explaining why).
- `codexBuildManifest.ContextCapsule` (`cmd/codex_build.go:114`) plus its "resolve once" call site
  (`cmd/codex_build.go:1685`) — the exact pattern D-11 copies for continue's new manifest fields.
- `writeTestPheromones`/`setupIntermediateContinueState` — existing test fixture helpers, reused
  rather than hand-rolled for the new criterion-3 test.

### Established Patterns
- **One canonical helper, all readers converge on it.** Phase 188's D-02/D-04
  (`loadMiddenFile`/`continueSupersededResult`) is the direct precedent for D-03
  (`codex.HandoffFieldsSummary`, referenced from both build's and continue's wrapper-external
  composition layers, never hand-copied a third time).
- **Wrapper-external-only content lives in the composition wrapper, not the shared renderer.**
  Already established by this exact file (`composeBuildManifestBrief`'s own doc comment) for
  pheromone signals and prior-worker handoffs; D-04/D-06 apply the identical rule to the new
  handoff-schema content, for both build and continue.
- **Manifest-level fields for colony-wide content, per-dispatch fields for per-worker content.**
  `codexBuildManifest.ContextCapsule` (manifest-level) vs. `codexBuildDispatch.SkillSection`
  (per-dispatch) is the existing split; D-11 applies the same split to continue's new capsule
  (manifest-level) and pheromone (manifest-level, since it too is colony-wide) fields, keeping
  `SkillSection` per-dispatch as it already is.
- **A guard/test that finds nothing must fail, not pass.** Established by Phase 187's and 188's own
  ratchets; the new criterion-1 test must assert against a REAL multi-task dispatch (not an
  edge case that happens to render nothing), and the new criterion-3 test must assert the manifest
  fields are actually non-empty when real signals/state exist, not merely that the struct fields
  exist.

### Integration Points
- 189-02 (reviewer parity, wave 2) depends on 189-01 (build-side contract, wave 1) because it
  references `codex.HandoffFieldsSummary`, exported by 189-01's first task. This is a genuine
  compile-order dependency, not a file-ownership overlap (189-02 never modifies any file 189-01
  modifies) — recorded via `depends_on`, per the framework's interface-first ordering guidance.
- Criterion 3's wrapper-markdown task and criterion 3's Go-manifest task both land in 189-02
  (not split into a third plan) because the wrapper instruction text directly names the two new JSON
  keys the Go task creates — writing the prose before the keys exist would require the executor to
  guess at names never confirmed against real code, and Phase 188 established that plans finish
  fastest when contracts are defined before things that consume them, not decoupled arbitrarily.

### Test-Coverage Reality
- Criterion 1's merge logic (`mergeDispatchInto`, `dispatchesFormOneJob`, `CoveredTaskIDs`) has ZERO
  existing dedicated test coverage — confirmed by grep across `cmd/codex_build_test.go`. The new
  test is genuinely new coverage (Nyquist: MISSING today), not a duplicate of an existing check.
- No test named anything like `IsMostlyTask` exists for continue's briefs — confirmed by grep across
  `cmd/codex_continue_test.go`/`cmd/codex_continue_plan_test.go`. There is no equivalent
  scaffolding-ratio budget to protect on the continue side, which is part of why D-06's placement
  choice, while still correct for duplication reasons, carries no ratio risk either way.
- `TestLifecycleFlatMirrorsMatchCanonical` already exists and already enforces the flat-mirror
  byte-equality this phase must preserve — it does not need a new test, only a correct edit (write
  the same content to all three D-09 paths).

</code_context>

<specifics>
## Specific Ideas

- The single most important finding in this phase's research is that "add the handoff schema to the
  brief" is not one fix but a fix that must land in a DIFFERENT function than the one that renders
  the shared task content, for BOTH build and continue — landing it in the shared renderer would
  silently duplicate the schema for every native-Codex worker (build AND continue), a defect that
  would not show up in any wrapper-facing test (since wrapper-facing tests only look at the
  manifest/wrapper path) and would only be caught by inspecting the assembled native-Codex prompt
  directly. D-04/D-06's placement, and the new duplication-proof tests in both plans, exist
  specifically to make this failure mode structurally impossible rather than merely unlikely.
- The scaffolding-ratio tension the orchestrator flagged as mandatory to resolve turns out to
  resolve itself once the handoff-schema placement question (above) is answered correctly — but this
  is a discovered consequence of the placement decision, not an independent test change, and the
  plan still requires the executor to prove it by running the test, not by trusting this paragraph.
- `dispatch.Task`'s existing merge-numbering (`"1. ...\n2. ..."`, set by `mergeDispatchInto`) is
  already correct and already reaches the worker today — only the Constraints/Hints/Task-Success-
  Criteria sections lag behind it. The fix brings those three sections up to the same standard the
  Assignment section already met, using the same per-task numbering convention for consistency.

</specifics>

<deferred>
## Deferred Ideas

- **Full migration of `midden_cmds.go`/`spawn_budget.go`-style writers, or any other Phase
  188-adjacent cleanup** — not this phase's concern; noted only to avoid confusion since both phases
  touch adjacent worker-context machinery.
- **Phase 190's own deduplication and `brief_path`-only manifest work** — this phase is written to
  avoid creating new duplication for 190 to remove (D-04/D-06/D-11), but does not perform any of
  190's own scope (e.g., converting continue's manifest to `brief_path`-only, removing TS-host hive
  double-injection, or auditing `--print-brief` for zero-duplication across ALL sections).
  190 `depends_on: 189` precisely so it can build on a clean base, not a duplicated one.
- **Dropping `renderBriefChecklist`'s now-dead `"Constraints"` switch case** (D-08) — left as
  discretion, not required.
- **Migrating `midden_cmds.go`-style hand-rolled decodes elsewhere in `cmd/codex_continue*.go`** —
  out of scope; not touched by this phase's changes.

</deferred>

---

## Success Criteria Coverage

| # | ROADMAP Success Criterion | Covered By |
|---|---|---|
| 1 | The brief for a merged dispatch contains every covered task's constraints and success criteria | 189-01 (Task 2) |
| 2 | The brief contains the handoff/return schema the finalizer enforces | 189-01 (Task 1, Task 3 — build) + 189-02 (Task 1 — continue) |
| 3 | Continue's external dispatches carry capsule, skill and pheromone sections, and the wrappers (all three platforms, parity-tested) instruct their delivery | 189-02 (Task 1 — Go manifest fields; Task 2 — wrapper delivery instructions) |

Every one of the three ROADMAP success criteria is covered by at least one plan. None was found to
be already satisfied in the code for the wrapper-external/build-brief path specifically; the closest
candidate (continue's native/internal dispatch path already carrying capsule+skill+pheromone) is a
*different* path from the one the ROADMAP criterion names, confirmed by reading — see D-09 and the
"Out of scope" list in `<domain>`.

---

*Phase: 189-Complete Worker Contract*
*Context gathered: 2026-08-20*
