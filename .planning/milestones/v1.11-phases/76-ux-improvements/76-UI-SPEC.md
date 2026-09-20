---
phase: 76
slug: ux-improvements
status: approved
reviewed_at: 2026-04-29
shadcn_initialized: false
preset: none
created: 2026-04-29
platform: go-cli
---

# Phase 76 -- UI Design Contract

> Visual and interaction contract for Phase 76 (UX Improvements).
> This phase targets a Go CLI application, not a web frontend.
> All visual specifications use terminal/ANSI conventions.

---

## Design System

| Property | Value |
|----------|-------|
| Tool | none (Go CLI -- no frontend framework) |
| Preset | not applicable |
| Component library | none (go-pretty/v6 for tables, progressbar/v3 for progress bars) |
| Icon library | Unicode emoji (existing caste emoji map in codex_visuals.go) |
| Font | System terminal monospace font (user-controlled) |
| Color system | ANSI 256-color via existing `shouldUseANSIColors()` + caste color map |
| Terminal detection | `shouldRenderVisualOutput()` -- respects AETHER_OUTPUT_MODE, AETHER_FORCE_VISUAL, NO_COLOR, CLICOLOR_FORCE |

### Existing Visual Primitives (Reuse, Don't Replace)

| Primitive | Location | Purpose |
|-----------|----------|---------|
| `renderBanner(emoji, title)` | cmd/codex_visuals.go:259 | Section headers: `━━ emoji T I T L E ━━` (spaced uppercase via `spacedTitle()`) |
| `renderStageMarker(title)` | cmd/codex_visuals.go:274 | Sub-sections: `── Title ──` |
| `renderNextUp(primary, alts...)` | cmd/codex_visuals.go:304 | Next-step guidance blocks with primary + alternatives |
| `visualDivider` | cmd/codex_visuals.go:15 | Full-width separator: `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━` |
| `renderAetherWordmark()` | cmd/codex_visuals.go:263 | ASCII art wordmark (cyan ANSI when colors enabled) |
| `colorizeCaste(caste)` | cmd/codex_visuals.go | ANSI-colored caste labels |
| `shouldRenderVisualOutput(w)` | cmd/codex_visuals.go:181 | TTY detection + output mode gating |
| `isTerminalWriter(w)` | cmd/codex_visuals.go:197 | Raw isatty check |
| `shouldUseANSIColors()` | cmd/codex_visuals.go:2448 | Color support detection (NO_COLOR, CLICOLOR_FORCE, AETHER_FORCE_COLOR) |
| `commandEmoji(cmd)` | cmd/codex_visuals.go:112 | Per-command emoji lookup (50+ commands mapped) |
| `generateProgressBar(cur, total, w)` | cmd/status.go:519 | Static Unicode progress bar: `[████░░░░░░]` |
| `workflowSuggestionsForState(state)` | cmd/codex_visuals.go:377 | Rule-based next-step suggestions (existing, to be extended) |
| `renderNoColonyStatusVisual()` | cmd/status.go:43 | No-colony guidance (existing, unchanged) |

---

## Spacing Scale

Terminal line spacing (no pixel values -- CLI uses line counts and indent levels):

| Token | Value | Usage |
|-------|-------|-------|
| tight | 0 blank lines | Within a section (consecutive items, table rows) |
| default | 1 blank line | Between dashboard sections |
| relaxed | 2 blank lines | Between major visual blocks (banner to content, via `renderNextUp`) |
| indent | 2 spaces | Sub-item indentation (next-step alternatives, error next steps, progress steps) |
| divider | `visualDivider` (40 chars `━`) | Between top-level command output blocks |

Exceptions: none. All terminal output follows this model.

---

## Typography

Terminal typography (font is user-controlled; we control emphasis through structure and ANSI):

