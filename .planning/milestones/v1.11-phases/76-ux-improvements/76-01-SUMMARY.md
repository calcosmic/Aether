---
phase: 76-ux-improvements
plan: 01
subsystem: cmd
tags: [ux, onboarding, error-handling, progress]
dependency_graph:
  requires: []
  provides: [first-run-welcome, friendly-errors, ceremony-progress]
  affects: [cmd/root.go, cmd/helpers.go, cmd/codex_build.go, cmd/codex_continue.go]
tech-stack:
  added:
    - "github.com/schollz/progressbar/v3 v3.19.0"
  patterns:
    - "TTY/non-TTY detection via isTerminalWriter"
    - "Nil-safe progress pattern"
    - "Case-insensitive error pattern matching"
key-files:
  created:
    - cmd/ux_firstrun.go
    - cmd/ux_firstrun_test.go
    - cmd/ux_friendly_errors.go
    - cmd/ux_friendly_errors_test.go
    - cmd/ux_progress.go
    - cmd/ux_progress_test.go
  modified:
    - cmd/root.go
    - cmd/helpers.go
    - cmd/codex_build.go
    - cmd/codex_continue.go
    - go.mod
    - go.sum
decisions: []
metrics:
  duration: "13m30s"
  completed_date: "2026-04-29"
  tasks_completed: 2
  tasks_total: 2
  files_created: 6
  files_modified: 6
  tests_added: 25
---

# Phase 76 Plan 01: UX Foundations Summary

First-run welcome banner, friendly error messages with next-step suggestions, and ceremony progress bars with TTY/non-TTY fallback -- all wired into the Go runtime.

## What Changed

### First-Run Welcome Banner (UX-01)
- New users see a welcome banner with quick-start guidance on their first command
- Banner suppressed by `.welcomed` marker file in `.aether/data/`
- Skipped when colony already exists (COLONY_STATE.json present)
- Skipped in JSON output mode (only visual rendering)
- Wired into `root.go` PersistentPreRunE after store initialization

### Friendly Error Messages (UX-02)
- 7 error patterns mapped to plain-language explanations with actionable next steps
- Patterns ordered from most specific to least to avoid false matches
- Matched errors replaced entirely in visual mode (not just appended)
- Unknown errors get generic hint appended: "Run `aether patrol` for diagnostics or `aether status` to check colony health."
- JSON output mode completely unaffected -- raw error envelope preserved

### Ceremony Progress (UX-03)
- `ceremonyProgress` wrapper provides step-based progress with elapsed timing
- TTY mode: progressbar/v3 renders an animated progress bar
- Non-TTY mode: plain text "Step N/M: name (elapsed)" lines
- Build ceremony: Prepare, Context, Dispatch, Verify, Complete (5 advances)
- Continue ceremony: Verification, Housekeeping, Advance, Complete (4 advances)
- Nil-safe pattern prevents crashes when visual output is disabled

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] progressbar/v3 API mismatch**
- **Found during:** Task 2
- **Issue:** Plan specified `progressbar.SetDescription()` etc. but v3.19.0 uses `progressbar.OptionSetDescription()` with Option prefix
- **Fix:** Used correct `Option*` function names from the actual API
- **Files modified:** cmd/ux_progress.go

**2. [Rule 1 - Bug] renderBanner spacedTitle breaks string matching**
- **Found during:** Task 1
- **Issue:** `renderBanner` uses `spacedTitle()` which uppercases and spaces out letters ("Welcome to Aether" becomes "W E L C O M E   T O   A E T H E R"), so `strings.Contains(output, "Welcome to Aether")` fails
- **Fix:** Updated test to check for "W E L C O M E" instead
- **Files modified:** cmd/ux_firstrun_test.go

**3. [Rule 1 - Bug] Test environment not forcing visual mode**
- **Found during:** Task 1
- **Issue:** Tests using `bytes.Buffer` as stdout/stderr get JSON output because `shouldRenderVisualOutput()` returns false for non-TTY writers
- **Fix:** Set `AETHER_OUTPUT_MODE=visual` in tests that need to verify visual rendering
- **Files modified:** cmd/ux_firstrun_test.go, cmd/ux_friendly_errors_test.go

**4. [Rule 2 - Missing Critical] Error pattern ordering**
- **Found during:** Task 1
- **Issue:** "permission denied" pattern came before "failed to initialize store" in the map, causing the less specific match to win when an error message contained both strings
- **Fix:** Reordered pattern map so "failed to initialize store" (more specific) comes before "permission denied" (less specific)
- **Files modified:** cmd/ux_friendly_errors.go

**5. [Rule 1 - Bug] JSON parse error test used wrong substring**
- **Found during:** Task 1
- **Issue:** Test used "invalid character 'x'" but the pattern map matches on "json" substring, which that message doesn't contain
- **Fix:** Changed test to use "json: cannot unmarshal string into Go value" which does contain "json"
- **Files modified:** cmd/ux_friendly_errors_test.go

**6. [Rule 3 - Blocking] Worktree missing embedded asset directories**
- **Found during:** Task 1
- **Issue:** Worktree was missing `.aether/rules/`, `.claude/rules/`, `.opencode/`, `.codex/` directories needed by the `//go:embed` directive in `embedded_assets.go`
- **Fix:** Copied directories from main repo to worktree (build-time only, not committed)
- **Files modified:** (none committed -- worktree-local only)

## Auth Gates

None encountered.

## Known Stubs

None.

## Threat Flags

None -- no new network endpoints, auth paths, or trust boundary changes.

## Self-Check: PASSED

- [x] cmd/ux_firstrun.go exists
- [x] cmd/ux_firstrun_test.go exists
- [x] cmd/ux_friendly_errors.go exists
- [x] cmd/ux_friendly_errors_test.go exists
- [x] cmd/ux_progress.go exists
- [x] cmd/ux_progress_test.go exists
- [x] cmd/root.go modified (checkAndEmitFirstRun call)
- [x] cmd/helpers.go modified (friendly error + generic hint)
- [x] cmd/codex_build.go modified (5 progress advances + finish)
- [x] cmd/codex_continue.go modified (3 progress advances + finish)
- [x] go.mod updated (progressbar/v3 v3.19.0)
- [x] Commit 90d8d730 verified
- [x] Commit 7a9911db verified
- [x] All 25 new tests pass
