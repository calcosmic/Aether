#!/usr/bin/env bash
# Tests for hive-read subcommand
# Task 1.3: Read wisdom entries with domain filtering, access_count tracking

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

# ============================================================================
# Helper: Create a test hive environment with wisdom.json
# ============================================================================
setup_hive_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    local hive_dir="$tmpdir/.aether/hive"
    mkdir -p "$hive_dir"
    echo "$tmpdir"
}

# Helper: run hive subcommands with isolated HOME
run_hive_cmd() {
    local tmpdir="$1"
    shift
    HOME="$tmpdir" bash "$AETHER_UTILS" "$@" 2>/dev/null
}

# Helper: create wisdom.json with test entries
create_test_wisdom() {
    local tmpdir="$1"
    local hive_dir="$tmpdir/.aether/hive"
    mkdir -p "$hive_dir"

    cat > "$hive_dir/wisdom.json" << 'WISEOF'
{
  "version": "1.0.0",
  "created_at": "2026-03-20T00:00:00Z",
  "last_updated": "2026-03-20T00:00:00Z",
  "entries": [
    {
      "id": "aaa111222333",
      "text": "Always validate user input before database queries",
      "category": "security",
      "confidence": 0.95,
      "domain_tags": ["web", "api"],
      "source_repos": ["/repo/alpha"],
      "validated_count": 3,
      "created_at": "2026-03-01T00:00:00Z",
      "last_accessed": "2026-03-15T00:00:00Z",
      "access_count": 5
    },
    {
      "id": "bbb222333444",
      "text": "Use structured logging with correlation IDs",
      "category": "observability",
      "confidence": 0.80,
      "domain_tags": ["api", "backend"],
      "source_repos": ["/repo/beta"],
      "validated_count": 2,
      "created_at": "2026-03-02T00:00:00Z",
      "last_accessed": "2026-03-14T00:00:00Z",
      "access_count": 3
    },
    {
      "id": "ccc333444555",
      "text": "Prefer composition over inheritance",
      "category": "design",
      "confidence": 0.70,
      "domain_tags": ["general"],
      "source_repos": ["/repo/gamma"],
      "validated_count": 1,
      "created_at": "2026-03-03T00:00:00Z",
      "last_accessed": "2026-03-13T00:00:00Z",
      "access_count": 1
    },
    {
      "id": "ddd444555666",
      "text": "Cache database connection pools",
      "category": "performance",
      "confidence": 0.60,
      "domain_tags": ["backend", "database"],
      "source_repos": ["/repo/delta"],
      "validated_count": 1,
      "created_at": "2026-03-04T00:00:00Z",
      "last_accessed": "2026-03-12T00:00:00Z",
      "access_count": 0
    },
    {
      "id": "eee555666777",
      "text": "Use semantic versioning for all packages",
      "category": "general",
      "confidence": 0.50,
      "domain_tags": ["general"],
      "source_repos": ["/repo/epsilon"],
      "validated_count": 1,
      "created_at": "2026-03-05T00:00:00Z",
      "last_accessed": "2026-03-11T00:00:00Z",
      "access_count": 0
    }
  ],
  "metadata": {
    "total_entries": 5,
    "max_entries": 200,
    "contributing_repos": ["/repo/alpha", "/repo/beta", "/repo/gamma", "/repo/delta", "/repo/epsilon"]
  }
}
WISEOF
}

# ============================================================================
# Tests
# ============================================================================

