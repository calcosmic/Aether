#!/usr/bin/env bash
# Curation Orchestrator Tests
# Tests curation-run via aether-utils.sh: executes all 8 ants in order,
# respects --dry-run, handles missing files gracefully, reports correct counts.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

if [[ ! -f "$AETHER_UTILS" ]]; then
    log_error "aether-utils.sh not found at: $AETHER_UTILS"
    exit 1
fi

# ============================================================================
# Helper: isolated env with aether-utils.sh + all utils
# ============================================================================
setup_orch_env() {
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

    # Minimal required data files
    cat > "$tmpdir/.aether/data/COLONY_STATE.json" <<'EOF'
{
  "version": "3.0",
  "goal": "Test orchestrator",
  "state": "READY",
  "current_phase": 1,
  "session_id": "test-orch",
  "initialized_at": "2026-01-01T00:00:00Z",
  "build_started_at": null,
  "plan": { "phases": [{ "id": 1, "name": "Test Phase", "status": "pending" }] },
  "memory": { "phase_learnings": [], "decisions": [], "instincts": [] },
  "errors": { "records": [], "flagged_patterns": [] },
  "events": [],
  "signals": [],
  "graveyards": []
}
EOF

    cat > "$tmpdir/.aether/data/pheromones.json" <<'EOF'
{"version":"2.0","signals":[]}
EOF

    echo "$tmpdir"
}

run_cmd() {
    local tmpdir="$1"
    shift
    AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        COLONY_DATA_DIR="$tmpdir/.aether/data" \
        bash "$tmpdir/.aether/aether-utils.sh" "$@" 2>/dev/null || true
}

# ============================================================================
# Test 1: curation-run executes all 8 steps in order
# ============================================================================
test_curation_run_all_steps() {
    local tmpdir
    tmpdir=$(setup_orch_env)

    local result
    result=$(run_cmd "$tmpdir" curation-run)

    rm -rf "$tmpdir"

    # Must be ok
    assert_ok_true "$result" || { log_error "curation-run did not return ok: $result"; return 1; }

    # Must have exactly 8 steps
    local total
    total=$(echo "$result" | jq '.result.total_steps')
    [[ "$total" -eq 8 ]] || { log_error "total_steps should be 8, got: $total"; return 1; }

    # Steps array must have 8 entries
    local step_count
    step_count=$(echo "$result" | jq '.result.steps | length')
    [[ "$step_count" -eq 8 ]] || { log_error "steps length should be 8, got: $step_count"; return 1; }

    # Sentinel must be first
    local first_name
    first_name=$(echo "$result" | jq -r '.result.steps[0].name')
    [[ "$first_name" == "sentinel" ]] || { log_error "first step should be sentinel, got: $first_name"; return 1; }

    # Scribe must be last
    local last_name
    last_name=$(echo "$result" | jq -r '.result.steps[7].name')
    [[ "$last_name" == "scribe" ]] || { log_error "last step should be scribe, got: $last_name"; return 1; }

    # All expected names must appear
    local names
    names=$(echo "$result" | jq -r '.result.steps[].name' | tr '\n' ' ')
    for expected_name in sentinel nurse critic herald janitor archivist librarian scribe; do
        assert_contains "$names" "$expected_name" || { log_error "step '$expected_name' missing from: $names"; return 1; }
    done

    # succeeded + failed must equal 8
    local succeeded failed sum
    succeeded=$(echo "$result" | jq '.result.succeeded')
    failed=$(echo "$result" | jq '.result.failed')
    sum=$(( succeeded + failed ))
    [[ "$sum" -eq 8 ]] || { log_error "succeeded+failed should be 8, got: $sum"; return 1; }

    # duration_ms must be present and >= 0
    local dur
    dur=$(echo "$result" | jq '.result.duration_ms')
    [[ "$dur" -ge 0 ]] || { log_error "duration_ms should be >= 0, got: $dur"; return 1; }
}

