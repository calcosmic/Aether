# Domain Pitfalls

**Domain:** Multi-Agent Systems with Event Polling, LLM Testing, and CLI Visual Indicators
**Researched:** 2026-02-02
**Overall Confidence:** MEDIUM

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

### Pitfall 6: Polling Thundering Herd (V2)

**What goes wrong:**
All agents start polling simultaneously at the same interval, creating synchronized load spikes. The system appears functional during development but collapses under concurrent load when multiple agents poll events.json or external services simultaneously. Resources become saturated, response times spike, and the system becomes unresponsive.

**Why it happens:**
Developers naturally use the same polling interval across all agents for simplicity. Without randomized jitter, all agents synchronize their polling cycles. This is exacerbated when polling is triggered by shared events (like phase completion) rather than independent timers.

**Consequences:**
- Synchronized I/O spikes cause system freezes
- events.json file contention leads to read/write errors
- Multiple agents waste cycles polling when no events exist
- System appears unresponsive during polling spikes
- Resource exhaustion (CPU, disk I/O) at regular intervals

**Prevention:**
1. **Randomized jitter**: Add ±20-30% jitter to all polling intervals
2. **Exponential backoff**: When no events detected, increase interval exponentially (max 60s)
3. **Staggered initialization**: Add random startup delay (0-5s) before first poll
4. **Agent-type intervals**: Different agents poll at different rates (VerifierAnt: 10s, ExecutorAnt: 2s)
5. **Event-based triggers**: Only poll when notified of new events, not continuously
6. **Minimum interval enforcement**: Never poll faster than 1 second regardless of triggers

**Detection:**
- Monitor for synchronized request patterns in logs
- Watch for CPU/disk I/O spikes at regular intervals
- Check events.json for contention (multiple readers/writers blocking)
- Track polling rate vs. event processing rate

**Phase to address:** Phase 1 (Reactive Event Integration)

---

### Pitfall 7: Event Saturation - Signal Drowning in Noise (V2)

**What goes wrong:**
The event system generates too many low-value events, making it impossible to identify important signals. Agents spend more time processing irrelevant events than doing useful work. The events.json file grows unbounded, causing performance degradation.

**Why it happens:**
Developers log events eagerly for debugging without considering long-term impact. There's no filtering or prioritization of events by importance. Every state transition, pheromone detection, and task update gets logged indiscriminately.

**Consequences:**
- events.json grows unbounded (megabytes, then gigabytes)
- Event processing becomes bottleneck (agents spend 80% time filtering events)
- Important signals lost in noise (Redirect pheromones ignored)
- File I/O slows (reading massive files takes seconds)
- Memory exhaustion (loading all events into RAM)

**Prevention:**
1. **Event priority levels**: CRITICAL, HIGH, MEDIUM, LOW, DEBUG — only log CRITICAL/HIGH by default
2. **Adaptive filtering**: Only log DEBUG events when system in "debug mode"
3. **Contextual relevance scoring**: Use research-based adaptive filtering for data saturation
4. **Temporal decay**: Auto-prune low-priority events after 24 hours, high-priority after 7 days
5. **Event aggregation**: Combine similar events within time windows (e.g., "5 files read" not 5 separate events)
6. **Size-based rotation**: Rotate events.json when it exceeds 10MB

**Detection:**
- Monitor events.json size growth rate (alert if > 1MB/hour)
- Track ratio of events processed vs. meaningful actions taken (should be > 1:10)
- Measure agent latency attributable to event processing (should be < 20% of total)
- Alert when event processing exceeds 50% of agent CPU time

**Phase to address:** Phase 1 (Reactive Event Integration)

---

### Pitfall 8: LLM Test Flakiness from Non-Determinism (V2)

**What goes wrong:**
Tests generated by LLMs pass sometimes and fail other times without code changes. This erodes trust in the testing system. Developers start ignoring test failures or disabling tests, defeating the purpose of verification.

**Why it happens:**
LLMs are inherently non-deterministic. The same prompt can produce different outputs across runs. Test assertions may be too strict (exact string matching) or too permissive (catching nothing). Insufficient context provided to LLMs causes flakiness.

**Consequences:**
- CI builds fail randomly (same code, different result)
- Developers lose trust in test suite (ignore failures)
- Time wasted debugging "flaky" tests instead of real issues
- Test suite disabled entirely (defeating purpose of verification)
- False confidence (tests pass but bugs exist)

