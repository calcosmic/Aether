#!/bin/bash
# Test suite for oracle.sh AETHER_ROOT derivation
# Task 1.2 — validates that AETHER_ROOT resolves to repo root from .aether/utils/oracle/

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ORACLE_SCRIPT="$AETHER_ROOT/.aether/utils/oracle/oracle.sh"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

run_test() {
  local name="$1"
  local expected="$2"
  local actual="$3"

  TESTS_RUN=$((TESTS_RUN + 1))

  if [[ "$actual" == "$expected" ]]; then
    echo -e "${GREEN}PASS${NC}: $name"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    return 0
  else
    echo -e "${RED}FAIL${NC}: $name"
    echo "  Expected: $expected"
    echo "  Actual:   $actual"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    return 1
  fi
}

echo "=== Oracle AETHER_ROOT Derivation Tests ==="
echo ""

# ── Test 1: AETHER_ROOT line resolves to repo root ──
# Extract and evaluate the AETHER_ROOT derivation from oracle.sh
# We simulate what oracle.sh does: set SCRIPT_DIR to its location, then derive AETHER_ROOT
ORACLE_DIR="$(cd "$(dirname "$ORACLE_SCRIPT")" && pwd)"

# Get the AETHER_ROOT as oracle.sh would compute it
# We extract line 16 logic: AETHER_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
# But we test by sourcing just the derivation, not the whole script
DERIVED_ROOT="$(cd "$ORACLE_DIR/../../.." && pwd)"

run_test \
  "AETHER_ROOT (3 levels up) resolves to repo root" \
  "$AETHER_ROOT" \
  "$DERIVED_ROOT"

# ── Test 2: oracle.sh line 16 uses three-level parent traversal ──
# Grep the actual line to confirm the pattern
LINE_16=$(sed -n '16p' "$ORACLE_SCRIPT")
if echo "$LINE_16" | grep -q '\.\./\.\./\.\.'; then
  TESTS_RUN=$((TESTS_RUN + 1))
  TESTS_PASSED=$((TESTS_PASSED + 1))
  echo -e "${GREEN}PASS${NC}: Line 16 contains ../../.. (three-level traversal)"
else
  TESTS_RUN=$((TESTS_RUN + 1))
  TESTS_FAILED=$((TESTS_FAILED + 1))
  echo -e "${RED}FAIL${NC}: Line 16 should contain ../../.. (three-level traversal)"
  echo "  Actual line 16: $LINE_16"
fi

# ── Test 3: AETHER_ROOT references use .aether/ prefix ──
# All AETHER_ROOT file references should use $AETHER_ROOT/.aether/ pattern
# (since AETHER_ROOT is the repo root, not .aether/ itself)
BAD_REFS=$(grep 'AETHER_ROOT' "$ORACLE_SCRIPT" | grep -v '\.aether' | grep -v '^AETHER_ROOT=' | grep -v 'read_steering_signals' || true)
if [[ -z "$BAD_REFS" ]]; then
  TESTS_RUN=$((TESTS_RUN + 1))
  TESTS_PASSED=$((TESTS_PASSED + 1))
  echo -e "${GREEN}PASS${NC}: All AETHER_ROOT file refs use .aether/ prefix"
else
  TESTS_RUN=$((TESTS_RUN + 1))
  TESTS_FAILED=$((TESTS_FAILED + 1))
  echo -e "${RED}FAIL${NC}: Found AETHER_ROOT refs without .aether/ prefix"
  echo "  Lines: $BAD_REFS"
fi

# ── Test 4: Verify atomic-write.sh path resolves to existing file ──
ATOMIC_PATH="$DERIVED_ROOT/.aether/utils/atomic-write.sh"
if [[ -f "$ATOMIC_PATH" ]]; then
  TESTS_RUN=$((TESTS_RUN + 1))
  TESTS_PASSED=$((TESTS_PASSED + 1))
  echo -e "${GREEN}PASS${NC}: atomic-write.sh resolves to existing file"
else
  TESTS_RUN=$((TESTS_RUN + 1))
  TESTS_FAILED=$((TESTS_FAILED + 1))
  echo -e "${RED}FAIL${NC}: atomic-write.sh not found at $ATOMIC_PATH"
fi

# ── Test 5: Verify aether-utils.sh path resolves from derived root ──
UTILS_PATH="$DERIVED_ROOT/.aether/aether-utils.sh"
if [[ -f "$UTILS_PATH" ]]; then
  TESTS_RUN=$((TESTS_RUN + 1))
  TESTS_PASSED=$((TESTS_PASSED + 1))
  echo -e "${GREEN}PASS${NC}: aether-utils.sh resolves to existing file"
else
  TESTS_RUN=$((TESTS_RUN + 1))
  TESTS_FAILED=$((TESTS_FAILED + 1))
  echo -e "${RED}FAIL${NC}: aether-utils.sh not found at $UTILS_PATH"
fi

echo ""
echo "=== Results: $TESTS_PASSED/$TESTS_RUN passed, $TESTS_FAILED failed ==="

if [[ $TESTS_FAILED -gt 0 ]]; then
  exit 1
fi
exit 0
