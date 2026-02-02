# Stack Research

**Domain:** Claude-native multi-agent system enhancements (event polling, LLM testing, CLI visual indicators)
**Researched:** 2026-02-02
**Confidence:** HIGH

## Executive Summary

This research focuses on the **technology stack for Aether v2 enhancements**: reactive event polling integration, E2E LLM test guides, and CLI visual indicators. These enhancements build on Aether v1's proven foundation (19 commands, 10 Worker Ants, 26 utility scripts, pub/sub event bus).

**Key findings:**
1. **Pull-based event polling** is the correct pattern for prompt-based Worker Ants (execute, poll, exit - not persistent daemons)
2. **Markdown test guides** executed by Claude provide the right balance of structure and LLM judgment for E2E testing
3. **Unicode emojis** (🐜, ✓, ✗, ⟳) offer semantic, color-independent visual indicators for colony activity

The stack remains deliberately minimal: **Bash + jq** for event infrastructure, **Markdown** for test documentation, **Unicode emojis** for visual feedback. No new dependencies required - all enhancements leverage existing Aether utilities (event-bus.sh, atomic-write.sh, file-lock.sh).

---

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| **Bash + jq** | Bash 4.0+, jq 1.6+ | Event polling infrastructure | Already used in Aether's event-bus.sh (879 lines). Proven pattern for pull-based event delivery. File locking via fcntl prevents race conditions. jq handles JSON querying efficiently. |
| **Markdown test guides** | GitHub Flavored Markdown | E2E LLM test documentation | Human-readable test cases that Claude can execute directly. No test runner dependencies. Fits Claude-native constraint (prompts as code). |
| **Unicode emojis** | Standard Unicode 15.0+ | CLI visual indicators | Terminal-compatible, no external dependencies. 🐜 for colony activity, ✓/✗ for status, ⟳ for processing. Works across macOS/Linux terminals. |
| **TAP format** | TAP version 13 | Test output standard | Already used in existing tests (full-workflow.test.sh). Human-readable, parseable by CI tools. Industry standard for test reporting. |
| **JSON state files** | JSON RFC 8259 | Test execution tracking | Claude reads/writes natively. Git-diffable for test history validation. No database overhead for test suites. |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| **atomic-write.sh** | Existing Aether utility | Corruption-safe test state updates | Use when writing test results JSON. Prevents test data corruption from concurrent test runs. |
| **file-lock.sh** | Existing Aether utility | Concurrent test execution safety | Use when tests access shared colony state. Prevents race conditions in parallel test scenarios. |
| **event-bus.sh** | Existing Aether utility (879 lines) | Event polling for test orchestration | Use `get_events_for_subscriber()` for reactive test triggers. Already implements pull-based delivery. |
| **event-metrics.sh** | Existing Aether utility | Test performance tracking | Use to capture test execution timing, event delivery latency. Already integrated with event bus. |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| **Manual test execution via Claude Code** | Run E2E LLM tests | Claude interprets test guide markdown, executes commands, validates results. No test runner needed. |
| **Git diff for test result validation** | Verify test outcomes | Test results committed to repo allow `git diff` to catch regressions. Part of documentation cleanup goal. |
| **Terminal with Unicode support** | Render emoji indicators | Most modern terminals (iTerm2, Terminal.app, GNOME Terminal) support Unicode 15.0 emojis. Verify with `echo "🐜"` |

---

## Installation

```bash
# Core (already installed in Aether)
# jq for JSON manipulation
brew install jq  # macOS
# apt-get install jq  # Linux

# Existing Aether utilities (no installation needed)
source .aether/utils/event-bus.sh
source .aether/utils/atomic-write.sh
source .aether/utils/file-lock.sh

# Dev dependencies (for test guide creation)
# None required - use existing Claude Code CLI
```

