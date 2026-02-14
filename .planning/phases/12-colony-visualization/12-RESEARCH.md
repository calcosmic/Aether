# Phase 12: Colony Visualization - Research

**Researched:** 2026-02-14
**Domain:** Terminal UI, Real-time CLI Display, Activity Tracking, ASCII Art
**Confidence:** HIGH

## Summary

This research covers the implementation of immersive real-time colony visualization for Aether. The phase extends existing infrastructure (`/ant:swarm`, `/ant:watch`, telemetry system) with live activity display, collapsible views, and comprehensive metrics tracking.

**Key findings:**
1. The project already has solid foundations: `colorize-log.sh`, `watch-spawn-tree.sh`, `telemetry.js`, and `update-progress` in `aether-utils.sh`
2. Real-time display will use a **scrolling activity log pattern** (like `tail -f`) rather than a static dashboard
3. **picocolors** is already the chosen library for terminal colors (in `bin/lib/colors.js`)
4. Activity tracking data model exists in telemetry.json and spawn-tree.txt formats
5. Milestone detection is already implemented (`milestone-detect` command)

**Primary recommendation:** Build on existing infrastructure rather than introducing new dependencies. Extend `aether-utils.sh` with visualization commands, create new `ant:maturity` command file, and enhance existing colorization scripts.

## Standard Stack

### Core (Already in Project)
| Library/Tool | Version | Purpose | Why Standard |
|--------------|---------|---------|--------------|
| picocolors | ^1.1.1 | Terminal colors | Already used in `bin/lib/colors.js`, NO_COLOR friendly |
| bash | 3.2+ | Scripting | Existing infrastructure in `.aether/aether-utils.sh` |
| jq | 1.6+ | JSON processing | Used throughout for state manipulation |
| tmux | - | Multi-pane display | Already used in `/ant:watch` command |

### Data Sources (Existing)
| Source | Location | Contains |
|--------|----------|----------|
| COLONY_STATE.json | `.aether/data/` | Goal, phase, plan, memory, events |
| telemetry.json | `.aether/data/` | Model performance, routing decisions |
| spawn-tree.txt | `.aether/data/` | Parent-child relationships, status |
| activity.log | `.aether/data/` | Timestamped activity entries |
| flags.json | `.aether/data/` | Blockers, issues, notes |
| Chambers | `.aether/chambers/` | Entombed colony manifests |

### No New Dependencies Required
The project philosophy emphasizes minimal external dependencies. All visualization can be achieved with:
- ANSI escape codes (via picocolors or direct bash)
- File polling/watching (existing `watch-spawn-tree.sh` pattern)
- JSON processing (existing jq usage)

## Architecture Patterns

### Recommended Project Structure
```
.aether/
├── data/
│   ├── COLONY_STATE.json       # Existing - colony state
│   ├── telemetry.json          # Existing - model/caste performance
│   ├── spawn-tree.txt          # Existing - hierarchy data
│   ├── activity.log            # Existing - activity stream
│   ├── watch-status.txt        # Existing - tmux pane content
│   ├── watch-progress.txt      # Existing - progress display
│   └── swarm-display.json      # NEW - real-time swarm state
├── utils/
│   ├── colorize-log.sh         # Existing - activity colorization
│   ├── watch-spawn-tree.sh     # Existing - tree visualization
│   └── swarm-display.sh        # NEW - real-time swarm display
└── visualizations/             # NEW - ASCII art assets
    ├── anthill-stages/
    │   ├── first-mound.txt
    │   ├── open-chambers.txt
    │   ├── brood-stable.txt
    │   ├── ventilated-nest.txt
    │   ├── sealed-chambers.txt
    │   └── crowned-anthill.txt
    └── caste-sprites/          # Optional - caste-specific art
.claude/commands/ant/
├── swarm.md                    # Existing - to be enhanced
├── watch.md                    # Existing
├── status.md                   # Existing
└── maturity.md                 # NEW - milestone visualization
```

