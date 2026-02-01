# Project Research Summary

**Project:** Aether v2 — Claude-native multi-agent system
**Domain:** Claude-native multi-agent systems
**Researched:** 2026-02-01
**Confidence:** HIGH

## Executive Summary

Aether v2 is a **Claude-native multi-agent system** that represents a paradigm shift from traditional Python-based frameworks like AutoGen and LangGraph. Instead of code-based orchestration, Aether uses **prompts as code** and **JSON as state**, enabling autonomous agent spawning without external dependencies. The core innovation is agents that detect capability gaps and spawn specialists via Claude's Task tool, coordinated through a pheromone-based signal system implementing stigmergic communication.

The recommended approach is deliberately minimal: prompt files (`.md` with XML tags), JSON state persistence, and Claude's native Task tool for spawning. This stack is validated by official Anthropic documentation, leaked system prompts, and a working Aether implementation. The architecture consists of six layers: Command (prompts), State (JSON), Memory (triple-layer compression), Communication (pheromone signals), Orchestration (state machine), and Spawning (Task tool). Critical risks include context rot in long sessions, infinite spawning loops, and JSON state corruption from race conditions—all with proven mitigation strategies.

Key differentiators include autonomous spawning (no existing system supports this), pheromone-based stigmergy (unique to Aether), triple-layer memory (addresses LLM context limitations systematically), and meta-learning loop (agents learn which specialists work best). The system avoids common anti-patterns: predefined workflows, real-time dashboards, external vector databases, and Python orchestration layers—all problematic in practice.

## Key Findings

### Recommended Stack

Aether v2 uses a deliberately minimal, Claude-native stack validated by official Anthropic documentation and production usage.

**Core technologies:**
- **Claude Code CLI 2.0+**: Native execution environment providing Task tool, sub-agent spawning, agent skills, hooks—no external framework needed
- **Prompt files (`.md`)**: Agent behavior definition via `.claude/commands/` with XML tags for clear instruction boundaries—de facto standard for Claude-native systems
- **Task tool**: Official mechanism for autonomous sub-agent spawning with 5 agent types (general-purpose, Explore, Plan, claude-code-guide, statusline-setup)
- **JSON state**: Simple, human-readable persistence for colony state, pheromones, memory—Claude reads/writes natively, no databases required for this scope
- **Agent skills**: Domain expertise loaded on-demand via `.claude/skills/` folders (official feature, Oct 2025)

**Explicitly avoided:** Python async/await (Claude handles concurrency via Task tool), external vector DBs (overkill for this scope), Docker/Kubernetes (Claude Code has built-in sandboxing), Redis/PostgreSQL (JSON sufficient for prototype), pre-2025 multi-agent patterns (obsolete).

### Expected Features

**Must have (table stakes):**
- **Command System** — Users expect slash commands (`/ant:init`, `/ant:execute`) for any multi-agent system
- **State Persistence** — Work must survive context refreshes; JSON-based colony state is essential
- **Agent Roles/Castes** — At minimum: Planner, Executor, Verifier; single-type feels incomplete
- **Task Spawning** — Core multi-agent capability via Claude's Task tool; without this, it's just single-agent prompting
- **Phase-Based Execution** — Structured progress with checkpoints; pure emergence feels chaotic
- **Status Visibility** — Users need to see active agents, phases, progress via `/ant:status`
- **Basic Memory** — Working memory for current session; table stakes for any useful system

**Should have (competitive):**
- **Autonomous Agent Spawning** — Revolutionary differentiator: agents detect capability gaps and spawn specialists without human orchestration
- **Pheromone Signal System** — Stigmergic coordination via semantic signals (INIT, FOCUS, REDIRECT, FEEDBACK) with decay
- **Triple-Layer Memory** — Working (200k) → Short-term (2.5x compression) → Long-term with associative links
- **Voting-Based Verification** — Multiple verifiers with weighted voting; 13.2% improvement in reasoning
- **Meta-Learning Loop** — Bayesian confidence scoring for specialist selection improvement over time

**Defer (v2+):**
- **Long-Term Memory** — Persistent patterns across projects; substantial infrastructure needed
- **Semantic Communication Layer** — Intent-based protocol; adds complexity, needs research validation
- **Event-Driven Communication (Pub/Sub)** — Scaling to hundreds of agents; overkill for MVP
- **Multi-Colony Support** — Multiple projects with separate colonies; adds complexity to state management

### Architecture Approach

