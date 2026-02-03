# Phase 19: Audit Fixes + Utility Scaffold - Research

**Researched:** 2026-02-03
**Domain:** Shell scripting, JSON state management, audit remediation
**Confidence:** HIGH

## Summary

Phase 19 fixes 11 audit-identified issues in the Aether Queen Ant Colony and creates a scaffold for `aether-utils.sh` with subcommand dispatch. The codebase has two layers of history: the v1/v2 committed code (nested COLONY_STATE schema, `active_pheromones` array with `signal`/`timestamp` fields, Python utilities, complex worker objects) and the v3 working-directory code (flat COLONY_STATE schema, `signals` array with `content`/`created_at` fields, pure prompt-based commands). The v3 rebuild already simplified the schema significantly, meaning several audit issues identified against the v1/v2 code are partially or fully resolved in the working copy.

The primary work is: (1) ensure the v3 flat schema is the canonical committed version, (2) fix remaining inconsistencies within the v3 commands themselves, (3) harden all state operations with temp file uniqueness, jq error checking, and backup rotation, and (4) create the aether-utils.sh scaffold that Phase 20 will populate with modules.

**Primary recommendation:** Address fixes in dependency order -- schema canonicalization first (FIX-01 through FIX-03), then hardening (FIX-04 through FIX-08), then polish (FIX-09 through FIX-11), then utility scaffold (UTIL-01 through UTIL-04). Group into 3 plans matching these natural clusters.

## Standard Stack

### Core (Already Present)
| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| bash | 3.2+ (macOS default) | Shell scripting for utilities | Zero external deps, Claude Bash tool compatible |
| jq | 1.6+ | JSON manipulation in shell | Standard for JSON ops in shell, already used throughout |
| python3 | 3.x | JSON validation (used in atomic-write.sh) | Already present on macOS, used for validation fallback |
| git | 2.x | Version control, checkpoint mechanism | Already used for phase checkpoints |

### Supporting
| Tool | Purpose | When to Use |
|------|---------|-------------|
| date | Timestamps for backups and temp files | Every state operation |
| mktemp | Could replace manual temp file creation | Alternative to PID+timestamp pattern |
| cp | Backup creation | Before state modifications |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Manual PID+timestamp temp files | mktemp | mktemp is cleaner but less explicit in naming; PID+timestamp chosen per requirements |
| python3 JSON validation | jq validation (`jq . file.json > /dev/null 2>&1`) | jq is lighter, already a dependency; python3 adds startup overhead |
| Separate backup scripts | Existing atomic-write.sh backup functions | atomic-write.sh already has create_backup/rotate_backups -- use those |

## Architecture Patterns

### Current File Layout
```
.aether/
  data/
    COLONY_STATE.json      # Colony state (flat schema in v3)
    pheromones.json         # Pheromone signals
    errors.json             # Error tracking
    memory.json             # Phase learnings, decisions
    events.json             # Event log
    PROJECT_PLAN.json       # Project phases and tasks
    backups/                # Timestamped state backups (to be created)
  utils/
    file-lock.sh            # File locking (acquire_lock, release_lock)
    atomic-write.sh         # Atomic write with backup (sources file-lock.sh)
  aether-utils.sh           # NEW: Entry point with subcommand dispatch
  temp/                     # Temp files for atomic operations
  locks/                    # Lock files
.claude/
  commands/ant/
    ant.md                  # Help text / command listing
    init.md                 # Initialize colony
    plan.md                 # Generate project plan
    build.md                # Execute phase
    continue.md             # Advance to next phase
    status.md               # Display colony dashboard
    phase.md                # View phase details
    focus.md                # Emit FOCUS pheromone
    redirect.md             # Emit REDIRECT pheromone
    feedback.md             # Emit FEEDBACK pheromone
    colonize.md             # Analyze existing codebase
    pause-colony.md         # Save state for session break
    resume-colony.md        # Restore from pause
```

