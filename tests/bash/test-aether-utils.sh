#!/usr/bin/env bash
# Aether Utils Integration Tests
# Tests aether-utils.sh subcommands for valid JSON output and correct behavior

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
AETHER_UTILS_SOURCE="$PROJECT_ROOT/.aether/aether-utils.sh"

# Source test helpers
source "$SCRIPT_DIR/test-helpers.sh"

# Verify jq is available
require_jq

# Verify aether-utils.sh exists
if [[ ! -f "$AETHER_UTILS_SOURCE" ]]; then
    log_error "aether-utils.sh not found at: $AETHER_UTILS_SOURCE"
    exit 1
fi

# ============================================================================
# Helper: Create isolated test environment with aether-utils.sh
# ============================================================================
setup_isolated_env() {
    local tmp_dir
    tmp_dir=$(mktemp -d)
    mkdir -p "$tmp_dir/.aether/data"

    # Copy aether-utils.sh to temp location so it uses temp data dir
    cp "$AETHER_UTILS_SOURCE" "$tmp_dir/.aether/aether-utils.sh"
    chmod +x "$tmp_dir/.aether/aether-utils.sh"

    # Copy utils directory if it exists (needed for acquire_lock, etc.)
    local utils_source="$(dirname "$AETHER_UTILS_SOURCE")/utils"
    if [[ -d "$utils_source" ]]; then
        cp -r "$utils_source" "$tmp_dir/.aether/"
    fi

    echo "$tmp_dir"
}

