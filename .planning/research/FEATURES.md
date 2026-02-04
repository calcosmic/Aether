# Feature Research: Multi-Agent Colony System Hardening (v4.4)

**Domain:** Multi-agent colony system hardening -- recursive delegation, automated QA, learning persistence, conflict prevention, adaptive complexity
**Researched:** 2026-02-04
**Confidence:** MEDIUM-HIGH (strong ecosystem coverage; some claims WebSearch-only)

---

## Feature Landscape

### Table Stakes (Users Expect These)

Features that every production multi-agent system provides in 2025-2026. Missing these means Aether falls below the baseline set by AutoGen, CrewAI, LangGraph, and OpenAI Agents SDK. Organized by the five focus areas from field notes.

| Feature | Why Expected | Complexity | How Competitors Do It | Aether Status |
|---------|--------------|------------|----------------------|---------------|
| **File conflict prevention in parallel execution** | Every parallel coding system solves this; without it, agents overwrite each other's work | MEDIUM | Git worktrees (Cursor, VS Code), container isolation (Dagger), dependency-graph file locking (Claude multi-agent), explicit write-rule allowlists (Codex CLI, Swarm-IOSM) | MISSING -- field notes 10, 13 confirm same-file overwrites occurred |
| **Task delegation depth limits** | Recursive delegation without bounds causes runaway spawning and context explosion | LOW | AutoGen: `max_turns` parameter on agent loop. OpenAI Agents SDK: `max_turns` on Runner. CrewAI: `allow_delegation=False` on leaf agents. LangGraph: graph compilation catches cycles | MISSING -- currently workers prohibited from spawning entirely (opposite extreme) |
| **Persistent cross-session memory** | Users expect agents to remember what worked across sessions; every major framework has this | MEDIUM | CrewAI: 4-tier memory (short-term ChromaDB, long-term SQLite3, entity, contextual). LangGraph: customizable stores. AutoGen: shared memory with Mem0 integration. Amazon AgentCore: managed long-term memory service | PARTIAL -- memory.json persists learnings but no structured retrieval, no tiered storage, no cross-project learning |
| **Automated code review at build boundaries** | Standard in 2025-2026 agentic coding; users assume review happens after code generation | LOW-MED | Amazon Q: `/review` agent. Qodo: agentic code review. Zencoder: built-in verification after parallel execution. Every major coding agent now has review as a default step | PARTIAL -- Watcher caste exists but scores flat 8/10; no auto-spawned reviewer |
| **Adaptive complexity scaling** | Simple tasks should not require full orchestration machinery | MEDIUM | AutoGen: choose single-agent vs multi-agent based on task. CrewAI: simple vs hierarchical process selection. LangGraph: start simple, add complexity incrementally. Industry consensus: "agents are a last resort" | MISSING -- full colony overhead always applied (field note 31, 21) |
| **State persistence before context clear** | Claude Code context windows are finite; state must survive clears | LOW | Unique to Claude-native systems (not applicable to Python frameworks with persistent state). CDS solves this with explicit save-before-clear protocol | MISSING -- field note 5 marks this as non-negotiable |
| **Auto-continue / batch execution** | Users should not need to manually approve every phase transition for straightforward work | LOW | CrewAI: sequential process runs all tasks without gates. LangGraph: graph executes to completion by default. AutoGen: conversation runs until termination condition | MISSING -- field note 26; 10 manual interventions for 5 straightforward phases |
| **Error attribution to execution context** | Errors without context (which phase, which agent) are useless for debugging | LOW | All frameworks include agent/step context in error logs. LangGraph: full state trace per node. AutoGen: conversation history with agent attribution | MISSING -- field note 18; errors.json has no phase field |

### Differentiators (Competitive Advantage)

Features where Aether can lead the field. These either do not exist in competing systems or exist in inferior form. Aether's unique position as a Claude Code-native stigmergic system creates openings no Python framework can match.

