#!/bin/bash
# Pheromone Exchange Module
# JSON/XML bidirectional conversion for pheromone signals
#
# Usage: source .aether/exchange/pheromone-xml.sh
#        xml-pheromone-export <pheromone_json> [output_xml]
#        xml-pheromone-import <pheromone_xml> [output_json]
#        xml-pheromone-validate <pheromone_xml>

set -euo pipefail

# Source dependencies
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$SCRIPT_DIR/utils/xml-core.sh"

# ============================================================================
# Pheromone Export (JSON to XML)
# ============================================================================

# xml-pheromone-export: Convert pheromone JSON to XML format
# Usage: xml-pheromone-export <pheromone_json_file> [output_xml_file]
# Returns: {"ok":true,"result":{"xml":"...","path":"..."}}
xml-pheromone-export() {
    local json_file="${1:-}"
    local output_xml="${2:-}"
    local xsd_file="${3:-.aether/schemas/pheromone.xsd}"

    [[ -z "$json_file" ]] && { xml_json_err "MISSING_ARG" "Missing JSON file argument"; return 1; }
    [[ -f "$json_file" ]] || { xml_json_err "FILE_NOT_FOUND" "JSON file not found: $json_file"; return 1; }

    # Validate JSON
    if ! jq empty "$json_file" 2>/dev/null; then
        xml_json_err "PARSE_ERROR" "Invalid JSON file: $json_file"
        return 1
    fi

    # Generate ISO timestamp
    local generated_at
    generated_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Extract metadata
    local version colony_id
    version=$(jq -r '.version // "1.0.0"' "$json_file")
    colony_id=$(jq -r '.colony_id // "unknown"' "$json_file")

    # Build XML header
    local xml_output
    xml_output="<?xml version=\"1.0\" encoding=\"UTF-8\"?>
<pheromones xmlns=\"http://aether.colony/schemas/pheromones\"
            xmlns:ph=\"http://aether.colony/schemas/pheromones\"
            version=\"$version\"
            generated_at=\"$generated_at\"
            colony_id=\"$colony_id\">"

    # Add metadata
    local source_type context
    source_type=$(jq -r '.metadata.source.type // "system"' "$json_file" 2>/dev/null || echo "system")
    context=$(jq -r '.metadata.context // "Colony pheromone signals"' "$json_file" 2>/dev/null || echo "Colony pheromone signals")

    xml_output="$xml_output
  <metadata>
    <source type=\"$source_type\">aether-pheromone-converter</source>
    <context>$(echo "$context" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g')</context>
  </metadata>"

    # Process signals
    local sig_array_length
    sig_array_length=$(jq '.signals | length' "$json_file" 2>/dev/null || echo "0")

    local sig_idx=0
    while [[ $sig_idx -lt $sig_array_length ]]; do
        local signal
        signal=$(jq -c ".signals[$sig_idx]" "$json_file" 2>/dev/null)
        [[ -n "$signal" ]] || { ((sig_idx++)); continue; }

        # Extract signal fields
        local sig_id sig_type priority source created_at expires_at active
        sig_id=$(echo "$signal" | jq -r '.id // "sig_'"$(date +%s)"'_'"$sig_idx"'"')
        sig_type=$(echo "$signal" | jq -r '.type // "FOCUS"' | tr '[:lower:]' '[:upper:]')
        priority=$(echo "$signal" | jq -r '.priority // "normal"' | tr '[:upper:]' '[:lower:]')
        source=$(echo "$signal" | jq -r '.source // "system"')
        created_at=$(echo "$signal" | jq -r '.created_at // "'"$generated_at"'"')
        expires_at=$(echo "$signal" | jq -r '.expires_at // empty')
        active=$(echo "$signal" | jq -r '.active // true')

        # Validate signal type
        case "$sig_type" in
            FOCUS|REDIRECT|FEEDBACK) ;;
            *) sig_type="FOCUS" ;;
        esac

        # Validate priority
        case "$priority" in
            critical|high|normal|low) ;;
            *) priority="normal" ;;
        esac

        # Build signal element
        xml_output="$xml_output
  <signal id=\"$(echo "$sig_id" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g; s/"/\&quot;/g')\"
          type=\"$sig_type\"
          priority=\"$priority\"
          source=\"$(echo "$source" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g; s/"/\&quot;/g')\"
          created_at=\"$created_at\""

        [[ -n "$expires_at" && "$expires_at" != "null" ]] && xml_output="$xml_output
          expires_at=\"$expires_at\""

        xml_output="$xml_output
          active=\"$active\">"

        # Content section
        local content_text
        content_text=$(echo "$signal" | jq -r '.content.text // .message // ""')
        if [[ -n "$content_text" ]]; then
            xml_output="$xml_output
    <content>
      <text>$(echo "$content_text" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g')</text>
    </content>"
        fi

        xml_output="$xml_output
  </signal>"

        ((sig_idx++))
    done

    xml_output="$xml_output
</pheromones>"

    # Validate against schema if available
    local validated=false
    if [[ -f "$xsd_file" && "$XMLLINT_AVAILABLE" == "true" ]]; then
        local temp_xml
        temp_xml=$(mktemp)
        echo "$xml_output" > "$temp_xml"
        if xmllint --nonet --noent --noout --schema "$xsd_file" "$temp_xml" 2>/dev/null; then
            validated=true
        fi
        rm -f "$temp_xml"
    fi

    # Output result
    if [[ -n "$output_xml" ]]; then
        echo "$xml_output" > "$output_xml"
        xml_json_ok "{\"path\":\"$output_xml\",\"validated\":$validated}"
    else
        local escaped_xml
        escaped_xml=$(echo "$xml_output" | jq -Rs '.')
        xml_json_ok "{\"xml\":$escaped_xml,\"validated\":$validated}"
    fi
}

