# Global Session Freshness Detection System - Implementation Plan

**Date:** 2026-02-16
**Author:** LLM Architect
**Estimated Lines:** ~740 new, ~200 modified
**Prerequisite:** Existing `survey-verify-fresh` and `survey-clear` in aether-utils.sh

---

## Executive Summary

This document provides a step-by-step implementation plan for adding freshness detection to Aether commands that spawn background agents or manage session files. The goal is to prevent stale data from silently breaking workflows.

**Core Pattern (from colonize fix):**
1. Capture `SESSION_START=$(date +%s)` before any work
2. Check for stale files using `session-verify-fresh`
3. Auto-clear if stale or force flag present
4. Verify files are fresh after spawning

---

## Phase 1: Core Utilities - `session-verify-fresh` and `session-clear`

**Prerequisites:** None (foundation phase)
**Estimated LOC:** ~120 new

### Step 1.1: Design Core Utility Functions

Create generalized versions of `survey-verify-fresh` and `survey-clear` that accept a command context.

**File:** `.aether/aether-utils.sh` (append to existing file)

**Key Design Decisions:**
- Generic `session-verify-fresh` command that takes `--command <name>` parameter
- Environment variable override: `{COMMAND}_DIR` for testing (e.g., `ORACLE_DIR`)
- Cross-platform timestamp handling preserved from survey pattern
- JSON output format: `{"ok":bool,"fresh":[],"stale":[],"missing":[],"session_id":"..."}`

### Step 1.2: Implement `session-verify-fresh`

Add to `.aether/aether-utils.sh` after line 3249 (after `survey-clear`):

```bash
  session-verify-fresh)
    # Generic session freshness verification
    # Usage: bash .aether/aether-utils.sh session-verify-fresh --command <name> [--force] <session_start_unixtime>
    # Returns: JSON with pass/fail status and file details

    # Parse arguments
    command_name=""
    force_mode=""
    session_start_time=""

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --command) command_name="$2"; shift 2 ;;
        --force) force_mode="--force"; shift ;;
        *) session_start_time="$1"; shift ;;
      esac
    done

    # Validate command name
    [[ -z "$command_name" ]] && json_err "$E_VALIDATION_FAILED" "Usage: session-verify-fresh --command <name> [--force] <session_start>"

    # Map command to directory and files (using env var override pattern)
    case "$command_name" in
      survey)
        session_dir="${SURVEY_DIR:-.aether/data/survey}"
        required_docs="PROVISIONS.md TRAILS.md BLUEPRINT.md CHAMBERS.md DISCIPLINES.md SENTINEL-PROTOCOLS.md PATHOGENS.md"
        ;;
      oracle)
        session_dir="${ORACLE_DIR:-.aether/oracle}"
        required_docs="progress.md research.json"
        ;;
      watch)
        session_dir="${WATCH_DIR:-.aether/data}"
        required_docs="watch-status.txt watch-progress.txt"
        ;;
      swarm)
        session_dir="${SWARM_DIR:-.aether/data/swarm}"
        required_docs="findings.json"
        ;;
      init)
        session_dir="${INIT_DIR:-.aether/data}"
        required_docs="COLONY_STATE.json constraints.json"
        ;;
      seal|entomb)
        session_dir="${ARCHIVE_DIR:-.aether/data/archive}"
        required_docs="manifest.json"
        ;;
      *)
        json_err "$E_VALIDATION_FAILED" "Unknown command: $command_name" '{"commands":["survey","oracle","watch","swarm","init","seal","entomb"]}'
        ;;
    esac

    # Initialize result arrays
    fresh_docs=""
    stale_docs=""
    missing_docs=""
    total_lines=0

    for doc in $required_docs; do
      doc_path="$session_dir/$doc"

      if [[ ! -f "$doc_path" ]]; then
        # Check if doc exists at root level (for watch files in data/)
        if [[ "$command_name" == "watch" && -f "$session_dir/$doc" ]]; then
          doc_path="$session_dir/$doc"
        elif [[ "$command_name" == "init" && -f "$session_dir/$doc" ]]; then
          doc_path="$session_dir/$doc"
        else
          missing_docs="$missing_docs $doc"
          continue
        fi
      fi

      # Get line count
      lines=$(wc -l < "$doc_path" 2>/dev/null | tr -d ' ' || echo "0")
      total_lines=$((total_lines + lines))

      # In force mode, accept any existing file
      if [[ "$force_mode" == "--force" ]]; then
        fresh_docs="$fresh_docs $doc"
        continue
      fi

      # Check timestamp if session_start_time provided
      if [[ -n "$session_start_time" ]]; then
        # Cross-platform stat: macOS uses -f %m, Linux uses -c %Y
        file_mtime=$(stat -f %m "$doc_path" 2>/dev/null || stat -c %Y "$doc_path" 2>/dev/null || echo "0")

        if [[ "$file_mtime" -ge "$session_start_time" ]]; then
          fresh_docs="$fresh_docs $doc"
        else
          stale_docs="$stale_docs $doc"
        fi
      else
        # No start time provided - accept existing file (backward compatible)
        fresh_docs="$fresh_docs $doc"
      fi
    done

    # Determine pass/fail
    pass=false
    if [[ -z "$missing_docs" ]]; then
      if [[ "$force_mode" == "--force" ]] || [[ -z "$stale_docs" ]]; then
        pass=true
      fi
    fi

    # Build JSON response
    fresh_json=""
    for item in $fresh_docs; do fresh_json="$fresh_json\"$item\","; done
    fresh_json="[${fresh_json%,}]"

    stale_json=""
    for item in $stale_docs; do stale_json="$stale_json\"$item\","; done
    stale_json="[${stale_json%,}]"

    missing_json=""
    for item in $missing_docs; do missing_json="$missing_json\"$item\","; done
    missing_json="[${missing_json%,}]"

    echo "{\"ok\":$pass,\"command\":\"$command_name\",\"fresh\":$fresh_json,\"stale\":$stale_json,\"missing\":$missing_json,\"total_lines\":$total_lines}"
    exit 0
    ;;
```

