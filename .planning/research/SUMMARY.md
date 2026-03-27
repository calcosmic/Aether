# Project Research Summary

**Project:** Aether v2.3 Per-Caste Model Routing
**Domain:** Multi-agent AI development orchestration -- per-worker model selection via Claude Code native model slots
**Researched:** 2026-03-27
**Confidence:** HIGH

## Executive Summary

Aether currently runs all 22 worker agents on the same model. The v2.3 milestone aims to route reasoning-heavy castes (Prime, Oracle, Archaeologist, Route-Setter, Architect) to GLM-5 via the `opus` model slot, while execution castes (Builder, Watcher, Scout, Chaos, Colonizer) stay on GLM-5-turbo via the `sonnet` slot. A previous attempt (v1, archived 2026-02-15) failed because it relied on environment variable injection, which Claude Code's Task tool does not support for spawned subagents. Two developments since then make this feasible: (1) Claude Code's agent frontmatter `model:` field accepts slot aliases (`opus`, `sonnet`, `haiku`, `inherit`) that bypass the env var limitation entirely, and (2) the GSD system proves the Task tool also accepts a `model` parameter directly, enabling dynamic runtime routing. The recommended approach is a hybrid: use agent frontmatter for default routing (static, declarative) and the Task tool `model` parameter for runtime overrides (dynamic, profile-based). A single settings.json change (`ANTHROPIC_DEFAULT_OPUS_MODEL` from `glm-5-turbo` to `glm-5`) activates the opus-to-GLM-5 mapping chain through the existing LiteLLM proxy.

The research identifies one key architectural disagreement to resolve: STACK.md recommends Approach A (changing agent frontmatter directly), while ARCHITECTURE.md and FEATURES.md recommend Approach B (passing `model=` parameter in Task tool calls). Both are proven working. The recommendation is to start with Approach A for the MVP because it is simpler (one-word changes to 14 agent files, zero playbook changes), then layer on Approach B in a follow-up for profile-based switching. The primary risk is GLM-5 instability in agent contexts -- GLM-5 requires tight constraints (temperature 0.4, top_p 0.85, max_tokens 2500) and can loop without them. A secondary risk is that 184 hardcoded model name references across 6 test files will break on any YAML changes, so test infrastructure must be refactored first.

## Key Findings

### Recommended Stack

The implementation requires no new infrastructure, models, or environment variables. The entire routing mechanism relies on Claude Code's native model slot system, which already exists in the codebase but is unused (all 22 agents have `model: inherit`).

**Core changes:**
- **settings.json** (1 line): Change `ANTHROPIC_DEFAULT_OPUS_MODEL` from `glm-5-turbo` to `glm-5` -- activates the opus-to-GLM-5 mapping through the LiteLLM proxy. Same change in `settings.json.glm`.
- **Agent frontmatter** (14 files): Change `model: inherit` to `model: opus` for 3 reasoning agents (queen, archaeologist, route-setter) and `model: sonnet` for 11 execution agents (builder, watcher, scout, chaos, probe, weaver, ambassador, 4 surveyors). Keep 8 specialist agents on `inherit`.
- **model-profiles.yaml**: Update `worker_models` to split castes into GLM-5/GLM-5-turbo tiers. Optionally add `model_slots` section mapping castes to Claude Code slot names.
- **bin/lib/model-profiles.js**: Add `getModelSlotForCaste()` function for querying slot assignments. Add `model-slot get <caste>` subcommand to aether-utils.sh.

**What does NOT change:** No new proxy infrastructure, no new models, no new environment variables, no changes to LiteLLM proxy config, no env var workarounds.

### Expected Features

**Must have (table stakes):**
- Slot-based caste-to-model mapping -- users say "prime uses opus, builder uses sonnet" without caring about model names
- Worker spawns actually use different models -- the core mechanism that makes routing real
- Dual-mode support -- Claude native (opus=claude-opus, sonnet=claude-sonnet) and GLM proxy (opus=glm-5, sonnet=glm-5-turbo) with identical Aether routing code
- Fix workers.md incorrect claim that per-caste routing is impossible
- CLI override for per-spawn model (`/ant:build 3 --model opus`)

