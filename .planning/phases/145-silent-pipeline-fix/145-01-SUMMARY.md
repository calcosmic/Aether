---
phase: 145-silent-pipeline-fix
plan: 01
subsystem: docs
tags: [playbooks, error-handling, cli, aether]

requires:
  - phase: 145-silent-pipeline-fix
    provides: Research and planning for honest error handling policy

provides:
  - All playbook data-persistence CLI calls fail visibly instead of silently
  - Playbook prose documents honest-error policy correctly
  - Read-only and fallback suppressions remain intact

affects:
  - 146-command-classification
  - 147-critical-test-coverage
  - 148-learning-and-workflow-restoration

tech-stack:
  added: []
  patterns:
    - "Honest error handling: data-persistence commands surface stderr and propagate non-zero exit"
    - "Read-only probes may still suppress stderr to avoid noise"

key-files:
  created: []
  modified:
    - .aether/docs/command-playbooks/build-complete.md
    - .aether/docs/command-playbooks/build-context.md
    - .aether/docs/command-playbooks/build-full.md
    - .aether/docs/command-playbooks/build-verify.md
    - .aether/docs/command-playbooks/build-wave.md
    - .aether/docs/command-playbooks/continue-advance.md
    - .aether/docs/command-playbooks/continue-finalize.md
    - .aether/docs/command-playbooks/continue-full.md
    - .aether/docs/command-playbooks/continue-verify.md

key-decisions:
  - "Prose updated to 'non-blocking but visible' instead of 'silent and non-blocking' to match D-01 policy"
  - "Section headings like '(SILENT, NON-BLOCKING)' simplified to '(NON-BLOCKING)'"

patterns-established:
  - "Data-persistence calls: never use 2>/dev/null || true"
  - "Read-only probes (grep, curl, stat, ls): suppression allowed"
  - "State-mutate guards: keep 2>/dev/null without || true"
  - "jq fallback defaults: keep 2>/dev/null || echo pattern"

requirements-completed:
  - PIPE-01

duration: 15min
completed: 2026-05-20
---

# Phase 145 Plan 01: Silent Pipeline Fix Summary

**Removed error suppression from 29+ data-persistence CLI calls across 9 playbook files, updated prose to reflect honest-error policy, preserved all read-only fallback suppressions.**

## Performance

- **Duration:** 15 min
- **Started:** 2026-05-20T22:32:00Z
- **Completed:** 2026-05-20T22:47:00Z
- **Tasks:** 2
- **Files modified:** 9

## Accomplishments

- Removed `2>/dev/null || true` from all data-persistence CLI calls in playbook instructions
- Updated playbook prose to document visible-error behavior instead of silent failure
- Preserved read-only fallback suppressions (grep, curl, stat, ls, git rev-parse, wc, find)
- Preserved state-mutate guard calls and jq fallback defaults

## Task Commits

1. **Task 1: Audit and categorize all suppression patterns + Task 2: Update playbook prose** - `3264856c` (fix)

## Files Created/Modified

- `.aether/docs/command-playbooks/build-complete.md` - Removed suppression from memory-capture
- `.aether/docs/command-playbooks/build-context.md` - Updated suggest-analyze error handling prose
- `.aether/docs/command-playbooks/build-full.md` - Removed suppression from suggest-approve, midden-write, memory-capture, pheromone-write
- `.aether/docs/command-playbooks/build-verify.md` - Removed suppression from midden-write, memory-capture
- `.aether/docs/command-playbooks/build-wave.md` - Removed suppression from midden-write, memory-capture, pheromone-write
- `.aether/docs/command-playbooks/continue-advance.md` - Removed suppression from memory-capture, instinct-create, pheromone-write, rm; updated prose
- `.aether/docs/command-playbooks/continue-finalize.md` - Removed suppression from backup-prune-global, temp-clean
- `.aether/docs/command-playbooks/continue-full.md` - Removed suppression from pheromone-write, memory-capture, backup-prune-global, temp-clean; updated prose
- `.aether/docs/command-playbooks/continue-verify.md` - Removed suppression from survey-verify; updated prose

## Decisions Made

- Prose updated to "non-blocking but visible" instead of "silent and non-blocking" to match D-01 policy
- Section headings like "(SILENT, NON-BLOCKING)" simplified to "(NON-BLOCKING)"

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- Worktree HEAD was at an older commit (710f355e) than the local main branch (ef74ee63). The fix commit e26112f8 existed on main but the worktree was created from an older base. This required re-applying the suppression removal to the worktree files.
- Read tool caching caused temporary confusion when verifying file contents; switched to Python for reliable content inspection.

## Next Phase Readiness

- Pipeline playbooks now surface data-persistence errors honestly
- Ready for Phase 146: Command Classification

---
*Phase: 145-silent-pipeline-fix*
*Completed: 2026-05-20*
