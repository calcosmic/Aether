# Feature Research: Integration Gap Fixes

**Domain:** Self-improving multi-agent colony system — integration loop wiring
**Researched:** 2026-03-14
**Confidence:** HIGH

## Context: What Is Already Built

This milestone fixes integration gaps in an existing system. Before mapping the feature landscape, here is the baseline — what is already implemented and tested:

| Feature | Built State | Gap |
|---------|------------|-----|
| Pheromone emit (pheromone-write) | Complete | Not called at all decision points |
| Pheromone read (pheromone-read) | Complete | Workers don't poll mid-task |
| Pheromone display/decay/expire | Complete | No gaps |
| Instinct create (instinct-create) | Complete | Not called when confidence threshold is crossed |
| Instinct read (colony-prime) | Complete | Instincts are primed into context but not always fresh |
| Midden failure write (midden-write) | Complete | Not called at all failure points |
| Midden failure query (midden-recent-failures) | Complete | Query results not always injected into worker context |
| Memory capture (memory-capture) | Complete | Called after learnings but not at decision or failure points inline |
| Learning observe (learning-observe) | Complete | Only called via memory-capture |
| Learning promote (learning-check-promotion) | Complete | User-review proposals shown but auto-path underused |
| Colony prime (colony-prime) | Complete | Called pre-build but not refreshed between waves |

The problem is connectivity: each subsystem works independently, but the wiring between them has gaps. The playbook instructions (continue-advance.md, build-wave.md, build-verify.md) need explicit wiring at the four integration points.

---

## The Four Integration Loops

These are the four specific wiring problems to solve:

### Loop 1: Decision → Pheromone

**Current state:** Decisions recorded in CONTEXT.md "Recent Decisions" table. continue-advance.md Step 2.1b reads this table and emits FEEDBACK pheromones — but only at phase-end during continue. Decisions made mid-phase (during build-wave execution) do not emit pheromones. Workers making approach changes during a task do not emit pheromones.

**What the research says:** Reflexion (NeurIPS 2023) established verbal reinforcement: converting sparse execution outcomes into natural language signals stored as episodic memory. Meta-Policy Reflexion (2025) extends this to predicate-style rules with confidence scores. The key finding: signal emission must happen at the decision point itself, not in a batch post-phase sweep, or the signal arrives too late to steer the workers who produced it. Voyager's skill library captures successful trajectories immediately on success — the same temporal principle.

**What is already wired:** Post-phase batch in continue-advance.md Step 2.1b (cap: 3 per run, dedup against existing signals).

**What is missing:** Inline emission at the moment of decision — specifically when (a) context-update decision is called during build, and (b) workers log approach changes to approach-changes.md.

### Loop 2: Learnings → Instincts with Confidence

**Current state:** continue-advance.md Steps 3, 3a, 3b extract instincts from phase patterns, midden error patterns, and success patterns. Confidence values are assigned manually by the AI: 0.7 for success patterns, 0.8 for error resolutions, 0.9 for user feedback. instinct-create handles deduplication and confidence boosting if an instinct already exists. The 30-instinct cap evicts lowest-confidence instincts.

**What the research says:** The 2025 hierarchical procedural memory literature (Learning Hierarchical Procedural Memory for LLM Agents) describes Bayesian selection: patterns promoted to instinct status when they demonstrate recurrence across multiple instances AND success metrics. High-confidence instincts get priority activation; low-confidence candidates remain dormant until more evidence arrives. Confidence scoring in AI agents (SparkCo, 2025) establishes the 0.8 execution threshold — below this, the system requests more information rather than acting. LangMem's multi-factor ranking uses recency, frequency, and importance together.

**What is missing:** The confidence scoring in continue-advance.md is pure AI judgment, not evidence-based. The learning-observations.json file tracks observation count and threshold_met, but this count is not used to gate or calibrate instinct-create confidence. An instinct created from a single observation gets the same base confidence (0.7) as one from an observation seen 5 times. The recurrence signal is being tracked but not consumed.

### Loop 3: Midden Failures → Behavioral Change

