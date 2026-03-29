#!/bin/bash
# Spawn Tree Reconstruction Module
# Parses spawn-tree.txt and provides tree traversal functions
#
# Usage: source .aether/utils/spawn-tree.sh
# All functions output JSON to stdout

# Data directory - can be overridden
SPAWN_TREE_DATA_DIR="${SPAWN_TREE_DATA_DIR:-.aether/data}"
SPAWN_TREE_FILE="${SPAWN_TREE_FILE:-$SPAWN_TREE_DATA_DIR/spawn-tree.txt}"

# Parse spawn-tree.txt into structured data
# Usage: parse_spawn_tree [file_path]
# Outputs: JSON representation of all spawns
parse_spawn_tree() {
  local file_path="${1:-$SPAWN_TREE_FILE}"

  if [[ ! -f "$file_path" ]] || [[ ! -s "$file_path" ]]; then
    echo '{"spawns":[],"metadata":{"total_count":0,"active_count":0,"completed_count":0,"file_exists":false}}'
    return 0
  fi

  awk -F'|' '
  BEGIN { n=0; active=0; completed_n=0 }
  NF == 7 && $7 == "spawned" {
    names[n] = $4; parents[n] = $2; castes[n] = $3
    tasks[n] = $5; statuses[n] = "spawned"; timestamps[n] = $1
    models[n] = $6; completed_at[n] = ""; children_str[n] = ""
    name_to_idx[$4] = n; n++
  }
  $3 ~ /^(completed|failed|blocked)$/ && NF >= 4 {
    ant = $2
    if (ant in name_to_idx) {
      idx = name_to_idx[ant]; statuses[idx] = $3; completed_at[idx] = $1
    }
  }
  END {
    for (i = 0; i < n; i++) {
      p = parents[i]
      if (p in name_to_idx) {
        pidx = name_to_idx[p]
        if (children_str[pidx] == "") children_str[pidx] = i
        else children_str[pidx] = children_str[pidx] " " i
      }
    }
    for (i = 0; i < n; i++) {
      if (statuses[i] == "spawned" || statuses[i] == "active") active++
      else if (statuses[i] ~ /^(completed|failed|blocked)$/) completed_n++
    }
    printf "{"
    printf "\"spawns\":["
    for (i = 0; i < n; i++) {
      if (i > 0) printf ","
      nm = names[i]; gsub(/\\/, "\\\\", nm); gsub(/"/, "\\\"", nm); gsub(/\t/, "\\t", nm)
      pr = parents[i]; gsub(/\\/, "\\\\", pr); gsub(/"/, "\\\"", pr); gsub(/\t/, "\\t", pr)
      tk = tasks[i]; gsub(/\\/, "\\\\", tk); gsub(/"/, "\\\"", tk); gsub(/\t/, "\\t", tk)
      printf "{\"name\":\"%s\",\"parent\":\"%s\",\"caste\":\"%s\",", nm, pr, castes[i]
      printf "\"task\":\"%s\",\"status\":\"%s\",", tk, statuses[i]
      printf "\"spawned_at\":\"%s\",\"completed_at\":\"%s\",", timestamps[i], completed_at[i]
      printf "\"children\":["
      if (children_str[i] != "") {
        split(children_str[i], cidxs, " ")
        for (j = 1; j <= length(cidxs); j++) {
          if (j > 1) printf ","
          cn = names[cidxs[j]+0]
          gsub(/\\/, "\\\\", cn); gsub(/"/, "\\\"", cn); gsub(/\t/, "\\t", cn)
          printf "\"%s\"", cn
        }
      }
      printf "]}"
    }
    printf "],"
    printf "\"metadata\":{\"total_count\":%d,\"active_count\":%d,\"completed_count\":%d,\"file_exists\":true}", n, active, completed_n
    printf "}"
  }
  ' "$file_path"
}

# Get spawn depth for a given ant name
# Usage: get_spawn_depth <ant_name>
# Returns: JSON with ant name and depth
get_spawn_depth() {
  local ant_name="${1:-}"

  if [[ -z "$ant_name" || "$ant_name" == "Queen" ]]; then
    echo "{\"ant\":\"${ant_name:-Queen}\",\"depth\":0}"
    return 0
  fi

  local file_path="${SPAWN_TREE_FILE}"

  if [[ ! -f "$file_path" ]]; then
    echo "{\"ant\":\"$ant_name\",\"depth\":1,\"found\":false}"
    return 0
  fi

  # Check if ant exists
  if ! grep -q "|$ant_name|" "$file_path" 2>/dev/null; then
    echo "{\"ant\":\"$ant_name\",\"depth\":1,\"found\":false}"
    return 0
  fi

  # Calculate depth by traversing parent chain
  local depth=1
  local current="$ant_name"
  local safety=0

  while [[ $safety -lt 5 ]]; do
    # Find who spawned this ant
    local parent
    parent=$(grep "|$current|" "$file_path" 2>/dev/null | grep "|spawned$" | head -1 | cut -d'|' -f2 || echo "")

    if [[ -z "$parent" || "$parent" == "Queen" ]]; then
      break
    fi

    ((depth++))
    current="$parent"
    ((safety++))
  done

  echo "{\"ant\":\"$ant_name\",\"depth\":$depth,\"found\":true}"
}

