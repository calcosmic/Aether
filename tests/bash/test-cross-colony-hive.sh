#!/usr/bin/env bash
# Tests for Phase 19: Cross-colony hive wisdom flow
# Covers: seal domain tag fix, domain auto-detection, queen-seed-from-hive,
# deduplication, domain scoping, and end-to-end seal-to-init flow.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

# ============================================================================
# Helper: Create isolated test environment
# ============================================================================
setup_hive_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    mkdir -p "$tmpdir/.aether/data"
    mkdir -p "$tmpdir/.aether/templates"
    # Copy the QUEEN.md template so queen-init can find it
    cp "$REPO_ROOT/.aether/templates/QUEEN.md.template" "$tmpdir/.aether/templates/"
    echo "$tmpdir"
}

# Helper: run aether-utils against a test env
run_utils() {
    local tmpdir="$1"
    shift
    HOME="$tmpdir" AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$AETHER_UTILS" "$@" 2>&1
}

# ============================================================================
# Test 1: Seal promotes with registry domain tags
# ============================================================================
test_seal_promotes_with_registry_domain_tags() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    # Create colony state with high-confidence instincts in .memory.instincts[]
    cat > "$tmpdir/.aether/data/COLONY_STATE.json" << 'COLSTATE'
{
  "goal": "Test colony",
  "state": "active",
  "current_phase": 2,
  "version": "3.0",
  "initialized_at": "2026-03-01T00:00:00Z",
  "plan": {"phases": []},
  "memory": {
    "instincts": [
      {"trigger": "writing API endpoints", "action": "validate input first", "confidence": 0.9, "domain": "testing", "source": "phase-1", "evidence": ["test"]},
      {"trigger": "deploying services", "action": "run smoke tests", "confidence": 0.85, "domain": "patterns", "source": "phase-1", "evidence": ["deploy"]}
    ]
  },
  "errors": {"records": []},
  "events": [],
  "session_id": "test-session"
}
COLSTATE

    # Create registry with domain_tags for this test repo path
    mkdir -p "$tmpdir/.aether"
    cat > "$tmpdir/.aether/registry.json" << REGISTRY
{
  "repos": [
    {
      "path": "$tmpdir",
      "version": "2.0.0",
      "domain_tags": ["node", "typescript"],
      "active_colony": true
    }
  ]
}
REGISTRY

    # Initialize hive
    run_utils "$tmpdir" hive-init > /dev/null

    # Simulate the seal Step 3.7 snippet: read domain tags from registry and promote
    local repo_domain_tags
    repo_domain_tags=$(jq -r --arg repo "$tmpdir" \
      '[.repos[] | select(.path == $repo) | .domain_tags // []] | .[0] // [] | join(",")' \
      "$tmpdir/.aether/registry.json" 2>/dev/null || echo "")

    local high_conf_instincts
    high_conf_instincts=$(jq -r '.memory.instincts[] | select(.confidence >= 0.8) | @base64' \
      "$tmpdir/.aether/data/COLONY_STATE.json" 2>/dev/null || echo "")

    local hive_promoted_count=0
    for encoded in $high_conf_instincts; do
        [[ -z "$encoded" ]] && continue
        local trigger action confidence
        trigger=$(echo "$encoded" | base64 -d | jq -r '.trigger // empty')
        action=$(echo "$encoded" | base64 -d | jq -r '.action // empty')
        confidence=$(echo "$encoded" | base64 -d | jq -r '.confidence // 0.7')
        trigger_clean=$(echo "$trigger" | sed 's/^[Ww]hen //')
        local promote_text="When ${trigger_clean}: ${action}"
        local promote_args=(hive-promote --text "$promote_text" --source-repo "$tmpdir" --confidence "$confidence")
        [[ -n "$repo_domain_tags" ]] && promote_args+=(--domain "$repo_domain_tags")
        local result
        result=$(run_utils "$tmpdir" "${promote_args[@]}") || true
        local was_promoted
        was_promoted=$(echo "$result" | jq -r '.result.action // "skipped"' 2>/dev/null || echo "skipped")
        if [[ "$was_promoted" == "promoted" || "$was_promoted" == "merged" ]]; then
            hive_promoted_count=$((hive_promoted_count + 1))
        fi
    done

    # Should have promoted 2 instincts
    if [[ "$hive_promoted_count" -ne 2 ]]; then
        test_fail "should promote 2 instincts" "got: $hive_promoted_count"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify hive entries have node/typescript domain tags (not "testing" or "patterns")
    local read_result
    read_result=$(run_utils "$tmpdir" hive-read --domain "node" --format json)
    local node_count
    node_count=$(echo "$read_result" | jq -r '.result.total_matched // 0' 2>/dev/null)

    if [[ "$node_count" -lt 1 ]]; then
        test_fail "should find entries with node domain tag" "got: $node_count"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify that "testing" (the instinct.domain) is NOT used as a domain_tag
    local testing_result
    testing_result=$(run_utils "$tmpdir" hive-read --domain "testing" --format json)
    local testing_count
    testing_count=$(echo "$testing_result" | jq -r '.result.total_matched // 0' 2>/dev/null)

    if [[ "$testing_count" -gt 0 ]]; then
        test_fail "should NOT find entries with instinct domain 'testing'" "got: $testing_count"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 2: Domain detection from file presence
