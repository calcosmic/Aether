# Aether E2E Test Guide

**Version:** 1.0
**Last Updated:** 2026-02-02
**Phase:** 13 - E2E Testing

---

## Introduction

### Purpose of Manual E2E Testing

Automated tests are excellent for catching component bugs, but manual End-to-End (E2E) testing is essential for validating LLM-based autonomous systems like Aether. Here's why:

**LLMs are Non-Deterministic:** The same input can produce different but equally valid outputs. Automated tests that rely on exact string matching will fail due to minor wording differences, even when the core functionality works correctly.

**Reasoning Quality Matters:** Automated tests verify that code runs, but manual tests verify that the colony makes intelligent decisions. A Worker Ant might successfully spawn a specialist (automated test passes), but did it choose the right specialist for the task? (requires human judgment).

**Emergent Behavior:** The colony's behavior emerges from the interaction of multiple components. Manual tests can validate that the colony behaves intelligently in real scenarios, not just that individual components work in isolation.

**What Manual Tests Catch:**
- Reasoning errors (e.g., spawning the wrong specialist for a task)
- Context handling (e.g., Worker Ants missing critical information)
- Decision quality (e.g., poor prioritization or incorrect problem decomposition)
- Edge cases in autonomous behavior (e.g., circuit breaker triggers, spawn depth limits)

**What Automated Tests Catch:**
- Component bugs (e.g., broken JSON parsing, null pointer exceptions)
- Integration errors (e.g., event delivery failures, state corruption)
- Performance regressions (e.g., slow event polling, memory leaks)

### How to Use This Guide

This guide provides step-by-step instructions for testing all core Aether colony workflows:

1. **Read the Test Overview:** Understand what the test validates and why it matters
2. **Check Prerequisites:** Ensure the colony is in the correct state before testing
3. **Follow Test Steps:** Execute each step in sequence
4. **Verify Expected Outputs:** Compare your actual output with the expected output
5. **Run Verification Checks:** Execute the verification commands to validate state changes
6. **Document Results:** Note any deviations or unexpected behavior

**Testing Tips:**
- Use a clean colony state for each test (see Test Environment Setup)
- Run tests in order (some tests depend on previous tests)
- Take notes on any unexpected behavior
- If a test fails, document the actual output vs. expected output

### Verification IDs

Each verification check has a unique ID (e.g., VERIF-01, VERIF-02) that maps to requirements in the project documentation. These IDs ensure:

- **Traceability:** Every verification check can be traced back to a requirement
- **Coverage Tracking:** We can verify that all requirements have corresponding tests
- **Issue Tracking:** When issues are found, they can be linked to specific verification checks

**Verification ID Format:**
- VERIF-XX: Unique identifier for the verification check
- Maps to TEST-XX requirements (see Appendix A)
- Used in bug reports and test results

**Example:**
- VERIF-01: "Colony state file exists at `.aether/data/COLONY_STATE.json`"
- Maps to TEST-01 (Init workflow)
- If this check fails, we know exactly which requirement is not met

### What Makes E2E Testing Different

| Aspect | Automated Unit/Integration Tests | Manual E2E Tests |
|--------|--------------------------------|------------------|
| **Focus** | Component correctness | System behavior and decision quality |
| **Execution** | Automated by scripts | Manual by human tester |
| **Validation** | Exact output matching | Intelligent behavior validation |
| **Scope** | Individual components | Entire workflows |
| **Output** | Pass/Fail | Qualitative assessment + verification checks |
| **Purpose** | Catch bugs and regressions | Validate reasoning and emergent behavior |

**Best Practice:** Use both automated and manual tests. Automated tests provide fast feedback during development; manual tests validate that the colony behaves intelligently in real-world scenarios.

---

## Test Environment Setup

### Prerequisites

Before running the E2E tests, ensure you have the following:

**Required Tools:**
- **Git:** Version control for state backup/restore
  ```bash
  git --version  # Should be 2.30+
  ```

- **Bash:** Shell for running Aether commands and test scripts
  ```bash
  bash --version  # Should be 5.0+
  ```

- **jq:** JSON parser for verifying state files
  ```bash
  jq --version  # Should be 1.6+
  ```

**Required Aether Commands:**
- `/ant:init` - Initialize colony with goal
- `/ant:execute` - Execute a phase
- `/ant:status` - View colony status
- `/ant:build` - Build Worker Ants

**Required Data Files:**
The following files should exist in `.aether/data/`:
- `COLONY_STATE.json` - Main colony state
- `worker_ants.json` - Worker Ant castes
- `pheromones.json` - Active pheromones
- `memory.json` - Triple-layer memory
- `events.json` - Event bus

**Verify Prerequisites:**
```bash
# Check tools
git --version
bash --version
jq --version

# Check data files exist
ls -la .aether/data/

# Check commands available (via Claude)
# The commands should be in .claude/commands/ant/
ls .claude/commands/ant/
```

### Colony State Backup and Restore

To ensure test isolation, backup the colony state before each test and restore it after:

**Backup Colony State:**
```bash
# Create backup directory
mkdir -p .aether/backups/e2e_tests

# Backup all state files
cp .aether/data/COLONY_STATE.json .aether/backups/e2e_tests/COLONY_STATE.json.backup
cp .aether/data/worker_ants.json .aether/backups/e2e_tests/worker_ants.json.backup
cp .aether/data/pheromones.json .aether/backups/e2e_tests/pheromones.json.backup
cp .aether/data/memory.json .aether/backups/e2e_tests/memory.json.backup
cp .aether/data/events.json .aether/backups/e2e_tests/events.json.backup

echo "Backup complete"
```

**Restore Colony State:**
```bash
# Restore all state files
mv .aether/backups/e2e_tests/COLONY_STATE.json.backup .aether/data/COLONY_STATE.json
mv .aether/backups/e2e_tests/worker_ants.json.backup .aether/data/worker_ants.json
mv .aether/backups/e2e_tests/pheromones.json.backup .aether/data/pheromones.json
mv .aether/backups/e2e_tests/memory.json.backup .aether/data/memory.json
mv .aether/backups/e2e_tests/events.json.backup .aether/data/events.json

echo "Restore complete"
```

**Automated Backup/Restore Script:**
```bash
# Save this as .aether/utils/e2e-backup.sh
#!/bin/bash

BACKUP_DIR=".aether/backups/e2e_tests"

case "$1" in
  backup)
    mkdir -p "$BACKUP_DIR"
    cp .aether/data/*.json "$BACKUP_DIR/"
    echo "Backup complete: $BACKUP_DIR"
    ;;
  restore)
    mv "$BACKUP_DIR"/*.json .aether/data/
    echo "Restore complete"
    ;;
  *)
    echo "Usage: $0 {backup|restore}"
    exit 1
    ;;
esac
```

### Clean Slate Initialization

For tests that require a fresh colony, use the clean slate procedure:

**Option 1: Full Reset (Destructive)**
```bash
# WARNING: This deletes all colony state
rm -rf .aether/data/
rm -rf .aether/backups/

# Reinitialize colony state
source .aether/utils/initialize-state.sh

# Colony is now in IDLE state, ready for /ant:init
```

**Option 2: Soft Reset (Preserves Configuration)**
```bash
# Reset colony state to IDLE
jq '
  .colony_status.state = "IDLE" |
  .colony_status.current_phase = 0 |
  .queen_intention.goal = null |
  .queen_intention.initialized_at = null |
  .active_pheromones = [] |
  .spawn_tracking.depth = 0 |
  .spawn_tracking.spawn_history = [] |
  .resource_budgets.current_spawns = 0 |
  .working_memory.items = []
' .aether/data/COLONY_STATE.json > /tmp/colony_state.tmp

# Atomic write
source .aether/utils/atomic-write.sh
atomic_write_from_file .aether/data/COLONY_STATE.json /tmp/colony_state.tmp

# Colony is now in IDLE state, ready for /ant:init
```

**Verify Clean Slate:**
```bash
# Check colony status
jq -r '.colony_status.state' .aether/data/COLONY_STATE.json
# Expected: "IDLE"

# Check goal is null
jq -r '.queen_intention.goal' .aether/data/COLONY_STATE.json
# Expected: null or empty

# Check no active pheromones
jq '.active_pheromones | length' .aether/data/COLONY_STATE.json
# Expected: 0
```

### Test Isolation Best Practices

**1. Backup Before Each Test:**
Always create a fresh backup before running a test. This ensures you can restore to a known good state if the test fails or modifies state unexpectedly.

