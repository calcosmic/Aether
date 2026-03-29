#!/usr/bin/env bash
# Queen utility functions -- extracted from aether-utils.sh
# Provides: _queen_init, _queen_read, _queen_thresholds, _queen_promote
# Also includes: _extract_wisdom_sections (helper used only by _queen_read)
# Note: Uses get_wisdom_threshold() and get_wisdom_thresholds_json() which remain in the main file

# ============================================================================
# _extract_wisdom_sections
# Helper function to extract wisdom sections from a QUEEN.md file
# Uses line number approach to avoid macOS awk range issues
# Usage: _extract_wisdom_sections <file_path>
# Returns: JSON object with wisdom sections
# ============================================================================
_extract_wisdom_sections() {
      local file="$1"

      # Format detection: check for v2 header "## Build Learnings"
      # If present -> v2 format (4 sections). Otherwise -> v1 format (6 emoji sections, mapped).
      if grep -q '^## Build Learnings$' "$file" 2>/dev/null; then
        # === V2 FORMAT (4 clean sections) ===
        local uprefs_line=$(awk '/^## User Preferences$/ {print NR; exit}' "$file")
        local cpat_line=$(awk '/^## Codebase Patterns$/ {print NR; exit}' "$file")
        local blearn_line=$(awk '/^## Build Learnings$/ {print NR; exit}' "$file")
        local inst_line=$(awk '/^## Instincts$/ {print NR; exit}' "$file")
        local evo_line=$(awk '/^## Evolution Log$/ {print NR; exit}' "$file")

        local user_prefs codebase_patterns build_learnings instincts

        # User Preferences: between uprefs_line and next section
        local uprefs_end="${cpat_line:-${blearn_line:-${inst_line:-${evo_line:-999999}}}}"
        if [[ -n "$uprefs_line" ]]; then
          # SUPPRESS:OK -- read-default: text escaping returns fallback on empty input
          user_prefs=$(awk -v s="$uprefs_line" -v e="$uprefs_end" 'NR > s && NR < e {print}' "$file" | sed '/^$/d' | sed '/^---$/d' | jq -Rs '.' 2>/dev/null || echo '""')
        else user_prefs='""'; fi

        # Codebase Patterns: between cpat_line and next section
        local cpat_end="${blearn_line:-${inst_line:-${evo_line:-999999}}}"
        if [[ -n "$cpat_line" ]]; then
          # SUPPRESS:OK -- read-default: text escaping returns fallback on empty input
          codebase_patterns=$(awk -v s="$cpat_line" -v e="$cpat_end" 'NR > s && NR < e {print}' "$file" | sed '/^$/d' | sed '/^---$/d' | jq -Rs '.' 2>/dev/null || echo '""')
        else codebase_patterns='""'; fi

        # Build Learnings: between blearn_line and next section
        local blearn_end="${inst_line:-${evo_line:-999999}}"
        if [[ -n "$blearn_line" ]]; then
          # SUPPRESS:OK -- read-default: text escaping returns fallback on empty input
          build_learnings=$(awk -v s="$blearn_line" -v e="$blearn_end" 'NR > s && NR < e {print}' "$file" | sed '/^$/d' | sed '/^---$/d' | jq -Rs '.' 2>/dev/null || echo '""')
        else build_learnings='""'; fi

        # Instincts: between inst_line and evo_line (or end)
        if [[ -n "$inst_line" ]]; then
          # SUPPRESS:OK -- read-default: text escaping returns fallback on empty input
          instincts=$(awk -v s="$inst_line" -v e="${evo_line:-999999}" 'NR > s && NR < e {print}' "$file" | sed '/^$/d' | sed '/^---$/d' | jq -Rs '.' 2>/dev/null || echo '""')
        else instincts='""'; fi

        # Output v2 JSON
        echo "{\"user_prefs\":$user_prefs,\"codebase_patterns\":$codebase_patterns,\"build_learnings\":$build_learnings,\"instincts\":$instincts}"

      else
        # === V1 FORMAT (6 emoji sections, mapped to v2 keys) ===
        local p_line=$(awk '/^## ..? ?Philosophies$/ {print NR; exit}' "$file")
        local pat_line=$(awk '/^## ..? ?Patterns$/ {print NR; exit}' "$file")
        local red_line=$(awk '/^## ..? ?Redirects$/ {print NR; exit}' "$file")
        local stack_line=$(awk '/^## ..? ?Stack Wisdom$/ {print NR; exit}' "$file")
        local dec_line=$(awk '/^## ..? ?Decrees$/ {print NR; exit}' "$file")
        local prefs_line=$(awk '/^## ..? ?User Preferences$/ {print NR; exit}' "$file")
        local evo_line=$(awk '/^## ..? ?Evolution Log$/ {print NR; exit}' "$file")

        local philosophies patterns redirects stack_wisdom decrees user_prefs

        if [[ -n "$p_line" && -n "$pat_line" ]]; then
          # SUPPRESS:OK -- read-default: text escaping returns fallback on empty input
          philosophies=$(awk -v s="$p_line" -v e="$pat_line" 'NR > s && NR < e {print}' "$file" | sed '/^$/d' | jq -Rs '.' 2>/dev/null || echo '""')
        else philosophies='""'; fi

        if [[ -n "$pat_line" && -n "$red_line" ]]; then
          # SUPPRESS:OK -- read-default: text escaping returns fallback on empty input
          patterns=$(awk -v s="$pat_line" -v e="$red_line" 'NR > s && NR < e {print}' "$file" | sed '/^$/d' | jq -Rs '.' 2>/dev/null || echo '""')
        else patterns='""'; fi

        if [[ -n "$red_line" && -n "$stack_line" ]]; then
          # SUPPRESS:OK -- read-default: text escaping returns fallback on empty input
          redirects=$(awk -v s="$red_line" -v e="$stack_line" 'NR > s && NR < e {print}' "$file" | sed '/^$/d' | jq -Rs '.' 2>/dev/null || echo '""')
        else redirects='""'; fi

        if [[ -n "$stack_line" && -n "$dec_line" ]]; then
          # SUPPRESS:OK -- read-default: text escaping returns fallback on empty input
          stack_wisdom=$(awk -v s="$stack_line" -v e="$dec_line" 'NR > s && NR < e {print}' "$file" | sed '/^$/d' | jq -Rs '.' 2>/dev/null || echo '""')
        else stack_wisdom='""'; fi

        local dec_end="${prefs_line:-${evo_line:-999999}}"
        if [[ -n "$dec_line" ]]; then
          # SUPPRESS:OK -- read-default: text escaping returns fallback on empty input
          decrees=$(awk -v s="$dec_line" -v e="$dec_end" 'NR > s && NR < e {print}' "$file" | sed '/^$/d' | jq -Rs '.' 2>/dev/null || echo '""')
        else decrees='""'; fi

        if [[ -n "$prefs_line" ]]; then
          # SUPPRESS:OK -- read-default: text escaping returns fallback on empty input
          user_prefs=$(awk -v s="$prefs_line" -v e="${evo_line:-999999}" 'NR > s && NR < e {print}' "$file" | sed '/^$/d' | jq -Rs '.' 2>/dev/null || echo '""')
        else user_prefs='""'; fi

        # Map v1 sections to v2 keys:
        # Philosophies + Patterns + Redirects + Stack Wisdom -> codebase_patterns
        # Decrees + old User Preferences -> user_prefs
        # build_learnings and instincts -> empty for v1 files
        local combined_codebase
        combined_codebase=$(jq -n \
          --arg phil "$philosophies" \
          --arg pat "$patterns" \
          --arg red "$redirects" \
          --arg stack "$stack_wisdom" \
          '[$phil, $pat, $red, $stack] | map(select(. != "" and . != null)) | join("\n")' 2>/dev/null || echo '""')

        local combined_uprefs
        combined_uprefs=$(jq -n \
          --arg dec "$decrees" \
          --arg up "$user_prefs" \
          '[$dec, $up] | map(select(. != "" and . != null)) | join("\n")' 2>/dev/null || echo '""')

        echo "{\"user_prefs\":$combined_uprefs,\"codebase_patterns\":$combined_codebase,\"build_learnings\":\"\",\"instincts\":\"\"}"
      fi
    }

# ============================================================================
# _queen_init
# Initialize QUEEN.md from template
# Creates .aether/QUEEN.md from template if missing
# Usage: Called via dispatcher as "queen-init"
# ============================================================================
_queen_init() {
    local queen_file template_file timestamp path
    queen_file="$AETHER_ROOT/.aether/QUEEN.md"

    # Check multiple locations for template
    # Order: hub (system/) -> dev (.aether/) -> repo local -> legacy
    template_file=""
    for path in \
      "$HOME/.aether/system/templates/QUEEN.md.template" \
      "$AETHER_ROOT/.aether/templates/QUEEN.md.template" \
      "$HOME/.aether/templates/QUEEN.md.template"; do
      if [[ -f "$path" ]]; then
        template_file="$path"
        break
      fi
    done

    # Ensure .aether directory exists
    mkdir -p "$AETHER_ROOT/.aether"

    # Check if QUEEN.md already exists and has content
    if [[ -f "$queen_file" ]] && [[ -s "$queen_file" ]]; then
      json_ok '{"created":false,"path":".aether/QUEEN.md","reason":"already_exists"}'
      exit 0
    fi

    # Check if template was found
    if [[ -z "$template_file" ]]; then
      json_err "$E_FILE_NOT_FOUND" \
        "Template not found. Run: npm install -g aether && aether install to restore it." \
        '{"templates_checked":["~/.aether/system/templates/QUEEN.md.template",".aether/templates/QUEEN.md.template","~/.aether/templates/QUEEN.md.template"]}'
      exit 1
    fi

    # Create QUEEN.md from template with timestamp substitution
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    sed -e "s/{TIMESTAMP}/$timestamp/g" "$template_file" > "$queen_file"

    if [[ -f "$queen_file" ]]; then
      json_ok "$(jq -n --arg source "$template_file" '{created: true, path: ".aether/QUEEN.md", source: $source}')"
    else
      json_err "$E_FILE_NOT_FOUND" "Failed to create QUEEN.md" '{"path":".aether/QUEEN.md"}'
      exit 1
    fi
}

