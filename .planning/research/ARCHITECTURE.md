# Architecture Patterns: Aether v2 Integration

**Domain:** Reactive event integration, LLM testing, and CLI visual indicators for Claude-native multi-agent systems
**Researched:** 2026-02-02
**Overall confidence:** HIGH

## Executive Summary

Aether v2 enhances the existing Claude-native Queen Ant Colony with three critical integrations: **reactive event polling**, **E2E LLM testing**, and **visual process indicators**. These features integrate into the existing architecture without breaking constraints: prompt-based agents, JSON persistence, and zero external dependencies.

**Key architectural insight:** The v1 event bus (Phase 9) implemented pull-based async delivery optimal for prompt-based Worker Ants. However, Worker Ant prompts don't yet call `get_events_for_subscriber()`. v2 integrates event polling into Worker Ant execution loops, enabling reactive behavior: Workers spawn autonomously when events indicate capability gaps, errors trigger Watcher verification, and phase completion coordinates colony-wide transitions.

The research reveals three integration points:
1. **Event Polling Integration** - Worker Ant prompts call `get_events_for_subscriber()` at execution boundaries
2. **E2E Testing Guide** - Manual test suite covering init → execute → spawn → memory → voting workflows
3. **Visual Indicators** - CLI output formatting (emoji, progress bars, section headers) for colony activity visibility

## Recommended Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    AETHER V2 INTEGRATION LAYER                  │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  EVENT POLLING LAYER (NEW)                              │  │
│  │  Worker Ant prompts call get_events_for_subscriber()     │  │
│  │  Integration points: Colonizer, Builder, Watcher, Scout  │  │
│  └──────────────────────────────────────────────────────────┘  │
│                          ↓                                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  VISUAL INDICATORS LAYER (NEW)                          │  │
│  │  CLI output formatting: emoji, progress bars, sections  │  │
│  │  Integration points: All command prompts                 │  │
│  └──────────────────────────────────────────────────────────┘  │
│                          ↓                                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  E2E TESTING LAYER (NEW)                                │  │
│  │  Manual test guide: init → execute → spawn → verify     │  │
│  │  Output: docs/E2E_TEST_GUIDE.md                          │  │
│  └──────────────────────────────────────────────────────────┘  │
│                          ↓                                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  EXISTING AETHER V1 ARCHITECTURE                        │  │
│  │  Command Layer → State → Memory → Comm → Orchestration  │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### Component Boundaries

| Component | Responsibility | Communicates With | Integration Point |
|-----------|---------------|-------------------|-------------------|
| **Event Polling Layer** | Worker Ants poll events at execution boundaries | Event bus (get_events_for_subscriber), State Layer (publish events) | Worker Ant prompts (Colonizer, Builder, Watcher, Scout) |
| **Visual Indicators Layer** | Format CLI output with emoji, progress bars, sections | All command prompts (via template patterns) | Command prompts (init.md, status.md, execute.md, etc.) |
| **E2E Testing Layer** | Manual test guide covering core workflows | All layers (tests end-to-end) | Documentation (E2E_TEST_GUIDE.md) |

### Data Flow

**1. Event Polling Flow (NEW)**
```
Worker Ant executes task
    ↓
Execution boundary (file write, command complete, phase transition)
    ↓
Poll: get_events_for_subscriber(subscriber_id, caste)
    ↓
Events returned: [events matching subscriptions since last poll]
    ↓
Process events:
  - phase_complete → Update state, trigger transition
  - error → Log error, spawn Watcher if critical
  - spawn_request → Check capability gap, spawn specialist
  - task_started/task_completed → Update meta-learning
    ↓
mark_events_delivered(subscriber_id, events)
    ↓
Continue execution or react to events
```

**2. Visual Indicators Flow (NEW)**
```
Command prompt executes
    ↓
Output section with emoji markers:
  🐜 Colony activity
  📊 Progress metrics
  ⚠️ Warnings
  ✓ Success confirmations
    ↓
Progress bars: [████████....] 80%
    ↓
Structured sections with borders:
╔══════════════════════════════════════╗
║  SECTION HEADER                      ║
╠══════════════════════════════════════╣
║  Content                             ║
╚══════════════════════════════════════╝
```

