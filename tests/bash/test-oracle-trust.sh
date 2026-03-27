#!/bin/bash
# Test suite for oracle trust scoring functions
# Phase 09 Plan 02 -- validates compute_trust_scores, backward compatibility,
# generate_research_plan trust section, and validate-oracle-state v1.1

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ORACLE_SH="$AETHER_ROOT/.aether/utils/oracle/oracle.sh"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Colors for output (if terminal supports it)
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Helper: Run test and track result
run_test() {
  local name="$1"
  local expected="$2"
  local actual="$3"

  TESTS_RUN=$((TESTS_RUN + 1))

  if [[ "$actual" == *"$expected"* ]]; then
    echo -e "${GREEN}PASS${NC}: $name"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    return 0
  else
    echo -e "${RED}FAIL${NC}: $name"
    echo "  Expected: $expected"
    echo "  Actual: $actual"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    return 1
  fi
}

# Helper: Setup temporary directory
setup_tmpdir() {
  mktemp -d
}

# Helper: Cleanup temporary directory
cleanup_tmpdir() {
  local dir="$1"
  rm -rf "$dir"
}

# Helper: Write state.json
write_state() {
  local dir="$1"
  local phase="$2"
  local iteration="${3:-0}"
  cat > "$dir/state.json" <<EOF
{
  "version": "1.0",
  "topic": "Test topic",
  "scope": "codebase",
  "phase": "$phase",
  "iteration": $iteration,
  "max_iterations": 15,
  "target_confidence": 95,
  "overall_confidence": 0,
  "started_at": "2026-03-13T00:00:00Z",
  "last_updated": "2026-03-13T00:00:00Z",
  "status": "active"
}
EOF
}

# Helper: Write v1.0 plan.json with string findings (legacy)
write_plan() {
  local dir="$1"
  local questions_json="$2"
  cat > "$dir/plan.json" <<EOF
{
  "version": "1.0",
  "questions": $questions_json,
  "created_at": "2026-03-13T00:00:00Z",
  "last_updated": "2026-03-13T00:00:00Z"
}
EOF
}

# Helper: Write v1.1 plan.json with structured findings and sources registry
# Uses jq to construct valid JSON from argument strings
write_plan_with_sources() {
  local dir="$1"
  local questions_json="$2"
  local sources_json="${3:-\{\}}"
  jq -n \
    --argjson q "$questions_json" \
    --argjson s "$sources_json" \
    '{version: "1.1", sources: $s, questions: $q, created_at: "2026-03-13T00:00:00Z", last_updated: "2026-03-13T00:00:00Z"}' \
    > "$dir/plan.json"
}


# ---- Test 1: compute_trust_scores basic counting ----
test_compute_trust_scores_basic() {
  local tmpdir=$(setup_tmpdir)

  # Setup: 3 structured findings -- 1 multi-source, 1 single-source, 1 no-source
  write_plan_with_sources "$tmpdir" '[
    {
      "id": "q1", "text": "Q1?", "status": "partial", "confidence": 70,
      "key_findings": [
        {"text": "Multi finding", "source_ids": ["S1", "S2"], "iteration": 1},
        {"text": "Single finding", "source_ids": ["S1"], "iteration": 1},
        {"text": "No source finding", "source_ids": [], "iteration": 1}
      ],
      "iterations_touched": [1]
    }
  ]' '{"S1": {"url": "https://a.com", "title": "A", "date_accessed": "2026-03-13", "type": "documentation"}, "S2": {"url": "https://b.com", "title": "B", "date_accessed": "2026-03-13", "type": "blog"}}'

  # Extract and run compute_trust_scores
  bash -c "
    set +e
    eval \"\$(sed -n '/^compute_trust_scores()/,/^}/p' '$ORACLE_SH')\"
    compute_trust_scores '$tmpdir/plan.json'
  "

  # Assertion 1: total_findings == 3
  local total
  total=$(jq '.trust_summary.total_findings' "$tmpdir/plan.json")
  run_test "trust_scores_total_findings" "3" "$total"

  # Assertion 2: multi_source == 1
  local multi
  multi=$(jq '.trust_summary.multi_source' "$tmpdir/plan.json")
  run_test "trust_scores_multi_source" "1" "$multi"

  # Assertion 3: single_source == 1
  local single
  single=$(jq '.trust_summary.single_source' "$tmpdir/plan.json")
  run_test "trust_scores_single_source" "1" "$single"

  # Assertion 4: trust_ratio == 33
  local ratio
  ratio=$(jq '.trust_summary.trust_ratio' "$tmpdir/plan.json")
  run_test "trust_scores_trust_ratio" "33" "$ratio"

  cleanup_tmpdir "$tmpdir"
}


# ---- Test 2: compute_trust_scores backward compatibility ----
test_compute_trust_scores_backward_compat() {
  local tmpdir=$(setup_tmpdir)

  # v1.0 plan with string key_findings
  write_plan "$tmpdir" '[
    {"id":"q1","text":"Q1?","status":"partial","confidence":60,"key_findings":["finding one","finding two"],"iterations_touched":[1]},
    {"id":"q2","text":"Q2?","status":"open","confidence":30,"key_findings":["finding three"],"iterations_touched":[1]}
  ]'

  # Run compute_trust_scores
  bash -c "
    set +e
    eval \"\$(sed -n '/^compute_trust_scores()/,/^}/p' '$ORACLE_SH')\"
    compute_trust_scores '$tmpdir/plan.json'
  "

  # Assertion: trust_summary should NOT exist (function returns early)
  local trust_val
  trust_val=$(jq '.trust_summary // null' "$tmpdir/plan.json")
  run_test "backward_compat_no_trust_summary" "null" "$trust_val"

  cleanup_tmpdir "$tmpdir"
}


