# Phase 3: Pheromone Communication - Research

**Researched:** 2025-02-01
**Domain:** Prompt-based stigmergic signaling system
**Confidence:** HIGH

## Summary

Phase 3 implements the pheromone communication layer that enables the Queen (user) to guide the colony through signals rather than commands. The system uses JSON-based state files with prompt-based computation - there is NO code execution for decay calculations. Instead, decay is computed on-demand when pheromones are read.

**Key Finding**: This is a **hybrid system** - JSON state persists signals, but all "computation" (decay, effective strength, combinations) happens in prompts through natural language instructions and examples. Worker Ants read pheromones.json and respond based on their sensitivity profiles.

**Primary recommendation**: Implement pheromone commands as pure JSON manipulation with bash/jq commands, following the exact pattern from init.md. Worker Ants already have sensitivity profiles in worker_ants.json - Phase 3 just needs to make them functional.

## Standard Stack

The system uses **no external libraries** - it's a pure bash/jq/JSON implementation.

### Core
| Component | Version | Purpose | Why Standard |
|-----------|---------|---------|--------------|
| jq | CLI | JSON manipulation | Standard JSON query tool, already used in init.md |
| bash | POSIX | State file operations | Atomic writes, file locking via .aether/utils/atomic-write.sh |
| JSON | RFC 8259 | State persistence | Human-readable, git-friendly, Claude-native |

### File-Based Architecture (No Libraries)
| File | Purpose | Pattern |
|------|---------|---------|
| `.aether/data/pheromones.json` | Active signals | Read/modify via jq, atomic writes |
| `.aether/data/worker_ants.json` | Caste sensitivities | Pre-populated, read-only for Workers |
| `.aether/data/COLONY_STATE.json` | Colony state | Shared state, already exists |
| `.aether/data/memory.json` | Learning storage | Feedback patterns stored here |

### Command Structure
| Command | File | Purpose |
|---------|------|---------|
| `/ant:focus "<area>"` | focus.md | Emit attract signal (1h decay) |
| `/ant:redirect "<pattern>"` | redirect.md | Emit repel signal (24h decay) |
| `/ant:feedback "<message>"` | feedback.md | Emit guidance signal (6h decay) |

**Installation**: No packages needed - all tools are pre-existing bash utilities.

## Architecture Patterns

### The Prompt-Based Computation Pattern

**Critical Insight**: Decay and effective strength are NOT calculated by code. They are **interpreted by Worker Ants** when they read pheromones.json.

```
Traditional System:
  pheromone.created_at → decay_function() → current_strength

Aether System:
  pheromone.created_at → Worker Ant reads → Ant interprets "this is 30min old" → Ant uses 50% strength
```

### Recommended Command Structure

Follow the **exact pattern from init.md**:

```bash
# 1. Validate input
if [ -z "$1" ]; then
  echo "Usage: /ant:focus "<area>""
  exit 1
fi

# 2. Load state
PHEROMONES=".aether/data/pheromones.json"

# 3. Create new pheromone object
timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
pheromone_id="focus_$(date +%s)"

# 4. Add to active_pheromones array
jq --arg id "$pheromone_id" \
   --arg timestamp "$timestamp" \
   --arg context "$1" \
   '
   .active_pheromones += [{
     "id": $id,
     "type": "FOCUS",
     "strength": 0.7,
     "created_at": $timestamp,
     "decay_rate": 3600,
     "metadata": {
       "source": "queen",
       "caste": null,
       "context": $context
     }
   }]
   ' "$PHEROMONES" > /tmp/pheromones.tmp

# 5. Atomic write
.aether/utils/atomic-write.sh atomic_write_from_file "$PHEROMONES" /tmp/pheromones.tmp

# 6. Display formatted output
```

### Pattern 1: Pheromone Creation

**What**: Commands append new pheromone objects to active_pheromones array

**When to use**: Every time Queen emits a signal

**Example**:
```bash
# Source: Based on init.md pattern
jq --arg id "$pheromone_id" \
   --arg timestamp "$timestamp" \
   --arg context "$1" \
   '
   .active_pheromones += [{
     "id": $id,
     "type": "FOCUS",
     "strength": 0.7,
     "created_at": $timestamp,
     "decay_rate": 3600,
     "metadata": {
       "source": "queen",
       "caste": null,
       "context": $context
     }
   }]
   ' pheromones.json > /tmp/pheromones.tmp
```

