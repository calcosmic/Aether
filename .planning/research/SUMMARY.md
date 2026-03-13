# Project Research Summary

**Project:** Aether v1.2 — Integration Gap Fixes
**Domain:** Multi-agent colony system — wiring existing components together
**Researched:** 2026-03-14
**Confidence:** HIGH

## Executive Summary

This is not a build milestone. It is a wiring milestone. Aether v1.1 ships a complete self-improving colony system — pheromones, instincts, midden failure tracking, memory capture, decisions, and queen promotion are all implemented and tested. The problem is that these subsystems operate in isolation. The colony memory is nearly empty in practice: decisions [], instincts [], only 1 phase_learning. The four integration loops (decisions → pheromones, learnings → instincts, midden → behavior, memory capture consistency) are documented in playbooks and backed by working subcommands, but the wiring between them has gaps that prevent natural accumulation during real build/continue cycles.

The recommended approach is purely additive. No new subcommands, no new state files, no new languages. Every fix is a targeted markdown edit to existing playbook files (continue-advance.md, build-wave.md, build-verify.md, build-complete.md) that either adds a missing subcommand call at an existing trigger point, or verifies and tightens an existing call that fires inconsistently. The entire implementation surface is 4-6 playbook files and approximately 30-50 lines of bash additions. Every temptation to add infrastructure should be resisted — the system already has too much; the challenge is connecting what exists.

The primary risks are behavioral, not technical. New playbook instructions compete silently with existing ones, silent failure paths swallow errors without signal, and memory-capture calls with generic content strings inflate observation counts without adding knowledge. Every phase must include runtime artifact verification — confirming that midden.json, pheromones.json, and learning-observations.json contain actual new entries after a test build cycle. Code review of playbook edits is necessary but not sufficient.

---

## Key Findings

### Recommended Stack

No stack changes. The existing runtime is Bash 3.2+ with jq 1.6+, awk, and sed. These are the only dependencies across the entire 9,808-line system and they cover all four integration gaps without modification. Every capability needed for the fixes — pheromone-write, instinct-create, midden-write, memory-capture — is already implemented and verified in aether-utils.sh. The implementation surface is playbook markdown, not shell functions.

**Core technologies:**
- **Bash 3.2+**: Orchestration and all subcommand dispatch — already in use everywhere, no change needed
- **jq 1.6+**: JSON read/write for all state files — required by 150+ subcommands; defensive reads (`// "fallback"` pattern) are mandatory for every new field access to ensure backward compatibility
- **awk / sed**: CONTEXT.md parsing and inline file updates — used in the decision→pheromone pipeline; deduplication format alignment needs verification

See `.planning/research/STACK.md` for exact subcommand signatures, argument shapes, and JSON schemas for all four data files (pheromones.json, COLONY_STATE.json, midden.json, learning-observations.json).

### Expected Features

The six table-stakes features (T1-T6 from FEATURES.md) close all four integration loops and constitute the full v1.2 MVP. They are all LOW to MEDIUM complexity, all wire existing subcommands to existing call sites, and all can be built in any order without dependencies between them.

**Must have (close the four integration loops):**
- **T1: Decision-to-pheromone inline emission** — `context-update decision` must call `pheromone-write FEEDBACK` at the moment of the decision, not only in the post-phase batch in continue-advance.md Step 2.1b
- **T2: Approach-change memory capture** — approach-changes.md (written by builders during task execution, never read afterward) must feed `midden-write` and `memory-capture "failure"` to make abandoned approaches discoverable
- **T3: Recurrence-calibrated instinct confidence** — `learning-observations.json` already tracks `observation_count` per content hash; this count must drive instinct confidence (`min(0.9, 0.7 + (obs_count - 1) * 0.05)`) rather than always defaulting to 0.7 regardless of recurrence evidence
- **T4: Intra-phase midden threshold check** — the midden REDIRECT threshold (3+ occurrences of same error category) must fire within a build phase in build-wave.md, not only at phase-end during continue
- **T5: Failure capture at all failure points** — `memory-capture "failure"` is missing at Gatekeeper CRITICAL and Auditor CRITICAL paths in continue-gates.md; must be wired consistently with the pattern already used in build-wave.md Step 5.2
- **T6: Rolling-summary fed into colony-prime** — `memory-capture` maintains a rolling-summary on every event but `colony-prime` does not include it; last 5 rolling-summary entries must appear in the priming payload so workers have recent activity awareness

**Should have (enhance the loops after validation):**
- **D4: User-feedback → instinct auto-create** — `/ant:feedback` FEEDBACK pheromones with strength > 0.7 should auto-create an instinct with confidence 0.9, closing the user → colony knowledge loop directly
- **D3: Instinct-to-pheromone echo at build start** — high-confidence instincts (>=0.85) should echo as FOCUS pheromones before each build wave, converting durable learned knowledge into current-phase signals

