#!/usr/bin/env bash
# Learning Module Smoke Tests
# Tests learning.sh extracted module functions via aether-utils.sh subcommands

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
AETHER_UTILS_SOURCE="$PROJECT_ROOT/.aether/aether-utils.sh"

# Source test helpers
source "$SCRIPT_DIR/test-helpers.sh"

# Verify jq is available
require_jq

# Verify aether-utils.sh exists
if [[ ! -f "$AETHER_UTILS_SOURCE" ]]; then
    log_error "aether-utils.sh not found at: $AETHER_UTILS_SOURCE"
    exit 1
fi

# ============================================================================
# Helper: Create isolated test environment with learning support
# ============================================================================
setup_learning_env() {
    local tmp_dir
    tmp_dir=$(mktemp -d)
    mkdir -p "$tmp_dir/.aether/data" "$tmp_dir/.aether/utils"

    cp "$AETHER_UTILS_SOURCE" "$tmp_dir/.aether/aether-utils.sh"
    chmod +x "$tmp_dir/.aether/aether-utils.sh"

    local utils_source="$(dirname "$AETHER_UTILS_SOURCE")/utils"
    if [[ -d "$utils_source" ]]; then
        cp -r "$utils_source" "$tmp_dir/.aether/"
    fi

    local exchange_source="$(dirname "$AETHER_UTILS_SOURCE")/exchange"
    if [[ -d "$exchange_source" ]]; then
        cp -r "$exchange_source" "$tmp_dir/.aether/"
    fi

    # Write a minimal COLONY_STATE.json with memory structure
    cat > "$tmp_dir/.aether/data/COLONY_STATE.json" << 'CSEOF'
{
  "version": "3.0",
  "goal": "Test learning module",
  "state": "READY",
  "current_phase": 1,
  "milestone": "First Mound",
  "session_id": "colony_testcolony_123",
  "initialized_at": "2026-01-01T00:00:00Z",
  "tasks": [],
  "events": [],
  "instincts": [],
  "errors": {"count": 0, "last_error": null},
  "memory": {
    "observations": [],
    "key_decisions": [],
    "learnings": [],
    "instincts": [
      {
        "id": "instinct_test1",
        "trigger": "when testing patterns",
        "action": "use TDD approach",
        "confidence": 0.8,
        "status": "hypothesis",
        "domain": "testing",
        "source": "test-setup",
        "evidence": ["initial test"],
        "tested": false,
        "created_at": "2026-01-01T00:00:00Z",
        "last_applied": null,
        "applications": 0,
        "successes": 0,
        "failures": 0
      }
    ]
  }
}
CSEOF

    # Write learning-observations.json
    cat > "$tmp_dir/.aether/data/learning-observations.json" << 'OBSEOF'
{
  "observations": [
    {
      "content_hash": "sha256:abc123",
      "content": "Always validate JSON before writing",
      "wisdom_type": "pattern",
      "observation_count": 3,
      "first_seen": "2026-01-01T00:00:00Z",
      "last_seen": "2026-01-02T00:00:00Z",
      "colonies": ["testcolony"]
    }
  ]
}
OBSEOF

    # Write learnings.json for learning-inject
    cat > "$tmp_dir/.aether/data/learnings.json" << 'LEOF'
{
  "learnings": [
    {
      "id": "global_test1",
      "content": "Always validate inputs",
      "source_project": "test-project",
      "source_phase": "phase-1",
      "tags": ["validation", "testing"],
      "promoted_at": "2026-01-01T00:00:00Z"
    }
  ],
  "version": 1
}
LEOF

    echo "$tmp_dir"
}

run_learning_cmd() {
    local tmp_dir="$1"
    shift
    bash "$tmp_dir/.aether/aether-utils.sh" "$@" 2>/dev/null
}

# ============================================================================
# Test 1: Module exists and passes syntax check
# ============================================================================
test_module_exists() {
    local module_path="$PROJECT_ROOT/.aether/utils/learning.sh"
    assert_file_exists "$module_path" || return 1
    bash -n "$module_path" 2>/dev/null || return 1
}

# ============================================================================
# Test 2: instinct-read returns JSON with instincts
# ============================================================================
test_instinct_read() {
    local tmp_dir
    tmp_dir=$(setup_learning_env)

    local result
    result=$(run_learning_cmd "$tmp_dir" instinct-read)

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    local total
    total=$(echo "$result" | jq -r '.result.total')
    [[ "$total" -ge 1 ]] || { rm -rf "$tmp_dir"; return 1; }

    local first_trigger
    first_trigger=$(echo "$result" | jq -r '.result.instincts[0].trigger')
    [[ "$first_trigger" == "when testing patterns" ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test 3: learning-inject filters learnings by keyword
# ============================================================================
test_learning_inject() {
    local tmp_dir
    tmp_dir=$(setup_learning_env)

    local result
    result=$(run_learning_cmd "$tmp_dir" learning-inject "validation")

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    local count
    count=$(echo "$result" | jq -r '.result.count')
    [[ "$count" -ge 1 ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test 4: learning-check-promotion returns valid status
# ============================================================================
test_learning_check_promotion() {
    local tmp_dir
    tmp_dir=$(setup_learning_env)

    local result
    result=$(run_learning_cmd "$tmp_dir" learning-check-promotion)

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    # Should return proposals array (may be empty or populated depending on thresholds)
    local has_proposals
    has_proposals=$(echo "$result" | jq 'has("result") and (.result | has("proposals"))')
    [[ "$has_proposals" == "true" ]] || { rm -rf "$tmp_dir"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Run all tests
# ============================================================================
echo "=== Learning Module Smoke Tests ==="
echo ""

run_test test_module_exists "learning.sh exists and passes syntax check"
run_test test_instinct_read "instinct-read returns JSON with instincts"
run_test test_learning_inject "learning-inject filters learnings by keyword"
run_test test_learning_check_promotion "learning-check-promotion returns valid status"

test_summary