### Step 1.3: Implement `session-clear`

Add to `.aether/aether-utils.sh` after `session-verify-fresh`:

```bash
  session-clear)
    # Generic session file clearing
    # Usage: bash .aether/aether-utils.sh session-clear --command <name> [--dry-run]

    # Parse arguments
    command_name=""
    dry_run=""

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --command) command_name="$2"; shift 2 ;;
        --dry-run) dry_run="--dry-run"; shift ;;
        *) shift ;;
      esac
    done

    [[ -z "$command_name" ]] && json_err "$E_VALIDATION_FAILED" "Usage: session-clear --command <name> [--dry-run]"

    # Map command to directory and files
    case "$command_name" in
      survey)
        session_dir="${SURVEY_DIR:-.aether/data/survey}"
        files="PROVISIONS.md TRAILS.md BLUEPRINT.md CHAMBERS.md DISCIPLINES.md SENTINEL-PROTOCOLS.md PATHOGENS.md"
        ;;
      oracle)
        session_dir="${ORACLE_DIR:-.aether/oracle}"
        files="progress.md research.json .stop"
        # Also clear discoveries subdirectory
        subdir_files="discoveries/*"
        ;;
      watch)
        session_dir="${WATCH_DIR:-.aether/data}"
        files="watch-status.txt watch-progress.txt"
        ;;
      swarm)
        session_dir="${SWARM_DIR:-.aether/data/swarm}"
        files="findings.json display.json timing.json"
        ;;
      init)
        # Init clear is destructive - only clear with explicit confirmation
        session_dir="${INIT_DIR:-.aether/data}"
        files=""  # Never auto-clear init files
        ;;
      seal|entomb)
        # Archive operations should never be auto-cleared
        session_dir="${ARCHIVE_DIR:-.aether/data/archive}"
        files=""
        ;;
      *)
        json_err "$E_VALIDATION_FAILED" "Unknown command: $command_name"
        ;;
    esac

    cleared=""
    errors=""

    if [[ -d "$session_dir" && -n "$files" ]]; then
      for doc in $files; do
        doc_path="$session_dir/$doc"
        if [[ -f "$doc_path" ]]; then
          if [[ "$dry_run" == "--dry-run" ]]; then
            cleared="$cleared $doc"
          else
            if rm -f "$doc_path" 2>/dev/null; then
              cleared="$cleared $doc"
            else
              errors="$errors $doc"
            fi
          fi
        fi
      done

      # Handle oracle discoveries subdirectory
      if [[ "$command_name" == "oracle" && -d "$session_dir/discoveries" ]]; then
        if [[ "$dry_run" == "--dry-run" ]]; then
          cleared="$cleared discoveries/"
        else
          rm -rf "$session_dir/discoveries" 2>/dev/null && cleared="$cleared discoveries/" || errors="$errors discoveries/"
        fi
      fi
    fi

    json_ok "{\"command\":\"$command_name\",\"cleared\":\"${cleared// /}\",\"errors\":\"${errors// /}\",\"dry_run\":$([[ "$dry_run" == "--dry-run" ]] && echo "true" || echo "false")}"
    ;;
```

### Step 1.4: Update Help Command List

Add the new commands to the help output in `.aether/aether-utils.sh` at line ~110:

```bash
# Find the help command's JSON array and add:
"session-verify-fresh","session-clear"
```

### REVIEW CHECKPOINT (Phase 1)

Before proceeding, verify:
- [ ] `session-verify-fresh` accepts `--command` parameter
- [ ] All 7 commands are mapped correctly
- [ ] Cross-platform stat works on both macOS and Linux
- [ ] JSON output matches expected format
- [ ] `session-clear` respects `--dry-run` flag
- [ ] Init/seal/entomb have protected clear operations

### TEST (Phase 1)