# ============================================================================
test_domain_detect_from_file_presence() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    # Create package.json and tsconfig.json
    echo '{"name":"test"}' > "$tmpdir/package.json"
    echo '{}' > "$tmpdir/tsconfig.json"

    local result
    result=$(run_utils "$tmpdir" domain-detect)

    if ! assert_ok_true "$result"; then
        test_fail "domain-detect should return ok:true" "got: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local tags
    tags=$(echo "$result" | jq -r '.result.tags // ""')

    if ! assert_contains "$tags" "node"; then
        test_fail "should detect node from package.json" "got: $tags"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_contains "$tags" "typescript"; then
        test_fail "should detect typescript from tsconfig.json" "got: $tags"
        rm -rf "$tmpdir"
        return 1
    fi

    # Test Rust detection
    local tmpdir2
    tmpdir2=$(setup_hive_env)
    echo '[package]' > "$tmpdir2/Cargo.toml"

    local result2
    result2=$(run_utils "$tmpdir2" domain-detect)
    local tags2
    tags2=$(echo "$result2" | jq -r '.result.tags // ""')

    if ! assert_contains "$tags2" "rust"; then
        test_fail "should detect rust from Cargo.toml" "got: $tags2"
        rm -rf "$tmpdir" "$tmpdir2"
        return 1
    fi

    # Test empty env (no known files)
    local tmpdir3
    tmpdir3=$(setup_hive_env)

    local result3
    result3=$(run_utils "$tmpdir3" domain-detect)
    local tags3
    tags3=$(echo "$result3" | jq -r '.result.tags // ""')

    if [[ -n "$tags3" ]]; then
        test_fail "should have empty tags for unknown project" "got: $tags3"
        rm -rf "$tmpdir" "$tmpdir2" "$tmpdir3"
        return 1
    fi

    rm -rf "$tmpdir" "$tmpdir2" "$tmpdir3"
    return 0
}

# ============================================================================
# Test 3: queen-seed-from-hive writes entries
# ============================================================================
test_queen_seed_from_hive_writes_entries() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    # Initialize hive and store 3 entries with domain "node"
    run_utils "$tmpdir" hive-init > /dev/null

    run_utils "$tmpdir" hive-store \
        --text "Always validate input before processing" \
        --domain "node" --source-repo "/other/repo" --confidence 0.8 > /dev/null

    run_utils "$tmpdir" hive-store \
        --text "Use async/await instead of callbacks" \
        --domain "node" --source-repo "/other/repo" --confidence 0.75 > /dev/null

    run_utils "$tmpdir" hive-store \
        --text "Log errors with structured context" \
        --domain "node" --source-repo "/other/repo" --confidence 0.9 > /dev/null

    # Initialize QUEEN.md from template
    run_utils "$tmpdir" queen-init > /dev/null

    # Seed from hive
    local result
    result=$(run_utils "$tmpdir" queen-seed-from-hive --domain "node" --limit 3)

    if ! assert_ok_true "$result"; then
        test_fail "queen-seed-from-hive should return ok:true" "got: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local seeded
    seeded=$(echo "$result" | jq -r '.result.seeded // 0')

    if [[ "$seeded" -ne 3 ]]; then
        test_fail "should seed 3 entries" "got: $seeded"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify QUEEN.md contains [hive] tagged entries
    local queen_file="$tmpdir/.aether/QUEEN.md"
    local hive_entries
    hive_entries=$(grep -c '\[hive\]' "$queen_file" || true)

    if [[ "$hive_entries" -ne 3 ]]; then
        test_fail "QUEEN.md should have 3 [hive] entries" "got: $hive_entries"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify placeholder is removed
    if grep -q "No codebase patterns recorded yet" "$queen_file"; then
        test_fail "placeholder should be removed" "still found in QUEEN.md"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 4: queen-seed deduplicates existing entries
