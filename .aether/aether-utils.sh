#!/bin/bash
# Aether Colony Utility Layer
# Single entry point for deterministic colony operations
#
# Usage: bash .aether/aether-utils.sh <subcommand> [args...]
#
# All subcommands output JSON to stdout.
# Non-zero exit on error with JSON error message to stderr.

set -euo pipefail

# Set up structured error handling for unexpected failures
# This works alongside set -e but provides better context (line number, command)
# The error_handler function is defined in error-handler.sh if sourced
trap 'if type error_handler &>/dev/null; then error_handler ${LINENO} "$BASH_COMMAND" $?; fi' ERR

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$SCRIPT_DIR/.." && pwd 2>/dev/null || echo "$SCRIPT_DIR")"
DATA_DIR="$AETHER_ROOT/.aether/data"

# Initialize lock state before sourcing (file-lock.sh trap needs these)
LOCK_ACQUIRED=${LOCK_ACQUIRED:-false}
CURRENT_LOCK=${CURRENT_LOCK:-""}

# Source shared infrastructure if available
[[ -f "$SCRIPT_DIR/utils/file-lock.sh" ]] && source "$SCRIPT_DIR/utils/file-lock.sh"
[[ -f "$SCRIPT_DIR/utils/atomic-write.sh" ]] && source "$SCRIPT_DIR/utils/atomic-write.sh"
[[ -f "$SCRIPT_DIR/utils/error-handler.sh" ]] && source "$SCRIPT_DIR/utils/error-handler.sh"

# Feature detection for graceful degradation
# These checks run silently - failures are logged but don't block operation
if type feature_disable &>/dev/null; then
  # Check if DATA_DIR is writable for activity logging
  [[ -w "$DATA_DIR" ]] 2>/dev/null || feature_disable "activity_log" "DATA_DIR not writable"

  # Check if git is available for git integration
  command -v git &>/dev/null || feature_disable "git_integration" "git not installed"

  # Check if jq is available for JSON processing
  command -v jq &>/dev/null || feature_disable "json_processing" "jq not installed"

  # Check if lock utilities are available
  [[ -f "$SCRIPT_DIR/utils/file-lock.sh" ]] || feature_disable "file_locking" "lock utilities not available"
fi

# Fallback atomic_write if not sourced (uses temp file + mv for true atomicity)
if ! type atomic_write &>/dev/null; then
  atomic_write() {
    local target="$1"
    local content="$2"
    local temp
    temp=$(mktemp)
    echo "$content" > "$temp"
    mv "$temp" "$target"
  }
fi

# --- JSON output helpers ---
# Success: JSON to stdout, exit 0
json_ok() { printf '{"ok":true,"result":%s}\n' "$1"; }

# Error: JSON to stderr, exit 1
# Use enhanced json_err from error-handler.sh if available, otherwise fallback
if ! type json_err &>/dev/null; then
  # Fallback: simple error format for backward compatibility
  json_err() {
    local message="${2:-$1}"
    printf '{"ok":false,"error":"%s"}\n' "$message" >&2
    exit 1
  }
fi

# --- Caste emoji helper ---
get_caste_emoji() {
  case "$1" in
    *Queen*|*QUEEN*|*queen*) echo "👑" ;;
    *Builder*|*builder*|*Bolt*|*Hammer*|*Forge*|*Mason*|*Brick*|*Anvil*|*Weld*) echo "🔨" ;;
    *Watcher*|*watcher*|*Vigil*|*Sentinel*|*Guard*|*Keen*|*Sharp*|*Hawk*|*Alert*) echo "👁️" ;;
    *Scout*|*scout*|*Swift*|*Dash*|*Ranger*|*Track*|*Seek*|*Path*|*Roam*|*Quest*) echo "🔍" ;;
    *Colonizer*|*colonizer*|*Pioneer*|*Map*|*Chart*|*Venture*|*Explore*|*Compass*|*Atlas*|*Trek*) echo "🗺️" ;;
    *Architect*|*architect*|*Blueprint*|*Draft*|*Design*|*Plan*|*Schema*|*Frame*|*Sketch*|*Model*) echo "🏛️" ;;
    *Chaos*|*chaos*|*Probe*|*Stress*|*Shake*|*Twist*|*Snap*|*Breach*|*Surge*|*Jolt*) echo "🎲" ;;
    *Archaeologist*|*archaeologist*|*Relic*|*Fossil*|*Dig*|*Shard*|*Epoch*|*Strata*|*Lore*|*Glyph*) echo "🏺" ;;
    *Oracle*|*oracle*|*Sage*|*Seer*|*Vision*|*Augur*|*Mystic*|*Sibyl*|*Delph*|*Pythia*) echo "🔮" ;;
    *Route*|*route*) echo "📋" ;;
    *) echo "🐜" ;;
  esac
}

# --- Subcommand dispatch ---
cmd="${1:-help}"
shift 2>/dev/null || true

