---
name: ant:watch
description: Set up tmux session to watch ants working in real-time
---

You are the **Queen**. Set up live visibility into colony activity.

## Instructions

### Step 1: Check Prerequisites

Use Bash to check if tmux is available:
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

Ensure activity log exists:
```bash
mkdir -p .aether/data
touch .aether/data/activity.log
```

### Step 3: Create Status File

Write initial status to `.aether/data/watch-status.txt`:

```
AETHER COLONY :: LIVE STATUS
=============================

State: IDLE
Phase: -/-
Confidence: --%
Iteration: -/-

Active Workers:
  (none)

Last Activity:
  (waiting for colony activity)
```

### Step 4: Create or Attach to tmux Session

Check if session exists:
```bash
tmux has-session -t aether-colony 2>/dev/null && echo "exists" || echo "new"
```

**If session exists:** Attach to it
```bash
tmux attach-session -t aether-colony
```
Output: `Attached to existing aether-colony session.`
Stop here.

**If session is new:** Create the layout.

### Step 5: Create tmux Layout

Use Bash to create the session with panes:

```bash
# Create session with first pane (Activity Log)
tmux new-session -d -s aether-colony -n colony

# Split horizontally: left = status, right = activity log
tmux split-window -h -t aether-colony:colony

# Split left pane vertically: top = status, bottom = progress
tmux split-window -v -t aether-colony:colony.0

# Set pane contents
# Pane 0 (top-left): Status display
tmux send-keys -t aether-colony:colony.0 'watch -n 1 cat .aether/data/watch-status.txt' C-m

# Pane 1 (bottom-left): Progress bar (updates via file)
tmux send-keys -t aether-colony:colony.1 'watch -n 1 cat .aether/data/watch-progress.txt' C-m

# Pane 2 (right): Activity log stream
tmux send-keys -t aether-colony:colony.2 'tail -f .aether/data/activity.log' C-m

# Set pane titles (if supported)
tmux select-pane -t aether-colony:colony.0 -T "Status"
tmux select-pane -t aether-colony:colony.1 -T "Progress"
tmux select-pane -t aether-colony:colony.2 -T "Activity Log"

# Resize panes: left side 40%, right side 60%
tmux resize-pane -t aether-colony:colony.2 -x 60%

echo "Session created"
```

### Step 6: Create Progress File

Write initial progress to `.aether/data/watch-progress.txt`:

```
Progress
========

[                    ] 0%

Target: 95% confidence

Iteration: 0/50
```

### Step 7: Attach and Display

```bash
tmux attach-session -t aether-colony
```

Before attaching, output:

```
+=====================================================+
|  AETHER COLONY :: WATCH                              |
+=====================================================+

tmux session 'aether-colony' created.

Layout:
  +-----------------+---------------------------+
  | Status          | Activity Log              |
  |                 |                           |
  +-----------------+                           |
  | Progress        |                           |
  +-----------------+---------------------------+

Commands:
  Ctrl+B D          Detach from session
  Ctrl+B [          Scroll mode (q to exit)
  tmux kill-session -t aether-colony   Stop watching

The session will update in real-time as colony works.
Attaching now...
```

---

## Status Update Protocol

Workers and commands update watch files as they work:

### Activity Log
Workers write via: `bash ~/.aether/aether-utils.sh activity-log "ACTION" "caste" "description"`

### Status File
Commands update `.aether/data/watch-status.txt` with current state:
- State: PLANNING, EXECUTING, READY
- Phase: current/total
- Active Workers: list of working castes
- Last Activity: most recent log entry

### Progress File
Planning loop updates `.aether/data/watch-progress.txt`:
- Progress bar based on confidence percentage
- Current iteration count
- Target threshold

---

## Cleanup

To stop watching:
```bash
tmux kill-session -t aether-colony
```

This stops the session but preserves all log files.
