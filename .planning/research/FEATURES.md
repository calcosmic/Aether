# Feature Research: Aether v2.1 Production Hardening

**Domain:** Multi-agent AI development orchestration system -- production readiness
**Researched:** 2026-03-23
**Confidence:** HIGH (Oracle audit at 82% confidence with 55 findings, codebase analysis, ecosystem research)

---

## Context

Aether v2.0 shipped with 22 agents, 44 commands, 28 skills, a pheromone signal system, and cross-colony wisdom sharing. An Oracle audit (12 iterations, 55 findings) exposed the gap between "impressive demo" and "trusted production tool": 338 silent error suppression instances, 43% dead code, state desync risks, shallow planning, and documentation that describes aspirational behavior rather than implemented behavior.

The user reports that results feel "small/incremental" because planning lacks depth -- the system decomposes goals quickly but does not research *before* building, leading to naive implementations that miss context.

This features research focuses on: What bridges that gap? What does production-ready mean in this domain? What must change for users to trust Aether with real work?

---

## Feature Landscape

### Table Stakes (Users Expect These)

Missing any of these = users lose trust and revert to running Claude manually.

| # | Feature | Why Expected | Complexity | Pain Point |
|---|---------|--------------|------------|------------|
| T1 | **Error visibility -- no silent failures** | When a tool fails silently 338 times, users get wrong results with no indication why. Production tools surface errors, they do not hide them. | HIGH | Oracle Q3: Three-layer error silence creates invisible failures. Memory pipeline death is invisible to orchestrator. Users cannot trust a system that hides its own failures. |
| T2 | **State checkpoint and rollback** | LangGraph checkpoints state at every graph step. Users expect to recover from bad phases -- not lose everything. Autopilot chains 5+ phases with no rollback; a mid-run corruption is total loss. | MEDIUM | Oracle Q5 Rec 1+7: No COLONY_STATE.json backup before builds. No git checkpoint before autopilot phases. Subtly bad work that passes gates contaminates subsequent phases. |
| T3 | **Planning depth -- research before building** | Research shows structured plans achieve 61% first-attempt success vs 23% for ad-hoc prompts (3.2x improvement). Current planning runs max 8 scout iterations but scouts only look at the codebase -- they do not research libraries, patterns, or approaches. | HIGH | User's core complaint: "results feel small/incremental." Planning generates phases with 2-sentence task descriptions. No per-phase research means builders lack context for good decisions. |
| T4 | **Verification that catches lies** | Schema-only validation means a Builder can claim "completed" while tests actually fail. Demo tools validate structure; production tools validate truth. | MEDIUM | Oracle Q3: Worker response validation is schema-only, not semantic. The system's primary hallucination vector. Existing TDD evidence gate pattern proves this is solvable. |
| T5 | **Documentation matches behavior** | 6 confirmed instances where docs describe aspirational behavior, not actual behavior. Users who rely on docs to understand guarantees get burned. | LOW | Oracle Pattern 1: "Rolling summary highest priority" in CLAUDE.md but code trims it FIRST. "Security gate" label oversells check-antipattern's 6-pattern coverage. 178 subcommands documented as 125. |
| T6 | **Dead code removal** | 76 subcommands (43%) are never invoked. 11,272 lines in a single file. Every change risks touching dead code and creating false coupling. Production codebases are lean. | MEDIUM | Oracle Q1: Dead code categories include semantic search (6), swarm display (10), learning display (8), spawning diagnostics (5). Removing unused code reduces aether-utils.sh by 15-20%. |
| T7 | **Memory pipeline resilience** | A single corrupted JSON file permanently disables all learning (5 downstream steps die silently). Production systems detect AND recover -- not just detect. | MEDIUM | Oracle Q4 Finding 12 + Q5 Rec 8: learning-observations.json corruption kills the entire memory-capture pipeline. Callers wrap with `2>/dev/null || true`. Detection without remediation. |
| T8 | **Autopilot state consistency** | Autopilot tracks state in run-state.json separately from COLONY_STATE.json. If one updates and the other does not, they desync with no detection or reconciliation. | LOW | Oracle Q5 Rec 9: Desync manifests as the system believing it is on a different phase than reality. Combined with no rollback (T2), this is a "quiet catastrophe" failure mode. |
| T9 | **Type-safe data operations** | String-typed confidence values silently exclude valid hive wisdom from retrieval. Colonies re-learn patterns they should already know because jq comparison returns false for string vs number. | LOW | Oracle Q3 Finding 7 + confirmed by midden evidence + REDIRECT signal. A 5-line fix with outsized reliability impact. |