# ============================================================================
# Test: help subcommand
# ============================================================================
test_help() {
    local output
    output=$(bash "$AETHER_UTILS_SOURCE" help 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON"
        return 1
    fi

    if ! assert_json_has_field "$output" "commands"; then
        test_fail "has 'commands' field" "field missing"
        return 1
    fi

    if ! assert_json_has_field "$output" "description"; then
        test_fail "has 'description' field" "field missing"
        return 1
    fi

    # Verify commands array is not empty
    local cmd_count
    cmd_count=$(echo "$output" | jq '.commands | length')
    if [[ "$cmd_count" -eq 0 ]]; then
        test_fail "non-empty commands array" "empty array"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: version subcommand
# ============================================================================
test_version() {
    local output
    output=$(bash "$AETHER_UTILS_SOURCE" version 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        return 1
    fi

    if ! assert_json_field_equals "$output" ".result" "1.0.0"; then
        test_fail '"1.0.0"' "$(echo "$output" | jq -r '.result')"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: validate-state colony
# ============================================================================
test_validate_state_colony() {
    local output
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Create valid COLONY_STATE.json (v3.0 format to avoid migration during test)
    cat > "$tmp_dir/.aether/data/COLONY_STATE.json" << 'EOF'
{
  "version": "3.0",
  "goal": "test",
  "state": "active",
  "current_phase": 1,
  "plan": {"id": "test"},
  "memory": {},
  "errors": {"records": []},
  "events": [],
  "signals": [],
  "graveyards": [],
  "session_id": "test",
  "initialized_at": "2026-02-13T16:00:00Z"
}
EOF

    output=$(bash "$tmp_dir/.aether/aether-utils.sh" validate-state colony 2>&1) || true
    rm -rf "$tmp_dir"

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: validate-state constraints
# ============================================================================
test_validate_state_constraints() {
    local output
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Create valid constraints.json
    cat > "$tmp_dir/.aether/data/constraints.json" << 'EOF'
{
  "focus": ["testing"],
  "constraints": ["test"]
}
EOF

    output=$(bash "$tmp_dir/.aether/aether-utils.sh" validate-state constraints 2>&1) || true
    rm -rf "$tmp_dir"

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: validate-state missing file
# ============================================================================
test_validate_state_missing() {
    local output
    local exit_code
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Don't create any data files - test missing file handling
    set +e
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" validate-state colony 2>&1)
    exit_code=$?
    set -e
    rm -rf "$tmp_dir"

    # Should return non-zero exit code for error
    if [[ "$exit_code" -eq 0 ]]; then
        test_fail "non-zero exit code" "exit code 0"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON error" "invalid JSON: $output"
        return 1
    fi

    if ! assert_ok_false "$output"; then
        test_fail '{"ok":false}' "$output"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: activity-log-init
# ============================================================================
test_activity_log_init() {
    local output
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    output=$(bash "$tmp_dir/.aether/aether-utils.sh" activity-log-init 1 "Test Phase" 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify activity.log was created
    if [[ ! -f "$tmp_dir/.aether/data/activity.log" ]]; then
        test_fail "activity.log created" "file not found"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: activity-log-read
# ============================================================================
test_activity_log_read() {
    local output
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Create an activity log
    echo "[12:00:00] Test entry" > "$tmp_dir/.aether/data/activity.log"

    output=$(bash "$tmp_dir/.aether/aether-utils.sh" activity-log-read 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: flag-list (empty)
# ============================================================================
test_flag_list_empty() {
    local output
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    output=$(bash "$tmp_dir/.aether/aether-utils.sh" flag-list 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Should return empty flags array
    local count
    count=$(echo "$output" | jq '.result.flags | length')
    if [[ "$count" -ne 0 ]]; then
        test_fail "0 flags" "$count flags"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: flag-add and flag-list
# ============================================================================
test_flag_add_and_list() {
    local output
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Add a flag
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" flag-add issue "Test Issue" "Test description" manual 1 2>&1)

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON from flag-add" "invalid JSON: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # List flags
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" flag-list 2>&1)

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON from flag-list" "invalid JSON: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    local count
    count=$(echo "$output" | jq '.result.flags | length')
    if [[ "$count" -ne 1 ]]; then
        test_fail "1 flag" "$count flags"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: generate-ant-name
# ============================================================================
test_generate_ant_name() {
    local output
    output=$(bash "$AETHER_UTILS_SOURCE" generate-ant-name builder 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        return 1
    fi

    # Verify result is a non-empty string with expected format (Prefix-Number)
    local name
    name=$(echo "$output" | jq -r '.result')
    if [[ ! "$name" =~ ^[A-Za-z]+-[0-9]+$ ]]; then
        test_fail "name matching Pattern-Number format" "$name"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: error-summary (empty state)
# ============================================================================
test_error_summary_empty() {
    local output
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Create COLONY_STATE.json with empty errors
    cat > "$tmp_dir/.aether/data/COLONY_STATE.json" << 'EOF'
{
  "goal": "test",
  "state": "active",
  "current_phase": 1,
  "plan": {},
  "memory": {},
  "errors": {"records": []},
  "events": []
}
EOF

    output=$(bash "$tmp_dir/.aether/aether-utils.sh" error-summary 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify total is 0
    local total
    total=$(echo "$output" | jq '.result.total')
    if [[ "$total" -ne 0 ]]; then
        test_fail "total: 0" "total: $total"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: invalid subcommand
# ============================================================================
test_invalid_subcommand() {
    local output
    local exit_code

    set +e
    output=$(bash "$AETHER_UTILS_SOURCE" invalid-command 2>&1)
    exit_code=$?
    set -e

    # Should return non-zero exit code
    if [[ "$exit_code" -eq 0 ]]; then
        test_fail "non-zero exit code" "exit code 0"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON error" "invalid JSON: $output"
        return 1
    fi

    if ! assert_ok_false "$output"; then
        test_fail '{"ok":false}' "$output"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: check-antipattern
# ============================================================================
test_check_antipattern() {
    local output
    local tmp_dir
    tmp_dir=$(mktemp -d)

    # Create a test file with a TODO
    echo "// TODO: fix this" > "$tmp_dir/test.js"

    output=$(bash "$AETHER_UTILS_SOURCE" check-antipattern "$tmp_dir/test.js" 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: bootstrap-system (requires hub)
# ============================================================================
test_bootstrap_system() {
    local output
    local tmp_dir
    tmp_dir=$(mktemp -d)
    mkdir -p "$tmp_dir/.aether"

    # Create mock hub system directory
    mkdir -p "$tmp_dir/.aether-hub/system"
    echo "# test" > "$tmp_dir/.aether-hub/system/aether-utils.sh"

    # Copy script to temp location
    cp "$AETHER_UTILS_SOURCE" "$tmp_dir/.aether/aether-utils.sh"

    # Override HOME to point to mock hub
    export HOME="$tmp_dir"

    output=$(bash "$tmp_dir/.aether/aether-utils.sh" bootstrap-system 2>&1) || true

    unset HOME

    # Filter out fallback json_err diagnostic warning (stderr line from ERR-01 fix)
    local json_output
    json_output=$(echo "$output" | grep -v '^\[aether\] Warning:')

    # This may fail if hub doesn't exist, that's OK - just verify JSON output
    if [[ -n "$json_output" ]]; then
        if ! assert_json_valid "$json_output"; then
            test_fail "valid JSON" "invalid JSON: $json_output"
            rm -rf "$tmp_dir"
            return 1
        fi
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Helper: Create isolated env WITHOUT utils/ directory (forces fallback json_err)
# ============================================================================
setup_isolated_env_no_utils() {
    local tmp_dir
    tmp_dir=$(mktemp -d)
    mkdir -p "$tmp_dir/.aether/data"

    # Copy aether-utils.sh only — deliberately omit utils/ so error-handler.sh won't load
    cp "$AETHER_UTILS_SOURCE" "$tmp_dir/.aether/aether-utils.sh"
    chmod +x "$tmp_dir/.aether/aether-utils.sh"

    echo "$tmp_dir"
}

# ============================================================================
# Test: fallback json_err emits both code and message fields (ERR-01)
# ============================================================================
test_fallback_json_err() {
    local stderr_output
    local exit_code
    local tmp_dir
    tmp_dir=$(setup_isolated_env_no_utils)

    # Run queen-init without any template — will trigger json_err "$E_FILE_NOT_FOUND" "Template not found..."
    # Override HOME to a temp dir with no hub templates so no template is found
    local tmp_home
    tmp_home=$(mktemp -d)

    set +e
    stderr_output=$(HOME="$tmp_home" bash "$tmp_dir/.aether/aether-utils.sh" queen-init 2>&1 >/dev/null)
    exit_code=$?
    set -e

    rm -rf "$tmp_dir" "$tmp_home"

    # Should exit non-zero
    if [[ "$exit_code" -eq 0 ]]; then
        test_fail "non-zero exit code" "exit code 0"
        return 1
    fi

    # stderr should contain the diagnostic warning
    if ! assert_contains "$stderr_output" "error-handler.sh not loaded"; then
        test_fail "stderr contains 'error-handler.sh not loaded'" "$stderr_output"
        return 1
    fi

    # Extract the JSON line from stderr (skip the warning line)
    local json_line
    json_line=$(echo "$stderr_output" | grep -v '^\[aether\]' | tail -1)

    # JSON must be valid
    if ! assert_json_valid "$json_line"; then
        test_fail "valid JSON on stderr" "invalid JSON: $json_line"
        return 1
    fi

    # Must have ok:false
    if ! assert_ok_false "$json_line"; then
        test_fail '{"ok":false}' "$json_line"
        return 1
    fi

    # .error.code must be a non-empty string
    local code
    code=$(echo "$json_line" | jq -r '.error.code' 2>/dev/null || echo "")
    if [[ -z "$code" ]] || [[ "$code" == "null" ]]; then
        test_fail "non-empty .error.code" "$code"
        return 1
    fi

    # .error.message must be a non-empty string
    local message
    message=$(echo "$json_line" | jq -r '.error.message' 2>/dev/null || echo "")
    if [[ -z "$message" ]] || [[ "$message" == "null" ]]; then
        test_fail "non-empty .error.message" "$message"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: fallback json_err with single argument defaults correctly (ERR-01)
# ============================================================================
test_fallback_json_err_single_arg() {
    local stderr_output
    local tmp_dir
    tmp_dir=$(setup_isolated_env_no_utils)

    # Create a tiny caller script in the isolated env that invokes the fallback
    # directly by loading only the fallback definition block from aether-utils.sh.
    # We use a subshell script that does NOT source the full aether-utils.sh
    # (to avoid set -euo pipefail complications) but replicates the fallback block.
    local caller_script
    caller_script="$tmp_dir/invoke_fallback.sh"
    cat > "$caller_script" << 'CALLER'
#!/bin/bash
# This script replicates the fallback json_err block in isolation and calls it
# with a single argument to test default handling.
if ! type json_err &>/dev/null; then
  json_err() {
    local code="${1:-E_UNKNOWN}"
    local message="${2:-An unknown error occurred}"
    printf '[aether] Warning: error-handler.sh not loaded — using minimal fallback\n' >&2
    printf '{"ok":false,"error":{"code":"%s","message":"%s"}}\n' "$code" "$message" >&2
    exit 1
  }
fi
json_err "MY_ERROR_CODE"
CALLER
    chmod +x "$caller_script"

    set +e
    stderr_output=$(bash "$caller_script" 2>&1 >/dev/null)
    set -e

    rm -rf "$tmp_dir"

    # The warning must appear
    if ! assert_contains "$stderr_output" "error-handler.sh not loaded"; then
        test_fail "stderr contains 'error-handler.sh not loaded'" "$stderr_output"
        return 1
    fi

    # Extract JSON line
    local json_line
    json_line=$(echo "$stderr_output" | grep -v '^\[aether\]' | tail -1)

    if ! assert_json_valid "$json_line"; then
        test_fail "valid JSON" "invalid JSON: $json_line"
        return 1
    fi

    # .error.code should be the single arg passed
    local code
    code=$(echo "$json_line" | jq -r '.error.code' 2>/dev/null || echo "")
    if [[ "$code" != "MY_ERROR_CODE" ]]; then
        test_fail ".error.code = MY_ERROR_CODE" "$code"
        return 1
    fi

    # .error.message should be the default
    local message
    message=$(echo "$json_line" | jq -r '.error.message' 2>/dev/null || echo "")
    if [[ -z "$message" ]] || [[ "$message" == "null" ]]; then
        test_fail "non-empty default .error.message" "$message"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: queen-init finds template via hub path first (ARCH-01)
# ============================================================================
test_queen_init_template_hub_path() {
    local output
    local exit_code
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Simulate npm-installed user: remove runtime/templates/ if it exists
    rm -rf "$tmp_dir/runtime"

    # Create a fake hub at a temp HOME
    local tmp_home
    tmp_home=$(mktemp -d)
    mkdir -p "$tmp_home/.aether/system/templates"

    # Copy the real QUEEN.md.template to the fake hub
    local real_template="$PROJECT_ROOT/runtime/templates/QUEEN.md.template"
    if [[ -f "$real_template" ]]; then
        cp "$real_template" "$tmp_home/.aether/system/templates/QUEEN.md.template"
    else
        # Create a minimal template if real one not available
        cat > "$tmp_home/.aether/system/templates/QUEEN.md.template" << 'TMPL'
# QUEEN.md — Colony Context
Generated: {TIMESTAMP}
TMPL
    fi

    set +e
    output=$(HOME="$tmp_home" bash "$tmp_dir/.aether/aether-utils.sh" queen-init 2>&1)
    exit_code=$?
    set -e

    rm -rf "$tmp_dir" "$tmp_home"

    # Should succeed
    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code: $output"
        return 1
    fi

    # Output should be valid JSON
    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        return 1
    fi

    # Should have ok:true
    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        return 1
    fi

    # Should report created:true (first time, no existing QUEEN.md)
    local created
    created=$(echo "$output" | jq -r '.result.created' 2>/dev/null || echo "false")
    if [[ "$created" != "true" ]]; then
        test_fail '"created":true' "created: $created"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: queen-init error message is actionable when no template found (ARCH-01)
# ============================================================================
test_queen_init_template_not_found_message() {
    local stderr_output
    local exit_code
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Remove runtime/ from the isolated env
    rm -rf "$tmp_dir/runtime"

    # Override HOME to a temp dir with no hub templates
    local tmp_home
    tmp_home=$(mktemp -d)

    set +e
    stderr_output=$(HOME="$tmp_home" bash "$tmp_dir/.aether/aether-utils.sh" queen-init 2>&1 >/dev/null)
    exit_code=$?
    set -e

    rm -rf "$tmp_dir" "$tmp_home"

    # Should exit non-zero
    if [[ "$exit_code" -eq 0 ]]; then
        test_fail "non-zero exit code" "exit code 0"
        return 1
    fi

    # Extract JSON line (skip any warning lines)
    local json_line
    json_line=$(echo "$stderr_output" | grep -v '^\[aether\]' | tail -1)

    if ! assert_json_valid "$json_line"; then
        test_fail "valid JSON error" "invalid JSON: $stderr_output"
        return 1
    fi

    if ! assert_ok_false "$json_line"; then
        test_fail '{"ok":false}' "$json_line"
        return 1
    fi

    # Error message must contain actionable instructions
    # Note: .error may be a string (simple fallback) or object (full handler)
    local err_message
    err_message=$(echo "$json_line" | jq -r 'if (.error | type) == "object" then .error.message else .error end // ""' 2>/dev/null || echo "")
    if ! assert_contains "$err_message" "aether install" && ! assert_contains "$err_message" "restore"; then
        test_fail "error message contains 'aether install' or 'restore'" "$err_message"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: ERR-03 regression — no bare-string json_err calls in aether-utils.sh
# ============================================================================
test_no_bare_string_json_err_calls() {
    local count
    # grep -c returns exit code 1 when count is 0 (no matches), but still prints "0" to stdout.
    # Use 'set +e' to avoid the script aborting on exit code 1, capture the count directly.
    set +e
    count=$(grep -c 'json_err "[^\$]' "$AETHER_UTILS_SOURCE" 2>/dev/null)
    set -e
    count="${count:-0}"
    if [[ "$count" -ne 0 ]]; then
        log_error "Found $count bare-string json_err call(s) in aether-utils.sh"
        log_error "All json_err calls must use \$E_* constants as first argument"
        grep -n 'json_err "[^\$]' "$AETHER_UTILS_SOURCE" >&2 || true
        return 1
    fi
    return 0
}

# ============================================================================
# Test: ERR-03 regression — no bare-string json_err calls in chamber scripts
# ============================================================================
test_no_bare_string_json_err_in_chamber_scripts() {
    local count=0
    local chamber_utils="$PROJECT_ROOT/.aether/utils/chamber-utils.sh"
    local chamber_compare="$PROJECT_ROOT/.aether/utils/chamber-compare.sh"

    # grep -c returns exit code 1 when count is 0, but still prints "0" to stdout.
    # Capture output directly with set +e to avoid false errors.
    local part
    set +e
    if [[ -f "$chamber_utils" ]]; then
        part=$(grep -c 'json_err "[^\$]' "$chamber_utils" 2>/dev/null)
        count=$((count + ${part:-0}))
    fi
    if [[ -f "$chamber_compare" ]]; then
        part=$(grep -c 'json_err "[^\$]' "$chamber_compare" 2>/dev/null)
        count=$((count + ${part:-0}))
    fi
    set -e

    # Phase 17-02 fixed the chamber script json_err override bug.
    # Baseline is now 0 — any bare-string calls are regressions.
    local known_baseline=0
    if [[ "$count" -gt "$known_baseline" ]]; then
        log_error "Chamber script bare-string json_err count ($count) exceeds baseline ($known_baseline)"
        log_error "New bare-string calls have been introduced — fix them before merging"
        return 1
    fi
    return 0
}

# ============================================================================
# Test: ERR-04 runtime — flag-resolve missing flags file returns E_FILE_NOT_FOUND
# ============================================================================
test_flag_resolve_missing_flags_file_error_code() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)
    # Deliberately do NOT create flags.json — should trigger E_FILE_NOT_FOUND

    set +e
    local stderr_output
    stderr_output=$(bash "$tmp_dir/.aether/aether-utils.sh" flag-resolve some_flag_id 2>&1)
    set -e

    rm -rf "$tmp_dir"

    # Extract .error.code from the last JSON line on stderr
    local json_line
    json_line=$(echo "$stderr_output" | grep -v '^\[aether\]' | grep '"ok":false' | tail -1)

    local code
    code=$(echo "$json_line" | jq -r '.error.code' 2>/dev/null || echo "")

    if [[ "$code" != "E_FILE_NOT_FOUND" ]]; then
        test_fail ".error.code = E_FILE_NOT_FOUND" ".error.code = ${code:-<empty>}"
        return 1
    fi
    return 0
}

# ============================================================================
# Test: ERR-04 runtime — flag-add missing arguments returns E_VALIDATION_FAILED
# ============================================================================
test_flag_add_missing_args_error_code() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)
    # Invoke flag-add with no title argument — should trigger E_VALIDATION_FAILED

    set +e
    local stderr_output
    stderr_output=$(bash "$tmp_dir/.aether/aether-utils.sh" flag-add 2>&1)
    set -e

    rm -rf "$tmp_dir"

    # Extract .error.code from the last JSON line on stderr
    local json_line
    json_line=$(echo "$stderr_output" | grep -v '^\[aether\]' | grep '"ok":false' | tail -1)

    local code
    code=$(echo "$json_line" | jq -r '.error.code' 2>/dev/null || echo "")

    if [[ "$code" != "E_VALIDATION_FAILED" ]]; then
        test_fail ".error.code = E_VALIDATION_FAILED" ".error.code = ${code:-<empty>}"
        return 1
    fi
    return 0
}

# ============================================================================
# Test: ERR-04 runtime — flag-add with held lock returns E_LOCK_FAILED
# ============================================================================
test_flag_add_lock_failure_error_code() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Create flags.json so the file-not-found check passes
    echo '{"version":1,"flags":[]}' > "$tmp_dir/.aether/data/flags.json"

    # Determine the lock directory that file-lock.sh will use.
    # file-lock.sh uses git rev-parse --show-toplevel, which returns the Aether
    # repo root when tests run from the repo. Fall back to cwd if git fails.
    local repo_root
    repo_root=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
    local lock_dir="$repo_root/.aether/locks"
    local lock_file="$lock_dir/flags.json.lock"
    local lock_pid_file="${lock_file}.pid"

    # Pre-create a lock with a nonexistent PID to simulate a held lock.
    # In non-interactive mode, file-lock.sh treats this as a stale lock and
    # returns 1. flag-add then emits json_err "$E_LOCK_FAILED".
    mkdir -p "$lock_dir"
    echo "99999" > "$lock_file"
    echo "99999" > "$lock_pid_file"

    set +e
    local stderr_output
    stderr_output=$(bash "$tmp_dir/.aether/aether-utils.sh" flag-add issue "test-lock-flag" "testing lock failure" 2>&1)
    set -e

    # Always clean up lock files — must happen even if test fails
    rm -f "$lock_file" "$lock_pid_file"
    rm -rf "$tmp_dir"

    # stderr contains both E_LOCK_STALE (from file-lock.sh) and E_LOCK_FAILED (from flag-add).
    # Parse the last {"ok":false,...} line to verify flag-add emitted E_LOCK_FAILED.
    local json_line
    json_line=$(echo "$stderr_output" | grep '"ok":false' | tail -1)

    local code
    code=$(echo "$json_line" | jq -r '.error.code' 2>/dev/null || echo "")

    if [[ "$code" != "E_LOCK_FAILED" ]]; then
        test_fail ".error.code = E_LOCK_FAILED" ".error.code = ${code:-<empty>}"
        return 1
    fi
    return 0
}

# ============================================================================
# Test: ARCH-09 regression — feature detection block appears after fallback json_err
# ============================================================================
test_feature_detection_after_fallbacks() {
    local fallback_line feature_line
    fallback_line=$(grep -n 'json_err()' "$AETHER_UTILS_SOURCE" | grep 'Fallback\|fallback\|json_err()' | head -1 | cut -d: -f1)
    # json_err() definition line (inside the fallback block)
    fallback_line=$(grep -n 'json_err()' "$AETHER_UTILS_SOURCE" | head -1 | cut -d: -f1)
    feature_line=$(grep -n 'feature_disable "activity_log"' "$AETHER_UTILS_SOURCE" | head -1 | cut -d: -f1)
    if [[ -z "$fallback_line" ]] || [[ -z "$feature_line" ]]; then
        test_fail "both fallback json_err and feature detection lines found" "fallback=$fallback_line feature=$feature_line"
        return 1
    fi
    if [[ "$feature_line" -le "$fallback_line" ]]; then
        test_fail "feature detection (line $feature_line) after fallback json_err (line $fallback_line)" "feature before fallback"
        return 1
    fi
    return 0
}

# ============================================================================
# Test: ARCH-10 regression — _aether_exit_cleanup calls both cleanup functions
# ============================================================================
test_composed_exit_trap_exists() {
    if ! grep -q '_aether_exit_cleanup' "$AETHER_UTILS_SOURCE"; then
        test_fail "_aether_exit_cleanup function exists" "not found"
        return 1
    fi
    if ! grep -A5 '_aether_exit_cleanup()' "$AETHER_UTILS_SOURCE" | grep -q 'cleanup_locks'; then
        test_fail "_aether_exit_cleanup calls cleanup_locks" "not found"
        return 1
    fi
    if ! grep -A5 '_aether_exit_cleanup()' "$AETHER_UTILS_SOURCE" | grep -q 'cleanup_temp_files'; then
        test_fail "_aether_exit_cleanup calls cleanup_temp_files" "not found"
        return 1
    fi
    return 0
}

# ============================================================================
# Test: ARCH-03 regression — _rotate_spawn_tree function exists in session-init
# ============================================================================
test_spawn_tree_rotation_exists() {
    if ! grep -q '_rotate_spawn_tree' "$AETHER_UTILS_SOURCE"; then
        test_fail "_rotate_spawn_tree function exists" "not found"
        return 1
    fi
    if ! grep -q 'spawn-tree-archive' "$AETHER_UTILS_SOURCE"; then
        test_fail "spawn-tree-archive directory reference" "not found"
        return 1
    fi
    return 0
}

# ============================================================================
# Test: queen-read has JSON validation gates (ARCH-06)
# ============================================================================
test_queen_read_validates_metadata() {
    # Verify Gate 1: metadata validation before --argjson
    if ! grep -q 'malformed METADATA' "$AETHER_UTILS_SOURCE"; then
        test_fail "queen-read has metadata validation gate (Gate 1)" "not found"
        return 1
    fi
    # Verify Gate 2: result validation before json_ok
    if ! grep -q 'assemble queen-read' "$AETHER_UTILS_SOURCE"; then
        test_fail "queen-read has result validation gate (Gate 2)" "not found"
        return 1
    fi
    return 0
}

# ============================================================================
# Test: validate-state has schema migration logic (ARCH-02)
# ============================================================================
test_validate_state_has_schema_migration() {
    # Verify migration function exists
    if ! grep -q '_migrate_colony_state' "$AETHER_UTILS_SOURCE"; then
        test_fail "validate-state has _migrate_colony_state function" "not found"
        return 1
    fi
    # Verify migration emits W_MIGRATED warning on version change
    if ! grep -q 'W_MIGRATED' "$AETHER_UTILS_SOURCE"; then
        test_fail "migration emits W_MIGRATED warning" "not found"
        return 1
    fi
    return 0
}

# ============================================================================
# Test: ARCH-07 — model-get/model-list do not use exec bash model-profile
# ============================================================================
test_model_get_no_exec_pattern() {
    set +e
    local count
    count=$(grep -c 'exec bash.*model-profile' "$AETHER_UTILS_SOURCE" 2>/dev/null)
    set -e
    count="${count:-0}"
    if [[ "$count" -gt 0 ]]; then
        test_fail "zero exec bash model-profile calls (ARCH-07)" "$count found"
        return 1
    fi
    return 0
}

# ============================================================================
# Test: ARCH-07 — model-get error message includes Try: suggestion
# ============================================================================
test_model_get_error_has_try_suggestion() {
    # model-get with empty caste should emit friendly error with Try: suggestion
    set +e
    local output
    output=$(bash "$AETHER_UTILS_SOURCE" model-get "" 2>&1)
    set -e
    if ! echo "$output" | grep -q 'Try:'; then
        test_fail "model-get error includes 'Try:' suggestion (ARCH-07)" "not found in output: $output"
        return 1
    fi
    return 0
}

# ============================================================================
# Test: help has Queen Commands section with backward compat (ARCH-08)
# ============================================================================

test_help_queen_commands_section() {
    local output
    output=$(bash "$AETHER_UTILS_SOURCE" help 2>&1)

    # Verify sections field exists
    if ! echo "$output" | jq -e '.sections' >/dev/null 2>&1; then
        test_fail "help has 'sections' field" "field missing"
        return 1
    fi

    # Verify Queen Commands section exists
    if ! echo "$output" | jq -e '.sections."Queen Commands"' >/dev/null 2>&1; then
        test_fail "help has 'Queen Commands' section" "section missing"
        return 1
    fi

    # Verify queen-init is in Queen Commands section with a description
    local has_queen_init
    has_queen_init=$(echo "$output" | jq '[.sections."Queen Commands"[] | select(.name == "queen-init")] | length')
    if [[ "$has_queen_init" != "1" ]]; then
        test_fail "queen-init in Queen Commands section" "not found"
        return 1
    fi

    # Verify backward compat: flat commands array still has queen-init
    if ! echo "$output" | jq -e '.commands | index("queen-init")' >/dev/null 2>&1; then
        test_fail "queen-init in flat commands array (backward compat)" "not found"
        return 1
    fi

    return 0
}

# ============================================================================
# Main Test Runner
# ============================================================================

main() {
    log "${YELLOW}=== Aether Utils Integration Tests ===${NC}"
    log "Testing: $AETHER_UTILS_SOURCE"
    log ""

    # Run all tests
    run_test "test_help" "help returns valid JSON with commands"
    run_test "test_version" "version returns ok:true with 1.0.0"
    run_test "test_validate_state_colony" "validate-state colony validates COLONY_STATE.json"
    run_test "test_validate_state_constraints" "validate-state constraints validates constraints.json"
    run_test "test_validate_state_missing" "validate-state handles missing files"
    run_test "test_activity_log_init" "activity-log-init creates activity.log"
    run_test "test_activity_log_read" "activity-log-read returns log content"
    run_test "test_flag_list_empty" "flag-list returns empty array when no flags"
    run_test "test_flag_add_and_list" "flag-add creates flag, flag-list retrieves it"
    run_test "test_generate_ant_name" "generate-ant-name returns valid name"
    run_test "test_error_summary_empty" "error-summary with empty state"
    run_test "test_invalid_subcommand" "invalid subcommand returns error"
    run_test "test_check_antipattern" "check-antipattern analyzes files"
    run_test "test_bootstrap_system" "bootstrap-system handles missing hub gracefully"

    # ERR-01: fallback json_err tests
    run_test "test_fallback_json_err" "fallback json_err emits code and message fields without error-handler.sh"
    run_test "test_fallback_json_err_single_arg" "fallback json_err single-arg uses provided code and default message"

    # ARCH-01: queen-init template resolution tests
    run_test "test_queen_init_template_hub_path" "queen-init finds template via hub path (npm-install scenario)"
    run_test "test_queen_init_template_not_found_message" "queen-init error message is actionable when no template found"

    # ERR-03/04: regression grep and runtime error code tests
    run_test "test_no_bare_string_json_err_calls" "no bare-string json_err calls in aether-utils.sh (ERR-03 regression)"
    run_test "test_no_bare_string_json_err_in_chamber_scripts" "chamber scripts bare-string count does not exceed known baseline (ERR-03)"
    run_test "test_flag_resolve_missing_flags_file_error_code" "flag-resolve missing flags.json returns E_FILE_NOT_FOUND (ERR-04)"
    run_test "test_flag_add_missing_args_error_code" "flag-add missing args returns E_VALIDATION_FAILED (ERR-04)"
    run_test "test_flag_add_lock_failure_error_code" "flag-add with held lock returns E_LOCK_FAILED (ERR-04)"

    # ARCH-09/10/03: startup ordering, composed trap, spawn-tree rotation regression tests
    run_test "test_feature_detection_after_fallbacks" "feature detection block is after fallback json_err (ARCH-09)"
    run_test "test_composed_exit_trap_exists" "_aether_exit_cleanup calls both cleanup_locks and cleanup_temp_files (ARCH-10)"
    run_test "test_spawn_tree_rotation_exists" "_rotate_spawn_tree function exists with archive reference (ARCH-03)"

    # ARCH-06/02: queen-read validation gates and validate-state schema migration (Phase 18-04)
    run_test "test_queen_read_validates_metadata" "queen-read has JSON validation gates for metadata and assembled result (ARCH-06)"
    run_test "test_validate_state_has_schema_migration" "validate-state has _migrate_colony_state with W_MIGRATED notification (ARCH-02)"

    # ARCH-07/04: model-get subprocess pattern and spawn failure event logging (Phase 18-02)
    run_test "test_model_get_no_exec_pattern" "model-get and model-list do not use exec bash model-profile (ARCH-07)"
    run_test "test_model_get_error_has_try_suggestion" "model-get error message includes Try: suggestion (ARCH-07)"

    # ARCH-08: help sections with Queen Commands group and backward compat (Phase 18-03)
    run_test "test_help_queen_commands_section" "help has Queen Commands section with backward-compat flat commands array (ARCH-08)"

    # Print summary
    test_summary
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
