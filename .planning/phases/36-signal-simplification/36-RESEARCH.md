# Phase 36: Signal Simplification - Research

**Researched:** 2026-02-06
**Domain:** Pheromone/signal system simplification (TTL-based expiration)
**Confidence:** HIGH

## Summary

Phase 36 replaces the pheromone exponential decay system with simple TTL-based expiration. The current system uses half-life math with sensitivity matrices to compute "effective signal strength" for each caste. This is over-engineered for what amounts to "show workers their guidance."

Current complexity being removed:
- **aether-utils.sh:** 5 pheromone subcommands (decay, effective, batch, cleanup, validate) totaling ~80 lines
- **Commands:** Sensitivity matrix tables in build.md, status.md, continue.md
- **Signal format:** `strength` + `half_life_seconds` + decay calculations

New simplicity:
- **Signal format:** `expires_at` timestamp + `priority` field (high/normal/low)
- **Expiration:** Filter on read, no decay math
- **Pause-awareness:** Track pause duration, extend `expires_at` on resume

The user decisions from CONTEXT.md constrain this implementation:
- TTL specified at emit time with sensible default (until phase completion)
- Pause-aware: TTL stops when colony paused, resumes when active
- Priority affects display prominence AND worker behavior
- Show time remaining in status output
- Log expiration events when signals expire mid-task

**Primary recommendation:** Replace the current signal schema with `{id, type, content, priority, created_at, expires_at, source}`, remove all pheromone math from aether-utils.sh, and update commands to filter expired signals on read.

## Standard Stack

This phase involves no external libraries. All work is JSON schema updates, command file refactoring, and aether-utils.sh simplification.

### Core
| Component | Purpose | Why Standard |
|-----------|---------|--------------|
| JSON | Signal storage in COLONY_STATE.json | Already in use |
| Bash date | Timestamp comparison | Native, no dependencies |
| jq (optional) | JSON processing in shell | Already used by aether-utils.sh |

### Supporting
None needed - pure refactoring work.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| expires_at timestamp | TTL seconds field | Timestamp is explicit, no math at read time |
| high/normal/low priority | Numeric (1-3) | Words are clearer in JSON, easier to read |
| Filter on read | Background cleanup | Filtering is simpler, no cleanup command needed |

## Architecture Patterns

### Recommended Signal Schema

**Current (to remove):**
```json
{
  "id": "focus_1738838400",
  "type": "FOCUS",
  "content": "WebSocket security",
  "strength": 0.7,
  "half_life_seconds": 3600,
  "created_at": "2026-02-06T12:00:00Z",
  "source": "user",
  "auto": false
}
```

**New (to implement):**
```json
{
  "id": "focus_1738838400",
  "type": "FOCUS",
  "content": "WebSocket security",
  "priority": "high",
  "created_at": "2026-02-06T12:00:00Z",
  "expires_at": "2026-02-06T14:00:00Z",
  "source": "user"
}
```

### Priority Mapping

Per SIMP-03 requirement:
| Signal Type | Default Priority | Rationale |
|-------------|-----------------|-----------|
| REDIRECT | high | Hard constraints, must be seen first |
| FOCUS | normal | Attention guidance, standard priority |
| FEEDBACK | low | Observational, lower urgency |

Workers check high priority signals first, then normal. Low priority appears in status but doesn't demand attention.

### TTL Defaults

Per CONTEXT.md: default TTL is "until phase completion" (not wall-clock based).

**Implementation:**
- If user specifies `--ttl <duration>`: calculate `expires_at = now + duration`
- If user omits TTL: set `expires_at = "phase_end"` (special marker)
- On phase completion: filter out signals with `expires_at = "phase_end"`
- Wall-clock TTLs: filter when `current_time > expires_at`

