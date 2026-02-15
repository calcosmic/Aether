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

  # Check if file exists
  if [[ ! -f "$file_path" ]]; then
    echo '{"spawns":[],"metadata":{"total_count":0,"active_count":0,"completed_count":0,"file_exists":false}}'
    return 0
  fi

  # Temporary files for data storage (Bash 3.2 compatible)
  local tmpdir
  tmpdir=$(mktemp -d)
  local names_file="$tmpdir/names"
  local parents_file="$tmpdir/parents"
  local castes_file="$tmpdir/castes"
  local tasks_file="$tmpdir/tasks"
  local statuses_file="$tmpdir/statuses"
  local timestamps_file="$tmpdir/timestamps"
  local completed_file="$tmpdir/completed"
  local children_file="$tmpdir/children"

  touch "$names_file" "$parents_file" "$castes_file" "$tasks_file" "$statuses_file" "$timestamps_file" "$completed_file" "$children_file"

  # Read file line by line
  while IFS= read -r line || [[ -n "$line" ]]; do
    # Skip empty lines
    [[ -z "$line" ]] && continue

    # Count pipe separators to determine line type
    local pipe_count
    pipe_count=$(echo "$line" | tr -cd '|' | wc -c | tr -d ' ')

    if [[ $pipe_count -eq 5 ]]; then
      # Spawn event: timestamp|parent|caste|child_name|task|spawned
      local timestamp parent caste child_name task spawn_status
      timestamp=$(echo "$line" | cut -d'|' -f1)
      parent=$(echo "$line" | cut -d'|' -f2)
      caste=$(echo "$line" | cut -d'|' -f3)
      child_name=$(echo "$line" | cut -d'|' -f4)
      task=$(echo "$line" | cut -d'|' -f5)
      spawn_status=$(echo "$line" | cut -d'|' -f6)

      # Add to files
      echo "$child_name" >> "$names_file"
      echo "$parent" >> "$parents_file"
      echo "$caste" >> "$castes_file"
      echo "$task" >> "$tasks_file"
      echo "$spawn_status" >> "$statuses_file"
      echo "$timestamp" >> "$timestamps_file"
      echo "" >> "$completed_file"
      echo "" >> "$children_file"

    elif [[ $pipe_count -eq 3 ]]; then
      # Completion event: timestamp|ant_name|status|summary
      local timestamp ant_name complete_status summary
      timestamp=$(echo "$line" | cut -d'|' -f1)
      ant_name=$(echo "$line" | cut -d'|' -f2)
      complete_status=$(echo "$line" | cut -d'|' -f3)

      # Find the ant and update its status
      local idx=0
      local found=0
      while IFS= read -r name; do
        if [[ "$name" == "$ant_name" ]]; then
          found=1
          break
        fi
        ((idx++))
      done < "$names_file"

      if [[ $found -eq 1 ]]; then
        # Update status at line idx+1 (sed is 1-indexed)
        sed -i.bak "${idx}d" "$statuses_file" 2>/dev/null || sed -i "${idx}d" "$statuses_file"
        sed -i.bak "${idx}i\\
$complete_status" "$statuses_file" 2>/dev/null || sed -i "${idx}i\\
$complete_status" "$statuses_file"
        sed -i.bak "${idx}d" "$completed_file" 2>/dev/null || sed -i "${idx}d" "$completed_file"
        sed -i.bak "${idx}i\\
$timestamp" "$completed_file" 2>/dev/null || sed -i "${idx}i\\
$timestamp" "$completed_file"
        rm -f "$statuses_file.bak" "$completed_file.bak" 2>/dev/null
      fi
    fi
  done < "$file_path"

  # Build parent-child relationships
  local total_count
  total_count=$(wc -l < "$names_file" | tr -d ' ')

  if [[ $total_count -gt 0 ]]; then
    local i
    for ((i = 1; i <= total_count; i++)); do
      local parent
      parent=$(sed -n "${i}p" "$parents_file")

      # Find parent index
      local parent_idx=0
      local found=0
      while IFS= read -r name; do
        if [[ "$name" == "$parent" ]]; then
          found=1
          break
        fi
        ((parent_idx++))
      done < "$names_file"

      if [[ $found -eq 1 ]]; then
        # Add child i-1 to parent's children (0-indexed)
        local child_idx=$((i - 1))
        local current_children
        current_children=$(sed -n "$((parent_idx + 1))p" "$children_file")
        if [[ -z "$current_children" ]]; then
          sed -i.bak "$((parent_idx + 1))d" "$children_file" 2>/dev/null || sed -i "$((parent_idx + 1))d" "$children_file"
          sed -i.bak "$((parent_idx + 1))i\\
$child_idx" "$children_file" 2>/dev/null || sed -i "$((parent_idx + 1))i\\
$child_idx" "$children_file"
        else
          sed -i.bak "$((parent_idx + 1))d" "$children_file" 2>/dev/null || sed -i "$((parent_idx + 1))d" "$children_file"
          sed -i.bak "$((parent_idx + 1))i\\
