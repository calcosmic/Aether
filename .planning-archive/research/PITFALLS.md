# Pitfalls Research

**Domain:** Wisdom Accumulation & Pheromone Evolution for Agent Colony Systems
**Researched:** 2026-02-20
**Confidence:** HIGH

---

## Critical Pitfalls

### Pitfall 1: Silent Wisdom Drift from Auto-Promotion

**What goes wrong:**
Automatically promoting learnings to wisdom without user approval causes the QUEEN.md file to accumulate content that doesn't match user priorities. Over time, the wisdom becomes noise rather than signal.

**Why it happens:**
The system sees a pattern repeated 3 times and automatically promotes it. But the user may not consider that pattern valuable or may disagree with its classification.

**How to avoid:**
- Always require user approval for promotion
- Display proposed wisdom with clear type labels
- Let user select which items to promote (tick-to-approve UX)
- Never auto-promote, even when threshold is met

**Warning signs:**
- QUEEN.md grows without user interaction
- Wisdom entries contradict each other
- Workers receive guidance user didn't approve

**Phase to address:** Continue command enhancement — build approval UX before any promotion happens

---

### Pitfall 2: Pheromone Noise from Over-Signaling

**What goes wrong:**
Too many pheromone signals (FOCUS, REDIRECT, FEEDBACK) dilute their effectiveness. Workers can't distinguish important signals from noise.

**Why it happens:**
Every small preference gets encoded as a signal. Without decay or cleanup, signals accumulate.

**How to avoid:**
- Use signal decay (already implemented: effective_strength calculation)
- Reserve REDIRECT for hard constraints only
- Use FEEDBACK for preferences, not rules
- Expire old signals regularly

**Warning signs:**
- pheromone-count returns high numbers
- Worker prompts include conflicting signals
- Signals reference outdated patterns

**Phase to address:** Ongoing — pheromone-expire should run periodically

---

### Pitfall 3: Threshold Mismatch Causes Premature or Delayed Promotion

**What goes wrong:**
Setting thresholds too low causes premature promotion (noise enters wisdom). Setting them too high delays valuable wisdom (user waits forever for patterns to be recognized).

**Why it happens:**
The thresholds (philosophy:5, pattern:3, redirect:2, stack:1, decree:0) are defaults that may not match the user's confidence level or project complexity.

**How to avoid:**
- Document thresholds clearly in QUEEN.md metadata
- Allow user to adjust thresholds per project
- Provide rationale for each threshold level
- Monitor: if proposals are too frequent, raise thresholds; if never, lower them

**Warning signs:**
- Promotion proposals appear on first occurrence
- Patterns that seem obvious never get proposed
- User manually promotes everything (system not helping)

**Phase to address:** Requirements definition — make thresholds configurable

---

### Pitfall 4: Learning Observation Tracking Becomes Stale

**What goes wrong:**
Observation counts become outdated. A pattern observed 5 times in one colony sits at count=5 forever, even though it should accumulate across colonies.

**Why it happens:**
Observations are tracked per-colony but wisdom is meant to be cross-colony. The tracking doesn't follow the wisdom.

**How to avoid:**
- Track observations with colony identifiers
- Accumulate observations across colonies
- Reset observation counts when wisdom is promoted (starts fresh)
- Handle colony name collisions gracefully

**Warning signs:**
- Same learning proposed in every colony
- Observation counts never increase beyond 1-2
- Wisdom doesn't seem to "learn" from experience

**Phase to address:** Observation tracking implementation — design for cross-colony accumulation

---

## Moderate Pitfalls

### Pitfall 5: QUEEN.md Missing Breaks Worker Priming

**What goes wrong:**
If QUEEN.md is deleted or corrupted, `queen-read` fails. Workers don't receive wisdom. No explicit error — just degraded behavior.

**Why it happens:**
QUEEN.md is a file that can be accidentally deleted or corrupted by other tools.

**How to avoid:**
- Graceful degradation: `queen-read` returns `{}` if file missing
- Log warning when QUEEN.md not found
- `/ant:status` checks QUEEN.md health
- init.md creates QUEEN.md if missing

**Warning signs:**
- Workers ignore accumulated wisdom
- No error messages but behavior regresses
- `queen-read` returns empty JSON

**Phase to address:** init.md and build.md integration — add graceful fallbacks

