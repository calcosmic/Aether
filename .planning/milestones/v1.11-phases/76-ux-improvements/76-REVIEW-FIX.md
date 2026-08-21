---
phase: 76
fixed_at: 2026-04-29T21:40:00Z
review_path: .planning/phases/76-ux-improvements/76-REVIEW.md
iteration: 1
findings_in_scope: 6
fixed: 6
skipped: 0
status: all_fixed
---

# Phase 76: Code Review Fix Report

**Fixed at:** 2026-04-29T21:40:00Z
**Source review:** `.planning/phases/76-ux-improvements/76-REVIEW.md`
**Iteration:** 1

**Summary:**
- Findings in scope: 6 (1 Critical, 5 Warning)
- Fixed: 6
- Skipped: 0

## Fixed Issues

### CR-01: Overly broad "json" error pattern matches any error containing "json"

**Files modified:** `cmd/ux_friendly_errors.go`
**Commit:** `399d68c7`
**Applied fix:** Changed pattern from `"json"` to `"json:"` to match Go's `encoding/json` error prefix format (e.g., `json: cannot unmarshal`). This prevents false matches on arbitrary strings containing "json" like filenames or field names.

### WR-01: Progress bar step names defined but never used during Advance

**Files modified:** `cmd/ux_progress.go`
**Commit:** `922f0fea`
**Applied fix:** Updated `Advance()` to fall back to the `steps` array when `stepName` is empty. If a caller omits the step name, the method resolves it from `p.steps[p.current-1]`.

### WR-02: Ceremony progress ChangeMax called every Advance is redundant

**Files modified:** `cmd/ux_progress.go`
**Commit:** `922f0fea`
**Applied fix:** Removed the `p.bar.ChangeMax(len(p.steps))` call from `Advance()`. The max is already set once in the constructor via `progressbar.NewOptions(len(steps))`, so every subsequent call was a no-op.

### WR-03: "flag --" and "missing flag --" error patterns are duplicates

**Files modified:** `cmd/ux_friendly_errors.go`
**Commit:** `399d68c7`
**Applied fix:** Removed the `"missing flag --"` entry from `errorPatternMap`. The `"flag --"` pattern already matches all errors containing `"missing flag --"` since it is a substring, making the more specific entry unreachable dead code.

### WR-04: First-run marker file written with world-readable 0644 permissions

**Files modified:** `cmd/ux_firstrun.go`
**Commit:** `e51143db`
**Applied fix:** Changed `os.WriteFile(markerPath, []byte(""), 0644)` to `os.WriteFile(markerPath, []byte(""), 0600)` for owner-only read/write permissions, consistent with the project's protective posture for data directory files.

### WR-05: Progress bar description never updated during ceremony

**Files modified:** `cmd/ux_progress.go`
**Commit:** `922f0fea`
**Applied fix:** Added `p.bar.Describe(name)` call in `Advance()` when running in TTY mode. The progressbar description now updates to show the current step name (e.g., "Verification", "Housekeeping") instead of staying stuck on "Starting ceremony...".

---

_Fixed: 2026-04-29T21:40:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
