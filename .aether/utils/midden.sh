#!/bin/bash
# Midden (failure tracking) utility functions — extracted from aether-utils.sh
# Provides: _midden_write, _midden_recent_failures, _midden_review, _midden_acknowledge
#
# These functions are sourced by aether-utils.sh at startup.
# All shared infrastructure (json_ok, json_err, atomic_write, acquire_lock,
# release_lock, LOCK_DIR, DATA_DIR, SCRIPT_DIR, error constants) is available.

_midden_try_write() {
    # Helper: write updated JSON to midden file with retry
    # Usage: _midden_try_write <updated_json> <midden_file>
    # Returns: 0 on success, 1 on failure
    local mtw_json="$1"
    local mtw_file="$2"
    local mtw_tmp="${mtw_file}.tmp.$$"

    if ! { printf '%s\n' "$mtw_json" > "$mtw_tmp" && mv "$mtw_tmp" "$mtw_file"; }; then
      # Silent retry (once)
      if ! { printf '%s\n' "$mtw_json" > "$mtw_tmp" && mv "$mtw_tmp" "$mtw_file"; }; then
        echo "Warning: Midden write failed after retry -- entry may not have been saved." >&2
        return 1
      fi
    fi
    return 0
}

_midden_write() {
    # Write a warning/observation to the midden for later review
    # Usage: midden-write <category> <message> <source>
    # Example: midden-write "security" "High CVEs found: 3" "gatekeeper"
    # Returns: JSON with success status and entry details

    mw_category="${1:-general}"
    mw_message="${2:-}"
    mw_source="${3:-unknown}"

    # Graceful degradation: if no message, return success but note it
    if [[ -z "$mw_message" ]]; then
      json_ok "{\"success\":true,\"warning\":\"no_message_provided\",\"entry_id\":null}"
      return 0
    fi

    mw_midden_dir="$DATA_DIR/midden"
    mw_midden_file="$mw_midden_dir/midden.json"
    mw_timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    mw_entry_id="midden_$(date +%s)_$$"

    # Create midden directory if it doesn't exist
    mkdir -p "$mw_midden_dir"

    # Initialize midden.json if it doesn't exist
    if [[ ! -f "$mw_midden_file" ]]; then
      printf '%s\n' '{"version":"1.0.0","entries":[]}' > "$mw_midden_file"
    fi

    # Create the new entry using jq for safe JSON construction
    mw_new_entry=$(jq -n \
      --arg id "$mw_entry_id" \
      --arg ts "$mw_timestamp" \
      --arg cat "$mw_category" \
      --arg src "$mw_source" \
      --arg msg "$mw_message" \
      '{id: $id, timestamp: $ts, category: $cat, source: $src, message: $msg, reviewed: false}')

    # Append to midden.json using jq with locking
    if acquire_lock "$mw_midden_file" 2>/dev/null; then
      mw_updated_midden=$(jq --argjson entry "$mw_new_entry" '
        .entries += [$entry] |
        .entry_count = (.entries | length)
      ' "$mw_midden_file" 2>/dev/null)

      if [[ -n "$mw_updated_midden" ]]; then
        _midden_try_write "$mw_updated_midden" "$mw_midden_file"
        release_lock 2>/dev/null || true
        mw_total=$(jq '.entries | length' "$mw_midden_file" 2>/dev/null || echo 0)
        json_ok "$(jq -n --arg entry_id "$mw_entry_id" --arg category "$mw_category" --argjson midden_total "$mw_total" \
          '{success: true, entry_id: $entry_id, category: $category, midden_total: $midden_total}')"
      else
        release_lock 2>/dev/null || true
        json_ok "{\"success\":true,\"warning\":\"jq_processing_failed\",\"entry_id\":null}"
      fi
    else
      # Lock failed — graceful degradation, try without lock
      echo "Warning: Midden write completed without lock -- if another write happened at the same time, one entry may be missing." >&2
      mw_updated_midden=$(jq --argjson entry "$mw_new_entry" '
        .entries += [$entry] |
        .entry_count = (.entries | length)
      ' "$mw_midden_file" 2>/dev/null)

      if [[ -n "$mw_updated_midden" ]]; then
        _midden_try_write "$mw_updated_midden" "$mw_midden_file"
        json_ok "$(jq -n --arg entry_id "$mw_entry_id" --arg category "$mw_category" \
          '{success: true, entry_id: $entry_id, category: $category, warning: "lock_unavailable"}')"
      else
        json_ok "{\"success\":true,\"warning\":\"jq_processing_failed\",\"entry_id\":null}"
      fi
    fi
}

_midden_recent_failures() {
    # Extract recent failure entries from midden.json
    # Usage: midden-recent-failures [limit]
    # Returns: JSON with count and failures array

    limit="${1:-5}"
    midden_file="$DATA_DIR/midden/midden.json"

    if [[ ! -f "$midden_file" ]]; then
      echo '{"count":0,"failures":[]}'
      return 0
    fi

    # Extract failures from .entries[], sort by timestamp descending, limit results
    result=$(jq --argjson limit "$limit" '{
      "count": ([.entries[]?] | length),
      "failures": ([.entries[]?] | sort_by(.timestamp) | reverse | .[:$limit] | [.[] | {timestamp, category, source, message}])
    }' "$midden_file" 2>/dev/null)

    if [[ -z "$result" ]]; then
      echo '{"count":0,"failures":[]}'
    else
      echo "$result"
    fi
    return 0
}

_midden_review() {
    # Review unacknowledged midden entries grouped by category
    # Usage: midden-review [--category <cat>] [--limit N] [--include-acknowledged]
    # Returns: JSON with unacknowledged_count, categories summary, and entries array

    mr_category=""
    mr_limit=20
    mr_include_ack=false

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --category)             mr_category="${2:-}"; shift 2 ;;
        --limit)                mr_limit="${2:-20}"; shift 2 ;;
        --include-acknowledged) mr_include_ack=true; shift ;;
        *) shift ;;
      esac
    done

    mr_midden_file="$DATA_DIR/midden/midden.json"

    if [[ ! -f "$mr_midden_file" ]]; then
      json_ok '{"unacknowledged_count":0,"categories":{},"entries":[]}'
      return 0
    fi

    # Build jq filter based on options
    mr_result=$(jq \
      --arg category "$mr_category" \
      --argjson limit "$mr_limit" \
      --argjson include_ack "$mr_include_ack" \
      '
      # Start with all entries
      [.entries // [] | .[] |
        # Filter acknowledged unless --include-acknowledged
        if $include_ack then . else select(.acknowledged != true) end |
        # Filter by category if specified
        if ($category | length) > 0 then select(.category == $category) else . end
      ] |
      # Sort by timestamp descending
      sort_by(.timestamp) | reverse |
      # Compute categories before limiting
      . as $all |
      # Apply limit
      ($all | .[:$limit]) as $limited |
      # Group $all by category for counts
      ($all | group_by(.category) | map({key: .[0].category, value: length}) | from_entries) as $cats |
      {
        unacknowledged_count: ($all | length),
        categories: $cats,
        entries: $limited
      }
      ' "$mr_midden_file" 2>/dev/null)

    if [[ -z "$mr_result" ]]; then
      json_ok '{"unacknowledged_count":0,"categories":{},"entries":[]}'
    else
      json_ok "$mr_result"
    fi
    return 0
}

