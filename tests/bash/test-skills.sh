#!/usr/bin/env bash
# Tests for the Aether skills engine
# Task 1.1: RED phase — these tests should fail because skill subcommands don't exist yet

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS_SOURCE="$REPO_ROOT/.aether/aether-utils.sh"

# ============================================================================
# Helper: Create an isolated test environment with sample SKILL.md files
# ============================================================================
setup_skills_env() {
    TEST_DIR=$(mktemp -d)
    SKILLS_DIR="$TEST_DIR/skills"
    mkdir -p "$SKILLS_DIR/colony/test-skill"
    mkdir -p "$SKILLS_DIR/domain/test-domain"

    # Create test colony skill
    cat > "$SKILLS_DIR/colony/test-skill/SKILL.md" << 'SKILL'
---
name: test-skill
description: Use when testing the skills system
type: colony
domains: [testing, quality]
agent_roles: [builder, watcher]
priority: normal
version: "1.0"
---

Test skill content here.
SKILL

    # Create test domain skill with detection patterns
    cat > "$SKILLS_DIR/domain/test-domain/SKILL.md" << 'SKILL'
---
name: test-domain
description: Use when working with test frameworks
type: domain
domains: [testing, frontend]
agent_roles: [builder]
detect_files: ["*.test.js", "jest.config.*"]
detect_packages: ["jest", "vitest"]
priority: normal
version: "1.0"
---

Domain skill content for testing.
SKILL

    export AETHER_SKILLS_DIR="$SKILLS_DIR"
}

cleanup_skills_env() {
    if [[ -n "${TEST_DIR:-}" && -d "${TEST_DIR:-}" ]]; then
        rm -rf "$TEST_DIR"
    fi
    unset TEST_DIR SKILLS_DIR AETHER_SKILLS_DIR
}

# ============================================================================
# Test 1: parse-frontmatter returns valid JSON with name field
# ============================================================================
test_parse_frontmatter() {
    test_start "parse-frontmatter returns valid JSON for colony skill"
    setup_skills_env

    local output exit_code=0
    output=$(bash "$AETHER_UTILS_SOURCE" skill-parse-frontmatter "$SKILLS_DIR/colony/test-skill/SKILL.md" 2>&1) || exit_code=$?

    if ! assert_exit_code "$exit_code" 0; then
        test_fail "Expected exit code 0" "Got $exit_code — subcommand likely does not exist"
        cleanup_skills_env
        return 0
    fi

    if ! assert_json_valid "$output"; then
        test_fail "Expected valid JSON output" "$output"
        cleanup_skills_env
        return 0
    fi

    local name
    name=$(echo "$output" | jq -r '.result.name')
    if [[ "$name" == "test-skill" ]]; then
        test_pass
    else
        test_fail "Expected name 'test-skill'" "Got '$name'"
    fi

    cleanup_skills_env
}

# ============================================================================
# Test 2: parse-frontmatter extracts domains array correctly
# ============================================================================
test_parse_frontmatter_domains() {
    test_start "parse-frontmatter extracts domains array"
    setup_skills_env

    local output exit_code=0
    output=$(bash "$AETHER_UTILS_SOURCE" skill-parse-frontmatter "$SKILLS_DIR/colony/test-skill/SKILL.md" 2>&1) || exit_code=$?

    if ! assert_exit_code "$exit_code" 0; then
        test_fail "Expected exit code 0" "Got $exit_code — subcommand likely does not exist"
        cleanup_skills_env
        return 0
    fi

    local domains
    domains=$(echo "$output" | jq -r '.result.domains | length')
    if [[ "$domains" == "2" ]]; then
        test_pass
    else
        test_fail "Expected 2 domains" "Got '$domains'"
    fi

    cleanup_skills_env
}