# Get list of active spawns
# Usage: get_active_spawns [file_path]
# Returns: JSON array of active spawns
get_active_spawns() {
  local file_path="${1:-$SPAWN_TREE_FILE}"

  if [[ ! -f "$file_path" ]] || [[ ! -s "$file_path" ]]; then
    echo "[]"
    return 0
  fi

  awk -F'|' '
  BEGIN { spawn_n=0 }
  $3 ~ /^(completed|failed|blocked)$/ && NF >= 4 { done_set[$2] = 1 }
  NF == 7 && $7 == "spawned" {
    spawn_names[spawn_n] = $4; spawn_parents[spawn_n] = $2
    spawn_castes[spawn_n] = $3; spawn_tasks[spawn_n] = $5
    spawn_ts[spawn_n] = $1; spawn_n++
  }
  END {
    printf "["
    first = 1
    for (i = 0; i < spawn_n; i++) {
      if (!(spawn_names[i] in done_set)) {
        if (!first) printf ","
        first = 0
        nm = spawn_names[i]; gsub(/\\/, "\\\\", nm); gsub(/"/, "\\\"", nm); gsub(/\t/, "\\t", nm)
        pr = spawn_parents[i]; gsub(/\\/, "\\\\", pr); gsub(/"/, "\\\"", pr); gsub(/\t/, "\\t", pr)
        tk = spawn_tasks[i]; gsub(/\\/, "\\\\", tk); gsub(/"/, "\\\"", tk); gsub(/\t/, "\\t", tk)
        printf "{\"name\":\"%s\",\"caste\":\"%s\",\"parent\":\"%s\",\"task\":\"%s\",\"spawned_at\":\"%s\"}", nm, spawn_castes[i], pr, tk, spawn_ts[i]
      }
    }
    printf "]"
  }
  ' "$file_path"
}

# Get direct children of a spawn
# Usage: get_spawn_children <ant_name> [file_path]
# Returns: JSON array of child names
get_spawn_children() {
  local ant_name="${1:-}"
  local file_path="${2:-$SPAWN_TREE_FILE}"

  if [[ -z "$ant_name" || ! -f "$file_path" ]]; then
    echo "[]"
    return 0
  fi

  local children_json="["
  local first=true

  # Find all spawns where parent matches
  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ -z "$line" ]] && continue

    local pipe_count
    pipe_count=$(echo "$line" | tr -cd '|' | wc -c | tr -d ' ')

    if [[ $pipe_count -eq 6 ]]; then
      local parent child_name
      parent=$(echo "$line" | cut -d'|' -f2)
      child_name=$(echo "$line" | cut -d'|' -f4)

      if [[ "$parent" == "$ant_name" ]]; then
        if [[ "$first" == "true" ]]; then
          first=false
        else
          children_json+=","
        fi
        children_json+="\"$child_name\""
      fi
    fi
  done < "$file_path"

  children_json+="]"
  echo "$children_json"
}

# Get full lineage from ant up to Queen
# Usage: get_spawn_lineage <ant_name> [file_path]
# Returns: JSON array from ant up to Queen (inclusive)
get_spawn_lineage() {
  local ant_name="${1:-}"
  local file_path="${2:-$SPAWN_TREE_FILE}"

  if [[ -z "$ant_name" ]]; then
    echo "[]"
    return 0
  fi

  if [[ ! -f "$file_path" ]]; then
    echo "[\"$ant_name\",\"Queen\"]"
    return 0
  fi

  # Build lineage array (ant first, then ancestors)
  local lineage=""
  local current="$ant_name"
  local safety=0

  lineage="\"$current\""

  while [[ $safety -lt 5 ]]; do
    # Find who spawned this ant
    local parent
    parent=$(grep "|$current|" "$file_path" 2>/dev/null | grep "|spawned$" | head -1 | cut -d'|' -f2 || echo "")

    if [[ -z "$parent" || "$parent" == "Queen" ]]; then
      lineage+=",\"Queen\""
      break
    fi

    lineage+=",\"$parent\""
    current="$parent"
    ((safety++))
  done

  echo "[$lineage]"
}

# Reconstruct full tree as JSON
# Usage: reconstruct_tree_json [file_path]
# Returns: Complete spawn tree with metadata
reconstruct_tree_json() {
  local file_path="${1:-$SPAWN_TREE_FILE}"
  parse_spawn_tree "$file_path"
}

# Export functions if being sourced (Bash 3.2 compatible)
if [[ "${BASH_SOURCE[0]:-}" != "${0}" ]]; then
  export -f parse_spawn_tree 2>/dev/null || true
  export -f get_spawn_depth 2>/dev/null || true
  export -f get_active_spawns 2>/dev/null || true
  export -f get_spawn_children 2>/dev/null || true
  export -f get_spawn_lineage 2>/dev/null || true
  export -f reconstruct_tree_json 2>/dev/null || true
fi
