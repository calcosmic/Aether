---
name: ant:recover
description: Recover colony from checkpoint - manual recovery or automatic crash detection
---

<objective>
Restore colony state from checkpoint, either manually by specifying checkpoint ID or automatically via crash detection.
</objective>

<process>
You are the **Queen Ant Colony** executing checkpoint recovery.

## Step 1: Source Dependencies

```bash
# Source utility scripts
source .aether/utils/atomic-write.sh
source .aether/utils/checkpoint.sh

# Set paths
COLONY_STATE=".aether/data/COLONY_STATE.json"
CHECKPOINT_DIR=".aether/data/checkpoints"
```

## Step 2: Check for Arguments

```bash
# Get checkpoint ID from arguments (if provided)
checkpoint_id="${1:-}"
```

## Step 3: List Mode (No Arguments)

If no checkpoint ID provided, show available checkpoints:

```bash
if [ -z "$checkpoint_id" ]; then
    echo "COLONY RECOVERY"
    echo ""

    # Show current colony state
    if [ -f "$COLONY_STATE" ]; then
        echo "Current colony state:"
        echo "  State: $(jq -r '.colony_status.state' "$COLONY_STATE")"
        echo "  Phase: $(jq -r '.colony_status.current_phase' "$COLONY_STATE")"
        echo ""
    fi

    # Show latest checkpoint
    if [ -f ".aether/data/checkpoint.json" ]; then
        latest_file=$(cat ".aether/data/checkpoint.json")
        if [ -f "$latest_file" ]; then
            echo "Latest checkpoint:"
            echo "  ID: $(jq -r '.checkpoint_id' "$latest_file")"
            echo "  Label: $(jq -r '.label' "$latest_file")"
            echo "  Timestamp: $(jq -r '.timestamp' "$latest_file")"
            echo "  State: $(jq -r '.colony_state.colony_status.state' "$latest_file")"
            echo ""
        fi
    fi

    # List all checkpoints
    list_checkpoints
    echo ""
    echo "Usage:"
    echo "  /ant:recover          - Show this list"
    echo "  /ant:recover latest   - Restore from latest checkpoint"
    echo "  /ant:recover [id]     - Restore from checkpoint ID"
fi
```

## Step 4: Recovery Mode (With Argument)

If checkpoint ID provided, perform recovery:

```bash
else
    echo "Recovering from checkpoint: $checkpoint_id"
    echo ""

    if load_checkpoint "$checkpoint_id"; then
        echo ""
        echo "Recovery complete. Colony restored."
        echo ""
        echo "Next steps:"
        echo "  - Review state: /ant:status"
        echo "  - Continue execution: /ant:execute $(jq -r '.colony_status.current_phase' "$COLONY_STATE")"
    else
        echo "Recovery failed. Check checkpoint ID and try again."
        exit 1
    fi
fi
```

## Important Notes

- The `load_checkpoint()` function handles both "latest" and numeric IDs
- Checkpoint integrity is validated before restoration
- All 4 colony state files are restored atomically:
  - COLONY_STATE.json
  - pheromones.json
  - worker_ants.json
  - memory.json
- Next steps are provided after successful recovery

</process>

<context>
# AETHER COLONY RECOVERY - Checkpoint System

## Checkpoint Storage

Checkpoints are stored in `.aether/data/checkpoints/` directory:
- File format: `checkpoint_N.json` where N is the checkpoint number
- Each checkpoint contains complete colony state:
  - colony_state (from COLONY_STATE.json)
  - pheromones (from pheromones.json)
  - worker_ants (from worker_ants.json)
  - memory (from memory.json)

## Checkpoint Reference

`.aether/data/checkpoint.json` contains the path to the latest checkpoint:
- Used by "latest" recovery option
- Updated atomically when checkpoints are saved

## Checkpoint Rotation

- System keeps only 10 most recent checkpoints
- Older checkpoints are automatically rotated out
- Checkpoints are created before and after state transitions

## Crash Detection

The colony includes automatic crash detection:
- Triggered when state is EXECUTING/VERIFYING but no active workers exist
- Triggered when state has been EXECUTING/VERIFYING for >30 minutes
- Automatic recovery restores from latest checkpoint
- Colony transitions to PLANNING state after recovery

## Recovery Process

1. **Checkpoint Selection**: Choose checkpoint ID or "latest"
2. **Integrity Check**: Validate JSON structure
3. **Atomic Restoration**: Restore all 4 state files atomically
4. **State Verification**: Confirm restored state
5. **Next Steps**: Provide recovery actions

## Available Checkpoints

Use `/ant:recover` to list all available checkpoints with:
- Checkpoint ID
- Size and timestamp
- Label (transition that created it)
</context>

<reference>
# Example Output: List Mode

```
COLONY RECOVERY

Current colony state:
  State: IDLE
  Phase: null

Latest checkpoint:
  ID: 4
  Label: post_IDLE_to_INIT
  Timestamp: 2026-02-01T17:42:36Z
  State: INIT

Available checkpoints:

  [4] 8.2K Feb 1 17:42 - post_IDLE_to_INIT
  [3] 8.1K Feb 1 17:34 - post_IDLE_to_INIT
  [2] 8.1K Feb 1 17:33 - post_IDLE_to_INIT
  [1] 7.8K Feb 1 15:47 - pre_IDLE_to_INIT
  [0] 7.8K Feb 1 15:47 - pre_IDLE_to_INIT

Usage:
  /ant:recover          - Show this list
  /ant:recover latest   - Restore from latest checkpoint
  /ant:recover [id]     - Restore from checkpoint ID
```

# Example Output: Recovery Mode

```
Recovering from checkpoint: latest

Restoring COLONY_STATE.json...
Restoring pheromones.json...
Restoring worker_ants.json...
Restoring memory.json...

Colony restored from checkpoint: .aether/data/checkpoints/checkpoint_4.json
Restored state: INIT
Restored phase: null

Next steps:
  - Review state: /ant:status
  - Continue phase: /ant:execute null

Recovery complete. Colony restored.

Next steps:
  - Review state: /ant:status
  - Continue execution: /ant:execute null
```

# Example Output: Automatic Crash Detection

```
Crash detected: State=EXECUTING but no active workers
Recovering from last checkpoint...
Restoring COLONY_STATE.json...
Restoring pheromones.json...
Restoring worker_ants.json...
Restoring memory.json...

Colony restored from checkpoint: .aether/data/checkpoints/checkpoint_5.json
Restored state: PLANNING
Restored phase: 1

State transition: EXECUTING -> PLANNING
Trigger: crash_recovery
Timestamp: 2026-02-01T17:50:00Z
```
</reference>

<allowed-tools>
Read
Bash
Glob
</allowed-tools>
