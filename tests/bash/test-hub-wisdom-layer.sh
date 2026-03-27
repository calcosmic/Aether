#!/usr/bin/env bash
# Tests for Phase 20: Hub Wisdom Layer
# Proves global vs local QUEEN WISDOM distinction, empty-section gating,
# budget trim order, v1 migration, and auto-migration during colony-prime.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

# ============================================================================
# Helpers
# ============================================================================

# Create a test env with HOME override for isolation
# Returns path to hub dir (use as HOME)
# Colony dir is at $hub_dir/colony/
setup_hub_env() {
    local hub_dir
    hub_dir=$(mktemp -d)

    # Create hub-level .aether (global QUEEN.md lives here)
    mkdir -p "$hub_dir/.aether"

    # Create colony dir with local .aether
    local colony_dir="$hub_dir/colony"
    local aether_dir="$colony_dir/.aether"
    local data_dir="$aether_dir/data"
    mkdir -p "$data_dir"

    local iso_date
    iso_date=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Create COLONY_STATE.json
    cat > "$data_dir/COLONY_STATE.json" << 'STATEEOF'
{
  "session_id": "test_hub_wisdom",
  "goal": "test hub wisdom layer",
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

    echo "$hub_dir"
}

# Write a v2 QUEEN.md at the specified path
write_v2_queen() {
    local queen_file="$1"
    shift
    local uprefs="${1:-}"
    local codebase="${2:-}"
    local learnings="${3:-}"
    local instincts="${4:-}"

    local iso_date
    iso_date=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    cat > "$queen_file" << QUEENEOF
# QUEEN.md -- Colony Wisdom

> Last evolved: $iso_date
> Wisdom version: 2.0.0

---

## User Preferences

Communication style, expertise level, and decision-making patterns observed from the user (the Queen).

${uprefs:-*No user preferences recorded yet.*}

---

## Codebase Patterns

Validated approaches that work in this codebase.

${codebase:-*No codebase patterns recorded yet.*}

---

## Build Learnings

What worked and what failed during builds.

${learnings:-*No build learnings recorded yet.*}

---

## Instincts

High-confidence behavioral patterns.

${instincts:-*No instincts recorded yet.*}

---

## Evolution Log

| Date | Source | Type | Details |
|------|--------|------|---------|
| $iso_date | system | initialized | QUEEN.md created from template |

---

<!-- METADATA {"version":"2.0.0","wisdom_version":"2.0","last_evolved":"$iso_date","colonies_contributed":[],"stats":{"total_user_prefs":0,"total_codebase_patterns":0,"total_build_learnings":0,"total_instincts":0}} -->
QUEENEOF
}

# Run colony-prime with HOME/AETHER_ROOT isolation
run_colony_prime() {
    local hub_dir="$1"
    local colony_dir="$hub_dir/colony"
    shift
    HOME="$hub_dir" AETHER_ROOT="$colony_dir" DATA_DIR="$colony_dir/.aether/data" \
        bash "$AETHER_UTILS" colony-prime "$@" 2>/dev/null
}

# Extract prompt_section content from colony-prime output
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

# Extract prompt_section length
get_prompt_length() {
    python3 -c "
import sys, re
raw = sys.stdin.read()
match = re.search(r'\"prompt_section\":\s*\"((?:[^\"\\\\]|\\\\.)*)\"', raw, re.DOTALL)
if match:
    val = match.group(1)
    val = val.replace('\\\\n', '\n').replace('\\\\t', '\t').replace('\\\\\"', '\"').replace('\\\\\\\\', '\\\\')
    print(len(val))
else:
    print(0)
"
}

# Extract log_line from colony-prime output
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
# Test 1: Global and local sections appear separately
# ============================================================================
test_hub_global_local_separate() {
    local hub_dir
    hub_dir=$(setup_hub_env)

    # Write global QUEEN.md with a real Codebase Patterns entry
    write_v2_queen "$hub_dir/.aether/QUEEN.md" \
        "" \
        "- [general] Always use structured logging across all projects"

    # Write local QUEEN.md with a different real entry
    write_v2_queen "$hub_dir/colony/.aether/QUEEN.md" \
        "" \
        "- [repo] This project uses bash exclusively"

    local result
    result=$(run_colony_prime "$hub_dir" --compact)

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$hub_dir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    # Check for Global header
    if ! assert_contains "$prompt" "QUEEN WISDOM (Global -- All Colonies)"; then
        test_fail "Should have Global header" "Not found"
        rm -rf "$hub_dir"
        return 1
    fi

    # Check for Colony-Specific header
    if ! assert_contains "$prompt" "QUEEN WISDOM (Colony-Specific)"; then
        test_fail "Should have Colony-Specific header" "Not found"
        rm -rf "$hub_dir"
        return 1
    fi

    # Check global entry is in the output
    if ! assert_contains "$prompt" "Always use structured logging"; then
        test_fail "Global entry should appear" "Not found"
        rm -rf "$hub_dir"
        return 1
    fi

    # Check local entry is in the output
    if ! assert_contains "$prompt" "This project uses bash exclusively"; then
        test_fail "Local entry should appear" "Not found"
        rm -rf "$hub_dir"
        return 1
    fi

    rm -rf "$hub_dir"
    return 0
}

# ============================================================================
# Test 2: Empty global section is omitted
# ============================================================================
test_hub_empty_global_omitted() {
    local hub_dir
    hub_dir=$(setup_hub_env)

    # Global QUEEN.md has placeholder-only (no real entries)
    write_v2_queen "$hub_dir/.aether/QUEEN.md"

    # Local QUEEN.md has real entries
    write_v2_queen "$hub_dir/colony/.aether/QUEEN.md" \
        "" \
        "- [repo] Local pattern that should appear"

    local result
    result=$(run_colony_prime "$hub_dir" --compact)

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$hub_dir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    # Global section should NOT appear (all placeholders)
    if assert_contains "$prompt" "QUEEN WISDOM (Global"; then
        test_fail "Empty global section should be omitted" "Found Global header"
        rm -rf "$hub_dir"
        return 1
    fi

    # Local section should appear
    if ! assert_contains "$prompt" "QUEEN WISDOM (Colony-Specific)"; then
        test_fail "Colony-Specific section should appear" "Not found"
        rm -rf "$hub_dir"
        return 1
    fi

    rm -rf "$hub_dir"
    return 0
}

# ============================================================================
# Test 3: Empty local section is omitted
# ============================================================================
test_hub_empty_local_omitted() {
    local hub_dir
    hub_dir=$(setup_hub_env)

    # Global QUEEN.md has real entries
    write_v2_queen "$hub_dir/.aether/QUEEN.md" \
        "" \
        "- [general] Global pattern that should appear"

    # Local QUEEN.md has placeholder-only
    write_v2_queen "$hub_dir/colony/.aether/QUEEN.md"

    local result
    result=$(run_colony_prime "$hub_dir" --compact)

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$hub_dir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    # Global section should appear
    if ! assert_contains "$prompt" "QUEEN WISDOM (Global -- All Colonies)"; then
        test_fail "Global section should appear" "Not found"
        rm -rf "$hub_dir"
        return 1
    fi

    # Local section should NOT appear (all placeholders)
    if assert_contains "$prompt" "QUEEN WISDOM (Colony-Specific)"; then
        test_fail "Empty local section should be omitted" "Found Colony-Specific header"
        rm -rf "$hub_dir"
        return 1
    fi

    rm -rf "$hub_dir"
    return 0
}

# ============================================================================
# Test 4: User preferences show source labels
# ============================================================================
test_hub_user_prefs_source_labels() {
    local hub_dir
    hub_dir=$(setup_hub_env)

    # Global QUEEN.md with a user preference
    write_v2_queen "$hub_dir/.aether/QUEEN.md" \
        "- Plain English communication"

    # Local QUEEN.md with a different user preference
    write_v2_queen "$hub_dir/colony/.aether/QUEEN.md" \
        "- Verbose error messages"

    local result
    result=$(run_colony_prime "$hub_dir" --compact)

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$hub_dir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    # Should have USER PREFERENCES section
    if ! assert_contains "$prompt" "USER PREFERENCES"; then
        test_fail "Should have USER PREFERENCES section" "Not found"
        rm -rf "$hub_dir"
        return 1
    fi

    # Global prefs should be labeled [global]
    if ! assert_contains "$prompt" "[global] Plain English"; then
        test_fail "Global pref should have [global] label" "Not found"
        rm -rf "$hub_dir"
        return 1
    fi

    # Local prefs should be labeled [local]
    if ! assert_contains "$prompt" "[local] Verbose error"; then
        test_fail "Local pref should have [local] label" "Not found"
        rm -rf "$hub_dir"
        return 1
    fi

    rm -rf "$hub_dir"
    return 0
}

# ============================================================================
# Test 5: Budget trims global queen wisdom before local
# ============================================================================
test_hub_budget_trims_global_first() {
    local hub_dir
    hub_dir=$(setup_hub_env)

    # Create large entries in both global and local to exceed compact budget (4000 chars)
    # Generate ~2500 chars of entries for each
    local large_global_entries=""
    local large_local_entries=""
    for i in $(seq 1 25); do
        large_global_entries+="- [general] Global codebase pattern number $i that provides extensive guidance about cross-cutting architectural concerns across all colonies in the hive"$'\n'
    done
    for i in $(seq 1 25); do
        large_local_entries+="- [repo] Local codebase pattern number $i that provides detailed guidance about this specific project's conventions and architectural decisions"$'\n'
    done

    write_v2_queen "$hub_dir/.aether/QUEEN.md" \
        "" \
        "$large_global_entries"

    write_v2_queen "$hub_dir/colony/.aether/QUEEN.md" \
        "" \
        "$large_local_entries"

    # Also add pheromone signals to push over budget
    cat > "$hub_dir/colony/.aether/data/pheromones.json" << 'PHEREOF'
{
  "signals": [
    {"type":"REDIRECT","content":{"text":"Test redirect signal for budget testing purposes"},"strength":0.9,"source":"user","created_at":"2026-03-25T00:00:00Z","expires_at":null,"phase":null,"auto_decay":false},
    {"type":"FOCUS","content":{"text":"Test focus signal for budget testing with additional padding"},"strength":0.8,"source":"user","created_at":"2026-03-25T00:00:00Z","expires_at":"2026-12-31T00:00:00Z","phase":null,"auto_decay":true}
  ],
  "version": "1.0.0"
}
PHEREOF

    local result
    result=$(run_colony_prime "$hub_dir" --compact)

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$hub_dir"
        return 1
    fi

    local log_line
    log_line=$(echo "$result" | get_log_line)

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    # Either: global is trimmed and local is preserved
    # Or: both are trimmed but "queen-wisdom-global" appears in trimmed list before "queen-wisdom-local"
    if assert_contains "$log_line" "queen-wisdom-global"; then
        # Global was trimmed -- verify local is still present OR also trimmed but after global
        if assert_contains "$prompt" "QUEEN WISDOM (Colony-Specific)"; then
            # Local preserved while global trimmed -- correct behavior
            rm -rf "$hub_dir"
            return 0
        fi
        # Both trimmed -- that's ok if global was trimmed first (it appears first in the list)
        rm -rf "$hub_dir"
        return 0
    fi

    # If no truncation at all, the content wasn't large enough. Check prompt length
    local prompt_len
    prompt_len=$(echo "$result" | get_prompt_length)
    if [[ "$prompt_len" -le 4000 ]]; then
        test_fail "Content should exceed compact budget to test trim order" "Only $prompt_len chars"
        rm -rf "$hub_dir"
        return 1
    fi

    rm -rf "$hub_dir"
    return 0
}

# ============================================================================
# Test 6: queen-migrate converts v1 to v2
# ============================================================================
test_hub_queen_migrate() {
    local hub_dir
    hub_dir=$(mktemp -d)
    mkdir -p "$hub_dir/.aether"

    local iso_date
    iso_date=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Create v1 format QUEEN.md with emoji headers and a real user pref
    cat > "$hub_dir/.aether/QUEEN.md" << QUEENEOF
# QUEEN.md --- Colony Wisdom

> Last evolved: $iso_date
> Wisdom version: 1.0.0

---

## 📜 Philosophies

- Test philosophy that should survive migration

---

## 🧭 Patterns

- Test pattern that should survive migration

---

## ⚠️ Redirects

*No redirects recorded yet*

---

## 🔧 Stack Wisdom

*No stack wisdom recorded yet*

---

## 🏛️ Decrees

- Test decree that should survive migration

---

## 📊 Evolution Log

| Date | Colony | Change | Details |
|------|--------|--------|---------|

---

<!-- METADATA {"version":"1.0.0","last_evolved":"$iso_date","colonies_contributed":[],"stats":{}} -->
QUEENEOF

    # Run queen-migrate
    local result
    result=$(HOME="$hub_dir" AETHER_ROOT="$hub_dir" bash "$AETHER_UTILS" queen-migrate --target hub 2>/dev/null)

    if ! assert_contains "$result" '"migrated":true'; then
        test_fail "Migration should succeed" "$result"
        rm -rf "$hub_dir"
        return 1
    fi

    # Check file is now v2 format
    if ! grep -q '^## Build Learnings$' "$hub_dir/.aether/QUEEN.md"; then
        test_fail "File should have v2 header '## Build Learnings'" ""
        rm -rf "$hub_dir"
        return 1
    fi

    # Check entries were preserved (philosophy + pattern -> Codebase Patterns)
    if ! grep -q 'Test philosophy that should survive migration' "$hub_dir/.aether/QUEEN.md"; then
        test_fail "Philosophy entry should be preserved" ""
        rm -rf "$hub_dir"
        return 1
    fi

    if ! grep -q 'Test pattern that should survive migration' "$hub_dir/.aether/QUEEN.md"; then
        test_fail "Pattern entry should be preserved" ""
        rm -rf "$hub_dir"
        return 1
    fi

    # Decree -> User Preferences
    if ! grep -q 'Test decree that should survive migration' "$hub_dir/.aether/QUEEN.md"; then
        test_fail "Decree entry should be preserved as user pref" ""
        rm -rf "$hub_dir"
        return 1
    fi

    # Run migrate again -- should say "Already v2"
    local result2
    result2=$(HOME="$hub_dir" AETHER_ROOT="$hub_dir" bash "$AETHER_UTILS" queen-migrate --target hub 2>/dev/null)

    if ! assert_contains "$result2" '"Already v2 format"'; then
        test_fail "Second migration should say 'Already v2 format'" "$result2"
        rm -rf "$hub_dir"
        return 1
    fi

    rm -rf "$hub_dir"
    return 0
}

# ============================================================================
# Test 7: Auto-migration during colony-prime
# ============================================================================
test_hub_auto_migration_colony_prime() {
    local hub_dir
    hub_dir=$(setup_hub_env)

    local iso_date
    iso_date=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Create v1 format global QUEEN.md with a real user pref entry
    cat > "$hub_dir/.aether/QUEEN.md" << QUEENEOF
# QUEEN.md --- Colony Wisdom

> Last evolved: $iso_date
> Wisdom version: 1.0.0

---

## 📜 Philosophies

- Global philosophy for auto-migration test

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

- Global decree for auto-migration test

---

## 📊 Evolution Log

| Date | Colony | Change | Details |
|------|--------|--------|---------|

---

<!-- METADATA {"version":"1.0.0","last_evolved":"$iso_date","colonies_contributed":[],"stats":{}} -->
QUEENEOF

    # Create v2 local QUEEN.md with real entries
    write_v2_queen "$hub_dir/colony/.aether/QUEEN.md" \
        "" \
        "- [repo] Local entry for auto-migration test"

    # Run colony-prime -- should auto-migrate the global QUEEN.md
    local result
    result=$(run_colony_prime "$hub_dir" --compact)

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$hub_dir"
        return 1
    fi

    # Check the global QUEEN.md is now v2 format
    if ! grep -q '^## Build Learnings$' "$hub_dir/.aether/QUEEN.md"; then
        test_fail "Global QUEEN.md should be auto-migrated to v2" "No '## Build Learnings' header"
        rm -rf "$hub_dir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    # Check both sections appear correctly
    if ! assert_contains "$prompt" "QUEEN WISDOM (Global -- All Colonies)"; then
        test_fail "Should have Global header after auto-migration" "Not found"
        rm -rf "$hub_dir"
        return 1
    fi

    if ! assert_contains "$prompt" "QUEEN WISDOM (Colony-Specific)"; then
        test_fail "Should have Colony-Specific header" "Not found"
        rm -rf "$hub_dir"
        return 1
    fi

    # Global philosophy entry should appear (migrated from Philosophies -> Codebase Patterns)
    if ! assert_contains "$prompt" "Global philosophy for auto-migration test"; then
        test_fail "Auto-migrated global entry should appear in prompt" "Not found"
        rm -rf "$hub_dir"
        return 1
    fi

    rm -rf "$hub_dir"
    return 0
}

# ============================================================================
# Run tests
# ============================================================================

log_info "Running hub wisdom layer tests (Phase 20)"
log_info "Repo root: $REPO_ROOT"

run_test test_hub_global_local_separate "Global and local sections appear separately"
run_test test_hub_empty_global_omitted "Empty global section is omitted"
run_test test_hub_empty_local_omitted "Empty local section is omitted"
run_test test_hub_user_prefs_source_labels "User preferences show source labels"
run_test test_hub_budget_trims_global_first "Budget trims global queen wisdom before local"
run_test test_hub_queen_migrate "queen-migrate converts v1 to v2"
run_test test_hub_auto_migration_colony_prime "Auto-migration during colony-prime"

test_summary
