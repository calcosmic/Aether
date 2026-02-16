# Session Freshness Detection System - Implementation Handoff

**Date:** 2026-02-16
**Status:** Phases 1-2 Complete, Phases 3-9 Pending
**Next Action:** Implement Phase 3 (Oracle Freshness)

---

## Quick Start for Next Agent

```
Read this file + docs/session-freshness-implementation-plan.md
Then implement Phases 3-9 following the plan
```

---

## Current State

### What's Done (Phases 1-2) ✅

**Core utilities in `.aether/aether-utils.sh`:**

| Lines | Component | Status |
|-------|-----------|--------|
| 3136-3158 | `survey-verify-fresh` backward compat wrapper | ✅ |
| 3160-3178 | `survey-clear` backward compat wrapper | ✅ |
| 3181-3296 | `session-verify-fresh` generic command | ✅ |
| 3298-3381 | `session-clear` generic command | ✅ |

**Colonize command updated:**
- `.claude/commands/ant/colonize.md` lines 85-101 use the new pattern

### Verified Working

```bash
# Test commands that pass:
bash .aether/aether-utils.sh session-verify-fresh --command survey "" $(date +%s)
bash .aether/aether-utils.sh session-clear --command survey --dry-run
bash .aether/aether-utils.sh session-verify-fresh --command oracle "" $(date +%s)
bash .aether/aether-utils.sh survey-verify-fresh "" $(date +%s)  # backward compat
```

---

## Remaining Work (Phases 3-9)

### Phase 3: Oracle Command (~100 lines)

**File:** `.claude/commands/ant/oracle.md`

**What to add:**
1. Parse `--force` / `--force-research` flag in argument parsing section
2. Add `ORACLE_START=$(date +%s)` before research begins
3. Check for stale oracle session files
4. Show options if stale files found (without --force)
5. Auto-clear if --force flag present
6. Verify files are fresh after initialization

**Files affected:**
- `.aether/oracle/progress.md`
- `.aether/oracle/research.json`
- `.aether/oracle/discoveries/`

**Pattern to follow (from colonize.md):**
```bash
ORACLE_START=$(date +%s)
stale_check=$(bash .aether/aether-utils.sh session-verify-fresh --command oracle "" "$ORACLE_START")
has_stale=$(echo "$stale_check" | jq -r '.stale | length')
has_fresh=$(echo "$stale_check" | jq -r '.fresh | length')

if [[ "$has_stale" -gt 0 ]] || [[ "$has_fresh" -gt 0 ]]; then
  if [[ "$force_research" == "true" ]]; then
    bash .aether/aether-utils.sh session-clear --command oracle
    echo "Cleared stale oracle session for fresh research"
  else
    echo "Found existing oracle session. Options:"
    echo "  /ant:oracle status     - View current session"
    echo "  /ant:oracle --force    - Restart with fresh session"
    echo "  /ant:oracle stop       - Stop current session"
    # Don't proceed
  fi
fi
```

---

### Phase 4: Watch Command (~60 lines)

**File:** `.claude/commands/ant/watch.md`

**What to add:**
1. Add `WATCH_START=$(date +%s)` at start
2. Check for stale watch files
3. Add session timestamp to status file header

**Files affected:**
- `.aether/data/watch-status.txt`
- `.aether/data/watch-progress.txt`

**Pattern:**
```bash
WATCH_START=$(date +%s)
stale_check=$(bash .aether/aether-utils.sh session-verify-fresh --command watch "" "$WATCH_START")
# Files will be overwritten by new session - that's expected behavior
```

---

### Phase 5: Swarm Command (~70 lines)

**File:** `.claude/commands/ant/swarm.md`

**What to add:**
1. Add `SWARM_START=$(date +%s)` after reading state
2. Check for stale swarm files
3. Auto-clear stale files (safe - findings are temporary)
4. Verify files fresh after initialization

**Files affected:**
- `.aether/data/swarm/findings.json`
- `.aether/data/swarm/display.json`
- `.aether/data/swarm/timing.json`

**Pattern:**
```bash
SWARM_START=$(date +%s)
stale_check=$(bash .aether/aether-utils.sh session-verify-fresh --command swarm "" "$SWARM_START")
has_stale=$(echo "$stale_check" | jq -r '.stale | length')

if [[ "$has_stale" -gt 0 ]]; then
  bash .aether/aether-utils.sh session-clear --command swarm
  echo "Cleared stale swarm findings for fresh investigation"
fi
```

---

### Phase 6: Init Command (~50 lines) - PROTECTED

**File:** `.claude/commands/ant/init.md`

**What to add:**
1. Add `INIT_START=$(date +%s)` at start
2. Check freshness of existing COLONY_STATE.json
3. Show warning with session age
4. **NEVER auto-clear** - protected operation

**Files affected:**
- `.aether/data/COLONY_STATE.json`
- `.aether/data/constraints.json`

**Pattern:**
```bash
INIT_START=$(date +%s)
fresh_check=$(bash .aether/aether-utils.sh session-verify-fresh --command init "" "$INIT_START")
is_stale=$(echo "$fresh_check" | jq -r '.stale | length')

if [[ -f ".aether/data/COLONY_STATE.json" ]]; then
  echo "Colony already initialized"
  if [[ "$is_stale" -gt 0 ]]; then
    echo "Warning: State file is stale (old session)"
  fi
  echo "To reinitialize, the current state will be reset."
fi
# Note: session-clear for init has empty files list - never auto-clears
```

---

### Phase 7: Seal Command (~40 lines) - PROTECTED

**File:** `.claude/commands/ant/seal.md`

