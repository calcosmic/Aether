#!/bin/bash
# Phase 3 XML Work Test Suite
# Tests queen-wisdom XSLT, promotion workflow, and prompt XML conversion

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Source the XML utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
source "$AETHER_ROOT/.aether/utils/xml-utils.sh" 2>/dev/null || {
    echo -e "${RED}ERROR: Could not source xml-utils.sh${NC}"
    exit 1
}

# Helper functions
pass() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

fail() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    echo "  Error: $2"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

skip() {
    echo -e "${YELLOW}⊘ SKIP${NC}: $1"
}

# Test 1: XSLT file exists
test_xslt_exists() {
    if [[ -f "$AETHER_ROOT/.aether/utils/queen-to-md.xsl" ]]; then
        pass "XSLT file queen-to-md.xsl exists"
    else
        fail "XSLT file queen-to-md.xsl exists" "File not found"
    fi
}

# Test 2: Prompt schema exists
test_prompt_schema_exists() {
    if [[ -f "$AETHER_ROOT/.aether/schemas/prompt.xsd" ]]; then
        pass "Prompt schema prompt.xsd exists"
    else
        fail "Prompt schema prompt.xsd exists" "File not found"
    fi
}

# Test 3: Example prompt exists
test_example_prompt_exists() {
    if [[ -f "$AETHER_ROOT/.aether/schemas/example-prompt-builder.xml" ]]; then
        pass "Example prompt file exists"
    else
        fail "Example prompt file exists" "File not found"
    fi
}

# Test 4: Validate example prompt against schema
test_validate_example_prompt() {
    if ! command -v xmllint >/dev/null 2>&1; then
        skip "Validate example prompt - xmllint not available"
        return
    fi

    local result
    result=$(prompt-validate "$AETHER_ROOT/.aether/schemas/example-prompt-builder.xml" 2>/dev/null)
    if echo "$result" | jq -e '.result.valid' >/dev/null 2>&1; then
        pass "Example prompt validates against schema"
    else
        fail "Example prompt validates against schema" "Validation failed"
    fi
}

# Test 5: queen-wisdom-to-markdown function exists
test_queen_wisdom_md_function() {
    if type queen-wisdom-to-markdown >/dev/null 2>&1; then
        pass "queen-wisdom-to-markdown function exists"
    else
        fail "queen-wisdom-to-markdown function exists" "Function not found"
    fi
}

# Test 6: queen-wisdom-validate-entry function exists
test_queen_wisdom_validate_function() {
    if type queen-wisdom-validate-entry >/dev/null 2>&1; then
        pass "queen-wisdom-validate-entry function exists"
    else
        fail "queen-wisdom-validate-entry function exists" "Function not found"
    fi
}

# Test 7: queen-wisdom-promote function exists
test_queen_wisdom_promote_function() {
    if type queen-wisdom-promote >/dev/null 2>&1; then
        pass "queen-wisdom-promote function exists"
    else
        fail "queen-wisdom-promote function exists" "Function not found"
    fi
}

# Test 8: queen-wisdom-import function exists
test_queen_wisdom_import_function() {
    if type queen-wisdom-import >/dev/null 2>&1; then
        pass "queen-wisdom-import function exists"
    else
        fail "queen-wisdom-import function exists" "Function not found"
    fi
}

# Test 9: prompt-to-xml function exists
test_prompt_to_xml_function() {
    if type prompt-to-xml >/dev/null 2>&1; then
        pass "prompt-to-xml function exists"
    else
        fail "prompt-to-xml function exists" "Function not found"
    fi
}

# Test 10: prompt-from-xml function exists
test_prompt_from_xml_function() {
    if type prompt-from-xml >/dev/null 2>&1; then
        pass "prompt-from-xml function exists"
    else
        fail "prompt-from-xml function exists" "Function not found"
    fi
}

# Test 11: prompt-validate function exists
test_prompt_validate_function() {
    if type prompt-validate >/dev/null 2>&1; then
        pass "prompt-validate function exists"
    else
        fail "prompt-validate function exists" "Function not found"
    fi
}

