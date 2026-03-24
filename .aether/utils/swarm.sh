#!/usr/bin/env bash
# Swarm utility functions -- extracted from aether-utils.sh
# Provides: _autofix_checkpoint, _autofix_rollback, _swarm_findings_init, _swarm_findings_add,
#   _swarm_findings_read, _swarm_solution_set, _swarm_cleanup, _swarm_activity_log,
#   _swarm_display_init, _swarm_display_update, _swarm_display_get, _swarm_display_render,
#   _swarm_display_inline, _swarm_display_text, _swarm_timing_start, _swarm_timing_get,
#   _swarm_timing_eta
# Note: Uses get_caste_emoji() which remains in the main file (shared helper)

# ============================================================================
# _autofix_checkpoint
# Create checkpoint before applying auto-fix
# Usage: _autofix_checkpoint [label]
# Returns: {type: "stash"|"commit"|"none", ref: "..."}
# IMPORTANT: Only stash Aether-related files, never touch user work
# ============================================================================
_autofix_checkpoint() {
    if git rev-parse --git-dir >/dev/null 2>&1; then  # SUPPRESS:OK -- existence-test: may not be a git repo
      # Check if there are changes to Aether-managed files only
      # Target directories that Aether is allowed to modify
      target_dirs=".aether .claude/commands/ant .claude/commands/st .opencode bin"
      has_changes=false

      for dir in $target_dirs; do
        if [[ -d "$dir" ]] && [[ -n "$(git status --porcelain "$dir" 2>/dev/null)" ]]; then  # SUPPRESS:OK -- existence-test: may not be a git repo
          has_changes=true
          break
        fi
      done

      if [[ "$has_changes" == "true" ]]; then
        label="${1:-autofix-$(date +%s)}"
        stash_name="aether-checkpoint: $label"
        # Only stash Aether-managed directories, never touch user files
        if git stash push -m "$stash_name" -- $target_dirs >/dev/null 2>&1; then  # SUPPRESS:OK -- existence-test: stash operation may fail
          json_ok "{\"type\":\"stash\",\"ref\":\"$stash_name\"}"
        else
          # Stash failed (possibly due to conflicts), record commit hash
          hash=$(git rev-parse HEAD 2>/dev/null || echo "unknown")  # SUPPRESS:OK -- read-default: may not have commits yet
          json_ok "{\"type\":\"commit\",\"ref\":\"$hash\"}"
        fi
      else
        # No changes in Aether-managed directories, just record commit hash
        hash=$(git rev-parse HEAD 2>/dev/null || echo "unknown")  # SUPPRESS:OK -- read-default: may not have commits yet
        json_ok "{\"type\":\"commit\",\"ref\":\"$hash\"}"
      fi
    else
      json_ok '{"type":"none","ref":null}'
    fi
}

# ============================================================================
# _autofix_rollback
# Rollback from checkpoint if fix failed
# Usage: _autofix_rollback <type> <ref>
# Returns: {rolled_back: bool, method: "stash"|"reset"|"none"}
# ============================================================================
_autofix_rollback() {
    ref_type="${1:-none}"
    ref="${2:-}"

    case "$ref_type" in
      stash)
        # Find and pop the stash
        stash_ref=$(git stash list 2>/dev/null | grep "$ref" | head -1 | cut -d: -f1 || echo "")  # SUPPRESS:OK -- existence-test: stash operation may fail
        if [[ -n "$stash_ref" ]]; then
          if git stash pop "$stash_ref" >/dev/null 2>&1; then  # SUPPRESS:OK -- existence-test: stash operation may fail
            json_ok '{"rolled_back":true,"method":"stash"}'
          else
            json_ok '{"rolled_back":false,"method":"stash","error":"stash pop failed"}'
          fi
        else
          json_ok '{"rolled_back":false,"method":"stash","error":"stash not found"}'
        fi
        ;;
      commit)
        # Reset to the commit
        if [[ -n "$ref" && "$ref" != "unknown" ]]; then
          if git reset --hard "$ref" >/dev/null 2>&1; then  # SUPPRESS:OK -- existence-test: reset may fail
            json_ok '{"rolled_back":true,"method":"reset"}'
          else
            json_ok '{"rolled_back":false,"method":"reset","error":"reset failed"}'
          fi
        else
          json_ok '{"rolled_back":false,"method":"reset","error":"invalid ref"}'
        fi
        ;;
      none|*)
        json_ok '{"rolled_back":false,"method":"none"}'
        ;;
    esac
}

# ============================================================================
# _swarm_findings_init
# Initialize swarm findings file
# Usage: _swarm_findings_init <swarm_id>
# ============================================================================
_swarm_findings_init() {
    swarm_id="${1:-swarm-$(date +%s)}"
    findings_file="$DATA_DIR/swarm-findings-$swarm_id.json"

    mkdir -p "$DATA_DIR"
    cat > "$findings_file" <<EOF
{
  "swarm_id": "$swarm_id",
  "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "status": "active",
  "findings": [],
  "solution": null
}
EOF
    json_ok "{\"swarm_id\":\"$swarm_id\",\"file\":\"$findings_file\"}"
}

