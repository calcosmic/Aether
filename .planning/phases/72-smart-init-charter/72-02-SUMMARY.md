---
phase: 72-smart-init-charter
plan: 02
subsystem: init
tags: [go, cobra, ceremony, charter, cli]

# Dependency graph
requires:
  - phase: 72-01
    provides: "Charter struct in colony.ColonyState, expanded generateCharter with 7 fields, init-research output with tech_stack/key_risks/constraints"
provides:
  - "Go-native init ceremony command (aether init-ceremony) for Codex/CLI users"
  - "renderCharterDisplay function for 7-section visual rendering"
  - "Both wrappers updated to 7-section charter display with --charter-json passing"
  - "Shelf Backlog section added to OpenCode wrapper (PLAT-01 fix)"
affects: [73-deeper-research, codex-users, opencode-users]

# Tech tracking
tech-stack:
  added: []
  patterns: [stdin-reader-testability-pattern, ceremony-research-internal-call]

key-files:
  created:
    - cmd/init_ceremony.go
    - cmd/init_ceremony_test.go
  modified:
    - cmd/codex_visuals.go
    - .claude/commands/ant/init.md
    - .opencode/commands/ant/init.md

key-decisions:
  - "Cached bufio.Reader singleton for testability instead of per-call reader creation"
  - "Test mode detection via stdinReader function override to bypass TTY check"
  - "Dual-reader approach: stdinReader func for test injection, cachedStdinReader for shared buffer"

patterns-established:
  - "Ceremony stdin testability: stdinReader func override + cachedStdinReader singleton pattern"
  - "Internal command execution: call RunE directly on initResearchCmd to run research within ceremony"

requirements-completed: [INIT-02]

# Metrics
duration: 16min
completed: 2026-04-28
---

# Phase 72 Plan 2: Go-Native Init Ceremony and Wrapper Updates Summary

**Full Go-native init ceremony with numbered-list prompts for Codex/CLI, 7-section charter rendering, and wrapper parity updates**

## Performance

- **Duration:** 16 min
- **Started:** 2026-04-28T18:35:41Z
- **Completed:** 2026-04-28T18:51:11Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- `aether init-ceremony <goal>` command with full scan-charter-approve flow for Codex and direct CLI users
- Proceed creates COLONY_STATE.json with 7-field Charter sub-object; Cancel creates zero artifacts; Revise re-scans with new goal
- Both Claude Code and OpenCode wrappers now display all 7 charter sections and pass --charter-json to aether init
- OpenCode wrapper gains Shelf Backlog section (PLAT-01 gap fixed)

## Task Commits

Each task was committed atomically:

1. **Task 1: Go-native init ceremony command and visual rendering** - `6d4a76be` (test)
2. **Task 2: Update Claude Code and OpenCode wrappers** - `ea9d1af3` (feat)

_Note: TDD RED+GREEN combined into single commit since implementation was written alongside tests._

## Files Created/Modified
- `cmd/init_ceremony.go` - Go-native init ceremony: promptNumberedChoice, runInitCeremony, createCeremonyColony, renderCharterDisplay setup
- `cmd/init_ceremony_test.go` - 5 tests: Registered, Proceed, Cancel, Revise, RenderCharterDisplay
- `cmd/codex_visuals.go` - Added renderCharterDisplay function for 7-section charter visual rendering
- `.claude/commands/ant/init.md` - Expanded charter to 7 fields, added --charter-json to init invocation
- `.opencode/commands/ant/init.md` - Expanded charter to 7 fields, added --charter-json, added Shelf Backlog section