---

### Pitfall 6: Evolution Log Becomes Unusable

**What goes wrong:**
The evolution log in QUEEN.md grows without bound. Eventually it dominates the file, making it hard to read and slow to parse.

**Why it happens:**
Every promotion adds a log entry. No cleanup mechanism.

**How to avoid:**
- Cap evolution log to last N entries (e.g., 50)
- Older entries summarized or archived
- Log only promotions, not observations
- Consider separate log file for audit purposes

**Warning signs:**
- QUEEN.md is mostly evolution log
- File takes noticeable time to parse
- Users stop reading the log

**Phase to address:** queen-promote implementation — add log rotation

---

### Pitfall 7: Wisdom Categories Overlap

**What goes wrong:**
A learning could fit in multiple categories (e.g., "Always use transactions" could be Pattern or Philosophy). Users pick inconsistently, making wisdom harder to search.

**Why it happens:**
Categories are conceptual, not strictly defined. Without guidance, users classify based on mood.

**How to avoid:**
- Provide clear category definitions:
  - **Philosophy:** Core belief, rarely changes
  - **Pattern:** Best practice, applies often
  - **Redirect:** Anti-pattern, avoid this
  - **Stack Wisdom:** Tech-specific, may become obsolete
  - **Decree:** User mandate, always applies
- Ask clarifying questions during promotion
- Allow reclassification after promotion

**Warning signs:**
- Same type of learning in different categories
- User asks "which category should this go in?"
- Categories feel arbitrary

**Phase to address:** Promotion UX — include category guidance

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Skip observation tracking | Faster implementation | Can't enforce thresholds | Never for P1 features |
| Auto-promote without approval | Simpler UX | Wisdom drift | Never |
| Hardcode thresholds | No configuration needed | Doesn't adapt to project needs | Prototype only |
| Ignore missing QUEEN.md | No error handling | Silent degradation | Never |
| Unlimited evolution log | No cleanup code | File bloat | Never |

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Wisdom drift | HIGH | Manually review QUEEN.md, remove unwanted entries, update evolution log |
| Pheromone noise | MEDIUM | Run pheromone-expire, manually clean pheromones.json |
| Wrong threshold | LOW | Update QUEEN.md metadata, re-run promotion check |
| Stale observations | MEDIUM | Reset observation counts, rebuild from colony history |
| Missing QUEEN.md | LOW | Run queen-init, manually restore from backup if needed |
| Log bloat | LOW | Archive old log entries, keep last 50 |

---

## Sources

- Existing pheromone system behavior analysis
- QUEEN.md template inspection
- User-provided v3.0 vision and requirements
- Cross-session learning patterns from other agent systems

---
*Pitfalls research for: v3.0 Wisdom & Pheromone Evolution*
*Researched: 2026-02-20*

---

# Appendix: Context Restoration Pitfalls

**Domain:** Adding instant session restoration to existing Aether colony system
**Researched:** 2026-02-21
**Confidence:** HIGH

---

## Critical Pitfalls

### Pitfall A1: State Schema Migration Hell

**What goes wrong:**
Adding new context fields to `COLONY_STATE.json` breaks existing colonies. Old state files lack new required fields, causing `jq` parsing failures, null reference errors, or silent corruption. Users lose work or must manually migrate.

**Why it happens:**
- Commands assume new fields exist without validation
- No version checking before accessing nested properties (e.g., `.memory.decisions` when `memory` key missing)
- Migration commands (`/ant:migrate-state`) are separate/optional, not automatic
- Template-based initialization writes new schema, but existing files never updated

**How to avoid:**
1. **Additive-only changes** — never remove or rename fields in minor versions
2. **Default value injection** — on read, use `jq '.new_field // default'` pattern
3. **Auto-migration on access** — `session-read` and `load-state` should silently upgrade old schemas
4. **Version gating** — check `.version` before accessing v3.0+ fields; branch logic accordingly
5. **Graceful degradation** — if field missing, function with reduced capability, not crash

**Warning signs:**
- `jq: error: null has no keys` errors in logs
- Commands fail only on older colonies
- `memory.decisions` is always empty (field exists but never populated)
- Migration command exists but users don't know to run it

**Phase to address:** Phase 1 (Core Infrastructure) — schema validation and auto-migration must be in place before any new fields added

