#!/usr/bin/env bash
# Session utility functions -- extracted from aether-utils.sh
# Provides: _session_verify_fresh, _session_clear, _session_init, _session_update,
#           _session_read, _session_is_stale, _session_clear_context,
#           _session_mark_resumed, _session_summary
# Also includes: _rotate_spawn_tree (helper used only by _session_init)

# ============================================================================
# _session_verify_fresh
# Generic session freshness verification
# Usage: _session_verify_fresh [args...] (same args as before: --command <name> [--force] <session_start_unixtime>)
# Returns: JSON with pass/fail status and file details
# ============================================================================
_session_verify_fresh() {
    # Parse arguments
    local command_name=""
    local force_mode=""
    local session_start_time=""

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --command) command_name="$2"; shift 2 ;;
        --force) force_mode="--force"; shift ;;
        *) session_start_time="$1"; shift ;;
      esac
    done

    # Validate command name
    [[ -z "$command_name" ]] && json_err "$E_VALIDATION_FAILED" "Usage: session-verify-fresh --command <name> [--force] <session_start>"

    # Map command to directory and files (using env var override pattern)
    local session_dir required_docs
    case "$command_name" in
      survey)
        session_dir="${SURVEY_DIR:-.aether/data/survey}"
        required_docs="PROVISIONS.md TRAILS.md BLUEPRINT.md CHAMBERS.md DISCIPLINES.md SENTINEL-PROTOCOLS.md PATHOGENS.md"
        ;;
      oracle)
        session_dir="${ORACLE_DIR:-.aether/oracle}"
        required_docs="state.json plan.json gaps.md synthesis.md research-plan.md"
        ;;
      watch)
        session_dir="${WATCH_DIR:-.aether/data}"
        required_docs="watch-status.txt watch-progress.txt"
        ;;
      swarm)
        session_dir="${SWARM_DIR:-.aether/data/swarm}"
        required_docs="findings.json"
        ;;
      init)
        session_dir="${INIT_DIR:-.aether/data}"
        required_docs="COLONY_STATE.json constraints.json"
        ;;
      seal|entomb)
        session_dir="${ARCHIVE_DIR:-.aether/data/archive}"
        required_docs="manifest.json"
        ;;
      *)
        json_err "$E_VALIDATION_FAILED" "Unknown command: $command_name" '{"commands":["survey","oracle","watch","swarm","init","seal","entomb"]}'
        ;;
    esac

    # Initialize result arrays
    local fresh_docs=""
    local stale_docs=""
    local missing_docs=""
    local total_lines=0

    for doc in $required_docs; do
      local doc_path="$session_dir/$doc"

      if [[ ! -f "$doc_path" ]]; then
        missing_docs="${missing_docs:+$missing_docs }$doc"
        continue
      fi

      # Get line count
      local lines
      lines=$(wc -l < "$doc_path" 2>/dev/null | tr -d ' ' || echo "0")  # SUPPRESS:OK -- read-default: file may not exist
      total_lines=$((total_lines + lines))

      # In force mode, accept any existing file
      if [[ "$force_mode" == "--force" ]]; then
        fresh_docs="${fresh_docs:+$fresh_docs }$doc"
        continue
      fi

      # Check timestamp if session_start_time provided
      if [[ -n "$session_start_time" ]]; then
        # Cross-platform stat: macOS uses -f %m, Linux uses -c %Y
        local file_mtime
        file_mtime=$(stat -f %m "$doc_path" 2>/dev/null || stat -c %Y "$doc_path" 2>/dev/null || echo "0")  # SUPPRESS:OK -- cross-platform: macOS stat syntax

        if [[ "$file_mtime" -ge "$session_start_time" ]]; then
          fresh_docs="${fresh_docs:+$fresh_docs }$doc"
        else
          stale_docs="${stale_docs:+$stale_docs }$doc"
        fi
      else
        # No start time provided - accept existing file (backward compatible)
        fresh_docs="${fresh_docs:+$fresh_docs }$doc"
      fi
    done

    # Determine pass/fail
    # pass = true if: no stale files (fresh files can coexist with missing files)
    # missing files are ok - they will be created during the session
    local pass=false
    if [[ "$force_mode" == "--force" ]] || [[ -z "$stale_docs" ]]; then
      pass=true
    fi

    # Build JSON response
    local fresh_json=""
    for item in $fresh_docs; do fresh_json="$fresh_json\"$item\","; done
    fresh_json="[${fresh_json%,}]"

    local stale_json=""
    for item in $stale_docs; do stale_json="$stale_json\"$item\","; done
    stale_json="[${stale_json%,}]"

    local missing_json=""
    for item in $missing_docs; do missing_json="$missing_json\"$item\","; done
    missing_json="[${missing_json%,}]"

    echo "{\"ok\":$pass,\"command\":\"$command_name\",\"fresh\":$fresh_json,\"stale\":$stale_json,\"missing\":$missing_json,\"total_lines\":$total_lines}"
    exit 0
}

