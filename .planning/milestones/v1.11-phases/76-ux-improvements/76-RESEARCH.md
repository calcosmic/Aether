# Phase 76: UX Improvements - Research

**Researched:** 2026-04-29
**Domain:** Go CLI UX (onboarding, error messages, progress indicators, dashboard redesign)
**Confidence:** HIGH

## Summary

Phase 76 adds four UX improvements to the Aether Go CLI: a first-run welcome banner, friendly error messages with next-step suggestions, progress indicators for build/continue ceremonies, and a redesigned status dashboard that surfaces actionable information. The codebase already has strong visual rendering primitives in `cmd/codex_visuals.go` (banners, dividers, progress bars, stage markers) and a clean interception point for friendly errors at `renderVisualError()` in `cmd/helpers.go`. The status dashboard in `cmd/status.go` already has a `workflowSuggestionsForState()` function that provides rule-based next-step suggestions -- this is the foundation for the redesign.

The only new dependency needed is a lightweight Go progress bar library. Two candidates stand out: `github.com/schollz/progressbar/v3` (v3.19.0, zero transitive deps, actively maintained) and `github.com/cheggaaa/pb/v3` (v3.1.7, zero transitive deps). Both are viable; progressbar/v3 has a simpler API surface and more recent releases, making it the stronger choice for this use case.

**Primary recommendation:** Implement all four requirements using existing visual primitives, add progressbar/v3 as the sole new dependency, and extend `renderVisualError()` and `renderDashboard()` rather than building parallel systems.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**First-Run Experience (UX-01)**
- **D-01:** Show a welcome banner with 2-3 quick-start commands when a user runs Aether for the first time. Not a tutorial -- just enough context to get moving. Same message on any command that detects first-run state.
- **D-02:** First-run detection via a local marker file (e.g., `.aether/.welcomed`) in the repo's `.aether/data/` directory. Created on first display, checked on every command run. Survives colony init/entomb. Simple, no extra deps, no coupling to hub.
- **D-03:** The welcome banner should appear on any command that detects no colony and no marker file -- not just `status`. This way users get guidance regardless of which command they try first.

**Error Messages (UX-02)**
- **D-04:** Build an error pattern map: common error patterns (no colony initialized, missing required flags, file not found, store errors, permission errors) map to plain-language explanations + suggested next steps. When an error matches a pattern, render the friendly version instead of the raw error.
- **D-05:** Internal/unexpected errors stay technical but get a generic hint appended: "try /ant-patrol for diagnostics or /ant-status to check colony health." No false suggestions for errors we don't understand.
- **D-06:** The pattern map approach keeps the friendly error logic centralized and cleanly separated from existing error handling. Individual commands don't need to change -- the map intercepts at the `outputError`/`renderVisualError` layer.

**Progress Feedback (UX-03)**
- **D-07:** Add a progress bar with elapsed time and estimated completion for build and continue ceremonies. Users see where they are in the flow and how long each step takes.
- **D-08:** Use a lightweight third-party Go progress library (e.g., `progressbar`, `uiprogress`, or similar). This is the one exception to the zero-new-deps principle -- the UX improvement justifies a focused dependency. Pick the lightest library that gives us progress bar + timing + graceful fallback in non-terminal environments (CI, pipes).
- **D-09:** Progress applies to ceremony steps (build-wave, build-verify, continue-verify, continue-advance, etc.). Non-ceremony commands don't get progress bars -- they're fast enough.

**Status Actionability (UX-04)**
- **D-10:** Redesign the status dashboard layout to integrate warnings, blockers, and next-step suggestions alongside existing information. Not just appending sections -- a cohesive redesign that puts actionable information front and center.
- **D-11:** Rule-based next-step suggestions, deterministic from colony state:
  - Sealed colony -> suggest `/ant-entomb`
  - All phases complete -> suggest `/ant-seal`
  - Phase failed -> suggest `/ant-build N` (rebuild)
  - Post-build (completed but not continued) -> suggest `/ant-continue`
  - No colony -> suggest `/ant-init "goal"` or `/ant-lay-eggs`