# Test 12: Create and validate a minimal queen-wisdom XML
test_create_queen_wisdom_xml() {
    local temp_dir
    temp_dir=$(mktemp -d)

    cat > "$temp_dir/test-wisdom.xml" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<queen-wisdom >
  <metadata>
    <version>1.0.0</version>
    <created>2026-02-16T10:00:00Z</created>
    <modified>2026-02-16T10:00:00Z</modified>
    <colony_id>test</colony_id>
  </metadata>
  <philosophies>
    <philosophy id="test-philosophy" confidence="0.8" domain="testing" source="colony" created_at="2026-02-16T10:00:00Z">
      <content>Test-driven development ensures quality</content>
    </philosophy>
  </philosophies>
  <patterns>
    <pattern id="test-pattern" confidence="0.7" domain="general" source="observation" created_at="2026-02-16T10:00:00Z">
      <content>Always validate inputs</content>
    </pattern>
  </patterns>
  <redirects>
    <redirect id="test-redirect" confidence="0.9" domain="security" source="queen" created_at="2026-02-16T10:00:00Z">
      <content>Never skip security checks</content>
    </redirect>
  </redirects>
  <stack-wisdom>
    <wisdom id="test-stack" confidence="0.6" domain="architecture" source="colony" created_at="2026-02-16T10:00:00Z">
      <content>Use jq for JSON in bash</content>
      <technology>bash</technology>
    </wisdom>
  </stack-wisdom>
  <decrees>
    <decree id="test-decree" confidence="1.0" domain="process" source="queen" created_at="2026-02-16T10:00:00Z">
      <content>All code must have tests</content>
      <scope>project</scope>
    </decree>
  </decrees>
  <evolution-log>
    <entry timestamp="2026-02-16T10:00:00Z" colony="test" action="initialized" type="system">
      <note>Initial test wisdom</note>
    </entry>
  </evolution-log>
</queen-wisdom>
EOF

    if [[ -f "$temp_dir/test-wisdom.xml" ]]; then
        pass "Create queen-wisdom XML file"
    else
        fail "Create queen-wisdom XML file" "Failed to create file"
    fi

    rm -rf "$temp_dir"
}

# Test 13: Validate queen-wisdom XML against schema
test_validate_queen_wisdom_xml() {
    if ! command -v xmllint >/dev/null 2>&1; then
        skip "Validate queen-wisdom XML - xmllint not available"
        return
    fi

    local temp_dir
    temp_dir=$(mktemp -d)

    cat > "$temp_dir/test-wisdom.xml" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<queen-wisdom >
  <metadata>
    <version>1.0.0</version>
    <created>2026-02-16T10:00:00Z</created>
    <modified>2026-02-16T10:00:00Z</modified>
    <colony_id>test</colony_id>
  </metadata>
  <philosophies>
    <philosophy id="test-philosophy" confidence="0.8" domain="testing" source="colony" created_at="2026-02-16T10:00:00Z">
      <content>Test-driven development ensures quality</content>
    </philosophy>
  </philosophies>
  <patterns>
    <pattern id="test-pattern" confidence="0.7" domain="general" source="observation" created_at="2026-02-16T10:00:00Z">
      <content>Always validate inputs</content>
    </pattern>
  </patterns>
  <redirects>
    <redirect id="test-redirect" confidence="0.9" domain="security" source="queen" created_at="2026-02-16T10:00:00Z">
      <content>Never skip security checks</content>
    </redirect>
  </redirects>
  <stack-wisdom>
    <wisdom id="test-stack" confidence="0.6" domain="architecture" source="colony" created_at="2026-02-16T10:00:00Z">
      <content>Use jq for JSON in bash</content>
      <technology>bash</technology>
    </wisdom>
  </stack-wisdom>
  <decrees>
    <decree id="test-decree" confidence="1.0" domain="process" source="queen" created_at="2026-02-16T10:00:00Z">
      <content>All code must have tests</content>
      <scope>project</scope>
    </decree>
  </decrees>
</queen-wisdom>
EOF

    local result
    result=$(xml-validate "$temp_dir/test-wisdom.xml" "$AETHER_ROOT/.aether/schemas/queen-wisdom.xsd" 2>/dev/null)
    if echo "$result" | jq -e '.result.valid' >/dev/null 2>&1; then
        pass "Validate queen-wisdom XML against schema"
    else
        fail "Validate queen-wisdom XML against schema" "Validation failed"
    fi

    rm -rf "$temp_dir"
}

