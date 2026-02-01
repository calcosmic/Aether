#!/bin/bash
# Aether Memory Operations
# Working Memory read/write/update operations with LRU eviction policy
#
# Usage:
#   source .aether/utils/memory-ops.sh
#   add_working_memory_item "content" "type" relevance_score
#   get_working_memory_item "item_id"
#   update_working_memory_item "item_id" '{"field": "value"}'
#   list_working_memory_items [limit]

# Source atomic-write utilities
MEMORY_OPS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source atomic-write.sh (handle both sourced and direct execution)
if [ -f "$MEMORY_OPS_DIR/atomic-write.sh" ]; then
    source "$MEMORY_OPS_DIR/atomic-write.sh"
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
MEMORY_FILE="${MEMORY_OPS_DIR}/../data/memory.json"
# If MEMORY_FILE doesn't exist, try the absolute path
if [ ! -f "$MEMORY_FILE" ]; then
    MEMORY_FILE="$(pwd)/.aether/data/memory.json"
fi

# Add item to Working Memory
# Arguments: content, type, relevance_score (optional, default 0.5)
# Returns: item_id
add_working_memory_item() {
    local content="$1"
    local item_type="$2"
    local relevance="${3:-0.5}"

    # Validate inputs
    if [ -z "$content" ] || [ -z "$item_type" ]; then
        echo "Error: content and type are required" >&2
        return 1
    fi

    # Generate metadata
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    item_id="wm_$(date +%s)_$(echo "$content" | md5sum | cut -c1-8)"

    # Estimate tokens: 4 chars per token heuristic (95% accurate)
    token_count=$(( ( ${#content} + 3 ) / 4 ))

    # Check capacity and evict if needed
    current_tokens=$(jq -r '.working_memory.current_tokens' "$MEMORY_FILE")
    max_tokens=$(jq -r '.working_memory.max_capacity_tokens' "$MEMORY_FILE")
    threshold=$(( max_tokens * 80 / 100 ))

    if [ $(( current_tokens + token_count )) -gt "$threshold" ]; then
        evict_lru_working_memory "$token_count"
    fi

    # Add item via jq
    jq --arg id "$item_id" \
       --arg timestamp "$timestamp" \
       --arg content "$content" \
       --arg type "$item_type" \
       --argjson relevance "$relevance" \
       --argjson tokens "$token_count" \
       '
       .working_memory.items += [{
         "id": $id,
         "type": $type,
         "content": $content,
         "metadata": {
           "timestamp": $timestamp,
           "relevance_score": $relevance,
           "access_count": 0,
           "last_accessed": $timestamp,
           "source": "queen",
           "caste": null
         },
         "associative_links": [],
         "token_count": $tokens
       }] |
       .working_memory.current_tokens += $tokens
       ' "$MEMORY_FILE" > /tmp/memory_add.tmp

    # Atomic write
    atomic_write_from_file "$MEMORY_FILE" /tmp/memory_add.tmp
    rm -f /tmp/memory_add.tmp

    echo "$item_id"
}

# Get item from Working Memory by ID
# Arguments: item_id
# Returns: JSON object of the item
get_working_memory_item() {
    local item_id="$1"

    if [ -z "$item_id" ]; then
        echo "Error: item_id is required" >&2
        return 1
    fi

    # Update access metadata and retrieve item
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Get the item and update access_count + last_accessed
    jq --arg id "$item_id" \
       --arg timestamp "$timestamp" \
       '
       .working_memory.items[] |
       select(.id == $id) |
       .metadata.last_accessed = $timestamp |
       .metadata.access_count += 1
       ' "$MEMORY_FILE" > /tmp/memory_get_item.tmp

    # Update the item in memory.json
    jq --arg id "$item_id" \
       --arg timestamp "$timestamp" \
       '
       .working_memory.items |= map(
         if .id == $id then
           .metadata.last_accessed = $timestamp |
           .metadata.access_count += 1
         else
           .
         end
       )
       ' "$MEMORY_FILE" > /tmp/memory_update_access.tmp

    atomic_write_from_file "$MEMORY_FILE" /tmp/memory_update_access.tmp
    rm -f /tmp/memory_update_access.tmp

    # Output the item
    cat /tmp/memory_get_item.tmp
    rm -f /tmp/memory_get_item.tmp
}

# Update item in Working Memory
# Arguments: item_id, updates_json
# Returns: 0 on success
update_working_memory_item() {
    local item_id="$1"
    local updates_json="$2"

    if [ -z "$item_id" ] || [ -z "$updates_json" ]; then
        echo "Error: item_id and updates_json are required" >&2
        return 1
    fi

    # Validate updates_json is valid JSON
    if ! echo "$updates_json" | jq . >/dev/null 2>&1; then
        echo "Error: updates_json must be valid JSON" >&2
        return 1
    fi

    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Merge updates with existing item
    jq --arg id "$item_id" \
       --arg timestamp "$timestamp" \
       --argjson updates "$updates_json" \
       '
       .working_memory.items |= map(
         if .id == $id then
           .metadata.last_accessed = $timestamp |
           (. + $updates)
         else
           .
         end
       )
       ' "$MEMORY_FILE" > /tmp/memory_update.tmp

    atomic_write_from_file "$MEMORY_FILE" /tmp/memory_update.tmp
    rm -f /tmp/memory_update.tmp

    return 0
}

# List all Working Memory items (or limited count)
# Arguments: limit (optional)
# Returns: JSON array of items sorted by last_accessed descending
list_working_memory_items() {
    local limit="${1:-0}"

    if [ "$limit" -eq 0 ]; then
        # Return all items
        jq '
            .working_memory.items
            | sort_by(.metadata.last_accessed) | reverse
            ' "$MEMORY_FILE"
    else
        # Return limited items
        jq --argjson limit "$limit" '
            .working_memory.items
            | sort_by(.metadata.last_accessed) | reverse
            | .[0:$limit]
            ' "$MEMORY_FILE"
    fi
}

# Evict oldest items from Working Memory (LRU policy)
# Arguments: needed_tokens
# Returns: 0 on success
evict_lru_working_memory() {
    local needed_tokens="$1"

    if [ -z "$needed_tokens" ]; then
        echo "Error: needed_tokens is required" >&2
        return 1
    fi

    # Get current state
    current_tokens=$(jq -r '.working_memory.current_tokens' "$MEMORY_FILE")
    max_tokens=$(jq -r '.working_memory.max_capacity_tokens' "$MEMORY_FILE")
    threshold=$(( max_tokens * 80 / 100 ))

    # Only evict if above threshold
    if [ "$current_tokens" -lt "$threshold" ]; then
        return 0
    fi

    # Evict items until we have enough space
    while true; do
        current_tokens=$(jq -r '.working_memory.current_tokens' "$MEMORY_FILE")
        available=$(( max_tokens - current_tokens ))

        if [ "$available" -ge "$needed_tokens" ]; then
            break
        fi

        # Get oldest item (sort by last_accessed ascending)
        oldest_item=$(jq '
            .working_memory.items
            | sort_by(.metadata.last_accessed)
            | .[0]
            ' "$MEMORY_FILE")

        # Check if there are any items to evict
        if [ -z "$oldest_item" ] || [ "$oldest_item" = "null" ]; then
            break
        fi

        oldest_id=$(echo "$oldest_item" | jq -r '.id')
        oldest_tokens=$(echo "$oldest_item" | jq -r '.token_count // 0')

        # Remove oldest item and update metrics
        jq --arg id "$oldest_id" \
           --argjson tokens "$oldest_tokens" \
           '
           .working_memory.items = [.working_memory.items[] | select(.id != $id)] |
           .working_memory.current_tokens -= $tokens |
           .metrics.working_memory_evictions += 1
           ' "$MEMORY_FILE" > /tmp/memory_evict.tmp

        atomic_write_from_file "$MEMORY_FILE" /tmp/memory_evict.tmp
        rm -f /tmp/memory_evict.tmp
    done

    return 0
}

# Export functions
export -f add_working_memory_item get_working_memory_item update_working_memory_item list_working_memory_items evict_lru_working_memory
