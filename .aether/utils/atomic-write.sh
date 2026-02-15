#!/bin/bash
# Aether Atomic Write Utility
# Implements atomic write pattern (temp file + rename) for corruption safety
#
# Usage:
#   source .aether/utils/atomic-write.sh
#   atomic_write /path/to/file.json "content"
#   atomic_write_from_file /path/to/target.json /path/to/temp.json

# Source required utilities
# Get the directory where this script is located
_AETHER_UTILS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# If BASH_SOURCE[0] is empty (can happen in some contexts), use repo-local path
if [ -z "$_AETHER_UTILS_DIR" ] || [ "$_AETHER_UTILS_DIR" = "$(pwd)" ]; then
    if git rev-parse --show-toplevel >/dev/null 2>&1; then
        _AETHER_UTILS_DIR="$(git rev-parse --show-toplevel)/.aether/utils"
    else
        _AETHER_UTILS_DIR="$PWD/.aether/utils"
    fi
fi
# Verify the path exists and file-lock.sh is there
if [ ! -f "$_AETHER_UTILS_DIR/file-lock.sh" ]; then
    # Try one more fallback - relative to script location
    _AETHER_UTILS_DIR="$(dirname "${BASH_SOURCE[0]}")"
fi
source "$_AETHER_UTILS_DIR/file-lock.sh"

# Aether root detection - use git root if available, otherwise use current directory
if git rev-parse --show-toplevel >/dev/null 2>&1; then
    AETHER_ROOT="$(git rev-parse --show-toplevel)"
else
    AETHER_ROOT="$(pwd)"
fi

TEMP_DIR="$AETHER_ROOT/.aether/temp"
BACKUP_DIR="$AETHER_ROOT/.aether/data/backups"

# Create directories
mkdir -p "$TEMP_DIR" "$BACKUP_DIR"

# Number of backups to keep
MAX_BACKUPS=3

# Atomic write: write content to file via temporary file
# Arguments: target_file, content
# Returns: 0 on success, 1 on failure
atomic_write() {
    local target_file="$1"
    local content="$2"

    # Ensure target directory exists
    local target_dir=$(dirname "$target_file")
    mkdir -p "$target_dir"

    # Create unique temp file
    local temp_file="${TEMP_DIR}/$(basename "$target_file").$$.$(date +%s%N).tmp"

    # Write content to temp file
    if ! echo "$content" > "$temp_file"; then
        echo "Failed to write to temp file: $temp_file"
        rm -f "$temp_file"
        return 1
    fi

    # Create backup if target exists (do this BEFORE validation to avoid race condition)
    if [ -f "$target_file" ]; then
        create_backup "$target_file"
    fi

    # Validate JSON if it's a JSON file
    if [[ "$target_file" == *.json ]]; then
        if ! python3 -c "import json; json.load(open('$temp_file'))" 2>/dev/null; then
            echo "Invalid JSON in temp file: $temp_file"
            rm -f "$temp_file"
            return 1
        fi
    fi

    # Atomic rename (overwrites target if exists)
    if ! mv "$temp_file" "$target_file"; then
        echo "Failed to rename temp file to target: $target_file"
        rm -f "$temp_file"
        return 1
    fi

    # Sync to disk
    if command -v sync >/dev/null 2>&1; then
        sync "$target_file" 2>/dev/null || true
    fi

    return 0
}

# Atomic write from source file to target
# Arguments: target_file, source_file
# Returns: 0 on success, 1 on failure
atomic_write_from_file() {
    local target_file="$1"
    local source_file="$2"

    if [ ! -f "$source_file" ]; then
        echo "Source file does not exist: $source_file"
        return 1
    fi

    # Ensure target directory exists
    local target_dir=$(dirname "$target_file")
    mkdir -p "$target_dir"

    # Create unique temp file
    local temp_file="${TEMP_DIR}/$(basename "$target_file").$$.$(date +%s%N).tmp"

    # Copy source to temp
    if ! cp "$source_file" "$temp_file"; then
        echo "Failed to copy source to temp: $source_file -> $temp_file"
        rm -f "$temp_file"
        return 1
    fi

    # Validate JSON if it's a JSON file
    if [[ "$target_file" == *.json ]]; then
        if ! python3 -c "import json; json.load(open('$temp_file'))" 2>/dev/null; then
            echo "Invalid JSON in temp file: $temp_file"
            rm -f "$temp_file"
            return 1
        fi
    fi

    # Create backup if target exists
    if [ -f "$target_file" ]; then
        create_backup "$target_file"
    fi

    # Atomic rename
    if ! mv "$temp_file" "$target_file"; then
        echo "Failed to rename temp file to target: $target_file"
        rm -f "$temp_file"
        return 1
    fi

    # Sync to disk
    if command -v sync >/dev/null 2>&1; then
        sync "$target_file" 2>/dev/null || true
    fi

    return 0
}

# Create backup of file
# Arguments: file_path
create_backup() {
    local file_path="$1"
    local base_name=$(basename "$file_path")
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="${BACKUP_DIR}/${base_name}.${timestamp}.backup"

    cp "$file_path" "$backup_file" 2>/dev/null || return 1

    # Rotate old backups
    rotate_backups "$base_name"

    return 0
}

# Rotate backups, keeping only MAX_BACKUPS
# Arguments: base_name
rotate_backups() {
    local base_name="$1"
    local backups=$(ls -t "${BACKUP_DIR}/${base_name}".*.backup 2>/dev/null | wc -l)

    if [ "$backups" -gt "$MAX_BACKUPS" ]; then
        ls -t "${BACKUP_DIR}/${base_name}".*.backup | tail -n +$((MAX_BACKUPS + 1)) | xargs rm -f
    fi
}

# Restore from backup
# Arguments: target_file, [backup_number]
# Returns: 0 on success, 1 on failure
restore_backup() {
    local target_file="$1"
    local backup_num="${2:-1}"  # Default to most recent backup
    local base_name=$(basename "$target_file")

    local backup_file=$(ls -t "${BACKUP_DIR}/${base_name}".*.backup 2>/dev/null | sed -n "${backup_num}p")

    if [ -z "$backup_file" ] || [ ! -f "$backup_file" ]; then
        echo "No backup found for: $target_file"
        return 1
    fi

    if ! atomic_write_from_file "$target_file" "$backup_file"; then
        echo "Failed to restore from backup: $backup_file"
        return 1
    fi

    echo "Restored from: $backup_file"
    return 0
}

# List available backups
# Arguments: file_path
list_backups() {
    local file_path="$1"
    local base_name=$(basename "$file_path")

    echo "Available backups for $base_name:"
    ls -lh "${BACKUP_DIR}/${base_name}".*.backup 2>/dev/null || echo "No backups found"
}

# Cleanup temp files older than 1 hour
cleanup_temp_files() {
    find "$TEMP_DIR" -name "*.tmp" -mtime +1/24 -delete 2>/dev/null || true
}

# Export functions
export -f atomic_write atomic_write_from_file create_backup rotate_backups
export -f restore_backup list_backups cleanup_temp_files
