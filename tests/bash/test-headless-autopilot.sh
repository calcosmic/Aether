#!/usr/bin/env bash
# Headless Autopilot Tests
# Tests pending-decision-add, pending-decision-list, pending-decision-resolve,
# autopilot-headless-check, and autopilot-set-headless subcommands

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

    # Copy utils directory (contains _pending_decision_* and _autopilot_* functions)
    local utils_source="$(dirname "$AETHER_UTILS_SOURCE")/utils"
    if [[ -d "$utils_source" ]]; then
        cp -r "$utils_source" "$tmp_dir/.aether/"
    fi

    # Copy exchange directory if it exists (needed for XML functions)
    local exchange_source="$(dirname "$AETHER_UTILS_SOURCE")/exchange"
    if [[ -d "$exchange_source" ]]; then
        cp -r "$exchange_source" "$tmp_dir/.aether/"
    fi

    # Copy schemas directory if it exists (needed for XML validation)
    local schemas_source="$(dirname "$AETHER_UTILS_SOURCE")/schemas"
    if [[ -d "$schemas_source" ]]; then
        cp -r "$schemas_source" "$tmp_dir/.aether/"
    fi

    echo "$tmp_dir"
}

# ============================================================================
# pending-decision-add tests
# ============================================================================

# Test: Add a decision creates pending-decisions.json
test_pending_decision_add_creates_file() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" pending-decision-add \
        --type "replan" \
        --description "Need to re-evaluate phase plan" 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code — output: $output"
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

    if [[ ! -f "$tmp_dir/.aether/data/pending-decisions.json" ]]; then
        test_fail "pending-decisions.json created" "file not found"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify file has version field
    local version
    version=$(cat "$tmp_dir/.aether/data/pending-decisions.json" | jq -r '.version')
    if [[ "$version" != "1.0" ]]; then
        test_fail "file version=1.0" "version=$version"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# Test: Add with all options (--type, --description, --phase, --source)
test_pending_decision_add_all_options() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" pending-decision-add \
        --type "escalation" \
        --description "We need to pick between Postgres and SQLite" \
        --phase "2" \
        --source "chaos-agent" 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code — output: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify the id starts with pd_
    local stored_id
    stored_id=$(echo "$output" | jq -r '.result.id')
    if [[ ! "$stored_id" =~ ^pd_ ]]; then
        test_fail "id starts with pd_" "id=$stored_id"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify decision_count is 1
    local count
    count=$(echo "$output" | jq -r '.result.decision_count')
    if [[ "$count" != "1" ]]; then
        test_fail "decision_count=1" "decision_count=$count"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify stored fields
    local decisions
    decisions=$(cat "$tmp_dir/.aether/data/pending-decisions.json")

    local stored_type stored_desc stored_source stored_resolved
    stored_type=$(echo "$decisions" | jq -r '.decisions[0].type')
    stored_desc=$(echo "$decisions" | jq -r '.decisions[0].description')
    stored_source=$(echo "$decisions" | jq -r '.decisions[0].source')
    stored_resolved=$(echo "$decisions" | jq -r '.decisions[0].resolved')

    local failed=false
    [[ "$stored_type" == "escalation" ]] || { test_fail "type=escalation" "type=$stored_type"; failed=true; }
    [[ "$stored_desc" == "We need to pick between Postgres and SQLite" ]] || { test_fail "description stored" "description=$stored_desc"; failed=true; }
    [[ "$stored_source" == "chaos-agent" ]] || { test_fail "source=chaos-agent" "source=$stored_source"; failed=true; }
    [[ "$stored_resolved" == "false" ]] || { test_fail "resolved=false" "resolved=$stored_resolved"; failed=true; }

    rm -rf "$tmp_dir"
    [[ "$failed" == "false" ]]
}

# Test: Add multiple decisions — count increments
test_pending_decision_add_multiple() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    bash "$tmp_dir/.aether/aether-utils.sh" pending-decision-add \
        --type "replan" --description "First decision" 2>&1 >/dev/null
    bash "$tmp_dir/.aether/aether-utils.sh" pending-decision-add \
        --type "escalation" --description "Second decision" 2>&1 >/dev/null

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" pending-decision-add \
        --type "replan" --description "Third decision" 2>&1)

    local count
    count=$(echo "$output" | jq '.result.decision_count')

    if [[ "$count" != "3" ]]; then
        test_fail "decision_count=3" "decision_count=$count"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# Test: Add with missing required fields should fail gracefully
test_pending_decision_add_missing_type() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    set +e
    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" pending-decision-add \
        --description "Missing type" 2>&1)
    local exit_code=$?
    set -e

    if [[ "$exit_code" -eq 0 ]]; then
        test_fail "non-zero exit" "exit code 0 — output: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON error response" "invalid JSON: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# pending-decision-list tests
# ============================================================================

