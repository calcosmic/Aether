---
phase: 30-niche-agents
plan: "01"
subsystem: agents
tags: [claude-code, subagents, accessibility, performance, security, git-archaeology, chaos-testing]

requires: []
provides:
  - "aether-chaos: adversarial testing agent with 5-category investigation framework"
  - "aether-archaeologist: regression prevention via git history excavation"
  - "aether-measurer: performance profiling and bottleneck identification"
  - "aether-gatekeeper: static supply chain dependency audit (no Bash)"
  - "aether-includer: static WCAG 2.1 AA accessibility audit (no Bash)"
  - "aether-sage: project analytics and trend analysis via git data extraction"
affects: [30-02, 30-03, agent-quality-tests]

tech-stack:
  added: []
  patterns:
    - "Read/Bash/Grep/Glob pattern for investigation agents that need probing (chaos, archaeologist, measurer)"
    - "Read/Grep/Glob only pattern for strictest read-only agents (gatekeeper, includer)"
    - "Read/Grep/Glob/Bash without Write pattern for analysis agents (sage)"
    - "All 6 agents follow 8-section XML template: role, execution_flow, critical_rules, return_format, success_criteria, failure_modes, escalation, boundaries"
    - "Static analysis scope honesty — includer and gatekeeper document what cannot be assessed without runtime tools"

key-files:
  created:
    - .claude/agents/ant/aether-chaos.md
    - .claude/agents/ant/aether-archaeologist.md
    - .claude/agents/ant/aether-measurer.md
    - .claude/agents/ant/aether-gatekeeper.md
    - .claude/agents/ant/aether-includer.md
    - .claude/agents/ant/aether-sage.md
  modified: []

key-decisions:
  - "Gatekeeper and Includer have no Bash — static analysis only, no npm audit or axe-core execution — Builder handles command execution"
  - "Includer always includes analysis_method: 'manual static analysis' in return and documents runtime testing gaps explicitly"
  - "Archaeologist leads with regression prevention framing — primary deliverable is regression_risks array, not general archaeology"
  - "Sage has Bash for git data extraction but no Write — if findings need to be persisted, caller routes to Keeper"
  - "Chaos severity ratings must reflect actual risk not theoretical concern — CRITICAL requires realistic scenario, not contrived preconditions"

patterns-established:
  - "Tooling gap honesty: agents that cannot run dynamic tools (Gatekeeper, Includer) document the gap and provide builder_command recommendations"
  - "Runtime testing gaps: Includer documents dynamic concerns (computed contrast, screen reader behavior) as out-of-scope gaps, not failures"
  - "Static CVE pattern matching: Gatekeeper matches known patterns and labels findings as provisional, requiring Builder to confirm with npm audit"

requirements-completed: [NICHE-01, NICHE-02, NICHE-05, NICHE-06, NICHE-07, NICHE-08]

duration: 10min
completed: 2026-02-20
---

# Phase 30 Plan 01: Niche Agents (Read-Only Batch) Summary

**6 read-only specialist agents — chaos tester, regression preventer, performance profiler, supply chain auditor, accessibility auditor, and project analyst — completing the niche agent cohort with differentiated tool restrictions enforced at the platform level.**

## Performance

- **Duration:** 10 min
- **Started:** 2026-02-20T10:29:28Z
- **Completed:** 2026-02-20T10:40:14Z
- **Tasks:** 2
- **Files modified:** 6 created

## Accomplishments

- Created 6 niche agents with substantive, task-specific content (268-373 lines each)
- Enforced differentiated tool restrictions: 3 agents with Bash, 2 strictly no-Bash, 1 with Bash but no Write/Edit
- All agents pass 8-section XML template structure and zero OpenCode pattern contamination
- Archaeologist framed as regression prevention first — primary deliverable is regression_risks, not archaeology
- Gatekeeper and Includer explicitly document static-analysis limitations and provide builder_command recommendations for runtime tooling gaps

## Task Commits

Each task was committed atomically:

1. **Task 1: Create Chaos, Archaeologist, and Measurer agents** - `967ab58` (feat)
2. **Task 2: Create Gatekeeper, Includer, and Sage agents** - `308c1ba` (feat)

**Plan metadata:** [created in this step] (docs: complete plan)

## Files Created/Modified

- `.claude/agents/ant/aether-chaos.md` — Adversarial tester; 5-category framework (edge cases, boundaries, error handling, state corruption, unexpected inputs); tools: Read/Bash/Grep/Glob
- `.claude/agents/ant/aether-archaeologist.md` — Regression preventer via git history excavation; regression_risks as primary output; tools: Read/Bash/Grep/Glob
- `.claude/agents/ant/aether-measurer.md` — Performance profiler; detects project type, performs static complexity analysis and dynamic benchmarking; tools: Read/Bash/Grep/Glob
- `.claude/agents/ant/aether-gatekeeper.md` — Supply chain auditor; static manifest/lock file inspection only, no Bash; provides builder_command for npm audit; tools: Read/Grep/Glob
- `.claude/agents/ant/aether-includer.md` — WCAG 2.1 AA accessibility auditor; manual static inspection of HTML/ARIA/CSS; documents runtime testing gaps explicitly; tools: Read/Grep/Glob
- `.claude/agents/ant/aether-sage.md` — Project analytics via git history; churn hotspots, bug density, knowledge concentration, velocity trends; tools: Read/Grep/Glob/Bash

## Decisions Made

- Gatekeeper and Includer have no Bash — static analysis only. When dynamic tooling is needed, they document the gap and provide a builder_command for Builder to run.
- Includer always returns `analysis_method: "manual static analysis"` and explicitly lists `runtime_testing_gaps` — scope honesty is non-negotiable.
- Archaeologist leads with regression prevention framing per locked decision from research phase.
- Sage has no Write tool by design — findings flow to Keeper for persistence, not written directly.
- Chaos severity ratings must cite realistic scenarios; CRITICAL requires realistic attack vectors, not theoretical preconditions.

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

None. All 6 agents created, verified, and committed without issues. Agent count confirmed at 22 total (14 pre-existing + ambassador + chronicler from 30-02 + 6 from this plan), which means TEST-05 in the agent quality test suite will now pass as intended.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- All 6 niche read-only agents complete and committed
- Phase 30 Plan 02 (Ambassador and Chronicler) was already completed before this plan — 30-02-SUMMARY.md exists
- TEST-05 in `tests/unit/agent-quality.test.js` (hardcoded to expect 22 agents) will now pass
- Phase 30 Plan 03 (agent quality tests for the new agents) is the remaining work in Phase 30

---
*Phase: 30-niche-agents*
*Completed: 2026-02-20*
