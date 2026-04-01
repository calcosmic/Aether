#!/usr/bin/env bash
# Instinct Store Module Tests
# Tests instinct-store.sh functions via aether-utils.sh subcommands:
#   instinct-store, instinct-read-trusted, instinct-decay-all, instinct-archive

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

if [[ ! -f "$AETHER_UTILS" ]]; then
    log_error "aether-utils.sh not found at: $AETHER_UTILS"
    exit 1
fi

# ============================================================================
# Helper: isolated env with aether-utils.sh + all utils
# ============================================================================
setup_instinct_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    mkdir -p "$tmpdir/.aether/data"

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

    # Write minimal COLONY_STATE.json so colony resolution works
    cat > "$tmpdir/.aether/data/COLONY_STATE.json" <<'JSON'
{
  "version": "3.0",
  "goal": "Test instinct store",
  "state": "active",
  "current_phase": 1,
  "session_id": "test-session",
  "initialized_at": "2026-01-01T00:00:00Z",
  "plan": {"phases": []},
  "memory": {"phase_learnings": [], "decisions": [], "instincts": []},
  "errors": {"records": [], "flagged_patterns": []},
  "events": [],
  "signals": [],
  "graveyards": []
}
JSON

    echo "$tmpdir"
}

run_cmd() {
    local tmpdir="$1"
    shift
    AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$tmpdir/.aether/aether-utils.sh" "$@" 2>/dev/null || true
}

run_cmd_with_stderr() {
    local tmpdir="$1"
    shift
    AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$tmpdir/.aether/aether-utils.sh" "$@" 2>&1 || true
}

# ============================================================================
# TEST 1: instinct-store creates the file and stores the instinct
# ============================================================================
test_store_creates_file_and_instinct() {
    local tmpdir
    tmpdir=$(setup_instinct_env)

    local result
    result=$(run_cmd "$tmpdir" instinct-store \
        --trigger "when tests are slow" \
        --action "run with --parallel flag" \
        --domain "testing" \
        --confidence 0.75 \
        --source "phase-1" \
        --evidence "test run showed 30s improvement")

    local instincts_file="$tmpdir/.aether/data/instincts.json"

    # File must exist
    [[ -f "$instincts_file" ]] || { rm -rf "$tmpdir"; return 1; }

    # Result must be ok:true
    assert_ok_true "$result" || { rm -rf "$tmpdir"; return 1; }

    # instincts.json must have 1 entry
    local count
    count=$(jq '.instincts | length' "$instincts_file")
    [[ "$count" -eq 1 ]] || { rm -rf "$tmpdir"; return 1; }

    rm -rf "$tmpdir"
}

# ============================================================================
# TEST 2: instinct-store deduplicates on matching trigger prefix (first 50 chars)
# ============================================================================
test_store_dedup_on_trigger() {
    local tmpdir
    tmpdir=$(setup_instinct_env)

    # Store same trigger twice
    run_cmd "$tmpdir" instinct-store \
        --trigger "when tests are slow run with parallel flag to speed things" \
        --action "use --parallel" \
        --domain "testing" \
        --confidence 0.6 \
        --source "phase-1" \
        --evidence "observed once" > /dev/null

    run_cmd "$tmpdir" instinct-store \
        --trigger "when tests are slow run with parallel flag to speed things" \
        --action "use --parallel" \
        --domain "testing" \
        --confidence 0.8 \
        --source "phase-2" \
        --evidence "observed again" > /dev/null

    local instincts_file="$tmpdir/.aether/data/instincts.json"
    local count
    count=$(jq '.instincts | length' "$instincts_file")

    # Should still be 1 — deduplication merged into the existing entry
    [[ "$count" -eq 1 ]] || { rm -rf "$tmpdir"; return 1; }

    rm -rf "$tmpdir"
}