**Current state:** Failures are written to midden.json via midden-write at: worker failures (build-wave.md Step 5.2), chaos findings (build-wave.md Step 5.7), watcher failures (build-verify.md Step 5.8), Gatekeeper high-severity CVEs (continue-gates.md Step 1.8), Auditor high-severity quality issues (continue-gates.md Step 1.9). midden-recent-failures is queried per wave and injected into builder context as midden_context.

**What the research says:** Partnership on AI (2025) identifies that multi-agent behavioral change from failure tracking requires three components: (1) threshold detection (not all failures warrant behavioral change — recurrence matters), (2) behavioral modification at the right scope (individual worker context vs colony-wide redirect), and (3) explicit behavioral signal (logging failures is not the same as preventing recurrence). The 17x error trap analysis shows that without recurrence-gated behavioral change, failures in complex pipelines compound exponentially. The midden injection into worker context is the right pattern, but it only covers individual workers within a wave — colony-wide behavioral change (REDIRECT pheromone) only happens after continue, at threshold 3+ occurrences.

**What is missing:** The threshold for midden-to-REDIRECT emission in continue-advance.md Step 2.1c is correctly set at 3+ occurrences, but this only runs at phase-end. Within a single phase, a failure that occurs 3 times in different waves does not trigger a REDIRECT until continue runs. Also, the midden_context injection in build-wave.md is capped at last 5 failures — sufficient for individual worker context but not for pattern detection across a phase.

### Loop 4: Memory Capture at Decision and Failure Points

**Current state:** memory-capture is called: (a) in continue-advance.md Step 2.5 for each phase learning, (b) in build-wave.md Step 5.2 for worker failures, (c) in build-wave.md Step 5.7 for chaos resilience findings, (d) in build-verify.md Step 5.8 for watcher verification failures. This covers post-phase learnings and failure events.

**What the research says:** LangMem's memory management documentation shows two pathway architectures: active/conscious (captures during the event, adds latency) and background/subconscious (captures asynchronously after, no latency impact). For a multi-agent system running parallel workers, the background pathway is strongly preferred — the capture call should not block the worker. The memory survey (Memory in the Age of AI Agents, 2025) identifies the key integration pattern: memory capture must be wired to both success trajectories AND failure trajectories at the moment of the event, not in a post-hoc sweep.

**What is missing:** memory-capture is not called when context-update decision is executed (approach changes are logged to approach-changes.md markdown but not through the memory pipeline). The rolling-summary is updated by memory-capture but the rolling-summary content is not fed back into colony-prime for worker priming. Decision memory and failure memory are captured at different fidelity levels — failures get full memory-capture pipeline (pheromone + promotion check), decisions get only the CONTEXT.md table and a post-phase pheromone batch.

---

## Feature Landscape

### Table Stakes (Integration Loops Require These)

Features that are non-negotiable for the integration loops to function correctly. These are the direct wiring fixes.

| # | Feature | Why Expected | Complexity | Dependencies |
|---|---------|--------------|------------|-------------|
| T1 | **Decision-to-pheromone inline emission** | context-update decision already exists as an aether-utils.sh subcommand but does not call pheromone-write after writing the decision. Every call to `context-update decision` should emit a FEEDBACK pheromone. This is the same pattern already used in continue-advance.md Step 2.1b — it just needs to move from batch-post-phase to inline. | LOW | context-update subcommand, pheromone-write |
| T2 | **Approach-change memory capture** | Workers log approach changes to approach-changes.md (documented in build-wave.md Builder prompt). These represent valuable failure signals — an approach tried and abandoned. Currently the approach-changes.md file is never read or processed. The midden-write call should be added at approach-change log time. | LOW | midden-write, memory-capture, approach-changes.md |
| T3 | **Recurrence-calibrated instinct confidence** | learning-observe already tracks observation_count per content_hash. instinct-create should receive a confidence value derived from this count: base 0.7 for first observation, +0.05 per additional observation, capped at 0.9. This consumes the recurrence signal that is already being tracked but currently ignored in the confidence calculation. | LOW | learning-observe observation_count, instinct-create --confidence parameter |
| T4 | **Intra-phase midden threshold check** | continue-advance.md Step 2.1c correctly emits REDIRECT for categories with 3+ midden occurrences, but only runs at phase-end. Build-wave.md needs a midden threshold check after each wave's failures are logged. If any category reaches 3+ occurrences during the phase, emit REDIRECT immediately rather than waiting for continue. | MEDIUM | midden-recent-failures, pheromone-write REDIRECT, build-wave.md wave completion |
| T5 | **Failure-to-memory-capture at all failure points** | memory-capture is correctly called for worker failures and chaos findings. It needs to be called equally for: (a) Gatekeeper CRITICAL security findings (currently writes to midden but not memory pipeline), (b) Auditor CRITICAL quality findings (currently writes to midden but not memory pipeline), (c) approach changes (currently written to markdown file only). | LOW | memory-capture subcommand, continue-gates.md, build-wave.md |
| T6 | **Rolling-summary fed into colony-prime** | memory-capture calls `rolling-summary add` on every event. rolling-summary is a condensed activity log. colony-prime assembles the worker priming payload but does not include rolling-summary content. The priming payload should include the last N lines of rolling-summary to give workers awareness of recent activity. | LOW | rolling-summary, colony-prime subcommand |

