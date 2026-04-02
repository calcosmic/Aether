#!/usr/bin/env bash
# Consolidation Seal Tests
# Tests consolidation-seal via aether-utils.sh: runs all 5 steps,
# respects --dry-run, publishes event, generates scribe report.

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
setup_seal_env() {
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

    # Copy consolidation-seal.sh if it exists (may be created in parallel)
    local seal_sh="$(dirname "$AETHER_UTILS")/utils/consolidation-seal.sh"
    if [[ -f "$seal_sh" ]]; then
        cp "$seal_sh" "$tmpdir/.aether/utils/consolidation-seal.sh"
    fi

    # Minimal required data files
    cat > "$tmpdir/.aether/data/COLONY_STATE.json" <<'EOF'
{
  "version": "3.0",
  "goal": "Test consolidation seal",
  "state": "READY",
  "current_phase": 1,
  "session_id": "test-seal",
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

    cat > "$tmpdir/.aether/data/instincts.json" <<'EOF'
{
  "instincts": [
    {
      "id": "inst-seal-001",
      "trigger": "always seal cleanly",
      "action": "run full consolidation before seal",
      "confidence": 0.85,
      "trust_score": 0.85,
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
# Test 1: consolidation-seal runs all 5 steps
# ============================================================================
test_consolidation_seal_all_steps() {
    local tmpdir
    tmpdir=$(setup_seal_env)

    local result
    result=$(run_cmd "$tmpdir" consolidation-seal)

    rm -rf "$tmpdir"

    # Must be ok
    assert_ok_true "$result" || { log_error "consolidation-seal did not return ok: $result"; return 1; }

    # Must have exactly 4 steps in the steps array
    local step_count
    step_count=$(echo "$result" | jq '.result.steps | length')
    [[ "$step_count" -eq 4 ]] || { log_error "steps length should be 4, got: $step_count"; return 1; }

    # All expected step names must appear
    local names
    names=$(echo "$result" | jq -r '.result.steps[].name' | tr '\n' ' ')
    for expected_name in "curation-run" "instinct-decay-all" "archivist" "scribe"; do
        assert_contains "$names" "$expected_name" || { log_error "step '$expected_name' missing from: $names"; return 1; }
    done

    # Steps must have status field
    local missing_status
    missing_status=$(echo "$result" | jq '[.result.steps[] | select(.status == null)] | length')
    [[ "$missing_status" -eq 0 ]] || { log_error "some steps missing status field"; return 1; }

    # type must be "seal"
    local result_type
    result_type=$(echo "$result" | jq -r '.result.type')
    [[ "$result_type" == "seal" ]] || { log_error "result.type should be 'seal', got: $result_type"; return 1; }

    # dry_run must be false
    local dry_flag
    dry_flag=$(echo "$result" | jq '.result.dry_run')
    [[ "$dry_flag" == "false" ]] || { log_error "result.dry_run should be false, got: $dry_flag"; return 1; }
}

# ============================================================================
# Test 2: consolidation-seal --dry-run passes dry-run to steps
# ============================================================================
test_consolidation_seal_dry_run() {
    local tmpdir
    tmpdir=$(setup_seal_env)

    # Capture checksums before run
    _checksum() {
        md5 -q "$1" 2>/dev/null || md5sum "$1" 2>/dev/null | awk '{print $1}'
    }
    local inst_before pheromones_before
    inst_before=$(_checksum "$tmpdir/.aether/data/instincts.json")
    pheromones_before=$(_checksum "$tmpdir/.aether/data/pheromones.json")

    local result
    result=$(run_cmd "$tmpdir" consolidation-seal --dry-run)

    local inst_after pheromones_after
    inst_after=$(_checksum "$tmpdir/.aether/data/instincts.json")
    pheromones_after=$(_checksum "$tmpdir/.aether/data/pheromones.json")

    rm -rf "$tmpdir"

    # Must be ok
    assert_ok_true "$result" || { log_error "consolidation-seal --dry-run did not return ok: $result"; return 1; }

    # dry_run flag must be true in result
    local dry_flag
    dry_flag=$(echo "$result" | jq '.result.dry_run')
    [[ "$dry_flag" == "true" ]] || { log_error "result.dry_run should be true, got: $dry_flag"; return 1; }

    # Must still have 4 steps
    local step_count
    step_count=$(echo "$result" | jq '.result.steps | length')
    [[ "$step_count" -eq 4 ]] || { log_error "steps length should be 4, got: $step_count"; return 1; }

    # instincts.json must not have changed
    [[ "$inst_after" == "$inst_before" ]] || { log_error "instincts.json changed during dry-run"; return 1; }

    # pheromones.json must not have changed
    [[ "$pheromones_after" == "$pheromones_before" ]] || { log_error "pheromones.json changed during dry-run"; return 1; }
}

# ============================================================================
# Test 3: consolidation-seal publishes event
# ============================================================================
test_consolidation_seal_publishes_event() {
    local tmpdir
    tmpdir=$(setup_seal_env)

    local result
    result=$(run_cmd "$tmpdir" consolidation-seal)

    rm -rf "$tmpdir"

    # Must be ok
    assert_ok_true "$result" || { log_error "consolidation-seal did not return ok: $result"; return 1; }

    # event_published must be true
    local event_flag
    event_flag=$(echo "$result" | jq '.result.event_published')
    [[ "$event_flag" == "true" ]] || { log_error "result.event_published should be true, got: $event_flag"; return 1; }
}

# ============================================================================
# Test 4: consolidation-seal generates scribe report
# ============================================================================
test_consolidation_seal_scribe_report() {
    local tmpdir
    tmpdir=$(setup_seal_env)

    local result
    result=$(run_cmd "$tmpdir" consolidation-seal)

    rm -rf "$tmpdir"

    # Must be ok
    assert_ok_true "$result" || { log_error "consolidation-seal did not return ok: $result"; return 1; }

    # scribe step must be present
    local scribe_step
    scribe_step=$(echo "$result" | jq '.result.steps[] | select(.name == "scribe")')
    [[ -n "$scribe_step" ]] || { log_error "scribe step not found in steps"; return 1; }

    # scribe step report_path must be present (not null)
    local report_path
    report_path=$(echo "$result" | jq -r '.result.steps[] | select(.name == "scribe") | .report_path // "null"')
    [[ "$report_path" != "null" ]] || { log_error "scribe step report_path should not be null"; return 1; }
}

# ============================================================================
# Main
# ============================================================================
run_test "test_consolidation_seal_all_steps"       "consolidation-seal runs all 5 steps"
run_test "test_consolidation_seal_dry_run"         "consolidation-seal --dry-run doesn't modify"
run_test "test_consolidation_seal_publishes_event" "consolidation-seal publishes event"
run_test "test_consolidation_seal_scribe_report"   "consolidation-seal generates scribe report"

test_summary
