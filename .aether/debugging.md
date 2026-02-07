# Systematic Debugging Discipline

## The Iron Law

```
NO FIXES WITHOUT ROOT CAUSE INVESTIGATION FIRST
```

If you haven't completed Phase 1, you cannot propose fixes. Symptom fixes are failure.

## When to Use

Use for ANY technical issue encountered during colony work:
- Test failures
- Build errors
- Unexpected behavior
- Performance problems
- Integration issues

**Use ESPECIALLY when:**
- Under time pressure (emergencies make guessing tempting)
- "Just one quick fix" seems obvious
- You've already tried multiple fixes
- Previous fix didn't work

## The Four Phases

Complete each phase before proceeding to the next.

### Phase 1: Root Cause Investigation

**BEFORE attempting ANY fix:**

1. **Read Error Messages Carefully**
   - Don't skip past errors or warnings
   - Read stack traces completely
   - Note line numbers, file paths, error codes

2. **Reproduce Consistently**
   - Can you trigger it reliably?
   - What are the exact steps?
   - If not reproducible → gather more data, don't guess

3. **Check Recent Changes**
   - What changed that could cause this?
   - Git diff, recent commits
   - New dependencies, config changes

4. **Gather Evidence in Multi-Component Systems**
   ```
   For EACH component boundary:
     - Log what data enters component
     - Log what data exits component
     - Verify environment/config propagation

   Run once to gather evidence showing WHERE it breaks
   THEN analyze to identify failing component
   ```

5. **Trace Data Flow (Root Cause Tracing)**

   When error is deep in call stack:
   - Where does bad value originate?
   - What called this with bad value?
   - Keep tracing UP until you find the source
   - Fix at source, not at symptom

   ```
   Symptom → Immediate cause → What called this? → Keep tracing → Original trigger → FIX HERE
   ```

### Phase 2: Pattern Analysis

1. **Find Working Examples**
   - Locate similar working code in same codebase
   - What works that's similar to what's broken?

2. **Compare Against References**
   - Read reference implementation COMPLETELY
   - Don't skim - understand fully

3. **Identify Differences**
   - What's different between working and broken?
   - List every difference, however small

### Phase 3: Hypothesis and Testing

1. **Form Single Hypothesis**
   - State clearly: "I think X is the root cause because Y"
   - Be specific, not vague

2. **Test Minimally**
   - Make SMALLEST possible change to test hypothesis
   - One variable at a time
   - Don't fix multiple things at once

3. **Verify Before Continuing**
   - Did it work? → Phase 4
   - Didn't work? → Form NEW hypothesis
   - DON'T add more fixes on top

### Phase 4: Implementation

1. **Create Failing Test Case**
   - Simplest possible reproduction
   - MUST have before fixing

2. **Implement Single Fix**
   - Address root cause identified
   - ONE change at a time
   - No "while I'm here" improvements

3. **Verify Fix**
   - Test passes now?
   - No other tests broken?
   - Issue actually resolved?

4. **If Fix Doesn't Work**
   - Count: How many fixes have you tried?
   - If < 3: Return to Phase 1, re-analyze
   - **If ≥ 3: STOP - question the architecture**

## The 3-Fix Rule

**If 3+ fixes have failed:**

Pattern indicating architectural problem:
- Each fix reveals new problem in different place
- Fixes require "massive refactoring"
- Each fix creates new symptoms elsewhere

**STOP and question fundamentals:**
- Is this pattern fundamentally sound?
- Should we refactor architecture vs. continue fixing symptoms?

Report to parent/Queen with architectural concern before attempting more fixes.

## Red Flags - STOP and Follow Process

If you catch yourself thinking:
- "Quick fix for now, investigate later"
- "Just try changing X and see if it works"
- "Add multiple changes, run tests"
- "It's probably X, let me fix that"
- "I don't fully understand but this might work"
- "One more fix attempt" (when already tried 2+)

**ALL of these mean: STOP. Return to Phase 1.**

## Common Rationalizations

| Excuse | Reality |
|--------|---------|
| "Issue is simple, don't need process" | Simple issues have root causes too |
| "Emergency, no time for process" | Systematic is FASTER than thrashing |
| "Just try this first, then investigate" | First fix sets the pattern. Do it right. |
| "Multiple fixes at once saves time" | Can't isolate what worked. Causes new bugs. |
| "I see the problem, let me fix it" | Seeing symptoms ≠ understanding root cause |

## Quick Reference

| Phase | Key Activities | Exit Criteria |
|-------|---------------|---------------|
| **1. Root Cause** | Read errors, reproduce, trace data flow | Understand WHAT and WHY |
| **2. Pattern** | Find working examples, compare | Identify differences |
| **3. Hypothesis** | Form theory, test minimally | Confirmed or new hypothesis |
| **4. Implementation** | Create test, fix, verify | Bug resolved, tests pass |

## Debugging Report Format

When reporting debugging work:

```
🔍 Debug Report
===============

Issue: {what broke}

Phase 1 - Root Cause:
  Error: {exact error message}
  Reproduced: {yes/no, steps}
  Trace: {call chain to source}
  Root cause: {the actual source}

Phase 2 - Pattern:
  Working example: {reference found}
  Key difference: {what differs}

Phase 3 - Hypothesis:
  Theory: {X causes Y because Z}
  Test: {minimal change made}
  Result: {confirmed/refuted}

Phase 4 - Fix:
  Change: {what was changed}
  Test: {failing test created}
  Verified: {tests pass, issue resolved}

Fix count: {N}/3 max
```

## Real-World Impact

- Systematic approach: 15-30 minutes to fix
- Random fixes approach: 2-3 hours of thrashing
- First-time fix rate: 95% vs 40%
- New bugs introduced: Near zero vs common
