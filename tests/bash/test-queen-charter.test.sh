#!/usr/bin/env bash
# Charter Management Integration Tests
# Tests charter-write and colony-name subcommands from queen.sh
# Covers: colony-name derivation, first init, re-init safety, no new headers,
#         METADATA accuracy, content truncation, error handling

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
AETHER_UTILS_SOURCE="$PROJECT_ROOT/.aether/aether-utils.sh"
QUEEN_TEMPLATE="$PROJECT_ROOT/.aether/templates/QUEEN.md.template"
COLONY_STATE_TEMPLATE="$PROJECT_ROOT/.aether/templates/colony-state.template.json"

# Source test helpers
source "$SCRIPT_DIR/test-helpers.sh"

# Verify jq is available
require_jq

# Verify aether-utils.sh exists
if [[ ! -f "$AETHER_UTILS_SOURCE" ]]; then
    log_error "aether-utils.sh not found at: $AETHER_UTILS_SOURCE"
    exit 1
fi

if [[ ! -f "$QUEEN_TEMPLATE" ]]; then
    log_error "QUEEN.md.template not found at: $QUEEN_TEMPLATE"
    exit 1
fi

# ============================================================================
# Helper: Create isolated charter test environment
# Sets up a temp dir with aether-utils.sh, utils, QUEEN.md, and COLONY_STATE.json
# ============================================================================
setupCharterTest() {
    local test_dir
    test_dir=$(mktemp -d)
    mkdir -p "$test_dir/.aether/data" "$test_dir/.aether/utils"

    cp "$AETHER_UTILS_SOURCE" "$test_dir/.aether/aether-utils.sh"
    chmod +x "$test_dir/.aether/aether-utils.sh"

    local utils_source="$(dirname "$AETHER_UTILS_SOURCE")/utils"
    if [[ -d "$utils_source" ]]; then
        cp -r "$utils_source" "$test_dir/.aether/"
    fi

    local exchange_source="$(dirname "$AETHER_UTILS_SOURCE")/exchange"
    if [[ -d "$exchange_source" ]]; then
        cp -r "$exchange_source" "$test_dir/.aether/"
    fi

    # Copy QUEEN.md template and replace placeholders
    cp "$QUEEN_TEMPLATE" "$test_dir/.aether/QUEEN.md"
    local ts
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    sed -i.bak "s/{TIMESTAMP}/$ts/g" "$test_dir/.aether/QUEEN.md" && rm -f "${test_dir}/.aether/QUEEN.md.bak"

    # Write a minimal valid COLONY_STATE.json (no colony_name set)
    cat > "$test_dir/.aether/data/COLONY_STATE.json" << EOF
{
  "version": "3.0",
  "goal": "test goal",
  "state": "READY",
  "colony_name": null,
  "current_phase": 0,
  "session_id": "session_1234_test",
  "initialized_at": "2026-03-27T00:00:00Z",
  "plan": { "phases": [] },
  "memory": { "phase_learnings": [], "decisions": [], "instincts": [] },
  "errors": { "records": [], "flagged_patterns": [] },
  "events": [],
  "signals": [],
  "graveyards": []
}
EOF

    echo "$test_dir"
}

# Run colony-name against a temp directory
run_colony_name() {
    local tmp_dir="$1"
    AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" \
        bash "$tmp_dir/.aether/aether-utils.sh" colony-name 2>/dev/null
}

# Run charter-write against a temp directory
run_charter_write() {
    local tmp_dir="$1"
    shift
    AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" \
        bash "$tmp_dir/.aether/aether-utils.sh" charter-write "$@" 2>/dev/null
}

