#!/bin/bash
#
# Test XML Schema Validation
#
# Tests for shared types schema and schema imports
#
# Note: Not using 'set -e' because we handle failures with pass/fail functions

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
SCHEMAS_DIR="$AETHER_DIR/.aether/schemas"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
pass() {
    echo -e "${GREEN}✓${NC} $1"
    ((TESTS_PASSED++))
}

fail() {
    echo -e "${RED}✗${NC} $1"
    ((TESTS_FAILED++))
}

info() {
    echo -e "${YELLOW}→${NC} $1"
}

# Check if xmllint is available
if ! command -v xmllint &> /dev/null; then
    echo "Error: xmllint is required but not installed"
    exit 1
fi

info "Testing Aether XML Schemas..."
echo

# ============================================================
# Test 1: Shared types schema is well-formed
# ============================================================
info "Test 1: aether-types.xsd is well-formed XML"
if xmllint --noout "$SCHEMAS_DIR/aether-types.xsd" 2>/dev/null; then
    pass "aether-types.xsd is well-formed"
else
    fail "aether-types.xsd has XML syntax errors"
fi

# ============================================================
# Test 2: Pheromone schema imports shared types
# ============================================================
info "Test 2: pheromone.xsd imports shared types"
if grep -q 'schemaLocation="aether-types.xsd"' "$SCHEMAS_DIR/pheromone.xsd"; then
    pass "pheromone.xsd imports aether-types.xsd"
else
    fail "pheromone.xsd missing import for aether-types.xsd"
fi

# ============================================================
# Test 3: Worker-priming schema imports shared types
# ============================================================
info "Test 3: worker-priming.xsd imports shared types"
if grep -q 'schemaLocation="aether-types.xsd"' "$SCHEMAS_DIR/worker-priming.xsd"; then
    pass "worker-priming.xsd imports aether-types.xsd"
else
    fail "worker-priming.xsd missing import for aether-types.xsd"
fi

# ============================================================
# Test 4: Prompt schema imports shared types
# ============================================================
info "Test 4: prompt.xsd imports shared types"
if grep -q 'schemaLocation="aether-types.xsd"' "$SCHEMAS_DIR/prompt.xsd"; then
    pass "prompt.xsd imports aether-types.xsd"
else
    fail "prompt.xsd missing import for aether-types.xsd"
fi

# ============================================================
# Test 5: CasteEnum is defined in shared types
# ============================================================
info "Test 5: CasteEnum defined in shared types"
if grep -q 'name="CasteEnum"' "$SCHEMAS_DIR/aether-types.xsd"; then
    pass "CasteEnum found in aether-types.xsd"
else
    fail "CasteEnum not found in aether-types.xsd"
fi

# ============================================================
# Test 6: All 22 castes are defined
# ============================================================
info "Test 6: All 22 castes defined in CasteEnum"
EXPECTED_CASTES=(
    "builder" "watcher" "scout" "chaos" "oracle"
    "architect" "prime" "colonizer" "route_setter" "archaeologist"
    "ambassador" "auditor" "chronicler" "gatekeeper" "guardian"
    "includer" "keeper" "measurer" "probe" "sage"
    "tracker" "weaver"
)
ALL_CASTES_FOUND=true
for caste in "${EXPECTED_CASTES[@]}"; do
    if ! grep -q "value=\"$caste\"" "$SCHEMAS_DIR/aether-types.xsd"; then
        fail "Missing caste: $caste"
        ALL_CASTES_FOUND=false
    fi
done
if $ALL_CASTES_FOUND; then
    pass "All 22 castes found in aether-types.xsd"
fi

# ============================================================
# Test 7: Pheromone uses types:CasteEnum
# ============================================================
info "Test 7: pheromone.xsd uses types:CasteEnum"
if grep -q 'type="types:CasteEnum"' "$SCHEMAS_DIR/pheromone.xsd"; then
    pass "pheromone.xsd references types:CasteEnum"
else
    fail "pheromone.xsd does not reference types:CasteEnum"
fi

# ============================================================
# Test 8: Worker-priming uses types:CasteEnum
# ============================================================
info "Test 8: worker-priming.xsd uses types:CasteEnum"
if grep -q 'type="types:CasteEnum"' "$SCHEMAS_DIR/worker-priming.xsd"; then
    pass "worker-priming.xsd references types:CasteEnum"
else
    fail "worker-priming.xsd does not reference types:CasteEnum"
fi

# ============================================================
# Test 9: Prompt uses types:CasteEnum
# ============================================================
info "Test 9: prompt.xsd uses types:CasteEnum"
if grep -q 'type="types:CasteEnum"' "$SCHEMAS_DIR/prompt.xsd"; then
    pass "prompt.xsd references types:CasteEnum"
else
    fail "prompt.xsd does not reference types:CasteEnum"
fi

# ============================================================
# Test 10: VersionType is defined in shared types
# ============================================================
info "Test 10: VersionType defined in shared types"
if grep -q 'name="VersionType"' "$SCHEMAS_DIR/aether-types.xsd"; then
    pass "VersionType found in aether-types.xsd"
else
    fail "VersionType not found in aether-types.xsd"
fi

# ============================================================
# Test 11: TimestampType is defined in shared types
# ============================================================
info "Test 11: TimestampType defined in shared types"
if grep -q 'name="TimestampType"' "$SCHEMAS_DIR/aether-types.xsd"; then
    pass "TimestampType found in aether-types.xsd"
else
    fail "TimestampType not found in aether-types.xsd"
fi

