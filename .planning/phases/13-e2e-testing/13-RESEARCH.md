# Phase 13: E2E Testing - Research

**Researched:** 2026-02-02
**Domain:** Manual E2E Testing Documentation for LLM-Based Systems
**Confidence:** HIGH

## Summary

This phase requires creating comprehensive manual test guides that document all core Aether colony workflows with step-by-step instructions, expected outputs, and verification checks. The research focused on understanding best practices for manual E2E testing documentation, particularly for LLM-based autonomous agent systems where deterministic automated testing is insufficient.

**Key findings:**
- Manual testing is essential for LLM-based systems due to non-deterministic behavior
- Test documentation should follow structured format: Overview → Prerequisites → Test Steps → Expected Outputs → Verification Checks
- Verification checks should use traceable IDs (VERIF-01, VERIF-02, etc.) for requirement mapping
- Each workflow needs success, failure, and edge case coverage
- State verification before/after each test is critical for autonomous systems

**Primary recommendation:** Create a markdown-based E2E test guide with 6 workflow sections (init, execute, spawning, memory, voting, event), each containing 3 test scenarios (happy path, failure case, edge case) with numbered steps, code-block expected outputs, and bullet-point verification checks with traceable IDs.

## Standard Stack

### Core Documentation Tools
| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| Markdown | GitHub Flavored | Test documentation format | Universal readability, version control friendly, supports code blocks |
| JSON Schema | N/A | State file validation | Existing colony state uses JSON, verification requires schema validation |
| Bash | 5.0+ | Test execution commands | Existing utility scripts use bash, colony commands are bash-based |

### Supporting Tools
| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| jq | 1.6+ | JSON parsing for verification | Verifying state file changes, checking event delivery |
| Git | 2.30+ | State backup/restore | Test setup/teardown requires colony state snapshots |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Markdown | AsciiDoc | Markdown has better tooling, wider adoption |
| Manual verification | Automated test suite | LLMs are non-deterministic; manual tests catch reasoning issues |

**Installation:**
```bash
# Standard tools (likely already installed)
jq --version    # Verify jq available
bash --version   # Verify bash 5.0+
git --version    # Verify git available
```

## Architecture Patterns

### Recommended Test Guide Structure

```
E2E-TEST-GUIDE.md
├── Introduction                   # How to use this guide
├── Test Environment Setup         # Prerequisites for running tests
├── Workflow 1: Init              # Colony initialization tests
│   ├── Overview
│   ├── Prerequisites
│   ├── Test 1.1: Happy Path      # Successful initialization
│   ├── Test 1.2: Failure Case    # Re-initialization attempt
│   └── Test 1.3: Edge Case       # Invalid goal input
├── Workflow 2: Execute           # Phase execution tests
│   ├── Overview
│   ├── Prerequisites
│   ├── Test 2.1: Happy Path      # Successful phase execution
│   ├── Test 2.2: Failure Case    # Phase execution with errors
│   └── Test 2.3: Edge Case       # Phase with blocked tasks
├── Workflow 3: Spawning          # Autonomous spawning tests
│   ├── Overview
│   ├── Prerequisites
│   ├── Test 3.1: Happy Path      # Successful specialist spawn
│   ├── Test 3.2: Failure Case    # Circuit breaker activation
│   └── Test 3.3: Edge Case       # Max spawn depth reached
├── Workflow 4: Memory            # Memory compression tests
│   ├── Overview
│   ├── Prerequisites
│   ├── Test 4.1: Happy Path      # DAST compression working
│   ├── Test 4.2: Failure Case    # Memory overflow handling
│   └── Test 4.3: Edge Case       # Associative link creation
├── Workflow 5: Voting            # Multi-perspective verification tests
│   ├── Overview
│   ├── Prerequisites
│   ├── Test 5.1: Happy Path      # Supermajority approval
│   ├── Test 5.2: Failure Case    # Critical veto blocks approval
│   └── Test 5.3: Edge Case       # Weight calculation edge cases
├── Workflow 6: Event             # Event polling and delivery tests
│   ├── Overview
│   ├── Prerequisites
│   ├── Test 6.1: Happy Path      # Event polling and delivery
│   ├── Test 6.2: Failure Case    # Event delivery failure
│   └── Test 6.3: Edge Case       # Caste-specific filtering
└── Appendix A: Verification ID Mapping  # VERIF-XX to requirement mapping
```

