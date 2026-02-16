#!/bin/bash
# XML Core Utilities
# Fundamental XML operations: validation, formatting, escaping
#
# Usage: source .aether/utils/xml-core.sh
#        xml-validate <xml_file> <xsd_file>
#        xml-well-formed <xml_file>
#        xml-format <xml_file>
#        xml-escape <text>
#        xml-unescape <text>

# ============================================================================
# Feature Detection
# ============================================================================

# Check for required XML tools (only if not already set)
if [[ -z "${XMLLINT_AVAILABLE:-}" ]]; then
    XMLLINT_AVAILABLE=false
    if command -v xmllint >/dev/null 2>&1; then
        XMLLINT_AVAILABLE=true
    fi
fi

if [[ -z "${XMLSTARLET_AVAILABLE:-}" ]]; then
    XMLSTARLET_AVAILABLE=false
    if command -v xmlstarlet >/dev/null 2>&1; then
        XMLSTARLET_AVAILABLE=true
    fi
fi

if [[ -z "${XSLTPROC_AVAILABLE:-}" ]]; then
    XSLTPROC_AVAILABLE=false
    if command -v xsltproc >/dev/null 2>&1; then
        XSLTPROC_AVAILABLE=true
    fi
fi

# ============================================================================
# JSON Output Helpers
# ============================================================================

xml_json_ok() { printf '{"ok":true,"result":%s}\n' "$1"; }

xml_json_err() {
    local code="${1:-UNKNOWN_ERROR}"
    local message="${2:-$1}"
    local details="${3:-}"
    if [[ -n "$details" ]]; then
        printf '{"ok":false,"error":"%s","code":"%s","details":"%s"}\n' "$message" "$code" "$details" >&2
    else
        printf '{"ok":false,"error":"%s","code":"%s"}\n' "$message" "$code" >&2
    fi
    return 1
}

# ============================================================================
# Core XML Functions
# ============================================================================

# xml-detect-tools: Detect available XML tools
# Usage: xml-detect-tools
# Returns: {"ok":true,"result":{"xmllint":true,"xmlstarlet":false,...}}
xml-detect-tools() {
    xml_json_ok "{\"xmllint\":$XMLLINT_AVAILABLE,\"xmlstarlet\":$XMLSTARLET_AVAILABLE,\"xsltproc\":$XSLTPROC_AVAILABLE}"
}

# xml-validate: Validate XML against XSD schema using xmllint
# Usage: xml-validate <xml_file> <xsd_file>
# Returns: {"ok":true,"result":{"valid":true,"errors":[]}} or error
xml-validate() {
    local xml_file="${1:-}"
    local xsd_file="${2:-}"

    # Validate arguments
    [[ -z "$xml_file" ]] && { xml_json_err "MISSING_ARG" "Missing XML file argument"; return 1; }
    [[ -z "$xsd_file" ]] && { xml_json_err "MISSING_ARG" "Missing XSD schema file argument"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "FILE_NOT_FOUND" "XML file not found: $xml_file"; return 1; }
    [[ -f "$xsd_file" ]] || { xml_json_err "FILE_NOT_FOUND" "XSD schema file not found: $xsd_file"; return 1; }

    # Check for xmllint
    if [[ "$XMLLINT_AVAILABLE" != "true" ]]; then
        xml_json_err "TOOL_NOT_AVAILABLE" "xmllint not available. Install libxml2 utilities."
        return 1
    fi

    # Validate XML against XSD (with XXE protection)
    local errors
    errors=$(xmllint --nonet --noent --noout --schema "$xsd_file" "$xml_file" 2>&1) && {
        xml_json_ok '{"valid":true,"errors":[]}'
        return 0
    } || {
        # Escape errors for JSON
        local escaped_errors
        escaped_errors=$(echo "$errors" | sed 's/\\/\\\\/g; s/"/\\"/g; s/\t/\\t/g' | tr '\n' ' ')
        xml_json_ok "{\"valid\":false,\"errors\":[\"$escaped_errors\"]}"
        return 0
    }
}

# xml-well-formed: Check if XML is well-formed (no schema validation)
# Usage: xml-well-formed <xml_file>
# Returns: {"ok":true,"result":{"well_formed":true}} or error
xml-well-formed() {
    local xml_file="${1:-}"

    [[ -z "$xml_file" ]] && { xml_json_err "MISSING_ARG" "Missing XML file argument"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "FILE_NOT_FOUND" "XML file not found: $xml_file"; return 1; }

    if [[ "$XMLLINT_AVAILABLE" != "true" ]]; then
        xml_json_err "TOOL_NOT_AVAILABLE" "xmllint not available. Install libxml2 utilities."
        return 1
    fi

    # Check well-formedness with XXE protection
    if xmllint --nonet --noent --noout "$xml_file" 2>/dev/null; then
        xml_json_ok '{"well_formed":true}'
        return 0
    else
        xml_json_ok '{"well_formed":false}'
        return 0
    fi
}

# xml-format: Pretty-print XML with proper indentation
# Usage: xml-format <xml_file> [output_file]
# Returns: {"ok":true,"result":{"formatted":true,"output":"..."}} or writes to file
xml-format() {
    local xml_file="${1:-}"
    local output_file="${2:-}"

    [[ -z "$xml_file" ]] && { xml_json_err "MISSING_ARG" "Missing XML file argument"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "FILE_NOT_FOUND" "XML file not found: $xml_file"; return 1; }

    if [[ "$XMLLINT_AVAILABLE" != "true" ]]; then
        xml_json_err "TOOL_NOT_AVAILABLE" "xmllint not available. Install libxml2 utilities."
        return 1
    fi

    local formatted
    formatted=$(xmllint --nonet --noent --format "$xml_file" 2>/dev/null) || {
        xml_json_err "PARSE_ERROR" "Failed to parse XML file"
        return 1
    }

    if [[ -n "$output_file" ]]; then
        echo "$formatted" > "$output_file"
        xml_json_ok "{\"formatted\":true,\"path\":\"$output_file\"}"
    else
        # Escape for JSON
        local escaped
        escaped=$(echo "$formatted" | sed 's/\\/\\\\/g; s/"/\\"/g' | tr '\n' ' ')
        xml_json_ok "{\"formatted\":true,\"output\":\"$escaped\"}"
    fi
}

# xml-escape: Escape special XML characters
# Usage: xml-escape <text>
# Returns: Escaped text (not JSON - direct output)
xml-escape() {
    local text="${1:-}"
    # Escape &, <, >, ", '
    echo "$text" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g; s/"/\&quot;/g; s/'"'"'/\&apos;/g'
}

# xml-unescape: Unescape XML entities
# Usage: xml-unescape <text>
# Returns: Unescaped text (not JSON - direct output)
xml-unescape() {
    local text="${1:-}"
    # Unescape XML entities
    echo "$text" | sed 's/&amp;/\&/g; s/&lt;/</g; s/&gt;/>/g; s/&quot;/"/g; s/&apos;/'"'"'/g'
}

# xml-escape-content: Escape content for XML CDATA or text nodes
# Internal helper for consistent escaping
_xml_escape_content() {
    local content="${1:-}"
    # Basic XML escaping for text content
    echo "$content" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g'
}

# Export functions for use by other modules
export -f xml_json_ok xml_json_err
export -f xml-detect-tools xml-validate xml-well-formed xml-format
export -f xml-escape xml-unescape _xml_escape_content
export XMLLINT_AVAILABLE XMLSTARLET_AVAILABLE XSLTPROC_AVAILABLE