| Feature | Value Proposition | Complexity | Competitor Gap | Aether Advantage |
|---------|-------------------|------------|---------------|-----------------|
| **Stigmergic conflict prevention** | Pheromone signals communicate file ownership without explicit locks; agents sense which files are "claimed" through signal strength | HIGH | All competitors use explicit mechanisms (locks, worktrees, containers, allowlists). None use emergent/stigmergic coordination for conflict avoidance | Pheromone system already exists; extending FOCUS signals to convey "I am working on X file" is natural and unique |
| **Multi-perspective colonization** | Multiple ants independently review codebase, then synthesize -- catches blind spots a single reviewer misses | MEDIUM | No competitor does multi-agent code exploration with synthesis. CrewAI and AutoGen have single-agent exploration | Colonizer caste already exists; running 3 colonizers with different sensitivity profiles and merging findings is differentiated |
| **Pheromone-driven planning** | Colony signals (FOCUS, REDIRECT) shape the plan before it is created, not after -- user intent propagates through emergent channels | MEDIUM | All competitors use explicit task assignment. No framework has signals-before-plan flow | Pheromone system is mature; making plan generation responsive to pheromone landscape is a natural extension |
| **Bayesian caste learning** | Colony tracks which specialist types succeed on which task types and improves spawning decisions over time | HIGH | AutoGen AutoBuild explores automatic agent selection but is experimental. CrewAI long-term memory stores outcomes but does not do Bayesian updating. No framework has caste-specific success tracking | Alpha/beta spawn tracking already implemented; extending to influence Phase Lead decisions is uniquely Aether |
| **Tiered learning (project + global)** | Learnings like "same-file tasks to one worker" are universal; project-specific patterns stay local | MEDIUM | CrewAI: memory is per-crew only. LangGraph: no built-in cross-project memory. No framework has a promotion mechanism from project to global | Field note 12 identifies the exact design; no competitor has project-to-global learning promotion |
| **Tech debt surfacing as colony output** | Persistent issues flagged across phases get aggregated into a tech debt report -- the colony produces organizational knowledge, not just code | LOW | No competitor generates tech debt reports. Watcher-type agents exist but do not aggregate cross-phase observations | Watcher already flags pre-existing issues; aggregation into a report is low-complexity, high-value |
| **Context-clear-safe architecture** | Entire system designed so user can `/clear` at any boundary with zero information loss -- unique to prompt-based agents | LOW | Not applicable to Python frameworks (persistent state by default). Other Claude Code tools do not have structured save-before-clear | Aether already persists to JSON; making every command end with "safe to clear" messaging is a UX polish |
| **Pheromone recommendations to user** | Colony suggests pheromone commands based on its own observations -- ants guide the Queen | MEDIUM | No competitor has agents recommending coordination signals to users. All frameworks treat users as commanders, not signal receivers | Inverts the typical user-commands-agent pattern; unique to stigmergic model |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem useful but create fragility, overhead, or maintenance burden in multi-agent systems. Based on real failures documented in industry (Cognition, Anthropic, LangChain).

