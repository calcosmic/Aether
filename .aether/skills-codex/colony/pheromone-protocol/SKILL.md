---
name: pheromone-protocol
description: Use when pheromone signals are present in your assembled context and you need to interpret and comply with them
type: colony
domains: [pheromones, protocol, signals, compliance]
agent_roles: [builder, watcher, scout, chaos, oracle, architect, colonizer, route_setter, archaeologist, chronicler, keeper, tracker, probe, weaver, auditor, gatekeeper, includer, measurer, sage, ambassador]
priority: normal
version: "1.0"
---

# Pheromone Protocol

## Purpose

Pheromone signals are the colony's steering mechanism. Every agent must know how to read, interpret, and comply with them. This skill standardizes signal handling across all 22 agent roles.

## Signal Types and Required Response

### REDIRECT -- Hard Constraint

REDIRECT signals are non-negotiable. You MUST comply.

- If a REDIRECT says "avoid pattern X", you must not use pattern X in any code you write, any recommendation you make, or any approach you take.
- REDIRECTs marked `[error-pattern]` come from repeated colony failures (midden threshold). These are lessons learned the hard way -- treat them with extra seriousness.
- If complying with a REDIRECT conflicts with your task, report the conflict as a blocker rather than violating the REDIRECT.
- In your output, explicitly acknowledge each REDIRECT and state how you complied.

### FOCUS -- Prioritize This Area

FOCUS signals direct your attention. You SHOULD prioritize.

- When choosing between approaches, prefer the one aligned with active FOCUS signals.
- FOCUS areas receive extra effort: more thorough analysis, additional test coverage, deeper investigation.
- If your task touches a FOCUS area, give it proportionally more attention than non-focused areas.
- You may deprioritize (but not ignore) non-focused areas to spend more time on focused ones.

### FEEDBACK -- Preference Adjustment

FEEDBACK signals are calibration from past experience. You may incorporate flexibly.

- Use FEEDBACK to adjust patterns: coding style, naming conventions, architectural preferences.
- You may deviate from FEEDBACK with good reason, but note the deviation in your output.
- FEEDBACK accumulates -- multiple FEEDBACK signals pointing the same direction carry more weight than a single one.

## Signal Conflict and History Awareness

When multiple signals overlap, preserve the strongest actionable guidance:

- A REDIRECT overrides conflicting FOCUS or FEEDBACK.
- Recent reinforced signals carry more weight than old weak signals.
- If two active signals conflict at the same priority, surface the conflict instead of silently choosing one.
- When a signal changes worker behavior, mention that influence in the result summary.

## Role-Specific Adaptations

Different roles adapt signal handling to their function:

| Role | REDIRECT Adaptation | FOCUS Adaptation |
|------|-------------------|-----------------|
| Builder | Avoid flagged patterns in code | Write extra tests for focused areas |
| Watcher | Flag violations in review | Deeper checks on focused areas |
| Scout | Exclude flagged approaches from research | Prioritize focused topics |
| Chaos | Test flagged patterns for regression | Focus chaos testing on focused areas |
| Route Setter | Exclude flagged approaches from plans | Weight focused areas in phase design |
| Architect | Reject designs using flagged patterns | Center architecture around focused areas |

All other roles follow the general protocol above.

## Compliance Reporting

In your output summary, include a signal compliance section:

```
Signal Compliance:
- REDIRECT "no inline styles": Complied -- used Tailwind utility classes throughout
- FOCUS "security": Applied -- added input validation to all endpoints
- FEEDBACK "prefer composition": Followed -- used composition pattern for new components
```

If no signals were present in your context, note: "No active pheromone signals."

## Signal Absence

If your context contains no pheromone signals section, this means either:
- No signals are currently active (normal for new colonies).
- The signals section was trimmed for token budget (normal under compact mode).

In either case, proceed with your task using default judgment. Do not treat signal absence as an error.
