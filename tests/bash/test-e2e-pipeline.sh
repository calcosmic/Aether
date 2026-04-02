#!/usr/bin/env bash
# End-to-End Integration Test: Structural Learning Stack Pipeline
# Exercises the full lifecycle: observe -> instinct-store -> graph -> consolidation -> curation -> seal

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
# Helper: build a fully isolated environment for one test
# ============================================================================
setup_e2e_env() {
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

    # Minimal COLONY_STATE.json
    cat > "$tmpdir/.aether/data/COLONY_STATE.json" <<'EOF'
{
  "version": "3.0",
  "goal": "e2e pipeline test",
  "state": "READY",
  "current_phase": 1,
  "session_id": "test-e2e",
  "initialized_at": "2026-01-01T00:00:00Z",
  "build_started_at": null,
  "plan": { "phases": [{ "id": 1, "name": "E2E Phase", "status": "pending" }] },
  "memory": { "phase_learnings": [], "decisions": [], "instincts": [] },
  "errors": { "records": [], "flagged_patterns": [] },
  "events": [],
  "signals": [],
  "graveyards": []
}
EOF

    cat > "$tmpdir/.aether/data/pheromones.json" <<'EOF'
{"version":"2.0","signals":[]}
EOF

    # Empty observation and instinct files
    echo '{"observations":[]}' > "$tmpdir/.aether/data/learning-observations.json"
    echo '{"instincts":[]}' > "$tmpdir/.aether/data/instincts.json"

    echo "$tmpdir"
}

# Run a subcommand in the isolated env, suppressing stderr
run_cmd() {
    local tmpdir="$1"
    shift
    AETHER_ROOT="$tmpdir" \
        DATA_DIR="$tmpdir/.aether/data" \
        COLONY_DATA_DIR="$tmpdir/.aether/data" \
        bash "$tmpdir/.aether/aether-utils.sh" "$@" 2>/dev/null || true
}

# ============================================================================
# TEST 1: Full Pipeline — Observe to QUEEN.md
# Exercises: learning-observe -> instinct-store -> graph-link -> graph-neighbors
#            -> consolidation-phase-end
# ============================================================================
test_full_pipeline_observe_to_queen() {
    local tmpdir
    tmpdir=$(setup_e2e_env)

    # Step 1: Observe a learning
    local obs_result
    obs_result=$(run_cmd "$tmpdir" learning-observe \
        "Always write tests before implementation" \
        "pattern" \
        "test-colony" \
        "observation" \
        "anecdotal")

    assert_ok_true "$obs_result" || {
        log_error "learning-observe failed: $obs_result"
        rm -rf "$tmpdir"; return 1
    }

    # Step 2: Verify trust_score is present in observation output
    local trust_score
    trust_score=$(echo "$obs_result" | jq -r '.result.trust_score // "null"')
    [[ "$trust_score" != "null" ]] || {
        log_error "trust_score missing from learning-observe result: $obs_result"
        rm -rf "$tmpdir"; return 1
    }
    local ts_valid
    ts_valid=$(awk "BEGIN{print ($trust_score > 0)}" 2>/dev/null || echo "0")
    [[ "$ts_valid" == "1" ]] || {
        log_error "trust_score should be > 0, got: $trust_score"
        rm -rf "$tmpdir"; return 1
    }

    # Step 3: Store an instinct
    local store_result
    store_result=$(run_cmd "$tmpdir" instinct-store \
        --trigger "writing new code" \
        --action "write tests before implementation" \
        --domain "testing" \
        --confidence 0.8 \
        --source "e2e-test" \
        --evidence "observed pattern")

    assert_ok_true "$store_result" || {
        log_error "instinct-store failed: $store_result"
        rm -rf "$tmpdir"; return 1
    }

    # Step 4: Verify the instinct was stored and read back
    local read_result
    read_result=$(run_cmd "$tmpdir" instinct-read-trusted --min-score 0.5)

    assert_ok_true "$read_result" || {
        log_error "instinct-read-trusted failed: $read_result"
        rm -rf "$tmpdir"; return 1
    }

    local inst_count
    inst_count=$(echo "$read_result" | jq '.result.instincts | length')
    [[ "$inst_count" -ge 1 ]] || {
        log_error "expected at least 1 instinct, got: $inst_count"
        rm -rf "$tmpdir"; return 1
    }

    # Retrieve the stored instinct ID for graph linking
    local inst_id
    inst_id=$(jq -r '.instincts[0].id' "$tmpdir/.aether/data/instincts.json")

    # Step 5: Graph link — link the instinct to a companion node
    local companion_id="inst_companion_001"
    local link_result
    link_result=$(run_cmd "$tmpdir" graph-link \
        --source "$inst_id" \
        --target "$companion_id" \
        --relationship reinforces)

    assert_ok_true "$link_result" || {
        log_error "graph-link failed: $link_result"
        rm -rf "$tmpdir"; return 1
    }

    # Step 6: Verify graph link — neighbors should include companion
    local neighbors_result
    neighbors_result=$(run_cmd "$tmpdir" graph-neighbors --id "$inst_id")

    assert_ok_true "$neighbors_result" || {
        log_error "graph-neighbors failed: $neighbors_result"
        rm -rf "$tmpdir"; return 1
    }

    local neighbor_count
    neighbor_count=$(echo "$neighbors_result" | jq -r '.result.count')
    [[ "$neighbor_count" -ge 1 ]] || {
        log_error "expected at least 1 neighbor, got: $neighbor_count"
        rm -rf "$tmpdir"; return 1
    }

    local neighbor_ids
    neighbor_ids=$(echo "$neighbors_result" | jq -r '.result.neighbors[].id')
    echo "$neighbor_ids" | grep -q "$companion_id" || {
        log_error "companion node not found in neighbors: $neighbor_ids"
        rm -rf "$tmpdir"; return 1
    }

    # Step 7: Consolidation — run phase-end to exercise nurse/herald/janitor
    local consol_result
    consol_result=$(run_cmd "$tmpdir" consolidation-phase-end)

    assert_ok_true "$consol_result" || {
        log_error "consolidation-phase-end failed: $consol_result"
        rm -rf "$tmpdir"; return 1
    }

    # Steps array must be present with at least 3 entries (nurse, herald, janitor)
    local step_count
    step_count=$(echo "$consol_result" | jq '.result.steps | length')
    [[ "$step_count" -ge 3 ]] || {
        log_error "expected at least 3 consolidation steps, got: $step_count"
        rm -rf "$tmpdir"; return 1
    }

    rm -rf "$tmpdir"
}