# ============================================================================
# _swarm_findings_add
# Add a finding from a scout
# Usage: _swarm_findings_add <swarm_id> <scout_type> <confidence> <finding_json>
# ============================================================================
_swarm_findings_add() {
    swarm_id="${1:-}"
    scout_type="${2:-}"
    confidence="${3:-0.5}"
    finding="${4:-}"

    [[ -z "$swarm_id" || -z "$scout_type" || -z "$finding" ]] && json_err "$E_VALIDATION_FAILED" "Usage: swarm-findings-add <swarm_id> <scout_type> <confidence> <finding_json>"

    findings_file="$DATA_DIR/swarm-findings-$swarm_id.json"
    [[ ! -f "$findings_file" ]] && json_err "$E_FILE_NOT_FOUND" "Swarm findings file not found: $swarm_id"

    ts=$(date -u +%Y-%m-%dT%H:%M:%SZ)

    # Add finding to array
    updated=$(jq --arg scout "$scout_type" --arg conf "$confidence" --arg ts "$ts" --argjson finding "$finding" '
      .findings += [{
        "scout": $scout,
        "confidence": ($conf | tonumber),
        "timestamp": $ts,
        "finding": $finding
      }]
    ' "$findings_file")

    atomic_write "$findings_file" "$updated" || {
      _aether_log_error "Could not save swarm findings"
      json_err "$E_UNKNOWN" "Failed to write swarm findings file"
    }
    count=$(echo "$updated" | jq '.findings | length')
    json_ok "{\"added\":true,\"scout\":\"$scout_type\",\"total_findings\":$count}"
}

# ============================================================================
# _swarm_findings_read
# Read all findings for a swarm
# Usage: _swarm_findings_read <swarm_id>
# ============================================================================
_swarm_findings_read() {
    swarm_id="${1:-}"
    [[ -z "$swarm_id" ]] && json_err "$E_VALIDATION_FAILED" "Usage: swarm-findings-read <swarm_id>"

    findings_file="$DATA_DIR/swarm-findings-$swarm_id.json"
    [[ ! -f "$findings_file" ]] && json_err "$E_FILE_NOT_FOUND" "Swarm findings file not found: $swarm_id"

    json_ok "$(cat "$findings_file")"
}

# ============================================================================
# _swarm_solution_set
# Set the chosen solution for a swarm
# Usage: _swarm_solution_set <swarm_id> <solution_json>
# ============================================================================
_swarm_solution_set() {
    swarm_id="${1:-}"
    solution="${2:-}"

    [[ -z "$swarm_id" || -z "$solution" ]] && json_err "$E_VALIDATION_FAILED" "Usage: swarm-solution-set <swarm_id> <solution_json>"

    findings_file="$DATA_DIR/swarm-findings-$swarm_id.json"
    [[ ! -f "$findings_file" ]] && json_err "$E_FILE_NOT_FOUND" "Swarm findings file not found: $swarm_id"

    ts=$(date -u +%Y-%m-%dT%H:%M:%SZ)

    updated=$(jq --argjson solution "$solution" --arg ts "$ts" '
      .solution = $solution |
      .status = "resolved" |
      .resolved_at = $ts
    ' "$findings_file")

    atomic_write "$findings_file" "$updated" || {
      _aether_log_error "Could not save swarm solution"
      json_err "$E_UNKNOWN" "Failed to write swarm findings file"
    }
    json_ok "{\"solution_set\":true,\"swarm_id\":\"$swarm_id\"}"
}

# ============================================================================
# _swarm_cleanup
# Clean up swarm files after completion
# Usage: _swarm_cleanup <swarm_id> [--archive]
# ============================================================================
_swarm_cleanup() {
    swarm_id="${1:-}"
    archive="${2:-}"

    [[ -z "$swarm_id" ]] && json_err "$E_VALIDATION_FAILED" "Usage: swarm-cleanup <swarm_id> [--archive]"

    findings_file="$DATA_DIR/swarm-findings-$swarm_id.json"

    if [[ -f "$findings_file" ]]; then
      if [[ "$archive" == "--archive" ]]; then
        mkdir -p "$DATA_DIR/swarm-archive"
        mv "$findings_file" "$DATA_DIR/swarm-archive/"
        json_ok "{\"archived\":true,\"swarm_id\":\"$swarm_id\"}"
      else
        rm -f "$findings_file"
        json_ok "{\"deleted\":true,\"swarm_id\":\"$swarm_id\"}"
      fi
    else
      json_ok "{\"not_found\":true,\"swarm_id\":\"$swarm_id\"}"
    fi
}

# ============================================================================
# _swarm_activity_log
# Log an activity entry for swarm visualization
# Usage: _swarm_activity_log <ant_name> <action> <details>
# ============================================================================
_swarm_activity_log() {
    ant_name="${1:-}"
    action="${2:-}"
    details="${3:-}"
    [[ -z "$ant_name" || -z "$action" || -z "$details" ]] && json_err "$E_VALIDATION_FAILED" "Usage: swarm-activity-log <ant_name> <action> <details>"

    mkdir -p "$DATA_DIR"
    log_file="$DATA_DIR/swarm-activity.log"
    ts=$(date -u +"%H:%M:%S")
    echo "[$ts] $ant_name: $action $details" >> "$log_file"
    json_ok '"logged"'
}

# ============================================================================
# _swarm_display_init
# Initialize swarm display state file
# Usage: _swarm_display_init <swarm_id>
# ============================================================================
_swarm_display_init() {
    swarm_id="${1:-swarm-$(date +%s)}"
    mkdir -p "$DATA_DIR"

    display_file="$DATA_DIR/swarm-display.json"
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    atomic_write "$display_file" "{
  \"swarm_id\": \"$swarm_id\",
  \"timestamp\": \"$ts\",
  \"active_ants\": [],
  \"summary\": { \"total_active\": 0, \"by_caste\": {}, \"by_zone\": {} },
  \"chambers\": {
    \"fungus_garden\": {\"activity\": 0, \"icon\": \"🍄\"},
    \"nursery\": {\"activity\": 0, \"icon\": \"🥚\"},
    \"refuse_pile\": {\"activity\": 0, \"icon\": \"🗑️\"},
    \"throne_room\": {\"activity\": 0, \"icon\": \"👑\"},
    \"foraging_trail\": {\"activity\": 0, \"icon\": \"🌿\"}
  }
}"
    json_ok "{\"swarm_id\":\"$swarm_id\",\"initialized\":true}"
}

