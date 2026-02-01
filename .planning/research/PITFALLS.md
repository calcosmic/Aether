# Domain Pitfalls

**Domain:** Claude-native multi-agent systems
**Researched:** 2026-02-01
**Confidence:** HIGH

## Critical Pitfalls

Mistakes that cause rewrites or major issues.

### Pitfall 1: Context Rot in Long-Running Sessions

**What goes wrong:**
As conversations extend beyond 50-100 messages, LLM attention degrades. Agents forget earlier instructions, contradict themselves, and lose coherence. Research confirms context rot is "real with long-running chats" in 2025.

**Why it happens:**
LLMs have limited attention spans. Claude Sonnet 4.5 maintains focus for ~30 hours internally, but production systems often exceed this. Each token added dilutes attention to earlier tokens. Multi-agent systems compound this by passing degraded context between agents.

**Consequences:**
- Agents ignore pheromone signals emitted hours ago
- Spawned specialists receive incorrect context from parent
- Contradictory behaviors emerge (e.g., Executor violates Redirect signals)
- User instructions forgotten mid-session
- Quality degrades gradually, making it hard to detect

**Prevention:**
1. **Triple-layer memory with aggressive compression**: DAST (Discriminative Abstractive Summarization Technique) compresses working memory 2.5x at phase boundaries
2. **Context window budgeting**: Never exceed 20% of 200k token limit for active context. Reddit community confirms this strategy
3. **Signal decay with explicit renewal**: Pheromone signals have half-lives (1-24 hours). Critical signals must be re-emitted before decay
4. **Context pruning**: Remove irrelevant historical context, keep only signals and current task state
5. **Context quality over quantity**: 2025 research emphasizes optimizing context quality, not maximizing token count

**Detection:**
- Agent contradicts earlier instruction (warning sign)
- Spawned specialist asks questions parent should know
- Pheromone signals ignored despite recent emission
- Response quality drops measurably over time
- Agent "loses the plot" mid-task

**Phase to address:** Phase 3 (Triple-Layer Memory implementation)

---

### Pitfall 2: Infinite Spawning Loops

**What goes wrong:**
Agents spawn specialists who spawn more specialists recursively, exhausting quota and creating infinite trees. OpenAI agents Python repo has active issues about this. Research shows "more than one-third of agents fall into infinite loops during task execution."

**Why it happens:**
Agent detects capability gap → spawns specialist → specialist detects different gap → spawns another specialist → loop. No global visibility into spawn tree depth or quota. Circuit breaker never triggers because each spawn looks legitimate locally.

**Consequences:**
- API quota exhaustion (95% wasted in failed pilots)
- Cost explosion (each spawn = new API call)
- System hangs waiting for spawn tree that never completes
- Claude Code rate limiting kicks in
- User abandons system due to cost/speed

**Prevention:**
1. **Global spawn depth limit**: Max 3 levels deep (configurable). Aether Python prototype has this at lines 61-64
2. **Per-phase spawn quota**: Max 10 specialists per phase (configurable)
3. **Spawn circuit breaker**: Auto-triggers after 3 failed spawns in 5 minutes, requires cooldown before reset
4. **Capability gap cache**: Before spawning, check if this gap was already addressed. Don't spawn same specialist twice for same task
5. **Spawn cost awareness**: Each spawn decrement quota, display remaining to user
6. **Max iterations limit**: Set `max_iterations = 5` for any agent loop (industry best practice)

**Detection:**
- Spawn depth exceeds 3 levels
- Same specialist type spawned multiple times for same task
- Phase quota exhausted before completion
- Agent spends more time spawning than working
- API cost spikes unexpectedly

**Phase to address:** Phase 1 (Autonomous Spawning with circuit breakers)

---

### Pitfall 3: JSON State Corruption from Race Conditions

**What goes wrong:**
Multiple agents read/write same JSON file simultaneously. Last write wins, losing intermediate updates. Langflow GitHub issue #8791 documents actual "data corruption" from this. Aether's meta-learning state and pheromone history are vulnerable.

