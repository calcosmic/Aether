---
name: build-discipline
description: Use when implementing code, writing tests, or executing build tasks as a builder worker
type: colony
domains: [building, testing, quality, implementation]
agent_roles: [builder]
priority: normal
version: "1.0"
---

# Build Discipline

## Purpose

Builder workers must follow a disciplined implementation process. This skill ensures consistent quality, avoids repeating known failures, and keeps work aligned with the phase plan.

## Pre-Build Checklist

Before writing any code, complete these checks in order:

### 1. Check the Midden

Query recent failures using `midden-recent-failures`. Read what went wrong in previous builds. The midden exists to prevent you from repeating the same mistakes.

- If the midden contains failures related to your current task, adjust your approach before starting.
- If a failure has been acknowledged, note the resolution and follow it.
- Never ignore midden entries -- they represent real colony learning.

### 2. Check Active Pheromone Signals

Read all injected REDIRECT signals. These are hard constraints -- you must not violate them under any circumstances. Examples:
- "No inline styles" means you must use CSS classes or utility frameworks.
- "Avoid raw SQL" means you must use an ORM or query builder.

Read all FOCUS signals. These indicate where to direct extra attention:
- A FOCUS on "security" means add input validation, check for injection, review auth flows.
- A FOCUS on "testing" means write more comprehensive test cases than usual.

Read all FEEDBACK signals. These are preferences to incorporate naturally:
- "Prefer functional components" means choose functions over classes when both work.

### 3. Review the Phase Plan

Read the current phase description and task list. Understand exactly what you are supposed to build. Do not add features not in the plan. Do not refactor code outside your task scope. Stay focused.

## Implementation Process

### Write Tests First

Follow TDD discipline where the task permits:
1. Write a failing test that describes the expected behavior.
2. Run the test -- confirm it fails for the right reason.
3. Write the minimal code to make the test pass.
4. Refactor while keeping tests green.

If TDD is not practical for the task (e.g., configuration changes, documentation), note why in your output.

### Log Failures Before Changing Approach

If your implementation fails (test failure, build error, runtime crash):
1. Log the failure to the midden using `midden-write` before trying an alternative.
2. Include: what you tried, what error occurred, and what you will try next.
3. This ensures the colony learns from every failed attempt, not just the final success.

### Stay On Task

- Reference the phase plan task list in your output.
- Mark tasks as complete as you finish them.
- If you discover work that needs doing but is outside your task scope, flag it -- do not do it.
- If a task is blocked, report the blocker clearly instead of working around it silently.

## Output Standards

End your work summary with:
- Tasks completed (by name from the plan).
- Tests written and their pass/fail status.
- Any REDIRECT signals you complied with and how.
- Any blockers or flags raised.
