# Requirements — v1.25 Switch It On

**Milestone goal:** Make Aether usable daily on an inexpensive model by switching on machinery that already exists and has never run.

**How this differs from the first draft:** the original v1.25 was built on a diagnosis that six review agents refuted. It planned to delete the TypeScript host, which turned out to contain the only playbook loader, the only confidence loop, and the only worker dashboard — the exact things later phases proposed to rebuild. Every requirement below is traceable to a finding verified in code, not inferred from line counts or file names.

**Evidence standard:** a requirement may only assert a loss that was confirmed by reading or executing current code. Where a previous claim was refuted, the correction is recorded rather than quietly dropped.

---

## Corrections carried forward

These were asserted in the first draft and are now **verified false**. No work is scoped against them.

| Claim | Reality |
|---|---|
| Playbooks are orphaned; nothing loads them | Go loads 4 (`cmd/codex_build.go:894`); the TS host loads all 11 (`playbook-loader.ts:46`) |
| Commit `64c018f6` severed playbook loading | `aa0da1c9` (17 Apr) severed it. `64c018f6` (24 Apr) partially **restored** it |
| The Queen's spawn decision was switched off | `recommendQueenExecutionPolicy` / `enrichQueenExecutionPolicyWithSpawnBudget` run throughout `cmd/codex_build.go` |
| A four-mode Queen policy is compiled into Go | No such policy exists anywhere. `codexQueenExecutionPolicy` is a three-value depth. Four modes would be net-new work |
| Caste emoji and colours are suppressed | `build.md` already sets `AETHER_FORCE_COLOR=1`, which short-circuits both kill switches |
| Circuit breaker is missing | Present and wired: `cmd/fixer_dispatch.go:19`, `--circuit-breaker-threshold` default 3, escalation at `cmd/queen_decision.go:43` |
| Bayesian confidence scoring was lost | `pkg/memory/trust.go:49` — weighted linear with 60-day decay, faithfully ported |
| `oracle.md` was hollowed out (678 → 147 lines) | Not hollow. Its RALF loop moved into Go |
| Dream is one of the 27 castes | Dream is a command, not a caste. No Dream worker can be dispatched |
| Prior-colony context, council, session lifecycle, skill matching, governance detection are broken | All verified wired |
| Context Reaches Workers (CONTEXT block): "four things are genuinely disconnected from the active build path" | Half stale by the time Phase 163 executed: `survey-load`'s replacement (`resolveSurveySection`) and phase research (`resolvePhaseResearchSection`) were already reconnected to `renderCodexBuildWorkerBrief` by commits `281dd34a` and `a2c8288e` on 2026-07-26, before Phase 160 ran. Only the colony-prime context capsule and `suggest-analyze` remained genuinely disconnected. Four CONTEXT requirement lines (survey presence, research presence, the budget list, and the staged benchmark) reworded 2026-07-29 per Phase 163 D-07/D-08 to reflect this |

---

## Fail Loudly (LOUD)

*Foundation. Nothing downstream can be trusted while calls fail into `/dev/null`. Seven playbook CLI calls were executed against a freshly built binary and failed; the project's own drift test cannot see them because its regex matches only `--flags`, never positional arguments.*

- [x] **LOUD-01**: `aether survey-load "{phase_name}"` works as the playbooks call it, or the playbooks are corrected — currently `cobra.NoArgs`, so it exits 1 (`build-context.md:45`)
- [x] **LOUD-02**: `aether check-antipattern "{file_path}"` works as called — currently fails, and this is the **Gatekeeper security gate** (`continue-gates.md:113`); the real flag is `--file`
- [x] **LOUD-03**: The remaining five confirmed-broken calls are fixed: `print-next-up`, `verify-claims`, `state-checkpoint`, `generate-progress-bar`, `skill-detect`
- [x] **LOUD-04**: `cmd/cli_flag_audit_test.go` detects positional-argument drift, not only `--flag` drift, and fails on all seven of the above before they are fixed
- [x] **LOUD-05**: Every `aether` invocation across the playbooks and wrappers is verified by **execution**, not regex — a drift audit that runs the commands
- [x] **LOUD-06**: Playbook and wrapper calls no longer redirect stderr to `/dev/null` where the result is load-bearing; a failed context, survey, or gate call is visible
- [x] **LOUD-07**: Documentation that describes behaviour which does not happen is corrected: `.aether/docs/structural-learning-stack.md:227`, `CLAUDE.md:820`, and `AGENTS.md:863` all state that `/ant-continue` runs phase-end consolidation; it never has
- [x] **LOUD-08**: `cmd/unblock_cmd.go:126` stops telling users to run `/ant-unblock`, a slash command that exists on no platform — either the wrapper is added or the message points at the CLI
- [x] **LOUD-09**: Worker failure evidence survives. Debug artifacts are written on every worker failure mode — parse failure, timeout, and non-zero exit — carrying exit code, duration and provider session id; they are capped so the directory cannot grow without bound; they are pruned via `aether data-clean`; and in worktree mode they are written to the tracking root so they survive `git worktree remove`. *(Folded into Phase 160 on 2026-07-27 from the dispatch investigation; captured as decisions D-03/D-04/D-05 in `160-CONTEXT.md`. Evidence that vanishes is the quietest silent failure there is.)*

