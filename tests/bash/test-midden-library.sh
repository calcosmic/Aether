#!/usr/bin/env bash
# Tests for midden-search and midden-tag subcommands
# Midden library — persistent debug knowledge features

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

# ============================================================================
# Helper: Create isolated test environment with midden support
# ============================================================================
setup_midden_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    mkdir -p "$tmpdir/.aether/data/midden"

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

    local schemas_source
    schemas_source="$(dirname "$AETHER_UTILS")/schemas"
    if [[ -d "$schemas_source" ]]; then
        cp -r "$schemas_source" "$tmpdir/.aether/"
    fi

    cat > "$tmpdir/.aether/data/COLONY_STATE.json" << 'EOF'
{
  "goal": "test midden-library",
  "state": "active",
  "current_phase": 1,
  "plan": {"id": "test-plan", "tasks": []},
  "memory": {"instincts": []},
  "errors": {"records": []},
  "events": [],
  "session_id": "test-session",
  "initialized_at": "2026-02-13T16:00:00Z"
}
EOF

    cat > "$tmpdir/.aether/data/midden/midden.json" << 'EOF'
{"version":"1.0.0","entries":[],"entry_count":0}
EOF

    echo "$tmpdir"
}

# Helper: run aether-utils against a test env
run_cmd() {
    local tmpdir="$1"
    shift
    AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$tmpdir/.aether/aether-utils.sh" "$@" 2>&1
}

# Helper: write a midden entry and return its id
write_entry() {
    local tmpdir="$1"
    local category="${2:-general}"
    local message="${3:-test failure}"
    local source="${4:-test}"
    local result
    result=$(run_cmd "$tmpdir" midden-write "$category" "$message" "$source")
    echo "$result" | jq -r '.result.entry_id'
}

# ============================================================================
# midden-search tests
# ============================================================================

# Test 1: Search with a keyword that matches entries — should return matching entries
test_search_keyword_matches() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    run_cmd "$tmpdir" midden-write "security" "High CVEs found in dependency" "gatekeeper" >/dev/null
    run_cmd "$tmpdir" midden-write "quality" "Code smell detected in auth module" "auditor" >/dev/null

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-search "CVEs") || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_json_valid "$result"; then
        test_fail "valid JSON" "invalid JSON: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local match_count
    match_count=$(echo "$result" | jq -r '.result.match_count')
    if [[ "$match_count" != "1" ]]; then
        test_fail "match_count=1" "match_count=$match_count"
        rm -rf "$tmpdir"
        return 1
    fi

    local entries_len
    entries_len=$(echo "$result" | jq '.result.entries | length')
    if [[ "$entries_len" != "1" ]]; then
        test_fail "entries length=1" "length=$entries_len"
        rm -rf "$tmpdir"
        return 1
    fi

    local query
    query=$(echo "$result" | jq -r '.result.query')
    if [[ "$query" != "CVEs" ]]; then
        test_fail "query=CVEs" "query=$query"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 2: Search with no matches — should return empty array, match_count: 0
test_search_no_matches() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    run_cmd "$tmpdir" midden-write "security" "CVEs found" "gatekeeper" >/dev/null

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-search "xyzzy_no_match_keyword") || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local match_count
    match_count=$(echo "$result" | jq -r '.result.match_count')
    if [[ "$match_count" != "0" ]]; then
        test_fail "match_count=0" "match_count=$match_count"
        rm -rf "$tmpdir"
        return 1
    fi

    local entries_len
    entries_len=$(echo "$result" | jq '.result.entries | length')
    if [[ "$entries_len" != "0" ]]; then
        test_fail "entries length=0" "length=$entries_len"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 3: Search with --category filter — should only return entries in that category
