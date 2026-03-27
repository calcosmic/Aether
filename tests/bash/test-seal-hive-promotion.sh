#!/usr/bin/env bash
# Tests for seal.md Step 3.7: Hive Promotion integration
# Task 4.1: Verifies that seal hive-promotion extracts instincts correctly
# and calls hive-promote with --text and --source-repo (NOT --instinct)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

# ============================================================================
# Helper: Create isolated test environment with colony state and hive
# ============================================================================
setup_seal_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    mkdir -p "$tmpdir/.aether/data"
    echo "$tmpdir"
}

# Helper: run aether-utils against a test env
run_utils() {
    local tmpdir="$1"
    shift
    HOME="$tmpdir" AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$AETHER_UTILS" "$@" 2>&1
}

# ============================================================================
# Tests
# ============================================================================

test_seal_hive_promotion_extracts_high_confidence_instincts() {
    # The seal Step 3.7 bash snippet should extract instincts with confidence >= 0.8
    # and pass them to hive-promote with --text and --source-repo
    local tmpdir
    tmpdir=$(setup_seal_env)

    # Create colony state with instincts of varying confidence
    cat > "$tmpdir/.aether/data/COLONY_STATE.json" << 'COLSTATE'
{
  "goal": "Test colony",
  "state": "active",
  "current_phase": 2,
  "version": "3.0",
  "initialized_at": "2026-03-01T00:00:00Z",
  "plan": {"phases": []},
  "instincts": [
    {"trigger": "writing tests", "action": "use TDD discipline", "confidence": 0.9, "domain": "testing", "source": "phase-1", "evidence": ["test"]},
    {"trigger": "handling errors", "action": "use early returns", "confidence": 0.8, "domain": "patterns", "source": "phase-1", "evidence": ["err"]},
    {"trigger": "naming variables", "action": "use camelCase", "confidence": 0.5, "domain": "style", "source": "phase-2", "evidence": ["style"]}
  ],
  "memory": {},
  "errors": {"records": []},
  "events": [],
  "session_id": "test-session"
}
COLSTATE

    # Run the actual seal hive-promotion snippet (the corrected version using --text and --source-repo)
    # This simulates what seal.md Step 3.7 does
    local hive_promoted_count=0
    local source_repo_name
    source_repo_name=$(basename "$tmpdir")

    high_conf_instincts=$(jq -r '.instincts[] | select(.confidence >= 0.8) | @base64' "$tmpdir/.aether/data/COLONY_STATE.json" 2>/dev/null || echo "")

    local instinct_count=0
    for encoded in $high_conf_instincts; do
        [[ -z "$encoded" ]] && continue
        instinct_count=$((instinct_count + 1))

        trigger=$(echo "$encoded" | base64 -d | jq -r '.trigger // empty')
        action=$(echo "$encoded" | base64 -d | jq -r '.action // empty')
        confidence=$(echo "$encoded" | base64 -d | jq -r '.confidence // 0.7')
        domain=$(echo "$encoded" | base64 -d | jq -r '.domain // empty')
        promote_text="When ${trigger}: ${action}"

        result=$(run_utils "$tmpdir" hive-promote \
            --text "$promote_text" \
            --source-repo "$tmpdir" \
            --confidence "$confidence" \
            ${domain:+--domain "$domain"}) || true

        was_promoted=$(echo "$result" | jq -r '.result.action // "skipped"' 2>/dev/null || echo "skipped")
        if [[ "$was_promoted" == "promoted" || "$was_promoted" == "merged" ]]; then
            hive_promoted_count=$((hive_promoted_count + 1))
        fi
    done

    # Should have found exactly 2 instincts with confidence >= 0.8
    if [[ "$instinct_count" -ne 2 ]]; then
        test_fail "should extract 2 high-confidence instincts" "got: $instinct_count"
        rm -rf "$tmpdir"
        return 1
    fi

    # Both should have been promoted
    if [[ "$hive_promoted_count" -ne 2 ]]; then
        test_fail "should promote 2 instincts" "got: $hive_promoted_count"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_seal_hive_promotion_uses_text_not_instinct() {
    # Verify that the hive-promote call uses --text (not --instinct which does not exist)
    local tmpdir
    tmpdir=$(setup_seal_env)

    cat > "$tmpdir/.aether/data/COLONY_STATE.json" << 'COLSTATE'
{
  "goal": "Test colony",
  "state": "active",
  "current_phase": 1,
  "version": "3.0",
  "initialized_at": "2026-03-01T00:00:00Z",
  "plan": {"phases": []},
  "instincts": [
    {"trigger": "deploying code", "action": "run smoke tests first", "confidence": 0.85, "domain": "devops", "source": "phase-1", "evidence": ["deploy"]}
  ],
  "memory": {},
  "errors": {"records": []},
  "events": [],
  "session_id": "test-session"
}
COLSTATE

    # Call hive-promote with --text format (correct API)
    local result exit_code=0
    result=$(run_utils "$tmpdir" hive-promote \
        --text "When deploying code: run smoke tests first" \
        --source-repo "$tmpdir" \
        --confidence 0.85) || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "hive-promote with --text should succeed" "exit code: $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "should return ok:true" "got: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify the promoted text contains the trigger:action format
    local abstracted
    abstracted=$(echo "$result" | jq -r '.result.original // empty')
    if ! assert_contains "$abstracted" "deploying code"; then
        test_fail "original should contain trigger text" "got: $abstracted"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_seal_hive_promotion_instinct_flag_fails() {
    # Verify that --instinct is NOT a valid flag (would fail validation)
    local tmpdir
    tmpdir=$(setup_seal_env)

    local result exit_code=0
    result=$(run_utils "$tmpdir" hive-promote \
        --instinct '{"trigger":"test","action":"test"}' \
        --source-repo "$tmpdir") || exit_code=$?

    # Should fail because --instinct is not recognized; --text is required
    if [[ "$exit_code" -eq 0 ]]; then
        # Check if it actually errored via JSON
        local ok_val
        ok_val=$(echo "$result" | jq -r '.ok // "true"' 2>/dev/null)
        if [[ "$ok_val" == "true" ]]; then
            test_fail "should fail with --instinct (not a valid flag)" "succeeded unexpectedly"
            rm -rf "$tmpdir"
            return 1
        fi
    fi

    # If we get here, it correctly failed (either exit code or ok:false)
    rm -rf "$tmpdir"
    return 0
}

test_seal_hive_promotion_no_instincts_is_silent() {
    # When there are no high-confidence instincts, no promotions should occur
    local tmpdir
    tmpdir=$(setup_seal_env)

    cat > "$tmpdir/.aether/data/COLONY_STATE.json" << 'COLSTATE'
{
  "goal": "Test colony",
  "state": "active",
  "current_phase": 1,
  "version": "3.0",
  "initialized_at": "2026-03-01T00:00:00Z",
  "plan": {"phases": []},
  "instincts": [
    {"trigger": "low confidence thing", "action": "maybe do this", "confidence": 0.3, "domain": "misc", "source": "phase-1", "evidence": ["test"]}
  ],
  "memory": {},
  "errors": {"records": []},
  "events": [],
  "session_id": "test-session"
}
COLSTATE

    local high_conf_instincts
    high_conf_instincts=$(jq -r '.instincts[] | select(.confidence >= 0.8) | @base64' "$tmpdir/.aether/data/COLONY_STATE.json" 2>/dev/null || echo "")

    local count=0
    for encoded in $high_conf_instincts; do
        [[ -z "$encoded" ]] && continue
        count=$((count + 1))
    done

    if [[ "$count" -ne 0 ]]; then
        test_fail "should find 0 high-confidence instincts" "got: $count"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_seal_hive_promotion_trigger_action_format() {
    # Verify the "When {trigger}: {action}" text format
    local tmpdir
    tmpdir=$(setup_seal_env)

    cat > "$tmpdir/.aether/data/COLONY_STATE.json" << 'COLSTATE'
{
  "goal": "Test colony",
  "state": "active",
  "current_phase": 1,
  "version": "3.0",
  "initialized_at": "2026-03-01T00:00:00Z",
  "plan": {"phases": []},
  "instincts": [
    {"trigger": "writing subcommands", "action": "override LOCK_DIR first", "confidence": 0.9, "domain": "architecture", "source": "phase-1", "evidence": ["lock"]}
  ],
  "memory": {},
  "errors": {"records": []},
  "events": [],
  "session_id": "test-session"
}
COLSTATE

    local high_conf_instincts
    high_conf_instincts=$(jq -r '.instincts[] | select(.confidence >= 0.8) | @base64' "$tmpdir/.aether/data/COLONY_STATE.json" 2>/dev/null || echo "")

    for encoded in $high_conf_instincts; do
        [[ -z "$encoded" ]] && continue
        local trigger action promote_text
        trigger=$(echo "$encoded" | base64 -d | jq -r '.trigger // empty')
        action=$(echo "$encoded" | base64 -d | jq -r '.action // empty')
        promote_text="When ${trigger}: ${action}"

        if [[ "$promote_text" != "When writing subcommands: override LOCK_DIR first" ]]; then
            test_fail "promote_text should be 'When {trigger}: {action}'" "got: $promote_text"
            rm -rf "$tmpdir"
            return 1
        fi

        # Verify the format works with hive-promote
        local result
        result=$(run_utils "$tmpdir" hive-promote \
            --text "$promote_text" \
            --source-repo "$tmpdir") || true

        if ! assert_ok_true "$result"; then
            test_fail "hive-promote should accept trigger:action format" "got: $result"
            rm -rf "$tmpdir"
            return 1
        fi
    done

    rm -rf "$tmpdir"
    return 0
}

test_seal_hive_promotion_passes_confidence_and_domain() {
    # Verify confidence and domain from instinct are forwarded to hive-promote
    local tmpdir
    tmpdir=$(setup_seal_env)

    cat > "$tmpdir/.aether/data/COLONY_STATE.json" << 'COLSTATE'
{
  "goal": "Test colony",
  "state": "active",
  "current_phase": 1,
  "version": "3.0",
  "initialized_at": "2026-03-01T00:00:00Z",
  "plan": {"phases": []},
  "instincts": [
    {"trigger": "writing APIs", "action": "validate input schemas", "confidence": 0.95, "domain": "security", "source": "phase-1", "evidence": ["api"]}
  ],
  "memory": {},
  "errors": {"records": []},
  "events": [],
  "session_id": "test-session"
}
COLSTATE

    local result
    result=$(run_utils "$tmpdir" hive-promote \
        --text "When writing APIs: validate input schemas" \
        --source-repo "$tmpdir" \
        --confidence 0.95 \
        --domain "security")

    if ! assert_ok_true "$result"; then
        test_fail "should succeed with confidence and domain" "got: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local stored_confidence
    stored_confidence=$(echo "$result" | jq -r '.result.confidence')
    if [[ "$stored_confidence" != "0.95" ]]; then
        test_fail "confidence should be 0.95" "got: $stored_confidence"
        rm -rf "$tmpdir"
        return 1
    fi

    # Check domain was stored by reading back
    local read_result
    read_result=$(HOME="$tmpdir" AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$AETHER_UTILS" hive-read --domain "security" --format json 2>&1)

    local entry_count
    entry_count=$(echo "$read_result" | jq '.result.entries | length' 2>/dev/null || echo "0")

    if [[ "$entry_count" -lt 1 ]]; then
        test_fail "should find entry by domain tag 'security'" "got $entry_count entries"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Run all tests
# ============================================================================

run_test test_seal_hive_promotion_extracts_high_confidence_instincts "seal-hive: extracts instincts with confidence >= 0.8"
run_test test_seal_hive_promotion_uses_text_not_instinct "seal-hive: uses --text not --instinct"
run_test test_seal_hive_promotion_instinct_flag_fails "seal-hive: --instinct flag is not valid"
run_test test_seal_hive_promotion_no_instincts_is_silent "seal-hive: no promotions when no high-confidence instincts"
run_test test_seal_hive_promotion_trigger_action_format "seal-hive: formats text as When {trigger}: {action}"
run_test test_seal_hive_promotion_passes_confidence_and_domain "seal-hive: passes confidence and domain to hive-promote"

test_summary
