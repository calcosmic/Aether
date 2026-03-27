#!/usr/bin/env bash
# Integration tests for the full hive abstraction+promotion pipeline
# Task 2.3: Tests hive-abstract -> hive-promote -> hive-read working together
#
# Unlike test-hive-abstract.sh (unit tests for abstraction transforms) and
# test-hive-promote.sh (unit tests for promotion orchestration), this file
# tests the FULL pipeline end-to-end: promote repo-specific instincts, verify
# abstracted storage, cross-repo merging, domain filtering on promoted entries,
# confidence sorting, and access_count tracking.

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

# Helper: run any hive subcommand with isolated HOME
run_hive_cmd() {
    local tmpdir="$1"
    shift
    HOME="$tmpdir" AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$AETHER_UTILS" "$@" 2>&1
}

# ============================================================================
# Test 1: Promote a repo-specific instinct -> verify stored with abstracted text
# ============================================================================

test_promote_stores_abstracted_text() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    local result
    result=$(run_hive_cmd "$tmpdir" hive-promote \
        --text "The MyApp module fails when parsing config" \
        --source-repo "/Users/me/repos/MyApp")

    if ! assert_ok_true "$result"; then
        test_fail "promote should succeed" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    # The returned abstracted text should NOT contain the repo name
    local abstracted
    abstracted=$(echo "$result" | jq -r '.result.abstracted')

    if assert_contains "$abstracted" "MyApp"; then
        test_fail "abstracted text should not contain repo name" "got: $abstracted"
        rm -rf "$tmpdir"
        return 1
    fi

    # Read from the hive and verify the stored text is the abstracted version
    local read_result
    read_result=$(run_hive_cmd "$tmpdir" hive-read --format json)
    local stored_text
    stored_text=$(echo "$read_result" | jq -r '.result.entries[0].text')

    if assert_contains "$stored_text" "MyApp"; then
        test_fail "stored text should be abstracted (no repo name)" "got: $stored_text"
        rm -rf "$tmpdir"
        return 1
    fi

    # Note: hive-store escapes angle brackets for safety, so <project> becomes &lt;project&gt;
    if ! assert_contains "$stored_text" "&lt;project&gt;"; then
        test_fail "stored text should contain escaped <project> placeholder" "got: $stored_text"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 2: Promote with file paths in text -> verify paths stripped in stored entry
# ============================================================================

test_promote_strips_file_paths() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    local result
    result=$(run_hive_cmd "$tmpdir" hive-promote \
        --text "Error in /Users/dev/repos/WebApp/src/utils/auth.js when validating tokens" \
        --source-repo "/Users/dev/repos/WebApp")

    if ! assert_ok_true "$result"; then
        test_fail "promote should succeed" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Read back and verify the stored entry has no file paths
    local read_result
    read_result=$(run_hive_cmd "$tmpdir" hive-read --format json)
    local stored_text
    stored_text=$(echo "$read_result" | jq -r '.result.entries[0].text')

    if assert_contains "$stored_text" "/Users/dev"; then
        test_fail "stored text should not contain file paths" "got: $stored_text"
        rm -rf "$tmpdir"
        return 1
    fi

    # Note: hive-store escapes angle brackets for safety
    if ! assert_contains "$stored_text" "&lt;source-file&gt;"; then
        test_fail "stored text should contain escaped <source-file> placeholder" "got: $stored_text"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 3: Promote with repo name in text -> verify repo name generalized
# ============================================================================

test_promote_generalizes_repo_name() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    local result
    result=$(run_hive_cmd "$tmpdir" hive-promote \
        --text "The AcmeProject API returns 500 errors under load" \
        --source-repo "/Users/dev/repos/AcmeProject")

    if ! assert_ok_true "$result"; then
        test_fail "promote should succeed" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Read back and verify
    local read_result
    read_result=$(run_hive_cmd "$tmpdir" hive-read --format json)
    local stored_text
    stored_text=$(echo "$read_result" | jq -r '.result.entries[0].text')

    if assert_contains "$stored_text" "AcmeProject"; then
        test_fail "stored text should not contain repo name" "got: $stored_text"
        rm -rf "$tmpdir"
        return 1
    fi

    # Note: hive-store escapes angle brackets for safety
    if ! assert_contains "$stored_text" "&lt;project&gt;"; then
        test_fail "stored text should contain escaped <project> placeholder" "got: $stored_text"
        rm -rf "$tmpdir"
        return 1
    fi

    # Core insight should survive
    if ! assert_contains "$stored_text" "API returns 500 errors under load"; then
        test_fail "core insight should survive abstraction" "got: $stored_text"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 4: Promote same instinct from same repo -> verify skip
