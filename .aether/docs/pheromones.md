# Pheromone Signals -- User Guide

Pheromones are how you communicate with the colony. Instead of micromanaging individual ants, you emit chemical signals that influence their behavior. Signals have a TTL (time-to-live) and priority level. By default, signals last until the current phase completes.

## How Pheromones Work

- **You emit** signals using `/ant:focus`, `/ant:redirect`, `/ant:feedback`
- **The colony also emits** signals automatically after builds (FEEDBACK after every phase, REDIRECT when error patterns recur)
- **Signals expire** based on their `expires_at` field -- default is `"phase_end"` (lasts until phase completes)
- **Optional TTL** -- use `--ttl` flag for wall-clock expiration (e.g., `--ttl 2h` for 2 hours)
- **Priority levels** determine worker attention: high (REDIRECT), normal (FOCUS), low (FEEDBACK)
- **Expired signals** are filtered on read -- no cleanup process needed

Run `/ant:status` at any time to see all active pheromones.

---

## FOCUS -- Guide Attention

**Command:** `/ant:focus "<area>"`
**Priority:** normal | **Default expiration:** end of phase

**What it does:** Tells the colony "pay extra attention here." Workers read FOCUS signals and weight this area higher in their task execution.

### When to use FOCUS

**Scenario 1: Steering the next build phase**
You're about to run `/ant:build 3` and Phase 3 has tasks touching both the API layer and the database layer. You know the database schema is fragile:

```
/ant:focus "database schema -- handle migrations carefully"
/ant:build 3
```

**Scenario 2: Time-limited focus**
You want attention on auth for the next 2 hours, then let it fade:

```
/ant:focus "auth middleware correctness" --ttl 2h
```

**Scenario 3: Directing colonization**
You're colonizing a new project and want the colonizer to pay special attention to testing:

```
/ant:focus "test framework and coverage gaps"
/ant:colonize
```

### When NOT to use FOCUS

- Don't stack 5+ FOCUS signals -- the colony can't prioritize everything
- Don't FOCUS on things the colony already handles (like "write good code") -- be specific
- Don't FOCUS after a phase completes if you're about to `/clear` context -- emit it fresh before the next build

---

## REDIRECT -- Warn Away

**Command:** `/ant:redirect "<pattern to avoid>"`
**Priority:** high | **Default expiration:** end of phase

**What it does:** Acts as a hard constraint. Workers with high priority signal awareness will actively avoid the specified pattern. This is the strongest signal type.

### When to use REDIRECT

**Scenario 1: Preventing a known bad approach**
Your project uses Next.js Edge Runtime, and you know `jsonwebtoken` doesn't work there:

```
/ant:redirect "Don't use jsonwebtoken -- use jose library instead (Edge Runtime compatible)"
/ant:build 2
```

**Scenario 2: Long-lived constraint**
You want to enforce a constraint across multiple phases (24 hours):

```
/ant:redirect "No global mutable state -- use request-scoped context" --ttl 1d
```

**Scenario 3: Steering away from a previous failure**
Phase 1 tried to use synchronous file reads and caused performance issues:

```
/ant:redirect "No synchronous file I/O -- use async fs/promises"
```

### When NOT to use REDIRECT

- Don't REDIRECT for preferences -- use it for hard constraints ("will break" not "I don't like")
- Don't REDIRECT on vague patterns ("don't write bad code") -- be specific
- Don't use REDIRECT when FOCUS would suffice

---

## FEEDBACK -- Adjust Course

**Command:** `/ant:feedback "<observation>"`
**Priority:** low | **Default expiration:** end of phase

**What it does:** Provides gentle course correction. Unlike FOCUS (attention) or REDIRECT (avoidance), FEEDBACK adjusts the colony's approach based on your observations.

### When to use FEEDBACK

**Scenario 1: Mid-project course correction**
After building Phase 2, you notice the code is over-engineered:

```
/ant:feedback "Code is too abstract -- prefer simple, direct implementations over clever abstractions"
```

**Scenario 2: Positive reinforcement**
Phase 3 produced clean, well-tested code. You want more of the same:

```
/ant:feedback "Great test coverage in Phase 3 -- maintain this level of testing"
```

**Scenario 3: Quality emphasis shift**
You're noticing the code lacks error handling:

```
/ant:feedback "Need more error handling -- happy path works but edge cases are unhandled"
```

### When NOT to use FEEDBACK

- Don't use FEEDBACK for hard constraints -- that's REDIRECT's job
- Don't use FEEDBACK before the colony has produced anything
- Don't emit multiple conflicting FEEDBACK signals

---

## Auto-Emitted Pheromones

The colony emits pheromones automatically during builds. You don't need to manage these:

- **FEEDBACK after every phase:** build.md (Step 7b) emits a FEEDBACK pheromone summarizing what worked and what failed
- **REDIRECT on error patterns:** If errors.json has recurring flagged patterns, build.md and continue.md auto-emit REDIRECT signals
- **FEEDBACK from global learnings:** When colonizing a new project, colonize.md injects relevant global learnings as FEEDBACK pheromones

Auto-emitted signals have their `source` field set to indicate origin: `"worker:builder"`, `"worker:continue"`, or `"global:inject"`.

---

## Signal Combinations

Pheromones combine. Workers check all active signals and respond accordingly:

| Combination | Effect |
|-------------|--------|
| FOCUS + FEEDBACK | Workers concentrate on the focused area and adjust approach based on feedback |
| FOCUS + REDIRECT | Workers prioritize the focused area while avoiding the redirected pattern |
| FEEDBACK + REDIRECT | Workers adjust approach (feedback) and avoid specific patterns (redirect) |
| All three | Full steering: attention (FOCUS), avoidance (REDIRECT), and adjustment (FEEDBACK) |

**Priority processing:** Workers check high priority signals first, then normal, then low.

---

## TTL Options

By default, signals expire at phase end (`expires_at: "phase_end"`). Use `--ttl` flag for wall-clock expiration:

```
/ant:focus "database schema"              # expires at phase end
/ant:focus "API layer" --ttl 2h           # expires in 2 hours
/ant:redirect "No JWT" --ttl 1d           # expires in 1 day
/ant:feedback "keep tests simple" --ttl 30m  # expires in 30 minutes
```

**Duration format:** `<number><unit>` where unit is:
- `m` = minutes (e.g., `30m`)
- `h` = hours (e.g., `2h`)
- `d` = days (e.g., `1d`)

---

## Pause-Aware TTL

When the colony is paused (`paused_at` timestamp recorded):

- **Wall-clock TTLs** are extended by the pause duration on resume
- **Phase-scoped signals** (`expires_at: "phase_end"`) are unaffected by pause

This ensures signals don't expire while you're away from the project.

---

## Viewing Active Pheromones

**Command:** `/ant:pheromones [filter]`

Displays all active signals in a formatted table with:
- Signal type (FOCUS/REDIRECT/FEEDBACK)
- Current strength percentage (accounting for decay)
- Elapsed time since creation
- Remaining time before expiration

**Filters:**
- `/ant:pheromones` — Show all active signals
- `/ant:pheromones focus` — Show only FOCUS signals
- `/ant:pheromones redirect` — Show only REDIRECT signals
- `/ant:pheromones feedback` — Show only FEEDBACK signals
- `/ant:pheromones clear` — Clear expired/inactive signals

---

## Automatic Suggestions (v6.0)

The colony can analyze your codebase and suggest pheromones worth capturing.

**During builds:** At Step 4.2, the colony automatically analyzes the codebase and proposes relevant signals:
- Detects hardcoded values that should be configurable
- Finds error-prone patterns from the midden
- Identifies architectural constraints from directory structure
- Suggests FOCUS areas based on file modification patterns
- Proposes REDIRECT signals for anti-patterns

**Tick-to-approve UI:** When suggestions are found, the build pauses to show:

```
🎯 Proposed Pheromone 1 of 3

Type: FOCUS
Content: "Audio processing — maintain thread safety"
Confidence: High (3 similar patterns found)

[A]pprove  [R]eject  [S]kip  [D]ismiss All
```

**Actions:**
- **A**pprove — Creates the pheromone signal
- **R**eject — Discards this suggestion (logs reason)
- **S**kip — Defer decision (shown again next build)
- **D**ismiss All — Skip all remaining suggestions

---

## Signal Decay

Pheromones lose strength over time (natural decay):

| Signal Type | Half-life | Full decay |
|-------------|-----------|------------|
| FOCUS | 15 days | 30 days |
| REDIRECT | 30 days | 60 days |
| FEEDBACK | 45 days | 90 days |

Signals below 10% strength are considered inactive but remain in history.

---

## Quick Reference

| Signal | Command | Priority | Default Expiration | Use for |
|--------|---------|----------|-------------------|---------|
| FOCUS | `/ant:focus "<area>"` | normal | phase end | "Pay attention to this" |
| REDIRECT | `/ant:redirect "<avoid>"` | high | phase end | "Don't do this" |
| FEEDBACK | `/ant:feedback "<note>"` | low | phase end | "Adjust based on this" |

**Rule of thumb:**
- Before a build: FOCUS + REDIRECT to steer
- After a build: FEEDBACK to adjust
- For hard constraints: REDIRECT (high priority)
- For gentle nudges: FEEDBACK (low priority)
- For attention: FOCUS (normal priority)
- For short-lived signals: use `--ttl` flag
- To see current signals: `/ant:pheromones`
- To clear expired: `/ant:pheromones clear`
