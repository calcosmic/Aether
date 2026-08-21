---
phase: 76-ux-improvements
reviewed: 2026-04-29T00:00:00Z
depth: standard
files_reviewed: 14
files_reviewed_list:
  - cmd/codex_build.go
  - cmd/codex_continue.go
  - cmd/codex_visuals.go
  - cmd/status.go
  - cmd/status_ux_test.go
  - cmd/ux_firstrun.go
  - cmd/ux_firstrun_test.go
  - cmd/ux_friendly_errors.go
  - cmd/ux_friendly_errors_test.go
  - cmd/ux_progress.go
  - cmd/ux_progress_test.go
  - cmd/helpers.go
  - cmd/root.go
  - go.mod
findings:
  critical: 1
  warning: 5
  info: 2
  total: 8
status: issues_found
---

# Phase 76: Code Review Report

**Reviewed:** 2026-04-29T00:00:00Z
**Depth:** standard
**Files Reviewed:** 14
**Status:** issues_found

## Summary

Reviewed Phase 76 (UX Improvements) which adds four UX features: first-run welcome banner detection, friendly error messages with pattern matching, ceremony progress bars (TTY/non-TTY), and dashboard warnings with extended next-step suggestions. The implementation is well-structured with clean separation into dedicated files (`ux_firstrun.go`, `ux_friendly_errors.go`, `ux_progress.go`). Tests are comprehensive and all pass. However, there is one critical issue with the friendly error pattern matching where a very broad "json" pattern will incorrectly match any error message containing "json", and there are several warning-level issues around progress bar step labeling and missing step name usage.

## Critical Issues

### CR-01: Overly broad "json" error pattern matches any error containing "json"

**File:** `cmd/ux_friendly_errors.go:64-70`
**Issue:** The `errorPatternMap` contains a catch-all pattern `"json"` that will match any error message containing the substring "json", regardless of context. This will produce misleading friendly error messages for any error that happens to mention JSON (e.g., a filename containing "json", a configuration key, a field name in an error). For example, an error like `"failed to load settings from custom_json_handler.go"` would be incorrectly mapped to the "data file corrupted" explanation, which is wrong and could mislead users into corrupting their data files.

```go
{
    Pattern:    "json",
    Explanation: "Aether's data file is corrupted or was modified outside of Aether.",
    NextSteps: []string{
        "Run `aether patrol` for diagnostics.",
        "Check `.aether/data/COLONY_STATE.json` for syntax errors.",
    },
},
```

**Fix:** Replace the bare `"json"` pattern with a more specific one that matches actual Go JSON parsing errors. The test at line 167-175 (`TestFriendlyErrorPatternMatchJSONParse`) tests with `"json: cannot unmarshal"` which suggests the intended match is Go's `encoding/json` error prefix.

```go
{
    Pattern:    "json:",
    Explanation: "Aether's data file is corrupted or was modified outside of Aether.",
    NextSteps: []string{
        "Run `aether patrol` for diagnostics.",
        "Check `.aether/data/COLONY_STATE.json` for syntax errors.",
    },
},
```

This matches Go's standard JSON error format (`json: cannot unmarshal`, `json: unexpected token`, etc.) without catching arbitrary strings containing "json".

## Warnings

### WR-01: Progress bar step names are defined but never used during Advance

**File:** `cmd/codex_build.go:267-268` and `cmd/codex_continue.go:383`
**Issue:** Both `runCodexBuildWithOptions` and `runCodexContinue` define named step arrays for the progress tracker (`buildSteps := []string{"Prepare", "Context", "Dispatch", "Verify", "Complete"}` and `continueSteps := []string{"Verification", "Housekeeping", "Advance", "Complete"}`), but the `Advance()` calls at lines 287, 301, 309, 343, 363-364 (build) and 393, 636, 687 (continue) pass hardcoded strings that do NOT correspond to the step array names. For example, `progress.Advance("Prepare")` advances to step 1, but the step array says step 1 is "Prepare" -- these happen to match. However, `progress.Advance("Context")` at line 301 advances to step 2, which is also "Context" in the array. The problem is the step names passed to `Advance()` are decorative -- they are not validated against the `steps` array. If someone adds a step to the array but forgets to update the `Advance()` calls, or vice versa, the progress bar will show misleading state. More importantly, the `stepName` parameter in `Advance()` is never displayed -- it's completely unused in the TTY path (progressbar handles description separately) and only used as a label in the non-TTY path, so it silently diverges from the declared step list.

**Fix:** Either remove the `steps` field from `ceremonyProgress` and the `NewCeremonyProgress` constructor parameter (since step names are never referenced), or update `Advance()` to validate the step name against the steps array and use the array's name instead of the caller-supplied name. The simplest fix is to make the step name optional and fall back to the array:

```go
func (p *ceremonyProgress) Advance(stepName string) {
    p.current++
    name := stepName
    if name == "" && p.current <= len(p.steps) {
        name = p.steps[p.current-1]
    }
    // ... use name instead of stepName
}
```

### WR-02: Ceremony progress ChangeMax called every Advance is redundant