# ============================================================================
# _swarm_display_update
# Update ant activity in swarm display
# Usage: _swarm_display_update <ant_name> <caste> <ant_status> <task> [parent] [tools_json] [tokens] [chamber] [progress]
# ============================================================================
_swarm_display_update() {
    ant_name="${1:-}"
    caste="${2:-}"
    ant_status="${3:-}"
    task="${4:-}"
    parent="${5:-}"
    tools_json="${6:-}"
    [[ -z "$tools_json" ]] && tools_json="{}"
    tokens="${7:-0}"
    chamber="${8:-}"
    progress="${9:-0}"

    [[ -z "$ant_name" || -z "$caste" || -z "$ant_status" ]] && json_err "$E_VALIDATION_FAILED" "Usage: swarm-display-update <ant_name> <caste> <ant_status> <task> [parent] [tools_json] [tokens] [chamber] [progress]"

    # Tolerate malformed argument ordering from LLM-generated commands.
    # Common failure mode: tools_json omitted, so tokens/chamber/progress shift left.
    tools_type=$(echo "$tools_json" | jq -r 'type' 2>/dev/null || echo "invalid")  # SUPPRESS:OK -- read-default: returns fallback if missing
    if [[ "$tools_type" != "object" ]]; then
      if [[ "$tools_json" =~ ^[0-9]+$ ]] && [[ ! "$tokens" =~ ^[0-9]+$ ]] && [[ "$chamber" =~ ^[0-9]+$ ]]; then
        progress="$chamber"
        chamber="$tokens"
        tokens="$tools_json"
      fi
      tools_json="{}"
    fi

    # Ensure numeric fields are always valid for --argjson.
    [[ "$tokens" =~ ^-?[0-9]+$ ]] || tokens=0
    [[ "$progress" =~ ^-?[0-9]+$ ]] || progress=0

    display_file="$DATA_DIR/swarm-display.json"

    # Initialize if doesn't exist
    if [[ ! -f "$display_file" ]]; then
      bash "$0" swarm-display-init "default-swarm" >/dev/null 2>&1 || _aether_log_error "Could not initialize swarm display"
    fi

    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Read current display and update using jq
    updated=$(jq --arg ant "$ant_name" --arg caste "$caste" --arg ant_status "$ant_status" \
      --arg task "$task" --arg parent "$parent" --argjson tools "$tools_json" \
      --argjson tokens "$tokens" --arg ts "$ts" --arg chamber "$chamber" --argjson progress "$progress" '
      # Find existing ant or create new entry
      (.active_ants | map(select(.name == $ant)) | length) as $exists |
      # Get old chamber if ant exists
      (if $exists > 0 then
        (.active_ants[] | select(.name == $ant) | .chamber // "")
      else
        ""
      end) as $old_chamber |
      # Determine new chamber
      (if $chamber != "" then $chamber else $old_chamber end) as $new_chamber |
      if $exists > 0 then
        # Update existing ant
        .active_ants = [.active_ants[] | if .name == $ant then
          . + {
            caste: $caste,
            status: $ant_status,
            task: $task,
            parent: (if $parent != "" then $parent else .parent end),
            tools: (if $tools != {} then $tools else .tools end),
            tokens: (.tokens + $tokens),
            chamber: (if $chamber != "" then $chamber else (.chamber // null) end),
            progress: (if $progress > 0 then $progress else (.progress // 0) end),
            updated_at: $ts
          }
        else . end]
      else
        # Add new ant
        .active_ants += [{
          name: $ant,
          caste: $caste,
          status: $ant_status,
          task: $task,
          parent: (if $parent != "" then $parent else null end),
          tools: (if $tools != {} then $tools else {read:0,grep:0,edit:0,bash:0} end),
          tokens: $tokens,
          chamber: (if $chamber != "" then $chamber else null end),
          progress: $progress,
          started_at: $ts,
          updated_at: $ts
        }]
      end |
      # Recalculate summary
      .summary.total_active = (.active_ants | length) |
      .summary.by_caste = (.active_ants | group_by(.caste) | map({key: .[0].caste, value: length}) | from_entries) |
      .summary.by_zone = (.active_ants | group_by(.status) | map({key: .[0].status, value: length}) | from_entries) |
      # Update chamber activity counts
      # Decrement old chamber if changed
      (if $old_chamber != "" and $old_chamber != $new_chamber and has("chambers") and (.chambers | has($old_chamber)) then
        .chambers[$old_chamber].activity = ([(.chambers[$old_chamber].activity // 1) - 1, 0] | max)
      else
        .
      end) |
      # Increment new chamber
      (if $new_chamber != "" and has("chambers") and (.chambers | has($new_chamber)) then
        .chambers[$new_chamber].activity = (.chambers[$new_chamber].activity // 0) + 1
      else
        .
      end)
    ' "$display_file") || json_err "$E_JSON_INVALID" "Failed to update swarm display"

    atomic_write "$display_file" "$updated"

    # Get emoji for response
    emoji=$(get_caste_emoji "$caste")
    json_ok "{\"updated\":true,\"ant\":\"$ant_name\",\"caste\":\"$caste\",\"emoji\":\"$emoji\",\"chamber\":\"$chamber\",\"progress\":$progress}"
}

# ============================================================================
# _swarm_display_get
# Get current swarm display state
# Usage: _swarm_display_get
# ============================================================================
_swarm_display_get() {
    display_file="$DATA_DIR/swarm-display.json"

    if [[ ! -f "$display_file" ]]; then
      json_ok '{"swarm_id":null,"active_ants":[],"summary":{"total_active":0,"by_caste":{},"by_zone":{}},"chambers":{}}'
    else
      json_ok "$(cat "$display_file")"
    fi
}

# ============================================================================
# _swarm_display_render
# Render the swarm display to terminal
# Usage: _swarm_display_render [swarm_id]
# ============================================================================
_swarm_display_render() {
    _deprecation_warning "swarm-display-render"
    swarm_id="${1:-default-swarm}"

    display_script="$SCRIPT_DIR/utils/swarm-display.sh"

    if [[ -f "$display_script" ]]; then
      # Execute the display script
      bash "$display_script" "$swarm_id" 2>/dev/null || _aether_log_error "Could not run swarm display script"
      json_ok '{"rendered":true}'
    else
      json_err "$E_FILE_NOT_FOUND" "Display script not found: $display_script"
    fi
}

# ============================================================================
# Display helper functions (used by _swarm_display_inline)
# These are local helpers that were defined inside the swarm-display-inline case block
# ============================================================================

# Caste colors (ANSI)
_sw_get_caste_color() {
    case "$1" in
      builder) echo "$_SW_BLUE" ;;
      watcher) echo "$_SW_GREEN" ;;
      scout) echo "$_SW_YELLOW" ;;
      chaos) echo "$_SW_RED" ;;
      prime) echo "$_SW_MAGENTA" ;;
      oracle) echo "$_SW_MAGENTA" ;;
      route_setter) echo "$_SW_MAGENTA" ;;
      *) echo "$_SW_RESET" ;;
    esac
}

# Caste emojis with ant (local copy -- may differ from main file's get_caste_emoji)
_sw_get_caste_emoji() {
    case "$1" in
      builder) echo "🔨🐜" ;;
      watcher) echo "👁️🐜" ;;
      scout) echo "🔍🐜" ;;
      chaos) echo "🎲🐜" ;;
      prime) echo "👑🐜" ;;
      oracle) echo "🔮🐜" ;;
      route_setter) echo "🧭🐜" ;;
      archaeologist) echo "🏺🐜" ;;
      chronicler) echo "📝🐜" ;;
      gatekeeper) echo "📦🐜" ;;
      guardian) echo "🛡️🐜" ;;
      includer) echo "♿🐜" ;;
      keeper) echo "📚🐜" ;;
      measurer) echo "⚡🐜" ;;
      probe) echo "🧪🐜" ;;
      sage) echo "📜🐜" ;;
      tracker) echo "🐛🐜" ;;
      weaver) echo "🔄🐜" ;;
      colonizer) echo "🌱🐜" ;;
      dreamer) echo "💭🐜" ;;
      *) echo "🐜" ;;
    esac
}