# ============================================================================

test_promote_same_repo_skips() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    # First promotion
    run_hive_cmd "$tmpdir" hive-promote \
        --text "Always validate user input before processing" \
        --source-repo "/Users/me/repos/Alpha" >/dev/null

    # Second promotion (same text, same repo)
    local result
    result=$(run_hive_cmd "$tmpdir" hive-promote \
        --text "Always validate user input before processing" \
        --source-repo "/Users/me/repos/Alpha")

    local action
    action=$(echo "$result" | jq -r '.result.action')

    if [[ "$action" != "skipped" ]]; then
        test_fail "same-repo duplicate should be skipped" "got action: $action"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify only 1 entry exists
    local read_result
    read_result=$(run_hive_cmd "$tmpdir" hive-read --format json)
    local total
    total=$(echo "$read_result" | jq -r '.result.total_matched')

    if [[ "$total" != "1" ]]; then
        test_fail "should have exactly 1 entry after skip" "got: $total"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 5: Promote same instinct from different repo -> verify merge
# ============================================================================

test_promote_cross_repo_merges() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    # Promote from repo A
    run_hive_cmd "$tmpdir" hive-promote \
        --text "Always validate user input before processing" \
        --source-repo "/Users/me/repos/RepoAlpha" >/dev/null

    # Promote same text from repo B
    local result
    result=$(run_hive_cmd "$tmpdir" hive-promote \
        --text "Always validate user input before processing" \
        --source-repo "/Users/me/repos/RepoBeta")

    local action
    action=$(echo "$result" | jq -r '.result.action')

    if [[ "$action" != "merged" ]]; then
        test_fail "cross-repo duplicate should be merged" "got action: $action"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify only 1 entry exists (merged, not duplicated)
    local read_result
    read_result=$(run_hive_cmd "$tmpdir" hive-read --format json)
    local total
    total=$(echo "$read_result" | jq -r '.result.total_matched')

    if [[ "$total" != "1" ]]; then
        test_fail "should have 1 entry after merge (not 2)" "got: $total"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify validated_count is 2
    local validated
    validated=$(echo "$read_result" | jq -r '.result.entries[0].validated_count')

    if [[ "$validated" != "2" ]]; then
        test_fail "validated_count should be 2 after cross-repo merge" "got: $validated"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 6: Read promoted wisdom with domain filtering
# ============================================================================

test_domain_filtering_on_promoted_entries() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    # Promote entries with different domains
    run_hive_cmd "$tmpdir" hive-promote \
        --text "Use HTTPS for all external API calls" \
        --source-repo "/tmp/repo-a" \
        --domain "security,api" >/dev/null

    run_hive_cmd "$tmpdir" hive-promote \
        --text "Responsive layouts need media queries" \
        --source-repo "/tmp/repo-b" \
        --domain "web,frontend" >/dev/null

    run_hive_cmd "$tmpdir" hive-promote \
        --text "Index foreign keys in database tables" \
        --source-repo "/tmp/repo-c" \
        --domain "database,backend" >/dev/null

    # Filter by security -> should get 1
    local result_security
    result_security=$(run_hive_cmd "$tmpdir" hive-read --domain "security" --format json)
    local security_count
    security_count=$(echo "$result_security" | jq -r '.result.total_matched')

    if [[ "$security_count" != "1" ]]; then
        test_fail "domain=security should return 1 entry" "got: $security_count"
        rm -rf "$tmpdir"
        return 1
    fi

    # Filter by web -> should get 1
    local result_web
    result_web=$(run_hive_cmd "$tmpdir" hive-read --domain "web" --format json)
    local web_count
    web_count=$(echo "$result_web" | jq -r '.result.total_matched')

    if [[ "$web_count" != "1" ]]; then
        test_fail "domain=web should return 1 entry" "got: $web_count"
        rm -rf "$tmpdir"
        return 1
    fi

    # Filter by nonexistent domain -> should get 0
    local result_mobile
    result_mobile=$(run_hive_cmd "$tmpdir" hive-read --domain "mobile" --format json)
    local mobile_count
    mobile_count=$(echo "$result_mobile" | jq -r '.result.total_matched')

    if [[ "$mobile_count" != "0" ]]; then
        test_fail "domain=mobile should return 0 entries" "got: $mobile_count"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 7: Promote with different confidence levels -> verify stored correctly
