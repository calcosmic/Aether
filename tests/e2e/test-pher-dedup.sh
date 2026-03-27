#!/usr/bin/env bash
# test-pher-dedup.sh — Pheromone Deduplication E2E Tests (DEDUP-01 through DEDUP-05)
# Task 1.4: SHA-256 content deduplication for pheromone-write
#
# Requirements tested:
#   DEDUP-01: Writing same signal twice results in one active signal (reinforced)
#   DEDUP-02: Reinforced signal has updated created_at and max strength
#   DEDUP-03: Different content for same type creates separate signals
#   DEDUP-04: JSON output distinguishes created vs reinforced action
#   DEDUP-05: content_hash field present on all written signals
#
# NOTE: Written for bash 3.2 (macOS default). No associative arrays.

set -euo pipefail

E2E_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$E2E_SCRIPT_DIR/../.." && pwd)"

# Parse --results-file flag (for master runner integration)
EXTERNAL_RESULTS_FILE=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --results-file) EXTERNAL_RESULTS_FILE="$2"; shift 2 ;;
    *) shift ;;
  esac
done

# Source shared e2e infrastructure
# shellcheck source=./e2e-helpers.sh
source "$E2E_SCRIPT_DIR/e2e-helpers.sh"

# ============================================================================
# Test Setup
# ============================================================================

AREA="DEDUP"
init_results

teardown_test() {
    teardown_e2e_env
}

trap teardown_test EXIT

# setup_clean_env creates an env with empty pheromones.json (no pre-existing signals)
setup_clean_env() {
    local tmp
    tmp=$(setup_e2e_env)

    # Overwrite pheromones.json with clean empty state
    cat > "$tmp/.aether/data/pheromones.json" << 'EOF'
{
  "version": "1.0.0",
  "colony_id": "test-colony",
  "generated_at": "2026-02-18T00:00:00Z",
  "signals": []
}
EOF
    echo "$tmp"
}

# ============================================================================
# DEDUP-01: Writing same signal twice results in one active signal
# ============================================================================

run_dedup01() {
    log_info "DEDUP-01: Duplicate write results in one active signal (reinforced)"

    local tmp
    tmp=$(setup_clean_env)

    local notes=""
    local status="PASS"

    # Write a FOCUS signal
    local write1_out
    write1_out=$(run_in_isolated_env "$tmp" pheromone-write FOCUS "security review needed" --strength 0.7)
    local write1_json
    write1_json=$(extract_json "$write1_out")

    local ok1
    ok1=$(echo "$write1_json" | jq -r '.ok // empty' 2>/dev/null)
    if [[ "$ok1" != "true" ]]; then
        status="FAIL"
        notes="First pheromone-write did not return ok:true (got: $write1_json)"
        teardown_e2e_env
        record_result "DEDUP-01" "$status" "$notes"
        return
    fi

    # Write the SAME FOCUS signal again (same type + same content)
    local write2_out
    write2_out=$(run_in_isolated_env "$tmp" pheromone-write FOCUS "security review needed" --strength 0.8)
    local write2_json
    write2_json=$(extract_json "$write2_out")

    local ok2
    ok2=$(echo "$write2_json" | jq -r '.ok // empty' 2>/dev/null)
    if [[ "$ok2" != "true" ]]; then
        status="FAIL"
        notes="${notes:+$notes; }Second pheromone-write did not return ok:true (got: $write2_json)"
        teardown_e2e_env
        record_result "DEDUP-01" "$status" "$notes"
        return
    fi

    # Count active FOCUS signals - should be exactly 1
    local active_focus
    active_focus=$(jq '[.signals[] | select(.active == true and .type == "FOCUS")] | length' "$tmp/.aether/data/pheromones.json" 2>/dev/null || echo "0")
    if [[ "$active_focus" -ne 1 ]]; then
        status="FAIL"
        notes="${notes:+$notes; }Expected 1 active FOCUS signal after duplicate write, got $active_focus"
    fi

    teardown_e2e_env
    record_result "DEDUP-01" "$status" "${notes:-Duplicate write produces single active signal}"
}

# ============================================================================
# DEDUP-02: Reinforced signal has updated created_at and max strength
# ============================================================================

run_dedup02() {
    log_info "DEDUP-02: Reinforced signal updates created_at and uses max strength"

    local tmp
    tmp=$(setup_clean_env)

    local notes=""
    local status="PASS"

    # Write with strength 0.7
    run_in_isolated_env "$tmp" pheromone-write FOCUS "performance matters" --strength 0.7 > /dev/null

    # Capture created_at of the first signal
    local first_created
    first_created=$(jq -r '[.signals[] | select(.active == true and .type == "FOCUS")][0].created_at' "$tmp/.aether/data/pheromones.json" 2>/dev/null)

    # Small delay to ensure different timestamp
    sleep 1

    # Write same content with higher strength 0.9
    run_in_isolated_env "$tmp" pheromone-write FOCUS "performance matters" --strength 0.9 > /dev/null

    # Check strength is max(0.7, 0.9) = 0.9
    local final_strength
    final_strength=$(jq '[.signals[] | select(.active == true and .type == "FOCUS")][0].strength' "$tmp/.aether/data/pheromones.json" 2>/dev/null)
    if [[ "$final_strength" != "0.9" ]]; then
        status="FAIL"
        notes="Expected strength 0.9 (max), got $final_strength"
    fi

    # Check created_at was updated (should be different from first)
    local final_created
    final_created=$(jq -r '[.signals[] | select(.active == true and .type == "FOCUS")][0].created_at' "$tmp/.aether/data/pheromones.json" 2>/dev/null)
    if [[ "$first_created" == "$final_created" ]]; then
        status="FAIL"
        notes="${notes:+$notes; }created_at was not updated on reinforcement (still $first_created)"
    fi

    # Check reinforcement_count exists and is >= 1
    local reinf_count
    reinf_count=$(jq '[.signals[] | select(.active == true and .type == "FOCUS")][0].reinforcement_count // 0' "$tmp/.aether/data/pheromones.json" 2>/dev/null)
    if [[ "$reinf_count" -lt 1 ]]; then
        status="FAIL"
        notes="${notes:+$notes; }reinforcement_count should be >= 1, got $reinf_count"
    fi

    teardown_e2e_env
    record_result "DEDUP-02" "$status" "${notes:-Reinforced signal has updated timestamp, max strength, and reinforcement_count}"
}

