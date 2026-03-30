#!/usr/bin/env bash
# Tests for midden cross-branch collection subcommands
# midden-collect, midden-handle-revert, midden-cross-pr-analysis, midden-prune

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
  "goal": "test midden-collection",
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

# Helper: create a fake worktree with midden entries
setup_branch_worktree() {
    local main_tmpdir="$1"
    local branch="$2"
    shift 2

    local branch_dir="$main_tmpdir/.aether/worktrees/$branch/.aether/data/midden"
    mkdir -p "$branch_dir"

    # Create branch midden with entries passed as arguments
    # Each arg is "category:message:source"
    local entries="[]"
    for entry_spec in "$@"; do
        local category="${entry_spec%%:*}"
        local rest="${entry_spec#*:}"
        local message="${rest%%:*}"
        local source="${rest##*:}"
        local entry_id="midden_$(date +%s)_$$$(printf '%s' "$category$message" | cksum | cut -d' ' -f1)"
        local ts
        ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
        entries=$(echo "$entries" | jq --arg id "$entry_id" --arg ts "$ts" \
            --arg cat "$category" --arg src "$source" --arg msg "$message" \
            '. + [{id: $id, timestamp: $ts, category: $cat, source: $src, message: $msg, reviewed: false}]')
    done

    printf '%s\n' "$(jq -n --argjson entries "$entries" '{version: "1.0.0", entries: $entries}')" > "$branch_dir/midden.json"
    echo "$branch_dir"
}

# ============================================================================
# midden-collect tests
# ============================================================================

# Test 1: Basic collection — entries appear in main's midden with enrichment fields
test_collect_basic() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    setup_branch_worktree "$tmpdir" "feature/test-branch" "resilience:Test flaked 3 times:gatekeeper" "security:CVE found:auditor" >/dev/null

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-collect --branch "feature/test-branch" --merge-sha "abc123def") || exit_code=$?

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

    local status
    status=$(echo "$result" | jq -r '.result.status')
    if [[ "$status" != "collected" ]]; then
        test_fail "status=collected" "status=$status"
        rm -rf "$tmpdir"
        return 1
    fi

    local collected
    collected=$(echo "$result" | jq -r '.result.entries_collected')
    if [[ "$collected" != "2" ]]; then
        test_fail "entries_collected=2" "entries_collected=$collected"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify enrichment fields in main's midden
    local has_collected_from
    has_collected_from=$(jq '[.entries[] | select(.collected_from == "feature/test-branch")] | length' "$tmpdir/.aether/data/midden/midden.json")
    if [[ "$has_collected_from" != "2" ]]; then
        test_fail "2 entries with collected_from" "found $has_collected_from"
        rm -rf "$tmpdir"
        return 1
    fi

    local has_merge_commit
    has_merge_commit=$(jq '[.entries[] | select(.merge_commit == "abc123def")] | length' "$tmpdir/.aether/data/midden/midden.json")
    if [[ "$has_merge_commit" != "2" ]]; then
        test_fail "2 entries with merge_commit" "found $has_merge_commit"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify collected-merges.json was written
    if [[ ! -f "$tmpdir/.aether/data/midden/collected-merges.json" ]]; then
        test_fail "collected-merges.json exists" "file not found"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 2: Idempotent — second collect returns already_collected with 0 entries
test_collect_idempotent() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    setup_branch_worktree "$tmpdir" "feature/idem-branch" "general:test entry:test" >/dev/null

    # First collect
    run_cmd "$tmpdir" midden-collect --branch "feature/idem-branch" --merge-sha "idem123" >/dev/null

    # Second collect
    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-collect --branch "feature/idem-branch" --merge-sha "idem123") || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0 on second collect" "exit code $exit_code"
        rm -rf "$tmpdir"
        return 1
    fi

    local status
    status=$(echo "$result" | jq -r '.result.status')
    if [[ "$status" != "already_collected" ]]; then
        test_fail "status=already_collected" "status=$status"
        rm -rf "$tmpdir"
        return 1
    fi

    local collected
    collected=$(echo "$result" | jq -r '.result.entries_collected')
    if [[ "$collected" != "0" ]]; then
        test_fail "entries_collected=0" "entries_collected=$collected"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify main midden still has only 1 entry (not duplicated)
    local total
    total=$(jq '.entries | length' "$tmpdir/.aether/data/midden/midden.json")
    if [[ "$total" != "1" ]]; then
        test_fail "1 total entry after idempotent collect" "total=$total"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 3: Collect from worktree with empty midden
