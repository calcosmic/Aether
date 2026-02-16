#!/usr/bin/env bash
# XML Utilities Test Suite
# Tests for .aether/utils/xml-utils.sh
#
# Run with: bash tests/bash/test-xml-utils.sh

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
XML_UTILS_SOURCE="$PROJECT_ROOT/.aether/utils/xml-utils.sh"

# Source test helpers
source "$SCRIPT_DIR/test-helpers.sh"

# Source XML utilities
if [[ ! -f "$XML_UTILS_SOURCE" ]]; then
    log_error "xml-utils.sh not found at: $XML_UTILS_SOURCE"
    exit 1
fi
source "$XML_UTILS_SOURCE"

# ============================================================================
# Test: xml-detect-tools
# ============================================================================
test_xml_detect_tools() {
    local output
    output=$(xml-detect-tools 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        return 1
    fi

    # Verify expected fields (nested under .result)
    # Check field existence (has key) not value
    if ! echo "$output" | jq -e '.result | has("xmllint")' > /dev/null 2>&1; then
        test_fail "has 'xmllint' field in result" "field missing"
        return 1
    fi

    if ! echo "$output" | jq -e '.result | has("xmlstarlet")' > /dev/null 2>&1; then
        test_fail "has 'xmlstarlet' field in result" "field missing"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: xml-well-formed with valid XML
# ============================================================================
test_xml_well_formed_valid() {
    local tmp_dir
    tmp_dir=$(mktemp -d)

    # Create valid XML
    cat > "$tmp_dir/valid.xml" << 'EOF'
<?xml version="1.0"?>
<root>
  <item>Test</item>
</root>
EOF

    local output
    output=$(xml-well-formed "$tmp_dir/valid.xml" 2>&1)
    local exit_code=$?
    rm -rf "$tmp_dir"

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        return 1
    fi

    if ! echo "$output" | jq -e '.result.well_formed == true' > /dev/null 2>&1; then
        test_fail "well_formed: true" "well_formed: false"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: xml-well-formed with invalid XML
# ============================================================================
test_xml_well_formed_invalid() {
    local tmp_dir
    tmp_dir=$(mktemp -d)

    # Create invalid XML
    cat > "$tmp_dir/invalid.xml" << 'EOF'
<?xml version="1.0"?>
<root>
  <unclosed>
</root>
EOF

    local output
    output=$(xml-well-formed "$tmp_dir/invalid.xml" 2>&1)
    local exit_code=$?
    rm -rf "$tmp_dir"

    # Should still return 0 but with well_formed: false
    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        return 1
    fi

    if ! echo "$output" | jq -e '.result.well_formed == false' > /dev/null 2>&1; then
        test_fail "well_formed: false" "well_formed: true"
        return 1
    fi

    if ! echo "$output" | jq -e '.result.error != null' > /dev/null 2>&1; then
        test_fail "has error message" "error is null"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: xml-well-formed with missing file
# ============================================================================
test_xml_well_formed_missing() {
    local output
    local exit_code

    set +e
    output=$(xml-well-formed "/nonexistent/file.xml" 2>&1)
    exit_code=$?
    set -e

    # Should return non-zero exit code
    if [[ "$exit_code" -eq 0 ]]; then
        test_fail "non-zero exit code" "exit code 0"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON error" "invalid JSON: $output"
        return 1
    fi

    if ! assert_ok_false "$output"; then
        test_fail '{"ok":false}' "$output"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: xml-escape and xml-unescape
# ============================================================================
test_xml_escape_unescape() {
    local test_string='Test with <special> & "characters"'

    # Test escape
    local escaped_output
    escaped_output=$(xml-escape "$test_string" 2>&1)

    if ! assert_json_valid "$escaped_output"; then
        test_fail "valid JSON from escape" "invalid JSON"
        return 1
    fi

    local escaped
    escaped=$(echo "$escaped_output" | jq -r '.result')

    # Verify special chars are escaped
    if [[ "$escaped" != *"&lt;"* ]] || [[ "$escaped" != *"&gt;"* ]] || [[ "$escaped" != *"&amp;"* ]]; then
        test_fail "special chars escaped" "not properly escaped: $escaped"
        return 1
    fi

    # Test unescape
    local unescaped_output
    unescaped_output=$(xml-unescape "$escaped" 2>&1)

    if ! assert_json_valid "$unescaped_output"; then
        test_fail "valid JSON from unescape" "invalid JSON"
        return 1
    fi

    local unescaped
    unescaped=$(echo "$unescaped_output" | jq -r '.result')

    if [[ "$unescaped" != "$test_string" ]]; then
        test_fail "original string restored" "got: $unescaped"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: json-to-xml with valid JSON
# ============================================================================
test_json_to_xml_valid() {
    local tmp_dir
    tmp_dir=$(mktemp -d)

    # Create test JSON
    cat > "$tmp_dir/test.json" << 'EOF'
{
  "name": "Test Colony",
  "phase": 1,
  "active": true
}
EOF

    local output
    output=$(json-to-xml "$tmp_dir/test.json" "colony" 2>&1)
    local exit_code=$?
    rm -rf "$tmp_dir"

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        return 1
    fi

    # Verify XML content
    local xml_content
    xml_content=$(echo "$output" | jq -r '.result.xml')

    if [[ "$xml_content" != *"<colony>"* ]] || [[ "$xml_content" != *"</colony>"* ]]; then
        test_fail "XML has colony root element" "root element missing"
        return 1
    fi

    if [[ "$xml_content" != *"<name>"* ]]; then
        test_fail "XML has name element" "name element missing"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: json-to-xml with invalid JSON
# ============================================================================
test_json_to_xml_invalid() {
    local tmp_dir
    tmp_dir=$(mktemp -d)

    # Create invalid JSON
    echo '{"invalid json' > "$tmp_dir/invalid.json"

    local output
    local exit_code

    set +e
    output=$(json-to-xml "$tmp_dir/invalid.json" 2>&1)
    exit_code=$?
    set -e
    rm -rf "$tmp_dir"

    if [[ "$exit_code" -eq 0 ]]; then
        test_fail "non-zero exit code" "exit code 0"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON error" "invalid JSON: $output"
        return 1
    fi

    if ! assert_ok_false "$output"; then
        test_fail '{"ok":false}' "$output"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: xml-format
# ============================================================================
test_xml_format() {
    local tmp_dir
    tmp_dir=$(mktemp -d)

    # Create unformatted XML
    cat > "$tmp_dir/unformatted.xml" << 'EOF'
<?xml version="1.0"?>
<root><item>Test</item><item>Test2</item></root>
EOF

    local output
    output=$(xml-format "$tmp_dir/unformatted.xml" 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Check file was formatted (should have newlines)
    local formatted_content
    formatted_content=$(cat "$tmp_dir/unformatted.xml")

    if [[ "$formatted_content" != *$'\n'* ]]; then
        test_fail "formatted XML has newlines" "no newlines found"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: pheromone-to-xml
# ============================================================================
test_pheromone_to_xml() {
    local tmp_dir
    tmp_dir=$(mktemp -d)

    # Create pheromone JSON
    cat > "$tmp_dir/pheromone.json" << 'EOF'
{
  "signal": "focus",
  "priority": "high",
  "message": "Pay attention to authentication",
  "source": "queen"
}
EOF

    local output
    output=$(pheromone-to-xml "$tmp_dir/pheromone.json" 2>&1)
    local exit_code=$?
    rm -rf "$tmp_dir"

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        return 1
    fi

    # Verify XML structure
    local xml_content
    xml_content=$(echo "$output" | jq -r '.result.xml')

    if [[ "$xml_content" != *"<pheromones"* ]]; then
        test_fail "XML has pheromones root element" "pheromones element missing"
        return 1
    fi

    if [[ "$xml_content" != *"type=\"FOCUS\""* ]]; then
        test_fail "XML has signal with FOCUS type" "signal type missing"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: queen-wisdom-to-xml
# ============================================================================
test_queen_wisdom_to_xml() {
    local tmp_dir
    tmp_dir=$(mktemp -d)

    # Create queen wisdom JSON
    cat > "$tmp_dir/wisdom.json" << 'EOF'
{
  "directive": "Implement authentication",
  "patterns": ["JWT tokens", "Session management"],
  "constraints": ["No plaintext passwords", "Rate limiting required"]
}
EOF

    local output
    output=$(queen-wisdom-to-xml "$tmp_dir/wisdom.json" 2>&1)
    local exit_code=$?
    rm -rf "$tmp_dir"

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        return 1
    fi

    # Verify XML structure
    local xml_content
    xml_content=$(echo "$output" | jq -r '.result.xml')

    if [[ "$xml_content" != *"<queen-wisdom"* ]]; then
        test_fail "XML has queen-wisdom element" "queen-wisdom element missing"
        return 1
    fi

    if [[ "$xml_content" != *"<directive>"* ]]; then
        test_fail "XML has directive element" "directive element missing"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: registry-to-xml
# ============================================================================
test_registry_to_xml() {
    local tmp_dir
    tmp_dir=$(mktemp -d)

    # Create registry JSON
    cat > "$tmp_dir/registry.json" << 'EOF'
{
  "colonies": [
    {
      "id": "alpha",
      "name": "Alpha Colony",
      "status": "active",
      "location": "/repos/alpha"
    },
    {
      "id": "beta",
      "name": "Beta Colony",
      "status": "sealed",
      "location": "/repos/beta"
    }
  ]
}
EOF

    local output
    output=$(registry-to-xml "$tmp_dir/registry.json" 2>&1)
    local exit_code=$?
    rm -rf "$tmp_dir"

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        return 1
    fi

    # Verify XML structure
    local xml_content
    xml_content=$(echo "$output" | jq -r '.result.xml')

    if [[ "$xml_content" != *"<colony-registry"* ]]; then
        test_fail "XML has colony-registry element" "colony-registry element missing"
        return 1
    fi

    if [[ "$xml_content" != *'id="alpha"'* ]]; then
        test_fail "XML has alpha colony" "alpha colony missing"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: xml-validate (if xmllint available)
# ============================================================================
test_xml_validate() {
    # Skip if xmllint not available
    if [[ "$XMLLINT_AVAILABLE" != "true" ]]; then
        log_warn "xmllint not available, skipping validation test"
        return 0
    fi

    local tmp_dir
    tmp_dir=$(mktemp -d)

    # Create XSD schema
    cat > "$tmp_dir/schema.xsd" << 'EOF'
<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="xs:string"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>
EOF

    # Create valid XML
    cat > "$tmp_dir/valid.xml" << 'EOF'
<?xml version="1.0"?>
<root>
  <item>Test</item>
</root>
EOF

    local output
    output=$(xml-validate "$tmp_dir/valid.xml" "$tmp_dir/schema.xsd" 2>&1)
    local exit_code=$?
    rm -rf "$tmp_dir"

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        return 1
    fi

    if ! echo "$output" | jq -e '.result.valid == true' > /dev/null 2>&1; then
        test_fail "valid: true" "validation failed"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: xml-query (if xmlstarlet available)
# ============================================================================
test_xml_query() {
    # Skip if xmlstarlet not available
    if [[ "$XMLSTARLET_AVAILABLE" != "true" ]]; then
        log_warn "xmlstarlet not available, skipping query test"
        return 0
    fi

    local tmp_dir
    tmp_dir=$(mktemp -d)

    # Create test XML
    cat > "$tmp_dir/test.xml" << 'EOF'
<?xml version="1.0"?>
<colony>
  <worker id="1">
    <name>Alpha</name>
  </worker>
  <worker id="2">
    <name>Beta</name>
  </worker>
</colony>
EOF

    local output
    output=$(xml-query "$tmp_dir/test.xml" "//worker/name" 2>&1)
    local exit_code=$?
    rm -rf "$tmp_dir"

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        return 1
    fi

    # Verify matches
    local count
    count=$(echo "$output" | jq '.result.count')
    if [[ "$count" -ne 2 ]]; then
        test_fail "2 matches" "$count matches"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: pheromone-export
# ============================================================================
test_pheromone_export() {
    local tmp_dir
    tmp_dir=$(mktemp -d)

    # Create test pheromones.json
    cat > "$tmp_dir/pheromones.json" << 'EOF'
{
  "version": "1.0.0",
  "colony_id": "test-colony",
  "signals": [
    {
      "id": "sig_test_001",
      "type": "FOCUS",
      "priority": "normal",
      "source": "user",
      "created_at": "2026-02-16T10:00:00Z",
      "expires_at": "2026-02-17T10:00:00Z",
      "active": true,
      "content": {
        "text": "Test pheromone signal"
      },
      "tags": [
        {"value": "test", "weight": 1.0, "category": "test"}
      ],
      "scope": {
        "global": true
      }
    }
  ]
}
EOF

    # Create output directory
    mkdir -p "$tmp_dir/eternal"

    local output
    output=$(pheromone-export "$tmp_dir/pheromones.json" "$tmp_dir/eternal/pheromones.xml" "" "$PROJECT_ROOT/.aether/schemas/pheromone.xsd" 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify output file exists
    if [[ ! -f "$tmp_dir/eternal/pheromones.xml" ]]; then
        test_fail "output file created" "file not found"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify exported count
    local signal_count
    signal_count=$(echo "$output" | jq -r '.result.signals')
    if [[ "$signal_count" -ne 1 ]]; then
        test_fail "1 signal exported" "$signal_count signals"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify XML content
    local xml_content
    xml_content=$(cat "$tmp_dir/eternal/pheromones.xml")

    if [[ "$xml_content" != *"<pheromones"* ]]; then
        test_fail "XML has pheromones root" "root element missing"
        rm -rf "$tmp_dir"
        return 1
    fi

    if [[ "$xml_content" != *'id="sig_test_001"'* ]]; then
        test_fail "XML has signal ID" "signal ID missing"
        rm -rf "$tmp_dir"
        return 1
    fi

    if [[ "$xml_content" != *'type="FOCUS"'* ]]; then
        test_fail "XML has signal type" "signal type missing"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: pheromone-export with missing file
# ============================================================================
test_pheromone_export_missing() {
    local tmp_dir
    tmp_dir=$(mktemp -d)

    local output
    local exit_code

    set +e
    output=$(pheromone-export "/nonexistent/pheromones.json" "$tmp_dir/output.xml" 2>&1)
    exit_code=$?
    set -e
    rm -rf "$tmp_dir"

    if [[ "$exit_code" -eq 0 ]]; then
        test_fail "non-zero exit code" "exit code 0"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON error" "invalid JSON: $output"
        return 1
    fi

    if ! assert_ok_false "$output"; then
        test_fail '{"ok":false}' "$output"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: xml-merge
# ============================================================================
test_xml_merge() {
    local tmp_dir
    tmp_dir=$(mktemp -d)

    # Create main XML
    cat > "$tmp_dir/main.xml" << 'EOF'
<?xml version="1.0"?>
<colony>
  <xi:include href="included.xml" xmlns:xi="http://www.w3.org/2001/XInclude"/>
</colony>
EOF

    # Create included XML
    cat > "$tmp_dir/included.xml" << 'EOF'
<workers>
  <worker>Test</worker>
</workers>
EOF

    local output
    output=$(xml-merge "$tmp_dir/output.xml" "$tmp_dir/main.xml" 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify output file exists
    if [[ ! -f "$tmp_dir/output.xml" ]]; then
        test_fail "output file created" "file not found"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: generate-colony-namespace
# ============================================================================
test_generate_colony_namespace() {
    local output
    output=$(generate-colony-namespace "test-session-123" 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        return 1
    fi

    # Verify namespace URI format
    local namespace
    namespace=$(echo "$output" | jq -r '.result.namespace')
    if [[ "$namespace" != "http://aether.dev/colony/test-session-123" ]]; then
        test_fail "namespace URI correct" "got: $namespace"
        return 1
    fi

    # Verify prefix format
    local prefix
    prefix=$(echo "$output" | jq -r '.result.prefix')
    if [[ "$prefix" != col_* ]]; then
        test_fail "prefix starts with col_" "got: $prefix"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: generate-cross-colony-prefix
# ============================================================================
test_generate_cross_colony_prefix() {
    local output
    output=$(generate-cross-colony-prefix "external-session-456" "local-session-123" 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        return 1
    fi

    # Verify external prefix format
    local prefix
    prefix=$(echo "$output" | jq -r '.result.prefix')
    if [[ "$prefix" != ext_* ]]; then
        test_fail "prefix starts with ext_" "got: $prefix"
        return 1
    fi

    # Verify full prefix includes local hash
    local full_prefix
    full_prefix=$(echo "$output" | jq -r '.result.full_prefix')
    if [[ "$full_prefix" != *_ext_* ]]; then
        test_fail "full prefix has format {hash}_ext_{hash}" "got: $full_prefix"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: prefix-pheromone-id
# ============================================================================
test_prefix_pheromone_id() {
    local output
    output=$(prefix-pheromone-id "sig_001" "col_abc123" 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        return 1
    fi

    # Verify prefixed ID
    local prefixed_id
    prefixed_id=$(echo "$output" | jq -r '.result')
    if [[ "$prefixed_id" != "col_abc123_sig_001" ]]; then
        test_fail "col_abc123_sig_001" "got: $prefixed_id"
        return 1
    fi

    # Test idempotent - already prefixed should not double-prefix
    output=$(prefix-pheromone-id "col_abc123_sig_001" "col_abc123" 2>&1)
    prefixed_id=$(echo "$output" | jq -r '.result')
    if [[ "$prefixed_id" != "col_abc123_sig_001" ]]; then
        test_fail "idempotent: col_abc123_sig_001" "got: $prefixed_id"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: validate-colony-namespace
# ============================================================================
test_validate_colony_namespace() {
    # Test valid colony namespace
    local output
    output=$(validate-colony-namespace "http://aether.dev/colony/test-session" 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        return 1
    fi

    if ! echo "$output" | jq -e '.result.valid == true' > /dev/null 2>&1; then
        test_fail "valid: true for colony namespace" "$output"
        return 1
    fi

    if ! echo "$output" | jq -e '.result.type == "colony"' > /dev/null 2>&1; then
        test_fail "type: colony" "$output"
        return 1
    fi

    # Test valid schema namespace
    output=$(validate-colony-namespace "http://aether.colony/schemas/pheromones" 2>&1)
    if ! echo "$output" | jq -e '.result.valid == true' > /dev/null 2>&1; then
        test_fail "valid: true for schema namespace" "$output"
        return 1
    fi

    if ! echo "$output" | jq -e '.result.type == "schema"' > /dev/null 2>&1; then
        test_fail "type: schema" "$output"
        return 1
    fi

    # Test invalid namespace
    output=$(validate-colony-namespace "http://invalid.namespace/test" 2>&1)
    if ! echo "$output" | jq -e '.result.valid == false' > /dev/null 2>&1; then
        test_fail "valid: false for invalid namespace" "$output"
        return 1
    fi

    return 0
}

# ============================================================================
# Main Test Runner
# ============================================================================

main() {
    log "${YELLOW}=== XML Utilities Test Suite ===${NC}"
    log "Testing: $XML_UTILS_SOURCE"
    log ""

    # Run all tests
    run_test "test_xml_detect_tools" "xml-detect-tools returns available tools"
    run_test "test_xml_well_formed_valid" "xml-well-formed returns true for valid XML"
    run_test "test_xml_well_formed_invalid" "xml-well-formed returns false for invalid XML"
    run_test "test_xml_well_formed_missing" "xml-well-formed handles missing files"
    run_test "test_xml_escape_unescape" "xml-escape and xml-unescape work correctly"
    run_test "test_json_to_xml_valid" "json-to-xml converts valid JSON to XML"
    run_test "test_json_to_xml_invalid" "json-to-xml handles invalid JSON"
    run_test "test_xml_format" "xml-format pretty-prints XML"
    run_test "test_pheromone_to_xml" "pheromone-to-xml converts pheromone format"
    run_test "test_queen_wisdom_to_xml" "queen-wisdom-to-xml converts wisdom format"
    run_test "test_registry_to_xml" "registry-to-xml converts colony registry"
    run_test "test_xml_validate" "xml-validate validates against XSD"
    run_test "test_xml_query" "xml-query executes XPath queries"
    run_test "test_xml_merge" "xml-merge merges XML documents"
    run_test "test_pheromone_export" "pheromone-export exports pheromones to XML"
    run_test "test_pheromone_export_missing" "pheromone-export handles missing files"
    run_test "test_generate_colony_namespace" "generate-colony-namespace creates namespace URI"
    run_test "test_generate_cross_colony_prefix" "generate-cross-colony-prefix creates collision-free prefix"
    run_test "test_prefix_pheromone_id" "prefix-pheromone-id prefixes pheromone IDs"
    run_test "test_validate_colony_namespace" "validate-colony-namespace validates namespace URIs"

    # Print summary
    test_summary
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
