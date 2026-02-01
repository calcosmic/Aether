#!/bin/bash
# Aether Memory Compression Utilities
# Short-term Memory session creation and Working Memory clearing
#
# Usage:
#   source .aether/utils/memory-compress.sh
#   create_short_term_session "phase" "compressed_json"
#   clear_working_memory
#   get_compression_stats

# COMPRESSION TRIGGER WIRING
# ===========================
#
# Who calls what, when:
#
# 1. PHASE BOUNDARY COMPRESSION
#    - Trigger: pheromones.json has phase_complete signal
#    - Caller: Phase boundary orchestrator (future) or manual Queen command
#    - Sequence:
#      a. prepare_compression_data(phase_number)
#         → Creates /tmp/working_memory_for_compression_{phase}.json
#      b. Architect Ant reads temp file, applies DAST compression (LLM task)
#         → Produces compressed_json string
#      c. trigger_phase_boundary_compression(phase_number, compressed_json)
#         → Stores in Short-term, clears Working Memory
#
# 2. TOKEN THRESHOLD COMPRESSION
#    - Trigger: Working Memory reaches 80% capacity (160k tokens)
#    - Caller: auto_compress_if_needed() called during add_working_memory_item
#    - Sequence: Same as phase boundary compression
#
# 3. PATTERN EXTRACTION
#    - Trigger: After Short-term session created, or before eviction
#    - Caller: create_short_term_session() calls trigger_pattern_extraction()
#    - Sequence:
#      a. trigger_pattern_extraction()
#         → Scans Short-term sessions for high-value items
#         → Calls extract_pattern_to_long_term() for repeated patterns
#         → Updates Long-term Memory with associative links
#
# FILES INVOLVED:
# - .aether/utils/memory-compress.sh: This file (bash functions)
# - .aether/workers/architect-ant.md: DAST compression prompt for LLM
# - .aether/data/memory.json: All three memory layers
# - .aether/data/pheromones.json: Phase completion signals
# - .aether/data/COLONY_STATE.json: Current phase tracking

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

# Prepare Working Memory data for compression
# Who calls: Phase boundary orchestrator or manual Queen command
# When: At phase completion or when compression is needed
# Arguments: phase_number
# Returns: 0 (data ready) with file path to stdout, or 1 (skip compression)
# Side effects: Creates temporary file /tmp/working_memory_for_compression_{phase}.json
prepare_compression_data() {
    local phase="$1"

    if [ -z "$phase" ]; then
        echo "Error: phase_number is required" >&2
        return 1
    fi

    # Check if phase is complete (read pheromones.json for phase_complete signal)
    local phase_complete=$(jq -r '.active_pheromones[] | select(.type == "INIT") | .id' "$MEMORY_COMPRESS_DIR/../data/pheromones.json" 2>/dev/null)

    # Get current Working Memory items
    local items=$(jq '.working_memory.items' "$MEMORY_FILE")
    local current_tokens=$(jq -r '.working_memory.current_tokens' "$MEMORY_FILE")
    local item_count=$(jq -r '.working_memory.items | length' "$MEMORY_FILE")

    # If Working Memory is empty or below threshold, return 1 (skip compression)
    if [ "$item_count" -eq 0 ]; then
        echo "Working Memory is empty, skipping compression" >&2
        return 1
    fi

    # Create temporary file with Working Memory items
    local temp_file="/tmp/working_memory_for_compression_${phase}.json"
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    jq -n \
       --arg phase "$phase" \
       --argjson items "$items" \
       --argjson tokens "$current_tokens" \
       --argjson count "$item_count" \
       --arg ts "$timestamp" \
       '{
         "phase": $phase,
         "items": $items,
         "total_tokens": $tokens,
         "item_count": $count,
         "prepared_at": $ts
       }' > "$temp_file"

    # Output file path to stdout (for Architect Ant to read)
    echo "$temp_file"
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

    # Before evicting, check oldest session's high_value_items for patterns
    extract_high_value_patterns "$oldest_id"

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