# ============================================================
# Test 12: PriorityType is defined in shared types
# ============================================================
info "Test 12: PriorityType defined in shared types"
if grep -q 'name="PriorityType"' "$SCHEMAS_DIR/aether-types.xsd"; then
    pass "PriorityType found in aether-types.xsd"
else
    fail "PriorityType not found in aether-types.xsd"
fi

# ============================================================
# Test 13: Priority levels are correct
# ============================================================
info "Test 13: Priority levels in shared types"
PRIORITIES=("critical" "high" "normal" "low")
ALL_PRIORITIES_FOUND=true
for priority in "${PRIORITIES[@]}"; do
    if ! grep -A 15 'name="PriorityType"' "$SCHEMAS_DIR/aether-types.xsd" | grep -q "value=\"$priority\""; then
        fail "Missing priority: $priority"
        ALL_PRIORITIES_FOUND=false
    fi
done
if $ALL_PRIORITIES_FOUND; then
    pass "All 4 priority levels found"
fi

# ============================================================
# Test 14: PheromoneTypeEnum includes extended types
# ============================================================
info "Test 14: PheromoneTypeEnum has extended signal types"
if grep -q 'value="PHILOSOPHY"' "$SCHEMAS_DIR/aether-types.xsd"; then
    pass "PheromoneTypeEnum includes extended types (PHILOSOPHY, etc.)"
else
    fail "PheromoneTypeEnum missing extended types"
fi

# ============================================================
# Test 15: No local CasteEnum in pheromone.xsd
# ============================================================
info "Test 15: pheromone.xsd has no local CasteEnum"
if grep -q 'name="CasteEnum"' "$SCHEMAS_DIR/pheromone.xsd"; then
    fail "pheromone.xsd still has local CasteEnum (should use shared)"
else
    pass "pheromone.xsd uses shared CasteEnum"
fi

# ============================================================
# Test 16: No local casteType in worker-priming.xsd
# ============================================================
info "Test 16: worker-priming.xsd has no local casteType"
if grep -q 'name="casteType"' "$SCHEMAS_DIR/worker-priming.xsd"; then
    fail "worker-priming.xsd still has local casteType (should use shared)"
else
    pass "worker-priming.xsd uses shared CasteEnum"
fi

# ============================================================
# Test 17: No local casteType in prompt.xsd
# ============================================================
info "Test 17: prompt.xsd has no local casteType"
if grep -q 'name="casteType"' "$SCHEMAS_DIR/prompt.xsd"; then
    fail "prompt.xsd still has local casteType (should use shared)"
else
    pass "prompt.xsd uses shared CasteEnum"
fi

# ============================================================
# Test 18: WeightType is in shared types
# ============================================================
info "Test 18: WeightType defined in shared types"
if grep -q 'name="WeightType"' "$SCHEMAS_DIR/aether-types.xsd"; then
    pass "WeightType found in aether-types.xsd"
else
    fail "WeightType not found in aether-types.xsd"
fi

# ============================================================
# Test 19: MatchEnum is in shared types
# ============================================================
info "Test 19: MatchEnum defined in shared types"
if grep -q 'name="MatchEnum"' "$SCHEMAS_DIR/aether-types.xsd"; then
    pass "MatchEnum found in aether-types.xsd"
else
    fail "MatchEnum not found in aether-types.xsd"
fi

# ============================================================
# Test 20: SourceTypeEnum is in shared types
# ============================================================
info "Test 20: SourceTypeEnum defined in shared types"
if grep -q 'name="SourceTypeEnum"' "$SCHEMAS_DIR/aether-types.xsd"; then
    pass "SourceTypeEnum found in aether-types.xsd"
else
    fail "SourceTypeEnum not found in aether-types.xsd"
fi

# ============================================================
# Test 21: DataFormatEnum is in shared types
# ============================================================
info "Test 21: DataFormatEnum defined in shared types"
if grep -q 'name="DataFormatEnum"' "$SCHEMAS_DIR/aether-types.xsd"; then
    pass "DataFormatEnum found in aether-types.xsd"
else
    fail "DataFormatEnum not found in aether-types.xsd"
fi

# ============================================================
# Test 22: IdentifierType is in shared types
# ============================================================
info "Test 22: IdentifierType defined in shared types"
if grep -q 'name="IdentifierType"' "$SCHEMAS_DIR/aether-types.xsd"; then
    pass "IdentifierType found in aether-types.xsd"
else
    fail "IdentifierType not found in aether-types.xsd"
fi

# ============================================================
# Test 23: WorkerIdType is in shared types
# ============================================================
info "Test 23: WorkerIdType defined in shared types"
if grep -q 'name="WorkerIdType"' "$SCHEMAS_DIR/aether-types.xsd"; then
    pass "WorkerIdType found in aether-types.xsd"
else
    fail "WorkerIdType not found in aether-types.xsd"
fi

# ============================================================
# Test 24: WisdomIdType is in shared types
# ============================================================
info "Test 24: WisdomIdType defined in shared types"
if grep -q 'name="WisdomIdType"' "$SCHEMAS_DIR/aether-types.xsd"; then
    pass "WisdomIdType found in aether-types.xsd"
else
    fail "WisdomIdType not found in aether-types.xsd"
fi

# ============================================================
# Test 25: ConfidenceType is in shared types
# ============================================================
info "Test 25: ConfidenceType defined in shared types"
if grep -q 'name="ConfidenceType"' "$SCHEMAS_DIR/aether-types.xsd"; then
    pass "ConfidenceType found in aether-types.xsd"
else
    fail "ConfidenceType not found in aether-types.xsd"
fi

echo
echo "========================================"
echo "Schema Tests Complete"
echo "========================================"
echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"
echo

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