# ============================================================================

test_promote_preserves_confidence_levels() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    # Promote with high confidence
    run_hive_cmd "$tmpdir" hive-promote \
        --text "High confidence pattern about testing" \
        --source-repo "/tmp/repo-a" \
        --confidence 0.95 >/dev/null

    # Promote with low confidence
    run_hive_cmd "$tmpdir" hive-promote \
        --text "Low confidence pattern about logging" \
        --source-repo "/tmp/repo-b" \
        --confidence 0.4 >/dev/null

    # Promote with default confidence (0.7)
    run_hive_cmd "$tmpdir" hive-promote \
        --text "Default confidence pattern about caching" \
        --source-repo "/tmp/repo-c" >/dev/null

    # Read all and verify confidence levels
    local read_result
    read_result=$(run_hive_cmd "$tmpdir" hive-read --format json)
    local total
    total=$(echo "$read_result" | jq -r '.result.total_matched')

    if [[ "$total" != "3" ]]; then
        test_fail "should have 3 entries" "got: $total"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify high confidence entry is stored
    local high_conf
    high_conf=$(echo "$read_result" | jq '[.result.entries[] | select(.confidence == 0.95)] | length')
    if [[ "$high_conf" != "1" ]]; then
        test_fail "should have 1 entry with confidence=0.95" "got: $high_conf"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify low confidence entry is stored
    local low_conf
    low_conf=$(echo "$read_result" | jq '[.result.entries[] | select(.confidence == 0.4)] | length')
    if [[ "$low_conf" != "1" ]]; then
        test_fail "should have 1 entry with confidence=0.4" "got: $low_conf"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 8: Promote multiple instincts -> verify sorted by confidence in read
# ============================================================================

test_promote_multiple_sorted_by_confidence() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    # Promote entries with varying confidence (out of order)
    run_hive_cmd "$tmpdir" hive-promote \
        --text "Medium priority wisdom about error handling" \
        --source-repo "/tmp/repo-m" \
        --confidence 0.6 >/dev/null

    run_hive_cmd "$tmpdir" hive-promote \
        --text "Top priority wisdom about security" \
        --source-repo "/tmp/repo-t" \
        --confidence 0.99 >/dev/null

    run_hive_cmd "$tmpdir" hive-promote \
        --text "Low priority wisdom about formatting" \
        --source-repo "/tmp/repo-l" \
        --confidence 0.3 >/dev/null

    # Read all -> should be sorted by confidence descending
    local read_result
    read_result=$(run_hive_cmd "$tmpdir" hive-read --format json)

    local first_conf second_conf third_conf
    first_conf=$(echo "$read_result" | jq -r '.result.entries[0].confidence')
    second_conf=$(echo "$read_result" | jq -r '.result.entries[1].confidence')
    third_conf=$(echo "$read_result" | jq -r '.result.entries[2].confidence')

    if [[ "$first_conf" != "0.99" ]]; then
        test_fail "first entry should have highest confidence (0.99)" "got: $first_conf"
        rm -rf "$tmpdir"
        return 1
    fi

    if [[ "$second_conf" != "0.6" ]]; then
        test_fail "second entry should have medium confidence (0.6)" "got: $second_conf"
        rm -rf "$tmpdir"
        return 1
    fi

    if [[ "$third_conf" != "0.3" ]]; then
        test_fail "third entry should have lowest confidence (0.3)" "got: $third_conf"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 9: Full pipeline: init -> promote N entries -> read with filters -> verify access_count
# ============================================================================

