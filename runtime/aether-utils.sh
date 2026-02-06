#!/bin/bash
# Aether Colony Utilities - Minimal Edition
# Provides: validate-state, error-add, pheromone-validate, activity-log
#
# Usage: bash runtime/aether-utils.sh <subcommand> [args...]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DATA_DIR="$PWD/.aether/data"

# Inline atomic write (no external dependency)
atomic_write() {
  local file="$1" content="$2"
  echo "$content" > "$file"
}

# --- JSON output helpers ---
json_ok() { printf '{"ok":true,"result":%s}\n' "$1"; }
json_err() { printf '{"ok":false,"error":"%s"}\n' "$1" >&2; exit 1; }

# --- Subcommand dispatch ---
cmd="${1:-help}"
shift 2>/dev/null || true

case "$cmd" in
  help)
    cat <<'EOF'
{"ok":true,"commands":["help","validate-state","pheromone-validate","error-add","activity-log"],"description":"Aether Colony Utilities - Minimal Edition"}
EOF
    ;;

  validate-state)
    case "${1:-colony}" in
      colony)
        [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "COLONY_STATE.json not found"
        json_ok "$(jq -e 'has("goal") and has("state")' "$DATA_DIR/COLONY_STATE.json" > /dev/null 2>&1 && echo '{"pass":true}' || echo '{"pass":false}')"
        ;;
      all)
        [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "COLONY_STATE.json not found"
        json_ok "$(jq -e 'has("goal") and has("state") and has("signals") and has("plan")' "$DATA_DIR/COLONY_STATE.json" > /dev/null 2>&1 && echo '{"pass":true}' || echo '{"pass":false}')"
        ;;
      *)
        json_err "Usage: validate-state [colony|all]"
        ;;
    esac
    ;;

  pheromone-validate)
    content="${1:-}"
    max_len="${2:-500}"
    if [[ ${#content} -gt $max_len ]]; then
      json_ok "{\"pass\":false,\"reason\":\"too_long\",\"length\":${#content},\"max\":$max_len}"
    elif [[ ${#content} -lt 20 ]]; then
      json_ok "{\"pass\":false,\"reason\":\"too_short\",\"length\":${#content},\"min\":20}"
    else
      json_ok "{\"pass\":true,\"length\":${#content}}"
    fi
    ;;

  error-add)
    [[ $# -ge 3 ]] || json_err "Usage: error-add <category> <severity> <description>"
    [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "COLONY_STATE.json not found"
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    updated=$(jq --arg cat "$1" --arg sev "$2" --arg desc "$3" --arg ts "$ts" '
      .errors += [{category:$cat, severity:$sev, message:$desc, timestamp:$ts}] |
      if (.errors|length) > 50 then .errors = .errors[-50:] else . end
    ' "$DATA_DIR/COLONY_STATE.json") || json_err "Failed to update state"
    atomic_write "$DATA_DIR/COLONY_STATE.json" "$updated"
    json_ok '"added"'
    ;;

  activity-log)
    action="${1:-}"
    caste="${2:-}"
    description="${3:-}"
    [[ -z "$action" || -z "$caste" || -z "$description" ]] && json_err "Usage: activity-log <action> <caste> <description>"
    log_file="$DATA_DIR/activity.log"
    ts=$(date -u +"%H:%M:%S")
    echo "[$ts] $action $caste: $description" >> "$log_file"
    json_ok '"logged"'
    ;;

  *)
    json_err "Unknown command: $cmd. Run 'help' for available commands."
    ;;
esac
