#!/bin/bash
# Aether Memory Compression Utilities
# Short-term Memory session creation and Working Memory clearing
#
# Usage:
#   source .aether/utils/memory-compress.sh
#   create_short_term_session "phase" "compressed_json"
#   clear_working_memory
#   get_compression_stats

# Source atomic-write utilities
MEMORY_COMPRESS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source atomic-write.sh (handle both sourced and direct execution)
if [ -f "$MEMORY_COMPRESS_DIR/atomic-write.sh" ]; then
    source "$MEMORY_COMPRESS_DIR/atomic-write.sh"
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
MEMORY_FILE="${MEMORY_COMPRESS_DIR}/../data/memory.json"
# If MEMORY_FILE doesn't exist, try the absolute path
if [ ! -f "$MEMORY_FILE" ]; then
    MEMORY_FILE="$(pwd)/.aether/data/memory.json"
fi

# Create Short-term Memory session from compressed JSON
# Arguments: phase, compressed_json
# Returns: session_id
create_short_term_session() {
    local phase="$1"
    local compressed_json="$2"

    if [ -z "$phase" ] || [ -z "$compressed_json" ]; then
        echo "Error: phase and compressed_json are required" >&2
        return 1
    fi

    # Validate compressed_json is valid JSON
    if ! echo "$compressed_json" | jq . >/dev/null 2>&1; then
        echo "Error: compressed_json must be valid JSON" >&2
        return 1
    fi

    # Validate required fields
    local id=$(echo "$compressed_json" | jq -r '.id // empty')
    local session_id=$(echo "$compressed_json" | jq -r '.session_id // empty')
    local compressed_at=$(echo "$compressed_json" | jq -r '.compressed_at // empty')
    local summary=$(echo "$compressed_json" | jq -r '.summary // empty')
    local original_tokens=$(echo "$compressed_json" | jq -r '.original_tokens // 0')
    local compressed_tokens=$(echo "$compressed_json" | jq -r '.compressed_tokens // 0')

    if [ -z "$id" ] || [ -z "$session_id" ] || [ -z "$compressed_at" ] || [ -z "$summary" ]; then
        echo "Error: compressed_json missing required fields (id, session_id, compressed_at, summary)" >&2
        return 1
    fi

    # Calculate compression_ratio if not provided
    local compression_ratio=$(echo "$compressed_json" | jq -r '.compression_ratio // empty')
    if [ -z "$compression_ratio" ] || [ "$compression_ratio" = "null" ]; then
        if [ "$compressed_tokens" -gt 0 ]; then
            compression_ratio=$(echo "scale=2; $original_tokens / $compressed_tokens" | bc)
        else
            compression_ratio=0
        fi
    fi

    # Add session to short_term_memory.sessions array via jq
    jq --argjson session "$compressed_json" \
       --argjson ratio "$compression_ratio" \
       '
       .short_term_memory.sessions += [$session |
         .compression_ratio = $ratio
       ] |
       .short_term_memory.current_sessions += 1 |
       .metrics.total_compressions += 1
       ' "$MEMORY_FILE" > /tmp/memory_add_session.tmp

    # Atomic write
    atomic_write_from_file "$MEMORY_FILE" /tmp/memory_add_session.tmp
    rm -f /tmp/memory_add_session.tmp

    # Check if eviction needed (max 10 sessions)
    current_sessions=$(jq -r '.short_term_memory.current_sessions' "$MEMORY_FILE")
    max_sessions=$(jq -r '.short_term_memory.max_sessions' "$MEMORY_FILE")

    if [ "$current_sessions" -gt "$max_sessions" ]; then
        evict_short_term_session
    fi

    echo "$session_id"
}

# Clear Working Memory after compression
# Arguments: none
# Returns: 0 on success
clear_working_memory() {
    # Set working_memory.items to empty array
    # Set working_memory.current_tokens to 0
    jq '
       .working_memory.items = [] |
       .working_memory.current_tokens = 0
       ' "$MEMORY_FILE" > /tmp/memory_clear_wm.tmp

    # Atomic write
    atomic_write_from_file "$MEMORY_FILE" /tmp/memory_clear_wm.tmp
    rm -f /tmp/memory_clear_wm.tmp

    return 0
}

# Get compression statistics
# Arguments: none
# Returns: Formatted statistics
get_compression_stats() {
    echo "Memory Compression Statistics:"
    echo "=============================="
    echo ""

    # Total compressions
    local total_compressions=$(jq -r '.metrics.total_compressions // 0' "$MEMORY_FILE")
    echo "Total Compressions: $total_compressions"

    # Average compression ratio
    local avg_ratio=$(jq -r '
        if .short_term_memory.sessions | length > 0 then
            ([.short_term_memory.sessions[].compression_ratio // 0] | add / length)
        else
            0
        end
        ' "$MEMORY_FILE")
    printf "Average Compression Ratio: %.2fx\n" "$avg_ratio"

    # Current sessions
    local current_sessions=$(jq -r '.short_term_memory.current_sessions' "$MEMORY_FILE")
    echo "Current Short-term Sessions: $current_sessions / $(jq -r '.short_term_memory.max_sessions' "$MEMORY_FILE")"

    # Working Memory evictions
    local wm_evictions=$(jq -r '.metrics.working_memory_evictions // 0' "$MEMORY_FILE")
    echo "Working Memory Evictions: $wm_evictions"

    # Short-term evictions
    local st_evictions=$(jq -r '.metrics.short_term_evictions // 0' "$MEMORY_FILE")
    echo "Short-term Evictions: $st_evictions"

    echo ""
    echo "Short-term Sessions:"
    echo "--------------------"

    # List all sessions with details
    jq -r '
        .short_term_memory.sessions[]
        | "\(.session_id): Phase \(.phase), \(.compressed_at), \(.original_tokens) → \(.compressed_tokens) tokens (\(.compression_ratio)x)"
        ' "$MEMORY_FILE" 2>/dev/null || echo "  (No sessions yet)"
}

# Evict oldest Short-term Memory session (LRU policy)
# Arguments: none
# Returns: 0 on success
evict_short_term_session() {
    local max_sessions=$(jq -r '.short_term_memory.max_sessions' "$MEMORY_FILE")
    local current_sessions=$(jq -r '.short_term_memory.current_sessions' "$MEMORY_FILE")

    if [ "$current_sessions" -le "$max_sessions" ]; then
        return 0
    fi

    # Get oldest session (sort by compressed_at ascending)
    local oldest_session=$(jq '
        .short_term_memory.sessions
        | sort_by(.compressed_at)
        | .[0]
        ' "$MEMORY_FILE")

    local oldest_id=$(echo "$oldest_session" | jq -r '.id')

    # Remove oldest session and update metrics
    jq --arg id "$oldest_id" \
       '
       .short_term_memory.sessions = [.short_term_memory.sessions[] | select(.id != $id)] |
       .short_term_memory.current_sessions -= 1 |
       .metrics.short_term_evictions += 1
       ' "$MEMORY_FILE" > /tmp/memory_evict_st.tmp

    atomic_write_from_file "$MEMORY_FILE" /tmp/memory_evict_st.tmp
    rm -f /tmp/memory_evict_st.tmp

    return 0
}

# Export functions
export -f create_short_term_session clear_working_memory get_compression_stats evict_short_term_session