### Pattern 1: Scrolling Activity Log (VIZ-01, VIZ-06)
**What:** Real-time display showing ants currently working with caste emoji, updating like `tail -f`
**When to use:** During active colony operations (`/ant:swarm`, `/ant:build`)
**Implementation approach:**
```bash
# Source: Existing colorize-log.sh pattern
# New: swarm-display.sh will read from swarm-display.json

# Data flow:
# 1. Commands write to swarm-display.json (atomic writes)
# 2. swarm-display.sh polls file changes (fswatch/inotifywait/fallback polling)
# 3. Render with ANSI colors, emojis, and animations

# Format for swarm-display.json:
{
  "timestamp": "2026-02-14T20:00:00Z",
  "active_ants": [
    {
      "name": "Hammer-42",
      "caste": "builder",
      "status": "excavating",
      "task": "Implementing auth module",
      "tools": {"read": 5, "grep": 3, "edit": 2, "bash": 1},
      "tokens": 1250,
      "started_at": "2026-02-14T19:55:00Z",
      "parent": "Prime"
    }
  ],
  "summary": {
    "total_active": 3,
    "by_caste": {"builder": 2, "scout": 1}
  }
}
```

### Pattern 2: Collapsible Tunnel View (VIZ-02)
**What:** Expand/collapse to see nested agent spawns
**When to use:** In spawn tree visualization, swarm detail view
**Implementation approach:**
```bash
# Source: Existing watch-spawn-tree.sh pattern
# Enhancement: Add "collapsed" state per parent

# Display format:
# 👑 Queen
# │
# ├── 🔨 Hammer-42: Implementing auth [depth 1] [+2 hidden] ▼
# │   ├── 🔍 Swift-7: Finding patterns [depth 2]
# │   └── 🔍 Dash-3: Researching docs [depth 2]
# └── 🔨 Forge-12: Fixing tests [depth 1] ▶
#
# Interaction: Click/keyboard to expand/collapse
# State stored in: .aether/data/view-state.json
```

### Pattern 3: Progress Bar with Animation (VIZ-08)
**What:** Live excavation progress bars for long-running operations
**When to use:** Phase builds, swarm operations, long tasks
**Implementation approach:**
```bash
# Source: Existing update-progress in aether-utils.sh
# Enhancement: Add animated text indicator

# Current implementation (aether-utils.sh:438-488):
# - ASCII progress bar with █ and ░
# - Spinner frames (⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏)
# - Status icons (✅ 🔨)

# New: Add animated text phrases
phrases=("excavating..." "foraging..." "tunneling..." "carrying...")
# Cycle through based on caste and elapsed time
```

### Pattern 4: Chamber Activity Map (VIZ-07)
**What:** Show which nest zones have active ants
**When to use:** Real-time status, summary views
**Implementation approach:**
```bash
# Zone mapping (from CONTEXT.md):
# - Fungus Garden 🍄 (food/research)
# - Nursery 🥚 (new ants/spawns)
# - Refuse Pile 🗑️ (errors/failures)
# - Throne Room 👑 (Queen/Prime)
# - Foraging Trail 🌿 (active work)

# Display format:
# 🍄 Fungus Garden 🔥🔥 (3 ants researching)
# 🥚 Nursery 🔥 (1 ant spawning)
# 🗑️ Refuse Pile (0 ants)
# 👑 Throne Room (Queen observing)
# 🌿 Foraging Trail 🔥🔥🔥 (5 ants working)
#
# Fire intensity = number of ants (1-2:🔥, 3-4:🔥🔥, 5+:🔥🔥🔥)
# Empty zones hidden by default
```

### Pattern 5: Caste Color + Emoji (VIZ-09)
**What:** Distinct color per caste AND emoji together
**When to use:** Everywhere caste is displayed
**Implementation approach:**
```javascript
// Source: bin/lib/colors.js and CONTEXT.md
// Mapping locked by decision:

const casteStyles = {
  builder:  { color: 'blue',   emoji: '🔨', ansi: '\033[34m' },
  watcher:  { color: 'green',  emoji: '👁️',  ansi: '\033[32m' },
  scout:    { color: 'yellow', emoji: '🔍', ansi: '\033[33m' },
  chaos:    { color: 'red',    emoji: '🎲', ansi: '\033[31m' },
  prime:    { color: 'magenta',emoji: '👑', ansi: '\033[35m' }
};

// Display format: "🔨 Builder" where BOTH are colored
// Parent ants: bold + underline
// Completed ants: gray/dim
```

