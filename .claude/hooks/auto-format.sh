#!/bin/bash
# Auto-format files after Edit operations
# Runs prettier/eslint --fix on modified files

# Don't use set -e - we want to fail gracefully

# Read input from Claude Code (may be empty)
INPUT=""
if [ -p /dev/stdin ]; then
    INPUT=$(cat)
fi

# Skip if no input
if [ -z "$INPUT" ]; then
    exit 0
fi

# Extract file path from JSON (safely)
FILE_PATH=$(echo "$INPUT" | grep -o '"file_path"[[:space:]]*:[[:space:]]*"[^"]*"' 2>/dev/null | sed 's/"file_path"[[:space:]]*:[[:space:]]*"\([^"]*\)"/\1/' | head -1) || FILE_PATH=""

if [ -z "$FILE_PATH" ]; then
    exit 0
fi

# Only format files that exist
if [ ! -f "$FILE_PATH" ]; then
    exit 0
fi

# Get file extension
EXT="${FILE_PATH##*.}"

# Run appropriate formatter based on file type (silently)
case "$EXT" in
    js|jsx|ts|tsx|mjs|cjs)
        command -v prettier >/dev/null 2>&1 && prettier --write "$FILE_PATH" 2>/dev/null || true
        command -v eslint >/dev/null 2>&1 && eslint --fix "$FILE_PATH" 2>/dev/null || true
        ;;
    json|md|yaml|yml)
        command -v prettier >/dev/null 2>&1 && prettier --write "$FILE_PATH" 2>/dev/null || true
        ;;
    sh|bash)
        command -v shfmt >/dev/null 2>&1 && shfmt -w "$FILE_PATH" 2>/dev/null || true
        ;;
esac

exit 0
