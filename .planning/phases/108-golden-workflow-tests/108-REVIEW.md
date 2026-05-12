---
phase: 108-golden-workflow-tests
reviewed: 2026-05-12T15:00:00Z
depth: standard
files_reviewed: 4
files_reviewed_list:
  - cmd/golden_workflow_test.go
  - cmd/testdata/golden_plan.txt
  - cmd/testdata/golden_build.txt
  - cmd/testdata/golden_continue.txt
findings:
  critical: 0
  warning: 3
  info: 3
  total: 6
status: issues_found
---

# Phase 108: Code Review Report

**Reviewed:** 2026-05-12T15:00:00Z
**Depth:** standard
**Files Reviewed:** 4
**Status:** issues_found

## Summary

Reviewed the golden workflow test files: one Go test file (`golden_workflow_test.go`) and three golden text fixture files. The tests exercise plan, build, and continue commands through the full visual output pipeline, comparing normalized output against golden files.

The test code is well-structured with proper setup/teardown, shared helper reuse, and content assertions beyond pure golden comparison. Three warnings and three info-level findings were identified. No security vulnerabilities or data loss risks were found.

## Warnings

### WR-01: `stripANSI` reimplements unexported function -- will silently diverge

**File:** `cmd/golden_workflow_test.go:28-46`
**Issue:** The `stripANSI` function is a copy of `pkg/codex/platform_dispatch.go:stripANSIEscapeCodes` (line 1082-1100). The comment acknowledges this. If the production version is ever fixed or extended (e.g., to handle OSC sequences, cursor position queries, or other CSI sequences that do not terminate with a single letter), the test copy will not follow. This is a maintenance trap: golden comparisons will silently break or silently pass on output that the production code handles differently. There is no test or build guard detecting divergence between the two copies.
**Fix:** Extract `stripANSIEscapeCodes` into a shared internal package (e.g., `pkg/ansi/strip.go`) that both the production code and test code import. Alternatively, add a test that compares the two implementations' output against a representative set of ANSI escape sequences to detect divergence.

### WR-02: `regexp.MatchString` called inside a per-line loop without caching compiled regex

**File:** `cmd/golden_workflow_test.go:81,114,117`
**Issue:** Three calls to `regexp.MatchString` are made inside the `normalizeForGolden` function, which processes every line of output. `regexp.MatchString` compiles a new regex on every call. While this is not a performance-critical path (test-only), the idiomatic and more robust approach is to pre-compile these as package-level `*regexp.Regexp` variables (as already done with `workerNameRe` on line 51). The inconsistency also makes it easier to accidentally introduce a malformed pattern that only fails at runtime during specific test inputs.
**Fix:** Pre-compile the three regex patterns as package-level variables, matching the pattern already used for `workerNameRe` on line 51:
```go
var (
    waveProgressRe   = regexp.MustCompile(`^Wave \d+: \d+/\d+ `)
    tableSeparatorRe = regexp.MustCompile(`^\+[-+]+\+$`)
    tableRowRe       = regexp.MustCompile(`^\|.*\|.*\|.*\|`)
)
```
Then use `waveProgressRe.MatchString(trimmed)` etc. in the filter function.

### WR-03: `TestGoldenContinueVisualOutput` sets `PhaseInProgress` on one task but seeds a dispatch for a different task

**File:** `cmd/golden_workflow_test.go:287-317`
**Issue:** The colony state on lines 297-301 sets phase 1 task with ID `"g-task-1"` and status `TaskInProgress`. But the `seedContinueBuildPacket` call on line 317 seeds dispatches where the wave dispatch uses `TaskID: taskID` (which is `"g-task-1"`), so this part is consistent. However, the state declares two phases with tasks, and phase 1 has only one task, while phase 2 has `"g-task-2"`. The dispatches seeded include a builder for `"Golden builder task"` (matching the phase 1 task goal) and a watcher for `"Independent verification"`. This is fine for the test but the golden_continue.txt fixture shows verification output referencing "Golden phase" (line 5, 33) which matches. The test is internally consistent but the setup is fragile: if `continue` ever validates that all tasks in the phase have dispatches (not just the first), this test would break. This is a warning rather than blocker because the current continue logic accepts partial dispatch coverage.
**Fix:** Document the assumption in a comment, or add a second builder dispatch for any additional tasks if the test intends to be robust against future continue validation changes.

## Info

### IN-01: `normalizeForGolden` filtering logic is complex with overlapping conditions

**File:** `cmd/golden_workflow_test.go:69-124`
**Issue:** The `normalizeForGolden` function contains 10+ filter conditions that process lines sequentially. Some conditions overlap (e.g., line 95-99 checks for indented lines containing "completed task=" or "simulated worker heartbeat", and line 103 checks for indented lines containing "Worker: Worker-XX"). A line matching both conditions would be handled by whichever comes first. This is not a bug today, but the overlapping predicates make the filter fragile to maintain.
**Fix:** Consider consolidating related filters or adding a comment explaining the intended priority between overlapping conditions.

### IN-02: `compareGolden` trims the golden file content with `TrimRight` but normalizes the live output differently

**File:** `cmd/golden_workflow_test.go:147`
**Issue:** The comparison on line 147 applies `strings.TrimRight(string(data), "\n\t ")+"\n"` to the file content but applies `normalizeForGolden` to the live output. The `normalizeForGolden` function already does `strings.TrimSpace(filtered.String()) + "\n"` on line 124. The asymmetric normalization means trailing whitespace differences in the golden file are silently accepted but leading whitespace differences are not. This is intentional (tolerant of editor trailing-newline behavior) but the asymmetry is not documented.
**Fix:** Add a brief comment explaining why the golden file gets `TrimRight` normalization while the live output gets full `normalizeForGolden` treatment.

### IN-03: `loadTestColonyState` helper is only used by `TestGoldenStateMutations`

**File:** `cmd/golden_workflow_test.go:340-350`
**Issue:** The `loadTestColonyState` helper is a thin wrapper around `loadColonyState()` that is only called from one test. The function adds nil-check safety, which is valuable, but as a single-call-site helper it could be inlined. Keeping it as a helper is fine for readability, but if more tests are added this should be moved to the shared test helper file.
**Fix:** No action required. Minor organizational note for future test expansion.

---

_Reviewed: 2026-05-12T15:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