**2. Use Test-Specific Goals:**
When testing init workflow, use a test-specific goal that won't conflict with real work. For example:
- "Test goal for E2E testing"
- "E2E test: Verify colony initialization"
- NOT: "Build production API" (too vague for testing)

**3. Run Tests in Order:**
Tests are designed to run sequentially. Some tests depend on state created by previous tests. If you skip tests, you may need to manually set up the required state.

**4. Document Deviations:**
If a test fails or produces unexpected output, document:
- What you expected vs. what you got
- Any error messages or warnings
- Colony state at time of failure
- Steps you took to try to resolve

**5. Clean Up Between Tests:**
After completing a test, restore the colony to the expected state for the next test. Use the backup/restore procedure or reset the specific state fields that were modified.

**6. Verify Test Isolation:**
Before running a test, verify that the colony is in the expected state. After the test, verify that only the expected state fields changed.

**Example Test Isolation Checklist:**
```bash
# Before test
echo "=== Pre-test State ==="
jq -r '.colony_status.state' .aether/data/COLONY_STATE.json
jq -r '.queen_intention.goal' .aether/data/COLONY_STATE.json

# Run test...
# /ant:init "Test goal for E2E testing"

# After test
echo "=== Post-test State ==="
jq -r '.colony_status.state' .aether/data/COLONY_STATE.json
jq -r '.queen_intention.goal' .aether/data/COLONY_STATE.json

# Compare pre/post to verify expected changes
```

---

## Workflow 1: Init (Colony Initialization)

**Overview:** The Init workflow initializes the Aether Queen Ant Colony by setting the Queen's intention, creating all state files, mobilizing Worker Ants, and emitting the INIT pheromone. This is the foundation workflow that must complete successfully before any other colony operations can proceed.

**Why It Matters:** Without successful initialization, the colony cannot operate. The init workflow creates all necessary state files, sets up the colony structure, and prepares Worker Ants for autonomous emergence.

### Test 1.1: Happy Path - Successful Colony Initialization

**Overview:** Verify that the colony initializes correctly with a valid goal, creating all necessary state files and mobilizing Worker Ants with proper status indicators.

**Prerequisites:**
- Colony not initialized (COLONY_STATE.json has null goal or colony_status is IDLE)
- `/ant:init` command available
- Git repository initialized

**Test Steps:**

1. **Verify Pre-Test State:**
   ```bash
   # Check colony is in IDLE state or not initialized
   jq -r '.colony_status.state' .aether/data/COLONY_STATE.json
   # Expected: "IDLE" or file doesn't exist

   # Check goal is null
   jq -r '.queen_intention.goal' .aether/data/COLONY_STATE.json
   # Expected: null or empty
   ```

2. **Initialize Colony with Valid Goal:**
   ```
   /ant:init "Build a REST API for task management"
   ```

3. **Verify Initialization Output:**
   - Check for initialization progress steps (7 steps total)
   - Confirm colony mobilized message shown
   - Verify Worker Ants listed with emoji status indicators

**Expected Output:**

```
📊 Initialization Progress:
  [→] Step 1/7: Validate Preconditions...
  [✓] Step 1/7: Validate Preconditions
  [→] Step 2/7: Receive Intention...
  [✓] Step 2/7: Receive Intention
  [→] Step 3/7: Initialize Colony State...
  [✓] Step 3/7: Initialize Colony State
  [→] Step 4/7: Emit INIT Pheromone...
  [✓] Step 4/7: Emit INIT Pheromone
  [→] Step 5/7: Set Worker Ants to Ready...
  [✓] Step 5/7: Set Worker Ants to Ready
  [→] Step 6/7: Initialize Working Memory...
  [✓] Step 6/7: Initialize Working Memory
  [→] Step 7/7: Present Results...
  [✓] Step 7/7: Present Results

╔══════════════════════════════════════════════════════════════╗
║  🐜 Queen Ant Colony Initialized                             ║
╠══════════════════════════════════════════════════════════════╣
║  Session: session_1738392000_12345                          ║
║  Initialized: 2025-02-01T15:00:00Z                          ║
║                                                               ║
║  Queen's Intention:                                           ║
║  "Build a REST API for task management"                     ║
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

**Verification Checks:**

- **VERIF-01:** Colony state file exists at `.aether/data/COLONY_STATE.json`
  ```bash
  [ -f .aether/data/COLONY_STATE.json ] && echo "EXISTS" || echo "MISSING"
  ```

- **VERIF-02:** Queen's intention stored correctly
  ```bash
  jq -r '.queen_intention.goal' .aether/data/COLONY_STATE.json
  # Expected: "Build a REST API for task management"
  ```

- **VERIF-03:** Colony status set to "INIT"
  ```bash
  jq -r '.colony_status.state' .aether/data/COLONY_STATE.json
  # Expected: "INIT"
  ```

- **VERIF-04:** Current phase set to 1
  ```bash
  jq -r '.colony_status.current_phase' .aether/data/COLONY_STATE.json
  # Expected: "1"
  ```

- **VERIF-05:** All 6 Worker Ant castes have status "ready"
  ```bash
  jq '[.castes[] | select(.status != "ready")] | length' .aether/data/worker_ants.json
  # Expected: 0 (all ready)
  ```

- **VERIF-06:** INIT pheromone active with strength 1.0
  ```bash
  jq '.active_pheromones[] | select(.type == "INIT") | .strength' .aether/data/pheromones.json
  # Expected: "1.0"
  ```

- **VERIF-07:** Session ID generated and non-empty
  ```bash
  jq -r '.colony_metadata.session_id' .aether/data/COLONY_STATE.json
  # Expected: Non-null value matching "session_<timestamp>_<random>"
  ```

- **VERIF-08:** Working memory contains intention
  ```bash
  jq '.working_memory.items[] | select(.type == "intention") | .content' .aether/data/memory.json
  # Expected: "Build a REST API for task management"
  ```

**State Verification:**

**Before Test:**
```bash
jq -r '.colony_status.state' .aether/data/COLONY_STATE.json
# Expected: "IDLE" or file doesn't exist

jq '.resource_budgets.current_spawns' .aether/data/COLONY_STATE.json
# Expected: 0
```

**After Test:**
```bash
jq -r '.colony_status.state' .aether/data/COLONY_STATE.json
# Expected: "INIT"

jq '.resource_budgets.current_spawns' .aether/data/COLONY_STATE.json
# Expected: 0 (no spawns yet)