**Affected functions:**
- `session-read` in `.aether/aether-utils.sh:6914`
- `load-state` (sources `state-loader.sh`)
- `validate-state colony` (already has migration logic — extend it)

---

### Pitfall A2: Session Stale vs. Fresh Confusion

**What goes wrong:**
The system cannot distinguish between "user cleared context intentionally" vs. "session crashed/ended unexpectedly." This leads to:
- Offering resume when user started fresh intentionally
- Not offering resume when user wants to continue
- `context_cleared` flag ignored or misinterpreted

**Why it happens:**
- `session.json` persists across `/clear` in Claude Code
- `context_cleared` flag exists but not checked consistently
- Stale detection (>24 hours) is time-based, not intent-based
- Multiple session files (`session.json`, `COLONY_STATE.json`, `HANDOFF.md`) can disagree

**How to avoid:**
1. **Intent tracking** — record *why* session ended: `user_cleared`, `completed`, `crashed`, `unknown`
2. **Unified freshness check** — single source of truth function used by ALL commands
3. **Explicit resume required** — never auto-resume; always ask "Resume previous session?" with preview
4. **Session lineage** — link sessions with parent_session_id to detect gaps

**Warning signs:**
- `/ant:resume` shows stale data after user ran `/ant:init` fresh
- `session-verify-fresh` returns different results than `session-is-stale`
- Users report "it keeps going back to old work"

**Phase to address:** Phase 1 (Core Infrastructure) — fix freshness detection before building on it

**Affected functions:**
- `session-verify-fresh` in `.aether/aether-utils.sh:3181`
- `session-is-stale` in `.aether/aether-utils.sh:6942`
- `session-mark-resumed` in `.aether/aether-utils.sh:6989`

---

### Pitfall A3: Context Overload — Too Much Information

**What goes wrong:**
Context restoration dumps everything (all events, all decisions, full plan) into the session, overwhelming the user and exceeding context windows. The "continue here" marker is lost in noise.

**Why it happens:**
- No prioritization of what matters now vs. what mattered then
- `memory.decisions[]` grows unbounded (no cap enforcement in read path)
- Events array includes every minor state change
- No "summary vs. detail" tiering

**How to avoid:**
1. **Recency-weighted pruning** — only last 5 decisions, last 10 events, current phase only
2. **Hierarchical context** — brief summary first, details on request
3. **Compression** — condense multiple related events into summary sentences
4. **Explicit "you are here" marker** — single line stating current position, always visible

**Warning signs:**
- `/ant:resume` output exceeds 100 lines
- Users scroll past context to find the command prompt
- Claude context window warnings during resume

**Phase to address:** Phase 2 (Context Aggregation) — implement caps and summarization before adding more context sources

**Current caps (from `continue.md`):**
- Max 20 phase_learnings
- Max 30 decisions
- Max 30 instincts
- Max 100 events

These caps exist for *write* but not consistently for *read/display*.

---

### Pitfall A4: Template Explosion

**What goes wrong:**
Adding context restoration spawns new templates for every scenario: resume, handoff, seal, entomb, pause. Templates diverge, contain conflicting placeholders, and maintenance burden explodes.

**Why it happens:**
- Each command creates its own template instead of composing from shared parts
- Placeholder naming inconsistent (`{{GOAL}}` vs `__GOAL__` vs `{goal}`)
- Template inheritance/composition not supported
- No template validation (placeholders left unfilled)

**How to avoid:**
1. **Template composition** — shared header/footer, command-specific body only
2. **Strict naming convention** — all caps with underscores: `{{COLONY_GOAL}}`
3. **Validation layer** — warn if any `{{.*}}` remains after substitution
4. **Minimal templates** — prefer code-generated content over templates for dynamic data

**Warning signs:**
- `.aether/templates/` has 10+ files with similar content
- Same placeholder defined differently in different templates
- Unfilled placeholders appearing in output

**Phase to address:** Phase 1 (Core Infrastructure) — establish template patterns before adding more

**Existing templates:**
- `handoff.template.md`
- `handoff-build-error.template.md`
- `handoff-build-success.template.md`
- `crowned-anthill.template.md`

All use `{{PLACEHOLDER}}` format — maintain this convention.

---

### Pitfall A5: Backwards Compatibility Break in Commands