### Differentiators (Meaningful Improvements Beyond Gap-Fill)

Features that go beyond closing the gaps to actively improve the intelligence of the integration loops.

| # | Feature | Value Proposition | Complexity | Notes |
|---|---------|-------------------|------------|-------|
| D1 | **Confidence decay for unverified instincts** | Instincts created from hypothesis-status learnings (status: "hypothesis" not "validated") should decay toward 0.5 over time. Currently instinct confidence is set once and only increases via reinforcement. A hypothesis that is never re-observed should lose confidence, not maintain it. This makes the instinct store self-correcting. | MEDIUM | instinct-create --confidence, learning-observe status field, requires scheduled decay logic |
| D2 | **Cross-phase midden pattern surfacing** | midden-recent-failures queries by recency (last N failures) not by recurrence. A failure category that appears once per phase across 5 phases would not trigger the 3-occurrence threshold within any single query window. Adding a cross-phase recurrence view that groups by category across all time surfaces patterns the current windowed query misses. | MEDIUM | midden.json schema, new midden-pattern-summary subcommand |
| D3 | **Instinct-to-pheromone echo at build start** | When colony-prime assembles the priming payload at build start, high-confidence instincts (>=0.85) relevant to the phase domain should echo as FOCUS pheromones. This converts durable learned knowledge into current-phase signals. Currently instincts are read via colony-prime but are not echoed as pheromones — the two subsystems operate in parallel rather than reinforcing each other. | MEDIUM | instinct-read --min-confidence, pheromone-write FOCUS, colony-prime, build-wave.md Step 5 |
| D4 | **User-feedback loop: FEEDBACK pheromone → instinct** | When a user emits a FEEDBACK pheromone via `/ant:feedback`, this is the highest-confidence learning signal (0.9 per the confidence guidelines). Currently FEEDBACK pheromones steer workers but do not automatically create instincts. A FEEDBACK pheromone with sufficient strength (>0.7) should create an instinct with confidence 0.9 and source "user:feedback". This closes the user → instinct loop. | LOW | pheromone-write FEEDBACK source tracking, instinct-create, feedback command playbook |

### Anti-Features (Do Not Build These)

Features that appear related but would create problems.