jq -r '.queen_intention.goal' .aether/data/COLONY_STATE.json
# Expected: "Build a REST API for task management"
```

### Test 1.2: Failure Case - Re-Initialization Attempt

**Overview:** Verify that the colony correctly rejects re-initialization attempts when already initialized, preserving the original goal and state.

**Prerequisites:**
- Colony already initialized with a goal
- Original goal is set in COLONY_STATE.json

**Test Steps:**

1. **Verify Colony is Initialized:**
   ```bash
   jq -r '.queen_intention.goal' .aether/data/COLONY_STATE.json
   # Expected: Non-null goal from Test 1.1

   jq -r '.colony_status.state' .aether/data/COLONY_STATE.json
   # Expected: "INIT" or other initialized state
   ```

2. **Attempt Re-Initialization:**
   ```
   /ant:init "This should fail - different goal"
   ```

3. **Verify Error Output:**
   - Error message displayed
   - Colony state unchanged

**Expected Output:**

```
⚠️  Colony already initialized with goal: Build a REST API for task management
Use /ant:status to view current state
```

**Verification Checks:**

- **VERIF-09:** Second init fails with error message
  ```bash
  # Manual check: Error message should be displayed
  ```

- **VERIF-10:** Original goal preserved
  ```bash
  jq -r '.queen_intention.goal' .aether/data/COLONY_STATE.json
  # Expected: "Build a REST API for task management" (original goal)
  ```

- **VERIF-11:** State unchanged from before re-init attempt
  ```bash
  jq -r '.colony_status.state' .aether/data/COLONY_STATE.json
  # Expected: "INIT" (unchanged)
  ```

**State Verification:**

**Before Test:**
```bash
jq -r '.queen_intention.goal' .aether/data/COLONY_STATE.json
# Expected: "Build a REST API for task management"
```

**After Test:**
```bash
jq -r '.queen_intention.goal' .aether/data/COLONY_STATE.json
# Expected: "Build a REST API for task management" (unchanged)
```

### Test 1.3: Edge Case - Empty or Invalid Goal Input

**Overview:** Verify that the colony handles empty or invalid goal inputs gracefully, with proper validation and error messages.

**Prerequisites:**
- Colony in IDLE state or not initialized
- Use soft reset to clear previous initialization

**Test Steps:**

1. **Reset Colony to IDLE (Soft Reset):**
   ```bash
   # Reset colony state to IDLE
   jq '
     .colony_status.state = "IDLE" |
     .colony_status.current_phase = 0 |
     .queen_intention.goal = null |
     .queen_intention.initialized_at = null |
     .active_pheromones = [] |
     .spawn_tracking.depth = 0 |
     .spawn_tracking.spawn_history = [] |
     .resource_budgets.current_spawns = 0 |
     .working_memory.items = []
   ' .aether/data/COLONY_STATE.json > /tmp/colony_state.tmp

   # Atomic write
   source .aether/utils/atomic-write.sh
   atomic_write_from_file .aether/data/COLONY_STATE.json /tmp/colony_state.tmp
   ```

2. **Attempt Init with Empty Goal:**
   ```
   /ant:init ""
   ```

3. **Verify Error Handling:**
   - Check for validation error
   - Colony not partially initialized

**Expected Output:**

```
⚠️  Invalid goal: Goal cannot be empty
Please provide a valid goal for colony initialization.
```

**Verification Checks:**

- **VERIF-12:** Input validation works
  ```bash
  # Manual check: Error message for empty goal
  ```

- **VERIF-13:** Colony not partially initialized
  ```bash
  jq -r '.colony_status.state' .aether/data/COLONY_STATE.json
  # Expected: "IDLE" (not changed to INIT)

   jq -r '.queen_intention.goal' .aether/data/COLONY_STATE.json
  # Expected: null (not set)
  ```

- **VERIF-14:** Clear error message provided
  ```bash
  # Manual check: Error message is descriptive
  ```

**Cleanup:**
```bash
# Reset colony for next test
source .aether/utils/e2e-backup.sh restore
```

---

## Workflow 2: Execute (Phase Execution)

**Overview:** The Execute workflow runs a phase with pure emergence, where Worker Ants self-organize, spawn subagents autonomously, and complete tasks. The workflow demonstrates autonomous spawning, pheromone-based coordination, and event-driven communication.

**Why It Matters:** This is the core workflow that demonstrates autonomous emergence. Worker Ants detect capability gaps, spawn specialists, coordinate via pheromones, and complete tasks without human orchestration.

### Test 2.1: Happy Path - Successful Phase Execution with Autonomous Spawning

**Overview:** Verify that a phase executes successfully, with Worker Ants mobilizing, autonomous spawning occurring via Task tool invocations, and phase completion.

**Prerequisites:**
- Colony initialized with goal
- Phase 1 exists and is in "ready" status

**Test Steps:**

1. **Verify Pre-Execution State:**
   ```bash
   # Check phase status
   jq '.phases.roadmap[0].status' .aether/data/COLONY_STATE.json
   # Expected: "ready"

   # Check spawn budget
   jq '.resource_budgets.current_spawns' .aether/data/COLONY_STATE.json
   # Expected: 0
   ```

2. **Execute Phase 1:**
   ```
   /ant:execute 1
   ```

3. **Observe Execution:**
   - Watch for step progress (6 steps)
   - Note Worker Ant mobilization messages
   - Look for autonomous spawning messages (Task tool invocations)

**Expected Output:**

```
📊 Execution Progress:
  [→] Step 1/6: Validate Input...
  [✓] Step 1/6: Validate Input
  [→] Step 2/6: Load Colony State...
  [✓] Step 2/6: Load Colony State
  [→] Step 3/6: Emit Init Pheromone for Phase...
  [✓] Step 3/6: Emit Init Pheromone for Phase
  [→] Step 4/6: Set Phase to In Progress...
  [✓] Step 4/6: Set Phase to In Progress
  [→] Step 5/6: Spawn Worker Ants for Execution...
  [✓] Step 5/6: Spawn Worker Ants for Execution
  [→] Step 6/6: Execute with Emergence...
  [✓] Step 6/6: Execute with Emergence

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🐜 Queen Ant Colony - Phase Execution

PHASE 1: Colony Foundation

Emitting INIT pheromone...
Colony mobilizing...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

TASK PROGRESS:
  ✅ Task 1.1: Create colony state JSON schema
  🔄 Task 1.2: Build pheromone signal layer (in progress)
  ⏳ Task 1.3: Initialize Worker Ant castes (pending)

WORKER ANTS ACTIVE:
  • Builder: implementing pheromone emission functions
  • Route-setter: planning Worker Ant caste initialization

SUBAGENTS SPAWNED: 2

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

PHASE 1 COMPLETE!

SUMMARY:
  ✓ 3/3 tasks completed
  ✓ 2 milestones reached
  ✓ 0 issues found and fixed

DURATION: 5m 23s
AGENTS SPAWNED: 2

KEY LEARNINGS:
  • JSON schema validation working correctly
  • Pheromone emission functions emit signals with proper metadata

ISSUES RESOLVED:
  • None

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📋 NEXT STEPS:
  1. /ant:review 1   - Review completed work
  2. /ant:feedback "<msg>"    - Provide feedback
  3. /ant:phase continue      - Continue to next phase

💡 COLONY RECOMMENDATION:
   Review the work before continuing.

🔄 CONTEXT: REFRESH RECOMMENDED
   Phase execution used significant context.
   Refresh Claude with /ant:review 1 before continuing.
```

**Verification Checks:**

- **VERIF-15:** Phase status changed to "in_progress" then "completed"
  ```bash
  jq -r '.phases.roadmap[0].status' .aether/data/COLONY_STATE.json
  # Expected: "completed" or "awaiting_review"
  ```

- **VERIF-16:** Autonomous spawning occurred
  ```bash
  # Manual check: Task tool was used to spawn specialists
  # Look for messages like "Spawning specialist..." or "Task: <specialist_type>"
  ```

- **VERIF-17:** Task tool used for spawning
  ```bash
  # Manual check: Task tool invocations present in execution output
  ```

- **VERIF-18:** Worker Ant activity indicators updated
  ```bash
  # Check Worker Ant status changed during execution
  jq '.castes[] | select(.status == "active") | .caste' .aether/data/worker_ants.json
  # Expected: At least one Worker Ant was active during execution
  ```

- **VERIF-19:** Pheromones emitted
  ```bash
  jq '.active_pheromones | length' .aether/data/pheromones.json
  # Expected: > 0 (new pheromones emitted during execution)
  ```

- **VERIF-20:** Events published
  ```bash
  jq '.events | length' .aether/data/events.json
  # Expected: > 0 (events published during execution)
  ```

- **VERIF-21:** Phase completed
  ```bash
  jq -r '.phases.roadmap[0].status' .aether/data/COLONY_STATE.json
  # Expected: "completed" or "awaiting_review"
  ```

- **VERIF-22:** Spawn tracking populated
  ```bash
  jq '.spawn_tracking.spawn_history | length' .aether/data/COLONY_STATE.json
  # Expected: > 0 (spawns recorded during execution)
  ```

**State Verification:**

**Before Test:**
```bash
jq '.phases.roadmap[0].status' .aether/data/COLONY_STATE.json
# Expected: "ready"

jq '.resource_budgets.current_spawns' .aether/data/COLONY_STATE.json
# Expected: 0
```

**After Test:**
```bash
jq '.phases.roadmap[0].status' .aether/data/COLONY_STATE.json
# Expected: "completed" or "awaiting_review"

jq '.spawn_tracking.spawn_history | length' .aether/data/COLONY_STATE.json
# Expected: > 0
```

### Test 2.2: Failure Case - Phase Execution with Blocked Tasks

**Overview:** Verify that the colony handles blocked tasks gracefully, logging errors and not marking the phase as complete when tasks cannot complete.

**Prerequisites:**
- Colony initialized
- Phase exists with tasks that can be blocked (e.g., missing dependencies)

**Test Steps:**

1. **Create Blocked Task Scenario:**
   - This test requires manual setup of a blocked scenario
   - Example: Task requires a file that doesn't exist

2. **Attempt Phase Execution:**
   ```
   /ant:execute <phase_id>
   ```

3. **Verify Error Handling:**
   - Check for error messages
   - Verify phase status not marked complete

**Expected Output:**

```
⚠️  Task execution error: Missing required dependency
Phase <phase_id> incomplete: 1/3 tasks completed

