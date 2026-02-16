#!/bin/bash
# Log all tool actions to ledger for audit trail
# Runs after every tool use (PostToolUse)

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

# Get repo root (silently fail if not in git)
REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null) || REPO_ROOT=""
if [ -z "$REPO_ROOT" ]; then
    exit 0
fi

LEDGER="$REPO_ROOT/.aether/ledger.jsonl"

# Create ledger directory if needed
mkdir -p "$(dirname "$LEDGER")" 2>/dev/null || exit 0

# Extract tool name (safely)
TOOL_NAME=$(echo "$INPUT" | grep -o '"tool"[[:space:]]*:[[:space:]]*"[^"]*"' 2>/dev/null | sed 's/"tool"[[:space:]]*:[[:space:]]*"\([^"]*\)"/\1/' | head -1) || TOOL_NAME=""
if [ -z "$TOOL_NAME" ]; then
    TOOL_NAME="unknown"
fi

# Create JSON log entry
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null) || TIMESTAMP="unknown"
SESSION_ID="${CLAUDE_SESSION_ID:-unknown}"

# Simple log entry
LOG_ENTRY="{\"timestamp\":\"$TIMESTAMP\",\"session\":\"$SESSION_ID\",\"tool\":\"$TOOL_NAME\"}"

# Append to ledger (silently fail if can't write)
echo "$LOG_ENTRY" >> "$LEDGER" 2>/dev/null || true

# Rotate ledger if too large (keep last 10000 entries)
if [ -f "$LEDGER" ]; then
    LINES=$(wc -l < "$LEDGER" 2>/dev/null) || LINES=0
    if [ "$LINES" -gt 10000 ]; then
        tail -10000 "$LEDGER" > "$LEDGER.tmp" 2>/dev/null && mv "$LEDGER.tmp" "$LEDGER" 2>/dev/null || true
    fi
fi

exit 0
