#!/bin/bash
# Instinct Store utility functions — standalone instinct storage with trust scoring
# Provides: _instinct_store, _instinct_read_trusted, _instinct_decay_all, _instinct_archive
#
# These functions are sourced by aether-utils.sh at startup.
# All shared infrastructure (json_ok, json_err, atomic_write, acquire_lock,
# release_lock, COLONY_DATA_DIR, SCRIPT_DIR, error constants) is available.
# Depends on trust-scoring.sh for trust score computation.
#
# State file: $COLONY_DATA_DIR/instincts.json
# Schema version: 1.0
# Cap: 50 instincts (lowest trust archived on overflow)

# ============================================================================
# _instinct_store
# Store a new instinct or reinforce an existing one (dedup by trigger prefix).
#
# Usage: instinct-store --trigger <t> --action <a> --domain <d> --confidence <f>
#                       --source <s> --evidence <e> [--source-type <type>]
#
# Deduplication: first 50 chars of trigger matched against existing entries.
# On match: boost confidence to max of existing/new, recompute trust score.
# Cap: 50 instincts. When exceeded, archive the entry with lowest trust_score.
# ============================================================================
_instinct_store() {
    local trigger=""
    local action=""
    local domain=""
    local confidence=""
    local source=""
    local evidence=""
    local source_type="observation"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --trigger)     trigger="${2:-}";      shift 2 ;;
            --action)      action="${2:-}";       shift 2 ;;
            --domain)      domain="${2:-}";       shift 2 ;;
            --confidence)  confidence="${2:-}";   shift 2 ;;
            --source)      source="${2:-}";       shift 2 ;;
            --evidence)    evidence="${2:-}";     shift 2 ;;
            --source-type) source_type="${2:-}";  shift 2 ;;
            *) shift ;;
        esac
    done

    [[ -z "$trigger" ]]     && json_err "$E_VALIDATION_FAILED" "Usage: instinct-store --trigger <t> --action <a> --domain <d> --confidence <f> --source <s> --evidence <e>"
    [[ -z "$action" ]]      && json_err "$E_VALIDATION_FAILED" "Usage: instinct-store --trigger <t> --action <a> --domain <d> --confidence <f> --source <s> --evidence <e>"
    [[ -z "$domain" ]]      && json_err "$E_VALIDATION_FAILED" "Usage: instinct-store --trigger <t> --action <a> --domain <d> --confidence <f> --source <s> --evidence <e>"
    [[ -z "$confidence" ]]  && json_err "$E_VALIDATION_FAILED" "Usage: instinct-store --trigger <t> --action <a> --domain <d> --confidence <f> --source <s> --evidence <e>"
    [[ -z "$source" ]]      && json_err "$E_VALIDATION_FAILED" "Usage: instinct-store --trigger <t> --action <a> --domain <d> --confidence <f> --source <s> --evidence <e>"
    [[ -z "$evidence" ]]    && json_err "$E_VALIDATION_FAILED" "Usage: instinct-store --trigger <t> --action <a> --domain <d> --confidence <f> --source <s> --evidence <e>"

    # Validate confidence is a number
    if ! [[ "$confidence" =~ ^[0-9]+(\.[0-9]+)?$ ]]; then
        json_err "$E_VALIDATION_FAILED" "--confidence must be a number between 0 and 1, got: $confidence"
    fi

    mkdir -p "$COLONY_DATA_DIR"
    local instincts_file="$COLONY_DATA_DIR/instincts.json"

    # Initialize file if missing
    if [[ ! -f "$instincts_file" ]]; then
        atomic_write "$instincts_file" '{"version":"1.0","instincts":[]}'
    fi

    local ts
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Compute trust score via trust-calculate
    local trust_result trust_score trust_tier
    trust_result=$(_trust_calculate --source "$source_type" --evidence "$evidence" --days-since 0 2>/dev/null) || true
    if echo "$trust_result" | jq -e '.ok == true' > /dev/null 2>&1; then
        trust_score=$(echo "$trust_result" | jq -r '.result.score')
        trust_tier=$(echo "$trust_result" | jq -r '.result.tier')
    else
        # Fallback: use confidence as trust score, derive tier
        trust_score="$confidence"
        trust_tier=$(_trust_score_to_tier "$confidence" 2>/dev/null || echo "provisional")
    fi

    # Acquire lock for atomic update
    acquire_lock "$instincts_file" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on instincts.json"
    trap 'release_lock 2>/dev/null || true' EXIT  # SUPPRESS:OK -- cleanup: lock may not be held

    # Dedup: check if a matching trigger prefix (first 50 chars) already exists
    local trigger_prefix
    trigger_prefix=$(echo "$trigger" | cut -c1-50)

    local existing_id
    existing_id=$(jq -r --arg prefix "$trigger_prefix" '
        .instincts[]
        | select(.archived == false)
        | select((.trigger | .[0:50]) == $prefix)
        | .id
        | select(. != null)
    ' "$instincts_file" 2>/dev/null | head -1)

    local updated
    if [[ -n "$existing_id" ]]; then
        # Reinforce: boost confidence to max, recompute trust score
        local new_confidence
        new_confidence=$(jq -r --arg id "$existing_id" \
            '.instincts[] | select(.id == $id) | .confidence' "$instincts_file")
        local boosted_confidence
        boosted_confidence=$(awk "BEGIN{
            a=$new_confidence; b=$confidence;
            printf \"%.4f\", (a > b ? a : b)
        }")

        # Recompute trust with boosted confidence
        local boost_trust_result
        boost_trust_result=$(_trust_calculate --source "$source_type" --evidence "$evidence" --days-since 0 2>/dev/null) || true
        if echo "$boost_trust_result" | jq -e '.ok == true' > /dev/null 2>&1; then
            trust_score=$(echo "$boost_trust_result" | jq -r '.result.score')
            trust_tier=$(echo "$boost_trust_result" | jq -r '.result.tier')
        fi

        updated=$(jq \
            --arg id "$existing_id" \
            --argjson conf "$boosted_confidence" \
            --argjson ts_score "$trust_score" \
            --arg ts_tier "$trust_tier" \
            --arg ts "$ts" '
            .instincts = [.instincts[] | if .id == $id then
                .confidence = $conf |
                .trust_score = $ts_score |
                .trust_tier = $ts_tier |
                .provenance.last_applied = $ts |
                .provenance.application_count += 1
            else . end]
        ' "$instincts_file") || json_err "$E_JSON_INVALID" "Failed to reinforce instinct"
        atomic_write "$instincts_file" "$updated"
        trap - EXIT
        release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
        json_ok "$(jq -n --arg id "$existing_id" --arg action "reinforced" \
            '{id: $id, action: $action}')"
        return
    fi

    # New instinct: generate id and build entry
    local id
    id="inst_$(date -u +%s)_$(head -c 3 /dev/urandom | od -An -tx1 | tr -d ' \n' | cut -c1-6)"

    local new_entry
    new_entry=$(jq -n \
        --arg id "$id" \
        --arg trigger "$trigger" \
        --arg action "$action" \
        --arg domain "$domain" \
        --argjson trust_score "$trust_score" \
        --arg trust_tier "$trust_tier" \
        --argjson confidence "$confidence" \
        --arg source "$source" \
        --arg source_type "$source_type" \
        --arg evidence "$evidence" \
        --arg ts "$ts" \
        '{
            id: $id,
            trigger: $trigger,
            action: $action,
            domain: $domain,
            trust_score: $trust_score,
            trust_tier: $trust_tier,
            confidence: $confidence,
            provenance: {
                source: $source,
                source_type: $source_type,
                evidence: $evidence,
                created_at: $ts,
                last_applied: null,
                application_count: 0
            },
            application_history: [],
            related_instincts: [],
            archived: false
        }')

    # Append entry
    updated=$(jq --argjson entry "$new_entry" '.instincts += [$entry]' "$instincts_file") \
        || json_err "$E_JSON_INVALID" "Failed to append instinct"

    # Enforce 50-entry cap: archive lowest-trust non-archived instinct if over limit
    local active_count
    active_count=$(echo "$updated" | jq '[.instincts[] | select(.archived == false)] | length')
    if [[ "$active_count" -gt 50 ]]; then
        updated=$(echo "$updated" | jq '
            # Find the id of the lowest-trust active instinct
            (
                [.instincts[] | select(.archived == false)]
                | sort_by(.trust_score)
                | .[0].id
            ) as $lowest_id
            |
            .instincts = [.instincts[] | if .id == $lowest_id then .archived = true else . end]
        ') || json_err "$E_JSON_INVALID" "Failed to enforce instinct cap"
    fi

    atomic_write "$instincts_file" "$updated"
    trap - EXIT
    release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
    json_ok "$(jq -n --arg id "$id" --arg action "stored" '{id: $id, action: $action}')"
}

# ============================================================================
# _instinct_read_trusted
# Read trusted instincts, sorted by trust_score descending.
#
# Usage: instinct-read-trusted [--min-score <f>] [--domain <d>] [--limit <N>]
#
# Defaults: min-score=0.5, limit=20. Excludes archived entries.
# ============================================================================
_instinct_read_trusted() {
    local min_score="0.5"
    local domain=""
    local limit="20"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --min-score) min_score="${2:-0.5}"; shift 2 ;;
            --domain)    domain="${2:-}";       shift 2 ;;
            --limit)     limit="${2:-20}";      shift 2 ;;
            *) shift ;;
        esac
    done

    local instincts_file="$COLONY_DATA_DIR/instincts.json"
    if [[ ! -f "$instincts_file" ]]; then
        json_ok '{"instincts":[],"count":0}'
        return
    fi

    local result
    result=$(jq -n \
        --argjson min_score "$min_score" \
        --arg domain "$domain" \
        --argjson limit "$limit" \
        --slurpfile data "$instincts_file" '
        $data[0].instincts
        | [.[] | select(.archived == false)]
        | [.[] | select(.trust_score >= $min_score)]
        | (if $domain != "" then [.[] | select(.domain == $domain)] else . end)
        | sort_by(-.trust_score)
        | .[0:$limit]
        | {instincts: ., count: length}
    ') || json_err "$E_JSON_INVALID" "Failed to read trusted instincts"

    json_ok "$result"
}