# ============================================================================
# _queen_read
# Read QUEEN.md and return wisdom as JSON for worker priming
# Supports two-level loading: global (~/.aether/QUEEN.md) first, then local (.aether/QUEEN.md)
# Local wisdom extends global - entries are combined per category
# Usage: Called via dispatcher as "queen-read"
# ============================================================================
_queen_read() {
    local queen_global queen_local has_global has_local global_wisdom local_wisdom combined metadata
    local user_prefs codebase_patterns build_learnings instincts result
    queen_global="$HOME/.aether/QUEEN.md"
    queen_local="$AETHER_ROOT/.aether/QUEEN.md"

    # Track which files exist
    has_global=false
    has_local=false

    # Check for global QUEEN.md
    if [[ -f "$queen_global" ]]; then
      has_global=true
    fi

    # Check for local QUEEN.md
    if [[ -f "$queen_local" ]]; then
      has_local=true
    fi

    # FAIL HARD if no QUEEN.md found at all
    if [[ "$has_global" == "false" && "$has_local" == "false" ]]; then
      json_err "$E_FILE_NOT_FOUND" "QUEEN.md not found" '{"global_path":"~/.aether/QUEEN.md","local_path":".aether/QUEEN.md"}'
      exit 1
    fi

    # Extract wisdom from global (if exists) -- _extract_wisdom_sections returns v2 keys
    global_wisdom='{"user_prefs":"","codebase_patterns":"","build_learnings":"","instincts":""}'
    if [[ "$has_global" == "true" ]]; then
      global_wisdom=$(_extract_wisdom_sections "$queen_global")
    fi

    # Extract wisdom from local (if exists)
    local_wisdom='{"user_prefs":"","codebase_patterns":"","build_learnings":"","instincts":""}'
    if [[ "$has_local" == "true" ]]; then
      local_wisdom=$(_extract_wisdom_sections "$queen_local")
    fi

    # Combine wisdom: local extends global - content appended (v2 keys)
    combined=$(jq -n \
      --argjson global "$global_wisdom" \
      --argjson local "$local_wisdom" \
      '
      def combine(a; b):
        if a == "" or a == null then b
        elif b == "" or b == null then a
        else a + "\n" + b
        end;

      {
        user_prefs: combine($global.user_prefs; $local.user_prefs),
        codebase_patterns: combine($global.codebase_patterns; $local.codebase_patterns),
        build_learnings: combine($global.build_learnings; $local.build_learnings),
        instincts: combine($global.instincts; $local.instincts)
      }
      ')

    # Get metadata from local (preferred) or global
    metadata='{"version":"unknown","last_evolved":null,"source":"none"}'
    if [[ "$has_local" == "true" ]]; then
      metadata=$(sed -n '/<!-- METADATA/,/-->/p' "$queen_local" | sed '1d;$d' | tr -d '\n' | sed 's/^[[:space:]]*//')
    elif [[ "$has_global" == "true" ]]; then
      metadata=$(sed -n '/<!-- METADATA/,/-->/p' "$queen_global" | sed '1d;$d' | tr -d '\n' | sed 's/^[[:space:]]*//')
    fi

    # If no metadata found, return empty structure
    if [[ -z "$metadata" ]]; then
      metadata='{"version":"unknown","last_evolved":null,"source":"none","stats":{}}'
    fi

    # Gate 1: Validate metadata is parseable JSON BEFORE using as --argjson
    if ! echo "$metadata" | jq -e . >/dev/null 2>&1; then  # SUPPRESS:OK -- validation: testing JSON validity
      json_err "$E_JSON_INVALID" \
        "QUEEN.md has a malformed METADATA block — the JSON between <!-- METADATA and --> is invalid. Try: fix the JSON in .aether/QUEEN.md or run queen-init to reset."
    fi

    # Extract individual combined wisdom values (v2 keys)
    user_prefs=$(echo "$combined" | jq -r '.user_prefs')
    codebase_patterns=$(echo "$combined" | jq -r '.codebase_patterns')
    build_learnings=$(echo "$combined" | jq -r '.build_learnings')
    instincts=$(echo "$combined" | jq -r '.instincts')

    # Build JSON output (v2 keys)
    # Pass shell-level file existence flags to jq (these are authoritative, not $meta.source)
    local hg_json="false"
    local hl_json="false"
    [[ "$has_global" == "true" ]] && hg_json="true"
    [[ "$has_local" == "true" ]] && hl_json="true"

    result=$(jq -n \
      --argjson meta "$metadata" \
      --arg user_prefs "$user_prefs" \
      --arg codebase_patterns "$codebase_patterns" \
      --arg build_learnings "$build_learnings" \
      --arg instincts "$instincts" \
      --argjson has_g "$hg_json" \
      --argjson has_l "$hl_json" \
      '{
        metadata: $meta,
        wisdom: {
          user_prefs: $user_prefs,
          codebase_patterns: $codebase_patterns,
          build_learnings: $build_learnings,
          instincts: $instincts
        },
        priming: {
          has_user_prefs: ([$user_prefs | split("\n")[] | select(startswith("- "))] | length) > 0,
          has_codebase_patterns: ([$codebase_patterns | split("\n")[] | select(startswith("- "))] | length) > 0,
          has_build_learnings: ([$build_learnings | split("\n")[] | select(startswith("- ") or startswith("#"))] | length) > 0,
          has_instincts: ([$instincts | split("\n")[] | select(startswith("- "))] | length) > 0
        },
        sources: {
          has_global: $has_g,
          has_local: $has_l
        }
      }')

    # Gate 2: Validate assembled result before returning
    if [[ -z "$result" ]] || ! echo "$result" | jq -e . >/dev/null 2>&1; then  # SUPPRESS:OK -- validation: testing JSON validity
      json_err "$E_JSON_INVALID" \
        "Couldn't assemble queen-read output. QUEEN.md may have formatting issues. Try: run queen-init to reset."
    fi
    json_ok "$result"
}

# ============================================================================
# _queen_thresholds
# Return proposal and auto-promotion thresholds for each wisdom type
# Usage: Called via dispatcher as "queen-thresholds"
# Note: Uses get_wisdom_thresholds_json() which remains in the main file
# ============================================================================
_queen_thresholds() {
    json_ok "$(get_wisdom_thresholds_json)"
}

# ============================================================================
# _queen_promote
# Promote a learning to QUEEN.md wisdom
# Usage: Called via dispatcher as "queen-promote <type> <content> <colony_name>"
# Types: philosophy, pattern, redirect, stack, decree, failure
# Note: Uses get_wisdom_threshold() which remains in the main file
# ============================================================================
_queen_promote() {
    local wisdom_type content colony_name valid_types type_valid vt queen_file threshold
    local observations_file content_hash observation_data obs_count obs_colonies
    local ts entry tmp_file section_header section_line next_section_line section_end
    local has_placeholder entry_prefix ev_entry ev_separator current_count new_count stat_key
    # Usage: queen-promote <type> <content> <colony_name>
    # Types: philosophy, pattern, redirect, stack, decree
    wisdom_type="${1:-}"
    content="${2:-}"
    colony_name="${3:-}"

    # Validate required arguments
    [[ -z "$wisdom_type" ]] && json_err "$E_VALIDATION_FAILED" "Usage: queen-promote <type> <content> <colony_name>" '{"missing":"type"}'
    [[ -z "$content" ]] && json_err "$E_VALIDATION_FAILED" "Usage: queen-promote <type> <content> <colony_name>" '{"missing":"content"}'
    [[ -z "$colony_name" ]] && json_err "$E_VALIDATION_FAILED" "Usage: queen-promote <type> <content> <colony_name>" '{"missing":"colony_name"}'

    # Validate type (failure observations map to pattern when promoted)
    valid_types=("philosophy" "pattern" "redirect" "stack" "decree" "failure")
    type_valid=false
    for vt in "${valid_types[@]}"; do
      [[ "$wisdom_type" == "$vt" ]] && type_valid=true && break
    done
    [[ "$type_valid" == "false" ]] && json_err "$E_VALIDATION_FAILED" "Invalid type: $wisdom_type" '{"valid_types":["philosophy","pattern","redirect","stack","decree","failure"]}'

    queen_file="$AETHER_ROOT/.aether/QUEEN.md"

    # Check if QUEEN.md exists
    if [[ ! -f "$queen_file" ]]; then
      json_err "$E_FILE_NOT_FOUND" "QUEEN.md not found" '{"path":".aether/QUEEN.md"}'
      exit 1
    fi

    # Thresholds come from the shared command policy to keep promotion behavior consistent.
    threshold=$(get_wisdom_threshold "$wisdom_type" "propose")

    # QUEEN-04: Check threshold against learning-observations.json
    # For decrees, always promote immediately (threshold 0)
    # For other types, verify observation count meets threshold
    observations_file="$DATA_DIR/learning-observations.json"
    content_hash="sha256:$(echo -n "$content" | sha256sum | cut -d' ' -f1)"

    if [[ "$wisdom_type" != "decree" ]] && [[ -f "$observations_file" ]]; then
      # Check if this content has been observed enough times
      # SUPPRESS:OK -- read-default: query may return empty
      observation_data=$(jq -r --arg hash "$content_hash" '.observations[] | select(.content_hash == $hash) | {count: .observation_count, colonies: .colonies}' "$observations_file" 2>/dev/null || echo '{}')

      if [[ -n "$observation_data" ]] && [[ "$observation_data" != '{}' ]]; then
        obs_count=$(echo "$observation_data" | jq -r '.count // 0')
        obs_colonies=$(echo "$observation_data" | jq -r '.colonies // []')

        if [[ "$obs_count" -lt "$threshold" ]]; then
          json_err "$E_VALIDATION_FAILED" "Threshold not met: $obs_count/$threshold observations" "{\"observation_count\":$obs_count,\"threshold\":$threshold,\"content_hash\":\"$content_hash\"}"
        fi
      else
        # No observations found for this content
        json_err "$E_VALIDATION_FAILED" "No observations found for this content" "{\"threshold\":$threshold,\"content_hash\":\"$content_hash\"}"
      fi
    fi

    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Map type to section header (v2 format)
    # Old types map to new sections; detect QUEEN.md format first
    local is_v2=false
    if grep -q '^## Build Learnings$' "$queen_file" 2>/dev/null; then
      is_v2=true
    fi

    if [[ "$is_v2" == "true" ]]; then
      # V2 format: map old types to new clean section headers
      case "$wisdom_type" in
        philosophy) section_header="## Codebase Patterns" ; entry_prefix="[general] " ;;
        pattern|failure) section_header="## Codebase Patterns" ; entry_prefix="" ;;
        redirect) section_header="## Codebase Patterns" ; entry_prefix="AVOID: " ;;
        stack) section_header="## Codebase Patterns" ; entry_prefix="[repo] " ;;
        decree) section_header="## User Preferences" ; entry_prefix="" ;;
      esac
    else
      # V1 format: use original emoji headers
      case "$wisdom_type" in
        philosophy) section_header="## 📜 Philosophies" ; entry_prefix="" ;;
        pattern|failure) section_header="## 🧭 Patterns" ; entry_prefix="" ;;
        redirect) section_header="## ⚠️ Redirects" ; entry_prefix="" ;;
        stack) section_header="## 🔧 Stack Wisdom" ; entry_prefix="" ;;
        decree) section_header="## 🏛️ Decrees" ; entry_prefix="" ;;
      esac
    fi

    # Build the new entry
    entry="- ${entry_prefix}**${colony_name}** (${ts}): ${content}"

    # Create temp file for atomic write
    tmp_file="${queen_file}.tmp.$$"

    # Trap-based cleanup for intermediate temp files on exit/interrupt
    # Compose with _aether_exit_cleanup to preserve lock/temp cleanup
    # SUPPRESS:OK -- cleanup: files may not exist yet
    trap 'rm -f "${tmp_file}" "${tmp_file}".*; _aether_exit_cleanup 2>/dev/null || true' EXIT TERM INT HUP

    # Find line numbers for section boundaries
    section_line=$(grep -n "^${section_header}$" "$queen_file" | head -1 | cut -d: -f1)
    next_section_line=$(tail -n +$((section_line + 1)) "$queen_file" | grep -n "^## " | head -1 | cut -d: -f1)
    if [[ -n "$next_section_line" ]]; then
      section_end=$((section_line + next_section_line - 1))
    else
      section_end=$(wc -l < "$queen_file")
    fi

    # SUPPRESS:OK -- read-default: operation returns fallback on failure
    # Check if section has placeholder (grep returns 1 when no matches, handle with || true)
    # SUPPRESS:OK -- existence-test: grep returns 1 when no matches
    has_placeholder=$(sed -n "${section_line},${section_end}p" "$queen_file" | grep -c "No.*recorded yet" || true)
    has_placeholder=${has_placeholder:-0}

    if [[ "$has_placeholder" -gt 0 ]]; then
      # Replace placeholder with entry - only within the target section
      # Find the specific line number of the placeholder within the section
      placeholder_line=$(sed -n "${section_line},${section_end}p" "$queen_file" | grep -n "^\\*No .* recorded yet" | head -1 | cut -d: -f1)
      if [[ -n "$placeholder_line" ]]; then
        actual_line=$((section_line + placeholder_line - 1))
        # Use head/tail instead of sed c-command for newline safety (sed c\ breaks on multi-line content on macOS)
        {
          head -n $((actual_line - 1)) "$queen_file"
          echo "$entry"
          tail -n +$((actual_line + 1)) "$queen_file"
        } > "$tmp_file"
      else
        # Fallback: insert after section header using head/tail
        {
          head -n "$section_line" "$queen_file"
          echo "$entry"
          tail -n +$((section_line + 1)) "$queen_file"
        } > "$tmp_file"
      fi
    else
      # Insert entry after the description paragraph (after the second empty line in section)
      # The structure is: header, blank, description, blank, [entries...]
      # We want to insert after the blank line following the description
      empty_lines=$(sed -n "$((section_line + 1)),${section_end}p" "$queen_file" | grep -n "^$" | cut -d: -f1)
      # Get the second empty line (after description)
      insert_line=$(echo "$empty_lines" | sed -n '2p')
      if [[ -n "$insert_line" ]]; then
        insert_line=$((section_line + insert_line))
      else
        # Fallback: use first empty line
        insert_line=$(echo "$empty_lines" | head -1)
        if [[ -n "$insert_line" ]]; then
          insert_line=$((section_line + insert_line))
        else
          insert_line=$((section_line + 1))
        fi
      fi
      # Insert the entry after the found line using head/tail for newline safety
      {
        head -n "$insert_line" "$queen_file"
        echo "$entry"
        tail -n +$((insert_line + 1)) "$queen_file"
      } > "$tmp_file"
    fi

    # Update Evolution Log in temp file
    ev_entry="| ${ts} | ${colony_name} | promoted_${wisdom_type} | Added: ${content:0:50}... |"
    # Find the line after the separator in Evolution Log table
    ev_separator=$(grep -n "^|------|" "$tmp_file" | tail -1 | cut -d: -f1 || true)

    # Use awk for cross-platform insertion (only if separator found)
    if [[ -n "$ev_separator" ]]; then
      awk -v line="$ev_separator" -v entry="$ev_entry" 'NR==line{print; print entry; next}1' "$tmp_file" > "${tmp_file}.ev" && mv "${tmp_file}.ev" "$tmp_file"
    fi

    # Update METADATA stats in temp file
    # Map wisdom_type to stat key -- detect v2 vs v1 METADATA format
    if [[ "$is_v2" == "true" ]]; then
      # V2 stats keys
      case "$wisdom_type" in
        philosophy|pattern|failure|redirect|stack) stat_key="total_codebase_patterns" ;;
        decree) stat_key="total_user_prefs" ;;
        *) stat_key="total_codebase_patterns" ;;
      esac
    else
      # V1 stats keys (irregular plurals handled)
      case "$wisdom_type" in
        stack) stat_key="total_stack_entries" ;;
        philosophy) stat_key="total_philosophies" ;;
        *) stat_key="total_${wisdom_type}s" ;;
      esac
    fi
    # Read current count from temp file (which has the latest state)
    current_count=$(grep "\"${stat_key}\":" "$tmp_file" 2>/dev/null | grep -o '[0-9]*' | head -1 || true)  # SUPPRESS:OK -- read-default: file may not exist
    current_count=${current_count:-0}
    new_count=$((current_count + 1))

    # Update last_evolved using awk
    awk -v ts="$ts" '/"last_evolved":/ { gsub(/"last_evolved": "[^"]*"/, "\"last_evolved\": \"" ts "\""); } {print}' "$tmp_file" > "${tmp_file}.meta" && mv "${tmp_file}.meta" "$tmp_file"

    # Update stats count using awk
    awk -v type="$stat_key" -v count="$new_count" '{
      gsub("\"" type "\": [0-9]*", "\"" type "\": " count)
      print
    }' "$tmp_file" > "${tmp_file}.stats" && mv "${tmp_file}.stats" "$tmp_file"

    # META-02: Update evolution_log in METADATA JSON
    # Add entry with timestamp, action, wisdom_type, content_hash
    ev_log_entry="{\"timestamp\": \"$ts\", \"action\": \"promote\", \"wisdom_type\": \"$wisdom_type\", \"content_hash\": \"$content_hash\", \"colony\": \"$colony_name\"}"

    # Check if evolution_log exists in metadata, add if not
    if ! grep -q '"evolution_log"' "$tmp_file"; then
      # Add evolution_log array after stats
      awk -v entry="$ev_log_entry" '
        /"stats": \{/ {
          print
          # Read until closing brace of stats
          while (getline > 0) {
            print
            if (/\}/) break
          }
          # Add comma and evolution_log
          print ","
          print "  \"evolution_log\": [" entry "]"
          next
        }
        { print }
      ' "$tmp_file" > "${tmp_file}.evlog" && mv "${tmp_file}.evlog" "$tmp_file"
    else
      # Append to existing evolution_log array
      awk -v entry="$ev_log_entry" '
        /"evolution_log": \[/ {
          # Check if array is empty or has items
          if (/\]/) {
            # Empty array - replace with entry
            gsub(/"evolution_log": \[\]/, "\"evolution_log\": [" entry "]")
          } else {
            # Has items - need to add before closing bracket
            # For now, just print and handle in next iteration
          }
          print
          next
        }
        # Handle multi-line evolution_log arrays
        /"evolution_log": \[/ && !/\]/ {
          print
          getline
          if (/\]/) {
            # Was empty, now add entry
            print entry
            print "]"
          } else {
            # Has items, add comma and entry before closing
            print
            while (getline > 0) {
              if (/^\s*\]/) {
                print ","
                print entry
                print "]"
                break
              }
              print
            }
          }
          next
        }
        { print }
      ' "$tmp_file" > "${tmp_file}.evlog" && mv "${tmp_file}.evlog" "$tmp_file"
    fi

    # META-04: Update colonies_contributed mapping in METADATA JSON
    # This maps content_hash to array of colonies that contributed
    # Get colonies from observations file if available
    colonies_json="[]"
    if [[ -f "$observations_file" ]]; then
      # SUPPRESS:OK -- read-default: query may return empty
      colonies_from_obs=$(jq -r --arg hash "$content_hash" '.observations[] | select(.content_hash == $hash) | .colonies // [] | @json' "$observations_file" 2>/dev/null || echo '[]')
      if [[ -n "$colonies_from_obs" ]] && [[ "$colonies_from_obs" != "null" ]]; then
        colonies_json="$colonies_from_obs"
      fi
    fi

    # Add colonies_contributed object if not present
    if ! grep -q '"colonies_contributed"' "$tmp_file"; then
      # Add after evolution_log or stats
      awk -v hash="$content_hash" -v colonies="$colonies_json" '
        /"evolution_log": / {
          print
          # Skip to end of evolution_log array
          brace_count = 1
          while (getline > 0) {
            print
            if (/\[/) brace_count++
            if (/\]/) brace_count--
            if (brace_count == 0) break
          }
          print ","
          print "  \"colonies_contributed\": {"
          print "    \"" hash "\": " colonies
          print "  }"
          next
        }
        { print }
      ' "$tmp_file" > "${tmp_file}.colmap" && mv "${tmp_file}.colmap" "$tmp_file"
    else
      # Update existing colonies_contributed - add/update entry for this hash
      # Use jq for reliable JSON manipulation
      meta_section=$(sed -n '/<!-- METADATA/,/-->/p' "$tmp_file" | sed '1d;$d' | tr -d '\n' | sed 's/^[[:space:]]*//')
      if [[ -n "$meta_section" ]]; then
        # SUPPRESS:OK -- read-default: returns fallback on failure
        updated_meta=$(echo "$meta_section" | jq --arg hash "$content_hash" --argjson cols "$colonies_json" '.colonies_contributed[$hash] = $cols' 2>/dev/null || echo "$meta_section")
        # Replace metadata section
        new_comment="<!-- METADATA"
        new_comment="$new_comment