| Anti-Feature | Why It Seems Good | Why It Fails | What To Do Instead |
|--------------|-------------------|--------------|-------------------|
| **Unlimited recursive delegation** | "Let ants spawn ants spawn ants -- true emergence!" | Context cost compounds at each nesting level. Anthropic found subagents misinterpret vague instructions and duplicate work. AutoGen research shows 79-100% of agents can be blocked by recursive attacks within 1.6 turns. Debugging recursive trees is orders of magnitude harder | Depth-limited delegation (max 2 levels). Phase Lead can spawn workers, but workers cannot spawn sub-workers. Hub-and-spoke with optional one-level depth for complex subtasks |
| **Full colony mode for all projects** | "Consistency -- same system for everything" | Field note 31 proves overhead exceeded value for a 21-task config project. Cognition (Devin team) argues multi-agent collaboration in 2025 produces fragile systems. Memory sprawl, context serialization, and coordination overhead dominate for simple tasks | Adaptive complexity: lightweight mode (single-agent with learning) for small projects, full colony for complex ones. Let the colonizer assess complexity and recommend mode |
| **Agent-to-agent direct messaging** | "More efficient than pheromone signaling" | Destroys stigmergic model. Creates point-to-point dependencies. Anthropic found direct messages cause duplication when agents don't coordinate on division of labor. Violates Aether's core differentiator | Keep pheromone-based indirect coordination. If an agent needs to communicate specific information, it emits a FEEDBACK pheromone that other agents sense through their sensitivity profiles |
| **Persistent daemon processes** | "Real-time monitoring of colony activity" | Breaks Claude-native model. Claude Code agents are prompt-based (execute once, exit). Background daemons require separate infrastructure, add complexity, and create state synchronization problems | Pull-based state checking via `/ant:status`. Activity log for async visibility. Event-driven file watching if needed |
| **Complex memory hierarchies (vector DB, embedding services)** | "Better retrieval, semantic search over learnings" | Adds external dependencies (Aether constraint: zero external deps). ChromaDB/Pinecone/Qdrant require running services. CrewAI users report SQLite3 long-term memory limits scalability. Over-engineered for project-scale memory | JSON file storage with structured keys. Claude's native semantic understanding handles relevance filtering at read time. Promote to `~/.aether/global/` for cross-project patterns |
| **Web dashboard for colony visualization** | "See the colony in real-time with a UI" | Breaks CLI-only constraint. Requires separate server process. Splits UX between terminal and browser. Maintenance burden of a frontend for a backend system | Rich CLI output with emoji, progress indicators, animated spinners (field note 15). `/ant:status` as the single source of truth. Colony activity visible in-terminal |
| **Automatic conflict resolution (merge)** | "Just auto-merge when two agents edit the same file" | Last-write-wins is exactly the bug from field note 13. Auto-merge requires understanding code semantics which LLMs get wrong at merge boundaries. Cursor 2.0 deliberately uses worktrees to avoid this entirely | Conflict prevention: tasks touching the same file go to the same worker. File-level claim signals via pheromones. Detect potential conflicts at planning time, not execution time |
| **Every agent type for every phase** | "Full caste representation ensures comprehensive coverage" | Field note 21: Builder was 56% of spawns, Scout dropped after 3 uses, Colonizer/Route-setter/Architect never spawned as workers. Spawning unused castes wastes context budget and adds noise | Let the Phase Lead select relevant castes per phase. Simple phases get a Builder only. Complex phases get full caste representation. Bayesian tracking informs caste selection over time |

---

## Feature Dependencies

```
CRITICAL PATH (must be built in order):

[Fix pheromone decay math] -----> [Pheromone-driven planning]
    (broken math makes signals meaningless)     (requires working signal strength)

[Fix activity log append] -----> [Tech debt report generation]
    (need cross-phase history)         (aggregates from full activity log)

[Fix error phase attribution] -----> [Auto-spawned reviewer/debugger]
    (need to know which phase errored)       (triggers on phase-specific errors)

[Same-file conflict prevention] -----> [Aggressive wave parallelism]
    (safety net)                             (unlocks safe parallelism)

[Adaptive complexity assessment] -----> [Lightweight colony mode]
    (need to measure complexity)              (route to appropriate mode)

ENHANCEMENT CHAINS (independent, can be built in parallel):

[Multi-ant colonization] -----> [Pheromone-first flow]
    (richer codebase knowledge)       (pheromones informed by colonization)

[Watcher scoring rubric] -----> [Phase Lead auto-approval]
    (meaningful quality signal)        (auto-approve when score > threshold)

[Per-project learning] -----> [Global learning tier] -----> [Learning promotion]
    (baseline)                    (cross-project store)         (mechanism)

[Context clear prompting] (independent -- no dependencies, immediate value)
[Auto-continue mode] (independent -- no dependencies, immediate value)
[Animated build indicators] (independent -- no dependencies, immediate value)
[Decision log wiring] (independent -- fix existing mechanism)
```

---

## MVP Definition

### Phase 1: Bug Fixes and Safety (pre-requisite for everything)

These must land first. Broken foundations invalidate every feature built on top.