Errors logged to colony state.
Use /ant:status to view details.
```

**Verification Checks:**

- **VERIF-23:** Errors logged
  ```bash
  # Manual check: Error messages present in output
  ```

- **VERIF-24:** Phase not marked complete
  ```bash
  jq -r '.phases.roadmap[<phase_index>].status' .aether/data/COLONY_STATE.json
  # Expected: "in_progress" or "blocked" (not "completed")
  ```

- **VERIF-25:** Spawn budget not exhausted
  ```bash
  jq '.resource_budgets.current_spawns' .aether/data/COLONY_STATE.json
  # Expected: < max_spawns_per_phase (not exhausted)
  ```

- **VERIF-26:** Colony recovers
  ```bash
  # Manual check: Colony can still execute other tasks/phases
  ```

### Test 2.3: Edge Case - Re-Execute Completed Phase

**Overview:** Verify that the colony correctly rejects attempts to re-execute a completed phase.

**Prerequisites:**
- Phase already completed (from Test 2.1)

**Test Steps:**

1. **Attempt Re-Execution:**
   ```
   /ant:execute 1
   ```

2. **Verify Rejection:**
   - Error message displayed
   - Phase status unchanged

**Expected Output:**

```
✅ Phase 1 is already complete
Use /ant:review 1 to review completed work
```

**Verification Checks:**

- **VERIF-27:** Execution rejected
  ```bash
  # Manual check: Rejection message displayed
  ```

- **VERIF-28:** Phase status unchanged
  ```bash
  jq -r '.phases.roadmap[0].status' .aether/data/COLONY_STATE.json
  # Expected: "completed" or "awaiting_review" (unchanged)
  ```

- **VERIF-29:** No duplicate spawns
  ```bash
  # Manual check: No new spawn events in output
  ```

---

## Workflow 3: Spawning (Autonomous Worker Spawning)

**Overview:** The Spawning workflow demonstrates autonomous Worker Ant spawning, where Worker Ants detect capability gaps and spawn specialist subagents using the Task tool. This workflow tests Bayesian confidence updates, circuit breaker behavior, and spawn depth limits.

**Why It Matters:** Autonomous spawning is the core of the colony's emergence. Worker Ants should be able to recognize when they need help and spawn appropriate specialists without human intervention, while safeguards prevent infinite spawn loops.

### Test 3.1: Happy Path - Successful Specialist Spawn with Bayesian Confidence Update

**Overview:** Verify that a Worker Ant detects a capability gap, spawns a specialist via Task tool, and Bayesian confidence is updated correctly.

**Prerequisites:**
- Colony initialized and phase executing
- Task that requires specialist capability (e.g., database task)

**Test Steps:**

1. **Create Scenario Requiring Specialist:**
   - Execute a task that requires database specialization
   - Example: "Implement JWT authentication with database token storage"

2. **Observe Autonomous Spawning:**
   - Watch for capability gap detection
   - Note Task tool invocation for specialist spawn
   - Verify spawn tracking record

**Expected Output:**

```
🔍 Capability Gap Detected:
  Required capabilities: jwt, authentication, database
  Available capabilities: basic implementation
  Gap: database_specialist needed

🔄 Spawning Specialist:
  Task: database_specialist

  You are a database_specialist spawned by builder.

  TASK: Implement JWT authentication with database token storage

  INHERITED CONTEXT:
  - Goal: Build a REST API for task management
  - Active Pheromones: INIT (strength 1.0)
  - Parent's Context: Working memory contains task details
  - Constraints: Use PostgreSQL for token storage

  CAPABILITY GAPS DETECTED: database, jwt token storage
  REASON: Parent Worker Ant (builder) lacks database expertise

  Execute autonomously. Report results when complete.

✓ Spawn recorded: spawn_db_1234567890
✓ Bayesian confidence: database_specialist.database = 0.6 (+0.1)
```

**Verification Checks:**

- **VERIF-30:** Capability gap detected
  ```bash
  # Manual check: Capability gap analysis displayed
  ```

- **VERIF-31:** Specialist spawned (not duplicate)
  ```bash
  # Manual check: Specialist spawned via Task tool
  # Check that it's not a duplicate spawn (same specialist for same task)
  ```

- **VERIF-32:** Spawn recorded
  ```bash
  jq '.spawn_tracking.spawn_history[-1]' .aether/data/COLONY_STATE.json
  # Expected: Last spawn entry contains specialist details
  ```

- **VERIF-33:** Bayesian confidence updated
  ```bash
  jq -r '.meta_learning.specialist_confidence."database_specialist"."database"' .aether/data/COLONY_STATE.json
  # Expected: Value between 0.0-1.0 (typically 0.6 after first success)
  ```

- **VERIF-34:** Circuit breaker not triggered
  ```bash
  jq '.resource_budgets.circuit_breaker_trips' .aether/data/COLONY_STATE.json
  # Expected: 0 or < 3 (not triggered)
  ```

- **VERIF-35:** Spawn depth within limit
  ```bash
  jq '.spawn_tracking.depth' .aether/data/COLONY_STATE.json
  # Expected: < max_depth (typically 1 or 2)
  ```

- **VERIF-36:** Spawn budget available
  ```bash
  jq '.resource_budgets.current_spawns' .aether/data/COLONY_STATE.json
  # Expected: < max_spawns_per_phase (typically 10)
  ```

- **VERIF-37:** Meta-learning data populated
  ```bash
  jq '.meta_learning.spawn_outcomes | length' .aether/data/COLONY_STATE.json
  # Expected: > 0 (spawn outcomes recorded)
  ```

**State Verification:**

**Before Test:**
```bash
jq -r '.meta_learning.specialist_confidence."database_specialist"."database"' .aether/data/COLONY_STATE.json
# Expected: 0.5 (default) or null (not yet set)
```

**After Test:**
```bash
jq -r '.meta_learning.specialist_confidence."database_specialist"."database"' .aether/data/COLONY_STATE.json
# Expected: 0.6 (increased after successful spawn)
```

### Test 3.2: Failure Case - Circuit Breaker Activation After Repeated Failures

**Overview:** Verify that the circuit breaker activates after 3 failed spawns, blocking further spawns of the same specialist type.

**Prerequisites:**
- Colony initialized
- Ability to simulate spawn failures (manual test setup)

**Test Steps:**

1. **Simulate 3 Spawn Failures:**
   ```bash
   # Record spawn failures for testing
   source .aether/utils/spawn-tracker.sh
   record_spawn_failure "database_specialist" "test_spawn_1" "Test failure 1"
   record_spawn_failure "database_specialist" "test_spawn_2" "Test failure 2"
   record_spawn_failure "database_specialist" "test_spawn_3" "Test failure 3"
   ```

2. **Attempt Another Spawn:**
   - Try to spawn the same specialist again

3. **Verify Circuit Breaker Activation:**
   - Spawn rejected
   - Circuit breaker status shows "open"

**Expected Output:**

```
⚠️  Circuit Breaker Activated:
  Specialist: database_specialist
  Failure count: 3
  Cooldown until: 2025-02-01T16:00:00Z

🚫 Spawn blocked: Circuit breaker is open
  Reason: 3 consecutive failures for database_specialist
  Wait for cooldown before retrying
```

**Verification Checks:**

- **VERIF-38:** Circuit breaker status open
  ```bash
  jq -r '.resource_budgets.circuit_breaker_status' .aether/data/COLONY_STATE.json
  # Expected: "open" or "tripped"
  ```

- **VERIF-39:** Spawn rejected
  ```bash
  # Manual check: Spawn rejected message displayed
  ```

- **VERIF-40:** Failure count tracked
  ```bash
  jq '.resource_budgets.circuit_breaker_trips' .aether/data/COLONY_STATE.json
  # Expected: 3 or more
  ```

- **VERIF-41:** Circuit breaker timestamp set
  ```bash
  jq -r '.resource_budgets.circuit_breaker_cooldown_until // "null"' .aether/data/COLONY_STATE.json
  # Expected: Non-null timestamp
  ```

- **VERIF-42:** Spawn blocked flag set
  ```bash
  # Manual check: Spawn blocked due to circuit breaker
  ```

**Cleanup:**
```bash
# Reset circuit breaker for next test
source .aether/utils/circuit-breaker.sh
reset_circuit_breaker
```

### Test 3.3: Edge Case - Max Spawn Depth Reached

**Overview:** Verify that spawns are rejected when the maximum spawn depth is reached, preventing infinite spawn loops.

**Prerequisites:**
- Colony initialized
- Spawn depth set to maximum (3)

**Test Steps:**

1. **Set Spawn Depth to Maximum:**
   ```bash
   # Set depth to max (3)
   jq '.spawn_tracking.depth = 3' .aether/data/COLONY_STATE.json > /tmp/colony_state.tmp
   source .aether/utils/atomic-write.sh
   atomic_write_from_file .aether/data/COLONY_STATE.json /tmp/colony_state.tmp
   ```

2. **Attempt Spawn at Max Depth:**
   - Try to spawn another specialist

3. **Verify Depth Limit Enforcement:**
   - Spawn rejected
   - Error message about depth limit

**Expected Output:**

```
⚠️  Spawn Depth Limit Reached:
  Current depth: 3
  Max depth: 3
  Cannot spawn: Maximum spawn depth exceeded

