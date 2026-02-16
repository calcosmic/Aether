# Project Research Summary

**Project:** Aether Colony System
**Domain:** AI agent orchestration framework with multi-model routing and colony lifecycle management
**Researched:** 2026-02-14
**Confidence:** HIGH

## Executive Summary

The Aether Colony System is an AI agent orchestration framework using an ant colony metaphor to manage software development workflows. Two parallel workstreams emerged from research: **v1.1 bug fixes** address critical infrastructure gaps (missing package-lock.json, checkpoint data loss, testing infrastructure), while **v3.1 "Open Chambers"** introduces model routing verification and colony lifecycle management.

The most critical finding: **model routing configuration exists but may not be actually executing**. Dream session research explicitly noted "model routing isn't actually happening" — workers may all be using default models regardless of caste assignments in model-profiles.yaml. This gap between configuration and execution must be closed before v3.1 features can be built reliably.

Expert practice for CLI-based orchestration systems emphasizes: (1) deterministic builds require lockfile commitment, (2) user data boundaries must be enforced by allowlist (never blocklist), and (3) state transitions must be verified before mutation. The recommended approach is infrastructure-first: harden v1.1 foundation (testing, checkpoints, deterministic builds), then verify model routing execution, then build lifecycle features on that verified foundation.

## Key Findings

### Recommended Stack

The existing Node.js CLI stack (commander.js, AVA, picocolors) is validated. Key additions:

**Core technologies (keep):**
- **Node.js >=16.0.0**: Runtime — meets requirements
- **commander ^12.1.0**: CLI argument parsing — proven stable
- **AVA ^6.0.0**: Unit testing — already configured
- **picocolors ^1.1.1**: Colored output — lightweight

**Required additions for v1.1:**
- **package-lock.json**: Deterministic builds — generate via `npm install`, commit, use `npm ci` in CI
- **sinon ^17.0.0**: Test mocking — industry standard, required for unit testing sync functions
- **proxyquire ^2.1.3**: Dependency injection — enables mocking `fs` module in cli.js tests
- **tmp ^0.2.1**: Temporary directories — handles OS-specific temp locations, auto-cleanup

**What to avoid:**
- Jest (heavier, slower than AVA)
- mock-fs (less flexible than sinon + proxyquire)
- Git stash with `--include-untracked` (causes data loss)

### Expected Features

**v1.1 Must have (P0 — table stakes fixes):**
- Targeted git checkpoints — Only stash system files, never user data
- Deterministic dependency builds — Commit package-lock.json
- Unit tests for core sync functions — Test `syncDirWithCleanup`, `hashFileSync`
- Synchronous worker spawns — Fix output timing by removing `run_in_background`

**v3.1 Must have (table stakes):**
- Model Verification Command (`/ant:models`) — Display current assignments
- Archive Command (`/ant:archive`) — Preserve colony history
- Foundation Command (`/ant:foundation`) — Start fresh colony
- Milestone Auto-Detection — Compute maturity from state
- Model Override (`--model` flag) — Force specific model per command
- Proxy Health Check — Verify LiteLLM proxy before operations

**v3.1 Should have (differentiators):**
- Task-Based Model Routing — Keyword detection ("design" → glm-5)
- Model Performance Telemetry — Track success rates per model/caste
- Milestone Progress Visualization — ASCII art maturity journey
- Colony History Timeline — Browse archived colonies

**Defer (v3.2+):**
- Intelligent Model Selection — AI-driven complexity analysis (high cost)
- Cross-Colony Analytics — Compare performance across colonies
- Cloud-Based Model Routing — Violates local-first principle

### Architecture Approach

The system uses a four-layer architecture with a new Model Routing layer:

**Major components:**
1. **Queen Layer** — Orchestration, phase management, worker spawning
2. **Constraints Layer** — Declarative focus/avoid rules
3. **Worker Layer** — Task execution with depth limits
4. **Model Routing Layer** — Caste-to-model mapping via `model-profiles.yaml`, LiteLLM proxy
5. **Utility Layer** — Deterministic operations via `aether-utils.sh`
6. **Data Layer** — Persistent state in `COLONY_STATE.json`

**Key patterns:**
- Environment variable injection: `ANTHROPIC_MODEL` set before Task spawns
- Proxy routing: LiteLLM at `localhost:4000` maps model aliases to provider APIs
- State-driven lifecycle: Colony maturity computed from COLONY_STATE.json
- Archive with manifest: All archives include metadata.json for reproducibility

### Critical Pitfalls

1. **Git stash captures user data (CRITICAL)** — Checkpoint system stashes ALL dirty files. **Avoid by:** explicit allowlist approach, verify `git status --porcelain` before stash, never use `--include-untracked`.

2. **Model Routing Without Verification (CRITICAL)** — Configuration exists in YAML but execution path isn't verified. Workers may all use default model. **Avoid by:** Logging actual model at spawn, health-checking proxy, including model in worker prompts.

3. **Proxy Authentication Failures Silently Defaulting (CRITICAL)** — LiteLLM returns 401 but system continues. **Avoid by:** Testing auth explicitly, monitoring proxy logs, rejecting spawn on auth failure.

4. **Pause/Resume Loses Model Context (CRITICAL)** — Model assignments not persisted in COLONY_STATE.json. **Avoid by:** Storing `model_used` in task metadata, versioning profiles, including models in HANDOFF.md.

5. **Archive/Reset Destroys User Data (CRITICAL)** — User learnings/decisions treated as ephemeral. **Avoid by:** Separating user data from system state, preserving memory objects in archives.

6. **Phase advancement loops (CRITICAL)** — State machine lacks guard conditions. **Avoid by:** Enforcing Iron Law (no advancement without verification), checking `state != "COMPLETED"` before transition.