### Pattern 1: Test Case Format

**What:** Standardized structure for each test case

**When to use:** Every test case in the guide

**Example:**
```markdown
## Test 1.1: Happy Path - Successful Colony Initialization

### Overview
Verify that the colony initializes correctly with a valid goal, creating all necessary state files and mobilizing Worker Ants.

### Prerequisites
- Colony not initialized (no `.aether/data/COLONY_STATE.json` or goal is null)
- `/ant:init` command available
- Git repository initialized

### Test Steps

1. **Initialize colony with valid goal**
   ```bash
   /ant:init "Build a REST API for task management"
   ```

2. **Verify initialization output displays correctly**
   - Check for initialization progress steps
   - Confirm colony mobilized message shown

3. **Check colony state file created**
   ```bash
   cat .aether/data/COLONY_STATE.json | jq '.queen_intention.goal'
   ```

### Expected Outputs

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
```

### Verification Checks

- **VERIF-01:** Colony state file exists at `.aether/data/COLONY_STATE.json`
- **VERIF-02:** Queen's intention stored correctly: `jq -r '.queen_intention.goal'` returns "Build a REST API for task management"
- **VERIF-03:** Colony status set to "INIT": `jq -r '.colony_status.state'` returns "INIT"
- **VERIF-04:** Current phase set to 1: `jq -r '.colony_status.current_phase'` returns "1"
- **VERIF-05:** All 6 Worker Ant castes have status "ready": `jq '.worker_registry | to_entries[] | select(.value.status != "ready")'` returns empty
- **VERIF-06:** INIT pheromone active with strength 1.0: `jq '.active_pheromones[] | select(.type == "INIT") | .strength'` returns "1.0"
- **VERIF-07:** Session ID generated and non-empty: `jq -r '.colony_metadata.session_id'` returns non-null value
- **VERIF-08:** Working memory contains intention: `jq '.working_memory.items[] | select(.type == "intention")'` returns intention entry
```

### Pattern 2: State Verification Pattern

**What:** Before/after state verification for autonomous systems

**When to use:** Every test that modifies colony state

**Example:**
```markdown
### State Verification

**Before Test:**
```bash
# Colony should be in clean state
jq -r '.colony_status.state' .aether/data/COLONY_STATE.json  # Should return "IDLE" or not exist
jq '.resource_budgets.current_spawns' .aether/data/COLONY_STATE.json  # Should return 0
```

**After Test:**
```bash
# Verify state changes
jq -r '.colony_status.state' .aether/data/COLONY_STATE.json  # Should return "INIT"
jq '.resource_budgets.current_spawns' .aether/data/COLONY_STATE.json  # Should still be 0
jq -r '.queen_intention.goal' .aether/data/COLONY_STATE.json  # Should match input goal
```
```

### Pattern 3: Verification ID Mapping

**What:** Traceability matrix linking verification checks to requirements

**When to use:** Appendix section for requirement traceability

**Example:**
```markdown
## Appendix A: Verification ID Mapping