_midden_ingest_errors() {
    # Ingest entries from errors.log into midden
    # Usage: midden-ingest-errors [--dry-run]
    # Returns: JSON with count of ingested entries
    # After ingestion, moves errors.log to errors.log.ingested

    mie_dry_run=false
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --dry-run) mie_dry_run=true; shift ;;
        *) shift ;;
      esac
    done

    mie_errors_file="$DATA_DIR/errors.log"

    # No errors.log → nothing to ingest
    if [[ ! -f "$mie_errors_file" ]]; then
      json_ok '{"ingested":0}'
      return 0
    fi

    # Empty file → nothing to ingest
    if [[ ! -s "$mie_errors_file" ]]; then
      json_ok '{"ingested":0}'
      return 0
    fi

    mie_count=0

    # Read line by line (avoid pipe-to-while subshell)
    while IFS= read -r mie_line; do
      # Skip blank lines
      [[ -z "$mie_line" ]] && continue

      # Parse timestamp from [YYYY-...Z] prefix
      mie_timestamp=""
      mie_message="$mie_line"
      if [[ "$mie_line" =~ ^\[([^\]]+)\]\ (.*) ]]; then
        mie_timestamp="${BASH_REMATCH[1]}"
        mie_message="${BASH_REMATCH[2]}"
      fi

      mie_count=$((mie_count + 1))

      if [[ "$mie_dry_run" == "false" ]]; then
        _midden_write "error_log" "$mie_message" "error-handler" >/dev/null 2>&1 || true
      fi
    done < "$mie_errors_file"

    # Move the file (not dry-run only)
    if [[ "$mie_dry_run" == "false" && "$mie_count" -gt 0 ]]; then
      mv "$mie_errors_file" "${mie_errors_file}.ingested"
    fi

    json_ok "{\"ingested\":$mie_count}"
    return 0
}