1. **Pheromone decay math** -- FOCUS strength growing instead of decaying makes the entire signal system unreliable
2. **Activity log append** -- losing cross-phase history means learning extraction, tech debt reporting, and debugging all fail
3. **Error phase attribution** -- errors without phase context cannot drive automated review or debugging
4. **Decision log wiring** -- decisions made but not recorded means the colony cannot learn from its own reasoning
5. **Same-file conflict prevention** -- planning-time check that tasks touching the same file go to one worker

### Phase 2: Critical UX (user retention)

Without these, users abandon the system due to friction, not capability.

6. **Context clear prompting** -- non-negotiable per field note 5; every command saves state then tells user "safe to clear"
7. **Auto-continue mode** -- `/ant:continue --all` or similar; eliminates 10 manual interventions for straightforward projects

### Phase 3: Colony Intelligence (competitive advantage)

These make the colony genuinely smarter, not just functional.

8. **Adaptive complexity / lightweight mode** -- colonizer assesses project complexity and recommends colony vs lean mode
9. **Multi-ant colonization** -- 3 colonizers with different focus areas, synthesis step
10. **Aggressive wave parallelism** -- Phase Lead identifies parallelizable tasks and runs them simultaneously
11. **Watcher scoring rubric** -- rubric with specific criteria rather than default 8/10
12. **Phase Lead auto-approval** -- auto-approve plans below complexity threshold

### Phase 4: New Capabilities (expansion)

13. **Pheromone-first flow** -- colonize produces pheromone recommendations, user emits before planning
14. **Pheromone recommendations** -- after each build, suggest specific pheromone commands
15. **Auto-spawned reviewer/debugger** -- watcher auto-reviews after builder; debugger kicks in on test failure
16. **Tech debt report** -- aggregate persistent cross-phase issues into a post-project report
17. **Animated build indicators** -- spinners, color per caste, progress animation

### Phase 5: Design Decisions (architecture evolution)

18. **Tiered learning** -- per-project in `.aether/data/`, global in `~/.aether/global/`, promotion mechanism
19. **Depth-limited recursive delegation** -- Phase Lead can spawn workers who can spawn one level of helpers (max depth 2)
20. **Organizer/archivist ant** -- conservative stale-file detection at phase boundaries
21. **Pheromone user documentation** -- when/why to use each signal, practical scenarios, value proposition
22. **Colonizer visual output** -- restore emoji/progress markers in colonize command

---

## Feature Prioritization Matrix

| Feature | User Value | Impl Cost | Risk If Deferred | Priority | Phase |
|---------|-----------|-----------|------------------|----------|-------|
| Fix pheromone decay math | HIGH | LOW | System broken | P0 | 1 |
| Fix activity log append | HIGH | LOW | Data loss | P0 | 1 |
| Fix error phase attribution | HIGH | LOW | Debugging blind | P0 | 1 |
| Wire decision logging | MED | LOW | Learning gap | P0 | 1 |
| Same-file conflict prevention | HIGH | MED | Data corruption | P0 | 1 |
| Context clear prompting | HIGH | LOW | User abandonment | P0 | 2 |
| Auto-continue mode | HIGH | LOW | Friction death | P0 | 2 |
| Adaptive complexity / lightweight mode | HIGH | MED | Overhead kills small projects | P1 | 3 |
| Multi-ant colonization | MED | MED | Single-perspective blind spots | P1 | 3 |
| Aggressive wave parallelism | MED | MED | Slow execution | P1 | 3 |
| Watcher scoring rubric | MED | LOW | Flat quality signal | P1 | 3 |
| Phase Lead auto-approval | MED | LOW | UX friction | P1 | 3 |
| Pheromone-first flow | MED | MED | Weaker plans | P2 | 4 |
| Pheromone recommendations | MED | LOW | Missed teaching moments | P2 | 4 |
| Auto-spawned reviewer/debugger | MED | MED | Manual QA burden | P2 | 4 |
| Tech debt report | LOW | LOW | Knowledge loss | P2 | 4 |
| Animated build indicators | LOW | MED | Perception gap | P2 | 4 |
| Tiered learning | MED | HIGH | No cross-project learning | P2 | 5 |
| Depth-limited recursive delegation | MED | HIGH | Queen bottleneck at scale | P2 | 5 |
| Organizer/archivist ant | LOW | HIGH | Stale files accumulate | P3 | 5 |
| Pheromone user docs | MED | LOW | Onboarding gap | P2 | 5 |
| Colonizer visual output | LOW | LOW | Cosmetic | P3 | 5 |

