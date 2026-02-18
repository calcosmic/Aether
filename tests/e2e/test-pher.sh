#!/usr/bin/env bash
# test-pher.sh — Pheromone System E2E Tests (PHER-01 through PHER-05)
# Phase 9 Plan 02: Verify pheromone write/read/prime/instinct cycle
#
# Requirements tested:
#   PHER-01: FOCUS signals written and readable
#   PHER-02: REDIRECT signals written and readable
#   PHER-03: FEEDBACK signals written and readable
#   PHER-04: pheromone-prime produces ACTIVE SIGNALS block for builders
#   PHER-05: instinct-read returns valid JSON; pheromone-prime includes instinct section
#
# NOTE: Written for bash 3.2 (macOS default). No associative arrays.
# Response format: {"ok":true,"result":{...}} (not "data")

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

AREA="PHER"
init_results

teardown_test() {
    teardown_e2e_env
}

trap teardown_test EXIT

# setup_clean_env creates an env with empty pheromones.json (no pre-existing signals)
# This avoids fixture signal format issues (string vs object content)
setup_clean_env() {
    local tmp
    tmp=$(setup_e2e_env)

    # Overwrite pheromones.json with clean empty state (proper format)
    cat > "$tmp/.aether/data/pheromones.json" << 'EOF'
{
  "version": "1.0.0",
  "colony_id": "test-colony",
  "generated_at": "2026-02-18T00:00:00Z",
  "signals": [],
  "midden": []
}
EOF
    echo "$tmp"
}

# ============================================================================
# PHER-01: FOCUS signals
# ============================================================================

run_pher01() {
    log_info "PHER-01: FOCUS signals write/read"

    local tmp
    tmp=$(setup_clean_env)

    local notes=""
    local status="PASS"

    # Write a FOCUS signal
    local write_out
    write_out=$(run_in_isolated_env "$tmp" pheromone-write FOCUS "test focus area" --strength 0.8)
    local write_json
    write_json=$(extract_json "$write_out")

    # Assert ok:true from write
    local ok
    ok=$(echo "$write_json" | jq -r '.ok // empty' 2>/dev/null)
    if [[ "$ok" != "true" ]]; then
        status="FAIL"
        notes="pheromone-write FOCUS did not return ok:true (got: $write_json)"
    fi

    # Assert signal_id is present (in .result, not .data)
    local sig_id
    sig_id=$(echo "$write_json" | jq -r '.result.signal_id // empty' 2>/dev/null)
    if [[ -z "$sig_id" ]]; then
        status="FAIL"
        notes="${notes:+$notes; }No signal_id in .result (got: $write_json)"
    fi

    # Read all pheromones and assert FOCUS signal is present (.result.signals[])
    local read_out
    read_out=$(run_in_isolated_env "$tmp" pheromone-read)
    local read_json
    read_json=$(extract_json "$read_out")

    local focus_count
    focus_count=$(echo "$read_json" | jq '[.result.signals[]? | select(.type == "FOCUS")] | length' 2>/dev/null || echo "0")
    if [[ "$focus_count" -eq 0 ]]; then
        status="FAIL"
        notes="${notes:+$notes; }pheromone-read returned no FOCUS signals in .result.signals"
    fi

    # Read with FOCUS type filter
    local filter_out
    filter_out=$(run_in_isolated_env "$tmp" pheromone-read FOCUS)
    local filter_json
    filter_json=$(extract_json "$filter_out")

    local filtered_count
    filtered_count=$(echo "$filter_json" | jq '[.result.signals[]? | select(.type == "FOCUS")] | length' 2>/dev/null || echo "0")
    if [[ "$filtered_count" -eq 0 ]]; then
        status="FAIL"
        notes="${notes:+$notes; }pheromone-read FOCUS filter returned no signals"
    fi

    teardown_e2e_env
    record_result "PHER-01" "$status" "${notes:-FOCUS signals write/read/filter all work}"
}

# ============================================================================
# PHER-02: REDIRECT signals
# ============================================================================