### Pattern 1: Canonical Flat Schema
**What:** COLONY_STATE.json uses flat top-level fields for the 3 most-accessed values.
**When to use:** Every command that reads or writes colony state.

The v3 commands (init.md) write this schema:
```json
{
  "goal": "<user's goal>",
  "state": "READY",
  "current_phase": 0,
  "session_id": "<session_id>",
  "initialized_at": "<timestamp>",
  "workers": {
    "colonizer": "idle",
    "route-setter": "idle",
    "builder": "idle",
    "watcher": "idle",
    "scout": "idle",
    "architect": "idle"
  },
  "spawn_outcomes": { ... }
}
```

This is the canonical schema. All commands read `.goal`, `.state`, `.current_phase` at root level. No nested `queen_intention.goal`, `colony_status.state`, or `phases.current_phase`.

### Pattern 2: Pheromone Signal Schema
**What:** Pheromones use `signals` array with `content`/`created_at`/`half_life_seconds` fields.
**When to use:** All pheromone operations.

The v3 commands write this schema:
```json
{
  "signals": [
    {
      "id": "init_<timestamp>",
      "type": "INIT",
      "content": "<description>",
      "strength": 1.0,
      "half_life_seconds": null,
      "created_at": "<ISO-8601>"
    }
  ]
}
```

This replaces the old `active_pheromones` array with `signal`/`timestamp` fields. The field `content` replaces `signal`, `created_at` replaces `timestamp`, and `half_life_seconds` replaces `decay_rate`.

### Pattern 3: Subcommand Dispatch
**What:** Single entry point script with case statement dispatch.
**When to use:** aether-utils.sh scaffold.

```bash
#!/bin/bash
# aether-utils.sh - Aether Colony Utility Layer
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Source shared infrastructure
source "$SCRIPT_DIR/utils/file-lock.sh"
source "$SCRIPT_DIR/utils/atomic-write.sh"

# JSON output helpers
json_success() { echo "{\"ok\":true,\"result\":$1}"; }
json_error()   { echo "{\"ok\":false,\"error\":\"$1\"}" >&2; exit 1; }

case "${1:-help}" in
  help)
    cat <<'HELP'
{"ok":true,"commands":["help","version"]}
HELP
    ;;
  version)
    json_success '"0.1.0"'
    ;;
  *)
    json_error "Unknown command: $1"
    ;;
esac
```

### Anti-Patterns to Avoid
- **Nested field access for hot-path data:** Do not add `queen_intention.goal` or `colony_status.current_phase`. The flat schema `.goal` and `.current_phase` is canonical.
- **Hardcoded temp file paths:** Never use `/tmp/colony_state.tmp`. Always use `$$.$(date +%s%N)` suffix.
- **Silent jq failures:** Never redirect jq output without checking exit code.
- **Python where jq suffices:** Do not use python3 for JSON operations that jq can handle. Reserve python3 only for validation where jq is insufficient.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Atomic file writes | Manual temp+mv | `atomic_write` / `atomic_write_from_file` from atomic-write.sh | Already handles temp file creation, JSON validation, backup, sync |
| File locking | Manual lock files | `acquire_lock` / `release_lock` from file-lock.sh | Already handles stale lock detection, PID tracking, retry with timeout |
| Backup rotation | Manual cleanup logic | `create_backup` / `rotate_backups` from atomic-write.sh | Already implemented, just needs MAX_BACKUPS adjusted from 5 to 3 |
| JSON validation | Custom validators | `jq . file.json > /dev/null 2>&1` or existing python3 pattern in atomic-write.sh | Sufficient for Phase 19; Phase 20 adds schema-level validation |
| Subcommand parsing | getopt/getopts | Simple `case` statement | Under 10 subcommands; case is clearer and more portable |

## Common Pitfalls