# ============================================================================
# Test 3: parse-frontmatter extracts detect_files and detect_packages
# ============================================================================
test_parse_frontmatter_detect() {
    test_start "parse-frontmatter extracts detect_files and detect_packages"
    setup_skills_env

    local output exit_code=0
    output=$(bash "$AETHER_UTILS_SOURCE" skill-parse-frontmatter "$SKILLS_DIR/domain/test-domain/SKILL.md" 2>&1) || exit_code=$?

    if ! assert_exit_code "$exit_code" 0; then
        test_fail "Expected exit code 0" "Got $exit_code — subcommand likely does not exist"
        cleanup_skills_env
        return 0
    fi

    local detect_files
    detect_files=$(echo "$output" | jq -r '.result.detect_files | length')
    local detect_packages
    detect_packages=$(echo "$output" | jq -r '.result.detect_packages | length')

    if [[ "$detect_files" == "2" && "$detect_packages" == "2" ]]; then
        test_pass
    else
        test_fail "Expected 2 detect_files + 2 detect_packages" "Got $detect_files files + $detect_packages packages"
    fi

    cleanup_skills_env
}

# ============================================================================
# Test 4: skill-index builds from SKILL.md files with correct count
# ============================================================================
test_skill_index_build() {
    test_start "skill-index builds index from SKILL.md files"
    setup_skills_env

    local output exit_code=0
    output=$(AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-index "$SKILLS_DIR" 2>&1) || exit_code=$?

    if ! assert_exit_code "$exit_code" 0; then
        test_fail "Expected exit code 0" "Got $exit_code — subcommand likely does not exist"
        cleanup_skills_env
        return 0
    fi

    if ! assert_json_valid "$output"; then
        test_fail "Expected valid JSON output" "$output"
        cleanup_skills_env
        return 0
    fi

    local count
    count=$(echo "$output" | jq -r '.result.skill_count')
    if [[ "$count" == "2" ]]; then
        test_pass
    else
        test_fail "Expected 2 skills in index" "Got $count"
    fi

    cleanup_skills_env
}

# ============================================================================
# Test 5: skill-index creates cache file at .index.json
# ============================================================================
test_skill_index_cache() {
    test_start "skill-index creates cache file at .index.json"
    setup_skills_env

    # Build the index (allow failure since subcommand may not exist)
    AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-index "$SKILLS_DIR" > /dev/null 2>&1 || true

    # Verify cache file was created
    if [[ -f "$SKILLS_DIR/.index.json" ]]; then
        test_pass
    else
        test_fail "Expected cache file at $SKILLS_DIR/.index.json" "File not found"
    fi

    cleanup_skills_env
}

# ============================================================================
# Test 6: skill-match filters by agent role (builder matches both skills)
# ============================================================================
test_skill_match_by_role() {
    test_start "skill-match filters by agent role (builder matches colony + domain)"
    setup_skills_env

    # Build index first (allow failure)
    AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-index "$SKILLS_DIR" > /dev/null 2>&1 || true

    local output exit_code=0
    output=$(AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-match "builder" "" "$SKILLS_DIR" 2>&1) || exit_code=$?

    if ! assert_exit_code "$exit_code" 0; then
        test_fail "Expected exit code 0" "Got $exit_code — subcommand likely does not exist"
        cleanup_skills_env
        return 0
    fi

    if ! assert_json_valid "$output"; then
        test_fail "Expected valid JSON output" "$output"
        cleanup_skills_env
        return 0
    fi

    # Builder should match test-skill (colony, roles: builder,watcher)
    local colony_count
    colony_count=$(echo "$output" | jq '[.result.colony_skills[]?] | length')
    if [[ "$colony_count" -ge 1 ]]; then
        test_pass
    else
        test_fail "Expected at least 1 colony match for builder" "Got $colony_count"
    fi

    cleanup_skills_env
}

# ============================================================================
# Test 7: skill-match excludes skills for non-matching role (chronicler gets 0)
# ============================================================================
test_skill_match_role_filter() {
    test_start "skill-match excludes skills for non-matching role (chronicler)"
    setup_skills_env

    # Build index first (allow failure)
    AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-index "$SKILLS_DIR" > /dev/null 2>&1 || true

    local output exit_code=0
    output=$(AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-match "chronicler" "" "$SKILLS_DIR" 2>&1) || exit_code=$?

    if ! assert_exit_code "$exit_code" 0; then
        test_fail "Expected exit code 0" "Got $exit_code — subcommand likely does not exist"
        cleanup_skills_env
        return 0
    fi

    # Chronicler shouldn't match test-skill (only builder,watcher) or test-domain (only builder)
    local colony_count
    colony_count=$(echo "$output" | jq '[.result.colony_skills[]?] | length')
    local domain_count
    domain_count=$(echo "$output" | jq '[.result.domain_skills[]?] | length')

    if [[ "$colony_count" == "0" && "$domain_count" == "0" ]]; then
        test_pass
    else
        test_fail "Expected 0 colony + 0 domain matches for chronicler" "Got $colony_count colony + $domain_count domain"
    fi

    cleanup_skills_env
}