## Cheap Models By Design (MODEL)

*The milestone's headline goal, and the mechanism already exists. `colony/policies/model-routing.yaml` maps every caste to a provider and model — builders, watchers, scouts and surveyors to the cheaper tier; oracle, architect, route-setter and archaeologist to the expensive one. It has zero readers.*

- [ ] **MODEL-01**: `colony/policies/model-routing.yaml` is read at dispatch time and determines which model each caste runs on
- [ ] **MODEL-02**: The routing decision is visible — the user can see which model each worker was given and why
- [ ] **MODEL-03**: A user can change model routing by editing that YAML, with no Go rebuild
- [ ] **MODEL-04**: The six other unread policy files gain readers or are deleted: `autopilot.yaml`, `memory-rules.yaml`, `pheromone-lifecycle.yaml`, `safety-gates.yaml`, `signal-rules.yaml`, `skill-creation.yaml`. `safety-gates.yaml` having no reader is the one with teeth
- [ ] **MODEL-05**: `colony/` assets are distributed to other repositories — currently `cmd/policy_loader.go:19` resolves a **relative** path, `colony/` is not embedded, not published, and `~/.aether/system/colony` does not exist, so every policy silently falls back to compiled defaults outside the Aether repo
- [ ] **MODEL-06**: Go fallback defaults are retained wherever a policy file may be absent, so a missing `colony/` degrades rather than breaks

## Switch On Learning (LEARN)

*`pkg/memory/pipeline.go` wires Observe → Promote → Queen → Consolidate. It is constructed in exactly two places — `cmd/graph_consolidation_cmds.go:217` and `:303`, inside `consolidation-phase-end` and `consolidation-seal` — and neither subcommand is invoked by any wrapper, playbook, or Go call site. The colony described in CLAUDE.md has never learned anything.*

- [x] **LEARN-01**: `consolidation-phase-end` runs at the end of every phase, so observations become learnings and learnings become instincts
- [x] **LEARN-02**: `consolidation-seal` runs at seal — the full eight-ant pass, `instinct-decay-all`, archivist archive, scribe report. This appeared in no requirement of the first draft
- [x] **LEARN-03**: The two competing learning systems are reconciled. `pkg/learn` runs live on every continue (`cmd/codex_continue_finalize.go:482-555`); `pkg/memory` is the one the documentation describes and the one that is dormant. One is authoritative and the other is retired or explicitly subordinate
- [x] **LEARN-04**: The Hive Brain default is decided deliberately. `cmd/hive_policy.go:18` returns `hivePolicyOff` unless an environment variable is set, and `hiveRetrievalOptedIn()` (`cmd/hive.go:303`) needs a consent file that defaults to false. Either the default changes, or the documentation and any requirement depending on hive retrieval is corrected
- [x] **LEARN-05**: A worker's output demonstrably differs when colony memory is populated versus wiped — the learning loop is measured, not assumed

## Context Reaches Workers (CONTEXT)

*Four things are genuinely disconnected from the active build path, confirmed independently by two agents: the colony-prime context capsule, `survey-load`, phase research, and `suggest-analyze`. Separately, `context_capsule`, `hive_section` and `pheromone_section` exist on `internalWorkerDispatchRequest` but not on the wrapper-facing dispatch.*

