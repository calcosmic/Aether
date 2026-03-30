#!/usr/bin/env bash
# Council Module Smoke Tests
# Tests council.sh deliberation functions via aether-utils.sh subcommands

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
AETHER_UTILS_SOURCE="$PROJECT_ROOT/.aether/aether-utils.sh"

source "$SCRIPT_DIR/test-helpers.sh"

require_jq

if [[ ! -f "$AETHER_UTILS_SOURCE" ]]; then
    log_error "aether-utils.sh not found at: $AETHER_UTILS_SOURCE"
    exit 1
fi

# ============================================================================
# Helper: Create isolated test environment
# ============================================================================
setup_council_env() {
    local tmp_dir
    tmp_dir=$(mktemp -d)
    mkdir -p "$tmp_dir/.aether/data" "$tmp_dir/.aether/utils"

    cp "$AETHER_UTILS_SOURCE" "$tmp_dir/.aether/aether-utils.sh"
    chmod +x "$tmp_dir/.aether/aether-utils.sh"

    local utils_source
    utils_source="$(dirname "$AETHER_UTILS_SOURCE")/utils"
    if [[ -d "$utils_source" ]]; then
        cp -r "$utils_source" "$tmp_dir/.aether/"
    fi

    local exchange_source
    exchange_source="$(dirname "$AETHER_UTILS_SOURCE")/exchange"
    if [[ -d "$exchange_source" ]]; then
        cp -r "$exchange_source" "$tmp_dir/.aether/"
    fi

    cat > "$tmp_dir/.aether/data/COLONY_STATE.json" << 'CSEOF'
{
  "version": "3.0",
  "goal": "Test council module",
  "state": "READY",
  "current_phase": 1,
  "session_id": "test-session",
  "initialized_at": "2026-01-01T00:00:00Z",
  "build_started_at": null,
  "plan": { "phases": [{ "id": 1, "name": "Test Phase", "status": "pending" }] },
  "memory": { "phase_learnings": [], "decisions": [], "instincts": [] },
  "errors": { "records": [], "flagged_patterns": [] },
  "events": [],
  "signals": [],
  "graveyards": []
}
CSEOF

    echo "$tmp_dir"
}

run_council_cmd() {
    local tmp_dir="$1"
    shift
    AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" bash "$tmp_dir/.aether/aether-utils.sh" "$@" 2>/dev/null
}

# ============================================================================
# Test: council.sh module file exists and has valid syntax
# ============================================================================
test_module_exists() {
    local module_path="$PROJECT_ROOT/.aether/utils/council.sh"

    assert_file_exists "$module_path" || return 1
    bash -n "$module_path" 2>/dev/null || return 1
}