# ============================================================================
# _session_clear
# Generic session file clearing
# Usage: _session_clear [args...] (same args as before: --command <name> [--dry-run])
# ============================================================================
_session_clear() {
    # Parse arguments
    local command_name=""
    local dry_run=""

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --command) command_name="$2"; shift 2 ;;
        --dry-run) dry_run="--dry-run"; shift ;;
        *) shift ;;
      esac
    done

    [[ -z "$command_name" ]] && json_err "$E_VALIDATION_FAILED" "Usage: session-clear --command <name> [--dry-run]"

    # Map command to directory and files
    local session_dir="" files="" subdir_files=""
    case "$command_name" in
      survey)
        session_dir="${SURVEY_DIR:-.aether/data/survey}"
        files="PROVISIONS.md TRAILS.md BLUEPRINT.md CHAMBERS.md DISCIPLINES.md SENTINEL-PROTOCOLS.md PATHOGENS.md"
        ;;
      oracle)
        session_dir="${ORACLE_DIR:-.aether/oracle}"
        files="state.json plan.json gaps.md synthesis.md research-plan.md .stop .last-topic"
        # Also clear discoveries subdirectory
        subdir_files="discoveries/*"
        ;;
      watch)
        session_dir="${WATCH_DIR:-.aether/data}"
        files="watch-status.txt watch-progress.txt"
        ;;
      swarm)
        session_dir="${SWARM_DIR:-.aether/data/swarm}"
        files="findings.json display.json timing.json"
        ;;
      init)
        # Init clear is destructive - blocked for auto-clear
        json_err "$E_VALIDATION_FAILED" "Command 'init' is protected and cannot be auto-cleared. Use manual removal of COLONY_STATE.json if absolutely necessary."
        ;;
      seal|entomb)
        # Archive operations should never be auto-cleared
        json_err "$E_VALIDATION_FAILED" "Command '$command_name' is protected and cannot be auto-cleared. Archives and chambers must be managed manually."
        ;;
      *)
        json_err "$E_VALIDATION_FAILED" "Unknown command: $command_name"
        ;;
    esac

    local cleared=""
    local errors=""

    if [[ -d "$session_dir" && -n "$files" ]]; then
      for doc in $files; do
        local doc_path="$session_dir/$doc"
        if [[ -f "$doc_path" ]]; then
          if [[ "$dry_run" == "--dry-run" ]]; then
            cleared="$cleared $doc"
          else
            if rm -f "$doc_path" 2>/dev/null; then  # SUPPRESS:OK -- cleanup: file may not exist
              cleared="$cleared $doc"
            else
              errors="$errors $doc"
            fi
          fi
        fi
      done

      # Handle oracle discoveries subdirectory
      if [[ "$command_name" == "oracle" && -d "$session_dir/discoveries" ]]; then
        if [[ "$dry_run" == "--dry-run" ]]; then
          cleared="$cleared discoveries/"
        else
          # SUPPRESS:OK -- cleanup: file may not exist
          rm -rf "$session_dir/discoveries" 2>/dev/null && cleared="$cleared discoveries/" || errors="$errors discoveries/"
        fi
      fi
    fi

    json_ok "{\"command\":\"$command_name\",\"cleared\":\"${cleared// /}\",\"errors\":\"${errors// /}\",\"dry_run\":$([[ "$dry_run" == "--dry-run" ]] && echo "true" || echo "false")}"
}