### Pitfall 1: Audit Report vs Current Code Mismatch
**What goes wrong:** The audit report (SYSTEM_AUDIT_REPORT.md) was written against the v1/v2 code. The v3 rebuild (phases 14-17) already fixed several issues but the audit report does not reflect this.
**Why it happens:** The v3 rebuild changed the command prompts and simplified the schema, but this happened after the audit was written.
**How to avoid:** Cross-reference every audit finding against the CURRENT command files (which use the v3 flat schema). Do not blindly apply audit fixes that reference `queen_intention.goal` or `active_pheromones` -- those paths no longer exist in the v3 commands.
**Warning signs:** Audit fix references `.queen_intention.goal`, `.colony_status.current_phase`, `.active_pheromones`, or `worker_ants` (with nested objects instead of simple strings).

**Specific findings:**
| Audit Issue | Status in Current Code | Action Needed |
|-------------|----------------------|---------------|
| FIX-01: atomic-write.sh missing file-lock.sh | ALREADY FIXED (line 22 of current file) | Verify only -- already works |
| FIX-02: Duplicate goal/current_phase fields | PARTIALLY FIXED by v3 flat schema | Ensure committed state matches v3 schema |
| FIX-03: Inconsistent field paths | PARTIALLY FIXED by v3 commands | Verify all 13 commands use flat paths |
| FIX-07: Pheromone schema mismatch | PARTIALLY FIXED by v3 schema | Ensure `signals` array with `content`/`created_at` is canonical |
| FIX-09: Worker status casing | v3 uses lowercase "idle" in init.md | Requirement says lowercase; v3 already does this |

### Pitfall 2: Committed vs Working Directory State
**What goes wrong:** The committed COLONY_STATE.json has the OLD nested schema. The working directory has the NEW flat schema. If someone checks out fresh, they get the old schema.
**Why it happens:** The v3 rebuild modified data files but they were never committed as the canonical reset state.
**How to avoid:** Phase 19 must commit the canonical v3 schema as the reset/default state for COLONY_STATE.json and pheromones.json.
**Warning signs:** `git show HEAD:.aether/data/COLONY_STATE.json` shows nested `colony_status.state` while commands expect flat `.state`.

### Pitfall 3: Backup Directory Location
**What goes wrong:** atomic-write.sh uses `$AETHER_ROOT/.aether/backups/` but the success criteria says `.aether/data/backups/`.
**Why it happens:** The backup directory was set when atomic-write.sh was created, before the data/ subdirectory convention was established.
**How to avoid:** Change BACKUP_DIR in atomic-write.sh to `$AETHER_ROOT/.aether/data/backups/` to match success criteria.
**Warning signs:** Backups created in wrong directory, success criteria #5 fails.

### Pitfall 4: MAX_BACKUPS Mismatch
**What goes wrong:** atomic-write.sh sets `MAX_BACKUPS=5` but success criteria says "at most 3 backups retained per file".
**Why it happens:** Original code used 5, requirement changed to 3.
**How to avoid:** Change MAX_BACKUPS from 5 to 3 in atomic-write.sh.

### Pitfall 5: aether-utils.sh Location
**What goes wrong:** Placing aether-utils.sh in wrong directory.
**Why it happens:** Success criteria says `bash .aether/aether-utils.sh help` -- so it lives at `.aether/aether-utils.sh`, NOT in `.aether/utils/`.
**How to avoid:** Place at `.aether/aether-utils.sh` (one level up from utils/).

### Pitfall 6: Commands Use Write Tool, Not Shell
**What goes wrong:** Trying to add jq error checking to command .md files as shell code.
**Why it happens:** The v3 commands use Claude's Write tool to update JSON files directly -- they do NOT use jq or shell scripts. The jq-based patterns from the audit report apply to the OLD v1/v2 Python/bash commands, not the current v3 prompt-based commands.
**How to avoid:** For v3 commands, FIX-05 (jq error checking) and FIX-04 (unique temp files) apply only to the shell utility scripts (atomic-write.sh, file-lock.sh, future aether-utils.sh), NOT to the .md command prompts. The .md files instruct Claude to use Read/Write tools for state operations.
**Warning signs:** Adding `jq` commands to .md files when the .md files use `Write tool` for file operations.

