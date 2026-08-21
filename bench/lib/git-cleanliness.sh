#!/usr/bin/env bash
# git-cleanliness.sh — orphan-branch, tool-residue and uncommitted-residue
# checker for the Aether-vs-GSD benchmark harness.
#
# Usage: git-cleanliness.sh REPO_PATH STARTING_BRANCH STARTING_SHA ALLOWLIST_FILE
#
# Emits one JSON object on stdout and nothing else. Diagnostics/failures go
# to stderr and exit non-zero — never a silent zero, because a cleanliness
# score computed from a missing allowlist would report a false-perfect
# result.
#
# Allowlist file format: one glob pattern per line. A pattern ending in
# `/**` matches the directory and everything beneath it (e.g. `bench/**`
# matches `bench/foo` and `bench/foo/bar`); any other pattern is matched via
# bash's plain (extglob-free) `[[ $path == $pattern ]]` globbing, so `*`
# matches within a path segment the way a normal shell glob does.
#
# The allowlist has two blocks, separated by a marker comment line:
#   - everything ABOVE the line `# lane working state — not counted as
#     unnecessary` is the task-content block (paths the task itself is
#     expected to touch)
#   - everything at/below that line is the lane's legitimate working-state
#     block (tool-internal paths a lane creates as normal operation, e.g.
#     `.aether/data/**` for an Aether lane) — patterns in this block are
#     used for BOTH the unnecessary_modifications check and the
#     tool_internal_paths check.
set -euo pipefail

fail() { echo "GIT-CLEANLINESS FAIL: $*" >&2; exit 1; }

# _json_array_from PATH...
#
# Renders a bash array of strings as a JSON array. Written as its own
# function (rather than `printf '%s\n' "${arr[@]}"` inline at each call
# site) because bash 3.2 (macOS default, no namerefs, stricter `set -u`
# unbound-array handling than 4.4+) can trip an unbound-variable error on
# `"${arr[@]}"` when the array is empty; guarding with `$#` here keeps every
# call site safe without disabling `set -u`.
_json_array_from() {
  if [ "$#" -eq 0 ]; then
    echo '[]'
    return
  fi
  printf '%s\n' "$@" | jq -R . | jq -s -c 'map(select(length > 0))'
}