$updated_meta
-->"
        awk -v new="$new_comment" '/<!-- METADATA/,/-->/{ if (/<!-- METADATA/) print new; next }1' "$tmp_file" > "${tmp_file}.metaupd" && mv "${tmp_file}.metaupd" "$tmp_file"
      fi
    fi

    # Add colony to colonies_contributed array (legacy) if not present
    if ! grep -q "\"${colony_name}\"" "$tmp_file"; then
      # Add to colonies_contributed array using awk - handle empty and non-empty arrays
      awk -v colony="$colony_name" '
        /"colonies_contributed": \[\]/ {
          gsub(/"colonies_contributed": \[\]/, "\"colonies_contributed\": [\"" colony "\"]")
          print
          next
        }
        /"colonies_contributed": \[/ && !/\]/ {
          # Multi-line array, add at next closing bracket
          print
          next
        }
        /"colonies_contributed": \[/ {
          # Single-line array with elements
          gsub(/\]$/, "\"" colony "\", ]")
          print
          next
        }
        { print }
      ' "$tmp_file" > "${tmp_file}.col" && mv "${tmp_file}.col" "$tmp_file"
    fi

    # Restore default cleanup trap before final move (file becomes permanent)
    trap '_aether_exit_cleanup 2>/dev/null || true' EXIT TERM INT HUP

    # Safety guard: never overwrite QUEEN.md with empty content
    if [[ ! -s "$tmp_file" ]]; then
      rm -f "$tmp_file"
      json_err "$E_INTERNAL" "queen-promote produced empty output — aborting to protect QUEEN.md"
      return 1
    fi

    # Atomic move
    mv "$tmp_file" "$queen_file"

    json_ok "$(jq -n --arg type "$wisdom_type" --arg colony "$colony_name" --arg ts "$ts" --argjson threshold "$threshold" --argjson new_count "$new_count" --arg hash "$content_hash" '{promoted: true, type: $type, colony: $colony, timestamp: $ts, threshold: $threshold, new_count: $new_count, content_hash: $hash}')"
}

