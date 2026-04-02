#!/usr/bin/env bash
# Curation Ants Core Tests
# Tests nurse.sh, herald.sh, librarian.sh, and critic.sh via aether-utils.sh subcommands:
#   curation-nurse, curation-herald, curation-librarian, curation-critic

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
AETHER_UTILS_SOURCE="$PROJECT_ROOT/.aether/aether-utils.sh"

source "$SCRIPT_DIR/test-helpers.sh"

require_jq

if [[ ! -f "$AETHER_UTILS_SOURCE" ]]; then
    log_error "aether-utils.sh not found at: $AETHER_UTILS_SOURCE"
    exit 1
fi

# ============================================================================
# Helper: Create isolated test environment
# ============================================================================
setup_curation_env() {
    local tmp_dir
    tmp_dir=$(mktemp -d)
    mkdir -p "$tmp_dir/.aether/data/midden"

    cp "$AETHER_UTILS_SOURCE" "$tmp_dir/.aether/aether-utils.sh"
    chmod +x "$tmp_dir/.aether/aether-utils.sh"

    local utils_source
    utils_source="$(dirname "$AETHER_UTILS_SOURCE")/utils"
    if [[ -d "$utils_source" ]]; then
        cp -r "$utils_source" "$tmp_dir/.aether/"
    fi

    local exchange_source
    exchange_source="$(dirname "$AETHER_UTILS_SOURCE")/exchange"
    if [[ -d "$exchange_source" ]]; then
        cp -r "$exchange_source" "$tmp_dir/.aether/"
    fi

    echo "$tmp_dir"
}

run_cmd() {
    local tmp_dir="$1"
    shift
    AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" \
        bash "$tmp_dir/.aether/aether-utils.sh" "$@" 2>/dev/null || true
}

run_cmd_err() {
    local tmp_dir="$1"
    shift
    AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" \
        bash "$tmp_dir/.aether/aether-utils.sh" "$@" 2>&1 || true
}

# ============================================================================
# Test 1: nurse recalculates trust scores on observations
# ============================================================================
test_nurse_recalculates_trust() {
    local tmp_dir
    tmp_dir=$(setup_curation_env)

    # Seed observations with source_type and evidence_type fields
    cat > "$tmp_dir/.aether/data/learning-observations.json" << 'EOF'
{
  "observations": [
    {
      "content_hash": "sha256:aaa",
      "content": "Always use explicit error handling",
      "wisdom_type": "pattern",
      "source_type": "success_pattern",
      "evidence_type": "multi_phase",
      "observation_count": 2,
      "first_seen": "2026-01-01T00:00:00Z",
      "last_seen": "2026-03-01T00:00:00Z",
      "colonies": ["test-colony"]
    },
    {
      "content_hash": "sha256:bbb",
      "content": "No source_type observation — should be skipped",
      "wisdom_type": "pattern",
      "observation_count": 1,
      "first_seen": "2026-02-01T00:00:00Z",
      "last_seen": "2026-02-01T00:00:00Z",
      "colonies": ["test-colony"]
    }
  ]
}
EOF

    local result
    result=$(run_cmd "$tmp_dir" curation-nurse)

    rm -rf "$tmp_dir"

    assert_ok_true "$result" || return 1

    local obs_updated
    obs_updated=$(echo "$result" | jq -r '.result.observations_updated')
    [[ "$obs_updated" -ge 1 ]] || return 1
}

# ============================================================================
# Test 2: nurse dry-run does not modify observations file
# ============================================================================
test_nurse_dry_run() {
    local tmp_dir
    tmp_dir=$(setup_curation_env)

    cat > "$tmp_dir/.aether/data/learning-observations.json" << 'EOF'
{
  "observations": [
    {
      "content_hash": "sha256:ccc",
      "content": "Test observation",
      "source_type": "observation",
      "evidence_type": "single_phase",
      "last_seen": "2026-03-01T00:00:00Z",
      "observation_count": 1,
      "colonies": []
    }
  ]
}
EOF

    local before_mtime
    before_mtime=$(stat -f "%m" "$tmp_dir/.aether/data/learning-observations.json" 2>/dev/null || stat -c "%Y" "$tmp_dir/.aether/data/learning-observations.json")

    local result
    result=$(run_cmd "$tmp_dir" curation-nurse --dry-run)

    local after_mtime
    after_mtime=$(stat -f "%m" "$tmp_dir/.aether/data/learning-observations.json" 2>/dev/null || stat -c "%Y" "$tmp_dir/.aether/data/learning-observations.json")

    rm -rf "$tmp_dir"

    assert_ok_true "$result" || return 1

    local dry_run_val
    dry_run_val=$(echo "$result" | jq -r '.result.dry_run')
    [[ "$dry_run_val" == "true" ]] || return 1

    [[ "$before_mtime" == "$after_mtime" ]] || return 1
}