# _path_matches_pattern PATH PATTERN
# Returns 0 (true) if PATH matches PATTERN per the header's matching rule.
_path_matches_pattern() {
  local path="$1"
  local pattern="$2"
  if [[ "$pattern" == */\*\* ]]; then
    local dir="${pattern%/**}"
    [[ "$path" == "$dir" || "$path" == "$dir"/* ]]
    return
  fi
  [[ "$path" == $pattern ]]
}

# _path_matches_any PATH PATTERN...
#
# Bash 3.2 (macOS default, no namerefs) compatible: patterns are passed as
# remaining positional args rather than an array-name reference.
_path_matches_any() {
  local path="$1"
  shift
  local pattern
  for pattern in "$@"; do
    [ -z "$pattern" ] && continue
    if _path_matches_pattern "$path" "$pattern"; then
      return 0
    fi
  done
  return 1
}

# git_cleanliness REPO_PATH STARTING_BRANCH STARTING_SHA ALLOWLIST_FILE
git_cleanliness() {
  local repo_path="$1"
  local starting_branch="$2"
  local starting_sha="$3"
  local allowlist_file="$4"

  [ -n "$repo_path" ] || fail "repo path argument is required"
  [ -n "$starting_branch" ] || fail "starting branch argument is required"
  [ -n "$starting_sha" ] || fail "starting SHA argument is required"
  [ -n "$allowlist_file" ] || fail "allowlist path argument is required"

  [ -f "$allowlist_file" ] || fail "allowlist file does not exist: $allowlist_file — a cleanliness score computed from a missing allowlist would silently report a perfect result"

  (cd "$repo_path" && git rev-parse --is-inside-work-tree >/dev/null 2>&1) \
    || fail "repo path is not a git repository: $repo_path"

  (cd "$repo_path" && git cat-file -e "${starting_sha}^{commit}" 2>/dev/null) \
    || fail "starting SHA is not a valid commit in $repo_path: $starting_sha"

  # Split the allowlist into task-content patterns and lane-working-state
  # patterns, using the marker comment line.
  local -a task_patterns=()
  local -a lane_patterns=()
  local in_lane_block=0
  local line
  while IFS= read -r line || [ -n "$line" ]; do
    if [[ "$line" == "# lane working state"* ]]; then
      in_lane_block=1
      continue
    fi
    [[ "$line" =~ ^[[:space:]]*# ]] && continue
    [[ -z "${line// }" ]] && continue
    if [ "$in_lane_block" -eq 1 ]; then
      lane_patterns+=("$line")
    else
      task_patterns+=("$line")
    fi
  done < "$allowlist_file"

  local -a all_allowed_patterns=("${task_patterns[@]:-}" "${lane_patterns[@]:-}")

  # orphan_branches: local branches other than the starting branch that
  # contain commits not reachable from the starting branch's tip.
  local -a orphan_branch_names=()
  local branch
  while IFS= read -r branch; do
    [ -z "$branch" ] && continue
    [ "$branch" = "$starting_branch" ] && continue
    orphan_branch_names+=("$branch")
  done < <(cd "$repo_path" && git branch --no-merged "$starting_branch" --format='%(refname:short)' 2>/dev/null || true)
  local orphan_branches="${#orphan_branch_names[@]}"

  # uncommitted_residue: entries in `git status --porcelain`.
  local -a uncommitted_residue_paths=()
  local status_line status_path
  while IFS= read -r status_line; do
    [ -z "$status_line" ] && continue
    status_path="${status_line:3}"
    uncommitted_residue_paths+=("$status_path")
  done < <(cd "$repo_path" && git status --porcelain 2>/dev/null || true)
  local uncommitted_residue="${#uncommitted_residue_paths[@]}"

  # Candidate changed paths: diff of starting SHA..HEAD, plus untracked
  # files (also present in the porcelain scan above as "??" entries).
  local -a changed_paths=()
  local diff_path
  while IFS= read -r diff_path; do
    [ -z "$diff_path" ] && continue
    changed_paths+=("$diff_path")
  done < <(cd "$repo_path" && git diff --name-only "${starting_sha}..HEAD" 2>/dev/null || true)
  while IFS= read -r status_line; do
    [ -z "$status_line" ] && continue
    if [[ "$status_line" == '??'* ]]; then
      changed_paths+=("${status_line:3}")
    fi
  done < <(cd "$repo_path" && git status --porcelain 2>/dev/null || true)

  # tool_internal_paths: working-tree paths matching tool-internal patterns
  # that are NOT covered by the lane's legitimate working-state block.
  local -a tool_internal_prefixes=(".aether/" ".planning/" ".claude/" ".codex/" ".opencode/")
  local -a tool_internal_paths=()
  local p prefix is_tool_internal is_lane_allowed
  for p in "${changed_paths[@]:-}"; do
    [ -z "$p" ] && continue
    is_tool_internal=0
    for prefix in "${tool_internal_prefixes[@]:-}"; do
      if [[ "$p" == "$prefix"* ]]; then
        is_tool_internal=1
        break
      fi
    done
    [ "$is_tool_internal" -eq 0 ] && continue
    if _path_matches_any "$p" "${lane_patterns[@]:-}"; then
      is_lane_allowed=1
    else
      is_lane_allowed=0
    fi
    if [ "$is_lane_allowed" -eq 0 ]; then
      tool_internal_paths+=("$p")
    fi
  done

  # unnecessary_modifications: changed paths matching NEITHER the
  # task-content block NOR the lane block at all.
  local -a unnecessary_modification_paths=()
  for p in "${changed_paths[@]:-}"; do
    [ -z "$p" ] && continue
    if ! _path_matches_any "$p" "${all_allowed_patterns[@]:-}"; then
      unnecessary_modification_paths+=("$p")
    fi
  done

  local orphan_count="$orphan_branches"
  local tool_internal_count="${#tool_internal_paths[@]}"
  local unnecessary_count="${#unnecessary_modification_paths[@]}"

  # cleanliness_score formula (auditable, not magic): start at 100, subtract
  # 10 per orphan branch, 5 per tool-internal residue path, 2 per
  # uncommitted-residue entry, floored at 0.
  local cleanliness_score=$(( 100 - (orphan_count * 10) - (tool_internal_count * 5) - (uncommitted_residue * 2) ))
  if [ "$cleanliness_score" -lt 0 ]; then
    cleanliness_score=0
  fi

  jq -n \
    --argjson orphan_branches "$orphan_count" \
    --argjson orphan_branch_names "$(_json_array_from "${orphan_branch_names[@]:-}")" \
    --argjson uncommitted_residue "$uncommitted_residue" \
    --argjson uncommitted_residue_paths "$(_json_array_from "${uncommitted_residue_paths[@]:-}")" \
    --argjson tool_internal_paths_count "$tool_internal_count" \
    --argjson tool_internal_paths "$(_json_array_from "${tool_internal_paths[@]:-}")" \
    --argjson unnecessary_modifications "$unnecessary_count" \
    --argjson unnecessary_modification_paths "$(_json_array_from "${unnecessary_modification_paths[@]:-}")" \
    --argjson cleanliness_score "$cleanliness_score" \
    '{
      orphan_branches: $orphan_branches,
      orphan_branch_names: $orphan_branch_names,
      uncommitted_residue: $uncommitted_residue,
      uncommitted_residue_paths: $uncommitted_residue_paths,
      tool_internal_paths_count: $tool_internal_paths_count,
      tool_internal_paths: $tool_internal_paths,
      unnecessary_modifications: $unnecessary_modifications,
      unnecessary_modification_paths: $unnecessary_modification_paths,
      cleanliness_score: $cleanliness_score
    }'
}

# Allow standalone invocation.
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
  if [ $# -lt 4 ]; then
    echo "usage: git-cleanliness.sh REPO_PATH STARTING_BRANCH STARTING_SHA ALLOWLIST_FILE" >&2
    exit 1
  fi
  git_cleanliness "$1" "$2" "$3" "$4"
fi
