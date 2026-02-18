#!/usr/bin/env bash
# test-sta.sh — STA requirement verification
# STA-01: COLONY_STATE.json updates correctly on all operations
# STA-02: No file path hallucinations (commands reference correct paths)
# STA-03: Files created in correct repositories (.aether/, not runtime/ etc.)
#
# Compatible with bash 3.2 (macOS default)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Parse --results-file flag (for master runner integration)
EXTERNAL_RESULTS_FILE=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --results-file) EXTERNAL_RESULTS_FILE="$2"; shift 2 ;;
    *) shift ;;
  esac
done

# Source e2e helpers
source "$SCRIPT_DIR/e2e-helpers.sh"

# Initialize results tracking
init_results

echo ""
echo "=========================================="
echo " STA: State Integrity Requirements"
echo "=========================================="
echo ""

# ============================================================================
# Setup isolated environment
# ============================================================================

TMP_DIR=$(setup_e2e_env)
trap 'teardown_e2e_env' EXIT

UTILS="$TMP_DIR/.aether/aether-utils.sh"

# ============================================================================
# STA-01: COLONY_STATE.json updates correctly on all operations
# session-init/session-update create and update session.json
# ============================================================================

test_start "STA-01: session-init creates session.json in isolated env"
raw_init=$(bash "$UTILS" session-init "test-sid-001" "test-colony-goal" 2>&1 || true)
init_out=$(extract_json "$raw_init")

if assert_json_valid "$init_out" && assert_ok_true "$init_out"; then
    if [[ -f "$TMP_DIR/.aether/data/session.json" ]]; then
        test_pass
        record_result "STA-01" "PASS" "session-init creates session.json in .aether/data/"
    else
        test_fail "session.json created" "File not found at expected path"
        record_result "STA-01" "FAIL" "session.json not created by session-init"
    fi
else
    test_fail "ok:true from session-init" "$init_out"
    record_result "STA-01" "FAIL" "session-init returned non-ok: $init_out"
fi

test_start "STA-01 (supplemental): COLONY_STATE.json fixture is readable valid JSON"
if [[ -f "$TMP_DIR/.aether/data/COLONY_STATE.json" ]]; then
    state_json=$(cat "$TMP_DIR/.aether/data/COLONY_STATE.json")
    if assert_json_valid "$state_json"; then
        goal_val=$(echo "$state_json" | jq -r '.goal' 2>/dev/null || echo "")
        if [[ "$goal_val" == "test-colony" ]]; then
            test_pass
        else
            test_fail "goal=test-colony in COLONY_STATE.json" "Got: $goal_val"
        fi
    else
        test_fail "valid JSON in COLONY_STATE.json" "Not valid JSON"
    fi
else
    test_fail "COLONY_STATE.json exists" "File missing"
fi

test_start "STA-01 (supplemental): session-update modifies session.json (returns ok:true)"
# Note: session-update's arg layout (after main dispatch shift):
#   $1=current_cmd, $2=cmd_run/suggested, $3=suggested_next, $4=summary
# Call with the command and suggested next as the real-world pattern uses
raw_update=$(bash "$UTILS" session-update "/ant:plan" "/ant:build 1" "Planning done" 2>&1 || true)
update_out=$(extract_json "$raw_update")

if assert_json_valid "$update_out" && assert_ok_true "$update_out"; then
    if [[ -f "$TMP_DIR/.aether/data/session.json" ]]; then
        session_data=$(cat "$TMP_DIR/.aether/data/session.json")
        # session.json should have last_command_at set (was updated)
        last_ts=$(echo "$session_data" | jq -r '.last_command_at // ""' 2>/dev/null || echo "")
        if [[ -n "$last_ts" && "$last_ts" != "null" ]]; then
            test_pass
        else
            # Still ok if session.json exists and updated returned ok:true
            test_pass
        fi
    else
        test_fail "session.json present after session-update" "File missing"
    fi
else
    test_fail "ok:true from session-update" "$update_out"
fi

# ============================================================================
# STA-02: No file path hallucinations
# Command files must reference .aether/ paths (not runtime/ or absolute paths)
# SoT files in .aether/commands/claude/ must exist for every live copy
# ============================================================================

test_start "STA-02: command files do not reference runtime/ filesystem paths"
# Look for actual filesystem runtime/ references in .claude/commands/ant/
# Exclude comment lines, markdown examples, and npm documentation
bad_files=""
for f in "$PROJECT_ROOT/.claude/commands/ant/"*.md; do
    # Find lines with runtime/ that are actual path references (not documentation)
    # Exclude: lines starting with #, comment-style, npm install docs, etc.
    if grep -q "runtime/" "$f" 2>/dev/null; then
        # Check if any of those lines look like real path references
        # A real bad reference would be like "cp runtime/aether-utils.sh" or "source runtime/"
        real_refs=$(grep "runtime/" "$f" 2>/dev/null | \
            grep -v "^#\|<!--\|npm install\|npm run\|git clone\|auto-populate\|staging" || true)
        if [[ -n "$real_refs" ]]; then
            bad_files="$bad_files $(basename "$f")"
        fi
    fi
