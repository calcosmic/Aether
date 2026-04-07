# =============================================================================
# DEPRECATED — This script has been superseded by the Go binary (aether CLI).
# All functionality is now available via: aether <subcommand>
# Do NOT modify this file — it is retained for reference only.
# See: cmd/ (Go source) | Run: aether --help
# =============================================================================
#
#!/usr/bin/env bash
# Suggest utility functions -- extracted from aether-utils.sh
# Provides: _suggest_analyze, _suggest_record, _suggest_check,
#           _suggest_clear, _suggest_approve, _suggest_quick_dismiss
# Also includes: get_type_emoji (helper used only by _suggest_approve)
# Note: suggest-clear is deprecated (moved with domain per user decision)

# ============================================================================
# get_type_emoji
# Helper function for suggest-approve display (bash 3.2 compatible)
# Usage: get_type_emoji <TYPE>
# Returns: emoji string for the given pheromone type
# ============================================================================
get_type_emoji() {
  case "$1" in
    FOCUS) echo "🎯" ;;
    REDIRECT) echo "🚫" ;;
    FEEDBACK) echo "💬" ;;
    *) echo "📝" ;;
  esac
}

# ============================================================================
# _suggest_analyze
# Analyze codebase and return pheromone suggestions based on code patterns
# Usage: _suggest_analyze [--source-dir DIR] [--max-suggestions N] [--dry-run]
# Returns: JSON with suggestions array and analysis metadata
# ============================================================================
_suggest_analyze() {
    # Disable ERR trap for this command (grep returns 1 on no match, which triggers trap)
    trap '' ERR

    source_dir=""
    max_suggestions=5
    dry_run=false

    # Parse arguments - note: $1 is already shifted by the main dispatch
    # So $1 here is the first argument after 'suggest-analyze'
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --source-dir) source_dir="$2"; shift 2 ;;
        --max-suggestions) max_suggestions="$2"; shift 2 ;;
        --dry-run) dry_run=true; shift ;;
        *) shift ;;
      esac
    done

    # Auto-detect source directory if not provided
    if [[ -z "$source_dir" ]]; then
      if [[ -d "$AETHER_ROOT/src" ]]; then
        source_dir="$AETHER_ROOT/src"
      elif [[ -d "$AETHER_ROOT/lib" ]]; then
        source_dir="$AETHER_ROOT/lib"
      else
        source_dir="$AETHER_ROOT"
      fi
    fi

    # Validate source directory
    if [[ ! -d "$source_dir" ]]; then
      json_err "$E_FILE_NOT_FOUND" "Source directory not found: $source_dir"
    fi

    # Build JSON array of suggestions using jq
    # We use jq to handle deduplication since bash 3.2 doesn't support associative arrays
    pheromones_file="$COLONY_DATA_DIR/pheromones.json"
    session_file="$COLONY_DATA_DIR/session.json"

    # Create temp file for collecting raw suggestions
    raw_suggestions=$(mktemp)
    echo "[]" > "$raw_suggestions"

    analyzed_count=0
    patterns_found=0

    # Define exclusions (use word boundaries to avoid matching partial paths)
    exclude_pattern="node_modules/|/.aether/|/dist/|/build/|/\\.git/|/coverage/|\\.min\\.js"

    # Find files to analyze (respecting exclusions)
    while IFS= read -r file || [[ -n "$file" ]]; do
      analyzed_count=$((analyzed_count + 1))

      # Skip excluded paths
      if echo "$file" | grep -qE "$exclude_pattern"; then
        continue
      fi

      # Get file extension
      ext="${file##*.}"

      # Check file size (large files > 300 lines)
      line_count=$(wc -l < "$file" 2>/dev/null || echo "0")  # SUPPRESS:OK -- read-default: file may not exist
      if [[ $line_count -gt 300 ]]; then
        patterns_found=$((patterns_found + 1))
        content="Large file: consider refactoring ($line_count lines)"
        reason="File exceeds 300 lines, consider breaking into smaller modules"
        hash=$(echo -n "$file:FOCUS:$content" | shasum -a 256 2>/dev/null | cut -d' ' -f1) || {  # SUPPRESS:OK -- read-default: hash generation with fallback
          _aether_log_error "Could not generate content hash -- using timestamp fallback"
          hash="$(date +%s%N)"
        }

        # Append suggestion to raw_suggestions using jq
        new_suggestion=$(jq -n --arg type "FOCUS" --arg content "$content" --arg file "$file" --arg reason "$reason" --arg hash "$hash" --arg priority "7" '{type: $type, content: $content, file: $file, reason: $reason, hash: $hash, priority: ($priority | tonumber)}')
        jq --argjson suggestion "$new_suggestion" '. += [$suggestion]' "$raw_suggestions" > "${raw_suggestions}.tmp" && mv "${raw_suggestions}.tmp" "$raw_suggestions"
      fi

      # Check for TODO/FIXME/XXX comments
      if [[ "$ext" =~ ^(ts|tsx|js|jsx|py|sh|md)$ ]]; then
        todo_matches=$( (grep -n "TODO\\|FIXME\\|XXX" "$file" 2>/dev/null || true) | wc -l | tr -d ' \n')  # SUPPRESS:OK -- read-default: file may not exist
        if [[ $todo_matches -gt 0 ]]; then
          patterns_found=$((patterns_found + 1))
          content="$todo_matches pending TODO/FIXME comments"
          reason="Unresolved markers indicate technical debt"
          # SUPPRESS:OK -- read-default: hash generation with fallback
          hash=$(echo -n "$file:FEEDBACK:$content" | shasum -a 256 2>/dev/null | cut -d' ' -f1) || {
            _aether_log_error "Could not generate content hash -- using timestamp fallback"
            hash="$(date +%s%N)"
          }

          new_suggestion=$(jq -n --arg type "FEEDBACK" --arg content "$content" --arg file "$file" --arg reason "$reason" --arg hash "$hash" --arg priority "4" '{type: $type, content: $content, file: $file, reason: $reason, hash: $hash, priority: ($priority | tonumber)}')
          jq --argjson suggestion "$new_suggestion" '. += [$suggestion]' "$raw_suggestions" > "${raw_suggestions}.tmp" && mv "${raw_suggestions}.tmp" "$raw_suggestions"
        fi
      fi

      # Check for debug artifacts (console.log, debugger)
      if [[ "$ext" =~ ^(ts|tsx|js|jsx)$ ]]; then
        # SUPPRESS:OK -- read-default: returns fallback on failure
        debug_matches=$( (grep -n "console\\.log\\|debugger" "$file" 2>/dev/null || true) | wc -l | tr -d ' \n')
        if [[ $debug_matches -gt 0 ]]; then
          patterns_found=$((patterns_found + 1))
          content="Remove debug artifacts before commit ($debug_matches found)"
          reason="Debug statements should not be committed to production code"
          # SUPPRESS:OK -- read-default: hash generation with fallback
          hash=$(echo -n "$file:REDIRECT:$content" | shasum -a 256 2>/dev/null | cut -d' ' -f1) || {
            _aether_log_error "Could not generate content hash -- using timestamp fallback"
            hash="$(date +%s%N)"
          }

          new_suggestion=$(jq -n --arg type "REDIRECT" --arg content "$content" --arg file "$file" --arg reason "$reason" --arg hash "$hash" --arg priority "9" '{type: $type, content: $content, file: $file, reason: $reason, hash: $hash, priority: ($priority | tonumber)}')
          jq --argjson suggestion "$new_suggestion" '. += [$suggestion]' "$raw_suggestions" > "${raw_suggestions}.tmp" && mv "${raw_suggestions}.tmp" "$raw_suggestions"
        fi
      fi

      # Check for type safety gaps (: any, : unknown)
      if [[ "$ext" =~ ^(ts|tsx)$ ]]; then
        type_gaps=$( (grep -n ": any\\|: unknown" "$file" 2>/dev/null || true) | wc -l | tr -d ' \n')  # SUPPRESS:OK -- read-default: file may not exist
        if [[ $type_gaps -gt 0 ]]; then
          patterns_found=$((patterns_found + 1))
          content="Type safety gaps detected ($type_gaps instances)"
          reason="Using 'any' or 'unknown' bypasses TypeScript's type checking"
          # SUPPRESS:OK -- read-default: hash generation with fallback
          hash=$(echo -n "$file:FEEDBACK:$content" | shasum -a 256 2>/dev/null | cut -d' ' -f1) || {
            _aether_log_error "Could not generate content hash -- using timestamp fallback"
            hash="$(date +%s%N)"
          }

          new_suggestion=$(jq -n --arg type "FEEDBACK" --arg content "$content" --arg file "$file" --arg reason "$reason" --arg hash "$hash" --arg priority "5" '{type: $type, content: $content, file: $file, reason: $reason, hash: $hash, priority: ($priority | tonumber)}')
          jq --argjson suggestion "$new_suggestion" '. += [$suggestion]' "$raw_suggestions" > "${raw_suggestions}.tmp" && mv "${raw_suggestions}.tmp" "$raw_suggestions"
        fi
      fi

      # Check for high complexity (function count)
      if [[ "$ext" =~ ^(ts|tsx|js|jsx|py|sh)$ ]]; then
        # SUPPRESS:OK -- existence-test: grep returns 1 when no matches
        func_count=$(grep -cE "^function|^def |^const.*=.*function|^const.*=.*=>" "$file" 2>/dev/null | tr -d ' \n' || echo "0")
        if [[ $func_count -gt 20 ]]; then
          patterns_found=$((patterns_found + 1))
          content="Complex module: test carefully ($func_count functions)"
          reason="High function count may indicate multiple concerns; verify test coverage"
          hash=$(echo -n "$file:FOCUS:$content" | shasum -a 256 2>/dev/null | cut -d' ' -f1) || {  # SUPPRESS:OK -- read-default: hash generation with fallback
            _aether_log_error "Could not generate content hash -- using timestamp fallback"
            hash="$(date +%s%N)"
          }

          new_suggestion=$(jq -n --arg type "FOCUS" --arg content "$content" --arg file "$file" --arg reason "$reason" --arg hash "$hash" --arg priority "6" '{type: $type, content: $content, file: $file, reason: $reason, hash: $hash, priority: ($priority | tonumber)}')
          jq --argjson suggestion "$new_suggestion" '. += [$suggestion]' "$raw_suggestions" > "${raw_suggestions}.tmp" && mv "${raw_suggestions}.tmp" "$raw_suggestions"
        fi
      fi

      # Check for test coverage gaps
      if [[ "$ext" =~ ^(ts|tsx|js|jsx|py)$ ]] && [[ ! "$file" =~ \\.test\\. ]] && [[ ! "$file" =~ \\.spec\\. ]]; then
        base_name=$(basename "$file" ".${ext}")
        dir_name=$(dirname "$file")

        # Look for corresponding test file
        if [[ -f "$dir_name/$base_name.test.$ext" ]] || [[ -f "$dir_name/$base_name.spec.$ext" ]] || \
           [[ -f "$dir_name/__tests__/$base_name.test.$ext" ]] || [[ -f "$dir_name/../tests/$base_name.test.$ext" ]]; then
          : # Test file exists
        else
          # Only suggest for files with functions (not config/pure data files)
          # SUPPRESS:OK -- existence-test: grep returns 1 when no matches
          if grep -qE "^function|^def |^const.*=.*function|^const.*=.*=>|^export.*function|^class " "$file" 2>/dev/null || false; then
            patterns_found=$((patterns_found + 1))
            content="Add tests for uncovered module: $base_name"
            reason="No corresponding test file found for module with functions"
            # SUPPRESS:OK -- read-default: hash generation with fallback
            hash=$(echo -n "$file:FOCUS:$content" | shasum -a 256 2>/dev/null | cut -d' ' -f1) || {
              _aether_log_error "Could not generate content hash -- using timestamp fallback"
              hash="$(date +%s%N)"
            }

            new_suggestion=$(jq -n --arg type "FOCUS" --arg content "$content" --arg file "$file" --arg reason "$reason" --arg hash "$hash" --arg priority "5" '{type: $type, content: $content, file: $file, reason: $reason, hash: $hash, priority: ($priority | tonumber)}')
            jq --argjson suggestion "$new_suggestion" '. += [$suggestion]' "$raw_suggestions" > "${raw_suggestions}.tmp" && mv "${raw_suggestions}.tmp" "$raw_suggestions"
          fi
        fi
      fi

    # SUPPRESS:OK -- existence-test: directory may not exist
    done < <(find "$source_dir" -type f \( -name "*.ts" -o -name "*.tsx" -o -name "*.js" -o -name "*.jsx" -o -name "*.py" -o -name "*.sh" -o -name "*.md" \) 2>/dev/null | head -100)

    # Deduplicate against existing pheromones and session suggestions using jq
    # Get existing signal content hashes
    existing_hashes="[]"
    if [[ -f "$pheromones_file" ]]; then
      # SUPPRESS:OK -- read-default: query may return empty
      existing_hashes=$(jq -r '[.signals[] | select(.active == true) | .content.text] | @json' "$pheromones_file" 2>/dev/null || echo "[]")
    fi

    session_hashes="[]"
    if [[ -f "$session_file" ]]; then
      # SUPPRESS:OK -- read-default: query may return empty
      session_hashes=$(jq -r '[.suggested_pheromones[]?.hash // empty] | @json' "$session_file" 2>/dev/null || echo "[]")
    fi

    # Filter suggestions: remove duplicates and sort by priority
    suggestions_json=$(jq --argjson existing "$existing_hashes" --argjson session "$session_hashes" --argjson max "$max_suggestions" '
      # Remove suggestions whose content matches existing signals
      map(select(.content as $c | $existing | index($c) | not)) |
      # Remove suggestions whose hash is in session
      map(select(.hash as $h | $session | index($h) | not)) |
      # Sort by priority descending and limit
      sort_by(.priority) | reverse | .[:$max]
    ' "$raw_suggestions" 2>/dev/null || echo "[]")  # SUPPRESS:OK -- read-default: file may not exist yet

    # Clean up temp file
    rm -f "$raw_suggestions"

    # Build result
    result=$(jq -n \
      --argjson suggestions "$suggestions_json" \
      --argjson analyzed "$analyzed_count" \
      --argjson patterns "$patterns_found" \
      '{suggestions: $suggestions, analyzed_files: $analyzed, patterns_found: $patterns}')

    if [[ "$dry_run" == "true" ]]; then
      echo "Dry run - analyzed: $source_dir" >&2
    fi

    # Re-enable ERR trap before exiting
    trap 'if type error_handler &>/dev/null; then error_handler ${LINENO} "$BASH_COMMAND" $?; fi' ERR

    json_ok "$result"
}

