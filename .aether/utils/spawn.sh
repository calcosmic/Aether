#!/bin/bash
# Spawn utility functions — extracted from aether-utils.sh
# Provides: _spawn_log, _spawn_complete, _spawn_can_spawn, _spawn_get_depth, _spawn_can_spawn_swarm, _spawn_tree_load, _spawn_tree_active, _spawn_tree_depth, _spawn_efficiency
#
# These functions are sourced by aether-utils.sh at startup.
# All shared infrastructure (json_ok, json_err, json_warn, atomic_write, acquire_lock,
# release_lock, feature_enabled, LOCK_DIR, DATA_DIR, SCRIPT_DIR, error constants) is available.
# Note: get_caste_emoji is defined in the main file and available to this module at call time.

_spawn_log() {
    # Usage: spawn-log <parent_id> <child_caste> <child_name> <task_summary> [model] [status]
    parent_id="${1:-}"
    child_caste="${2:-}"
    child_name="${3:-}"
    task_summary="${4:-}"
    model="${5:-default}"
    status="${6:-spawned}"
    # Auto-resolve model slot from caste if not explicitly provided
    if [[ "$model" == "default" ]]; then
      slot=$(bash "$0" model-slot get "$child_caste" 2>/dev/null | jq -r '.result // "inherit"')
      [[ -n "$slot" && "$slot" != "null" ]] && model="$slot"
    fi
    [[ -z "$parent_id" || -z "$child_caste" || -z "$task_summary" ]] && json_err "$E_VALIDATION_FAILED" "Usage: spawn-log <parent_id> <child_caste> <child_name> <task_summary> [model] [status]"
    mkdir -p "$DATA_DIR"
    ts=$(date -u +"%H:%M:%S")
    ts_full=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    emoji=$(get_caste_emoji "$child_caste")
    parent_emoji=$(get_caste_emoji "$parent_id")
    # Log to activity log with spawn format, emojis, and model info
    echo "[$ts] ⚡ SPAWN $parent_emoji $parent_id -> $emoji $child_name ($child_caste): $task_summary [model: $model]" >> "$DATA_DIR/activity.log"
    # Log to spawn tree file for visualization (NEW FORMAT: includes model field)
    echo "$ts_full|$parent_id|$child_caste|$child_name|$task_summary|$model|$status" >> "$DATA_DIR/spawn-tree.txt"
    # Return emoji-formatted result for display
    json_ok "\"⚡ $emoji $child_name spawned\""
}

_spawn_complete() {
    # Migrated to state-api facade: uses _state_mutate for failed spawn event logging
    # Usage: spawn-complete <ant_name> <status> [summary]
    ant_name="${1:-}"
    status="${2:-completed}"
    summary="${3:-}"
    [[ -z "$ant_name" ]] && json_err "$E_VALIDATION_FAILED" "Usage: spawn-complete <ant_name> <status> [summary]"
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
    # Log failed spawns to events array as pipe-delimited strings (matching template format)
    if [[ "$status" == "failed" ]] || [[ "$status" == "error" ]]; then
      if [[ -f "$DATA_DIR/COLONY_STATE.json" ]]; then
        SC_EVENT="$ts_full|spawn_failed|$ant_name|${summary:-unknown}" \
          _state_mutate '
            .events += [env.SC_EVENT]
          ' >/dev/null 2>&1 || _aether_log_error "Failed to log spawn failure to colony state"
      fi
    fi
    # Return emoji-formatted result for display
    json_ok "\"$status_icon $emoji $ant_name: ${summary:-$status}\""
}

_spawn_can_spawn() {
    # Check if spawning is allowed at given depth
    # Usage: spawn-can-spawn [depth] [--enforce]
    # Returns: {can_spawn: bool, depth: N, max_spawns: N, current_total: N, global_cap: N}
    # --enforce: fail with non-zero exit when spawning is not allowed
    depth=""
    enforce_mode=false
    for arg in "$@"; do
      case "$arg" in
        --enforce) enforce_mode=true ;;
        *)
          if [[ -z "$depth" ]]; then
            depth="$arg"
          else
            json_err "$E_VALIDATION_FAILED" "Usage: spawn-can-spawn [depth] [--enforce]"
          fi
          ;;
      esac
    done
    [[ -z "$depth" ]] && depth=1
    [[ "$depth" =~ ^[0-9]+$ ]] || json_err "$E_VALIDATION_FAILED" "Depth must be a non-negative integer" "{\"provided\":\"$depth\"}"

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
      current=$(grep -c "|spawned$" "$DATA_DIR/spawn-tree.txt" 2>/dev/null || echo 0)  # SUPPRESS:OK -- read-default: count defaults to 0 if file missing
    fi

    # Global cap of 10 workers per phase
    global_cap=10

    # Can spawn if: depth < 3 AND under global cap
    if [[ $depth -lt 3 && $current -lt $global_cap ]]; then
      can="true"
    else
      can="false"
    fi

    if [[ "$enforce_mode" == "true" && "$can" == "false" ]]; then
      json_err "$E_VALIDATION_FAILED" "Spawn cap exceeded: depth=$depth current=$current max=$global_cap"
    fi

    json_ok "{\"can_spawn\":$can,\"depth\":$depth,\"max_spawns\":$max_for_depth,\"current_total\":$current,\"global_cap\":$global_cap}"
}

