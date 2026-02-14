# Phase 10: Entombment & Egg Laying - Research

**Researched:** 2026-02-14
**Domain:** Colony lifecycle management, state archival, milestone detection
**Confidence:** HIGH

## Summary

Phase 10 implements colony lifecycle management: archiving completed colonies (entombment), starting fresh colonies (laying eggs), browsing history (exploring tunnels), and automatic milestone detection. This is the core "colony lifecycle" feature that enables users to preserve learnings while starting fresh.

**Primary recommendation:** Implement flat chamber structure with `{goal}-{timestamp}` naming, copy-then-verify safety pattern, and automatic milestone detection based on phases completed and state fields.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Node.js fs | built-in | File operations | Standard for CLI, supports recursive operations |
| Node.js path | built-in | Path manipulation | Cross-platform compatibility |
| Bash | system | Shell utilities | Existing pattern in aether-utils.sh |
| jq | system | JSON processing | Existing dependency, used throughout |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| crypto (Node) | built-in | Hash verification | For manifest integrity checks |
| readline (Node) | built-in | User confirmation | For interactive prompts |

**No additional dependencies required** — all functionality achievable with existing stack.

## Architecture Patterns

### Recommended Chamber Structure

```
.aether/
├── chambers/                    # Entombed colonies (flat structure)
│   ├── add-user-auth-2026-02-14T153022Z/
│   │   ├── COLONY_STATE.json   # Archived state (minimal)
│   │   └── manifest.json       # Pheromone trail metadata
│   ├── fix-loop-bugs-2026-02-13T204000Z/
│   │   ├── COLONY_STATE.json
│   │   └── manifest.json
│   └── ...
├── data/                        # Current active colony
│   └── COLONY_STATE.json
└── ...
```

**Why flat over nested:**
- Simpler to implement and browse
- No need to manage milestone-based directory hierarchy
- Easier to list and filter
- Aligns with "tunnels" metaphor (linear exploration)

### Pattern 1: Copy-Then-Verify Safety
**What:** Copy files to destination, verify manifest integrity, then remove source
**When to use:** All destructive operations (entombment, laying eggs)
**Example:**
```bash
# From aether-utils.sh (existing pattern)
# 1. Copy to destination
mkdir -p "$chamber_dir"
cp "$DATA_DIR/COLONY_STATE.json" "$chamber_dir/"

# 2. Create and verify manifest
write_manifest "$chamber_dir"
verify_manifest "$chamber_dir"

# 3. Only then remove source
rm -f "$DATA_DIR/COLONY_STATE.json"
```

### Pattern 2: State Reset with Pheromone Preservation
**What:** Clear progress fields but preserve learnings/decisions
**When to use:** When laying eggs (starting fresh colony)
**Fields to preserve:**
- `memory.phase_learnings` (validated learnings)
- `memory.decisions` (architectural decisions)
- `memory.instincts` (high-confidence instincts)

**Fields to reset:**
- `goal` (new goal required)
- `state` → "READY"
- `current_phase` → 0
- `session_id` (new session)
- `plan.phases` → []
- `errors` → {records: [], flagged_patterns: []}
- `signals` → []
- `graveyards` → []
- `events` → [colony_initialized event]

### Pattern 3: Milestone Auto-Detection
**What:** Compute milestone from colony state automatically
**When to use:** On any state change, display in status
**Algorithm:**
```javascript
function detectMilestone(state) {
  const phases = state.plan?.phases || [];
  const completedCount = phases.filter(p => p.status === 'completed').length;
  const totalPhases = phases.length;

  // Check for failed colony (has critical errors)
  const hasCriticalErrors = state.errors?.records?.some(e => e.severity === 'critical');
  if (hasCriticalErrors) return { name: 'Failed Mound', version: 'v0.0' };

  // Check completion status
  const allCompleted = totalPhases > 0 && completedCount === totalPhases;

  if (allCompleted) {
    // Check if explicitly sealed
    if (state.milestone === 'Crowned Anthill') {
      return { name: 'Crowned Anthill', version: computeVersion(phases) };
    }
    return { name: 'Sealed Chambers', version: computeVersion(phases) };
  }

  // Progress-based milestones
  if (completedCount >= 5) {
    return { name: 'Ventilated Nest', version: computeVersion(phases) };
  }
  if (completedCount >= 3) {
    return { name: 'Brood Stable', version: computeVersion(phases) };
  }
  if (completedCount >= 1) {
    return { name: 'Open Chambers', version: computeVersion(phases) };
  }

  return { name: 'First Mound', version: 'v0.1' };
}

function computeVersion(phases) {
  const major = Math.floor(phases.length / 10);
  const minor = phases.length % 10;
  const patch = phases.filter(p => p.status === 'completed').length;
  return `v${major}.${minor}.${patch}`;
}
```