### Pattern 6: ASCII Art Anthill (LIFE-06)
**What:** Visual representation of colony maturity journey
**When to use:** `/ant:maturity` command
**Implementation approach:**
```
# 6 stages, 40+ lines each, intricate detail
# Stage 1: First Mound (small, simple)
# Stage 2: Open Chambers (tunnels visible)
# Stage 3: Brood Stable (eggs visible)
# Stage 4: Ventilated Nest (complex tunnels)
# Stage 5: Sealed Chambers (protected)
# Stage 6: Crowned Anthill (grand, ornate)

# Files stored in: .aether/visualizations/anthill-stages/
# Selected based on milestone-detect output
```

### Pattern 7: Chamber Comparison (LIFE-07)
**What:** Compare pheromone trails across two entombed colonies
**When to use:** `/ant:tunnels` with comparison mode
**Implementation approach:**
```bash
# Side-by-side diff format:
# ┌─────────────────┬─────────────────┐
# │ Chamber A       │ Chamber B       │
# │ v1.2.3          │ v2.0.0          │
# ├─────────────────┼─────────────────┤
# │ 🍄 12 trails    │ 🍄 15 trails    │
# │ 🥚 3 brood      │ 🥚 8 brood      │
# │ 🗑️ 2 errors     │ 🗑️ 0 errors     │
# └─────────────────┴─────────────────┘
#
# Data from chamber manifests (chamber-utils.sh)
```

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Terminal colors | Custom ANSI sequences | picocolors (existing) | NO_COLOR support, tested, minimal |
| Progress bars | Custom drawing | Existing update-progress | Already handles spinner, icons, formatting |
| JSON state updates | Direct file writes | atomic_write (existing) | Prevents corruption during concurrent access |
| File watching | Custom polling | fswatch/inotifywait with fallback | Existing pattern in watch-spawn-tree.sh |
| Caste detection | String parsing | get_caste_emoji (existing) | Comprehensive name matching already implemented |
| Milestone detection | Custom logic | milestone-detect command | Already computes based on phases completed |
| Activity logging | Direct echo | activity-log command | Consistent format, emoji injection |
| Spawn tracking | Manual file appends | spawn-log/spawn-complete | Parent-child hierarchy, status tracking |

**Key insight:** The project has invested heavily in utility functions. Custom solutions would duplicate effort and risk inconsistency with existing patterns.

## Common Pitfalls

### Pitfall 1: Emoji Width in Terminals
**What goes wrong:** Emojis like 👁️ (eye) render as 2 columns but some terminals count as 1, causing misalignment
**Why it happens:** Unicode width ambiguity, terminal differences
**How to avoid:**
- Test alignment in both iTerm2 and Terminal.app
- Use `string-width` logic or pad conservatively
- Avoid emoji in fixed-width columns
**Warning signs:** Progress bars or tables look "off" by 1-2 characters

### Pitfall 2: TTY Detection for Colors
**What goes wrong:** Colors appear in piped output, breaking scripts that parse CLI output
**Why it happens:** Not checking `isatty()` or `NO_COLOR`
**How to avoid:**
- Use existing `isColorEnabled()` in `bin/lib/colors.js`
- Respect `--no-color` flag
- Respect `NO_COLOR` environment variable
**Warning signs:** `aether status | grep something` shows ANSI codes

### Pitfall 3: File Polling CPU Usage
**What goes wrong:** High CPU usage from aggressive polling when fswatch/inotifywait unavailable
**Why it happens:** Fallback polling too frequent
**How to avoid:**
- Use 2-second interval for fallback (existing pattern)
- Only poll when display is active
- Cache file stat to detect changes without reading
**Warning signs:** Terminal feels sluggish, battery drain

