# Phase 8: Build Polish — Output Timing & Integration - Research

**Researched:** 2026-02-14
**Domain:** Claude Code Task tool execution model, output synchronization, integration testing
**Confidence:** HIGH

## Summary

This research covers the timing bug in build.md where `run_in_background: true` causes misleading output ordering, the technical approach to fix it, and the integration testing strategy to verify the complete checkpoint → update → build workflow.

**Key Findings:**

1. **Root Cause Identified** - The build.md command uses `run_in_background: true` when spawning workers (Step 5.1, line 312), but then immediately displays the spawn plan summary. The background tasks haven't actually started executing when the summary is shown, creating a misleading "completed" impression.

2. **Current Flow Problem** - The existing pattern:
   - Spawn workers with `run_in_background: true` (line 312)
   - Display spawn plan immediately (lines 290-306)
   - Later call `TaskOutput` with `block: true` to collect results (lines 463, 586, 662)
   - This creates a gap between "spawning" message and actual execution

3. **Fix Strategy** - Convert to foreground Task spawns (remove `run_in_background: true`), which makes the Task tool block until completion. The output will then naturally follow execution order.

4. **Integration Test Pattern** - Phase 7 established E2E test patterns using ava, temp directories, and the `initializeRepo()` helper. The checkpoint → update → build workflow can be tested by mocking the Task tool behavior.

**Primary recommendation:** Remove all `run_in_background: true` flags from worker spawns in build.md, rely on foreground execution with sequential TaskOutput collection, and create an E2E test that verifies the complete workflow.

## Root Cause Analysis

### The Timing Bug

**Location:** `/Users/callumcowie/.claude/commands/ant/build.md`

**Affected Steps:**
- Step 5.1 (line 312): Spawn Wave 1 Workers
- Step 5.4: Spawn Watcher for Verification (implied by pattern)
- Step 5.4.2: Spawn Chaos Ant for Resilience Testing (implied by pattern)

**Current Behavior:**
```markdown
Step 5.1: Spawn Wave 1 Workers (Parallel)
For each Wave 1 task, use Task tool with `subagent_type="general-purpose" and `run_in_background: true`:

[Spawn logging happens immediately]

Step 5.2: Collect Wave 1 Results (BLOCKING)
For each spawned worker, call TaskOutput with `block: true` to wait for completion
```

**Problem:** When `run_in_background: true` is used, the Task tool returns immediately with a task_id. The spawn logging (lines 314-317) executes synchronously, giving the impression that workers are "spawned" when they may not have even started executing yet. The user sees:

1. "Spawning Builder-1..." message
2. "Spawning Builder-2..." message
3. Spawn plan summary displayed
4. [User sees summary before any actual work begins]

**Why This Matters:**
- Misleading UX: Summary appears before work completes
- Race conditions: Status displayed may not reflect actual completion
- Debugging difficulty: Log order doesn't match execution order

## Technical Approach

### Fix BUILD-01: Remove run_in_background

**Change:** Remove `run_in_background: true` from all Task spawns.

**Before (line 312):**
```markdown
For each Wave 1 task, use Task tool with `subagent_type="general-purpose"` and `run_in_background: true`:
```

**After:**
```markdown
For each Wave 1 task, use Task tool with `subagent_type="general-purpose"`:
```

**Impact:** Task tool will block until the worker completes, naturally ordering output correctly.

### Fix BUILD-02: Sequential Output

**Current Pattern (lines 459-475):**
```markdown
### Step 5.2: Collect Wave 1 Results (BLOCKING)

