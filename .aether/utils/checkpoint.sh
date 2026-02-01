#!/bin/bash
# Aether Checkpoint Utility
# Implements checkpoint save/load/rotate functions for colony recovery
#
# Usage:
#   source .aether/utils/checkpoint.sh
#   save_checkpoint "pre_IDLE_to_INIT"
#   load_checkpoint "latest"
#   list_checkpoints

# Source required utilities
_AETHER_UTILS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$_AETHER_UTILS_DIR/atomic-write.sh"

# Checkpoint constants
CHECKPOINT_DIR=".aether/data/checkpoints"
CHECKPOINT_FILE=".aether/data/checkpoint.json"
COLONY_STATE=".aether/data/COLONY_STATE.json"
PHEROMONES_FILE=".aether/data/pheromones.json"
WORKER_ANTS_FILE=".aether/data/worker_ants.json"
MEMORY_FILE=".aether/data/memory.json"

# Save checkpoint with complete colony state
# Args: label (e.g., "pre_IDLE_to_INIT")
# Returns: 0 on success, 1 on failure
save_checkpoint() {
    local label="$1"

    # Create checkpoint directory if not exists
    mkdir -p "$CHECKPOINT_DIR"

    # Generate checkpoint metadata
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Get next checkpoint number from COLONY_STATE
    local checkpoint_num=$(jq -r '.checkpoints.checkpoint_count // 0' "$COLONY_STATE")
    local checkpoint_file="checkpoint_${checkpoint_num}.json"

    # Capture complete colony state using jq
    local temp_file="/tmp/checkpoint.$$.tmp"

    if ! jq --arg label "$label" \
       --arg timestamp "$timestamp" \
       --arg num "$checkpoint_num" \
       '{
         "checkpoint_id": $num,
         "label": $label,
         "timestamp": $timestamp,
         "colony_state": .,
         "pheromones": input,
         "worker_ants": input,
         "memory": input
       }' \
       "$COLONY_STATE" \
       "$PHEROMONES_FILE" \
       "$WORKER_ANTS_FILE" \
       "$MEMORY_FILE" > "$temp_file" 2>/dev/null; then
        echo "Failed to create checkpoint JSON" >&2
        rm -f "$temp_file"
        return 1
    fi

    # Validate JSON integrity
    if ! python3 -c "import json; json.load(open('$temp_file'))" 2>/dev/null; then
        echo "Checkpoint validation failed" >&2
        rm -f "$temp_file"
        return 1
    fi

    # Atomic write to checkpoint archive
    local checkpoint_path="${CHECKPOINT_DIR}/${checkpoint_file}"
    if ! atomic_write_from_file "$checkpoint_path" "$temp_file"; then
        echo "Failed to write checkpoint atomically" >&2
        rm -f "$temp_file"
        return 1
    fi

    # Cleanup temp file
    rm -f "$temp_file"

    # Update latest checkpoint reference
    if ! echo "$checkpoint_path" > "$CHECKPOINT_FILE.tmp"; then
        echo "Failed to create checkpoint reference" >&2
        return 1
    fi
    mv "$CHECKPOINT_FILE.tmp" "$CHECKPOINT_FILE"

    # Update checkpoint count in COLONY_STATE
    local state_temp="/tmp/colony_state.$$.tmp"
    if ! jq --arg checkpoint "$checkpoint_file" \
       '.checkpoints.latest_checkpoint = $checkpoint | .checkpoints.checkpoint_count += 1' \
       "$COLONY_STATE" > "$state_temp"; then
        echo "Failed to update checkpoint count" >&2
        rm -f "$state_temp"
        return 1
    fi

    if ! atomic_write_from_file "$COLONY_STATE" "$state_temp"; then
        echo "Failed to update COLONY_STATE" >&2
        rm -f "$state_temp"
        return 1
    fi

    rm -f "$state_temp"

    # Rotate old checkpoints (keep last 10)
    rotate_checkpoints

    # Echo confirmation
    echo "Checkpoint saved: $checkpoint_path"
    echo "Label: $label"

    return 0
}

