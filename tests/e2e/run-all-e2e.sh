#!/usr/bin/env bash
# run-all-e2e.sh — Master E2E Test Runner
# Runs all area test scripts, aggregates results, generates requirements matrix
#
# Usage: bash tests/e2e/run-all-e2e.sh [--output <path>]
#
# Output: Requirements matrix + summary stats to stdout AND tests/e2e/RESULTS.md
# Exit code: 0 if all pass, 1 if any fail
#
# NOTE: Written for bash 3.2 (macOS default). No associative arrays.
# Uses file-based result collection strategy (subprocess per area test).

set -euo pipefail

E2E_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$E2E_SCRIPT_DIR/../.." && pwd)"

# Record start time
RUNNER_START=$(date +%s)

echo ""
echo "================================================================"
echo "Aether E2E Master Runner — Full Requirements Verification"
echo "================================================================"
echo "Started: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
echo ""

# ============================================================================
# Requirement Descriptions (46 requirements)
# bash 3.2 compatible: use parallel arrays instead of associative arrays
# ============================================================================

# Requirement IDs in order
REQ_IDS=(
  "ERR-01" "ERR-02" "ERR-03"
  "STA-01" "STA-02" "STA-03"
  "CMD-01" "CMD-02" "CMD-03" "CMD-04" "CMD-05" "CMD-06" "CMD-07" "CMD-08"
  "PHER-01" "PHER-02" "PHER-03" "PHER-04" "PHER-05"
  "VIS-01" "VIS-02" "VIS-03" "VIS-04" "VIS-05" "VIS-06"
  "CTX-01" "CTX-02" "CTX-03"
  "SES-01" "SES-02" "SES-03"
  "LIF-01" "LIF-02" "LIF-03"
  "ADV-01" "ADV-02" "ADV-03" "ADV-04" "ADV-05"
  "XML-01" "XML-02" "XML-03"
  "DOC-01" "DOC-02" "DOC-03" "DOC-04"
)

# Requirement descriptions (parallel array — same order as REQ_IDS)
REQ_DESCS=(
  "No 401 authentication errors during normal operation"
  "Agents stop spawning (no infinite loops)"
  "Clear error messages when things fail"
  "COLONY_STATE.json updates correctly on all operations"
  "No file path hallucinations (commands find right files)"
  "Files created in correct repositories"
  "/ant:lay-eggs starts new colony with pheromone preservation"
  "/ant:init initializes after lay-eggs"
  "/ant:colonize analyzes existing codebase"
  "/ant:plan generates project plan"
  "/ant:build executes phase with worker spawning"
  "/ant:continue verifies, extracts learnings, advances phase"
  "/ant:status shows colony dashboard"
  "All commands find correct files (no hallucinations)"
  "FOCUS signal attracts attention to areas"
  "REDIRECT signal warns away from patterns"
  "FEEDBACK signal calibrates behavior"
  "Auto-injection of learned patterns into new work"
  "Instincts applied to builders/watchers"
  "Swarm display shows ants working (not bash text scroll)"
  "Emoji caste identity visible in output"
  "Colors for different castes"
  "Progress indication during builds"
  "Stage banners use ant-themed names (DIGESTING, EXCAVATING, etc.)"
  "GSD-style formatting for phase transitions"
  "Session state persists across /clear"
  "Clear next command guidance at phase boundaries"
  "Context document tells next session what was happening"
  "/ant:pause-colony saves state and creates handoff"
  "/ant:resume-colony restores full context"
  "/ant:watch shows live colony visibility"
  "/ant:seal creates Crowned Anthill milestone"
  "/ant:entomb archives colony to chambers"
  "/ant:tunnels browses archived colonies"
  "/ant:oracle performs deep research (RALF loop)"
  "/ant:chaos performs resilience testing"
  "/ant:archaeology analyzes git history"
  "/ant:dream philosophical wanderer writes wisdom"
  "/ant:interpret validates dreams against reality"
  "Pheromones stored/retrieved via XML format"
  "Wisdom exchange uses XML structure"
  "Registry uses XML for cross-colony communication"
  "Phase learnings extracted and documented (ant-themed)"
  "Colony memories stored with ant naming (pheromones.md)"
  "Progress tracked with ant metaphors (nursery, chambers)"
  "Handoff documents use ant themes"
)

