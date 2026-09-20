# Phase 76: UX Improvements - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-29
**Phase:** 76-ux-improvements
**Areas discussed:** First-run experience, Error messages, Progress feedback, Status actionability

---

## First-Run Experience

| Option | Description | Selected |
|--------|-------------|----------|
| Welcome banner + quick tips | Show welcome banner with 2-3 quick-start commands on first run. Same message on any command detecting first-run state. | ✓ |
| Guided walkthrough | Short interactive walkthrough: "Here's what Aether does." More hand-holding. | |
| Status message only | Just improve existing "No colony initialized" message in status. Simplest. | |

**User's choice:** Welcome banner + quick tips

| Option | Description | Selected |
|--------|-------------|----------|
| Marker file | Create `.aether/.welcomed` on first display. Check on every command run. Simple, no extra deps. | ✓ |
| Hub-level flag | Use `~/.aether/` to track welcome seen. Works across repos but couples to hub. | |

**User's choice:** Marker file

---

## Error Messages

| Option | Description | Selected |
|--------|-------------|----------|
| Common errors + next steps | Replace developer-facing errors with plain-language for common failures. Each includes next step. Internal errors get generic hint. | ✓ |
| All errors rewritten | Rewrite every error message in CLI. Comprehensive but large scope. | |
| Generic hints on all errors | Add generic "try these" hint to existing error banner. Minimal but may not match actual error. | |

**User's choice:** Common errors + next steps

| Option | Description | Selected |
|--------|-------------|----------|
| Error pattern map | Map of common error patterns → plain message + suggested fix. Clean separation. | ✓ |
| Inline per-command | Change each command's error handling inline. More precise but spreads logic. | |

**User's choice:** Error pattern map

---

## Progress Feedback

| Option | Description | Selected |
|--------|-------------|----------|
| Progress bar with timing | Progress bar with elapsed/estimated time. More visual but requires time estimation logic and terminal control. | ✓ |
| Step counter in ceremonies | Show "Step 3 of 7: Verifying..." before each step. Simple text, no animation. | |
| More stage markers | Just add more stage markers between steps. Minimal change. | |

**User's choice:** Progress bar with timing

| Option | Description | Selected |
|--------|-------------|----------|
| Third-party progress library | Use lightweight Go library like progressbar or uiprogress. More features but adds dependency. | ✓ |
| ANSI progress bar in Go | Go-only progress bar using ANSI escape codes. No new deps. | |

**User's choice:** Third-party progress library

**Notes:** This is the one exception to the zero-new-deps principle. User accepted the trade-off for better UX.

---

## Status Actionability

| Option | Description | Selected |
|--------|-------------|----------|
| Both suggestions and warnings | Next-step suggestions AND warnings/blockers. Most useful dashboard. | ✓ |
| Next-step suggestions | "What to do next" section based on colony state. | |
| Warnings + blockers | Highlight failed phases, stale state, test failures. More information-dense. | |

**User's choice:** Both suggestions and warnings

| Option | Description | Selected |
|--------|-------------|----------|
| Phase failures, stale state, midden, pheromones | Warnings cover failed phases, stale state (>7 days), unacknowledged midden, expiring pheromones. | ✓ |
| Minimal: failures + staleness only | Smallest useful set. | |

**User's choice:** Full warning set (failures, staleness, midden, pheromones)

| Option | Description | Selected |
|--------|-------------|----------|
| Rule-based next steps | Deterministic rules based on colony state. Clear logic, no AI. | ✓ |
| Rule-based + session heuristics | Add heuristic layer considering session patterns. More context-aware. | |

**User's choice:** Rule-based next steps

| Option | Description | Selected |
|--------|-------------|----------|
| Redesigned layout | Redesign dashboard to integrate warnings and next steps tightly. Cleaner but more work. | ✓ |
| Append to existing dashboard | Keep existing layout, append new sections at bottom. Incremental. | |

**User's choice:** Redesigned layout

---

## Claude's Discretion

- Welcome banner copy and quick-start command selection
- Which specific errors to include in the pattern map
- Third-party progress library selection
- Dashboard redesign layout details
- Warning threshold values
- Whether progress bar shows estimated time remaining or just elapsed time
