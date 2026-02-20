---
phase: 29-specialist-agents-agent-tests
plan: "03"
subsystem: testing
tags: [ava, agent-quality, test-suite, automation]
dependency_graph:
  requires: ["29-01", "29-02"]
  provides: ["agent quality gate", "TEST-01 through TEST-05 enforcement"]
  affects: ["npm test output", ".claude/agents/ant/*.md"]
tech_stack:
  added: []
  patterns: ["dynamic file discovery with readdirSync", "YAML frontmatter parsing with js-yaml", "forbidden-only tool constraint validation"]
key_files:
  created:
    - tests/unit/agent-quality.test.js
  modified: []
decisions:
  - "Forbidden pattern matching refined from bare strings to aether-utils.sh invocation form — eliminates false positives from Queen's documentation of prohibited patterns"
  - "READ_ONLY_CONSTRAINTS registry: forbidden-only approach (not exact match) for flexible constraint checking"
  - "Single test file: all 6 test functions coherent as agent-quality suite"
metrics:
  duration: "3 minutes"
  completed: "2026-02-20"
  tasks_completed: 1
  files_created: 1
  files_modified: 0
---

# Phase 29 Plan 03: Agent Quality Test Suite Summary

AVA test suite enforcing frontmatter, naming, read-only tools, OpenCode invocation patterns, agent count tracking, and XML body quality across all 14 agent files in `.claude/agents/ant/`.

## What Was Built

Single test file `tests/unit/agent-quality.test.js` with 6 test functions:

| Test | Requirement | Result |
|------|-------------|--------|
| TEST-01 | All agents have name, description, tools frontmatter | PASS (14/14 agents) |
| TEST-02 | Agent names match aether-{role} pattern + filename | PASS (14/14 agents) |
| TEST-03 | Tracker (no Write/Edit), Auditor (no Write/Edit/Bash) | PASS |
| TEST-04 | No agent body invokes OpenCode patterns via aether-utils.sh | PASS (14/14 agents) |
| TEST-05 | Agent count = 22 | INTENTIONALLY FAILS (14 found, 8 needed from Phase 30) |
| Body quality | 8 XML sections present, non-empty, >= 50 chars each | PASS (14/14 agents) |

## Key Implementation Decisions

**Dynamic discovery:** `fs.readdirSync(AGENTS_DIR).filter(f => f.endsWith('.md')).sort()` — tests automatically cover any future agents added to `.claude/agents/ant/` without code changes.

**Tools parsed as array:** `parseTools()` helper splits comma-separated string before comparisons. Never calls `.includes()` on raw tools string (avoids substring false positives).

**TEST-05 is the Phase 30 tracker:** Hardcoded to 22. After Phase 29: 14 agents exist. Phase 30 must deliver 8 more (ambassador, archaeologist, chaos, chronicler, gatekeeper, includer, measurer, sage) to turn this test green.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Refined forbidden pattern matching to prevent false positives**
- **Found during:** Task 1 (first test run)
- **Issue:** The plan specified bare patterns like `/activity-log/` for TEST-04. The Queen's `critical_rules` section contains the line: "Do not use: `activity-log`, `spawn-can-spawn`, `generate-ant-name`, `spawn-log`, `spawn-complete`..." — documentation of prohibited patterns, not actual invocations. The bare patterns matched this documentation line, causing TEST-04 to falsely fail for the queen.
- **Fix:** Changed all forbidden patterns to match the `aether-utils.sh <command>` invocation form (e.g., `/aether-utils\.sh activity-log/`). Every actual OpenCode invocation routes through `aether-utils.sh`. Documentation references to pattern names are not invocations and should not fail the test.
- **Result:** TEST-04 now passes for all 14 agents. The fix is more precise about intent — it catches actual invocations, not documentation of what NOT to invoke.
- **Files modified:** `tests/unit/agent-quality.test.js` (FORBIDDEN_PATTERNS array, test name)
- **Commit:** a56e703

## Expected Post-Phase-29 State

`npm test` shows exactly 1 failing test:
```
✘ agent-quality › TEST-05: agent count matches expected 22
  Expected 22 agents, found 14. Remaining: 8 agents needed (Phase 30).
```

All other tests pass. This is the documented and intentional state after Phase 29.

## Self-Check: PASSED

- tests/unit/agent-quality.test.js: FOUND
- 29-03-SUMMARY.md: FOUND
- commit a56e703: FOUND
