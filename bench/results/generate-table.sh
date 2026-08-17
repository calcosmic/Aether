#!/usr/bin/env bash
# generate-table.sh — generates the benchmark results table (markdown) from
# per-run run.json evidence files. Never hand-edited; every number in the
# table traces back to a run.json a run actually wrote.
#
# Usage: generate-table.sh RESULTS_DIR [--partial]
#   RESULTS_DIR — a dated results directory (e.g. bench/results/2026-08-17)
#                 containing one <lane>__<category>/run.json per completed
#                 cell.
#   --partial   — emit the table even if fewer than 12 run.json files are
#                 present, with a clearly-marked incomplete banner naming
#                 the missing cells. Without this flag, fewer than 12
#                 run.json files is a hard failure — no partial table is
#                 silently emitted.
set -euo pipefail

fail() { echo "GENERATE-TABLE FAIL: $*" >&2; exit 1; }

LANES=(gsd aether-interactive aether-autopilot)
CATEGORIES=(01-bug-fix 02-brownfield-feature 03-interrupted-execution 04-fresh-repo-lifecycle)
EXPECTED_COUNT=12

RESULTS_DIR="${1:-}"
PARTIAL=0
if [ "${2:-}" = "--partial" ] || [ "${1:-}" = "--partial" ]; then
  PARTIAL=1
fi
if [ "$RESULTS_DIR" = "--partial" ]; then
  RESULTS_DIR="${2:-}"
fi

[ -n "$RESULTS_DIR" ] || fail "usage: generate-table.sh RESULTS_DIR [--partial]"
[ -d "$RESULTS_DIR" ] || fail "results directory does not exist: $RESULTS_DIR"

# --- collect every run.json beneath RESULTS_DIR -----------------------------
# Bash 3.2 (macOS default) has no associative arrays, so cell presence is
# tracked with plain arrays rather than a `declare -A` map — same
# compatibility constraint bench/lib/git-cleanliness.sh documents.
declare -a RUN_JSON_FILES=()
declare -a MISSING_CELLS=()
for lane in "${LANES[@]}"; do
  for category in "${CATEGORIES[@]}"; do
    f="$RESULTS_DIR/${lane}__${category}/run.json"
    if [ -f "$f" ]; then
      RUN_JSON_FILES+=("$f")
    else
      MISSING_CELLS+=("${lane}__${category}")
    fi
  done
done

FOUND_COUNT="${#RUN_JSON_FILES[@]}"

if [ "$FOUND_COUNT" -lt "$EXPECTED_COUNT" ] && [ "$PARTIAL" -eq 0 ]; then
  fail "found $FOUND_COUNT/$EXPECTED_COUNT run.json files under $RESULTS_DIR — refusing to emit a partial table without --partial. Missing: ${MISSING_CELLS[*]}"
fi

# --- emit the markdown table -------------------------------------------------
if [ "$FOUND_COUNT" -lt "$EXPECTED_COUNT" ]; then
  echo "> **INCOMPLETE RESULTS — $FOUND_COUNT/$EXPECTED_COUNT cells present.**"
  echo ">"
  echo "> Missing cells: ${MISSING_CELLS[*]}"
  echo
fi

echo "# Benchmark results: $RESULTS_DIR"
echo
echo "| Lane | Category | Repo | Acceptance | Interventions (scripted/unscripted) | Hallucinated completion | Unnecessary mods | Cleanliness | Tokens | Model calls | Wall-clock net (s) |"
echo "|---|---|---|---|---|---|---|---|---|---|---|"