test_full_pipeline_with_access_tracking() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    # Explicitly init the hive
    local init_result
    init_result=$(run_hive_cmd "$tmpdir" hive-init)
    if ! assert_ok_true "$init_result"; then
        test_fail "hive-init should succeed" "$init_result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Promote 4 entries with different domains and confidence
    run_hive_cmd "$tmpdir" hive-promote \
        --text "Always sanitize HTML output" \
        --source-repo "/tmp/repo-1" \
        --domain "web,security" \
        --confidence 0.9 >/dev/null

    run_hive_cmd "$tmpdir" hive-promote \
        --text "Use connection pooling for databases" \
        --source-repo "/tmp/repo-2" \
        --domain "database,backend" \
        --confidence 0.85 >/dev/null

    run_hive_cmd "$tmpdir" hive-promote \
        --text "Rate limit public endpoints" \
        --source-repo "/tmp/repo-3" \
        --domain "api,security" \
        --confidence 0.8 >/dev/null

    run_hive_cmd "$tmpdir" hive-promote \
        --text "Cache repeated lookups" \
        --source-repo "/tmp/repo-4" \
        --domain "backend,performance" \
        --confidence 0.7 >/dev/null

    # Read all -> 4 entries
    local all_result
    all_result=$(run_hive_cmd "$tmpdir" hive-read --format json)
    local all_count
    all_count=$(echo "$all_result" | jq -r '.result.total_matched')

    if [[ "$all_count" != "4" ]]; then
        test_fail "should have 4 total entries" "got: $all_count"
        rm -rf "$tmpdir"
        return 1
    fi

    # Filter by security domain -> should get 2 (sanitize HTML + rate limit)
    local sec_result
    sec_result=$(run_hive_cmd "$tmpdir" hive-read --domain "security" --format json)
    local sec_count
    sec_count=$(echo "$sec_result" | jq -r '.result.total_matched')

    if [[ "$sec_count" != "2" ]]; then
        test_fail "security domain should match 2 entries" "got: $sec_count"
        rm -rf "$tmpdir"
        return 1
    fi

    # Filter by min-confidence 0.85 -> should get 2 (0.9 + 0.85)
    local high_result
    high_result=$(run_hive_cmd "$tmpdir" hive-read --min-confidence 0.85 --format json)
    local high_count
    high_count=$(echo "$high_result" | jq -r '.result.total_matched')

    if [[ "$high_count" != "2" ]]; then
        test_fail "min-confidence 0.85 should match 2 entries" "got: $high_count"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify access_count incremented by reads
    local wisdom_file="$tmpdir/.aether/hive/wisdom.json"

    # We did 3 reads above (all, security, high-confidence)
    # Each read increments access_count for matched entries
    # The entry "Always sanitize HTML output" was matched by all 3 reads
    local sanitize_access
    sanitize_access=$(jq '[.entries[] | select(.text | contains("sanitize HTML"))] | .[0].access_count' "$wisdom_file")

    if [[ "$sanitize_access" -lt 3 ]]; then
        test_fail "sanitize HTML entry should have access_count >= 3" "got: $sanitize_access"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 10: Verify hive-promote calls hive-abstract (transformations in result)
# ============================================================================

test_promote_includes_abstraction_transformations() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    # Text with multiple things to abstract: repo name, file path, version, branch
    local result
    result=$(run_hive_cmd "$tmpdir" hive-promote \
        --text "In SuperApp v3.2.1 the /Users/dev/repos/SuperApp/lib/core.rb module on feature/auth-fix leaks memory" \
        --source-repo "/Users/dev/repos/SuperApp")

    if ! assert_ok_true "$result"; then
        test_fail "promote should succeed" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify transformations array is present and contains expected transforms
    local transforms
    transforms=$(echo "$result" | jq -c '.result.transformations')

    if [[ "$transforms" == "null" ]] || [[ "$transforms" == "[]" ]]; then
        test_fail "transformations should not be empty for repo-specific text" "got: $transforms"
        rm -rf "$tmpdir"
        return 1
    fi

    # Check path_strip was applied
    if ! assert_contains "$transforms" "path_strip"; then
        test_fail "transformations should include path_strip" "got: $transforms"
        rm -rf "$tmpdir"
        return 1
    fi

    # Check repo_name_strip was applied
    if ! assert_contains "$transforms" "repo_name_strip"; then
        test_fail "transformations should include repo_name_strip" "got: $transforms"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify the abstracted text doesn't contain repo-specific details
    local abstracted
    abstracted=$(echo "$result" | jq -r '.result.abstracted')

    if assert_contains "$abstracted" "SuperApp"; then
        test_fail "abstracted should not contain repo name" "got: $abstracted"
        rm -rf "$tmpdir"
        return 1
    fi

    if assert_contains "$abstracted" "/Users/dev"; then
        test_fail "abstracted should not contain file paths" "got: $abstracted"
        rm -rf "$tmpdir"
        return 1
    fi

    # Core insight should survive
    if ! assert_contains "$abstracted" "leaks memory"; then
        test_fail "core insight 'leaks memory' should survive" "got: $abstracted"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 11: Verify hive-promote auto-initializes hive (no explicit hive-init needed)
# ============================================================================

