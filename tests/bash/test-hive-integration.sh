#!/usr/bin/env bash
# Integration tests for hive-init + hive-store + hive-read pipeline
# Task 1.4: Cross-command integration testing for the full hive workflow

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
        bash "$AETHER_UTILS" "$@" 2>/dev/null
}

# ============================================================================
# Test 1: Init -> Store -> Read end-to-end pipeline
# ============================================================================

test_e2e_init_store_read() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    # Step 1: Init
    local init_result
    init_result=$(run_hive_cmd "$tmpdir" hive-init)
    if ! assert_ok_true "$init_result"; then
        test_fail "hive-init should return ok=true" "$init_result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Step 2: Store
    local store_result
    store_result=$(run_hive_cmd "$tmpdir" hive-store \
        --text "Always write tests before implementation" \
        --domain "testing,tdd" \
        --source-repo "/tmp/project-alpha" \
        --confidence 0.9 \
        --category "patterns")
    if ! assert_ok_true "$store_result"; then
        test_fail "hive-store should return ok=true" "$store_result"
        rm -rf "$tmpdir"
        return 1
    fi
    if ! assert_json_field_equals "$store_result" ".result.action" "stored"; then
        test_fail "Expected action=stored" "$store_result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Step 3: Read
    local read_result
    read_result=$(run_hive_cmd "$tmpdir" hive-read)
    if ! assert_ok_true "$read_result"; then
        test_fail "hive-read should return ok=true" "$read_result"
        rm -rf "$tmpdir"
        return 1
    fi

    local total_matched
    total_matched=$(echo "$read_result" | jq -r '.result.total_matched')
    if [[ "$total_matched" != "1" ]]; then
        test_fail "Expected total_matched=1" "Got $total_matched"
        rm -rf "$tmpdir"
        return 1
    fi

    local entry_text
    entry_text=$(echo "$read_result" | jq -r '.result.entries[0].text')
    if ! assert_contains "$entry_text" "write tests before implementation"; then
        test_fail "Expected stored text in read results" "$entry_text"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 2: Store from multiple repos, read with domain filtering
# ============================================================================

test_multi_repo_store_domain_read() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    run_hive_cmd "$tmpdir" hive-init >/dev/null

    # Store from repo-a with web domain
    run_hive_cmd "$tmpdir" hive-store \
        --text "Use responsive design for all pages" \
        --domain "web,frontend" \
        --source-repo "/tmp/repo-a" \
        --confidence 0.85 \
        --category "design" >/dev/null

    # Store from repo-b with api domain
    run_hive_cmd "$tmpdir" hive-store \
        --text "Rate-limit all public API endpoints" \
        --domain "api,security" \
        --source-repo "/tmp/repo-b" \
        --confidence 0.92 \
        --category "security" >/dev/null

    # Store from repo-c with backend domain
    run_hive_cmd "$tmpdir" hive-store \
        --text "Use connection pooling for database access" \
        --domain "backend,database" \
        --source-repo "/tmp/repo-c" \
        --confidence 0.78 \
        --category "performance" >/dev/null

    # Read with domain=web -> should get 1 result
    local result_web
    result_web=$(run_hive_cmd "$tmpdir" hive-read --domain "web")
    local web_count
    web_count=$(echo "$result_web" | jq -r '.result.total_matched')
    if [[ "$web_count" != "1" ]]; then
        test_fail "Expected 1 entry for domain=web" "Got $web_count"
        rm -rf "$tmpdir"
        return 1
    fi

    # Read with domain=api -> should get 1 result
    local result_api
    result_api=$(run_hive_cmd "$tmpdir" hive-read --domain "api")
    local api_count
    api_count=$(echo "$result_api" | jq -r '.result.total_matched')
    if [[ "$api_count" != "1" ]]; then
        test_fail "Expected 1 entry for domain=api" "Got $api_count"
        rm -rf "$tmpdir"
        return 1
    fi

    # Read with domain=web,api -> should get 2 results (OR matching)
    local result_multi
    result_multi=$(run_hive_cmd "$tmpdir" hive-read --domain "web,api")
    local multi_count
    multi_count=$(echo "$result_multi" | jq -r '.result.total_matched')
    if [[ "$multi_count" != "2" ]]; then
        test_fail "Expected 2 entries for domain=web,api" "Got $multi_count"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 3: Same-repo duplicate -> skip
# ============================================================================