## Implications for Roadmap

### Phase 1: v1.1 Foundation (Week 1)
**Rationale:** Safe checkpoint system and testing infrastructure provide foundation for all subsequent work
**Delivers:** Targeted checkpoint system, unit test framework with mocking, package-lock.json
**Addresses:** Targeted git checkpoints, Unit tests for core sync, Deterministic builds
**Avoids:** Git stash data loss, Missing unit tests pitfalls
**Research Flag:** SKIP — standard patterns, well-documented

### Phase 2: v1.1 Core Fixes (Week 2)
**Rationale:** Update system depends on safe checkpoint system; phase advancement depends on testing infrastructure
**Delivers:** Update system with rollback, phase advancement guards with idempotency keys
**Addresses:** Cross-repo sync reliability, Phase advancement guards, Synchronous worker spawns
**Avoids:** Phase advancement loops, Update stash not recovered, Misleading output timing
**Research Flag:** SKIP — patterns documented, standard state machine practices

### Phase 3: v3.1 Model Routing Verification (Week 3-4)
**Rationale:** Must verify routing works before building features on top of it
**Delivers:** Verified model assignments, proxy health checks, model verification command
**Addresses:** Model Verification Command, Proxy Health Integration
**Avoids:** Model routing without verification, Proxy auth silent fallback, Caste-model mismatch
**Research Flag:** NEEDS RESEARCH — Environment variable inheritance in Claude Code Task tool needs empirical testing

### Phase 4: v3.1 Colony Lifecycle (Week 5-6)
**Rationale:** Archive/foundation commands enable safe experimentation; independent of routing verification
**Delivers:** Archive command, foundation command, maturity detection
**Addresses:** Archive Command, Foundation Command, Milestone Auto-Detection
**Avoids:** Pause/resume loses model context, Archive destroys user data
**Research Flag:** SKIP — builds on existing state patterns

### Phase 5: v3.1 Advanced Routing (Week 7-8)
**Rationale:** Enhance routing once basic verification is solid
**Delivers:** Task-based routing, model override flag, performance telemetry
**Addresses:** Task-Based Routing, Model Override, Model Performance Telemetry
**Avoids:** Task routing never triggered, Latency coordination issues
**Research Flag:** NEEDS RESEARCH — Task keyword matching strategy needs validation

### Phase Ordering Rationale

- **v1.1 before v3.1:** Testing infrastructure enables safe changes to core routing
- **Checkpoint system first** — Provides rollback safety net; fixes critical data loss risk
- **Verification before features:** Must prove routing works before building on it
- **Lifecycle parallel to routing:** Archive/foundation are independent, can ship while verifying routing
- **Advanced routing last:** Task keyword matching requires production data to validate

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 3 (Model Routing Verification):** Environment variable inheritance in Claude Code Task tool is undocumented — needs empirical testing
- **Phase 5 (Advanced Routing):** Task keyword matching strategy — need to validate keyword detection accuracy

Phases with standard patterns (skip research-phase):
- **Phase 1-2 (v1.1):** Standard npm/Node.js testing patterns, well-documented state machine practices
- **Phase 4 (Colony Lifecycle):** Builds on existing state management patterns

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Based on existing codebase, standard npm practices |
| v1.1 Features | HIGH | Based on documented bugs, real data loss incident |
| v3.1 Features | HIGH | Based on existing command patterns and yaml config |
| Architecture | HIGH | Existing four-layer system is well-documented |
| Pitfalls | HIGH | Dream session explicitly identified gaps, direct testing confirmed proxy auth issues |

**Overall confidence:** HIGH

The research is based on:
- Direct codebase analysis (workers.md, model-profiles.yaml, COLONY_STATE.json)
- Dream session findings that explicitly flagged "model routing isn't actually happening"
- Direct proxy testing that revealed 401 auth errors
- Documented v1.1 bugs with full context (data loss from stash)

### Gaps to Address

1. **Environment Inheritance:** How exactly does Claude Code Task tool inherit environment variables? Documented behavior vs actual behavior needs validation during Phase 3.

2. **Task Keyword Matching:** What keywords reliably indicate task complexity? "Design" → glm-5 seems logical but needs validation with real task descriptions during Phase 5.

3. **Proxy Auth Configuration:** Current proxy returns 401 — is this a configuration issue or expected behavior? Needs investigation before Phase 3.

4. **Allowlist Completeness:** Must audit all `.aether/` subdirectories to confirm which are system vs user data before implementing checkpoint fix.

5. **State Migration Path:** If COLONY_STATE.json format changes, what's the migration strategy? Version field exists but migration logic unclear.

## Sources

### Primary (HIGH confidence)
- `.aether/workers.md` — Worker caste definitions, model assignments
- `.aether/model-profiles.yaml` — Model metadata, routing configuration
- `.aether/dreams/2026-02-14-0238.md` — Gap analysis: "model routing isn't actually happening"
- `bin/cli.js` — Current CLI implementation
- `package.json` — Existing stack validation
- Aether TO-DOs.md — Documented v1.1 bugs with full context
- Aether CONCERNS.md — Technical debt and security audit

### Secondary (MEDIUM confidence)
- `.claude/commands/ant/build.md` — Current spawn logic patterns
- `.claude/commands/ant/init.md` — Colony initialization patterns
- `.claude/commands/ant/seal.md` — Archive/sealing patterns
- `.aether/data/COLONY_STATE.json` — State structure analysis
- `.aether/aether-utils.sh` — State management commands

### Tertiary (LOW confidence / needs validation)
- Direct LiteLLM proxy test — Returned 401, needs configuration investigation
- Task tool environment inheritance — Behavior not explicitly documented

---
*Research completed: 2026-02-14*
*Ready for roadmap: yes*