**Prevention:**
1. **Golden datasets**: Use consistent test inputs with known expected outputs
2. **Fuzzy assertions**: Semantic similarity, not exact string matching (e.g., "contains 'user authenticated'")
3. **Temperature control**: Set temperature 0.0-0.2 for deterministic test generation
4. **Comprehensive context**: Provide code under test, requirements, examples to LLM
5. **Consensus testing**: Run LLM tests 3 times, require 2/3 to pass (majority vote)
6. **Separate test suites**: Isolate LLM tests from traditional unit tests (different CI jobs)
7. **Flakiness tracking**: Track test pass rate over time, flag tests with < 80% consistency

**Detection:**
- Track test flakiness rate (tests that pass < 80% of time without code changes)
- Monitor correlation between test failures and LLM model version changes
- Flag tests with high variance in execution time (> 2x deviation)
- Alert when same test fails 2x in 10 runs with no code changes

**Phase to address:** Phase 2 (LLM Testing Integration)

---

### Pitfall 9: Visual Clutter - Emoji Overload (V2)

**What goes wrong:**
The CLI output becomes unreadable due to excessive emoji and visual indicators. Users can't quickly identify important information amidst decorative symbols. The terminal output looks "busy" but communicates poorly. In terminals without emoji support, output appears as garbled characters.

**Why it happens:**
Developers add emojis to every status message for visual appeal without considering information hierarchy. There's no distinction between decorative and functional indicators. The system doesn't check for terminal Unicode support before using emojis.

**Consequences:**
- Users can't find errors in verbose emoji-filled output
- Terminal renders as garbled text in CI/CD logs (no emoji support)
- Accessibility issues (screen readers announce emoji names, not semantic meaning)
- Output scrolls too fast to read (too much visual noise)
- Professional appearance lost (looks like toy, not tool)

**Prevention:**
1. **Visual hierarchy**: Use emojis only for state changes and critical alerts (5-7 core indicators max)
2. **Adaptive emoji support**: Detect terminal capabilities, fall back to text symbols (+, x, >)
3. **Core indicators only**: ✓ success, ✗ error, ⚠ warning, ⟳ in-progress, ℹ info
4. **Color coding primary**: Use ANSI colors as primary indicator, emojis as secondary
5. **--plain flag**: Provide option to disable all visual flourishes
6. **Clarity over decoration**: Follow "Clarity and Control" principles for terminal UX

**Detection:**
- User complaints about unreadable output
- Terminal rendering issues in CI/CD logs (garbled characters)
- Time to scan output for errors increases (> 5 seconds to find error)
- A/B test shows users find plain-text output faster to parse

**Phase to address:** Phase 2 (CLI Visual Indicators)

---

### Pitfall 10: Test Generation Without Coverage Goals (V2)

**What goes wrong:**
The LLM generates many tests that cover the same happy path while missing edge cases and error conditions. Coverage reports look good (80%+) but critical failure modes remain untested. The system passes all tests but fails in production.

**Why it happens:**
LLMs tend to generate obvious, straightforward tests. Without explicit coverage goals, they focus on normal operation rather than boundary conditions. There's no feedback loop to identify what's NOT being tested.

**Consequences:**
- False confidence (high coverage, low protection)
- Production bugs in untested edge cases (null inputs, concurrent access)
- Test suite passes but system crashes on invalid input
- Time wasted writing redundant tests (same path tested 10x)
- Missing tests for error handling (exceptions, timeouts)

**Prevention:**
1. **Coverage targets before generation**: Require branch coverage > 70%, line coverage > 80%
2. **Explicit edge case prompting**: "Generate tests for: null inputs, empty strings, negative numbers, concurrent access"
3. **Coverage gap feedback**: Analyze coverage, feed gaps back to LLM for additional tests
4. **Property-based testing**: Use Hypothesis for data validation logic (test invariants, not examples)
5. **Uncovered code list**: Maintain list of untested functions, require explicit sign-off
6. **Negative test requirement**: Require at least 1 error test per function (what happens on failure?)

**Detection:**
- Coverage reports show high line coverage (80%+) but low branch coverage (< 50%)
- Production bugs occur in code paths with no tests
- Mutation testing survival rate > 20% (tests too weak to kill mutants)
- Test suite passes but code crashes on edge case (e.g., empty input)

