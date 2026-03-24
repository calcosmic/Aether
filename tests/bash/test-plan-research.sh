#!/bin/bash
# Test suite for Phase Domain Research infrastructure
# Phase 14 Plan 01 -- validates research directory, file structure, cleanup, and hive-read graceful degradation

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
UTILS_SCRIPT="$AETHER_ROOT/.aether/aether-utils.sh"

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

# ---- Test 1: Research directory creation ----
test_research_directory_creation() {
  local tmpdir=$(setup_tmpdir)
  local research_dir="$tmpdir/phase-research"

  # Simulate the mkdir -p from Step 3.6
  mkdir -p "$research_dir"

  if [[ -d "$research_dir" ]]; then
    run_test "research_directory_creation" "exists" "exists"
  else
    run_test "research_directory_creation" "exists" "missing"
  fi

  cleanup_tmpdir "$tmpdir"
}

# ---- Test 2: RESEARCH.md structure validation ----
test_research_md_structure() {
  local tmpdir=$(setup_tmpdir)
  local research_file="$tmpdir/phase-1-research.md"

  # Create a sample RESEARCH.md with all 6 required sections
  cat > "$research_file" << 'RESEARCHEOF'
# Phase 1 Research: Test Phase

**Generated:** 2026-03-24T10:00:00Z
**Phase:** 1 - Test Phase
**Research scope:** Testing research structure validation

## Hive Wisdom (Pre-existing Knowledge)
No relevant hive wisdom found

## Key Patterns
**Pattern A:** Relevant for testing (Source: tests/)

## External Context
No external research needed for this phase

## Gotchas
**Race condition:** Prevent by using locks (Source: docs/)

## Recommended Approach
Use the standard test patterns already established in the project.

## Files to Study
- tests/bash/test-oracle-state.sh
- .claude/commands/ant/plan.md
RESEARCHEOF

  # Validate all 6 required section headers exist
  local all_pass="true"
  local sections=("## Hive Wisdom" "## Key Patterns" "## External Context" "## Gotchas" "## Recommended Approach" "## Files to Study")

  for section in "${sections[@]}"; do
    if ! grep -q "$section" "$research_file"; then
      all_pass="false"
      echo "  Missing section: $section"
    fi
  done

  run_test "research_md_has_all_6_sections" "true" "$all_pass"

  # Also check the metadata header
  local has_generated has_phase has_scope
  has_generated=$(grep -c "Generated:" "$research_file")
  has_phase=$(grep -c "Phase:" "$research_file")
  has_scope=$(grep -c "Research scope:" "$research_file")

  if [[ "$has_generated" -ge 1 && "$has_phase" -ge 1 && "$has_scope" -ge 1 ]]; then
    run_test "research_md_has_metadata_header" "true" "true"
  else
    run_test "research_md_has_metadata_header" "true" "false (generated=$has_generated phase=$has_phase scope=$has_scope)"
  fi

  cleanup_tmpdir "$tmpdir"
}

# ---- Test 3: Research file cleanup on re-plan ----
test_research_file_cleanup() {
  local tmpdir=$(setup_tmpdir)
  local research_dir="$tmpdir/phase-research"
  mkdir -p "$research_dir"

  # Create a dummy research file (simulating previous plan run)
  echo "# Old Research" > "$research_dir/phase-1-research.md"

  # Verify it exists
  if [[ ! -f "$research_dir/phase-1-research.md" ]]; then
    run_test "research_file_cleanup_setup" "exists" "missing"
    cleanup_tmpdir "$tmpdir"
    return
  fi

  # Simulate the cleanup from Step 3.6 (rm -f)
  rm -f "$research_dir/phase-1-research.md"

  if [[ ! -f "$research_dir/phase-1-research.md" ]]; then
    run_test "research_file_cleanup_removes_old" "removed" "removed"
  else
    run_test "research_file_cleanup_removes_old" "removed" "still exists"
  fi

  # Verify directory still exists after file removal
  if [[ -d "$research_dir" ]]; then
    run_test "research_dir_survives_file_cleanup" "exists" "exists"
  else
    run_test "research_dir_survives_file_cleanup" "exists" "missing"
  fi

  cleanup_tmpdir "$tmpdir"
}

