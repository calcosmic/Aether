---
phase: 28-orchestration-layer-surveyor-variants
plan: "02"
subsystem: agent-layer
tags: [agents, scout, route-setter, claude-code, subagents, pwr-compliant]
dependency_graph:
  requires: [27-02, 27-03]
  provides: [aether-scout-agent, aether-route-setter-agent]
  affects: [ant:oracle, ant:plan, ant:build]
tech_stack:
  added: []
  patterns: [yaml-frontmatter-xml-body, 8-section-template, pwr-compliant-agent, read-only-posture, graceful-degradation]
key_files:
  created:
    - .claude/agents/ant/aether-scout.md
    - .claude/agents/ant/aether-route-setter.md
  modified: []
decisions:
  - "Scout gets WebSearch/WebFetch but no Bash — read-only posture enforced via explicit tools field"
  - "Route-Setter Task tool documented with graceful degradation note for subagent context where Task may be ineffective"
  - "spawns field removed from Scout return format — Claude Code subagents cannot spawn other subagents"
metrics:
  duration: "2 minutes"
  completed_date: "2026-02-20"
  tasks_completed: 2
  files_created: 2
---

# Phase 28 Plan 02: Scout and Route-Setter Agents Summary

Scout and Route-Setter added as PWR-compliant Claude Code subagents — Scout with WebSearch/WebFetch for external research (read-only), Route-Setter with Task tool and planning discipline for goal decomposition.

## What Was Built

Two orchestration-layer agents that complete the colony's research and planning capabilities:

**Scout** (`aether-scout.md`) — The colony's quick researcher. Has WebSearch and WebFetch tools (unique in the colony — no other agent has them). Strictly read-only: no Write, Edit, or Bash. Returns structured JSON findings with source citations. Differentiates from Oracle: Scout handles quick lookups, Oracle handles deep iterative research.

**Route-Setter** (`aether-route-setter.md`) — The colony's planner. Has the Task tool included per the locked decision from Phase 28 planning. Enforces planning discipline (bite-sized tasks, exact file paths, expected outputs, TDD flow). Includes graceful degradation note: if running as a subagent where Task may be unavailable, escalate verification to the calling orchestrator.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Create Scout agent | ad0ffcc | `.claude/agents/ant/aether-scout.md` |
| 2 | Create Route-Setter agent | bc6e025 | `.claude/agents/ant/aether-route-setter.md` |

## Verification Results

- Both files exist at `.claude/agents/ant/`
- Scout tools: Read, Grep, Glob, WebSearch, WebFetch — confirmed no Write/Edit/Bash
- Route-Setter tools: Read, Grep, Glob, Bash, Write, Task — Task confirmed present
- Both have all 8 XML sections (role, execution_flow, critical_rules, return_format, success_criteria, failure_modes, escalation, boundaries)
- Zero OpenCode-specific patterns (activity-log, spawn-can-spawn, generate-ant-name, etc.) in either file
- Scout: 142 lines (min 100)
- Route-Setter: 173 lines (min 120)
- Graceful degradation for Task tool documented in Route-Setter escalation section

## Deviations from Plan

None — plan executed exactly as written.

## Self-Check: PASSED

Files created:
- FOUND: .claude/agents/ant/aether-scout.md
- FOUND: .claude/agents/ant/aether-route-setter.md

Commits verified:
- FOUND: ad0ffcc (Scout agent)
- FOUND: bc6e025 (Route-Setter agent)