### Differentiators (Competitive Advantage)

Features that make Aether genuinely better than running Claude Code manually. Not required, but these are why someone would use Aether.

| # | Feature | Value Proposition | Complexity | Notes |
|---|---------|-------------------|------------|-------|
| D1 | **Per-phase research loop** | Before building each phase, spawn a researcher that investigates the specific domain: reads docs, checks patterns, understands the libraries involved. This is what GSD does that Aether does not -- and it is the primary reason GSD's output quality is higher. Planning depth is the single highest-leverage differentiator. | HIGH | Research shows "Plan-and-Execute" pattern where a capable model creates strategy reduces costs by 90% compared to frontier models doing everything. Aether's current scout does codebase exploration but not domain research. |
| D2 | **Context trimming transparency** | When colony-prime trims context to fit budget, workers receive silently truncated context. Adding a single notice line ("Context trimmed: rolling-summary, phase-learnings removed") lets workers know what they are missing and query for more if needed. No competitor does this. | LOW | Oracle Q5 Rec 5: ~30 characters added to trimmed output. The information already exists in `cp_budget_trimmed_list` -- it just is not routed to workers. |
| D3 | **Agent fallback visibility** | When a specialized agent falls back to general-purpose (losing 200+ lines of discipline, format, and boundary constraints), log it as a midden entry and warn the operator. Currently invisible. | LOW | Oracle Q5 Rec 6: A general-purpose agent with "You are a Chaos Ant - resilience tester" has none of the full agent's discipline. Users deserve to know when the system runs at reduced capability. |
| D4 | **Evidence-based verification** | Extend existing TDD evidence gate pattern: (a) capture test runner exit code during build-verify and cross-reference against Watcher's `verification_passed` claim, (b) verify Builder's `files_created` list against actual filesystem. If claims contradict evidence, reject the response. | MEDIUM | Oracle Q5 Rec 2: Follows the pattern already proven in continue-gates.md Step 1.10. Two targeted additions (~20 lines each) that close the semantic verification gap. |
| D5 | **Cross-colony learning that actually works** | Fix the type coercion bug so hive wisdom retrieval works correctly, then build on the existing multi-repo confidence boosting. When a pattern is confirmed across 4+ repos, confidence reaches 0.95 -- this is unique in the ecosystem. CrewAI, LangGraph, AutoGen are all stateless. | LOW | The infrastructure exists and is well-designed. The bug (T9) prevents it from working. Once fixed, Aether has the only production-grade cross-session learning pipeline in the multi-agent space. |
| D6 | **Structured error triage** | Not all 338 error suppressions are bad. Categorize into: (a) Correct (optional/fallback paths), (b) Lazy (hiding real errors), (c) Dangerous (data-writing operations). Fix dangerous first, lazy second, leave correct alone. | HIGH | Oracle Q5 Rec 3: The largest effort but addresses the root cause behind multiple findings. The suggest-analyze ERR trap gap (200 lines without error trapping during builds) is the highest-priority target. |
| D7 | **Build output quality scoring** | Track quality metrics across builds: test pass rate, files created vs claimed, error count, lint violations. Surface as a quality trend. This transforms "did it work?" into "how well is it working over time?" | MEDIUM | No competitor provides build-over-build quality trends. Aether already has the Auditor and Measurer agents -- wire their output into persistent quality tracking. |
| D8 | **First-user onboarding polish** | The gap between "npm install -g aether-colony" and "first successful build" is where users are won or lost. Validate the entire flow works without errors, confusion, or silent failures. | MEDIUM | Research shows 80% of production effort is refinement, not initial development. The onboarding flow exists but was not validated end-to-end with the v2.0 changes (skills system, oracle distribution). |