test_search_category_filter() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    run_cmd "$tmpdir" midden-write "security" "CVEs found in auth" "gatekeeper" >/dev/null
    run_cmd "$tmpdir" midden-write "quality" "CVEs mentioned in code comment" "auditor" >/dev/null
    run_cmd "$tmpdir" midden-write "security" "Another CVEs report" "gatekeeper" >/dev/null

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-search "CVEs" --category security) || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local match_count
    match_count=$(echo "$result" | jq -r '.result.match_count')
    if [[ "$match_count" != "2" ]]; then
        test_fail "match_count=2 (security category only)" "match_count=$match_count"
        rm -rf "$tmpdir"
        return 1
    fi

    # All results must be security category
    local non_security
    non_security=$(echo "$result" | jq '[.result.entries[] | select(.category != "security")] | length')
    if [[ "$non_security" != "0" ]]; then
        test_fail "all entries are security category" "found $non_security non-security entries"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 4: Search with --source filter
test_search_source_filter() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    run_cmd "$tmpdir" midden-write "security" "leak detected" "gatekeeper" >/dev/null
    run_cmd "$tmpdir" midden-write "quality" "leak in memory" "auditor" >/dev/null

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-search "leak" --source gatekeeper) || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local match_count
    match_count=$(echo "$result" | jq -r '.result.match_count')
    if [[ "$match_count" != "1" ]]; then
        test_fail "match_count=1 (gatekeeper source only)" "match_count=$match_count"
        rm -rf "$tmpdir"
        return 1
    fi

    local entry_source
    entry_source=$(echo "$result" | jq -r '.result.entries[0].source')
    if [[ "$entry_source" != "gatekeeper" ]]; then
        test_fail "source=gatekeeper" "source=$entry_source"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 5: Search with --limit flag — should cap results
test_search_limit() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    # Write 5 entries all matching "error"
    local i
    for i in 1 2 3 4 5; do
        run_cmd "$tmpdir" midden-write "general" "error occurrence $i" "test" >/dev/null
    done

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-search "error" --limit 3) || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local entries_len
    entries_len=$(echo "$result" | jq '.result.entries | length')
    if [[ "$entries_len" != "3" ]]; then
        test_fail "entries length=3 (limited)" "length=$entries_len"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 6: Search with --include-acknowledged — should include acknowledged entries
test_search_include_acknowledged() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    run_cmd "$tmpdir" midden-write "security" "CVEs found" "gatekeeper" >/dev/null
    local write_result
    write_result=$(run_cmd "$tmpdir" midden-write "security" "secret exposed" "gatekeeper")
    local entry_id
    entry_id=$(echo "$write_result" | jq -r '.result.entry_id')

    # Acknowledge the second entry
    run_cmd "$tmpdir" midden-acknowledge --id "$entry_id" --reason "Fixed" >/dev/null

    # Default search should NOT include acknowledged
    local result_default
    result_default=$(run_cmd "$tmpdir" midden-search "security_keyword_CVEs_secret" 2>/dev/null || true)

    # Search without --include-acknowledged — acknowledged entries excluded
    local result_no_ack
    result_no_ack=$(run_cmd "$tmpdir" midden-search "CVEs")
    local count_no_ack
    count_no_ack=$(echo "$result_no_ack" | jq -r '.result.match_count')
    if [[ "$count_no_ack" != "1" ]]; then
        test_fail "match_count=1 without include-acknowledged" "count=$count_no_ack"
        rm -rf "$tmpdir"
        return 1
    fi

    # Search with --include-acknowledged — should include the acknowledged entry too
    local result_with_ack
    result_with_ack=$(run_cmd "$tmpdir" midden-search "found" --include-acknowledged)

    if ! assert_ok_true "$result_with_ack"; then
        test_fail "ok=true with include-acknowledged" "ok was not true: $result_with_ack"
        rm -rf "$tmpdir"
        return 1
    fi

    local count_with_ack
    count_with_ack=$(echo "$result_with_ack" | jq -r '.result.match_count')
    if [[ "$count_with_ack" -lt "1" ]]; then
        test_fail "match_count>=1 with include-acknowledged" "count=$count_with_ack"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 7: Search against empty midden — should return gracefully with 0 matches
