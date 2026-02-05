# Pheromone Signals -- User Guide

Pheromones are how you communicate with the colony. Instead of micromanaging individual ants, you emit chemical signals that influence their behavior based on each caste's sensitivity. Signals decay over time, so the colony naturally returns to neutral behavior.

## How Pheromones Work

- **You emit** signals using `/ant:focus`, `/ant:redirect`, `/ant:feedback`
- **The colony also emits** signals automatically after builds (FEEDBACK after every phase, REDIRECT when error patterns recur)
- **Signals decay** exponentially -- a FOCUS signal at strength 0.7 with a 1-hour half-life will be at ~0.35 after an hour
- **Each caste responds differently** -- builders are highly sensitive to FOCUS (0.9), while architects barely notice it (0.4)

Run `/ant:status` at any time to see all active pheromones and their current strength.

## FOCUS -- Guide Attention

**Command:** `/ant:focus "<area>"`
**Strength:** 0.7 | **Half-life:** 1 hour | **Decays to noise in:** ~3-4 hours

**What it does:** Tells the colony "pay extra attention here." Workers with high FOCUS sensitivity will prioritize this area in their next task.

**Highest sensitivity:** Builder (0.9), Scout (0.9), Watcher (0.8)
**Lowest sensitivity:** Architect (0.4), Route-setter (0.5)

### When to use FOCUS

**Scenario 1: Steering the next build phase**
You're about to run `/ant:build 3` and Phase 3 has tasks touching both the API layer and the database layer. You know the database schema is fragile and needs careful attention:

```
/ant:focus "database schema -- handle migrations carefully"
/ant:build 3
```

The builder-ant (sensitivity 0.9) will receive an effective signal of 0.63, crossing the PRIORITIZE threshold (>0.5). It will weight database-related tasks with extra care and thoroughness.

**Scenario 2: Post-build quality concern**
The watcher scored Phase 2 at 6/10 and flagged auth middleware issues. Before continuing to Phase 3:

```
/ant:focus "auth middleware correctness"
/ant:continue
```

The next phase's watcher (sensitivity 0.8) will scrutinize auth middleware more carefully during verification.

**Scenario 3: Directing colonization**
You're colonizing a new project and want the colonizer to pay special attention to the testing infrastructure:

```
/ant:focus "test framework and coverage gaps"
/ant:colonize
```

The colonizer (sensitivity 0.7) will weight test infrastructure discovery higher in its analysis.

### When NOT to use FOCUS