### Anti-Features (Do NOT Build)

Features that seem valuable but create complexity without proportional value.

| # | Anti-Feature | Why Requested | Why Problematic | What to Do Instead |
|---|--------------|---------------|-----------------|-------------------|
| A1 | **Web/TUI dashboard** | "Visualize colony state in a browser" | Massive scope increase. Requires server, frontend framework, state synchronization -- all orthogonal to core value of shipping code. Research confirms CLI tools dominate AI agent development workflows in 2026. | Keep ASCII dashboards (`pheromone-display`, `swarm-display`, `/ant:status`). They work in the terminal where users already are. |
| A2 | **Real-time inter-worker messaging** | "Workers should talk to each other during execution" | Claude Code subagents cannot communicate mid-execution (confirmed by platform constraints). Building this requires an architecture rewrite for uncertain platform support. | Keep Queen-as-coordinator pattern with `prompt_section` injection at spawn. The current one-way coupling is the "healthiest coupling pattern in the system" (Oracle Q2 Finding 4). |
| A3 | **More agent types** | "Add a Debugger agent, a Deployer agent" | 22 agents already strain context windows. Oracle found 43% dead code -- adding more increases surface area without fixing integration gaps. The fix ratio is rising (33.8% to 45.8%), meaning the error surface grows faster than repairs. | Fix integration between existing 22 agents first. The current agents cover all needed castes. |
| A4 | **Complex pheromone decay algorithms** | "Signals should decay based on relevance, not just time" | Over-engineering. Relevance-based decay requires understanding intent, which is unsolved. The current linear decay with configurable half-lives (15/30/45 days) works. | Keep current decay model. Fix the existing bugs (type coercion in hive-read) before adding sophistication. |
| A5 | **Full aether-utils.sh rewrite** | "Rewrite from scratch in a better language" | 11,272 lines of working code with 530+ tests. A rewrite risks losing edge-case handling that was earned through real failures (17 midden entries). The Oracle found the architecture is sound -- risks are in specific implementation details. | Extract dead code into optional modules. Modularize the monolith incrementally. Keep the test suite green throughout. |
| A6 | **Automatic documentation generation** | "Auto-generate docs from code" | The problem is not generating docs -- it is keeping docs accurate. Auto-generation creates a false sense of currency. Documentation that describes aspirational behavior (the current problem) would persist if generated from comments that describe aspirations. | Manual documentation pass after all fixes are complete. Docs describe the system *as it is*, not as code comments say it should be. |
| A7 | **Performance optimization (caching, lock backoff)** | "Make state access faster" | Not blocking anything. COLONY_STATE.json is ~1.3KB. Lock contention is theoretical (Oracle reached "operational ceiling" -- static analysis cannot determine actual collision rate). Premature optimization diverts from reliability work. | Defer unless runtime monitoring (which does not exist yet) reveals actual bottlenecks. |
| A8 | **Multi-repo colony coordination** | "Coordinate work across multiple repositories" | Requires fundamental architecture changes. The hub system (registry, hive, eternal) provides cross-colony wisdom but not coordination. Building coordination requires distributed state management -- a different class of problem. | Cross-colony *wisdom sharing* (which exists) is sufficient. Coordination is a future major version concern. |

---

## Feature Dependencies

