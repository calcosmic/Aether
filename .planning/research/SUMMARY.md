# Aether v2 Research Summary

**Project:** Aether v2 — Reactive Event Integration, LLM Testing, and CLI Visual Indicators
**Domain:** Claude-native multi-agent system with event polling, testing infrastructure, and UX enhancements
**Researched:** 2026-02-02
**Confidence:** HIGH

## Executive Summary

Aether v2 enhances the existing Queen Ant Colony system with three critical integrations that transform it from a prompt-based autonomous agent framework into a reactive, testable, and user-friendly multi-agent system. Research confirms that **pull-based event polling** is the correct pattern for prompt-based Worker Ants (which execute and exit, not persistent processes), **manual E2E test guides** are the right approach for validating LLM behavior where automated tests cannot assess reasoning quality, and **Unicode emoji indicators** provide semantic, color-independent visual feedback for colony activity.

The recommended approach builds on Aether v1's proven foundation (19 commands, 10 Worker Ants, 879-line event bus) without introducing new dependencies. Event polling integrates via existing `get_events_for_subscriber()` function, LLM testing uses human-readable markdown guides executed by Claude Code, and visual indicators leverage terminal emoji support. The stack remains deliberately minimal: **Bash + jq** for event infrastructure, **Markdown** for test documentation, **Unicode emojis** for visual feedback.

Key risks are well-documented with clear mitigation strategies. The top pitfalls for v2 are **polling thundering herd** (prevented with randomized jitter and exponential backoff), **event saturation** (prevented with priority levels and adaptive filtering), and **LLM test flakiness** (prevented with golden datasets and fuzzy assertions). All v1 risks remain relevant (context rot, infinite spawning loops, JSON corruption) but have established prevention patterns in the codebase.

## Key Findings

### Recommended Stack

**Core technologies:**
- **Bash + jq** (Bash 4.0+, jq 1.6+) — Event polling infrastructure; proven pattern in Aether's event-bus.sh (879 lines); file locking via fcntl prevents race conditions
- **Markdown test guides** (GitHub Flavored Markdown) — E2E LLM test documentation; human-readable test cases Claude can execute directly; no test runner dependencies
- **Unicode emojis** (Standard Unicode 15.0+) — CLI visual indicators; terminal-compatible, semantic meaning independent of color; accessibility-friendly
- **TAP format** (TAP version 13) — Test output standard; already used in existing tests; industry standard for test reporting
- **JSON state files** (JSON RFC 8259) — Test execution tracking; Claude reads/writes natively; git-diffable for test history validation

**Supporting utilities (existing):**
- `atomic-write.sh` — Corruption-safe test state updates
- `file-lock.sh` — Concurrent test execution safety
- `event-bus.sh` — Event polling infrastructure (pull-based delivery already implemented)

### Expected Features

**Must have (table stakes):**
- **Event Polling Integration** — Worker Ants call `get_events_for_subscriber()` at execution boundaries; users expect agents to react asynchronously to events
- **Event Delivery Tracking** — Workers call `mark_events_delivered()` after processing; prevents duplicate event processing
- **Event Filtering** — Topic-based subscriptions prevent event noise; users expect agents to only receive relevant events
- **Manual E2E Test Guide** — Documented test procedures for init, execute, spawning, memory, voting workflows; table stakes for production systems
- **Visual Activity Indicators** — Basic status indicators (🐜 for active, ⚪ for idle, ⏳ for pending); expected in all modern CLI tools
- **Progress Feedback** — Progress bars, step counters, completion percentages; standard in modern CLI tools

**Should have (competitive):**
- **Reactive Event Polling for Prompt-Based Agents** — First event system designed for prompt-based agents (not persistent processes); unique to Claude-native systems
- **Caste-Specific Event Sensitivity** — Different Worker Ant castes react differently to same event based on sensitivity profiles; implements pheromone-like signal response
- **Manual LLM Test Suite with Real Execution** — Tests validate actual LLM behavior (not just code coverage); catches LLM-specific issues traditional tests miss
- **Visual Colony Activity Dashboard** — Real-time visualization of all Worker Ant activity with emoji indicators; makes emergence visible