🚫 Spawn blocked: Depth limit enforced
  Reason: Already at maximum spawn depth (3 levels)
  Handle task with current Worker Ant
```

**Verification Checks:**

- **VERIF-43:** Spawn depth at maximum
  ```bash
  jq '.spawn_tracking.depth' .aether/data/COLONY_STATE.json
  # Expected: 3 (max depth)
  ```

- **VERIF-44:** Spawn rejected
  ```bash
  # Manual check: Spawn rejected due to depth limit
  ```

- **VERIF-45:** Depth limit error logged
  ```bash
  # Manual check: Error message mentions depth limit
  ```

- **VERIF-46:** No infinite spawn loop
  ```bash
  # Manual check: Spawn count stabilized, no continuous spawning
  ```

**Cleanup:**
```bash
# Reset spawn depth for next test
jq '.spawn_tracking.depth = 0' .aether/data/COLONY_STATE.json > /tmp/colony_state.tmp
source .aether/utils/atomic-write.sh
atomic_write_from_file .aether/data/COLONY_STATE.json /tmp/colony_state.tmp
```

---

## Workflow 4: Memory (Triple-Layer Memory with DAST Compression)

**Overview:** The Memory workflow tests the triple-layer memory system (working → short-term → long-term) with DAST (Declarative Associative Semantic Timestamp) compression. This workflow validates memory capacity limits, associative link creation, and memory overflow handling.

**Why It Matters:** The memory system is critical for the colony's ability to retain and retrieve information. DAST compression prevents memory overflow while preserving important context through associative links.

### Test 4.1: Happy Path - DAST Compression Triggered When Working Memory Full

**Overview:** Verify that DAST compression is triggered when working memory exceeds capacity (10 items), moving items to short-term memory and creating associative links.

**Prerequisites:**
- Colony initialized
- Working memory can be filled via tasks or manual injection

**Test Steps:**

1. **Fill Working Memory Beyond Capacity:**
   ```bash
   # Add 15 items to working memory (exceeds 10-item limit)
   for i in {1..15}; do
     memory_id="mem_$(date +%s)_$i"
     jq --arg id "$memory_id" --arg num "$i" '
       .working_memory.items += [{
         "id": $id,
         "type": "test",
         "content": "Test memory item \($num)",
         "metadata": {
           "timestamp": "'$(date -u +"%Y-%m-%dT%H:%M:%SZ")'",
           "relevance_score": 1.0,
           "access_count": 1,
           "last_accessed": "'$(date -u +"%Y-%m-%dT%H:%M:%SZ")'",
           "source": "test",
           "caste": null
         },
         "associative_links": []
       }]
     ' .aether/data/memory.json > /tmp/memory.tmp
     mv /tmp/memory.tmp .aether/data/memory.json
   done
   ```

2. **Trigger Compression:**
   - The memory system should automatically trigger compression
   - Or manually trigger via memory utility

3. **Verify Compression:**
   - Working memory count reduced to ≤ 10
   - Short-term memory populated
   - Associative links created

**Expected Output:**

```
🧠 DAST Compression Triggered:
  Working memory: 15 items (exceeds capacity of 10)
  Compressing to short-term memory...

✓ Compression complete:
  - Moved 5 items to short-term memory
  - Created 3 associative links
  - Working memory: 10 items
  - Short-term memory: 5 items

Associative Links Created:
  • mem_1 ↔ mem_2 (same task_id)
  • mem_3 ↔ mem_4 (same phase_id)
  • mem_5 ↔ mem_6 (same topic)

Memory Summary:
  Working: 10/10 items (100% capacity)
  Short-term: 5 items
  Long-term: 0 items
```

**Verification Checks:**

- **VERIF-47:** DAST compression triggered
  ```bash
  # Manual check: Compression message displayed
  ```

- **VERIF-48:** Working memory reduced
  ```bash
  jq '.working_memory.items | length' .aether/data/memory.json
  # Expected: ≤ 10 (at or below capacity)
  ```

- **VERIF-49:** Short-term memory populated
  ```bash
  jq '.short_term_memory.items | length' .aether/data/memory.json
  # Expected: > 0 (items moved from working memory)
  ```

- **VERIF-50:** Associative links created
  ```bash
  jq '.associative_links | length' .aether/data/memory.json
  # Expected: > 0 (links created during compression)
  ```

- **VERIF-51:** Memory metadata updated
  ```bash
  jq -r '.memory_metadata.last_compression // "null"' .aether/data/memory.json
  # Expected: Non-null timestamp
  ```

- **VERIF-52:** Memory files updated
  ```bash
  # Manual check: memory.json file modified (check timestamp)
  ```

- **VERIF-53:** No memory loss
  ```bash
  # Total item count should be preserved
  working=$(jq '.working_memory.items | length' .aether/data/memory.json)
  short_term=$(jq '.short_term_memory.items | length' .aether/data/memory.json)
  echo "Total: $((working + short_term))"
  # Expected: 15 (all items preserved)
  ```

### Test 4.2: Failure Case - Memory Overflow Handling

**Overview:** Verify that the colony handles memory overflow gracefully when both working and short-term memory are full, archiving oldest items to long-term memory.

**Prerequisites:**
- Working memory at capacity (10 items)
- Short-term memory at capacity (20 items)

**Test Steps:**

1. **Fill Both Memory Layers:**
   ```bash
   # Fill working memory (10 items)
   # Fill short-term memory (20 items)
   # Then add more items to trigger overflow
   ```

2. **Trigger Overflow:**
   - Add items beyond both capacities

3. **Verify Overflow Handling:**
   - Oldest items archived to long-term memory
   - Error or graceful degradation message

**Expected Output:**

```
⚠️  Memory Overflow Detected:
  Working memory: 10/10 items (full)
  Short-term memory: 20/20 items (full)
  Archiving oldest items to long-term memory...

✓ Overflow handled:
  - Archived 5 oldest items to long-term memory
  - No memory loss
  - Colony remains stable

Memory Summary:
  Working: 10/10 items
  Short-term: 20/20 items
  Long-term: 5 items (archived)
```

**Verification Checks:**

- **VERIF-54:** Overflow detected
  ```bash
  # Manual check: Overflow detection message
  ```

- **VERIF-55:** Error logged
  ```bash
  # Manual check: Overflow logged in colony state
  ```

- **VERIF-56:** Colony stable on overflow
  ```bash
  # Manual check: Colony continues operating normally
  ```

- **VERIF-57:** Oldest items archived
  ```bash
  jq '.long_term_memory.items | length' .aether/data/memory.json
  # Expected: > 0 (oldest items archived)
  ```

### Test 4.3: Edge Case - Associative Link Creation During Compression

**Overview:** Verify that associative links are created between related memory items during DAST compression, improving retrieval performance.

**Prerequisites:**
- Working memory with related items (same task, phase, or topic)

**Test Steps:**

1. **Create Related Memory Items:**
   ```bash
   # Add items with same task_id, phase_id, or topic
   # These should be linked during compression
   ```

2. **Trigger Compression:**
   - Force compression with related items

3. **Verify Associative Links:**
   - Links created between related items
   - Association strength calculated
   - Retrieval improved

**Expected Output:**

```
🔗 Associative Link Creation:
  Analyzing relationships between memory items...

✓ Links created:
  • mem_1 ↔ mem_2 (task_id: task_123, strength: 0.9)
  • mem_3 ↔ mem_4 (phase_id: phase_1, strength: 0.8)
  • mem_5 ↔ mem_6 (topic: auth, strength: 0.7)

Retrieval Test:
  Query: "task_123"
  Results: mem_1, mem_2 (both retrieved via associative link)
  Performance: 2ms (vs 15ms without links)