# ============================================================================
# _queen_write_learnings
# Write build learnings directly to QUEEN.md Build Learnings section
# Bypasses observation thresholds -- every build writes learnings
# Usage: queen-write-learnings <phase_id> <phase_name> <learnings_json>
# learnings_json: [{"claim":"what happened","tag":"repo|general","evidence":"..."}]
# ============================================================================
_queen_write_learnings() {
    local phase_id="${1:-}"
    local phase_name="${2:-}"
    local learnings_json="${3:-}"

    # Validate required arguments
    [[ -z "$phase_id" ]] && { json_err "$E_VALIDATION_FAILED" "Usage: queen-write-learnings <phase_id> <phase_name> <learnings_json>" '{"missing":"phase_id"}'; return 1; }
    [[ -z "$phase_name" ]] && { json_err "$E_VALIDATION_FAILED" "Usage: queen-write-learnings <phase_id> <phase_name> <learnings_json>" '{"missing":"phase_name"}'; return 1; }
    [[ -z "$learnings_json" ]] && { json_err "$E_VALIDATION_FAILED" "Usage: queen-write-learnings <phase_id> <phase_name> <learnings_json>" '{"missing":"learnings_json"}'; return 1; }

    # Validate learnings_json is valid JSON array
    if ! echo "$learnings_json" | jq -e 'type == "array"' >/dev/null 2>&1; then
      json_err "$E_JSON_INVALID" "learnings_json must be a JSON array" '{"received":"not_array"}'
      return 1
    fi

    local queen_file="$AETHER_ROOT/.aether/QUEEN.md"

    if [[ ! -f "$queen_file" ]]; then
      json_err "$E_FILE_NOT_FOUND" "QUEEN.md not found. Run queen-init first." '{"path":".aether/QUEEN.md"}'
      return 1
    fi

    local ts
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local date_short
    date_short=$(date -u +"%Y-%m-%d")

    # Build entries from learnings_json
    local written=0
    local entries=""
    local subsection_header="### Phase ${phase_id}: ${phase_name}"

    while IFS= read -r encoded_entry; do
      [[ -z "$encoded_entry" ]] && continue
      local claim tag evidence
      claim=$(echo "$encoded_entry" | base64 -d 2>/dev/null | jq -r '.claim // ""' 2>/dev/null) || continue
      tag=$(echo "$encoded_entry" | base64 -d 2>/dev/null | jq -r '.tag // "repo"' 2>/dev/null) || tag="repo"
      evidence=$(echo "$encoded_entry" | base64 -d 2>/dev/null | jq -r '.evidence // ""' 2>/dev/null) || evidence=""

      [[ -z "$claim" ]] && continue

      # Dedup check: skip if claim already in QUEEN.md
      if grep -Fq -- "$claim" "$queen_file" 2>/dev/null; then
        continue
      fi

      local entry_line="- [${tag}] ${claim} -- *Phase ${phase_id} (${phase_name})* (${date_short})"
      entries="${entries}${entry_line}"$'\n'
      written=$((written + 1))
    done < <(echo "$learnings_json" | jq -r '.[] | @base64')

    # If nothing to write, return early
    if [[ "$written" -eq 0 ]]; then
      json_ok "$(jq -n --arg phase "$phase_id" --arg ts "$ts" \
        '{written: 0, phase: $phase, timestamp: $ts, reason: "all_duplicates_or_empty"}')"
      return 0
    fi

    # Create temp file for atomic write
    local tmp_file="${queen_file}.tmp.$$"
    cp "$queen_file" "$tmp_file"

    # Find Build Learnings section
    local section_line
    section_line=$(grep -n '^## Build Learnings$' "$tmp_file" | head -1 | cut -d: -f1)

    if [[ -z "$section_line" ]]; then
      rm -f "$tmp_file"
      json_err "$E_VALIDATION_FAILED" "Build Learnings section not found in QUEEN.md. Is this a v2 format file?" '{"section":"Build Learnings"}'
      return 1
    fi

    # Find end of section (next ## header or end of file)
    local next_section_line
    next_section_line=$(tail -n +$((section_line + 1)) "$tmp_file" | grep -n "^## " | head -1 | cut -d: -f1)
    local section_end
    if [[ -n "$next_section_line" ]]; then
      section_end=$((section_line + next_section_line - 1))
    else
      section_end=$(wc -l < "$tmp_file")
    fi

    # Remove placeholder if present
    # SUPPRESS:OK -- existence-test: grep returns 1 when no matches
    local has_placeholder
    has_placeholder=$(sed -n "${section_line},${section_end}p" "$tmp_file" | grep -c "No build learnings recorded yet" || true)
    has_placeholder=${has_placeholder:-0}

    if [[ "$has_placeholder" -gt 0 ]]; then
      local placeholder_line
      placeholder_line=$(sed -n "${section_line},${section_end}p" "$tmp_file" | grep -n "^\\*No build learnings recorded yet" | head -1 | cut -d: -f1)
      if [[ -n "$placeholder_line" ]]; then
        local actual_line=$((section_line + placeholder_line - 1))
        sed -i.bak "${actual_line}d" "$tmp_file" && rm -f "${tmp_file}.bak"
        section_end=$((section_end - 1))
      fi
    fi

    # Check if subsection header already exists
    local has_subsection
    has_subsection=$(sed -n "${section_line},${section_end}p" "$tmp_file" | grep -c "^### Phase ${phase_id}:" || true)
    has_subsection=${has_subsection:-0}

    if [[ "$has_subsection" -gt 0 ]]; then
      # Find the subsection header line and append entries after it
      local sub_line
      sub_line=$(sed -n "${section_line},${section_end}p" "$tmp_file" | grep -n "^### Phase ${phase_id}:" | head -1 | cut -d: -f1)
      local actual_sub_line=$((section_line + sub_line - 1))
      # Find end of subsection (next ### or next ## or --- or end of section)
      local sub_end_rel
      sub_end_rel=$(tail -n +$((actual_sub_line + 1)) "$tmp_file" | grep -n "^###\|^## \|^---$" | head -1 | cut -d: -f1)
      local insert_at
      if [[ -n "$sub_end_rel" ]]; then
        insert_at=$((actual_sub_line + sub_end_rel - 1))
      else
        insert_at=$section_end
      fi
      # Insert entries before the boundary
      local temp_entries="${tmp_file}.entries"
      {
        head -n "$insert_at" "$tmp_file"
        printf '%s' "$entries"
        tail -n +$((insert_at + 1)) "$tmp_file"
      } > "$temp_entries" && mv "$temp_entries" "$tmp_file"
    else
      # Create new subsection at end of Build Learnings section (before --- or next ##)
      # Find last content line in section (skip trailing --- and blanks)
      local insert_at
      # Insert before the separator (---) that ends the section, or before next ## header
      local sep_line
      sep_line=$(sed -n "${section_line},${section_end}p" "$tmp_file" | grep -n "^---$" | tail -1 | cut -d: -f1)
      if [[ -n "$sep_line" ]]; then
        insert_at=$((section_line + sep_line - 2))
      else
        insert_at=$section_end
      fi

      local temp_entries="${tmp_file}.entries"
      {
        head -n "$insert_at" "$tmp_file"
        echo ""
        echo "$subsection_header"
        printf '%s' "$entries"
        tail -n +$((insert_at + 1)) "$tmp_file"
      } > "$temp_entries" && mv "$temp_entries" "$tmp_file"
    fi

    # Update Evolution Log
    local ev_entry="| ${ts} | phase-${phase_id} | build_learnings | Added ${written} learnings from Phase ${phase_id}: ${phase_name} |"
    local ev_separator
    ev_separator=$(grep -n "^|------|" "$tmp_file" | tail -1 | cut -d: -f1 || true)
    if [[ -n "$ev_separator" ]]; then
      awk -v line="$ev_separator" -v entry="$ev_entry" 'NR==line{print; print entry; next}1' "$tmp_file" > "${tmp_file}.ev" && mv "${tmp_file}.ev" "$tmp_file"
    fi

    # Update METADATA stats: increment total_build_learnings
    local current_count
    current_count=$(grep '"total_build_learnings":' "$tmp_file" 2>/dev/null | grep -o '[0-9]*' | head -1 || true)
    current_count=${current_count:-0}
    local new_count=$((current_count + written))
    awk -v count="$new_count" '{
      gsub(/"total_build_learnings": [0-9]*/, "\"total_build_learnings\": " count)
      print
    }' "$tmp_file" > "${tmp_file}.stats" && mv "${tmp_file}.stats" "$tmp_file"

    # Update last_evolved
    awk -v ts="$ts" '/"last_evolved":/ { gsub(/"last_evolved": "[^"]*"/, "\"last_evolved\": \"" ts "\""); } {print}' "$tmp_file" > "${tmp_file}.meta" && mv "${tmp_file}.meta" "$tmp_file"

    # Safety guard: never overwrite QUEEN.md with empty content
    if [[ ! -s "$tmp_file" ]]; then
      rm -f "$tmp_file"
      json_err "$E_INTERNAL" "queen-write-learnings produced empty output — aborting to protect QUEEN.md"
      return 1
    fi

    # Atomic move
    mv "$tmp_file" "$queen_file"

    json_ok "$(jq -n --argjson written "$written" --arg phase "$phase_id" --arg ts "$ts" \
      '{written: $written, phase: $phase, timestamp: $ts}')"
}

