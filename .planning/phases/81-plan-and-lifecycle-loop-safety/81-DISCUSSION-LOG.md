# Phase 81: Plan and Lifecycle Loop Safety - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-30
**Phase:** 81-plan-and-lifecycle-loop-safety
**Areas discussed:** Circular dependency detection scope, Lifecycle command error behavior, Dependency graph implementation

---

## Circular dependency detection scope

| Option | Description | Selected |
|--------|-------------|----------|
| Phase-level only | Check only depends_on in ROADMAP.md phases | |
| Phase + task level | Check both phase-level and task-level depends_on | |
| Phase at plan time, task at build time | Phase check during planning, task check during execution | |
| Task-level in plan | Only validate task-level depends_on within the generated plan | ✓ |

**User's choice:** "you decide" — Claude selected task-level cycle detection in plan
**Notes:** User pointed out Aether doesn't use ROADMAP.md directly — dependencies are tracked at the task level within plans, not between phases. The circular check should run where cycles actually matter: on the plan's task dependency graph.

## Lifecycle command error behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Error + text suggestion | Print error with 'Next step:' suggestion | |
| Error + interactive menu | Error followed by numbered recovery options | ✓ |
| Error only, no suggestion | Just print the error | |

**User's choice:** Error + interactive menu
**Notes:** User wants hand-holding when lifecycle commands fail.

## Dependency graph implementation

| Option | Description | Selected |
|--------|-------------|----------|
| One-time cycle check on plan | DFS cycle detection after plan generation, no persistent graph | ✓ |
| Persistent graph in colony state | Build and maintain dependency graph in COLONY_STATE.json | |
| Build-time validation only | Cycle detection at execution time, not plan time | |

**User's choice:** One-time cycle check on plan
**Notes:** Simplest approach — runs once per plan, rejects if cycle found. No state management overhead.

## Recovery menu option determination

| Option | Description | Selected |
|--------|-------------|----------|
| Fixed recovery per command | Hardcoded recovery options per lifecycle command | |
| Dynamic recovery engine | Context-aware suggestions based on error type analysis | ✓ |

**User's choice:** Dynamic recovery engine
**Notes:** More intelligent than hardcoded options. Recovery engine analyzes error type and context to produce relevant suggestions.

## Claude's Discretion
- Task-level cycle detection (user said "you decide")
- One-time DFS cycle check (user said "you decide")
- Dynamic recovery engine (explicit user choice)

## Deferred Ideas

None.
