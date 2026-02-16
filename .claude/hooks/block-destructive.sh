#!/bin/bash
# Block destructive bash commands
# Returns exit code 1 to block the command, 0 to allow

set -euo pipefail

# Read the command from stdin (passed by Claude Code)
COMMAND=""
while IFS= read -r line; do
    COMMAND="$COMMAND $line"
done

# Normalize for matching
COMMAND_LOWER=$(echo "$COMMAND" | tr '[:upper:]' '[:lower:]')

# Blocked patterns
BLOCKED_PATTERNS=(
    "rm -rf /"
    "rm -rf /*"
    "rm -rf ~"
    "rm -rf \$home"
    "rm -rf \$home/"
    ":(){ :|:& };:"  # Fork bomb
    "dd if="
    "mkfs"
    "> /dev/sd"
    "chmod -r 777"
    "chown -r"
    "curl.*|.*bash"
    "wget.*|.*bash"
    "curl.*|.*sh"
    "wget.*|.*sh"
)

# Check each pattern
for pattern in "${BLOCKED_PATTERNS[@]}"; do
    if echo "$COMMAND_LOWER" | grep -qE "$pattern"; then
        echo "BLOCKED: Destructive command pattern detected: $pattern"
        echo "Command: $COMMAND"
        echo ""
        echo "If you really need to run this command, please ask the user to approve it directly."
        exit 1
    fi
done

# Check for sudo with dangerous commands
if echo "$COMMAND_LOWER" | grep -q "sudo"; then
    DANGEROUS_SUDO=("rm" "chmod" "chown" "dd" "mkfs" "fdisk" "parted")
    for danger in "${DANGEROUS_SUDO[@]}"; do
        if echo "$COMMAND_LOWER" | grep -q "sudo.*$danger"; then
            echo "BLOCKED: sudo with potentially destructive command: $danger"
            echo "Command: $COMMAND"
            exit 1
        fi
    done
fi

# Allow the command
exit 0