$current_children $child_idx" "$children_file" 2>/dev/null || sed -i "$((parent_idx + 1))i\\
$current_children $child_idx" "$children_file"
        fi
        rm -f "$children_file.bak" 2>/dev/null
      fi
    done
  fi

  # Count statuses
  local active_count=0
  local completed_count=0
  while IFS= read -r status; do
    if [[ "$status" == "active" || "$status" == "spawned" ]]; then
      ((active_count++))
    elif [[ "$status" == "completed" || "$status" == "failed" || "$status" == "blocked" ]]; then
      ((completed_count++))
    fi
  done < "$statuses_file"

  # Build spawns array
  local spawns_json="["
  local i
  for ((i = 1; i <= total_count; i++)); do
    if [[ $i -gt 1 ]]; then
      spawns_json+=","
    fi

    local name parent caste task status timestamp completed children
    name=$(sed -n "${i}p" "$names_file")
    parent=$(sed -n "${i}p" "$parents_file")
    caste=$(sed -n "${i}p" "$castes_file")
    task=$(sed -n "${i}p" "$tasks_file")
    status=$(sed -n "${i}p" "$statuses_file")
    timestamp=$(sed -n "${i}p" "$timestamps_file")
    completed=$(sed -n "${i}p" "$completed_file")
    children=$(sed -n "${i}p" "$children_file")

    # Build children array
    local children_json="["
    if [[ -n "$children" ]]; then
      local first_child=true
      for child_idx in $children; do
        if [[ "$first_child" == "true" ]]; then
          first_child=false
        else
          children_json+=","
        fi
        local child_name
        child_name=$(sed -n "$((child_idx + 1))p" "$names_file")
        children_json+="\"$child_name\""
      done
    fi
    children_json+="]"

    # Escape task for JSON
    task=$(echo "$task" | sed 's/\\/\\\\/g; s/"/\\"/g; s/\t/\\t/g')

    spawns_json+="{"
    spawns_json+="\"name\":\"$name\","
    spawns_json+="\"parent\":\"$parent\","
    spawns_json+="\"caste\":\"$caste\","
    spawns_json+="\"task\":\"$task\","
    spawns_json+="\"status\":\"$status\","
    spawns_json+="\"spawned_at\":\"$timestamp\","
    spawns_json+="\"completed_at\":\"$completed\","
    spawns_json+="\"children\":$children_json"
    spawns_json+="}"
  done
  spawns_json+="]"

  # Output JSON
  echo "{"
  echo "  \"spawns\": $spawns_json,"
  echo "  \"metadata\": {"
  echo "    \"total_count\": $total_count,"
  echo "    \"active_count\": $active_count,"
  echo "    \"completed_count\": $completed_count,"
  echo "    \"file_exists\": true"
  echo "  }"
  echo "}"

  # Cleanup
  rm -rf "$tmpdir"
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

  if [[ ! -f "$file_path" ]]; then
    echo "[]"
    return 0
  fi

  local active_json="["
  local first=true

  # Read spawn events and check if completed
  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ -z "$line" ]] && continue

    local pipe_count
    pipe_count=$(echo "$line" | tr -cd '|' | wc -c | tr -d ' ')

    if [[ $pipe_count -eq 5 ]]; then
      # Spawn event: timestamp|parent|caste|child_name|task|spawned
      local timestamp parent caste child_name task spawn_status
      timestamp=$(echo "$line" | cut -d'|' -f1)
      parent=$(echo "$line" | cut -d'|' -f2)
      caste=$(echo "$line" | cut -d'|' -f3)
      child_name=$(echo "$line" | cut -d'|' -f4)
      task=$(echo "$line" | cut -d'|' -f5)

      # Check if this ant has a completion event
      local is_completed=false
      if grep -q "^[^|]*|$child_name|completed\|^[^|]*|$child_name|failed\|^[^|]*|$child_name|blocked" "$file_path" 2>/dev/null; then
        is_completed=true
      fi

      if [[ "$is_completed" == "false" ]]; then
        if [[ "$first" == "true" ]]; then
          first=false
        else
          active_json+=","
        fi

        # Escape task for JSON
        task=$(echo "$task" | sed 's/\\/\\\\/g; s/"/\\"/g; s/\t/\\t/g')

        active_json+="{"
        active_json+="\"name\":\"$child_name\","
        active_json+="\"caste\":\"$caste\","
        active_json+="\"parent\":\"$parent\","
        active_json+="\"task\":\"$task\","
        active_json+="\"spawned_at\":\"$timestamp\""
        active_json+="}"
      fi
    fi
  done < "$file_path"

  active_json+="]"
  echo "$active_json"
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

    if [[ $pipe_count -eq 5 ]]; then
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