Claude-native multi-agent systems use **prompts as code** and **JSON as state**—a declarative paradigm where Claude interprets prompt files directly, not Python runtime. The architecture has six layers: Command Layer (prompt files define behaviors), State Layer (JSON persists system state), Memory Layer (triple-layer hierarchy with compression), Communication Layer (pheromone signals for coordination), Orchestration Layer (state machine for phase transitions), and Spawning Layer (Task tool for autonomous agent creation).

**Major components:**
1. **Command Layer** — Prompt files in `.claude/commands/ant/` define agent behaviors via XML-tagged sections (objective, process, context, reference)
2. **State Layer** — JSON files (COLONY_STATE.json, pheromones.json, memory.json) persist system state with git-tracked, diff-able format
3. **Memory Layer** — Triple-layer hierarchy: Working (200k in-context) → Short-term (10 sessions, 2.5x compressed) → Long-term (persistent patterns)
4. **Communication Layer** — Pheromone signals (INIT, FOCUS, REDIRECT, FEEDBACK) with half-life decay enable stigmergic coordination without central orchestrator
5. **Orchestration Layer** — State machine (IDLE → INIT → PLANNING → IN_PROGRESS → AWAITING_REVIEW → COMPLETED) manages phase transitions
6. **Spawning Layer** — Task tool spawns autonomous sub-agents when capability gaps detected; 5 agent types with inherited context

