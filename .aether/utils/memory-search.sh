#!/bin/bash
# Aether Memory Search Utilities
# Cross-layer search with relevance ranking
#
# Usage:
#   source .aether/utils/memory-search.sh
#   search_memory "query" [limit]
#   search_working_memory "query" [limit]
#   search_short_term_memory "query" [limit]
#   search_long_term_memory "query" [limit]

# Source atomic-write utilities
MEMORY_SEARCH_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source atomic-write.sh (handle both sourced and direct execution)
if [ -f "$MEMORY_SEARCH_DIR/atomic-write.sh" ]; then
    source "$MEMORY_SEARCH_DIR/atomic-write.sh"
else
    # Fallback to common location
    if [ -f ".aether/utils/atomic-write.sh" ]; then
        source .aether/utils/atomic-write.sh
    else
        echo "Error: Cannot find atomic-write.sh" >&2
        exit 1
    fi
fi

# Memory file location
MEMORY_FILE="${MEMORY_SEARCH_DIR}/../data/memory.json"
# If MEMORY_FILE doesn't exist, try the absolute path
if [ ! -f "$MEMORY_FILE" ]; then
    MEMORY_FILE="$(pwd)/.aether/data/memory.json"
fi

# Search Working Memory for query
# Arguments: query, limit (optional, default 20)
# Returns: JSON array of matches with relevance scores
# Side effects: Updates access_count and last_accessed for matched items
search_working_memory() {
    local query="$1"
    local limit="${2:-20}"

    if [ -z "$query" ]; then
        echo "Error: query is required" >&2
        return 1
    fi

    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Search and update access metadata in single jq operation
    # For exact match (content equals query): relevance = 1.0
    # For contains match (content includes query): relevance = 0.7
    # Sort by relevance (descending), then last_accessed (descending)
    jq --arg query "$query" \
       --arg timestamp "$timestamp" \
       --argjson limit "$limit" \
       '
       # Update access metadata for matches
       .working_memory.items |= map(
         if .content | ascii_downcase | contains($query | ascii_downcase) then
           .metadata.last_accessed = $timestamp |
           .metadata.access_count += 1
         else
           .
         end
       ) |
       # Find matches and calculate relevance
       .working_memory.items
       | map(select(.content | ascii_downcase | contains($query | ascii_downcase)))
       | map(. + {
           relevance: (if .content | ascii_downcase == ($query | ascii_downcase) then
                        1.0
                      else
                        0.7
                      end),
           layer: "working_memory"
         })
       | sort_by(.relevance, .metadata.last_accessed) | reverse
       | .[0:$limit]
       ' "$MEMORY_FILE"

    return 0
}

# Search Short-term Memory for query
# Arguments: query, limit (optional, default 20)
# Returns: JSON array of matches with relevance scores
search_short_term_memory() {
    local query="$1"
    local limit="${2:-20}"

    if [ -z "$query" ]; then
        echo "Error: query is required" >&2
        return 1
    fi

    # Search in summary, key_decisions[], and outcomes[]
    jq --arg query "$query" \
       --argjson limit "$limit" \
       '
       .short_term_memory.sessions
       | map(
           # Check summary
           (.summary | ascii_downcase | contains($query | ascii_downcase)) as $summary_match |
           # Check key_decisions
           ([.key_decisions[].decision | ascii_downcase | contains($query | ascii_downcase)] | any) as $decisions_match |
           # Check outcomes
           ([.outcomes[].result | ascii_downcase | contains($query | ascii_downcase)] | any) as $outcomes_match |
           select($summary_match or $decisions_match or $outcomes_match)
         )
       | map(. + {
           relevance: 0.7,
           layer: "short_term_memory"
         })
       | sort_by(.compressed_at) | reverse
       | .[0:$limit]
       ' "$MEMORY_FILE"

    return 0
}

# Search Long-term Memory for query
# Arguments: query, limit (optional, default 20)
# Returns: JSON array of matches with relevance = pattern.confidence
search_long_term_memory() {
    local query="$1"
    local limit="${2:-20}"

    if [ -z "$query" ]; then
        echo "Error: query is required" >&2
        return 1
    fi

    # Search in pattern field, relevance = pattern.confidence
    jq --arg query "$query" \
       --argjson limit "$limit" \
       '
       .long_term_memory.patterns
       | map(select(.pattern | ascii_downcase | contains($query | ascii_downcase)))
       | map(. + {
           relevance: .confidence,
           layer: "long_term_memory"
         })
       | sort_by(.confidence) | reverse
       | .[0:$limit]
       ' "$MEMORY_FILE"

    return 0
}

