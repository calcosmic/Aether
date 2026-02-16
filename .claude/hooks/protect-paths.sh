#!/bin/bash
# Protect sensitive paths from Edit/Write operations
# Returns exit code 1 to block, 0 to allow

set -euo pipefail

# Read the file path from the tool arguments
# The hook receives JSON input from Claude Code
INPUT=""
while IFS= read -r line; do
    INPUT="$INPUT$line"
done

# Extract file path from JSON (simple extraction)
# Expected format: {"file_path": "/path/to/file", ...}
FILE_PATH=$(echo "$INPUT" | grep -o '"file_path"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/"file_path"[[:space:]]*:[[:space:]]*"\([^"]*\)"/\1/' | head -1)

if [ -z "$FILE_PATH" ]; then
    # Could not extract path, allow (fail open for tool compatibility)
    exit 0
fi

# Protected path patterns (relative to repo root)
PROTECTED_PATTERNS=(
    "^\.aether/data/"
    "^\.aether/checkpoints/"
    "^\.aether/locks/"
    "^\.env"
    "^\.env\."
    "^\.claude/settings.*\.json$"
    "^\.github/workflows/"
    "^runtime/"
)

# Get repo root
REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)

# Make path relative for matching
REL_PATH="${FILE_PATH#$REPO_ROOT/}"
if [ "$REL_PATH" = "$FILE_PATH" ]; then
    # Path was already relative
    REL_PATH="$FILE_PATH"
fi

# Check each protected pattern
for pattern in "${PROTECTED_PATTERNS[@]}"; do
    if echo "$REL_PATH" | grep -qE "$pattern"; then
        echo "BLOCKED: Protected path: $REL_PATH"
        echo ""
        echo "This path is protected and cannot be modified by agents."
        echo "If you need to modify this file, please ask the user to do it directly."
        exit 1
    fi
done

# Allow the edit
exit 0
