#!/bin/bash
# Worktree utility functions -- extracted from aether-utils.sh
# Provides: _worktree_create, _worktree_cleanup
#
# These functions are sourced by aether-utils.sh at startup.
# All shared infrastructure (json_ok, json_err, atomic_write, acquire_lock,
# release_lock, DATA_DIR, COLONY_DATA_DIR, SCRIPT_DIR, AETHER_ROOT, error
# constants) is available.

# Default worktree location relative to AETHER_ROOT
WORKTREE_BASE_DIR="${AETHER_ROOT}/.aether/worktrees"

# _worktree_create
# Creates a git worktree for an agent working on a specific task.
#
# Usage: _worktree_create --branch <branch-name> [--base <base-branch>] [--task-id <task-id>]
# Returns JSON: {ok:true, result:{path, branch, base, worktree_dir, task_id}}
_worktree_create() {
    local branch=""
    local base=""
    local task_id=""

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --branch) branch="${2:-}"; shift 2 ;;
            --base)   base="${2:-}";   shift 2 ;;
            --task-id) task_id="${2:-}"; shift 2 ;;
            *) shift ;;
        esac
    done

    # Validate required arguments
    if [[ -z "$branch" ]]; then
        json_err "$E_VALIDATION_FAILED" "Usage: worktree-create --branch <branch-name> [--base <base-branch>] [--task-id <task-id>]"
    fi

    # Sanitize branch name: reject obviously dangerous patterns
    if [[ "$branch" == *..* ]] || [[ "$branch" == */* ]] || [[ "$branch" == *\\* ]]; then
        json_err "$E_VALIDATION_FAILED" "Branch name must not contain '..', '/', or backslashes"
    fi

    # Default base to current branch
    if [[ -z "$base" ]]; then
        base=$(git -C "$AETHER_ROOT" rev-parse --abbrev-ref HEAD 2>/dev/null || echo "main")
    fi

    local worktree_dir="$WORKTREE_BASE_DIR/$branch"

    # Check if worktree already exists
    if [[ -d "$worktree_dir" ]]; then
        json_err "$E_VALIDATION_FAILED" "Worktree already exists for branch '$branch' at $worktree_dir"
    fi

    # Check if branch already exists as a git branch (would indicate duplicate)
    if git -C "$AETHER_ROOT" show-ref --verify --quiet "refs/heads/$branch" 2>/dev/null; then
        json_err "$E_VALIDATION_FAILED" "Branch '$branch' already exists"
    fi

    # Ensure base branch exists
    if ! git -C "$AETHER_ROOT" show-ref --verify --quiet "refs/heads/$base" 2>/dev/null; then
        json_err "$E_GIT_ERROR" "Base branch '$base' does not exist"
    fi

    # Ensure parent directory exists
    mkdir -p "$WORKTREE_BASE_DIR"

    # Create the worktree (git worktree add creates the branch automatically)
    if ! git -C "$AETHER_ROOT" worktree add "$worktree_dir" -b "$branch" "$base" >/dev/null 2>&1; then
        json_err "$E_GIT_ERROR" "Failed to create worktree for branch '$branch'"
    fi

    # Copy .aether/data/ structure to the new worktree so the agent has colony context
    # Per state-contract-design.md, branch-local state lives in .aether/data/ (gitignored)
    # and each worktree gets its own independent copy for colony context isolation.
    if [[ -d "$AETHER_ROOT/.aether/data" ]]; then
        mkdir -p "$worktree_dir/.aether/data"
        cp -r "$AETHER_ROOT/.aether/data/." "$worktree_dir/.aether/data/" 2>/dev/null || true  # SUPPRESS:OK -- copy: data dir may be empty
    fi

    # Copy exchange scripts so xml-utils.sh can source pheromone-xml.sh etc.
    if [[ -d "$AETHER_ROOT/.aether/exchange" ]]; then
        mkdir -p "$worktree_dir/.aether/exchange"
        cp -r "$AETHER_ROOT/.aether/exchange/." "$worktree_dir/.aether/exchange/" 2>/dev/null || true
    fi

    # Inject main's pheromone signals into the worktree (per D-01)
    # Non-blocking: if injection fails, worktree still works without injected signals
    if [[ -f "$worktree_dir/.aether/data/pheromones.json" ]]; then
      local main_head
      main_head=$(git -C "$AETHER_ROOT" rev-parse HEAD 2>/dev/null || echo "unknown")
      (
        cd "$worktree_dir" 2>/dev/null && \
        DATA_DIR="$worktree_dir/.aether/data" \
        COLONY_DATA_DIR="$worktree_dir/.aether/data" \
        bash "$AETHER_ROOT/.aether/aether-utils.sh" \
          pheromone-snapshot-inject --from-branch "$base" --from-commit "$main_head" \
        >/dev/null 2>&1 || true
      )
    fi

    # Build result JSON
    local result
    result=$(jq -n \
        --arg path "$worktree_dir" \
        --arg branch "$branch" \
        --arg base "$base" \
        --arg worktree_dir "$worktree_dir" \
        --arg task_id "${task_id:-null}" \
        '{
            path: $path,
            branch: $branch,
            base: $base,
            worktree_dir: $worktree_dir,
            task_id: (if $task_id == "null" then null else $task_id end)
        }')

    json_ok "$result"
}

# _worktree_cleanup
# Removes a git worktree and cleans up tracking.
#
# Usage: _worktree_cleanup --branch <branch-name> [--force]
# Returns JSON: {ok:true, result:{removed, branch, path}}
_worktree_cleanup() {
    local branch=""
    local force=false

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --branch) branch="${2:-}"; shift 2 ;;
            --force)  force=true; shift ;;
            *) shift ;;
        esac
    done

    # Validate required arguments
    if [[ -z "$branch" ]]; then
        json_err "$E_VALIDATION_FAILED" "Usage: worktree-cleanup --branch <branch-name> [--force]"
    fi

    # Sanitize branch name
    if [[ "$branch" == *..* ]] || [[ "$branch" == */* ]] || [[ "$branch" == *\\* ]]; then
        json_err "$E_VALIDATION_FAILED" "Branch name must not contain '..', '/', or backslashes"
    fi

    local worktree_dir="$WORKTREE_BASE_DIR/$branch"

    # Check if worktree exists
    if [[ ! -d "$worktree_dir" ]]; then
        json_err "$E_RESOURCE_NOT_FOUND" "No worktree found for branch '$branch'"
    fi

    # Check for uncommitted changes (unless --force)
    # Exclude .aether/ files since they are branch-local state copies, not user changes
    if [[ "$force" == "false" ]]; then
        local dirty_count
        dirty_count=$(git -C "$worktree_dir" status --porcelain 2>/dev/null \
            | grep -v '\.aether/' \
            | wc -l \
            | tr -d ' ') || dirty_count=0

        if [[ "$dirty_count" -gt 0 ]]; then
            json_err "$E_VALIDATION_FAILED" "Worktree '$branch' has $dirty_count uncommitted changes. Use --force to remove anyway."
        fi
    fi

    # Remove the worktree using git worktree remove
    if ! git -C "$AETHER_ROOT" worktree remove "$worktree_dir" --force 2>/dev/null; then
        # Fallback: manual cleanup if git worktree remove fails
        rm -rf "$worktree_dir" 2>/dev/null || true
        # Also prune stale worktree entries
        git -C "$AETHER_ROOT" worktree prune 2>/dev/null || true
    fi

    # Attempt to delete the branch (best-effort -- may fail if branch is checked out elsewhere)
    git -C "$AETHER_ROOT" branch -D "$branch" >/dev/null 2>&1 || true

    # Build result JSON
    local result
    result=$(jq -n \
        --arg branch "$branch" \
        --arg path "$worktree_dir" \
        '{removed: true, branch: $branch, path: $path}')

    json_ok "$result"
}
