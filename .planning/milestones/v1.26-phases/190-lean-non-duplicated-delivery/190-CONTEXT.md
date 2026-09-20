# Phase 190: Lean, Non-Duplicated Delivery - Context

**Gathered:** 2026-08-20
**Status:** Ready for planning
**Source:** Direct codebase reconnaissance by the planner (no `/gsd-discuss-phase` session exists for
this phase — the owner was asleep when this was requested). Every decision below was reached by
reading and tracing the live source (not by inference from documentation, and not by trusting a prior
phase's SUMMARY prose without re-verifying it), and is marked **auto-decided (owner asleep) — revisit
if wrong** per the planning brief's instruction. None of these are open questions blocking execution;
each is a concrete, conservative choice with its reasoning stated.

The orchestrator's own scouting map (grep-verified before dispatch) correctly identified that
`BriefPath` exists and is populated *somewhere*, that `briefOwnedSections` exists, and that a hive
double-injection is plausible. Reading the full call graph turned up three corrections to that map,
recorded as decisions below (D-01/D-09/D-10), not silently fixed.

<domain>
## Phase Boundary

This phase closes four independent gaps, all confirmed by reading (not assuming) the current code and
by tracing which functions the *live wrapper/autopilot paths* actually call — not just which functions
exist.

### Criterion 1 — `brief_path` for the path the wrapper actually uses

`codexBuildDispatch.BriefPath` (`cmd/codex_build.go:58-61`) is real, and `writeCodexBuildArtifacts`
(`cmd/codex_build.go:2052-2100`) populates it — but **only for the direct/native `aether build
<phase>` dispatch path** (`writeCodexBuildArtifacts` is called at `cmd/codex_build.go:606,684`, both
inside the function backing plain `aether build <phase>` with no `--plan-only`). The function backing
`aether build <phase> --plan-only` — the exact command `.claude/commands/ant/build.md:108` instructs
the wrapper to run, and the ONLY command the wrapper is allowed to run (`build.md:348`: "Do NOT run
`aether host build` from this wrapper... Fetch the manifest with `aether build $ARGUMENTS
--plan-only`") — is `runCodexBuildPlanOnlyWithOptions` (`cmd/codex_build.go:207-381`). It calls
`attachBuildDispatchContext` (line 281, sets `.Brief` only) and **never** calls
`writeCodexBuildArtifacts`. Confirmed a third way: `TestBuildPlanOnlyPrintsDispatchManifestWithoutMutatingState`
(`cmd/codex_build_test.go:460+`) asserts today, as a passing test, `if workerBriefs :=
manifest["worker_briefs"].([]interface{}); len(workerBriefs) != 0 { t.Fatalf("plan-only manifest
should not write worker briefs...") }` — i.e. the CURRENT, tested contract is that plan-only writes
NO brief files and therefore sets NO `brief_path`. `build.md`'s own prose already says to prefer
`brief_path` "when present" — but for the actual wrapper flow, it is **never** present today, so
100% of real dispatches take the inline-`brief` branch. This is what "partially exists already" meant
and what needed tracing rather than assuming.

A second, distinct duplication compounds this: `buildCodexBuildManifest` (`cmd/codex_build.go:1876`)
builds `manifest.Dispatches` as `append([]codexBuildDispatch{}, dispatches...)` (a value copy) while
`codexBuildDispatchMaps(dispatches)` (`cmd/codex_build.go:1997`) independently builds
`result["dispatches"]` from the SAME source values. Both representations ship in the same plan-only
JSON envelope (`result["dispatches"]` and `result["dispatch_manifest"]["dispatches"]`), and both
currently carry the full inline `brief` text whenever `.Brief` is non-empty — meaning the SAME brief
text ships **twice** in a single response today, independent of the brief_path question.

### Criterion 2 — `--print-brief` has the ownership registry but no duplication assertion

`briefOwnedSections` (`cmd/build_print_brief.go:192-212`) and `splitBriefSections`
(`cmd/build_print_brief.go:214-246`) exist and correctly *attribute* headings to sections — but
neither *counts* occurrences. `splitBriefSections` restarts `current`/`size` on every matching `## `
line, so if a heading occurred twice, `renderBriefComposition`/`renderBriefChecklist` would silently
show it as two separate rows (or `checklistRowFor`'s `strings.Index`-based lookup would only ever see
the FIRST occurrence and stay blind to a second one) — never fail, never warn. There is no mechanism
today that turns "a section appeared twice" into a visible, asserted failure. The handoff-schema
sentence (`composeBuildManifestBrief`, `cmd/codex_build.go:3230-3263`) is plain text with no `## `
heading of its own (by design, per 189's D-04), so a heading-counting check alone would not catch a
handoff-schema duplication — a second, sentence-based check is needed for "handoffs get one home."

`--print-brief` exists only on **build** (`cmd/codex_workflow_cmds.go:1332`) and **plan**
(`cmd/codex_workflow_cmds.go:1323`, a structurally separate implementation — confirmed no shared
renderer with build's). **Continue has no `--print-brief` command at all** (confirmed:
`grep -n "print-brief" cmd/codex_continue*.go` returns zero matches). 189-02-SUMMARY.md's own phrase
"continue-path `--print-brief` zero-duplication cleanup" is imprecise — no such command exists to
clean up. See D-06/D-07.

### Criterion 3 — the TS-host hive double-injection, confirmed and precisely located

Two independent, live channels deliver "HIVE WISDOM (Cross-Colony Patterns)" content to the same
worker prompt:

1. **Go colony-prime capsule** (`cmd/colony_prime_context.go:708-741`, inside `buildColonyPrimeOutput`,
   reached via `resolveCodexWorkerContext()`, `cmd/colony_prime_context.go:1126`): reads
   `~/.aether/hive/wisdom.json` directly, domain-scopes it, and embeds a
   `## HIVE WISDOM (Cross-Colony Patterns)` section into the context capsule. This capsule reaches
   EVERY dispatched worker on every workflow — `resolveCodexWorkerContext()` is called from
   `cmd/codex_build.go:1685,1909`, `cmd/codex_continue.go:1388,1791`, `cmd/codex_continue_plan.go:163`,
   `cmd/codex_plan.go:1478`, `cmd/codex_colonize.go:678`, `cmd/swarm_cmd.go:1039`,
   `cmd/seal_final_review.go:1017`, `cmd/internal_worker_adapter.go:421` (fallback when the caller
   didn't already supply one) — it is THE established single channel, confirmed by breadth of use, not
   assumed.

2. **TS-host `hive-injector.ts`** (`.aether/ts-host/src/hive-injector.ts`, `readHiveWisdom` /
   `resolveDomainTags`): calls Go's `hive-read`/`registry-list` CLI SEPARATELY, formats its own
   `## HIVE WISDOM (Cross-Colony Patterns)` section (the identical header string, confirmed by
   reading `formatHiveWisdomSection`, `hive-injector.ts:144`), and attaches it as
   `dispatch.hive_section` in `.aether/ts-host/src/host.ts` at **four call sites**, each following the
   identical two-line pattern (`hiveSection = await prepareHiveSection(bridge); for (const d of
   dispatches) d.hive_section = hiveSection;`):
   - `runDryRunDispatchedCommand` (host.ts:603-659) — the SHARED dry-run function used by `aether host
     build|plan|continue --dry-run` for ALL THREE workflows via one `definition.ceremonyWorkflow`
     switch.
   - `runDispatchedBuildCommand` (host.ts:1098-1288) — build's real (non-dry-run) dispatch runner.
   - `runDispatchedPlanCommand` (host.ts:1294-1402) — plan's real dispatch runner.
   - `runDispatchedContinueCommand` (host.ts:1408-1478) — continue's real dispatch runner.

   `dispatch.hive_section` then flows through `toWorkerDispatches` (host.ts:480-482) into
   `worker-dispatch.ts`'s `dispatchRealWorker` (line 287: `if (dispatch.hive_section !== undefined)
   request.hive_section = dispatch.hive_section;`), which POSTs a JSON request to Go's
   `internal-worker-adapter` subcommand. On the Go side, `cmd/internal_worker_adapter.go:404`:
   `skillSection := joinInternalWorkerSections(request.SkillSection, request.HiveSection)` — this
   MERGES the TS-computed hive text into `codex.WorkerConfig.SkillSection`, which
   `AssemblePrompt`/`AssembleHostedPrompt` (`pkg/codex/worker.go:381`, `pkg/codex/platform_dispatch.go:892,947`)
   renders as its own section in the assembled prompt — ALONGSIDE `ContextCapsule`, which (per point 1
   above) already contains the SAME hive content. A worker dispatched through
   `internal-worker-adapter` sees "HIVE WISDOM (Cross-Colony Patterns)" **twice**.

   **Both named live paths are confirmed reachable, not hypothetical:**
   - "Continue dry-run" is the wrapper's OWN documented path:
     `.opencode/commands/ant/continue.md:263`: *"Do NOT run `aether host continue --classic-ceremony`
     without `--dry-run` from this wrapper; that triggers the TS host dispatched path which
     duplicates the wrapper's own reviewer spawning. Always use `aether host continue --dry-run`."*
     `.claude/commands/ant/continue.md` uses the identical `aether host continue --dry-run` call (per
     189-VERIFICATION.md's traced chain). Every interactive Claude Code/OpenCode `/ant-continue` run
     hits `runDryRunDispatchedCommand`.
   - "Build path" is the AUTOPILOT lane: `build.md:348` explicitly forbids the interactive wrapper
     from running `aether host build` — so `runDispatchedBuildCommand`'s real dispatch is reached by
     `/ant-run` (autopilot) or any other host-driven invocation, not the interactive wrapper. STATE.md
     confirms both lanes ("Both Aether lanes (interactive and `/ant-run`) must pass the benchmark
     gate") are real, tested surfaces this milestone cares about.

   `runDispatchedPlanCommand`'s identical pattern (plan's real dispatch, host.ts:1313-1315) and
   `runDryRunDispatchedCommand`'s coverage of plan-dry-run and build-dry-run were not named in the
   ROADMAP's parenthetical but are the SAME defect via the SAME shared function — see D-10 for why all
   four sites are fixed together, not just the two named.

### Criterion 4 — the measurement

"Byte count measurably drops" is satisfied by an invariant, not a raw number (per CLAUDE.md's
Definition of Done and this repo's own 186x-undercount lesson): once criterion 1 lands, a dispatch
with a successfully-written `brief_path` never ALSO carries the same text under `brief` in the JSON
envelope returned by `aether build <phase> --plan-only` — checked by asserting the composed brief's
own distinguishing text (e.g. a task's `Goal` string, unique per test fixture) is **absent** from the
raw JSON bytes of the response when `brief_path` is present. This is a stronger, more durable proof
than a byte-count number, which would need re-baselining every time an unrelated field grows. The
measured percentage/byte-delta for a realistic multi-dispatch phase is still recorded in 190-01's
SUMMARY for traceability, matching the "record the measured share, whether or not it changed" pattern
189-01 established for `TestBuildWorkerBriefIsMostlyTask`.

**Out of scope, explicitly:**
- **Continue's own `brief_path` mechanism.** Criterion 1's literal text names `build.md` only ("the
  wrapper passes paths (build.md prose updated to match)"). Continue's external dispatch (`codexContinueExternalDispatch.Brief`,
  `cmd/codex_continue_plan.go:34`) still ships inline-only; adding a parallel `brief_path` there is
  real future work but is not what this criterion asks for, and inventing it here would be scope
  expansion beyond the roadmap's own words. Noted, not built.
- **PLAN's separate `--print-brief` implementation.** Structurally independent of build's (confirmed
  by reading); criterion 2 names `--print-brief` generically but the concrete, load-bearing,
  ROADMAP-motivated instance is build's (the one 189's own SUMMARYs point at). Not touched.
- **`prompt-assembler.ts`'s `assemblePrompt`/`hiveSection` config field.** Confirmed via repo-wide
  grep to have ZERO non-test callers — it appears to be dead code, unconnected to the runtime path
  (`worker-dispatch.ts` builds requests for Go's `internal-worker-adapter` directly; it never calls
  TS's own `assemblePrompt`). This is an *unused* function, not a *duplicating* one — it is not part
  of criterion 3's "double injection" (nothing is doubly injected through code nobody calls). Left
  untouched; a candidate for Phase 191's Dead Wood sweep, not this phase's problem.
- **The stray `.claude/worktrees/agent-*/` directories** containing old copies of
  `aether-colony-build-cycle/SKILL.md`. Pre-existing orphaned-worktree hygiene issue, unrelated to
  Phase 190's four criteria. Not touched, not mentioned again.
- **`colony/agents/*.yaml`, `model-routing.yaml`, and the other zero-reader configs Phase 191 owns.**
  No file overlap with this phase's targets.

</domain>

<decisions>
## Implementation Decisions

### Criterion 1 — build's plan-only `brief_path`

- **D-01 (auto-decided):** Extract the per-dispatch brief-writing loop currently inline inside
  `writeCodexBuildArtifacts` (`cmd/codex_build.go:2057-2080`) into a new shared function,
  `writeBuildWorkerBriefFiles(root string, phase colony.Phase, buildDirRel string, dispatches
  []codexBuildDispatch, startedAt time.Time, clearInlineBrief bool) ([]string, []codexBuildDispatch,
  error)`. `writeCodexBuildArtifacts` calls it with `clearInlineBrief=false` (its existing, TESTED
  contract — see D-02). `runCodexBuildPlanOnlyWithOptions` calls it with `clearInlineBrief=true` right
  after `attachBuildDispatchContext` (line 281), before `buildCodexBuildManifest` is constructed
  (line 293) and before `codexBuildDispatchMaps` builds its map (line 323) — both of which read off
  the SAME `dispatches` values, so blanking once, early, fixes both JSON representations identically.
  This is the same "one canonical helper, all readers converge on it" idiom Phase 188 established
  (`loadMiddenFile`/`continueSupersededResult`) and Phase 189 reused for `codex.HandoffFieldsSummary`.

- **D-02 (auto-decided — the reason `clearInlineBrief` is a flag, not a universal change):**
  `writeCodexBuildArtifacts`'s existing behavior (Brief stays populated after writing to disk) is
  **not** changed, because two EXISTING, PASSING tests require it: the composed-brief/pheromone-parity
  test (`cmd/codex_build_test.go:~280-372`, asserting `dispatch.Brief` is non-empty in the persisted
  `manifest.json` for direct `aether build 1`) and `TestDispatchEntryCarriesBriefPath`
  (`cmd/codex_build_test.go:378-458`, same direct-path assumption). Both exercise `aether build <phase>`
  WITHOUT `--plan-only` — the native/direct dispatch path, a genuinely different scenario from the
  wrapper-facing plan-only manifest this phase's criterion targets. Silently changing shared behavior
  to satisfy a NEW requirement while breaking an OLD, still-valid one is exactly the kind of
  unannounced regression CLAUDE.md's Definition of Done exists to prevent. The flag keeps both
  contracts intact and named.

- **D-03 (auto-decided, consistency bonus):** `buildCodexBuildManifest` already accepts a
  `workerBriefs []string` parameter (`cmd/codex_build.go:1876`) and the plan-only call site
  (`cmd/codex_build.go:293`) currently passes `nil`. Once `writeBuildWorkerBriefFiles` returns real
  paths, pass them through instead of `nil`, so `manifest.WorkerBriefs` is populated for plan-only the
  same way it already is for the direct path — closing a pre-existing, unrelated inconsistency found
  during the same read pass, not a new requirement.

- **D-04 (auto-decided — names a deliberate, foreseen test-contract change, not a regression):**
  `TestBuildPlanOnlyPrintsDispatchManifestWithoutMutatingState`'s assertion at
  `cmd/codex_build_test.go:568-569` (`"plan-only manifest should not write worker briefs"`) tests
  TODAY's gap directly and must be INVERTED, not left alone — plan-only now legitimately writes brief
  files as a side effect (consistent with the many OTHER side effects `--plan-only` already has:
  `beginBuildAttempt`, `store.SaveJSON(manifestRel, manifest)` — it is not, and was never claimed to
  be, a pure/dry-run function; that role belongs to `printWorkerBriefs`/`--print-brief`, a fully
  separate call graph by its own doc comment, `cmd/build_print_brief.go:21-26`). The task's action
  must call this out explicitly so the executor does not mistake the assertion flip for a design
  smell to route around.

### Criterion 2 — `--print-brief`'s duplication assertion

- **D-05 (auto-decided):** New function `duplicatedBriefSections(assembled string)
  []string` in `cmd/build_print_brief.go`, checking TWO distinct duplication shapes in one pass over
  the SAME assembled text `printWorkerBriefs` already has in hand (capsule + composed brief + skill
  section, matching the denominator `renderBriefComposition`/`renderBriefChecklist` already use, per
  WR-04's precedent that the assembled total is capsule+brief+skills, not brief alone):
  1. Any heading registered in `briefOwnedSections` occurring more than once (extends
     `splitBriefSections`'s counting to detect repeats instead of silently overwriting `current`).
  2. The handoff-schema sentence (anchored on a stable substring drawn from
     `codex.HandoffFieldsSummary`, since it is plain text with no heading of its own by 189's own D-04
     design) occurring more than once.
  `printWorkerBriefs` calls this for every matched dispatch and returns an error (non-zero exit for
  `aether build <phase> --print-brief`) naming the dispatch and the duplicated section(s) if any are
  found — "asserts," per the ROADMAP's own word, means a command that can fail, not an informational
  row nobody reads.

- **D-06 (auto-decided — scope, corrects imprecise phrasing in a prior phase's own SUMMARY):**
  Criterion 2's mechanism is build's `--print-brief` specifically. `continue` has **no**
  `--print-brief` command (confirmed absent by grep); 189-02-SUMMARY.md's phrase "continue-path
  `--print-brief` zero-duplication cleanup" does not describe an existing surface to clean up. `plan`'s
  `--print-brief` is a structurally separate implementation, not named as the motivating instance by
  either the ROADMAP text or 189's own SUMMARYs (both of which point at build's brief). Not touched.

- **D-07 (auto-decided — an already-satisfied part, recorded honestly per the planning brief's
  instruction rather than re-planned):** Continue's "handoffs and pheromones get one home" property is
  **already structurally guaranteed** by 189: `codexContinuePlanManifest.ContextCapsule` /
  `.PheromoneSection` are manifest-level fields resolved ONCE (`cmd/codex_continue_plan.go:62-63,163`),
  never per-dispatch, so they cannot duplicate within a single dispatch's brief by construction; the
  handoff schema is appended exactly once via `continueExternalBriefWithHandoffSchema`
  (`cmd/codex_continue_plan.go`, added in 189-02), proven by
  `TestContinueExternalDispatchBriefsStateHandoffSchemaOnceNotOnNativePath`. No new continue-side test
  or code is required for criterion 2's literal ask. This phase adds no new continue work under
  criterion 2 — see the Success Criteria Coverage table for the explicit "already satisfied" mark.

### Criterion 3 — TS-host hive double-injection

- **D-08 (auto-decided):** The fix is removal, not reconciliation. Go's colony-prime capsule
  (`resolveCodexWorkerContext()`) is the established, single, already-multiply-consumed channel (nine
  call sites across build/continue/plan/colonize/swarm/seal — see `<domain>` above); the TS-host's
  independently-computed `hive_section` is the newer, narrower, redundant one (a single module,
  `hive-injector.ts`, with one real consumer). Remove `hive_section` end to end: `prepareHiveSection`,
  `emitHiveSummary`, all four `host.ts` attachment sites, the `hive-injector.ts` module itself, the
  `HiveSection` field on `cmd/internal_worker_adapter.go`'s request struct, and the
  `joinInternalWorkerSections` call that merged it into `SkillSection`.

- **D-09 (auto-decided — corrects the orchestrator's implicit framing of "two" injection points):**
  The double-injection pattern exists at **four** TS call sites, not two: the shared
  `runDryRunDispatchedCommand` (covering build/plan/continue dry-run in one function) plus three
  separate real-dispatch runners (`runDispatchedBuildCommand`, `runDispatchedPlanCommand`,
  `runDispatchedContinueCommand`). The ROADMAP's parenthetical ("build path and continue dry-run")
  names two REPRESENTATIVE, confirmed-live examples (autopilot's build dispatch; the wrapper's own
  continue dry-run), not an exhaustive list — `runDispatchedPlanCommand`'s real path and
  `runDryRunDispatchedCommand`'s coverage of plan/build-dry-run carry the byte-for-byte identical
  defect via the byte-for-byte identical two-line pattern. Per `planner_authority_limits` (no
  legitimate reason to leave a mechanically-identical instance of a named defect unfixed) and per this
  phase's own goal ("no context section delivered twice," unqualified), all four sites are fixed
  together in one plan.

- **D-10 (auto-decided):** `joinInternalWorkerSections` (`cmd/internal_worker_adapter.go:431-439`) has
  exactly one call site (line 404). Once `HiveSection` is removed, its only remaining argument is
  `request.SkillSection` — a single-argument "join" is a no-op wrapper around `strings.TrimSpace`.
  Delete the function outright and use `strings.TrimSpace(request.SkillSection)` directly, mirroring
  189's own D-02 precedent ("once both migrate, the singular function is deleted outright — no
  orphaned function left behind for a future reachability ratchet to flag").

- **D-11 (auto-decided — the repo-quirk safety net the orchestrator flagged):** `.aether/ts-host/dist/`
  is explicitly UN-ignored in `.gitignore` (`!.aether/ts-host/dist/`, `!.aether/ts-host/dist/**`,
  lines 112-113) and IS tracked in git (confirmed: `git ls-files .aether/ts-host/dist/` returns 84
  files). `cmd/ts_host_artifacts.go:13` hardcodes `tsHostEntryRelPath = "dist/host.js"` — this
  compiled file, not `src/host.ts`, is the literal runtime entrypoint the Go binary invokes. The task
  MUST run `rm -rf .aether/ts-host/dist && npm run build --prefix .aether/ts-host` (a clean rebuild,
  not an incremental one — `tsc` does not delete stale output for deleted source files, so an
  incremental build would leave `dist/hive-injector.js`/`.d.ts` behind even after `src/hive-injector.ts`
  is deleted) and then verify via `git status --porcelain .aether/ts-host/dist/` that: (a) changes
  exist at all (proves the rebuild touched something, not a no-op), and (b)
  `dist/hive-injector.js`/`dist/hive-injector.d.ts` no longer appear in `git ls-files`. `src/**/*.js`
  and `test/**/*.js` are separately gitignored (`.gitignore:114-117`) — any stray compiled `.js` files
  already sitting in `src/` or `test/` locally are untracked build byproducts, not something this task
  manages.

- **D-12 (auto-decided):** `.aether/ts-host/src/hive-injector.ts` and its dedicated test,
  `.aether/ts-host/test/hive-injector.test.ts`, are deleted outright — confirmed by repo-wide grep
  that `resolveDomainTags`/`readHiveWisdom`/`formatHiveWisdomSection` have no callers outside
  `host.ts` (being removed) and this test file itself.

### Criterion 4 — the measurement

- **D-13 (auto-decided):** Measured as the invariant described in `<domain>` above (brief text absent
  from the plan-only JSON envelope once `brief_path` is present), landing in 190-01's Task 1 alongside
  the D-01 code change, plus a recorded before/after byte count for a realistic multi-dispatch fixture
  in the plan's SUMMARY — matching CLAUDE.md's stated preference for invariants over one-off numbers,
  while still leaving a traceable measurement.

### Claude's Discretion

- Exact wording of `--print-brief`'s duplicate-section failure message, provided it names the
  dispatch and the specific duplicated section(s).
- Exact Go test function names, provided each is a genuine invariant/proportion check (Definition of
  Done), not a named-string-exists check.
- Whether `duplicatedBriefSections`'s two checks (heading-count, handoff-sentence-count) are one
  function or two closely-related ones in the same file — either satisfies D-05 as written.
- Exact anchor substring chosen from `codex.HandoffFieldsSummary` for the handoff-sentence duplication
  check, provided it is specific enough not to false-match unrelated text.

### Folded Todos

Not applicable. This CONTEXT.md was authored directly by the planner because no `/gsd-discuss-phase`
session exists for this phase (owner asleep) — there is no todo backlog to score against it.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase governance
- `.planning/ROADMAP.md` § "Phase 190" (line ~968) — the goal and four success criteria this phase is
  judged against; `depends_on: 189`
- `.planning/ROADMAP.md` § "Phase 191" (line ~973) and § "Phase 192" (line ~978) — read to confirm
  neither creates or removes scope this phase would collide with (confirmed: no overlap)
- `.planning/phases/189-complete-worker-contract/189-CONTEXT.md`,
  `.planning/phases/189-complete-worker-contract/189-01-SUMMARY.md`,
  `.planning/phases/189-complete-worker-contract/189-02-SUMMARY.md`,
  `.planning/phases/189-complete-worker-contract/189-03-SUMMARY.md` — the schema-placement decisions
  (D-03/D-04/D-06 there) this phase must not undo; both SUMMARYs explicitly flag Phase 190 as their
  consumer
- `CLAUDE.md` § "Definition of Done" — a requirement is satisfied only when a command exists that
  fails when the requirement is unmet; prefer invariant/proportion tests over named-section checks

### Criterion 1 — `brief_path`
- `cmd/codex_build.go:22-69` — `codexBuildDispatch` struct, `Brief`/`BriefPath` fields and their doc
  comments (lines 51-61)
- `cmd/codex_build.go:79-141` — `codexBuildManifest` struct, incl. `WorkerBriefs` (104),
  `Dispatches` (115, raw struct marshal — a SECOND representation of the same data alongside
  `codexBuildDispatchMaps`'s map)
- `cmd/codex_build.go:207-381` — `runCodexBuildPlanOnlyWithOptions`, full function; insertion point is
  immediately after line 281 (`attachBuildDispatchContext`), before line 293
  (`buildCodexBuildManifest`) and line 323 (`codexBuildDispatchMaps`)
- `cmd/codex_build.go:1876-1946` — `buildCodexBuildManifest`, full function; line 1890-1893 shows
  `workerBriefs` becomes `manifest.WorkerBriefs`; line 1936 shows `Dispatches` is a VALUE COPY
  (`append([]codexBuildDispatch{}, dispatches...)`), so blanking `.Brief` on the source slice before
  this call is what makes both downstream JSON representations agree
- `cmd/codex_build.go:1997-2050` — `codexBuildDispatchMaps`, full function; lines 2029-2034 already
  correctly gate `"brief"`/`"brief_path"` on non-empty — no change needed there once the underlying
  data is blanked
- `cmd/codex_build.go:2052-2100` — `writeCodexBuildArtifacts`, full function; lines 2057-2080 are the
  loop to extract into `writeBuildWorkerBriefFiles`
- `cmd/codex_build_test.go:~260-372` — the composed-brief/pheromone-parity test (direct `aether build
  1`, no `--plan-only`) whose `dispatch.Brief`-non-empty assertions (line ~331-333) must survive
  UNCHANGED
- `cmd/codex_build_test.go:374-458` — `TestDispatchEntryCarriesBriefPath` (direct path, unaffected by
  this change — do not confuse with the plan-only test below)
- `cmd/codex_build_test.go:460-629` — `TestBuildPlanOnlyPrintsDispatchManifestWithoutMutatingState`;
  lines 568-569 are the assertion that must be INVERTED (see D-04)

### Criterion 2 — duplication assertion
- `cmd/build_print_brief.go:12-123` — `printWorkerBriefs`, full function (the insertion point for the
  new duplication check, both checklist and `--full` branches)
- `cmd/build_print_brief.go:185-246` — `briefOwnedSections`, `splitBriefSections` (the existing
  first-match-only scaffolding to extend into a counting check)
- `cmd/build_print_brief.go:350-427` — `renderBriefChecklist`, the existing warning-row pattern
  (`STALE MAP WARNING`) to mirror stylistically, though duplication should fail the command, not just
  annotate a row
- `cmd/codex_build.go:3218-3263` — `composeBuildManifestBrief`, incl. the exact handoff-schema
  sentence text (line 3233) to anchor the sentence-duplication check on
- `cmd/build_print_brief_test.go` (full file) — existing test fixture patterns
  (`printBriefFixture`, `basePrintBriefState`, `runPrintBriefCmd`) to reuse for the new tests

### Criterion 3 — TS-host hive fix
- `cmd/colony_prime_context.go:708-741` — the Go-side hive wisdom section inside
  `buildColonyPrimeOutput`, the canonical channel that survives
- `cmd/colony_prime_context.go:1126` — `resolveCodexWorkerContext()`, confirmed called from nine sites
  across the codebase (see `<domain>` above) — the breadth that makes it "canonical," not an assertion
- `.aether/ts-host/src/hive-injector.ts` (full file, 150 lines) — `readHiveWisdom`,
  `resolveDomainTags`, `formatHiveWisdomSection`, all to be deleted with the file
- `.aether/ts-host/src/host.ts:513-544` — `emitHiveSummary`, `prepareHiveSection`, to be deleted
- `.aether/ts-host/src/host.ts:603-659,1098-1288,1294-1402,1408-1478` — the four call sites (dry-run
  shared runner; build/plan/continue real runners), each with its own `hiveSection =
  prepareHiveSection(...)` + attachment loop + `emitHiveSummary(dispatches)` call to remove
- `.aether/ts-host/src/host.ts:447-497` — `toWorkerDispatches`; lines 480-482 map `hive_section`
  through — remove
- `.aether/ts-host/src/host.ts:367-372` — `CeremonyDispatchLike` interface, `hive_section` field to
  remove
- `.aether/ts-host/src/host.ts:422-426` — `HostInjectedDispatchFields` type, `hive_section` field to
  remove (keep `task_brief`)
- `.aether/ts-host/src/worker-dispatch.ts:266-290` — `dispatchRealWorker`; line 287
  (`if (dispatch.hive_section !== undefined) request.hive_section = ...`) to remove
- `.aether/ts-host/src/worker-dispatch.ts:367-383` — `GoWorkerDispatchRequest` interface,
  `hive_section?: string;` to remove
- `.aether/ts-host/src/types.ts:95-114` — `BuildDispatch` interface; line 111-112 (`hive_section` +
  its doc comment) to remove
- `cmd/internal_worker_adapter.go:28-46` — `internalWorkerDispatchRequest` struct; line 40
  (`HiveSection`) to remove
- `cmd/internal_worker_adapter.go:404` — `joinInternalWorkerSections(request.SkillSection,
  request.HiveSection)`, to simplify to `strings.TrimSpace(request.SkillSection)` directly
- `cmd/internal_worker_adapter.go:431-439` — `joinInternalWorkerSections`, to delete outright (D-10)
- `cmd/internal_worker_adapter_test.go:24` — test fixture with a `"hive_section"` key, to update
- `.aether/ts-host/test/host-integration.test.ts:686,722,753,774,795,1779` — the six `it(...)` blocks
  naming "hive"/"hive_section" to remove or rewrite (1779's is a mixed test — strip only the
  hive-specific assertions, keep the spawn/iteration coverage)
- `.aether/ts-host/test/worker-dispatch.test.ts:193,202` — the hive_section-specific assertions to
  remove
- `.aether/ts-host/test/hive-injector.test.ts` — delete entirely (D-12)
- `.gitignore:33,111-117` — the `dist/` un-ignore for `.aether/ts-host/dist/` specifically, and the
  separate ignores for `src/**/*.js` and `test/**/*.js`
- `cmd/ts_host_artifacts.go:13` — `tsHostEntryRelPath = "dist/host.js"`, proof `dist/` (not `src/`) is
  the runtime entrypoint
- `.aether/ts-host/package.json` — `"build": "tsc -p tsconfig.build.json"`,
  `"typecheck": "tsc --noEmit"`, `"test": "node --import tsx ... --test test/*.test.ts"`
- `.aether/ts-host/tsconfig.json` — `include: ["src/**/*.ts", "test/**/*.ts"]` (typecheck covers test
  files; a stale test reference to a removed field/module fails `npm run typecheck`)
- `.aether/ts-host/tsconfig.build.json` — `exclude: ["test/**/*.ts"]` (the BUILD step does not need
  tests fixed first; `npm run build` can succeed with tests still referencing the old shape, which is
  why Task 2/3 can be sequenced source-then-tests)

### Wrapper/doc surfaces (criterion 1's "build.md prose updated to match")
- `.claude/commands/ant/build.md:249,259,272,348` — the exact prose to update: line 259's "both carry
  the same bytes, so use either VERBATIM" becomes inaccurate once brief is blanked whenever brief_path
  succeeds; line 272's "falling back to `dispatch.brief` inline when it is absent" becomes the
  documented RARE case (disk-write failure), not the routine one
- `.claude/commands/ant-build.md`, `.opencode/commands/ant/build.md` — confirmed byte-identical to the
  canonical file today (`diff` exit 0 both pairs) — all three edited identically, in one task, per the
  established wrapper-triplet rule (CLAUDE.md, 189-CONTEXT.md D-09/D-10)
- `.aether/commands/build.yaml:31` — the YAML source's own "both carry the same bytes, byte for byte"
  phrasing, same fix
- `cmd/command_guide.go:319` (inside `catalog["build"]`) — the Codex-lane command guide's identical
  phrasing, same fix; this is Go source, requires `go build`/`go test` after editing, not just a
  markdown change
- `.aether/skills/colony/aether-colony-build-cycle/SKILL.md:205` — the Codex skill's identical
  phrasing, same fix
- `build.md:337-344` "Cross-Platform Drift Guard" section — the file's OWN instruction naming these
  three companion surfaces (`build.yaml`, `command_guide.go`, the skill) as required companions to any
  manifest-handling prose change; followed here, not invented
- `cmd/command_guide_test.go` — run in full as part of verification (contains
  `TestCodexLifecycleYamlAndGuidesAgreeOnWorkerActivity`,
  `TestCodexLifecycleSkillMirrorsWorkerActivityContract`,
  `TestWrapperSourcesUseTypeScriptHostManifestSpine`, and 14 other parity tests spanning
  YAML/Go-guide/skill/wrapper agreement)
- `cmd/lifecycle_wrapper_contract_test.go:129-184` — `TestLifecycleFlatMirrorsMatchCanonical`, the
  byte-equality enforcement between `.claude/commands/ant/build.md` and `.claude/commands/ant-build.md`
- `cmd/build_wrapper_ceremony_test.go` — structural parity between the Claude and OpenCode build
  wrappers

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `store.AtomicWrite` / `displayDataPath` (already used inside `writeCodexBuildArtifacts`) — the exact
  disk-write and path-display mechanism `writeBuildWorkerBriefFiles` reuses, not reinvents.
- `briefOwnedSections` / `splitBriefSections` (`cmd/build_print_brief.go`) — the existing heading
  registry and section-splitting scaffolding `duplicatedBriefSections` extends rather than
  reimplements.
- `resolveCodexWorkerContext()` — already the established, multiply-consumed single channel for hive
  wisdom; criterion 3 removes a redundant SECOND channel, it does not build a new first one.
- `codex.HandoffFieldsSummary` (`pkg/codex/handoff.go`, from 189) — the exact anchor text source for
  the handoff-sentence duplication check.

### Established Patterns
- **One canonical helper, all readers converge on it.** Phase 188's D-02/D-04
  (`loadMiddenFile`/`continueSupersededResult`), reused by Phase 189 for `codex.HandoffFieldsSummary`,
  reused again here for `writeBuildWorkerBriefFiles` (D-01) and for keeping `resolveCodexWorkerContext()`
  as hive wisdom's sole channel (D-08).
- **Delete the now-single-caller/now-trivial helper outright rather than leaving it as harmless dead
  code.** Established by 189's D-02 (`findDispatchTask` deleted once both call sites migrated), applied
  here to `joinInternalWorkerSections` (D-10).
- **A flag on a shared function, not a behavior change to the function itself, when two callers need
  genuinely different, both-still-valid contracts.** New to this phase (D-02) — `writeCodexBuildArtifacts`'s
  callers need `Brief` to survive; `runCodexBuildPlanOnlyWithOptions` needs it cleared. Neither caller's
  existing tests may regress.
- **A guard/test that finds nothing must fail, not pass.** Established by Phase 187/188/189's own
  ratchets; `duplicatedBriefSections` must be proven against a REAL duplicate (not an edge case that
  happens to render nothing), and the criterion-4 invariant test must assert on a fixture with
  genuinely present `brief_path` + non-trivial brief content, not an empty-dispatch edge case.
- **Compiled output is tracked and load-bearing; a source edit alone changes nothing at runtime.**
  This project's own CLAUDE.md and prior sessions' memory both name this exact failure mode for
  `.aether/ts-host/`. D-11 exists specifically to make this phase's own criterion-3 work immune to it.

### Integration Points
- Plan 190-01 (build brief_path + duplicate detector + wrapper/doc prose) and Plan 190-02 (TS-host hive
  fix) share ZERO files (`cmd/codex_build.go`/`cmd/build_print_brief.go`/wrapper docs vs.
  `cmd/internal_worker_adapter.go`/`.aether/ts-host/**`) — both run in Wave 1, fully parallel, no
  `depends_on` between them.
- Within 190-01, Task 3 (wrapper/doc prose) depends on Task 1's exact field-presence behavior to
  describe accurately (brief_path becomes the routine case, not a co-equal alternative) — sequenced
  after Task 1, though it shares no files with Task 1 or Task 2.
- Within 190-02, Task 3 (TS test updates) depends on Task 2 (TS source removal) having already deleted
  the fields/module the old tests reference — `npm run typecheck` (which DOES include test files, per
  `tsconfig.json`'s `include`) will fail if sequenced the other way.

### Test-Coverage Reality
- The brief-writing loop currently inside `writeCodexBuildArtifacts` has coverage via the
  composed-brief/pheromone-parity test and `TestDispatchEntryCarriesBriefPath` — both exercise the
  DIRECT path only; the plan-only path's brief_path behavior has ZERO existing coverage today
  (confirmed: `TestBuildPlanOnlyPrintsDispatchManifestWithoutMutatingState` explicitly asserts the
  OPPOSITE of what this phase delivers). The new/inverted assertions are genuinely new coverage
  (Nyquist: MISSING today for the plan-only path specifically), not a duplicate of the direct-path
  tests.
- No existing test anywhere counts occurrences of a `## ` heading (all existing brief tests check
  presence/absence via `strings.Contains`, never a count) — `duplicatedBriefSections`'s tests are
  genuinely new coverage.
- `.aether/ts-host/test/host-integration.test.ts`'s SIX hive-related tests today PROVE the double
  injection exists (they assert `hive_section` gets attached) — they are the artifacts of the original
  feature-add commit and must be rewritten to assert the field's absence, not merely deleted and
  forgotten (a silent deletion would leave no regression lock against the field quietly returning).

</code_context>

<specifics>
## Specific Ideas

- The single most load-bearing finding in this phase's research is that `BriefPath` being "populated"
  (confirmed true) and `BriefPath` being populated **on the path the wrapper actually calls** are two
  different claims, and only the first was true before this phase. The orchestrator's own brief
  anticipated exactly this trap ("does the wrapper actually pass paths instead of inline briefs?") —
  the answer, after tracing the real call graph and finding a passing test that asserts the opposite of
  what criterion 1 wants, is that it does not yet, and the fix is a small, mechanical extraction, not a
  redesign.
- The hive double-injection (criterion 3) is not a hypothetical "could happen" — both named paths
  (continue's wrapper-driven dry-run, build's autopilot-driven real dispatch) are confirmed live by
  reading the wrapper markdown's own guardrail text and this milestone's own "both lanes" framing in
  STATE.md. This is closer in spirit to Phase 189's `TestConsolidationPhaseEndDryRunDoesNotMutate`-style
  finding than a theoretical cleanup — it is currently shipping duplicated content on every interactive
  continue and every autopilot build.
- Criterion 1 and criterion 4 are, mechanically, the same fix observed from two angles: making
  `brief_path` real (criterion 1) is what makes the byte duplication provably absent (criterion 4).
  They are planned as one task, not two, because splitting them would mean writing the SAME
  `writeBuildWorkerBriefFiles`/blank-on-success change twice under different task headings.

</specifics>

<deferred>
## Deferred Ideas

- **Continue's own `brief_path` mechanism** (a parallel fix to `codexContinueExternalDispatch`,
  mirroring build's). Real future work; not what criterion 1's literal text asks for. See `<domain>`
  "Out of scope."
- **`prompt-assembler.ts`'s dead `assemblePrompt`/`hiveSection` config field.** Confirmed unused by the
  runtime path; not part of criterion 3's "double injection" (nothing duplicates through code nobody
  calls). Candidate for Phase 191, not built or flagged as a phase-190 gap.
- **A general orchestrator-relay byte-budget audit** beyond the brief-duplication fix (e.g. auditing
  `CasteRoster`/`DispatchContract` for unrelated bloat). Criterion 4 asks for a measurable drop
  attributable to THIS phase's fix, not a general audit.

</deferred>

---

## Success Criteria Coverage

| # | ROADMAP Success Criterion | Covered By | Already Satisfied? |
|---|---|---|---|
| 1 | Plan-only manifests carry `brief_path` to files on disk and the wrapper passes paths (build.md prose updated to match) | 190-01 (Task 1, Task 3) | No — confirmed unsatisfied for the actual wrapper-facing plan-only path; `BriefPath` exists only for the unrelated direct-dispatch path today |
| 2 | `--print-brief` asserts zero duplicated sections (pheromones and handoffs get one home each) | 190-01 (Task 2) for build; continue's equivalent already satisfied structurally by 189 (D-07) | Partially — continue's "one home" property already holds (189); build's `--print-brief` has no assertion mechanism yet (new work) |
| 3 | TS-host hive double-injection removed (build path and continue dry-run `hive_section`) | 190-02 (all 3 tasks) | No — confirmed live and reachable on both named paths, and on two additional mechanically-identical sites (plan real-dispatch, plan/build dry-run) fixed in the same sweep |
| 4 | Orchestrator-relay byte count measurably drops | 190-01 (Task 1's invariant test + recorded before/after in SUMMARY) | No — new invariant test; mechanically the same fix as criterion 1 |

Every one of the four ROADMAP success criteria is covered by at least one plan. Criterion 2's
continue-side "one home" property is the only already-satisfied component found — it is called out
explicitly here rather than silently re-tested or silently skipped.

---

*Phase: 190-Lean, Non-Duplicated Delivery*
*Context gathered: 2026-08-20*