### Pitfall 4: Concurrent State Updates
**What goes wrong:** Corrupted JSON when multiple ants write to swarm-display.json simultaneously
**Why it happens:** No file locking during read-modify-write
**How to avoid:**
- Use atomic_write pattern (write temp, rename)
- For complex updates, use file locking (existing lock utilities)
- Consider separate files per ant, aggregated on read
**Warning signs:** JSON parse errors, missing ants in display

### Pitfall 5: Terminal Clear/Redraw Flicker
**What goes wrong:** Visible flickering during updates in real-time display
**Why it happens:** Full clear + redraw instead of in-place updates
**How to avoid:**
- Use ANSI cursor positioning (\033[H to go home)
- Only redraw changed lines
- For simple cases, `clear` is acceptable
**Warning signs:** Display "flashes" or flickers during updates

### Pitfall 6: Nested Spawn Depth Explosion
**What goes wrong:** Display becomes unreadable with deeply nested spawns (depth 5+)
**Why it happens:** Indentation accumulates, horizontal space exhausted
**How to avoid:**
- Collapse by default at depth 3+
- Show "...and N more" for deep nesting
- Limit total displayed ants to screen height
**Warning signs:** Lines wrap, tree becomes unreadable

## Code Examples

### Real-time Display Script (swarm-display.sh)
```bash
#!/bin/bash
# Real-time swarm activity display
# Usage: bash swarm-display.sh [swarm_id]

SWARM_ID="${1:-current}"
DATA_DIR="${DATA_DIR:-.aether/data}"
DISPLAY_FILE="$DATA_DIR/swarm-display.json"

# ANSI colors (matching existing patterns)
BLUE='\033[34m'
GREEN='\033[32m'
YELLOW='\033[33m'
RED='\033[31m'
MAGENTA='\033[35m'
BOLD='\033[1m'
UNDERLINE='\033[4m'
DIM='\033[2m'
RESET='\033[0m'

# Caste colors (from CONTEXT.md decisions)
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

# Caste emojis (from aether-utils.sh)
get_caste_emoji() {
  case "$1" in
    builder)  echo "🔨" ;;
    watcher)  echo "👁️" ;;
    scout)    echo "🔍" ;;
    chaos)    echo "🎲" ;;
    prime)    echo "👑" ;;
    *)        echo "🐜" ;;
  esac
}

# Render active ants
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

  if [[ ! -f "$DISPLAY_FILE" ]]; then
    echo -e "${DIM}Waiting for swarm activity...${RESET}"
    return
  fi

  # Read and display active ants
  jq -r '.active_ants[] |
    "\(.name)|\(.caste)|\(.status)|\(.task)|\(.tokens)"' "$DISPLAY_FILE" 2>/dev/null | \
  while IFS='|' read -r name caste status task tokens; do
    color=$(get_caste_color "$caste")
    emoji=$(get_caste_emoji "$caste")

    # Parent ants: bold + underline
    if [[ "$name" == "Prime"* ]] || [[ "$name" == "Queen"* ]]; then
      style="${BOLD}${UNDERLINE}"
    else
      style="${BOLD}"
    fi

    echo -e "${color}${emoji} ${style}${name}${RESET}${color}: ${status} ${task} ${DIM}(🍯 ${tokens})${RESET}"
  done

  # Summary line
  total=$(jq '.summary.total_active // 0' "$DISPLAY_FILE" 2>/dev/null)
  echo ""
  echo -e "${DIM}${total} foragers excavating...${RESET}"
}

# Main loop
render_swarm
if command -v fswatch &>/dev/null; then
  fswatch -o "$DISPLAY_FILE" 2>/dev/null | while read; do render_swarm; done
else
  while true; do sleep 1; render_swarm; done
fi
```

### Activity Data Structure (swarm-display.json)
```json
{
  "timestamp": "2026-02-14T20:00:00Z",
  "swarm_id": "swarm-1739568000",
  "active_ants": [
    {
      "name": "Hammer-42",
      "caste": "builder",
      "status": "excavating",
      "task": "Implementing auth module",
      "parent": "Prime",
      "depth": 1,
      "started_at": "2026-02-14T19:55:00Z",
      "tools": {
        "read": 5,
        "grep": 3,
        "edit": 2,
        "bash": 1
      },
      "tokens": 1250,
      "completed": false
    }
  ],
  "summary": {
    "total_active": 3,
    "by_caste": {
      "builder": 2,
      "scout": 1
    },
    "by_zone": {
      "fungus_garden": 2,
      "foraging_trail": 1
    }
  },
  "chambers": {
    "fungus_garden": {"activity": 2, "icon": "🍄"},
    "nursery": {"activity": 0, "icon": "🥚"},
    "refuse_pile": {"activity": 0, "icon": "🗑️"},
    "throne_room": {"activity": 1, "icon": "👑"},
    "foraging_trail": {"activity": 1, "icon": "🌿"}
  }
}
```

### Tool Usage Tracking Integration
```javascript
// Extend telemetry.js to track tool usage per spawn
// Add to recordSpawnTelemetry():

const decision = {
  timestamp: decisionTimestamp,
  task: task || 'unknown',
  caste: caste || 'unknown',
  selected_model: model || 'default',
  source: source || 'unknown',
  tools: { read: 0, grep: 0, edit: 0, bash: 0 }, // Initialize counters
  tokens: 0
};

// New function to update tool usage
function updateToolUsage(repoPath, spawnId, toolType, count = 1) {
  const data = loadTelemetry(repoPath);
  const decision = data.routing_decisions.find(d => d.timestamp === spawnId);
  if (decision && decision.tools) {
    decision.tools[toolType] = (decision.tools[toolType] || 0) + count;
    saveTelemetry(repoPath, data);
  }
}

// New function to update token consumption (trophallaxis)
function updateTokenUsage(repoPath, spawnId, tokens) {
  const data = loadTelemetry(repoPath);
  const decision = data.routing_decisions.find(d => d.timestamp === spawnId);
  if (decision) {
    decision.tokens = (decision.tokens || 0) + tokens;
    saveTelemetry(repoPath, data);
  }
}
```

### Collapsible View State Management
```bash
# View state stored in .aether/data/view-state.json
# Tracks which parents are expanded/collapsed

{
  "swarm_display": {
    "expanded": ["Hammer-42", "Forge-12"],
    "collapsed": ["Mason-8"],
    "default_expand_depth": 2
  },
  "tunnel_view": {
    "expanded": [],
    "collapsed": ["all_depth_3_plus"],
    "show_completed": false
  }
}

# Toggle function in aether-utils.sh
view-toggle-expand() {
  local view="$1"  # swarm_display or tunnel_view
  local item="$2"  # ant name or pattern
  # Read current state, toggle item, write back
}
```

### ASCII Art Anthill Stage Example (first-mound.txt)
```
       .-.
      (o o)  AETHER COLONY
      | O |  First Mound
       `-`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

              .      .
             /|\    /|\
            / | \  / | \
           /  |  \/  |  \
          /   |   /\  |   \
         /____|__/___\|____\
        /      /     \      \
       /______/_______\______\
      /________________________\
     /__________________________\

🐜 A humble beginning...
   Just a small mound of earth,
   but full of potential.

Milestone: First Mound
Phases: 0 completed
Colony Age: Newborn
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Static status display | Real-time scrolling log | Phase 12 | More immersive, better for long operations |
| Simple progress % | Progress bar + animation | Phase 12 | Better visual feedback |
| Text-only caste labels | Color + emoji together | Phase 12 | Instant visual recognition |
| Flat worker list | Hierarchical tree view | Phase 12 | Shows orchestration structure |
| Single milestone | 6-stage maturity journey | Phase 12 | Gamification, progression sense |

**Deprecated/outdated:**
- Old caste colors in `colorize-log.sh` (yellow for Builder) - update to match CONTEXT.md (blue for Builder)
- Static tmux panes - enhance with real-time updates

## Open Questions

1. **Animation Speed for Progress Indicators**
   - What we know: Spinner uses 10 frames, cycles every second
   - What's unclear: Optimal phrase rotation speed for "excavating..." text
   - Recommendation: Start with 3-second rotation, adjust based on user feedback

2. **Token Count Granularity**
   - What we know: Telemetry tracks per-spawn, need per-task display
   - What's unclear: Whether to show cumulative or incremental updates
   - Recommendation: Show cumulative with delta indicator (+50)

3. **Collapsible View Interaction**
   - What we know: Need expand/collapse for nested spawns
   - What's unclear: Keyboard shortcuts vs mouse interaction in terminal
   - Recommendation: Start with keyboard (e+number to expand, c+number to collapse)

4. **Chamber Zone Assignment Logic**
   - What we know: 5 zones defined with emojis
   - What's unclear: How to map tasks to zones automatically
   - Recommendation: Keyword-based mapping ("research"->Fungus Garden, "fix"->Refuse Pile)

## Sources

### Primary (HIGH confidence)
- `/Users/callumcowie/repos/Aether/.planning/phases/12-colony-visualization/12-CONTEXT.md` - Phase decisions and constraints
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` - Existing utility functions (activity-log, spawn-log, update-progress, milestone-detect)
- `/Users/callumcowie/repos/Aether/runtime/utils/colorize-log.sh` - Terminal colorization patterns
- `/Users/callumcowie/repos/Aether/runtime/utils/watch-spawn-tree.sh` - Tree visualization implementation
- `/Users/callumcowie/repos/Aether/bin/lib/colors.js` - picocolors usage patterns
- `/Users/callumcowie/repos/Aether/bin/lib/telemetry.js` - Telemetry data model
- `/Users/callumcowie/repos/Aether/.aether/utils/chamber-utils.sh` - Chamber/entombment utilities

### Secondary (MEDIUM confidence)
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/swarm.md` - Current swarm command structure
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/watch.md` - Tmux pane layout pattern
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/status.md` - Status display patterns

### Tertiary (LOW confidence)
- General terminal UI best practices (common knowledge)
- ANSI escape code behavior across terminals (empirical testing needed)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All tools already in project
- Architecture: HIGH - Builds on existing patterns
- Pitfalls: MEDIUM - Based on common terminal UI issues, limited project-specific validation
- Caste colors: HIGH - Explicitly defined in CONTEXT.md
- ASCII art design: LOW - Creative element, needs iterative refinement

**Research date:** 2026-02-14
**Valid until:** 2026-03-14 (30 days for stable terminal patterns)

## RESEARCH COMPLETE

**Phase:** 12 - Colony Visualization
**Confidence:** HIGH

### Key Findings

1. **No new dependencies needed** - picocolors, bash, jq, tmux already in use
2. **Scrolling activity log pattern** chosen over static dashboard (per CONTEXT.md)
3. **Existing infrastructure to extend:** colorize-log.sh, watch-spawn-tree.sh, telemetry.js, milestone-detect
4. **Caste colors locked:** Builder=blue, Watcher=green, Scout=yellow, Chaos=red, Prime=purple
5. **Six milestone stages** for ASCII art progression: First Mound → Crowned Anthill

### File Created

`.planning/phases/12-colony-visualization/12-RESEARCH.md`

### Confidence Assessment

| Area | Level | Reason |
|------|-------|--------|
| Standard Stack | HIGH | All tools verified in package.json and existing code |
| Architecture | HIGH | Extends proven patterns from watch-spawn-tree.sh, colorize-log.sh |
| Pitfalls | MEDIUM | Common terminal UI issues, needs testing on target terminals |
| Implementation Approach | HIGH | Clear path based on existing aether-utils.sh commands |

### Open Questions

1. Animation speed for progress text (recommendation: 3-second rotation)
2. Token count display format (recommendation: cumulative with delta)
3. Collapsible interaction method (recommendation: keyboard shortcuts)
4. Chamber zone auto-assignment (recommendation: keyword-based)

### Ready for Planning

Research complete. Planner can now create PLAN.md files with confidence in the approach.