# ============================================================================
# Test 8: skill-inject loads full skill content within budget
# ============================================================================
test_skill_inject() {
    test_start "skill-inject loads full skill content within budget"
    setup_skills_env

    # Build index first (allow failure)
    AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-index "$SKILLS_DIR" > /dev/null 2>&1 || true

    # Construct match JSON pointing to the test colony skill
    local match_json='{"colony_skills":[{"name":"test-skill","file_path":"'"$SKILLS_DIR"'/colony/test-skill/SKILL.md"}],"domain_skills":[]}'

    local output exit_code=0
    output=$(bash "$AETHER_UTILS_SOURCE" skill-inject "$match_json" 2>&1) || exit_code=$?

    if ! assert_exit_code "$exit_code" 0; then
        test_fail "Expected exit code 0" "Got $exit_code — subcommand likely does not exist"
        cleanup_skills_env
        return 0
    fi

    # The injected section should contain the skill body content
    local section
    section=$(echo "$output" | jq -r '.result.skill_section')
    if [[ "$section" == *"Test skill content"* ]]; then
        test_pass
    else
        test_fail "Expected 'Test skill content' in skill_section" "Got: $section"
    fi

    cleanup_skills_env
}

# ============================================================================
# Test 9: skill-list returns all installed skills with count
# ============================================================================
test_skill_list() {
    test_start "skill-list returns all installed skills with count"
    setup_skills_env

    # Build index first (allow failure)
    AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-index "$SKILLS_DIR" > /dev/null 2>&1 || true

    local output exit_code=0
    output=$(AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-list "$SKILLS_DIR" 2>&1) || exit_code=$?

    if ! assert_exit_code "$exit_code" 0; then
        test_fail "Expected exit code 0" "Got $exit_code — subcommand likely does not exist"
        cleanup_skills_env
        return 0
    fi

    if ! assert_json_valid "$output"; then
        test_fail "Expected valid JSON output" "$output"
        cleanup_skills_env
        return 0
    fi

    local count
    count=$(echo "$output" | jq '.result.skill_count')
    if [[ "$count" == "2" ]]; then
        test_pass
    else
        test_fail "Expected 2 skills in list" "Got $count"
    fi

    cleanup_skills_env
}

# ============================================================================
# Test 10: parse-frontmatter returns error for missing file
# ============================================================================
test_parse_frontmatter_missing_file() {
    test_start "parse-frontmatter returns error for missing file"
    setup_skills_env

    local output exit_code=0
    output=$(bash "$AETHER_UTILS_SOURCE" skill-parse-frontmatter "/nonexistent/SKILL.md" 2>&1) || exit_code=$?

    local ok
    ok=$(echo "$output" | jq -r '.ok' 2>/dev/null)
    if [[ "$ok" == "false" ]]; then
        test_pass
    else
        test_fail "Expected ok=false for missing file" "Got ok='$ok'"
    fi

    cleanup_skills_env
}

# ============================================================================
# Test 11: skill-index-read uses cache when fresh
# ============================================================================
test_skill_read_index_uses_cache() {
    test_start "skill-index-read uses cache when fresh"
    setup_skills_env

    # Build index
    AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-index "$SKILLS_DIR" > /dev/null 2>&1 || true

    # Read from cache
    local output exit_code=0
    output=$(AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-index-read "$SKILLS_DIR" 2>/dev/null) || exit_code=$?

    if ! assert_exit_code "$exit_code" 0; then
        test_fail "Expected exit code 0" "Got $exit_code"
        cleanup_skills_env
        return 0
    fi

    if ! assert_json_valid "$output"; then
        test_fail "Expected valid JSON from cache read" "$output"
        cleanup_skills_env
        return 0
    fi

    local count
    count=$(echo "$output" | jq -r '.result.skill_count')
    if [[ "$count" == "2" ]]; then
        test_pass
    else
        test_fail "Expected 2 skills from cache" "Got $count"
    fi

    cleanup_skills_env
}

