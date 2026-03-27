---
name: worker-priming
description: Use when starting work as a spawned worker to understand what context you have been given and how to use it
type: colony
domains: [context, priming, awareness]
agent_roles: [builder, watcher, scout, chaos, oracle, architect, colonizer, route_setter, archaeologist, chronicler, keeper, tracker, probe, weaver, auditor, gatekeeper, includer, measurer, sage, ambassador]
priority: normal
version: "1.0"
---

# Worker Priming

## Purpose

When you are spawned as a worker, your prompt contains assembled context from colony-prime. Understanding what is in your context -- and what might be missing -- helps you work more effectively. This skill teaches you how to interpret your assembled context.

## The 9 Context Sections

Colony-prime assembles up to 9 sections into your prompt. Here is what each contains and how to use it:

### 1. Rolling Summary
The most important section. Contains a condensed summary of what has happened in the colony so far: completed phases, key outcomes, current state. Read this first to orient yourself.

### 2. Phase Learnings
Lessons extracted from previous phases. These are things the colony learned while building -- patterns that worked, approaches that failed, technical discoveries. Use these to avoid repeating mistakes.

### 3. Key Decisions
Major decisions made during the colony's lifetime. These include architectural choices, technology selections, and scope changes. Respect these decisions unless your task explicitly requires revisiting them.

### 4. Hive Wisdom
Cross-colony patterns from the Hive Brain. These are generalized learnings from other projects on this machine. They are guidance, not absolute rules. If hive wisdom conflicts with project-specific learnings, prefer the project-specific learning.

### 5. Context Capsule
A snapshot of the most recent session state. Contains the current phase, recent task completions, and in-progress work.

### 6. User Preferences
The user's stated preferences from QUEEN.md. These override general patterns. If a user preference says "prefer dark mode" and a hive wisdom says "use light mode", follow the user preference.

### 7. QUEEN Wisdom
Strategic guidance from the colony's QUEEN.md file. This includes project-level principles, workflow preferences, and communication style notes.

### 8. Blocker Warnings
Any active blockers that may affect your work. If a blocker is relevant to your task, address it or report that you cannot proceed because of it.

### 9. Pheromone Signals
Active FOCUS, REDIRECT, and FEEDBACK signals. See the pheromone-protocol skill for how to handle these. Always check this section before starting work.

## Missing Sections

Some sections may be absent from your context. This happens when the token budget requires trimming. The trim order is:

1. Pheromone signals (trimmed first -- lowest priority)
2. QUEEN wisdom
3. User preferences
4. Context capsule
5. Hive wisdom
6. Key decisions
7. Phase learnings
8. Rolling summary (trimmed last -- highest priority)

If a section is missing, it was trimmed for budget reasons. This is normal and not an error. Work with whatever context you have.

## Priority Rules

When context sources conflict, resolve in this order (highest priority first):

1. **Explicit task instructions** -- What you were specifically asked to do.
2. **REDIRECT signals** -- Hard constraints that override everything except the task itself.
3. **User preferences** -- The user's stated wishes.
4. **FOCUS signals** -- Areas to prioritize.
5. **Phase learnings** -- Project-specific experience.
6. **Hive wisdom** -- Cross-colony patterns.
7. **FEEDBACK signals** -- Gentle adjustments.

## Before Starting Work

Every time you begin a task:
1. Read the rolling summary to understand where the colony is.
2. Check blocker warnings for anything that affects your task.
3. Review pheromone signals for constraints and focus areas.
4. Note any missing sections and adjust expectations accordingly.
5. Begin your task with full awareness of your assembled context.