**3. E2E Testing Flow (NEW)**
```
User opens E2E_TEST_GUIDE.md
    ↓
Follow test scenario:
  1. Initialize colony: /ant:init "Build REST API"
  2. Check status: /ant:status
  3. Execute phase: /ant:execute 1
  4. Spawn specialist: (triggered by autonomous spawning)
  5. Verify memory: /ant:memory search
  6. Emit pheromone: /ant:focus "auth"
  7. Review vote: (multi-perspective verification)
    ↓
Record results: ✓/✗ for each step
    ↓
Report bugs or gaps
```

## Patterns to Follow

### Pattern 1: Event Polling at Execution Boundaries

**What:** Worker Ant prompts call `get_events_for_subscriber()` after significant actions (file writes, command completion, phase transitions).

**When:** Workers need to react asynchronously to colony events (phase complete, errors, spawn requests).

**Example:**
```bash
# In Worker Ant prompt (e.g., builder-ant.md)

## Your Workflow

### 5. Poll for Events After Actions
After completing significant actions (file writes, command execution), poll for events:

```bash
# Source event bus
source .aether/utils/event-bus.sh

# Poll for events
SUBSCRIBER_ID="builder_$(jq -r '.colony_metadata.session_id' .aether/data/COLONY_STATE.json)"
CASTE="builder"

EVENTS=$(get_events_for_subscriber "$SUBSCRIBER_ID" "$CASTE")
EVENT_COUNT=$(echo "$EVENTS" | jq 'length')

if [ "$EVENT_COUNT" -gt 0 ]; then
  echo "🔔 Received $EVENT_COUNT events"

  # Process events
  echo "$EVENTS" | jq -r '.[] | "\(.topic): \(.type)"' | while read -r event_line; do
    echo "  → $event_line"
  done

  # Mark events as delivered
  mark_events_delivered "$SUBSCRIBER_ID" "$CASTE" "$EVENTS"

  # React to critical events
  PHASE_COMPLETE=$(echo "$EVENTS" | jq '[.[] | select(.topic == "phase_complete")] | length')
  if [ "$PHASE_COMPLETE" -gt 0 ]; then
    echo "⚠️ Phase completed - updating state"
    # Update state, prepare for next phase
  fi
fi
```
```

**Why:** Enables reactive Worker Ant behavior without persistent processes or background daemons. Workers poll when they execute, not continuously.

### Pattern 2: Subscription During Initialization

**What:** Worker Ants subscribe to relevant event topics when spawned or during phase initialization.

**When:** Worker needs to receive specific event types (errors, spawn requests, phase transitions).

**Example:**
```bash
# In Worker Ant spawn template or command initialization

## Subscribe to Events

```bash
# Source event bus
source .aether/utils/event-bus.sh

# Subscribe to relevant topics
SUBSCRIBER_ID="builder_$(jq -r '.colony_metadata.session_id' .aether/data/COLONY_STATE.json)_$(date +%s)"
CASTE="builder"

# Subscribe to error events
subscribe_to_events "$SUBSCRIBER_ID" "$CASTE" "error" '{}' > /dev/null

# Subscribe to spawn requests
subscribe_to_events "$SUBSCRIBER_ID" "$CASTE" "spawn_request" '{"target_caste": "builder"}' > /dev/null

# Subscribe to phase completion
subscribe_to_events "$SUBSCRIBER_ID" "$CASTE" "phase_complete" '{}' > /dev/null

echo "✓ Subscribed to events (subscriber_id: $SUBSCRIBER_ID)"
```
```

**Why:** Ensures Workers receive relevant events. Filter criteria prevent event spam (e.g., only spawn_requests targeting this caste).

### Pattern 3: Visual Indicators in Command Output

**What:** CLI output uses emoji, progress bars, and structured sections for visibility.

**When:** All command prompts (init, status, execute, focus, etc.) need clear, scannable output.