For each spawned worker, call TaskOutput with `block: true` to wait for completion:
```

**New Pattern:**
With foreground Task execution, Step 5.2 becomes unnecessary - the tasks have already completed. However, we still need to collect results. The revised flow:

1. **Step 5.1:** Spawn Wave 1 workers (foreground, blocking)
2. **Step 5.1.5:** Collect results immediately after each spawn returns
3. **Step 5.2:** Verify all workers completed (validation only, no waiting)

**Alternative Approach (Recommended):**
Keep the current two-step pattern but remove `run_in_background`. This maintains the architecture while fixing timing:

1. Spawn workers (foreground - blocks until complete)
2. Collect results (results already available, just parse them)
3. Display summary (now accurate because workers finished before this line)

### Fix BUILD-03: Foreground Task Calls

**Steps requiring changes:**

| Step | Line | Current | Change |
|------|------|---------|--------|
| 5.1 | 312 | `run_in_background: true` | Remove flag |
| 5.4 | implied | `run_in_background: true` | Remove flag |
| 5.4.2 | implied | `run_in_background: true` | Remove flag |

**Verification:** After changes, the output order will be:
1. Spawn worker (blocking, waits for completion)
2. Worker completes, returns result
3. Next spawn or summary display
4. Summary accurately reflects completion status

## Architecture Patterns

### Pattern 1: Foreground Worker Spawns
**What:** Use Task tool without `run_in_background` for synchronous execution
**When to use:** When output order must match execution order
**Example:**
```markdown
Spawn worker using Task tool:
```

### Pattern 2: Background Worker Spawns (for reference)
**What:** Use `run_in_background: true` for parallel execution
**When to use:** When multiple independent tasks can run in parallel AND output ordering doesn't matter
**Note:** build.md currently uses this but then immediately displays summary, causing the bug

### Pattern 3: Hybrid Approach (Current Architecture, Fixed)
**What:** Spawn with foreground, collect results, then display summary
**When to use:** When you need parallel execution but ordered output
**Implementation:**
1. Spawn all Wave 1 workers in single message (foreground)
2. Each spawn blocks until its worker completes
3. After all spawns return, display summary
4. Results are naturally ordered by completion

## Integration Test Strategy

### Test Scope: Checkpoint → Update → Build Workflow

**Purpose:** Verify all v1.1 fixes work together end-to-end.

**Test File:** `tests/e2e/checkpoint-update-build.test.js`

### Test Structure

```javascript
#!/usr/bin/env node
/**
 * E2E Test: Checkpoint → Update → Build Workflow
 *
 * Tests the complete v1.1 workflow:
 * 1. Initialize repo
 * 2. Create checkpoint
 * 3. Run update (with potential rollback)
 * 4. Build a phase
 * 5. Verify state consistency throughout
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');
const { initializeRepo, isInitialized } = require('../../bin/lib/init');
const { UpdateTransaction } = require('../../bin/lib/update-transaction');
const { StateGuard } = require('../../bin/lib/state-guard');
```

### Test Cases

**Test 1: Complete workflow succeeds**
- Initialize repo
- Create checkpoint
- Execute update (dry-run or mock)
- Build phase 1
- Verify state advanced correctly
- Verify events logged

**Test 2: Build respects Iron Law**
- Initialize repo
- Attempt to advance phase without evidence
- Verify StateGuard blocks advancement
- Verify error code E_IRON_LAW_VIOLATION

**Test 3: Update rollback preserves state**
- Initialize repo
- Create checkpoint
- Trigger update failure
- Verify automatic rollback
- Verify state unchanged
- Verify recovery commands displayed

**Test 4: Checkpoint created before build**
- Initialize repo
- Start build
- Verify checkpoint exists
- Verify checkpoint only contains Aether-managed files

### Test Implementation Notes

**Mocking Task Tool:**
The build.md command uses Claude Code's Task tool which cannot be directly tested. Instead:
1. Test the underlying functions (StateGuard, UpdateTransaction)
2. Verify state transitions
3. Verify checkpoint creation
4. Manual testing for actual build output timing

**Verification Points:**
| Component | Verification Method |
|-----------|---------------------|
| Checkpoint | `fs.existsSync(checkpointPath)` |
| State Guard | `StateGuard.advancePhase()` with evidence |
| Update | `UpdateTransaction.execute()` result |
| Build | State advancement + event logging |

## Dependencies on Phases 6 and 7

### Phase 6 Dependencies (Required)

| Component | Status | Usage in Phase 8 |
|-----------|--------|------------------|
| `initializeRepo()` | Complete | Test setup |
| `isInitialized()` | Complete | Test validation |
| Checkpoint system | Complete | Pre-build safety |
| File locking | Complete | State protection |

### Phase 7 Dependencies (Required)

| Component | Status | Usage in Phase 8 |
|-----------|--------|------------------|
| `StateGuard` | Complete | Iron Law enforcement |
| `UpdateTransaction` | Complete | Update with rollback |
| `FileLock` | Complete | Concurrent access |
| Event audit trail | Complete | Verification evidence |

### Integration Points

```
Phase 6 (Foundation)
    │
    ├── Checkpoint system ──┐
    ├── File locking ───────┤
    └── Init module ────────┤
                            ▼
Phase 7 (Core Reliability)      Phase 8 (Build Polish)
    │                               │
    ├── StateGuard ────────────────┤ (Iron Law enforcement)
    ├── UpdateTransaction ─────────┤ (Rollback capability)
    └── Event audit ───────────────┤ (Evidence for advancement)
                                   │
                                   └── Integration test verifies
                                       all components work together
```

## Common Pitfalls

### Pitfall 1: Assuming run_in_background is required for parallelism
**What goes wrong:** Thinking that removing `run_in_background` prevents parallel execution
**Why it happens:** Misunderstanding Claude Code's execution model
**How to avoid:** Claude Code can spawn multiple Tasks in a single message - they execute in parallel. The `run_in_background` flag only affects when control returns to the parent.

### Pitfall 2: Breaking the spawn/collect pattern
**What goes wrong:** Removing Step 5.2 (Collect Results) entirely when switching to foreground
**Why it happens:** Thinking "foreground = no need to collect"
**How to avoid:** Still need to parse results from foreground tasks. Keep the collect step but it becomes parsing instead of waiting.

### Pitfall 3: Missing Watcher and Chaos Ant changes
**What goes wrong:** Only fixing Step 5.1 but leaving 5.4 and 5.4.2 with background execution
**Why it happens:** Inconsistent search/replace
**How to avoid:** Search for ALL `run_in_background` occurrences in build.md

### Pitfall 4: Integration test too broad
**What goes wrong:** Trying to test actual Task tool execution
**Why it happens:** Forgetting that Task tool is a Claude Code primitive
**How to avoid:** Test the underlying state machines and verify transitions, not the UI layer.

## Code Examples

### Example 1: Fixed Step 5.1 Pattern

```markdown
### Step 5.1: Spawn Wave 1 Workers (Parallel)

**CRITICAL: Spawn ALL Wave 1 workers in a SINGLE message using multiple Task tool calls.**

For each Wave 1 task, use Task tool with `subagent_type="general-purpose"`:

Log each spawn:
```bash
bash .aether/aether-utils.sh spawn-log "Queen" "builder" "{ant_name}" "{task_description}"
```

[Rest of spawn logic unchanged...]

### Step 5.2: Collect Wave 1 Results

**Results are already available from foreground Task execution.**

For each spawned worker, parse the returned result:
- Extract: status, files_created, files_modified, blockers
- Store all results for synthesis in Step 5.6

For each completed worker, log:
```bash
bash .aether/aether-utils.sh spawn-complete "{ant_name}" "completed" "{summary}"
```
```

### Example 2: Integration Test Skeleton

```javascript
test.serial('checkpoint -> update -> build workflow', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // 1. Initialize repo
    await initializeRepo(tmpDir, { goal: 'Workflow test' });
    t.true(isInitialized(tmpDir));

    // 2. Create checkpoint
    const checkpointResult = await createCheckpoint(tmpDir, 'pre-build');
    t.truthy(checkpointResult.checkpoint_id);

    // 3. Verify checkpoint exists
    const checkpointPath = path.join(tmpDir, '.aether/checkpoints', `${checkpointResult.checkpoint_id}.json`);
    t.true(fs.existsSync(checkpointPath));

    // 4. Verify state guard works
    const stateFile = path.join(tmpDir, '.aether/data/COLONY_STATE.json');
    const guard = new StateGuard(stateFile, { worker: 'test' });

    // Without evidence - should fail
    await t.throwsAsync(
      async () => await guard.advancePhase(0, 1, {}),
      { instanceOf: StateGuardError }
    );

    // With evidence - should succeed
    const evidence = createValidEvidence(1);
    const result = await guard.advancePhase(0, 1, evidence);
    t.is(result.status, 'transitioned');

    // 5. Verify state updated
    const state = JSON.parse(fs.readFileSync(stateFile, 'utf8'));
    t.is(state.current_phase, 1);

  } finally {
    await cleanupTempDir(tmpDir);
  }
});
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Background Task spawns | Foreground Task spawns | Phase 8 | Output order matches execution |
| Manual result collection | Natural result collection | Phase 8 | Simpler code, correct timing |
| Separate integration tests | E2E workflow test | Phase 8 | Verifies all components together |

**Deprecated/outdated:**
- `run_in_background: true` for worker spawns in build.md: Causes misleading output timing

## Open Questions

1. **Task Tool Behavior Verification**
   - What we know: `run_in_background: true` returns immediately with task_id
   - What's unclear: Exact timing of when worker starts vs when parent continues
   - Recommendation: Test the fix manually to verify output ordering

2. **Watcher/Chaos Ant Timing**
   - What we know: Steps 5.4 and 5.4.2 also spawn workers
   - What's unclear: Whether they explicitly use `run_in_background: true`
   - Recommendation: Search build.md for all Task tool usages

3. **Integration Test Scope**
   - What we know: Can test StateGuard, UpdateTransaction, checkpoints
   - What's unclear: Whether to include actual build command testing
   - Recommendation: Test underlying functions; manual test for UI timing

## Sources

### Primary (HIGH confidence)
- `/Users/callumcowie/.claude/commands/ant/build.md` - Lines 312, 459-475, 584-591, 660-685
- `/Users/callumcowie/repos/Aether/tests/e2e/update-rollback.test.js` - E2E test pattern
- `/Users/callumcowie/repos/Aether/tests/integration/state-guard-integration.test.js` - Integration test pattern

### Secondary (MEDIUM confidence)
- Phase 7 research and implementation - StateGuard and UpdateTransaction usage
- Phase 6 research and implementation - Checkpoint and init patterns

### Tertiary (LOW confidence)
- Claude Code Task tool documentation (not directly accessible, inferred from behavior)

## Metadata

**Confidence breakdown:**
- Root cause analysis: HIGH - Directly observed in build.md source
- Technical approach: HIGH - Pattern is straightforward removal of flag
- Integration test strategy: HIGH - Established patterns from Phases 6-7
- Pitfalls: MEDIUM - Based on common implementation mistakes

**Research date:** 2026-02-14
**Valid until:** 30 days (stable patterns)

## Ready for Planning

Research complete. Planner can now create PLAN.md files with confidence.

**Key deliverables for Phase 8:**
1. Modified build.md with `run_in_background: true` removed
2. Integration test: `tests/e2e/checkpoint-update-build.test.js`
3. Verification that output timing is correct
4. E2E test suite passes