**File:** `cmd/ux_progress.go:46`
**Issue:** `p.bar.ChangeMax(len(p.steps))` is called on every `Advance()`, but the max never changes (it's always `len(p.steps)`). This is a no-op on every call after the first. While harmless, it suggests the design anticipates dynamic step counts that don't exist.

**Fix:** Remove the `ChangeMax` call from `Advance()` and set the max once in the constructor:

```go
func newCeremonyProgress(steps []string, out io.Writer) *ceremonyProgress {
    p := &ceremonyProgress{
        steps: steps,
        start: time.Now(),
        tty:   isTerminalWriter(out),
        out:   out,
    }
    if p.tty {
        p.bar = progressbar.NewOptions(len(steps),
            progressbar.OptionSetDescription("Starting ceremony..."),
            progressbar.OptionSetWriter(out),
            progressbar.OptionSetWidth(40),
            progressbar.OptionSetRenderBlankState(true),
        )
    }
    return p
}
```

And in `Advance()`:

```go
func (p *ceremonyProgress) Advance(stepName string) {
    p.current++
    if p.tty && p.bar != nil {
        _ = p.bar.Set(p.current)
    } else {
        fmt.Fprintf(p.out, "  Step %d/%d: %s (%s)\n", p.current, len(p.steps), stepName, time.Since(p.start).Round(time.Second))
    }
}
```

### WR-03: Friendly error patterns "flag --" and "missing flag --" are duplicates

**File:** `cmd/ux_friendly_errors.go:36-47`
**Issue:** Both `"flag --"` and `"missing flag --"` patterns produce identical explanations and next steps. The `"flag --"` pattern will match everything `"missing flag --"` matches (since it's a substring), making the more specific entry unreachable. This is dead code.

```go
{
    Pattern:    "flag --",
    Explanation: "This command needs more information to run. Check the required flags and try again.",
    NextSteps: []string{
        "Run `aether <command> --help` to see available flags.",
    },
},
{
    Pattern:    "missing flag --",
    Explanation: "This command needs more information to run. Check the required flags and try again.",
    NextSteps: []string{
        "Run `aether <command> --help` to see available flags.",
    },
},
```

**Fix:** Remove the `"missing flag --"` entry since `"flag --"` already covers it. Or, if the intent is to have different guidance for missing vs. other flag issues, differentiate the explanations.

### WR-04: First-run marker file written with 0644 permissions

**File:** `cmd/ux_firstrun.go:34`
**Issue:** The `.welcomed` marker file is written with mode `0644` (world-readable). While this is a minor concern for a simple marker file, the Aether project's own documentation emphasizes protecting `.aether/data/` paths. The marker file is written to the data directory and could leak information about Aether usage if the directory is on a shared filesystem.

```go
_ = os.WriteFile(markerPath, []byte(""), 0644)
```

**Fix:** Use `0600` permissions consistent with the protective posture for data directory files:

```go
_ = os.WriteFile(markerPath, []byte(""), 0600)
```

### WR-05: Progress bar description never updated during ceremony

**File:** `cmd/ux_progress.go:33`
**Issue:** The progress bar is initialized with `progressbar.OptionSetDescription("Starting ceremony...")` but this description is never updated during the ceremony. As steps advance, the user sees "Starting ceremony..." throughout the entire build or continue process, which becomes inaccurate once execution begins. The `stepName` parameter in `Advance()` would be a natural description but is not wired to the progressbar.

**Fix:** Update the progressbar description on each `Advance()`:

```go
func (p *ceremonyProgress) Advance(stepName string) {
    p.current++
    if p.tty && p.bar != nil {
        p.bar.ChangeMax(len(p.steps))
        if stepName != "" {
            p.bar.Describe(stepName)
        }
        _ = p.bar.Set(p.current)
    } else {
        fmt.Fprintf(p.out, "  Step %d/%d: %s (%s)\n", p.current, len(p.steps), stepName, time.Since(p.start).Round(time.Second))
    }
}
```

## Info

### IN-01: Unused `steps` field and `Steps()` method on ceremonyProgress

**File:** `cmd/ux_progress.go:17,63-65`
**Issue:** The `steps` field is populated in the constructor and exposed via `Steps()`, but the step names are never used internally -- `Advance()` takes a `stepName` parameter from the caller and ignores the `steps` array. The `Steps()` method and the exported `NewCeremonyProgress` constructor accept a `steps` parameter that serves no functional purpose.

**Fix:** Consider removing the `steps` field and `Steps()` method, or refactor `Advance()` to use the step array as the source of truth for step names (removing the `stepName` parameter).

### IN-02: `renderFriendlyError` receives `rawMessage` parameter but never uses it

**File:** `cmd/ux_friendly_errors.go:87`
**Issue:** The `rawMessage` parameter is accepted but never referenced in the function body. The raw error message could be useful for display (e.g., showing it below the friendly explanation), but it is silently ignored.

```go
func renderFriendlyError(entry friendlyError, rawMessage string) string {
    var b strings.Builder
    b.WriteString(renderBanner("\u274C", "Error"))
    b.WriteString(visualDivider)
    b.WriteString(entry.Explanation)
    // rawMessage is never used
```

**Fix:** Either use `rawMessage` in the output (e.g., append it after the explanation for debugging context) or remove the parameter.

---

_Reviewed: 2026-04-29T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
