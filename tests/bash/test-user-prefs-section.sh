#!/usr/bin/env bash
# Tests for User Preferences section in QUEEN.md parsing
# Task 2.1: Add user preferences section to QUEEN.md template and both extract functions

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"
TEMPLATE_FILE="$REPO_ROOT/.aether/templates/QUEEN.md.template"

# ============================================================================
# Helper: Create a test colony with user preferences content
# ============================================================================
setup_colony_with_prefs() {
    local tmpdir
    tmpdir=$(mktemp -d)
    local aether_dir="$tmpdir/.aether"
    local data_dir="$aether_dir/data"
    mkdir -p "$data_dir"

    local iso_date
    iso_date=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Create QUEEN.md with all sections including User Preferences
    cat > "$aether_dir/QUEEN.md" << QUEENEOF
# QUEEN.md --- Colony Wisdom

> Last evolved: $iso_date
> Colonies contributed: 0
> Wisdom version: 1.0.0

---

## 📜 Philosophies

*No philosophies recorded yet*

---

## 🧭 Patterns

*No patterns recorded yet*

---

## ⚠️ Redirects

*No redirects recorded yet*

---

## 🔧 Stack Wisdom

*No stack wisdom recorded yet*

---

## 🏛️ Decrees

*No decrees recorded yet*

---

## 👤 User Preferences

- Communication style: Plain English, no jargon
- Expertise level: Non-technical founder
- Decision pattern: Prefers quick iteration

---

## 📊 Evolution Log

| Date | Colony | Change | Details |
|------|--------|--------|---------|

---

<!-- METADATA {"version":"1.0.0","last_evolved":"$iso_date","colonies_contributed":[],"promotion_thresholds":{"philosophy":1,"pattern":1,"redirect":1,"stack":1,"decree":0,"user_prefs":0},"stats":{"total_philosophies":0,"total_patterns":0,"total_redirects":0,"total_stack_entries":0,"total_decrees":0,"total_user_prefs":3}} -->
QUEENEOF

    # Create minimal COLONY_STATE.json
    cat > "$data_dir/COLONY_STATE.json" << 'STATEEOF'
{
  "session_id": "test_prefs",
  "goal": "test user preferences parsing",
  "state": "BUILDING",
  "current_phase": 1,
  "colony_name": "test-prefs",
  "tasks": [],
  "memory": {
    "phase_learnings": [],
    "decisions": [],
    "blockers": [],
    "rolling_context": ""
  }
}
STATEEOF

    # Create pheromones.json
    cat > "$data_dir/pheromones.json" << 'PHEREOF'
{
  "signals": [],
  "instincts": [],
  "version": "1.0.0"
}
PHEREOF

    echo "$tmpdir"
}

# Helper: run queen-read against a test env
run_queen_read() {
    local tmpdir="$1"
    shift
    HOME="$tmpdir" AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$AETHER_UTILS" queen-read "$@" 2>/dev/null
}

# Helper: run colony-prime against a test env
run_colony_prime() {
    local tmpdir="$1"
    shift
    HOME="$tmpdir" AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$AETHER_UTILS" colony-prime "$@" 2>/dev/null
}

# ============================================================================
# Test 1: Template contains User Preferences section
# ============================================================================
test_template_has_user_prefs_section() {
    test_start "Template contains User Preferences section"

    if grep -q '## 👤 User Preferences' "$TEMPLATE_FILE"; then
        test_pass
    else
        test_fail "Template should contain '## 👤 User Preferences'" "Not found"
    fi
}

# ============================================================================
# Test 2: Template has User Preferences between Decrees and Evolution Log
# ============================================================================
test_template_section_order() {
    test_start "Template has User Preferences between Decrees and Evolution Log"

    local dec_line=$(grep -n '## 🏛️ Decrees' "$TEMPLATE_FILE" | head -1 | cut -d: -f1)
    local prefs_line=$(grep -n '## 👤 User Preferences' "$TEMPLATE_FILE" | head -1 | cut -d: -f1)
    local evo_line=$(grep -n '## 📊 Evolution Log' "$TEMPLATE_FILE" | head -1 | cut -d: -f1)

    if [[ -n "$dec_line" && -n "$prefs_line" && -n "$evo_line" ]]; then
        if [[ "$dec_line" -lt "$prefs_line" && "$prefs_line" -lt "$evo_line" ]]; then
            test_pass
        else
            test_fail "Order: Decrees($dec_line) < UserPrefs($prefs_line) < EvolutionLog($evo_line)" "Wrong order"
        fi
    else
        test_fail "All three sections should exist" "dec=$dec_line prefs=$prefs_line evo=$evo_line"
    fi
}