# ============================================================================
# _suggest_record
# Record a suggested pheromone hash to session.json for deduplication
# Usage: _suggest_record <hash> <type>
# Returns: JSON success/failure
# ============================================================================
_suggest_record() {
    record_hash="${1:-}"
    record_type="${2:-FEEDBACK}"

    if [[ -z "$record_hash" ]]; then
      json_err "$E_VALIDATION_FAILED" "suggest-record requires <hash> argument"
    fi

    session_file="$COLONY_DATA_DIR/session.json"

    # Initialize suggested_pheromones array if missing
    if [[ -f "$session_file" ]]; then
      # Check if suggested_pheromones field exists
      has_field=$(jq 'has("suggested_pheromones")' "$session_file" 2>/dev/null || echo "false")  # SUPPRESS:OK -- read-default: file may not exist yet
      if [[ "$has_field" != "true" ]]; then
        # Add the field
        jq '. + {"suggested_pheromones": []}' "$session_file" > "${session_file}.tmp" || {
          _aether_log_error "Could not add suggestions field to session file"
          rm -f "${session_file}.tmp"
        }
        if [[ -s "${session_file}.tmp" ]]; then
          mv "${session_file}.tmp" "$session_file" || _aether_log_error "Could not finalize session field addition"
        fi
      fi

      # Append new suggestion
      record_entry=$(jq -n --arg hash "$record_hash" --arg type "$record_type" --arg suggested_at "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" '{hash: $hash, type: $type, suggested_at: $suggested_at}')
      jq --argjson entry "$record_entry" '.suggested_pheromones += [$entry]' "$session_file" > "${session_file}.tmp" || {
        _aether_log_error "Could not record suggestion in session file"
        rm -f "${session_file}.tmp"
      }
      if [[ -s "${session_file}.tmp" ]]; then
        mv "${session_file}.tmp" "$session_file" || _aether_log_error "Could not finalize suggestion recording"
      fi
    else
      # Create session.json with suggested_pheromones
      record_entry=$(jq -n --arg hash "$record_hash" --arg type "$record_type" --arg suggested_at "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" '{hash: $hash, type: $type, suggested_at: $suggested_at}')
      local sr_init_content
      sr_init_content=$(jq -n --argjson entry "$record_entry" '{suggested_pheromones: [$entry]}')
      atomic_write "$session_file" "$sr_init_content" || _aether_log_error "Could not create session file with suggestion"
    fi

    json_ok '{"recorded":true}'
}