| Verification ID | Requirement | Description |
|----------------|-------------|-------------|
| VERIF-01 | TEST-01 | Init workflow - state file creation |
| VERIF-02 | TEST-01 | Init workflow - intention storage |
| VERIF-03 | TEST-01 | Init workflow - colony status transition |
| VERIF-10 | TEST-02 | Execute workflow - autonomous spawning occurs |
| VERIF-11 | TEST-02 | Execute workflow - Task tool used for spawning |
| VERIF-20 | TEST-03 | Spawning workflow - Bayesian confidence update |
| VERIF-30 | TEST-04 | Memory workflow - DAST compression triggered |
| VERIF-40 | TEST-05 | Voting workflow - supermajority calculation |
| VERIF-41 | TEST-05 | Voting workflow - Critical veto power |
| VERIF-50 | TEST-06 | Event workflow - polling retrieves events |
| VERIF-51 | TEST-06 | Event workflow - delivery tracking prevents reprocessing |
```

### Anti-Patterns to Avoid

- **Vague expected outputs:** Don't use "colony initializes successfully" - specify exact output format with emojis and structure
- **Missing verification checks:** Every test should have at least 3-5 verifiable checks with IDs
- **No state verification:** Autonomous systems require before/after state checks to prevent cascading failures
- **Missing edge cases:** Don't only test happy paths - test failures, boundaries, and error conditions
- **Unclear prerequisites:** Always specify what state the colony must be in before testing

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Test environment management | Custom setup/teardown scripts | Existing `.aether/utils/` scripts for state backup/restore | Already handles atomic writes, file locking |
| JSON validation | Custom jq validation | Existing test patterns from `test-voting-system.sh`, `test-event-polling-integration.sh` | Proven patterns for state verification |
| Verification check formatting | Custom verification format | VERIF-XX IDs matching existing test suite patterns | Consistent with existing automation tests |
| Progress display | Custom bash progress | Existing step tracking from `/ant:init` command | Reuses visual indicators from Phase 12 |
| Event verification | Manual event checking | `get_events_for_subscriber()`, `mark_events_delivered()` from event-bus.sh | Already tested integration patterns |

**Key insight:** The existing test utilities (test-voting-system.sh, test-event-polling-integration.sh, test-spawning-safeguards.sh) provide proven patterns for bash-based testing. The E2E guide should document manual testing steps that mirror these automated patterns but focus on LLM behavior validation rather than component testing.

## Common Pitfalls

### Pitfall 1: Insufficient State Verification

**What goes wrong:** Tests verify output visually but don't check actual state file changes, leading to false positives where output looks correct but internal state is corrupted.

**Why it happens:** Manual testing focuses on visible outputs; autonomous systems have hidden state changes that aren't visible in command output.

**How to avoid:** Every test must include "State Verification" section with before/after jq checks for all relevant state files (COLONY_STATE.json, worker_ants.json, pheromones.json, memory.json, events.json).

**Warning signs:** Test passes but subsequent tests fail mysteriously; colony state inconsistent after tests; need to reinitialize colony between tests.

### Pitfall 2: Missing Edge Cases for Autonomous Behavior

**What goes wrong:** Tests only verify happy paths, missing autonomous system edge cases like circuit breaker activation, spawn depth limits, or Bayesian confidence boundaries.

**Why it happens:** Manual testing naturally focuses on common cases; edge cases require deliberate effort to identify and test.

**How to avoid:** Each workflow MUST include 3 test scenarios: happy path (success), failure case (error handling), edge case (boundary condition). Use existing test suites (test-spawning-safeguards.sh, test-voting-system.sh) to identify edge cases.

**Warning signs:** All tests pass on first try; no tests trigger error conditions; resource limits never tested.

### Pitfall 3: Inconsistent Verification Check Formatting

**What goes wrong:** Verification checks use different formats, lack traceable IDs, making it impossible to map to requirements or track coverage.

**Why it happens:** Natural language verification checks are easier to write but harder to track and validate.

**How to avoid:** Use strict VERIF-XX format for all verification checks, maintain mapping table in Appendix, ensure each requirement (TEST-01 through TEST-06) maps to multiple verification checks.

**Warning signs:** Verification checks are prose paragraphs; cannot count verification checks per requirement; no mapping to TEST-XX requirements.

### Pitfall 4: Ignoring LLM Non-Determinism

**What goes wrong:** Tests assume deterministic outputs, failing when LLM produces equivalent but differently worded responses.

**Why it happens:** Automated testing mindset carries over to manual testing; LLMs are inherently non-deterministic.

**How to avoid:** Focus verification on structural correctness (state files, JSON schemas, required fields) rather than exact output text. Use jq for JSON validation rather than text matching. Allow for output variation while verifying core functionality.

**Warning signs:** Tests fail due to minor wording differences; verification requires exact string matching; tests are fragile.

### Pitfall 5: Inadequate Test Isolation

**What goes wrong:** Tests depend on each other or modify shared state, making it impossible to run tests individually or diagnose failures.

**Why it happens:** Manual testing often assumes sequential execution; cleanup steps are forgotten or incomplete.

**How to avoid:** Each test should include setup (prerequisites) and teardown (cleanup) sections. Use colony state backup/restore for isolation. Document any tests that MUST be run sequentially.

**Warning signs:** Tests fail unless run in specific order; need to reinitialize colony frequently; test results vary based on previous tests.

## Code Examples

### Test Case Template

```markdown
## Test X.Y: [Test Name]