test_collect_empty_branch() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    setup_branch_worktree "$tmpdir" "feature/empty-branch" >/dev/null

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-collect --branch "feature/empty-branch" --merge-sha "empty123") || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code"
        rm -rf "$tmpdir"
        return 1
    fi

    local status
    status=$(echo "$result" | jq -r '.result.status')
    if [[ "$status" != "empty_branch_midden" ]]; then
        test_fail "status=empty_branch_midden" "status=$status"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 4: Collect from non-existent branch returns worktree_not_found
test_collect_worktree_not_found() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-collect --branch "feature/nonexistent" --merge-sha "nope123") || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code"
        rm -rf "$tmpdir"
        return 1
    fi

    local status
    status=$(echo "$result" | jq -r '.result.status')
    if [[ "$status" != "worktree_not_found" ]]; then
        test_fail "status=worktree_not_found" "status=$status"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 5: Layer 2 dedup — manually add entry with same ID, verify it's skipped
test_collect_layer2_dedup() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    # Create branch with one entry using a known ID
    local branch_dir="$tmpdir/.aether/worktrees/feature/dedup-branch/.aether/data/midden"
    mkdir -p "$branch_dir"
    local known_id="midden_12345_999"
    printf '%s\n' "$(jq -n --arg id "$known_id" '{version: "1.0.0", entries: [{id: $id, timestamp: "2026-03-30T12:00:00Z", category: "test", source: "test", message: "dup test", reviewed: false}]}')" > "$branch_dir/midden.json"

    # Manually add the same ID to main's midden
    local main_midden="$tmpdir/.aether/data/midden/midden.json"
    local main_updated
    main_updated=$(jq --arg id "$known_id" '.entries += [{id: $id, timestamp: "2026-03-30T11:00:00Z", category: "existing", source: "existing", message: "already here", reviewed: false}]' "$main_midden")
    printf '%s\n' "$main_updated" > "$main_midden"

    # Collect with a DIFFERENT merge-sha (so Layer 1 passes)
    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-collect --branch "feature/dedup-branch" --merge-sha "unique_sha_layer2") || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local collected
    collected=$(echo "$result" | jq -r '.result.entries_collected')
    if [[ "$collected" != "0" ]]; then
        test_fail "entries_collected=0 (Layer 2 dedup)" "entries_collected=$collected"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify skipped count
    local skipped
    skipped=$(echo "$result" | jq -r '.result.entries_skipped_dup')
    if [[ "$skipped" != "1" ]]; then
        test_fail "entries_skipped_dup=1" "entries_skipped_dup=$skipped"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# midden-handle-revert tests
# ============================================================================

# Test 6: Basic revert — entries get reverted:<sha> tag
test_revert_basic() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    setup_branch_worktree "$tmpdir" "feature/revert-test" "general:failure one:test" "general:failure two:test" >/dev/null

    # Collect first
    run_cmd "$tmpdir" midden-collect --branch "feature/revert-test" --merge-sha "mergeABC" >/dev/null

    # Revert
    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-handle-revert --revert-commit "revertDEF" --original-merge "mergeABC") || exit_code=$?

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

    local tagged
    tagged=$(echo "$result" | jq -r '.result.entries_tagged')
    if [[ "$tagged" != "2" ]]; then
        test_fail "entries_tagged=2" "entries_tagged=$tagged"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify entries have reverted: tag in midden.json
    local reverted_tag
    reverted_tag=$(jq '[.entries[] | .tags // [] | map(select(startswith("reverted:"))) | length] | add' "$tmpdir/.aether/data/midden/midden.json")
    if [[ "$reverted_tag" != "2" ]]; then
        test_fail "2 reverted tags in midden.json" "found $reverted_tag"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 7: Revert preserves entries (NOT deleted)
test_revert_preserves_entries() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    setup_branch_worktree "$tmpdir" "feature/preserve-test" "general:preserved entry:test" >/dev/null

    run_cmd "$tmpdir" midden-collect --branch "feature/preserve-test" --merge-sha "preserve123" >/dev/null

    local before_count
    before_count=$(jq '.entries | length' "$tmpdir/.aether/data/midden/midden.json")

    run_cmd "$tmpdir" midden-handle-revert --revert-commit "revert456" --original-merge "preserve123" >/dev/null

    local after_count
    after_count=$(jq '.entries | length' "$tmpdir/.aether/data/midden/midden.json")

    if [[ "$after_count" != "$before_count" ]]; then
        test_fail "entry count unchanged after revert" "before=$before_count after=$after_count"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 8: Revert for non-existent merge