### Pattern 2: Worker Ant Pheromone Reading

**What**: Worker Ant prompts include instructions to read and interpret pheromones

**When to use**: Every Worker Ant task execution

**Example** (from builder-ant.md):
```
## Your Workflow

### 1. Receive Task
Extract from context:
- **Task**: What needs to be built/implemented
- **Acceptance Criteria**: How to know when it's done
- **Active Pheromones**: Read from .aether/data/pheromones.json

### 2. Interpret Pheromones

Read active_pheromones and calculate effective strength:

For each active pheromone:
1. Check your caste's sensitivity (from worker_ants.json)
2. Calculate decay:
   - FOCUS: strength × 0.5^((now - created_at) / 3600)
   - REDIRECT: strength × 0.5^((now - created_at) / 86400)
   - FEEDBACK: strength × 0.5^((now - created_at) / 21600)
   - INIT: No decay (persists until phase complete)
3. Calculate effective strength: decayed_strength × your_sensitivity
4. If effective_strength > 0.1, respond to signal

Example:
  FOCUS created 30 minutes ago, strength 0.7
  → Decay: 0.7 × 0.5^(0.5) = 0.49
  → Builder sensitivity: 1.0
  → Effective: 0.49 × 1.0 = 0.49 (strong response)

### 3. Adjust Behavior Based on Signals

FOCUS signals (attract):
- effective_strength > 0.5: Prioritize this work immediately
- effective_strength 0.3-0.5: Include in planning with priority
- effective_strength < 0.3: Note but don't prioritize

REDIRECT signals (repel):
- effective_strength > 0.5: Avoid this pattern completely
- effective_strength 0.3-0.5: Seek alternatives, document decision
- effective_strength < 0.3: Note constraint, proceed with caution

FEEDBACK signals (guidance):
- Quality feedback: Increase testing, add validation
- Speed feedback: Simplify approach, increase parallelization
- Direction feedback: Reconsider approach, pivot if needed
- Positive feedback: Record pattern for reuse
```

### Pattern 3: Pheromone Combination Response

**What**: Worker Ants interpret multiple signals together

**When to use**: When multiple active pheromones exist

**Example**:
```
### Pheromone Combinations

When multiple pheromones are active, combine their effects:

FOCUS + FEEDBACK (same topic):
- If feedback is positive: Increase prioritization further
- If feedback is quality: Add extra validation to focused work
- If feedback is direction: Pivot focused area

INIT + REDIRECT:
- Goal established, but avoid specific approaches
- Plan alternative paths to goal
- Document constraints in working memory

Multiple FOCUS signals:
- Prioritize by effective strength (signal × sensitivity)
- Work on highest-strength focus first
- Note lower-priority focuses for later

Example:
  Active signals:
  - FOCUS "WebSocket security" (effective 0.6)
  - FEEDBACK "Great progress on authentication" (positive)
  - REDIRECT "Don't use callbacks" (effective 0.8)

  Response:
  - Prioritize WebSocket security work (0.6 > threshold)
  - Use authentication patterns as reference (positive feedback)
  - Avoid callback-based WebSocket implementations (redirect)
  - Use async/await instead
```

### Anti-Patterns to Avoid

- **Don't implement decay calculation in code**: The system is prompt-based. Decay is interpreted, not calculated.
- **Don't create a separate "decay process"**: No background jobs needed. Worker Ants compute decay on-read.
- **Don't over-engineer signal combination**: Keep it simple - natural language instructions in prompts.
- **Don't modify worker_ants.json caste sensitivities**: These are already set. Worker Ants read their own sensitivity.
- **Don't create complex signal validation**: The schema is already defined. Just populate it.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Decay calculation | Python/Node script to calculate decay | Prompt instructions + natural language examples | System is prompt-based, not code-based |
| Signal propagation | Complex event bus | JSON file reading | Worker Ants read state directly |
| Pheromone validation | Custom validation logic | Schema already in pheromones.json | Structure is pre-defined |
| Atomic writes | Manual file locking | .aether/utils/atomic-write.sh | Already exists, proven pattern |
| JSON manipulation | Python/Node scripts | jq (standard CLI tool) | Used in init.md, reliable |
| Command parsing | Custom argument parser | Bash $1, $2 pattern | Simple, proven |

**Key insight**: This is a **minimal computation** system. The "intelligence" is in the prompts, not the code. JSON stores state, Worker Ants interpret state.