- [ ] **CONTEXT-01**: Territory survey findings are demonstrably present in a build worker's actual prompt on both dispatch paths, delivered by the runtime brief renderer (reworded 2026-07-29 per Phase 163 D-07: the original named `build-context.md` and `codexBuildPlaybooks()`, both deleted in Phase 160, and the playbook corpus stays dead — pinned by `TestSurveyLoadAbsentAndUncalled`; the outcome is now delivered by `resolveSurveySection()` inside `renderCodexBuildWorkerBrief`)
- [ ] **CONTEXT-02**: `context_capsule`, `hive_section` and `pheromone_section` reach wrapper-spawned workers, sourced from `resolveCodexWorkerContext()` — not a second assembly path
- [ ] **CONTEXT-03**: Phase-scoped context is carried at **manifest** level, not duplicated per dispatch. With eight workers, a per-dispatch 8K capsule adds roughly 16K tokens to the orchestrator's own context on a milestone about running cheaply
- [ ] **CONTEXT-04**: Phase research findings are demonstrably present in a build worker's actual prompt via `resolvePhaseResearchSection()`, and a worker is told when the survey map is stale rather than grounding on fiction (reworded 2026-07-29 per Phase 163 D-07: `survey-load` was deleted in Phase 160 and is pinned absent — see `TestSurveyLoadAbsentAndUncalled`)
- [ ] **CONTEXT-05**: `suggest-analyze` actually runs during a build and its pheromone suggestions reach the user — it has never executed
- [ ] **CONTEXT-06**: The approved colony charter reaches workers. Governance detection is healthy (`cmd/init_research.go:117`, with ESLint, golangci-lint and Prettier parsers), but `cmd/colony_prime_context.go` contains zero charter or governance references — so a user can approve "TDD required, ESLint enforced" and no builder ever learns of it
- [x] **CONTEXT-07**: `aether build <n> --print-brief` (or equivalent) makes the assembled worker prompt inspectable, so context presence is checkable by a person rather than by reading Go
- [x] **CONTEXT-08**: Total context is measured and bounded. Budgets today: colony-prime 8000/4000, skills ~8000, phase research 3500, codegraph 2200 (corrected 2026-07-29 per Phase 163 D-07: the prior budget list's "playbook 7K" figure is stale — playbook injection into worker prompts was removed entirely, measured at 5,733 of 7,485 characters before removal per the code comment near `renderCodexBuildWorkerBrief`; the requirement's substance is unchanged, only the stale figure is removed). This is tested before more context is added
- [x] **CONTEXT-09**: Descoped per Phase 163 D-08 (2026-07-29): the user's decision is that testing happens through real repos in real use, not a staged before/after benchmark on an inexpensive model. Replacement evidence is the brief inspector plus the automated presence and budget tests this phase ships (`TestBuildWorkerBriefIncludesSurveyAndResearch`, `TestBuildWorkerBriefIsMostlyTask`). Real-world validation is explicitly post-phase, not abandoned

## Research Feeds Planning (RESEARCH)

*Nothing in the review refuted this, and keeping the TS host makes it easier: `.aether/ts-host/src/confidence-loop.ts` (250 lines, `maxIterations` at :84/:94/:207) is a working implementation of the target-confidence loop. Current `plan.md` contains zero references to Oracle — research is a standalone command the user must remember to run and paste in. `v5.4.0` `plan.md` Step 3.6 "Phase Domain Research" is the reference.*

- [x] **RESEARCH-01**: The Queen decides whether a phase needs research before planning it, states the decision and its reason, and the user can override either way
- [x] **RESEARCH-02**: When research is warranted it runs automatically before the plan is drafted — the user does not have to remember to invoke it
- [x] **RESEARCH-03**: Findings persist to a durable per-phase artifact and are injected into the planner's context, not pasted by hand
- [x] **RESEARCH-04**: Re-planning a phase re-researches rather than reusing stale findings, as `v5.4.0` did
- [x] **RESEARCH-05**: Territory survey context, where it exists, reaches both the research worker and the planner
- [x] **RESEARCH-06**: Research output feeds the existing decision, assumption and plan-revision model — no new planning store
- [x] **RESEARCH-07**: The existing `confidence-loop.ts` is used rather than reimplemented, and its progress is visible while it runs
- [x] **RESEARCH-08**: Depth binds a target and an iteration budget — fast 80%/4, balanced 90%/6, deep 95%/8, exhaustive 99%/12 — with an accept override to exit early
- [x] **RESEARCH-09**: The existing depth controls are reachable and explained at plan time: plan granularity (1-3 / 4-7 / 8-12 / 13-20 phases via `pkg/colony/granularity.go`), task decomposition depth, and verification depth
- [x] **RESEARCH-10**: The Queen proposes granularity and both depths with a plain-English reason rather than asking cold; the user accepts or changes them