# Load checkpoint and restore colony state
# Args: checkpoint_id (e.g., "5" or "latest")
# Returns: 0 on success, 1 on failure
load_checkpoint() {
    local checkpoint_id="${1:-latest}"

    # Determine checkpoint file path
    local checkpoint_file
    if [ "$checkpoint_id" = "latest" ]; then
        # Read checkpoint file path from CHECKPOINT_FILE
        if [ ! -f "$CHECKPOINT_FILE" ]; then
            echo "No checkpoint reference found: $CHECKPOINT_FILE" >&2
            return 1
        fi
        checkpoint_file=$(cat "$CHECKPOINT_FILE")
    else
        # Construct path from checkpoint ID
        checkpoint_file="${CHECKPOINT_DIR}/checkpoint_${checkpoint_id}.json"
    fi

    # Verify checkpoint exists
    if [ ! -f "$checkpoint_file" ]; then
        echo "Checkpoint not found: $checkpoint_file" >&2
        return 1
    fi

    # Verify checkpoint integrity
    if ! python3 -c "import json; json.load(open('$checkpoint_file'))" 2>/dev/null; then
        echo "Checkpoint corrupted: $checkpoint_file" >&2
        return 1
    fi

    # Restore COLONY_STATE.json
    echo "Restoring COLONY_STATE.json..."
    local temp_state="/tmp/restore_state.$$.tmp"
    if ! jq '.colony_state' "$checkpoint_file" > "$temp_state"; then
        echo "Failed to extract colony_state from checkpoint" >&2
        rm -f "$temp_state"
        return 1
    fi

    if ! atomic_write_from_file "$COLONY_STATE" "$temp_state"; then
        echo "Failed to restore COLONY_STATE.json" >&2
        rm -f "$temp_state"
        return 1
    fi

    rm -f "$temp_state"

    # Restore pheromones.json
    echo "Restoring pheromones.json..."
    local temp_pheromones="/tmp/restore_pheromones.$$.tmp"
    if ! jq '.pheromones' "$checkpoint_file" > "$temp_pheromones"; then
        echo "Failed to extract pheromones from checkpoint" >&2
        rm -f "$temp_pheromones"
        return 1
    fi

    if ! atomic_write_from_file "$PHEROMONES_FILE" "$temp_pheromones"; then
        echo "Failed to restore pheromones.json" >&2
        rm -f "$temp_pheromones"
        return 1
    fi

    rm -f "$temp_pheromones"

    # Restore worker_ants.json
    echo "Restoring worker_ants.json..."
    local temp_workers="/tmp/restore_workers.$$.tmp"
    if ! jq '.worker_ants' "$checkpoint_file" > "$temp_workers"; then
        echo "Failed to extract worker_ants from checkpoint" >&2
        rm -f "$temp_workers"
        return 1
    fi

    if ! atomic_write_from_file "$WORKER_ANTS_FILE" "$temp_workers"; then
        echo "Failed to restore worker_ants.json" >&2
        rm -f "$temp_workers"
        return 1
    fi

    rm -f "$temp_workers"

    # Restore memory.json
    echo "Restoring memory.json..."
    local temp_memory="/tmp/restore_memory.$$.tmp"
    if ! jq '.memory' "$checkpoint_file" > "$temp_memory"; then
        echo "Failed to extract memory from checkpoint" >&2
        rm -f "$temp_memory"
        return 1
    fi

    if ! atomic_write_from_file "$MEMORY_FILE" "$temp_memory"; then
        echo "Failed to restore memory.json" >&2
        rm -f "$temp_memory"
        return 1
    fi

    rm -f "$temp_memory"

    # Display recovery summary
    local restored_state=$(jq -r '.colony_status.state' "$COLONY_STATE")
    local restored_phase=$(jq -r '.colony_status.current_phase' "$COLONY_STATE")

    echo ""
    echo "Colony restored from checkpoint: $checkpoint_file"
    echo "Restored state: $restored_state"
    echo "Restored phase: $restored_phase"
    echo ""
    echo "Next steps:"
    echo "  - Review state: /ant:status"
    echo "  - Continue phase: /ant:execute $restored_phase"

    return 0
}

# Rotate old checkpoints, keeping only 10 most recent
rotate_checkpoints() {
    ls -t "$CHECKPOINT_DIR"/checkpoint_*.json 2>/dev/null | \
        tail -n +11 | \
        xargs rm -f 2>/dev/null || true
}

# List all available checkpoints
list_checkpoints() {
    echo "Available checkpoints:"
    echo ""

    if [ ! -d "$CHECKPOINT_DIR" ]; then
        echo "  No checkpoints directory found"
        return 0
    fi

    local checkpoints=$(ls -1 "$CHECKPOINT_DIR"/checkpoint_*.json 2>/dev/null | wc -l)

    if [ "$checkpoints" -eq 0 ]; then
        echo "  No checkpoints found"
        return 0
    fi

    ls -lh "$CHECKPOINT_DIR"/checkpoint_*.json 2>/dev/null | \
        awk '{print $9, $5, $6, $7, $8}' | \
        while read -r file size date time; do
            local id=$(basename "$file" .json | sed 's/checkpoint_//')
            local label=$(jq -r '.label' "$file" 2>/dev/null || echo "unknown")
            echo "  [$id] $size $date $time - $label"
        done
}

# Export functions for use in other scripts
export -f save_checkpoint load_checkpoint rotate_checkpoints list_checkpoints