---

## Competitor Feature Analysis

### Recursive Delegation

| Capability | AutoGen | CrewAI | LangGraph | OpenAI Agents SDK | Aether (current) | Aether (v4.4 target) |
|-----------|---------|--------|-----------|-------------------|-------------------|----------------------|
| Delegation model | Conversation-based; agents delegate via messages | Manager agent delegates to specialists | Graph nodes with Send API for dynamic workers | Handoff-based; agents hand off to specialists | Hub-and-spoke only; Queen spawns all workers | Depth-limited (max 2); Phase Lead spawns workers, workers can spawn one helper |
| Depth control | `max_turns` on agent loop | `allow_delegation=False` on leaf agents | Graph compilation catches infinite cycles | `max_turns` on Runner | Workers explicitly prohibited from spawning | Depth counter in spawn context; hard limit enforced by utility |
| Anti-loop protection | Termination conditions in GroupChat | Role scoping prevents delegation loops | Immutable graph after compilation | Agent loop exits when output type matched or no tool calls | N/A (no recursion) | Depth counter + spawn-check utility gate |
| Known weakness | Vulnerable to Corba recursive blocking attack (79-100% agents blocked) | Delegation type errors in hierarchical mode; documentation gaps | Debugging cyclic graphs is hard; non-determinism | Vague instructions cause subagent misinterpretation | Queen bottleneck; cannot delegate to specialists at sub-task level | Context cost at depth 2; need to verify quality of depth-2 outputs |

**Recommendation for Aether:** Implement depth-limited delegation (max 2). Phase Lead at depth 0 can spawn workers at depth 1. Workers at depth 1 can spawn a single helper at depth 2 for specific sub-problems (e.g., builder spawns scout for investigation). Workers at depth 2 cannot spawn. Enforce via `spawn-check` utility that reads depth from spawn context. This avoids the Queen bottleneck while preventing the context explosion and attack vulnerabilities seen in AutoGen.

### Conflict Prevention in Parallel Execution

| Capability | Git Worktrees (Cursor/VS Code) | Container Isolation (Dagger) | File Locking (Claude multi-agent) | Write Rules (Swarm-IOSM/Codex) | Aether (current) | Aether (v4.4 target) |
|-----------|-------------------------------|------------------------------|----------------------------------|-------------------------------|-------------------|----------------------|
| Isolation model | Each agent gets own worktree branch | Each agent runs in own container | Dependency graph analysis; lock conflicting files | Explicit allowlists per task defining writable files | None; same-file conflicts observed (field note 13) | Planning-time grouping: tasks touching same file assigned to same worker |
| Merge strategy | Manual review + merge of worktree branches | PR-like review of container outputs | Lock prevents concurrent access | Validator checks write boundaries | Last-write-wins (broken) | No merge needed; same-worker handles all changes to a file |
| Overhead | High (full repo copy per worktree) | Very high (container per agent) | Medium (dependency analysis) | Low (static rules) | None | Low (analysis at planning time only) |
| Fits Claude-native model? | Partially (worktrees work but add git complexity) | No (requires Docker) | Yes (file analysis is prompt-friendly) | Yes (rules in task specs) | N/A | Yes (Phase Lead groups tasks; no infrastructure needed) |

**Recommendation for Aether:** Use planning-time task grouping, not runtime isolation. The Phase Lead analyzes which tasks touch which files (from task descriptions and file paths) and groups overlapping tasks to the same worker. This is the simplest approach that fits the Claude-native model. No git worktrees, no containers, no file locks. If the Phase Lead is uncertain about file overlap, it defaults to sequential execution for those tasks. This matches the Swarm-IOSM philosophy: "conflict prevention, not conflict resolution."