```bash
# Test session-verify-fresh with survey command
bash .aether/aether-utils.sh session-verify-fresh --command survey "" $(date +%s)
# Expected: {"ok":false/false,"command":"survey","fresh":[],"stale":[...],"missing":[...],"total_lines":...}

# Test session-clear dry-run
bash .aether/aether-utils.sh session-clear --command survey --dry-run
# Expected: {"ok":true,"result":{"command":"survey","cleared":"...","errors":"","dry_run":true}}

# Test with environment variable override
SURVEY_DIR=/tmp/test-survey bash .aether/aether-utils.sh session-verify-fresh --command survey "" 0
# Expected: {"ok":false,...,"missing":[...all files...]}
```

### Estimated LOC for Phase 1: ~120 lines

---

## Phase 2: Refactor Colonize to Use Core Utilities

**Prerequisites:** Phase 1 complete
**Estimated LOC:** ~30 modified, ~0 new

### Step 2.1: Update Colonize Command

**File:** `.aether/commands/claude/colonize.md`

Replace the direct calls to `survey-verify-fresh` and `survey-clear` with generic versions.

**Before (current):**
```bash
stale_check=$(bash .aether/aether-utils.sh survey-verify-fresh "" "$SURVEY_START")
bash .aether/aether-utils.sh survey-clear
verify_result=$(bash .aether/aether-utils.sh survey-verify-fresh "" "$SURVEY_START")
```

**After (refactored):**
```bash
stale_check=$(bash .aether/aether-utils.sh session-verify-fresh --command survey "" "$SURVEY_START")
bash .aether/aether-utils.sh session-clear --command survey
verify_result=$(bash .aether/aether-utils.sh session-verify-fresh --command survey "" "$SURVEY_START")
```

### Step 2.2: Create Compatibility Wrappers (Optional)

For backward compatibility, add wrappers in `aether-utils.sh`:

```bash
  survey-verify-fresh)
    # Backward compatibility wrapper - delegates to session-verify-fresh
    bash "$SCRIPT_DIR/aether-utils.sh" session-verify-fresh --command survey "$@"
    ;;

  survey-clear)
    # Backward compatibility wrapper - delegates to session-clear
    bash "$SCRIPT_DIR/aether-utils.sh" session-clear --command survey "$@"
    ;;
```

### REVIEW CHECKPOINT (Phase 2)

- [ ] Colonize command uses `--command survey` parameter
- [ ] Backward compatibility wrappers work
- [ ] Existing tests still pass
- [ ] No change in user-facing behavior

### TEST (Phase 2)

```bash
# Run existing colonize tests (if any)
npm test -- --grep colonize

# Manual test: run colonize and verify files are fresh
/ant:colonize
bash .aether/aether-utils.sh session-verify-fresh --command survey "" $(date +%s)
```

### Estimated LOC for Phase 2: ~30 modified

---

## Phase 3: Oracle Command Freshness Detection

**Prerequisites:** Phase 1 complete
**Estimated LOC:** ~80 new in command, ~20 modified

### Step 3.1: Understand Oracle Session Files

Oracle writes to:
- `.aether/oracle/progress.md` - Research progress log
- `.aether/oracle/research.json` - Configuration
- `.aether/oracle/discoveries/` - Research findings
- `.aether/oracle/.stop` - Stop signal file

### Step 3.2: Update Oracle Command

**File:** `.aether/commands/claude/oracle.md`

Add at Step 1 (after argument parsing):

```markdown
### Step 1.5: Check for Stale Oracle Session

Before starting new research, check for existing oracle session files.

Capture session start time:
```bash
ORACLE_START=$(date +%s)
```

Check for stale files:
```bash
stale_check=$(bash .aether/aether-utils.sh session-verify-fresh --command oracle "" "$ORACLE_START")
has_stale=$(echo "$stale_check" | jq -r '.stale | length')
has_progress=$(echo "$stale_check" | jq -r '.fresh | length')

if [[ "$has_stale" -gt 0 ]] || [[ "$has_progress" -gt 0 ]]; then
  # Found existing oracle session
  if [[ "$force_research" == "true" ]]; then
    bash .aether/aether-utils.sh session-clear --command oracle
    echo "Cleared stale oracle session for fresh research"
  else
    # Existing session found - prompt user
    echo "Found existing oracle session. Options:"
    echo "  /ant:oracle status     - View current session"
    echo "  /ant:oracle --force    - Restart with fresh session"
    echo "  /ant:oracle stop       - Stop current session"
    # Don't proceed - let user decide
    exit 0
  fi
fi
```
```

### Step 3.3: Add `--force` Flag Parsing

In Step 0 (argument parsing), add:

```markdown
Parse `$ARGUMENTS`:
- If contains `--no-visual`: set `visual_mode = false`
- If contains `--force` or `--force-research`: set `force_research = true`
- Remove flags from arguments before routing
```

### Step 3.4: Update Step 2 (Configure Research)

Add verification after writing config files:

```bash
# Verify oracle files are fresh after writing
verify_result=$(bash .aether/aether-utils.sh session-verify-fresh --command oracle "" "$ORACLE_START")
fresh_count=$(echo "$verify_result" | jq -r '.fresh | length')
if [[ "$fresh_count" -lt 2 ]]; then
  echo "Warning: Oracle files not properly initialized"
fi
```