**Example:**
```markdown
## Step 7: Present Results

Show the Queen (user) the colony initialization:

```
╔══════════════════════════════════════════════════════════════╗
║  🐜 Queen Ant Colony Initialized                             ║
╠══════════════════════════════════════════════════════════════╣
║  Session: {session_id}                                       ║
║  Initialized: {timestamp}                                    ║
║                                                               ║
║  Queen's Intention:                                           ║
║  "{goal}"                                                    ║
║                                                               ║
║  Colony Status: INIT                                         ║
║  Current Phase: 1 - Colony Foundation                        ║
║  Roadmap: 10 phases ready                                    ║
║                                                               ║
║  Active Pheromones:                                          ║
║  ✓ INIT (strength 1.0, persists)                             ║
║                                                               ║
║  Worker Ants Mobilized:                                      ║
║  ✓ Colonizer (ready)                                         ║
║  ✓ Route-setter (ready)                                      ║
║  ✓ Builder (ready)                                           ║
║  ✓ Watcher (ready)                                           ║
║  ✓ Scout (ready)                                             ║
║  ✓ Architect (ready)                                         ║
╚══════════════════════════════════════════════════════════════╝

✨ COLONY MOBILIZED

Next Steps:
  /ant:status   - View detailed colony status
  /ant:plan     - Show full 10-phase roadmap
  /ant:phase 1  - Review Phase 1 details
  /ant:focus    - Guide colony attention (optional)
```
```

**Why:** Users can quickly scan colony state at a glance. Emoji indicate status (🐜 activity, ✓ success, ⚠️ warning, ✗ error). Progress bars show completion percentage.

### Pattern 4: E2E Test Guide Structure

**What:** Manual test guide with step-by-step scenarios covering all core workflows.

**When:** Validating end-to-end functionality before release.

**Example:**
```markdown
# E2E Test Guide for Aether v2

## Test Scenario 1: Colony Initialization

**Goal:** Verify colony initializes correctly with all state files.

**Steps:**
1. Run: `/ant:init "Build a REST API with authentication"`
2. Verify:
   - [ ] Output shows "🐜 Queen Ant Colony Initialized"
   - [ ] Session ID displayed
   - [ ] Queen's intention shown
   - [ ] All 6 Worker Ants show "✓ (ready)"
   - [ ] INIT pheromone active
3. Check state files:
   ```bash
   cat .aether/data/COLONY_STATE.json | jq '.queen_intention'
   cat .aether/data/pheromones.json | jq '.active_pheromones'
   cat .aether/data/worker_ants.json | jq '.castes'
   ```
4. Expected: All JSON files valid, intention set, workers mobilized

**Result:** ✓ PASS / ✗ FAIL

**Notes:**
```

**Why:** Manual testing catches issues automated tests miss (UX, edge cases, integration bugs). Guide serves as documentation too.

## Anti-Patterns to Avoid

### Anti-Pattern 1: Polling in Tight Loops

**What:** Calling `get_events_for_subscriber()` repeatedly in a loop without delay.

**Why bad:** Wastes computation, no events arrive between polls (event bus is async, not real-time).

**Instead:** Poll only at execution boundaries (after file writes, command completion, phase transitions). Worker Ants are prompt-based (execute and exit), not persistent processes.

### Anti-Pattern 2: Blocking on Events

**What:** Waiting for events to arrive before continuing execution.

**Why bad:** Defeats async design. Workers should execute autonomously, not wait for events.

**Instead:** Poll, process available events, continue. If critical event missing, note it and proceed. Workers can check again on next execution boundary.

### Anti-Pattern 3: Over-Subscription

**What:** Subscribing to all events without filter criteria.

**Why bad:** Event spam, irrelevant events, wasted processing.

**Instead:** Subscribe to specific topics with filter criteria (e.g., `spawn_request` with `{"target_caste": "builder"}`). Use wildcard patterns carefully (e.g., `error.*` vs `*`).

### Anti-Pattern 4: Visual Indicator Clutter

**What:** Adding emoji to every line, excessive progress bars, noisy output.

**Why bad:** Reduces scannability, users tune out.

