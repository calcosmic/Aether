#!/bin/bash
# Protect sensitive paths from Edit/Write operations
# Returns exit code 1 to block, 0 to allow

# Don't use set -e - we want to fail gracefully (allow by default)

# Read input (may be empty)
INPUT=""
if [ -p /dev/stdin ]; then
    INPUT=$(cat)
fi

# Skip if no input (allow by default)
if [ -z "$INPUT" ]; then
    exit 0
fi

# Extract file path from JSON (safely)
FILE_PATH=$(echo "$INPUT" | grep -o '"file_path"[[:space:]]*:[[:space:]]*"[^"]*"' 2>/dev/null | sed 's/"file_path"[[:space:]]*:[[:space:]]*"\([^"]*\)"/\1/' | head -1) || FILE_PATH=""

if [ -z "$FILE_PATH" ]; then
    exit 0
fi

# Get repo root (silently fail if not in git)
REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null) || REPO_ROOT=""
if [ -z "$REPO_ROOT" ]; then
    exit 0
fi

# Make path relative for matching
REL_PATH="${FILE_PATH#$REPO_ROOT/}"
if [ "$REL_PATH" = "$FILE_PATH" ]; then
    REL_PATH="$FILE_PATH"
fi

# Check protected patterns (allow by default if grep fails)
if echo "$REL_PATH" | grep -qE "^\.aether/data/|^\.aether/checkpoints/|^\.env|^\.claude/settings"; then
    echo "BLOCKED: Protected path: $REL_PATH"
    exit 1
fi

# Allow the edit
exit 0
