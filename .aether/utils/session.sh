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

    echo "$(jq -n --argjson ok "$pass" --arg command "$command_name" \
      --argjson fresh "$fresh_json" --argjson stale "$stale_json" \
      --argjson missing "$missing_json" --argjson total_lines "$total_lines" \
      '{ok: $ok, command: $command, fresh: $fresh, stale: $stale, missing: $missing, total_lines: $total_lines}')"
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

    local dry_run_bool=$([[ "$dry_run" == "--dry-run" ]] && echo "true" || echo "false")
    json_ok "$(jq -n --arg command "$command_name" --arg cleared "${cleared// /}" \
      --arg errors "${errors// /}" --argjson dry_run "$dry_run_bool" \
      '{command: $command, cleared: $cleared, errors: $errors, dry_run: $dry_run}')"
}

# ============================================================================
# _rotate_spawn_tree (helper -- used only by _session_init)
# ARCH-03: Rotate spawn-tree.txt at session start to prevent unbounded growth.
# Archives previous session's tree to a timestamped file; caps archive count at 5.
# ============================================================================
_rotate_spawn_tree() {
    local tree_file="$COLONY_DATA_DIR/spawn-tree.txt"
    [[ -f "$tree_file" ]] && [[ -s "$tree_file" ]] || return 0
    mkdir -p "$COLONY_DATA_DIR/spawn-tree-archive"
    local archive_ts
    archive_ts=$(date +%Y%m%d_%H%M%S)
    if ! cp "$tree_file" "$COLONY_DATA_DIR/spawn-tree-archive/spawn-tree.${archive_ts}.txt" 2>/dev/null; then  # SUPPRESS:OK -- cleanup: backup copy is best-effort
      _aether_log_error "Could not archive spawn-tree before rotation"
    fi
    > "$tree_file"  # Truncate in-place — preserves file handle for tail -f watchers
    # Keep only 5 archives
    # SUPPRESS:OK -- read-default: directory may not exist
    # SUPPRESS:OK -- cleanup: rotation cleanup is best-effort
    ls -t "$COLONY_DATA_DIR/spawn-tree-archive"/spawn-tree.*.txt 2>/dev/null \
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

    local session_file="$COLONY_DATA_DIR/session.json"
    local baseline
    baseline=$(git rev-parse HEAD 2>/dev/null || echo "")  # SUPPRESS:OK -- read-default: may not have commits yet

    jq -n --arg sid "$session_id" --arg started "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
      --arg goal "$goal" --arg baseline "$baseline" \
      '{
        session_id: $sid,
        started_at: $started,
        last_command: null,
        last_command_at: null,
        colony_goal: $goal,
        current_phase: 0,
        current_milestone: "First Mound",
        suggested_next: "/ant:plan",
        context_cleared: false,
        baseline_commit: $baseline,
        resumed_at: null,
        active_todos: [],
        summary: "Session initialized"
      }' > "$session_file.tmp"
    mv "$session_file.tmp" "$session_file"

    # Mirror to legacy path so session recovery (which checks .aether/data/session.json) works
    local legacy_session="$DATA_DIR/session.json"
    if [[ "$session_file" != "$legacy_session" ]]; then
      cp "$session_file" "$legacy_session" 2>/dev/null || true
    fi

    json_ok "$(jq -n --arg sid "$session_id" --arg goal "$goal" --arg file "$session_file" \
      '{session_id: $sid, goal: $goal, file: $file}')"
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

    local session_file="$COLONY_DATA_DIR/session.json"

    if [[ ! -f "$session_file" ]]; then
      # Auto-initialize if doesn't exist
      bash "$SCRIPT_DIR/aether-utils.sh" session-init "auto_$(date +%s)" ""
    fi

    # Read current session
    local current_session
    current_session=$(cat "$session_file" 2>/dev/null || echo '{}')  # SUPPRESS:OK -- read-default: file may not exist yet

    # Extract current values for preservation
    local current_goal current_phase current_milestone
    current_goal=$(sanitize_read_value "$(echo "$current_session" | jq -r '.colony_goal // empty')")
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
      current_goal=$(sanitize_read_value "$(jq -r '.goal // empty' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null || echo "$current_goal")")
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

    # Mirror to legacy path so session recovery (which checks .aether/data/session.json) works
    local legacy_session="$DATA_DIR/session.json"
    if [[ "$session_file" != "$legacy_session" ]]; then
      cp "$session_file" "$legacy_session" 2>/dev/null || true
    fi

    json_ok "$(jq -n --arg cmd "$cmd_run" '{updated: true, command: $cmd}')"
}