# ============================================================================
# Test 3: nurse handles missing observations file gracefully
# ============================================================================
test_nurse_missing_file() {
    local tmp_dir
    tmp_dir=$(setup_curation_env)

    local result
    result=$(run_cmd "$tmp_dir" curation-nurse)

    rm -rf "$tmp_dir"

    assert_ok_true "$result" || return 1

    local obs_updated
    obs_updated=$(echo "$result" | jq -r '.result.observations_updated')
    [[ "$obs_updated" -eq 0 ]] || return 1
}

# ============================================================================
# Test 4: herald promotes qualifying instincts
# ============================================================================
test_herald_promotes_instincts() {
    local tmp_dir
    tmp_dir=$(setup_curation_env)

    # Create instincts file with one qualifying and one below threshold
    cat > "$tmp_dir/.aether/data/instincts.json" << 'EOF'
{
  "version": "1.0",
  "instincts": [
    {
      "id": "inst_001",
      "trigger": "when building complex logic",
      "action": "write tests first",
      "domain": "tdd",
      "trust_score": 0.85,
      "trust_tier": "trusted",
      "confidence": 0.85,
      "archived": false,
      "provenance": {
        "source": "test",
        "source_type": "success_pattern",
        "evidence": "multi_phase",
        "created_at": "2026-01-01T00:00:00Z",
        "last_applied": null,
        "application_count": 0
      },
      "application_history": [],
      "related_instincts": []
    },
    {
      "id": "inst_002",
      "trigger": "when starting a new file",
      "action": "add a header comment",
      "domain": "style",
      "trust_score": 0.50,
      "trust_tier": "provisional",
      "confidence": 0.50,
      "archived": false,
      "provenance": {
        "source": "test",
        "source_type": "heuristic",
        "evidence": "anecdotal",
        "created_at": "2026-01-01T00:00:00Z",
        "last_applied": null,
        "application_count": 0
      },
      "application_history": [],
      "related_instincts": []
    }
  ]
}
EOF

    # Create a minimal QUEEN.md with Instincts section
    mkdir -p "$tmp_dir/.aether"
    cat > "$tmp_dir/.aether/QUEEN.md" << 'EOF'
# QUEEN.md

## User Preferences

## Codebase Patterns

## Build Learnings

## Instincts

*No instincts recorded yet.*

---

## Evolution Log
EOF

    local result
    result=$(run_cmd "$tmp_dir" curation-herald)

    rm -rf "$tmp_dir"

    assert_ok_true "$result" || return 1

    local eligible
    eligible=$(echo "$result" | jq -r '.result.eligible')
    [[ "$eligible" -ge 1 ]] || return 1
}

# ============================================================================
# Test 5: herald dry-run reports without writing
# ============================================================================
test_herald_dry_run() {
    local tmp_dir
    tmp_dir=$(setup_curation_env)

    cat > "$tmp_dir/.aether/data/instincts.json" << 'EOF'
{
  "version": "1.0",
  "instincts": [
    {
      "id": "inst_dry",
      "trigger": "when testing dry run behavior",
      "action": "verify no writes occur",
      "domain": "test",
      "trust_score": 0.90,
      "trust_tier": "canonical",
      "confidence": 0.90,
      "archived": false,
      "provenance": {
        "source": "test",
        "source_type": "test_verified",
        "evidence": "test_verified",
        "created_at": "2026-01-01T00:00:00Z",
        "last_applied": null,
        "application_count": 0
      },
      "application_history": [],
      "related_instincts": []
    }
  ]
}
EOF

    mkdir -p "$tmp_dir/.aether"
    cat > "$tmp_dir/.aether/QUEEN.md" << 'EOF'
# QUEEN.md

## Instincts

*No instincts recorded yet.*

---
EOF

    local result
    result=$(run_cmd "$tmp_dir" curation-herald --dry-run)

    rm -rf "$tmp_dir"

    assert_ok_true "$result" || return 1

    local dry_run_val
    dry_run_val=$(echo "$result" | jq -r '.result.dry_run')
    [[ "$dry_run_val" == "true" ]] || return 1
}

# ============================================================================
# Test 6: herald handles missing instincts file gracefully
# ============================================================================
test_herald_missing_file() {
    local tmp_dir
    tmp_dir=$(setup_curation_env)

    local result
    result=$(run_cmd "$tmp_dir" curation-herald)

    rm -rf "$tmp_dir"

    assert_ok_true "$result" || return 1

    local eligible
    eligible=$(echo "$result" | jq -r '.result.eligible')
    [[ "$eligible" -eq 0 ]] || return 1
}

