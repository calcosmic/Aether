#!/bin/bash
# Pheromone Exchange Module
# JSON/XML bidirectional conversion for pheromone signals
#
# Usage: source .aether/exchange/pheromone-xml.sh
#        xml-pheromone-export <pheromone_json> [output_xml]
#        xml-pheromone-import <pheromone_xml> [output_json]
#        xml-pheromone-validate <pheromone_xml>
#        xml-pheromone-merge <colony_prefix> <xml_files...> [output_xml]

# Don't use set -e for library-style scripts - let callers handle errors
# set -euo pipefail

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
    command -v xmllint >/dev/null 2>&1 && XMLLINT_AVAILABLE=true
fi
if [[ -z "${XMLSTARLET_AVAILABLE:-}" ]]; then
    XMLSTARLET_AVAILABLE=false
    command -v xmlstarlet >/dev/null 2>&1 && XMLSTARLET_AVAILABLE=true
fi

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
        # Handle both string content (.content = "text") and object content (.content.text = "text")
        local content_text
        content_text=$(echo "$signal" | jq -r 'if (.content | type) == "string" then .content elif .content.text then .content.text else .message // "" end')
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
# Pheromone Import (XML to JSON)
# ============================================================================

# xml-pheromone-import: Convert pheromone XML back to JSON
# Usage: xml-pheromone-import <pheromone_xml_file> [output_json_file]
# Returns: {"ok":true,"result":{"json":"...","signals":N,"path":"..."}}
xml-pheromone-import() {
    local xml_file="${1:-}"
    local output_json="${2:-}"
    local preserve_prefixes="${3:-false}"

    [[ -z "$xml_file" ]] && { xml_json_err "MISSING_ARG" "Missing XML file argument"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "FILE_NOT_FOUND" "XML file not found: $xml_file"; return 1; }

    # Check well-formedness
    xmllint --nonet --noent --noout "$xml_file" 2>/dev/null || {
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

    # Build JSON structure
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

    # Extract signals - use xmlstarlet if available, fallback to grep/awk
    if [[ "$XMLSTARLET_AVAILABLE" == "true" ]]; then
        # Use xmlstarlet for proper namespace handling
        local signals_json
        signals_json=$(xmlstarlet sel -N ph="http://aether.colony/schemas/pheromones" \
            -t -m "//ph:signal" \
            -o '{"id":"' -v "@id" -o '","type":"' -v "@type" -o '","priority":"' -v "@priority" -o '","source":"' -v "@source" -o '","created_at":"' -v "@created_at" -o '","active":' -v "@active" -o '}' \
            -n "$xml_file" 2>/dev/null | jq -s '.')

        # Extract content text for each signal
        local idx=0
        local enriched_signals="[]"
        while true; do
            local content_text
            content_text=$(xmlstarlet sel -N ph="http://aether.colony/schemas/pheromones" \
                -t -v "//ph:signal[$idx + 1]/ph:content/ph:text" "$xml_file" 2>/dev/null || echo "")

            if [[ -z "$content_text" && $idx -gt 0 ]]; then
                break
            fi

            # Add content to signal
            enriched_signals=$(echo "$signals_json" | jq --arg idx "$idx" --arg text "$content_text" '
                .[$idx | tonumber] |= . + {content: {text: $text}}'
            )

            ((idx++))
            [[ $idx -gt 100 ]] && break  # Safety limit
        done

        # Merge signals into output
        if [[ "$enriched_signals" != "[]" ]]; then
            json_output=$(echo "$json_output" | jq --argjson signals "$enriched_signals" '.signals = $signals')
        else
            json_output=$(echo "$json_output" | jq --argjson signals "$signals_json" '.signals = $signals')
        fi
    else
        # Fallback: basic extraction with grep/sed
        local fallback_signals="[]"
        while IFS= read -r line; do
            if [[ "$line" =~ id=\"([^\"]+)\" ]]; then
                local sid="${BASH_REMATCH[1]}"
                local stype="FOCUS"
                local spriority="normal"

                # Try to extract type and priority
                if [[ "$line" =~ type=\"([^\"]+)\" ]]; then
                    stype="${BASH_REMATCH[1]}"
                fi
                if [[ "$line" =~ priority=\"([^\"]+)\" ]]; then
                    spriority="${BASH_REMATCH[1]}"
                fi

                # Remove namespace prefix if not preserving
                if [[ "$preserve_prefixes" != "true" ]]; then
                    sid=$(echo "$sid" | sed 's/^[^:]*://')
                fi

                fallback_signals=$(echo "$fallback_signals" | jq \
                    --arg id "$sid" \
                    --arg type "$stype" \
                    --arg priority "$spriority" \
                    '. + [{id: $id, type: $type, priority: $priority, source: "xml-import", created_at: "'"$generated_at"'", active: true}]')
            fi
        done < <(grep '<signal' "$xml_file")

        json_output=$(echo "$json_output" | jq --argjson signals "$fallback_signals" '.signals = $signals')
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
# Pheromone Merge (Multiple Colonies)
# ============================================================================

# xml-pheromone-merge: Merge pheromone XML from multiple colonies
# Usage: xml-pheromone-merge <output_xml> <input_xml_files...>
# Options:
#   --namespace <prefix>  - Add colony prefix to signal IDs (default: auto-generate from colony_id)
#   --deduplicate         - Remove duplicate signals by ID (default: true)
#   --target <path>       - Target output file (default: ~/.aether/eternal/pheromones.xml)
# Returns: {"ok":true,"result":{"path":"...","signals":N,"colonies":M}}
xml-pheromone-merge() {
    local output_file=""
    local namespace_prefix=""
    local deduplicate=true
    local input_files=()
    local arg_idx=0

    # Parse arguments
    while [[ $arg_idx -lt $# ]]; do
        local arg="${*:$((arg_idx + 1)):1}"
        case "$arg" in
            --namespace)
                namespace_prefix="${*:$((arg_idx + 2)):1}"
                ((arg_idx += 2))
                ;;
            --no-deduplicate)
                deduplicate=false
                ((arg_idx++))
                ;;
            --deduplicate)
                deduplicate=true
                ((arg_idx++))
                ;;
            --target)
                output_file="${*:$((arg_idx + 2)):1}"
                ((arg_idx += 2))
                ;;
            -*)
                xml_json_err "INVALID_ARG" "Unknown option: $arg"
                return 1
                ;;
            *)
                if [[ -z "$output_file" ]]; then
                    output_file="$arg"
                else
                    input_files+=("$arg")
                fi
                ((arg_idx++))
                ;;
        esac
    done

    # Default output path
    [[ -z "$output_file" ]] && output_file="${HOME}/.aether/eternal/pheromones.xml"

    # Validate input files
    if [[ ${#input_files[@]} -eq 0 ]]; then
        xml_json_err "MISSING_ARG" "No input XML files specified"
        return 1
    fi

    for file in "${input_files[@]}"; do
        [[ -f "$file" ]] || { xml_json_err "FILE_NOT_FOUND" "Input file not found: $file"; return 1; }
    done

    # Ensure output directory exists
    mkdir -p "$(dirname "$output_file")"

    # Generate merged XML
    local generated_at
    generated_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    local merged_xml="<?xml version=\"1.0\" encoding=\"UTF-8\"?>
<pheromones xmlns=\"http://aether.colony/schemas/pheromones\"
            xmlns:ph=\"http://aether.colony/schemas/pheromones\"
            version=\"1.0.0\"
            generated_at=\"$generated_at\"
            colony_id=\"merged\">"

    merged_xml="$merged_xml
  <metadata>
    <source type=\"system\">aether-pheromone-merge</source>
    <context>Merged pheromones from ${#input_files[@]} colonies</context>
  </metadata>"

    # Track unique signal IDs for deduplication
    declare -A seen_ids
    local total_signals=0
    local unique_signals=0
    local colonies_merged=0

    for input_file in "${input_files[@]}"; do
        # Get colony ID from source file
        local source_colony_id
        if [[ "$XMLLINT_AVAILABLE" == "true" ]]; then
            source_colony_id=$(xmllint --nonet --noent --xpath "string(/*/@colony_id)" "$input_file" 2>/dev/null || echo "colony_$colonies_merged")
        else
            source_colony_id=$(grep -o 'colony_id="[^"]*"' "$input_file" | head -1 | cut -d'"' -f2 || echo "colony_$colonies_merged")
        fi

        # Determine prefix for this colony
        local colony_prefix
        if [[ -n "$namespace_prefix" ]]; then
            colony_prefix="$namespace_prefix"
        else
            colony_prefix="$source_colony_id"
        fi

        # Extract signals from this file using xmlstarlet
        if [[ "$XMLSTARLET_AVAILABLE" == "true" ]]; then
            # Get signal count
            local sig_count
            sig_count=$(xmlstarlet sel -N ph="http://aether.colony/schemas/pheromones" \
                -t -v "count(//ph:signal)" "$input_file" 2>/dev/null || echo "0")

            local idx=1
            while [[ $idx -le $sig_count ]]; do
                # Extract full signal element
                local sig_xml
                sig_xml=$(xmlstarlet sel -N ph="http://aether.colony/schemas/pheromones" \
                    -t -c "//ph:signal[$idx]" "$input_file" 2>/dev/null)

                if [[ -n "$sig_xml" ]]; then
                    # Extract original ID
                    local orig_id
                    orig_id=$(echo "$sig_xml" | grep -oE 'id="[^"]+"' | head -1 | cut -d'"' -f2)
                    local new_id="${colony_prefix}:${orig_id}"

                    ((total_signals++))

                    # Check for duplicates
                    if [[ "$deduplicate" == "true" ]]; then
                        if [[ -n "${seen_ids[$new_id]:-}" ]]; then
                            ((idx++))
                            continue
                        fi
                        seen_ids[$new_id]=1
                    fi

                    # Replace ID with prefixed version
                    sig_xml=$(echo "$sig_xml" | sed "s/id=\"$orig_id\"/id=\"$new_id\"/")

                    # Add signal to merged XML
                    merged_xml="$merged_xml
  $sig_xml"
                    ((unique_signals++))
                fi
                ((idx++))
            done
        else
            # Fallback: extract with awk and sed
            local sig_block=""
            local in_signal=false
            while IFS= read -r line; do
                if echo "$line" | grep -qE '^[[:space:]]*<signal[[:space:]]'; then
                    in_signal=true
                    sig_block="$line"
                    # Extract and prefix ID
                    if [[ "$line" =~ id=\"([^\"]+)\" ]]; then
                        local orig_id="${BASH_REMATCH[1]}"
                        local new_id="${colony_prefix}:${orig_id}"
                        line=$(echo "$line" | sed "s/id=\"$orig_id\"/id=\"$new_id\"/")
                    fi
                elif [[ "$in_signal" == true ]]; then
                    sig_block="$sig_block
$line"
                    if echo "$line" | grep -qE '</signal>'; then
                        # Process complete signal
                        local signal_id
                        if [[ "$sig_block" =~ id=\"([^\"]+)\" ]]; then
                            signal_id="${BASH_REMATCH[1]}"
                            ((total_signals++))

                            # Check for duplicates
                            if [[ "$deduplicate" == "true" ]]; then
                                if [[ -n "${seen_ids[$signal_id]:-}" ]]; then
                                    in_signal=false
                                    sig_block=""
                                    continue
                                fi
                                seen_ids[$signal_id]=1
                            fi

                            # Add signal to merged XML
                            merged_xml="$merged_xml
  $sig_block"
                            ((unique_signals++))
                        fi
                        in_signal=false
                        sig_block=""
                    fi
                fi
            done < "$input_file"
        fi

        ((colonies_merged++))
    done

    # Close root element
    merged_xml="$merged_xml
</pheromones>"

    # Write output
    echo "$merged_xml" > "$output_file"

    xml_json_ok "{\"path\":\"$output_file\",\"signals\":$unique_signals,\"colonies\":$colonies_merged,\"duplicates_removed\":$((total_signals - unique_signals))}"
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
export -f xml-pheromone-export xml-pheromone-import xml-pheromone-validate xml-pheromone-merge
export -f xml-pheromone-prefix-id xml-pheromone-deprefix-id