## Core Lifecycle Commands (CMD)

*Not refuted. `build.md` opens with an ownership-split table and instructions to parse `result.manifest.dispatch_manifest` — protocol, not method. Line counts are a weak signal (`oracle.md` shrank because its loop moved into Go), so these are scoped by content rather than length.*

- [x] **CMD-01**: `init`, `plan`, `build` and `continue` wrappers carry engineering method — stage purpose, files to read, spawn choreography, synthesis instructions, stop conditions
- [x] **CMD-02**: No lifecycle wrapper instructs the model to parse an internal JSON envelope or write to a temporary manifest file as its primary job
- [x] **CMD-03**: A user reading `build.md` can describe what each stage does without opening Go source
- [x] **CMD-04**: Specialist and delight commands — `chaos`, `archaeology`, `dream`, `oracle`, `swarm`, `sage`, `colonize`, `council` — keep working unchanged
- [x] **CMD-05**: `build.md` is edited by at most one phase of this milestone, or the merge order between phases touching it is stated explicitly

## Full Colony On Demand (COLONY)

*All 27 castes are markdown and YAML; they cost nothing at rest. Dream is a command, not a caste — corrected from the first draft.*

- [ ] **COLONY-01**: All 27 castes remain available; none deleted or merged
- [ ] **COLONY-02**: Caste selection is driven by phase fit — Archaeologist before legacy changes, Chaos for hardening, Gatekeeper on auth, Sage at milestone close — observable in the dispatch log
- [ ] **COLONY-03**: A dispatch loads only the caste definitions it needs. The criterion must be able to fail: a legacy-touching phase loads Archaeologist and does not load Includer
- [ ] **COLONY-04**: Adding or editing a caste requires editing only `colony/agents/*.yaml` — which today has no Go reader at all, only `control-ts`
- [ ] **COLONY-05**: Whether Dream becomes a 28th caste or stays a command is decided and recorded *(decision-shaped)*

## Typed Control (TYPED)

*Prose-to-control-flow inference is at ten-plus sites, not the three originally scoped. The most dangerous is not `InferPhaseMode` — it is `plan_grounding.go:40`, where `isResearchPhase()` exempts a phase from the grounding gate entirely based on its name.*

- [ ] **TYPED-01**: A migration backfills `mode` on every existing phase **before** any validation requires it. `mode` is `omitempty` and `runMigrateState` never touches it, so requiring it first would hard-block every colony planned to date, including Aether's own
- [ ] **TYPED-02**: Backfill uses `InferPhaseMode` **once, at migration time**, writing the result explicitly to disk — the honest use of keyword inference, producing a durable auditable value
- [ ] **TYPED-03**: `mode` is required on newly planned phases, and the planner supplies it explicitly rather than leaving it to inference
- [ ] **TYPED-04**: `effectiveQueenPhaseMode` (`cmd/queen_spawn_budget.go:217`) stops falling back to inference — this is the only runtime override site; every other consumer already reads `phase.Mode` directly
- [ ] **TYPED-05**: `plan_grounding.go:40` no longer exempts phases from the grounding gate based on name keywords
- [ ] **TYPED-06**: `review_depth.go:153` no longer selects review depth from phase-name keywords; keyword lists become escalate-only suggestions that are stated, never silent
- [ ] **TYPED-07**: The remaining inference sites are catalogued with a decision recorded for each: `oracle_loop.go:88` and `:154`, `recovery_engine.go:46`, `codex_continue.go:3499`, `hive.go:209`, `codex_visuals.go:222`
- [ ] **TYPED-08**: A regression test proves a phase whose description contains "research" dispatches a Builder when its typed mode says production

## Your Eyes Back (SEE)

*Most of this is built. `colony-vital-signs` computes build velocity, error rate, signal health, memory pressure, colony age and overall health 0-100 — v5.4.0's `status.md` called it and nothing does now. The TS host holds `dashboard.ts`, `swarm-display.ts` and `narrator.ts`.*