## Code Examples

### Example 1: Canonical COLONY_STATE.json (v3 flat schema)
```json
{
  "goal": null,
  "state": "IDLE",
  "current_phase": 0,
  "session_id": null,
  "initialized_at": null,
  "workers": {
    "colonizer": "idle",
    "route-setter": "idle",
    "builder": "idle",
    "watcher": "idle",
    "scout": "idle",
    "architect": "idle"
  },
  "spawn_outcomes": {
    "colonizer":    {"alpha": 1, "beta": 1, "total_spawns": 0, "successes": 0, "failures": 0},
    "route-setter": {"alpha": 1, "beta": 1, "total_spawns": 0, "successes": 0, "failures": 0},
    "builder":      {"alpha": 1, "beta": 1, "total_spawns": 0, "successes": 0, "failures": 0},
    "watcher":      {"alpha": 1, "beta": 1, "total_spawns": 0, "successes": 0, "failures": 0},
    "scout":        {"alpha": 1, "beta": 1, "total_spawns": 0, "successes": 0, "failures": 0},
    "architect":    {"alpha": 1, "beta": 1, "total_spawns": 0, "successes": 0, "failures": 0}
  }
}
```

### Example 2: Canonical pheromones.json (v3 schema)
```json
{
  "signals": []
}
```

Each signal uses this structure:
```json
{
  "id": "init_1770053000",
  "type": "INIT",
  "content": "Build a REST API with authentication",
  "strength": 1.0,
  "half_life_seconds": null,
  "created_at": "2026-02-03T12:00:00Z"
}
```

### Example 3: aether-utils.sh Scaffold
```bash
#!/bin/bash
# Aether Colony Utility Layer
# Single entry point for deterministic colony operations
#
# Usage: bash .aether/aether-utils.sh <subcommand> [args...]
#
# All subcommands output JSON to stdout.
# Non-zero exit on error with JSON error message.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$SCRIPT_DIR/.." && pwd 2>/dev/null || echo "$SCRIPT_DIR")"
DATA_DIR="$AETHER_ROOT/.aether/data"

# Source shared infrastructure
source "$SCRIPT_DIR/utils/file-lock.sh"
source "$SCRIPT_DIR/utils/atomic-write.sh"

# --- JSON output helpers ---
json_ok()    { printf '{"ok":true,"result":%s}\n' "$1"; }
json_err()   { printf '{"ok":false,"error":"%s"}\n' "$1" >&2; return 1; }

# --- Subcommand dispatch ---
cmd="${1:-help}"
shift 2>/dev/null || true

case "$cmd" in
  help)
    cat <<'EOF'
{"ok":true,"commands":["help","version"],"description":"Aether Colony Utility Layer"}
EOF
    ;;
  version)
    json_ok '"0.1.0"'
    ;;
  *)
    json_err "Unknown command: $cmd"
    ;;
esac
```

### Example 4: Unique Temp File Pattern
```bash
# In shell scripts (NOT in .md command prompts):
TEMP_FILE="${TEMP_DIR}/$(basename "$target_file").$$_$(date +%s%N).tmp"

# $$ = current PID
# $(date +%s%N) = unix timestamp with nanoseconds
# Combined: unique even under concurrent access
```

### Example 5: jq Error Checking Pattern (for shell scripts only)
```bash
# Pattern for aether-utils.sh subcommands:
result=$(jq '.goal' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null) || {
  json_err "Failed to read COLONY_STATE.json -- file may be corrupted"
}
```

### Example 6: Backup with Rotation to 3
```bash
# In atomic-write.sh, change:
MAX_BACKUPS=5
# To:
MAX_BACKUPS=3

# And change:
BACKUP_DIR="$AETHER_ROOT/.aether/backups"
# To:
BACKUP_DIR="$AETHER_ROOT/.aether/data/backups"
```