# ============================================================================
# TEST 3: instinct-read-trusted filters by minimum trust score
# ============================================================================
test_read_trusted_filters_by_min_score() {
    local tmpdir
    tmpdir=$(setup_instinct_env)

    # Store a high-trust instinct
    run_cmd "$tmpdir" instinct-store \
        --trigger "when deploying to production" \
        --action "run smoke tests first" \
        --domain "workflow" \
        --confidence 0.9 \
        --source "phase-3" \
        --evidence "multi_phase" > /dev/null

    # Store a low-trust instinct
    run_cmd "$tmpdir" instinct-store \
        --trigger "when reviewing PRs" \
        --action "check test coverage" \
        --domain "workflow" \
        --confidence 0.3 \
        --source "phase-1" \
        --evidence "anecdotal" > /dev/null

    # Read with min-score 0.7 — should only return high-trust
    local result
    result=$(run_cmd "$tmpdir" instinct-read-trusted --min-score 0.7)

    assert_ok_true "$result" || { rm -rf "$tmpdir"; return 1; }

    local count
    count=$(echo "$result" | jq '.result.instincts | length')
    [[ "$count" -eq 1 ]] || { rm -rf "$tmpdir"; return 1; }

    rm -rf "$tmpdir"
}

# ============================================================================
# TEST 4: instinct-read-trusted filters by domain
# ============================================================================
test_read_trusted_filters_by_domain() {
    local tmpdir
    tmpdir=$(setup_instinct_env)

    # Store instincts in different domains
    run_cmd "$tmpdir" instinct-store \
        --trigger "when writing tests use descriptive names" \
        --action "use describe/it blocks" \
        --domain "testing" \
        --confidence 0.8 \
        --source "phase-1" \
        --evidence "single_phase" > /dev/null

    run_cmd "$tmpdir" instinct-store \
        --trigger "when designing API endpoints" \
        --action "follow REST conventions" \
        --domain "architecture" \
        --confidence 0.8 \
        --source "phase-2" \
        --evidence "single_phase" > /dev/null

    # Read domain=testing only
    local result
    result=$(run_cmd "$tmpdir" instinct-read-trusted --domain "testing")

    assert_ok_true "$result" || { rm -rf "$tmpdir"; return 1; }

    local count
    count=$(echo "$result" | jq '.result.instincts | length')
    [[ "$count" -eq 1 ]] || { rm -rf "$tmpdir"; return 1; }

    local domain
    domain=$(echo "$result" | jq -r '.result.instincts[0].domain')
    [[ "$domain" == "testing" ]] || { rm -rf "$tmpdir"; return 1; }

    rm -rf "$tmpdir"
}

# ============================================================================
# TEST 5: instinct-decay-all applies trust decay to all instincts
# ============================================================================
test_decay_all_applies_decay() {
    local tmpdir
    tmpdir=$(setup_instinct_env)

    run_cmd "$tmpdir" instinct-store \
        --trigger "when debugging start with logs" \
        --action "check error logs first" \
        --domain "workflow" \
        --confidence 0.8 \
        --source "phase-1" \
        --evidence "single_phase" > /dev/null

    # Capture initial trust score
    local initial_score
    initial_score=$(jq -r '.instincts[0].trust_score' "$tmpdir/.aether/data/instincts.json")

    # Apply decay with 120 days (2 half-lives worth, so score should drop)
    local result
    result=$(run_cmd "$tmpdir" instinct-decay-all --days 120)

    assert_ok_true "$result" || { rm -rf "$tmpdir"; return 1; }

    # Score should have decreased
    local decayed_score
    decayed_score=$(jq -r '.instincts[0].trust_score' "$tmpdir/.aether/data/instincts.json")

    local decreased
    decreased=$(awk "BEGIN{print ($decayed_score < $initial_score)}" 2>/dev/null || echo "0")
    [[ "$decreased" == "1" ]] || { rm -rf "$tmpdir"; return 1; }

    rm -rf "$tmpdir"
}