### REVIEW CHECKPOINT (Phase 3)

- [ ] `--force` flag correctly clears stale sessions
- [ ] Existing oracle sessions are detected
- [ ] User is prompted with options instead of auto-clearing
- [ ] Progress.md and research.json are verified fresh

### TEST (Phase 3)

```bash
# Create stale oracle session
mkdir -p .aether/oracle
echo "# Old Progress" > .aether/oracle/progress.md
sleep 1

# Test without force - should show options
/ant:oracle "test research"
# Expected: Shows "Found existing oracle session" message

# Test with force - should clear and proceed
/ant:oracle --force "test research"
# Expected: Clears stale files, proceeds to wizard
```

### Estimated LOC for Phase 3: ~100 lines (80 new, 20 modified)

---

## Phase 4: Watch Command Freshness Detection

**Prerequisites:** Phase 1 complete
**Estimated LOC:** ~60 new in command

### Step 4.1: Understand Watch Session Files

Watch writes to:
- `.aether/data/watch-status.txt` - Current status display
- `.aether/data/watch-progress.txt` - Progress bar display
- tmux session: `aether-colony`

### Step 4.2: Update Watch Command

**File:** `.aether/commands/claude/watch.md`

Add at Step 1 (after prerequisites check):

```markdown
### Step 1.5: Check for Stale Watch Session

Capture session start time:
```bash
WATCH_START=$(date +%s)
```

Check for stale watch files:
```bash
stale_check=$(bash .aether/aether-utils.sh session-verify-fresh --command watch "" "$WATCH_START")
has_stale=$(echo "$stale_check" | jq -r '.stale | length')
```

If stale files exist, they will be overwritten by the new watch session.
The tmux session check in Step 4 handles concurrent sessions.
```

### Step 4.3: Update Step 3 (Create Status File)

Add timestamp to status file:

```bash
# Update status file with session start
cat > .aether/data/watch-status.txt << EOF
       .-.
      (o o)  AETHER COLONY
      | O |  Live Status
       \`-\`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Session Started: $(date -u +"%Y-%m-%dT%H:%M:%SZ")
State: IDLE
Phase: -/-

Active Workers:
  (none)

Last Activity:
  (waiting for colony activity)
EOF
```

### REVIEW CHECKPOINT (Phase 4)

- [ ] Watch files are overwritten on new session
- [ ] Session start time captured
- [ ] tmux session attachment preserved

### TEST (Phase 4)

```bash
# Create stale watch files
touch -t 202501010000 .aether/data/watch-status.txt
sleep 1

# Start watch - should overwrite
/ant:watch
# Then detach: Ctrl+B D

# Verify files are fresh
bash .aether/aether-utils.sh session-verify-fresh --command watch "" $(date +%s)
# Expected: fresh files
```

### Estimated LOC for Phase 4: ~60 lines

---

## Phase 5: Swarm Command Freshness Detection

**Prerequisites:** Phase 1 complete
**Estimated LOC:** ~70 new in command

### Step 5.1: Understand Swarm Session Files

Swarm writes to:
- `.aether/data/swarm/findings.json` - Scout findings
- `.aether/data/swarm/display.json` - Display state
- `.aether/data/swarm/timing.json` - Timing data

### Step 5.2: Update Swarm Command

**File:** `.aether/commands/claude/swarm.md`

Add at Step 2 (after reading state):

```markdown
### Step 2.5: Check for Stale Swarm Session

Capture session start time:
```bash
SWARM_START=$(date +%s)
```

Check for stale swarm files:
```bash
stale_check=$(bash .aether/aether-utils.sh session-verify-fresh --command swarm "" "$SWARM_START")
has_stale=$(echo "$stale_check" | jq -r '.stale | length')

if [[ "$has_stale" -gt 0 ]]; then
  # Auto-clear stale swarm findings
  bash .aether/aether-utils.sh session-clear --command swarm
  echo "Cleared stale swarm findings for fresh investigation"
fi
```

### Step 5.3: Update swarm-findings-init

After initializing findings, verify:

```bash
# Verify swarm files are fresh
verify_result=$(bash .aether/aether-utils.sh session-verify-fresh --command swarm "" "$SWARM_START")
if [[ $(echo "$verify_result" | jq -r '.missing | length') -gt 0 ]]; then
  echo "Warning: Swarm files not properly initialized"
fi
```

### REVIEW CHECKPOINT (Phase 5)

- [ ] Stale swarm findings are auto-cleared
- [ ] New swarm session has fresh files
- [ ] Swarm ID is captured for tracking

### TEST (Phase 5)

```bash
# Create stale swarm files
mkdir -p .aether/data/swarm
echo '{"old": true}' > .aether/data/swarm/findings.json
touch -t 202501010000 .aether/data/swarm/findings.json
sleep 1

