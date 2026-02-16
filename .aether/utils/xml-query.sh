#!/bin/bash
# XML Query Utilities
# XPath queries with xmlstarlet and xmllint fallback
#
# Usage: source .aether/utils/xml-query.sh
#        xml-query <xml_file> <xpath_expression>
#        xml-query-attr <xml_file> <xpath_expression>
#        xml-query-text <xml_file> <element_name>
#        xml-query-count <xml_file> <xpath_expression>

set -euo pipefail

# Source xml-core.sh for JSON helpers and tool detection
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/xml-core.sh"

# ============================================================================
# XPath Query Functions
# ============================================================================

# xml-query: Execute XPath query against XML file
# Usage: xml-query <xml_file> <xpath_expression>
# Returns: {"ok":true,"result":{"matches":["value1","value2",...]}}
xml-query() {
    local xml_file="${1:-}"
    local xpath="${2:-}"

    [[ -z "$xml_file" ]] && { xml_json_err "MISSING_ARG" "Missing XML file argument"; return 1; }
    [[ -z "$xpath" ]] && { xml_json_err "MISSING_ARG" "Missing XPath expression"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "FILE_NOT_FOUND" "XML file not found: $xml_file"; return 1; }

    local results=""

    # Prefer xmlstarlet for full XPath support
    if [[ "$XMLSTARLET_AVAILABLE" == "true" ]]; then
        results=$(xmlstarlet sel -t -v "$xpath" "$xml_file" 2>/dev/null | tr '\n' '|')
        # Remove trailing pipe
        results="${results%|}"
    elif [[ "$XMLLINT_AVAILABLE" == "true" ]]; then
        # xmllint has limited XPath but works for basic queries
        # Note: xmllint --xpath returns the text content of matched nodes
        results=$(xmllint --nonet --noent --xpath "$xpath" "$xml_file" 2>/dev/null | \
                  sed 's/<[^>]*>//g' | tr '\n' '|' | sed 's/|$//')
    else
        xml_json_err "TOOL_NOT_AVAILABLE" "No XPath-capable tool available (install xmlstarlet or libxml2)"
        return 1
    fi

    # Build JSON array from pipe-separated results
    if [[ -n "$results" ]]; then
        local json_array="["
        local first=true
        IFS='|' read -ra matches <<< "$results"
        for match in "${matches[@]}"; do
            # Trim whitespace and escape for JSON
            match=$(echo "$match" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
            match=$(echo "$match" | sed 's/\\/\\\\/g; s/"/\\"/g')
            if [[ "$first" == "true" ]]; then
                first=false
                json_array="$json_array\"$match\""
            else
                json_array="$json_array,\"$match\""
            fi
        done
        json_array="$json_array]"
        xml_json_ok "{\"matches\":$json_array}"
    else
        xml_json_ok '{"matches":[]}'
    fi
}

# xml-query-attr: Extract attribute values using XPath
# Usage: xml-query-attr <xml_file> <element_xpath> <attribute_name>
# Returns: {"ok":true,"result":{"attribute":"name","values":["val1","val2"]}}
xml-query-attr() {
    local xml_file="${1:-}"
    local element_xpath="${2:-}"
    local attr_name="${3:-}"

    [[ -z "$xml_file" ]] && { xml_json_err "MISSING_ARG" "Missing XML file argument"; return 1; }
    [[ -z "$element_xpath" ]] && { xml_json_err "MISSING_ARG" "Missing element XPath"; return 1; }
    [[ -z "$attr_name" ]] && { xml_json_err "MISSING_ARG" "Missing attribute name"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "FILE_NOT_FOUND" "XML file not found: $xml_file"; return 1; }

    local full_xpath="${element_xpath}/@${attr_name}"
    local results=""

    if [[ "$XMLSTARLET_AVAILABLE" == "true" ]]; then
        results=$(xmlstarlet sel -t -v "$full_xpath" "$xml_file" 2>/dev/null | tr '\n' '|')
        results="${results%|}"
    elif [[ "$XMLLINT_AVAILABLE" == "true" ]]; then
        # xmllint can extract attributes with //@attrname syntax
        results=$(xmllint --nonet --noent --xpath "$full_xpath" "$xml_file" 2>/dev/null | tr '\n' '|')
        results="${results%|}"
    else
        xml_json_err "TOOL_NOT_AVAILABLE" "No XPath-capable tool available"
        return 1
    fi

    # Build JSON array
    if [[ -n "$results" ]]; then
        local json_array="["
        local first=true
        IFS='|' read -ra values <<< "$results"
        for val in "${values[@]}"; do
            val=$(echo "$val" | sed 's/\\/\\\\/g; s/"/\\"/g')
            if [[ "$first" == "true" ]]; then
                first=false
                json_array="$json_array\"$val\""
            else
                json_array="$json_array,\"$val\""
            fi
        done
        json_array="$json_array]"
        xml_json_ok "{\"attribute\":\"$attr_name\",\"values\":$json_array}"
    else
        xml_json_ok "{\"attribute\":\"$attr_name\",\"values\":[]}"
    fi
}

# xml-query-text: Extract text content of elements
# Usage: xml-query-text <xml_file> <element_name>
# Returns: {"ok":true,"result":{"element":"name","text":["text1","text2"]}}
xml-query-text() {
    local xml_file="${1:-}"
    local element_name="${2:-}"

    [[ -z "$xml_file" ]] && { xml_json_err "MISSING_ARG" "Missing XML file argument"; return 1; }
    [[ -z "$element_name" ]] && { xml_json_err "MISSING_ARG" "Missing element name"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "FILE_NOT_FOUND" "XML file not found: $xml_file"; return 1; }

    local xpath="//$element_name"
    local results=""

    if [[ "$XMLSTARLET_AVAILABLE" == "true" ]]; then
        # Use -m to match nodes and extract text
        results=$(xmlstarlet sel -t -m "$xpath" -v "." -n "$xml_file" 2>/dev/null | tr '\n' '|')
        results="${results%|}"
    elif [[ "$XMLLINT_AVAILABLE" == "true" ]]; then
        # xmllint --xpath returns text content directly for simple paths
        results=$(xmllint --nonet --noent --xpath "$xpath" "$xml_file" 2>/dev/null | \
                  sed 's/<[^>]*>//g' | tr '\n' '|' | sed 's/|$//')
    else
        xml_json_err "TOOL_NOT_AVAILABLE" "No XPath-capable tool available"
        return 1
    fi

    if [[ -n "$results" ]]; then
        local json_array="["
        local first=true
        IFS='|' read -ra texts <<< "$results"
        for text in "${texts[@]}"; do
            text=$(echo "$text" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
            text=$(echo "$text" | sed 's/\\/\\\\/g; s/"/\\"/g')
            if [[ "$first" == "true" ]]; then
                first=false
                json_array="$json_array\"$text\""
            else
                json_array="$json_array,\"$text\""
            fi
        done
        json_array="$json_array]"
        xml_json_ok "{\"element\":\"$element_name\",\"text\":$json_array}"
    else
        xml_json_ok "{\"element\":\"$element_name\",\"text\":[]}"
    fi
}

# xml-query-count: Count nodes matching XPath expression
# Usage: xml-query-count <xml_file> <xpath_expression>
# Returns: {"ok":true,"result":{"count":5}}
xml-query-count() {
    local xml_file="${1:-}"
    local xpath="${2:-}"

    [[ -z "$xml_file" ]] && { xml_json_err "MISSING_ARG" "Missing XML file argument"; return 1; }
    [[ -z "$xpath" ]] && { xml_json_err "MISSING_ARG" "Missing XPath expression"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "FILE_NOT_FOUND" "XML file not found: $xml_file"; return 1; }

    local count=0

    if [[ "$XMLSTARLET_AVAILABLE" == "true" ]]; then
        # Use count() XPath function
        local count_result
        count_result=$(xmlstarlet sel -t -v "count($xpath)" "$xml_file" 2>/dev/null)
        count="${count_result:-0}"
    elif [[ "$XMLLINT_AVAILABLE" == "true" ]]; then
        # Count matching lines (approximate for simple cases)
        local matches
        matches=$(xmllint --nonet --noent --xpath "$xpath" "$xml_file" 2>/dev/null | grep -c '<' || true)
        count="${matches:-0}"
    else
        xml_json_err "TOOL_NOT_AVAILABLE" "No XPath-capable tool available"
        return 1
    fi

    xml_json_ok "{\"count\":$count}"
}

# Export functions
export -f xml-query xml-query-attr xml-query-text xml-query-count