# ============================================================================
# _session_read
# Read and return current session state
# ============================================================================
_session_read() {
    local session_file="$COLONY_DATA_DIR/session.json"

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

    json_ok "$(jq -n --argjson is_stale "$is_stale" --argjson age "$age_hours" \
      --argjson session "$session_data" \
      '{exists: true, is_stale: $is_stale, age_hours: $age, session: $session}')"
}

# ============================================================================
# _session_is_stale
# Check if session is stale (returns JSON with is_stale boolean)
# ============================================================================
_session_is_stale() {
    _deprecation_warning "session-is-stale"
    local session_file="$COLONY_DATA_DIR/session.json"

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
    local session_file="$COLONY_DATA_DIR/session.json"

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
    local session_file="$COLONY_DATA_DIR/session.json"

    if [[ -f "$session_file" ]]; then
      jq --arg ts "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
         '.resumed_at = $ts | .context_cleared = false' "$session_file" > "$session_file.tmp" || {
        _aether_log_error "Could not process session resume update"
        rm -f "$session_file.tmp"
      }
      if [[ -s "$session_file.tmp" ]]; then
        mv "$session_file.tmp" "$session_file" || _aether_log_error "Could not finalize session resume"
      fi
      json_ok "$(jq -n --arg ts "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" '{resumed: true, timestamp: $ts}')"
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
    local session_file="$COLONY_DATA_DIR/session.json"
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
    goal=$(sanitize_read_value "$(jq -r '.colony_goal // "No goal set"' "$session_file")")
    phase=$(jq -r '.current_phase // 0' "$session_file")
    milestone=$(jq -r '.current_milestone // "First Mound"' "$session_file")
    last_cmd=$(jq -r '.last_command // "None"' "$session_file")
    last_at=$(jq -r '.last_command_at // "Unknown"' "$session_file")
    suggested=$(jq -r '.suggested_next // "None"' "$session_file")
    cleared=$(jq -r '.context_cleared // false' "$session_file")

    if [[ "$json_mode" == "true" ]]; then
      json_ok "$(jq -n --arg goal "$goal" --argjson phase "$phase" \
        --arg milestone "$milestone" --arg last_cmd "$last_cmd" \
        --arg last_at "$last_at" --arg suggested "$suggested" \
        --argjson cleared "$cleared" \
        '{exists: true, goal: $goal, phase: $phase, milestone: $milestone, last_command: $last_cmd, last_active: $last_at, suggested_next: $suggested, context_cleared: $cleared}')"
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

# ============================================================================
# _pending_decision_add
# Add a decision to the pending decisions queue
# Usage: pending-decision-add --type <type> --description <desc> [--phase N] [--source <src>]
# Types: visual_checkpoint, replan, escalation, runtime_verification, user_input
# ============================================================================
_pending_decision_add() {
    local pd_type=""
    local pd_description=""
    local pd_phase="null"
    local pd_source=""

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --type) pd_type="$2"; shift 2 ;;
        --description) pd_description="$2"; shift 2 ;;
        --phase) pd_phase="$2"; shift 2 ;;
        --source) pd_source="$2"; shift 2 ;;
        *) shift ;;
      esac
    done

    [[ -z "$pd_type" ]] && json_err "$E_VALIDATION_FAILED" "pending-decision-add requires --type"
    [[ -z "$pd_description" ]] && json_err "$E_VALIDATION_FAILED" "pending-decision-add requires --description"

    local pd_file="$COLONY_DATA_DIR/pending-decisions.json"
    local pd_id="pd_$(date +%s)_$$"
    local pd_now
    pd_now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Acquire lock for concurrent access
    if type acquire_lock &>/dev/null; then
      acquire_lock "$pd_file" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on pending-decisions.json"
      trap 'release_lock 2>/dev/null || true' EXIT  # SUPPRESS:OK -- cleanup: lock may not be held
    fi

    # Initialize file if missing
    if [[ ! -f "$pd_file" ]]; then
      echo '{"version":"1.0","decisions":[]}' > "$pd_file"
    fi

    local pd_current
    pd_current=$(cat "$pd_file" 2>/dev/null || echo '{"version":"1.0","decisions":[]}')  # SUPPRESS:OK -- read-default: file may not exist yet

    # Build new decision entry
    local pd_phase_val
    if [[ "$pd_phase" == "null" ]]; then
      pd_phase_val="null"
    else
      pd_phase_val="$pd_phase"
    fi

    local pd_updated
    pd_updated=$(echo "$pd_current" | jq \
      --arg id "$pd_id" \
      --arg type "$pd_type" \
      --arg description "$pd_description" \
      --argjson phase "${pd_phase_val}" \
      --arg source "$pd_source" \
      --arg created_at "$pd_now" \
      '.decisions += [{
        id: $id,
        type: $type,
        description: $description,
        phase: $phase,
        source: $source,
        created_at: $created_at,
        resolved: false
      }]' 2>/dev/null) || {
      type release_lock &>/dev/null && release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
      json_err "$E_JSON_INVALID" "Failed to append decision to pending-decisions.json"
    }

    atomic_write "$pd_file" "$pd_updated" || {
      type release_lock &>/dev/null && release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
      json_err "$E_JSON_INVALID" "Failed to write pending-decisions.json"
    }

    type release_lock &>/dev/null && { release_lock 2>/dev/null || true; trap - EXIT; }  # SUPPRESS:OK -- cleanup: lock may not be held

    local pd_count
    pd_count=$(echo "$pd_updated" | jq '.decisions | length')

    json_ok "$(jq -n --arg id "$pd_id" --argjson count "$pd_count" \
      '{id: $id, decision_count: $count}')"
}

