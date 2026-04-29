# Phase 76: UX Improvements - Context

**Gathered:** 2026-04-29
**Status:** Ready for planning

<domain>
## Phase Boundary

Aether provides clear guidance to new users, explains errors in plain language with next steps, shows progress during long ceremony operations, and surfaces actionable information in the status dashboard.

Requirements: UX-01, UX-02, UX-03, UX-04.

**What this phase delivers:**
- First-run welcome banner with quick-start tips, detected via local marker file
- Error pattern map that replaces developer-facing messages with plain-language explanations + suggested next steps for common failure modes
- Progress bar with elapsed/estimated timing for build and continue ceremonies (using a lightweight third-party Go library)
- Redesigned status dashboard with rule-based next-step suggestions and warning/blocker indicators

**What this phase does NOT deliver:**
- Interactive tutorials or guided walkthroughs
- Rewriting every error message in the CLI (only common failure modes)
- Progress bars for non-ceremony commands
- AI-powered or heuristic-based suggestions

</domain>

<decisions>
## Implementation Decisions

### First-Run Experience (UX-01)
- **D-01:** Show a welcome banner with 2-3 quick-start commands when a user runs Aether for the first time. Not a tutorial — just enough context to get moving. Same message on any command that detects first-run state.
- **D-02:** First-run detection via a local marker file (e.g., `.aether/.welcomed`) in the repo's `.aether/data/` directory. Created on first display, checked on every command run. Survives colony init/entomb. Simple, no extra deps, no coupling to hub.
- **D-03:** The welcome banner should appear on any command that detects no colony and no marker file — not just `status`. This way users get guidance regardless of which command they try first.

### Error Messages (UX-02)
- **D-04:** Build an error pattern map: common error patterns (no colony initialized, missing required flags, file not found, store errors, permission errors) map to plain-language explanations + suggested next steps. When an error matches a pattern, render the friendly version instead of the raw error.
- **D-05:** Internal/unexpected errors stay technical but get a generic hint appended: "try /ant-patrol for diagnostics or /ant-status to check colony health." No false suggestions for errors we don't understand.
- **D-06:** The pattern map approach keeps the friendly error logic centralized and cleanly separated from existing error handling. Individual commands don't need to change — the map intercepts at the `outputError`/`renderVisualError` layer.

### Progress Feedback (UX-03)
- **D-07:** Add a progress bar with elapsed time and estimated completion for build and continue ceremonies. Users see where they are in the flow and how long each step takes.
- **D-08:** Use a lightweight third-party Go progress library (e.g., `progressbar`, `uiprogress`, or similar). This is the one exception to the zero-new-deps principle — the UX improvement justifies a focused dependency. Pick the lightest library that gives us progress bar + timing + graceful fallback in non-terminal environments (CI, pipes).
- **D-09:** Progress applies to ceremony steps (build-wave, build-verify, continue-verify, continue-advance, etc.). Non-ceremony commands don't get progress bars — they're fast enough.

### Status Actionability (UX-04)
- **D-10:** Redesign the status dashboard layout to integrate warnings, blockers, and next-step suggestions alongside existing information. Not just appending sections — a cohesive redesign that puts actionable information front and center.
- **D-11:** Rule-based next-step suggestions, deterministic from colony state:
  - Sealed colony → suggest `/ant-entomb`
  - All phases complete → suggest `/ant-seal`
  - Phase failed → suggest `/ant-build N` (rebuild)
  - Post-build (completed but not continued) → suggest `/ant-continue`
  - No colony → suggest `/ant-init "goal"` or `/ant-lay-eggs`
