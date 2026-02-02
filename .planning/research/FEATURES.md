# Feature Research: Event Polling, LLM Testing, and CLI Visual Indicators

**Domain:** Multi-Agent System Enhancement (Reactive Event Integration)
**Researched:** 2026-02-02
**Confidence:** MEDIUM

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist in any production-ready multi-agent system with event handling, testing, and user feedback. Missing these = product feels incomplete or unusable.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Event Polling Integration** | Users expect agents to react asynchronously to events, not just publish them | MEDIUM | Worker Ants must call `get_events_for_subscriber()` to check for relevant events; standard pattern in event-driven systems |
| **Event Filtering** | Users expect agents to only receive events they care about | LOW | Topic-based filtering (e.g., "phase_complete", "error", "spawn_request"); prevents event noise |
| **Event Delivery Tracking** | Users expect events to be marked as delivered to avoid reprocessing | MEDIUM | `mark_events_delivered()` prevents duplicate processing; essential for reliable event systems |
| **Manual E2E Test Guide** | Users need documented test procedures for complex multi-agent workflows | LOW | Step-by-step guide for testing init, execute, spawning, memory, voting; table stakes for production systems |
| **Visual Activity Indicators** | Users expect to see what agents are doing at a glance | LOW | Basic status indicators (🐜 for active, ⚪ for idle, ⏳ for pending); expected in all CLI tools |
| **Progress Feedback** | Users expect visible progress during long-running operations | LOW | Progress bars, step counters, completion percentages; standard in modern CLI tools |
| **Error Indicators** | Users expect clear visual cues when something goes wrong | LOW | Red indicators (✗, ❌, 🔴) for errors; green (✓, ✅, 🟢) for success; universal UX pattern |

**Why these are table stakes:**
- **Event polling** is foundational to event-driven architecture; without it, agents can't react to colony events
- **Manual test guides** are essential for validating complex autonomous systems where automated tests can't cover all scenarios
- **Visual indicators** are standard in all modern CLI tools (kubectl, docker, gh, npm) for user experience

### Differentiators (Competitive Advantage)

Features that set Aether apart from AutoGen, LangGraph, CrewAI, and other frameworks. Not required, but valuable and aligned with Aether's Core Value.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Reactive Event Polling for Prompt-Based Agents** | First event system designed for prompt-based agents (not persistent processes) | HIGH | Pull-based delivery via `get_events_for_subscriber()`; no background daemons; unique to Claude-native systems |
| **Caste-Specific Event Sensitivity** | Different Worker Ant castes react differently to same event based on sensitivity profiles | MEDIUM | Colonizer prioritizes "spawn_request" events; Watcher prioritizes "error" events; implements pheromone-like signal response |
| **Manual LLM Test Suite with Real Execution** | Tests validate actual LLM behavior (not just code coverage) | HIGH | Real `/ant:init`, `/ant:execute`, spawning, memory compression, voting; catches LLM-specific issues traditional tests miss |
| **Visual Colony Activity Dashboard** | Real-time visualization of all Worker Ant activity with emoji indicators | MEDIUM | Dashboard shows 🟢 ACTIVE / ⚪ IDLE / 🔴 ERROR for each caste; makes emergence visible |
| **Event-Driven Spawning Triggers** | Workers spawn specialists in response to events (not just capability detection) | MEDIUM | "spawn_request" event triggers Bayesian spawning; event + pheromone = true stigmergy |
| **Emoji-Based State Indicators** | Universal visual language (🐜⏳✅❌🔴🟢) for colony state | LOW | Requires no localization; instantly recognizable; improves accessibility |
| **Historical Event Replay for Testing** | Test suite can replay past events to verify agent responses | HIGH | Events logged with timestamps; enables deterministic testing of async behavior |
| **Visual Pheromone Strength Indicators** | Users see pheromone signal strength visually (e.g., `[===]` for 0.5, `[======]` for 1.0) | MEDIUM | Makes invisible signals visible; helps users understand colony guidance |