# ============================================================================
# Test 7: librarian returns inventory stats
# ============================================================================
test_librarian_inventory() {
    local tmp_dir
    tmp_dir=$(setup_curation_env)

    # Seed data files
    cat > "$tmp_dir/.aether/data/learning-observations.json" << 'EOF'
{"observations":[{"content_hash":"sha256:x","content":"obs1"},{"content_hash":"sha256:y","content":"obs2"}]}
EOF

    cat > "$tmp_dir/.aether/data/instincts.json" << 'EOF'
{
  "version": "1.0",
  "instincts": [
    {"id":"i1","archived":false},
    {"id":"i2","archived":false},
    {"id":"i3","archived":true}
  ]
}
EOF

    cat > "$tmp_dir/.aether/data/instinct-graph.json" << 'EOF'
{"version":"1.0","edges":[{"source":"i1","target":"i2","relationship":"reinforces"}]}
EOF

    printf '{"topic":"t1"}\n{"topic":"t2"}\n' > "$tmp_dir/.aether/data/event-bus.jsonl"

    cat > "$tmp_dir/.aether/data/pheromones.json" << 'EOF'
{"signals":[{"id":"s1","active":true},{"id":"s2","active":false}]}
EOF

    cat > "$tmp_dir/.aether/data/midden/midden.json" << 'EOF'
{"entries":[{"id":"m1"},{"id":"m2"}]}
EOF

    local result
    result=$(run_cmd "$tmp_dir" curation-librarian)

    rm -rf "$tmp_dir"

    assert_ok_true "$result" || return 1

    local obs_count
    obs_count=$(echo "$result" | jq -r '.result.observations')
    [[ "$obs_count" -eq 2 ]] || return 1

    local inst_total
    inst_total=$(echo "$result" | jq -r '.result.instincts.total')
    [[ "$inst_total" -eq 3 ]] || return 1

    local inst_active
    inst_active=$(echo "$result" | jq -r '.result.instincts.active')
    [[ "$inst_active" -eq 2 ]] || return 1

    local inst_archived
    inst_archived=$(echo "$result" | jq -r '.result.instincts.archived')
    [[ "$inst_archived" -eq 1 ]] || return 1

    local edges
    edges=$(echo "$result" | jq -r '.result.graph_edges')
    [[ "$edges" -eq 1 ]] || return 1

    local events
    events=$(echo "$result" | jq -r '.result.events')
    [[ "$events" -eq 2 ]] || return 1

    local signals_active
    signals_active=$(echo "$result" | jq -r '.result.signals.active')
    [[ "$signals_active" -eq 1 ]] || return 1

    local midden_count
    midden_count=$(echo "$result" | jq -r '.result.midden')
    [[ "$midden_count" -eq 2 ]] || return 1
}

# ============================================================================
# Test 8: librarian handles all missing files gracefully
# ============================================================================
test_librarian_all_missing() {
    local tmp_dir
    tmp_dir=$(setup_curation_env)

    local result
    result=$(run_cmd "$tmp_dir" curation-librarian)

    rm -rf "$tmp_dir"

    assert_ok_true "$result" || return 1

    # All counts should be 0 when no files exist
    local obs_count
    obs_count=$(echo "$result" | jq -r '.result.observations')
    [[ "$obs_count" -eq 0 ]] || return 1

    local inst_total
    inst_total=$(echo "$result" | jq -r '.result.instincts.total')
    [[ "$inst_total" -eq 0 ]] || return 1
}

# ============================================================================
# Test 9: critic detects contradictions between instincts
# ============================================================================
test_critic_detects_contradictions() {
    local tmp_dir
    tmp_dir=$(setup_curation_env)

    cat > "$tmp_dir/.aether/data/instincts.json" << 'EOF'
{
  "version": "1.0",
  "instincts": [
    {
      "id": "inst_a",
      "trigger": "when writing functions",
      "action": "always add docstrings",
      "domain": "style",
      "trust_score": 0.80,
      "trust_tier": "trusted",
      "confidence": 0.80,
      "archived": false,
      "provenance": {"created_at": "2026-01-01T00:00:00Z"},
      "application_history": [],
      "related_instincts": []
    },
    {
      "id": "inst_b",
      "trigger": "when writing functions",
      "action": "never add docstrings",
      "domain": "style",
      "trust_score": 0.60,
      "trust_tier": "emerging",
      "confidence": 0.60,
      "archived": false,
      "provenance": {"created_at": "2026-01-01T00:00:00Z"},
      "application_history": [],
      "related_instincts": []
    }
  ]
}
EOF

    local result
    result=$(run_cmd "$tmp_dir" curation-critic)

    rm -rf "$tmp_dir"

    assert_ok_true "$result" || return 1

    local contradiction_count
    contradiction_count=$(echo "$result" | jq -r '.result.count')
    [[ "$contradiction_count" -ge 1 ]] || return 1

    local first_a
    first_a=$(echo "$result" | jq -r '.result.contradictions[0].instinct_a')
    local first_b
    first_b=$(echo "$result" | jq -r '.result.contradictions[0].instinct_b')
    [[ -n "$first_a" && "$first_a" != "null" ]] || return 1
    [[ -n "$first_b" && "$first_b" != "null" ]] || return 1
}