# ============================================================================
test_queen_seed_deduplicates_existing_entries() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    # Initialize hive with 2 entries
    run_utils "$tmpdir" hive-init > /dev/null

    run_utils "$tmpdir" hive-store \
        --text "Always validate input before processing" \
        --domain "node" --source-repo "/other/repo" --confidence 0.8 > /dev/null

    run_utils "$tmpdir" hive-store \
        --text "Use structured error logging" \
        --domain "node" --source-repo "/other/repo" --confidence 0.7 > /dev/null

    # Initialize QUEEN.md and pre-populate with one of the entries
    run_utils "$tmpdir" queen-init > /dev/null

    local queen_file="$tmpdir/.aether/QUEEN.md"
    # Insert the first entry manually into Codebase Patterns section
    # (simulating it was already there from a previous seed)
    local section_line
    section_line=$(grep -n '^## Codebase Patterns$' "$queen_file" | head -1 | cut -d: -f1)
    local placeholder_line
    placeholder_line=$(grep -n 'No codebase patterns recorded yet' "$queen_file" | head -1 | cut -d: -f1)

    # Replace the placeholder with the pre-existing entry
    if [[ -n "$placeholder_line" ]]; then
        sed -i.bak "${placeholder_line}s|.*|- [hive] Always validate input before processing (cross-colony, confidence: 0.8)|" "$queen_file"
        rm -f "${queen_file}.bak"
    fi

    # Now seed from hive
    local result
    result=$(run_utils "$tmpdir" queen-seed-from-hive --domain "node" --limit 5)

    local seeded
    seeded=$(echo "$result" | jq -r '.result.seeded // 0')

    # Should only seed 1 (the second entry) since first is already present
    if [[ "$seeded" -ne 1 ]]; then
        test_fail "should seed only 1 new entry (dedup)" "got: $seeded"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify no duplicates in QUEEN.md
    local validate_count
    validate_count=$(grep -c "Always validate input" "$queen_file" || true)

    if [[ "$validate_count" -ne 1 ]]; then
        test_fail "should not duplicate existing entry" "found $validate_count occurrences"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 5: queen-seed domain scoping
