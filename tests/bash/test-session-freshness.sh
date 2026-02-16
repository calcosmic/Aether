#!/bin/bash
# Test suite for session freshness detection utilities
# Phase 9 of Session Freshness Detection implementation

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
YELLOW='\033[1;33m'
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

# Test: session-verify-fresh with missing files
test_verify_fresh_missing() {
  local tmpdir=$(setup_tmpdir)

  local result
  result=$(SURVEY_DIR="$tmpdir" bash "$UTILS_SCRIPT" session-verify-fresh --command survey "" 0)

  # Missing files are OK - they will be created during the session
  # ok=true when no stale files exist (even if all files are missing)
  run_test "verify_fresh_missing" '"ok":true' "$result"
  # Check that stale array is empty (no files are stale when none exist)
  if [[ "$result" == *'"stale":[]'* ]]; then
    run_test "verify_fresh_missing_empty_stale" "empty stale array" "empty stale array"
  else
    run_test "verify_fresh_missing_empty_stale" "empty stale array" "stale array not empty"
  fi

  cleanup_tmpdir "$tmpdir"
}

# Test: session-verify-fresh with stale files
test_verify_fresh_stale() {
  local tmpdir=$(setup_tmpdir)

  # Create all required files with old timestamp
  for doc in PROVISIONS.md TRAILS.md BLUEPRINT.md CHAMBERS.md DISCIPLINES.md SENTINEL-PROTOCOLS.md PATHOGENS.md; do
    touch -t 202501010000 "$tmpdir/$doc"
  done

  local result
  result=$(SURVEY_DIR="$tmpdir" bash "$UTILS_SCRIPT" session-verify-fresh --command survey "" "$(date +%s)")

  # Verify stale array contains all files (they all have old timestamps)
  if [[ "$result" == *'"stale":["PROVISIONS.md"'* ]]; then
    run_test "verify_fresh_stale_contains" "stale contains PROVISIONS.md" "stale contains PROVISIONS.md"
  else
    run_test "verify_fresh_stale_contains" "stale contains PROVISIONS.md" "stale missing PROVISIONS.md"
  fi
  run_test "verify_fresh_stale_ok_false" '"ok":false' "$result"

  cleanup_tmpdir "$tmpdir"
}

# Test: session-verify-fresh with fresh files
test_verify_fresh_fresh() {
  local tmpdir=$(setup_tmpdir)

  # Create all required files fresh
  local start_time=$(date +%s)
  for doc in PROVISIONS.md TRAILS.md BLUEPRINT.md CHAMBERS.md DISCIPLINES.md SENTINEL-PROTOCOLS.md PATHOGENS.md; do
    echo "test" > "$tmpdir/$doc"
  done

  local result
  result=$(SURVEY_DIR="$tmpdir" bash "$UTILS_SCRIPT" session-verify-fresh --command survey "" "$start_time")

  # Verify fresh array contains all files
  if [[ "$result" == *'"fresh":["PROVISIONS.md"'* ]]; then
    run_test "verify_fresh_fresh_contains" "fresh contains PROVISIONS.md" "fresh contains PROVISIONS.md"
  else
    run_test "verify_fresh_fresh_contains" "fresh contains PROVISIONS.md" "fresh missing PROVISIONS.md"
  fi
  run_test "verify_fresh_fresh_ok_true" '"ok":true' "$result"

  cleanup_tmpdir "$tmpdir"
}

# Test: session-verify-fresh with force mode
test_verify_fresh_force() {
  local tmpdir=$(setup_tmpdir)

  # Create stale file
  touch -t 202501010000 "$tmpdir/PROVISIONS.md"

  local result
  result=$(SURVEY_DIR="$tmpdir" bash "$UTILS_SCRIPT" session-verify-fresh --command survey --force "" "$(date +%s)")

  # With --force, even stale files should be accepted
  run_test "verify_fresh_force" '"ok":true' "$result"

  cleanup_tmpdir "$tmpdir"
}

# Test: session-clear dry-run
test_clear_dry_run() {
  local tmpdir=$(setup_tmpdir)

  echo "test content" > "$tmpdir/PROVISIONS.md"

  local result
  result=$(SURVEY_DIR="$tmpdir" bash "$UTILS_SCRIPT" session-clear --command survey --dry-run)

  run_test "clear_dry_run" '"dry_run":true' "$result"

  # Verify file still exists
  if [[ -f "$tmpdir/PROVISIONS.md" ]]; then
    run_test "clear_dry_run_preserved" "PROVISIONS.md exists" "PROVISIONS.md exists"
  else
    run_test "clear_dry_run_preserved" "PROVISIONS.md exists" "PROVISIONS.md MISSING"
  fi

  cleanup_tmpdir "$tmpdir"
}

# Test: session-clear actual
test_clear_actual() {
  local tmpdir=$(setup_tmpdir)

  echo "test content" > "$tmpdir/PROVISIONS.md"

  local result
  result=$(SURVEY_DIR="$tmpdir" bash "$UTILS_SCRIPT" session-clear --command survey)

  # Verify file removed
  if [[ ! -f "$tmpdir/PROVISIONS.md" ]]; then
    run_test "clear_actual_removed" "file removed" "file removed"
  else
    run_test "clear_actual_removed" "file removed" "file still exists"
  fi

  cleanup_tmpdir "$tmpdir"
}

