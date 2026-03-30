#!/usr/bin/env bash
# Tests for the immune response system:
#   trophallaxis-diagnose, trophallaxis-retry,
#   scar-add, scar-list, scar-check, immune-auto-scar

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

# ============================================================================
# Helper: Create isolated test environment with midden + immune support
# ============================================================================
setup_immune_env() {
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

    cat > "$tmpdir/.aether/data/COLONY_STATE.json" << 'EOF'
{
  "goal": "test immune response",
  "state": "active",
  "current_phase": 2,
  "plan": {"id": "test-plan", "tasks": []},
  "memory": {"instincts": []},
  "errors": {"records": []},
  "events": [],
  "session_id": "test-session",
  "initialized_at": "2026-02-13T16:00:00Z"
}
EOF

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
# trophallaxis-diagnose tests
# ============================================================================

# Test 1: Diagnose with matching midden entries — should find related failures
test_diagnose_with_matching_midden() {
    local tmpdir
    tmpdir=$(setup_immune_env)

    # Seed midden with a related failure
    run_cmd "$tmpdir" midden-write "build" "authentication module failed to compile" "builder" >/dev/null

    local result exit_code=0
    result=$(run_cmd "$tmpdir" trophallaxis-diagnose \
        --task-id "task_001" \
        --failure "authentication module compile error" \
        --phase 2) || exit_code=$?

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

    local related_failures
    related_failures=$(echo "$result" | jq -r '.result.related_failures')
    if [[ "$related_failures" -lt 1 ]]; then
        test_fail "related_failures >= 1" "related_failures=$related_failures"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 2: Diagnose with no midden entries — should return empty related_failures (0)
test_diagnose_no_midden_entries() {
    local tmpdir
    tmpdir=$(setup_immune_env)

    local result exit_code=0
    result=$(run_cmd "$tmpdir" trophallaxis-diagnose \
        --task-id "task_002" \
        --failure "totally novel failure never seen before xyzzy") || exit_code=$?

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

    local related_failures
    related_failures=$(echo "$result" | jq -r '.result.related_failures')
    if [[ "$related_failures" != "0" ]]; then
        test_fail "related_failures=0 when midden is empty" "related_failures=$related_failures"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 3: Diagnose returns a suggested_approach field
test_diagnose_returns_suggested_approach() {
    local tmpdir
    tmpdir=$(setup_immune_env)

    local result exit_code=0
    result=$(run_cmd "$tmpdir" trophallaxis-diagnose \
        --task-id "task_003" \
        --failure "dependency resolution failed") || exit_code=$?

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

    local suggested_approach
    suggested_approach=$(echo "$result" | jq -r '.result.suggested_approach')
    if [[ -z "$suggested_approach" || "$suggested_approach" == "null" ]]; then
        test_fail "suggested_approach is non-empty string" "suggested_approach=$suggested_approach"
        rm -rf "$tmpdir"
        return 1
    fi

    # Also verify diagnosis and confidence fields exist
    local diagnosis
    diagnosis=$(echo "$result" | jq -r '.result.diagnosis')
    if [[ -z "$diagnosis" || "$diagnosis" == "null" ]]; then
        test_fail "diagnosis field present" "diagnosis=$diagnosis"
        rm -rf "$tmpdir"
        return 1
    fi

    local confidence
    confidence=$(echo "$result" | jq -r '.result.confidence')
    if [[ -z "$confidence" || "$confidence" == "null" ]]; then
        test_fail "confidence field present" "confidence=$confidence"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# trophallaxis-retry tests
# ============================================================================

# Test 4: First retry — retry_count should be 1
test_retry_first() {
    local tmpdir
    tmpdir=$(setup_immune_env)

    # Build a minimal diagnosis JSON to pass in
    local diagnosis
    diagnosis='{"diagnosis":"try alternative approach","related_failures":0,"suggested_approach":"check dependencies","confidence":0.5}'

    local result exit_code=0
    result=$(run_cmd "$tmpdir" trophallaxis-retry \
        --task-id "task_004" \
        --diagnosis "$diagnosis") || exit_code=$?

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

    local retry_count
    retry_count=$(echo "$result" | jq -r '.result.retry_count')
    if [[ "$retry_count" != "1" ]]; then
        test_fail "retry_count=1 on first retry" "retry_count=$retry_count"
        rm -rf "$tmpdir"
        return 1
    fi

    local task_id
    task_id=$(echo "$result" | jq -r '.result.task_id')
    if [[ "$task_id" != "task_004" ]]; then
        test_fail "task_id=task_004" "task_id=$task_id"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 5: Second retry — retry_count should be 2
test_retry_second() {
    local tmpdir
    tmpdir=$(setup_immune_env)

    local diagnosis
    diagnosis='{"diagnosis":"alternative approach","related_failures":0,"suggested_approach":"rebuild from scratch","confidence":0.6}'

    # First retry
    run_cmd "$tmpdir" trophallaxis-retry \
        --task-id "task_005" \
        --diagnosis "$diagnosis" >/dev/null

    # Second retry
    local result exit_code=0
    result=$(run_cmd "$tmpdir" trophallaxis-retry \
        --task-id "task_005" \
        --diagnosis "$diagnosis") || exit_code=$?

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

    local retry_count
    retry_count=$(echo "$result" | jq -r '.result.retry_count')
    if [[ "$retry_count" != "2" ]]; then
        test_fail "retry_count=2 on second retry" "retry_count=$retry_count"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 6: Retry creates retry-log.json if missing
test_retry_creates_log_file() {
    local tmpdir
    tmpdir=$(setup_immune_env)

    # Confirm no retry-log.json yet
    if [[ -f "$tmpdir/.aether/data/retry-log.json" ]]; then
        rm -f "$tmpdir/.aether/data/retry-log.json"
    fi

    local diagnosis
    diagnosis='{"diagnosis":"test","related_failures":0,"suggested_approach":"retry","confidence":0.4}'

    run_cmd "$tmpdir" trophallaxis-retry \
        --task-id "task_006" \
        --diagnosis "$diagnosis" >/dev/null

    if [[ ! -f "$tmpdir/.aether/data/retry-log.json" ]]; then
        test_fail "retry-log.json created" "file does not exist at $tmpdir/.aether/data/retry-log.json"
        rm -rf "$tmpdir"
        return 1
    fi

    # Must be valid JSON
    if ! jq empty "$tmpdir/.aether/data/retry-log.json" 2>/dev/null; then
        test_fail "retry-log.json is valid JSON" "file contains invalid JSON"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 7: Retry sets diagnosis_injected=true
test_retry_diagnosis_injected() {
    local tmpdir
    tmpdir=$(setup_immune_env)

    local diagnosis
    diagnosis='{"diagnosis":"check imports","related_failures":1,"suggested_approach":"verify module paths","confidence":0.7}'

    local result exit_code=0
    result=$(run_cmd "$tmpdir" trophallaxis-retry \
        --task-id "task_007" \
        --diagnosis "$diagnosis") || exit_code=$?

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

    local diagnosis_injected
    diagnosis_injected=$(echo "$result" | jq -r '.result.diagnosis_injected')
    if [[ "$diagnosis_injected" != "true" ]]; then
        test_fail "diagnosis_injected=true" "diagnosis_injected=$diagnosis_injected"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# scar-add tests
# ============================================================================

# Test 8: Add a scar — creates scars.json and returns scar id
test_scar_add_creates_file() {
    local tmpdir
    tmpdir=$(setup_immune_env)

    local result exit_code=0
    result=$(run_cmd "$tmpdir" scar-add \
        --pattern "authentication module repeatedly fails to compile") || exit_code=$?

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

    local scar_id
    scar_id=$(echo "$result" | jq -r '.result.id')
    if [[ -z "$scar_id" || "$scar_id" == "null" ]]; then
        test_fail "id field present in result" "id=$scar_id"
        rm -rf "$tmpdir"
        return 1
    fi

    if [[ "$scar_id" != scar_* ]]; then
        test_fail "id starts with 'scar_'" "id=$scar_id"
        rm -rf "$tmpdir"
        return 1
    fi

    if [[ ! -f "$tmpdir/.aether/data/scars.json" ]]; then
        test_fail "scars.json created" "file does not exist"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 9: Add scar with all options — severity, phase, source
test_scar_add_all_options() {
    local tmpdir
    tmpdir=$(setup_immune_env)

    local result exit_code=0
    result=$(run_cmd "$tmpdir" scar-add \
        --pattern "database connection pool exhausted under load" \
        --severity high \
        --phase 3 \
        --source builder) || exit_code=$?

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

    local scar_id
    scar_id=$(echo "$result" | jq -r '.result.id')

    # Verify persisted data contains all fields
    local severity
    severity=$(jq --arg id "$scar_id" \
        '[.scars[] | select(.id == $id)] | first | .severity' \
        "$tmpdir/.aether/data/scars.json")
    if [[ "$severity" != '"high"' ]]; then
        test_fail "severity=high persisted" "severity=$severity"
        rm -rf "$tmpdir"
        return 1
    fi

    local phase
    phase=$(jq --arg id "$scar_id" \
        '[.scars[] | select(.id == $id)] | first | .phase' \
        "$tmpdir/.aether/data/scars.json")
    if [[ "$phase" != "3" ]]; then
        test_fail "phase=3 persisted" "phase=$phase"
        rm -rf "$tmpdir"
        return 1
    fi

    local source
    source=$(jq --arg id "$scar_id" \
        '[.scars[] | select(.id == $id)] | first | .source' \
        "$tmpdir/.aether/data/scars.json")
    if [[ "$source" != '"builder"' ]]; then
        test_fail "source=builder persisted" "source=$source"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 10: Multiple scars — scar_count increments
test_scar_add_increments_count() {
    local tmpdir
    tmpdir=$(setup_immune_env)

    run_cmd "$tmpdir" scar-add --pattern "first persistent failure pattern" >/dev/null

    local result exit_code=0
    result=$(run_cmd "$tmpdir" scar-add \
        --pattern "second persistent failure pattern") || exit_code=$?

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

    local scar_count
    scar_count=$(echo "$result" | jq -r '.result.scar_count')
    if [[ "$scar_count" != "2" ]]; then
        test_fail "scar_count=2 after second add" "scar_count=$scar_count"
        rm -rf "$tmpdir"
        return 1
    fi

    # Also verify via file
    local stored_count
    stored_count=$(jq '.scars | length' "$tmpdir/.aether/data/scars.json")
    if [[ "$stored_count" != "2" ]]; then
        test_fail "2 scars in scars.json" "stored_count=$stored_count"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# scar-list tests
# ============================================================================

# Test 11: List when empty — returns total=0, active=0
test_scar_list_empty() {
    local tmpdir
    tmpdir=$(setup_immune_env)

    local result exit_code=0
    result=$(run_cmd "$tmpdir" scar-list) || exit_code=$?

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

    local total
    total=$(echo "$result" | jq -r '.result.total')
    if [[ "$total" != "0" ]]; then
        test_fail "total=0 on empty list" "total=$total"
        rm -rf "$tmpdir"
        return 1
    fi

    local scars_len
    scars_len=$(echo "$result" | jq '.result.scars | length')
    if [[ "$scars_len" != "0" ]]; then
        test_fail "scars array empty" "length=$scars_len"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 12: List all scars
test_scar_list_all() {
    local tmpdir
    tmpdir=$(setup_immune_env)

    run_cmd "$tmpdir" scar-add --pattern "failure alpha" --severity low >/dev/null
    run_cmd "$tmpdir" scar-add --pattern "failure beta" --severity high >/dev/null
    run_cmd "$tmpdir" scar-add --pattern "failure gamma" --severity medium >/dev/null

    local result exit_code=0
    result=$(run_cmd "$tmpdir" scar-list) || exit_code=$?

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

    local total
    total=$(echo "$result" | jq -r '.result.total')
    if [[ "$total" != "3" ]]; then
        test_fail "total=3" "total=$total"
        rm -rf "$tmpdir"
        return 1
    fi

    local scars_len
    scars_len=$(echo "$result" | jq '.result.scars | length')
    if [[ "$scars_len" != "3" ]]; then
        test_fail "scars length=3" "length=$scars_len"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 13: List with --active filter
test_scar_list_active_filter() {
    local tmpdir
    tmpdir=$(setup_immune_env)

    run_cmd "$tmpdir" scar-add --pattern "active scar one" >/dev/null
    run_cmd "$tmpdir" scar-add --pattern "active scar two" >/dev/null

    local result exit_code=0
    result=$(run_cmd "$tmpdir" scar-list --active) || exit_code=$?

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

    local active
    active=$(echo "$result" | jq -r '.result.active')
    if [[ "$active" -lt 1 ]]; then
        test_fail "active >= 1 (newly added scars are active by default)" "active=$active"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 14: List with --severity filter
test_scar_list_severity_filter() {
    local tmpdir
    tmpdir=$(setup_immune_env)

    run_cmd "$tmpdir" scar-add --pattern "low severity thing" --severity low >/dev/null
    run_cmd "$tmpdir" scar-add --pattern "high severity issue" --severity high >/dev/null
    run_cmd "$tmpdir" scar-add --pattern "another high severity" --severity high >/dev/null

    local result exit_code=0
    result=$(run_cmd "$tmpdir" scar-list --severity high) || exit_code=$?

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

    local scars_len
    scars_len=$(echo "$result" | jq '.result.scars | length')
    if [[ "$scars_len" != "2" ]]; then
        test_fail "scars length=2 (only high severity)" "length=$scars_len"
        rm -rf "$tmpdir"
        return 1
    fi

    # All returned scars must have severity=high
    local non_high
    non_high=$(echo "$result" | jq '[.result.scars[] | select(.severity != "high")] | length')
    if [[ "$non_high" != "0" ]]; then
        test_fail "all listed scars have severity=high" "found $non_high non-high scars"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# scar-check tests
# ============================================================================

# Test 15: Check task matching a scar pattern — should return match
test_scar_check_matching() {
    local tmpdir
    tmpdir=$(setup_immune_env)

    run_cmd "$tmpdir" scar-add \
        --pattern "authentication module compile error" \
        --severity high >/dev/null

    local result exit_code=0
    result=$(run_cmd "$tmpdir" scar-check \
        --task "fix authentication module compile error in login service") || exit_code=$?

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

    local matches
    matches=$(echo "$result" | jq -r '.result.matches')
    if [[ "$matches" -lt 1 ]]; then
        test_fail "matches >= 1 for matching task description" "matches=$matches"
        rm -rf "$tmpdir"
        return 1
    fi

    local scars_len
    scars_len=$(echo "$result" | jq '.result.scars | length')
    if [[ "$scars_len" -lt 1 ]]; then
        test_fail "scars array has at least 1 entry" "length=$scars_len"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 16: Check task with no matching scars — should return matches=0
test_scar_check_no_match() {
    local tmpdir
    tmpdir=$(setup_immune_env)

    run_cmd "$tmpdir" scar-add \
        --pattern "database connection pool exhausted" \
        --severity medium >/dev/null

    local result exit_code=0
    result=$(run_cmd "$tmpdir" scar-check \
        --task "implement user profile avatar upload feature") || exit_code=$?

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

    local matches
    matches=$(echo "$result" | jq -r '.result.matches')
    if [[ "$matches" != "0" ]]; then
        test_fail "matches=0 for unrelated task" "matches=$matches"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# immune-auto-scar tests
# ============================================================================

# Test 17: Auto-scar when retry_count < 3 — should NOT scar
test_immune_auto_scar_below_threshold() {
    local tmpdir
    tmpdir=$(setup_immune_env)

    # Set up retry log with retry_count = 2 (below threshold)
    cat > "$tmpdir/.aether/data/retry-log.json" << 'EOF'
{
  "version": "1.0.0",
  "entries": [
    {
      "task_id": "task_below",
      "retry_count": 2,
      "last_failure": "build error occurred",
      "last_diagnosis": {"diagnosis":"check deps","suggested_approach":"reinstall","confidence":0.5},
      "updated_at": "2026-03-29T10:00:00Z"
    }
  ]
}
EOF

    local result exit_code=0
    result=$(run_cmd "$tmpdir" immune-auto-scar --task-id "task_below") || exit_code=$?

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

    local auto_scarred
    auto_scarred=$(echo "$result" | jq -r '.result.auto_scarred')
    if [[ "$auto_scarred" != "false" ]]; then
        test_fail "auto_scarred=false when retry_count < 3" "auto_scarred=$auto_scarred"
        rm -rf "$tmpdir"
        return 1
    fi

    local retry_count
    retry_count=$(echo "$result" | jq -r '.result.retry_count')
    if [[ "$retry_count" != "2" ]]; then
        test_fail "retry_count=2 reflected in result" "retry_count=$retry_count"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 18: Auto-scar when retry_count >= 3 — should auto-create scar
test_immune_auto_scar_at_threshold() {
    local tmpdir
    tmpdir=$(setup_immune_env)

    # Set up retry log with retry_count = 3 (at threshold)
    cat > "$tmpdir/.aether/data/retry-log.json" << 'EOF'
{
  "version": "1.0.0",
  "entries": [
    {
      "task_id": "task_at_threshold",
      "retry_count": 3,
      "last_failure": "persistent compile error in auth module",
      "last_diagnosis": {"diagnosis":"deep issue","suggested_approach":"rewrite module","confidence":0.8},
      "updated_at": "2026-03-29T10:00:00Z"
    }
  ]
}
EOF

    local result exit_code=0
    result=$(run_cmd "$tmpdir" immune-auto-scar --task-id "task_at_threshold") || exit_code=$?

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

    local auto_scarred
    auto_scarred=$(echo "$result" | jq -r '.result.auto_scarred')
    if [[ "$auto_scarred" != "true" ]]; then
        test_fail "auto_scarred=true when retry_count >= 3" "auto_scarred=$auto_scarred"
        rm -rf "$tmpdir"
        return 1
    fi

    local retry_count
    retry_count=$(echo "$result" | jq -r '.result.retry_count')
    if [[ "$retry_count" != "3" ]]; then
        test_fail "retry_count=3 reflected in result" "retry_count=$retry_count"
        rm -rf "$tmpdir"
        return 1
    fi

    # A scar should have been created
    if [[ ! -f "$tmpdir/.aether/data/scars.json" ]]; then
        test_fail "scars.json created by auto-scar" "file does not exist"
        rm -rf "$tmpdir"
        return 1
    fi

    local scar_count
    scar_count=$(jq '.scars | length' "$tmpdir/.aether/data/scars.json")
    if [[ "$scar_count" -lt 1 ]]; then
        test_fail "at least 1 scar in scars.json after auto-scar" "scar_count=$scar_count"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 19: Auto-scar with no retry log entry — should return auto_scarred=false gracefully
test_immune_auto_scar_no_entry() {
    local tmpdir
    tmpdir=$(setup_immune_env)

    local result exit_code=0
    result=$(run_cmd "$tmpdir" immune-auto-scar --task-id "task_unknown") || exit_code=$?

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

    local auto_scarred
    auto_scarred=$(echo "$result" | jq -r '.result.auto_scarred')
    if [[ "$auto_scarred" != "false" ]]; then
        test_fail "auto_scarred=false when no retry entry exists" "auto_scarred=$auto_scarred"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Run all tests
# ============================================================================

run_test test_diagnose_with_matching_midden   "trophallaxis-diagnose: finds related failures in midden"
run_test test_diagnose_no_midden_entries      "trophallaxis-diagnose: returns related_failures=0 when midden empty"
run_test test_diagnose_returns_suggested_approach "trophallaxis-diagnose: returns diagnosis, suggested_approach, confidence"
run_test test_retry_first                     "trophallaxis-retry: first retry returns retry_count=1"
run_test test_retry_second                    "trophallaxis-retry: second retry returns retry_count=2"
run_test test_retry_creates_log_file          "trophallaxis-retry: creates retry-log.json if missing"
run_test test_retry_diagnosis_injected        "trophallaxis-retry: sets diagnosis_injected=true"
run_test test_scar_add_creates_file           "scar-add: creates scars.json, id starts with 'scar_'"
run_test test_scar_add_all_options            "scar-add: persists severity, phase, source"
run_test test_scar_add_increments_count       "scar-add: scar_count increments with each add"
run_test test_scar_list_empty                 "scar-list: returns total=0 when no scars"
run_test test_scar_list_all                   "scar-list: returns all scars"
run_test test_scar_list_active_filter         "scar-list: --active returns only active scars"
run_test test_scar_list_severity_filter       "scar-list: --severity filters to matching severity"
run_test test_scar_check_matching             "scar-check: matching task returns matches >= 1"
run_test test_scar_check_no_match             "scar-check: unrelated task returns matches=0"
run_test test_immune_auto_scar_below_threshold "immune-auto-scar: retry_count < 3 does not scar"
run_test test_immune_auto_scar_at_threshold   "immune-auto-scar: retry_count >= 3 auto-creates scar"
run_test test_immune_auto_scar_no_entry       "immune-auto-scar: unknown task returns auto_scarred=false gracefully"

test_summary