### Overview
[Brief description of what this test validates and why it matters]

### Prerequisites
- Colony state: [specific state required]
- Files needed: [any files that must exist]
- Configuration: [any config requirements]

### Test Steps

1. **[Step title]**
   ```bash
   [Command or action]
   ```
   [Additional context or notes]

2. **[Step title]**
   [Continue with remaining steps...]

### Expected Outputs

```
[Expected terminal output or visual result]
```

### Verification Checks

- **VERIF-XX:** [Specific check that can be verified]
- **VERIF-XX:** [Another specific check]
- [Continue with 3-5+ verification checks]

### State Verification

**Before Test:**
```bash
[jq commands to verify initial state]
```

**After Test:**
```bash
[jq commands to verify final state]
```

### Cleanup (if needed)

```bash
[Commands to restore colony to clean state]
```
```

### Verification Check Examples

```markdown
# State file verification
- **VERIF-01:** Colony state file exists: `[ -f .aether/data/COLONY_STATE.json ]`

# JSON field verification
- **VERIF-02:** Goal stored correctly: `jq -r '.queen_intention.goal' .aether/data/COLONY_STATE.json` returns "Build a REST API"

# Count verification
- **VERIF-03:** All 6 castes mobilized: `jq '.worker_registry | length' .aether/data/worker_ants.json` returns "6"

# Status verification
- **VERIF-04:** All workers ready: `jq '[.worker_registry[] | select(.status != "ready")] | length' .aether/data/worker_ants.json` returns "0"

# Pheromone verification
- **VERIF-05:** INIT pheromone active: `jq '.active_pheromones[] | select(.type == "INIT") | .strength' .aether/data/pheromones.json` returns "1.0"

# Event verification
- **VERIF-06:** Event delivered: `jq -r '.delivered_to[] | select(.subscriber_id == "test_colonizer")' .aether/data/events.json` contains subscriber

# Spawn verification
- **VERIF-07:** Spawn recorded: `jq '.spawn_tracking.spawn_history[-1].outcome' .aether/data/COLONY_STATE.json` returns "success" or "pending"

# Memory verification
- **VERIF-08:** Intention in working memory: `jq '.working_memory.items[] | select(.type == "intention") | .content' .aether/data/memory.json` returns goal text

# Voting verification
- **VERIF-09:** Supermajority calculated: `jq -r '.verification.last_supermajority' .aether/data/COLONY_STATE.json` returns "APPROVED" or "REJECTED"

# Bayesian confidence verification
- **VERIF-10:** Confidence updated: `jq -r '.meta_learning.specialist_confidence."database_specialist"."database"' .aether/data/COLONY_STATE.json` returns value between 0.0-1.0
```

### State Backup/Restore Pattern

```markdown
### Test Setup

```bash
# Backup colony state before test
cp .aether/data/COLONY_STATE.json .aether/data/COLONY_STATE.test.backup
cp .aether/data/worker_ants.json .aether/data/worker_ants.test.backup
cp .aether/data/pheromones.json .aether/data/pheromones.test.backup
cp .aether/data/memory.json .aether/data/memory.test.backup
cp .aether/data/events.json .aether/data/events.test.backup
```

### Test Cleanup