# ============================================================================
# TEST 6: instinct-archive soft-deletes (archived: true, not removed)
# ============================================================================
test_archive_soft_deletes() {
    local tmpdir
    tmpdir=$(setup_instinct_env)

    run_cmd "$tmpdir" instinct-store \
        --trigger "when code review takes too long" \
        --action "break PRs into smaller chunks" \
        --domain "workflow" \
        --confidence 0.7 \
        --source "phase-1" \
        --evidence "single_phase" > /dev/null

    local id
    id=$(jq -r '.instincts[0].id' "$tmpdir/.aether/data/instincts.json")

    local result
    result=$(run_cmd "$tmpdir" instinct-archive --id "$id")

    assert_ok_true "$result" || { rm -rf "$tmpdir"; return 1; }

    # Entry still exists in file
    local count
    count=$(jq '.instincts | length' "$tmpdir/.aether/data/instincts.json")
    [[ "$count" -eq 1 ]] || { rm -rf "$tmpdir"; return 1; }

    # But archived=true
    local archived
    archived=$(jq -r '.instincts[0].archived' "$tmpdir/.aether/data/instincts.json")
    [[ "$archived" == "true" ]] || { rm -rf "$tmpdir"; return 1; }

    # instinct-read-trusted should exclude it
    local read_result
    read_result=$(run_cmd "$tmpdir" instinct-read-trusted)
    local read_count
    read_count=$(echo "$read_result" | jq '.result.instincts | length')
    [[ "$read_count" -eq 0 ]] || { rm -rf "$tmpdir"; return 1; }

    rm -rf "$tmpdir"
}

# ============================================================================
# TEST 7: Missing required args return error
# ============================================================================
test_missing_args_error_handling() {
    local tmpdir
    tmpdir=$(setup_instinct_env)

    # Missing --trigger
    local result
    result=$(run_cmd_with_stderr "$tmpdir" instinct-store \
        --action "do something" \
        --domain "testing" \
        --confidence 0.7 \
        --source "phase-1" \
        --evidence "anecdotal")

    # Should fail with ok:false or error message
    local ok
    ok=$(echo "$result" | jq -r '.ok // "error"' 2>/dev/null || echo "error")
    [[ "$ok" == "false" || "$ok" == "error" ]] || { rm -rf "$tmpdir"; return 1; }

    # Missing --id for archive
    local archive_result
    archive_result=$(run_cmd_with_stderr "$tmpdir" instinct-archive)
    ok=$(echo "$archive_result" | jq -r '.ok // "error"' 2>/dev/null || echo "error")
    [[ "$ok" == "false" || "$ok" == "error" ]] || { rm -rf "$tmpdir"; return 1; }

    rm -rf "$tmpdir"
}

# ============================================================================
# TEST 8: trust_score appears in the stored instinct
# ============================================================================
test_trust_score_present_in_stored_instinct() {
    local tmpdir
    tmpdir=$(setup_instinct_env)

    run_cmd "$tmpdir" instinct-store \
        --trigger "when onboarding new developers" \
        --action "provide runbook with setup steps" \
        --domain "workflow" \
        --confidence 0.75 \
        --source "phase-2" \
        --evidence "single_phase" > /dev/null

    local instincts_file="$tmpdir/.aether/data/instincts.json"

    # trust_score must exist and be a number > 0
    local trust_score
    trust_score=$(jq -r '.instincts[0].trust_score' "$instincts_file")

    [[ "$trust_score" != "null" ]] || { rm -rf "$tmpdir"; return 1; }

    local valid
    valid=$(awk "BEGIN{print ($trust_score > 0)}" 2>/dev/null || echo "0")
    [[ "$valid" == "1" ]] || { rm -rf "$tmpdir"; return 1; }

    # trust_tier must also be present
    local trust_tier
    trust_tier=$(jq -r '.instincts[0].trust_tier' "$instincts_file")
    [[ "$trust_tier" != "null" && -n "$trust_tier" ]] || { rm -rf "$tmpdir"; return 1; }

    rm -rf "$tmpdir"
}