| Role | Style | When |
|------|-------|------|
| Body | Plain monospace | Default text, explanations, table content |
| Label | Plain monospace with `:` suffix | Key-value pairs in dashboard (`Goal: ...`, `State: ...`, `Depth: ...`) |
| Heading | `renderBanner(emoji, title)` -- spaced uppercase | Top-level command output sections |
| Sub-heading | `renderStageMarker(title)` | Sub-sections within command output |
| Display | `renderAetherWordmark()` -- ASCII art, cyan ANSI | Install/update commands only -- NOT in welcome banner |
| Emphasis | ANSI bold (`\x1b[1m`) | Warnings (new), error titles, next-step command names |
| Muted | ANSI dim (`\x1b[2m`) | Secondary hints, elapsed time labels, timestamp display |

No custom font sizes. Weight and color are the only emphasis tools. Structural position (banner > stage marker > body > label) provides the primary hierarchy.

---

## Color

ANSI color assignments (respecting `shouldUseANSIColors()` and `NO_COLOR`):

| Role | ANSI Code | Usage |
|------|-----------|-------|
| Dominant (no color) | default terminal fg | Body text, descriptions, explanations, table content |
| Secondary (muted) | `\x1b[2m` (dim) | Timestamps, secondary labels, hints |
| Accent (cyan) | `\x1b[96m` | Aether wordmark only (existing convention in `renderAetherWordmark()`) |
| Error (red) | `\x1b[91m` | Error banners via `renderVisualError()` |
| Warning (yellow) | `\x1b[93m` | Warning section headers in dashboard (NEW), stale state indicators |
| Success (green) | `\x1b[92m` | Next-step command suggestions in dashboard (NEW), completion markers |
| Caste colors | per `casteColorMap` (cmd/codex_visuals.go:56) | Worker identity labels (existing, not changed) |

Accent (cyan) reserved for: Aether wordmark rendering only. Do not use cyan for new UX elements -- use the semantic colors above (yellow for warnings, green for suggestions).

### Color Rules

1. All colored output gated by `shouldUseANSIColors()` -- never emit ANSI codes when colors disabled
2. NO_COLOR environment variable respected (existing behavior in `shouldUseANSIColors()`)
3. JSON output mode never contains ANSI codes (existing behavior in `shouldRenderVisualOutput()`)
4. Progress bar colors handled by progressbar/v3 library (respects NO_COLOR internally)
5. New warning/next-step colors must also gate through `shouldUseANSIColors()`

---

## Copywriting Contract

### Welcome Banner (UX-01) -- Source: D-01, D-02, D-03

| Element | Copy |
|---------|------|
| Banner title | `Welcome to Aether` |
| Banner emoji | `🐜` (via `renderBanner("🐜", "Welcome to Aether")`) |
| Body line 1 | `Aether manages your development colony -- a team of AI workers that plan, build, and verify code together.` |
| Body line 2 | `To get started, set up this repo and create your first colony:` |
| Quick-start 1 | `aether lay-eggs` -- Set up Aether in this repo |
| Quick-start 2 | `aether init "your goal here"` -- Start a colony with a goal |
| Quick-start 3 | `aether status` -- Check on your colony |

Constraints: 5-8 lines total (including banner). No wordmark (per RESEARCH.md recommendation -- wordmark is 6 lines tall, too heavy for first impression). Plain text only in body (no ANSI colors -- keeps it clean on first impression). Not shown in JSON output mode. Marker file: `.aether/data/.welcomed` (zero bytes).

### Friendly Error Messages (UX-02) -- Source: D-04, D-05, D-06

| Error Pattern | Explanation | Next Steps |
|---------------|-------------|------------|
| `no colony initialized` | Aether needs a colony to work with. A colony is a workspace for building toward a specific goal. | `aether init "your goal"` to start a colony. `aether lay-eggs` first if this repo is brand new. |
| `flag --%s is required` | This command needs more information to run. | `aether <command> --help` to see available flags. |
| `file not found` (store/colony files) | Aether could not find a file it needs. This usually means Aether has not been set up in this repo yet. | `aether lay-eggs` to set up Aether. `aether status` to check if a colony exists. |
| `permission denied` | Aether does not have permission to access a file or directory. | Check file permissions. On macOS/Linux: `ls -la <path>` to inspect. |
| Store errors (JSON parse failures) | Aether's data file is corrupted or was modified outside of Aether. | `aether patrol` for diagnostics. If the problem persists, check `.aether/data/COLONY_STATE.json` for syntax errors. |
| Unknown/unexpected errors | (raw error text preserved) | `aether patrol` for diagnostics or `aether status` to check colony health. |