# Test: List when no decisions exist returns empty result
test_pending_decision_list_empty() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" pending-decision-list 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code — output: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    local total unresolved
    total=$(echo "$output" | jq '.result.total')
    unresolved=$(echo "$output" | jq '.result.unresolved')

    local failed=false
    [[ "$total" == "0" ]] || { test_fail "total=0" "total=$total"; failed=true; }
    [[ "$unresolved" == "0" ]] || { test_fail "unresolved=0" "unresolved=$unresolved"; failed=true; }

    rm -rf "$tmp_dir"
    [[ "$failed" == "false" ]]
}

# Test: List shows all unresolved by default
test_pending_decision_list_shows_unresolved() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    bash "$tmp_dir/.aether/aether-utils.sh" pending-decision-add \
        --type "replan" --description "Decision A" 2>&1 >/dev/null
    bash "$tmp_dir/.aether/aether-utils.sh" pending-decision-add \
        --type "escalation" --description "Decision B" 2>&1 >/dev/null

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" pending-decision-list 2>&1)

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    local count
    count=$(echo "$output" | jq '.result.decisions | length')
    if [[ "$count" != "2" ]]; then
        test_fail "list count=2" "count=$count"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# Test: List with --type filter returns only matching type
test_pending_decision_list_type_filter() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    bash "$tmp_dir/.aether/aether-utils.sh" pending-decision-add \
        --type "replan" --description "Replan A" 2>&1 >/dev/null
    bash "$tmp_dir/.aether/aether-utils.sh" pending-decision-add \
        --type "escalation" --description "Escalation" 2>&1 >/dev/null
    bash "$tmp_dir/.aether/aether-utils.sh" pending-decision-add \
        --type "replan" --description "Replan B" 2>&1 >/dev/null

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" pending-decision-list \
        --type "replan" 2>&1)

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    local count
    count=$(echo "$output" | jq '.result.decisions | length')
    if [[ "$count" != "2" ]]; then
        test_fail "filtered count=2 (replan only)" "count=$count"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# Test: List with --unresolved flag returns only unresolved decisions
test_pending_decision_list_unresolved_flag() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Add two decisions
    local add_out
    add_out=$(bash "$tmp_dir/.aether/aether-utils.sh" pending-decision-add \
        --type "replan" --description "Decision to resolve" 2>&1)
    bash "$tmp_dir/.aether/aether-utils.sh" pending-decision-add \
        --type "escalation" --description "Decision to keep" 2>&1 >/dev/null

    # Resolve the first one
    local first_id
    first_id=$(echo "$add_out" | jq -r '.result.id')
    bash "$tmp_dir/.aether/aether-utils.sh" pending-decision-resolve \
        --id "$first_id" --resolution "We chose option A" 2>&1 >/dev/null

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" pending-decision-list \
        --unresolved 2>&1)

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    local count
    count=$(echo "$output" | jq '.result.decisions | length')
    if [[ "$count" != "1" ]]; then
        test_fail "unresolved count=1" "count=$count"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify the remaining one is unresolved
    local resolved
    resolved=$(echo "$output" | jq -r '.result.decisions[0].resolved')
    if [[ "$resolved" != "false" ]]; then
        test_fail "resolved=false" "resolved=$resolved"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# pending-decision-resolve tests
# ============================================================================

# Test: Resolve a decision marks it as resolved
test_pending_decision_resolve() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    local add_out
    add_out=$(bash "$tmp_dir/.aether/aether-utils.sh" pending-decision-add \
        --type "replan" --description "Resolve me" 2>&1)
    local decision_id
    decision_id=$(echo "$add_out" | jq -r '.result.id')

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" pending-decision-resolve \
        --id "$decision_id" \
        --resolution "We picked option A because it was simpler" 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code — output: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify the response has resolved=true
    local result_resolved
    result_resolved=$(echo "$output" | jq -r '.result.resolved')
    if [[ "$result_resolved" != "true" ]]; then
        test_fail "result.resolved=true" "resolved=$result_resolved"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify the decision is now resolved in the file
    local decisions
    decisions=$(cat "$tmp_dir/.aether/data/pending-decisions.json")
    local stored_resolved stored_resolution
    stored_resolved=$(echo "$decisions" | jq -r ".decisions[] | select(.id == \"$decision_id\") | .resolved")
    stored_resolution=$(echo "$decisions" | jq -r ".decisions[] | select(.id == \"$decision_id\") | .resolution")

    local failed=false
    [[ "$stored_resolved" == "true" ]] || { test_fail "stored resolved=true" "resolved=$stored_resolved"; failed=true; }
    [[ "$stored_resolution" == "We picked option A because it was simpler" ]] || { test_fail "resolution stored" "resolution=$stored_resolution"; failed=true; }

    rm -rf "$tmp_dir"
    [[ "$failed" == "false" ]]
}

# Test: Resolve non-existent id returns error
test_pending_decision_resolve_nonexistent() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Add at least one decision so the file exists
    bash "$tmp_dir/.aether/aether-utils.sh" pending-decision-add \
        --type "replan" --description "Exists" 2>&1 >/dev/null

    set +e
    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" pending-decision-resolve \
        --id "pd_999_nonexistent" \
        --resolution "some resolution" 2>&1)
    local exit_code=$?
    set -e

    if [[ "$exit_code" -eq 0 ]]; then
        test_fail "non-zero exit" "exit code 0 — output: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON error response" "invalid JSON: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# autopilot-headless-check tests