# ============================================================================
# DEDUP-03: Different content for same type creates separate signals
# ============================================================================

run_dedup03() {
    log_info "DEDUP-03: Different content creates separate signals"

    local tmp
    tmp=$(setup_clean_env)

    local notes=""
    local status="PASS"

    # Write first FOCUS signal
    run_in_isolated_env "$tmp" pheromone-write FOCUS "focus on security" > /dev/null

    # Write different FOCUS signal
    run_in_isolated_env "$tmp" pheromone-write FOCUS "focus on performance" > /dev/null

    # Should have 2 active FOCUS signals
    local active_focus
    active_focus=$(jq '[.signals[] | select(.active == true and .type == "FOCUS")] | length' "$tmp/.aether/data/pheromones.json" 2>/dev/null || echo "0")
    if [[ "$active_focus" -ne 2 ]]; then
        status="FAIL"
        notes="Expected 2 active FOCUS signals for different content, got $active_focus"
    fi

    teardown_e2e_env
    record_result "DEDUP-03" "$status" "${notes:-Different content creates separate signals}"
}

# ============================================================================
# DEDUP-04: JSON output distinguishes created vs reinforced
# ============================================================================

run_dedup04() {
    log_info "DEDUP-04: Output JSON distinguishes created vs reinforced action"

    local tmp
    tmp=$(setup_clean_env)

    local notes=""
    local status="PASS"

    # First write should return action: created
    local write1_out
    write1_out=$(run_in_isolated_env "$tmp" pheromone-write REDIRECT "avoid globals" --strength 0.9)
    local write1_json
    write1_json=$(extract_json "$write1_out")

    local action1
    action1=$(echo "$write1_json" | jq -r '.result.action // empty' 2>/dev/null)
    if [[ "$action1" != "created" ]]; then
        status="FAIL"
        notes="First write action should be 'created', got '$action1' (full: $write1_json)"
    fi

    # Second write (same content) should return action: reinforced
    local write2_out
    write2_out=$(run_in_isolated_env "$tmp" pheromone-write REDIRECT "avoid globals" --strength 0.9)
    local write2_json
    write2_json=$(extract_json "$write2_out")

    local action2
    action2=$(echo "$write2_json" | jq -r '.result.action // empty' 2>/dev/null)
    if [[ "$action2" != "reinforced" ]]; then
        status="FAIL"
        notes="${notes:+$notes; }Second write action should be 'reinforced', got '$action2' (full: $write2_json)"
    fi

    teardown_e2e_env
    record_result "DEDUP-04" "$status" "${notes:-Output JSON correctly distinguishes created vs reinforced}"
}

# ============================================================================
# DEDUP-05: content_hash field present on written signals
# ============================================================================

run_dedup05() {
    log_info "DEDUP-05: content_hash field present on written signals"

    local tmp
    tmp=$(setup_clean_env)

    local notes=""
    local status="PASS"

    # Write a signal
    run_in_isolated_env "$tmp" pheromone-write FEEDBACK "code style preference" > /dev/null

    # Check that content_hash field exists and is a 64-char hex string (SHA-256)
    local hash_val
    hash_val=$(jq -r '.signals[0].content_hash // empty' "$tmp/.aether/data/pheromones.json" 2>/dev/null)
    if [[ -z "$hash_val" ]]; then
        status="FAIL"
        notes="content_hash field is missing from written signal"
    elif [[ ${#hash_val} -ne 64 ]]; then
        status="FAIL"
        notes="content_hash should be 64 chars (SHA-256), got ${#hash_val} chars: $hash_val"
    elif ! echo "$hash_val" | grep -Eq '^[0-9a-f]{64}$'; then
        status="FAIL"
        notes="content_hash is not valid hex: $hash_val"
    fi

    # Verify hash is deterministic — same content should produce same hash
    local expected_hash
    expected_hash=$(echo -n "code style preference" | shasum -a 256 | cut -d' ' -f1)
    if [[ "$hash_val" != "$expected_hash" ]]; then
        status="FAIL"
        notes="${notes:+$notes; }content_hash does not match expected SHA-256 (got: $hash_val, expected: $expected_hash)"
    fi

    teardown_e2e_env
    record_result "DEDUP-05" "$status" "${notes:-content_hash field present and correct SHA-256}"
}

# ============================================================================
# Main
# ============================================================================

echo ""
echo "========================================"
echo "  DEDUP: Pheromone Dedup Requirements"
echo "========================================"
echo ""

run_dedup01
run_dedup02
run_dedup03
run_dedup04
run_dedup05

# Write external results file if requested (for master runner)
if [[ -n "$EXTERNAL_RESULTS_FILE" && -n "$RESULTS_FILE" && -f "$RESULTS_FILE" ]]; then
  while IFS='|' read -r req_id status notes; do
    echo "${req_id}=${status}" >> "$EXTERNAL_RESULTS_FILE"
  done < "$RESULTS_FILE"
fi

print_area_results "$AREA"
