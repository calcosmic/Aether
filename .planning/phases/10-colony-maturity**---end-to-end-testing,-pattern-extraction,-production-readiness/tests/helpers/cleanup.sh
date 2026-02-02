#!/bin/bash
# Colony Cleanup Helper for Tests
# Provides state cleanup functionality for test isolation
#
# Usage:
#   source tests/helpers/cleanup.sh
#   cleanup_test_colony

# Get git root directory for path resolution
get_git_root() {
    git rev-parse --show-toplevel 2>/dev/null || echo "$PWD"
}

# Colony data paths
GIT_ROOT=$(get_git_root)
DATA_DIR="${GIT_ROOT}/.aether/data"
BACKUPS_DIR="${GIT_ROOT}/.aether/backups"
CHECKPOINTS_DIR="${GIT_ROOT}/.aether/data/checkpoints"

# Cleanup test colony state
# Returns: 0 on success, 1 on failure
cleanup_test_colony() {
    echo "# Cleaning up test colony state..."

    # Clean data files using git clean
    if [ -d "$DATA_DIR" ]; then
        echo "# Cleaning data directory: $DATA_DIR"

        # Remove all JSON files from data directory
        find "$DATA_DIR" -name "*.json" -type f -delete 2>/dev/null || true

        # Remove any checkpoint files
        find "$DATA_DIR/checkpoints" -type f -delete 2>/dev/null || true

        echo "# Data files cleaned"
    else
        echo "# Data directory does not exist: $DATA_DIR (nothing to clean)"
    fi

    # Clean backups directory
    if [ -d "$BACKUPS_DIR" ]; then
        echo "# Cleaning backups directory: $BACKUPS_DIR"

        # Remove all backup files
        find "$BACKUPS_DIR" -type f -delete 2>/dev/null || true

        echo "# Backups cleaned"
    else
        echo "# Backups directory does not exist: $BACKUPS_DIR (nothing to clean)"
    fi

    # Clean checkpoints directory
    if [ -d "$CHECKPOINTS_DIR" ]; then
        echo "# Cleaning checkpoints directory: $CHECKPOINTS_DIR"

        # Remove all checkpoint files
        find "$CHECKPOINTS_DIR" -type f -delete 2>/dev/null || true

        echo "# Checkpoints cleaned"
    else
        echo "# Checkpoints directory does not exist (nothing to clean)"
    fi

    # Verify clean slate
    local remaining_files=$(find "$DATA_DIR" -name "*.json" -type f 2>/dev/null | wc -l | tr -d ' ')

    if [ "$remaining_files" -eq 0 ]; then
        echo "# Cleanup verified: .aether/data is clean"
    else
        echo "# Warning: $remaining_files file(s) remain in .aether/data" >&2
        find "$DATA_DIR" -name "*.json" -type f 2>/dev/null | while read -r file; do
            echo "#   Remaining: $file"
        done
    fi

    echo "# Test colony cleanup complete"

    return 0
}

# Force cleanup using git clean
# More aggressive cleanup, removes all untracked files in .aether
# Returns: 0 on success, 1 on failure
force_cleanup_test_colony() {
    echo "# Force cleaning test colony state using git clean..."

    # Change to git root to run git clean
    local original_dir="$PWD"
    cd "$GIT_ROOT" || {
        echo "# Error: Failed to cd to git root: $GIT_ROOT" >&2
        return 1
    }

    # Clean untracked files in .aether/data
    git clean -fd .aether/data 2>/dev/null || {
        echo "# Warning: git clean encountered issues, continuing..." >&2
    }

    # Clean backups
    git clean -fd .aether/backups 2>/dev/null || true

    # Clean checkpoints
    git clean -fd .aether/data/checkpoints 2>/dev/null || true

    # Return to original directory
    cd "$original_dir" || true

    echo "# Force cleanup complete"

    return 0
}

# Verify clean slate
# Returns: 0 if clean (no JSON files), 1 if files remain
verify_clean_slate() {
    if [ ! -d "$DATA_DIR" ]; then
        echo "# Data directory does not exist (clean)"
        return 0
    fi

    local remaining_files=$(find "$DATA_DIR" -name "*.json" -type f 2>/dev/null | wc -l | tr -d ' ')

    if [ "$remaining_files" -eq 0 ]; then
        echo "# Verified: Clean slate (0 JSON files in .aether/data)"
        return 0
    else
        echo "# Not clean: $remaining_files JSON file(s) remain in .aether/data" >&2
        find "$DATA_DIR" -name "*.json" -type f 2>/dev/null | while read -r file; do
            echo "#   Found: $file"
        done
        return 1
    fi
}

# Export functions for use in tests
export -f get_git_root cleanup_test_colony force_cleanup_test_colony verify_clean_slate
export GIT_ROOT DATA_DIR BACKUPS_DIR CHECKPOINTS_DIR
