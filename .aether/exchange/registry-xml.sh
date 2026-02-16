#!/bin/bash
# Colony Registry Exchange Module
# JSON/XML bidirectional conversion for colony registry
#
# Usage: source .aether/exchange/registry-xml.sh
#        xml-registry-export <registry_json> [output_xml]
#        xml-registry-import <registry_xml> [output_json]
#        xml-registry-lineage <registry_json> <colony_id>

# Source dependencies - handle being sourced vs executed
if [[ -n "${BASH_SOURCE[0]:-}" ]]; then
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
else
    SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
fi
source "$SCRIPT_DIR/utils/xml-core.sh"

# Ensure tool availability variables are available
if [[ -z "${XMLLINT_AVAILABLE:-}" ]]; then
    XMLLINT_AVAILABLE=false
    command -v xmllint > /dev/null 2>&1 && XMLLINT_AVAILABLE=true
fi
if [[ -z "${XMLSTARLET_AVAILABLE:-}" ]]; then
    XMLSTARLET_AVAILABLE=false
    command -v xmlstarlet > /dev/null 2>&1 && XMLSTARLET_AVAILABLE=true
fi

# ============================================================================
# Registry Export (JSON to XML)
# ============================================================================

# xml-registry-export: Convert colony registry JSON to XML
# Usage: xml-registry-export <registry_json_file> [output_xml_file]
# Returns: {"ok":true,"result":{"path":"...","colonies":N}}
xml-registry-export() {
    local json_file="${1:-}"
    local output_xml="${2:-}"

    [[ -z "$json_file" ]] && { xml_json_err "MISSING_ARG" "Missing JSON file argument"; return 1; }
    [[ -f "$json_file" ]] || { xml_json_err "FILE_NOT_FOUND" "JSON file not found: $json_file"; return 1; }

    # Validate JSON
    if ! jq empty "$json_file" 2>/dev/null; then
        xml_json_err "PARSE_ERROR" "Invalid JSON file: $json_file"
        return 1
    fi

    local version generated_at
    version=$(jq -r '.version // "1.0.0"' "$json_file")
    generated_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Build XML
    local xml_output
    xml_output="<?xml version=\"1.0\" encoding=\"UTF-8\"?>
<colony-registry xmlns=\"http://aether.colony/schemas/registry/1.0\"
                 version=\"$version\"
                 generated_at=\"$generated_at\">"

    # Process colonies
    local colony_count
    colony_count=$(jq '.colonies | length' "$json_file" 2>/dev/null || echo "0")

    local idx=0
    while [[ $idx -lt $colony_count ]]; do
        local colony
        colony=$(jq -c ".colonies[$idx]" "$json_file")

        local id name created_at status parent_id
        id=$(echo "$colony" | jq -r '.id')
        name=$(echo "$colony" | jq -r '.name // "Unnamed Colony"')
        created_at=$(echo "$colony" | jq -r '.created_at // "'"$generated_at"'"')
        status=$(echo "$colony" | jq -r '.status // "active"')
        parent_id=$(echo "$colony" | jq -r '.parent_id // empty')

        xml_output="$xml_output
  <colony id=\"$id\" status=\"$status\" created_at=\"$created_at\">"

        [[ -n "$parent_id" && "$parent_id" != "null" ]] && \
            xml_output="$xml_output
    <parent_id>$parent_id</parent_id>"

        xml_output="$xml_output
    <name>$(echo "$name" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g')</name>"

        # Add lineage if present
        local ancestors
        ancestors=$(echo "$colony" | jq -c '.ancestors // []')
        if [[ "$ancestors" != "[]" && -n "$ancestors" ]]; then
            xml_output="$xml_output
    <lineage>"
            local anc_count anc_idx
            anc_count=$(echo "$ancestors" | jq 'length')
            anc_idx=0
            while [[ $anc_idx -lt $anc_count ]]; do
                local ancestor_id
                ancestor_id=$(echo "$ancestors" | jq -r ".[$anc_idx]")
                xml_output="$xml_output
      <ancestor id=\"$ancestor_id\"/>
"
                ((anc_idx++))
            done
            xml_output="$xml_output
    </lineage>"
        fi

        xml_output="$xml_output
  </colony>"
        ((idx++))
    done

    xml_output="$xml_output
</colony-registry>"

    # Output
    if [[ -n "$output_xml" ]]; then
        echo "$xml_output" > "$output_xml"
        xml_json_ok "{\"path\":\"$output_xml\",\"colonies\":$colony_count}"
    else
        local escaped
        escaped=$(echo "$xml_output" | jq -Rs '.')
        xml_json_ok "{\"xml\":$escaped,\"colonies\":$colony_count}"
    fi
}

# ============================================================================
# Registry Import (XML to JSON) - Round-Trip
# ============================================================================

# xml-registry-import: Convert colony registry XML to JSON
# Usage: xml-registry-import <registry_xml_file> [output_json_file]
# Returns: {"ok":true,"result":{"colonies":N,"path":"..."}}
xml-registry-import() {
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
    local version generated_at
    if [[ "$XMLLINT_AVAILABLE" == "true" ]]; then
        version=$(xmllint --nonet --noent --xpath "string(/*/@version)" "$xml_file" 2>/dev/null || echo "1.0.0")
        generated_at=$(xmllint --nonet --noent --xpath "string(/*/@generated_at)" "$xml_file" 2>/dev/null || date -u +"%Y-%m-%dT%H:%M:%SZ")
    else
        version="1.0.0"
        generated_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    fi

    # Build JSON structure
    local json_output
    json_output=$(jq -n \
        --arg version "$version" \
        --arg generated_at "$generated_at" \
        '{
            version: $version,
            generated_at: $generated_at,
            colonies: []
        }')

    local colony_count=0

    # Extract colonies using xmlstarlet if available
    if [[ "$XMLSTARLET_AVAILABLE" == "true" ]]; then
        local colony_array
        colony_array=$(xmlstarlet sel -t -m "//colony" \
            -o '{"id":"' -v "@id" -o '","name":"' -v "name" -o '","status":"' -v "@status" -o '","created_at":"' -v "@created_at" -o '"}' \
            -n "$xml_file" 2>/dev/null | jq -s '.')

        colony_count=$(echo "$colony_array" | jq 'length')
        if [[ $colony_count -gt 0 ]]; then
            json_output=$(echo "$json_output" | jq --argjson colonies "$colony_array" '.colonies = $colonies')
        fi
    fi

    # Output
    if [[ -n "$output_json" ]]; then
        echo "$json_output" > "$output_json"
        xml_json_ok "{\"path\":\"$output_json\",\"colonies\":$colony_count}"
    else
        local escaped
        escaped=$(echo "$json_output" | jq -Rs '.')
        xml_json_ok "{\"json\":$escaped,\"colonies\":$colony_count}"
    fi
}

# ============================================================================
# Registry Validation
# ============================================================================

# xml-registry-validate: Validate registry XML against schema
# Usage: xml-registry-validate <registry_xml> [xsd_schema]
xml-registry-validate() {
    local xml_file="${1:-}"
    local xsd_file="${2:-.aether/schemas/colony-registry.xsd}"

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
# Lineage Queries
# ============================================================================

# xml-registry-lineage: Get ancestry chain for a colony
# Usage: xml-registry-lineage <registry_json> <colony_id>
# Returns: {"ok":true,"result":{"colony_id":"...","ancestors":[...],"depth":N}}
xml-registry-lineage() {
    local json_file="${1:-}"
    local colony_id="${2:-}"

    [[ -z "$json_file" ]] && { xml_json_err "MISSING_ARG" "Missing JSON file argument"; return 1; }
    [[ -z "$colony_id" ]] && { xml_json_err "MISSING_ARG" "Missing colony ID argument"; return 1; }
    [[ -f "$json_file" ]] || { xml_json_err "FILE_NOT_FOUND" "JSON file not found: $json_file"; return 1; }

    # Find the colony
    local colony
    colony=$(jq --arg id "$colony_id" '.colonies[] | select(.id == $id)' "$json_file")

    if [[ -z "$colony" || "$colony" == "null" ]]; then
        xml_json_err "COLONY_NOT_FOUND" "Colony not found in registry: $colony_id"
        return 1
    fi

    # Build ancestry chain
    local ancestors="[]"
    local current_id="$colony_id"
    local depth=0
    local max_depth=10  # Prevent infinite loops

    while [[ $depth -lt $max_depth ]]; do
        local parent_id
        parent_id=$(jq --arg id "$current_id" -r '.colonies[] | select(.id == $id) | .parent_id // empty' "$json_file")

        [[ -z "$parent_id" ]] && break

        ancestors=$(echo "$ancestors" | jq --arg parent "$parent_id" '. + [$parent]')
        current_id="$parent_id"
        ((depth++))
    done

    xml_json_ok "{\"colony_id\":\"$colony_id\",\"ancestors\":$ancestors,\"depth\":$depth}"
}

# Export functions
export -f xml-registry-export xml-registry-import xml-registry-validate xml-registry-lineage
