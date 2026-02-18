#!/bin/bash
# Aether File Lock Utility
# Implements file locking for concurrent colony access prevention
#
# Usage:
#   source .aether/utils/file-lock.sh
#   acquire_lock /path/to/file.lock
#   # ... critical section ...
#   release_lock

# Aether root detection - use git root if available, otherwise use current directory
if git rev-parse --show-toplevel >/dev/null 2>&1; then
    AETHER_ROOT="$(git rev-parse --show-toplevel)"
else
    AETHER_ROOT="$(pwd)"
fi

LOCK_DIR="$AETHER_ROOT/.aether/locks"
LOCK_TIMEOUT=300  # 5 minutes max lock time
LOCK_RETRY_INTERVAL=0.5  # Wait 500ms between retries
LOCK_MAX_RETRIES=100  # Total 50 seconds max wait

# Create lock directory if it doesn't exist
mkdir -p "$LOCK_DIR"

# Acquire a file lock using noclobber
# Arguments: file_path (the resource to lock)
# Returns: 0 on success, 1 on failure
# Globals: LOCK_ACQUIRED (set to true when lock acquired), CURRENT_LOCK (set to lock file path)
acquire_lock() {
    local file_path="$1"
    local lock_file="${LOCK_DIR}/$(basename "$file_path").lock"
    local lock_pid_file="${lock_file}.pid"

    # Check if lock file exists and is stale
    if [ -f "$lock_file" ]; then
        local lock_pid
        lock_pid=$(cat "$lock_pid_file" 2>/dev/null || echo "")
        local is_stale=false

        # Check age FIRST (before PID check) to handle PID reuse race
        local lock_mtime=0
        # Platform-portable mtime: macOS uses stat -f %m, Linux uses stat -c %Y
        if stat -f %m "$lock_file" >/dev/null 2>&1; then
            lock_mtime=$(stat -f %m "$lock_file" 2>/dev/null || echo 0)
        else
            lock_mtime=$(stat -c %Y "$lock_file" 2>/dev/null || echo 0)
        fi
        local lock_age=$(( $(date +%s) - lock_mtime ))

        if [[ $lock_age -gt $LOCK_TIMEOUT ]]; then
            is_stale=true
        elif [[ -n "$lock_pid" ]] && ! kill -0 "$lock_pid" 2>/dev/null; then
            is_stale=true
        fi

        if [[ "$is_stale" == "true" ]]; then
            # TTY-gated user prompt — never auto-remove stale locks (locked user decision)
            if [[ -t 2 ]]; then
                echo "" >&2
                echo "Warning: stale lock detected (PID ${lock_pid:-unknown} not running, age ${lock_age}s)" >&2
                echo "Lock file: $lock_file" >&2
                printf "Remove stale lock and continue? [y/N] " >&2
                local response
                read -r response < /dev/tty
                if [[ "$response" =~ ^[Yy]$ ]]; then
                    rm -f "$lock_file" "$lock_pid_file"
                else
                    echo "Lock removal declined. Remove manually: rm $lock_file" >&2
                    return 1
                fi
            else
                # Non-interactive: fail with structured JSON error — do NOT auto-remove
                printf '{"ok":false,"error":{"code":"E_LOCK_STALE","message":"Stale lock found. Remove manually: %s"}}\n' "$lock_file" >&2
                return 1
            fi
        fi
    fi

    # Try to acquire lock with timeout
    local retry_count=0
    while [ $retry_count -lt $LOCK_MAX_RETRIES ]; do
        # Try to create lock file atomically
        if (set -o noclobber; echo $$ > "$lock_file") 2>/dev/null; then
            echo $$ > "$lock_pid_file"
            export LOCK_ACQUIRED=true
            export CURRENT_LOCK="$lock_file"
            return 0
        fi

        retry_count=$((retry_count + 1))
        if [ $retry_count -lt $LOCK_MAX_RETRIES ]; then
            sleep $LOCK_RETRY_INTERVAL
        fi
    done

    echo "Failed to acquire lock for $file_path after $LOCK_MAX_RETRIES attempts" >&2
    return 1
}

# Release a file lock
# Arguments: None (uses CURRENT_LOCK global set by acquire_lock)
release_lock() {
    if [ "$LOCK_ACQUIRED" = "true" ] && [ -n "$CURRENT_LOCK" ]; then
        rm -f "$CURRENT_LOCK" "${CURRENT_LOCK}.pid"
        export LOCK_ACQUIRED=false
        export CURRENT_LOCK=""
        return 0
    fi
    return 1
}

# Cleanup function for script exit
cleanup_locks() {
    if [ "$LOCK_ACQUIRED" = "true" ]; then
        release_lock
    fi
}

# Register cleanup on exit — includes HUP for SSH disconnect safety
trap cleanup_locks EXIT TERM INT HUP

# Check if a file is currently locked
is_locked() {
    local file_path="$1"
    local lock_file="${LOCK_DIR}/$(basename "$file_path").lock"
    [ -f "$lock_file" ]
}

# Get PID of process holding lock
get_lock_holder() {
    local file_path="$1"
    local lock_file="${LOCK_DIR}/$(basename "$file_path").lock.pid"
    cat "$lock_file" 2>/dev/null || echo ""
}

# Wait for lock to be released
wait_for_lock() {
    local file_path="$1"
    local max_wait=${2:-$LOCK_TIMEOUT}
    local waited=0

    while is_locked "$file_path" && [ $waited -lt $max_wait ]; do
        sleep 1
        waited=$((waited + 1))
    done

    if [ $waited -ge $max_wait ]; then
        return 1
    fi
    return 0
}

# Export functions for use in other scripts
export -f acquire_lock release_lock is_locked get_lock_holder wait_for_lock cleanup_locks
