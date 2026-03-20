#!/usr/bin/env bash
# Tests for hive-wisdom injection into colony-prime
# Task 3.1: colony-prime loads high_value_signals from ~/.aether/eternal/memory.json

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

# ============================================================================
# Helper: Create a test colony environment (same pattern as budget tests)
# ============================================================================
setup_colony_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    local aether_dir="$tmpdir/.aether"
    local data_dir="$aether_dir/data"
    mkdir -p "$data_dir"

    local iso_date
    iso_date=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Create QUEEN.md
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

## 📊 Evolution Log

| Date | Colony | Change | Details |
|------|--------|--------|---------|

---

<!-- METADATA {"version":"1.0.0","last_evolved":"$iso_date","colonies_contributed":[],"promotion_thresholds":{"philosophy":1,"pattern":1,"redirect":1,"stack":1,"decree":0},"stats":{"total_philosophies":0,"total_patterns":0,"total_redirects":0,"total_stack_entries":0,"total_decrees":0}} -->
QUEENEOF

    # Create COLONY_STATE.json
    cat > "$data_dir/COLONY_STATE.json" << 'STATEEOF'
{
  "session_id": "test_hive",
  "goal": "test hive wisdom",
  "state": "BUILDING",
  "current_phase": 1,
  "plan": { "phases": [] },
  "memory": {
    "instincts": [],
    "phase_learnings": [],
    "decisions": []
  },
  "errors": { "flagged_patterns": [] },
  "events": []
}
STATEEOF

    # Create pheromones.json (empty)
    cat > "$data_dir/pheromones.json" << 'PHEREOF'
{
  "signals": [],
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

# Helper: create eternal memory with high_value_signals
create_eternal_memory() {
    local tmpdir="$1"
    local signal_count="${2:-3}"
    local eternal_dir="$tmpdir/.aether/eternal"
    mkdir -p "$eternal_dir"

    python3 -c "
import json
signals = []
for i in range(1, $signal_count + 1):
    signals.append({
        'content': f'Always validate user input before database operations (signal {i})',
        'type': 'REDIRECT',
        'strength': round(0.5 + (i * 0.1), 1),
        'source_colony': f'colony-{i}',
        'promoted_at': '2026-03-01T00:00:00Z'
    })
with open('$eternal_dir/memory.json', 'w') as f:
    json.dump({
        'version': '1.0',
        'created_at': '2026-02-17T22:25:34Z',
        'colonies': [],
        'high_value_signals': signals,
        'cross_session_patterns': []
    }, f, indent=2)
"
}

# Helper: create eternal memory with mixed signal types
create_mixed_eternal_memory() {
    local tmpdir="$1"
    local eternal_dir="$tmpdir/.aether/eternal"
    mkdir -p "$eternal_dir"

    cat > "$eternal_dir/memory.json" << 'MEMEOF'
{
  "version": "1.0",
  "created_at": "2026-02-17T22:25:34Z",
  "colonies": [],
  "high_value_signals": [
    {
      "content": "Use structured logging with correlation IDs",
      "type": "PATTERN",
      "strength": 0.9,
      "source_colony": "colony-alpha",
      "promoted_at": "2026-03-01T00:00:00Z"
    },
    {
      "content": "Never store secrets in environment variables without encryption",
      "type": "REDIRECT",
      "strength": 0.8,
      "source_colony": "colony-beta",
      "promoted_at": "2026-03-05T00:00:00Z"
    },
    {
      "content": "Prefer composition over inheritance for flexible design",
      "type": "PHILOSOPHY",
      "strength": 0.7,
      "source_colony": "colony-gamma",
      "promoted_at": "2026-03-10T00:00:00Z"
    }
  ],
  "cross_session_patterns": []
}
MEMEOF
}

# ============================================================================
# Tests
# ============================================================================

test_hive_section_present_with_signals() {
    # When eternal memory has high_value_signals, colony-prime should include HIVE WISDOM section
    local tmpdir
    tmpdir=$(setup_colony_env)
    create_eternal_memory "$tmpdir" 3

    local result
    result=$(run_colony_prime "$tmpdir")

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true in output" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    if ! assert_contains "$prompt" "HIVE WISDOM"; then
        test_fail "Expected HIVE WISDOM section in prompt" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_contains "$prompt" "Cross-Colony Patterns"; then
        test_fail "Expected 'Cross-Colony Patterns' label" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_absent_when_no_eternal_memory() {
    # When ~/.aether/eternal/memory.json does not exist, no HIVE WISDOM section
    local tmpdir
    tmpdir=$(setup_colony_env)
    # Do NOT create eternal memory

    local result
    result=$(run_colony_prime "$tmpdir")

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    if assert_contains "$prompt" "HIVE WISDOM"; then
        test_fail "Should NOT have HIVE WISDOM when no eternal memory" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_absent_when_empty_signals() {
    # When high_value_signals is empty array, no HIVE WISDOM section
    local tmpdir
    tmpdir=$(setup_colony_env)
    create_eternal_memory "$tmpdir" 0

    local result
    result=$(run_colony_prime "$tmpdir")

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    if assert_contains "$prompt" "HIVE WISDOM"; then
        test_fail "Should NOT have HIVE WISDOM when signals array is empty" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_shows_content_type_strength() {
    # Each hive entry should show content, type, and strength
    local tmpdir
    tmpdir=$(setup_colony_env)
    create_mixed_eternal_memory "$tmpdir"

    local result
    result=$(run_colony_prime "$tmpdir")

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    # Check content appears
    if ! assert_contains "$prompt" "structured logging"; then
        test_fail "Expected signal content in prompt" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    # Check type appears
    if ! assert_contains "$prompt" "PATTERN"; then
        test_fail "Expected signal type in prompt" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    # Check strength appears (0.9)
    if ! assert_contains "$prompt" "0.9"; then
        test_fail "Expected signal strength in prompt" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_capped_at_5_normal_mode() {
    # Normal mode: max 5 entries
    local tmpdir
    tmpdir=$(setup_colony_env)
    create_eternal_memory "$tmpdir" 8

    local result
    result=$(run_colony_prime "$tmpdir")

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    # Signal 5 should be present
    if ! assert_contains "$prompt" "signal 5"; then
        test_fail "Signal 5 should be present" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    # Signal 6 should NOT be present (capped at 5)
    if assert_contains "$prompt" "signal 6"; then
        test_fail "Signal 6 should NOT be present (cap at 5)" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_capped_at_3_compact_mode() {
    # Compact mode: max 3 entries
    local tmpdir
    tmpdir=$(setup_colony_env)
    create_eternal_memory "$tmpdir" 8

    local result
    result=$(run_colony_prime "$tmpdir" --compact)

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    # Signal 3 should be present
    if ! assert_contains "$prompt" "signal 3"; then
        test_fail "Signal 3 should be present in compact" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    # Signal 4 should NOT be present (capped at 3)
    if assert_contains "$prompt" "signal 4"; then
        test_fail "Signal 4 should NOT be present in compact (cap at 3)" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_log_line_includes_count() {
    # log_line should include hive wisdom entry count
    local tmpdir
    tmpdir=$(setup_colony_env)
    create_eternal_memory "$tmpdir" 3

    local result
    result=$(run_colony_prime "$tmpdir")

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local log_line
    log_line=$(echo "$result" | get_log_line)

    if ! assert_contains "$log_line" "3 hive"; then
        test_fail "Expected '3 hive' in log_line" "$log_line"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_placement_after_user_prefs_before_learnings() {
    # HIVE WISDOM should appear AFTER USER PREFERENCES and BEFORE PHASE LEARNINGS
    local tmpdir
    tmpdir=$(setup_colony_env)
    create_eternal_memory "$tmpdir" 2

    # Add user prefs to QUEEN.md
    local iso_date
    iso_date=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    cat > "$tmpdir/.aether/QUEEN.md" << QUEENEOF
# QUEEN.md --- Colony Wisdom

> Last evolved: $iso_date

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

- Prefer dark mode
- Use concise commit messages

---

## 📊 Evolution Log

| Date | Colony | Change | Details |
|------|--------|--------|---------|

---

<!-- METADATA {"version":"1.0.0","last_evolved":"$iso_date","colonies_contributed":[],"promotion_thresholds":{"philosophy":1,"pattern":1,"redirect":1,"stack":1,"decree":0},"stats":{"total_philosophies":0,"total_patterns":0,"total_redirects":0,"total_stack_entries":0,"total_decrees":0}} -->
QUEENEOF

    # Add phase learnings
    python3 -c "
import json
state_file = '$tmpdir/.aether/data/COLONY_STATE.json'
with open(state_file) as f:
    state = json.load(f)
state['current_phase'] = 3
state['memory']['phase_learnings'] = [{
    'phase': '1',
    'phase_name': 'Foundation',
    'learnings': [{'claim': 'Test learning claim', 'status': 'validated', 'evidence': ['test'], 'confidence': 0.9}]
}]
with open(state_file, 'w') as f:
    json.dump(state, f)
"

    local result
    result=$(run_colony_prime "$tmpdir")

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    # All three sections should exist
    if ! assert_contains "$prompt" "USER PREFERENCES"; then
        test_fail "Expected USER PREFERENCES section" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi
    if ! assert_contains "$prompt" "HIVE WISDOM"; then
        test_fail "Expected HIVE WISDOM section" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi
    if ! assert_contains "$prompt" "PHASE LEARNINGS"; then
        test_fail "Expected PHASE LEARNINGS section" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    # Check ordering: USER PREFERENCES before HIVE WISDOM before PHASE LEARNINGS
    local prefs_pos hive_pos learnings_pos
    prefs_pos=$(echo "$prompt" | grep -n "USER PREFERENCES" | head -1 | cut -d: -f1)
    hive_pos=$(echo "$prompt" | grep -n "HIVE WISDOM" | head -1 | cut -d: -f1)
    learnings_pos=$(echo "$prompt" | grep -n "PHASE LEARNINGS" | head -1 | cut -d: -f1)

    if [[ "$prefs_pos" -ge "$hive_pos" ]]; then
        test_fail "USER PREFERENCES ($prefs_pos) should come before HIVE WISDOM ($hive_pos)" ""
        rm -rf "$tmpdir"
        return 1
    fi

    if [[ "$hive_pos" -ge "$learnings_pos" ]]; then
        test_fail "HIVE WISDOM ($hive_pos) should come before PHASE LEARNINGS ($learnings_pos)" ""
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_handles_malformed_json() {
    # Malformed eternal memory should be skipped gracefully
    local tmpdir
    tmpdir=$(setup_colony_env)

    local eternal_dir="$tmpdir/.aether/eternal"
    mkdir -p "$eternal_dir"
    echo "NOT VALID JSON {{{" > "$eternal_dir/memory.json"

    local result
    result=$(run_colony_prime "$tmpdir")

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true even with malformed eternal memory" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    if assert_contains "$prompt" "HIVE WISDOM"; then
        test_fail "Should NOT have HIVE WISDOM with malformed JSON" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Run tests
# ============================================================================

log_info "Running hive-wisdom injection tests"
log_info "Repo root: $REPO_ROOT"

run_test test_hive_section_present_with_signals "HIVE WISDOM section present when signals exist"
run_test test_hive_absent_when_no_eternal_memory "No HIVE WISDOM when eternal memory missing"
run_test test_hive_absent_when_empty_signals "No HIVE WISDOM when signals array empty"
run_test test_hive_shows_content_type_strength "Hive entries show content, type, and strength"
run_test test_hive_capped_at_5_normal_mode "Normal mode caps at 5 entries"
run_test test_hive_capped_at_3_compact_mode "Compact mode caps at 3 entries"
run_test test_hive_log_line_includes_count "Log line includes hive wisdom count"
run_test test_hive_placement_after_user_prefs_before_learnings "HIVE WISDOM placed after USER PREFS before PHASE LEARNINGS"
run_test test_hive_handles_malformed_json "Malformed JSON handled gracefully"

test_summary
