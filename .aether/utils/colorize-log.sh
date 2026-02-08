#!/bin/bash
# Colorized activity log stream for tmux watch pane
# Usage: bash colorize-log.sh [log_file]

LOG_FILE="${1:-.aether/data/activity.log}"

# ANSI color codes
YELLOW='\033[33m'
GREEN='\033[32m'
RED='\033[31m'
CYAN='\033[36m'
MAGENTA='\033[35m'
BLUE='\033[34m'
BOLD='\033[1m'
RESET='\033[0m'

# Colony header
echo -e "${BOLD}${CYAN}"
cat << 'EOF'
       .-.
      (o o)  AETHER COLONY
      | O |  Activity Stream
       `-`
EOF
echo -e "${RESET}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Stream and colorize log entries
tail -f "$LOG_FILE" 2>/dev/null | while IFS= read -r line; do
  case "$line" in
    *SPAWN*)
      printf "${YELLOW}${BOLD}%s${RESET}\n" "$line"
      ;;
    *COMPLETE*|*completed*)
      printf "${GREEN}%s${RESET}\n" "$line"
      ;;
    *ERROR*|*FAILED*|*failed*)
      printf "${RED}${BOLD}%s${RESET}\n" "$line"
      ;;
    *CREATED*)
      printf "${CYAN}%s${RESET}\n" "$line"
      ;;
    *MODIFIED*)
      printf "${BLUE}%s${RESET}\n" "$line"
      ;;
    *RESEARCH*|*EXPLORING*)
      printf "${MAGENTA}%s${RESET}\n" "$line"
      ;;
    *EXECUTING*|*BUILDING*)
      printf "${YELLOW}%s${RESET}\n" "$line"
      ;;
    "# Phase"*)
      # Phase headers get special treatment
      printf "\n${BOLD}${CYAN}%s${RESET}\n" "$line"
      echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
      ;;
    *)
      echo "$line"
      ;;
  esac
done