# ============================================================================
# _queen_promote_instinct
# Promote a high-confidence instinct to QUEEN.md Instincts section
# Usage: queen-promote-instinct <trigger> <action> [confidence] [domain]
# ============================================================================
_queen_promote_instinct() {
    local trigger="${1:-}"
    local action="${2:-}"
    local confidence="${3:-0.8}"
    local domain="${4:-workflow}"

    # Validate required arguments
    [[ -z "$trigger" ]] && { json_err "$E_VALIDATION_FAILED" "Usage: queen-promote-instinct <trigger> <action> [confidence] [domain]" '{"missing":"trigger"}'; return 1; }
    [[ -z "$action" ]] && { json_err "$E_VALIDATION_FAILED" "Usage: queen-promote-instinct <trigger> <action> [confidence] [domain]" '{"missing":"action"}'; return 1; }

    local queen_file="$AETHER_ROOT/.aether/QUEEN.md"

    if [[ ! -f "$queen_file" ]]; then
      json_err "$E_FILE_NOT_FOUND" "QUEEN.md not found. Run queen-init first." '{"path":".aether/QUEEN.md"}'
      return 1
    fi

    # Dedup check: skip if action already in QUEEN.md
    if grep -Fq -- "$action" "$queen_file" 2>/dev/null; then
      json_ok '{"promoted":false,"written":0,"reason":"duplicate"}'
      return 0
    fi

    local ts
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Build entry
    local entry="- [instinct] **${domain}** (${confidence}): When ${trigger}, then ${action}"

    # Create temp file for atomic write
    local tmp_file="${queen_file}.tmp.$$"
    cp "$queen_file" "$tmp_file"

    # Find Instincts section
    local section_line
    section_line=$(grep -n '^## Instincts$' "$tmp_file" | head -1 | cut -d: -f1)

    if [[ -z "$section_line" ]]; then
      rm -f "$tmp_file"
      json_err "$E_VALIDATION_FAILED" "Instincts section not found in QUEEN.md. Is this a v2 format file?" '{"section":"Instincts"}'
      return 1
    fi

    # Find end of section
    local next_section_line
    next_section_line=$(tail -n +$((section_line + 1)) "$tmp_file" | grep -n "^## " | head -1 | cut -d: -f1)
    local section_end
    if [[ -n "$next_section_line" ]]; then
      section_end=$((section_line + next_section_line - 1))
    else
      section_end=$(wc -l < "$tmp_file")
    fi

    # Check for placeholder and replace or append
    # SUPPRESS:OK -- existence-test: grep returns 1 when no matches
    local has_placeholder
    has_placeholder=$(sed -n "${section_line},${section_end}p" "$tmp_file" | grep -c "No instincts recorded yet" || true)
    has_placeholder=${has_placeholder:-0}

    if [[ "$has_placeholder" -gt 0 ]]; then
      local placeholder_line
      placeholder_line=$(sed -n "${section_line},${section_end}p" "$tmp_file" | grep -n "^\\*No instincts recorded yet" | head -1 | cut -d: -f1)
      if [[ -n "$placeholder_line" ]]; then
        local actual_line=$((section_line + placeholder_line - 1))
        # Use head/tail instead of sed c-command for newline safety
        {
          head -n $((actual_line - 1)) "$tmp_file"
          echo "$entry"
          tail -n +$((actual_line + 1)) "$tmp_file"
        } > "${tmp_file}.rep" && mv "${tmp_file}.rep" "$tmp_file"
      fi
    else
      # Append entry before section separator (---) or at end
      local sep_line
      sep_line=$(sed -n "${section_line},${section_end}p" "$tmp_file" | grep -n "^---$" | tail -1 | cut -d: -f1)
      local insert_at
      if [[ -n "$sep_line" ]]; then
        insert_at=$((section_line + sep_line - 2))
      else
        insert_at=$section_end
      fi

      local temp_entries="${tmp_file}.entries"
      {
        head -n "$insert_at" "$tmp_file"
        echo "$entry"
        tail -n +$((insert_at + 1)) "$tmp_file"
      } > "$temp_entries" && mv "$temp_entries" "$tmp_file"
    fi

    # Update Evolution Log
    local ev_entry="| ${ts} | instinct | promoted_instinct | ${domain}: ${action:0:50}... |"
    local ev_separator
    ev_separator=$(grep -n "^|------|" "$tmp_file" | tail -1 | cut -d: -f1 || true)
    if [[ -n "$ev_separator" ]]; then
      awk -v line="$ev_separator" -v entry="$ev_entry" 'NR==line{print; print entry; next}1' "$tmp_file" > "${tmp_file}.ev" && mv "${tmp_file}.ev" "$tmp_file"
    fi

    # Update METADATA stats: increment total_instincts
    local current_count
    current_count=$(grep '"total_instincts":' "$tmp_file" 2>/dev/null | grep -o '[0-9]*' | head -1 || true)
    current_count=${current_count:-0}
    local new_count=$((current_count + 1))
    awk -v count="$new_count" '{
      gsub(/"total_instincts": [0-9]*/, "\"total_instincts\": " count)
      print
    }' "$tmp_file" > "${tmp_file}.stats" && mv "${tmp_file}.stats" "$tmp_file"

    # Update last_evolved
    awk -v ts="$ts" '/"last_evolved":/ { gsub(/"last_evolved": "[^"]*"/, "\"last_evolved\": \"" ts "\""); } {print}' "$tmp_file" > "${tmp_file}.meta" && mv "${tmp_file}.meta" "$tmp_file"

    # Safety guard: never overwrite QUEEN.md with empty content
    if [[ ! -s "$tmp_file" ]]; then
      rm -f "$tmp_file"
      json_err "$E_INTERNAL" "queen-promote-instinct produced empty output — aborting to protect QUEEN.md"
      return 1
    fi

    # Atomic move
    mv "$tmp_file" "$queen_file"

    json_ok "$(jq -n --arg domain "$domain" --argjson confidence "$confidence" --arg ts "$ts" \
      '{promoted: true, written: 1, domain: $domain, confidence: $confidence, timestamp: $ts}')"
}

