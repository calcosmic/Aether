# =============================================================================
# DEPRECATED — This script has been superseded by the Go binary (aether CLI).
# All functionality is now available via: aether <subcommand>
# Do NOT modify this file — it is retained for reference only.
# See: cmd/ (Go source) | Run: aether --help
# =============================================================================
#
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

    mw_midden_dir="$COLONY_DATA_DIR/midden"
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
    midden_file="$COLONY_DATA_DIR/midden/midden.json"

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

    mr_midden_file="$COLONY_DATA_DIR/midden/midden.json"

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

    mie_errors_file="$COLONY_DATA_DIR/errors.log"

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

_midden_search() {
    # Search midden entries by keyword match in message field
    # Usage: midden-search <query> [--category <cat>] [--source <src>] [--limit N] [--include-acknowledged]
    # Returns: JSON with query, match_count, and entries array

    ms_query=""
    ms_category=""
    ms_source=""
    ms_limit=10
    ms_include_ack=false

    # First positional arg is the query
    if [[ $# -gt 0 && "$1" != --* ]]; then
      ms_query="$1"
      shift
    fi

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --category)             ms_category="${2:-}"; shift 2 ;;
        --source)               ms_source="${2:-}"; shift 2 ;;
        --limit)                ms_limit="${2:-10}"; shift 2 ;;
        --include-acknowledged) ms_include_ack=true; shift ;;
        *) shift ;;
      esac
    done

    ms_midden_file="$COLONY_DATA_DIR/midden/midden.json"

    if [[ ! -f "$ms_midden_file" ]]; then
      json_ok "{\"query\":$(printf '%s' "$ms_query" | jq -Rs .),\"match_count\":0,\"entries\":[]}"
      return 0
    fi

    ms_result=$(jq \
      --arg query "$ms_query" \
      --arg category "$ms_category" \
      --arg source "$ms_source" \
      --argjson limit "$ms_limit" \
      --argjson include_ack "$ms_include_ack" \
      '
      [.entries // [] | .[] |
        # Filter acknowledged unless --include-acknowledged
        if $include_ack then . else select(.acknowledged != true) end |
        # Filter by category if specified
        if ($category | length) > 0 then select(.category == $category) else . end |
        # Filter by source if specified
        if ($source | length) > 0 then select(.source == $source) else . end |
        # Filter by keyword match in message (case-insensitive)
        if ($query | length) > 0 then
          select(.message | ascii_downcase | contains($query | ascii_downcase))
        else
          .
        end
      ] |
      sort_by(.timestamp) | reverse |
      . as $all |
      {
        query: $query,
        match_count: ($all | length),
        entries: ($all | .[:$limit])
      }
      ' "$ms_midden_file" 2>/dev/null)

    if [[ -z "$ms_result" ]]; then
      json_ok "{\"query\":$(printf '%s' "$ms_query" | jq -Rs .),\"match_count\":0,\"entries\":[]}"
    else
      json_ok "$ms_result"
    fi
    return 0
}