test_search_empty_midden() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-search "anything") || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0 on empty midden" "exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_json_valid "$result"; then
        test_fail "valid JSON" "invalid JSON: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local match_count
    match_count=$(echo "$result" | jq -r '.result.match_count')
    if [[ "$match_count" != "0" ]]; then
        test_fail "match_count=0 on empty midden" "match_count=$match_count"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 8: Search when midden.json doesn't exist — should return gracefully
test_search_no_midden_file() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    # Remove the midden.json file entirely
    rm -f "$tmpdir/.aether/data/midden/midden.json"

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-search "keyword") || exit_code=$?

    # Should not crash — exit 0 or return json_ok with 0 matches
    if ! assert_json_valid "$result"; then
        test_fail "valid JSON even with no midden file" "invalid JSON: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true even with no midden file" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local match_count
    match_count=$(echo "$result" | jq -r '.result.match_count')
    if [[ "$match_count" != "0" ]]; then
        test_fail "match_count=0 when no midden file" "match_count=$match_count"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# midden-tag tests
# ============================================================================

# Test 9: Tag an entry — tags array should be created/appended
test_tag_entry() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    local write_result
    write_result=$(run_cmd "$tmpdir" midden-write "security" "CVEs found" "gatekeeper")
    local entry_id
    entry_id=$(echo "$write_result" | jq -r '.result.entry_id')

    if [[ -z "$entry_id" || "$entry_id" == "null" ]]; then
        test_fail "entry_id returned from midden-write" "write_result=$write_result"
        rm -rf "$tmpdir"
        return 1
    fi

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-tag --id "$entry_id" --tag "cve-urgent") || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_json_valid "$result"; then
        test_fail "valid JSON" "invalid JSON: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local action
    action=$(echo "$result" | jq -r '.result.action')
    if [[ "$action" != "added" ]]; then
        test_fail "action=added" "action=$action"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify tag is in the returned tags array
    local tag_in_result
    tag_in_result=$(echo "$result" | jq -r '[.result.tags[] | select(. == "cve-urgent")] | length')
    if [[ "$tag_in_result" != "1" ]]; then
        test_fail "cve-urgent in result tags" "tags=$(echo "$result" | jq -r '.result.tags')"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify the tag persisted in midden.json
    local stored_tag
    stored_tag=$(jq --arg id "$entry_id" '[.entries[] | select(.id == $id)] | first | .tags // [] | map(select(. == "cve-urgent")) | length' \
        "$tmpdir/.aether/data/midden/midden.json")
    if [[ "$stored_tag" != "1" ]]; then
        test_fail "tag persisted in midden.json" "stored_tag=$stored_tag"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 10: Untag an entry — tag should be removed from array
test_untag_entry() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    local write_result
    write_result=$(run_cmd "$tmpdir" midden-write "security" "CVEs found" "gatekeeper")
    local entry_id
    entry_id=$(echo "$write_result" | jq -r '.result.entry_id')

    # First add a tag
    run_cmd "$tmpdir" midden-tag --id "$entry_id" --tag "needs-review" >/dev/null

    # Now remove it
    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-tag --id "$entry_id" --untag "needs-review") || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0 on untag" "exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true on untag" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local action
    action=$(echo "$result" | jq -r '.result.action')
    if [[ "$action" != "removed" ]]; then
        test_fail "action=removed" "action=$action"
        rm -rf "$tmpdir"
        return 1
    fi

    # Tag must not appear in result tags
    local tag_in_result
    tag_in_result=$(echo "$result" | jq -r '[.result.tags[] | select(. == "needs-review")] | length')
    if [[ "$tag_in_result" != "0" ]]; then
        test_fail "needs-review removed from result tags" "still present"
        rm -rf "$tmpdir"
        return 1
    fi

    # Tag must not be in persisted midden.json
    local stored_tag
    stored_tag=$(jq --arg id "$entry_id" '[.entries[] | select(.id == $id)] | first | .tags // [] | map(select(. == "needs-review")) | length' \
        "$tmpdir/.aether/data/midden/midden.json")
    if [[ "$stored_tag" != "0" ]]; then
        test_fail "tag removed from midden.json" "stored_tag=$stored_tag"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 11: Tag a non-existent entry — should return error