_spawn_get_depth() {
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
    if ! grep -q "|$ant_name|" "$DATA_DIR/spawn-tree.txt" 2>/dev/null; then  # SUPPRESS:OK -- existence-test: file may not exist
      json_ok "{\"ant\":\"$ant_name\",\"depth\":1,\"found\":false}"
      exit 0
    fi

    # Find the spawn record for this ant and trace parents
    depth=1
    current_ant="$ant_name"

    # Find who spawned this ant (look for lines with |spawned)
    while true; do
      # Format: timestamp|parent|caste|child_name|task|spawned
      # SUPPRESS:OK -- read-default: returns fallback on failure
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
}

_spawn_can_spawn_swarm() {
    # Check if swarm can spawn more scouts (separate from phase workers)
    # Usage: spawn-can-spawn-swarm <swarm_id>
    # Swarm has its own cap of 6 (4 scouts + 2 sub-scouts max)
    swarm_id="${1:-swarm}"
    swarm_cap=6

    current=0
    if [[ -f "$DATA_DIR/spawn-tree.txt" ]]; then
      # SUPPRESS:OK -- existence-test: grep returns 1 when no matches
      current=$(grep -c "|swarm:$swarm_id$" "$DATA_DIR/spawn-tree.txt" 2>/dev/null) || current=0
    fi

    if [[ $current -lt $swarm_cap ]]; then
      can="true"
      remaining=$((swarm_cap - current))
    else
      can="false"
      remaining=0
    fi

    json_ok "{\"can_spawn\":$can,\"current\":$current,\"cap\":$swarm_cap,\"remaining\":$remaining,\"swarm_id\":\"$swarm_id\"}"
}

_spawn_tree_load() {
    source "$SCRIPT_DIR/utils/spawn-tree.sh" 2>/dev/null || {  # SUPPRESS:OK -- read-default: utility may not be installed
      json_err "$E_FILE_NOT_FOUND" "spawn-tree.sh not found"
      exit 1
    }
    tree_json=$(reconstruct_tree_json)
    if echo "$tree_json" | jq -e . >/dev/null 2>&1; then
      json_ok "$tree_json"
    else
      json_err "$E_VALIDATION_FAILED" "spawn tree reconstruction produced invalid JSON"
      return 1
    fi
}

_spawn_tree_active() {
    source "$SCRIPT_DIR/utils/spawn-tree.sh" 2>/dev/null || {  # SUPPRESS:OK -- read-default: utility may not be installed
      json_err "$E_FILE_NOT_FOUND" "spawn-tree.sh not found"
      exit 1
    }
    active=$(get_active_spawns)
    json_ok "$active"
}

_spawn_tree_depth() {
    ant_name="${1:-}"
    [[ -z "$ant_name" ]] && json_err "$E_VALIDATION_FAILED" "Usage: spawn-tree-depth <ant_name>"
    source "$SCRIPT_DIR/utils/spawn-tree.sh" 2>/dev/null || {  # SUPPRESS:OK -- read-default: utility may not be installed
      json_err "$E_FILE_NOT_FOUND" "spawn-tree.sh not found"
      exit 1
    }
    depth=$(get_spawn_depth "$ant_name")
    json_ok "$depth"
}

_spawn_efficiency() {
    # Calculate spawn efficiency metrics from spawn-tree.txt
    # Usage: spawn-efficiency
    spawn_tree_file="$DATA_DIR/spawn-tree.txt"
    total=0
    completed=0
    failed=0

    if [[ -f "$spawn_tree_file" ]]; then
      total=$(grep -c "|spawned$" "$spawn_tree_file" 2>/dev/null || echo 0)  # SUPPRESS:OK -- read-default: count defaults to 0 if file missing
      completed=$(grep -c "|completed$" "$spawn_tree_file" 2>/dev/null || echo 0)  # SUPPRESS:OK -- read-default: count defaults to 0 if file missing
      failed=$(grep -c "|failed$" "$spawn_tree_file" 2>/dev/null || echo 0)  # SUPPRESS:OK -- read-default: count defaults to 0 if file missing
    fi

    if [[ "$total" -gt 0 ]]; then
      efficiency=$(( completed * 100 / total ))
    else
      efficiency=0
    fi

    json_ok "{\"total\":$total,\"completed\":$completed,\"failed\":$failed,\"efficiency_pct\":$efficiency}"
}