# ============================================================================
# Test: council-deliberate creates a deliberation record
# ============================================================================
test_council_deliberate() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    local result
    result=$(run_council_cmd "$tmp_dir" council-deliberate --proposal "Should we use microservices?")

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }
    assert_json_field_equals "$result" ".result.status" "pending" || { rm -rf "$tmp_dir"; return 1; }
    assert_json_field_equals "$result" ".result.budget" "3" || { rm -rf "$tmp_dir"; return 1; }

    local id
    id=$(echo "$result" | jq -r '.result.id')
    [[ "$id" == delib_* ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: council-deliberate respects --budget flag
# ============================================================================
test_council_deliberate_budget() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    local result
    result=$(run_council_cmd "$tmp_dir" council-deliberate --proposal "Test budget" --budget 5)

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }
    assert_json_field_equals "$result" ".result.budget" "5" || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: council-deliberate persists to deliberations.json
# ============================================================================
test_council_deliberate_persists() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    run_council_cmd "$tmp_dir" council-deliberate --proposal "Persistence test" > /dev/null

    local delib_file="$tmp_dir/.aether/data/council/deliberations.json"
    assert_file_exists "$delib_file" || { rm -rf "$tmp_dir"; return 1; }

    local version
    version=$(jq -r '.version' "$delib_file")
    [[ "$version" == "1.0" ]] || { rm -rf "$tmp_dir"; return 1; }

    local count
    count=$(jq '.deliberations | length' "$delib_file")
    [[ "$count" -eq 1 ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: council-advocate records argument for a deliberation
# ============================================================================
test_council_advocate() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    local delib_result
    delib_result=$(run_council_cmd "$tmp_dir" council-deliberate --proposal "Advocate test")
    local delib_id
    delib_id=$(echo "$delib_result" | jq -r '.result.id')

    local result
    result=$(run_council_cmd "$tmp_dir" council-advocate --deliberation-id "$delib_id" --argument "This is a great idea because it scales")

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }
    assert_json_field_equals "$result" ".result.role" "advocate" || { rm -rf "$tmp_dir"; return 1; }
    assert_json_field_equals "$result" ".result.recorded" "true" || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: council-challenger records argument against a deliberation
# ============================================================================
test_council_challenger() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    local delib_result
    delib_result=$(run_council_cmd "$tmp_dir" council-deliberate --proposal "Challenger test")
    local delib_id
    delib_id=$(echo "$delib_result" | jq -r '.result.id')

    local result
    result=$(run_council_cmd "$tmp_dir" council-challenger --deliberation-id "$delib_id" --argument "This adds operational complexity")

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }
    assert_json_field_equals "$result" ".result.role" "challenger" || { rm -rf "$tmp_dir"; return 1; }
    assert_json_field_equals "$result" ".result.recorded" "true" || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: council-sage synthesizes and marks deliberation complete
# ============================================================================
test_council_sage() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    local delib_result
    delib_result=$(run_council_cmd "$tmp_dir" council-deliberate --proposal "Sage test")
    local delib_id
    delib_id=$(echo "$delib_result" | jq -r '.result.id')

    local result
    result=$(run_council_cmd "$tmp_dir" council-sage \
        --deliberation-id "$delib_id" \
        --synthesis "Both sides have merit" \
        --recommendation "Proceed with caution")

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }
    assert_json_field_equals "$result" ".result.role" "sage" || { rm -rf "$tmp_dir"; return 1; }
    assert_json_field_equals "$result" ".result.deliberation_complete" "true" || { rm -rf "$tmp_dir"; return 1; }

    local rec
    rec=$(echo "$result" | jq -r '.result.recommendation')
    [[ "$rec" == "Proceed with caution" ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: council-sage marks deliberation status as complete in storage
# ============================================================================
test_council_sage_marks_complete() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    local delib_result
    delib_result=$(run_council_cmd "$tmp_dir" council-deliberate --proposal "Complete status test")
    local delib_id
    delib_id=$(echo "$delib_result" | jq -r '.result.id')

    run_council_cmd "$tmp_dir" council-sage \
        --deliberation-id "$delib_id" \
        --synthesis "Balanced view" \
        --recommendation "Go ahead" > /dev/null

    local delib_file="$tmp_dir/.aether/data/council/deliberations.json"
    local status
    status=$(jq -r --arg id "$delib_id" '.deliberations[] | select(.id == $id) | .status' "$delib_file")
    [[ "$status" == "complete" ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: council-history returns deliberations list
# ============================================================================
test_council_history() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    run_council_cmd "$tmp_dir" council-deliberate --proposal "History test 1" > /dev/null
    run_council_cmd "$tmp_dir" council-deliberate --proposal "History test 2" > /dev/null

    local result
    result=$(run_council_cmd "$tmp_dir" council-history)

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    local total
    total=$(echo "$result" | jq -r '.result.total')
    [[ "$total" -ge 2 ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: council-history respects --limit flag
# ============================================================================
test_council_history_limit() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    run_council_cmd "$tmp_dir" council-deliberate --proposal "Limit test 1" > /dev/null
    run_council_cmd "$tmp_dir" council-deliberate --proposal "Limit test 2" > /dev/null
    run_council_cmd "$tmp_dir" council-deliberate --proposal "Limit test 3" > /dev/null

    local result
    result=$(run_council_cmd "$tmp_dir" council-history --limit 2)

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    local count
    count=$(echo "$result" | jq '.result.deliberations | length')
    [[ "$count" -le 2 ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: council-budget-check returns allowed status
# ============================================================================
test_council_budget_check() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    local result
    result=$(run_council_cmd "$tmp_dir" council-budget-check --budget 3)

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    # result.allowed must be a boolean
    local allowed
    allowed=$(echo "$result" | jq -r '.result.allowed')
    [[ "$allowed" == "true" || "$allowed" == "false" ]] || { rm -rf "$tmp_dir"; return 1; }

    # result.budget must be 3
    assert_json_field_equals "$result" ".result.budget" "3" || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: council-history gracefully handles no deliberations file
# ============================================================================
test_council_history_empty() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    local result
    result=$(run_council_cmd "$tmp_dir" council-history)

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }
    assert_json_field_equals "$result" ".result.total" "0" || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: council-advocate fails gracefully with missing deliberation-id
# ============================================================================
test_council_advocate_missing_id() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    # Capture both stdout and stderr; command will exit non-zero on validation error
    local result
    result=$(AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" \
        bash "$tmp_dir/.aether/aether-utils.sh" council-advocate --argument "test" 2>&1) || true

    # Must not return ok:true — either ok:false or empty (error on stderr)
    local ok
    ok=$(echo "$result" | jq -r '.ok' 2>/dev/null || echo "")
    [[ "$ok" != "true" ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: council-deliberate fails gracefully with missing proposal
# ============================================================================
test_council_deliberate_missing_proposal() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    # Capture both stdout and stderr; command will exit non-zero on validation error
    local result
    result=$(AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" \
        bash "$tmp_dir/.aether/aether-utils.sh" council-deliberate 2>&1) || true

    # Must not return ok:true — either ok:false or empty (error on stderr)
    local ok
    ok=$(echo "$result" | jq -r '.ok' 2>/dev/null || echo "")
    [[ "$ok" != "true" ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: council subcommands appear in help manifest
# ============================================================================
test_council_in_help() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    local help_output
    help_output=$(run_council_cmd "$tmp_dir" help)

    echo "$help_output" | jq -r '.commands[]' | grep -q "council-deliberate" || { rm -rf "$tmp_dir"; return 1; }
    echo "$help_output" | jq -r '.commands[]' | grep -q "council-advocate" || { rm -rf "$tmp_dir"; return 1; }
    echo "$help_output" | jq -r '.commands[]' | grep -q "council-challenger" || { rm -rf "$tmp_dir"; return 1; }
    echo "$help_output" | jq -r '.commands[]' | grep -q "council-sage" || { rm -rf "$tmp_dir"; return 1; }
    echo "$help_output" | jq -r '.commands[]' | grep -q "council-history" || { rm -rf "$tmp_dir"; return 1; }
    echo "$help_output" | jq -r '.commands[]' | grep -q "council-budget-check" || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Run all tests
# ============================================================================
echo "=== Council Module Smoke Tests ==="
echo ""

run_test test_module_exists "council.sh exists and passes syntax check"
run_test test_council_deliberate "council-deliberate creates deliberation with pending status"
run_test test_council_deliberate_budget "council-deliberate respects --budget flag"
run_test test_council_deliberate_persists "council-deliberate persists to deliberations.json"
run_test test_council_advocate "council-advocate records argument with correct role"
run_test test_council_challenger "council-challenger records argument with correct role"
run_test test_council_sage "council-sage synthesizes and marks complete"
run_test test_council_sage_marks_complete "council-sage marks deliberation status as complete"
run_test test_council_history "council-history returns deliberation list"
run_test test_council_history_limit "council-history respects --limit flag"
run_test test_council_budget_check "council-budget-check returns allowed status"
run_test test_council_history_empty "council-history gracefully handles no deliberations file"
run_test test_council_advocate_missing_id "council-advocate fails gracefully with missing id"
run_test test_council_deliberate_missing_proposal "council-deliberate fails gracefully with missing proposal"
run_test test_council_in_help "council subcommands appear in help manifest"

test_summary
