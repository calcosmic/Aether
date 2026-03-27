#!/usr/bin/env bash
# Tests for hive-promote subcommand
# Task 2.2: Orchestrates abstract+store pipeline for cross-colony wisdom promotion

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

# Helper: run hive-promote against a test env (stderr merged to stdout)
run_hive_promote() {
    local tmpdir="$1"
    shift
    HOME="$tmpdir" AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$AETHER_UTILS" hive-promote "$@" 2>&1
}

# ============================================================================
# Tests
# ============================================================================

test_hive_promote_missing_text() {
    # Should fail when --text is missing
    local tmpdir
    tmpdir=$(setup_hive_env)

    local result exit_code=0
    result=$(run_hive_promote "$tmpdir" --source-repo "/Users/me/repos/MyApp") || exit_code=$?

    if [[ "$exit_code" -eq 0 ]]; then
        test_fail "should fail without --text" "got exit code 0"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_contains "$result" "text"; then
        test_fail "error should mention 'text'" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_promote_missing_source_repo() {
    # Should fail when --source-repo is missing
    local tmpdir
    tmpdir=$(setup_hive_env)

    local result exit_code=0
    result=$(run_hive_promote "$tmpdir" --text "Always validate input") || exit_code=$?

    if [[ "$exit_code" -eq 0 ]]; then
        test_fail "should fail without --source-repo" "got exit code 0"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_contains "$result" "source_repo"; then
        test_fail "error should mention 'source_repo'" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_promote_basic_success() {
    # Should return success JSON with correct structure on valid input
    local tmpdir
    tmpdir=$(setup_hive_env)

    local result
    result=$(run_hive_promote "$tmpdir" \
        --text "Always validate input before saving" \
        --source-repo "/Users/me/repos/MyApp")

    if ! assert_ok_true "$result"; then
        test_fail "should succeed" "got: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Check required fields exist in result
    local has_action has_original has_abstracted has_store_action has_confidence has_source
    has_action=$(echo "$result" | jq -e '.result.action' >/dev/null 2>&1 && echo yes || echo no)
    has_original=$(echo "$result" | jq -e '.result.original' >/dev/null 2>&1 && echo yes || echo no)
    has_abstracted=$(echo "$result" | jq -e '.result.abstracted' >/dev/null 2>&1 && echo yes || echo no)
    has_store_action=$(echo "$result" | jq -e '.result.store_action' >/dev/null 2>&1 && echo yes || echo no)
    has_confidence=$(echo "$result" | jq -e '.result.confidence' >/dev/null 2>&1 && echo yes || echo no)
    has_source=$(echo "$result" | jq -e '.result.source_repo' >/dev/null 2>&1 && echo yes || echo no)

    if [[ "$has_action" != "yes" ]] || [[ "$has_original" != "yes" ]] || \
       [[ "$has_abstracted" != "yes" ]] || [[ "$has_store_action" != "yes" ]] || \
       [[ "$has_confidence" != "yes" ]] || [[ "$has_source" != "yes" ]]; then
        test_fail "result should have action, original, abstracted, store_action, confidence, source_repo" \
            "action=$has_action original=$has_original abstracted=$has_abstracted store_action=$has_store_action confidence=$has_confidence source=$has_source"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_promote_action_promoted_for_new() {
    # For a new entry, action should be "promoted" (maps from store's "stored")
    local tmpdir
    tmpdir=$(setup_hive_env)

    local result
    result=$(run_hive_promote "$tmpdir" \
        --text "Always validate input before saving" \
        --source-repo "/Users/me/repos/MyApp")

    local action
    action=$(echo "$result" | jq -r '.result.action')

    if [[ "$action" != "promoted" ]]; then
        test_fail "action should be 'promoted' for new entry" "got: $action"
        rm -rf "$tmpdir"
        return 1
    fi

    local store_action
    store_action=$(echo "$result" | jq -r '.result.store_action')

    if [[ "$store_action" != "stored" ]]; then
        test_fail "store_action should be 'stored'" "got: $store_action"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_promote_action_skipped_for_duplicate() {
    # Promoting the same text from the same repo twice should return "skipped"
    local tmpdir
    tmpdir=$(setup_hive_env)

    # First promotion
    run_hive_promote "$tmpdir" \
        --text "Always validate input before saving" \
        --source-repo "/Users/me/repos/MyApp" >/dev/null

    # Second promotion (same text, same repo)
    local result
    result=$(run_hive_promote "$tmpdir" \
        --text "Always validate input before saving" \
        --source-repo "/Users/me/repos/MyApp")

    local action
    action=$(echo "$result" | jq -r '.result.action')

    if [[ "$action" != "skipped" ]]; then
        test_fail "action should be 'skipped' for same-repo duplicate" "got: $action"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_promote_action_merged_for_cross_repo() {
    # Same text from different repo should return "merged"
    local tmpdir
    tmpdir=$(setup_hive_env)

    # First promotion from repo A
    run_hive_promote "$tmpdir" \
        --text "Always validate input before saving" \
        --source-repo "/Users/me/repos/AppA" >/dev/null

    # Second promotion from repo B (same text)
    local result
    result=$(run_hive_promote "$tmpdir" \
        --text "Always validate input before saving" \
        --source-repo "/Users/me/repos/AppB")

    local action
    action=$(echo "$result" | jq -r '.result.action')

    if [[ "$action" != "merged" ]]; then
        test_fail "action should be 'merged' for cross-repo duplicate" "got: $action"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_promote_includes_transformations() {
    # Should include transformations from hive-abstract
    local tmpdir
    tmpdir=$(setup_hive_env)

    local result
    result=$(run_hive_promote "$tmpdir" \
        --text "Error in /Users/foo/repos/bar/src/thing.js when saving" \
        --source-repo "/Users/foo/repos/bar")

    local has_transforms
    has_transforms=$(echo "$result" | jq -e '.result.transformations' >/dev/null 2>&1 && echo yes || echo no)

    if [[ "$has_transforms" != "yes" ]]; then
        test_fail "should include transformations" "not found in result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Should have stripped paths
    local transforms
    transforms=$(echo "$result" | jq -c '.result.transformations')
    if ! assert_contains "$transforms" "path_strip"; then
        test_fail "transformations should include path_strip" "got: $transforms"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_promote_abstracts_text() {
    # The abstracted text should differ from original when repo-specific
    local tmpdir
    tmpdir=$(setup_hive_env)

    local result
    result=$(run_hive_promote "$tmpdir" \
        --text "The MyApp module fails on startup" \
        --source-repo "/Users/me/repos/MyApp")

    local original abstracted
    original=$(echo "$result" | jq -r '.result.original')
    abstracted=$(echo "$result" | jq -r '.result.abstracted')

    # Abstracted should not contain "MyApp"
    if assert_contains "$abstracted" "MyApp"; then
        test_fail "abstracted text should not contain repo name" "got: $abstracted"
        rm -rf "$tmpdir"
        return 1
    fi

    # Abstracted should contain the generic replacement
    if ! assert_contains "$abstracted" "<project>"; then
        test_fail "abstracted text should contain <project>" "got: $abstracted"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_promote_default_confidence() {
    # Default confidence should be 0.7
    local tmpdir
    tmpdir=$(setup_hive_env)

    local result
    result=$(run_hive_promote "$tmpdir" \
        --text "Always validate input" \
        --source-repo "/Users/me/repos/MyApp")

    local confidence
    confidence=$(echo "$result" | jq -r '.result.confidence')

    if [[ "$confidence" != "0.7" ]]; then
        test_fail "default confidence should be 0.7" "got: $confidence"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_promote_custom_confidence() {
    # Should accept custom confidence
    local tmpdir
    tmpdir=$(setup_hive_env)

    local result
    result=$(run_hive_promote "$tmpdir" \
        --text "Always validate input" \
        --source-repo "/Users/me/repos/MyApp" \
        --confidence 0.9)

    local confidence
    confidence=$(echo "$result" | jq -r '.result.confidence')

    if [[ "$confidence" != "0.9" ]]; then
        test_fail "confidence should be 0.9" "got: $confidence"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_promote_custom_category() {
    # Should pass custom category to hive-store
    local tmpdir
    tmpdir=$(setup_hive_env)

    local result
    result=$(run_hive_promote "$tmpdir" \
        --text "Always validate input" \
        --source-repo "/Users/me/repos/MyApp" \
        --category "security")

    if ! assert_ok_true "$result"; then
        test_fail "should succeed with custom category" "got: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify the entry was stored with the correct category by reading back
    local read_result
    read_result=$(HOME="$tmpdir" AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$AETHER_UTILS" hive-read --format json 2>&1)

    local stored_category
    stored_category=$(echo "$read_result" | jq -r '.result.entries[0].category')

    if [[ "$stored_category" != "security" ]]; then
        test_fail "stored category should be 'security'" "got: $stored_category"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_promote_with_domain_tags() {
    # Should pass domain tags through the pipeline
    local tmpdir
    tmpdir=$(setup_hive_env)

    local result
    result=$(run_hive_promote "$tmpdir" \
        --text "Always validate input" \
        --source-repo "/Users/me/repos/MyApp" \
        --domain "web,api")

    if ! assert_ok_true "$result"; then
        test_fail "should succeed with domain tags" "got: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify domain tags stored correctly
    local read_result
    read_result=$(HOME="$tmpdir" AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$AETHER_UTILS" hive-read --domain "web" --format json 2>&1)

    local entry_count
    entry_count=$(echo "$read_result" | jq '.result.entries | length')

    if [[ "$entry_count" -lt 1 ]]; then
        test_fail "should find entry by domain tag" "got $entry_count entries"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_promote_registered_in_help() {
    # hive-promote should appear in the help output
    local tmpdir
    tmpdir=$(setup_hive_env)

    local result
    result=$(HOME="$tmpdir" AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$AETHER_UTILS" help 2>/dev/null)

    # Check flat array
    if ! assert_contains "$result" "hive-promote"; then
        test_fail "hive-promote should be in help commands array" "not found"
        rm -rf "$tmpdir"
        return 1
    fi

    # Check Hive Intelligence section
    local in_section
    in_section=$(echo "$result" | jq '.sections["Hive Intelligence"][] | select(.name == "hive-promote")' 2>/dev/null)

    if [[ -z "$in_section" ]]; then
        test_fail "hive-promote should be in Hive Intelligence section" "not found in section"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_promote_initializes_hive() {
    # Should work even when hive directory does not exist yet
    local tmpdir
    tmpdir=$(setup_hive_env)

    # Ensure no hive directory exists
    rm -rf "$tmpdir/.aether/hive"

    local result
    result=$(run_hive_promote "$tmpdir" \
        --text "Always validate input" \
        --source-repo "/Users/me/repos/MyApp")

    if ! assert_ok_true "$result"; then
        test_fail "should succeed even without pre-existing hive dir" "got: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify hive was created
    if [[ ! -f "$tmpdir/.aether/hive/wisdom.json" ]]; then
        test_fail "hive wisdom.json should exist after promote" "file not found"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Run all tests
# ============================================================================

run_test test_hive_promote_missing_text "hive-promote: fails without --text"
run_test test_hive_promote_missing_source_repo "hive-promote: fails without --source-repo"
run_test test_hive_promote_basic_success "hive-promote: returns correct JSON structure"
run_test test_hive_promote_action_promoted_for_new "hive-promote: action is 'promoted' for new entry"
run_test test_hive_promote_action_skipped_for_duplicate "hive-promote: action is 'skipped' for same-repo duplicate"
run_test test_hive_promote_action_merged_for_cross_repo "hive-promote: action is 'merged' for cross-repo duplicate"
run_test test_hive_promote_includes_transformations "hive-promote: includes transformations from abstract"
run_test test_hive_promote_abstracts_text "hive-promote: abstracts repo-specific text"
run_test test_hive_promote_default_confidence "hive-promote: default confidence is 0.7"
run_test test_hive_promote_custom_confidence "hive-promote: accepts custom confidence"
run_test test_hive_promote_custom_category "hive-promote: passes custom category to store"
run_test test_hive_promote_with_domain_tags "hive-promote: passes domain tags through pipeline"
run_test test_hive_promote_registered_in_help "hive-promote: registered in help"
run_test test_hive_promote_initializes_hive "hive-promote: initializes hive if missing"

test_summary