_midden_acknowledge() {
    # Acknowledge midden entries by id or by category
    # Usage: midden-acknowledge --id <entry_id> [--reason <reason>]
    #    OR: midden-acknowledge --category <cat> --reason <reason>
    # Returns: JSON with acknowledged=true, count, and reason

    ma_id=""
    ma_category=""
    ma_reason=""

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --id)       ma_id="${2:-}"; shift 2 ;;
        --category) ma_category="${2:-}"; shift 2 ;;
        --reason)   ma_reason="${2:-}"; shift 2 ;;
        *) shift ;;
      esac
    done

    # Validate: need either --id or --category
    if [[ -z "$ma_id" && -z "$ma_category" ]]; then
      json_err "$E_VALIDATION_FAILED" "midden-acknowledge requires --id or --category"
    fi

    ma_midden_file="$DATA_DIR/midden/midden.json"

    if [[ ! -f "$ma_midden_file" ]]; then
      json_err "$E_FILE_NOT_FOUND" "midden.json not found"
    fi

    ma_now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Acquire lock with trap-based cleanup
    acquire_lock "$ma_midden_file" || {
      json_err "$E_LOCK_FAILED" "Failed to acquire lock on midden.json"
    }
    trap 'release_lock 2>/dev/null || true' EXIT

    if [[ -n "$ma_id" ]]; then
      # Acknowledge single entry by id
      ma_exists=$(jq --arg id "$ma_id" '[.entries[]? | select(.id == $id)] | length > 0' "$ma_midden_file" 2>/dev/null || echo "false")
      if [[ "$ma_exists" != "true" ]]; then
        trap - EXIT
        release_lock 2>/dev/null || true
        json_err "$E_RESOURCE_NOT_FOUND" "Midden entry '$ma_id' not found"
      fi

      ma_updated=$(jq \
        --arg id "$ma_id" \
        --arg now "$ma_now" \
        --arg reason "$ma_reason" \
        '
        .entries = [.entries[] |
          if .id == $id then
            . + {acknowledged: true, acknowledged_at: $now, acknowledge_reason: $reason}
          else
            .
          end
        ]
        ' "$ma_midden_file" 2>/dev/null)

      ma_count=1
    else
      # Acknowledge all entries matching category
      ma_count=$(jq --arg cat "$ma_category" '[.entries[]? | select(.category == $cat and .acknowledged != true)] | length' "$ma_midden_file" 2>/dev/null || echo "0")

      ma_updated=$(jq \
        --arg cat "$ma_category" \
        --arg now "$ma_now" \
        --arg reason "$ma_reason" \
        '
        .entries = [.entries[] |
          if .category == $cat and .acknowledged != true then
            . + {acknowledged: true, acknowledged_at: $now, acknowledge_reason: $reason}
          else
            .
          end
        ]
        ' "$ma_midden_file" 2>/dev/null)
    fi

    if [[ -z "$ma_updated" ]]; then
      trap - EXIT
      release_lock 2>/dev/null || true
      json_err "$E_INTERNAL" "Failed to update midden.json"
    fi

    atomic_write "$ma_midden_file" "$ma_updated"

    trap - EXIT
    release_lock 2>/dev/null || true

    json_ok "$(jq -n --argjson count "$ma_count" --arg reason "$ma_reason" \
      '{acknowledged: true, count: $count, reason: $reason}')"
    return 0
}
