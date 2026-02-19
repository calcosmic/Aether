#!/bin/bash
# Aether Chamber Utilities
# Manages entombed colonies — directory management, manifest generation, integrity verification
#
# Usage:
#   source .aether/utils/chamber-utils.sh
#   chamber_create <chamber_dir> <state_file> <goal> <phases_completed> <total_phases> <milestone> <version> <decisions_json> <learnings_json>
#   chamber_verify <chamber_dir>
#   chamber_list <chambers_root>
#   chamber_sanitize_goal <goal>

set -euo pipefail

# Initialize lock state before sourcing (file-lock.sh trap needs these)
LOCK_ACQUIRED=${LOCK_ACQUIRED:-false}
CURRENT_LOCK=${CURRENT_LOCK:-""}

# Get script directory for sourcing (preserve parent SCRIPT_DIR if set)
__chamber_utils_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$__chamber_utils_dir/../.." && pwd 2>/dev/null || echo "$__chamber_utils_dir/../..")"

# Use parent SCRIPT_DIR if available, otherwise use local
SCRIPT_DIR="${SCRIPT_DIR:-$__chamber_utils_dir}"

# Source atomic-write for safe file operations
[[ -f "$SCRIPT_DIR/atomic-write.sh" ]] && source "$SCRIPT_DIR/atomic-write.sh"

# --- JSON output helpers ---
json_ok() { printf '{"ok":true,"result":%s}\n' "$1"; }

# Guard: yield to error-handler.sh's enhanced json_err when already loaded
if ! type json_err &>/dev/null; then
  json_err() {
    local code="${1:-E_UNKNOWN}"
    local message="${2:-An unknown error occurred}"
    printf '{"ok":false,"error":{"code":"%s","message":"%s"}}\n' "$code" "$message" >&2
    exit 1
  }
fi

# Fallback E_* constants (no-ops when error-handler.sh is already loaded)
: "${E_UNKNOWN:=E_UNKNOWN}"
: "${E_VALIDATION_FAILED:=E_VALIDATION_FAILED}"
: "${E_FILE_NOT_FOUND:=E_FILE_NOT_FOUND}"
: "${E_BASH_ERROR:=E_BASH_ERROR}"
: "${E_JSON_INVALID:=E_JSON_INVALID}"

# --- Chamber Functions ---

