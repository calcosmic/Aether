# Data Flow & Artifact Wiring Audit

**Phase:** 103 (Data Flow & Artifact Wiring)
**Generated:** 2026-05-07
**Status:** For Phase 105 remediation

---

## 1. Severity Summary

| Severity | Count |
|----------|-------|
| Critical | 0     |
| Warning  | 2     |
| Info     | 5     |

---

## 2. Colony-Prime Section Map

Extracted from `cmd/colony_prime_context.go` `buildColonyPrimeOutput()` function. Each section is a `colonyPrimeSection` with a unique `name` field registered in the worker context assembly.

| # | Section Name | Title | Source File | Priority | Protected |
|---|-------------|-------|-------------|----------|-----------|
| 1 | `state` | Colony State | COLONY_STATE.json | 5 | yes (configurable) |
| 2 | `review_depth` | Review Depth | COLONY_STATE.json | 6 | no |
| 3 | `pheromones` | Pheromone Signals | pheromones.json | 9 | yes (configurable) |
| 4 | `instincts` | Active Instincts | instincts.json (or COLONY_STATE.json fallback) | 6 | no |
| 5 | `decisions` | Key Decisions | COLONY_STATE.json | 3 | no |
| 6 | `learnings` | Phase Learnings | COLONY_STATE.json | 2 | no |
| 7 | `worker_handoffs` | Previous Worker Handoffs | handoffs/worker-handoffs.json | 4 | no |
| 8 | `hive_wisdom` | Hive Wisdom | ~/.aether/hive/wisdom.json | 4 | no |
| 9 | `learned_memory` | Learned Memory | entries.json (via learn.ColonyStore) | 5 | no |
| 10 | `global_queen_md` | Global Queen Wisdom | ~/.aether/QUEEN.md | 5 | yes (configurable) |
| 11 | `user_preferences` | User Preferences | ~/.aether/QUEEN.md + repo QUEEN.md | 7 | yes (configurable) |
| 12 | `prior_reviews` | Prior Reviews | reviews/_summary_cache.json | 8 | no |
| 13 | `local_queen_wisdom` | Local Queen Wisdom | repo/.aether/QUEEN.md | 5 | no |
| 14 | `clarified_intent` | Clarified Intent | pending-decisions.json | 8 | yes (configurable) |
| 15 | `blockers` | Active Blockers | pending-decisions.json (or flags.json fallback) | 10 | yes (configurable) |
| 16 | `medic_health` | Colony Health Issues | medic-last-scan.json | 9 | yes (configurable) |

**Total: 16 named sections.**

---

## 3. Context Capsule Section Map

Extracted from `cmd/context.go` `buildContextCapsuleOutput()` function. The context capsule is the legacy/compact fallback path used by `resolveCodexWorkerContext()` when colony-prime produces insufficient context.

| # | Section Name | Title | Source File |
|---|-------------|-------|-------------|
| 1 | `state` | Context Capsule State | COLONY_STATE.json |
| 2 | `signals` | Active Signals | pheromones.json |
| 3 | `decisions` | Recent Decisions | COLONY_STATE.json |
| 4 | `risks` | Open Risks | flags.json or pending-decisions.json |
| 5 | `recent_narrative` | Recent Narrative | rolling-summary.log |

**Total: 5 named sections.**

---

## 4. Core Artifact Inventory (.aether/data/)

Each artifact is traced from its writer function to its reader/consumer at command + prompt section level. Verified against source code using SaveJSON/AtomicWrite/AppendJSONL/UpdateFile/UpdateJSONAtomically (writers) and LoadJSON/ReadJSONL/ReadFile/LoadRawJSON (readers).

