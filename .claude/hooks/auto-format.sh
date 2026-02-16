#!/bin/bash
# Auto-format files after Edit operations
# Runs prettier/eslint --fix on modified files

set -euo pipefail

# Read the file path from the tool arguments
INPUT=""
while IFS= read -r line; do
    INPUT="$INPUT$line"
done

# Extract file path from JSON
FILE_PATH=$(echo "$INPUT" | grep -o '"file_path"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/"file_path"[[:space:]]*:[[:space:]]*"\([^"]*\)"/\1/' | head -1)

if [ -z "$FILE_PATH" ]; then
    exit 0
fi

# Only format files that exist
if [ ! -f "$FILE_PATH" ]; then
    exit 0
fi

# Get file extension
EXT="${FILE_PATH##*.}"

# Run appropriate formatter based on file type
case "$EXT" in
    js|jsx|ts|tsx|mjs|cjs)
        # Check if prettier is available
        if command -v prettier >/dev/null 2>&1; then
            prettier --write "$FILE_PATH" 2>/dev/null || true
        fi
        # Check if eslint is available
        if command -v eslint >/dev/null 2>&1; then
            eslint --fix "$FILE_PATH" 2>/dev/null || true
        fi
        ;;
    json|md|yaml|yml)
        if command -v prettier >/dev/null 2>&1; then
            prettier --write "$FILE_PATH" 2>/dev/null || true
        fi
        ;;
    sh|bash)
        if command -v shfmt >/dev/null 2>&1; then
            shfmt -w "$FILE_PATH" 2>/dev/null || true
        fi
        ;;
esac

exit 0