### Learning Persistence Across Sessions

| Capability | CrewAI | LangGraph | AutoGen | OpenAI Agents SDK | Aether (current) | Aether (v4.4 target) |
|-----------|--------|-----------|---------|-------------------|-------------------|----------------------|
| Short-term memory | ChromaDB with RAG | Customizable (checkpoints) | Shared memory context | Conversation state | Working state in JSON files | Same (working state in JSON) |
| Long-term memory | SQLite3 for task results | Custom stores | Mem0 integration | None built-in | memory.json learnings array | Same + structured retrieval |
| Cross-session | Yes (SQLite persists) | Yes (if custom store configured) | Yes (Mem0 persists) | No (stateless by default) | Yes (memory.json persists) | Yes + global tier in ~/.aether/ |
| Cross-project | No (per-crew) | No (per-graph) | Possible via Mem0 | No | No | Yes (global learning tier with promotion) |
| Learning from outcomes | Adaptive (per docs) | No built-in mechanism | No built-in mechanism | No | Bayesian spawn tracking | Bayesian + cross-project promotion |
| Entity memory | Yes (entity relationships) | No built-in | No built-in | No | No | Not planned (unnecessary for code tasks) |
| Memory pruning | LRU + capacity limits | Manual | Manual | N/A | Not implemented | Decay-based (mirror pheromone decay) |

**Recommendation for Aether:** Implement two-tier learning. Tier 1 (project): `.aether/data/memory.json` stays as-is but gets structured tagging (category: caste-selection, conflict-avoidance, pattern, etc.). Tier 2 (global): `~/.aether/global/learnings.json` stores universal patterns. Promotion mechanism: after a learning appears in 3+ projects, it gets auto-promoted to global. Manual promotion via `/ant:feedback` for immediate universal learnings. This gives Aether something no competitor has: genuine cross-project meta-learning.

### Adaptive Complexity

| Capability | AutoGen | CrewAI | LangGraph | Industry Consensus | Aether (current) | Aether (v4.4 target) |
|-----------|---------|--------|-----------|-------------------|-------------------|----------------------|
| Complexity assessment | Implicit (user chooses single vs multi-agent) | Implicit (sequential vs hierarchical process) | Explicit (start simple, add complexity per docs) | "Agents are a last resort" -- use simplest approach that works | None (full colony always) | Colonizer assesses complexity; recommends lean vs full mode |
| Lightweight mode | Single AssistantAgent | Sequential process (no manager) | Single-node graph | Single LLM call | Not available | Lean mode: single builder-ant with learning, no Phase Lead, no waves |
| Full mode | GroupChat with multiple agents | Hierarchical process with manager | Multi-node graph with orchestrator | Multi-agent with supervisor | Full colony (Queen + castes + pheromones) | Full colony with aggressive parallelism |
| Mode switching | Manual | Manual | Manual | Manual or heuristic | N/A | Colonizer recommends; user confirms; system configures |
| Triggers for complexity | User decision | User decision | User decision | Task analysis | N/A | File count, dependency depth, technology diversity, task count |

**Recommendation for Aether:** After colonization, assess project complexity on four axes: (1) file count and directory depth, (2) number of distinct technologies/languages, (3) cross-module dependency density, (4) estimated task count. Below thresholds (e.g., <20 files, single language, <10 tasks), recommend lean mode. Above thresholds, recommend full colony. User always has final say. This addresses field notes 21, 22, and 31 directly.

### Automated Code Review / QA Agents