## Common Pitfalls

### Pitfall 1: Over-Engineering Decay Calculation

**What goes wrong**: Implementing a background process to update pheromone strengths over time

**Why it happens**: Traditional thinking assumes decay must be calculated continuously

**How to avoid**: Remember this is a **prompt-based system**. Decay is calculated on-demand by Worker Ants when they read pheromones.json. No background process needed.

**Warning signs**: Planning a cron job, daemon, or "decay service" → Stop, re-read init.md pattern

### Pitfall 2: Modifying Existing Commands

**What goes wrong**: Changing init.md or status.md to match new pheromone structure

**Why it happens**: Wanting consistency across all commands

**How to avoid**: init.md is working. Don't touch it. Only create NEW commands (focus.md, redirect.md, feedback.md). The existing commands work fine.

**Warning signs**: "I need to update init.md to use the new schema" → No, init.md is fine as-is

### Pitfall 3: Ignoring Worker Ant Sensitivity Profiles

**What goes wrong**: Creating new sensitivity calculation logic instead of using existing profiles

**Why it happens**: Not reading worker_ants.json before implementation

**How to avoid**: worker_ants.json ALREADY has sensitivity_profile for each caste. Use these values directly. Don't create new ones.

**Warning signs**: "I need to define caste sensitivities" → No, they're already defined

### Pitfall 4: Complex Pheromone Query Logic

**What goes wrong**: Building complex jq queries to filter pheromones by type, age, strength

**Why it happens**: Wanting "efficient" pheromone lookup

**How to avoid**: Worker Ants read the entire active_pheromones array and interpret in prompts. Simple jq to read the array is enough. No complex filtering needed.

**Warning signs**: Designing a pheromone query language → Stop, just read JSON in prompts

### Pitfall 5: Forgetting INIT Pheromone

**What goes wrong**: Only implementing FOCUS/REDIRECT/FEEDBACK, forgetting INIT exists

**Why it happens**: INIT is created in init.md, not in a dedicated command

**How to avoid**: INIT is already created by /ant:init. Worker Ants must respond to it too. INIT has no decay.

**Warning signs**: "What about the INIT signal?" → It's already in pheromones.json after init

## Code Examples

### Pheromone Creation (focus.md)

```bash
# Source: Based on init.md lines 69-95
timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
pheromone_id="focus_$(date +%s)"

jq --arg id "$pheromone_id" \
   --arg timestamp "$timestamp" \
   --arg context "$1" \
   --arg strength "0.7" \
   --arg decay_rate "3600" \
   '
   .active_pheromones += [{
     "id": $id,
     "type": "FOCUS",
     "strength": ($strength | tonumber),
     "created_at": $timestamp,
     "decay_rate": ($decay_rate | tonumber),
     "metadata": {
       "source": "queen",
       "caste": null,
       "context": $context
     }
   }]
   ' .aether/data/pheromones.json > /tmp/pheromones.tmp

.aether/utils/atomic-write.sh atomic_write_from_file .aether/data/pheromones.json /tmp/pheromones.tmp
```

### Pheromone Reading (Worker Ant Prompt)

```markdown
# Source: Builder Ant prompt pattern

## Read Active Pheromones

Before starting work, read current pheromone signals:

\`\`\`bash
# Read pheromones
cat .aether/data/pheromones.json

# Read your caste's sensitivity
cat .aether/data/worker_ants.json | jq '.castes.builder.sensitivity_profile'
\`\`\`

## Interpret Signals

Your caste (builder) has these sensitivities:
- INIT: 0.9 - Respond when implementation is needed
- FOCUS: 1.0 - Highly responsive, prioritize focused areas
- REDIRECT: 0.7 - Avoid redirected patterns
- FEEDBACK: 0.9 - Adjust approach based on feedback

For each active pheromone:

1. **Calculate decay**:
   \`\`\`
   hours_elapsed = (now - created_at) / 3600
   decay_factor = 0.5 ^ hours_elapsed
   current_strength = strength × decay_factor
   \`\`\`

2. **Calculate effective strength**:
   \`\`\`
   effective = current_strength × your_sensitivity
   \`\`\`

3. **Respond if effective > 0.1**:
   - FOCUS > 0.5: Prioritize immediately
   - REDIRECT > 0.5: Avoid completely
   - FEEDBACK > 0.3: Adjust behavior

Example calculation:
  FOCUS "WebSocket security" created 30min ago
  - strength: 0.7
  - hours: 0.5
  - decay: 0.5^0.5 = 0.707
  - current: 0.7 × 0.707 = 0.495
  - builder sensitivity: 1.0
  - effective: 0.495 × 1.0 = 0.495
  - Action: Prioritize (0.495 > 0.3 threshold)
```

