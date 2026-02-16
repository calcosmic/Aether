#!/bin/bash
# Real-time swarm activity display
# Usage: bash swarm-display.sh [swarm_id]

SWARM_ID="${1:-current}"
DATA_DIR="${DATA_DIR:-.aether/data}"
DISPLAY_FILE="$DATA_DIR/swarm-display.json"

# ANSI colors (matching caste-colors.js)
BLUE='\033[34m'
GREEN='\033[32m'
YELLOW='\033[33m'
RED='\033[31m'
MAGENTA='\033[35m'
BOLD='\033[1m'
UNDERLINE='\033[4m'
DIM='\033[2m'
RESET='\033[0m'

# Caste colors (must match caste-colors.js)
get_caste_color() {
  case "$1" in
    builder)  echo "$BLUE" ;;
    watcher)  echo "$GREEN" ;;
    scout)    echo "$YELLOW" ;;
    chaos)    echo "$RED" ;;
    prime)    echo "$MAGENTA" ;;
    *)        echo "$RESET" ;;
  esac
}

# Caste emojis (must match aether-utils.sh)
get_caste_emoji() {
  case "$1" in
    builder)  echo "🔨🐜" ;;
    watcher)  echo "👁️🐜" ;;
    scout)    echo "🔍🐜" ;;
    chaos)    echo "🎲🐜" ;;
    prime)    echo "👑🐜" ;;
    *)        echo "🐜" ;;
  esac
}

# Animated status phrases
get_status_phrase() {
  local caste="$1"
  local idx=$(($(date +%s) % 4))
  case "$caste" in
    builder)
      phrases=("excavating..." "building..." "forging..." "constructing...")
      ;;
    watcher)
      phrases=("observing..." "monitoring..." "watching..." "tracking...")
      ;;
    scout)
      phrases=("exploring..." "searching..." "investigating..." "probing...")
      ;;
    chaos)
      phrases=("disrupting..." "testing..." "probing..." "stressing...")
      ;;
    *)
      phrases=("working..." "foraging..." "excavating..." "tunneling...")
      ;;
  esac
  echo "${phrases[$idx]}"
}