test_same_repo_duplicate_skip() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    run_hive_cmd "$tmpdir" hive-init >/dev/null

    # Store first
    run_hive_cmd "$tmpdir" hive-store \
        --text "Always use parameterized queries" \
        --source-repo "/tmp/same-repo" \
        --category "security" >/dev/null

    # Same text, same repo -> should skip
    local result
    result=$(run_hive_cmd "$tmpdir" hive-store \
        --text "Always use parameterized queries" \
        --source-repo "/tmp/same-repo" \
        --category "security")

    if ! assert_json_field_equals "$result" ".result.action" "skipped"; then
        test_fail "Expected action=skipped for same-repo duplicate" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify only 1 entry exists
    local read_result
    read_result=$(run_hive_cmd "$tmpdir" hive-read)
    local count
    count=$(echo "$read_result" | jq -r '.result.total_matched')
    if [[ "$count" != "1" ]]; then
        test_fail "Expected 1 entry after skip" "Got $count"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 4: Cross-repo duplicate -> merge (validated_count increments)
# ============================================================================

test_cross_repo_duplicate_merge() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    run_hive_cmd "$tmpdir" hive-init >/dev/null

    # Store from repo-a
    run_hive_cmd "$tmpdir" hive-store \
        --text "Validate all user inputs" \
        --source-repo "/tmp/repo-a" \
        --confidence 0.7 \
        --category "security" >/dev/null

    # Same text, different repo -> should merge
    local merge_result
    merge_result=$(run_hive_cmd "$tmpdir" hive-store \
        --text "Validate all user inputs" \
        --source-repo "/tmp/repo-b" \
        --confidence 0.7 \
        --category "security")

    if ! assert_json_field_equals "$merge_result" ".result.action" "merged"; then
        test_fail "Expected action=merged for cross-repo duplicate" "$merge_result"
        rm -rf "$tmpdir"
        return 1
    fi
    if ! assert_json_field_equals "$merge_result" ".result.validated_count" "2"; then
        test_fail "Expected validated_count=2 after merge" "$merge_result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Read and verify the merged entry
    local read_result
    read_result=$(run_hive_cmd "$tmpdir" hive-read)
    local entry_count
    entry_count=$(echo "$read_result" | jq -r '.result.total_matched')
    if [[ "$entry_count" != "1" ]]; then
        test_fail "Expected 1 entry after merge (not 2)" "Got $entry_count"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify validated_count is 2 in the read result
    local validated
    validated=$(echo "$read_result" | jq -r '.result.entries[0].validated_count')
    if [[ "$validated" != "2" ]]; then
        test_fail "Expected validated_count=2 in read result" "Got $validated"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 5: Read with --min-confidence after storing varied confidence entries
# ============================================================================

test_min_confidence_filtering() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    run_hive_cmd "$tmpdir" hive-init >/dev/null

    # Store entries with different confidence levels
    run_hive_cmd "$tmpdir" hive-store --text "Low confidence wisdom" \
        --source-repo "/tmp/repo" --confidence 0.3 --category "general" >/dev/null
    run_hive_cmd "$tmpdir" hive-store --text "Medium confidence wisdom" \
        --source-repo "/tmp/repo" --confidence 0.6 --category "general" >/dev/null
    run_hive_cmd "$tmpdir" hive-store --text "High confidence wisdom" \
        --source-repo "/tmp/repo" --confidence 0.9 --category "general" >/dev/null

    # Read with min-confidence 0.5 -> should get 2 entries
    local result
    result=$(run_hive_cmd "$tmpdir" hive-read --min-confidence 0.5)
    local matched
    matched=$(echo "$result" | jq -r '.result.total_matched')
    if [[ "$matched" != "2" ]]; then
        test_fail "Expected 2 entries with min-confidence=0.5" "Got $matched"
        rm -rf "$tmpdir"
        return 1
    fi

    # Read with min-confidence 0.8 -> should get 1 entry
    result=$(run_hive_cmd "$tmpdir" hive-read --min-confidence 0.8)
    matched=$(echo "$result" | jq -r '.result.total_matched')
    if [[ "$matched" != "1" ]]; then
        test_fail "Expected 1 entry with min-confidence=0.8" "Got $matched"
        rm -rf "$tmpdir"
        return 1
    fi

    # The entry with highest confidence should be the high-confidence one
    local top_text
    top_text=$(echo "$result" | jq -r '.result.entries[0].text')
    if ! assert_contains "$top_text" "High confidence"; then
        test_fail "Expected high confidence entry" "$top_text"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 6: Read with --limit after storing more entries
# ============================================================================

test_limit_after_multiple_stores() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    run_hive_cmd "$tmpdir" hive-init >/dev/null

    # Store 5 entries
    for i in 1 2 3 4 5; do
        run_hive_cmd "$tmpdir" hive-store \
            --text "Wisdom entry number $i for limit test" \
            --source-repo "/tmp/repo-$i" \
            --confidence "0.${i}0" \
            --category "testing" >/dev/null
    done

    # Read with limit 3
    local result
    result=$(run_hive_cmd "$tmpdir" hive-read --limit 3)

    local entries_count
    entries_count=$(echo "$result" | jq '.result.entries | length')
    if [[ "$entries_count" != "3" ]]; then
        test_fail "Expected 3 entries with --limit 3" "Got $entries_count"
        rm -rf "$tmpdir"
        return 1
    fi

    # total_matched should reflect all 5 (pre-limit count)
    local total
    total=$(echo "$result" | jq -r '.result.total_matched')
    if [[ "$total" != "5" ]]; then
        test_fail "Expected total_matched=5 (pre-limit)" "Got $total"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 7: Read updates access_count — verify with second read
