---
schema_version: "1.0"
id: codex-direct-cli-ux-playbook
kind: playbook
category: playbooks
title: Codex Direct CLI UX Playbook
description: "How Codex users interact with Aether via direct CLI, visual output system, and runtime-native UX."
output_types: [platform-review, ux-review, codex-guide]
agent_roles: [builder, watcher, architect, chronicler, queen]
task_types: [codex, cli, visual, ux, platform]
task_keywords: [codex, cli, visual, caste, emoji, stage, ceremony, banner, runtime, ANSI, terminal, identity]
workflow_triggers: [build, continue]
priority: normal
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4400
---

# Codex Direct CLI UX Playbook

This playbook describes how Aether works on Codex CLI, where users interact
directly with the Go runtime without wrapper markdown, using visual output
from the terminal renderer.

## For Beginners

On Claude Code and OpenCode, Aether uses markdown "wrappers" that add framing,
narration, and context around the runtime commands. Codex is different: it
calls the Go binary directly, and all visual output comes from the Go code
itself. There is no markdown layer. This means the Go runtime must handle all
presentation -- banners, progress, identity, and structure -- through terminal
output (ANSI colors, emojis, text formatting).

## Core Principle: Runtime-Native UX

Codex has no wrapper markdown. Every visual element is produced by the Go
runtime in `cmd/codex_visuals.go`. This has important implications:

- UX improvements for Codex are made in Go code, not in markdown files
- The same runtime commands produce output on all platforms, but Codex
  renders them directly while others wrap them
- Codex users see exactly what the runtime produces, nothing more, nothing less

## Caste Identity System

Every worker in Aether has a visual identity composed of three elements:

### 1. Emoji Prefix

Each caste has a single decorative emoji that appears before the worker label.

| Caste | Emoji |
|-------|-------|
| builder | hammer |
| watcher | eyes |
| scout | compass |
| chaos | lightning |
| oracle | crystal ball |
| architect | ruler |
| colonizer | magnifying glass |
| route_setter | map |
| archaeologist | scroll |

The emoji is decorative only. It does not carry semantic meaning; the colored
label does that.

### 2. ANSI-Colored Label

The primary identity is the caste name, rendered in an ANSI color specific to
the caste. Color maps are defined in `casteColorMap` in `cmd/codex_visuals.go`.

| Caste | Color | Purpose |
|-------|-------|---------|
| builder | Yellow | Implementation work |
| watcher | Blue | Verification and quality |
| scout | Green | Research and discovery |
| architect | Magenta | Design and planning |

The `casteLabel()` function renders the label with the caste's ANSI color.
Terminal detection ensures colors are only used when the output supports them.

### 3. Deterministic Name

Each worker gets a deterministic name based on a hash of the caste and task.
The name format follows a "Profession-Number" pattern, for example:
"Mason-67", "Sentinel-23", "Pathfinder-91".

The `casteIdentity()` function computes the name deterministically, which means
the same task always gets the same worker name. This aids in log analysis and
debugging: you can trace "Mason-67" across multiple output lines.

### Full Identity Format

```
[emoji] [Caste] [Name]  Task description
🔨 Builder Mason-67  Implement queen decision logic
```

## Stage Markers

Build and continue output uses stage separators to divide the output into
logical sections. Each separator follows the format:

```
── Stage Name ──
```

### Standard Stages

| Stage | When It Appears |
|-------|----------------|
| Context | Start of build/continue, showing assembled context |
| Tasks | Task breakdown for the current wave |
| Dispatch | Worker spawning and task assignment |
| Verification | Post-build verification results |
| Housekeeping | Cleanup, learning extraction, handoff storage |
| Next Phase | Phase advance summary |
| Colony Complete | Final summary when all phases are done |

Stage markers are rendered by the `stageMarker()` function in
`cmd/codex_visuals.go`. They use a consistent visual style (em-dash borders)
that works in any terminal.

## Color Maps

The visual system uses three lookup maps defined in `cmd/codex_visuals.go`:

- `casteColorMap`: Maps caste names to ANSI color codes
- `casteEmojiMap`: Maps caste names to single emoji characters
- `casteLabelMap`: Maps caste names to human-readable display names

These maps are the single source of truth for caste visual identity. All
rendering functions read from these maps rather than hard-coding values.

## Ceremony Display

Key lifecycle moments produce ceremony output -- visually distinct banners
or markers that highlight important events:

- **Colony init:** Welcome banner with goal display
- **Build start:** Phase and wave information
- **Verification pass/fail:** Clear pass/fail indicators with color
- **Phase advance:** Summary of completed phase
- **Colony seal:** Completion ceremony with maturity milestone

Ceremony output uses wider borders and more prominent formatting to stand
out from routine stage output.

## Banner System

Banners are multi-line terminal art used for major events. They are rendered
by the `renderBanner()` function and adapt to terminal width when possible.

Banner content is generated from templates in the Go code, not from external
files. This keeps the Codex UX self-contained within the binary.

## Differences from Other Platforms

| Aspect | Claude Code / OpenCode | Codex |
|--------|----------------------|-------|
| Command wrappers | Markdown files in `.claude/` or `.opencode/` | None -- direct CLI |
| Visual output | Markdown rendering + runtime | Runtime only |
| Queen narration | Wrapper markdown adds framing | Runtime stage markers |
| Agent definitions | Markdown (`.claude/agents/ant/`) | TOML (`.codex/agents/`) |
| UX changes | Edit markdown wrappers + Go code | Edit Go code only |

## Practical Implications

**For Builders working on Codex UX:**
- All visual changes go in `cmd/codex_visuals.go`
- Do not create markdown wrapper files for Codex
- Test visual output in a real terminal to verify ANSI rendering

**For Watchers verifying Codex output:**
- Check that caste identity renders correctly (emoji + color + name)
- Verify stage markers appear in the correct order
- Ensure ceremony output is visually distinct from routine output

**For Architects designing new Codex features:**
- Design for terminal output first, not markdown
- Consider how information hierarchy works without formatting options like
  headers, bold, or collapsible sections
- Use color and spacing as the primary visual hierarchy tools
