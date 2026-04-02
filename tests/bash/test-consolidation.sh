#!/usr/bin/env bash
# Consolidation Module Tests
# Tests consolidation-phase-end via aether-utils.sh

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
setup_consolidation_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    mkdir -p "$tmpdir/.aether/data"

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

    # Minimal COLONY_STATE.json
    cat > "$tmpdir/.aether/data/COLONY_STATE.json" <<'EOF'
{
  "version": "3.0",
  "goal": "Test consolidation",
  "state": "READY",
  "current_phase": 2,
  "session_id": "test-session",
  "initialized_at": "2026-01-01T00:00:00Z",
  "build_started_at": null,
  "plan": { "phases": [{ "id": 2, "name": "Test Phase", "status": "pending" }] },
  "memory": { "phase_learnings": [], "decisions": [], "instincts": [] },
  "errors": { "records": [], "flagged_patterns": [] },
  "events": [],
  "signals": [],
  "graveyards": []
}
EOF

    # Minimal pheromones.json (required by several curation ants)
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
# TEST 1: consolidation-phase-end runs all 3 steps and returns ok
# ============================================================================
test_phase_end_runs_all_steps() {
    local tmpdir
    tmpdir=$(setup_consolidation_env)

    local result
    result=$(run_cmd "$tmpdir" consolidation-phase-end)

    rm -rf "$tmpdir"

    assert_ok_true "$result" || { log_error "not ok: $result"; return 1; }

    # type should be "phase_end"
    assert_json_field_equals "$result" ".result.type" "phase_end" || {
        log_error "expected type=phase_end, got: $(echo "$result" | jq -r '.result.type')"; return 1
    }

    # steps array should have 3 entries
    assert_json_array_length "$result" ".result.steps" 3 || {
        log_error "expected 3 steps, got: $(echo "$result" | jq '.result.steps | length')"; return 1
    }

    # each step should have name and status fields
    local names
    names=$(echo "$result" | jq -r '[.result.steps[].name] | join(",")' 2>/dev/null || echo "")
    [[ "$names" == "nurse,herald,janitor" ]] || {
        log_error "expected steps nurse,herald,janitor got: $names"; return 1
    }

    # dry_run should be false
    assert_json_field_equals "$result" ".result.dry_run" "false" || {
        log_error "expected dry_run=false"; return 1
    }
}

# ============================================================================
# TEST 2: consolidation-phase-end --dry-run sets dry_run=true and each step
#         receives --dry-run (does not write files)
# ============================================================================
test_phase_end_dry_run() {
    local tmpdir
    tmpdir=$(setup_consolidation_env)

    local result
    result=$(run_cmd "$tmpdir" consolidation-phase-end --dry-run)

    rm -rf "$tmpdir"

    assert_ok_true "$result" || { log_error "not ok: $result"; return 1; }

    assert_json_field_equals "$result" ".result.dry_run" "true" || {
        log_error "expected dry_run=true, got: $(echo "$result" | jq -r '.result.dry_run')"; return 1
    }

    # steps array still present with 3 entries
    assert_json_array_length "$result" ".result.steps" 3 || {
        log_error "expected 3 steps"; return 1
    }
}

# ============================================================================
# TEST 3: consolidation-phase-end handles missing data files gracefully
#         (no instincts.json, no learning-observations.json, no event-bus.jsonl)
# ============================================================================
test_phase_end_missing_data_files() {
    local tmpdir
    tmpdir=$(setup_consolidation_env)

    # Remove optional data files so curation ants work on empty state
    rm -f "$tmpdir/.aether/data/instincts.json"
    rm -f "$tmpdir/.aether/data/learning-observations.json"
    rm -f "$tmpdir/.aether/data/event-bus.jsonl"

    local result
    result=$(run_cmd "$tmpdir" consolidation-phase-end)

    rm -rf "$tmpdir"

    # Must still return ok — each step is non-blocking
    assert_ok_true "$result" || { log_error "expected ok=true with missing files: $result"; return 1; }

    assert_json_array_length "$result" ".result.steps" 3 || {
        log_error "expected 3 steps even with missing files"; return 1
    }
}

# ============================================================================
# TEST 4: consolidation-phase-end publishes an event to event-bus.jsonl
# ============================================================================
test_phase_end_publishes_event() {
    local tmpdir
    tmpdir=$(setup_consolidation_env)

    run_cmd "$tmpdir" consolidation-phase-end > /dev/null

    local bus_file="$tmpdir/.aether/data/event-bus.jsonl"

    # Event bus file should exist after the run
    assert_file_exists "$bus_file" || {
        rm -rf "$tmpdir"
        log_error "event-bus.jsonl was not created"
        return 1
    }

    # At least one event with topic "consolidation.phase_end"
    local match_count
    match_count=$(jq -c 'select(.topic == "consolidation.phase_end")' "$bus_file" 2>/dev/null | wc -l | tr -d ' ')

    rm -rf "$tmpdir"

    [[ "$match_count" -ge 1 ]] || {
        log_error "no consolidation.phase_end event found in event-bus.jsonl"
        return 1
    }
}

# ============================================================================
# Main
# ============================================================================
run_test "test_phase_end_runs_all_steps"      "consolidation-phase-end runs all 3 steps"
run_test "test_phase_end_dry_run"             "consolidation-phase-end --dry-run sets dry_run=true"
run_test "test_phase_end_missing_data_files"  "consolidation-phase-end handles missing data files gracefully"
run_test "test_phase_end_publishes_event"     "consolidation-phase-end publishes event to event-bus.jsonl"

test_summary