# ============================================================================
# Test 1: colony-name from directory basename (fallback)
# ============================================================================
test_colony_name_from_directory() {
    # Create a temp dir with a predictable kebab-case name
    local tmp_dir
    tmp_dir=$(mktemp -d)
    local named_dir="$tmp_dir/my-test-project"
    mkdir -p "$named_dir/.aether/data" "$named_dir/.aether/utils"

    cp "$AETHER_UTILS_SOURCE" "$named_dir/.aether/aether-utils.sh"
    chmod +x "$named_dir/.aether/aether-utils.sh"
    local utils_source="$(dirname "$AETHER_UTILS_SOURCE")/utils"
    [[ -d "$utils_source" ]] && cp -r "$utils_source" "$named_dir/.aether/"
    local exchange_source="$(dirname "$AETHER_UTILS_SOURCE")/exchange"
    [[ -d "$exchange_source" ]] && cp -r "$exchange_source" "$named_dir/.aether/"

    # No package.json, no COLONY_STATE.json -- must fall back to directory
    local output result name source
    output=$(AETHER_ROOT="$named_dir" DATA_DIR="$named_dir/.aether/data" \
        bash "$named_dir/.aether/aether-utils.sh" colony-name 2>/dev/null) || true
    result=$(echo "$output" | jq -r '.result // empty')
    name=$(echo "$result" | jq -r '.name // empty')
    source=$(echo "$result" | jq -r '.source // empty')

    if [[ "$name" != "My Test Project" ]]; then
        test_fail "name = 'My Test Project'" "name = '$name'"
        rm -rf "$tmp_dir"
        return 1
    fi

    if [[ "$source" != "directory" ]]; then
        test_fail "source = 'directory'" "source = '$source'"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 2: colony-name from package.json
# ============================================================================
test_colony_name_from_package_json() {
    local tmp_dir
    tmp_dir=$(setupCharterTest)

    # Create package.json with a name
    echo '{"name":"my-cool-app"}' > "$tmp_dir/package.json"

    local output result name source
    output=$(run_colony_name "$tmp_dir")
    result=$(echo "$output" | jq -r '.result')
    name=$(echo "$result" | jq -r '.name')
    source=$(echo "$result" | jq -r '.source')

    if [[ "$name" != "My Cool App" ]]; then
        test_fail "name = 'My Cool App'" "name = '$name'"
        rm -rf "$tmp_dir"
        return 1
    fi

    if [[ "$source" != "package_json" ]]; then
        test_fail "source = 'package_json'" "source = '$source'"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 3: colony-name from COLONY_STATE.json
# ============================================================================
test_colony_name_from_colony_state() {
    local tmp_dir
    tmp_dir=$(setupCharterTest)

    # Set colony_name in COLONY_STATE.json
    local tmp_state="${tmp_dir}/.aether/data/COLONY_STATE.json.tmp.$$"
    jq '.colony_name = "Custom Name"' "$tmp_dir/.aether/data/COLONY_STATE.json" > "$tmp_state" && mv "$tmp_state" "$tmp_dir/.aether/data/COLONY_STATE.json"

    # Also create package.json -- colony_state should take precedence
    echo '{"name":"package-name"}' > "$tmp_dir/package.json"

    local output result name source
    output=$(run_colony_name "$tmp_dir")
    result=$(echo "$output" | jq -r '.result')
    name=$(echo "$result" | jq -r '.name')
    source=$(echo "$result" | jq -r '.source')

    if [[ "$name" != "Custom Name" ]]; then
        test_fail "name = 'Custom Name'" "name = '$name'"
        rm -rf "$tmp_dir"
        return 1
    fi

    if [[ "$source" != "colony_state" ]]; then
        test_fail "source = 'colony_state'" "source = '$source'"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 4: colony-name strips @scope/ prefix
# ============================================================================
test_colony_name_strips_scope() {
    local tmp_dir
    tmp_dir=$(setupCharterTest)

    # Create package.json with scoped name
    echo '{"name":"@org/my-app"}' > "$tmp_dir/package.json"

    local output result name
    output=$(run_colony_name "$tmp_dir")
    result=$(echo "$output" | jq -r '.result')
    name=$(echo "$result" | jq -r '.name')

    if [[ "$name" != "My App" ]]; then
        test_fail "name = 'My App' (scope stripped)" "name = '$name'"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 5: charter-write first init -- all 4 fields
# ============================================================================
test_charter_write_first_init() {
    local tmp_dir
    tmp_dir=$(setupCharterTest)

    local output result written updated
    output=$(run_charter_write "$tmp_dir" \
        --intent "Build the best app ever" \
        --vision "World domination through code" \
        --governance "Strict TDD, no shortcuts" \
        --goals "Ship v1 by end of month")

    result=$(echo "$output" | jq -r '.result')
    written=$(echo "$result" | jq -r '.written')
    updated=$(echo "$result" | jq -r '.updated')

    # Verify written count
    if [[ "$written" != "4" ]]; then
        test_fail "written = 4" "written = $written"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify this is a first init (not update)
    if [[ "$updated" != "false" ]]; then
        test_fail "updated = false" "updated = $updated"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify QUEEN.md has charter entries in User Preferences
    local queen_file="$tmp_dir/.aether/QUEEN.md"
    local up_section
    up_section=$(sed -n '/^## User Preferences$/,/^## /p' "$queen_file" | sed '$d')

    if ! echo "$up_section" | grep -q '\[charter\] \*\*Intent\*\*:'; then
        test_fail "User Preferences has [charter] **Intent**:" "not found"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! echo "$up_section" | grep -q '\[charter\] \*\*Vision\*\*:'; then
        test_fail "User Preferences has [charter] **Vision**:" "not found"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify QUEEN.md has charter entries in Codebase Patterns
    local cp_section
    cp_section=$(sed -n '/^## Codebase Patterns$/,/^## /p' "$queen_file" | sed '$d')

    if ! echo "$cp_section" | grep -q '\[charter\] \*\*Governance\*\*:'; then
        test_fail "Codebase Patterns has [charter] **Governance**:" "not found"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! echo "$cp_section" | grep -q '\[charter\] \*\*Goal\*\*:'; then
        test_fail "Codebase Patterns has [charter] **Goal**:" "not found"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify no new ## headers (count before template vs after)
    local header_count
    header_count=$(grep -c '^## ' "$queen_file")
    if [[ "$header_count" -ne 5 ]]; then
        test_fail "exactly 5 ## headers" "$header_count headers"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 6: charter-write partial fields (only --intent and --governance)
# ============================================================================
test_charter_write_partial_fields() {
    local tmp_dir
    tmp_dir=$(setupCharterTest)

    local output result written
    output=$(run_charter_write "$tmp_dir" \
        --intent "Build the best app" \
        --governance "Strict TDD")

    result=$(echo "$output" | jq -r '.result')
    written=$(echo "$result" | jq -r '.written')

    if [[ "$written" != "2" ]]; then
        test_fail "written = 2" "written = $written"
        rm -rf "$tmp_dir"
        return 1
    fi

    local queen_file="$tmp_dir/.aether/QUEEN.md"

    # Verify intent is in User Preferences
    if ! grep -q '\[charter\] \*\*Intent\*\*:' "$queen_file"; then
        test_fail "User Preferences has Intent entry" "not found"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify vision is NOT present
    if grep -q '\[charter\] \*\*Vision\*\*:' "$queen_file"; then
        test_fail "Vision entry absent" "Vision found but should not be"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify governance is in Codebase Patterns
    if ! grep -q '\[charter\] \*\*Governance\*\*:' "$queen_file"; then
        test_fail "Codebase Patterns has Governance entry" "not found"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify goals is NOT present
    if grep -q '\[charter\] \*\*Goal\*\*:' "$queen_file"; then
        test_fail "Goal entry absent" "Goal found but should not be"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 7: charter-write re-init safety (non-charter content preserved)
# ============================================================================
test_charter_write_reinit_safety() {
    local tmp_dir
    tmp_dir=$(setupCharterTest)

    local queen_file="$tmp_dir/.aether/QUEEN.md"

    # First write
    run_charter_write "$tmp_dir" \
        --intent "Original intent" \
        --vision "Original vision" \
        --governance "Original governance" \
        --goals "Original goals" >/dev/null

    # Add non-charter wisdom entries to both sections
    # Add wisdom to User Preferences (after placeholder removal, there's now content)
    sed -i.bak '/^## User Preferences$/,/^## /{
        /^---$/a\
- [wisdom] User prefers detailed commit messages
}' "$queen_file" && rm -f "${queen_file}.bak"

    # Add wisdom to Codebase Patterns
    sed -i.bak '/^## Codebase Patterns$/,/^## /{
        /^---$/a\
- [repo] Always use async/await for promises
}' "$queen_file" && rm -f "${queen_file}.bak"

    # Verify non-charter entries are present before re-init
    if ! grep -q '\[wisdom\] User prefers detailed commit messages' "$queen_file"; then
        test_fail "pre-condition: wisdom entry present" "not found"
        rm -rf "$tmp_dir"
        return 1
    fi
    if ! grep -q '\[repo\] Always use async/await for promises' "$queen_file"; then
        test_fail "pre-condition: repo entry present" "not found"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Second write (re-init) with different content
    local output result written updated
    output=$(run_charter_write "$tmp_dir" \
        --intent "Updated intent" \
        --vision "Updated vision" \
        --governance "Updated governance" \
        --goals "Updated goals")

    result=$(echo "$output" | jq -r '.result')
    written=$(echo "$result" | jq -r '.written')
    updated=$(echo "$result" | jq -r '.updated')

    # Verify updated flag
    if [[ "$updated" != "true" ]]; then
        test_fail "updated = true" "updated = $updated"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify charter entries updated to new content
    if ! grep -q '\[charter\] \*\*Intent\*\*: Updated intent' "$queen_file"; then
        test_fail "Intent updated to new content" "not found or still old"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify non-charter entries still present
    if ! grep -q '\[wisdom\] User prefers detailed commit messages' "$queen_file"; then
        test_fail "non-charter wisdom preserved" "not found after re-init"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! grep -q '\[repo\] Always use async/await for promises' "$queen_file"; then
        test_fail "non-charter repo entry preserved" "not found after re-init"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify old charter content is gone
    if grep -q '\[charter\] \*\*Intent\*\*: Original intent' "$queen_file"; then
        test_fail "old charter content removed" "old intent still present"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 8: charter-write never creates new ## headers
# ============================================================================
test_charter_write_no_new_headers() {
    local tmp_dir
    tmp_dir=$(setupCharterTest)
    local queen_file="$tmp_dir/.aether/QUEEN.md"

    # Count ## headers before charter-write
    local headers_before
    headers_before=$(grep -c '^## ' "$queen_file")

    # Write charter
    run_charter_write "$tmp_dir" \
        --intent "Test intent" \
        --vision "Test vision" \
        --governance "Test governance" \
        --goals "Test goals" >/dev/null

    # Count ## headers after
    local headers_after
    headers_after=$(grep -c '^## ' "$queen_file")

    if [[ "$headers_before" -ne "$headers_after" ]]; then
        test_fail "headers unchanged ($headers_before = $headers_after)" "before=$headers_before after=$headers_after"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Also test with QUEEN.md containing existing wisdom
    local tmp_dir2
    tmp_dir2=$(setupCharterTest)
    local queen_file2="$tmp_dir2/.aether/QUEEN.md"

    # Add existing wisdom to User Preferences section
    sed -i.bak '/^## User Preferences$/,/^## /{
        /^---$/a\
- [general] Some existing wisdom pattern
}' "$queen_file2" && rm -f "${queen_file2}.bak"

    local headers_before2
    headers_before2=$(grep -c '^## ' "$queen_file2")

    run_charter_write "$tmp_dir2" \
        --intent "Another intent" \
        --vision "Another vision" >/dev/null

    local headers_after2
    headers_after2=$(grep -c '^## ' "$queen_file2")

    if [[ "$headers_before2" -ne "$headers_after2" ]]; then
        test_fail "headers unchanged with existing wisdom ($headers_before2 = $headers_after2)" "before=$headers_before2 after=$headers_after2"
        rm -rf "$tmp_dir" "$tmp_dir2"
        return 1
    fi

    rm -rf "$tmp_dir" "$tmp_dir2"
    return 0
}

# ============================================================================
# Test 9: charter-write METADATA stats accuracy
# ============================================================================
test_charter_write_metadata_stats() {
    local tmp_dir
    tmp_dir=$(setupCharterTest)
    local queen_file="$tmp_dir/.aether/QUEEN.md"

    # First charter-write with 2 user prefs + 2 codebase patterns
    run_charter_write "$tmp_dir" \
        --intent "Test intent" \
        --vision "Test vision" \
        --governance "Test governance" \
        --goals "Test goals" >/dev/null

    # Extract METADATA stats
    local metadata
    metadata=$(sed -n '/<!-- METADATA/,/-->/p' "$queen_file" | sed '1d;$d' | tr -d '\n' | sed 's/^[[:space:]]*//')

    local up_count cp_count
    up_count=$(echo "$metadata" | jq -r '.stats.total_user_prefs')
    cp_count=$(echo "$metadata" | jq -r '.stats.total_codebase_patterns')

    if [[ "$up_count" != "2" ]]; then
        test_fail "total_user_prefs = 2 after first write" "total_user_prefs = $up_count"
        rm -rf "$tmp_dir"
        return 1
    fi

    if [[ "$cp_count" != "2" ]]; then
        test_fail "total_codebase_patterns = 2 after first write" "total_codebase_patterns = $cp_count"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Add a non-charter wisdom entry
    sed -i.bak '/^## User Preferences$/,/^## /{
        /^---$/a\
- [decree] Always use descriptive variable names
}' "$queen_file" && rm -f "${queen_file}.bak"

    # Re-init with same 4 fields
    run_charter_write "$tmp_dir" \
        --intent "Test intent" \
        --vision "Test vision" \
        --governance "Test governance" \
        --goals "Test goals" >/dev/null

    # Extract METADATA stats again
    metadata=$(sed -n '/<!-- METADATA/,/-->/p' "$queen_file" | sed '1d;$d' | tr -d '\n' | sed 's/^[[:space:]]*//')
    up_count=$(echo "$metadata" | jq -r '.stats.total_user_prefs')
    cp_count=$(echo "$metadata" | jq -r '.stats.total_codebase_patterns')

    # Should be: 1 non-charter decree + 2 charter entries = 3
    if [[ "$up_count" != "3" ]]; then
        test_fail "total_user_prefs = 3 after re-init (1 decree + 2 charter)" "total_user_prefs = $up_count"
        rm -rf "$tmp_dir"
        return 1
    fi

    if [[ "$cp_count" != "2" ]]; then
        test_fail "total_codebase_patterns = 2 after re-init" "total_codebase_patterns = $cp_count"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 10: charter-write persists colony_name in COLONY_STATE.json
# ============================================================================
test_charter_write_colony_name_persisted() {
    local tmp_dir
    tmp_dir=$(setupCharterTest)

    # Verify colony_name is null before write
    local current_name
    current_name=$(jq -r '.colony_name // empty' "$tmp_dir/.aether/data/COLONY_STATE.json" 2>/dev/null || echo "")
    if [[ -n "$current_name" ]]; then
        test_fail "colony_name null before write" "colony_name = '$current_name'"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Write charter
    run_charter_write "$tmp_dir" --intent "Test intent" >/dev/null

    # Verify colony_name is set in COLONY_STATE.json
    local persisted_name
    persisted_name=$(jq -r '.colony_name // empty' "$tmp_dir/.aether/data/COLONY_STATE.json" 2>/dev/null || echo "")

    if [[ -z "$persisted_name" ]]; then
        test_fail "colony_name set after charter-write" "colony_name is still null/empty"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Name should be title-cased version of directory basename
    # Since the directory is a temp dir with a random name, just verify it's non-empty and title-cased
    if [[ "$persisted_name" =~ ^[A-Z] ]]; then
        : # Good - starts with uppercase
    else
        test_fail "colony_name starts with uppercase" "colony_name = '$persisted_name'"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 11: charter-write content truncation at 200 chars
# ============================================================================
test_charter_write_content_truncation() {
    local tmp_dir
    tmp_dir=$(setupCharterTest)

    # Build a 300-char intent string
    local long_intent
    long_intent=$(python3 -c "print('A' * 300)" 2>/dev/null || printf '%0.sA' $(seq 1 300))

    run_charter_write "$tmp_dir" --intent "$long_intent" >/dev/null

    local queen_file="$tmp_dir/.aether/QUEEN.md"

    # Extract the Intent line
    local intent_line
    intent_line=$(grep '\[charter\] \*\*Intent\*\*:' "$queen_file")

    # Extract the intent value (everything after ": ")
    local intent_value
    intent_value=$(echo "$intent_line" | sed 's/.*\*\*Intent\*\*: //' | sed 's/ (Colony:.*//' )

    local intent_len=${#intent_value}

    # Should be 203 chars (200 + "...")
    if [[ "$intent_len" -gt 203 ]]; then
        test_fail "intent truncated to <= 203 chars (200 + ...)" "intent length = $intent_len"
        rm -rf "$tmp_dir"
        return 1
    fi

    if [[ "$intent_len" -lt 200 ]]; then
        test_fail "intent at least 200 chars before truncation" "intent length = $intent_len"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify it ends with "..."
    if [[ "$intent_value" != *"..." ]]; then
        test_fail "intent ends with '...'" "intent = '${intent_value: -10}'"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 12: charter-write returns error when QUEEN.md missing
# ============================================================================
test_charter_write_queen_missing() {
    local tmp_dir
    tmp_dir=$(setupCharterTest)

    # Delete QUEEN.md
    rm -f "$tmp_dir/.aether/QUEEN.md"

    # json_err writes to stderr, so capture stderr
    local output ok
    output=$(AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" \
        bash "$tmp_dir/.aether/aether-utils.sh" charter-write --intent "Test intent" 2>&1 || true)
    ok=$(echo "$output" | jq -r '.ok' 2>/dev/null || echo "unknown")

    if [[ "$ok" != "false" ]]; then
        test_fail "ok = false for missing QUEEN.md" "ok = $ok"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify error code
    local code
    code=$(echo "$output" | jq -r '.error.code' 2>/dev/null || echo "none")
    if [[ "$code" != "E_FILE_NOT_FOUND" ]]; then
        test_fail "error code = E_FILE_NOT_FOUND" "error code = $code"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Run all tests
# ============================================================================
log_info "Running Charter Management integration tests"

run_test test_colony_name_from_directory "colony-name: fallback to directory basename with title case"
run_test test_colony_name_from_package_json "colony-name: derive from package.json name"
run_test test_colony_name_from_colony_state "colony-name: COLONY_STATE.json takes precedence"
run_test test_colony_name_strips_scope "colony-name: strip @scope/ prefix from package name"
run_test test_charter_write_first_init "charter-write: first init writes all 4 fields to correct sections"
run_test test_charter_write_partial_fields "charter-write: partial fields (only intent + governance)"
run_test test_charter_write_reinit_safety "charter-write: re-init preserves non-charter content"
run_test test_charter_write_no_new_headers "charter-write: never creates new ## headers"
run_test test_charter_write_metadata_stats "charter-write: METADATA stats accurate after write and re-init"
run_test test_charter_write_colony_name_persisted "charter-write: colony_name persisted in COLONY_STATE.json"
run_test test_charter_write_content_truncation "charter-write: content truncated to 200 chars + ..."
run_test test_charter_write_queen_missing "charter-write: E_FILE_NOT_FOUND when QUEEN.md missing"

test_summary