| Capability | Amazon Q | Qodo | Zencoder | CrewAI | Aether (current) | Aether (v4.4 target) |
|-----------|---------|------|----------|--------|-------------------|----------------------|
| Review trigger | `/review` command or PR hook | Continuous on PR | After parallel agent execution | Manual task assignment | Manual (Watcher in verification phase) | Auto-spawn after each builder completes |
| Review scope | Full PR diff | Full PR with context | Per-task output verification | Per-task | Per-phase (all tasks reviewed together) | Per-builder: each builder's output reviewed individually |
| Scoring | Pass/fail with comments | Quality score + suggestions | Pass/fail with verification | No scoring | Flat 8/10 every time | Rubric: correctness (0-3), completeness (0-3), style (0-2), risk (0-2) |
| Auto-fix | Suggestions only | Some auto-fixes | Re-run on failure | No | No | Debugger ant spawns on test failure; attempts fix before escalating |
| Integration point | PR workflow | IDE + CI | Build pipeline | Task completion | Phase boundary | Builder completion + test failure events |

**Recommendation for Aether:** Implement auto-spawned review at two trigger points. (1) After each builder completes: a Watcher reviews the builder's specific output using a 10-point rubric (correctness 0-3, completeness 0-3, style 0-2, risk 0-2). (2) On test failure: a Builder in debugger mode spawns to investigate and fix. The Watcher review should not require user approval for scores above 7/10. Below 7/10, the Watcher flags the issue and the Phase Lead decides whether to re-assign or proceed. This matches the industry pattern (Zencoder's "built-in verification after parallel execution") while adding Aether's scoring granularity.

---

## Competitor Feature Summary Matrix

| Feature Area | AutoGen | CrewAI | LangGraph | OpenAI Agents SDK | Aether v4.4 Target |
|-------------|---------|--------|-----------|-------------------|---------------------|
| **Recursive delegation** | Conversation-based, max_turns | Manager delegates, leaf agents blocked | Graph with Send API | Handoff chains | Depth-limited (max 2) with spawn-check |
| **Conflict prevention** | Not addressed (Python scope) | Role-based scoping | State reducers, checkpoints | Not addressed | Planning-time task grouping by file |
| **Learning persistence** | Mem0 integration (optional) | 4-tier built-in memory | Custom stores | None built-in | 2-tier (project + global) with promotion |
| **Adaptive complexity** | Manual agent count | Sequential vs hierarchical | Graph complexity is manual | Single-agent default | Colonizer-assessed with lean/full modes |
| **Auto code review** | Not built-in | Not built-in | Not built-in | Not built-in | Auto-spawned Watcher with 10-point rubric |
| **Claude-native** | No (Python) | No (Python) | No (Python) | No (Python) | Yes (prompt + shell) |
| **Stigmergic coordination** | No (message passing) | No (delegation) | No (graph edges) | No (handoffs) | Yes (pheromone signals with decay) |

---

## Sources

### Ecosystem Research (WebSearch -- MEDIUM confidence)

