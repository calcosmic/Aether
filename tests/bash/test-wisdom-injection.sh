#!/usr/bin/env bash
# Tests for QUEEN wisdom injection: fresh vs accumulated colony behavior
# Phase 18: Local Wisdom Injection

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

# ============================================================================
# Helper: Create a test colony with v2 QUEEN.md (placeholder-only)
# ============================================================================
setup_colony_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    local aether_dir="$tmpdir/.aether"
    local data_dir="$aether_dir/data"
    mkdir -p "$data_dir"

    local iso_date
    iso_date=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Create v2 QUEEN.md with placeholder-only text (no real entries)
    cat > "$aether_dir/QUEEN.md" << QUEENEOF
# QUEEN.md -- Colony Wisdom

> Last evolved: $iso_date
> Wisdom version: 2.0.0

---

## User Preferences

Communication style, expertise level, and decision-making patterns observed from the user (the Queen). These shape how the colony communicates and what it prioritizes.

*No user preferences recorded yet.*

---

## Codebase Patterns

Validated approaches that work in this codebase, and anti-patterns to avoid. Includes architecture conventions, naming patterns, error handling style, and technology-specific insights.

*No codebase patterns recorded yet.*

---

## Build Learnings

What worked and what failed during builds. Captures the full picture of colony experience -- successes, failures, and adjustments.

*No build learnings recorded yet.*

---

## Instincts

High-confidence behavioral patterns that have been validated through repeated colony work. Auto-promoted when confidence reaches 0.8 or higher.

*No instincts recorded yet.*

---

## Evolution Log

| Date | Source | Type | Details |
|------|--------|------|---------|
| $iso_date | system | initialized | QUEEN.md created from template |

---

<!-- METADATA {"version":"2.0.0","wisdom_version":"2.0","last_evolved":"$iso_date","colonies_contributed":[],"stats":{"total_user_prefs":0,"total_codebase_patterns":0,"total_build_learnings":0,"total_instincts":0}} -->
QUEENEOF

    # Create COLONY_STATE.json
    cat > "$data_dir/COLONY_STATE.json" << 'STATEEOF'
{
  "session_id": "test_wisdom",
  "goal": "test wisdom injection",
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

# Helper: add real entries to a v2 QUEEN.md to simulate accumulated wisdom
add_accumulated_wisdom() {
    local tmpdir="$1"
    local queen_file="$tmpdir/.aether/QUEEN.md"
    local iso_date
    iso_date=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    cat > "$queen_file" << QUEENEOF
# QUEEN.md -- Colony Wisdom

> Last evolved: $iso_date
> Wisdom version: 2.0.0

---

## User Preferences

Communication style, expertise level, and decision-making patterns observed from the user (the Queen). These shape how the colony communicates and what it prioritizes.

*No user preferences recorded yet.*

---

## Codebase Patterns

Validated approaches that work in this codebase, and anti-patterns to avoid. Includes architecture conventions, naming patterns, error handling style, and technology-specific insights.

- [general] Test pattern that should appear in prompt

---

## Build Learnings

What worked and what failed during builds. Captures the full picture of colony experience -- successes, failures, and adjustments.

### Phase 1: test
- [repo] Test learning that should appear in prompt

---

## Instincts

High-confidence behavioral patterns that have been validated through repeated colony work. Auto-promoted when confidence reaches 0.8 or higher.

- [instinct] testing (0.9): Test instinct that should appear in prompt

---

## Evolution Log

| Date | Source | Type | Details |
|------|--------|------|---------|
| $iso_date | system | initialized | QUEEN.md created from template |

---

<!-- METADATA {"version":"2.0.0","wisdom_version":"2.0","last_evolved":"$iso_date","colonies_contributed":[],"stats":{"total_user_prefs":0,"total_codebase_patterns":1,"total_build_learnings":1,"total_instincts":1}} -->
QUEENEOF
}

# ============================================================================
# Tests
# ============================================================================

test_fresh_colony_no_queen_wisdom() {
    # Fresh colony with placeholder-only QUEEN.md should NOT have QUEEN WISDOM section
    local tmpdir
    tmpdir=$(setup_colony_env)

    local result
    result=$(run_colony_prime "$tmpdir" --compact)

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    if assert_contains "$prompt" "QUEEN WISDOM"; then
        test_fail "Fresh colony should NOT have QUEEN WISDOM section" "Found QUEEN WISDOM in prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_accumulated_colony_has_queen_wisdom() {
    # Colony with real entries should have QUEEN WISDOM (Colony-Specific) section
    local tmpdir
    tmpdir=$(setup_colony_env)
    add_accumulated_wisdom "$tmpdir"

    local result
    result=$(run_colony_prime "$tmpdir" --compact)

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    if ! assert_contains "$prompt" "QUEEN WISDOM (Colony-Specific)"; then
        test_fail "Accumulated colony should have QUEEN WISDOM (Colony-Specific)" "Not found in prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_description_text_stripped() {
    # Accumulated colony should NOT have description paragraphs in prompt
    local tmpdir
    tmpdir=$(setup_colony_env)
    add_accumulated_wisdom "$tmpdir"

    local result
    result=$(run_colony_prime "$tmpdir" --compact)

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    if assert_contains "$prompt" "Validated approaches"; then
        test_fail "Description text 'Validated approaches' should be stripped" ""
        rm -rf "$tmpdir"
        return 1
    fi
    if assert_contains "$prompt" "What worked and what failed"; then
        test_fail "Description text 'What worked and what failed' should be stripped" ""
        rm -rf "$tmpdir"
        return 1
    fi
    if assert_contains "$prompt" "High-confidence behavioral patterns"; then
        test_fail "Description text 'High-confidence behavioral patterns' should be stripped" ""
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_entry_content_preserved() {
    # Accumulated colony should have actual entry content in prompt
    local tmpdir
    tmpdir=$(setup_colony_env)
    add_accumulated_wisdom "$tmpdir"

    local result
    result=$(run_colony_prime "$tmpdir" --compact)

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    if ! assert_contains "$prompt" "Test pattern"; then
        test_fail "Entry 'Test pattern' should be preserved" ""
        rm -rf "$tmpdir"
        return 1
    fi
    if ! assert_contains "$prompt" "Test learning"; then
        test_fail "Entry 'Test learning' should be preserved" ""
        rm -rf "$tmpdir"
        return 1
    fi
    if ! assert_contains "$prompt" "Test instinct"; then
        test_fail "Entry 'Test instinct' should be preserved" ""
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_phase_headers_preserved() {
    # Phase subsection headers (### Phase N: ...) should be preserved
    local tmpdir
    tmpdir=$(setup_colony_env)
    add_accumulated_wisdom "$tmpdir"

    local result
    result=$(run_colony_prime "$tmpdir" --compact)

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    if ! assert_contains "$prompt" "### Phase 1: test"; then
        test_fail "Phase header '### Phase 1: test' should be preserved" ""
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Run tests
# ============================================================================

log_info "Running wisdom injection tests"
log_info "Repo root: $REPO_ROOT"

run_test test_fresh_colony_no_queen_wisdom "Fresh colony has no QUEEN WISDOM section"
run_test test_accumulated_colony_has_queen_wisdom "Accumulated colony has QUEEN WISDOM section"
run_test test_description_text_stripped "Description text stripped from wisdom injection"
run_test test_entry_content_preserved "Entry content preserved in wisdom injection"
run_test test_phase_headers_preserved "Phase headers preserved in wisdom injection"

test_summary