**Defer to v1.3+ (needs more data or infrastructure):**
- D1: Confidence decay for unverified instincts — requires T3 first; adds maintenance overhead
- D2: Cross-phase midden pattern surfacing — requires schema extension and a new `midden-pattern-summary` subcommand

**Anti-features to reject explicitly:**
- Automatic pheromone emission from every decision — causes signal saturation; the 3-per-continue cap exists precisely for this reason
- Real-time instinct updates during build — concurrent write conflicts; the instinct store is designed for phase-boundary updates only
- Midden as a blocking gate — midden is a record system, not a quality gate; block only on Gatekeeper CRITICAL findings (already implemented)

### Architecture Approach

All four gaps share the same structural pattern: a source event (decision, failure, success, learning) occurs in a playbook step, but the downstream pipeline call (memory-capture, midden-write, pheromone-write) is either missing at that step or fires in a post-hoc batch that arrives too late to steer the workers who produced the event. The fix in every case is to add the pipeline call at the event's call site, not to restructure the pipeline itself.

The established signal propagation chain is correct and requires no changes: `continue run N → pheromone-write → pheromones.json → build run N+1 → colony-prime → prompt_section → builder worker sees signal`. The gaps are in the upstream trigger points, not in the propagation path.

**Major components (integration-relevant):**
1. **Build playbooks** (build-wave.md, build-verify.md, build-complete.md) — event sources for failures, successes, and approach changes; missing `midden-write` and `memory-capture` calls at several failure and success points
2. **Continue playbooks** (continue-advance.md) — aggregation point for decision→pheromone and midden→REDIRECT conversion; code exists but deduplication format alignment and step ordering need verification
3. **aether-utils.sh** (150 subcommands) — fully implemented with no changes needed; all callers must conform to exact argument signatures documented in STACK.md
4. **colony-prime** — assembles worker priming payload from pheromones + instincts + QUEEN.md; does not yet include rolling-summary content (T6 gap)

Patterns to follow for all new integration calls: fail-safe execution (`2>/dev/null || true`), silent-when-empty for promotion proposals, capped emissions (3 per continue run for decisions and error patterns), and deduplication before emission for pheromones. Note: `memory-capture` handles its own deduplication internally via content hash — do not add external dedup checks around memory-capture calls.

### Critical Pitfalls

1. **Competing playbook instructions** — a new instruction added to an existing step can conflict with an earlier instruction in the same step; the agent satisfies the earlier one and silently ignores the new one, producing no error. Prevention: read the full step context before adding any instruction; explicitly state the additive relationship; verify both old and new behavior fire in a test build.

2. **Wiring correct but never observed** — all integration calls are added correctly, but `2>/dev/null || true` swallows silent failures. The developer declares the phase complete based on code review, but colony memory stays empty because a call is failing silently. Prevention: each phase's success criteria must include runtime artifact verification — confirm that specific files contain new entries after a test build cycle.

3. **Generic memory-capture content strings** — content like "Builder failed" hashes to similar values across unrelated events, inflates observation counts, and triggers auto-promotion of vague patterns to QUEEN.md. Prevention: every `memory-capture` call site must include discriminating specifics (task ID, agent name, category); content must describe a pattern, not an occurrence.

4. **Lowering the midden threshold instead of fixing the write path** — the 3+ occurrence threshold is a deliberate calibration; lowering it causes a noise spiral where ephemeral failures become 30-day REDIRECT constraints that steer builders away from already-fixed areas. Prevention: fix the write path (add more midden-write call sites) before considering threshold changes.

5. **Schema backward compatibility breaks** — new jq reads that assume fields added in v1.0/v1.1 fail silently on colonies initialized before those fields existed. Prevention: every new field read must use `// "fallback"` pattern; always test on a fresh `/ant:init` colony, not only the current dev colony.

---

## Implications for Roadmap

Based on research, 5 phases are recommended, ordered by risk from lowest to highest. Phases 3 and 4 can be parallelized after Phase 2 completes — they edit different playbooks with no shared call sites.

### Phase 1: Verification and Test Infrastructure

**Rationale:** The gaps are in existing code, not in missing code. Phase 1 establishes ground truth — what actually fires today — before any edits are made. Without a verified baseline, later phases have nothing to measure against and no way to confirm that a fix worked rather than a previously-working call broke.
**Delivers:** Integration tests that assert each pipeline fires; baseline state of midden.json, pheromones.json, learning-observations.json documented; exact format of all relevant JSON fields confirmed against live data; test infrastructure in place for subsequent phases
**Addresses:** Pitfall 6 — wiring correct but never observed; creates the observability layer that makes all subsequent phases verifiable
**Avoids:** Making changes to a system whose current behavior is misunderstood; producing fixes that coincidentally break existing behavior that was actually working