**Defer (v2+):**
- **Real-Time Event Streaming** — WebSocket-based event stream; requires persistent processes (breaks Claude-native model)
- **Event Replay for "Time Travel"** — Full colony state snapshotting and replay; complex to implement correctly
- **Web-Based Dashboard** — Separate GUI dashboard; breaks Claude-native workflow
- **Automated LLM Behavior Testing** — Framework for programmatic LLM validation; beyond v2 scope

### Architecture Approach

Aether v2 adds three integration layers atop the existing v1 architecture:

1. **Event Polling Layer** — Worker Ant prompts call `get_events_for_subscriber()` at execution boundaries (after file writes, command completion, phase transitions). Integration points: Colonizer, Builder, Watcher, Scout.

2. **Visual Indicators Layer** — CLI output formatting with emoji, progress bars, and structured sections. Integration points: All command prompts (init, status, execute, etc.).

3. **E2E Testing Layer** — Manual test guide covering init → execute → spawn → memory → voting workflows. Output: `docs/E2E_TEST_GUIDE.md`.

**Major components:**
- **Event Polling Layer** — Worker Ants poll events at execution boundaries; communicates with Event bus (get_events_for_subscriber) and State Layer (publish events)
- **Visual Indicators Layer** — Format CLI output with emoji, progress bars, sections; communicates with all command prompts via template patterns
- **E2E Testing Layer** — Manual test guide covering core workflows; validates all layers end-to-end

### Critical Pitfalls

**Top v2 pitfalls:**

1. **Polling Thundering Herd** — All agents poll simultaneously causing synchronized load spikes. **Prevention:** Add ±20-30% randomized jitter to polling intervals, implement exponential backoff when no events detected, stagger agent initialization with 0-5s random startup delay.

2. **Event Saturation** — Too many low-value events drown important signals. **Prevention:** Event priority levels (CRITICAL, HIGH, MEDIUM, LOW, DEBUG), adaptive filtering (only log DEBUG events in debug mode), temporal decay (auto-prune low-priority events after 24 hours), event aggregation (combine similar events within time windows).

3. **LLM Test Flakiness from Non-Determinism** — Tests pass sometimes and fail other times without code changes. **Prevention:** Golden datasets with consistent test inputs, fuzzy assertions (semantic similarity not exact string matching), temperature control (0.0-0.2 for deterministic generation), consensus testing (run 3 times, require 2/3 pass).

4. **Visual Clutter from Emoji Overload** — CLI output becomes unreadable due to excessive emoji. **Prevention:** Visual hierarchy (use emojis only for state changes and critical alerts, 5-7 core indicators max), adaptive emoji support (detect terminal capabilities, fall back to text symbols), --plain flag to disable visual flourishes.

5. **Polling Without Backpressure** — Events queue up faster than agents can process, causing death spiral. **Prevention:** Backpressure monitoring (track queue depth, increase polling interval when queue > threshold), event batching (return max 10 events per poll), circuit breaker (stop polling if queue depth > 1000).

**Top v1 pitfalls (still relevant):**

1. **Context Rot in Long-Running Sessions** — LLM attention degrades beyond 50-100 messages. **Prevention:** Triple-layer memory with DAST compression (2.5x ratio), 20% context window budget (max 40k tokens of 200k limit), signal decay with explicit renewal.

2. **Infinite Spawning Loops** — Agents spawn specialists who spawn more specialists recursively. **Prevention:** Global spawn depth limit (max 3 levels), per-phase spawn quota (max 10 specialists), spawn circuit breaker (auto-triggers after 3 failed spawns in 5 minutes).

3. **JSON State Corruption from Race Conditions** — Multiple agents read/write same JSON file simultaneously. **Prevention:** File locking via `fcntl.flock()`, atomic write pattern (write to temp file, then atomic rename), state versioning with optimistic locking.