Supported duration formats (Claude's discretion per CONTEXT.md):
- `--ttl 30m` (30 minutes)
- `--ttl 2h` (2 hours)
- `--ttl phase` (until phase completion, same as default)

### Pause-Aware TTL

Per CONTEXT.md: pause-aware means TTL timer stops when colony is paused.

**Implementation:**
- Track `paused_at` timestamp in COLONY_STATE.json when pause command runs
- On resume: calculate `pause_duration = resume_time - paused_at`
- For each signal with wall-clock `expires_at`: extend by `pause_duration`
- Signals with `expires_at = "phase_end"` unaffected (already phase-scoped)

### Expiration Handling

Per CONTEXT.md:
1. **Filter on read:** Commands read signals array, filter where `expires_at < now`
2. **Status display:** Show time remaining (e.g., "FOCUS: API layer (12min left)")
3. **Log expiration:** When filtering removes a signal, append event to log

**Filtering logic (all commands that read signals):**
```
For each signal in state.signals:
  if signal.expires_at == "phase_end":
    keep (phase-scoped, not time-expired)
  elif signal.expires_at < current_time:
    skip (expired)
    log_event("signal_expired", signal.type, signal.content)
  else:
    keep (active)
```

### Signal Source Tracking

Per CONTEXT.md: track signal source for debugging.

**Source values:**
- `"user"` - User-emitted via /ant:focus, /ant:redirect, /ant:feedback
- `"worker:builder"` - Auto-emitted by builder in build.md Step 7b
- `"worker:continue"` - Auto-emitted by continue.md Step 4.5
- `"global:inject"` - Injected from global learnings during colonization

This replaces the current `source` and `auto` fields with a single unified `source`.

### Status Display Format

Per CONTEXT.md: show time remaining.

**Current format (to change):**
```
FOCUS      [###########         ] 0.55
  "WebSocket security"
```

**New format:**
```
FOCUS [high] "WebSocket security" (12min left)
REDIRECT [high] "No synchronous I/O" (phase)
FEEDBACK [low] "Good test coverage" (expired)
```

Key changes:
- Priority shown in brackets
- Time remaining or "phase" for phase-scoped
- "expired" marker for signals being filtered out this read (before removal)
- No strength bar (no longer meaningful)

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Expiration math | Decay formulas | Timestamp comparison | `expires_at < now` is trivial |
| Priority ordering | Weighted sorting | Simple array order | high first, then normal, then low |
| Pause tracking | Complex state machine | Two timestamps | `paused_at` and extend on resume |
| Time formatting | Custom duration parser | Pattern matching | "30m" -> 30*60 seconds |

**Key insight:** The current pheromone system uses exponential decay to simulate biological signal fading. This is thematic but unnecessary. "Signal expires in 2 hours" achieves the same goal with zero math.

## Common Pitfalls

### Pitfall 1: Breaking Auto-Emitted Signals
**What goes wrong:** build.md and continue.md auto-emit pheromones after phases; these break if schema changes
**Why it happens:** Auto-emission code not updated to new format
**How to avoid:** Audit all auto-emit locations (build.md Step 7b, continue.md Step 4.5)
**Warning signs:** Missing or malformed signals after builds

### Pitfall 2: Pause State Not Persisted
**What goes wrong:** Colony paused, context cleared, resume fails to extend TTLs
**Why it happens:** `paused_at` not written to COLONY_STATE.json
**How to avoid:** Ensure pause command writes timestamp to state, not just HANDOFF.md
**Warning signs:** Signals expire during legitimate pauses

### Pitfall 3: Phase-End Signals Never Expire
**What goes wrong:** Signals with `expires_at = "phase_end"` accumulate forever
**Why it happens:** No cleanup on phase completion
**How to avoid:** continue.md must filter phase-end signals when advancing phases
**Warning signs:** Status shows stale signals from many phases ago

### Pitfall 4: Time Zone Confusion
**What goes wrong:** Signal created in one timezone, read in another, wrong expiration
**Why it happens:** Inconsistent timestamp handling
**How to avoid:** Always use ISO-8601 UTC (already standard in codebase)
**Warning signs:** Signals expiring immediately or never

### Pitfall 5: Breaking Sensitivity Display in Status
**What goes wrong:** status.md shows "Per-Caste Sensitivity" section that references decay math
**Why it happens:** status.md Step 2 calls `pheromone-batch` for decay calculation
**How to avoid:** Remove sensitivity display, show priority-based ordering instead
**Warning signs:** status command errors or misleading output

## Code Examples

### Current Pheromone Emission (to change)

**focus.md Step 3:**
```markdown
Add a new signal to the `signals` array:
{
  "id": "focus_<unix_timestamp>",
  "type": "FOCUS",
  "content": "<the focus area>",
  "strength": 0.7,
  "half_life_seconds": 3600,
  "created_at": "<ISO-8601 UTC timestamp>"
}
```

**New focus.md Step 3:**
```markdown
Add a new signal to the `signals` array:
{
  "id": "focus_<unix_timestamp>",
  "type": "FOCUS",
  "content": "<the focus area>",
  "priority": "normal",
  "created_at": "<ISO-8601 UTC timestamp>",
  "expires_at": "<ISO-8601 UTC or 'phase_end'>",
  "source": "user"
}

TTL flag parsing:
- If $ARGUMENTS contains --ttl followed by a duration:
  - Parse duration (e.g., "30m" = 30 minutes, "2h" = 2 hours)
  - Set expires_at = created_at + duration
- Otherwise: set expires_at = "phase_end"
```

### Current Decay Calculation (to remove)

**aether-utils.sh pheromone-batch:**
```bash
json_ok "$(jq --arg now "$now" '.signals | map(. + {
  current_strength: (
    if .half_life_seconds == null then .strength
    else
      (($now|tonumber) - (.created_at | sub("\\.[0-9]+Z$";"Z") | fromdate)) as $elapsed |
      (.strength * ((-0.693147180559945 * $elapsed / .half_life_seconds) | exp))
    end | . * 1000 | round / 1000)
})' "$DATA_DIR/pheromones.json")"
```

**New (no aether-utils.sh needed):**

Commands filter directly:
```markdown
For each signal in state.signals:
  if expires_at == "phase_end" or expires_at > current_time:
    signal is active
  else:
    signal is expired, log and skip
```

### Status Display Update

**Current status.md Step 2:**
```markdown
Use the Bash tool to run:
bash ~/.aether/aether-utils.sh pheromone-batch

This returns JSON with current_strength...
```

**New status.md Step 2:**
```markdown
Read signals from COLONY_STATE.json. For each signal:
- Check if expired (expires_at < now AND expires_at != "phase_end")
- Calculate time remaining if not expired
- Display with priority and time remaining

Format:
  {TYPE} [{priority}] "{content}" ({time_remaining} or "phase" or "expired")

Group by priority: high first, then normal, then low.
```

### Pause-Aware Implementation

**pause-colony.md addition:**
```markdown
When pausing:
- Write `paused_at: "<ISO-8601 UTC>"` to COLONY_STATE.json state section
```

**resume-colony.md addition:**
```markdown
When resuming:
- Read `paused_at` from COLONY_STATE.json
- Calculate pause_duration = now - paused_at
- For each signal where expires_at is a timestamp (not "phase_end"):
  - Extend: expires_at = expires_at + pause_duration
- Clear `paused_at` from state
```

## Files to Modify

### aether-utils.sh
| Subcommand | Action | Lines Removed |
|------------|--------|---------------|
| pheromone-decay | DELETE | 12 |
| pheromone-effective | DELETE | 5 |
| pheromone-batch | DELETE | 16 |
| pheromone-cleanup | DELETE | 18 |
| pheromone-validate | KEEP (content validation still useful) | 0 |
| validate-state pheromones | UPDATE (new schema validation) | ~5 rewrite |
| help output | UPDATE (remove deleted commands) | 1 |

**Estimated reduction:** ~50 lines removed from aether-utils.sh

### Command Files

| Command | Changes |
|---------|---------|
| focus.md | New signal schema, add TTL flag parsing |
| redirect.md | New signal schema, add TTL flag parsing |
| feedback.md | New signal schema, add TTL flag parsing |
| status.md | Remove pheromone-batch call, new display format |
| build.md | Remove sensitivity matrix, filter expired signals |
| continue.md | Filter phase-end signals on advance, update auto-emit |
| pause-colony.md | Write paused_at timestamp |
| resume-colony.md | Extend signal TTLs |

### State Schema

**COLONY_STATE.json signals array:**
```json
"signals": [
  {
    "id": "string",
    "type": "FOCUS|REDIRECT|FEEDBACK",
    "content": "string",
    "priority": "high|normal|low",
    "created_at": "ISO-8601",
    "expires_at": "ISO-8601|phase_end",
    "source": "user|worker:builder|worker:continue|global:inject"
  }
]
```

**COLONY_STATE.json new field:**
```json
"paused_at": "ISO-8601|null"
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Half-life decay | TTL expiration | v5.1 (this phase) | Removes exponential math |
| Sensitivity matrices | Priority levels | v5.1 (this phase) | Removes per-caste math |
| Cleanup command | Filter on read | v5.1 (this phase) | No maintenance needed |
| strength 0.0-1.0 | priority high/normal/low | v5.1 (this phase) | Clearer semantics |

**Deprecated/outdated (after this phase):**
- `pheromone-decay` command
- `pheromone-effective` command
- `pheromone-batch` command
- `pheromone-cleanup` command
- `strength` field in signals
- `half_life_seconds` field in signals
- `auto` field in signals (merged into `source`)
- Sensitivity matrix tables in commands
- Caste sensitivity display in status

## Open Questions

1. **INIT signal handling:** The current system has an INIT signal type with strength 1.0 and no decay. With TTL, should INIT have `expires_at = "phase_end"` or be a different construct entirely?
   - Recommendation: INIT becomes `priority: "high"` with `expires_at = "phase_end"` - survives until first phase completes
   - Alternative: Remove INIT concept, the colony goal is already in COLONY_STATE.json

2. **Backward compatibility:** What happens if old-format signals exist in COLONY_STATE.json?
   - Recommendation: Migration during init, or filter out signals missing `expires_at`
   - Risk: Low - most colonies are short-lived, few persistent signals

3. **Event logging volume:** Logging every signal expiration could bloat events array
   - Recommendation: Only log when signal expires mid-task (worker was using it)
   - Alternative: Batch log at phase boundaries

## Sources

### Primary (HIGH confidence)
- `.aether/aether-utils.sh` - Direct read, lines 44-109 (pheromone commands)
- `.aether/docs/pheromones.md` - Current pheromone documentation
- `commands/ant/focus.md`, `redirect.md`, `feedback.md` - Signal emission commands
- `commands/ant/status.md` - Signal display command
- `commands/ant/build.md` - Sensitivity matrix, pheromone-batch usage
- `commands/ant/continue.md` - Auto-emit logic
- `.planning/REQUIREMENTS.md` - SIMP-03 requirement
- `.planning/phases/36-signal-simplification/36-CONTEXT.md` - User decisions

### Secondary (MEDIUM confidence)
- Previous research (33-RESEARCH.md, 35-RESEARCH.md) - Pattern for simplification

### Tertiary (LOW confidence)
- None - all findings from direct codebase inspection

## Metadata

**Confidence breakdown:**
- Signal schema: HIGH - Straightforward replacement
- TTL implementation: HIGH - Simple timestamp comparison
- Pause-aware: HIGH - Two timestamps, extend on resume
- Priority mapping: HIGH - Per SIMP-03 requirement
- Command updates: MEDIUM - Many files, integration complexity
- Event logging: MEDIUM - Volume concerns uncertain

**Research date:** 2026-02-06
**Valid until:** No expiration - internal refactoring, not external dependency