**Data flow:** User invokes `/ant:init "<goal>" → Command layer executes prompt → Orchestration sets state to INIT → Communication emits INIT pheromone → Spawning layer spawns Planner agent → State layer saves phase plan to JSON. Execution flow: `/ant:execute <phase_id>` → State layer loads phase → Orchestration sets IN_PROGRESS → Communication emits INIT pheromone → Spawning spawns Coordinator → Coordinator spawns specialists → Agents work autonomously, update JSON → Phase completion triggers state transition.

### Critical Pitfalls

**Top 5 pitfalls with prevention strategies:**

1. **Context Rot in Long-Running Sessions** — LLM attention degrades beyond 50-100 messages; agents forget instructions, contradict themselves, lose coherence. **Prevention:** Triple-layer memory with DAST compression (2.5x at phase boundaries), 20% context window budget (never exceed 40k tokens), signal decay with explicit renewal, context pruning (remove irrelevant history), context quality over quantity.

2. **Infinite Spawning Loops** — Agents spawn specialists who spawn more specialists recursively, exhausting quota. Research shows 1/3 of agents fall into infinite loops. **Prevention:** Global spawn depth limit (max 3 levels), per-phase spawn quota (max 10 specialists), spawn circuit breaker (auto-trigger after 3 failed spawns in 5 minutes), capability gap cache (don't spawn same specialist twice), spawn cost awareness, max iterations limit (5 for agent loops).

3. **JSON State Corruption from Race Conditions** — Multiple agents read/write same JSON simultaneously; last write wins, losing updates. Langflow has confirmed data corruption from this. **Prevention:** File locking (`fcntl.flock()` or `portalocker`), atomic write pattern (write temp file then rename), state versioning (optimistic locking with `_version` field), single-writer architecture (only one agent writes to shared state), JSONL for append-only logs, state validation after load.

4. **Prompt Brittleness and Complexity Explosion** — Prompts grow to 3000+ tokens with hardcoded logic, conditional branches, fragile instructions. Anthropic explicitly warns against this. **Prevention:** Prompt modules not monoliths (each agent role = separate file), structured outputs over prompt logic (JSON schema), prompt versioning (git-track, version in filenames), prompt testing (assert outputs match schema), tool calling over prompting (for logic), prompt length budget (max 500 tokens per agent), prompt linter for anti-patterns.

5. **Memory Bloat from Unbounded Working Memory** — Working memory grows unbounded, eventually exceeds context window or causes OOM. Research shows 10-100x degradation without forgetting mechanisms. **Prevention:** Tiered eviction policy (evict oldest/lowest-priority at 80% capacity), automatic compression (compress to short-term every N messages or M minutes), relevance scoring (rank entries by relevance, evict low-relevance first), working memory cap (hard limit 150k tokens), semantic summarization (DAST compress related entries), forgetting by design (FadeMem research shows 10-100x improvement).

**Moderate pitfalls:** Invisible state mutations (no audit trail), coordination token waste (agents talking instead of working), hallucination cascades (false confidence grows). **Minor pitfalls:** Hardcoded phase tasks (no adaptation to project), pheromone signal saturation (noise drowning signal).

## Implications for Roadmap

Based on combined research (feature dependencies, architecture layering, pitfall prevention), recommended phase structure:

### Phase 1: Foundation — State & Command Infrastructure
**Rationale:** All layers depend on JSON state persistence and prompt command structure. Critical pitfall #3 (JSON corruption) must be addressed first before multi-agent execution.
**Delivers:** State layer with file locking, command layer structure, basic state read/write patterns, atomic writes, state versioning
**Addresses:** Command System, State Persistence (from FEATURES.md table stakes)
**Avoids:** JSON state corruption from race conditions (PITFALLS.md #3)
**Stack elements:** JSON state, file locking (portalocker), atomic write pattern, prompt file structure

### Phase 2: Interactive Commands — Planning & Signals
**Rationale:** User needs command interface to set goals and see plans. Phase-based execution (table stakes feature) requires planning before memory system.
**Delivers:** `/ant:init` command, basic `/ant:status`, pheromone signal emission (INIT only), dynamic task generation (not hardcoded), signal filtering
**Addresses:** Phase-Based Execution, Status Visibility, Pheromone Signals (INIT only)
**Avoids:** Hardcoded phase tasks (PITFALLS.md #9), pheromone signal saturation (PITFALLS.md #10)
**Implements:** Command Layer (partial), Communication Layer (basic signals), Orchestration Layer (state machine foundation)
**Uses:** Task tool for spawning Planner agent

### Phase 3: Triple-Layer Memory — Compression & Retrieval
**Rationale:** Critical pitfall #1 (context rot) and #5 (memory bloat) must be addressed before autonomous spawning. Memory system needed before multi-agent execution to prevent context degradation.
**Delivers:** Working memory (200k token budget), DAST compression algorithm (2.5x ratio), short-term memory (10 sessions), automatic compression triggers, eviction policy, relevance scoring
**Addresses:** Basic Memory, Triple-Layer Memory (FEATURES.md differentiator)
**Avoids:** Context rot (PITFALLS.md #1), memory bloat (PITFALLS.md #5)
**Implements:** Memory Layer (full hierarchy)
**Uses:** JSON state for memory persistence, DAST compression (prompt-based)

### Phase 4: Autonomous Spawning — Core Differentiator
**Rationale:** Highest-value feature. Requires all previous layers (state, commands, memory) to function safely. Critical pitfall #2 (infinite spawning loops) must be prevented.
**Delivers:** Capability gap detection (in prompt logic), autonomous specialist spawning via Task tool, spawn depth limit (max 3), per-phase spawn quota (max 10), circuit breaker (3 failed spawns → cooldown), spawn cost tracking, basic agent castes (Mapper, Planner, Executor)
**Addresses:** Autonomous Agent Spawning (FEATURES.md core differentiator), Agent Roles/Castes (table stakes)
**Avoids:** Infinite spawning loops (PITFALLS.md #2)
**Implements:** Spawning Layer (Task tool integration), basic autonomous behavior
**Uses:** Task tool spawning, JSON state for spawn history, meta-learning data structure (foundation)

### Phase 5: Verification & Quality Assurance
**Rationale:** Prevents hallucination cascades (PITFALLS.md #8). Voting-based verification is differentiator that improves reliability significantly.
**Delivers:** Verifier caste, weighted voting logic, belief calibration, cross-agent validation, ground truth checks, source verification (all claims cite sources), confidence calibration
**Addresses:** Voting-Based Verification (FEATURES.md differentiator)
**Avoids:** Hallucination cascades (PITFALLS.md #8)
**Implements:** Spawning Layer (verifier agents), meta-learning foundation

### Phase 6: Advanced Features — Optimization & Learning
**Rationale:** Enhancements after core system works. Meta-learning makes spawning smarter over time.
**Delivers:** Meta-learning loop (Bayesian confidence scoring for specialist selection), remaining pheromone signals (FOCUS, REDIRECT, FEEDBACK), session persistence (pause/resume), prompt modularization (anti-brittleness), efficient handoffs (reduce token waste)
**Addresses:** Meta-Learning Loop (FEATURES.md differentiator), Remaining Pheromone Signals, Session Persistence
**Avoids:** Prompt brittleness (PITFALLS.md #4), coordination token waste (PITFALLS.md #7)
**Implements:** Meta-learning over full system, Communication Layer (full signal set)

### Phase Ordering Rationale

**Dependency-based ordering:** Phase 1 (state/commands) → Phase 2 (planning/signals) → Phase 3 (memory) → Phase 4 (spawning) → Phase 5 (verification) → Phase 6 (optimization). This order ensures each phase has required infrastructure. Spawning (Phase 4) requires state locking (Phase 1), planning interface (Phase 2), and memory compression (Phase 3) to avoid infinite loops, JSON corruption, and context rot. Verification (Phase 5) requires spawning to work first. Meta-learning (Phase 6) requires spawn history from Phase 4-5.

**Pitfall avoidance by design:** Phase 1 prevents JSON corruption before multi-agent execution. Phase 3 prevents context rot before spawning (long sessions during spawning would rot). Phase 4 includes circuit breakers to prevent infinite loops. Phase 5 prevents hallucination cascades. Phase 6 prevents prompt brittleness and token waste after core system validated.

**Feature delivery cadence:** Phase 1-2 deliver table stakes (commands, state, planning). Phase 3-4 deliver core differentiators (autonomous spawning, triple-layer memory). Phase 5-6 enhance quality and intelligence (verification, meta-learning). This validates autonomous spawning concept (MVP) before investing in optimization.

### Research Flags

**Phases likely needing deeper research during planning:**
- **Phase 3 (Triple-Layer Memory):** DAST compression algorithm needs prompt-based implementation—no existing Claude-native examples. Research optimal compression triggers (N messages vs M minutes) and relevance scoring heuristics.
- **Phase 4 (Autonomous Spawning):** Capability gap detection is core innovation—no existing systems to reference. Need research on prompt patterns for detecting "I can't do this, need specialist." Spawn depth limit (3) and quota (10) are heuristics—may need tuning based on testing.
- **Phase 5 (Verification):** Voting-based verification and belief calibration are research-backed but need prompt-based implementation patterns. Cross-agent validation logic needs design (how do verifiers independently check without repeating work?).

**Phases with standard patterns (skip research-phase):**
- **Phase 1 (State & Commands):** JSON state with file locking is standard pattern. Official Anthropic docs cover prompt file structure. Bash tool has atomic rename. HIGH confidence, implement directly.
- **Phase 2 (Planning & Signals):** Task tool spawning is well-documented. State machine orchestration is standard pattern. Basic pheromone signals (INIT) are simple JSON emission. MEDIUM confidence, proceed with implementation.
- **Phase 6 (Advanced Features):** Meta-learning uses Beta distribution (from research). Pheromone signal extensions add types, not new concepts. Session persistence is checkpoint/resume pattern. MEDIUM confidence, implement after core validated.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Official Anthropic documentation (sandboxing, agent skills), leaked system prompts (Task tool schema), working Aether implementation all validate prompt-based + JSON approach. No gaps. |
| Features | HIGH | Feature analysis from 5+ sources (AutoGen, LangGraph, CrewAI, Claude Code, Continue.dev) with consensus on table stakes. Differentiators validated by research papers (voting verification, pheromone coordination). Clear MVP definition. |
| Architecture | HIGH | Six-layer architecture validated by official Anthropic multi-agent research system docs. Prompt-as-code pattern proven in existing Aether codebase. Data flow patterns verified against Task tool schema. Minor gap: CDS architecture not officially documented (workaround: analyzed existing Aether implementation). |
| Pitfalls | HIGH | All 5 critical pitfalls verified by multiple sources: context rot (confirmed 2025), infinite loops (arXiv research), JSON corruption (Langflow issue #8791), prompt brittleness (Anthropic warning), memory bloat (FadeMem research). Prevention strategies tested in production systems. |

**Overall confidence:** HIGH

### Gaps to Address

**Minor gaps (non-blocking):**
- **DAST compression implementation:** Research confirms 2.5x compression ratio is optimal, but prompt-based implementation pattern doesn't exist. **Handle during Phase 3 planning:** Design compression prompt based on Anthropic's context engineering principles, test with sample sessions, iterate.
- **Capability gap detection heuristics:** No existing systems implement autonomous spawning—this is Aether's innovation. **Handle during Phase 4 planning:** Design prompt pattern for "I'm stuck, need specialist" detection, test with capability gaps, refine threshold for spawning.
- **Meta-learning confidence thresholds:** Beta distribution scoring is theoretically sound but production tuning unknown. **Handle during Phase 6 planning:** Start with conservative thresholds (alpha=1, beta=1), track spawn outcomes, adjust based on empirical data.

**No blocking gaps:** Research is sufficient for roadmap creation. All core decisions validated by high-confidence sources. Implementation details (compression prompts, detection thresholds) are engineering problems, not research gaps.

## Sources

### Primary (HIGH confidence)

**Official Anthropic documentation:**
- [Claude Code Sandboxing](https://www.anthropic.com/engineering/claude-code-sandboxing) — Security model, filesystem/network isolation, 84% permission prompt reduction
- [Agent Skills](https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills) — Official feature (Oct 16, 2025), load-on-demand pattern, skills architecture
- [Multi-Agent Research System](https://www.anthropic.com/engineering/multi-agent-research-system) — Official multi-agent architecture patterns, state machine orchestration
- [Effective Context Engineering](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents) — Context optimization strategies, compression techniques
- [Claude Code Best Practices](https://www.anthropic.com/engineering/claude-code-best-practices) — Slash command architecture, prompt organization

**Academic research (peer-reviewed):**
- [Voting or Consensus? Decision-Making in Multi-Agent Systems](https://arxiv.org/abs/2502.19130) — arXiv 2025, voting-based verification outperforms single agents
- [Multi-Agent Coordination Using Stigmergy](https://www.sciencedirect.com/science/article/abs/pii/S0166361503001234) — Peer-reviewed, 226 citations, validates pheromone approach
- [1/3 Agents Fall Into Infinite Loops](https://arxiv.org/html/2512.01939v1) — arXiv 2025, quantifies infinite loop problem
- [Memoria: Scalable Agentic Memory Framework](https://arxiv.org/html/2512.12686v1) — arXiv 2025, peer-reviewed, memory hierarchy patterns

**Verified issues:**
- [Langflow race condition data corruption](https://github.com/langflow-ai/langflow/issues/8791) — Confirmed JSON corruption bug
- [Claude Code context overflow](https://github.com/anthropics/claude-code/issues/6186) — Official issue, context window limits
- [OpenAI agents infinite recursion](https://github.com/openai/openai-agents-python/issues/668) — Confirmed spawn loop problem

### Secondary (MEDIUM confidence)

**Community analysis:**
- [Claude Skills Deep Dive](https://leehanchung.github.io/blogs/2025/10/26/claude-skills-deep-dive/) — Technical analysis of skills architecture (Oct 2025)
- [Understanding Claude Code's Full Stack](https://alexop.dev/posts/understanding-claude-code-full-stack/) — Evolution timeline, MCP → Claude Code → Plugins
- [Inside Claude Code Skills](https://mikhail.io/2025/10/claude-code-skills/) — Skills folder structure, SKILL.md format
- [Framework comparison: LangGraph vs CrewAI vs AutoGen](https://o-mega.ai/articles/langgraph-vs-crewai-vs-autogen-top-10-agent-frameworks-2026) — Competitor analysis (Jan 2026)
- [Why multi-agent systems fail](https://medium.com/@umairamin2004/why-multi-agent-systems-fail-in-production-and-how-to-fix-them-3bedbdd4975b) — Context rot, infinite loops, token waste (2025)

### Tertiary (LOW confidence)

**Unverified sources (needs validation):**
- [Claude Code system prompt leak](https://sankalp.bearblog.dev/my-experience-with-claude-code-20-and-how-to-get-better-at-using-coding-agents/) — Reverse-engineered Task tool schema (Dec 2025)
- [国外大神逆向了Claude Code](https://zhuanlan.zhihu.com/p/1943399204027373513) — Chinese reverse engineering (Aug 2025)

### Internal Aether Research

**Primary sources (HIGH confidence):**
- `.claude/commands/ant/*.md` — 15 prompt files, verified working patterns (XML tags, Task tool usage)
- `.aether/data/*.json` — 6 state files, proven schema (COLONY_STATE, pheromones, memory)
- `.aether/memory/meta_learning_demo.json` — Meta-learning implementation (Beta distribution confidence scoring)
- `.planning/codebase/CONCERNS.md` — Internal analysis of Python prototype pitfalls
- `.ralph/AUTONOMOUS_AGENT_SPAWNING_RESEARCH.md` — Internal spawning research
- `.ralph/MEMORY_ARCHITECTURE_RESEARCH.md` — Internal memory research

---
*Research completed: 2026-02-01*
*Ready for roadmap: yes*