**Instead:**
- Use emoji sparingly (section headers, status indicators)
- One progress bar per metric (don't show multiple bars for same thing)
- Structured sections with clear borders
- Consistent emoji meanings (🐜 activity, ✓ success, ⚠️ warning, ✗ error)

### Anti-Pattern 5: E2E Tests Requiring Automated Execution

**What:** Writing E2E tests as scripts that must be run automatically.

**Why bad:** Aether is Claude-native (prompt-based), not script-based. Automated tests miss UX issues.

**Instead:** Manual test guide that humans follow. Document expected outputs and verification steps. Users can execute commands in Claude and observe results.

## Scalability Considerations

| Concern | At 10 events/min | At 100 events/min | At 1000 events/min |
|---------|------------------|-------------------|--------------------|
| **Polling overhead** | Negligible (<1ms per poll) | Acceptable (<10ms per poll) | Consider polling frequency |
| **Event log size** | events.json < 100KB | events.json ~1MB | Ring buffer trims old events |
| **Subscription matching** | Linear scan fine | Linear scan acceptable | May need indexing |
| **Visual indicators** | No impact | No impact | No impact (CLI output) |
| **E2E test time** | ~10 minutes | ~20 minutes | ~30 minutes |

## Integration Points by Worker Ant

### Colonizer Ant
**Event subscriptions:**
- `phase_complete` - React to phase completion for new codebase analysis
- `spawn_request` with `{"capability": "codebase_analysis"}` - Spawn when requested

**Polling points:**
- After semantic index built
- After pattern detection complete
- Before spawning specialist

**Visual indicators:**
- 🐜 "Colonizing codebase..."
- ✓ "Semantic index built: {file_count} files"
- 📊 "Patterns detected: {pattern_count}"

### Builder Ant
**Event subscriptions:**
- `error` - React to build errors
- `spawn_request` with `{"target_caste": "builder"}` - Spawn when requested
- `task_started` - Track parallel work

**Polling points:**
- After file writes
- After command execution
- Before spawning specialist

**Visual indicators:**
- 🏗️ "Building: {task_description}"
- ✓ "Build complete: {files_changed} files"
- ✗ "Build failed: {error}"

### Watcher Ant
**Event subscriptions:**
- `task_completed` - Verify completed work
- `error` - Critical errors trigger verification
- `phase_complete` - Phase completion verification

**Polling points:**
- After validation complete
- After spawning 4 parallel Watchers
- Before voting

**Visual indicators:**
- 🔍 "Verifying: {work_description}"
- ✓ "Verification passed: {score}/10"
- ⚠️ "Issues found: {issue_count}"

### Scout Ant
**Event subscriptions:**
- `spawn_request` with `{"capability": "research"}` - Spawn when research needed
- `error` with `{"type": "unknown_domain"}` - Research unknown domains

**Polling points:**
- After information gathering
- After documentation search
- Before spawning specialist

**Visual indicators:**
- 🔭 "Researching: {topic}"
- ✓ "Found: {result_count} results"
- 📚 "Documentation: {doc_count} sources"

## Build Order Implications

Based on dependencies between event polling, testing, and visuals:

### Phase 1: Event Polling Integration (HIGH PRIORITY)
**Components:** Event polling in Worker Ant prompts
**Why:** Enables reactive behavior, foundational for async coordination

**Deliverables:**
- Worker Ant prompts updated with polling sections
- Subscription patterns during initialization
- Event reaction logic (phase_complete, error, spawn_request)

**Dependencies:** None (event bus already exists from v1 Phase 9)

### Phase 2: Visual Indicators (MEDIUM PRIORITY)
**Components:** CLI output formatting
**Why:** Improves UX, makes colony activity visible

**Deliverables:**
- Command prompt output templates with emoji
- Progress bar formatting
- Structured section borders

**Dependencies:** None (independent feature)

### Phase 3: E2E Testing Guide (LOW PRIORITY)
**Components:** Manual test suite
**Why:** Validates integration, can be done after features work

**Deliverables:**
- E2E_TEST_GUIDE.md with scenarios
- Expected outputs documented
- Verification checklists

**Dependencies:** Phase 1 and 2 (tests event polling and visuals)

## Architectural Tradeoffs

| Decision | Option A (Recommended) | Option B | Why A |
|----------|------------------------|----------|-------|
| **Polling frequency** | At execution boundaries only | Continuous polling loop | Workers are prompt-based (execute/exit), not persistent |
| **Event reaction** | Process and continue | Block and wait for events | Async design, no blocking |
| **Visual indicators** | Emoji + progress bars | ANSI color codes | Emoji work in all terminals, no color configuration |
| **E2E tests** | Manual guide | Automated test script | Aether is Claude-native, not script-based |
| **Subscription scope** | Specific topics with filters | Wildcard for all events | Prevent event spam, reduce processing |

## Comparison with v1 Architecture

| Aspect | v1 (Phase 9) | v2 (Integration) |
|--------|--------------|------------------|
| **Event bus** | Implemented (event-bus.sh) | Workers now call get_events_for_subscriber() |
| **Worker behavior** | Autonomous spawning | Reactive to events (phase complete, errors) |
| **CLI output** | Basic text | Structured with emoji, progress bars |
| **Testing** | Unit tests per component | E2E manual test guide |
| **Visibility** | State files only | Visual indicators in command output |
| **Coordination** | Pheromone signals | Events + pheromones (hybrid) |

## Implementation Checklist

### Event Polling Integration
- [ ] Add `source .aether/utils/event-bus.sh` to Worker Ant prompts
- [ ] Add subscription section to Worker Ant initialization
- [ ] Add polling section after execution boundaries
- [ ] Add event reaction logic (phase_complete, error, spawn_request)
- [ ] Update Worker Ant prompts: colonizer-ant.md, builder-ant.md, watcher-ant.md, scout-ant.md
- [ ] Test: Publish event, verify Worker Ant receives on next execution

### Visual Indicators
- [ ] Update command prompts with emoji: init.md, status.md, execute.md
- [ ] Add progress bar formatting to status output
- [ ] Add structured sections with borders
- [ ] Define emoji meanings (🐜 activity, ✓ success, ⚠️ warning, ✗ error)
- [ ] Test: Run commands, verify output is scannable

### E2E Testing
- [ ] Create E2E_TEST_GUIDE.md in /docs
- [ ] Document test scenarios: init, execute, spawn, memory, voting
- [ ] Add expected outputs for each scenario
- [ ] Add verification checklists
- [ ] Test: Follow guide manually, verify all steps work

## Sources

### HIGH Confidence (Official Documentation & Codebase Analysis)

- `/Users/callumcowie/repos/Aether/.aether/utils/event-bus.sh` - Event bus implementation (878 lines, verified)
- `/Users/callumcowie/repos/Aether/.aether/utils/event-metrics.sh` - Event metrics tracking (231 lines, verified)
- `/Users/callumcowie/repos/Aether/.aether/utils/test-event-async.sh` - Async delivery tests (195 lines, verified)
- `/Users/callumcowie/repos/Aether/.planning/phases/09-stigmergic-events/09-stigmergic-events-VERIFICATION.md` - Phase 9 verification (47/47 must-haves)
- `/Users/callumcowie/repos/Aether/.aether/workers/builder-ant.md` - Builder Ant prompt (571 lines, analyzed)
- `/Users/callumcowie/repos/Aether/.aether/workers/watcher-ant.md` - Watcher Ant prompt (809 lines, analyzed)
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/init.md` - Init command (277 lines, analyzed)
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/status.md` - Status command (406 lines, analyzed)
- `/Users/callumcowie/repos/Aether/.planning/PROJECT.md` - Project context and v2 requirements

### MEDIUM Confidence (Verified Patterns)

- Pull-based async design optimal for prompt-based agents (from test-event-async.sh, lines 6-194)
- Event filtering prevents spam (from event-bus.sh, lines 511-593)
- Subscription tracking enables polling semantics (from event-bus.sh, lines 325-423)

### LOW Confidence (Assumptions - Flagged for Validation)

- **Assumption:** Worker Ants will poll events frequently enough for timely reaction. **Validation needed:** Define "frequently enough" - is once per task execution sufficient?
- **Assumption:** Visual indicators improve UX significantly. **Validation needed:** User testing to confirm emoji and progress bars enhance scannability.
- **Assumption:** E2E manual test guide catches bugs automated tests miss. **Validation needed:** Compare manual test results with automated test suite after v2 complete.

### Gap Analysis

**Missing:** Specific guidance on when Workers should poll relative to task execution frequency. If Workers execute infrequently (e.g., once per hour), event reaction may be delayed. Need to define polling strategy:
- Option A: Poll after every action (file write, command execution)
- Option B: Poll at task start and end
- Option C: Poll on timer (e.g., every 5 minutes during long tasks)

**Recommendation:** Start with Option A (poll after every action) for v2, measure latency, optimize in v3 if needed.

**Missing:** User testing data on visual indicators. Emoji and progress bars seem helpful, but no user feedback. Recommend gathering feedback after v2 ships.

**Missing:** E2E test coverage metrics. What percentage of bugs should manual tests catch? Recommend tracking manual vs automated test findings post-release.