# ============================================================================
# Test 12: skill-detect detects matching files in repo
# ============================================================================
test_skill_detect_codebase() {
    test_start "skill-detect detects matching files in repo"
    setup_skills_env

    # Build index first
    AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-index "$SKILLS_DIR" > /dev/null 2>&1 || true

    # Create a test repo dir with a matching file
    local repo_dir="$TEST_DIR/repo"
    mkdir -p "$repo_dir"
    touch "$repo_dir/app.test.js"

    local output exit_code=0
    output=$(AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-detect "$repo_dir" "$SKILLS_DIR" 2>&1) || exit_code=$?

    if ! assert_exit_code "$exit_code" 0; then
        test_fail "Expected exit code 0" "Got $exit_code"
        cleanup_skills_env
        return 0
    fi

    if ! assert_json_valid "$output"; then
        test_fail "Expected valid JSON from skill-detect" "$output"
        cleanup_skills_env
        return 0
    fi

    local detection_count
    detection_count=$(echo "$output" | jq '[.result.detections[]?] | length')
    if [[ "$detection_count" -ge 1 ]]; then
        test_pass
    else
        test_fail "Expected at least 1 detection" "Got $detection_count"
    fi

    cleanup_skills_env
}

# ============================================================================
# Test 13: skill-detect detects matching packages in package.json
# ============================================================================
test_skill_detect_packages() {
    test_start "skill-detect detects matching packages in package.json"
    setup_skills_env

    # Build index
    AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-index "$SKILLS_DIR" > /dev/null 2>&1 || true

    # Create repo with package.json containing jest
    local repo_dir="$TEST_DIR/repo"
    mkdir -p "$repo_dir"
    cat > "$repo_dir/package.json" << 'EOF'
{
  "dependencies": {},
  "devDependencies": {
    "jest": "^29.0.0"
  }
}
EOF

    local output exit_code=0
    output=$(AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-detect "$repo_dir" "$SKILLS_DIR" 2>&1) || exit_code=$?

    if ! assert_exit_code "$exit_code" 0; then
        test_fail "Expected exit code 0" "Got $exit_code"
        cleanup_skills_env
        return 0
    fi

    local detection_count
    detection_count=$(echo "$output" | jq '[.result.detections[]?] | length')
    if [[ "$detection_count" -ge 1 ]]; then
        test_pass
    else
        test_fail "Expected at least 1 package detection" "Got $detection_count"
    fi

    cleanup_skills_env
}

# ============================================================================
# Test 14: skill-manifest-read returns manifest contents
# ============================================================================
test_skill_manifest_read() {
    test_start "skill-manifest-read returns manifest contents"
    setup_skills_env

    # Create a manifest
    cat > "$SKILLS_DIR/colony/.manifest.json" << 'EOF'
{
    "managed_by": "aether",
    "version": "2.1.0",
    "skills": ["test-skill"]
}
EOF

    local output exit_code=0
    output=$(bash "$AETHER_UTILS_SOURCE" skill-manifest-read "$SKILLS_DIR/colony/.manifest.json" 2>/dev/null) || exit_code=$?

    if ! assert_exit_code "$exit_code" 0; then
        test_fail "Expected exit code 0" "Got $exit_code"
        cleanup_skills_env
        return 0
    fi

    if ! assert_json_valid "$output"; then
        test_fail "Expected valid JSON from manifest read" "$output"
        cleanup_skills_env
        return 0
    fi

    local managed_by
    managed_by=$(echo "$output" | jq -r '.result.managed_by')
    if [[ "$managed_by" == "aether" ]]; then
        test_pass
    else
        test_fail "Expected managed_by='aether'" "Got '$managed_by'"
    fi

    cleanup_skills_env
}

# ============================================================================
# Test 15: skill-manifest-read returns default for missing manifest
# ============================================================================
test_skill_manifest_missing() {
    test_start "skill-manifest-read returns default for missing manifest"
    setup_skills_env

    local output exit_code=0
    output=$(bash "$AETHER_UTILS_SOURCE" skill-manifest-read "$SKILLS_DIR/nonexistent/.manifest.json" 2>/dev/null) || exit_code=$?

    if ! assert_exit_code "$exit_code" 0; then
        test_fail "Expected exit code 0" "Got $exit_code"
        cleanup_skills_env
        return 0
    fi

    if ! assert_json_valid "$output"; then
        test_fail "Expected valid JSON for missing manifest" "$output"
        cleanup_skills_env
        return 0
    fi

    local skill_count
    skill_count=$(echo "$output" | jq '.result.skills | length')
    if [[ "$skill_count" == "0" ]]; then
        test_pass
    else
        test_fail "Expected 0 skills in default manifest" "Got $skill_count"
    fi

    cleanup_skills_env
}