# Status phrases
_sw_get_status_phrase() {
    case "$1" in
      builder) echo "excavating..." ;;
      watcher) echo "observing..." ;;
      scout) echo "exploring..." ;;
      chaos) echo "testing..." ;;
      prime) echo "coordinating..." ;;
      oracle) echo "researching..." ;;
      route_setter) echo "planning..." ;;
      *) echo "working..." ;;
    esac
}

# Excavation phrase based on progress
_sw_get_excavation_phrase() {
    local progress="${1:-0}"
    if [[ "$progress" -lt 25 ]]; then
      echo "🚧 Starting excavation..."
    elif [[ "$progress" -lt 50 ]]; then
      echo "⛏️  Digging deeper..."
    elif [[ "$progress" -lt 75 ]]; then
      echo "🪨 Moving earth..."
    elif [[ "$progress" -lt 100 ]]; then
      echo "🏗️  Almost there..."
    else
      echo "✅ Excavation complete!"
    fi
}

# Format tools: "📖5 🔍3 ✏️2 ⚡1"
_sw_format_tools() {
    local read="${1:-0}"
    local grep="${2:-0}"
    local edit="${3:-0}"
    local bash="${4:-0}"
    local result=""
    [[ "$read" -gt 0 ]] && result="${result}📖${read} "
    [[ "$grep" -gt 0 ]] && result="${result}🔍${grep} "
    [[ "$edit" -gt 0 ]] && result="${result}✏️${edit} "
    [[ "$bash" -gt 0 ]] && result="${result}⚡${bash}"
    echo "$result"
}

# Render progress bar (green when working)
_sw_render_progress_bar() {
    local percent="${1:-0}"
    local width="${2:-20}"
    [[ "$percent" -lt 0 ]] && percent=0
    [[ "$percent" -gt 100 ]] && percent=100
    local filled=$((percent * width / 100))
    local empty=$((width - filled))
    local bar=""
    for ((i=0; i<filled; i++)); do bar+="█"; done
    for ((i=0; i<empty; i++)); do bar+="░"; done
    echo -e "${_SW_GREEN}[$bar]${_SW_RESET} ${percent}%"
}

# Format duration
_sw_format_duration() {
    local seconds="${1:-0}"
    if [[ "$seconds" -lt 60 ]]; then
      echo "${seconds}s"
    else
      local mins=$((seconds / 60))
      local secs=$((seconds % 60))
      echo "${mins}m${secs}s"
    fi
}