## Implications for Roadmap

Based on research, suggested phase structure for v2:

### Phase 1: Event Polling Integration
**Rationale:** Event polling is foundational for reactive agent behavior. Workers must call `get_events_for_subscriber()` to respond to async events (phase_complete, errors, spawn_requests). This is the highest-priority v2 feature and enables other reactive capabilities.

**Delivers:** Worker Ant prompts updated with polling sections, subscription patterns during initialization, event reaction logic (phase_complete, error, spawn_request).

**Addresses:** Event Polling Integration, Event Delivery Tracking, Event Filtering (from FEATURES.md table stakes).

**Avoids:** Polling thundering herd, polling without backpressure, event loss during async operations (from PITFALLS.md).

**Dependencies:** None (event bus already exists from v1 Phase 9).

**Research needed:** Standard patterns (event bus implementation verified). Skip `/cds:research-phase`.

### Phase 2: Visual Indicators & E2E Testing
**Rationale:** Visual indicators improve UX immediately and are independent of event polling. E2E testing guide validates both event polling and visual indicators, so it naturally follows implementation. These are medium-priority features that polish v2 for release.

**Delivers:** Command prompt output templates with emoji, progress bar formatting, structured section borders; E2E_TEST_GUIDE.md with scenarios (init, execute, spawn, memory, voting), expected outputs, verification checklists.

**Addresses:** Visual Activity Indicators, Progress Feedback, Manual E2E Test Guide (from FEATURES.md table stakes).

**Avoids:** Visual clutter from emoji overload, LLM test flakiness, test brittleness from exact assertions (from PITFALLS.md).

**Uses:** Unicode emojis, Markdown test guides (from STACK.md).

**Research needed:** Medium — CLI visual patterns have community consensus but need user validation. Consider `/cds:research-phase` for CLI UX patterns.

### Phase 3: Caste-Specific Event Sensitivity & Advanced Features
**Rationale:** Caste-specific sensitivity enhances event polling without changing its core. Once basic polling works, making it caste-specific is a natural improvement. Event-driven spawning triggers and visual dashboard are "nice to have" features that complete the v2 vision.

**Delivers:** Caste-specific event subscriptions (Colonizer prioritizes "spawn_request", Watcher prioritizes "error"), event-driven spawning triggers ("spawn_request" events trigger Bayesian spawning), visual pheromone strength indicators.

**Addresses:** Caste-Specific Event Sensitivity, Event-Driven Spawning Triggers, Visual Colony Activity Dashboard, Visual Pheromone Strength Indicators (from FEATURES.md competitive features).

**Uses:** Pheromone sensitivity profiles from v1, event polling from Phase 1.

**Research needed:** Low — extends existing patterns. Skip `/cds:research-phase`.

### Phase Ordering Rationale

**Why this order based on dependencies:**
- Phase 1 (Event Polling) must come first because it enables reactive behavior, foundational for async coordination. Visual indicators and testing both depend on event polling working.
- Phase 2 (Visuals + Testing) groups independent features that polish v2. Visual indicators are UX improvements that can be built in parallel with event polling. E2E testing validates both Phase 1 and Phase 2, so it naturally follows implementation.
- Phase 3 (Advanced Features) builds on Phase 1's event polling foundation. Caste-specific sensitivity and event-driven spawning are enhancements to the core polling mechanism.

**Why this grouping based on architecture patterns:**
- Phase 1 implements Event Polling Layer (from ARCHITECTURE.md)
- Phase 2 implements Visual Indicators Layer and E2E Testing Layer (from ARCHITECTURE.md)
- Phase 3 refines Event Polling Layer with caste-specific behavior

**How this avoids pitfalls from research:**
- Phase 1 avoids thundering herd by implementing jitter and backpressure from day one
- Phase 2 avoids visual clutter by following emoji hierarchy guidelines
- Phase 2 avoids test flakiness by using golden datasets and fuzzy assertions
- All phases avoid v1 pitfalls (context rot, infinite spawning, JSON corruption) by following established prevention patterns

