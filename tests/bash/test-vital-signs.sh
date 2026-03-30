#!/usr/bin/env bash
# Tests for colony-vital-signs subcommand

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

# ============================================================================
# Helper: Create isolated test environment
# ============================================================================
setup_vital_signs_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    mkdir -p "$tmpdir/.aether/data/midden"

    cp "$AETHER_UTILS" "$tmpdir/.aether/aether-utils.sh"
    chmod +x "$tmpdir/.aether/aether-utils.sh"

    local utils_source
    utils_source="$(dirname "$AETHER_UTILS")/utils"
    if [[ -d "$utils_source" ]]; then
        cp -r "$utils_source" "$tmpdir/.aether/"
    fi

    local exchange_source
    exchange_source="$(dirname "$AETHER_UTILS")/exchange"
    if [[ -d "$exchange_source" ]]; then
        cp -r "$exchange_source" "$tmpdir/.aether/"
    fi

    local schemas_source
    schemas_source="$(dirname "$AETHER_UTILS")/schemas"
    if [[ -d "$schemas_source" ]]; then
        cp -r "$schemas_source" "$tmpdir/.aether/"
    fi

    # Create minimal COLONY_STATE.json with a few events
    cat > "$tmpdir/.aether/data/COLONY_STATE.json" << 'EOF'
{
  "version": "3.0",
  "goal": "test vital signs",
  "state": "EXECUTING",
  "current_phase": 2,
  "session_id": "test-session",
  "initialized_at": "2026-01-01T00:00:00Z",
  "build_started_at": null,
  "plan": {
    "generated_at": null,
    "confidence": null,
    "phases": []
  },
  "memory": {
    "phase_learnings": "[]",
    "decisions": [],
    "instincts": "[]"
  },
  "errors": {"records": [], "flagged_patterns": []},
  "signals": [],
  "graveyards": [],
  "events": [
    "2026-01-02T10:00:00Z|phase_completed|build|Phase 1 done",
    "2026-01-03T12:00:00Z|phase_completed|build|Phase 2 done"
  ]
}
EOF

    # Empty pheromones
    cat > "$tmpdir/.aether/data/pheromones.json" << 'EOF'
{"version":"1.0.0","signals":[]}
EOF

    # Empty midden
    cat > "$tmpdir/.aether/data/midden/midden.json" << 'EOF'
{"version":"1.0.0","entries":[],"entry_count":0}
EOF

    echo "$tmpdir"
}

# Helper: run aether-utils against a test env
run_cmd() {
    local tmpdir="$1"
    shift
    AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$tmpdir/.aether/aether-utils.sh" "$@" 2>&1
}