```

**Verification Checks:**

- **VERIF-58:** Associative links created
  ```bash
  jq '.associative_links | length' .aether/data/memory.json
  # Expected: > 0 (links created)
  ```

- **VERIF-59:** Related items linked
  ```bash
  # Check that items with same task_id/phase_id are linked
  jq '.associative_links[] | select(.reason | contains("task_id"))' .aether/data/memory.json
  # Expected: Links present for same task_id
  ```

- **VERIF-60:** Retrieval improved
  ```bash
  # Manual check: Query returns related items via associative links
  ```

---

## Workflow 5: Voting (Multi-Perspective Verification with Weighted Voting)

**Overview:** The Voting workflow tests the multi-perspective verification system where 4 Watchers (security, performance, quality, test_coverage) vote on task/phase completion with weighted voting and Critical veto power.

**Why It Matters:** The voting system ensures comprehensive validation from multiple perspectives. Weighted voting allows domain experts to have more influence, while Critical veto power prevents dangerous code from being approved.

### Test 5.1: Happy Path - Supermajority Approval with Weighted Votes

**Overview:** Verify that 4 Watchers vote correctly, supermajority is calculated (≥ 67% = APPROVED), weights are applied, and decision outcome is logged.

**Prerequisites:**
- Colony initialized
- Phase or task completed (triggers verification)
- All 4 Watchers available

**Test Steps:**

1. **Trigger Verification:**
   - Complete a task or phase
   - Watchers should automatically vote

2. **Observe Voting Process:**
   - All 4 Watchers cast votes
   - Supermajority calculated
   - Result displayed

**Expected Output:**

```
🗳️  Multi-Perspective Verification:
  Collecting votes from 4 Watchers...

Watchers Voting:
  ✓ Security Watcher: APPROVE (weight: 1.0)
  ✓ Performance Watcher: APPROVE (weight: 1.0)
  ✓ Quality Watcher: APPROVE (weight: 1.0)
  ✓ Test Coverage Watcher: APPROVE (weight: 1.0)

Supermajority Calculation:
  Approvals: 4/4 (100%)
  Required: 67%
  Result: ✓ APPROVED

Decision Outcome:
  Status: APPROVED
  Supermajority: 100% (unanimous)
  Issues Found: 0
  Critical Issues: 0

Vote Metadata:
  Verification ID: ver_1234567890
  Timestamp: 2025-02-01T15:30:00Z
  Watchers Participated: 4/4
```

**Verification Checks:**

- **VERIF-61:** All 4 Watchers vote
  ```bash
  # Manual check: All 4 Watchers participated
  ```

- **VERIF-62:** Votes recorded
  ```bash
  jq '.verification.votes | length' .aether/data/COLONY_STATE.json
  # Expected: > 0 (votes recorded)
  ```

- **VERIF-63:** Supermajority calculated
  ```bash
  jq -r '.verification.last_supermajority' .aether/data/COLONY_STATE.json
  # Expected: "APPROVED" or "REJECTED"
  ```

- **VERIF-64:** Weights applied
  ```bash
  # Manual check: Weighted votes used in calculation
  # Check watcher weights in state
  jq '.watcher_weights' .aether/data/watcher_weights.json
  # Expected: Weights present and applied
  ```

- **VERIF-65:** Decision outcome logged
  ```bash
  jq -r '.verification.last_decision' .aether/data/COLONY_STATE.json
  # Expected: "APPROVED" or "REJECTED"
  ```

- **VERIF-66:** Issues deduplicated
  ```bash
  # Manual check: Duplicate issues merged
  ```

- **VERIF-67:** Vote metadata recorded
  ```bash
  jq '.verification.votes[-1]' .aether/data/COLONY_STATE.json
  # Expected: Vote entry with timestamp, watcher, decision
  ```

- **VERIF-68:** Supermajority status updated
  ```bash
  jq -r '.verification.last_supermajority' .aether/data/COLONY_STATE.json
  # Expected: "APPROVED" (100% ≥ 67%)
  ```

### Test 5.2: Failure Case - Critical Veto Blocks Approval Despite Majority

**Overview:** Verify that a Critical issue from any Watcher (especially Security) blocks approval even with 75% approval rate.

**Prerequisites:**
- Colony initialized
- 3 Watchers approve, 1 Critical Watcher rejects

**Test Steps:**

1. **Create Critical Veto Scenario:**
   - 3 Watchers APPROVE
   - Security Watcher VETOS with Critical issue

2. **Trigger Verification:**
   - Calculate supermajority
   - Apply Critical veto

3. **Verify Veto Power:**
   - Result REJECTED despite 75% approval
   - Veto reason logged

**Expected Output:**

```
🗳️  Multi-Perspective Verification:
  Collecting votes from 4 Watchers...

Watchers Voting:
  ✓ Security Watcher: REJECT (weight: 1.0) [CRITICAL VETO]
    Issue: SQL injection vulnerability in auth module
    Severity: Critical
  ✓ Performance Watcher: APPROVE (weight: 1.0)
  ✓ Quality Watcher: APPROVE (weight: 1.0)
  ✓ Test Coverage Watcher: APPROVE (weight: 1.0)

Supermajority Calculation:
  Approvals: 3/4 (75%)
  Required: 67%
  Raw Result: APPROVED

⚠️  CRITICAL VETO EXERCISED:
  Watcher: Security
  Reason: SQL injection vulnerability
  Severity: Critical

Final Decision: REJECTED
  Reason: Critical issue veto overrides majority
  Required Action: Fix Critical issue before re-verification
```

**Verification Checks:**

- **VERIF-69:** Critical veto exercised
  ```bash
  # Manual check: Critical veto displayed
  ```

- **VERIF-70:** Veto blocks approval
  ```bash
  jq -r '.verification.last_decision' .aether/data/COLONY_STATE.json
  # Expected: "REJECTED" (despite 75% approval)
  ```

- **VERIF-71:** Veto reason logged
  ```bash
  jq -r '.verification.last_veto_reason // "null"' .aether/data/COLONY_STATE.json
  # Expected: Non-null veto reason
  ```

- **VERIF-72:** Veto flag set
  ```bash
  jq -r '.verification.critical_veto_exercised // false' .aether/data/COLONY_STATE.json
  # Expected: true
  ```

- **VERIF-73:** Critical watcher power verified
  ```bash
  # Manual check: Security Watcher's Critical issue blocked approval
  ```

### Test 5.3: Edge Case - Weight Calculation Edge Cases

**Overview:** Verify that the voting system handles edge cases like 0 watchers or all abstentions gracefully.

**Prerequisites:**
- Colony initialized
- Edge case scenario (0 watchers or all abstain)

**Test Steps:**

1. **Create Edge Case:**
   - Either 0 watchers available
   - Or all watchers abstain

2. **Trigger Verification:**
   - Calculate supermajority with edge case

3. **Verify Graceful Handling:**
   - No crash
   - Default decision or manual escalation

**Expected Output:**

```
⚠️  Voting Edge Case Detected:
  Watchers Participating: 0/4
  Reason: No watchers available

Default Decision: PENDING_MANUAL_REVIEW
  Reason: Cannot verify without watchers
  Action Required: Manual review needed
```

**Verification Checks:**

- **VERIF-74:** Edge case handled
  ```bash
  # Manual check: Edge case detected and handled
  ```

- **VERIF-75:** No crash on edge cases
  ```bash
  # Manual check: Colony remains stable
  ```

- **VERIF-76:** Vote count = 0 or all abstain
  ```bash
  # Manual check: Vote count reflects edge case
  ```

- **VERIF-77:** Default decision applied
  ```bash
  jq -r '.verification.last_decision' .aether/data/COLONY_STATE.json
  # Expected: "PENDING_MANUAL_REVIEW" or similar default
  ```

---

## Workflow 6: Event (Event Polling and Delivery)

**Overview:** The Event workflow tests the event bus system where Worker Ants poll for events, receive caste-specific events, mark events as delivered, and prevent reprocessing. This workflow validates pub/sub communication across the colony.

**Why It Matters:** Event-driven communication is critical for colony coordination. Worker Ants need to receive relevant events based on their caste subscriptions, and delivery tracking prevents duplicate processing.

### Test 6.1: Happy Path - Event Polling Retrieves Events, Delivery Prevents Reprocessing

**Overview:** Verify that get_events_for_subscriber() returns matching events, mark_events_delivered() adds subscriber to delivered_to array, and second poll returns empty (already delivered).

**Prerequisites:**
- Colony initialized
- Event bus initialized
- Worker Ants subscribed to topics

**Test Steps:**

1. **Publish Test Events:**
   ```bash
   # Source event bus
   source .aether/utils/event-bus.sh

   # Publish test events
   publish_event "phase_complete" "test_phase" '{"phase": "1"}' "test_publisher" "colonizer"
   publish_event "task_started" "test_task" '{"task": "Build auth"}' "test_publisher" "builder"
   publish_event "error" "test_error" '{"message": "Test error"}' "test_publisher" "builder"
   ```

2. **Poll for Events:**
   ```bash
   # Subscribe to topics
   subscribe_to_events "test_colonizer_1" "colonizer" "phase_complete" '{}'
   subscribe_to_events "test_colonizer_1" "colonizer" "error" '{}'

   # Poll for events
   events=$(get_events_for_subscriber "test_colonizer_1" "colonizer")
   echo "$events" | jq '.'
   ```

3. **Mark as Delivered:**
   ```bash
   mark_events_delivered "test_colonizer_1" "colonizer" "$events"
   ```

4. **Poll Again (Should Return Empty):**
   ```bash
   events_after=$(get_events_for_subscriber "test_colonizer_1" "colonizer")
   echo "$events_after" | jq '.'
   # Expected: [] (empty array)
   ```

**Expected Output:**

```
📡 Event Polling:
  Subscriber: test_colonizer_1 (colonizer)
  Topics: phase_complete, error

