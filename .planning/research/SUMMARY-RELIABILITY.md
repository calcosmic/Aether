# Project Research Summary

**Project:** Aether v1.23 Daily Driver Reliability
**Domain:** CLI colony framework -- reliability restoration of flagship workflows after shell-to-Go migration
**Researched:** 2026-05-20
**Confidence:** HIGH

## Executive Summary

Aether v1.23 is a reliability restoration milestone, not a feature build. The Go runtime migration (v1.0-v1.5) and manifest-protocol cutover (v1.16-v1.18) were architecturally correct decisions -- Go owns state mutations, manifest protocol provides clean boundary enforcement, and ceremony rendering is deterministic. But the migration lost 3,786 lines of behavioral specification from the Classic v5.4.0 playbooks, leaving the system with 70 untested source files (35% of cmd/), 12 HIGH-severity gaps where failures silently drop user data, and a documentation-to-runtime contract that disagrees on execution modes, watcher behavior, and spawning rules.

The highest-impact work falls into two camps. First, fix the silent failure pipeline: 80+ playbook instructions use `2>/dev/null || true` which makes the learning pipeline, failure tracking, and pheromone system completely hollow -- they appear to work but silently discard all output. Second, close the test coverage gap on 12 source files that handle colony intelligence (eventbus, queen, instinct, hive, spawn), where untested failures cause data loss or state corruption. Everything else -- spawning optimization, ceremony restoration, workflow parity -- is downstream of these two foundations.

The key risk is that "everything looks fine but nothing is actually being recorded." A user can run builds, continues, and full colony lifecycles with no visible errors, while learnings are never extracted, failures are never tracked, pheromone signals are never written, and worker artifacts are silently dropped on interrupt. The mitigation is straightforward: remove error suppression, add smoke tests for data-persistence paths, and reconcile the documented execution policy with what the Go runtime actually implements.

## Key Findings

### Command Surface and Test Coverage

**Summary from STACK.md -- the test gap is the reliability gap.**

Aether has 389 Cobra-registered commands, 60 YAML wrappers per platform, and 5,075 passing tests. The surface is large but the coverage is not uniform: 70 Go source files (35% of cmd/) have no dedicated test file. These are not edge cases -- they include the event bus (6 commands, backbone of wisdom pipeline), queen system (8 commands, central colony intelligence), instinct management (5+ commands, learning pipeline), midden failure tracking (10 commands, entire subsystem untested), hive cross-colony wisdom (6 commands), and spawn tracking (10 commands, worker lifecycle).

**Core gap:**
- 12 HIGH-severity untested files covering ~67 commands where failure causes data loss or silent corruption
- 9 public utility commands with no test file (preferences, pause-colony, resume-colony, data-clean)
- 5 lifecycle wrappers delegate through TypeScript host but lack end-to-end tests through that path
- 5 pure-wrapper commands (chaos, dream, archaeology, interpret, organize) have no runtime backing -- low priority, exclude from test targets

### Behavioral Baselines and Regressions

**Summary from FEATURES.md -- what "good" looked like and what got lost.**

The Classic v5.4.0 era was the behavioral high-water mark. It had rich ceremony at every step, playbook-driven execution (5-stage build, 4-stage continue), first-class learning with hypothesis/validated/disproven lifecycle, a 7-question Oracle wizard, 15-step seal ceremony with Sage analytics and Chronicler audit, and 4-scout swarm with cross-comparison ranking.

The Go migration kept the right architecture but lost behavioral fidelity. The most critical regression: the learning extraction lifecycle in `/ant-continue`. Classic continue extracted learnings as hypotheses, tracked evidence against them, promoted only validated knowledge to instincts, and piped through the full memory pipeline. The current Go runtime's default path (fast, Go-only) is correct for daily use, but it needs verification that the hypothesis lifecycle still exists in any depth mode. If it does not, this is the single highest-priority behavioral regression -- without learning extraction, the colony accumulates zero wisdom across phases.