test_revert_merge_not_found() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-handle-revert --revert-commit "revertXXX" --original-merge "nonexistent") || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code"
        rm -rf "$tmpdir"
        return 1
    fi

    local status
    status=$(echo "$result" | jq -r '.result.status')
    if [[ "$status" != "merge_not_found" ]]; then
        test_fail "status=merge_not_found" "status=$status"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# midden-cross-pr-analysis tests
# ============================================================================

# Test 9: Detect systemic pattern across 2+ branches in same category
test_cross_pr_detect_systemic() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    # Create entries from 2 branches in same category (3+ entries)
    setup_branch_worktree "$tmpdir" "feature/branch-a" "resilience:flake test a1:test" "resilience:flake test a2:test" >/dev/null
    setup_branch_worktree "$tmpdir" "feature/branch-b" "resilience:flake test b1:test" >/dev/null

    run_cmd "$tmpdir" midden-collect --branch "feature/branch-a" --merge-sha "sysA" >/dev/null
    run_cmd "$tmpdir" midden-collect --branch "feature/branch-b" --merge-sha "sysB" >/dev/null

    # Create a writable pheromones.json for auto-REDIRECT emission
    mkdir -p "$tmpdir/.aether/data"
    printf '%s\n' '{"version":"1.0.0","signals":[]}' > "$tmpdir/.aether/data/pheromones.json"

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-cross-pr-analysis) || exit_code=$?

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

    local classification
    classification=$(echo "$result" | jq -r '.result.categories.resilience.classification')
    if [[ "$classification" != "cross-pr-systemic" ]]; then
        test_fail "classification=cross-pr-systemic" "classification=$classification"
        rm -rf "$tmpdir"
        return 1
    fi

    local redirect_emitted
    redirect_emitted=$(echo "$result" | jq -r '.result.categories.resilience.auto_redirect_emitted')
    if [[ "$redirect_emitted" != "true" ]]; then
        test_fail "auto_redirect_emitted=true" "auto_redirect_emitted=$redirect_emitted"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 10: Cross-PR analysis excludes reverted entries
test_cross_pr_excludes_reverted() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    setup_branch_worktree "$tmpdir" "feature/rev-branch-a" "quality:quality issue a1:test" "quality:quality issue a2:test" >/dev/null
    setup_branch_worktree "$tmpdir" "feature/rev-branch-b" "quality:quality issue b1:test" >/dev/null

    run_cmd "$tmpdir" midden-collect --branch "feature/rev-branch-a" --merge-sha "revA" >/dev/null
    run_cmd "$tmpdir" midden-collect --branch "feature/rev-branch-b" --merge-sha "revB" >/dev/null

    # Revert one merge
    run_cmd "$tmpdir" midden-handle-revert --revert-commit "revRevertB" --original-merge "revB" >/dev/null

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-cross-pr-analysis) || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code"
        rm -rf "$tmpdir"
        return 1
    fi

    # With revB reverted, quality has 2 entries from 1 branch = single-pr
    local classification
    classification=$(echo "$result" | jq -r '.result.categories.quality.classification // "none"')
    if [[ "$classification" == "cross-pr-systemic" ]]; then
        test_fail "classification != cross-pr-systemic after revert" "classification=$classification"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 11: Single PR — entries from 1 branch only
test_cross_pr_single_pr() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    setup_branch_worktree "$tmpdir" "feature/single-branch" "general:single entry 1:test" "general:single entry 2:test" "general:single entry 3:test" >/dev/null

    run_cmd "$tmpdir" midden-collect --branch "feature/single-branch" --merge-sha "single123" >/dev/null

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-cross-pr-analysis) || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code"
        rm -rf "$tmpdir"
        return 1
    fi

    local classification
    classification=$(echo "$result" | jq -r '.result.categories.general.classification // "none"')
    if [[ "$classification" != "single-pr" ]]; then
        test_fail "classification=single-pr" "classification=$classification"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# midden-prune tests
# ============================================================================

