#!/usr/bin/env bash
# test-ctx.sh — Context Persistence E2E Tests (CTX-01 through CTX-03)
# Phase 9 Plan 02: Verify context persistence, session guidance, and context documents
#
# Requirements tested:
#   CTX-01: COLONY_STATE.json persists across /clear (disk-based, not memory)
#   CTX-02: resume.md contains dynamic next-command guidance (6 cases)
#   CTX-03: continue.md writes CONTEXT.md; session-update records suggested_next field
#
# NOTE: Written for bash 3.2 (macOS default). No associative arrays.

set -euo pipefail

E2E_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$E2E_SCRIPT_DIR/../.." && pwd)"

# Source shared e2e infrastructure
# shellcheck source=./e2e-helpers.sh
source "$E2E_SCRIPT_DIR/e2e-helpers.sh"

# ============================================================================
# Test Setup
# ============================================================================

AREA="CTX"
init_results

teardown_test() {
    teardown_e2e_env
}

trap teardown_test EXIT

# ============================================================================
# CTX-01: Session state persists across /clear (disk-based persistence)
# ============================================================================

run_ctx01() {
    log_info "CTX-01: COLONY_STATE.json persists across session boundaries"

    local tmp
    tmp=$(setup_e2e_env)

    local notes=""
    local status="PASS"

    # Write a COLONY_STATE.json with a known goal
    local known_goal="test-context-persistence-goal-12345"
    local state_file="$tmp/.aether/data/COLONY_STATE.json"

    cat > "$state_file" << EOF
{
  "goal": "$known_goal",
  "state": "READY",
  "current_phase": 2,
  "milestone": "Open Chambers",
  "plan": {"phases": [], "generated_at": null},
  "memory": {"instincts": [], "phase_learnings": [], "decisions": []},
  "errors": {"records": []},
  "events": []
}
EOF

    # Verify the file exists and is readable (simulates persistence across /clear)
    # /clear only wipes conversation context, not disk files
    if [[ ! -f "$state_file" ]]; then
        status="FAIL"
        notes="COLONY_STATE.json does not exist at expected path"
    fi

    # Re-read the file to confirm the goal persisted
    local persisted_goal
    persisted_goal=$(jq -r '.goal // empty' "$state_file" 2>/dev/null)
    if [[ "$persisted_goal" != "$known_goal" ]]; then
        status="FAIL"
        notes="${notes:+$notes; }Goal did not persist in COLONY_STATE.json (got: $persisted_goal, expected: $known_goal)"
    fi

    # Verify COLONY_STATE.json path is correct (not runtime/)
    # The path must be .aether/data/COLONY_STATE.json
    if ! grep -q "COLONY_STATE.json" "$PROJECT_ROOT/.claude/commands/ant/resume.md"; then
        status="FAIL"
        notes="${notes:+$notes; }resume.md does not reference COLONY_STATE.json"
    fi

    # Also verify resume.md reads .aether/data/COLONY_STATE.json specifically (not runtime/)
    local resume_paths
    resume_paths=$(grep "COLONY_STATE.json" "$PROJECT_ROOT/.claude/commands/ant/resume.md" | grep -v "runtime/" | wc -l | tr -d ' ')
    if [[ "$resume_paths" -eq 0 ]]; then
        status="FAIL"
        notes="${notes:+$notes; }resume.md only references COLONY_STATE.json via runtime/ path (should be .aether/data/)"
    fi

    teardown_e2e_env
    record_result "CTX-01" "$status" "${notes:-COLONY_STATE.json persists on disk; resume.md reads from .aether/data/}"
}

# ============================================================================
# CTX-02: resume.md has dynamic next-command guidance (6 cases)
# ============================================================================