---

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| **Pull-based event polling** | Push-based webhooks | Push when you need real-time event delivery to external systems. Pull is better for prompt-based agents (Worker Ants execute, poll, exit - no persistent process). |
| **Markdown test guides** | Automated test frameworks (Jest, pytest) | Automated frameworks when you have deterministic, repeatable tests. Markdown guides for LLM tests where you need Claude's judgment (e.g., "Did this response meet quality standards?"). |
| **Unicode emojis** | ASCII art or colored output | ASCII when terminal doesn't support Unicode (legacy systems). Colored output when you need severity indicators (red=error, green=success). Emojis for semantic meaning (🐜=colony activity). |
| **TAP format** | Custom test output formats | Custom formats when you have specific reporting needs (e.g., JUnit XML for CI). TAP for simplicity and broad tool support. |
| **JSON test state** | SQLite or external database | SQLite for complex test relationships and queries. JSON for simple test tracking and git diffing. Aether's scope fits JSON. |

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| **Python async/await for event delivery** | Claude Code doesn't support persistent async processes. Worker Ants are prompt-based (execute, poll, exit), not long-running daemons. | Pull-based polling via `get_events_for_subscriber()` - Worker Ants call this when they execute, process events, mark delivered, exit. |
| **External test frameworks (Jest, Mocha, pytest)** | Violates "Claude-Native Only" constraint. Requires Node.js/Python runtime. Adds dependency overhead for simple validation tests. | Markdown test guides executed by Claude Code. Claude reads test steps, executes commands, validates results. |
| **WebSocket or real-time push notifications** | Worker Ants aren't persistent processes - can't maintain WebSocket connections. Push requires daemon processes. | Event bus polling: Workers pull events when they run. Publishers push to events.json (non-blocking), Workers pull on next execution. |
| **Heavy test runners with assertion libraries** | Overkill for manual LLM testing. You want Claude's judgment, not programmatic assertions (e.g., "Is this code review helpful?"). | Human-readable test guides with validation steps. Claude executes step, observes result, makes judgment call based on instructions. |
| **Colored terminal output libraries** | Color alone doesn't convey semantic meaning. Red/green problematic for colorblind users. | Emojis: 🐜 (colony activity), ✓ (success), ✗ (failure), ⟳ (processing), ⚠ (warning), 🔍 (inspecting). Universal symbols, color-independent. |
| **External vector databases for test artifacts** | Overkill for test suite scope. Adds infrastructure dependency. Aether constraint: "No External Dependencies." | JSON files for test results. Git-tracked, diff-able, human-readable. Search with jq when needed. |

---

## Stack Patterns by Variant

**If testing reactive event polling:**
- Use **pull-based event pattern** (Worker Ants call `get_events_for_subscriber()`)
- Because Worker Ants are prompt-based agents that execute and exit, not persistent processes
- Implement polling in Worker Ant prompts: "Before starting task, check for events by calling get_events_for_subscriber()"
- Event bus already supports this pattern (lines 508-593 in event-bus.sh)

**If testing E2E LLM workflows:**
- Use **markdown test guides** with step-by-step instructions
- Because Claude can interpret natural language test steps, execute commands, validate results
- Structure: Test scenario → Setup → Execution steps → Validation assertions → Cleanup
- Example test guide structure mirrors existing full-workflow.test.sh but in markdown format

**If adding CLI visual indicators:**
- Use **Unicode emoji prefixes** for log output
- Because emojis convey semantic meaning without color dependencies
- Pattern: `🐜 [CASTE] message` for colony activity, `✓ Test passed` for validation
- Ensure terminal compatibility (test with `echo "🐜"` first)

**If running concurrent tests:**
- Use **file locking** via existing `file-lock.sh`
- Because tests may access shared colony state (COLONY_STATE.json, events.json)
- Pattern: Acquire lock before reading/writing shared state, release after
- Prevents test race conditions (known pitfall from CONCERNS.md)