# ============================================================================
# _rotate_spawn_tree (helper -- used only by _session_init)
# ARCH-03: Rotate spawn-tree.txt at session start to prevent unbounded growth.
# Archives previous session's tree to a timestamped file; caps archive count at 5.
# ============================================================================
_rotate_spawn_tree() {
    local tree_file="$DATA_DIR/spawn-tree.txt"
    [[ -f "$tree_file" ]] && [[ -s "$tree_file" ]] || return 0
    mkdir -p "$DATA_DIR/spawn-tree-archive"
    local archive_ts
    archive_ts=$(date +%Y%m%d_%H%M%S)
    if ! cp "$tree_file" "$DATA_DIR/spawn-tree-archive/spawn-tree.${archive_ts}.txt" 2>/dev/null; then  # SUPPRESS:OK -- cleanup: backup copy is best-effort
      _aether_log_error "Could not archive spawn-tree before rotation"
    fi
    > "$tree_file"  # Truncate in-place — preserves file handle for tail -f watchers
    # Keep only 5 archives
    # SUPPRESS:OK -- read-default: directory may not exist
    # SUPPRESS:OK -- cleanup: rotation cleanup is best-effort
    ls -t "$DATA_DIR/spawn-tree-archive"/spawn-tree.*.txt 2>/dev/null \
        | tail -n +6 | while IFS= read -r file; do rm -f "$file"; done 2>/dev/null || true  # SUPPRESS:OK -- cleanup: file may not exist
}

# ============================================================================
# _session_init
# Initialize a new session tracking file
# Usage: _session_init [session_id] [goal]
# ============================================================================
_session_init() {
    local session_id="${1:-$(date +%s)_$(openssl rand -hex 4 2>/dev/null || echo $$)}"  # SUPPRESS:OK -- read-default: openssl may not be available
    local goal="${2:-}"

    _rotate_spawn_tree

    local session_file="$DATA_DIR/session.json"
    local baseline
    baseline=$(git rev-parse HEAD 2>/dev/null || echo "")  # SUPPRESS:OK -- read-default: may not have commits yet

    cat > "$session_file.tmp" << EOF
{
  "session_id": "$session_id",
  "started_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "last_command": null,
  "last_command_at": null,
  "colony_goal": "$goal",
  "current_phase": 0,
  "current_milestone": "First Mound",
  "suggested_next": "/ant:plan",
  "context_cleared": false,
  "baseline_commit": "$baseline",
  "resumed_at": null,
  "active_todos": [],
  "summary": "Session initialized"
}
EOF
    mv "$session_file.tmp" "$session_file"
    json_ok "{\"session_id\":\"$session_id\",\"goal\":\"$goal\",\"file\":\"$session_file\"}"
}

# ============================================================================
# _session_update
# Update session with latest activity
# Usage: _session_update <command> [suggested_next] [summary]
# ============================================================================
_session_update() {
    local cmd_run="${1:-}"
    local suggested="${2:-}"
    local summary="${3:-}"

    local session_file="$DATA_DIR/session.json"

    if [[ ! -f "$session_file" ]]; then
      # Auto-initialize if doesn't exist
      bash "$SCRIPT_DIR/aether-utils.sh" session-init "auto_$(date +%s)" ""
    fi

    # Read current session
    local current_session
    current_session=$(cat "$session_file" 2>/dev/null || echo '{}')  # SUPPRESS:OK -- read-default: file may not exist yet

    # Extract current values for preservation
    local current_goal current_phase current_milestone
    current_goal=$(echo "$current_session" | jq -r '.colony_goal // empty')
    current_phase=$(echo "$current_session" | jq -r '.current_phase // 0')
    current_milestone=$(echo "$current_session" | jq -r '.current_milestone // "First Mound"')

    # Get top 3 TODOs if TO-DOs.md exists
    local todos="[]"
    if [[ -f "TO-DOs.md" ]]; then
      todos=$(grep "^### " TO-DOs.md 2>/dev/null | head -3 | sed 's/^### //' | jq -R . | jq -s .)  # SUPPRESS:OK -- existence-test: file may not exist
    fi

    # MIGRATE: direct COLONY_STATE.json access -- use _state_read_field instead
    # Get colony state if exists
    if [[ -f "$DATA_DIR/COLONY_STATE.json" ]]; then
      # SUPPRESS:OK -- read-default: query may return empty
      current_goal=$(jq -r '.goal // empty' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null || echo "$current_goal")
      # SUPPRESS:OK -- read-default: query may return empty
      current_phase=$(jq -r '.current_phase // 0' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null || echo "$current_phase")
      # SUPPRESS:OK -- read-default: query may return empty
      current_milestone=$(jq -r '.milestone // "First Mound"' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null || echo "$current_milestone")
    fi

    # Capture current git HEAD for drift detection
    local baseline
    baseline=$(git rev-parse HEAD 2>/dev/null || echo "")  # SUPPRESS:OK -- read-default: may not have commits yet

    # Build updated session
    echo "$current_session" | jq --arg cmd "$cmd_run" \
      --arg ts "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
      --arg suggested "$suggested" \
      --arg summary "$summary" \
      --arg goal "$current_goal" \
      --argjson phase "$current_phase" \
      --arg milestone "$current_milestone" \
      --argjson todos "$todos" \
      --arg baseline "$baseline" \
      '.last_command = $cmd |
       .last_command_at = $ts |
       .suggested_next = $suggested |
       .summary = $summary |
       .colony_goal = $goal |
       .current_phase = $phase |
       .current_milestone = $milestone |
       .active_todos = $todos |
       .baseline_commit = $baseline' > "$session_file.tmp" || {
      _aether_log_error "Could not process session update"
      rm -f "$session_file.tmp"
      json_err "$E_UNKNOWN" "Failed to update session file"
    }
    [[ -s "$session_file.tmp" ]] || {
      _aether_log_error "Session update produced empty result -- not overwriting"
      rm -f "$session_file.tmp"
      json_err "$E_JSON_INVALID" "Session update produced empty result"
    }
    mv "$session_file.tmp" "$session_file" || {
      _aether_log_error "Could not finalize session file update"
      rm -f "$session_file.tmp"
      json_err "$E_UNKNOWN" "Failed to rename temporary session file"
    }

    json_ok "{\"updated\":true,\"command\":\"$cmd_run\"}"
}

