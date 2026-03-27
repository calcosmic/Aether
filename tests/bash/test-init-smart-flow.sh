#!/usr/bin/env bash
# Smart Init Flow Integration Tests
# Tests bash components used by the smart init flow:
# - Prompt assembly from scan data
# - Re-init detection (with and without existing colony)
# - Charter extraction from QUEEN.md
# - Charter-write after approval
# - Re-init preserves existing wisdom
# - Scan graceful degradation on empty directories

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
AETHER_UTILS_SOURCE="$PROJECT_ROOT/.aether/aether-utils.sh"
QUEEN_TEMPLATE="$PROJECT_ROOT/.aether/templates/QUEEN.md.template"

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
# Helper: Create isolated smart init test environment
# Sets up a temp dir with aether-utils.sh, utils, exchange, QUEEN.md, COLONY_STATE
# ============================================================================
setup_smart_init_env() {
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

    # Copy QUEEN.md template
    cp "$QUEEN_TEMPLATE" "$test_dir/.aether/QUEEN.md"
    local ts
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    sed -i.bak "s/{TIMESTAMP}/$ts/g" "$test_dir/.aether/QUEEN.md" && rm -f "${test_dir}/.aether/QUEEN.md.bak"

    # Write a minimal valid COLONY_STATE.json
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

# Run init-research against a temp directory
run_init_research() {
    local tmp_dir="$1"
    local target_dir="${2:-$tmp_dir}"
    AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" \
        bash "$tmp_dir/.aether/aether-utils.sh" init-research --target "$target_dir" 2>/dev/null
}

# Run charter-write against a temp directory
run_charter_write() {
    local tmp_dir="$1"
    shift
    AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" \
        bash "$tmp_dir/.aether/aether-utils.sh" charter-write "$@" 2>/dev/null
}

# ============================================================================
# Test 1: Prompt assembly from scan data
# Verifies init-research produces fields needed for prompt assembly
# ============================================================================
test_prompt_assembly_from_scan_data() {
    local tmp_dir
    tmp_dir=$(setup_smart_init_env)

    # Create a small repo with a package.json to have detectable tech
    cat > "$tmp_dir/package.json" << 'EOF'
{
  "name": "test-project",
  "version": "1.0.0",
  "dependencies": {
    "express": "^4.18.0"
  }
}
EOF

    local output result
    output=$(run_init_research "$tmp_dir")
    result=$(echo "$output" | jq -r '.result')

    # Verify result is valid JSON
    if ! assert_json_valid "$result"; then
        test_fail "valid JSON result" "invalid JSON"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify tech_stack.languages is extractable via jq
    local has_languages
    has_languages=$(echo "$result" | jq -e '.tech_stack.languages' >/dev/null 2>&1 && echo "true" || echo "false")
    if [[ "$has_languages" != "true" ]]; then
        test_fail "tech_stack.languages extractable" "not found"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify tech_stack.frameworks is extractable via jq
    local has_frameworks
    has_frameworks=$(echo "$result" | jq -e '.tech_stack.frameworks' >/dev/null 2>&1 && echo "true" || echo "false")
    if [[ "$has_frameworks" != "true" ]]; then
        test_fail "tech_stack.frameworks extractable" "not found"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify complexity.size is one of: small, medium, large
    local size
    size=$(echo "$result" | jq -r '.complexity.size')
    if [[ "$size" != "small" && "$size" != "medium" && "$size" != "large" ]]; then
        test_fail "complexity.size is small/medium/large" "size = $size"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify survey_status.suggestion.reason is present or empty
    local suggestion_reason
    suggestion_reason=$(echo "$result" | jq -r '.survey_status.suggestion.reason // empty')
    # Should not error -- just check it doesn't crash
    if [[ $? -ne 0 ]]; then
        test_fail "survey_status.suggestion.reason extractable" "jq error"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify all top-level fields have non-null values or appropriate defaults
    local required_keys=("schema_version" "tech_stack" "directory_structure" "git_history" "survey_status" "prior_colonies" "complexity" "scanned_at")
    for key in "${required_keys[@]}"; do
        if ! assert_json_has_field "$result" "$key"; then
            test_fail "has required field '$key'" "missing"
            rm -rf "$tmp_dir"
            return 1
        fi
    done

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 2: Re-init detection with existing colony
# Verifies COLONY_STATE.json with goal triggers re-init mode
# ============================================================================
test_reinit_detection_with_existing_colony() {
    local tmp_dir
    tmp_dir=$(setup_smart_init_env)

    # COLONY_STATE.json already has goal from setup_smart_init_env
    local state_file="$tmp_dir/.aether/data/COLONY_STATE.json"

    # Verify goal is readable via jq
    local goal
    goal=$(jq -r '.goal' "$state_file")
    if [[ "$goal" != "test goal" ]]; then
        test_fail "goal = 'test goal'" "goal = '$goal'"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify re-init mode would be set to true (goal is non-null)
    local is_non_null
    is_non_null=$(jq -e '.goal != null and .goal != ""' "$state_file" >/dev/null 2>&1 && echo "true" || echo "false")
    if [[ "$is_non_null" != "true" ]]; then
        test_fail "re-init mode = true (goal is non-null)" "re-init mode = false"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 3: Re-init detection no existing colony
# Verifies absence of COLONY_STATE.json means fresh init
# ============================================================================
test_reinit_detection_no_existing_colony() {
    local tmp_dir
    tmp_dir=$(setup_smart_init_env)

    # Remove COLONY_STATE.json
    rm -f "$tmp_dir/.aether/data/COLONY_STATE.json"

    # Verify file check returns "not found"
    local state_file="$tmp_dir/.aether/data/COLONY_STATE.json"
    if [[ -f "$state_file" ]]; then
        test_fail "COLONY_STATE.json not found" "file exists"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify re-init mode would be set to false
    local reinit_mode="false"
    if [[ -f "$state_file" ]]; then
        local goal
        goal=$(jq -r '.goal // empty' "$state_file" 2>/dev/null || true)
        [[ -n "$goal" ]] && reinit_mode="true"
    fi

    if [[ "$reinit_mode" != "false" ]]; then
        test_fail "re-init mode = false" "re-init mode = $reinit_mode"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 4: Charter extraction from QUEEN.md
# Verifies charter entries can be extracted via grep+sed pattern
# ============================================================================
test_charter_extraction_from_queen_md() {
    local tmp_dir
    tmp_dir=$(setup_smart_init_env)
    local queen_file="$tmp_dir/.aether/QUEEN.md"

    # Write charter entries to QUEEN.md via charter-write
    run_charter_write "$tmp_dir" \
        --intent "Build an amazing app" \
        --vision "Change the world through software" \
        --governance "Strict TDD, no shortcuts" \
        --goals "Ship v1 by end of month" >/dev/null

    # Extract charter fields using the same pattern as init.md Step 4
    local intent vision governance goals

    intent=$(grep '\[charter\] \*\*Intent\*\*:' "$queen_file" 2>/dev/null | sed 's/.*\*\*Intent\*\*: //' | sed 's/ (Colony:.*//' || true)
    vision=$(grep '\[charter\] \*\*Vision\*\*:' "$queen_file" 2>/dev/null | sed 's/.*\*\*Vision\*\*: //' | sed 's/ (Colony:.*//' || true)
    governance=$(grep '\[charter\] \*\*Governance\*\*:' "$queen_file" 2>/dev/null | sed 's/.*\*\*Governance\*\*: //' | sed 's/ (Colony:.*//' || true)
    goals=$(grep '\[charter\] \*\*Goal\*\*:' "$queen_file" 2>/dev/null | sed 's/.*\*\*Goal\*\*: //' | sed 's/ (Colony:.*//' || true)

    # Verify each field is extractable and matches expected value
    if [[ "$intent" != "Build an amazing app" ]]; then
        test_fail "Intent extracted correctly" "intent = '$intent'"
        rm -rf "$tmp_dir"
        return 1
    fi

    if [[ "$vision" != "Change the world through software" ]]; then
        test_fail "Vision extracted correctly" "vision = '$vision'"
        rm -rf "$tmp_dir"
        return 1
    fi

    if [[ "$governance" != "Strict TDD, no shortcuts" ]]; then
        test_fail "Governance extracted correctly" "governance = '$governance'"
        rm -rf "$tmp_dir"
        return 1
    fi

    if [[ "$goals" != "Ship v1 by end of month" ]]; then
        test_fail "Goal extracted correctly" "goals = '$goals'"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify (Colony: ...) suffix is stripped
    local raw_line
    raw_line=$(grep '\[charter\] \*\*Intent\*\*:' "$queen_file" 2>/dev/null || true)
    if [[ -n "$raw_line" && "$intent" == *" (Colony:"* ]]; then
        test_fail "(Colony: ...) suffix stripped" "suffix still present in intent"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 5: Charter-write after approval
# Verifies charter-write produces correct entries without creating new headers
# ============================================================================
test_charter_write_after_approval() {
    local tmp_dir
    tmp_dir=$(setup_smart_init_env)
    local queen_file="$tmp_dir/.aether/QUEEN.md"

    # Count ## headers before charter-write
    local headers_before
    headers_before=$(grep -c '^## ' "$queen_file")

    # Write charter
    local output result written
    output=$(run_charter_write "$tmp_dir" \
        --intent "Build the best app ever" \
        --vision "World domination through code" \
        --governance "Strict TDD, no shortcuts" \
        --goals "Ship v1 by end of month")

    result=$(echo "$output" | jq -r '.result')
    written=$(echo "$result" | jq -r '.written')

    # Verify written count
    if [[ "$written" != "4" ]]; then
        test_fail "written = 4" "written = $written"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify QUEEN.md contains all 4 charter entries
    if ! grep -q '\[charter\] \*\*Intent\*\*:' "$queen_file"; then
        test_fail "[charter] **Intent**: present" "not found"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! grep -q '\[charter\] \*\*Vision\*\*:' "$queen_file"; then
        test_fail "[charter] **Vision**: present" "not found"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! grep -q '\[charter\] \*\*Governance\*\*:' "$queen_file"; then
        test_fail "[charter] **Governance**: present" "not found"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! grep -q '\[charter\] \*\*Goal\*\*:' "$queen_file"; then
        test_fail "[charter] **Goal**: present" "not found"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify no new ## headers were created
    local headers_after
    headers_after=$(grep -c '^## ' "$queen_file")
    if [[ "$headers_before" -ne "$headers_after" ]]; then
        test_fail "no new ## headers ($headers_before = $headers_after)" "before=$headers_before after=$headers_after"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify calling charter-write again with different values updates in-place
    output=$(run_charter_write "$tmp_dir" \
        --intent "New intent" \
        --vision "New vision" \
        --governance "New governance" \
        --goals "New goals")
    result=$(echo "$output" | jq -r '.result')
    local updated
    updated=$(echo "$result" | jq -r '.updated')

    if [[ "$updated" != "true" ]]; then
        test_fail "updated = true on second write" "updated = $updated"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! grep -q '\[charter\] \*\*Intent\*\*: New intent' "$queen_file"; then
        test_fail "Intent updated to new value" "still old value"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Headers still unchanged after update
    local headers_final
    headers_final=$(grep -c '^## ' "$queen_file")
    if [[ "$headers_before" -ne "$headers_final" ]]; then
        test_fail "no new ## headers after update" "before=$headers_before after=$headers_final"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 6: Re-init preserves existing wisdom
# Verifies non-charter wisdom entries survive charter-write update
# ============================================================================
test_reinit_preserves_existing_wisdom() {
    local tmp_dir
    tmp_dir=$(setup_smart_init_env)
    local queen_file="$tmp_dir/.aether/QUEEN.md"

    # First charter-write
    run_charter_write "$tmp_dir" \
        --intent "Original intent" \
        --vision "Original vision" \
        --governance "Original governance" \
        --goals "Original goals" >/dev/null

    # Add non-charter wisdom entries to User Preferences section
    sed -i.bak '/^## User Preferences$/,/^## /{
        /^---$/a\
- [wisdom] User prefers detailed commit messages
}' "$queen_file" && rm -f "${queen_file}.bak"

    # Add non-charter wisdom to Codebase Patterns section
    sed -i.bak '/^## Codebase Patterns$/,/^## /{
        /^---$/a\
- [repo] Always use async/await for promises
}' "$queen_file" && rm -f "${queen_file}.bak"

    # Add a build learning entry
    sed -i.bak '/^## Build Learnings$/,/^## /{
        /^---$/a\
- [phase-1] Error handling patterns need improvement
}' "$queen_file" && rm -f "${queen_file}.bak"

    # Verify non-charter entries are present before re-init
    if ! grep -q '\[wisdom\] User prefers detailed commit messages' "$queen_file"; then
        test_fail "pre-condition: wisdom entry in User Preferences" "not found"
        rm -rf "$tmp_dir"
        return 1
    fi
    if ! grep -q '\[repo\] Always use async/await for promises' "$queen_file"; then
        test_fail "pre-condition: repo entry in Codebase Patterns" "not found"
        rm -rf "$tmp_dir"
        return 1
    fi
    if ! grep -q '\[phase-1\] Error handling patterns need improvement' "$queen_file"; then
        test_fail "pre-condition: build learning entry" "not found"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Second charter-write (re-init) with different content
    run_charter_write "$tmp_dir" \
        --intent "Updated intent" \
        --vision "Updated vision" \
        --governance "Updated governance" \
        --goals "Updated goals" >/dev/null

    # Verify charter entries were updated
    if ! grep -q '\[charter\] \*\*Intent\*\*: Updated intent' "$queen_file"; then
        test_fail "charter Intent updated" "still old content"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify non-charter entries are preserved
    if ! grep -q '\[wisdom\] User prefers detailed commit messages' "$queen_file"; then
        test_fail "wisdom entry preserved in User Preferences" "not found after re-init"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! grep -q '\[repo\] Always use async/await for promises' "$queen_file"; then
        test_fail "repo entry preserved in Codebase Patterns" "not found after re-init"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! grep -q '\[phase-1\] Error handling patterns need improvement' "$queen_file"; then
        test_fail "build learning entry preserved" "not found after re-init"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify old charter content is gone
    if grep -q '\[charter\] \*\*Intent\*\*: Original intent' "$queen_file"; then
        test_fail "old charter Intent removed" "old content still present"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 7: Scan graceful degradation on empty directory
# Verifies init-research returns valid JSON with defaults on empty dirs
# ============================================================================
test_scan_graceful_degradation() {
    local tmp_dir
    tmp_dir=$(setup_smart_init_env)

    # Create an empty target directory (no git, no .aether, no source files)
    local scan_target
    scan_target=$(mktemp -d)

    local output result
    output=$(run_init_research "$tmp_dir" "$scan_target")
    result=$(echo "$output" | jq -r '.result')

    # Verify returns valid JSON (no errors)
    if ! assert_json_valid "$result"; then
        test_fail "valid JSON on empty directory" "invalid JSON"
        rm -rf "$tmp_dir" "$scan_target"
        return 1
    fi

    # Verify ok field is true
    if ! assert_ok_true "$output"; then
        test_fail "ok = true" "ok not true"
        rm -rf "$tmp_dir" "$scan_target"
        return 1
    fi

    # Verify result object exists
    if ! assert_json_has_field "$output" "result"; then
        test_fail "result field exists" "missing"
        rm -rf "$tmp_dir" "$scan_target"
        return 1
    fi

    # Verify empty arrays for languages
    local lang_count
    lang_count=$(echo "$result" | jq '.tech_stack.languages | length')
    if [[ "$lang_count" != "0" ]]; then
        test_fail "languages = [] (empty)" "count = $lang_count"
        rm -rf "$tmp_dir" "$scan_target"
        return 1
    fi

    # Verify empty arrays for frameworks
    local fwk_count
    fwk_count=$(echo "$result" | jq '.tech_stack.frameworks | length')
    if [[ "$fwk_count" != "0" ]]; then
        test_fail "frameworks = [] (empty)" "count = $fwk_count"
        rm -rf "$tmp_dir" "$scan_target"
        return 1
    fi

    # Verify complexity is "small" for empty directory
    local size
    size=$(echo "$result" | jq -r '.complexity.size')
    if [[ "$size" != "small" ]]; then
        test_fail "complexity = small for empty dir" "complexity = $size"
        rm -rf "$tmp_dir" "$scan_target"
        return 1
    fi

    rm -rf "$tmp_dir" "$scan_target"
    return 0
}

# ============================================================================
# Run all tests
# ============================================================================
log_info "Running Smart Init Flow integration tests"

run_test test_prompt_assembly_from_scan_data "Prompt assembly: init-research fields extractable for prompt"
run_test test_reinit_detection_with_existing_colony "Re-init detection: COLONY_STATE.json with goal triggers re-init"
run_test test_reinit_detection_no_existing_colony "Re-init detection: no COLONY_STATE.json means fresh init"
run_test test_charter_extraction_from_queen_md "Charter extraction: grep+sed pattern extracts values with (Colony:) stripped"
run_test test_charter_write_after_approval "Charter-write: 4 fields written, no new headers, in-place update works"
run_test test_reinit_preserves_existing_wisdom "Re-init preserves: non-charter wisdom survives charter update"
run_test test_scan_graceful_degradation "Scan graceful degradation: empty directory returns valid defaults"

test_summary