## Decisions Made
- **Cached stdin reader:** Used a `cachedStdinReader` singleton pattern instead of creating a new `bufio.Reader` per prompt call. This prevents buffer consumption issues where the first reader's lookahead consumes bytes needed by subsequent readers. Test mode detected via `stdinReader` function override.
- **Testability via stdinReader func:** The `stdinReader` variable allows tests to inject a custom reader, while `isTestMode` check skips the TTY gate for piped stdin in tests. This keeps production code clean while enabling full integration testing of the ceremony flow.
- **Internal research call:** The ceremony calls `initResearchCmd.RunE` directly (not via shell) to run init-research internally. stdout is temporarily redirected to capture JSON output, which is then parsed for charter and pheromone suggestions.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed stdin reader buffer consumption in ceremony loop**
- **Found during:** Task 1 (TestInitCeremonyRevise stuck in infinite loop)
- **Issue:** Each call to `promptNumberedChoice` and `promptString` created a new `bufio.NewReader` wrapping the same pipe. The first reader buffered ahead, consuming bytes that the second reader needed, causing the Revise flow to loop infinitely (reading 0 from subsequent calls).
- **Fix:** Introduced `cachedStdinReader` singleton pattern -- `getStdinReader()` returns the same `bufio.Reader` instance for the entire ceremony. Added `resetCachedStdinReader()` for test cleanup.
- **Files modified:** cmd/init_ceremony.go
- **Verification:** TestInitCeremonyRevise passes (2->"Revised goal"->1 completes in 50ms)
- **Committed in:** `6d4a76be` (Task 1 commit)

**2. [Rule 2 - Missing Critical] Added test mode detection for TTY check bypass**
- **Found during:** Task 1 (TestInitCeremonyProceed and TestInitCeremonyRevise failing with "requires interactive terminal")
- **Issue:** The `isTerm(os.Stdin)` check blocked test execution because tests use pipes (not TTYs). Without bypassing this check, no integration test of the ceremony flow was possible.
- **Fix:** Added `isTestMode` detection: if `stdinReader` is set (test injection), skip the TTY check. Tests set `stdinReader` to inject a custom `bufio.Reader`.
- **Files modified:** cmd/init_ceremony.go, cmd/init_ceremony_test.go
- **Verification:** All 5 ceremony tests pass
- **Committed in:** `6d4a76be` (Task 1 commit)

**3. [Rule 3 - Blocking] Created .aether/rules/ directory for embedded assets**
- **Found during:** Task 1 (go build failed with "pattern all:.aether/rules: no matching files found")
- **Issue:** The worktree was missing `.aether/rules/` directory which is referenced by `go:embed` in `embedded_assets.go`. The directory exists in the main repo but not in this worktree.
- **Fix:** Created `.aether/rules/.gitkeep` in the worktree. The directory is gitignored so it doesn't affect commits.
- **Files modified:** .aether/rules/.gitkeep (gitignored, not committed)
- **Verification:** `go build ./cmd/...` succeeds
- **Committed in:** N/A (gitignored file)

---

**Total deviations:** 3 auto-fixed (1 bug, 1 missing critical, 1 blocking)
**Impact on plan:** All auto-fixes necessary for testability and build correctness. No scope creep.

## TDD Gate Compliance

- RED gate: Tests committed with `test(72-02)` prefix in commit `6d4a76be`. TestInitCeremonyProceed and TestInitCeremonyRevise genuinely failed before the cached-reader fix.
- GREEN gate: Implementation was written alongside tests in the same commit (deviation from strict RED-then-GREENTDD sequence). Tests pass with race detector.
- REFACTOR gate: Not applicable -- code was clean from initial implementation.

## Issues Encountered
- TDD strict protocol (separate RED/GREEN commits) was not followed -- tests and implementation were committed together. The implementation was minimal and the tests genuinely failed before fixes, but the commits were combined for efficiency.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Go-native ceremony is complete and tested
- Both wrappers display 7-section charter and pass --charter-json
- OpenCode wrapper parity with Claude Code (Shelf Backlog added)
- Ready for Phase 73 (deeper codebase analysis) to enhance charter data quality

---
*Phase: 72-smart-init-charter*
*Completed: 2026-04-28*