| Artifact | Writer Functions | Reader Functions | Colony-Prime Section | Capsule Section | User-Facing CLI | Classification | Dead End? |
|----------|-----------------|------------------|---------------------|-----------------|-----------------|----------------|-----------|
| COLONY_STATE.json | init_cmd.go (initCmd), codex_plan.go (planCmd), codex_build.go (buildCmd), codex_continue.go (continueCmd), codex_continue_finalize.go (finalizeCmd), state_cmds.go (stateMutateCmd), entomb_cmd.go (entombCmd), session_flow_cmds.go (sessionCmds), state_repair.go, phase_skip.go | colony_prime_context.go (state, review_depth, decisions, learnings sections), context.go (state, decisions capsule sections), status.go (statusCmd), codex_plan.go (planCmd), codex_build.go (buildCmd), codex_continue.go (continueCmd), seal_final_review.go, entomb_cmd.go (entombCmd), recovery_snapshot.go, graph_consolidation_cmds.go, queen.go, init_cmd.go, discuss.go | state, review_depth, decisions, learnings | state, decisions | status, resume, plan | colony-prime-injected | NO |
| pheromones.json | pheromone_mgmt.go (pheromoneWriteCmd, pheromoneSyncCmd), suggest_approve.go (suggestApproveCmd), codex_build.go (buildCmd), codex_continue.go (continueCmd), seal_ceremony, entomb_cmd.go (entombCmd) | colony_prime_context.go (pheromones section), context.go (signals capsule section), suggest_analyze.go (suggestAnalyzeCmd), build_flow_cmds.go, entomb_cmd.go, pheromones_read.go (pheromoneDisplayCmd) | pheromones | signals | pheromones display | colony-prime-injected | NO |
| instincts.json | instinct_runtime.go (instinctCreateCmd), internal_cmds.go, codex_continue.go (continueCmd), codex_continue_finalize.go (finalizeCmd) | colony_prime_context.go (instincts section), queen.go (queenPromoteInstinctCmd), memory_health.go | instincts | -- | memory-details | colony-prime-injected | NO |
| pending-decisions.json | discuss.go (discussCmd), flag_cmds.go (flagCmds), pending_decision.go, assumptions.go (assumptionsCmd) | colony_prime_context.go (clarified_intent, blockers sections), context.go (risks capsule section), codex_workflow_cmds.go (checkSealBlockers) | clarified_intent, blockers | risks | flags list | colony-prime-injected | NO |
| flags.json | flag_cmds.go (flagCmds), init_ceremony.go | colony_prime_context.go (blockers fallback), context.go (risks fallback), shelf_seal.go, codex_workflow_cmds.go (checkSealBlockers), flags.go | blockers (fallback) | risks (fallback) | flags list | colony-prime-injected (legacy fallback) | NO |
| session.json | init_cmd.go (initCmd), session_cmds.go (sessionCmds), hook_cmds.go, recovery_snapshot.go | build_flow_cmds.go (session-verify-fresh), recovery_snapshot.go, hook_cmds.go, session_cmds.go (sessionCmds) | -- | -- | session display | cli-consumed | NO |
| handoffs/worker-handoffs.json | codex_dispatch_contract.go (renderWorkerHandoffSection writes handoff data) | colony_prime_context.go (worker_handoffs section) | worker_handoffs | -- | -- | colony-prime-injected | NO |
| entries.json | learn.ColonyStore via codex_continue_finalize.go (learnEntryCmd) | colony_prime_context.go (learned_memory section via learn.NewColonyStore) | learned_memory | -- | -- | colony-prime-injected | NO |
| midden.json | codex_build.go (buildCmd), codex_continue.go (continueCmd), entomb_cmd.go (entombCmd) | midden_cmds.go (middenReviewCmd, middenRecentFailuresCmd), entomb_cmd.go, context.go (context capsule midden section), immune.go | -- | midden (in context capsule) | midden-review, midden-recent-failures | capsule-injected | NO |
| event-bus.jsonl | pkg/events/bus.go (bus.Publish via AppendJSONL), ceremony_emitter.go | medic_scanner.go (scanJSONLFile), colony_prime_test.go, ceremony_emitter_test.go (tests only) | -- | -- | -- (TTL cleanup by janitor pattern) | async-pipeline | NO (async pipeline) |
| behavior-observations.jsonl | profile.go (behaviorObserveCmd via AppendJSONL) | profile.go (profile analysis commands) | -- | -- | profile command | specialized-consumer | NO |
| profile.json | profile.go (generateProfileCmd via SaveJSON) | profile.go (promote to QUEEN.md) | -- | -- | profile display | specialized-consumer | NO (promotes to QUEEN.md which is colony-prime injected) |
| rolling-summary.log | context.go (contextUpdateCmd via AtomicWrite) | context.go (extractRollingSummary via ReadFile) | -- | recent_narrative | -- | capsule-injected | NO |
| constraints.json | internal_cmds.go (writes constraints.json) | medic_scanner.go (flags as ghost file), medic_repair.go, session_cmds.go | -- | -- | -- | dead-end | YES (ghost file) |
| assumptions.json | assumptions.go (assumptionsCmd via SaveJSON) | medic_scanner.go (existence check only), no production content reader | -- | -- | assumption-list | cli-consumed | NO (user-facing CLI) |
| medic-last-scan.json | medic_auto_spawn.go (loadMedicLastScan writes via SaveJSON) | colony_prime_context.go (medic_health section), medic_auto_spawn.go | medic_health | -- | -- | colony-prime-injected | NO |
| colony.db | learn.NewSQLiteColonyStore (codex_continue_finalize.go) | hive_search.go, skill_curator.go, skill_lifecycle.go | -- | -- | skill/hive commands | specialized-consumer | NO (SQLite DB for learning search) |