# ============================================================================
# Pheromone Import (XML to JSON) - NEW Round-Trip Function
# ============================================================================

# xml-pheromone-import: Convert pheromone XML back to JSON
# Usage: xml-pheromone-import <pheromone_xml_file> [output_json_file]
# Returns: {"ok":true,"result":{"json":"...","signals":N,"path":"..."}}
xml-pheromone-import() {
    local xml_file="${1:-}"
    local output_json="${2:-}"

    [[ -z "$xml_file" ]] && { xml_json_err "MISSING_ARG" "Missing XML file argument"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "FILE_NOT_FOUND" "XML file not found: $xml_file"; return 1; }

    # Check well-formedness
    local well_formed
    well_formed=$(xmllint --nonet --noent --noout "$xml_file" 2>&1) || {
        xml_json_err "PARSE_ERROR" "XML is not well-formed"
        return 1
    }

    # Extract metadata using XPath
    local version colony_id generated_at
    if [[ "$XMLLINT_AVAILABLE" == "true" ]]; then
        version=$(xmllint --nonet --noent --xpath "string(/*/@version)" "$xml_file" 2>/dev/null || echo "1.0.0")
        colony_id=$(xmllint --nonet --noent --xpath "string(/*/@colony_id)" "$xml_file" 2>/dev/null || echo "unknown")
        generated_at=$(xmllint --nonet --noent --xpath "string(/*/@generated_at)" "$xml_file" 2>/dev/null || echo "$(date -u +"%Y-%m-%dT%H:%M:%SZ")")
    else
        xml_json_err "TOOL_NOT_AVAILABLE" "xmllint required for XML import"
        return 1
    fi

    # Build JSON structure using jq
    local json_output
    json_output=$(jq -n \
        --arg version "$version" \
        --arg colony_id "$colony_id" \
        --arg generated_at "$generated_at" \
        '{
            version: $version,
            colony_id: $colony_id,
            generated_at: $generated_at,
            signals: []
        }')

    # Extract signals and convert to JSON
    # This is a simplified implementation - full XPath extraction would require xmlstarlet
    if [[ "$XMLSTARLET_AVAILABLE" == "true" ]]; then
        local signals_json
        signals_json=$(xmlstarlet sel -N ph="http://aether.colony/schemas/pheromones" \
            -t -m "//ph:signal" \
            -o '{"id":"' -v "@id" -o '","type":"' -v "@type" -o '","priority":"' -v "@priority" -o '","source":"' -v "@source" -o '","created_at":"' -v "@created_at" -o '"}' \
            -n "$xml_file" 2>/dev/null | jq -s '.')

        # Merge signals into output
        json_output=$(echo "$json_output" | jq --argjson signals "$signals_json" '.signals = $signals')
    fi

    local signal_count
    signal_count=$(echo "$json_output" | jq '.signals | length')

    # Output result
    if [[ -n "$output_json" ]]; then
        echo "$json_output" > "$output_json"
        xml_json_ok "{\"path\":\"$output_json\",\"signals\":$signal_count}"
    else
        local escaped_json
        escaped_json=$(echo "$json_output" | jq -Rs '.')
        xml_json_ok "{\"json\":$escaped_json,\"signals\":$signal_count}"
    fi
}

