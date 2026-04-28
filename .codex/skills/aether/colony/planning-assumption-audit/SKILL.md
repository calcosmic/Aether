---
name: planning-assumption-audit
description: Use before planning to surface hidden assumptions that could cause expensive wrong turns
type: colony
domains: [assumptions, alignment, planning, risk-identification]
agent_roles: [oracle, architect, scout, route_setter]
workflow_triggers: [discuss, plan]
task_keywords: [assumption, ambiguity, implicit, risk, unclear]
priority: normal
version: "1.0"
---

# Planning Assumption Audit

## Purpose

Surface the invisible decisions hiding inside an approach before they become expensive mistakes. When someone describes what to build, they're also implicitly deciding how to build it -- technology choices, scope boundaries, priority orderings, and user model assumptions they may not realize they're making. This skill drags those assumptions into the light so they can be confirmed, corrected, or consciously accepted.

## When to Use

- Before planning a phase, especially if the phase description is brief or high-level
- When transitioning from "what to build" to "how to build it"
- When multiple people have different mental models of the same feature
- After a discuss-phase when the conversation revealed ambiguity
- When you catch yourself thinking "I'll just assume..." about something important

## Instructions

### 1. Read the Phase Description Carefully

Absorb what's written. Then notice what's not written -- the gaps, the hand-waves, the "obviously we'll just..." moments. Those gaps are where assumptions live.

### 2. Identify Assumption Categories

Scan for assumptions across these dimensions:

| Category | What to Look For | Risk If Wrong |
|----------|-----------------|---------------|
| **Technology** | "We'll use X" without explicit choice | Wrong tool for the job, rework |
| **Scope** | "Just the basics" without definition | Scope creep or under-delivery |
| **User Model** | "Users will understand that..." | UX that confuses real users |
| **Data** | "The data looks like..." without seeing it | Schema mismatches, missing fields |
| **Integration** | "It connects to..." without API docs | Blocked by incompatible interfaces |
| **Performance** | "It'll be fast enough" without measurement | Slow in production |
| **Priority** | "This part first" without explicit ranking | Wrong order, dependent work blocked |
| **Security** | "We'll secure it later" | Vulnerabilities shipped |
| **Migration** | "We'll migrate the old data" without a plan | Data loss or downtime |
| **Team** | "Someone will handle..." without assignment | Dropped balls |

### 3. Surface Each Assumption

For each assumption found, write it in this format:

```markdown
### ASSUMPTION-{id}: {title}

**The assumption:** {what we're assuming, stated plainly}
**Why it's hidden:** {why this feels "obvious" enough to not mention}
**If we're right:** {what goes well}
**If we're wrong:** {what breaks, how badly}
**How to verify:** {one concrete check that confirms or denies this}
**Confidence:** {high | medium | low | wild-guess}
```

### 4. Categorize by Risk

Sort assumptions by the combination of confidence and impact:

| Zone | Confidence | Impact if Wrong | Action |
|------|-----------|-----------------|--------|
| **Verify First** | Low | High | Must resolve before planning |
| **Flag for Discussion** | Medium | High | Raise with decision-maker before building |
| **Document and Proceed** | Low | Low | Note it, build, validate later |
| **Safe to Assume** | High | Any | Proceed, revisit if signals change |

### 5. Produce ASSUMPTIONS.md

```markdown
# Assumptions: Phase {N} -- {name}

**Surfaced:** {date}
**Phase description:** {original brief}

## Summary
{count} assumptions identified. {critical} need verification before planning.

## Must Verify Before Planning
{assumptions in the "Verify First" zone}

## Flag for Discussion
{assumptions in the "Flag for Discussion" zone}

## Documented Assumptions
{assumptions in "Document and Proceed" and "Safe to Assume" zones}

## Assumptions Not Made
{explicitly list things you considered assuming but decided not to -- this prevents silent scope creep}

---
*Surfaced by assumption-surfacers skill. Review and confirm or correct before planning.*
```

### 6. Route the Outcome

| Outcome | Route |
|---------|-------|
| All assumptions confirmed or low-risk | Proceed to planning |
| 1-3 assumptions need verification | Run targeted spikes for those items |
| Fundamental disagreement on approach | Return to discuss-phase for realignment |
| Missing critical information | Route to research-isolator |

## Key Patterns

### The "Obviously" Trap
Every time you write "obviously" or "of course," stop. That's an assumption wearing a disguise. Ask: "Obvious to whom? Would a new team member find it obvious?"

### The Negation Test
For each assumption, try negating it. If the negated version is also plausible, the assumption is worth surfacing. Example: "Users will search by name" -- negated: "Users will search by category." Both plausible? Surface it.

### The Fresh Eyes Pattern
Imagine explaining the approach to someone who's never seen this project. What would they ask? Those questions reveal your blind spots.

### The Dependency Chain
One assumption often depends on another. Map the chain: "We'll use PostgreSQL (A) -> assumes we need relational data (B) -> assumes data has consistent schema (C)." If C is wrong, A and B collapse too. Surface the deepest assumption in the chain.

### The Boring Assumption Pattern
The most dangerous assumptions are the boring ones -- things so mundane they don't seem worth mentioning. "We'll deploy to the same server" or "the API will stay on v2." Boring assumptions cause silent failures.

## Output Format

- Writes `ASSUMPTIONS.md` to the phase's planning directory
- Returns summary with risk-prioritized list to conversation
- Suggests specific next actions for high-risk assumptions

## Examples

### Example 1: Before planning a notification system

```markdown
### ASSUMPTION-001: Push notifications only

**The assumption:** Users want push notifications (mobile/desktop), not in-app or email.
**Why it's hidden:** "Notifications" typically means push in casual conversation.
**If we're right:** We build one notification channel and ship faster.
**If we're wrong:** Users ignore push, want an in-app bell icon. Rework to add second channel.
**How to verify:** Check if target users have push enabled on their devices; ask 3 users what "notification" means to them.
**Confidence:** medium
```

### Example 2: Before planning a data migration

```markdown
### ASSUMPTION-004: Schema is consistent across all legacy records

**The assumption:** All 50k records in the legacy system follow the documented schema.
**Why it's hidden:** The docs show a clean schema, and we haven't looked at actual data.
**If we're right:** Straightforward migration, estimate holds.
**If we're wrong:** Data cleaning phase needed, migration timeline doubles.
**How to verify:** Sample 200 random records and check against schema.
**Confidence:** low
```

### Example 3: Summary output

```
Surfaced 8 assumptions for Phase 5 "Real-time Dashboard":
- 2 MUST VERIFY (data format assumption, WebSocket compatibility)
- 3 FLAG FOR DISCUSSION (refresh strategy, mobile priority, permission model)
- 3 DOCUMENTED (node version, timezone handling, chart library choice)

Recommendation: Run spikes for the 2 verify items before planning. Schedule 15-minute discussion on the 3 flagged items.
ASSUMPTIONS.md written to planning directory.
```
