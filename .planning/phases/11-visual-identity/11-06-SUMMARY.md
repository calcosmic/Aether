---
phase: 11-visual-identity
plan: 06
subsystem: visual-language
tags: [banners, next-up, dividers, consistency, gap-closure]
dependency_graph:
  requires: [11-01, 11-02, 11-03, 11-04, 11-05]
  provides: [SC2-complete, SC4-complete, banner-standardization]
  affects: [build.md, organize.md, oracle.md, status.md]
tech_stack:
  added: []
  patterns: [print-next-up-helper, 50-char-banner-standard, unicode-dividers]
key_files:
  modified:
    - .claude/commands/ant/build.md
    - .claude/commands/ant/organize.md
    - .claude/commands/ant/oracle.md
    - .claude/commands/ant/status.md
decisions:
  - Remove hardcoded '🐜 Next Steps:' block entirely from build.md verbose output template — state-based print-next-up helper is strictly superior to conditional hardcoded suggestions
  - Apply print-next-up to both compact and verbose build output — both paths now produce standard Next Up blocks
  - Rename 'Conditional Next Steps:' label in build.md to avoid false positive on 'Next Steps:' grep
  - Replace === and --- dividers in organize.md worker output template — spawned worker reports will now match colony visual language
  - Replace ANSI-wrapped printf banner in organize.md with plain ━━━━ printf — no ANSI colors per project rule
metrics:
  duration_minutes: 4
  tasks_completed: 3
  files_modified: 4
  completed_date: 2026-02-18
---

# Phase 11 Plan 06: Visual Identity Gap Closure Summary

**One-liner:** Closed all Phase 11 verification gaps — build.md now uses print-next-up helper (SC2 complete), and all ━━━━ banners across 4 files standardized to 50 chars (SC4 complete)

## What Was Built

This plan closed the final two gaps from the 11-VERIFICATION.md report:

**Gap 1 — build.md missing standard Next Up block (SC2)**
build.md was the sole holdout from the standard "🐜 Next Up" block format. It used a hardcoded "🐜 Next Steps:" section (lines 1018-1026) embedded inside its verbose output code block template, with conditional logic based on build outcome. This was replaced with a `print-next-up` bash call placed AFTER the output display in BOTH compact and verbose modes. The helper handles state-based routing automatically, adapting to the colony's current state rather than guessing at build outcomes.

**Gap 2 — Banner width inconsistency (SC4)**
Four files had non-standard ━━━━ banner widths:
- `build.md`: compact output had 32-char banner → fixed to 50
- `oracle.md`: status sub-command had two 32-char banners → both fixed to 50
- `status.md`: header had 53-char banner → fixed to 50
- `organize.md`: worker output template used === and --- divider styles (not ━━━━ at all) → replaced with 50-char ━━━━

## Tasks Completed

| Task | Name | Commit | Key Changes |
|------|------|--------|-------------|
| 1 | Replace build.md Next Steps with print-next-up, fix compact banner | eaa7025 | Removed hardcoded '🐜 Next Steps:' block; added 2 print-next-up calls (compact + verbose); fixed 32→50 char banner |
| 2 | Replace organize.md worker output template dividers with ━━━━ | 50239f9 | 5 section dividers updated (=====, -----); ANSI printf banner replaced with plain ━━━━ style |
| 3 | Fix banner widths in oracle.md and status.md to 50-char standard | abe0942 | oracle.md: 2 banners 32→50; status.md: 1 banner 53→50 |

## Verification Results

All 8 verification checks pass:

1. `grep 'Next Steps:' build.md` = 0 — PASS
2. `grep 'Next Up' build.md` >= 1 — PASS (3 occurrences)
3. `grep 'print-next-up' build.md` >= 1 — PASS (2 occurrences)
4. All ━ banners in build.md are 50 chars — PASS (7 banners: lines 179, 181, 405, 407, 946, 972, 974)
5. No ===== in organize.md — PASS (0 occurrences)
6. No ----- in organize.md output format section — PASS
7. All ━ banners in oracle.md are 50 chars — PASS (12 banners, all 50)
8. All ━ banners in status.md are 50 chars — PASS (1 banner at line 181)

## Success Criteria Status

- [x] build.md completion output (both verbose and compact) ends with standard Next Up block via print-next-up helper
- [x] build.md has zero instances of "Next Steps:" heading
- [x] organize.md worker output template uses ━━━━ dividers (no === or multi-dash --- patterns)
- [x] All standalone ━ banners across build.md, oracle.md, and status.md are exactly 50 chars
- [x] 34/34 commands now have standard Next Up blocks (build.md was the last holdout)

## Phase 11 Overall Completion

With this plan, Phase 11 — Visual Identity — is fully complete:
- SC1 (caste emojis): VERIFIED (plan 01)
- SC2 (Next Up blocks on every command): COMPLETE — 34/34 (plans 03-06)
- SC3 (visual progress bars in /ant:status): VERIFIED (plan 02)
- SC4 (consistent banner/divider style): COMPLETE — all ━━━━ at 50 chars (plans 03-06)
- SC5 (single caste emoji source): VERIFIED (plan 01)

## Deviations from Plan

None — plan executed exactly as written.

## Self-Check

### Files Exist

- [x] `.claude/commands/ant/build.md` — exists, modified
- [x] `.claude/commands/ant/organize.md` — exists, modified
- [x] `.claude/commands/ant/oracle.md` — exists, modified
- [x] `.claude/commands/ant/status.md` — exists, modified

### Commits Exist

- [x] eaa7025 — feat(11-06): replace Next Steps with print-next-up in build.md, fix compact banner
- [x] 50239f9 — feat(11-06): replace === and --- dividers with ━━━━ in organize.md output template
- [x] abe0942 — feat(11-06): standardize ━━━━ banner widths to 50 chars in oracle.md and status.md

## Self-Check: PASSED
