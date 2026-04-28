---
name: hypothesis-debugging
description: Use when an elusive or intermittent bug needs hypothesis-driven debugging and evidence collection
type: colony
domains: [debugging, investigation, hypothesis-testing, root-cause-analysis]
agent_roles: [tracker, probe, scout]
workflow_triggers: [build, medic]
task_keywords: [debug, hypothesis, intermittent, root cause, experiment]
priority: normal
version: "1.0"
---

# Hypothesis Debugging

## Purpose

Systematic debugging through the scientific method. When bugs laugh at your first guess and reproduce when they feel like it, this skill brings structure to the chaos. Track hypotheses, collect evidence, run controlled experiments, and converge on root causes with confidence instead of hoping the next change fixes it.

## When to Use

- A bug reproduces inconsistently or only in specific conditions
- You've tried the obvious fix and it didn't work (or made it worse)
- Multiple symptoms might be related but the connection is unclear
- The bug is in a critical path and you can't afford to guess
- You need to debug something that will span multiple sessions

## Instructions

### 1. Observe and Document the Phenomenon

Start by capturing exactly what's happening, free of interpretation:

```markdown
## Bug Observation

**Symptom:** {what the user sees -- exact error, behavior, output}
**Expected:** {what should happen instead}
**Reproduction:** {steps that trigger it, or "intermittent" with frequency estimate}
**Environment:** {browser, OS, runtime, version, any relevant config}
**First seen:** {when did this start -- after what change, deploy, or event}
**Impact:** {who's affected, how badly, is there a workaround}
```

### 2. Formulate Hypotheses

Generate 2-5 possible explanations, ranked by likelihood:

| # | Hypothesis | Likelihood | Test | Expected Evidence |
|---|-----------|------------|------|-------------------|
| H1 | {most likely explanation} | High | {what to check} | {what you'd see if true} |
| H2 | {second explanation} | Medium | {what to check} | {what you'd see if true} |
| H3 | {creative explanation} | Low | {what to check} | {what you'd see if true} |

**Hypothesis quality rules:**
- Each hypothesis must be **falsifiable** -- there must be an observation that proves it wrong
- Each hypothesis must be **testable** -- you can check it without rewriting the app
- Include at least one "unlikely but possible" hypothesis -- these sometimes win
- Rank by likelihood, not by ease of testing (easy tests can wait)

### 3. Design and Run Experiments

For each hypothesis, design the minimum experiment:

```markdown
## Experiment: {name}

**Testing:** {hypothesis ID}
**Method:** {what you'll do}
**Control:** {what you keep constant}
**Variable:** {what you change}
**Measurement:** {what you'll observe}
```

**Experiment patterns:**

| Pattern | When to Use | Example |
|---------|-------------|---------|
| **Isolation** | Complex system, narrow the blast zone | "Does it still happen if I call this function directly?" |
| **Substitution** | Suspect a specific component | "Does it work with the old version of this dependency?" |
| **Logging** | Intermittent, can't reproduce on demand | "Add timestamped logs at these 5 points" |
| **Boundary** | Suspect edge cases | "What happens with 0 items? 1 item? Max items?" |
| **Elimination** | Multiple suspects | "Comment out module A -- still broken? Then module B?" |
| **Reproduction** | Only happens in production | "Can I replicate the exact conditions locally?" |

### 4. Collect Evidence

Record what actually happened, not what you expected:

```markdown
## Evidence Log

### Experiment 1: {name}
**Hypothesis:** {which one you tested}
**Result:** {what you observed -- be precise}
**Verdict:** {supports / contradicts / inconclusive}
**New information:** {anything unexpected you learned}

### Experiment 2: {name}
{same structure}
```

### 5. Draw Conclusions

When evidence converges, document the finding:

```markdown
## Root Cause Analysis

**Root cause:** {the specific mechanism causing the bug}
**Evidence chain:** {how experiments led to this conclusion}
**Why it happened:** {the underlying reason -- design flaw, edge case, race condition, etc.}
**Why it was hard to find:** {what made it elusive}

## Fix
**Approach:** {how to fix it}
**Prevention:** {how to prevent this class of bug in the future}
**Verification:** {how to confirm the fix works}
```

### 6. Checkpoint and Continue

If debugging spans sessions, save state:

```markdown
# Debug Session Checkpoint

**Bug:** {reference}
**Status:** {in-progress / waiting-for-evidence / narrowed-down}
**Hypotheses tested:** {list with results}
**Hypotheses remaining:** {list}
**Leading theory:** {current best explanation}
**Next experiment:** {what to try next}
**Context needed:** {anything a future session needs to know}
```

## Key Patterns

### The Don't-Guess Pattern
If you catch yourself thinking "let me just try changing X," stop. That's not debugging, that's wishful thinking. Form the hypothesis first, then test it.

### The Control Variable
When testing, change exactly one thing at a time. If you change two things and the bug disappears, you've learned nothing about which one mattered.

### The Negative Result
An experiment that disproves your favorite hypothesis is not a failure -- it's progress. Eliminating possibilities is how you converge.

### The Intermittent Strategy
For bugs that don't reproduce on demand:
1. Add logging, not breakpoints (you can't step through something you can't trigger)
2. Record everything -- timestamps, state snapshots, input values
3. Look for patterns in when it happens vs. when it doesn't
4. Check clock-dependent things: race conditions, caching, TTLs, scheduled jobs

### The Explain-It Pattern
If you can't explain the bug in plain language, you don't understand it yet. Try explaining it to an imaginary colleague. If you stumble, that's where the gap is.

### The Fresh Eyes Pattern
After 4+ experiments without progress, stop and re-read the original observation. Sometimes you've drifted so far from the symptom that you're debugging a different problem.

## Output Format

- Creates `debug/{bug-slug}/` directory with investigation notes
- Maintains `debug/{bug-slug}/EVIDENCE.md` with experiment log
- Writes `debug/{bug-slug}/ROOT-CAUSE.md` when concluded
- Produces checkpoint files for session continuity

## Examples

### Example 1: Intermittent crash

```
Human: "The app crashes sometimes when switching tabs. Can't figure out when exactly."
Debugger: Let me set up a scientific approach.
Observation captured: intermittent crash on tab switch, no consistent trigger.
Hypotheses:
  H1: Race condition between tab unmount and data fetch (High likelihood)
  H2: Memory leak accumulating over time, crashes when threshold hit (Medium)
  H3: Specific data payload in one tab causes render error (Low)
Running Experiment 1: Add timing logs to tab switch lifecycle...
```

### Example 2: Persistent state corruption

```
Human: "User settings randomly reset to defaults. Happens about once a week."
Debugger: Capturing observation: settings reset to defaults, weekly frequency, no obvious trigger.
Hypotheses:
  H1: localStorage cleared by browser storage pressure (High)
  H2: Settings fetch returns empty during network timeout, overwrites local (Medium)
  H3: Auth token refresh clears storage (Low)
Experiment: Add interceptor to log all localStorage writes and settings API calls...
[after evidence collection]
Root cause: H2 confirmed -- settings API has a 3-second timeout that returns empty object, which overwrites localStorage. Fix: don't overwrite on empty response.
```

### Example 3: Checkpoint for continuation

```
Debugger: Saving checkpoint -- leaving off at experiment 3 of 5.
H1 (race condition) contradicted -- added synchronization, still crashes.
H2 (memory leak) inconclusive -- memory usage is flat.
H3 (data payload) testing next -- checking if crash correlates with specific tab data.
Next session: run data-payload boundary test on the three tabs that have custom widgets.
Checkpoint saved to debug/tab-crash/CHECKPOINT.md
```
