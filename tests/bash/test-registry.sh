#!/usr/bin/env bash
# Registry Subcommand Tests
# Tests registry-add enhancements (domain_tags, last_colony_goal, active_colony)
# and the new registry-list subcommand

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
AETHER_UTILS_SOURCE="$PROJECT_ROOT/.aether/aether-utils.sh"

# Source test helpers
source "$SCRIPT_DIR/test-helpers.sh"

# Verify jq is available
require_jq

# Verify aether-utils.sh exists
if [[ ! -f "$AETHER_UTILS_SOURCE" ]]; then
    log_error "aether-utils.sh not found at: $AETHER_UTILS_SOURCE"
    exit 1
fi

# ============================================================================
# Helper: Create isolated test environment with its own HOME
# ============================================================================
setup_registry_env() {
    local tmp_dir
    tmp_dir=$(mktemp -d)
    mkdir -p "$tmp_dir/.aether/data"
    mkdir -p "$tmp_dir/fakehome/.aether"

    # Copy aether-utils.sh to temp location
    cp "$AETHER_UTILS_SOURCE" "$tmp_dir/.aether/aether-utils.sh"
    chmod +x "$tmp_dir/.aether/aether-utils.sh"

    # Copy utils directory if it exists
    local utils_source="$(dirname "$AETHER_UTILS_SOURCE")/utils"
    if [[ -d "$utils_source" ]]; then
        cp -r "$utils_source" "$tmp_dir/.aether/"
    fi

    # Copy exchange directory if it exists (needed for XML utils)
    local exchange_source="$(dirname "$AETHER_UTILS_SOURCE")/exchange"
    if [[ -d "$exchange_source" ]]; then
        cp -r "$exchange_source" "$tmp_dir/.aether/"
    fi

    # Copy schemas directory if it exists
    local schemas_source="$(dirname "$AETHER_UTILS_SOURCE")/schemas"
    if [[ -d "$schemas_source" ]]; then
        cp -r "$schemas_source" "$tmp_dir/.aether/"
    fi

    echo "$tmp_dir"
}

