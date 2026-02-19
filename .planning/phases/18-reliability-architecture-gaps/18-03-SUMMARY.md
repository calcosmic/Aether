---
phase: 18-reliability-architecture-gaps
plan: "03"
subsystem: help-discoverability
tags: [arch-08, arch-05, help, queen-commands, documentation, sync-allowlist]
dependency_graph:
  requires: ["18-01"]
  provides: ["queen-commands-discoverable", "queen-commands-reference"]
  affects: [".aether/aether-utils.sh", "bin/sync-to-runtime.sh", "bin/lib/update-transaction.js"]
tech_stack:
  added: []
  patterns: ["help-sections-json", "backward-compat-flat-array"]
key_files:
  created:
    - .aether/docs/queen-commands.md
  modified:
    - .aether/aether-utils.sh
    - bin/sync-to-runtime.sh
    - bin/lib/update-transaction.js
    - tests/bash/test-aether-utils.sh
decisions:
  - "Preserved flat commands array exactly for backward compatibility (callers use jq '.commands[]')"
  - "Used HELP_EOF heredoc delimiter instead of EOF to avoid collision with nested content"
  - "queen-commands.md added adjacent to error-codes.md in both allowlists (same distribution pattern)"
metrics:
  duration: "~6 minutes"
  completed: "2026-02-19"
  tasks_completed: 2
  files_modified: 5
---

# Phase 18 Plan 03: Help Command Sections and Queen Commands Reference Summary

Help command now groups all 88 commands into 9 labeled sections with the "Queen Commands" group making queen-init/queen-read/queen-promote discoverable, plus a contributor reference doc that distributes to all repos.

## What Was Built

Two changes that work together to make queen-* commands discoverable:

1. The `help` command in aether-utils.sh now returns a `sections` key alongside the existing `commands` array. Commands are grouped into 9 sections (Core, Colony State, Queen Commands, Model Routing, Spawn Management, Flag Management, Chamber Management, Swarm Operations, Pheromone System, Utilities). The flat `commands` array is preserved exactly for backward compatibility — any code that currently does `jq '.commands[]'` still works.

2. A new `queen-commands.md` reference document lives in `.aether/docs/` and is added to both sync allowlists so it reaches target repos via `aether update`. The document covers all three queen commands (init, read, promote) with usage examples, argument tables, return formats, and a contributor guide for adding new queen commands.

## Tasks Completed

| Task | Description | Commit | Key Files |
|------|-------------|--------|-----------|
| 1 | Add sections key to help command JSON output | c414809 | .aether/aether-utils.sh |
| 2 | Create queen-commands.md, add to allowlists, add test | ef3f1f6 | .aether/docs/queen-commands.md, bin/sync-to-runtime.sh, bin/lib/update-transaction.js, tests/bash/test-aether-utils.sh |

## Verification Results

All plan success criteria met:

- `bash .aether/aether-utils.sh help | jq '.sections."Queen Commands"[].name'` outputs "queen-init", "queen-read", "queen-promote"
- `bash .aether/aether-utils.sh help | jq '.commands | length'` returns 88 (unchanged — no commands lost)
- `cat .aether/docs/queen-commands.md` — full documentation exists
- Both sync allowlists updated: `grep 'queen-commands.md' bin/sync-to-runtime.sh bin/lib/update-transaction.js`
- 31 bash tests, 0 new failures (2 pre-existing validate-state failures are unrelated)
- npm test: 4 pre-existing failures, 0 new regressions

## Deviations from Plan

None — plan executed exactly as written.

## Self-Check: PASSED

Files created/modified:
- FOUND: .aether/docs/queen-commands.md
- FOUND: .aether/aether-utils.sh (help block replaced)
- FOUND: bin/sync-to-runtime.sh (queen-commands.md entry)
- FOUND: bin/lib/update-transaction.js (queen-commands.md entry)
- FOUND: tests/bash/test-aether-utils.sh (test_help_queen_commands_section)

Commits:
- c414809: feat(18-03): add sections key to help command JSON output
- ef3f1f6: feat(18-03): create queen-commands.md, add to allowlists, add help sections test