| # | Feature | Why Requested | Why Problematic | Alternative |
|---|---------|---------------|-----------------|-------------|
| A1 | **Automatic pheromone emission from every decision** | "Every decision should become a pheromone to maximize signal density." | Pheromone saturation degrades signal quality. If every minor micro-decision emits a pheromone, the pheromone store fills with low-value signals and the meaningful ones (user REDIRECT, midden error patterns) get buried. The current cap of 3 auto-pheromones per continue run exists for this reason. | Emit pheromones only from meaningful decision categories: architectural decisions, approach changes after failure, phase-level decisions. Not routine implementation choices. |
| A2 | **Real-time instinct updates during build** | "Builders should update instincts as they discover things." | Instincts are colony-level state (30-cap enforced). Workers updating instincts during build create concurrent write conflicts and would overflow the cap during a multi-worker parallel wave. The instinct store is designed for phase-boundary updates, not intra-phase mutation. | Workers capture learnings in their JSON output. Queen extracts instincts at continue time with proper deduplication and cap enforcement. |
| A3 | **Midden as blocking gate for every failure** | "If the midden shows 3+ failures of a type, block phase advancement automatically." | Midden failures include noise — chaos findings at medium/low severity, performance baselines, integration plans, refactoring notes. Blocking on midden count would create false gates. The midden is a record system, not a quality gate. | Block only on critical/security failures (already done via Gatekeeper and Auditor gates). Use midden for REDIRECT pheromone generation (already done at 3+ recurrences), not for blocking. |
| A4 | **Confidence score visible to users as a metric** | "Show average instinct confidence score on the /ant:status dashboard." | Confidence is an internal calibration mechanism. Exposing it as a user-facing metric encourages gaming it (artificially boosting confidence by recording the same observation repeatedly). The value is in behavioral influence, not in the number itself. | Status dashboard shows instinct count and domain distribution. Internal confidence drives colony-prime weighting without surfacing the number. |
| A5 | **Pheromone-to-instinct auto-conversion** | "Strong pheromones should automatically become instincts." | Pheromones and instincts are different abstractions: pheromones decay and are time-scoped; instincts are durable patterns. Auto-converting pheromones to instincts would persist what should expire. A FOCUS pheromone for "current security audit" is not a durable instinct about security. | The user-feedback loop (D4) handles the one case where a pheromone should become an instinct: high-strength user FEEDBACK signals a durable preference, not a temporary focus. |

---

## Feature Dependencies

```
[T1] Decision-to-pheromone inline emission
    +--extends--> context-update decision subcommand
    +--calls---> pheromone-write FEEDBACK (already implemented)
    +--enables-> [D3] Instinct-to-pheromone echo (same infrastructure)

[T2] Approach-change memory capture
    +--reads---> approach-changes.md (Builder worker output)
    +--calls---> midden-write (already implemented)
    +--calls---> memory-capture "failure" (already implemented)
    +--feeds---> [T4] Intra-phase midden threshold (adds to midden count)

[T3] Recurrence-calibrated instinct confidence
    +--reads---> learning-observe observation_count (already tracked)
    +--modifies-> instinct-create --confidence value (already a parameter)
    +--required-by-> [D1] Confidence decay (decay from evidence-based baseline)

[T4] Intra-phase midden threshold check
    +--reads---> midden-recent-failures (already implemented)
    +--calls---> pheromone-write REDIRECT (already implemented)
    +--placed-in-> build-wave.md wave completion (post-Step 5.2, 5.7)
    +--requires-> [T2] and [T5] to maximize signal quality

[T5] Failure-to-memory-capture at all failure points
    +--calls---> memory-capture "failure" (already implemented)
    +--placed-in-> continue-gates.md Steps 1.8, 1.9 (Gatekeeper, Auditor)
    +--placed-in-> build-wave.md approach-change logging
    +--feeds---> [T4] (more failure data = better threshold detection)

[T6] Rolling-summary fed into colony-prime
    +--reads---> rolling-summary (already maintained by memory-capture)
    +--extends-> colony-prime --compact output (already called at build start)
    +--required-by-> [D3] (prime payload must include recent context)

[D1] Confidence decay for unverified instincts
    +--requires-> [T3] (evidence-based baseline makes decay meaningful)
    +--modifies-> instinct-read (needs to apply decay on read)

[D2] Cross-phase midden pattern surfacing
    +--requires-> midden.json having sufficient history
    +--enables---> more accurate [T4] intra-phase threshold detection

[D3] Instinct-to-pheromone echo at build start
    +--requires-> [T6] (prime payload enrichment)
    +--requires-> [T1] (same pattern: read → emit → inform)
    +--reads---> instinct-read --min-confidence 0.85
    +--calls---> pheromone-write FOCUS

[D4] User-feedback → instinct auto-create
    +--reads---> pheromones.json for user:feedback signals
    +--calls---> instinct-create --confidence 0.9
    +--placed-in-> /ant:feedback command (or post-feedback hook)
```

### Dependency Notes

**T1 through T5 are independent of each other.** Each is a targeted wiring addition to an existing call site. They can be built in any order and shipped incrementally. No dependency chain gates them.