# ============================================================================
# Pheromone Validation
# ============================================================================

# xml-pheromone-validate: Validate pheromone XML against schema
# Usage: xml-pheromone-validate <pheromone_xml> [xsd_schema]
# Returns: {"ok":true,"result":{"valid":true,"errors":[]}}
xml-pheromone-validate() {
    local xml_file="${1:-}"
    local xsd_file="${2:-.aether/schemas/pheromone.xsd}"

    [[ -z "$xml_file" ]] && { xml_json_err "MISSING_ARG" "Missing XML file argument"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "FILE_NOT_FOUND" "XML file not found: $xml_file"; return 1; }

    if [[ "$XMLLINT_AVAILABLE" != "true" ]]; then
        xml_json_err "TOOL_NOT_AVAILABLE" "xmllint required for validation"
        return 1
    fi

    if [[ ! -f "$xsd_file" ]]; then
        xml_json_err "SCHEMA_NOT_FOUND" "XSD schema not found: $xsd_file"
        return 1
    fi

    local errors
    errors=$(xmllint --nonet --noent --noout --schema "$xsd_file" "$xml_file" 2>&1) && {
        xml_json_ok '{"valid":true,"errors":[]}'
        return 0
    } || {
        local escaped_errors
        escaped_errors=$(echo "$errors" | sed 's/\\/\\\\/g; s/"/\\"/g; s/\t/\\t/g' | tr '\n' ' ')
        xml_json_ok "{\"valid\":false,\"errors\":[\"$escaped_errors\"]}"
        return 0
    }
}

# ============================================================================
# Namespace Utilities
# ============================================================================

# xml-pheromone-prefix-id: Add namespace prefix to signal ID
# Usage: xml-pheromone-prefix-id <signal_id> <colony_prefix>
# Returns: Prefixed ID (direct output, not JSON)
xml-pheromone-prefix-id() {
    local signal_id="${1:-}"
    local colony_prefix="${2:-}"

    [[ -z "$signal_id" ]] && { echo ""; return 1; }
    [[ -z "$colony_prefix" ]] && { echo "$signal_id"; return 0; }

    echo "${colony_prefix}:${signal_id}"
}

# xml-pheromone-deprefix-id: Remove namespace prefix from signal ID
# Usage: xml-pheromone-deprefix-id <prefixed_id>
# Returns: Original ID (direct output, not JSON)
xml-pheromone-deprefix-id() {
    local prefixed_id="${1:-}"

    [[ -z "$prefixed_id" ]] && { echo ""; return 1; }

    # Extract ID after colon
    echo "$prefixed_id" | sed 's/^[^:]*://'
}

# Export functions
export -f xml-pheromone-export xml-pheromone-import xml-pheromone-validate
export -f xml-pheromone-prefix-id xml-pheromone-deprefix-id