# ============================================================================
# Test 2: curation-run --dry-run doesn't modify files
# ============================================================================
test_curation_run_dry_run() {
    local tmpdir
    tmpdir=$(setup_orch_env)

    # Seed a minimal instincts.json so ants have something to work with
    cat > "$tmpdir/.aether/data/instincts.json" <<'EOF'
{
  "instincts": [
    {
      "id": "inst-dry-001",
      "trigger": "always use dry-run in tests",
      "action": "use --dry-run flag",
      "confidence": 0.8,
      "trust_score": 0.8,
      "trust_tier": "trusted",
      "archived": false,
      "provenance": {
        "created_at": "2026-01-01T00:00:00Z",
        "source": "test"
      }
    }
  ]
}
EOF

    # Capture checksums before the run (support both macOS md5 and Linux md5sum)
    _checksum() {
        md5 -q "$1" 2>/dev/null || md5sum "$1" 2>/dev/null | awk '{print $1}'
    }
    local inst_before pheromones_before
    inst_before=$(_checksum "$tmpdir/.aether/data/instincts.json")
    pheromones_before=$(_checksum "$tmpdir/.aether/data/pheromones.json")

    local result
    result=$(run_cmd "$tmpdir" curation-run --dry-run)

    local inst_after pheromones_after
    inst_after=$(_checksum "$tmpdir/.aether/data/instincts.json")
    pheromones_after=$(_checksum "$tmpdir/.aether/data/pheromones.json")

    rm -rf "$tmpdir"

    # Must be ok
    assert_ok_true "$result" || { log_error "curation-run --dry-run did not return ok: $result"; return 1; }

    # dry_run flag must be true in result
    local dry_flag
    dry_flag=$(echo "$result" | jq '.result.dry_run')
    [[ "$dry_flag" == "true" ]] || { log_error "result.dry_run should be true, got: $dry_flag"; return 1; }

    # instincts.json must not have changed
    [[ "$inst_after" == "$inst_before" ]] || { log_error "instincts.json changed during dry-run"; return 1; }

    # pheromones.json must not have changed
    [[ "$pheromones_after" == "$pheromones_before" ]] || { log_error "pheromones.json changed during dry-run"; return 1; }
}

# ============================================================================
# Test 3: curation-run handles missing instincts.json gracefully
# ============================================================================
test_curation_run_missing_instincts() {
    local tmpdir
    tmpdir=$(setup_orch_env)

    # Deliberately do NOT create instincts.json
    rm -f "$tmpdir/.aether/data/instincts.json"

    local result
    result=$(run_cmd "$tmpdir" curation-run)

    rm -rf "$tmpdir"

    # Must still return ok (graceful degradation)
    assert_ok_true "$result" || { log_error "curation-run failed without instincts.json: $result"; return 1; }

    # Must still have 8 steps
    local total
    total=$(echo "$result" | jq '.result.total_steps')
    [[ "$total" -eq 8 ]] || { log_error "total_steps should still be 8 without instincts.json, got: $total"; return 1; }
}

# ============================================================================
# Test 4: curation-run reports correct step counts
# ============================================================================
test_curation_run_step_counts() {
    local tmpdir
    tmpdir=$(setup_orch_env)

    local result
    result=$(run_cmd "$tmpdir" curation-run)

    rm -rf "$tmpdir"

    assert_ok_true "$result" || { log_error "curation-run did not return ok: $result"; return 1; }

    # Each step entry must have name, status, and summary fields
    local missing_fields
    missing_fields=$(echo "$result" | jq '[.result.steps[] | select((.name == null) or (.status == null) or (.summary == null))] | length')
    [[ "$missing_fields" -eq 0 ]] || { log_error "some steps missing name/status/summary fields: $result"; return 1; }

    # Each step status must be one of ok, failed, skipped
    local invalid_statuses
    invalid_statuses=$(echo "$result" | jq '[.result.steps[] | select(.status | IN("ok","failed","skipped") | not)] | length')
    [[ "$invalid_statuses" -eq 0 ]] || { log_error "some steps have invalid status values: $result"; return 1; }

    # succeeded count must equal count of steps with status "ok"
    local steps_ok reported_succeeded
    steps_ok=$(echo "$result" | jq '[.result.steps[] | select(.status == "ok")] | length')
    reported_succeeded=$(echo "$result" | jq '.result.succeeded')
    [[ "$reported_succeeded" -eq "$steps_ok" ]] || {
        log_error "succeeded ($reported_succeeded) != steps with status ok ($steps_ok)"; return 1;
    }

    # failed count must equal count of steps with status "failed"
    local steps_failed reported_failed
    steps_failed=$(echo "$result" | jq '[.result.steps[] | select(.status == "failed")] | length')
    reported_failed=$(echo "$result" | jq '.result.failed')
    [[ "$reported_failed" -eq "$steps_failed" ]] || {
        log_error "failed ($reported_failed) != steps with status failed ($steps_failed)"; return 1;
    }
}

# ============================================================================
# Main
# ============================================================================
run_test "test_curation_run_all_steps"       "curation-run executes all 8 steps in order"
run_test "test_curation_run_dry_run"         "curation-run --dry-run doesn't modify files"
run_test "test_curation_run_missing_instincts" "curation-run handles missing instincts.json gracefully"
run_test "test_curation_run_step_counts"     "curation-run reports correct step counts"

test_summary