- [AutoGen: LLM-Driven Multi-Agent Framework](https://www.emergentmind.com/topics/autogen) -- AutoGen patterns and recursive delegation
- [Task Decomposition | AutoGen 0.2](https://microsoft.github.io/autogen/0.2/docs/topics/task_decomposition/) -- AutoGen task decomposition approach
- [CrewAI Hierarchical Process](https://docs.crewai.com/en/learn/hierarchical-process) -- Manager agent delegation model
- [Hierarchical AI Agents: A Guide to CrewAI Delegation](https://activewizards.com/blog/hierarchical-ai-agents-a-guide-to-crewai-delegation) -- Delegation loop prevention
- [LangGraph Multi-Agent Orchestration 2025](https://latenode.com/blog/ai-frameworks-technical-infrastructure/langgraph-multi-agent-orchestration/langgraph-multi-agent-orchestration-complete-framework-guide-architecture-analysis-2025) -- Parallel execution and conflict resolution
- [OpenAI Agents SDK](https://openai.github.io/openai-agents-python/) -- Handoff-based delegation model
- [Orchestrating Multiple Agents](https://openai.github.io/openai-agents-python/multi_agent/) -- Multi-agent patterns in OpenAI SDK

### Conflict Prevention (WebSearch -- MEDIUM confidence)

- [Multi-Agent Orchestration: Running 10+ Claude Instances in Parallel](https://dev.to/bredmond1019/multi-agent-orchestration-running-10-claude-instances-in-parallel-part-3-29da) -- File locking with dependency graph
- [Parallel Agents Are Easy. Shipping Without Chaos Isn't.](https://dev.to/rokoss21/parallel-agents-are-easy-shipping-without-chaos-isnt-1kek) -- Swarm-IOSM conflict prevention philosophy
- [Embracing the parallel coding agent lifestyle](https://simonwillison.net/2025/Oct/5/parallel-coding-agents/) -- Simon Willison on parallel agent patterns
- [Container Use: Isolated Parallel Coding Agents](https://www.infoq.com/news/2025/08/container-use/) -- Dagger container-based isolation
- [VS Code 1.107 Multi-Agent Orchestration](https://visualstudiomagazine.com/articles/2025/12/12/vs-code-1-107-november-2025-update-expands-multi-agent-orchestration-model-management.aspx) -- Git worktree isolation per agent

### Memory and Learning (WebSearch -- MEDIUM confidence)

- [CrewAI Memory Documentation](https://docs.crewai.com/en/concepts/memory) -- 4-tier memory architecture
- [Deep Dive into CrewAI Memory Systems](https://sparkco.ai/blog/deep-dive-into-crewai-memory-systems) -- ChromaDB + SQLite3 implementation
- [Memory in the Age of AI Agents](https://arxiv.org/abs/2512.13564) -- Survey of agent memory architectures (Dec 2025)
- [AI Agent Memory Comparative Analysis](https://dev.to/foxgem/ai-agent-memory-a-comparative-analysis-of-langgraph-crewai-and-autogen-31dp) -- LangGraph vs CrewAI vs AutoGen memory
- [Building Smarter AI Agents: AgentCore Long-Term Memory](https://aws.amazon.com/blogs/machine-learning/building-smarter-ai-agents-agentcore-long-term-memory-deep-dive/) -- Amazon's managed memory approach

### Anti-Patterns and Production Readiness (WebSearch -- MEDIUM confidence)

- [Anti-Patterns in Multi-Agent Gen AI Solutions](https://medium.com/@armankamran/anti-patterns-in-multi-agent-gen-ai-solutions-enterprise-pitfalls-and-best-practices-ea39118f3b70) -- Enterprise pitfalls (May 2025)
- [Cognition: Don't Build Multi-Agents](https://cognition.ai/blog/dont-build-multi-agents) -- Why multi-agent systems are fragile
- [How and When to Build Multi-Agent Systems](https://www.blog.langchain.com/how-and-when-to-build-multi-agent-systems/) -- LangChain's guidance on when agents are appropriate
- [Patterns for Building Production-Ready Multi-Agent Systems](https://dzone.com/articles/production-ready-multi-agent-systems-patterns) -- Production maturity patterns
- [Architecting Efficient Context-Aware Multi-Agent Framework](https://developers.googleblog.com/architecting-efficient-context-aware-multi-agent-framework-for-production/) -- Google's production multi-agent framework

### Automated Code Review (WebSearch -- MEDIUM confidence)

- [Best AI Coding Agents for 2026](https://www.faros.ai/blog/best-ai-coding-agents-2026) -- Landscape of coding agents with review capabilities
- [2026 Agentic Coding Trends Report](https://resources.anthropic.com/hubfs/2026%20Agentic%20Coding%20Trends%20Report.pdf?hsLang=en) -- Anthropic's trends report
- [5 Key Trends Shaping Agentic Development in 2026](https://thenewstack.io/5-key-trends-shaping-agentic-development-in-2026/) -- Multi-agent parallelism and review patterns

### Primary Sources (HIGH confidence)

- `/Users/callumcowie/repos/Aether/.planning/v5-FIELD-NOTES.md` -- 32 field notes from first real-world test
- `/Users/callumcowie/repos/Aether/.planning/PROJECT.md` -- Project context and constraints
- `/Users/callumcowie/repos/Aether/.planning/codebase/ARCHITECTURE.md` -- Current system architecture

---

*Feature research for: Aether v4.4 Colony Hardening and Real-World Readiness*
*Researched: 2026-02-04*
