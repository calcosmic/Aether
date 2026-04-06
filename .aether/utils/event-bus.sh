#!/bin/bash
# Event bus utility functions for the Aether Structural Learning Stack
# Provides: _event_publish, _event_subscribe, _event_cleanup, _event_replay
#
# These functions are sourced by aether-utils.sh at startup.
# All shared infrastructure (json_ok, json_err, json_warn, atomic_write, acquire_lock,
# release_lock, feature_enabled, LOCK_DIR, DATA_DIR, SCRIPT_DIR, error constants) is available.

# Default TTL for events in days
_EVENT_BUS_DEFAULT_TTL=30
_EVENT_BUS_DEFAULT_LIMIT=50

# ============================================================================
# _event_publish
# Publish an event to the JSONL event bus.
# Usage: event-publish --topic <topic> --payload <json> [--source <src>] [--ttl <days>]
# ============================================================================
_event_publish() {
    local ep_topic=""
    local ep_payload=""
    local ep_source="system"
    local ep_ttl="$_EVENT_BUS_DEFAULT_TTL"

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --topic)
                ep_topic="${2:-}"
                shift 2
                ;;
            --payload)
                ep_payload="${2:-}"
                shift 2
                ;;
            --source)
                ep_source="${2:-system}"
                shift 2
                ;;
            --ttl)
                ep_ttl="${2:-$_EVENT_BUS_DEFAULT_TTL}"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done

    [[ -z "$ep_topic" ]] && json_err "$E_VALIDATION_FAILED" "event-publish requires --topic"
    [[ -z "$ep_payload" ]] && json_err "$E_VALIDATION_FAILED" "event-publish requires --payload"

    # Validate payload is valid JSON
    echo "$ep_payload" | jq empty 2>/dev/null \
        || json_err "$E_JSON_INVALID" "event-publish --payload must be valid JSON"

    mkdir -p "$COLONY_DATA_DIR"
    local bus_file="$COLONY_DATA_DIR/event-bus.jsonl"

    # Generate unique ID and timestamps
    local ep_id
    ep_id="evt_$(date +%s)_$(head -c 2 /dev/urandom | od -An -tx1 | tr -d ' \n')"
    local ep_ts
    ep_ts=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    local ep_expires
    ep_expires=$(date -u -v+"${ep_ttl}"d +%Y-%m-%dT%H:%M:%SZ 2>/dev/null \
        || date -u -d "+${ep_ttl} days" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null \
        || echo "2099-01-01T00:00:00Z")

    # Build event JSON line
    local ep_line
    ep_line=$(jq -nc \
        --arg id "$ep_id" \
        --arg topic "$ep_topic" \
        --argjson payload "$ep_payload" \
        --arg source "$ep_source" \
        --arg ts "$ep_ts" \
        --argjson ttl_days "$ep_ttl" \
        --arg expires_at "$ep_expires" \
        '{id:$id,topic:$topic,payload:$payload,source:$source,timestamp:$ts,ttl_days:$ttl_days,expires_at:$expires_at}')

    # Acquire lock for safe concurrent append
    acquire_lock "event-bus" 5 2>/dev/null \
        || json_err "$E_LOCK_FAILED" "event-publish: failed to acquire lock"
    trap 'release_lock "event-bus" 2>/dev/null || true' EXIT

    echo "$ep_line" >> "$bus_file"

    release_lock "event-bus" 2>/dev/null || true

    json_ok "$(jq -nc \
        --arg event_id "$ep_id" \
        --arg topic "$ep_topic" \
        --argjson ttl_days "$ep_ttl" \
        '{event_id:$event_id,topic:$topic,ttl_days:$ttl_days}')"
}

# ============================================================================
# _event_subscribe
# Read events matching a topic pattern from the JSONL bus.
# Usage: event-subscribe --topic <pattern> [--since <ISO-8601>] [--limit <N>]
# Pattern supports exact match or prefix with trailing '*' (e.g., "learning.*")
# ============================================================================
_event_subscribe() {
    local es_topic=""
    local es_since=""
    local es_limit="$_EVENT_BUS_DEFAULT_LIMIT"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --topic)
                es_topic="${2:-}"
                shift 2
                ;;
            --since)
                es_since="${2:-}"
                shift 2
                ;;
            --limit)
                es_limit="${2:-$_EVENT_BUS_DEFAULT_LIMIT}"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done

    [[ -z "$es_topic" ]] && json_err "$E_VALIDATION_FAILED" "event-subscribe requires --topic"

    local bus_file="$COLONY_DATA_DIR/event-bus.jsonl"
    local now_ts
    now_ts=$(date -u +%Y-%m-%dT%H:%M:%SZ)

    # If bus file does not exist, return empty result
    if [[ ! -f "$bus_file" ]]; then
        json_ok "$(jq -nc \
            --arg pattern "$es_topic" \
            '{events:[],count:0,topic_pattern:$pattern}')"
        return 0
    fi

    # Determine if pattern is prefix match (ends with *) or exact match
    local es_jq_filter
    if [[ "$es_topic" == *"*" ]]; then
        local es_prefix="${es_topic%\*}"
        es_jq_filter="startswith(\"$es_prefix\")"
    else
        es_jq_filter=". == \"$es_topic\""
    fi

    # Build jq filter: topic match + not expired + since filter
    local es_jq_expr
    es_jq_expr=". | select(.topic | $es_jq_filter) | select(.expires_at > \"$now_ts\")"
    if [[ -n "$es_since" ]]; then
        es_jq_expr="$es_jq_expr | select(.timestamp >= \"$es_since\")"
    fi

    local es_events
    es_events=$(jq -sc \
        --argjson limit "$es_limit" \
        "[.[] | $es_jq_expr] | .[:(\$limit)]" \
        "$bus_file" 2>/dev/null || echo "[]")

    local es_count
    es_count=$(echo "$es_events" | jq 'length')

    json_ok "$(jq -nc \
        --argjson events "$es_events" \
        --argjson count "$es_count" \
        --arg topic_pattern "$es_topic" \
        '{events:$events,count:$count,topic_pattern:$topic_pattern}')"
}

