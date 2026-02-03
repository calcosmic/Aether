#!/bin/bash
# Aether Colony Utility Layer
# Single entry point for deterministic colony operations
#
# Usage: bash .aether/aether-utils.sh <subcommand> [args...]
#
# All subcommands output JSON to stdout.
# Non-zero exit on error with JSON error message to stderr.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$SCRIPT_DIR/.." && pwd 2>/dev/null || echo "$SCRIPT_DIR")"
DATA_DIR="$SCRIPT_DIR/data"

# Initialize lock state before sourcing (file-lock.sh trap needs these)
LOCK_ACQUIRED=${LOCK_ACQUIRED:-false}
CURRENT_LOCK=${CURRENT_LOCK:-""}

# Source shared infrastructure
source "$SCRIPT_DIR/utils/file-lock.sh"
source "$SCRIPT_DIR/utils/atomic-write.sh"

# --- JSON output helpers ---
# Success: JSON to stdout, exit 0
json_ok() { printf '{"ok":true,"result":%s}\n' "$1"; }

# Error: JSON to stderr, exit 1
json_err() { printf '{"ok":false,"error":"%s"}\n' "$1" >&2; exit 1; }

# --- Subcommand dispatch ---
cmd="${1:-help}"
shift 2>/dev/null || true

case "$cmd" in
  help)
    cat <<'EOF'
{"ok":true,"commands":["help","version","pheromone-decay","pheromone-effective","pheromone-batch","pheromone-cleanup","pheromone-combine"],"description":"Aether Colony Utility Layer — deterministic ops for the ant colony"}
EOF
    ;;
  version)
    json_ok '"0.1.0"'
    ;;
  pheromone-decay)
    [[ $# -ge 3 ]] || json_err "Usage: pheromone-decay <strength> <elapsed_seconds> <half_life>"
    json_ok "$(jq -n --arg s "$1" --arg e "$2" --arg h "$3" \
      '{strength: (($s|tonumber) * ((-0.693147180559945 * ($e|tonumber) / ($h|tonumber)) | exp) | . * 1000000 | round / 1000000)}')"
    ;;
  pheromone-effective)
    [[ $# -ge 2 ]] || json_err "Usage: pheromone-effective <sensitivity> <strength>"
    json_ok "$(jq -n --arg sens "$1" --arg str "$2" \
      '{effective_signal: (($sens|tonumber) * ($str|tonumber) | . * 1000000 | round / 1000000)}')"
    ;;
  pheromone-batch)
    [[ -f "$DATA_DIR/pheromones.json" ]] || json_err "pheromones.json not found"
    now=$(date -u +%s)
    json_ok "$(jq --arg now "$now" '.signals | map(. + {
      current_strength: (
        if .half_life_seconds == null then .strength
        else .strength * ((-0.693147180559945 * (($now|tonumber) - (.created_at | sub("\\.[0-9]+Z$";"Z") | fromdate)) / .half_life_seconds) | exp)
        end | . * 1000 | round / 1000)
    })' "$DATA_DIR/pheromones.json")" || json_err "Failed to read pheromones.json"
    ;;
  pheromone-cleanup)
    [[ -f "$DATA_DIR/pheromones.json" ]] || json_err "pheromones.json not found"
    now=$(date -u +%s)
    before=$(jq '.signals | length' "$DATA_DIR/pheromones.json")
    result=$(jq --arg now "$now" '.signals |= map(select(
      .half_life_seconds == null or
      (.strength * ((-0.693147180559945 * (($now|tonumber) - (.created_at | sub("\\.[0-9]+Z$";"Z") | fromdate)) / .half_life_seconds) | exp)) >= 0.05
    ))' "$DATA_DIR/pheromones.json") || json_err "Failed to process pheromones.json"
    atomic_write "$DATA_DIR/pheromones.json" "$result"
    after=$(echo "$result" | jq '.signals | length')
    json_ok "{\"removed\":$((before - after)),\"remaining\":$after}"
    ;;
  pheromone-combine)
    [[ $# -ge 2 ]] || json_err "Usage: pheromone-combine <signal1_strength> <signal2_strength>"
    json_ok "$(jq -n --arg s1 "$1" --arg s2 "$2" '{
      net_effect: ((($s1|tonumber) - ($s2|tonumber)) | if . < 0 then 0 else . end | . * 1000 | round / 1000),
      dominant: (if ($s1|tonumber) >= ($s2|tonumber) then "signal1" else "signal2" end),
      ratio: (if ($s2|tonumber) == 0 then null else (($s1|tonumber) / ($s2|tonumber)) | . * 1000 | round / 1000 end)
    }')"
    ;;
  *)
    json_err "Unknown command: $cmd"
    ;;
esac