**Should have (competitive):**
- Per-profile caste assignments (deep/default/fast profiles like GSD's quality/balanced/budget)
- Runtime profile switching (`/ant:set-profile fast`)
- Task complexity-based auto-routing (analyze task description, route complex tasks to opus)
- Model usage tracking per phase (display in `/ant:status`)

**Defer (v2.4+):**
- Task complexity routing -- keyword matching is brittle, ship fixed assignments first
- Profiles and runtime switching -- nice-to-have once base routing works
- Usage tracking -- observability comes after functionality
- All anti-features: per-worker env var injection, model name routing in Aether code, cost-based auto-selection, health-check-gated routing

### Architecture Approach

The routing chain is: Aether model-profiles.yaml defines caste-to-slot mapping. At spawn time, the slot name (`opus` or `sonnet`) reaches Claude Code either via agent frontmatter (static) or Task tool `model` parameter (dynamic). Claude Code resolves the slot to an actual model name via `ANTHROPIC_DEFAULT_OPUS_MODEL` / `ANTHROPIC_DEFAULT_SONNET_MODEL` environment variables in settings.json. The LiteLLM proxy (when active) receives the model name and routes to the appropriate provider backend. Aether never knows or cares about actual model names -- it routes by slot, the environment resolves slots to names.

**Two implementation approaches (both proven working):**

1. **Approach A -- Agent frontmatter (static):** Change `model: inherit` to `model: opus`/`model: sonnet` in agent `.md` files. Simple, declarative, automatic. Downside: requires updating 22+ files across 3 directories (`.claude/agents/ant/`, `.aether/agents-claude/`, `.opencode/agents/`), and cannot do per-build overrides without changing agent definitions.
2. **Approach B -- Task tool `model` parameter (dynamic):** Pass `model="{slot}"` in each Task tool call, resolved from model-profiles.yaml at build time. Flexible, supports overrides and task-based routing. Downside: must update ~20 spawn points across 7 command files.

**Recommendation:** Start with Approach A for the MVP. It is the simplest path to working per-caste routing with zero playbook changes. Add Approach B later for profile-based switching.

### Critical Pitfalls

1. **Do NOT use environment variable injection.** The v1 archive proves Claude Code's Task tool does not forward env vars to spawned subagents. This is the exact failure mode that archived the previous attempt. Use the `model` parameter or agent frontmatter instead.

2. **Use Claude Code slot names, not GLM model names, in Aether config.** Aether should route by `opus`/`sonnet`/`haiku`, never by `glm-5`/`glm-5-turbo`. Model names belong in the user's environment (settings.json env vars and LiteLLM proxy config). Mixing abstractions creates coupling and breaks dual-mode support.

3. **GLM-5 can loop in agent contexts.** GLM-5 requires tight generation constraints (temperature 0.4, top_p 0.85, max_tokens 2500). If these constraints are not enforced (proxy restart without config, temperature override in settings, nested agent calls), GLM-5 workers may loop indefinitely. Add application-level loop detection and consider whether Prime should stay on GLM-5-turbo for safety despite being a "reasoning" caste.

4. **184 hardcoded model names in tests will break on any YAML change.** Six test files contain 184 occurrences of `glm-5-turbo`/`glm-5` in mock profiles. These mocks duplicate YAML data rather than reading from it. Refactor test infrastructure to centralize mocks and read from YAML before changing any model assignments.

5. **The two-model-config-file dance must stay synchronized.** Both `settings.json` and `settings.json.glm` need the `ANTHROPIC_DEFAULT_OPUS_MODEL` change. Make the change atomically and document the coordination.

## Implications for Roadmap

Based on combined research, the implementation should proceed in 5 phases ordered by dependencies and risk.

### Phase 1: Test Infrastructure Refactor
**Rationale:** 184 hardcoded model names across 6 test files create a minefield. Any YAML change before centralizing test mocks will cause cascading, hard-to-debug test failures. This must come first. PITFALLS.md identifies this as the critical ordering constraint.
**Delivers:** Centralized mock profile helper (`tests/helpers/mock-profiles.js`), all test files reading from YAML, consistency validation test
**Addresses:** Pitfall 4 (184 hardcoded model names)
**Avoids:** Wasting time debugging "expected glm-5 but got glm-5-turbo" after every config change
**Research flag:** SKIP -- well-understood refactoring, clear pattern (extract shared helper)

### Phase 2: Pre-Implementation Cleanup and Config Foundation
**Rationale:** Fix the incorrect "model routing impossible" claim in workers.md, annotate or delete the archived v1 routing files that confuse developers, and make the single settings.json change that activates the opus-to-GLM-5 mapping chain. These are all low-risk, independent changes that unblock the main work.
**Delivers:** Corrected workers.md, archived files annotated, `ANTHROPIC_DEFAULT_OPUS_MODEL` set to `glm-5` in both settings files
**Addresses:** T3 (fix workers.md), Pitfall 9 (archive confusion), the settings.json prerequisite
**Avoids:** Developers referencing the failed v1 approach, config swap breakage
**Research flag:** SKIP -- trivial text changes + 1-line env var edit, all verified by direct read

### Phase 3: Core Routing Mechanism
**Rationale:** This is the main event. Two approaches exist (frontmatter vs Task tool param). STACK.md recommends frontmatter (simpler), ARCHITECTURE.md recommends Task param (flexible). Both are proven working. The safest path is to implement Approach A (frontmatter) first since it requires only 14 one-word changes to agent files and zero playbook changes. Then validate end-to-end with a single build. If Approach B is desired for profile support, it can be layered on top in Phase 4.
**Delivers:** 14 agent files updated (3 to `model: opus`, 11 to `model: sonnet`), model-profiles.yaml updated with two-tier caste mapping, `getModelSlotForCaste()` function in model-profiles.js, `model-slot get` subcommand, new slot tests
**Addresses:** T1 (slot mapping), T2 (spawn model param), T4 (dual-mode resolution), T5 (CLI override)
**Implements:** The complete routing chain from config to Claude Code to proxy to GLM
**Avoids:** Pitfall 1 (fiction routing), Pitfall 7 (wrong model names in config), Pitfall 8 (parser divergence)
**Research flag:** NEEDS RESEARCH for Task tool `model` parameter override precedence -- does a Task tool `model="opus"` override agent frontmatter `model: sonnet`? This determines whether Approach B can coexist with Approach A. Test with a single Task call before committing to the full implementation.

### Phase 4: Profile System and Dynamic Overrides
**Rationale:** Once base routing works, add the ability to switch profiles at runtime (deep/default/fast) and override per-build. This is where Approach B (Task tool `model` parameter) becomes valuable -- the build playbook resolves the active profile, looks up the caste's model slot, and passes it in the Task call. Also extends routing to non-build commands (swarm, seal, colonize, patrol, organize).
**Delivers:** Profile system in model-profiles.yaml, `/ant:set-profile` command, build playbook model resolution step, routing in 7 non-build command files, CLI `--model` flag accepting slot names
**Addresses:** D2 (per-profile assignments), D3 (runtime switching), Pitfall 5 (missing model logging), Pitfall 6 (lifecycle edge cases)
**Uses:** getModelSlotForCaste() from Phase 3, model-profiles.yaml profiles section
**Research flag:** NEEDS RESEARCH for OpenCode agent spawning -- does OpenCode support the `model:` frontmatter field or equivalent? OpenCode parity is a low-priority concern but needs investigation.

### Phase 5: Verification, Polish, and GLM-5 Loop Prevention
**Rationale:** After routing is working, harden the system. Add end-to-end model verification (spawn-tree.txt shows actual model per caste), application-level loop detection for GLM-5 castes, update all documentation, and verify the config swap workflow works bidirectionally.
**Delivers:** spawn-tree.txt model field populated, loop detection warnings, updated verify-castes.md, CLAUDE.md documentation, config swap verification
**Addresses:** D4 (usage tracking), Pitfall 2 (config swap breakage), Pitfall 3 (GLM-5 looping), Pitfall 10 (stale metadata)
**Research flag:** NEEDS RESEARCH for GLM-5 constraint passing -- does Claude Code pass temperature/top_p/max_tokens to subagent API calls? If not, GLM-5 may loop in reasoning caste contexts. This is the highest-risk unknown and may require a design decision about whether Prime stays on GLM-5 or falls back to GLM-5-turbo.

### Phase Ordering Rationale

- Phase 1 must come first because test infrastructure blocks all subsequent YAML changes (Pitfall 4)
- Phase 2 is zero-risk cleanup that unblocks Phase 3 by fixing docs and enabling the settings.json mapping chain
- Phase 3 is the core deliverable -- everything else depends on base routing working
- Phase 4 extends Phase 3 with user-facing features (profiles, overrides, broader command coverage)
- Phase 5 is safety hardening and documentation -- comes after the feature is working
- This ordering front-loads risk (prove routing works in Phase 3) and defers polish (Phase 5)

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 3:** Task tool `model` parameter override precedence -- test whether `Task(model="opus")` overrides agent frontmatter `model: sonnet`. One live test resolves this.
- **Phase 4:** OpenCode `model:` frontmatter support -- check OpenCode agent spec for equivalent. Low priority but needed for parity.
- **Phase 5:** GLM-5 constraint passing through Claude Code subagent spawning -- this is the highest-risk unknown. If constraints are not passed, Prime may loop on GLM-5. May require a design decision.

Phases with standard patterns (skip research-phase):
- **Phase 1:** Test refactoring -- well-understood pattern, extract shared helper
- **Phase 2:** Config changes -- trivial edits verified by direct file reads

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Settings.json mechanism verified by direct read. Agent frontmatter confirmed by official Anthropic docs. The mapping chain (frontmatter -> env var -> proxy -> API) is documented Claude Code behavior. |
| Features | HIGH | GSD provides a working production reference for slot-based routing, profiles, CLI overrides. Caste classification rationale is grounded in GLM-5 vs GLM-5-turbo documented strengths/weaknesses. |
| Architecture | HIGH | Both approaches (frontmatter and Task param) are proven working in GSD. The routing chain is fully traced from config through Claude Code to proxy. One open question on override precedence. |
| Pitfalls | HIGH | All pitfalls grounded in direct codebase inspection (184 test occurrences, 20 spawn points, parser divergence verified). The v1 failure mode is documented in detail. |

**Overall confidence:** HIGH

### Gaps to Address

- **GLM-5 constraint passing through subagent spawning:** Does Claude Code forward temperature/top_p/max_tokens to Task-spawned subagent API calls? If not, GLM-5 reasoning castes may produce unstable output. Resolution: live test during Phase 3 implementation. If constraints are not passed, consider keeping Prime on GLM-5-turbo (inherit) and only routing Archaeologist/Route-Setter to GLM-5.
- **Task tool `model` parameter vs frontmatter precedence:** If a Task call specifies `model="opus"` but the agent frontmatter says `model: sonnet`, which wins? Resolution: single live test during Phase 3. If Task param does not override, Approach A (frontmatter) is the only viable option.
- **OpenCode parity:** Does OpenCode support agent frontmatter `model:` field? Resolution: check OpenCode docs during Phase 4. If not, OpenCode agents stay on `inherit` -- acceptable as OpenCode is the secondary target.
- **Oracle and Architect castes lack dedicated agent files:** Their work runs through Queen or direct CLI invocation. Queen gets `model: opus`, so Queen-performed oracle/architect work uses GLM-5. But direct `/ant:oracle` CLI calls use the parent session model (currently glm-5-turbo). Resolution: acceptable for MVP, document the gap.

## Sources

### Primary (HIGH confidence)
- Official Anthropic docs: Claude Code sub-agents page -- confirms `model:` field accepts `sonnet`, `opus`, `haiku`, `inherit`
- GSD `resolveModelInternal()` (gsd-tools.cjs:3970-3985) -- working production pattern for slot-based routing
- GSD `MODEL_PROFILES` table (gsd-tools.cjs:128-140) -- 11 agents, 3 profiles, slot-based mapping
- GSD `model-profiles.md` / `model-profile-resolution.md` references -- documents `"inherit"` rationale and Task tool usage
- `~/.claude/settings.json` and `~/.claude/settings.json.glm` -- verified by direct read
- `.claude/agents/ant/*.md` (22 files) -- all verified, all have `model: inherit` on line 6
- `.aether/model-profiles.yaml` -- verified by direct read, all 10 castes currently map to glm-5-turbo
- `.aether/archive/model-routing/README.md` -- v1 failure documentation, confirms env var limitation

### Secondary (MEDIUM confidence)
- Claude Code Task tool `model` parameter behavior -- observed working in GSD, not formally documented by Anthropic
- LiteLLM proxy model name resolution -- proxy config not examined directly, assumed to be correctly configured
- OpenCode agent frontmatter support -- not yet verified

---
*Research completed: 2026-03-27*
*Ready for roadmap: yes*
