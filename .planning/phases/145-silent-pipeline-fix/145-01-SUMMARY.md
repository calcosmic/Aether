# Plan 145-01: Remove Error Suppression — Summary

**Status:** Complete
**Commits:** 1
**Files Modified:** 8

## What Was Built

Removed `2>/dev/null || true` error suppression from 33+ data-persistence CLI calls across all playbook instructions and wrapper markdown.

### Files Changed

| File | Lines Changed | Commands Un-suppressed |
|------|--------------|------------------------|
| `build-complete.md` | 1 | `memory-capture` |
| `build-full.md` | 10 | `midden-write`, `memory-capture`, `pheromone-write`, `suggest-approve` |
| `build-verify.md` | 5 | `midden-write`, `memory-capture` |
| `build-wave.md` | 5 | `midden-write`, `memory-capture`, `pheromone-write` |
| `continue-advance.md` | 6 | `memory-capture`, `pheromone-write`, `instinct-create` |
| `continue-finalize.md` | 2 | `backup-prune-global`, `temp-clean` |
| `continue-full.md` | 7 | `pheromone-write`, `memory-capture`, `backup-prune-global`, `temp-clean` |
| `continue-verify.md` | 1 | `survey-verify` |

### Prose Updates

Also updated playbook prose to reflect the honest-error policy:
- Changed "silent and non-blocking" → "non-blocking — errors are visible (honest stderr) but do not halt execution"
- Changed "skip silently" → "skip without error" where appropriate

## Verification

- `grep -rn "2>/dev/null || true" .aether/docs/command-playbooks/ .claude/commands/ant/archaeology.md | grep -E "(memory-capture|midden-write|pheromone-write|instinct-create|backup-prune-global|temp-clean|survey-verify|suggest-approve)"` → **0 matches** ✓
- Read-only fallbacks (`grep`, `curl`, `stat`, `ls`, `git rev-parse`, `wc`, `find`) still have suppression where appropriate ✓

## Acceptance Criteria

- [x] Zero data-persistence commands remain with `2>/dev/null || true` across all 10 target files
- [x] Each changed line previously had `2>/dev/null || true` and now has neither
- [x] No new suppression patterns introduced
- [x] Read-only grep fallbacks in archaeology.md remain unchanged
- [x] Playbook prose accurately describes honest error behavior

## Decisions

- **Scope:** Un-suppressed all `aether *` commands that write colony state, including `instinct-create` (not explicitly in the plan's list but writes to COLONY_STATE.json)
- **Read-only safety:** Kept suppression on `grep`, `curl`, `stat`, `ls`, `git rev-parse`, `wc`, `find`, `rm` — these don't write colony data

## Risks

- Removing `|| true` may cause builds to fail that previously "succeeded" with silent data loss. This is the intended behavior change per D-01.
- Wrappers that expected silent failures may need updates if they don't handle honest stderr correctly.