run_pher02() {
    log_info "PHER-02: REDIRECT signals write/read"

    local tmp
    tmp=$(setup_clean_env)

    local notes=""
    local status="PASS"

    # Write a REDIRECT signal
    local write_out
    write_out=$(run_in_isolated_env "$tmp" pheromone-write REDIRECT "avoid this pattern" --strength 0.9)
    local write_json
    write_json=$(extract_json "$write_out")

    local ok
    ok=$(echo "$write_json" | jq -r '.ok // empty' 2>/dev/null)
    if [[ "$ok" != "true" ]]; then
        status="FAIL"
        notes="pheromone-write REDIRECT did not return ok:true (got: $write_json)"
    fi

    # Read with REDIRECT filter
    local filter_out
    filter_out=$(run_in_isolated_env "$tmp" pheromone-read REDIRECT)
    local filter_json
    filter_json=$(extract_json "$filter_out")

    local redirect_count
    redirect_count=$(echo "$filter_json" | jq '[.result.signals[]? | select(.type == "REDIRECT")] | length' 2>/dev/null || echo "0")
    if [[ "$redirect_count" -eq 0 ]]; then
        status="FAIL"
        notes="${notes:+$notes; }pheromone-read REDIRECT returned no signals"
    fi

    teardown_e2e_env
    record_result "PHER-02" "$status" "${notes:-REDIRECT signals write/read filter work}"
}

# ============================================================================
# PHER-03: FEEDBACK signals
# ============================================================================

run_pher03() {
    log_info "PHER-03: FEEDBACK signals write/read"

    local tmp
    tmp=$(setup_clean_env)

    local notes=""
    local status="PASS"

    # Write a FEEDBACK signal
    local write_out
    write_out=$(run_in_isolated_env "$tmp" pheromone-write FEEDBACK "observed behavior" --strength 0.5)
    local write_json
    write_json=$(extract_json "$write_out")

    local ok
    ok=$(echo "$write_json" | jq -r '.ok // empty' 2>/dev/null)
    if [[ "$ok" != "true" ]]; then
        status="FAIL"
        notes="pheromone-write FEEDBACK did not return ok:true (got: $write_json)"
    fi

    # Read with FEEDBACK filter
    local filter_out
    filter_out=$(run_in_isolated_env "$tmp" pheromone-read FEEDBACK)
    local filter_json
    filter_json=$(extract_json "$filter_out")

    local feedback_count
    feedback_count=$(echo "$filter_json" | jq '[.result.signals[]? | select(.type == "FEEDBACK")] | length' 2>/dev/null || echo "0")
    if [[ "$feedback_count" -eq 0 ]]; then
        status="FAIL"
        notes="${notes:+$notes; }pheromone-read FEEDBACK returned no signals"
    fi

    teardown_e2e_env
    record_result "PHER-03" "$status" "${notes:-FEEDBACK signals write/read filter work}"
}

# ============================================================================
# PHER-04: Auto-injection into builders via pheromone-prime
# ============================================================================

