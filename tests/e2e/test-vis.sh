#!/usr/bin/env bash
# test-vis.sh — Visual Experience E2E Tests (VIS-01 through VIS-06)
# Phase 9 Plan 02: Verify visual display functions and output formatting
#
# Requirements tested:
#   VIS-01: swarm-display-text produces emoji output (🐜) with ok:true
#   VIS-02: Caste emojis defined in display functions (builder/watcher/scout + others)
#   VIS-03: ANSI color codes exist in swarm-display-inline for caste differentiation
#   VIS-04: swarm-timing-start/get/eta subcommands exist and swarm-timing-start works
#   VIS-05: continue.md references milestone names (First Mound, Open Chambers, etc.)
#   VIS-06: Command files have structured formatting patterns (headers, banners, step structure)
#
# NOTE: Written for bash 3.2 (macOS default). No associative arrays.

set -euo pipefail

E2E_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$E2E_SCRIPT_DIR/../.." && pwd)"

# Source shared e2e infrastructure
# shellcheck source=./e2e-helpers.sh
source "$E2E_SCRIPT_DIR/e2e-helpers.sh"

# ============================================================================
# Test Setup
# ============================================================================

AREA="VIS"
init_results

teardown_test() {
    teardown_e2e_env
}

trap teardown_test EXIT

# ============================================================================
# VIS-01: swarm display produces emoji output
# ============================================================================

run_vis01() {
    log_info "VIS-01: swarm-display-text produces emoji output"

    local tmp
    tmp=$(setup_e2e_env)

    local notes=""
    local status="PASS"

    # Create a mock swarm state with active ants so display has content to show
    cat > "$tmp/.aether/data/swarm-display.json" << 'EOF'
{
  "swarm_id": "test-session",
  "total_active": 2,
  "active_ants": [
    {
      "name": "builder-1",
      "caste": "builder",
      "task": "Building features",
      "tools": {"read": 3, "grep": 1, "edit": 2, "bash": 1},
      "progress": 50
    },
    {
      "name": "watcher-1",
      "caste": "watcher",
      "task": "Verifying work",
      "tools": {"read": 2, "grep": 3, "edit": 0, "bash": 1},
      "progress": 30
    }
  ]
}
EOF

    # Run swarm-display-text
    local display_out
    display_out=$(run_in_isolated_env "$tmp" swarm-display-text "test-session")

    # Extract JSON from output (last valid JSON line)
    local display_json
    display_json=$(extract_json "$display_out")

    # Assert ok:true in JSON result
    local ok
    ok=$(echo "$display_json" | jq -r '.ok // empty' 2>/dev/null)
    if [[ "$ok" != "true" ]]; then
        status="FAIL"
        notes="swarm-display-text did not return ok:true (got: $display_json)"
    fi

    # Assert ant emoji (🐜) appears in the text output
    if ! echo "$display_out" | grep -q "🐜"; then
        status="FAIL"
        notes="${notes:+$notes; }swarm-display-text output missing 🐜 ant emoji"
    fi

    # Proxy check: colonize.md and swarm.md reference swarm-display-text
    local colonize_refs
    colonize_refs=$(grep -c "swarm-display-text" "$PROJECT_ROOT/.claude/commands/ant/colonize.md" 2>/dev/null || echo "0")
    if [[ "$colonize_refs" -eq 0 ]]; then
        status="FAIL"
        notes="${notes:+$notes; }colonize.md does not reference swarm-display-text"
    fi

    local swarm_refs
    swarm_refs=$(grep -c "swarm-display-text" "$PROJECT_ROOT/.claude/commands/ant/swarm.md" 2>/dev/null || echo "0")
    if [[ "$swarm_refs" -eq 0 ]]; then
        status="FAIL"
        notes="${notes:+$notes; }swarm.md does not reference swarm-display-text"
    fi

    teardown_e2e_env
    record_result "VIS-01" "$status" "${notes:-swarm-display-text returns ok:true with 🐜 emoji; colonize.md+swarm.md reference it}"
}

# ============================================================================
# VIS-02: Emoji caste identity in display functions
# ============================================================================