### Phase 2: Success Capture Additions (Gap 4, additive only)

**Rationale:** Lowest risk phase — purely additive, cannot break existing behavior. Adds `memory-capture "success"` at build-verify.md Step 5.7 (chaos reports strong resilience) and build-complete.md Step 5.9 (synthesis patterns_observed). These call sites are missing success capture entirely; adding it cannot affect existing failure paths.
**Delivers:** Success events enter the memory pipeline for the first time; learning-observations.json begins accumulating positive patterns; Gap 4 partially closed on the success side
**Uses:** `memory-capture "success"` (existing subcommand, ~10 lines added across 2 files)
**Implements:** Gap 4 success-capture; T5 partial (success side)
**Avoids:** Competing instruction pitfall — additive-only changes introduce no conflicts with existing instructions

### Phase 3: Midden Write Path Expansion (Gap 3)

**Rationale:** The midden threshold (3+ occurrences → REDIRECT) is correctly configured but the write path is narrow — only builder failures in build-wave.md currently write to midden. Watcher, Chaos, and verification failures are not recorded. Expanding the write path makes the threshold reachable for all failure categories without touching the threshold number itself.
**Delivers:** All failure types write to midden; midden data reflects actual colony failure patterns across all agent types; intra-phase threshold check (T4) has meaningful data to act on; approach-changes.md is processed for the first time
**Uses:** `midden-write` (existing subcommand); approach-changes.md processing for T2
**Implements:** Gap 3 write-path expansion; T2 approach-change capture
**Avoids:** Threshold lowering pitfall — fixing the write path is the correct intervention, not reducing the aggregation threshold

### Phase 4: Decision→Pheromone and Learning→Instinct Verification (Gaps 1 and 2)

**Rationale:** These gaps have existing code in place — the question is whether it fires correctly. Phase 4 verifies and tightens continue-advance.md Steps 2.1b (decision→pheromone) and Steps 3/3a/3b (learnings→instincts). The recurrence-calibrated instinct confidence (T3, using observation_count from learning-observations.json) is added here.
**Delivers:** Decision pheromones emit reliably after every continue run; instinct confidence reflects actual recurrence evidence rather than a fixed 0.7 baseline; deduplication format alignment verified and fixed if needed (PHER-01 dedup check)
**Implements:** Gap 1 verification and fix; Gap 2 ordering confirmation; T3 confidence calibration
**Avoids:** Over-precise instruction pitfall — each modified step retains a graceful degradation path with explicit non-blocking behavior

### Phase 5: Colony-Prime Enrichment and Intra-Phase Threshold (T4, T6)

**Rationale:** These features depend on Phases 2-4 producing data. T6 (rolling-summary in colony-prime) is only meaningful once memory-capture fires at all new call sites; T4 (intra-phase midden threshold) is only meaningful once the midden receives data from all failure points established in Phase 3.
**Delivers:** Workers receive rolling-summary context in priming payload for the first time; REDIRECT pheromones can fire within a build phase rather than only after continue; the colony knowledge loop is fully closed
**Implements:** T6 (rolling-summary in colony-prime `--compact` output); T4 (intra-phase midden threshold check added to build-wave.md wave completion)
**Avoids:** Building features on sparse data — T4 requires Phase 3's write-path expansion and T6 requires Phase 2-4's broad memory-capture coverage before they produce value

### Phase Ordering Rationale

- Phase 1 before everything: establishes the baseline and test infrastructure that makes all subsequent phases verifiable; without it, there is no way to distinguish a working fix from a coincidental passing test
- Phase 2 before 3 and 4: additive-only changes build confidence in the test infrastructure before touching existing code paths
- Phases 3 and 4 can be parallelized: they edit different playbook files (build-wave.md/continue-verify.md vs continue-advance.md) with no shared call sites or overlapping state
- Phase 5 last: T4 depends on Phase 3's write-path data; T6 depends on Phase 2-4's memory-capture coverage being established

### Research Flags

Phases with well-documented patterns (standard implementation, no additional research needed):
- **Phase 2:** `memory-capture "success"` pattern is established in existing MEM-02 blocks in build-wave.md; replicate the pattern, do not reinvent it
- **Phase 3:** `midden-write` call sites follow the builder failure path in build-wave.md Step 5.2 as a template; extend to Watcher/Chaos/verification paths using identical structure

