# =============================================================================
# DEPRECATED — This script has been superseded by the Go binary (aether CLI).
# All functionality is now available via: aether <subcommand>
# Do NOT modify this file — it is retained for reference only.
# See: cmd/ (Go source) | Run: aether --help
# =============================================================================
#
#!/bin/bash
# Hive Brain utility functions — extracted from aether-utils.sh
# Provides: _hive_init, _hive_store, _hive_read, _hive_abstract, _hive_promote
#
# These functions are sourced by aether-utils.sh at startup.
# All shared infrastructure (json_ok, json_err, atomic_write, acquire_lock,
# release_lock, LOCK_DIR, DATA_DIR, SCRIPT_DIR, error constants) is available.
#
# Lock handling: hive-store, hive-read, hive-init use acquire_lock_at/release_lock_at
# with colony-tagged lock files for cross-repo mutual exclusion on hub-level resources.
# This avoids global LOCK_DIR mutation (previous approach).

_hive_init() {
    # Initialize the ~/.aether/hive/ directory and wisdom.json schema
    # Usage: hive-init
    # Idempotent: safe to call multiple times — will NOT overwrite existing wisdom.json

    hv_hive_dir="$HOME/.aether/hive"
    hv_wisdom_file="$hv_hive_dir/wisdom.json"
    hv_already_existed="false"

    mkdir -p "$hv_hive_dir"

    if [[ -f "$hv_wisdom_file" ]]; then
      hv_already_existed="true"
    else
      hv_created_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
      hv_initial_schema=$(jq -n \
        --arg created_at "$hv_created_at" \
        --arg last_updated "$hv_created_at" \
        '{
          version: "1.0.0",
          created_at: $created_at,
          last_updated: $last_updated,
          entries: [],
          metadata: {
            total_entries: 0,
            max_entries: 200,
            contributing_repos: []
          }
        }')

      hv_lock_held=false
      hv_lock_file=""
      if type acquire_lock_at &>/dev/null; then
        hv_colony_tag=$(bash "$0" colony-name 2>/dev/null | jq -r '.result.name // ""' | tr '[:upper:]' '[:lower:]' | tr -cs '[:alnum:]' '-' | sed 's/^-//;s/-$//') || true
        [[ -z "$hv_colony_tag" ]] && hv_colony_tag="unknown"
        acquire_lock_at "$hv_wisdom_file" "$hv_hive_dir" "$hv_colony_tag" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on wisdom.json"
        hv_lock_file="$LOCK_AT_FILE"
        hv_lock_held=true
      fi

      atomic_write "$hv_wisdom_file" "$hv_initial_schema" || {
        [[ "$hv_lock_held" == "true" ]] && { release_lock_at "$hv_lock_file" 2>/dev/null || true; hv_lock_held=false; }
        json_err "$E_JSON_INVALID" "Failed to write wisdom.json"
      }

      [[ "$hv_lock_held" == "true" ]] && { release_lock_at "$hv_lock_file" 2>/dev/null || true; hv_lock_held=false; }
    fi

    json_ok "$(jq -n --arg dir "$hv_hive_dir" --argjson already_existed "$hv_already_existed" '{dir: $dir, initialized: true, already_existed: $already_existed}')"
}