**T6 is a prerequisite for D3.** Colony-prime must include rolling-summary content before instinct echoing to the priming payload is meaningful.

**T3 is a prerequisite for D1.** Confidence decay from a non-evidence-based fixed value (0.7) would be arbitrary. Evidence-based confidence (recurrence count calibrated) gives decay a meaningful starting point.

**D2 (cross-phase midden patterns) enhances T4 but is not required for it.** T4 works with the windowed query; D2 makes it more comprehensive.

---

## MVP Definition

### Build First (This Milestone — Close the Core Gaps)

The minimum set to close the four integration loops. All are LOW complexity, all wire existing subcommands to existing call sites.

- [ ] **T1: Decision-to-pheromone inline emission** — Modify context-update decision to call pheromone-write FEEDBACK immediately. One bash call added to the subcommand or to the playbook instructions where context-update decision is called.
- [ ] **T2: Approach-change memory capture** — Add midden-write + memory-capture calls in the approach-change logging block of build-wave.md Builder prompt. Currently this block writes only to a markdown file.
- [ ] **T3: Recurrence-calibrated instinct confidence** — Add a learning-observations.json read before each instinct-create call in continue-advance.md Steps 3, 3a, 3b. Compute confidence as `min(0.9, 0.7 + (obs_count - 1) * 0.05)`.
- [ ] **T4: Intra-phase midden threshold check** — Add midden threshold check after each wave's failure processing (after Step 5.2 and Step 5.7 in build-wave.md). Emit REDIRECT if category reaches 3+ within the phase.
- [ ] **T5: Failure-to-memory-capture at all failure points** — Add memory-capture "failure" calls in continue-gates.md Steps 1.8 (Gatekeeper CRITICAL) and 1.9 (Auditor CRITICAL). Match the pattern already used in build-wave.md Step 5.2.
- [ ] **T6: Rolling-summary fed into colony-prime** — Modify colony-prime --compact to include the last 5 entries from rolling-summary in the prompt_section output.

**Why this set:** These six features collectively close all four integration loops. T1 closes Loop 1 (decision → pheromone). T2 + T5 close Loop 4 (memory capture at all failure points). T3 closes Loop 2 (recurrence calibration for instinct confidence). T4 closes Loop 3 (intra-phase midden threshold). T6 closes the feedback loop from memory capture back into worker context.

### Add After Validation (After Core Gaps Are Closed)

Features that enhance the integration once the core loops are verified working.

- [ ] **D4: User-feedback → instinct auto-create** — Wire the /ant:feedback command to call instinct-create with confidence 0.9 when the pheromone is FEEDBACK type and source is user. Completes the user → colony knowledge loop.
- [ ] **D3: Instinct-to-pheromone echo at build start** — After T6 is proven to work (rolling-summary in colony-prime), add instinct-read --min-confidence 0.85 before each build wave and emit matching high-confidence instincts as FOCUS pheromones.

**Trigger for adding:** When core gaps are closed and verified, and tests confirm the loops are functioning end-to-end.

### Future Consideration (After v1.2 Milestone)

Features that require more infrastructure work or empirical data.

- [ ] **D1: Confidence decay for unverified instincts** — Requires a scheduled decay pass or on-read decay calculation. Needs T3 first. Adds maintenance overhead. Defer until instinct store quality is measurable.
- [ ] **D2: Cross-phase midden pattern surfacing** — Needs schema extension and a new midden-pattern-summary subcommand. Valuable but not blocking the core loops.

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| T1: Decision-to-pheromone inline | HIGH | LOW | **P1** |
| T2: Approach-change memory capture | HIGH | LOW | **P1** |
| T3: Recurrence-calibrated confidence | HIGH | LOW | **P1** |
| T4: Intra-phase midden threshold | HIGH | MEDIUM | **P1** |
| T5: Failure capture at all points | MEDIUM | LOW | **P1** |
| T6: Rolling-summary in colony-prime | MEDIUM | LOW | **P1** |
| D4: User-feedback → instinct | HIGH | LOW | **P2** |
| D3: Instinct-to-pheromone echo | MEDIUM | MEDIUM | **P2** |
| D1: Confidence decay | LOW | MEDIUM | **P3** |
| D2: Cross-phase midden patterns | LOW | MEDIUM | **P3** |