- [ ] **SEE-01**: `colony-vital-signs` is wired back into `/ant-status` and surfaces colony health — this is the context-health indicator, already implemented
- [ ] **SEE-02**: The existing TS host worker dashboard and swarm display are surfaced during builds rather than reimplemented
- [ ] **SEE-03**: What "live worker panel" means on Claude Code specifically is decided and written down before it is built — the wrapper spawns via the Task tool and has no repaintable surface, so the honest answer may be a per-worker status line plus a running counter
- [ ] **SEE-04**: Lifecycle commands carry rich visual guidance — banner-framed stages, structured summaries, and an explicit "what happens next". The Go runtime already computes next-step guidance (`nextCommandFromState`, `closeoutNextCommand`, `continueNextCommandForAssessment`); zero of 60 wrappers surface it
- [ ] **SEE-05**: When a command needs a decision from the user, it appears in a visually distinct block
- [ ] **SEE-06**: Context health reaching "replace" writes a handoff, confirms it saved, and `/ant-resume` restores from it without the user reconstructing anything
- [ ] **SEE-07**: The real reason caste identity feels absent is diagnosed before any requirement is written against it — colours already render via `AETHER_FORCE_COLOR=1`, so the cause is elsewhere and currently unknown *(decision-shaped)*
- [ ] **SEE-08**: A rolling worker panel shows each worker as it spawns, runs and completes, with caste emoji, ANSI-coloured caste label and deterministic name (`🔨 Builder Mason-67`), in whatever form `SEE-03` establishes is achievable on Claude Code
- [ ] **SEE-09**: Progress is visible incrementally while a build runs, not only in a summary printed after it finishes
- [ ] **SEE-10**: The colony charter ceremony is presented with its full structure — Prior Context, Charter (Intent, Vision, Governance, Goals), Context, Pheromone suggestions — with genuine approve / edit / cancel
- [ ] **SEE-11**: Re-init preserves all colony state, wisdom, instincts, learnings, pheromones and phase progress. This is a data-safety requirement, not a ceremony one, and gets its own test
- [ ] **SEE-12**: Command output follows a progressive-disclosure standard — compact single-line summaries by default, bracketed counts marking expandable detail, verbose on request
- [ ] **SEE-13**: Worker task packets are written for a worker with zero prior context, one action per task, with explicit acceptance criteria and the exact command to verify
- [ ] **SEE-14**: Every lifecycle command ends by stating what happens next. The Go runtime already computes this (`nextCommandFromState`, `closeoutNextCommand`, `continueNextCommandForAssessment`); zero of 60 wrappers surface it

## Reclaim The Unreachable (RECLAIM)

*Of 83 subcommands that lost every caller since v5.4.0, 46 return automatically if the playbooks are reconnected. These 37 do not — no caller anywhere. These are the ones worth having back.*

- [ ] **RECLAIM-01**: `autofix-checkpoint` and `autofix-rollback` are reachable — checkpoint before risky auto-repair, roll back after. This is the missing half of state-transition checkpointing
- [ ] **RECLAIM-02**: `registry-add` runs at init and seal, populating the colony registry that supplies the domain tags used to scope hive wisdom
- [ ] **RECLAIM-03**: The XML exchange half is reachable or retired: `colony-archive-xml`, `wisdom-export-xml`, `wisdom-import-xml`, `registry-export-xml`, `registry-import-xml`. v5.4.0's `seal.md` exported a colony archive; nothing does now, while `.aether/exchange/` holds stale files and `.aether/docs/xml-utilities.md` still documents them
- [ ] **RECLAIM-04**: `recover` (`cmd/recover_scanner.go`) is reachable from a command
- [ ] **RECLAIM-05**: `build-full.md` (1,712 lines) and `continue-full.md` (1,763 lines) are deleted as duplicates of the split playbooks — 3,475 of the 9,278 playbook lines, cutting the surface by 37%
- [ ] **RECLAIM-06**: `colony/playbooks/`, `colony/phases/`, and the 27 unread files in `colony/prompts/` are each given a reader or deleted. `colony/playbooks/` is a third parallel copy of build/continue/plan choreography
- [ ] **RECLAIM-07**: The swarm state quartet is reachable — `swarm-findings-init`, `swarm-findings-add`, `swarm-solution-set`, `swarm-cleanup`. `/ant-swarm` is one of the flagship workflows and its state machine has no caller
- [ ] **RECLAIM-08**: The remaining Class A orphans are reached or retired with a reason recorded: `pheromone-count`, `data-safety-stats`, `clash-setup`, `domain-detect`, `skill-list`
- [ ] **RECLAIM-09**: `queen-seed-from-hive` (`cmd/queen.go:325`), called by `v5.4.0`'s `init.md` and now unreferenced, is reconnected or retired alongside the `LEARN-04` hive decision

