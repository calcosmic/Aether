# Constraints -- User Guide

Constraints are how you guide the colony. Instead of micromanaging individual ants, you set focus areas and patterns to avoid. The colony reads these constraints before each build.

## How Constraints Work

- **You set** constraints using `/ant:focus` and `/ant:redirect`
- **Constraints persist** until you remove them or reset the colony
- **Focus areas** tell the colony "pay extra attention here"
- **Avoid patterns** tell the colony "don't do this"
- **Run `/ant:status`** to see active constraints

---

## FOCUS -- Guide Attention

**Command:** `/ant:focus "<area>"`

**What it does:** Adds an area to the focus list. Workers prioritize focused areas in their task execution.

### When to use FOCUS

**Scenario 1: Steering the next build phase**
You're about to run `/ant:build 3` and Phase 3 has tasks touching both the API layer and the database layer. You know the database schema is fragile:

```
/ant:focus "database schema -- handle migrations carefully"
/ant:build 3
```

**Scenario 2: Directing colonization**
You're colonizing a new project and want the colonizer to pay special attention to testing:

```
/ant:focus "test framework and coverage gaps"
/ant:colonize
```

### When NOT to use FOCUS

- Don't stack 5+ FOCUS areas -- the colony can't prioritize everything (max 5 enforced)
- Don't FOCUS on things the colony already handles (like "write good code") -- be specific

---

## REDIRECT -- Warn Away

**Command:** `/ant:redirect "<pattern to avoid>"`

**What it does:** Adds an AVOID constraint. Workers actively avoid the specified pattern. This is the strongest guidance type.

### When to use REDIRECT

**Scenario 1: Preventing a known bad approach**
Your project uses Next.js Edge Runtime, and you know `jsonwebtoken` doesn't work there:

```
/ant:redirect "Don't use jsonwebtoken -- use jose library instead (Edge Runtime compatible)"
/ant:build 2
```

**Scenario 2: Steering away from a previous failure**
Phase 1 tried to use synchronous file reads and caused performance issues:

```
/ant:redirect "No synchronous file I/O -- use async fs/promises"
```

### When NOT to use REDIRECT

- Don't REDIRECT for preferences -- use it for hard constraints ("will break" not "I don't like")
- Don't REDIRECT on vague patterns ("don't write bad code") -- be specific

---

## Storage

Constraints are stored in `.aether/data/constraints.json`:

```json
{
  "version": "1.0",
  "focus": [
    "database schema",
    "error handling"
  ],
  "constraints": [
    {
      "id": "c_1707345678123",
      "type": "AVOID",
      "content": "Don't use jsonwebtoken",
      "source": "user:redirect",
      "created_at": "2026-02-07T12:34:56Z"
    }
  ]
}
```

**Limits:**
- Max 5 focus areas (oldest removed when exceeded)
- Max 10 constraints (oldest removed when exceeded)

---

## Quick Reference

| Command | Effect | Limit |
|---------|--------|-------|
| `/ant:focus "<area>"` | Add to focus list | 5 max |
| `/ant:redirect "<avoid>"` | Add AVOID constraint | 10 max |
| `/ant:status` | View active constraints | - |

**Rule of thumb:**
- Before a build: FOCUS + REDIRECT to steer
- For hard constraints: REDIRECT
- For attention: FOCUS