**What to restore (ranked by user impact):**
1. Learning extraction with hypothesis lifecycle -- verify in Go runtime, restore if missing
2. Worker context quality -- verify workers receive pheromones, skills, survey data, colony goal
3. Oracle promote-to-colony pipeline -- verify end-to-end: oracle > instincts > learnings > QUEEN.md > hive
4. Autopilot pause conditions -- verify all 10 Classic conditions implemented in Go `aether run`
5. Swarm 4-scout cross-comparison and 3-attempt architectural escalation -- verify parity
6. Seal ceremony (Sage analytics, Chronicler audit, wisdom approval) -- restore as optional ceremony

**What NOT to revert:** Go-owned state mutation (Frankenstein state proved LLMs cannot safely reconstruct JSON), manifest protocol for worker dispatch, Go ceremony rendering, golden workflow tests, loop safety, depth controls.

### Architecture Gaps: Execution Policy and Spawning

**Summary from ARCHITECTURE.md -- the documentation and runtime disagree.**

The Queen orchestration system has two partially decoupled layers that have drifted apart. CLAUDE.md describes three execution modes (fast / standard / final-review) with specific watcher and specialist behavior. The Go runtime uses `VerificationDepth` (light / standard / heavy) with different rules. The mismatch is not cosmetic: CLAUDE.md says fast and standard modes skip watcher subprocess, but the Go code requires watcher at all depths for continue flow. Either the documentation was aspirational or the implementation chose a different path. Either way, users following the documentation will have wrong expectations about what each mode does.

The spawning system (caste relevance scoring in `cmd/caste_relevance.go`) is sound in isolation but conflicts with playbook instructions in 9 identified ways. Playbooks spawn Auditor as mandatory for all continues; Go says only at heavy depth. Playbooks list Chaos in every spawn plan; Go only dispatches at full depth. Playbooks spawn Oracle and Architect for deep builds; Go's caste allowlist does not even permit them in build flow. These mismatches mean the playbooks sometimes spawn agents that the runtime would reject, or skip agents that the runtime would require.

**Key components:**
1. Caste relevance scoring -- keyword-based matching against phase descriptions, with base scores, threshold checks, and suppression rules. This is correct and well-tested.
2. Queen decision layer (gate resolution) -- classifies gate failures as hard_block / soft_block / advisory, with auto-resolve budgets and circuit breakers. This is sound.
3. Playbook-Go contract -- this is where the reliability problems live. Playbooks need to match Go behavior, not contradict it.

### Critical Pitfalls

**Summary from PITFALLS-RELIABILITY.md -- six failure modes that silently lose user work.**

1. **Frankenstein state corruption** -- LLM reconstructs full COLONY_STATE.json from stale context, mixing data from prior colonies. Prevention: every state write must use `state-mutate` with targeted jq. Audit all playbooks for raw "Write COLONY_STATE.json" instructions.

2. **Silent failure pipeline** -- 80+ playbook instructions append `2>/dev/null || true`, making the learning pipeline, midden, pheromone system, and memory capture all fail silently. Workers report success while critical side-effects never happen. This is the most insidious pitfall because nothing appears broken.

3. **Worker artifact loss on interrupt** -- Build workers complete work but if the build is interrupted before finalization, the completion JSON is never written. Completed code exists on disk but is not recorded in colony state. No recovery path exists beyond `--force` redispatch.

4. **Pending decisions leaking across colonies** -- `pending-decisions.json` is scoped by session_id but only `discuss` filters by scope. Flag commands and plan commands read all decisions including stale ones from prior colonies, incorporating irrelevant constraints into new plans.

5. **Worktree branch orphaning** -- Build waves spawn workers into git worktrees, but merge-back only runs during `continue-advance`, not `build-complete` or on interrupt. Valuable code accumulates on invisible branches (13 orphaned branches documented with real production code).