# ============================================================================
# _queen_seed_from_hive
# Seed QUEEN.md Codebase Patterns section from cross-colony hive wisdom
# Usage: queen-seed-from-hive [--domain <csv>] [--limit <N>]
# Writes [hive]-tagged entries to Codebase Patterns. NON-BLOCKING.
# ============================================================================
_queen_seed_from_hive() {
    local qs_domain=""
    local qs_limit="5"

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --domain) qs_domain="${2:-}"; shift 2 ;;
        --limit)  qs_limit="${2:-5}"; shift 2 ;;
        *) shift ;;
      esac
    done

    local queen_file="$AETHER_ROOT/.aether/QUEEN.md"
    [[ -f "$queen_file" ]] || { json_ok '{"seeded":0,"reason":"no_queen_md"}'; return 0; }

    # Read hive wisdom with domain filter
    local hive_args=(hive-read --limit "$qs_limit" --min-confidence 0.5 --format json)
    [[ -n "$qs_domain" ]] && hive_args+=(--domain "$qs_domain")
    local hive_result
    hive_result=$(bash "$0" "${hive_args[@]}" 2>/dev/null) || { json_ok '{"seeded":0,"reason":"hive_read_failed"}'; return 0; }

    local entry_count
    entry_count=$(echo "$hive_result" | jq -r '.result.total_matched // 0' 2>/dev/null)
    [[ "$entry_count" -eq 0 ]] && { json_ok '{"seeded":0,"reason":"no_matching_wisdom"}'; return 0; }

    # Build entries for QUEEN.md Codebase Patterns section
    local entries=""
    local seeded=0
    while IFS= read -r encoded; do
      [[ -z "$encoded" ]] && continue
      local text confidence
      text=$(echo "$encoded" | base64 -d | jq -r '.text // empty')
      confidence=$(echo "$encoded" | base64 -d | jq -r '.confidence // 0')
      [[ -z "$text" ]] && continue

      # Dedup: skip if already in QUEEN.md
      if grep -Fq -- "$text" "$queen_file" 2>/dev/null; then continue; fi

      entries="${entries}- [hive] ${text} (cross-colony, confidence: ${confidence})"$'\n'
      seeded=$((seeded + 1))
    done < <(echo "$hive_result" | jq -r '.result.entries[] | @base64')

    [[ "$seeded" -eq 0 ]] && { json_ok '{"seeded":0,"reason":"all_duplicates"}'; return 0; }

    # Write to Codebase Patterns section (reuse placeholder removal pattern from _queen_write_learnings)
    local tmp_file="${queen_file}.tmp.$$"
    cp "$queen_file" "$tmp_file"

    local section_line
    section_line=$(grep -n '^## Codebase Patterns$' "$tmp_file" | head -1 | cut -d: -f1)

    if [[ -z "$section_line" ]]; then
      rm -f "$tmp_file"
      json_ok '{"seeded":0,"reason":"no_codebase_patterns_section"}'
      return 0
    fi

    # Find end of section (next ## header or end of file)
    local next_section_line
    next_section_line=$(tail -n +$((section_line + 1)) "$tmp_file" | grep -n "^## " | head -1 | cut -d: -f1)
    local section_end
    if [[ -n "$next_section_line" ]]; then
      section_end=$((section_line + next_section_line - 1))
    else
      section_end=$(wc -l < "$tmp_file")
    fi

    # Remove placeholder if present
    # SUPPRESS:OK -- existence-test: grep returns 1 when no matches
    local has_placeholder
    has_placeholder=$(sed -n "${section_line},${section_end}p" "$tmp_file" | grep -c "No codebase patterns recorded yet" || true)
    has_placeholder=${has_placeholder:-0}

    if [[ "$has_placeholder" -gt 0 ]]; then
      local placeholder_line
      placeholder_line=$(sed -n "${section_line},${section_end}p" "$tmp_file" | grep -n "^\\*No codebase patterns recorded yet" | head -1 | cut -d: -f1)
      if [[ -n "$placeholder_line" ]]; then
        local actual_line=$((section_line + placeholder_line - 1))
        sed -i.bak "${actual_line}d" "$tmp_file" && rm -f "${tmp_file}.bak"
        section_end=$((section_end - 1))
      fi
    fi

    # Insert entries before the separator (---) that ends the section, or at end
    local sep_line
    sep_line=$(sed -n "${section_line},${section_end}p" "$tmp_file" | grep -n "^---$" | tail -1 | cut -d: -f1)
    local insert_at
    if [[ -n "$sep_line" ]]; then
      insert_at=$((section_line + sep_line - 2))
    else
      insert_at=$section_end
    fi

    local temp_entries="${tmp_file}.entries"
    {
      head -n "$insert_at" "$tmp_file"
      printf '%s' "$entries"
      tail -n +$((insert_at + 1)) "$tmp_file"
    } > "$temp_entries" && mv "$temp_entries" "$tmp_file"

    # Update Evolution Log with seed event
    local ts
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local ev_entry="| ${ts} | hive | seed | Seeded ${seeded} cross-colony patterns from hive |"
    local ev_separator
    ev_separator=$(grep -n "^|------|" "$tmp_file" | tail -1 | cut -d: -f1 || true)
    if [[ -n "$ev_separator" ]]; then
      awk -v line="$ev_separator" -v entry="$ev_entry" 'NR==line{print; print entry; next}1' "$tmp_file" > "${tmp_file}.ev" && mv "${tmp_file}.ev" "$tmp_file"
    fi

    # Update METADATA stats: increment total_codebase_patterns
    local current_count
    current_count=$(grep '"total_codebase_patterns":' "$tmp_file" 2>/dev/null | grep -o '[0-9]*' | head -1 || true)
    current_count=${current_count:-0}
    local new_count=$((current_count + seeded))
    awk -v count="$new_count" '{
      gsub(/"total_codebase_patterns": [0-9]*/, "\"total_codebase_patterns\": " count)
      print
    }' "$tmp_file" > "${tmp_file}.stats" && mv "${tmp_file}.stats" "$tmp_file"

    # Update last_evolved
    awk -v ts="$ts" '/"last_evolved":/ { gsub(/"last_evolved": "[^"]*"/, "\"last_evolved\": \"" ts "\""); } {print}' "$tmp_file" > "${tmp_file}.meta" && mv "${tmp_file}.meta" "$tmp_file"

    # Safety guard: never overwrite QUEEN.md with empty content
    if [[ ! -s "$tmp_file" ]]; then
      rm -f "$tmp_file"
      json_err "$E_INTERNAL" "queen-seed-from-hive produced empty output — aborting to protect QUEEN.md"
      return 1
    fi

    # Atomic move
    mv "$tmp_file" "$queen_file"

    json_ok "$(jq -n --argjson seeded "$seeded" '{seeded: $seeded}')"
}

# ============================================================================
# _domain_detect
# Auto-detect repo domain tags from file/directory presence
# Usage: domain-detect
# Returns: {"tags":"node,typescript,..."}
# ============================================================================
_domain_detect() {
    local tags=""
    local root="${AETHER_ROOT:-.}"

    [[ -f "$root/package.json" ]] && tags="${tags:+$tags,}node"
    [[ -f "$root/tsconfig.json" ]] && tags="${tags:+$tags,}typescript"
    [[ -f "$root/Cargo.toml" ]] && tags="${tags:+$tags,}rust"
    [[ -f "$root/go.mod" ]] && tags="${tags:+$tags,}go"
    [[ -f "$root/requirements.txt" || -f "$root/pyproject.toml" ]] && tags="${tags:+$tags,}python"
    [[ -d "$root/wp-content" || -f "$root/wp-config.php" ]] && tags="${tags:+$tags,}wordpress"
    [[ -f "$root/Gemfile" ]] && tags="${tags:+$tags,}ruby"
    [[ -f "$root/.aether/aether-utils.sh" ]] && tags="${tags:+$tags,}aether"
    [[ -d "$root/.next" || -f "$root/next.config.js" || -f "$root/next.config.ts" ]] && tags="${tags:+$tags,}nextjs"
    [[ -f "$root/docker-compose.yml" || -f "$root/Dockerfile" ]] && tags="${tags:+$tags,}docker"

    json_ok "$(jq -n --arg tags "$tags" '{tags: $tags}')"
}

# ============================================================================
# _queen_migrate
# Convert a v1 QUEEN.md (emoji headers) to v2 (clean headers) format
# Usage: queen-migrate [--target hub|local]
# --target hub  -> migrates $HOME/.aether/QUEEN.md
# --target local -> migrates $AETHER_ROOT/.aether/QUEEN.md (default)
# If already v2 format, prints message and exits 0.
# ============================================================================
_queen_migrate() {
    local qm_target="local"

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --target) qm_target="${2:-local}"; shift 2 ;;
        *) shift ;;
      esac
    done

    local qm_file
    if [[ "$qm_target" == "hub" ]]; then
      qm_file="$HOME/.aether/QUEEN.md"
    else
      qm_file="$AETHER_ROOT/.aether/QUEEN.md"
    fi

    if [[ ! -f "$qm_file" ]]; then
      json_err "$E_FILE_NOT_FOUND" "QUEEN.md not found at $qm_file" '{"target":"'"$qm_target"'"}'
      exit 1
    fi

    # Check if already v2 format
    if grep -q '^## Build Learnings$' "$qm_file" 2>/dev/null; then  # SUPPRESS:OK -- existence-test: format detection
      json_ok '{"migrated":false,"reason":"Already v2 format"}'
      return 0
    fi

    # Extract content from v1 format using _extract_wisdom_sections (maps v1 -> v2 keys)
    local qm_wisdom
    qm_wisdom=$(_extract_wisdom_sections "$qm_file")

    # Extract real entries from each section
    # v1 _extract_wisdom_sections produces double-encoded JSON strings, so we need
    # to unwrap them: jq -r removes outer quotes, then sed strips remaining inner quotes
    local qm_uprefs qm_codebase qm_learnings qm_instincts
    qm_uprefs=$(echo "$qm_wisdom" | jq -r '.user_prefs // ""' 2>/dev/null | sed 's/^"//;s/"$//' | sed 's/\\n/\n/g')  # SUPPRESS:OK -- read-default: may be empty
    qm_uprefs=$(echo "$qm_uprefs" | grep -E '^- ' || true)  # SUPPRESS:OK -- grep returns 1 on no matches
    qm_codebase=$(echo "$qm_wisdom" | jq -r '.codebase_patterns // ""' 2>/dev/null | sed 's/^"//;s/"$//' | sed 's/\\n/\n/g')  # SUPPRESS:OK -- read-default: may be empty
    qm_codebase=$(echo "$qm_codebase" | grep -E '^- ' || true)  # SUPPRESS:OK -- grep returns 1 on no matches
    qm_learnings=$(echo "$qm_wisdom" | jq -r '.build_learnings // ""' 2>/dev/null | sed 's/^"//;s/"$//' | sed 's/\\n/\n/g')  # SUPPRESS:OK -- read-default: may be empty
    qm_learnings=$(echo "$qm_learnings" | grep -E '^- ' || true)  # SUPPRESS:OK -- grep returns 1 on no matches
    qm_instincts=$(echo "$qm_wisdom" | jq -r '.instincts // ""' 2>/dev/null | sed 's/^"//;s/"$//' | sed 's/\\n/\n/g')  # SUPPRESS:OK -- read-default: may be empty
    qm_instincts=$(echo "$qm_instincts" | grep -E '^- ' || true)  # SUPPRESS:OK -- grep returns 1 on no matches

    local qm_ts
    qm_ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Build fresh v2 QUEEN.md
    local qm_tmp="${qm_file}.migrate.$$"
    cat > "$qm_tmp" << MIGRATEEOF
