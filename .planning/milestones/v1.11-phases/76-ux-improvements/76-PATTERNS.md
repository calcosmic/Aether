# Phase 76: UX Improvements - Pattern Map

**Mapped:** 2026-04-29
**Files analyzed:** 8 (4 new, 4 modified)
**Analogs found:** 8 / 8

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `cmd/ux_firstrun.go` | utility/middleware | request-response | `cmd/state_load.go` (store check + file-based state) | role-match |
| `cmd/ux_firstrun_test.go` | test | transform | `cmd/helpers_test.go` (stdout capture + package-level override) | exact |
| `cmd/ux_friendly_errors.go` | utility | transform | `cmd/helpers.go` (error rendering, outputError interception) | exact |
| `cmd/ux_friendly_errors_test.go` | test | transform | `cmd/helpers_test.go` (stderr capture + pattern validation) | exact |
| `cmd/ux_progress.go` | utility | streaming | `cmd/codex_visuals.go` (visual primitives, terminal detection) | role-match |
| `cmd/ux_progress_test.go` | test | transform | `cmd/swarm_display_test.go` (temp dir setup + writer capture) | exact |
| `cmd/status.go` | controller (modified) | request-response | `cmd/status.go` (self -- existing dashboard) | self-modify |
| `cmd/helpers.go` | utility (modified) | transform | `cmd/helpers.go` (self -- existing error rendering) | self-modify |
| `cmd/root.go` | config (modified) | request-response | `cmd/root.go` (self -- PersistentPreRunE hook) | self-modify |

## Pattern Assignments

### `cmd/ux_firstrun.go` (utility/middleware, request-response)

**Analog:** `cmd/state_load.go` (lines 17-37)

This file introduces first-run detection via a marker file and renders a welcome banner. It hooks into the root command's `PersistentPreRunE`. The closest analog is `state_load.go` because it demonstrates the pattern of checking store/file state and branching on result (no colony vs colony exists).

**Imports pattern** -- follow `cmd/state_load.go` lines 1-13:
```go
package cmd

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/calcosmic/Aether/pkg/storage"
)
```

**File-check + branch pattern** from `cmd/state_load.go` lines 17-31:
```go
func loadActiveColonyState() (colony.ColonyState, error) {
    if store == nil {
        return colony.ColonyState{}, fmt.Errorf("no store initialized")
    }
    state, err := loadColonyStateWithCompatibilityRepair()
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            return colony.ColonyState{}, errNoColonyInitialized
        }
        return colony.ColonyState{}, fmt.Errorf("failed to load colony state: %w", err)
    }
    // ...
}
```

**Visual rendering pattern** -- use `renderBanner()` from `cmd/codex_visuals.go` line 259-261:
```go
func renderBanner(emoji, title string) string {
    return fmt.Sprintf("━━ %s %s ━━\n", emoji, spacedTitle(title))
}
```

**Next-up suggestions pattern** from `cmd/codex_visuals.go` line 304-318:
```go
func renderNextUp(primary string, alternatives ...string) string {
    var b strings.Builder
    b.WriteString("\n")
    b.WriteString(renderBanner(commandEmoji("next-up"), "Next Up"))
    if strings.TrimSpace(primary) != "" {
        b.WriteString(primary)
        b.WriteString("\n")
    }
    for _, alt := range alternatives {
        alt = strings.TrimSpace(alt)
        if alt == "" {
            continue
        }
        b.WriteString(alt)
        b.WriteString("\n")
    }
    return b.String()
}
```

**Visual-mode gating** -- always gate on `shouldRenderVisualOutput()` from `cmd/codex_visuals.go` lines 181-195:
```go
func shouldRenderVisualOutput(w io.Writer) bool {
    mode := strings.ToLower(strings.TrimSpace(os.Getenv("AETHER_OUTPUT_MODE")))
    switch mode {
    case "json":
        return false
    case "visual", "human", "pretty":
        return true
    }
    if os.Getenv("AETHER_FORCE_VISUAL") == "1" {
        return true
    }
    return isTerminalWriter(w)
}
```

---

### `cmd/ux_firstrun_test.go` (test, transform)

**Analog:** `cmd/helpers_test.go` (lines 1-80)

Uses the standard test pattern: override `stdout`/`stderr` package-level variables with `bytes.Buffer`, defer restore, assert on captured output.

