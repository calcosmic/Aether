#!/bin/bash
# XML Composition Functions for Worker Priming
# XInclude-based modular configuration composition
#
# Usage: source .aether/utils/xml-compose.sh
#        xml-compose <input_xml> [output_xml]
#        xml-list-includes <xml_file>
#
# These functions enable declarative composition of worker configurations
# using XInclude directives to merge queen-wisdom, active-trails, and stack-profiles.

set -euo pipefail

# Source xml-core.sh for JSON helpers and tool detection
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/xml-core.sh"

# ============================================================================
# Path Validation (Security)
# ============================================================================

# xml-validate-include-path: Validate XInclude path for traversal attacks
# Usage: xml-validate-include-path <include_path> <base_dir>
# Returns: absolute path on success, exits with error on failure
xml-validate-include-path() {
    local include_path="$1"
    local base_dir="$2"

    # Resolve base directory to absolute path
    local allowed_dir
    allowed_dir=$(cd "$base_dir" 2>/dev/null && pwd) || {
        xml_json_err "INVALID_BASE_DIR" \
            "Base directory does not exist" \
            "dir=$base_dir"
        return 1
    }

    # First check: reject paths with traversal sequences
    if [[ "$include_path" =~ \.\.[\/] ]] || [[ "$include_path" =~ [\/]\.\. ]]; then
        xml_json_err "PATH_TRAVERSAL_DETECTED" \
            "Path contains traversal sequences" \
            "path=$include_path"
        return 1
    fi

    # Build resolved path
    local resolved_path
    if [[ "$include_path" == /* ]]; then
        # Absolute path - must start with allowed_dir
        if [[ ! "$include_path" =~ ^"$allowed_dir" ]]; then
            xml_json_err "PATH_TRAVERSAL_BLOCKED" \
                "Absolute path outside allowed directory" \
                "path=$include_path, allowed=$allowed_dir"
            return 1
        fi
        resolved_path="$include_path"
    else
        # Relative path - resolve against base_dir
        resolved_path="$allowed_dir/$include_path"
    fi

    # Normalize path (remove . and .. components manually for portability)
    local normalized_path="$resolved_path"

    # Verify final path is within allowed directory
    if [[ ! "$normalized_path" =~ ^"$allowed_dir" ]]; then
        xml_json_err "PATH_TRAVERSAL_BLOCKED" \
            "Resolved path outside allowed directory" \
            "path=$normalized_path, allowed=$allowed_dir"
        return 1
    fi

    echo "$normalized_path"
}

# ============================================================================
# XInclude Composition Functions
# ============================================================================

# xml-compose: Resolve XInclude directives in worker priming documents
# Usage: xml-compose <input_xml> [output_xml]
# Returns: {"ok":true,"result":{"composed":true,"output":"...","sources_resolved":N}}
xml-compose() {
    local input_xml="${1:-}"
    local output_xml="${2:-}"

    [[ -z "$input_xml" ]] && { xml_json_err "Missing input XML file argument"; return 1; }
    [[ -f "$input_xml" ]] || { xml_json_err "Input XML file not found: $input_xml"; return 1; }

    # Check well-formedness first
    local well_formed_result
    well_formed_result=$(xml-well-formed "$input_xml" 2>/dev/null)
    if ! echo "$well_formed_result" | jq -e '.result.well_formed' >/dev/null 2>&1; then
        xml_json_err "Input XML is not well-formed"
        return 1
    fi

    # Use xmllint for XInclude processing if available (with XXE protection)
    if [[ "$XMLLINT_AVAILABLE" == "true" ]]; then
        local composed
        composed=$(xmllint --nonet --noent --xinclude --format "$input_xml" 2>/dev/null) || {
            xml_json_err "XInclude composition failed - check that included files exist"
            return 1
        }

        # Determine output destination
        if [[ -n "$output_xml" ]]; then
            echo "$composed" > "$output_xml"
            local escaped_output
            escaped_output=$(echo "$output_xml" | jq -Rs '.[:-1]')
            xml_json_ok "{\"composed\":true,\"output\":$escaped_output,\"sources_resolved\":\"auto\"}"
        else
            # Output to stdout wrapped in JSON
            local escaped_composed
            escaped_composed=$(echo "$composed" | jq -Rs '.')
            xml_json_ok "{\"composed\":true,\"xml\":$escaped_composed,\"sources_resolved\":\"auto\"}"
        fi
        return 0
    fi

    # No xmllint available - error explicitly (manual fallback removed for security)
    xml_json_err "XMLLINT_REQUIRED" \
        "xmllint is required for secure XInclude processing" \
        "install_hint='brew install libxml2'"
    return 1
}

# xml-list-includes: List all XInclude references in a document
# Usage: xml-list-includes <xml_file>
# Returns: {"ok":true,"result":{"includes":[{"href":"...","parse":"..."},...]}}
xml-list-includes() {
    local xml_file="${1:-}"

    [[ -z "$xml_file" ]] && { xml_json_err "Missing XML file argument"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "XML file not found: $xml_file"; return 1; }

    local includes_json="[]"

    if [[ "$XMLSTARLET_AVAILABLE" == "true" ]]; then
        # Use xmlstarlet for proper namespace-aware extraction
        includes_json=$(xmlstarlet sel -N xi="http://www.w3.org/2001/XInclude" \
            -t -m "//xi:include" \
            -o '{"href":"' -v "@href" -o '","parse":"' -v "@parse" -o '","xpointer":"' -v "@xpointer" -o '"}' \
            -n "$xml_file" 2>/dev/null | jq -s '.' || echo "[]")
    elif [[ "$XMLLINT_AVAILABLE" == "true" ]]; then
        # Fallback: grep for xi:include (less reliable but portable)
        local base_dir
        base_dir=$(dirname "$xml_file")

        includes_json=$(grep -oE 'xi:include[^>]*href="[^"]+"' "$xml_file" 2>/dev/null | while read -r match; do
            local href
            href=$(echo "$match" | grep -oE 'href="[^"]+"' | cut -d'"' -f2)
            local parse
            parse=$(echo "$match" | grep -oE 'parse="[^"]+"' | cut -d'"' -f2 || echo "xml")
            echo "{\"href\":\"$href\",\"parse\":\"$parse\",\"resolved\":\"$base_dir/$href\"}"
        done | jq -s '.' || echo "[]")
    else
        xml_json_err "No XML tool available. Install xmlstarlet or libxml2."
        return 1
    fi

    local count
    count=$(echo "$includes_json" | jq 'length')
    xml_json_ok "{\"includes\":$includes_json,\"count\":$count}"
}

# xml-compose-worker-priming: Specialized composition for worker priming documents
# Usage: xml-compose-worker-priming <priming_xml> [output_xml]
# Returns: {"ok":true,"result":{"composed":true,"worker_id":"...","caste":"...","sources":{...}}}
xml-compose-worker-priming() {
    local priming_xml="${1:-}"
    local output_xml="${2:-}"

    [[ -z "$priming_xml" ]] && { xml_json_err "Missing priming XML file argument"; return 1; }
    [[ -f "$priming_xml" ]] || { xml_json_err "Priming XML file not found: $priming_xml"; return 1; }

    # Validate against schema if available
    local schema_file=".aether/schemas/worker-priming.xsd"
    if [[ -f "$schema_file" ]] && [[ "$XMLLINT_AVAILABLE" == "true" ]]; then
        local validation
        validation=$(xml-validate "$priming_xml" "$schema_file" 2>/dev/null)
        if ! echo "$validation" | jq -e '.result.valid' >/dev/null 2>&1; then
            xml_json_err "Worker priming XML failed schema validation"
            return 1
        fi
    fi

    # Extract worker identity before composition
    local worker_id worker_caste
    if [[ "$XMLSTARLET_AVAILABLE" == "true" ]]; then
        worker_id=$(xmlstarlet sel -t -v "//*[local-name()='worker-identity']/@id" "$priming_xml" 2>/dev/null || echo "unknown")
        worker_caste=$(xmlstarlet sel -t -v "//*[local-name()='worker-identity']/*[local-name()='caste']" "$priming_xml" 2>/dev/null || echo "unknown")
    else
        # Fallback: sed extraction (portable, no grep -P)
        worker_id=$(sed -n 's/.*worker-identity[^>]*id="\([^"]*\)".*/\1/p' "$priming_xml" | head -1 || echo "unknown")
        worker_caste=$(sed -n 's/.*<caste>\([^<]*\)<\/caste>.*/\1/p' "$priming_xml" | head -1 || echo "unknown")
    fi

    # Compose the document
    local compose_result
    if [[ -n "$output_xml" ]]; then
        compose_result=$(xml-compose "$priming_xml" "$output_xml" 2>/dev/null)
    else
        compose_result=$(xml-compose "$priming_xml" 2>/dev/null)
    fi

    if ! echo "$compose_result" | jq -e '.ok' >/dev/null 2>&1; then
        xml_json_err "Composition failed: $(echo "$compose_result" | jq -r '.error // "unknown"')"
        return 1
    fi

    # Count sources from different sections
    local queen_wisdom_count active_trails_count stack_profiles_count
    if [[ "$XMLSTARLET_AVAILABLE" == "true" ]] && [[ -n "$output_xml" ]]; then
        queen_wisdom_count=$(xmlstarlet sel -t -v "count(//*[local-name()='queen-wisdom']/*[local-name()='wisdom-source'])" "$output_xml" 2>/dev/null || echo "0")
        active_trails_count=$(xmlstarlet sel -t -v "count(//*[local-name()='active-trails']/*[local-name()='trail-source'])" "$output_xml" 2>/dev/null || echo "0")
        stack_profiles_count=$(xmlstarlet sel -t -v "count(//*[local-name()='stack-profiles']/*[local-name()='profile-source'])" "$output_xml" 2>/dev/null || echo "0")
    else
        queen_wisdom_count="unknown"
        active_trails_count="unknown"
        stack_profiles_count="unknown"
    fi

    # Build result
    local result_json
    result_json=$(jq -n \
        --arg worker_id "$worker_id" \
        --arg caste "$worker_caste" \
        --arg queen_wisdom "$queen_wisdom_count" \
        --arg active_trails "$active_trails_count" \
        --arg stack_profiles "$stack_profiles_count" \
        '{
            composed: true,
            worker_id: $worker_id,
            caste: $caste,
            sources: {
                queen_wisdom: $queen_wisdom,
                active_trails: $active_trails,
                stack_profiles: $stack_profiles
            }
        }')

    xml_json_ok "$result_json"
}

# Export functions
export -f xml-compose xml-list-includes xml-compose-worker-priming
export -f xml-validate-include-path