# Extract pattern to Long-term Memory
# Arguments: pattern_type, pattern_content, confidence, context, source_session_id
# Returns: pattern_id
extract_pattern_to_long_term() {
    local pattern_type="$1"
    local pattern_content="$2"
    local confidence="$3"
    local context="$4"
    local source_session_id="$5"

    if [ -z "$pattern_type" ] || [ -z "$pattern_content" ]; then
        echo "Error: pattern_type and pattern_content are required" >&2
        return 1
    fi

    # Validate pattern_type
    case "$pattern_type" in
        success_pattern|failure_pattern|preference|constraint) ;;
        *)
            echo "Error: pattern_type must be one of: success_pattern, failure_pattern, preference, constraint" >&2
            return 1
            ;;
    esac

    # Generate pattern_id
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local pattern_id="pattern_$(date +%s)_$(echo "$pattern_content" | md5sum | cut -c1-8)"

    # Add pattern to long_term_memory.patterns array
    jq --arg id "$pattern_id" \
       --arg type "$pattern_type" \
       --arg pattern "$pattern_content" \
       --argjson confidence "$confidence" \
       --arg ts "$timestamp" \
       --arg context "$context" \
       '
       .long_term_memory.patterns += [{
         "id": $id,
         "type": $type,
         "pattern": $pattern,
         "confidence": $confidence,
         "occurrences": 1,
         "created_at": $ts,
         "last_seen": $ts,
         "associative_links": [],
         "metadata": {
           "context": $context,
           "related_castes": [],
           "related_phases": []
         }
       }] |
       .metrics.total_pattern_extractions += 1
       ' "$MEMORY_FILE" > /tmp/memory_add_pattern.tmp

    atomic_write_from_file "$MEMORY_FILE" /tmp/memory_add_pattern.tmp
    rm -f /tmp/memory_add_pattern.tmp

    # If source_session_id provided, create associative link
    if [ -n "$source_session_id" ]; then
        create_associative_link "$pattern_id" "short_term_session" "$source_session_id" "extracted_from"
    fi

    echo "$pattern_id"
}