_midden_tag() {
    # Add or remove a tag from a midden entry's tags array
    # Usage: midden-tag --id <entry_id> --tag <tag_name>
    #    OR: midden-tag --id <entry_id> --untag <tag_name>
    # Returns: JSON with entry_id, tags array, and action

    mt_id=""
    mt_tag=""
    mt_untag=""

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --id)    mt_id="${2:-}"; shift 2 ;;
        --tag)   mt_tag="${2:-}"; shift 2 ;;
        --untag) mt_untag="${2:-}"; shift 2 ;;
        *) shift ;;
      esac
    done

    # Validate: need --id
    if [[ -z "$mt_id" ]]; then
      json_err "$E_VALIDATION_FAILED" "midden-tag requires --id"
    fi

    # Validate: need --tag or --untag (but not both)
    if [[ -z "$mt_tag" && -z "$mt_untag" ]]; then
      json_err "$E_VALIDATION_FAILED" "midden-tag requires --tag or --untag"
    fi

    if [[ -n "$mt_tag" && -n "$mt_untag" ]]; then
      json_err "$E_VALIDATION_FAILED" "midden-tag requires --tag or --untag, not both"
    fi

    mt_midden_file="$COLONY_DATA_DIR/midden/midden.json"

    if [[ ! -f "$mt_midden_file" ]]; then
      json_err "$E_FILE_NOT_FOUND" "midden.json not found"
    fi

    # Check entry exists
    mt_exists=$(jq --arg id "$mt_id" '[.entries[]? | select(.id == $id)] | length > 0' "$mt_midden_file" 2>/dev/null || echo "false")
    if [[ "$mt_exists" != "true" ]]; then
      json_err "$E_RESOURCE_NOT_FOUND" "Midden entry '$mt_id' not found"
    fi

    # Acquire lock with trap-based cleanup
    acquire_lock "$mt_midden_file" || {
      json_err "$E_LOCK_FAILED" "Failed to acquire lock on midden.json"
    }
    trap 'release_lock 2>/dev/null || true' EXIT

    if [[ -n "$mt_tag" ]]; then
      # Add tag — create tags array if absent, append if tag not already present
      mt_updated=$(jq \
        --arg id "$mt_id" \
        --arg tag "$mt_tag" \
        '
        .entries = [.entries[] |
          if .id == $id then
            . + {tags: ((.tags // []) | if contains([$tag]) then . else . + [$tag] end)}
          else
            .
          end
        ]
        ' "$mt_midden_file" 2>/dev/null)
      mt_action="added"
    else
      # Remove tag — remove from tags array if present
      mt_updated=$(jq \
        --arg id "$mt_id" \
        --arg tag "$mt_untag" \
        '
        .entries = [.entries[] |
          if .id == $id then
            . + {tags: ((.tags // []) | map(select(. != $tag)))}
          else
            .
          end
        ]
        ' "$mt_midden_file" 2>/dev/null)
      mt_action="removed"
      mt_tag="$mt_untag"
    fi

    if [[ -z "$mt_updated" ]]; then
      trap - EXIT
      release_lock 2>/dev/null || true
      json_err "$E_INTERNAL" "Failed to update midden.json"
    fi

    atomic_write "$mt_midden_file" "$mt_updated"

    trap - EXIT
    release_lock 2>/dev/null || true

    # Read back the updated tags for the entry
    mt_tags=$(jq --arg id "$mt_id" '[.entries[]? | select(.id == $id) | .tags // []] | .[0] // []' "$mt_midden_file" 2>/dev/null || echo "[]")

    json_ok "$(jq -n \
      --arg entry_id "$mt_id" \
      --argjson tags "$mt_tags" \
      --arg action "$mt_action" \
      '{entry_id: $entry_id, tags: $tags, action: $action}')"
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

    ma_midden_file="$COLONY_DATA_DIR/midden/midden.json"

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

# ============================================================================
# Cross-Branch Midden Collection (Phase 41)
# ============================================================================

_midden_collect() {
    # Collect midden entries from a merged branch worktree into main's midden
    # Usage: midden-collect --branch <name> --merge-sha <sha> [--dry-run]
    # Returns: JSON with collection status and counts
    #
    # Dual-layer idempotency:
    #   Layer 1: Merge fingerprint in collected-merges.json (fast path)
    #   Layer 2: Per-entry ID dedup (safety net)

    mc_branch=""
    mc_merge_sha=""
    mc_dry_run=false

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --branch)     mc_branch="${2:-}"; shift 2 ;;
        --merge-sha)  mc_merge_sha="${2:-}"; shift 2 ;;
        --dry-run)    mc_dry_run=true; shift ;;
        *) shift ;;
      esac
    done

    # Validate required args
    if [[ -z "$mc_branch" ]]; then
      json_err "$E_VALIDATION_FAILED" "midden-collect requires --branch"
    fi
    if [[ -z "$mc_merge_sha" ]]; then
      json_err "$E_VALIDATION_FAILED" "midden-collect requires --merge-sha"
    fi

    # Resolve worktree midden path
    mc_worktree_midden=""
    mc_candidate="$AETHER_ROOT/.aether/worktrees/$mc_branch/.aether/data/midden/midden.json"

    if [[ -f "$mc_candidate" ]]; then
      mc_worktree_midden="$mc_candidate"
    else
      # Fallback: check git worktree list
      mc_wt_path=$(git -C "$AETHER_ROOT" worktree list --porcelain 2>/dev/null | grep -F "worktree" | head -1 | cut -d' ' -f2 || true)
      if [[ -n "$mc_wt_path" && -f "$mc_wt_path/.aether/data/midden/midden.json" ]]; then
        mc_worktree_midden="$mc_wt_path/.aether/data/midden/midden.json"
      fi
    fi

    if [[ -z "$mc_worktree_midden" || ! -f "$mc_worktree_midden" ]]; then
      json_ok "$(jq -n --arg branch "$mc_branch" \
        '{status: "worktree_not_found", entries_collected: 0, branch: $branch}')"
      return 0
    fi

    # Read branch midden.json
    mc_branch_data=$(cat "$mc_worktree_midden" 2>/dev/null || echo "")

    if [[ -z "$mc_branch_data" ]]; then
      json_ok '{"status":"empty_branch_midden","entries_collected":0}'
      return 0
    fi

    # Validate branch midden.json is valid JSON
    if ! echo "$mc_branch_data" | jq empty 2>/dev/null; then
      json_err "$E_INTERNAL" "Branch midden.json is corrupt"
    fi

    # Check for entries
    mc_branch_count=$(echo "$mc_branch_data" | jq '[.entries[]?] | length' 2>/dev/null || echo "0")
    if [[ "$mc_branch_count" -eq 0 ]]; then
      json_ok '{"status":"empty_branch_midden","entries_collected":0}'
      return 0
    fi

    mc_midden_dir="$COLONY_DATA_DIR/midden"
    mc_midden_file="$mc_midden_dir/midden.json"
    mc_merges_file="$mc_midden_dir/collected-merges.json"
    mc_timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    mkdir -p "$mc_midden_dir"

    # Initialize midden.json if missing
    if [[ ! -f "$mc_midden_file" ]]; then
      printf '%s\n' '{"version":"1.0.0","entries":[]}' > "$mc_midden_file"
    fi

    # Initialize collected-merges.json if missing
    if [[ ! -f "$mc_merges_file" ]]; then
      printf '%s\n' '{"version":"1.0.0","merges":[]}' > "$mc_merges_file"
    fi

    # LAYER 1: Check merge fingerprint
    mc_already=$(jq --arg sha "$mc_merge_sha" --arg branch "$mc_branch" \
      '[.merges[]? | select(.merge_commit == $sha and .branch_name == $branch)] | length > 0' \
      "$mc_merges_file" 2>/dev/null || echo "false")

    if [[ "$mc_already" == "true" ]]; then
      json_ok "$(jq -n --arg sha "$mc_merge_sha" --arg branch "$mc_branch" \
        '{status: "already_collected", merge_commit: $sha, branch: $branch, entries_collected: 0}')"
      return 0
    fi

    if [[ "$mc_dry_run" == "true" ]]; then
      json_ok "$(jq -n --arg branch "$mc_branch" --arg sha "$mc_merge_sha" --argjson count "$mc_branch_count" \
        '{status: "dry_run", branch: $branch, merge_commit: $sha, entries_would_collect: $count}')"
      return 0
    fi

    # LAYER 2: Per-entry ID dedup — get existing IDs from main's midden
    mc_existing_ids=$(jq -r '[.entries[]?.id] | map(select(. != null))' "$mc_midden_file" 2>/dev/null || echo "[]")

    # Filter branch entries: exclude those with IDs already in main, then enrich
    mc_new_entries=$(jq --argjson existing_ids "$mc_existing_ids" \
      --arg branch "$mc_branch" \
      --arg ts "$mc_timestamp" \
      --arg sha "$mc_merge_sha" \
      '
      [.entries[]?] |
      map(select([.id] | inside($existing_ids) | not)) |
      map(. + {
        collected_from: $branch,
        collected_at: $ts,
        merge_commit: $sha,
        original_entry_id: .id
      })
      ' "$mc_worktree_midden" 2>/dev/null || echo "[]")

    mc_new_count=$(echo "$mc_new_entries" | jq 'length' 2>/dev/null || echo "0")
    mc_skipped=$((mc_branch_count - mc_new_count))

    if [[ "$mc_new_count" -gt 0 ]]; then
      # Append enriched entries to main's midden
      acquire_lock "$mc_midden_file" || {
        json_err "$E_LOCK_FAILED" "Failed to acquire lock on midden.json"
      }
      trap 'release_lock 2>/dev/null || true' EXIT

      mc_updated=$(jq --argjson new_entries "$mc_new_entries" \
        '.entries += $new_entries' "$mc_midden_file" 2>/dev/null)

      if [[ -z "$mc_updated" ]]; then
        trap - EXIT
        release_lock 2>/dev/null || true
        json_err "$E_INTERNAL" "Failed to update midden.json with collected entries"
      fi

      atomic_write "$mc_midden_file" "$mc_updated"

      trap - EXIT
      release_lock 2>/dev/null || true
    fi

    # Write fingerprint to collected-merges.json
    mc_fingerprint=$(printf '%s|%s|%d' "$mc_branch" "$mc_merge_sha" "$mc_new_count" | shasum -a 256 | cut -d' ' -f1)

    acquire_lock "$mc_merges_file" || {
      # Non-fatal — entries were collected but fingerprint may be missing
      json_ok "$(jq -n --arg branch "$mc_branch" --arg sha "$mc_merge_sha" \
        --argjson collected "$mc_new_count" --argjson skipped "$mc_skipped" \
        '{status: "collected", entries_collected: $collected, entries_skipped_dup: $skipped, branch: $branch, merge_commit: $sha, warning: "fingerprint_write_failed"}')"
      return 0
    }
    trap 'release_lock 2>/dev/null || true' EXIT

    mc_merges_updated=$(jq --arg sha "$mc_merge_sha" --arg branch "$mc_branch" \
      --arg ts "$mc_timestamp" --argjson collected "$mc_new_count" \
      --argjson skipped "$mc_skipped" --arg fp "$mc_fingerprint" \
      '.merges += [{
        merge_commit: $sha,
        branch_name: $branch,
        collected_at: $ts,
        entries_collected: $collected,
        entries_skipped_dup: $skipped,
        fingerprint: $fp
      }]' "$mc_merges_file" 2>/dev/null)

    if [[ -n "$mc_merges_updated" ]]; then
      atomic_write "$mc_merges_file" "$mc_merges_updated"
    fi

    trap - EXIT
    release_lock 2>/dev/null || true

    json_ok "$(jq -n --arg branch "$mc_branch" --arg sha "$mc_merge_sha" \
      --argjson collected "$mc_new_count" --argjson skipped "$mc_skipped" \
      '{status: "collected", entries_collected: $collected, entries_skipped_dup: $skipped, branch: $branch, merge_commit: $sha}')"
    return 0
}

_midden_handle_revert() {
    # Tag entries from a reverted merge commit (not delete)
    # Usage: midden-handle-revert --sha <revert-sha>
    #    OR: midden-handle-revert --revert-commit <sha> --original-merge <sha>
    # Returns: JSON with revert status and tagged count

    mhr_revert_sha=""
    mhr_original_merge=""

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --sha)            mhr_revert_sha="${2:-}"; shift 2 ;;
        --revert-commit)  mhr_revert_sha="${2:-}"; shift 2 ;;
        --original-merge) mhr_original_merge="${2:-}"; shift 2 ;;
        *) shift ;;
      esac
    done

    if [[ -z "$mhr_revert_sha" ]]; then
      json_err "$E_VALIDATION_FAILED" "midden-handle-revert requires --sha or --revert-commit"
    fi

    mc_midden_dir="$COLONY_DATA_DIR/midden"
    mc_merges_file="$mc_midden_dir/collected-merges.json"

    # If no original-merge given, try to find it from collected-merges by parsing commit message
    if [[ -z "$mhr_original_merge" ]]; then
      # Try git log to find the reverted merge
      mhr_original_merge=$(git -C "$AETHER_ROOT" log -1 --format="%b" "$mhr_revert_sha" 2>/dev/null \
        | grep -oE '[0-9a-f]{7,40}' | head -1 || true)
    fi

    if [[ -z "$mhr_original_merge" ]]; then
      json_ok "$(jq -n --arg sha "$mhr_revert_sha" \
        '{status: "original_merge_not_resolved", revert_commit: $sha, entries_tagged: 0}')"
      return 0
    fi

    # Check collected-merges.json exists
    if [[ ! -f "$mc_merges_file" ]]; then
      json_ok "$(jq -n --arg sha "$mhr_revert_sha" --arg merge "$mhr_original_merge" \
        '{status: "merge_not_found", revert_commit: $sha, original_merge: $merge, entries_tagged: 0}')"
      return 0
    fi

    # Check if the original merge exists in collected-merges
    mhr_found=$(jq --arg merge "$mhr_original_merge" \
      '[.merges[]? | select(.merge_commit == $merge)] | length > 0' \
      "$mc_merges_file" 2>/dev/null || echo "false")

    if [[ "$mhr_found" != "true" ]]; then
      json_ok "$(jq -n --arg sha "$mhr_revert_sha" --arg merge "$mhr_original_merge" \
        '{status: "merge_not_found", revert_commit: $sha, original_merge: $merge, entries_tagged: 0}')"
      return 0
    fi

    mhr_midden_file="$mc_midden_dir/midden.json"
    mhr_timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    mhr_tagged=0

    # Tag entries in main's midden.json
    if [[ -f "$mhr_midden_file" ]]; then
      acquire_lock "$mhr_midden_file" || {
        json_err "$E_LOCK_FAILED" "Failed to acquire lock on midden.json"
      }
      trap 'release_lock 2>/dev/null || true' EXIT

      mhr_updated=$(jq --arg revert_sha "$mhr_revert_sha" --arg merge_sha "$mhr_original_merge" \
        '
        .entries = [.entries[] |
          if .merge_commit == $merge_sha then
            . + {
              tags: ((.tags // []) + ["reverted:" + $revert_sha]) | unique,
              reviewed: false
            }
          else
            .
          end
        ]
        ' "$mhr_midden_file" 2>/dev/null)

      if [[ -n "$mhr_updated" ]]; then
        atomic_write "$mhr_midden_file" "$mhr_updated"
        mhr_tagged=$(echo "$mhr_updated" | jq --arg merge_sha "$mhr_original_merge" \
          '[.entries[] | select(.merge_commit == $merge_sha)] | length' 2>/dev/null || echo "0")
      fi

      trap - EXIT
      release_lock 2>/dev/null || true
    fi

    # Update collected-merges.json: mark merge as reverted
    acquire_lock "$mc_merges_file" || true
    trap 'release_lock 2>/dev/null || true' EXIT

    mhr_merges_updated=$(jq --arg revert_sha "$mhr_revert_sha" \
      --arg merge_sha "$mhr_original_merge" --arg ts "$mhr_timestamp" \
      '
      .merges = [.merges[] |
        if .merge_commit == $merge_sha then
          . + {reverted_by: $revert_sha, reverted_at: $ts, status: "reverted"}
        else
          .
        end
      ]
      ' "$mc_merges_file" 2>/dev/null)

    if [[ -n "$mhr_merges_updated" ]]; then
      atomic_write "$mc_merges_file" "$mhr_merges_updated"
    fi

    trap - EXIT
    release_lock 2>/dev/null || true

    json_ok "$(jq -n --arg sha "$mhr_revert_sha" --arg merge "$mhr_original_merge" \
      --argjson tagged "$mhr_tagged" \
      '{revert_commit: $sha, original_merge: $merge, entries_tagged: $tagged, entries_deleted: 0}')"
    return 0
}

_midden_cross_pr_analysis() {
    # Detect cross-PR failure patterns and auto-emit REDIRECT for systemic issues
    # Usage: midden-cross-pr-analysis [--category <cat>] [--window <days>]
    # Returns: JSON with category analysis, scores, classifications

    mca_category=""
    mca_window=14

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --category) mca_category="${2:-}"; shift 2 ;;
        --window)   mca_window="${2:-14}"; shift 2 ;;
        *) shift ;;
      esac
    done

    mca_midden_file="$COLONY_DATA_DIR/midden/midden.json"

    if [[ ! -f "$mca_midden_file" ]]; then
      json_ok "$(jq -n --argjson window "$mca_window" \
        '{analysis_timestamp: "now", window_days: $window, total_entries_scanned: 0, categories: {}, systemic_categories: []}')"
      return 0
    fi

    mca_timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Calculate cutoff timestamp: NOW - window_days
    mca_cutoff=$(date -u -v-"${mca_window}d" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || \
      python3 -c "import datetime; print((datetime.datetime.utcnow() - datetime.timedelta(days=$mca_window)).strftime('%Y-%m-%dT%H:%M:%SZ'))" 2>/dev/null || \
      date -u -d "$mca_window days ago" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo "")

    mca_result=$(jq --arg cutoff "$mca_cutoff" \
      --arg category "$mca_category" --argjson window "$mca_window" \
      '
      # Collect cross-branch entries within window, excluding reverted
      [.entries // [] | .[] |
        select(.collected_from != null) |
        select(.reviewed != true) |
        select(.tags // [] | map(startswith("reverted:")) | any | not) |
        select(if ($cutoff | length) > 0 then .timestamp >= $cutoff else true end) |
        if ($category | length) > 0 then select(.category == $category) else . end
      ] |
      . as $entries |

      # Group by category
      ($entries | group_by(.category) | map({key: .[0].category, value: .}) | from_entries) as $by_cat |

      # Compute metrics per category
      ($by_cat | to_entries | map({
        key: .key,
        value: {
          total_entries: (.value | length),
          unique_prs: ([.value[].collected_from] | unique | length),
          entries_per_pr: (.value | group_by(.collected_from) | map({key: .[0].collected_from, value: length}) | from_entries),
          cross_pr_score: (
            (([.value[].collected_from] | unique | length) / 5) * 0.6 +
            ((.value | length) / 10) * 0.4
          ),
          classification: (
            if (([.value[].collected_from] | unique | length) >= 3) and ((.value | length) >= 5) then
              "cross-pr-critical"
            elif (([.value[].collected_from] | unique | length) >= 2) and ((.value | length) >= 3) then
              "cross-pr-systemic"
            else
              "single-pr"
            end
          ),
          auto_redirect_emitted: false
        }
      }) | from_entries) as $analysis |

      # Collect systemic categories
      [$analysis | to_entries[] | select(.value.classification == "cross-pr-systemic" or .value.classification == "cross-pr-critical") | .key] as $systemic |

      {
        total_entries_scanned: ($entries | length),
        categories: $analysis,
        systemic_categories: $systemic
      }
      ' "$mca_midden_file" 2>/dev/null)

    if [[ -z "$mca_result" ]]; then
      json_ok "$(jq -n --argjson window "$mca_window" \
        '{analysis_timestamp: "now", window_days: $window, total_entries_scanned: 0, categories: {}, systemic_categories: []}')"
      return 0
    fi

    # Auto-emit REDIRECT for systemic/critical categories
    mca_systemic=$(echo "$mca_result" | jq -r '.systemic_categories // [] | .[]' 2>/dev/null || true)
    for mca_cat in $mca_systemic; do
      mca_cat_data=$(echo "$mca_result" | jq --arg cat "$mca_cat" '.categories[$cat]' 2>/dev/null || echo "{}")
      mca_unique_prs=$(echo "$mca_cat_data" | jq -r '.unique_prs // 0' 2>/dev/null || echo "0")
      mca_total=$(echo "$mca_cat_data" | jq -r '.total_entries // 0' 2>/dev/null || echo "0")
      mca_score=$(echo "$mca_cat_data" | jq -r '.cross_pr_score // 0' 2>/dev/null || echo "0")

      # Compute strength: 0.5 + (score * 0.5), capped at 1.0
      mca_strength=$(jq -n --argjson score "$mca_score" '0.5 + ($score * 0.5) | if . > 1.0 then 1.0 else . * 100 | round / 100 end' 2>/dev/null || echo "0.7")

      # NON-BLOCKING: emit REDIRECT, swallow all output
      bash "$AETHER_ROOT/.aether/aether-utils.sh" pheromone-write REDIRECT \
        "[cross-pr-pattern] $mca_cat failures across $mca_unique_prs PRs in $mca_window days ($mca_total entries)" \
        --strength "$mca_strength" \
        --source "auto:cross-pr" \
        --reason "Auto-emitted: cross-PR systemic failure pattern detected" \
        --ttl "30d" >/dev/null 2>&1 || true

      # Mark as emitted in the result
      mca_result=$(echo "$mca_result" | jq --arg cat "$mca_cat" '.categories[$cat].auto_redirect_emitted = true' 2>/dev/null || echo "$mca_result")
    done

    json_ok "$(echo "$mca_result" | jq --arg ts "$mca_timestamp" --argjson window "$mca_window" \
      '. + {analysis_timestamp: $ts, window_days: $window}' 2>/dev/null || echo "$mca_result")"
    return 0
}

_midden_prune() {
    # Retention cleanup for collected merges and reverted entries
    # Usage: midden-prune --stale-merges
    #    OR: midden-prune --reverted --age <days>
    # Returns: JSON with prune counts

    mp_stale_merges=false
    mp_reverted=false
    mp_age=30

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --stale-merges) mp_stale_merges=true; shift ;;
        --reverted)     mp_reverted=true; shift ;;
        --age)          mp_age="${2:-30}"; shift 2 ;;
        *) shift ;;
      esac
    done

    mp_midden_dir="$COLONY_DATA_DIR/midden"
    mp_merges_file="$mp_midden_dir/collected-merges.json"
    mp_midden_file="$mp_midden_dir/midden.json"
    mp_pruned_merges=0
    mp_pruned_reverted=0

    if [[ "$mp_stale_merges" == "true" ]]; then
      if [[ -f "$mp_merges_file" ]]; then
        mp_cutoff=$(date -u -v-"90d" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || \
          python3 -c "import datetime; print((datetime.datetime.utcnow() - datetime.timedelta(days=90)).strftime('%Y-%m-%dT%H:%M:%SZ'))" 2>/dev/null || \
          date -u -d "90 days ago" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo "")

        if [[ -n "$mp_cutoff" ]]; then
          acquire_lock "$mp_merges_file" || true
          trap 'release_lock 2>/dev/null || true' EXIT

          mp_before=$(jq '.merges | length' "$mp_merges_file" 2>/dev/null || echo "0")
          mp_before=${mp_before:-0}
          mp_merges_updated=$(jq --arg cutoff "$mp_cutoff" \
            '.merges = [.merges // [] | .[] | select(.collected_at >= $cutoff)]' \
            "$mp_merges_file" 2>/dev/null)
          mp_after=$(echo "$mp_merges_updated" | jq '.merges | length' 2>/dev/null || echo "0")
          mp_after=${mp_after:-0}

          if [[ -n "$mp_merges_updated" ]]; then
            atomic_write "$mp_merges_file" "$mp_merges_updated"
            mp_pruned_merges=$((mp_before - mp_after))
          fi

          trap - EXIT
          release_lock 2>/dev/null || true
        fi
      fi
    fi

    if [[ "$mp_reverted" == "true" ]]; then
      if [[ -f "$mp_midden_file" && -f "$mp_merges_file" ]]; then
        mp_cutoff=$(date -u -v-"${mp_age}d" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || \
          python3 -c "import datetime; print((datetime.datetime.utcnow() - datetime.timedelta(days=$mp_age)).strftime('%Y-%m-%dT%H:%M:%SZ'))" 2>/dev/null || \
          date -u -d "$mp_age days ago" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo "")

        if [[ -n "$mp_cutoff" ]]; then
          # Find revert timestamps from collected-merges.json
          mp_revert_map=$(jq -r '[.merges // [] | .[] | select(.status == "reverted")] | map({merge_commit: .merge_commit, reverted_at: .reverted_at})' "$mp_merges_file" 2>/dev/null || echo "[]")

          acquire_lock "$mp_midden_file" || true
          trap 'release_lock 2>/dev/null || true' EXIT

          mp_now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

          # Acknowledge reverted entries older than threshold
          mp_updated=$(jq --argjson revert_map "$mp_revert_map" \
            --arg cutoff "$mp_cutoff" --arg now "$mp_now" --argjson age "$mp_age" \
            '
            .entries = [.entries // [] | .[] |
              if .tags // [] | map(startswith("reverted:")) | any then
                # Find the revert timestamp for this entry
                ($revert_map | map(select(.merge_commit == .merge_commit)) | first // {}) as $merge_info |
                if ($merge_info.reverted_at // "") != "" and ($merge_info.reverted_at < $cutoff) and .acknowledged != true then
                  . + {
                    acknowledged: true,
                    acknowledged_at: $now,
                    acknowledge_reason: ("auto-pruned: reverted entry older than " + ($age | tostring) + " days")
                  }
                else .
                end
              else .
              end
            ]
            ' "$mp_midden_file" 2>/dev/null)

          if [[ -n "$mp_updated" ]]; then
            mp_before=$(jq '[.entries // [] | .[] | select(.tags // [] | map(startswith("reverted:")) | any and .acknowledged != true)] | length' "$mp_midden_file" 2>/dev/null || echo "0")
            mp_before=${mp_before:-0}
            mp_after=$(jq '[.entries // [] | .[] | select(.tags // [] | map(startswith("reverted:")) | any and .acknowledged != true)] | length' <<< "$mp_updated" 2>/dev/null || echo "0")
            mp_after=${mp_after:-0}
            atomic_write "$mp_midden_file" "$mp_updated"
            mp_pruned_reverted=$((mp_before - mp_after))
          fi

          trap - EXIT
          release_lock 2>/dev/null || true
        fi
      fi
    fi

    json_ok "$(jq -n --argjson pruned_merges "$mp_pruned_merges" --argjson pruned_reverted "$mp_pruned_reverted" \
      '{pruned_merges: $pruned_merges, pruned_reverted: $pruned_reverted}')"
    return 0
}