### Additional Discovered Artifacts

| Artifact | Writer Functions | Reader Functions | Colony-Prime Section | Capsule Section | User-Facing CLI | Classification | Dead End? |
|----------|-----------------|------------------|---------------------|-----------------|-----------------|----------------|-----------|
| spawn-tree.txt | codex_dispatch_contract.go (build worker dispatch) | medic_scanner.go (scanJSONLFile), codex_plan_test.go | -- | -- | -- | async-pipeline | NO (transient, regenerated per run) |
| runtime-spawn-runs.jsonl | codex_dispatch_contract.go | medic_scanner.go (scanJSONLFile) | -- | -- | -- | async-pipeline | NO (transient, regenerated per run) |
| planning/ (directory) | codex_plan.go (planCmd writes plan files) | codex_plan.go (planCmd reads plans) | -- | -- | -- | specialized-consumer | NO |
| phase-research/ (directory) | codex_plan.go (research storage) | codex_plan.go, codex_build.go | -- | -- | -- | specialized-consumer | NO |

---

## 5. Survey Artifacts (.aether/data/survey/)

All survey artifacts are written by `codex_colonize.go` during the colonize command and read by `loadCodexSurveyContext()` in `codex_plan.go`. The `loadCodexSurveyContext()` function is called by `codex_plan.go` (planCmd), `discuss.go` (discussCmd), and `assumptions.go` (assumptionsCmd). The `recover_scanner.go` also reads survey files for recovery scanning.

**D-03 Wiring Verification:** Grep for "survey" and "graph" in `cmd/colony_prime_context.go` returns zero matches. Survey artifacts are NOT wired into colony-prime context injection.

| Artifact | Writer | Reader | Colony-Prime? | Classification | Dead End? |
|----------|--------|--------|---------------|----------------|-----------|
| survey/blueprint.json | codex_colonize.go (plannedSurveyors) | codex_plan.go (loadCodexSurveyContext), recover_scanner.go | NOT wired | specialized-consumer | NO |
| survey/chambers.json | codex_colonize.go (plannedSurveyors) | codex_plan.go (loadCodexSurveyContext), recover_scanner.go | NOT wired | specialized-consumer | NO |
| survey/disciplines.json | codex_colonize.go (plannedSurveyors) | codex_plan.go (loadCodexSurveyContext), recover_scanner.go | NOT wired | specialized-consumer | NO |
| survey/provisions.json | codex_colonize.go (plannedSurveyors) | codex_plan.go (loadCodexSurveyContext), recover_scanner.go | NOT wired | specialized-consumer | NO |
| survey/pathogens.json | codex_colonize.go (plannedSurveyors) | codex_plan.go (loadCodexSurveyContext), recover_scanner.go | NOT wired | specialized-consumer | NO |

---

## 6. Graph Artifacts

**D-03 Wiring Verification:** Grep for "graph" in `cmd/colony_prime_context.go` returns zero matches. Graph artifacts are NOT wired into colony-prime context injection.

| Artifact | Writer | Reader | Colony-Prime? | Classification | Dead End? |
|----------|--------|--------|---------------|----------------|-----------|
| codebase-graph.json | codegraph.go (graph-build via graph.Save), codex_colonize.go (colonize finalize) | codegraph_context.go (renderCodegraphContextForText), codegraph.go (graph-related, graph-query) | NOT wired | specialized-consumer | NO |
| instinct-graph.json | graph_consolidation_cmds.go (graph-consolidation-merge, graph-consolidation-prune, graph-consolidation-stats) | graph_consolidation_cmds.go (only), medic_scanner.go (existence check) | NOT wired | specialized-consumer | Partial -- only consumed by its own consolidation commands and medic existence check |

