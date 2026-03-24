# Project Research Summary

**Project:** Aether v2.1 Production Hardening
**Domain:** Multi-agent CLI orchestration (bash + Node.js)
**Researched:** 2026-03-23
**Confidence:** HIGH

## Executive Summary

Aether v2.0 is a shipped product with a solid architectural foundation — a proven bash+Node.js dual runtime, 22 agents, 44 commands, 28 skills, and cross-colony learning infrastructure that no competitor has. The Oracle audit (12 iterations, 55 findings, 82% confidence) exposed the precise gap between "impressive demo" and "trusted production tool": 338 silent error suppression patterns, 43% dead code, state desync risks under autopilot, and documentation that describes aspirational behavior rather than implemented behavior. This milestone is not about adding features — it is about making the features that exist actually work reliably.

The recommended approach is a tightly sequenced hardening effort that respects the dependency graph of fixes. The Oracle audit provides enough specificity to skip ambiguous analysis phases and go straight to targeted repairs. Six of the highest-impact fixes are trivially small (under 20 lines each): type coercion in hive-read, midden temp file race, memory pipeline circuit breaker, state checkpointing, continue-advance lock gap, and context trimming transparency. These should be executed first as quick wins to demonstrate forward momentum and reduce the blast radius of the larger modularization work.

The primary risk is the refactoring death spiral: the rising bug-fix ratio (33.8% to 45.8%) signals that error handling is load-bearing. Removing `|| true` patterns without replacing them with proper fallback behavior at each call site will cascade failures across the build/continue lifecycle. The mitigation is clear: triage before you touch. Categorize all 338 error suppressions as correct/lazy/dangerous, fix dangerous ones first, and never remove any suppression without an explicit replacement. Dead code removal must precede modularization; documentation must come last.

---

## Key Findings

### Recommended Stack

The fundamental stack (bash + jq + Node.js + JSON files) stays. This is not a technology preference but a hard constraint — 530+ tests, 43 slash commands, and 22 agent definitions all depend on the existing interface contract. The research confirmed this is the right call: migrating bash to Node.js would double the Node.js surface area with no reliability benefit; adding SQLite would introduce native binary dependencies for a single-user CLI; rewriting aether-utils.sh would risk losing edge-case handling earned through 17 real midden failures.

The hardening is entirely about patterns, not new libraries. The existing `hive.sh`, `midden.sh`, and `skills.sh` extractions prove the modularization pattern works. The existing `atomic-write.sh` and `file-lock.sh` prove the concurrency infrastructure is sound. Every recommended fix extends or corrects what exists rather than replacing it.

**Core pattern changes:**
- Error triage: Three-category classification (correct/lazy/dangerous) with `# SUPPRESS:OK` comment markers for auditable review
- State protection: `cp COLONY_STATE.json COLONY_STATE.phase-N.bak` before each build — bounded to last 3 checkpoints
- Memory circuit breaker: Detect-recover-log at the learning-observations.json boundary using the existing template file
- jq type safety: `(.confidence | tonumber? // 0)` at all numeric comparison sites — 5-line fix
- Bash modularization: Continue the domain extraction pattern (pheromone.sh, learning.sh, colony.sh) following the proven `hive.sh` precedent
- ShellCheck escalation: Add `.shellcheckrc` and expand scope from 6 files to all `.sh` files

**Version constraint to change:** Bump Node.js engine requirement from `>=16.0.0` to `>=20.0.0` to align with AVA v6 test requirement. Node 16 and 18 are both EOL; Aether's target audience (Claude Code and OpenCode users) will have Node 20+ installed.

### Expected Features

The features research reveals two distinct tiers: reliability fixes that users need to trust the system, and differentiating improvements that make Aether better than running Claude Code manually.

**Must have (table stakes — users lose trust without these):**
- Error visibility: Surface failures instead of hiding them — 338 silent suppressions erode confidence in all outputs
- State checkpoint and rollback: Per-phase backup before builds — total loss on mid-autopilot corruption is unacceptable
- Verification that catches lies: Cross-reference test exit codes against Watcher claims — schema-only validation is the primary hallucination vector
- Dead code removal: 76 unused subcommands (43%) at 11,272 lines — maintenance burden grows with every change
- Memory pipeline resilience: Corrupted learning-observations.json permanently kills all 5 downstream memory steps silently
- Autopilot state reconciliation: run-state.json and COLONY_STATE.json can desync with no detection
- Type-safe data operations: String-typed confidence values silently exclude valid hive wisdom — 5-line fix with outsized impact
- Documentation matches behavior: 6 confirmed inaccuracies (trim order inversion, subcommand count, security gate label oversell, etc.)

