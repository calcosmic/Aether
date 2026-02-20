---
phase: 25-queen-coordination
plan: "03"
subsystem: agent-definitions
tags: [agent-count, cleanup, caste-system, documentation, consolidation]

# Dependency graph
requires:
  - phase: 25-queen-coordination/25-02
    provides: "Architect merged into Keeper, Guardian merged into Auditor, agent files deleted"
provides:
  - "Consistent 23-agent identity across all user-facing docs"
  - "Queen Worker Castes section updated to reflect 23 agents"
  - "caste-system.md architect/guardian rows annotated as merged (emoji rows preserved)"
  - "workers.md Architect section annotated as merged into Keeper"
  - "README.md and OPENCODE.md updated to 23 agents"
affects: [future phases referencing agent count, any doc referencing the caste system]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Annotate-don't-delete pattern: preserve historical rows/sections with merge notes for emoji resolution continuity"

key-files:
  created: []
  modified:
    - ".opencode/agents/aether-queen.md"
    - ".aether/docs/caste-system.md"
    - ".aether/workers.md"
    - "README.md"
    - ".opencode/OPENCODE.md"

key-decisions:
  - "Preserve architect/guardian rows in caste-system.md with merge annotations — get_caste_emoji() still maps those name patterns to emojis"
  - "workers.md Architect section annotated (not deleted) — historical reference value preserved"
  - "No Guardian section existed in workers.md — only figurative use of 'guardian' in Watcher description"

requirements-completed: [COORD-03, COORD-04]

# Metrics
duration: 5min
completed: 2026-02-20
---

# Phase 25 Plan 03: Agent Count Cleanup (25 -> 23) Summary

**Consistent 23-agent identity established across all user-facing docs — Queen Worker Castes, caste-system.md annotations, workers.md, README.md, and OPENCODE.md all aligned**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-20T01:02:55Z
- **Completed:** 2026-02-20T01:03:45Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Queen agent `## Worker Castes` updated: Architect removed from Core Castes, Guardian removed from Quality Cluster; Keeper and Auditor entries updated to note absorbed capabilities
- caste-system.md architect and guardian rows annotated as "merged into Keeper/Auditor — no dedicated agent file"; Notes section explains why emoji rows are preserved for `get_caste_emoji()` pattern matching
- workers.md `## Architect` section heading updated to `## Architect (Merged into Keeper)` with merge note — no Guardian section existed to update
- README.md updated from "25 Specialized Agents" to "23 Specialized Agents" (both occurrences: Key Features list and Installation section)
- OPENCODE.md agent file listing cleaned: aether-guardian.md removed (aether-architect.md was already absent from listing)

## Task Commits

1. **Task 1: Update Queen Worker Castes and caste-system.md** - `71e8a43` (feat)
2. **Task 2: Update workers.md, README.md, and OPENCODE.md agent counts** - `43f76c5` (feat)

## Files Created/Modified

- `.opencode/agents/aether-queen.md` — Removed Architect from Core Castes, removed Guardian from Quality Cluster, extended Keeper and Auditor entries with capability notes
- `.aether/docs/caste-system.md` — Annotated architect/guardian rows with "merged" note in Role column; added Notes entry explaining emoji row preservation
- `.aether/workers.md` — Renamed `## Architect` to `## Architect (Merged into Keeper)` with merge note block
- `README.md` — Updated "25 Specialized Agents" -> "23 Specialized Agents" in Key Features and Installation sections
- `.opencode/OPENCODE.md` — Removed `aether-guardian.md` line from Agent Files listing

## Decisions Made

- **Preserve emoji rows in caste-system.md:** architect and guardian caste rows kept because `get_caste_emoji()` matches worker names against patterns (e.g., "Blueprint-3" matches architect pattern). Deleting the rows would break emoji resolution for any worker named with those patterns.
- **Annotate workers.md instead of delete:** The Architect section contains valuable model and workflow context. Merge annotation at the heading preserves the reference value while making the merged status clear.
- **No Guardian section to update in workers.md:** The only "guardian" reference in workers.md was figurative ("The colony's guardian — when work is done, you verify it's correct") in the Watcher section. No dedicated Guardian section existed.

## Deviations from Plan

**1. [Pre-completion] Most changes already committed from prior sessions**
- Found during: Initial file inspection
- Issue: Plans 25-01 and 25-02 had already made the majority of changes specified in this plan. Both task commits (`71e8a43`, `43f76c5`) were already in git history.
- Fix: Verified all success criteria were met, confirmed all must_haves.truths matched actual file state, confirmed zero remaining "25 agents" references in user-facing docs.
- Impact: None — end state matches plan specification exactly.

## Issues Encountered

None beyond the pre-completion state. `npm run lint:sync` shows command count in sync (34 each). Content drift is pre-existing known debt across 10+ files, not caused by this plan.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 25 complete — all 4 requirements (COORD-01 through COORD-04) satisfied
- 23-agent identity is fully consistent across all user-facing documentation
- No stale "25 agents" references remain in docs, README, or agent listings
- Pre-existing known debt: content-level drift between Claude Code and OpenCode command files (10+ files) — deferred

## Self-Check: PASSED

- FOUND: .opencode/agents/aether-queen.md
- FOUND: .aether/docs/caste-system.md
- FOUND: .aether/workers.md
- FOUND: README.md
- FOUND: .opencode/OPENCODE.md
- FOUND: .planning/phases/25-queen-coordination/25-03-SUMMARY.md
- VERIFIED: commit 71e8a43 (Task 1)
- VERIFIED: commit 43f76c5 (Task 2)

---
*Phase: 25-queen-coordination*
*Completed: 2026-02-20*