- **D-12:** Warning indicators for: failed phases (with fix suggestion), stale state (last activity > 7 days), unacknowledged midden entries, pheromone signals approaching expiry. All deterministic from colony state -- no guessing.
- **D-13:** Warnings and next steps are visual-mode only (they don't appear in JSON output mode). JSON consumers can derive the same information from the existing structured data.

### Claude's Discretion
- Exact welcome banner copy and quick-start command selection
- Which specific errors to include in the pattern map (beyond the obvious common ones)
- Third-party progress library selection (pick the lightest suitable option)
- Dashboard redesign layout details (section ordering, visual formatting)
- Warning threshold values (staleness period, pheromone expiry proximity)
- Whether progress bar shows estimated time remaining or just elapsed time

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.

</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| UX-01 | First-run experience provides clear guidance for new users | D-01/D-02/D-03: marker file pattern, PersistentPreRunE hook, renderBanner primitive |
| UX-02 | Error messages explain what happened and suggest next steps | D-04/D-05/D-06: renderVisualError interception, error pattern map architecture |
| UX-03 | Build and continue ceremonies provide progress feedback | D-07/D-08/D-09: progressbar/v3 library, ceremony step structure in codex_build.go |
| UX-04 | Status command surfaces actionable information | D-10/D-11/D-12/D-13: renderDashboard redesign, workflowSuggestionsForState foundation |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| First-run detection | Go CLI (root command PersistentPreRunE) | -- | Filesystem check, must happen before any command runs |
| Welcome banner rendering | Go CLI (cmd/) | -- | Pure visual output, same layer as all other rendering |
| Error pattern matching | Go CLI (cmd/helpers.go) | -- | Intercepts at renderVisualError, centralized |
| Progress bars | Go CLI (cmd/) | -- | Ceremony commands own their execution flow |
| Status dashboard | Go CLI (cmd/status.go) | -- | Single command, pure rendering logic |
| Warning indicators | Go CLI (cmd/status.go) | pkg/colony | Reads colony state, pheromone data, midden data |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/schollz/progressbar/v3 | v3.19.0 | Progress bars with timing for ceremonies | Zero transitive deps, simple API, actively maintained (2025-12-26), auto-detects non-TTY |

### Existing (no changes needed)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/spf13/cobra | v1.10.2 | CLI framework | Already in use, PersistentPreRunE for first-run hook |
| github.com/jedib0t/go-pretty/v6 | v6.7.8 | Table rendering | Already used by status dashboard |
| github.com/calcosmic/Aether/pkg/colony | -- | Colony state types | ColonyState struct, pheromone expiry, state lifecycle |
| github.com/calcosmic/Aether/pkg/storage | -- | JSON file storage | Marker file read/write via existing store |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| progressbar/v3 | cheggaaa/pb/v3 | pb/v3 is more feature-rich (spinner, multi-bar) but heavier API surface. Both have zero transitive deps. progressbar/v3 is simpler for step-based progress. |
| progressbar/v3 | Hand-rolled ANSI | Hand-rolled avoids the dependency but adds ~200 lines of edge-case code (non-TTY fallback, Windows compat, resize handling). Not worth it. |

**Installation:**
```bash
go get github.com/schollz/progressbar/v3@v3.19.0
```

**Version verification:**
```bash
go list -m -versions github.com/schollz/progressbar/v3
# Latest: v3.19.0 (2025-12-26) -- [VERIFIED: go module registry]
```

## Architecture Patterns

### System Architecture Diagram

```
User runs any command
        |
        v
  PersistentPreRunE (root.go)
        |
        +-- Check .aether/data/.welcomed marker
        |       |
        |       +-- Missing & no colony -> emitWelcomeBanner() -> create marker
        |       +-- Present -> skip
        |
        v
  Command execution
        |
        +-- Error path -> outputError() -> renderVisualError()
        |                                       |
        |                                       +-- friendlyErrorForPattern(message)
        |                                       |       |
        |                                       |       +-- Match found -> friendly message + next steps
        |                                       |       +-- No match -> raw error + generic hint
        |                                       |
        |                                       v
        |                                   stderr (visual or JSON)
        |
        +-- Build/Continue ceremony
        |       |
        |       +-- Step 1: Context    -> progress(1/5, "Context")
        |       +-- Step 2: Tasks     -> progress(2/5, "Tasks")
        |       +-- Step 3: Dispatch  -> progress(3/5, "Dispatch")
        |       +-- Step 4: Verify    -> progress(4/5, "Verification")
        |       +-- Step 5: Housekeep -> progress(5/5, "Housekeeping")
        |       |
        |       +-- Non-TTY? -> plain text step markers (no ANSI)
        |
        +-- Status command -> renderDashboard()
                |
                +-- Warnings section (stale, failed, midden, pheromones)
                +-- Next Steps section (rule-based from colony state)
                +-- Existing sections (progress, memory, signals, etc.)
```

### Recommended Project Structure

```
cmd/
  ux_firstrun.go          # First-run detection + welcome banner
  ux_firstrun_test.go     # Tests for marker file logic, banner output
  ux_friendly_errors.go   # Error pattern map + friendly rendering
  ux_friendly_errors_test.go
  ux_progress.go          # Progress bar wrapper (progressbar/v3 adapter)
  ux_progress_test.go
  # status.go             # MODIFIED: redesign renderDashboard()
  # helpers.go            # MODIFIED: extend renderVisualError()
  # root.go               # MODIFIED: add first-run check to PersistentPreRunE
```

### Pattern 1: Error Pattern Map

**What:** A `map[string]friendlyError` that matches error message substrings to friendly explanations and next-step suggestions. Intercepts at `renderVisualError()`.

**When to use:** Any command that calls `outputError()` automatically benefits. No per-command changes needed.

**Example:**
```go
// Source: [verified from cmd/helpers.go:170-184]
type friendlyError struct {
    Pattern    string   // substring match
    Explanation string  // plain-language what happened
    NextSteps  []string // suggested actions
}

var errorPatternMap = []friendlyError{
    {
        Pattern:    "no colony initialized",
        Explanation: "Aether needs a colony to work with. A colony is a workspace for building toward a specific goal.",
        NextSteps:  []string{
            "Run `aether init \"your goal\"` to start a colony.",
            "Run `aether lay-eggs` first if this repo is brand new.",
        },
    },
    {
        Pattern:    "flag --%s is required",
        Explanation: "This command needs more information to run. Check the required flags and try again.",
        NextSteps:  []string{
            "Run `aether <command> --help` to see available flags.",
        },
    },
    // ... more patterns
}
```

### Pattern 2: First-Run Marker File

**What:** A zero-byte file `.aether/data/.welcomed` that acts as a "seen the welcome" flag. Checked in `PersistentPreRunE` before any command runs.

**When to use:** Once per repo. Survives colony init/entomb because it lives in `.aether/data/` which persists across colony lifecycles.

**Example:**
```go
// Source: [designed from root.go:168-183 PersistentPreRunE pattern]
func checkAndEmitFirstRun(dataDir string) {
    markerPath := filepath.Join(dataDir, ".welcomed")
    if _, err := os.Stat(markerPath); err == nil {
        return // already welcomed
    }

    // Check if colony exists -- if yes, don't show welcome
    if _, err := os.Stat(filepath.Join(dataDir, "COLONY_STATE.json")); err == nil {
        return // has colony, not first run
    }

    // Emit welcome banner to stdout
    fmt.Fprint(stdout, renderWelcomeBanner())
    os.WriteFile(markerPath, []byte(""), 0644)
}
```

### Pattern 3: Ceremony Progress Wrapper

**What:** A thin wrapper around progressbar/v3 that provides step-based progress with elapsed time. Falls back to plain text when not connected to a TTY.

**When to use:** Wrap ceremony step execution in `codex_build.go` and `codex_continue.go`.

**Example:**
```go
// Source: [designed from codex_build.go ceremony structure]
type ceremonyProgress struct {
    bar     *progressbar.ProgressBar
    steps   []string
    current int
    start   time.Time
    tty     bool
}