**Why these differentiate:**
- **Reactive polling for prompt-based agents** is unique: All other frameworks assume persistent agent processes (Python/Node); Aether works with Claude's prompt-based execution model
- **Caste-specific sensitivity** mirrors pheromone response profiles in real ant colonies; no other framework implements this
- **Manual LLM test suite** addresses the gap between unit tests (code) and LLM behavior (reasoning); most frameworks lack this
- **Visual dashboard** makes emergence visible; traditional systems are opaque

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems in practice for Claude-native multi-agent systems.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| **Push-Based Event Delivery (Background Daemons)** | Seems "more reactive" than polling | Breaks Claude-native model; requires persistent processes; agents are prompt-based (execute once, exit); push requires daemons monitoring events | Pull-based polling via `get_events_for_subscriber()`; workers poll when they execute |
| **Real-Time Event Streaming UI** | Users want to see events flow in real-time | Creates complexity without value; requires WebSocket/server; Claude-native systems are text-based, not GUI | Event log in `events.json`; `/ant:status` shows recent events |
| **Automated LLM Testing Only** | Automated tests seem faster and more reliable | LLMs are non-deterministic; automated tests can't validate reasoning quality; manual tests catch issues automation misses | Hybrid: Automated bash tests (integration/stress) + Manual LLM test guide for workflows |
| **Color-Based Indicators (ANSI Colors)** | Seems more professional than emojis | Not universally supported; breaks in some terminals/log files; requires color libraries; adds dependency | Emoji indicators (🐜⏳✅❌🟢🔴) work everywhere; no dependencies |
| **Complex Event Schemas (Avro/Protobuf)** | Seems more scalable than JSON | Adds build step; requires schema registry; overkill for colony-scale events; breaks Claude-native simplicity | JSON events with simple schema; human-readable; works with all tools |
| **Event Replay for "Time Travel"** | Users want to replay colony execution | Complex to implement correctly; state snapshotting required; colony state is multi-file (COLONY_STATE.json, memory.json, events.json) | Event log for audit; manual testing for validation; focus on forward execution |
| **Web-Based Dashboard** | Users want GUI visualization | Breaks Claude-native workflow; requires separate server; adds maintenance burden; splits UX between CLI and web | CLI dashboard in `/ant:status`; text-based visualization; stays in Claude |
| **Animated Progress Spinners** | Seems more engaging than static indicators | Distracts from actual work; requires async rendering; breaks in some shells; adds complexity | Static indicators (⏳✅❌) with clear meaning; functional over decorative |

## Feature Dependencies

```
[Event Polling Integration]
    ├──requires──> [Event Bus System] ✓ (v1 delivered)
    ├──requires──> [Subscriber Registration] ✓ (v1 delivered)
    └──enhances──> [Reactive Agent Behavior]

[Manual LLM Test Guide]
    ├──requires──> [Documented Workflows]
    ├──requires──> [Test Data Fixtures]
    └──validates──> [All Core Features]

[Visual Activity Indicators]
    ├──requires──> [Agent State Tracking] ✓ (v1 delivered)
    ├──requires──> [Status Command] ✓ (v1 delivered)
    └──enhances──> [User Experience]

[Caste-Specific Event Sensitivity]
    ├──requires──> [Event Polling Integration]
    ├──requires──> [Pheromone Sensitivity Profiles] ✓ (v1 delivered)
    └──enhances──> [Stigmergic Coordination]

[Event-Driven Spawning Triggers]
    ├──requires──> [Event Polling Integration]
    ├──requires──> [Autonomous Spawning] ✓ (v1 delivered)
    └──enhances──> [Colony Emergence]
```

### Dependency Notes

- **Event Polling Integration requires Event Bus System:** v1 delivered pub/sub event bus; polling is the natural next step for reactive agents
- **Event Polling enhances Reactive Agent Behavior:** Without polling, agents can't respond to async events; polling enables true event-driven behavior
- **Manual LLM Test Guide validates All Core Features:** Manual tests catch LLM reasoning issues that bash tests (code-level) miss
- **Visual Activity Indicators requires Agent State Tracking:** v1 delivered COLONY_STATE.json with active_workers; indicators surface this data visually
- **Caste-Specific Event Sensitivity requires Pheromone Sensitivity Profiles:** v1 delivered caste-specific pheromone sensitivity; extend to events
- **Event-Driven Spawning enhances Colony Emergence:** Spawning in response to events (not just capability gaps) makes colony more adaptive

## MVP Definition

### Launch With (v2.0)

Minimum viable features to validate reactive event integration concept.