### Effective Strength Table (Quick Reference)

```markdown
# Source: worker_ants.json caste_sensitivities

| Caste | INIT | FOCUS | REDIRECT | FEEDBACK |
|-------|------|-------|----------|----------|
| Colonizer | 1.0 | 0.8 | 0.9 | 0.7 |
| Route-setter | 1.0 | 0.9 | 0.8 | 0.8 |
| Builder | 0.9 | 1.0 | 0.7 | 0.9 |
| Watcher | 0.8 | 0.9 | 1.0 | 1.0 |
| Scout | 0.9 | 0.7 | 0.8 | 0.8 |
| Architect | 0.8 | 0.8 | 0.9 | 1.0 |

Response thresholds:
- effective > 0.5: Strong response
- effective 0.3-0.5: Moderate response
- effective 0.1-0.3: Weak response
- effective < 0.1: No response
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| N/A (new system) | Prompt-based decay interpretation | Initial design | No background processes needed |
| Traditional event bus | JSON file with atomic writes | Phase 1 | Simple, git-friendly state |
| Code-based signal propagation | Worker Ants read state directly | Phase 3 design | Emergent behavior from prompts |

**Key Design Decision**: The Aether system is **NOT** a traditional software system. It's a **prompt-based coordination system** where "computation" happens through natural language interpretation, not code execution.

### Why This Approach?

1. **Claude-Native**: Works within Claude's context window without external services
2. **Git-Friendly**: All state is JSON, can be versioned
3. **Observable**: Queen can see all signals in pheromones.json
4. **Simple**: No daemons, cron jobs, or background processes
5. **Emergent**: Behavior arises from prompt instructions, not rigid code

## Open Questions

1. **Pheromone Cleanup**: Should old pheromones be removed from active_pheromones?
   - **What we know**: pheromones.json has active_pheromones array
   - **What's unclear**: When to remove expired pheromones (strength < 0.01)
   - **Recommendation**: Add cleanup step to commands - remove pheromones older than 4× half-life before adding new ones

2. **Feedback Storage**: Where should feedback history be stored?
   - **What we know**: memory.json has learning_patterns section
   - **What's unclear**: Exact structure for feedback_history
   - **Recommendation**: Store in memory.json.working_memory.items with type="feedback"

3. **Pheromone Persistence**: Should pheromones persist across context refreshes?
   - **What we know**: All state is in JSON files
   - **What's unclear**: Should old signals be cleared on new session?
   - **Recommendation**: Keep INIT and REDIRECT (persistent), auto-expire FOCUS and FEEDBACK (4× half-life)

## Sources

### Primary (HIGH confidence)
- `.claude/commands/ant/init.md` - Command pattern, JSON manipulation, atomic writes
- `.claude/commands/ant/status.md` - State reading pattern, jq usage
- `.claude/commands/ant/focus.md` - Existing focus command (draft)
- `.claude/commands/ant/redirect.md` - Existing redirect command (draft)
- `.claude/commands/ant/feedback.md` - Existing feedback command (draft)
- `.aether/data/pheromones.json` - Pheromone schema, caste sensitivities
- `.aether/data/worker_ants.json` - Worker Ant sensitivity profiles
- `.aether/workers/builder-ant.md` - Worker Ant prompt pattern
- `.aether/workers/watcher-ant.md` - Worker Ant prompt pattern
- `.aether/QUEEN_ANT_ARCHITECTURE.md` - Architecture philosophy

### Secondary (MEDIUM confidence)
- Phase 3 task list (03-pheromone-communication/) - Requirements
- Existing command files - Pattern verification

### Tertiary (LOW confidence)
- None - all sources are primary project files

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All tools are bash/jq, already used in project
- Architecture: HIGH - Pattern is clearly defined in init.md
- Pitfalls: HIGH - Examined existing draft commands for anti-patterns
- Implementation: HIGH - Clear pattern from init.md, schema already exists

**Research date:** 2025-02-01
**Valid until:** 2025-03-01 (30 days - stable architecture, low risk of changes)