Events Retrieved:
  [
    {
      "id": "event_1234567890",
      "topic": "phase_complete",
      "subscriber_id": "test_colonizer_1",
      "timestamp": "2025-02-01T15:00:00Z",
      "data": {"phase": "1"},
      "delivered_to": []
    },
    {
      "id": "event_1234567891",
      "topic": "error",
      "subscriber_id": "test_colonizer_1",
      "timestamp": "2025-02-01T15:00:01Z",
      "data": {"message": "Test error"},
      "delivered_to": []
    }
  ]

✓ Events marked as delivered:
  - event_1234567890: delivered_to = ["test_colonizer_1"]
  - event_1234567891: delivered_to = ["test_colonizer_1"]

📡 Second Poll (should return empty):
  Events Retrieved: []
  Reason: All events already delivered to this subscriber
```

**Verification Checks:**

- **VERIF-78:** get_events_for_subscriber() returns matching events
  ```bash
  # Manual check: Events returned match subscription criteria
  ```

- **VERIF-79:** Topic filtering works
  ```bash
  # Check that only subscribed topics returned
  echo "$events" | jq -r '.[].topic' | sort -u
  # Expected: phase_complete, error (not task_started)
  ```

- **VERIF-80:** Event schema valid
  ```bash
  echo "$events" | jq '.[0]' | jq 'has("id"), has("topic"), has("subscriber_id"), has("timestamp"), has("data")'
  # Expected: true, true, true, true, true
  ```

- **VERIF-81:** Delivery marks processed
  ```bash
  # After marking delivered, check delivered_to array
  jq '.events[] | select(.id == "event_1234567890") | .delivered_to' .aether/data/events.json
  # Expected: Contains ["test_colonizer_1"]
  ```

- **VERIF-82:** Reprocessing prevented
  ```bash
  # Second poll should return empty
  echo "$events_after" | jq '.'
  # Expected: []
  ```

- **VERIF-83:** Caste subscriptions work
  ```bash
  # Manual check: Different castes receive different events
  ```

- **VERIF-84:** Event logging records polls
  ```bash
  # Check event bus logs for polling activity
  ```

- **VERIF-85:** Delivery tracking updated
  ```bash
  # Check delivery_tracking in events.json
  jq '.delivery_tracking' .aether/data/events.json
  # Expected: Updated with subscriber deliveries
  ```

### Test 6.2: Failure Case - Event Delivery Failure (Corrupted Event Data)

**Overview:** Verify that the colony handles invalid event data gracefully, logging errors and continuing operation.

**Prerequisites:**
- Event bus initialized
- Ability to publish invalid event

**Test Steps:**

1. **Publish Invalid Event:**
   ```bash
   # Create event with invalid JSON
   echo '{"invalid": "json", "missing": "fields"' > /tmp/invalid_event.json
   ```

2. **Attempt to Process:**
   - Worker Ant attempts to process invalid event

3. **Verify Error Handling:**
   - Error logged
   - Colony continues

**Expected Output:**

```
⚠️  Invalid Event Detected:
  Event ID: event_invalid_123
  Error: Invalid JSON schema
  Reason: Missing required fields (topic, subscriber_id)

✓ Error handled gracefully:
  - Invalid event skipped
  - Colony continues operation
  - Error logged to event tracking

Event Status: FAILED
  Reason: Invalid event schema
  Action: Skipped, not processed
```

**Verification Checks:**

- **VERIF-86:** Invalid event detected
  ```bash
  # Manual check: Invalid event detection message
  ```

- **VERIF-87:** Error logged
  ```bash
  # Check error logs
  jq '.event_tracking.errors[] | select(.event_id == "event_invalid_123")' .aether/data/events.json
  # Expected: Error entry present
  ```

- **VERIF-88:** Colony continues on error
  ```bash
  # Manual check: Colony still operational
  ```

- **VERIF-89:** Failed events marked
  ```bash
  # Check event status
  jq '.events[] | select(.id == "event_invalid_123") | .status' .aether/data/events.json
  # Expected: "failed" or similar
  ```

### Test 6.3: Edge Case - Caste-Specific Filtering (Different Castes Receive Different Events)

**Overview:** Verify that different castes receive different events based on their caste-specific subscriptions, with proper topic filtering and error topic received by all.

**Prerequisites:**
- Event bus initialized
- Multiple castes subscribed to different topics

**Test Steps:**

1. **Subscribe Different Castes:**
   ```bash
   # Colonizer subscribes to phase_complete, spawn_request, error
   subscribe_to_events "test_colonizer_2" "colonizer" "phase_complete" '{}'
   subscribe_to_events "test_colonizer_2" "colonizer" "spawn_request" '{}'
   subscribe_to_events "test_colonizer_2" "colonizer" "error" '{}'

   # Watcher subscribes to task_completed, task_failed, phase_complete, error
   subscribe_to_events "test_watcher_1" "watcher" "task_completed" '{}'
   subscribe_to_events "test_watcher_1" "watcher" "task_failed" '{}'
   subscribe_to_events "test_watcher_1" "watcher" "phase_complete" '{}'
   subscribe_to_events "test_watcher_1" "watcher" "error" '{}'
   ```

2. **Publish Diverse Events:**
   ```bash
   publish_event "phase_complete" "test_phase" '{"phase": "1"}' "queen" "colonizer"
   publish_event "spawn_request" "test_spawn" '{"specialist": "scout"}' "colonizer" "colonizer"
   publish_event "task_started" "test_task" '{"task": "Build auth"}' "queen" "builder"
   publish_event "error" "test_error" '{"message": "Test error"}' "builder" "builder"
   ```

3. **Poll for Each Caste:**
   ```bash
   # Colonizer polls
   colonizer_events=$(get_events_for_subscriber "test_colonizer_2" "colonizer")
   echo "Colonizer events:"
   echo "$colonizer_events" | jq -r '.[].topic'

   # Watcher polls
   watcher_events=$(get_events_for_subscriber "test_watcher_1" "watcher")
   echo "Watcher events:"
   echo "$watcher_events" | jq -r '.[].topic'
   ```

4. **Verify Caste-Specific Filtering:**
   - Colonizer receives phase_complete, spawn_request, error
   - Watcher receives phase_complete, error (not spawn_request)

**Expected Output:**

```
📊 Caste-Specific Event Filtering Test:

Colonizer Events:
  Topics: phase_complete, spawn_request, error
  Count: 3

Watcher Events:
  Topics: phase_complete, error
  Count: 2
  (spawn_request not included - not subscribed)

✓ Caste-Specific Filtering Working:
  - Colonizer received 3/4 published events
  - Watcher received 2/4 published events
  - Error topic received by all castes
  - Topic crossover prevented