- Don't stack 5+ FOCUS signals -- the colony can't prioritize everything (that's prioritizing nothing)
- Don't FOCUS on things the colony already handles (like "write good code") -- be specific
- Don't FOCUS after a phase completes if you're about to `/clear` context -- the signal persists in pheromones.json, but emit it fresh before the next build for strongest effect

---

## REDIRECT -- Warn Away

**Command:** `/ant:redirect "<pattern to avoid>"`
**Strength:** 0.9 | **Half-life:** 24 hours | **Decays to noise in:** ~3-4 days

**What it does:** Acts as a hard constraint. Workers with high REDIRECT sensitivity will actively avoid the specified pattern. This is the strongest, longest-lasting signal.

**Highest sensitivity:** Builder (0.9), Route-setter (0.8)
**Lowest sensitivity:** Architect (0.3), Colonizer (0.3)

### When to use REDIRECT

**Scenario 1: Preventing a known bad approach**
Your project uses Next.js Edge Runtime, and you know `jsonwebtoken` doesn't work there (CommonJS issues). Before the colony builds auth:

```
/ant:redirect "Don't use jsonwebtoken -- use jose library instead (Edge Runtime compatible)"
/ant:build 2
```

The builder (sensitivity 0.9) receives effective signal 0.81 -- deep in PRIORITIZE territory. It will actively avoid jsonwebtoken and use jose instead.

**Scenario 2: Steering away from a previous failure**
Phase 1 tried to use synchronous file reads and caused performance issues (you saw this in the watcher report). Before Phase 2:

```
/ant:redirect "No synchronous file I/O -- use async fs/promises"
```

The route-setter (sensitivity 0.8) will exclude synchronous patterns from the task plan. The builder will avoid them in implementation.

**Scenario 3: Architectural constraint**
You want the colony to avoid global mutable state because the project might need server-side rendering:

```
/ant:redirect "No global mutable state -- use request-scoped context"
```

This signal persists for 24 hours (half-life), covering multiple build phases without re-emission.

### When NOT to use REDIRECT

- Don't REDIRECT for preferences -- use it for hard constraints ("will break" not "I don't like")
- Don't REDIRECT on vague patterns ("don't write bad code") -- be specific about the approach to avoid and why
- Don't use REDIRECT when FOCUS would suffice -- if you want more attention on testing, FOCUS on testing rather than REDIRECT away from skipping tests

---

## FEEDBACK -- Adjust Course

**Command:** `/ant:feedback "<observation>"`
**Strength:** 0.5 | **Half-life:** 6 hours | **Decays to noise in:** ~20 hours

**What it does:** Provides gentle course correction. Unlike FOCUS (attention) or REDIRECT (avoidance), FEEDBACK adjusts the colony's approach based on your observations. The watcher is most sensitive to this signal.

**Highest sensitivity:** Watcher (0.9), Builder (0.7), Route-setter (0.7)
**Lowest sensitivity:** Architect (0.6), Colonizer (0.5), Scout (0.5)

### When to use FEEDBACK

**Scenario 1: Mid-project course correction**
After building Phase 2, you review the output and notice the code is over-engineered -- too many abstractions for a simple feature:

```
/ant:feedback "Code is too abstract -- prefer simple, direct implementations over clever abstractions"
```

The builder (0.7) and watcher (0.9) both pick this up. The builder simplifies its approach in Phase 3. The watcher (effective signal 0.45 -- NOTE range) factors simplicity into its quality assessment.

**Scenario 2: Positive reinforcement**
Phase 3 produced clean, well-tested code. You want more of the same:

```
/ant:feedback "Great test coverage in Phase 3 -- maintain this level of testing"
```

The colony records this as a positive signal. The builder continues the pattern; the watcher validates against it.

**Scenario 3: Quality emphasis shift**
The watcher is scoring phases highly but you're noticing the code lacks error handling:

```
/ant:feedback "Need more error handling -- happy path works but edge cases are unhandled"
```

The watcher (sensitivity 0.9, effective 0.45) will increase scrutiny on error handling in its next verification pass.

### When NOT to use FEEDBACK

- Don't use FEEDBACK for hard constraints -- that's REDIRECT's job
- Don't use FEEDBACK before the colony has produced anything -- it responds to observations, not predictions
- Don't emit multiple conflicting FEEDBACK signals ("more abstraction" + "keep it simple") -- the colony can't resolve contradictions

---

## Auto-Emitted Pheromones

The colony emits pheromones automatically during builds. You don't need to manage these:

- **FEEDBACK after every phase:** build.md (Step 7b) emits a FEEDBACK pheromone summarizing what worked and what failed. Source: `auto:build`
- **REDIRECT on error patterns:** If errors.json has recurring flagged patterns, build.md and continue.md auto-emit REDIRECT signals to steer future phases away from the problematic pattern. Source: `auto:build` or `auto:continue`
- **FEEDBACK from global learnings:** When colonizing a new project, colonize.md injects relevant global learnings as FEEDBACK pheromones. Source: `global:inject`

Auto-emitted signals have `"auto": true` in pheromones.json and are visible in `/ant:status`.

---

## Signal Combinations

Pheromones combine. Each worker's spec defines combination effects:

| Combination | Effect |
|-------------|--------|
| FOCUS + FEEDBACK | Workers concentrate on the focused area and adjust approach based on feedback |
| FOCUS + REDIRECT | Workers prioritize the focused area while avoiding the redirected pattern |
| FEEDBACK + REDIRECT | Workers adjust approach (feedback) and avoid specific patterns (redirect) |
| All three | Full steering: attention (FOCUS), avoidance (REDIRECT), and adjustment (FEEDBACK) |

**Effective signal formula:** `caste_sensitivity x current_strength`

Thresholds:
- **> 0.5 PRIORITIZE** -- worker restructures behavior around this signal
- **0.3-0.5 NOTE** -- worker is aware, factors into decisions
- **< 0.3 IGNORE** -- signal too weak to act on

---

## Quick Reference

| Signal | Command | Strength | Half-life | Use for |
|--------|---------|----------|-----------|---------|
| FOCUS | `/ant:focus "<area>"` | 0.7 | 1 hour | "Pay attention to this" |
| REDIRECT | `/ant:redirect "<avoid>"` | 0.9 | 24 hours | "Don't do this" |
| FEEDBACK | `/ant:feedback "<note>"` | 0.5 | 6 hours | "Adjust based on this" |

| | FOCUS | REDIRECT | FEEDBACK |
|---|---|---|---|
| Builder | 0.9 | 0.9 | 0.7 |
| Scout | 0.9 | 0.4 | 0.5 |
| Watcher | 0.8 | 0.5 | 0.9 |
| Colonizer | 0.7 | 0.3 | 0.5 |
| Route-setter | 0.5 | 0.8 | 0.7 |
| Architect | 0.4 | 0.3 | 0.6 |

**Rule of thumb:**
- Before a build: FOCUS + REDIRECT to steer
- After a build: FEEDBACK to adjust
- For hard constraints: REDIRECT (strongest signal, longest decay)
- For gentle nudges: FEEDBACK (moderate signal, medium decay)
- For attention: FOCUS (moderate signal, short decay -- re-emit if needed)