# ============================================================================
# _event_cleanup
# Remove expired events from the JSONL bus.
# Usage: event-cleanup [--dry-run]
# ============================================================================
_event_cleanup() {
    local ec_dry_run="false"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dry-run)
                ec_dry_run="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done

    local bus_file="$COLONY_DATA_DIR/event-bus.jsonl"
    local now_ts
    now_ts=$(date -u +%Y-%m-%dT%H:%M:%SZ)

    # If bus file does not exist, nothing to clean
    if [[ ! -f "$bus_file" ]]; then
        json_ok "$(jq -nc \
            --argjson dry_run "$ec_dry_run" \
            '{removed:0,remaining:0,dry_run:$dry_run}')"
        return 0
    fi

    local ec_total
    ec_total=$(wc -l < "$bus_file" | tr -d ' ')

    local ec_kept
    ec_kept=$(jq -c "select(.expires_at > \"$now_ts\")" "$bus_file" 2>/dev/null || true)

    local ec_kept_count=0
    [[ -n "$ec_kept" ]] && ec_kept_count=$(echo "$ec_kept" | wc -l | tr -d ' ')

    local ec_removed=$(( ec_total - ec_kept_count ))

    if [[ "$ec_dry_run" == "false" ]]; then
        # Acquire lock for safe atomic rewrite
        acquire_lock "event-bus" 5 2>/dev/null \
            || json_err "$E_LOCK_FAILED" "event-cleanup: failed to acquire lock"
        trap 'release_lock "event-bus" 2>/dev/null || true' EXIT

        if [[ -n "$ec_kept" ]]; then
            atomic_write "$bus_file" "$ec_kept"
        else
            # All events expired — write empty file (touch creates it, or truncate)
            : > "$bus_file"
        fi

        release_lock "event-bus" 2>/dev/null || true
    fi

    json_ok "$(jq -nc \
        --argjson removed "$ec_removed" \
        --argjson remaining "$ec_kept_count" \
        --argjson dry_run "$ec_dry_run" \
        '{removed:$removed,remaining:$remaining,dry_run:$dry_run}')"
}

# ============================================================================
# _event_replay
# Replay events for a topic from a given timestamp.
# Usage: event-replay --topic <topic> --since <ISO-8601> [--limit <N>]
# ============================================================================
_event_replay() {
    local er_topic=""
    local er_since=""
    local er_limit="$_EVENT_BUS_DEFAULT_LIMIT"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --topic)
                er_topic="${2:-}"
                shift 2
                ;;
            --since)
                er_since="${2:-}"
                shift 2
                ;;
            --limit)
                er_limit="${2:-$_EVENT_BUS_DEFAULT_LIMIT}"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done

    [[ -z "$er_topic" ]] && json_err "$E_VALIDATION_FAILED" "event-replay requires --topic"
    [[ -z "$er_since" ]] && json_err "$E_VALIDATION_FAILED" "event-replay requires --since"

    local bus_file="$COLONY_DATA_DIR/event-bus.jsonl"
    local now_ts
    now_ts=$(date -u +%Y-%m-%dT%H:%M:%SZ)

    if [[ ! -f "$bus_file" ]]; then
        json_ok "$(jq -nc \
            --arg replayed_from "$er_since" \
            '{events:[],count:0,replayed_from:$replayed_from}')"
        return 0
    fi

    local er_events
    er_events=$(jq -sc \
        --arg topic "$er_topic" \
        --arg since "$er_since" \
        --arg now "$now_ts" \
        --argjson limit "$er_limit" \
        '[.[] | select(.topic == $topic) | select(.expires_at > $now) | select(.timestamp >= $since)] | sort_by(.timestamp) | .[:$limit]' \
        "$bus_file" 2>/dev/null || echo "[]")

    local er_count
    er_count=$(echo "$er_events" | jq 'length')

    json_ok "$(jq -nc \
        --argjson events "$er_events" \
        --argjson count "$er_count" \
        --arg replayed_from "$er_since" \
        '{events:$events,count:$count,replayed_from:$replayed_from}')"
}
