---
phase: 22-config-foundation-core-routing
plan: 03
subsystem: docs
tags: [routing, agent-frontmatter, model-selection, documentation]

# Dependency graph
requires:
  - phase: 22-01
    provides: "Slot-based worker_models in model-profiles.yaml"
  - phase: 22-02
    provides: "Agent frontmatter model: field across 22 agents"
provides:
  - "Updated Model Selection documentation in workers.md (two-tier routing, GLM-5 activation, dual-mode switching)"
  - "Updated verify-castes command showing caste-to-slot assignments (8 opus, 11 sonnet, 3 inherit)"
  - "Removed all 'per-caste routing is impossible' claims from both files"
affects: [documentation, onboarding, dual-mode-operation]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Two-tier routing documentation: slot assignment + environment variable resolution"

key-files:
  created: []
  modified:
    - .aether/workers.md
    - .claude/commands/ant/verify-castes.md
    - .opencode/commands/ant/verify-castes.md

key-decisions:
  - "OpenCode verify-castes mirror updated alongside Claude Code version (sync policy parity)"

patterns-established:
  - "Historical note pattern: distinguish failed v1 approach from working v2 approach"

requirements-completed: [ROUTE-05, ROUTE-07, ROUTE-08]

# Metrics
duration: 2min
completed: 2026-03-27
---

# Phase 22 Plan 03: Documentation Rewrite Summary

**Rewrote workers.md Model Selection and verify-castes command to document working two-tier per-caste routing via agent frontmatter, replacing outdated "impossible" claims with GLM-5 activation instructions and dual-mode switching docs**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-27T05:55:59Z
- **Completed:** 2026-03-27T05:58:18Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Replaced workers.md Model Selection section with complete two-tier routing documentation including caste-to-slot table, slot resolution mechanism, GLM-5 activation via settings.json, and dual-mode switching instructions
- Rewrote verify-castes.md to display 22 castes grouped by model slot (8 opus, 11 sonnet, 3 inherit) with frontmatter source references
- Removed all claims that per-caste routing is impossible from both files
- Updated historical notes to correctly distinguish failed v1 (env var injection) from working v2 (agent frontmatter)
- Synced OpenCode verify-castes mirror for command parity

## Task Commits

Each task was committed atomically:

1. **Task 1: Rewrite workers.md Model Selection section** - `3658695` (docs)
2. **Task 2: Update verify-castes.md to show slot assignments** - `c7170ae` (docs)

## Files Created/Modified
- `.aether/workers.md` - Replaced Model Selection section with two-tier routing docs (lines 53-91)
- `.claude/commands/ant/verify-castes.md` - Rewrote Steps 1, 3, 4, and Historical Note for slot assignments
- `.opencode/commands/ant/verify-castes.md` - Synced OpenCode mirror with same updates (Rule 2 deviation)

## Decisions Made
- OpenCode verify-castes mirror updated alongside Claude Code version to maintain the project's sync policy, even though the plan only specified `.claude/commands/ant/`

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Updated OpenCode verify-castes.md mirror**
- **Found during:** Task 2 (update verify-castes.md)
- **Issue:** Plan only specified `.claude/commands/ant/verify-castes.md` but the project maintains an OpenCode mirror at `.opencode/commands/ant/verify-castes.md` with identical outdated content ("not possible", 10 castes, old LiteLLM models)
- **Fix:** Applied the same four replacements (Step 1, Step 3, Step 4, Historical Note) to the OpenCode mirror with border character adaptation (double-line `=` instead of single-line `=` for OpenCode style)
- **Files modified:** `.opencode/commands/ant/verify-castes.md`
- **Verification:** Same impossibility claims removed, all three slot tiers present
- **Committed in:** `c7170ae` (part of Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** Necessary for command sync parity. No scope creep.

## Issues Encountered
None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Phase 22 (Per-Caste Model Routing) is complete -- all 3 plans executed
- workers.md accurately documents the working routing system for user reference
- verify-castes command correctly displays caste-to-slot assignments
- Dual-mode switching instructions available for GLM proxy vs Claude API

---
*Phase: 22-config-foundation-core-routing*
*Completed: 2026-03-27*

## Self-Check: PASSED

- FOUND: .aether/workers.md
- FOUND: .claude/commands/ant/verify-castes.md
- FOUND: .opencode/commands/ant/verify-castes.md
- FOUND: 22-03-SUMMARY.md
- FOUND: 3658695 (Task 1 commit)
- FOUND: c7170ae (Task 2 commit)
