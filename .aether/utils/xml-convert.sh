#!/bin/bash
# XML Conversion Utilities
# Bidirectional JSON/XML conversion and document merging
#
# Usage: source .aether/utils/xml-convert.sh
#        xml-to-json <xml_file> [--pretty]
#        json-to-xml <json_file> [root_element]
#        xml-merge <output_file> <main_xml_file>
#        xml-convert-detect-format <file>

set -euo pipefail

# Source xml-core.sh for JSON helpers and tool detection
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/xml-core.sh"

# Additional tool detection for conversion
XML2JSON_AVAILABLE=false
if command -v xml2json >/dev/null 2>&1; then
    XML2JSON_AVAILABLE=true
fi

# ============================================================================
# Format Detection
# ============================================================================

# xml-convert-detect-format: Detect if file is XML or JSON
# Usage: xml-convert-detect-format <file>
# Returns: {"ok":true,"result":{"format":"xml|json|unknown","confidence":"high|medium|low"}}
xml-convert-detect-format() {
    local file="${1:-}"

    [[ -z "$file" ]] && { xml_json_err "MISSING_ARG" "Missing file argument"; return 1; }
    [[ -f "$file" ]] || { xml_json_err "FILE_NOT_FOUND" "File not found: $file"; return 1; }

    # Read first 1KB for analysis
    local header
    header=$(head -c 1024 "$file" 2>/dev/null || head -c 1024 < "$file")

    # Check for XML signatures
    if echo "$header" | grep -qE '^\s*<\?xml\s+version'; then
        xml_json_ok '{"format":"xml","confidence":"high","signature":"xml_declaration"}'
        return 0
    fi

    if echo "$header" | grep -qE '^\s*<[a-zA-Z_][a-zA-Z0-9_]*[\s>]'; then
        xml_json_ok '{"format":"xml","confidence":"medium","signature":"root_element"}'
        return 0
    fi

    # Check for JSON signatures
    if echo "$header" | grep -qE '^\s*(\{|\[)'; then
        # Verify it's valid JSON
        if jq empty "$file" 2>/dev/null; then
            xml_json_ok '{"format":"json","confidence":"high","signature":"valid_json"}'
            return 0
        else
            xml_json_ok '{"format":"json","confidence":"low","signature":"json_like","note":"Invalid JSON syntax"}'
            return 0
        fi
    fi

    xml_json_ok '{"format":"unknown","confidence":"low","signature":"none"}'
}

# ============================================================================
# XML to JSON Conversion
# ============================================================================

# xml-to-json: Convert XML to JSON format
# Usage: xml-to-json <xml_file> [--pretty]
# Returns: {"ok":true,"result":{"json":"...","format":"object"}}
xml-to-json() {
    local xml_file="${1:-}"
    local pretty=false

    # Parse optional arguments
    shift || true
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --pretty) pretty=true; shift ;;
            *) shift ;;
        esac
    done

    [[ -z "$xml_file" ]] && { xml_json_err "MISSING_ARG" "Missing XML file argument"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "FILE_NOT_FOUND" "XML file not found: $xml_file"; return 1; }

    # Check well-formedness first
    local well_formed_result
    well_formed_result=$(xml-well-formed "$xml_file" 2>/dev/null)
    if ! echo "$well_formed_result" | jq -e '.result.well_formed' >/dev/null 2>&1; then
        xml_json_err "PARSE_ERROR" "XML is not well-formed"
        return 1
    fi

    # Try xml2json if available (npm package)
    if [[ "$XML2JSON_AVAILABLE" == "true" ]]; then
        local json_output
        if json_output=$(xml2json "$xml_file" 2>/dev/null); then
            if [[ "$pretty" == "true" ]]; then
                json_output=$(echo "$json_output" | jq '.')
            fi
            local escaped_json
            escaped_json=$(echo "$json_output" | jq -Rs '.')
            xml_json_ok "{\"format\":\"object\",\"json\":$escaped_json}"
            return 0
        fi
    fi

    # Fallback: Use xsltproc with built-in XSLT
    if [[ "$XSLTPROC_AVAILABLE" == "true" ]]; then
        local xslt_script
        xslt_script=$(cat << 'XSLT'
<?xml version="1.0"?>
<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
<xsl:output method="text"/>
<xsl:template match="/">
<xsl:text>{"root":</xsl:text>
<xsl:apply-templates select="*"/>
<xsl:text>}</xsl:text>
</xsl:template>
<xsl:template match="*">
<xsl:text>{"</xsl:text>
<xsl:value-of select="name()"/>
<xsl:text>":</xsl:text>
<xsl:choose>
<xsl:when test="count(*) > 0">
<xsl:text>[</xsl:text>
<xsl:apply-templates select="*"/>
<xsl:text>]</xsl:text>
</xsl:when>
<xsl:otherwise>
<xsl:text>"</xsl:text>
<xsl:value-of select="."/>
<xsl:text>"</xsl:text>
</xsl:otherwise>
</xsl:choose>
<xsl:text>}</xsl:text>
<xsl:if test="position() != last()">,</xsl:if>
</xsl:template>
</xsl:stylesheet>
XSLT
)
        local json_result
        json_result=$(echo "$xslt_script" | xsltproc - "$xml_file" 2>/dev/null) || {
            xml_json_err "CONVERSION_ERROR" "XSLT conversion failed"
            return 1
        }
        xml_json_ok "{\"format\":\"object\",\"json\":$(echo "$json_result" | jq -Rs '.')}"
        return 0
    fi

    # Last resort: Use xmlstarlet if available
    if [[ "$XMLSTARLET_AVAILABLE" == "true" ]]; then
        local json_result
        json_result=$(xmlstarlet sel -t -m "/" -o '{"root":{' -m "*" -v "name()" -o ':"' -v "." -o '"' -b -o '}}' "$xml_file" 2>/dev/null) || {
            xml_json_err "CONVERSION_ERROR" "xmlstarlet conversion failed"
            return 1
        }
        xml_json_ok "{\"format\":\"object\",\"json\":$(echo "$json_result" | jq -Rs '.')}"
        return 0
    fi

    xml_json_err "TOOL_NOT_AVAILABLE" "No XML to JSON conversion tool available. Install xml2json, xsltproc, or xmlstarlet."
    return 1
}