```
[T1] Error visibility
    |-- enables --> [D6] Structured error triage (must see errors before categorizing them)
    |-- enables --> [T7] Memory pipeline resilience (must detect failures to add recovery)

[T2] State checkpoint and rollback
    |-- enables --> [T8] Autopilot state consistency (reconciliation needs backup to rollback to)
    |-- enables --> [D7] Build quality scoring (need checkpoint to measure quality delta)

[T3] Planning depth (per-phase research)
    |-- requires --> [T6] Dead code removal (research must understand what code actually matters)
    |-- enhances --> [D1] Per-phase research loop (deeper planning enables per-phase research)

[T4] Verification that catches lies
    |-- enhances --> [D4] Evidence-based verification (same domain, different scope)
    |-- requires --> [T1] Error visibility (verification must report clearly, not silently)

[T5] Documentation matches behavior
    |-- requires --> ALL other features complete (document the system as-is, not as-planned)

[T6] Dead code removal
    |-- independent -- can start immediately
    |-- enhances --> [T1] Error visibility (fewer code paths = fewer error hiding spots)

[T7] Memory pipeline resilience
    |-- requires --> [T1] Error visibility (need error surfacing before adding recovery)
    |-- enhances --> [D5] Cross-colony learning (resilient pipeline = reliable wisdom promotion)

[T9] Type-safe data operations
    |-- independent -- can start immediately (5-line fix)
    |-- enables --> [D5] Cross-colony learning (hive-read must work correctly)

[D1] Per-phase research loop
    |-- requires --> [T3] Planning depth (research infrastructure must exist)

[D2] Context trimming transparency
    |-- independent -- can start immediately (~30 chars)

[D8] First-user onboarding polish
    |-- requires --> [T5] Documentation matches behavior
    |-- requires --> [T6] Dead code removal (clean package)
```

### Dependency Notes

- **T1 (Error visibility) is the critical enabler.** Three other features depend on it. Without seeing errors, you cannot categorize them (D6), recover from them (T7), or verify truthfully (T4). This must come first.
- **T5 (Documentation) must come last.** Every other change invalidates documentation. Updating docs mid-work creates double-work.
- **T9 and D2 are quick wins with zero dependencies.** Both can be done in parallel with anything else. T9 is a 5-line jq fix; D2 is ~30 characters added to colony-prime output.
- **T2 (State checkpoint) unlocks autopilot reliability.** Without rollback capability, T8 (state reconciliation) and D7 (quality scoring) cannot meaningfully recover from bad states.
- **D1 (Per-phase research) is the highest-leverage differentiator** but requires T3 (planning depth infrastructure) first. This is the feature that transforms Aether from "fast but shallow" to "thorough and reliable."

---

## MVP Definition

### Immediate Wins (can ship in first sprint)

These are independent, low-effort, high-impact fixes identified by the Oracle audit:

- [x] **T9: Type coercion in hive jq filters** -- 5-line fix, confirmed bug with midden evidence. Unblocks cross-colony learning.
- [x] **D2: Context trimming notification** -- ~30 characters. Information exists but is not routed to workers.
- [x] **D3: Agent fallback visibility** -- ~10 lines. Log degradation to midden and warn operator.
- [x] **T8: Autopilot state reconciliation** -- ~10 lines. Compare run-state.json phase with COLONY_STATE.json phase at loop start.

### Foundation Layer (must complete before other work)

- [ ] **T2: State checkpoint before builds** -- `cp COLONY_STATE.json COLONY_STATE.json.phase-N.bak` before each build-wave. Keep last 3. Near-zero cost.
- [ ] **T1: Error visibility triage** -- Categorize 338 error suppressions. Fix dangerous ones first (data-writing operations). Fix the suggest-analyze ERR trap gap.
- [ ] **T6: Dead code extraction** -- Move 76 unused subcommands into optional modules. Reduce aether-utils.sh by 15-20%.

### Core Quality (the differentiating work)

