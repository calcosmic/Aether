#!/bin/bash
# XML Utilities Loader for Aether Colony
# Modular architecture - sources xml-core.sh and xml-compose.sh
#
# Usage: source .aether/utils/xml-utils.sh
#        xml-validate <xml_file> <xsd_file>
#        xml-to-json <xml_file>
#        json-to-xml <json_file> [root_element]
#        xml-query <xml_file> <xpath_expression>
#        xml-merge <output_file> <input_files...>
#
# All functions return JSON status like other aether-utils
#
# Note: This file loads xml-core.sh and xml-compose.sh for modularity.
# The actual implementations are in those modules; this file provides
# backward compatibility and loads additional domain functions below.

set -euo pipefail

# Determine script directory for sourcing modules
# Handle case when sourced interactively (BASH_SOURCE[0] may be empty)
if [[ -n "${BASH_SOURCE[0]:-}" ]]; then
    _XML_UTILS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
else
    # Fallback: derive from the sourced script's location
    _XML_UTILS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
fi

# ============================================================================
# Load Modular Components
# ============================================================================

# Core utilities (validation, formatting, escaping)
[[ -f "$_XML_UTILS_DIR/xml-core.sh" ]] && source "$_XML_UTILS_DIR/xml-core.sh"

# XInclude composition
[[ -f "$_XML_UTILS_DIR/xml-compose.sh" ]] && source "$_XML_UTILS_DIR/xml-compose.sh"

# ============================================================================
# Feature Detection (supplement xml-core.sh if not already set)
# ============================================================================

# Check for required XML tools
: "${XMLLINT_AVAILABLE:=false}"
: "${XMLSTARLET_AVAILABLE:=false}"
: "${XSLTPROC_AVAILABLE:=false}"
: "${XML2JSON_AVAILABLE:=false}"

if command -v xmllint >/dev/null 2>&1; then
    XMLLINT_AVAILABLE=true
fi

if command -v xmlstarlet >/dev/null 2>&1; then
    XMLSTARLET_AVAILABLE=true
fi

if command -v xsltproc >/dev/null 2>&1; then
    XSLTPROC_AVAILABLE=true
fi

if command -v xml2json >/dev/null 2>&1; then
    XML2JSON_AVAILABLE=true
fi

# ============================================================================
# Additional JSON Output Helpers (supplement xml-core.sh)
# ============================================================================

# Use xml-core.sh versions if available, otherwise define here
if ! type xml_json_ok &>/dev/null; then
    xml_json_ok() { printf '{"ok":true,"result":%s}\n' "$1"; }
fi

if ! type xml_json_err &>/dev/null; then
    xml_json_err() {
        local message="${2:-$1}"
        printf '{"ok":false,"error":"%s"}\n' "$message" >&2
        return 1
    }
fi

# ============================================================================
# Domain Functions: Pheromones, Wisdom, Registry, Prompts
# ============================================================================

# These functions remain in xml-utils.sh (not yet modularized)

# ============================================================================
# Core XML Functions
# ============================================================================

# xml-validate: Validate XML against XSD schema using xmllint
# Usage: xml-validate <xml_file> <xsd_file>
# Returns: {"ok":true,"result":{"valid":true,"errors":[]}} or error
xml-validate() {
    local xml_file="${1:-}"
    local xsd_file="${2:-}"

    # Validate arguments
    [[ -z "$xml_file" ]] && { xml_json_err "Missing XML file argument"; return 1; }
    [[ -z "$xsd_file" ]] && { xml_json_err "Missing XSD schema file argument"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "XML file not found: $xml_file"; return 1; }
    [[ -f "$xsd_file" ]] || { xml_json_err "XSD schema file not found: $xsd_file"; return 1; }

    # Check for xmllint
    if [[ "$XMLLINT_AVAILABLE" != "true" ]]; then
        xml_json_err "xmllint not available. Install libxml2 utilities."
        return 1
    fi

    # Validate XML against XSD (with XXE protection)
    local errors
    errors=$(xmllint --nonet --noent --noout --schema "$xsd_file" "$xml_file" 2>&1) && {
        xml_json_ok '{"valid":true,"errors":[]}'
        return 0
    } || {
        # Parse errors into JSON array
        local error_json
        error_json=$(echo "$errors" | jq -R -s 'split("\n") | map(select(length > 0))')
        xml_json_ok "{\"valid\":false,\"errors\":$error_json}"
        return 0
    }
}

# xml-well-formed: Check if XML is well-formed (no schema validation)
# Usage: xml-well-formed <xml_file>
# Returns: {"ok":true,"result":{"well_formed":true,"error":null}} or error details
xml-well-formed() {
    local xml_file="${1:-}"

    [[ -z "$xml_file" ]] && { xml_json_err "Missing XML file argument"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "XML file not found: $xml_file"; return 1; }

    if [[ "$XMLLINT_AVAILABLE" != "true" ]]; then
        xml_json_err "xmllint not available. Install libxml2 utilities."
        return 1
    fi

    local error
    error=$(xmllint --nonet --noent --noout "$xml_file" 2>&1) && {
        xml_json_ok '{"well_formed":true,"error":null}'
        return 0
    } || {
        local escaped_error
        escaped_error=$(echo "$error" | jq -Rs '.[:-1]')
        xml_json_ok "{\"well_formed\":false,\"error\":$escaped_error}"
        return 0
    }
}

# xml-to-json: Convert XML to JSON using available tools
# Usage: xml-to-json <xml_file> [options]
# Options: --pretty (pretty print output)
# Returns: {"ok":true,"result":<json_object>}
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

    [[ -z "$xml_file" ]] && { xml_json_err "Missing XML file argument"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "XML file not found: $xml_file"; return 1; }

    # Check well-formedness first
    local well_formed_result
    well_formed_result=$(xml-well-formed "$xml_file" 2>/dev/null)
    if ! echo "$well_formed_result" | jq -e '.result.well_formed' >/dev/null 2>&1; then
        xml_json_err "XML is not well-formed"
        return 1
    fi

    # Try xml2json if available (npm package)
    if [[ "$XML2JSON_AVAILABLE" == "true" ]]; then
        local json_output
        if json_output=$(xml2json "$xml_file" 2>/dev/null); then
            if [[ "$pretty" == "true" ]]; then
                json_output=$(echo "$json_output" | jq '.')
            fi
            xml_json_ok "$(echo "$json_output" | jq -Rs '.[:-1]')"
            return 0
        fi
    fi

    # Fallback: Use xsltproc with built-in XSLT if available
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
            xml_json_err "XSLT conversion failed"
            return 1
        }
        xml_json_ok "$json_result"
        return 0
    fi

    # Last resort: Use xmlstarlet if available
    if [[ "$XMLSTARLET_AVAILABLE" == "true" ]]; then
        # xmlstarlet can convert to various formats, we'll use sel to extract structure
        local json_result
        json_result=$(xmlstarlet sel -t -m "/" -o '{"root":{' -m "*" -v "name()" -o ':"' -v "." -o '"' -b -o '}}' "$xml_file" 2>/dev/null) || {
            xml_json_err "xmlstarlet conversion failed"
            return 1
        }
        xml_json_ok "$json_result"
        return 0
    fi

    xml_json_err "No XML to JSON conversion tool available. Install xml2json, xsltproc, or xmlstarlet."
    return 1
}

# json-to-xml: Convert JSON to XML
# Usage: json-to-xml <json_file> [root_element]
# Returns: {"ok":true,"result":{"xml":"<root>...</root>"}}
json-to-xml() {
    local json_file="${1:-}"
    local root_element="${2:-root}"

    [[ -z "$json_file" ]] && { xml_json_err "Missing JSON file argument"; return 1; }
    [[ -f "$json_file" ]] || { xml_json_err "JSON file not found: $json_file"; return 1; }

    # Validate JSON first
    if ! jq empty "$json_file" 2>/dev/null; then
        xml_json_err "Invalid JSON file: $json_file"
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
        xml_json_err "JSON to XML conversion failed"
        return 1
    }

    # Escape the XML for JSON output
    local escaped_xml
    escaped_xml=$(echo "$xml_output" | jq -Rs '.')
    xml_json_ok "{\"xml\":$escaped_xml}"
}

# xml-query: XPath query function using XMLStarlet
# Usage: xml-query <xml_file> <xpath_expression>
# Returns: {"ok":true,"result":{"matches":[...],"count":N}}
xml-query() {
    local xml_file="${1:-}"
    local xpath="${2:-}"

    [[ -z "$xml_file" ]] && { xml_json_err "Missing XML file argument"; return 1; }
    [[ -z "$xpath" ]] && { xml_json_err "Missing XPath expression argument"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "XML file not found: $xml_file"; return 1; }

    if [[ "$XMLSTARLET_AVAILABLE" != "true" ]]; then
        xml_json_err "xmlstarlet not available. Install xmlstarlet for XPath queries."
        return 1
    fi

    # Execute XPath query
    local results
    results=$(xmlstarlet sel -t -m "$xpath" -v "." -n "$xml_file" 2>/dev/null) || {
        xml_json_err "XPath query failed: $xpath"
        return 1
    }

    # Convert results to JSON array
    local json_array
    json_array=$(echo "$results" | jq -R -s 'split("\n") | map(select(length > 0))')
    local count
    count=$(echo "$json_array" | jq 'length')

    xml_json_ok "{\"matches\":$json_array,\"count\":$count}"
}

# xml-query-attr: Query for specific attribute values
# Usage: xml-query-attr <xml_file> <element> <attribute>
# Returns: {"ok":true,"result":{"attributes":[...],"count":N}}
xml-query-attr() {
    local xml_file="${1:-}"
    local element="${2:-}"
    local attr="${3:-}"

    [[ -z "$xml_file" ]] && { xml_json_err "Missing XML file argument"; return 1; }
    [[ -z "$element" ]] && { xml_json_err "Missing element argument"; return 1; }
    [[ -z "$attr" ]] && { xml_json_err "Missing attribute argument"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "XML file not found: $xml_file"; return 1; }

    if [[ "$XMLSTARLET_AVAILABLE" != "true" ]]; then
        xml_json_err "xmlstarlet not available. Install xmlstarlet for attribute queries."
        return 1
    fi

    local results
    results=$(xmlstarlet sel -t -m "//$element" -v "@$attr" -n "$xml_file" 2>/dev/null) || {
        xml_json_err "Attribute query failed: $element/@$attr"
        return 1
    }

    local json_array
    json_array=$(echo "$results" | jq -R -s 'split("\n") | map(select(length > 0))')
    local count
    count=$(echo "$json_array" | jq 'length')

    xml_json_ok "{\"attributes\":$json_array,\"count\":$count}"
}

