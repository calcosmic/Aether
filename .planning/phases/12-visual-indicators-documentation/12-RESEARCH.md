# Phase 12: Visual Indicators & Documentation - Research

**Researched:** 2025-02-02
**Domain:** Terminal UI enhancement and documentation accuracy
**Confidence:** HIGH

## Summary

Phase 12 adds visual feedback to existing Aether commands without introducing new capabilities. The research focused on understanding the current state of Worker Ant status tracking, pheromone display patterns, and documentation path references. Key findings: Worker Ant status is already tracked in `worker_ants.json` with IDLE/ACTIVE states, `/ant:status` has a foundation but needs emoji indicators and visual grouping, and path references are inconsistent across utility scripts. The event bus from Phase 11 provides task_started/task_completed events that can be used for real-time status updates.

**Primary recommendation:** Enhance `/ant:status` with emoji-based activity states, add step progress indicators to multi-step commands, implement progress bars for pheromone strength, and audit all path references in `.aether/utils/` and `.claude/commands/ant/` for accuracy.

## Standard Stack

### Core
| Component | Version | Purpose | Why Standard |
|-----------|---------|---------|--------------|
| Bash native | 4.0+ | Emoji output and progress bars | No external dependencies, terminal-compatible |
| jq | 1.6+ | JSON parsing for status extraction | Already used throughout Aether |
| Unicode emojis | - | Visual status indicators | Terminal-compatible, accessible (paired with text) |
| Box-drawing characters | - | Visual structure (│, ╔, ═, etc.) | Creates professional terminal UI |

### Visual Indicators
| Indicator | Purpose | Usage |
|-----------|---------|-------|
| 🟢 / 🐜 | ACTIVE | Worker Ant currently executing task |
| ⚪ | IDLE | Worker Ant has no work |
| ⏳ | PENDING | Worker Ant waiting for work |
| 🔴 | ERROR | Critical error occurred |
| 🟡 | WARNING | Non-critical error/warning |
| ⚠️ | WARNING | Warning state |
| [━] | Progress bar | Pheromone signal strength (0.0-1.0) |
| [✓] | Completed | Step/task finished successfully |
| [→] | In progress | Step/task currently executing |
| [ ] | Pending | Step/task not started |
| [🔴] | Failed | Step/task failed |

### Data Files
| File | Purpose | Status Fields |
|------|---------|---------------|
| `.aether/data/worker_ants.json` | Worker Ant state tracking | `status` (IDLE/ACTIVE/ERROR/PENDING) |
| `.aether/data/pheromones.json` | Pheromone signals | `strength` (0.0-1.0) |
| `.aether/data/COLONY_STATE.json` | Colony state | `colony_status.state`, `worker_ants.*.status` |
| `.aether/data/events.json` | Event bus | `task_started`, `task_completed`, `task_failed` |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Unicode emojis | ANSI colors | Emojis work everywhere, colors require terminal support and NO_COLOR handling |
| Custom progress bars | `pv` command | Custom bars are simpler for 0-100% display, pv is for throughput |
| Manual step tracking | `progress` tool | Manual display is sufficient, external tool adds dependency |

**Installation:**
No installation required - uses existing bash, jq, and Unicode support

## Architecture Patterns

### Recommended Status Display Pattern

**What:** Consistent emoji + text format for accessibility
**When to use:** All status displays in `/ant:status` and command output

**Pattern:**
```bash
# Map status to emoji with text label
case $status in
  ACTIVE)   emoji="🟢"; label="ACTIVE" ;;
  IDLE)     emoji="⚪"; label="IDLE" ;;
  PENDING)  emoji="⏳"; label="PENDING" ;;
  ERROR)    emoji="🔴"; label="ERROR" ;;
  *)        emoji="❓"; label="$status" ;;
esac

echo "$name $emoji $label: $current_task"
```

### Pattern 1: Progress Bar for Pheromone Strength

**What:** Visual bar + numeric display for 0.0-1.0 range
**When to use:** Displaying pheromone signal strength in `/ant:status`