## Retire What Is Genuinely Dead (RETIRE)

*Only one thing in this repository is safe to delete outright.*

- [x] **RETIRE-01**: `control-ts/` is deleted — zero Go references, zero CI references, not embedded, not published, and its own `package.json` describes it as "Retired experimental Aether control-plane prototype". Done standalone as the first commit of the milestone
- [x] **RETIRE-02**: `.aether/ts-host/` is **kept**. It holds the only playbook loader, the only confidence loop (`confidence-loop.ts`, 250 lines), and the only worker dashboard, swarm display and narrator. Deleting it also breaks `go build` outright via `//go:embed` at `embedded_assets.go:13`, and hard-fails `aether publish`, `aether integrity`, and both GitHub workflows
- [x] **RETIRE-03**: `.aether/ts/` (the narrator package) is not deleted by association with the other two — it is embedded, workflow-verified, and consumed by `cmd/narrator_launcher_test.go`
- [x] **RETIRE-04**: Any test deleted during this milestone is recorded in a ledger as dead, re-covered by a named surviving test, or knowingly uncovered. **Named explicitly: `control-ts/tests/schemas/policy.schema.test.ts` is the only thing in the repo validating `model-routing.yaml`'s schema, and `MODEL-01` makes that file load-bearing for dispatch. A replacement schema test must exist before or alongside its deletion**

## Stay Switched On (LOCK)

*The structural finding no requirement previously addressed. Build orchestration was rearchitected at least five times in eight weeks. v1.22 diagnosed and fixed playbook loading; it silently reverted and nobody noticed for nine weeks. Eleven of twenty-five milestones are named for restoring something already built. Switching machinery on is worth little if nothing keeps it on — this is the difference between an eighth repair and a last one.*

- [ ] **LOCK-01**: Every connection this milestone switches on has a regression test that fails if it is disconnected — the consolidation pipeline being invoked, `model-routing.yaml` being read, `build-context.md` being in the playbook list, `colony-vital-signs` reaching status, the charter reaching colony-prime
- [ ] **LOCK-02**: A single "wired-ness" suite runs all of the above together, so a future milestone cannot quietly un-wire one of them
- [ ] **LOCK-03**: The wired-ness suite runs in CI, not only locally
- [ ] **LOCK-04**: Documentation claims about behaviour are testable or removed — the failure mode where `structural-learning-stack.md`, `CLAUDE.md` and `AGENTS.md` all described consolidation running for months while it never ran once

## Prove It (PROOF)

*The existing benchmark cannot do the job assigned to it. `aether_bench/aether_colony.py:143-166` runs `claude --dangerously-skip-permissions --print` — the identical invocation as the solo arm — and never calls a single Aether command. It compares Claude Code to Claude Code.*

- [ ] **PROOF-01**: Three real development tasks are completed in real repositories on this machine using an inexpensive model, with operator interventions counted. This is the milestone's primary verdict
- [ ] **PROOF-02**: An interrupted task resumes in a fresh session without the user re-explaining the work
- [ ] **PROOF-03**: A full lifecycle runs in a separate downstream repo without Aether modifying itself — the case where undistributed `colony/` assets would fail
- [ ] **PROOF-04**: The `aether-bench` colony arm is either written so it actually invokes Aether, or the harness is explicitly retired. It is not described as "needing repair". **Full path: `/Users/callumcowie/- MASTER - Aether /aether-bench/src/aether_bench/aether_colony.py` — outside this repository, which is why a reviewer searching `repos/` could not find it**
- [ ] **PROOF-06**: The three tasks of `PROOF-01` are named in the roadmap **before** any code changes, so they cannot be chosen afterwards to flatter the result
- [ ] **PROOF-07**: "Operator intervention" is defined in writing before measurement begins — whether an approval prompt counts, whether `/ant-continue` counts — otherwise the number is unfalsifiable
- [ ] **PROOF-08**: The inexpensive model is pinned by exact model ID, so the verdict is reproducible
- [ ] **PROOF-09**: A pre-milestone baseline is recorded on those same three tasks against the current build, converting an anecdote into a before/after
- [ ] **PROOF-10**: Token and dollar cost is recorded alongside intervention counts — the milestone is about running cheaply and nothing currently measures cost
- [ ] **PROOF-11**: A failure threshold is pre-registered — for example, if operator interventions do not fall by at least 30%, the milestone did not achieve its goal. A verdict that cannot fail is not a verdict
- [ ] **PROOF-05**: If the benchmark runs, its result is recorded in the repository including if it is unfavourable