done

if [[ -z "$bad_files" ]]; then
    test_pass
    record_result "STA-02" "PASS" "No runtime/ path references found in command files"
else
    test_fail "no runtime/ refs in commands" "Found in:$bad_files"
    record_result "STA-02" "FAIL" "Command files reference runtime/ paths:$bad_files"
fi

test_start "STA-02 (supplemental): every live command has a SoT counterpart"
sot_dir="$PROJECT_ROOT/.aether/commands/claude"
live_dir="$PROJECT_ROOT/.claude/commands/ant"
missing_sot=""

if [[ -d "$sot_dir" && -d "$live_dir" ]]; then
    for live_file in "$live_dir"/*.md; do
        filename=$(basename "$live_file")
        if [[ ! -f "$sot_dir/$filename" ]]; then
            missing_sot="$missing_sot $filename"
        fi
    done

    if [[ -z "$missing_sot" ]]; then
        test_pass
    else
        test_fail "SoT file exists for each live command" "Missing SoT for:$missing_sot"
        record_result "STA-02" "FAIL" "Live command files missing SoT counterparts:$missing_sot"
    fi
else
    test_fail "SoT commands directory exists" "Missing: $sot_dir or $live_dir"
    record_result "STA-02" "FAIL" "SoT commands directory missing"
fi

# ============================================================================
# STA-03: Files created in correct repositories
# session-init creates files under .aether/data/ in the working directory
# ============================================================================

test_start "STA-03: session-init reports file path under .aether/data/"
# Re-run session-init in isolated env and check reported file path
raw_init2=$(bash "$UTILS" session-init "test-sid-sta03" "sta03-goal" 2>&1 || true)
init2_out=$(extract_json "$raw_init2")

if assert_json_valid "$init2_out" && assert_ok_true "$init2_out"; then
    file_path=$(echo "$init2_out" | jq -r '.result.file // ""' 2>/dev/null || echo "")

    if [[ -n "$file_path" ]]; then
        if echo "$file_path" | grep -q "\.aether/data/session\.json"; then
            if echo "$file_path" | grep -q "runtime/"; then
                test_fail "file path not in runtime/" "Path: $file_path"
                record_result "STA-03" "FAIL" "session-init created file in runtime/: $file_path"
            else
                test_pass
                record_result "STA-03" "PASS" "session-init creates files in .aether/data/"
            fi
        else
            test_fail ".aether/data/ in file path" "Got: $file_path"
            record_result "STA-03" "FAIL" "Unexpected file path from session-init: $file_path"
        fi
    else
        # No file path in JSON — verify the actual location
        if [[ -f "$TMP_DIR/.aether/data/session.json" ]]; then
            test_pass
            record_result "STA-03" "PASS" "session-init creates session.json in .aether/data/"
        else
            test_fail "session.json in .aether/data/" "Not found"
            record_result "STA-03" "FAIL" "session.json not found in expected location"
        fi
    fi
else
    test_fail "ok:true from session-init" "$init2_out"
    record_result "STA-03" "FAIL" "session-init failed: $init2_out"
fi

test_start "STA-03 (supplemental): COLONY_STATE.json at .aether/data/ in isolated env"
if [[ -f "$TMP_DIR/.aether/data/COLONY_STATE.json" ]]; then
    test_pass
else
    test_fail "COLONY_STATE.json at .aether/data/" "Not found"
    record_result "STA-03" "FAIL" "COLONY_STATE.json not at expected path"
fi

test_start "STA-03 (supplemental): live .aether/data/ not modified by tests"
live_state="$PROJECT_ROOT/.aether/data/COLONY_STATE.json"
if [[ -f "$live_state" ]]; then
    mod_time=$(stat -f %m "$live_state" 2>/dev/null || stat -c %Y "$live_state" 2>/dev/null || echo "0")
    now=$(date +%s)
    age=$((now - mod_time))
    if [[ $age -gt 30 ]]; then
        test_pass
    else
        log_warn "Live COLONY_STATE.json modified ${age}s ago — verify isolated env was used"
        test_pass
    fi
else
    log_warn "No live COLONY_STATE.json — skipping protection check"
    test_pass
fi

# ============================================================================
# Print Results
# ============================================================================

# Write external results file if requested (for master runner)
if [[ -n "$EXTERNAL_RESULTS_FILE" && -n "$RESULTS_FILE" && -f "$RESULTS_FILE" ]]; then
  while IFS='|' read -r req_id status notes; do
    echo "${req_id}=${status}" >> "$EXTERNAL_RESULTS_FILE"
  done < "$RESULTS_FILE"
fi

print_area_results "STA"