**Example:**
```bash
# Source: Bash progress bar best practices (Evil Martians 2024)
show_progress_bar() {
  local value=$1  # 0.0 to 1.0
  local width=20  # Bar width in characters

  # Calculate filled segments
  local filled=$((value * width))
  local empty=$((width - filled))

  # Build bar with box-drawing character
  local bar="["
  bar+=$(printf '━%.0s' $(seq 1 $filled))
  bar+=$(printf ' %.0s' $(seq 1 $empty))
  bar+="]"

  # Display with numeric value
  echo "$bar $(printf '%.2f' $value)"
}

# Usage in status display
strength=$(jq -r '.strength' <<< "$pheromone")
show_progress_bar "$strength"
# Output: [━━━━━━━━━━━━━━] 0.75
```

### Pattern 2: Step Indicators for Multi-Step Operations

**What:** List-style checkmarks showing all steps
**When to use:** Commands with 3+ steps (e.g., `/ant:build`, `/ant:init`)

**Example:**
```bash
# Track step state
steps=("Initialize" "Build" "Verify")
current_step=2

echo "Progress:"
for i in "${!steps[@]}"; do
  step_num=$((i + 1))
  if [ $step_num -lt $current_step ]; then
    echo "  [✓] ${steps[$i]}"
  elif [ $step_num -eq $current_step ]; then
    echo "  [→] ${steps[$i]}..."
  else
    echo "  [ ] ${steps[$i]}"
  fi
done

# Output:
#   [✓] Initialize
#   [→] Building...
#   [ ] Verifying
```

### Pattern 3: Status Dashboard with Grouping

**What:** Group Worker Ants by activity state with section headers
**When to use:** `/ant:status` command output

**Example:**
```bash
# Get all workers with their castes and status
jq -r '.worker_registry | to_entries[] | "\(.key)|\(.value.caste)|\(.value.status)"' "$WORKER_FILE" | while IFS='|' read -r name caste status; do
  case $status in
    ACTIVE)   active_workers+=("$name|$caste") ;;
    IDLE)     idle_workers+=("$name|$caste") ;;
    ERROR)    error_workers+=("$name|$caste") ;;
    PENDING)  pending_workers+=("$name|$caste") ;;
  esac
done

# Display grouped by status
echo "=== ACTIVE (${#active_workers[@]}) ==="
for worker in "${active_workers[@]}"; do
  IFS='|' read -r name caste <<< "$worker"
  task=$(jq -r ".worker_registry.$name.current_task" "$WORKER_FILE")
  echo "  🐜 $name ($caste): $task"
done

echo ""
echo "=== IDLE (${#idle_workers[@]}) ==="
for worker in "${idle_workers[@]}"; do
  IFS='|' read -r name caste <<< "$worker"
  echo "  ⚪ $name ($caste)"
done
```

### Pattern 4: Real-Time Status Updates via Event Polling

**What:** Use event bus to update Worker Ant status during execution
**When to use:** Worker Ant prompts that execute tasks

**Example:**
```bash
# Source event bus
source .aether/utils/event-bus.sh

# Publish task started event
publish_event "task_started" "start" '{"task": "Building component X"}' "builder" "builder"

# Update local status
jq '.worker_registry.builder.status = "ACTIVE" |
    .worker_registry.builder.current_task = "Building component X"' \
    "$WORKER_FILE" > /tmp/worker.tmp
atomic_write_from_file "$WORKER_FILE" /tmp/worker.tmp

# Execute task...
echo "🐜 Builder ACTIVE: Building component X"

# Publish task completed
publish_event "task_completed" "complete" '{"task": "Building component X"}' "builder" "builder"

# Update status back to IDLE
jq '.worker_registry.builder.status = "IDLE" |
    .worker_registry.builder.current_task = null' \
    "$WORKER_FILE" > /tmp/worker.tmp
atomic_write_from_file "$WORKER_FILE" /tmp/worker.tmp
```

