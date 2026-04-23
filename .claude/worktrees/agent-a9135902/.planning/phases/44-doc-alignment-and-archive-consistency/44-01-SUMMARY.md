---
phase: 44-doc-alignment-and-archive-consistency
plan: 01
subsystem: docs
tags: [publish, integrity, stale-detection, runbook, operations-guide]

# Dependency graph
requires:
  - phase: 43
    provides: "aether integrity command, scanIntegrity in medic --deep, stale publish detection in update"
provides:
  - "All five core docs reference aether publish as primary path"
  - "aether integrity documented in operations guide and runbook"
  - "aether update stale publish detection documented with classification table"
  - "aether version --check documented in runbook verification checklist"
  - "Dev channel publish workflow documented with aether publish --channel dev"
affects: [45, 46, operators, release-workflow]

# Tech tracking
tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified:
    - AETHER-OPERATIONS-GUIDE.md
    - .aether/docs/publish-update-runbook.md
    - CLAUDE.md
    - .codex/CODEX.md
    - .opencode/OPENCODE.md

key-decisions:
  - "aether publish documented as primary command; aether install --package-dir kept as backward-compatible alternative"
  - "aether integrity section inserted as Section 11 in operations guide, sections renumbered 11-14"
  - "Stale publish detection classification table added to both operations guide and runbook"

patterns-established: []

requirements-completed: [REL-03]

# Metrics
duration: 3min
completed: 2026-04-23
---

# Phase 44 Plan 01: Doc Alignment Summary

**All five core docs aligned to reference aether publish as primary path, document aether integrity command, and describe stale publish detection behavior**

## Performance

- **Duration:** 3 min
- **Started:** 2026-04-23T19:40:26Z
- **Completed:** 2026-04-23T19:43:18Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- AETHER-OPERATIONS-GUIDE.md updated with new aether integrity section (Section 11), stale publish detection docs in Section 7, and Section 14 now uses aether publish
- publish-update-runbook.md updated with Publish Command section (full flag reference), Integrity Check section, Stale Publish Detection section, and aether version --check in verification checklist
- CLAUDE.md Publishing Changes section leads with aether publish, documents aether integrity and aether version --check
- CODEX.md Publishing Changes section leads with aether publish, documents aether integrity and stale publish detection
- OPENCODE.md updated across all publish references to use aether publish, adds aether integrity and aether version --check

## Task Commits

Each task was committed atomically:

1. **Task 1: Update AETHER-OPERATIONS-GUIDE.md with publish, integrity, and update channel docs** - `dcf6fcae` (docs)
2. **Task 2: Update publish-update-runbook.md, CLAUDE.md, CODEX.md, OPENCODE.md** - `70d14946` (docs)

## Files Created/Modified
- `AETHER-OPERATIONS-GUIDE.md` - Added integrity section, stale detection, updated Section 14 to use aether publish
- `.aether/docs/publish-update-runbook.md` - Added Publish Command, Integrity Check, Stale Publish Detection sections; updated workflows
- `CLAUDE.md` - Updated Publishing Changes to lead with aether publish, added integrity and version --check
- `.codex/CODEX.md` - Updated Publishing Changes to lead with aether publish, added integrity reference
- `.opencode/OPENCODE.md` - Updated all publish references, added integrity and version --check to verification

## Decisions Made
- aether publish documented as primary command in all five docs; aether install --package-dir preserved as backward-compatible alternative with explicit notes
- Integrity section inserted as Section 11 in operations guide, pushing Safe Testing Matrix to 12, Do/Don't to 13, Short Version to 14
- Stale publish detection documented with full classification table (ok/info/warning/critical) matching actual Go runtime behavior

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All five docs are internally consistent with each other and with actual Go runtime behavior
- Plan 02 (44-02) can proceed with remaining doc alignment work if needed
- REL-03 (R064) requirement satisfied

---
*Phase: 44-doc-alignment-and-archive-consistency*
*Completed: 2026-04-23*