# ============================================================================
# _session_read
# Read and return current session state
# ============================================================================
_session_read() {
    local session_file="$DATA_DIR/session.json"

    if [[ ! -f "$session_file" ]]; then
      json_ok "{\"exists\":false,\"session\":null}"
      exit 0
    fi

    local session_data
    session_data=$(cat "$session_file" 2>/dev/null || echo '{}')  # SUPPRESS:OK -- read-default: file may not exist yet

    # Check if stale (> 24 hours)
    local last_cmd_ts="" is_stale="" age_hours=""
    last_cmd_ts=$(echo "$session_data" | jq -r '.last_command_at // .started_at // empty')
    if [[ -n "$last_cmd_ts" ]]; then
      local last_epoch=0 now_epoch=0
      # SUPPRESS:OK -- cross-platform: macOS date syntax
      # SUPPRESS:OK -- cross-platform: macOS vs Linux date/stat flags
      last_epoch=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "$last_cmd_ts" +%s 2>/dev/null \
        || date -d "$last_cmd_ts" +%s 2>/dev/null \
        || echo 0)
      now_epoch=$(date +%s)
      age_hours=$(( (now_epoch - last_epoch) / 3600 ))
      [[ $age_hours -gt 24 ]] && is_stale=true || is_stale=false
    else
      is_stale="false"
      age_hours="unknown"
    fi

    json_ok "{\"exists\":true,\"is_stale\":$is_stale,\"age_hours\":$age_hours,\"session\":$session_data}"
}

# ============================================================================
# _session_is_stale
# Check if session is stale (returns JSON with is_stale boolean)
# ============================================================================
_session_is_stale() {
    _deprecation_warning "session-is-stale"
    local session_file="$DATA_DIR/session.json"

    if [[ ! -f "$session_file" ]]; then
      json_ok '{"is_stale":true}'
      exit 0
    fi

    local last_cmd_ts
    last_cmd_ts=$(jq -r '.last_command_at // .started_at // empty' "$session_file" 2>/dev/null)  # SUPPRESS:OK -- read-default: file may not exist yet

    if [[ -z "$last_cmd_ts" ]]; then
      json_ok '{"is_stale":true}'
      exit 0
    fi

    # macOS uses -j -f, Linux uses -d
    # SUPPRESS:OK -- cross-platform: macOS date syntax
    # SUPPRESS:OK -- cross-platform: macOS vs Linux date/stat flags
    local last_epoch now_epoch age_hours
    last_epoch=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "$last_cmd_ts" +%s 2>/dev/null \
      || date -d "$last_cmd_ts" +%s 2>/dev/null \
      || echo 0)
    now_epoch=$(date +%s)
    age_hours=$(( (now_epoch - last_epoch) / 3600 ))

    if [[ $age_hours -gt 24 ]]; then
      json_ok '{"is_stale":true}'
    else
      json_ok '{"is_stale":false}'
    fi
}