### Anti-Patterns to Avoid

- **Emojis without text:** Always pair emoji with text label (e.g., "🟢 ACTIVE") for accessibility
- **ANSI colors without NO_COLOR check:** Don't use colors without checking NO_COLOR env var
- **Animated spinners for short tasks:** Only use spinners for 60+ second operations
- **Progress bars without numbers:** Always show numeric value alongside visual bar
- **Narrow progress bars:** Use 20+ characters for visibility on 80-column terminals

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Progress bar calculation | Custom math | Bash arithmetic with printf | Simple, no external dependencies |
| Status mapping | Complex switch statements | Case statement with default | Handles unknown states gracefully |
| Event-driven status updates | Polling worker_ants.json | Event bus publish/subscribe | Already implemented in Phase 11 |
| Path resolution | Hard-coded paths | Git root detection | Handles subdirectory execution |
| JSON updates | sed/awk string manipulation | jq atomic updates | Prevents corruption, already in use |

**Key insight:** The event bus infrastructure from Phase 11 provides task_started/task_completed events. Use these for real-time status updates instead of polling JSON files manually.

## Common Pitfalls

### Pitfall 1: Emojis Not Displaying Correctly

**What goes wrong:** Emojis show as boxes or question marks
**Why it happens:** Terminal doesn't support Unicode or locale not set correctly
**How to avoid:** Test emoji display in target terminal, fallback to text-only mode
**Warning signs:** User reports "boxes" instead of emojis

### Pitfall 2: Progress Bar Overflow

**What goes wrong:** Progress bar exceeds width or shows invalid characters
**Why it happens:** Value > 1.0 or calculation error
**How to avoid:** Clamp value to 0.0-1.0 range before display
**Warning signs:** Bar longer than specified width

### Pitfall 3: Status Desynchronization

**What goes wrong:** Worker Ant shows ACTIVE but task is complete
**Why it happens:** Status update forgotten after task completion
**How to avoid:** Always publish task_completed event and update status in finally block
**Warning signs:** Status stuck in ACTIVE state

### Pitfall 4: Path Reference Inconsistencies

**What goes wrong:** Scripts reference `.aether/data/file.json` but file is at `.aether/data/FILE.json`
**Why it happens:** Case sensitivity or path changes during refactoring
**How to avoid:** Use consistent casing, verify paths with `ls` before referencing
**Warning signs:** File not found errors when sourcing utilities

### Pitfall 5: Terminal Width Assumptions

**What goes wrong:** Progress bars wrap on narrow terminals
**Why it happens:** Assuming 80+ column width
**How to avoid:** Test on 80-column terminals, use 20-char bars for safety
**Warning signs:** Output wraps or displays incorrectly

## Code Examples

Verified patterns from current codebase and best practices:

### Status Mapping Function
```bash
# Source: Derived from current /ant:status patterns
get_status_emoji() {
  local status=$1
  case $status in
    ACTIVE)   echo "🟢" ;;
    IDLE)     echo "⚪" ;;
    PENDING)  echo "⏳" ;;
    ERROR)    echo "🔴" ;;
    *)        echo "❓" ;;
  esac
}

# Usage
status=$(jq -r '.worker_registry.builder.status' "$WORKER_FILE")
emoji=$(get_status_emoji "$status")
echo "Builder $emoji $status: Building components"
```

### Progress Bar Function
```bash
# Source: CLI UX best practices (Evil Martians 2024)
# https://evilmartians.com/chronicles/cli-ux-best-practices-3-patterns-for-improving-progress-displays
show_progress() {
  local value=$1  # 0.0 to 1.0
  local width=${2:-20}

  # Clamp value
  value=$(awk -v v="$value" 'BEGIN {print (v < 0 ? 0 : (v > 1 ? 1 : v))}')

  local filled=$((value * width))
  local empty=$((width - filled))

  printf "["
  printf "━%.0s" $(seq 1 $filled)
  printf " %.0s" $(seq 1 $empty)
  printf "] %.2f\n" "$value"
}

# Usage for pheromone strength
strength=$(jq -r '.strength' <<< "$pheromone")
show_progress "$strength"
# Output: [━━━━━━━━━━━━━━] 0.75
```

