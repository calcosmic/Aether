#!/usr/bin/env bash
# operator-log.sh — append-only, timestamped operator-input log for the
# Aether-vs-GSD benchmark harness.
#
# Sourceable library backing an append-only JSONL log at the path given by
# the OPERATOR_LOG environment variable. Every write is an atomic append of
# one JSON line. Timestamps use `date -u +%Y-%m-%dT%H:%M:%SZ` (RFC3339 UTC).
#
# UNLIKE the hook-side capture pattern this shape is adapted from
# (cmd/spend_session_capture.go's recordSpendSessionFromHook, which is
# fail-soft by design — a capture failure must never change a hook's
# allow/deny answer), failures HERE are NOT silent. If OPERATOR_LOG is unset
# or its directory is not writable, every logging function fails loudly with
# a named message. A benchmark run whose operator log silently vanished
# would report an intervention count of zero and look like the best result
# in the results table — exactly the failure mode this harness exists to
# prevent.
#
# Deliberately NOT `set -euo pipefail` at the top level (unlike this
# directory's other libraries): every public function here returns non-zero
# on failure so the CALLER can inspect `$?` after a call (this is exactly
# what oplog_input's acceptance test does), and a top-level `set -e` would
# abort the calling shell at the failing statement before that inspection
# ever runs. `pipefail` alone is safe and kept; each function still fails
# loudly on its own error paths via explicit `return 1`.
set -uo pipefail

_oplog_fail() { echo "OPERATOR-LOG FAIL: $*" >&2; return 1; }

# _oplog_check_configured — every public function calls this first.
_oplog_check_configured() {
  if [ -z "${OPERATOR_LOG:-}" ]; then
    _oplog_fail "OPERATOR_LOG is unset — an operator-log call with no configured log path would silently lose the intervention record"
    return 1
  fi
  local log_dir
  log_dir="$(dirname "$OPERATOR_LOG")"
  if [ ! -d "$log_dir" ] || [ ! -w "$log_dir" ]; then
    _oplog_fail "OPERATOR_LOG directory is not writable: $log_dir (OPERATOR_LOG=$OPERATOR_LOG)"
    return 1
  fi
}

_oplog_now() {
  date -u +%Y-%m-%dT%H:%M:%SZ
}

# _oplog_append JSON_LINE — atomic single-line append.
_oplog_append() {
  printf '%s\n' "$1" >> "$OPERATOR_LOG"
}

# oplog_start RUN_ID LANE CATEGORY REPO MODEL
oplog_start() {
  _oplog_check_configured || return 1
  local run_id="$1" lane="$2" category="$3" repo="$4" model="$5"
  local ts
  ts="$(_oplog_now)"
  _oplog_append "$(jq -nc \
    --arg event "run_start" \
    --arg ts "$ts" \
    --arg run_id "$run_id" \
    --arg lane "$lane" \
    --arg category "$category" \
    --arg repo "$repo" \
    --arg model "$model" \
    '{event: $event, timestamp: $ts, run_id: $run_id, lane: $lane, category: $category, repo: $repo, model: $model}')"
}

# oplog_input TYPE TEXT — TYPE must be exactly "scripted" or "unscripted".
oplog_input() {
  _oplog_check_configured || return 1
  local input_type="$1" text="${2:-}"
  if [ "$input_type" != "scripted" ] && [ "$input_type" != "unscripted" ]; then
    _oplog_fail "oplog_input: input_type must be exactly 'scripted' or 'unscripted', got '$input_type' — an untyped input would be silently uncounted, and the intervention count is one of the two headline metrics"
    return 1
  fi
  local ts
  ts="$(_oplog_now)"
  _oplog_append "$(jq -nc \
    --arg event "operator_input" \
    --arg ts "$ts" \
    --arg input_type "$input_type" \
    --arg text "$text" \
    '{event: $event, timestamp: $ts, input_type: $input_type, text: $text}')"
}

