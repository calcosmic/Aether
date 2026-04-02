#!/usr/bin/env bash
# Curation Ops Tests
# Tests sentinel, janitor, archivist, and scribe functions via aether-utils.sh

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
setup_curation_env() {
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
  "goal": "Test curation",
  "state": "READY",
  "current_phase": 1,
  "session_id": "test-session",
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

    # Minimal pheromones.json
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

run_cmd_with_stderr() {
    local tmpdir="$1"
    shift
    AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        COLONY_DATA_DIR="$tmpdir/.aether/data" \
        bash "$tmpdir/.aether/aether-utils.sh" "$@" 2>&1 || true
}

# ============================================================================
# TEST 1: sentinel reports healthy on valid stores
# ============================================================================
test_sentinel_healthy_stores() {
    local tmpdir
    tmpdir=$(setup_curation_env)

    # Create valid learning-observations.json
    cat > "$tmpdir/.aether/data/learning-observations.json" <<'EOF'
{"version":"1.0","observations":[]}
EOF

    local result
    result=$(run_cmd "$tmpdir" curation-sentinel)

    rm -rf "$tmpdir"

    assert_ok_true "$result" || return 1

    local healthy_count
    healthy_count=$(echo "$result" | jq '.result.healthy')
    [[ "$healthy_count" -gt 0 ]] || return 1

    local issues
    issues=$(echo "$result" | jq '.result.issues')
    [[ "$issues" -eq 0 ]] || return 1
}

# ============================================================================
# TEST 2: sentinel detects missing file
# ============================================================================
test_sentinel_detects_missing_file() {
    local tmpdir
    tmpdir=$(setup_curation_env)

    # Remove pheromones.json so sentinel can flag it missing
    rm -f "$tmpdir/.aether/data/pheromones.json"

    local result
    result=$(run_cmd "$tmpdir" curation-sentinel)

    rm -rf "$tmpdir"

    assert_ok_true "$result" || return 1

    local issues
    issues=$(echo "$result" | jq '.result.issues')
    [[ "$issues" -gt 0 ]] || return 1

    # At least one check should be "missing"
    local missing_count
    missing_count=$(echo "$result" | jq '[.result.checks[] | select(.status == "missing")] | length')
    [[ "$missing_count" -gt 0 ]] || return 1
}

# ============================================================================
# TEST 3: janitor removes expired events from event-bus.jsonl
# ============================================================================
test_janitor_removes_expired_events() {
    local tmpdir
    tmpdir=$(setup_curation_env)

    # Create event-bus.jsonl with one expired event and one valid event
    local past_ts="2020-01-01T00:00:00Z"
    local future_ts="2099-12-31T23:59:59Z"

    printf '{"id":"evt_old","topic":"test","payload":{},"source":"test","timestamp":"2020-01-01T00:00:00Z","ttl_days":30,"expires_at":"%s"}\n' "$past_ts" \
        > "$tmpdir/.aether/data/event-bus.jsonl"
    printf '{"id":"evt_new","topic":"test","payload":{},"source":"test","timestamp":"2026-01-01T00:00:00Z","ttl_days":30,"expires_at":"%s"}\n' "$future_ts" \
        >> "$tmpdir/.aether/data/event-bus.jsonl"

    local result
    result=$(run_cmd "$tmpdir" curation-janitor)

    rm -rf "$tmpdir"

    assert_ok_true "$result" || return 1

    local events_removed
    events_removed=$(echo "$result" | jq '.result.events_removed')
    [[ "$events_removed" -ge 1 ]] || return 1
}

# ============================================================================
# TEST 4: archivist archives low-trust instincts
# ============================================================================
test_archivist_archives_low_trust() {
    local tmpdir
    tmpdir=$(setup_curation_env)

    # Create instincts.json with one low-trust instinct
    cat > "$tmpdir/.aether/data/instincts.json" <<'EOF'
{
  "version": "1.0",
  "instincts": [
    {
      "id": "inst_001",
      "trigger": "Low trust pattern",
      "action": "do something",
      "domain": "testing",
      "confidence": 0.1,
      "trust_score": 0.1,
      "trust_tier": "provisional",
      "archived": false,
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    },
    {
      "id": "inst_002",
      "trigger": "High trust pattern",
      "action": "do something else",
      "domain": "testing",
      "confidence": 0.9,
      "trust_score": 0.9,
      "trust_tier": "core",
      "archived": false,
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    }
  ]
}
EOF

    local result
    result=$(run_cmd "$tmpdir" curation-archivist --threshold 0.25)

    rm -rf "$tmpdir"

    assert_ok_true "$result" || return 1

    local archived
    archived=$(echo "$result" | jq '.result.archived')
    [[ "$archived" -eq 1 ]] || return 1

    local below_threshold
    below_threshold=$(echo "$result" | jq '.result.below_threshold')
    [[ "$below_threshold" -eq 1 ]] || return 1
}

# ============================================================================
# TEST 5: scribe generates report file
# ============================================================================
test_scribe_generates_report() {
    local tmpdir
    tmpdir=$(setup_curation_env)

    # Provide learning-observations.json so librarian call works gracefully
    cat > "$tmpdir/.aether/data/learning-observations.json" <<'EOF'
{"version":"1.0","observations":[]}
EOF

    local output_path="$tmpdir/.aether/data/curation-report.md"

    local result
    result=$(run_cmd "$tmpdir" curation-scribe --output "$output_path")

    local file_ok=false
    [[ -f "$output_path" ]] && file_ok=true

    rm -rf "$tmpdir"

    assert_ok_true "$result" || return 1

    local report_path
    report_path=$(echo "$result" | jq -r '.result.report_path')
    [[ -n "$report_path" ]] || return 1

    [[ "$file_ok" == "true" ]] || return 1
}

# ============================================================================
# TEST 6: each handler is graceful when optional files are absent
# ============================================================================
test_handlers_graceful_without_optional_files() {
    local tmpdir
    tmpdir=$(setup_curation_env)

    # Do NOT create instincts.json, instinct-graph.json, event-bus.jsonl,
    # or learning-observations.json — only COLONY_STATE.json and pheromones.json exist.

    local sentinel_result janitor_result archivist_result
    sentinel_result=$(run_cmd "$tmpdir" curation-sentinel)
    janitor_result=$(run_cmd "$tmpdir" curation-janitor --dry-run)
    archivist_result=$(run_cmd "$tmpdir" curation-archivist --dry-run)

    rm -rf "$tmpdir"

    assert_ok_true "$sentinel_result"  || { log_error "sentinel not ok: $sentinel_result"; return 1; }
    assert_ok_true "$janitor_result"   || { log_error "janitor not ok: $janitor_result"; return 1; }
    assert_ok_true "$archivist_result" || { log_error "archivist not ok: $archivist_result"; return 1; }
}

# ============================================================================
# Main
# ============================================================================
run_test "test_sentinel_healthy_stores"         "sentinel reports healthy on valid stores"
run_test "test_sentinel_detects_missing_file"   "sentinel detects missing pheromones file"
run_test "test_janitor_removes_expired_events"  "janitor removes expired events"
run_test "test_archivist_archives_low_trust"    "archivist archives low-trust instincts"
run_test "test_scribe_generates_report"         "scribe generates report file"
run_test "test_handlers_graceful_without_optional_files" "all handlers graceful with missing optional files"

test_summary
