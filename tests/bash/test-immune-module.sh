#!/usr/bin/env bash
# Tests for immune system module — trophallaxis-diagnose, trophallaxis-retry,
# scar-add, scar-list, scar-check, immune-auto-scar

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

# ============================================================================
# Helper: Create isolated test environment
# ============================================================================
setup_immune_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    mkdir -p "$tmpdir/.aether/data/midden"
    mkdir -p "$tmpdir/.aether/data/immune"

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

    # Minimal COLONY_STATE.json with a colony_name
    cat > "$tmpdir/.aether/data/COLONY_STATE.json" << 'EOF'
{
  "colony_name": "test-immune",
  "goal": "test immune module",
  "state": "active",
  "current_phase": 1,
  "plan": {"id": "test-plan", "tasks": []},
  "memory": {"instincts": []},
  "errors": {"records": []},
  "events": [],
  "session_id": "test-session",
  "initialized_at": "2026-01-01T00:00:00Z"
}
EOF

    # Minimal midden
    cat > "$tmpdir/.aether/data/midden/midden.json" << 'EOF'
{"version":"1.0.0","entries":[],"entry_count":0}
EOF

    echo "$tmpdir"
}

run_cmd() {
    local tmpdir="$1"
    shift
    local exit_code=0
    local output
    output=$(AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$tmpdir/.aether/aether-utils.sh" "$@" 2>&1) || exit_code=$?
    echo "$output"
    return 0
}

# ============================================================================
# TEST: trophallaxis-diagnose — basic diagnosis with no midden data
# ============================================================================
test_diagnose_no_midden() {
    local tmpdir
    tmpdir=$(setup_immune_env)
    local result
    result=$(run_cmd "$tmpdir" trophallaxis-diagnose --task-id "task_001" --failure "npm test failed")
    rm -rf "$tmpdir"

    if ! assert_ok_true "$result"; then
        test_fail '{"ok":true}' "$result"
        return 1
    fi
    if ! echo "$result" | jq -e '.result | has("diagnosis") and has("related_failures") and has("suggested_approach") and has("confidence")' >/dev/null 2>&1; then
        test_fail "result has diagnosis, related_failures, suggested_approach, confidence" "$result"
        return 1
    fi
    return 0
}

# ============================================================================
# TEST: trophallaxis-diagnose — finds related midden entries by keyword
# ============================================================================
test_diagnose_finds_related() {
    local tmpdir
    tmpdir=$(setup_immune_env)
    run_cmd "$tmpdir" midden-write "build" "npm test failed with exit code 1" "builder" >/dev/null
    run_cmd "$tmpdir" midden-write "build" "npm test timeout error" "builder" >/dev/null
    run_cmd "$tmpdir" midden-write "security" "CVE found in dependency" "gatekeeper" >/dev/null
    local result
    result=$(run_cmd "$tmpdir" trophallaxis-diagnose --task-id "task_002" --failure "npm test failed")
    local related
    related=$(echo "$result" | jq -r '.result.related_failures' 2>/dev/null || echo "0")
    rm -rf "$tmpdir"

    if ! assert_ok_true "$result"; then
        test_fail '{"ok":true}' "$result"
        return 1
    fi
    if [[ "$related" -lt 2 ]]; then
        test_fail "related_failures >= 2" "related=$related"
        return 1
    fi
    return 0
}

# ============================================================================
# TEST: trophallaxis-diagnose — requires --task-id
# ============================================================================
test_diagnose_requires_task_id() {
    local tmpdir
    tmpdir=$(setup_immune_env)
    local result
    result=$(run_cmd "$tmpdir" trophallaxis-diagnose --failure "some error")
    rm -rf "$tmpdir"

    if echo "$result" | jq -e '.ok == false' >/dev/null 2>&1; then
        return 0
    fi
    test_fail '{"ok":false}' "$result"
    return 1
}

# ============================================================================
# TEST: trophallaxis-retry — records retry attempt
# ============================================================================
test_retry_records_attempt() {
    local tmpdir
    tmpdir=$(setup_immune_env)
    local diag
    diag='{"diagnosis":"test diag","related_failures":0,"suggested_approach":"try again","confidence":0.5}'
    local result
    result=$(run_cmd "$tmpdir" trophallaxis-retry --task-id "task_001" --diagnosis "$diag")
    rm -rf "$tmpdir"

    if ! assert_ok_true "$result"; then
        test_fail '{"ok":true}' "$result"
        return 1
    fi
    local retry_count injected task_id
    retry_count=$(echo "$result" | jq -r '.result.retry_count' 2>/dev/null || echo "-1")
    injected=$(echo "$result" | jq -r '.result.diagnosis_injected' 2>/dev/null || echo "false")
    task_id=$(echo "$result" | jq -r '.result.task_id' 2>/dev/null || echo "")
    if [[ "$retry_count" -lt 1 || "$injected" != "true" || "$task_id" != "task_001" ]]; then
        test_fail "retry_count>=1, diagnosis_injected=true, task_id=task_001" "retry_count=$retry_count, injected=$injected, task_id=$task_id"
        return 1
    fi
    return 0
}

# ============================================================================
# TEST: trophallaxis-retry — increments retry_count on subsequent retries
# ============================================================================
test_retry_increments_count() {
    local tmpdir
    tmpdir=$(setup_immune_env)
    local diag
    diag='{"diagnosis":"test","related_failures":0,"suggested_approach":"try","confidence":0.5}'
    run_cmd "$tmpdir" trophallaxis-retry --task-id "task_multi" --diagnosis "$diag" >/dev/null
    run_cmd "$tmpdir" trophallaxis-retry --task-id "task_multi" --diagnosis "$diag" >/dev/null
    local result
    result=$(run_cmd "$tmpdir" trophallaxis-retry --task-id "task_multi" --diagnosis "$diag")
    local retry_count
    retry_count=$(echo "$result" | jq -r '.result.retry_count' 2>/dev/null || echo "0")
    rm -rf "$tmpdir"

    if ! assert_ok_true "$result"; then
        test_fail '{"ok":true}' "$result"
        return 1
    fi
    if [[ "$retry_count" -ne 3 ]]; then
        test_fail "retry_count=3" "retry_count=$retry_count"
        return 1
    fi
    return 0
}

# ============================================================================
# TEST: trophallaxis-retry — writes to immune/retry-log.json
# ============================================================================
test_retry_writes_log() {
    local tmpdir
    tmpdir=$(setup_immune_env)
    local diag
    diag='{"diagnosis":"check","related_failures":1,"suggested_approach":"fix dep","confidence":0.7}'
    run_cmd "$tmpdir" trophallaxis-retry --task-id "task_log" --diagnosis "$diag" >/dev/null
    local log_file
    log_file=$(find "$tmpdir/.aether/data" -name "retry-log.json" 2>/dev/null | head -1)
    rm -rf "$tmpdir"

    if [[ -n "$log_file" ]]; then
        return 0
    fi
    test_fail "retry-log.json to exist in colony data directory" "file not found"
    return 1
}

# ============================================================================
# TEST: scar-add — creates a scar entry
# ============================================================================
test_scar_add_creates_entry() {
    local tmpdir
    tmpdir=$(setup_immune_env)
    local result
    result=$(run_cmd "$tmpdir" scar-add --pattern "npm install hangs on CI" --severity "high")
    rm -rf "$tmpdir"

    if ! assert_ok_true "$result"; then
        test_fail '{"ok":true}' "$result"
        return 1
    fi
    local scar_id scar_count
    scar_id=$(echo "$result" | jq -r '.result.id' 2>/dev/null || echo "")
    scar_count=$(echo "$result" | jq -r '.result.scar_count' 2>/dev/null || echo "0")
    if [[ "$scar_id" != scar_* || "$scar_count" -lt 1 ]]; then
        test_fail "id=scar_*, scar_count>=1" "id=$scar_id, count=$scar_count"
        return 1
    fi
    return 0
}

# ============================================================================
# TEST: scar-add — requires --pattern
# ============================================================================
test_scar_add_requires_pattern() {
    local tmpdir
    tmpdir=$(setup_immune_env)
    local result
    result=$(run_cmd "$tmpdir" scar-add --severity "low")
    rm -rf "$tmpdir"

    if echo "$result" | jq -e '.ok == false' >/dev/null 2>&1; then
        return 0
    fi
    test_fail '{"ok":false}' "$result"
    return 1
}

# ============================================================================
# TEST: scar-add — validates severity
# ============================================================================
test_scar_add_validates_severity() {
    local tmpdir
    tmpdir=$(setup_immune_env)
    local result
    result=$(run_cmd "$tmpdir" scar-add --pattern "test pattern" --severity "critical")
    rm -rf "$tmpdir"

    if echo "$result" | jq -e '.ok == false' >/dev/null 2>&1; then
        return 0
    fi
    test_fail '{"ok":false}' "$result"
    return 1
}

# ============================================================================
# TEST: scar-list — lists all scars
# ============================================================================
test_scar_list_all() {
    local tmpdir
    tmpdir=$(setup_immune_env)
    run_cmd "$tmpdir" scar-add --pattern "pattern one" --severity "low" >/dev/null
    run_cmd "$tmpdir" scar-add --pattern "pattern two" --severity "medium" >/dev/null
    local result
    result=$(run_cmd "$tmpdir" scar-list)
    local total
    total=$(echo "$result" | jq -r '.result.total' 2>/dev/null || echo "0")
    rm -rf "$tmpdir"

    if ! assert_ok_true "$result"; then
        test_fail '{"ok":true}' "$result"
        return 1
    fi
    if [[ "$total" -ne 2 ]]; then
        test_fail "total=2" "total=$total"
        return 1
    fi
    return 0
}

# ============================================================================
# TEST: scar-list --active filters inactive scars
# ============================================================================
test_scar_list_active() {
    local tmpdir
    tmpdir=$(setup_immune_env)
    run_cmd "$tmpdir" scar-add --pattern "active scar" --severity "low" >/dev/null
    run_cmd "$tmpdir" scar-add --pattern "another scar" --severity "medium" >/dev/null
    local result
    result=$(run_cmd "$tmpdir" scar-list --active)
    local active
    active=$(echo "$result" | jq -r '.result.active' 2>/dev/null || echo "0")
    rm -rf "$tmpdir"

    if ! assert_ok_true "$result"; then
        test_fail '{"ok":true}' "$result"
        return 1
    fi
    if [[ "$active" -lt 1 ]]; then
        test_fail "active>=1" "active=$active"
        return 1
    fi
    return 0
}

# ============================================================================
# TEST: scar-list --severity filters by severity
# ============================================================================
test_scar_list_severity_filter() {
    local tmpdir
    tmpdir=$(setup_immune_env)
    run_cmd "$tmpdir" scar-add --pattern "low scar" --severity "low" >/dev/null
    run_cmd "$tmpdir" scar-add --pattern "high scar" --severity "high" >/dev/null
    local result
    result=$(run_cmd "$tmpdir" scar-list --severity "high")
    local total
    total=$(echo "$result" | jq -r '.result.total' 2>/dev/null || echo "0")
    rm -rf "$tmpdir"

    if ! assert_ok_true "$result"; then
        test_fail '{"ok":true}' "$result"
        return 1
    fi
    if [[ "$total" -ne 1 ]]; then
        test_fail "total=1 (only high severity)" "total=$total"
        return 1
    fi
    return 0
}

# ============================================================================
# TEST: scar-list — empty when no scars
# ============================================================================
test_scar_list_empty() {
    local tmpdir
    tmpdir=$(setup_immune_env)
    local result
    result=$(run_cmd "$tmpdir" scar-list)
    local total
    total=$(echo "$result" | jq -r '.result.total' 2>/dev/null || echo "-1")
    rm -rf "$tmpdir"

    if ! assert_ok_true "$result"; then
        test_fail '{"ok":true}' "$result"
        return 1
    fi
    if [[ "$total" -ne 0 ]]; then
        test_fail "total=0" "total=$total"
        return 1
    fi
    return 0
}

# ============================================================================
# TEST: scar-check — matches task against active scars
# ============================================================================
test_scar_check_matches() {
    local tmpdir
    tmpdir=$(setup_immune_env)
    run_cmd "$tmpdir" scar-add --pattern "npm install hangs" --severity "high" >/dev/null
    run_cmd "$tmpdir" scar-add --pattern "database connection timeout" --severity "medium" >/dev/null
    local result
    result=$(run_cmd "$tmpdir" scar-check --task "Run npm install and build dependencies")
    local matches
    matches=$(echo "$result" | jq -r '.result.matches' 2>/dev/null || echo "0")
    rm -rf "$tmpdir"

    if ! assert_ok_true "$result"; then
        test_fail '{"ok":true}' "$result"
        return 1
    fi
    if [[ "$matches" -lt 1 ]]; then
        test_fail "matches>=1" "matches=$matches"
        return 1
    fi
    return 0
}

# ============================================================================
# TEST: scar-check — returns zero matches when no overlap
# ============================================================================
test_scar_check_no_match() {
    local tmpdir
    tmpdir=$(setup_immune_env)
    run_cmd "$tmpdir" scar-add --pattern "redis connection refused" --severity "medium" >/dev/null
    local result
    result=$(run_cmd "$tmpdir" scar-check --task "Write unit tests for auth module")
    local matches
    matches=$(echo "$result" | jq -r '.result.matches' 2>/dev/null || echo "-1")
    rm -rf "$tmpdir"

    if ! assert_ok_true "$result"; then
        test_fail '{"ok":true}' "$result"
        return 1
    fi
    if [[ "$matches" -ne 0 ]]; then
        test_fail "matches=0" "matches=$matches"
        return 1
    fi
    return 0
}

# ============================================================================
# TEST: scar-check — requires --task
# ============================================================================
test_scar_check_requires_task() {
    local tmpdir
    tmpdir=$(setup_immune_env)
    local result
    result=$(run_cmd "$tmpdir" scar-check)
    rm -rf "$tmpdir"

    if echo "$result" | jq -e '.ok == false' >/dev/null 2>&1; then
        return 0
    fi
    test_fail '{"ok":false}' "$result"
    return 1
}

# ============================================================================
# TEST: immune-auto-scar — does not scar when retry_count < 3
# ============================================================================
test_auto_scar_below_threshold() {
    local tmpdir
    tmpdir=$(setup_immune_env)
    local diag
    diag='{"diagnosis":"low retry","related_failures":0,"suggested_approach":"check logs","confidence":0.4}'
    run_cmd "$tmpdir" trophallaxis-retry --task-id "task_low" --diagnosis "$diag" >/dev/null
    run_cmd "$tmpdir" trophallaxis-retry --task-id "task_low" --diagnosis "$diag" >/dev/null
    local result
    result=$(run_cmd "$tmpdir" immune-auto-scar --task-id "task_low")
    local auto_scarred
    auto_scarred=$(echo "$result" | jq -r '.result.auto_scarred' 2>/dev/null || echo "true")
    rm -rf "$tmpdir"

    if ! assert_ok_true "$result"; then
        test_fail '{"ok":true}' "$result"
        return 1
    fi
    if [[ "$auto_scarred" != "false" ]]; then
        test_fail "auto_scarred=false" "auto_scarred=$auto_scarred"
        return 1
    fi
    return 0
}

# ============================================================================
# TEST: immune-auto-scar — auto-scars when retry_count >= 3
# ============================================================================
test_auto_scar_at_threshold() {
    local tmpdir
    tmpdir=$(setup_immune_env)
    local diag
    diag='{"diagnosis":"persistent failure","related_failures":2,"suggested_approach":"escalate","confidence":0.8}'
    run_cmd "$tmpdir" trophallaxis-retry --task-id "task_hot" --diagnosis "$diag" >/dev/null
    run_cmd "$tmpdir" trophallaxis-retry --task-id "task_hot" --diagnosis "$diag" >/dev/null
    run_cmd "$tmpdir" trophallaxis-retry --task-id "task_hot" --diagnosis "$diag" >/dev/null
    local result
    result=$(run_cmd "$tmpdir" immune-auto-scar --task-id "task_hot")
    local auto_scarred retry_count
    auto_scarred=$(echo "$result" | jq -r '.result.auto_scarred' 2>/dev/null || echo "false")
    retry_count=$(echo "$result" | jq -r '.result.retry_count' 2>/dev/null || echo "0")
    rm -rf "$tmpdir"

    if ! assert_ok_true "$result"; then
        test_fail '{"ok":true}' "$result"
        return 1
    fi
    if [[ "$auto_scarred" != "true" || "$retry_count" -ne 3 ]]; then
        test_fail "auto_scarred=true, retry_count=3" "auto_scarred=$auto_scarred, retry_count=$retry_count"
        return 1
    fi
    return 0
}

# ============================================================================
# TEST: immune-auto-scar — requires --task-id
# ============================================================================
test_auto_scar_requires_task_id() {
    local tmpdir
    tmpdir=$(setup_immune_env)
    local result
    result=$(run_cmd "$tmpdir" immune-auto-scar)
    rm -rf "$tmpdir"

    if echo "$result" | jq -e '.ok == false' >/dev/null 2>&1; then
        return 0
    fi
    test_fail '{"ok":false}' "$result"
    return 1
}

# ============================================================================
# TEST: immune-auto-scar — graceful when no retry log exists
# ============================================================================
test_auto_scar_no_log() {
    local tmpdir
    tmpdir=$(setup_immune_env)
    local result
    result=$(run_cmd "$tmpdir" immune-auto-scar --task-id "task_unknown")
    local auto_scarred
    auto_scarred=$(echo "$result" | jq -r '.result.auto_scarred' 2>/dev/null || echo "true")
    rm -rf "$tmpdir"

    if ! assert_ok_true "$result"; then
        test_fail '{"ok":true}' "$result"
        return 1
    fi
    if [[ "$auto_scarred" != "false" ]]; then
        test_fail "auto_scarred=false" "auto_scarred=$auto_scarred"
        return 1
    fi
    return 0
}

# ============================================================================
# Run all tests
# ============================================================================

run_test test_diagnose_no_midden              "trophallaxis-diagnose: ok:true with no midden data, has required fields"
run_test test_diagnose_finds_related          "trophallaxis-diagnose: finds related midden entries by keyword"
run_test test_diagnose_requires_task_id       "trophallaxis-diagnose: requires --task-id"
run_test test_retry_records_attempt           "trophallaxis-retry: records retry attempt with correct fields"
run_test test_retry_increments_count          "trophallaxis-retry: increments retry_count on repeated calls"
run_test test_retry_writes_log                "trophallaxis-retry: writes retry-log.json to colony data directory"
run_test test_scar_add_creates_entry          "scar-add: creates scar entry with id and scar_count"
run_test test_scar_add_requires_pattern       "scar-add: requires --pattern argument"
run_test test_scar_add_validates_severity     "scar-add: validates severity is low|medium|high"
run_test test_scar_list_all                   "scar-list: returns all scars with correct total"
run_test test_scar_list_active                "scar-list --active: only returns active scars"
run_test test_scar_list_severity_filter       "scar-list --severity: filters by severity level"
run_test test_scar_list_empty                 "scar-list: returns total=0 when no scars exist"
run_test test_scar_check_matches              "scar-check: matches task description against active scars"
run_test test_scar_check_no_match             "scar-check: returns matches=0 when no overlap"
run_test test_scar_check_requires_task        "scar-check: requires --task argument"
run_test test_auto_scar_below_threshold       "immune-auto-scar: auto_scarred=false when retry_count < 3"
run_test test_auto_scar_at_threshold          "immune-auto-scar: auto_scarred=true when retry_count >= 3"
run_test test_auto_scar_requires_task_id      "immune-auto-scar: requires --task-id"
run_test test_auto_scar_no_log                "immune-auto-scar: graceful when no retry log exists"

test_summary
