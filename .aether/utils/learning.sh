# =============================================================================
# DEPRECATED — This script has been superseded by the Go binary (aether CLI).
# All functionality is now available via: aether <subcommand>
# Do NOT modify this file — it is retained for reference only.
# See: cmd/ (Go source) | Run: aether --help
# =============================================================================
#
#!/usr/bin/env bash
# Learning and instinct utility functions -- extracted from aether-utils.sh
# Provides: _learning_promote, _learning_inject, _learning_observe, _learning_check_promotion,
#           _learning_promote_auto, _learning_display_proposals, _learning_select_proposals,
#           _learning_defer_proposals, _learning_approve_proposals, _learning_undo_promotions,
#           _instinct_read, _instinct_create, _instinct_apply, _learning_extract_fallback
# Note: Uses get_wisdom_threshold() and get_wisdom_thresholds_json() from main file.
#       Cross-domain calls (queen-promote, pheromone-write, activity-log, rolling-summary,
#       generate-threshold-bar, parse-selection) are all via subprocess dispatch (bash "$0").

# ============================================================================
# _learning_promote
# Promote a learning to the global learnings file
# Usage: learning-promote <content> <source_project> <source_phase> [tags]
# ============================================================================
_learning_promote() {
    [[ $# -ge 3 ]] || json_err "$E_VALIDATION_FAILED" "Usage: learning-promote <content> <source_project> <source_phase> [tags]"
    content="$1"
    source_project="$2"
    source_phase="$3"
    tags="${4:-}"

    mkdir -p "$DATA_DIR"
    global_file="$COLONY_DATA_DIR/learnings.json"

    if [[ ! -f "$global_file" ]]; then
      atomic_write "$global_file" '{"learnings":[],"version":1}' || json_err "$E_UNKNOWN" "Failed to initialize learnings file"
    fi

    id="global_$(date -u +%s)_$(head -c 2 /dev/urandom | od -An -tx1 | tr -d ' ')"
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    if [[ -n "$tags" ]]; then
      tags_json=$(echo "$tags" | jq -R 'split(",")')
    else
      tags_json="[]"
    fi

    current_count=$(jq '.learnings | length' "$global_file")
    if [[ $current_count -ge 50 ]]; then
      json_ok "{\"promoted\":false,\"reason\":\"cap_reached\",\"current_count\":$current_count,\"cap\":50}"
      exit 0
    fi

    updated=$(jq --arg id "$id" --arg content "$content" --arg sp "$source_project" \
      --arg phase "$source_phase" --argjson tags "$tags_json" --arg ts "$ts" '
      .learnings += [{
        id: $id,
        content: $content,
        source_project: $sp,
        source_phase: $phase,
        tags: $tags,
        promoted_at: $ts
      }]
    ' "$global_file") || json_err "$E_JSON_INVALID" "Failed to update learnings.json"

    atomic_write "$global_file" "$updated" || {
      _aether_log_error "Could not save updated learnings"
      json_err "$E_UNKNOWN" "Failed to write learnings file"
    }
    json_ok "$(jq -n --arg id "$id" --argjson count "$((current_count + 1))" '{promoted: true, id: $id, count: $count, cap: 50}')"
}

# ============================================================================
# _learning_inject
# Filter learnings by tech keywords for worker context injection
# Usage: learning-inject <tech_keywords_csv>
# ============================================================================
_learning_inject() {
    [[ $# -ge 1 ]] || json_err "$E_VALIDATION_FAILED" "Usage: learning-inject <tech_keywords_csv>"
    keywords="$1"

    global_file="$COLONY_DATA_DIR/learnings.json"

    if [[ ! -f "$global_file" ]]; then
      json_ok '{"learnings":[],"count":0}'
      exit 0
    fi

    json_ok "$(jq --arg kw "$keywords" '
      ($kw | split(",") | map(ascii_downcase | ltrimstr(" ") | rtrimstr(" "))) as $keywords |
      .learnings | map(
        select(
          .tags as $tags |
          ($keywords | any(. as $k | $tags | any(ascii_downcase | contains($k))))
        )
      ) | {learnings: ., count: length}
    ' "$global_file")"
}

# ============================================================================
# _learning_observe
# Record observation of a learning across colonies
# Usage: learning-observe <content> <wisdom_type> [colony_name] [source_type] [evidence_type]
# Returns: JSON with observation_count, threshold status, colonies list, and trust_score
# ============================================================================
_learning_observe() {
    # Record observation of a learning across colonies
    # Usage: learning-observe <content> <wisdom_type> [colony_name] [source_type] [evidence_type]
    # Returns: JSON with observation_count, threshold status, colonies list, and trust_score
    content="${1:-}"
    wisdom_type="${2:-}"
    colony_name="${3:-unknown}"
    local source_type="${4:-observation}"
    local evidence_type="${5:-anecdotal}"

    # Validate required arguments
    [[ -z "$content" ]] && json_err "$E_VALIDATION_FAILED" "Usage: learning-observe <content> <wisdom_type> [colony_name]" '{"missing":"content"}'
    [[ -z "$wisdom_type" ]] && json_err "$E_VALIDATION_FAILED" "Usage: learning-observe <content> <wisdom_type> [colony_name]" '{"missing":"wisdom_type"}'

    # Validate wisdom_type
    valid_types=("philosophy" "pattern" "redirect" "stack" "decree" "failure")
    type_valid=false
    for vt in "${valid_types[@]}"; do
      [[ "$wisdom_type" == "$vt" ]] && type_valid=true && break
    done
    [[ "$type_valid" == "false" ]] && json_err "$E_VALIDATION_FAILED" "Invalid wisdom_type: $wisdom_type" '{"valid_types":["philosophy","pattern","redirect","stack","decree","failure"]}'

    # Generate SHA256 hash of content for deduplication
    content_hash="sha256:$(echo -n "$content" | sha256sum | cut -d' ' -f1)"

    # Observations file path
    observations_file="$COLONY_DATA_DIR/learning-observations.json"

    # Ensure data directory exists
    [[ ! -d "$DATA_DIR" ]] && mkdir -p "$DATA_DIR"

    # Initialize file if it doesn't exist
    if [[ ! -f "$observations_file" ]]; then
      atomic_write "$observations_file" '{"observations":[]}' || {
        _aether_log_error "Could not initialize learning observations file"
        json_err "$E_UNKNOWN" "Failed to create learning observations file"
      }
    fi

    # Validate JSON structure — circuit breaker with backup recovery
    if ! jq -e . "$observations_file" >/dev/null 2>&1; then  # SUPPRESS:OK -- validation: testing JSON validity
      # Try to recover from backup (with retry-once per user decision)
      lo_recovered=false
      for lo_attempt in 1 2; do
        for lo_bak in "${observations_file}.bak.1" "${observations_file}.bak.2" "${observations_file}.bak.3"; do
          if [[ -f "$lo_bak" ]] && jq -e . "$lo_bak" >/dev/null 2>&1; then  # SUPPRESS:OK -- validation: testing JSON validity
            if cp "$lo_bak" "$observations_file" 2>/dev/null; then  # SUPPRESS:OK -- cleanup: backup copy is best-effort
              lo_recovered=true
              echo "Warning: Learning observations file was corrupted -- restored from backup. Some recent entries may be missing." >&2
              break 2
            fi
            # cp failed -- will retry on next attempt (silent first retry)
          fi
        done
        # If first attempt found a valid backup but cp failed, the second attempt retries
        # If no valid backup exists, the second attempt won't help -- break early
        [[ "$lo_attempt" -eq 1 ]] && [[ "$lo_recovered" != "true" ]] && break
      done

      if [[ "$lo_recovered" != "true" ]]; then
        # Check if any backups exist at all
        lo_has_any_backup=false
        for lo_bak in "${observations_file}.bak.1" "${observations_file}.bak.2" "${observations_file}.bak.3"; do
          [[ -f "$lo_bak" ]] && lo_has_any_backup=true && break
        done

        if [[ "$lo_has_any_backup" == "true" ]]; then
          # Backups exist but ALL are corrupted -- stop and tell user (per locked decision)
          json_err "$E_JSON_INVALID" "Learning observations and all 3 backups are corrupted. Manual recovery needed."
        else
          # No backups ever existed -- safe to reset from template (first-time corruption)
          echo "Warning: Learning observations file was corrupted. Starting fresh -- this is a first-time recovery." >&2
          atomic_write "$observations_file" '{"observations":[]}'
        fi
      fi
    fi

    # Acquire lock for concurrent access
    if type acquire_lock &>/dev/null; then
      acquire_lock "$observations_file" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on learning-observations.json"
      trap 'release_lock 2>/dev/null || true' EXIT  # SUPPRESS:OK -- cleanup: lock may not be held
    fi

    # Get current timestamp
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Initialize trust_score (set for new observations; left as default for existing)
    local trust_score="0.490000"

    # --- Legacy migration: backfill missing fields on any entries that lack trust_score ---
    local has_legacy
    has_legacy=$(jq '[.observations[] | select(.trust_score == null)] | length' "$observations_file")
    if [[ "$has_legacy" -gt 0 ]]; then
      jq '.observations |= map(
        if .trust_score == null then
          .trust_score = 0.49 |
          .source_type = "legacy" |
          .evidence_type = "indirect" |
          .compression_level = 0
        else . end
      )' "$observations_file" > "${observations_file}.migrate.$$" && \
      mv "${observations_file}.migrate.$$" "$observations_file" 2>/dev/null || {
        rm -f "${observations_file}.migrate.$$"
        _aether_log_error "Could not backfill legacy observation fields"
      }
    fi

    # Check if observation with same hash already exists
    existing_index=$(jq -r --arg hash "$content_hash" '.observations | to_entries[] | select(.value.content_hash == $hash) | .key' "$observations_file" | head -1)

    if [[ -n "$existing_index" ]]; then
      # Existing observation: increment count, update last_seen, add colony if new
      # Recalculate trust_score based on existing source/evidence types and days since first_seen
      local existing_source existing_evidence existing_first_seen days_since
      existing_source=$(jq -r --arg hash "$content_hash" '.observations[] | select(.content_hash == $hash) | .source_type // "observation"' "$observations_file")
      existing_evidence=$(jq -r --arg hash "$content_hash" '.observations[] | select(.content_hash == $hash) | .evidence_type // "anecdotal"' "$observations_file")

      # Map legacy/invalid types to valid trust-calculate types
      case "$existing_source" in
        user_feedback|error_resolution|success_pattern|observation|heuristic) ;;
        *) existing_source="observation" ;;
      esac
      case "$existing_evidence" in
        test_verified|multi_phase|single_phase|anecdotal) ;;
        *) existing_evidence="anecdotal" ;;
      esac
      existing_first_seen=$(jq -r --arg hash "$content_hash" '.observations[] | select(.content_hash == $hash) | .first_seen' "$observations_file")

      # Compute days since first_seen
      if [[ -n "$existing_first_seen" ]]; then
        local epoch_first epoch_now
        epoch_first=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "$existing_first_seen" "+%s" 2>/dev/null || echo "0")
        epoch_now=$(date -u +%s)
        days_since=$(( (epoch_now - epoch_first) / 86400 ))
        [[ "$days_since" -lt 0 ]] && days_since=0
      else
        days_since=0
      fi

      # Calculate trust score for the re-observation
      local trust_result
      trust_result=$(bash "$0" trust-calculate --source "$existing_source" --evidence "$existing_evidence" --days-since "$days_since" 2>/dev/null)
      trust_score=$(echo "$trust_result" | jq -r '.result.score // "0.490000"' 2>/dev/null)

      # Rotate backups before write (uses .bak.N naming)
      if [[ -f "$observations_file" ]]; then
        cp -f "${observations_file}.bak.2" "${observations_file}.bak.3" 2>/dev/null || _aether_log_error "Could not rotate observations backup .bak.2 to .bak.3"
        cp -f "${observations_file}.bak.1" "${observations_file}.bak.2" 2>/dev/null || _aether_log_error "Could not rotate observations backup .bak.1 to .bak.2"
        cp -f "$observations_file" "${observations_file}.bak.1" 2>/dev/null || _aether_log_error "Could not create observations backup .bak.1"
      fi
      tmp_file="${observations_file}.tmp.$$"

      jq --arg hash "$content_hash" \
         --arg colony "$colony_name" \
         --arg ts "$ts" \
         --argjson trust_score "$trust_score" \
         '
         .observations |= map(
           if .content_hash == $hash then
             .observation_count += 1 |
             .last_seen = $ts |
             .colonies = ((.colonies + [$colony]) | unique) |
             .trust_score = ($trust_score | tonumber)
           else
             .
           end
         )' "$observations_file" > "$tmp_file" || {
        _aether_log_error "Could not process observation update"
        rm -f "$tmp_file"
        json_err "$E_JSON_INVALID" "Failed to update observation data"
      }

      [[ -s "$tmp_file" ]] || {
        _aether_log_error "Observation update produced empty result -- not overwriting"
        rm -f "$tmp_file"
        json_err "$E_JSON_INVALID" "Observation update produced empty result"
      }

      mv "$tmp_file" "$observations_file" || {
        _aether_log_error "Could not finalize observation file update"
        rm -f "$tmp_file"
        json_err "$E_UNKNOWN" "Failed to rename temporary observations file"
      }

      # Get updated observation data
      observation_count=$(jq -r --arg hash "$content_hash" '.observations[] | select(.content_hash == $hash) | .observation_count' "$observations_file")
      colonies=$(jq -r --arg hash "$content_hash" '.observations[] | select(.content_hash == $hash) | .colonies' "$observations_file")
      is_new=false
    else
      # New observation: create entry
      # Rotate backups before write (uses .bak.N naming)
      if [[ -f "$observations_file" ]]; then
        cp -f "${observations_file}.bak.2" "${observations_file}.bak.3" 2>/dev/null || _aether_log_error "Could not rotate observations backup .bak.2 to .bak.3"
        cp -f "${observations_file}.bak.1" "${observations_file}.bak.2" 2>/dev/null || _aether_log_error "Could not rotate observations backup .bak.1 to .bak.2"
        cp -f "$observations_file" "${observations_file}.bak.1" 2>/dev/null || _aether_log_error "Could not create observations backup .bak.1"
      fi
      tmp_file="${observations_file}.tmp.$$"

      # Compute trust score for new observation
      local trust_result
      trust_result=$(bash "$0" trust-calculate --source "$source_type" --evidence "$evidence_type" --days-since 0 2>/dev/null)
      local trust_score
      trust_score=$(echo "$trust_result" | jq -r '.result.score // "0.490000"' 2>/dev/null)

      jq --arg hash "$content_hash" \
         --arg content "$content" \
         --arg type "$wisdom_type" \
         --arg colony "$colony_name" \
         --arg ts "$ts" \
         --arg trust_score "$trust_score" \
         --arg source_type "$source_type" \
         --arg evidence_type "$evidence_type" \
         '.observations += [{
           "content_hash": $hash,
           "content": $content,
           "wisdom_type": $type,
           "observation_count": 1,
           "first_seen": $ts,
           "last_seen": $ts,
           "colonies": [$colony],
           "trust_score": ($trust_score | tonumber),
           "source_type": $source_type,
           "evidence_type": $evidence_type,
           "compression_level": 0
         }]' "$observations_file" > "$tmp_file" || {
        _aether_log_error "Could not create new observation entry"
        rm -f "$tmp_file"
        json_err "$E_JSON_INVALID" "Failed to create observation data"
      }

      [[ -s "$tmp_file" ]] || {
        _aether_log_error "New observation entry produced empty result -- not overwriting"
        rm -f "$tmp_file"
        json_err "$E_JSON_INVALID" "New observation produced empty result"
      }

      mv "$tmp_file" "$observations_file" || {
        _aether_log_error "Could not finalize new observation file update"
        rm -f "$tmp_file"
        json_err "$E_UNKNOWN" "Failed to rename temporary observations file"
      }

      observation_count=1
      colonies="[\"$colony_name\"]"
      is_new=true
    fi

    # Release lock
    if type release_lock &>/dev/null; then
      release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
    fi
    trap - EXIT

    # Propose-threshold determines when a learning is queueable/reviewable.
    threshold=$(get_wisdom_threshold "$wisdom_type" "propose")

    # Determine if threshold is met
    threshold_met=false
    [[ "$observation_count" -ge "$threshold" ]] && threshold_met=true

    # Return result
    result=$(jq -n \
      --arg hash "$content_hash" \
      --arg content "$content" \
      --arg type "$wisdom_type" \
      --argjson count "$observation_count" \
      --argjson threshold "$threshold" \
      --argjson threshold_met "$threshold_met" \
      --argjson colonies "$colonies" \
      --argjson is_new "$is_new" \
      --arg trust_score "$trust_score" \
      '{
        content_hash: $hash,
        content: $content,
        wisdom_type: $type,
        observation_count: $count,
        threshold: $threshold,
        threshold_met: $threshold_met,
        colonies: $colonies,
        is_new: $is_new,
        trust_score: ($trust_score | tonumber)
      }')

    json_ok "$result"
}