6. **Provider API errors misreported as parse errors** -- When a worker process gets an auth failure or rate limit from the provider, the finalizer reports "parse worker output: no JSON found" instead of "provider auth failure." Users waste time debugging their worker config instead of fixing API credentials.

## Implications for Roadmap

Based on the combined research, this milestone needs a "fix the foundation first" structure. The work that makes everything else reliable must come before the work that makes everything else better.

### Phase 1: Silent Pipeline Fix

**Rationale:** This is the single highest-impact fix. Currently, 80+ `|| true` patterns make the learning pipeline, failure tracking, pheromone system, and memory capture all silently non-functional. Fixing these unlocks the entire behavioral correctness chain -- without them, nothing downstream (learning extraction, colony wisdom, pheromone signals) can be verified because the underlying writes are being suppressed.

**Delivers:** All playbook CLI calls produce honest errors instead of silent drops. Midden, pheromone-write, memory-capture, and pheromone-write calls propagate failures properly.

**Addresses:** Pitfall 2 (silent failure pipeline), unblocks verification of all learning/memory features from FEATURES.md

**Avoids:** The "looks done but isn't" trap where builds appear to complete successfully while no data is persisted

**Also includes:**
- State-mutate migration: audit all playbooks for raw COLONY_STATE.json writes, replace with state-mutate calls (Pitfall 1)
- Pending decision scoping: ensure all readers filter by session scope, not just discuss.go (Pitfall 4)
- Session cleanup on init: clear stale session.json from prior colonies (Pitfall 7)
- Event array cap enforcement: move the 100-entry cap into state-mutate or add to build path (Pitfall 10)

### Phase 2: Critical Test Coverage

**Rationale:** Once the pipeline is honest, we need tests to keep it honest. The 12 HIGH-severity untested files handle data-persistence paths that can silently corrupt or lose data. Without tests, these paths regress silently. The STACK.md research identified these files and ranked them by data-loss risk.

**Delivers:** Test files for the 12 most critical untested source files (~67 commands). Estimated 60-100 new test functions.

**Addresses:** STACK.md Phase 2 recommendation, unblocks safe refactoring of spawning and learning paths

**Priority order within this phase (data-loss risk):**
1. eventbus.go -- wisdom pipeline backbone
2. queen.go -- colony intelligence core
3. instinct.go -- learning pipeline
4. midden_cmds.go -- entire failure tracking subsystem
5. hive.go + hive_search.go -- cross-colony wisdom
6. spawn.go + spawn_runs.go + spawn_track.go -- worker lifecycle
7. autopilot.go -- colony autopilot
8. flag_cmds.go, council.go, shelf_cmd.go -- secondary data paths

**Avoids:** Starting behavioral restoration work (Phase 3+) on untested code

### Phase 3: Learning Extraction Verification and Restoration

**Rationale:** With an honest pipeline and test coverage in place, we can now verify the most critical behavioral regression: the learning extraction lifecycle. Classic continue extracted learnings as hypotheses, tracked evidence, promoted validated knowledge. The current Go runtime needs verification that this lifecycle still exists at any depth. If not, it must be restored -- this is the core value proposition of the colony system (learning across phases).

**Delivers:** Verified (or restored) hypothesis/validated/disproven learning lifecycle in continue. Verified worker context injection quality (pheromones, skills, survey data, colony goal, phase description). Verified Oracle promote-to-colony pipeline end-to-end.

**Addresses:** FEATURES.md restoration targets 1-3 (learning extraction, worker context quality, oracle pipeline)

**Uses:** Test coverage from Phase 2 to prevent regressions

### Phase 4: Worker Artifact Recovery and Worktree Safety

**Rationale:** Worker artifact loss on interrupt (Pitfall 3) and worktree branch orphaning (Pitfall 5) cause real user work to silently disappear. These are harder to fix because they require new commands (build-reconcile) and behavioral changes (merge-back on build-complete), but they directly prevent data loss.