# ============================================================================
# TEST 2: Trust Decay Lifecycle
# Stores a high-trust instinct, applies 2 half-lives of decay,
# verifies score dropped and tier updated to "dormant"
# ============================================================================
test_trust_decay_lifecycle() {
    local tmpdir
    tmpdir=$(setup_e2e_env)

    # Store instinct with high confidence (maps to high trust_score)
    run_cmd "$tmpdir" instinct-store \
        --trigger "when deploying always run smoke tests" \
        --action "execute smoke test suite before release" \
        --domain "workflow" \
        --confidence 0.9 \
        --source "e2e-test" \
        --evidence "test_verified" > /dev/null

    # Capture initial trust score
    local initial_score
    initial_score=$(jq -r '.instincts[0].trust_score' "$tmpdir/.aether/data/instincts.json")

    # Sanity: should be > 0.5 for a high-confidence instinct
    local high_enough
    high_enough=$(awk "BEGIN{print ($initial_score > 0.5)}" 2>/dev/null || echo "0")
    [[ "$high_enough" == "1" ]] || {
        log_error "initial trust_score too low for high-confidence instinct: $initial_score"
        rm -rf "$tmpdir"; return 1
    }

    # Apply decay with 120 days (2 half-lives at 60-day half-life)
    local decay_result
    decay_result=$(run_cmd "$tmpdir" instinct-decay-all --days 120)

    assert_ok_true "$decay_result" || {
        log_error "instinct-decay-all failed: $decay_result"
        rm -rf "$tmpdir"; return 1
    }

    # Read the decayed score from the file
    local decayed_score
    decayed_score=$(jq -r '.instincts[0].trust_score' "$tmpdir/.aether/data/instincts.json")

    # After 2 half-lives: score * 0.25 (allow floor at 0.2)
    # Score should be less than the initial
    local decreased
    decreased=$(awk "BEGIN{print ($decayed_score < $initial_score)}" 2>/dev/null || echo "0")
    [[ "$decreased" == "1" ]] || {
        log_error "trust_score should have decreased from $initial_score, got: $decayed_score"
        rm -rf "$tmpdir"; return 1
    }

    # After 2 half-lives from ~0.9 initial, score should be near floor (~0.225 or 0.2)
    # Accept any score <= 0.35 to handle floor enforcement
    local near_floor
    near_floor=$(awk "BEGIN{print ($decayed_score <= 0.35)}" 2>/dev/null || echo "0")
    [[ "$near_floor" == "1" ]] || {
        log_error "expected decayed score <= 0.35 after 2 half-lives, got: $decayed_score"
        rm -rf "$tmpdir"; return 1
    }

    # Trust tier should be "dormant" or "suspect" at this low score
    local trust_tier
    trust_tier=$(jq -r '.instincts[0].trust_tier' "$tmpdir/.aether/data/instincts.json")
    [[ "$trust_tier" == "dormant" || "$trust_tier" == "suspect" ]] || {
        log_error "expected trust_tier dormant or suspect after heavy decay, got: $trust_tier"
        rm -rf "$tmpdir"; return 1
    }

    rm -rf "$tmpdir"
}