```

**Verification Checks:**

- **VERIF-90:** Colonizer subscriptions verified
  ```bash
  # Check colonizer subscriptions
  jq '.subscribers[] | select(.subscriber_id == "test_colonizer_2") | .topics' .aether/data/events.json
  # Expected: ["phase_complete", "spawn_request", "error"]
  ```

- **VERIF-91:** Watcher subscriptions verified
  ```bash
  # Check watcher subscriptions
  jq '.subscribers[] | select(.subscriber_id == "test_watcher_1") | .topics' .aether/data/events.json
  # Expected: ["task_completed", "task_failed", "phase_complete", "error"]
  ```

- **VERIF-92:** Topic filtering prevents crossover
  ```bash
  # Verify colonizer didn't receive task_started
  echo "$colonizer_events" | jq -r '.[] | select(.topic == "task_started")'
  # Expected: empty (no task_started events)
  ```

- **VERIF-93:** Error topic received by all
  ```bash
  # Both castes should have error event
  echo "$colonizer_events" | jq -r '.[] | select(.topic == "error")'
  echo "$watcher_events" | jq -r '.[] | select(.topic == "error")'
  # Expected: Both return error event
  ```

- **VERIF-94:** Subscription criteria applied
  ```bash
  # Manual check: Subscription filters work correctly
  ```

---

## Appendix A: Verification ID Mapping

This appendix maps all verification IDs (VERIF-01 through VERIF-94) to their corresponding requirements (TEST-01 through TEST-06), ensuring full requirement traceability.

| Verification ID | Requirement | Description |
|----------------|-------------|-------------|
| VERIF-01 | TEST-01 | Init workflow - state file creation |
| VERIF-02 | TEST-01 | Init workflow - intention storage |
| VERIF-03 | TEST-01 | Init workflow - colony status transition |
| VERIF-04 | TEST-01 | Init workflow - current phase set |
| VERIF-05 | TEST-01 | Init workflow - Worker Ants mobilized |
| VERIF-06 | TEST-01 | Init workflow - INIT pheromone emitted |
| VERIF-07 | TEST-01 | Init workflow - session ID generated |
| VERIF-08 | TEST-01 | Init workflow - working memory initialized |
| VERIF-09 | TEST-01 | Init workflow - re-initialization rejected |
| VERIF-10 | TEST-01 | Init workflow - original goal preserved |
| VERIF-11 | TEST-01 | Init workflow - state unchanged on re-init |
| VERIF-12 | TEST-01 | Init workflow - input validation works |
| VERIF-13 | TEST-01 | Init workflow - empty goal handled |
| VERIF-14 | TEST-01 | Init workflow - partial init prevented |
| VERIF-15 | TEST-02 | Execute workflow - phase status in_progress |
| VERIF-16 | TEST-02 | Execute workflow - autonomous spawning occurs |
| VERIF-17 | TEST-02 | Execute workflow - Task tool used |
| VERIF-18 | TEST-02 | Execute workflow - Worker Ant activity updates |
| VERIF-19 | TEST-02 | Execute workflow - pheromones emitted |
| VERIF-20 | TEST-02 | Execute workflow - events published |
| VERIF-21 | TEST-02 | Execute workflow - phase completed |
| VERIF-22 | TEST-02 | Execute workflow - spawn tracking populated |
| VERIF-23 | TEST-02 | Execute workflow - errors logged on blocked tasks |
| VERIF-24 | TEST-02 | Execute workflow - phase not complete on errors |
| VERIF-25 | TEST-02 | Execute workflow - spawn budget not exhausted |
| VERIF-26 | TEST-02 | Execute workflow - colony recovers from errors |
| VERIF-27 | TEST-02 | Execute workflow - re-execute rejected |
| VERIF-28 | TEST-02 | Execute workflow - phase status unchanged |
| VERIF-29 | TEST-02 | Execute workflow - no duplicate spawns |
| VERIF-30 | TEST-03 | Spawning workflow - capability gap detected |
| VERIF-31 | TEST-03 | Spawning workflow - specialist spawned |
| VERIF-32 | TEST-03 | Spawning workflow - spawn recorded |
| VERIF-33 | TEST-03 | Spawning workflow - Bayesian confidence updated |
| VERIF-34 | TEST-03 | Spawning workflow - circuit breaker not triggered |
| VERIF-35 | TEST-03 | Spawning workflow - spawn depth within limit |
| VERIF-36 | TEST-03 | Spawning workflow - spawn budget available |
| VERIF-37 | TEST-03 | Spawning workflow - meta-learning data populated |
| VERIF-38 | TEST-03 | Spawning workflow - circuit breaker activated |
| VERIF-39 | TEST-03 | Spawning workflow - spawn rejected after failures |
| VERIF-40 | TEST-03 | Spawning workflow - failure count tracked |
| VERIF-41 | TEST-03 | Spawning workflow - circuit breaker timestamp set |
| VERIF-42 | TEST-03 | Spawning workflow - spawn blocked flag set |
| VERIF-43 | TEST-03 | Spawning workflow - max depth reached |
| VERIF-44 | TEST-03 | Spawning workflow - depth limit enforced |
| VERIF-45 | TEST-03 | Spawning workflow - depth error logged |
| VERIF-46 | TEST-03 | Spawning workflow - no infinite loops |
| VERIF-47 | TEST-04 | Memory workflow - DAST compression triggered |
| VERIF-48 | TEST-04 | Memory workflow - working memory reduced |
| VERIF-49 | TEST-04 | Memory workflow - short-term memory populated |
| VERIF-50 | TEST-04 | Memory workflow - associative links created |
| VERIF-51 | TEST-04 | Memory workflow - memory metadata updated |
| VERIF-52 | TEST-04 | Memory workflow - memory files updated |
| VERIF-53 | TEST-04 | Memory workflow - no memory loss |
| VERIF-54 | TEST-04 | Memory workflow - overflow detected |
| VERIF-55 | TEST-04 | Memory workflow - error logged |
| VERIF-56 | TEST-04 | Memory workflow - colony stable on overflow |
| VERIF-57 | TEST-04 | Memory workflow - oldest items archived |
| VERIF-58 | TEST-04 | Memory workflow - associative links created |
| VERIF-59 | TEST-04 | Memory workflow - related items linked |
| VERIF-60 | TEST-04 | Memory workflow - retrieval improved |
| VERIF-61 | TEST-05 | Voting workflow - all Watchers vote |
| VERIF-62 | TEST-05 | Voting workflow - votes recorded |
| VERIF-63 | TEST-05 | Voting workflow - supermajority calculated |
| VERIF-64 | TEST-05 | Voting workflow - weights applied |
| VERIF-65 | TEST-05 | Voting workflow - decision outcome logged |
| VERIF-66 | TEST-05 | Voting workflow - issues deduplicated |
| VERIF-67 | TEST-05 | Voting workflow - vote metadata recorded |
| VERIF-68 | TEST-05 | Voting workflow - supermajority status updated |
| VERIF-69 | TEST-05 | Voting workflow - Critical veto exercised |
| VERIF-70 | TEST-05 | Voting workflow - veto blocks approval |
| VERIF-71 | TEST-05 | Voting workflow - veto reason logged |
| VERIF-72 | TEST-05 | Voting workflow - veto flag set |
| VERIF-73 | TEST-05 | Voting workflow - Critical watcher power verified |
| VERIF-74 | TEST-05 | Voting workflow - zero watcher edge case handled |
| VERIF-75 | TEST-05 | Voting workflow - abstentions handled |
| VERIF-76 | TEST-05 | Voting workflow - no crash on edge cases |
| VERIF-77 | TEST-05 | Voting workflow - default decision applied |
| VERIF-78 | TEST-06 | Event workflow - polling retrieves events |
| VERIF-79 | TEST-06 | Event workflow - topic filtering works |
| VERIF-80 | TEST-06 | Event workflow - event schema valid |
| VERIF-81 | TEST-06 | Event workflow - delivery marks processed |
| VERIF-82 | TEST-06 | Event workflow - reprocessing prevented |
| VERIF-83 | TEST-06 | Event workflow - caste subscriptions work |
| VERIF-84 | TEST-06 | Event workflow - event logging records polls |
| VERIF-85 | TEST-06 | Event workflow - delivery tracking updated |
| VERIF-86 | TEST-06 | Event workflow - invalid events handled |
| VERIF-87 | TEST-06 | Event workflow - error logged |
| VERIF-88 | TEST-06 | Event workflow - colony continues on error |
| VERIF-89 | TEST-06 | Event workflow - failed events marked |
| VERIF-90 | TEST-06 | Event workflow - colonizer subscriptions verified |
| VERIF-91 | TEST-06 | Event workflow - watcher subscriptions verified |
| VERIF-92 | TEST-06 | Event workflow - topic filtering prevents crossover |
| VERIF-93 | TEST-06 | Event workflow - error topic received by all |
| VERIF-94 | TEST-06 | Event workflow - subscription criteria applied |

### Requirement Coverage Summary

| Requirement | Verification Checks | Coverage |
|-------------|-------------------|----------|
| TEST-01 (Init) | VERIF-01 through VERIF-14 | 14 checks |
| TEST-02 (Execute) | VERIF-15 through VERIF-29 | 15 checks |
| TEST-03 (Spawning) | VERIF-30 through VERIF-46 | 17 checks |
| TEST-04 (Memory) | VERIF-47 through VERIF-60 | 14 checks |
| TEST-05 (Voting) | VERIF-61 through VERIF-77 | 17 checks |
| TEST-06 (Event) | VERIF-78 through VERIF-94 | 17 checks |
| **Total** | **VERIF-01 through VERIF-94** | **94 checks** |

---

**End of E2E Test Guide**

For questions or issues with this guide, refer to:
- Phase 13 CONTEXT.md - Implementation decisions
- Phase 13 RESEARCH.md - Testing best practices
- PROJECT.md - Overall project context