# Sanitize goal string for use in directory names
# Converts to lowercase, replaces spaces/special chars with hyphens, removes non-alphanumeric
chamber_sanitize_goal() {
  local goal="$1"
  # Convert to lowercase, replace spaces and special chars with hyphens
  local sanitized=$(echo "$goal" | tr '[:upper:]' '[:lower:]' | tr -cs '[:alnum:]' '-')
  # Remove leading/trailing hyphens
  sanitized=$(echo "$sanitized" | sed 's/^-//;s/-$//')
  # Limit length to avoid overly long directory names
  if [[ ${#sanitized} -gt 50 ]]; then
    sanitized="${sanitized:0:50}"
  fi
  echo "$sanitized"
}

# Compute SHA256 hash of a file
# Returns hash string or empty on error
chamber_compute_hash() {
  local file_path="$1"
  if [[ ! -f "$file_path" ]]; then
    echo ""
    return 1
  fi

  # Try sha256sum first (Linux), then shasum -a 256 (macOS)
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file_path" | cut -d' ' -f1
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file_path" | cut -d' ' -f1
  else
    echo ""
    return 1
  fi
}

# Create a new chamber (entomb a colony)
# Arguments:
#   chamber_dir: Directory to create for this chamber
#   state_file: Path to COLONY_STATE.json to archive
#   goal: Colony goal string
#   phases_completed: Number of completed phases
#   total_phases: Total number of phases
#   milestone: Milestone name
#   version: Version string
#   decisions_json: JSON array of decisions
#   learnings_json: JSON array of learnings
chamber_create() {
  local chamber_dir="$1"
  local state_file="$2"
  local goal="$3"
  local phases_completed="$4"
  local total_phases="$5"
  local milestone="$6"
  local version="$7"
  local decisions_json="$8"
  local learnings_json="$9"

  # Validate inputs
  [[ -z "$chamber_dir" ]] && json_err "$E_VALIDATION_FAILED" "chamber_dir argument is required. Try: pass the chamber directory path."
  [[ -z "$state_file" ]] && json_err "$E_VALIDATION_FAILED" "state_file argument is required. Try: pass the state file path."
  [[ ! -f "$state_file" ]] && json_err "$E_FILE_NOT_FOUND" "State file not found: $state_file. Try: check the file path."

  # Create chamber directory
  mkdir -p "$chamber_dir" || json_err "$E_BASH_ERROR" "Couldn't create chamber directory: $chamber_dir. Try: check disk space and permissions."

  # Copy state file to chamber
  local target_state="$chamber_dir/COLONY_STATE.json"
  cp "$state_file" "$target_state" || json_err "$E_BASH_ERROR" "Couldn't copy the state file. Try: check disk space and permissions."

  # Compute hash of the copied state file
  local state_hash=$(chamber_compute_hash "$target_state")
  [[ -z "$state_hash" ]] && json_err "$E_BASH_ERROR" "Couldn't compute state file hash. Try: check that shasum is available."

  # Generate timestamp
  local entombed_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

  # Create manifest.json
  local manifest_file="$chamber_dir/manifest.json"
  local manifest_content=$(cat <<EOF
{
  "entombed_at": "$entombed_at",
  "goal": $(echo "$goal" | jq -Rs '.[:-1]'),
  "phases_completed": $phases_completed,
  "total_phases": $total_phases,
  "milestone": $(echo "$milestone" | jq -Rs '.[:-1]'),
  "version": $(echo "$version" | jq -Rs '.[:-1]'),
  "decisions": $decisions_json,
  "learnings": $learnings_json,
  "files": {
    "COLONY_STATE.json": "$state_hash"
  }
}
EOF
)

  # Write manifest atomically if atomic_write is available, otherwise direct
  if type atomic_write &>/dev/null; then
    atomic_write "$manifest_file" "$manifest_content" || json_err "$E_BASH_ERROR" "Couldn't write chamber manifest. Try: check disk space."
  else
    echo "$manifest_content" > "$manifest_file" || json_err "$E_BASH_ERROR" "Couldn't write chamber manifest. Try: check disk space."
  fi

  # Verify the manifest was written correctly
  if [[ ! -f "$manifest_file" ]]; then
    json_err "$E_FILE_NOT_FOUND" "Chamber manifest wasn't created. Try: check disk space and permissions."
  fi

  # Return success with chamber info
  local result=$(cat <<EOF
{
  "chamber_dir": "$chamber_dir",
  "manifest": {
    "entombed_at": "$entombed_at",
    "goal": $(echo "$goal" | jq -Rs '.[:-1]'),
    "phases_completed": $phases_completed,
    "total_phases": $total_phases,
    "milestone": $(echo "$milestone" | jq -Rs '.[:-1]'),
    "version": $(echo "$version" | jq -Rs '.[:-1]')
  }
}
EOF
)

  json_ok "$result"
}

# Verify chamber integrity
# Arguments:
#   chamber_dir: Directory containing the chamber
chamber_verify() {
  local chamber_dir="$1"

  # Validate inputs
  [[ -z "$chamber_dir" ]] && json_err "$E_VALIDATION_FAILED" "chamber_dir argument is required. Try: pass the chamber directory path."
  [[ ! -d "$chamber_dir" ]] && json_err "$E_FILE_NOT_FOUND" "Chamber directory not found: $chamber_dir. Try: check the path."

  local manifest_file="$chamber_dir/manifest.json"
  local state_file="$chamber_dir/COLONY_STATE.json"

  # Check required files exist
  [[ ! -f "$manifest_file" ]] && json_err "$E_FILE_NOT_FOUND" "Manifest not found in chamber. Try: verify the chamber was created correctly."
  [[ ! -f "$state_file" ]] && json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found in chamber. Try: re-entomb the colony."

  # Read stored hash from manifest
  local stored_hash=$(jq -r '.files["COLONY_STATE.json"] // empty' "$manifest_file" 2>/dev/null)
  [[ -z "$stored_hash" ]] && json_err "$E_JSON_INVALID" "No hash found in manifest. Try: re-entomb the colony."

  # Compute current hash
  local current_hash=$(chamber_compute_hash "$state_file")
  [[ -z "$current_hash" ]] && json_err "$E_BASH_ERROR" "Couldn't compute state file hash. Try: check that shasum is available."

  # Compare hashes
  if [[ "$stored_hash" != "$current_hash" ]]; then
    local result=$(cat <<EOF
{
  "verified": false,
  "chamber_dir": "$chamber_dir",
  "error": "hash mismatch",
  "stored_hash": "$stored_hash",
  "current_hash": "$current_hash"
}
EOF
)
    json_ok "$result"
    return 0
  fi

  # Verification passed
  local result=$(cat <<EOF
{
  "verified": true,
  "chamber_dir": "$chamber_dir",
  "hash": "$current_hash"
}
EOF
)

  json_ok "$result"
}

# List all chambers
# Arguments:
#   chambers_root: Root directory containing chambers (default: .aether/chambers/)
chamber_list() {
  local chambers_root="${1:-$AETHER_ROOT/.aether/chambers}"

  # Default to current directory's chambers if AETHER_ROOT not set
  if [[ -z "$chambers_root" || "$chambers_root" == "/.aether/chambers" ]]; then
    chambers_root="$(pwd)/.aether/chambers"
  fi

  # Check if chambers directory exists
  if [[ ! -d "$chambers_root" ]]; then
    json_ok "[]"
    return 0
  fi

  # Build array of chamber summaries
  local chambers="["
  local first=true

  # Find all directories in chambers_root
  while IFS= read -r -d '' chamber_dir; do
    local chamber_name=$(basename "$chamber_dir")
    local manifest_file="$chamber_dir/manifest.json"

    # Skip if no manifest
    [[ ! -f "$manifest_file" ]] && continue

    # Read manifest fields
    local goal=$(jq -r '.goal // "unknown"' "$manifest_file" 2>/dev/null)
    local milestone=$(jq -r '.milestone // "unknown"' "$manifest_file" 2>/dev/null)
    local phases_completed=$(jq -r '.phases_completed // 0' "$manifest_file" 2>/dev/null)
    local entombed_at=$(jq -r '.entombed_at // ""' "$manifest_file" 2>/dev/null)

    # Escape for JSON
    goal=$(echo "$goal" | jq -Rs '.[:-1]')
    milestone=$(echo "$milestone" | jq -Rs '.[:-1]')

    # Add comma if not first
    if [[ "$first" == "true" ]]; then
      first=false
    else
      chambers+=","
    fi

    chambers+=$(cat <<EOF
{
  "name": $(echo "$chamber_name" | jq -Rs '.[:-1]'),
  "goal": $goal,
  "milestone": $milestone,
  "phases_completed": $phases_completed,
  "entombed_at": $(echo "$entombed_at" | jq -Rs '.[:-1]')
}
EOF
)
  done < <(find "$chambers_root" -mindepth 1 -maxdepth 1 -type d -print0 2>/dev/null || true)

  chambers+="]"

  # Sort by entombed_at descending using jq
  local sorted=$(echo "$chambers" | jq 'sort_by(.entombed_at) | reverse')

  json_ok "$sorted"
}

# Export functions for use in other scripts
export -f chamber_sanitize_goal chamber_compute_hash chamber_create chamber_verify chamber_list