### Pattern 4: Command Structure (from existing commands)
**What:** Consistent command pattern across all ant commands
**Structure:**
```markdown
---
name: ant:<command>
description: "<emoji> <description>"
---

You are the <Role>. <Brief description>.

## Instructions

### Step 1: Read State
Read `.aether/data/COLONY_STATE.json`
Handle missing file or null goal

### Step 2: Validate Preconditions
Check colony state, phase status, etc.
Stop with clear message if preconditions fail

### Step 3: User Confirmation (for destructive ops)
Always require explicit confirmation
Show what will happen

### Step 4: Execute Operation
Perform the action with error handling

### Step 5: Update State
Write changes to COLONY_STATE.json

### Step 6: Display Result
Show success/failure with relevant details
```

### Anti-Patterns to Avoid
- **Don't archive failed colonies:** Only completed/collected colonies should be entombed
- **Don't allow multi-colony:** Single active colony only — laying eggs implies destructive transition
- **Don't spawn from archive:** Entombed colonies are read-only
- **Don't skip verification:** Always verify manifest after copy
- **Don't lose pheromones:** Preserve learnings/decisions when laying eggs

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| File copying with verification | Custom copy logic | `cp` + hash comparison | Edge cases with permissions, symlinks |
| JSON validation | Schema validator | jq + simple checks | Overkill for simple structure validation |
| Date formatting | Custom formatters | `date -u +%Y-%m-%dT%H:%M:%SZ` | ISO-8601 standard, consistent |
| Directory listing | Custom recursion | `ls` / `find` with sorting | Already battle-tested |
| User confirmation | Custom prompts | Simple echo + read pattern | Minimal, works everywhere |

## Common Pitfalls

### Pitfall 1: Data Loss During Entombment
**What goes wrong:** Power loss or error during copy leaves colony in inconsistent state
**Why it happens:** Copy not atomic, cleanup happens before verification
**How to avoid:**
- Copy to temp location first
- Verify manifest integrity
- Atomic move to final location
- Only then clean up source
**Warning signs:** Partial chamber directory, missing manifest.json

### Pitfall 2: Pheromone Loss When Laying Eggs
**What goes wrong:** Fresh colony starts with empty memory, losing accumulated wisdom
**Why it happens:** Over-eager reset clears all fields
**How to avoid:** Explicit whitelist of fields to preserve vs reset
**Warning signs:** Colony makes same mistakes as previous colonies

### Pitfall 3: Duplicate Chamber Names
**What goes wrong:** Same goal + timestamp collision (rare but possible)
**Why it happens:** Rapid successive entombments
**How to avoid:** Append counter if directory exists: `{goal}-{timestamp}-{n}`
**Warning signs:** Directory already exists error

### Pitfall 4: Milestone Stagnation
**What goes wrong:** Milestone doesn't update as phases complete
**Why it happens:** Detection only runs on explicit commands
**How to avoid:** Detect on every status check, store in state
**Warning signs:** Status shows old milestone despite progress

### Pitfall 5: Tunnel Browsing Performance
**What goes wrong:** Listing many chambers becomes slow
**Why it happens:** Loading every manifest.json for summary
**How to avoid:** Cache tunnel index, lazy-load details
**Warning signs:** Slow `/ant:tunnels` response

## Code Examples

### Entombment Command Structure
```markdown
---
name: ant:entomb
description: "🏺🐜🏺 Entomb completed colony in chambers"
---

You are the **Queen**. Archive the completed colony.

## Instructions

### Step 1: Read State
Read `.aether/data/COLONY_STATE.json`
If missing or goal is null: "No colony to entomb. Run /ant:init first."

### Step 2: Validate Colony Can Be Entombed
Check:
- All phases have status "completed" (or no phases exist)
- State is not "EXECUTING"
- No unresolved blockers (optional but recommended)

If not complete: "Cannot entomb incomplete colony. Run /ant:continue to complete phases."

### Step 3: Compute Milestone
Run milestone detection to determine final milestone
Should be "Sealed Chambers" or "Crowned Anthill"

### Step 4: User Confirmation
Display:
```
Entomb colony: "{goal}"
  Phases: {completed}/{total} completed
  Milestone: {milestone}
  Archive will include: COLONY_STATE.json, manifest.json

