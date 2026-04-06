#!/bin/bash
# Clash detection utility -- detects file conflicts across git worktrees.
#
# When multiple agents work in parallel worktrees, they could accidentally
# edit the same file from different worktrees. This script checks for that.
#
# Usage: clash-detect --file <path> [--worktree <path>]
# Returns JSON: {ok:true, result:{conflict:false}} or
#               {ok:true, result:{conflict:true, conflicting_worktrees:[...]}}
#
# Environment:
#   AETHER_ROOT  - repo root (auto-detected if not set)
#   WORKTREE_DIR - current worktree path (to exclude self from checks)
#
# When sourced by aether-utils.sh, provides _clash_detect function.
# When run directly, executes _clash_detect with provided arguments.

# Only set strict mode when run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    set -uo pipefail
fi

# Set AETHER_ROOT if not already set (e.g., when sourced)
: "${AETHER_ROOT:=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"

# Fallback error constant for standalone execution (when not sourced by aether-utils.sh)
: "${E_VALIDATION_FAILED:=E_VALIDATION_FAILED}"

# Allowlist of path patterns that bypass clash detection (branch-local state)
# These files live in .aether/data/ and are unique per worktree/branch
_CLASH_ALLOWLIST=(
    ".aether/data/"
)

# Check if a file path matches the allowlist
_clash_is_allowlisted() {
    local file="$1"
    for pattern in "${_CLASH_ALLOWLIST[@]}"; do
        if [[ "$file" == "$pattern"* ]]; then
            return 0
        fi
    done
    return 1
}

# Normalize a path for comparison on macOS (handles /var vs /private/var)
_clash_normalize_path() {
    local p="$1"
    if [[ -z "$p" ]]; then
        echo ""
        return
    fi
    if command -v realpath >/dev/null 2>&1; then
        realpath "$p" 2>/dev/null && return
    fi
    (cd "$p" 2>/dev/null && pwd) || echo "$p"
}

# JSON output helpers (standalone, for direct execution mode)
_clash_json_ok() {
    echo "{\"ok\":true,\"result\":$1}"
}

_clash_json_err() {
    echo "{\"ok\":false,\"error\":$1}" >&2
    exit 1
}

# Bridge: use aether-utils JSON helpers when sourced, standalone when direct
_cok() {
    if type json_ok &>/dev/null; then
        json_ok "$1"
    else
        _clash_json_ok "$1"
    fi
}

_cerr() {
    if type json_err &>/dev/null; then
        json_err "${2:-$E_UNKNOWN}" "$1"
    else
        _clash_json_err "\"$1\""
    fi
}