# QUEEN.md -- Colony Wisdom

> Last evolved: $qm_ts
> Wisdom version: 2.0.0

---

## User Preferences

Communication style, expertise level, and decision-making patterns observed from the user (the Queen). These shape how the colony communicates and what it prioritizes.

MIGRATEEOF

    if [[ -n "$qm_uprefs" ]]; then
      echo "$qm_uprefs" >> "$qm_tmp"
    else
      echo "*No user preferences recorded yet.*" >> "$qm_tmp"
    fi

    cat >> "$qm_tmp" << 'MIGRATEEOF'

---

## Codebase Patterns

Validated approaches that work in this codebase, and anti-patterns to avoid. Includes architecture conventions, naming patterns, error handling style, and technology-specific insights.

MIGRATEEOF

    if [[ -n "$qm_codebase" ]]; then
      echo "$qm_codebase" >> "$qm_tmp"
    else
      echo "*No codebase patterns recorded yet.*" >> "$qm_tmp"
    fi

    cat >> "$qm_tmp" << 'MIGRATEEOF'

---

## Build Learnings

What worked and what failed during builds. Captures the full picture of colony experience -- successes, failures, and adjustments.

MIGRATEEOF

    if [[ -n "$qm_learnings" ]]; then
      echo "$qm_learnings" >> "$qm_tmp"
    else
      echo "*No build learnings recorded yet.*" >> "$qm_tmp"
    fi

    cat >> "$qm_tmp" << 'MIGRATEEOF'

---

## Instincts

High-confidence behavioral patterns that have been validated through repeated colony work. Auto-promoted when confidence reaches 0.8 or higher.

MIGRATEEOF

    if [[ -n "$qm_instincts" ]]; then
      echo "$qm_instincts" >> "$qm_tmp"
    else
      echo "*No instincts recorded yet.*" >> "$qm_tmp"
    fi

    cat >> "$qm_tmp" << MIGRATEEOF

---

## Evolution Log

| Date | Source | Type | Details |
|------|--------|------|---------|
| $qm_ts | system | migration | Migrated from v1 to v2 format |

---

<!-- METADATA {"version":"2.0.0","wisdom_version":"2.0","last_evolved":"$qm_ts","colonies_contributed":[],"stats":{"total_user_prefs":0,"total_codebase_patterns":0,"total_build_learnings":0,"total_instincts":0}} -->
MIGRATEEOF

    # Atomic move
    mv "$qm_tmp" "$qm_file"

    json_ok '{"migrated":true,"target":"'"$qm_target"'","format":"v2"}'
}

# ============================================================================
# _colony_name()
# Derive human-readable colony name from repo context
# Fallback chain: COLONY_STATE.json -> package.json -> directory basename
# Usage: colony-name
# Returns: {"ok":true,"result":{"name":"Aether Colony","source":"colony_state"}}
# ============================================================================
_colony_name() {
    local name=""
    local source="directory"

    # 1. Check COLONY_STATE.json for pre-set colony_name
    if [[ -f "$DATA_DIR/COLONY_STATE.json" ]]; then
        local preset
        preset=$(jq -r '.colony_name // empty' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null || echo "")
        if [[ -n "$preset" ]]; then
            name="$preset"
            source="colony_state"
        fi
    fi

    # 2. Fall back to package.json name
    if [[ -z "$name" ]] && [[ -f "$AETHER_ROOT/package.json" ]]; then
        local pkg_name
        pkg_name=$(jq -r '.name // empty' "$AETHER_ROOT/package.json" 2>/dev/null || echo "")
        if [[ -n "$pkg_name" ]]; then
            # Strip @scope/ prefix
            name="${pkg_name#@*/}"
            source="package_json"
        fi
    fi

    # 3. Fall back to directory basename
    if [[ -z "$name" ]]; then
        name="$(basename "$AETHER_ROOT")"
        source="directory"
    fi

    # Convert kebab-case to title case
    name=$(echo "$name" | sed 's/-/ /g' | awk '{for(i=1;i<=NF;i++) $i=toupper(substr($i,1,1)) substr($i,2)};1')

    json_ok "$(jq -n --arg name "$name" --arg source "$source" '{name: $name, source: $source}')"
}