# ============================================================================
# Test 16: skill-is-user-created identifies user-created vs managed skills
# ============================================================================
test_skill_is_user_created() {
    test_start "skill-is-user-created identifies user-created skills"
    setup_skills_env

    cat > "$SKILLS_DIR/colony/.manifest.json" << 'EOF'
{
    "managed_by": "aether",
    "version": "2.1.0",
    "skills": ["test-skill"]
}
EOF

    # test-skill is in manifest -> not user-created
    local result1 exit_code1=0
    result1=$(bash "$AETHER_UTILS_SOURCE" skill-is-user-created "test-skill" "$SKILLS_DIR/colony/.manifest.json" 2>/dev/null) || exit_code1=$?

    # custom-skill is NOT in manifest -> user-created
    local result2 exit_code2=0
    result2=$(bash "$AETHER_UTILS_SOURCE" skill-is-user-created "custom-skill" "$SKILLS_DIR/colony/.manifest.json" 2>/dev/null) || exit_code2=$?

    if [[ "$result1" == "false" && "$result2" == "true" ]]; then
        test_pass
    else
        test_fail "Expected false+true" "Got '$result1' + '$result2'"
    fi

    cleanup_skills_env
}

# ============================================================================
# Test 17: skill-diff returns error for nonexistent skill
# ============================================================================
test_skill_diff_not_found() {
    test_start "skill-diff returns error for nonexistent skill"
    setup_skills_env

    local output exit_code=0
    output=$(bash "$AETHER_UTILS_SOURCE" skill-diff "nonexistent" "$SKILLS_DIR" 2>&1) || exit_code=$?

    local ok
    ok=$(echo "$output" | jq -r '.ok' 2>/dev/null)
    if [[ "$ok" == "false" ]]; then
        test_pass
    else
        test_fail "Expected ok=false for nonexistent skill" "Got ok='$ok'"
    fi

    cleanup_skills_env
}

# ============================================================================
# Test 18: skill-diff reports user_only when no system equivalent
# ============================================================================
test_skill_diff_user_only() {
    test_start "skill-diff reports user_only when no system equivalent"
    setup_skills_env

    local output exit_code=0
    output=$(AETHER_SYSTEM_DIR="$TEST_DIR/empty-system" bash "$AETHER_UTILS_SOURCE" skill-diff "test-skill" "$SKILLS_DIR" 2>&1) || exit_code=$?

    if ! assert_exit_code "$exit_code" 0; then
        test_fail "Expected exit code 0" "Got $exit_code"
        cleanup_skills_env
        return 0
    fi

    local status
    status=$(echo "$output" | jq -r '.result.status' 2>/dev/null)
    if [[ "$status" == "user_only" ]]; then
        test_pass
    else
        test_fail "Expected status='user_only'" "Got '$status'"
    fi

    cleanup_skills_env
}

# ============================================================================
# Test 19: skill-inject respects 12K char budget
# ============================================================================
test_skill_inject_budget() {
    test_start "skill-inject respects 12K char budget"
    setup_skills_env

    # Create a huge skill that would exceed 12K
    mkdir -p "$SKILLS_DIR/domain/huge-skill"
    {
        echo "---"
        echo "name: huge-skill"
        echo "description: A very large skill"
        echo "type: domain"
        echo "domains: [testing]"
        echo "agent_roles: [builder]"
        echo "priority: normal"
        echo 'version: "1.0"'
        echo "---"
        echo ""
        # Generate 15000 chars of content
        python3 -c "print('A' * 15000)"
    } > "$SKILLS_DIR/domain/huge-skill/SKILL.md"

    local match_json='{"colony_skills":[{"name":"test-skill","file_path":"'"$SKILLS_DIR"'/colony/test-skill/SKILL.md"}],"domain_skills":[{"name":"huge-skill","file_path":"'"$SKILLS_DIR"'/domain/huge-skill/SKILL.md"}]}'

    # Capture stdout only (stderr has trimming log messages)
    local output exit_code=0
    output=$(bash "$AETHER_UTILS_SOURCE" skill-inject "$match_json" 2>/dev/null) || exit_code=$?

    if ! assert_exit_code "$exit_code" 0; then
        test_fail "Expected exit code 0" "Got $exit_code"
        cleanup_skills_env
        return 0
    fi

    local total_chars
    total_chars=$(echo "$output" | jq -r '.result.total_chars')
    if [[ "$total_chars" -le 12000 ]]; then
        test_pass
    else
        test_fail "Expected total_chars <= 12000" "Got $total_chars"
    fi

    cleanup_skills_env
}

