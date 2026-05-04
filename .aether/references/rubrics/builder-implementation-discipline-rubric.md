---
schema_version: "1.0"
id: builder-implementation-discipline-rubric
kind: rubric
category: rubrics
title: Builder Implementation Discipline Rubric
description: "Quality checks for builder worker output: scope, targeting, test discipline, and handoff quality."
output_types: [code-review, quality-gate, build-review]
agent_roles: [watcher, auditor, queen, architect]
task_types: [build, implement, code, review, quality]
task_keywords: [builder, scope, refactor, targeting, test, handoff, discipline, evidence, TDD, scoring, gate]
workflow_triggers: [build, continue]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4000
---

# Builder Implementation Discipline Rubric

This rubric defines quality criteria for evaluating builder worker output.
Use it during continue verification, code review, and quality gates to
assess whether a builder met discipline standards.

## For Beginners

When a builder finishes a task, someone needs to check whether they did a
good job. This rubric is a scoring guide -- a checklist of things that
separate disciplined work from sloppy work. A watcher or auditor uses it to
evaluate the builder's output fairly and consistently.

## Scoring Criteria

Each criterion is scored on a three-point scale:

| Score | Meaning |
|-------|---------|
| Meets standard | The criterion is fully satisfied |
| Partial | The criterion is partially satisfied, with minor gaps |
| Does not meet | The criterion is not satisfied, with significant gaps |

### 1. Scope Discipline (Weight: High)

**Criterion:** The builder made only the changes required by the task
description, without scope creep.

**Meets standard:**
- Changed files are exactly those needed for the task
- No unrelated refactoring or "while I was here" changes
- No new dependencies added without explicit task requirement
- Changes are proportional to the task complexity

**Partial:**
- One or two minor unrelated changes that do not affect functionality
- Small cleanup alongside the main change

**Does not meet:**
- Significant refactoring outside task scope
- New files or features not requested
- Changes to unrelated subsystems

### 2. File Targeting (Weight: High)

**Criterion:** Changes were made in the correct files following Aether's
architecture conventions.

**Meets standard:**
- Go changes in `cmd/` or `pkg/` as appropriate
- Agent definitions in the canonical location (`.claude/agents/ant/`)
- Agent mirrors updated to match
- No edits to protected paths (`.aether/data/`, `.aether/dreams/`)

**Partial:**
- Correct target files but one mirror not updated
- Correct logic but wrong package placement

**Does not meet:**
- Changes to protected paths
- Wrong architectural layer entirely
- Source edited when mirror should have been updated (or vice versa)

### 3. Test Discipline (Weight: Critical)

**Criterion:** The builder followed TDD principles and provided adequate test
coverage for the changes.

**Meets standard:**
- New functions have corresponding tests
- Edge cases are covered
- Tests are deterministic (no flaky tests)
- Tests follow existing test patterns in the codebase
- `go test ./... -race` passes

**Partial:**
- Main paths tested but edge cases missing
- Tests exist but follow inconsistent patterns
- Race detection reveals one minor issue

**Does not meet:**
- No tests for new code
- Tests that pass regardless of implementation
- Race conditions detected
- Tests that depend on external state

### 4. Handoff Quality (Weight: Medium)

**Criterion:** The builder produced a complete and accurate handoff for
subsequent workers.

**Meets standard:**
- `changed_files` lists all modified files with correct paths
- `commands_run` includes the actual commands executed
- `verification_status` honestly reflects test results
- `known_failures` lists any unresolved issues
- `next_worker_instructions` provides specific, actionable guidance
- `do_not_repeat` documents failed approaches

**Partial:**
- Handoff present but missing one or two fields
- Vague `next_worker_instructions`
- `verification_status` is accurate but `known_failures` is incomplete

**Does not meet:**
- No handoff produced
- `verification_status` says "pass" but tests fail
- Vague or misleading instructions for the next worker

### 5. Code Quality (Weight: Medium)

**Criterion:** The code follows Go idioms, project conventions, and
maintainability standards.

**Meets standard:**
- Follows existing code patterns in the file
- Clear function and variable names
- No unnecessary complexity
- Error handling follows project conventions
- `go vet` passes with no warnings

**Partial:**
- Mostly follows conventions with minor deviations
- One or two `go vet` warnings

**Does not meet:**
- Significantly different style from surrounding code
- Complex logic without comments
- Multiple `go vet` warnings
- Ignored errors or panics

### 6. Evidence Requirements (Weight: Medium)

**Criterion:** The builder provided concrete evidence that the work is
correct, not just assertions.

**Meets standard:**
- Test output shown (actual test run, not just "tests pass")
- Build verification with `go build` and `go vet`
- Specific test names or coverage areas mentioned
- For CLI changes: actual command output demonstrating correctness

**Partial:**
- Claims "tests pass" without showing output
- Mentions verification but without specific evidence

**Does not meet:**
- No verification evidence
- Claims of correctness contradicted by test output
- "It should work" without running anything

## Scoring Summary

| Criterion | Weight | Meets | Partial | Does Not Meet |
|-----------|--------|-------|---------|---------------|
| Scope discipline | High | +3 | +1 | 0 |
| File targeting | High | +3 | +1 | 0 |
| Test discipline | Critical | +4 | +2 | 0 |
| Handoff quality | Medium | +2 | +1 | 0 |
| Code quality | Medium | +2 | +1 | 0 |
| Evidence | Medium | +2 | +1 | 0 |

**Maximum score:** 16
**Passing threshold:** 11 (must include "meets" on test discipline)
**Block threshold:** Below 8

Any builder output scoring below the block threshold should be flagged as a
soft_block for Queen escalation. A score below 8 with failing tests should be
classified as a hard_block.