# ============================================================================

test_read_increments_access_count() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    run_hive_cmd "$tmpdir" hive-init >/dev/null

    run_hive_cmd "$tmpdir" hive-store \
        --text "Track access patterns carefully" \
        --source-repo "/tmp/repo" \
        --confidence 0.85 \
        --category "observability" >/dev/null

    # First read
    run_hive_cmd "$tmpdir" hive-read >/dev/null

    # Check access_count in the file
    local wisdom_file="$tmpdir/.aether/hive/wisdom.json"
    local count_after_first
    count_after_first=$(jq '.entries[0].access_count' "$wisdom_file")
    if [[ "$count_after_first" != "1" ]]; then
        test_fail "Expected access_count=1 after first read" "Got $count_after_first"
        rm -rf "$tmpdir"
        return 1
    fi

    # Second read
    run_hive_cmd "$tmpdir" hive-read >/dev/null

    local count_after_second
    count_after_second=$(jq '.entries[0].access_count' "$wisdom_file")
    if [[ "$count_after_second" != "2" ]]; then
        test_fail "Expected access_count=2 after second read" "Got $count_after_second"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 8: Read with --format text produces readable output
# ============================================================================

test_text_format_after_store() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    run_hive_cmd "$tmpdir" hive-init >/dev/null

    run_hive_cmd "$tmpdir" hive-store \
        --text "Keep functions under 50 lines" \
        --domain "general,code-quality" \
        --source-repo "/tmp/repo" \
        --confidence 0.88 \
        --category "patterns" >/dev/null

    local result
    result=$(run_hive_cmd "$tmpdir" hive-read --format text)

    if ! assert_ok_true "$result"; then
        test_fail "Expected ok=true" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    local text_output
    text_output=$(echo "$result" | jq -r '.result.text')

    # Should contain the entry text
    if ! assert_contains "$text_output" "functions under 50 lines"; then
        test_fail "Expected entry text in text output" "$text_output"
        rm -rf "$tmpdir"
        return 1
    fi

    # Should contain confidence
    if ! assert_contains "$text_output" "0.88"; then
        test_fail "Expected confidence in text output" "$text_output"
        rm -rf "$tmpdir"
        return 1
    fi

    # Should contain category
    if ! assert_contains "$text_output" "patterns"; then
        test_fail "Expected category in text output" "$text_output"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 9: 200-entry cap enforcement via store pipeline
# ============================================================================

test_200_entry_cap_via_store() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    run_hive_cmd "$tmpdir" hive-init >/dev/null

    # Pre-populate 200 entries directly for speed
    local wisdom_file="$tmpdir/.aether/hive/wisdom.json"
    python3 -c "
import json, datetime

entries = []
for i in range(200):
    padded = str(i).zfill(3)
    ts = f'2026-01-01T00:{padded[:2]}:{padded[2:]}Z' if int(padded[:2]) < 60 and int(padded[2:]) < 60 else '2026-01-01T00:00:00Z'
    entries.append({
        'id': f'cap_{padded}',
        'text': f'Cap test entry {padded}',
        'category': 'cap-test',
        'confidence': 0.5,
        'domain_tags': [],
        'source_repos': [f'/tmp/repo-{padded}'],
        'validated_count': 1,
        'created_at': ts,
        'last_accessed': ts,
        'access_count': 0
    })

wisdom = {
    'version': '1.0.0',
    'created_at': '2026-01-01T00:00:00Z',
    'last_updated': '2026-01-01T00:00:00Z',
    'entries': entries,
    'metadata': {
        'total_entries': 200,
        'max_entries': 200,
        'contributing_repos': [e['source_repos'][0] for e in entries]
    }
}
with open('$wisdom_file', 'w') as f:
    json.dump(wisdom, f)