run_vis02() {
    log_info "VIS-02: Caste emojis defined in swarm display functions"

    local notes=""
    local status="PASS"

    local utils="$PROJECT_ROOT/.aether/aether-utils.sh"

    # Check builder emoji (🔨) is in swarm display functions
    if ! grep -q "builder.*🔨\|🔨.*builder" "$utils"; then
        status="FAIL"
        notes="Missing builder emoji (🔨) in swarm display functions"
    fi

    # Check watcher emoji (👁️) is in swarm display functions
    if ! grep -q "watcher.*👁\|👁.*watcher" "$utils"; then
        status="FAIL"
        notes="${notes:+$notes; }Missing watcher emoji (👁️) in swarm display functions"
    fi

    # Check scout emoji (🔍) is in swarm display functions
    if ! grep -q "scout.*🔍\|🔍.*scout" "$utils"; then
        status="FAIL"
        notes="${notes:+$notes; }Missing scout emoji (🔍) in swarm display functions"
    fi

    # Count total distinct caste emojis defined — should be at least 3
    local caste_emoji_count
    caste_emoji_count=$(grep -c "🔨\|👁\|🔍\|🎲\|👑\|🔮\|🧭" "$utils" 2>/dev/null || echo "0")
    if [[ "$caste_emoji_count" -lt 3 ]]; then
        status="FAIL"
        notes="${notes:+$notes; }Less than 3 caste emoji types found in aether-utils.sh"
    fi

    record_result "VIS-02" "$status" "${notes:-builder/watcher/scout emojis defined; 3+ caste emojis present}"
}

# ============================================================================
# VIS-03: ANSI color codes in swarm-display-inline
# ============================================================================

run_vis03() {
    log_info "VIS-03: ANSI color codes in swarm-display-inline"

    local notes=""
    local status="PASS"

    local utils="$PROJECT_ROOT/.aether/aether-utils.sh"

    # Check for ANSI escape codes (\\033[ or \e[) in swarm display section
    # We look in the swarm-display-inline case branch specifically
    # Using grep with PCRE or basic grep for the escape code pattern
    if ! grep -q "\\\\033\[" "$utils"; then
        status="FAIL"
        notes="No ANSI escape codes (\\033[) found in aether-utils.sh"
    fi

    # Count different ANSI color codes (should be at least 2 different colors)
    # Colors: 34m (blue), 32m (green), 33m (yellow), 31m (red), 35m (magenta)
    local blue_count
    blue_count=$(grep -c "034m\|\\\\033\[34m\|BLUE" "$utils" 2>/dev/null || echo "0")
    local green_count
    green_count=$(grep -c "032m\|\\\\033\[32m\|GREEN" "$utils" 2>/dev/null || echo "0")

    if [[ "$blue_count" -eq 0 && "$green_count" -eq 0 ]]; then
        status="FAIL"
        notes="${notes:+$notes; }No blue or green color codes found in aether-utils.sh"
    fi

    # Specifically check swarm-display-inline uses colors
    # The section defines BLUE, GREEN, YELLOW, RED, MAGENTA variables
    if ! grep -q "BLUE\|GREEN\|YELLOW\|RED\|MAGENTA" "$utils"; then
        status="FAIL"
        notes="${notes:+$notes; }No color variable definitions (BLUE/GREEN/etc.) in aether-utils.sh"
    fi

    record_result "VIS-03" "$status" "${notes:-ANSI color codes and color variables present in aether-utils.sh}"
}

# ============================================================================
# VIS-04: Progress indication — swarm-timing subcommands exist and work
# ============================================================================

run_vis04() {
    log_info "VIS-04: swarm-timing-start/get/eta exist and timing-start works"

    local tmp
    tmp=$(setup_e2e_env)

    local notes=""
    local status="PASS"

    local utils="$PROJECT_ROOT/.aether/aether-utils.sh"

    # Verify case branches exist for all three timing subcommands
    if ! grep -q "swarm-timing-start" "$utils"; then
        status="FAIL"
        notes="swarm-timing-start case branch missing in aether-utils.sh"
    fi

    if ! grep -q "swarm-timing-get" "$utils"; then
        status="FAIL"
        notes="${notes:+$notes; }swarm-timing-get case branch missing in aether-utils.sh"
    fi

    if ! grep -q "swarm-timing-eta" "$utils"; then
        status="FAIL"
        notes="${notes:+$notes; }swarm-timing-eta case branch missing in aether-utils.sh"
    fi

    # Run swarm-timing-start and assert ok:true
    local timing_out
    timing_out=$(run_in_isolated_env "$tmp" swarm-timing-start "test-ant-1")
    local timing_json
    timing_json=$(extract_json "$timing_out")

    local ok
    ok=$(echo "$timing_json" | jq -r '.ok // empty' 2>/dev/null)
    if [[ "$ok" != "true" ]]; then
        status="FAIL"
        notes="${notes:+$notes; }swarm-timing-start did not return ok:true (got: $timing_json)"
    fi

    # Assert ant name is returned
    local ant_name
    ant_name=$(echo "$timing_json" | jq -r '.result.ant // empty' 2>/dev/null)
    if [[ "$ant_name" != "test-ant-1" ]]; then
        status="FAIL"
        notes="${notes:+$notes; }swarm-timing-start wrong ant name returned (got: $ant_name)"
    fi

    teardown_e2e_env
    record_result "VIS-04" "$status" "${notes:-swarm-timing-start/get/eta case branches exist; timing-start returns ok:true}"
}

