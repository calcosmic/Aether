#!/usr/bin/env bash
# Council Expansion Tests
# Tests council subcommands: council-deliberate, council-advocate,
# council-challenger, council-sage, council-history, council-budget-check

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

    # Write a minimal COLONY_STATE.json
    cat > "$tmp_dir/.aether/data/COLONY_STATE.json" << 'CSEOF'
{
  "version": "3.0",
  "goal": "Test council expansion",
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
    AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" \
        bash "$tmp_dir/.aether/aether-utils.sh" "$@" 2>/dev/null
}

run_council_cmd_with_stderr() {
    local tmp_dir="$1"
    shift
    AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" \
        bash "$tmp_dir/.aether/aether-utils.sh" "$@" 2>&1
}

# ============================================================================
# council-deliberate tests
# ============================================================================

test_deliberate_creates_with_id_and_pending_status() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    local result
    result=$(run_council_cmd "$tmp_dir" council-deliberate --proposal "Should we use TypeScript?")

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }
    assert_json_field_equals "$result" ".result.status" "pending" || { rm -rf "$tmp_dir"; return 1; }

    local id
    id=$(echo "$result" | jq -r '.result.id')
    [[ "$id" == delib_* ]] || {
        log_error "Expected id to start with 'delib_', got: $id"
        rm -rf "$tmp_dir"
        return 1
    }

    local proposal
    proposal=$(echo "$result" | jq -r '.result.proposal')
    [[ "$proposal" == "Should we use TypeScript?" ]] || {
        log_error "Expected proposal to match, got: $proposal"
        rm -rf "$tmp_dir"
        return 1
    }

    rm -rf "$tmp_dir"
}

test_deliberate_custom_budget() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    local result
    result=$(run_council_cmd "$tmp_dir" council-deliberate --proposal "Budget test" --budget 5)

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    local budget
    budget=$(echo "$result" | jq -r '.result.budget')
    [[ "$budget" == "5" ]] || {
        log_error "Expected budget=5, got: $budget"
        rm -rf "$tmp_dir"
        return 1
    }

    rm -rf "$tmp_dir"
}

test_deliberate_missing_proposal_fails() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    local result
    result=$(run_council_cmd_with_stderr "$tmp_dir" council-deliberate)

    # Should fail (ok:false or non-zero exit) when no proposal provided
    local ok
    ok=$(echo "$result" | jq -r '.ok' 2>/dev/null || echo "false")
    [[ "$ok" == "false" ]] || {
        log_error "Expected ok:false for missing proposal, got ok=$ok"
        rm -rf "$tmp_dir"
        return 1
    }

    rm -rf "$tmp_dir"
}

# ============================================================================
# council-advocate tests
# ============================================================================

test_advocate_records_argument() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    # First create a deliberation
    local delib_result
    delib_result=$(run_council_cmd "$tmp_dir" council-deliberate --proposal "Advocate test proposal")
    local delib_id
    delib_id=$(echo "$delib_result" | jq -r '.result.id')

    local result
    result=$(run_council_cmd "$tmp_dir" council-advocate \
        --deliberation-id "$delib_id" \
        --argument "TypeScript improves code quality and catches bugs early")

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }
    assert_json_field_equals "$result" ".result.role" "advocate" || { rm -rf "$tmp_dir"; return 1; }
    assert_json_field_equals "$result" ".result.recorded" "true" || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

test_advocate_nonexistent_deliberation_errors() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    local result
    result=$(run_council_cmd_with_stderr "$tmp_dir" council-advocate \
        --deliberation-id "delib_nonexistent_123" \
        --argument "This should fail")

    local ok
    ok=$(echo "$result" | jq -r '.ok' 2>/dev/null || echo "false")
    [[ "$ok" == "false" ]] || {
        log_error "Expected ok:false for nonexistent deliberation, got ok=$ok"
        rm -rf "$tmp_dir"
        return 1
    }

    rm -rf "$tmp_dir"
}

# ============================================================================
# council-challenger tests
# ============================================================================

test_challenger_records_argument() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    local delib_result
    delib_result=$(run_council_cmd "$tmp_dir" council-deliberate --proposal "Challenger test proposal")
    local delib_id
    delib_id=$(echo "$delib_result" | jq -r '.result.id')

    local result
    result=$(run_council_cmd "$tmp_dir" council-challenger \
        --deliberation-id "$delib_id" \
        --argument "TypeScript adds complexity and slows down prototyping")

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }
    assert_json_field_equals "$result" ".result.role" "challenger" || { rm -rf "$tmp_dir"; return 1; }
    assert_json_field_equals "$result" ".result.recorded" "true" || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# council-sage tests
# ============================================================================