- [ ] **T3+D1: Planning depth with per-phase research** -- The single highest-impact feature. Add research step before each build phase. Scouts investigate not just codebase but domain knowledge, library docs, patterns.
- [ ] **T4+D4: Verification evidence chain** -- Cross-reference test exit codes against Watcher claims. Verify claimed files exist. Reject fabricated responses.
- [ ] **T7: Memory pipeline circuit breaker** -- If learning-observations.json is corrupted, reset to template, log midden entry, retry. Transforms permanent silent failure into recoverable event.

### Polish (after core is solid)

- [ ] **D6: Structured error triage (remaining instances)** -- Address lazy suppression after dangerous ones are fixed.
- [ ] **D7: Build quality scoring** -- Wire Auditor and Measurer output into persistent quality tracking.
- [ ] **D8: First-user onboarding validation** -- End-to-end test of install through first successful build with v2.1 changes.
- [ ] **T5: Documentation accuracy pass** -- Update all docs to match reality. Must come last.

### Deferred (future milestones)

- [ ] **A2: Inter-worker communication** -- Requires platform changes (Claude Code subagent communication).
- [ ] **A8: Multi-repo coordination** -- Fundamentally different architecture challenge.
- [ ] **A7: Performance optimization** -- Defer until runtime monitoring reveals actual bottlenecks.

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Risk if Skipped | Priority |
|---------|-----------|---------------------|-----------------|----------|
| T9: Type coercion fix | HIGH | TRIVIAL (5 lines) | Cross-colony learning broken | P0 |
| D2: Trim notification | MEDIUM | TRIVIAL (30 chars) | Workers operate blind | P0 |
| D3: Fallback visibility | MEDIUM | LOW (10 lines) | Silent degradation | P0 |
| T8: State reconciliation | MEDIUM | LOW (10 lines) | Quiet autopilot desync | P0 |
| T2: State checkpoint | HIGH | LOW (3 lines/site) | Total loss on corruption | P1 |
| T1: Error visibility | HIGH | HIGH (audit 338) | All other fixes unreliable | P1 |
| T6: Dead code removal | HIGH | MEDIUM | Maintenance burden grows | P1 |
| T3+D1: Planning depth | HIGH | HIGH | Users stay disappointed | P1 |
| T4+D4: Verification | HIGH | MEDIUM (20 lines/check) | Hallucination vector open | P1 |
| T7: Memory resilience | HIGH | LOW (15 lines) | Permanent silent learning loss | P1 |
| D6: Error triage (full) | MEDIUM | HIGH (338 instances) | Error surface keeps growing | P2 |
| D7: Quality scoring | MEDIUM | MEDIUM | No quality trends | P2 |
| D8: Onboarding polish | HIGH | MEDIUM | New users bounce | P2 |
| T5: Documentation | HIGH | LOW | Trust gap persists | P2 (must be last) |

**Priority key:**
- P0: Do immediately -- independent, trivial effort, outsized impact
- P1: Foundation + core quality -- blocks user trust and differentiation
- P2: Polish -- important but depends on P0+P1 completion

---

## Competitor Feature Analysis

| Feature | CrewAI | LangGraph | AutoGen | Claude Code (manual) | Aether (current) | Aether (v2.1 target) |
|---------|--------|-----------|---------|---------------------|-------------------|----------------------|
| **State checkpointing** | None | Full (every graph step, PostgresSaver) | Basic | Git only | None (total loss risk) | Per-phase backup + git tag |
| **Error visibility** | Basic logging | Structured tracing | Conversation history | Terminal output | 338 silent suppressions | Categorized: dangerous/lazy/correct |
| **Planning depth** | Role-based (shallow) | Graph definition (manual) | Conversational | User-driven | Scout + Route-Setter (codebase only) | Per-phase research + domain investigation |
| **Verification** | Schema only | Human-in-the-loop | Schema only | User judgment | Schema only | Evidence-based (exit codes, file existence) |
| **Cross-session learning** | None (stateless) | Checkpoints (no learning) | None | CLAUDE.md only | Instinct pipeline + hive (buggy) | Fixed hive + type coercion + circuit breaker |
| **Dead code management** | N/A (library) | N/A (library) | N/A (library) | N/A | 43% dead code (76 subcommands) | Modularized, optional modules |
| **Observability** | Basic | OpenTelemetry integration | Basic | None | Midden + activity log (fire-and-forget) | Structured logging, fallback visibility |
| **Context management** | Static prompts | State channels | Message history | Context window | Colony-prime with budget (trimming silent) | Trim notification, per-phase context refresh |