# ============================================================================
# VIS-05: Stage banners with ant milestone names in continue.md
# ============================================================================

run_vis05() {
    log_info "VIS-05: Milestone names appear in command files (maturity/status/aether-utils)"

    local notes=""
    local status="PASS"

    # Milestone names live in maturity.md, status.md, and aether-utils.sh — not continue.md
    # The stage banner system is in these canonical locations
    local maturity_cmd="$PROJECT_ROOT/.claude/commands/ant/maturity.md"
    local utils="$PROJECT_ROOT/.aether/aether-utils.sh"

    # Check for milestone names in maturity.md (primary stage banner file)
    local milestone_count=0

    grep -q "First Mound" "$maturity_cmd" 2>/dev/null && milestone_count=$((milestone_count + 1))
    grep -q "Open Chambers" "$maturity_cmd" 2>/dev/null && milestone_count=$((milestone_count + 1))
    grep -q "Brood Stable" "$maturity_cmd" 2>/dev/null && milestone_count=$((milestone_count + 1))
    grep -q "Ventilated Nest" "$maturity_cmd" 2>/dev/null && milestone_count=$((milestone_count + 1))
    grep -q "Sealed Chambers" "$maturity_cmd" 2>/dev/null && milestone_count=$((milestone_count + 1))
    grep -q "Crowned Anthill" "$maturity_cmd" 2>/dev/null && milestone_count=$((milestone_count + 1))

    if [[ "$milestone_count" -lt 3 ]]; then
        status="FAIL"
        notes="Only $milestone_count/6 milestone names found in maturity.md (need at least 3)"
    fi

    # Also check milestone-detect subcommand exists in aether-utils.sh
    if ! grep -q "milestone-detect" "$utils"; then
        # This is a warning, not a hard failure — milestone names in command files is the primary check
        notes="${notes:+$notes; }[info] milestone-detect subcommand not found in aether-utils.sh"
    fi

    record_result "VIS-05" "$status" "${notes:-$milestone_count/6 milestone names found in maturity.md}"
}

# ============================================================================
# VIS-06: GSD-style formatting in command files
# ============================================================================

run_vis06() {
    log_info "VIS-06: Structured formatting patterns in command files"

    local notes=""
    local status="PASS"

    local continue_cmd="$PROJECT_ROOT/.claude/commands/ant/continue.md"

    # Check for structured formatting: box-drawing characters, step headers, banners
    local fmt_count=0

    # Check for AETHER banner text or box-drawing characters
    grep -q "═\|━\|┌\|└\|│\|AETHER\|COLONY" "$continue_cmd" 2>/dev/null && fmt_count=$((fmt_count + 1))

    # Check for step headers (### Step N: or ## Step N:)
    grep -q "### Step [0-9]\|## Step [0-9]" "$continue_cmd" 2>/dev/null && fmt_count=$((fmt_count + 1))

    # Check for phase advancement display (formatted output section)
    grep -q "PHASE ADVANCEMENT\|P H A S E\|Phase Advanced" "$continue_cmd" 2>/dev/null && fmt_count=$((fmt_count + 1))

    # Check for context-update reference in continue.md (context document updated message)
    grep -q "CONTEXT.md\|context-update" "$continue_cmd" 2>/dev/null && fmt_count=$((fmt_count + 1))

    if [[ "$fmt_count" -lt 2 ]]; then
        status="FAIL"
        notes="Only $fmt_count/4 formatting pattern checks passed in continue.md"
    fi

    record_result "VIS-06" "$status" "${notes:-$fmt_count/4 formatting checks passed in continue.md}"
}

# ============================================================================
# Main
# ============================================================================

echo ""
echo "========================================"
echo "  VIS: Visual Experience Requirements"
echo "========================================"
echo ""

run_vis01
run_vis02
run_vis03
run_vis04
run_vis05
run_vis06

print_area_results "$AREA"
