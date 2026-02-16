# Aether Dev Handoff - Phase 1: Core Utilities

**Date:** 2026-02-16
**Status:** Ready for Implementation
**Phase:** 1 of 9
**Next Action:** Implement `session-verify-fresh` and `session-clear` subcommands

---

## Quick Reference

| Item | Value |
|------|-------|
| **Phase Goal** | Create generic core utilities for session freshness detection |
| **Files to Modify** | `.aether/aether-utils.sh` |
| **Estimated LOC** | ~120 new lines |
| **Full Plan** | `docs/session-freshness-implementation-plan.md` |
| **Parent Context** | `docs/session-freshness-handoff.md` |

---

## What We're Building

Generic versions of the existing `survey-verify-fresh` and `survey-clear` commands that work for all Aether commands (survey, oracle, watch, swarm, init, seal, entomb).

### New Commands

```bash
# Verify session files exist and are fresh
bash .aether/aether-utils.sh session-verify-fresh --command <name> [--force] <session_start_unixtime>

# Clear session files
bash .aether/aether-utils.sh session-clear --command <name> [--dry-run]
```

### Command Mappings

| Command | Directory | Files |
|---------|-----------|-------|
| survey | `.aether/data/survey/` | PROVISIONS.md, TRAILS.md, BLUEPRINT.md, CHAMBERS.md, DISCIPLINES.md, SENTINEL-PROTOCOLS.md, PATHOGENS.md |
| oracle | `.aether/oracle/` | progress.md, research.json |
| watch | `.aether/data/` | watch-status.txt, watch-progress.txt |
| swarm | `.aether/data/swarm/` | findings.json |
| init | `.aether/data/` | COLONY_STATE.json, constraints.json |
| seal | `.aether/data/archive/` | manifest.json |
| entomb | `.aether/chambers/` | manifest.json |

---

## Implementation Steps

### Step 1: Add `session-verify-fresh` Subcommand

**Location:** `.aether/aether-utils.sh` after line 3249 (after `survey-clear`)

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

    # Map command to directory and files
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
        json_err "$E_VALIDATION_FAILED" "Unknown command: $command_name"
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
        missing_docs="$missing_docs $doc"
        continue
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

### Step 2: Add `session-clear` Subcommand

**Location:** `.aether/aether-utils.sh` after `session-verify-fresh`

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
        files=""
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

### Step 3: Add Backward Compatibility Wrappers

**Location:** `.aether/aether-utils.sh` after `session-clear`

```bash
  survey-verify-fresh)
    # Backward compatibility - delegate to session-verify-fresh
    bash "$0" session-verify-fresh --command survey "$@"
    ;;

  survey-clear)
    # Backward compatibility - delegate to session-clear
    bash "$0" session-clear --command survey "$@"
    ;;
```

### Step 4: Update Help Command

Add `session-verify-fresh` and `session-clear` to the help output JSON array (around line 110).

---

## Test Plan

After implementing, run these tests:

```bash
# Test 1: Verify fresh with missing files
bash .aether/aether-utils.sh session-verify-fresh --command survey "" $(date +%s)
# Expected: {"ok":false,"command":"survey","fresh":[],"stale":[],"missing":["PROVISIONS.md",...],"total_lines":0}

# Test 2: Create fresh file and verify
mkdir -p .aether/data/survey
echo "test" > .aether/data/survey/PROVISIONS.md
start_time=$(date +%s)
sleep 1
echo "test2" > .aether/data/survey/TRAILS.md
bash .aether/aether-utils.sh session-verify-fresh --command survey "" $start_time
# Expected: {"ok":false,"fresh":["TRAILS.md"],"stale":["PROVISIONS.md"],...}

# Test 3: Clear with dry-run
bash .aether/aether-utils.sh session-clear --command survey --dry-run
# Expected: {"ok":true,...,"cleared":"PROVISIONS.mdTRAILS.md",...,"dry_run":true}
# Files should still exist

# Test 4: Clear actual
bash .aether/aether-utils.sh session-clear --command survey
# Expected: Files removed

# Test 5: Backward compatibility
bash .aether/aether-utils.sh survey-verify-fresh "" $(date +%s)
# Expected: Same output as session-verify-fresh --command survey

# Test 6: Unknown command
bash .aether/aether-utils.sh session-verify-fresh --command unknown "" 0
# Expected: Error "Unknown command: unknown"
```

---

## Success Criteria

- [ ] `session-verify-fresh` accepts `--command` parameter
- [ ] All 7 commands mapped correctly
- [ ] Cross-platform stat works (macOS `stat -f %m`, Linux `stat -c %Y`)
- [ ] JSON output matches expected format
- [ ] `session-clear` respects `--dry-run` flag
- [ ] Init/seal/entomb have empty file lists (protected operations)
- [ ] Backward compatibility wrappers work
- [ ] All 6 test cases pass

---

## Next Phase

After completing Phase 1:
1. Update this handoff doc with results
2. Create `docs/aether_dev_handoff_phase2.md` for Phase 2 (Refactor Colonize)
3. Proceed to Phase 2

---

## Rollback

If needed, rollback Phase 1:
```bash
git checkout HEAD -- .aether/aether-utils.sh
```

---

**Ready to implement Phase 1.**