# ============================================================================
# _swarm_display_inline
# Inline swarm display for Claude Code (no loop, no clear)
# Usage: _swarm_display_inline [swarm_id]
# ============================================================================
_swarm_display_inline() {
    _deprecation_warning "swarm-display-inline"
    swarm_id="${1:-default-swarm}"
    display_file="$DATA_DIR/swarm-display.json"

    # ANSI colors
    _SW_BLUE='\033[34m'
    _SW_GREEN='\033[32m'
    _SW_YELLOW='\033[33m'
    _SW_RED='\033[31m'
    _SW_MAGENTA='\033[35m'
    _SW_BOLD='\033[1m'
    _SW_DIM='\033[2m'
    _SW_RESET='\033[0m'

    # Check for display file
    if [[ ! -f "$display_file" ]]; then
      echo -e "${_SW_DIM}🐜 No active swarm data${_SW_RESET}"
      json_ok '{"displayed":false,"reason":"no_data"}'
      exit 0
    fi

    # Check for jq
    if ! command -v jq >/dev/null 2>&1; then
      echo -e "${_SW_DIM}🐜 Swarm active (jq not available for details)${_SW_RESET}"
      json_ok '{"displayed":true,"warning":"jq_missing"}'
      exit 0
    fi

    # Read swarm data
    total_active=$(jq -r '.summary.total_active // 0' "$display_file" 2>/dev/null || echo "0")  # SUPPRESS:OK -- read-default: file may not exist yet

    if [[ "$total_active" -eq 0 ]]; then
      echo -e "${_SW_DIM}🐜 Colony idle${_SW_RESET}"
      json_ok '{"displayed":true,"ants":0}'
      exit 0
    fi

    # Render header with ant logo
    echo ""
    cat << 'ANTLOGO'


                                      ▁▐▖      ▁
                            ▗▇▇███▆▇▃▅████▆▇▆▅▟██▛▇
                             ▝▜▅▛██████████████▜▅██
                          ▁▂▀▇▆██▙▜██████████▛▟███▛▁▃▁
                         ▕▂▁▉▅████▙▞██████▜█▚▟████▅▊ ▐
                        ▗▁▐█▀▜████▛▃▝▁████▍▘▟▜████▛▀█▂ ▖
                    ▁▎▝█▁▝▍▆▜████▊▐▀▏▀▍▂▂▝▀▕▀▌█████▀▅▐▚ █▏▁▁
                      ▂▚▃▇▙█▟████▛▏ ▝▜▐▛▀▍▛▘ ▕█████▆▊▐▂▃▞▂▔
                       ▚▔█▛██████▙▟▍▜▍▜▃▃▖▟▛▐██████▛▛▜▔▔▞
                        ▋▖▍▊▖██████▇▃▁▝██▘▝▃████▜█▜ ▋▐▐▗
                        ▍▌▇█▅▂▜██████████████████▉▃▄▋▖  ▝
                      ▁▎▍▁▜▟███▀▀▜████████████▛▀▀███▆▂  ▁▁
                     ██ ▆▇▌▁▕▚▅▆███▛████████▜███▆▄▞▁▁▐▅▎ █▉
                     ▆█████▛▃▟█▀████████████████▛█▙▙▜▉▟▛▜█▌▗
                     ▅▆▋ ▁▁▁▔▕▁▁▁▇█████▛▀▀▀▁▜▇▇▁▁▁▁▁▁▁▁ ▐▊▗
                   ▗▆▃▃▃▔███▖▔██▀▀▝▀██▀▍█▛▁▐█▏█▛▀▀▏█▛▀▜█▆▃▃▆▖
                   ▝▗▖  ▟█▟█▙ █▛▀▘  █▊ ▕█▛▀▜█▏█▛▀▘ █▋▆█▛  ▗▖
                   ▘ ▘ ▟▛  ▝▀▘▀▀▀▀▘ ▀▀▂▂█▙▂▐▀▏▀▀▀▀▘▀▘ ▝▀▅▂▝ ▕▏
                    ▕▕  ▃▗▄▔▗▄▄▗▗▗▔▄▄▄▄▗▄▄▗▔▃▃▃▗▄▂▄▃▗▄▂▖▖ ▏▁
                    ▝▘▏ ▔▔   ▁▔▁▔▔▁▔▔▔▔▔▔▔▁▁ ▔▔   ▔▔▔▔
                             ▀ ▀▝▘▀▀▔▘▘▀▝▕▀▀▝▝▀▔▀ ▀▔▘
                            ▘ ▗▅▁▝▚▃▀▆▟██▙▆▝▃ ▘ ▁▗▌
                               ▔▀▔▝ ▔▀▟▜▛▛▀▔    ▀


ANTLOGO
    echo -e "${_SW_BOLD}AETHER COLONY :: Colony Activity${_SW_RESET}"
    echo -e "${_SW_DIM}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${_SW_RESET}"
    echo ""

    # Render each active ant (limit to 5)
    # SUPPRESS:OK -- read-default: display file may not exist yet
    jq -r '.active_ants[0:5][] | "\(.name)|\(.caste)|\(.status // "")|\(.task // "")|\(.tools.read // 0)|\(.tools.grep // 0)|\(.tools.edit // 0)|\(.tools.bash // 0)|\(.tokens // 0)|\(.started_at // "")|\(.parent // "Queen")|\(.progress // 0)"' "$display_file" 2>/dev/null | while IFS='|' read -r ant_name ant_caste ant_status ant_task read_ct grep_ct edit_ct bash_ct tokens started_at parent progress; do
      color=$(_sw_get_caste_color "$ant_caste")
      emoji=$(_sw_get_caste_emoji "$ant_caste")
      phrase=$(_sw_get_status_phrase "$ant_caste")

      # Format tools
      tools_str=$(_sw_format_tools "$read_ct" "$grep_ct" "$edit_ct" "$bash_ct")

      # Truncate task if too long
      display_task="$ant_task"
      [[ ${#display_task} -gt 35 ]] && display_task="${display_task:0:32}..."

      # Calculate elapsed time
      elapsed_str=""
      started_ts="${started_at:-}"
      if [[ -n "$started_ts" ]] && [[ "$started_ts" != "null" ]]; then
        started_ts=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "$started_ts" +%s 2>/dev/null)  # SUPPRESS:OK -- cross-platform: macOS date syntax
        if [[ -z "$started_ts" ]] || [[ "$started_ts" == "null" ]]; then
          started_ts=$(date -d "$started_ts" +%s 2>/dev/null) || started_ts=0  # SUPPRESS:OK -- cross-platform: Linux date syntax
        fi
        now_ts=$(date +%s)
        elapsed=0
        if [[ -n "$started_ts" ]] && [[ "$started_ts" -gt 0 ]] 2>/dev/null; then  # SUPPRESS:OK -- existence-test: value may not be numeric
          elapsed=$((now_ts - started_ts))
        fi
        if [[ ${elapsed:-0} -gt 0 ]]; then
          elapsed_str="($(_sw_format_duration $elapsed))"
        fi
      fi

      # Token indicator
      token_str=""
      if [[ -n "$tokens" ]] && [[ "$tokens" -gt 0 ]]; then
        token_str="🍯${tokens}"
      fi

      # Output ant line: "🐜 Builder: excavating... Implement auth 📖5 🔍3 (2m3s) 🍯1250"
      echo -e "${color}${emoji} ${_SW_BOLD}${ant_name}${_SW_RESET}${color}: ${phrase}${_SW_RESET} ${display_task}"
      echo -e "   ${tools_str} ${_SW_DIM}${elapsed_str}${_SW_RESET} ${token_str}"

      # Show progress bar if progress > 0
      if [[ -n "$progress" ]] && [[ "$progress" -gt 0 ]]; then
        progress_bar=$(_sw_render_progress_bar "$progress" 15)
        excavation_phrase=$(_sw_get_excavation_phrase "$progress")
        echo -e "   ${_SW_DIM}${progress_bar}${_SW_RESET}"
        echo -e "   ${_SW_DIM}${excavation_phrase}${_SW_RESET}"
      fi

      echo ""
    done

    # Chamber activity map
    echo -e "${_SW_DIM}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${_SW_RESET}"
    echo ""
    echo -e "${_SW_BOLD}Chamber Activity:${_SW_RESET}"

    # Show active chambers with fire intensity
    has_chamber_activity=0
    # SUPPRESS:OK -- read-default: returns fallback on failure
    jq -r '.chambers | to_entries[] | "\(.key)|\(.value.activity)|\(.value.icon)"' "$display_file" 2>/dev/null | \
    while IFS='|' read -r chamber activity icon; do
      if [[ -n "$activity" ]] && [[ "$activity" -gt 0 ]]; then
        has_chamber_activity=1
        if [[ "$activity" -ge 5 ]]; then
          fires="🔥🔥🔥"
        elif [[ "$activity" -ge 3 ]]; then
          fires="🔥🔥"
        else
          fires="🔥"
        fi
        chamber_name="${chamber//_/ }"
        echo -e "  ${icon} ${chamber_name} ${fires} (${activity} ants)"
      fi
    done

    if [[ "$has_chamber_activity" -eq 0 ]]; then
      echo -e "${_SW_DIM}  (no chamber activity)${_SW_RESET}"
    fi

    # Summary
    echo ""
    echo -e "${_SW_DIM}${total_active} forager$([[ "$total_active" -eq 1 ]] || echo "s") excavating...${_SW_RESET}"

    json_ok "{\"displayed\":true,\"ants\":$total_active}"
}

# ============================================================================
# Display helper functions (used by _swarm_display_text)
# These are local helpers that were defined inside the swarm-display-text case block
# ============================================================================

# Caste emoji lookup (text-only version)
_sw_get_emoji() {
    case "$1" in
      builder)       echo "🔨🐜" ;;
      watcher)       echo "👁️🐜" ;;
      scout)         echo "🔍🐜" ;;
      chaos)         echo "🎲🐜" ;;
      prime)         echo "👑🐜" ;;
      oracle)        echo "🔮🐜" ;;
      route_setter)  echo "🧭🐜" ;;
      archaeologist) echo "🏺🐜" ;;
      surveyor)      echo "📊🐜" ;;
      *)             echo "🐜" ;;
    esac
}

