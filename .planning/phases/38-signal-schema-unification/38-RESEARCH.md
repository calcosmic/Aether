# Phase 38: Signal Schema Unification - Research

**Researched:** 2026-02-06
**Domain:** Signal schema consistency and path unification
**Confidence:** HIGH

## Summary

Phase 38 is a gap closure phase addressing critical integration issues identified in the v5.1-MILESTONE-AUDIT.md. The core problem: Phase 36 defined a TTL-based signal schema but init.md still writes the old schema, and build.md reads from pheromones.json while signal commands now write to COLONY_STATE.json.

Three specific issues need resolution:
1. **init.md schema mismatch:** Writes legacy `strength` + `half_life_seconds` instead of TTL `priority` + `expires_at`
2. **Signal path inconsistency:** focus.md, redirect.md, feedback.md write to COLONY_STATE.json signals array, but build.md and continue.md read from pheromones.json
3. **Consumer-producer disconnect:** No unified signal path means signals emitted via commands don't reach build.md

This is a straightforward fix phase with clear scope: update init.md signal schema and update signal consumers to read from the correct location.

**Primary recommendation:** Update init.md to use TTL schema, update build.md and continue.md to read signals from COLONY_STATE.json (not pheromones.json), and remove/deprecate pheromones.json entirely.

## Current State Analysis

### Issue 1: init.md Legacy Schema (Lines 107-125)

**Location:** `commands/ant/init.md` lines 107-125

**Current (BROKEN):**
```json
{
  "signals": [
    {
      "id": "init_<unix_timestamp>",
      "type": "INIT",
      "content": "<the user's goal>",
      "strength": 1.0,
      "half_life_seconds": null,
      "created_at": "<ISO-8601 timestamp>"
    }
  ]
}
```

**Required (TTL schema per Phase 36):**
```json
{
  "id": "init_<unix_timestamp>",
  "type": "INIT",
  "content": "<the user's goal>",
  "priority": "high",
  "created_at": "<ISO-8601 timestamp>",
  "expires_at": "phase_end",
  "source": "system:init"
}
```

**Impact:** INIT signals created by `/ant:init` fail TTL filtering in `/ant:build` because they lack `priority` and `expires_at` fields.

### Issue 2: Signal Path Inconsistency

**Commands that WRITE signals:**
| Command | Writes To | Schema | Status |
|---------|-----------|--------|--------|
| focus.md | COLONY_STATE.json signals | TTL | CORRECT |
| redirect.md | COLONY_STATE.json signals | TTL | CORRECT |
| feedback.md | COLONY_STATE.json signals | TTL | CORRECT |
| init.md | pheromones.json | Legacy | BROKEN |
| build.md Step 7b | pheromones.json | TTL | WRONG PATH |
| continue.md Step 4.5 | pheromones.json | TTL | WRONG PATH |

**Commands that READ signals:**
| Command | Reads From | Status |
|---------|-----------|--------|
| build.md Step 2, 3 | pheromones.json | WRONG PATH |
| continue.md Step 1 | pheromones.json | WRONG PATH |
| status.md | COLONY_STATE.json | CORRECT |
| plan.md | pheromones.json | WRONG PATH |
| organize.md | pheromones.json | WRONG PATH |
| pause-colony.md | pheromones.json | WRONG PATH |
| resume-colony.md | pheromones.json | WRONG PATH |
| colonize.md | pheromones.json | WRONG PATH (if applicable) |

**Root cause:** Phase 33 defined COLONY_STATE.json with signals section, Phase 37 updated signal commands to write there, but build.md and continue.md were not updated to read from there.

### Issue 3: Validation Expects pheromones.json

**Location:** `~/.aether/aether-utils.sh` lines 75-86

The `validate-state pheromones` command validates pheromones.json, not COLONY_STATE.json signals. This needs to either:
- Be updated to validate COLONY_STATE.json signals section
- Or be deprecated if pheromones.json is removed

Current validation checks for TTL fields (`priority`, `expires_at`) which is correct, but wrong file.

## Files Requiring Changes

### Must Fix

| File | Lines | Change Required |
|------|-------|-----------------|
| commands/ant/init.md | 107-125 | Change signal schema from legacy to TTL |
| commands/ant/init.md | 108 | Change write target from pheromones.json to COLONY_STATE.json signals |
| commands/ant/build.md | 57 | Remove pheromones.json from read list |
| commands/ant/build.md | 70 | Change filter source from pheromones.json to COLONY_STATE.json |
| commands/ant/build.md | 902-916 | Change write target from pheromones.json to COLONY_STATE.json |
| commands/ant/build.md | 948 | Remove pheromones.json write reference |
| commands/ant/continue.md | 23 | Remove pheromones.json from read list |
| commands/ant/continue.md | 370-422 | Change signal operations to COLONY_STATE.json |
| commands/ant/continue.md | 445-454 | Change filter source to COLONY_STATE.json |
| commands/ant/plan.md | 14, 31 | Change signal read from pheromones.json to COLONY_STATE.json |
| commands/ant/organize.md | 17, 27 | Change signal read from pheromones.json to COLONY_STATE.json |
| commands/ant/pause-colony.md | 14, 21 | Change signal read from pheromones.json to COLONY_STATE.json |
| commands/ant/resume-colony.md | 15, 37, 47 | Change signal operations to COLONY_STATE.json |