case "$cmd" in
  help)
    cat <<'EOF'
{"ok":true,"commands":["help","version","validate-state","load-state","unload-state","error-add","error-pattern-check","error-summary","activity-log","activity-log-init","activity-log-read","learning-promote","learning-inject","generate-ant-name","spawn-log","spawn-complete","spawn-can-spawn","spawn-get-depth","spawn-tree-load","spawn-tree-active","spawn-tree-depth","update-progress","check-antipattern","error-flag-pattern","signature-scan","signature-match","flag-add","flag-check-blockers","flag-resolve","flag-acknowledge","flag-list","flag-auto-resolve","autofix-checkpoint","autofix-rollback","spawn-can-spawn-swarm","swarm-findings-init","swarm-findings-add","swarm-findings-read","swarm-solution-set","swarm-cleanup","grave-add","grave-check","generate-commit-message","version-check","registry-add","bootstrap-system","model-profile","model-get","model-list"],"description":"Aether Colony Utility Layer — deterministic ops for the ant colony"}
EOF
    ;;
  version)
    json_ok '"1.0.0"'
    ;;
  validate-state)
    case "${1:-}" in
      colony)
        [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found" '{"file":"COLONY_STATE.json"}'
        json_ok "$(jq '
          def chk(f;t): if has(f) then (if (.[f]|type) as $a | t | any(. == $a) then "pass" else "fail: \(f) is \(.[f]|type), expected \(t|join("|"))" end) else "fail: missing \(f)" end;
          def opt(f;t): if has(f) then (if (.[f]|type) as $a | t | any(. == $a) then "pass" else "fail: \(f) is \(.[f]|type), expected \(t|join("|"))" end) else "pass" end;
          {file:"COLONY_STATE.json", checks:[
            chk("goal";["null","string"]),
            chk("state";["string"]),
            chk("current_phase";["number"]),
            chk("plan";["object"]),
            chk("memory";["object"]),
            chk("errors";["object"]),
            chk("events";["array"]),
            opt("session_id";["string","null"]),
            opt("initialized_at";["string","null"]),
            opt("build_started_at";["string","null"])
          ]} | . + {pass: ([.checks[] | select(. == "pass")] | length) == (.checks | length)}
        ' "$DATA_DIR/COLONY_STATE.json")"
        ;;
      constraints)
        [[ -f "$DATA_DIR/constraints.json" ]] || json_err "$E_FILE_NOT_FOUND" "constraints.json not found" '{"file":"constraints.json"}'
        json_ok "$(jq '
          def arr(f): if has(f) and (.[f]|type) == "array" then "pass" else "fail: \(f) not array" end;
          {file:"constraints.json", checks:[
            arr("focus"),
            arr("constraints")
          ]} | . + {pass: ([.checks[] | select(. == "pass")] | length) == (.checks | length)}
        ' "$DATA_DIR/constraints.json")"
        ;;
      all)
        results=()
        for target in colony constraints; do
          results+=("$(bash "$SCRIPT_DIR/aether-utils.sh" validate-state "$target" 2>/dev/null || echo '{"ok":false}')")
        done
        combined=$(printf '%s\n' "${results[@]}" | jq -s '[.[] | .result // {file:"unknown",pass:false}]')
        all_pass=$(echo "$combined" | jq 'all(.pass)')
        json_ok "{\"pass\":$all_pass,\"files\":$combined}"
        ;;
      *)
        json_err "$E_VALIDATION_FAILED" "Usage: validate-state colony|constraints|all"
        ;;
    esac
    ;;
  error-add)
    [[ $# -ge 3 ]] || json_err "$E_VALIDATION_FAILED" "Usage: error-add <category> <severity> <description> [phase]"
    [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found" '{"file":"COLONY_STATE.json"}'
    id="err_$(date -u +%s)_$(head -c 2 /dev/urandom | od -An -tx1 | tr -d ' ')"
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    phase_val="${4:-null}"
    if [[ "$phase_val" =~ ^[0-9]+$ ]]; then
      phase_jq="$phase_val"
    else
      phase_jq="null"
    fi
    updated=$(jq --arg id "$id" --arg cat "$1" --arg sev "$2" --arg desc "$3" --argjson phase "$phase_jq" --arg ts "$ts" '
      .errors.records += [{id:$id, category:$cat, severity:$sev, description:$desc, root_cause:null, phase:$phase, task_id:null, timestamp:$ts}] |
      if (.errors.records|length) > 50 then .errors.records = .errors.records[-50:] else . end
    ' "$DATA_DIR/COLONY_STATE.json") || json_err "$E_JSON_INVALID" "Failed to update COLONY_STATE.json"
    atomic_write "$DATA_DIR/COLONY_STATE.json" "$updated"
    json_ok "\"$id\""
    ;;
  error-pattern-check)
    [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found" '{"file":"COLONY_STATE.json"}'
    json_ok "$(jq '
      .errors.records | group_by(.category) | map(select(length >= 3) |
        {category: .[0].category, count: length,
         first_seen: (sort_by(.timestamp) | first.timestamp),
         last_seen: (sort_by(.timestamp) | last.timestamp)})
    ' "$DATA_DIR/COLONY_STATE.json")"
    ;;
  error-summary)
    [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found" '{"file":"COLONY_STATE.json"}'
    json_ok "$(jq '{
      total: (.errors.records | length),
      by_category: (.errors.records | group_by(.category) | map({key: .[0].category, value: length}) | from_entries),
      by_severity: (.errors.records | group_by(.severity) | map({key: .[0].severity, value: length}) | from_entries)
    }' "$DATA_DIR/COLONY_STATE.json")"
    ;;
  activity-log)
    # Usage: activity-log <action> <caste_or_name> <description>
    # The caste_or_name can be: "Builder", "Hammer-42 (Builder)", etc.
    action="${1:-}"
    caste="${2:-}"
    description="${3:-}"
    [[ -z "$action" || -z "$caste" || -z "$description" ]] && json_err "$E_VALIDATION_FAILED" "Usage: activity-log <action> <caste_or_name> <description>"

    # Graceful degradation: check if activity logging is enabled
    if type feature_enabled &>/dev/null && ! feature_enabled "activity_log"; then
      json_warn "W_DEGRADED" "Activity logging disabled: $(type _feature_reason &>/dev/null && _feature_reason activity_log || echo 'unknown')"
      exit 0
    fi

    log_file="$DATA_DIR/activity.log"
    mkdir -p "$DATA_DIR"
    ts=$(date -u +"%H:%M:%S")
    emoji=$(get_caste_emoji "$caste")
    echo "[$ts] $emoji $action $caste: $description" >> "$log_file"
    json_ok '"logged"'
    ;;
  activity-log-init)
    phase_num="${1:-}"
    phase_name="${2:-}"
    [[ -z "$phase_num" ]] && json_err "$E_VALIDATION_FAILED" "Usage: activity-log-init <phase_num> [phase_name]"

    # Graceful degradation: check if activity logging is enabled
    if type feature_enabled &>/dev/null && ! feature_enabled "activity_log"; then
      json_warn "W_DEGRADED" "Activity logging disabled: $(type _feature_reason &>/dev/null && _feature_reason activity_log || echo 'unknown')"
      exit 0
    fi
    log_file="$DATA_DIR/activity.log"
    mkdir -p "$DATA_DIR"
    archive_file="$DATA_DIR/activity-phase-${phase_num}.log"
    # Copy current log to per-phase archive (preserve combined log intact)
    if [ -f "$log_file" ] && [ -s "$log_file" ]; then
      # Handle retry scenario: don't overwrite existing archive
      if [ -f "$archive_file" ]; then
        archive_file="$DATA_DIR/activity-phase-${phase_num}-$(date -u +%s).log"
      fi
      cp "$log_file" "$archive_file"
    fi
    # Append phase header to combined log (NOT truncate)
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    echo "" >> "$log_file"
    echo "🐜 ═══════════════════════════════════════════════════" >> "$log_file"
    echo "   P H A S E   $phase_num: ${phase_name:-unnamed}" >> "$log_file"
    echo "   $ts" >> "$log_file"
    echo "═══════════════════════════════════════════════════ 🐜" >> "$log_file"
    archived_flag="false"
    [ -f "$archive_file" ] && archived_flag="true"
    json_ok "{\"archived\":$archived_flag}"
    ;;
  activity-log-read)
    caste_filter="${1:-}"

    # Graceful degradation: check if activity logging is enabled
    if type feature_enabled &>/dev/null && ! feature_enabled "activity_log"; then
      json_warn "W_DEGRADED" "Activity logging disabled: $(type _feature_reason &>/dev/null && _feature_reason activity_log || echo 'unknown')"
      exit 0
    fi

    log_file="$DATA_DIR/activity.log"
    [[ -f "$log_file" ]] || json_err "$E_FILE_NOT_FOUND" "activity.log not found" '{"file":"activity.log"}'
    if [ -n "$caste_filter" ]; then
      content=$(grep "$caste_filter" "$log_file" | tail -20)
    else
      content=$(cat "$log_file")
    fi
    json_ok "$(echo "$content" | jq -Rs '.')"
    ;;
  learning-promote)
    [[ $# -ge 3 ]] || json_err "Usage: learning-promote <content> <source_project> <source_phase> [tags]"
    content="$1"
    source_project="$2"
    source_phase="$3"
    tags="${4:-}"

    mkdir -p "$DATA_DIR"
    global_file="$DATA_DIR/learnings.json"

    if [[ ! -f "$global_file" ]]; then
      echo '{"learnings":[],"version":1}' > "$global_file"
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
    ' "$global_file") || json_err "Failed to update learnings.json"

    echo "$updated" > "$global_file"
    json_ok "{\"promoted\":true,\"id\":\"$id\",\"count\":$((current_count + 1)),\"cap\":50}"
    ;;
  learning-inject)
    [[ $# -ge 1 ]] || json_err "Usage: learning-inject <tech_keywords_csv>"
    keywords="$1"

    global_file="$DATA_DIR/learnings.json"

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
    ;;
  spawn-log)
    # Usage: spawn-log <parent_id> <child_caste> <child_name> <task_summary> [model] [status]
    parent_id="${1:-}"
    child_caste="${2:-}"
    child_name="${3:-}"
    task_summary="${4:-}"
    model="${5:-default}"
    status="${6:-spawned}"
    [[ -z "$parent_id" || -z "$child_caste" || -z "$task_summary" ]] && json_err "Usage: spawn-log <parent_id> <child_caste> <child_name> <task_summary> [model] [status]"
    mkdir -p "$DATA_DIR"
    ts=$(date -u +"%H:%M:%S")
    ts_full=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    emoji=$(get_caste_emoji "$child_caste")
    parent_emoji=$(get_caste_emoji "$parent_id")
    # Log to activity log with spawn format, emojis, and model info
    echo "[$ts] ⚡ SPAWN $parent_emoji $parent_id -> $emoji $child_name ($child_caste): $task_summary [model: $model]" >> "$DATA_DIR/activity.log"
    # Log to spawn tree file for visualization (NEW FORMAT: includes model field)
    echo "$ts_full|$parent_id|$child_caste|$child_name|$task_summary|$model|$status" >> "$DATA_DIR/spawn-tree.txt"
    json_ok '"logged"'
    ;;
  spawn-complete)
    # Usage: spawn-complete <ant_name> <status> [summary]
    ant_name="${1:-}"
    status="${2:-completed}"
    summary="${3:-}"
    [[ -z "$ant_name" ]] && json_err "Usage: spawn-complete <ant_name> <status> [summary]"
    mkdir -p "$DATA_DIR"
    ts=$(date -u +"%H:%M:%S")
    ts_full=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    emoji=$(get_caste_emoji "$ant_name")
    status_icon="✅"
    [[ "$status" == "failed" ]] && status_icon="❌"
    [[ "$status" == "blocked" ]] && status_icon="🚫"
    echo "[$ts] $status_icon $emoji $ant_name: $status${summary:+ - $summary}" >> "$DATA_DIR/activity.log"
    # Update spawn tree
    echo "$ts_full|$ant_name|$status|$summary" >> "$DATA_DIR/spawn-tree.txt"
    json_ok '"logged"'
    ;;
  spawn-can-spawn)
    # Check if spawning is allowed at given depth
    # Usage: spawn-can-spawn <depth>
    # Returns: {can_spawn: bool, depth: N, max_spawns: N, current_total: N}
    depth="${1:-1}"

    # Depth limits: 1→4 spawns, 2→2 spawns, 3+→0 spawns
    if [[ $depth -eq 1 ]]; then
      max_for_depth=4
    elif [[ $depth -eq 2 ]]; then
      max_for_depth=2
    else
      max_for_depth=0
    fi

    # Count current spawns in this session (from spawn-tree.txt)
    current=0
    if [[ -f "$DATA_DIR/spawn-tree.txt" ]]; then
      current=$(grep -c "|spawned$" "$DATA_DIR/spawn-tree.txt" 2>/dev/null || echo 0)
    fi

    # Global cap of 10 workers per phase
    global_cap=10

    # Can spawn if: depth < 3 AND under global cap
    if [[ $depth -lt 3 && $current -lt $global_cap ]]; then
      can="true"
    else
      can="false"
    fi

    json_ok "{\"can_spawn\":$can,\"depth\":$depth,\"max_spawns\":$max_for_depth,\"current_total\":$current,\"global_cap\":$global_cap}"
    ;;
  spawn-get-depth)
    # Return depth for a given ant name by tracing spawn tree
    # Usage: spawn-get-depth <ant_name>
    # Queen = depth 0, Queen's spawns = depth 1, their spawns = depth 2, etc.
    ant_name="${1:-Queen}"

    if [[ "$ant_name" == "Queen" ]]; then
      json_ok '{"ant":"Queen","depth":0}'
      exit 0
    fi

    # Check if spawn tree exists
    if [[ ! -f "$DATA_DIR/spawn-tree.txt" ]]; then
      json_ok "{\"ant\":\"$ant_name\",\"depth\":1,\"found\":false}"
      exit 0
    fi

    # Check if ant exists in spawn tree (gracefully handle missing ants)
    if ! grep -q "|$ant_name|" "$DATA_DIR/spawn-tree.txt" 2>/dev/null; then
      json_ok "{\"ant\":\"$ant_name\",\"depth\":1,\"found\":false}"
      exit 0
    fi

    # Find the spawn record for this ant and trace parents
    depth=1
    current_ant="$ant_name"

    # Find who spawned this ant (look for lines with |spawned)
    while true; do
      # Format: timestamp|parent|caste|child_name|task|spawned
      parent=$(grep "|$current_ant|" "$DATA_DIR/spawn-tree.txt" 2>/dev/null | grep "|spawned$" | head -1 | cut -d'|' -f2 || echo "")

      if [[ -z "$parent" || "$parent" == "Queen" ]]; then
        break
      fi

      depth=$((depth + 1))
      current_ant="$parent"

      # Safety limit
      if [[ $depth -gt 5 ]]; then
        break
      fi
    done

    json_ok "{\"ant\":\"$ant_name\",\"depth\":$depth,\"found\":true}"
    ;;
  update-progress)
    # Usage: update-progress <percent> <message> [phase] [total_phases]
    percent="${1:-0}"
    message="${2:-Working...}"
    phase="${3:-1}"
    total="${4:-1}"
    mkdir -p "$DATA_DIR"

    # Calculate bar width (30 chars)
    bar_width=30
    filled=$((percent * bar_width / 100))
    empty=$((bar_width - filled))

    # Build progress bar with ASCII
    bar=""
    for ((i=0; i<filled; i++)); do bar+="█"; done
    for ((i=0; i<empty; i++)); do bar+="░"; done

    # Spinner frames for animation
    spinners=("⠋" "⠙" "⠹" "⠸" "⠼" "⠴" "⠦" "⠧" "⠇" "⠏")
    spin_idx=$(($(date +%s) % 10))
    spinner="${spinners[$spin_idx]}"

    # Status indicator
    if [[ $percent -ge 100 ]]; then
      status_icon="✅"
    elif [[ $percent -ge 50 ]]; then
      status_icon="🔨"
    else
      status_icon="$spinner"
    fi

    # Write progress file
    cat > "$DATA_DIR/watch-progress.txt" << EOF
       .-.
      (o o)  AETHER COLONY
      | O |  Progress
       \`-\`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Phase: $phase / $total

[$bar] $percent%

$status_icon $message

Target: 95% confidence

EOF
    json_ok "{\"percent\":$percent,\"phase\":$phase}"
    ;;
  error-flag-pattern)
    # Usage: error-flag-pattern <pattern_name> <description> [severity]
    # Tracks recurring error patterns across sessions for colony learning
    pattern_name="${1:-}"
    description="${2:-}"
    severity="${3:-warning}"
    [[ -z "$pattern_name" || -z "$description" ]] && json_err "Usage: error-flag-pattern <pattern_name> <description> [severity]"

    patterns_file="$DATA_DIR/error-patterns.json"
    mkdir -p "$DATA_DIR"

    if [[ ! -f "$patterns_file" ]]; then
      echo '{"patterns":[],"version":1}' > "$patterns_file"
    fi

    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    project_name=$(basename "$PWD")

    # Check if pattern already exists
    existing=$(jq --arg name "$pattern_name" '.patterns[] | select(.name == $name)' "$patterns_file" 2>/dev/null)

    if [[ -n "$existing" ]]; then
      # Update existing pattern - increment count
      updated=$(jq --arg name "$pattern_name" --arg ts "$ts" --arg proj "$project_name" '
        .patterns = [.patterns[] | if .name == $name then
          .occurrences += 1 |
          .last_seen = $ts |
          .projects = ((.projects + [$proj]) | unique)
        else . end]
      ' "$patterns_file") || json_err "Failed to update pattern"
      echo "$updated" > "$patterns_file"
      count=$(echo "$updated" | jq --arg name "$pattern_name" '.patterns[] | select(.name == $name) | .occurrences')
      json_ok "{\"updated\":true,\"pattern\":\"$pattern_name\",\"occurrences\":$count}"
    else
      # Add new pattern
      updated=$(jq --arg name "$pattern_name" --arg desc "$description" --arg sev "$severity" --arg ts "$ts" --arg proj "$project_name" '
        .patterns += [{
          "name": $name,
          "description": $desc,
          "severity": $sev,
          "first_seen": $ts,
          "last_seen": $ts,
          "occurrences": 1,
          "projects": [$proj],
          "resolved": false
        }]
      ' "$patterns_file") || json_err "Failed to add pattern"
      echo "$updated" > "$patterns_file"
      json_ok "{\"created\":true,\"pattern\":\"$pattern_name\"}"
    fi
    ;;
  error-patterns-check)
    # Check for known error patterns in a file or codebase
    # Returns patterns that should be avoided
    global_file="$DATA_DIR/error-patterns.json"

    if [[ ! -f "$global_file" ]]; then
      json_ok '{"patterns":[],"count":0}'
      exit 0
    fi

    # Return patterns with 2+ occurrences (recurring issues)
    json_ok "$(jq '{
      patterns: [.patterns[] | select(.occurrences >= 2 and .resolved == false)],
      count: ([.patterns[] | select(.occurrences >= 2 and .resolved == false)] | length)
    }' "$global_file")"
    ;;
  check-antipattern)
    # Usage: check-antipattern <file_path>
    # Returns JSON with critical issues and warnings
    file_path="${1:-}"
    [[ -z "$file_path" ]] && json_err "Usage: check-antipattern <file_path>"
    [[ ! -f "$file_path" ]] && json_ok '{"critical":[],"warnings":[],"clean":true}'

    criticals=()
    warnings=()

    # Detect file type
    ext="${file_path##*.}"

    case "$ext" in
      swift)
        # Swift didSet infinite recursion check
        if grep -n "didSet" "$file_path" 2>/dev/null | grep -q "self\."; then
          line=$(grep -n "didSet" "$file_path" | grep "self\." | head -1 | cut -d: -f1)
          criticals+=("{\"pattern\":\"didSet-recursion\",\"file\":\"$file_path\",\"line\":$line,\"message\":\"Potential didSet infinite recursion - self assignment in didSet\"}")
        fi
        ;;
      ts|tsx|js|jsx)
        # TypeScript any type check
        if grep -nE '\bany\b' "$file_path" 2>/dev/null | grep -qv "//.*any"; then
          count=$(grep -cE '\bany\b' "$file_path" 2>/dev/null || echo "0")
          warnings+=("{\"pattern\":\"typescript-any\",\"file\":\"$file_path\",\"count\":$count,\"message\":\"Found $count uses of 'any' type\"}")
        fi
        # Console.log in production code (not in test files)
        if [[ ! "$file_path" =~ \.test\. && ! "$file_path" =~ \.spec\. ]]; then
          if grep -n "console\.log" "$file_path" 2>/dev/null | grep -qv "//"; then
            count=$(grep -c "console\.log" "$file_path" 2>/dev/null || echo "0")
            warnings+=("{\"pattern\":\"console-log\",\"file\":\"$file_path\",\"count\":$count,\"message\":\"Found $count console.log statements\"}")
          fi
        fi
        ;;
      py)
        # Python bare except
        if grep -n "except:" "$file_path" 2>/dev/null | grep -qv "#"; then
          line=$(grep -n "except:" "$file_path" | head -1 | cut -d: -f1)
          warnings+=("{\"pattern\":\"bare-except\",\"file\":\"$file_path\",\"line\":$line,\"message\":\"Bare except clause - specify exception type\"}")
        fi
        ;;
    esac

    # Common patterns across all languages
    # Exposed secrets check (critical)
    if grep -nE "(api_key|apikey|secret|password|token)\s*=\s*['\"][^'\"]+['\"]" "$file_path" 2>/dev/null | grep -qvi "example\|test\|mock\|fake"; then
      line=$(grep -nE "(api_key|apikey|secret|password|token)\s*=\s*['\"]" "$file_path" | head -1 | cut -d: -f1)
      criticals+=("{\"pattern\":\"exposed-secret\",\"file\":\"$file_path\",\"line\":${line:-0},\"message\":\"Potential hardcoded secret or credential\"}")
    fi

    # TODO/FIXME check (warning)
    if grep -nE "(TODO|FIXME|XXX|HACK)" "$file_path" 2>/dev/null | head -1 | grep -q .; then
      count=$(grep -cE "(TODO|FIXME|XXX|HACK)" "$file_path" 2>/dev/null || echo "0")
      warnings+=("{\"pattern\":\"todo-comment\",\"file\":\"$file_path\",\"count\":$count,\"message\":\"Found $count TODO/FIXME comments\"}")
    fi

    # Build result JSON
    crit_json="[]"
    warn_json="[]"
    if [[ ${#criticals[@]} -gt 0 ]]; then
      crit_json=$(printf '%s\n' "${criticals[@]}" | jq -s '.')
    fi
    if [[ ${#warnings[@]} -gt 0 ]]; then
      warn_json=$(printf '%s\n' "${warnings[@]}" | jq -s '.')
    fi

    clean="true"
    [[ ${#criticals[@]} -gt 0 || ${#warnings[@]} -gt 0 ]] && clean="false"

    json_ok "{\"critical\":$crit_json,\"warnings\":$warn_json,\"clean\":$clean}"
    ;;
  signature-scan)
    # Scan a file for a signature pattern
    # Usage: signature-scan <target_file> <signature_name>
    # Returns matching signature details as JSON if found, empty result if no match
    # Exit code 0 if no match, 1 if match found
    target_file="${1:-}"
    signature_name="${2:-}"
    [[ -z "$target_file" || -z "$signature_name" ]] && json_err "Usage: signature-scan <target_file> <signature_name>"

    # Handle missing target file gracefully
    if [[ ! -f "$target_file" ]]; then
      json_ok '{"found":false,"signature":null}'
      exit 0
    fi

    # Read signature details from signatures.json
    signatures_file="$DATA_DIR/signatures.json"
    if [[ ! -f "$signatures_file" ]]; then
      json_ok '{"found":false,"signature":null}'
      exit 0
    fi

    # Extract signature details using jq
    signature_data=$(jq --arg name "$signature_name" '.signatures[] | select(.name == $name)' "$signatures_file" 2>/dev/null)

    if [[ -z "$signature_data" ]]; then
      # Signature not found in storage
      json_ok '{"found":false,"signature":null}'
      exit 0
    fi

    # Extract pattern and confidence threshold
    pattern_string=$(echo "$signature_data" | jq -r '.pattern_string // empty')
    confidence_threshold=$(echo "$signature_data" | jq -r '.confidence_threshold // 0.8')

    if [[ -z "$pattern_string" || "$pattern_string" == "null" ]]; then
      json_ok '{"found":false,"signature":null}'
      exit 0
    fi

    # Use grep to search for the pattern in target file
    if grep -q -- "$pattern_string" "$target_file" 2>/dev/null; then
      # Match found - return signature details with match info
      match_count=$(grep -c -- "$pattern_string" "$target_file" 2>/dev/null || echo "1")
      json_ok "{\"found\":true,\"signature\":$signature_data,\"match_count\":$match_count}"
      exit 1
    else
      # No match
      json_ok '{"found":false,"signature":null}'
      exit 0
    fi
    ;;
  signature-match)
    # Scan a directory for files matching high-confidence signatures
    # Usage: signature-match <directory> [file_pattern]
    # Returns results per file showing which signatures matched
    target_dir="${1:-}"
    file_pattern="${2:-}"
    # Set default pattern if not provided - avoid zsh brace expansion quirk by setting it explicitly
    if [[ -z "$file_pattern" ]]; then
      file_pattern="*"
    fi
    [[ -z "$target_dir" ]] && json_err "Usage: signature-match <directory> [file_pattern]"

    # Validate directory exists
    [[ ! -d "$target_dir" ]] && json_err "Directory not found: $target_dir"

    # Path to signatures file
    signatures_file="$DATA_DIR/signatures.json"
    [[ ! -f "$signatures_file" ]] && json_err "Signatures file not found"

    # Read high-confidence signatures (confidence >= 0.7) using jq -c for compact single-line output
    high_conf_signatures=$(jq -c '.signatures[] | select(.confidence_threshold >= 0.7)' "$signatures_file" 2>/dev/null)

    # Check if any high-confidence signatures exist
    sig_count=$(echo "$high_conf_signatures" | grep -c '{' || echo 0)
    if [[ "$sig_count" -eq 0 ]]; then
      json_ok '{"files_scanned":0,"matches":{},"signatures_checked":0}'
      exit 0
    fi

    # Find all files to scan
    declare -a files=()
    if [[ -n "$file_pattern" ]]; then
      # User specified pattern - use it directly
      while IFS= read -r -d '' file; do
        files+=("$file")
      done < <(find "$target_dir" -type f -name "$file_pattern" -print0 2>/dev/null || true)
    else
      # Default: match common code file types
      while IFS= read -r -d '' file; do
        files+=("$file")
      done < <(find "$target_dir" -type f \( -name "*.js" -o -name "*.ts" -o -name "*.py" -o -name "*.sh" -o -name "*.txt" -o -name "*.md" \) -print0 2>/dev/null || true)
    fi

    file_count=${#files[@]}

    # If no files found, return empty result
    if [[ "$file_count" -eq 0 ]]; then
      json_ok "{\"files_scanned\":0,\"matches\":{},\"signatures_checked\":$sig_count}"
      exit 0
    fi

    # Collect matches per file - process each file (bash 3.2 compatible: build JSON directly)
    matched_files="{}"

    # Read signatures into array first (avoid subshell issues)
    sig_array=""
    while IFS= read -r sig_entry; do
      [[ -z "$sig_entry" ]] && continue
      sig_array="${sig_array}${sig_entry}"$'\n'
    done <<< "$high_conf_signatures"

    for file in "${files[@]}"; do
      # For each file, check each signature - use process subst to avoid subshell
      file_key=$(basename "$file")
      matches_for_file="[]"

      while IFS= read -r sig_entry; do
        [[ -z "$sig_entry" ]] && continue
        sig_name=$(echo "$sig_entry" | jq -r '.name')
        sig_pattern=$(echo "$sig_entry" | jq -r '.pattern_string')
        sig_conf=$(echo "$sig_entry" | jq -r '.confidence_threshold')
        sig_desc=$(echo "$sig_entry" | jq -r '.description')

        # Skip if pattern is null/empty
        [[ -z "$sig_pattern" || "$sig_pattern" == "null" ]] && continue

        # Check if pattern matches in file using grep
        if grep -q -- "$sig_pattern" "$file" 2>/dev/null; then
          match_count=$(grep -c -- "$sig_pattern" "$file" 2>/dev/null || echo "1")

          # Add to results
          matches_for_file=$(echo "$matches_for_file" | jq --arg n "$sig_name" --arg d "$sig_desc" --argjson c "$sig_conf" --argjson m "$match_count" \
            '. += [{"name":$n,"description":$d,"confidence_threshold":$c,"match_count":$m}]')
        fi
      done < <(echo "$high_conf_signatures" | jq -c '.' 2>/dev/null || true)

      # If any signatures matched, add to results
      sig_result_count=$(echo "$matches_for_file" | jq 'length')
      if [[ "$sig_result_count" -gt 0 ]]; then
        temp_result=$(mktemp)
        echo "$matched_files" | jq --arg k "$file_key" --argjson v "$matches_for_file" '. + {($k): $v}' > "$temp_result"
        matched_files=$(cat "$temp_result")
        rm -f "$temp_result"
      fi
    done

    json_ok "{\"files_scanned\":$file_count,\"matches\":$matched_files,\"signatures_checked\":$sig_count}"
    ;;
  flag-add)
    # Add a project-specific flag (blocker, issue, or note)
    # Usage: flag-add <type> <title> <description> [source] [phase]
    # Types: blocker (critical, blocks advancement), issue (high, warning), note (low, info)
    type="${1:-issue}"
    title="${2:-}"
    desc="${3:-}"
    source="${4:-manual}"
    phase="${5:-null}"
    [[ -z "$title" ]] && json_err "Usage: flag-add <type> <title> <description> [source] [phase]"

    mkdir -p "$DATA_DIR"
    flags_file="$DATA_DIR/flags.json"

    if [[ ! -f "$flags_file" ]]; then
      echo '{"version":1,"flags":[]}' > "$flags_file"
    fi

    id="flag_$(date -u +%s)_$(head -c 2 /dev/urandom | od -An -tx1 | tr -d ' ')"
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Acquire lock for atomic flag update (degrade gracefully if locking unavailable)
    if type feature_enabled &>/dev/null && ! feature_enabled "file_locking"; then
      json_warn "W_DEGRADED" "File locking disabled - proceeding without lock: $(type _feature_reason &>/dev/null && _feature_reason file_locking || echo 'unknown')"
    else
      acquire_lock "$flags_file" || {
        if type json_err &>/dev/null; then
          json_err "$E_LOCK_FAILED" "Failed to acquire lock on flags.json"
        else
          echo '{"ok":false,"error":"Failed to acquire lock on flags.json"}' >&2
          exit 1
        fi
      }
    fi

    # Map type to severity
    case "$type" in
      blocker)  severity="critical" ;;
      issue)    severity="high" ;;
      note)     severity="low" ;;
      *)        severity="medium" ;;
    esac

    # Handle phase as number or null
    if [[ "$phase" =~ ^[0-9]+$ ]]; then
      phase_jq="$phase"
    else
      phase_jq="null"
    fi

    updated=$(jq --arg id "$id" --arg type "$type" --arg sev "$severity" \
      --arg title "$title" --arg desc "$desc" --arg source "$source" \
      --argjson phase "$phase_jq" --arg ts "$ts" '
      .flags += [{
        id: $id,
        type: $type,
        severity: $sev,
        title: $title,
        description: $desc,
        source: $source,
        phase: $phase,
        created_at: $ts,
        acknowledged_at: null,
        resolved_at: null,
        resolution: null,
        auto_resolve_on: (if $type == "blocker" and ($source | test("chaos") | not) then "build_pass" else null end)
      }]
    ' "$flags_file") || { release_lock "$flags_file"; json_err "Failed to add flag"; }

    atomic_write "$flags_file" "$updated"
    release_lock "$flags_file"
    json_ok "{\"id\":\"$id\",\"type\":\"$type\",\"severity\":\"$severity\"}"
    ;;
  flag-check-blockers)
    # Count unresolved blockers for the current phase
    # Usage: flag-check-blockers [phase]
    phase="${1:-}"
    flags_file="$DATA_DIR/flags.json"

    if [[ ! -f "$flags_file" ]]; then
      json_ok '{"blockers":0,"issues":0,"notes":0}'
      exit 0
    fi

    if [[ -n "$phase" && "$phase" =~ ^[0-9]+$ ]]; then
      # Filter by phase
      result=$(jq --argjson phase "$phase" '{
        blockers: [.flags[] | select(.type == "blocker" and .resolved_at == null and (.phase == $phase or .phase == null))] | length,
        issues: [.flags[] | select(.type == "issue" and .resolved_at == null and (.phase == $phase or .phase == null))] | length,
        notes: [.flags[] | select(.type == "note" and .resolved_at == null and (.phase == $phase or .phase == null))] | length
      }' "$flags_file")
    else
      # All unresolved
      result=$(jq '{
        blockers: [.flags[] | select(.type == "blocker" and .resolved_at == null)] | length,
        issues: [.flags[] | select(.type == "issue" and .resolved_at == null)] | length,
        notes: [.flags[] | select(.type == "note" and .resolved_at == null)] | length
      }' "$flags_file")
    fi

    json_ok "$result"
    ;;
  flag-resolve)
    # Resolve a flag with optional resolution message
    # Usage: flag-resolve <flag_id> [resolution_message]
    flag_id="${1:-}"
    resolution="${2:-Resolved}"
    [[ -z "$flag_id" ]] && json_err "Usage: flag-resolve <flag_id> [resolution_message]"

    flags_file="$DATA_DIR/flags.json"
    [[ ! -f "$flags_file" ]] && json_err "No flags file found"

    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Acquire lock for atomic flag update (degrade gracefully if locking unavailable)
    if type feature_enabled &>/dev/null && ! feature_enabled "file_locking"; then
      json_warn "W_DEGRADED" "File locking disabled - proceeding without lock"
    else
      acquire_lock "$flags_file" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on flags.json"
    fi

    updated=$(jq --arg id "$flag_id" --arg res "$resolution" --arg ts "$ts" '
      .flags = [.flags[] | if .id == $id then
        .resolved_at = $ts |
        .resolution = $res
      else . end]
    ' "$flags_file") || {
      if type feature_enabled &>/dev/null && feature_enabled "file_locking"; then
        release_lock "$flags_file"
      fi
      json_err "$E_JSON_INVALID" "Failed to resolve flag"
    }

    atomic_write "$flags_file" "$updated"
    release_lock "$flags_file"
    json_ok "{\"resolved\":\"$flag_id\"}"
    ;;
  flag-acknowledge)
    # Acknowledge a flag (issue continues but noted)
    # Usage: flag-acknowledge <flag_id>
    flag_id="${1:-}"
    [[ -z "$flag_id" ]] && json_err "Usage: flag-acknowledge <flag_id>"

    flags_file="$DATA_DIR/flags.json"
    [[ ! -f "$flags_file" ]] && json_err "No flags file found"

    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Acquire lock for atomic flag update (degrade gracefully if locking unavailable)
    if type feature_enabled &>/dev/null && ! feature_enabled "file_locking"; then
      json_warn "W_DEGRADED" "File locking disabled - proceeding without lock"
    else
      acquire_lock "$flags_file" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on flags.json"
    fi

    updated=$(jq --arg id "$flag_id" --arg ts "$ts" '
      .flags = [.flags[] | if .id == $id then
        .acknowledged_at = $ts
      else . end]
    ' "$flags_file") || {
      if type feature_enabled &>/dev/null && feature_enabled "file_locking"; then
        release_lock "$flags_file"
      fi
      json_err "$E_JSON_INVALID" "Failed to acknowledge flag"
    }

    atomic_write "$flags_file" "$updated"
    release_lock "$flags_file"
    json_ok "{\"acknowledged\":\"$flag_id\"}"
    ;;
  flag-list)
    # List flags, optionally filtered
    # Usage: flag-list [--all] [--type blocker|issue|note] [--phase N]
    flags_file="$DATA_DIR/flags.json"
    show_all="false"
    filter_type=""
    filter_phase=""

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --all) show_all="true"; shift ;;
        --type) filter_type="$2"; shift 2 ;;
        --phase) filter_phase="$2"; shift 2 ;;
        *) shift ;;
      esac
    done

    if [[ ! -f "$flags_file" ]]; then
      json_ok '{"flags":[],"count":0}'
      exit 0
    fi

    # Build jq filter
    jq_filter='.flags'

    if [[ "$show_all" != "true" ]]; then
      jq_filter+=' | [.[] | select(.resolved_at == null)]'
    fi

    if [[ -n "$filter_type" ]]; then
      jq_filter+=" | [.[] | select(.type == \"$filter_type\")]"
    fi

    if [[ -n "$filter_phase" && "$filter_phase" =~ ^[0-9]+$ ]]; then
      jq_filter+=" | [.[] | select(.phase == $filter_phase or .phase == null)]"
    fi

    result=$(jq "{flags: ($jq_filter), count: ($jq_filter | length)}" "$flags_file")
    json_ok "$result"
    ;;
  flag-auto-resolve)
    # Auto-resolve flags based on trigger (e.g., build_pass)
    # Usage: flag-auto-resolve <trigger>
    trigger="${1:-build_pass}"
    flags_file="$DATA_DIR/flags.json"

    if [[ ! -f "$flags_file" ]]; then json_ok '{"resolved":0}'; exit 0; fi

    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Acquire lock for atomic flag update (degrade gracefully if locking unavailable)
    if type feature_enabled &>/dev/null && ! feature_enabled "file_locking"; then
      json_warn "W_DEGRADED" "File locking disabled - proceeding without lock"
    else
      acquire_lock "$flags_file" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on flags.json"
    fi

    # Count how many will be resolved
    count=$(jq --arg trigger "$trigger" '
      [.flags[] | select(.auto_resolve_on == $trigger and .resolved_at == null)] | length
    ' "$flags_file")

    # Resolve them
    updated=$(jq --arg trigger "$trigger" --arg ts "$ts" '
      .flags = [.flags[] | if .auto_resolve_on == $trigger and .resolved_at == null then
        .resolved_at = $ts |
        .resolution = "Auto-resolved on " + $trigger
      else . end]
    ' "$flags_file")

    atomic_write "$flags_file" "$updated"
    release_lock "$flags_file"
    json_ok "{\"resolved\":$count,\"trigger\":\"$trigger\"}"
    ;;
  generate-ant-name)
    caste="${1:-builder}"
    # Caste-specific prefixes for personality
    case "$caste" in
      builder)  prefixes=("Chip" "Hammer" "Forge" "Mason" "Brick" "Anvil" "Weld" "Bolt") ;;
      watcher)  prefixes=("Vigil" "Sentinel" "Guard" "Keen" "Sharp" "Hawk" "Watch" "Alert") ;;
      scout)    prefixes=("Swift" "Dash" "Ranger" "Track" "Seek" "Path" "Roam" "Quest") ;;
      colonizer) prefixes=("Pioneer" "Map" "Chart" "Venture" "Explore" "Compass" "Atlas" "Trek") ;;
      architect) prefixes=("Blueprint" "Draft" "Design" "Plan" "Schema" "Frame" "Sketch" "Model") ;;
      prime)    prefixes=("Prime" "Alpha" "Lead" "Chief" "First" "Core" "Apex" "Crown") ;;
      chaos)    prefixes=("Probe" "Stress" "Shake" "Twist" "Snap" "Breach" "Surge" "Jolt") ;;
      archaeologist) prefixes=("Relic" "Fossil" "Dig" "Shard" "Epoch" "Strata" "Lore" "Glyph") ;;
      oracle)   prefixes=("Sage" "Seer" "Vision" "Augur" "Mystic" "Sibyl" "Delph" "Pythia") ;;
      *)        prefixes=("Ant" "Worker" "Drone" "Toiler" "Marcher" "Runner" "Carrier" "Helper") ;;
    esac
    # Pick random prefix and add random number
    idx=$((RANDOM % ${#prefixes[@]}))
    prefix="${prefixes[$idx]}"
    num=$((RANDOM % 99 + 1))
    name="${prefix}-${num}"
    json_ok "\"$name\""
    ;;

  # ============================================
  # SWARM UTILITIES (ant:swarm support)
  # ============================================

  autofix-checkpoint)
    # Create checkpoint before applying auto-fix
    # Usage: autofix-checkpoint [label]
    # Returns: {type: "stash"|"commit"|"none", ref: "..."}
    # IMPORTANT: Only stash Aether-related files, never touch user work
    if git rev-parse --git-dir >/dev/null 2>&1; then
      # Check if there are changes to Aether-managed files only
      # Target directories that Aether is allowed to modify
      target_dirs=".aether .claude/commands/ant .claude/commands/st .opencode runtime bin"
      has_changes=false

      for dir in $target_dirs; do
        if [[ -d "$dir" ]] && [[ -n "$(git status --porcelain "$dir" 2>/dev/null)" ]]; then
          has_changes=true
          break
        fi
      done

      if [[ "$has_changes" == "true" ]]; then
        label="${1:-autofix-$(date +%s)}"
        stash_name="aether-checkpoint: $label"
        # Only stash Aether-managed directories, never touch user files
        if git stash push -m "$stash_name" -- $target_dirs >/dev/null 2>&1; then
          json_ok "{\"type\":\"stash\",\"ref\":\"$stash_name\"}"
        else
          # Stash failed (possibly due to conflicts), record commit hash
          hash=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
          json_ok "{\"type\":\"commit\",\"ref\":\"$hash\"}"
        fi
      else
        # No changes in Aether-managed directories, just record commit hash
        hash=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
        json_ok "{\"type\":\"commit\",\"ref\":\"$hash\"}"
      fi
    else
      json_ok '{"type":"none","ref":null}'
    fi
    ;;

  autofix-rollback)
    # Rollback from checkpoint if fix failed
    # Usage: autofix-rollback <type> <ref>
    # Returns: {rolled_back: bool, method: "stash"|"reset"|"none"}
    ref_type="${1:-none}"
    ref="${2:-}"

    case "$ref_type" in
      stash)
        # Find and pop the stash
        stash_ref=$(git stash list 2>/dev/null | grep "$ref" | head -1 | cut -d: -f1 || echo "")
        if [[ -n "$stash_ref" ]]; then
          if git stash pop "$stash_ref" >/dev/null 2>&1; then
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
          if git reset --hard "$ref" >/dev/null 2>&1; then
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
    ;;

  spawn-can-spawn-swarm)
    # Check if swarm can spawn more scouts (separate from phase workers)
    # Usage: spawn-can-spawn-swarm <swarm_id>
    # Swarm has its own cap of 6 (4 scouts + 2 sub-scouts max)
    swarm_id="${1:-swarm}"
    swarm_cap=6

    current=0
    if [[ -f "$DATA_DIR/spawn-tree.txt" ]]; then
      current=$(grep -c "|swarm:$swarm_id$" "$DATA_DIR/spawn-tree.txt" 2>/dev/null || echo 0)
    fi

    if [[ $current -lt $swarm_cap ]]; then
      can="true"
      remaining=$((swarm_cap - current))
    else
      can="false"
      remaining=0
    fi

    json_ok "{\"can_spawn\":$can,\"current\":$current,\"cap\":$swarm_cap,\"remaining\":$remaining,\"swarm_id\":\"$swarm_id\"}"
    ;;

  swarm-findings-init)
    # Initialize swarm findings file
    # Usage: swarm-findings-init <swarm_id>
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
    ;;

  swarm-findings-add)
    # Add a finding from a scout
    # Usage: swarm-findings-add <swarm_id> <scout_type> <confidence> <finding_json>
    swarm_id="${1:-}"
    scout_type="${2:-}"
    confidence="${3:-0.5}"
    finding="${4:-}"

    [[ -z "$swarm_id" || -z "$scout_type" || -z "$finding" ]] && json_err "Usage: swarm-findings-add <swarm_id> <scout_type> <confidence> <finding_json>"

    findings_file="$DATA_DIR/swarm-findings-$swarm_id.json"
    [[ ! -f "$findings_file" ]] && json_err "Swarm findings file not found: $swarm_id"

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

    echo "$updated" > "$findings_file"
    count=$(echo "$updated" | jq '.findings | length')
    json_ok "{\"added\":true,\"scout\":\"$scout_type\",\"total_findings\":$count}"
    ;;

  swarm-findings-read)
    # Read all findings for a swarm
    # Usage: swarm-findings-read <swarm_id>
    swarm_id="${1:-}"
    [[ -z "$swarm_id" ]] && json_err "Usage: swarm-findings-read <swarm_id>"

    findings_file="$DATA_DIR/swarm-findings-$swarm_id.json"
    [[ ! -f "$findings_file" ]] && json_err "Swarm findings file not found: $swarm_id"

    json_ok "$(cat "$findings_file")"
    ;;

  swarm-solution-set)
    # Set the chosen solution for a swarm
    # Usage: swarm-solution-set <swarm_id> <solution_json>
    swarm_id="${1:-}"
    solution="${2:-}"

    [[ -z "$swarm_id" || -z "$solution" ]] && json_err "Usage: swarm-solution-set <swarm_id> <solution_json>"

    findings_file="$DATA_DIR/swarm-findings-$swarm_id.json"
    [[ ! -f "$findings_file" ]] && json_err "Swarm findings file not found: $swarm_id"

    ts=$(date -u +%Y-%m-%dT%H:%M:%SZ)

    updated=$(jq --argjson solution "$solution" --arg ts "$ts" '
      .solution = $solution |
      .status = "resolved" |
      .resolved_at = $ts
    ' "$findings_file")

    echo "$updated" > "$findings_file"
    json_ok "{\"solution_set\":true,\"swarm_id\":\"$swarm_id\"}"
    ;;

  swarm-cleanup)
    # Clean up swarm files after completion
    # Usage: swarm-cleanup <swarm_id> [--archive]
    swarm_id="${1:-}"
    archive="${2:-}"

    [[ -z "$swarm_id" ]] && json_err "Usage: swarm-cleanup <swarm_id> [--archive]"

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
    ;;

  grave-add)
    # Record a grave marker when a builder fails at a file
    # Usage: grave-add <file> <ant_name> <task_id> <phase> <failure_summary> [function] [line]
    [[ $# -ge 5 ]] || json_err "Usage: grave-add <file> <ant_name> <task_id> <phase> <failure_summary> [function] [line]"
    [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found" '{"file":"COLONY_STATE.json"}'
    file="$1"
    ant_name="$2"
    task_id="$3"
    phase="$4"
    failure_summary="$5"
    func="${6:-null}"
    line="${7:-null}"
    id="grave_$(date -u +%s)_$(head -c 2 /dev/urandom | od -An -tx1 | tr -d ' ')"
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    if [[ "$phase" =~ ^[0-9]+$ ]]; then
      phase_jq="$phase"
    else
      phase_jq="null"
    fi
    if [[ "$func" == "null" ]]; then
      func_jq="null"
    else
      func_jq="\"$func\""
    fi
    if [[ "$line" =~ ^[0-9]+$ ]]; then
      line_jq="$line"
    else
      line_jq="null"
    fi
    updated=$(jq --arg id "$id" --arg file "$file" --arg ant "$ant_name" --arg tid "$task_id" \
      --argjson phase "$phase_jq" --arg summary "$failure_summary" \
      --argjson func "$func_jq" --argjson line "$line_jq" --arg ts "$ts" '
      (.graveyards // []) as $graves |
      . + {graveyards: ($graves + [{
        id: $id,
        file: $file,
        ant_name: $ant,
        task_id: $tid,
        phase: $phase,
        failure_summary: $summary,
        function: $func,
        line: $line,
        timestamp: $ts
      }])} |
      if (.graveyards | length) > 30 then .graveyards = .graveyards[-30:] else . end
    ' "$DATA_DIR/COLONY_STATE.json") || json_err "$E_JSON_INVALID" "Failed to update COLONY_STATE.json"
    atomic_write "$DATA_DIR/COLONY_STATE.json" "$updated"
    json_ok "\"$id\""
    ;;

  grave-check)
    # Query for grave markers near a file path
    # Usage: grave-check <file_path>
    # Read-only, never modifies state
    [[ $# -ge 1 ]] || json_err "Usage: grave-check <file_path>"
    [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found" '{"file":"COLONY_STATE.json"}'
    check_file="$1"
    check_dir=$(dirname "$check_file")
    json_ok "$(jq --arg file "$check_file" --arg dir "$check_dir" '
      (.graveyards // []) as $graves |
      ($graves | map(select(.file == $file))) as $exact |
      ($graves | map(select((.file | split("/")[:-1] | join("/")) == $dir))) as $dir_matches |
      ($exact | length) as $exact_count |
      ($dir_matches | length) as $dir_count |
      (if $exact_count > 0 then "high"
       elif $dir_count >= 2 then "high"
       elif $dir_count == 1 then "low"
       else "none" end) as $caution |
      {graves: $dir_matches, count: $dir_count, exact_matches: $exact_count, caution_level: $caution}
    ' "$DATA_DIR/COLONY_STATE.json")"
    ;;

  # ============================================
  # GIT COMMIT UTILITIES
  # ============================================

  generate-commit-message)
    # Generate an intelligent commit message from colony context
    # Usage: generate-commit-message <type> <phase_id> <phase_name> [summary]
    # Types: "milestone" | "pause" | "fix"
    # Returns: {"message": "...", "body": "...", "files_changed": N}

    msg_type="${1:-milestone}"
    phase_id="${2:-0}"
    phase_name="${3:-unknown}"
    summary="${4:-}"

    # Count changed files
    files_changed=0
    if git rev-parse --git-dir >/dev/null 2>&1; then
      files_changed=$(git diff --stat --cached HEAD 2>/dev/null | tail -1 | grep -oE '[0-9]+ file' | grep -oE '[0-9]+' || echo "0")
      if [[ "$files_changed" == "0" ]]; then
        files_changed=$(git status --porcelain 2>/dev/null | wc -l | tr -d ' ')
      fi
    fi

    case "$msg_type" in
      milestone)
        # Format: aether-milestone: phase N complete -- <name>
        if [[ -n "$summary" ]]; then
          message="aether-milestone: phase ${phase_id} complete -- ${summary}"
        else
          message="aether-milestone: phase ${phase_id} complete -- ${phase_name}"
        fi
        body="All verification gates passed. User confirmed runtime behavior."
        ;;
      pause)
        message="aether-checkpoint: session pause -- phase ${phase_id} in progress"
        body="Colony paused mid-session. Handoff document saved."
        ;;
      fix)
        if [[ -n "$summary" ]]; then
          message="fix: ${summary}"
        else
          message="fix: resolve issue in phase ${phase_id}"
        fi
        body="Swarm-verified fix applied and tested."
        ;;
      *)
        message="aether-checkpoint: phase ${phase_id}"
        body=""
        ;;
    esac

    # Enforce 72-char limit on subject line (truncate if needed)
    if [[ ${#message} -gt 72 ]]; then
      message="${message:0:69}..."
    fi

    json_ok "{\"message\":\"$message\",\"body\":\"$body\",\"files_changed\":$files_changed}"
    ;;

  # ============================================
  # REGISTRY & UPDATE UTILITIES
  # ============================================

  version-check)
    # Compare local .aether/version.json vs ~/.aether/version.json
    # Outputs a notice string if versions differ, empty if matched or missing
    local_version_file="$AETHER_ROOT/.aether/version.json"
    hub_version_file="$HOME/.aether/version.json"

    # Silent exit if either file is missing
    if [[ ! -f "$local_version_file" || ! -f "$hub_version_file" ]]; then
      json_ok '""'
      exit 0
    fi

    local_ver=$(jq -r '.version // "unknown"' "$local_version_file" 2>/dev/null || echo "unknown")
    hub_ver=$(jq -r '.version // "unknown"' "$hub_version_file" 2>/dev/null || echo "unknown")

    if [[ "$local_ver" == "$hub_ver" ]]; then
      json_ok '""'
    else
      json_ok "\"Update available: $local_ver -> $hub_ver (run /ant:update)\""
    fi
    ;;

  registry-add)
    # Add or update a repo entry in ~/.aether/registry.json
    # Usage: registry-add <repo_path> <version>
    repo_path="${1:-}"
    repo_version="${2:-}"
    [[ -z "$repo_path" || -z "$repo_version" ]] && json_err "Usage: registry-add <repo_path> <version>"

    registry_file="$HOME/.aether/registry.json"
    mkdir -p "$HOME/.aether"

    if [[ ! -f "$registry_file" ]]; then
      echo '{"schema_version":1,"repos":[]}' > "$registry_file"
    fi

    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Check if repo already exists in registry
    existing=$(jq --arg path "$repo_path" '.repos[] | select(.path == $path)' "$registry_file" 2>/dev/null)

    if [[ -n "$existing" ]]; then
      # Update existing entry
      updated=$(jq --arg path "$repo_path" --arg ver "$repo_version" --arg ts "$ts" '
        .repos = [.repos[] | if .path == $path then
          .version = $ver |
          .updated_at = $ts
        else . end]
      ' "$registry_file") || json_err "Failed to update registry"
    else
      # Add new entry
      updated=$(jq --arg path "$repo_path" --arg ver "$repo_version" --arg ts "$ts" '
        .repos += [{
          "path": $path,
          "version": $ver,
          "registered_at": $ts,
          "updated_at": $ts
        }]
      ' "$registry_file") || json_err "Failed to update registry"
    fi

    echo "$updated" > "$registry_file"
    json_ok "{\"registered\":true,\"path\":\"$repo_path\",\"version\":\"$repo_version\"}"
    ;;

  bootstrap-system)
    # Copy system files from ~/.aether/system/ into local .aether/
    # Uses explicit allowlist — never touches colony data
    hub_system="$HOME/.aether/system"
    local_aether="$AETHER_ROOT/.aether"

    [[ ! -d "$hub_system" ]] && json_err "Hub system directory not found: $hub_system"

    # Allowlist of system files to copy (relative to system/)
    allowlist=(
      "aether-utils.sh"
      "coding-standards.md"
      "debugging.md"
      "DISCIPLINES.md"
      "learning.md"
      "planning.md"
      "QUEEN_ANT_ARCHITECTURE.md"
      "tdd.md"
      "verification-loop.md"
      "verification.md"
      "workers.md"
      "docs/constraints.md"
      "docs/pathogen-schema-example.json"
      "docs/pathogen-schema.md"
      "docs/pheromones.md"
      "docs/progressive-disclosure.md"
      "utils/atomic-write.sh"
      "utils/colorize-log.sh"
      "utils/file-lock.sh"
      "utils/watch-spawn-tree.sh"
    )

    copied=0
    for file in "${allowlist[@]}"; do
      src="$hub_system/$file"
      dest="$local_aether/$file"
      if [[ -f "$src" ]]; then
        mkdir -p "$(dirname "$dest")"
        cp "$src" "$dest"
        # Preserve executable bit for shell scripts
        if [[ "$file" == *.sh ]]; then
          chmod 755 "$dest"
        fi
        copied=$((copied + 1))
      fi
    done

    json_ok "{\"copied\":$copied,\"total\":${#allowlist[@]}}"
    ;;

  load-state)
    source "$SCRIPT_DIR/utils/state-loader.sh" 2>/dev/null || {
      json_err "$E_FILE_NOT_FOUND" "state-loader.sh not found"
      exit 1
    }
    load_colony_state
    if [[ $? -eq 0 ]]; then
      # Output success with handoff info if detected
      if [[ "$HANDOFF_DETECTED" == "true" ]]; then
        json_ok "{\"loaded\":true,\"handoff_detected\":true,\"handoff_summary\":\"$(get_handoff_summary)\"}"
      else
        json_ok '{"loaded":true}'
      fi
    fi
    # Note: load_colony_state handles its own error output
    ;;

  unload-state)
    source "$SCRIPT_DIR/utils/state-loader.sh" 2>/dev/null || {
      json_err "$E_FILE_NOT_FOUND" "state-loader.sh not found"
      exit 1
    }
    unload_colony_state
    json_ok '{"unloaded":true}'
    ;;

  spawn-tree-load)
    source "$SCRIPT_DIR/utils/spawn-tree.sh" 2>/dev/null || {
      json_err "$E_FILE_NOT_FOUND" "spawn-tree.sh not found"
      exit 1
    }
    tree_json=$(reconstruct_tree_json)
    json_ok "$tree_json"
    ;;

  spawn-tree-active)
    source "$SCRIPT_DIR/utils/spawn-tree.sh" 2>/dev/null || {
      json_err "$E_FILE_NOT_FOUND" "spawn-tree.sh not found"
      exit 1
    }
    active=$(get_active_spawns)
    json_ok "$active"
    ;;

  spawn-tree-depth)
    ant_name="${1:-}"
    [[ -z "$ant_name" ]] && json_err "$E_VALIDATION_FAILED" "Usage: spawn-tree-depth <ant_name>"
    source "$SCRIPT_DIR/utils/spawn-tree.sh" 2>/dev/null || {
      json_err "$E_FILE_NOT_FOUND" "spawn-tree.sh not found"
      exit 1
    }
    depth=$(get_spawn_depth "$ant_name")
    json_ok "$depth"
    ;;

  # --- Model Profile Commands ---
  model-profile)
    action="${1:-get}"
    case "$action" in
      get)
        caste="${2:-}"
        [[ -z "$caste" ]] && json_err "$E_VALIDATION_FAILED" "Usage: model-profile get <caste>"

        profile_file="$AETHER_ROOT/.aether/model-profiles.yaml"
        if [[ ! -f "$profile_file" ]]; then
          json_ok '{"model":"kimi-k2.5","source":"default","caste":"'$caste'"}'
          exit 0
        fi

        # Extract model for caste using awk (bash-compatible YAML parsing)
        model=$(awk '/^worker_models:/{found=1; next} found && /^[^ ]/{exit} found && /^  '$caste':/{print $2; exit}' "$profile_file" 2>/dev/null)

        [[ -z "$model" ]] && model="kimi-k2.5"
        json_ok '{"model":"'$model'","source":"profile","caste":"'$caste'"}'
        ;;

      list)
        profile_file="$AETHER_ROOT/.aether/model-profiles.yaml"
        if [[ ! -f "$profile_file" ]]; then
          json_ok '{"models":{},"source":"default"}'
          exit 0
        fi

        # Extract all caste:model pairs as JSON
        # Lines look like: "  prime: glm-5           # Complex coordination..."
        models=$(awk '/^worker_models:/{found=1; next} found && /^[^ ]/{exit} found && /^  [a-z_]+:/{gsub(/:/,""); printf "\"%s\":\"%s\",", $1, $2}' "$profile_file" 2>/dev/null)
        # Remove trailing comma
        models="${models%,}"

        json_ok '{"models":{'$models'},"source":"profile"}'
        ;;

      verify)
        profile_file="$AETHER_ROOT/.aether/model-profiles.yaml"
        [[ ! -f "$profile_file" ]] && json_err "$E_FILE_NOT_FOUND" "Profile not found" '{"file":"model-profiles.yaml"}'

        # Check proxy health
        proxy_health=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:4000/health 2>/dev/null || echo "000")
        proxy_status=$([[ "$proxy_health" == "200" ]] && echo "healthy" || echo "unhealthy")

        # Count castes
        caste_count=$(awk '/^worker_models:/{found=1; next} found && /^[^ ]/{exit} found && /^  [a-z_]+:/{count++} END{print count+0}' "$profile_file" 2>/dev/null)

        json_ok '{"profile_exists":true,"caste_count":'$caste_count',"proxy_status":"'$proxy_status'","proxy_endpoint":"http://localhost:4000"}'
        ;;

      *)
        json_err "$E_VALIDATION_FAILED" "Usage: model-profile get <caste>|list|verify"
        ;;
    esac
    ;;

  model-get)
    # Shortcut: model-get <caste>
    caste="${1:-}"
    [[ -z "$caste" ]] && json_err "$E_VALIDATION_FAILED" "Usage: model-get <caste>"

    # Delegate to model-profile get
    exec bash "$0" model-profile get "$caste"
    ;;

  model-list)
    # Shortcut: list all models
    exec bash "$0" model-profile list
    ;;

  *)
    json_err "Unknown command: $cmd"
    ;;
esac
