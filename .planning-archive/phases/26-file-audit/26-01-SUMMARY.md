---
phase: 26-file-audit
plan: 01
subsystem: infra
tags: [repo-hygiene, dead-files, cleanup, npm-package]

# Dependency graph
requires: []
provides:
  - "Clean repo root: logo files, Aether Notes, planning/, .cursor/, .worktrees/ deleted"
  - "Clean .aether/ root: workers-new-castes.md, recover.sh, debugging artifacts, Python prototypes, examples/ deleted"
  - "Dead hub migration references removed from bin/cli.js"
  - ".claude/commands/gsd/new-project.md.bak deleted (CLEAN-07 partial)"
  - ".opencode/agents/workers.md stale duplicate deleted (CLEAN-07 partial)"
  - "CLEAN-02 and CLEAN-03 confirmed: .aether/agents/ and .aether/commands/ were already gone"
affects: [26-02, 26-03, 26-04]

# Tech tracking
tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified:
    - "bin/cli.js"

key-decisions:
  - "Dead file deletions were split: tracked files via git rm, gitignored dirs via direct rm"
  - "bin/cli.js hub migration list cleaned: workers-new-castes.md and recover.sh removed from systemFiles array (one-time migration, fs.existsSync made it safe but now fully clean)"
  - "CLEAN-02 and CLEAN-03 confirmed already satisfied: .aether/agents/ and .aether/commands/ do not exist"
  - "Content-level command drift (lint:sync warnings) is pre-existing known debt — did not block Task 2"
  - "Two prior commits (96e93cd, 9bfb4ea labeled 26-02) had already deleted many 26-01 targets in the same batch — Task 1 cleanup was completing residual work"

patterns-established: []

requirements-completed:
  - CLEAN-01
  - CLEAN-02
  - CLEAN-03
  - CLEAN-07

# Metrics
duration: 3min
completed: 2026-02-20
---

# Phase 26 Plan 01: Delete Dead Files (Repo Root, .aether/ Root, .claude/, .opencode/) Summary

**Deleted 20+ dead files from repo root, .aether/ root, and tooling directories — removing logo binaries, Python prototypes, debugging artifacts, dated handoffs, and stale duplicates; cli.js migration list cleaned of deleted file references**

## Performance

- **Duration:** ~3 min
- **Started:** 2026-02-20T03:49:45Z
- **Completed:** 2026-02-20T03:52:30Z
- **Tasks:** 2
- **Files modified:** 1 (bin/cli.js); ~20 files deleted

## Accomplishments

- Repo root cleaned: removed "Aether Notes" (macOS alias), aether-logo.png (1.2MB), logo_block.txt, logo_block_color.txt, planning/ (empty dir), .cursor/ (IDE config), .worktrees/ (stale local data)
- .aether/ root cleaned: removed workers-new-castes.md, recover.sh, HANDOFF_AETHER_DEV_2026-02-15.md, PHASE-0-ANALYSIS.md, RESEARCH-SHARED-DATA.md, diagnose-self-reference.md, DIAGNOSIS_PROMPT.md, pheromone_system.py, semantic_layer.py, __pycache__/, examples/
- bin/cli.js hub migration list pruned: workers-new-castes.md and recover.sh removed from the one-time migration systemFiles array
- CLEAN-07 satisfied: .claude/commands/gsd/new-project.md.bak and .opencode/agents/workers.md deleted
- CLEAN-02 and CLEAN-03 confirmed: .aether/agents/ and .aether/commands/ were already absent

## Task Commits

1. **Task 1: Delete repo root and .aether/ root dead files** - `89d3af7` (chore)
2. **Task 2: Delete .claude/ and .opencode/ dead duplicates** - `5acd585` (chore)

**Note:** Many Task 1 file deletions were already committed in prior session commits (96e93cd, 9bfb4ea) that ran before this plan execution. Task 1 here focused on the remaining cleanup: untracked directory removal and the bin/cli.js migration list fix.

## Files Created/Modified

- `/Users/callumcowie/repos/Aether/bin/cli.js` — Removed workers-new-castes.md and recover.sh from hub migration systemFiles array (these files no longer exist)

## Decisions Made

- Workers-new-castes.md and recover.sh were removed from the hub migration code in cli.js, not just deleted from disk — the migration code used `fs.existsSync` so it was harmless, but referencing deleted files is dead code
- .opencode/agents/workers.md deletion: confirmed no agent file references it by path; OpenCode loads individual agent .md files, not workers.md; authoritative source is .aether/workers.md via hub
- Pre-existing content drift in lint:sync (10+ command files differ between Claude Code and OpenCode platforms) did not block task — this is documented pre-existing debt

## Deviations from Plan

None — plan executed exactly as written. The only observation is that two prior commits had already deleted most of the Task 1 targets; this plan confirmed those deletions and cleaned up what remained (untracked dirs + cli.js reference).

## Issues Encountered

None. npm pack --dry-run succeeded (181 files, 1.8MB), npm test showed only 2 pre-existing validate-state failures (baseline unchanged), lint:sync showed 34/34 commands in sync.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 26-01 complete: CLEAN-01 (partial), CLEAN-02, CLEAN-03, CLEAN-07 satisfied
- Repo root and .aether/ root are now clean
- Ready for 26-02: .aether/docs/ dead docs audit (if not already committed in prior session)
- Ready for 26-03: docs/plans/, .planning/milestones/ cleanup, TO-DOS.md cleanup

---
*Phase: 26-file-audit*
*Completed: 2026-02-20*