### Research Flags

**Phases likely needing deeper research during planning:**
- **Phase 2 (CLI Visual Indicators):** Medium confidence on CLI UX patterns. Emoji usage has community consensus but lacks rigorous user testing. Consider `/cds:research-phase` to validate terminal compatibility and accessibility.

**Phases with standard patterns (skip research-phase):**
- **Phase 1 (Event Polling Integration):** HIGH confidence. Event bus implementation verified in codebase (879 lines). Pull-based polling pattern confirmed optimal for prompt-based agents.
- **Phase 2 (E2E Testing):** HIGH confidence. Manual LLM testing approach validated by multiple sources (Maxim AI, Orq.ai, QAWerk). TAP format already used in existing tests.
- **Phase 3 (Caste-Specific Sensitivity):** HIGH confidence. Extends existing pheromone sensitivity profiles from v1. Pattern is well-understood.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Verified against existing Aether codebase (event-bus.sh, atomic-write.sh, file-lock.sh). Pull-based polling pattern confirmed optimal for prompt-based agents. |
| Features | HIGH | Table stakes features identified from competitor analysis (AutoGen, LangGraph, CrewAI). Differentiators confirmed unique to Claude-native systems. |
| Architecture | HIGH | Integration points verified against v1 codebase. Component boundaries clear. Data flow patterns follow established event-driven architecture. |
| Pitfalls | HIGH | v1 pitfalls documented in CONCERNS.md (internal analysis). v2 pitfalls researched from multiple sources (academic papers, GitHub issues, engineering blogs). |

**Overall confidence:** HIGH

### Gaps to Address

- **Polling frequency tuning:** Research confirms "poll at execution boundaries" but doesn't define optimal frequency. How often is "often enough"? **Handle during implementation:** Start with "poll after every action" (file writes, command execution), measure latency, optimize in v3 if needed.
- **LLM judgment calibration:** Manual test guides require Claude to make judgment calls (e.g., "Is this response helpful?"). Research doesn't specify how to structure validation instructions for consistent judgment. **Handle during implementation:** Iterate on test guide instructions during Phase 2, gather feedback from manual test runs.
- **Emoji fallback testing:** Unicode emoji support verified for modern terminals, but ASCII fallback pattern not specified. **Handle during implementation:** Test emoji support during Phase 2, implement adaptive output (--plain flag) if terminals show ? boxes.
- **User validation for visual indicators:** Emoji and progress bars seem helpful, but no user feedback data. **Handle during implementation:** Gather user feedback after v2 ships, iterate on visual hierarchy in v3.

## Sources

### Primary (HIGH confidence)

**Official Aether codebase:**
- `.aether/utils/event-bus.sh` — 879 lines, pull-based event delivery pattern verified
- `.aether/utils/atomic-write.sh` — Corruption-safe write pattern
- `.aether/utils/file-lock.sh` — Concurrent access prevention
- `.planning/phases/10-colony-maturity/tests/integration/full-workflow.test.sh` — TAP format test pattern
- `.planning/codebase/CONCERNS.md` — Known pitfalls, race condition risks

