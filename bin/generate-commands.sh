#!/bin/bash
# generate-commands.sh - Sync commands between Claude Code and OpenCode
#
# This script helps keep commands in sync between the two platforms.
# Currently it provides a simple diff-based sync, with plans for full
# YAML-based generation in the future.
#
# Usage:
#   ./bin/generate-commands.sh [check|sync|diff]
#
# Commands:
#   check  - Check if commands are in sync (exit 1 if not)
#   sync   - Copy Claude Code commands to OpenCode (with tool name translation)
#   diff   - Show differences between command sets

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

CLAUDE_DIR="$PROJECT_DIR/.claude/commands/ant"
OPENCODE_DIR="$PROJECT_DIR/.opencode/commands/ant"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Count commands in each directory
count_commands() {
    local dir="$1"
    if [[ -d "$dir" ]]; then
        find "$dir" -name "*.md" | wc -l | tr -d ' '
    else
        echo "0"
    fi
}

# List command files
list_commands() {
    local dir="$1"
    if [[ -d "$dir" ]]; then
        find "$dir" -name "*.md" -exec basename {} \; | sort
    fi
}

# Check if directories are in sync (by file count and names)
check_sync() {
    log_info "Checking command sync status..."

    local claude_count=$(count_commands "$CLAUDE_DIR")
    local opencode_count=$(count_commands "$OPENCODE_DIR")

    echo "Claude Code commands: $claude_count"
    echo "OpenCode commands:    $opencode_count"

    if [[ "$claude_count" != "$opencode_count" ]]; then
        log_error "Command counts don't match!"
        return 1
    fi

    # Check file names match
    local claude_files=$(list_commands "$CLAUDE_DIR")
    local opencode_files=$(list_commands "$OPENCODE_DIR")

    if [[ "$claude_files" != "$opencode_files" ]]; then
        log_error "Command file names don't match!"
        echo ""
        echo "Only in Claude Code:"
        comm -23 <(echo "$claude_files") <(echo "$opencode_files") | sed 's/^/  /'
        echo ""
        echo "Only in OpenCode:"
        comm -13 <(echo "$claude_files") <(echo "$opencode_files") | sed 's/^/  /'
        return 1
    fi

    log_info "Commands are in sync ($claude_count commands)"
    return 0
}

# Check content-level sync using checksums (Pass 2)
# Compares each matching file pair by SHA-1 hash and reports diffs
check_content() {
    log_info "Checking content-level sync (checksums)..."

    local claude_files=$(list_commands "$CLAUDE_DIR")
    local drift_count=0
    local drift_files=""

    for file in $claude_files; do
        local claude_file="$CLAUDE_DIR/$file"
        local opencode_file="$OPENCODE_DIR/$file"

        # Skip if OpenCode file doesn't exist (already caught by Pass 1)
        if [[ ! -f "$opencode_file" ]]; then
            continue
        fi

        # Compare SHA-1 checksums (portable across macOS and Linux)
        local claude_hash opencode_hash
        claude_hash=$(shasum "$claude_file" | cut -d' ' -f1)
        opencode_hash=$(shasum "$opencode_file" | cut -d' ' -f1)

        if [[ "$claude_hash" != "$opencode_hash" ]]; then
            drift_count=$((drift_count + 1))
            drift_files="${drift_files}  ${file}\n"

            log_warn "Content drift: $file"
            echo "  Claude:  $claude_hash"
            echo "  OpenCode: $opencode_hash"

            # Show first 10 lines of diff for context (|| true to handle set -e)
            echo "  ---"
            diff -u "$claude_file" "$opencode_file" | head -20 || true
            echo "  ---"
            echo ""
        fi
    done

    if [[ "$drift_count" -gt 0 ]]; then
        echo ""
        log_error "Content drift detected in $drift_count file(s):"
        echo -e "$drift_files"
        return 1
    fi

    log_info "All file contents match (checksums verified)"
    return 0
}

# Show diff between command sets
show_diff() {
    log_info "Comparing command sets..."

    local claude_files=$(list_commands "$CLAUDE_DIR")

    for file in $claude_files; do
        local claude_file="$CLAUDE_DIR/$file"
        local opencode_file="$OPENCODE_DIR/$file"

        if [[ ! -f "$opencode_file" ]]; then
            log_warn "$file exists only in Claude Code"
            continue
        fi

        # Compare file sizes as a quick check
        local claude_size=$(wc -l < "$claude_file" | tr -d ' ')
        local opencode_size=$(wc -l < "$opencode_file" | tr -d ' ')

        if [[ "$claude_size" != "$opencode_size" ]]; then
            echo "$file: $claude_size lines (Claude) vs $opencode_size lines (OpenCode)"
        fi
    done
}

# Display help
show_help() {
    echo "Aether Command Sync Tool"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  check   Check if commands are in sync"
    echo "  diff    Show differences between command sets"
    echo "  help    Show this help message"
    echo ""
    echo "Directories:"
    echo "  Claude Code: $CLAUDE_DIR"
    echo "  OpenCode:    $OPENCODE_DIR"
    echo ""
    echo "Note: Commands are maintained manually in both directories."
    echo "Use this tool to verify they stay in sync."
}

# Main
case "${1:-check}" in
    check)
        # Pass 1: file count + name check
        check_sync
        # Pass 2: content-level checksum comparison
        check_content
        ;;
    diff)
        show_diff
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        log_error "Unknown command: $1"
        show_help
        exit 1
        ;;
esac
