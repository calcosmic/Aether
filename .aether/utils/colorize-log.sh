#!/bin/bash
# Colorized activity log stream for tmux watch pane
# Usage: bash colorize-log.sh [log_file]

LOG_FILE="${1:-.aether/data/activity.log}"

# ANSI color codes by caste (as per V2 improvement plan)
QUEEN='\033[35m'        # Magenta
BUILDER='\033[33m'      # Yellow
WATCHER='\033[36m'      # Cyan
SCOUT='\033[32m'        # Green
COLONIZER='\033[34m'    # Blue
ARCHITECT='\033[37m'    # White

# Action colors (bright variants)
SPAWN='\033[93m'        # Bright Yellow
COMPLETE='\033[92m'     # Bright Green
ERROR='\033[91m'        # Bright Red
CREATED='\033[96m'      # Bright Cyan
MODIFIED='\033[94m'     # Bright Blue

# Base colors
YELLOW='\033[33m'
GREEN='\033[32m'
RED='\033[31m'
CYAN='\033[36m'
MAGENTA='\033[35m'
BLUE='\033[34m'
BOLD='\033[1m'
DIM='\033[2m'
RESET='\033[0m'

# Get caste color from ant name patterns
get_caste_color() {
  case "$1" in
    *Queen*|*QUEEN*)
      echo "$QUEEN"
      ;;
    *Builder*|*Bolt*|*Hammer*|*Forge*|*Mason*|*Brick*|*Anvil*|*Weld*)
      echo "$BUILDER"
      ;;
    *Watcher*|*Vigil*|*Sentinel*|*Guard*|*Keen*|*Sharp*|*Hawk*|*Watch*|*Alert*)
      echo "$WATCHER"
      ;;
    *Scout*|*Swift*|*Dash*|*Ranger*|*Track*|*Seek*|*Path*|*Roam*|*Quest*)
      echo "$SCOUT"
      ;;
    *Colonizer*|*Pioneer*|*Map*|*Chart*|*Venture*|*Explore*|*Compass*|*Atlas*|*Trek*)
      echo "$COLONIZER"
      ;;
    *Architect*|*Blueprint*|*Draft*|*Design*|*Plan*|*Schema*|*Frame*|*Sketch*|*Model*)
      echo "$ARCHITECT"
      ;;
    *Prime*|*Alpha*|*Lead*|*Chief*|*First*|*Core*|*Apex*|*Crown*)
      echo "$MAGENTA"
      ;;
    *)
      echo "$RESET"
      ;;
  esac
}

# Get emoji for caste
get_emoji() {
  case "$1" in
    *Queen*|*QUEEN*) echo "👑" ;;
    *Builder*|*Bolt*|*Hammer*|*Forge*|*Mason*|*Brick*|*Anvil*|*Weld*) echo "🔨" ;;
    *Watcher*|*Vigil*|*Sentinel*|*Guard*|*Keen*|*Sharp*|*Hawk*|*Watch*|*Alert*) echo "👁️" ;;
    *Scout*|*Swift*|*Dash*|*Ranger*|*Track*|*Seek*|*Path*|*Roam*|*Quest*) echo "🔍" ;;
    *Colonizer*|*Pioneer*|*Map*|*Chart*|*Venture*|*Explore*|*Compass*|*Atlas*|*Trek*) echo "🗺️" ;;
    *Architect*|*Blueprint*|*Draft*|*Design*|*Plan*|*Schema*|*Frame*|*Sketch*|*Model*) echo "🏛️" ;;
    *Prime*|*Alpha*|*Lead*|*Chief*|*First*|*Core*|*Apex*|*Crown*) echo "👑" ;;
    *) echo "🐜" ;;
  esac
}

# Colony header
echo -e "${BOLD}${MAGENTA}"
cat << 'EOF'
       .-.
      (o o)  AETHER COLONY
      | O |  Activity Stream
       `-`
EOF
echo -e "${RESET}"
echo -e "${DIM}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
echo ""

# Stream and colorize log entries
tail -f "$LOG_FILE" 2>/dev/null | while IFS= read -r line; do
  # Extract caste color if ant name is in the line
  caste_color=$(get_caste_color "$line")
  emoji=$(get_emoji "$line")

  case "$line" in
    *SPAWN*)
      printf "${SPAWN}⚡ %s${RESET}\n" "$line"
      ;;
    *COMPLETE*|*completed*)
      printf "${COMPLETE}✅ %s${RESET}\n" "$line"
      ;;
    *ERROR*|*FAILED*|*failed*)
      printf "${ERROR}${BOLD}❌ %s${RESET}\n" "$line"
      ;;
    *CREATED*)
      printf "${caste_color}✨ %s${RESET}\n" "$line"
      ;;
    *MODIFIED*)
      printf "${caste_color}📝 %s${RESET}\n" "$line"
      ;;
    *RESEARCH*|*EXPLORING*)
      printf "${caste_color}🔬 %s${RESET}\n" "$line"
      ;;
    *EXECUTING*|*BUILDING*)
      printf "${caste_color}⚙️  %s${RESET}\n" "$line"
      ;;
    "# Phase"*)
      # Phase headers get special treatment
      printf "\n${BOLD}${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}\n"
      printf "${BOLD}${MAGENTA}🐜 %s${RESET}\n" "$line"
      printf "${BOLD}${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}\n"
      ;;
    *)
      # Apply caste color if detected, otherwise default
      if [[ "$caste_color" != "$RESET" ]]; then
        printf "${caste_color}%s %s${RESET}\n" "$emoji" "$line"
      else
        echo "$line"
      fi
      ;;
  esac
done