This will reset the active colony. Continue? (yes/no)
```

Require explicit "yes" to proceed.

### Step 5: Create Chamber
Generate chamber name: `{sanitized_goal}-{timestamp}`
Sanitize: lowercase, replace spaces with hyphens, remove special chars
Timestamp: ISO-8601 basic format (no colons)

Create directory: `.aether/chambers/{chamber_name}/`

### Step 6: Copy and Create Manifest
Copy `COLONY_STATE.json` to chamber
Create `manifest.json`:
```json
{
  "entombed_at": "2026-02-14T15:30:22Z",
  "goal": "original goal",
  "phases_completed": 5,
  "total_phases": 5,
  "milestone": "Sealed Chambers",
  "version": "v0.5.5",
  "decisions": [...],
  "learnings": [...],
  "files": {
    "COLONY_STATE.json": "sha256:..."
  }
}
```

### Step 7: Verify and Cleanup
Verify manifest integrity (hashes match)
If verification fails: "Entombment failed - verification error. Chamber preserved at {path}"

If verified:
- Backup: `mv COLONY_STATE.json COLONY_STATE.json.bak`
- Reset state (see Pattern 2)
- Remove backup

### Step 8: Display Result
```
🏺 ═══════════════════════════════════════════════════
   C O L O N Y   E N T O M B E D
══════════════════════════════════════════════════ 🏺

✅ Colony archived successfully

👑 Goal: {goal}
📍 Phases: {completed} completed
🏆 Milestone: {milestone}

📦 Chamber: .aether/chambers/{chamber_name}/

🐜 The colony rests. Its learnings are preserved.
   Run /ant:lay-eggs to begin anew.
```
```

### Lay Eggs Command Structure
```markdown
---
name: ant:lay-eggs
description: "🥚🐜🥚 Lay first eggs of new colony"
---

You are the **Queen**. Begin a new colony, preserving pheromones.

## Instructions

### Step 1: Check Current Colony
Read `.aether/data/COLONY_STATE.json`

If colony exists with goal and phases:
Display: "Active colony exists: {goal}. Run /ant:entomb first to archive."
Stop here.

### Step 2: Validate Goal
If `$ARGUMENTS` is empty:
Display usage with examples
Stop here.

### Step 3: Check for Prior Knowledge
Check `.aether/data/completion-report.md` exists
If yes: extract instincts (confidence >= 0.5) and learnings

### Step 4: Create New Colony State
Generate new session_id and timestamps
Preserve from prior: memory.phase_learnings, memory.decisions, memory.instincts
Reset everything else per Pattern 2

### Step 5: Display Result
```
🥚 ═══════════════════════════════════════════════════
   F I R S T   E G G S   L A I D
══════════════════════════════════════════════════ 🥚

👑 New colony goal: {goal}
📋 Session: {session_id}

{If inherited knowledge:}
🧠 Inherited from prior colonies:
   {N} instinct(s) | {N} learning(s)
{End if}

🏆 Milestone: First Mound (v0.1)

🐜 The colony begins anew.
   Run /ant:plan to chart the course.
```
```

### Tunnels Command Structure
```markdown
---
name: ant:tunnels
description: "🕳️🐜🕳️ Explore tunnels (browse archived colonies)"
---

You are the **Queen**. Browse the colony history.

## Instructions

### Step 1: List Chambers
Check `.aether/chambers/` exists
If not: "No chambers found. Archive colonies with /ant:entomb first."

List all directories in chambers/
Sort by entombed_at (from manifest.json) descending

### Step 2: Load Summaries
For each chamber, read manifest.json
Extract: goal, milestone, phases_completed, entombed_at

### Step 3: Display Tree View
```
🕳️ ═══════════════════════════════════════════════════
   T U N N E L S   (Colony History)
══════════════════════════════════════════════════ 🕳️

Chambers: {count} colonies archived

{For each chamber:}
📦 {chamber_name}
   👑 {goal (truncated to 50 chars)}
   🏆 {milestone} ({version})
   📍 {phases_completed} phases | 📅 {date}
{End for}