**Should have (competitive — why use Aether over manual Claude):**
- Per-phase research loop: The single highest-leverage differentiator; structured plans achieve 61% first-attempt success vs 23% ad-hoc (3.2x improvement per published research)
- Evidence-based verification: Verify Builder's claimed files exist on filesystem; captures fabricated responses
- Context trimming transparency: One notice line when colony-prime trims context — workers currently operate blind
- Agent fallback visibility: Log degradation to midden when specialized agent falls back to general-purpose
- Cross-colony learning that works: The infrastructure exists and is unique in the ecosystem; the type coercion bug prevents it from functioning

**Defer to v3+:**
- Web/TUI dashboard — orthogonal to core value, massive scope increase
- Real-time inter-worker messaging — requires platform changes Claude Code does not support
- More agent types — fix integration gaps in 22 existing agents before adding more
- Multi-repo colony coordination — fundamentally different architecture problem

### Architecture Approach

The target architecture keeps the single-entry-point bash dispatch contract intact (`bash .aether/aether-utils.sh <subcommand>`) while extracting the monolith's logical domains into independently testable modules in `.aether/utils/`. The consumer layer (43 slash commands + 22 agents) sees no change — all 412 call-site references continue to work. The state layer gets a `state-api.sh` facade that centralizes COLONY_STATE.json access and eliminates the dual-write-path race (90 inline jq references in the monolith + 20+ bypass reads in slash commands).

**Major components:**
1. `aether-utils.sh` (slimmed dispatcher, ~1,500 lines after extraction) — setup, sourcing, case dispatch, shared helpers only
2. `state-api.sh` (new) — all COLONY_STATE.json reads/writes with lock/validate/migrate encapsulated; eliminates dual-access path
3. `pheromone.sh` (extract, ~1,800 lines) — pheromone system + colony-prime; eliminates self-invocation overhead by replacing subprocess spawning with direct function calls
4. `learning.sh` (extract, ~1,200 lines) — learning engine + memory-capture pipeline; dependent on state-api and pheromone
5. `queen.sh`, `colony.sh`, `swarm.sh`, `session.sh`, `spawn.sh`, `flag.sh`, `suggest.sh` — remaining domain extractions in dependency order
6. Error handling audit (final pass across all modules) — classify remaining `2>/dev/null` instances after each module is isolated

**Build order is strict:** state-api first, then pheromone, then learning, then queen, then parallel remaining domains, then error audit last. This dependency graph is non-negotiable — violating it is the primary architecture risk.

### Critical Pitfalls

1. **Refactoring death spiral** — The 338 error-swallowing patterns are load-bearing; callers were written to expect silence. Triage before touching. Fix dangerous suppressions (state-writing paths, ~40 instances) first; mark correct suppressions with `# SUPPRESS:OK`; address lazy suppressions last. Never remove `|| true` without an explicit replacement and a regression test.

2. **Monolith extraction breaks dispatch** — aether-utils.sh functions share file-scoped state invisibly. Remove the 76 dead subcommands before any extraction — dead code creates false dependency signals. Write a contract test for each extraction before moving code. Preserve `set -euo pipefail` in every extracted module (ERR traps are not inherited across `source` boundaries in all bash versions).

3. **State protection deadlock** — COLONY_STATE.json has a dual-access path: bash subcommands (with locks) AND the LLM Write tool (no locks). Do not add lock requirements to the Write tool path — it will deadlock. Implement checkpointing (purely additive `cp`) first; add post-write validation second; defer lock changes until after checkpointing is stable and tested.

4. **Dead code removal kills live functions** — The Oracle's static analysis covered `.claude/commands/ant/` but NOT `.opencode/commands/ant/` (40 additional commands) or user-created skills at `~/.aether/skills/domain/`. Before removing any subcommand, check all three surfaces. Use deprecation warnings first, removal one cycle later.

5. **Documentation drift after early correction** — Fixing docs before fixing code creates a window where docs are "accurate" but the code is about to change. Document inaccuracies as `[KNOWN INACCURACY]` annotations until the corresponding code is stable. Documentation accuracy pass is the final phase, not the first.

---

## Implications for Roadmap

Based on the combined research, the hardening work falls into a strict dependency graph that maps naturally to 8 phases. The pitfalls research is unusually specific about ordering constraints — violating the sequence below is the primary risk factor for the entire milestone.

### Phase 1: Quick Wins and Baseline Stabilization

**Rationale:** Six independent fixes with zero dependencies and outsized impact. Establishing a green baseline before any structural work begins is mandatory — the test suite has 1 existing failure (`pheromone-prime --compact respects max signal limit`), and hardening with a red baseline provides no signal for regressions introduced by later changes.