# Extract high-value patterns from a session
# Arguments: session_id
# Returns: number of patterns extracted
extract_high_value_patterns() {
    local session_id="$1"

    if [ -z "$session_id" ]; then
        echo "Error: session_id is required" >&2
        return 1
    fi

    # Get session's high_value_items with relevance_score > 0.8
    local high_value_items=$(jq -r --arg sid "$session_id" \
        '
        .short_term_memory.sessions[]
        | select(.id == $sid)
        | .high_value_items[]
        | select(.relevance_score > 0.8)
        | .content
        ' "$MEMORY_FILE")

    local extracted_count=0

    # Process each high-value item
    while IFS= read -r item_content; do
        [ -z "$item_content" ] && continue

        # Check if similar pattern exists (case-insensitive substring match)
        local existing_pattern=$(jq -r --arg content "$(echo "$item_content" | tr '[:upper:]' '[:lower:]')" \
            '
            .long_term_memory.patterns[]
            | select(.pattern | ascii_downcase | contains($content))
            | .id
            ' "$MEMORY_FILE" 2>/dev/null)

        if [ -n "$existing_pattern" ]; then
            # Pattern exists: increment occurrences, update confidence and last_seen
            local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
            local new_occurrences=$(jq -r --arg pid "$existing_pattern" \
                '
                .long_term_memory.patterns[]
                | select(.id == $pid)
                | .occurrences
                ' "$MEMORY_FILE")

            new_occurrences=$((new_occurrences + 1))

            # Update confidence: min(1.0, current + 0.1)
            local current_confidence=$(jq -r --arg pid "$existing_pattern" \
                '
                .long_term_memory.patterns[]
                | select(.id == $pid)
                | .confidence
                ' "$MEMORY_FILE")

            local new_confidence=$(echo "$current_confidence + 0.1" | bc)
            new_confidence=$(echo "$new_confidence < 1.0 ? $new_confidence : 1.0" | bc -l)

            jq --arg pid "$existing_pattern" \
               --argjson occurrences "$new_occurrences" \
               --argjson confidence "$new_confidence" \
               --arg ts "$timestamp" \
               '
               (.long_term_memory.patterns[] | select(.id == $pid)) |= (
                 .occurrences = $occurrences |
                 .confidence = $confidence |
                 .last_seen = $ts
               )
               ' "$MEMORY_FILE" > /tmp/memory_update_pattern.tmp

            atomic_write_from_file "$MEMORY_FILE" /tmp/memory_update_pattern.tmp
            rm -f /tmp/memory_update_pattern.tmp
        else
            # New pattern: determine type based on content
            local pattern_type="preference"
            if echo "$item_content" | grep -qi "error\|failure\|bug"; then
                pattern_type="failure_pattern"
            elif echo "$item_content" | grep -qi "success\|worked\|achieved"; then
                pattern_type="success_pattern"
            elif echo "$item_content" | grep -qi "must\|should\|constraint\|avoid"; then
                pattern_type="constraint"
            fi

            extract_pattern_to_long_term "$pattern_type" "$item_content" 0.7 "Extracted from session $session_id" "$session_id"
            extracted_count=$((extracted_count + 1))
        fi
    done <<< "$high_value_items"

    echo "$extracted_count"
}

# Detect patterns across all Short-term sessions
# Arguments: none
# Returns: number of patterns detected
detect_patterns_across_sessions() {
    # Get all high_value_items across all sessions
    local all_items=$(jq -r '
        .short_term_memory.sessions[].high_value_items[]
        | select(.relevance_score > 0.8)
        | .content
        ' "$MEMORY_FILE")

    local detected_count=0

    # Count occurrences of each item
    declare -A item_counts
    while IFS= read -r item; do
        [ -z "$item" ] && continue
        item_counts["$item"]=$((${item_counts["$item"]:-0} + 1))
    done <<< "$all_items"

    # For items appearing 3+ times, extract as high-confidence pattern
    for item in "${!item_counts[@]}"; do
        local count=${item_counts[$item]}
        if [ "$count" -ge 3 ]; then
            # Check if pattern already exists
            local exists=$(jq -r --arg content "$(echo "$item" | tr '[:upper:]' '[:lower:]')" \
                '
                .long_term_memory.patterns[]
                | select(.pattern | ascii_downcase | contains($content))
                | .id
                ' "$MEMORY_FILE" 2>/dev/null)

            if [ -z "$exists" ]; then
                # Determine pattern type
                local pattern_type="preference"
                if echo "$item" | grep -qi "error\|failure\|bug"; then
                    pattern_type="failure_pattern"
                elif echo "$item" | grep -qi "success\|worked\|achieved"; then
                    pattern_type="success_pattern"
                elif echo "$item" | grep -qi "must\|should\|constraint\|avoid"; then
                    pattern_type="constraint"
                fi

                # Calculate confidence: 0.5 + occurrences * 0.1, max 1.0
                local confidence=$(echo "0.5 + $count * 0.1" | bc)
                confidence=$(echo "$confidence < 1.0 ? $confidence : 1.0" | bc -l)

                extract_pattern_to_long_term "$pattern_type" "$item" "$confidence" "Detected across $count sessions" ""
                detected_count=$((detected_count + 1))
            fi
        fi
    done

    echo "$detected_count"
}

# Create associative link between items across layers
# Arguments: source_pattern_id, target_type, target_id, link_type
# Returns: link_id
create_associative_link() {
    local source_pattern_id="$1"
    local target_type="$2"
    local target_id="$3"
    local link_type="$4"

    if [ -z "$source_pattern_id" ] || [ -z "$target_type" ] || [ -z "$target_id" ] || [ -z "$link_type" ]; then
        echo "Error: source_pattern_id, target_type, target_id, and link_type are required" >&2
        return 1
    fi

    # Validate target_type
    case "$target_type" in
        short_term_session|working_memory_item) ;;
        *)
            echo "Error: target_type must be 'short_term_session' or 'working_memory_item'" >&2
            return 1
            ;;
    esac

    # Validate link_type
    case "$link_type" in
        originated_from|related_to|extracted_from) ;;
        *)
            echo "Error: link_type must be one of: originated_from, related_to, extracted_from" >&2
            return 1
            ;;
    esac

    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local link_id="link_$(date +%s)_$(echo "$source_pattern_id$target_id$link_type" | md5sum | cut -c1-8)"

    # Add link to pattern's associative_links array
    jq --arg pid "$source_pattern_id" \
       --arg lid "$link_id" \
       --arg tid "$target_id" \
       --arg ttype "$target_type" \
       --arg ltype "$link_type" \
       --arg ts "$timestamp" \
       '
       (.long_term_memory.patterns[] | select(.id == $pid)) |= (
         .associative_links += [{
           "link_id": $lid,
           "target_id": $tid,
           "target_type": $ttype,
           "link_type": $ltype,
           "created_at": $ts
         }]
       )
       ' "$MEMORY_FILE" > /tmp/memory_add_link.tmp

    atomic_write_from_file "$MEMORY_FILE" /tmp/memory_add_link.tmp
    rm -f /tmp/memory_add_link.tmp

    # If target_type is short_term_session, add reverse link to session
    if [ "$target_type" = "short_term_session" ]; then
        jq --arg sid "$target_id" \
           --arg pid "$source_pattern_id" \
           --arg lid "$link_id" \
           '
           (.short_term_memory.sessions[] | select(.id == $sid)) |= (
             .metadata.related_patterns += [$pid]
           )
           ' "$MEMORY_FILE" > /tmp/memory_add_reverse_link.tmp

        atomic_write_from_file "$MEMORY_FILE" /tmp/memory_add_reverse_link.tmp
        rm -f /tmp/memory_add_reverse_link.tmp
    fi

    echo "$link_id"
}

# Export functions
export -f create_short_term_session clear_working_memory get_compression_stats evict_short_term_session
export -f extract_pattern_to_long_term extract_high_value_patterns detect_patterns_across_sessions create_associative_link