# ============================================================================
# _learning_check_promotion
# Check which learnings meet promotion thresholds
# Usage: learning-check-promotion [path_to_observations_file]
# Returns: JSON array of proposals meeting thresholds
# ============================================================================
_learning_check_promotion() {
    observations_file="${1:-$COLONY_DATA_DIR/learning-observations.json}"

    # Default to empty file path if not provided and data dir doesn't exist
    if [[ -z "${1:-}" ]] && [[ ! -d "$DATA_DIR" ]]; then
      observations_file=""
    fi

    # If file doesn't exist or is empty, return empty proposals
    if [[ -z "$observations_file" ]] || [[ ! -f "$observations_file" ]]; then
      json_ok '{"proposals":[]}'
      exit 0
    fi

    # Validate JSON structure
    if ! jq -e . "$observations_file" >/dev/null 2>&1; then  # SUPPRESS:OK -- validation: testing JSON validity
      json_err "$E_JSON_INVALID" "learning-observations.json has invalid JSON"
    fi

    # Build proposals array using the shared threshold table.
    thresholds_json=$(get_wisdom_thresholds_json)
    result=$(jq --argjson thresholds "$thresholds_json" '
      def get_threshold(type):
        ($thresholds[type].propose // 1);

      {
        proposals: [
          .observations[] |
          select((.observation_count // 0) >= get_threshold(.wisdom_type)) |
          {
            content: .content,
            wisdom_type: .wisdom_type,
            observation_count: .observation_count,
            threshold: get_threshold(.wisdom_type),
            colonies: (.colonies // []),
            ready: true
          }
        ]
      }
    ' "$observations_file" 2>/dev/null || echo '{"proposals":[]}')  # SUPPRESS:OK -- read-default: file may not exist yet

    json_ok "$result"
}

# ============================================================================
# _learning_promote_auto
# Auto-promote high-confidence learnings using recurrence policy
# Usage: learning-promote-auto <wisdom_type> <content> [colony_name] [event_type]
# ============================================================================
_learning_promote_auto() {
    wisdom_type="${1:-}"
    content="${2:-}"
    colony_name="${3:-}"
    event_type="${4:-learning}"

    [[ -z "$wisdom_type" ]] && json_err "$E_VALIDATION_FAILED" "Usage: learning-promote-auto <wisdom_type> <content> [colony_name] [event_type]" '{"missing":"wisdom_type"}'
    [[ -z "$content" ]] && json_err "$E_VALIDATION_FAILED" "Usage: learning-promote-auto <wisdom_type> <content> [colony_name] [event_type]" '{"missing":"content"}'

    if [[ -z "$colony_name" ]]; then
      colony_name=$(bash "$0" colony-name 2>/dev/null | jq -r '.result.name // ""') || colony_name="unknown"
      [[ -z "$colony_name" ]] && colony_name="unknown"
    fi

    policy_threshold=$(get_wisdom_threshold "$wisdom_type" "auto")

    observations_file="$COLONY_DATA_DIR/learning-observations.json"
    content_hash="sha256:$(echo -n "$content" | sha256sum | cut -d' ' -f1)"
    observation_count=0
    colony_count=0

    if [[ -f "$observations_file" ]]; then
      # SUPPRESS:OK -- read-default: query may return empty
      observation_count=$(jq -r --arg hash "$content_hash" '.observations[]? | select(.content_hash == $hash) | .observation_count // 0' "$observations_file" 2>/dev/null | head -1)
      # SUPPRESS:OK -- read-default: query may return empty
      colony_count=$(jq -r --arg hash "$content_hash" '.observations[]? | select(.content_hash == $hash) | (.colonies // [] | length)' "$observations_file" 2>/dev/null | head -1)
      [[ -z "$observation_count" ]] && observation_count=0
      [[ -z "$colony_count" ]] && colony_count=0
    fi

    # LRN-01: Recurrence-calibrated confidence
    # Formula: min(0.7 + (observation_count - 1) * 0.05, 0.9)
    lp_confidence=$(awk -v c="${observation_count:-1}" 'BEGIN {
      v = 0.7 + (c - 1) * 0.05
      if (v > 0.9) v = 0.9
      if (v < 0.7) v = 0.7
      printf "%.2f", v
    }')

    if [[ "$policy_threshold" -gt 0 && "$observation_count" -lt "$policy_threshold" ]]; then
      json_ok "$(jq -n --argjson pt "$policy_threshold" --argjson oc "$observation_count" --argjson cc "$colony_count" --arg et "$event_type" '{promoted: false, reason: "threshold_not_met", policy_threshold: $pt, observation_count: $oc, colony_count: $cc, event_type: $et}')"
      exit 0
    fi

    queen_file="$AETHER_ROOT/.aether/QUEEN.md"
    if [[ ! -f "$queen_file" ]]; then
      json_ok "{\"promoted\":false,\"reason\":\"queen_missing\",\"policy_threshold\":$policy_threshold,\"observation_count\":$observation_count,\"colony_count\":$colony_count}"
      exit 0
    fi

    if grep -Fq -- "$content" "$queen_file" 2>/dev/null; then  # SUPPRESS:OK -- existence-test: file may not exist
      json_ok "{\"promoted\":false,\"reason\":\"already_promoted\",\"policy_threshold\":$policy_threshold,\"observation_count\":$observation_count,\"colony_count\":$colony_count}"
      exit 0
    fi

    # SUPPRESS:OK -- read-default: returns fallback on failure
    promote_result=$(bash "$0" queen-promote "$wisdom_type" "$content" "$colony_name" 2>/dev/null || echo '{}')
    if echo "$promote_result" | jq -e '.ok == true' >/dev/null 2>&1; then  # SUPPRESS:OK -- validation: testing JSON field
      # Also create an instinct from the promoted learning
      bash "$0" instinct-create \
        --trigger "working on $wisdom_type patterns" \
        --action "$content" \
        --confidence "$lp_confidence" \
        --domain "$wisdom_type" \
        --source "promoted_from_learning" \
        # SUPPRESS:OK -- read-default: returns fallback on failure
        --evidence "Auto-promoted after $observation_count observations (confidence: $lp_confidence)" 2>/dev/null \
        || _aether_log_error "Could not create instinct from promoted learning"
      json_ok "$(jq -nc --argjson pt "$policy_threshold" --argjson oc "$observation_count" --argjson cc "$colony_count" --arg et "$event_type" '{promoted: true, mode: "auto", policy_threshold: $pt, observation_count: $oc, colony_count: $cc, event_type: $et}')"
    else
      # SUPPRESS:OK -- read-default: query may return empty
      promote_msg=$(echo "$promote_result" | jq -r '.error.message // "promotion_failed"' 2>/dev/null || echo "promotion_failed")
      result=$(jq -nc \
        --arg reason "promotion_failed" \
        --arg message "$promote_msg" \
        --argjson policy_threshold "$policy_threshold" \
        --argjson observation_count "$observation_count" \
        --argjson colony_count "$colony_count" \
        '{promoted:false, reason:$reason, message:$message, policy_threshold:$policy_threshold, observation_count:$observation_count, colony_count:$colony_count}')
      json_ok "$result"
    fi
}

# ============================================================================
# _learning_display_proposals
# Display promotion proposals with checkbox-style UI
# Usage: learning-display-proposals [observations_file] [--verbose] [--no-color]
# Returns: Formatted display output (not JSON - for human consumption)
# ============================================================================
_learning_display_proposals() {
    verbose=false
    no_color=false
    observations_file=""

    # Parse arguments
    for arg in "$@"; do
      case "$arg" in
        --verbose) verbose=true ;;
        --no-color) no_color=true ;;
        *)
          # If argument doesn't start with --, treat as file path
          if [[ "$arg" != --* ]] && [[ -z "$observations_file" ]]; then
            observations_file="$arg"
          fi
          ;;
      esac
    done

    # Detect color support
    use_color=false
    if [[ "$no_color" == "false" ]] && [[ -t 1 ]]; then
      use_color=true
    fi

    # Color codes
    reset=""
    yellow=""
    red=""
    cyan=""
    if [[ "$use_color" == "true" ]]; then
      reset="\033[0m"
      yellow="\033[33m"
      red="\033[31m"
      cyan="\033[36m"
    fi

    # Determine observations file path
    if [[ -z "$observations_file" ]]; then
      observations_file="$COLONY_DATA_DIR/learning-observations.json"
    fi

    # Check if file exists and has content
    if [[ ! -f "$observations_file" ]] || [[ ! -s "$observations_file" ]]; then
      echo "No observations found."
      echo ""
      echo "Observations accumulate as colonies report learnings."
      echo "Run this command again after more activity."
      exit 0
    fi

    # Get all observations with their thresholds
    thresholds_json=$(get_wisdom_thresholds_json)
    proposals_json=$(jq --argjson thresholds "$thresholds_json" '
      def get_threshold(type):
        ($thresholds[type].propose // 1);

      {
        proposals: [
          .observations[] |
          {
            content: .content,
            wisdom_type: .wisdom_type,
            observation_count: .observation_count,
            threshold: get_threshold(.wisdom_type),
            colonies: .colonies
          }
        ]
      }
    ' "$observations_file" 2>/dev/null || echo '{"proposals":[]}')  # SUPPRESS:OK -- read-default: file may not exist yet

    # Check if there are any proposals
    proposal_count=$(echo "$proposals_json" | jq '.proposals | length')
    if [[ "$proposal_count" -eq 0 ]]; then
      echo "No proposals ready for promotion."
      echo ""
      echo "Observations accumulate as colonies report learnings."
      echo "Run this command again after more activity."
      exit 0
    fi

    # Define wisdom types and their display properties
    types=("philosophy" "pattern" "redirect" "stack" "decree" "failure")
    type_emojis=("📜" "🧭" "⚠️" "🔧" "🏛️" "❌")
    type_names=("Philosophies" "Patterns" "Redirects" "Stack Wisdom" "Decrees" "Failures")
    type_thresholds=(
      "$(get_wisdom_threshold philosophy propose)"
      "$(get_wisdom_threshold pattern propose)"
      "$(get_wisdom_threshold redirect propose)"
      "$(get_wisdom_threshold stack propose)"
      "$(get_wisdom_threshold decree propose)"
      "$(get_wisdom_threshold failure propose)"
    )

    echo ""
    echo "🧠 Promotion Proposals"
    echo "====================="
    echo ""
    echo "Select proposals to promote to QUEEN.md wisdom:"
    echo "(Enter numbers like '1 3 5', or press Enter to defer all)"
    echo ""

    # Build flat list of all proposals with global numbering
    global_idx=1
    declare -a all_proposals

    for i in "${!types[@]}"; do
      type="${types[$i]}"
      threshold="${type_thresholds[$i]}"

      # Get proposals of this type
      type_proposals=$(echo "$proposals_json" | jq --arg t "$type" '.proposals | map(select(.wisdom_type == $t))')
      type_count=$(echo "$type_proposals" | jq 'length')

      [[ "$type_count" -eq 0 ]] && continue

      # Print group header
      echo "${type_emojis[$i]} ${type_names[$i]} (threshold: $threshold)"

      # Process each proposal using index to avoid subshell issues
      for ((j=0; j<type_count; j++)); do
        proposal=$(echo "$type_proposals" | jq -c ".[$j]")
        content=$(sanitize_read_value "$(echo "$proposal" | jq -r '.content')")
        count=$(echo "$proposal" | jq -r '.observation_count')
        prop_threshold=$(echo "$proposal" | jq -r '.threshold')

        # Truncate content if not verbose
        display_content="$content"
        if [[ "$verbose" != "true" && ${#content} -gt 40 ]]; then
          display_content="${content:0:37}..."
        fi

        # Get threshold bar
        # SUPPRESS:OK -- read-default: returns fallback on failure
        bar_result=$(bash "$0" generate-threshold-bar "$count" "$prop_threshold" 2>/dev/null | jq -r '.result.bar')

        # Build warning for below-threshold
        warning=""
        if [[ "$count" -lt "$prop_threshold" ]]; then
          if [[ "$use_color" == "true" ]]; then
            warning=" ${yellow}⚠️ below threshold${reset}"
          else
            warning=" ⚠️ below threshold"
          fi
        fi

        # Print formatted line
        printf "  [ ] %d. \"%s\" %s (%d/%d)%s\n" "$global_idx" "$display_content" "$bar_result" "$count" "$prop_threshold" "$warning"

        # Store for later reference
        all_proposals+=("$proposal")

        global_idx=$((global_idx + 1))
      done

      echo ""
    done

    echo "───────────────────────────────────────────────────"
    echo ""
}

# ============================================================================
# _learning_select_proposals
# Interactive selection of proposals for promotion [DEPRECATED]
# Usage: learning-select-proposals [--verbose] [--dry-run] [--yes]
# Returns: JSON with selected/deferred arrays and action taken
# ============================================================================
_learning_select_proposals() {
    _deprecation_warning "learning-select-proposals"
    # Interactive selection of proposals for promotion
    # Usage: learning-select-proposals [--verbose] [--dry-run] [--yes]
    # Returns: JSON with selected/deferred arrays and action taken
    #
    # Flow: display proposals -> capture input -> parse selection -> output JSON

    verbose=false
    dry_run=false
    skip_confirm=false

    # Parse arguments
    for arg in "$@"; do
      case "$arg" in
        --verbose) verbose=true ;;
        --dry-run) dry_run=true ;;
        --yes) skip_confirm=true ;;
      esac
    done

    # Get all observations (not just threshold-meeting ones) for display consistency
    # This matches learning-display-proposals behavior
    observations_file="$COLONY_DATA_DIR/learning-observations.json"
    if [[ ! -f "$observations_file" ]]; then
      json_ok '{"selected":[],"deferred":[],"count":0,"action":"none","reason":"no_observations_file"}'
      exit 0
    fi

    # Build proposals array using same logic as learning-display-proposals
    thresholds_json=$(get_wisdom_thresholds_json)
    proposals_json=$(jq --argjson thresholds "$thresholds_json" '
      def get_threshold(type):
        ($thresholds[type].propose // 1);

      {
        proposals: [
          .observations[] |
          {
            content: .content,
            wisdom_type: .wisdom_type,
            observation_count: .observation_count,
            threshold: get_threshold(.wisdom_type),
            colonies: .colonies
          }
        ]
      }
    ' "$observations_file" 2>/dev/null || echo '{"proposals":[]}')  # SUPPRESS:OK -- read-default: file may not exist yet

    # Check if we have any proposals
    proposal_count=$(echo "$proposals_json" | jq '.proposals | length')
    if [[ "$proposal_count" -eq 0 ]]; then
      json_ok '{"selected":[],"deferred":[],"count":0,"action":"none","reason":"no_proposals"}'
      exit 0
    fi

    # Display proposals
    if [[ "$dry_run" == "false" ]]; then
      if [[ "$verbose" == "true" ]]; then
        bash "$0" learning-display-proposals --verbose
      else
        bash "$0" learning-display-proposals
      fi
    fi

    # Capture user input (unless dry-run)
    selection=""
    if [[ "$dry_run" == "true" ]]; then
      # In dry-run mode, select all proposals
      selection=$(seq 1 $proposal_count | tr '\n' ' ')
      echo "Dry run: would select all $proposal_count proposals"
    else
      echo -n "Enter numbers to select (e.g., '1 3 5'), or press Enter to defer all: "
      read -r selection
    fi

    # Parse the selection
    parse_result=$(bash "$0" parse-selection "$selection" "$proposal_count")

    # Check for parse errors
    if ! echo "$parse_result" | jq -e '.ok' >/dev/null 2>&1; then  # SUPPRESS:OK -- validation: testing JSON field
      # Return the error
      echo "$parse_result"
      exit 1
    fi

    # Extract selected and deferred arrays
    selected_indices=$(echo "$parse_result" | jq -r '.result.selected // []')
    deferred_indices=$(echo "$parse_result" | jq -r '.result.deferred // []')
    action=$(echo "$parse_result" | jq -r '.result.action // "select"')
    selected_count=$(echo "$selected_indices" | jq 'length')
    deferred_count=$(echo "$deferred_indices" | jq 'length')

    # Show summary
    if [[ "$dry_run" == "false" ]]; then
      echo ""
      echo "$selected_count proposal(s) selected, $deferred_count deferred"
    fi

    # Preview and confirmation (if selections made and not skipping)
    confirmed=true
    if [[ "$selected_count" -gt 0 && "$dry_run" == "false" && "$skip_confirm" == "false" ]]; then
      echo ""
      echo "───────────────────────────────────────────────────"
      echo "📋 Selected for Promotion:"
      echo ""

      # Track below-threshold count for warning
      below_threshold_count=0

      # Display each selected proposal with full details
      echo "$selected_indices" | jq -r '.[]' | while read -r idx; do
        proposal=$(echo "$proposals_json" | jq -r ".proposals[$idx]")
        content=$(sanitize_read_value "$(echo "$proposal" | jq -r '.content')")
        ptype=$(echo "$proposal" | jq -r '.wisdom_type')
        count=$(echo "$proposal" | jq -r '.observation_count')
        threshold=$(echo "$proposal" | jq -r '.threshold')

        # Capitalize type for display
        type_display=$(echo "$ptype" | awk '{print toupper(substr($0,1,1)) tolower(substr($0,2))}')

        # Check if below threshold
        status=""
        if [[ "$count" -lt "$threshold" ]]; then
          status=" [⚠️ Early promotion - below threshold]"
          below_threshold_count=$((below_threshold_count + 1))
        fi

        echo "  • $type_display: \"$content\"$status"
      done

      # Show warning if any below threshold
      if [[ "$below_threshold_count" -gt 0 ]]; then
        echo ""
        echo "⚠️  $below_threshold_count item(s) below threshold will be early promoted"
      fi

      # Confirmation prompt
      echo ""
      echo -n "Proceed with promotion? (y/n): "
      read -r confirm_response

      if [[ ! "$confirm_response" =~ ^[Yy]$ ]]; then
        confirmed=false
        echo "Selection cancelled. Treating as defer-all."
        # Move all to deferred
        action="defer_all"
        deferred_indices=$(jq -n --argjson s "$selected_indices" --argjson d "$deferred_indices" '($s + $d)')
        selected_indices="[]"
        selected_count=0
        deferred_count=$(echo "$deferred_indices" | jq 'length')
      fi
    fi

    # Build result JSON
    result=$(jq -n \
      --argjson selected "$selected_indices" \
      --argjson deferred "$deferred_indices" \
      --argjson proposals "$proposals_json" \
      --arg action "$action" \
      --argjson count "$proposal_count" \
      --arg confirmed "$confirmed" \
      '{
        selected: $selected,
        deferred: $deferred,
        count: $count,
        action: $action,
        confirmed: ($confirmed == "true"),
        proposals: $proposals.proposals
      }')

    json_ok "$result"
}

# ============================================================================
# _learning_defer_proposals
# Store unselected proposals in learning-deferred.json for later review
# Usage: echo '[{proposal1}, {proposal2}]' | bash aether-utils.sh learning-defer-proposals
# Returns: JSON with count of newly deferred items
# ============================================================================
_learning_defer_proposals() {
    # Read proposals from stdin
    proposals_json=$(cat)

    # Validate input
    if [[ -z "$proposals_json" ]] || [[ "$proposals_json" == "[]" ]]; then
      json_ok '{"deferred":0,"new":0,"expired":0}'
      exit 0
    fi

    deferred_file="$COLONY_DATA_DIR/learning-deferred.json"

    # Ensure data directory exists
    [[ ! -d "$DATA_DIR" ]] && mkdir -p "$DATA_DIR"

    # Acquire lock
    acquire_lock "$deferred_file" 5 || {
      json_err "$E_LOCK_TIMEOUT" "Could not acquire lock on deferred file"
      exit 1
    }

    # Read existing deferred file or create empty structure
    if [[ -f "$deferred_file" ]]; then
      existing_deferred=$(jq '.deferred // []' "$deferred_file" 2>/dev/null || echo '[]')  # SUPPRESS:OK -- read-default: returns fallback if missing
    else
      existing_deferred='[]'
    fi

    # Current timestamp for new entries
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    current_epoch=$(date +%s)

    # Add deferred_at timestamp to each new proposal
    new_proposals=$(echo "$proposals_json" | jq --arg ts "$ts" '
      map(. + {deferred_at: $ts})
    ')

    # Calculate TTL cutoff (30 days ago)
    ttl_cutoff=$((current_epoch - 30 * 24 * 60 * 60))

    # Filter existing deferred: remove expired entries
    filtered_existing=$(echo "$existing_deferred" | jq --argjson cutoff "$ttl_cutoff" '
      map(select(
        (.deferred_at | sub("\\.[0-9]+Z$"; "Z") | fromdateiso8601) > $cutoff
      ))
    ' 2>/dev/null || echo '[]')  # SUPPRESS:OK -- read-default: returns fallback on failure

    # Count expired items
    expired_count=$(echo "$existing_deferred" | jq --argjson cutoff "$ttl_cutoff" '
      map(select(
        (.deferred_at | sub("\\.[0-9]+Z$"; "Z") | fromdateiso8601) <= $cutoff
      )) | length
    ' 2>/dev/null || echo '0')  # SUPPRESS:OK -- read-default: returns fallback on failure

    # Merge new proposals with existing, avoiding duplicates by content_hash
    merged=$(jq -s --argjson new "$new_proposals" '
      def unique_by_hash:
        group_by(.content_hash) | map(first);

      (.[0] // []) + $new | unique_by_hash
    ' <<< "$filtered_existing")

    # Count new items (those that weren't in existing)
    existing_hashes=$(echo "$filtered_existing" | jq -r 'map(.content_hash) | join(" ")')
    new_count=0
    if [[ -n "$existing_hashes" ]]; then
      new_count=$(echo "$new_proposals" | jq --arg existing "$existing_hashes" '
        [$existing | split(" ")[]] as $hashes |
        map(select(.content_hash as $h | $hashes | index($h) | not)) |
        length
      ')
    else
      new_count=$(echo "$new_proposals" | jq 'length')
    fi

    # Write atomically
    tmp_file="${deferred_file}.tmp.$$"
    jq -n --argjson deferred "$merged" '{deferred: $deferred}' > "$tmp_file"
    mv "$tmp_file" "$deferred_file"

    # Release lock
    release_lock

    # Log activity
    total_count=$(echo "$merged" | jq 'length')
    bash "$0" activity-log "DEFERRED" "Queen" "Stored $new_count new deferred proposal(s), $expired_count expired removed, $total_count total"

    json_ok "{\"deferred\":$total_count,\"new\":$new_count,\"expired\":$expired_count}"
}

# ============================================================================
# _learning_approve_proposals
# Orchestrate full approval workflow: one-at-a-time display with Approve/Reject/Skip
# Usage: learning-approve-proposals [--verbose] [--dry-run] [--yes] [--deferred]
# Returns: JSON summary {promoted, deferred, failed, undo_offered}
# ============================================================================
_learning_approve_proposals() {
    verbose=false
    dry_run=false
    skip_confirm=false
    deferred_mode=false
    undo_mode=false

    # Parse arguments
    for arg in "$@"; do
      case "$arg" in
        --verbose) verbose=true ;;
        --dry-run) dry_run=true ;;
        --yes) skip_confirm=true ;;
        --deferred) deferred_mode=true ;;
        --undo) undo_mode=true ;;
      esac
    done

    # Handle --undo mode
    if [[ "$undo_mode" == "true" ]]; then
      undo_result=$(bash "$0" learning-undo-promotions 2>&1)
      echo "$undo_result"
      exit 0
    fi

    # Get colony name via proper subcommand
    colony_name=$(bash "$0" colony-name 2>/dev/null | jq -r '.result.name // ""') || colony_name="unknown"
    [[ -z "$colony_name" ]] && colony_name="unknown"

    # Load proposals based on mode
    if [[ "$deferred_mode" == "true" ]]; then
      # Load from deferred file
      if [[ ! -f "$COLONY_DATA_DIR/learning-deferred.json" ]]; then
        echo "No deferred proposals to review."
        json_ok '{"promoted":0,"deferred":0,"failed":null,"undo_offered":false}'
        exit 0
      fi
      # SUPPRESS:OK -- read-default: returns fallback on failure
      proposals_json=$(jq '{proposals: .deferred}' "$COLONY_DATA_DIR/learning-deferred.json" 2>/dev/null || echo '{"proposals":[]}')
      echo "📦 Reviewing deferred proposals..."
      echo ""
    else
      # Get proposals directly from learning-check-promotion
      proposals_result=$(bash "$0" learning-check-promotion 2>/dev/null || echo '{"proposals":[]}')  # SUPPRESS:OK -- read-default: subcommand may fail
      proposals_json=$(echo "$proposals_result" | jq '{proposals: .result.proposals // []}')

      # Check if there were any proposals
      proposal_count=$(echo "$proposals_json" | jq '.proposals | length')
      if [[ "$proposal_count" -eq 0 ]]; then
        json_ok '{"promoted":0,"deferred":0,"failed":null,"undo_offered":false}'
        exit 0
      fi
    fi

    # Get proposal count
    proposal_count=$(echo "$proposals_json" | jq '.proposals | length')
    if [[ "$proposal_count" -eq 0 ]]; then
      echo "No proposals available."
      json_ok '{"promoted":0,"deferred":0,"failed":null,"undo_offered":false}'
      exit 0
    fi

    # Define wisdom type emojis and names for display
    declare -A type_emojis
    declare -A type_names
    type_emojis=(
      ["philosophy"]="📜"
      ["pattern"]="🧭"
      ["redirect"]="⚠️"
      ["stack"]="🔧"
      ["decree"]="🏛️"
      ["failure"]="❌"
    )
    type_names=(
      ["philosophy"]="Philosophy"
      ["pattern"]="Pattern"
      ["redirect"]="Redirect"
      ["stack"]="Stack Wisdom"
      ["decree"]="Decree"
      ["failure"]="Failure"
    )

    # Arrays to track results
    approved_proposals=()
    rejected_proposals=()
    skipped_proposals=()

    # Process proposals one at a time
    echo ""
    echo "🧠 Wisdom Promotion Review"
    echo "══════════════════════════"
    echo ""
    echo "$proposal_count proposal(s) ready for review"
    echo ""

    for ((i=0; i<proposal_count; i++)); do
      proposal=$(echo "$proposals_json" | jq ".proposals[$i]")
      ptype=$(echo "$proposal" | jq -r '.wisdom_type')
      content=$(sanitize_read_value "$(echo "$proposal" | jq -r '.content')")
      count=$(echo "$proposal" | jq -r '.observation_count // 1')
      threshold=$(echo "$proposal" | jq -r '.threshold // 1')

      emoji="${type_emojis[$ptype]:-📝}"
      name="${type_names[$ptype]:-$ptype}"

      # Display proposal
      echo "───────────────────────────────────────────────────"
      echo "Proposal $((i+1)) of $proposal_count"
      echo "───────────────────────────────────────────────────"
      echo ""
      echo "$emoji $name (observed $count time(s), threshold: $threshold)"
      echo ""
      echo "$content"
      echo ""
      echo "───────────────────────────────────────────────────"

      # Handle dry-run mode
      if [[ "$dry_run" == "true" ]]; then
        echo "Dry run: would approve"
        approved_proposals+=("$proposal")
        echo ""
        continue
      fi

      # Handle --yes mode (auto-approve all)
      if [[ "$skip_confirm" == "true" ]]; then
        approved_proposals+=("$proposal")
        echo "✓ Auto-approved (--yes mode)"
        echo ""
        continue
      fi

      # Prompt for action
      echo -n "[A]pprove  [R]eject  [S]kip  Your choice: "
      read -r choice

      case "$choice" in
        [Aa]|"approve"|"Approve")
          approved_proposals+=("$proposal")
          echo "✓ Approved"
          ;;
        [Rr]|"reject"|"Reject")
          rejected_proposals+=("$proposal")
          echo "✗ Rejected"
          ;;
        [Ss]|""|"skip"|"Skip")
          skipped_proposals+=("$proposal")
          echo "→ Skipped"
          ;;
        *)
          # Invalid input - default to skip
          skipped_proposals+=("$proposal")
          echo "→ Skipped (invalid input)"
          ;;
      esac
      echo ""
    done

    # Execute promotions for approved proposals
    promoted_count=0
    failed_item=""
    promoted_items=()

    if [[ ${#approved_proposals[@]} -gt 0 ]]; then
      echo ""
      echo "Promoting ${#approved_proposals[@]} observation(s)..."
      echo ""

      for proposal in "${approved_proposals[@]}"; do
        ptype=$(echo "$proposal" | jq -r '.wisdom_type')
        content=$(sanitize_read_value "$(echo "$proposal" | jq -r '.content')")

        if [[ "$dry_run" == "true" ]]; then
          echo "Dry run: would promote $ptype: \"$content\""
          ((promoted_count++))
          promoted_items+=("$proposal")
          continue
        fi

        # Call queen-promote
        promote_result=$(bash "$0" queen-promote "$ptype" "$content" "$colony_name" 2>&1) || {
          echo "✗ Failed to promote: $content"
          echo "  Error: $promote_result"
          failed_item="$content"
          # Prompt for retry on failure
          echo ""
          echo -n "Write to QUEEN.md failed. Retry? (y/n): "
          read -r retry_response
          if [[ "$retry_response" =~ ^[Yy]$ ]]; then
            # Retry once
            promote_result=$(bash "$0" queen-promote "$ptype" "$content" "$colony_name" 2>&1) || {
              echo "✗ Retry failed. Keeping proposal pending."
              skipped_proposals+=("$proposal")
              continue
            }
          else
            echo "Skipping this proposal. It will remain pending."
            skipped_proposals+=("$proposal")
            continue
          fi
        }

        echo "✓ Promoted ${ptype^}: \"$content\""
        ((promoted_count++))
        promoted_items+=("$proposal")
      done
    fi

    # Handle deferred proposals (skipped ones go to deferred)
    deferred_count=${#skipped_proposals[@]}
    if [[ "$dry_run" == "false" ]] && [[ $deferred_count -gt 0 ]]; then
      # Convert skipped proposals to JSON array and defer
      skipped_json=$(printf '%s\n' "${skipped_proposals[@]}" | jq -s '.')
      echo "$skipped_json" | bash "$0" learning-defer-proposals >/dev/null 2>&1 || _aether_log_error "Could not defer learning proposals"
    fi

    # Log activity
    if [[ "$dry_run" == "false" ]]; then
      bash "$0" activity-log "PROMOTED" "Queen" "Promoted $promoted_count observation(s), deferred $deferred_count, rejected ${#rejected_proposals[@]}"
    fi

    # Display summary
    echo ""
    echo "═══════════════════════════════════════════════════"
    echo "Summary: $promoted_count approved, ${#rejected_proposals[@]} rejected, $deferred_count skipped"
    echo "═══════════════════════════════════════════════════"
    echo ""

    # Offer undo if promotions succeeded
    undo_offered=false
    if [[ "$promoted_count" -gt 0 ]] && [[ "$dry_run" == "false" ]] && [[ -z "$failed_item" ]]; then
      undo_offered=true

      # Store undo info
      undo_file="$COLONY_DATA_DIR/.promotion-undo.json"
      promoted_json=$(printf '%s\n' "${promoted_items[@]}" | jq -s '.')
      jq -n --argjson items "$promoted_json" --arg ts "$(date +%s)" '{promoted: $items, timestamp: ($ts | tonumber)}' > "$undo_file"

      echo -n "Undo these promotions? (y/n): "
      read -r undo_response

      if [[ "$undo_response" =~ ^[Yy]$ ]]; then
        echo "Reverting promotions..."
        undo_result=$(bash "$0" learning-undo-promotions 2>&1)
        if echo "$undo_result" | jq -e '.ok' >/dev/null 2>&1; then  # SUPPRESS:OK -- validation: testing JSON field
          undone_count=$(echo "$undo_result" | jq -r '.result.undone // 0')
          echo "$undone_count promotion(s) reverted."
          promoted_count=0
        else
          echo "Undo failed: $(echo "$undo_result" | jq -r '.error.message // "Unknown error"')"
        fi
      else
        echo "Promotions kept."
      fi
    fi

    # Build result
    result=$(jq -n \
      --argjson promoted "$promoted_count" \
      --argjson deferred "$deferred_count" \
      --argjson rejected "${#rejected_proposals[@]}" \
      --arg failed "${failed_item:-null}" \
      --argjson undo "$undo_offered" \
      '{promoted: $promoted, deferred: $deferred, rejected: $rejected, failed: $failed, undo_offered: $undo}')

    json_ok "$result"
}

# ============================================================================
# _learning_undo_promotions
# Revert promotions from QUEEN.md using undo file
# Usage: learning-undo-promotions
# Returns: JSON with count of undone items
# ============================================================================
_learning_undo_promotions() {
    undo_file="$COLONY_DATA_DIR/.promotion-undo.json"

    # Check if undo file exists
    if [[ ! -f "$undo_file" ]]; then
      json_err "$E_FILE_NOT_FOUND" "No undo file found. Cannot undo promotions."
      exit 1
    fi

    # Read undo data
    undo_data=$(cat "$undo_file")
    undo_timestamp=$(echo "$undo_data" | jq -r '.timestamp // 0')
    current_time=$(date +%s)

    # Check 24h TTL
    ttl_seconds=$((24 * 60 * 60))
    time_diff=$((current_time - undo_timestamp))

    if [[ $time_diff -gt $ttl_seconds ]]; then
      # Remove expired undo file
      rm -f "$undo_file"
      json_err "$E_VALIDATION_FAILED" "Undo window expired (24h limit)"
      exit 1
    fi

    queen_file="$AETHER_ROOT/.aether/QUEEN.md"

    # Check if QUEEN.md exists
    if [[ ! -f "$queen_file" ]]; then
      json_err "$E_FILE_NOT_FOUND" "QUEEN.md not found"
      exit 1
    fi

    # Process each promoted item
    undone_count=0
    failed_items=()

    # Read promoted items from undo file
    promoted_items=$(echo "$undo_data" | jq -c '.promoted[]?')

    if [[ -z "$promoted_items" ]]; then
      rm -f "$undo_file"
      json_err "$E_VALIDATION_FAILED" "No promoted items in undo file"
      exit 1
    fi

    # Create temp file for atomic write
    tmp_file="${queen_file}.tmp.$$"

    # Copy current QUEEN.md to temp
    cp "$queen_file" "$tmp_file"

    # Process each item
    while IFS= read -r item; do
      [[ -z "$item" ]] && continue

      ptype=$(echo "$item" | jq -r '.wisdom_type')
      content=$(sanitize_read_value "$(echo "$item" | jq -r '.content')")

      # Map type to section header
      case "$ptype" in
        philosophy) section_header="## 📜 Philosophies" ;;
        pattern) section_header="## 🧭 Patterns" ;;
        redirect) section_header="## ⚠️ Redirects" ;;
        stack) section_header="## 🔧 Stack Wisdom" ;;
        decree) section_header="## 🏛️ Decrees" ;;
        *) continue ;;
      esac

      # Escape content for sed (basic escaping)
      escaped_content=$(echo "$content" | sed 's/[\\/&]/\\&/g')

      # Find and remove the entry from the section
      # Pattern: - **colony_name** (timestamp): content
      # We match based on content since that's the unique part
      if grep -q "${escaped_content}" "$tmp_file" 2>/dev/null; then  # SUPPRESS:OK -- existence-test: file may not exist
        # Remove line containing this content within the section
        # Use awk to handle section-aware removal
        awk -v section="$section_header" -v content="$content" '
          BEGIN { in_section = 0 }
          $0 == section { in_section = 1 }
          in_section && $0 ~ /^## / && $0 != section { in_section = 0 }
          in_section && $0 ~ content { skip = 1; next }
          { if (!skip) print; skip = 0 }
        ' "$tmp_file" > "${tmp_file}.new" && mv "${tmp_file}.new" "$tmp_file"

        ((undone_count++))
      else
        # Entry already removed or not found
        failed_items+=("$content")
      fi
    done <<< "$promoted_items"

    # Update METADATA stats in temp file - decrement counts
    case "$ptype" in
      stack) stat_key="total_stack_entries" ;;
      philosophy) stat_key="total_philosophies" ;;
      *) stat_key="total_${ptype}s" ;;
    esac

    # Decrement stats (but not below 0)
    current_count=$(grep "\"${stat_key}\":" "$tmp_file" 2>/dev/null | grep -o '[0-9]*' | head -1 || echo "0")  # SUPPRESS:OK -- read-default: file may not exist
    current_count=${current_count:-0}
    if [[ $current_count -gt 0 ]]; then
      new_count=$((current_count - 1))
      awk -v type="$stat_key" -v count="$new_count" '{
        gsub("\"" type "\": [0-9]*", "\"" type "\": " count)
        print
      }' "$tmp_file" > "${tmp_file}.stats" && mv "${tmp_file}.stats" "$tmp_file"
    fi

    # Atomic move
    mv "$tmp_file" "$queen_file"

    # Remove undo file after successful revert
    rm -f "$undo_file"

    # Log activity
    bash "$0" activity-log "UNDONE" "Queen" "Reverted $undone_count promotion(s) from QUEEN.md"

    # Build result
    if [[ ${#failed_items[@]} -gt 0 ]]; then
      failed_json=$(printf '%s\n' "${failed_items[@]}" | jq -R . | jq -s '.')
      result=$(jq -n --argjson undone "$undone_count" --argjson failed "$failed_json" '{undone: $undone, not_found: $failed}')
    else
      result=$(jq -n --argjson undone "$undone_count" '{undone: $undone, not_found: []}')
    fi

    json_ok "$result"
}

# ============================================================================
# _instinct_read
# Read learned instincts from COLONY_STATE.json memory
# Migrated to state-api facade: uses _state_read_field for read-only access
# Usage: instinct-read [--min-confidence N] [--max N] [--domain DOMAIN]
# Returns: JSON with filtered, confidence-sorted instincts
# ============================================================================
_instinct_read() {
    ir_min_confidence="0.5"
    ir_max="5"
    ir_domain=""

    # Parse flags from positional args
    ir_shift=1
    while [[ $ir_shift -le $# ]]; do
      eval "ir_arg=\${$ir_shift}"
      ir_shift=$((ir_shift + 1))
      case "$ir_arg" in
        --min-confidence)
          eval "ir_min_confidence=\${$ir_shift}"
          ir_shift=$((ir_shift + 1))
          ;;
        --max)
          eval "ir_max=\${$ir_shift}"
          ir_shift=$((ir_shift + 1))
          ;;
        --domain)
          eval "ir_domain=\${$ir_shift}"
          ir_shift=$((ir_shift + 1))
          ;;
      esac
    done

    # Read full state via facade
    ir_state=$(_state_read_field '.')
    if [[ -z "$ir_state" ]]; then
      json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found. Run /ant:init first."
    fi

    # Check if memory.instincts exists
    ir_has_instincts=$(echo "$ir_state" | jq 'if .memory.instincts then "yes" else "no" end' 2>/dev/null || echo "no")
    if [[ "$ir_has_instincts" != '"yes"' ]]; then
      json_ok '{"instincts":[],"total":0,"filtered":0}'
      exit 0
    fi

    ir_result=$(echo "$ir_state" | jq -c \
      --argjson min_conf "$ir_min_confidence" \
      --argjson max_count "$ir_max" \
      --arg domain_filter "$ir_domain" \
      '
      (.memory.instincts // []) as $all |
      ($all | length) as $total |
      $all
      | map(select(
          (.confidence // 0) >= $min_conf
          and (.status // "hypothesis") != "disproven"
          and (if $domain_filter != "" then (.domain // "") == $domain_filter else true end)
        ))
      | sort_by(-.confidence)
      | .[:$max_count]
      | {
          instincts: .,
          total: $total,
          filtered: (. | length)
        }
      ' 2>/dev/null)

    if [[ -z "$ir_result" || "$ir_result" == "null" ]]; then
      json_ok '{"instincts":[],"total":0,"filtered":0}'
    else
      json_ok "$ir_result"
    fi
}

# ============================================================================
# _normalize_text
# Canonical text form for fuzzy comparison: lowercase, strip punctuation,
# collapse whitespace, synonym substitution, stop word removal
# Usage: _normalize_text "When Implementing Tests"
# Output: stdout (e.g., "writing testing")
# ============================================================================
_normalize_text() {
    local text="$1"

    # Guard: empty input
    [[ -z "$text" ]] && echo "" && return 0

    # Lowercase
    text=$(echo "$text" | tr '[:upper:]' '[:lower:]')

    # Strip punctuation (keep alphanumeric, spaces, hyphens)
    text=$(echo "$text" | tr -cd '[:alnum:][:space:]-')

    # Collapse whitespace
    text=$(echo "$text" | awk '{$1=$1};1')

    # Synonym substitution + stop word removal via awk
    text=$(echo "$text" | awk 'BEGIN {
        syn["implementing"] = "writing"; syn["creating"] = "writing"; syn["building"] = "writing";
        syn["implement"] = "writing"; syn["create"] = "writing"; syn["build"] = "writing";
        syn["write"] = "writing";
        syn["tests"] = "testing"; syn["checking"] = "testing"; syn["verifying"] = "testing";
        syn["fixing"] = "resolving"; syn["repairing"] = "resolving"; syn["patching"] = "resolving";
        syn["fix"] = "resolving"; syn["repair"] = "resolving"; syn["patch"] = "resolving";
        syn["resolve"] = "resolving"
    }
    {
        n = split($0, words, " ")
        out = 0
        for (i = 1; i <= n; i++) {
            w = words[i]
            if (w == "") continue
            if (w in syn) w = syn[w]
            # Stop words: when, while, during, before, after
            if (w == "when" || w == "while" || w == "during" || w == "before" || w == "after") continue
            printf "%s%s", (out > 0 ? " " : ""), w
            out++
        }
        printf "\n"
    }')

    echo "$text"
}

# ============================================================================
# _jaccard_similarity
# Word-level Jaccard similarity between two strings
# Usage: _jaccard_similarity "when writing tests" "when implementing tests"
# Output: stdout (e.g., "0.80")
# ============================================================================
_jaccard_similarity() {
    local text_a="$1"
    local text_b="$2"

    # Normalize both texts
    local norm_a norm_b
    norm_a=$(_normalize_text "$text_a")
    norm_b=$(_normalize_text "$text_b")

    # Guard: empty after normalization
    [[ -z "$norm_a" || -z "$norm_b" ]] && echo "0.00" && return 0

    # Compute Jaccard via awk using NUL delimiter between the two texts
    # Both texts are already normalized (no newlines, no special chars)
    printf '%s\037%s\n' "$norm_a" "$norm_b" | awk -F'\037' '
    {
        split($1, a_words, " ")
        split($2, b_words, " ")

        # Build set A
        for (i in a_words) if (a_words[i] != "") set_a[a_words[i]] = 1

        # Build set B
        for (i in b_words) if (b_words[i] != "") set_b[b_words[i]] = 1

        # Compute intersection and union
        intersection = 0
        union = 0
        for (key in set_a) {
            union++
            if (key in set_b) intersection++
        }
        for (key in set_b) {
            if (!(key in set_a)) union++
        }

        # Guard: avoid division by zero
        if (union == 0) { printf "0.00\n"; exit }

        printf "%.2f\n", intersection / union
    }'
}

# ============================================================================
# _instinct_create
# Create or update an instinct in COLONY_STATE.json
# Migrated to state-api facade: uses _state_read_field for reads, _state_mutate for atomic writes
# Usage: instinct-create --trigger "when X" --action "do Y" --confidence 0.5 --domain "architecture" --source "phase-3" --evidence "observation"
# Deduplicates: if trigger+action matches existing instinct, boosts confidence instead
# Fuzzy dedup: if trigger AND action both have >= 0.80 Jaccard similarity, merges into best match
# Cap: max 30 instincts, evicts lowest confidence when exceeded
# ============================================================================
_instinct_create() {
    ic_trigger=""
    ic_action=""
    ic_confidence="0.5"
    ic_domain="workflow"
    ic_source=""
    ic_evidence=""

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --trigger)    ic_trigger="$2"; shift 2 ;;
        --action)     ic_action="$2"; shift 2 ;;
        --confidence) ic_confidence="$2"; shift 2 ;;
        --domain)     ic_domain="$2"; shift 2 ;;
        --source)     ic_source="$2"; shift 2 ;;
        --evidence)   ic_evidence="$2"; shift 2 ;;
        *) shift ;;
      esac
    done

    [[ -z "$ic_trigger" ]] && json_err "$E_VALIDATION_FAILED" "instinct-create requires --trigger"
    [[ -z "$ic_action" ]] && json_err "$E_VALIDATION_FAILED" "instinct-create requires --action"

    # Validate confidence range
    if ! [[ "$ic_confidence" =~ ^(0(\.[0-9]+)?|1(\.0+)?)$ ]]; then
      ic_confidence="0.5"
    fi

    ic_now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    ic_epoch=$(date +%s)
    ic_id="instinct_${ic_epoch}"

    # Check for existing instinct with matching trigger+action via facade
    ic_existing=$(_state_read_field "$(printf '[(.memory.instincts // [])[] | select(.trigger == "%s" and .action == "%s")] | first // null' "$ic_trigger" "$ic_action")")

    if [[ -n "$ic_existing" && "$ic_existing" != "null" ]]; then
      # Update existing: boost confidence by +0.1, increment applications
      IC_TRIGGER="$ic_trigger" IC_ACTION="$ic_action" IC_NOW="$ic_now" \
        _state_mutate '
          .memory.instincts = [
            (.memory.instincts // [])[] |
            if .trigger == env.IC_TRIGGER and .action == env.IC_ACTION then
              .confidence = ([(.confidence + 0.1), 1.0] | min) |
              .applications = ((.applications // 0) + 1) |
              .last_applied = env.IC_NOW
            else
              .
            end
          ]
        ' >/dev/null

      # Read updated confidence
      ic_new_conf=$(_state_read_field "$(printf '[(.memory.instincts // [])[] | select(.trigger == "%s" and .action == "%s")] | first | .confidence // 0' "$ic_trigger" "$ic_action")")
      json_ok "{\"instinct_id\":\"existing\",\"action\":\"updated\",\"confidence\":$ic_new_conf}"
    else
      # --- Fuzzy dedup: check for semantically similar instinct ---
      ic_all_instincts=$(_state_read_field '.memory.instincts // []')
      ic_fuzzy_match=""

      if [[ -n "$ic_all_instincts" && "$ic_all_instincts" != "null" && "$ic_all_instincts" != "[]" ]]; then
        ic_best_sim="0.00"
        ic_best_id=""
        ic_best_conf=""

        # Iterate over existing instincts to find best fuzzy match
        while IFS= read -r ic_line; do
          [[ -z "$ic_line" ]] && continue
          ic_ex_trigger=$(echo "$ic_line" | jq -r '.trigger // empty')
          ic_ex_action=$(echo "$ic_line" | jq -r '.action // empty')
          ic_ex_id=$(echo "$ic_line" | jq -r '.id // empty')
          ic_ex_conf=$(echo "$ic_line" | jq -r '.confidence // 0')

          [[ -z "$ic_ex_trigger" || -z "$ic_ex_action" || "$ic_ex_trigger" == "null" || "$ic_ex_action" == "null" ]] && continue

          # Compute Jaccard similarity for trigger and action independently
          ic_trig_sim=$(_jaccard_similarity "$ic_trigger" "$ic_ex_trigger")
          ic_act_sim=$(_jaccard_similarity "$ic_action" "$ic_ex_action")

          # Both must exceed 0.80 threshold
          if (( $(echo "$ic_trig_sim >= 0.80" | bc -l) )) && (( $(echo "$ic_act_sim >= 0.80" | bc -l) )); then
            # Pick highest similarity; tie-break by higher confidence
            ic_combined=$(echo "$ic_trig_sim + $ic_act_sim" | bc -l)
            ic_best_combined=$(echo "${ic_best_sim:-0} + 0" | bc -l)
            ic_best_conf_num="${ic_best_conf:-0}"
            if (( $(echo "$ic_combined > $ic_best_combined" | bc -l) )) || \
               (( $(echo "$ic_combined == $ic_best_combined && $ic_ex_conf >= $ic_best_conf_num" | bc -l) )); then
              ic_best_sim="$ic_combined"
              ic_best_id="$ic_ex_id"
              ic_best_conf="$ic_ex_conf"
              ic_fuzzy_match="$ic_line"
            fi
          fi
        done < <(echo "$ic_all_instincts" | jq -c '.[]')
      fi

      if [[ -n "$ic_fuzzy_match" ]]; then
        # Merge into best matching instinct
        ic_ex_conf_num=$(echo "$ic_fuzzy_match" | jq -r '.confidence // 0')
        ic_ex_evidence=$(echo "$ic_fuzzy_match" | jq -c '.evidence // []')
        ic_ex_trigger=$(echo "$ic_fuzzy_match" | jq -r '.trigger')
        ic_ex_action=$(echo "$ic_fuzzy_match" | jq -r '.action')

        # Average confidences (use printf to ensure leading zero for valid JSON)
        ic_new_conf=$(printf "%.2f" "$(echo "scale=4; ($ic_ex_conf_num + $ic_confidence) / 2" | bc -l)")
        # Keep longer trigger
        ic_merged_trigger="$ic_ex_trigger"
        [[ ${#ic_trigger} -gt ${#ic_merged_trigger} ]] && ic_merged_trigger="$ic_trigger"
        # Keep longer action
        ic_merged_action="$ic_ex_action"
        [[ ${#ic_action} -gt ${#ic_merged_action} ]] && ic_merged_action="$ic_action"

        # Build evidence array: existing + new
        if [[ "$ic_evidence" != "" && "$ic_evidence" != "null" ]]; then
          ic_merged_evidence=$(echo "$ic_ex_evidence" | jq --arg ev "$ic_evidence" '. + [$ev]')
        else
          ic_merged_evidence="$ic_ex_evidence"
        fi

        IC_FUZZY_ID="$ic_best_id" IC_MERGED_TRIGGER="$ic_merged_trigger" IC_MERGED_ACTION="$ic_merged_action" \
        IC_NEW_CONF="$ic_new_conf" IC_MERGED_EVIDENCE="$ic_merged_evidence" IC_NOW="$ic_now" \
          _state_mutate '
            .memory.instincts = [
              (.memory.instincts // [])[] |
              if .id == env.IC_FUZZY_ID then
                .trigger = env.IC_MERGED_TRIGGER |
                .action = env.IC_MERGED_ACTION |
                .confidence = (env.IC_NEW_CONF | tonumber) |
                .evidence = (env.IC_MERGED_EVIDENCE | fromjson) |
                .applications = ((.applications // 0) + 1) |
                .last_applied = env.IC_NOW
              else
                .
              end
            ]
          ' >/dev/null

        json_ok "$(jq -n --arg iid "$ic_best_id" --argjson conf "$ic_new_conf" '{instinct_id: $iid, action: "merged", confidence: $conf}')"
        exit 0
      fi

      # Create new instinct via _state_mutate (handles locking and backup)
      IC_ID="$ic_id" IC_TRIGGER="$ic_trigger" IC_ACTION="$ic_action" IC_CONFIDENCE="$ic_confidence" \
      IC_DOMAIN="$ic_domain" IC_SOURCE="$ic_source" IC_EVIDENCE="$ic_evidence" IC_NOW="$ic_now" \
        _state_mutate '
          .memory.instincts = (
            ((.memory.instincts // []) + [{
              id: env.IC_ID,
              trigger: env.IC_TRIGGER,
              action: env.IC_ACTION,
              confidence: (env.IC_CONFIDENCE | tonumber),
              status: "hypothesis",
              domain: env.IC_DOMAIN,
              source: env.IC_SOURCE,
              evidence: [env.IC_EVIDENCE],
              tested: false,
              created_at: env.IC_NOW,
              last_applied: null,
              applications: 0,
              successes: 0,
              failures: 0
            }])
            | sort_by(-.confidence)
            | .[:30]
          )
        ' >/dev/null

      json_ok "$(jq -n --arg iid "$ic_id" --argjson conf "$ic_confidence" '{instinct_id: $iid, action: "created", confidence: $conf}')"
    fi
    exit 0
}

# ============================================================================
# _instinct_apply
# Record when an instinct was actually used in practice
# Migrated to state-api facade: uses _state_read_field for reads, _state_mutate for atomic writes
# Usage: instinct-apply --id <instinct_id> [--outcome success|failure]
# Success: boosts confidence by 0.05 (cap 1.0), increments successes
# Failure: reduces confidence by 0.1 (floor 0.1), increments failures
# ============================================================================
_instinct_apply() {
    ia_id=""
    ia_outcome="success"

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --id)      ia_id="$2"; shift 2 ;;
        --outcome) ia_outcome="$2"; shift 2 ;;
        *) shift ;;
      esac
    done

    [[ -z "$ia_id" ]] && json_err "$E_VALIDATION_FAILED" "instinct-apply requires --id"

    # Validate outcome
    if [[ "$ia_outcome" != "success" && "$ia_outcome" != "failure" ]]; then
      json_err "$E_VALIDATION_FAILED" "instinct-apply --outcome must be 'success' or 'failure'"
    fi

    # Check instinct exists via facade
    ia_exists=$(_state_read_field "$(printf '[(.memory.instincts // [])[] | select(.id == "%s")] | length > 0' "$ia_id")")
    if [[ "$ia_exists" != "true" ]]; then
      json_err "$E_RESOURCE_NOT_FOUND" "Instinct '$ia_id' not found"
    fi

    ia_now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Update the instinct based on outcome via _state_mutate (handles locking and backup)
    if [[ "$ia_outcome" == "success" ]]; then
      IA_ID="$ia_id" IA_NOW="$ia_now" \
        _state_mutate '
          .memory.instincts = [
            (.memory.instincts // [])[] |
            if .id == env.IA_ID then
              .applications = ((.applications // 0) + 1) |
              .successes = ((.successes // 0) + 1) |
              .confidence = ([(.confidence + 0.05), 1.0] | min) |
              .last_applied = env.IA_NOW
            else
              .
            end
          ]
        ' >/dev/null
    else
      IA_ID="$ia_id" IA_NOW="$ia_now" \
        _state_mutate '
          .memory.instincts = [
            (.memory.instincts // [])[] |
            if .id == env.IA_ID then
              .applications = ((.applications // 0) + 1) |
              .failures = ((.failures // 0) + 1) |
              .confidence = ([(.confidence - 0.1), 0.1] | max) |
              .last_applied = env.IA_NOW
            else
              .
            end
          ]
        ' >/dev/null
    fi

    # Extract updated values for response via facade
    ia_new_apps=$(_state_read_field "$(printf '[(.memory.instincts // [])[] | select(.id == "%s")] | first | .applications' "$ia_id")")
    ia_new_conf=$(_state_read_field "$(printf '[(.memory.instincts // [])[] | select(.id == "%s")] | first | .confidence' "$ia_id")")

    json_ok "$(jq -n --arg iid "$ia_id" --argjson apps "$ia_new_apps" --argjson conf "$ia_new_conf" '{applied: true, instinct_id: $iid, applications: $apps, new_confidence: $conf}')"
    exit 0
}

# ============================================================================
# _learning_extract_fallback
# Deterministic fallback for learning extraction when builders skip learning output.
# Produces structured learning objects from git diff and feeds them through instinct-create.
# Usage: learning-extract-fallback (no args -- reads git state and colony state)
# Returns: JSON {"learnings": [...], "count": N}
# Note: Uses jq for grouping/sorting (bash 3.2 compatible -- no associative arrays)
# ============================================================================
_learning_extract_fallback() {
    # Pre-flight: verify git history exists
    if ! git rev-parse HEAD~1 >/dev/null 2>&1; then
        json_ok '{"learnings":[],"count":0}'
        exit 0
    fi

    # Pre-flight: verify colony state exists
    if [[ ! -f "$DATA_DIR/COLONY_STATE.json" ]]; then
        json_ok '{"learnings":[],"count":0}'
        exit 0
    fi

    # Read git diff data
    lef_stat=$(git diff --stat HEAD~1 2>/dev/null || echo "")
    lef_files=$(git diff --name-only HEAD~1 2>/dev/null || echo "")

    # Guard: no changes
    if [[ -z "$lef_files" ]]; then
        json_ok '{"learnings":[],"count":0}'
        exit 0
    fi

    # Read last-build-claims.json if it exists for task context
    lef_claims=""
    if [[ -f "$COLONY_DATA_DIR/last-build-claims.json" ]]; then
        lef_claims=$(cat "$COLONY_DATA_DIR/last-build-claims.json" 2>/dev/null || echo "")
    fi

    # Parse stat data into JSON: extract file path, insertions, deletions per line
    # git diff --stat format: " path/to/file | 42 +++++---" or " path/to/file | 10"
    # Also include the full file list for files with 0 insertions (binary/new files)
    lef_files_json=$(printf '%s\n' "$lef_stat" | awk '
    {
        # Skip empty lines
        if ($0 == "") next
        # Skip the summary line (e.g., "13 files changed, 693 insertions(+), 128 deletions(-)")
        if ($0 ~ /files? changed/) next
        # Split on " | " to get file path and stat
        idx = index($0, " | ")
        if (idx > 0) {
            fpath = substr($0, 1, idx - 1)
            rest = substr($0, idx + 3)
            # Extract first two numbers from rest (insertions, deletions)
            gsub(/[^0-9 ]/, "", rest)
            n = split(rest, nums, " ")
            ins = (nums[1] != "" ? nums[1] : 0)
            del = (nums[2] != "" ? nums[2] : 0)
        } else {
            fpath = $0
            ins = 0
            del = 0
        }
        # Trim whitespace from path (git stat pads with trailing spaces)
        gsub(/^[[:space:]]+|[[:space:]]+$/, "", fpath)
        if (fpath != "") printf "{\"path\":\"%s\",\"ins\":%d,\"del\":%d}\n", fpath, ins, del
    }' | jq -sc '.')

    # Filter noise files and categorize using jq
    lef_filtered=$(echo "$lef_files_json" | jq '[.[] |
        select((.path | startswith(".aether/data/")) | not) |
        select((.path | startswith(".aether/dreams/")) | not) |
        select(.path != "package-lock.json") |
        select((.path | startswith("node_modules/")) | not) |
        # Categorize (use extra parens around `or` expressions for jq parser)
        .category = (
            if ((.path | test("^(tests|test)/")) or (.path | test("_test\\.")) or (.path | test("\\.test\\.")) or (.path | test("\\.spec\\.")))
            then "testing"
            elif ((.path | startswith("src/")) or (.path | startswith("lib/")))
            then "source"
            elif ((.path | startswith("docs/")) or (.path | endswith(".md")))
            then "documentation"
            elif (.path | test("\\.(json|yaml|yml|toml|env|conf|config)$"))
            then "configuration"
            else "source"
            end
        ) |
        # Calculate absolute net change
        .abs_net = ((.ins - .del) | if . < 0 then -. else . end) |
        # Determine if test file
        .is_test = (.category == "testing") |
        # Skip trivial non-test changes (abs_net < 3)
        select((.is_test == true) or (.abs_net >= 3)) |
        # Extract directory
        .dir = (.path | split("/") | .[0:-1] | join("/"))
    ]')

    # Guard: no significant changes after filtering
    lef_sig_count=$(echo "$lef_filtered" | jq 'length')
    if [[ "$lef_sig_count" -eq 0 ]]; then
        json_ok '{"learnings":[],"count":0}'
        exit 0
    fi

    # Group by category, sort by total change magnitude, cap at 5
    # Using jq for grouping (bash 3.2 compatible)
    lef_categories=$(echo "$lef_filtered" | jq -r '
        group_by(.category) |
        map({
            category: .[0].category,
            file_count: length,
            total_ins: (map(.ins) | add),
            total_del: (map(.del) | add),
            total_change: ((map(.ins) | add) + (map(.del) | add)),
            dir: (map(.dir) | group_by(.) | map({d: .[0], c: length}) | sort_by(-.c) | .[0].d)
        }) |
        sort_by(-.total_change) |
        .[:5]
    ')

    # Get current phase for source tag
    lef_phase=$(_state_read_field '.current_phase // 0')
    lef_phase=${lef_phase:-0}

    # Generate learning objects
    lef_learnings="[]"
    lef_count=0

    lef_cat_count=$(echo "$lef_categories" | jq 'length')
    for ((i=0; i<lef_cat_count && i<5; i++)); do
        lef_cat_data=$(echo "$lef_categories" | jq ".[$i]")
        lef_cat=$(echo "$lef_cat_data" | jq -r '.category')
        lef_fcount=$(echo "$lef_cat_data" | jq -r '.file_count')
        lef_fins=$(echo "$lef_cat_data" | jq -r '.total_ins')
        lef_fdel=$(echo "$lef_cat_data" | jq -r '.total_del')
        lef_fdir=$(echo "$lef_cat_data" | jq -r '.dir')

        # Build fact string
        lef_fact="Modified $lef_fcount files in $lef_fdir ($lef_fins+/$lef_fdel- lines)"

        # Build interpretation based on category
        case "$lef_cat" in
            testing)
                if [[ "$lef_fdel" == "0" ]]; then
                    lef_interp="Added tests, improving test coverage"
                else
                    lef_interp="Modified tests, likely improving test coverage"
                fi
                ;;
            source)
                if [[ "$lef_fdel" -gt "$lef_fins" ]]; then
                    lef_interp="Refactored or simplified $lef_fdir module code"
                else
                    lef_interp="Extended or modified $lef_fdir module code"
                fi
                ;;
            configuration)
                lef_interp="Updated project configuration"
                ;;
            documentation)
                lef_interp="Updated project documentation"
                ;;
            *)
                lef_interp="Modified $lef_fdir files"
                ;;
        esac

        # Build trigger and action for instinct-create
        lef_trigger="when working on $lef_cat"
        lef_action="$lef_interp"

        # Escape strings for JSON
        lef_fact_json=$(echo "$lef_fact" | jq -Rs '.')
        lef_interp_json=$(echo "$lef_interp" | jq -Rs '.')

        # Append to learnings array
        lef_learnings=$(echo "$lef_learnings" | jq --arg trigger "$lef_trigger" \
            --arg action "$lef_action" \
            --argjson fact "$lef_fact_json" \
            --argjson interp "$lef_interp_json" \
            '. += [{trigger: $trigger, action: $action, fact: $fact, interpretation: $interp}]')

        # Feed through instinct-create (non-blocking -- failures are logged but don't stop extraction)
        bash "$0" instinct-create \
            --trigger "$lef_trigger" \
            --action "$lef_action" \
            --confidence 0.5 \
            --domain "$lef_cat" \
            --source "fallback-phase-$lef_phase" \
            --evidence "$lef_fact" >/dev/null 2>&1 || _aether_log_error "Fallback instinct-create failed for $lef_cat"

        lef_count=$((lef_count + 1))
    done

    json_ok "{\"learnings\":$lef_learnings,\"count\":$lef_count}"
    exit 0
}
