#!/usr/bin/env bash
# Tests for hive-init subcommand
# Task 1.1: Create ~/.aether/hive/wisdom.json

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

# ============================================================================
# Helper: Create isolated test environment with its own HOME
# ============================================================================
setup_hive_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    mkdir -p "$tmpdir/.aether/data"
    echo "$tmpdir"
}

# Helper: run hive-init against a test env
run_hive_init() {
    local tmpdir="$1"
    shift
    HOME="$tmpdir" AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$AETHER_UTILS" hive-init "$@" 2>/dev/null
}

# ============================================================================
# Tests
# ============================================================================

test_hive_init_creates_directory() {
    # hive-init should create ~/.aether/hive/ directory
    local tmpdir
    tmpdir=$(setup_hive_env)

    local result
    result=$(run_hive_init "$tmpdir")
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_dir_exists "$tmpdir/.aether/hive"; then
        test_fail "~/.aether/hive/ directory should exist" "directory missing"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_init_creates_wisdom_json() {
    # hive-init should create wisdom.json with correct schema
    local tmpdir
    tmpdir=$(setup_hive_env)

    local result
    result=$(run_hive_init "$tmpdir")

    if ! assert_file_exists "$tmpdir/.aether/hive/wisdom.json"; then
        test_fail "wisdom.json should exist" "file missing"
        rm -rf "$tmpdir"
        return 1
    fi

    local wisdom
    wisdom=$(cat "$tmpdir/.aether/hive/wisdom.json")

    if ! assert_json_valid "$wisdom"; then
        test_fail "wisdom.json should be valid JSON" "invalid JSON"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_init_schema_version() {
    # wisdom.json should have version "1.0.0"
    local tmpdir
    tmpdir=$(setup_hive_env)

    run_hive_init "$tmpdir" >/dev/null

    local wisdom
    wisdom=$(cat "$tmpdir/.aether/hive/wisdom.json")

    if ! assert_json_field_equals "$wisdom" ".version" "1.0.0"; then
        test_fail "version should be 1.0.0" "$(echo "$wisdom" | jq -r '.version')"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_init_schema_entries() {
    # wisdom.json should have empty entries array
    local tmpdir
    tmpdir=$(setup_hive_env)

    run_hive_init "$tmpdir" >/dev/null

    local wisdom
    wisdom=$(cat "$tmpdir/.aether/hive/wisdom.json")

    if ! assert_json_array_length "$wisdom" ".entries" 0; then
        test_fail "entries should be empty array" "$(echo "$wisdom" | jq '.entries')"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_init_schema_metadata() {
    # wisdom.json should have metadata with correct fields
    local tmpdir
    tmpdir=$(setup_hive_env)

    run_hive_init "$tmpdir" >/dev/null

    local wisdom
    wisdom=$(cat "$tmpdir/.aether/hive/wisdom.json")

    if ! assert_json_field_equals "$wisdom" ".metadata.total_entries" "0"; then
        test_fail "metadata.total_entries should be 0" "$(echo "$wisdom" | jq '.metadata.total_entries')"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_json_field_equals "$wisdom" ".metadata.max_entries" "200"; then
        test_fail "metadata.max_entries should be 200" "$(echo "$wisdom" | jq '.metadata.max_entries')"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_json_array_length "$wisdom" ".metadata.contributing_repos" 0; then
        test_fail "metadata.contributing_repos should be empty" "$(echo "$wisdom" | jq '.metadata.contributing_repos')"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_init_schema_timestamps() {
    # wisdom.json should have created_at and last_updated timestamps
    local tmpdir
    tmpdir=$(setup_hive_env)

    run_hive_init "$tmpdir" >/dev/null

    local wisdom
    wisdom=$(cat "$tmpdir/.aether/hive/wisdom.json")

    local created_at
    created_at=$(echo "$wisdom" | jq -r '.created_at')

    if [[ "$created_at" == "null" || -z "$created_at" ]]; then
        test_fail "created_at should be set" "null or empty"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify ISO-8601 UTC format (YYYY-MM-DDTHH:MM:SSZ)
    if [[ ! "$created_at" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$ ]]; then
        test_fail "created_at should be ISO-8601 UTC" "$created_at"
        rm -rf "$tmpdir"
        return 1
    fi

    local last_updated
    last_updated=$(echo "$wisdom" | jq -r '.last_updated')

    if [[ "$last_updated" != "$created_at" ]]; then
        test_fail "last_updated should equal created_at on init" "created=$created_at, updated=$last_updated"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_init_returns_json_ok() {
    # hive-init should return json_ok response
    local tmpdir
    tmpdir=$(setup_hive_env)

    local result
    result=$(run_hive_init "$tmpdir")

    if ! assert_ok_true "$result"; then
        test_fail "ok should be true" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_json_field_equals "$result" ".result.initialized" "true"; then
        test_fail "result.initialized should be true" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_json_field_equals "$result" ".result.already_existed" "false"; then
        test_fail "result.already_existed should be false" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_init_idempotent() {
    # Running hive-init twice should NOT overwrite existing wisdom.json
    local tmpdir
    tmpdir=$(setup_hive_env)

    # First init
    run_hive_init "$tmpdir" >/dev/null

    # Capture the original created_at
    local original_created
    original_created=$(jq -r '.created_at' "$tmpdir/.aether/hive/wisdom.json")

    # Wait a moment to ensure different timestamp if overwritten
    sleep 1

    # Second init
    local result
    result=$(run_hive_init "$tmpdir")

    # Should report already_existed=true
    if ! assert_json_field_equals "$result" ".result.already_existed" "true"; then
        test_fail "already_existed should be true on second run" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    # File should still have original created_at (not overwritten)
    local current_created
    current_created=$(jq -r '.created_at' "$tmpdir/.aether/hive/wisdom.json")

    if [[ "$current_created" != "$original_created" ]]; then
        test_fail "wisdom.json should not be overwritten" "original=$original_created, current=$current_created"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_init_registered_in_help() {
    # hive-init should appear in help output
    local result
    result=$(bash "$AETHER_UTILS" help 2>/dev/null)

    if ! assert_contains "$result" "hive-init"; then
        test_fail "hive-init should be in help output" "not found in help"
        return 1
    fi

    return 0
}

# ============================================================================
# Run tests
# ============================================================================

log_info "Running hive-init subcommand tests"
log_info "Repo root: $REPO_ROOT"

run_test test_hive_init_creates_directory "hive-init creates ~/.aether/hive/ directory"
run_test test_hive_init_creates_wisdom_json "hive-init creates wisdom.json"
run_test test_hive_init_schema_version "wisdom.json has version 1.0.0"
run_test test_hive_init_schema_entries "wisdom.json has empty entries array"
run_test test_hive_init_schema_metadata "wisdom.json has correct metadata"
run_test test_hive_init_schema_timestamps "wisdom.json has ISO-8601 timestamps"
run_test test_hive_init_returns_json_ok "hive-init returns json_ok response"
run_test test_hive_init_idempotent "hive-init is idempotent (does not overwrite)"
run_test test_hive_init_registered_in_help "hive-init registered in help"

test_summary
