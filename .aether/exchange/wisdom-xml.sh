#!/bin/bash
# Queen Wisdom Exchange Module
# JSON/XML bidirectional conversion for wisdom entries
#
# Usage: source .aether/exchange/wisdom-xml.sh
#        xml-wisdom-export <wisdom_json> [output_xml]
#        xml-wisdom-import <wisdom_xml> [output_json]
#        xml-wisdom-promote <entry_id>

set -euo pipefail

# Source dependencies
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$SCRIPT_DIR/utils/xml-core.sh"

# Default promotion threshold (confidence level required)
PROMOTION_THRESHOLD=0.8

# ============================================================================
# Wisdom Export (JSON to XML)
# ============================================================================

# xml-wisdom-export: Convert queen-wisdom JSON to XML
# Usage: xml-wisdom-export <wisdom_json_file> [output_xml_file]
# Returns: {"ok":true,"result":{"path":"...","entries":N}}
xml-wisdom-export() {
    local json_file="${1:-}"
    local output_xml="${2:-}"

    [[ -z "$json_file" ]] && { xml_json_err "MISSING_ARG" "Missing JSON file argument"; return 1; }
    [[ -f "$json_file" ]] || { xml_json_err "FILE_NOT_FOUND" "JSON file not found: $json_file"; return 1; }

    # Validate JSON
    if ! jq empty "$json_file" 2>/dev/null; then
        xml_json_err "PARSE_ERROR" "Invalid JSON file: $json_file"
        return 1
    fi

    local version created modified colony_id
    version=$(jq -r '.version // "1.0.0"' "$json_file")
    created=$(jq -r '.metadata.created // "'"$(date -u +"%Y-%m-%dT%H:%M:%SZ")"'"' "$json_file" 2>/dev/null || date -u +"%Y-%m-%dT%H:%M:%SZ")
    modified=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    colony_id=$(jq -r '.metadata.colony_id // "unknown"' "$json_file" 2>/dev/null || echo "unknown")

    # Build XML
    local xml_output
    xml_output="<?xml version=\"1.0\" encoding=\"UTF-8\"?>
<queen-wisdom xmlns=\"http://aether.colony/schemas/queen-wisdom/1.0\"
               xmlns:qw=\"http://aether.colony/schemas/queen-wisdom/1.0\">
  <metadata>
    <version>$version</version>
    <created>$created</created>
    <modified>$modified</modified>
    <colony_id>$colony_id</colony_id>
  </metadata>"

    # Process philosophies
    xml_output="$xml_output
  <philosophies>"

    local phil_count
    phil_count=$(jq '.philosophies | length' "$json_file" 2>/dev/null || echo "0")
    local idx=0
    while [[ $idx -lt $phil_count ]]; do
        local entry
        entry=$(jq -c ".philosophies[$idx]" "$json_file")
        local id confidence domain source content
        id=$(echo "$entry" | jq -r '.id')
        confidence=$(echo "$entry" | jq -r '.confidence // "0.5"')
        domain=$(echo "$entry" | jq -r '.domain // "general"')
        source=$(echo "$entry" | jq -r '.source // "observation"')
        content=$(echo "$entry" | jq -r '.content // ""' | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g')

        xml_output="$xml_output
    <philosophy id=\"$id\" confidence=\"$confidence\" domain=\"$domain\" source=\"$source\" created_at=\"$modified\">
      <content>$content</content>
    </philosophy>"
        ((idx++))
    done

    xml_output="$xml_output
  </philosophies>"

    # Process patterns
    xml_output="$xml_output
  <patterns>"

    local pattern_count
    pattern_count=$(jq '.patterns | length' "$json_file" 2>/dev/null || echo "0")
    idx=0
    while [[ $idx -lt $pattern_count ]]; do
        local entry
        entry=$(jq -c ".patterns[$idx]" "$json_file")
        local id confidence domain source
        id=$(echo "$entry" | jq -r '.id')
        confidence=$(echo "$entry" | jq -r '.confidence // "0.5"')
        domain=$(echo "$entry" | jq -r '.domain // "general"')
        source=$(echo "$entry" | jq -r '.source // "observation"')

        xml_output="$xml_output
    <pattern id=\"$id\" confidence=\"$confidence\" domain=\"$domain\" source=\"$source\" created_at=\"$modified\">
      <content>$(echo "$entry" | jq -r '.content // ""' | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g')</content>
    </pattern>"
        ((idx++))
    done

    xml_output="$xml_output
  </patterns>
</queen-wisdom>"

    local entry_count=$((phil_count + pattern_count))

    # Output
    if [[ -n "$output_xml" ]]; then
        echo "$xml_output" > "$output_xml"
        xml_json_ok "{\"path\":\"$output_xml\",\"entries\":$entry_count}"
    else
        local escaped
        escaped=$(echo "$xml_output" | jq -Rs '.')
        xml_json_ok "{\"xml\":$escaped,\"entries\":$entry_count}"
    fi
}

# ============================================================================
# Wisdom Import (XML to JSON) - Round-Trip
# ============================================================================

# xml-wisdom-import: Convert queen-wisdom XML to JSON
# Usage: xml-wisdom-import <wisdom_xml_file> [output_json_file]
# Returns: {"ok":true,"result":{"entries":N,"path":"..."}}
xml-wisdom-import() {
    local xml_file="${1:-}"
    local output_json="${2:-}"

    [[ -z "$xml_file" ]] && { xml_json_err "MISSING_ARG" "Missing XML file argument"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "FILE_NOT_FOUND" "XML file not found: $xml_file"; return 1; }

    # Check well-formedness
    if [[ "$XMLLINT_AVAILABLE" == "true" ]]; then
        xmllint --nonet --noent --noout "$xml_file" 2>/dev/null || {
            xml_json_err "PARSE_ERROR" "XML is not well-formed"
            return 1
        }
    fi

    # Extract metadata
    local version created modified colony_id
    if [[ "$XMLLINT_AVAILABLE" == "true" ]]; then
        version=$(xmllint --nonet --noent --xpath "string(/*/metadata/version)" "$xml_file" 2>/dev/null || echo "1.0.0")
        created=$(xmllint --nonet --noent --xpath "string(/*/metadata/created)" "$xml_file" 2>/dev/null || date -u +"%Y-%m-%dT%H:%M:%SZ")
        modified=$(xmllint --nonet --noent --xpath "string(/*/metadata/modified)" "$xml_file" 2>/dev/null || date -u +"%Y-%m-%dT%H:%M:%SZ")
        colony_id=$(xmllint --nonet --noent --xpath "string(/*/metadata/colony_id)" "$xml_file" 2>/dev/null || echo "unknown")
    else
        version="1.0.0"
        created=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
        modified="$created"
        colony_id="unknown"
    fi

    # Build JSON structure
    local json_output
    json_output=$(jq -n \
        --arg version "$version" \
        --arg created "$created" \
        --arg modified "$modified" \
        --arg colony_id "$colony_id" \
        '{
            version: $version,
            metadata: {
                created: $created,
                modified: $modified,
                colony_id: $colony_id
            },
            philosophies: [],
            patterns: [],
            redirects: [],
            stack_wisdom: [],
            decrees: []
        }')

    local entry_count=0

    # Extract philosophies using xmlstarlet if available
    if [[ "$XMLSTARLET_AVAILABLE" == "true" ]]; then
        local phil_array
        phil_array=$(xmlstarlet sel -t -m "//philosophy" \
            -o '{"id":"' -v "@id" -o '","confidence":' -v "@confidence" -o ',"domain":"' -v "@domain" -o '","source":"' -v "@source" -o '","content":"' -v "content" -o '"}' \
            -n "$xml_file" 2>/dev/null | jq -s '.')

        local phil_count
        phil_count=$(echo "$phil_array" | jq 'length')
        if [[ $phil_count -gt 0 ]]; then
            json_output=$(echo "$json_output" | jq --argjson philosophies "$phil_array" '.philosophies = $philosophies')
            entry_count=$((entry_count + phil_count))
        fi

        # Extract patterns
        local pattern_array
        pattern_array=$(xmlstarlet sel -t -m "//pattern" \
            -o '{"id":"' -v "@id" -o '","confidence":' -v "@confidence" -o ',"domain":"' -v "@domain" -o '"}' \
            -n "$xml_file" 2>/dev/null | jq -s '.')

        local pattern_count
        pattern_count=$(echo "$pattern_array" | jq 'length')
        if [[ $pattern_count -gt 0 ]]; then
            json_output=$(echo "$json_output" | jq --argjson patterns "$pattern_array" '.patterns = $patterns')
            entry_count=$((entry_count + pattern_count))
        fi
    fi

    # Output
    if [[ -n "$output_json" ]]; then
        echo "$json_output" > "$output_json"
        xml_json_ok "{\"path\":\"$output_json\",\"entries\":$entry_count}"
    else
        local escaped
        escaped=$(echo "$json_output" | jq -Rs '.')
        xml_json_ok "{\"json\":$escaped,\"entries\":$entry_count}"
    fi
}

# ============================================================================
# Wisdom Validation
# ============================================================================

# xml-wisdom-validate: Validate wisdom XML against schema
# Usage: xml-wisdom-validate <wisdom_xml> [xsd_schema]
xml-wisdom-validate() {
    local xml_file="${1:-}"
    local xsd_file="${2:-.aether/schemas/queen-wisdom.xsd}"

    [[ -z "$xml_file" ]] && { xml_json_err "MISSING_ARG" "Missing XML file argument"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "FILE_NOT_FOUND" "XML file not found: $xml_file"; return 1; }

    if [[ "$XMLLINT_AVAILABLE" != "true" ]]; then
        xml_json_err "TOOL_NOT_AVAILABLE" "xmllint required for validation"
        return 1
    fi

    if [[ ! -f "$xsd_file" ]]; then
        xml_json_ok '{"valid":true,"warning":"Schema not found, skipping validation","schema":"'$xsd_file'"}'
        return 0
    fi

    xmllint --nonet --noent --noout --schema "$xsd_file" "$xml_file" 2>/dev/null && {
        xml_json_ok '{"valid":true}'
    } || {
        xml_json_ok '{"valid":false}'
    }
}

# ============================================================================
# Wisdom Promotion
# ============================================================================

# xml-wisdom-promote: Promote a pattern to philosophy if confidence threshold met
# Usage: xml-wisdom-promote <wisdom_json> <entry_id>
# Returns: {"ok":true,"result":{"promoted":true,"new_domain":"philosophy"}}
xml-wisdom-promote() {
    local json_file="${1:-}"
    local entry_id="${2:-}"

    [[ -z "$json_file" ]] && { xml_json_err "MISSING_ARG" "Missing JSON file argument"; return 1; }
    [[ -z "$entry_id" ]] && { xml_json_err "MISSING_ARG" "Missing entry ID argument"; return 1; }
    [[ -f "$json_file" ]] || { xml_json_err "FILE_NOT_FOUND" "JSON file not found: $json_file"; return 1; }

    # Find entry in patterns
    local entry
    entry=$(jq --arg id "$entry_id" '.patterns[] | select(.id == $id)' "$json_file")

    if [[ -z "$entry" || "$entry" == "null" ]]; then
        xml_json_err "ENTRY_NOT_FOUND" "Pattern entry not found: $entry_id"
        return 1
    fi

    # Check confidence
    local confidence
    confidence=$(echo "$entry" | jq -r '.confidence // 0')

    if (( $(echo "$confidence < $PROMOTION_THRESHOLD" | bc -l) )); then
        xml_json_ok "{\"promoted\":false,\"reason\":\"confidence_below_threshold\",\"confidence\":$confidence,\"threshold\":$PROMOTION_THRESHOLD}"
        return 0
    fi

    # Promote: add to philosophies, remove from patterns
    local updated_json
    updated_json=$(jq --arg id "$entry_id" '
        (.patterns[] | select(.id == $id)) as $entry |
        .philosophies += [$entry] |
        .patterns = [.patterns[] | select(.id != $id)]
    ' "$json_file")

    echo "$updated_json" > "$json_file"

    xml_json_ok "{\"promoted\":true,\"new_domain\":\"philosophy\",\"confidence\":$confidence}"
}

# Export functions
export -f xml-wisdom-export xml-wisdom-import xml-wisdom-validate xml-wisdom-promote
export PROMOTION_THRESHOLD
