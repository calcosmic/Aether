---
phase: 29-specialist-agents-agent-tests
plan: "02"
subsystem: agents
tags: [agents, claude-code, probe, weaver, testing, refactoring]
dependency_graph:
  requires: []
  provides:
    - aether-probe agent (test generation + coverage analysis)
    - aether-weaver agent (behavior-preserving refactoring with revert protocol)
  affects:
    - .claude/agents/ant/ (12 agents total after this plan)
tech_stack:
  added: []
  patterns:
    - 8-section XML agent body (role, execution_flow, critical_rules, return_format, success_criteria, failure_modes, escalation, boundaries)
    - Explicit revert protocol in failure_modes (not just documentation)
    - Write-scope restriction via boundaries section
key_files:
  created:
    - .claude/agents/ant/aether-probe.md
    - .claude/agents/ant/aether-weaver.md
  modified: []
decisions:
  - "Probe BOTH writes AND runs tests — untested tests are incomplete work; Bash available specifically for this"
  - "Weaver revert protocol uses explicit git commands (git checkout -- {files}, git stash) — behavior preservation enforced not documented"
  - "Test expectations prohibition in Weaver critical_rules — changing assertions to make tests pass is a behavior change, not a refactor"
metrics:
  duration: "~3 minutes"
  completed: 2026-02-20
  tasks_completed: 2
  files_created: 2
---

# Phase 29 Plan 02: Probe and Weaver Specialist Agents Summary

**One-liner:** Probe writes and runs tests (boundaries restrict to test files only); Weaver refactors with git-based revert-on-test-failure enforcement in failure_modes.

## What Was Built

Two specialist Claude Code subagents in `.claude/agents/ant/`:

**aether-probe** — The colony's quality assurance specialist. Scans for untested paths, generates tests using boundary value analysis and equivalence partitioning, then runs all new tests to verify they pass. Boundaries strictly restrict writes to test directories only (`tests/`, `__tests__/`, `*.test.*`, `*.spec.*`). Never touches source code.

**aether-weaver** — The colony's craftsperson for refactoring. Captures a baseline test count before starting, applies one refactoring technique per step, runs tests after each step, and reverts immediately (using `git checkout -- {files}` or `git stash`) if tests break. The revert is explicit and automatic — not optional guidance.

Both agents follow the established Phase 27–28 format: YAML frontmatter with explicit `tools:` field, 8-section XML body, zero OpenCode patterns.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Create Probe agent | 2e67ccd | .claude/agents/ant/aether-probe.md |
| 2 | Create Weaver agent | 51cdd59 | .claude/agents/ant/aether-weaver.md |

## Decisions Made

**Probe writes AND runs tests** — The plan specified "Claude's Discretion" on whether Probe writes only or writes and runs. Resolution: Probe has Bash specifically to run the test suite. Writing tests without running them is incomplete work. The `<success_criteria>` requires: "Run all new tests — they must pass." This matches the OpenCode Probe success criteria and is practical with Bash available.

**Weaver revert protocol is explicit, not aspirational** — The locked decision from CONTEXT.md states "behavior preservation is enforced, not just documented." The `<failure_modes>` section includes the exact git commands for reverting (`git checkout -- {files}`, `git stash`), and names the revert as automatic (not a recommendation). The research Pitfall 5 specifically warned about Weaver failure_modes that only mention "behavior preservation" without revert language — addressed directly.

**Test expectations prohibition in critical_rules** — Weaver's `<critical_rules>` explicitly states: "Never change test expectations to make tests 'pass' after a refactoring — that is a behavior change, not a refactor." This closes the loophole where an agent could technically "pass" tests by updating their assertions.

## Cross-Reference Escalation

Both agents include explicit cross-referencing:
- Probe escalates to: Tracker (bugs discovered during testing), Builder (source needs seams), Weaver (source needs testability refactoring), Queen (architectural changes)
- Weaver escalates to: Queen (architectural changes), Probe (tests missing before refactor), Tracker (bugs revealed during refactoring)

The colony feels connected — "If refactoring exposes untested paths, Probe should add tests before Weaver continues."

## Deviations from Plan

None — plan executed exactly as written.

## Self-Check: PASSED

| Item | Status |
|------|--------|
| .claude/agents/ant/aether-probe.md | FOUND |
| .claude/agents/ant/aether-weaver.md | FOUND |
| .planning/phases/29-specialist-agents-agent-tests/29-02-SUMMARY.md | FOUND |
| Commit 2e67ccd (probe) | FOUND |
| Commit 51cdd59 (weaver) | FOUND |