# ============================================================================
# TEST 9: Full schema verification on stored instinct
# ============================================================================
test_full_schema_present() {
    local tmpdir
    tmpdir=$(setup_instinct_env)

    run_cmd "$tmpdir" instinct-store \
        --trigger "when schema verification needed" \
        --action "check all required fields" \
        --domain "testing" \
        --confidence 0.75 \
        --source "phase-1" \
        --evidence "single_phase" > /dev/null

    local instincts_file="$tmpdir/.aether/data/instincts.json"
    local entry
    entry=$(jq '.instincts[0]' "$instincts_file")

    # Top-level required fields
    for field in id trigger action domain trust_score trust_tier confidence archived; do
        local val
        val=$(echo "$entry" | jq -r ".$field")
        [[ "$val" != "null" && -n "$val" ]] || { rm -rf "$tmpdir"; return 1; }
    done

    # Provenance sub-object fields (last_applied is null for fresh instincts)
    for pfield in source source_type evidence created_at application_count; do
        local pval
        pval=$(echo "$entry" | jq -r ".provenance.$pfield")
        [[ "$pval" != "null" ]] || { rm -rf "$tmpdir"; return 1; }
    done

    # last_applied must exist as a field (null is valid for fresh instincts)
    local has_last_applied
    has_last_applied=$(echo "$entry" | jq '.provenance | has("last_applied")')
    [[ "$has_last_applied" == "true" ]] || { rm -rf "$tmpdir"; return 1; }

    # application_history must be an array
    local ah_type
    ah_type=$(echo "$entry" | jq -r '.application_history | type')
    [[ "$ah_type" == "array" ]] || { rm -rf "$tmpdir"; return 1; }

    # related_instincts must be an array
    local ri_type
    ri_type=$(echo "$entry" | jq -r '.related_instincts | type')
    [[ "$ri_type" == "array" ]] || { rm -rf "$tmpdir"; return 1; }

    rm -rf "$tmpdir"
}

# ============================================================================
# TEST 10: Reinforcement updates provenance fields
# ============================================================================
test_reinforcement_updates_provenance() {
    local tmpdir
    tmpdir=$(setup_instinct_env)

    run_cmd "$tmpdir" instinct-store \
        --trigger "reinforcement trigger test case" \
        --action "original action" \
        --domain "testing" \
        --confidence 0.6 \
        --source "phase-1" \
        --evidence "first observation" > /dev/null

    local instincts_file="$tmpdir/.aether/data/instincts.json"

    # Store initial provenance values
    local initial_app_count
    initial_app_count=$(jq -r '.instincts[0].provenance.application_count' "$instincts_file")
    local initial_last_applied
    initial_last_applied=$(jq -r '.instincts[0].provenance.last_applied' "$instincts_file")

    # Reinforce with higher confidence
    run_cmd "$tmpdir" instinct-store \
        --trigger "reinforcement trigger test case" \
        --action "updated action" \
        --domain "testing" \
        --confidence 0.9 \
        --source "phase-2" \
        --evidence "second observation" > /dev/null

    local new_app_count
    new_app_count=$(jq -r '.instincts[0].provenance.application_count' "$instincts_file")
    local new_last_applied
    new_last_applied=$(jq -r '.instincts[0].provenance.last_applied' "$instincts_file")
    local new_confidence
    new_confidence=$(jq -r '.instincts[0].confidence' "$instincts_file")

    # application_count should have incremented
    [[ "$new_app_count" -gt "$initial_app_count" ]] || { rm -rf "$tmpdir"; return 1; }

    # last_applied should now be non-null
    [[ "$new_last_applied" != "null" && -n "$new_last_applied" ]] || { rm -rf "$tmpdir"; return 1; }

    # confidence should be max of original and new (0.9)
    local is_max
    is_max=$(awk "BEGIN{print ($new_confidence == 0.9)}" 2>/dev/null || echo "0")
    [[ "$is_max" == "1" ]] || { rm -rf "$tmpdir"; return 1; }

    rm -rf "$tmpdir"
}

