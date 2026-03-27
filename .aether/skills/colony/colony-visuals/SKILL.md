---
name: colony-visuals
description: Use when producing any user-facing output including banners, progress indicators, section headers, or command results
type: colony
domains: [visuals, ux, formatting]
agent_roles: [builder, watcher, chronicler, scout]
priority: normal
version: "1.0"
---

# Colony Visuals

## Purpose

All colony output must look consistent. Users should never see mismatched banner styles, missing progress bars, or unformatted raw output. This skill standardizes every visual element.

## Banner Format

Use spaced-letter format for all section banners. The standard pattern is:

```
━━ S E C T I O N   T I T L E ━━
```

Rules:
- Every letter is separated by a single space.
- Words are separated by three spaces.
- Bordered by `━━` on both sides with a single space between the border and text.
- Use `print-standard-banner` utility when available. If unavailable, format manually following this pattern exactly.
- Never use alternative banner styles (e.g., `===`, `###`, `***`, `---` banners).

## Progress Bars

Always use the `generate-progress-bar` utility for progress indicators. Never construct progress bars manually with characters like `[=====>    ]`.

Format: `[Phase 3/7] ████████░░░░ 57%`

Show progress bars:
- At the start of each phase build.
- After each worker completes during a build wave.
- In status dashboard output.

## Output Block Structure

Every command output follows this three-part structure:

1. **Header banner** -- Spaced-letter banner identifying the command or section.
2. **Content** -- The actual information, tables, results, or status.
3. **Next Up footer** -- What the user should do next (see colony-lifecycle skill).

Wrap content sections in `━━━` dividers (3+ characters) for visual separation.

## Emoji Usage

- **One emoji per section header** -- Place it before the section title text.
- **Never use emoji in body text** -- Keep body text clean and professional.
- **Consistent emoji mapping** -- Use the same emoji for the same concept across all commands. Examples: build, test, verify, complete, warning, error.

## Tables

When presenting structured data, use aligned markdown tables:

```
| Column 1 | Column 2 | Status |
|----------|----------|--------|
| value    | value    | done   |
```

Ensure columns are padded for alignment. Never use unaligned or ragged tables.

## Color and Emphasis

- **Bold** for labels and headings.
- *Italic* for supplementary notes.
- Use color sparingly and consistently -- green for success, red for errors, yellow for warnings.