**Why it happens:**
Claude Code spawns agents in parallel for speed. Each agent reads JSON, modifies in memory, writes back. No file locking or atomic updates. Python's `json.load()` → modify → `json.dump()` is not atomic. Two agents doing this simultaneously = race condition.

**Consequences:**
- Meta-learning confidence scores lost (alpha/beta counts corrupted)
- Pheromone signals overwritten (emitted signals disappear)
- Spawn history incomplete (events missing)
- Colony state inconsistent (working memory says X, short-term says Y)
- Silent corruption — no error, just wrong data

**Prevention:**
1. **File locking**: Use `fcntl.flock()` or `portalocker` for all JSON writes
2. **Atomic write pattern**: Write to temp file, then atomic rename (prevents partial writes)
3. **State versioning**: Add `_version` field to all JSON. Read version, modify, write only if version unchanged (optimistic locking)
4. **Single-writer architecture**: Only one agent (Queen?) should write to shared state. Others send messages
5. **JSONL over JSON**: Use JSONL (JSON Lines) for append-only logs like spawn history. Claude Code uses this for session storage
6. **State validation**: After load, validate JSON schema. Reject corrupted files with clear error

**Detection:**
- Confidence scores don't sum correctly (alpha + beta ≠ total)
- Pheromone signal count decreases over time
- JSON parse errors on startup
- Spawn history has gaps in timestamps
- Inconsistent state between memory layers

**Phase to address:** Phase 2 (Interactive Commands with state management)

---

### Pitfall 4: Prompt Brittleness and Complexity Explosion

**What goes wrong:**
Prompts grow to 3000+ tokens with hardcoded logic, conditional branches, and fragile instructions. Small changes break behavior. Anthropic explicitly warns against "hardcoded, complex, brittle logic in prompts."

**Why it happens:**
Each edge case gets added as prompt instruction. "If X, then do Y, unless Z." Prompt becomes control flow. No refactoring. Prompts are copy-pasted between agents with slight variations. One change requires updating 10 files.

**Consequences:**
- Prompts unmaintainable (3000+ tokens of conditional logic)
- Behavior changes unpredictably from minor wording tweaks
- New features require prompt surgery
- Prompts diverge between agents (same caste, different behavior)
- Testing impossible — can't unit test prompt logic

**Prevention:**
1. **Prompt modules, not monoliths**: Each agent role = separate prompt file. Import/include, don't copy-paste
2. **Structured outputs over prompt logic**: Use Claude's structured outputs (JSON schema) instead of "if X then output Y" instructions
3. **Prompt versioning**: Git-track prompts. Version in filenames (e.g., `executor-v3.md`)
4. **Prompt testing**: Assert outputs match expected schema. CI tests for prompt behavior
5. **Tool calling over prompting**: For logic, use tools/functions. Prompts should describe goals, not algorithms
6. **Prompt length budget**: Max 500 tokens per agent prompt. Use includes/shared sections for common text
7. **Prompt linter**: Check for anti-patterns (over-specific logic, hardcoded values, branching)

**Detection:**
- Prompt file exceeds 500 tokens
- Same instruction repeated across 3+ agent prompts
- Prompt contains "if X then Y" conditional logic
- Wording changes cause behavior regressions
- Can't explain prompt logic in 1 sentence

**Phase to address:** Phase 4 (Agent Architecture with prompt modularization)

---

### Pitfall 5: Memory Bloat from Unbounded Working Memory

**What goes wrong:**
Working memory grows unbounded. Every file read, tool call, and intermediate result stored. Eventually exceeds context window or causes OOM. Research on "memory overload" shows 10-100x degradation without forgetting mechanisms.

**Why it happens:**
No eviction policy. Working memory treated as "save everything" log. Compression only happens at phase boundaries, but phases can run for hours. Short-term memory compression is manual, not automatic.

