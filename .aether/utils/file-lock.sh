#!/bin/bash
# Aether File Lock Utility
# Implements file locking for concurrent colony access prevention
#
# Usage:
#   source .aether/utils/file-lock.sh
#   acquire_lock /path/to/file.lock
#   # ... critical section ...
#   release_lock

# Aether root detection - respect existing AETHER_ROOT, or use git root, or use current directory
if [[ -z "${AETHER_ROOT:-}" ]]; then
    if git rev-parse --show-toplevel >/dev/null 2>&1; then
        AETHER_ROOT="$(git rev-parse --show-toplevel)"
    else
        AETHER_ROOT="$(pwd)"
    fi
fi

LOCK_DIR="$AETHER_ROOT/.aether/locks"
LOCK_TIMEOUT=300  # 5 minutes max lock time
LOCK_RETRY_INTERVAL=0.5  # Wait 500ms between retries
LOCK_MAX_RETRIES=100  # Total 50 seconds max wait
LOCK_AT_FILE=""  # Tracks lock file path for acquire_lock_at/release_lock_at

# Fallback constant — ensures E_LOCK_STALE is defined whether or not error-handler.sh was loaded
: "${E_LOCK_STALE:=E_LOCK_STALE}"

# Create lock directory if it doesn't exist
mkdir -p "$LOCK_DIR"

# Acquire a file lock using noclobber
# Arguments: file_path (the resource to lock)
# Returns: 0 on success, 1 on failure
# Globals: LOCK_ACQUIRED (set to true when lock acquired), CURRENT_LOCK (set to lock file path)
# Behavior:
#   - In non-interactive mode, stale locks are auto-cleaned by default.
#   - Override with AETHER_STALE_LOCK_MODE=error|prompt|auto.
acquire_lock() {
    local file_path="$1"
    local lock_file="${LOCK_DIR}/$(basename "$file_path").lock"
    local lock_pid_file="${lock_file}.pid"
    local stale_mode="${AETHER_STALE_LOCK_MODE:-}"

    if [[ -z "$stale_mode" ]]; then
        if [[ -t 2 ]]; then
            stale_mode="prompt"
        else
            stale_mode="auto"
        fi
    fi

    # Check if lock file exists and is stale
    if [ -f "$lock_file" ]; then
        local lock_pid
        lock_pid=$(cat "$lock_pid_file" 2>/dev/null || echo "")
        if [[ -z "$lock_pid" ]]; then
            # Fallback to lock file payload if .pid sidecar is missing/corrupt.
            lock_pid=$(cat "$lock_file" 2>/dev/null || echo "")
        fi
        lock_pid=$(echo "$lock_pid" | tr -d '[:space:]')
        [[ "$lock_pid" =~ ^[0-9]+$ ]] || lock_pid=""

        local is_stale=false

        # Compute lock age for timeout-based stale checks when PID is unavailable.
        local lock_mtime=0
        # Platform-portable mtime: macOS uses stat -f %m, Linux uses stat -c %Y
        if stat -f %m "$lock_file" >/dev/null 2>&1; then
            lock_mtime=$(stat -f %m "$lock_file" 2>/dev/null || echo 0)
        else
            lock_mtime=$(stat -c %Y "$lock_file" 2>/dev/null || echo 0)
        fi
        local lock_age=$(( $(date +%s) - lock_mtime ))

        # Mark stale only when we can do so safely:
        # - PID is known and not running
        # - No PID could be determined and lock exceeded timeout
        if [[ -n "$lock_pid" ]] && ! kill -0 "$lock_pid" 2>/dev/null; then
            is_stale=true
        elif [[ -z "$lock_pid" ]] && [[ $lock_age -gt $LOCK_TIMEOUT ]]; then
            is_stale=true
        fi

        if [[ "$is_stale" == "true" ]]; then
            case "$stale_mode" in
                auto)
                    rm -f "$lock_file" "$lock_pid_file"
                    # Track stale lock cleanup in safety stats (best-effort)
                    type _safety_stats_increment &>/dev/null && _safety_stats_increment "stale_locks_cleaned" 2>/dev/null || true
                    ;;
                prompt)
                    if [[ -t 2 ]]; then
                        echo "" >&2
                        echo "Warning: stale lock detected (PID ${lock_pid:-unknown} not running, age ${lock_age}s)" >&2
                        echo "Lock file: $lock_file" >&2
                        printf "Remove stale lock and continue? [y/N] " >&2
                        local response
                        read -r response < /dev/tty
                        if [[ "$response" =~ ^[Yy]$ ]]; then
                            rm -f "$lock_file" "$lock_pid_file"
                            type _safety_stats_increment &>/dev/null && _safety_stats_increment "stale_locks_cleaned" 2>/dev/null || true
                        else
                            echo "Lock removal declined. Remove manually: rm $lock_file" >&2
                            return 1
                        fi
                    else
                        printf '{"ok":false,"error":{"code":"%s","message":"Stale lock found. Remove manually: %s"}}\n' "$E_LOCK_STALE" "$lock_file" >&2
                        return 1
                    fi
                    ;;
                error|*)
                    printf '{"ok":false,"error":{"code":"%s","message":"Stale lock found. Remove manually: %s"}}\n' "$E_LOCK_STALE" "$lock_file" >&2
                    return 1
                    ;;
            esac
        fi
    fi

    # Try to acquire lock with timeout
    local retry_count=0
    while [ $retry_count -lt $LOCK_MAX_RETRIES ]; do
        # Try to create lock file atomically
        if (set -o noclobber; echo $$ > "$lock_file") 2>/dev/null; then
            echo $$ > "$lock_pid_file" 2>/dev/null || true
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