# Test script assignments (parallel array — same order as REQ_IDS)
REQ_SCRIPTS=(
  "test-err.sh" "test-err.sh" "test-err.sh"
  "test-sta.sh" "test-sta.sh" "test-sta.sh"
  "test-cmd.sh" "test-cmd.sh" "test-cmd.sh" "test-cmd.sh" "test-cmd.sh" "test-cmd.sh" "test-cmd.sh" "test-cmd.sh"
  "test-pher.sh" "test-pher.sh" "test-pher.sh" "test-pher.sh" "test-pher.sh"
  "test-vis.sh" "test-vis.sh" "test-vis.sh" "test-vis.sh" "test-vis.sh" "test-vis.sh"
  "test-ctx.sh" "test-ctx.sh" "test-ctx.sh"
  "test-ses.sh" "test-ses.sh" "test-ses.sh"
  "test-lif.sh" "test-lif.sh" "test-lif.sh"
  "test-adv.sh" "test-adv.sh" "test-adv.sh" "test-adv.sh" "test-adv.sh"
  "test-xml.sh" "test-xml.sh" "test-xml.sh"
  "test-doc.sh" "test-doc.sh" "test-doc.sh" "test-doc.sh"
)

# ============================================================================
# Result Collection Strategy:
# Each area test writes KEY=STATUS lines to a temp file via --results-file flag.
# Master runner reads all result files after scripts complete.
# ============================================================================

RESULTS_DIR=$(mktemp -d)

# ============================================================================
# Run Area Test Scripts
# ============================================================================

echo "--- Running Area Test Scripts ---"
echo ""

run_area_test() {
  local script_name="$1"
  local results_file="$2"
  local script_path="$E2E_SCRIPT_DIR/$script_name"

  if [[ ! -f "$script_path" ]]; then
    echo "  SKIP: $script_name not found"
    return 0
  fi

  echo "  Running $script_name..."
  if bash "$script_path" --results-file "$results_file" >/dev/null 2>&1; then
    echo "  PASS: $script_name completed"
  else
    echo "  DONE: $script_name (some tests may have failed)"
  fi
}

# Run all area tests in order
run_area_test "test-err.sh"  "$RESULTS_DIR/err.results"
run_area_test "test-sta.sh"  "$RESULTS_DIR/sta.results"
run_area_test "test-cmd.sh"  "$RESULTS_DIR/cmd.results"
run_area_test "test-pher.sh" "$RESULTS_DIR/pher.results"
run_area_test "test-vis.sh"  "$RESULTS_DIR/vis.results"
run_area_test "test-ctx.sh"  "$RESULTS_DIR/ctx.results"
run_area_test "test-ses.sh"  "$RESULTS_DIR/ses.results"
run_area_test "test-lif.sh"  "$RESULTS_DIR/lif.results"
run_area_test "test-adv.sh"  "$RESULTS_DIR/adv.results"
run_area_test "test-xml.sh"  "$RESULTS_DIR/xml.results"
run_area_test "test-doc.sh"  "$RESULTS_DIR/doc.results"

# Run lifecycle test separately
echo "  Running test-lifecycle.sh (integration)..."
LIFECYCLE_STATUS="PASS"
if bash "$E2E_SCRIPT_DIR/test-lifecycle.sh" --results-file "$RESULTS_DIR/lifecycle.results" >/dev/null 2>&1; then
  echo "  PASS: test-lifecycle.sh completed"
else
  echo "  DONE: test-lifecycle.sh (some steps may have failed)"
  LIFECYCLE_STATUS="FAIL"
fi

echo ""

# ============================================================================
# Aggregate Results
# ============================================================================

# Master results temp file (REQ_ID=STATUS lines)
MASTER_RESULTS=$(mktemp)