**Test scaffold pattern** from `cmd/helpers_test.go` lines 1-36:
```go
package cmd

import (
    "bytes"
    "os"
    "strings"
    "testing"
)

func TestOutputOK(t *testing.T) {
    var buf bytes.Buffer
    stdout = &buf
    defer func() { stdout = os.Stdout }()

    outputOK("test")

    got := strings.TrimSpace(buf.String())
    expected := `{"ok":true,"result":"test"}`

    if got != expected {
        t.Errorf("outputOK(\"test\") = %q, want %q", got, expected)
    }
}
```

**Temp dir + data dir pattern** from `cmd/swarm_display_test.go` lines 12-29:
```go
func setupSwarmDisplayTest(t *testing.T) string {
    t.Helper()
    tmpDir := t.TempDir()
    dataDir := tmpDir + "/.aether/data"
    if err := os.MkdirAll(dataDir, 0755); err != nil {
        t.Fatalf("failed to create temp data dir: %v", err)
    }
    os.Setenv("AETHER_ROOT", tmpDir)
    t.Cleanup(func() { os.Unsetenv("AETHER_ROOT") })
    return dataDir
}
```

---

### `cmd/ux_friendly_errors.go` (utility, transform)

**Analog:** `cmd/helpers.go` (lines 32-57, 170-184)

This is the core error pattern map. It intercepts at `renderVisualError()` and adds friendly messages. The analog is `helpers.go` itself because that is the modification target.

**Current error envelope pattern** from `cmd/helpers.go` lines 32-57:
```go
func outputError(code int, message string, details interface{}) {
    if shouldRenderVisualOutput(stderr) {
        fmt.Fprint(stderr, renderVisualError(message, details))
        return
    }
    envelope := struct {
        OK      bool        `json:"ok"`
        Error   string      `json:"error"`
        Code    int         `json:"code"`
        Details interface{} `json:"details,omitempty"`
    }{
        OK:    false,
        Error: message,
        Code:  code,
    }
    if details != nil {
        envelope.Details = details
    }
    payload, err := json.Marshal(envelope)
    if err != nil {
        msgJSON, _ := json.Marshal(message)
        fmt.Fprintf(stderr, "{\"ok\":false,\"error\":%s,\"code\":%d}\n", string(msgJSON), code)
        return
    }
    fmt.Fprintf(stderr, "%s\n", string(payload))
}
```

**Current visual error rendering** from `cmd/helpers.go` lines 170-184:
```go
func renderVisualError(message string, details interface{}) string {
    var b strings.Builder
    b.WriteString(renderBanner("Error", "Error"))
    b.WriteString(visualDivider)
    b.WriteString(strings.TrimSpace(message))
    b.WriteString("\n")
    if details != nil {
        detailText := strings.TrimSpace(fmt.Sprint(details))
        if detailText != "" && detailText != "<nil>" {
            b.WriteString(detailText)
            b.WriteString("\n")
        }
    }
    return b.String()
}
```

**Error sentinel pattern** from `cmd/state_load.go` line 15:
```go
var errNoColonyInitialized = errors.New("no colony initialized")
```

**Known error message patterns** to match against (from grep of `outputError` call sites):
- `"no colony initialized"` -- from `errNoColonyInitialized` in `cmd/state_load.go:15`
- `"flag --%s is required"` -- from `cmd/helpers.go:73` (`mustGetString`)
- `"missing flag --%s"` -- from `cmd/helpers.go:69`
- `"failed to initialize store"` -- from `cmd/root.go:177`
- `"failed to load colony state"` -- from `cmd/state_load.go:27`
- `"No colony initialized"` -- from `cmd/state_load.go:127` (`colonyStateLoadMessage`)

---

### `cmd/ux_friendly_errors_test.go` (test, transform)

**Analog:** `cmd/helpers_test.go` (lines 63-80)

Same pattern as first-run tests but testing stderr capture and error message transformation.

**Error test pattern** from `cmd/helpers_test.go` lines 63-80:
```go
func TestOutputError(t *testing.T) {
    var buf bytes.Buffer
    stderr = &buf
    defer func() { stderr = os.Stderr }()

    outputError(1, "fail", nil)

    got := strings.TrimSpace(buf.String())
    expected := `{"ok":false,"error":"fail","code":1}`

    if got != expected {
        t.Errorf("outputError(1, \"fail\", nil) = %q, want %q", got, expected)
    }
}
```

---

### `cmd/ux_progress.go` (utility, streaming)

**Analog:** `cmd/codex_visuals.go` (visual primitives + terminal detection)

This file wraps progressbar/v3 with ceremony-step semantics. The analog is `codex_visuals.go` because it provides the visual rendering infrastructure (banners, stage markers, terminal detection, ANSI gating).