Format: `renderBanner("❌", "Error")` + `visualDivider` + explanation + blank line + `Next steps:` header + 2-space indented bullet list.

### Progress Feedback (UX-03) -- Source: D-07, D-08, D-09

| Element | Copy |
|---------|------|
| Progress bar description | Current ceremony step name (e.g., "Context", "Tasks", "Dispatch", "Verification", "Housekeeping") |
| Step format (TTY) | `Step {n}/{total}: {name} [{progress_bar}] {elapsed}` |
| Step format (non-TTY) | `  Step {n}/{total}: {name} ({elapsed})` |
| Completion line | `Ceremony complete in {total_elapsed}` |

Timing: Show elapsed time prominently. Show ETA as secondary only if progressbar/v3 provides it and step-duration fluctuation is under 50% (per RESEARCH.md recommendation).

### Status Dashboard (UX-04) -- Source: D-10, D-11, D-12, D-13

**Dashboard section order (redesigned):**

| # | Section | Heading | Source | Notes |
|---|---------|---------|--------|-------|
| 1 | Banner + divider | `━━ 📊 C O L O N Y   S T A T U S ━━` | existing | Unchanged |
| 2 | Goal + version + signals | (inline) | existing | Unchanged |
| 3 | **Warnings** | `⚠ Warnings` | NEW | Only rendered if warnings exist; skip if empty |
| 4 | **Next Steps** | via `renderNextUp()` | NEW (extended) | Rule-based from `workflowSuggestionsForState()` |
| 5 | Progress | `Progress` | existing | Unchanged |
| 6 | Metadata | (inline: Focus, Instincts, Flags, Scope, Milestone, Depth, Granularity, Parallel) | existing | Unchanged |
| 7 | Proof | `Proof` | existing | Unchanged |
| 8 | Memory Health | `Memory Health` (go-pretty table) | existing | Unchanged |
| 9 | Review Findings | `Review Findings` (go-pretty table) | existing | Conditional on data |
| 10 | Active Pheromones | `Active Pheromones` (go-pretty table) | existing | Unchanged |
| 11 | Spawn Activity | `Spawn Activity` | existing | Conditional on data |
| 12 | Active Workers | `Active Workers` | existing | Conditional on live spawn |
| 13 | Recent Outcomes | `Recent Outcomes` | existing | Conditional on data |
| 14 | Recovery | `Recovery` | existing | Conditional on guidance |
| 15 | Recent Instincts | `Recent Instincts` | existing | Conditional on data |
| 16 | State line + footer Next Up | (inline) | existing | Kept for backward compatibility |

Next-step rules (extended from existing `workflowSuggestionsForState()` per D-11):

| Colony State | Primary Suggestion |
|--------------|-------------------|
| Sealed (COMPLETED) | `aether entomb` to archive the colony |
| All phases complete | `aether seal` to mark as Crowned Anthill |
| Phase failed | `aether build {N}` to retry the failed phase |
| Post-build (EXECUTING/BUILT) | `aether continue` to verify and advance |
| No colony | `aether init "goal"` or `aether lay-eggs` |
| Paused | `aether resume` to restore the colony |
| No plan | `aether discuss` then `aether plan` |

Warning indicators (from D-12, deterministic from colony state):

| Warning | Threshold | Display Copy |
|---------|-----------|-------------|
| Stale state | Last activity > 7 days | `Stale: colony was last active {N} days ago. Run \`aether status\` after recent work.` |
| Failed phases | Any phase with failed status | `Failed phase {N} ({name}). Run \`aether build {N}\` to retry.` |
| Unacknowledged midden | Any unacknowledged entries in midden | `{N} unacknowledged failure(s). Run \`aether data-clean\` to inspect.` |
| Pheromone expiry | Signal expires within 3 days (25% remaining lifetime) | `Expiring signal: {type} -- {content} (expires {date})` |