# Cross-layer search: combine results from all three layers
# Arguments: query, limit_per_layer (optional, default 20)
# Returns: Combined JSON array ranked by layer (working first) and relevance
search_memory() {
    local query="$1"
    local limit="${2:-20}"

    if [ -z "$query" ]; then
        echo "Error: query is required" >&2
        return 1
    fi

    # Get results from each layer
    local working_results=$(search_working_memory "$query" "$limit")
    local short_term_results=$(search_short_term_memory "$query" "$limit")
    local long_term_results=$(search_long_term_memory "$query" "$limit")

    # Combine results using jq
    # Sort by: layer (working first), then relevance (descending), then timestamp (descending)
    jq -n \
       --argjson working "$working_results" \
       --argjson short_term "$short_term_results" \
       --argjson long_term "$long_term_results" \
       '
       ($working + $short_term + $long_term)
       | map(. + {
           layer_priority: (if .layer == "working_memory" then
                              0
                            elif .layer == "short_term_memory" then
                              1
                            else
                              2
                            end)
         })
       | sort_by(.layer_priority, .relevance) | reverse
       '

    return 0
}

# Get memory status with formatted statistics
# Arguments: none
# Returns: Formatted memory statistics
get_memory_status() {
    # Gather statistics
    local wm_items=$(jq -r '.working_memory.items | length' "$MEMORY_FILE")
    local wm_current=$(jq -r '.working_memory.current_tokens' "$MEMORY_FILE")
    local wm_max=$(jq -r '.working_memory.max_capacity_tokens' "$MEMORY_FILE")
    local wm_percent=$(echo "scale=1; $wm_current * 100 / $wm_max" | bc)
    local threshold=$((wm_max * 80 / 100))

    local st_sessions=$(jq -r '.short_term_memory.current_sessions' "$MEMORY_FILE")
    local st_max=$(jq -r '.short_term_memory.max_sessions' "$MEMORY_FILE")
    local st_percent=$(echo "scale=1; $st_sessions * 100 / $st_max" | bc)

    local lt_patterns=$(jq -r '.long_term_memory.patterns | length' "$MEMORY_FILE")
    local lt_success=$(jq -r '[.long_term_memory.patterns[] | select(.type == "success_pattern")] | length' "$MEMORY_FILE")
    local lt_failure=$(jq -r '[.long_term_memory.patterns[] | select(.type == "failure_pattern")] | length' "$MEMORY_FILE")
    local lt_preference=$(jq -r '[.long_term_memory.patterns[] | select(.type == "preference")] | length' "$MEMORY_FILE")
    local lt_constraint=$(jq -r '[.long_term_memory.patterns[] | select(.type == "constraint")] | length' "$MEMORY_FILE")

    local total_compressions=$(jq -r '.metrics.total_compressions' "$MEMORY_FILE")
    local avg_ratio=$(jq -r '.metrics.average_compression_ratio' "$MEMORY_FILE")
    local wm_evictions=$(jq -r '.metrics.working_memory_evictions' "$MEMORY_FILE")
    local st_evictions=$(jq -r '.metrics.short_term_evictions' "$MEMORY_FILE")
    local pattern_extractions=$(jq -r '.metrics.total_pattern_extractions' "$MEMORY_FILE")

    # Format output
    echo "MEMORY STATUS"
    echo ""
    echo "Working Memory:"
    echo "  Items: $wm_items"
    echo "  Tokens: $wm_current / $wm_max (200,000) (${wm_percent}%)"
    echo "  Eviction Threshold: ${threshold} tokens (80%)"
    echo "  Max Capacity: 200,000 tokens (hard limit)"
    echo ""
    echo "Short-term Memory:"
    echo "  Sessions: $st_sessions / $st_max (${st_percent}%)"
    echo "  Compression Ratio: 2.5x target"
    echo ""
    echo "Long-term Memory:"
    echo "  Patterns: $lt_patterns"
    echo "  Types: success=$lt_success, failure=$lt_failure, preference=$lt_preference, constraint=$lt_constraint"
    echo ""
    echo "Metrics:"
    echo "  Total Compressions: $total_compressions"
    echo "  Average Compression Ratio: $avg_ratio"
    echo "  Working Memory Evictions: $wm_evictions"
    echo "  Short-term Evictions: $st_evictions"
    echo "  Pattern Extractions: $pattern_extractions"
}

# Verify 200k token limit enforcement
# Arguments: none
# Returns: 0 if all checks pass, 1 if violation detected
verify_token_limit() {
    local max_capacity=$(jq -r '.working_memory.max_capacity_tokens' "$MEMORY_FILE")
    local current_tokens=$(jq -r '.working_memory.current_tokens' "$MEMORY_FILE")
    local threshold=$((max_capacity * 80 / 100))
    local percent=$(echo "scale=1; $current_tokens * 100 / $max_capacity" | bc)

    echo "TOKEN LIMIT VERIFICATION"
    echo "Max Capacity: ${max_capacity} tokens"
    echo "Current Usage: ${current_tokens} tokens (${percent}%)"
    echo "Compression Threshold: ${threshold} tokens (80%)"

    # Check violations
    if [ "$current_tokens" -gt "$max_capacity" ]; then
        echo "Status: FAIL - Exceeded hard limit"
        return 1
    elif [ "$current_tokens" -gt "$threshold" ]; then
        echo "Status: WARNING - Approaching compression threshold"
        return 0
    else
        echo "Status: PASS - Current usage within safe limits"
        return 0
    fi
}

# Export functions
export -f search_memory search_working_memory search_short_term_memory search_long_term_memory
export -f get_memory_status verify_token_limit
