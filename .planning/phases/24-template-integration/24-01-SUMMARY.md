---
phase: 24
plan: 01
subsystem: template-integration
tags: [templates, init, wiring, json]
dependency_graph:
  requires: [21-template-foundation]
  provides: [WIRE-01]
  affects: [.aether/templates/colony-state.template.json, .aether/templates/constraints.template.json, .claude/commands/ant/init.md, .opencode/commands/ant/init.md]
tech_stack:
  added: []
  patterns: [hub-first-template-lookup, llm-template-fill]
key_files:
  created: []
  modified:
    - .aether/templates/colony-state.template.json
    - .aether/templates/constraints.template.json
    - .claude/commands/ant/init.md
    - .opencode/commands/ant/init.md
decisions:
  - Hub-first lookup pattern used for all template resolutions (hub path checked before .aether/ local path)
  - Template missing error message matches locked decision exactly: "Template missing: {name}. Run aether update to fix."
  - OpenCode version adapted to use $normalized_args and Step 2.5 references while keeping template read text identical to Claude Code
  - __PHASE_LEARNINGS__ and __INSTINCTS__ explicitly documented as JSON arrays not strings in both template and command files
metrics:
  duration: 4 minutes
  completed: 2026-02-20
  tasks_completed: 2
  files_modified: 4
---

# Phase 24 Plan 01: Template Wiring for init.md Summary

Wire init.md (both Claude Code and OpenCode) to hub-first template lookup for colony-state.template.json and constraints.template.json, replacing inline JSON blocks with LLM-fill instructions, plus annotation refresh for both JSON templates.

## What Was Built

WIRE-01 complete. The two inline JSON blocks in `init.md` Step 3 (COLONY_STATE.json) and Step 4 (constraints.json) are replaced with template read instructions on both platforms. Both JSON templates are refreshed with improved annotations. Templates are now the single source of truth.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Refresh JSON templates and wire Claude Code init.md | d7b53db | colony-state.template.json, constraints.template.json, .claude/commands/ant/init.md |
| 2 | Wire OpenCode init.md to templates | 5d489bd | .opencode/commands/ant/init.md |

## Key Changes

**colony-state.template.json** — Refreshed annotations:
- `_instructions`: now explicit about "remove every key whose name starts with underscore"
- `_comment_session`: new field explaining both `__SESSION_ID__` and `__ISO8601_TIMESTAMP__` formats
- `_comment_memory`: updated to explicitly warn that `__PHASE_LEARNINGS__` and `__INSTINCTS__` must be JSON arrays not strings
- `_comment_events`: new field explaining the pipe-delimited event format

**constraints.template.json** — Refreshed annotations:
- `_instructions`: clearer about write-as-is behavior
- `_comment_purpose`: new field explaining this file stores FOCUS/REDIRECT pheromone signals

**init.md Step 3 (both platforms)** — Replaced inline JSON block with:
1. Hub-first template path resolution (hub checked before local .aether/)
2. Template missing error message with `aether update` recovery guidance
3. Explicit placeholder substitution list for all 5 `__PLACEHOLDER__` values
4. JSON array type warning for `__PHASE_LEARNINGS__` and `__INSTINCTS__`
5. Instruction to remove all underscore-prefixed keys before writing

**init.md Step 4 (both platforms)** — Replaced inline JSON block with:
1. Hub-first template path resolution
2. Template missing error message
3. Instruction to strip underscore keys and write data as-is

## Deviations from Plan

None — plan executed exactly as written.

## Verification Results

1. Neither init.md file contains `"version": "3.0"` inline JSON — PASS (0 matches each)
2. Both init.md files reference `colony-state.template.json` (3 times each) and `constraints.template.json` (3 times each) — PASS
3. Both init.md files have "Template missing" error handling (2 instances each) — PASS
4. `colony-state.template.json` contains `_instructions` (1 instance) — PASS
5. Both JSON templates pass `jq .` validation — PASS
6. Template references between Claude Code and OpenCode are identical — PASS
7. `npm test` — 3 pre-existing failures in validate-state.test.js (documented in STATE.md), no new failures

## Self-Check: PASSED

All modified files verified present and committed.