# Run swarm
/ant:swarm "test bug"
# Expected: Clears stale files, starts fresh investigation
```

### Estimated LOC for Phase 5: ~70 lines

---

## Phase 6: Init Command Freshness Detection

**Prerequisites:** Phase 1 complete
**Estimated LOC:** ~50 new in command

### Step 6.1: Understand Init Session Files

Init writes to:
- `.aether/data/COLONY_STATE.json` - Colony state
- `.aether/data/constraints.json` - Focus/redirect constraints

### Step 6.2: Update Init Command

**File:** `.aether/commands/claude/init.md`

Update Step 2 (Read Current State):

```markdown
### Step 2: Read Current State with Freshness Check

Capture session start time:
```bash
INIT_START=$(date +%s)
```

Use the Read tool to read `.aether/data/COLONY_STATE.json`.

Check freshness of existing state:
```bash
fresh_check=$(bash .aether/aether-utils.sh session-verify-fresh --command init "" "$INIT_START")
is_stale=$(echo "$fresh_check" | jq -r '.stale | length')
```

If the `goal` field is not null:
- If state is stale (old session): Warn user but proceed
- If state is fresh (active session): Strongly recommend continuation

```
Colony already initialized with goal: "{existing_goal}"

State freshness: {fresh|stale}
Session: {session_id}
Initialized: {initialized_at}

To reinitialize with a new goal, the current state will be reset.
Proceeding with new goal: "{new_goal}"
```
```

### Step 6.3: Protected Clear Operation

Init should NOT auto-clear. The user explicitly chooses to reinitialize.

```markdown
**Note:** Init never auto-clears COLONY_STATE.json. Reinitialization is an explicit user choice.
```

### REVIEW CHECKPOINT (Phase 6)

- [ ] Freshness check shows session age
- [ ] User warned about existing sessions
- [ ] No auto-clear for init (protected operation)
- [ ] Session start time captured

### TEST (Phase 6)

```bash
# Initialize colony
/ant:init "Test goal"
sleep 2

# Try to reinitialize (should warn)
/ant:init "New goal"
# Expected: Shows existing goal with freshness info, proceeds with warning
```

### Estimated LOC for Phase 6: ~50 lines

---

## Phase 7: Seal Command Freshness Detection

**Prerequisites:** Phase 1 complete
**Estimated LOC:** ~40 new in command

### Step 7.1: Understand Seal Session Files

Seal writes to:
- `.aether/data/archive/session_<timestamp>_archive/` - Archive directory
- `.aether/data/archive/manifest.json` - Archive manifest

### Step 7.2: Update Seal Command

**File:** `.aether/commands/claude/seal.md`

Add at Step 1 (after visual mode init):

```markdown
### Step 1.5: Check for Concurrent Seal Operations

Capture session start time:
```bash
SEAL_START=$(date +%s)
```

Check for existing seal operations (in-progress archives):
```bash
# Check for incomplete archive directories (no manifest.json)
incomplete_archives=$(find .aether/data/archive -type d -name "session_*_archive" 2>/dev/null | while read dir; do
  if [[ ! -f "$dir/manifest.json" ]]; then
    echo "$dir"
  fi
done)

if [[ -n "$incomplete_archives" ]]; then
  echo "Warning: Incomplete archive operations detected:"
  echo "$incomplete_archives"
  echo ""
  echo "These may be from interrupted seal operations."
  echo "Proceeding will create a new archive."
fi
```

### Step 7.3: Verify Archive Completion

Add after Step 5 (Archive Colony State):

```markdown
### Step 5.5: Verify Archive Integrity

Verify the archive was created successfully:
```bash
if [[ -f "$archive_dir/manifest.json" ]]; then
  echo "Archive verified: $archive_dir"
else
  echo "Error: Archive creation incomplete"
  # Don't proceed with milestone update
  exit 1
fi
```
```

### REVIEW CHECKPOINT (Phase 7)

- [ ] Incomplete archives are detected
- [ ] Archive integrity verified
- [ ] No auto-clear (archives are precious)
- [ ] Warning shown for concurrent operations

### TEST (Phase 7)

```bash
# Create incomplete archive (simulating interrupted seal)
mkdir -p .aether/data/archive/session_incomplete_archive
# No manifest.json

# Attempt seal
/ant:seal
# Expected: Warning about incomplete archive, proceeds with new archive
```

### Estimated LOC for Phase 7: ~40 lines

---

## Phase 8: Entomb Command Freshness Detection

**Prerequisites:** Phase 1 complete
**Estimated LOC:** ~40 new in command

### Step 8.1: Understand Entomb Session Files

Entomb writes to:
- `.aether/chambers/<chamber-name>/` - Chamber directory
- `.aether/chambers/<chamber-name>/manifest.json` - Chamber manifest

### Step 8.2: Update Entomb Command

**File:** `.aether/commands/claude/entomb.md`

Add at Step 1 (after visual mode init):

```markdown
### Step 1.5: Check for Concurrent Entomb Operations

Capture session start time:
```bash
ENTOMB_START=$(date +%s)
```

Check for incomplete chamber operations:
```bash
# Check for incomplete chambers (no colony-state.json)
incomplete_chambers=$(find .aether/chambers -type d -mindepth 1 -maxdepth 1 2>/dev/null | while read dir; do
  if [[ ! -f "$dir/colony-state.json" ]]; then
    echo "$dir"
  fi
done)