# ============================================================================
# TEST 11: 50-instinct cap archives lowest-trust entry
# ============================================================================
test_fifty_instinct_cap() {
    local tmpdir
    tmpdir=$(setup_instinct_env)

    # Store 51 instincts with varying trust levels
    for i in $(seq 1 51); do
        run_cmd "$tmpdir" instinct-store \
            --trigger "cap test trigger number $i unique suffix here" \
            --action "cap test action $i" \
            --domain "testing" \
            --confidence "0.$(printf '%02d' "$i")" \
            --source "phase-1" \
            --evidence "cap test evidence $i" > /dev/null
    done

    local instincts_file="$tmpdir/.aether/data/instincts.json"

    # Active (non-archived) count should be at most 50
    local active_count
    active_count=$(jq '[.instincts[] | select(.archived == false)] | length' "$instincts_file")
    [[ "$active_count" -le 50 ]] || { rm -rf "$tmpdir"; return 1; }

    # Total entries should be 51 (archived ones still in file)
    local total_count
    total_count=$(jq '.instincts | length' "$instincts_file")
    [[ "$total_count" -eq 51 ]] || { rm -rf "$tmpdir"; return 1; }

    # At least 1 entry should be archived
    local archived_count
    archived_count=$(jq '[.instincts[] | select(.archived == true)] | length' "$instincts_file")
    [[ "$archived_count" -ge 1 ]] || { rm -rf "$tmpdir"; return 1; }

    rm -rf "$tmpdir"
}

# ============================================================================
# TEST 12: decay-all --dry-run does not write changes
# ============================================================================
test_decay_dry_run_no_write() {
    local tmpdir
    tmpdir=$(setup_instinct_env)

    run_cmd "$tmpdir" instinct-store \
        --trigger "dry run test trigger" \
        --action "dry run action" \
        --domain "testing" \
        --confidence 0.9 \
        --source "phase-1" \
        --evidence "single_phase" > /dev/null

    local instincts_file="$tmpdir/.aether/data/instincts.json"
    local initial_score
    initial_score=$(jq -r '.instincts[0].trust_score' "$instincts_file")

    local result
    result=$(run_cmd "$tmpdir" instinct-decay-all --days 120 --dry-run)

    assert_ok_true "$result" || { rm -rf "$tmpdir"; return 1; }

    # Verify dry_run flag in result
    local is_dry
    is_dry=$(echo "$result" | jq -r '.result.dry_run')
    [[ "$is_dry" == "true" ]] || { rm -rf "$tmpdir"; return 1; }

    # Score should NOT have changed (dry-run)
    local score_after
    score_after=$(jq -r '.instincts[0].trust_score' "$instincts_file")
    [[ "$score_after" == "$initial_score" ]] || { rm -rf "$tmpdir"; return 1; }

    rm -rf "$tmpdir"
}

# ============================================================================
# TEST 13: decay-all archives instincts below 0.25 threshold
# ============================================================================
test_decay_archives_low_trust() {
    local tmpdir
    tmpdir=$(setup_instinct_env)

    # Store a very low confidence instinct (will get low trust score)
    run_cmd "$tmpdir" instinct-store \
        --trigger "low trust decay test trigger" \
        --action "low trust action" \
        --domain "testing" \
        --confidence 0.1 \
        --source "phase-1" \
        --evidence "anecdotal" > /dev/null

    local instincts_file="$tmpdir/.aether/data/instincts.json"
    local id
    id=$(jq -r '.instincts[0].id' "$instincts_file")

    # Apply heavy decay (300 days = ~5 half-lives)
    local result
    result=$(run_cmd "$tmpdir" instinct-decay-all --days 300)

    assert_ok_true "$result" || { rm -rf "$tmpdir"; return 1; }

    # The low-trust instinct should have been archived
    local archived
    archived=$(jq -r --arg id "$id" '.instincts[] | select(.id == $id) | .archived' "$instincts_file")
    [[ "$archived" == "true" ]] || { rm -rf "$tmpdir"; return 1; }

    # archived count in result should be >= 1
    local result_archived
    result_archived=$(echo "$result" | jq -r '.result.archived')
    [[ "$result_archived" -ge 1 ]] || { rm -rf "$tmpdir"; return 1; }

    rm -rf "$tmpdir"
}