# ============================================================================
# _pending_decision_list
# List decisions from the pending decisions queue
# Usage: pending-decision-list [--unresolved] [--type <type>]
# Default: show only unresolved
# ============================================================================
_pending_decision_list() {
    local pd_unresolved_only="true"
    local pd_filter_type=""

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --unresolved) pd_unresolved_only="true"; shift ;;
        --type) pd_filter_type="$2"; shift 2 ;;
        *) shift ;;
      esac
    done

    local pd_file="$COLONY_DATA_DIR/pending-decisions.json"

    if [[ ! -f "$pd_file" ]]; then
      json_ok '{"total":0,"unresolved":0,"decisions":[]}'
      exit 0
    fi

    local pd_data
    pd_data=$(cat "$pd_file" 2>/dev/null || echo '{"version":"1.0","decisions":[]}')  # SUPPRESS:OK -- read-default: file may not exist yet

    # Build jq filter
    local pd_filter='.decisions'

    # Apply type filter if provided
    if [[ -n "$pd_filter_type" ]]; then
      pd_filter="$pd_filter | map(select(.type == \"$pd_filter_type\"))"
    fi

    # Apply resolved filter (default: only unresolved)
    if [[ "$pd_unresolved_only" == "true" ]]; then
      pd_filter="$pd_filter | map(select(.resolved == false))"
    fi

    local pd_total pd_unresolved pd_decisions
    pd_total=$(echo "$pd_data" | jq '.decisions | length')
    pd_unresolved=$(echo "$pd_data" | jq '[.decisions[] | select(.resolved == false)] | length')
    pd_decisions=$(echo "$pd_data" | jq "$pd_filter")

    json_ok "$(jq -n --argjson total "$pd_total" --argjson unresolved "$pd_unresolved" \
      --argjson decisions "$pd_decisions" \
      '{total: $total, unresolved: $unresolved, decisions: $decisions}')"
}