**What to add:**
1. Add `SEAL_START=$(date +%s)` at start
2. Check for incomplete archive directories
3. Show warning about concurrent operations
4. Verify archive integrity after creation
5. **NEVER auto-clear** - protected operation

**Files affected:**
- `.aether/data/archive/session_*_archive/`
- `.aether/data/archive/manifest.json`

**Pattern:**
```bash
SEAL_START=$(date +%s)

# Check for incomplete archives (no manifest.json)
incomplete_archives=$(find .aether/data/archive -type d -name "session_*_archive" 2>/dev/null | while read dir; do
  if [[ ! -f "$dir/manifest.json" ]]; then
    echo "$dir"
  fi
done)

if [[ -n "$incomplete_archives" ]]; then
  echo "Warning: Incomplete archive operations detected:"
  echo "$incomplete_archives"
fi
```

---

### Phase 8: Entomb Command (~40 lines) - PROTECTED

**File:** `.claude/commands/ant/entomb.md`

**What to add:**
1. Add `ENTOMB_START=$(date +%s)` at start
2. Check for incomplete chamber directories
3. Verify chamber integrity before state reset
4. Restore backup on failure
5. **NEVER auto-clear** - protected operation

**Files affected:**
- `.aether/chambers/<chamber-name>/`
- `.aether/chambers/<chamber-name>/colony-state.json`

**Pattern:**
```bash
ENTOMB_START=$(date +%s)

# Check for incomplete chambers
incomplete_chambers=$(find .aether/chambers -type d -mindepth 1 -maxdepth 1 2>/dev/null | while read dir; do
  if [[ ! -f "$dir/colony-state.json" ]]; then
    echo "$dir"
  fi
done)

if [[ -n "$incomplete_chambers" ]]; then
  echo "Warning: Incomplete chamber operations detected"
fi
```

---

### Phase 9: Testing & Documentation (~200 lines)

**Files to create:**

1. **`tests/bash/test-session-freshness.sh`** - Unit tests
2. **`docs/session-freshness-api.md`** - API documentation
3. **Update `CHANGELOG.md`** - Add changelog entry

**Test cases to cover:**
- [ ] Fresh file detection (mtime >= session_start)
- [ ] Stale file detection (mtime < session_start)
- [ ] Missing file detection
- [ ] --force flag bypass
- [ ] All 7 command mappings
- [ ] --dry-run mode
- [ ] Protected commands don't clear
- [ ] JSON output validity
- [ ] Backward compatibility

---

## Key Files Reference

| File | Purpose | Lines |
|------|---------|-------|
| `.aether/aether-utils.sh` | Core utilities | 3136-3381 |
| `.claude/commands/ant/colonize.md` | Reference implementation | 80-110 |
| `.claude/commands/ant/oracle.md` | Phase 3 target | - |
| `.claude/commands/ant/watch.md` | Phase 4 target | - |
| `.claude/commands/ant/swarm.md` | Phase 5 target | - |
| `.claude/commands/ant/init.md` | Phase 6 target | - |
| `.claude/commands/ant/seal.md` | Phase 7 target | - |
| `.claude/commands/ant/entomb.md` | Phase 8 target | - |

---

## Command Mapping Reference

| Command | Directory | Files | Protected? |
|---------|-----------|-------|------------|
| survey | `.aether/data/survey/` | PROVISIONS.md, TRAILS.md, BLUEPRINT.md, CHAMBERS.md, DISCIPLINES.md, SENTINEL-PROTOCOLS.md, PATHOGENS.md | No |
| oracle | `.aether/oracle/` | progress.md, research.json, discoveries/* | No |
| watch | `.aether/data/` | watch-status.txt, watch-progress.txt | No |
| swarm | `.aether/data/swarm/` | findings.json, display.json, timing.json | No |
| init | `.aether/data/` | COLONY_STATE.json, constraints.json | **YES** |
| seal | `.aether/data/archive/` | manifest.json | **YES** |
| entomb | `.aether/chambers/` | manifest.json, colony-state.json | **YES** |

---

## Rollback Instructions

```bash
# Per-phase rollback
git checkout HEAD -- .claude/commands/ant/<command>.md

# Full rollback of all command changes
git checkout HEAD -- .claude/commands/ant/oracle.md
git checkout HEAD -- .claude/commands/ant/watch.md
git checkout HEAD -- .claude/commands/ant/swarm.md
git checkout HEAD -- .claude/commands/ant/init.md
git checkout HEAD -- .claude/commands/ant/seal.md
git checkout HEAD -- .claude/commands/ant/entomb.md
```

---

## Design Decisions (Already Implemented)

1. **Hybrid approach**: Core utilities + command wrappers (not central registry)
2. **Backward compatibility**: `survey-verify-fresh` delegates to `session-verify-fresh --command survey`
3. **Protected operations**: init/seal/entomb have empty `files=""` in session-clear, preventing auto-clear
4. **Cross-platform**: macOS (`stat -f %m`) and Linux (`stat -c %Y`) support
5. **No jq dependency**: JSON built with bash string manipulation

---

## Minor Issues Found (Optional Fixes)

1. **Unused variable** - `subdir_files` in session-clear line 3326 (cosmetic)
2. **Documentation** - Empty string `""` in colonize.md line 88 is unnecessary

---

## References

- **Full Implementation Plan:** `docs/session-freshness-implementation-plan.md`
- **Original Handoff:** `docs/session-freshness-handoff.md`
- **Code Review Results:** Agent a45e91b (approved)

---

**Ready for implementation. Start with Phase 3 (Oracle).**