# ============================================================================
# _suggest_check
# Check if a hash was already suggested this session
# Usage: _suggest_check <hash>
# Returns: JSON {already_suggested: true/false}
# ============================================================================
_suggest_check() {
    check_hash="${1:-}"

    if [[ -z "$check_hash" ]]; then
      json_err "$E_VALIDATION_FAILED" "suggest-check requires <hash> argument"
    fi

    session_file="$COLONY_DATA_DIR/session.json"
    already_suggested="false"

    if [[ -f "$session_file" ]]; then
      # SUPPRESS:OK -- read-default: query may return empty
      count=$(jq --arg hash "$check_hash" '[.suggested_pheromones[]? | select(.hash == $hash)] | length' "$session_file" 2>/dev/null || echo "0")
      if [[ "$count" -gt 0 ]]; then
        already_suggested="true"
      fi
    fi

    json_ok "$(jq -n --argjson already "$already_suggested" '{already_suggested: $already}')"
}

# ============================================================================
# _suggest_clear
# Clear the suggested_pheromones array from session.json
# Usage: _suggest_clear
# Returns: JSON success with count cleared
# NOTE: This subcommand is deprecated
# ============================================================================
_suggest_clear() {
    _deprecation_warning "suggest-clear"
    session_file="$COLONY_DATA_DIR/session.json"
    cleared_count=0

    if [[ -f "$session_file" ]]; then
      cleared_count=$(jq '.suggested_pheromones | length' "$session_file" 2>/dev/null || echo "0")  # SUPPRESS:OK -- read-default: file may not exist yet
      jq 'del(.suggested_pheromones)' "$session_file" > "${session_file}.tmp" || {
        _aether_log_error "Could not clear suggestions from session file"
        rm -f "${session_file}.tmp"
      }
      if [[ -s "${session_file}.tmp" ]]; then
        mv "${session_file}.tmp" "$session_file" || _aether_log_error "Could not finalize suggestion clearing"
      fi
    fi

    json_ok "$(jq -n --argjson cleared "$cleared_count" '{cleared: $cleared}')"
}

