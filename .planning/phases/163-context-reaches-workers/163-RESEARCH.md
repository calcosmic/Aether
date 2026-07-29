# Phase 163: Context Reaches Workers - Research

**Researched:** 2026-07-29
**Domain:** Go worker-prompt assembly (two parallel dispatch paths: hosted/subprocess vs. wrapper Task-tool spawn), Claude Code hook enforcement, colony charter/governance data model, dead-command reachability
**Confidence:** HIGH (every claim below is `[VERIFIED]` against the current tree at commit `a223c047`, via direct `Read`/`grep`/`go build`, not recalled from training data, unless explicitly tagged `[ASSUMED]`)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**D-01:** Grounding, not brevity. Cheap models worked when "agents are writing the right files and they're doing the right things" via roadmaps and colonize. Restoring that grounding is the priority payload — never "about necessarily shortening anything."

**D-02:** Compact delivery forms preferred. Evaluate graph/summary representations for the survey map rather than full-text dumps. Note idle graph machinery (`pkg/codegraph`, `pkg/graph`) and existing colonize output at `.aether/data/survey/`.

**D-03:** Measurement (CONTEXT-08) is a guard, not a gate. Instrument per-section prompt sizes so every addition is honest against a budget; the phase is not a diet program.

**D-04:** Sanctioned scratch areas. Specific workflow dirs (`.aether/data/planning/`, `.aether/data/worker-debug/`, and peers the plan identifies) become declared-writable in the distributed rules. Colony state, session files, and the rest of `.aether/data/` stay protected.

**D-05:** The two folded contract bugs ship with this phase: the permission profile that says "without repository writes" while the brief orders writing phase-plan.json (`pkg/codex/permission_profile.go:96`, same class `cmd/phase_research.go:124`), and the `artifacts` sub-schema that accepts only `{}` (`pkg/codex/worker.go:921-926`).

**D-06:** Default `--print-brief` output is a sectioned checklist: each context section (task, survey map, charter, research, signals, memory) with present/absent and size, plus total against budget. A `--full` flag prints the raw assembled prompt.

**D-07:** Intent over letter. CONTEXT-01 and CONTEXT-04 are reworded to name outcomes (survey and build context demonstrably present in worker prompts via the manifest/brief path) rather than the deleted mechanisms (`codexBuildPlaybooks()` playbook loading, `survey-load`).

**D-08:** No staged benchmark. Phase verification relies on the brief inspector plus automated presence/invariant tests.

**D-09:** Charter rules plus gate check. Approved charter rules arrive in worker briefs as hard rules, AND continue verification checks the mechanically-enforceable ones.

**D-10:** Manual refresh with a loud staleness warning. No auto-refresh of the codebase map.

**D-11:** End-of-build summary. `suggest-analyze` results collect during the build and present once at the end as a tick-to-approve list; approved signals take effect from the next build. No mid-build interruptions.

### Claude's Discretion
Graph/summary format mechanics, budget numbers and thresholds, brief section ordering, staleness-warning wording, the exact scratch-dir list, where the writable-dirs declaration lives, gate-check mechanics for charter rules.

### Deferred Ideas (OUT OF SCOPE)
- Staged cheap-model benchmark fixture (D-08)
- Auto-refresh of the codebase map at phase start (D-10)
- Mid-build suggestion prompts (D-11)
- Phase 161 residue (`colony/` distribution, zero-reader policy file reckoning)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description (as reworded by this research, see Reconciliation) | Research Support |
|----|-------------|------------------|
| CONTEXT-01/04 (merge, per D-07) | Survey findings and phase research are demonstrably present in a build worker's prompt via the manifest/brief path (not the deleted playbook loader) | **Already substantially shipped.** `resolveSurveySection()` (`cmd/helpers.go:222-259`) and `resolvePhaseResearchSection()` (`cmd/phase_research.go:146-170`) are both wired into `renderCodexBuildWorkerBrief` (`cmd/codex_build.go:2321-2456`), which feeds both dispatch paths. Remaining work: staleness warning (D-10), discoverability in `--print-brief`. |
| CONTEXT-02 | Colony-prime's context capsule (state, decisions, learnings, instincts, hive wisdom, prior reviews, blockers, user preferences) reaches wrapper-spawned build workers, sourced from `resolveCodexWorkerContext()` | **Confirmed real gap.** `composeBuildManifestBrief` (`cmd/codex_build.go:2727-2748`) never calls `resolveCodexWorkerContext()`. The hosted/subprocess path already does (`cmd/codex_build.go:1400`). Pheromone section is a **partial exception** — it already reaches the wrapper path via a narrower resolver (`resolvePheromoneSection`), added in commit `281dd34a` — this corrects the requirement's premise. |
| CONTEXT-03 | Phase-scoped context is carried at manifest level, once, not duplicated per dispatch | **Partially already true** for the hosted path (`capsule := resolveCodexWorkerContext()` computed once at `cmd/codex_build.go:1400`, reused per dispatch). **Not yet true** for the wrapper-manifest path — no top-level field exists on `codexBuildManifest` (`cmd/codex_build.go:62-102`) for this. |
| CONTEXT-05 | `suggest-analyze` runs during a build and its suggestions reach the user | **Confirmed dead**, zero live callers anywhere (`cmd/suggest_analyze.go`, `cmd/suggest_approve.go` fully built, referenced only in the dead `.aether/docs/command-playbooks/build-context.md`). Natural seam: `runCodexBuildFinalize` (`cmd/codex_build_finalize.go:201`). |
| CONTEXT-06 | The approved charter's governance rules reach worker briefs, and continue checks the mechanically-enforceable ones | **Confirmed real gap.** `pkg/colony.Charter` (`pkg/colony/colony.go:271-279`) exists on `ColonyState` but has zero references in `cmd/colony_prime_context.go` or `cmd/codex_build.go`'s brief renderer. Gate producer pattern already exists (`checkAntiPatternGate`, `cmd/gate.go:432`, wired at `cmd/codex_continue.go:2944`) to copy. |
| CONTEXT-07 | `--print-brief` makes the assembled worker prompt inspectable | **Already exists**, but doesn't match D-06's spec. `aether build <n> --print-brief [--worker <name>]` (`cmd/codex_workflow_cmds.go:127-136`, `cmd/build_print_brief.go`) prints the full raw brief **plus** a composition table unconditionally — there is no default-checklist / `--full` toggle, and it never shows the 8000/4000-char budget. It is also undocumented in any wrapper or YAML source. |
| CONTEXT-08 | Total context is measured and bounded against real budgets | Budget stack is **not** what the requirement text says (see Reconciliation — "playbook injection 7K" is confirmed deleted). Real current budgets: colony-prime 8000/4000 (`cmd/colony_prime_context.go:335-338`), phase research 3500 (`cmd/phase_research.go`), codegraph 2200 (`cmd/codegraph_context.go:12`), skills ~8000 (CLAUDE.md §Skills System — separate budget). `TestBuildWorkerBriefIsMostlyTask` (`cmd/codex_build_test.go:3186-3221`) is the existing proportion-test model to replicate. |
| CONTEXT-09 (descoped per D-08) | — | No benchmark; validation is inspector + tests only. |
| D-04 write-guardrail fix | Sanctioned scratch dirs become writable | Real enforcement point found: `protectedHookWriteReason` (`cmd/hook_cmds.go:217-239`), a Claude-Code-only `PreToolUse` hook (`.claude/settings.json`) that blanket-blocks any Write/Edit under `/.aether/data/` with **zero carve-outs**. This is the actual mechanism that produced the 2026-07-28 M4L incident, not just prompt text. |
| D-05 contract bugs | Permission/brief/schema contradictions fixed | Both pinpointed precisely (see Architecture Patterns and Common Pitfalls). |
</phase_requirements>

