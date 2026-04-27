---
phase: 67-seal-hive-brain-wiring
plan: 02
status: complete
requirements:
  - CERE-04
---

# Plan 67-02: Fix Wrapper Parity for Seal Command

## What Was Built

Updated both Claude and OpenCode seal.md wrappers to reflect the new hive promotion confirmation output, replacing the old SUGGESTION relay text. Closed the Phase 62 verification gaps for CERE-02 and wrapper parity.

## Changes Made

### Task 1: Sync seal.md wrappers
- **Claude seal.md**: Replaced SUGGESTION relay bullet with "Hive Brain promotions: {count promoted} instinct(s) promoted to Hive Brain" and failure warning relay
- **OpenCode seal.md**: Full body sync with Claude seal.md — added missing Shelf Candidate Detection section, hive promotion confirmation text, removed stale content
- Parity test no longer reports seal.md drift

### Task 2: Update Phase 62 VERIFICATION.md
- Changed status from `gaps_found` to `gaps_resolved`
- Updated score from 4/5 to 5/5
- CERE-02 gap: status `closed` with evidence of promoteToHive() in seal ceremony
- Parity gap: status `closed` with evidence of OpenCode seal.md sync
- Updated Observable Truths row 2 from FAILED to VERIFIED
- Updated Key Link for OpenCode seal.md from NOT_WIRED to WIRED
- Removed Auto-Promotion anti-pattern row

## Key Files

### key-files.modified
- `.claude/commands/ant/seal.md` — Updated Post-Seal Report with hive promotion confirmation
- `.opencode/commands/ant/seal.md` — Full body sync with Claude seal.md
- `.planning/phases/62-lifecycle-ceremony-seal-and-init/62-VERIFICATION.md` — Both gaps marked closed, score 5/5

## Deviations

None — plan executed as specified.

## Self-Check: PASSED

- Both seal.md files contain "Hive Brain promotions"
- Both seal.md files contain "Shelf Candidate Detection"
- Parity test no longer reports seal.md drift (pre-existing drift in entomb/init/update remains)
- 62-VERIFICATION.md: status gaps_resolved, score 5/5, both gaps closed