# ============================================================================
# Test 20: skill-index-read emits deprecation warning
# ============================================================================
test_skill_index_read_deprecation_warning() {
    test_start "skill-index-read emits deprecation warning"
    setup_skills_env

    # Build index first
    AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-index "$SKILLS_DIR" > /dev/null 2>&1 || true

    local _stderr
    _stderr=$(AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-index-read "$SKILLS_DIR" 2>&1 >/dev/null || true)
    if [[ "$_stderr" == *"[deprecated]"* ]]; then
        test_pass
    else
        test_fail "stderr contains [deprecated]" "$_stderr"
    fi

    cleanup_skills_env
}

# ============================================================================
# Test 21: skill-manifest-read emits deprecation warning
# ============================================================================
test_skill_manifest_read_deprecation_warning() {
    test_start "skill-manifest-read emits deprecation warning"
    setup_skills_env

    cat > "$SKILLS_DIR/colony/.manifest.json" << 'EOF'
{
    "managed_by": "aether",
    "version": "2.1.0",
    "skills": ["test-skill"]
}
EOF

    local _stderr
    _stderr=$(bash "$AETHER_UTILS_SOURCE" skill-manifest-read "$SKILLS_DIR/colony/.manifest.json" 2>&1 >/dev/null || true)
    if [[ "$_stderr" == *"[deprecated]"* ]]; then
        test_pass
    else
        test_fail "stderr contains [deprecated]" "$_stderr"
    fi

    cleanup_skills_env
}

# ============================================================================
# Test 22: skill-is-user-created emits deprecation warning
# ============================================================================
test_skill_is_user_created_deprecation_warning() {
    test_start "skill-is-user-created emits deprecation warning"
    setup_skills_env

    cat > "$SKILLS_DIR/colony/.manifest.json" << 'EOF'
{
    "managed_by": "aether",
    "version": "2.1.0",
    "skills": ["test-skill"]
}
EOF

    local _stderr
    _stderr=$(bash "$AETHER_UTILS_SOURCE" skill-is-user-created "test-skill" "$SKILLS_DIR/colony/.manifest.json" 2>&1 >/dev/null || true)
    if [[ "$_stderr" == *"[deprecated]"* ]]; then
        test_pass
    else
        test_fail "stderr contains [deprecated]" "$_stderr"
    fi

    cleanup_skills_env
}

# ============================================================================
# Run all tests
# ============================================================================

log_info "Running skills engine tests"
log_info "Repo root: $REPO_ROOT"
log_info "Aether utils: $AETHER_UTILS_SOURCE"

# Frontmatter parsing
test_parse_frontmatter
test_parse_frontmatter_domains
test_parse_frontmatter_detect
test_parse_frontmatter_missing_file

# Index building and caching
test_skill_index_build
test_skill_index_cache
test_skill_read_index_uses_cache

# Codebase detection
test_skill_detect_codebase
test_skill_detect_packages

# Skill matching
test_skill_match_by_role
test_skill_match_role_filter

# Skill injection
test_skill_inject
test_skill_inject_budget

# Skill listing
test_skill_list

# Manifest operations
test_skill_manifest_read
test_skill_manifest_missing

# User-created skill check
test_skill_is_user_created

# Skill diff
test_skill_diff_not_found
test_skill_diff_user_only

# Deprecation warnings
test_skill_index_read_deprecation_warning
test_skill_manifest_read_deprecation_warning
test_skill_is_user_created_deprecation_warning

test_summary