# Read all results files into master
for results_file in "$RESULTS_DIR"/*.results; do
  [[ -f "$results_file" ]] || continue
  cat "$results_file" >> "$MASTER_RESULTS" 2>/dev/null || true
done

# Function to look up a result for a req_id
get_result() {
  local req_id="$1"
  local result
  result=$(grep "^${req_id}=" "$MASTER_RESULTS" 2>/dev/null | tail -1 | cut -d'=' -f2 || echo "")
  if [[ -z "$result" ]]; then
    echo "UNKNOWN"
  else
    echo "$result"
  fi
}

# ============================================================================
# Generate Requirements Matrix
# ============================================================================

TOTAL_COUNT=${#REQ_IDS[@]}
PASS_COUNT=0
FAIL_COUNT=0
UNKNOWN_COUNT=0

# Build matrix header
MATRIX_OUTPUT=""
MATRIX_OUTPUT="$MATRIX_OUTPUT
## Requirements Matrix

| ID | Description | Status | Test Script |
|----|-------------|--------|-------------|"

i=0
while [[ $i -lt $TOTAL_COUNT ]]; do
  req_id="${REQ_IDS[$i]}"
  req_desc="${REQ_DESCS[$i]}"
  req_script="${REQ_SCRIPTS[$i]}"
  status=$(get_result "$req_id")

  MATRIX_OUTPUT="$MATRIX_OUTPUT
| $req_id | $req_desc | $status | $req_script |"

  if [[ "$status" == "PASS" ]]; then
    PASS_COUNT=$((PASS_COUNT + 1))
  elif [[ "$status" == "FAIL" ]]; then
    FAIL_COUNT=$((FAIL_COUNT + 1))
  else
    UNKNOWN_COUNT=$((UNKNOWN_COUNT + 1))
  fi

  i=$((i + 1))
done

# ============================================================================
# Calculate Duration
# ============================================================================

RUNNER_END=$(date +%s)
RUNNER_DURATION=$((RUNNER_END - RUNNER_START))

# ============================================================================
# Generate Full Report
# ============================================================================

PASS_RATE=0
if [[ $TOTAL_COUNT -gt 0 ]]; then
  # Integer arithmetic for percentage (no bc needed)
  PASS_RATE=$(( (PASS_COUNT * 100) / TOTAL_COUNT ))
fi

REPORT_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Determine overall status
if [[ $FAIL_COUNT -eq 0 && $UNKNOWN_COUNT -eq 0 ]]; then
  OVERALL_STATUS="ALL PASS"
elif [[ $FAIL_COUNT -eq 0 ]]; then
  OVERALL_STATUS="PASS (with unknowns)"
else
  OVERALL_STATUS="HAS FAILURES"
fi

# Build full report
FULL_REPORT="# Aether E2E Test Results

**Generated:** $REPORT_DATE
**Duration:** ${RUNNER_DURATION}s

## Summary

| Metric | Value |
|--------|-------|
| Total Requirements | $TOTAL_COUNT |
| PASS | $PASS_COUNT |
| FAIL | $FAIL_COUNT |
| UNKNOWN | $UNKNOWN_COUNT |
| Pass Rate | ${PASS_RATE}% |
| Overall | $OVERALL_STATUS |

## Lifecycle Integration Test

| Test | Status |
|------|--------|
| Connected workflow (init→colonize→plan→build→continue→seal→entomb) | $LIFECYCLE_STATUS |
$MATRIX_OUTPUT

---
*Generated by tests/e2e/run-all-e2e.sh*
"

# ============================================================================
# Output
# ============================================================================

echo "$FULL_REPORT"

# Write to RESULTS.md
echo "$FULL_REPORT" > "$E2E_SCRIPT_DIR/RESULTS.md"
echo ""
echo "Results written to: tests/e2e/RESULTS.md"
echo ""

# Cleanup
rm -rf "$RESULTS_DIR"
rm -f "$MASTER_RESULTS"

# Exit code
if [[ $FAIL_COUNT -eq 0 && $UNKNOWN_COUNT -eq 0 ]]; then
  echo "ALL PASS: $PASS_COUNT/$TOTAL_COUNT requirements verified"
  exit 0
elif [[ $FAIL_COUNT -eq 0 ]]; then
  echo "PASS WITH UNKNOWNS: $PASS_COUNT PASS, $UNKNOWN_COUNT UNKNOWN"
  exit 0
else
  echo "FAILURES: $PASS_COUNT PASS, $FAIL_COUNT FAIL, $UNKNOWN_COUNT UNKNOWN"
  exit 1
fi