# Format tool counts (only non-zero)
_sw_format_tools_text() {
    local r="${1:-0}" g="${2:-0}" e="${3:-0}" b="${4:-0}"
    local result=""
    [[ "$r" -gt 0 ]] && result="${result}📖${r} "
    [[ "$g" -gt 0 ]] && result="${result}🔍${g} "
    [[ "$e" -gt 0 ]] && result="${result}✏️${e} "
    [[ "$b" -gt 0 ]] && result="${result}⚡${b}"
    echo "$result"
}

# Progress bar using block characters (no ANSI)
_sw_render_bar_text() {
    local pct="${1:-0}" w="${2:-10}"
    [[ "$pct" -lt 0 ]] && pct=0
    [[ "$pct" -gt 100 ]] && pct=100
    local filled=$((pct * w / 100))
    local empty=$((w - filled))
    local bar=""
    for ((i=0; i<filled; i++)); do bar+="█"; done
    for ((i=0; i<empty; i++)); do bar+="░"; done
    echo "[$bar] ${pct}%"
}

# Helper: parse ISO-8601 timestamp to epoch (macOS + Linux)
_sw_iso_to_epoch_text() {
    local iso="$1"
    local epoch=""
    epoch=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "$iso" +%s 2>/dev/null || true)  # SUPPRESS:OK -- cross-platform: macOS date syntax
    if [[ -z "$epoch" ]]; then
      epoch=$(date -d "$iso" +%s 2>/dev/null || true)  # SUPPRESS:OK -- cross-platform: Linux date syntax
    fi
    echo "${epoch:-0}"
}