**Terminal detection** from `cmd/codex_visuals.go` lines 197-208:
```go
func isTerminalWriter(w io.Writer) bool {
    file, ok := w.(*os.File)
    if !ok {
        return false
    }
    info, err := file.Stat()
    if err != nil {
        return false
    }
    return (info.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}
```

**Stage marker primitive** from `cmd/codex_visuals.go` lines 274-280:
```go
func renderStageMarker(title string) string {
    title = strings.TrimSpace(title)
    if title == "" {
        return ""
    }
    return "-- " + title + " --\n"
}
```

**Visual divider constant** from `cmd/codex_visuals.go` line 15:
```go
const visualDivider = "----------------------------------------\n"
```

**Non-TTY fallback pattern** -- ceremony commands currently use `renderStageMarker()` for step output (plain text, no ANSI). The progress wrapper should fall back to this same pattern when not in a TTY.

---

### `cmd/ux_progress_test.go` (test, transform)

**Analog:** `cmd/swarm_display_test.go` (lines 1-59)

Demonstrates the temp dir + env var + writer capture pattern used across cmd tests.

**Test helper pattern** from `cmd/swarm_display_test.go` lines 12-29 (see first-run test above).

For progress tests, the key pattern is writing to a `bytes.Buffer` (non-TTY) and asserting plain-text step markers are emitted. For TTY tests, mock the writer or test the struct state directly without calling the underlying progressbar library.

---

### `cmd/status.go` (modified -- dashboard redesign)

**Analog:** `cmd/status.go` (self-modify, lines 56-305)

The redesign target. Currently renders sections in this order: Banner, Goal, Version, Signals, Progress, Constraints, Instincts, Flags, Scope, Milestone, Depth, Granularity, Parallel, Proof, Memory Health, Review Findings, Pheromones, Spawn Activity, Active Workers, Recent Outcomes, Recovery, Recent Instincts, State, Next Up.

**Current dashboard structure** from `cmd/status.go` lines 56-305:
```go
func renderDashboard(state colony.ColonyState, s *storage.Store) string {
    var b strings.Builder
    // Banner
    b.WriteString(renderBanner(commandEmoji("status"), "Colony Status"))
    b.WriteString(visualDivider)
    // Goal (truncated to 60 chars)
    // ... [sections in order]
    // Next Up (at bottom)
    primary, alternatives := workflowSuggestionsForState(state)
    alternatives = append(alternatives, `Run `+"`aether proof`"+`...`)
    b.WriteString(renderNextUp(primary, alternatives...))
    return b.String()
}
```