# ---- Test 3: trust scores all multi-source ----
test_trust_scores_all_multi_source() {
  local tmpdir=$(setup_tmpdir)

  write_plan_with_sources "$tmpdir" '[
    {
      "id": "q1", "text": "Q1?", "status": "answered", "confidence": 90,
      "key_findings": [
        {"text": "Finding A", "source_ids": ["S1", "S2"], "iteration": 1},
        {"text": "Finding B", "source_ids": ["S1", "S3"], "iteration": 2}
      ],
      "iterations_touched": [1, 2]
    }
  ]' '{"S1": {"url": "https://a.com", "title": "A", "date_accessed": "2026-03-13", "type": "documentation"}, "S2": {"url": "https://b.com", "title": "B", "date_accessed": "2026-03-13", "type": "blog"}, "S3": {"url": "https://c.com", "title": "C", "date_accessed": "2026-03-13", "type": "codebase"}}'

  bash -c "
    set +e
    eval \"\$(sed -n '/^compute_trust_scores()/,/^}/p' '$ORACLE_SH')\"
    compute_trust_scores '$tmpdir/plan.json'
  "

  local ratio
  ratio=$(jq '.trust_summary.trust_ratio' "$tmpdir/plan.json")
  run_test "all_multi_source_trust_ratio_100" "100" "$ratio"

  cleanup_tmpdir "$tmpdir"
}


# ---- Test 4: generate_research_plan trust section ----
test_generate_research_plan_trust_section() {
  local tmpdir=$(setup_tmpdir)

  write_state "$tmpdir" "investigate" 3
  write_plan_with_sources "$tmpdir" '[
    {
      "id": "q1", "text": "What is X?", "status": "partial", "confidence": 60,
      "key_findings": [
        {"text": "F1", "source_ids": ["S1", "S2"], "iteration": 1}
      ],
      "iterations_touched": [1, 2, 3]
    }
  ]' '{"S1": {"url": "https://a.com", "title": "A", "date_accessed": "2026-03-13", "type": "documentation"}}'

  # Pre-set trust_summary as if compute_trust_scores already ran
  jq '.trust_summary = {"total_findings": 1, "multi_source": 1, "single_source": 0, "no_source": 0, "trust_ratio": 100}' \
    "$tmpdir/plan.json" > "$tmpdir/plan.json.tmp" && mv "$tmpdir/plan.json.tmp" "$tmpdir/plan.json"

  # Extract and run generate_research_plan
  bash -c "
    set +e
    STATE_FILE='$tmpdir/state.json'
    PLAN_FILE='$tmpdir/plan.json'
    RESEARCH_PLAN_FILE='$tmpdir/research-plan.md'
    eval \"\$(sed -n '/^generate_research_plan()/,/^}/p' '$ORACLE_SH')\"
    generate_research_plan
  "

  # Assertion 1: research-plan.md contains Source Trust section
  local content
  content=$(cat "$tmpdir/research-plan.md" 2>/dev/null || echo "")
  run_test "research_plan_has_source_trust" "Source Trust" "$content"

  # Assertion 2: research-plan.md contains the trust ratio value
  run_test "research_plan_has_trust_ratio_value" "100%" "$content"

  cleanup_tmpdir "$tmpdir"
}


# ---- Test 5: validate-oracle-state v1.1 acceptance ----
test_validate_oracle_state_v11() {
  local tmpdir=$(setup_tmpdir)

  write_plan_with_sources "$tmpdir" '[
    {
      "id": "q1", "text": "What is X?", "status": "open", "confidence": 30,
      "key_findings": [
        {"text": "Initial finding", "source_ids": ["S1"], "iteration": 1}
      ],
      "iterations_touched": [1]
    }
  ]' '{"S1": {"url": "https://example.com", "title": "Example", "date_accessed": "2026-03-13", "type": "documentation"}}'

  local result
  result=$(ORACLE_DIR="$tmpdir" bash "$AETHER_ROOT/.aether/aether-utils.sh" validate-oracle-state plan 2>/dev/null)

  # Assertion: output contains "pass": true
  run_test "validate_v11_plan_passes" '"pass": true' "$result"

  cleanup_tmpdir "$tmpdir"
}


# Run all tests
echo "========================================="
echo "Oracle Trust Scoring Test Suite"
echo "========================================="
echo ""

test_compute_trust_scores_basic
test_compute_trust_scores_backward_compat
test_trust_scores_all_multi_source
test_generate_research_plan_trust_section
test_validate_oracle_state_v11

echo ""
echo "========================================="
echo "Oracle Trust Tests: $TESTS_PASSED passed, $TESTS_FAILED failed out of $TESTS_RUN"
echo "========================================="

if [[ $TESTS_FAILED -eq 0 ]]; then
  echo -e "${GREEN}All tests passed!${NC}"
  exit 0
else
  echo -e "${RED}Some tests failed!${NC}"
  exit 1
fi
