---
phase: 44-suggest-pheromones
plan: 02
type: execute
wave: 1
subsystem: pheromone-system
tags: [pheromones, ui, tick-to-approve, suggestions]
dependency_graph:
  requires: [44-01]
  provides: [44-03]
  affects: [pheromone-system, build-flow]
tech_stack:
  added: []
  patterns: [tick-to-approve, one-at-a-time-ui, json-response]
key_files:
  created: []
  modified:
    - .aether/aether-utils.sh
      - suggest-approve command (257 lines)
      - suggest-quick-dismiss command
decisions:
  - Reused learning-approve-proposals UI pattern for consistency
  - Emoji mapping: FOCUS=🎯, REDIRECT=🚫, FEEDBACK=💬
  - Non-interactive mode auto-skips to prevent blocking CI/CD
  - Empty array handling for bash 3.2 compatibility
metrics:
  duration: "30 minutes"
  completed_date: "2026-02-22"
  tasks_completed: 2
  files_modified: 1
  lines_added: 257
---

# Phase 44 Plan 02: Tick-to-Approve UI for Pheromone Suggestions

## Summary

Implemented the tick-to-approve UI for pheromone suggestions, allowing users to review and approve code-analysis-based recommendations one at a time. The UI follows the established `learning-approve-proposals` pattern for consistency across the colony system.

## What Was Built

### suggest-approve Command

A new command in `aether-utils.sh` that orchestrates the pheromone suggestion approval workflow:

**Features:**
- One-at-a-time display of suggestions with clear formatting
- Four user actions: Approve, Reject, Skip, Dismiss All
- Proper emoji per pheromone type for visual clarity
- Flags for automation: `--yes`, `--dry-run`, `--no-suggest`, `--verbose`
- Non-interactive mode detection (prevents blocking in CI/CD)
- JSON summary with counts and created signal IDs

**User Interface:**
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   S U G G E S T E D   P H E R O M O N E S
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Based on code analysis, the colony suggests these signals:

───────────────────────────────────────────────────
Suggestion 1 of 3
───────────────────────────────────────────────────

🎯 FOCUS (priority: 7/10)

Large file: consider refactoring (450 lines)

Detected in: src/utils/helpers.ts
Reason: File exceeds 300 lines, consider breaking into smaller modules

───────────────────────────────────────────────────
[A]pprove  [R]eject  [S]kip  [D]ismiss All  Your choice:
```

### suggest-quick-dismiss Command

A helper command for bulk dismissal of suggestions:
- Records all current suggestion hashes to prevent re-suggestion
- Useful when user wants to clear suggestions without reviewing individually
- Returns JSON with count of dismissed suggestions

## Commands Added

| Command | Purpose | Flags |
|---------|---------|-------|
| `suggest-approve` | Interactive approval UI | `--yes`, `--dry-run`, `--no-suggest`, `--verbose` |
| `suggest-quick-dismiss` | Bulk dismiss all suggestions | none |

## Verification

All verification tests pass:

```bash
# Test --dry-run (shows non-interactive mode detection)
bash .aether/aether-utils.sh suggest-approve --dry-run

# Test --yes (auto-approve mode)
bash .aether/aether-utils.sh suggest-approve --yes

# Test --no-suggest (skip entirely)
bash .aether/aether-utils.sh suggest-approve --no-suggest

# Test quick dismiss
bash .aether/aether-utils.sh suggest-quick-dismiss
```

## Integration Points

The `suggest-approve` command integrates with:
- `suggest-analyze` (Plan 44-01) — Gets suggestions to display
- `pheromone-write` — Creates approved signals
- `suggest-record` — Records hashes to prevent duplicates

## Deviations from Plan

None — plan executed exactly as written.

## Self-Check: PASSED

- [x] suggest-approve command exists with full tick-to-approve UI
- [x] One-at-a-time display with Approve/Reject/Skip/Dismiss All options
- [x] Proper emoji per pheromone type (🎯 🚫 💬)
- [x] Approved suggestions written as FOCUS signals via pheromone-write
- [x] Rejected/Skipped suggestions handled correctly
- [x] --yes, --dry-run, --no-suggest flags work
- [x] Non-interactive mode detected (no tty = skip)
- [x] JSON summary returned
- [x] suggest-quick-dismiss command exists

## Commits

| Commit | Message |
|--------|---------|
| a695c66 | feat(44-02): add suggest-approve command with tick-to-approve UI |

## Next Steps

Plan 44-03 will integrate the suggestion system into the build flow, automatically running `suggest-approve` during colony initialization or build phases.