for f in "${RUN_JSON_FILES[@]}"; do
  jq -r '
    [
      .lane,
      .category,
      .repo,
      (if .acceptance_pass then "PASS" else "FAIL" end),
      ((.operator.scripted_inputs // 0) | tostring) + "/" + ((.operator.unscripted_inputs // 0) | tostring),
      (.hallucinated_completion // "unknown"),
      ((.cleanliness.unnecessary_modifications // "n/a") | tostring),
      ((.cleanliness.cleanliness_score // "n/a") | tostring),
      ((.tokens.total_tokens // "n/a") | tostring),
      ((.tokens.model_calls // "n/a") | tostring),
      ((.operator.wall_clock_net_seconds // "n/a") | tostring)
    ] | "| " + join(" | ") + " |"
  ' "$f"
done

echo
echo "## Per-lane summary"
echo
echo "| Lane | Runs completed | Autonomous successes (out of 4) | Total tokens | Median tokens | Total unscripted interventions | Hallucinated completions | Mean cleanliness |"
echo "|---|---|---|---|---|---|---|---|"

for lane in "${LANES[@]}"; do
  declare -a lane_files=()
  for category in "${CATEGORIES[@]}"; do
    f="$RESULTS_DIR/${lane}__${category}/run.json"
    [ -f "$f" ] && lane_files+=("$f")
  done
  count="${#lane_files[@]}"
  if [ "$count" -eq 0 ]; then
    echo "| $lane | 0 | 0/4 | n/a | n/a | n/a | n/a | n/a |"
    continue
  fi
  {
    for f in "${lane_files[@]}"; do
      cat "$f"
    done
  } | jq -s -r \
    --arg lane "$lane" \
    --argjson count "$count" '
    (map(select(.acceptance_pass == true)) | length) as $autonomous_successes
    |
    (map(.tokens.total_tokens // 0) ) as $token_list
    |
    ($token_list | add // 0) as $total_tokens
    |
    ($token_list | sort) as $sorted_tokens
    |
    (
      if ($sorted_tokens | length) == 0 then 0
      elif ($sorted_tokens | length) % 2 == 1 then $sorted_tokens[($sorted_tokens | length - 1) / 2]
      else (($sorted_tokens[($sorted_tokens | length / 2) - 1] + $sorted_tokens[$sorted_tokens | length / 2]) / 2)
      end
    ) as $median_tokens
    |
    (map(.operator.unscripted_inputs // 0) | add // 0) as $total_unscripted
    |
    (map(select(.hallucinated_completion == "true")) | length) as $hallucinated_count
    |
    (map(.cleanliness.cleanliness_score // 0) | add / length) as $mean_cleanliness
    |
    "| \($lane) | \($count) | \($autonomous_successes)/4 | \($total_tokens) | \($median_tokens) | \($total_unscripted) | \($hallucinated_count) | \($mean_cleanliness) |"
  '
  unset lane_files
done

echo
echo "## Recorded deviations"
echo
DEVIATIONS_FOUND=0
for f in "${RUN_JSON_FILES[@]}"; do
  MODEL_DEV="$(jq -r '.model_deviation // empty' "$f")"
  if [ -n "$MODEL_DEV" ]; then
    CELL="$(jq -r '.lane + "__" + .category' "$f")"
    echo "- **$CELL:** model deviation — $MODEL_DEV"
    DEVIATIONS_FOUND=1
  fi
  PARALLEL_NOTE="$(jq -r '.parallel_mode_note // empty' "$f")"
  if [ -n "$PARALLEL_NOTE" ]; then
    CELL="$(jq -r '.lane + "__" + .category' "$f")"
    echo "- **$CELL:** parallel mode is \`$(jq -r '.parallel_mode' "$f")\` — $PARALLEL_NOTE"
    DEVIATIONS_FOUND=1
  fi
  HALLUC="$(jq -r '.hallucinated_completion // empty' "$f")"
  if [ "$HALLUC" = "unknown" ]; then
    CELL="$(jq -r '.lane + "__" + .category' "$f")"
    echo "- **$CELL:** hallucinated-completion flag is \`unknown\` — the claimed-success transcript file was absent for this cell"
    DEVIATIONS_FOUND=1
  fi
done
if [ "$DEVIATIONS_FOUND" -eq 0 ]; then
  echo "None recorded."
fi

echo
echo "---"
if [ "$FOUND_COUNT" -gt 0 ]; then
  FIRST_FILE="${RUN_JSON_FILES[0]}"
  TOKEN_SOURCE="$(jq -r '.tokens.source // "unknown"' "$FIRST_FILE")"
  PINNED_MODEL="$(jq -r '.model // "unknown"' "$FIRST_FILE")"
  echo "Token measurement mechanism: \`$TOKEN_SOURCE\`. Pinned model: \`$PINNED_MODEL\` (read from run.json, not hardcoded)."
else
  echo "No run.json files present — token measurement mechanism and pinned model cannot be reported."
fi
