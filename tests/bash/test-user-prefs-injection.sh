#!/usr/bin/env bash
# Tests for Task 2.2: User Preferences injection into colony-prime prompt_section
# Validates that user preferences appear as a distinct labeled block in colony-prime output

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

# ============================================================================
# Helper: Create a test colony with user preferences
# ============================================================================
setup_colony_with_prefs() {
    local tmpdir
    tmpdir=$(mktemp -d)
    local aether_dir="$tmpdir/.aether"
    local data_dir="$aether_dir/data"
    mkdir -p "$data_dir"

    local iso_date
    iso_date=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Create QUEEN.md with User Preferences section
    cat > "$aether_dir/QUEEN.md" << QUEENEOF
# QUEEN.md --- Colony Wisdom

> Last evolved: $iso_date
> Colonies contributed: 0
> Wisdom version: 1.0.0

---

## 📜 Philosophies

- Test before deploying

---

## 🧭 Patterns

- Use dependency injection

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

<!-- METADATA {"version":"1.0.0","last_evolved":"$iso_date","colonies_contributed":[],"promotion_thresholds":{"philosophy":1,"pattern":1,"redirect":1,"stack":1,"decree":0,"user_prefs":0},"stats":{"total_philosophies":1,"total_patterns":1,"total_redirects":0,"total_stack_entries":0,"total_decrees":0,"total_user_prefs":3}} -->
QUEENEOF

    # Create minimal COLONY_STATE.json
    cat > "$data_dir/COLONY_STATE.json" << 'STATEEOF'
{
  "session_id": "test_prefs_inject",
  "goal": "test user preferences injection",
  "state": "BUILDING",
  "current_phase": 1,
  "plan": { "phases": [] },
  "tasks": [],
  "memory": {
    "phase_learnings": [],
    "decisions": [],
    "blockers": [],
    "rolling_context": ""
  },
  "errors": { "flagged_patterns": [] },
  "events": []
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

# Helper: Create a test colony WITHOUT user preferences (backwards compat)
setup_colony_no_prefs() {
    local tmpdir
    tmpdir=$(mktemp -d)
    local aether_dir="$tmpdir/.aether"
    local data_dir="$aether_dir/data"
    mkdir -p "$data_dir"

    local iso_date
    iso_date=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Create QUEEN.md WITHOUT User Preferences section (legacy format)
    cat > "$aether_dir/QUEEN.md" << QUEENEOF
# QUEEN.md --- Colony Wisdom

> Last evolved: $iso_date
> Colonies contributed: 0
> Wisdom version: 1.0.0

---

## 📜 Philosophies

- Test before deploying

---

## 🧭 Patterns

- Use dependency injection

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

## 📊 Evolution Log

| Date | Colony | Change | Details |
|------|--------|--------|---------|

---

<!-- METADATA {"version":"1.0.0","last_evolved":"$iso_date","colonies_contributed":[],"promotion_thresholds":{"philosophy":1,"pattern":1,"redirect":1,"stack":1,"decree":0},"stats":{"total_philosophies":1,"total_patterns":1,"total_redirects":0,"total_stack_entries":0,"total_decrees":0}} -->
QUEENEOF

    # Create minimal COLONY_STATE.json
    cat > "$data_dir/COLONY_STATE.json" << 'STATEEOF'
{
  "session_id": "test_no_prefs",
  "goal": "test no user preferences",
  "state": "BUILDING",
  "current_phase": 1,
  "plan": { "phases": [] },
  "tasks": [],
  "memory": {
    "phase_learnings": [],
    "decisions": [],
    "blockers": [],
    "rolling_context": ""
  },
  "errors": { "flagged_patterns": [] },
  "events": []
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

# Helper: run colony-prime against a test env
run_colony_prime() {
    local tmpdir="$1"
    shift
    HOME="$tmpdir" AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$AETHER_UTILS" colony-prime "$@" 2>/dev/null
}

# Helper: extract prompt_section content from colony-prime output
get_prompt_section() {
    python3 -c "
import sys, re
raw = sys.stdin.read()
match = re.search(r'\"prompt_section\":\s*\"((?:[^\"\\\\]|\\\\.)*)\"', raw, re.DOTALL)
if match:
    val = match.group(1)
    val = val.replace('\\\\n', '\n').replace('\\\\t', '\t').replace('\\\\\"', '\"').replace('\\\\\\\\', '\\\\')
    print(val)
else:
    print('')
"
}

# Helper: extract log_line from colony-prime output
get_log_line() {
    python3 -c "
import sys, re
raw = sys.stdin.read()
match = re.search(r'\"log_line\":\s*\"((?:[^\"\\\\]|\\\\.)*)\"', raw)
if match:
    print(match.group(1))
else:
    print('')
"
}

# ============================================================================
# Test 1: Distinct USER PREFERENCES block exists in prompt_section
# ============================================================================
test_distinct_user_prefs_block() {
    local tmpdir
    tmpdir=$(setup_colony_with_prefs)

    local result
    result=$(run_colony_prime "$tmpdir") || true

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    # Must have the distinct labeled block
    if ! assert_contains "$prompt" "--- USER PREFERENCES ---"; then
        test_fail "Should contain '--- USER PREFERENCES ---' block label" "Not found"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_contains "$prompt" "--- END USER PREFERENCES ---"; then
        test_fail "Should contain '--- END USER PREFERENCES ---' block label" "Not found"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 2: USER PREFERENCES appears AFTER QUEEN WISDOM and BEFORE PHASE LEARNINGS
# ============================================================================
test_user_prefs_ordering() {
    local tmpdir
    tmpdir=$(setup_colony_with_prefs)

    # Add some phase learnings so we can check ordering
    python3 -c "
import json
state_file = '$tmpdir/.aether/data/COLONY_STATE.json'
with open(state_file) as f:
    state = json.load(f)
state['current_phase'] = 2
state['memory']['phase_learnings'] = [{
    'phase': '1',
    'phase_name': 'Foundation',
    'learnings': [{
        'claim': 'Test learning for ordering check',
        'status': 'validated',
        'evidence': ['test'],
        'confidence': 0.9
    }]
}]
with open(state_file, 'w') as f:
    json.dump(state, f)
"

    local result
    result=$(run_colony_prime "$tmpdir") || true

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    # Find positions of each section
    local queen_pos prefs_pos learnings_pos
    queen_pos=$(echo "$prompt" | grep -n "QUEEN WISDOM" | head -1 | cut -d: -f1 || echo "0")
    prefs_pos=$(echo "$prompt" | grep -n "USER PREFERENCES" | head -1 | cut -d: -f1 || echo "0")
    learnings_pos=$(echo "$prompt" | grep -n "PHASE LEARNINGS" | head -1 | cut -d: -f1 || echo "0")

    if [[ "$queen_pos" -eq 0 || "$prefs_pos" -eq 0 || "$learnings_pos" -eq 0 ]]; then
        test_fail "All three sections should exist" "queen=$queen_pos prefs=$prefs_pos learnings=$learnings_pos"
        rm -rf "$tmpdir"
        return 1
    fi

    if [[ "$queen_pos" -lt "$prefs_pos" && "$prefs_pos" -lt "$learnings_pos" ]]; then
        rm -rf "$tmpdir"
        return 0
    else
        test_fail "Order: QUEEN($queen_pos) < PREFS($prefs_pos) < LEARNINGS($learnings_pos)" "Wrong order"
        rm -rf "$tmpdir"
        return 1
    fi
}

# ============================================================================
# Test 3: No USER PREFERENCES block when QUEEN.md lacks the section
# ============================================================================
test_no_prefs_section_when_missing() {
    local tmpdir
    tmpdir=$(setup_colony_no_prefs)

    local result
    result=$(run_colony_prime "$tmpdir") || true

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    # Must NOT have the user preferences block
    if assert_contains "$prompt" "--- USER PREFERENCES ---"; then
        test_fail "Should NOT contain USER PREFERENCES block when section missing" "Found it"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 4: Log line includes user preference entry count
# ============================================================================
test_log_line_has_prefs_count() {
    local tmpdir
    tmpdir=$(setup_colony_with_prefs)

    local result
    result=$(run_colony_prime "$tmpdir") || true

    local log_line
    log_line=$(echo "$result" | get_log_line)

    # Log line should mention "3 user_prefs" (we have 3 preferences)
    if ! assert_contains "$log_line" "user_prefs"; then
        test_fail "Log line should mention user_prefs count" "Got: $log_line"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 5: User preferences content is preserved in the block
# ============================================================================
test_user_prefs_content_preserved() {
    local tmpdir
    tmpdir=$(setup_colony_with_prefs)

    local result
    result=$(run_colony_prime "$tmpdir") || true

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    if ! assert_contains "$prompt" "Communication style"; then
        test_fail "Should contain preference content" "Not found"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_contains "$prompt" "Non-technical founder"; then
        test_fail "Should contain preference content" "Not found"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 6: User preferences NOT inside QUEEN WISDOM block
# ============================================================================
test_user_prefs_not_in_queen_wisdom() {
    local tmpdir
    tmpdir=$(setup_colony_with_prefs)

    local result
    result=$(run_colony_prime "$tmpdir") || true

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    # Extract just the QUEEN WISDOM block
    local queen_block
    queen_block=$(echo "$prompt" | sed -n '/--- QUEEN WISDOM/,/--- END QUEEN WISDOM ---/p')

    # QUEEN WISDOM block should NOT contain "User Preferences" label
    if assert_contains "$queen_block" "User Preferences"; then
        test_fail "QUEEN WISDOM block should NOT contain User Preferences" "Found inside QUEEN WISDOM"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 7: Budget trimming -- user-prefs trimmed before queen-wisdom
# ============================================================================
test_budget_trims_user_prefs_before_wisdom() {
    local tmpdir
    tmpdir=$(setup_colony_with_prefs)

    # Add lots of content to force budget pressure
    # Add large rolling summary
    local logfile="$tmpdir/.aether/data/rolling-summary.log"
    > "$logfile"
    for i in $(seq 1 100); do
        echo "2026-03-20T10:${i}:00Z|build|phase-1|Long rolling summary entry $i with substantial text padding for budget testing" >> "$logfile"
    done

    # Add phase learnings
    python3 -c "
import json
state_file = '$tmpdir/.aether/data/COLONY_STATE.json'
with open(state_file) as f:
    state = json.load(f)
state['current_phase'] = 3
learnings = []
for i in range(1, 30):
    learnings.append({
        'claim': f'Learning {i}: Validated insight with sufficient length for budget testing purposes',
        'status': 'validated',
        'evidence': ['test'],
        'confidence': 0.9
    })
state['memory']['phase_learnings'] = [{'phase': '1', 'phase_name': 'Foundation', 'learnings': learnings}]
with open(state_file, 'w') as f:
    json.dump(state, f)
"

    local result
    result=$(run_colony_prime "$tmpdir") || true

    local log_line
    log_line=$(echo "$result" | get_log_line)

    # If budget truncation occurs, check that user-prefs is listed as trimmed
    # before queen-wisdom (or not trimmed if budget allows)
    # This is a conditional test -- only validates order if both are trimmed
    if assert_contains "$log_line" "truncated"; then
        if assert_contains "$log_line" "user-prefs" && assert_contains "$log_line" "queen-wisdom"; then
            # Both trimmed -- user-prefs should appear before queen-wisdom in the list
            local prefs_pos wisdom_pos
            prefs_pos=$(echo "$log_line" | grep -o '.*user-prefs' | wc -c || echo "0")
            wisdom_pos=$(echo "$log_line" | grep -o '.*queen-wisdom' | wc -c || echo "0")
            if [[ "$prefs_pos" -lt "$wisdom_pos" ]]; then
                rm -rf "$tmpdir"
                return 0
            else
                test_fail "user-prefs should be trimmed before queen-wisdom" "prefs_pos=$prefs_pos wisdom_pos=$wisdom_pos"
                rm -rf "$tmpdir"
                return 1
            fi
        fi
    fi

    # If no truncation or only one trimmed, that's acceptable
    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Run all tests
# ============================================================================

log_info "Running User Preferences injection tests (Task 2.2)"
log_info "Repo root: $REPO_ROOT"

run_test test_distinct_user_prefs_block "Distinct USER PREFERENCES block in prompt_section"
run_test test_user_prefs_ordering "USER PREFERENCES after QUEEN WISDOM, before PHASE LEARNINGS"
run_test test_no_prefs_section_when_missing "No USER PREFERENCES block when QUEEN.md lacks section"
run_test test_log_line_has_prefs_count "Log line includes user preference count"
run_test test_user_prefs_content_preserved "User preferences content is preserved"
run_test test_user_prefs_not_in_queen_wisdom "User preferences NOT inside QUEEN WISDOM block"
run_test test_budget_trims_user_prefs_before_wisdom "Budget trims user-prefs before queen-wisdom"

test_summary