Dashboard constraint: All warnings and next steps are visual-mode only (D-13). JSON output continues to return structured colony state data unchanged.

### Destructive Actions

No destructive actions are introduced in this phase. All new UX elements are informational and additive.

---

## Interaction Patterns

### First-Run Detection (UX-01)

```
User runs: aether [any command]
  -> PersistentPreRunE in root.go (after store init succeeds)
    -> Check .aether/data/.welcomed marker file
      -> Missing AND no COLONY_STATE.json?
        -> shouldRenderVisualOutput(stdout)?
          -> Yes: emit welcome banner to stdout, create .welcomed marker (0644), continue
          -> No: skip (JSON mode), continue
      -> Marker exists OR colony exists?
        -> skip, continue to command
```

### Friendly Error Flow (UX-02)

```
Command fails -> outputError(code, message, details)
  -> shouldRenderVisualOutput(stderr)?
    -> Yes: friendlyErrorForPattern(message)
      -> Pattern match found in errorPatternMap?
        -> Yes: render banner + divider + explanation + "Next steps:" + indented suggestions
        -> No: render banner + divider + raw error + blank line + generic hint
    -> No: render JSON envelope {"ok":false,"error":"...","code":N,"details":...}
```

### Ceremony Progress (UX-03)

```
Ceremony starts (build or continue)
  -> newCeremonyProgress(steps, stdout)
    -> isTerminalWriter(stdout)?
      -> Yes: create progressbar/v3 instance (width 40, auto-update)
      -> No: create plain-text wrapper
  -> For each ceremony step:
    -> progress.Advance(stepName)
      -> TTY: update progress bar with elapsed time, conditional ETA
      -> Non-TTY: emit "  Step N/M: name (elapsed)\n"
  -> After final step: emit "Ceremony complete in {total_elapsed}\n"
```

### Status Dashboard (UX-04)

```
User runs: aether status
  -> loadActiveColonyState()
    -> Error / no colony?
      -> Yes: renderNoColonyStatusVisual() (unchanged), return
      -> No: renderDashboard(state, store)
        -> Compute warnings (stale, failed, midden, pheromone expiry)
        -> Compute next steps via extended workflowSuggestionsForState()
        -> Render sections in order (see table above)
        -> Warnings section: only if warnings exist (skip empty)
        -> Next Steps section: always rendered when colony exists
  -> shouldRenderVisualOutput(stdout)?
    -> No: JSON consumers get structured data only (no warnings/next-steps text)
```

---

## Registry Safety

| Registry | Blocks Used | Safety Gate |
|----------|-------------|-------------|
| Go module proxy | progressbar/v3 v3.19.0 | Verified: zero transitive deps via go module registry (2026-04-29) |

No shadcn. No third-party component registries. The only new dependency is a Go module (progressbar/v3) vetted in RESEARCH.md with zero transitive dependencies confirmed via `go list -m -json`.

---

## File Structure (New Files)

```
cmd/
  ux_firstrun.go          # First-run detection + welcome banner rendering
  ux_firstrun_test.go     # Tests for UX-01
  ux_friendly_errors.go   # Error pattern map + friendly rendering
  ux_friendly_errors_test.go  # Tests for UX-02
  ux_progress.go          # Ceremony progress wrapper (progressbar/v3 adapter)
  ux_progress_test.go     # Tests for UX-03
  # status.go             # MODIFIED: redesign renderDashboard() for UX-04
  # helpers.go            # MODIFIED: extend renderVisualError() for UX-02
  # root.go               # MODIFIED: add first-run check to PersistentPreRunE for UX-01
```

---

## Checker Sign-Off

- [ ] Dimension 1 Copywriting: PASS
- [ ] Dimension 2 Visuals: PASS
- [ ] Dimension 3 Color: PASS
- [ ] Dimension 4 Typography: PASS
- [ ] Dimension 5 Spacing: PASS
- [ ] Dimension 6 Registry Safety: PASS

**Approval:** pending