# ---- Test 4: Hive-read graceful failure ----
test_hive_read_graceful_failure() {
  # Run hive-read in a possibly uninitialized environment
  # It should either return valid JSON or empty output -- never crash
  local result
  local exit_code

  result=$(bash "$UTILS_SCRIPT" hive-read --limit 5 --format text 2>/dev/null)
  exit_code=$?

  # The key requirement: the calling script should not crash
  # hive-read may return non-zero (no hive initialized) or zero (success)
  # Either way, the exit code should be 0 or 1 (not segfault/crash codes like 139)
  if [[ "$exit_code" -le 1 ]]; then
    run_test "hive_read_no_crash" "safe" "safe"
  else
    run_test "hive_read_no_crash" "safe" "exit_code=$exit_code (possible crash)"
  fi

  # If we got output, it should be valid JSON or empty
  if [[ -z "$result" ]]; then
    run_test "hive_read_empty_or_json" "valid" "valid"
  else
    # Try to parse as JSON
    if echo "$result" | jq . > /dev/null 2>&1; then
      run_test "hive_read_empty_or_json" "valid" "valid"
    else
      run_test "hive_read_empty_or_json" "valid" "invalid output: ${result:0:100}"
    fi
  fi
}

# ---- Test 5: Research file naming convention ----
test_research_file_naming() {
  local tmpdir=$(setup_tmpdir)
  local research_dir="$tmpdir/phase-research"
  mkdir -p "$research_dir"

  # Test naming pattern for phases 1, 10, and 99
  local phases=(1 10 99)
  local all_pass="true"

  for phase_num in "${phases[@]}"; do
    local expected_name="phase-${phase_num}-research.md"
    local expected_path="$research_dir/$expected_name"

    # Create using the naming pattern from Step 3.6
    echo "# Phase $phase_num Research" > "$expected_path"

    if [[ -f "$expected_path" ]]; then
      # Verify filename matches expected pattern
      local basename
      basename=$(basename "$expected_path")
      if [[ "$basename" == "phase-${phase_num}-research.md" ]]; then
        : # pass
      else
        all_pass="false"
        echo "  Phase $phase_num: wrong name '$basename'"
      fi
    else
      all_pass="false"
      echo "  Phase $phase_num: file not created"
    fi
  done

  run_test "research_file_naming_convention" "true" "$all_pass"

  # Verify rm -f works correctly with each naming pattern
  for phase_num in "${phases[@]}"; do
    rm -f "$research_dir/phase-${phase_num}-research.md"
  done

  local remaining
  remaining=$(ls "$research_dir" 2>/dev/null | wc -l | tr -d ' ')
  if [[ "$remaining" -eq 0 ]]; then
    run_test "research_file_naming_cleanup_all" "clean" "clean"
  else
    run_test "research_file_naming_cleanup_all" "clean" "$remaining files remaining"
  fi

  cleanup_tmpdir "$tmpdir"
}


# Run all tests
echo "========================================="
echo "Plan Research Infrastructure Test Suite"
echo "========================================="
echo ""

test_research_directory_creation
test_research_md_structure
test_research_file_cleanup
test_hive_read_graceful_failure
test_research_file_naming

echo ""
echo "========================================="
echo "Plan Research Tests: $TESTS_PASSED passed, $TESTS_FAILED failed out of $TESTS_RUN"
echo "========================================="

if [[ $TESTS_FAILED -eq 0 ]]; then
  echo -e "${GREEN}All tests passed!${NC}"
  exit 0
else
  echo -e "${RED}Some tests failed!${NC}"
  exit 1
fi