_hive_store() {
    # Store a wisdom entry in ~/.aether/hive/wisdom.json
    # Usage: hive-store --text <text> --domain <csv> --source-repo <path> --confidence <0-1> --category <cat>
    # Deduplicates by content hash. Same-repo dups skipped, cross-repo dups merged.
    # Enforces 200 entry cap — evicts oldest by last_accessed when full.

    hs_text=""
    hs_domain=""
    hs_source_repo=""
    hs_confidence="0.5"
    hs_category="general"

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --text)        hs_text="${2:-}"; shift 2 ;;
        --domain)      hs_domain="${2:-}"; shift 2 ;;
        --source-repo) hs_source_repo="${2:-}"; shift 2 ;;
        --confidence)  hs_confidence="${2:-0.5}"; shift 2 ;;
        --category)    hs_category="${2:-general}"; shift 2 ;;
        *) shift ;;
      esac
    done

    # Validate required fields
    [[ -z "$hs_text" ]] && json_err "$E_VALIDATION_FAILED" "Missing required --text argument" '{"missing":"text"}'
    [[ -z "$hs_source_repo" ]] && json_err "$E_VALIDATION_FAILED" "Missing required --source-repo argument" '{"missing":"source_repo"}'

    # Validate confidence range
    if ! [[ "$hs_confidence" =~ ^(0(\.[0-9]+)?|1(\.0+)?)$ ]]; then
      json_err "$E_VALIDATION_FAILED" "Confidence must be a number between 0.0 and 1.0" "{\"provided\":\"$hs_confidence\"}"
    fi

    # Content sanitization (matches pheromone-write pattern)
    if echo "$hs_text" | grep -Eiq '<[[:space:]]*/?(system|prompt|instructions|system-reminder|assistant|user|human)'; then
      json_err "$E_VALIDATION_FAILED" "Wisdom content rejected: XML tag injection pattern detected"
    fi
    hs_text="${hs_text//</&lt;}"
    hs_text="${hs_text//>/&gt;}"
    hs_text="${hs_text:0:500}"
    if echo "$hs_text" | grep -Eiq '(\$\(|`|(^|[[:space:]])curl([[:space:]]|$)|(^|[[:space:]])wget([[:space:]]|$)|(^|[[:space:]])rm([[:space:]]|$))'; then
      json_err "$E_VALIDATION_FAILED" "Wisdom content rejected: potential injection pattern"
    fi
    if echo "$hs_text" | grep -Eiq '(ignore\s+(all\s+)?(previous\s+|prior\s+|above\s+)?instructions|disregard\s+(above|previous|all)|you are now |new instructions:|system prompt)'; then
      json_err "$E_VALIDATION_FAILED" "Wisdom content rejected: prompt injection pattern detected"
    fi

    # Ensure hive is initialized
    bash "$0" hive-init >/dev/null 2>&1 || json_err "$E_FILE_NOT_FOUND" "Unable to initialize hive"

    hs_wisdom_file="$HOME/.aether/hive/wisdom.json"
    [[ -f "$hs_wisdom_file" ]] || json_err "$E_FILE_NOT_FOUND" "Hive wisdom file not found"

    if ! jq -e . "$hs_wisdom_file" >/dev/null 2>&1; then
      json_err "$E_JSON_INVALID" "Hive wisdom JSON is invalid"
    fi

    # Generate content hash (first 12 chars of SHA-256)
    hs_content_hash=$(printf '%s' "$hs_text" | shasum -a 256 | cut -c1-12)

    # Parse domain tags CSV into JSON array
    hs_domain_json="[]"
    if [[ -n "$hs_domain" ]]; then
      hs_domain_json=$(echo "$hs_domain" | tr ',' '\n' | jq -R 'gsub("^\\s+|\\s+$";"")' | jq -s '.')
    fi

    hs_now_iso=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Acquire lock — use colony-tagged lock in hub dir for cross-repo mutual exclusion
    hs_lock_held=false
    hs_lock_file=""
    if type acquire_lock_at &>/dev/null; then
      hs_colony_tag=$(bash "$0" colony-name 2>/dev/null | jq -r '.result.name // ""' | tr '[:upper:]' '[:lower:]' | tr -cs '[:alnum:]' '-' | sed 's/^-//;s/-$//') || true
      [[ -z "$hs_colony_tag" ]] && hs_colony_tag="unknown"
      acquire_lock_at "$hs_wisdom_file" "$HOME/.aether/hive" "$hs_colony_tag" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on wisdom.json"
      hs_lock_file="$LOCK_AT_FILE"
      hs_lock_held=true
    fi

    # Check for existing entry with same content hash
    hs_existing_idx=$(jq --arg hash "$hs_content_hash" '
      .entries | to_entries | map(select(.value.id == $hash)) | .[0].key // -1
    ' "$hs_wisdom_file" 2>/dev/null)

    if [[ "$hs_existing_idx" != "-1" ]] && [[ "$hs_existing_idx" != "null" ]] && [[ -n "$hs_existing_idx" ]]; then
      # Entry exists — check if same repo
      hs_has_repo=$(jq --arg hash "$hs_content_hash" --arg repo "$hs_source_repo" '
        .entries[] | select(.id == $hash) | .source_repos | map(select(. == $repo)) | length > 0
      ' "$hs_wisdom_file" 2>/dev/null)

      if [[ "$hs_has_repo" == "true" ]]; then
        # Same repo duplicate — skip
        [[ "$hs_lock_held" == "true" ]] && { release_lock_at "$hs_lock_file" 2>/dev/null || true; hs_lock_held=false; }
        json_ok "$(jq -n --arg id "$hs_content_hash" \
          '{action: "skipped", reason: "duplicate from same repo", id: $id}')"
      else
        # Different repo — merge: increment validated_count, add repo, boost confidence
        hs_updated=$(jq --arg hash "$hs_content_hash" \
          --arg repo "$hs_source_repo" \
          --arg now "$hs_now_iso" '
          .entries = [.entries[] |
            if .id == $hash then
              .validated_count = (.validated_count + 1) |
              .source_repos = (.source_repos + [$repo] | unique) |
              .last_accessed = $now |
              # Confidence boosting: tier based on source_repos count
              # 2 repos -> 0.7, 3 repos -> 0.85, 4+ repos -> 0.95
              # Never downgrade: use max(current, tier)
              (.source_repos | length) as $repo_count |
              (if $repo_count >= 4 then 0.95
               elif $repo_count == 3 then 0.85
               elif $repo_count == 2 then 0.7
               else .confidence end) as $tier_confidence |
              .confidence = ([.confidence, $tier_confidence] | max)
            else . end
          ] |
          .metadata.contributing_repos = ([.entries[].source_repos[]] | unique) |
          .last_updated = $now
        ' "$hs_wisdom_file" 2>/dev/null) || {
          [[ "$hs_lock_held" == "true" ]] && { release_lock_at "$hs_lock_file" 2>/dev/null || true; hs_lock_held=false; }
          json_err "$E_JSON_INVALID" "Failed to merge wisdom entry"
        }

        atomic_write "$hs_wisdom_file" "$hs_updated" || {
          [[ "$hs_lock_held" == "true" ]] && { release_lock_at "$hs_lock_file" 2>/dev/null || true; hs_lock_held=false; }
          json_err "$E_JSON_INVALID" "Failed to write merged wisdom entry"
        }

        [[ "$hs_lock_held" == "true" ]] && { release_lock_at "$hs_lock_file" 2>/dev/null || true; hs_lock_held=false; }
        hs_new_count=$(echo "$hs_updated" | jq --arg hash "$hs_content_hash" '.entries[] | select(.id == $hash) | .validated_count')
        hs_new_confidence=$(echo "$hs_updated" | jq --arg hash "$hs_content_hash" '.entries[] | select(.id == $hash) | .confidence')
        json_ok "$(jq -n --arg id "$hs_content_hash" --argjson validated_count "$hs_new_count" \
          --argjson confidence "$hs_new_confidence" \
          '{action: "merged", id: $id, validated_count: $validated_count, confidence: $confidence}')"
      fi
    else
      # New entry — build and append
      hs_entry=$(jq -n \
        --arg id "$hs_content_hash" \
        --arg text "$hs_text" \
        --arg category "$hs_category" \
        --argjson confidence "$hs_confidence" \
        --argjson domain_tags "$hs_domain_json" \
        --arg source_repo "$hs_source_repo" \
        --arg created_at "$hs_now_iso" \
        --arg last_accessed "$hs_now_iso" \
        '{
          id: $id,
          text: $text,
          category: $category,
          confidence: $confidence,
          domain_tags: $domain_tags,
          source_repos: [$source_repo],
          validated_count: 1,
          created_at: $created_at,
          last_accessed: $last_accessed,
          access_count: 0
        }')

      # Append entry and enforce 200 cap (evict oldest by last_accessed)
      hs_updated=$(jq --argjson entry "$hs_entry" --arg now "$hs_now_iso" '
        .entries = (.entries + [$entry]) |
        if (.entries | length) > 200 then
          .entries = (.entries | sort_by(.last_accessed) | .[-200:])
        else . end |
        .metadata.total_entries = (.entries | length) |
        .metadata.contributing_repos = ([.entries[].source_repos[]] | unique) |
        .last_updated = $now
      ' "$hs_wisdom_file" 2>/dev/null) || {
        [[ "$hs_lock_held" == "true" ]] && { release_lock_at "$hs_lock_file" 2>/dev/null || true; hs_lock_held=false; }
        json_err "$E_JSON_INVALID" "Failed to append wisdom entry"
      }

      atomic_write "$hs_wisdom_file" "$hs_updated" || {
        [[ "$hs_lock_held" == "true" ]] && { release_lock_at "$hs_lock_file" 2>/dev/null || true; hs_lock_held=false; }
        json_err "$E_JSON_INVALID" "Failed to write new wisdom entry"
      }

      [[ "$hs_lock_held" == "true" ]] && { release_lock_at "$hs_lock_file" 2>/dev/null || true; hs_lock_held=false; }
      json_ok "$(jq -n --arg id "$hs_content_hash" --arg category "$hs_category" \
        '{action: "stored", id: $id, category: $category}')"
    fi
}

_hive_read() {
    # Read wisdom entries from ~/.aether/hive/wisdom.json with filtering and access tracking
    # Usage: hive-read [--domain <csv>] [--limit <N>] [--min-confidence <0.0-1.0>] [--format <json|text>]
    # Increments access_count and updates last_accessed for returned entries.

    hr_domain=""
    hr_limit="10"
    hr_min_confidence="0.0"
    hr_format="json"

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --domain)         hr_domain="${2:-}"; shift 2 ;;
        --limit)          hr_limit="${2:-10}"; shift 2 ;;
        --min-confidence) hr_min_confidence="${2:-0.0}"; shift 2 ;;
        --format)         hr_format="${2:-json}"; shift 2 ;;
        *) shift ;;
      esac
    done

    # Validate limit is a positive integer
    if ! [[ "$hr_limit" =~ ^[0-9]+$ ]] || [[ "$hr_limit" -lt 1 ]]; then
      json_err "$E_VALIDATION_FAILED" "Limit must be a positive integer" "{\"provided\":\"$hr_limit\"}"
    fi

    # Validate min-confidence is a valid number 0.0-1.0
    if ! [[ "$hr_min_confidence" =~ ^(0(\.[0-9]+)?|1(\.0+)?)$ ]]; then
      json_err "$E_VALIDATION_FAILED" "Min-confidence must be a number between 0.0 and 1.0" "{\"provided\":\"$hr_min_confidence\"}"
    fi

    # Validate format
    if [[ "$hr_format" != "json" ]] && [[ "$hr_format" != "text" ]]; then
      json_err "$E_VALIDATION_FAILED" "Format must be 'json' or 'text'" "{\"provided\":\"$hr_format\"}"
    fi

    hr_wisdom_file="$HOME/.aether/hive/wisdom.json"

    # Fallback: no wisdom file
    if [[ ! -f "$hr_wisdom_file" ]]; then
      json_ok '{"entries":[],"total_matched":0,"fallback":"no_hive"}'
      exit 0
    fi

    # Validate JSON
    if ! jq -e . "$hr_wisdom_file" >/dev/null 2>&1; then
      json_ok '{"entries":[],"total_matched":0,"fallback":"invalid_json"}'
      exit 0
    fi

    # Parse domain tags CSV into JSON array for jq filtering
    hr_domain_json="[]"
    if [[ -n "$hr_domain" ]]; then
      hr_domain_json=$(echo "$hr_domain" | tr ',' '\n' | jq -R 'gsub("^\\s+|\\s+$";"")' | jq -s '.')
    fi

    # Filter, sort, and select entries using jq
    hr_filtered=$(jq \
      --argjson domain_filter "$hr_domain_json" \
      --argjson min_conf "$hr_min_confidence" \
      --argjson limit "$hr_limit" '
      .entries
      | map(
          select(((.confidence // 0) | tonumber) >= $min_conf)
          | if ($domain_filter | length) > 0 then
              select(
                [.domain_tags[] as $dt | $domain_filter[] | select(. == $dt)] | length > 0
              )
            else . end
        )
      | sort_by(-((.confidence // 0) | tonumber), -.validated_count)
      | { total_matched: length, entries: .[:$limit], returned_ids: [.[:$limit][].id] }
    ' "$hr_wisdom_file" 2>/dev/null) || {
      json_ok '{"entries":[],"total_matched":0,"fallback":"filter_error"}'
      exit 0
    }

    hr_total_matched=$(echo "$hr_filtered" | jq -r '.total_matched')
    hr_returned_ids=$(echo "$hr_filtered" | jq -c '.returned_ids')
    hr_entries=$(echo "$hr_filtered" | jq -c '.entries')

    # Update access_count and last_accessed for returned entries
    if [[ "$hr_total_matched" -gt 0 ]] && [[ $(echo "$hr_returned_ids" | jq 'length') -gt 0 ]]; then
      hr_now_iso=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

      hr_lock_held=false
      hr_lock_file=""
      if type acquire_lock_at &>/dev/null; then
        hr_colony_tag=$(bash "$0" colony-name 2>/dev/null | jq -r '.result.name // ""' | tr '[:upper:]' '[:lower:]' | tr -cs '[:alnum:]' '-' | sed 's/^-//;s/-$//') || true
        [[ -z "$hr_colony_tag" ]] && hr_colony_tag="unknown"
        acquire_lock_at "$hr_wisdom_file" "$HOME/.aether/hive" "$hr_colony_tag" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on wisdom.json"
        hr_lock_file="$LOCK_AT_FILE"
        hr_lock_held=true
      fi

      hr_updated=$(jq \
        --argjson returned_ids "$hr_returned_ids" \
        --arg now "$hr_now_iso" '
        .entries = [.entries[] |
          if (.id as $id | $returned_ids | index($id)) != null then
            .access_count = ((.access_count // 0) + 1) |
            .last_accessed = $now
          else . end
        ] |
        .last_updated = $now
      ' "$hr_wisdom_file" 2>/dev/null) || {
        [[ "$hr_lock_held" == "true" ]] && { release_lock_at "$hr_lock_file" 2>/dev/null || true; hr_lock_held=false; }
        json_err "$E_JSON_INVALID" "Failed to update access tracking in wisdom.json"
      }

      atomic_write "$hr_wisdom_file" "$hr_updated" || {
        [[ "$hr_lock_held" == "true" ]] && { release_lock_at "$hr_lock_file" 2>/dev/null || true; hr_lock_held=false; }
        json_err "$E_JSON_INVALID" "Failed to write updated wisdom.json"
      }

      [[ "$hr_lock_held" == "true" ]] && { release_lock_at "$hr_lock_file" 2>/dev/null || true; hr_lock_held=false; }
    fi

    # Format output
    if [[ "$hr_format" == "text" ]]; then
      hr_text_output=$(echo "$hr_entries" | jq -r '
        . as $entries |
        if ($entries | length) == 0 then "(no wisdom entries)"
        else
          [range($entries | length)] |
          map(
            $entries[.] |
            "[\(.confidence | tostring)] [\(.category)] \(.text) (validated: \(.validated_count), domains: \(.domain_tags | join(", ")))"
          ) | join("\n")
        end
      ' 2>/dev/null)

      hr_text_escaped=$(echo "$hr_text_output" | jq -Rs '.')
      json_ok "$(jq -n --argjson entries "$hr_entries" --argjson total_matched "$hr_total_matched" \
        --argjson text "$hr_text_escaped" \
        '{entries: $entries, total_matched: $total_matched, text: $text}')"
    else
      json_ok "$(jq -n --argjson entries "$hr_entries" --argjson total_matched "$hr_total_matched" \
        '{entries: $entries, total_matched: $total_matched}')"
    fi
}

_hive_abstract() {
    # Abstract a repo-specific instinct into generalized cross-colony wisdom
    # Usage: hive-abstract --text <instinct_text> --source-repo <repo_path> [--domain <csv>]
    # Returns: JSON with original text, abstracted text, and transformations applied.
    # This is a TEXT TRANSFORMATION only — does NOT write to wisdom.json.

    ha_text=""
    ha_source_repo=""
    ha_domain=""

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --text)        ha_text="${2:-}"; shift 2 ;;
        --source-repo) ha_source_repo="${2:-}"; shift 2 ;;
        --domain)      ha_domain="${2:-}"; shift 2 ;;
        *) shift ;;
      esac
    done

    # Validate required fields
    [[ -z "$ha_text" ]] && json_err "$E_VALIDATION_FAILED" "Missing required --text argument" '{"missing":"text"}'
    [[ -z "$ha_source_repo" ]] && json_err "$E_VALIDATION_FAILED" "Missing required --source-repo argument" '{"missing":"source_repo"}'

    # Content sanitization (same pattern as hive-store)
    if echo "$ha_text" | grep -Eiq '<[[:space:]]*/?(system|prompt|instructions|system-reminder|assistant|user|human)'; then
      json_err "$E_VALIDATION_FAILED" "Content rejected: XML tag injection pattern detected"
    fi
    ha_text="${ha_text//</&lt;}"
    ha_text="${ha_text//>/&gt;}"
    ha_text="${ha_text:0:500}"
    if echo "$ha_text" | grep -Eiq '(\$\(|`|(^|[[:space:]])curl([[:space:]]|$)|(^|[[:space:]])wget([[:space:]]|$)|(^|[[:space:]])rm([[:space:]]|$))'; then
      json_err "$E_VALIDATION_FAILED" "Content rejected: potential injection pattern"
    fi
    if echo "$ha_text" | grep -Eiq '(ignore\s+(all\s+)?(previous\s+|prior\s+|above\s+)?instructions|disregard\s+(above|previous|all)|you are now |new instructions:|system prompt)'; then
      json_err "$E_VALIDATION_FAILED" "Content rejected: prompt injection pattern detected"
    fi

    # Save original (post-sanitization) for output
    ha_original="$ha_text"

    # Track which transformations are applied
    ha_transforms=()

    # Extract repo basename for name stripping
    ha_repo_basename=$(basename "$ha_source_repo")

    # 1. Strip absolute file paths (match /path/to/file.ext patterns)
    #    Replace paths that start with / and contain at least 2 segments
    ha_abstracted="$ha_text"
    if echo "$ha_abstracted" | grep -qE '/[A-Za-z_][A-Za-z0-9_./-]*/[A-Za-z0-9_./-]+'; then
      ha_abstracted=$(echo "$ha_abstracted" | sed -E 's|/[A-Za-z_][A-Za-z0-9_./-]*/[A-Za-z0-9_./-]+|<source-file>|g')
      ha_transforms+=("path_strip")
    fi

    # 2. Strip repo basename (case-sensitive match of the project name)
    if [[ -n "$ha_repo_basename" ]] && echo "$ha_abstracted" | grep -qF "$ha_repo_basename"; then
      ha_abstracted=$(echo "$ha_abstracted" | sed "s|${ha_repo_basename}|<project>|g")
      ha_transforms+=("repo_name_strip")
    fi

    # 3. Strip version numbers (v1.2.3, v2.0.0, etc.)
    if echo "$ha_abstracted" | grep -qE 'v[0-9]+\.[0-9]+\.[0-9]+'; then
      ha_abstracted=$(echo "$ha_abstracted" | sed -E 's/v[0-9]+\.[0-9]+\.[0-9]+/<version>/g')
      ha_transforms+=("version_strip")
    fi

    # 4. Strip branch names (feature/xyz, bugfix/abc, hotfix/def, release/xyz)
    if echo "$ha_abstracted" | grep -qE '(feature|bugfix|hotfix|release)/[A-Za-z0-9_.-]+'; then
      ha_abstracted=$(echo "$ha_abstracted" | sed -E 's/(feature|bugfix|hotfix|release)\/[A-Za-z0-9_.-]+/<branch>/g')
      ha_transforms+=("branch_strip")
    fi

    # Parse domain tags CSV into JSON array
    ha_domain_json="[]"
    if [[ -n "$ha_domain" ]]; then
      ha_domain_json=$(echo "$ha_domain" | tr ',' '\n' | jq -R 'gsub("^\\s+|\\s+$";"")' | jq -s '.')
    fi

    # Build transformations JSON array
    ha_transforms_json="[]"
    if [[ ${#ha_transforms[@]} -gt 0 ]]; then
      ha_transforms_json=$(printf '%s\n' "${ha_transforms[@]}" | jq -R '.' | jq -s '.')
    fi

    # Build result JSON using jq for proper escaping
    ha_result=$(jq -n \
      --arg original "$ha_original" \
      --arg abstracted "$ha_abstracted" \
      --arg source_repo "$ha_source_repo" \
      --argjson domain_tags "$ha_domain_json" \
      --argjson transformations "$ha_transforms_json" \
      '{
        original: $original,
        abstracted: $abstracted,
        source_repo: $source_repo,
        domain_tags: $domain_tags,
        transformations_applied: $transformations
      }')

    json_ok "$ha_result"
}

_hive_promote() {
    # Orchestrate the full promotion pipeline: abstract an instinct, then store it
    # Usage: hive-promote --text <instinct_text> --source-repo <repo_path> [--domain <csv>] [--confidence <0-1>] [--category <cat>]
    # Calls hive-abstract internally to generalize the text, then hive-store to persist it.
    # Returns: combined result with action mapping (stored->promoted, merged->merged, skipped->skipped).

    hp_text=""
    hp_source_repo=""
    hp_domain=""
    hp_confidence="0.7"
    hp_category="pattern"

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --text)        hp_text="${2:-}"; shift 2 ;;
        --source-repo) hp_source_repo="${2:-}"; shift 2 ;;
        --domain)      hp_domain="${2:-}"; shift 2 ;;
        --confidence)  hp_confidence="${2:-0.7}"; shift 2 ;;
        --category)    hp_category="${2:-pattern}"; shift 2 ;;
        *) shift ;;
      esac
    done

    # Validate required fields
    [[ -z "$hp_text" ]] && json_err "$E_VALIDATION_FAILED" "Missing required --text argument" '{"missing":"text"}'
    [[ -z "$hp_source_repo" ]] && json_err "$E_VALIDATION_FAILED" "Missing required --source-repo argument" '{"missing":"source_repo"}'

    # Ensure hive is initialized (idempotent)
    bash "$0" hive-init >/dev/null 2>&1 || json_err "$E_FILE_NOT_FOUND" "Unable to initialize hive"

    # Step 1: Abstract the instinct text
    hp_abstract_args=(hive-abstract --text "$hp_text" --source-repo "$hp_source_repo")
    [[ -n "$hp_domain" ]] && hp_abstract_args+=(--domain "$hp_domain")

    hp_abstract_result=$(bash "$0" "${hp_abstract_args[@]}" 2>&1) || {
      json_err "$E_VALIDATION_FAILED" "Abstraction failed: $(echo "$hp_abstract_result" | jq -r '.error.message // "unknown error"' 2>/dev/null)"
    }

    # Extract abstracted text and transformations from abstract result
    hp_abstracted=$(echo "$hp_abstract_result" | jq -r '.result.abstracted // empty' 2>/dev/null)
    hp_original=$(echo "$hp_abstract_result" | jq -r '.result.original // empty' 2>/dev/null)
    hp_transforms=$(echo "$hp_abstract_result" | jq -c '.result.transformations_applied // []' 2>/dev/null)

    if [[ -z "$hp_abstracted" ]]; then
      json_err "$E_VALIDATION_FAILED" "Abstraction returned empty text"
    fi

    # Step 2: Store the abstracted text in the hive
    hp_store_args=(hive-store --text "$hp_abstracted" --source-repo "$hp_source_repo" --confidence "$hp_confidence" --category "$hp_category")
    [[ -n "$hp_domain" ]] && hp_store_args+=(--domain "$hp_domain")

    hp_store_result=$(bash "$0" "${hp_store_args[@]}" 2>&1) || {
      json_err "$E_VALIDATION_FAILED" "Store failed: $(echo "$hp_store_result" | jq -r '.error.message // "unknown error"' 2>/dev/null)"
    }

    # Extract store action
    hp_store_action=$(echo "$hp_store_result" | jq -r '.result.action // "unknown"' 2>/dev/null)

    # Map store action to promote action: stored->promoted, merged->merged, skipped->skipped
    hp_action="$hp_store_action"
    [[ "$hp_store_action" == "stored" ]] && hp_action="promoted"

    # Build combined result
    hp_result=$(jq -n \
      --arg action "$hp_action" \
      --arg original "$hp_original" \
      --arg abstracted "$hp_abstracted" \
      --argjson transformations "$hp_transforms" \
      --arg store_action "$hp_store_action" \
      --argjson confidence "$hp_confidence" \
      --arg source_repo "$hp_source_repo" \
      '{
        action: $action,
        original: $original,
        abstracted: $abstracted,
        transformations: $transformations,
        store_action: $store_action,
        confidence: $confidence,
        source_repo: $source_repo
      }')

    json_ok "$hp_result"
}