test_sage_records_synthesis_and_marks_complete() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    local delib_result
    delib_result=$(run_council_cmd "$tmp_dir" council-deliberate --proposal "Sage test proposal")
    local delib_id
    delib_id=$(echo "$delib_result" | jq -r '.result.id')

    # Record advocate and challenger first
    run_council_cmd "$tmp_dir" council-advocate \
        --deliberation-id "$delib_id" \
        --argument "Pro argument" > /dev/null

    run_council_cmd "$tmp_dir" council-challenger \
        --deliberation-id "$delib_id" \
        --argument "Con argument" > /dev/null

    local result
    result=$(run_council_cmd "$tmp_dir" council-sage \
        --deliberation-id "$delib_id" \
        --synthesis "Both sides have merit; use TypeScript for core modules" \
        --recommendation "adopt-typescript")

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }
    assert_json_field_equals "$result" ".result.role" "sage" || { rm -rf "$tmp_dir"; return 1; }
    assert_json_field_equals "$result" ".result.deliberation_complete" "true" || { rm -rf "$tmp_dir"; return 1; }

    local recommendation
    recommendation=$(echo "$result" | jq -r '.result.recommendation')
    [[ -n "$recommendation" ]] || {
        log_error "Expected recommendation to be non-empty"
        rm -rf "$tmp_dir"
        return 1
    }

    rm -rf "$tmp_dir"
}

# ============================================================================
# council-history tests
# ============================================================================

test_history_empty_returns_total_zero() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    local result
    result=$(run_council_cmd "$tmp_dir" council-history)

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }
    assert_json_field_equals "$result" ".result.total" "0" || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

test_history_with_entries_returns_correct_total() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    # Create two deliberations
    run_council_cmd "$tmp_dir" council-deliberate --proposal "First proposal" > /dev/null
    run_council_cmd "$tmp_dir" council-deliberate --proposal "Second proposal" > /dev/null

    local result
    result=$(run_council_cmd "$tmp_dir" council-history)

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    local total
    total=$(echo "$result" | jq -r '.result.total')
    [[ "$total" -ge 2 ]] || {
        log_error "Expected total >= 2, got: $total"
        rm -rf "$tmp_dir"
        return 1
    }

    rm -rf "$tmp_dir"
}

test_history_with_limit_flag() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    # Create three deliberations
    run_council_cmd "$tmp_dir" council-deliberate --proposal "First" > /dev/null
    run_council_cmd "$tmp_dir" council-deliberate --proposal "Second" > /dev/null
    run_council_cmd "$tmp_dir" council-deliberate --proposal "Third" > /dev/null

    local result
    result=$(run_council_cmd "$tmp_dir" council-history --limit 2)

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    local returned_count
    returned_count=$(echo "$result" | jq '.result.deliberations | length')
    [[ "$returned_count" -le 2 ]] || {
        log_error "Expected at most 2 deliberations with --limit 2, got: $returned_count"
        rm -rf "$tmp_dir"
        return 1
    }

    rm -rf "$tmp_dir"
}

# ============================================================================
# council-budget-check tests
# ============================================================================

test_budget_check_with_available_budget() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    local result
    result=$(run_council_cmd "$tmp_dir" council-budget-check --budget 10)

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }
    assert_json_field_equals "$result" ".result.allowed" "true" || { rm -rf "$tmp_dir"; return 1; }

    local budget
    budget=$(echo "$result" | jq -r '.result.budget')
    [[ "$budget" == "10" ]] || {
        log_error "Expected budget=10, got: $budget"
        rm -rf "$tmp_dir"
        return 1
    }

    rm -rf "$tmp_dir"
}

test_budget_check_default_budget() {
    local tmp_dir
    tmp_dir=$(setup_council_env)

    local result
    result=$(run_council_cmd "$tmp_dir" council-budget-check)

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    # Should return a budget value (the default)
    local budget
    budget=$(echo "$result" | jq -r '.result.budget')
    [[ "$budget" =~ ^[0-9]+$ ]] || {
        log_error "Expected numeric budget, got: $budget"
        rm -rf "$tmp_dir"
        return 1
    }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Run all tests
# ============================================================================
echo "=== Council Expansion Tests ==="
echo ""

# council-deliberate
run_test test_deliberate_creates_with_id_and_pending_status \
    "council-deliberate: creates deliberation with id and pending status"
run_test test_deliberate_custom_budget \
    "council-deliberate: custom budget is stored"
run_test test_deliberate_missing_proposal_fails \
    "council-deliberate: missing proposal fails gracefully"

# council-advocate
run_test test_advocate_records_argument \
    "council-advocate: records argument (recorded=true, role=advocate)"
run_test test_advocate_nonexistent_deliberation_errors \
    "council-advocate: nonexistent deliberation returns error"

# council-challenger
run_test test_challenger_records_argument \
    "council-challenger: records argument (recorded=true, role=challenger)"

# council-sage
run_test test_sage_records_synthesis_and_marks_complete \
    "council-sage: records synthesis and marks deliberation complete"

# council-history
run_test test_history_empty_returns_total_zero \
    "council-history: empty history returns total=0"
run_test test_history_with_entries_returns_correct_total \
    "council-history: history with entries returns correct total"
run_test test_history_with_limit_flag \
    "council-history: --limit flag caps returned deliberations"

# council-budget-check
run_test test_budget_check_with_available_budget \
    "council-budget-check: available budget returns allowed=true"
run_test test_budget_check_default_budget \
    "council-budget-check: default budget returns numeric value"

test_summary