# ============================================================================
# _session_clear_context
# Mark session context as cleared (preserves file but marks context_cleared)
# ============================================================================
_session_clear_context() {
    _deprecation_warning "session-clear-context"
    local preserve="${1:-false}"
    local session_file="$DATA_DIR/session.json"

    if [[ -f "$session_file" ]]; then
      if [[ "$preserve" == "true" ]]; then
        # Just mark as cleared
        jq '.context_cleared = true' "$session_file" > "$session_file.tmp" || {
          _aether_log_error "Could not mark session as cleared"
          rm -f "$session_file.tmp"
        }
        if [[ -s "$session_file.tmp" ]]; then
          mv "$session_file.tmp" "$session_file" || _aether_log_error "Could not finalize session clear"
        fi
        json_ok "{\"cleared\":true,\"preserved\":true}"
      else
        # Remove file entirely
        rm -f "$session_file"
        json_ok "{\"cleared\":true,\"preserved\":false}"
      fi
    else
      json_ok "{\"cleared\":false,\"reason\":\"no_session_exists\"}"
    fi
}

# ============================================================================
# _session_mark_resumed
# Mark session as resumed
# ============================================================================
_session_mark_resumed() {
    local session_file="$DATA_DIR/session.json"

    if [[ -f "$session_file" ]]; then
      jq --arg ts "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
         '.resumed_at = $ts | .context_cleared = false' "$session_file" > "$session_file.tmp" || {
        _aether_log_error "Could not process session resume update"
        rm -f "$session_file.tmp"
      }
      if [[ -s "$session_file.tmp" ]]; then
        mv "$session_file.tmp" "$session_file" || _aether_log_error "Could not finalize session resume"
      fi
      json_ok "{\"resumed\":true,\"timestamp\":\"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\"}"
    else
      json_err "$E_RESOURCE_NOT_FOUND" "No active session to mark as resumed. Try: run /ant:init to start a new session."
    fi
}

# ============================================================================
# _session_summary
# Get session summary (human-readable or JSON)
# ============================================================================
_session_summary() {
    _deprecation_warning "session-summary"
    local session_file="$DATA_DIR/session.json"
    local json_mode="false"

    # Parse --json flag (command name already shifted by main dispatch)
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --json)
          json_mode="true"
          shift
          ;;
        *)
          shift
          ;;
      esac
    done

    if [[ ! -f "$session_file" ]]; then
      if [[ "$json_mode" == "true" ]]; then
        json_ok '{"exists":false,"goal":null,"phase":0}'
      else
        echo "No active session found."
      fi
      exit 0
    fi

    local goal phase milestone last_cmd last_at suggested cleared
    goal=$(jq -r '.colony_goal // "No goal set"' "$session_file")
    phase=$(jq -r '.current_phase // 0' "$session_file")
    milestone=$(jq -r '.current_milestone // "First Mound"' "$session_file")
    last_cmd=$(jq -r '.last_command // "None"' "$session_file")
    last_at=$(jq -r '.last_command_at // "Unknown"' "$session_file")
    suggested=$(jq -r '.suggested_next // "None"' "$session_file")
    cleared=$(jq -r '.context_cleared // false' "$session_file")

    if [[ "$json_mode" == "true" ]]; then
      # Escape goal for JSON
      local goal_escaped milestone_escaped last_cmd_escaped last_at_escaped suggested_escaped
      goal_escaped=$(echo "$goal" | jq -Rs . | tr -d '\n')
      milestone_escaped=$(echo "$milestone" | jq -Rs . | tr -d '\n')
      last_cmd_escaped=$(echo "$last_cmd" | jq -Rs . | tr -d '\n')
      last_at_escaped=$(echo "$last_at" | jq -Rs . | tr -d '\n')
      suggested_escaped=$(echo "$suggested" | jq -Rs . | tr -d '\n')
      json_ok "{\"exists\":true,\"goal\":$goal_escaped,\"phase\":$phase,\"milestone\":$milestone_escaped,\"last_command\":$last_cmd_escaped,\"last_active\":$last_at_escaped,\"suggested_next\":$suggested_escaped,\"context_cleared\":$cleared}"
    else
      echo "Session Summary"
      echo "=================="
      echo "Goal: $goal"
      [[ "$phase" != "0" ]] && echo "Phase: $phase"
      echo "Milestone: $milestone"
      echo "Last Command: $last_cmd"
      echo "Last Active: $last_at"
      [[ "$suggested" != "None" ]] && echo "Suggested Next: $suggested"
      [[ "$cleared" == "true" ]] && echo "Status: Context was cleared"
    fi
}