**Key insight:** Aether's competitive advantage is *persistent learning across sessions*. No competitor has this. But it is currently broken (type coercion bug) and fragile (memory pipeline kills silently). Fixing reliability of the existing unique features is higher leverage than building new features that competitors already have.

**Second key insight:** Planning depth is where Aether loses to manual Claude Code usage. A human using Claude manually will naturally research before coding. Aether's planning skips research and jumps to task decomposition. Fixing this is the single highest-impact change for output quality.

---

## Sources

### Primary (HIGH confidence -- direct codebase analysis + Oracle audit)

- Oracle audit synthesis: `.aether/oracle/synthesis.md` -- 55 findings across 5 questions at 82% confidence
- Oracle research plan: `.aether/oracle/research-plan.md` -- 85% trust ratio (47/55 multi-source findings)
- Oracle gaps analysis: `.aether/oracle/gaps.md` -- All questions answered, 9 contradictions resolved
- Codebase concerns: `.planning/codebase/CONCERNS.md` -- Tech debt, known bugs, security considerations
- Current plan.md: `.claude/commands/ant/plan.md` -- Planning loop implementation
- Build playbooks: `.aether/docs/command-playbooks/build-*.md` -- Build execution flow

### External (MEDIUM confidence -- multiple sources agree)

- [AI Agents in Production: What Actually Works in 2026](https://47billion.com/blog/ai-agents-in-production-frameworks-protocols-and-what-actually-works-in-2026/) -- "80% of effort is refinement, not initial development"
- [Spec-Driven Verification for Autonomous Coding Agents](https://agent-wars.com/news/2026-03-14-spec-driven-verification-claude-code-agents) -- Independent verification, spec discipline matters more than architectural elaboration
- [Plan-and-Act: Improving Planning of Agents for Long-Horizon Tasks](https://arxiv.org/html/2503.09572v3) -- Structured plans achieve 61% first-attempt success vs 23% ad-hoc (3.2x)
- [Dapr Agents GA for Production Reliability](https://www.cncf.io/announcements/2026/03/23/general-availability-of-dapr-agents-delivers-production-reliability-for-enterprise-ai/) -- Durable workflows, state management for production
- [LangGraph Production: Checkpointing and Error Recovery](https://markaicode.com/langgraph-production-agent/) -- State snapshots at every step, PostgresSaver for production
- [Azure Agent Observability Best Practices](https://azure.microsoft.com/en-us/blog/agent-factory-top-5-agent-observability-best-practices-for-reliable-ai/) -- Structured logging, tracing, version correlation
- [OpenTelemetry AI Agent Observability](https://opentelemetry.io/blog/2025/ai-agent-observability/) -- Emerging standards for agent monitoring
- [Top AI Agent Frameworks 2026](https://www.shakudo.io/blog/top-9-ai-agent-frameworks) -- Framework comparison and production readiness criteria
- [AI Agent Planning Workflow: Plan-Export-Verify](https://www.loadsys.com/blog/ai-agent-planning-workflow-plan-export-verify/) -- Plan quality determines execution quality

### Previous Milestone (HIGH confidence -- our own validated work)

- v1.3 FEATURES.md: `.planning/research/FEATURES.md` (2026-03-19) -- Pheromone integration, learning pipeline, XML exchange, fresh install hardening

---

*Feature research for: Aether v2.1 Production Hardening*
*Researched: 2026-03-23*