## State of the Art

| Old Approach (v1/v2) | Current Approach (v3) | When Changed | Impact |
|----------------------|----------------------|--------------|--------|
| Nested COLONY_STATE: `queen_intention.goal`, `colony_status.state` | Flat: `.goal`, `.state`, `.current_phase` | v3.0 (Phase 14-17) | All commands simplified; must commit as canonical |
| `active_pheromones` array with `signal`/`timestamp` fields | `signals` array with `content`/`created_at`/`half_life_seconds` | v3.0 (Phase 14-17) | Cleaner schema; old data in committed state must be replaced |
| Worker objects: `{status, current_task, spawned_subagents}` | Simple strings: `"idle"`, `"active"` | v3.0 (Phase 14-17) | Simpler state; worker detail moved to runtime |
| Python/bash utility scripts for operations | Prompt-only (Claude Read/Write tools) | v3.0 (Phase 14-17) | Commands don't use shell; v4.0 adds shell back as optional utility layer |
| Multiple state files (worker_ants.json, watcher_weights.json) | Single COLONY_STATE.json + pheromones.json + errors.json + memory.json + events.json | v3.0 (Phase 14-17) | Fewer files, clearer ownership |

**Deprecated/outdated:**
- `worker_ants.json` -- replaced by `workers` field in COLONY_STATE.json
- `watcher_weights.json` -- no longer used in v3 commands
- `active_pheromones` array -- replaced by `signals` array
- `queen_intention.goal` path -- replaced by flat `.goal`
- `colony_status.current_phase` path -- replaced by flat `.current_phase`
- All Python files (.aether/*.py, .aether/memory/*.py) -- v3 is prompt-only + shell utilities

## Detailed Fix Analysis

### FIX-01: atomic-write.sh sources file-lock.sh
**Status:** ALREADY FIXED in current working code (lines 10-22 of atomic-write.sh)
**Action:** Verify only. Source command test: `source .aether/utils/atomic-write.sh && type acquire_lock`
**Confidence:** HIGH -- verified by running the command.

### FIX-02: Canonical goal and current_phase paths
**Status:** v3 commands already use flat `.goal` and `.current_phase` at root
**Action:** Commit the v3 canonical COLONY_STATE.json schema. Remove any leftover nested references.
**Canonical paths:** `.goal`, `.state`, `.current_phase`, `.session_id`, `.initialized_at`, `.workers`, `.spawn_outcomes`
**Confidence:** HIGH -- verified by reading all 13 command files.

### FIX-03: All commands use canonical field paths
**Status:** All v3 commands consistently use `goal` (not `queen_intention.goal`), `state` (not `colony_status.state`), and `current_phase` (not `colony_status.current_phase` or `phases.current_phase`)
**Action:** Verify all 13 command .md files use flat paths. One potential issue: phase.md line 24 says "COLONY_STATE.current_phase" which is prose, not a jq path -- it means the field `current_phase` in COLONY_STATE.json. This is correct.
**Confidence:** HIGH -- all commands read.

### FIX-04: Unique temp file suffixes
**Status:** atomic-write.sh already uses `.$$.$(date +%s%N).tmp` pattern (line 52, 107)
**Action:** Verify pattern is correct. The v3 commands don't use temp files directly (they use Write tool). This fix applies only to shell utility code.
**Confidence:** HIGH -- verified in atomic-write.sh code.

### FIX-05: jq exit code checking
**Status:** The v3 commands don't use jq (they use Claude Write tool). atomic-write.sh uses python3 for JSON validation.
**Action:** Add jq error checking to aether-utils.sh scaffold pattern. Not needed in .md command files.
**Confidence:** HIGH -- scope is clear.

### FIX-06: State file backups before critical updates
**Status:** atomic-write.sh already creates backups (lines 71-73). Needs: (a) change backup dir to `.aether/data/backups/`, (b) change MAX_BACKUPS from 5 to 3.
**Action:** Two-line change in atomic-write.sh.
**Confidence:** HIGH -- straightforward.

### FIX-07: Pheromone schema consistency
**Status:** v3 commands all use `signals` array with `id`, `type`, `content`, `strength`, `half_life_seconds`, `created_at`. The auto-emit pattern in continue.md uses slightly different field names (`emitted_at` instead of `created_at`, and adds `source` and `auto` fields).
**Action:** Standardize continue.md auto-emit to use `created_at` (matching all other commands). The `source` and `auto` fields are additive and not harmful -- keep them.
**Specific fix in continue.md:** Change `"emitted_at"` to `"created_at"` in the auto-emit pheromone template (lines 133, 148).
**Confidence:** HIGH -- verified by reading all pheromone-emitting commands.

### FIX-08: State file validation on load
**Status:** v3 commands check if files exist and if goal is null, but don't validate JSON structure.
**Action:** Add validation guidance to commands that read state. For v3 commands (which use Read tool), this means: "If the file content is not valid JSON, output error message with recovery instructions." The shell-level validation will be in aether-utils.sh (Phase 20).
**Confidence:** MEDIUM -- prompt-based validation is advisory, not enforced.

### FIX-09: Worker status casing
**Status:** v3 init.md already uses lowercase: `"idle"`. build.md sets workers to `"active"`. continue.md resets to `"idle"`.
**Action:** Verify all commands that set worker status use lowercase. Add explicit note about canonical values in a comment.
**Canonical values:** "idle", "active", "error"
**Note:** The requirement says lowercase. The committed v1/v2 code used "IDLE". The v3 code already uses lowercase. Just need to ensure all paths are consistent.
**Confidence:** HIGH -- verified in command files.

### FIX-10: Expired pheromone cleanup
**Status:** continue.md Step 5 already cleans expired pheromones (removes signals below 0.05 strength). status.md computes decay but only for display, doesn't clean.
**Action:** continue.md already handles this. Could add cleanup to status.md as well (write cleaned pheromones after display). The requirement says "during reads" -- status.md is the main read path.
**Confidence:** HIGH -- continue.md already implements this.

### FIX-11: Colony mode documentation
**Status:** The v3 system doesn't use a colony_mode concept. The old v1/v2 had production/development modes. In v3, init.md simply initializes the colony with a goal -- there's no mode selection.
**Action:** Since v3 simplified away the mode concept, FIX-11 should document how the colony works in ant.md and init.md help text. This is a documentation-only change.
**Confidence:** HIGH -- straightforward text change.

## Utility Scaffold Specification

### UTIL-01: aether-utils.sh entry point
**Location:** `.aether/aether-utils.sh` (per success criteria: `bash .aether/aether-utils.sh help`)
**Pattern:** case-based subcommand dispatch
**Initial subcommands:** `help`, `version` (Phase 20 adds the real modules)
**Line budget:** Under 50 lines for scaffold (Phase 20 adds modules within 300 total)

### UTIL-02: Sources shared infrastructure
**Sources:** `file-lock.sh` and `atomic-write.sh` from `.aether/utils/`
**Path resolution:** Use `BASH_SOURCE[0]` dirname to find utils/ relative to script

### UTIL-03: JSON stdout output
**Pattern:** All subcommands output JSON. Success: `{"ok":true,"result":...}`. Help outputs JSON listing available commands.

### UTIL-04: Non-zero exit on error
**Pattern:** Error exits with code 1 and JSON error message to stderr: `{"ok":false,"error":"..."}`

## Plan Grouping Recommendation

The 15 requirements (FIX-01 through FIX-11, UTIL-01 through UTIL-04) naturally group into 3 plans:

### Plan 19-01: Schema Canonicalization (FIX-01, FIX-02, FIX-03, FIX-07, FIX-09)
- Verify FIX-01 (already fixed)
- Commit canonical v3 COLONY_STATE.json schema (FIX-02)
- Verify all commands use flat paths (FIX-03)
- Fix continue.md `emitted_at` -> `created_at` (FIX-07)
- Verify lowercase worker statuses (FIX-09)
- Commit canonical v3 pheromones.json schema

### Plan 19-02: Hardening (FIX-04, FIX-05, FIX-06, FIX-08, FIX-10)
- Verify temp file uniqueness pattern in atomic-write.sh (FIX-04)
- Add jq error checking pattern for utility scripts (FIX-05)
- Fix backup dir to `.aether/data/backups/`, MAX_BACKUPS=3 (FIX-06)
- Add state validation guidance to commands (FIX-08)
- Add pheromone cleanup to status.md (FIX-10)

### Plan 19-03: Documentation + Utility Scaffold (FIX-11, UTIL-01, UTIL-02, UTIL-03, UTIL-04)
- Add colony system description to ant.md and init.md (FIX-11)
- Create aether-utils.sh scaffold with dispatch (UTIL-01)
- Source file-lock.sh and atomic-write.sh (UTIL-02)
- JSON stdout output pattern (UTIL-03)
- Non-zero exit with JSON error (UTIL-04)

## Open Questions

1. **Deleted files in working directory:** The git status shows many deleted files (.aether/*.py, .aether/utils/*.sh scripts like state-machine.sh, event-bus.sh, etc.). These were v1/v2 code removed during v3 rebuild. Should Phase 19 commit these deletions, or is that out of scope?
   - What we know: The v3 system does not use these files. Commands are prompt-only.
   - What's unclear: Whether committing the deletions should happen in Phase 19 or was expected to happen in v3.
   - Recommendation: Phase 19 should commit the clean v3 state (including deletions) as its first action, establishing a clean baseline.

2. **ant:memory command:** The audit references a `/ant:memory` command, but no memory.md exists in the current command files. It was likely deleted in the v3 rebuild. The SYSTEM_AUDIT_REPORT discusses it extensively (Issue #3) but it's not in scope for Phase 19 fixes.
   - Recommendation: Ignore Issue #3 from the audit. It references deleted code.

3. **continue.md auto-emit pheromone fields:** The auto-emit template uses `emitted_at` and `source` fields not present in other pheromone signals. Should these be added to the canonical schema or removed?
   - Recommendation: Rename `emitted_at` to `created_at` for consistency. Keep `source` and `auto` as optional fields (they provide useful provenance).

## Sources

### Primary (HIGH confidence)
- Direct file reads of all 13 command .md files in `.claude/commands/ant/`
- Direct file reads of `.aether/utils/atomic-write.sh` and `.aether/utils/file-lock.sh`
- Direct file reads of `.aether/data/COLONY_STATE.json` (working copy) and `git show HEAD:.aether/data/COLONY_STATE.json` (committed)
- Direct file reads of `.aether/data/pheromones.json` (working copy) and committed version
- Bash verification: `source .aether/utils/atomic-write.sh && type acquire_lock` confirmed FIX-01 already resolved
- `.ralph/SYSTEM_AUDIT_REPORT.md` -- full audit with 11 issues (cross-referenced against current code)
- `.planning/ROADMAP.md` -- v4.0 phase definitions and success criteria
- `.planning/REQUIREMENTS.md` -- all 15 Phase 19 requirements

### Secondary (MEDIUM confidence)
- Git history (`git log --oneline` for atomic-write.sh) confirming fix was applied in commit 59283ae

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all tools already present in codebase, verified
- Architecture: HIGH -- full codebase read, all 13 commands analyzed
- Pitfalls: HIGH -- critical mismatch between audit report and current code identified and documented
- Fix analysis: HIGH -- every fix cross-referenced against actual current code

**Research date:** 2026-02-03
**Valid until:** 2026-03-03 (stable codebase, no external dependencies changing)
