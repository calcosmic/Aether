#!/bin/bash
# Aether Colony Utility Layer
# Single entry point for deterministic colony operations
#
# Usage: bash ~/.aether/aether-utils.sh <subcommand> [args...]
#
# All subcommands output JSON to stdout.
# Non-zero exit on error with JSON error message to stderr.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$SCRIPT_DIR/.." && pwd 2>/dev/null || echo "$SCRIPT_DIR")"
DATA_DIR="$PWD/.aether/data"

# Initialize lock state before sourcing (file-lock.sh trap needs these)
LOCK_ACQUIRED=${LOCK_ACQUIRED:-false}
CURRENT_LOCK=${CURRENT_LOCK:-""}

# Source shared infrastructure if available
[[ -f "$SCRIPT_DIR/utils/file-lock.sh" ]] && source "$SCRIPT_DIR/utils/file-lock.sh"
[[ -f "$SCRIPT_DIR/utils/atomic-write.sh" ]] && source "$SCRIPT_DIR/utils/atomic-write.sh"

# Fallback atomic_write if not sourced
if ! type atomic_write &>/dev/null; then
  atomic_write() { echo "$2" > "$1"; }
fi

# --- JSON output helpers ---
# Success: JSON to stdout, exit 0
json_ok() { printf '{"ok":true,"result":%s}\n' "$1"; }

# Error: JSON to stderr, exit 1
json_err() { printf '{"ok":false,"error":"%s"}\n' "$1" >&2; exit 1; }

# --- Subcommand dispatch ---
cmd="${1:-help}"
shift 2>/dev/null || true

case "$cmd" in
  help)
    cat <<'EOF'
{"ok":true,"commands":["help","version","validate-state","error-add","error-pattern-check","error-summary","activity-log","activity-log-init","activity-log-read","learning-promote","learning-inject","generate-ant-name","spawn-log","spawn-complete","update-progress","check-antipattern","error-flag-pattern"],"description":"Aether Colony Utility Layer — deterministic ops for the ant colony"}
EOF
    ;;
  version)
    json_ok '"1.0.0"'
    ;;
  validate-state)
    case "${1:-}" in
      colony)
        [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "COLONY_STATE.json not found"
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
        [[ -f "$DATA_DIR/constraints.json" ]] || json_err "constraints.json not found"
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
        json_err "Usage: validate-state colony|constraints|all"
        ;;
    esac
    ;;
  error-add)
    [[ $# -ge 3 ]] || json_err "Usage: error-add <category> <severity> <description> [phase]"
    [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "COLONY_STATE.json not found"
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
    ' "$DATA_DIR/COLONY_STATE.json") || json_err "Failed to update COLONY_STATE.json"
    atomic_write "$DATA_DIR/COLONY_STATE.json" "$updated"
    json_ok "\"$id\""
    ;;
  error-pattern-check)
    [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "COLONY_STATE.json not found"
    json_ok "$(jq '
      .errors.records | group_by(.category) | map(select(length >= 3) |
        {category: .[0].category, count: length,
         first_seen: (sort_by(.timestamp) | first.timestamp),
         last_seen: (sort_by(.timestamp) | last.timestamp)})
    ' "$DATA_DIR/COLONY_STATE.json")"
    ;;
  error-summary)
    [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "COLONY_STATE.json not found"
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
    [[ -z "$action" || -z "$caste" || -z "$description" ]] && json_err "Usage: activity-log <action> <caste_or_name> <description>"
    log_file="$DATA_DIR/activity.log"
    mkdir -p "$DATA_DIR"
    ts=$(date -u +"%H:%M:%S")
    echo "[$ts] $action $caste: $description" >> "$log_file"
    json_ok '"logged"'
    ;;
  activity-log-init)
    phase_num="${1:-}"
    phase_name="${2:-}"
    [[ -z "$phase_num" ]] && json_err "Usage: activity-log-init <phase_num> [phase_name]"
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
    echo "# Phase $phase_num: ${phase_name:-unnamed} -- $ts" >> "$log_file"
    archived_flag="false"
    [ -f "$archive_file" ] && archived_flag="true"
    json_ok "{\"archived\":$archived_flag}"
    ;;
  activity-log-read)
    caste_filter="${1:-}"
    log_file="$DATA_DIR/activity.log"
    [[ -f "$log_file" ]] || json_err "activity.log not found"
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

    global_dir="$HOME/.aether"
    global_file="$global_dir/learnings.json"
    mkdir -p "$global_dir"

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
      --argjson phase "$source_phase" --argjson tags "$tags_json" --arg ts "$ts" '
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

    global_file="$HOME/.aether/learnings.json"

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
    # Usage: spawn-log <parent_id> <child_caste> <child_name> <task_summary>
    parent_id="${1:-}"
    child_caste="${2:-}"
    child_name="${3:-}"
    task_summary="${4:-}"
    [[ -z "$parent_id" || -z "$child_caste" || -z "$task_summary" ]] && json_err "Usage: spawn-log <parent_id> <child_caste> <child_name> <task_summary>"
    mkdir -p "$DATA_DIR"
    ts=$(date -u +"%H:%M:%S")
    ts_full=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    # Log to activity log with spawn format
    echo "[$ts] SPAWN $parent_id -> $child_name ($child_caste): $task_summary" >> "$DATA_DIR/activity.log"
    # Log to spawn tree file for visualization
    echo "$ts_full|$parent_id|$child_caste|$child_name|$task_summary|spawned" >> "$DATA_DIR/spawn-tree.txt"
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
    echo "[$ts] COMPLETE $ant_name: $status${summary:+ - $summary}" >> "$DATA_DIR/activity.log"
    # Update spawn tree
    echo "$ts_full|$ant_name|$status|$summary" >> "$DATA_DIR/spawn-tree.txt"
    json_ok '"logged"'
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

    global_dir="$HOME/.aether"
    patterns_file="$global_dir/error-patterns.json"
    mkdir -p "$global_dir"

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
    global_file="$HOME/.aether/error-patterns.json"

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
      *)        prefixes=("Ant" "Worker" "Drone" "Toiler" "Marcher" "Runner" "Carrier" "Helper") ;;
    esac
    # Pick random prefix and add random number
    idx=$((RANDOM % ${#prefixes[@]}))
    prefix="${prefixes[$idx]}"
    num=$((RANDOM % 99 + 1))
    name="${prefix}-${num}"
    json_ok "\"$name\""
    ;;
  *)
    json_err "Unknown command: $cmd"
    ;;
esac