**Consequences:**
- Context window overflow (Claude Code has GitHub issue #6186 about this)
- Token count exceeds 200k, truncation loses critical info
- Memory search slows (linear scan of 1000s of entries)
- JSON files become megabytes (slow load/save)
- Agent attention diluted by irrelevant historical context

**Prevention:**
1. **Tiered eviction policy**: When working memory hits 80% capacity, evict oldest/lowest-priority entries
2. **Automatic compression**: Compress to short-term every N messages or M minutes, not just at phase boundaries
3. **Relevance scoring**: Rank entries by relevance to current task. Evict low-relevance first
4. **Working memory cap**: Hard limit at 150k tokens (leave 50k buffer). Explicit error if exceeded
5. **Semantic summarization**: Use DAST to compress related entries into single summary
6. **Forgetting by design**: Not everything needs to be remembered. FadeMem research shows strategic forgetting improves efficiency 10-100x

**Detection:**
- Working memory token count > 150k
- JSON file > 1MB
- Search takes > 1 second
- Context truncation warnings
- Agent responses reference old, irrelevant context

**Phase to address:** Phase 3 (Triple-Layer Memory with automatic compression)

---

## Moderate Pitfalls

Mistakes that cause delays or technical debt.

### Pitfall 6: Invisible State Mutations

**What goes wrong:**
Agent modifies shared state without logging. Other agents can't reproduce or debug issues. Research identifies this as a top failure mode: "invisible state mutations."

**Why it happens:**
No audit trail. JSON files mutated in place. No "who changed what, when" tracking. State changes are side effects of agent actions, not explicit operations.

**Consequences:**
- Impossible to debug ("why is confidence score 0.75?")
- Can't roll back bad state changes
- Reproducing bugs requires replaying entire session
- No accountability for bad decisions

**Prevention:**
1. **Audit log**: Every state change logged with timestamp, agent, and reason
2. **Immutable operations**: Don't mutate in place. Create new version, link to old
3. **State diff tools**: Show what changed between operations
4. **Undo/redo**: Track state history for rollback
5. **Explicit state transitions**: State changes are first-class operations, not side effects

**Phase to address:** Phase 2 (Interactive Commands with audit logging)

---

### Pitfall 7: Coordination Token Waste

**What goes wrong:**
Agents spend tokens talking to each other instead of working. "Token budgets lost to coordination chatter." Multi-agent systems can waste 95% of tokens on internal communication.

**Why it happens:**
Every spawn includes full context. Agents re-state what they know. Verbose handoffs. No compression of shared context.

**Consequences:**
- API costs 10-20x higher than necessary
- Slower execution (more tokens = slower responses)
- Context window fills with chatter
- Effective work rate drops

**Prevention:**
1. **Minimal handoff context**: Spawn with only task + relevant context, not full conversation
2. **Shared context compression**: Maintain shared state separately, don't duplicate in each message
3. **Structured handoffs**: Use JSON schemas for handoffs, not natural language summaries
4. **Token budget monitoring**: Track coordination vs. work tokens. Alert if coordination > 30%
5. **Result caching**: Don't re-compute what another agent already did

**Phase to address:** Phase 4 (Agent Architecture with efficient handoffs)

---

### Pitfall 8: Halucination Cascades

**What goes wrong:**
One agent hallucinates fact, next agent builds on it, third agent "verifies" it. False confidence grows. Research on multi-agent systems shows "hallucination propagation" as a critical failure mode.

**Why it happens:**
Agents trust each other's output. No ground truth verification. Confidence scores treat hallucinated info as real. No cross-checking against authoritative sources.

**Consequences:**
- System confidently asserts falsehoods
- Bugs introduced based on hallucinated APIs
- Time wasted debugging non-existent issues
- User loses trust in system

**Prevention:**
1. **Source verification**: All claims must cite sources (docs, code, web search)
2. **Confidence calibration**: Low confidence should trigger verification, not blind trust
3. **Cross-agent validation**: Verifier agents independently check, don't just confirm
4. **Ground truth checks**: Validate against codebase/docs, not other agents
5. **Hallucination detection**: Flag claims without sources or with low confidence

**Phase to address:** Phase 5 (Verification with cross-checking)

---

## Minor Pitfalls

Mistakes that cause annoyance but are fixable.

### Pitfall 9: Hardcoded Phase Tasks

**What goes wrong:**
Every phase returns same predefined task list. No adaptation to actual project needs. Aether Python prototype has this at lines 918-955.

**Why it happens:**
Template-based generation. Planner Ant doesn't analyze codebase, just returns hardcoded list.

**Consequences:**
- Generic tasks that don't match project
- Wasted time on irrelevant work
- Missed project-specific needs
- Feels like "scripted demo" not intelligent system

**Prevention:**
1. **Dynamic task generation**: Planner analyzes goal + codebase → custom tasks
2. **Context-aware planning**: Tasks depend on project state, not templates
3. **User feedback loop**: Allow user to adjust tasks before execution
4. **Task prioritization**: Rank tasks by impact/dependencies

**Phase to address:** Phase 2 (Interactive Commands with dynamic planning)

---

### Pitfall 10: Pheromone Signal Saturation

**What goes wrong:**
Too many signals emitted. Agents can't distinguish important from noise. Signal decay doesn't clear fast enough.

**Why it happens:**
Every action emits signal. No filtering. Signals accumulate faster than decay.

**Consequences:**
- Agents ignore all signals (noise drowning signal)
- Redirect signals ineffective
- Focus signals lost in noise
- Signal system becomes useless

**Prevention:**
1. **Signal emission threshold**: Only emit if confidence > threshold or importance > threshold
2. **Signal coalescing**: Merge similar signals (e.g., multiple "focus database" → one stronger signal)
3. **Faster decay for low-importance**: Weak signals decay in hours, strong in days
4. **Signal cap**: Max N active signals. Drop weakest when exceeded
5. **Signal relevance scoring**: Boost signals relevant to current task

**Phase to address:** Phase 2 (Interactive Commands with signal optimization)

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| **Phase 1: Autonomous Spawning** | Infinite spawn loops | Circuit breaker + depth limit + spawn cost tracking |
| **Phase 2: Interactive Commands** | JSON state corruption | File locking + atomic writes + state versioning |
| **Phase 3: Triple-Layer Memory** | Context rot + memory bloat | Automatic compression + eviction policy + 20% context budget |
| **Phase 4: Agent Architecture** | Prompt brittleness | Prompt modules + structured outputs + 500-token budget |
| **Phase 5: Verification** | Hallucination cascades | Source verification + cross-agent validation + ground truth checks |
| **Phase 6: Research Synthesis** | Coordination token waste | Minimal handoffs + shared context + token monitoring |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Context rot | Phase 3 (Memory compression) | Test 100-message session for coherence |
| Infinite spawning | Phase 1 (Circuit breakers) | Spawn depth never exceeds 3 |
| JSON corruption | Phase 2 (File locking) | Concurrent agent writes don't lose data |
| Prompt brittleness | Phase 4 (Prompt modules) | Change 1 instruction, only 1 file affected |
| Memory bloat | Phase 3 (Eviction policy) | Working memory stays under 150k tokens |
| State mutations | Phase 2 (Audit logging) | Every state change has timestamp + agent |
| Token waste | Phase 4 (Efficient handoffs) | Coordination tokens < 30% of total |
| Hallucination cascades | Phase 5 (Verification) | All claims cite sources |
| Hardcoded tasks | Phase 2 (Dynamic planning) | Tasks adapt to project needs |
| Signal saturation | Phase 2 (Signal filtering) | < 10 active signals at any time |

## "Looks Done But Isn't" Checklist

Things that appear complete but are missing critical pieces.

- [ ] **Autonomous spawning**: Often missing circuit breaker reset — verify spawn depth limit actually prevents infinite loops
- [ ] **Memory compression**: Often missing automatic trigger — verify compression happens without manual command
- [ ] **State persistence**: Often missing atomic writes — verify concurrent agents don't corrupt JSON
- [ ] **Prompt modularity**: Often missing shared sections — verify changing instruction doesn't require editing 10 files
- [ ] **Pheromone signals**: Often missing relevance scoring — verify agents distinguish important signals from noise
- [ ] **Verification**: Often missing ground truth checks — verify agents don't just confirm each other's hallucinations
- [ ] **Session persistence**: Often missing context restoration — verify resume has same working memory state
- [ ] **Meta-learning**: Often missing validation — verify confidence scores actually improve spawn decisions

## Recovery Strategies

When pitfalls occur despite prevention, how to recover.

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| **Context rot** | HIGH | Compress working memory to short-term, reload compressed state, clear stale signals |
| **Infinite spawn loop** | LOW | Kill spawn tree, reset circuit breaker, add spawn depth check to prevent recurrence |
| **JSON corruption** | HIGH | Restore from last checkpoint, implement file locking before next run |
| **Prompt brittleness** | MEDIUM | Refactor prompt into modules, version control, A/B test changes |
| **Memory bloat** | MEDIUM | Force manual compression, implement automatic eviction policy |
| **State mutation bug** | MEDIUM | Replay audit log to identify bad state, revert to pre-bug checkpoint |
| **Token waste** | LOW | Compress handoff context, implement shared state store |
| **Hallucination cascade** | HIGH | Identify hallucination source, mark as untrusted, regenerate with verification |

## Technical Debt Patterns

Shortcuts that seem reasonable but create long-term problems.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| **Hardcoded specialist types** | Quick prototype | Can't adapt to new domains | Never — blocks autonomous spawning |
| **Manual memory compression** | Simpler implementation | Sessions rot between phases | MVP only, must auto-compress in Phase 3 |
| **No file locking** | Simpler code | Data corruption under load | Single-agent only, never in multi-agent |
| **Monolithic prompts** | Faster initial dev | Unmaintainable at 5+ agents | Never — starts causing problems immediately |
| **JSON for everything** | Simple, human-readable | Slow at scale, no querying | MVP only, migrate to SQLite/db at scale |
| **Stub verification** | Appears to work | Hallucinations propagate | Never — verification is critical |
| **No audit trail** | Less code | Impossible to debug | Never, debugging multi-agent is hard enough |
| **Signal spam** | More "activity" | System becomes unusable | Never, signals must be selective |

## Integration Gotchas

Common mistakes when connecting to external services.

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| **Claude API** | Ignoring token limits in streaming responses | Track tokens proactively, budget for full response |
| **File system** | Assuming write operations are atomic | Use write-then-rename pattern for atomicity |
| **Git integration** | Committing without diff review | Always show user what will be committed before running `git commit` |
| **Web search** | Trusting search results without verification | Treat as LOW confidence until verified with official docs |
| **Code execution** | Running commands without dry-run | Show user command, ask confirmation, then execute |
| **MCP servers** | Loading all servers globally (context overflow) | Load servers agent-scoped, unload when not needed |
| **JSON persistence** | Mutating in place, losing history | Version all state, keep audit trail |

## Performance Traps

Patterns that work at small scale but fail as usage grows.

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| **Linear search memory** | Search slows as memory grows | Add inverted index or vector search | > 1000 memory entries |
| **Synchronous JSON I/O** | UI freezes during save | Use background threads for file ops | > 1MB JSON files |
| **No connection pooling** | Each operation opens new connection | Pool connections to external services | > 10 operations/minute |
| **Redundant embeddings** | CPU spike on repeated text | Cache embeddings with LRU | > 100 embeddings/session |
| **In-memory vector store** | RAM usage grows unbounded | Use disk-based vector DB | > 10k vectors |
| **Unbounded log growth** | Disk space fills | Rotate/compress logs, keep retention policy | > 1GB logs |
| **No rate limiting** | API quota exhaustion | Track usage, throttle before limit | > 100 spawns/hour |

## Security Mistakes

Domain-specific security issues beyond general web security.

| Mistake | Risk | Prevention |
|---------|------|------------|
| **eval() on JSON keys** | Code injection if JSON tampered | Use `json.loads()` with proper parsing |
| **Unsafe file paths** | Path traversal attacks | Validate/sanitize all file paths, whitelist allowed dirs |
| **No auth on operations** | Unauthorized colony control | Add authentication for all colony-changing operations |
| **Plain text secrets** | Credentials exposed if filesystem compromised | Encrypt sensitive data at rest |
| **Arbitrary code execution** | Agent runs malicious commands | Sandbox command execution, whitelist allowed commands |
| **Unchecked imports** | Code injection via import paths | Whitelist allowed modules, validate import paths |
| **Error leakage** | Stack traces expose system internals | Sanitize error output before display |

## Sources

### Context Rot & Memory Issues
- [Medium: Context rot confirmed real in 2025](https://medium.com/@umairamin2004/why-multi-agent-systems-fail-in-production-and-how-to-fix-them-3bedbdd4975b) — MEDIUM confidence, 2025
- [Reddit: Claude Code context window strategy (20% rule)](https://www.reddit.com/r/ClaudeAI/comments/1p05r7p/my_claude_code_context_window_strategy_200k_is) — MEDIUM confidence, community practice
- [Claude Sonnet 4.5 maintains 30+ hour focus](https://sparkco.ai/blog/mastering-claudes-context-window-a-2025-deep-dive) — LOW confidence, marketing claim
- [AWS: Context window overflow breakdown](https://aws.amazon.com/blogs/security/context-window-overflow-breaking-the-barrier/) — HIGH confidence, official AWS
- [GitHub: Agent-scoped MCP servers to prevent overflow](https://github.com/anthropics/claude-code/issues/6186) — HIGH confidence, official issue
- [Anthropic: Context editing docs](https://platform.claude.com/docs/en/build-with-claude/context-editing) — HIGH confidence, official docs
- [Medium: Context quality over quantity](https://hyperdev.matsuoka.com/p/how-claude-code-got-better-by-protecting) — MEDIUM confidence, 2025 analysis

### Infinite Loops & Spawning
- [Medium: Why multi-agent systems fail](https://medium.com/@umairamin2004/why-multi-agent-systems-fail-in-production-and-how-to-fix-them-3bedbdd4975b) — MEDIUM confidence, cites infinite loops as top failure
- [arXiv: 1/3 agents fall into infinite loops](https://arxiv.org/html/2512.01939v1) — HIGH confidence, academic research
- [Science Direct: Infinite planning loops](https://www.sciencedirect.com/science/article/pii/S1566253525006712) — HIGH confidence, academic taxonomy
- [Substack: Set max_iterations=5](https://pub.towardsai.net/building-ai-agents-in-2025-your-zero-to-hero-guide-328884708efa) — MEDIUM confidence, best practice guide
- [ServiceNow: Eliminate recursive loops](https://www.servicenow.com/community/ceg-ai-coe-articles/limit-assist-consumption-by-designing-ai-agents-which-avoid/ta-p/3450013) — MEDIUM confidence, recent article
- [GitHub: OpenAI agents infinite recursion issue](https://github.com/openai/openai-agents-python/issues/668) — HIGH confidence, confirmed bug
- [The New Stack: Recursive security loops](https://thenewstack.io/is-your-ai-assistant-creating-a-recursive-security-loop/) — MEDIUM confidence, security analysis

### JSON State & Race Conditions
- [GitHub: Langflow race condition data corruption](https://github.com/langflow-ai/langflow/issues/8791) — HIGH confidence, confirmed bug
- [Medium: Invisible state mutations](https://medium.com/@sahin.samia/engineering-challenges-and-failure-modes-in-agentic-ai-systems-a-practical-guide-f9c43aa0ae3f) — HIGH confidence, 2025 guide
- [IETF Draft: HTTP profile for agentic state](https://datatracker.ietf.org/doc/draft-jurkovikj-httpapi-agentic-state/) — HIGH confidence, standards body
- [arXiv: Safeguarding multi-agent data](https://arxiv.org/html/2505.12490v1) — HIGH confidence, academic paper
- [Claude SDK: JSON serialization bug](https://github.com/anthropics/claude-agent-sdk-python/issues/510) — HIGH confidence, official issue
- [Claude SDK: Structured output errors](https://github.com/anthropics/claude-agent-sdk-typescript/issues/77) — HIGH confidence, official issue
- [Milvus Blog: Claude Code JSONL for stability](https://milvus.io/blog/why-claude-code-feels-so-stable-a-developers-deep-dive-into-its-local-storage-design.md) — MEDIUM confidence, technical analysis

### Prompt Brittleness & Complexity
- [Anthropic: Effective context engineering](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents) — HIGH confidence, official Anthropic engineering blog
- [Orq.ai: Prompt brittleness and error accumulation](https://orq.ai/blog/llm-agents) — MEDIUM confidence, 2025 analysis
- [arXiv: Multiple tools increase prompt complexity](https://arxiv.org/html/2512.08769v1) — HIGH confidence, academic research
- [Maxim.ai: Hardcoded prompt logic creates fragility](https://www.getmaxim.ai/articles/improving-prompt-engineering-for-enterprise-ai-agents/) — MEDIUM confidence, enterprise guide
- [Medium: Why your multi-agent system is failing](https://towardsdatascience.com/why-your-multi-agent-system-is-failing-escaping-the-17x-error-trap-of-the-bag-of-agents/) — MEDIUM confidence, technical analysis

### Memory & Forgetting
- [Ranjankumar.in: State management in multi-agent systems](https://ranjankumar.in/building-agents-that-remember-state-management-in-multi-agent-ai-systems) — MEDIUM confidence, 2025 retrospective
- [Medium: Engineering challenges in agentic AI](https://medium.com/@sahin.samia/engineering-challenges-and-failure-modes-in-agentic-ai-systems-a-practical-guide-f9c43aa0ae3f) — HIGH confidence, comprehensive guide
- [Galileo.ai: Agent failure modes guide](https://galileo.ai/blog/agent-failure-modes-guide) — MEDIUM confidence, 2025 guide
- [Composio: AI agent pilots fail](https://composio.dev/blog/why-ai-agent-pilots-fail-2026-integration-roadmap) — MEDIUM confidence, 2025 report

### Coordination & Token Waste
- [Towards Data Science: Why your multi-agent system is failing](https://towardsdatascience.com/why-your-multi-agent-system-is-failing-escaping-the-17x-error-trap-of-the-bag-of-agents/) — HIGH confidence, explicitly discusses coordination token waste
- [Medium: Building effective enterprise agents](https://www.bcg.com/assets/2025/building-effective-enterprise-agents.pdf) — MEDIUM confidence, BCG report
- [Orq.ai: Why do multi-agent LLM systems fail](https://orq.ai/blog/why-do-multi-agent-llm-systems-fail) — MEDIUM confidence, analysis

### Hallucination & Verification
- [Galileo.ai: 7 AI agent failure modes](https://galileo.ai/blog/agent-failure-modes-guide) — MEDIUM confidence, covers hallucination propagation
- [Medium: Why multi-agent systems fail](https://medium.com/@umairamin2004/why-multi-agent-systems-fail-in-production-and-how-to-fix-them-3bedbdd4975b) — MEDIUM confidence, discusses hallucination cascades

### Aether-Specific Issues
- [Aether concerns document](/Users/callumcowie/repos/Aether/.planning/codebase/CONCERNS.md) — HIGH confidence, internal analysis of Python prototype
- [Aether autonomous spawning research](/Users/callumcowie/repos/Aether/.ralph/AUTONOMOUS_AGENT_SPAWNING_RESEARCH.md) — HIGH confidence, internal research
- [Aether memory architecture research](/Users/callumcowie/repos/Aether/.ralph/MEMORY_ARCHITECTURE_RESEARCH.md) — HIGH confidence, internal research

---

**Pitfalls research for: Claude-native multi-agent systems**
**Researched: 2026-02-01**