"

    # Store entry 201
    local store_result
    store_result=$(run_hive_cmd "$tmpdir" hive-store \
        --text "Brand new entry that should survive eviction" \
        --source-repo "/tmp/repo-201" \
        --category "overflow")

    if ! assert_json_field_equals "$store_result" ".result.action" "stored"; then
        test_fail "Expected action=stored for entry 201" "$store_result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Read all entries and verify count is 200
    local read_result
    read_result=$(run_hive_cmd "$tmpdir" hive-read --limit 200)
    local total
    total=$(echo "$read_result" | jq -r '.result.total_matched')
    if [[ "$total" != "200" ]]; then
        test_fail "Expected 200 entries after cap enforcement" "Got $total"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify the new entry is present in the file
    local new_entry_exists
    new_entry_exists=$(jq '[.entries[] | select(.text == "Brand new entry that should survive eviction")] | length' "$wisdom_file")
    if [[ "$new_entry_exists" != "1" ]]; then
        test_fail "New entry should be present after eviction" "Found $new_entry_exists"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 10: Init idempotency — init, store data, init again -> data preserved
# ============================================================================

test_init_preserves_stored_data() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    # First init
    run_hive_cmd "$tmpdir" hive-init >/dev/null

    # Store an entry
    run_hive_cmd "$tmpdir" hive-store \
        --text "Precious wisdom that must survive re-init" \
        --source-repo "/tmp/repo" \
        --category "critical" >/dev/null

    # Second init (should be idempotent)
    local reinit_result
    reinit_result=$(run_hive_cmd "$tmpdir" hive-init)
    if ! assert_json_field_equals "$reinit_result" ".result.already_existed" "true"; then
        test_fail "Expected already_existed=true on re-init" "$reinit_result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Read — the stored entry should still be there
    local read_result
    read_result=$(run_hive_cmd "$tmpdir" hive-read)
    local total
    total=$(echo "$read_result" | jq -r '.result.total_matched')
    if [[ "$total" != "1" ]]; then
        test_fail "Expected 1 entry after re-init (data preserved)" "Got $total"
        rm -rf "$tmpdir"
        return 1
    fi

    local entry_text
    entry_text=$(echo "$read_result" | jq -r '.result.entries[0].text')
    if ! assert_contains "$entry_text" "Precious wisdom"; then
        test_fail "Stored entry text should survive re-init" "$entry_text"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 11: Read from empty hive (after init but no stores)
# ============================================================================

test_read_from_empty_hive() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    run_hive_cmd "$tmpdir" hive-init >/dev/null

    # Read with no entries stored
    local result
    result=$(run_hive_cmd "$tmpdir" hive-read)

    if ! assert_ok_true "$result"; then
        test_fail "Expected ok=true for empty hive" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    local total
    total=$(echo "$result" | jq -r '.result.total_matched')
    if [[ "$total" != "0" ]]; then
        test_fail "Expected total_matched=0 for empty hive" "Got $total"
        rm -rf "$tmpdir"
        return 1
    fi

    local entries_len
    entries_len=$(echo "$result" | jq '.result.entries | length')
    if [[ "$entries_len" != "0" ]]; then
        test_fail "Expected 0 entries for empty hive" "Got $entries_len"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 12: Read with domain that matches nothing -> empty results
# ============================================================================

test_read_domain_no_match() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    run_hive_cmd "$tmpdir" hive-init >/dev/null

    # Store entries with specific domains
    run_hive_cmd "$tmpdir" hive-store \
        --text "Only relevant to web" \
        --domain "web" \
        --source-repo "/tmp/repo" \
        --category "general" >/dev/null

    run_hive_cmd "$tmpdir" hive-store \
        --text "Only relevant to api" \
        --domain "api" \
        --source-repo "/tmp/repo" \
        --category "general" >/dev/null

    # Read with a domain that exists in no entry
    local result
    result=$(run_hive_cmd "$tmpdir" hive-read --domain "mobile")

    if ! assert_ok_true "$result"; then
        test_fail "Expected ok=true for no-match domain" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    local total
    total=$(echo "$result" | jq -r '.result.total_matched')
    if [[ "$total" != "0" ]]; then
        test_fail "Expected total_matched=0 for non-matching domain" "Got $total"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Run tests
# ============================================================================

log_info "Running hive integration tests (init + store + read pipeline)"
log_info "Repo root: $REPO_ROOT"

run_test test_e2e_init_store_read "E2E: init -> store -> read pipeline"
run_test test_multi_repo_store_domain_read "Multi-repo store with domain filtering"
run_test test_same_repo_duplicate_skip "Same-repo duplicate is skipped"
run_test test_cross_repo_duplicate_merge "Cross-repo duplicate merges (validated_count)"
run_test test_min_confidence_filtering "Min-confidence filtering after varied stores"
run_test test_limit_after_multiple_stores "Limit caps entries after multiple stores"
run_test test_read_increments_access_count "Read increments access_count across calls"
run_test test_text_format_after_store "Text format output after real store"
run_test test_200_entry_cap_via_store "200-entry cap enforcement via store pipeline"
run_test test_init_preserves_stored_data "Init idempotency preserves stored data"
run_test test_read_from_empty_hive "Read from empty hive returns empty results"
run_test test_read_domain_no_match "Domain filter with no matches returns empty"

test_summary
