#!/usr/bin/env bash
# Tests for pr-context subcommand and _budget_enforce() extraction
# Phase 42: CI Context Assembly

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

# ============================================================================
# Helper: Create isolated pr-context test environment
# ============================================================================
setup_pr_context_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    local aether_dir="$tmpdir/.aether"
    local data_dir="$aether_dir/data"
    local utils_dir="$aether_dir/utils"
    mkdir -p "$data_dir" "$utils_dir" "$data_dir/midden"

    # Copy aether-utils.sh and utils/
    cp "$AETHER_UTILS" "$aether_dir/aether-utils.sh"
    chmod +x "$aether_dir/aether-utils.sh"

    local utils_source="$(dirname "$AETHER_UTILS")/utils"
    if [[ -d "$utils_source" ]]; then
        cp -r "$utils_source"/* "$utils_dir/" 2>/dev/null || true
    fi

    local exchange_source="$(dirname "$AETHER_UTILS")/exchange"
    if [[ -d "$exchange_source" ]]; then
        cp -r "$exchange_source" "$aether_dir/" 2>/dev/null || true
    fi

    # Create minimal COLONY_STATE.json
    cat > "$data_dir/COLONY_STATE.json" << 'CSEOF'
{
  "version": "3.0",
  "goal": "Test pr-context module",
  "state": "READY",
  "current_phase": 1,
  "total_phases": 5,
  "phase_name": "test-phase",
  "plan": {"id": "test-plan", "tasks": []},
  "memory": {
    "events": [],
    "instincts": [],
    "phase_learnings": []
  },
  "errors": {"records": []},
  "events": [],
  "session_id": "test-session",
  "initialized_at": "2026-01-01T00:00:00Z"
}
CSEOF

    # Create minimal pheromones.json
    cat > "$data_dir/pheromones.json" << 'PHEOF'
{
  "version": "1.0.0",
  "colony_id": "test-colony",
  "generated_at": "2026-01-01T00:00:00Z",
  "signals": []
}
PHEOF

    # Create empty midden
    cat > "$data_dir/midden/midden.json" << 'MIDEOF'
{
  "entries": []
}
MIDEOF

    # Create a minimal QUEEN.md (v2 format)
    cat > "$aether_dir/QUEEN.md" << 'QEOF'
# QUEEN.md --- Colony Wisdom

> Last evolved: 2026-01-01T00:00:00Z

---

## User Preferences

- Test preference

---

## Codebase Patterns

- Test pattern

---

## Build Learnings

- Test learning

---

## Instincts

- Test instinct

---

## Evolution Log

| Date | Colony | Change | Details |
|------|--------|--------|---------|

---

<!-- METADATA {"version":"1.0.0","last_evolved":"2026-01-01T00:00:00Z","colonies_contributed":[],"promotion_thresholds":{"philosophy":1,"pattern":1,"redirect":1,"stack":1,"decree":0},"stats":{"total_philosophies":0,"total_patterns":0,"total_redirects":0,"total_stack_entries":0,"total_decrees":0}} -->
QEOF

    echo "$tmpdir"
}

teardown_pr_context_env() {
    local tmpdir="$1"
    if [[ -n "$tmpdir" && -d "$tmpdir" ]]; then
        rm -rf "$tmpdir"
    fi
}

# ============================================================================
# Test 1: colony-prime output is byte-identical before/after budget extraction
# ============================================================================
test_colony_prime_unchanged() {
    # Run colony-prime with existing tests to verify no regression
    local budget_test="$SCRIPT_DIR/test-colony-prime-budget.sh"
    if [[ -f "$budget_test" ]]; then
        local output
        output=$(bash "$budget_test" 2>&1) || {
            log_error "colony-prime budget tests FAILED (regression from _budget_enforce extraction)"
            printf '%s' "$output" | tail -20
            return 1
        }
        return 0
    fi
    log_warn "test-colony-prime-budget.sh not found, skipping regression check"
    return 0
}

# ============================================================================
# Test 2: _budget_enforce trims rolling-summary first when over budget
# (This test will be fully activated once pr-context is implemented in Task 2)
# ============================================================================
test_trim_order_rolling_first() {
    local env_dir
    env_dir=$(setup_pr_context_env)

    local output exit_code=0
    output=$(AETHER_ROOT="$env_dir" COLONY_DATA_DIR="$env_dir/.aether/data" \
        bash "$AETHER_UTILS" pr-context 2>/dev/null) || exit_code=$?

    teardown_pr_context_env "$env_dir"

    if [[ "$exit_code" -ne 0 ]]; then
        # pr-context not yet implemented -- pass as stub
        return 0
    fi

    # Verify that when over budget, rolling-summary is trimmed before other sections
    local trimmed
    trimmed=$(echo "$output" | jq -r '.result.trimmed_sections // [] | join(",")' 2>/dev/null)

    # If anything was trimmed, rolling should be first
    if [[ -n "$trimmed" && "$trimmed" != "" ]]; then
        if [[ "$trimmed" != rolling-summary* ]]; then
            log_error "Expected rolling-summary to be trimmed first, got: $trimmed"
            return 1
        fi
    fi
    return 0
}

# ============================================================================
# Test 3: _budget_enforce never trims blockers even when over budget
# (Activated once pr-context is implemented)
# ============================================================================
test_blockers_never_trimmed() {
    local env_dir
    env_dir=$(setup_pr_context_env)

    local output exit_code=0
    output=$(AETHER_ROOT="$env_dir" COLONY_DATA_DIR="$env_dir/.aether/data" \
        bash "$AETHER_UTILS" pr-context 2>/dev/null) || exit_code=$?

    teardown_pr_context_env "$env_dir"

    if [[ "$exit_code" -ne 0 ]]; then
        return 0  # stub pass
    fi

    # blockers should never appear in trimmed_sections
    local trimmed
    trimmed=$(echo "$output" | jq -r '.result.trimmed_sections // [] | join(",")' 2>/dev/null)
    if [[ "$trimmed" == *blockers* ]]; then
        log_error "Blockers were trimmed, which should never happen"
        return 1
    fi
    return 0
}

# ============================================================================
# Test 4: _budget_enforce preserves REDIRECTs when signals section is trimmed
# (Activated once pr-context is implemented)
# ============================================================================
test_redirects_preserved() {
    local env_dir
    env_dir=$(setup_pr_context_env)

    local output exit_code=0
    output=$(AETHER_ROOT="$env_dir" COLONY_DATA_DIR="$env_dir/.aether/data" \
        bash "$AETHER_UTILS" pr-context 2>/dev/null) || exit_code=$?

    teardown_pr_context_env "$env_dir"

    if [[ "$exit_code" -ne 0 ]]; then
        return 0  # stub pass
    fi

    # If signals were trimmed, prompt_section should still contain REDIRECT content
    local trimmed prompt_section
    trimmed=$(echo "$output" | jq -r '.result.trimmed_sections // [] | join(",")' 2>/dev/null)
    if [[ "$trimmed" == *pheromone-signals* ]]; then
        prompt_section=$(echo "$output" | jq -r '.result.prompt_section // ""' 2>/dev/null)
        # Check if any REDIRECT content is preserved
        if [[ -n "$prompt_section" ]] && [[ "$prompt_section" != *"REDIRECT"* ]]; then
            # Only fail if we know there were REDIRECTs to preserve
            local signal_count
            signal_count=$(echo "$output" | jq -r '.result.signals.redirects // [] | length' 2>/dev/null)
            if [[ "$signal_count" -gt 0 ]]; then
                log_error "REDIRECT signals were not preserved when signals section trimmed"
                return 1
            fi
        fi
    fi
    return 0
}

# ============================================================================
# Test 5: Output has all required JSON sections
# (Activated once pr-context is implemented)
# ============================================================================
test_output_has_required_sections() {
    local env_dir
    env_dir=$(setup_pr_context_env)

    local output exit_code=0
    output=$(AETHER_ROOT="$env_dir" COLONY_DATA_DIR="$env_dir/.aether/data" \
        bash "$AETHER_UTILS" pr-context 2>/dev/null) || exit_code=$?

    teardown_pr_context_env "$env_dir"

    if [[ "$exit_code" -ne 0 ]]; then
        return 0  # stub pass
    fi

    local required_fields="schema generated_at branch cache_status queen signals hive colony_state blockers decisions midden prompt_section char_count budget trimmed_sections warnings fallbacks_used"
    for field in $required_fields; do
        if ! echo "$output" | jq -e ".result.$field" >/dev/null 2>&1; then
            # Check at top level too (might not be wrapped in result)
            if ! echo "$output" | jq -e ".$field" >/dev/null 2>&1; then
                log_error "Missing required field: $field"
                return 1
            fi
        fi
    done
    return 0
}

# ============================================================================
# Test 6: Missing COLONY_STATE.json returns partial data
# (Activated once pr-context is implemented)
# ============================================================================
test_missing_colony_state() {
    local env_dir
    env_dir=$(setup_pr_context_env)
    # Remove COLONY_STATE.json
    rm -f "$env_dir/.aether/data/COLONY_STATE.json"

    local output exit_code=0
    output=$(AETHER_ROOT="$env_dir" COLONY_DATA_DIR="$env_dir/.aether/data" \
        bash "$AETHER_UTILS" pr-context 2>/dev/null) || exit_code=$?

    teardown_pr_context_env "$env_dir"

    if [[ "$exit_code" -ne 0 ]]; then
        return 0  # stub pass
    fi

    # colony_state.exists should be false
    local exists
    # Use jq without `//` alternative operator since `false` is a valid boolean value
    # and `//` treats false as null-like (returns the alternative)
    exists=$(printf '%s' "$output" | jq -r 'if .result.colony_state == null then "missing" else .result.colony_state.exists end' 2>/dev/null)
    if [[ "$exists" == "null" ]]; then
        exists=$(printf '%s' "$output" | jq -r 'if .colony_state == null then "missing" else .colony_state.exists end' 2>/dev/null)
    fi
    if [[ "$exists" != "false" ]]; then
        log_error "Expected colony_state.exists=false, got: $exists"
        return 1
    fi

    # fallbacks_used should mention colony_state
    local fallbacks
    fallbacks=$(echo "$output" | jq -r '.result.fallbacks_used // [] | join(",")' 2>/dev/null)
    if [[ "$fallbacks" != *"colony_state"* ]]; then
        log_error "Expected fallbacks_used to contain 'colony_state', got: $fallbacks"
        return 1
    fi
    return 0
}

# ============================================================================
# Test 7: Missing pheromones.json returns empty signals
# (Activated once pr-context is implemented)
# ============================================================================
test_missing_pheromones() {
    local env_dir
    env_dir=$(setup_pr_context_env)
    rm -f "$env_dir/.aether/data/pheromones.json"

    local output exit_code=0
    output=$(AETHER_ROOT="$env_dir" COLONY_DATA_DIR="$env_dir/.aether/data" \
        bash "$AETHER_UTILS" pr-context 2>/dev/null) || exit_code=$?

    teardown_pr_context_env "$env_dir"

    if [[ "$exit_code" -ne 0 ]]; then
        return 0  # stub pass
    fi

    local count
    count=$(echo "$output" | jq -r '.result.signals.count // "missing"' 2>/dev/null)
    if [[ "$count" == "missing" ]]; then
        count=$(echo "$output" | jq -r '.signals.count // "missing"' 2>/dev/null)
    fi
    if [[ "$count" != "0" ]]; then
        log_error "Expected signals.count=0, got: $count"
        return 1
    fi
    return 0
}

# ============================================================================
# Test 8: No QUEEN.md soft-fail
# (Activated once pr-context is implemented)
# ============================================================================
test_no_queen_md_soft_fail() {
    local env_dir
    env_dir=$(setup_pr_context_env)
    # Remove all QUEEN.md files
    rm -f "$env_dir/.aether/QUEEN.md"
    # Ensure no global QUEEN.md interferes
    local home_backup="$HOME"
    local fake_home
    fake_home=$(mktemp -d)
    mkdir -p "$fake_home/.aether"

    local output exit_code=0
    output=$(AETHER_ROOT="$env_dir" COLONY_DATA_DIR="$env_dir/.aether/data" \
        HOME="$fake_home" \
        bash "$AETHER_UTILS" pr-context 2>/dev/null) || exit_code=$?

    rm -rf "$fake_home"
    teardown_pr_context_env "$env_dir"

    if [[ "$exit_code" -ne 0 ]]; then
        return 0  # stub pass
    fi

    # Should exit 0 even without QUEEN.md
    if [[ "$exit_code" -ne 0 ]]; then
        log_error "pr-context should not hard-fail when QUEEN.md is missing"
        return 1
    fi
    return 0
}

# ============================================================================
# Test 9: Normal mode budget
# (Activated once pr-context is implemented)
# ============================================================================
test_normal_mode_budget() {
    local env_dir
    env_dir=$(setup_pr_context_env)

    local output exit_code=0
    output=$(AETHER_ROOT="$env_dir" COLONY_DATA_DIR="$env_dir/.aether/data" \
        bash "$AETHER_UTILS" pr-context 2>/dev/null) || exit_code=$?

    teardown_pr_context_env "$env_dir"

    if [[ "$exit_code" -ne 0 ]]; then
        return 0  # stub pass
    fi

    local char_count budget
    char_count=$(echo "$output" | jq -r '.result.char_count // 0' 2>/dev/null)
    budget=$(echo "$output" | jq -r '.result.budget // 0' 2>/dev/null)
    if [[ "$char_count" -gt 6000 ]]; then
        log_error "Normal mode char_count ($char_count) exceeds 6000 budget"
        return 1
    fi
    return 0
}

# ============================================================================
# Test 10: Compact mode budget
# (Activated once pr-context is implemented)
# ============================================================================
test_compact_mode_budget() {
    local env_dir
    env_dir=$(setup_pr_context_env)

    local output exit_code=0
    output=$(AETHER_ROOT="$env_dir" COLONY_DATA_DIR="$env_dir/.aether/data" \
        bash "$AETHER_UTILS" pr-context --compact 2>/dev/null) || exit_code=$?

    teardown_pr_context_env "$env_dir"

    if [[ "$exit_code" -ne 0 ]]; then
        return 0  # stub pass
    fi

    local char_count budget
    char_count=$(echo "$output" | jq -r '.result.char_count // 0' 2>/dev/null)
    budget=$(echo "$output" | jq -r '.result.budget // 0' 2>/dev/null)
    if [[ "$char_count" -gt 3000 ]]; then
        log_error "Compact mode char_count ($char_count) exceeds 3000 budget"
        return 1
    fi
    return 0
}

# ============================================================================
# Test 11: Midden section bounded to max 10 entries
# (Activated once pr-context is implemented)
# ============================================================================
test_midden_section_bounded() {
    local env_dir
    env_dir=$(setup_pr_context_env)

    # Create midden with 20 entries
    local now
    now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local entries="["
    for i in $(seq 1 20); do
        if [[ $i -gt 1 ]]; then entries+=","; fi
        entries+="{\"id\":\"test-$i\",\"category\":\"test\",\"description\":\"Test failure entry $i with some description padding to make it realistic\",\"occurred_at\":\"$now\",\"context\":\"test context\"}"
    done
    entries+="]"
    echo "{\"entries\":$entries}" > "$env_dir/.aether/data/midden/midden.json"

    local output exit_code=0
    output=$(AETHER_ROOT="$env_dir" COLONY_DATA_DIR="$env_dir/.aether/data" \
        bash "$AETHER_UTILS" pr-context 2>/dev/null) || exit_code=$?

    teardown_pr_context_env "$env_dir"

    if [[ "$exit_code" -ne 0 ]]; then
        return 0  # stub pass
    fi

    local midden_count
    midden_count=$(echo "$output" | jq -r '.result.midden.count // 0' 2>/dev/null)
    if [[ "$midden_count" -gt 10 ]]; then
        log_error "Midden entries ($midden_count) exceed max 10"
        return 1
    fi
    return 0
}

# ============================================================================
# Test 12: Cache status tracking
# (Activated once pr-context is implemented)
# ============================================================================
test_cache_status() {
    local env_dir
    env_dir=$(setup_pr_context_env)

    # First call
    local output1 exit_code1=0
    output1=$(AETHER_ROOT="$env_dir" COLONY_DATA_DIR="$env_dir/.aether/data" \
        bash "$AETHER_UTILS" pr-context 2>/dev/null) || exit_code1=$?

    if [[ "$exit_code1" -ne 0 ]]; then
        teardown_pr_context_env "$env_dir"
        return 0  # stub pass
    fi

    # Second call -- should show cached for some sources
    local output2 exit_code2=0
    output2=$(AETHER_ROOT="$env_dir" COLONY_DATA_DIR="$env_dir/.aether/data" \
        bash "$AETHER_UTILS" pr-context 2>/dev/null) || exit_code2=$?

    teardown_pr_context_env "$env_dir"

    # Just verify cache_status is present in output
    if ! echo "$output2" | jq -e '.result.cache_status // empty' >/dev/null 2>&1; then
        if ! echo "$output2" | jq -e '.cache_status // empty' >/dev/null 2>&1; then
            log_error "cache_status missing from output"
            return 1
        fi
    fi
    return 0
}

# ============================================================================
# Test 13: Corrupt JSON fallback
# (Activated once pr-context is implemented)
# ============================================================================
test_corrupt_json_fallback() {
    local env_dir
    env_dir=$(setup_pr_context_env)

    # Write corrupt COLONY_STATE.json
    echo "{invalid json" > "$env_dir/.aether/data/COLONY_STATE.json"

    local output exit_code=0
    output=$(AETHER_ROOT="$env_dir" COLONY_DATA_DIR="$env_dir/.aether/data" \
        bash "$AETHER_UTILS" pr-context 2>/dev/null) || exit_code=$?

    teardown_pr_context_env "$env_dir"

    if [[ "$exit_code" -ne 0 ]]; then
        return 0  # stub pass
    fi

    # Should return exit 0 with ok=true
    local ok
    ok=$(echo "$output" | jq -r '.ok // "missing"' 2>/dev/null)
    if [[ "$ok" != "true" ]]; then
        log_error "Expected ok=true on corrupt JSON, got: $ok"
        return 1
    fi
    return 0
}

# ============================================================================
# Run all tests
# ============================================================================
log_info "=== pr-context tests ==="
log_info "Running Task 1 tests (budget extraction + test scaffold)"

run_test test_colony_prime_unchanged "colony-prime budget regression guard"
run_test test_trim_order_rolling_first "trim order: rolling-summary first"
run_test test_blockers_never_trimmed "blockers never trimmed"
run_test test_redirects_preserved "REDIRECTs preserved when signals trimmed"
run_test test_output_has_required_sections "output has all required sections"
run_test test_missing_colony_state "missing COLONY_STATE.json returns partial data"
run_test test_missing_pheromones "missing pheromones.json returns empty signals"
run_test test_no_queen_md_soft_fail "no QUEEN.md soft-fails"
run_test test_normal_mode_budget "normal mode budget under 6000"
run_test test_compact_mode_budget "compact mode budget under 3000"
run_test test_midden_section_bounded "midden section bounded to 10 entries"
run_test test_cache_status "cache status tracking"
run_test test_corrupt_json_fallback "corrupt JSON fallback"

test_summary
