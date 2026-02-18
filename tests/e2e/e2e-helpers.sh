#!/usr/bin/env bash
# e2e-helpers.sh — Shared infrastructure for Phase 9 e2e test scripts
# Sources test-helpers.sh for assert_* functions
# Provides isolated environment setup, requirement tracking, result collection
#
# NOTE: Written for bash 3.2 (macOS default). No associative arrays.

set -euo pipefail

# ============================================================================
# Project Root Detection
# ============================================================================

E2E_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$E2E_SCRIPT_DIR/../.." && pwd)"

# Source test-helpers.sh for existing assert_* functions
# shellcheck source=../bash/test-helpers.sh
source "$PROJECT_ROOT/tests/bash/test-helpers.sh"

# ============================================================================
# E2E Environment Setup/Teardown
# ============================================================================

# E2E_TMP_DIR — set by setup_e2e_env, used by teardown_e2e_env
E2E_TMP_DIR=""

# setup_e2e_env — Creates isolated temp directory with full aether structure
# Returns the temp dir path via stdout
# Sets E2E_TMP_DIR global
setup_e2e_env() {
    local tmp
    tmp=$(mktemp -d)

    # Create directory structure
    mkdir -p "$tmp/.aether/data"
    mkdir -p "$tmp/.aether/exchange"
    mkdir -p "$tmp/.aether/utils"
    mkdir -p "$tmp/.aether/docs"

    # Copy aether-utils.sh
    cp "$PROJECT_ROOT/.aether/aether-utils.sh" "$tmp/.aether/"

    # Copy utils/ directory if it exists
    if [[ -d "$PROJECT_ROOT/.aether/utils" ]]; then
        cp -r "$PROJECT_ROOT/.aether/utils/." "$tmp/.aether/utils/"
    fi

    # Copy exchange/ directory for XML tests
    if [[ -d "$PROJECT_ROOT/.aether/exchange" ]]; then
        cp -r "$PROJECT_ROOT/.aether/exchange/." "$tmp/.aether/exchange/" 2>/dev/null || true
    fi

    # Create minimal COLONY_STATE.json fixture
    cat > "$tmp/.aether/data/COLONY_STATE.json" << 'EOF'
{
  "goal": "test-colony",
  "state": "active",
  "current_phase": 1,
  "milestone": "First Mound",
  "plan": {"id": "test-plan", "tasks": []},
  "memory": {
    "instincts": [],
    "phase_learnings": [],
    "decisions": []
  },
  "errors": {"records": []},
  "events": [],
  "session_id": "test-session-001",
  "initialized_at": "2026-02-18T00:00:00Z"
}
EOF

    # Create minimal pheromones.json fixture (with FOCUS and REDIRECT signals)
    cat > "$tmp/.aether/data/pheromones.json" << 'EOF'
{
  "signals": [
    {
      "id": "sig_focus_test001",
      "type": "FOCUS",
      "content": "Test focus area",
      "strength": 0.8,
      "effective_strength": 0.8,
      "active": true,
      "created_at": "2026-02-18T00:00:00Z",
      "expires_at": "phase_end",
      "source": "test"
    },
    {
      "id": "sig_redirect_test001",
      "type": "REDIRECT",
      "content": "Test redirect constraint",
      "strength": 0.9,
      "effective_strength": 0.9,
      "active": true,
      "created_at": "2026-02-18T00:00:00Z",
      "expires_at": "phase_end",
      "source": "test"
    }
  ],
  "midden": []
}
EOF

    # Create minimal constraints.json fixture
    cat > "$tmp/.aether/data/constraints.json" << 'EOF'
{
  "focus": ["test focus area"],
  "constraints": ["test redirect constraint"]
}
EOF

    E2E_TMP_DIR="$tmp"
    echo "$tmp"
}

# teardown_e2e_env — Removes isolated temp dir
teardown_e2e_env() {
    if [[ -n "${E2E_TMP_DIR:-}" && -d "$E2E_TMP_DIR" ]]; then
        rm -rf "$E2E_TMP_DIR"
        log_info "E2E environment cleaned up: $E2E_TMP_DIR"
    fi
    E2E_TMP_DIR=""
}

# ============================================================================
# Requirement Tracking
# Results stored in a temp file as "REQ_ID|STATUS|NOTES" lines
# Compatible with bash 3.2 (no associative arrays)
# ============================================================================

# RESULTS_FILE — temp file for storing requirement results
RESULTS_FILE=""

# init_results — Initialize results tracking for an area
init_results() {
    RESULTS_FILE=$(mktemp)
}

# record_result — Record pass/fail for a requirement
# Usage: record_result "REQ-ID" "PASS|FAIL" "optional notes"
record_result() {
    local req_id="$1"
    local status="$2"
    local notes="${3:-}"

    if [[ -n "$RESULTS_FILE" && -f "$RESULTS_FILE" ]]; then
        # Remove any existing entry for this req_id (update semantics)
        local tmp
        tmp=$(mktemp)
        grep -v "^${req_id}|" "$RESULTS_FILE" > "$tmp" 2>/dev/null || true
        echo "${req_id}|${status}|${notes}" >> "$tmp"
        mv "$tmp" "$RESULTS_FILE"
    fi
}

# print_area_results — Output a markdown table of results for the area
# Returns 0 if all PASS, 1 if any FAIL
print_area_results() {
    local area_name="${1:-Area}"
    echo ""
    echo "## $area_name Requirements Results"
    echo ""
    echo "| Requirement | Status | Notes |"
    echo "|-------------|--------|-------|"

    local pass_count=0
    local fail_count=0

    if [[ -n "$RESULTS_FILE" && -f "$RESULTS_FILE" ]]; then
        while IFS='|' read -r req_id status notes; do
            echo "| $req_id | $status | $notes |"
            if [[ "$status" == "PASS" ]]; then
                pass_count=$((pass_count + 1))
            else
                fail_count=$((fail_count + 1))
            fi
        done < <(sort "$RESULTS_FILE")
        rm -f "$RESULTS_FILE"
        RESULTS_FILE=""
    fi

    echo ""
    echo "**Summary:** $pass_count PASS, $fail_count FAIL"
    echo ""

    if [[ $fail_count -eq 0 ]]; then
        return 0
    else
        return 1
    fi
}

# ============================================================================
# JSON Extraction Helper
# ============================================================================

# extract_json — Strip non-JSON lines from output, return first valid JSON line
# Handles "Lock stale" and other prefix messages that appear before JSON
# Usage: clean_json=$(extract_json "$raw_output")
extract_json() {
    local raw="$1"
    local line
    while IFS= read -r line; do
        if echo "$line" | jq empty 2>/dev/null; then
            echo "$line"
            return 0
        fi
    done <<< "$raw"
    # If no line parsed, return raw
    echo "$raw"
}

# run_in_isolated_env — Run aether-utils.sh subcommand in isolated environment
# Usage: output=$(run_in_isolated_env "$tmp_dir" "subcommand" [args...])
run_in_isolated_env() {
    local tmp_dir="$1"
    local subcommand="$2"
    shift 2
    local utils="$tmp_dir/.aether/aether-utils.sh"
    bash "$utils" "$subcommand" "$@" 2>&1 || true
}

# Export functions for use in test scripts
export -f setup_e2e_env teardown_e2e_env
export -f init_results record_result print_area_results
export -f extract_json run_in_isolated_env
export PROJECT_ROOT E2E_SCRIPT_DIR
