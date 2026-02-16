#!/usr/bin/env bash
# Pheromone-to-XML Comprehensive Test Suite
# Tests for pheromone-to-xml function with XSD validation

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

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# ============================================================================
# Test Helper
# ============================================================================

run_pheromone_test() {
    local test_name="$1"
    local test_json="$2"
    local validation_check="${3:-}"

    local tmp_dir
    tmp_dir=$(mktemp -d)

    echo "$test_json" > "$tmp_dir/test.json"

    local output
    local exit_code=0
    output=$(pheromone-to-xml "$tmp_dir/test.json" "$tmp_dir/test.xml" "$PROJECT_ROOT/.aether/schemas/pheromone.xsd" 2>&1) || exit_code=$?

    if [[ $exit_code -ne 0 ]]; then
        log_error "$test_name: pheromone-to-xml failed with exit code $exit_code"
        echo "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Check validation
    local valid
    valid=$(xml-validate "$tmp_dir/test.xml" "$PROJECT_ROOT/.aether/schemas/pheromone.xsd" 2>&1)
    if ! echo "$valid" | jq -e '.result.valid' > /dev/null 2>&1; then
        log_error "$test_name: XML validation failed"
        cat "$tmp_dir/test.xml"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Run additional checks if provided
    if [[ -n "$validation_check" ]]; then
        if ! eval "$validation_check" "$tmp_dir/test.xml"; then
            log_error "$test_name: Additional validation failed"
            rm -rf "$tmp_dir"
            return 1
        fi
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Tests
# ============================================================================

test_full_pheromone() {
    local json='{
        "version": "1.0.0",
        "colony_id": "test-colony-123",
        "metadata": {
            "source": { "type": "user", "version": "1.0.0" },
            "context": "Test conversion"
        },
        "signals": [
            {
                "id": "sig_001",
                "type": "FOCUS",
                "priority": "high",
                "source": "user",
                "created_at": "2026-02-16T10:00:00Z",
                "expires_at": "2026-02-17T10:00:00Z",
                "active": true,
                "content": { "text": "Test signal" },
                "tags": [{"value": "test", "weight": 1.0, "category": "test"}],
                "scope": { "global": false, "castes": ["builder"], "paths": ["src/**"] }
            }
        ]
    }'

    if run_pheromone_test "Full pheromone with metadata" "$json"; then
        return 0
    else
        return 1
    fi
}

test_legacy_format() {
    local json='{ "id": "sig_002", "type": "FOCUS", "priority": "normal", "message": "Legacy test", "source": "user" }'

    if run_pheromone_test "Legacy single-signal format" "$json"; then
        return 0
    else
        return 1
    fi
}

test_all_signal_types() {
    local json='{
        "signals": [
            { "id": "s1", "type": "FOCUS", "priority": "normal", "source": "user", "content": { "text": "Focus" } },
            { "id": "s2", "type": "REDIRECT", "priority": "high", "source": "user", "content": { "text": "Redirect" } },
            { "id": "s3", "type": "FEEDBACK", "priority": "low", "source": "user", "content": { "text": "Feedback" } }
        ]
    }'

    check_types() {
        local xml_file="$1"
        grep -q 'type="FOCUS"' "$xml_file" && \
        grep -q 'type="REDIRECT"' "$xml_file" && \
        grep -q 'type="FEEDBACK"' "$xml_file"
    }

    if run_pheromone_test "All signal types" "$json" "check_types"; then
        return 0
    else
        return 1
    fi
}