# ============================================================================
# _queen_write_charter()
# Write or update colony charter content in QUEEN.md using [charter] tags
# Writes intent+vision to User Preferences, governance+goals to Codebase Patterns
# Handles re-init: removes existing [charter] entries before writing new ones
# Usage: charter-write --intent "text" --vision "text" --governance "text" --goals "text" [--colony-name "name"]
# Returns: {"ok":true,"result":{"written":N,"updated":bool,"colony_name":"...","timestamp":"..."}}
# ============================================================================
_queen_write_charter() {
    local cw_intent=""
    local cw_vision=""
    local cw_governance=""
    local cw_goals=""
    local cw_colony_name=""

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --intent)      cw_intent="${2:-}"; shift 2 ;;
            --vision)      cw_vision="${2:-}"; shift 2 ;;
            --governance)  cw_governance="${2:-}"; shift 2 ;;
            --goals)       cw_goals="${2:-}"; shift 2 ;;
            --colony-name) cw_colony_name="${2:-}"; shift 2 ;;
            *) shift ;;
        esac
    done

    # At least one content field must be provided
    if [[ -z "$cw_intent" && -z "$cw_vision" && -z "$cw_governance" && -z "$cw_goals" ]]; then
        json_err "$E_VALIDATION_FAILED" "Usage: charter-write --intent <text> --vision <text> --governance <text> --goals <text> [--colony-name <name>]" '{"missing":"all_fields"}'
        return 1
    fi

    local queen_file="$AETHER_ROOT/.aether/QUEEN.md"
    if [[ ! -f "$queen_file" ]]; then
        json_err "$E_FILE_NOT_FOUND" "QUEEN.md not found. Run queen-init first." '{"path":".aether/QUEEN.md"}'
        return 1
    fi

    local ts
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Derive colony name if not provided
    if [[ -z "$cw_colony_name" ]]; then
        local cn_result
        cn_result=$(bash "$0" colony-name 2>/dev/null) || true
        cw_colony_name=$(echo "$cn_result" | jq -r '.result.name // ""' 2>/dev/null) || true
    fi

    # Cap each content field at 200 characters
    _cap_200() {
        local val="$1"
        if [[ ${#val} -gt 200 ]]; then
            echo "${val:0:197}..."
        else
            echo "$val"
        fi
    }
    cw_intent=$(_cap_200 "$cw_intent")
    cw_vision=$(_cap_200 "$cw_vision")
    cw_governance=$(_cap_200 "$cw_governance")
    cw_goals=$(_cap_200 "$cw_goals")

    # Auto-set colony_name in COLONY_STATE.json if not already set
    if [[ -f "$DATA_DIR/COLONY_STATE.json" ]]; then
        local current_name
        current_name=$(jq -r '.colony_name // empty' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null) || true
        if [[ -z "$current_name" && -n "$cw_colony_name" ]]; then
            local tmp_state="${DATA_DIR}/COLONY_STATE.json.tmp.$$"
            jq --arg cn "$cw_colony_name" '.colony_name = $cn' "$DATA_DIR/COLONY_STATE.json" > "$tmp_state" && mv "$tmp_state" "$DATA_DIR/COLONY_STATE.json"
        fi
    fi

    # Create temp file for atomic write
    local tmp_file="${queen_file}.tmp.$$"
    cp "$queen_file" "$tmp_file"

    # Count existing charter entries before removal (to detect re-init)
    local existing_charter_count
    existing_charter_count=$(grep -c '^- \[charter\] ' "$tmp_file" 2>/dev/null || true)
    existing_charter_count=${existing_charter_count:-0}
    local is_update="false"
    if [[ "$existing_charter_count" -gt 0 ]]; then
        is_update="true"
    fi

    # Remove ALL existing charter entries (re-init safety -- CHARTER-03)
    sed -i.bak '/^- \[charter\] /d' "$tmp_file" && rm -f "${tmp_file}.bak"

    # Helper: insert entries into a QUEEN.md section
    # Usage: _insert_section_entries <tmp_file> <section_name> <placeholder_text> <entries>
    _insert_section_entries() {
        local is_file="$1"
        local is_section="$2"
        local is_placeholder="$3"
        local is_entries="$4"
        local is_tmp="${is_file}.insert"

        local is_section_line
        is_section_line=$(grep -n "^## ${is_section}\$" "$is_file" | head -1 | cut -d: -f1)

        if [[ -z "$is_section_line" ]]; then
            return 1
        fi

        # Find end of section (next ## header or end of file)
        local is_next_section
        is_next_section=$(tail -n +$((is_section_line + 1)) "$is_file" | grep -n "^## " | head -1 | cut -d: -f1)
        local is_section_end
        if [[ -n "$is_next_section" ]]; then
            is_section_end=$((is_section_line + is_next_section - 1))
        else
            is_section_end=$(wc -l < "$is_file")
        fi

        # Remove placeholder if present
        # SUPPRESS:OK -- existence-test: grep returns 1 when no matches
        local is_has_placeholder
        is_has_placeholder=$(sed -n "${is_section_line},${is_section_end}p" "$is_file" | grep -c "No .* recorded yet" || true)
        is_has_placeholder=${is_has_placeholder:-0}

        if [[ "$is_has_placeholder" -gt 0 ]]; then
            local is_pl_line
            is_pl_line=$(sed -n "${is_section_line},${is_section_end}p" "$is_file" | grep -n "^\\*No .* recorded yet" | head -1 | cut -d: -f1)
            if [[ -n "$is_pl_line" ]]; then
                local is_actual_line=$((is_section_line + is_pl_line - 1))
                # Replace placeholder with entries
                {
                    head -n $((is_actual_line - 1)) "$is_file"
                    printf '%s\n' "$is_entries"
                    tail -n +$((is_actual_line + 1)) "$is_file"
                } > "$is_tmp" && mv "$is_tmp" "$is_file"
                return 0
            fi
        fi

        # No placeholder -- insert before section separator (---) or at section end
        local is_sep_line
        is_sep_line=$(sed -n "${is_section_line},${is_section_end}p" "$is_file" | grep -n "^---$" | tail -1 | cut -d: -f1)
        local is_insert_at
        if [[ -n "$is_sep_line" ]]; then
            is_insert_at=$((is_section_line + is_sep_line - 2))
        else
            is_insert_at=$is_section_end
        fi

        {
            head -n "$is_insert_at" "$is_file"
            printf '%s\n' "$is_entries"
            tail -n +$((is_insert_at + 1)) "$is_file"
        } > "$is_tmp" && mv "$is_tmp" "$is_file"
        return 0
    }

    # Build User Preferences entries (intent + vision)
    local up_entries=""
    local written=0
    local up_failed=0
    local cp_failed=0
    if [[ -n "$cw_intent" ]]; then
        up_entries="${up_entries}- [charter] **Intent**: ${cw_intent} (Colony: ${cw_colony_name})"
        written=$((written + 1))
    fi
    if [[ -n "$cw_vision" ]]; then
        up_entries="${up_entries}"$'\n'"- [charter] **Vision**: ${cw_vision} (Colony: ${cw_colony_name})"
        written=$((written + 1))
    fi

    if [[ -n "$up_entries" ]]; then
        if ! _insert_section_entries "$tmp_file" "User Preferences" "user preferences" "$up_entries"; then
            up_failed=1
            echo "[charter-write] WARNING: '## User Preferences' section not found in QUEEN.md; entries dropped" >&2
        fi
    fi

    # Build Codebase Patterns entries (governance + goals)
    local cp_entries=""
    if [[ -n "$cw_governance" ]]; then
        cp_entries="${cp_entries}- [charter] **Governance**: ${cw_governance} (Colony: ${cw_colony_name})"
        written=$((written + 1))
    fi
    if [[ -n "$cw_goals" ]]; then
        cp_entries="${cp_entries}"$'\n'"- [charter] **Goal**: ${cw_goals} (Colony: ${cw_colony_name})"
        written=$((written + 1))
    fi

    if [[ -n "$cp_entries" ]]; then
        if ! _insert_section_entries "$tmp_file" "Codebase Patterns" "codebase patterns" "$cp_entries"; then
            cp_failed=1
            echo "[charter-write] WARNING: '## Codebase Patterns' section not found in QUEEN.md; entries dropped" >&2
        fi
    fi

    # Update Evolution Log
    local ev_type="charter_initialized"
    local ev_details="Colony charter created for ${cw_colony_name}"
    if [[ "$is_update" == "true" ]]; then
        ev_type="charter_updated"
        ev_details="Colony charter updated for ${cw_colony_name}"
    fi
    local ev_entry="| ${ts} | system | ${ev_type} | ${ev_details} |"
    local ev_separator
    ev_separator=$(grep -n "^|------|" "$tmp_file" | tail -1 | cut -d: -f1 || true)
    if [[ -n "$ev_separator" ]]; then
        awk -v line="$ev_separator" -v entry="$ev_entry" 'NR==line{print; print entry; next}1' "$tmp_file" > "${tmp_file}.ev" && mv "${tmp_file}.ev" "$tmp_file"
    fi

    # Update METADATA stats -- count non-charter list items in each section, add charter entries
    local up_section_line
    up_section_line=$(grep -n '^## User Preferences$' "$tmp_file" | head -1 | cut -d: -f1 || true)

    # Count charter entries written to User Preferences
    local up_charter_written=0
    [[ -n "$cw_intent" ]] && up_charter_written=$((up_charter_written + 1))
    [[ -n "$cw_vision" ]] && up_charter_written=$((up_charter_written + 1))

    local up_total=0
    if [[ -n "$up_section_line" ]]; then
        local up_next_section
        up_next_section=$(tail -n +$((up_section_line + 1)) "$tmp_file" | grep -n "^## " | head -1 | cut -d: -f1)
        local up_section_end
        if [[ -n "$up_next_section" ]]; then
            up_section_end=$((up_section_line + up_next_section - 1))
        else
            up_section_end=$(wc -l < "$tmp_file")
        fi

        # Count non-charter list items in User Preferences
        local up_non_charter
        # Count all list items, then subtract charter ones
        local up_all_items
        up_all_items=$(sed -n "${up_section_line},${up_section_end}p" "$tmp_file" | grep -c '^- ' || true)
        up_all_items=${up_all_items:-0}
        local up_charter_items
        up_charter_items=$(sed -n "${up_section_line},${up_section_end}p" "$tmp_file" | grep -c '^- \[charter\] ' || true)
        up_charter_items=${up_charter_items:-0}
        up_non_charter=$((up_all_items - up_charter_items))
        up_total=$((up_non_charter + up_charter_written))
    fi

    # Count for Codebase Patterns
    local cp_section_line
    cp_section_line=$(grep -n '^## Codebase Patterns$' "$tmp_file" | head -1 | cut -d: -f1 || true)

    # Count charter entries written to Codebase Patterns
    local cp_charter_written=0
    [[ -n "$cw_governance" ]] && cp_charter_written=$((cp_charter_written + 1))
    [[ -n "$cw_goals" ]] && cp_charter_written=$((cp_charter_written + 1))

    local cp_total=0
    if [[ -n "$cp_section_line" ]]; then
        local cp_next_section
        cp_next_section=$(tail -n +$((cp_section_line + 1)) "$tmp_file" | grep -n "^## " | head -1 | cut -d: -f1)
        local cp_section_end
        if [[ -n "$cp_next_section" ]]; then
            cp_section_end=$((cp_section_line + cp_next_section - 1))
        else
            cp_section_end=$(wc -l < "$tmp_file")
        fi
        local cp_all_items
        cp_all_items=$(sed -n "${cp_section_line},${cp_section_end}p" "$tmp_file" | grep -c '^- ' || true)
        cp_all_items=${cp_all_items:-0}
        local cp_charter_items
        cp_charter_items=$(sed -n "${cp_section_line},${cp_section_end}p" "$tmp_file" | grep -c '^- \[charter\] ' || true)
        cp_charter_items=${cp_charter_items:-0}
        local cp_non_charter=$((cp_all_items - cp_charter_items))
        cp_total=$((cp_non_charter + cp_charter_written))
    fi

    # Update METADATA stats
    awk -v up="$up_total" -v cp="$cp_total" '{
        gsub(/"total_user_prefs": [0-9]*/, "\"total_user_prefs\": " up)
        gsub(/"total_codebase_patterns": [0-9]*/, "\"total_codebase_patterns\": " cp)
        print
    }' "$tmp_file" > "${tmp_file}.stats" && mv "${tmp_file}.stats" "$tmp_file"

    # Update last_evolved
    awk -v ts="$ts" '/"last_evolved":/ { gsub(/"last_evolved": "[^"]*"/, "\"last_evolved\": \"" ts "\""); } {print}' "$tmp_file" > "${tmp_file}.meta" && mv "${tmp_file}.meta" "$tmp_file"

    # Safety guard: never overwrite QUEEN.md with empty content
    if [[ ! -s "$tmp_file" ]]; then
      rm -f "$tmp_file"
      json_err "$E_INTERNAL" "queen-write-charter produced empty output — aborting to protect QUEEN.md"
      return 1
    fi

    # Atomic move
    mv "$tmp_file" "$queen_file"

    local sections_failed=$((up_failed + cp_failed))
    if [[ "$written" -eq 0 && "$sections_failed" -gt 0 ]]; then
        json_err "$E_VALIDATION_FAILED" "charter-write: no entries written; all target sections missing from QUEEN.md" \
            "{\"written\":${written},\"sections_failed\":${sections_failed},\"colony_name\":\"${cw_colony_name}\",\"timestamp\":\"${ts}\"}"
        return 1
    fi
    json_ok "$(jq -n --argjson written "$written" --argjson updated "$is_update" \
      --arg colony_name "$cw_colony_name" --arg ts "$ts" --argjson sections_failed "$sections_failed" \
      '{written: $written, updated: $updated, colony_name: $colony_name, timestamp: $ts, sections_failed: $sections_failed}')"
}