---

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| **jq 1.6+** | Bash 4.0+, macOS 10.15+, all Linux distros | jq required for all event bus operations. Aether's event-bus.sh uses jq extensively (879 lines). |
| **Bash 4.0+** | All macOS 10.15+, Linux kernels 3.0+ | Bash arrays used in event polling. Bash 3.x (macOS default) may have issues - upgrade with Homebrew. |
| **Unicode 15.0 emojis** | Terminal.app (macOS 10.13+), iTerm2, GNOME Terminal 3.28+ | Legacy terminals may display ? box fallback. Test emoji support before using in production. |
| **TAP version 13** | Any TAP-compatible test harness (prove, tap-runner, CI parsers) | Existing Aether tests use TAP format (full-workflow.test.sh line 18). Maintains compatibility. |
| **JSON RFC 8259** | Any JSON parser (jq, python3 -m json.tool) | All state files use valid JSON. Cross-platform compatible. |

---

## Event Polling Architecture (Pull-Based)

### Current Implementation (v1)

Aether's event bus ([`.aether/utils/event-bus.sh`](/.aether/utils/event-bus.sh), 879 lines) implements **pull-based delivery**:

```bash
# Publisher (Worker Ant) publishes event
publish_event "topic" "type" '{"data": "value"}' "publisher" "caste"
# → Writes to events.json event_log
# → Returns immediately (non-blocking)

# Subscriber (Worker Ant) polls for events
EVENTS=$(get_events_for_subscriber "subscriber_id" "caste")
# → Returns events matching subscriptions since last poll
# → Worker processes events when it executes

# Mark events as delivered
mark_events_delivered "subscriber_id" "caste" "$EVENTS"
```

### Why Pull-Based for Worker Ants?

