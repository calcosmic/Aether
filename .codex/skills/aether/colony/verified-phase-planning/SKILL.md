---
name: verified-phase-planning
description: Use when producing or checking a phase plan before builders begin work
type: colony
domains: [planning, architecture, verification]
agent_roles: [route_setter, architect]
workflow_triggers: [plan]
task_keywords: [plan, phase, tasks, dependency order, risk assessment]
priority: normal
version: "1.0"
---

# Verified Phase Planning

## Purpose
Produces a verified PLAN.md for a colony phase by running a planner-checker loop. The planner drafts the plan, the checker verifies it against quality gates, and the cycle repeats until all gates pass or a maximum of 3 iterations is reached.

## When to Use
- `aether plan` is invoked without an existing PLAN.md for the target phase
- A phase is about to be built but lacks a plan artifact
- The queen reassigns a phase that failed verification and needs replanning
- The architect determines a phase is too complex for ad-hoc execution

## Instructions

### Step 1 -- Gather Context
1. Read the colony goal from `.aether/colony.md` (the `goal` field).
2. Read the roadmap from `.aether/roadmap.md` to understand phase numbering, dependencies, and scope.
3. Read any existing SPEC.md or CONTEXT.md for the target phase from `.aether/phases/{phase}/`.
4. Scan the codebase for files related to the phase scope using glob patterns matching the phase description keywords.
5. If a previous phase was completed, read its LEARNINGS.md (if present) for carry-forward insights.

### Step 2 -- Draft Plan (Planner Pass)
Write the initial PLAN.md with the following structure:

```markdown
# Phase {N} Plan: {Title}

## Goal
{One sentence describing what this phase delivers}

## Scope
- In scope: {bullet list}
- Out of scope: {bullet list}

## Tasks
### Task 1: {Name}
- Files: {paths to create or modify}
- Steps: {numbered implementation steps}
- Verification: {how to confirm this task is done}
- Depends on: {other task numbers or "none"}

### Task 2: {Name}
...

## Dependency Order
{Topological sort of tasks -- which must complete before which}

## Risk Assessment
| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| {risk} | {H/M/L} | {H/M/L} | {action} |

## Verification Criteria
- [ ] All tasks completed
- [ ] All verification steps pass
- [ ] No regressions in existing functionality
- [ ] LEARNINGS.md written
```

### Step 3 -- Verify Plan (Checker Pass)
Evaluate the draft against these quality gates:

1. **Completeness Gate**: Every task has files, steps, and verification criteria.
2. **Dependency Gate**: No circular dependencies; dependency order is a valid DAG.
3. **Scope Gate**: Nothing in scope violates the roadmap boundaries or colony goal.
4. **Risk Gate**: Every HIGH-likelihood or HIGH-impact risk has a concrete mitigation.
5. **Verification Gate**: Each verification criterion is falsifiable (can be proven false).

Scoring: Each gate scores PASS or FAIL. All must PASS.

### Step 4 -- Loop
- If all gates PASS: finalize the PLAN.md and write it to `.aether/phases/{phase}/PLAN.md`.
- If any gate FAILS: rewrite the failing sections, increment the iteration counter, and re-run the checker. Maximum 3 iterations.
- After 3 iterations with failures: write the plan as-is but prefix with a warning block listing unresolved gates. Escalate to the queen.

### Step 5 -- Record State
Update `.aether/phases/{phase}/state.md` with:
```
phase: {N}
status: planned
plan_iterations: {count}
quality_gates: {all_passed | list_failed}
planned_at: {ISO timestamp}
```

## Key Patterns

### Dependency Topological Sort
When listing task dependencies, always produce a valid topological order. If tasks A and B are independent, they can run in parallel (wave-based). If B depends on A, A must come first. Never create cycles (A->B->C->A).

### Falsifiable Verification
Every verification criterion must be testable. Avoid vague checks like "works correctly." Prefer:
- "Running `npm test` exits with code 0"
- "The endpoint `/api/health` returns 200"
- "No TypeScript compilation errors"

### Risk Matrix
Use High/Medium/Low for both likelihood and impact. Any cell with both HIGH requires a mitigation strategy that reduces at least one dimension.

## Output Format
- `.aether/phases/{phase}/PLAN.md` -- the verified execution plan
- `.aether/phases/{phase}/state.md` -- updated phase state

## Examples

### Example 1 -- Planning a REST API Phase
Triggered by `aether plan` for phase 2 ("Build REST API endpoints").

1. Context gathered: colony goal is "Build a task management app", phase 1 completed the database schema, SPEC.md exists with endpoint requirements.
2. Planner drafts tasks: Task 1 (Create route handlers), Task 2 (Add validation middleware), Task 3 (Write integration tests). Tasks 2 and 3 depend on Task 1.
3. Checker verifies: all gates pass on first iteration.
4. PLAN.md written to `.aether/phases/2/PLAN.md`.

### Example 2 -- Replanning After Failure
Phase 3 failed verification (integration tests flaky). Queen triggers replan.

1. Previous PLAN.md read, LEARNINGS.md from failed attempt consulted ("Test timing issues with async operations").
2. Planner adjusts Task 3 to use deterministic test patterns, adds explicit wait strategies.
3. Checker: Risk gate initially FAILS (flaky tests marked as HIGH likelihood with no mitigation). Planner adds retry logic and seed data. Second iteration PASSES.
4. Updated PLAN.md replaces the old one.

### Example 3 -- Complex Phase with Wave Execution
Phase 5 covers "Frontend dashboard + real-time updates."

1. Planner identifies 7 tasks, groups into 3 dependency waves:
   - Wave 1: Task 1 (API client), Task 2 (WebSocket connection)
   - Wave 2: Task 3 (Dashboard layout), Task 4 (Chart components), Task 5 (Notification panel)
   - Wave 3: Task 6 (Integration), Task 7 (E2E tests)
2. Checker confirms valid DAG, all gates pass.
3. Plan explicitly marks waves for the wave-executor skill to consume.