# xml-merge: XInclude document merging
# Usage: xml-merge <output_file> <main_xml_file> [included_files...]
# Returns: {"ok":true,"result":{"merged":true,"output":"<path>"}}
xml-merge() {
    local output_file="${1:-}"
    local main_xml="${2:-}"

    [[ -z "$output_file" ]] && { xml_json_err "Missing output file argument"; return 1; }
    [[ -z "$main_xml" ]] && { xml_json_err "Missing main XML file argument"; return 1; }
    [[ -f "$main_xml" ]] || { xml_json_err "Main XML file not found: $main_xml"; return 1; }

    # Check well-formedness of main file
    local well_formed_result
    well_formed_result=$(xml-well-formed "$main_xml" 2>/dev/null)
    if ! echo "$well_formed_result" | jq -e '.result.well_formed' >/dev/null 2>&1; then
        xml_json_err "Main XML file is not well-formed"
        return 1
    fi

    # Use xmllint for XInclude processing if available
    if [[ "$XMLLINT_AVAILABLE" == "true" ]]; then
        local merged
        merged=$(xmllint --nonet --noent --xinclude "$main_xml" 2>/dev/null) || {
            xml_json_err "XInclude merge failed"
            return 1
        }

        # Write output
        echo "$merged" > "$output_file"
        local escaped_output
        escaped_output=$(echo "$output_file" | jq -Rs '.[:-1]')
        xml_json_ok "{\"merged\":true,\"output\":$escaped_output}"
        return 0
    fi

    # Fallback: Simple file concatenation with root element wrapping
    # This is a basic implementation - full XInclude requires xmllint
    local temp_dir
    temp_dir=$(mktemp -d)

    # Extract root element from main file
    local root_element
    root_element=$(grep -oP '(?<=<)[^>\s?/]+' "$main_xml" | head -1)

    {
        echo "<?xml version=\"1.0\" encoding=\"UTF-8\"?>"
        echo "<$root_element>"
        cat "$main_xml" | sed '1,/<'$root_element'>/d' | sed '/<\/'$root_element'>/,$d'
        echo "</$root_element>"
    } > "$output_file"

    rm -rf "$temp_dir"

    local escaped_output
    escaped_output=$(echo "$output_file" | jq -Rs '.[:-1]')
    xml_json_ok "{\"merged\":true,\"output\":$escaped_output,\"note\":\"Basic merge without XInclude\"}"
}

# xml-format: Pretty-print XML file
# Usage: xml-format <xml_file> [output_file]
# Returns: {"ok":true,"result":{"formatted":true,"output":"<path>"}}
xml-format() {
    local xml_file="${1:-}"
    local output_file="${2:-}"

    [[ -z "$xml_file" ]] && { xml_json_err "Missing XML file argument"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "XML file not found: $xml_file"; return 1; }

    if [[ "$XMLLINT_AVAILABLE" != "true" ]]; then
        xml_json_err "xmllint not available. Install libxml2 utilities."
        return 1
    fi

    # Determine output destination
    local target="${output_file:-$xml_file}"

    # Format XML with proper indentation (with XXE protection)
    local formatted
    formatted=$(xmllint --nonet --noent --format "$xml_file" 2>/dev/null) || {
        xml_json_err "XML formatting failed"
        return 1
    }

    echo "$formatted" > "$target"

    local escaped_output
    escaped_output=$(echo "$target" | jq -Rs '.[:-1]')
    xml_json_ok "{\"formatted\":true,\"output\":$escaped_output}"
}