# ============================================================================
test_queen_seed_domain_scoping() {
    local tmpdir
    tmpdir=$(setup_hive_env)

    # Initialize hive with entries in different domains
    run_utils "$tmpdir" hive-init > /dev/null

    # 2 node-only entries
    run_utils "$tmpdir" hive-store \
        --text "Use package-lock.json for deterministic installs" \
        --domain "node" --source-repo "/repo-a" --confidence 0.8 > /dev/null

    run_utils "$tmpdir" hive-store \
        --text "Prefer async/await over raw promises" \
        --domain "node" --source-repo "/repo-a" --confidence 0.75 > /dev/null

    # 2 rust-only entries
    run_utils "$tmpdir" hive-store \
        --text "Use Result type for error handling" \
        --domain "rust" --source-repo "/repo-b" --confidence 0.85 > /dev/null

    run_utils "$tmpdir" hive-store \
        --text "Prefer owned types over references in public APIs" \
        --domain "rust" --source-repo "/repo-b" --confidence 0.7 > /dev/null

    # 1 node+rust entry
    run_utils "$tmpdir" hive-store \
        --text "Always handle errors explicitly" \
        --domain "node,rust" --source-repo "/repo-c" --confidence 0.9 > /dev/null

    # Initialize QUEEN.md
    run_utils "$tmpdir" queen-init > /dev/null

    # Seed with node domain only
    local result
    result=$(run_utils "$tmpdir" queen-seed-from-hive --domain "node" --limit 10)

    local seeded
    seeded=$(echo "$result" | jq -r '.result.seeded // 0')

    # Should get 3 entries: 2 node-only + 1 node,rust
    if [[ "$seeded" -ne 3 ]]; then
        test_fail "should seed 3 node-matching entries" "got: $seeded"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify rust-only entries are NOT in QUEEN.md
    local queen_file="$tmpdir/.aether/QUEEN.md"

    if grep -q "Result type for error" "$queen_file"; then
        test_fail "rust-only entry should NOT be in QUEEN.md" "found 'Result type' entry"
        rm -rf "$tmpdir"
        return 1
    fi

    if grep -q "Prefer owned types" "$queen_file"; then
        test_fail "rust-only entry should NOT be in QUEEN.md" "found 'owned types' entry"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify node+rust entry IS present
    if ! grep -q "Always handle errors explicitly" "$queen_file"; then
        test_fail "node,rust entry SHOULD be in QUEEN.md" "not found"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 6: End-to-end seal-to-init flow
# ============================================================================
test_end_to_end_seal_to_init_flow() {
    local shared_home
    shared_home=$(mktemp -d)

    # --- Repo A: seal promotes instincts to hive ---
    local repo_a="$shared_home/repo-a"
    mkdir -p "$repo_a/.aether/data"
    mkdir -p "$repo_a/.aether/templates"
    cp "$REPO_ROOT/.aether/templates/QUEEN.md.template" "$repo_a/.aether/templates/"

    # Colony state with high-confidence instincts
    cat > "$repo_a/.aether/data/COLONY_STATE.json" << 'COLSTATE'
{
  "goal": "Build web app",
  "state": "active",
  "current_phase": 3,
  "version": "3.0",
  "initialized_at": "2026-03-01T00:00:00Z",
  "plan": {"phases": []},
  "memory": {
    "instincts": [
      {"trigger": "creating API routes", "action": "add input validation middleware", "confidence": 0.92, "domain": "security", "source": "phase-2", "evidence": ["api"]},
      {"trigger": "writing database queries", "action": "use parameterized queries", "confidence": 0.88, "domain": "security", "source": "phase-3", "evidence": ["db"]}
    ]
  },
  "errors": {"records": []},
  "events": [],
  "session_id": "test-session"
}
COLSTATE

    # Registry entry for repo A with domain tags
    mkdir -p "$shared_home/.aether"
    cat > "$shared_home/.aether/registry.json" << REGISTRY
{
  "repos": [
    {
      "path": "$repo_a",
      "version": "2.0.0",
      "domain_tags": ["web", "node"],
      "active_colony": true
    }
  ]
}
REGISTRY

    # Initialize hive in shared HOME
    HOME="$shared_home" AETHER_ROOT="$repo_a" DATA_DIR="$repo_a/.aether/data" \
        bash "$AETHER_UTILS" hive-init > /dev/null 2>&1

    # Simulate seal Step 3.7: promote instincts with registry domain tags
    local repo_domain_tags
    repo_domain_tags=$(jq -r --arg repo "$repo_a" \
      '[.repos[] | select(.path == $repo) | .domain_tags // []] | .[0] // [] | join(",")' \
      "$shared_home/.aether/registry.json" 2>/dev/null || echo "")

    local high_conf_instincts
    high_conf_instincts=$(jq -r '.memory.instincts[] | select(.confidence >= 0.8) | @base64' \
      "$repo_a/.aether/data/COLONY_STATE.json" 2>/dev/null || echo "")

    for encoded in $high_conf_instincts; do
        [[ -z "$encoded" ]] && continue
        local trigger action confidence
        trigger=$(echo "$encoded" | base64 -d | jq -r '.trigger // empty')
        action=$(echo "$encoded" | base64 -d | jq -r '.action // empty')
        confidence=$(echo "$encoded" | base64 -d | jq -r '.confidence // 0.7')
        trigger_clean=$(echo "$trigger" | sed 's/^[Ww]hen //')
        local promote_text="When ${trigger_clean}: ${action}"
        HOME="$shared_home" AETHER_ROOT="$repo_a" DATA_DIR="$repo_a/.aether/data" \
            bash "$AETHER_UTILS" hive-promote \
            --text "$promote_text" --source-repo "$repo_a" \
            --confidence "$confidence" --domain "$repo_domain_tags" > /dev/null 2>&1
    done

    # Verify entries are in hive
    local hive_check
    hive_check=$(HOME="$shared_home" AETHER_ROOT="$repo_a" DATA_DIR="$repo_a/.aether/data" \
        bash "$AETHER_UTILS" hive-read --domain "web,node" --format json 2>&1)
    local hive_count
    hive_count=$(echo "$hive_check" | jq -r '.result.total_matched // 0' 2>/dev/null)

    if [[ "$hive_count" -lt 1 ]]; then
        test_fail "hive should have entries after seal promotion" "got: $hive_count"
        rm -rf "$shared_home"
        return 1
    fi

    # --- Repo B: init seeds from hive ---
    local repo_b="$shared_home/repo-b"
    mkdir -p "$repo_b/.aether/data"
    mkdir -p "$repo_b/.aether/templates"
    cp "$REPO_ROOT/.aether/templates/QUEEN.md.template" "$repo_b/.aether/templates/"

    # Initialize QUEEN.md for repo B
    HOME="$shared_home" AETHER_ROOT="$repo_b" DATA_DIR="$repo_b/.aether/data" \
        bash "$AETHER_UTILS" queen-init > /dev/null 2>&1

    # Seed from hive with matching domain
    local seed_result
    seed_result=$(HOME="$shared_home" AETHER_ROOT="$repo_b" DATA_DIR="$repo_b/.aether/data" \
        bash "$AETHER_UTILS" queen-seed-from-hive --domain "web,node" --limit 5 2>&1)

    local seeded_count
    seeded_count=$(echo "$seed_result" | jq -r '.result.seeded // 0' 2>/dev/null)

    if [[ "$seeded_count" -lt 1 ]]; then
        test_fail "repo B should seed entries from hive" "got: $seeded_count"
        rm -rf "$shared_home"
        return 1
    fi

    # Verify Repo B's QUEEN.md has [hive] entries
    local queen_b="$repo_b/.aether/QUEEN.md"

    if ! grep -q '\[hive\]' "$queen_b"; then
        test_fail "repo B QUEEN.md should have [hive] entries" "none found"
        rm -rf "$shared_home"
        return 1
    fi

    # Verify the content originated from Repo A's instincts (check for key phrases)
    local has_api_validation=false
    local has_parameterized=false

    if grep -q "input validation" "$queen_b" || grep -q "creating API" "$queen_b"; then
        has_api_validation=true
    fi
    if grep -q "parameterized" "$queen_b" || grep -q "database queries" "$queen_b"; then
        has_parameterized=true
    fi

    if [[ "$has_api_validation" != "true" ]] && [[ "$has_parameterized" != "true" ]]; then
        test_fail "repo B QUEEN.md should contain wisdom from repo A's instincts" "no matching content found"
        rm -rf "$shared_home"
        return 1
    fi

    # Verify placeholder is removed
    if grep -q "No codebase patterns recorded yet" "$queen_b"; then
        test_fail "placeholder should be removed after seeding" "still present"
        rm -rf "$shared_home"
        return 1
    fi

    rm -rf "$shared_home"
    return 0
}

# ============================================================================
# Run all tests
# ============================================================================

run_test test_seal_promotes_with_registry_domain_tags "cross-colony: seal promotes with registry domain tags"
run_test test_domain_detect_from_file_presence "cross-colony: domain detection from file presence"
run_test test_queen_seed_from_hive_writes_entries "cross-colony: queen-seed-from-hive writes [hive] entries"
run_test test_queen_seed_deduplicates_existing_entries "cross-colony: queen-seed deduplicates existing entries"
run_test test_queen_seed_domain_scoping "cross-colony: queen-seed respects domain scoping"
run_test test_end_to_end_seal_to_init_flow "cross-colony: end-to-end seal -> hive -> init flow"

test_summary