# ============================================================================
# Test 10: critic auto-resolve archives lower-trust instinct
# ============================================================================
test_critic_auto_resolve() {
    local tmp_dir
    tmp_dir=$(setup_curation_env)

    cat > "$tmp_dir/.aether/data/instincts.json" << 'EOF'
{
  "version": "1.0",
  "instincts": [
    {
      "id": "inst_high",
      "trigger": "when deploying code",
      "action": "always run full test suite",
      "domain": "workflow",
      "trust_score": 0.85,
      "trust_tier": "trusted",
      "confidence": 0.85,
      "archived": false,
      "provenance": {"created_at": "2026-01-01T00:00:00Z"},
      "application_history": [],
      "related_instincts": []
    },
    {
      "id": "inst_low",
      "trigger": "when deploying code",
      "action": "never run full test suite",
      "domain": "workflow",
      "trust_score": 0.55,
      "trust_tier": "provisional",
      "confidence": 0.55,
      "archived": false,
      "provenance": {"created_at": "2026-01-01T00:00:00Z"},
      "application_history": [],
      "related_instincts": []
    }
  ]
}
EOF

    local result
    result=$(run_cmd "$tmp_dir" curation-critic --auto-resolve)

    local instincts_after
    instincts_after=$(cat "$tmp_dir/.aether/data/instincts.json")

    rm -rf "$tmp_dir"

    assert_ok_true "$result" || return 1

    local count
    count=$(echo "$result" | jq -r '.result.count')
    [[ "$count" -ge 1 ]] || return 1

    # The lower-trust instinct should now be archived
    local low_archived
    low_archived=$(echo "$instincts_after" | jq -r '.instincts[] | select(.id == "inst_low") | .archived')
    [[ "$low_archived" == "true" ]] || return 1

    # The higher-trust instinct should still be active
    local high_archived
    high_archived=$(echo "$instincts_after" | jq -r '.instincts[] | select(.id == "inst_high") | .archived')
    [[ "$high_archived" == "false" ]] || return 1
}

# ============================================================================
# Test 11: critic handles missing instincts file gracefully
# ============================================================================
test_critic_missing_file() {
    local tmp_dir
    tmp_dir=$(setup_curation_env)

    local result
    result=$(run_cmd "$tmp_dir" curation-critic)

    rm -rf "$tmp_dir"

    assert_ok_true "$result" || return 1

    local count
    count=$(echo "$result" | jq -r '.result.count')
    [[ "$count" -eq 0 ]] || return 1
}

# ============================================================================
# Test 12: All module files exist and have valid bash syntax
# ============================================================================
test_module_files_valid_syntax() {
    local modules=(
        "$PROJECT_ROOT/.aether/utils/curation-ants/nurse.sh"
        "$PROJECT_ROOT/.aether/utils/curation-ants/herald.sh"
        "$PROJECT_ROOT/.aether/utils/curation-ants/librarian.sh"
        "$PROJECT_ROOT/.aether/utils/curation-ants/critic.sh"
    )

    for module in "${modules[@]}"; do
        assert_file_exists "$module" || return 1
        bash -n "$module" 2>/dev/null || return 1
    done
}

# ============================================================================
# Run all tests
# ============================================================================
run_test "test_module_files_valid_syntax"   "module files exist with valid syntax"
run_test "test_nurse_recalculates_trust"    "nurse recalculates trust scores"
run_test "test_nurse_dry_run"               "nurse dry-run does not write"
run_test "test_nurse_missing_file"          "nurse handles missing observations file"
run_test "test_herald_promotes_instincts"   "herald promotes qualifying instincts"
run_test "test_herald_dry_run"              "herald dry-run reports without writing"
run_test "test_herald_missing_file"         "herald handles missing instincts file"
run_test "test_librarian_inventory"         "librarian returns inventory stats"
run_test "test_librarian_all_missing"       "librarian handles all missing files"
run_test "test_critic_detects_contradictions" "critic detects contradictions"
run_test "test_critic_auto_resolve"         "critic auto-resolve archives lower-trust"
run_test "test_critic_missing_file"         "critic handles missing instincts file"

test_summary