**What goes wrong:**
Modifying existing commands (`/ant:init`, `/ant:status`) to support context restoration breaks behavior users rely on. Scripts or muscle memory that worked before now fail or act differently.

**Why it happens:**
- Adding required parameters to existing commands
- Changing output format (JSON structure, field names)
- New interactive prompts block non-interactive usage
- Default behavior changes (e.g., auto-resume vs. start fresh)

**How to avoid:**
1. **New commands for new behavior** — `/ant:resume` exists; don't change `/ant:init` to auto-resume
2. **Opt-in flags** — new features behind `--restore-context` or similar
3. **Output format versioning** — if JSON structure changes, provide `--format v2` option
4. **Deprecation cycle** — warn before changing, maintain old behavior as fallback

**Warning signs:**
- Users report "my script broke"
- Documentation examples no longer work
- Different behavior in CI vs. interactive

**Phase to address:** All phases — maintain compatibility discipline throughout

**At-risk commands:**
- `/ant:init` — must not auto-resume; preserve overwrite warning behavior
- `/ant:status` — output format consumed by scripts; maintain JSON structure
- `/ant:build` — must not change phase execution logic

---

### Pitfall A6: Data Loss During Migration

**What goes wrong:**
Migrating state to add context restoration fields loses user data: decisions, learnings, or pheromone signals disappear. User trusts system, continues work, later discovers context is gone.

**Why it happens:**
- Migration script only handles "happy path" fields, misses edge case data
- Arrays overwritten instead of merged
- Timestamps or IDs regenerated, breaking continuity
- No verification that migration preserved all data

**How to avoid:**
1. **Backup before migrate** — always create `.aether/data/backup-{timestamp}/`
2. **Field inventory** — list every field in old state, ensure each has destination
3. **Verification step** — post-migration, confirm decision count, learning count match
4. **Rollback capability** — single command to restore from backup

**Warning signs:**
- Decision count drops after migration
- `memory.phase_learnings` empty after upgrade
- Users say "it feels like it forgot what we were doing"

**Phase to address:** Phase 1 (Core Infrastructure) — migration safety is foundational

**Existing migration (from `migrate-state.md`):**
- Already backs up to `backup-v1/`
- Consolidates 6 files into 1
- Pattern exists — extend for context restoration

---

### Pitfall A7: Lock Contention on Context Files

**What goes wrong:**
Multiple commands try to read/write context simultaneously (e.g., `/ant:build` updating progress while `/ant:status` reading), causing:
- JSON corruption (partial writes)
- Lock timeouts (commands hang)
- Deadlocks (circular lock acquisition)

**Why it happens:**
- Context update added to long-running commands without lock strategy
- File-level locks held too long (duration of entire build)
- Different lock files for different resources (state vs. context vs. session)

**How to avoid:**
1. **Single lock hierarchy** — always acquire locks in same order: state → context → session
2. **Short lock windows** — read data, release lock, process, reacquire only to write
3. **Lock timeouts** — fail fast if lock held >5 seconds, don't hang indefinitely
4. **Atomic writes** — use `atomic_write` utility already in `.aether/utils/atomic-write.sh`

**Warning signs:**
- Commands hang intermittently
- `COLONY_STATE.json` contains malformed JSON
- `.aether/locks/` has stale lock files

**Phase to address:** Phase 1 (Core Infrastructure) — locking strategy must be solid before adding concurrent context updates

**Existing lock infrastructure:**
- `acquire_lock` / `release_lock` in `file-lock.sh`
- `atomic_write` in `atomic-write.sh`
- Already used by `context-update` (GAP-009 fixed in Phase 16)

---

### Pitfall A8: Naming Conflicts with User Data

**What goes wrong:**
New context restoration fields conflict with user-created fields or conventions. User has `session_id` in their own data, or `context` means something different in their domain.

**Why it happens:**
- Generic field names (`context`, `session`, `state`) likely to collide
- No namespacing convention for system vs. user fields
- User data (`.aether/dreams/`, `TO-DOs.md`) not formally separated

**How to avoid:**
1. **Namespace system fields** — `aether_` prefix or `_system` nested object
2. **Reserved field list** — document which field names are system-owned
3. **User data isolation** — never write to user files (dreams, TO-DOs.md)
4. **Migration path for conflicts** — if field exists, rename user's version