if [[ -n "$incomplete_chambers" ]]; then
  echo "Warning: Incomplete chamber operations detected:"
  echo "$incomplete_chambers"
fi
```

### Step 8.3: Verify Chamber Completion

After Step 7 (Create Chamber Using Utilities), add:

```markdown
### Step 7.5: Verify Chamber Integrity

```bash
if [[ -f ".aether/chambers/{chamber_name}/manifest.json" ]]; then
  echo "Chamber verified: .aether/chambers/{chamber_name}/"
else
  echo "Error: Chamber creation incomplete"
  # Restore state from backup
  mv .aether/data/COLONY_STATE.json.bak .aether/data/COLONY_STATE.json
  exit 1
fi
```
```

### REVIEW CHECKPOINT (Phase 8)

- [ ] Incomplete chambers detected
- [ ] Chamber integrity verified before state reset
- [ ] Backup restored on failure
- [ ] No auto-clear (chambers are precious)

### TEST (Phase 8)

```bash
# Create incomplete chamber
mkdir -p .aether/chambers/incomplete-chamber
# No colony-state.json

# Attempt entomb
/ant:entomb
# Expected: Warning about incomplete chamber
```

### Estimated LOC for Phase 8: ~40 lines

---

## Phase 9: Testing, Documentation, Integration

**Prerequisites:** Phases 1-8 complete
**Estimated LOC:** ~150 new (tests), ~50 new (docs)

### Step 9.1: Create Bash Unit Tests

**File:** `tests/bash/test-session-freshness.sh`

```bash
#!/bin/bash
# Test suite for session freshness detection utilities

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
source "$AETHER_ROOT/tests/bash/test-helpers.sh"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Helper: Run test and track result
run_test() {
  local name="$1"
  local expected="$2"
  local actual="$3"

  TESTS_RUN=$((TESTS_RUN + 1))

  if [[ "$actual" == *"$expected"* ]]; then
    echo "PASS: $name"
    TESTS_PASSED=$((TESTS_PASSED + 1))
  else
    echo "FAIL: $name"
    echo "  Expected: $expected"
    echo "  Actual: $actual"
    TESTS_FAILED=$((TESTS_FAILED + 1))
  fi
}

# Test: session-verify-fresh with missing files
test_verify_fresh_missing() {
  local result
  result=$(SURVEY_DIR=/tmp/nonexistent bash "$AETHER_ROOT/.aether/aether-utils.sh" session-verify-fresh --command survey "" 0)
  run_test "verify_fresh_missing" '"missing":\[' "$result"
}

# Test: session-verify-fresh with stale files
test_verify_fresh_stale() {
  local tmpdir=$(mktemp -d)
  touch -t 202501010000 "$tmpdir/PROVISIONS.md"

  local result
  result=$(SURVEY_DIR="$tmpdir" bash "$AETHER_ROOT/.aether/aether-utils.sh" session-verify-fresh --command survey "" $(date +%s))
  run_test "verify_fresh_stale" '"stale":\["PROVISIONS.md"\]' "$result"

  rm -rf "$tmpdir"
}

# Test: session-verify-fresh with fresh files
test_verify_fresh_fresh() {
  local tmpdir=$(mktemp -d)
  touch "$tmpdir/PROVISIONS.md"
  local start_time=$(date +%s)
  sleep 1
  touch "$tmpdir/PROVISIONS.md"

  local result
  result=$(SURVEY_DIR="$tmpdir" bash "$AETHER_ROOT/.aether/aether-utils.sh" session-verify-fresh --command survey "" $start_time)
  run_test "verify_fresh_fresh" '"fresh":\["PROVISIONS.md"\]' "$result"

  rm -rf "$tmpdir"
}

# Test: session-clear dry-run
test_clear_dry_run() {
  local tmpdir=$(mktemp -d)
  touch "$tmpdir/PROVISIONS.md"

  local result
  result=$(SURVEY_DIR="$tmpdir" bash "$AETHER_ROOT/.aether/aether-utils.sh" session-clear --command survey --dry-run)
  run_test "clear_dry_run" '"dry_run":true' "$result"

  # Verify file still exists
  if [[ -f "$tmpdir/PROVISIONS.md" ]]; then
    run_test "clear_dry_run_preserved" "PROVISIONS.md exists" "PROVISIONS.md exists"
  fi

  rm -rf "$tmpdir"
}

# Test: session-clear actual
test_clear_actual() {
  local tmpdir=$(mktemp -d)
  touch "$tmpdir/PROVISIONS.md"

  local result
  result=$(SURVEY_DIR="$tmpdir" bash "$AETHER_ROOT/.aether/aether-utils.sh" session-clear --command survey)

  # Verify file removed
  if [[ ! -f "$tmpdir/PROVISIONS.md" ]]; then
    run_test "clear_actual_removed" "file removed" "file removed"
  fi

  rm -rf "$tmpdir"
}