# Main clash detection logic
_clash_detect() {
    local file=""
    local worktree="${WORKTREE_DIR:-}"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file) file="${2:-}"; shift 2 ;;
            --worktree) worktree="${2:-}"; shift 2 ;;
            *) shift ;;
        esac
    done

    if [[ -z "$file" ]]; then
        _cerr "Usage: clash-detect --file <path> [--worktree <path>]" "$E_VALIDATION_FAILED"
    fi

    # Allowlisted files never clash (branch-local state)
    if _clash_is_allowlisted "$file"; then
        _cok '{"conflict":false}'
        return
    fi

    # Verify we're in a git repo
    if ! git -C "$AETHER_ROOT" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
        _cok '{"conflict":false}'
        return
    fi

    # Normalize the current worktree path for comparison
    if [[ -n "$worktree" ]]; then
        worktree="$(_clash_normalize_path "$worktree")" || worktree=""
    fi

    # Enumerate all worktrees
    local conflicting=()
    while IFS= read -r line; do
        local wt_path
        wt_path=$(echo "$line" | awk '{print $1}')
        [[ -z "$wt_path" ]] && continue

        local abs_wt_path
        abs_wt_path="$(_clash_normalize_path "$wt_path")" || continue

        # Skip if this is the current worktree (self-check)
        if [[ -n "$worktree" && "$abs_wt_path" == "$worktree" ]]; then
            continue
        fi

        # Check if this worktree has uncommitted changes to the target file
        local file_status
        file_status=$(git -C "$abs_wt_path" status --porcelain -- "$file" 2>/dev/null) || continue

        if [[ -n "$file_status" ]]; then
            local branch_name
            branch_name=$(git -C "$abs_wt_path" rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
            conflicting+=("\"$branch_name\"")
        fi
    done < <(git -C "$AETHER_ROOT" worktree list 2>/dev/null | tail -n +2)

    if [[ ${#conflicting[@]} -eq 0 ]]; then
        _cok '{"conflict":false}'
    else
        local conflict_list
        conflict_list=$(printf '%s,' "${conflicting[@]}")
        conflict_list="[${conflict_list%,}]"
        _cok "{\"conflict\":true,\"conflicting_worktrees\":$conflict_list}"
    fi
}

# _clash_setup
# Install or uninstall the clash detection PreToolUse hook.
#
# Usage: clash-setup [--install] [--uninstall]
# Returns JSON: {ok:true, result:{hook_installed:true/false}}
#
# Environment:
#   CLASH_SETTINGS_PATH - path to settings.json (default: .claude/settings.json)
_clash_setup() {
    local action=""
    local settings_path="${CLASH_SETTINGS_PATH:-$AETHER_ROOT/.claude/settings.json}"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --install)   action="install"; shift ;;
            --uninstall) action="uninstall"; shift ;;
            *) shift ;;
        esac
    done

    if [[ -z "$action" ]]; then
        _cerr "Usage: clash-setup [--install] [--uninstall]" "$E_VALIDATION_FAILED"
    fi

    # Ensure settings file exists
    if [[ ! -f "$settings_path" ]]; then
        echo '{}' > "$settings_path" 2>/dev/null || {
            _cerr "Cannot create settings file at $settings_path" "$E_FILE_NOT_FOUND"
        }
    fi

    # Read current settings
    local settings
    settings=$(cat "$settings_path" 2>/dev/null) || settings='{}'

    local hook_command='node .aether/utils/hooks/clash-pre-tool-use.js'
    local hook_entry="{\"type\":\"command\",\"command\":\"$hook_command\",\"timeout\":5}"

    if [[ "$action" == "install" ]]; then
        # Check if hook already exists in PreToolUse
        if echo "$settings" | jq -e '.hooks.PreToolUse[]?.hooks[]?.command' 2>/dev/null \
            | grep -q "clash-pre-tool-use"; then
            # Already installed
            _cok '{"hook_installed":true}'
            return
        fi

        # Add the hook to PreToolUse array
        settings=$(echo "$settings" | jq --arg entry "$hook_entry" '
            if .hooks.PreToolUse then
                .hooks.PreToolUse += [{"matcher": "Edit|Write", "hooks": [($entry | fromjson)]}]
            else
                .hooks.PreToolUse = [{"matcher": "Edit|Write", "hooks": [($entry | fromjson)]}]
            end
        ' 2>/dev/null)

        if [[ $? -ne 0 ]] || [[ -z "$settings" ]]; then
            _cerr "Failed to update settings file" "$E_UNKNOWN"
        fi

        echo "$settings" > "$settings_path"

        _cok '{"hook_installed":true}'

    elif [[ "$action" == "uninstall" ]]; then
        # Remove the hook from PreToolUse
        settings=$(echo "$settings" | jq '
            if .hooks.PreToolUse then
                .hooks.PreToolUse = [.hooks.PreToolUse[] |
                    select(.hooks[]?.command | . != "node .aether/utils/hooks/clash-pre-tool-use.js")
                ]
            end
        ' 2>/dev/null)

        echo "$settings" > "$settings_path"

        _cok '{"hook_installed":false}'
    fi
}

# Run if executed directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    _clash_detect "$@"
fi
