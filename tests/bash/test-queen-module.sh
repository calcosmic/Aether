#!/usr/bin/env bash
# Queen Module Smoke Tests
# Tests queen.sh extracted module functions via aether-utils.sh subcommands

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
# Helper: Create isolated test environment with queen support
# ============================================================================
setup_queen_env() {
    local tmp_dir
    tmp_dir=$(mktemp -d)
    mkdir -p "$tmp_dir/.aether/data" "$tmp_dir/.aether/utils" "$tmp_dir/.aether/templates"

    cp "$AETHER_UTILS_SOURCE" "$tmp_dir/.aether/aether-utils.sh"
    chmod +x "$tmp_dir/.aether/aether-utils.sh"

    local utils_source="$(dirname "$AETHER_UTILS_SOURCE")/utils"
    if [[ -d "$utils_source" ]]; then
        cp -r "$utils_source" "$tmp_dir/.aether/"
    fi

    local exchange_source="$(dirname "$AETHER_UTILS_SOURCE")/exchange"
    if [[ -d "$exchange_source" ]]; then
        cp -r "$exchange_source" "$tmp_dir/.aether/"
    fi

    # Copy templates if available (needed for queen-init)
    local templates_source="$(dirname "$AETHER_UTILS_SOURCE")/templates"
    if [[ -d "$templates_source" ]]; then
        cp -r "$templates_source" "$tmp_dir/.aether/"
    fi

    # Write a minimal COLONY_STATE.json
    cat > "$tmp_dir/.aether/data/COLONY_STATE.json" << 'CSEOF'
{
  "version": "3.0",
  "goal": "Test queen module",
  "state": "READY",
  "current_phase": 1,
  "milestone": "First Mound",
  "session_id": "test-queen",
  "initialized_at": "2026-01-01T00:00:00Z",
  "build_started_at": null,
  "plan": { "phases": [{ "id": 1, "name": "Test Phase", "status": "pending" }] },
  "memory": { "phase_learnings": [], "decisions": [], "instincts": [] },
  "errors": { "records": [], "flagged_patterns": [] },
  "events": [],
  "signals": [],
  "graveyards": [],
  "workers": [],
  "spawn_tree": []
}
CSEOF

    echo "$tmp_dir"
}

run_queen_cmd() {
    local tmp_dir="$1"
    shift
    HOME="$tmp_dir" AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" bash "$tmp_dir/.aether/aether-utils.sh" "$@" 2>/dev/null
}

# ============================================================================
# Test: queen.sh module file exists and has valid syntax
# ============================================================================
test_module_exists() {
    local module_path="$PROJECT_ROOT/.aether/utils/queen.sh"

    assert_file_exists "$module_path" || return 1
    bash -n "$module_path" 2>/dev/null || return 1
}

# ============================================================================
# Test: queen-init creates QUEEN.md from template
# ============================================================================
test_queen_init() {
    local tmp_dir
    tmp_dir=$(setup_queen_env)

    # Ensure no existing QUEEN.md
    rm -f "$tmp_dir/.aether/QUEEN.md"

    local result
    result=$(run_queen_cmd "$tmp_dir" queen-init)

    # Check if we got a valid JSON response
    if echo "$result" | jq -e '.ok' >/dev/null 2>&1; then
        local ok_val
        ok_val=$(echo "$result" | jq -r '.ok')
        if [[ "$ok_val" == "true" ]]; then
            # Verify QUEEN.md was created
            if [[ -f "$tmp_dir/.aether/QUEEN.md" ]]; then
                rm -rf "$tmp_dir"
                return 0
            fi
        fi
    fi

    rm -rf "$tmp_dir"
    return 1
}