Run /ant:tunnels <chamber_name> to view details
```

### Step 4: Detail View (if argument provided)
If `$ARGUMENTS` is a valid chamber name:
Display full manifest + key state fields
Show decisions and learnings preserved
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Single colony forever | Lifecycle with archival | Phase 10 | Users can start fresh while preserving wisdom |
| Manual milestone setting | Auto-detection | Phase 10 | Milestones reflect actual progress |
| Archive in data/archive/ | Dedicated chambers/ | Phase 10 | Clearer separation of concerns |
| Full state archive | Minimal (state + manifest) | Phase 10 | Faster, less disk usage |

**Current milestone progression:**
1. **First Mound** — Colony initialized (v0.1)
2. **Open Chambers** — 1+ phases complete (v0.x)
3. **Brood Stable** — 3+ phases complete (v0.x)
4. **Ventilated Nest** — 5+ phases complete (v0.x)
5. **Sealed Chambers** — All phases complete (v1.x.x)
6. **Crowned Anthill** — User explicitly sealed (v1.x.x)

## Manifest Schema

```typescript
interface ChamberManifest {
  // Required metadata
  entombed_at: string;        // ISO-8601 timestamp
  goal: string;               // Original colony goal

  // Progress tracking
  phases_completed: number;   // Count of completed phases
  total_phases: number;       // Total phases in plan
  milestone: string;          // Final milestone achieved
  version: string;            // Computed version string

  // Pheromone preservation
  decisions: Array<{
    id: string;
    description: string;
    timestamp: string;
  }>;
  learnings: Array<{
    id: string;
    content: string;
    phase: string;
    status: string;
  }>;

  // Integrity verification
  files: Record<string, string>;  // filename -> sha256 hash
}
```

## Open Questions

1. **Milestone trigger frequency**
   - What we know: Milestones should auto-detect based on state
   - What's unclear: Should detection run on every command or just status?
   - Recommendation: Run on `/ant:status` and after phase completion

2. **Tunnel browsing detail level**
   - What we know: Users want to browse archived colonies
   - What's unclear: How much detail to show in list vs detail view
   - Recommendation: Summary in list, full manifest + key state fields in detail

3. **Multi-colony future compatibility**
   - What we know: Single active colony only for now
   - What's unclear: How to structure for future multi-colony support
   - Recommendation: Keep flat chamber structure, add active_colony pointer in future

## Sources

### Primary (HIGH confidence)
- `/Users/callumcowie/repos/Aether/.aether/data/COLONY_STATE.json` - State structure v3.0
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/init.md` - Command pattern for init
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/status.md` - Status display pattern
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/seal.md` - Archive pattern (similar to entomb)
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` - Utility patterns, JSON handling
- `/Users/callumcowie/repos/Aether/bin/cli.js` - CLI structure, checkpoint patterns
- `/Users/callumcowie/repos/Aether/.planning/phases/10-entombment-egg-laying/10-CONTEXT.md` - User decisions

### Secondary (MEDIUM confidence)
- `/Users/callumcowie/repos/Aether/.planning/STATE.md` - Project state, milestone definitions
- `/Users/callumcowie/repos/Aether/.planning/REQUIREMENTS.md` - Phase 10 requirements

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - uses existing dependencies only
- Architecture: HIGH - based on existing patterns (seal.md, init.md)
- Pitfalls: MEDIUM - derived from similar archival patterns, not yet tested

**Research date:** 2026-02-14
**Valid until:** 2026-03-14 (30 days for stable features)

---

## Implementation Notes for Planner

### Files to Create
1. `.claude/commands/ant/entomb.md` - Archive colony command
2. `.claude/commands/ant/lay-eggs.md` - Start fresh colony command
3. `.claude/commands/ant/tunnels.md` - Browse history command
4. `.opencode/commands/ant/` mirrors for all three

### Files to Modify
1. `.claude/commands/ant/status.md` - Add milestone display
2. `bin/cli.js` - Add `aether chambers` subcommand (optional)

### New Directories
1. `.aether/chambers/` - Archive storage (created on first entomb)

### State Schema Additions
Add to COLONY_STATE.json:
```json
{
  "milestone": "First Mound",
  "milestone_updated_at": "2026-02-14T15:30:22Z",
  "milestone_version": "v0.1.0"
}
```

### CLI Additions (Optional)
```bash
aether chambers list          # List all chambers
aether chambers show <name>   # Show chamber details
aether chambers verify <name> # Verify chamber integrity
```