---

## Out of Scope

| Item | Reason |
|------|--------|
| Deleting `.aether/ts-host/` | It contains the confidence loop, playbook loader, dashboard and swarm display this milestone wants. Deleting it also breaks `go build` |
| A four-mode Queen execution policy | Does not exist and never did; it would be net-new work, not restoration |
| Collapsing the caste library | Castes cost nothing at rest |
| Codex parity as a design constraint | Best-effort adapter |
| Hive trust redesign | Shelved; `LEARN-04` only requires the default be decided and documented honestly |
| Restoring the other 54 commands to full depth | They keep working |
| A Dream caste | Dream is a command; creating a 28th caste is a separate decision |

## Deferred

Carried from v1.23, still open, not addressed here: `CATALOG-01/02`, `TEST-01/02`, `WORKFLOW-01`…`09`, `RUNTIME-01/02`.

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| LOUD-01 | Phase 160 | Complete |
| LOUD-02 | Phase 160 | Complete |
| LOUD-03 | Phase 160 | Complete |
| LOUD-04 | Phase 160 | Complete |
| LOUD-05 | Phase 160 | Complete |
| LOUD-06 | Phase 160 | Complete |
| LOUD-07 | Phase 160 | Complete |
| LOUD-08 | Phase 160 | Complete |
| LOUD-09 | Phase 160 | Complete |
| RETIRE-01 | Phase 160 | Complete |
| RETIRE-02 | Phase 160 | Complete |
| RETIRE-03 | Phase 160 | Complete |
| RETIRE-04 | Phase 160 | Complete |
| MODEL-01 | Phase 161 | Pending |
| MODEL-02 | Phase 161 | Pending |
| MODEL-03 | Phase 161 | Pending |
| MODEL-04 | Phase 161 | Pending |
| MODEL-05 | Phase 161 | Pending |
| MODEL-06 | Phase 161 | Pending |
| LEARN-01 | Phase 162 | Complete |
| LEARN-02 | Phase 162 | Complete |
| LEARN-03 | Phase 162 | Complete |
| LEARN-04 | Phase 162 | Complete |
| LEARN-05 | Phase 162 | Complete |
| CONTEXT-01 | Phase 163 | Pending |
| CONTEXT-02 | Phase 163 | Pending |
| CONTEXT-03 | Phase 163 | Pending |
| CONTEXT-04 | Phase 163 | Pending |
| CONTEXT-05 | Phase 163 | Pending |
| CONTEXT-06 | Phase 163 | Pending |
| CONTEXT-07 | Phase 163 | Complete |
| CONTEXT-08 | Phase 163 | Complete |
| CONTEXT-09 | Phase 163 | Complete |
| RESEARCH-01 | Phase 164 | Complete |
| RESEARCH-02 | Phase 164 | Complete |
| RESEARCH-03 | Phase 164 | Complete |
| RESEARCH-04 | Phase 164 | Complete |
| RESEARCH-05 | Phase 164 | Complete |
| RESEARCH-06 | Phase 164 | Complete |
| RESEARCH-07 | Phase 164 | Complete |
| RESEARCH-08 | Phase 164 | Complete |
| RESEARCH-09 | Phase 164 | Complete |
| RESEARCH-10 | Phase 164 | Complete |
| CMD-01 | Phase 165 | Complete |
| CMD-02 | Phase 165 | Complete |
| CMD-03 | Phase 165 | Complete |
| CMD-04 | Phase 165 | Complete |
| CMD-05 | Phase 165 | Complete |
| COLONY-01 | Phase 166 | Pending |
| COLONY-02 | Phase 166 | Pending |
| COLONY-03 | Phase 166 | Pending |
| COLONY-04 | Phase 166 | Pending |
| COLONY-05 | Phase 166 | Pending |
| TYPED-01 | Phase 167 | Pending |
| TYPED-02 | Phase 167 | Pending |
| TYPED-03 | Phase 167 | Pending |
| TYPED-04 | Phase 167 | Pending |
| TYPED-05 | Phase 167 | Pending |
| TYPED-06 | Phase 167 | Pending |
| TYPED-07 | Phase 167 | Pending |
| TYPED-08 | Phase 167 | Pending |
| SEE-01 | Phase 168 | Pending |
| SEE-02 | Phase 168 | Pending |
| SEE-03 | Phase 168 | Pending |
| SEE-04 | Phase 168 | Pending |
| SEE-05 | Phase 168 | Pending |
| SEE-06 | Phase 168 | Pending |
| SEE-07 | Phase 168 | Pending |
| SEE-08 | Phase 168 | Pending |
| SEE-09 | Phase 168 | Pending |
| SEE-14 | Phase 168 | Pending |
| SEE-10 | Phase 169 | Pending |
| SEE-11 | Phase 169 | Pending |
| SEE-12 | Phase 169 | Pending |
| SEE-13 | Phase 169 | Pending |
| RECLAIM-01 | Phase 170 | Pending |
| RECLAIM-02 | Phase 170 | Pending |
| RECLAIM-03 | Phase 170 | Pending |
| RECLAIM-04 | Phase 170 | Pending |
| RECLAIM-05 | Phase 170 | Pending |
| RECLAIM-06 | Phase 170 | Pending |
| RECLAIM-07 | Phase 170 | Pending |
| RECLAIM-08 | Phase 170 | Pending |
| RECLAIM-09 | Phase 170 | Pending |
| PROOF-01 | Phase 171 | Pending |
| PROOF-02 | Phase 171 | Pending |
| PROOF-03 | Phase 171 | Pending |
| PROOF-04 | Phase 171 | Pending |
| PROOF-05 | Phase 171 | Pending |