**Delivers:** Fixed hive wisdom retrieval (type coercion, 5 lines), stabilized midden writes (PID-scoped temp file, 3 lines), context trimming transparency (~30 chars added to colony-prime output), autopilot state reconciliation (run-state vs COLONY_STATE comparison at loop start, ~10 lines), agent fallback logging (midden entry on degradation, ~10 lines), and a fully green test suite.

**Addresses:** T9 (type coercion), T8 (autopilot reconciliation), D2 (trim notification), D3 (fallback visibility), Pitfall 7 (hive type coercion migration), Pitfall 8 (test baseline green)

**Avoids:** Starting structural work on a red test suite; compounding existing bugs with new changes.

**Research flag:** No deeper research needed — all fixes are pinpointed to exact lines by the Oracle audit.

### Phase 2: Error Handling Triage

**Rationale:** T1 (error visibility) is the critical enabler — three other features depend on it, and the modularization work in Phase 6 will touch the same code paths. This phase is read-only analysis plus dangerous-category fixes only. Attempting modularization before triage creates the refactoring death spiral.

**Delivers:** All 338 error suppressions classified into correct/lazy/dangerous categories with `# SUPPRESS:OK` markers on confirmed-correct instances; ~40 dangerous-category suppressions on state-writing paths replaced with proper error handling; suggest-analyze ERR trap gap (200 lines at lines 10236-10427) fixed; concurrent test failure audit complete.

**Addresses:** T1 (error visibility), D6 (structured error triage), Pitfall 1 (death spiral prevention), Pitfall 8 (tests that assert broken behavior)

**Avoids:** The refactoring death spiral; touching lazy/correct suppressions before dangerous ones are stable.

**Research flag:** No deeper research needed — Oracle Q3 maps all three categories with line-level specificity.

### Phase 3: State Protection and Memory Pipeline

**Rationale:** State checkpointing is purely additive (a `cp` before each build) and cannot break anything. The memory circuit breaker is the other critical non-structural fix. Both must complete before modularization because Phase 6 will touch the code paths these fixes affect.

**Delivers:** Per-phase COLONY_STATE.json checkpoints with 3-checkpoint rotation before each build-wave; memory-capture circuit breaker with template recovery and midden audit trail; continue-advance state write routed through `state-advance` subcommand to close the LLM Write tool lock gap.

**Addresses:** T2 (state checkpoint), T7 (memory pipeline resilience), Pitfall 4 (state protection without deadlock), Pitfall 6 (circuit breaker correctness)

**Avoids:** Total loss on mid-autopilot corruption; permanent silent learning death from corrupted observations file; deadlock from adding bash locks to the LLM Write tool path.

**Research flag:** No deeper research needed — Oracle Rec 1, 7, 8, 9 provide exact implementation patterns.

### Phase 4: Dead Code Deprecation

**Rationale:** Dead code must be deprecated before modularization — live code within the 76 "dead" subcommands creates false dependency signals during extraction. Deprecation (warnings, not removal) is safe; removal happens one cycle later after deprecation warnings confirm no callers exist across all three command surfaces.

**Delivers:** All 76 suspected-dead subcommands emit deprecation warnings to stderr; full three-surface grep audit (`.claude/`, `.opencode/`, `~/.aether/skills/`) confirms which are truly dead; Node.js engine requirement bumped to `>=20.0.0`.