run_pher04() {
    log_info "PHER-04: pheromone-prime produces ACTIVE SIGNALS block"

    local tmp
    tmp=$(setup_clean_env)

    local notes=""
    local status="PASS"

    # Write signals first so prime has something to inject
    run_in_isolated_env "$tmp" pheromone-write FOCUS "test focus" --strength 0.8 > /dev/null
    run_in_isolated_env "$tmp" pheromone-write REDIRECT "avoid test" --strength 0.9 > /dev/null

    # Run pheromone-prime
    local prime_out
    prime_out=$(run_in_isolated_env "$tmp" pheromone-prime)
    local prime_json
    prime_json=$(extract_json "$prime_out")

    # Assert ok:true
    local ok
    ok=$(echo "$prime_json" | jq -r '.ok // empty' 2>/dev/null)
    if [[ "$ok" != "true" ]]; then
        status="FAIL"
        notes="pheromone-prime did not return ok:true (got: $prime_json)"
    fi

    # Assert signal_count > 0
    local sig_count
    sig_count=$(echo "$prime_json" | jq -r '.result.signal_count // 0' 2>/dev/null)
    if [[ "$sig_count" -eq 0 ]]; then
        status="FAIL"
        notes="${notes:+$notes; }pheromone-prime signal_count is 0"
    fi

    # Assert prompt_section contains "ACTIVE SIGNALS"
    local section
    section=$(echo "$prime_json" | jq -r '.result.prompt_section // empty' 2>/dev/null)
    if [[ -z "$section" || "$section" == "null" ]]; then
        status="FAIL"
        notes="${notes:+$notes; }prompt_section is empty"
    elif ! echo "$section" | grep -q "ACTIVE SIGNALS"; then
        status="FAIL"
        notes="${notes:+$notes; }prompt_section missing 'ACTIVE SIGNALS' header (got: ${section:0:100})"
    fi

    # Proxy check: build.md must reference pheromone-prime
    if ! grep -q "pheromone-prime" "$PROJECT_ROOT/.claude/commands/ant/build.md"; then
        status="FAIL"
        notes="${notes:+$notes; }build.md does not reference pheromone-prime"
    fi

    teardown_e2e_env
    record_result "PHER-04" "$status" "${notes:-pheromone-prime returns ACTIVE SIGNALS; build.md references it}"
}

# ============================================================================
# PHER-05: Instincts applied — instinct-read returns valid JSON
# ============================================================================

run_pher05() {
    log_info "PHER-05: instinct-read returns valid JSON with instincts array"

    local tmp
    tmp=$(setup_clean_env)

    local notes=""
    local status="PASS"

    # instinct-read on fresh colony (no instincts yet) should return empty instincts array
    local read_out
    read_out=$(run_in_isolated_env "$tmp" instinct-read)
    local read_json
    read_json=$(extract_json "$read_out")

    # Assert ok:true
    local ok
    ok=$(echo "$read_json" | jq -r '.ok // empty' 2>/dev/null)
    if [[ "$ok" != "true" ]]; then
        status="FAIL"
        notes="instinct-read did not return ok:true (got: $read_json)"
    fi

    # Assert result has instincts array (.result.instincts)
    local has_instincts
    has_instincts=$(echo "$read_json" | jq 'if .result.instincts | type == "array" then "yes" else "no" end' 2>/dev/null || echo "no")
    if [[ "$has_instincts" != '"yes"' ]]; then
        status="FAIL"
        notes="${notes:+$notes; }instinct-read response missing .result.instincts array (got: $read_json)"
    fi

    # Verify pheromone-prime source contains INSTINCTS section header
    if ! grep -q "INSTINCTS" "$PROJECT_ROOT/.aether/aether-utils.sh"; then
        status="FAIL"
        notes="${notes:+$notes; }aether-utils.sh does not contain INSTINCTS section in pheromone-prime"
    fi

    # Proxy: verify build.md references pheromone_section variable injection
    if ! grep -q "pheromone_section" "$PROJECT_ROOT/.claude/commands/ant/build.md"; then
        status="FAIL"
        notes="${notes:+$notes; }build.md missing pheromone_section injection reference"
    fi

    teardown_e2e_env
    record_result "PHER-05" "$status" "${notes:-instinct-read valid; pheromone-prime INSTINCTS block exists; build.md injects pheromone_section}"
}

# ============================================================================
# Main
# ============================================================================

echo ""
echo "========================================"
echo "  PHER: Pheromone System Requirements"
echo "========================================"
echo ""

run_pher01
run_pher02
run_pher03
run_pher04
run_pher05

# Write external results file if requested (for master runner)
if [[ -n "$EXTERNAL_RESULTS_FILE" && -n "$RESULTS_FILE" && -f "$RESULTS_FILE" ]]; then
  while IFS='|' read -r req_id status notes; do
    echo "${req_id}=${status}" >> "$EXTERNAL_RESULTS_FILE"
  done < "$RESULTS_FILE"
fi

print_area_results "$AREA"