# ============================================================================
# TEST 3: Curation-run End-to-End
# Seeds observations + instincts + graph edges + expired events,
# runs curation-run, verifies all 8 steps executed.
# ============================================================================
test_curation_run_e2e() {
    local tmpdir
    tmpdir=$(setup_e2e_env)

    # Seed instincts with mixed trust
    cat > "$tmpdir/.aether/data/instincts.json" <<'EOF'
{
  "instincts": [
    {
      "id": "inst-e2e-001",
      "trigger": "always validate inputs before processing",
      "action": "add validation at entry points",
      "confidence": 0.85,
      "trust_score": 0.85,
      "trust_tier": "trusted",
      "domain": "quality",
      "archived": false,
      "provenance": {
        "created_at": "2026-01-01T00:00:00Z",
        "source": "e2e-test"
      }
    },
    {
      "id": "inst-e2e-002",
      "trigger": "when refactoring large functions",
      "action": "split into smaller composable units",
      "confidence": 0.3,
      "trust_score": 0.22,
      "trust_tier": "dormant",
      "domain": "architecture",
      "archived": false,
      "provenance": {
        "created_at": "2025-01-01T00:00:00Z",
        "source": "old-phase"
      }
    }
  ]
}
EOF

    # Seed observations
    cat > "$tmpdir/.aether/data/learning-observations.json" <<'EOF'
{
  "observations": [
    {
      "content_hash": "sha256:e2e001",
      "content": "Validate at entry points reduces downstream errors",
      "wisdom_type": "pattern",
      "observation_count": 3,
      "first_seen": "2026-01-01T00:00:00Z",
      "last_seen": "2026-03-01T00:00:00Z",
      "colonies": ["e2e-colony"]
    }
  ]
}
EOF

    # Seed a graph edge between the two instincts
    cat > "$tmpdir/.aether/data/instinct-graph.json" <<'EOF'
{
  "version": "1.0",
  "edges": [
    {
      "source": "inst-e2e-001",
      "target": "inst-e2e-002",
      "relationship": "reinforces",
      "weight": 0.6,
      "created_at": "2026-01-01T00:00:00Z"
    }
  ]
}
EOF

    # Seed an expired event in the event bus
    local two_years_ago
    two_years_ago=$(date -u -v-730d '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || \
                    date -u --date='730 days ago' '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || \
                    echo "2024-01-01T00:00:00Z")
    echo "{\"topic\":\"test.expired\",\"timestamp\":\"$two_years_ago\",\"payload\":{}}" \
        > "$tmpdir/.aether/data/event-bus.jsonl"

    # Run curation
    local result
    result=$(run_cmd "$tmpdir" curation-run)

    assert_ok_true "$result" || {
        log_error "curation-run failed: $result"
        rm -rf "$tmpdir"; return 1
    }

    # Must have all 8 steps
    local total
    total=$(echo "$result" | jq '.result.total_steps')
    [[ "$total" -eq 8 ]] || {
        log_error "expected 8 total_steps, got: $total"
        rm -rf "$tmpdir"; return 1
    }

    # Verify all 8 step names are present
    local names
    names=$(echo "$result" | jq -r '.result.steps[].name' | tr '\n' ' ')
    for expected_name in sentinel nurse critic herald janitor archivist librarian scribe; do
        assert_contains "$names" "$expected_name" || {
            log_error "step '$expected_name' missing from: $names"
            rm -rf "$tmpdir"; return 1
        }
    done

    # succeeded + failed must equal 8
    local succeeded failed sum
    succeeded=$(echo "$result" | jq '.result.succeeded')
    failed=$(echo "$result" | jq '.result.failed')
    sum=$(( succeeded + failed ))
    [[ "$sum" -eq 8 ]] || {
        log_error "succeeded ($succeeded) + failed ($failed) should equal 8"
        rm -rf "$tmpdir"; return 1
    }

    rm -rf "$tmpdir"
}