**Delivers:** `build-reconcile` command for post-interrupt recovery. Worktree merge-back moved to build-complete. Pre-build check for orphaned worktree branches. Status command reports unreconciled worker changes.

**Addresses:** Pitfall 3 (artifact loss), Pitfall 5 (worktree orphaning), Pitfall 8 (completion file race conditions)

**Also includes:** Provider error classification before JSON parsing (Pitfall 6)

### Phase 5: Execution Policy Alignment

**Rationale:** The CLAUDE.md-to-runtime mapping gap (3 documented execution modes vs 3 Go VerificationDepth values) and the 9 playbook-Go spawning mismatches are real but less urgent than data loss. Fix them after the foundation is solid. The recommendation from ARCHITECTURE.md is to update CLAUDE.md to match Go behavior (safer than changing Go to match aspirational docs) and update playbooks to match Go scoring.

**Delivers:** CLAUDE.md execution mode documentation aligned with Go runtime. Playbook spawning instructions match Go caste relevance system. Ambassador and Measurer gated on phase mode, not keyword match. Sequential continue gates parallelized where possible.

**Addresses:** ARCHITECTURE.md recommendations R1-R5

### Phase 6: Workflow Parity and Ceremony Restoration

**Rationale:** The remaining behavioral regressions (Oracle wizard richness, seal ceremony, swarm ceremony, autopilot pause conditions) are quality-of-life improvements that make Aether feel like the polished tool it was at v5.4.0. They should come last because they depend on the learning pipeline working correctly (Phase 3) and the spawning system being predictable (Phase 5).

**Delivers:** Verified autopilot pause conditions (all 10 from Classic). Verified swarm 4-scout cross-comparison and rollback. Optional Sage analytics and Chronicler audit at seal. Oracle research brief formulation step.

**Addresses:** FEATURES.md restoration targets 4-6 (Sage analytics, swarm verification, autopilot conditions)

### Phase 7: Public Utility Test Gaps and E2E Coverage

**Rationale:** After core reliability is established, address the 9 untested public utility commands (preferences, pause-colony, resume-colony, data-clean, insert-phase, quick, verify-castes, bump-version, maturity) and add end-to-end tests through the TypeScript host for the 5 lifecycle commands.

**Delivers:** Test files for 9 public utility commands. E2E tests through TS host for build, continue, seal, plan, colonize.

**Addresses:** STACK.md Phase 3-4 recommendations

### Phase Ordering Rationale

- Phases 1-2 must come first because they establish honest error handling and test coverage -- without these, every subsequent phase is built on a hollow foundation where failures are hidden
- Phase 3 is next because learning extraction is the single most valuable behavioral feature and depends on the pipeline being honest (Phase 1) and tested (Phase 2)
- Phase 4 addresses data loss on interrupt, which is the remaining critical user-impact issue
- Phase 5 fixes the documentation-runtime contract, which matters for correctness but is lower urgency than data loss
- Phase 6 restores ceremony quality-of-life after the system is reliable
- Phase 7 closes remaining test gaps once the architecture is stable

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 3 (Learning Extraction):** Need to trace the Go runtime's continue path to determine if the hypothesis lifecycle exists in any form. This requires reading `cmd/codex_continue.go` and related learning pipeline code. If the lifecycle is entirely missing, restoration is a larger effort than verification.
- **Phase 4 (Worker Artifact Recovery):** The `build-reconcile` command does not exist and needs design research -- what heuristics to use for detecting unrecorded worker changes from filesystem state, how to create a synthetic build packet, and what the UX should be.
- **Phase 5 (Execution Policy):** Need to decide whether to update CLAUDE.md to match Go (recommended, safer) or implement the CLAUDE.md intent in Go (watcher skip for light/standard). The former is a documentation change; the latter is a behavioral change that needs design and testing.