```bash
# Restore colony state after test
mv .aether/data/COLONY_STATE.test.backup .aether/data/COLONY_STATE.json
mv .aether/data/worker_ants.test.backup .aether/data/worker_ants.json
mv .aether/data/pheromones.test.backup .aether/data/pheromones.json
mv .aether/data/memory.test.backup .aether/data/memory.json
mv .aether/data/events.test.backup .aether/data/events.json
```
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Automated-only testing | Manual E2E testing for LLM workflows | 2024-2025 | LLM non-determinism requires manual validation |
| Text-based verification | ID-based verification with traceability | 2025 | Requirement traceability becomes critical |
| Single test per workflow | Happy path + failure + edge case per workflow | 2025 | Comprehensive coverage requires multiple scenarios |
| Output verification only | State verification (before/after) | 2025 | Autonomous systems require state validation |
| Ad-hoc test documentation | Structured test case format | 2025 | Reusability and maintainability improve |

**Deprecated/outdated:**
- **Automated-only testing for LLM systems:** LLMs are non-deterministic; manual tests catch reasoning issues that automated tests miss
- **Exact string matching for verification:** LLM output varies; structural verification (JSON, state files) is more reliable
- **Single-scenario testing:** Edge cases and failure modes are critical for autonomous systems

## Open Questions

1. **Verification check granularity:** How many verification checks per test case? Research suggests 3-5 minimum for simple tests, 8-12 for complex workflows. Recommendation: Use judgment based on workflow complexity, ensure each requirement (TEST-01 through TEST-06) maps to at least 5 verification checks.

2. **Test independence vs. sequential execution:** Should tests be fully independent (requires setup/teardown for each) or assume sequential execution? Recommendation: Design for independence but note sequential dependencies where they exist. Document tests that require specific execution order.

3. **LLM output tolerance:** How much variation in LLM output is acceptable? Research focuses on structural verification rather than exact text matching. Recommendation: Verify core functionality (state changes, JSON fields, schemas) rather than exact wording.

## Sources

### Primary (HIGH confidence)
- **Existing Aether test utilities:** `.aether/utils/test-voting-system.sh`, `.aether/utils/test-event-polling-integration.sh`, `.aether/utils/test-spawning-safeguards.sh` - Proven patterns for bash-based testing, verification check formatting, state validation
- **Aether command documentation:** `.claude/commands/ant/init.md`, `.claude/commands/ant/execute.md`, `.claude/commands/ant/status.md` - Source for workflow steps, expected outputs, state transitions
- **Phase 13 CONTEXT.md:** Locked decisions on guide structure, test format, verification depth - Direct constraints on implementation

### Secondary (MEDIUM confidence)
- **[Best Practices for End-to-End Testing in 2025](https://www.bunnyshell.com/blog/best-practices-for-end-to-end-testing-in-2025/)** (Bunnyshell, 2025) - Shift-left testing, focus on critical user journeys, test environment management
- **[7 Best Practices for End-to-End (E2E) Testing](https://www.ibm.com/think/insights/end-to-end-end-to-end-testing-best-practices)** (IBM, 2025) - Consider application users, build documented test cases, conducive test environment
- **[Testing Documents in QA: A Complete Guide (2025)](https://www.botgauge.com/blog/testing-documents-in-qa-guide)** (Aug 2, 2025) - Test case documentation standards, sample outlines
- **[How to Document Manual Testing Results Effectively](https://www.testdevlab.com/blog/effective-documentation-for-manual-testing)** (May 6, 2025) - Manual testing documentation best practices
- **[Markdown test specifications](https://www.matrixprojects.net/p/markdown-test-specifications/)** - Markdown format for test cases with expected output validation

### Tertiary (LOW confidence)
- **[LLM Testing in 2025: The Ultimate Guide](https://orq.ai/blog/llm-testing)** (Feb 27, 2025) - LLM testing challenges, accuracy and security focus (not specific to manual testing)
- **[A Practical Guide to LLM & Agent Evaluation](https://trilogyai.substack.com/p/a-practical-guide-to-llm-and-agent)** (2025) - LLM agent evaluation approaches (general guidance, not format-specific)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Based on existing Aether tooling and proven patterns
- Architecture: HIGH - Locked decisions from CONTEXT.md, proven test patterns from existing utilities
- Pitfalls: HIGH - Based on research into 2025 testing best practices and LLM-specific challenges
- Code examples: HIGH - Derived from existing test utilities and command documentation

**Research date:** 2026-02-02
**Valid until:** 30 days (manual testing documentation patterns are stable; LLM testing practices are evolving but core principles established)