# ============================================================================
# TEST 14: instinct-read-trusted --limit works correctly
# ============================================================================
test_read_trusted_limit() {
    local tmpdir
    tmpdir=$(setup_instinct_env)

    # Store 5 instincts
    for i in $(seq 1 5); do
        run_cmd "$tmpdir" instinct-store \
            --trigger "limit test trigger $i unique suffix" \
            --action "limit test action $i" \
            --domain "testing" \
            --confidence "0.8" \
            --source "phase-1" \
            --evidence "single_phase" > /dev/null
    done

    # Request limit=2
    local result
    result=$(run_cmd "$tmpdir" instinct-read-trusted --min-score 0.0 --limit 2)

    assert_ok_true "$result" || { rm -rf "$tmpdir"; return 1; }

    local count
    count=$(echo "$result" | jq '.result.instincts | length')
    [[ "$count" -eq 2 ]] || { rm -rf "$tmpdir"; return 1; }

    rm -rf "$tmpdir"
}

# ============================================================================
# TEST 15: instinct-read-trusted returns empty for missing file
# ============================================================================
test_read_trusted_missing_file() {
    local tmpdir
    tmpdir=$(setup_instinct_env)

    # Do not store anything -- instincts.json does not exist
    local result
    result=$(run_cmd "$tmpdir" instinct-read-trusted)

    assert_ok_true "$result" || { rm -rf "$tmpdir"; return 1; }

    local count
    count=$(echo "$result" | jq '.result.count')
    [[ "$count" -eq 0 ]] || { rm -rf "$tmpdir"; return 1; }

    rm -rf "$tmpdir"
}

# ============================================================================
# TEST 16: decay-all handles empty/missing file gracefully
# ============================================================================
test_decay_missing_file() {
    local tmpdir
    tmpdir=$(setup_instinct_env)

    # Do not create instincts.json
    local result
    result=$(run_cmd "$tmpdir" instinct-decay-all --days 30)

    assert_ok_true "$result" || { rm -rf "$tmpdir"; return 1; }

    local processed
    processed=$(echo "$result" | jq '.result.processed')
    [[ "$processed" -eq 0 ]] || { rm -rf "$tmpdir"; return 1; }

    rm -rf "$tmpdir"
}

# ============================================================================
# Main: run all tests
# ============================================================================

log_info "Running instinct-store module tests..."
log_info ""

run_test "test_store_creates_file_and_instinct"     "instinct-store: creates file and stores instinct"
run_test "test_store_dedup_on_trigger"              "instinct-store: dedup — matching trigger prefix merges entry"
run_test "test_read_trusted_filters_by_min_score"   "instinct-read-trusted: filters by min trust score"
run_test "test_read_trusted_filters_by_domain"      "instinct-read-trusted: filters by domain"
run_test "test_decay_all_applies_decay"             "instinct-decay-all: applies trust decay to all instincts"
run_test "test_archive_soft_deletes"                "instinct-archive: soft-deletes (archived:true, excluded from reads)"
run_test "test_missing_args_error_handling"         "instinct-store/archive: missing required args return error"
run_test "test_trust_score_present_in_stored_instinct" "instinct-store: trust_score and trust_tier present in stored instinct"
run_test "test_full_schema_present"                 "instinct-store: full schema with provenance, application_history, related_instincts"
run_test "test_reinforcement_updates_provenance"    "instinct-store: reinforcement updates provenance.last_applied and application_count"
run_test "test_fifty_instinct_cap"                  "instinct-store: 50-instinct cap archives lowest-trust entry"
run_test "test_decay_dry_run_no_write"              "instinct-decay-all: --dry-run does not write changes"
run_test "test_decay_archives_low_trust"            "instinct-decay-all: archives instincts below 0.25 threshold"
run_test "test_read_trusted_limit"                  "instinct-read-trusted: --limit works correctly"
run_test "test_read_trusted_missing_file"           "instinct-read-trusted: returns empty for missing file"
run_test "test_decay_missing_file"                  "instinct-decay-all: handles missing file gracefully"

test_summary
