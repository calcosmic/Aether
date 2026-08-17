# Raw report 04 — Context Correctness Audit: what a dispatched worker actually receives

*Verbatim output of the "Audit worker context assembly" investigation, 2026-08-17, branch `oracle-reinstate`.*

## 0. The two live delivery paths

**Path A — Claude/OpenCode wrapper path (primary).** `/ant-build` → `aether build --plan-only` → `runCodexBuildPlanOnlyWithOptions` → `attachBuildDispatchContext` (cmd/codex_build.go:281, 3080) composes per-dispatch briefs → the Queen LLM spawns each worker with prompt = context_capsule + brief + skill_section, per build.md:259, 272 ("Nothing else, nothing invented").

**Path B — Go subprocess path (Codex/hosted).** `executeCodexBuildDispatches` (:1673) builds `codex.WorkerDispatch` with TaskBrief/ContextCapsule/HandoffSection/SkillSection/PheromoneSection (:1698-1720) → `AssemblePrompt` (pkg/codex/prompt.go:58) → agent TOML instructions + capsule + handoff + skill + pheromone + brief + permission profile + **Final Response Contract** (pkg/codex/worker.go:392, 807).

## 1. Brief assembly, budgets, trim order

`renderCodexBuildWorkerBrief` (:2659-2806) emits: `# Build Dispatch` header → Assignment (raw Task string) → Phase Objective → Dependencies → per-task Constraints/Hints/Task Success Criteria (:2700-2738, only if `findDispatchTask` matches) → Phase Success Criteria → Codebase Graph Context (≤2,200 chars, cmd/codegraph_context.go:12) → Territory Survey (pointers, cmd/helpers.go:225) → Phase Research (≤3,500, cmd/phase_research.go:209-238) → Colony Research (≤3,000) → Expected Output (one generic caste line, :2833) → Verification Command (one line). `composeBuildManifestBrief` (:3110-3142) appends Pheromone Signals, a git-baseline section for verifying castes, and Previous Worker Handoffs.

Budgets: capsule compact 4,000 (colony_prime_context.go:22-23; `resolveCodexWorkerContext` :1126 always compact); skills 8,000 (cmd/skills.go:126-129); derived ceiling 23,700 = 4,000 + 8,000 + 3,500 + 2,200 + 6,000 task allowance (cmd/build_print_brief.go:281-301). Subprocess path hard-trims to 24,000 (pkg/codex/prompt.go:13) order skill → pheromone → handoff → context → brief → instructions (:151-183).

**Documented trim order is stale.** CLAUDE.md's 9-step static ladder is not the mechanism; actual is score-based `RankContextCandidates` (pkg/colony/context_ranking.go:52), score = 0.20·trust + 0.20·freshness + 0.20·confirmation + 0.40·relevance (:128), relevance a static per-section constant (cmd/context_weighting.go:183) except ask mode. Seven protected sections: state, pheromones, blockers, user_preferences, clarified_intent, global_queen_md, charter (:220-239).

## 2. Per-source classification

| Source | Classification | Evidence |
|---|---|---|
| Colony goal | Injected (live), protected | colony_prime_context.go:412-413 |
| User decisions/clarified intent | Injected (live), protected | :870-897 |
| Charter | Injected (live), protected | :485-562 |
| Pheromones | Injected (live), TWICE on Path A — capsule (:565-601) AND brief (codex_build.go:3114-3124) | |
| Instincts/learnings/decisions | Injected (live), unprotected → trimmed first at 4,000 | :603-693 |
| Hive/learned memory/QUEEN.md/prefs | Injected (live) via capsule | :708-868 |
| Blockers/medic issues | Injected (live), blockers protected priority 10 | :899-968 |
| Skill injection (build) | Injected (live) both paths — :3088, :1710; skills need task/workspace evidence (skills.go:867) | |
| Skill injection (continue, Claude path) | Computed but DROPPED — manifest carries SkillSection (codex_continue_plan.go:307, 332) but continue.md says "pass brief verbatim" (:134, 168, 176); no capsule/skill/pheromone reaches Claude-path reviewers | |
| Handoffs | Injected (live) — capsule (:695-706) + per-dispatch (:3094); relevance = workflow match + phase ∈ [N−1,N] + not-self, freshness sort, cap 5 — no content relevance (codex_dispatch_contract.go:714-735) | |
| Phase research (Scout) | Injected (live), approval-gated (phase_research.go:76-107; codex_plan.go:819: unanswered batch = no research); ≤3,500 into brief | |
| Colony research (--research) | Injected (live), fail-closed on bad paths | colony_research.go:71-143 |
| Oracle research (build wave 1) | Effectively lost within the build — see finding 4 | codex_build.go:1262, 1340 |
| Per-task acceptance criteria | Injected IF the plan populated them (Route-Setter scaffold requests them, codex_plan.go:2251) | :2700-2738 |
| Expected output/handoff schema | Path B: injected (worker.go:807-835). Path A: MISSING — finding 1 | |
| Per-task codebase context | Partial — codegraph keyword match ≤2,200 (codegraph_context.go:105-120); survey is pointers; worker warm-started, still reads files itself | |