- **D-12:** Warning indicators for: failed phases (with fix suggestion), stale state (last activity > 7 days), unacknowledged midden entries, pheromone signals approaching expiry. All deterministic from colony state — no guessing.
- **D-13:** Warnings and next steps are visual-mode only (they don't appear in JSON output mode). JSON consumers can derive the same information from the existing structured data.

### Claude's Discretion
- Exact welcome banner copy and quick-start command selection
- Which specific errors to include in the pattern map (beyond the obvious common ones)
- Third-party progress library selection (pick the lightest suitable option)
- Dashboard redesign layout details (section ordering, visual formatting)
- Warning threshold values (staleness period, pheromone expiry proximity)
- Whether progress bar shows estimated time remaining or just elapsed time

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements
- `.planning/REQUIREMENTS.md` — UX-01 through UX-04 define the four UX improvement requirements

### Roadmap
- `.planning/ROADMAP.md` — Phase 76 goal, success criteria, dependency on Phase 71

### Error Handling (current implementation)
- `cmd/helpers.go:32-57` — `outputError()` and `outputErrorMessage()` — current error envelope format
- `cmd/helpers.go:170-184` — `renderVisualError()` — current visual error rendering (basic banner + message, no suggestions)
- `cmd/helpers.go:64-77` — `mustGetString()` — example of developer-facing error message ("missing flag --%s")

### Status Dashboard (current implementation)
- `cmd/status.go:18-37` — `statusCmd` entry point, loads colony state and renders dashboard
- `cmd/status.go:43-53` — `renderNoColonyStatusVisual()` — existing no-colony guidance
- `cmd/status.go:56+` — `renderDashboard()` — full dashboard rendering (goal, version, signals, progress, memory health)

### Ceremony Commands (where progress bars go)
- `cmd/codex_build.go` — Build ceremony with parallel worker dispatch and wave management
- `cmd/codex_continue.go` — Continue ceremony with verification and advancement
- `cmd/codex_build_worktree.go` — Worktree-based build dispatch
- `.aether/docs/command-playbooks/build-wave.md` — Build wave playbook (step-by-step)
- `.aether/docs/command-playbooks/continue-advance.md` — Continue advance playbook
- `.aether/docs/command-playbooks/continue-verify.md` — Continue verification playbook

### Visual Rendering (existing patterns)
- `cmd/codex_visuals.go` — Caste identity system, stage markers, banner rendering
- `cmd/codex_visuals.go` — `renderBanner()`, `visualDivider`, `renderNextUp()` patterns

### Colony State (for warnings and suggestions)
- `pkg/colony/colony.go` — `ColonyState` struct, phase statuses, signal data
- `pkg/colony/pheromones.go` — Pheromone signal types and expiry tracking
- `cmd/midden_cmds.go` — Midden failure tracking (unacknowledged entries)

### Architecture
- `CLAUDE.md` — Platform policy, UX architecture, wrapper-runtime contract
- `.planning/codebase/CONVENTIONS.md` — Go code style, output patterns, error handling conventions
- `.planning/codebase/STRUCTURE.md` — Directory structure, command categories

### Prior Phase Context
- `.planning/phases/71-platform-hardening/71-CONTEXT.md` — Phase 71 decisions on cross-platform consistency
- `.planning/phases/75-intelligence-core/75-CONTEXT.md` — Phase 75 decisions on trust scoring and circuit breaker

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `renderBanner()`, `visualDivider`, `renderNextUp()` in `cmd/codex_visuals.go` — existing visual building blocks for consistent rendering
- `renderNoColonyStatusVisual()` in `cmd/status.go` — pattern for showing guidance when colony doesn't exist
- `renderDashboard()` in `cmd/status.go` — existing dashboard to redesign
- `outputError()` / `renderVisualError()` in `cmd/helpers.go` — interception point for friendly error messages
- `loadActiveColonyState()` — existing function that already handles "no colony" case
- `shouldRenderVisualOutput()` — already distinguishes visual vs JSON output mode

### Established Patterns
- `outputOK()` / `outputError()` for all command output (JSON envelope + visual rendering)
- Visual rendering gated by `shouldRenderVisualOutput()` — JSON mode stays clean
- Banner + divider pattern for consistent section headers
- Colony state loaded via `loadActiveColonyState()` with error message extraction
- `.aether/data/` directory for local state files (marker file goes here)

### Integration Points
- `cmd/helpers.go` — `renderVisualError()` is the interception point for friendly errors
- `cmd/status.go` — `renderDashboard()` is the redesign target
- `cmd/codex_build.go` / `cmd/codex_continue.go` — ceremony entry points where progress bars wrap
- `cmd/codex_visuals.go` — visual rendering utilities to extend
- `cmd/root.go` — persistent pre-run hook where first-run detection could be added

</code_context>

<specifics>
## Specific Ideas

- The progress bar should fall back gracefully in CI/piped environments — no ANSI escape codes when stdout isn't a terminal
- Welcome banner should feel warm, not verbose — 5-8 lines max with 3 actionable commands
- Error pattern map should be easy to extend — new patterns added as a map entry, not code changes
- Dashboard redesign should keep the information density users are used to, just reorganized with actionable items more prominent
- Progress library choice matters — pick something with minimal API surface and no transitive dependency explosion

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 76-ux-improvements*
*Context gathered: 2026-04-29*