- [ ] **Event Polling Integration** — Worker Ant prompts call `get_events_for_subscriber("subscriber_id" "caste")` at start of execution
- [ ] **Event Delivery Tracking** — Workers call `mark_events_delivered()` after processing to prevent reprocessing
- [ ] **Event Filtering** — Workers subscribe to relevant topics (e.g., "phase_complete", "error", "spawn_request")
- [ ] **Manual E2E Test Guide** — Documented test procedures for init, execute, spawning, memory, voting workflows
- [ ] **Basic Visual Indicators** — Add 🐜 emoji and activity state (🟢/⚪/🔴) to `/ant:status` output
- [ ] **Progress Feedback** — Show step progress in command output (e.g., "Step 1/3: Initializing...")

**Rationale:** These are the minimum features to enable reactive event handling and user visibility. Without polling, events are publish-only (agents can't react). Without manual tests, we can't validate LLM behavior. Without visual indicators, users can't see what's happening.

### Add After Validation (v2.x)

Features to add once reactive event integration is proven to work.

- [ ] **Caste-Specific Event Sensitivity** — Different castes prioritize different events based on sensitivity profiles
- [ ] **Event-Driven Spawning Triggers** — "spawn_request" events trigger Bayesian spawning (currently only capability gaps)
- [ ] **Visual Colony Activity Dashboard** — Dedicated `/ant:dashboard` command showing all agent activity with emoji indicators
- [ ] **Visual Pheromone Strength Indicators** — Show signal strength as progress bars (e.g., `[======] 1.0`)
- [ ] **Historical Event Replay for Testing** — Test suite can replay events from events.json for deterministic testing
- [ ] **Advanced Visual Indicators** — Spinners, progress bars, colored output for enhanced UX

**Rationale:** These enhance the core without changing it. Once polling works, making it caste-specific, spawning-aware, and more visual are natural improvements.

### Future Consideration (v3+)

Features to defer until reactive event integration is production-ready.

- [ ] **Real-Time Event Streaming** — WebSocket-based event stream for real-time visibility
- [ ] **Event Replay for "Time Travel"** — Full colony state snapshotting and replay
- [ ] **Web-Based Dashboard** — Separate GUI dashboard (breaks Claude-native model, defer until clear need)
- [ ] **Complex Event Schemas** — Avro/Protobuf for large-scale event processing
- [ ] **Automated LLM Behavior Testing** — Framework for programmatic LLM validation (beyond manual tests)

**Rationale:** These are power-user features. Get reactive polling working first, then add advanced capabilities.

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Event Polling Integration | HIGH | MEDIUM | P1 |
| Event Delivery Tracking | HIGH | LOW | P1 |
| Event Filtering | HIGH | LOW | P1 |
| Manual E2E Test Guide | HIGH | MEDIUM | P1 |
| Basic Visual Indicators (🐜) | HIGH | LOW | P1 |
| Progress Feedback | MEDIUM | LOW | P1 |
| Caste-Specific Event Sensitivity | MEDIUM | MEDIUM | P2 |
| Event-Driven Spawning Triggers | MEDIUM | MEDIUM | P2 |
| Visual Colony Activity Dashboard | MEDIUM | MEDIUM | P2 |
| Visual Pheromone Strength Indicators | LOW | LOW | P2 |
| Historical Event Replay for Testing | LOW | HIGH | P3 |
| Real-Time Event Streaming | LOW | HIGH | P3 |
| Event Replay for "Time Travel" | LOW | HIGH | P3 |
| Web-Based Dashboard | LOW | HIGH | P3 |
| Complex Event Schemas (Avro) | LOW | HIGH | P3 |
| Automated LLM Behavior Testing | LOW | HIGH | P3 |

**Priority key:**
- P1: Must have for v2.0 MVP
- P2: Should have, add when possible (v2.x)
- P3: Nice to have, future consideration (v3+)

## Competitor Feature Analysis

| Feature | AutoGen | LangGraph | CrewAI | Aether v2 |
|---------|---------|-----------|--------|-----------|
| **Event Polling** | ✗ (Python async/await) | ✗ (DAG execution) | ✗ (Python processes) | ✓ (Pull-based for prompts) |
| **Event Filtering** | ✓ (Python code) | ✓ (DAG routing) | ✓ (Python code) | ✓ (Topic subscriptions) |
| **Manual LLM Test Guide** | ✗ | ✗ | ✗ | Planned (✓ differentiator) |
| **Visual Indicators** | ✗ (Python logs) | ✗ (Python logs) | ✗ (Python logs) | ✓ (🐜 emoji) |
| **Caste-Specific Sensitivity** | ✗ | ✗ | ✗ | Planned (✓ differentiator) |
| **Event-Driven Spawning** | ✗ | ✗ | ✗ | Planned (✓ differentiator) |
| **Claude-Native** | ✗ (Python) | ✗ (Python) | ✗ (Python) | ✓ |

**Key Insights:**
- **All Python frameworks** use push-based event delivery (async/await, persistent processes); Aether's pull-based polling is unique to prompt-based agents
- **No existing system has manual LLM test guides** - all rely on automated unit/integration tests; Aether's manual guide addresses LLM non-determinism
- **Visual indicators are rare** - most frameworks use plain text logs; Aether's emoji-based indicators improve UX
- **Caste-specific sensitivity is unique** - extends pheromone sensitivity profiles to events; no other framework has this

## Implementation Details

### Event Polling Integration

**Current State (v1):**
- ✓ Event bus implemented (`event-bus.sh`)
- ✓ `publish_event()` for publishing events
- ✓ `subscribe_to_events()` for registering subscriptions
- ✓ Event log in `.aether/data/events.json`
- ✗ Workers don't poll for events (publish-only)

**v2 Implementation:**

1. **Add polling to Worker Ant prompts:**
   ```bash
   # At start of each Worker Ant execution
   source .aether/utils/event-bus.sh
   events=$(get_events_for_subscriber "colonizer_1" "colonizer")
   if [ -n "$events" ]; then
       # Process events
       mark_events_delivered "colonizer_1" "$event_ids"
   fi
   ```

2. **Event types for polling:**
   - `phase_complete` - Workers react when phase completes
   - `error` - Watcher reacts to errors
   - `spawn_request` - Route-setter reacts to spawn requests
   - `task_started` - Builder reacts to task starts
   - `task_completed` - Watcher reacts to task completions
   - `task_failed` - All workers react to failures

3. **Caste-specific subscriptions:**
   - Colonizer: `phase_complete`, `spawn_request`
   - Route-setter: `phase_complete`, `spawn_request`, `task_failed`
   - Builder: `task_started`, `task_completed`, `error`
   - Watcher: `task_completed`, `error`, `task_failed`
   - Scout: `spawn_request` (research requests)
   - Architect: `phase_complete` (compression triggers)

### Manual E2E Test Guide

**Test Scenarios:**

1. **Init Workflow:**
   - Execute `/ant:init "Build a REST API"`
   - Verify COLONY_STATE.json created
   - Verify pheromone signal emitted
   - Verify phase structure created

2. **Execute Workflow:**
   - Execute `/ant:execute 1`
   - Verify workers spawn autonomously
   - verify task execution
   - Verify phase completion

3. **Spawning Workflow:**
   - Trigger capability gap (e.g., "need security analysis")
   - Verify Bayesian confidence scoring
   - Verify specialist spawn
   - Verify meta-learning update

4. **Memory Workflow:**
   - Fill working memory (>60%)
   - Trigger compression
   - Verify DAST compression (2.5x ratio)
   - Verify short-term memory storage
   - Verify cross-layer search

5. **Voting Workflow:**
   - Trigger verification
   - Verify 4 watchers vote
   - Verify weighted voting
   - Verify Critical veto

6. **Event Workflow:**
   - Publish event
   - Verify worker polling
   - Verify event delivery
   - Verify delivery tracking

**Test Guide Format:**
- Step-by-step instructions
- Expected outputs (JSON snippets, console output)
- Verification checks (jq assertions)
- Common failures and troubleshooting

### Visual Activity Indicators

**Indicator Types:**

1. **Activity State:**
   - 🟢 ACTIVE - Worker is executing
   - ⚪ IDLE - Worker is waiting
   - 🔴 ERROR - Worker encountered error
   - ⏳ PENDING - Worker is queued

2. **Progress Indicators:**
   - ⏳ In progress
   - ✅ Complete
   - ❌ Failed
   - ⚠️ Warning

3. **Caste Indicators:**
   - 🐜 Colony (generic)
   - 🔍 Colonizer
   - 📋 Route-setter
   - 🔨 Builder
   - 👁️ Watcher
   - 🔬 Scout
   - 🏗️ Architect

**Implementation:**
- Add to `/ant:status` command output
- Use emoji in command headers (already partially done)
- Show activity state in worker list
- Display progress during multi-step operations

**Example Output:**
```
🐜 Worker Ant Colony:
  🔍 COLONIZER [🟢 ACTIVE] - Exploring codebase
  📋 ROUTE-SETTER [⚪ IDLE] - Waiting for phase
  🔨 BUILDER [⏳ PENDING] - Queued for task
  👁️ WATCHER [🟢 ACTIVE] - Testing module
  🔬 SCOUT [⚪ IDLE] - No research tasks
  🏗️ ARCHITECT [⚪ IDLE] - Memory at 45%
```

## Sources

### Event-Driven Multi-Agent Systems
- [AI Agent Coordination: 8 Proven Patterns [2026]](https://tacnode.io/post/ai-agent-coordination) - MEDIUM confidence, 2026-01-28
- [Event-Driven Agent Patterns: Building Reactive AI Systems that Scale](https://lijojose.medium.com/event-driven-agent-patterns-building-reactive-ai-systems-that-scale-b31da40ad852) - MEDIUM confidence, 2025
- [Event-Driven Architecture and MCP/Multi-Agentic Systems](https://portkey.ai/blog/event-driven-architecture-for-ai-agents) - MEDIUM confidence, 2025-01-13
- [Multi-Agent AI Systems in 2026: Comparing LangGraph, CrewAI, AutoGen](https://brlikhon.engineer/blog/multi-agent-ai-systems-in-2026-comparing-langgraph-crewai-autogen-and-pydantic-ai-for-production-use-cases) - MEDIUM confidence, 2026-01-18

### LLM Testing Best Practices
- [LLM Testing in 2026: Top Methods and Strategies](https://www.confident-ai.com/blog/llm-testing-in-2024-top-methods-and-strategies) - MEDIUM confidence
- [The Best LLM Evaluation Tools of 2026](https://medium.com/online-inference/the-best-llm-evaluation-tools-of-2026-40fd9b654dce) - MEDIUM confidence, 2026
- [LLMs in Software Testing: Use-Cases, Limits, & Risks in 2026](https://www.accelq.com/blog/llm-in-software-testing/) - MEDIUM confidence, 2026-01-29
- [Evaluating and Testing LLM Applications: A Comprehensive Guide](https://dasroot.net/posts/2026/01/evaluating-testing-llm-applications-comprehensive-guide/) - MEDIUM confidence, 2026-01-30
- [Testing for LLM Applications: A Practical Guide](https://langfuse.com/blog/2025-10-21-testing-llm-applications) - MEDIUM confidence, 2025-10-21

### CLI Visual Indicators & UX
- [CLI Output UX Enhancement - GitHub Issue](https://github.com/josharsh/mcp-jest/issues/19) - LOW confidence (issue discussion), 2025-11-27
- [Terminal UI Libraries - Linux Foundation Insights](https://insights.linuxfoundation.org/collection/terminal-ui-libraries) - MEDIUM confidence
- [UX Principles for Terminal Scripts](https://www.transifex.com/blog/2020/ux-terminal-scripts) - LOW confidence (older, 2020)
- [SE Radio 669: Text-Based User Interfaces](https://se-radio.net/2025/05/se-radio-669-will-mcgugan-on-text-based-user-interfaces/) - MEDIUM confidence, 2025-05-21

### Internal Research (Aether Codebase)
- `.aether/utils/event-bus.sh` - Event bus implementation (HIGH confidence, primary source)
- `.aether/utils/test-event-*.sh` - Event testing utilities (HIGH confidence, primary source)
- `.claude/commands/ant/*.md` - Command definitions with emoji usage (HIGH confidence, primary source)
- `.aether/visualization.py` - Existing visualization system (HIGH confidence, primary source)
- `.planning/PROJECT.md` - v2 milestone requirements (HIGH confidence, primary source)
- `README.md` - Project documentation (HIGH confidence, primary source)

---
*Feature research for: Aether v2 Reactive Event Integration*
*Researched: 2026-02-02*