# Test 12: Prune stale merges — old entries removed from collected-merges.json
test_prune_stale_merges() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    setup_branch_worktree "$tmpdir" "feature/stale-branch" "general:stale entry:test" >/dev/null

    # Manually create an old merge record (100 days ago)
    local merges_file="$tmpdir/.aether/data/midden/collected-merges.json"
    local old_ts
    old_ts=$(python3 -c "import datetime; print((datetime.datetime.utcnow() - datetime.timedelta(days=100)).strftime('%Y-%m-%dT%H:%M:%SZ'))" 2>/dev/null || echo "2025-01-01T00:00:00Z")

    printf '%s\n' "$(jq -n --arg ts "$old_ts" '{version: "1.0.0", merges: [{merge_commit: "stale123", branch_name: "feature/old", collected_at: $ts, entries_collected: 1, entries_skipped_dup: 0, fingerprint: "old"}]}')" > "$merges_file"

    local before_count
    before_count=$(jq '.merges | length' "$merges_file")

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-prune --stale-merges) || exit_code=$?

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

    local pruned
    pruned=$(echo "$result" | jq -r '.result.pruned_merges')
    if [[ "$pruned" != "1" ]]; then
        test_fail "pruned_merges=1" "pruned_merges=$pruned"
        rm -rf "$tmpdir"
        return 1
    fi

    local after_count
    after_count=$(jq '.merges | length' "$merges_file")
    if [[ "$after_count" != "0" ]]; then
        test_fail "0 merges after prune" "after_count=$after_count"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# Test 13: Prune reverted — old reverted entries get acknowledged
test_prune_reverted() {
    local tmpdir
    tmpdir=$(setup_midden_env)

    # Set up: collect, then revert, then prune
    setup_branch_worktree "$tmpdir" "feature/prune-rev" "general:prunable entry:test" >/dev/null

    run_cmd "$tmpdir" midden-collect --branch "feature/prune-rev" --merge-sha "pruneMerge" >/dev/null

    # Revert with an old timestamp
    run_cmd "$tmpdir" midden-handle-revert --revert-commit "pruneRevert" --original-merge "pruneMerge" >/dev/null

    # Make the revert timestamp old (40 days ago) in collected-merges.json
    local merges_file="$tmpdir/.aether/data/midden/collected-merges.json"
    local old_ts
    old_ts=$(python3 -c "import datetime; print((datetime.datetime.utcnow() - datetime.timedelta(days=40)).strftime('%Y-%m-%dT%H:%M:%SZ'))" 2>/dev/null || echo "2025-01-01T00:00:00Z")
    local updated_merges
    updated_merges=$(jq --arg ts "$old_ts" '(.merges[] | select(.merge_commit == "pruneMerge")) |= (.reverted_at = $ts)' "$merges_file")
    printf '%s\n' "$updated_merges" > "$merges_file"

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-prune --reverted --age 30) || exit_code=$?

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

    # Verify the entry was acknowledged
    local ack
    ack=$(jq '[.entries[] | select(.merge_commit == "pruneMerge")] | first | .acknowledged' "$tmpdir/.aether/data/midden/midden.json")
    if [[ "$ack" != "true" ]]; then
        test_fail "entry acknowledged after prune" "acknowledged=$ack"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Run all tests
# ============================================================================

run_test test_collect_basic            "midden-collect: basic collection with enrichment fields"
run_test test_collect_idempotent       "midden-collect: dual-layer idempotency (already_collected)"
run_test test_collect_empty_branch     "midden-collect: empty branch midden returns entries_collected=0"
run_test test_collect_worktree_not_found "midden-collect: non-existent branch returns worktree_not_found"
run_test test_collect_layer2_dedup     "midden-collect: Layer 2 ID dedup skips existing entries"
run_test test_revert_basic             "midden-handle-revert: tags entries with reverted:<sha>"
run_test test_revert_preserves_entries "midden-handle-revert: entries preserved (not deleted)"
run_test test_revert_merge_not_found   "midden-handle-revert: non-existent merge returns merge_not_found"
run_test test_cross_pr_detect_systemic "midden-cross-pr-analysis: detects cross-pr-systemic pattern"
run_test test_cross_pr_excludes_reverted "midden-cross-pr-analysis: excludes reverted entries"
run_test test_cross_pr_single_pr       "midden-cross-pr-analysis: single PR classified as single-pr"
run_test test_prune_stale_merges       "midden-prune: stale merges removed from collected-merges.json"
run_test test_prune_reverted           "midden-prune: old reverted entries get acknowledged"

test_summary