# Test: oracle command mapping
test_oracle_mapping() {
  local result
  result=$(ORACLE_DIR=/tmp/test-oracle bash "$AETHER_ROOT/.aether/aether-utils.sh" session-verify-fresh --command oracle "" 0)
  run_test "oracle_mapping" '"command":"oracle"' "$result"
}

# Test: unknown command error
test_unknown_command() {
  local result
  result=$(bash "$AETHER_ROOT/.aether/aether-utils.sh" session-verify-fresh --command unknown "" 0 2>&1 || true)
  run_test "unknown_command" 'Unknown command' "$result"
}

# Run all tests
echo "=== Session Freshness Tests ==="
test_verify_fresh_missing
test_verify_fresh_stale
test_verify_fresh_fresh
test_clear_dry_run
test_clear_actual
test_oracle_mapping
test_unknown_command

# Summary
echo ""
echo "=== Test Summary ==="
echo "Tests run: $TESTS_RUN"
echo "Passed: $TESTS_PASSED"
echo "Failed: $TESTS_FAILED"

exit $TESTS_FAILED
```

### Step 9.2: Create Integration Tests

**File:** `tests/integration/session-freshness.test.js`

```javascript
const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');
const os = require('os');

describe('Session Freshness Integration', () => {
  let tmpDir;

  beforeEach(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'aether-session-'));
  });

  afterEach(() => {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  });

  test('session-verify-fresh detects stale survey files', () => {
    // Create stale survey files
    fs.mkdirSync(path.join(tmpDir, 'survey'));
    fs.writeFileSync(path.join(tmpDir, 'survey', 'PROVISIONS.md'), 'old content');
    fs.writeFileSync(path.join(tmpDir, 'survey', 'TRAILS.md'), 'old content');

    // Make them stale
    const staleTime = Math.floor(Date.now() / 1000) - 3600;
    fs.utimesSync(path.join(tmpDir, 'survey', 'PROVISIONS.md'), staleTime, staleTime);

    const result = execSync(
      `SURVEY_DIR=${tmpDir}/survey bash .aether/aether-utils.sh session-verify-fresh --command survey "" $(date +%s)`,
      { encoding: 'utf-8', cwd: process.cwd() }
    );

    const json = JSON.parse(result);
    expect(json.ok).toBe(false);
    expect(json.stale).toContain('PROVISIONS.md');
  });

  test('session-clear removes files', () => {
    fs.mkdirSync(path.join(tmpDir, 'survey'));
    fs.writeFileSync(path.join(tmpDir, 'survey', 'PROVISIONS.md'), 'content');

    execSync(
      `SURVEY_DIR=${tmpDir}/survey bash .aether/aether-utils.sh session-clear --command survey`,
      { cwd: process.cwd() }
    );

    expect(fs.existsSync(path.join(tmpDir, 'survey', 'PROVISIONS.md'))).toBe(false);
  });

  test('backward compatibility: survey-verify-fresh still works', () => {
    fs.mkdirSync(path.join(tmpDir, 'survey'));

    const result = execSync(
      `SURVEY_DIR=${tmpDir}/survey bash .aether/aether-utils.sh survey-verify-fresh "" $(date +%s)`,
      { encoding: 'utf-8', cwd: process.cwd() }
    );

    const json = JSON.parse(result);
    expect(json).toHaveProperty('fresh');
    expect(json).toHaveProperty('stale');
    expect(json).toHaveProperty('missing');
  });
});
```

### Step 9.3: Update Documentation

**File:** `docs/session-freshness-api.md`

```markdown
# Session Freshness Detection API

## Overview

The session freshness detection system prevents stale session files from silently breaking Aether workflows.

## Commands

### session-verify-fresh

Verify that session files exist and were created after a specified timestamp.

**Usage:**
```bash
bash .aether/aether-utils.sh session-verify-fresh --command <name> [--force] <session_start_unixtime>
```

**Parameters:**
- `--command <name>` - Command context (survey, oracle, watch, swarm, init, seal, entomb)
- `--force` - Accept any existing file regardless of timestamp
- `<session_start_unixtime>` - Unix timestamp to compare against (optional)

**Output:**
```json
{
  "ok": boolean,
  "command": "string",
  "fresh": ["file1.md", ...],
  "stale": ["file2.md", ...],
  "missing": ["file3.md", ...],
  "total_lines": number
}
```

### session-clear

Clear session files for a command.

**Usage:**
```bash
bash .aether/aether-utils.sh session-clear --command <name> [--dry-run]
```

**Parameters:**
- `--command <name>` - Command context
- `--dry-run` - List files that would be cleared without actually deleting

**Output:**
```json
{
  "ok": true,
  "result": {
    "command": "string",
    "cleared": "file1.md file2.md",
    "errors": "",
    "dry_run": boolean
  }
}
```

## Command-Specific Mappings