# ============================================================================
# Test 3: Template METADATA includes total_user_prefs
# ============================================================================
test_template_metadata_has_user_prefs() {
    test_start "Template METADATA includes total_user_prefs"

    if grep -q 'total_user_prefs' "$TEMPLATE_FILE"; then
        test_pass
    else
        test_fail "METADATA should contain total_user_prefs" "Not found"
    fi
}

# ============================================================================
# Test 4: queen-read (_extract_wisdom_sections) parses user_prefs
# ============================================================================
test_queen_read_parses_user_prefs() {
    test_start "queen-read parses user_prefs from QUEEN.md"

    local tmpdir
    tmpdir=$(setup_colony_with_prefs)

    local output
    output=$(run_queen_read "$tmpdir") || true

    # Check that output contains user_prefs in wisdom
    local has_user_prefs
    has_user_prefs=$(echo "$output" | jq -r '.result.wisdom.user_prefs // "MISSING"' 2>/dev/null || echo "MISSING")

    if [[ "$has_user_prefs" != "MISSING" && "$has_user_prefs" != "" && "$has_user_prefs" != "null" ]]; then
        # Verify it contains our test content
        if echo "$has_user_prefs" | grep -q "Communication style"; then
            test_pass
        else
            test_fail "user_prefs should contain 'Communication style'" "Got: $has_user_prefs"
        fi
    else
        test_fail "wisdom.user_prefs should exist in output" "Got: $has_user_prefs"
    fi

    rm -rf "$tmpdir"
}

# ============================================================================
# Test 5: colony-prime (_extract_wisdom) parses user_prefs
# ============================================================================
test_colony_prime_parses_user_prefs() {
    test_start "colony-prime parses user_prefs from QUEEN.md"

    local tmpdir
    tmpdir=$(setup_colony_with_prefs)

    local output
    output=$(run_colony_prime "$tmpdir") || true

    # colony-prime returns prompt_section which should contain user prefs as a distinct labeled block
    local prompt_section
    prompt_section=$(echo "$output" | jq -r '.result.prompt_section // ""' 2>/dev/null || echo "")

    # Check for the distinct USER PREFERENCES block label (Task 2.2 format)
    if echo "$prompt_section" | grep -q "USER PREFERENCES"; then
        test_pass
    else
        test_fail "prompt_section should contain '--- USER PREFERENCES ---' block" "Not found in prompt_section"
    fi

    rm -rf "$tmpdir"
}

# ============================================================================
# Test 6: queen-read has_user_prefs priming flag
# ============================================================================
test_queen_read_has_priming_flag() {
    test_start "queen-read has has_user_prefs priming flag"

    local tmpdir
    tmpdir=$(setup_colony_with_prefs)

    local output
    output=$(run_queen_read "$tmpdir") || true

    local has_flag
    has_flag=$(echo "$output" | jq -r '.result.priming.has_user_prefs // "MISSING"' 2>/dev/null || echo "MISSING")

    if [[ "$has_flag" == "true" ]]; then
        test_pass
    else
        test_fail "priming.has_user_prefs should be true" "Got: $has_flag"
    fi

    rm -rf "$tmpdir"
}

# ============================================================================
# Run all tests
# ============================================================================
log_info "Running User Preferences section tests (Task 2.1)"

test_template_has_user_prefs_section
test_template_section_order
test_template_metadata_has_user_prefs
test_queen_read_parses_user_prefs
test_colony_prime_parses_user_prefs
test_queen_read_has_priming_flag

test_summary
exit $TESTS_FAILED