**Warning signs:**
- User reports "my session_id got overwritten"
- Unexpected values appearing in system fields
- User data corruption in `dreams/` or `TO-DOs.md`

**Phase to address:** Phase 1 (Core Infrastructure) — establish naming conventions early

**Protected paths (from `known-issues.md`):**
- `.aether/data/` — system state
- `.aether/dreams/` — user notes (NEVER touch)
- `TO-DOs.md` — user notes (NEVER touch)

---

### Pitfall A9: The "Continue Here" Marker Ambiguity

**What goes wrong:**
Multiple competing markers for "where to resume":
- `current_phase` in COLONY_STATE.json
- `suggested_next` in session.json
- Last event in events array
- `HANDOFF.md` narrative

They disagree, and the system picks wrong one or shows conflicting guidance.

**Why it happens:**
- Each feature added its own marker without unifying
- No single source of truth for "current position"
- Markers updated at different times (async drift)

**How to avoid:**
1. **Single position tracker** — `COLONY_STATE.json` is source of truth
2. **Derived, not stored** — `suggested_next` computed from state, not cached
3. **Consistency check** — on resume, verify all markers agree; warn if not
4. **User override** — allow explicit "continue from phase X" to bypass markers

**Warning signs:**
- `/ant:resume` suggests phase 3, but `current_phase` is 2
- `HANDOFF.md` describes different work than status shows
- Users confused about "what should I do next"

**Phase to address:** Phase 2 (Context Aggregation) — unify position tracking before adding more markers

**Current markers:**
- `COLONY_STATE.json:current_phase` — authoritative
- `session.json:suggested_next` — cached recommendation (can be stale)
- `events[]` — audit trail (not position)

---

### Pitfall A10: Context Restoration Without Validation

**What goes wrong:**
Restored context is corrupt, outdated, or wrong project, but system proceeds anyway. User works on wrong assumptions, makes bad decisions based on stale context.

**Why it happens:**
- No checksum/hash validation of context files
- No verification that restored context matches current repo state
- No sanity checks (e.g., phase count reasonable, dates not in future)

**How to avoid:**
1. **Integrity checks** — hash of critical files, verify on restore
2. **Repo matching** — store git remote/branch, warn if restored context from different repo
3. **Sanity bounds** — reject context with impossible values (negative phases, future dates)
4. **Preview before apply** — show what will be restored, ask confirmation

**Warning signs:**
- Context from different project appears
- Timestamps in future or far past
- Phase numbers don't match plan

**Phase to address:** Phase 3 (Validation & Safety) — validation layer before trusting restored context

---

## Context Restoration: Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Store context in session.json only | Simpler, one file | Lost on `/clear`, not persistent | Never — use COLONY_STATE.json |
| Skip migration for "unlikely" edge cases | Less code | Data loss for those users | Never — handle all cases |
| Cache `suggested_next` in session.json | Faster resume | Stale recommendations | Only with timestamp + recompute fallback |
| Use sed/awk for JSON updates | No jq dependency | Corruption risk, no validation | Only for non-critical display updates |
| Hardcode field names in commands | Faster to write | Breaks on schema changes | Never — use shared constants |

---

## Context Restoration: Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Claude Code `/clear` | Assuming session.json survives | Store in `.aether/data/` not `/tmp/`, check on every command |
| Git state | Not tracking baseline commit | Store `baseline_commit`, detect drift on resume |
| jq parsing | Using `.field` without `// default` | Always provide defaults: `.field // "unknown"` |
| File locking | Holding lock during long operations | Release lock before external calls (git, npm) |
| Template rendering | Not escaping user content | Escape `{{` in user strings to prevent injection |

---

## Context Restoration: Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Loading full event history | Slow resume, large memory | Cap events at 100, archive older | >1000 events |
| Unbounded decisions array | JSON parsing slows down | Enforce 30 decision cap | >100 decisions |
| Recomputing context on every command | Lag in interactive use | Cache with invalidation | Every command feels slow |
| Reading multiple large files | I/O bottleneck | Single consolidated state file | Slow disks, network drives |

---

## Context Restoration: "Looks Done But Isn't" Checklist