test_all_priorities() {
    local json='{
        "signals": [
            { "id": "p1", "type": "FOCUS", "priority": "critical", "source": "user", "content": { "text": "Critical" } },
            { "id": "p2", "type": "FOCUS", "priority": "high", "source": "user", "content": { "text": "High" } },
            { "id": "p3", "type": "FOCUS", "priority": "normal", "source": "user", "content": { "text": "Normal" } },
            { "id": "p4", "type": "FOCUS", "priority": "low", "source": "user", "content": { "text": "Low" } }
        ]
    }'

    check_priorities() {
        local xml_file="$1"
        grep -q 'priority="critical"' "$xml_file" && \
        grep -q 'priority="high"' "$xml_file" && \
        grep -q 'priority="normal"' "$xml_file" && \
        grep -q 'priority="low"' "$xml_file"
    }

    if run_pheromone_test "All priority levels" "$json" "check_priorities"; then
        return 0
    else
        return 1
    fi
}

test_case_normalization() {
    local json='{ "signals": [{ "id": "s", "type": "focus", "priority": "HIGH", "source": "user", "content": { "text": "Test" } }] }'

    check_normalization() {
        local xml_file="$1"
        grep -q 'type="FOCUS"' "$xml_file" && \
        grep -q 'priority="high"' "$xml_file"
    }

    if run_pheromone_test "Case normalization" "$json" "check_normalization"; then
        return 0
    else
        return 1
    fi
}

test_invalid_fallback() {
    local json='{ "signals": [{ "id": "s", "type": "INVALID", "priority": "INVALID", "source": "user", "content": { "text": "Test" } }] }'

    check_fallback() {
        local xml_file="$1"
        grep -q 'type="FOCUS"' "$xml_file" && \
        grep -q 'priority="normal"' "$xml_file"
    }

    if run_pheromone_test "Invalid type/priority fallback" "$json" "check_fallback"; then
        return 0
    else
        return 1
    fi
}

test_xml_escaping() {
    local json='{ "signals": [{ "id": "s", "type": "FOCUS", "priority": "normal", "source": "user", "content": { "text": "Test with <special> & \"chars\"" } }] }'

    check_escaping() {
        local xml_file="$1"
        grep -q '&lt;special&gt;' "$xml_file"
    }

    if run_pheromone_test "XML special character escaping" "$json" "check_escaping"; then
        return 0
    else
        return 1
    fi
}

test_all_castes() {
    local json='{
        "signals": [{
            "id": "s", "type": "FOCUS", "priority": "normal", "source": "user", "content": { "text": "Test" },
            "scope": { "castes": ["builder", "watcher", "scout", "chaos", "oracle", "architect", "prime", "colonizer", "route_setter", "archaeologist", "ambassador", "auditor", "chronicler", "gatekeeper", "guardian", "includer", "keeper", "measurer", "probe", "sage", "tracker", "weaver"] }
        }]
    }'

    check_castes() {
        local xml_file="$1"
        local count
        count=$(grep -c '<caste>' "$xml_file" || echo "0")
        [[ "$count" -eq 22 ]]
    }

    if run_pheromone_test "All valid castes (22)" "$json" "check_castes"; then
        return 0
    else
        return 1
    fi
}

test_invalid_castes_filtered() {
    local json='{
        "signals": [{
            "id": "s", "type": "FOCUS", "priority": "normal", "source": "user", "content": { "text": "Test" },
            "scope": { "castes": ["builder", "invalid1", "guardian", "invalid2"] }
        }]
    }'

    check_filtered() {
        local xml_file="$1"
        local count
        count=$(grep -c '<caste>' "$xml_file" || echo "0")
        [[ "$count" -eq 2 ]]
    }

    if run_pheromone_test "Invalid castes filtered" "$json" "check_filtered"; then
        return 0
    else
        return 1
    fi
}

test_empty_signals() {
    local json='{ "version": "1.0.0", "signals": [] }'

    if run_pheromone_test "Empty signals array" "$json"; then
        return 0
    else
        return 1
    fi
}

test_tags_with_metadata() {
    local json='{
        "signals": [{
            "id": "s", "type": "FOCUS", "priority": "normal", "source": "user", "content": { "text": "Test" },
            "tags": [
                { "value": "urgent", "weight": 1.0, "category": "priority" },
                { "value": "simple", "weight": 0.5 }
            ]
        }]
    }'

    check_tags() {
        local xml_file="$1"
        grep -q 'category="priority"' "$xml_file" && \
        grep -q 'weight="1.0"' "$xml_file" && \
        grep -q 'weight="0.5"' "$xml_file"
    }

    if run_pheromone_test "Tags with weight and category" "$json" "check_tags"; then
        return 0
    else
        return 1
    fi
}

