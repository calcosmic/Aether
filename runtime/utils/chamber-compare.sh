#!/bin/bash
# Chamber comparison utilities
# Usage: bash chamber-compare.sh <chamber_a> <chamber_b>

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHAMBERS_DIR="${CHAMBERS_DIR:-.aether/chambers}"

# JSON output helpers
json_ok() { printf '{"ok":true,"result":%s}\n' "$1"; }
json_err() {
  local message="${2:-$1}"
  printf '{"ok":false,"error":"%s"}\n' "$message" >&2
  exit 1
}

# Load chamber manifest
load_chamber() {
  local chamber_name="$1"
  local manifest_file="$CHAMBERS_DIR/$chamber_name/manifest.json"

  if [[ ! -f "$manifest_file" ]]; then
    json_err "Chamber not found: $chamber_name"
  fi

  cat "$manifest_file"
}

# Compare two chambers
cmd="${1:-help}"
shift 2>/dev/null || true

case "$cmd" in
  help)
    cat <<'EOF'
{"ok":true,"commands":["compare","diff","stats"],"description":"Chamber comparison utilities"}
EOF
    ;;

  compare)
    chamber_a="${1:-}"
    chamber_b="${2:-}"
    [[ -z "$chamber_a" || -z "$chamber_b" ]] && json_err "Usage: compare <chamber_a> <chamber_b>"

    # Load both manifests
    manifest_a=$(load_chamber "$chamber_a")
    manifest_b=$(load_chamber "$chamber_b")

    # Extract key fields for comparison
    result=$(jq -n \
      --arg a_name "$chamber_a" \
      --arg b_name "$chamber_b" \
      --argjson a "$manifest_a" \
      --argjson b "$manifest_b" \
      '{
        chamber_a: {
          name: $a_name,
          goal: $a.goal,
          milestone: $a.milestone,
          version: $a.version,
          phases_completed: $a.phases_completed,
          total_phases: $a.total_phases,
          entombed_at: $a.entombed_at,
          decisions_count: ($a.decisions | length),
          learnings_count: ($a.learnings | length)
        },
        chamber_b: {
          name: $b_name,
          goal: $b.goal,
          milestone: $b.milestone,
          version: $b.version,
          phases_completed: $b.phases_completed,
          total_phases: $b.total_phases,
          entombed_at: $b.entombed_at,
          decisions_count: ($b.decisions | length),
          learnings_count: ($b.learnings | length)
        },
        comparison: {
          phases_diff: ($b.phases_completed - $a.phases_completed),
          decisions_diff: (($b.decisions | length) - ($a.decisions | length)),
          learnings_diff: (($b.learnings | length) - ($a.learnings | length)),
          same_milestone: ($a.milestone == $b.milestone),
          time_between: (
            (($b.entombed_at | fromdateiso8601) - ($a.entombed_at | fromdateiso8601)) / 86400 | floor
          )
        }
      }')

    json_ok "$result"
    ;;

  diff)
    chamber_a="${1:-}"
    chamber_b="${2:-}"
    [[ -z "$chamber_a" || -z "$chamber_b" ]] && json_err "Usage: diff <chamber_a> <chamber_b>"

    manifest_a=$(load_chamber "$chamber_a")
    manifest_b=$(load_chamber "$chamber_b")

    # Find decisions in B but not in A (new decisions)
    # Find learnings in B but not in A (new learnings)
    result=$(jq -n \
      --arg a_name "$chamber_a" \
      --arg b_name "$chamber_b" \
      --argjson a "$manifest_a" \
      --argjson b "$manifest_b" \
      '{
        new_decisions: [
          $b.decisions[] | select(
            .content as $content |
            $a.decisions | map(.content) | contains([$content]) | not
          )
        ],
        new_learnings: [
          $b.learnings[] | select(
            .content as $content |
            $a.learnings | map(.content) | contains([$content]) | not
          )
        ],
        preserved_decisions: [
          $a.decisions[] | select(
            .content as $content |
            $b.decisions | map(.content) | contains([$content])
          )
        ],
        preserved_learnings: [
          $a.learnings[] | select(
            .content as $content |
            $b.learnings | map(.content) | contains([$content])
          )
        ]
      }')

    json_ok "$result"
    ;;

  stats)
    chamber_a="${1:-}"
    chamber_b="${2:-}"
    [[ -z "$chamber_a" || -z "$chamber_b" ]] && json_err "Usage: stats <chamber_a> <chamber_b>"

    manifest_a=$(load_chamber "$chamber_a")
    manifest_b=$(load_chamber "$chamber_b")

    # Calculate detailed statistics
    result=$(jq -n \
      --arg a_name "$chamber_a" \
      --arg b_name "$chamber_b" \
      --argjson a "$manifest_a" \
      --argjson b "$manifest_b" \
      '{
        summary: {
          a_phases: $a.phases_completed,
          b_phases: $b.phases_completed,
          growth: "\($b.phases_completed - $a.phases_completed) phases",
          a_duration_days: null,
          b_duration_days: null
        },
        knowledge_transfer: {
          decisions_preserved: ($a.decisions | map(.content) | intersection($b.decisions | map(.content)) | length),
          decisions_new: ($b.decisions | map(.content) - ($a.decisions | map(.content)) | length),
          learnings_preserved: ($a.learnings | map(.content) | intersection($b.learnings | map(.content)) | length),
          learnings_new: ($b.learnings | map(.content) - ($a.learnings | map(.content)) | length)
        },
        evolution: {
          milestone_changed: ($a.milestone != $b.milestone),
          from_milestone: $a.milestone,
          to_milestone: $b.milestone,
          version_delta: "\($a.version) → \($b.version)"
        }
      }')

    json_ok "$result"
    ;;

  *)
    json_err "Unknown command: $cmd. Use: compare, diff, stats"
    ;;
esac