# Test: oracle command mapping
test_oracle_mapping() {
  local tmpdir=$(setup_tmpdir)

  local result
  result=$(ORACLE_DIR="$tmpdir" bash "$UTILS_SCRIPT" session-verify-fresh --command oracle "" 0)

  run_test "oracle_mapping" '"command":"oracle"' "$result"

  cleanup_tmpdir "$tmpdir"
}

# Test: watch command mapping
test_watch_mapping() {
  local tmpdir=$(setup_tmpdir)

  local result
  result=$(WATCH_DIR="$tmpdir" bash "$UTILS_SCRIPT" session-verify-fresh --command watch "" 0)

  run_test "watch_mapping" '"command":"watch"' "$result"

  cleanup_tmpdir "$tmpdir"
}

# Test: swarm command mapping
test_swarm_mapping() {
  local tmpdir=$(setup_tmpdir)

  local result
  result=$(SWARM_DIR="$tmpdir" bash "$UTILS_SCRIPT" session-verify-fresh --command swarm "" 0)

  run_test "swarm_mapping" '"command":"swarm"' "$result"

  cleanup_tmpdir "$tmpdir"
}

# Test: unknown command error
test_unknown_command() {
  local result
  result=$(bash "$UTILS_SCRIPT" session-verify-fresh --command unknown "" 0 2>&1 || true)

  run_test "unknown_command" 'Unknown command' "$result"
}

# Test: protected command init should fail
test_protected_init() {
  local result
  result=$(bash "$UTILS_SCRIPT" session-clear --command init 2>&1 || true)

  run_test "protected_init" 'protected' "$result"
}

# Test: protected command seal should fail
test_protected_seal() {
  local result
  result=$(bash "$UTILS_SCRIPT" session-clear --command seal 2>&1 || true)

  run_test "protected_seal" 'protected' "$result"
}

# Test: protected command entomb should fail
test_protected_entomb() {
  local result
  result=$(bash "$UTILS_SCRIPT" session-clear --command entomb 2>&1 || true)

  run_test "protected_entomb" 'protected' "$result"
}

# Test: backward compatibility survey-verify-fresh
test_backward_compat_verify() {
  local tmpdir=$(setup_tmpdir)

  local result
  result=$(SURVEY_DIR="$tmpdir" bash "$UTILS_SCRIPT" survey-verify-fresh "" 0)

  # Backward compat wrapper should work and return ok:true when no stale files
  run_test "backward_compat_verify" '"ok":true' "$result"

  cleanup_tmpdir "$tmpdir"
}

# Test: backward compatibility survey-clear
test_backward_compat_clear() {
  local tmpdir=$(setup_tmpdir)

  echo "test" > "$tmpdir/PROVISIONS.md"

  local result
  result=$(SURVEY_DIR="$tmpdir" bash "$UTILS_SCRIPT" survey-clear --dry-run)

  run_test "backward_compat_clear" '"dry_run":true' "$result"

  cleanup_tmpdir "$tmpdir"
}

# Test: JSON output with empty arrays
test_empty_arrays() {
  local tmpdir=$(setup_tmpdir)

  # Create all required files fresh
  local start_time=$(date +%s)
  for doc in PROVISIONS.md TRAILS.md BLUEPRINT.md CHAMBERS.md DISCIPLINES.md SENTINEL-PROTOCOLS.md PATHOGENS.md; do
    echo "test" > "$tmpdir/$doc"
  done

  local result
  result=$(SURVEY_DIR="$tmpdir" bash "$UTILS_SCRIPT" session-verify-fresh --command survey "" "$start_time")

  # Verify no empty string in arrays (the critical bug fix)
  if [[ "$result" == *'""'* ]]; then
    run_test "empty_arrays_no_empty_strings" "no empty strings" "found empty strings in JSON"
  else
    run_test "empty_arrays_no_empty_strings" "no empty strings" "no empty strings"
  fi

  cleanup_tmpdir "$tmpdir"
}

# Test: cross-platform stat works
test_cross_platform_stat() {
  local tmpdir=$(setup_tmpdir)

  echo "test" > "$tmpdir/PROVISIONS.md"
  local start_time=$(date +%s)

  local result
  result=$(SURVEY_DIR="$tmpdir" bash "$UTILS_SCRIPT" session-verify-fresh --command survey "" "$start_time")

  # Just verify it doesn't error - the stat command worked
  run_test "cross_platform_stat" '"total_lines":' "$result"

  cleanup_tmpdir "$tmpdir"
}

# Run all tests
echo "========================================="
echo "Session Freshness Detection Test Suite"
echo "========================================="
echo ""

test_verify_fresh_missing
test_verify_fresh_stale
test_verify_fresh_fresh
test_verify_fresh_force
test_clear_dry_run
test_clear_actual
test_oracle_mapping
test_watch_mapping
test_swarm_mapping
test_unknown_command
test_protected_init
test_protected_seal
test_protected_entomb
test_backward_compat_verify
test_backward_compat_clear
test_empty_arrays
test_cross_platform_stat

echo ""
echo "========================================="
echo "Test Summary"
echo "========================================="
echo -e "Tests run:   $TESTS_RUN"
echo -e "${GREEN}Passed:      $TESTS_PASSED${NC}"
if [[ $TESTS_FAILED -gt 0 ]]; then
  echo -e "${RED}Failed:      $TESTS_FAILED${NC}"
fi

if [[ $TESTS_FAILED -eq 0 ]]; then
  echo -e "\n${GREEN}All tests passed!${NC}"
  exit 0
else
  echo -e "\n${RED}Some tests failed!${NC}"
  exit 1
fi