func newCeremonyProgress(steps []string, out io.Writer) *ceremonyProgress {
    tty := isTerminalWriter(out)
    if !tty {
        return &ceremonyProgress{steps: steps, start: time.Now(), tty: false}
    }
    bar := progressbar.NewOptions(len(steps),
        progressbar.SetDescription("Starting..."),
        progressbar.SetWriter(out),
        progressbar.SetWidth(40),
    )
    return &ceremonyProgress{bar: bar, steps: steps, start: time.Now(), tty: true}
}

func (p *ceremonyProgress) Advance(stepName string) {
    p.current++
    if p.tty {
        p.bar.ChangeMax(len(p.steps))
        elapsed := time.Since(p.start).Round(time.Second)
        _ = p.bar.Set(p.current)
        // Description includes elapsed time
    } else {
        fmt.Fprintf(stdout, "  Step %d/%d: %s (%s)\n", p.current, len(p.steps), stepName, time.Since(p.start).Round(time.Second))
    }
}
```

### Anti-Patterns to Avoid
- **Don't add welcome banner to JSON output mode:** JSON consumers should not get welcome text. Gate on `shouldRenderVisualOutput()`.
- **Don't add progress bars to non-ceremony commands:** Commands like `status`, `pheromones`, `history` are fast. Progress bars on them would feel sluggish, not helpful.
- **Don't guess at next steps for unknown errors:** Only suggest actions for recognized patterns. Unknown errors get the generic hint (D-05).
- **Don't break the existing JSON envelope format:** `outputError()` must continue to produce valid JSON in non-visual mode. Friendly errors only affect the visual rendering path.
- **Don't make the dashboard redesign lose information:** The current dashboard has a lot of useful data. The redesign reorganizes with warnings first, not removes sections.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Terminal progress bars | Custom ANSI escape sequences with carriage returns | progressbar/v3 | Handles non-TTY, Windows, resize, color detection. ~200 lines of edge cases avoided. |
| Terminal detection | Custom isatty check | `shouldRenderVisualOutput()` (existing) | Already implemented and handles AETHER_OUTPUT_MODE env var. |
| Colored output | Custom ANSI color codes | `colorizeCaste()` / `shouldUseANSIColors()` (existing) | Already handles NO_COLOR, CLICOLOR_FORCE, JSON mode. |
| Table rendering | Manual column alignment | go-pretty/v6 (existing) | Already a dependency, handles wrapping and borders. |

**Key insight:** The codebase already has excellent visual primitives. This phase extends them, not replaces them. The only genuinely new capability is the real-time progress bar, which justifies the single new dependency.

## Common Pitfalls

### Pitfall 1: Progress Bar Breaks Piped Output
**What goes wrong:** Progress bars use ANSI escape codes (carriage return `\r`) to overwrite the current line. When stdout is piped (e.g., `aether build 1 | tee log.txt`), this produces garbled output.
**Why it happens:** Progress bars assume a terminal that supports cursor movement.
**How to avoid:** Check `isTerminalWriter(stdout)` before creating a progress bar. In non-TTY mode, emit plain text step markers instead. progressbar/v3 has `progressbar.SetWriter()` and checks internally, but explicit gating at the Aether level is safer for consistency with `shouldRenderVisualOutput()`.
**Warning signs:** CI logs with `[===>      ]` repeated on separate lines instead of one updating line.

### Pitfall 2: First-Run Banner Fires on Every Command
**What goes wrong:** The welcome banner appears on every `aether status` call because the marker file was never created, or was created in the wrong location.
**Why it happens:** Race condition between checking and creating the marker, or the data directory path resolving differently.
**How to avoid:** Use `os.WriteFile()` with `0644` permissions immediately after emitting the banner. The `PersistentPreRunE` hook has access to the store's data directory, which is already resolved by that point.
**Warning signs:** Users report seeing the welcome every time they run any command.

### Pitfall 3: Error Pattern Map Is Too Greedy
**What goes wrong:** A pattern like "file not found" matches legitimate error messages that contain "file not found" as a substring but mean something different.
**Why it happens:** Substring matching without context.
**How to avoid:** Use specific error messages from `loadActiveColonyState()` (e.g., `errNoColonyInitialized.Error()` returns "no colony initialized") rather than generic substrings. For flag errors, match the exact format from `mustGetString()`. Prefer longer, more specific patterns over shorter ones.
**Warning signs:** Friendly error says "no colony initialized" when the actual problem was a missing config file.

### Pitfall 4: Dashboard Redesign Drops Information Users Rely On
**What goes wrong:** Moving warnings to the top pushes familiar information (progress, signals, memory health) further down or removes it.
**Why it happens:** Focusing on "actionable" without realizing users also scan for specific data points.
**How to avoid:** Keep all existing sections. The redesign reorders (warnings first, then the familiar sections) and adds a "Next Steps" section. Don't remove anything from the current dashboard.
**Warning signs:** Users complain they can't find information they used to see immediately.

### Pitfall 5: Progress Bar Library Adds Unexpected Transitive Dependencies
**What goes wrong:** `go get progressbar/v3` pulls in a large dependency tree, bloating the binary.
**Why it happens:** Some progress libraries depend on terminal detection libs, color libs, etc.
**How to avoid:** Both progressbar/v3 and pb/v3 have zero transitive dependencies (verified via `go list -m -json`). progressbar/v3 is the recommendation.
**Warning signs:** `go mod graph` shows more than 1 new line after adding the library.

## Code Examples

Verified patterns from existing codebase:

### Error Interception Point (cmd/helpers.go:170-184)
```go
func renderVisualError(message string, details interface{}) string {
    var b strings.Builder
    b.WriteString(renderBanner("❌", "Error"))
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
// MODIFICATION: Add friendly error lookup BEFORE writing raw message
```

### Existing Next-Step Suggestions (cmd/codex_visuals.go:377-421)
```go
func workflowSuggestionsForState(state colony.ColonyState) (string, []string) {
    // Already implements rule-based suggestions for:
    // - colonyNeedsEntomb -> suggest entomb
    // - paused -> suggest resume
    // - no plan -> suggest discuss/plan
    // - EXECUTING/BUILT -> suggest continue
    // - COMPLETED -> suggest entomb
    // - default -> suggest build N
    // EXTENSION: Add sealed/failed/post-build cases per D-11
}
```

### Terminal Detection (cmd/codex_visuals.go:181-208)
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

### PersistentPreRunE Hook (cmd/root.go:168-183)
```go
PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
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
    // EXTENSION: After store init, check for first-run and emit welcome
}
```

### Banner Rendering Primitive (cmd/codex_visuals.go:259-261)
```go
func renderBanner(emoji, title string) string {
    return fmt.Sprintf("━━ %s %s ━━\n", emoji, spacedTitle(title))
}
```

### Stage Marker Primitive (cmd/codex_visuals.go:274-280)
```go
func renderStageMarker(title string) string {
    title = strings.TrimSpace(title)
    if title == "" {
        return ""
    }
    return "── " + title + " ──\n"
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| No first-run guidance | Marker file + PersistentPreRunE hook | Phase 76 (this phase) | New users get oriented immediately |
| Raw error messages | Pattern map + friendly rendering at renderVisualError | Phase 76 (this phase) | Errors become actionable |
| No ceremony progress | progressbar/v3 step-based progress with timing | Phase 76 (this phase) | Users see where they are in long operations |
| Raw state dump in status | Redesigned dashboard with warnings + next steps | Phase 76 (this phase) | Status becomes actionable |

**No deprecated approaches in this domain -- this is net-new UX work.**

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | progressbar/v3 auto-detects non-TTY and falls back gracefully | Standard Stack | Low risk -- verified zero deps, but need to confirm non-TTY behavior in tests. Mitigated by explicit `isTerminalWriter()` gate in wrapper. |
| A2 | `.aether/data/` directory exists when PersistentPreRunE runs | First-Run Pattern | Low risk -- store init creates it. But if store init fails before first-run check, the welcome won't fire. Mitigated by checking after store init succeeds. |
| A3 | The `renderVisualError()` interception point covers all error output paths | Error Messages | Medium risk -- if any code paths write directly to stderr without going through `outputError()`, they won't get friendly messages. Need to verify with grep. |
| A4 | Pheromone expiry data is available from the colony state for warning indicators | Status Dashboard | Low risk -- `pkg/colony/pheromones.go:26` has `ExpiresAt` field, and `renderPheromoneSummary()` already computes signal lifetime. |
| A5 | The `workflowSuggestionsForState()` function covers all state transitions needed for D-11 | Status Dashboard | Medium risk -- D-11 adds "phase failed" and "post-build" cases that may not be in the current function. Need to extend it. |

## Open Questions (RESOLVED)

RESOLVED: 1. **Progress bar: elapsed time only, or estimated time remaining?** Elapsed time only -- ceremony steps are non-uniform, ETA would be misleading. Per plan Task 2: Advance shows step count and elapsed, no ETA.
   - What we know: D-08 says "elapsed time and estimated completion." progressbar/v3 supports ETA calculation based on step completion rate.
   - What's unclear: Ceremony steps are not uniform in duration (dispatch is fast, verify is slow). ETA may be misleading.
   - Recommendation: Show elapsed time prominently, ETA as secondary. If ETA fluctuates wildly (>50% change between steps), hide it.

RESOLVED: 2. **Welcome banner: should it include the Aether wordmark?** Skip the wordmark. Simple banner with renderBanner + 3 quick-start commands, 5-8 lines max per D-01. Per plan Task 1 Part A.
   - What we know: `renderAetherWordmark()` exists in codex_visuals.go and is used by install/update commands.
   - What's unclear: The wordmark is 6 lines tall. On a first run, it might feel heavy.
   - Recommendation: Skip the wordmark. Use a simple banner with `renderBanner("🐜", "Welcome to Aether")` instead. 5-8 lines max per D-01.

RESOLVED: 3. **How many error patterns should the initial map include?** 7 patterns: no colony, failed to load state, flag --, missing flag --, permission denied, failed to init store, json parse. Per plan Task 1 Part B.
   - What we know: D-04 lists "no colony initialized, missing required flags, file not found, store errors, permission errors" as starting points.
   - What's unclear: The exact error messages that the CLI currently produces for each case.
   - Recommendation: Start with 6-8 patterns covering the most common failure modes identified from `outputError()` call sites. The map is easy to extend later.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go 1.26.1 | Build | Yes | 1.26.1 | -- |
| go-pretty/v6 | Table rendering | Yes | v6.7.8 | -- |
| cobra | CLI framework | Yes | v1.10.2 | -- |
| progressbar/v3 | Progress bars (UX-03) | No | -- | Plain text step markers |

**Missing dependencies with fallback:**
- progressbar/v3: Must be added. Fallback is plain text step markers (non-TTY path).

**Missing dependencies with no fallback:**
- None. All other dependencies are already in go.mod.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) + testify (v1.11.1, existing) |
| Config file | none -- uses Go stdlib |
| Quick run command | `go test ./cmd/ -run "TestUx" -count=0` |
| Full suite command | `go test ./... -count=0` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| UX-01 | Marker file created on first display | unit | `go test ./cmd/ -run "TestFirstRun" -count=0` | No -- Wave 0 |
| UX-01 | Welcome banner not shown after marker exists | unit | `go test ./cmd/ -run "TestFirstRun" -count=0` | No -- Wave 0 |
| UX-01 | Welcome banner not shown when colony exists | unit | `go test ./cmd/ -run "TestFirstRun" -count=0` | No -- Wave 0 |
| UX-01 | Welcome banner not emitted in JSON mode | unit | `go test ./cmd/ -run "TestFirstRun" -count=0` | No -- Wave 0 |
| UX-02 | Known error pattern produces friendly message | unit | `go test ./cmd/ -run "TestFriendlyError" -count=0` | No -- Wave 0 |
| UX-02 | Unknown error gets generic hint | unit | `go test ./cmd/ -run "TestFriendlyError" -count=0` | No -- Wave 0 |
| UX-02 | Friendly errors only in visual mode | unit | `go test ./cmd/ -run "TestFriendlyError" -count=0` | No -- Wave 0 |
| UX-03 | Progress bar advances through ceremony steps | unit | `go test ./cmd/ -run "TestCeremonyProgress" -count=0` | No -- Wave 0 |
| UX-03 | Non-TTY falls back to plain text | unit | `go test ./cmd/ -run "TestCeremonyProgress" -count=0` | No -- Wave 0 |
| UX-04 | Dashboard includes warnings section | unit | `go test ./cmd/ -run "TestDashboard" -count=0` | No -- Wave 0 |
| UX-04 | Dashboard includes next-step suggestions | unit | `go test ./cmd/ -run "TestDashboard" -count=0` | No -- Wave 0 |
| UX-04 | Warnings not in JSON output | unit | `go test ./cmd/ -run "TestDashboard" -count=0` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/ -run "TestUx|TestFirstRun|TestFriendlyError|TestCeremonyProgress|TestDashboard" -count=0`
- **Per wave merge:** `go test ./... -count=0`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `cmd/ux_firstrun_test.go` -- covers UX-01 (marker file logic, banner content, visual/JSON gating)
- [ ] `cmd/ux_friendly_errors_test.go` -- covers UX-02 (pattern matching, unknown error fallback, visual/JSON gating)
- [ ] `cmd/ux_progress_test.go` -- covers UX-03 (step advancement, non-TTY fallback, timing display)
- [ ] `cmd/status_test.go` or extended `cmd/codex_visuals_test.go` -- covers UX-04 (warnings section, next steps, stale state, midden)
- [ ] Framework install: `go get github.com/schollz/progressbar/v3@v3.19.0` -- before UX-03 implementation

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | -- |
| V3 Session Management | no | -- |
| V4 Access Control | no | -- |
| V5 Input Validation | yes | Error pattern matching must not introduce injection via error messages. Use `strings.Contains()` for pattern matching, not regex with user input. |
| V6 Cryptography | no | -- |

### Known Threat Patterns for Go CLI

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Error message injection | Tampering | Error messages from internal sources only (no user input in pattern matching). Welcome banner is static text. |
| Marker file tampering | Tampering | Marker file is a boolean flag (exists/not-exists). No content to tamper with. If deleted, user sees welcome again (harmless). |
| Progress bar terminal escape injection | Tampering | progressbar/v3 handles sanitization internally. Aether's wrapper does not pass user input to progress bar descriptions. |

## Sources

### Primary (HIGH confidence)
- [VERIFIED: go module registry] -- progressbar/v3 v3.19.0 (2025-12-26), zero transitive deps
- [VERIFIED: go module registry] -- cheggaaa/pb/v3 v3.1.7 (2025-02-28), zero transitive deps
- [VERIFIED: codebase grep] -- cmd/helpers.go:170-184 renderVisualError() interception point
- [VERIFIED: codebase grep] -- cmd/root.go:168-183 PersistentPreRunE hook
- [VERIFIED: codebase grep] -- cmd/codex_visuals.go:377-421 workflowSuggestionsForState()
- [VERIFIED: codebase grep] -- cmd/status.go:56+ renderDashboard() current implementation
- [VERIFIED: codebase grep] -- pkg/colony/pheromones.go:26 ExpiresAt field exists
- [VERIFIED: codebase grep] -- go.mod current dependencies (no progress bar library present)

### Secondary (MEDIUM confidence)
- [VERIFIED: codebase] -- 147 test files in cmd/ with established testing patterns
- [VERIFIED: codebase] -- Existing visual primitives (renderBanner, renderStageMarker, renderNextUp, colorizeCaste) provide consistent building blocks

### Tertiary (LOW confidence)
- None -- all findings verified from codebase or module registry.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- progressbar/v3 verified via go module registry, zero transitive deps confirmed
- Architecture: HIGH -- all integration points identified from codebase (renderVisualError, PersistentPreRunE, renderDashboard)
- Pitfalls: HIGH -- based on known CLI UX anti-patterns, all verified against codebase behavior

**Research date:** 2026-04-29
**Valid until:** 30 days (stable domain -- Go CLI patterns don't change rapidly)