### Should Update

| File | Change |
|------|--------|
| commands/ant/ant.md | Line 83: Update data file list (remove pheromones.json or mark deprecated) |
| ~/.aether/aether-utils.sh | Update validate-state pheromones to validate COLONY_STATE.json signals |

### Can Remove (After Migration)

| File | Reason |
|------|--------|
| .aether/data/pheromones.json | Superseded by COLONY_STATE.json signals section |

## TTL Signal Schema (Authoritative Reference)

Per Phase 36 research and implementation, the canonical signal schema is:

```json
{
  "id": "<type>_<unix_timestamp_ms>",
  "type": "INIT|FOCUS|REDIRECT|FEEDBACK",
  "content": "<signal content string>",
  "priority": "high|normal|low",
  "created_at": "<ISO-8601 UTC>",
  "expires_at": "<ISO-8601 UTC>|phase_end",
  "source": "<origin identifier>"
}
```

### Priority Mapping

| Signal Type | Default Priority | Rationale |
|-------------|-----------------|-----------|
| INIT | high | Colony goal, must always be visible |
| REDIRECT | high | Hard constraints, check first |
| FOCUS | normal | Attention guidance, standard priority |
| FEEDBACK | low | Observational, lower urgency |

### Source Identifiers

| Source | When Used |
|--------|-----------|
| `system:init` | INIT signal from /ant:init |
| `user` | User-emitted via /ant:focus, /ant:redirect, /ant:feedback |
| `worker:builder` | Auto-emitted by build.md Step 7b |
| `worker:continue` | Auto-emitted by continue.md Step 4.5 |
| `global:inject` | Injected from global learnings |

### Expiration Values

| Value | Meaning |
|-------|---------|
| `"phase_end"` | Signal expires when phase advances (default for most signals) |
| ISO-8601 timestamp | Signal expires at specified wall-clock time |

## Architecture Decision: Unified Signal Location

**Decision:** All signals live in COLONY_STATE.json `signals` array. Remove pheromones.json.

**Rationale:**
1. Phase 33 created consolidated state with signals section
2. Phase 37 updated signal emission commands to use COLONY_STATE.json
3. Maintaining two signal stores creates drift and inconsistency
4. Single source of truth simplifies validation and debugging

**Migration path:**
1. Update init.md to write INIT signal to COLONY_STATE.json with TTL schema
2. Update build.md to read/write signals from COLONY_STATE.json
3. Update continue.md to read/write signals from COLONY_STATE.json
4. Update remaining commands (plan.md, organize.md, pause-colony.md, resume-colony.md)
5. Update aether-utils.sh validate-state to check COLONY_STATE.json signals
6. Stop creating pheromones.json in init.md
7. Mark pheromones.json as deprecated in ant.md

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Signal validation | Custom validation logic | aether-utils.sh validate-state | Centralized, tested |
| TTL filtering | Multiple implementations | Consistent pattern per Phase 36 | Already defined in working commands |
| Signal ID generation | Various formats | `<type>_<unix_timestamp_ms>` | Consistent with existing pattern |

**Key insight:** This phase is purely about consistency. The TTL system is already working in focus.md, redirect.md, feedback.md, status.md. The task is to propagate this to the remaining files.

## Common Pitfalls

### Pitfall 1: Forgetting Auto-Emit Locations
**What goes wrong:** build.md Step 7b and continue.md Step 4.5 auto-emit signals; if not updated, they write to wrong location
**Why it happens:** These are buried in large command files
**How to avoid:** Explicitly verify both auto-emit locations are updated
**Warning signs:** Signals emitted during build don't appear in status

### Pitfall 2: Partial Migration
**What goes wrong:** Some commands read from COLONY_STATE.json, others from pheromones.json
**Why it happens:** Not updating all consumers
**How to avoid:** Complete file list audit before planning
**Warning signs:** Signals visible in some commands but not others

### Pitfall 3: Validation Script Mismatch
**What goes wrong:** validate-state pheromones still checks pheromones.json after removal
**Why it happens:** Utility script not updated with commands
**How to avoid:** Update aether-utils.sh as part of this phase
**Warning signs:** validate-state fails after init

### Pitfall 4: Init Not Writing to COLONY_STATE.json
**What goes wrong:** init.md continues to create pheromones.json, COLONY_STATE.json has no signals
**Why it happens:** Step 5 in init.md writes to pheromones.json specifically
**How to avoid:** Change init.md to write INIT signal directly to COLONY_STATE.json
**Warning signs:** Fresh colony has no signals in status

