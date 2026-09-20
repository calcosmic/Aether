---
phase: 72-smart-init-charter
plan: 01
subsystem: data-model
tags: [charter, colony-state, init, go, json, backward-compat]

# Dependency graph
requires: []
provides:
  - Charter struct with 7 fields in pkg/colony
  - --charter-json flag on aether init for charter persistence
  - Expanded init-research charter output with tech_stack, key_risks, constraints
  - Backward-compatible COLONY_STATE.json schema (charter is omitempty)
affects: [72-02-smart-init-ceremony, wrappers]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pointer + omitempty for optional state fields (backward compat)"
    - "Input validation with max field length on user-controlled JSON (T-72-01)"

key-files:
  created: []
  modified:
    - pkg/colony/colony.go
    - pkg/colony/colony_test.go
    - pkg/colony/testdata/COLONY_STATE.golden.json
    - cmd/init_research.go
    - cmd/init_research_test.go
    - cmd/init_cmd.go
    - cmd/init_cmd_test.go

key-decisions:
  - "Used pointer type with omitempty for Charter field -- old COLONY_STATE.json files without charter unmarshal to nil Charter"
  - "Replaced standalone charterData struct with colony.Charter -- single source of truth for JSON field names"
  - "Added 2000-char per-field validation on --charter-json input to prevent unreasonably large state files (T-72-01)"
  - "TestCharterBackwardCompat uses inline old JSON instead of golden file -- golden now includes charter for forward-compat testing"

patterns-established:
  - "Optional state fields: pointer type + omitempty json tag for backward compatibility"
  - "User-input JSON: unmarshal into typed struct (rejects extra fields), validate field lengths"

requirements-completed: [INIT-01]

# Metrics
duration: 11min
completed: 2026-04-28
---

# Phase 72 Plan 01: Charter Data Expansion and Persistence Summary

**Charter struct with 7 fields in pkg/colony, --charter-json flag on aether init, expanded init-research with tech stack / key risks / constraints generation**

## Performance

- **Duration:** 11 min
- **Started:** 2026-04-28T18:14:44Z
- **Completed:** 2026-04-28T18:25:55Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Charter struct added to pkg/colony with 7 fields: intent, vision, governance, goals, tech_stack, key_risks, constraints
- ColonyState.Charter field uses pointer + omitempty for full backward compatibility with old state files
- init-research generates 3 new charter sections from scan data: tech stack (languages/frameworks), key risks (CI gaps, test gaps, lint gaps, secret exposure, no VCS), constraints (governance tool rules)
- --charter-json flag on `aether init` parses and validates charter JSON before persisting to COLONY_STATE.json
- Input validation prevents oversized charter fields (2000 char max per field, per T-72-01 threat mitigation)

## Task Commits

Each task was committed atomically (TDD: RED then GREEN per task):

1. **Task 1: Add Charter struct and expand charterData** - `41487d79` (test), `84f85f15` (feat)
2. **Task 2: Add --charter-json flag and wire charter persistence** - `f91b4b35` (test), `d257f244` (feat)

_Note: TDD tasks each have RED (failing tests) and GREEN (implementation) commits._

## Files Created/Modified
- `pkg/colony/colony.go` - Added Charter struct (7 fields) and Charter field on ColonyState
- `pkg/colony/colony_test.go` - Added TestCharterRoundTrip, TestCharterOmitEmpty, TestCharterBackwardCompat
- `pkg/colony/testdata/COLONY_STATE.golden.json` - Added charter sub-object for forward-compat golden test
- `cmd/init_research.go` - Expanded charterData to 7 fields, added generateTechStack/generateKeyRisks/generateConstraints, replaced charterData with colony.Charter
- `cmd/init_research_test.go` - Added TestInitResearchCharterExpanded, TestInitResearchCharterKeyRisksNoCI, TestInitResearchCharterConstraintsWithLinter
- `cmd/init_cmd.go` - Added --charter-json flag, JSON parsing with validation, validateCharterFieldLength helper
- `cmd/init_cmd_test.go` - Added TestInitWithCharterJSONFlag, TestInitWithoutCharterJSONFlag, TestInitInvalidCharterJSON

## Decisions Made
- Used pointer type (*Charter) with omitempty for the Charter field on ColonyState -- ensures old COLONY_STATE.json files without charter unmarshal cleanly to nil
- Replaced standalone charterData struct with colony.Charter in init_research.go -- eliminates JSON field name drift between research output and state persistence
- TestCharterBackwardCompat uses inline old-style JSON instead of the golden file -- the golden file now includes charter for forward-compat testing, so backward compat must use a separate fixture
- Added 2000-char per-field validation on --charter-json input (T-72-01 threat mitigation) -- prevents unreasonably large state files from user-controlled CLI input

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] TestCharterBackwardCompat failed after updating golden file**
- **Found during:** Task 1 GREEN phase
- **Issue:** Test loaded golden file expecting nil Charter, but golden was updated to include charter for forward-compat
- **Fix:** Changed test to use inline old-style JSON without charter field instead of the golden file
- **Files modified:** pkg/colony/colony_test.go
- **Committed in:** `84f85f15` (part of Task 1 GREEN commit)

**2. [Rule 3 - Blocking] Missing .aether/rules/ directory in worktree caused embedded asset build failure**
- **Found during:** Task 1 RED phase verification
- **Issue:** `embedded_assets.go` embeds `.aether/rules:*` which didn't exist in the worktree
- **Fix:** Copied .aether/rules/aether-colony.md from main repo to worktree
- **Files modified:** .aether/rules/aether-colony.md (worktree-local only, not committed)

---

**Total deviations:** 2 auto-fixed (1 bug, 1 blocking)
**Impact on plan:** Both auto-fixes were necessary for correctness. No scope creep.

## Issues Encountered
- 6 pre-existing test failures in cmd package (TestContinueEmitsLifecycleCeremonyEvents, TestContinueBlocksWhenWatcherUsesFakeInvoker, TestClaudeOpenCodeCommandParity, TestIntegrityDetectSourceContext, TestLifecycleCommandDocsPreferRuntimeCLI, TestQueenWisdomHygiene) -- confirmed identical on base commit, unrelated to this plan

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Charter struct is ready for Plan 02 (Go-native ceremony) to use internally
- --charter-json flag is ready for wrapper commands to pass approved charter data
- init-research output now includes all 7 charter fields for wrapper consumption
- No blockers or concerns

## TDD Gate Compliance

- RED gate: `41487d79` (Task 1 tests), `f91b4b35` (Task 2 tests) -- both confirmed failing before implementation
- GREEN gate: `84f85f15` (Task 1 impl), `d257f244` (Task 2 impl) -- all tests passing after implementation
- No REFACTOR commits needed -- implementation was clean on first pass

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: user_input | cmd/init_cmd.go | --charter-json accepts user-controlled JSON string, parsed into typed struct with field length validation |

---
*Phase: 72-smart-init-charter*
*Completed: 2026-04-28*