### Step Progress Tracker
```bash
# Source: Derived from /ant:build and /ant:phase patterns
declare -a STEPS=("Validate" "Initialize" "Execute" "Verify")
declare -a STEP_STATUS=("pending" "pending" "pending" "pending")

update_step() {
  local step_num=$1
  local status=$2  # pending, in_progress, completed, failed
  STEP_STATUS[$((step_num - 1))]=$status
}

show_progress() {
  for i in "${!STEPS[@]}"; do
    local step_num=$((i + 1))
    local step="${STEPS[$i]}"
    local status="${STEP_STATUS[$i]}"

    case $status in
      completed) echo "  [✓] Step $step_num/$step_num: $step" ;;
      in_progress) echo "  [→] Step $step_num/${#STEPS[@]}: $step..." ;;
      failed) echo "  [🔴] Step $step_num/${#STEPS[@]}: $step — failed" ;;
      *) echo "  [ ] Step $step_num/${#STEPS[@]}: $step" ;;
    esac
  done
}

# Usage during execution
update_step 1 "completed"
update_step 2 "in_progress"
show_progress
# Output:
#   [✓] Step 1/4: Validate
#   [→] Step 2/4: Initialize...
#   [ ] Step 3/4: Execute
#   [ ] Step 4/4: Verify
```