**Addresses:** T6 (dead code removal), Pitfall 5 (hidden live functions in Oracle's static analysis blind spots)

**Avoids:** Breaking OpenCode users or custom user skills by removing subcommands that were outside the Oracle's analysis scope.

**Research flag:** No deeper research needed. The deprecation pattern is standard bash; the work is grep analysis. Actual removal requires one development cycle of runtime confirmation.

### Phase 5: Verification Evidence Chain

**Rationale:** T4 (verification that catches lies) is the primary hallucination vector and is independent of the modularization work. Addressing it before Phase 6 keeps the scope clean and the test signal clear.

**Delivers:** Test runner exit code captured during build-verify and cross-referenced against Watcher's `verification_passed` claim; Builder's `files_created` list verified against actual filesystem; responses that contradict evidence are rejected before advancing the phase.

**Addresses:** T4 (verification), D4 (evidence-based verification)

**Avoids:** Workers fabricating completion claims that pass gates silently; planning advancing on false completion.

**Research flag:** No deeper research needed — Oracle Rec 2 provides the exact pattern; the TDD evidence gate in continue-gates.md Step 1.10 is the proven model to extend.

### Phase 6: Monolith Modularization

**Rationale:** This is the largest structural phase and has the strictest prerequisites — Phases 1 through 4 must all be complete. A green test suite, triaged error handling, protected state, and deprecated dead code are all required before extraction begins. Attempting this earlier is the highest-risk mistake in the entire hardening effort.

**Delivers:** `state-api.sh` facade for all COLONY_STATE.json access; `pheromone.sh` extraction (~1,800 lines including colony-prime); `learning.sh` extraction (~1,200 lines); `queen.sh`, `colony.sh`, and remaining domain modules extracted in dependency order; aether-utils.sh reduced from ~11,272 lines to ~1,500-line dispatcher; dead subcommands confirmed removed after one-cycle deprecation.

**Addresses:** Architecture target state, Pitfall 2 (dispatch contract preservation throughout extraction), T6 (dead code removal confirmed after deprecation cycle)

**Avoids:** Extraction order violations; breaking the single-entry-point bash interface; introducing new shared-state bugs across module boundaries.

**Research flag:** No deeper research needed — the extraction pattern is proven by three existing working examples (`hive.sh`, `midden.sh`, `skills.sh`). Follow the same protocol for each new module.

### Phase 7: Planning Depth and Per-Phase Research

**Rationale:** D1 (per-phase research loop) is the highest-impact differentiator — the gap between Aether and manual Claude Code usage. This comes after core hardening because it builds on stable infrastructure, and because shallow planning on an unstable platform produces research-backed phases that then fail to execute reliably.

**Delivers:** Per-phase research step before each build phase; scouts investigate domain knowledge, library docs, and patterns (not just codebase); research depth flag in phase plans; time-bounded investigation (one cycle, not a full Oracle RALF loop) to prevent the planning recursion pitfall.

**Addresses:** T3 (planning depth), D1 (per-phase research loop), Pitfall 10 (infinite recursion prevention through hard time budget and termination condition)

**Avoids:** Research becoming a bottleneck; planning taking longer than execution; changes to the phase plan schema that break continue.md and autopilot.

**Research flag:** This phase benefits from a targeted design spike. The key question: how to distinguish phases that need research from phases that do not (to avoid universal overhead on simple phases). GSD's pattern (the tool you are reading this from) is the reference model, but adapting it to Aether's scout/route-setter architecture requires a design decision before implementation.

### Phase 8: Documentation Accuracy and Onboarding Polish

**Rationale:** Documentation must come last — every prior code change makes earlier documentation corrections stale. Onboarding polish validates the full install-through-first-build flow with all v2.1 changes in place.

**Delivers:** All 6 known documentation inaccuracies corrected with corresponding doc-truth tests (CLAUDE.md trim order, subcommand count, security gate label, etc.); end-to-end onboarding flow validated on a clean machine; `aether install --force` recovery path for partial installs; post-install validation that warns on missing hub files.

**Addresses:** T5 (documentation matches behavior), D8 (onboarding polish), Pitfall 3 (documentation drift prevention via doc-truth tests), package distribution hardening

**Avoids:** Correcting docs before the code they describe is stable; creating a false sense of accuracy mid-milestone.

**Research flag:** No deeper research needed — the 6 specific inaccuracies are enumerated by the Oracle audit with exact file locations.

### Phase Ordering Rationale

- Phases 1-3 are strictly ordered: baseline green, then triage errors, then protect state. Each phase's safety net depends on the previous phase being complete.
- Phase 4 (dead code deprecation) and Phase 5 (verification chain) can overlap with each other but both must precede Phase 6 (modularization).
- Phase 6 is the critical path item — longest effort, most prerequisites, gates Phase 7.
- Phase 7 and Phase 8 are sequenced but relatively independent; Phase 8 documentation work could start in parallel with Phase 7's design spike if timeline pressure demands it.

### Research Flags

Phases likely needing a deeper research or design spike during planning:
- **Phase 7 (Planning depth):** The exact structure of the per-phase research prompt requires a design decision — specifically, how to distinguish "phases that need research" from "phases that do not" without adding universal overhead to simple phases.

Phases with standard patterns (skip deeper research):
- **Phase 1:** All fixes pinpointed to exact lines by Oracle audit. No ambiguity.
- **Phase 2:** Oracle Q3 provides line-level specificity on all 338 suppressions. Three-category triage is a well-understood pattern.
- **Phase 3:** Oracle Rec 1, 7, 8, 9 provide exact implementation patterns with code examples.
- **Phase 4:** Deprecation pattern is standard bash; the work is grep analysis across three surfaces.
- **Phase 5:** TDD evidence gate in continue-gates.md Step 1.10 is the proven model to extend.
- **Phase 6:** Extraction pattern is proven by three working examples. Dependency graph is confirmed by static analysis.
- **Phase 8:** The 6 inaccuracies are enumerated; correction is mechanical and paired with doc-truth tests.

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | No new dependencies. All patterns derived from direct codebase analysis and Oracle audit at 85% multi-source trust ratio. Anti-patterns confirmed against production bash codebase literature. |
| Features | HIGH | Oracle audit at 82% confidence with 55 findings; competitor analysis confirms Aether's unique cross-colony learning differentiator; feature dependencies traced explicitly with dependency graph. |
| Architecture | HIGH | Extraction pattern proven by three working examples in the same codebase. Dependency graph confirmed by static analysis. Consumer API contract (412 call sites) fully mapped. |
| Pitfalls | HIGH | Grounded in 17 real midden failures, a measured and rising bug-fix ratio, 572+ tests, and static analysis with explicit line citations. Not theoretical — observed behavior with evidence. |

**Overall confidence:** HIGH

### Gaps to Address

- **Dead code surface coverage:** The Oracle's static analysis confirms 76 dead subcommands via `.claude/` and `.aether/docs/` grep. The `.opencode/commands/ant/` surface (40 commands) was not in scope. The deprecation-before-removal strategy in Phase 4 handles this by using runtime warnings as a discovery mechanism.

- **Parallel builder collision rate:** The Oracle reached the "operational ceiling" for static analysis on actual lock contention frequency. The POSIX append atomicity guarantee for `activity.log` and `spawn-tree.txt` is theoretically sound but untested under load. This does not block any phase — it is a monitoring concern for after hardening is complete.

- **Per-phase research prompt design:** Phase 7 introduces a new capability without a prior Aether implementation to reference. The GSD tool provides the model, but adapting it to Aether's scout/route-setter architecture needs a design spike before Phase 7 implementation begins.

---

## Sources

### Primary (HIGH confidence — direct codebase analysis + Oracle audit)

- Oracle synthesis: `.aether/oracle/synthesis.md` — 55 findings, 12 iterations, 82% confidence, 85% multi-source trust ratio
- Oracle gaps: `.aether/oracle/gaps.md` — all 5 questions answered, 9 contradictions resolved
- Midden records: `.aether/data/midden/midden.json` — 17 real failure entries with timestamps and categories
- Codebase analysis: `.aether/aether-utils.sh` (11,272 lines), `.aether/utils/` (5,237 lines), `bin/lib/` (6,578 lines)
- Existing extraction examples: `hive.sh`, `midden.sh`, `skills.sh` in `.aether/utils/` — proven modularization pattern

### Secondary (MEDIUM confidence — multiple external sources agree)

- [Plan-and-Act: Improving Planning of Agents for Long-Horizon Tasks](https://arxiv.org/html/2503.09572v3) — 61% vs 23% first-attempt success rate for structured vs ad-hoc planning (3.2x)
- [AI Agents in Production: What Actually Works in 2026](https://47billion.com/blog/ai-agents-in-production-frameworks-protocols-and-what-actually-works-in-2026/) — 80% of production effort is refinement, not initial development
- [LangGraph Production: Checkpointing and Error Recovery](https://markaicode.com/langgraph-production-agent/) — state snapshot patterns as competitor baseline
- [Spec-Driven Verification for Autonomous Coding Agents](https://agent-wars.com/news/2026-03-14-spec-driven-verification-claude-code-agents) — verification discipline matters more than architectural elaboration
- [ShellCheck severity documentation](https://shellcheck.net/wiki/severity) — escalation path and `.shellcheckrc` configuration verified
- [IEEE: Refactoring, Bug Fixing, and New Development Effect on Technical Debt](https://ieeexplore.ieee.org/document/9226289/) — rising fix ratios as a technical debt signal (peer-reviewed)
- [Bash modularization best practices](https://medium.com/mkdir-awesome/the-ultimate-guide-to-modularizing-bash-script-code-f4a4d53000c2) — source-based modularization performance overhead confirmed negligible

### Tertiary (MEDIUM confidence — single source or domain inference)

- [Composio: Why AI Pilots Fail in Production](https://composio.dev/blog/why-ai-agent-pilots-fail-2026-integration-roadmap) — failure mode patterns in production AI agents
- [FreeCodeCamp: How to Refactor Complex Codebases](https://www.freecodecamp.org/news/how-to-refactor-complex-codebases/) — incremental refactoring strategies for large bash codebases

---

*Research completed: 2026-03-23*
*Ready for roadmap: yes*