| Command | Directory | Files |
|---------|-----------|-------|
| survey | `.aether/data/survey/` | PROVISIONS.md, TRAILS.md, BLUEPRINT.md, CHAMBERS.md, DISCIPLINES.md, SENTINEL-PROTOCOLS.md, PATHOGENS.md |
| oracle | `.aether/oracle/` | progress.md, research.json, discoveries/* |
| watch | `.aether/data/` | watch-status.txt, watch-progress.txt |
| swarm | `.aether/data/swarm/` | findings.json, display.json, timing.json |
| init | `.aether/data/` | COLONY_STATE.json, constraints.json |
| seal | `.aether/data/archive/` | manifest.json |
| entomb | `.aether/chambers/` | manifest.json |

## Environment Variables

Override directories for testing:
- `SURVEY_DIR` - Survey directory
- `ORACLE_DIR` - Oracle directory
- `WATCH_DIR` - Watch directory
- `SWARM_DIR` - Swarm directory
- `INIT_DIR` - Init directory
- `ARCHIVE_DIR` - Archive directory

## Protected Operations

Init, seal, and entomb operations are protected:
- `init` - Never auto-clears COLONY_STATE.json
- `seal` - Never auto-clears archives
- `entomb` - Never auto-clears chambers

## Cross-Platform Support

Timestamp detection works on both macOS and Linux:
- macOS: `stat -f %m`
- Linux: `stat -c %Y`
```

### Step 9.4: Update CHANGELOG.md

```markdown
## [Unreleased]

### Added
- Global session freshness detection system
  - `session-verify-fresh` command for generic freshness checking
  - `session-clear` command for clearing session files
  - Support for survey, oracle, watch, swarm, init, seal, and entomb commands
  - Cross-platform timestamp support (macOS/Linux)

### Changed
- `/ant:colonize` now uses generic session utilities
- `/ant:oracle` checks for stale sessions before starting
- `/ant:watch` captures session start time
- `/ant:swarm` auto-clears stale findings
- `/ant:init` shows session freshness information
- `/ant:seal` detects incomplete archives
- `/ant:entomb` detects incomplete chambers
```

### Step 9.5: Update Help Output

Add to `.aether/aether-utils.sh` help command list:
```bash
"session-verify-fresh","session-clear"
```

### REVIEW CHECKPOINT (Phase 9)

- [ ] All unit tests pass
- [ ] Integration tests pass
- [ ] Documentation is complete
- [ ] CHANGELOG updated
- [ ] Help output includes new commands
- [ ] Backward compatibility verified

### TEST (Phase 9)

```bash
# Run all bash tests
bash tests/bash/test-session-freshness.sh

# Run integration tests
npm test -- --grep "Session Freshness"

# Verify help output
bash .aether/aether-utils.sh help | grep session-verify-fresh
```

### Estimated LOC for Phase 9: ~200 lines (150 tests, 50 docs)

---

## Rollback Instructions

### Phase 1 Rollback
```bash
# Remove session-verify-fresh and session-clear from aether-utils.sh
git checkout HEAD -- .aether/aether-utils.sh
```

### Phase 2 Rollback
```bash
# Restore original colonize command
git checkout HEAD -- .aether/commands/claude/colonize.md
```

### Phase 3-8 Rollback
```bash
# Restore individual commands
git checkout HEAD -- .aether/commands/claude/oracle.md
git checkout HEAD -- .aether/commands/claude/watch.md
git checkout HEAD -- .aether/commands/claude/swarm.md
git checkout HEAD -- .aether/commands/claude/init.md
git checkout HEAD -- .aether/commands/claude/seal.md
git checkout HEAD -- .aether/commands/claude/entomb.md
```

### Full Rollback
```bash
# Remove all changes
git checkout HEAD -- .aether/aether-utils.sh
git checkout HEAD -- .aether/commands/claude/
rm -f tests/bash/test-session-freshness.sh
rm -f tests/integration/session-freshness.test.js
rm -f docs/session-freshness-api.md
```

---

## Summary

| Phase | Description | LOC | Status |
|-------|-------------|-----|--------|
| 1 | Core utilities | ~120 | Pending |
| 2 | Refactor colonize | ~30 | Pending |
| 3 | Oracle freshness | ~100 | Pending |
| 4 | Watch freshness | ~60 | Pending |
| 5 | Swarm freshness | ~70 | Pending |
| 6 | Init freshness | ~50 | Pending |
| 7 | Seal freshness | ~40 | Pending |
| 8 | Entomb freshness | ~40 | Pending |
| 9 | Testing & docs | ~200 | Pending |
| **Total** | | **~710** | |

---

## Handoff Document Format

After completing each phase, create a handoff document:

```markdown
# [Phase Name] - Implementation Handoff

**Date:** YYYY-MM-DD
**Status:** Complete/Blocked
**Next Action:** [Next phase or task]

## Summary
[Brief description of what was done]

## Files Modified
- [file path] - [description of changes]

## Testing Performed
- [Test commands run]
- [Results]

## Known Issues
- [Any issues discovered]

## Next Steps
- [What to do next]
```

---

**End of Implementation Plan**