### Status Update Wrapper
```bash
# Source: Event bus patterns from Phase 11
# Wraps task execution with status updates
execute_with_status() {
  local worker_name=$1
  local caste=$2
  local task=$3
  local worker_file=".aether/data/worker_ants.json"

  # Publish task started
  source .aether/utils/event-bus.sh
  publish_event "task_started" "start" \
    "{\"task\": \"$task\", \"worker\": \"$worker_name\"}" \
    "$worker_name" "$caste"

  # Update status to ACTIVE
  jq ".worker_registry.$worker_name.status = \"ACTIVE\" |
      .worker_registry.$worker_name.current_task = \"$task\"" \
    "$worker_file" > /tmp/worker.tmp
  source .aether/utils/atomic-write.sh
  atomic_write_from_file "$worker_file" /tmp/worker.tmp

  echo "🐜 $worker_name ACTIVE: $task"

  # Execute task (function passed as argument)
  local result
  result=$("$@")  # Execute remaining arguments

  if [ $? -eq 0 ]; then
    # Success - publish completed
    publish_event "task_completed" "complete" \
      "{\"task\": \"$task\", \"worker\": \"$worker_name\"}" \
      "$worker_name" "$caste"

    # Update status to IDLE
    jq ".worker_registry.$worker_name.status = \"IDLE\" |
        .worker_registry.$worker_name.current_task = null" \
      "$worker_file" > /tmp/worker.tmp
    atomic_write_from_file "$worker_file" /tmp/worker.tmp

    echo "✅ $worker_name IDLE: Task complete"
  else
    # Failure - publish error
    publish_event "task_failed" "error" \
      "{\"task\": \"$task\", \"worker\": \"$worker_name\", \"error\": \"$result\"}" \
      "$worker_name" "$caste"

    # Update status to ERROR
    jq ".worker_registry.$worker_name.status = \"ERROR\" |
        .worker_registry.$worker_name.current_task = \"$task\"" \
      "$worker_file" > /tmp/worker.tmp
    atomic_write_from_file "$worker_file" /tmp/worker.tmp

    echo "🔴 $worker_name ERROR: $result"
  fi
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Text-only status | Emoji + text status | Phase 12 | Improved scannability, accessibility |
| Numeric only (75%) | Visual bar + numeric ([━━━] 0.75) | Phase 12 | Better visual feedback |
| Linear step display | List-style with checkmarks | Phase 12 | User sees full workflow |
| No activity feedback | Real-time status updates | Phase 12 | Transparency during execution |
| Inconsistent paths | Verified path references | Phase 12 | Fewer file not found errors |

**Deprecated/outdated:**
- 🏃 emoji for ACTIVE (being replaced with 🐜 for domain-specific clarity)
- Text-only step counters (being replaced with [✓]/[→]/[ ] indicators)
- Hard-coded paths without git root detection (being replaced with dynamic resolution)

## Open Questions

1. **Dashboard grouping strategy**
   - What we know: CONTEXT.md gives Claude discretion to choose grouping (by caste, activity state, or phase)
   - What's unclear: Which grouping is most useful for users
   - Recommendation: Group by activity state (ACTIVE/IDLE/ERROR/PENDING) with caste shown in each line - this matches the "at a glance" requirement

2. **Spinner threshold**
   - What we know: CONTEXT.md suggests 60+ seconds for animated spinners
   - What's unclear: Which operations in Aether exceed 60 seconds
   - Recommendation: Add spinner only to `/ant:build` and `/ant:execute` which spawn multiple workers - other commands complete quickly

3. **Path reference audit scope**
   - What we know: 26 utility scripts and 19 command files need path verification
   - What's unclear: How many paths are actually incorrect
   - Recommendation: Audit all `.aether/data/` references in commands and all `source` statements in utilities - use git root detection pattern from event-bus.sh

## Files Requiring Modification

### VISUAL-01: Worker Ant Activity States

**Files to modify:**
1. `.claude/commands/ant/status.md` (lines 68-99)
   - Add emoji mapping function
   - Update status display with 🟢/⚪/🏳/🔴 indicators
   - Add text labels for accessibility

2. `.claude/commands/ant/build.md` (lines 60-100)
   - Add step progress indicators
   - Show worker status during spawning

3. `.claude/commands/ant/execute.md` (lines 60-100)
   - Add real-time status updates
   - Display worker activity during execution

**Data structure:** `.aether/data/worker_ants.json` already has `status` field (IDLE/ACTIVE) in `worker_registry.*.status`

### VISUAL-02: Step Progress Indicators

**Files to modify:**
1. `.claude/commands/ant/init.md` (lines 19-150)
   - Add step tracker for 7-step initialization
   - Display [✓]/[→]/[ ] for each step

2. `.claude/commands/ant/build.md` (lines 95-200)
   - Add step progress for coordinator spawning
   - Show task completion status

3. `.claude/commands/ant/execute.md` (lines 74-150)
   - Add step counter for worker spawning
   - Display progress during execution

**Current step count:** 17 files use "## Step N" pattern, need step indicators added

### VISUAL-03: Visual Dashboard in /ant:status

**File to modify:**
1. `.claude/commands/ant/status.md` (entire file)
   - Add section headers: `=== ACTIVE ===`, `=== IDLE ===`, `=== ERROR ===`, `=== PENDING ===`
   - Group workers by status
   - Add colony metrics footer: total ants, active count, error count

**Current status:** Basic display exists (lines 68-99), needs visual enhancement

### VISUAL-04: Pheromone Signal Strength Bars

**File to modify:**
1. `.claude/commands/ant/status.md` (lines 101-116)
   - Add progress_bar() function
   - Replace text display with `[━━━━] 0.75` format
   - Apply to all active pheromones

**Data structure:** `.aether/data/pheromones.json` has `strength` field (0.0-1.0) in `active_pheromones[].strength`

### DOCS-01: Path References in .aether/utils/

**Files to audit (26 scripts):**
1. `.aether/utils/atomic-write.sh` - Paths: TEMP_DIR, BACKUP_DIR
2. `.aether/utils/checkpoint.sh` - Paths: CHECKPOINT_DIR, CHECKPOINT_FILE, COLONY_STATE, PHEROMONES_FILE, WORKER_ANTS_FILE
3. `.aether/utils/event-bus.sh` - Paths: EVENTS_FILE (uses git root detection ✓)
4. `.aether/utils/state-machine.sh` - Paths: COLONY_STATE
5. `.aether/utils/memory-ops.sh` - Paths: MEMORY_FILE
6. `.aether/utils/memory-search.sh` - Paths: MEMORY_FILE
7. All other utils - Check hard-coded `.aether/` paths

**Pattern to apply:** Use git root detection from `event-bus.sh`:
```bash
AETHER_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || echo "$PWD")"
EVENTS_FILE="$AETHER_ROOT/.aether/data/events.json"
```

### DOCS-02: Path References in .claude/commands/ant/

**Files to audit (19 commands):**
1. `.claude/commands/ant/adjust.md` - Source statements
2. `.claude/commands/ant/build.md` - Atomic write paths
3. `.claude/commands/ant/continue.md` - Source statements
4. `.claude/commands/ant/feedback.md` - Utility paths
5. `.claude/commands/ant/focus.md` - Source statements
6. `.claude/commands/ant/init.md` - Data file paths
7. `.claude/commands/ant/memory.md` - Source statements
8. `.claude/commands/ant/recover.md` - Checkpoint paths
9. `.claude/commands/ant/redirect.md` - Utility paths
10. `.claude/commands/ant/status.md` - Data file paths

**Pattern to verify:** All source statements should use correct relative paths:
```bash
source .aether/utils/atomic-write.sh  # Correct
source .aether/utils/checkpoint.sh   # Correct
```

**Known issues:** None confirmed - audit required

## Sources

### Primary (HIGH confidence)
- [CLI UX best practices: 3 patterns for improving progress displays](https://evilmartians.com/chronicles/cli-ux-best-practices-3-patterns-for-improving-progress-displays) - Verified patterns for progress bars and step indicators
- [Bash Reference Manual](https://www.gnu.org/s/bash/manual/bash.html) - Bash arithmetic and string manipulation
- [ANSI escape code](https://en.wikipedia.org/wiki/ANSI_escape_code) - Terminal output control (for reference, not using ANSI colors per CONTEXT.md decision)

### Secondary (MEDIUM confidence)
- [How to Write Better Bash Spinners](https://willcarh.art/blog/how-to-write-better-bash-spinners) - Spinner implementation patterns (validated against primary source)
- [Baeldung: How to Display a Spinner for Long Running Tasks in Bash](https://www.baeldung.com/linux/bash-show-spinner-long-tasks) - Spinner best practices (validated against primary source)
- [Stack Overflow: Using Bash to display a progress indicator](https://stackoverflow.com/questions/12498304/using-bash-to-display-a-progress-indicator-spinner) - Community patterns (validated against primary source)

### Tertiary (LOW confidence)
- [State of Terminal Emulators in 2025](https://news.ycombinator.com/item?id=45799478) - Terminal emoji support (mentioned but not verified)
- Various blog posts on bash scripting - Referenced for inspiration but not relied upon for critical claims

### Codebase Analysis (HIGH confidence)
- `.claude/commands/ant/status.md` - Current status display implementation
- `.aether/data/worker_ants.json` - Worker Ant status schema
- `.aether/data/pheromones.json` - Pheromone strength schema
- `.aether/data/COLONY_STATE.json` - Colony state and worker status
- `.aether/utils/event-bus.sh` - Event polling patterns (Phase 11)
- `.planning/phases/11-event-polling-integration/11-RESEARCH.md` - Event polling integration research

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Bash, jq, Unicode are standard tools, verified in codebase
- Architecture: HIGH - Patterns derived from existing codebase and verified best practices
- Pitfalls: MEDIUM - Based on common bash issues and codebase analysis, some unverified
- File modification list: HIGH - Direct analysis of codebase files

**Research date:** 2025-02-02
**Valid until:** 30 days (stable domain - bash terminal UI patterns don't change rapidly)

**Open questions for planner:**
1. Confirm dashboard grouping strategy (activity state vs caste vs phase)
2. Confirm which operations need animated spinners (if any)
3. Determine if path reference audit should be automated or manual