# ============================================================================
# TEST 4: Consolidation-seal End-to-End
# Seeds mixed-trust instincts, runs consolidation-seal --dry-run,
# verifies all steps ran and scribe report was generated.
# ============================================================================
test_consolidation_seal_e2e() {
    local tmpdir
    tmpdir=$(setup_e2e_env)

    # Seed instincts with mixed trust scores
    cat > "$tmpdir/.aether/data/instincts.json" <<'EOF'
{
  "instincts": [
    {
      "id": "inst-seal-e2e-001",
      "trigger": "when writing integration tests isolate external services",
      "action": "use mocks or test doubles for external deps",
      "confidence": 0.9,
      "trust_score": 0.9,
      "trust_tier": "canonical",
      "domain": "testing",
      "archived": false,
      "provenance": {
        "created_at": "2026-01-01T00:00:00Z",
        "source": "e2e-test"
      }
    },
    {
      "id": "inst-seal-e2e-002",
      "trigger": "when reviewing performance metrics",
      "action": "baseline before optimising",
      "confidence": 0.6,
      "trust_score": 0.6,
      "trust_tier": "emerging",
      "domain": "performance",
      "archived": false,
      "provenance": {
        "created_at": "2026-02-01T00:00:00Z",
        "source": "e2e-test"
      }
    },
    {
      "id": "inst-seal-e2e-003",
      "trigger": "when handling deprecated patterns",
      "action": "archive and document the replacement",
      "confidence": 0.2,
      "trust_score": 0.2,
      "trust_tier": "dormant",
      "domain": "workflow",
      "archived": false,
      "provenance": {
        "created_at": "2025-06-01T00:00:00Z",
        "source": "old-phase"
      }
    }
  ]
}
EOF

    # Run seal with --dry-run so we verify steps without mutating state
    local result
    result=$(run_cmd "$tmpdir" consolidation-seal --dry-run)

    assert_ok_true "$result" || {
        log_error "consolidation-seal --dry-run failed: $result"
        rm -rf "$tmpdir"; return 1
    }

    # dry_run flag must be true
    local dry_flag
    dry_flag=$(echo "$result" | jq '.result.dry_run')
    [[ "$dry_flag" == "true" ]] || {
        log_error "result.dry_run should be true, got: $dry_flag"
        rm -rf "$tmpdir"; return 1
    }

    # Must have 4 steps: curation-run, instinct-decay-all, archivist, scribe
    local step_count
    step_count=$(echo "$result" | jq '.result.steps | length')
    [[ "$step_count" -eq 4 ]] || {
        log_error "expected 4 steps, got: $step_count"
        rm -rf "$tmpdir"; return 1
    }

    # All 4 step names must be present
    local names
    names=$(echo "$result" | jq -r '.result.steps[].name' | tr '\n' ' ')
    for expected_name in "curation-run" "instinct-decay-all" "archivist" "scribe"; do
        assert_contains "$names" "$expected_name" || {
            log_error "step '$expected_name' missing from: $names"
            rm -rf "$tmpdir"; return 1
        }
    done

    # Each step must have a status field
    local missing_status
    missing_status=$(echo "$result" | jq '[.result.steps[] | select(.status == null)] | length')
    [[ "$missing_status" -eq 0 ]] || {
        log_error "some steps missing status field"
        rm -rf "$tmpdir"; return 1
    }

    # Scribe step must have a report_path
    local report_path
    report_path=$(echo "$result" | jq -r \
        '.result.steps[] | select(.name == "scribe") | .report_path // "null"')
    [[ "$report_path" != "null" ]] || {
        log_error "scribe step report_path should not be null"
        rm -rf "$tmpdir"; return 1
    }

    # result.type must be "seal"
    local result_type
    result_type=$(echo "$result" | jq -r '.result.type')
    [[ "$result_type" == "seal" ]] || {
        log_error "result.type should be 'seal', got: $result_type"
        rm -rf "$tmpdir"; return 1
    }

    rm -rf "$tmpdir"
}

# ============================================================================
# Main
# ============================================================================
log_info "Running end-to-end pipeline integration tests..."
log_info ""

run_test "test_full_pipeline_observe_to_queen" \
    "E2E: Full Pipeline — observe -> instinct-store -> graph -> consolidation"
run_test "test_trust_decay_lifecycle" \
    "E2E: Trust Decay Lifecycle — high trust decays to near-floor after 2 half-lives"
run_test "test_curation_run_e2e" \
    "E2E: Curation-run end-to-end — all 8 steps execute with seeded data"
run_test "test_consolidation_seal_e2e" \
    "E2E: Consolidation-seal end-to-end — dry-run verifies all steps and scribe report"

test_summary
