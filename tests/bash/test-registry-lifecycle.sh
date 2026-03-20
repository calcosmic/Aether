#!/usr/bin/env bash
# Registry Lifecycle Tests
# Tests that registry-add correctly simulates init (--goal + --active true)
# and seal (--active false) lifecycle updates
#
# These tests verify the registry-add subcommand behaves correctly when called
# the way init.md and seal.md instruct agents to call it.

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

    # Copy exchange directory if it exists
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
# Test: Init lifecycle — registry-add sets goal and active=true
# Simulates what init.md Step 6.6 should do
# ============================================================================
test_init_lifecycle_sets_goal_and_active() {
    local tmp_dir
    tmp_dir=$(setup_registry_env)
    local registry_file="$tmp_dir/fakehome/.aether/registry.json"

    # Simulate what init.md should call:
    # registry-add "$(pwd)" "<version>" --goal "<colony_goal>" --active true
    local output
    output=$(HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-add "/tmp/my-project" "1.3.0" \
        --goal "Build a REST API" --active true 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify active_colony is true
    local active
    active=$(jq '.repos[0].active_colony' "$registry_file" 2>/dev/null)
    if [[ "$active" != "true" ]]; then
        test_fail "active_colony=true after init" "got: $active"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify last_colony_goal is set
    local goal
    goal=$(jq -r '.repos[0].last_colony_goal' "$registry_file" 2>/dev/null)
    if [[ "$goal" != "Build a REST API" ]]; then
        test_fail "last_colony_goal='Build a REST API'" "got: $goal"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: Seal lifecycle — registry-add sets active=false
# Simulates what seal.md should do after sealing
# ============================================================================
test_seal_lifecycle_sets_active_false() {
    local tmp_dir
    tmp_dir=$(setup_registry_env)
    local registry_file="$tmp_dir/fakehome/.aether/registry.json"

    # First, simulate init (registers repo with active=true)
    HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-add "/tmp/my-project" "1.3.0" \
        --goal "Build a REST API" --active true 2>&1 >/dev/null

    # Verify active is true before seal
    local active_before
    active_before=$(jq '.repos[0].active_colony' "$registry_file" 2>/dev/null)
    if [[ "$active_before" != "true" ]]; then
        test_fail "active_colony=true before seal" "got: $active_before"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Simulate what seal.md should call:
    # registry-add "$(pwd)" "<version>" --active false
    local output
    output=$(HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-add "/tmp/my-project" "1.3.0" --active false 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify active_colony is now false
    local active_after
    active_after=$(jq '.repos[0].active_colony' "$registry_file" 2>/dev/null)
    if [[ "$active_after" != "false" ]]; then
        test_fail "active_colony=false after seal" "got: $active_after"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify goal is preserved (seal doesn't clear the goal)
    local goal
    goal=$(jq -r '.repos[0].last_colony_goal' "$registry_file" 2>/dev/null)
    if [[ "$goal" != "Build a REST API" ]]; then
        test_fail "last_colony_goal preserved after seal" "got: $goal"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: Full lifecycle — init then seal updates registry correctly
# ============================================================================
test_full_lifecycle_init_then_seal() {
    local tmp_dir
    tmp_dir=$(setup_registry_env)
    local registry_file="$tmp_dir/fakehome/.aether/registry.json"

    # Step 1: Init — register with goal and active=true
    HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-add "/tmp/my-project" "1.3.0" \
        --goal "Implement Aether v2.0" --active true 2>&1 >/dev/null

    # Verify state after init
    local active_init
    active_init=$(jq '.repos[0].active_colony' "$registry_file" 2>/dev/null)
    local goal_init
    goal_init=$(jq -r '.repos[0].last_colony_goal' "$registry_file" 2>/dev/null)

    if [[ "$active_init" != "true" || "$goal_init" != "Implement Aether v2.0" ]]; then
        test_fail "init state: active=true, goal set" "active=$active_init, goal=$goal_init"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Step 2: Seal — deactivate colony
    HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-add "/tmp/my-project" "1.3.0" --active false 2>&1 >/dev/null

    # Verify state after seal
    local active_seal
    active_seal=$(jq '.repos[0].active_colony' "$registry_file" 2>/dev/null)
    local goal_seal
    goal_seal=$(jq -r '.repos[0].last_colony_goal' "$registry_file" 2>/dev/null)

    if [[ "$active_seal" != "false" ]]; then
        test_fail "active_colony=false after seal" "got: $active_seal"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Goal should be preserved even after seal
    if [[ "$goal_seal" != "Implement Aether v2.0" ]]; then
        test_fail "goal preserved after seal" "got: $goal_seal"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Step 3: Re-init with new goal
    HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-add "/tmp/my-project" "1.3.0" \
        --goal "Build feature Y" --active true 2>&1 >/dev/null

    # Verify state after re-init
    local active_reinit
    active_reinit=$(jq '.repos[0].active_colony' "$registry_file" 2>/dev/null)
    local goal_reinit
    goal_reinit=$(jq -r '.repos[0].last_colony_goal' "$registry_file" 2>/dev/null)

    if [[ "$active_reinit" != "true" ]]; then
        test_fail "active_colony=true after re-init" "got: $active_reinit"
        rm -rf "$tmp_dir"
        return 1
    fi

    if [[ "$goal_reinit" != "Build feature Y" ]]; then
        test_fail "goal updated on re-init" "got: $goal_reinit"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: Seal on unregistered repo — gracefully registers then deactivates
# ============================================================================
test_seal_unregistered_repo() {
    local tmp_dir
    tmp_dir=$(setup_registry_env)
    local registry_file="$tmp_dir/fakehome/.aether/registry.json"

    # Seal without prior init — should still work (registry-add upserts)
    local output
    output=$(HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-add "/tmp/new-project" "1.0.0" --active false 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Repo should be registered with active=false
    local active
    active=$(jq '.repos[0].active_colony' "$registry_file" 2>/dev/null)
    if [[ "$active" != "false" ]]; then
        test_fail "active_colony=false for unregistered repo" "got: $active"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Goal should be null (no goal was set)
    local goal
    goal=$(jq -r '.repos[0].last_colony_goal' "$registry_file" 2>/dev/null)
    if [[ "$goal" != "null" ]]; then
        test_fail "last_colony_goal=null for unregistered repo" "got: $goal"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: Init with special characters in goal
# ============================================================================
test_init_special_chars_in_goal() {
    local tmp_dir
    tmp_dir=$(setup_registry_env)
    local registry_file="$tmp_dir/fakehome/.aether/registry.json"

    local output
    output=$(HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-add "/tmp/my-project" "1.0.0" \
        --goal "Build API with auth & rate-limiting (v2)" --active true 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    local goal
    goal=$(jq -r '.repos[0].last_colony_goal' "$registry_file" 2>/dev/null)
    if [[ "$goal" != "Build API with auth & rate-limiting (v2)" ]]; then
        test_fail "goal with special chars preserved" "got: $goal"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: Multiple repos — seal only affects the target repo
# ============================================================================
test_seal_only_affects_target_repo() {
    local tmp_dir
    tmp_dir=$(setup_registry_env)
    local registry_file="$tmp_dir/fakehome/.aether/registry.json"

    # Init two repos
    HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-add "/tmp/project-a" "1.0.0" \
        --goal "Project A" --active true 2>&1 >/dev/null

    HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-add "/tmp/project-b" "1.0.0" \
        --goal "Project B" --active true 2>&1 >/dev/null

    # Seal only project-a
    HOME="$tmp_dir/fakehome" bash "$tmp_dir/.aether/aether-utils.sh" \
        registry-add "/tmp/project-a" "1.0.0" --active false 2>&1 >/dev/null

    # Project A should be inactive
    local active_a
    active_a=$(jq '.repos[] | select(.path == "/tmp/project-a") | .active_colony' "$registry_file" 2>/dev/null)
    if [[ "$active_a" != "false" ]]; then
        test_fail "project-a active=false after seal" "got: $active_a"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Project B should still be active
    local active_b
    active_b=$(jq '.repos[] | select(.path == "/tmp/project-b") | .active_colony' "$registry_file" 2>/dev/null)
    if [[ "$active_b" != "true" ]]; then
        test_fail "project-b active=true (unaffected)" "got: $active_b"
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
    log_info "Running registry lifecycle tests"

    # Init lifecycle
    run_test "test_init_lifecycle_sets_goal_and_active" "init lifecycle: registry-add sets goal and active=true"
    run_test "test_seal_lifecycle_sets_active_false" "seal lifecycle: registry-add sets active=false"
    run_test "test_full_lifecycle_init_then_seal" "full lifecycle: init -> seal -> re-init"
    run_test "test_seal_unregistered_repo" "seal on unregistered repo: graceful upsert with active=false"
    run_test "test_init_special_chars_in_goal" "init with special characters in goal"
    run_test "test_seal_only_affects_target_repo" "seal only affects target repo, not others"

    # Print summary
    test_summary
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