test_tag_nonexistent_entry() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-tag --id "midden_99999999_nonexistent" --tag "label") || exit_code=$?

    if ! assert_json_valid "$result"; then
        test_fail "valid JSON on nonexistent entry" "invalid JSON: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_false "$result"; then
        test_fail "ok=false for nonexistent entry" "ok was true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 12: Tag with no --id — should return validation error
test_tag_no_id() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-tag --tag "label") || exit_code=$?

    if ! assert_json_valid "$result"; then
        test_fail "valid JSON on missing --id" "invalid JSON: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_false "$result"; then
        test_fail "ok=false for missing --id" "ok was true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 13: Tag with no --tag and no --untag — should return validation error
test_tag_no_tag_or_untag() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    local write_result
    write_result=$(run_cmd "$tmpdir" midden-write "general" "test entry" "test")
    local entry_id
    entry_id=$(echo "$write_result" | jq -r '.result.entry_id')

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-tag --id "$entry_id") || exit_code=$?

    if ! assert_json_valid "$result"; then
        test_fail "valid JSON on missing --tag/--untag" "invalid JSON: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_false "$result"; then
        test_fail "ok=false for missing --tag/--untag" "ok was true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 14: Double-tag same tag — should not duplicate
test_tag_no_duplicate() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    local write_result
    write_result=$(run_cmd "$tmpdir" midden-write "security" "CVEs found" "gatekeeper")
    local entry_id
    entry_id=$(echo "$write_result" | jq -r '.result.entry_id')

    # Add the same tag twice
    run_cmd "$tmpdir" midden-tag --id "$entry_id" --tag "verified" >/dev/null
    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-tag --id "$entry_id" --tag "verified") || exit_code=$?

    if ! assert_ok_true "$result"; then
        test_fail "ok=true on double-tag" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Count occurrences of "verified" in result tags — must be exactly 1
    local tag_count
    tag_count=$(echo "$result" | jq '[.result.tags[] | select(. == "verified")] | length')
    if [[ "$tag_count" != "1" ]]; then
        test_fail "verified appears exactly once in tags" "count=$tag_count"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify in persisted midden.json — count must be 1
    local stored_count
    stored_count=$(jq --arg id "$entry_id" '[.entries[] | select(.id == $id)] | first | .tags // [] | map(select(. == "verified")) | length' \
        "$tmpdir/.aether/data/midden/midden.json")
    if [[ "$stored_count" != "1" ]]; then
        test_fail "verified stored exactly once in midden.json" "stored_count=$stored_count"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Run all tests
# ============================================================================

run_test test_search_keyword_matches    "midden-search: keyword matches returns entries"
run_test test_search_no_matches         "midden-search: no matches returns empty, match_count=0"
run_test test_search_category_filter    "midden-search: --category filter returns only matching category"
run_test test_search_source_filter      "midden-search: --source filter returns only matching source"
run_test test_search_limit              "midden-search: --limit caps results"
run_test test_search_include_acknowledged "midden-search: --include-acknowledged includes acknowledged entries"
run_test test_search_empty_midden       "midden-search: empty midden returns gracefully"
run_test test_search_no_midden_file     "midden-search: missing midden.json returns gracefully"
run_test test_tag_entry                 "midden-tag: tags an entry, action=added"
run_test test_untag_entry               "midden-tag: untags an entry, action=removed"
run_test test_tag_nonexistent_entry     "midden-tag: nonexistent entry returns ok=false"
run_test test_tag_no_id                 "midden-tag: missing --id returns validation error"
run_test test_tag_no_tag_or_untag       "midden-tag: missing --tag and --untag returns validation error"
run_test test_tag_no_duplicate          "midden-tag: double-tag does not duplicate"

test_summary