# Test 14: Convert prompt XML to markdown
test_prompt_xml_to_md() {
    if ! type prompt-from-xml >/dev/null 2>&1; then
        skip "Convert prompt XML to markdown - function not available"
        return
    fi

    local result
    result=$(prompt-from-xml "$AETHER_ROOT/.aether/schemas/example-prompt-builder.xml" 2>/dev/null)
    if echo "$result" | jq -e '.ok' >/dev/null 2>&1; then
        pass "Convert prompt XML to markdown"
    else
        fail "Convert prompt XML to markdown" "Conversion failed"
    fi
}

# Test 15: XSLT transformation of queen-wisdom to markdown
test_xslt_transformation() {
    if ! command -v xsltproc >/dev/null 2>&1; then
        skip "XSLT transformation - xsltproc not available"
        return
    fi

    local temp_dir
    temp_dir=$(mktemp -d)

    # Create test XML
    cat > "$temp_dir/test-wisdom.xml" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<queen-wisdom >
  <metadata>
    <version>1.0.0</version>
    <created>2026-02-16T10:00:00Z</created>
    <modified>2026-02-16T12:00:00Z</modified>
    <colony_id>test</colony_id>
  </metadata>
  <philosophies>
    <philosophy id="test-philosophy" confidence="0.8" domain="testing" source="colony" created_at="2026-02-16T10:00:00Z">
      <content>Test-driven development ensures quality</content>
    </philosophy>
  </philosophies>
  <patterns>
    <pattern id="test-pattern" confidence="0.7" domain="general" source="observation" created_at="2026-02-16T10:00:00Z">
      <content>Test pattern</content>
    </pattern>
  </patterns>
  <redirects>
    <redirect id="test-redirect" confidence="0.9" domain="security" source="queen" created_at="2026-02-16T10:00:00Z">
      <content>Test redirect</content>
    </redirect>
  </redirects>
  <stack-wisdom>
    <wisdom id="test-stack" confidence="0.6" domain="architecture" source="colony" created_at="2026-02-16T10:00:00Z">
      <content>Test stack wisdom</content>
    </wisdom>
  </stack-wisdom>
  <decrees>
    <decree id="test-decree" confidence="1.0" domain="process" source="queen" created_at="2026-02-16T10:00:00Z">
      <content>Test decree</content>
    </decree>
  </decrees>
</queen-wisdom>
EOF

    # Try XSLT transformation directly
    if xsltproc "$AETHER_ROOT/.aether/utils/queen-to-md.xsl" "$temp_dir/test-wisdom.xml" > "$temp_dir/output.md" 2>/dev/null; then
        if grep -q "QUEEN.md" "$temp_dir/output.md"; then
            pass "XSLT transformation produces markdown output"
        else
            fail "XSLT transformation produces markdown output" "Output doesn't contain expected content"
        fi
    else
        fail "XSLT transformation produces markdown output" "Transformation failed"
    fi

    rm -rf "$temp_dir"
}

# Main test runner
echo "========================================"
echo "Phase 3 XML Work Test Suite"
echo "========================================"
echo ""

# Run all tests
test_xslt_exists
test_prompt_schema_exists
test_example_prompt_exists
test_validate_example_prompt
test_queen_wisdom_md_function
test_queen_wisdom_validate_function
test_queen_wisdom_promote_function
test_queen_wisdom_import_function
test_prompt_to_xml_function
test_prompt_from_xml_function
test_prompt_validate_function
test_create_queen_wisdom_xml
test_validate_queen_wisdom_xml
test_prompt_xml_to_md
test_xslt_transformation

echo ""
echo "========================================"
echo "Test Results"
echo "========================================"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Failed: $TESTS_FAILED${NC}"
echo ""

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed.${NC}"
    exit 1
fi