# Helper: duration formatter (e.g., 45s, 3m12s)
_sw_format_duration_text() {
    local seconds="${1:-0}"
    if [[ "$seconds" -lt 60 ]]; then
      echo "${seconds}s"
    else
      local mins=$((seconds / 60))
      local secs=$((seconds % 60))
      echo "${mins}m${secs}s"
    fi
}

# Helper: compact number formatter (e.g., 1.2k, 2.4M)
_sw_format_compact_tokens() {
    local n="${1:-0}"
    if [[ "$n" -ge 1000000 ]]; then
      awk -v n="$n" 'BEGIN { printf "%.1fM", n/1000000 }'
    elif [[ "$n" -ge 1000 ]]; then
      awk -v n="$n" 'BEGIN { printf "%.1fk", n/1000 }'
    else
      echo "$n"
    fi
}

# ============================================================================
# _swarm_display_text
# Plain-text swarm display for Claude conversation (no ANSI codes)
# Usage: _swarm_display_text [swarm_id]
# ============================================================================
_swarm_display_text() {
    swarm_id="${1:-default-swarm}"
    display_file="$DATA_DIR/swarm-display.json"

    # Check for display file
    if [[ ! -f "$display_file" ]]; then
      echo "🐜 Colony idle"
      json_ok '{"displayed":false,"reason":"no_data"}'
      exit 0
    fi

    # Check for jq
    if ! command -v jq >/dev/null 2>&1; then
      echo "🐜 Swarm active (details unavailable)"
      json_ok '{"displayed":true,"warning":"jq_missing"}'
      exit 0
    fi

    # Read swarm data — handle both flat total_active and nested .summary.total_active
    # SUPPRESS:OK -- read-default: query may return empty
    total_active=$(jq -r '(.total_active // .summary.total_active // 0)' "$display_file" 2>/dev/null || echo "0")

    if [[ "$total_active" -eq 0 ]]; then
      echo "🐜 Colony idle"
      json_ok '{"displayed":true,"ants":0}'
      exit 0
    fi

    # Compact header
    echo "🐜 COLONY ACTIVITY"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    # SUPPRESS:OK -- read-default: query may return empty
    total_tokens=$(jq -r '[.active_ants[]?.tokens // 0] | add // 0' "$display_file" 2>/dev/null || echo "0")
    started_iso=$(jq -r '.timestamp // ""' "$display_file" 2>/dev/null || echo "")  # SUPPRESS:OK -- read-default: file may not exist yet
    elapsed_text="n/a"
    if [[ -n "$started_iso" && "$started_iso" != "null" ]]; then
      started_epoch=$(_sw_iso_to_epoch_text "$started_iso")
      now_epoch=$(date +%s)
      if [[ "$started_epoch" -gt 0 ]] 2>/dev/null; then  # SUPPRESS:OK -- existence-test: value may not be numeric
        total_elapsed=$((now_epoch - started_epoch))
        [[ "$total_elapsed" -lt 0 ]] && total_elapsed=0
        elapsed_text=$(_sw_format_duration_text "$total_elapsed")
      fi
    fi

    # Render each ant (max 5)
    # SUPPRESS:OK -- read-default: query may return empty
    jq -r '.active_ants[0:5][] | "\(.name)|\(.caste)|\(.task // "")|\(.tools.read // 0)|\(.tools.grep // 0)|\(.tools.edit // 0)|\(.tools.bash // 0)|\(.progress // 0)|\(.tokens // 0)|\(.started_at // "")"' "$display_file" 2>/dev/null | while IFS='|' read -r name caste task r g e b progress tokens started_at; do
      emoji=$(_sw_get_emoji "$caste")
      tools=$(_sw_format_tools_text "$r" "$g" "$e" "$b")
      bar=$(_sw_render_bar_text "${progress:-0}" 10)
      token_str=""
      elapsed_ant=""

      # Truncate task to 25 chars
      [[ ${#task} -gt 25 ]] && task="${task:0:22}..."

      if [[ -n "$tokens" && "$tokens" -gt 0 ]] 2>/dev/null; then  # SUPPRESS:OK -- existence-test: value may not be numeric
        token_str="🍯$(_sw_format_compact_tokens "$tokens")"
      fi

      if [[ -n "$started_at" && "$started_at" != "null" ]]; then
        ant_start_epoch=$(_sw_iso_to_epoch_text "$started_at")
        now_epoch=$(date +%s)
        if [[ "$ant_start_epoch" -gt 0 ]] 2>/dev/null; then  # SUPPRESS:OK -- existence-test: value may not be numeric
          ant_elapsed=$((now_epoch - ant_start_epoch))
          [[ "$ant_elapsed" -lt 0 ]] && ant_elapsed=0
          elapsed_ant="($(_sw_format_duration_text "$ant_elapsed"))"
        fi
      fi

      echo "${emoji} ${name} ${bar} ${task}"
      meta_line=""
      [[ -n "$tools" ]] && meta_line="${meta_line}${tools} "
      [[ -n "$elapsed_ant" ]] && meta_line="${meta_line}${elapsed_ant} "
      [[ -n "$token_str" ]] && meta_line="${meta_line}${token_str}"
      [[ -n "$meta_line" ]] && echo "   ${meta_line}"
      echo ""
    done

    # Overflow indicator
    if [[ "$total_active" -gt 5 ]]; then
      echo "   +$((total_active - 5)) more ants..."
      echo ""
    fi

    # Footer
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "⏱️ Elapsed: ${elapsed_text} | 🍯 Total: $(_sw_format_compact_tokens "$total_tokens") | ${total_active} ants active"

    json_ok "{\"displayed\":true,\"ants\":$total_active}"
}

# ============================================================================
# _swarm_timing_start
# Record start time for an ant
# Usage: _swarm_timing_start <ant_name>
# ============================================================================
_swarm_timing_start() {
    ant_name="${1:-}"
    [[ -z "$ant_name" ]] && json_err "$E_VALIDATION_FAILED" "Usage: swarm-timing-start <ant_name>"

    mkdir -p "$DATA_DIR"
    timing_file="$DATA_DIR/timing.log"
    ts=$(date +%s)
    ts_iso=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Remove any existing entry for this ant and append new one
    if [[ -f "$timing_file" ]]; then
      grep -v "^$ant_name|" "$timing_file" > "${timing_file}.tmp" 2>/dev/null || true  # SUPPRESS:OK -- read-default: file may not exist
      mv "${timing_file}.tmp" "$timing_file"
    fi
    echo "$ant_name|$ts|$ts_iso" >> "$timing_file"

    json_ok "{\"ant\":\"$ant_name\",\"started_at\":\"$ts_iso\",\"timestamp\":$ts}"
}

# ============================================================================
# _swarm_timing_get
# Get elapsed time for an ant
# Usage: _swarm_timing_get <ant_name>
# ============================================================================
_swarm_timing_get() {
    ant_name="${1:-}"
    [[ -z "$ant_name" ]] && json_err "$E_VALIDATION_FAILED" "Usage: swarm-timing-get <ant_name>"

    timing_file="$DATA_DIR/timing.log"

    if [[ ! -f "$timing_file" ]] || ! grep -q "^$ant_name|" "$timing_file" 2>/dev/null; then  # SUPPRESS:OK -- existence-test: file may not exist
      json_ok "{\"ant\":\"$ant_name\",\"started_at\":null,\"elapsed_seconds\":0,\"elapsed_formatted\":\"00:00\"}"
      exit 0
    fi

    # Read start time
    start_line=$(grep "^$ant_name|" "$timing_file" | tail -1)
    start_ts=$(echo "$start_line" | cut -d'|' -f2)
    start_iso=$(echo "$start_line" | cut -d'|' -f3)

    now=$(date +%s)
    elapsed=$((now - start_ts))

    # Format as MM:SS
    mins=$((elapsed / 60))
    secs=$((elapsed % 60))
    formatted=$(printf "%02d:%02d" $mins $secs)

    json_ok "{\"ant\":\"$ant_name\",\"started_at\":\"$start_iso\",\"elapsed_seconds\":$elapsed,\"elapsed_formatted\":\"$formatted\"}"
}

# ============================================================================
# _swarm_timing_eta
# Calculate ETA based on progress percentage
# Usage: _swarm_timing_eta <ant_name> <percent_complete>
# ============================================================================
_swarm_timing_eta() {
    ant_name="${1:-}"
    percent="${2:-0}"
    [[ -z "$ant_name" ]] && json_err "$E_VALIDATION_FAILED" "Usage: swarm-timing-eta <ant_name> <percent_complete>"

    # Validate percent is a number
    if ! [[ "$percent" =~ ^[0-9]+$ ]]; then
      percent=0
    fi

    # Clamp percent to 0-100
    if [[ $percent -lt 0 ]]; then
      percent=0
    elif [[ $percent -gt 100 ]]; then
      percent=100
    fi

    timing_file="$DATA_DIR/timing.log"

    if [[ ! -f "$timing_file" ]] || ! grep -q "^$ant_name|" "$timing_file" 2>/dev/null; then  # SUPPRESS:OK -- existence-test: file may not exist
      json_ok "{\"ant\":\"$ant_name\",\"percent\":$percent,\"eta_seconds\":null,\"eta_formatted\":\"--:--\"}"
      exit 0
    fi

    # Read start time
    start_ts=$(grep "^$ant_name|" "$timing_file" | tail -1 | cut -d'|' -f2)
    now=$(date +%s)
    elapsed=$((now - start_ts))

    # Calculate ETA
    if [[ $percent -le 0 ]]; then
      eta_seconds=null
      eta_formatted="--:--"
    elif [[ $percent -ge 100 ]]; then
      eta_seconds=0
      eta_formatted="00:00"
    else
      # ETA = (elapsed / percent) * (100 - percent)
      eta_seconds=$(( (elapsed * (100 - percent)) / percent ))
      mins=$((eta_seconds / 60))
      secs=$((eta_seconds % 60))
      eta_formatted=$(printf "%02d:%02d" $mins $secs)
    fi

    json_ok "{\"ant\":\"$ant_name\",\"percent\":$percent,\"eta_seconds\":$eta_seconds,\"eta_formatted\":\"$eta_formatted\"}"
}