test_global_scope() {
    local json='{
        "signals": [{
            "id": "s", "type": "FOCUS", "priority": "normal", "source": "user", "content": { "text": "Test" },
            "scope": { "global": true }
        }]
    }'

    check_global() {
        local xml_file="$1"
        grep -q 'global="true"' "$xml_file"
    }

    if run_pheromone_test "Global scope flag" "$json" "check_global"; then
        return 0
    else
        return 1
    fi
}

test_paths_in_scope() {
    local json='{
        "signals": [{
            "id": "s", "type": "FOCUS", "priority": "normal", "source": "user", "content": { "text": "Test" },
            "scope": { "paths": ["src/**/*.js", "tests/**/*.test.js"] }
        }]
    }'

    check_paths() {
        local xml_file="$1"
        grep -q 'src/\*\*/\*.js' "$xml_file"
    }

    if run_pheromone_test "Paths in scope" "$json" "check_paths"; then
        return 0
    else
        return 1
    fi
}

test_data_attachment() {
    local json='{
        "signals": [{
            "id": "s", "type": "FOCUS", "priority": "normal", "source": "user",
            "content": { "text": "Test", "data": { "format": "json", "key": "value" } }
        }]
    }'

    check_data() {
        local xml_file="$1"
        grep -q 'format="json"' "$xml_file"
    }

    if run_pheromone_test "Content with data attachment" "$json" "check_data"; then
        return 0
    else
        return 1
    fi
}

test_in_memory_validation() {
    local tmp_dir
    tmp_dir=$(mktemp -d)

    echo '{ "signals": [{ "id": "s", "type": "FOCUS", "priority": "normal", "source": "user", "content": { "text": "Test" } }] }' > "$tmp_dir/test.json"

    local output
    output=$(pheromone-to-xml "$tmp_dir/test.json" "" "$PROJECT_ROOT/.aether/schemas/pheromone.xsd" 2>&1)

    if ! echo "$output" | jq -e '.ok' > /dev/null 2>&1; then
        log_error "In-memory validation: not ok"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! echo "$output" | jq -e '.result.validated' > /dev/null 2>&1; then
        log_error "In-memory validation: not validated"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! echo "$output" | jq -r '.result.xml' | grep -q '<pheromones'; then
        log_error "In-memory validation: no XML in result"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Main Test Runner
# ============================================================================

main() {
    log "${YELLOW}=== Pheromone-to-XML Test Suite ===${NC}"
    log "Testing: $XML_UTILS_SOURCE"
    log "Schema: $PROJECT_ROOT/.aether/schemas/pheromone.xsd"
    log ""

    run_test "test_full_pheromone" "Full pheromone with all fields"
    run_test "test_legacy_format" "Legacy single-signal format"
    run_test "test_all_signal_types" "All signal types (FOCUS, REDIRECT, FEEDBACK)"
    run_test "test_all_priorities" "All priority levels (critical, high, normal, low)"
    run_test "test_case_normalization" "Case normalization"
    run_test "test_invalid_fallback" "Invalid type/priority fallback"
    run_test "test_xml_escaping" "XML special character escaping"
    run_test "test_all_castes" "All valid castes (22)"
    run_test "test_invalid_castes_filtered" "Invalid castes filtered"
    run_test "test_empty_signals" "Empty signals array"
    run_test "test_tags_with_metadata" "Tags with weight and category"
    run_test "test_global_scope" "Global scope flag"
    run_test "test_paths_in_scope" "Paths in scope"
    run_test "test_data_attachment" "Content with data attachment"
    run_test "test_in_memory_validation" "In-memory validation"

    test_summary
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
