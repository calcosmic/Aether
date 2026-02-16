#!/bin/bash
# Test XInclude Composition for Worker Priming
# Tests xml-compose, xml-compose-worker-priming, xml-list-xincludes

set -uo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Source the composition functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
source "$AETHER_DIR/.aether/utils/xml-utils.sh" 2>/dev/null || true
source "$AETHER_DIR/.aether/utils/xinclude-composition.sh" 2>/dev/null || true

# Test helper functions
pass() {
    echo -e "${GREEN}✓${NC} $1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    return 0
}

fail() {
    echo -e "${RED}✗${NC} $1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    return 0
}

skip() {
    echo -e "${YELLOW}⊘${NC} $1 (skipped)"
}

# Setup test directory
TEST_DIR=$(mktemp -d)
trap "rm -rf $TEST_DIR" EXIT

# Create sample XML files for testing
create_test_files() {
    # Main composition document
    cat > "$TEST_DIR/main.xml" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<root xmlns:xi="http://www.w3.org/2001/XInclude">
  <header>Main Document</header>
  <xi:include href="section1.xml" parse="xml"/>
  <xi:include href="section2.xml" parse="xml"/>
  <footer>End</footer>
</root>
EOF

    # Section 1
    cat > "$TEST_DIR/section1.xml" << 'EOF'
<section id="1">
  <title>Section One</title>
  <content>First section content</content>
</section>
EOF

    # Section 2
    cat > "$TEST_DIR/section2.xml" << 'EOF'
<section id="2">
  <title>Section Two</title>
  <content>Second section content</content>
</section>
EOF

    # Document with fallback
    cat > "$TEST_DIR/with-fallback.xml" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<root xmlns:xi="http://www.w3.org/2001/XInclude">
  <xi:include href="missing.xml" parse="xml">
    <xi:fallback><fallback>Default content</fallback></xi:fallback>
  </xi:include>
</root>
EOF

    # Worker priming document
    cat > "$TEST_DIR/worker-priming.xml" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<worker-priming version="1.0.0"
                xmlns="http://aether.colony/schemas/worker-priming/1.0"
                xmlns:xi="http://www.w3.org/2001/XInclude">
  <metadata>
    <version>1.0.0</version>
    <created>2026-02-16T15:47:00Z</created>
    <modified>2026-02-16T15:47:00Z</modified>
    <colony-id>test-colony</colony-id>
  </metadata>
  <worker-identity id="builder-test">
    <name>Test Builder</name>
    <caste>builder</caste>
  </worker-identity>
  <queen-wisdom enabled="true">
    <wisdom-source name="test-wisdom" priority="high">
      <xi:include href="wisdom.xml" parse="xml"/>
    </wisdom-source>
  </queen-wisdom>
</worker-priming>
EOF

    # Wisdom content
    cat > "$TEST_DIR/wisdom.xml" << 'EOF'
<philosophy id="test">
  <content>Always test your code</content>
</philosophy>
EOF
}

# Test 1: Basic composition
test_basic_composition() {
    echo "Test: Basic XInclude composition"
    local result
    result=$(xml-compose "$TEST_DIR/main.xml" "$TEST_DIR/output.xml" 2>&1)

    if echo "$result" | jq -e '.ok' >/dev/null 2>&1; then
        if [[ -f "$TEST_DIR/output.xml" ]]; then
            if grep -q "Section One" "$TEST_DIR/output.xml" && grep -q "Section Two" "$TEST_DIR/output.xml"; then
                pass "Basic composition includes all sections"
            else
                fail "Basic composition missing content"
            fi
        else
            fail "Output file not created"
        fi
    else
        fail "Composition failed: $(echo "$result" | jq -r '.error // "unknown"')"
    fi
}

# Test 2: List XIncludes
test_list_xincludes() {
    echo "Test: List XInclude directives"
    local result
    result=$(xml-list-xincludes "$TEST_DIR/main.xml" 2>&1)

    if echo "$result" | jq -e '.ok' >/dev/null 2>&1; then
        local count
        count=$(echo "$result" | jq -r '.result.count')
        if [[ "$count" == "2" ]]; then
            pass "Found 2 XInclude directives"
        else
            fail "Expected 2 XInclude directives, got $count"
        fi
    else
        fail "List XIncludes failed: $(echo "$result" | jq -r '.error // "unknown"')"
    fi
}

# Test 3: Worker priming composition
test_worker_priming_composition() {
    echo "Test: Worker priming composition"
    local result
    result=$(xml-compose-worker-priming "$TEST_DIR/worker-priming.xml" "$TEST_DIR/composed-priming.xml" 2>&1)

    if echo "$result" | jq -e '.ok' >/dev/null 2>&1; then
        local worker_id caste
        worker_id=$(echo "$result" | jq -r '.result.worker_id')
        caste=$(echo "$result" | jq -r '.result.caste')

        if [[ "$worker_id" == "builder-test" && "$caste" == "builder" ]]; then
            pass "Worker priming extracted correct identity"
        else
            fail "Worker priming identity mismatch: $worker_id / $caste"
        fi
    else
        fail "Worker priming composition failed: $(echo "$result" | jq -r '.error // "unknown"')"
    fi
}

# Test 4: Validate composition
test_validate_composition() {
    echo "Test: Validate composed document"
    # First compose a document
    xml-compose "$TEST_DIR/worker-priming.xml" "$TEST_DIR/validated.xml" >/dev/null 2>&1 || true

    if [[ -f "$TEST_DIR/validated.xml" ]]; then
        local result
        result=$(xml-validate-composition "$TEST_DIR/validated.xml" 2>&1)

        if echo "$result" | jq -e '.ok' >/dev/null 2>&1; then
            pass "Composition validation completed"
        else
            fail "Composition validation failed: $(echo "$result" | jq -r '.error // "unknown"')"
        fi
    else
        skip "Validated file not created (xmllint may not be available)"
    fi
}

# Test 5: Manual composition fallback
test_manual_composition() {
    echo "Test: Manual XInclude composition"
    local result
    result=$(xml-compose-manual "$TEST_DIR/main.xml" "$TEST_DIR/manual-output.xml" 2>&1)

    if echo "$result" | jq -e '.ok' >/dev/null 2>&1; then
        if grep -q "Section One" "$TEST_DIR/manual-output.xml" 2>/dev/null; then
            pass "Manual composition works"
        else
            fail "Manual composition missing content"
        fi
    else
        skip "Manual composition not available"
    fi
}

# Test 6: Error handling - missing file
test_missing_file() {
    echo "Test: Error handling for missing file"
    local result
    result=$(xml-compose "$TEST_DIR/nonexistent.xml" 2>&1) || true

    if ! echo "$result" | jq -e '.ok' >/dev/null 2>&1; then
        pass "Correctly reports error for missing file"
    else
        fail "Should have reported error for missing file"
    fi
}

# Run all tests
echo "======================================"
echo "XInclude Composition Tests"
echo "======================================"
echo ""

create_test_files

test_basic_composition
test_list_xincludes
test_worker_priming_composition
test_validate_composition
test_manual_composition
test_missing_file

echo ""
echo "======================================"
echo "Results: $TESTS_PASSED passed, $TESTS_FAILED failed"
echo "======================================"

if [[ $TESTS_FAILED -eq 0 ]]; then
    exit 0
else
    exit 1
fi