# ============================================================================
# _pending_decision_resolve
# Mark a pending decision as resolved
# Usage: pending-decision-resolve --id <id> --resolution <text>
# ============================================================================
_pending_decision_resolve() {
    local pd_id=""
    local pd_resolution=""

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --id) pd_id="$2"; shift 2 ;;
        --resolution) pd_resolution="$2"; shift 2 ;;
        *) shift ;;
      esac
    done

    [[ -z "$pd_id" ]] && json_err "$E_VALIDATION_FAILED" "pending-decision-resolve requires --id"
    [[ -z "$pd_resolution" ]] && json_err "$E_VALIDATION_FAILED" "pending-decision-resolve requires --resolution"

    local pd_file="$COLONY_DATA_DIR/pending-decisions.json"

    if [[ ! -f "$pd_file" ]]; then
      json_err "$E_RESOURCE_NOT_FOUND" "No pending decisions file found"
    fi

    # Acquire lock for concurrent access
    if type acquire_lock &>/dev/null; then
      acquire_lock "$pd_file" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on pending-decisions.json"
      trap 'release_lock 2>/dev/null || true' EXIT  # SUPPRESS:OK -- cleanup: lock may not be held
    fi

    local pd_data
    pd_data=$(cat "$pd_file" 2>/dev/null || echo '{"version":"1.0","decisions":[]}')  # SUPPRESS:OK -- read-default: file may not exist yet

    # Check if ID exists
    local pd_exists
    pd_exists=$(echo "$pd_data" | jq --arg id "$pd_id" '[.decisions[] | select(.id == $id)] | length')
    if [[ "$pd_exists" -eq 0 ]]; then
      type release_lock &>/dev/null && release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
      json_err "$E_RESOURCE_NOT_FOUND" "Decision not found: $pd_id"
    fi

    local pd_now
    pd_now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    local pd_updated
    pd_updated=$(echo "$pd_data" | jq \
      --arg id "$pd_id" \
      --arg resolution "$pd_resolution" \
      --arg resolved_at "$pd_now" \
      '(.decisions[] | select(.id == $id)) |= (. + {resolved: true, resolution: $resolution, resolved_at: $resolved_at})' 2>/dev/null) || {
      type release_lock &>/dev/null && release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
      json_err "$E_JSON_INVALID" "Failed to resolve decision in pending-decisions.json"
    }

    atomic_write "$pd_file" "$pd_updated" || {
      type release_lock &>/dev/null && release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
      json_err "$E_JSON_INVALID" "Failed to write pending-decisions.json"
    }

    type release_lock &>/dev/null && { release_lock 2>/dev/null || true; trap - EXIT; }  # SUPPRESS:OK -- cleanup: lock may not be held

    json_ok "$(jq -n --arg id "$pd_id" '{resolved: true, id: $id}')"
}

# ============================================================================
# _autopilot_headless_check
# Check whether headless mode is active in run-state.json
# Usage: autopilot-headless-check
# Returns: {"ok":true,"result":{"headless":true|false}}
# ============================================================================
_autopilot_headless_check() {
    local ah_state_file="$COLONY_DATA_DIR/run-state.json"

    if [[ ! -f "$ah_state_file" ]]; then
      json_ok '{"headless":false}'
      exit 0
    fi

    local ah_headless
    ah_headless=$(jq -r '.headless // false' "$ah_state_file" 2>/dev/null || echo "false")  # SUPPRESS:OK -- read-default: field may not exist

    # Normalize to boolean
    if [[ "$ah_headless" == "true" ]]; then
      json_ok '{"headless":true}'
    else
      json_ok '{"headless":false}'
    fi
}

# ============================================================================
# _autopilot_set_headless
# Set the headless flag in run-state.json
# Usage: autopilot-set-headless <true|false>
# Returns: {"ok":true,"result":{"headless":true|false,"updated":true}}
# ============================================================================
_autopilot_set_headless() {
    local ah_value="${1:-}"

    if [[ "$ah_value" != "true" && "$ah_value" != "false" ]]; then
      json_err "$E_VALIDATION_FAILED" "autopilot-set-headless requires true or false argument"
    fi

    local ah_state_file="$COLONY_DATA_DIR/run-state.json"

    if [[ ! -f "$ah_state_file" ]]; then
      json_err "$E_FILE_NOT_FOUND" "run-state.json not found — autopilot not active"
    fi

    local ah_headless_bool
    [[ "$ah_value" == "true" ]] && ah_headless_bool=true || ah_headless_bool=false

    local ah_current ah_updated
    ah_current=$(cat "$ah_state_file" 2>/dev/null || echo '{}')  # SUPPRESS:OK -- read-default: file may not exist yet
    ah_updated=$(echo "$ah_current" | jq --argjson headless "$ah_headless_bool" '.headless = $headless' 2>/dev/null) || {
      json_err "$E_JSON_INVALID" "Failed to update headless flag in run-state.json"
    }

    atomic_write "$ah_state_file" "$ah_updated" || {
      json_err "$E_JSON_INVALID" "Failed to write run-state.json"
    }

    json_ok "$(jq -n --argjson headless "$ah_headless_bool" '{headless: $headless, updated: true}')"
}