# ============================================================================
# Test: queen-thresholds returns JSON with threshold values
# ============================================================================
test_queen_thresholds() {
    local tmp_dir
    tmp_dir=$(setup_queen_env)

    local result
    result=$(run_queen_cmd "$tmp_dir" queen-thresholds)

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    # Verify result contains threshold data (JSON structure with .result)
    local has_data
    has_data=$(echo "$result" | jq -r '.result | keys | length')
    [[ "$has_data" -gt 0 ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: queen-init creates QUEEN.md with v2 section headers
# ============================================================================
test_queen_init_new_format() {
    local tmp_dir
    tmp_dir=$(setup_queen_env)
    rm -f "$tmp_dir/.aether/QUEEN.md"

    local result
    result=$(run_queen_cmd "$tmp_dir" queen-init)
    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    # Verify v2 section headers exist
    grep -q "^## User Preferences$" "$tmp_dir/.aether/QUEEN.md" || { rm -rf "$tmp_dir"; return 1; }
    grep -q "^## Codebase Patterns$" "$tmp_dir/.aether/QUEEN.md" || { rm -rf "$tmp_dir"; return 1; }
    grep -q "^## Build Learnings$" "$tmp_dir/.aether/QUEEN.md" || { rm -rf "$tmp_dir"; return 1; }
    grep -q "^## Instincts$" "$tmp_dir/.aether/QUEEN.md" || { rm -rf "$tmp_dir"; return 1; }

    # Verify NO emoji headers
    local emoji_count
    emoji_count=$(grep -cE "Philosophies|Redirects|Stack Wisdom|Decrees" "$tmp_dir/.aether/QUEEN.md" 2>/dev/null || true)
    [[ "$emoji_count" -eq 0 ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: queen-read returns JSON with v2 keys
# ============================================================================
test_queen_read_new_format() {
    local tmp_dir
    tmp_dir=$(setup_queen_env)
    rm -f "$tmp_dir/.aether/QUEEN.md"
    run_queen_cmd "$tmp_dir" queen-init >/dev/null

    local result
    result=$(run_queen_cmd "$tmp_dir" queen-read)
    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    # Verify new keys exist in wisdom
    local has_uprefs has_cp has_bl has_inst
    has_uprefs=$(echo "$result" | jq -r '.result.wisdom | has("user_prefs")' 2>/dev/null)
    has_cp=$(echo "$result" | jq -r '.result.wisdom | has("codebase_patterns")' 2>/dev/null)
    has_bl=$(echo "$result" | jq -r '.result.wisdom | has("build_learnings")' 2>/dev/null)
    has_inst=$(echo "$result" | jq -r '.result.wisdom | has("instincts")' 2>/dev/null)

    [[ "$has_uprefs" == "true" ]] || { rm -rf "$tmp_dir"; return 1; }
    [[ "$has_cp" == "true" ]] || { rm -rf "$tmp_dir"; return 1; }
    [[ "$has_bl" == "true" ]] || { rm -rf "$tmp_dir"; return 1; }
    [[ "$has_inst" == "true" ]] || { rm -rf "$tmp_dir"; return 1; }

    # Verify old keys do NOT exist
    local has_phil
    has_phil=$(echo "$result" | jq -r '.result.wisdom | has("philosophies")' 2>/dev/null)
    [[ "$has_phil" == "false" ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: queen-write-learnings writes entries to Build Learnings section
# ============================================================================
test_queen_write_learnings() {
    local tmp_dir
    tmp_dir=$(setup_queen_env)
    rm -f "$tmp_dir/.aether/QUEEN.md"
    run_queen_cmd "$tmp_dir" queen-init >/dev/null

    local learnings='[{"claim":"TDD catches regressions early","tag":"general","evidence":"phase 3"},{"claim":"awk range fails on macOS","tag":"repo","evidence":"phase 5"}]'
    local result
    result=$(run_queen_cmd "$tmp_dir" queen-write-learnings 1 "Foundation" "$learnings")
    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    # Verify written count is 2
    local written
    written=$(echo "$result" | jq -r '.result.written' 2>/dev/null)
    [[ "$written" -eq 2 ]] || { rm -rf "$tmp_dir"; return 1; }

    # Verify both entries appear in file
    grep -q "TDD catches regressions early" "$tmp_dir/.aether/QUEEN.md" || { rm -rf "$tmp_dir"; return 1; }
    grep -q "awk range fails on macOS" "$tmp_dir/.aether/QUEEN.md" || { rm -rf "$tmp_dir"; return 1; }

    # Verify placeholder was replaced
    local placeholder_count
    placeholder_count=$(grep -c "No build learnings recorded yet" "$tmp_dir/.aether/QUEEN.md" 2>/dev/null || true)
    [[ "$placeholder_count" -eq 0 ]] || { rm -rf "$tmp_dir"; return 1; }

    # Verify phase subsection header exists
    grep -q "### Phase 1: Foundation" "$tmp_dir/.aether/QUEEN.md" || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: queen-write-learnings deduplicates entries
# ============================================================================
test_queen_write_learnings_dedup() {
    local tmp_dir
    tmp_dir=$(setup_queen_env)
    rm -f "$tmp_dir/.aether/QUEEN.md"
    run_queen_cmd "$tmp_dir" queen-init >/dev/null

    local learnings='[{"claim":"always validate JSON input","tag":"repo","evidence":"e1"}]'
    run_queen_cmd "$tmp_dir" queen-write-learnings 1 "Phase One" "$learnings" >/dev/null

    # Write same claim again
    local result
    result=$(run_queen_cmd "$tmp_dir" queen-write-learnings 2 "Phase Two" "$learnings")
    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    # Second write should have written 0 (duplicate)
    local written
    written=$(echo "$result" | jq -r '.result.written' 2>/dev/null)
    [[ "$written" -eq 0 ]] || { rm -rf "$tmp_dir"; return 1; }

    # Verify only one occurrence in file
    local count
    count=$(grep -c "always validate JSON input" "$tmp_dir/.aether/QUEEN.md" 2>/dev/null || true)
    [[ "$count" -eq 1 ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: queen-promote-instinct writes entry to Instincts section
# ============================================================================
test_queen_promote_instinct() {
    local tmp_dir
    tmp_dir=$(setup_queen_env)
    rm -f "$tmp_dir/.aether/QUEEN.md"
    run_queen_cmd "$tmp_dir" queen-init >/dev/null

    local result
    result=$(run_queen_cmd "$tmp_dir" queen-promote-instinct "tests fail after extraction" "check imports first" 0.9 "debugging")
    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    # Verify promoted is true
    local promoted
    promoted=$(echo "$result" | jq -r '.result.promoted' 2>/dev/null)
    [[ "$promoted" == "true" ]] || { rm -rf "$tmp_dir"; return 1; }

    # Verify entry appears in file
    grep -q "check imports first" "$tmp_dir/.aether/QUEEN.md" || { rm -rf "$tmp_dir"; return 1; }
    grep -q "\[instinct\]" "$tmp_dir/.aether/QUEEN.md" || { rm -rf "$tmp_dir"; return 1; }
    grep -q "debugging" "$tmp_dir/.aether/QUEEN.md" || { rm -rf "$tmp_dir"; return 1; }

    # Verify placeholder was replaced
    local placeholder_count
    placeholder_count=$(grep -c "No instincts recorded yet" "$tmp_dir/.aether/QUEEN.md" 2>/dev/null || true)
    [[ "$placeholder_count" -eq 0 ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: queen-promote-instinct deduplicates entries
# ============================================================================
test_queen_promote_instinct_dedup() {
    local tmp_dir
    tmp_dir=$(setup_queen_env)
    rm -f "$tmp_dir/.aether/QUEEN.md"
    run_queen_cmd "$tmp_dir" queen-init >/dev/null

    run_queen_cmd "$tmp_dir" queen-promote-instinct "error occurs" "check logs first" 0.8 "ops" >/dev/null

    # Promote same action again
    local result
    result=$(run_queen_cmd "$tmp_dir" queen-promote-instinct "different trigger" "check logs first" 0.85 "ops")
    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    # Second promotion should return promoted:false (duplicate)
    local promoted
    promoted=$(echo "$result" | jq -r '.result.promoted' 2>/dev/null)
    [[ "$promoted" == "false" ]] || { rm -rf "$tmp_dir"; return 1; }

    # Verify only one instinct entry in Instincts section (not counting Evolution Log)
    local inst_section_line
    inst_section_line=$(grep -n "^## Instincts$" "$tmp_dir/.aether/QUEEN.md" | head -1 | cut -d: -f1)
    local evo_section_line
    evo_section_line=$(grep -n "^## Evolution Log$" "$tmp_dir/.aether/QUEEN.md" | head -1 | cut -d: -f1)
    local inst_count
    inst_count=$(sed -n "${inst_section_line},${evo_section_line}p" "$tmp_dir/.aether/QUEEN.md" | grep -c "check logs first" || true)
    [[ "$inst_count" -eq 1 ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test: queen-read with v1-format QUEEN.md returns mapped data
# ============================================================================
test_queen_read_v1_compat() {
    local tmp_dir
    tmp_dir=$(setup_queen_env)

    # Create a v1-format QUEEN.md manually (with emoji headers as in the original template)
    # Note: printf is used to ensure emoji bytes are preserved correctly
    cat > "$tmp_dir/.aether/QUEEN.md" << 'V1EOF'
# QUEEN.md -- Colony Wisdom

> Last evolved: 2026-01-01T00:00:00Z
> Wisdom version: 1.0.0

---

V1EOF
    # Use printf for emoji headers to avoid heredoc encoding issues
    printf '## \xf0\x9f\x93\x9c Philosophies\n\nTest philosophy entry\n\n---\n\n' >> "$tmp_dir/.aether/QUEEN.md"
    printf '## \xf0\x9f\xa7\xad Patterns\n\nTest pattern entry\n\n---\n\n' >> "$tmp_dir/.aether/QUEEN.md"
    printf '## \xe2\x9a\xa0\xef\xb8\x8f Redirects\n\nTest redirect entry\n\n---\n\n' >> "$tmp_dir/.aether/QUEEN.md"
    printf '## \xf0\x9f\x94\xa7 Stack Wisdom\n\nTest stack entry\n\n---\n\n' >> "$tmp_dir/.aether/QUEEN.md"
    printf '## \xf0\x9f\x8f\x9b\xef\xb8\x8f Decrees\n\nTest decree entry\n\n---\n\n' >> "$tmp_dir/.aether/QUEEN.md"
    printf '## \xf0\x9f\x91\xa4 User Preferences\n\nTest user pref entry\n\n---\n\n' >> "$tmp_dir/.aether/QUEEN.md"
    printf '## \xf0\x9f\x93\x8a Evolution Log\n\n' >> "$tmp_dir/.aether/QUEEN.md"
    cat >> "$tmp_dir/.aether/QUEEN.md" << 'V1EOF2'
| Date | Colony | Change | Details |
|------|--------|--------|---------|
| 2026-01-01T00:00:00Z | system | initialized | Created |

---

<!-- METADATA
{
  "version": "1.0.0",
  "last_evolved": "2026-01-01T00:00:00Z",
  "colonies_contributed": [],
  "stats": {
    "total_philosophies": 1,
    "total_patterns": 1,
    "total_redirects": 1,
    "total_stack_entries": 1,
    "total_decrees": 1,
    "total_user_prefs": 1
  }
}
-->
V1EOF2

    local result
    result=$(run_queen_cmd "$tmp_dir" queen-read)
    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    # Verify v2 keys are returned (mapped from v1)
    local has_cp has_bl has_inst has_up
    has_cp=$(echo "$result" | jq -r '.result.wisdom | has("codebase_patterns")' 2>/dev/null)
    has_bl=$(echo "$result" | jq -r '.result.wisdom | has("build_learnings")' 2>/dev/null)
    has_inst=$(echo "$result" | jq -r '.result.wisdom | has("instincts")' 2>/dev/null)
    has_up=$(echo "$result" | jq -r '.result.wisdom | has("user_prefs")' 2>/dev/null)

    [[ "$has_cp" == "true" ]] || { rm -rf "$tmp_dir"; return 1; }
    [[ "$has_bl" == "true" ]] || { rm -rf "$tmp_dir"; return 1; }
    [[ "$has_inst" == "true" ]] || { rm -rf "$tmp_dir"; return 1; }
    [[ "$has_up" == "true" ]] || { rm -rf "$tmp_dir"; return 1; }

    # Verify codebase_patterns contains old sections mapped together
    local cp_val
    cp_val=$(echo "$result" | jq -r '.result.wisdom.codebase_patterns' 2>/dev/null)
    assert_contains "$cp_val" "Test philosophy entry" || { rm -rf "$tmp_dir"; return 1; }
    assert_contains "$cp_val" "Test pattern entry" || { rm -rf "$tmp_dir"; return 1; }

    # Verify user_prefs contains old user prefs + decrees
    local up_val
    up_val=$(echo "$result" | jq -r '.result.wisdom.user_prefs' 2>/dev/null)
    assert_contains "$up_val" "Test user pref entry" || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Run all tests
# ============================================================================
echo "=== Queen Module Smoke Tests ==="
echo ""

run_test test_module_exists "queen.sh exists and passes syntax check"
run_test test_queen_init "queen-init creates QUEEN.md via dispatcher"
run_test test_queen_thresholds "queen-thresholds returns JSON with threshold values"
run_test test_queen_init_new_format "queen-init creates v2 format with 4 clean sections"
run_test test_queen_read_new_format "queen-read returns v2 keys (codebase_patterns, build_learnings, instincts)"
run_test test_queen_write_learnings "queen-write-learnings writes entries to Build Learnings section"
run_test test_queen_write_learnings_dedup "queen-write-learnings deduplicates same claim"
run_test test_queen_promote_instinct "queen-promote-instinct writes entry to Instincts section"
run_test test_queen_promote_instinct_dedup "queen-promote-instinct deduplicates same action"
run_test test_queen_read_v1_compat "queen-read with v1 format returns mapped v2 keys"

test_summary