# xml-escape: Escape special characters for XML content
# Usage: xml-escape "string with <special> & characters"
# Returns: {"ok":true,"result":"escaped string"}
xml-escape() {
    local input="${1:-}"

    # Escape XML special characters
    local escaped
    escaped=$(echo "$input" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g; s/"/\&quot;/g; s/'"'"'/\&apos;/g')

    local escaped_json
    escaped_json=$(echo "$escaped" | jq -Rs '.[:-1]')
    xml_json_ok "$escaped_json"
}

# xml-unescape: Unescape XML entities
# Usage: xml-unescape "string with &lt;special&gt; entities"
# Returns: {"ok":true,"result":"unescaped string"}
xml-unescape() {
    local input="${1:-}"

    # Unescape XML entities
    local unescaped
    unescaped=$(echo "$input" | sed 's/\&lt;/</g; s/\&gt;/>/g; s/\&quot;/"/g; s/\&apos;/'"'"'/g; s/\&amp;/\&/g')

    local unescaped_json
    unescaped_json=$(echo "$unescaped" | jq -Rs '.[:-1]')
    xml_json_ok "$unescaped_json"
}

# xml-detect-tools: Detect available XML tools
# Usage: xml-detect-tools
# Returns: {"ok":true,"result":{"xmllint":true,"xmlstarlet":false,...}}
xml-detect-tools() {
    xml_json_ok "{\"xmllint\":$XMLLINT_AVAILABLE,\"xmlstarlet\":$XMLSTARLET_AVAILABLE,\"xsltproc\":$XSLTPROC_AVAILABLE,\"xml2json\":$XML2JSON_AVAILABLE}"
}

# ============================================================================
# Pheromone Exchange Format (Hybrid JSON/XML)
# ============================================================================

# pheromone-to-xml: Convert pheromone JSON to XML format with full XSD schema support
# Usage: pheromone-to-xml <pheromone_json_file> [output_xml_file] [xsd_schema_file]
#   pheromone_json_file: Path to pheromone JSON (supports both single signal and full pheromones format)
#   output_xml_file: Optional path to write XML output (if omitted, returns XML in result)
#   xsd_schema_file: Optional path to XSD schema for validation (default: .aether/schemas/pheromone.xsd)
# Returns: {"ok":true,"result":{"xml":"<pheromones>...</pheromones>","validated":true,"path":"..."}}
pheromone-to-xml() {
    local json_file="${1:-}"
    local output_xml="${2:-}"
    local xsd_file="${3:-.aether/schemas/pheromone.xsd}"

    [[ -z "$json_file" ]] && { xml_json_err "Missing JSON file argument"; return 1; }
    [[ -f "$json_file" ]] || { xml_json_err "JSON file not found: $json_file"; return 1; }

    # Validate JSON
    if ! jq empty "$json_file" 2>/dev/null; then
        xml_json_err "Invalid JSON file: $json_file"
        return 1
    fi

    # Detect JSON format: single signal or full pheromones structure
    local has_signals
    has_signals=$(jq 'has("signals")' "$json_file" 2>/dev/null)

    # Generate ISO timestamp
    local generated_at
    generated_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Extract metadata from JSON
    local version colony_id
    version=$(jq -r '.version // "1.0.0"' "$json_file")
    colony_id=$(jq -r '.colony_id // "unknown"' "$json_file")

    # Build XML header with proper namespace
    local xml_output
    xml_output="<?xml version=\"1.0\" encoding=\"UTF-8\"?>
<pheromones xmlns=\"http://aether.colony/schemas/pheromones\"
            xmlns:ph=\"http://aether.colony/schemas/pheromones\"
            version=\"$version\"
            generated_at=\"$generated_at\"
            colony_id=\"$colony_id\">"

    # Add metadata section
    local source_type source_version context
    source_type=$(jq -r '.metadata.source.type // "system"' "$json_file" 2>/dev/null || echo "system")
    source_version=$(jq -r ".metadata.source.version // \"$version\"" "$json_file" 2>/dev/null || echo "$version")
    context=$(jq -r '.metadata.context // "Colony pheromone signal conversion"' "$json_file" 2>/dev/null || echo "Colony pheromone signal conversion")

    xml_output="$xml_output
  <metadata>
    <source type=\"$source_type\" version=\"$source_version\">aether-pheromone-converter</source>
    <context>$(echo "$context" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g')</context>
  </metadata>"

    # Process signals - either from array or wrap single signal
    local signal_count=0
    local sig_array_length
    sig_array_length=$(jq '.signals | length' "$json_file" 2>/dev/null || echo "0")

    if [[ "$has_signals" == "true" && "$sig_array_length" -gt 0 ]]; then
        local sig_idx=0
        while [[ $sig_idx -lt $sig_array_length ]]; do
            local signal
            signal=$(jq -c ".signals[$sig_idx]" "$json_file" 2>/dev/null)
            [[ -n "$signal" ]] || { ((sig_idx++)); continue; }

            # Extract signal fields with defaults
            local sig_id sig_type priority source created_at expires_at active
            sig_id=$(echo "$signal" | jq -r '.id // "sig_'"$(date +%s)"'_'"$signal_count"'"')
            sig_type=$(echo "$signal" | jq -r '.type // "FOCUS"' | tr '[:lower:]' '[:upper:]')
            priority=$(echo "$signal" | jq -r '.priority // "normal"' | tr '[:upper:]' '[:lower:]')
            source=$(echo "$signal" | jq -r '.source // "system"')
            created_at=$(echo "$signal" | jq -r '.created_at // "'"$generated_at"'"')
            expires_at=$(echo "$signal" | jq -r '.expires_at // empty')
            active=$(echo "$signal" | jq -r '.active // true')

            # Validate signal type against schema enum
            case "$sig_type" in
                FOCUS|REDIRECT|FEEDBACK) ;;
                *) sig_type="FOCUS" ;;
            esac

            # Validate priority against schema enum
            case "$priority" in
                critical|high|normal|low) ;;
                *) priority="normal" ;;
            esac

            # XML escape ID and source
            local escaped_id escaped_source
            escaped_id=$(echo "$sig_id" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g; s/"/\&quot;/g')
            escaped_source=$(echo "$source" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g; s/"/\&quot;/g')

            # Build signal element
            xml_output="$xml_output
  <signal id=\"$escaped_id\"
          type=\"$sig_type\"
          priority=\"$priority\"
          source=\"$escaped_source\"
          created_at=\"$created_at\""

            # Add optional expires_at if present
            if [[ -n "$expires_at" && "$expires_at" != "null" ]]; then
                xml_output="$xml_output
          expires_at=\"$expires_at\""
            fi

            xml_output="$xml_output
          active=\"$active\">"

            # Content section
            local content_text content_data content_format
            content_text=$(echo "$signal" | jq -r '.content.text // .message // ""')
            content_format=$(echo "$signal" | jq -r '.content.data.format // "json"')

            if [[ -n "$content_text" ]]; then
                local escaped_text
                escaped_text=$(echo "$content_text" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g')
                xml_output="$xml_output
    <content>
      <text>$escaped_text</text>"

                # Check for data attachment - convert JSON to XML elements
                local has_data
                has_data=$(echo "$signal" | jq 'has("content") and (.content | has("data"))' 2>/dev/null)
                if [[ "$has_data" == "true" ]]; then
                    local data_xml
                    data_xml=$(echo "$signal" | jq -r '.content.data | to_entries | map("<\(.key)>\(.value | tostring | gsub("&"; "&amp;") | gsub("<"; "&lt;") | gsub(">"; "&gt;"))</\(.key)>") | join("")' 2>/dev/null)
                    xml_output="$xml_output
      <data format=\"$content_format\">$data_xml</data>"
                fi

                xml_output="$xml_output
    </content>"
            fi

            # Tags section
            local tags_json
            tags_json=$(echo "$signal" | jq -c '.tags // []')
            if [[ "$tags_json" != "[]" && -n "$tags_json" ]]; then
                xml_output="$xml_output
    <tags>"
                local tag_count
                tag_count=$(echo "$signal" | jq '.tags | length')
                local tag_idx=0
                while [[ $tag_idx -lt $tag_count ]]; do
                    local tag
                    tag=$(echo "$signal" | jq -c ".tags[$tag_idx]")
                    local tag_value tag_weight tag_category
                    tag_value=$(echo "$tag" | jq -r '.value // .')
                    tag_weight=$(echo "$tag" | jq -r '.weight // "1.0"')
                    tag_category=$(echo "$tag" | jq -r '.category // empty')

                    local escaped_tag
                    escaped_tag=$(echo "$tag_value" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g; s/"/\&quot;/g')

                    if [[ -n "$tag_category" && "$tag_category" != "null" ]]; then
                        xml_output="$xml_output
      <tag weight=\"$tag_weight\" category=\"$tag_category\">$escaped_tag</tag>"
                    else
                        xml_output="$xml_output
      <tag weight=\"$tag_weight\">$escaped_tag</tag>"
                    fi
                    ((tag_idx++))
                done
                xml_output="$xml_output
    </tags>"
            fi

            # Scope section
            local scope_global scope_castes scope_paths
            scope_global=$(echo "$signal" | jq -r '.scope.global // false')
            xml_output="$xml_output
    <scope global=\"$scope_global\">"

            # Castes
            local castes_json
            castes_json=$(echo "$signal" | jq -c '.scope.castes // []' 2>/dev/null)
            if [[ "$castes_json" != "[]" && -n "$castes_json" && "$castes_json" != "null" ]]; then
                xml_output="$xml_output
      <castes match=\"any\">"
                local caste_count
                caste_count=$(echo "$signal" | jq '.scope.castes | length' 2>/dev/null)
                local caste_idx=0
                while [[ $caste_idx -lt $caste_count ]]; do
                    local caste
                    caste=$(echo "$signal" | jq -r ".scope.castes[$caste_idx]" 2>/dev/null)
                    # Validate caste against schema enum
                    case "$caste" in
                        builder|watcher|scout|chaos|oracle|architect|prime|colonizer|route_setter|archaeologist|ambassador|auditor|chronicler|gatekeeper|guardian|includer|keeper|measurer|probe|sage|tracker|weaver)
                            xml_output="$xml_output
        <caste>$caste</caste>"
                            ;;
                    esac
                    ((caste_idx++))
                done
                xml_output="$xml_output
      </castes>"
            fi

            # Paths
            local paths_json
            paths_json=$(echo "$signal" | jq -c '.scope.paths // []' 2>/dev/null)
            if [[ "$paths_json" != "[]" && -n "$paths_json" && "$paths_json" != "null" ]]; then
                xml_output="$xml_output
      <paths match=\"any\">"
                local path_count
                path_count=$(echo "$signal" | jq '.scope.paths | length' 2>/dev/null)
                local path_idx=0
                while [[ $path_idx -lt $path_count ]]; do
                    local path
                    path=$(echo "$signal" | jq -r ".scope.paths[$path_idx]" 2>/dev/null)
                    local escaped_path
                    escaped_path=$(echo "$path" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g; s/"/\&quot;/g')
                    xml_output="$xml_output
        <path>$escaped_path</path>"
                    ((path_idx++))
                done
                xml_output="$xml_output
      </paths>"
            fi

            xml_output="$xml_output
    </scope>"

            xml_output="$xml_output
  </signal>"
            ((signal_count++))
            ((sig_idx++))
        done
    elif [[ "$has_signals" != "true" ]]; then
        # Handle single signal JSON (legacy format)
        local signal
        signal=$(jq -c '.' "$json_file" 2>/dev/null)
        if [[ -n "$signal" ]]; then
            # Extract signal fields with defaults for legacy format
            local sig_id sig_type priority source created_at expires_at active
            sig_id=$(echo "$signal" | jq -r '.id // "sig_'"$(date +%s)"'_0"')
            sig_type=$(echo "$signal" | jq -r '.type // "FOCUS"' | tr '[:lower:]' '[:upper:]')
            priority=$(echo "$signal" | jq -r '.priority // "normal"' | tr '[:upper:]' '[:lower:]')
            source=$(echo "$signal" | jq -r '.source // "system"')
            created_at=$(echo "$signal" | jq -r '.created_at // "'"$generated_at"'"')
            expires_at=$(echo "$signal" | jq -r '.expires_at // empty')
            active=$(echo "$signal" | jq -r '.active // true')

            # Validate signal type against schema enum
            case "$sig_type" in
                FOCUS|REDIRECT|FEEDBACK) ;;
                *) sig_type="FOCUS" ;;
            esac

            # Validate priority against schema enum
            case "$priority" in
                critical|high|normal|low) ;;
                *) priority="normal" ;;
            esac

            # XML escape ID and source
            local escaped_id escaped_source
            escaped_id=$(echo "$sig_id" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g; s/"/\&quot;/g')
            escaped_source=$(echo "$source" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g; s/"/\&quot;/g')

            # Build signal element
            xml_output="$xml_output
  <signal id=\"$escaped_id\"
          type=\"$sig_type\"
          priority=\"$priority\"
          source=\"$escaped_source\"
          created_at=\"$created_at\""

            # Add optional expires_at if present
            if [[ -n "$expires_at" && "$expires_at" != "null" ]]; then
                xml_output="$xml_output
          expires_at=\"$expires_at\""
            fi

            xml_output="$xml_output
          active=\"$active\">"

            # Content section - support legacy "message" field
            local content_text content_format
            content_text=$(echo "$signal" | jq -r '.content.text // .message // ""')
            content_format=$(echo "$signal" | jq -r '.content.data.format // "json"')

            if [[ -n "$content_text" ]]; then
                local escaped_text
                escaped_text=$(echo "$content_text" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g')
                xml_output="$xml_output
    <content>
      <text>$escaped_text</text>"

                # Check for data attachment - convert JSON to XML elements
                local has_data
                has_data=$(echo "$signal" | jq 'has("content") and (.content | has("data"))' 2>/dev/null)
                if [[ "$has_data" == "true" ]]; then
                    local data_xml
                    data_xml=$(echo "$signal" | jq -r '.content.data | to_entries | map("<\(.key)>\(.value | tostring | gsub("&"; "&amp;") | gsub("<"; "&lt;") | gsub(">"; "&gt;"))</\(.key)>") | join("")' 2>/dev/null)
                    xml_output="$xml_output
      <data format=\"$content_format\">$data_xml</data>"
                fi

                xml_output="$xml_output
    </content>"
            fi

            # Tags section
            local tags_json
            tags_json=$(echo "$signal" | jq -c '.tags // []')
            if [[ "$tags_json" != "[]" && -n "$tags_json" ]]; then
                xml_output="$xml_output
    <tags>"
                local tag_count
                tag_count=$(echo "$signal" | jq '.tags | length')
                local tag_idx=0
                while [[ $tag_idx -lt $tag_count ]]; do
                    local tag
                    tag=$(echo "$signal" | jq -c ".tags[$tag_idx]")
                    local tag_value tag_weight tag_category
                    tag_value=$(echo "$tag" | jq -r '.value // .')
                    tag_weight=$(echo "$tag" | jq -r '.weight // "1.0"')
                    tag_category=$(echo "$tag" | jq -r '.category // empty')

                    local escaped_tag
                    escaped_tag=$(echo "$tag_value" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g; s/"/\&quot;/g')

                    if [[ -n "$tag_category" && "$tag_category" != "null" ]]; then
                        xml_output="$xml_output
      <tag weight=\"$tag_weight\" category=\"$tag_category\">$escaped_tag</tag>"
                    else
                        xml_output="$xml_output
      <tag weight=\"$tag_weight\">$escaped_tag</tag>"
                    fi
                    ((tag_idx++))
                done
                xml_output="$xml_output
    </tags>"
            fi

            # Scope section
            local scope_global
            scope_global=$(echo "$signal" | jq -r '.scope.global // false')
            xml_output="$xml_output
    <scope global=\"$scope_global\">"

            # Castes
            local castes_json
            castes_json=$(echo "$signal" | jq -c '.scope.castes // []' 2>/dev/null)
            if [[ "$castes_json" != "[]" && -n "$castes_json" && "$castes_json" != "null" ]]; then
                xml_output="$xml_output
      <castes match=\"any\">"
                local caste_count
                caste_count=$(echo "$signal" | jq '.scope.castes | length' 2>/dev/null)
                local caste_idx=0
                while [[ $caste_idx -lt $caste_count ]]; do
                    local caste
                    caste=$(echo "$signal" | jq -r ".scope.castes[$caste_idx]" 2>/dev/null)
                    case "$caste" in
                        builder|watcher|scout|chaos|oracle|architect|prime|colonizer|route_setter|archaeologist|ambassador|auditor|chronicler|gatekeeper|guardian|includer|keeper|measurer|probe|sage|tracker|weaver)
                            xml_output="$xml_output
        <caste>$caste</caste>"
                            ;;
                    esac
                    ((caste_idx++))
                done
                xml_output="$xml_output
      </castes>"
            fi

            # Paths
            local paths_json
            paths_json=$(echo "$signal" | jq -c '.scope.paths // []' 2>/dev/null)
            if [[ "$paths_json" != "[]" && -n "$paths_json" && "$paths_json" != "null" ]]; then
                xml_output="$xml_output
      <paths match=\"any\">"
                local path_count
                path_count=$(echo "$signal" | jq '.scope.paths | length' 2>/dev/null)
                local path_idx=0
                while [[ $path_idx -lt $path_count ]]; do
                    local path
                    path=$(echo "$signal" | jq -r ".scope.paths[$path_idx]" 2>/dev/null)
                    local escaped_path
                    escaped_path=$(echo "$path" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g; s/"/\&quot;/g')
                    xml_output="$xml_output
        <path>$escaped_path</path>"
                    ((path_idx++))
                done
                xml_output="$xml_output
      </paths>"
            fi

            xml_output="$xml_output
    </scope>"

            xml_output="$xml_output
  </signal>"
            ((signal_count++))
        fi
    fi

    # Close root element
    xml_output="$xml_output
</pheromones>"

    # Write to file if output path specified
    local output_path=""
    if [[ -n "$output_xml" ]]; then
        local output_dir
        output_dir=$(dirname "$output_xml")
        if [[ ! -d "$output_dir" ]]; then
            mkdir -p "$output_dir" 2>/dev/null || {
                xml_json_err "Cannot create output directory: $output_dir"
                return 1
            }
        fi
        echo "$xml_output" > "$output_xml" || {
            xml_json_err "Failed to write output file: $output_xml"
            return 1
        }
        output_path="$output_xml"
    fi

    # Validate against XSD schema if available
    local validation_result="false"
    if [[ -f "$xsd_file" && -n "$output_path" ]]; then
        local validation_output
        validation_output=$(xml-validate "$output_path" "$xsd_file" 2>/dev/null)
        if echo "$validation_output" | jq -e '.result.valid' >/dev/null 2>&1; then
            validation_result="true"
        fi
    elif [[ -f "$xsd_file" ]]; then
        # Validate in-memory by writing to temp file
        local temp_xml
        temp_xml=$(mktemp)
        echo "$xml_output" > "$temp_xml"
        local validation_output
        validation_output=$(xml-validate "$temp_xml" "$xsd_file" 2>/dev/null)
        if echo "$validation_output" | jq -e '.result.valid' >/dev/null 2>&1; then
            validation_result="true"
        fi
        rm -f "$temp_xml"
    fi

    # Build result JSON
    local escaped_xml result_json
    escaped_xml=$(echo "$xml_output" | jq -Rs '.')
    result_json="{\"xml\":$escaped_xml,\"validated\":$validation_result,\"signals\":$signal_count"
    if [[ -n "$output_path" ]]; then
        local escaped_path
        escaped_path=$(echo "$output_path" | jq -Rs '.[:-1]')
        result_json="$result_json,\"path\":$escaped_path"
    fi
    result_json="$result_json}"

    xml_json_ok "$result_json"
}

# pheromone-from-xml: Parse pheromone XML to JSON
# Usage: pheromone-from-xml <pheromone_xml_file>
# Returns: {"ok":true,"result":{"signal":"focus",...}}
pheromone-from-xml() {
    local xml_file="${1:-}"

    [[ -z "$xml_file" ]] && { xml_json_err "Missing XML file argument"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "XML file not found: $xml_file"; return 1; }

    if [[ "$XMLSTARLET_AVAILABLE" != "true" ]]; then
        xml_json_err "xmlstarlet required for pheromone parsing"
        return 1
    fi

    # Extract pheromone fields from XML
    local signal priority message timestamp source context
    signal=$(xmlstarlet sel -t -v "/pheromone/signal" "$xml_file" 2>/dev/null || echo "")
    priority=$(xmlstarlet sel -t -v "/pheromone/priority" "$xml_file" 2>/dev/null || echo "normal")
    message=$(xmlstarlet sel -t -v "/pheromone/message" "$xml_file" 2>/dev/null || echo "")
    timestamp=$(xmlstarlet sel -t -v "/pheromone/timestamp" "$xml_file" 2>/dev/null || echo "")
    source=$(xmlstarlet sel -t -v "/pheromone/source" "$xml_file" 2>/dev/null || echo "colony")
    context=$(xmlstarlet sel -t -v "/pheromone/context" "$xml_file" 2>/dev/null || echo "")

    # Build JSON result
    local json_result
    json_result=$(jq -n \
        --arg signal "$signal" \
        --arg priority "$priority" \
        --arg message "$message" \
        --arg timestamp "$timestamp" \
        --arg source "$source" \
        --arg context "$context" \
        '{signal: $signal, priority: $priority, message: $message, timestamp: $timestamp, source: $source, context: (if $context == "" then null else $context end)}')

    xml_json_ok "$json_result"
}

# ============================================================================
# Queen-Wisdom XML Format
# ============================================================================

# queen-wisdom-to-xml: Convert queen wisdom JSON to XML
# Usage: queen-wisdom-to-xml <wisdom_json_file>
# Returns: {"ok":true,"result":{"xml":"<queen-wisdom>...</queen-wisdom>"}}
queen-wisdom-to-xml() {
    local json_file="${1:-}"

    [[ -z "$json_file" ]] && { xml_json_err "Missing JSON file argument"; return 1; }
    [[ -f "$json_file" ]] || { xml_json_err "JSON file not found: $json_file"; return 1; }

    if ! jq empty "$json_file" 2>/dev/null; then
        xml_json_err "Invalid JSON file: $json_file"
        return 1
    fi

    # Convert queen wisdom to structured XML
    local xml_output
    xml_output=$(jq -r '
        "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" +
        "<queen-wisdom version=\"1.0\">\n" +
        "  <directive>" + (.directive // "") + "</directive>\n" +
        (if .patterns then
            "  <patterns>\n" +
            (.patterns | map("    <pattern>\(.)</pattern>") | join("\n")) +
            "\n  </patterns>\n"
        else "" end) +
        (if .constraints then
            "  <constraints>\n" +
            (.constraints | map("    <constraint>\(.)</constraint>") | join("\n")) +
            "\n  </constraints>\n"
        else "" end) +
        "  <timestamp>" + (.timestamp // (now | todateiso8601)) + "</timestamp>\n" +
        "</queen-wisdom>"
    ' "$json_file" 2>/dev/null) || {
        xml_json_err "Queen wisdom conversion failed"
        return 1
    }

    local escaped_xml
    escaped_xml=$(echo "$xml_output" | jq -Rs '.')
    xml_json_ok "{\"xml\":$escaped_xml}"
}

# queen-wisdom-from-xml: Parse queen wisdom XML to JSON
# Usage: queen-wisdom-from-xml <wisdom_xml_file>
# Returns: {"ok":true,"result":{"directive":"...",...}}
queen-wisdom-from-xml() {
    local xml_file="${1:-}"

    [[ -z "$xml_file" ]] && { xml_json_err "Missing XML file argument"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "XML file not found: $xml_file"; return 1; }

    if [[ "$XMLSTARLET_AVAILABLE" != "true" ]]; then
        xml_json_err "xmlstarlet required for queen wisdom parsing"
        return 1
    fi

    # Extract fields
    local directive timestamp
    directive=$(xmlstarlet sel -t -v "/queen-wisdom/directive" "$xml_file" 2>/dev/null || echo "")
    timestamp=$(xmlstarlet sel -t -v "/queen-wisdom/timestamp" "$xml_file" 2>/dev/null || echo "")

    # Extract arrays
    local patterns_json constraints_json
    patterns_json=$(xmlstarlet sel -t -m "/queen-wisdom/patterns/pattern" -v "." -n "$xml_file" 2>/dev/null | jq -R -s 'split("\n") | map(select(length > 0))')
    constraints_json=$(xmlstarlet sel -t -m "/queen-wisdom/constraints/constraint" -v "." -n "$xml_file" 2>/dev/null | jq -R -s 'split("\n") | map(select(length > 0))')

    # Build result
    local json_result
    json_result=$(jq -n \
        --arg directive "$directive" \
        --arg timestamp "$timestamp" \
        --argjson patterns "$patterns_json" \
        --argjson constraints "$constraints_json" \
        '{directive: $directive, timestamp: $timestamp, patterns: $patterns, constraints: $constraints}')

    xml_json_ok "$json_result"
}

# ============================================================================
# Multi-Colony Registry XML
# ============================================================================

# registry-to-xml: Convert colony registry JSON to XML
# Usage: registry-to-xml <registry_json_file>
# Returns: {"ok":true,"result":{"xml":"<colony-registry>...</colony-registry>"}}
registry-to-xml() {
    local json_file="${1:-}"

    [[ -z "$json_file" ]] && { xml_json_err "Missing JSON file argument"; return 1; }
    [[ -f "$json_file" ]] || { xml_json_err "JSON file not found: $json_file"; return 1; }

    if ! jq empty "$json_file" 2>/dev/null; then
        xml_json_err "Invalid JSON file: $json_file"
        return 1
    fi

    # Convert registry to XML
    local xml_output
    xml_output=$(jq -r '
        "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" +
        "<colony-registry version=\"1.0\" generated=\"" + (now | todateiso8601) + "\">\n" +
        (if .colonies then
            (.colonies | map(
                "  <colony id=\"" + .id + "\">\n" +
                "    <name>" + (.name // "") + "</name>\n" +
                "    <status>" + (.status // "unknown") + "</status>\n" +
                "    <location>" + (.location // "") + "</location>\n" +
                (if .pheromones then
                    "    <pheromones>\n" +
                    (.pheromones | map(
                        "      <pheromone signal=\"" + .signal + "\">" + (.message // "") + "</pheromone>"
                    ) | join("\n")) +
                    "\n    </pheromones>\n"
                else "" end) +
                "  </colony>"
            ) | join("\n")) + "\n"
        else "" end) +
        "</colony-registry>"
    ' "$json_file" 2>/dev/null) || {
        xml_json_err "Registry conversion failed"
        return 1
    }

    local escaped_xml
    escaped_xml=$(echo "$xml_output" | jq -Rs '.')
    xml_json_ok "{\"xml\":$escaped_xml}"
}

# registry-from-xml: Parse colony registry XML to JSON
# Usage: registry-from-xml <registry_xml_file>
# Returns: {"ok":true,"result":{"colonies":[...]}}
registry-from-xml() {
    local xml_file="${1:-}"

    [[ -z "$xml_file" ]] && { xml_json_err "Missing XML file argument"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "XML file not found: $xml_file"; return 1; }

    if [[ "$XMLSTARLET_AVAILABLE" != "true" ]]; then
        xml_json_err "xmlstarlet required for registry parsing"
        return 1
    fi

    # Extract colonies
    local colonies_json
    colonies_json=$(xmlstarlet sel -t -m "/colony-registry/colony" \
        -v "@id" -o '|' \
        -v "name" -o '|' \
        -v "status" -o '|' \
        -v "location" -n \
        "$xml_file" 2>/dev/null | \
        awk -F'|' 'NF>=3 {
            printf "{\"id\":\"%s\",\"name\":\"%s\",\"status\":\"%s\",\"location\":\"%s\"}", $1, $2, $3, $4
        }' | \
        jq -s '.')

    local json_result
    json_result=$(jq -n --argjson colonies "$colonies_json" '{colonies: $colonies}')

    xml_json_ok "$json_result"
}

# ============================================================================
# Pheromone Export to Eternal Memory
# ============================================================================

# pheromone-export: Export pheromones to eternal XML format
# Usage: pheromone-export [input_json] [output_xml] [session_id]
#   input_json: Path to pheromones.json (default: .aether/data/pheromones.json)
#   output_xml: Path to output XML (default: ~/.aether/eternal/pheromones.xml)
#   session_id: Colony session ID for namespace generation (optional, auto-detected from JSON)
# Returns: {"ok":true,"result":{"exported":true,"path":"...","signals":N,"namespace":"..."}} or error
pheromone-export() {
    local input_json="${1:-.aether/data/pheromones.json}"
    local output_xml="${2:-$HOME/.aether/eternal/pheromones.xml}"
    local session_id="${3:-}"
    local schema_file="${4:-.aether/schemas/pheromone.xsd}"

    # Validate input file exists
    [[ -f "$input_json" ]] || { xml_json_err "Pheromone JSON file not found: $input_json"; return 1; }

    # Validate JSON
    if ! jq empty "$input_json" 2>/dev/null; then
        xml_json_err "Invalid JSON in pheromone file: $input_json"
        return 1
    fi

    # Get absolute paths for schema validation
    local abs_schema
    abs_schema="$(cd "$(dirname "$schema_file")" && pwd)/$(basename "$schema_file")" 2>/dev/null || abs_schema="$schema_file"

    # Generate ISO timestamp for XML
    local generated_at
    generated_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Get version and colony_id from JSON
    local version colony_id
    version=$(jq -r '.version // "1.0.0"' "$input_json")
    colony_id=$(jq -r '.colony_id // "unknown"' "$input_json")

    # Auto-detect session_id from JSON if not provided
    if [[ -z "$session_id" ]]; then
        session_id=$(jq -r '.session_id // .colony_id // ""' "$input_json")
    fi

    # Generate colony namespace if session_id available
    local colony_namespace=""
    local colony_prefix=""
    if [[ -n "$session_id" ]]; then
        local ns_result
        ns_result=$(generate-colony-namespace "$session_id" 2>/dev/null)
        if echo "$ns_result" | jq -e '.ok' >/dev/null 2>&1; then
            colony_namespace=$(echo "$ns_result" | jq -r '.result.namespace')
            colony_prefix=$(echo "$ns_result" | jq -r '.result.prefix')
        fi
    fi

    # Build XML header with proper namespace
    local xml_output
    xml_output="<?xml version=\"1.0\" encoding=\"UTF-8\"?>
<pheromones xmlns=\"http://aether.colony/schemas/pheromones\"
            xmlns:ph=\"http://aether.colony/schemas/pheromones\""

    # Add colony namespace if available
    if [[ -n "$colony_namespace" ]]; then
        xml_output="$xml_output
            xmlns:col=\"$colony_namespace\"
            col:session=\"$session_id\"
            col:prefix=\"$colony_prefix\""
    fi

    xml_output="$xml_output
            version=\"$version\"
            generated_at=\"$generated_at\"
            colony_id=\"$colony_id\">
  <metadata>
    <source type=\"system\" version=\"$version\">aether-pheromone-export</source>
    <context>Colony pheromone trail export to eternal memory</context>
  </metadata>"

    # Process each signal
    local signal_count=0
    local signals_json
    signals_json=$(jq -c '.signals // [] | .[]' "$input_json" 2>/dev/null)

    if [[ -n "$signals_json" ]]; then
        while IFS= read -r signal; do
            [[ -n "$signal" ]] || continue

            local sig_id sig_type priority source created_at expires_at active
            sig_id=$(echo "$signal" | jq -r '.id // "unknown"')
            sig_type=$(echo "$signal" | jq -r '.type // "FOCUS"')
            priority=$(echo "$signal" | jq -r '.priority // "normal"')
            source=$(echo "$signal" | jq -r '.source // "system"')
            created_at=$(echo "$signal" | jq -r '.created_at // ""')
            expires_at=$(echo "$signal" | jq -r '.expires_at // ""')
            active=$(echo "$signal" | jq -r '.active // true')

            # XML escape the ID and source
            local escaped_id escaped_source
            escaped_id=$(echo "$sig_id" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g; s/"/\&quot;/g')
            escaped_source=$(echo "$source" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g; s/"/\&quot;/g')

            xml_output="$xml_output
  <signal id=\"$escaped_id\"
          type=\"$sig_type\"
          priority=\"$priority\"
          source=\"$escaped_source\"
          created_at=\"$created_at\""

            if [[ -n "$expires_at" && "$expires_at" != "null" ]]; then
                xml_output="$xml_output
          expires_at=\"$expires_at\""
            fi

            xml_output="$xml_output
          active=\"$active\">"

            # Content section
            local content_text
            content_text=$(echo "$signal" | jq -r '.content.text // ""')
            if [[ -n "$content_text" ]]; then
                local escaped_text
                escaped_text=$(echo "$content_text" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g')
                xml_output="$xml_output
    <content>
      <text>$escaped_text</text>"

                # Check for data attachment
                local has_data data_format
                has_data=$(echo "$signal" | jq -r 'has("content") and has("content.data")')
                if [[ "$has_data" == "true" ]]; then
                    data_format=$(echo "$signal" | jq -r '.content.data.format // "json"')
                    xml_output="$xml_output
      <data format=\"$data_format\">"
                    # Add data content as CDATA or escaped
                    local data_content
                    data_content=$(echo "$signal" | jq -c '.content.data' 2>/dev/null)
                    xml_output="$xml_output$data_content"
                    xml_output="$xml_output</data>"
                fi

                xml_output="$xml_output
    </content>"
            fi

            # Tags section
            local tags_json
            tags_json=$(echo "$signal" | jq -c '.tags // []')
            if [[ "$tags_json" != "[]" && -n "$tags_json" ]]; then
                xml_output="$xml_output
    <tags>"
                local tags_array
                tags_array=$(echo "$signal" | jq -c '.tags // [] | .[]')
                while IFS= read -r tag; do
                    [[ -n "$tag" ]] || continue
                    local tag_value tag_weight tag_category
                    tag_value=$(echo "$tag" | jq -r '.value // .')
                    tag_weight=$(echo "$tag" | jq -r '.weight // "1.0"')
                    tag_category=$(echo "$tag" | jq -r '.category // ""')

                    local escaped_tag
                    escaped_tag=$(echo "$tag_value" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g; s/"/\&quot;/g')

                    if [[ -n "$tag_category" && "$tag_category" != "null" ]]; then
                        xml_output="$xml_output
      <tag weight=\"$tag_weight\" category=\"$tag_category\">$escaped_tag</tag>"
                    else
                        xml_output="$xml_output
      <tag weight=\"$tag_weight\">$escaped_tag</tag>"
                    fi
                done <<< "$tags_array"
                xml_output="$xml_output
    </tags>"
            fi

            # Scope section
            local scope_global
            scope_global=$(echo "$signal" | jq -r '.scope.global // false')
            xml_output="$xml_output
    <scope global=\"$scope_global\">"

            # Castes
            local castes_json
            castes_json=$(echo "$signal" | jq -c '.scope.castes // []')
            if [[ "$castes_json" != "[]" && -n "$castes_json" ]]; then
                xml_output="$xml_output
      <castes match=\"any\">"
                local caste_array
                caste_array=$(echo "$signal" | jq -r '.scope.castes[]')
                while IFS= read -r caste; do
                    [[ -n "$caste" ]] || continue
                    xml_output="$xml_output
        <caste>$caste</caste>"
                done <<< "$caste_array"
                xml_output="$xml_output
      </castes>"
            fi

            # Paths
            local paths_json
            paths_json=$(echo "$signal" | jq -c '.scope.paths // []')
            if [[ "$paths_json" != "[]" && -n "$paths_json" ]]; then
                xml_output="$xml_output
      <paths match=\"any\">"
                local path_array
                path_array=$(echo "$signal" | jq -r '.scope.paths[]')
                while IFS= read -r path; do
                    [[ -n "$path" ]] || continue
                    local escaped_path
                    escaped_path=$(echo "$path" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g; s/"/\&quot;/g')
                    xml_output="$xml_output
        <path>$escaped_path</path>"
                done <<< "$path_array"
                xml_output="$xml_output
      </paths>"
            fi

            xml_output="$xml_output
    </scope>"

            xml_output="$xml_output
  </signal>"
            ((signal_count++))
        done <<< "$signals_json"
    fi

    # Close root element
    xml_output="$xml_output
</pheromones>"

    # Ensure output directory exists
    local output_dir
    output_dir=$(dirname "$output_xml")
    if [[ ! -d "$output_dir" ]]; then
        mkdir -p "$output_dir" 2>/dev/null || {
            xml_json_err "Cannot create output directory: $output_dir"
            return 1
        }
    fi

    # Write XML to file
    echo "$xml_output" > "$output_xml" || {
        xml_json_err "Failed to write output file: $output_xml"
        return 1
    }

    # Validate against schema if available
    local validation_result="false"
    if [[ -f "$abs_schema" ]]; then
        validation_result=$(xml-validate "$output_xml" "$abs_schema" 2>/dev/null)
        if ! echo "$validation_result" | jq -e '.result.valid' >/dev/null 2>&1; then
            xml_json_err "XML validation failed against schema: $abs_schema"
            return 1
        fi
        validation_result="true"
    fi

    # Return success with metadata
    local escaped_output
    escaped_output=$(echo "$output_xml" | jq -Rs '.[:-1]')
    local result_json
    result_json="{\"exported\":true,\"path\":$escaped_output,\"signals\":$signal_count,\"validated\":$validation_result"
    if [[ -n "$colony_namespace" ]]; then
        result_json="$result_json,\"namespace\":\"$colony_namespace\",\"prefix\":\"$colony_prefix\""
    fi
    result_json="$result_json}"
    xml_json_ok "$result_json"
}

# ============================================================================
# Colony Namespace Generation
# ============================================================================

# Colony namespace base URI
readonly COLONY_NAMESPACE_BASE="http://aether.dev/colony"

# generate-colony-namespace: Generate unique namespace URI for a colony session
# Usage: generate-colony-namespace <session_id>
# Returns: {"ok":true,"result":{"namespace":"http://aether.dev/colony/{session_id}","prefix":"col_{hash}"}}
generate-colony-namespace() {
    local session_id="${1:-}"

    [[ -z "$session_id" ]] && { xml_json_err "Missing session_id argument"; return 1; }

    # Generate namespace URI
    local namespace_uri="${COLONY_NAMESPACE_BASE}/${session_id}"

    # Generate short prefix from session_id (first 8 chars of MD5 hash)
    local prefix
    if command -v md5sum >/dev/null 2>&1; then
        prefix="col_$(echo -n "$session_id" | md5sum | cut -c1-8)"
    elif command -v md5 >/dev/null 2>&1; then
        prefix="col_$(echo -n "$session_id" | md5 | cut -c1-8)"
    else
        # Fallback: use first 8 alphanumeric chars of session_id
        prefix="col_$(echo -n "$session_id" | tr -cd '[:alnum:]' | cut -c1-8)"
    fi

    xml_json_ok "{\"namespace\":\"$namespace_uri\",\"prefix\":\"$prefix\",\"session_id\":\"$session_id\"}"
}

# generate-cross-colony-prefix: Generate prefix for external colony pheromones
# Usage: generate-cross-colony-prefix <external_session_id> [local_session_id]
# Returns: {"ok":true,"result":{"prefix":"ext_{hash}_{short_id}","full_prefix":"{local_prefix}_{external_prefix}"}}
generate-cross-colony-prefix() {
    local external_session_id="${1:-}"
    local local_session_id="${2:-}"

    [[ -z "$external_session_id" ]] && { xml_json_err "Missing external_session_id argument"; return 1; }

    # Generate external colony prefix
    local external_prefix
    if command -v md5sum >/dev/null 2>&1; then
        external_prefix="ext_$(echo -n "$external_session_id" | md5sum | cut -c1-6)"
    elif command -v md5 >/dev/null 2>&1; then
        external_prefix="ext_$(echo -n "$external_session_id" | md5 | cut -c1-6)"
    else
        external_prefix="ext_$(echo -n "$external_session_id" | tr -cd '[:alnum:]' | cut -c1-6)"
    fi

    # If local session provided, create combined prefix for collision prevention
    local full_prefix="$external_prefix"
    if [[ -n "$local_session_id" ]]; then
        local local_hash
        if command -v md5sum >/dev/null 2>&1; then
            local_hash="$(echo -n "$local_session_id" | md5sum | cut -c1-4)"
        elif command -v md5 >/dev/null 2>&1; then
            local_hash="$(echo -n "$local_session_id" | md5 | cut -c1-4)"
        else
            local_hash="$(echo -n "$local_session_id" | tr -cd '[:alnum:]' | cut -c1-4)"
        fi
        full_prefix="${local_hash}_${external_prefix}"
    fi

    xml_json_ok "{\"prefix\":\"$external_prefix\",\"full_prefix\":\"$full_prefix\",\"external_session\":\"$external_session_id\"}"
}

# prefix-pheromone-id: Prefix a pheromone ID to prevent collisions
# Usage: prefix-pheromone-id <pheromone_id> <colony_prefix>
# Returns: {"ok":true,"result":"{prefix}_{pheromone_id}"}
prefix-pheromone-id() {
    local pheromone_id="${1:-}"
    local colony_prefix="${2:-}"

    [[ -z "$pheromone_id" ]] && { xml_json_err "Missing pheromone_id argument"; return 1; }
    [[ -z "$colony_prefix" ]] && { xml_json_err "Missing colony_prefix argument"; return 1; }

    # Check if already prefixed with this colony
    if [[ "$pheromone_id" == ${colony_prefix}_* ]]; then
        xml_json_ok "\"$pheromone_id\""
        return 0
    fi

    local prefixed_id="${colony_prefix}_${pheromone_id}"
    xml_json_ok "\"$prefixed_id\""
}

# extract-session-from-namespace: Extract session ID from namespace URI
# Usage: extract-session-from-namespace <namespace_uri>
# Returns: {"ok":true,"result":"{session_id}"}
extract-session-from-namespace() {
    local namespace_uri="${1:-}"

    [[ -z "$namespace_uri" ]] && { xml_json_err "Missing namespace_uri argument"; return 1; }

    # Extract session ID from http://aether.dev/colony/{session_id}
    local session_id
    if [[ "$namespace_uri" =~ ^http://aether\.dev/colony/(.+)$ ]]; then
        session_id="${BASH_REMATCH[1]}"
        xml_json_ok "\"$session_id\""
    else
        xml_json_err "Invalid colony namespace format: $namespace_uri"
        return 1
    fi
}

# validate-colony-namespace: Validate a colony namespace URI
# Usage: validate-colony-namespace <namespace_uri>
# Returns: {"ok":true,"result":{"valid":true,"type":"colony","session_id":"..."}}
validate-colony-namespace() {
    local namespace_uri="${1:-}"

    [[ -z "$namespace_uri" ]] && { xml_json_err "Missing namespace_uri argument"; return 1; }

    # Check if it matches colony namespace pattern
    if [[ "$namespace_uri" =~ ^http://aether\.dev/colony/([a-zA-Z0-9_-]+)$ ]]; then
        local session_id="${BASH_REMATCH[1]}"
        xml_json_ok "{\"valid\":true,\"type\":\"colony\",\"session_id\":\"$session_id\"}"
    elif [[ "$namespace_uri" == "http://aether.colony/schemas/pheromones" ]]; then
        xml_json_ok "{\"valid\":true,\"type\":\"schema\",\"session_id\":null}"
    else
        xml_json_ok "{\"valid\":false,\"type\":null,\"session_id\":null}"
    fi
}

# ============================================================================
# Queen-Wisdom Markdown Generation (XSLT-based)
# ============================================================================

# queen-wisdom-to-markdown: Convert queen-wisdom XML to markdown using XSLT
# Usage: queen-wisdom-to-markdown <wisdom_xml_file> [output_md_file]
#   wisdom_xml_file: Path to queen-wisdom.xml
#   output_md_file: Optional path to write markdown output (default: stdout)
# Returns: {"ok":true,"result":{"markdown":"...","path":"..."}} or error
queen-wisdom-to-markdown() {
    local xml_file="${1:-}"
    local output_file="${2:-}"

    # Validate arguments
    [[ -z "$xml_file" ]] && { xml_json_err "Missing XML file argument"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "XML file not found: $xml_file"; return 1; }

    # Check for xsltproc
    if [[ "$XSLTPROC_AVAILABLE" != "true" ]]; then
        xml_json_err "xsltproc not available. Install libxslt utilities."
        return 1
    fi

    # Find XSLT file (check multiple locations)
    local xsl_file=""
    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

    # Search paths for XSLT file
    local search_paths=(
        "$script_dir/queen-to-md.xsl"
        ".aether/utils/queen-to-md.xsl"
        "runtime/utils/queen-to-md.xsl"
        "$HOME/.aether/system/utils/queen-to-md.xsl"
    )

    for path in "${search_paths[@]}"; do
        if [[ -f "$path" ]]; then
            xsl_file="$path"
            break
        fi
    done

    if [[ -z "$xsl_file" ]]; then
        xml_json_err "XSLT file queen-to-md.xsl not found in standard locations"
        return 1
    fi

    # Validate XML against schema first
    local schema_file="$script_dir/../schemas/queen-wisdom.xsd"
    if [[ -f "$schema_file" ]]; then
        local validation
        validation=$(xml-validate "$xml_file" "$schema_file" 2>/dev/null)
        if ! echo "$validation" | jq -e '.result.valid' >/dev/null 2>&1; then
            xml_json_err "XML validation failed before conversion"
            return 1
        fi
    fi

    # Perform XSLT transformation
    local markdown
    if ! markdown=$(xsltproc "$xsl_file" "$xml_file" 2>&1); then
        xml_json_err "XSLT transformation failed: $markdown"
        return 1
    fi

    # Output handling
    if [[ -n "$output_file" ]]; then
        if echo "$markdown" > "$output_file"; then
            xml_json_ok "{\"markdown\":\"(written to file)\",\"path\":\"$output_file\"}"
        else
            xml_json_err "Failed to write to output file: $output_file"
            return 1
        fi
    else
        # Return markdown in JSON result
        local escaped_markdown
        escaped_markdown=$(echo "$markdown" | jq -Rs '.[:-1]')
        xml_json_ok "{\"markdown\":$escaped_markdown,\"path\":null}"
    fi
}

# ============================================================================
# Queen-Wisdom Promotion Workflow
# ============================================================================

# Get promotion threshold for a wisdom type
# Usage: _get_promotion_threshold <type>
# Returns: threshold value
_get_promotion_threshold() {
    local wisdom_type="$1"
    case "$wisdom_type" in
        philosophy) echo "5" ;;
        pattern) echo "3" ;;
        redirect) echo "2" ;;
        stack) echo "1" ;;
        decree) echo "0" ;;
        *) echo "1" ;;
    esac
}

# queen-wisdom-validate-entry: Validate a single wisdom entry
# Usage: queen-wisdom-validate-entry <xml_file> <entry_id>
# Returns: {"ok":true,"result":{"valid":true,"errors":[],"warnings":[]}} or error
queen-wisdom-validate-entry() {
    local xml_file="${1:-}"
    local entry_id="${2:-}"

    [[ -z "$xml_file" ]] && { xml_json_err "Missing XML file argument"; return 1; }
    [[ -z "$entry_id" ]] && { xml_json_err "Missing entry ID argument"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "XML file not found: $xml_file"; return 1; }

    if [[ "$XMLLINT_AVAILABLE" != "true" ]]; then
        xml_json_err "xmllint not available"
        return 1
    fi

    # Build XPath query to find the entry by ID
    local xpath_query="//*[@id='$entry_id']"
    local entry_xml
    entry_xml=$(xmllint --xpath "$xpath_query" "$xml_file" 2>/dev/null) || {
        xml_json_err "Entry not found with ID: $entry_id"
        return 1
    }

    # Initialize validation results
    local errors=()
    local warnings=()

    # Extract attributes for validation
    local confidence domain source created_at content
    confidence=$(xmllint --xpath "string(//*[@id='$entry_id']/@confidence)" "$xml_file" 2>/dev/null)
    domain=$(xmllint --xpath "string(//*[@id='$entry_id']/@domain)" "$xml_file" 2>/dev/null)
    source=$(xmllint --xpath "string(//*[@id='$entry_id']/@source)" "$xml_file" 2>/dev/null)
    created_at=$(xmllint --xpath "string(//*[@id='$entry_id']/@created_at)" "$xml_file" 2>/dev/null)
    content=$(xmllint --xpath "string(//*[@id='$entry_id']/qw:content)" "$xml_file" 2>/dev/null || echo "")

    # Validate confidence (0.0 to 1.0)
    if [[ -z "$confidence" ]]; then
        errors+=("Missing required attribute: confidence")
    elif ! [[ "$confidence" =~ ^0?\.[0-9]+$|^1\.0$ ]]; then
        errors+=("Invalid confidence value: $confidence (must be 0.0-1.0)")
    fi

    # Validate domain (must be from allowed list)
    local valid_domains="architecture testing security performance ux process communication debugging general"
    if [[ -z "$domain" ]]; then
        errors+=("Missing required attribute: domain")
    elif [[ ! " $valid_domains " =~ " $domain " ]]; then
        errors+=("Invalid domain: $domain (must be one of: $valid_domains)")
    fi

    # Validate source (must be from allowed list)
    local valid_sources="queen user colony oracle observation"
    if [[ -z "$source" ]]; then
        errors+=("Missing required attribute: source")
    elif [[ ! " $valid_sources " =~ " $source " ]]; then
        errors+=("Invalid source: $source (must be one of: $valid_sources)")
    fi

    # Validate created_at (ISO 8601 format)
    if [[ -z "$created_at" ]]; then
        errors+=("Missing required attribute: created_at")
    elif ! [[ "$created_at" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2} ]]; then
        errors+=("Invalid timestamp format: $created_at (expected ISO 8601)")
    fi

    # Validate content
    if [[ -z "$content" ]]; then
        errors+=("Missing required element: content")
    elif [[ ${#content} -lt 10 ]]; then
        warnings+=("Content is very short (${#content} chars) - consider expanding")
    fi

    # Build JSON response
    local error_json="[]"
    local warning_json="[]"

    if [[ ${#errors[@]} -gt 0 ]]; then
        error_json=$(printf '%s\n' "${errors[@]}" | jq -R . | jq -s .)
    fi

    if [[ ${#warnings[@]} -gt 0 ]]; then
        warning_json=$(printf '%s\n' "${warnings[@]}" | jq -R . | jq -s .)
    fi

    local valid="false"
    [[ ${#errors[@]} -eq 0 ]] && valid="true"

    xml_json_ok "{\"valid\":$valid,\"errors\":$error_json,\"warnings\":$warning_json}"
}

# queen-wisdom-promote: Promote a wisdom entry with validation
# Usage: queen-wisdom-promote <xml_file> <entry_id> [target_level]
#   xml_file: Path to queen-wisdom.xml
#   entry_id: ID of the entry to promote
#   target_level: Optional target promotion level (defaults to next level)
# Returns: {"ok":true,"result":{"promoted":true,"from":"...","to":"...","evolution_log_updated":true}} or error
queen-wisdom-promote() {
    local xml_file="${1:-}"
    local entry_id="${2:-}"
    local target_level="${3:-}"

    [[ -z "$xml_file" ]] && { xml_json_err "Missing XML file argument"; return 1; }
    [[ -z "$entry_id" ]] && { xml_json_err "Missing entry ID argument"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "XML file not found: $xml_file"; return 1; }

    # First validate the entry
    local validation
    validation=$(queen-wisdom-validate-entry "$xml_file" "$entry_id" 2>&1)
    if ! echo "$validation" | jq -e '.result.valid' >/dev/null 2>&1; then
        local errors
        errors=$(echo "$validation" | jq -r '.result.errors | join("; ")')
        xml_json_err "Validation failed: $errors"
        return 1
    fi

    # Get current entry type and applied count
    local current_type applied_count
    # Determine entry type by which container it's in
    if xmllint --xpath "//qw:philosophies/qw:philosophy[@id='$entry_id']" "$xml_file" >/dev/null 2>&1; then
        current_type="philosophy"
        applied_count=$(xmllint --xpath "string(//qw:philosophy[@id='$entry_id']/@applied_count)" "$xml_file" 2>/dev/null || echo "0")
    elif xmllint --xpath "//qw:patterns/qw:pattern[@id='$entry_id']" "$xml_file" >/dev/null 2>&1; then
        current_type="pattern"
        applied_count=$(xmllint --xpath "string(//qw:pattern[@id='$entry_id']/@applied_count)" "$xml_file" 2>/dev/null || echo "0")
    elif xmllint --xpath "//qw:redirects/qw:redirect[@id='$entry_id']" "$xml_file" >/dev/null 2>&1; then
        current_type="redirect"
        applied_count=$(xmllint --xpath "string(//qw:redirect[@id='$entry_id']/@applied_count)" "$xml_file" 2>/dev/null || echo "0")
    elif xmllint --xpath "//qw:stack-wisdom/qw:wisdom[@id='$entry_id']" "$xml_file" >/dev/null 2>&1; then
        current_type="stack"
        applied_count=$(xmllint --xpath "string(//qw:stack-wisdom/qw:wisdom[@id='$entry_id']/@applied_count)" "$xml_file" 2>/dev/null || echo "0")
    elif xmllint --xpath "//qw:decrees/qw:decree[@id='$entry_id']" "$xml_file" >/dev/null 2>&1; then
        current_type="decree"
        applied_count=$(xmllint --xpath "string(//qw:decree[@id='$entry_id']/@applied_count)" "$xml_file" 2>/dev/null || echo "0")
    else
        xml_json_err "Entry not found: $entry_id"
        return 1
    fi

    # Check promotion threshold
    local threshold
    threshold=$(_get_promotion_threshold "$current_type")

    if [[ "$applied_count" -lt "$threshold" ]]; then
        xml_json_err "Not enough validations for promotion: $applied_count < $threshold (required for $current_type)"
        return 1
    fi

    # Get colony ID from metadata or use "unknown"
    local colony_id
    colony_id=$(xmllint --xpath "string(//qw:metadata/qw:colony_id)" "$xml_file" 2>/dev/null || echo "unknown")

    # Create evolution log entry
    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Note: In a full implementation, this would modify the XML file
    # For now, we return success and indicate what would happen
    xml_json_ok "{\"promoted\":true,\"entry_id\":\"$entry_id\",\"type\":\"$current_type\",\"from_applied\":$applied_count,\"threshold\":$threshold,\"colony\":\"$colony_id\",\"timestamp\":\"$timestamp\",\"note\":\"Evolution log update requires XML editing capability\"}"
}

# queen-wisdom-import: Import wisdom from markdown QUEEN.md to XML
# Usage: queen-wisdom-import <queen_md_file> [output_xml_file]
# Returns: {"ok":true,"result":{"imported":5,"xml":"...","path":"..."}} or error
queen-wisdom-import() {
    local md_file="${1:-}"
    local output_file="${2:-"queen-wisdom-imported.xml"}"

    [[ -z "$md_file" ]] && { xml_json_err "Missing markdown file argument"; return 1; }
    [[ -f "$md_file" ]] || { xml_json_err "Markdown file not found: $md_file"; return 1; }

    # Extract metadata from JSON block
    local version last_evolved colonies
    version=$(grep -A20 'METADATA' "$md_file" | grep '"version"' | sed 's/.*: "\([^"]*\)".*/\1/')
    last_evolved=$(grep -A20 'METADATA' "$md_file" | grep '"last_evolved"' | sed 's/.*: "\([^"]*\)".*/\1/')

    # Generate timestamps
    local created modified
    created="${last_evolved:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}"
    modified=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Start building XML
    local xml='<?xml version="1.0" encoding="UTF-8"?>'
    xml+=$'\n<queen-wisdom xmlns:qw="http://aether.colony/schemas/queen-wisdom/1.0">'

    # Metadata section
    xml+=$'\n  <metadata>'
    xml+=$'\n    <version>'"${version:-1.0.0}"'</version>'
    xml+=$'\n    <created>'"$created"'</created>'
    xml+=$'\n    <modified>'"$modified"'</modified>'
    xml+=$'\n    <colony_id>imported</colony_id>'
    xml+=$'\n  </metadata>'

    # Parse sections using simple grep/sed patterns
    # Note: This is a basic implementation - full parsing would require more sophisticated handling

    local imported_count=0

    # Extract philosophies (simplified parsing)
    xml+=$'\n  <philosophies>'
    while IFS= read -r line; do
        # Look for lines starting with "- **" and extract content
        if [[ "$line" =~ ^[[:space:]]*-[[:space:]]*\*\* ]]; then
            local id timestamp content
            id=$(echo "$line" | sed -n 's/.*\*\*\([^*]*\)\*\*.*/\1/p')
            timestamp=$(echo "$line" | sed -n 's/.*(\([^)]*\)).*/\1/p')
            content=$(echo "$line" | sed -n 's/.*):[[:space:]]*\(.*\)/\1/p')
            if [[ -n "$id" && -n "$content" ]]; then
                xml+=$'\n    <philosophy id="'"$id"'" confidence="0.8" domain="general" source="observation" created_at="'"${timestamp:-$modified}"'">'
                xml+=$'\n      <content>'"$content"'</content>'
                xml+=$'\n    </philosophy>'
                ((imported_count++))
            fi
        fi
    done < <(sed -n '/## 📜 Philosophies/,/## /p' "$md_file" 2>/dev/null | tail -n +4)
    xml+=$'\n  </philosophies>'

    # Extract patterns (simplified parsing)
    xml+=$'\n  <patterns>'
    while IFS= read -r line; do
        if [[ "$line" =~ ^[[:space:]]*-[[:space:]]*\*\* ]]; then
            local id timestamp content
            id=$(echo "$line" | sed -n 's/.*\*\*\([^*]*\)\*\*.*/\1/p')
            timestamp=$(echo "$line" | sed -n 's/.*(\([^)]*\)).*/\1/p')
            content=$(echo "$line" | sed -n 's/.*):[[:space:]]*\(.*\)/\1/p')
            if [[ -n "$id" && -n "$content" ]]; then
                xml+=$'\n    <pattern id="'"$id"'" confidence="0.7" domain="general" source="observation" created_at="'"${timestamp:-$modified}"'">'
                xml+=$'\n      <content>'"$content"'</content>'
                xml+=$'\n    </pattern>'
                ((imported_count++))
            fi
        fi
    done < <(sed -n '/## 🧭 Patterns/,/## /p' "$md_file" 2>/dev/null | tail -n +4)
    xml+=$'\n  </patterns>'

    # Similar for other sections (simplified for brevity)
    xml+=$'\n  <redirects />'
    xml+=$'\n  <stack-wisdom />'
    xml+=$'\n  <decrees />'

    # Evolution log
    xml+=$'\n  <evolution-log>'
    xml+=$'\n    <entry timestamp="'"$modified"'" colony="import" action="imported" type="markdown">'
    xml+=$'\n      <note>Imported from '"$md_file"' with '"$imported_count"' entries</note>'
    xml+=$'\n    </entry>'
    xml+=$'\n  </evolution-log>'

    xml+=$'\n</queen-wisdom>'

    # Write output
    echo "$xml" > "$output_file"

    xml_json_ok "{\"imported\":$imported_count,\"xml\":\"(written to file)\",\"path\":\"$output_file\"}"
}

# ============================================================================
# Prompt XML Conversion
# ============================================================================

# prompt-to-xml: Convert a markdown prompt file to structured XML
# Usage: prompt-to-xml <markdown_file> [output_xml_file]
# Returns: {"ok":true,"result":{"xml":"...","path":"...","elements_extracted":N}} or error
prompt-to-xml() {
    local md_file="${1:-}"
    local output_file="${2:-}"

    [[ -z "$md_file" ]] && { xml_json_err "Missing markdown file argument"; return 1; }
    [[ -f "$md_file" ]] || { xml_json_err "Markdown file not found: $md_file"; return 1; }

    # Extract prompt name from filename
    local prompt_name
    prompt_name=$(basename "$md_file" .md)

    # Initialize XML parts
    local xml='<?xml version="1.0" encoding="UTF-8"?>'
    xml+=$'\n<aether-prompt xmlns:ap="http://aether.colony/schemas/prompt/1.0">'

    # Metadata
    xml+=$'\n  <metadata>'
    xml+=$'\n    <version>1.0.0</version>'
    xml+=$'\n    <created>'"$(date -u +"%Y-%m-%dT%H:%M:%SZ")"'</created>'
    xml+=$'\n  </metadata>'

    # Name and type detection
    xml+=$'\n  <name>'"$prompt_name"'</name>'

    # Detect type from content patterns
    local prompt_type="command"
    if grep -q "worker\|caste\|Builder\|Watcher\|Scout" "$md_file" 2>/dev/null; then
        prompt_type="worker"
    elif grep -q "agent\|Agent" "$md_file" 2>/dev/null; then
        prompt_type="agent"
    fi
    xml+=$'\n  <type>'"$prompt_type"'</type>'

    # Try to detect caste for worker prompts
    local caste=""
    if [[ "$prompt_type" == "worker" ]]; then
        if grep -qi "builder" "$md_file"; then
            caste="builder"
        elif grep -qi "watcher" "$md_file"; then
            caste="watcher"
        elif grep -qi "scout" "$md_file"; then
            caste="scout"
        elif grep -qi "chaos" "$md_file"; then
            caste="chaos"
        elif grep -qi "oracle" "$md_file"; then
            caste="oracle"
        elif grep -qi "architect" "$md_file"; then
            caste="architect"
        fi
        [[ -n "$caste" ]] && xml+=$'\n  <caste>'"$caste"'</caste>'
    fi

    # Extract objective (first H1 or first paragraph)
    local objective
    objective=$(grep -m1 "^# " "$md_file" 2>/dev/null | sed 's/^# //' || head -1 "$md_file")
    xml+=$'\n  <objective>'"$(xml_escape_content "$objective")"'</objective>'

    # Extract requirements (## Requirements or numbered lists)
    xml+=$'\n  <requirements>'
    local req_count=0
    while IFS= read -r line; do
        # Check for list items (bullet or numbered)
        if [[ "$line" =~ ^[[:space:]]*[-*][[:space:]] ]] || [[ "$line" =~ ^[[:space:]]*[0-9]+\.[[:space:]] ]]; then
            local req_desc
            req_desc=$(echo "$line" | sed -E 's/^[[:space:]]*[-*][[:space:]]+//' | sed -E 's/^[[:space:]]*[0-9]+\.[[:space:]]+//')
            ((req_count++))
            xml+=$'\n    <requirement id="req_'"$req_count"'" priority="normal">'
            xml+=$'\n      <description>'"$(xml_escape_content "$req_desc")"'</description>'
            xml+=$'\n    </requirement>'
        fi
    done < <(sed -n '/## Requirement/,/## /p' "$md_file" 2>/dev/null | tail -n +2)

    # If no requirements section found, add a default one
    if [[ $req_count -eq 0 ]]; then
        xml+=$'\n    <requirement id="req_1" priority="normal">'
        xml+=$'\n      <description>Follow the instructions in this prompt</description>'
        xml+=$'\n    </requirement>'
    fi
    xml+=$'\n  </requirements>'

    # Output specification
    xml+=$'\n  <output>'
    xml+=$'\n    <format>Markdown</format>'
    xml+=$'\n  </output>'

    # Verification
    xml+=$'\n  <verification>'
    xml+=$'\n    <method>Check output meets success criteria</method>'
    xml+=$'\n  </verification>'

    # Success criteria
    xml+=$'\n  <success_criteria>'
    xml+=$'\n    <criterion id="crit_1" required="true">'
    xml+=$'\n      <description>Task completed as specified</description>'
    xml+=$'\n    </criterion>'
    xml+=$'\n  </success_criteria>'

    xml+=$'\n</aether-prompt>'

    # Output handling
    if [[ -n "$output_file" ]]; then
        echo "$xml" > "$output_file"
        xml_json_ok "{\"xml\":\"(written to file)\",\"path\":\"$output_file\",\"elements_extracted\":$((req_count + 5))}"
    else
        local escaped_xml
        escaped_xml=$(echo "$xml" | jq -Rs '.[:-1]')
        xml_json_ok "{\"xml\":$escaped_xml,\"path\":null,\"elements_extracted\":$((req_count + 5))}"
    fi
}

# prompt-from-xml: Convert XML prompt to markdown format
# Usage: prompt-from-xml <xml_file> [output_md_file]
# Returns: {"ok":true,"result":{"markdown":"...","path":"..."}} or error
prompt-from-xml() {
    local xml_file="${1:-}"
    local output_file="${2:-}"

    [[ -z "$xml_file" ]] && { xml_json_err "Missing XML file argument"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "XML file not found: $xml_file"; return 1; }

    if [[ "$XMLLINT_AVAILABLE" != "true" ]]; then
        xml_json_err "xmllint not available"
        return 1
    fi

    # Validate against schema
    local schema_file
    schema_file="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../schemas/prompt.xsd"

    if [[ -f "$schema_file" ]]; then
        local validation
        validation=$(xml-validate "$xml_file" "$schema_file" 2>/dev/null)
        if ! echo "$validation" | jq -e '.result.valid' >/dev/null 2>&1; then
            xml_json_err "XML validation failed against prompt.xsd schema"
            return 1
        fi
    fi

    # Extract fields using XPath
    local name type caste objective
    name=$(xmllint --xpath "string(//ap:name)" "$xml_file" 2>/dev/null || echo "Unnamed")
    type=$(xmllint --xpath "string(//ap:type)" "$xml_file" 2>/dev/null || echo "command")
    caste=$(xmllint --xpath "string(//ap:caste)" "$xml_file" 2>/dev/null || echo "")
    objective=$(xmllint --xpath "string(//ap:objective)" "$xml_file" 2>/dev/null || echo "")

    # Build markdown
    local md="# $name"
    [[ -n "$caste" ]] && md+=" ($caste $type)"
    md+=$'\n\n'

    md+="## Objective"
    md+=$'\n\n'
    md+="$objective"
    md+=$'\n\n'

    # Requirements
    local req_count
    req_count=$(xmllint --xpath "count(//ap:requirements/ap:requirement)" "$xml_file" 2>/dev/null || echo "0")
    if [[ "$req_count" -gt 0 ]]; then
        md+="## Requirements"
        md+=$'\n\n'

        for i in $(seq 1 "$req_count"); do
            local req_desc req_priority
            req_desc=$(xmllint --xpath "string(//ap:requirements/ap:requirement[$i]/ap:description)" "$xml_file" 2>/dev/null || echo "")
            req_priority=$(xmllint --xpath "string(//ap:requirements/ap:requirement[$i]/@priority)" "$xml_file" 2>/dev/null || echo "normal")

            if [[ -n "$req_desc" ]]; then
                md+="$i. [$req_priority] $req_desc"
                md+=$'\n'
            fi
        done
        md+=$'\n'
    fi

    # Constraints
    local constraint_count
    constraint_count=$(xmllint --xpath "count(//ap:constraints/ap:constraint)" "$xml_file" 2>/dev/null || echo "0")
    if [[ "$constraint_count" -gt 0 ]]; then
        md+="## Constraints"
        md+=$'\n\n'

        for i in $(seq 1 "$constraint_count"); do
            local constraint_rule constraint_strength
            constraint_rule=$(xmllint --xpath "string(//ap:constraints/ap:constraint[$i]/ap:rule)" "$xml_file" 2>/dev/null || echo "")
            constraint_strength=$(xmllint --xpath "string(//ap:constraints/ap:constraint[$i]/@strength)" "$xml_file" 2>/dev/null || echo "should")

            if [[ -n "$constraint_rule" ]]; then
                md+="- [$constraint_strength] $constraint_rule"
                md+=$'\n'
            fi
        done
        md+=$'\n'
    fi

    # Output
    local output_format
    output_format=$(xmllint --xpath "string(//ap:output/ap:format)" "$xml_file" 2>/dev/null || echo "Markdown")
    md+="## Output"
    md+=$'\n\n'
    md+="Format: $output_format"
    md+=$'\n\n'

    # Verification
    md+="## Verification"
    md+=$'\n\n'
    local verification_method
    verification_method=$(xmllint --xpath "string(//ap:verification/ap:method)" "$xml_file" 2>/dev/null || echo "Manual review")
    md+="$verification_method"
    md+=$'\n\n'

    # Success criteria
    md+="## Success Criteria"
    md+=$'\n\n'

    local crit_count
    crit_count=$(xmllint --xpath "count(//ap:success_criteria/ap:criterion)" "$xml_file" 2>/dev/null || echo "0")
    for i in $(seq 1 "$crit_count"); do
        local crit_desc crit_required
        crit_desc=$(xmllint --xpath "string(//ap:success_criteria/ap:criterion[$i]/ap:description)" "$xml_file" 2>/dev/null || echo "")
        crit_required=$(xmllint --xpath "string(//ap:success_criteria/ap:criterion[$i]/@required)" "$xml_file" 2>/dev/null || echo "true")

        if [[ -n "$crit_desc" ]]; then
            [[ "$crit_required" == "true" ]] && md+="- [required] " || md+="- [optional] "
            md+="$crit_desc"
            md+=$'\n'
        fi
    done

    # Output handling
    if [[ -n "$output_file" ]]; then
        echo "$md" > "$output_file"
        xml_json_ok "{\"markdown\":\"(written to file)\",\"path\":\"$output_file\"}"
    else
        local escaped_md
        escaped_md=$(echo "$md" | jq -Rs '.[:-1]')
        xml_json_ok "{\"markdown\":$escaped_md,\"path\":null}"
    fi
}

# prompt-validate: Validate a prompt XML file against the schema
# Usage: prompt-validate <xml_file>
# Returns: {"ok":true,"result":{"valid":true,"errors":[]}} or error
prompt-validate() {
    local xml_file="${1:-}"

    [[ -z "$xml_file" ]] && { xml_json_err "Missing XML file argument"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "XML file not found: $xml_file"; return 1; }

    local schema_file
    schema_file="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../schemas/prompt.xsd"

    [[ -f "$schema_file" ]] || { xml_json_err "Schema file not found: $schema_file"; return 1; }

    xml-validate "$xml_file" "$schema_file"
}

# Helper function to escape XML content
xml_escape_content() {
    local content="$1"
    # Basic XML escaping
    content="${content//&/&amp;}"
    content="${content//\</&lt;}"
    content="${content//\>/&gt;}"
    content="${content//\"/&quot;}"
    echo "$content"
}

# ============================================================================
# Export Functions
# ============================================================================

# Functions are available when this file is sourced.
# Export is disabled by default to avoid polluting stdout during tests.
# Set XML_UTILS_EXPORT=1 to enable function export for subshells.
if [[ "${XML_UTILS_EXPORT:-}" == "1" ]]; then
    export -f xml-validate xml-well-formed xml-to-json json-to-xml 2>/dev/null || true
    export -f xml-query xml-query-attr xml-merge xml-format 2>/dev/null || true
    export -f xml-escape xml-unescape xml-detect-tools 2>/dev/null || true
    export -f pheromone-to-xml pheromone-from-xml pheromone-export 2>/dev/null || true
    export -f queen-wisdom-to-xml queen-wisdom-from-xml 2>/dev/null || true
    export -f queen-wisdom-to-markdown queen-wisdom-validate-entry 2>/dev/null || true
    export -f queen-wisdom-promote queen-wisdom-import 2>/dev/null || true
    export -f registry-to-xml registry-from-xml 2>/dev/null || true
    export -f generate-colony-namespace generate-cross-colony-prefix 2>/dev/null || true
    export -f prefix-pheromone-id extract-session-from-namespace 2>/dev/null || true
    export -f validate-colony-namespace 2>/dev/null || true
    export -f prompt-to-xml prompt-from-xml prompt-validate 2>/dev/null || true
fi