# ============================================================================
# Test: registry-add with --tags stores domain_tags as array
# ============================================================================
test_registry_add_with_tags() {
    local tmp_dir
    tmp_dir=$(setup_registry_env)

    local output
    output=$(HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-add /tmp/test-repo 1.0.0 --tags "node,api,web" 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify registry.json has domain_tags as array
    local registry_file="$tmp_dir/fakehome/.aether/registry.json"
    if [[ ! -f "$registry_file" ]]; then
        test_fail "registry.json exists" "file not found"
        rm -rf "$tmp_dir"
        return 1
    fi

    local tags
    tags=$(jq -r '.repos[0].domain_tags | join(",")' "$registry_file" 2>/dev/null)
    if [[ "$tags" != "node,api,web" ]]; then
        test_fail "domain_tags=[node,api,web]" "got: $tags"
        rm -rf "$tmp_dir"
        return 1
    fi

    local tag_count
    tag_count=$(jq '.repos[0].domain_tags | length' "$registry_file" 2>/dev/null)
    if [[ "$tag_count" -ne 3 ]]; then
        test_fail "3 domain_tags" "got: $tag_count"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: registry-add with --goal stores last_colony_goal
# ============================================================================
test_registry_add_with_goal() {
    local tmp_dir
    tmp_dir=$(setup_registry_env)

    local output
    output=$(HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-add /tmp/test-repo 1.0.0 --goal "Build the API" 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify registry.json has last_colony_goal
    local registry_file="$tmp_dir/fakehome/.aether/registry.json"
    local goal
    goal=$(jq -r '.repos[0].last_colony_goal' "$registry_file" 2>/dev/null)
    if [[ "$goal" != "Build the API" ]]; then
        test_fail "last_colony_goal='Build the API'" "got: $goal"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: registry-add with --active stores active_colony boolean
# ============================================================================
test_registry_add_with_active() {
    local tmp_dir
    tmp_dir=$(setup_registry_env)

    local output
    output=$(HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-add /tmp/test-repo 1.0.0 --active true 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify registry.json has active_colony as boolean true
    local registry_file="$tmp_dir/fakehome/.aether/registry.json"
    local active
    active=$(jq '.repos[0].active_colony' "$registry_file" 2>/dev/null)
    if [[ "$active" != "true" ]]; then
        test_fail "active_colony=true" "got: $active"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: registry-add with --active false stores false
# ============================================================================
test_registry_add_active_false() {
    local tmp_dir
    tmp_dir=$(setup_registry_env)

    local output
    output=$(HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-add /tmp/test-repo 1.0.0 --active false 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    local registry_file="$tmp_dir/fakehome/.aether/registry.json"
    local active
    active=$(jq '.repos[0].active_colony' "$registry_file" 2>/dev/null)
    if [[ "$active" != "false" ]]; then
        test_fail "active_colony=false" "got: $active"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: registry-add with all flags combined
# ============================================================================
test_registry_add_all_flags() {
    local tmp_dir
    tmp_dir=$(setup_registry_env)

    local output
    output=$(HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-add /tmp/test-repo 2.0.0 \
        --tags "react,frontend" --goal "Build the UI" --active true 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    local registry_file="$tmp_dir/fakehome/.aether/registry.json"

    local tags
    tags=$(jq -r '.repos[0].domain_tags | join(",")' "$registry_file" 2>/dev/null)
    if [[ "$tags" != "react,frontend" ]]; then
        test_fail "domain_tags=[react,frontend]" "got: $tags"
        rm -rf "$tmp_dir"
        return 1
    fi

    local goal
    goal=$(jq -r '.repos[0].last_colony_goal' "$registry_file" 2>/dev/null)
    if [[ "$goal" != "Build the UI" ]]; then
        test_fail "last_colony_goal='Build the UI'" "got: $goal"
        rm -rf "$tmp_dir"
        return 1
    fi

    local active
    active=$(jq '.repos[0].active_colony' "$registry_file" 2>/dev/null)
    if [[ "$active" != "true" ]]; then
        test_fail "active_colony=true" "got: $active"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: registry-add without new flags defaults gracefully
# ============================================================================
test_registry_add_defaults() {
    local tmp_dir
    tmp_dir=$(setup_registry_env)

    local output
    output=$(HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-add /tmp/test-repo 1.0.0 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    local registry_file="$tmp_dir/fakehome/.aether/registry.json"

    # domain_tags should be empty array
    local tag_count
    tag_count=$(jq '.repos[0].domain_tags | length' "$registry_file" 2>/dev/null)
    if [[ "$tag_count" -ne 0 ]]; then
        test_fail "domain_tags=[] (empty)" "got length: $tag_count"
        rm -rf "$tmp_dir"
        return 1
    fi

    # last_colony_goal should be null
    local goal
    goal=$(jq -r '.repos[0].last_colony_goal' "$registry_file" 2>/dev/null)
    if [[ "$goal" != "null" ]]; then
        test_fail "last_colony_goal=null" "got: $goal"
        rm -rf "$tmp_dir"
        return 1
    fi

    # active_colony should default to false
    local active
    active=$(jq '.repos[0].active_colony' "$registry_file" 2>/dev/null)
    if [[ "$active" != "false" ]]; then
        test_fail "active_colony=false (default)" "got: $active"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: registry-add update preserves and updates new fields
# ============================================================================
test_registry_add_update_preserves_fields() {
    local tmp_dir
    tmp_dir=$(setup_registry_env)

    # First add
    HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-add /tmp/test-repo 1.0.0 --tags "node" --goal "Phase 1" --active true 2>&1 >/dev/null

    # Update with new values
    local output
    output=$(HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-add /tmp/test-repo 2.0.0 --tags "node,api" --goal "Phase 2" 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    local registry_file="$tmp_dir/fakehome/.aether/registry.json"

    # Version should be updated
    local ver
    ver=$(jq -r '.repos[0].version' "$registry_file" 2>/dev/null)
    if [[ "$ver" != "2.0.0" ]]; then
        test_fail "version=2.0.0" "got: $ver"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Tags should be updated
    local tags
    tags=$(jq -r '.repos[0].domain_tags | join(",")' "$registry_file" 2>/dev/null)
    if [[ "$tags" != "node,api" ]]; then
        test_fail "domain_tags=[node,api]" "got: $tags"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Goal should be updated
    local goal
    goal=$(jq -r '.repos[0].last_colony_goal' "$registry_file" 2>/dev/null)
    if [[ "$goal" != "Phase 2" ]]; then
        test_fail "last_colony_goal='Phase 2'" "got: $goal"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: backwards compatibility — existing entries without new fields parse OK
# ============================================================================
test_registry_backwards_compat() {
    local tmp_dir
    tmp_dir=$(setup_registry_env)

    # Create a legacy registry.json without the new fields
    mkdir -p "$tmp_dir/fakehome/.aether"
    cat > "$tmp_dir/fakehome/.aether/registry.json" << 'EOF'
{
  "schema_version": 1,
  "repos": [
    {
      "path": "/tmp/legacy-repo",
      "version": "0.9.0",
      "registered_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    }
  ]
}
EOF

    # Add a new entry — legacy entry should survive
    local output
    output=$(HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-add /tmp/new-repo 1.0.0 --tags "go" 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    local registry_file="$tmp_dir/fakehome/.aether/registry.json"

    # Legacy entry should still be present
    local legacy_ver
    legacy_ver=$(jq -r '.repos[] | select(.path == "/tmp/legacy-repo") | .version' "$registry_file" 2>/dev/null)
    if [[ "$legacy_ver" != "0.9.0" ]]; then
        test_fail "legacy entry preserved" "got: $legacy_ver"
        rm -rf "$tmp_dir"
        return 1
    fi

    # New entry should have tags
    local new_tags
    new_tags=$(jq -r '.repos[] | select(.path == "/tmp/new-repo") | .domain_tags | join(",")' "$registry_file" 2>/dev/null)
    if [[ "$new_tags" != "go" ]]; then
        test_fail "new entry has tags" "got: $new_tags"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Repo count should be 2
    local count
    count=$(jq '.repos | length' "$registry_file" 2>/dev/null)
    if [[ "$count" -ne 2 ]]; then
        test_fail "2 repos" "got: $count"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: registry-list outputs valid JSON
# ============================================================================
test_registry_list_json() {
    local tmp_dir
    tmp_dir=$(setup_registry_env)

    # Add two repos
    HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-add /tmp/repo-a 1.0.0 --tags "node,api" --goal "Build API" --active true 2>&1 >/dev/null
    HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-add /tmp/repo-b 2.0.0 --tags "react" --goal "Build UI" 2>&1 >/dev/null

    local output
    output=$(HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-list 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Should have 2 repos in the result
    local repo_count
    repo_count=$(echo "$output" | jq '.result.repos | length' 2>/dev/null)
    if [[ "$repo_count" -ne 2 ]]; then
        test_fail "2 repos in list" "got: $repo_count"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: registry-list with empty registry
# ============================================================================
test_registry_list_empty() {
    local tmp_dir
    tmp_dir=$(setup_registry_env)

    local output
    output=$(HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-list 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Should have 0 repos
    local repo_count
    repo_count=$(echo "$output" | jq '.result.repos | length' 2>/dev/null)
    if [[ "$repo_count" -ne 0 ]]; then
        test_fail "0 repos in empty list" "got: $repo_count"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: registry-list includes domain_tags, goal, and active fields
# ============================================================================
test_registry_list_includes_new_fields() {
    local tmp_dir
    tmp_dir=$(setup_registry_env)

    HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-add /tmp/repo-x 1.0.0 --tags "python,ml" --goal "Train model" --active true 2>&1 >/dev/null

    local output
    output=$(HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-list 2>&1)

    # Check that the repo entry has domain_tags
    local tags
    tags=$(echo "$output" | jq -r '.result.repos[0].domain_tags | join(",")' 2>/dev/null)
    if [[ "$tags" != "python,ml" ]]; then
        test_fail "domain_tags=[python,ml]" "got: $tags"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Check last_colony_goal
    local goal
    goal=$(echo "$output" | jq -r '.result.repos[0].last_colony_goal' 2>/dev/null)
    if [[ "$goal" != "Train model" ]]; then
        test_fail "last_colony_goal='Train model'" "got: $goal"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Check active_colony
    local active
    active=$(echo "$output" | jq '.result.repos[0].active_colony' 2>/dev/null)
    if [[ "$active" != "true" ]]; then
        test_fail "active_colony=true" "got: $active"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: registry-list handles legacy entries without new fields
# ============================================================================
test_registry_list_legacy_entries() {
    local tmp_dir
    tmp_dir=$(setup_registry_env)

    # Create a legacy registry with entries that lack new fields
    mkdir -p "$tmp_dir/fakehome/.aether"
    cat > "$tmp_dir/fakehome/.aether/registry.json" << 'EOF'
{
  "schema_version": 1,
  "repos": [
    {
      "path": "/tmp/old-repo",
      "version": "0.5.0",
      "registered_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z"
    }
  ]
}
EOF

    local output
    output=$(HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-list 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Legacy entry should show with defaults for missing fields
    local repo_count
    repo_count=$(echo "$output" | jq '.result.repos | length' 2>/dev/null)
    if [[ "$repo_count" -ne 1 ]]; then
        test_fail "1 repo listed" "got: $repo_count"
        rm -rf "$tmp_dir"
        return 1
    fi

    # domain_tags should default to empty array
    local tag_count
    tag_count=$(echo "$output" | jq '.result.repos[0].domain_tags | length' 2>/dev/null)
    if [[ "$tag_count" -ne 0 ]]; then
        test_fail "legacy domain_tags=[] (empty)" "got length: $tag_count"
        rm -rf "$tmp_dir"
        return 1
    fi

    # active_colony should default to false
    local active
    active=$(echo "$output" | jq '.result.repos[0].active_colony' 2>/dev/null)
    if [[ "$active" != "false" ]]; then
        test_fail "legacy active_colony=false" "got: $active"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Main: Run all tests
# ============================================================================
main() {
    log_info "Running registry subcommand tests"

    # registry-add enhancements
    run_test "test_registry_add_with_tags" "registry-add --tags stores domain_tags as JSON array"
    run_test "test_registry_add_with_goal" "registry-add --goal stores last_colony_goal"
    run_test "test_registry_add_with_active" "registry-add --active true stores boolean true"
    run_test "test_registry_add_active_false" "registry-add --active false stores boolean false"
    run_test "test_registry_add_all_flags" "registry-add with all flags stores all fields"
    run_test "test_registry_add_defaults" "registry-add without new flags uses defaults"
    run_test "test_registry_add_update_preserves_fields" "registry-add update preserves and updates new fields"
    run_test "test_registry_backwards_compat" "backwards compatibility with legacy entries"

    # registry-list subcommand
    run_test "test_registry_list_json" "registry-list returns valid JSON with repos"
    run_test "test_registry_list_empty" "registry-list with empty registry returns empty array"
    run_test "test_registry_list_includes_new_fields" "registry-list includes domain_tags, goal, active fields"
    run_test "test_registry_list_legacy_entries" "registry-list handles legacy entries without new fields"

    # Print summary
    test_summary
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