## Summary

This phase's requirement text and CONTEXT.md's premises were both written against a snapshot of the codebase that is now **materially stale**, in a way that matters a great deal for planning. Between the "Working Again" draft being written and this research session, three commits landed on `main` — `281dd34a` ("the colony hears you — brief ships to build workers"), `a2c8288e` ("plan researches for real"), and `713090fc` ("clean house — one honest update story") — all dated 2026-07-26, **before** Phase 160 even began executing, and all squarely inside this phase's scope. They already:

1. Wired a real `Brief` field onto the wrapper-facing dispatch manifest (`codexBuildDispatch.Brief`, `cmd/codex_build.go:44`), composed once per dispatch via `composeBuildManifestBrief` (`cmd/codex_build.go:2727`), which both `.claude/commands/ant/build.md` and `.opencode/commands/ant/build.md` are instructed to pass **verbatim** into each Task-tool spawn.
2. Wired `resolveSurveySection()` and `resolvePhaseResearchSection()` into that same brief renderer (`renderCodexBuildWorkerBrief`, `cmd/codex_build.go:2321`) — survey and phase research **already reach build workers today**, on both dispatch paths.
3. Wired `pkg/codegraph`'s import-dependency graph into the brief (`renderCodegraphContextForText`, budget 2200 chars) — a working, already-compact "code graph" delivery, which is most of what D-02 asks for.
4. Shipped `aether build --print-brief` (`cmd/build_print_brief.go`) — a working brief inspector, satisfying most of CONTEXT-07's letter already, just not D-06's exact UX (checklist-by-default + `--full`).
5. Deleted `survey-load`, `buildPlaybooksForDispatch`, and `renderBuildPlaybookContext` as dead code — independently confirming Phase 160's later finding and this phase's own D-07 premise.

**None of this shows up in `.planning/phases/163-context-reaches-workers/163-CONTEXT.md`'s premise-correction section**, which only accounts for the playbook-loader deletion. The roadmap's own Phase 163 goal text ("Four things are confirmed disconnected... the colony-prime context capsule, `survey-load`, phase research, and `suggest-analyze`") is **half true**: `survey-load`'s replacement (survey pointer list) and phase research are connected; the colony-prime context capsule and `suggest-analyze` are still genuinely disconnected, exactly as claimed.

**Primary recommendation:** Do not plan this phase as "wire four things from scratch." Plan it as: (a) one narrow, well-understood addition — get `resolveCodexWorkerContext()`'s output onto the wrapper-manifest path, once, at manifest level, matching the pattern the hosted path already proves works; (b) one net-new but small addition — charter reaches the same path, plus a gate check; (c) two small reconnections of fully-built, zero-caller commands — `suggest-analyze` at build-finalize, and a carve-out in the real enforcement code (`protectedHookWriteReason`) for sanctioned write dirs; (d) a UX refinement of an *existing* inspector (`--print-brief`) rather than building a new one. This is a much smaller phase than its own roadmap entry implies, and the plan should say so explicitly so verification doesn't get scoped against imaginary net-new-subsystem work.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Worker prompt assembly (brief composition) | Backend / Go runtime (`cmd/codex_build.go`) | — | `renderCodexBuildWorkerBrief` + `composeBuildManifestBrief` are the single source for both dispatch paths; no wrapper or TS code computes prompt content. |
| Context capsule ranking/budgeting | Backend / Go runtime (`cmd/colony_prime_context.go`, `pkg/colony` ranking) | — | `RankContextCandidates` and the trust/scoring machinery are Go-owned; wrappers only ever receive the rendered string. |
| Wrapper-spawned worker dispatch (Task tool) | Orchestrating LLM prompt surface (`.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md`) | Backend (manifest as source of truth) | The wrapper's only job is to pass `dispatch.brief` (and, after this phase, `manifest.context_capsule`) verbatim — it must not reconstruct or summarize. |
| Real-provider hosted dispatch | Backend / Go runtime (`pkg/codex`, `executeCodexBuildDispatches`) | — | Subprocess/API path; already fully wired to `resolveCodexWorkerContext()`. |
| Write-guardrail enforcement | Backend / Go runtime (`cmd/hook_cmds.go`, invoked via `.claude/settings.json` PreToolUse hook) | Claude Code only | OpenCode has no equivalent hook registration found (`.opencode/` has `opencode.json`/`package.json`, no PreToolUse hook config) — the guardrail-evasion problem as *code-enforced* is Claude-Code-specific; OpenCode's `OPENCODE.md` asserts the same protection in prose only, unenforced. |
| Charter storage | Backend / Go runtime (`pkg/colony.ColonyState.Charter`) | — | Populated at init ceremony (`cmd/init_ceremony.go`); consumed nowhere in prompt assembly today. |
| Gate pass/fail decisions | Backend / Go runtime (`cmd/gate.go`, `cmd/codex_continue.go`) | — | `checkAntiPatternGate` is the established producer pattern to copy for a charter-compliance gate. |
| Codebase/dependency graph | Backend / Go runtime (`pkg/codegraph`) | Filesystem (`.aether/data/codebase-graph.json`) | Populated at colonize (`runCodebaseScanFromColonize`, `cmd/codegraph.go:166`); consumed by `renderCodegraphContextForText`. Distinct from `pkg/graph` (instinct/knowledge relationship graph — Phase 162 territory, not code structure; do not conflate). |

## Standard Stack

No new libraries. This phase is entirely composition of existing Go structs, functions, and CLI flags — the "Don't Hand-Roll" table below is the load-bearing section, not a stack table.

## Architecture Patterns

### The true context path, end to end (both dispatch paths)

Aether has **two separate worker-dispatch systems** sharing one brief renderer:

**Path A — Hosted/subprocess dispatch** (`executeCodexBuildDispatches`, `cmd/codex_build.go:1388-1449`), used when Aether itself spawns a provider CLI subprocess (real Codex/Claude/OpenCode-compatible API calls):
```
capsule := resolveCodexWorkerContext()          // cmd/codex_build.go:1400 — computed ONCE
pheromoneSection := resolvePheromoneSection()   // cmd/codex_build.go:1403 — computed ONCE
for each dispatch:
    codex.WorkerDispatch{
        TaskBrief:        renderCodexBuildWorkerBrief(...),  // :1420
        ContextCapsule:   capsule,                            // :1421 — reused, not recomputed
        HandoffSection:   dispatch.HandoffSection,             // :1422
        SkillSection:      resolveSkillSectionForWorkflow(...), // :1425
        PheromoneSection: pheromoneSection,                    // :1426 — reused
    }
```
This path is **fully wired**. `ContextCapsule` (colony-prime's ranked, budgeted output) already reaches every hosted worker, computed once and shared — this already satisfies CONTEXT-03's "not duplicated per dispatch" concern, just for this one path.

**Path B — Wrapper-spawned dispatch** (Claude Code / OpenCode Task-tool spawn), driven by `aether build --plan-only`'s JSON manifest:
```
attachBuildDispatchContext(root, phase, dispatches, startedAt)   // cmd/codex_build.go:2702, loop over dispatches
  → dispatches[i].PermissionProfile = codex.PermissionProfileForCaste(...)
  → dispatches[i].SkillSection      = resolveWorkerSkillAssignmentForWorkflow(...)
  → dispatches[i].HandoffSection    = renderWorkerHandoffSection(...)
  → dispatches[i].Brief             = composeBuildManifestBrief(root, phase, dispatches[i], startedAt)  // :2713

composeBuildManifestBrief (cmd/codex_build.go:2727-2748):
  = renderCodexBuildWorkerBrief(...)          // assignment, phase objective, deps, constraints, hints,
                                                //   criteria, codegraph context, survey pointer list,
                                                //   phase research, expected output
  + "## Pheromone Signals" (resolvePheromoneSection(), computed PER-DISPATCH, not shared — small cost)
  + dispatch.HandoffSection
  // Does NOT call resolveCodexWorkerContext() at all.
```
`codexBuildDispatch.Brief` (`cmd/codex_build.go:38-44`, comment: *"Brief is the fully rendered worker prompt for wrapper-spawned workers... Wrappers must inject this verbatim, never reconstruct it"*) flows into `codexBuildManifest.Dispatches` (`cmd/codex_build.go:62-102`), which `aether build --plan-only` emits as JSON, and `.claude/commands/ant/build.md` / `.opencode/commands/ant/build.md` parse and pass to each Task-tool spawn verbatim (per commit `281dd34a`'s message; confirm exact wrapper wording during planning by reading `build.md`'s dispatch-loop section directly).

**The confirmed gap:** `composeBuildManifestBrief` never calls `resolveCodexWorkerContext()`. State, decisions, phase learnings, active instincts, hive wisdom, prior reviews, blockers, user preferences, clarified intent, and medic health — everything `buildColonyPrimeOutput` (`cmd/colony_prime_context.go:334-906`) assembles and ranks — **never reaches a wrapper-spawned build worker.** This is the one piece of CONTEXT-02's premise that is still fully true.

**A naming trap for the planner:** `internalWorkerDispatchRequest` (`cmd/internal_worker_adapter.go:25-43`) has a `HiveSection` field — but it is **not** colony-prime's `hive_wisdom` section. It is merged with `SkillSection` (`joinInternalWorkerSections(request.SkillSection, request.HiveSection)`, line 384) and appears to be Hive-Brain-domain-scoped wisdom fed into skill matching, a different "hive" concept than the `## HIVE WISDOM (Cross-Colony Patterns)` block colony-prime assembles (`cmd/colony_prime_context.go:602-619`). CONTEXT-02's original text treats these as the same thing; they are not. `[ASSUMED]` — the exact provenance of `HiveSection` in `internalWorkerDispatchRequest` was not traced to its caller in this session; the planner should grep `request.HiveSection` assignment sites before designing around it.

### Manifest-level carriage (the concrete design for CONTEXT-03)

`codexBuildManifest` (`cmd/codex_build.go:62-102`) has no top-level context field today. The manifest is constructed at `cmd/codex_build.go:1608` (inside `runCodexBuildPlanOnlyWithOptions`, which starts at line 161). The fix that mirrors Path A's already-proven pattern:

1. Add `ContextCapsule string \`json:"context_capsule,omitempty"\`` (and, after Q5's work, a `CharterSection string`) to `codexBuildManifest`.
2. Compute `resolveCodexWorkerContext()` **once**, before the dispatch loop, at the same call site that already does this for Path A (or a shared helper) — not inside `attachBuildDispatchContext`'s per-dispatch loop.
3. Assign it to the manifest's top-level field, not to each `dispatch.Brief`.
4. Update `.claude/commands/ant/build.md` / `.opencode/commands/ant/build.md` to read `manifest.context_capsule` once and prepend it when spawning each worker (each Task-tool call still gets full grounding in its own context window — the saving is in the orchestrator's own manifest-parsing overhead, not in what an individual worker sees).

This avoids exactly the 8-workers-× -8K-capsule duplication CONTEXT-03 describes, without needing new infrastructure — it is the same "compute once, share the reference" pattern Path A already uses successfully.

**Design tension to flag for the planner:** D-06's inspector wants one view showing "capsule, hive/pheromone, survey, research, charter... total vs budget." If capsule/charter live at manifest level (not inside `dispatch.Brief`), `printWorkerBriefs` (`cmd/build_print_brief.go:27`) must synthesize manifest-level + per-dispatch content together for *display* purposes, while the manifest itself keeps them separate for transport efficiency. This is a display-only concatenation, not a data-model change — call it out explicitly in the plan so a task doesn't accidentally re-embed the capsule into `Brief` and reintroduce the duplication CONTEXT-03 exists to prevent.

### Compact grounding delivery — what already exists (D-02)

Colonize already writes both narrative and compact forms to `.aether/data/survey/` — confirmed by reading the live directory: `BLUEPRINT.md` (17KB) pairs with `blueprint.json` (216 bytes, `{entry_points, frameworks, summary}`); same pattern for `chambers`/`disciplines`/`pathogens`/`provisions`/`anchors`. `resolveSurveySection()` (`cmd/helpers.go:222-259`) delivers a **pointer list of repo-relative filenames**, not file content — the worker reads the ones relevant to its task on demand. This is already the compact-delivery pattern D-02 asks for; no new compression format is needed for survey.

`pkg/codegraph` (`pkg/codegraph/codegraph.go`, 678 lines) is a working import-dependency graph — `Scan`, `FilesForTask`, `RelatedFiles`, `FormatRelatedFiles` — already wired into the build brief via `renderCodegraphContextForText` (`cmd/codegraph_context.go:14-24`, budget `codegraphWorkerContextBudgetChars = 2200`), populated automatically at colonize time (`runCodebaseScanFromColonize`, `cmd/codegraph.go:166`). This is already the "modern code-graph approach" the user pointed at — most of D-02's intent is shipped.

**Correction to CONTEXT.md's canonical_refs:** `pkg/graph/` (`pkg/graph/graph.go`, package doc: *"provides a knowledge graph layer for instinct relationships"*) is a **different system** — instinct/wisdom relationship graph for the Structural Learning Stack (Phase 162's territory), not a code-structure graph. CONTEXT.md's canonical_refs lists `pkg/codegraph/`, `pkg/graph/` together as "existing graph machinery relevant to D-02" — only `pkg/codegraph` is actually relevant here. Conflating them risks a task that tries to repurpose the instinct graph for code mapping, which is the wrong tool.

The only genuinely-missing compact-delivery target is **charter** (zero delivery today, any format).

## Reconciliation Corrections

| Source claim | What I verified | Verdict |
|---|---|---|
| ROADMAP/REQUIREMENTS: "the colony-prime context capsule, `survey-load`, phase research, and `suggest-analyze`" are all "confirmed disconnected from the active build path" | `survey-load`'s replacement (`resolveSurveySection`) and phase research (`resolvePhaseResearchSection`) are both live in `renderCodexBuildWorkerBrief` since commits `281dd34a`/`a2c8288e` (2026-07-26), which predate Phase 160's own execution. Only the colony-prime capsule and `suggest-analyze` remain genuinely disconnected. | **Materially stale — two of four are already connected.** |
| CONTEXT-02: "`context_capsule`, `hive_section` and `pheromone_section` exist on `internalWorkerDispatchRequest` but not on the wrapper-facing dispatch" | `pheromone_section` (via `resolvePheromoneSection()`, a narrower resolver than colony-prime's own pheromone section) **already reaches** the wrapper-facing `Brief` as of commit `281dd34a`. `context_capsule` genuinely does not. `hive_section` on `internalWorkerDispatchRequest` is a different "hive" concept (merged into skill matching, `cmd/internal_worker_adapter.go:384`) than colony-prime's `hive_wisdom` section — the two should not be treated as the same gap. | **Partially stale — reword to name only the capsule gap precisely.** |
| CONTEXT-08: budgets "currently stack — colony-prime 8K, skills 8K, playbook injection 7K" | Playbook injection is **confirmed removed**, with an explicit code comment recording why: *"Playbook injection removed. Measured on a real brief, it was 5,733 of 7,485 characters — 76.6% — against an assignment of 79"* (`cmd/codex_build.go:2432-2445` region). There is no third 7K stacking budget today. | **Stale — the "7K playbook" line item no longer exists; do not plan a task to remove it.** |
| CONTEXT-07: `--print-brief` "or equivalent" needs to be built | `aether build <n> --print-brief [--worker <name>]` already exists and works (`cmd/build_print_brief.go`, flags registered `cmd/codex_workflow_cmds.go:1166-1167`). | **The command exists; the remaining work is a UX refinement to match D-06's checklist/`--full` spec, plus documentation, not net-new construction.** |
| CONTEXT.md D-04: guardrail is a documentation/prompt-text problem | The actual enforcement is a Go function, `protectedHookWriteReason` (`cmd/hook_cmds.go:217-239`), invoked by a Claude-Code-only `PreToolUse` hook registered in `.claude/settings.json`. It hard-blocks any Write/Edit whose path contains `/.aether/data/`, with **zero carve-outs**, regardless of what any markdown rule file says. | **The fix must touch Go code (`protectedHookWriteReason`), not only markdown — this was under-scoped in CONTEXT.md's framing of D-04 as primarily a "distributed rules" change.** |
| CONTEXT.md D-04: implied the lockstep-update surface is large (every agent file repeats "Protected Paths") | Verified: the 27 per-caste agent files' own "Global Protected Paths" sections (`.claude/agents/ant/*.md`, `.opencode/agents/*.md`) do **not** mention `.aether/data/` at all — only `.aether/dreams/`, `.env*`, `.claude/settings.json`, `.github/workflows/`. The blanket `.aether/data/` protection claim exists only in `.claude/rules/aether-colony.md` (source `.aether/rules/aether-colony.md`) and `.opencode/OPENCODE.md`. | **The documentation lockstep surface is smaller than feared — 2-3 files, not 27+2.** |

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Context capsule for wrapper workers | A second context-assembly function for the manifest path | `resolveCodexWorkerContext()` (`cmd/colony_prime_context.go:908`), called once at manifest-build time, exactly as Path A already does at `cmd/codex_build.go:1400` | Two independent capsule-builders will drift; the whole point of `resolveCodexWorkerContext()` is to be the one source both paths share. |
| Compact codebase map | A new graph/summary format for survey or code structure | `resolveSurveySection()`'s pointer-list pattern (already compact) + `pkg/codegraph`'s existing `FormatRelatedFiles`/`renderCodegraphContextForText` (already wired, already budgeted) | Both already exist, already ship in the brief, and already match the "graph, not full-text-dump" instruction from D-02. |
| Charter delivery | A new charter-rendering subsystem | A new `colonyPrimeSection` in `buildColonyPrimeOutput` (following the exact pattern of the `state`/`instincts`/`decisions` sections already there, `cmd/colony_prime_context.go:365-399` etc.), sourced from `state.Charter.Governance` (and optionally `.Constraints`) | The section-ranking/budgeting/trust-scoring machinery already exists; charter just needs to become one more section, not a parallel system. |
| Charter compliance gate | New gate infrastructure | Copy the `checkAntiPatternGate` shape (`cmd/gate.go:432`) — a `checkCharterComplianceGate` producer wired into `runCodexContinueGates` (`cmd/codex_continue.go:2841`) the same way `checkAntiPatternGate` was wired at `:2944` | This is the second time this exact producer pattern is needed in two consecutive phases; treating it as a reusable shape (not a one-off) will save real effort. |
| Brief inspector | A new `aether inspect-brief` command | Refine the existing `printWorkerBriefs`/`renderBriefComposition` (`cmd/build_print_brief.go`) to match D-06 (default checklist, `--full` for raw text, add budget comparison) | The command, flag registration, and composition-breakdown renderer already exist and already work; this is a UX/output-mode change to an existing function, not new plumbing. |
| suggest-analyze wiring | A new suggestion-collection subsystem | Call the existing `suggest-analyze` command (or its internal logic directly) from `runCodexBuildFinalize` (`cmd/codex_build_finalize.go:201`), rendering via the existing `suggest_approve.go`/`filterActiveSuggestions` tick-to-approve machinery | Both ends (`colony.PendingSuggestion`, `pkg/colony/colony.go:286-294`, and the approve UI) are fully built; only the call site is missing. |
| Write-guardrail carve-out | A new permission/ACL system | A narrow `switch`/prefix-allowlist addition inside `protectedHookWriteReason` (`cmd/hook_cmds.go:217-239`) for the sanctioned subpaths, keeping the existing block for everything else under `.aether/data/` | This is a ~10-line change to one function with one existing test (`TestHookPreToolUseBlocksProtectedPath`, `cmd/hook_cmds_test.go:34`) to extend, not a new enforcement layer. |
| Proportion/invariant tests for CONTEXT-08 | A new test-composition framework | `splitBriefSections`/`briefOwnedSections` (`cmd/build_print_brief.go:170-201`) already exist and already power `TestBuildWorkerBriefIsMostlyTask` (`cmd/codex_build_test.go:3186-3221`, 40% floor) | Copy this exact pattern for a new budget-ceiling or scaffolding-share test rather than inventing a new measurement method. |

**Key insight:** every item above is "extend a function that already does 80% of this," matching the same pattern Phase 160's research found one phase early ("fully specified, zero readers"). This phase is not a build-a-context-pipeline phase; it is a connect-five-more-wires phase, on top of a pipeline that already connected three wires nobody told the roadmap about.

## Common Pitfalls

### Pitfall 1: Re-embedding the capsule into `dispatch.Brief` instead of the manifest top level
**What goes wrong:** A task adds `resolveCodexWorkerContext()`'s output inside `composeBuildManifestBrief` (per-dispatch), because that's the more obvious insertion point (it's literally where pheromones and handoffs are already added).
**Why it happens:** `composeBuildManifestBrief` is the function that already embeds two similar-looking sections (pheromones, handoffs) — adding a third feels consistent.
**How to avoid:** The capsule must go on `codexBuildManifest` (top level), computed once before the dispatch loop, not inside `attachBuildDispatchContext`'s per-dispatch iteration. Pheromones/handoffs are cheap (small text, genuinely per-worker-relevant for handoffs); the capsule is the literal 8K item CONTEXT-03 exists to stop duplicating.
**Warning signs:** A diff that adds a `resolveCodexWorkerContext()` call inside a `for i := range dispatches` loop.

### Pitfall 2: Treating `--print-brief`'s existing output as already satisfying D-06
**What goes wrong:** A task marks CONTEXT-07/D-06 done because `--print-brief` runs and prints something.
**Why it happens:** The command name and rough purpose match; the actual output shape (unconditional full-text dump, no budget line, no default-checklist mode) does not match D-06's explicit spec.
**How to avoid:** Verify against D-06's literal text: default = checklist (section name, present/absent, size), total vs budget, `--full` flag for raw text. `renderBriefComposition` today shows size and % of the brief's own total, but never against `budget := 8000`/`4000`, and there's no toggle — it always prints raw text too.
**Warning signs:** A verification step that only checks `--print-brief` exits 0 and prints non-empty output.

### Pitfall 3: Assuming `.opencode/`'s write-guardrail needs the same code fix as Claude Code's
**What goes wrong:** A task modifies `protectedHookWriteReason` and calls D-04 complete for both platforms.
**Why it happens:** The known-issues.md incident and CONTEXT.md's framing don't distinguish platforms.
**How to avoid:** `protectedHookWriteReason` is invoked only via `.claude/settings.json`'s `PreToolUse` hook (`aether hook-pre-tool-use`). No equivalent hook registration was found under `.opencode/` in this session (only `opencode.json`/`package.json`/`node_modules` present). OpenCode's protection is prose-only (`OPENCODE.md` lines 15, 84) — fixing the Go hook function does not, by itself, fix or need to fix anything OpenCode-side beyond the same doc-text correction every other rule file needs. `[ASSUMED]` this asymmetry is intentional platform-capability difference, not an oversight — confirm during planning by checking whether OpenCode has any hook-equivalent config format at all.
**Warning signs:** A task titled "fix write guardrail" that touches only `.claude/settings.json`-adjacent code but claims to fix cross-platform behavior.

### Pitfall 4: Scoping the scout permission fix (D-05) as a brief-text change instead of a permission-profile change
**What goes wrong:** A task edits `renderPhaseResearchBrief` to stop telling the scout to write a file (removing the instruction, not the enforcement contradiction), calling it a fix.
**Why it happens:** It's the smaller diff, and phase research still needs to persist to disk somehow.
**How to avoid:** The real bug is that `scout` is in `repositoryReadOnlyCastes` (`pkg/codex/permission_profile.go:55-58`), which every other caste that legitimately writes scoped artifacts (surveyors) is *not* — surveyors instead get the default `PermissionWorkspaceWrite` profile plus a behavioral-restriction string. The fix should make scout consistent with that existing pattern (remove it from `repositoryReadOnlyCastes`, or give it a new scoped-write profile), not just soften the prose telling it to write.
**Warning signs:** A diff that touches `cmd/phase_research.go`'s brief text but not `pkg/codex/permission_profile.go`.

### Pitfall 5: Building a new "wired-ness" test for CONTEXT-01/04 that re-asserts something already proven
**What goes wrong:** Effort goes into proving survey/research reach the brief (already true, already has coverage via `TestBuildWorkerBriefIncludesCodegraphContext` and friends in `cmd/codex_build_test.go`), while the capsule gap (genuinely untested) ships without one.
**Why it happens:** CONTEXT-01/04 read as the headline items in the requirement text; the capsule gap is only implicit in CONTEXT-02.
**How to avoid:** Check `cmd/codex_build_test.go` for existing survey/research presence tests before writing new ones (search for `resolveSurveySection`/`resolvePhaseResearchSection`/`TestBuildWorkerBrief*`); spend the new-test budget on the capsule/manifest-level assertion and the charter assertion, which have zero coverage today.

## Code Examples

### Existing capsule sharing pattern to replicate for the manifest path (Path A, already correct)
```go
// cmd/codex_build.go:1400-1403 — compute once, share across all dispatches
capsule := resolveCodexWorkerContext()
cleanupStaleWorkersBeforeDispatch(root)
pheromoneSection := resolvePheromoneSection()
// ... later, inside the per-dispatch loop:
workerDispatch := codex.WorkerDispatch{
    TaskBrief:        renderCodexBuildWorkerBrief(root, phase, dispatch, startedAt),
    ContextCapsule:   capsule,           // reused, not recomputed
    PheromoneSection: pheromoneSection,  // reused, not recomputed
    ...
}
```

### Existing gate producer pattern to replicate for charter compliance
```go
// cmd/gate.go:432 — the exact shape a checkCharterComplianceGate should follow
func checkAntiPatternGate(files []string) (gateCheck, gateCheck) {
    findingsCheck := gateCheck{Name: "anti_pattern"}
    executedCheck := gateCheck{Name: "anti_pattern_executed"}
    if store == nil {
        executedCheck.Passed = false
        executedCheck.Detail = "antipattern scan could not execute: no store initialized, colony root unresolvable"
        executedCheck.FixHint = gateRecoveryTemplate("anti_pattern")
        ...
    }
    ...
}
// Wired at cmd/codex_continue.go:2944:
// antiPatternCheck, antiPatternExecutedCheck := checkAntiPatternGate(verification.Claims.ScannedFiles)
```

### The exact confirmed contradiction for D-05 (scout permission vs. phase-research brief)
```go
// pkg/codex/permission_profile.go:55-58 — scout is hard read-only
var repositoryReadOnlyCastes = map[string]struct{}{
    "includer": {},
    "scout":    {},
}
// pkg/codex/permission_profile.go:65-73 — PermissionProfileForCaste("scout") returns:
//   Filesystem: FilesystemRepositoryReadOnly

// cmd/phase_research.go:70-77 — plannedPhaseResearchDispatches dispatches a "scout":
dispatches = append(dispatches, codexPlanningDispatch{
    Caste:     "scout",
    AgentName: "aether-scout",
    ...
})
// cmd/phase_research.go:124 — renderPhaseResearchBrief, sent to that same scout:
b.WriteString(fmt.Sprintf("Write your findings to `.aether/data/phase-research/phase-%d-research.md` with exactly these six sections:\n\n", candidate.ID))
```
Contrast with the surveyor pattern, which is internally consistent and should be the template for the fix:
```go
// pkg/codex/permission_profile.go:92-93 — surveyors are NOT in repositoryReadOnlyCastes,
// so they get the default PermissionWorkspaceWrite profile, scoped by behavioral restriction:
case "surveyor_disciplines", "surveyor_nest", "surveyor_pathogens", "surveyor_provisions":
    return []string{"write survey artifacts under .aether/data/survey only"}
```

### The exact confirmed enforcement gap for D-04
```go
// cmd/hook_cmds.go:217-239 — the ACTUAL code that blocked the M4L worker, not prompt text
func protectedHookWriteReason(target, cwd string) string {
    normalized := normalizeHookPath(target, cwd)
    ...
    slash := filepath.ToSlash(normalized)
    switch {
    case strings.Contains(slash, "/.aether/data/"):
        return "Protected colony state path. Update `.aether/data/*` through the `aether` CLI, not direct edits."
    // no carve-outs for /.aether/data/planning/, /.aether/data/survey/, /.aether/data/phase-research/
    ...
    }
}
```
Registered via `.claude/settings.json`:
```json
"PreToolUse": [{ "matcher": "Write|Edit", "hooks": [{ "type": "command", "command": "aether hook-pre-tool-use" }] }]
```
Existing test to extend, not replace: `TestHookPreToolUseBlocksProtectedPath` (`cmd/hook_cmds_test.go:34-64`) asserts `.aether/data/COLONY_STATE.json` is blocked (must stay blocked) — a companion test should assert `.aether/data/planning/phase-plan.json` is *not* blocked after the fix.

### The proportion-test model to replicate for CONTEXT-08
```go
// cmd/codex_build_test.go:3186-3221 — TestBuildWorkerBriefIsMostlyTask
brief := renderCodexBuildWorkerBrief(tmpDir, phase, dispatch, time.Now())
taskChars := 0
for _, section := range splitBriefSections(brief) {
    switch section.Name {
    case "Assignment", "Phase Objective", "Phase Success Criteria",
        "Task Success Criteria", "Dependencies", "Task Constraints", "Hints":
        taskChars += section.Chars
    }
}
share := float64(taskChars) / float64(len(brief)) * 100
if share < 40 {
    t.Errorf("task-relevant content is %.1f%% of the worker brief...", share)
}
```
A CONTEXT-08 test should follow this exact shape but assert a **ceiling** (e.g., total assembled context — brief + manifest-level capsule, for display purposes — stays under some multiple of the 8000-char colony-prime budget) rather than a floor, using the same `splitBriefSections`/`briefOwnedSections` machinery.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Build's plan-only manifest carried no worker brief at all — colonize, plan, and heavy-continue did, build didn't | `codexBuildDispatch.Brief` composed via `composeBuildManifestBrief`, verbatim-injected by both wrapper platforms | Commit `281dd34a`, 2026-07-26 (before Phase 160 execution began) | The roadmap's Phase 163 goal text describing this as still-broken is stale; re-verify before writing tasks against it. |
| Playbook markdown (`.aether/docs/command-playbooks/*.md`) was the described source of phase research/survey choreography | Survey and phase research are delivered as dedicated Go-rendered brief sections, independent of any playbook file | Same commit window, `a2c8288e`/`713090fc`, 2026-07-26 | `survey-load` and the playbook loader's deletion (Phase 160's LOUD-01/RETIRE finding) is not a regression to fix — the *replacement* mechanism was already built in parallel. |
| Playbook text was injected into every worker brief (up to 76.6% of a real brief's characters, per the code comment at `cmd/codex_build.go` near `renderCodexBuildWorkerBrief`) | Playbook injection removed entirely; orchestration lives in the host-manifest flow | Same window | CONTEXT-08's cited "playbook injection 7K" budget line item no longer exists — don't plan a task to remove it. |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `internalWorkerDispatchRequest.HiveSection` is unrelated to colony-prime's `hive_wisdom` section and instead feeds skill matching | Architecture Patterns | If wrong, a planned task might skip adding hive wisdom to the manifest capsule believing `HiveSection` already covers it, leaving hive wisdom absent from wrapper-spawned workers. Low-cost to verify: grep `request.HiveSection =` assignment sites before planning. |
| A2 | OpenCode has no PreToolUse-hook-equivalent enforcement mechanism, so D-04's Go-code fix is Claude-Code-only in effect | Common Pitfalls | If OpenCode does have some other enforcement path not surfaced by this session's `find`/`grep` of `.opencode/`, the plan would under-scope the fix. Low risk — `.opencode/` was directly listed and contains only `package.json`/`opencode.json`/`node_modules`. |
| A3 | `.claude/commands/ant/build.md` / `.opencode/commands/ant/build.md` currently instruct passing `dispatch.brief` verbatim into each Task-tool spawn, exactly as commit `281dd34a`'s message states | Architecture Patterns | This session read the commit message and the `codexBuildDispatch.Brief` field comment but did not do a full line-by-line read of the current `build.md` dispatch-loop section. If the wrapper text has since drifted, the planner should re-read `build.md`/`continue.md`'s relevant sections directly before writing the manifest-level-capsule task. |
| A4 | The 27-agent-file "Global Protected Paths" sections are identical (word-for-word) across all 27 castes, not just the 3 sampled (scout, builder, route-setter) | Reconciliation Corrections | If some castes have a different or additional protected-paths list that does mention `.aether/data/`, the lockstep-update surface for D-04's doc-text fix is larger than stated. Medium risk — sample size was 3 of 27; recommend a full grep sweep as a planning-time verification step, not a research-time one, since it's cheap and mechanical. |

## Open Questions

1. **Exact wording currently in `.claude/commands/ant/build.md`'s dispatch-spawn instructions.**
   - What we know: the field comment and commit message both assert "pass verbatim."
   - What's unclear: whether the wrapper text already anticipates a future `manifest.context_capsule` field, or would need explicit new instruction text added.
   - Recommendation: read `build.md`'s dispatch-loop section directly as the first planning-time action before writing the manifest-level-capsule task (this is a 2-minute read, deliberately deferred to planning rather than spent here, since the Go-side design is now well-grounded and the wrapper text is a small, mechanical follow-on edit).

2. **Where charter is populated, precisely, and whether `Governance` alone is sufficient for D-09's gate.**
   - What we know: `pkg/colony.Charter` has `Governance` as a free-text string field (`pkg/colony/colony.go:274`); `cmd/init_ceremony.go:76`'s `synthesizeLaunchBrief` takes a `*colony.Charter` parameter, implying charter is assembled/approved at init.
   - What's unclear: whether `Governance` is structured enough (e.g., contains literal tool names matching `governanceDetectors`' labels like "ESLint", "golangci-lint") to mechanically cross-check against detected tooling for D-09's gate, or whether it's free prose that only a human (or an LLM reading it) could interpret.
   - Recommendation: read `cmd/init_ceremony.go` and wherever `Charter.Governance` is populated (likely from `governanceDetectors`' output directly, given `cmd/init_research.go:117`'s proximity) before scoping the gate check's mechanical-vs-prose boundary. Only build the gate for whatever subset is genuinely structured; the rest goes to the brief as hard rules only, per D-09's two-part design.

3. **Full sweep of all 27 agent files' "Global Protected Paths" sections.**
   - What we know: 3 of 27 sampled are identical and do not mention `.aether/data/`.
   - What's unclear: full consistency across all 27.
   - Recommendation: `grep -A5 "Global Protected Paths" .claude/agents/ant/*.md | sort -u` as a cheap planning-time verification step before deciding the doc-text lockstep surface is only `.aether/rules/aether-colony.md` + `.opencode/OPENCODE.md`.

## Environment Availability

Skipped — this phase is entirely in-repo Go/markdown composition work with no new external tool, service, or runtime dependency. `go build ./cmd/aether` succeeds cleanly on the current tree (verified this session, no output = success).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` package (stdlib), invoked via `go test` |
| Config file | none — standard `go test ./...` |
| Quick run command | `go test ./cmd/... -run 'TestBuildWorkerBrief\|TestHookPreToolUse\|TestCharter\|TestPrintBrief\|TestSuggestAnalyze\|TestPermissionProfile' -count=1` |
| Full suite command | `go test ./... -count=1 -race` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CONTEXT-01/04 (reworded) | Survey + phase research present in brief for both dispatch paths | unit | `go test ./cmd -run TestBuildWorkerBriefIncludesSurveyAndResearch -count=1` | ⚠️ Likely partial coverage already exists under a different test name — planner must grep `cmd/codex_build_test.go` for existing `resolveSurveySection`/`resolvePhaseResearchSection` assertions before writing a new one (Pitfall 5) |
| CONTEXT-02 | Colony-prime capsule reaches wrapper-manifest path, once, at manifest level | unit + invariant | `go test ./cmd -run TestBuildManifestCarriesContextCapsuleOnce -count=1` | ❌ Wave 0 — new test asserting `codexBuildManifest.ContextCapsule` is non-empty when colony-prime has content, AND that it appears once (manifest-level field), not once-per-dispatch inside each `Brief` |
| CONTEXT-03 | Manifest-level, not per-dispatch duplication | invariant | Same test as above, plus an explicit assertion that `dispatch.Brief` does NOT contain the capsule's marker text for N>1 dispatches | ❌ Wave 0 |
| CONTEXT-05 | `suggest-analyze` runs at build-finalize, surfaces suggestions | integration | `go test ./cmd -run TestBuildFinalizeCollectsSuggestAnalyzeResults -count=1` | ❌ Wave 0 |
| CONTEXT-06 | Charter governance rules present in brief; mechanically-checkable rules gated at continue | unit + integration | `go test ./cmd -run TestBuildWorkerBriefIncludesCharter -count=1` and `go test ./cmd -run TestContinueCharterComplianceGate -count=1` | ❌ Wave 0 |
| CONTEXT-07/D-06 | `--print-brief` default = checklist w/ budget; `--full` = raw text | unit | `go test ./cmd -run TestPrintBriefChecklistDefaultAndFullFlag -count=1` | ❌ Wave 0 — extends existing `printWorkerBriefs`, which currently has **zero** test coverage (confirmed: no `TestPrintBrief*`/`TestPrintWorkerBriefs*` found in `cmd/*_test.go`) |
| CONTEXT-08 | Total context measured, bounded, reported against real budget | invariant | `go test ./cmd -run TestAssembledContextStaysUnderBudgetCeiling -count=1` (model: `TestBuildWorkerBriefIsMostlyTask`, `cmd/codex_build_test.go:3186`) | ❌ Wave 0 |
| D-04 (write guardrail) | Sanctioned dirs writable, rest of `.aether/data/` stays blocked | unit | `go test ./cmd -run TestHookPreToolUseAllowsSanctionedScratchDirs -count=1` (companion to existing `TestHookPreToolUseBlocksProtectedPath`, `cmd/hook_cmds_test.go:34`) | ❌ Wave 0 |
| D-05 (permission/schema bugs) | Scout permission matches phase-research brief instructions; artifacts schema accepts named fields | unit | `go test ./pkg/codex -run TestScoutPermissionProfileAllowsPhaseResearchWrite -count=1` and `go test ./pkg/codex -run TestWorkerArtifactsSchemaAcceptsNamedFields -count=1` | ❌ Wave 0 |
| D-09 (charter gate) | Mechanically-checkable charter rule surfaces as gate finding when violated | integration | `go test ./cmd -run TestContinueCharterComplianceGate -count=1` (same as CONTEXT-06) | ❌ Wave 0 |
| D-10 (staleness) | Survey/codegraph staleness warning renders in brief + inspector | unit | `go test ./cmd -run TestBuildWorkerBriefWarnsOnStaleSurvey -count=1` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** the quick run command above, scoped to the specific requirement's new test(s).
- **Per wave merge:** `go test ./cmd/... ./pkg/codex/... -count=1`.
- **Phase gate:** `go test ./... -count=1 -race`, `go build ./cmd/aether`, `go vet ./...`, `aether integrity` all green before `/gsd-verify-work`.

### Wave 0 Gaps
- [ ] `cmd/build_print_brief_test.go` — new file; `printWorkerBriefs`/`renderBriefComposition` have zero existing coverage today
- [ ] `cmd/hook_cmds_test.go` extension — sanctioned-dir carve-out companion test to the existing `TestHookPreToolUseBlocksProtectedPath`
- [ ] `cmd/codex_build_manifest_context_test.go` (name at planner's discretion) — CONTEXT-02/03 manifest-level capsule test
- [ ] `pkg/codex/permission_profile_test.go` extension — scout permission vs. phase-research brief contradiction test
- [ ] `pkg/codex/worker_test.go` extension — artifacts schema named-fields test
- [ ] `cmd/codex_build_finalize_test.go` extension — suggest-analyze end-of-build collection test
- [ ] Charter-in-brief and charter-gate tests — entirely new, no prior art beyond the `checkAntiPatternGate` shape to copy

*(No gap for CONTEXT-01/04's core mechanism — likely already covered by existing survey/codegraph tests; verify before adding, per Pitfall 5.)*

## Security Domain

`security_enforcement` is not set to `false` in `.planning/config.json` (absent = enabled), so this section is required.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V4 Access Control | yes | The scout permission-profile fix (D-05) and the write-guardrail carve-out (D-04) are both access-control corrections — narrowing/aligning what a worker process may write, not broadening it beyond what already happens in practice via workaround. |
| V5 Input Validation | yes | The `artifacts` schema fix (D-05, `pkg/codex/worker.go:921-926`) is a schema-validation fix — must add named, typed properties rather than removing validation (`additionalProperties: false` with real fields is safer than `additionalProperties: true`). |
| Secrets/credential exposure | yes (inherited from Phase 160's gate work) | The charter/capsule content now reaching worker prompts must not leak anything the existing prompt-integrity assessment (`colony.AssessPromptSource`, already run over every colony-prime section, `cmd/colony_prime_context.go:851`) wouldn't already catch — no new sanitization logic needed if charter is added as one more `colonyPrimeSection` (it inherits the existing trust/integrity pipeline for free). |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Widening the `.aether/data/` write carve-out too far (e.g., an unbounded prefix match instead of exact sanctioned subdirs) accidentally re-opens `COLONY_STATE.json` or `pheromones.json` to direct worker writes | Tampering | Implement the carve-out as an explicit allowlist of exact subpaths (`.aether/data/planning/`, `.aether/data/survey/`, `.aether/data/phase-research/`, `.aether/data/worker-debug/`), not a broadened substring match; the existing `TestHookPreToolUseBlocksProtectedPath` test must continue to pass unmodified as the negative case. |
| Loosening scout's `additionalProperties: false` artifacts schema to `true` instead of adding named fields | Tampering / Information Disclosure | Add explicit named properties (e.g. `research_file: {"type": "string"}`) rather than flipping `additionalProperties` to `true`, which would let a worker report arbitrary unvalidated JSON as "artifacts." |
| Charter content injected into worker prompts without going through the existing prompt-integrity trust assessment | Tampering / Spoofing | Add charter as a `colonyPrimeSection` (inherits `colony.AssessPromptSource` automatically) rather than string-concatenating it directly into the brief outside `buildColonyPrimeOutput`'s pipeline. |

## Sources

### Primary (HIGH confidence — direct code/file verification this session)
- `cmd/colony_prime_context.go` (full read, 918 lines) — colony-prime capsule assembly, ranking, budget
- `cmd/codex_build.go` (targeted reads: struct defs 22-140, `renderCodexBuildWorkerBrief` 2321-2456, `attachBuildDispatchContext`/`composeBuildManifestBrief` 2690-2820, `executeCodexBuildDispatches` 1388-1449, manifest construction 1608) — both dispatch paths
- `cmd/build_print_brief.go` (full read) — existing `--print-brief` implementation
- `cmd/internal_worker_adapter.go` (targeted read) — hosted/internal-adapter dispatch request schema
- `cmd/hook_cmds.go` (lines 1-280) — the real write-guardrail enforcement mechanism
- `cmd/phase_research.go` (full read) — phase research dispatch, brief, and section-resolution
- `cmd/helpers.go:222-259` — survey section resolver
- `cmd/codegraph_context.go`, `pkg/codegraph/codegraph.go` — codegraph wiring and format
- `pkg/graph/graph.go` (header + func signatures) — confirmed distinct from `pkg/codegraph`
- `pkg/codex/permission_profile.go` (full read) — permission profile model and the scout contradiction
- `pkg/codex/worker.go:900-935` — artifacts schema
- `pkg/colony/colony.go:266-294` — Charter and PendingSuggestion structs
- `cmd/init_research.go:95-140` — governance detector list
- `cmd/gate.go:422-450`, `cmd/codex_continue.go:2841-2944` — established gate producer/wiring pattern
- `cmd/suggest_analyze.go`, `cmd/suggest_approve.go` — confirmed fully built, zero live callers
- `.aether/references/contracts/protected-local-state-contract.md` (full read) — the more nuanced, existing "protected paths" reference doc, distinct from the blunt rules.md table
- `.claude/agents/ant/aether-scout.md`, `aether-builder.md`, `aether-route-setter.md` — sampled per-agent protected-paths sections
- `.claude/rules/aether-colony.md`, `.aether/rules/aether-colony.md`, `.opencode/OPENCODE.md` — the actual blunt-table lockstep surface
- `cmd/codex_build_test.go:3186-3221` (`TestBuildWorkerBriefIsMostlyTask`) — proportion-test model
- `cmd/hook_cmds_test.go:34-64` (`TestHookPreToolUseBlocksProtectedPath`) — existing enforcement test to extend
- `git log`/`git show` on commits `281dd34a`, `a2c8288e`, `713090fc`, `caa636bb`, `ecaddb7a` — the premise-correcting archaeology, all dated 2026-07-26, confirmed reachable from current `HEAD` (`a223c047`) via `git merge-base --is-ancestor`
- `go build ./cmd/aether` — confirmed clean build on current tree

### Secondary (MEDIUM confidence)
- `.opencode/` directory listing (`find .opencode -iname "*.json" -o -iname "settings*"`) — used to support the "no OpenCode hook equivalent" claim; this is a listing-based inference, not a read of OpenCode's own hook/plugin capability documentation.

### Tertiary (LOW confidence)
- None — every claim in this document traces to a direct `Read`/`grep`/`git`/`go build` invocation this session.

## Metadata

**Confidence breakdown:**
- Two-dispatch-path architecture and the capsule gap: HIGH — read both full code paths end to end, confirmed by struct field absence.
- "Already shipped" corrections (survey, research, codegraph, print-brief): HIGH — read the actual functions and their call sites, cross-checked against git history showing when they landed.
- Write-guardrail enforcement mechanism: HIGH — read the exact blocking function and its hook registration.
- D-05 permission/schema contradictions: HIGH — read both sides of each contradiction directly.
- Charter population flow (Open Question 2) and exact wrapper wording (Open Question 1): MEDIUM — deliberately deferred to planning time as cheap, mechanical verification rather than spending research budget on a 2-minute read.
- OpenCode hook-equivalent absence: MEDIUM — directory-listing-based, not exhaustive against OpenCode's own capability docs.

**Research date:** 2026-07-29
**Valid until:** ~7 days (this repo shipped three premise-shifting commits in 48 hours once already this milestone; re-verify live-vs-shipped status before executing if planning is delayed, especially by re-running the `git log`/`git show` archaeology in this document's Sources section against current `HEAD`).