# Acquire a lock in a specified directory with optional colony tag
# Unlike acquire_lock (which uses the global LOCK_DIR), this function takes an
# explicit lock directory parameter — no global state mutation required.
# Arguments:
#   $1 = file_path (resource to lock)
#   $2 = lock_dir (directory to place lock file in)
#   $3 = colony_tag (optional, included in lock filename for debuggability)
# Returns: 0 on success, 1 on failure
# Sets: LOCK_AT_FILE (path to the acquired lock file for release)
acquire_lock_at() {
    local file_path="$1"
    local lock_dir="$2"
    local colony_tag="${3:-}"

    local lock_name
    if [[ -n "$colony_tag" ]]; then
        lock_name="$(basename "$file_path").${colony_tag}.lock"
    else
        lock_name="$(basename "$file_path").lock"
    fi

    local lock_file="${lock_dir}/${lock_name}"
    local lock_pid_file="${lock_file}.pid"
    mkdir -p "$lock_dir"

    local stale_mode="${AETHER_STALE_LOCK_MODE:-}"
    if [[ -z "$stale_mode" ]]; then
        if [[ -t 2 ]]; then stale_mode="prompt"; else stale_mode="auto"; fi
    fi

    # Stale lock detection (reuses same logic as acquire_lock)
    if [[ -f "$lock_file" ]]; then
        local lock_pid
        lock_pid=$(cat "$lock_pid_file" 2>/dev/null || echo "")
        [[ -z "$lock_pid" ]] && lock_pid=$(cat "$lock_file" 2>/dev/null || echo "")
        lock_pid=$(echo "$lock_pid" | tr -d '[:space:]')
        [[ "$lock_pid" =~ ^[0-9]+$ ]] || lock_pid=""

        local is_stale=false
        local lock_mtime=0
        if stat -f %m "$lock_file" >/dev/null 2>&1; then
            lock_mtime=$(stat -f %m "$lock_file" 2>/dev/null || echo 0)
        else
            lock_mtime=$(stat -c %Y "$lock_file" 2>/dev/null || echo 0)
        fi
        local lock_age=$(( $(date +%s) - lock_mtime ))

        if [[ -n "$lock_pid" ]] && ! kill -0 "$lock_pid" 2>/dev/null; then
            is_stale=true
        elif [[ -z "$lock_pid" ]] && [[ $lock_age -gt $LOCK_TIMEOUT ]]; then
            is_stale=true
        fi

        if [[ "$is_stale" == "true" ]]; then
            case "$stale_mode" in
                auto)
                    rm -f "$lock_file" "$lock_pid_file"
                    type _safety_stats_increment &>/dev/null && _safety_stats_increment "stale_locks_cleaned" 2>/dev/null || true
                    ;;
                prompt)
                    if [[ -t 2 ]]; then
                        echo "" >&2
                        echo "Warning: stale lock detected (PID ${lock_pid:-unknown} not running, age ${lock_age}s)" >&2
                        echo "Lock file: $lock_file" >&2
                        printf "Remove stale lock and continue? [y/N] " >&2
                        local response
                        read -r response < /dev/tty
                        if [[ "$response" =~ ^[Yy]$ ]]; then
                            rm -f "$lock_file" "$lock_pid_file"
                            type _safety_stats_increment &>/dev/null && _safety_stats_increment "stale_locks_cleaned" 2>/dev/null || true
                        else
                            echo "Lock removal declined. Remove manually: rm $lock_file" >&2
                            return 1
                        fi
                    else
                        printf '{"ok":false,"error":{"code":"%s","message":"Stale lock found. Remove manually: %s"}}\n' "$E_LOCK_STALE" "$lock_file" >&2
                        return 1
                    fi
                    ;;
                error|*)
                    printf '{"ok":false,"error":{"code":"%s","message":"Stale lock found. Remove manually: %s"}}\n' "$E_LOCK_STALE" "$lock_file" >&2
                    return 1
                    ;;
            esac
        fi
    fi

    # Retry loop (same as acquire_lock)
    local retry_count=0
    while [[ $retry_count -lt $LOCK_MAX_RETRIES ]]; do
        if (set -o noclobber; echo $$ > "$lock_file") 2>/dev/null; then
            echo $$ > "$lock_pid_file" 2>/dev/null || true
            LOCK_AT_FILE="$lock_file"
            return 0
        fi
        retry_count=$((retry_count + 1))
        [[ $retry_count -lt $LOCK_MAX_RETRIES ]] && sleep $LOCK_RETRY_INTERVAL
    done

    echo "Failed to acquire lock for $file_path in $lock_dir after $LOCK_MAX_RETRIES attempts" >&2
    return 1
}

# Release a lock acquired with acquire_lock_at
# Arguments: $1 = lock_file (the LOCK_AT_FILE value from acquire_lock_at)
release_lock_at() {
    local lock_file="${1:-$LOCK_AT_FILE}"
    if [[ -n "$lock_file" && -f "$lock_file" ]]; then
        rm -f "$lock_file" "${lock_file}.pid"
        [[ "$lock_file" == "$LOCK_AT_FILE" ]] && LOCK_AT_FILE=""
        return 0
    fi
    return 1
}

# Cleanup function for script exit
cleanup_locks() {
    if [ "$LOCK_ACQUIRED" = "true" ]; then
        release_lock
    fi
    if [[ -n "${LOCK_AT_FILE:-}" ]]; then
        release_lock_at "$LOCK_AT_FILE"
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
export -f acquire_lock release_lock acquire_lock_at release_lock_at is_locked get_lock_holder wait_for_lock cleanup_locks