**Phase to address:** Phase 2 (LLM Testing Integration)

---

### Pitfall 11: Polling Without Backpressure (V2)

**What goes wrong:**
When events queue up faster than agents can process them, the system falls into a death spiral. Polling continues to add more events to the queue while processing falls further behind. Memory usage grows unbounded until the system crashes.

**Why it happens:**
The polling loop doesn't check if the previous event was processed before fetching new ones. There's no mechanism to detect "falling behind" and slow down polling. The get_events_for_subscriber() function may return all pending events without batching.

**Consequences:**
- Memory exhaustion (event queue grows without bound)
- Event processing latency increases (events processed hours after creation)
- System becomes unresponsive (all resources dedicated to event processing)
- Crash due to OOM (queue exceeds available RAM)
- Critical events lost (queue overflow drops oldest events)

**Prevention:**
1. **Backpressure monitoring**: Track processing queue depth, increase polling interval when queue > threshold
2. **Event batching**: Return max N events per poll (e.g., 10), process fully before next poll
3. **Circuit breaker**: Stop polling if queue depth > critical_threshold (e.g., 1000 events)
4. **Processing lag tracking**: Monitor time from event creation to processing, alert if > 60 seconds
5. **Priority queues**: Process CRITICAL events first, drop LOW events if overloaded
6. **Queue depth limits**: Hard limit on queue size (e.g., 1000 events), drop oldest when exceeded

**Detection:**
- Memory usage grows steadily over time (linear growth)
- Event processing latency increases (events take longer to process)
- Queue depth metrics show monotonic growth (never decreases)
- System becomes sluggish (all CPU spent polling/processing)

**Phase to address:** Phase 1 (Reactive Event Integration)

---

### Pitfall 12: Event Loss During Async Operations (V2)

**What goes wrong:**
Events published during async operations are lost because the publisher doesn't wait for confirmation. The events.json file is written but the write fails silently. Critical state transitions are never recorded, causing agents to miss important signals.

**Why it happens:**
The publish() function returns immediately without waiting for fsync(). Error handling in async code is incomplete (unhandled promise rejections). File locking is not used, causing race conditions between writers.

