# Colonize Command Fix - Implementation Handoff

**Date:** 2026-02-16
**Status:** Complete
**Next Action:** Apply same pattern to `/ant:oracle` command

---

## Summary

Fixed the `/ant:colonize` command to auto-detect and clear stale survey files, ensuring users don't need to use `--force-resurvey` flag.

### Problem
The 11:43 colonize run spawned 4 surveyor agents but they failed silently. Old survey files from 09:16 remained, satisfying the existence check but providing stale data.

### Solution
Added timestamp verification that:
1. Captures `SURVEY_START=$(date +%s)` before spawning agents
2. Checks if existing files have mtime >= SURVEY_START
3. Auto-clears stale files before spawning
4. Only checks for missing files during verification

---

## Files Modified

### 1. `.aether/aether-utils.sh`
Added two new subcommands:

**`survey-verify-fresh`** - Verifies survey documents exist and are fresh
```bash
# Usage: bash .aether/aether-utils.sh survey-verify-fresh [--force] <survey_start_unixtime>
# Returns: JSON with ok, fresh[], stale[], missing[], total_lines
```

**`survey-clear`** - Clears existing survey files
```bash
# Usage: bash .aether/aether-utils.sh survey-clear [--dry-run]
```

Key features:
- Cross-platform timestamp: `stat -f %m` (macOS) / `stat -c %Y` (Linux)
- No jq dependency (pure bash fallback)
- `SURVEY_DIR` env var override for testing

### 2. `.claude/commands/ant/colonize.md`
Updated Steps 3 and 4:

**Step 3 - Auto-detect stale files:**
```bash
SURVEY_START=$(date +%s)
mkdir -p .aether/data/survey

# Auto-detect and clear stale survey files
stale_check=$(bash .aether/aether-utils.sh survey-verify-fresh "" "$SURVEY_START")
has_stale=$(echo "$stale_check" | grep '"stale":\[' | grep -v '"stale":\[\]')

if [[ -n "$has_stale" ]] || [[ "$force_resurvey" == "true" ]]; then
  bash .aether/aether-utils.sh survey-clear
  echo "Cleared existing survey files for fresh survey"
fi
```

**Step 4 - Simplified verification:**
```bash
verify_result=$(bash .aether/aether-utils.sh survey-verify-fresh "" "$SURVEY_START")
missing_docs=$(echo "$verify_result" | grep -o '"missing":\["[^]]*"\]' | ...)
```

---

## Pattern for Applying to Other Commands

The timestamp verification pattern can be applied to any command that:
1. Spawns agents that write files
2. Needs to detect stale/crashed sessions
3. Should auto-clear old data before starting

### Template Implementation

```bash
# 1. Capture start time BEFORE any work
SESSION_START=$(date +%s)

# 2. Check for stale files
stale_check=$(bash .aether/aether-utils.sh <verify-subcommand> "" "$SESSION_START")
has_stale=$(echo "$stale_check" | grep '"stale":\[' | grep -v '"stale":\[\]')

# 3. Auto-clear if stale or force flag
if [[ -n "$has_stale" ]] || [[ "$force_flag" == "true" ]]; then
  bash .aether/aether-utils.sh <clear-subcommand>
  echo "Cleared stale files"
fi

# 4. Spawn agents...

# 5. Verify completion
verify_result=$(bash .aether/aether-utils.sh <verify-subcommand> "" "$SESSION_START")
```

---

## Next Command to Fix: `/ant:oracle`

### Why Oracle?
- Spawns long-running research agent
- Writes to `.aether/oracle/progress.md` and `.aether/oracle/discoveries/`
- Can leave stale progress if session crashes
- Currently has `stop` and `status` subcommands but no freshness detection

### Implementation Steps

1. **Add utility subcommands to `aether-utils.sh`:**
   - `oracle-verify-fresh [--force] <start_time>`
   - `oracle-clear [--dry-run]`

2. **Update `.claude/commands/ant/oracle.md`:**
   - Capture `ORACLE_START=$(date +%s)` at beginning
   - Check for stale oracle session
   - Auto-clear if stale or `--force-research` flag
   - Verify progress.md is fresh after spawning

3. **Add flag parsing:**
   - `--force-research` - explicitly restart oracle
   - Keep existing `--no-visual`

### Oracle-Specific Considerations
- Oracle runs autonomously in background (tmux)
- Creates `.aether/oracle/.stop` signal file
- Progress written incrementally to `progress.md`
- May need different stale threshold (e.g., >1 hour = stale)

---

## Commands That Could Benefit (Priority Order)

| Priority | Command | Issue | Files Affected |
|----------|---------|-------|----------------|
| 1 | `/ant:oracle` | Stale research sessions | `.aether/oracle/*` |
| 2 | `/ant:watch` | Stale watch sessions | `watch-status.txt`, `watch-progress.txt` |
| 3 | `/ant:seal` | Concurrent seal attempts | sealed colony records |
| 4 | `/ant:entomb` | Concurrent entomb attempts | chamber archives |
| 5 | `/ant:init` | Reinitialization without warning | `COLONY_STATE.json` |

---

## Testing the Pattern

To test the colonize fix:
```bash
# Check current survey files are detected as stale
bash .aether/aether-utils.sh survey-verify-fresh "" $(date +%s)
# Expected: {"ok":false,"fresh":[],"stale":[...],"missing":[],"total_lines":...}

# Run colonize (should auto-clear stale files)
/ant:colonize

# Verify files are fresh
bash .aether/aether-utils.sh survey-verify-fresh "" $(date +%s)
# Expected: {"ok":false,"fresh":[],"stale":[...]...} (all stale because no new files)
```

---

## Technical Notes

### Cross-Platform Stat
```bash
# macOS
stat -f %m "$file"  # Returns seconds since epoch

# Linux
stat -c %Y "$file"  # Returns seconds since epoch

# Fallback
file_mtime=$(stat -f %m "$file" 2>/dev/null || stat -c %Y "$file" 2>/dev/null || echo "0")
```

### JSON Without jq
```bash
# Build JSON array in pure bash
items="\"item1\" \"item2\""
json="["
for item in $items; do
  json="$json$item,"
done
json="${json%,}]"  # Remove trailing comma
```

### Environment Variables for Testing
- `SURVEY_DIR` - Override survey directory path
- `ORACLE_DIR` - Override oracle directory path (future)

---

## Backward Compatibility

The colonize fix maintains full backward compatibility:
- No new required flags
- `--force-resurvey` still works as before
- Default behavior improved (auto-clear stale files)
- No breaking changes to existing workflows

---

## Open Questions

1. **Oracle stale threshold:** Should oracle sessions be considered stale after:
   - Fixed time (e.g., 1 hour)?
   - Completion of research cycle?
   - User's explicit stop signal?

2. **Multiple concurrent oracles:** Should we prevent multiple oracle sessions running simultaneously?

3. **Watch command tmux:** Should watch auto-attach to existing session or create new one?

---

## References

- Original issue: Survey files from 09:16 accepted as valid for 11:43 run
- LLM Architect review: `/private/tmp/claude-501/-Users-callumcowie-repos-Aether/tasks/b3be112.output`
- Files changed:
  - `.aether/aether-utils.sh` (+116 lines)
  - `.claude/commands/ant/colonize.md` (updated Steps 3, 4, 6)

---

**Ready to proceed with `/ant:oracle` fix.**