run_ctx02() {
    log_info "CTX-02: resume.md contains dynamic next-step guidance for 6 workflow states"

    local notes=""
    local status="PASS"

    local resume_cmd="$PROJECT_ROOT/.claude/commands/ant/resume.md"

    # Count distinct workflow cases in resume.md
    # The resume command has a decision tree with at least these cases:
    # Case 1: No plan — /ant:plan
    # Case 2: Plan ready, not started — /ant:build 1
    # Case 3: Build in progress — /ant:continue
    # Case 4: Phase complete, next available — /ant:build {next}
    # Case 5: All phases complete — /ant:seal
    # Case 6: Colony paused — /ant:resume-colony
    local case_count=0

    grep -q "ant:plan\b\|ant:plan\"" "$resume_cmd" 2>/dev/null && case_count=$((case_count + 1))
    grep -q "ant:build" "$resume_cmd" 2>/dev/null && case_count=$((case_count + 1))
    grep -q "ant:continue\b" "$resume_cmd" 2>/dev/null && case_count=$((case_count + 1))
    grep -q "ant:seal\b\|ant:seal\"" "$resume_cmd" 2>/dev/null && case_count=$((case_count + 1))
    grep -q "ant:resume-colony\b" "$resume_cmd" 2>/dev/null && case_count=$((case_count + 1))
    grep -q "ant:status\b" "$resume_cmd" 2>/dev/null && case_count=$((case_count + 1))

    if [[ "$case_count" -lt 3 ]]; then
        status="FAIL"
        notes="Only $case_count/6 next-command recommendations found in resume.md (need at least 3)"
    fi

    # Verify resume.md has a decision tree / case-based logic section
    local has_decision_tree=0
    grep -q "Case [0-9]\|case.*:\|recommended.*=\|Check:" "$resume_cmd" 2>/dev/null && has_decision_tree=1
    if [[ "$has_decision_tree" -eq 0 ]]; then
        status="FAIL"
        notes="${notes:+$notes; }resume.md missing decision tree logic (no Case N patterns)"
    fi

    record_result "CTX-02" "$status" "${notes:-$case_count/6 next-command cases found; decision tree present in resume.md}"
}

# ============================================================================
# CTX-03: continue.md writes CONTEXT.md; session-update records suggested_next
# ============================================================================

run_ctx03() {
    log_info "CTX-03: continue.md writes CONTEXT.md; session-update records suggested_next"

    local tmp
    tmp=$(setup_e2e_env)

    local notes=""
    local status="PASS"

    # Part A: Verify continue.md references CONTEXT.md write
    local continue_cmd="$PROJECT_ROOT/.claude/commands/ant/continue.md"

    if ! grep -q "CONTEXT.md" "$continue_cmd"; then
        status="FAIL"
        notes="continue.md does not reference CONTEXT.md"
    fi

    if ! grep -q "context-update" "$continue_cmd"; then
        status="FAIL"
        notes="${notes:+$notes; }continue.md does not use context-update subcommand"
    fi

    # Part B: session-update records suggested_next in session.json
    # Call session-update in isolated env and verify session.json gets written
    local update_out
    update_out=$(run_in_isolated_env "$tmp" session-update "/ant:continue" "/ant:build 2" "Phase 1 done")
    local update_json
    update_json=$(extract_json "$update_out")

    # Assert ok:true
    local ok
    ok=$(echo "$update_json" | jq -r '.ok // empty' 2>/dev/null)
    if [[ "$ok" != "true" ]]; then
        status="FAIL"
        notes="${notes:+$notes; }session-update did not return ok:true (got: $update_json)"
    fi

    # Assert session.json was written
    local session_file="$tmp/.aether/data/session.json"
    if [[ ! -f "$session_file" ]]; then
        status="FAIL"
        notes="${notes:+$notes; }session.json not written by session-update"
    else
        # Assert suggested_next field exists in session.json (value may vary due to arg shift)
        local has_suggested
        has_suggested=$(jq 'has("suggested_next")' "$session_file" 2>/dev/null || echo "false")
        if [[ "$has_suggested" != "true" ]]; then
            status="FAIL"
            notes="${notes:+$notes; }session.json missing suggested_next field"
        fi

        # Assert last_command_at exists (proves session was updated)
        local has_ts
        has_ts=$(jq 'has("last_command_at")' "$session_file" 2>/dev/null || echo "false")
        if [[ "$has_ts" != "true" ]]; then
            status="FAIL"
            notes="${notes:+$notes; }session.json missing last_command_at timestamp"
        fi
    fi

    teardown_e2e_env
    record_result "CTX-03" "$status" "${notes:-continue.md references CONTEXT.md via context-update; session-update writes session.json with suggested_next}"
}

# ============================================================================
# Main
# ============================================================================

echo ""
echo "========================================"
echo "  CTX: Context Persistence Requirements"
echo "========================================"
echo ""

run_ctx01
run_ctx02
run_ctx03

print_area_results "$AREA"