# ============================================================================
# JSON to XML Conversion
# ============================================================================

# json-to-xml: Convert JSON to XML
# Usage: json-to-xml <json_file> [root_element]
# Returns: {"ok":true,"result":{"xml":"<root>...</root>"}}
json-to-xml() {
    local json_file="${1:-}"
    local root_element="${2:-root}"

    [[ -z "$json_file" ]] && { xml_json_err "MISSING_ARG" "Missing JSON file argument"; return 1; }
    [[ -f "$json_file" ]] || { xml_json_err "FILE_NOT_FOUND" "JSON file not found: $json_file"; return 1; }

    # Validate JSON first
    if ! jq empty "$json_file" 2>/dev/null; then
        xml_json_err "PARSE_ERROR" "Invalid JSON file: $json_file"
        return 1
    fi

    # Build XML using jq to generate structure
    local xml_output
    xml_output=$(jq -r --arg root "$root_element" '
        def to_xml:
            if type == "object" then
                to_entries | map(
                    "<\(.key)>\(.value | to_xml)</\(.key)>"
                ) | join("")
            elif type == "array" then
                map("<item>\(. | to_xml)</item>") | join("")
            elif type == "string" then
                .
            elif type == "number" then
                tostring
            elif type == "boolean" then
                tostring
            elif type == "null" then
                ""
            else
                tostring
            end;
        "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<\($root)>\n" + (to_xml) + "\n</\($root)>"
    ' "$json_file" 2>/dev/null) || {
        xml_json_err "CONVERSION_ERROR" "JSON to XML conversion failed"
        return 1
    }

    # Escape the XML for JSON output
    local escaped_xml
    escaped_xml=$(echo "$xml_output" | jq -Rs '.')
    xml_json_ok "{\"xml\":$escaped_xml}"
}

# ============================================================================
# XML Document Merging
# ============================================================================

# xml-merge: XInclude document merging
# Usage: xml-merge <output_file> <main_xml_file>
# Returns: {"ok":true,"result":{"merged":true,"output":"<path>","sources_resolved":N}}
xml-merge() {
    local output_file="${1:-}"
    local main_xml="${2:-}"

    [[ -z "$output_file" ]] && { xml_json_err "MISSING_ARG" "Missing output file argument"; return 1; }
    [[ -z "$main_xml" ]] && { xml_json_err "MISSING_ARG" "Missing main XML file argument"; return 1; }
    [[ -f "$main_xml" ]] || { xml_json_err "FILE_NOT_FOUND" "Main XML file not found: $main_xml"; return 1; }

    # Check well-formedness of main file
    local well_formed_result
    well_formed_result=$(xml-well-formed "$main_xml" 2>/dev/null)
    if ! echo "$well_formed_result" | jq -e '.result.well_formed' >/dev/null 2>&1; then
        xml_json_err "PARSE_ERROR" "Main XML file is not well-formed"
        return 1
    fi

    # Use xmllint for XInclude processing with security flags
    if [[ "$XMLLINT_AVAILABLE" == "true" ]]; then
        local merged
        merged=$(xmllint --nonet --noent --xinclude "$main_xml" 2>/dev/null) || {
            xml_json_err "MERGE_ERROR" "XInclude merge failed"
            return 1
        }

        # Write output
        echo "$merged" > "$output_file"

        # Count resolved includes (approximate)
        local resolved_count
        resolved_count=$(echo "$merged" | grep -c '<xi:include' 2>/dev/null || echo "0")

        local escaped_output
        escaped_output=$(echo "$output_file" | jq -Rs '.[:-1]')
        xml_json_ok "{\"merged\":true,\"output\":$escaped_output,\"sources_resolved\":$resolved_count}"
        return 0
    fi

    # Without xmllint, we cannot safely process XInclude
    xml_json_err "TOOL_NOT_AVAILABLE" "xmllint required for XInclude merging. Install libxml2 utilities."
    return 1
}

# Export functions
export -f xml-convert-detect-format xml-to-json json-to-xml xml-merge
export XML2JSON_AVAILABLE