---

## 7. Review Artifacts (.aether/data/reviews/)

| Artifact | Writer | Reader | Colony-Prime? | Classification | Dead End? |
|----------|--------|--------|---------------|----------------|-----------|
| reviews/{domain}/ledger.json (7 domains) | review_ledger.go (review-ledger-write), seal_final_review.go, codex_workflow_cmds.go | colony_prime_context.go (buildPriorReviewsSection), status.go, codex_workflow_cmds.go, entomb_cmd.go | YES (prior_reviews) | colony-prime-injected | NO |
| reviews/_summary_cache.json | colony_prime_context.go (buildPriorReviewsSection via SaveJSON) | colony_prime_context.go (buildPriorReviewsSection via LoadJSON) | YES (internal cache for prior_reviews) | colony-prime-injected | NO |

---

## 8. Hub-Level Artifacts (~/.aether/)

| Artifact | Writer | Reader | Colony-Prime? | Classification | Dead End? |
|----------|--------|--------|---------------|----------------|-----------|
| ~/.aether/QUEEN.md (global) | queen.go (queenPromoteCmd), profile.go (profile promote to QUEEN.md) | colony_prime_context.go (global_queen_md, user_preferences sections), context.go (queen_global, user_preferences capsule sections) | YES (2 sections) | colony-prime-injected | NO |
| repo/.aether/QUEEN.md (local) | queen.go (local writes) | colony_prime_context.go (local_queen_wisdom, user_preferences sections) | YES (2 sections) | colony-prime-injected | NO |
| ~/.aether/hive/wisdom.json | hive.go (hiveStoreCmd, hivePromoteCmd) | colony_prime_context.go (hive_wisdom section via readHiveWisdomEntriesForDomains), context.go (hive_wisdom capsule section) | YES (hive_wisdom) | colony-prime-injected | NO |
| ~/.aether/registry/registry.json | registry.go (registerCmd, listCmd), install_cmd.go (installCmd), entomb_cmd.go (entombCmd) | context_weighting.go (readRegistryDomainsForRepo), registry.go (listCmd), exchange.go, entomb_cmd.go | Indirect (filters hive wisdom by domain, not injected directly) | specialized-consumer | NO |
| ~/.aether/eternal/memory.json | internal_cmds.go (eternalStoreCmd) | context_weighting.go (readHiveWisdom fallback via readEternalMemory) | YES (fallback when hive has no matching entries) | colony-prime-injected (fallback) | NO |

---

## 9. Findings

### W-01: constraints.json is a ghost file with no meaningful production reader

constraints.json is written by internal_cmds.go (pheromone predecessor commands) but the Go runtime ignores its content. The medic scanner (medic_scanner.go:549-556) explicitly detects this condition and reports "constraints.json has content but Go code ignores it (ghost file)". The only production readers are the medic scanner (which flags it) and medic_repair.go (which can clean it). No colony-prime or capsule section reads constraints data. The file persists as a legacy artifact from the pheromone predecessor system.

Affected artifacts: constraints.json

Severity: Warning because the file is written by production code but provides zero value to any downstream consumer, and its presence misleads users into thinking it has effect.

### W-02: instinct-graph.json has no meaningful consumer beyond its own commands

instinct-graph.json is written and read exclusively by graph_consolidation_cmds.go (graph-consolidation-merge, graph-consolidation-prune, graph-consolidation-stats). The medic scanner checks that the file exists but does not read its content. No colony-prime section, context capsule section, or user-facing CLI command consumes graph data from this artifact. It is not wired into build, continue, plan, or any dispatched worker context.

Affected artifacts: instinct-graph.json

Severity: Warning because the artifact is maintained by production code but the graph data never reaches any worker or user-facing output beyond its own management commands.

### I-01: Survey artifacts are NOT wired into colony-prime but have specialized consumers