# Format tool usage: "📖5 🔍3 ✏️2 ⚡1"
format_tools() {
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

# Format duration from seconds
format_duration() {
  local seconds="${1:-0}"
  if [[ "$seconds" -lt 60 ]]; then
    echo "${seconds}s"
  else
    local mins=$((seconds / 60))
    local secs=$((seconds % 60))
    echo "${mins}m${secs}s"
  fi
}

# Render progress bar
render_progress_bar() {
  local percent="${1:-0}"
  local width="${2:-20}"

  # Clamp percent to 0-100
  [[ "$percent" -lt 0 ]] && percent=0
  [[ "$percent" -gt 100 ]] && percent=100

  local filled=$((percent * width / 100))
  local empty=$((width - filled))

  local bar=""
  for ((i=0; i<filled; i++)); do bar+="█"; done
  for ((i=0; i<empty; i++)); do bar+="░"; done

  echo "[$bar] $percent%"
}

# Get animated spinner
get_spinner() {
  local spinners=("⠋" "⠙" "⠹" "⠸" "⠼" "⠴" "⠦" "⠧" "⠇" "⠏")
  local idx=$(($(date +%s) % 10))
  echo "${spinners[$idx]}"
}

# Get excavation phrase based on progress
get_excavation_phrase() {
  local caste="$1"
  local progress="${2:-0}"

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

# Render the swarm display
render_swarm() {
  clear

  # Header
  echo -e "${BOLD}${MAGENTA}"
  cat << 'EOF'
       .-.
      (o o)  AETHER COLONY
      | O |  Swarm Activity
       `-`
EOF
  echo -e "${RESET}"
  echo -e "${DIM}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
  echo ""

  if [[ ! -f "$DISPLAY_FILE" ]]; then
    echo -e "${DIM}Waiting for swarm activity...${RESET}"
    echo ""
    echo -e "${DIM}0 foragers excavating...${RESET}"
    return
  fi

  # Read swarm data
  local swarm_data=$(cat "$DISPLAY_FILE" 2>/dev/null)
  if [[ -z "$swarm_data" ]]; then
    echo -e "${DIM}No active swarm data${RESET}"
    return
  fi

  # Check if we have active ants
  local total_active=$(echo "$swarm_data" | jq -r '.summary.total_active // 0')

  if [[ "$total_active" -eq 0 ]]; then
    echo -e "${DIM}No active foragers${RESET}"
    echo ""
    echo -e "${DIM}0 foragers excavating...${RESET}"
    return
  fi

  # Render each active ant
  echo "$swarm_data" | jq -r '.active_ants[] |
    "\(.name)|\(.caste)|\(.status)|\(.task // "")|\(.tools.read // 0)|\(.tools.grep // 0)|\(.tools.edit // 0)|\(.tools.bash // 0)|\(.tokens // 0)|\(.started_at // "")|\(.parent // "Queen")|\(.progress // 0)"' 2>/dev/null | \
  while IFS='|' read -r name caste status task read_count grep_count edit_count bash_count tokens started_at parent progress; do
    color=$(get_caste_color "$caste")
    emoji=$(get_caste_emoji "$caste")
    phrase=$(get_status_phrase "$caste")

    # Parent ants: bold + underline
    if [[ "$parent" == "Queen" ]] || [[ "$parent" == "Prime"* ]]; then
      style="${BOLD}${UNDERLINE}"
    else
      style="${BOLD}"
    fi

    # Format tools
    tools_str=$(format_tools "$read_count" "$grep_count" "$edit_count" "$bash_count")

    # Calculate elapsed time
    elapsed_str=""
    if [[ -n "$started_at" ]]; then
      started_ts=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "$started_at" +%s 2>/dev/null || date -d "$started_at" +%s 2>/dev/null || echo "0")
      now_ts=$(date +%s)
      elapsed=$((now_ts - started_ts))
      if [[ $elapsed -gt 0 ]]; then
        elapsed_str="${DIM}($(format_duration $elapsed))${RESET}"
      fi
    fi

    # Trophallaxis (token) indicator
    token_str=""
    if [[ "$tokens" -gt 0 ]]; then
      token_str="${DIM}🍯${tokens}${RESET}"
    fi

    # Truncate task if too long
    display_task="$task"
    [[ ${#display_task} -gt 35 ]] && display_task="${display_task:0:32}..."

    # Output line: "🔨 Builder: excavating... Implement auth 📖5 🔍3 (2m3s) 🍯1250"
    echo -e "${color}${emoji} ${style}${name}${RESET}${color}: ${phrase}${RESET} ${display_task}"
    echo -e "   ${tools_str} ${elapsed_str} ${token_str}"

    # Show progress bar if progress > 0
    if [[ "$progress" -gt 0 ]]; then
      progress_bar=$(render_progress_bar "$progress" 15)
      excavation_phrase=$(get_excavation_phrase "$caste" "$progress")
      echo -e "   ${DIM}${progress_bar}${RESET}"
      echo -e "   ${DIM}$(get_spinner) ${excavation_phrase}${RESET}"
    fi

    echo ""
  done

  # Chamber activity map (VIZ-07)
  echo -e "${DIM}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
  echo ""
  echo -e "${BOLD}Chamber Activity:${RESET}"

  # Show active chambers with fire intensity
  echo "$swarm_data" | jq -r '.chambers | to_entries[] | "\(.key)|\(.value.activity)|\(.value.icon)"' 2>/dev/null | \
  while IFS='|' read -r chamber activity icon; do
    if [[ "$activity" -gt 0 ]]; then
      # Fire intensity based on activity count
      if [[ "$activity" -ge 5 ]]; then
        fires="🔥🔥🔥"
      elif [[ "$activity" -ge 3 ]]; then
        fires="🔥🔥"
      else
        fires="🔥"
      fi
      echo -e "  ${icon} ${chamber//_/ } ${fires} (${activity} ants)"
    fi
  done

  # Summary line (VIZ-06: ant-themed presentation)
  echo ""
  echo -e "${DIM}${total_active} forager$([[ "$total_active" -eq 1 ]] || echo "s") excavating...${RESET}"
}

# Main loop with file watching
render_swarm

if command -v fswatch &>/dev/null; then
  fswatch -o "$DISPLAY_FILE" 2>/dev/null | while read; do render_swarm; done
elif command -v inotifywait &>/dev/null; then
  while inotifywait -q -e modify "$DISPLAY_FILE" 2>/dev/null; do render_swarm; done
else
  # Fallback: poll every 2 seconds
  while true; do sleep 2; render_swarm; done
fi