## Code Examples

### Updated init.md Step 5 (Signal Emission)

**Current (lines 107-125):**
```markdown
### Step 5: Emit INIT Pheromone

Use the Write tool to write `.aether/data/pheromones.json`:

{
  "signals": [
    {
      "id": "init_<unix_timestamp>",
      "type": "INIT",
      "content": "<the user's goal>",
      "strength": 1.0,
      "half_life_seconds": null,
      "created_at": "<ISO-8601 timestamp>"
    }
  ]
}
```

**New (TTL schema, COLONY_STATE.json):**
```markdown
### Step 5: Emit INIT Signal

Add an INIT signal to the COLONY_STATE.json you wrote in Step 3.

Add to the `signals` array (create if missing):

{
  "id": "init_<unix_timestamp_ms>",
  "type": "INIT",
  "content": "<the user's goal>",
  "priority": "high",
  "created_at": "<ISO-8601 timestamp>",
  "expires_at": "phase_end",
  "source": "system:init"
}

Note: The INIT signal uses `expires_at: "phase_end"` so it persists until the first phase completes.
```

### Updated build.md Step 2 (State Reading)

**Current (line 57):**
```markdown
- `.aether/data/pheromones.json`
```

**New:**
```markdown
(Remove pheromones.json from read list - signals are in COLONY_STATE.json)
```

### Updated build.md Step 3 (Signal Filtering)

**Current (line 70):**
```markdown
Filter signals from `pheromones.json` using TTL:
```

**New:**
```markdown
Filter signals from `COLONY_STATE.json` signals array using TTL:
```

### Updated build.md Step 7b (Auto-Emit)

**Current (lines 902-916):**
```markdown
Read `.aether/data/pheromones.json` (if not already in memory)...
Append the signal to the `signals` array in `pheromones.json`.
```

**New:**
```markdown
Read COLONY_STATE.json signals array (already in memory from Step 2).
Append the signal to the `signals` array in COLONY_STATE.json.
```

## Verification Criteria

After this phase, the following must be true:

1. **init.md writes TTL signals:** INIT signal has `priority`, `expires_at`, no `strength`, no `half_life_seconds`
2. **All signal writes go to COLONY_STATE.json:** focus.md, redirect.md, feedback.md, init.md, build.md, continue.md all write to COLONY_STATE.json signals array
3. **All signal reads come from COLONY_STATE.json:** build.md, continue.md, plan.md, organize.md, status.md, pause-colony.md, resume-colony.md all read from COLONY_STATE.json signals array
4. **pheromones.json is not created:** init.md no longer writes pheromones.json
5. **validate-state works:** Either updated to check COLONY_STATE.json signals, or pheromones validation deprecated/removed

## Open Questions

1. **pheromones.json removal timing:** Should we remove pheromones.json support entirely in this phase, or leave it as deprecated?
   - Recommendation: Remove completely. All references are being updated anyway.

2. **validate-state pheromones:** Update to validate COLONY_STATE.json signals, or remove?
   - Recommendation: Update. The validation logic for TTL fields is correct, just wrong file.

3. **Backward compatibility:** What if user has existing pheromones.json?
   - Recommendation: Ignore. Next `/ant:init` will reset all state anyway.

## Sources

### Primary (HIGH confidence)
- `commands/ant/init.md` - Direct read, lines 107-125 show legacy schema
- `commands/ant/build.md` - Direct read, lines 57, 70, 902-916 show pheromones.json references
- `commands/ant/continue.md` - Direct read, lines 23, 370-454 show pheromones.json references
- `commands/ant/focus.md` - Direct read, shows correct COLONY_STATE.json target
- `commands/ant/redirect.md` - Direct read, shows correct COLONY_STATE.json target
- `commands/ant/feedback.md` - Direct read, shows correct COLONY_STATE.json target
- `commands/ant/status.md` - Direct read, shows correct COLONY_STATE.json source
- `~/.aether/aether-utils.sh` - Direct read, validate-state pheromones logic
- `.planning/v5.1-MILESTONE-AUDIT.md` - Gap identification source
- `.planning/phases/36-signal-simplification/36-RESEARCH.md` - TTL schema specification
- `.planning/phases/36-signal-simplification/36-VERIFICATION.md` - Confirms TTL implementation
- `.planning/phases/37-command-trim-utilities/37-VERIFICATION.md` - Confirms signal commands write to COLONY_STATE.json

### Secondary (MEDIUM confidence)
- None - all findings from direct codebase inspection

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Problem identification: HIGH - Direct code inspection confirms all issues
- Solution approach: HIGH - Pattern already exists in working commands
- File list: HIGH - Comprehensive grep confirms all references
- Schema specification: HIGH - Phase 36 research is authoritative

**Research date:** 2026-02-06
**Valid until:** Until phase completion - no external dependencies