# ============================================================================

# Test: Check when no run-state exists defaults to false
test_autopilot_headless_check_no_state() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" autopilot-headless-check 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code — output: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    local headless
    headless=$(echo "$output" | jq -r '.result.headless')
    if [[ "$headless" != "false" ]]; then
        test_fail "headless=false (no run-state)" "headless=$headless"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# Test: Check after setting headless to true returns true
test_autopilot_headless_check_after_set_true() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Initialize autopilot first so run-state.json exists
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-init \
        --total-phases 4 --start-phase 1 2>&1 >/dev/null

    # Set headless to true
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-set-headless true 2>&1 >/dev/null

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" autopilot-headless-check 2>&1)

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    local headless
    headless=$(echo "$output" | jq -r '.result.headless')
    if [[ "$headless" != "true" ]]; then
        test_fail "headless=true" "headless=$headless"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# autopilot-set-headless tests
# ============================================================================

# Test: Set headless to true persists in run-state.json
test_autopilot_set_headless_true() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Initialize autopilot
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-init \
        --total-phases 4 --start-phase 1 2>&1 >/dev/null

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" autopilot-set-headless true 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code — output: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Response should have updated=true and headless=true
    local result_headless result_updated
    result_headless=$(echo "$output" | jq -r '.result.headless')
    result_updated=$(echo "$output" | jq -r '.result.updated')

    local failed=false
    [[ "$result_headless" == "true" ]] || { test_fail "result.headless=true" "headless=$result_headless"; failed=true; }
    [[ "$result_updated" == "true" ]] || { test_fail "result.updated=true" "updated=$result_updated"; failed=true; }

    # Verify in run-state.json
    local state_headless
    state_headless=$(cat "$tmp_dir/.aether/data/run-state.json" | jq -r '.headless')
    [[ "$state_headless" == "true" ]] || { test_fail "run-state.json headless=true" "headless=$state_headless"; failed=true; }

    rm -rf "$tmp_dir"
    [[ "$failed" == "false" ]]
}

# Test: Set headless to false persists in run-state.json
test_autopilot_set_headless_false() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Initialize autopilot and set to true first
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-init \
        --total-phases 4 --start-phase 1 2>&1 >/dev/null
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-set-headless true 2>&1 >/dev/null

    # Now set back to false
    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" autopilot-set-headless false 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code — output: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    local state_headless
    state_headless=$(cat "$tmp_dir/.aether/data/run-state.json" | jq -r '.headless')

    if [[ "$state_headless" != "false" ]]; then
        test_fail "run-state.json headless=false" "headless=$state_headless"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Help registration regression tests
# ============================================================================

test_new_commands_in_help() {
    local output
    output=$(bash "$AETHER_UTILS_SOURCE" help 2>&1)

    local failed=false

    for cmd in \
        "pending-decision-add" \
        "pending-decision-list" \
        "pending-decision-resolve" \
        "autopilot-headless-check" \
        "autopilot-set-headless"; do
        local found
        found=$(echo "$output" | jq --arg cmd "$cmd" '.commands | index($cmd) != null')
        [[ "$found" == "true" ]] || { test_fail "$cmd in commands" "not found"; failed=true; }
    done

    [[ "$failed" == "false" ]]
}

# ============================================================================
# Main test runner
# ============================================================================
main() {
    log "${YELLOW}=== Headless Autopilot Tests ===${NC}"

    # pending-decision-add tests
    run_test "test_pending_decision_add_creates_file" "pending-decision-add creates pending-decisions.json"
    run_test "test_pending_decision_add_all_options" "pending-decision-add stores all options"
    run_test "test_pending_decision_add_multiple" "pending-decision-add increments count"
    run_test "test_pending_decision_add_missing_type" "pending-decision-add fails gracefully on missing type"

    # pending-decision-list tests
    run_test "test_pending_decision_list_empty" "pending-decision-list returns empty when no decisions"
    run_test "test_pending_decision_list_shows_unresolved" "pending-decision-list shows all unresolved by default"
    run_test "test_pending_decision_list_type_filter" "pending-decision-list filters by --type"
    run_test "test_pending_decision_list_unresolved_flag" "pending-decision-list --unresolved excludes resolved"

    # pending-decision-resolve tests
    run_test "test_pending_decision_resolve" "pending-decision-resolve marks decision as resolved"
    run_test "test_pending_decision_resolve_nonexistent" "pending-decision-resolve returns error for unknown id"

    # autopilot-headless-check tests
    run_test "test_autopilot_headless_check_no_state" "autopilot-headless-check defaults to false with no run-state"
    run_test "test_autopilot_headless_check_after_set_true" "autopilot-headless-check returns true after set"

    # autopilot-set-headless tests
    run_test "test_autopilot_set_headless_true" "autopilot-set-headless true persists in run-state.json"
    run_test "test_autopilot_set_headless_false" "autopilot-set-headless false persists in run-state.json"

    # Help registration
    run_test "test_new_commands_in_help" "new headless commands appear in help output"

    # Print summary
    test_summary
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
