#!/bin/bash
# Block destructive bash commands
# Returns exit code 1 to block the command, 0 to allow

# Don't use set -e - we want to fail gracefully (allow by default)

# Read the command from stdin (may be empty)
COMMAND=""
if [ -p /dev/stdin ]; then
    COMMAND=$(cat)
fi

# Skip if no input (allow by default)
if [ -z "$COMMAND" ]; then
    exit 0
fi

# Normalize for matching
COMMAND_LOWER=$(echo "$COMMAND" | tr '[:upper:]' '[:lower:]')

# Blocked patterns (simplified for safety)
if echo "$COMMAND_LOWER" | grep -qE "rm -rf /|rm -rf ~|:\\\(\\\)\\{|mkfs|> /dev/sd"; then
    echo "BLOCKED: Destructive command pattern detected"
    exit 1
fi

# Check for sudo with dangerous commands
if echo "$COMMAND_LOWER" | grep -q "sudo"; then
    if echo "$COMMAND_LOWER" | grep -qE "sudo.*(rm|chmod|chown|dd|mkfs|fdisk)"; then
        echo "BLOCKED: sudo with potentially destructive command"
        exit 1
    fi
fi

# Allow the command
exit 0
