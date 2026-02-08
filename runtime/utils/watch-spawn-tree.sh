#!/bin/bash
# Live spawn tree visualization for tmux watch pane
# Usage: bash watch-spawn-tree.sh [data_dir]

DATA_DIR="${1:-.aether/data}"
SPAWN_FILE="$DATA_DIR/spawn-tree.txt"

# ANSI colors
YELLOW='\033[33m'
GREEN='\033[32m'
RED='\033[31m'
CYAN='\033[36m'
MAGENTA='\033[35m'
BOLD='\033[1m'
DIM='\033[2m'
RESET='\033[0m'

# Caste emojis
get_emoji() {
  case "$1" in
    builder)   echo "🔨" ;;
    watcher)   echo "👁️ " ;;
    scout)     echo "🔍" ;;
    colonizer) echo "🗺️ " ;;
    architect) echo "🏛️ " ;;
    prime)     echo "👑" ;;
    *)         echo "🐜" ;;
  esac
}

# Status colors
get_status_color() {
  case "$1" in
    completed) echo "$GREEN" ;;
    failed)    echo "$RED" ;;
    spawned)   echo "$YELLOW" ;;
    *)         echo "$CYAN" ;;
  esac
}

render_tree() {
  clear

  # Header
  echo -e "${BOLD}${CYAN}"
  cat << 'EOF'
       .-.
      (o o)  AETHER COLONY
      | O |  Spawn Tree
       `-`
EOF
  echo -e "${RESET}"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo ""

  # Always show Queen at depth 0
  echo -e "  ${BOLD}👑 Queen${RESET} ${DIM}(depth 0)${RESET}"
  echo -e "  ${DIM}│${RESET}"

  if [[ ! -f "$SPAWN_FILE" ]]; then
    echo -e "  ${DIM}└── (no workers spawned yet)${RESET}"
    return
  fi

  # Parse spawn tree file
  # Format: timestamp|parent_id|child_caste|child_name|task_summary|status
  declare -A workers
  declare -A worker_status
  declare -A worker_task
  declare -A worker_caste
  declare -a roots

  while IFS='|' read -r ts parent caste name task status rest; do
    [[ -z "$name" ]] && continue

    # Check if this is a status update (only 4 fields)
    if [[ -z "$task" && -n "$caste" ]]; then
      # This is a status update: ts|name|status|summary
      worker_status["$parent"]="$caste"
      continue
    fi

    workers["$name"]="$parent"
    worker_caste["$name"]="$caste"
    worker_task["$name"]="$task"
    worker_status["$name"]="${status:-spawned}"

    # Track root workers (spawned by Prime or Queen)
    if [[ "$parent" == "Prime"* || "$parent" == "prime"* || "$parent" == "Queen" ]]; then
      roots+=("$name")
    fi
  done < "$SPAWN_FILE"

  # Render workers in tree structure
  # Group by parent to show hierarchy
  printed=()

  # Function to render a worker and its children
  render_worker() {
    local name="$1"
    local indent="$2"
    local depth="$3"
    local is_last="$4"

    [[ " ${printed[*]} " =~ " $name " ]] && return
    printed+=("$name")

    emoji=$(get_emoji "${worker_caste[$name]}")
    status="${worker_status[$name]}"
    color=$(get_status_color "$status")
    task="${worker_task[$name]}"

    # Truncate task for display
    [[ ${#task} -gt 30 ]] && task="${task:0:27}..."

    # Tree connectors
    if [[ "$is_last" == "true" ]]; then
      connector="└──"
    else
      connector="├──"
    fi

    echo -e "${indent}${DIM}${connector}${RESET} ${emoji} ${color}${name}${RESET}: ${task} ${DIM}[depth $depth]${RESET}"

    # Find children of this worker
    local children=()
    for child in "${!workers[@]}"; do
      if [[ "${workers[$child]}" == "$name" ]]; then
        children+=("$child")
      fi
    done

    # Render children
    local child_indent="${indent}    "
    if [[ "$is_last" != "true" ]]; then
      child_indent="${indent}${DIM}│${RESET}   "
    fi

    local child_count=${#children[@]}
    local child_idx=0
    for child in "${children[@]}"; do
      child_idx=$((child_idx + 1))
      local child_is_last="false"
      [[ $child_idx -eq $child_count ]] && child_is_last="true"
      render_worker "$child" "$child_indent" $((depth + 1)) "$child_is_last"
    done
  }

  # Render root workers (spawned by Queen) at depth 1
  local root_count=${#roots[@]}
  local root_idx=0
  for name in "${roots[@]}"; do
    root_idx=$((root_idx + 1))
    local is_last="false"
    [[ $root_idx -eq $root_count ]] && is_last="true"
    render_worker "$name" "  " 1 "$is_last"
  done

  # Summary
  echo ""
  echo -e "${DIM}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
  completed=$(grep -c "completed" "$SPAWN_FILE" 2>/dev/null || echo "0")
  active=$(grep -c "spawned" "$SPAWN_FILE" 2>/dev/null || echo "0")
  echo -e "Workers: ${GREEN}$completed completed${RESET} | ${YELLOW}$active active${RESET}"
}

# Initial render
render_tree

# Watch for changes and re-render
if command -v fswatch &>/dev/null; then
  fswatch -o "$SPAWN_FILE" 2>/dev/null | while read; do
    render_tree
  done
elif command -v inotifywait &>/dev/null; then
  while inotifywait -q -e modify "$SPAWN_FILE" 2>/dev/null; do
    render_tree
  done
else
  # Fallback: poll every 2 seconds
  while true; do
    sleep 2
    render_tree
  done
fi