All 5 survey artifacts (blueprint.json, chambers.json, disciplines.json, provisions.json, pathogens.json) are absent from colony-prime context injection. They are consumed by loadCodexSurveyContext() which is called by codex_plan.go (planCmd), discuss.go (discussCmd), and assumptions.go (assumptionsCmd). This is a specialized consumer path for planning workers, not the general worker injection path through colony-prime.

Affected artifacts: survey/blueprint.json, survey/chambers.json, survey/disciplines.json, survey/provisions.json, survey/pathogens.json

Severity: Info because the data reaches planning-phase workers through an alternative path, but it is absent from the primary colony-prime assembly.

### I-02: codebase-graph.json is NOT wired into colony-prime but has a specialized consumer

codebase-graph.json is consumed by codegraph_context.go which adds a "Codebase Graph Context" section to build worker briefs. This is a parallel injection path to colony-prime but not through the buildColonyPrimeOutput() assembly. Graph data reaches build workers through codegraph_context.go, not through the standard colony-prime section registration.

Affected artifacts: codebase-graph.json

Severity: Info because the data reaches build workers through an alternative path (codegraph_context.go), but it is absent from the primary colony-prime assembly.

### I-03: event-bus.jsonl is primarily consumed by tests and medic scanner

event-bus.jsonl is written by pkg/events/bus.go (bus.Publish) and read primarily by test files (colony_prime_test.go, ceremony_emitter_test.go, context_test.go, narrator_launcher_test.go) and medic_scanner.go (scanJSONLFile for health checking). No colony-prime section injects event data into worker context. The event bus serves as an async pipeline for event recording with TTL cleanup.

Affected artifacts: event-bus.jsonl

Severity: Info because the artifact serves a valid async pipeline purpose (event recording, health scanning) but no production worker context consumes event data.

### I-04: profile.json and behavior-observations.jsonl form a pipeline that promotes to QUEEN.md

profile.json and behavior-observations.jsonl are not directly injected into colony-prime. Instead, they form a pipeline: behavior observations are collected in behavior-observations.jsonl, analyzed by profile commands, and the results are promoted to QUEEN.md (user_preferences section) via profile.go. QUEEN.md IS colony-prime injected, so profile data reaches workers indirectly through this promotion path.

Affected artifacts: profile.json, behavior-observations.jsonl

Severity: Info because the data reaches workers through the QUEEN.md promotion pipeline, not directly.

### I-05: assumptions.json has no programmatic production reader

assumptions.json is written by assumptions.go (assumptionsCmd) and read only by medic_scanner.go (existence check) and the user-facing assumption-list CLI command. No colony-prime section, context capsule section, or dispatched worker consumes assumptions data. The file is a user-facing artifact only.

Affected artifacts: assumptions.json

Severity: Info because the artifact is consumed by a user-facing CLI command but is not part of the worker context pipeline.

---

## 10. Verified Counts

| Category | Count | Notes |
|----------|-------|-------|
| Total artifacts inventoried | 33 | Core (17) + survey (5) + graph (2) + review (2) + hub (5) + transient (2) excludes directories |
| Colony-prime injected | 12 | COLONY_STATE, pheromones, instincts, pending-decisions (clarified_intent + blockers), entries, handoffs, hive_wisdom, global_queen_md, user_preferences, prior_reviews, local_queen_wisdom, medic-last-scan |
| Capsule injected (not colony-prime) | 3 | rolling-summary.log, midden.json, flags.json (some also colony-prime) |
| CLI consumed | 4 | session.json, assumptions.json, profile.json, behavior-observations.jsonl |
| Async pipeline consumed | 3 | event-bus.jsonl, spawn-tree.txt, runtime-spawn-runs.jsonl |
| Specialized consumer (not colony-prime) | 9 | survey (5), codebase-graph.json (1), instinct-graph.json (1), colony.db (1), registry.json (1) |
| Dead-end / ghost files | 1 | constraints.json |
| Total colony-prime sections | 16 | Verified against colony_prime_context.go source |
| Total context capsule sections | 5 | Verified against context.go source |
| "NOT wired" artifacts (colony-prime) | 8 | survey (5), graph (2), instinct-graph only consumed by own commands |
| Critical findings | 0 | No critical data flow gaps found |
| Warning findings | 2 | constraints.json ghost file, instinct-graph.json limited consumer |
| Info findings | 5 | survey wiring, codebase-graph wiring, event-bus pipeline, profile pipeline, assumptions CLI-only |
