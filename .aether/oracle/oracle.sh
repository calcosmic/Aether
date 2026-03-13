#!/bin/bash
# Oracle Ant - Deep research loop using RALF pattern
# Usage: ./oracle.sh [max_iterations_override]
# Based on: https://github.com/snarktank/ralph
#
# Configuration is read from state.json (written by /ant:oracle wizard).
# Command-line arg overrides max_iterations if provided.

set -e

# Unset CLAUDECODE to allow spawning Claude CLI from within Claude Code
unset CLAUDECODE 2>/dev/null || true

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Files
STATE_FILE="$SCRIPT_DIR/state.json"
PLAN_FILE="$SCRIPT_DIR/plan.json"
GAPS_FILE="$SCRIPT_DIR/gaps.md"
SYNTHESIS_FILE="$SCRIPT_DIR/synthesis.md"
RESEARCH_PLAN_FILE="$SCRIPT_DIR/research-plan.md"
STOP_FILE="$SCRIPT_DIR/.stop"
ARCHIVE_DIR="$SCRIPT_DIR/archive"
DISCOVERIES_DIR="$SCRIPT_DIR/discoveries"

# Generate research-plan.md from state.json and plan.json
generate_research_plan() {
  local state_file="$STATE_FILE"
  local plan_file="$PLAN_FILE"
  local output_file="$RESEARCH_PLAN_FILE"

  # Bail if source files don't exist
  [ -f "$state_file" ] || return 0
  [ -f "$plan_file" ] || return 0

  local topic iteration max_iter confidence status
  topic=$(jq -r '.topic' "$state_file")
  iteration=$(jq -r '.iteration' "$state_file")
  max_iter=$(jq -r '.max_iterations' "$state_file")
  confidence=$(jq -r '.overall_confidence' "$state_file")
  status=$(jq -r '.status' "$state_file")

  {
    echo "# Research Plan"
    echo ""
    echo "**Topic:** $topic"
    echo "**Status:** $status | **Iteration:** $iteration of $max_iter"
    echo "**Overall Confidence:** ${confidence}%"
    echo ""
    echo "## Questions"
    echo "| # | Question | Status | Confidence |"
    echo "|---|----------|--------|------------|"
    jq -r '.questions[] | "| \(.id) | \(.text) | \(.status) | \(.confidence)% |"' "$plan_file"
    echo ""
    echo "## Next Steps"
    local next
    next=$(jq -r '[.questions[] | select(.status != "answered")] | sort_by(.confidence) | first | .text // "All questions answered"' "$plan_file")
    echo "Next investigation: $next"
    echo ""
    echo "---"
    echo "*Generated from plan.json -- do not edit directly*"
  } > "$output_file"
}

# Check state.json exists (wizard must create it before launching oracle.sh)
if [ ! -f "$STATE_FILE" ]; then
  echo "Error: No state.json found. Run /ant:oracle to configure research first."
  exit 1
fi

# Read config from state.json (wizard writes these)
CURRENT_TOPIC=$(jq -r '.topic // empty' "$STATE_FILE" 2>/dev/null || echo "")
TARGET_CONFIDENCE=$(jq -r '.target_confidence // 95' "$STATE_FILE" 2>/dev/null || echo "95")
JSON_MAX_ITER=$(jq -r '.max_iterations // 50' "$STATE_FILE" 2>/dev/null || echo "50")

# Command-line arg overrides state.json
MAX_ITERATIONS=${1:-$JSON_MAX_ITER}

# Detect AI CLI (claude or opencode)
if command -v claude &>/dev/null; then
  AI_CMD="claude --dangerously-skip-permissions --print"
elif command -v opencode &>/dev/null; then
  AI_CMD="opencode --dangerously-skip-permissions --print"
else
  echo "Error: Neither 'claude' nor 'opencode' CLI found on PATH."
  exit 1
fi

# Archive previous run if topic changed
if [ -f "$STATE_FILE" ]; then
  LAST_TOPIC=$(jq -r '.topic // empty' "$STATE_FILE" 2>/dev/null || echo "")
  # If the wizard passed a new topic via environment, compare
  if [ -n "${ORACLE_NEW_TOPIC:-}" ] && [ -n "$LAST_TOPIC" ] && [ "$ORACLE_NEW_TOPIC" != "$LAST_TOPIC" ]; then
    ARCHIVE_FOLDER="$ARCHIVE_DIR/$(date +%Y-%m-%d-%H%M%S)"

    echo "Archiving previous research: $LAST_TOPIC"
    mkdir -p "$ARCHIVE_FOLDER"
    for f in state.json plan.json gaps.md synthesis.md research-plan.md; do
      [ -f "$SCRIPT_DIR/$f" ] && cp "$SCRIPT_DIR/$f" "$ARCHIVE_FOLDER/"
    done
    echo "   Archived to: $ARCHIVE_FOLDER"
    # Do NOT create empty files -- the wizard handles initial file creation
  fi
fi

# Initialize discoveries directory
mkdir -p "$DISCOVERIES_DIR"

echo ""
echo "==============================================================="
echo "  ORACLE ANT - Deep Research Loop"
echo "==============================================================="
echo "Topic:       $CURRENT_TOPIC"
echo "Iterations:  $MAX_ITERATIONS"
echo "Confidence:  $TARGET_CONFIDENCE%"
echo "CLI:         $AI_CMD"
echo ""

# Main loop
for i in $(seq 1 "$MAX_ITERATIONS"); do
  # Check for stop signal
  if [ -f "$STOP_FILE" ]; then
    rm -f "$STOP_FILE"
    echo ""
    echo "Oracle stopped by user at iteration $i"
    break
  fi

  echo ""
  echo "---------------------------------------------------------------"
  echo "  Iteration $i of $MAX_ITERATIONS"
  echo "---------------------------------------------------------------"

  # Run AI with oracle.md prompt
  OUTPUT=$($AI_CMD < "$SCRIPT_DIR/oracle.md" 2>&1 | tee /dev/stderr) || true

  # Basic jq validation after iteration (safety check -- Phase 8 adds recovery)
  if ! jq -e . "$STATE_FILE" >/dev/null 2>&1; then
    echo "WARNING: state.json is invalid JSON after iteration $i"
  fi
  if ! jq -e . "$PLAN_FILE" >/dev/null 2>&1; then
    echo "WARNING: plan.json is invalid JSON after iteration $i"
  fi

  # Regenerate research-plan.md from current state
  generate_research_plan

  # Check for completion signal
  if echo "$OUTPUT" | grep -q "<oracle>COMPLETE</oracle>"; then
    echo ""
    echo "==============================================================="
    echo "  ORACLE RESEARCH COMPLETE!"
    echo "==============================================================="
    echo "Completed at iteration $i"
    exit 0
  fi

  echo ""
  echo "Iteration $i complete. Continuing..."
  sleep 2
done

echo ""
echo "==============================================================="
echo "  ORACLE REACHED MAX ITERATIONS"
echo "==============================================================="
echo "Max iterations ($MAX_ITERATIONS) reached without completion."
echo "Check $RESEARCH_PLAN_FILE for current research status."
exit 1