# ============================================================================
# Test 1: Valid COLONY_STATE.json — should compute metrics
# ============================================================================
test_basic_metrics_computed() {
    local tmpdir
    tmpdir=$(setup_vital_signs_env)

    local result exit_code=0
    result=$(run_cmd "$tmpdir" colony-vital-signs) || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_json_valid "$result"; then
        test_fail "valid JSON" "invalid JSON: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    # All required fields present
    for field in build_velocity error_rate signal_health memory_pressure colony_age_hours overall_health; do
        local val
        val=$(echo "$result" | jq -r ".result.$field // \"MISSING\"")
        if [[ "$val" == "MISSING" ]]; then
            test_fail "field $field present" "field missing in: $result"
            rm -rf "$tmpdir"
            return 1
        fi
    done

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 2: Empty events — build_velocity should be 0
# ============================================================================
test_empty_events_velocity_zero() {
    local tmpdir
    tmpdir=$(setup_vital_signs_env)

    # Overwrite with no events
    jq '.events = []' "$tmpdir/.aether/data/COLONY_STATE.json" > "$tmpdir/.aether/data/COLONY_STATE.json.tmp" \
        && mv "$tmpdir/.aether/data/COLONY_STATE.json.tmp" "$tmpdir/.aether/data/COLONY_STATE.json"

    local result exit_code=0
    result=$(run_cmd "$tmpdir" colony-vital-signs) || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local velocity
    velocity=$(echo "$result" | jq -r '.result.build_velocity.phases_per_day')
    if [[ "$velocity" != "0" ]]; then
        test_fail "build_velocity.phases_per_day=0 with empty events" "phases_per_day=$velocity"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 3: No midden entries — error_rate should be 0
# ============================================================================
test_no_midden_error_rate_zero() {
    local tmpdir
    tmpdir=$(setup_vital_signs_env)

    # midden already empty from setup, verify
    local result exit_code=0
    result=$(run_cmd "$tmpdir" colony-vital-signs) || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local error_rate
    error_rate=$(echo "$result" | jq -r '.result.error_rate.errors_per_day')
    if [[ "$error_rate" != "0" ]]; then
        test_fail "error_rate.errors_per_day=0 with empty midden" "errors_per_day=$error_rate"
        rm -rf "$tmpdir"
        return 1
    fi

    local error_status
    error_status=$(echo "$result" | jq -r '.result.error_rate.status')
    if [[ "$error_status" != "clean" ]]; then
        test_fail "error_rate.status=clean with no errors" "error_rate.status=$error_status"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 4: No pheromones — signal_health should be 0 and signal_status "dormant"
# ============================================================================
test_no_pheromones_signal_dormant() {
    local tmpdir
    tmpdir=$(setup_vital_signs_env)

    # pheromones already empty from setup
    local result exit_code=0
    result=$(run_cmd "$tmpdir" colony-vital-signs) || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local signal_health
    signal_health=$(echo "$result" | jq -r '.result.signal_health.active_count')
    if [[ "$signal_health" != "0" ]]; then
        test_fail "signal_health.active_count=0 with no pheromones" "active_count=$signal_health"
        rm -rf "$tmpdir"
        return 1
    fi

    local signal_status
    signal_status=$(echo "$result" | jq -r '.result.signal_health.status')
    if [[ "$signal_status" != "dormant" ]]; then
        test_fail "signal_health.status=dormant with no pheromones" "signal_health.status=$signal_status"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 5: No instincts — memory_pressure should be 0, memory_status "empty"
# ============================================================================
test_no_instincts_memory_empty() {
    local tmpdir
    tmpdir=$(setup_vital_signs_env)

    # State has empty instincts from setup
    local result exit_code=0
    result=$(run_cmd "$tmpdir" colony-vital-signs) || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local memory_pressure
    memory_pressure=$(echo "$result" | jq -r '.result.memory_pressure.instinct_count')
    if [[ "$memory_pressure" != "0" ]]; then
        test_fail "memory_pressure.instinct_count=0 with no instincts" "instinct_count=$memory_pressure"
        rm -rf "$tmpdir"
        return 1
    fi

    local memory_status
    memory_status=$(echo "$result" | jq -r '.result.memory_pressure.status')
    if [[ "$memory_status" != "empty" ]]; then
        test_fail "memory_pressure.status=empty with no instincts" "memory_pressure.status=$memory_status"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 6: overall_health is between 0-100
# ============================================================================
test_overall_health_in_range() {
    local tmpdir
    tmpdir=$(setup_vital_signs_env)

    local result exit_code=0
    result=$(run_cmd "$tmpdir" colony-vital-signs) || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local score
    score=$(echo "$result" | jq -r '.result.overall_health')

    if [[ "$score" -lt 0 || "$score" -gt 100 ]]; then
        test_fail "overall_health in [0,100]" "overall_health=$score"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 7: Graceful degradation — missing COLONY_STATE.json
# ============================================================================
test_missing_state_file_graceful() {
    local tmpdir
    tmpdir=$(setup_vital_signs_env)

    rm -f "$tmpdir/.aether/data/COLONY_STATE.json"

    local result exit_code=0
    result=$(run_cmd "$tmpdir" colony-vital-signs) || exit_code=$?

    if ! assert_json_valid "$result"; then
        test_fail "valid JSON even with no COLONY_STATE.json" "invalid JSON: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true even with no COLONY_STATE.json" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Should return 0 for all numeric metrics
    local velocity
    velocity=$(echo "$result" | jq -r '.result.build_velocity.phases_per_day')
    if [[ "$velocity" != "0" ]]; then
        test_fail "build_velocity.phases_per_day=0 with missing state" "phases_per_day=$velocity"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 8: With active pheromone — signal_health reflects count, status "active"
# ============================================================================
test_active_pheromone_signal_active() {
    local tmpdir
    tmpdir=$(setup_vital_signs_env)

    cat > "$tmpdir/.aether/data/pheromones.json" << 'EOF'
{
  "version": "1.0.0",
  "signals": [
    {
      "id": "ph_001",
      "type": "FOCUS",
      "active": true,
      "content": {"text": "testing area"}
    }
  ]
}
EOF

    local result exit_code=0
    result=$(run_cmd "$tmpdir" colony-vital-signs) || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local signal_health
    signal_health=$(echo "$result" | jq -r '.result.signal_health.active_count')
    if [[ "$signal_health" != "1" ]]; then
        test_fail "signal_health.active_count=1 with one active pheromone" "active_count=$signal_health"
        rm -rf "$tmpdir"
        return 1
    fi

    local signal_status
    signal_status=$(echo "$result" | jq -r '.result.signal_health.status')
    if [[ "$signal_status" != "guided" ]]; then
        test_fail "signal_health.status=guided with one pheromone (1-3 = guided)" "signal_health.status=$signal_status"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 9: With instincts — memory_pressure and status reflect count
# ============================================================================
test_instincts_update_memory_pressure() {
    local tmpdir
    tmpdir=$(setup_vital_signs_env)

    # Write state with 3 instincts
    jq '.memory.instincts = [
      {"id":"i1","content":"test","confidence":0.8},
      {"id":"i2","content":"test2","confidence":0.7},
      {"id":"i3","content":"test3","confidence":0.9}
    ]' "$tmpdir/.aether/data/COLONY_STATE.json" > "$tmpdir/.aether/data/COLONY_STATE.json.tmp" \
        && mv "$tmpdir/.aether/data/COLONY_STATE.json.tmp" "$tmpdir/.aether/data/COLONY_STATE.json"

    local result exit_code=0
    result=$(run_cmd "$tmpdir" colony-vital-signs) || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local memory_pressure
    memory_pressure=$(echo "$result" | jq -r '.result.memory_pressure.instinct_count')
    if [[ "$memory_pressure" != "3" ]]; then
        test_fail "memory_pressure.instinct_count=3 with 3 instincts" "instinct_count=$memory_pressure"
        rm -rf "$tmpdir"
        return 1
    fi

    local memory_status
    memory_status=$(echo "$result" | jq -r '.result.memory_pressure.status')
    if [[ "$memory_status" != "growing" ]]; then
        test_fail "memory_pressure.status=growing with 3 instincts" "memory_pressure.status=$memory_status"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 10: Elevated errors increase error_rate and lower overall_health
# ============================================================================
test_errors_elevate_error_rate() {
    local tmpdir
    tmpdir=$(setup_vital_signs_env)

    # Use a recent timestamp (within 24h) for entries to pass the time-window filter
    local recent_ts
    recent_ts=$(date -u '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || echo "2099-01-01T00:00:00Z")

    # Write midden with 5 unreviewed entries using recent timestamps
    # Implementation checks .reviewed field (not .acknowledged)
    cat > "$tmpdir/.aether/data/midden/midden.json" << EOF
{
  "version": "1.0.0",
  "entries": [
    {"id":"e1","category":"general","message":"fail1","source":"test","timestamp":"$recent_ts","reviewed":false},
    {"id":"e2","category":"general","message":"fail2","source":"test","timestamp":"$recent_ts","reviewed":false},
    {"id":"e3","category":"general","message":"fail3","source":"test","timestamp":"$recent_ts","reviewed":false},
    {"id":"e4","category":"general","message":"fail4","source":"test","timestamp":"$recent_ts","reviewed":false},
    {"id":"e5","category":"general","message":"fail5","source":"test","timestamp":"$recent_ts","reviewed":false}
  ],
  "entry_count": 5
}
EOF

    local result exit_code=0
    result=$(run_cmd "$tmpdir" colony-vital-signs) || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local error_rate
    error_rate=$(echo "$result" | jq -r '.result.error_rate.errors_per_day')
    if [[ "$error_rate" != "5" ]]; then
        test_fail "error_rate.errors_per_day=5 with 5 unacknowledged entries" "errors_per_day=$error_rate"
        rm -rf "$tmpdir"
        return 1
    fi

    local error_status
    error_status=$(echo "$result" | jq -r '.result.error_rate.status')
    if [[ "$error_status" != "elevated" ]]; then
        test_fail "error_rate.status=elevated with 5 errors" "error_rate.status=$error_status"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify score is lower due to errors (no error bonus: 30vel + 0err + 0sig + 0mem = 30)
    local score
    score=$(echo "$result" | jq -r '.result.overall_health')
    if [[ "$score" -gt 50 ]]; then
        test_fail "overall_health reduced by 5 errors (should be <=50)" "overall_health=$score"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 11: Graceful when midden.json is missing (not just empty)
# ============================================================================
test_missing_midden_file_graceful() {
    local tmpdir
    tmpdir=$(setup_vital_signs_env)

    rm -f "$tmpdir/.aether/data/midden/midden.json"

    local result exit_code=0
    result=$(run_cmd "$tmpdir" colony-vital-signs) || exit_code=$?

    if ! assert_json_valid "$result"; then
        test_fail "valid JSON with missing midden file" "invalid JSON: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true with missing midden file" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local error_rate
    error_rate=$(echo "$result" | jq -r '.result.error_rate.errors_per_day')
    if [[ "$error_rate" != "0" ]]; then
        test_fail "error_rate.errors_per_day=0 with missing midden file" "errors_per_day=$error_rate"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 12: Graceful when pheromones.json is missing
# ============================================================================
test_missing_pheromones_file_graceful() {
    local tmpdir
    tmpdir=$(setup_vital_signs_env)

    rm -f "$tmpdir/.aether/data/pheromones.json"

    local result exit_code=0
    result=$(run_cmd "$tmpdir" colony-vital-signs) || exit_code=$?

    if ! assert_json_valid "$result"; then
        test_fail "valid JSON with missing pheromones file" "invalid JSON: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true with missing pheromones file" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local signal_health
    signal_health=$(echo "$result" | jq -r '.result.signal_health.active_count')
    if [[ "$signal_health" != "0" ]]; then
        test_fail "signal_health.active_count=0 with missing pheromones file" "active_count=$signal_health"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Run all tests
# ============================================================================

run_test test_basic_metrics_computed            "colony-vital-signs: computes all required metrics with valid state"
run_test test_empty_events_velocity_zero        "colony-vital-signs: build_velocity.phases_per_day=0 with empty events"
run_test test_no_midden_error_rate_zero         "colony-vital-signs: error_rate.errors_per_day=0 and status=clean with empty midden"
run_test test_no_pheromones_signal_dormant      "colony-vital-signs: signal_health.active_count=0 and status=dormant with no pheromones"
run_test test_no_instincts_memory_empty         "colony-vital-signs: memory_pressure.instinct_count=0 and status=empty with no instincts"
run_test test_overall_health_in_range           "colony-vital-signs: overall_health is in range [0,100]"
run_test test_missing_state_file_graceful       "colony-vital-signs: graceful when COLONY_STATE.json missing"
run_test test_active_pheromone_signal_active    "colony-vital-signs: signal_health.active_count=1 and status=guided with one pheromone"
run_test test_instincts_update_memory_pressure  "colony-vital-signs: memory_pressure.instinct_count=3 and status=growing with 3 instincts"
run_test test_errors_elevate_error_rate         "colony-vital-signs: error_rate.errors_per_day=5 and status=elevated with 5 recent midden entries"
run_test test_missing_midden_file_graceful      "colony-vital-signs: graceful when midden.json missing"
run_test test_missing_pheromones_file_graceful  "colony-vital-signs: graceful when pheromones.json missing"

test_summary