**Priority key:**
- P1: Required for this milestone — closes the four integration loops
- P2: Enhances loops after core gaps are closed
- P3: Future milestone — needs empirical data or more infrastructure

---

## Comparison: What Comparable Systems Do

This is an integration milestone, not a net-new feature milestone. No direct competitor analysis applies. But the research patterns inform the correct approach:

| Integration Mechanism | Reflexion (NeurIPS 2023) | Voyager (2023) | Hierarchical Procedural Memory (2025) | LangMem (2025) | Aether Target |
|----------------------|--------------------------|----------------|--------------------------------------|----------------|---------------|
| Decision → Signal | Verbal reflection at episode end | Skill stored on success | Procedure promoted after recurrence | Active or background path | **T1: Inline at decision point** |
| Confidence calibration | Binary success/fail | Not explicit | Bayesian with recurrence + success metrics | Multi-factor: recency + frequency + importance | **T3: obs_count-based confidence** |
| Failure → Behavior change | Sliding window of reflections (1-3) | N/A | Contrastive refinement on failure | History-based deletion by utility | **T4: Threshold REDIRECT at 3+ occurrences** |
| Memory capture timing | Episode boundary | Immediate on success | Contrastive at failure, accumulate on success | Both: active (immediate) and background (async) | **T5+T6: Capture at event + feed forward** |

**Key finding from comparison:** Every system that works captures signals at the event, not in a batch post-hoc sweep. Aether already has batch capture (continue-advance.md). The gap is inline capture at the moment of occurrence.

---

## Sources

### Primary (HIGH confidence — official documentation and peer-reviewed research)
- [Reflexion: Language Agents with Verbal Reinforcement Learning](https://arxiv.org/abs/2303.11366) — verbal memory accumulation pattern, sliding window memory (Ω = 1-3), failure-to-signal at episode boundary
- [Meta-Policy Reflexion](https://arxiv.org/abs/2509.03990) — predicate-style rules with confidence scores, episodic-to-policy-level memory promotion
- [LangMem Conceptual Guide](https://langchain-ai.github.io/langmem/concepts/conceptual_guide/) — active vs background memory paths, multi-factor ranking (recency + frequency + importance), episodic schema
- [Learning Hierarchical Procedural Memory for LLM Agents](https://arxiv.org/pdf/2512.18950) — Bayesian recurrence threshold for procedure promotion, contrastive refinement on failure, confidence-gated activation
- Aether codebase (`.aether/aether-utils.sh`, `.aether/docs/command-playbooks/`) — baseline state of all existing subcommands and playbook wiring points

### Secondary (MEDIUM confidence — practice and empirical analysis)
- [How Memory Management Impacts LLM Agents](https://arxiv.org/html/2505.16067v2) — experience-following property, error propagation from stored noise, history-based deletion pattern
- [Better Ways to Build Self-Improving AI Agents](https://yoheinakajima.com/better-ways-to-build-self-improving-ai-agents/) — self-generated in-context examples, verification-gated adaptation (SICA pattern), Voyager skill persistence
- [Mastering Confidence Scoring in AI Agents](https://sparkco.ai/blog/mastering-confidence-scoring-in-ai-agents) — 0.8 execution threshold, RL-based dynamic confidence, calibration requirement
- [Multi-Agent System Failure Analysis (MAST)](https://arxiv.org/pdf/2503.13657) — 41-86.7% failure rates in production MAS, topology-over-agents principle
- [Why Your Multi-Agent System is Failing](https://towardsdatascience.com/why-your-multi-agent-system-is-failing-escaping-the-17x-error-trap-of-the-bag-of-agents/) — error amplification topology, closed-loop suppression

### Tertiary (LOW confidence — patterns only)
- [Memory in the Age of AI Agents survey](https://arxiv.org/abs/2512.13564) — memory type taxonomy (episodic → semantic → procedural), consolidation pathways
- [Mastering Memory Consistency in AI Agents](https://sparkco.ai/blog/mastering-memory-consistency-in-ai-agents-2025-insights) — intelligent decay and consolidation, recency + relevance pruning

---
*Feature research for: Aether v1.2 Integration Gap Fixes*
*Researched: 2026-03-14*