test_hive_read_basic_json() {
    # Basic hive-read with no filters should return all entries in json format
    local tmpdir
    tmpdir=$(setup_hive_env)
    create_test_wisdom "$tmpdir"

    local result
    result=$(run_hive_cmd "$tmpdir" hive-read)

    if ! assert_ok_true "$result"; then
        test_fail "Expected ok=true" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    local total_matched
    total_matched=$(echo "$result" | jq -r '.result.total_matched')
    if [[ "$total_matched" != "5" ]]; then
        test_fail "Expected total_matched=5" "Got $total_matched"
        rm -rf "$tmpdir"
        return 1
    fi

    local entries_count
    entries_count=$(echo "$result" | jq '.result.entries | length')
    if [[ "$entries_count" != "5" ]]; then
        test_fail "Expected 5 entries" "Got $entries_count"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_read_sorted_by_confidence() {
    # Results should be sorted by confidence descending
    local tmpdir
    tmpdir=$(setup_hive_env)
    create_test_wisdom "$tmpdir"

    local result
    result=$(run_hive_cmd "$tmpdir" hive-read)

    if ! assert_ok_true "$result"; then
        test_fail "Expected ok=true" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    # First entry should be highest confidence (0.95)
    local first_conf
    first_conf=$(echo "$result" | jq -r '.result.entries[0].confidence')
    if [[ "$first_conf" != "0.95" ]]; then
        test_fail "Expected first entry confidence=0.95" "Got $first_conf"
        rm -rf "$tmpdir"
        return 1
    fi

    # Last entry should be lowest confidence (0.5)
    local last_conf
    last_conf=$(echo "$result" | jq -r '.result.entries[4].confidence')
    # jq may format as 0.5 or 0.50 — compare numerically
    if ! echo "$last_conf" | awk '{ exit ($1 == 0.5) ? 0 : 1 }'; then
        test_fail "Expected last entry confidence=0.5" "Got $last_conf"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_read_domain_filter() {
    # --domain should filter entries by matching domain_tags
    local tmpdir
    tmpdir=$(setup_hive_env)
    create_test_wisdom "$tmpdir"

    local result
    result=$(run_hive_cmd "$tmpdir" hive-read --domain "web")

    if ! assert_ok_true "$result"; then
        test_fail "Expected ok=true" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    local total_matched
    total_matched=$(echo "$result" | jq -r '.result.total_matched')
    if [[ "$total_matched" != "1" ]]; then
        test_fail "Expected total_matched=1 for domain 'web'" "Got $total_matched"
        rm -rf "$tmpdir"
        return 1
    fi

    # The matching entry should be the security one with domain_tags ["web","api"]
    local matched_id
    matched_id=$(echo "$result" | jq -r '.result.entries[0].id')
    if [[ "$matched_id" != "aaa111222333" ]]; then
        test_fail "Expected entry aaa111222333" "Got $matched_id"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_read_domain_filter_csv() {
    # --domain should accept CSV and match ANY tag
    local tmpdir
    tmpdir=$(setup_hive_env)
    create_test_wisdom "$tmpdir"

    local result
    result=$(run_hive_cmd "$tmpdir" hive-read --domain "web,database")

    if ! assert_ok_true "$result"; then
        test_fail "Expected ok=true" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Should match "web" (aaa111) and "database" (ddd444) entries
    local total_matched
    total_matched=$(echo "$result" | jq -r '.result.total_matched')
    if [[ "$total_matched" != "2" ]]; then
        test_fail "Expected total_matched=2 for domain 'web,database'" "Got $total_matched"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_read_min_confidence() {
    # --min-confidence should filter by confidence threshold
    local tmpdir
    tmpdir=$(setup_hive_env)
    create_test_wisdom "$tmpdir"

    local result
    result=$(run_hive_cmd "$tmpdir" hive-read --min-confidence 0.75)

    if ! assert_ok_true "$result"; then
        test_fail "Expected ok=true" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Should match entries with confidence >= 0.75 (0.95 and 0.80)
    local total_matched
    total_matched=$(echo "$result" | jq -r '.result.total_matched')
    if [[ "$total_matched" != "2" ]]; then
        test_fail "Expected total_matched=2 for min-confidence 0.75" "Got $total_matched"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_read_limit() {
    # --limit should cap returned entries
    local tmpdir
    tmpdir=$(setup_hive_env)
    create_test_wisdom "$tmpdir"

    local result
    result=$(run_hive_cmd "$tmpdir" hive-read --limit 2)

    if ! assert_ok_true "$result"; then
        test_fail "Expected ok=true" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    local entries_count
    entries_count=$(echo "$result" | jq '.result.entries | length')
    if [[ "$entries_count" != "2" ]]; then
        test_fail "Expected 2 entries with --limit 2" "Got $entries_count"
        rm -rf "$tmpdir"
        return 1
    fi

    # total_matched should reflect the pre-limit count
    local total_matched
    total_matched=$(echo "$result" | jq -r '.result.total_matched')
    if [[ "$total_matched" != "5" ]]; then
        test_fail "Expected total_matched=5 (pre-limit count)" "Got $total_matched"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_read_access_count_incremented() {
    # After reading, access_count should be incremented for returned entries
    local tmpdir
    tmpdir=$(setup_hive_env)
    create_test_wisdom "$tmpdir"

    # Read with limit 2 (returns top-2 by confidence)
    run_hive_cmd "$tmpdir" hive-read --limit 2 >/dev/null

    # Check wisdom.json was updated
    local wisdom_file="$tmpdir/.aether/hive/wisdom.json"
    if [[ ! -f "$wisdom_file" ]]; then
        test_fail "wisdom.json should still exist" ""
        rm -rf "$tmpdir"
        return 1
    fi

    # Entry aaa111 (conf 0.95) was returned — access_count was 5, should now be 6
    local access_count_aaa
    access_count_aaa=$(jq '.entries[] | select(.id == "aaa111222333") | .access_count' "$wisdom_file")
    if [[ "$access_count_aaa" != "6" ]]; then
        test_fail "Expected access_count=6 for aaa111222333" "Got $access_count_aaa"
        rm -rf "$tmpdir"
        return 1
    fi

    # Entry bbb222 (conf 0.80) was returned — access_count was 3, should now be 4
    local access_count_bbb
    access_count_bbb=$(jq '.entries[] | select(.id == "bbb222333444") | .access_count' "$wisdom_file")
    if [[ "$access_count_bbb" != "4" ]]; then
        test_fail "Expected access_count=4 for bbb222333444" "Got $access_count_bbb"
        rm -rf "$tmpdir"
        return 1
    fi

    # Entry ccc333 (conf 0.70) was NOT returned — access_count should still be 1
    local access_count_ccc
    access_count_ccc=$(jq '.entries[] | select(.id == "ccc333444555") | .access_count' "$wisdom_file")
    if [[ "$access_count_ccc" != "1" ]]; then
        test_fail "Expected access_count=1 for ccc333444555 (not returned)" "Got $access_count_ccc"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_read_last_accessed_updated() {
    # last_accessed should be updated for returned entries
    local tmpdir
    tmpdir=$(setup_hive_env)
    create_test_wisdom "$tmpdir"

    run_hive_cmd "$tmpdir" hive-read --limit 1 >/dev/null

    local wisdom_file="$tmpdir/.aether/hive/wisdom.json"

    # Entry aaa111 was returned — last_accessed should be updated (not the old date)
    local last_accessed
    last_accessed=$(jq -r '.entries[] | select(.id == "aaa111222333") | .last_accessed' "$wisdom_file")
    if [[ "$last_accessed" == "2026-03-15T00:00:00Z" ]]; then
        test_fail "last_accessed should have been updated from old date" "Still $last_accessed"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_read_no_wisdom_file_fallback() {
    # When wisdom.json doesn't exist, return empty with fallback
    local tmpdir
    tmpdir=$(setup_hive_env)
    # Do NOT create wisdom.json

    local result
    result=$(run_hive_cmd "$tmpdir" hive-read)

    if ! assert_ok_true "$result"; then
        test_fail "Expected ok=true for fallback" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    local total_matched
    total_matched=$(echo "$result" | jq -r '.result.total_matched')
    if [[ "$total_matched" != "0" ]]; then
        test_fail "Expected total_matched=0 for fallback" "Got $total_matched"
        rm -rf "$tmpdir"
        return 1
    fi

    local fallback
    fallback=$(echo "$result" | jq -r '.result.fallback')
    if [[ "$fallback" != "no_hive" ]]; then
        test_fail "Expected fallback=no_hive" "Got $fallback"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_read_text_format() {
    # --format text should produce readable text output
    local tmpdir
    tmpdir=$(setup_hive_env)
    create_test_wisdom "$tmpdir"

    local result
    result=$(run_hive_cmd "$tmpdir" hive-read --format text --limit 2)

    if ! assert_ok_true "$result"; then
        test_fail "Expected ok=true" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    local text_output
    text_output=$(echo "$result" | jq -r '.result.text')

    # Should contain entry text
    if ! assert_contains "$text_output" "validate user input"; then
        test_fail "Expected entry text in text output" "$text_output"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_read_combined_filters() {
    # Combine domain + min-confidence + limit
    local tmpdir
    tmpdir=$(setup_hive_env)
    create_test_wisdom "$tmpdir"

    local result
    result=$(run_hive_cmd "$tmpdir" hive-read --domain "api" --min-confidence 0.75 --limit 10)

    if ! assert_ok_true "$result"; then
        test_fail "Expected ok=true" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    # api entries: aaa111 (conf 0.95), bbb222 (conf 0.80)
    # min-confidence 0.75 keeps both
    local total_matched
    total_matched=$(echo "$result" | jq -r '.result.total_matched')
    if [[ "$total_matched" != "2" ]]; then
        test_fail "Expected total_matched=2 for api + min-confidence 0.75" "Got $total_matched"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_read_string_confidence() {
    # Entries with string confidence values should be handled correctly (tonumber coercion)
    local tmpdir
    tmpdir=$(setup_hive_env)
    local hive_dir="$tmpdir/.aether/hive"
    mkdir -p "$hive_dir"

    # Create wisdom.json with mixed confidence types: number and string
    cat > "$hive_dir/wisdom.json" << 'WISEOF'
{
  "version": "1.0.0",
  "created_at": "2026-03-20T00:00:00Z",
  "last_updated": "2026-03-20T00:00:00Z",
  "entries": [
    {
      "id": "num111222333",
      "text": "Numeric confidence entry",
      "category": "pattern",
      "confidence": 0.9,
      "domain_tags": ["general"],
      "source_repos": ["/repo/alpha"],
      "validated_count": 2,
      "created_at": "2026-03-01T00:00:00Z",
      "last_accessed": "2026-03-15T00:00:00Z",
      "access_count": 1
    },
    {
      "id": "str222333444",
      "text": "String confidence entry",
      "category": "pattern",
      "confidence": "0.8",
      "domain_tags": ["general"],
      "source_repos": ["/repo/beta"],
      "validated_count": 1,
      "created_at": "2026-03-02T00:00:00Z",
      "last_accessed": "2026-03-14T00:00:00Z",
      "access_count": 0
    }
  ],
  "metadata": {
    "total_entries": 2,
    "max_entries": 200,
    "contributing_repos": ["/repo/alpha", "/repo/beta"]
  }
}
WISEOF

    local result
    result=$(run_hive_cmd "$tmpdir" hive-read --min-confidence 0.5)

    if ! assert_ok_true "$result"; then
        test_fail "Expected ok=true" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Both entries should be returned (string "0.8" coerced to number >= 0.5)
    local total_matched
    total_matched=$(echo "$result" | jq -r '.result.total_matched')
    if [[ "$total_matched" != "2" ]]; then
        test_fail "Expected total_matched=2 (both string and number confidence)" "Got $total_matched"
        rm -rf "$tmpdir"
        return 1
    fi

    # Should be sorted correctly: 0.9 before 0.8
    local first_id
    first_id=$(echo "$result" | jq -r '.result.entries[0].id')
    if [[ "$first_id" != "num111222333" ]]; then
        test_fail "Expected first entry to be numeric confidence (0.9)" "Got $first_id"
        rm -rf "$tmpdir"
        return 1
    fi

    local second_id
    second_id=$(echo "$result" | jq -r '.result.entries[1].id')
    if [[ "$second_id" != "str222333444" ]]; then
        test_fail "Expected second entry to be string confidence (0.8)" "Got $second_id"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_read_null_confidence() {
    # Entries with null or missing confidence should be treated as 0 (not cause a jq error)
    local tmpdir
    tmpdir=$(setup_hive_env)
    local hive_dir="$tmpdir/.aether/hive"
    mkdir -p "$hive_dir"

    # Create wisdom.json with one null confidence and one missing confidence field
    cat > "$hive_dir/wisdom.json" << 'WISEOF'
{
  "version": "1.0.0",
  "created_at": "2026-03-20T00:00:00Z",
  "last_updated": "2026-03-20T00:00:00Z",
  "entries": [
    {
      "id": "null111222333",
      "text": "Entry with null confidence field",
      "category": "pattern",
      "confidence": null,
      "domain_tags": ["general"],
      "source_repos": ["/repo/alpha"],
      "validated_count": 1,
      "created_at": "2026-03-01T00:00:00Z",
      "last_accessed": "2026-03-15T00:00:00Z",
      "access_count": 0
    },
    {
      "id": "missing111222",
      "text": "Entry with missing confidence field",
      "category": "pattern",
      "domain_tags": ["general"],
      "source_repos": ["/repo/beta"],
      "validated_count": 1,
      "created_at": "2026-03-02T00:00:00Z",
      "last_accessed": "2026-03-14T00:00:00Z",
      "access_count": 0
    },
    {
      "id": "good111222333",
      "text": "Entry with normal confidence field",
      "category": "pattern",
      "confidence": 0.85,
      "domain_tags": ["general"],
      "source_repos": ["/repo/gamma"],
      "validated_count": 2,
      "created_at": "2026-03-03T00:00:00Z",
      "last_accessed": "2026-03-13T00:00:00Z",
      "access_count": 1
    }
  ],
  "metadata": {
    "total_entries": 3,
    "max_entries": 200,
    "contributing_repos": ["/repo/alpha", "/repo/beta", "/repo/gamma"]
  }
}
WISEOF

    # With default min-confidence (0.0), all three entries should be returned
    local result
    result=$(run_hive_cmd "$tmpdir" hive-read --min-confidence 0.0)

    if ! assert_ok_true "$result"; then
        test_fail "Expected ok=true (not filter_error fallback) for null/missing confidence entries" "$result"
        rm -rf "$tmpdir"
        return 1
    fi

    local total_matched
    total_matched=$(echo "$result" | jq -r '.result.total_matched')
    if [[ "$total_matched" != "3" ]]; then
        test_fail "Expected total_matched=3 (null/missing confidence treated as 0, not filtered)" "Got $total_matched"
        rm -rf "$tmpdir"
        return 1
    fi

    # Null/missing confidence entries should sort last (after the 0.85 entry)
    local first_id
    first_id=$(echo "$result" | jq -r '.result.entries[0].id')
    if [[ "$first_id" != "good111222333" ]]; then
        test_fail "Expected entry with confidence 0.85 to sort first" "Got $first_id"
        rm -rf "$tmpdir"
        return 1
    fi

    # With min-confidence 0.5, null/missing confidence entries (treated as 0) should be excluded
    local filtered_result
    filtered_result=$(run_hive_cmd "$tmpdir" hive-read --min-confidence 0.5)

    if ! assert_ok_true "$filtered_result"; then
        test_fail "Expected ok=true with min-confidence filter on null entries" "$filtered_result"
        rm -rf "$tmpdir"
        return 1
    fi

    local filtered_matched
    filtered_matched=$(echo "$filtered_result" | jq -r '.result.total_matched')
    if [[ "$filtered_matched" != "1" ]]; then
        test_fail "Expected total_matched=1 (only the 0.85 entry passes min-confidence 0.5)" "Got $filtered_matched"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_read_registered_in_help() {
    # hive-read should appear in help output
    local result
    result=$(bash "$AETHER_UTILS" help 2>/dev/null)

    if ! assert_contains "$result" "hive-read"; then
        test_fail "Expected hive-read in help commands" ""
        return 1
    fi

    return 0
}

# ============================================================================
# Run tests
# ============================================================================

log_info "Running hive-read subcommand tests"
log_info "Repo root: $REPO_ROOT"

run_test test_hive_read_basic_json "Basic hive-read returns all entries as JSON"
run_test test_hive_read_sorted_by_confidence "Results sorted by confidence descending"
run_test test_hive_read_domain_filter "Domain filter matches entries"
run_test test_hive_read_domain_filter_csv "Domain filter accepts CSV and matches any"
run_test test_hive_read_min_confidence "Min-confidence filter works"
run_test test_hive_read_limit "Limit caps returned entries"
run_test test_hive_read_access_count_incremented "Access count incremented for returned entries"
run_test test_hive_read_last_accessed_updated "Last accessed timestamp updated"
run_test test_hive_read_no_wisdom_file_fallback "Fallback when wisdom.json missing"
run_test test_hive_read_text_format "Text format produces readable output"
run_test test_hive_read_combined_filters "Combined filters work together"
run_test test_hive_read_string_confidence "hive-read handles string confidence values via tonumber coercion"
run_test test_hive_read_null_confidence "hive-read handles null and missing confidence fields via // 0 fallback"
run_test test_hive_read_registered_in_help "hive-read registered in help"

test_summary