**Official Anthropic documentation:**
- [Claude Code Sandboxing](https://www.anthropic.com/engineering/claude-code-sandboxing) — No persistent process support (validates pull-based approach)
- [Effective Context Engineering](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents) — Context optimization for testing

**Academic research:**
- [On the Flakiness of LLM-Generated Tests](https://arxiv.org/html/2601.08998v1) — Research on test flakiness (HIGH confidence, academic paper)
- [Understanding and Improving Flaky Test Classification](https://www.cs.cornell.edu/~saikatd/papers/flakylens-oopsla25.pdf) — Cornell research 2025 (HIGH confidence)
- [arXiv: 1/3 agents fall into infinite loops](https://arxiv.org/html/2512.01939v1) — Academic research on spawning loops (HIGH confidence)
- [Design, Implementation and Evaluation of a Real-Time Filtering System](https://arxiv.org/html/2508.18787v1) — Dynamic signal filtering (MEDIUM confidence)

### Secondary (MEDIUM confidence)

**Event polling patterns:**
- [Polling Is Not the Problem—Bad Polling Is](https://beingcraftsman.com/2025/12/31/polling-is-not-the-problem-bad-polling-is/) — Best practices for 2025
- [Data Ingestion Patterns: Push, Pull & Poll Explained (Dagster)](https://dagster.io/blog/data-ingestion-patterns-when-to-use-push-pull-and-poll) — Pull vs push use cases
- [Event-Driven Architecture: Watch Out For These Pitfalls](https://www.forbes.com/councils/forbestechcouncil/2025/11/26/event-driven-architecture-watch-out-for-these-pitfalls-and-drawbacks/) — Distributed system challenges

**LLM testing approaches:**
- [How to Evaluate AI Agents: A Practical Checklist for Production (Maxim AI)](https://www.getmaxim.ai/articles/how-to-evaluate-ai-agents-a-practical-checklist-for-production/) — Evaluation checklist
- [A Comprehensive Guide to Evaluating Multi-Agent LLM Systems (Orq.ai)](https://orq.ai/blog/multi-agent-llm-eval-system) — Multi-agent evaluation patterns
- [10 LLM Testing Strategies To Catch AI Failures](https://galileo.ai/blog/llm-testing-strategies) — Practical testing approaches

**CLI visual indicators:**
- [CLI UX best practices: 3 patterns for improving progress displays](https://evilmartians.com/chronicles/cli-ux-best-practices-3-patterns-for-improving-progress-displays) — Progress display patterns
- [Terminal UI Libraries (LFX Insights)](https://insights.linuxfoundation.org/collection/terminal-ui-libraries) — Terminal UI patterns

**Context rot & memory issues:**
- [Medium: Context rot confirmed real in 2025](https://medium.com/@umairamin2004/why-multi-agent-systems-fail-in-production-and-how-to-fix-them-3bedbdd4975b) — Multi-agent failure modes
- [Reddit: Claude Code context window strategy (20% rule)](https://www.reddit.com/r/ClaudeAI/comments/1p05r7p/my_claude_code_context_window_strategy_200k_is) — Community practice

**Infinite loops & spawning:**
- [Medium: Why multi-agent systems fail](https://medium.com/@umairamin2004/why-multi-agent-systems-fail-in-production-and-how-to-fix-them-3bedbdd4975b) — Failure modes analysis
- [GitHub: OpenAI agents infinite recursion issue](https://github.com/openai/openai-agents-python/issues/668) — Confirmed bug

**JSON state & race conditions:**
- [GitHub: Langflow race condition data corruption](https://github.com/langflow-ai/langflow/issues/8791) — Confirmed bug
- [Milvus Blog: Why Claude Code Feels So Stable](https://milvus.io/blog/why-claude-code-feels-so-stable-a-developers-deep-dive-into-its-local-storage-design.md) — JSONL for stability

### Tertiary (LOW confidence)

**Event-driven multi-agent systems:**
- [AI Agents Must Act, Not Wait: A Case for Event-Driven Multi-Agent Design (Medium)](https://seanfalconer.medium.com/ai-agents-must-act-not-wait-a-case-for-event-driven-multi-agent-design-d8007b50081f) — Event-driven design patterns
- [Stop Polling. Start Listening: Event-Driven Architecture](https://www.hkinfosoft.com/stop-polling.start-listening/the-power-of-event-driven-architecture/) — Pitfalls of polling

**CLI emoji usage:**
- [Add emoji/visual indicators to CLI output for better UX (GitHub Issue)](https://github.com/josharsh/mcp-jest/issues/19) — GitHub issue, single source
- [State of Terminal Emulators in 2025](https://news.ycombinator.com/item?id=45799478) — Community discussion

---
*Research completed: 2026-02-02*
*Ready for roadmap: yes*
