#!/bin/bash
# Oracle Ant - Deep research loop using RALF pattern
# Usage: ./oracle.sh [max_iterations_override]
# Based on: https://github.com/snarktank/ralph
#
# Configuration is read from research.json (written by /ant:oracle wizard).
# Command-line arg overrides max_iterations if provided.

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Files
RESEARCH_FILE="$SCRIPT_DIR/research.json"
PROGRESS_FILE="$SCRIPT_DIR/progress.md"
STOP_FILE="$SCRIPT_DIR/.stop"
ARCHIVE_DIR="$SCRIPT_DIR/archive"
DISCOVERIES_DIR="$SCRIPT_DIR/discoveries"

# Check research.json exists
if [ ! -f "$RESEARCH_FILE" ]; then
  echo "Error: No research.json found. Run /ant:oracle to configure research first."
  exit 1
fi

# Read config from research.json (wizard writes these)
CURRENT_TOPIC=$(jq -r '.topic // empty' "$RESEARCH_FILE" 2>/dev/null || echo "")
TARGET_CONFIDENCE=$(jq -r '.target_confidence // 95' "$RESEARCH_FILE" 2>/dev/null || echo "95")
JSON_MAX_ITER=$(jq -r '.max_iterations // 50' "$RESEARCH_FILE" 2>/dev/null || echo "50")

# Command-line arg overrides research.json
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
LAST_TOPIC_FILE="$SCRIPT_DIR/.last-topic"
if [ -f "$LAST_TOPIC_FILE" ] && [ -f "$PROGRESS_FILE" ]; then
  LAST_TOPIC=$(cat "$LAST_TOPIC_FILE" 2>/dev/null || echo "")
  if [ -n "$CURRENT_TOPIC" ] && [ -n "$LAST_TOPIC" ] && [ "$CURRENT_TOPIC" != "$LAST_TOPIC" ]; then
    DATE=$(date +%Y-%m-%d)
    TOPIC_SLUG=$(echo "$LAST_TOPIC" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/-/g' | sed 's/--*/-/g' | sed 's/^-\|-$//g')
    ARCHIVE_FOLDER="$ARCHIVE_DIR/$DATE-$TOPIC_SLUG"

    echo "Archiving previous research: $LAST_TOPIC"
    mkdir -p "$ARCHIVE_FOLDER"
    [ -f "$RESEARCH_FILE" ] && cp "$RESEARCH_FILE" "$ARCHIVE_FOLDER/"
    [ -f "$PROGRESS_FILE" ] && cp "$PROGRESS_FILE" "$ARCHIVE_FOLDER/"
    echo "   Archived to: $ARCHIVE_FOLDER"

    # Reset progress file
    echo "# Oracle Research Progress" > "$PROGRESS_FILE"
    echo "" >> "$PROGRESS_FILE"
  fi
fi

# Track current topic
if [ -n "$CURRENT_TOPIC" ]; then
  echo "$CURRENT_TOPIC" > "$LAST_TOPIC_FILE"
fi

# Initialize progress file if needed
if [ ! -f "$PROGRESS_FILE" ]; then
  echo "# Oracle Research Progress" > "$PROGRESS_FILE"
  echo "" >> "$PROGRESS_FILE"
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
echo "Check $PROGRESS_FILE for current research status."
exit 1
