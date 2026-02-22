#!/bin/bash
# generate-commands.sh - Sync checks for commands and agents
#
# This script helps keep command/agent surfaces in sync between platforms.
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
CLAUDE_AGENT_DIR="$PROJECT_DIR/.claude/agents/ant"
OPENCODE_AGENT_DIR="$PROJECT_DIR/.opencode/agents"
AETHER_AGENT_MIRROR_DIR="$PROJECT_DIR/.aether/agents-claude"

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

# Compute SHA hash with error handling
# Returns 0 on success, 1 on failure
# Echoes hash on success, error message on failure
compute_hash() {
    local file="$1"

    if [[ ! -r "$file" ]]; then
        echo "NOT_READABLE"
        return 1
    fi

    local hash
    hash=$(shasum "$file" 2>/dev/null | cut -d' ' -f1)
    if [[ -z "$hash" ]]; then
        echo "HASH_FAILED"
        return 1
    fi

    echo "$hash"
    return 0
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

# List command files (PLAN-006 fix #13 - warn about non-.md files)
list_commands() {
    local dir="$1"
    if [[ -d "$dir" ]]; then
        # Check for non-.md files and warn
        local non_md_count
        non_md_count=$(find "$dir" -type f ! -name "*.md" 2>/dev/null | wc -l | tr -d ' ')
        if [[ "$non_md_count" -gt 0 ]]; then
            log_warn "$non_md_count non-.md file(s) found in $dir (ignored)"
        fi

        find "$dir" -name "*.md" -exec basename {} \; | sort
    fi
}

# List agent definition files (*.md) by basename
list_agents() {
    local dir="$1"
    if [[ -d "$dir" ]]; then
        find "$dir" -name "*.md" -type f -exec basename {} \; | sort
    fi
}

# Check if directories are in sync (by file count and names)
check_sync() {
    log_info "Checking command sync status..."

    local claude_count=$(count_commands "$CLAUDE_DIR")
    local opencode_count=$(count_commands "$OPENCODE_DIR")

    echo "Claude Code commands: $claude_count"
    echo "OpenCode commands:    $opencode_count"

    # PLAN-006 fix #10 - warn about empty directories
    if [[ "$claude_count" -eq 0 ]] && [[ "$opencode_count" -eq 0 ]]; then
        log_warn "Both command directories are empty"
        echo "This may indicate a misconfiguration"
    fi

    # PLAN-006 fix #11 - warn about large command counts
    local max_commands=500
    if [[ "$claude_count" -gt "$max_commands" ]] || [[ "$opencode_count" -gt "$max_commands" ]]; then
        log_warn "Large number of commands ($claude_count/$opencode_count)"
        echo "This may cause performance issues during sync checks"
    fi

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

    local drift_count=0
    local error_count=0
    local match_count=0
    local drift_files=""
    local error_files=""

    # Use null delimiter for safe iteration (handles filenames with spaces)
    while IFS= read -r -d '' claude_file; do
        local file
        file=$(basename "$claude_file")
        local opencode_file="$OPENCODE_DIR/$file"

        # Skip if OpenCode file doesn't exist (already caught by Pass 1)
        if [[ ! -f "$opencode_file" ]]; then
            continue
        fi

        # Compute hashes with error handling
        local claude_hash opencode_hash
        claude_hash=$(compute_hash "$claude_file")
        if [[ $? -ne 0 ]]; then
            log_error "Cannot hash $claude_file ($claude_hash)"
            error_files="${error_files}  ${file} (${claude_hash})\n"
            error_count=$((error_count + 1))
            continue
        fi

        opencode_hash=$(compute_hash "$opencode_file")
        if [[ $? -ne 0 ]]; then
            log_error "Cannot hash $opencode_file ($opencode_hash)"
            error_files="${error_files}  ${file} (${opencode_hash})\n"
            error_count=$((error_count + 1))
            continue
        fi

        if [[ "$claude_hash" != "$opencode_hash" ]]; then
            drift_count=$((drift_count + 1))
            drift_files="${drift_files}  ${file}\n"

            log_warn "Content drift: $file"
            echo "  Claude:  $claude_hash"
            echo "  OpenCode: $opencode_hash"

            # PLAN-006 fix #12 - improved diff error handling
            echo "  ---"
            local diff_output
            if diff_output=$(diff -u "$claude_file" "$opencode_file" 2>&1); then
                # Files are same (shouldn't happen if hashes differ, but handle it)
                echo "$diff_output" | head -20
            else
                local diff_exit=$?
                if [[ "$diff_output" == *"diff:"* && "$diff_output" == *"No such file"* ]]; then
                    log_error "diff failed: $diff_output"
                else
                    # Normal diff output (exit 1 means files differ)
                    echo "$diff_output" | head -20
                fi
            fi
            echo "  ---"
            echo ""
        else
            match_count=$((match_count + 1))
        fi
    done < <(find "$CLAUDE_DIR" -name "*.md" -type f -print0 2>/dev/null | sort -z)

    # Report results
    if [[ "$error_count" -gt 0 ]]; then
        echo ""
        log_error "Hash errors in $error_count file(s):"
        echo -e "$error_files"
    fi

    if [[ "$drift_count" -gt 0 ]]; then
        echo ""
        log_warn "Content drift detected in $drift_count file(s) (non-blocking):"
        echo -e "$drift_files"
        # Content drift is advisory — structural sync is what matters
        log_info "Content checksum comparison completed ($match_count matched, $drift_count drifted)"
    else
        log_info "All file contents match (checksums verified: $match_count files)"
    fi

    if [[ "$error_count" -gt 0 ]]; then
        return 1
    fi

    return 0
}

# Check agent sync strategy:
# 1) Claude <-> OpenCode: structural parity (count + file names)
# 2) Claude <-> .aether mirror: exact parity (count + names + content hash)
check_agent_sync() {
    log_info "Checking agent sync status..."

    local claude_count
    local opencode_count
    local mirror_count
    claude_count=$(count_commands "$CLAUDE_AGENT_DIR")
    opencode_count=$(count_commands "$OPENCODE_AGENT_DIR")
    mirror_count=$(count_commands "$AETHER_AGENT_MIRROR_DIR")

    echo "Claude agents:        $claude_count"
    echo "OpenCode agents:      $opencode_count"
    echo "Aether mirror agents: $mirror_count"

    if [[ "$claude_count" != "$opencode_count" ]]; then
        log_error "Claude/OpenCode agent counts don't match!"
        return 1
    fi

    if [[ "$claude_count" != "$mirror_count" ]]; then
        log_error "Claude/.aether mirror agent counts don't match!"
        return 1
    fi

    local claude_files
    local opencode_files
    local mirror_files
    claude_files=$(list_agents "$CLAUDE_AGENT_DIR")
    opencode_files=$(list_agents "$OPENCODE_AGENT_DIR")
    mirror_files=$(list_agents "$AETHER_AGENT_MIRROR_DIR")

    if [[ "$claude_files" != "$opencode_files" ]]; then
        log_error "Claude/OpenCode agent file names don't match!"
        echo ""
        echo "Only in Claude:"
        comm -23 <(echo "$claude_files") <(echo "$opencode_files") | sed 's/^/  /'
        echo ""
        echo "Only in OpenCode:"
        comm -13 <(echo "$claude_files") <(echo "$opencode_files") | sed 's/^/  /'
        return 1
    fi

    if [[ "$claude_files" != "$mirror_files" ]]; then
        log_error "Claude/.aether mirror agent file names don't match!"
        echo ""
        echo "Only in Claude:"
        comm -23 <(echo "$claude_files") <(echo "$mirror_files") | sed 's/^/  /'
        echo ""
        echo "Only in .aether mirror:"
        comm -13 <(echo "$claude_files") <(echo "$mirror_files") | sed 's/^/  /'
        return 1
    fi

    # Claude and mirror should be byte-identical.
    local file
    while IFS= read -r file; do
        [[ -z "$file" ]] && continue
        local claude_file="$CLAUDE_AGENT_DIR/$file"
        local mirror_file="$AETHER_AGENT_MIRROR_DIR/$file"
        local claude_hash
        local mirror_hash
        claude_hash=$(compute_hash "$claude_file")
        mirror_hash=$(compute_hash "$mirror_file")
        if [[ "$claude_hash" != "$mirror_hash" ]]; then
            log_error "Claude/.aether mirror content drift: $file"
            return 1
        fi
    done <<< "$claude_files"

    log_info "Agents are in sync (Claude/OpenCode structural parity, Claude/.aether mirror exact parity)"
    return 0
}

# Show diff between command sets
show_diff() {
    log_info "Comparing command sets..."

    # Use null delimiter for safe iteration (handles filenames with spaces)
    while IFS= read -r -d '' claude_file; do
        local file
        file=$(basename "$claude_file")
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
    done < <(find "$CLAUDE_DIR" -name "*.md" -type f -print0 2>/dev/null | sort -z)
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
    echo "  Claude agents: $CLAUDE_AGENT_DIR"
    echo "  OpenCode agents: $OPENCODE_AGENT_DIR"
    echo "  Aether mirror agents: $AETHER_AGENT_MIRROR_DIR"
    echo ""
    echo "Note: command/agent specs are maintained manually."
    echo "Use this tool to verify structural and mirror sync constraints."
}

# Main
case "${1:-check}" in
    check)
        # Pass 1: file count + name check
        check_sync
        # Pass 2: content-level checksum comparison
        check_content
        # Pass 3: agent sync policy checks
        check_agent_sync
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