From research on [push vs pull patterns](https://dagster.io/blog/data-ingestion-patterns-when-to-use-push-pull-and-poll):

1. **Backpressure control** - Workers pull work they can handle (no overwhelming events)
2. **No persistent processes** - Worker Ants execute and exit (can't maintain WebSocket connections)
3. **Fault tolerance** - Workers can retry failed pulls (network blips don't lose events)
4. **Resource management** - Workers control their own polling frequency

### Enhancement for v2

**Reactive event integration**: Worker Ants proactively call `get_events_for_subscriber()` at key points:

1. **On spawn** - New Worker checks for relevant events
2. **Before task execution** - Check for FOCUS/REDIRECT signals
3. **After task completion** - Check for FEEDBACK signals
4. **On error** - Check for error handling events

**Pattern in Worker Ant prompts:**
```markdown
## Event Polling

Before starting any task:
1. Call get_events_for_subscriber() with your worker_id and caste
2. Process any FOCUS signals (adjust attention)
3. Process any REDIRECT signals (avoid approaches)
4. Process any FEEDBACK signals (learn preferences)
5. Mark events as delivered after processing
```

### Avoid: Push-Based Event Delivery

- **Why**: Requires persistent daemon processes (incompatible with prompt-based Worker Ants)
- **Alternative**: Workers poll when they execute (pull-based)
- **Hybrid option**: Publishers push to events.json (non-blocking write), Workers pull on next execution

---

## LLM Testing Approach (Manual Test Guides)

### Current Testing (v1)

Aether has 9 automated tests (TAP format, bash scripts):
- Integration tests (full-workflow, autonomous-spawn, memory-compress, voting-verify, meta-learning)
- Stress tests (concurrent-access, spawn-limits, event-scalability)
- Performance tests (timing-baseline)

**Gap**: These tests validate infrastructure (JSON state, event bus), not LLM behavior quality.

### Enhancement for v2

**E2E LLM test guide**: Manual test suite for real Queen/Worker execution.

Based on research from [Maxim AI's evaluation checklist](https://www.getmaxim.ai/articles/how-to-evaluate-ai-agents-a-practical-checklist-for-production/) and [multi-agent LLM eval guide](https://orq.ai/blog/multi-agent-llm-eval-system):

#### Test Guide Structure

```markdown
# E2E LLM Test Guide: [Test Name]

## Test Scenario
[Description of the colony behavior being tested]

## Pre-Conditions
- Colony initialized with goal: "[goal]"
- Worker Ants available: [list]
- Event bus subscribed: [topics]

## Test Steps

### Step 1: [Action]
1. Execute: `/ant:init "[goal]"`
2. Verify: Colony state = INIT
3. Check: events.json contains INIT pheromone

### Step 2: [Action]
...

## Validation Assertions

### Assertion 1: Colony spawned correct Worker Ants
- Expected: Colonizer, Route-setter, Builder
- Check: `jq '.active_workers | map(.caste)' .aether/data/worker_ants.json`
- LLM judgment: Are these castes appropriate for the goal?

### Assertion 2: Worker Ants responded to pheromones
- Expected: Workers reacted to INIT signal
- Check: events.json shows task_started events
- LLM judgment: Did workers self-organize correctly?

## Post-Conditions
- Colony state = COMPLETED
- All Worker Ants status = IDLE
- events.json contains complete event chain

## Cleanup
```bash
./.aether/utils/cleanup-colony.sh
```
```

#### Key LLM Evaluation Criteria

From [AI agent evaluation metrics](https://qawerk.com/blog/ai-agent-evaluation-metrics/):

1. **Task completion** - Did the colony achieve the goal?
2. **Agent coordination** - Did Worker Ants communicate effectively via events?
3. **Emergent behavior** - Did workers spawn appropriate specialists?
4. **Pheromone response** - Did workers respond to INIT/FOCUS/REDIRECT signals?
5. **State consistency** - Was colony state maintained correctly?

#### Test Categories

1. **Happy path** - Colony achieves goal successfully
2. **Error recovery** - Colony handles failures gracefully
3. **Edge cases** - Unusual goals, resource constraints
4. **Coordination** - Multiple workers collaborating
5. **Autonomous spawning** - Workers detect capability gaps and spawn specialists

### Avoid: Automated Assertion Libraries

- **Why**: LLM outputs are non-deterministic. Programmatic assertions can't capture "Is this response helpful?"
- **Alternative**: Claude executes test guide, makes judgment calls based on instructions
- **Hybrid**: Use bash assertions for JSON validation (state transitions), Claude judgment for LLM quality (response helpfulness)

---

## CLI Visual Indicators (Emoji Conventions)

### Emoji Standard (Unicode 15.0)

Based on research from [CLI emoji enhancement practices](https://github.com/josharsh/mcp-jest/issues/19) and [terminal UI libraries](https://insights.linuxfoundation.org/collection/terminal-ui-libraries):

#### Colony Activity Indicators

```bash
🐜 [CASTE] message              # Colony activity (all castes)
🐜 [QUEEN] Colony initialized with goal: "Build REST API"
🐜 [BUILDER] Writing code: src/api/auth.js
🐜 [WATCHER] Verifying test coverage
```

#### Status Indicators

```bash
✓ Test passed                  # Success
✗ Test failed                  # Failure
⚠ Warning detected             # Warning
ℹ Information                  # Info
```

#### Progress Indicators

```bash
⟳ Processing...                # In progress
⏳ Waiting for events          # Polling/waiting
⚡ Fast operation               # Quick operation
🔍 Inspecting                  # Investigating/debugging
```

#### Event Indicators

```bash
📡 Event published             # Event bus activity
📨 Event received              # Event delivery
📋 Event log                   # Event history
```

#### State Machine Indicators

```bash
→ State transition: IDLE → INIT  # State change
⏭ Checkpoint created           # State checkpoint
↩ State recovery               # State rollback
```

#### Caste-Specific Indicators

```bash
🐜 [COLONIZER] Exploring codebase...
🐜 [ROUTE-SETTER] Planning phase...
🐜 [BUILDER] Implementing feature...
🐜 [WATCHER] Validating output...
🐜 [SCOUT] Researching...
🐜 [ARCHITECT] Compressing memory...
```

### Implementation Pattern

Add emoji prefixes to Worker Ant prompt output:

```markdown
## Output Format

When reporting colony activity, use emoji prefixes:
- 🐜 [CASTE] for colony activity
- ✓ for successful operations
- ✗ for failures
- ⚠ for warnings

Example outputs:
🐜 [BUILDER] ✓ Created src/api/auth.js
🐜 [WATCHER] ✗ Test coverage below 80%
🐜 [QUEEN] ⚠ Resource budget: 8/10 spawns used
```

### Terminal Compatibility

Test emoji support before using:

```bash
# Test emoji rendering
echo "🐜 ✓ ✗ ⚠ ⟳ 📡"

# If you see ? boxes, your terminal doesn't support Unicode emojis
# Fallback: Use ASCII prefixes
# [COLONY] for activity
# [OK] for success
# [FAIL] for failure
```

### Avoid: Color-Only Indicators

- **Why**: Color alone doesn't convey semantic meaning. Red/green problematic for colorblind users.
- **Alternative**: Emojis provide semantic meaning independent of color
- **Best practice**: Combine emoji + color for accessibility (🐜 green = colony activity success)

---

## Sources

### Primary (HIGH confidence)

**Official Aether codebase:**
- [`.aether/utils/event-bus.sh`](/.aether/utils/event-bus.sh) — 879 lines, pull-based event delivery pattern verified
- [`.aether/utils/atomic-write.sh`](/.aether/utils/atomic-write.sh) — Corruption-safe write pattern
- [`.aether/utils/file-lock.sh`](/.aether/utils/file-lock.sh) — Concurrent access prevention
- [`.planning/phases/10-colony-maturity/tests/integration/full-workflow.test.sh`](/.planning/phases/10-colony-maturity**---end-to-end-testing,-pattern-extraction,-production-readiness/tests/integration/full-workflow.test.sh) — TAP format test pattern
- [`.planning/codebase/CONCERNS.md`](/.planning/codebase/CONCERNS.md) — Known pitfalls, race condition risks

**Official Anthropic documentation:**
- [Claude Code Sandboxing](https://www.anthropic.com/engineering/claude-code-sandboxing) — No persistent process support (validates pull-based approach)
- [Effective Context Engineering](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents) — Context optimization for testing

### Secondary (MEDIUM confidence)

**Event polling patterns:**
- [Data Ingestion Patterns: Push, Pull & Poll Explained (Dagster)](https://dagster.io/blog/data-ingestion-patterns-when-to-use-push-pull-and-poll) — Pull vs push use cases (validated pull-based for worker agents)
- [Push vs Pull Architecture (Medium)](https://medium.com/@aligolestan/push-vs-pull-architecture-understanding-the-two-communication-models-ebe24a4eb4e6) — Trade-offs between push/pull
- [Event Driven Architecture – Push vs Pull (Wellsky Engineering)](https://engineering.wellsky.com/post/event-driven-architecture---push-vs-pull) — When to use each pattern

**LLM testing approaches:**
- [How to Evaluate AI Agents: A Practical Checklist for Production (Maxim AI)](https://www.getmaxim.ai/articles/how-to-evaluate-ai-agents-a-practical-checklist-for-production/) — Evaluation checklist (informed test guide structure)
- [A Comprehensive Guide to Evaluating Multi-Agent LLM Systems (Orq.ai)](https://orq.ai/blog/multi-agent-llm-eval-system) — Multi-agent evaluation patterns
- [AI Agent Evaluation: Key Metrics to Measure Performance (QAWerk)](https://qawerk.com/blog/ai-agent-evaluation-metrics) — Performance metrics for test validation
- [A Practical Guide for Evaluating LLMs and LLM-Reliant Systems (arXiv)](https://arxiv.org/html/2506.13023v1) — Academic evaluation frameworks

**CLI visual indicators:**
- [Add emoji/visual indicators to CLI output for better UX (GitHub Issue)](https://github.com/josharsh/mcp-jest/issues/19) — CLI emoji enhancement discussion
- [Terminal UI Libraries (LFX Insights)](https://insights.linuxfoundation.org/collection/terminal-ui-libraries) — Terminal UI patterns
- [Rich Output Formatting (gookit/gcli)](https://zread.ai/gookit/gcli/15-rich-output-formatting) — Emoji subsystem in CLI tools

### Tertiary (LOW confidence)

**Event-driven multi-agent systems:**
- [AI Agents Must Act, Not Wait: A Case for Event-Driven Multi-Agent Design (Medium)](https://seanfalconer.medium.com/ai-agents-must-act-not-wait-a-case-for-event-driven-multi-agent-design-d8007b50081f) — Event-driven design patterns
- [A Distributed State of Mind: Event-Driven Multi-Agent Systems (Medium)](https://seanfalconer.medium.com/a-distributed-state-of-mind-event-driven-multi-agent-systems-226785b479e6) — Multi-agent coordination patterns
- [Best Architectural Patterns for Event-Driven Systems (Gravitee)](https://www.gravitee.io/blog/event-driven-architecture-patterns) — Event sourcing patterns

---

## Confidence Assessment

| Area | Confidence | Reasoning |
|------|------------|-----------|
| **Pull-based event polling** | HIGH | Verified against existing Aether event-bus.sh (879 lines). Research confirms pull pattern is correct for prompt-based agents. |
| **Markdown test guides** | HIGH | Research from multiple sources (Maxim AI, Orq.ai, QAWerk) confirms manual LLM testing approach. Fits Claude-native constraints. |
| **Unicode emoji indicators** | MEDIUM | Research confirms emoji usage in CLI tools. Terminal compatibility verified. Low confidence only on emoji fallback strategies. |
| **TAP format for tests** | HIGH | Existing Aether tests use TAP. Industry standard. No compatibility concerns. |
| **File locking for concurrency** | HIGH | Existing Aether utilities (file-lock.sh) implement this. Pattern verified against CONCERNS.md race condition risks. |

**Overall confidence:** HIGH

---

## Open Questions (Phase-Specific Research)

1. **Emoji fallback testing**: What's the best ASCII fallback pattern for legacy terminals? Test across terminal types during implementation.
2. **LLM judgment calibration**: How to structure test guide validation instructions for consistent Claude judgment? Iterate during test guide creation.
3. **Event polling frequency**: How often should Worker Ants poll? (Current: on key execution points). May need tuning based on real usage.

**Recommendation:** These are implementation details, not stack decisions. The stack is solid. Research these during implementation phases.

---

## Conclusion

**Aether v2's enhancement stack remains minimal:**

1. **Pull-based event polling** - Use existing event-bus.sh, Workers call `get_events_for_subscriber()`
2. **Markdown test guides** - Claude executes test steps, validates with LLM judgment
3. **Unicode emoji indicators** - Semantic, color-independent visual feedback (🐜, ✓, ✗, ⟳)

**No new dependencies required.** All enhancements leverage existing Aether infrastructure.

**Why this is the right stack:**
- **Claude-native**: Pull-based polling fits prompt-based agent model (execute, poll, exit)
- **Debuggable**: Markdown test guides are human-readable, git-tracked
- **Accessible**: Emojis provide semantic meaning independent of color
- **Compatible**: Builds on existing event-bus.sh (879 lines), atomic-write.sh, file-lock.sh
- **Future-proof**: Based on standard patterns (pull events, TAP tests, Unicode emojis)

**The stack enables the enhancements without introducing complexity.**

---

*Stack research for: Aether v2 enhancements*
*Researched: 2026-02-02*
*Confidence: HIGH*