## 3. Adversarial findings

1. **The handoff contract never reaches the Claude-path worker.** Finalizer hard-rejects completed workers without a handoff (codex_build_finalize.go:1146). The response contract exists only on Path B (worker.go:831). Path A worker prompt is "capsule + brief + skill_section. Nothing else" (build.md:272); the brief never says "handoff"; zero of 27 agent definitions contain the word — aether-builder.md's return_format (:122-153) has no handoff field. The Queen must synthesize the handoff the finalizer demands — the next phase inherits the orchestrator's paraphrase, not the worker's own account. (Silent fallback also synthesizes from files_created/blockers at codex_dispatch_contract.go:791-797.)
2. **Claude-path /ant-continue reviewers get a bare brief.** SkillSection computed into the manifest, never delivered; no capsule, no pheromone section. Watcher/Auditor/Gatekeeper on the primary platform review without colony memory or skills, while the Codex path gets all of it (codex_continue.go:1413-1432, 1815-1822).
3. **Coalesced dispatches silently drop all but the first task's constraints/hints/criteria.** `mergeDispatchInto` (:1077-1091) merges only Task text and DeclaredPaths; TaskID stays the first task's; `findDispatchTask` (:2821) matches TaskID only — a merged 3-step job renders step 1's criteria and none of steps 2-3's.
4. **No intra-build context refresh.** All briefs, HandoffSections, and the capsule are composed at manifest time, before wave 1 runs (:281; capsule :1899-1902/1685). Handoffs are persisted at finalize/per-result — after sections were rendered. Wave-1 Oracle/Architect/Gatekeeper findings never enter this build's wave-3 Builder prompt; only 4 castes get a write-to-ledger instruction (findingsInjectionForCaste :1340). Pre-wave research benefits the NEXT build.
5. **The proportion guard on total context is nearly vacuous.** Brief-only test holds a 40% task-share floor (codex_build_test.go:3692-3696), but the total-assembled-context floor is 5% (context_budget_test.go:24) — scaffolding+grounding may legitimately be 95% and both tests pass. Fixture tiny (capsule=334, brief=857, skill=0, total=1,191 vs ceiling 23,700); guards never exercised near realistic mass; build.md says real briefs run 6-22KB.
6. **The capsule is not task-aware.** Relevance is a static per-section constant; the 0.40 weight never looks at the dispatch's task (question-boost only in ask mode, :995, 1047). Every worker gets the same 4,000-char capsule; unprotected task-adjacent memory is trimmed first.
7. **Path A final assembly is LLM-mediated.** Verbatim concatenation is a prose instruction (build.md:259, 272) — the exact "human-mediated handoff" failure class colony_research.go:11-18 was written to eliminate. Nothing verifies the spawned prompt contained all three parts.
8. **Agent definition vs runtime contract contradiction.** aether-builder.md:152 says builders "do NOT return `completed`" / "will be rejected"; Path B's contract explicitly offers builders completed (worker.go:812-814). Agent defs 9.8KB (builder) to 20.4KB (sage), 352KB total, mean ~13KB; builder's is mostly operational instruction — but its Output Format is out of sync with what the runtime validates.

## 4. Representative full Builder prompt (Path A)

| # | Section | Source | Approx size |
|---|---|---|---|
| 0 | aether-builder.md (platform system prompt) | .claude/agents/ant/ | 9.8KB |
| 1 | context_capsule (up to 18 ranked sections) | colony-prime | ≤4,000 |
| 2-6 | Build Dispatch header, Assignment, Objective+Deps, Constraints/Criteria (first task only if merged), Phase criteria | brief | ~600-2,200 |
| 7 | Codebase Graph Context | brief | ≤2,200 |
| 8 | Territory Survey pointers | brief | ~200-500 |
| 9-10 | Phase research / colony research | brief | ≤3,500 / ≤3,000 |
| 11 | Expected Output + Verification Command | brief | ~150 |
| 12 | Pheromone Signals (duplicate of capsule's) | brief | 0-800 |
| 13 | Previous Worker Handoffs (≤5, prior builds) | brief | 0-2,000 |
| 14 | skill_section (≤3 colony + ≤3 domain) | manifest | ≤8,000 |
| — | Final Response Contract / handoff schema | absent on this path | 0 |

Overall: the 2025-era failure (playbook scaffolding at 76.6% of the prompt) is genuinely fixed and locked (codex_build.go:2765-2777, codex_build_test.go:3649). What replaced it is structurally sound with four real holes: handoff contract invisible to Claude-path workers, skill/capsule drop on Claude-path continue, merged-task criteria loss, zero intra-build knowledge flow — plus proportion guards too weak (5% floor) to catch a regression at realistic scale.