**Coverage:** 88/88 v1 requirements mapped. No orphans.

**Placement notes:**
- `RETIRE-01..04` remain grouped into Phase 160 (Fail Loudly) rather than given their own phase. `RETIRE-01` is the one piece of standalone build work (deleting `control-ts/`) and is explicitly a safe opener alongside the loud-failure fixes. `RETIRE-02`/`RETIRE-03` are "do not delete" guarantees verified as Phase 160 success criteria, not built. `RETIRE-04` (the test-deletion ledger) applies to the only deletion this milestone performs, so it lives there too.
- **RESEARCH, CMD, and COLONY are new phases** (164, 165, 166), inserted after Context Reaches Workers per the roadmap brief: RESEARCH consumes the same manifest-level context plumbing CONTEXT restores, so it follows directly; CMD documents both CONTEXT's and RESEARCH's work in the rewritten wrappers, so it follows them; COLONY has no hard dependency on either but is grouped adjacently since it's the third "restore something the review didn't refute" category.
- **SEE was split into two phases** (168 Live Visibility, 169 Charter & Standards) because 14 requirements in one phase was too broad to stay independently verifiable, and a real seam exists: live-render behavior (dashboard, panel, progress, next-step guidance — tested by watching a running command) versus document/standard conformance (charter structure, output format, task-packet format, and a data-safety test for re-init). `SEE-11` (re-init preserving colony state) lives in the Charter & Standards phase and its success criterion is explicitly a dedicated automated test, not an observation, per the roadmap brief.
- **`CMD-05` (`build.md` ownership) is resolved structurally, not just described**: Phase 165 (Core Lifecycle Commands) is the sole owner of `build.md`'s structure. Phase 160 (Fail Loudly) may only fix specific broken call arguments inside it, and merges first (165 depends on 160). Phase 168 (Live Visibility) may only append a next-step-guidance/visual layer on top of Phase 165's output — both phases' goals and success criteria state this explicitly, and Phase 165's own success criteria requires the merge order to be traceable in the commit/file history.
- **RECLAIM-07..09 folded into the existing Reclaim phase** (170), not a new phase — they are the same shape of work (subcommand reachability) as RECLAIM-01..06.
- No phase exists whose only content is not-doing-something.
- Every phase 161-170 depends on Phase 160 (shared foundation: nothing downstream is trustworthy while calls fail silently), plus the specific additional dependencies noted above where real content coupling exists (164→163, 165→163+164, 170→162). Phase 171 depends on all of Phases 160-170.

**This roadmap has been rebuilt twice.** First from the original "Working Again" draft (six revisions, 76 requirements) after six specialist review agents refuted its central diagnosis — that rebuild produced "Switch It On" at 58 requirements across 8 phases. That rebuild then over-corrected and dropped RESEARCH, CMD, and COLONY (no review agent had refuted them) and compressed SEE too far. This revision restores all of it: 88 requirements across 12 phases (160-171). See "Corrections carried forward" above for the claims that were genuinely refuted and remain excluded.

**This roadmap revision has not yet been approved by the user.**