test_promote_auto_initializes_hive() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    # Explicitly ensure no hive directory exists
    rm -rf "$tmpdir/.aether/hive"

    # Promote should work without explicit hive-init
    local result
    result=$(run_hive_cmd "$tmpdir" hive-promote \
        --text "Auto-init wisdom about testing" \
        --source-repo "/tmp/auto-init-repo")

    if ! assert_ok_true "$result"; then
        test_fail "promote should auto-initialize hive and succeed" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify hive was created
    if [[ ! -f "$tmpdir/.aether/hive/wisdom.json" ]]; then
        test_fail "wisdom.json should exist after auto-init promote" "file not found"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify the entry was stored
    local read_result
    read_result=$(run_hive_cmd "$tmpdir" hive-read --format json)
    local total
    total=$(echo "$read_result" | jq -r '.result.total_matched')

    if [[ "$total" != "1" ]]; then
        test_fail "should have 1 entry after auto-init promote" "got: $total"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 12: Triple cross-repo merge increments validated_count to 3
# ============================================================================

test_triple_cross_repo_merge() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    # Promote same wisdom from 3 different repos
    run_hive_cmd "$tmpdir" hive-promote \
        --text "Always handle errors gracefully" \
        --source-repo "/tmp/repo-one" >/dev/null

    run_hive_cmd "$tmpdir" hive-promote \
        --text "Always handle errors gracefully" \
        --source-repo "/tmp/repo-two" >/dev/null

    local result
    result=$(run_hive_cmd "$tmpdir" hive-promote \
        --text "Always handle errors gracefully" \
        --source-repo "/tmp/repo-three")

    local action
    action=$(echo "$result" | jq -r '.result.action')

    if [[ "$action" != "merged" ]]; then
        test_fail "third cross-repo promotion should be merged" "got: $action"
        rm -rf "$tmpdir"
        return 1
    fi

    # Read and verify validated_count is 3
    local read_result
    read_result=$(run_hive_cmd "$tmpdir" hive-read --format json)
    local validated
    validated=$(echo "$read_result" | jq -r '.result.entries[0].validated_count')

    if [[ "$validated" != "3" ]]; then
        test_fail "validated_count should be 3 after 3 repos" "got: $validated"
        rm -rf "$tmpdir"
        return 1
    fi

    # Still only 1 entry
    local total
    total=$(echo "$read_result" | jq -r '.result.total_matched')

    if [[ "$total" != "1" ]]; then
        test_fail "should still be only 1 entry after 3 merges" "got: $total"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 13: Promote with no repo-specific content -> text stored unchanged
# ============================================================================

test_promote_generic_text_unchanged() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    local result
    result=$(run_hive_cmd "$tmpdir" hive-promote \
        --text "Always write tests before implementation" \
        --source-repo "/Users/me/repos/SomeProject")

    if ! assert_ok_true "$result"; then
        test_fail "promote should succeed" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    # The text doesn't mention "SomeProject" anywhere, so abstraction is a no-op
    local original abstracted
    original=$(echo "$result" | jq -r '.result.original')
    abstracted=$(echo "$result" | jq -r '.result.abstracted')

    if [[ "$original" != "$abstracted" ]]; then
        test_fail "generic text should survive abstraction unchanged" \
            "original='$original' abstracted='$abstracted'"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Run all tests
# ============================================================================

log_info "Running hive abstraction+promotion pipeline integration tests"
log_info "Repo root: $REPO_ROOT"

run_test test_promote_stores_abstracted_text "Pipeline: promote stores abstracted text (repo name stripped)"
run_test test_promote_strips_file_paths "Pipeline: promote strips file paths in stored entry"
run_test test_promote_generalizes_repo_name "Pipeline: promote generalizes repo name in stored entry"
run_test test_promote_same_repo_skips "Pipeline: same-repo duplicate is skipped"
run_test test_promote_cross_repo_merges "Pipeline: cross-repo duplicate merges (validated_count=2)"
run_test test_domain_filtering_on_promoted_entries "Pipeline: domain filtering works on promoted entries"
run_test test_promote_preserves_confidence_levels "Pipeline: different confidence levels stored correctly"
run_test test_promote_multiple_sorted_by_confidence "Pipeline: read returns entries sorted by confidence desc"
run_test test_full_pipeline_with_access_tracking "Pipeline: full init->promote->read->filter with access_count"
run_test test_promote_includes_abstraction_transformations "Pipeline: promote result includes abstraction transformations"
run_test test_promote_auto_initializes_hive "Pipeline: promote auto-initializes hive without explicit init"
run_test test_triple_cross_repo_merge "Pipeline: triple cross-repo merge increments validated_count to 3"
run_test test_promote_generic_text_unchanged "Pipeline: generic text stored unchanged by abstraction"

test_summary