**Consequences:**
- Agents miss expected events (no response to pheromones)
- Event logs have gaps in sequence numbers (missing events)
- Critical signals lost (Redirect pheromone not recorded, agent continues banned approach)
- State inconsistency (some agents see event, others don't)
- Silent data loss (no error, just missing events)

**Prevention:**
1. **Write-ahead logging (WAL)**: Write to log first, then to events.json, can recover from log
2. **Atomic file operations**: Write to temp file, then atomic rename (prevents partial writes)
3. **Explicit error handling**: All async operations have .catch() handlers, log errors
4. **Event replay**: On startup, check for unprocessed events, re-send to subscribers
5. **File locking**: Use flock() for events.json writes, prevent concurrent writes
6. **Unhandled rejection tracking**: Monitor for unhandled promise rejections, alert immediately

**Detection:**
- Agents miss expected events (no response to pheromones)
- Event logs have gaps in sequence numbers
- Error logs show file write failures (ENOSPC, EACCES)
- State inconsistency between agents (some see event, others don't)

**Phase to address:** Phase 1 (Reactive Event Integration)

---

### Pitfall 13: Test Brittleness from Exact Assertions (V2)

**What goes wrong:**
LLM-generated tests use exact string matching for assertions. Any minor change in output formatting (added spaces, different wording) causes test failures even when functionality is correct. Developers waste time fixing tests instead of features.

**Why it happens:**
LLMs generate tests based on examples that use exact matching. There's no guidance on appropriate assertion strategies. The test framework defaults to strict equality without providing semantic comparison tools.

**Consequences:**
- High test failure rate after refactoring (no functionality changes)
- Tests fail on whitespace-only changes (e.g., code formatting)
- Developer time spent on test maintenance > feature development
- Tests become "brittle" (break easily, low signal-to-noise ratio)
- Team loses trust in tests (ignore failures, disable suite)

**Prevention:**
1. **Semantic assertions**: "Contains substring", "matches pattern", "JSON structure match" instead of exact equality
2. **Normalization**: Strip whitespace, lowercase before text comparison
3. **Custom matchers**: Provide matchers for common patterns (JSON has field X, datetime within Y, ID format Z)
4. **Assertion strategy guidelines**: Document and provide examples of good assertions to LLM
5. **Snapshot testing**: Use snapshot testing for complex outputs, intelligent diffing
6. **Assertion libraries**: Provide assertion helpers (assertContains, assertMatchesRegex, assertJsonStructure)

**Detection:**
- High test failure rate after refactoring (no functionality changes)
- Tests failing on whitespace-only changes
- Developer time spent on test maintenance > feature development
- Tests use exact equality (assertEqual, assertStrictEqual) extensively

**Phase to address:** Phase 2 (LLM Testing Integration)

---

## Moderate Pitfalls

Mistakes that cause delays or technical debt.

### Pitfall 14: Invisible State Mutations

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

### Pitfall 15: Coordination Token Waste

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

### Pitfall 16: Hallucination Cascades

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

### Pitfall 17: Hardcoded Phase Tasks

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

### Pitfall 18: Pheromone Signal Saturation

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
| **Phase 1: Reactive Event Integration** | Polling thundering herd | Randomized jitter + exponential backoff + staggered init |
| **Phase 1: Reactive Event Integration** | Event saturation | Priority levels + adaptive filtering + temporal decay |
| **Phase 1: Reactive Event Integration** | No backpressure | Queue depth monitoring + circuit breaker + priority queues |
| **Phase 1: Reactive Event Integration** | Event loss during async | Write-ahead logging + atomic writes + file locking |
| **Phase 2: Interactive Commands** | JSON state corruption | File locking + atomic writes + state versioning |
| **Phase 2: LLM Testing Integration** | Test flakiness from non-determinism | Golden datasets + fuzzy assertions + consensus testing |
| **Phase 2: LLM Testing Integration** | Missing coverage goals | Coverage targets before generation + gap feedback + negative tests |
| **Phase 2: LLM Testing Integration** | Brittleness from exact assertions | Semantic assertions + normalization + custom matchers |
| **Phase 2: CLI Visual Indicators** | Visual clutter from emoji overload | Visual hierarchy + adaptive emoji support + --plain flag |
| **Phase 3: Triple-Layer Memory** | Context rot + memory bloat | Automatic compression + eviction policy + 20% context budget |
| **Phase 4: Agent Architecture** | Prompt brittleness | Prompt modules + structured outputs + 500-token budget |
| **Phase 5: Verification** | Hallucination cascades | Source verification + cross-agent validation + ground truth checks |
| **Phase 6: Research Synthesis** | Coordination token waste | Minimal handoffs + shared context + token monitoring |

---

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
| **Polling thundering herd** | **Phase 1 (Event polling)** | **Request patterns have jitter, no synchronized spikes** |
| **Event saturation** | **Phase 1 (Event filtering)** | **events.json < 10MB, processing latency < 1s** |
| **LLM test flakiness** | **Phase 2 (LLM testing)** | **Test pass rate > 80% across 10 runs** |
| **Visual clutter** | **Phase 2 (CLI indicators)** | **Plain mode works, emoji < 10% of output** |
| **Coverage gaps** | **Phase 2 (LLM testing)** | **Branch coverage > 70%, mutation score < 20%** |
| **No backpressure** | **Phase 1 (Event polling)** | **Queue depth never exceeds threshold** |
| **Event loss** | **Phase 1 (Event persistence)** | **No gaps in event sequence numbers** |
| **Brittle assertions** | **Phase 2 (LLM testing)** | **Refactoring doesn't break tests** |

---

## "Looks Done But Isn't" Checklist

Things that appear complete but are missing critical pieces.

### v1 Pitfalls (Already Addressed)
- [ ] **Autonomous spawning**: Often missing circuit breaker reset — verify spawn depth limit actually prevents infinite loops
- [ ] **Memory compression**: Often missing automatic trigger — verify compression happens without manual command
- [ ] **State persistence**: Often missing atomic writes — verify concurrent agents don't corrupt JSON
- [ ] **Prompt modularity**: Often missing shared sections — verify changing instruction doesn't require editing 10 files
- [ ] **Pheromone signals**: Often missing relevance scoring — verify agents distinguish important signals from noise
- [ ] **Verification**: Often missing ground truth checks — verify agents don't just confirm each other's hallucinations
- [ ] **Session persistence**: Often missing context restoration — verify resume has same working memory state
- [ ] **Meta-learning**: Often missing validation — verify confidence scores actually improve spawn decisions

### v2 Pitfalls (New)
- [ ] **Event polling**: Often missing backpressure — verify queue depth monitoring and adaptive intervals
- [ ] **Event filtering**: Often missing priority levels — verify events are tagged and filtered by importance
- [ ] **LLM tests**: Often missing edge cases — verify error paths and boundary conditions are tested
- [ ] **Test assertions**: Often missing semantic matching — verify tests use fuzzy assertions not exact strings
- [ ] **Visual indicators**: Often missing terminal capability detection — verify fallback for non-Unicode terminals
- [ ] **Error handling**: Often missing from async operations — verify all promises have .catch() handlers
- [ ] **Event persistence**: Often missing atomic writes — verify temp file + rename pattern
- [ ] **Agent coordination**: Often missing deadlock prevention — verify timeout on all agent communication
- [ ] **Resource cleanup**: Often missing on agent termination — verify subagents are cleaned up, files closed
- [ ] **Test isolation**: Often missing state reset between tests — verify tests don't depend on execution order

---

## Recovery Strategies

When pitfalls occur despite prevention, how to recover.

### v1 Pitfalls
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

### v2 Pitfalls
| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| **Polling thundering herd** | HIGH | 1. Identify all polling locations. 2. Add random jitter (20-30%). 3. Implement staggered restart. 4. Add monitoring for synchronized patterns. |
| **Event saturation** | MEDIUM | 1. Stop non-critical event logging. 2. Partition events by priority. 3. Prune old events. 4. Implement aggregation for similar events. |
| **Flaky LLM tests** | HIGH | 1. Isolate flaky tests in separate suite. 2. Add retry logic (3 runs). 3. Implement golden datasets. 4. Switch to semantic assertions. |
| **Visual clutter** | LOW | 1. Audit emoji usage. 2. Remove decorative emojis. 3. Implement adaptive output. 4. Add --plain flag. |
| **Missing coverage** | MEDIUM | 1. Run coverage analysis. 2. Identify uncovered paths. 3. Explicitly prompt LLM for edge cases. 4. Add property-based tests. |
| **Queue overflow** | HIGH | 1. Stop polling immediately. 2. Implement backpressure. 3. Batch processing. 4. Add circuit breaker. |
| **Event loss** | HIGH | 1. Identify missing events. 2. Implement write-ahead logging. 3. Add atomic file operations. 4. Replay from backup. |
| **Brittle assertions** | MEDIUM | 1. Replace exact assertions with semantic matchers. 2. Add normalization. 3. Use snapshot testing for complex outputs. |

---

## Technical Debt Patterns

Shortcuts that seem reasonable but create long-term problems.

### v1 Technical Debt
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

### v2 Technical Debt
| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| **Hardcoded polling intervals** | Quick implementation, no configuration | Cannot tune per deployment, all agents synchronized | Never - use config with jitter from day 1 |
| **Logging all events to single file** | Simple pub/sub implementation | File contention, unbounded growth, no pruning | MVP only, must add partitioning before Phase 2 |
| **Using exact string assertions in tests** | Easy to generate, obvious failures | Brittle tests, high maintenance cost | Prototype only, must replace before integration |
| **Emoji decorations without detection** | Looks modern, adds visual interest | Garbled output in some terminals, accessibility issues | Never - implement adaptive output from start |
| **Synchronous test generation (block on LLM)** | Simple control flow | Slow feedback loop, poor UX | Local testing only, must be async for production |
| **Ignoring LLM non-determinism** | Tests pass initially | Flaky tests in CI, lost trust in testing | Never - address from Phase 2 start |

---

## Integration Gotchas

Common mistakes when connecting to external services.

### v1 Integration Issues
| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| **Claude API** | Ignoring token limits in streaming responses | Track tokens proactively, budget for full response |
| **File system** | Assuming write operations are atomic | Use write-then-rename pattern for atomicity |
| **Git integration** | Committing without diff review | Always show user what will be committed before running `git commit` |
| **Web search** | Trusting search results without verification | Treat as LOW confidence until verified with official docs |
| **Code execution** | Running commands without dry-run | Show user command, ask confirmation, then execute |
| **MCP servers** | Loading all servers globally (context overflow) | Load servers agent-scoped, unload when not needed |
| **JSON persistence** | Mutating in place, losing history | Version all state, keep audit trail |

### v2 Integration Issues
| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| **LLM API for test generation** | No retry logic, fail-fast on rate limits | Exponential backoff, queue requests, respect rate limits |
| **File system for events.json** | No file locking, async writes without fsync | Use file locks, write-ahead logging, atomic renames |
| **Terminal output** | Assume Unicode support, no width calculation | Detect capability, use unicode-width library, provide plain fallback |
| **Git operations for state recovery** | Assume clean working directory | Check status, stash changes, handle merge conflicts |
| **Subprocess spawning (new agents)** | No process cleanup, orphan processes | Track PIDs, implement graceful shutdown, use process groups |

---

## Performance Traps

Patterns that work at small scale but fail as usage grows.

### v1 Performance Traps
| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| **Linear search memory** | Search slows as memory grows | Add inverted index or vector search | > 1000 memory entries |
| **Synchronous JSON I/O** | UI freezes during save | Use background threads for file ops | > 1MB JSON files |
| **No connection pooling** | Each operation opens new connection | Pool connections to external services | > 10 operations/minute |
| **Redundant embeddings** | CPU spike on repeated text | Cache embeddings with LRU | > 100 embeddings/session |
| **In-memory vector store** | RAM usage grows unbounded | Use disk-based vector DB | > 10k vectors |
| **Unbounded log growth** | Disk space fills | Rotate/compress logs, keep retention policy | > 1GB logs |
| **No rate limiting** | API quota exhaustion | Track usage, throttle before limit | > 100 spawns/hour |

### v2 Performance Traps
| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| **Monolithic events.json file** | Read/write contention, slow polling | Partition by agent type, implement event rotation | > 5 agents, > 100 events/hour |
| **Unbounded event retention** | File grows until disk full | Temporal decay (auto-prune events > 24h), size limits | > 1000 events in file |
| **Synchronous LLM calls** | Agents block waiting for tests | Async test generation with callbacks | > 10 test generations/hour |
| **No event batching** | High overhead per event | Batch reads/writes, coalesce similar events | > 50 events/minute |
| **Linear event search** | Slow get_events_for_subscriber | Index by subscriber, maintain read caches | > 500 events in file |

---

## Security Mistakes

Domain-specific security issues beyond general web security.

### v1 Security Issues
| Mistake | Risk | Prevention |
|---------|------|------------|
| **eval() on JSON keys** | Code injection if JSON tampered | Use `json.loads()` with proper parsing |
| **Unsafe file paths** | Path traversal attacks | Validate/sanitize all file paths, whitelist allowed dirs |
| **No auth on operations** | Unauthorized colony control | Add authentication for all colony-changing operations |
| **Plain text secrets** | Credentials exposed if filesystem compromised | Encrypt sensitive data at rest |
| **Arbitrary code execution** | Agent runs malicious commands | Sandbox command execution, whitelist allowed commands |
| **Unchecked imports** | Code injection via import paths | Whitelist allowed modules, validate import paths |
| **Error leakage** | Stack traces expose system internals | Sanitize error output before display |

### v2 Security Issues
| Mistake | Risk | Prevention |
|---------|------|------------|
| **LLM prompt injection in test generation** | LLM generates malicious tests or code | Sanitize user input, use prompt templates, validate outputs |
| **Events.json path traversal** | Write files outside intended directory | Validate paths, use chroot, absolute paths only |
| **Terminal escape sequences in output** | Code execution via ANSI escape codes | Strip control characters, use safe output libraries |
| **Subprocess injection in agent spawning** | Arbitrary command execution | Whitelist allowed commands, use argument arrays (not strings) |
| **Test execution with elevated privileges** | Tests modify system files | Run tests in isolated environment, drop privileges |

---

## UX Pitfalls

Common user experience mistakes in this domain.

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| **Excessive emoji use (v2)** | Unreadable in some terminals, visual noise | Use sparingly (5-7 core indicators), adaptive output |
| **No progress indication for long tasks** | User thinks system hung, force-quits | Show spinner, update progress bar, estimate completion |
| **Output scrolls too fast to read** | Missed errors, lost information | Pagination, log levels, summary after completion |
| **No distinction between ant castes** | Hard to track who's doing what | Color-code by caste, prefix with caste name |
| **Silent failures** | User doesn't know something failed | Always show error message, even if "recoverable" |
| **Inconsistent status terminology** | Confusion about what "in progress" means | Define controlled vocabulary, use consistently |

---

## Sources

### Event Polling & Reactive Systems
- [Polling Is Not the Problem—Bad Polling Is](https://beingcraftsman.com/2025/12/31/polling-is-not-the-problem-bad-polling-is/) — HIGH confidence, best practices for 2025
- [Stop Polling. Start Listening: Event-Driven Architecture](https://www.hkinfosoft.com/stop-polling.start-listening/the-power-of-event-driven-architecture/) — MEDIUM confidence, pitfalls of polling
- [Event-Driven Architecture: Watch Out For These Pitfalls](https://www.forbes.com/councils/forbestechcouncil/2025/11/26/event-driven-architecture-watch-out-for-these-pitfalls-and-drawbacks/) — MEDIUM confidence, distributed system challenges
- [Managing Data Saturation in AI Systems](https://www.researchgate.net/publication/394929583_Managing_Data_Saturation_in_AI_Systems_A_Cross_Domain_Framework_Integrating_Human_Insights_and_Algorithmic_Verification) — MEDIUM confidence, saturation filtering strategies
- [Design, Implementation and Evaluation of a Real-Time Filtering System](https://arxiv.org/html/2508.18787v1) — MEDIUM confidence, dynamic signal filtering

### LLM Testing & Non-Determinism
- [On the Flakiness of LLM-Generated Tests](https://arxiv.org/html/2601.08998v1) — HIGH confidence, research paper on test flakiness
- [10 LLM Testing Strategies To Catch AI Failures](https://galileo.ai/blog/llm-testing-strategies) — MEDIUM confidence, practical testing approaches
- [LLM-As-Judge: 7 Best Practices & Evaluation Templates](https://www.montecarlodata.com/blog-llm-as-judge/) — MEDIUM confidence, avoiding flaky evaluations
- [Defeating Nondeterminism in LLM Inference](https://medium.com/lunas-orbit/mira-muratis-new-ai-lab-wants-to-fix-a-hidden-problem-in-llms-6ab72dffd6fe) — LOW confidence, needs verification
- [Understanding and Improving Flaky Test Classification](https://www.cs.cornell.edu/~saikatd/papers/flakylens-oopsla25.pdf) — HIGH confidence, Cornell research 2025
- [AI-Powered Testing Solutions for Resolving Flaky Tests](https://www.testmu.ai/blog/ai-powered-testing-solutions-for-flaky-tests/) — MEDIUM confidence, AI tools for flaky tests

### CLI Visual Indicators
- [CLI UX best practices: 3 patterns for improving progress displays](https://evilmartians.com/chronicles/cli-ux-best-practices-3-patterns-for-improving-progress-displays) — MEDIUM confidence, progress display patterns
- [Simple UX Principles for Creating Killer Terminal Scripts](https://www.transifex.com/blog/2020/ux-terminal-scripts) — MEDIUM confidence, clarity and control principles
- [State of Terminal Emulators in 2025](https://news.ycombinator.com/item?id=45799478) — LOW confidence, community discussion on emoji usage
- [Add emoji/visual indicators to CLI output for better UX](https://github.com/josharsh/mcp-jest/issues/19) — LOW confidence, GitHub issue, single source
- [Building CLI Apps in Rust — What You Should Consider](https://betterprogramming.pub/building-cli-apps-in-rust-what-you-should-consider-99cdcc67710c) — MEDIUM confidence, adaptive emoji usage

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
- `/Users/callumcowie/repos/Aether/.aether/worker_ants.py` — Current event system and agent structure
- `/Users/callumcowie/repos/Aether/.aether/error_prevention.py` — Error tracking and pattern detection
- `/Users/callumcowie/repos/Aether/.aether/queen_ant_system.py` — System integration and commands

---

**Pitfalls research for: Multi-Agent Reactive Event Integration with LLM Testing**
**Researched: 2026-02-02**
**Confidence: MEDIUM (some sources need verification, especially LLM non-determinism solutions)**
