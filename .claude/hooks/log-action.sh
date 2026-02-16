#!/bin/bash
# Log all tool actions to ledger for audit trail
# Runs after every tool use (PostToolUse)

set -euo pipefail

# Read input from Claude Code
INPUT=""
while IFS= read -r line; do
    INPUT="$INPUT$line"
done

# Get repo root
REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
LEDGER="$REPO_ROOT/.aether/ledger.jsonl"

# Create ledger directory if needed
mkdir -p "$(dirname "$LEDGER")"

# Extract tool name and create log entry
TOOL_NAME=$(echo "$INPUT" | grep -o '"tool"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/"tool"[[:space:]]*:[[:space:]]*"\([^"]*\)"/\1/' | head -1)
if [ -z "$TOOL_NAME" ]; then
    TOOL_NAME="unknown"
fi

# Create JSON log entry
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
SESSION_ID="${CLAUDE_SESSION_ID:-unknown}"

# Simple log entry
LOG_ENTRY="{\"timestamp\":\"$TIMESTAMP\",\"session\":\"$SESSION_ID\",\"tool\":\"$TOOL_NAME\"}"

# Append to ledger
echo "$LOG_ENTRY" >> "$LEDGER"

# Rotate ledger if too large (keep last 10000 entries)
if [ -f "$LEDGER" ]; then
    LINES=$(wc -l < "$LEDGER")
    if [ "$LINES" -gt 10000 ]; then
        tail -10000 "$LEDGER" > "$LEDGER.tmp"
        mv "$LEDGER.tmp" "$LEDGER"
    fi
fi

exit 0