Phases with standard patterns (skip research-phase):
- **Phase 1 (Silent Pipeline Fix):** Well-understood -- grep for patterns, remove suppression, add fallback handling. No design ambiguity.
- **Phase 2 (Test Coverage):** Standard Go testing against existing source files. Test patterns established in existing 2,591 cmd/ test functions.
- **Phase 6 (Workflow Parity):** Classic v5.4.0 source is available at git tag for comparison. The "what to restore" list is specific.
- **Phase 7 (Public Utility Tests):** Standard Go testing against existing source files.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Command Surface (STACK) | HIGH | Direct `audit-catalog --json` output, file system scans, live `go test` runs |
| Behavioral Baselines (FEATURES) | HIGH | Direct git tag inspection of v5.4.0 source, milestone audit trail |
| Architecture (ARCHITECTURE) | HIGH | Direct Go source code inspection of caste_relevance.go, codex_continue.go, queen_decision.go |
| Pitfalls (PITFALLS) | HIGH | Root causes confirmed against source code and incident records; many have memory entries |

**Overall confidence:** HIGH

### Gaps to Address

- **Go runtime learning extraction status:** The research could not confirm whether the hypothesis/validated/disproven lifecycle exists in the Go runtime's continue path. This needs source code inspection of `cmd/codex_continue.go` and the learning pipeline before Phase 3 planning. If the lifecycle is entirely absent, Phase 3 scope increases significantly.
- **Worktree merge-back feasibility in build-complete:** Research identified the gap but did not assess whether moving merge-back from continue-advance to build-complete creates ordering problems (e.g., worktree branches not ready at build-complete time). Needs design validation.
- **TS host end-to-end test infrastructure:** No information on how to test through the TypeScript host layer. The Go tests and Codex tests exist, but TS host testing may require different infrastructure.
- **Quantitative impact of `|| true` removal:** Research identified 80+ instances but did not measure what percentage of builds would fail if suppression were removed immediately. Some calls may legitimately need graceful degradation. Needs per-instance analysis during Phase 1.

## Sources

### Primary (HIGH confidence)
- `cmd/caste_relevance.go` -- caste relevance registry, scoring, threshold, always-required, suppression
- `cmd/codex_continue.go` -- continue dispatches, review specs, always-required for continue flow
- `cmd/queen_decision.go` -- gate classification, recommendations, circuit breaker, budget
- `cmd/codex_build_finalize.go` -- finalizer completion contract and validation
- `cmd/finalizer_completion_contract.go` -- temp path pattern and freshness validation
- `cmd/discuss.go` -- pending decision scope filtering
- `cmd/flag_cmds.go`, `cmd/pending_decision.go` -- pending decision reads
- `cmd/recover_scanner.go` -- 7 stuck-state detectors
- `pkg/colony/colony.go` -- VerificationDepth type, normalization
- v5.4.0 git tag source code -- Classic behavioral baselines
- `.aether/docs/command-playbooks/build-full.md`, `continue-full.md`, `continue-advance.md` -- playbook source
- `.aether/docs/command-playbooks/build-wave.md`, `continue-gates.md` -- spawning instructions
- `.aether/commands/*.yaml` -- YAML wrapper definitions
- `aether audit-catalog --json` -- live CLI output
- File system scans of cmd/*.go (201 files), cmd/*_test.go (219 files)
- `go test ./... --count=1` -- 5,075 test functions, 18/18 packages passing

### Secondary (MEDIUM confidence)
- `.claude/projects/*/memory/*.md` -- incident records (state corruption, CLI flag mismatch, worktree merge gap)
- `.aether/docs/known-issues.md` -- documented known issues
- `.aether/references/contracts/queen-execution-policy-contract.md` -- design intent
- v1.10-MILESTONE-AUDIT.md, v1.12-REQUIREMENTS.md, v1.16-MILESTONE-AUDIT.md -- milestone audit trails

---
*Research completed: 2026-05-20*
*Ready for roadmap: yes*