- [ ] **Schema migration:** Old colonies (v2.0, v1.0) can use new features without manual migration
- [ ] **Field defaults:** All new fields have sensible defaults when missing from old state
- [ ] **Lock safety:** No deadlocks possible even with concurrent context updates
- [ ] **Data preservation:** Migration never loses user decisions, learnings, or signals
- [ ] **Backwards compatibility:** `/ant:init`, `/ant:status`, `/ant:build` work exactly as before
- [ ] **Template validation:** No unfilled placeholders in generated output
- [ ] **Position consistency:** `current_phase`, `suggested_next`, and events all agree
- [ ] **Context caps:** Resume output bounded (decisions, events, learnings limited)
- [ ] **Freshness detection:** Correctly identifies stale vs. fresh sessions
- [ ] **User confirmation:** Preview before restoring context, especially after long gaps

---

## Context Restoration: Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Schema mismatch | LOW | Auto-migrate on access, user sees no interruption |
| Corrupt COLONY_STATE.json | MEDIUM | Restore from `.aether/data/backup-*/`, re-apply recent changes manually |
| Lost session.json | LOW | Reconstruct from COLONY_STATE.json + git log, may lose `suggested_next` |
| Lock deadlock | LOW | `aether force-unlock` command releases stuck locks |
| Wrong context restored | MEDIUM | `/ant:init` fresh start, manually re-apply decisions from memory |
| Template explosion | MEDIUM | Refactor to composition pattern, migrate existing templates |

---

## Context Restoration: Phase-to-Pitfall Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| State Schema Migration Hell | Phase 1: Core Infrastructure | Test with v1.0 and v2.0 state files; verify auto-migration |
| Session Stale vs. Fresh Confusion | Phase 1: Core Infrastructure | Unit tests for `session-verify-fresh` with various timestamps |
| Context Overload | Phase 2: Context Aggregation | Resume output < 50 lines; decisions/events capped |
| Template Explosion | Phase 1: Core Infrastructure | Count templates; verify shared components |
| Backwards Compatibility Break | All phases | Run existing test colonies through new code |
| Data Loss During Migration | Phase 1: Core Infrastructure | Pre/post field count comparison; backup verification |
| Lock Contention | Phase 1: Core Infrastructure | Concurrent command stress test |
| Naming Conflicts | Phase 1: Core Infrastructure | Audit field names against common user terms |
| Continue Here Marker Ambiguity | Phase 2: Context Aggregation | All markers agree in test scenarios |
| Context Restoration Without Validation | Phase 3: Validation & Safety | Corrupt state file handling test |

---

## Aether-Specific Risks

### The `memory.decisions[]` Gap

Current state (from `COLONY_STATE.json` sample):
```json
"memory": {
  "phase_learnings": [...],
  "decisions": [],
  "instincts": []
}
```

`decisions` is always empty. Adding context restoration without populating this field creates a "zombie feature" — users expect decisions to be restored, but none exist to restore.

**Mitigation:** Before enabling context restoration, ensure `/ant:continue` and other commands actually populate `memory.decisions`.

### The Sparse `plan.phases[]` Problem

Current `plan.phases` is often empty or minimally populated. Context restoration that depends on rich phase documentation will fail.

**Mitigation:** Phase 2 should include "plan enrichment" — backfilling phase descriptions even for existing colonies.

### HANDOFF.md vs. CONTEXT.md Duality

Two context documents with overlapping purposes:
- `HANDOFF.md` — created by `/ant:pause-colony`, read by `/ant:resume-colony`
- `CONTEXT.md` — updated by `context-update`, read by `/ant:resume`

Risk of divergence or one becoming stale.

**Mitigation:** Consolidate or establish clear ownership boundaries. Prefer `CONTEXT.md` as primary (already has update infrastructure).

---

## Sources

- Aether codebase analysis:
  - `.aether/data/COLONY_STATE.json` — current state schema
  - `.aether/data/session.json` — session tracking
  - `.aether/aether-utils.sh` — utility functions (session-read, load-state, context-update)
  - `.claude/commands/ant/resume.md` — resume command implementation
  - `.claude/commands/ant/migrate-state.md` — migration patterns
  - `.aether/docs/known-issues.md` — historical bugs and fixes
- CLI tool best practices (general domain knowledge)
- State management anti-patterns (general domain knowledge)

---

*Appendix for: Colony Context Enhancement milestone*
*Focus: Adding instant session restoration to existing working system*
