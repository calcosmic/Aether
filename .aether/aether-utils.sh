#!/bin/bash
# Aether Colony Utility Layer
# Single entry point for deterministic colony operations
#
# Usage: bash .aether/aether-utils.sh <subcommand> [args...]
#
# All subcommands output JSON to stdout.
# Non-zero exit on error with JSON error message to stderr.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$SCRIPT_DIR/.." && pwd 2>/dev/null || echo "$SCRIPT_DIR")"
DATA_DIR="$SCRIPT_DIR/data"

# Initialize lock state before sourcing (file-lock.sh trap needs these)
LOCK_ACQUIRED=${LOCK_ACQUIRED:-false}
CURRENT_LOCK=${CURRENT_LOCK:-""}

# Source shared infrastructure
source "$SCRIPT_DIR/utils/file-lock.sh"
source "$SCRIPT_DIR/utils/atomic-write.sh"

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
{"ok":true,"commands":["help","version","pheromone-decay","pheromone-effective","pheromone-batch","pheromone-cleanup","pheromone-validate","validate-state","spawn-check","memory-compress","error-add","error-pattern-check","error-summary","activity-log","activity-log-init","activity-log-read","learning-promote","learning-inject"],"description":"Aether Colony Utility Layer — deterministic ops for the ant colony"}
EOF
    ;;
  version)
    json_ok '"0.1.0"'
    ;;
  pheromone-decay)
    [[ $# -ge 3 ]] || json_err "Usage: pheromone-decay <strength> <elapsed_seconds> <half_life>"
    json_ok "$(jq -n --arg s "$1" --arg e "$2" --arg h "$3" '
      ($s|tonumber) as $strength |
      ([$e|tonumber, 0] | max) as $elapsed |
      ($h|tonumber) as $half_life |
      if $elapsed > ($half_life * 10) then
        {strength: 0}
      else
        ($strength * ((-0.693147180559945 * $elapsed / $half_life) | exp)) as $decayed |
        {strength: ([$decayed, $strength] | min | . * 1000000 | round / 1000000)}
      end
    ')"
    ;;
  pheromone-effective)
    [[ $# -ge 2 ]] || json_err "Usage: pheromone-effective <sensitivity> <strength>"
    json_ok "$(jq -n --arg sens "$1" --arg str "$2" \
      '{effective_signal: (($sens|tonumber) * ($str|tonumber) | . * 1000000 | round / 1000000)}')"
    ;;
  pheromone-batch)
    [[ -f "$DATA_DIR/pheromones.json" ]] || json_err "pheromones.json not found"
    now=$(date -u +%s)
    json_ok "$(jq --arg now "$now" '.signals | map(. + {
      current_strength: (
        if .half_life_seconds == null then .strength
        else
          (($now|tonumber) - (.created_at | sub("\\.[0-9]+Z$";"Z") | fromdate)) as $elapsed |
          if $elapsed < 0 then .strength
          elif $elapsed > (.half_life_seconds * 10) then 0
          else
            (.strength * ((-0.693147180559945 * $elapsed / .half_life_seconds) | exp)) as $d |
            if $d > .strength then .strength else $d end
          end
        end | . * 1000 | round / 1000)
    })' "$DATA_DIR/pheromones.json")" || json_err "Failed to read pheromones.json"
    ;;
  pheromone-cleanup)
    [[ -f "$DATA_DIR/pheromones.json" ]] || json_err "pheromones.json not found"
    now=$(date -u +%s)
    before=$(jq '.signals | length' "$DATA_DIR/pheromones.json")
    result=$(jq --arg now "$now" '.signals |= map(select(
      .half_life_seconds == null or
      (
        (($now|tonumber) - (.created_at | sub("\\.[0-9]+Z$";"Z") | fromdate)) as $elapsed |
        if $elapsed < 0 then true
        elif $elapsed > (.half_life_seconds * 10) then false
        else
          (.strength * ((-0.693147180559945 * $elapsed / .half_life_seconds) | exp)) >= 0.05
        end
      )
    ))' "$DATA_DIR/pheromones.json") || json_err "Failed to process pheromones.json"
    atomic_write "$DATA_DIR/pheromones.json" "$result"
    after=$(echo "$result" | jq '.signals | length')
    json_ok "{\"removed\":$((before - after)),\"remaining\":$after}"
    ;;
  pheromone-validate)
    content="${1:-}"
    len=${#content}
    if [[ -z "$content" ]]; then
      json_ok '{"pass":false,"reason":"empty","length":0,"min_length":20}'
    elif [[ $len -lt 20 ]]; then
      json_ok "{\"pass\":false,\"reason\":\"too_short\",\"length\":$len,\"min_length\":20}"
    else
      json_ok "{\"pass\":true,\"length\":$len,\"min_length\":20}"
    fi
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
            chk("workers";["object"]),
            chk("spawn_outcomes";["object"]),
            opt("spawn_tree";["object"])
          ]} | . + {pass: ([.checks[] | select(. == "pass")] | length) == (.checks | length)}
        ' "$DATA_DIR/COLONY_STATE.json")"
        ;;
      pheromones)
        [[ -f "$DATA_DIR/pheromones.json" ]] || json_err "pheromones.json not found"
        json_ok "$(jq '
          def arr(f): if has(f) and (.[f]|type) == "array" then "pass" else "fail: \(f) not array" end;
          def sig: ["id","type","content","strength","created_at"] as $req | [. as $s | $req[] | select($s[.] == null)] |
            if length == 0 then "pass" else "fail: signal missing \(join(","))" end;
          {file:"pheromones.json", checks:[
            arr("signals"),
            (.signals | if length == 0 then "pass" else [.[] | sig] | map(select(. != "pass")) |
              if length == 0 then "pass" else .[0] end end)
          ]} | . + {pass: ([.checks[] | select(. == "pass")] | length) == (.checks | length)}
        ' "$DATA_DIR/pheromones.json")"
        ;;
      errors)
        [[ -f "$DATA_DIR/errors.json" ]] || json_err "errors.json not found"
        json_ok "$(jq '
          def arr(f): if has(f) and (.[f]|type) == "array" then "pass" else "fail: \(f) not array" end;
          def erchk: ["id","category","severity","description","timestamp"] as $req | [. as $e | $req[] | select($e[.] == null)] |
            if length == 0 then "pass" else "fail: error missing \(join(","))" end;
          {file:"errors.json", checks:[
            arr("errors"), arr("flagged_patterns"),
            (.errors | if length == 0 then "pass" else [.[] | erchk] | map(select(. != "pass")) |
              if length == 0 then "pass" else .[0] end end)
          ]} | . + {pass: ([.checks[] | select(. == "pass")] | length) == (.checks | length)}
        ' "$DATA_DIR/errors.json")"
        ;;
      memory)
        [[ -f "$DATA_DIR/memory.json" ]] || json_err "memory.json not found"
        json_ok "$(jq '
          def arr(f): if has(f) and (.[f]|type) == "array" then "pass" else "fail: \(f) not array" end;
          {file:"memory.json", checks:[arr("phase_learnings"), arr("decisions"), arr("patterns")]}
          | . + {pass: ([.checks[] | select(. == "pass")] | length) == (.checks | length)}
        ' "$DATA_DIR/memory.json")"
        ;;
      events)
        [[ -f "$DATA_DIR/events.json" ]] || json_err "events.json not found"
        json_ok "$(jq '
          def arr(f): if has(f) and (.[f]|type) == "array" then "pass" else "fail: \(f) not array" end;
          def evchk: ["id","type","source","content","timestamp"] as $req | [. as $e | $req[] | select($e[.] == null)] |
            if length == 0 then "pass" else "fail: event missing \(join(","))" end;
          {file:"events.json", checks:[
            arr("events"),
            (.events | if length == 0 then "pass" else [.[] | evchk] | map(select(. != "pass")) |
              if length == 0 then "pass" else .[0] end end)
          ]} | . + {pass: ([.checks[] | select(. == "pass")] | length) == (.checks | length)}
        ' "$DATA_DIR/events.json")"
        ;;
      all)
        results=()
        for target in colony pheromones errors memory events; do
          results+=("$(bash "$SCRIPT_DIR/aether-utils.sh" validate-state "$target" 2>/dev/null || echo '{"ok":false}')")
        done
        combined=$(printf '%s\n' "${results[@]}" | jq -s '[.[] | .result // {file:"unknown",pass:false}]')
        all_pass=$(echo "$combined" | jq 'all(.pass)')
        json_ok "{\"pass\":$all_pass,\"files\":$combined}"
        ;;
      *)
        json_err "Usage: validate-state colony|pheromones|errors|memory|events|all"
        ;;
    esac
    ;;
  memory-compress)
    [[ -f "$DATA_DIR/memory.json" ]] || json_err "memory.json not found"
    threshold="${1:-10000}"
    result=$(jq --arg th "$threshold" '
      .phase_learnings |= (if length > 20 then .[-20:] else . end) |
      .decisions |= (if length > 30 then .[-30:] else . end) |
      . as $trimmed |
      ([.. | strings] | join(" ") | split(" ") | length | . * 1.3 | floor) as $tokens |
      if $tokens > ($th|tonumber) then
        .phase_learnings |= (if length > 10 then .[-10:] else . end) |
        .decisions |= (if length > 15 then .[-15:] else . end)
      else . end
    ' "$DATA_DIR/memory.json") || json_err "Failed to process memory.json"
    atomic_write "$DATA_DIR/memory.json" "$result"
    tokens=$(echo "$result" | jq '[.. | strings] | join(" ") | split(" ") | length | . * 1.3 | floor')
    json_ok "{\"compressed\":true,\"tokens\":$tokens}"
    ;;
  error-add)
    [[ $# -ge 3 ]] || json_err "Usage: error-add <category> <severity> <description> [phase]"
    [[ -f "$DATA_DIR/errors.json" ]] || json_err "errors.json not found"
    id="err_$(date -u +%s)_$(head -c 2 /dev/urandom | od -An -tx1 | tr -d ' ')"
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    phase_val="${4:-null}"
    if [[ "$phase_val" =~ ^[0-9]+$ ]]; then
      phase_jq="$phase_val"
    else
      phase_jq="null"
    fi
    updated=$(jq --arg id "$id" --arg cat "$1" --arg sev "$2" --arg desc "$3" --argjson phase "$phase_jq" --arg ts "$ts" '
      .errors += [{id:$id, category:$cat, severity:$sev, description:$desc, root_cause:null, phase:$phase, task_id:null, timestamp:$ts}] |
      if (.errors|length) > 50 then .errors = .errors[-50:] else . end
    ' "$DATA_DIR/errors.json") || json_err "Failed to update errors.json"
    atomic_write "$DATA_DIR/errors.json" "$updated"
    json_ok "\"$id\""
    ;;
  error-pattern-check)
    [[ -f "$DATA_DIR/errors.json" ]] || json_err "errors.json not found"
    json_ok "$(jq '
      .errors | group_by(.category) | map(select(length >= 3) |
        {category: .[0].category, count: length,
         first_seen: (sort_by(.timestamp) | first.timestamp),
         last_seen: (sort_by(.timestamp) | last.timestamp)})
    ' "$DATA_DIR/errors.json")"
    ;;
  error-summary)
    [[ -f "$DATA_DIR/errors.json" ]] || json_err "errors.json not found"
    json_ok "$(jq '{
      total: (.errors | length),
      by_category: (.errors | group_by(.category) | map({key: .[0].category, value: length}) | from_entries),
      by_severity: (.errors | group_by(.severity) | map({key: .[0].severity, value: length}) | from_entries)
    }' "$DATA_DIR/errors.json")"
    ;;
  spawn-check)
    depth="${1:-1}"
    [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "COLONY_STATE.json not found"
    json_ok "$(jq --arg d "$depth" '
      (.workers | to_entries | map(select(.value != "idle")) | length) as $active |
      ($d | tonumber) as $depth |
      {
        pass: ($active < 5 and $depth < 2),
        active_workers: $active,
        max_workers: 5,
        current_depth: $depth,
        max_depth: 2
      } | if .pass == false then
        . + {reason: (if $active >= 5 then "worker_limit" elif $depth >= 2 then "depth_limit" else "unknown" end)}
      else . end
    ' "$DATA_DIR/COLONY_STATE.json")"
    ;;
  activity-log)
    action="${1:-}"
    caste="${2:-}"
    description="${3:-}"
    [[ -z "$action" || -z "$caste" || -z "$description" ]] && json_err "Usage: activity-log <action> <caste> <description>"
    log_file="$DATA_DIR/activity.log"
    ts=$(date -u +"%H:%M:%S")
    echo "[$ts] $action $caste: $description" >> "$log_file"
    json_ok '"logged"'
    ;;
  activity-log-init)
    phase_num="${1:-}"
    phase_name="${2:-}"
    [[ -z "$phase_num" ]] && json_err "Usage: activity-log-init <phase_num> [phase_name]"
    log_file="$DATA_DIR/activity.log"
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
  *)
    json_err "Unknown command: $cmd"
    ;;
esac
