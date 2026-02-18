---
name: ant:watch
description: "👁️🔄🐜🏠🔄👁️ Set up tmux session to watch ants working in real-time"
---

You are the **Queen**. Set up live visibility into colony activity.

## Instructions

### Step 1: Check Prerequisites

Use Bash with description "Checking for tmux..." to check if tmux is available:
```bash
command -v tmux >/dev/null 2>&1 && echo "tmux_available" || echo "tmux_missing"
```

If tmux is missing:
```
tmux is required for live colony viewing.

Install with:
  macOS:  brew install tmux
  Ubuntu: sudo apt install tmux
  Fedora: sudo dnf install tmux
```
Stop here.

### Step 2: Initialize Activity Log

Ensure activity log exists by running using the Bash tool with description "Initializing watch files...":
```bash
mkdir -p .aether/data
touch .aether/data/activity.log
```

### Step 2.5: Check for Stale Watch Session

Capture session start time:
```bash
WATCH_START=$(date +%s)
```

Check for stale watch files by running using the Bash tool with description "Checking for stale watch session...":
```bash
stale_check=$(bash .aether/aether-utils.sh session-verify-fresh --command watch "" "$WATCH_START")
has_stale=$(echo "$stale_check" | jq -r '.stale | length')
```

If stale files exist, they will be overwritten by the new watch session.
The tmux session check in Step 4 handles concurrent sessions.

### Step 3: Create Status File

Write initial status to `.aether/data/watch-status.txt`:

```
       .-.
      (o o)  AETHER COLONY
      | O |  Live Status
       `-`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Session Started: $(date -u +"%Y-%m-%dT%H:%M:%SZ")
State: IDLE
Phase: -/-

Active Workers:
  (none)

Last Activity:
  (waiting for colony activity)
```

### Step 4: Create or Attach to tmux Session

Check if session exists by running using the Bash tool with description "Checking tmux session...":
```bash
tmux has-session -t aether-colony 2>/dev/null && echo "exists" || echo "new"
```

**If session exists:** Attach to it by running using the Bash tool with description "Attaching to watch session...":
```bash
tmux attach-session -t aether-colony
```
Output: `Attached to existing aether-colony session.`
Stop here.

**If session is new:** Create the layout.

### Step 5: Create tmux Layout (4-Pane)

Use Bash with description "Creating watch session layout..." to create the session with 4 panes in a 2x2 grid:

```bash
# Create session with first pane
tmux new-session -d -s aether-colony -n colony

# Split into 4 panes (2x2 grid)
# First split horizontally (left|right)
tmux split-window -h -t aether-colony:colony

# Split left side vertically (top-left, bottom-left)
tmux split-window -v -t aether-colony:colony.0

# Split right side vertically (top-right, bottom-right)
tmux split-window -v -t aether-colony:colony.2

# Set pane contents:
# Pane 0 (top-left): Status display
tmux send-keys -t aether-colony:colony.0 'watch -n 1 cat .aether/data/watch-status.txt' C-m

# Pane 1 (bottom-left): Progress bar
tmux send-keys -t aether-colony:colony.1 'watch -n 1 cat .aether/data/watch-progress.txt' C-m

# Pane 2 (top-right): Spawn tree visualization
tmux send-keys -t aether-colony:colony.2 'bash .aether/utils/watch-spawn-tree.sh .aether/data' C-m

# Pane 3 (bottom-right): Colorized activity log stream
tmux send-keys -t aether-colony:colony.3 'bash .aether/utils/colorize-log.sh .aether/data/activity.log' C-m

# Set pane titles (if supported)
tmux select-pane -t aether-colony:colony.0 -T "Status"
tmux select-pane -t aether-colony:colony.1 -T "Progress"
tmux select-pane -t aether-colony:colony.2 -T "Spawn Tree"
tmux select-pane -t aether-colony:colony.3 -T "Activity Log"

# Balance panes for even 2x2 grid
tmux select-layout -t aether-colony:colony tiled

echo "Session created"
```

### Step 6: Create Progress File

Write initial progress to `.aether/data/watch-progress.txt`:

```
       .-.
      (o o)  AETHER COLONY
      | O |  Progress
       `-`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Phase: -/-

[░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░] 0%

⏳ Waiting for build...

Target: 95% confidence
```

### Step 7: Attach and Display

Run using the Bash tool with description "Attaching to watch display...":
```bash
tmux attach-session -t aether-colony
```

Before attaching, output:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
👁️🔄🐜🏠🔄👁️  A E T H E R   C O L O N Y   W A T C H
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

tmux session 'aether-colony' created.

Layout (4-pane 2x2 grid):
  +------------------+------------------+
  | Status           | Spawn Tree       |
  | Colony state     | Worker hierarchy |
  +------------------+------------------+
  | Progress         | Activity Log     |
  | Phase progress   | Live stream      |
  +------------------+------------------+

Commands:
  Ctrl+B D          Detach from session
  Ctrl+B [          Scroll mode (q to exit)
  Ctrl+B Arrow      Navigate between panes
  tmux kill-session -t aether-colony   Stop watching

The session will update in real-time as colony works.
Attaching now...
```

Generate the state-based Next Up block by running using the Bash tool with description "Generating Next Up suggestions...":
```bash
state=$(jq -r '.state // "IDLE"' .aether/data/COLONY_STATE.json)
current_phase=$(jq -r '.current_phase // 0' .aether/data/COLONY_STATE.json)
total_phases=$(jq -r '.plan.phases | length' .aether/data/COLONY_STATE.json)
bash .aether/aether-utils.sh print-next-up "$state" "$current_phase" "$total_phases"
```

---

## Status Update Protocol

Workers and commands update watch files as they work:

### Activity Log
Workers write via: `bash .aether/aether-utils.sh activity-log "ACTION" "caste" "description"`

For named ants (recommended):
```bash
# Generate a name first
ant_name=$(bash .aether/aether-utils.sh generate-ant-name "builder" | jq -r '.result')
# Log with ant name
bash .aether/aether-utils.sh activity-log "CREATED" "$ant_name (Builder)" "Implemented auth module"
```

### Spawn Tracking
Log spawns for tree visualization:
```bash
bash .aether/aether-utils.sh spawn-log "Prime" "builder" "Hammer-42" "implementing auth"
bash .aether/aether-utils.sh spawn-complete "Hammer-42" "completed" "auth module done"
```

### Status File
Commands update `.aether/data/watch-status.txt` with current state:
- State: PLANNING, EXECUTING, READY
- Phase: current/total
- Active Workers: list of named ants
- Last Activity: most recent log entry

### Progress File
Update via: `bash .aether/aether-utils.sh update-progress <percent> "<message>" <phase> <total>`

Example:
```bash
bash .aether/aether-utils.sh update-progress 45 "Building auth module..." 2 5
```

---

## Cleanup

To stop watching:
```bash
tmux kill-session -t aether-colony
```

This stops the session but preserves all log files.