# oplog_wait_start — brackets the start of a period where the harness is
# waiting on the operator (not on the system under test).
oplog_wait_start() {
  _oplog_check_configured || return 1
  local ts
  ts="$(_oplog_now)"
  _oplog_append "$(jq -nc --arg event "wait_start" --arg ts "$ts" '{event: $event, timestamp: $ts}')"
}

# oplog_wait_end — brackets the end of an operator-wait period.
oplog_wait_end() {
  _oplog_check_configured || return 1
  local ts
  ts="$(_oplog_now)"
  _oplog_append "$(jq -nc --arg event "wait_end" --arg ts "$ts" '{event: $event, timestamp: $ts}')"
}

# oplog_event NAME JSON_FIELDS — a generic milestone record. JSON_FIELDS is
# a JSON object string (e.g. '{"pid":1234}'); pass '{}' for none.
oplog_event() {
  _oplog_check_configured || return 1
  local name="$1" json_fields="${2:-{\}}"
  local ts
  ts="$(_oplog_now)"
  _oplog_append "$(jq -nc \
    --arg event "$name" \
    --arg ts "$ts" \
    --argjson fields "$json_fields" \
    '{event: $event, timestamp: $ts} + $fields')"
}

# oplog_end STATUS
oplog_end() {
  _oplog_check_configured || return 1
  local status="$1"
  local ts
  ts="$(_oplog_now)"
  _oplog_append "$(jq -nc \
    --arg event "run_end" \
    --arg ts "$ts" \
    --arg status "$status" \
    '{event: $event, timestamp: $ts, status: $status}')"
}

# oplog_summary LOGFILE
#
# Reads a completed log and emits one JSON object: scripted_inputs,
# unscripted_inputs, autonomous (derived — true only when unscripted_inputs
# is 0, never accepted as an input), wall_clock_seconds (run_end minus
# run_start), wait_seconds (summed wait brackets), wall_clock_net_seconds
# (wall_clock_seconds minus wait_seconds).
oplog_summary() {
  local logfile="$1"
  if [ ! -f "$logfile" ]; then
    _oplog_fail "oplog_summary: log file does not exist: $logfile"
    return 1
  fi

  jq -s '
    def to_epoch: strptime("%Y-%m-%dT%H:%M:%SZ") | mktime;

    (map(select(.event == "operator_input" and .input_type == "scripted")) | length) as $scripted
    |
    (map(select(.event == "operator_input" and .input_type == "unscripted")) | length) as $unscripted
    |
    ((map(select(.event == "run_start")) | first | .timestamp) // null) as $start_ts
    |
    ((map(select(.event == "run_end")) | first | .timestamp) // null) as $end_ts
    |
    (if $start_ts != null and $end_ts != null
       then (($end_ts | to_epoch) - ($start_ts | to_epoch))
       else 0
     end) as $wall_clock_seconds
    |
    (map(select(.event == "wait_start" or .event == "wait_end")) | sort_by(.timestamp)) as $wait_events
    |
    (reduce (range(0; ($wait_events | length))) as $i
       ({total: 0, pending: null};
         if ($wait_events[$i].event == "wait_start")
           then (.pending = $wait_events[$i].timestamp)
         elif ($wait_events[$i].event == "wait_end" and .pending != null)
           then (.total += (($wait_events[$i].timestamp | to_epoch) - (.pending | to_epoch)) | .pending = null)
         else .
         end)
       | .total) as $wait_seconds
    |
    {
      scripted_inputs: $scripted,
      unscripted_inputs: $unscripted,
      autonomous: ($unscripted == 0),
      wall_clock_seconds: $wall_clock_seconds,
      wait_seconds: $wait_seconds,
      wall_clock_net_seconds: ($wall_clock_seconds - $wait_seconds)
    }
  ' "$logfile"
}

# Allow standalone invocation for quick manual checks:
#   OPERATOR_LOG=/path/to/log operator-log.sh summary /path/to/log
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
  case "${1:-}" in
    summary)
      oplog_summary "$2"
      ;;
    *)
      echo "usage: OPERATOR_LOG=... operator-log.sh summary LOGFILE" >&2
      exit 1
      ;;
  esac
fi