# ============================================================================
# _instinct_decay_all
# Apply trust-based time decay to all non-archived instincts.
#
# Usage: instinct-decay-all [--days <N>] [--dry-run]
#
# Archives instincts whose decayed score falls below 0.25.
# Updates trust_tier for all processed entries.
# ============================================================================
_instinct_decay_all() {
    local days="30"
    local dry_run="false"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --days)    days="${2:-30}"; shift 2 ;;
            --dry-run) dry_run="true"; shift ;;
            *) shift ;;
        esac
    done

    local instincts_file="$COLONY_DATA_DIR/instincts.json"
    if [[ ! -f "$instincts_file" ]]; then
        json_ok '{"processed":0,"archived":0,"dry_run":false}'
        return
    fi

    # Read all active instincts and apply decay
    local current_data
    current_data=$(cat "$instincts_file")

    local active_count
    active_count=$(echo "$current_data" | jq '[.instincts[] | select(.archived == false)] | length')

    if [[ "$active_count" -eq 0 ]]; then
        json_ok "$(jq -n --argjson days "$days" '{"processed":0,"archived":0,"dry_run":false}')"
        return
    fi

    # Build updated instincts array: apply decay to each active entry
    local updated_instincts="[]"
    local archived_count=0

    while IFS= read -r instinct_json; do
        local current_score
        current_score=$(echo "$instinct_json" | jq -r '.trust_score')

        local decay_result decayed_score new_tier
        decay_result=$(_trust_decay --score "$current_score" --days "$days" 2>/dev/null) || true

        if echo "$decay_result" | jq -e '.ok == true' > /dev/null 2>&1; then
            decayed_score=$(echo "$decay_result" | jq -r '.result.decayed')
        else
            decayed_score="$current_score"
        fi

        new_tier=$(_trust_score_to_tier "$decayed_score" 2>/dev/null || echo "dormant")

        local should_archive="false"
        local below_threshold
        below_threshold=$(awk "BEGIN{print ($decayed_score < 0.25)}" 2>/dev/null || echo "0")
        if [[ "$below_threshold" == "1" ]]; then
            should_archive="true"
            archived_count=$((archived_count + 1))
        fi

        local updated_entry
        updated_entry=$(echo "$instinct_json" | jq \
            --argjson score "$decayed_score" \
            --arg tier "$new_tier" \
            --argjson archive "$should_archive" \
            '.trust_score = $score | .trust_tier = $tier | .archived = $archive')

        updated_instincts=$(echo "$updated_instincts" | jq \
            --argjson entry "$updated_entry" \
            '. += [$entry]')
    done < <(echo "$current_data" | jq -c '.instincts[] | select(.archived == false)')

    # Merge: keep archived entries as-is, replace active entries with updated versions
    local merged
    merged=$(jq -n \
        --argjson updated "$updated_instincts" \
        --argjson original "$(echo "$current_data" | jq '.instincts')" \
        '
        # Build a lookup of updated entries by id
        ($updated | map({(.id): .}) | add // {}) as $lookup
        |
        [
            $original[] | if .archived == true then
                .
            elif ($lookup[.id] != null) then
                $lookup[.id]
            else
                .
            end
        ]
    ')

    local final_data
    final_data=$(echo "$current_data" | jq --argjson instincts "$merged" '.instincts = $instincts')

    if [[ "$dry_run" != "true" ]]; then
        acquire_lock "$instincts_file" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on instincts.json"
        trap 'release_lock 2>/dev/null || true' EXIT  # SUPPRESS:OK -- cleanup: lock may not be held
        atomic_write "$instincts_file" "$final_data"
        trap - EXIT
        release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
    fi

    json_ok "$(jq -n \
        --argjson processed "$active_count" \
        --argjson archived "$archived_count" \
        --argjson days "$days" \
        --argjson dry_run "$([ "$dry_run" == "true" ] && echo true || echo false)" \
        '{processed: $processed, archived: $archived, days: $days, dry_run: $dry_run}')"
}

# ============================================================================
# _instinct_archive
# Soft-delete an instinct by id (sets archived: true).
#
# Usage: instinct-archive --id <id>
# ============================================================================
_instinct_archive() {
    local id=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --id) id="${2:-}"; shift 2 ;;
            *) shift ;;
        esac
    done

    [[ -z "$id" ]] && json_err "$E_VALIDATION_FAILED" "Usage: instinct-archive --id <id>"

    local instincts_file="$COLONY_DATA_DIR/instincts.json"
    [[ ! -f "$instincts_file" ]] && json_err "$E_FILE_NOT_FOUND" "No instincts.json found"

    acquire_lock "$instincts_file" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on instincts.json"
    trap 'release_lock 2>/dev/null || true' EXIT  # SUPPRESS:OK -- cleanup: lock may not be held

    local updated
    updated=$(jq --arg id "$id" '
        .instincts = [.instincts[] | if .id == $id then .archived = true else . end]
    ' "$instincts_file") || json_err "$E_JSON_INVALID" "Failed to archive instinct"

    atomic_write "$instincts_file" "$updated"
    trap - EXIT
    release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
    json_ok "$(jq -n --arg id "$id" '{archived: $id}')"
}