**Existing next-step suggestion engine** from `cmd/codex_visuals.go` lines 377-421:
```go
func workflowSuggestionsForState(state colony.ColonyState) (string, []string) {
    state = normalizeLegacyColonyState(state)
    if colonyNeedsEntomb(state) {
        return `Run ` + "`aether entomb`" + ` to archive this sealed colony into chambers.`,
            []string{`Run ` + "`aether init \"next goal\"`" + `...`}
    }
    if state.Paused {
        return `Run ` + "`aether resume`" + `...`, ...
    }
    if len(state.Plan.Phases) == 0 {
        return `Run ` + "`aether discuss`" + `...`, ...
    }
    switch state.State {
    case colony.StateEXECUTING, colony.StateBUILT:
        // ...
        return `Run ` + "`aether continue`" + `...`, ...
    case colony.StateCOMPLETED:
        return `Run ` + "`aether entomb`" + `...`, nil
    default:
        return fmt.Sprintf("Run `aether build %d`...", nextPhase), ...
    }
}
```

**Colony state types needed for warnings** from `cmd/state_load.go` and `pkg/colony/`:
- `state.State` -- lifecycle state (READY, EXECUTING, BUILT, COMPLETED, etc.)
- `state.Plan.Phases[i].Status` -- PhaseCompleted, PhaseFailed, etc.
- `state.Milestone` -- maturity milestone string
- `state.Paused` -- paused flag

**go-pretty table pattern** from `cmd/status.go` lines 595-607:
```go
func renderMemoryHealthTable(b *strings.Builder, s *storage.Store) {
    summary := loadMemoryHealthSummary(s)
    t := table.NewWriter()
    t.AppendHeader(table.Row{"Metric", "Count", "Last Updated"})
    t.AppendRow(table.Row{"Wisdom Entries", summary.WisdomTotal, formatTimestamp(summary.LastLearning)})
    // ...
    t.SetStyle(table.StyleRounded)
    b.WriteString(t.Render() + "\n")
}
```

---

### `cmd/helpers.go` (modified -- extend renderVisualError)

**Analog:** `cmd/helpers.go` (self-modify, lines 170-184)

The modification target for friendly error interception. See `ux_friendly_errors.go` section above for the current code.

**Modification approach:** Call `friendlyErrorFor(message)` at the top of `renderVisualError()`. If a match is found, render the friendly version instead of the raw message. If no match, append the generic hint (D-05).

---

### `cmd/root.go` (modified -- add first-run check to PersistentPreRunE)

**Analog:** `cmd/root.go` (self-modify, lines 168-183)

The modification target for first-run detection hook.

**Current PersistentPreRunE** from `cmd/root.go` lines 168-183:
```go
PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
    // Skip store initialization for commands that don't need it.
    if skipStoreInit(cmd) {
        return nil
    }
    dataDir := storage.ResolveDataDir(context.Background())
    s, err := storage.NewStore(dataDir)
    if err != nil {
        return fmt.Errorf("failed to initialize store: %w", err)
    }
    store = s
    tracer = trace.NewTracer(s)
    return nil
},
```

**Modification approach:** After store init succeeds (after line 181), call `checkAndEmitFirstRun(dataDir)` which checks the marker file and emits the welcome banner if appropriate. Gate on `shouldRenderVisualOutput(stdout)` so JSON mode is unaffected.

**skipStoreInit guard** from `cmd/root.go` lines 187-195:
```go
func skipStoreInit(cmd *cobra.Command) bool {
    for c := cmd; c != nil; c = c.Parent() {
        switch c.Name() {
        case "completion", "version", "help":
            return true
        }
    }
    return false
}
```

---

## Shared Patterns

### Visual-Mode Gating
**Source:** `cmd/codex_visuals.go` lines 181-195 (`shouldRenderVisualOutput`)
**Apply to:** All new files that produce visual output (ux_firstrun.go, ux_friendly_errors.go, ux_progress.go, status.go modifications)

Every visual rendering call must be gated behind `shouldRenderVisualOutput()`. JSON mode must remain clean -- no welcome banners, no friendly error formatting, no progress bars in JSON output.

### Package-Level Writer Override (Testing)
**Source:** `cmd/root.go` lines 155-156 (`var stdout io.Writer = os.Stdout`)
**Apply to:** All test files (ux_firstrun_test.go, ux_friendly_errors_test.go, ux_progress_test.go)

Tests override `stdout`/`stderr` with `bytes.Buffer`, then defer restore. This is the established pattern across all cmd tests.

### Banner + Divider Rendering
**Source:** `cmd/codex_visuals.go` lines 15, 259-261 (`visualDivider`, `renderBanner`)
**Apply to:** ux_firstrun.go (welcome banner), ux_friendly_errors.go (friendly error banner), status.go (any new dashboard sections)

All visual sections use `renderBanner()` for headers and `visualDivider` for separators. Consistent visual language.

### Store-Based File Operations
**Source:** `cmd/state_load.go` lines 17-37, `cmd/helpers.go` lines 159-168 (`hubStore`, `loadActiveColonyState`)
**Apply to:** ux_firstrun.go (marker file check/create)

The marker file (`.welcomed`) lives in the same `.aether/data/` directory that the store manages. Use `storage.ResolveDataDir()` for the path, but the marker file itself is a simple `os.Stat`/`os.WriteFile` -- it does not need the store's JSON layer.

### Error Sentinels
**Source:** `cmd/state_load.go` line 15 (`var errNoColonyInitialized = errors.New(...)`)
**Apply to:** ux_friendly_errors.go (pattern matching against sentinel errors)

Error pattern matching should use `strings.Contains(message, pattern)` for substring matching against known error message strings. The sentinels in state_load.go provide the canonical error strings to match.

## No Analog Found

All files have close analogs in the codebase. No files require the planner to fall back to RESEARCH.md patterns exclusively.

| File | Reason All Analogs Found |
|------|------------------------|
| `cmd/ux_firstrun.go` | `state_load.go` provides file-check + branch pattern; `codex_visuals.go` provides rendering primitives |
| `cmd/ux_friendly_errors.go` | `helpers.go` IS the modification target; pattern map is a pure extension |
| `cmd/ux_progress.go` | `codex_visuals.go` provides terminal detection + stage markers + visual gating |
| `cmd/ux_progress_test.go` | `helpers_test.go` and `swarm_display_test.go` cover both stdout capture and temp dir patterns |

## Metadata

**Analog search scope:** `cmd/` directory (Go source and test files), `pkg/colony/` (state types)
**Files scanned:** 8 source files read in full, 4 files searched via grep, 90+ test files listed
**Pattern extraction date:** 2026-04-29