# ============================================================================
# _suggest_approve
# Orchestrate pheromone suggestion approval workflow
# Usage: _suggest_approve [--verbose] [--dry-run] [--yes] [--no-suggest]
# Returns: JSON summary {approved, rejected, skipped, signals_created}
# ============================================================================
_suggest_approve() {
    verbose=false
    dry_run=false
    skip_confirm=false
    no_suggest=false

    # Parse arguments
    for arg in "$@"; do
      case "$arg" in
        --verbose) verbose=true ;;
        --dry-run) dry_run=true ;;
        --yes) skip_confirm=true ;;
        --no-suggest) no_suggest=true ;;
      esac
    done

    # Handle --no-suggest flag - exit immediately
    if [[ "$no_suggest" == "true" ]]; then
      json_ok '{"approved":0,"rejected":0,"skipped":0,"signals_created":[],"reason":"--no-suggest flag"}'
      exit 0
    fi

    # Check for non-interactive mode (no tty)
    if [[ ! -t 0 ]] && [[ "$skip_confirm" != "true" ]]; then
      echo "Non-interactive mode: skipping suggestions (use --yes to auto-approve)" >&2
      json_ok '{"approved":0,"rejected":0,"skipped":0,"signals_created":[],"reason":"non-interactive mode"}'
      exit 0
    fi

    # Get suggestions from suggest-analyze
    suggestions_result=$(bash "$0" suggest-analyze 2>/dev/null || echo '{"suggestions":[]}')  # SUPPRESS:OK -- read-default: subcommand may fail
    suggestions_json=$(echo "$suggestions_result" | jq '.result.suggestions // []')

    # Check if there are any suggestions
    suggestion_count=$(echo "$suggestions_json" | jq 'length')
    if [[ "$suggestion_count" -eq 0 ]]; then
      # Exit silently when no suggestions
      json_ok '{"approved":0,"rejected":0,"skipped":0,"signals_created":[]}'
      exit 0
    fi

    # Arrays to track results
    approved_suggestions=()
    rejected_suggestions=()
    skipped_suggestions=()
    signals_created=()

    # Display header (to stderr so stdout is valid JSON)
    echo "" >&2
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" >&2
    echo "   S U G G E S T E D   P H E R O M O N E S" >&2
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" >&2
    echo "" >&2
    echo "Based on code analysis, the colony suggests these signals:" >&2
    echo "" >&2

    # Process suggestions one at a time
    for ((i=0; i<suggestion_count; i++)); do
      suggestion=$(echo "$suggestions_json" | jq ".[$i]")
      stype=$(echo "$suggestion" | jq -r '.type')
      content=$(echo "$suggestion" | jq -r '.content')
      file=$(echo "$suggestion" | jq -r '.file')
      reason=$(echo "$suggestion" | jq -r '.reason')
      priority=$(echo "$suggestion" | jq -r '.priority // 5')
      hash=$(echo "$suggestion" | jq -r '.hash')

      emoji=$(get_type_emoji "$stype")

      # Display suggestion (to stderr so stdout is valid JSON)
      echo "───────────────────────────────────────────────────" >&2
      echo "Suggestion $((i+1)) of $suggestion_count" >&2
      echo "───────────────────────────────────────────────────" >&2
      echo "" >&2
      echo "$emoji $stype (priority: $priority/10)" >&2
      echo "" >&2
      echo "$content" >&2
      echo "" >&2
      echo "Detected in: $file" >&2
      echo "Reason: $reason" >&2
      echo "" >&2
      echo "───────────────────────────────────────────────────" >&2

      # Handle dry-run mode
      if [[ "$dry_run" == "true" ]]; then
        echo "Dry run: would approve" >&2
        approved_suggestions+=("$suggestion")
        echo "" >&2
        continue
      fi

      # Handle --yes mode (auto-approve all)
      if [[ "$skip_confirm" == "true" ]]; then
        approved_suggestions+=("$suggestion")
        echo "✓ Auto-approved (--yes mode)" >&2
        echo "" >&2
        continue
      fi

      # Prompt for action (to stderr so stdout is valid JSON)
      echo -n "[A]pprove  [R]eject  [S]kip  [D]ismiss All  Your choice: " >&2
      read -r choice

      case "$choice" in
        [Aa]|"approve"|"Approve")
          approved_suggestions+=("$suggestion")
          echo "✓ Approved" >&2
          ;;
        [Rr]|"reject"|"Reject")
          rejected_suggestions+=("$suggestion")
          # Record hash to prevent re-suggestion
          bash "$0" suggest-record "$hash" "$stype" >/dev/null 2>&1 || _aether_log_error "Could not record suggestion"
          echo "✗ Rejected" >&2
          ;;
        [Dd]|"dismiss"|"Dismiss"|"dismiss all"|"Dismiss All")
          # Dismiss all remaining suggestions
          for ((j=i; j<suggestion_count; j++)); do
            remaining=$(echo "$suggestions_json" | jq ".[$j]")
            skipped_suggestions+=("$remaining")
          done
          echo "→ Dismissed all remaining suggestions" >&2
          break
          ;;
        [Ss]|""|"skip"|"Skip")
          skipped_suggestions+=("$suggestion")
          echo "→ Skipped" >&2
          ;;
        *)
          # Invalid input - default to skip
          skipped_suggestions+=("$suggestion")
          echo "→ Skipped (invalid input)" >&2
          ;;
      esac
      echo "" >&2
    done

    # Execute approvals for approved suggestions
    approved_count=0
    if [[ ${#approved_suggestions[@]} -gt 0 ]]; then
      echo "" >&2
      echo "Creating pheromone signals for ${#approved_suggestions[@]} approved suggestion(s)..." >&2
      echo "" >&2

      for suggestion in "${approved_suggestions[@]}"; do
        stype=$(echo "$suggestion" | jq -r '.type')
        content=$(echo "$suggestion" | jq -r '.content')
        reason=$(echo "$suggestion" | jq -r '.reason')
        hash=$(echo "$suggestion" | jq -r '.hash')

        if [[ "$dry_run" == "true" ]]; then
          echo "Dry run: would create $stype signal: \"$content\"" >&2
          ((approved_count++))
          signals_created+=("dry_run_sig_$approved_count")
          continue
        fi

        # Call pheromone-write to create the signal
        signal_result=$(bash "$0" pheromone-write "$stype" "$content" --source "system:suggestion" --reason "$reason" --ttl "phase_end" 2>&1)

        if echo "$signal_result" | jq -e '.ok' >/dev/null 2>&1; then  # SUPPRESS:OK -- validation: testing JSON field
          signal_id=$(echo "$signal_result" | jq -r '.result.signal_id // "unknown"')
          signals_created+=("$signal_id")
          echo "✓ Added $stype signal" >&2

          # Record hash to prevent duplicates
          bash "$0" suggest-record "$hash" "$stype" >/dev/null 2>&1 || _aether_log_error "Could not record suggestion"
          ((approved_count++))
        else
          echo "✗ Failed to create signal: $content" >&2
          echo "  Error: $(echo "$signal_result" | jq -r '.error.message // "Unknown error"')" >&2
        fi
      done
    fi

    # Record rejected suggestions (already recorded during loop, but ensure consistency)
    rejected_count=${#rejected_suggestions[@]}

    # Skipped suggestions (not recorded, may suggest again)
    skipped_count=${#skipped_suggestions[@]}

    # Display summary (to stderr so stdout is valid JSON)
    echo "" >&2
    echo "═══════════════════════════════════════════════════" >&2
    echo "Summary: $approved_count approved, $rejected_count rejected, $skipped_count skipped" >&2
    echo "═══════════════════════════════════════════════════" >&2
    echo "" >&2

    # Build result with signals_created as JSON array (handle empty array case)
    if [[ ${#signals_created[@]} -gt 0 ]]; then
      signals_json=$(printf '%s\n' "${signals_created[@]}" | jq -R . | jq -s .)
    else
      signals_json="[]"
    fi
    result=$(jq -n \
      --argjson approved "$approved_count" \
      --argjson rejected "$rejected_count" \
      --argjson skipped "$skipped_count" \
      --argjson signals "$signals_json" \
      '{approved: $approved, rejected: $rejected, skipped: $skipped, signals_created: $signals}')

    json_ok "$result"
}

# ============================================================================
# _suggest_quick_dismiss
# Quick dismiss all current suggestions - records hashes to prevent re-suggestion
# Usage: _suggest_quick_dismiss
# Returns: JSON {dismissed, hashes_recorded}
# ============================================================================
_suggest_quick_dismiss() {
    # Get current suggestions
    suggestions_result=$(bash "$0" suggest-analyze 2>/dev/null || echo '{"suggestions":[]}')  # SUPPRESS:OK -- read-default: subcommand may fail
    suggestions_json=$(echo "$suggestions_result" | jq '.result.suggestions // []')

    dismissed_count=0
    hashes_recorded=()

    suggestion_count=$(echo "$suggestions_json" | jq 'length')

    if [[ "$suggestion_count" -gt 0 ]]; then
      for ((i=0; i<suggestion_count; i++)); do
        suggestion=$(echo "$suggestions_json" | jq ".[$i]")
        hash=$(echo "$suggestion" | jq -r '.hash')
        stype=$(echo "$suggestion" | jq -r '.type')

        # Record hash to prevent re-suggestion
        bash "$0" suggest-record "$hash" "$stype" >/dev/null 2>&1 || _aether_log_error "Could not record suggestion"
        hashes_recorded+=("$hash")
        ((dismissed_count++))
      done
    fi

    # Output message to stderr so stdout is valid JSON only
    echo "Suggestions dismissed. Run with --yes to auto-approve in future." >&2

    # Build result with hashes as JSON array (handle empty array case)
    if [[ ${#hashes_recorded[@]} -gt 0 ]]; then
      hashes_json=$(printf '%s\n' "${hashes_recorded[@]}" | jq -R . | jq -s .)
    else
      hashes_json="[]"
    fi
    result=$(jq -n \
      --argjson dismissed "$dismissed_count" \
      --argjson hashes "$hashes_json" \
      '{dismissed: $dismissed, hashes_recorded: $hashes}')

    json_ok "$result"
}