Phases requiring closer attention during execution:
- **Phase 1:** Test infrastructure design — designing integration tests that accurately simulate a build/continue cycle without running the full AI agent loop requires understanding of the test fixture structure in `tests/integration/`
- **Phase 4:** Deduplication format alignment (Gap 1) cannot be confirmed by source reading alone; requires runtime verification against actual pheromones.json and CONTEXT.md data to confirm whether `contains()` substring matching works correctly for the two emission paths
- **Phase 5:** T4 intra-phase threshold timing — the sequence of midden-write calls earlier in a wave and the threshold check later in the same wave depends on file system flush behavior and jq read-after-write consistency; validate during execution

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All subcommands verified by direct source inspection of aether-utils.sh with exact line numbers; JSON schemas confirmed against live data files; no new dependencies |
| Features | HIGH | Six table-stakes features derived from direct gap analysis of playbook source; differentiator features validated against peer-reviewed research on agent memory systems (Reflexion, Voyager, LangMem, Hierarchical Procedural Memory) |
| Architecture | HIGH | All playbook steps read directly; integration map documents what exists vs what is missing at each call site; data flow verified against actual subcommand implementations; component boundaries are clear |
| Pitfalls | HIGH | Pitfalls derived from direct inspection of existing playbook patterns, prior milestone audit notes confirming memory is empty in real-world use despite wiring existing, and established multi-agent failure taxonomy research |

**Overall confidence: HIGH**

### Gaps to Address

- **Deduplication format alignment (Gap 1):** STACK.md notes the dedup check in continue-advance.md Step 2.1b may work correctly as-is because `contains()` does substring matching. This requires runtime verification in Phase 1 against actual pheromones.json and CONTEXT.md data — source reading alone is insufficient to confirm correct behavior.

- **Instinct confidence calibration formula (T3):** The formula `min(0.9, 0.7 + (obs_count - 1) * 0.05)` is a recommendation derived from research patterns on recurrence-based confidence, not an empirically validated value. After Phase 4 ships T3, monitor whether instinct confidence values in practice produce meaningful differentiation between single-observation and multi-observation instincts.

- **Intra-phase threshold timing (T4):** Whether T4's mid-wave threshold check can reliably read midden data written earlier in the same wave depends on file system flush timing and jq read-after-write consistency on macOS. This needs explicit validation during Phase 5 execution, not assumed.

---

## Sources

### Primary (HIGH confidence — direct source inspection)
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` — direct source inspection of all integration subcommands: memory-capture (line 5402), midden-write (line 8211), instinct-create (line 7252), pheromone-write (line 6774), context-update (line 2763), midden-recent-failures (line 9581)
- `/Users/callumcowie/repos/Aether/.aether/docs/command-playbooks/` — all 9 split playbooks read directly; integration map derived from actual step content, not documentation
- [Reflexion: Language Agents with Verbal Reinforcement Learning](https://arxiv.org/abs/2303.11366) — signal emission must happen at decision point, not in batch post-hoc sweep; sliding window memory pattern
- [Learning Hierarchical Procedural Memory for LLM Agents](https://arxiv.org/pdf/2512.18950) — Bayesian recurrence threshold for procedure promotion; confidence-gated activation; contrastive refinement on failure

### Secondary (MEDIUM confidence)
- [LangMem Conceptual Guide](https://langchain-ai.github.io/langmem/concepts/conceptual_guide/) — active vs background memory paths; multi-factor ranking (recency + frequency + importance)
- [Meta-Policy Reflexion](https://arxiv.org/abs/2509.03990) — predicate-style rules with confidence scores; episodic-to-policy-level memory promotion
- [Multi-Agent System Failure Analysis (MAST)](https://arxiv.org/pdf/2503.13657) — 41-86.7% failure rates in production MAS; topology-over-agents principle; silent failure propagation
- [Mastering Confidence Scoring in AI Agents](https://sparkco.ai/blog/mastering-confidence-scoring-in-ai-agents) — 0.8 execution threshold; RL-based dynamic confidence calibration
- `.planning/PROJECT.md` and `.planning/MILESTONES.md` — v1.0/v1.1 milestone audit; real-world confirmation that memory is empty despite wiring existing

### Tertiary (LOW confidence — patterns only)
- [Memory in the Age of AI Agents survey](https://arxiv.org/abs/2512.13564) — memory type taxonomy (episodic → semantic → procedural); consolidation pathways
- [Why Your Multi-Agent System is Failing](https://towardsdatascience.com/why-your-multi-agent-system-is-failing-escaping-the-17x-error-trap-of-the-bag-of-agents/) — error amplification topology; closed-loop suppression; 17x error trap

---
*Research completed: 2026-03-14*
*Ready for roadmap: yes*
