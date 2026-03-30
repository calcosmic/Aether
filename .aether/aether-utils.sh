#!/bin/bash
# Aether Colony Utility Layer
# Single entry point for deterministic colony operations
#
# Usage: bash .aether/aether-utils.sh <subcommand> [args...]
#
# All subcommands output JSON to stdout.
# Non-zero exit on error with JSON error message to stderr.

set -euo pipefail

# Set up structured error handling for unexpected failures
# This works alongside set -e but provides better context (line number, command)
# The error_handler function is defined in error-handler.sh if sourced
trap 'if type error_handler &>/dev/null; then error_handler ${LINENO} "$BASH_COMMAND" $?; fi' ERR

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="${AETHER_ROOT:-$(cd "$SCRIPT_DIR/.." && pwd 2>/dev/null || echo "$SCRIPT_DIR")}"  # SUPPRESS:OK -- read-default: directory may not exist
DATA_DIR="${DATA_DIR:-$AETHER_ROOT/.aether/data}"
COLONY_DATA_DIR="$DATA_DIR"  # Default, overridden by _resolve_colony_data_dir

# Initialize lock state before sourcing (file-lock.sh trap needs these)
LOCK_ACQUIRED=${LOCK_ACQUIRED:-false}
CURRENT_LOCK=${CURRENT_LOCK:-""}

# Source shared infrastructure if available
[[ -f "$SCRIPT_DIR/utils/file-lock.sh" ]] && source "$SCRIPT_DIR/utils/file-lock.sh"
[[ -f "$SCRIPT_DIR/utils/atomic-write.sh" ]] && source "$SCRIPT_DIR/utils/atomic-write.sh"
[[ -f "$SCRIPT_DIR/utils/error-handler.sh" ]] && source "$SCRIPT_DIR/utils/error-handler.sh"
[[ -f "$SCRIPT_DIR/utils/chamber-utils.sh" ]] && source "$SCRIPT_DIR/utils/chamber-utils.sh"
[[ -f "$SCRIPT_DIR/utils/xml-utils.sh" ]] && source "$SCRIPT_DIR/utils/xml-utils.sh"
[[ -f "$SCRIPT_DIR/utils/semantic-cli.sh" ]] && source "$SCRIPT_DIR/utils/semantic-cli.sh"
[[ -f "$SCRIPT_DIR/utils/hive.sh" ]] && source "$SCRIPT_DIR/utils/hive.sh"
[[ -f "$SCRIPT_DIR/utils/midden.sh" ]] && source "$SCRIPT_DIR/utils/midden.sh"
[[ -f "$SCRIPT_DIR/utils/skills.sh" ]] && source "$SCRIPT_DIR/utils/skills.sh"
[[ -f "$SCRIPT_DIR/utils/state-api.sh" ]] && source "$SCRIPT_DIR/utils/state-api.sh"
[[ -f "$SCRIPT_DIR/utils/flag.sh" ]] && source "$SCRIPT_DIR/utils/flag.sh"
[[ -f "$SCRIPT_DIR/utils/spawn.sh" ]] && source "$SCRIPT_DIR/utils/spawn.sh"
[[ -f "$SCRIPT_DIR/utils/session.sh" ]] && source "$SCRIPT_DIR/utils/session.sh"
[[ -f "$SCRIPT_DIR/utils/suggest.sh" ]] && source "$SCRIPT_DIR/utils/suggest.sh"
[[ -f "$SCRIPT_DIR/utils/queen.sh" ]] && source "$SCRIPT_DIR/utils/queen.sh"
[[ -f "$SCRIPT_DIR/utils/swarm.sh" ]] && source "$SCRIPT_DIR/utils/swarm.sh"
[[ -f "$SCRIPT_DIR/utils/learning.sh" ]] && source "$SCRIPT_DIR/utils/learning.sh"
[[ -f "$SCRIPT_DIR/utils/pheromone.sh" ]] && source "$SCRIPT_DIR/utils/pheromone.sh"
[[ -f "$SCRIPT_DIR/utils/scan.sh" ]] && source "$SCRIPT_DIR/utils/scan.sh"
[[ -f "$SCRIPT_DIR/utils/emoji-audit.sh" ]] && source "$SCRIPT_DIR/utils/emoji-audit.sh"
[[ -f "$SCRIPT_DIR/utils/immune.sh" ]] && source "$SCRIPT_DIR/utils/immune.sh"
[[ -f "$SCRIPT_DIR/utils/council.sh" ]] && source "$SCRIPT_DIR/utils/council.sh"

# Fallback error constants if error-handler.sh wasn't sourced
# This prevents "unbound variable" errors in older installations
: "${E_UNKNOWN:=E_UNKNOWN}"
: "${E_HUB_NOT_FOUND:=E_HUB_NOT_FOUND}"
: "${E_REPO_NOT_INITIALIZED:=E_REPO_NOT_INITIALIZED}"
: "${E_FILE_NOT_FOUND:=E_FILE_NOT_FOUND}"
: "${E_JSON_INVALID:=E_JSON_INVALID}"
: "${E_LOCK_FAILED:=E_LOCK_FAILED}"
: "${E_LOCK_STALE:=E_LOCK_STALE}"
: "${E_GIT_ERROR:=E_GIT_ERROR}"
: "${E_VALIDATION_FAILED:=E_VALIDATION_FAILED}"
: "${E_FEATURE_UNAVAILABLE:=E_FEATURE_UNAVAILABLE}"
: "${E_BASH_ERROR:=E_BASH_ERROR}"
: "${E_DEPENDENCY_MISSING:=E_DEPENDENCY_MISSING}"
: "${E_RESOURCE_NOT_FOUND:=E_RESOURCE_NOT_FOUND}"

# --- Deprecation warning helper ---
_deprecation_warning() {
  local cmd_name="$1"
  printf '[deprecated] %s -- will be removed in v3.0\n' "$cmd_name" >&2
}

# --- Sanitize-on-read helper ---
# Normalize a string value read from JSON that may contain raw control characters
# from pre-escaping-era writes. Strips literal control chars (except \n \t),
# and trims leading/trailing whitespace. Returns cleaned string on stdout.
# This is defensive normalization -- if data was written correctly, it's a no-op.
sanitize_read_value() {
  local raw="${1:-}"
  [[ -z "$raw" ]] && return 0
  # Strip ASCII control characters except newline (0x0A) and tab (0x09)
  printf '%s' "$raw" | tr -d '\000-\010\013\014\016-\037'
}

# Fallback atomic_write if not sourced (uses temp file + mv for true atomicity)
# Uses TEMP_DIR to avoid issues with paths containing spaces in $TMPDIR
if ! type atomic_write &>/dev/null; then
  atomic_write() {
    local target="$1"
    local content="$2"
    local temp_dir="${TEMP_DIR:-${AETHER_ROOT:-$PWD}/.aether/temp}"
    mkdir -p "$temp_dir" 2>/dev/null || true  # SUPPRESS:OK -- idempotent: harmless if exists
    local temp="${temp_dir}/atomic-write.$$.$(date +%s%N).tmp"
    echo "$content" > "$temp"
    mv "$temp" "$target"
  }
fi

# --- JSON output helpers ---
# Success: JSON to stdout, exit 0
json_ok() { printf '{"ok":true,"result":%s}\n' "$1"; }

# Error: JSON to stderr, exit 1
# Use enhanced json_err from error-handler.sh if available, otherwise fallback
if ! type json_err &>/dev/null; then
  # Fallback: error-handler.sh failed to load. Emits minimal but parseable JSON.
  # Diagnostic note tells the user their installation may be incomplete.
  json_err() {
    local code="${1:-E_UNKNOWN}"
    local message="${2:-An unknown error occurred}"
    printf '[aether] Warning: error-handler.sh not loaded — using minimal fallback\n' >&2
    printf '%s\n' "$(jq -nc --arg c "$code" --arg m "$message" '{ok:false,error:{code:$c,message:$m}}')" >&2
    exit 1
  }
fi

# Feature detection for graceful degradation
# ARCH-09: runs AFTER all fallback definitions (atomic_write, json_ok, json_err)
# so feature_disable is never called before those functions exist.
# These checks run silently - failures are logged but don't block operation
if type feature_disable &>/dev/null; then
  # Check if DATA_DIR is writable for activity logging
  [[ -w "$DATA_DIR" ]] 2>/dev/null || feature_disable "activity_log" "DATA_DIR not writable"  # SUPPRESS:OK -- existence-test: value may not be numeric

  # Check if git is available for git integration
  command -v git &>/dev/null || feature_disable "git_integration" "git not installed"

  # Check if jq is available for JSON processing
  command -v jq &>/dev/null || feature_disable "json_processing" "jq not installed"

  # Check if lock utilities are available
  [[ -f "$SCRIPT_DIR/utils/file-lock.sh" ]] || feature_disable "file_locking" "lock utilities not available"
fi

# Composed exit cleanup — replaces individual traps from file-lock.sh and atomic-write.sh
# ARCH-10: bash traps are single-valued per signal — last trap set wins.
# This function ensures both lock and temp cleanup run on every exit path.
# Must be set AFTER file-lock.sh is sourced so it overrides the individual
# 'trap cleanup_locks EXIT TERM INT HUP' set by file-lock.sh.
_aether_exit_cleanup() {
    cleanup_locks 2>/dev/null || true  # SUPPRESS:OK -- cleanup: exit handler must not fail
    cleanup_temp_files 2>/dev/null || true  # SUPPRESS:OK -- cleanup: exit handler must not fail
}
trap '_aether_exit_cleanup' EXIT TERM INT HUP

# Startup cleanup — remove temp files from dead sessions (PID-based orphan detection)
# ARCH-10: runs once at startup, silent (matches lock cleanup behavior)
_cleanup_orphaned_temp_files() {
    local temp_dir="${TEMP_DIR:-$AETHER_ROOT/.aether/temp}"
    [[ -d "$temp_dir" ]] || return 0
    while IFS= read -r -d '' tmp_file; do
        local file_pid
        file_pid=$(basename "$tmp_file" | awk -F'.' '{print $(NF-2)}')
        if [[ "$file_pid" =~ ^[0-9]+$ ]] && ! kill -0 "$file_pid" 2>/dev/null; then  # SUPPRESS:OK -- existence-test: checking if process is alive
            rm -f "$tmp_file" 2>/dev/null || true  # SUPPRESS:OK -- cleanup: file may not exist
        fi
    done < <(find "$temp_dir" -maxdepth 1 -name "*.tmp" -print0 2>/dev/null)  # SUPPRESS:OK -- existence-test: directory may not exist
}
# Run orphan cleanup on startup (silent — matches cleanup_locks behavior)
type cleanup_temp_files &>/dev/null && _cleanup_orphaned_temp_files

# --- Per-colony data directory resolution ---
# Resolves COLONY_DATA_DIR from COLONY_STATE.json colony_name field.
# If colony exists: $DATA_DIR/colonies/{sanitized-name}/
# If pre-init (no COLONY_STATE.json or no colony_name): $DATA_DIR (unchanged)
_resolve_colony_data_dir() {
    # Pre-init: no COLONY_STATE.json means no colony yet
    if [[ ! -f "$DATA_DIR/COLONY_STATE.json" ]]; then
        COLONY_DATA_DIR="$DATA_DIR"
        return 0
    fi

    # Read colony name from state (fast path: direct jq, avoids subprocess)
    local raw_name
    raw_name=$(jq -r '.colony_name // empty' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null) || true

    # No colony_name field yet -- might be a very old state file
    if [[ -z "$raw_name" ]]; then
        COLONY_DATA_DIR="$DATA_DIR"
        return 0
    fi

    # Sanitize for filesystem: lowercase, non-alphanumeric to hyphens, trim
    local sanitized
    sanitized=$(echo "$raw_name" | tr '[:upper:]' '[:lower:]' | tr -cs '[:alnum:]' '-' | sed 's/^-//;s/-$//')

    # Empty after sanitization (pathological input) -- fail loudly per locked decision
    if [[ -z "$sanitized" ]]; then
        echo "ERROR: Colony name '$raw_name' sanitizes to empty string. Fix COLONY_STATE.json colony_name field." >&2
        return 1
    fi

    COLONY_DATA_DIR="$DATA_DIR/colonies/$sanitized"
    mkdir -p "$COLONY_DATA_DIR" || {
        echo "ERROR: Cannot create colony data directory: $COLONY_DATA_DIR" >&2
        return 1
    }

    # Auto-migrate flat files on first access
    # NOTE: Migration moves files from DATA_DIR root to COLONY_DATA_DIR.
    # This is safe because all per-colony file references use COLONY_DATA_DIR.
    _maybe_migrate_colony_data "$COLONY_DATA_DIR"
}

# Auto-migrate flat files from DATA_DIR root to colony subdirectory
# Uses presence-based detection per locked decision (no version field)
_maybe_migrate_colony_data() {
    local colony_dir="$1"

    # Already migrated? Check if colony dir has any colony-specific files
    if [[ -f "$colony_dir/pheromones.json" ]] || [[ -f "$colony_dir/session.json" ]] || [[ -f "$colony_dir/flags.json" ]]; then
        return 0
    fi

    # Check for old-style flat files that need migration
    local needs_migration=false
    for f in pheromones.json learning-observations.json session.json run-state.json flags.json constraints.json; do
        if [[ -f "$DATA_DIR/$f" ]]; then
            needs_migration=true
            break
        fi
    done
    [[ "$needs_migration" == "false" ]] && return 0

    # Migration lock -- prevents concurrent migration attempts
    local migration_lock="$DATA_DIR/.migration.lock"
    if ! (set -o noclobber; echo $$ > "$migration_lock") 2>/dev/null; then
        # Another process is migrating -- wait briefly then check if done
        sleep 1
        if [[ -f "$colony_dir/pheromones.json" ]] || [[ -f "$colony_dir/session.json" ]]; then
            rm -f "$migration_lock" 2>/dev/null || true
            return 0
        fi
        echo "ERROR: Migration in progress by another process. If stuck, remove: $migration_lock" >&2
        return 1
    fi

    # Ensure colony dir exists
    mkdir -p "$colony_dir" || {
        rm -f "$migration_lock"
        echo "ERROR: Auto-migration failed: cannot create $colony_dir. Fix permissions and retry." >&2
        return 1
    }

    # Move per-colony files (with .bak.N companions)
    local per_colony_files=(
        pheromones.json learning-observations.json session.json run-state.json
        constraints.json flags.json activity.log rolling-summary.log
        error-patterns.json signatures.json view-state.json
        watch-progress.txt watch-status.txt spawn-tree.txt
        learnings.json learning-deferred.json
        last-build-claims.json last-build-result.json
        safety-stats.json completion-report.md timing.log
        AUDIT-REPORT.md checkpoint-allowlist.json
        swarm-activity.log
    )

    local moved=0
    for f in "${per_colony_files[@]}"; do
        if [[ -f "$DATA_DIR/$f" ]]; then
            mv "$DATA_DIR/$f" "$colony_dir/$f" || {
                rm -f "$migration_lock"
                echo "ERROR: Migration failed moving $f. Partial migration -- fix manually." >&2
                return 1
            }
            moved=$((moved + 1))
            # Move companion .bak.N files
            for bak in "$DATA_DIR/$f".bak.*; do
                [[ -f "$bak" ]] && mv "$bak" "$colony_dir/" 2>/dev/null || true
            done
        fi
    done

    # Move per-colony directories
    for d in midden spawn-tree-archive swarm-archive; do
        if [[ -d "$DATA_DIR/$d" ]]; then
            mv "$DATA_DIR/$d" "$colony_dir/$d" || {
                rm -f "$migration_lock"
                echo "ERROR: Migration failed moving directory $d. Partial migration -- fix manually." >&2
                return 1
            }
            moved=$((moved + 1))
        fi
    done

    # Move swarm-*.json files (dynamic names)
    for f in "$DATA_DIR"/swarm-*.json; do
        [[ -f "$f" ]] && mv "$f" "$colony_dir/" 2>/dev/null && moved=$((moved + 1)) || true
    done

    # Move activity-phase-*.log files
    for f in "$DATA_DIR"/activity-phase-*.log; do
        [[ -f "$f" ]] && mv "$f" "$colony_dir/" 2>/dev/null && moved=$((moved + 1)) || true
    done

    rm -f "$migration_lock"
    return 0
}

# Resolve COLONY_DATA_DIR at startup (AFTER file-lock.sh is sourced, BEFORE dispatch)
# If resolution fails (permissions/migration error), propagate the error and abort
# Export both variables so child processes (e.g. validate-state all spawning subcommands) inherit them
_resolve_colony_data_dir || {
    echo "FATAL: Colony data directory resolution failed. Cannot continue." >&2
    exit 1
}
export DATA_DIR COLONY_DATA_DIR

# --- Caste emoji helper ---
get_caste_emoji() {
  case "$1" in
    *Queen*|*QUEEN*|*queen*) echo "👑🐜" ;;
    *Builder*|*builder*|*Bolt*|*Hammer*|*Forge*|*Mason*|*Brick*|*Anvil*|*Weld*) echo "🔨🐜" ;;
    *Watcher*|*watcher*|*Vigil*|*Sentinel*|*Guard*|*Keen*|*Sharp*|*Hawk*|*Alert*) echo "👁️🐜" ;;
    *Scout*|*scout*|*Swift*|*Dash*|*Ranger*|*Track*|*Seek*|*Path*|*Roam*|*Quest*) echo "🔍🐜" ;;
    *Colonizer*|*colonizer*|*Pioneer*|*Map*|*Chart*|*Venture*|*Explore*|*Compass*|*Atlas*|*Trek*) echo "🗺️🐜" ;;
    *Surveyor*|*surveyor*|*Chart*|*Plot*|*Survey*|*Measure*|*Assess*|*Gauge*|*Sound*|*Fathom*) echo "📊🐜" ;;
    *Architect*|*architect*|*Blueprint*|*Draft*|*Design*|*Plan*|*Schema*|*Frame*|*Sketch*|*Model*) echo "🏛️🐜" ;;
    *Chaos*|*chaos*|*Probe*|*Stress*|*Shake*|*Twist*|*Snap*|*Breach*|*Surge*|*Jolt*) echo "🎲🐜" ;;
    *Archaeologist*|*archaeologist*|*Relic*|*Fossil*|*Dig*|*Shard*|*Epoch*|*Strata*|*Lore*|*Glyph*) echo "🏺🐜" ;;
    *Oracle*|*oracle*|*Sage*|*Seer*|*Vision*|*Augur*|*Mystic*|*Sibyl*|*Delph*|*Pythia*) echo "🔮🐜" ;;
    *Route*|*route*) echo "📋🐜" ;;
    *Ambassador*|*ambassador*|*Bridge*|*Connect*|*Link*|*Diplomat*|*Network*|*Protocol*) echo "🔌🐜" ;;
    *Auditor*|*auditor*|*Review*|*Inspect*|*Examine*|*Scrutin*|*Critical*|*Verify*) echo "👥🐜" ;;
    *Chronicler*|*chronicler*|*Document*|*Record*|*Write*|*Chronicle*|*Archive*|*Scribe*) echo "📝🐜" ;;
    *Gatekeeper*|*gatekeeper*|*Guard*|*Protect*|*Secure*|*Shield*|*Depend*|*Supply*) echo "📦🐜" ;;
    *Guardian*|*guardian*|*Defend*|*Patrol*|*Secure*|*Vigil*|*Watch*|*Safety*|*Security*) echo "🛡️🐜" ;;
    *Includer*|*includer*|*Access*|*Inclusive*|*A11y*|*WCAG*|*Barrier*|*Universal*) echo "♿🐜" ;;
    *Keeper*|*keeper*|*Archive*|*Store*|*Curate*|*Preserve*|*Knowledge*|*Wisdom*|*Pattern*) echo "📚🐜" ;;
    *Measurer*|*measurer*|*Metric*|*Benchmark*|*Profile*|*Optimize*|*Performance*|*Speed*) echo "⚡🐜" ;;
    *Probe*|*probe*|*Test*|*Excavat*|*Uncover*|*Edge*|*Case*|*Mutant*) echo "🧪🐜" ;;
    *Tracker*|*tracker*|*Debug*|*Trace*|*Follow*|*Bug*|*Hunt*|*Root*) echo "🐛🐜" ;;
    *Weaver*|*weaver*|*Refactor*|*Restruct*|*Transform*|*Clean*|*Pattern*|*Weave*) echo "🔄🐜" ;;
    *Dreamer*|*dreamer*|*Dream*|*Muse*|*Imagine*|*Wonder*|*Ponder*|*Reverie*) echo "💭🐜" ;;
    *) echo "🐜" ;;
  esac
}

# --- Progress bar helper ---
# Usage: generate-progress-bar <current> <total> [width]
# Returns: "[████████░░░░░░░░] 8/20" format string
generate-progress-bar() {
  local current="${1:-0}"
  local total="${2:-1}"
  local width="${3:-20}"

  # Prevent division by zero
  [[ "$total" -lt 1 ]] && total=1
  [[ "$current" -lt 0 ]] && current=0
  [[ "$current" -gt "$total" ]] && current="$total"

  # Calculate filled/empty segments
  local filled=$(( (current * width) / total ))
  local empty=$(( width - filled ))

  # Build bar with Unicode block characters
  local bar=""
  for ((i=0; i<filled; i++)); do bar+="█"; done
  for ((i=0; i<empty; i++)); do bar+="░"; done

  echo "[$bar] $current/$total"
}

# --- Standard banner helper ---
# Usage: print-standard-banner <title>
# Outputs a standardized banner with heavy horizontal lines (U+2501)
print-standard-banner() {
  local title="$1"

  # Convert title to spaced uppercase
  local spaced_title
  spaced_title=$(echo "$title" | tr '[:lower:]' '[:upper:]' | sed 's/./& /g' | sed 's/ $//')

  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "   $spaced_title"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# --- Next Up block helper ---
# Usage: print-next-up <state> [current_phase] [total_phases]
# Outputs a Next Up block with state-based suggestions
print-next-up() {
  local state="${1:-IDLE}"
  local current_phase="${2:-0}"
  local total_phases="${3:-0}"
  local next_phase=$((current_phase + 1))

  echo "──────────────────────────────────────────────────"
  echo "🐜 Next Up"
  echo "──────────────────────────────────────────────────"

  case "$state" in
    IDLE)
      echo "   /ant:init               🌱 Start a new colony"
      echo "   /ant:status             📊 Check current state"
      ;;
    READY)
      echo "   /ant:build $next_phase            🔨 Build phase $next_phase"
      echo "   /ant:phase $next_phase            📋 Review phase details"
      echo "   /ant:insert-phase       ➕ Insert a corrective phase"
      echo "   /ant:focus              🎯 Guide colony attention"
      ;;
    EXECUTING)
      echo "   /ant:continue           ➡️  Continue current build"
      echo "   /ant:insert-phase       ➕ Insert a corrective phase"
      echo "   /ant:status             📊 Check build progress"
      ;;
    PLANNING)
      echo "   /ant:plan               📝 Create execution plan"
      echo "   /ant:status             📊 Check current state"
      ;;
    *)
      echo "   /ant:status             📊 Check colony state"
      ;;
  esac
}

# ============================================
# CONTEXT UPDATE HELPER FUNCTION
# (Defined outside case block to fix SC2168: local outside function)
# ============================================
_cmd_context_update() {
  local ctx_action="${1:-}"
  local ctx_file="${AETHER_ROOT:-.}/.aether/CONTEXT.md"
  local ctx_tmp="${ctx_file}.tmp"
  local ctx_ts
  ctx_ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

  # Check for empty action first - show usage message
  if [[ -z "$ctx_action" ]]; then
    json_err "$E_VALIDATION_FAILED" "No action specified. Suggestion: Use one of: init, update-phase, activity, constraint, decision, safe-to-clear, build-start, worker-spawn, worker-complete, build-progress, build-complete"
  fi

  # Acquire lock for context-update operations (LOCK-04: prevent concurrent corruption)
  local _ctx_lock_held=false
  if type acquire_lock &>/dev/null && type feature_enabled &>/dev/null && feature_enabled "file_locking"; then
    acquire_lock "$ctx_file" || json_err "$E_LOCK_FAILED" "Failed to acquire CONTEXT.md lock for context-update"
    _ctx_lock_held=true
    trap 'release_lock 2>/dev/null || true' EXIT  # SUPPRESS:OK -- cleanup: lock may not be held
  fi

  ensure_context_dir() {
    local dir
    dir=$(dirname "$ctx_file")
    [[ -d "$dir" ]] || mkdir -p "$dir"
  }

  read_colony_state() {
    local state_file="${AETHER_ROOT:-.}/.aether/data/COLONY_STATE.json"
    if [[ -f "$state_file" ]]; then
      current_phase=$(jq -r '.current_phase // "unknown"' "$state_file" 2>/dev/null)  # SUPPRESS:OK -- read-default: file may not exist yet
      milestone=$(jq -r '.milestone // "unknown"' "$state_file" 2>/dev/null)  # SUPPRESS:OK -- read-default: file may not exist yet
      goal=$(jq -r '.goal // ""' "$state_file" 2>/dev/null)  # SUPPRESS:OK -- read-default: file may not exist yet
    else
      current_phase="unknown"
      milestone="unknown"
      goal=""
    fi
  }

  case "$ctx_action" in
    init)
      local init_goal="${2:-}"
      ensure_context_dir
      read_colony_state

      cat > "$ctx_file" << EOF
# Aether Colony — Current Context

> **This document is the colony's memory. If context collapses, read this file first.**

---

## 🚦 System Status

| Field | Value |
|-------|-------|
| **Last Updated** | $ctx_ts |
| **Current Phase** | 1 |
| **Phase Name** | initialization |
| **Milestone** | First Mound |
| **Colony Status** | initializing |
| **Safe to Clear?** | ⚠️ NO — Colony just initialized |

---

## 🎯 Current Goal

$init_goal

---

## 📍 What's In Progress

Colony initialization in progress...

---

## ⚠️ Active Constraints (REDIRECT Signals)

| Constraint | Source | Date Set |
|------------|--------|----------|
| In the Aether repo, \`.aether/\` IS the source of truth — published directly via npm (private dirs excluded by .npmignore) | CLAUDE.md | Permanent |
| Never push without explicit user approval | CLAUDE.md Safety | Permanent |

---

## 💭 Active Pheromones (FOCUS Signals)

*None active*

---

## 📝 Recent Decisions

| Date | Decision | Rationale | Made By |
|------|----------|-----------|---------|

---

## 📊 Recent Activity (Last 10 Actions)

| Timestamp | Command | Result | Files Changed |
|-----------|---------|--------|---------------|
| $ctx_ts | init | Colony initialized | — |

---

## 🔄 Next Steps

1. Run \`/ant:plan\` to generate phases for the goal
2. Run \`/ant:build 1\` to start building

---

## 🆘 If Context Collapses

**READ THIS SECTION FIRST**

### Immediate Recovery

1. **Read this file** — You're looking at it. Good.
2. **Check git status** — \`git status\` and \`git log --oneline -5\`
3. **Verify COLONY_STATE.json** — \`cat .aether/data/COLONY_STATE.json | jq .current_phase\`
4. **Resume work** — Continue from "Next Steps" above

### What We Were Doing

Colony was just initialized with goal: $init_goal

### Is It Safe to Continue?

- ✅ Colony is initialized
- ⚠️ No work completed yet
- ✅ All state in COLONY_STATE.json

**You can proceed safely.**

---

## 🐜 Colony Health

\`\`\`
Milestone:    First Mound   ░░░░░░░░░░ 0%
Phase:        1             ░░░░░░░░░░ initializing
Context:      Active        ░░░░░░░░░░ 0%
Git Commits:  0
\`\`\`

---

*This document updates automatically with every ant command. If you see old timestamps, run \`/ant:status\` to refresh.*

**Colony Memory Active** 🧠🐜
EOF
      json_ok "$(jq -nc --arg file "$ctx_file" '{updated:true,action:"init",file:$file}')"
      ;;

    update-phase)
      local new_phase="${2:-}"
      local new_phase_name="${3:-}"
      local safe_clear="${4:-NO}"
      local safe_reason="${5:-Phase in progress}"

      [[ -f "$ctx_file" ]] || { json_err "$E_FILE_NOT_FOUND" "Couldn't find CONTEXT.md. Try: run context-update init first."; }

      sed -i.bak "s/| \*\*Last Updated\*\* | .*/| **Last Updated** | $ctx_ts |/" "$ctx_file" && rm -f "$ctx_file.bak"
      sed -i.bak "s/| \*\*Current Phase\*\* | .*/| **Current Phase** | $new_phase |/" "$ctx_file" && rm -f "$ctx_file.bak"
      sed -i.bak "s/| \*\*Phase Name\*\* | .*/| **Phase Name** | $new_phase_name |/" "$ctx_file" && rm -f "$ctx_file.bak"
      sed -i.bak "s/| \*\*Safe to Clear?\*\* | .*/| **Safe to Clear?** | $safe_clear — $safe_reason |/" "$ctx_file" && rm -f "$ctx_file.bak"

      json_ok "$(jq -nc --argjson phase "$new_phase" '{updated:true,action:"update-phase",phase:$phase}')"
      ;;

    activity)
      local cmd="${2:-}"
      local result="${3:-}"
      local files_changed="${4:-—}"

      [[ -f "$ctx_file" ]] || { json_err "$E_FILE_NOT_FOUND" "Couldn't find CONTEXT.md. Try: run context-update init first."; }

      sed -i.bak "s/| \*\*Last Updated\*\* | .*/| **Last Updated** | $ctx_ts |/" "$ctx_file" && rm -f "$ctx_file.bak"

      local activity_line="| $ctx_ts | $cmd | $result | $files_changed |"

      awk -v line="$activity_line" '
        /\| Timestamp \| Command \| Result \| Files Changed \|/ {
          print
          getline
          print
          print line
          next
        }
        /^## 🆘 If Context Collapses/ { exit }
        { print }
      ' "$ctx_file" > "$ctx_tmp"

      mv "$ctx_tmp" "$ctx_file"
      json_ok "$(jq -nc --arg cmd "$cmd" '{updated:true,action:"activity",command:$cmd}')"
      ;;

    safe-to-clear)
      local safe="${2:-NO}"
      local reason="${3:-Unknown state}"

      [[ -f "$ctx_file" ]] || { json_err "$E_FILE_NOT_FOUND" "Couldn't find CONTEXT.md. Try: run context-update init first."; }

      sed -i.bak "s/| \*\*Last Updated\*\* | .*/| **Last Updated** | $ctx_ts |/" "$ctx_file" && rm -f "$ctx_file.bak"
      sed -i.bak "s/| \*\*Safe to Clear?\*\* | .*/| **Safe to Clear?** | $safe — $reason |/" "$ctx_file" && rm -f "$ctx_file.bak"

      json_ok "$(jq -nc --arg safe "$safe" '{updated:true,action:"safe-to-clear",safe:$safe}')"
      ;;

    constraint)
      local c_type="${2:-}"
      local c_message="${3:-}"
      local c_source="${4:-User}"

      [[ -f "$ctx_file" ]] || { json_err "$E_FILE_NOT_FOUND" "Couldn't find CONTEXT.md. Try: run context-update init first."; }

      sed -i.bak "s/| \*\*Last Updated\*\* | .*/| **Last Updated** | $ctx_ts |/" "$ctx_file" && rm -f "$ctx_file.bak"

      if [[ "$c_type" == "redirect" ]]; then
        sed -i.bak "/^## ⚠️ Active Constraints/,/^## /{ /^| Constraint |/a\\
| $c_message | $c_source | $ctx_ts |
}" "$ctx_file" && rm -f "$ctx_file.bak"
      elif [[ "$c_type" == "focus" ]]; then
        sed -i.bak "/^## 💭 Active Pheromones/,/^## /{ /^| Signal |/a\\
| FOCUS | $c_message | normal |
}" "$ctx_file" && rm -f "$ctx_file.bak"
      elif [[ "$c_type" == "feedback" ]]; then
        sed -i.bak "/^## 💭 Active Pheromones/,/^## /{ /^| Signal |/a\\
| FEEDBACK | $c_message | low |
}" "$ctx_file" && rm -f "$ctx_file.bak"
      fi

      json_ok "$(jq -nc --arg type "$c_type" '{updated:true,action:"constraint",type:$type}')"
      ;;

    decision)
      local decision="${2:-}"
      local rationale="${3:-}"
      local made_by="${4:-Colony}"

      [[ -f "$ctx_file" ]] || { json_err "$E_FILE_NOT_FOUND" "Couldn't find CONTEXT.md. Try: run context-update init first."; }

      sed -i.bak "s/| \*\*Last Updated\*\* | .*/| **Last Updated** | $ctx_ts |/" "$ctx_file" && rm -f "$ctx_file.bak"

      local decision_line="| $(echo $ctx_ts | cut -dT -f1) | $decision | $rationale | $made_by |"

      awk -v line="$decision_line" '
        /^## 📝 Recent Decisions/ { in_section=1 }
        in_section && /^\| [0-9]{4}-[0-9]{2}-[0-9]{2} / { last_decision=NR }
        in_section && /^## 📊 Recent Activity/ { in_section=0 }
        { lines[NR] = $0 }
        END {
          for (i=1; i<=NR; i++) {
            if (i == last_decision) {
              print lines[i]
              print line
            } else {
              print lines[i]
            }
          }
        }
      ' "$ctx_file" > "$ctx_tmp"

      mv "$ctx_tmp" "$ctx_file"

      # Auto-emit FEEDBACK pheromone for the decision so builders see it
      bash "$0" pheromone-write FEEDBACK "[decision] $decision" \
        --strength 0.6 \
        --source "auto:decision" \
        --reason "Auto-emitted from architectural decision" \
        --ttl "30d" 2>/dev/null \
        || _aether_log_error "Could not emit feedback signal for decision"  # SUPPRESS:OK -- read-default: returns fallback on failure

      json_ok "$(jq -nc '{updated:true,action:"decision"}')"
      ;;

    build-start)
      local phase_id="${2:-}"
      local worker_count="${3:-0}"
      local tasks_count="${4:-0}"

      [[ -f "$ctx_file" ]] || { json_err "$E_FILE_NOT_FOUND" "Couldn't find CONTEXT.md. Try: run context-update init first."; }

      sed -i.bak "s/| \*\*Last Updated\*\* | .*/| **Last Updated** | $ctx_ts |/" "$ctx_file" && rm -f "$ctx_file.bak"
      sed -i.bak "s/## 📍 What's In Progress/## 📍 What's In Progress\n\n**Phase $phase_id Build IN PROGRESS**\n- Workers: $worker_count | Tasks: $tasks_count\n- Started: $ctx_ts/" "$ctx_file" && rm -f "$ctx_file.bak"
      sed -i.bak "s/| \*\*Safe to Clear?\*\* | .*/| **Safe to Clear?** | ⚠️ NO — Build in progress |/" "$ctx_file" && rm -f "$ctx_file.bak"

      json_ok "$(jq -nc --argjson workers "$worker_count" '{updated:true,action:"build-start",workers:$workers}')"
      ;;

    worker-spawn)
      local ant_name="${2:-}"
      local caste="${3:-}"
      local task="${4:-}"

      [[ -f "$ctx_file" ]] || { json_err "$E_FILE_NOT_FOUND" "Couldn't find CONTEXT.md. Try: run context-update init first."; }

      awk -v ant="$ant_name" -v caste="$caste" -v task="$task" -v ts="$ctx_ts" '
        /^## 📍 What'\''s In Progress/ { in_progress=1 }
        in_progress && /^## / && $0 !~ /What'\''s In Progress/ { in_progress=0 }
        in_progress && /Workers:/ {
          print
          print "  - " ts ": Spawned " ant " (" caste ") for: " task
          next
        }
        { print }
      ' "$ctx_file" > "$ctx_tmp" && mv "$ctx_tmp" "$ctx_file"

      json_ok "$(jq -nc --arg ant "$ant_name" '{updated:true,action:"worker-spawn",ant:$ant}')"
      ;;

    worker-complete)
      local ant_name="${2:-}"
      local status="${3:-completed}"

      [[ -f "$ctx_file" ]] || { json_err "$E_FILE_NOT_FOUND" "Couldn't find CONTEXT.md. Try: run context-update init first."; }

      sed -i.bak "s/- .*$ant_name .*$/- $ant_name: $status (updated $ctx_ts)/" "$ctx_file" && rm -f "$ctx_file.bak"

      json_ok "$(jq -nc --arg ant "$ant_name" '{updated:true,action:"worker-complete",ant:$ant}')"
      ;;

    build-progress)
      local completed="${2:-0}"
      local total="${3:-1}"
      local percentage=$(( completed * 100 / total ))

      [[ -f "$ctx_file" ]] || { json_err "$E_FILE_NOT_FOUND" "Couldn't find CONTEXT.md. Try: run context-update init first."; }

      sed -i.bak "s/Build IN PROGRESS/Build IN PROGRESS ($percentage% complete)/" "$ctx_file" && rm -f "$ctx_file.bak"

      json_ok "$(jq -nc --argjson percent "$percentage" '{updated:true,action:"build-progress",percent:$percent}')"
      ;;

    build-complete)
      local status="${2:-completed}"
      local result="${3:-success}"

      [[ -f "$ctx_file" ]] || { json_err "$E_FILE_NOT_FOUND" "Couldn't find CONTEXT.md. Try: run context-update init first."; }

      sed -i.bak "s/| \*\*Last Updated\*\* | .*/| **Last Updated** | $ctx_ts |/" "$ctx_file" && rm -f "$ctx_file.bak"

      awk -v status="$status" -v result="$result" '
        /^## 📍 What'\''s In Progress/ { in_progress=1 }
        in_progress && /^## / && $0 !~ /What'\''s In Progress/ { in_progress=0 }
        in_progress && /Build IN PROGRESS/ {
          print "## 📍 What'\''s In Progress"
          print ""
          print "**Build " status "** — " result
          next
        }
        in_progress { next }
        { print }
      ' "$ctx_file" > "$ctx_tmp" && mv "$ctx_tmp" "$ctx_file"

      sed -i.bak "s/| \*\*Safe to Clear?\*\* | .*/| **Safe to Clear?** | ✅ YES — Build $status |/" "$ctx_file" && rm -f "$ctx_file.bak"

      json_ok "$(jq -nc --arg status "$status" '{updated:true,action:"build-complete",status:$status}')"
      ;;

    *)
      json_err "$E_VALIDATION_FAILED" "Unknown context action: '$ctx_action'. Suggestion: Use one of: init, update-phase, activity, constraint, decision, safe-to-clear, build-start, worker-spawn, worker-complete, build-progress, build-complete"
      ;;
  esac

  # Release lock on success (LOCK-04)
  # NOTE: Do NOT clear the EXIT trap here. This function RETURNS (it does not
  # call exit), so clearing the trap would remove the safety net without benefit.
  # The EXIT trap remains active as a true safety net for unexpected exit calls
  # elsewhere in the process. The _ctx_lock_held variable is the primary gate
  # for this function's own cleanup.
  if [[ "$_ctx_lock_held" == "true" ]]; then
    release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
    _ctx_lock_held=false
  fi
}

# --- Changelog helpers ---
# Append an entry to CHANGELOG.md with date-phase hierarchy format
# Usage: changelog-append <date> <phase> <plan> <files> <decisions> <worked> <requirements>
# Parameters:
#   date: ISO date string (e.g., "2026-02-21")
#   phase: Phase identifier (e.g., "36-memory-capture")
#   plan: Plan number (e.g., "01")
#   files: Comma-separated list of files changed
#   decisions: Semicolon-separated list of decisions
#   worked: What worked/didn't (semicolon-separated)
#   requirements: Comma-separated list of requirement IDs
changelog-append() {
  local date_str="${1:-$(date +%Y-%m-%d)}"
  local phase="${2:-}"
  local plan="${3:-}"
  local files="${4:-}"
  local decisions="${5:-}"
  local worked="${6:-}"
  local requirements="${7:-}"

  local changelog_file="${AETHER_ROOT}/CHANGELOG.md"
  local temp_file
  temp_file=$(mktemp)

  # Create CHANGELOG.md if it doesn't exist
  if [[ ! -f "$changelog_file" ]]; then
    cat > "$changelog_file" << 'EOF'
# Changelog

All notable changes to this project will be documented in this file.

EOF
  fi

  # Check if existing CHANGELOG.md uses Keep a Changelog format
  # and add separator if needed (only once)
  local has_separator=false
  if grep -q "Colony Work Log" "$changelog_file" 2>/dev/null; then  # SUPPRESS:OK -- existence-test: file may not exist
    has_separator=true
  fi

  # Detect Keep a Changelog format by looking for version headers
  local is_keep_a_changelog=false
  if grep -qE '^## \[.*\]' "$changelog_file" 2>/dev/null; then  # SUPPRESS:OK -- existence-test: file may not exist
    is_keep_a_changelog=true
  fi

  # If Keep a Changelog format and no separator yet, add it
  if [[ "$is_keep_a_changelog" == "true" && "$has_separator" == "false" ]]; then
    cat >> "$changelog_file" << 'EOF'

---

## Colony Work Log

The following entries are automatically generated by the colony during work phases.

EOF
  fi

  # Read current content
  cat "$changelog_file" > "$temp_file"

  # Check if date section exists
  if ! grep -q "^## ${date_str}$" "$changelog_file" 2>/dev/null; then  # SUPPRESS:OK -- existence-test: file may not exist
    # Add new date section
    echo "" >> "$temp_file"
    echo "## ${date_str}" >> "$temp_file"
  fi

  # Build the phase entry
  local phase_num
  phase_num=$(echo "$phase" | grep -oE '^[0-9]+' || echo "0")

  # Append phase subsection
  {
    echo ""
    echo "### Phase ${phase_num} — Plan ${plan}"
    echo ""

    # Files changed
    if [[ -n "$files" ]]; then
      echo -n "- **Files:** "
      # Format files with modification markers if present
      local file_list=""
      IFS=',' read -ra file_arr <<< "$files"
      for f in "${file_arr[@]}"; do
        f=$(echo "$f" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        if [[ -n "$file_list" ]]; then
          file_list+=", "
        fi
        file_list+="\`$f\`"
      done
      echo "$file_list"
    fi

    # Decisions
    if [[ -n "$decisions" ]]; then
      echo -n "- **Decisions:** "
      local first=true
      IFS=';' read -ra dec_arr <<< "$decisions"
      for d in "${dec_arr[@]}"; do
        d=$(echo "$d" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        if [[ -n "$d" ]]; then
          if [[ "$first" == "true" ]]; then
            first=false
          else
            echo -n "; "
          fi
          echo -n "$d"
        fi
      done
      echo ""
    fi

    # What worked
    if [[ -n "$worked" ]]; then
      echo -n "- **What Worked:** "
      local first=true
      IFS=';' read -ra work_arr <<< "$worked"
      for w in "${work_arr[@]}"; do
        w=$(echo "$w" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        if [[ -n "$w" ]]; then
          if [[ "$first" == "true" ]]; then
            first=false
          else
            echo -n "; "
          fi
          echo -n "$w"
        fi
      done
      echo ""
    fi

    # What didn't (placeholder - would come from midden)
    # This is intentionally left for future enhancement

    # Requirements
    if [[ -n "$requirements" ]]; then
      echo -n "- **Requirements:** "
      local req_list=""
      IFS=',' read -ra req_arr <<< "$requirements"
      for r in "${req_arr[@]}"; do
        r=$(echo "$r" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        if [[ -n "$req_list" ]]; then
          req_list+=", "
        fi
        req_list+="$r"
      done
      echo -n "$req_list addressed"
      echo ""
    fi
  } >> "$temp_file"

  # Atomically replace the file
  mv "$temp_file" "$changelog_file"

  json_ok "$(jq -nc --arg date "$date_str" --arg phase "$phase" --arg plan "$plan" '{appended:true,date:$date,phase:$phase,plan:$plan}')"
}

# Collect plan data for changelog entry
# Usage: changelog-collect-plan-data <phase> <plan>
# Returns: JSON with all collected data
changelog-collect-plan-data() {
  local phase="${1:-}"
  local plan="${2:-}"

  if [[ -z "$phase" || -z "$plan" ]]; then
    json_err "$E_VALIDATION_FAILED" "changelog-collect-plan-data requires phase and plan arguments"
    return 1
  fi

  # Extract phase number from phase identifier (e.g., "36-memory-capture" -> "36")
  local phase_num
  phase_num=$(echo "$phase" | grep -oE '^[0-9]+' || echo "")

  # Try both naming conventions for compatibility
  local plan_file="${AETHER_ROOT}/.planning/phases/${phase}/${phase_num}-${plan}-PLAN.md"
  if [[ ! -f "$plan_file" ]]; then
    # Fallback to full phase name format
    plan_file="${AETHER_ROOT}/.planning/phases/${phase}/${phase}-${plan}-PLAN.md"
  fi
  local state_file="${DATA_DIR}/COLONY_STATE.json"

  # Initialize JSON structure
  local files="[]"
  local requirements="[]"
  local decisions="[]"
  local worked="[]"
  local didnt_work="[]"

  # Read plan file if it exists
  if [[ -f "$plan_file" ]]; then
    # Extract files_modified from frontmatter
    local files_yaml
    # SUPPRESS:OK -- read-default: section may not exist in file
    files_yaml=$(grep -A 20 "^files_modified:" "$plan_file" 2>/dev/null | grep "^  - " | sed 's/^  - //' | tr '\n' ',' | sed 's/,$//')
    if [[ -n "$files_yaml" ]]; then
      # Convert to JSON array
      files="["
      local first=true
      IFS=',' read -ra file_arr <<< "$files_yaml"
      for f in "${file_arr[@]}"; do
        f=$(echo "$f" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        if [[ "$first" == "true" ]]; then
          first=false
        else
          files+=","
        fi
        files+="\"$(echo "$f" | sed 's/"/\\"/g')\""
      done
      files+="]"
    fi

    # Extract requirements from frontmatter
    local req_yaml
    # SUPPRESS:OK -- read-default: section may not exist in file
    req_yaml=$(grep -A 10 "^requirements:" "$plan_file" 2>/dev/null | grep "^  - " | sed 's/^  - //' | tr '\n' ',' | sed 's/,$//')
    if [[ -n "$req_yaml" ]]; then
      requirements="["
      local first=true
      IFS=',' read -ra req_arr <<< "$req_yaml"
      for r in "${req_arr[@]}"; do
        r=$(echo "$r" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        if [[ "$first" == "true" ]]; then
          first=false
        else
          requirements+=","
        fi
        requirements+="\"$(echo "$r" | sed 's/"/\\"/g')\""
      done
      requirements+="]"
    fi
  fi

  # Read decisions from COLONY_STATE.json (last 5)
  if [[ -f "$state_file" ]] && command -v jq &>/dev/null; then
    local recent_decisions
    recent_decisions=$(jq -r '.memory.decisions[-5:][]? // empty' "$state_file" 2>/dev/null)  # SUPPRESS:OK -- read-default: file may not exist yet
    if [[ -n "$recent_decisions" ]]; then
      decisions="["
      local first=true
      while IFS= read -r d || [[ -n "$d" ]]; do
        if [[ -n "$d" && "$d" != "null" ]]; then
          if [[ "$first" == "true" ]]; then
            first=false
          else
            decisions+=","
          fi
          decisions+="\"$(echo "$d" | sed 's/"/\\"/g')\""
        fi
      done <<< "$recent_decisions"
      decisions+="]"
    fi
  fi

  # Read from midden for worked/didn't work
  local midden_dir="${AETHER_ROOT}/.aether/midden"
  if [[ -d "$midden_dir" ]]; then
    # Check approach-changes.md for what worked
    if [[ -f "$midden_dir/approach-changes.md" ]]; then
      local approach_entries
      approach_entries=$(grep "^- " "$midden_dir/approach-changes.md" 2>/dev/null | tail -3) || true  # SUPPRESS:OK -- read-default: file may not exist
      if [[ -n "$approach_entries" ]]; then
        worked="["
        local first=true
        while IFS= read -r entry || [[ -n "$entry" ]]; do
          entry=$(echo "$entry" | sed 's/^- //' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
          if [[ -n "$entry" ]]; then
            if [[ "$first" == "true" ]]; then
              first=false
            else
              worked+=","
            fi
            worked+="\"$(echo "$entry" | sed 's/"/\\"/g')\""
          fi
        done <<< "$approach_entries"
        worked+="]"
      fi
    fi

    # Check build-failures.md for what didn't work
    if [[ -f "$midden_dir/build-failures.md" ]]; then
      local failure_entries
      failure_entries=$(grep "^- " "$midden_dir/build-failures.md" 2>/dev/null | tail -3) || true  # SUPPRESS:OK -- read-default: file may not exist
      if [[ -n "$failure_entries" ]]; then
        didnt_work="["
        local first=true
        while IFS= read -r entry || [[ -n "$entry" ]]; do
          entry=$(echo "$entry" | sed 's/^- //' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
          if [[ -n "$entry" ]]; then
            if [[ "$first" == "true" ]]; then
              first=false
            else
              didnt_work+=","
            fi
            didnt_work+="\"$(echo "$entry" | sed 's/"/\\"/g')\""
          fi
        done <<< "$failure_entries"
        didnt_work+="]"
      fi
    fi
  fi

  # Output JSON
  cat << EOF
{
  "phase": "$phase",
  "plan": "$plan",
  "files": $files,
  "requirements": $requirements,
  "decisions": $decisions,
  "worked": $worked,
  "didnt_work": $didnt_work
}
EOF
}

# --- Wisdom threshold helpers ---
# Single source of truth for promotion thresholds.
# `propose` controls queueing/visibility for queen review.
# `auto` controls automatic promotion to QUEEN.md.
get_wisdom_threshold() {
  local wisdom_type="${1:-}"
  local mode="${2:-propose}"

  case "$wisdom_type:$mode" in
    # V2 types (no threshold -- always write)
    build_learning:propose) echo 0 ;;
    build_learning:auto) echo 0 ;;
    instinct:propose) echo 0 ;;
    instinct:auto) echo 0 ;;
    # V1 types (backward compat)
    philosophy:propose) echo 1 ;;
    philosophy:auto) echo 3 ;;
    pattern:propose) echo 1 ;;
    pattern:auto) echo 1 ;;
    redirect:propose) echo 1 ;;
    redirect:auto) echo 2 ;;
    stack:propose) echo 1 ;;
    stack:auto) echo 2 ;;
    decree:propose) echo 0 ;;
    decree:auto) echo 0 ;;
    failure:propose) echo 1 ;;
    failure:auto) echo 2 ;;
    *:propose) echo 1 ;;
    *:auto) echo 2 ;;
    *) echo 1 ;;
  esac
}

get_wisdom_thresholds_json() {
  cat <<'EOF'
{
  "build_learning": {"propose": 0, "auto": 0},
  "instinct": {"propose": 0, "auto": 0},
  "philosophy": {"propose": 1, "auto": 3},
  "pattern": {"propose": 1, "auto": 1},
  "redirect": {"propose": 1, "auto": 2},
  "stack": {"propose": 1, "auto": 2},
  "decree": {"propose": 0, "auto": 0},
  "failure": {"propose": 1, "auto": 2}
}
EOF
}

# --- Subcommand dispatch ---
cmd="${1:-help}"
shift 2>/dev/null || true  # SUPPRESS:OK -- cleanup: shift is safe to fail

case "$cmd" in
  help)
    # Build help JSON with sections for discoverability.
    # The flat 'commands' array is kept for backward compatibility
    # (callers use: jq '.commands[]')
    cat <<'HELP_EOF'
{
  "ok": true,
  "commands": ["help","version","validate-state","validate-oracle-state","load-state","unload-state","error-add","error-pattern-check","error-summary","activity-log","activity-log-init","activity-log-read","learning-promote","learning-inject","learning-observe","learning-check-promotion","learning-promote-auto","memory-capture","queen-thresholds","context-capsule","rolling-summary","generate-ant-name","spawn-log","spawn-complete","spawn-can-spawn","spawn-get-depth","spawn-tree-load","spawn-tree-active","spawn-tree-depth","spawn-efficiency","validate-worker-response","update-progress","check-antipattern","error-flag-pattern","signature-scan","signature-match","flag-add","flag-check-blockers","flag-resolve","flag-acknowledge","flag-list","flag-auto-resolve","autofix-checkpoint","autofix-rollback","spawn-can-spawn-swarm","swarm-findings-init","swarm-findings-add","swarm-findings-read","swarm-solution-set","swarm-cleanup","swarm-activity-log","swarm-display-init","swarm-display-update","swarm-display-get","swarm-display-text","swarm-timing-start","swarm-timing-get","swarm-timing-eta","view-state-init","view-state-get","view-state-set","view-state-toggle","view-state-expand","view-state-collapse","grave-add","grave-check","phase-insert","generate-commit-message","version-check","registry-add","registry-list","bootstrap-system","chamber-create","chamber-verify","chamber-list","milestone-detect","queen-init","queen-read","queen-promote","incident-rule-add","survey-load","survey-verify","pheromone-export","pheromone-write","pheromone-count","pheromone-read","instinct-read","instinct-create","instinct-apply","pheromone-prime","colony-prime","pheromone-expire","eternal-init","eternal-store","pheromone-export-xml","pheromone-import-xml","pheromone-validate-xml","wisdom-export-xml","wisdom-import-xml","registry-export-xml","registry-import-xml","memory-metrics","midden-recent-failures","midden-review","midden-acknowledge","midden-search","midden-tag","entropy-score","force-unlock","changelog-append","changelog-collect-plan-data","suggest-approve","suggest-quick-dismiss","data-clean","autopilot-init","autopilot-update","autopilot-status","autopilot-stop","autopilot-check-replan","autopilot-set-headless","autopilot-headless-check","pending-decision-add","pending-decision-list","pending-decision-resolve","hive-init","hive-store","hive-read","hive-abstract","hive-promote","init-research","charter-write","colony-name","colony-vital-signs","emoji-audit","trophallaxis-diagnose","trophallaxis-retry","scar-add","scar-list","scar-check","immune-auto-scar","council-deliberate","council-advocate","council-challenger","council-sage","council-history","council-budget-check"],
  "sections": {
    "Core": [
      {"name": "help", "description": "List all available commands with sections"},
      {"name": "version", "description": "Show installed version"}
    ],
    "Colony State": [
      {"name": "validate-state", "description": "Validate COLONY_STATE.json or constraints.json"},
      {"name": "validate-oracle-state", "description": "Validate oracle state files (state.json, plan.json)"},
      {"name": "load-state", "description": "Load and lock COLONY_STATE.json"},
      {"name": "unload-state", "description": "Release COLONY_STATE.json lock"},
      {"name": "phase-insert", "description": "Insert a new phase after current phase and renumber safely"}
    ],
    "Queen Commands": [
      {"name": "queen-init", "description": "Initialize a new colony QUEEN.md from template"},
      {"name": "queen-read", "description": "Read QUEEN.md wisdom as JSON for worker priming"},
      {"name": "queen-promote", "description": "Promote a validated learning to QUEEN.md wisdom"},
      {"name": "queen-thresholds", "description": "Return propose/auto promotion thresholds by wisdom type"},
      {"name": "queen-write-learnings", "description": "Write build learnings directly to QUEEN.md Build Learnings section (no threshold)"},
      {"name": "queen-promote-instinct", "description": "Promote a high-confidence instinct to QUEEN.md Instincts section"},
      {"name": "queen-seed-from-hive", "description": "Seed QUEEN.md Codebase Patterns from cross-colony hive wisdom"},
      {"name": "queen-migrate", "description": "Migrate a QUEEN.md from v1 (emoji headers) to v2 (clean headers) format"},
      {"name": "charter-write", "description": "Write or update colony charter content in QUEEN.md"},
      {"name": "colony-name", "description": "Derive human-readable colony name from repo context"},
      {"name": "colony-depth", "description": "Read or write colony depth (light/standard/deep/full)"},
      {"name": "domain-detect", "description": "Auto-detect repo domain tags from file presence (node, typescript, rust, etc.)"},
      {"name": "incident-rule-add", "description": "Append an incident-derived rule to decree/constraint/gate stores"},
      {"name": "learning-observe", "description": "Record observation of a learning across colonies"},
      {"name": "learning-check-promotion", "description": "Check which learnings meet promotion thresholds"},
      {"name": "learning-promote-auto", "description": "Auto-promote high-confidence learnings based on recurrence policy"},
      {"name": "memory-capture", "description": "Capture learning/failure events with auto-pheromone and auto-promotion"},
      {"name": "context-capsule", "description": "Generate a compact state capsule for prompt injection"},
      {"name": "rolling-summary", "description": "Record/read rolling narrative context (last 15 entries)"}
    ],
    "Spawn Management": [
      {"name": "spawn-log", "description": "Log a spawn event to spawn-tree.txt"},
      {"name": "spawn-complete", "description": "Record spawn completion in spawn-tree.txt"},
      {"name": "spawn-can-spawn", "description": "Check if spawn budget allows another worker"},
      {"name": "spawn-get-depth", "description": "Get spawn depth for an ant name"},
      {"name": "spawn-tree-load", "description": "Load spawn-tree.txt as JSON"},
      {"name": "spawn-tree-active", "description": "List currently active spawns"},
      {"name": "spawn-tree-depth", "description": "Get depth for a named ant"},
      {"name": "spawn-efficiency", "description": "Calculate spawn completion efficiency metrics"},
      {"name": "validate-worker-response", "description": "Validate worker JSON output against caste schema"}
    ],
    "Flag Management": [
      {"name": "flag-add", "description": "Add a flag to flags.json"},
      {"name": "flag-check-blockers", "description": "Check for flags blocking a task"},
      {"name": "flag-resolve", "description": "Mark a flag as resolved"},
      {"name": "flag-acknowledge", "description": "Acknowledge a flag without resolving"},
      {"name": "flag-list", "description": "List all flags"},
      {"name": "flag-auto-resolve", "description": "Auto-resolve flags matching criteria"}
    ],
    "Chamber Management": [
      {"name": "chamber-create", "description": "Entomb a colony into a named chamber"},
      {"name": "chamber-verify", "description": "Verify chamber integrity"},
      {"name": "chamber-list", "description": "List all available chambers"}
    ],
    "Swarm Operations": [
      {"name": "swarm-findings-init", "description": "Initialize swarm findings file"},
      {"name": "swarm-findings-add", "description": "Add a finding to swarm results"},
      {"name": "swarm-findings-read", "description": "Read all swarm findings"},
      {"name": "swarm-solution-set", "description": "Set the chosen swarm solution"},
      {"name": "swarm-cleanup", "description": "Clean up swarm state files"},
      {"name": "swarm-display-init", "description": "Initialize swarm progress display"},
      {"name": "swarm-display-update", "description": "Update swarm display for an ant"},
      {"name": "swarm-timing-start", "description": "Start timing for a swarm operation"},
      {"name": "swarm-timing-get", "description": "Get elapsed time for a swarm"},
      {"name": "swarm-timing-eta", "description": "Estimate remaining time for a swarm"}
    ],
    "Pheromone System": [
      {"name": "pheromone-write", "description": "Write a pheromone signal"},
      {"name": "pheromone-read", "description": "Read pheromone signals"},
      {"name": "pheromone-count", "description": "Count active pheromone signals"},
      {"name": "pheromone-prime", "description": "Prime the pheromone system"},
      {"name": "colony-prime", "description": "Assemble unified worker priming payload"},
      {"name": "pheromone-expire", "description": "Expire old pheromone signals"},
      {"name": "eternal-store", "description": "Store high-value signals in eternal memory"},
      {"name": "pheromone-export", "description": "Export pheromone data to JSON"},
      {"name": "pheromone-export-xml", "description": "Export pheromone data to XML"},
      {"name": "pheromone-import-xml", "description": "Import pheromone data from XML"},
      {"name": "pheromone-validate-xml", "description": "Validate pheromone XML against schema"}
    ],
    "Utilities": [
      {"name": "generate-ant-name", "description": "Generate a unique ant name with caste prefix"},
      {"name": "activity-log", "description": "Append an entry to the activity log"},
      {"name": "activity-log-init", "description": "Initialize the activity log file"},
      {"name": "activity-log-read", "description": "Read recent activity log entries"},
      {"name": "generate-commit-message", "description": "Generate a commit message (types: milestone, pause, fix, contextual, seal)"},
      {"name": "version-check", "description": "Check if Aether version meets requirement"},
      {"name": "registry-add", "description": "Register a repo with Aether (supports --tags, --goal, --active)"},
      {"name": "registry-list", "description": "List all registered repos with metadata"},
      {"name": "bootstrap-system", "description": "Bootstrap minimal system files if missing"},
      {"name": "memory-metrics", "description": "Aggregate memory health across colony stores"},
      {"name": "midden-recent-failures", "description": "Read recent failure signals from midden"},
      {"name": "midden-review", "description": "Review unacknowledged midden entries grouped by category"},
      {"name": "midden-acknowledge", "description": "Acknowledge midden entries by id or category"},
      {"name": "midden-search", "description": "Search midden entries by keyword with optional category/source filters"},
      {"name": "midden-tag", "description": "Add or remove a tag from a midden entry"},
      {"name": "entropy-score", "description": "Compute colony entropy score (0-100)"},
      {"name": "colony-vital-signs", "description": "Compute colony health metrics from existing data (velocity, errors, signals, memory, overall score)"},
      {"name": "force-unlock", "description": "Emergency unlock — remove stale lock files"}
    ],
    "Changelog": [
      {"name": "changelog-append", "description": "Append entry to CHANGELOG.md with date-phase hierarchy"},
      {"name": "changelog-collect-plan-data", "description": "Collect plan data for changelog entry from state files"}
    ],
    "Suggestion System": [
      {"name": "suggest-approve", "description": "Tick-to-approve UI for pheromone suggestions"},
      {"name": "suggest-quick-dismiss", "description": "Dismiss all suggestions without approving"}
    ],
    "Maintenance": [
      {"name": "data-clean", "description": "Scan and remove test/synthetic artifacts from colony data files"}
    ],
    "Autopilot": [
      {"name": "autopilot-init", "description": "Initialize autopilot run state (run-state.json)"},
      {"name": "autopilot-update", "description": "Update autopilot state after phase action"},
      {"name": "autopilot-status", "description": "Return current autopilot state"},
      {"name": "autopilot-stop", "description": "Stop or complete an autopilot run with reason"},
      {"name": "autopilot-check-replan", "description": "Check if replan trigger should fire based on completed phases"},
      {"name": "autopilot-set-headless", "description": "Set headless mode flag in run-state.json (true or false)"},
      {"name": "autopilot-headless-check", "description": "Check if autopilot is running in headless mode"},
      {"name": "pending-decision-add", "description": "Queue a deferred decision for later review"},
      {"name": "pending-decision-list", "description": "List pending decisions, optionally filtered by type or status"},
      {"name": "pending-decision-resolve", "description": "Mark a pending decision as resolved with a resolution note"}
    ],
    "Hive Intelligence": [
      {"name": "hive-init", "description": "Initialize ~/.aether/hive/ directory and wisdom.json schema"},
      {"name": "hive-store", "description": "Store wisdom entry with dedup, merge, and 200-entry cap"},
      {"name": "hive-read", "description": "Read wisdom entries with domain filtering, confidence threshold, and access tracking"},
      {"name": "hive-abstract", "description": "Abstract repo-specific instinct into generalized cross-colony wisdom text"},
      {"name": "hive-promote", "description": "Orchestrate abstract+store pipeline to promote instinct to hive wisdom"}
    ],
    "Skills Engine": [
      {"name": "skill-parse-frontmatter", "description": "Parse YAML-like frontmatter from a SKILL.md file"},
      {"name": "skill-index", "description": "Scan skills dirs for SKILL.md files and build JSON index"},
      {"name": "skill-index-read", "description": "Read cached index, rebuild if stale (mtime check) [DEPRECATED]"},
      {"name": "skill-detect", "description": "Detect domain skills matching the repo (file patterns + packages)"},
      {"name": "skill-match", "description": "Smart-match skills to worker by role, pheromones, task description"},
      {"name": "skill-inject", "description": "Load full SKILL.md content for matched skills within 8K budget"},
      {"name": "skill-list", "description": "List all installed skills with type, domains, detection status"},
      {"name": "skill-manifest-read", "description": "Read .manifest.json for update safety [DEPRECATED]"},
      {"name": "skill-cache-rebuild", "description": "Force rebuild of skills index cache"},
      {"name": "skill-diff", "description": "Compare user skill with shipped Aether version"},
      {"name": "skill-is-user-created", "description": "Check if a skill is user-created (not in manifest) [DEPRECATED]"}
    ],
    "Smart Init": [
      {"name": "init-research", "description": "Scan repo and produce structured research JSON for smart init"}
    ],
    "Emoji Audit": [
      {"name": "emoji-audit", "description": "Audit emoji usage in command files against the canonical colony-visuals reference map"}
    ],
    "Immune System": [
      {"name": "trophallaxis-diagnose", "description": "Diagnose a task failure by searching midden for related failures and suggesting an approach"},
      {"name": "trophallaxis-retry", "description": "Record a retry attempt for a task with diagnosis context injected"},
      {"name": "scar-add", "description": "Record a persistent failure pattern as a scar to warn future workers"},
      {"name": "scar-list", "description": "List scars, optionally filtered by active status or severity"},
      {"name": "scar-check", "description": "Check if a task description matches any active scar patterns"},
      {"name": "immune-auto-scar", "description": "Auto-create a scar if a task has been retried 3 or more times"}
    ],
    "Council": [
      {"name": "council-deliberate", "description": "Start a new deliberation with Advocate/Challenger/Sage model and spawn budget guard"},
      {"name": "council-advocate", "description": "Record an advocate argument (FOR) for a deliberation"},
      {"name": "council-challenger", "description": "Record a challenger argument (AGAINST) for a deliberation"},
      {"name": "council-sage", "description": "Record sage synthesis and recommendation, marks deliberation complete"},
      {"name": "council-history", "description": "List past deliberations with their outcomes"},
      {"name": "council-budget-check", "description": "Check if current spawn budget allows N more spawns"}
    ],
    "Deprecated": [
      {"name": "checkpoint-check", "description": "Check dirty files against allowlist [DEPRECATED]"},
      {"name": "error-pattern-check", "description": "Check for error anti-patterns [DEPRECATED]"},
      {"name": "error-patterns-check", "description": "Scan for error handling anti-patterns [DEPRECATED]"},
      {"name": "error-summary", "description": "Summarize error handling patterns [DEPRECATED]"},
      {"name": "learning-select-proposals", "description": "Interactive learning proposal selector [DEPRECATED]"},
      {"name": "pheromone-export-eternal", "description": "Export pheromones to eternal memory format [DEPRECATED]"},
      {"name": "semantic-context", "description": "Semantic search context retrieval [DEPRECATED]"},
      {"name": "session-clear-context", "description": "Clear session context markers [DEPRECATED]"},
      {"name": "session-is-stale", "description": "Check session staleness [DEPRECATED]"},
      {"name": "session-summary", "description": "Generate session summary [DEPRECATED]"},
      {"name": "skill-index-read", "description": "Read cached skills index [DEPRECATED]"},
      {"name": "skill-is-user-created", "description": "Check if skill is user-created [DEPRECATED]"},
      {"name": "skill-manifest-read", "description": "Read skills manifest [DEPRECATED]"},
      {"name": "suggest-clear", "description": "Clear suggestion state [DEPRECATED]"},
      {"name": "survey-clear", "description": "Clear survey state [DEPRECATED]"},
      {"name": "survey-verify-fresh", "description": "Check survey freshness [DEPRECATED]"},
      {"name": "swarm-display-inline", "description": "Inline swarm display for Claude Code [DEPRECATED]"},
      {"name": "swarm-display-render", "description": "Terminal render wrapper for swarm display [DEPRECATED]"}
    ]
  },
  "description": "Aether Colony Utility Layer — deterministic ops for the ant colony"
}
HELP_EOF
    ;;
  version)
    # Read version from package.json if available, fallback to embedded
    _pkg_json="$SCRIPT_DIR/../package.json"
    if [[ -f "$_pkg_json" ]] && command -v jq >/dev/null 2>&1; then  # SUPPRESS:OK -- cleanup: output suppression for clean operation
      _ver=$(jq -r '.version // "unknown"' "$_pkg_json" 2>/dev/null)  # SUPPRESS:OK -- read-default: file may not exist yet
      json_ok "$(jq -nc --arg v "$_ver" '$v')"
    else
      json_ok '"1.1.5"'
    fi
    ;;
  validate-state)
    # Migrated to state-api facade: uses _state_read_field for reads, _state_migrate for schema migration
    case "${1:-}" in
      colony)
        # Read full state via facade (handles missing file error)
        vs_state=$(_state_read_field '.')
        [[ -n "$vs_state" ]] || json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found" '{"file":"COLONY_STATE.json"}'
        # Run schema migration before field validation (ensures v3.0 fields always present)
        _state_migrate "$DATA_DIR/COLONY_STATE.json"
        json_ok "$(echo "$vs_state" | jq '
          def chk(f;t): if has(f) then (if (.[f]|type) as $a | t | any(. == $a) then "pass" else "fail: \(f) is \(.[f]|type), expected \(t|join("|"))" end) else "fail: missing \(f)" end;
          def opt(f;t): if has(f) then (if (.[f]|type) as $a | t | any(. == $a) then "pass" else "fail: \(f) is \(.[f]|type), expected \(t|join("|"))" end) else "pass" end;
          {file:"COLONY_STATE.json", checks:[
            chk("goal";["null","string"]),
            chk("state";["string"]),
            chk("current_phase";["number"]),
            chk("plan";["object"]),
            chk("memory";["object"]),
            chk("errors";["object"]),
            chk("events";["array"]),
            opt("session_id";["string","null"]),
            opt("initialized_at";["string","null"]),
            opt("build_started_at";["string","null"])
          ]} | . + {pass: (([.checks[] | select(. == "pass")] | length) == (.checks | length))}
        ')"
        ;;
      constraints)
        [[ -f "$COLONY_DATA_DIR/constraints.json" ]] || json_err "$E_FILE_NOT_FOUND" "constraints.json not found" '{"file":"constraints.json"}'
        [[ -s "$COLONY_DATA_DIR/constraints.json" ]] || { json_ok '{"file":"constraints.json","checks":["pass","pass"],"pass":true}'; exit 0; }
        json_ok "$(jq '
          def arr(f): if has(f) and (.[f]|type) == "array" then "pass" else "fail: \(f) not array" end;
          {file:"constraints.json", checks:[
            arr("focus"),
            arr("constraints")
          ]} | . + {pass: (([.checks[] | select(. == "pass")] | length) == (.checks | length))}
        ' "$COLONY_DATA_DIR/constraints.json")"
        ;;
      all)
        results=()
        for target in colony constraints; do
          # SUPPRESS:OK -- read-default: validation may fail on missing files
          results+=("$(bash "$SCRIPT_DIR/aether-utils.sh" validate-state "$target" 2>/dev/null || echo '{"ok":false}')")
        done
        combined=$(printf '%s\n' "${results[@]}" | jq -s '[.[] | .result // {file:"unknown",pass:false}]')
        all_pass=$(echo "$combined" | jq 'all(.pass)')
        json_ok "{\"pass\":$all_pass,\"files\":$combined}"
        ;;
      *)
        json_err "$E_VALIDATION_FAILED" "Usage: validate-state colony|constraints|all"
        ;;
    esac
    ;;
  state-checkpoint)
    # Create a rolling backup of COLONY_STATE.json before builds
    # Usage: bash .aether/aether-utils.sh state-checkpoint [reason]
    # Uses create_backup from atomic-write.sh (timestamped naming in BACKUP_DIR, max 3)
    sc_reason="${1:-manual}"
    sc_state_file="$DATA_DIR/COLONY_STATE.json"

    # Validate state file exists
    if [[ ! -f "$sc_state_file" ]]; then
      json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found -- cannot checkpoint"
    fi

    # Refuse to checkpoint corrupt state
    if ! jq -e . "$sc_state_file" >/dev/null 2>&1; then  # SUPPRESS:OK -- validation: testing JSON validity
      json_err "$E_JSON_INVALID" "COLONY_STATE.json is corrupt -- refusing to checkpoint invalid state"
    fi

    # Create timestamped backup via create_backup (handles MAX_BACKUPS=3 rotation)
    if type create_backup &>/dev/null; then
      create_backup "$sc_state_file" || json_err "$E_UNKNOWN" "Failed to create state checkpoint backup"
    else
      json_err "$E_FEATURE_UNAVAILABLE" "create_backup function not available -- atomic-write.sh may not be sourced"
    fi

    json_ok "$(jq -nc --arg reason "$sc_reason" '{checkpointed:true,reason:$reason}')"
    ;;
  state-write)
    # MIGRATE: direct COLONY_STATE.json access -- use _state_write instead
    # Delegates to _state_write in state-api.sh (preserves backward compatibility)
    _state_write "$@"
    ;;
  state-read)
    # Read full COLONY_STATE.json -- delegates to state-api.sh
    _state_read "$@"
    ;;
  state-read-field)
    # Read a specific field from COLONY_STATE.json -- wraps _state_read_field with json_ok
    srf_field="${1:-}"
    srf_raw=$(_state_read_field "$srf_field")
    # Wrap raw value in json_ok for subcommand output
    # If the value is valid JSON (object/array), pass it through; otherwise quote as string
    if echo "$srf_raw" | jq -e . >/dev/null 2>&1; then
      json_ok "$srf_raw"
    else
      json_ok "$(jq -n --arg v "$srf_raw" '$v')"
    fi
    ;;
  state-mutate)
    # Read-modify-write COLONY_STATE.json with a jq expression -- delegates to state-api.sh
    _state_mutate "$@"
    ;;
  verify-claims)
    # Cross-reference worker claims against reality (QUAL-08)
    # Usage: bash .aether/aether-utils.sh verify-claims <builder-claims-json> <watcher-claims-json-or-path> <test-exit-code>
    # Returns JSON with verification_status, checks_run, mismatches, blocked, summary
    _verify_claims() {
      local vc_builder_claims="${1:-}"
      local vc_watcher_claims="${2:-}"
      local vc_test_exit_code="${3:-0}"
      local vc_blocked=false
      local vc_mismatches="[]"
      local vc_checks_run=0
      local vc_summary="Verification passed"

      # --- Check 1: Builder-claimed files exist ---
      vc_checks_run=$((vc_checks_run + 1))

      if [[ -n "$vc_builder_claims" && -f "$vc_builder_claims" ]]; then
        # Read files_created and files_modified arrays
        local vc_files
        vc_files=$(jq -r '(.files_created // []) + (.files_modified // []) | .[]' "$vc_builder_claims" 2>/dev/null || echo "")

        if [[ -n "$vc_files" ]]; then
          local vc_missing_files=""
          local vc_missing_count=0
          while IFS= read -r vc_file_path; do
            if [[ -n "$vc_file_path" && ! -f "$vc_file_path" ]]; then
              vc_missing_count=$((vc_missing_count + 1))
              vc_missing_files="${vc_missing_files}${vc_file_path},"
            fi
          done <<< "$vc_files"

          if [[ $vc_missing_count -gt 0 ]]; then
            vc_blocked=true
            # Build mismatches array entries for missing files
            local vc_total_claimed
            vc_total_claimed=$(echo "$vc_files" | wc -l | tr -d ' ')
            vc_missing_files="${vc_missing_files%,}" # trim trailing comma
            # Build JSON array of missing file mismatches
            local vc_mm_arr="[]"
            IFS=',' read -ra vc_mm_parts <<< "$vc_missing_files"
            for vc_mm_f in "${vc_mm_parts[@]}"; do
              vc_mm_arr=$(echo "$vc_mm_arr" | jq --arg f "$vc_mm_f" '. + [{"type":"missing_file","file":$f,"message":"Worker claimed file was created/modified, but file does not exist"}]')
            done
            vc_mismatches="$vc_mm_arr"
            vc_summary="Worker claimed $vc_total_claimed files, but $vc_missing_count missing: ${vc_missing_files}. Blocked."
          fi
        fi
      fi
      # If builder claims file does not exist, skip file check gracefully (no block)

      # --- Check 2: Test exit code vs Watcher verification_passed ---
      vc_checks_run=$((vc_checks_run + 1))

      local vc_watcher_passed="true"
      if [[ -n "$vc_watcher_claims" ]]; then
        if [[ -f "$vc_watcher_claims" ]]; then
          vc_watcher_passed=$(jq -r '.verification_passed // true' "$vc_watcher_claims" 2>/dev/null || echo "true")
        else
          # Treat as inline JSON
          vc_watcher_passed=$(echo "$vc_watcher_claims" | jq -r '.verification_passed // true' 2>/dev/null || echo "true")
        fi
      fi

      # Fabrication: test exit code != 0 but watcher claims passed
      if [[ "$vc_test_exit_code" != "0" && "$vc_watcher_passed" == "true" ]]; then
        vc_blocked=true
        vc_mismatches=$(echo "$vc_mismatches" | jq --arg code "$vc_test_exit_code" '. + [{"type":"test_mismatch","message":"Test exit code was \($code) but watcher claimed verification_passed: true"}]')
        if [[ "$vc_summary" == "Verification passed" ]]; then
          vc_summary="Test exit code $vc_test_exit_code but watcher claimed tests passed. Blocked."
        else
          vc_summary="${vc_summary} Also: test exit code $vc_test_exit_code but watcher claimed tests passed."
        fi
      fi

      # --- Build result ---
      local vc_status="passed"
      if [[ "$vc_blocked" == "true" ]]; then
        vc_status="blocked"
      fi

      json_ok "$(jq -n \
        --arg status "$vc_status" \
        --argjson checks "$vc_checks_run" \
        --argjson mismatches "$vc_mismatches" \
        --argjson blocked "$vc_blocked" \
        --arg summary "$vc_summary" \
        '{
          "verification_status": $status,
          "checks_run": $checks,
          "mismatches": $mismatches,
          "blocked": $blocked,
          "summary": $summary
        }'
      )"
    }
    _verify_claims "$@"
    ;;
  validate-oracle-state)
    # Validate oracle state files (state.json, plan.json)
    # Usage: bash .aether/aether-utils.sh validate-oracle-state state|plan|all
    # Uses ORACLE_DIR env var override, defaulting to .aether/oracle
    ORACLE_DIR="${ORACLE_DIR:-.aether/oracle}"

    case "${1:-}" in
      state)
        [[ -f "$ORACLE_DIR/state.json" ]] || json_err "$E_FILE_NOT_FOUND" "state.json not found" '{"file":"state.json"}'
        json_ok "$(jq '
          def chk(f;t): if has(f) then (if (.[f]|type) as $a | t | any(. == $a) then "pass" else "fail: \(f) is \(.[f]|type), expected \(t|join("|"))" end) else "fail: missing \(f)" end;
          def enum(f;vals): if has(f) then (if [.[f]] | inside(vals) then "pass" else "fail: \(f) is \(.[f]), expected one of \(vals|join("|"))" end) else "fail: missing \(f)" end;
          {file:"state.json", checks:[
            chk("version";["string"]),
            chk("topic";["string"]),
            chk("scope";["string"]),
            enum("scope";["codebase","web","both"]),
            chk("phase";["string"]),
            enum("phase";["survey","investigate","synthesize","verify"]),
            chk("iteration";["number"]),
            chk("max_iterations";["number"]),
            chk("target_confidence";["number"]),
            chk("overall_confidence";["number"]),
            chk("started_at";["string"]),
            chk("last_updated";["string"]),
            chk("status";["string"]),
            enum("status";["active","complete","stopped"]),
            if has("strategy") then enum("strategy";["breadth-first","depth-first","adaptive"]) else "pass" end,
            if has("focus_areas") then (if (.focus_areas | type) == "array" then "pass" else "fail: focus_areas not array" end) else "pass" end,
            if has("template") then enum("template";["tech-eval","architecture-review","bug-investigation","best-practices","custom"]) else "pass" end
          ]} | . + {pass: (([.checks[] | select(. == "pass")] | length) == (.checks | length))}
        ' "$ORACLE_DIR/state.json")"
        ;;
      plan)
        [[ -f "$ORACLE_DIR/plan.json" ]] || json_err "$E_FILE_NOT_FOUND" "plan.json not found" '{"file":"plan.json"}'
        json_ok "$(jq '
          def chk(f;t): if has(f) then (if (.[f]|type) as $a | t | any(. == $a) then "pass" else "fail: \(f) is \(.[f]|type), expected \(t|join("|"))" end) else "fail: missing \(f)" end;
          {file:"plan.json", checks:[
            chk("version";["string"]),
            chk("questions";["array"]),
            chk("created_at";["string"]),
            chk("last_updated";["string"]),
            if (.questions | length) >= 1 and (.questions | length) <= 8
              then "pass"
              else "fail: questions count \(.questions | length) outside 1-8 range"
            end,
            if (.questions | all(has("id","text","status","confidence","key_findings","iterations_touched")))
              then "pass"
              else "fail: questions missing required fields (id, text, status, confidence, key_findings, iterations_touched)"
            end,
            if (.questions | all(.status == "open" or .status == "partial" or .status == "answered"))
              then "pass"
              else "fail: invalid status value (must be open|partial|answered)"
            end,
            if (.questions | all(.confidence >= 0 and .confidence <= 100))
              then "pass"
              else "fail: confidence out of 0-100 range"
            end,
            if (.sources // null) != null then
              if (.sources | type) == "object" then
                if (.sources | to_entries | length == 0) or (.sources | to_entries | all(.value | has("url","title","date_accessed"))) then "pass"
                else "fail: sources entries missing required fields (url, title, date_accessed)"
                end
              else "fail: sources must be an object"
              end
            else "pass"
            end,
            if ([.questions[].key_findings[] | type] | any(. == "object")) then
              if ([.questions[].key_findings[] | select(type == "object") | has("text","source_ids")] | all) then "pass"
              else "fail: structured findings missing required fields (text, source_ids)"
              end
            else "pass"
            end
          ]} | . + {pass: (([.checks[] | select(. == "pass")] | length) == (.checks | length))}
        ' "$ORACLE_DIR/plan.json")"
        ;;
      all)
        results=()
        for target in state plan; do
          # SUPPRESS:OK -- read-default: validation may fail on missing files
          results+=("$(ORACLE_DIR="$ORACLE_DIR" bash "$SCRIPT_DIR/aether-utils.sh" validate-oracle-state "$target" 2>/dev/null || echo '{"ok":false}')")
        done
        combined=$(printf '%s\n' "${results[@]}" | jq -s '[.[] | .result // {file:"unknown",pass:false}]')
        all_pass=$(echo "$combined" | jq 'all(.pass)')
        json_ok "{\"pass\":$all_pass,\"files\":$combined}"
        ;;
      *)
        json_err "$E_VALIDATION_FAILED" "Usage: validate-oracle-state state|plan|all"
        ;;
    esac
    ;;
  error-add)
    # Migrated to state-api facade: uses _state_mutate for atomic read-modify-write
    [[ $# -ge 3 ]] || json_err "$E_VALIDATION_FAILED" "Usage: error-add <category> <severity> <description> [phase]"

    ea_id="err_$(date -u +%s)_$(head -c 2 /dev/urandom | od -An -tx1 | tr -d ' ')"
    ea_ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    ea_phase_val="${4:-null}"
    if [[ "$ea_phase_val" =~ ^[0-9]+$ ]]; then
      ea_phase_jq="$ea_phase_val"
    else
      ea_phase_jq="null"
    fi
    EA_ID="$ea_id" EA_CAT="$1" EA_SEV="$2" EA_DESC="$3" EA_PHASE="$ea_phase_jq" EA_TS="$ea_ts" \
      _state_mutate '
        .errors.records += [{
          id: env.EA_ID,
          category: env.EA_CAT,
          severity: env.EA_SEV,
          description: env.EA_DESC,
          root_cause: null,
          phase: (env.EA_PHASE | if . == "null" then null else tonumber end),
          task_id: null,
          timestamp: env.EA_TS
        }] |
        if (.errors.records|length) > 50 then .errors.records = .errors.records[-50:] else . end
      ' >/dev/null

    json_ok "$(jq -nc --arg id "$ea_id" '$id')"
    ;;
  error-pattern-check)
    _deprecation_warning "error-pattern-check"
    # MIGRATE: direct COLONY_STATE.json access -- use _state_read_field instead
    [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found" '{"file":"COLONY_STATE.json"}'
    json_ok "$(jq '
      .errors.records | group_by(.category) | map(select(length >= 3) |
        {category: .[0].category, count: length,
         first_seen: (sort_by(.timestamp) | first.timestamp),
         last_seen: (sort_by(.timestamp) | last.timestamp)})
    ' "$DATA_DIR/COLONY_STATE.json")"
    ;;
  error-summary)
    _deprecation_warning "error-summary"
    # MIGRATE: direct COLONY_STATE.json access -- use _state_read_field instead
    [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found" '{"file":"COLONY_STATE.json"}'
    json_ok "$(jq '{
      total: (.errors.records | length),
      by_category: (.errors.records | group_by(.category) | map({key: .[0].category, value: length}) | from_entries),
      by_severity: (.errors.records | group_by(.severity) | map({key: .[0].severity, value: length}) | from_entries)
    }' "$DATA_DIR/COLONY_STATE.json")"
    ;;
  activity-log)
    # Usage: activity-log <action> <caste_or_name> <description>
    # The caste_or_name can be: "Builder", "Hammer-42 (Builder)", etc.
    action="${1:-}"
    caste="${2:-}"
    description="${3:-}"
    [[ -z "$action" || -z "$caste" || -z "$description" ]] && json_err "$E_VALIDATION_FAILED" "Usage: activity-log <action> <caste_or_name> <description>"

    # Graceful degradation: check if activity logging is enabled
    if type feature_enabled &>/dev/null && ! feature_enabled "activity_log"; then
      json_warn "W_DEGRADED" "Activity logging disabled: $(type _feature_reason &>/dev/null && _feature_reason activity_log || echo 'unknown')"
      exit 0
    fi

    # Skip writes during test runs to prevent test noise in activity.log
    if [[ "${AETHER_TESTING:-}" == "1" ]]; then
      json_ok '"logged"'
      exit 0
    fi

    log_file="$COLONY_DATA_DIR/activity.log"
    mkdir -p "$COLONY_DATA_DIR"
    ts=$(date -u +"%H:%M:%S")
    emoji=$(get_caste_emoji "$caste")
    echo "[$ts] $emoji $action $caste: $description" >> "$log_file"
    json_ok '"logged"'
    ;;
  activity-log-init)
    phase_num="${1:-}"
    phase_name="${2:-}"
    [[ -z "$phase_num" ]] && json_err "$E_VALIDATION_FAILED" "Usage: activity-log-init <phase_num> [phase_name]"

    # Graceful degradation: check if activity logging is enabled
    if type feature_enabled &>/dev/null && ! feature_enabled "activity_log"; then
      json_warn "W_DEGRADED" "Activity logging disabled: $(type _feature_reason &>/dev/null && _feature_reason activity_log || echo 'unknown')"
      exit 0
    fi
    log_file="$COLONY_DATA_DIR/activity.log"
    mkdir -p "$COLONY_DATA_DIR"
    archive_file="$COLONY_DATA_DIR/activity-phase-${phase_num}.log"
    # Copy current log to per-phase archive (preserve combined log intact)
    if [ -f "$log_file" ] && [ -s "$log_file" ]; then
      # Handle retry scenario: don't overwrite existing archive
      if [ -f "$archive_file" ]; then
        archive_file="$COLONY_DATA_DIR/activity-phase-${phase_num}-$(date -u +%s).log"
      fi
      cp "$log_file" "$archive_file"
    fi
    # Append phase header to combined log (NOT truncate)
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    echo "" >> "$log_file"
    echo "🐜 ═══════════════════════════════════════════════════" >> "$log_file"
    echo "   P H A S E   $phase_num: ${phase_name:-unnamed}" >> "$log_file"
    echo "   $ts" >> "$log_file"
    echo "═══════════════════════════════════════════════════ 🐜" >> "$log_file"
    archived_flag="false"
    [ -f "$archive_file" ] && archived_flag="true"
    json_ok "{\"archived\":$archived_flag}"
    ;;
  activity-log-read)
    caste_filter="${1:-}"

    # Graceful degradation: check if activity logging is enabled
    if type feature_enabled &>/dev/null && ! feature_enabled "activity_log"; then
      json_warn "W_DEGRADED" "Activity logging disabled: $(type _feature_reason &>/dev/null && _feature_reason activity_log || echo 'unknown')"
      exit 0
    fi

    log_file="$COLONY_DATA_DIR/activity.log"
    [[ -f "$log_file" ]] || json_err "$E_FILE_NOT_FOUND" "activity.log not found" '{"file":"activity.log"}'
    if [ -n "$caste_filter" ]; then
      content=$(grep -F "$caste_filter" "$log_file" | tail -20)
    else
      content=$(cat "$log_file")
    fi
    json_ok "$(echo "$content" | jq -Rs '.')"
    ;;
  learning-promote) _learning_promote "$@" ;;
  learning-inject) _learning_inject "$@" ;;
  spawn-log) _spawn_log "$@" ;;
  spawn-complete) _spawn_complete "$@" ;;
  spawn-can-spawn) _spawn_can_spawn "$@" ;;
  spawn-get-depth) _spawn_get_depth "$@" ;;
  update-progress)
    # Usage: update-progress <percent> <message> [phase] [total_phases]
    percent="${1:-0}"
    message="${2:-Working...}"
    phase="${3:-1}"
    total="${4:-1}"
    mkdir -p "$COLONY_DATA_DIR"

    # Calculate bar width (30 chars)
    bar_width=30
    filled=$((percent * bar_width / 100))
    empty=$((bar_width - filled))

    # Build progress bar with ASCII
    bar=""
    for ((i=0; i<filled; i++)); do bar+="█"; done
    for ((i=0; i<empty; i++)); do bar+="░"; done

    # Spinner frames for animation
    spinners=("⠋" "⠙" "⠹" "⠸" "⠼" "⠴" "⠦" "⠧" "⠇" "⠏")
    spin_idx=$(($(date +%s) % 10))
    spinner="${spinners[$spin_idx]}"

    # Status indicator
    if [[ $percent -ge 100 ]]; then
      status_icon="✅"
    elif [[ $percent -ge 50 ]]; then
      status_icon="🔨"
    else
      status_icon="$spinner"
    fi

    # Write progress file
    cat > "$COLONY_DATA_DIR/watch-progress.txt" << EOF
       .-.
      (o o)  AETHER COLONY
      | O |  Progress
       \`-\`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Phase: $phase / $total

[$bar] $percent%

$status_icon $message

Target: 95% confidence

EOF
    json_ok "{\"percent\":$percent,\"phase\":$phase}"
    ;;
  error-flag-pattern)
    # Usage: error-flag-pattern <pattern_name> <description> [severity]
    # Tracks recurring error patterns across sessions for colony learning
    pattern_name="${1:-}"
    description="${2:-}"
    severity="${3:-warning}"
    [[ -z "$pattern_name" || -z "$description" ]] && json_err "$E_VALIDATION_FAILED" "Usage: error-flag-pattern <pattern_name> <description> [severity]"

    patterns_file="$COLONY_DATA_DIR/error-patterns.json"
    mkdir -p "$COLONY_DATA_DIR"

    if [[ ! -f "$patterns_file" ]]; then
      atomic_write "$patterns_file" '{"patterns":[],"version":1}' || json_err "$E_UNKNOWN" "Failed to initialize error patterns file"
    fi

    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    project_name=$(basename "$PWD")

    # Check if pattern already exists
    # SUPPRESS:OK -- read-default: query may return empty
    existing=$(jq --arg name "$pattern_name" '.patterns[] | select(.name == $name)' "$patterns_file" 2>/dev/null)

    if [[ -n "$existing" ]]; then
      # Update existing pattern - increment count
      updated=$(jq --arg name "$pattern_name" --arg ts "$ts" --arg proj "$project_name" '
        .patterns = [.patterns[] | if .name == $name then
          .occurrences += 1 |
          .last_seen = $ts |
          .projects = ((.projects + [$proj]) | unique)
        else . end]
      ' "$patterns_file") || json_err "$E_JSON_INVALID" "Failed to update pattern"
      atomic_write "$patterns_file" "$updated" || {
        _aether_log_error "Could not save updated error patterns"
        json_err "$E_UNKNOWN" "Failed to write error patterns file"
      }
      count=$(echo "$updated" | jq --arg name "$pattern_name" '.patterns[] | select(.name == $name) | .occurrences')
      json_ok "$(jq -nc --arg p "$pattern_name" --argjson c "$count" '{updated:true,pattern:$p,occurrences:$c}')"
    else
      # Add new pattern
      updated=$(jq --arg name "$pattern_name" --arg desc "$description" --arg sev "$severity" --arg ts "$ts" --arg proj "$project_name" '
        .patterns += [{
          "name": $name,
          "description": $desc,
          "severity": $sev,
          "first_seen": $ts,
          "last_seen": $ts,
          "occurrences": 1,
          "projects": [$proj],
          "resolved": false
        }]
      ' "$patterns_file") || json_err "$E_JSON_INVALID" "Failed to add pattern"
      atomic_write "$patterns_file" "$updated" || {
        _aether_log_error "Could not save new error pattern"
        json_err "$E_UNKNOWN" "Failed to write error patterns file"
      }
      json_ok "$(jq -nc --arg p "$pattern_name" '{created:true,pattern:$p}')"
    fi
    ;;
  error-patterns-check)
    _deprecation_warning "error-patterns-check"
    # Check for known error patterns in a file or codebase
    # Returns patterns that should be avoided
    global_file="$COLONY_DATA_DIR/error-patterns.json"

    if [[ ! -f "$global_file" ]]; then
      json_ok '{"patterns":[],"count":0}'
      exit 0
    fi

    # Return patterns with 2+ occurrences (recurring issues)
    json_ok "$(jq '{
      patterns: [.patterns[] | select(.occurrences >= 2 and .resolved == false)],
      count: ([.patterns[] | select(.occurrences >= 2 and .resolved == false)] | length)
    }' "$global_file")"
    ;;
  check-antipattern)
    # Usage: check-antipattern <file_path>
    # Returns JSON with critical issues and warnings
    file_path="${1:-}"
    [[ -z "$file_path" ]] && json_err "$E_VALIDATION_FAILED" "Usage: check-antipattern <file_path>"
    [[ ! -f "$file_path" ]] && json_ok '{"critical":[],"warnings":[],"clean":true}'

    criticals=()
    warnings=()

    # Detect file type
    ext="${file_path##*.}"

    case "$ext" in
      swift)
        # Swift didSet infinite recursion check
        if grep -n "didSet" "$file_path" 2>/dev/null | grep -q "self\."; then  # SUPPRESS:OK -- existence-test: file may not exist
          line=$(grep -n "didSet" "$file_path" | grep "self\." | head -1 | cut -d: -f1)
          criticals+=("{\"pattern\":\"didSet-recursion\",\"file\":\"$file_path\",\"line\":$line,\"message\":\"Potential didSet infinite recursion - self assignment in didSet\"}")
        fi
        ;;
      ts|tsx|js|jsx)
        # TypeScript any type check
        if grep -nE '\bany\b' "$file_path" 2>/dev/null | grep -qv "//.*any"; then  # SUPPRESS:OK -- existence-test: file may not exist
          count=$(grep -cE '\bany\b' "$file_path" 2>/dev/null || echo "0")  # SUPPRESS:OK -- read-default: count defaults to 0 if file missing
          warnings+=("{\"pattern\":\"typescript-any\",\"file\":\"$file_path\",\"count\":$count,\"message\":\"Found $count uses of 'any' type\"}")
        fi
        # Console.log in production code (not in test files)
        if [[ ! "$file_path" =~ \.test\. && ! "$file_path" =~ \.spec\. ]]; then
          if grep -n "console\.log" "$file_path" 2>/dev/null | grep -qv "//"; then  # SUPPRESS:OK -- existence-test: file may not exist
            count=$(grep -c "console\.log" "$file_path" 2>/dev/null || echo "0")  # SUPPRESS:OK -- read-default: count defaults to 0 if file missing
            warnings+=("{\"pattern\":\"console-log\",\"file\":\"$file_path\",\"count\":$count,\"message\":\"Found $count console.log statements\"}")
          fi
        fi
        ;;
      py)
        # Python bare except
        if grep -n "except:" "$file_path" 2>/dev/null | grep -qv "#"; then  # SUPPRESS:OK -- existence-test: file may not exist
          line=$(grep -n "except:" "$file_path" | head -1 | cut -d: -f1)
          warnings+=("{\"pattern\":\"bare-except\",\"file\":\"$file_path\",\"line\":$line,\"message\":\"Bare except clause - specify exception type\"}")
        fi
        ;;
    esac

    # Common patterns across all languages
    # Exposed secrets check (critical)
    # SUPPRESS:OK -- existence-test: grep returns 1 when no matches
    if grep -nE "(api_key|apikey|secret|password|token)\s*=\s*['\"][^'\"]+['\"]" "$file_path" 2>/dev/null | grep -qvi "example\|test\|mock\|fake"; then
      line=$(grep -nE "(api_key|apikey|secret|password|token)\s*=\s*['\"]" "$file_path" | head -1 | cut -d: -f1)
      criticals+=("{\"pattern\":\"exposed-secret\",\"file\":\"$file_path\",\"line\":${line:-0},\"message\":\"Potential hardcoded secret or credential\"}")
    fi

    # TODO/FIXME check (warning)
    if grep -nE "(TODO|FIXME|XXX|HACK)" "$file_path" 2>/dev/null | head -1 | grep -q .; then  # SUPPRESS:OK -- existence-test: file may not exist
      count=$(grep -cE "(TODO|FIXME|XXX|HACK)" "$file_path" 2>/dev/null || echo "0")  # SUPPRESS:OK -- read-default: count defaults to 0 if file missing
      warnings+=("{\"pattern\":\"todo-comment\",\"file\":\"$file_path\",\"count\":$count,\"message\":\"Found $count TODO/FIXME comments\"}")
    fi

    # Build result JSON
    crit_json="[]"
    warn_json="[]"
    if [[ ${#criticals[@]} -gt 0 ]]; then
      crit_json=$(printf '%s\n' "${criticals[@]}" | jq -s '.')
    fi
    if [[ ${#warnings[@]} -gt 0 ]]; then
      warn_json=$(printf '%s\n' "${warnings[@]}" | jq -s '.')
    fi

    clean="true"
    [[ ${#criticals[@]} -gt 0 || ${#warnings[@]} -gt 0 ]] && clean="false"

    json_ok "{\"critical\":$crit_json,\"warnings\":$warn_json,\"clean\":$clean}"
    ;;
  signature-scan)
    # Scan a file for a signature pattern
    # Usage: signature-scan <target_file> <signature_name>
    # Returns matching signature details as JSON if found, empty result if no match
    # Exit code 0 if no match, 1 if match found
    target_file="${1:-}"
    signature_name="${2:-}"
    [[ -z "$target_file" || -z "$signature_name" ]] && json_err "$E_VALIDATION_FAILED" "Usage: signature-scan <target_file> <signature_name>"

    # Handle missing target file gracefully
    if [[ ! -f "$target_file" ]]; then
      json_ok '{"found":false,"signature":null}'
      exit 0
    fi

    # Read signature details from signatures.json
    signatures_file="$COLONY_DATA_DIR/signatures.json"
    if [[ ! -f "$signatures_file" ]]; then
      json_ok '{"found":false,"signature":null}'
      exit 0
    fi

    # Extract signature details using jq
    # SUPPRESS:OK -- read-default: query may return empty
    signature_data=$(jq --arg name "$signature_name" '.signatures[] | select(.name == $name)' "$signatures_file" 2>/dev/null)

    if [[ -z "$signature_data" ]]; then
      # Signature not found in storage
      json_ok '{"found":false,"signature":null}'
      exit 0
    fi

    # Extract pattern and confidence threshold
    pattern_string=$(echo "$signature_data" | jq -r '.pattern_string // empty')
    confidence_threshold=$(echo "$signature_data" | jq -r '.confidence_threshold // 0.8')

    if [[ -z "$pattern_string" || "$pattern_string" == "null" ]]; then
      json_ok '{"found":false,"signature":null}'
      exit 0
    fi

    # Use grep to search for the pattern in target file
    if grep -q -- "$pattern_string" "$target_file" 2>/dev/null; then  # SUPPRESS:OK -- existence-test: file may not exist
      # Match found - return signature details with match info
      match_count=$(grep -c -- "$pattern_string" "$target_file" 2>/dev/null || echo "1")  # SUPPRESS:OK -- read-default: count defaults to 0 if file missing
      json_ok "{\"found\":true,\"signature\":$signature_data,\"match_count\":$match_count}"
      exit 1
    else
      # No match
      json_ok '{"found":false,"signature":null}'
      exit 0
    fi
    ;;
  signature-match)
    # Scan a directory for files matching high-confidence signatures
    # Usage: signature-match <directory> [file_pattern]
    # Returns results per file showing which signatures matched
    target_dir="${1:-}"
    file_pattern="${2:-}"
    # Set default pattern if not provided - avoid zsh brace expansion quirk by setting it explicitly
    if [[ -z "$file_pattern" ]]; then
      file_pattern="*"
    fi
    [[ -z "$target_dir" ]] && json_err "$E_VALIDATION_FAILED" "Usage: signature-match <directory> [file_pattern]"

    # Validate directory exists
    [[ ! -d "$target_dir" ]] && json_err "$E_FILE_NOT_FOUND" "Directory not found: $target_dir"

    # Path to signatures file
    signatures_file="$COLONY_DATA_DIR/signatures.json"
    [[ ! -f "$signatures_file" ]] && json_err "$E_FILE_NOT_FOUND" "Signatures file not found"

    # Read high-confidence signatures (confidence >= 0.7) using jq -c for compact single-line output
    # SUPPRESS:OK -- read-default: query may return empty
    high_conf_signatures=$(jq -c '.signatures[] | select(.confidence_threshold >= 0.7)' "$signatures_file" 2>/dev/null)

    # Check if any high-confidence signatures exist
    sig_count=$(echo "$high_conf_signatures" | grep -c '{' || echo 0)  # SUPPRESS:OK -- read-default: grep returns 1 when no matches
    if [[ "$sig_count" -eq 0 ]]; then
      json_ok '{"files_scanned":0,"matches":{},"signatures_checked":0}'
      exit 0
    fi

    # Find all files to scan
    declare -a files=()
    if [[ -n "$file_pattern" ]]; then
      # User specified pattern - use it directly
      while IFS= read -r -d '' file; do
        files+=("$file")
      done < <(find "$target_dir" -type f -name "$file_pattern" -print0 2>/dev/null || true)  # SUPPRESS:OK -- existence-test: directory may not exist
    else
      # Default: match common code file types
      while IFS= read -r -d '' file; do
        files+=("$file")
      # SUPPRESS:OK -- existence-test: directory may not exist
      done < <(find "$target_dir" -type f \( -name "*.js" -o -name "*.ts" -o -name "*.py" -o -name "*.sh" -o -name "*.txt" -o -name "*.md" \) -print0 2>/dev/null || true)
    fi

    file_count=${#files[@]}

    # If no files found, return empty result
    if [[ "$file_count" -eq 0 ]]; then
      json_ok "{\"files_scanned\":0,\"matches\":{},\"signatures_checked\":$sig_count}"
      exit 0
    fi

    # Collect matches per file - process each file (bash 3.2 compatible: build JSON directly)
    matched_files="{}"

    # Read signatures into array first (avoid subshell issues)
    sig_array=""
    while IFS= read -r sig_entry; do
      [[ -z "$sig_entry" ]] && continue
      sig_array="${sig_array}${sig_entry}"$'\n'
    done <<< "$high_conf_signatures"

    for file in "${files[@]}"; do
      # For each file, check each signature - use process subst to avoid subshell
      file_key=$(basename "$file")
      matches_for_file="[]"

      while IFS= read -r sig_entry; do
        [[ -z "$sig_entry" ]] && continue
        sig_name=$(echo "$sig_entry" | jq -r '.name')
        sig_pattern=$(echo "$sig_entry" | jq -r '.pattern_string')
        sig_conf=$(echo "$sig_entry" | jq -r '.confidence_threshold')
        sig_desc=$(echo "$sig_entry" | jq -r '.description')

        # Skip if pattern is null/empty
        [[ -z "$sig_pattern" || "$sig_pattern" == "null" ]] && continue

        # Check if pattern matches in file using grep
        if grep -q -- "$sig_pattern" "$file" 2>/dev/null; then  # SUPPRESS:OK -- existence-test: file may not exist
          match_count=$(grep -c -- "$sig_pattern" "$file" 2>/dev/null || echo "1")  # SUPPRESS:OK -- read-default: count defaults to 0 if file missing

          # Add to results
          matches_for_file=$(echo "$matches_for_file" | jq --arg n "$sig_name" --arg d "$sig_desc" --argjson c "$sig_conf" --argjson m "$match_count" \
            '. += [{"name":$n,"description":$d,"confidence_threshold":$c,"match_count":$m}]')
        fi
      done < <(echo "$high_conf_signatures" | jq -c '.' 2>/dev/null || true)  # SUPPRESS:OK -- read-default: returns fallback if missing

      # If any signatures matched, add to results
      sig_result_count=$(echo "$matches_for_file" | jq 'length')
      if [[ "$sig_result_count" -gt 0 ]]; then
        temp_result=$(mktemp)
        echo "$matched_files" | jq --arg k "$file_key" --argjson v "$matches_for_file" '. + {($k): $v}' > "$temp_result"
        matched_files=$(cat "$temp_result")
        rm -f "$temp_result"
      fi
    done

    json_ok "{\"files_scanned\":$file_count,\"matches\":$matched_files,\"signatures_checked\":$sig_count}"
    ;;
  flag-add) _flag_add "$@" ;;
  flag-check-blockers) _flag_check_blockers "$@" ;;
  flag-resolve) _flag_resolve "$@" ;;
  flag-acknowledge) _flag_acknowledge "$@" ;;
  flag-list) _flag_list "$@" ;;
  flag-auto-resolve) _flag_auto_resolve "$@" ;;
  generate-ant-name)
    caste="${1:-builder}"
    # Caste-specific prefixes for personality
    case "$caste" in
      builder)  prefixes=("Chip" "Hammer" "Forge" "Mason" "Brick" "Anvil" "Weld" "Bolt") ;;
      watcher)  prefixes=("Vigil" "Sentinel" "Guard" "Keen" "Sharp" "Hawk" "Watch" "Alert") ;;
      scout)    prefixes=("Swift" "Dash" "Ranger" "Track" "Seek" "Path" "Roam" "Quest") ;;
      colonizer) prefixes=("Pioneer" "Map" "Chart" "Venture" "Explore" "Compass" "Atlas" "Trek") ;;
      architect) prefixes=("Blueprint" "Draft" "Design" "Plan" "Schema" "Frame" "Sketch" "Model") ;;
      prime)    prefixes=("Prime" "Alpha" "Lead" "Chief" "First" "Core" "Apex" "Crown") ;;
      chaos)    prefixes=("Probe" "Stress" "Shake" "Twist" "Snap" "Breach" "Surge" "Jolt") ;;
      archaeologist) prefixes=("Relic" "Fossil" "Dig" "Shard" "Epoch" "Strata" "Lore" "Glyph") ;;
      oracle)   prefixes=("Sage" "Seer" "Vision" "Augur" "Mystic" "Sibyl" "Delph" "Pythia") ;;
      ambassador) prefixes=("Bridge" "Connect" "Link" "Diplomat" "Protocol" "Network" "Port" "Socket") ;;
      auditor)   prefixes=("Review" "Inspect" "Exam" "Scrutin" "Verify" "Check" "Audit" "Assess") ;;
      chronicler) prefixes=("Record" "Write" "Document" "Chronicle" "Scribe" "Archive" "Script" "Ledger") ;;
      gatekeeper) prefixes=("Guard" "Protect" "Secure" "Shield" "Defend" "Bar" "Gate" "Checkpoint") ;;
      guardian)  prefixes=("Defend" "Patrol" "Watch" "Vigil" "Shield" "Guard" "Armor" "Fort") ;;
      includer)  prefixes=("Access" "Include" "Open" "Welcome" "Reach" "Universal" "Equal" "A11y") ;;
      keeper)    prefixes=("Archive" "Store" "Curate" "Preserve" "Guard" "Keep" "Hold" "Save") ;;
      measurer)  prefixes=("Metric" "Gauge" "Scale" "Measure" "Benchmark" "Track" "Count" "Meter") ;;
      probe)     prefixes=("Test" "Probe" "Excavat" "Uncover" "Edge" "Mutant" "Trial" "Check") ;;
      tracker)   prefixes=("Track" "Trace" "Debug" "Hunt" "Follow" "Trail" "Find" "Seek") ;;
      weaver)    prefixes=("Weave" "Knit" "Spin" "Twine" "Transform" "Mend" "Weave" "Weave") ;;
      *)        prefixes=("Ant" "Worker" "Drone" "Toiler" "Marcher" "Runner" "Carrier" "Helper") ;;
    esac
    # Pick random prefix and add random number
    idx=$((RANDOM % ${#prefixes[@]}))
    prefix="${prefixes[$idx]}"
    num=$((RANDOM % 99 + 1))
    name="${prefix}-${num}"
    json_ok "$(jq -nc --arg n "$name" '$n')"
    ;;

  validate-worker-response)
    # Validate worker JSON payloads against caste-specific schemas.
    # Usage: validate-worker-response <caste> <json_or_file_path>
    vw_caste="${1:-}"
    vw_input="${2:-}"

    [[ -z "$vw_caste" ]] && json_err "$E_VALIDATION_FAILED" "Usage: validate-worker-response <caste> <json_or_file_path>" '{"missing":"caste"}'
    [[ -z "$vw_input" ]] && json_err "$E_VALIDATION_FAILED" "Usage: validate-worker-response <caste> <json_or_file_path>" '{"missing":"json_or_file_path"}'

    if [[ -f "$vw_input" ]]; then
      vw_json=$(cat "$vw_input" 2>/dev/null || echo "")  # SUPPRESS:OK -- read-default: file may not exist yet
    else
      vw_json="$vw_input"
    fi

    if [[ -z "$vw_json" ]] || ! echo "$vw_json" | jq -e . >/dev/null 2>&1; then  # SUPPRESS:OK -- validation: testing JSON validity
      json_err "$E_JSON_INVALID" "Worker response is not valid JSON"
    fi

    vw_caste=$(echo "$vw_caste" | tr '[:upper:]' '[:lower:]')

    case "$vw_caste" in
      builder)
        vw_required='["ant_name","task_id","status","summary","tool_count","files_created","files_modified","tests_written","blockers"]'
        vw_schema='
          (.ant_name|type=="string") and
          (.task_id|type=="string") and
          (.status|type=="string" and (.=="completed" or .=="failed" or .=="blocked")) and
          (.summary|type=="string") and
          (.tool_count|type=="number") and
          (.files_created|type=="array") and
          (.files_modified|type=="array") and
          (.tests_written|type=="array") and
          (.blockers|type=="array")
        '
        ;;
      watcher)
        vw_required='["ant_name","verification_passed","files_verified","issues_found","quality_score","tool_count","recommendation"]'
        vw_schema='
          (.ant_name|type=="string") and
          (.verification_passed|type=="boolean") and
          (.files_verified|type=="array") and
          (.issues_found|type=="array") and
          (.quality_score|type=="number") and
          (.tool_count|type=="number") and
          (.recommendation|type=="string" and (.=="proceed" or .=="fix_required"))
        '
        ;;
      probe)
        vw_required='["ant_name","status","summary","coverage","tests_added","edge_cases_discovered","mutation_score","weak_spots","blockers"]'
        vw_schema='
          (.ant_name|type=="string") and
          (.status|type=="string" and (.=="completed" or .=="failed" or .=="blocked")) and
          (.summary|type=="string") and
          (.coverage|type=="object") and
          (.coverage.lines|type=="number") and
          (.coverage.branches|type=="number") and
          (.coverage.functions|type=="number") and
          (.tests_added|type=="array") and
          (.edge_cases_discovered|type=="array") and
          (.mutation_score|type=="number") and
          (.weak_spots|type=="array") and
          (.blockers|type=="array")
        '
        ;;
      scout)
        vw_required='["ant_name","status","summary","key_findings","recommendations","sources","spawns"]'
        vw_schema='
          (.ant_name|type=="string") and
          (.status|type=="string" and (.=="completed" or .=="failed" or .=="blocked")) and
          (.summary|type=="string") and
          (.key_findings|type=="array") and
          (.recommendations|type=="array") and
          (.sources|type=="array") and
          (.spawns|type=="array")
        '
        ;;
      *)
        vw_required='["ant_name","status","summary"]'
        vw_schema='
          (.ant_name|type=="string") and
          (.status|type=="string" and (.=="completed" or .=="failed" or .=="blocked")) and
          (.summary|type=="string")
        '
        ;;
    esac

    # SUPPRESS:OK -- read-default: query may return empty
    missing_fields=$(echo "$vw_json" | jq -c --argjson req "$vw_required" '[ $req[] | select(has(.) | not) ]' 2>/dev/null || echo '[]')
    if [[ "$missing_fields" != "[]" ]]; then
      details=$(jq -n --arg caste "$vw_caste" --argjson missing "$missing_fields" '{caste:$caste,missing:$missing}')
      json_err "$E_VALIDATION_FAILED" "Worker response missing required fields" "$details"
    fi

    if ! echo "$vw_json" | jq -e "$vw_schema" >/dev/null 2>&1; then  # SUPPRESS:OK -- validation: testing JSON against schema
      json_err "$E_VALIDATION_FAILED" "Worker response failed schema validation" "{\"caste\":\"$vw_caste\"}"
    fi

    json_ok "$(jq -nc --arg c "$vw_caste" '{valid:true,caste:$c}')"
    ;;

  # ============================================
  # SWARM UTILITIES (ant:swarm support) — dispatched to utils/swarm.sh
  # ============================================

  autofix-checkpoint) _autofix_checkpoint "$@" ;;
  autofix-rollback) _autofix_rollback "$@" ;;

  spawn-can-spawn-swarm) _spawn_can_spawn_swarm "$@" ;;

  swarm-findings-init) _swarm_findings_init "$@" ;;
  swarm-findings-add) _swarm_findings_add "$@" ;;
  swarm-findings-read) _swarm_findings_read "$@" ;;
  swarm-solution-set) _swarm_solution_set "$@" ;;
  swarm-cleanup) _swarm_cleanup "$@" ;;

  grave-add)
    # Migrated to state-api facade: uses _state_mutate for atomic grave marker recording
    # Record a grave marker when a builder fails at a file
    # Usage: grave-add <file> <ant_name> <task_id> <phase> <failure_summary> [function] [line]
    [[ $# -ge 5 ]] || json_err "$E_VALIDATION_FAILED" "Usage: grave-add <file> <ant_name> <task_id> <phase> <failure_summary> [function] [line]"

    ga_file="$1"
    ga_ant_name="$2"
    ga_task_id="$3"
    ga_phase="$4"
    ga_failure_summary="$5"
    ga_func="${6:-}"
    ga_line="${7:-}"
    ga_id="grave_$(date -u +%s)_$(head -c 2 /dev/urandom | od -An -tx1 | tr -d ' ')"
    ga_ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    GA_ID="$ga_id" GA_FILE="$ga_file" GA_ANT="$ga_ant_name" GA_TID="$ga_task_id" \
    GA_PHASE="$ga_phase" GA_SUMMARY="$ga_failure_summary" GA_FUNC="$ga_func" \
    GA_LINE="$ga_line" GA_TS="$ga_ts" \
      _state_mutate '
        (env.GA_PHASE | if test("^[0-9]+$") then tonumber else null end) as $phase |
        (env.GA_FUNC | if . == "" or . == "null" then null else . end) as $func |
        (env.GA_LINE | if test("^[0-9]+$") then tonumber else null end) as $line |
        (.graveyards // []) as $graves |
        . + {graveyards: ($graves + [{
          id: env.GA_ID,
          file: env.GA_FILE,
          ant_name: env.GA_ANT,
          task_id: env.GA_TID,
          phase: $phase,
          failure_summary: env.GA_SUMMARY,
          function: $func,
          line: $line,
          timestamp: env.GA_TS
        }])} |
        if (.graveyards | length) > 30 then .graveyards = .graveyards[-30:] else . end
      ' >/dev/null

    json_ok "$(jq -nc --arg id "$ga_id" '$id')"
    ;;

  grave-check)
    # Migrated to state-api facade: uses _state_read_field for read-only access
    # Query for grave markers near a file path
    # Usage: grave-check <file_path>
    # Read-only, never modifies state
    [[ $# -ge 1 ]] || json_err "$E_VALIDATION_FAILED" "Usage: grave-check <file_path>"
    gc_check_file="$1"
    gc_check_dir=$(dirname "$gc_check_file")
    gc_state=$(_state_read_field '.')
    [[ -n "$gc_state" ]] || json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found" '{"file":"COLONY_STATE.json"}'
    json_ok "$(echo "$gc_state" | jq --arg file "$gc_check_file" --arg dir "$gc_check_dir" '
      (.graveyards // []) as $graves |
      ($graves | map(select(.file == $file))) as $exact |
      ($graves | map(select((.file | split("/")[:-1] | join("/")) == $dir))) as $dir_matches |
      ($exact | length) as $exact_count |
      ($dir_matches | length) as $dir_count |
      (if $exact_count > 0 then "high"
       elif $dir_count >= 2 then "high"
       elif $dir_count == 1 then "low"
       else "none" end) as $caution |
      {graves: $dir_matches, count: $dir_count, exact_matches: $exact_count, caution_level: $caution}
    ')"
    ;;

  # ============================================
  # GIT COMMIT UTILITIES
  # ============================================

  generate-commit-message)
    # Generate an intelligent commit message from colony context
    # Usage: generate-commit-message <type> <phase_id> <phase_name> [summary|ai_description] [plan_num]
    # Types: "milestone" | "pause" | "fix" | "contextual" | "seal"
    # Returns: {"message": "...", "body": "...", "files_changed": N, ...}

    msg_type="${1:-milestone}"
    phase_id="${2:-0}"
    phase_name="${3:-unknown}"
    summary="${4:-}"        # For milestone/fix types, or ai_description for contextual type
    plan_num="${5:-01}"     # Optional: plan number for contextual type (e.g., "01")

    # Count changed files
    files_changed=0
    if git rev-parse --git-dir >/dev/null 2>&1; then  # SUPPRESS:OK -- existence-test: may not be a git repo
      # SUPPRESS:OK -- existence-test: grep returns 1 when no matches
      files_changed=$(git diff --stat --cached HEAD 2>/dev/null | tail -1 | grep -oE '[0-9]+ file' | grep -oE '[0-9]+' || echo "0")
      if [[ "$files_changed" == "0" ]]; then
        files_changed=$(git status --porcelain 2>/dev/null | wc -l | tr -d ' ')  # SUPPRESS:OK -- existence-test: may not be a git repo
      fi
    fi

    case "$msg_type" in
      milestone)
        # Format: aether-milestone: phase N complete -- <name>
        if [[ -n "$summary" ]]; then
          message="aether-milestone: phase ${phase_id} complete -- ${summary}"
        else
          message="aether-milestone: phase ${phase_id} complete -- ${phase_name}"
        fi
        body="All verification gates passed. User confirmed runtime behavior."
        ;;
      pause)
        message="aether-checkpoint: session pause -- phase ${phase_id} in progress"
        body="Colony paused mid-session. Handoff document saved."
        ;;
      fix)
        if [[ -n "$summary" ]]; then
          message="fix: ${summary}"
        else
          message="fix: resolve issue in phase ${phase_id}"
        fi
        body="Swarm-verified fix applied and tested."
        ;;
      contextual)
        # NEW: Contextual commit with AI description and structured metadata
        # Derive subsystem from phase name (e.g., "11-foraging-specialization" -> "foraging")
        subsystem=$(echo "$phase_name" | sed -E 's/^[0-9]+-//' | sed -E 's/-[0-9]+.*$//' | tr '-' ' ')
        [[ -z "$subsystem" ]] && subsystem="phase"

        # Build message with AI description (summary parameter is reused as ai_description)
        if [[ -n "$summary" ]]; then
          message="aether-milestone: ${summary}"
        else
          # Fallback if no AI description provided
          message="aether-milestone: phase ${phase_id}.${plan_num} complete -- ${phase_name}"
        fi

        # Build structured body with metadata
        body="Scope: ${phase_id}.${plan_num}
Files: ${files_changed} files changed"

        # Truncate message if needed BEFORE JSON construction
        if [[ ${#message} -gt 72 ]]; then
          message="${message:0:69}..."
        fi

        # Return enhanced JSON with additional metadata
        json_ok "$(jq -nc --arg msg "$message" --arg body "$body" --arg sub "$subsystem" --arg scope "${phase_id}.${plan_num}" '{message:$msg,body:$body,files_changed:'"$files_changed"',subsystem:$sub,scope:$scope}')"
        exit 0
        ;;
      seal)
        # Format: aether-seal: <goal summary>
        # Usage: generate-commit-message seal <total_phases> <goal> [colony_age_days]
        # phase_id = total_phases, phase_name = goal, summary = colony_age_days
        seal_total="$phase_id"
        seal_goal="$phase_name"
        seal_age="${summary:-}"

        message="aether-seal: ${seal_goal}"

        # Build body with colony summary (use literal \n for JSON)
        body="Crowned Anthill reached. ${seal_total} phases completed."
        [[ -n "$seal_age" ]] && body="${body} Colony age: ${seal_age} days."

        # Truncate message before JSON
        if [[ ${#message} -gt 72 ]]; then
          message="${message:0:69}..."
        fi

        json_ok "$(jq -nc --arg msg "$message" --arg body "$body" '{message:$msg,body:$body,files_changed:'"$files_changed"'}')"
        exit 0
        ;;
      *)
        message="aether-checkpoint: phase ${phase_id}"
        body=""
        ;;
    esac

    # Enforce 72-char limit on subject line (truncate if needed)
    if [[ ${#message} -gt 72 ]]; then
      message="${message:0:69}..."
    fi

    json_ok "$(jq -nc --arg msg "$message" --arg body "$body" '{message:$msg,body:$body,files_changed:'"$files_changed"'}')"
    ;;

  # ============================================
  # CONTEXT PERSISTENCE SYSTEM
  # ============================================

  context-update)
    # Update .aether/CONTEXT.md with current colony state
    # Usage: context-update <action> [args...]
    #
    # Actions:
    #   init <goal>                              - Initialize new context
    #   update-phase <phase_id> <name>           - Update current phase
    #   activity <command> <result> [files]      - Log activity
    #   constraint <type> <message> [source]     - Add constraint (redirect/focus)
    #   decision <description> [rationale] [who] - Log decision
    #   safe-to-clear <yes|no> <reason>          - Set safe-to-clear status
    #   build-start <phase_id> <workers> <tasks> - Mark build starting
    #   worker-spawn <ant_name> <caste> <task>   - Log worker spawn
    #   worker-complete <ant_name> <status>      - Log worker completion
    #   build-progress <completed> <total>       - Update build progress
    #   build-complete <status> <result>         - Mark build complete
    #
    # Always call with explicit arguments - never rely on current directory
    # CONTEXT_FILE must be passed or detected from AETHER_ROOT
    _cmd_context_update "$@"
    ;;

  # ============================================
  # REGISTRY & UPDATE UTILITIES
  # ============================================

  version-check)
    # Compare local .aether/version.json vs ~/.aether/version.json
    # Outputs a notice string if versions differ, empty if matched or missing
    local_version_file="$AETHER_ROOT/.aether/version.json"
    hub_version_file="$HOME/.aether/version.json"

    # Silent exit if either file is missing
    if [[ ! -f "$local_version_file" || ! -f "$hub_version_file" ]]; then
      json_ok '""'
      exit 0
    fi

    local_ver=$(jq -r '.version // "unknown"' "$local_version_file" 2>/dev/null || echo "unknown")  # SUPPRESS:OK -- read-default: file may not exist yet
    hub_ver=$(jq -r '.version // "unknown"' "$hub_version_file" 2>/dev/null || echo "unknown")  # SUPPRESS:OK -- read-default: file may not exist yet

    if [[ "$local_ver" == "$hub_ver" ]]; then
      json_ok '""'
    else
      printf -v msg 'Update available: %s to %s (run /ant:update)' "$local_ver" "$hub_ver"
      json_ok "$(jq -nc --arg m "$msg" '$m')"
    fi
    ;;

  version-check-cached)
    # Cached version of version-check — skips if checked within TTL (3600s = 1 hour)
    # Usage: version-check-cached
    cache_file="$AETHER_ROOT/.aether/data/.version-check-cache"
    now=$(date +%s)

    if [[ -f "$cache_file" ]]; then
      cached_at=$(cat "$cache_file" 2>/dev/null || echo "0")  # SUPPRESS:OK -- read-default: file may not exist yet
      age=$((now - cached_at))
      if [[ $age -lt 3600 ]]; then
        # Within TTL — skip silently
        json_ok '""'
        exit 0
      fi
    fi

    # Cache miss or stale — run actual check
    mkdir -p "$(dirname "$cache_file")" 2>/dev/null || true  # SUPPRESS:OK -- idempotent: harmless if exists
    result=$("$0" version-check 2>/dev/null) || true  # SUPPRESS:OK -- read-default: subcommand may fail
    echo "$now" > "$cache_file" 2>/dev/null || _aether_log_error "Could not update cache file $(basename "$cache_file")"
    if [[ -n "$result" ]]; then
      echo "$result"
    else
      json_ok '""'
    fi
    ;;

  registry-add)
    # Add or update a repo entry in ~/.aether/registry.json
    # Usage: registry-add <repo_path> <version> [--tags "a,b,c"] [--goal "text"] [--active true|false]
    repo_path="${1:-}"
    repo_version="${2:-}"
    [[ -z "$repo_path" || -z "$repo_version" ]] && json_err "$E_VALIDATION_FAILED" "Usage: registry-add <repo_path> <version> [--tags \"a,b\"] [--goal \"text\"] [--active true|false]"

    # Parse optional flags after positional args
    shift 2
    ra_tags=""
    ra_goal=""
    ra_active="false"
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --tags)  ra_tags="${2:-}"; shift 2 ;;
        --goal)  ra_goal="${2:-}"; shift 2 ;;
        --active) ra_active="${2:-false}"; shift 2 ;;
        *) shift ;;
      esac
    done

    # Convert comma-separated tags to JSON array
    ra_tags_json="[]"
    if [[ -n "$ra_tags" ]]; then
      ra_tags_json=$(echo "$ra_tags" | jq -R 'split(",") | map(select(length > 0))') || ra_tags_json="[]"
    fi

    # Normalize active to boolean string
    if [[ "$ra_active" != "true" ]]; then
      ra_active="false"
    fi

    registry_file="$HOME/.aether/registry.json"
    mkdir -p "$HOME/.aether"

    if [[ ! -f "$registry_file" ]]; then
      atomic_write "$registry_file" '{"schema_version":1,"repos":[]}' || json_err "$E_UNKNOWN" "Failed to initialize registry file"
    fi

    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Acquire lock to prevent concurrent read-modify-write races
    acquire_lock "$registry_file" || {
      _aether_log_error "Could not lock the registry file"
      json_err "$E_LOCK_FAILED" "Cannot update registry without a lock"
    }

    # Check if repo already exists in registry
    # SUPPRESS:OK -- read-default: query may return empty
    existing=$(jq --arg path "$repo_path" '.repos[] | select(.path == $path)' "$registry_file" 2>/dev/null)

    if [[ -n "$existing" ]]; then
      # Update existing entry — merge new fields
      updated=$(jq \
        --arg path "$repo_path" \
        --arg ver "$repo_version" \
        --arg ts "$ts" \
        --argjson tags "$ra_tags_json" \
        --arg goal "$ra_goal" \
        --argjson active "$ra_active" '
        .repos = [.repos[] | if .path == $path then
          .version = $ver |
          .updated_at = $ts |
          .domain_tags = $tags |
          (if $goal != "" then .last_colony_goal = $goal else . end) |
          .active_colony = $active
        else . end]
      ' "$registry_file") || json_err "$E_JSON_INVALID" "Failed to update registry"
    else
      # Add new entry with all fields
      updated=$(jq \
        --arg path "$repo_path" \
        --arg ver "$repo_version" \
        --arg ts "$ts" \
        --argjson tags "$ra_tags_json" \
        --arg goal "$ra_goal" \
        --argjson active "$ra_active" '
        .repos += [{
          "path": $path,
          "version": $ver,
          "registered_at": $ts,
          "updated_at": $ts,
          "domain_tags": $tags,
          "last_colony_goal": (if $goal != "" then $goal else null end),
          "active_colony": $active
        }]
      ' "$registry_file") || json_err "$E_JSON_INVALID" "Failed to update registry"
    fi

    atomic_write "$registry_file" "$updated" || {
      release_lock "$registry_file" 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
      _aether_log_error "Could not save registry update"
      json_err "$E_UNKNOWN" "Failed to write registry file"
    }
    release_lock "$registry_file" 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
    json_ok "$(jq -nc --arg p "$repo_path" --arg v "$repo_version" '{registered:true,path:$p,version:$v}')"
    ;;

  registry-list)
    # List all registered repos with domain tags, goal, and active status
    # Usage: registry-list
    # Returns JSON with all repos and their metadata (defaults for missing fields)
    registry_file="$HOME/.aether/registry.json"

    if [[ ! -f "$registry_file" ]]; then
      json_ok '{"repos":[],"count":0}'
    else
      # Normalize legacy entries: default domain_tags=[], last_colony_goal=null, active_colony=false
      rl_result=$(jq '
        {
          "repos": [.repos[] | {
            "path": .path,
            "version": .version,
            "registered_at": .registered_at,
            "updated_at": .updated_at,
            "domain_tags": (if .domain_tags then .domain_tags else [] end),
            "last_colony_goal": (if .last_colony_goal then .last_colony_goal else null end),
            "active_colony": (if .active_colony then .active_colony else false end)
          }],
          "count": (.repos | length)
        }
      ' "$registry_file" 2>/dev/null) || json_err "$E_JSON_INVALID" "Failed to read registry"  # SUPPRESS:OK -- read-default: file may not exist yet
      json_ok "$rl_result"
    fi
    ;;

  bootstrap-system)
    # Copy system files from ~/.aether/system/ into local .aether/
    # Uses explicit allowlist — never touches colony data
    hub_system="$HOME/.aether/system"
    local_aether="$AETHER_ROOT/.aether"

    [[ ! -d "$hub_system" ]] && json_err "$E_HUB_NOT_FOUND" "Hub system directory not found: $hub_system"

    # Allowlist of system files to copy (relative to system/)
    # Keep this in sync with active runtime dependencies and docs layout.
    allowlist=(
      # Core runtime
      "aether-utils.sh"
      "workers.md"
      # Docs (active)
      "docs/README.md"
      "docs/QUEEN-SYSTEM.md"
      "docs/queen-commands.md"
      "docs/caste-system.md"
      "docs/error-codes.md"
      "docs/known-issues.md"
      "docs/pheromones.md"
      "docs/source-of-truth-map.md"
      "docs/xml-utilities.md"

      # Disciplines
      "docs/disciplines/DISCIPLINES.md"
      "docs/disciplines/coding-standards.md"
      "docs/disciplines/debugging.md"
      "docs/disciplines/learning.md"
      "docs/disciplines/tdd.md"
      "docs/disciplines/verification-loop.md"
      "docs/disciplines/verification.md"

      # Build/continue playbooks (required by orchestrators)
      "docs/command-playbooks/README.md"
      "docs/command-playbooks/build-prep.md"
      "docs/command-playbooks/build-context.md"
      "docs/command-playbooks/build-wave.md"
      "docs/command-playbooks/build-verify.md"
      "docs/command-playbooks/build-complete.md"
      "docs/command-playbooks/continue-verify.md"
      "docs/command-playbooks/continue-gates.md"
      "docs/command-playbooks/continue-advance.md"
      "docs/command-playbooks/continue-finalize.md"
      "docs/INCIDENT_TEMPLATE.md"

      # Templates used by runtime generation/bootstrap flows
      "templates/QUEEN.md.template"
      "templates/colony-state.template.json"
      "templates/constraints.template.json"
      "templates/pheromones.template.json"
      "templates/handoff.template.md"
      "templates/handoff-build-success.template.md"
      "templates/handoff-build-error.template.md"
      "templates/session.template.json"
      "templates/learning-observations.template.json"
      "templates/midden.template.json"
      "templates/crowned-anthill.template.md"
      "templates/colony-state-reset.jq.template"
      "scripts/weekly-audit.sh"
      "scripts/incident-test-add.sh"

      # Core utilities
      "utils/atomic-write.sh"
      "utils/chamber-utils.sh"
      "utils/colorize-log.sh"
      "utils/error-handler.sh"
      "utils/file-lock.sh"
      "utils/state-loader.sh"
      "utils/watch-spawn-tree.sh"

      # XML utilities and schemas (seal/entomb/tunnels support)
      "utils/xml-utils.sh"
      "utils/xml-core.sh"
      "utils/xml-compose.sh"
      "utils/xml-convert.sh"
      "utils/xml-query.sh"
      "exchange/pheromone-xml.sh"
      "exchange/wisdom-xml.sh"
      "exchange/registry-xml.sh"
      "schemas/pheromone.xsd"
      "schemas/queen-wisdom.xsd"
      "schemas/colony-registry.xsd"
      "schemas/aether-types.xsd"
    )

    copied=0
    for file in "${allowlist[@]}"; do
      src="$hub_system/$file"
      dest="$local_aether/$file"
      if [[ -f "$src" ]]; then
        mkdir -p "$(dirname "$dest")"
        cp "$src" "$dest"
        # Preserve executable bit for shell scripts
        if [[ "$file" == *.sh ]]; then
          chmod 755 "$dest"
        fi
        copied=$((copied + 1))
      fi
    done

    json_ok "{\"copied\":$copied,\"total\":${#allowlist[@]}}"
    ;;

  load-state)
    source "$SCRIPT_DIR/utils/state-loader.sh" 2>/dev/null || {  # SUPPRESS:OK -- read-default: utility may not be installed
      json_err "$E_FILE_NOT_FOUND" "state-loader.sh not found"
      exit 1
    }
    load_colony_state
    if [[ $? -eq 0 ]]; then
      # Output success with handoff info if detected
      if [[ "$HANDOFF_DETECTED" == "true" ]]; then
        _handoff_summary=$(get_handoff_summary)
        json_ok "$(jq -nc --arg hs "$_handoff_summary" '{loaded:true,handoff_detected:true,handoff_summary:$hs}')"
      else
        json_ok '{"loaded":true}'
      fi
    fi
    # Note: load_colony_state handles its own error output
    ;;

  unload-state)
    source "$SCRIPT_DIR/utils/state-loader.sh" 2>/dev/null || {  # SUPPRESS:OK -- read-default: utility may not be installed
      json_err "$E_FILE_NOT_FOUND" "state-loader.sh not found"
      exit 1
    }
    unload_colony_state
    json_ok '{"unloaded":true}'
    ;;

  spawn-tree-load) _spawn_tree_load "$@" ;;
  spawn-tree-active) _spawn_tree_active "$@" ;;
  spawn-tree-depth) _spawn_tree_depth "$@" ;;
  spawn-efficiency) _spawn_efficiency "$@" ;;

  # ============================================
  # CHAMBER UTILITIES (colony lifecycle)
  # ============================================

  chamber-create)
    # Create a new chamber (entomb a colony)
    # Usage: chamber-create <chamber_dir> <state_file> <goal> <phases_completed> <total_phases> <milestone> <version> <decisions_json> <learnings_json>
    [[ $# -ge 9 ]] || json_err "$E_VALIDATION_FAILED" "Usage: chamber-create <chamber_dir> <state_file> <goal> <phases_completed> <total_phases> <milestone> <version> <decisions_json> <learnings_json>"

    # Check if chamber-utils.sh is available
    if ! type chamber_create &>/dev/null; then
      json_err "$E_FILE_NOT_FOUND" "chamber-utils.sh not loaded"
    fi

    chamber_create "$1" "$2" "$3" "$4" "$5" "$6" "$7" "$8" "$9"
    ;;

  chamber-verify)
    # Verify chamber integrity
    # Usage: chamber-verify <chamber_dir>
    [[ $# -ge 1 ]] || json_err "$E_VALIDATION_FAILED" "Usage: chamber-verify <chamber_dir>"

    if ! type chamber_verify &>/dev/null; then
      json_err "$E_FILE_NOT_FOUND" "chamber-utils.sh not loaded"
    fi

    chamber_verify "$1"
    ;;

  chamber-list)
    # List all chambers
    # Usage: chamber-list [chambers_root]
    chambers_root="${1:-$AETHER_ROOT/.aether/chambers}"

    if ! type chamber_list &>/dev/null; then
      json_err "$E_FILE_NOT_FOUND" "chamber-utils.sh not loaded"
    fi

    chamber_list "$chambers_root"
    ;;

  milestone-detect)
    # Migrated to state-api facade: uses _state_read_field for read-only access
    # Detect colony milestone from state
    # Usage: milestone-detect
    # Returns: {ok: true, milestone: "...", version: "...", phases_completed: N, total_phases: N, progress_percent: N}

    md_state=$(_state_read_field '.')
    [[ -n "$md_state" ]] || json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found" '{"file":"COLONY_STATE.json"}'

    # Extract and compute milestone data using jq
    result=$(echo "$md_state" | jq '
      # Extract key data
      (.plan.phases // []) as $phases |
      (.errors.records // []) as $errors |
      (.milestone // null) as $stored_milestone |

      # Count completed phases
      ([$phases[] | select(.status == "completed")] | length) as $completed_count |
      ($phases | length) as $total_phases |

      # Check for critical errors
      ([$errors[] | select(.severity == "critical")] | length) as $critical_count |

      # Determine milestone based on state
      if $critical_count > 0 then
        "Failed Mound"
      elif $total_phases > 0 and $completed_count == $total_phases then
        if $stored_milestone == "Crowned Anthill" then
          "Crowned Anthill"
        else
          "Sealed Chambers"
        end
      elif $completed_count >= 5 then
        "Ventilated Nest"
      elif $completed_count >= 3 then
        "Brood Stable"
      elif $completed_count >= 1 then
        "Open Chambers"
      else
        "First Mound"
      end as $milestone |

      # Compute version: major = floor(total_phases / 10), minor = total_phases % 10, patch = completed_count
      ($total_phases / 10 | floor) as $major |
      ($total_phases % 10) as $minor |
      $completed_count as $patch |
      "v\($major).\($minor).\($patch)" as $version |

      # Calculate progress percentage
      (if $total_phases > 0 then ($completed_count * 100 / $total_phases | round) else 0 end) as $progress |

      # Return result
      {
        ok: true,
        milestone: $milestone,
        version: $version,
        phases_completed: $completed_count,
        total_phases: $total_phases,
        progress_percent: $progress
      }
    ')

    echo "$result"
    ;;

  phase-insert)
    # Migrated to state-api facade: uses _state_read_field for reads, _state_mutate for atomic write
    # Insert a new phase immediately after current_phase and renumber downstream phases safely.
    # Usage: phase-insert <phase_name> <goal> [constraints]
    phase_name="${1:-}"
    phase_goal="${2:-}"
    phase_constraints="${3:-}"

    [[ -n "$phase_name" ]] || json_err "$E_VALIDATION_FAILED" "Usage: phase-insert <phase_name> <goal> [constraints]" '{"missing":"phase_name"}'
    [[ -n "$phase_goal" ]] || json_err "$E_VALIDATION_FAILED" "Usage: phase-insert <phase_name> <goal> [constraints]" '{"missing":"goal"}'

    # Read phase count and current_phase via state-api facade
    pi_phase_count=$(_state_read_field '(.plan.phases // []) | length')
    [[ -n "$pi_phase_count" && "$pi_phase_count" -gt 0 ]] 2>/dev/null || json_err "$E_VALIDATION_FAILED" "No project plan found. Run /ant:plan first."

    pi_current_phase=$(_state_read_field '.current_phase // 0')
    [[ "$pi_current_phase" =~ ^[0-9]+$ ]] || pi_current_phase=0
    if [[ "$pi_current_phase" -gt "$pi_phase_count" ]]; then
      pi_current_phase="$pi_phase_count"
    fi
    if [[ "$pi_current_phase" -lt 0 ]]; then
      pi_current_phase=0
    fi

    insert_id=$((pi_current_phase + 1))
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Use _state_mutate for atomic read-modify-write (handles locking and backup)
    PI_INSERT_ID="$insert_id" PI_NAME="$phase_name" PI_GOAL="$phase_goal" \
    PI_CONSTRAINTS="$phase_constraints" PI_TS="$ts" \
      _state_mutate '
      (env.PI_INSERT_ID | tonumber) as $insert_id |
      env.PI_NAME as $name |
      env.PI_GOAL as $goal |
      env.PI_CONSTRAINTS as $constraints |
      env.PI_TS as $ts |

      def tail_id:
        if (type == "string") and test("^[0-9]+\\.") then
          capture("^[0-9]+\\.(?<tail>.+)$").tail
        else
          tostring
        end;
      def remap_dep(new_phase):
        if (type == "string") and test("^[0-9]+\\.") then
          ((new_phase|tostring) + "." + (capture("^[0-9]+\\.(?<tail>.+)$").tail))
        else
          .
        end;
      def remap_task(new_phase):
        . as $task
        | (if ($task.id // null) != null then
            .id = (
              if ($task.id|type) == "string" and ($task.id|test("^[0-9]+\\.")) then
                ((new_phase|tostring) + "." + ($task.id|capture("^[0-9]+\\.(?<tail>.+)$").tail))
              else
                $task.id
              end
            )
          else . end)
        | (if (.dependencies|type) == "array" then .dependencies |= map(remap_dep(new_phase)) else . end)
        | (if (.depends_on|type) == "array" then .depends_on |= map(remap_dep(new_phase)) else . end);
      def normalize_phase:
        if .status == null then .status = "pending" else . end
        | if .tasks == null then .tasks = [] else . end
        | if .success_criteria == null then .success_criteria = [] else . end;
      def shift_phase(insert_id):
        if (.id // 0) >= insert_id then
          (.id + 1) as $new_phase
          | .id = $new_phase
          | .tasks = ((.tasks // []) | map(remap_task($new_phase)))
        else
          .
        end;

      . as $root
      | ($root.plan.phases // []) as $phases
      | ($phases | map(normalize_phase | shift_phase($insert_id))) as $shifted
      | ($insert_id|tostring) as $pid
      | {
          id: $insert_id,
          name: $name,
          description: $goal,
          status: "pending",
          tasks: [
            {
              id: ($pid + ".1"),
              description: ("Diagnose and correct: " + $goal),
              success_criteria: [
                "Root cause identified with evidence",
                "Fix implemented in targeted area"
              ],
              status: "pending"
            },
            {
              id: ($pid + ".2"),
              description: "Validate the correction end-to-end",
              success_criteria: [
                "Previously failing behavior now works",
                "No regressions introduced in adjacent flows"
              ],
              status: "pending"
            }
          ],
          success_criteria: [
            "Inserted-phase objective is resolved in real usage",
            "User confirms expected behavior after changes"
          ]
        } as $new_phase
      | .plan.phases = (($shifted + [$new_phase]) | sort_by(.id))
      | .events = ((.events // []) + [($ts + "|phase_inserted|insert-phase|Inserted Phase " + ($insert_id|tostring) + ": " + $name)])
      ' >/dev/null

    # Emit guidance signals non-blocking to reinforce inserted phase intent.
    bash "$0" pheromone-write FOCUS "Inserted Phase $insert_id: $phase_goal" --strength 0.8 --source "user:insert-phase" --reason "Phase inserted to correct execution path" --ttl "30d" >/dev/null 2>&1 \
      || _aether_log_error "Could not emit FOCUS signal for inserted phase"
    if [[ -n "$phase_constraints" ]]; then
      bash "$0" pheromone-write REDIRECT "$phase_constraints" --strength 0.9 --source "user:insert-phase" --reason "Constraint captured during phase insertion" --ttl "30d" >/dev/null 2>&1 \
        || _aether_log_error "Could not emit REDIRECT signal for phase constraints"
    fi
    bash "$0" memory-capture "learning" "Inserted phase $insert_id ($phase_name): $phase_goal" "pattern" "system:phase-insert" >/dev/null 2>&1 \
      || _aether_log_error "Could not capture learning for phase insertion"

    result=$(jq -n \
      --argjson inserted_phase_id "$insert_id" \
      --arg phase_name "$phase_name" \
      --arg phase_goal "$phase_goal" \
      --arg constraints "$phase_constraints" \
      --argjson after_phase "$pi_current_phase" \
      '{
        inserted: true,
        inserted_phase_id: $inserted_phase_id,
        after_phase: $after_phase,
        phase_name: $phase_name,
        phase_goal: $phase_goal,
        constraints: $constraints
      }')
    json_ok "$result"
    ;;

  # ============================================
  # SWARM ACTIVITY TRACKING (colony visualization) — dispatched to utils/swarm.sh
  # ============================================

  swarm-activity-log) _swarm_activity_log "$@" ;;

  swarm-display-init) _swarm_display_init "$@" ;;

  swarm-display-update) _swarm_display_update "$@" ;;
  swarm-display-get) _swarm_display_get "$@" ;;
  swarm-display-render) _swarm_display_render "$@" ;;

  swarm-display-inline) _swarm_display_inline "$@" ;;


  swarm-display-text) _swarm_display_text "$@" ;;

  swarm-timing-start) _swarm_timing_start "$@" ;;
  swarm-timing-get) _swarm_timing_get "$@" ;;
  swarm-timing-eta) _swarm_timing_eta "$@" ;;

  # ============================================
  # VIEW STATE MANAGEMENT (collapsible views)
  # ============================================

  view-state-init)
    # Initialize view state file with default structure
    # Usage: view-state-init
    mkdir -p "$COLONY_DATA_DIR"
    view_state_file="$COLONY_DATA_DIR/view-state.json"

    if [[ ! -f "$view_state_file" ]]; then
      atomic_write "$view_state_file" '{
  "version": "1.0",
  "swarm_display": {
    "expanded": [],
    "collapsed": [],
    "default_expand_depth": 2
  },
  "tunnel_view": {
    "expanded": [],
    "collapsed": ["__depth_3_plus__"],
    "default_expand_depth": 2,
    "show_completed": true
  }
}'
      json_ok '{"initialized":true,"file":"view-state.json"}'
    else
      json_ok '{"initialized":false,"file":"view-state.json","exists":true}'
    fi
    ;;

  view-state-get)
    # Get view state or specific key
    # Usage: view-state-get [view_name] [key]
    view_name="${1:-}"
    key="${2:-}"
    view_state_file="$COLONY_DATA_DIR/view-state.json"

    if [[ ! -f "$view_state_file" ]]; then
      # Auto-initialize if not exists
      bash "$0" view-state-init >/dev/null 2>&1 || _aether_log_error "Could not initialize view state display"
    fi

    if [[ -z "$view_name" ]]; then
      # Return entire state
      json_ok "$(cat "$view_state_file")"
    elif [[ -z "$key" ]]; then
      # Return specific view
      json_ok "$(jq --arg v "$view_name" '.[$v] // {}' "$view_state_file")"
    else
      # Return specific key from view
      json_ok "$(jq --arg v "$view_name" --arg k "$key" '.[$v][$k] // null' "$view_state_file")"
    fi
    ;;

  view-state-set)
    # Set a specific key in a view
    # Usage: view-state-set <view_name> <key> <value>
    view_name="${1:-}"
    key="${2:-}"
    value="${3:-}"
    [[ -z "$view_name" || -z "$key" ]] && json_err "$E_VALIDATION_FAILED" "Usage: view-state-set <view_name> <key> <value>"

    view_state_file="$COLONY_DATA_DIR/view-state.json"

    if [[ ! -f "$view_state_file" ]]; then
      bash "$0" view-state-init >/dev/null 2>&1 || _aether_log_error "Could not initialize view state display"
    fi

    # Determine if value is JSON or string
    if [[ "$value" =~ ^\[.*\]$ ]] || [[ "$value" =~ ^\{.*\}$ ]] || [[ "$value" =~ ^(true|false|null|[0-9]+)$ ]]; then
      # Value appears to be JSON - use as-is
      updated=$(jq --arg view "$view_name" --arg key "$key" --argjson val "$value" '
        .[$view][$key] = $val
      ' "$view_state_file") || json_err "$E_JSON_INVALID" "Failed to update view state"
    else
      # Treat as string
      updated=$(jq --arg view "$view_name" --arg key "$key" --arg val "$value" '
        .[$view][$key] = $val
      ' "$view_state_file") || json_err "$E_JSON_INVALID" "Failed to update view state"
    fi

    atomic_write "$view_state_file" "$updated"
    json_ok "$(echo "$updated" | jq --arg v "$view_name" '.[$v]')"
    ;;

  view-state-toggle)
    # Toggle item between expanded and collapsed
    # Usage: view-state-toggle <view_name> <item>
    view_name="${1:-}"
    item="${2:-}"
    [[ -z "$view_name" || -z "$item" ]] && json_err "$E_VALIDATION_FAILED" "Usage: view-state-toggle <view_name> <item>"

    view_state_file="$COLONY_DATA_DIR/view-state.json"

    if [[ ! -f "$view_state_file" ]]; then
      bash "$0" view-state-init >/dev/null 2>&1 || _aether_log_error "Could not initialize view state display"
    fi

    # Check current state
    is_expanded=$(jq --arg view "$view_name" --arg item "$item" '
      .[$view].expanded | contains([$item])
    ' "$view_state_file")

    if [[ "$is_expanded" == "true" ]]; then
      # Move from expanded to collapsed
      updated=$(jq --arg view "$view_name" --arg item "$item" '
        .[$view].expanded -= [$item] |
        .[$view].collapsed += [$item]
      ' "$view_state_file")
      new_state="collapsed"
    else
      # Move from collapsed to expanded
      updated=$(jq --arg view "$view_name" --arg item "$item" '
        .[$view].collapsed -= [$item] |
        .[$view].expanded += [$item]
      ' "$view_state_file")
      new_state="expanded"
    fi

    atomic_write "$view_state_file" "$updated"
    json_ok "$(jq -nc --arg i "$item" --arg s "$new_state" --arg v "$view_name" '{item:$i,state:$s,view:$v}')"
    ;;

  view-state-expand)
    # Explicitly expand an item
    # Usage: view-state-expand <view_name> <item>
    view_name="${1:-}"
    item="${2:-}"
    [[ -z "$view_name" || -z "$item" ]] && json_err "$E_VALIDATION_FAILED" "Usage: view-state-expand <view_name> <item>"

    view_state_file="$COLONY_DATA_DIR/view-state.json"

    if [[ ! -f "$view_state_file" ]]; then
      bash "$0" view-state-init >/dev/null 2>&1 || _aether_log_error "Could not initialize view state display"
    fi

    updated=$(jq --arg view "$view_name" --arg item "$item" '
      .[$view].collapsed -= [$item] |
      .[$view].expanded += [$item]
    ' "$view_state_file") || json_err "$E_JSON_INVALID" "Failed to update view state"

    atomic_write "$view_state_file" "$updated"
    json_ok "$(jq -nc --arg i "$item" --arg v "$view_name" '{item:$i,state:"expanded",view:$v}')"
    ;;

  view-state-collapse)
    # Explicitly collapse an item
    # Usage: view-state-collapse <view_name> <item>
    view_name="${1:-}"
    item="${2:-}"
    [[ -z "$view_name" || -z "$item" ]] && json_err "$E_VALIDATION_FAILED" "Usage: view-state-collapse <view_name> <item>"

    view_state_file="$COLONY_DATA_DIR/view-state.json"

    if [[ ! -f "$view_state_file" ]]; then
      bash "$0" view-state-init >/dev/null 2>&1 || _aether_log_error "Could not initialize view state display"
    fi

    updated=$(jq --arg view "$view_name" --arg item "$item" '
      .[$view].expanded -= [$item] |
      .[$view].collapsed += [$item]
    ' "$view_state_file") || json_err "$E_JSON_INVALID" "Failed to update view state"

    atomic_write "$view_state_file" "$updated"
    json_ok "$(jq -nc --arg i "$item" --arg v "$view_name" '{item:$i,state:"collapsed",view:$v}')"
    ;;

  queen-init) _queen_init "$@" ;;

  queen-read) _queen_read "$@" ;;

  generate-threshold-bar)
    # Generate visual threshold progress bar
    # Usage: generate-threshold-bar <observation_count> <threshold>
    # Returns: Visual bar like "●●●○○ (3/5)" or "[=--] (1/3)" for ASCII
    obs_count="${1:-0}"
    threshold="${2:-1}"

    # Validate inputs are numbers
    if ! [[ "$obs_count" =~ ^[0-9]+$ ]]; then
      json_err "$E_VALIDATION_FAILED" "observation_count must be a number" "{\"provided\":\"$obs_count\"}"
    fi
    if ! [[ "$threshold" =~ ^[0-9]+$ ]]; then
      json_err "$E_VALIDATION_FAILED" "threshold must be a number" "{\"provided\":\"$threshold\"}"
    fi

    # Handle threshold = 0 (immediate promotion)
    if [[ "$threshold" -eq 0 ]]; then
      json_ok "{\"bar\":\"immediate\",\"count\":$obs_count,\"threshold\":0}"
      exit 0
    fi

    # Detect UTF-8 support
    use_utf8=false
    if [[ "${LANG:-}" =~ UTF-8 ]] || [[ "${LC_ALL:-}" =~ UTF-8 ]]; then
      use_utf8=true
    fi

    # Build the bar
    bar=""
    filled_char=""
    empty_char=""

    if [[ "$use_utf8" == "true" ]]; then
      filled_char="●"
      empty_char="○"
    else
      filled_char="="
      empty_char="-"
    fi

    # Cap count at threshold for display (don't show overfill)
    display_count=$obs_count
    if [[ "$display_count" -gt "$threshold" ]]; then
      display_count=$threshold
    fi

    # Build bar characters
    for ((i=0; i<threshold; i++)); do
      if [[ $i -lt $display_count ]]; then
        bar+="$filled_char"
      else
        bar+="$empty_char"
      fi
    done

    # For ASCII mode, wrap in brackets
    if [[ "$use_utf8" == "false" ]]; then
      bar="[$bar]"
    fi

    json_ok "$(jq -nc --arg b "$bar" --argjson c "$obs_count" --argjson t "$threshold" '{bar:$b,count:$c,threshold:$t}')"
    ;;

  parse-selection)
    # Parse user selection string into validated 0-indexed indices
    # Usage: parse-selection <input_string> <max_index>
    # Returns: JSON with selected indices, deferred count, and validation warnings
    #
    # Examples:
    #   parse-selection "1 3 5" 10  -> {"selected":[0,2,4],"deferred":[],"valid":true}
    #   parse-selection "" 5        -> {"selected":[],"deferred":[0,1,2,3,4],"action":"defer_all"}
    #   parse-selection "1 2 99" 5  -> {"selected":[0,1],"warnings":["99 out of range"]}
    input_string="${1:-}"
    max_index="${2:-0}"

    # Validate max_index is a number
    if ! [[ "$max_index" =~ ^[0-9]+$ ]]; then
      json_err "$E_VALIDATION_FAILED" "max_index must be a number" "{\"provided\":\"$max_index\"}"
    fi

    # Empty input signals defer-all
    if [[ -z "$input_string" ]]; then
      # Build deferred array with all indices
      deferred_array=""
      for ((i=0; i<max_index; i++)); do
        [[ -n "$deferred_array" ]] && deferred_array+=","
        deferred_array+="$i"
      done
      json_ok "{\"selected\":[],\"deferred\":[$deferred_array],\"count\":$max_index,\"action\":\"defer_all\"}"
      exit 0
    fi

    # Normalize input: remove extra spaces, keep only digits and spaces
    normalized=$(echo "$input_string" | tr -s ' ' | tr -cd '0-9 ')

    # Parse and validate each number
    declare -a selected_indices
    declare -a warnings
    seen_list=""  # Space-separated list for deduplication (bash 3.x compatible)

    for num in $normalized; do
      # Skip empty
      [[ -z "$num" ]] && continue

      # Validate range (1-indexed input)
      if [[ "$num" -lt 1 || "$num" -gt "$max_index" ]]; then
        warnings+=("\"$num out of range (1-$max_index)\"")
        continue
      fi

      # Deduplicate using string pattern matching (bash 3.x compatible)
      if [[ "$seen_list" == *" $num "* ]]; then
        continue
      fi
      seen_list="$seen_list $num "

      # Convert to 0-indexed and add to result
      idx=$((num - 1))
      selected_indices+=("$idx")
    done

    # Build selected array JSON
    selected_json=""
    for idx in "${selected_indices[@]}"; do
      [[ -n "$selected_json" ]] && selected_json+=","
      selected_json+="$idx"
    done

    # Build deferred array (all indices not selected)
    deferred_json=""
    for ((i=0; i<max_index; i++)); do
      # Check if i is in selected_indices
      is_selected=false
      for sel in "${selected_indices[@]}"; do
        if [[ "$sel" -eq "$i" ]]; then
          is_selected=true
          break
        fi
      done

      if [[ "$is_selected" == "false" ]]; then
        [[ -n "$deferred_json" ]] && deferred_json+=","
        deferred_json+="$i"
      fi
    done

    # Build warnings array JSON
    warnings_json=""
    if [[ ${#warnings[@]} -gt 0 ]]; then
      for w in "${warnings[@]}"; do
        [[ -n "$warnings_json" ]] && warnings_json+=","
        warnings_json+="$w"
      done
    fi

    # Construct result
    result="{\"selected\":[$selected_json],\"deferred\":[$deferred_json],\"count\":$max_index,\"valid\":true}"

    # Add warnings if any
    if [[ -n "$warnings_json" ]]; then
      result=$(echo "$result" | jq --argjson w "[$warnings_json]" '. + {warnings: $w}')
    fi

    json_ok "$result"
    ;;

  queen-thresholds) _queen_thresholds "$@" ;;

  incident-rule-add)
    # Append an incident-derived rule to long-term guidance stores.
    # Usage: incident-rule-add <incident_id> <rule_text> [decree|constraint|gate]
    ir_incident_id="${1:-}"
    ir_rule_text="${2:-}"
    ir_rule_type="${3:-decree}"

    [[ -z "$ir_incident_id" ]] && json_err "$E_VALIDATION_FAILED" "Usage: incident-rule-add <incident_id> <rule_text> [decree|constraint|gate]" '{"missing":"incident_id"}'
    [[ -z "$ir_rule_text" ]] && json_err "$E_VALIDATION_FAILED" "Usage: incident-rule-add <incident_id> <rule_text> [decree|constraint|gate]" '{"missing":"rule_text"}'

    ir_ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    case "$ir_rule_type" in
      decree)
        ir_queen_file="$AETHER_ROOT/.aether/QUEEN.md"
        [[ -f "$ir_queen_file" ]] || json_err "$E_FILE_NOT_FOUND" "QUEEN.md not found" '{"path":".aether/QUEEN.md"}'

        ir_entry="- [${ir_incident_id}] ${ir_rule_text}"
        ir_tmp="${ir_queen_file}.tmp.$$"
        awk -v header="## 🏛️ Decrees" -v entry="$ir_entry" '
          $0 == header && inserted == 0 {print; print ""; print entry; inserted=1; next}
          {print}
          END {
            if (inserted == 0) {
              print "";
              print "## 🏛️ Decrees";
              print "";
              print entry;
            }
          }
        ' "$ir_queen_file" > "$ir_tmp"

        ir_lock_held=false
        if type acquire_lock &>/dev/null; then
          acquire_lock "$ir_queen_file" || {
            rm -f "$ir_tmp" 2>/dev/null || true  # SUPPRESS:OK -- cleanup: file may not exist
            json_err "$E_LOCK_FAILED" "Failed to acquire lock on QUEEN.md"
          }
          ir_lock_held=true
        fi

        ir_content=$(cat "$ir_tmp" 2>/dev/null || echo "")  # SUPPRESS:OK -- read-default: file may not exist yet
        atomic_write "$ir_queen_file" "$ir_content" || {
          rm -f "$ir_tmp" 2>/dev/null || true  # SUPPRESS:OK -- cleanup: file may not exist
          [[ "$ir_lock_held" == "true" ]] && release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
          json_err "$E_FILE_NOT_FOUND" "Failed to append decree rule to QUEEN.md"
        }
        rm -f "$ir_tmp" 2>/dev/null || true  # SUPPRESS:OK -- cleanup: file may not exist
        [[ "$ir_lock_held" == "true" ]] && release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
        ;;

      constraint)
        ir_constraints_file="$COLONY_DATA_DIR/constraints.json"
        if [[ ! -f "$ir_constraints_file" ]]; then
          atomic_write "$ir_constraints_file" '{"version":"1.0","focus":[],"constraints":[]}' || json_err "$E_UNKNOWN" "Failed to initialize constraints file"
        fi

        ir_lock_held=false
        if type acquire_lock &>/dev/null; then
          acquire_lock "$ir_constraints_file" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on constraints.json"
          ir_lock_held=true
        fi

        ir_source="incident:${ir_incident_id}"
        ir_updated=$(jq --arg id "$ir_incident_id" --arg rule "$ir_rule_text" --arg source "$ir_source" --arg ts "$ir_ts" '
          .constraints += [{
            id: $id,
            type: "INCIDENT_RULE",
            content: $rule,
            source: $source,
            created_at: $ts
          }]
        ' "$ir_constraints_file" 2>/dev/null) || {  # SUPPRESS:OK -- read-default: file may not exist yet
          [[ "$ir_lock_held" == "true" ]] && release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
          json_err "$E_JSON_INVALID" "Failed to update constraints.json"
        }

        atomic_write "$ir_constraints_file" "$ir_updated" || {
          [[ "$ir_lock_held" == "true" ]] && release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
          json_err "$E_JSON_INVALID" "Failed to write constraints.json"
        }
        [[ "$ir_lock_held" == "true" ]] && release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
        ;;

      gate)
        ir_gate_file="$AETHER_ROOT/.aether/docs/verification-gates.md"
        mkdir -p "$(dirname "$ir_gate_file")"
        if [[ ! -f "$ir_gate_file" ]]; then
          printf '%s\n' "# Verification Gates" > "$ir_gate_file"
          printf '%s\n' "" >> "$ir_gate_file"
        fi
        printf '%s\n' "- [${ir_incident_id}] ${ir_rule_text}" >> "$ir_gate_file"
        ;;

      *)
        json_err "$E_VALIDATION_FAILED" "Invalid rule_type: $ir_rule_type. Use decree|constraint|gate"
        ;;
    esac

    json_ok "$(jq -nc --arg id "$ir_incident_id" --arg rt "$ir_rule_type" --arg ts "$ir_ts" '{incident_id:$id,rule_type:$rt,added:true,timestamp:$ts}')"
    ;;

  queen-promote) _queen_promote "$@" ;;
  queen-write-learnings) _queen_write_learnings "$@" ;;
  queen-promote-instinct) _queen_promote_instinct "$@" ;;
  queen-seed-from-hive) _queen_seed_from_hive "$@" ;;
  queen-migrate) _queen_migrate "$@" ;;
  charter-write) _queen_write_charter "$@" ;;
  colony-name) _colony_name "$@" ;;
  colony-depth) _colony_depth "$@" ;;
  domain-detect) _domain_detect "$@" ;;

  learning-observe) _learning_observe "$@" ;;
  learning-check-promotion) _learning_check_promotion "$@" ;;
  learning-promote-auto) _learning_promote_auto "$@" ;;

  memory-capture)
    # Capture learning/failure events with deterministic memory actions.
    # Usage: memory-capture <event_type> <content> [wisdom_type] [source]
    # event_type: learning|failure|redirect|feedback|success|resolution
    mc_event="${1:-}"
    mc_content="${2:-}"
    mc_wisdom_type="${3:-}"
    mc_source="${4:-system:memory-capture}"

    [[ -z "$mc_event" ]] && json_err "$E_VALIDATION_FAILED" "Usage: memory-capture <event_type> <content> [wisdom_type] [source]" '{"missing":"event_type"}'
    [[ -z "$mc_content" ]] && json_err "$E_VALIDATION_FAILED" "Usage: memory-capture <event_type> <content> [wisdom_type] [source]" '{"missing":"content"}'

    case "$mc_event" in
      learning|failure|redirect|feedback|success|resolution) ;;
      *) json_err "$E_VALIDATION_FAILED" "Invalid event_type: $mc_event" '{"valid_event_types":["learning","failure","redirect","feedback","success","resolution"]}' ;;
    esac

    if [[ -z "$mc_wisdom_type" ]]; then
      case "$mc_event" in
        failure) mc_wisdom_type="failure" ;;
        redirect) mc_wisdom_type="redirect" ;;
        feedback) mc_wisdom_type="pattern" ;;
        learning|success|resolution) mc_wisdom_type="pattern" ;;
      esac
    fi

    colony_name=$(bash "$0" colony-name 2>/dev/null | jq -r '.result.name // ""') || colony_name="unknown"
    [[ -z "$colony_name" ]] && colony_name="unknown"

    # SUPPRESS:OK -- read-default: returns fallback on failure
    observe_result=$(bash "$0" learning-observe "$mc_content" "$mc_wisdom_type" "$colony_name" 2>/dev/null || echo '{}')
    if ! echo "$observe_result" | jq -e '.ok == true' >/dev/null 2>&1; then  # SUPPRESS:OK -- validation: testing JSON field
      # SUPPRESS:OK -- read-default: query may return empty
      obs_msg=$(echo "$observe_result" | jq -r '.error.message // "learning_observe_failed"' 2>/dev/null || echo "learning_observe_failed")
      json_err "$E_VALIDATION_FAILED" "memory-capture failed at learning-observe: $obs_msg"
    fi

    obs_count=$(echo "$observe_result" | jq -r '.result.observation_count // 0' 2>/dev/null || echo "0")  # SUPPRESS:OK -- read-default: file may not exist yet
    obs_threshold=$(echo "$observe_result" | jq -r '.result.threshold // 1' 2>/dev/null || echo "1")  # SUPPRESS:OK -- read-default: file may not exist yet
    # SUPPRESS:OK -- read-default: query may return empty
    obs_threshold_met=$(echo "$observe_result" | jq -r '.result.threshold_met // false' 2>/dev/null || echo "false")

    pheromone_type=""
    pheromone_content=""
    pheromone_strength="0.6"
    pheromone_reason=""
    pheromone_ttl="30d"

    case "$mc_event" in
      failure)
        pheromone_type="REDIRECT"
        pheromone_content="Avoid repeating failure: $mc_content"
        pheromone_strength="0.7"
        pheromone_reason="Auto-emitted from failure event"
        ;;
      redirect)
        pheromone_type="REDIRECT"
        pheromone_content="$mc_content"
        pheromone_strength="0.7"
        pheromone_reason="Auto-emitted redirect guidance"
        ;;
      feedback)
        pheromone_type="FEEDBACK"
        pheromone_content="$mc_content"
        pheromone_strength="0.6"
        pheromone_reason="Auto-emitted feedback guidance"
        ;;
      learning|success)
        pheromone_type="FEEDBACK"
        pheromone_content="Learning captured: $mc_content"
        pheromone_strength="0.6"
        pheromone_reason="Auto-emitted from validated learning"
        ;;
      resolution)
        pheromone_type="FEEDBACK"
        pheromone_content="Resolved recurring issue: $mc_content"
        pheromone_strength="0.75"
        pheromone_reason="Auto-emitted from resolution event"
        ;;
    esac

    pheromone_created=false
    pheromone_signal_id=""
    if [[ -n "$pheromone_type" && -n "$pheromone_content" ]]; then
      # SUPPRESS:OK -- read-default: returns fallback on failure
      pheromone_result=$(bash "$0" pheromone-write "$pheromone_type" "$pheromone_content" --strength "$pheromone_strength" --source "$mc_source" --reason "$pheromone_reason" --ttl "$pheromone_ttl" 2>/dev/null || echo '{}')
      if echo "$pheromone_result" | jq -e '.ok == true' >/dev/null 2>&1; then  # SUPPRESS:OK -- validation: testing JSON field
        pheromone_created=true
        # SUPPRESS:OK -- read-default: query may return empty
        pheromone_signal_id=$(echo "$pheromone_result" | jq -r '.result.signal_id // ""' 2>/dev/null || echo "")
      fi
    fi

    # learning-promote-auto may emit multiple JSON lines (e.g. instinct-create output
    # followed by the actual result). Take only the last line as the authoritative result.
    # SUPPRESS:OK -- read-default: returns fallback on failure
    auto_result_raw=$(bash "$0" learning-promote-auto "$mc_wisdom_type" "$mc_content" "$colony_name" "$mc_event" 2>/dev/null || echo '{}')
    auto_result=$(echo "$auto_result_raw" | tail -1)
    auto_promoted=false
    auto_reason="promotion_skipped"
    if echo "$auto_result" | jq -e '.ok == true' >/dev/null 2>&1; then  # SUPPRESS:OK -- validation: testing JSON field
      # SUPPRESS:OK -- read-default: query may return empty
      auto_promoted=$(echo "$auto_result" | jq -r '.result.promoted // false' 2>/dev/null || echo "false")
      # SUPPRESS:OK -- read-default: query may return empty
      auto_reason=$(echo "$auto_result" | jq -r '.result.reason // "promoted"' 2>/dev/null || echo "unknown")
    fi

    bash "$0" activity-log "MEMORY" "system" "Captured $mc_event ($mc_wisdom_type): count=$obs_count auto_promoted=$auto_promoted" >/dev/null 2>&1 \
      || _aether_log_error "Could not log memory capture activity"
    bash "$0" rolling-summary add "$mc_event" "$mc_content" "$mc_source" >/dev/null 2>&1 \
      || _aether_log_error "Could not update rolling summary"

    json_ok "$(jq -nc --arg et "$mc_event" --arg wt "$mc_wisdom_type" --argjson oc "$obs_count" --argjson th "$obs_threshold" --argjson tm "$obs_threshold_met" --argjson pc "$pheromone_created" --arg sid "$pheromone_signal_id" --argjson ap "$auto_promoted" --arg pr "$auto_reason" '{event_type:$et,wisdom_type:$wt,observation_count:$oc,threshold:$th,threshold_met:$tm,pheromone_created:$pc,signal_id:$sid,auto_promoted:$ap,promotion_reason:$pr}')"
    ;;

  learning-display-proposals) _learning_display_proposals "$@" ;;
  learning-select-proposals) _learning_select_proposals "$@" ;;
  learning-defer-proposals) _learning_defer_proposals "$@" ;;
  learning-approve-proposals) _learning_approve_proposals "$@" ;;
  learning-undo-promotions) _learning_undo_promotions "$@" ;;

  survey-load)
    phase_type="${1:-}"
    survey_dir=".aether/data/survey"

    if [[ ! -d "$survey_dir" ]]; then
      json_err "$E_FILE_NOT_FOUND" "No survey found"
    fi

    docs=""
    case "$phase_type" in
      *frontend*|*component*|*UI*|*page*|*button*)
        docs="DISCIPLINES.md,CHAMBERS.md"
        ;;
      *API*|*endpoint*|*backend*|*route*)
        docs="BLUEPRINT.md,DISCIPLINES.md"
        ;;
      *database*|*schema*|*model*|*migration*)
        docs="BLUEPRINT.md,PROVISIONS.md"
        ;;
      *test*|*spec*|*coverage*)
        docs="SENTINEL-PROTOCOLS.md,DISCIPLINES.md"
        ;;
      *integration*|*external*|*client*)
        docs="TRAILS.md,PROVISIONS.md"
        ;;
      *refactor*|*cleanup*|*debt*)
        docs="PATHOGENS.md,BLUEPRINT.md"
        ;;
      *setup*|*config*|*initialize*)
        docs="PROVISIONS.md,CHAMBERS.md"
        ;;
      *)
        docs="PROVISIONS.md,BLUEPRINT.md"
        ;;
    esac

    json_ok "$(jq -nc --arg d "$docs" --arg dir "$survey_dir" '{ok:true,docs:$d,dir:$dir}')"
    ;;

  survey-verify)
    survey_dir=".aether/data/survey"
    required="PROVISIONS.md TRAILS.md BLUEPRINT.md CHAMBERS.md DISCIPLINES.md SENTINEL-PROTOCOLS.md PATHOGENS.md"
    missing=""
    counts=""

    for doc in $required; do
      if [[ ! -f "$survey_dir/$doc" ]]; then
        missing="$missing $doc"
      else
        lines=$(wc -l < "$survey_dir/$doc" | tr -d ' ')
        counts="$counts $doc:$lines"
      fi
    done

    if [[ -n "$missing" ]]; then
      json_err "$E_FILE_NOT_FOUND" "Missing survey documents" "{\"missing\":\"$missing\"}"
    fi

    json_ok "$(jq -nc --arg c "$counts" '{ok:true,counts:$c}')"
    ;;

  checkpoint-check)
    _deprecation_warning "checkpoint-check"
    allowlist_file="$COLONY_DATA_DIR/checkpoint-allowlist.json"

    if [[ ! -f "$allowlist_file" ]]; then
      json_err "$E_FILE_NOT_FOUND" "Allowlist not found" "{\"path\":\"$allowlist_file\"}"
    fi

    # Get dirty files from git (staged or unstaged)
    dirty_files=$(git status --porcelain 2>/dev/null | awk '{print $2}' || true)  # SUPPRESS:OK -- existence-test: may not be a git repo

    if [[ -z "$dirty_files" ]]; then
      json_ok '{"ok":true,"system_files":[],"user_files":[],"has_user_files":false}'
      exit 0
    fi

    # Temporary files for building JSON
    system_files_tmp=$(mktemp)
    user_files_tmp=$(mktemp)

    # Check each file against allowlist patterns
    for file in $dirty_files; do
      is_system=false

      # Check against system file patterns
      if [[ "$file" == ".aether/aether-utils.sh" ]]; then
        is_system=true
      elif [[ "$file" == ".aether/workers.md" ]]; then
        is_system=true
      elif [[ "$file" == .aether/docs/*.md ]]; then
        is_system=true
      elif [[ "$file" == .claude/commands/ant/*.md ]] || [[ "$file" == .claude/commands/ant/**/*.md ]]; then
        is_system=true
      elif [[ "$file" == .claude/commands/st/*.md ]] || [[ "$file" == .claude/commands/st/**/*.md ]]; then
        is_system=true
      elif [[ "$file" == .opencode/commands/ant/*.md ]] || [[ "$file" == .opencode/commands/ant/**/*.md ]]; then
        is_system=true
      elif [[ "$file" == .opencode/agents/*.md ]] || [[ "$file" == .opencode/agents/**/*.md ]]; then
        is_system=true
      elif [[ "$file" == bin/* ]]; then
        is_system=true
      fi

      if [[ "$is_system" == "true" ]]; then
        echo "$file" >> "$system_files_tmp"
      else
        echo "$file" >> "$user_files_tmp"
      fi
    done

    # Build JSON using jq if available, otherwise use simple format
    if command -v jq >/dev/null 2>&1; then
      # SUPPRESS:OK -- read-default: file may not exist yet
      result=$(jq -n \
        # SUPPRESS:OK -- read-default: text escaping returns fallback on empty input
        --argjson system "$(jq -R . < "$system_files_tmp" 2>/dev/null | jq -s .)" \
        # SUPPRESS:OK -- read-default: text escaping returns fallback on empty input
        --argjson user "$(jq -R . < "$user_files_tmp" 2>/dev/null | jq -s .)" \
        '{ok: true, system_files: $system, user_files: $user, has_user_files: ($user | length > 0)}')
    else
      # Fallback without jq - simple output
      system_count=$(wc -l < "$system_files_tmp" 2>/dev/null | tr -d ' ' || echo "0")  # SUPPRESS:OK -- read-default: file may not exist
      user_count=$(wc -l < "$user_files_tmp" 2>/dev/null | tr -d ' ' || echo "0")  # SUPPRESS:OK -- read-default: file may not exist
      has_user=false
      [[ "$user_count" -gt 0 ]] && has_user=true
      result="{\"ok\":true,\"system_files\":[],\"user_files\":[],\"has_user_files\":$has_user}"
    fi

    rm -f "$system_files_tmp" "$user_files_tmp"
    echo "$result"
    exit 0
    ;;

  normalize-args)
    # Normalize arguments from Claude Code ($ARGUMENTS) or OpenCode ($@)
    # Usage: bash .aether/aether-utils.sh normalize-args [args...]
    # Returns: a plain normalized argument string (safe for direct string parsing)
    #
    # Claude Code passes args in $ARGUMENTS variable
    # OpenCode passes args in $@ (positional parameters)
    # This command outputs the normalized arguments as a single string

    normalized=""

    # Try Claude Code style first ($ARGUMENTS environment variable)
    if [ -n "${ARGUMENTS:-}" ]; then
      normalized="$ARGUMENTS"
    # Fall back to OpenCode style ($@ positional params)
    elif [ $# -gt 0 ]; then
      # Join positional args into one parseable string without adding synthetic quotes
      normalized="$*"
    fi

    # Collapse line breaks and repeated whitespace to avoid shell parse edge cases
    if [[ -n "$normalized" ]]; then
      normalized="$(printf '%s' "$normalized" | tr '\r\n' '  ' | sed 's/[[:space:]][[:space:]]*/ /g; s/^ //; s/ $//')"
    fi

    # Output normalized arguments
    printf '%s\n' "$normalized"
    exit 0
    ;;

  # Backward compatibility wrappers for session commands
  survey-verify-fresh)
    _deprecation_warning "survey-verify-fresh"
    # Backward compatibility: delegate to session-verify-fresh --command survey
    # Usage: bash .aether/aether-utils.sh survey-verify-fresh [--force] <survey_start_unixtime>

    force_mode=""
    survey_start_time=""

    # Parse arguments
    for arg in "$@"; do
      if [[ "$arg" == "--force" ]]; then
        force_mode="--force"
      elif [[ "$arg" =~ ^[0-9]+$ ]]; then
        survey_start_time="$arg"
      fi
    done

    # Delegate to generic command
    if [[ -n "$force_mode" ]]; then
      $0 session-verify-fresh --command survey --force "$survey_start_time"
    else
      $0 session-verify-fresh --command survey "$survey_start_time"
    fi
    ;;

  survey-clear)
    _deprecation_warning "survey-clear"
    # Backward compatibility: delegate to session-clear --command survey
    # Usage: bash .aether/aether-utils.sh survey-clear [--dry-run]

    dry_run=""

    # Parse arguments
    for arg in "$@"; do
      if [[ "$arg" == "--dry-run" ]]; then
        dry_run="--dry-run"
      fi
    done

    # Delegate to generic command
    if [[ "$dry_run" == "--dry-run" ]]; then
      $0 session-clear --command survey --dry-run
    else
      $0 session-clear --command survey
    fi
    ;;

  session-verify-fresh) _session_verify_fresh "$@" ;;
  session-clear) _session_clear "$@" ;;


  pheromone-export-eternal) _deprecation_warning "pheromone-export-eternal"; _pheromone_export_eternal "$@" ;;
  pheromone-write) _pheromone_write "$@" ;;
  pheromone-count) _pheromone_count "$@" ;;
  pheromone-display) _pheromone_display "$@" ;;
  pheromone-read) _pheromone_read "$@" ;;

  instinct-read) _instinct_read "$@" ;;
  instinct-create) _instinct_create "$@" ;;
  instinct-apply) _instinct_apply "$@" ;;
  learning-extract-fallback) _learning_extract_fallback "$@" ;;


  pheromone-prime) _pheromone_prime "$@" ;;
  colony-prime) _colony_prime "$@" ;;
  pheromone-expire) _pheromone_expire "$@" ;;
  eternal-init) _eternal_init "$@" ;;
  eternal-store) _eternal_store "$@" ;;

  hive-init) _hive_init "$@" ;;

  hive-store) _hive_store "$@" ;;

  hive-read) _hive_read "$@" ;;

  hive-abstract) _hive_abstract "$@" ;;

  hive-promote) _hive_promote "$@" ;;

  midden-write) _midden_write "$@" ;;

  midden-ingest-errors) _midden_ingest_errors "$@" ;;

  # ============================================================================
  # XML Exchange Commands
  # ============================================================================


  pheromone-export-xml) _pheromone_export_xml "$@" ;;
  pheromone-import-xml) _pheromone_import_xml "$@" ;;
  pheromone-validate-xml) _pheromone_validate_xml "$@" ;;


  wisdom-export-xml)
    # Export queen wisdom to XML format
    # Usage: wisdom-export-xml [input_json] [output_xml]
    # Default input: .aether/data/queen-wisdom.json
    # Default output: .aether/exchange/queen-wisdom.xml

    wex_input="${1:-$DATA_DIR/queen-wisdom.json}"
    wex_output="${2:-$SCRIPT_DIR/exchange/queen-wisdom.xml}"

    # Graceful degradation: check for xmllint
    if ! command -v xmllint >/dev/null 2>&1; then
      json_err "$E_FEATURE_UNAVAILABLE" "xmllint is not installed. Try: xcode-select --install on macOS."
    fi

    # Look for wisdom data: check specified file, then COLONY_STATE memory
    if [[ ! -f "$wex_input" ]]; then
      # MIGRATE: direct COLONY_STATE.json access -- use _state_read_field instead
      # Try to extract from COLONY_STATE.json memory field
      if [[ -f "$DATA_DIR/COLONY_STATE.json" ]]; then
        wex_memory=$(jq '.memory // {}' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null || echo '{}')  # SUPPRESS:OK -- read-default: returns fallback if missing
        if [[ "$wex_memory" != "{}" && "$wex_memory" != "null" ]]; then
          # Create minimal wisdom JSON from colony memory
          wex_created_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
          printf '%s\n' "{
  \"version\": \"1.0.0\",
  # SUPPRESS:OK -- read-default: query may return empty
  \"metadata\": {\"created\": \"$wex_created_at\", \"colony_id\": \"$(jq -r '.goal // \"unknown\"' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null)\"},
  \"philosophies\": [],
  # SUPPRESS:OK -- read-default: query may return empty
  \"patterns\": $(echo "$wex_memory" | jq '[.instincts // [] | .[] | {\"id\": (. | @base64), \"content\": ., \"confidence\": 0.7, \"domain\": \"general\", \"source\": \"colony_memory\"}]' 2>/dev/null || echo '[]')
}" > "$wex_input"
        fi
      fi
    fi

    # If still no wisdom data, create minimal skeleton
    if [[ ! -f "$wex_input" ]]; then
      wex_created_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
      mkdir -p "$(dirname "$wex_input")"
      printf '%s\n' "{
  \"version\": \"1.0.0\",
  \"metadata\": {\"created\": \"$wex_created_at\", \"colony_id\": \"unknown\"},
  \"philosophies\": [],
  \"patterns\": []
}" > "$wex_input"
    fi

    # Ensure output directory exists
    mkdir -p "$(dirname "$wex_output")"

    # Source the exchange script
    source "$SCRIPT_DIR/exchange/wisdom-xml.sh"

    # Call the export function
    xml-wisdom-export "$wex_input" "$wex_output"
    ;;

  wisdom-import-xml)
    # Import wisdom from XML into JSON format
    # Usage: wisdom-import-xml <xml_file> [output_json]

    wix_xml="${1:-}"
    wix_output="${2:-$DATA_DIR/queen-wisdom.json}"

    if [[ -z "$wix_xml" ]]; then
      json_err "$E_VALIDATION_FAILED" "Missing XML file argument. Try: wisdom-import-xml <xml_file> [output_json]."
    fi

    if [[ ! -f "$wix_xml" ]]; then
      json_err "$E_FILE_NOT_FOUND" "XML file not found: $wix_xml. Try: check the file path."
    fi

    # Graceful degradation: check for xmllint
    if ! command -v xmllint >/dev/null 2>&1; then
      json_err "$E_FEATURE_UNAVAILABLE" "xmllint is not installed. Try: xcode-select --install on macOS."
    fi

    # Ensure output directory exists
    mkdir -p "$(dirname "$wix_output")"

    # Source the exchange script
    source "$SCRIPT_DIR/exchange/wisdom-xml.sh"

    # Call the import function
    xml-wisdom-import "$wix_xml" "$wix_output"
    ;;

  registry-export-xml)
    # Export colony registry to XML format
    # Usage: registry-export-xml [input_json] [output_xml]
    # Default input: .aether/data/colony-registry.json
    # Default output: .aether/exchange/colony-registry.xml

    rex_input="${1:-$DATA_DIR/colony-registry.json}"
    rex_output="${2:-$SCRIPT_DIR/exchange/colony-registry.xml}"

    # Graceful degradation: check for xmllint
    if ! command -v xmllint >/dev/null 2>&1; then
      json_err "$E_FEATURE_UNAVAILABLE" "xmllint is not installed. Try: xcode-select --install on macOS."
    fi

    # If no registry file exists, generate from chambers
    if [[ ! -f "$rex_input" ]]; then
      rex_chambers_dir="$AETHER_ROOT/.aether/chambers"
      rex_generated_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
      rex_colonies="[]"

      if [[ -d "$rex_chambers_dir" ]]; then
        # Scan chambers for manifest.json files
        rex_colonies=$(
          for manifest in "$rex_chambers_dir"/*/manifest.json; do
            [[ -f "$manifest" ]] || continue
            jq -c '{
              id: (.colony_id // .goal // "unknown"),
              name: (.goal // "Unnamed Colony"),
              created_at: (.created_at // "unknown"),
              sealed_at: (.sealed_at // null),
              status: (if .sealed_at then "sealed" else "active" end),
              chamber: input_filename
            }' "$manifest" 2>/dev/null || true  # SUPPRESS:OK -- cleanup: operation is best-effort
          done | jq -s '.' 2>/dev/null || echo '[]'  # SUPPRESS:OK -- read-default: returns fallback if missing
        )
      fi

      mkdir -p "$(dirname "$rex_input")"
      printf '%s\n' "{
  \"version\": \"1.0.0\",
  \"generated_at\": \"$rex_generated_at\",
  \"colonies\": $rex_colonies
}" > "$rex_input"
    fi

    # Ensure output directory exists
    mkdir -p "$(dirname "$rex_output")"

    # Source the exchange script
    source "$SCRIPT_DIR/exchange/registry-xml.sh"

    # Call the export function
    xml-registry-export "$rex_input" "$rex_output"
    ;;

  registry-import-xml)
    # Import colony registry from XML into JSON format
    # Usage: registry-import-xml <xml_file> [output_json]

    rix_xml="${1:-}"
    rix_output="${2:-$DATA_DIR/colony-registry.json}"

    if [[ -z "$rix_xml" ]]; then
      json_err "$E_VALIDATION_FAILED" "Missing XML file argument. Try: registry-import-xml <xml_file> [output_json]."
    fi

    if [[ ! -f "$rix_xml" ]]; then
      json_err "$E_FILE_NOT_FOUND" "XML file not found: $rix_xml. Try: check the file path."
    fi

    # Graceful degradation: check for xmllint
    if ! command -v xmllint >/dev/null 2>&1; then
      json_err "$E_FEATURE_UNAVAILABLE" "xmllint is not installed. Try: xcode-select --install on macOS."
    fi

    # Ensure output directory exists
    mkdir -p "$(dirname "$rix_output")"

    # Source the exchange script
    source "$SCRIPT_DIR/exchange/registry-xml.sh"

    # Call the import function
    xml-registry-import "$rix_xml" "$rix_output"
    ;;

  colony-archive-xml) _colony_archive_xml "$@" ;;

  rolling-summary)
    # Maintain a bounded rolling narrative log for low-token context recovery.
    # Usage:
    #   rolling-summary add <event_type> <summary> [source]
    #   rolling-summary read [--json]
    rs_action="${1:-read}"
    rs_file="$COLONY_DATA_DIR/rolling-summary.log"

    case "$rs_action" in
      add)
        rs_event="${2:-}"
        rs_summary="${3:-}"
        rs_source="${4:-system}"
        [[ -z "$rs_event" || -z "$rs_summary" ]] && json_err "$E_VALIDATION_FAILED" "Usage: rolling-summary add <event_type> <summary> [source]"

        mkdir -p "$COLONY_DATA_DIR"
        touch "$rs_file"

        rs_clean_summary=$(printf '%s' "$rs_summary" | tr '\n' ' ' | sed 's/[[:space:]]\+/ /g; s/^ //; s/ $//' | sed 's/|/\\/g' | cut -c1-180)
        rs_clean_source=$(printf '%s' "$rs_source" | tr '\n' ' ' | sed 's/[[:space:]]\+/ /g; s/^ //; s/ $//' | sed 's/|/\\/g' | cut -c1-40)
        rs_clean_event=$(printf '%s' "$rs_event" | tr '\n' ' ' | sed 's/[[:space:]]\+/ /g; s/^ //; s/ $//' | sed 's/|/\\/g' | cut -c1-24)
        rs_ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

        printf '%s|%s|%s|%s\n' "$rs_ts" "$rs_clean_event" "$rs_clean_source" "$rs_clean_summary" >> "$rs_file"
        tail -n 15 "$rs_file" > "$rs_file.tmp" 2>/dev/null || true  # SUPPRESS:OK -- read-default: file may not exist
        mv "$rs_file.tmp" "$rs_file" || {
          _aether_log_error "Could not save rolling summary update"
          json_err "$E_UNKNOWN" "Failed to finalize rolling summary file"
        }

        json_ok "$(jq -nc --arg e "$rs_clean_event" --arg s "$rs_clean_source" '{added:true,event:$e,source:$s}')"
        ;;

      read)
        rs_json=false
        if [[ "${2:-}" == "--json" ]]; then
          rs_json=true
        fi

        if [[ ! -f "$rs_file" ]]; then
          if [[ "$rs_json" == "true" ]]; then
            json_ok '{"entries":[],"count":0}'
          else
            echo "No rolling summary entries."
          fi
          exit 0
        fi

        if [[ "$rs_json" == "true" ]]; then
          # SUPPRESS:OK -- read-default: text escaping returns fallback on empty input
          rs_entries=$(awk -F'|' 'NF >= 4 {print $0}' "$rs_file" | tail -n 15 | jq -R 'split("|") | {timestamp: .[0], event: .[1], source: .[2], summary: (.[3:] | join("|"))}' | jq -s '.' 2>/dev/null || echo '[]')
          rs_count=$(echo "$rs_entries" | jq 'length' 2>/dev/null || echo 0)  # SUPPRESS:OK -- read-default: file may not exist yet
          json_ok "{\"entries\":$rs_entries,\"count\":$rs_count}"
        else
          tail -n 15 "$rs_file"
        fi
        ;;

      *)
        json_err "$E_VALIDATION_FAILED" "Usage: rolling-summary add|read ..."
        ;;
    esac
    ;;

  context-capsule)
    # Generate a compact, bounded context block for prompt injection.
    # Usage: context-capsule [--compact] [--json] [--max-signals N] [--max-decisions N] [--max-risks N] [--max-words N]
    cc_compact=false
    cc_json=false
    cc_max_signals=8
    cc_max_decisions=3
    cc_max_risks=2
    cc_max_words=220

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --compact) cc_compact=true ;;
        --json) cc_json=true ;;
        --max-signals) shift; cc_max_signals="${1:-8}" ;;
        --max-decisions) shift; cc_max_decisions="${1:-3}" ;;
        --max-risks) shift; cc_max_risks="${1:-2}" ;;
        --max-words) shift; cc_max_words="${1:-220}" ;;
      esac
      shift
    done

    [[ "$cc_max_signals" =~ ^[0-9]+$ ]] || cc_max_signals=8
    [[ "$cc_max_decisions" =~ ^[0-9]+$ ]] || cc_max_decisions=3
    [[ "$cc_max_risks" =~ ^[0-9]+$ ]] || cc_max_risks=2
    [[ "$cc_max_words" =~ ^[0-9]+$ ]] || cc_max_words=220
    [[ "$cc_max_signals" -lt 1 ]] && cc_max_signals=1
    [[ "$cc_max_decisions" -lt 1 ]] && cc_max_decisions=1
    [[ "$cc_max_risks" -lt 1 ]] && cc_max_risks=1
    [[ "$cc_max_words" -lt 80 ]] && cc_max_words=80

    # MIGRATE: direct COLONY_STATE.json access -- use _state_read_field instead
    cc_state_file="$DATA_DIR/COLONY_STATE.json"
    cc_flags_file="$COLONY_DATA_DIR/flags.json"
    cc_pher_file="$COLONY_DATA_DIR/pheromones.json"
    cc_roll_file="$COLONY_DATA_DIR/rolling-summary.log"
    cc_now_iso=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    if [[ ! -f "$cc_state_file" ]]; then
      json_ok '{"exists":false,"prompt_section":"","word_count":0}'
      exit 0
    fi

    cc_goal=$(jq -r '.goal // "No goal set"' "$cc_state_file" 2>/dev/null || echo "No goal set")  # SUPPRESS:OK -- read-default: file may not exist yet
    cc_state=$(jq -r '.state // "IDLE"' "$cc_state_file" 2>/dev/null || echo "IDLE")  # SUPPRESS:OK -- read-default: file may not exist yet
    cc_current_phase=$(jq -r '.current_phase // 0' "$cc_state_file" 2>/dev/null || echo 0)  # SUPPRESS:OK -- read-default: file may not exist yet
    cc_total_phases=$(jq -r '.plan.phases | length // 0' "$cc_state_file" 2>/dev/null || echo 0)  # SUPPRESS:OK -- read-default: file may not exist yet
    # SUPPRESS:OK -- cross-platform: macOS vs Linux date/stat flags
    cc_phase_name=$(jq -r --argjson p "$cc_current_phase" 'if $p > 0 then (.plan.phases[]? | select(.id == $p) | .name) else "" end' "$cc_state_file" 2>/dev/null | head -1)
    [[ -z "$cc_phase_name" ]] && cc_phase_name="(unnamed)"

    cc_next_action="/ant:status"
    if [[ "$cc_total_phases" -eq 0 ]]; then
      cc_next_action="/ant:plan"
    elif [[ "$cc_state" == "EXECUTING" ]]; then
      cc_next_action="/ant:continue"
    elif [[ "$cc_state" == "READY" && "$cc_current_phase" -eq 0 ]]; then
      cc_next_action="/ant:build 1"
    elif [[ "$cc_state" == "READY" && "$cc_current_phase" -gt 0 && "$cc_current_phase" -lt "$cc_total_phases" ]]; then
      cc_next_action="/ant:build $((cc_current_phase + 1))"
    elif [[ "$cc_state" == "READY" && "$cc_current_phase" -ge "$cc_total_phases" ]]; then
      cc_next_action="/ant:seal"
    elif [[ "$cc_state" == "PAUSED" ]]; then
      cc_next_action="/ant:resume-colony"
    fi

    cc_decisions=$(jq -r --argjson n "$cc_max_decisions" '
      (.memory.decisions // [])
      | reverse
      | .[:$n]
      | map(
          if type == "object" then
            (.decision // .summary // .description // .content // tostring)
          else
            tostring
          end
        )
      | .[]
      ' "$cc_state_file" 2>/dev/null | sed 's/[[:space:]]\+/ /g; s/^ //; s/ $//' | cut -c1-160)  # SUPPRESS:OK -- read-default: file may not exist yet

    cc_risks=""
    if [[ -f "$cc_flags_file" ]]; then
      cc_risks=$(jq -r --argjson n "$cc_max_risks" '
        (.flags // [])
        | map(select((.resolved // false) != true and ((.type // "issue") == "blocker" or (.type // "issue") == "issue"))
          | (.title // .description // .details // tostring))
        | .[:$n]
        | .[]
      ' "$cc_flags_file" 2>/dev/null | sed 's/[[:space:]]\+/ /g; s/^ //; s/ $//' | cut -c1-160)  # SUPPRESS:OK -- read-default: file may not exist yet
    fi

    cc_signals=""
    if [[ -f "$cc_pher_file" ]]; then
      cc_signals=$(jq -r --arg now_iso "$cc_now_iso" --argjson max "$cc_max_signals" '
        def to_epoch(ts):
          if ts == null or ts == "" or ts == "phase_end" then null
          else
            (ts | split("T")) as $parts |
            ($parts[0] | split("-")) as $d |
            ($parts[1] | rtrimstr("Z") | split(":")) as $t |
            (($d[0] | tonumber) - 1970) * 365 * 86400 +
            (($d[1] | tonumber) - 1) * 30 * 86400 +
            (($d[2] | tonumber) - 1) * 86400 +
            ($t[0] | tonumber) * 3600 +
            ($t[1] | tonumber) * 60 +
            ($t[2] | rtrimstr("Z") | tonumber)
          end;
        def decay_days(t):
          if t == "FOCUS" then 30 elif t == "REDIRECT" then 60 else 90 end;
        (to_epoch($now_iso)) as $now |
        .signals
        | map(
            (to_epoch(.created_at)) as $created_epoch |
            (if $created_epoch != null then ($now - $created_epoch) / 86400 else 0 end) as $elapsed_days |
            (decay_days(.type)) as $dd |
            ((.strength // 0.8) * (1 - ($elapsed_days / $dd))) as $eff_raw |
            (if $eff_raw < 0 then 0 else $eff_raw end) as $eff |
            . + {
              effective_strength: (($eff * 100 | round) / 100),
              priority: (if .type == "REDIRECT" then 1 elif .type == "FOCUS" then 2 elif .type == "FEEDBACK" then 3 elif .type == "POSITION" then 4 else 5 end)
            }
          )
        | map(select((if .active == false then false else true end) == true and (.effective_strength // 0) >= 0.1))
        | sort_by(.priority, -(.effective_strength // 0))
        | .[:$max]
        | map((.type // "UNKNOWN") + ": " + (.content.text // (if (.content | type) == "string" then .content else "" end)))
        | .[]
      ' "$cc_pher_file" 2>/dev/null | sed 's/[[:space:]]\+/ /g; s/^ //; s/ $//' | cut -c1-160)  # SUPPRESS:OK -- read-default: file may not exist yet
    fi

    cc_roll=""
    if [[ -f "$cc_roll_file" ]]; then
      # SUPPRESS:OK -- read-default: file may not exist or format may vary
      cc_roll=$(tail -n 3 "$cc_roll_file" 2>/dev/null | awk -F'|' 'NF >= 4 {print $2 ": " $4}' | sed 's/[[:space:]]\+/ /g; s/^ //; s/ $//' | cut -c1-160)
    fi

    cc_section="--- CONTEXT CAPSULE ---"$'\n'
    cc_section+="Goal: $(printf '%s' "$cc_goal" | tr '\n' ' ' | sed 's/[[:space:]]\+/ /g; s/^ //; s/ $//' | cut -c1-160)"$'\n'
    cc_section+="State: $cc_state"$'\n'
    cc_section+="Phase: $cc_current_phase/$cc_total_phases - $(printf '%s' "$cc_phase_name" | tr '\n' ' ' | sed 's/[[:space:]]\+/ /g; s/^ //; s/ $//' | cut -c1-120)"$'\n'
    cc_section+="Next: $cc_next_action"$'\n'

    if [[ -n "$cc_signals" ]]; then
      cc_section+=$'\n'"Active signals:"$'\n'
      while IFS= read -r line; do
        [[ -n "$line" ]] && cc_section+="- $line"$'\n'
      done <<< "$cc_signals"
    fi

    if [[ -n "$cc_decisions" ]]; then
      cc_section+=$'\n'"Recent decisions:"$'\n'
      while IFS= read -r line; do
        [[ -n "$line" ]] && cc_section+="- $line"$'\n'
      done <<< "$cc_decisions"
    fi

    if [[ -n "$cc_risks" ]]; then
      cc_section+=$'\n'"Open risks:"$'\n'
      while IFS= read -r line; do
        [[ -n "$line" ]] && cc_section+="- $line"$'\n'
      done <<< "$cc_risks"
    fi

    if [[ -n "$cc_roll" ]]; then
      cc_section+=$'\n'"Recent narrative:"$'\n'
      while IFS= read -r line; do
        [[ -n "$line" ]] && cc_section+="- $line"$'\n'
      done <<< "$cc_roll"
    fi

    cc_section+="--- END CONTEXT CAPSULE ---"

    if [[ "$cc_compact" == "true" ]]; then
      cc_words=$(printf '%s' "$cc_section" | wc -w | tr -d ' ')
      if [[ "$cc_words" -gt "$cc_max_words" ]]; then
        cc_section=$(printf '%s' "$cc_section" | awk '
          BEGIN{keep=1}
          /^Recent narrative:/{keep=0}
          /^--- END CONTEXT CAPSULE ---$/{print; next}
          {if(keep==1) print}
        ')
      fi
      cc_words=$(printf '%s' "$cc_section" | wc -w | tr -d ' ')
      if [[ "$cc_words" -gt "$cc_max_words" ]]; then
        cc_section=$(printf '%s' "$cc_section" | awk '
          BEGIN{keep=1}
          /^Open risks:/{keep=0}
          /^--- END CONTEXT CAPSULE ---$/{print; next}
          {if(keep==1) print}
        ')
      fi
    fi

    cc_words=$(printf '%s' "$cc_section" | wc -w | tr -d ' ')
    cc_prompt_json=$(printf '%s' "$cc_section" | jq -Rs '.' 2>/dev/null || echo '""')  # SUPPRESS:OK -- read-default: returns fallback if missing
    json_ok "$(jq -nc --arg st "$cc_state" --arg na "$cc_next_action" --argjson wc "$cc_words" --argjson ps "$cc_prompt_json" '{exists:true,state:$st,next_action:$na,word_count:$wc,prompt_section:$ps}')"
    ;;

  session-init) _session_init "$@" ;;
  session-update) _session_update "$@" ;;
  session-read) _session_read "$@" ;;
  session-is-stale) _session_is_stale "$@" ;;
  session-clear-context) _session_clear_context "$@" ;;
  session-mark-resumed) _session_mark_resumed "$@" ;;
  session-summary) _session_summary "$@" ;;

  pending-decision-add) _pending_decision_add "$@" ;;
  pending-decision-list) _pending_decision_list "$@" ;;
  pending-decision-resolve) _pending_decision_resolve "$@" ;;

  autopilot-headless-check) _autopilot_headless_check "$@" ;;
  autopilot-set-headless) _autopilot_set_headless "$@" ;;

  generate-progress-bar)
    generate-progress-bar "$@"
    ;;
  print-standard-banner)
    print-standard-banner "$@"
    ;;
  print-next-up)
    print-next-up "$@"
    ;;

  # ============================================
  # CHANGELOG COMMANDS
  # ============================================

  changelog-append)
    # Append entry to CHANGELOG.md
    # Usage: changelog-append <date> <phase> <plan> <files> <decisions> <worked> <requirements>
    changelog-append "$@"
    ;;

  changelog-collect-plan-data)
    # Collect plan data for changelog entry
    # Usage: changelog-collect-plan-data <phase> <plan>
    changelog-collect-plan-data "$@"
    ;;

  # ============================================
  # LOCK MANAGEMENT
  # ============================================

  force-unlock)
    # Emergency lock cleanup — remove all locks or stale-only locks
    # Usage: force-unlock [--yes] [--stale-only]
    # Without --yes, lists locks and asks for confirmation in interactive mode
    lock_dir="${AETHER_ROOT:-.}/.aether/locks"
    auto_yes=false
    stale_only=false

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --yes) auto_yes=true ;;
        --stale-only) stale_only=true ;;
      esac
      shift
    done

    if [[ ! -d "$lock_dir" ]]; then
      json_ok '{"removed":0,"message":"No locks directory found"}'
      exit 0
    fi

    if [[ "$stale_only" == "true" ]]; then
      lock_timeout="${LOCK_TIMEOUT:-300}"
      scanned=0
      removed=0
      skipped_live=0

      is_lock_stale_for_cleanup() {
        local _lock_file="$1"
        local _pid_file="${_lock_file}.pid"
        local _pid
        _pid=$(cat "$_pid_file" 2>/dev/null || echo "")  # SUPPRESS:OK -- read-default: file may not exist yet
        if [[ -z "$_pid" ]]; then
          _pid=$(cat "$_lock_file" 2>/dev/null || echo "")  # SUPPRESS:OK -- read-default: file may not exist yet
        fi
        _pid=$(echo "$_pid" | tr -d '[:space:]')
        [[ "$_pid" =~ ^[0-9]+$ ]] || _pid=""

        if [[ -n "$_pid" ]]; then
          kill -0 "$_pid" 2>/dev/null && return 1  # SUPPRESS:OK -- existence-test: checking if process is alive
          return 0
        fi

        local _mtime=0
        if stat -f %m "$_lock_file" >/dev/null 2>&1; then  # SUPPRESS:OK -- cleanup: output suppression for clean operation
          _mtime=$(stat -f %m "$_lock_file" 2>/dev/null || echo 0)  # SUPPRESS:OK -- cross-platform: macOS stat syntax
        else
          _mtime=$(stat -c %Y "$_lock_file" 2>/dev/null || echo 0)  # SUPPRESS:OK -- cross-platform: Linux stat syntax
        fi
        local _age=$(( $(date +%s) - _mtime ))
        [[ "$_age" -gt "$lock_timeout" ]]
      }

      for lock_file in "$lock_dir"/*.lock; do
        [[ -e "$lock_file" ]] || continue
        scanned=$((scanned + 1))
        if is_lock_stale_for_cleanup "$lock_file"; then
          rm -f "$lock_file" "${lock_file}.pid" 2>/dev/null || true  # SUPPRESS:OK -- cleanup: file may not exist
          removed=$((removed + 1))
        else
          skipped_live=$((skipped_live + 1))
        fi
      done

      if [[ "$removed" -eq 0 && "$scanned" -eq 0 ]]; then
        json_ok '{"removed":0,"scanned":0,"skipped_live":0,"message":"No lock files found","mode":"stale-only"}'
      else
        json_ok "{\"removed\":$removed,\"scanned\":$scanned,\"skipped_live\":$skipped_live,\"message\":\"Stale locks cleared\",\"mode\":\"stale-only\"}"
      fi
      exit 0
    fi

    lock_files=$(find "$lock_dir" -name "*.lock" -o -name "*.lock.pid" 2>/dev/null)  # SUPPRESS:OK -- existence-test: directory may not exist

    if [[ -z "$lock_files" ]]; then
      json_ok '{"removed":0,"message":"No lock files found"}'
      exit 0
    fi

    lock_count=$(echo "$lock_files" | grep -c '\.lock$' || echo "0")  # SUPPRESS:OK -- read-default: grep returns 1 when no matches

    if [[ "$auto_yes" != "true" ]]; then
      if [[ -t 2 ]]; then
        echo "" >&2
        echo "Lock files found in $lock_dir:" >&2
        echo "$lock_files" | while read -r f; do
          [[ "$f" == *.pid ]] && continue
          pid_content=$(cat "${f}.pid" 2>/dev/null || echo "unknown")  # SUPPRESS:OK -- read-default: file may not exist yet
          echo "  $f (PID: $pid_content)" >&2
        done
        printf "Remove all %d lock(s)? [y/N] " "$lock_count" >&2
        read -r response < /dev/tty
        if [[ ! "$response" =~ ^[Yy]$ ]]; then
          json_ok '{"removed":0,"message":"Cancelled by user"}'
          exit 0
        fi
      else
        json_err "$E_VALIDATION_FAILED" "force-unlock requires --yes flag in non-interactive mode"
      fi
    fi

    rm -f "$lock_dir"/*.lock "$lock_dir"/*.lock.pid 2>/dev/null || true  # SUPPRESS:OK -- cleanup: file may not exist
    export LOCK_ACQUIRED=false
    export CURRENT_LOCK=""
    json_ok "{\"removed\":$lock_count,\"message\":\"All locks cleared\"}"
    ;;

  #=============================================================================
  # SEMANTIC COMMANDS
  #=============================================================================

  semantic-init)
    # Initialize semantic store
    semantic-init
    ;;

  semantic-index)
    # Index text for semantic search
    # Usage: semantic-index <text> <source> [entry_id]
    text="${2:-}"
    source="${3:-unknown}"
    entry_id="${4:-}"

    if [[ -z "$text" ]]; then
      json_err "$E_VALIDATION_FAILED" "semantic-index requires text argument"
      exit 1
    fi

    semantic-index "$text" "$source" "$entry_id"
    ;;

  semantic-search)
    # Search for similar entries
    # Usage: semantic-search <query> [top_k] [threshold] [source_filter]
    query="${2:-}"
    top_k="${3:-5}"
    threshold="${4:-0.5}"
    source_filter="${5:-}"

    if [[ -z "$query" ]]; then
      json_err "$E_VALIDATION_FAILED" "semantic-search requires query argument"
      exit 1
    fi

    semantic-search "$query" "$top_k" "$threshold" "$source_filter"
    ;;

  semantic-rebuild)
    # Rebuild semantic index from all data sources
    semantic-rebuild
    ;;

  semantic-status)
    # Get semantic layer status
    semantic-status
    ;;

  semantic-context)
    _deprecation_warning "semantic-context"
    # Get context for task (for worker injection)
    # Usage: semantic-context <task_description> [max_results]
    task="${2:-}"
    max_results="${3:-3}"

    if [[ -z "$task" ]]; then
      json_ok "[]" "No task provided"
      exit 0
    fi

    semantic-get-context "$task" "$max_results"
    ;;

  entropy-score)
    # Calculate colony entropy score (0-100). Higher means more disorder.
    # Usage: entropy-score
    spawn_count=0
    failure_count=0
    rule_count=0
    signal_count=0

    if [[ -f "$COLONY_DATA_DIR/spawn-tree.txt" ]]; then
      spawn_count=$(grep -c "|spawned$" "$COLONY_DATA_DIR/spawn-tree.txt" 2>/dev/null || echo 0)  # SUPPRESS:OK -- read-default: count defaults to 0 if file missing
    fi

    if [[ -f "$COLONY_DATA_DIR/midden/midden.json" ]]; then
      # SUPPRESS:OK -- read-default: query may return empty
      failure_count=$(jq '[.entries[]? | select(.category == "failure")] | length' "$COLONY_DATA_DIR/midden/midden.json" 2>/dev/null || echo 0)
      if [[ "$failure_count" == "0" ]]; then
        # Backward compatibility for older midden schema
        # SUPPRESS:OK -- read-default: file may not exist yet
        failure_count=$(jq '[.signals[]? | select(.type == "failure")] | length' "$COLONY_DATA_DIR/midden/midden.json" 2>/dev/null || echo 0)
      fi
    fi

    if [[ -f "$AETHER_ROOT/.aether/QUEEN.md" ]]; then
      rule_count=$(grep -c "^-" "$AETHER_ROOT/.aether/QUEEN.md" 2>/dev/null || echo 0)  # SUPPRESS:OK -- read-default: count defaults to 0 if file missing
    fi

    if [[ -f "$COLONY_DATA_DIR/pheromones.json" ]]; then
      signal_count=$(jq '.signals | length' "$COLONY_DATA_DIR/pheromones.json" 2>/dev/null || echo 0)  # SUPPRESS:OK -- read-default: file may not exist yet
    fi

    raw_score=$(( (spawn_count * 2) + (failure_count * 5) + signal_count - (rule_count / 2) ))
    score=$raw_score
    [[ "$score" -lt 0 ]] && score=0
    [[ "$score" -gt 100 ]] && score=100

    json_ok "{\"score\":$score,\"spawn_count\":$spawn_count,\"failure_count\":$failure_count,\"rule_count\":$rule_count,\"signal_count\":$signal_count,\"raw\":$raw_score}"
    ;;

  memory-metrics)
    # Aggregate memory health metrics from all data sources
    # Usage: memory-metrics
    # Returns: JSON with wisdom, pending, recent_failures, and last_activity

    queen_file="$AETHER_ROOT/.aether/QUEEN.md"
    observations_file="$COLONY_DATA_DIR/learning-observations.json"
    deferred_file="$COLONY_DATA_DIR/learning-deferred.json"
    midden_file="$COLONY_DATA_DIR/midden/midden.json"

    # Initialize result structure
    wisdom_total=0
    wisdom_by_type='{"philosophy":0,"pattern":0,"redirect":0,"stack":0,"decree":0}'
    queen_last_updated="null"

    # Read QUEEN.md metadata if available
    if [[ -f "$queen_file" ]]; then
      # Extract metadata block from HTML comment
      # SUPPRESS:OK -- read-default: section may not exist in file
      metadata=$(sed -n '/<!-- METADATA/,/-->/p' "$queen_file" 2>/dev/null | sed 's/<!-- METADATA//' | sed 's/-->//')
      if [[ -n "$metadata" ]]; then
        # Parse stats from metadata
        wisdom_total=$(echo "$metadata" | jq -r '.stats.total_philosophies // 0 + .stats.total_patterns // 0 + .stats.total_redirects // 0 + .stats.total_stack_entries // 0 + .stats.total_decrees // 0')
        philosophy_count=$(echo "$metadata" | jq -r '.stats.total_philosophies // 0')
        pattern_count=$(echo "$metadata" | jq -r '.stats.total_patterns // 0')
        redirect_count=$(echo "$metadata" | jq -r '.stats.total_redirects // 0')
        stack_count=$(echo "$metadata" | jq -r '.stats.total_stack_entries // 0')
        decree_count=$(echo "$metadata" | jq -r '.stats.total_decrees // 0')
        wisdom_by_type="{\"philosophy\":$philosophy_count,\"pattern\":$pattern_count,\"redirect\":$redirect_count,\"stack\":$stack_count,\"decree\":$decree_count}"
        queen_last_updated=$(echo "$metadata" | jq -r '.last_evolved // "null"')
        [[ "$queen_last_updated" == "null" ]] && queen_last_updated="null" || queen_last_updated="\"$queen_last_updated\""
      fi

      # Get file mtime as fallback for last_updated
      if [[ "$queen_last_updated" == "null" ]]; then
        # SUPPRESS:OK -- cross-platform: macOS vs Linux date/stat flags
        queen_mtime=$(stat -f %m "$queen_file" 2>/dev/null || stat -c %Y "$queen_file" 2>/dev/null || echo "0")
        if [[ "$queen_mtime" != "0" ]]; then
          # SUPPRESS:OK -- cross-platform: macOS vs Linux date/stat flags
          queen_last_updated="\"$(date -u -r "$queen_mtime" '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date -u -d "@$queen_mtime" '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || echo "null")\""
          [[ "$queen_last_updated" == '"null"' ]] && queen_last_updated="null"
        fi
      fi
    fi

    # Count pending observations (meeting threshold but not promoted)
    pending_observations=0
    learning_captured="null"
    if [[ -f "$observations_file" ]]; then
      # Get threshold for each wisdom type
      thresholds=$(bash "$0" queen-thresholds 2>/dev/null | jq -c '.result // {}')  # SUPPRESS:OK -- read-default: subcommand may fail

      # Count observations meeting thresholds that aren't in QUEEN.md
      if [[ -f "$queen_file" ]]; then
        queen_content=$(cat "$queen_file" 2>/dev/null)  # SUPPRESS:OK -- read-default: file may not exist yet
        pending_observations=$(jq --argjson thresholds "$thresholds" --arg queen "$queen_content" '
          [.observations[]? | select(
            (.observation_count // 0) >= ($thresholds[.wisdom_type].propose // 1) and
            ($queen | contains(.content) | not)
          )] | length
        ' "$observations_file" 2>/dev/null || echo "0")  # SUPPRESS:OK -- read-default: file may not exist yet
      else
        pending_observations=$(jq --argjson thresholds "$thresholds" '
          [.observations[]? | select((.observation_count // 0) >= ($thresholds[.wisdom_type].propose // 1))] | length
        ' "$observations_file" 2>/dev/null || echo "0")  # SUPPRESS:OK -- read-default: file may not exist yet
      fi

      # Get last learning timestamp
      last_obs=$(jq -r '[.observations[]?.last_seen] | max // empty' "$observations_file" 2>/dev/null)  # SUPPRESS:OK -- read-default: file may not exist yet
      if [[ -n "$last_obs" ]]; then
        learning_captured="\"$last_obs\""
      fi
    fi

    # Count deferred proposals
    deferred_count=0
    if [[ -f "$deferred_file" ]]; then
      deferred_count=$(jq '[.deferred[]?] | length' "$deferred_file" 2>/dev/null || echo "0")  # SUPPRESS:OK -- read-default: file may not exist yet
    fi

    # Count recent failures from midden
    failure_count=0
    last_failure="null"
    failures_json="[]"
    if [[ -f "$midden_file" ]]; then
      # Get failures sorted by created_at descending
      # SUPPRESS:OK -- read-default: file may not exist yet
      failures_data=$(jq '[.signals[]? | select(.type == "failure")] | sort_by(.created_at) | reverse' "$midden_file" 2>/dev/null || echo "[]")
      failure_count=$(echo "$failures_data" | jq 'length')

      if [[ "$failure_count" -gt 0 ]]; then
        last_failure=$(echo "$failures_data" | jq -r '.[0].created_at // "null"')
        [[ "$last_failure" == "null" ]] || last_failure="\"$last_failure\""

        # Get last 5 failures for details
        # SUPPRESS:OK -- read-default: returns fallback on failure
        failures_json=$(echo "$failures_data" | jq '[.[:5][] | {created_at, source, context, content: .content.text}]' 2>/dev/null || echo "[]")
      fi
    fi

    # Build final JSON
    result=$(cat <<EOF
{
  "wisdom": {
    "total": $wisdom_total,
    "by_type": $wisdom_by_type,
    "last_updated": $queen_last_updated
  },
  "pending": {
    "observations": $pending_observations,
    "deferred": $deferred_count,
    "total": $((pending_observations + deferred_count))
  },
  "recent_failures": {
    "count": $failure_count,
    "last_failure": $last_failure,
    "failures": $failures_json
  },
  "last_activity": {
    "queen_md_updated": $queen_last_updated,
    "learning_captured": $learning_captured
  }
}
EOF
)

    echo "$result"
    exit 0
    ;;

  midden-recent-failures) _midden_recent_failures "$@" ;;

  midden-review) _midden_review "$@" ;;

  midden-acknowledge) _midden_acknowledge "$@" ;;

  midden-search) _midden_search "$@" ;;

  midden-tag) _midden_tag "$@" ;;

  trophallaxis-diagnose) _trophallaxis_diagnose "$@" ;;

  trophallaxis-retry) _trophallaxis_retry "$@" ;;

  scar-add) _scar_add "$@" ;;

  scar-list) _scar_list "$@" ;;

  scar-check) _scar_check "$@" ;;

  immune-auto-scar) _immune_auto_scar "$@" ;;

  # ── Council Deliberation ────────────────────────────────────────────────────
  council-deliberate) _council_deliberate "$@" ;;

  council-advocate) _council_advocate "$@" ;;

  council-challenger) _council_challenger "$@" ;;

  council-sage) _council_sage "$@" ;;

  council-history) _council_history "$@" ;;

  council-budget-check) _council_budget_check "$@" ;;

  resume-dashboard)
    # Generate dashboard data for /ant:resume command
    # Usage: resume-dashboard
    # Returns: JSON with current state, memory health, and recent activity

    # MIGRATE: direct COLONY_STATE.json access -- use _state_read_field instead
    colony_state_file="$DATA_DIR/COLONY_STATE.json"

    # Get current state from COLONY_STATE.json
    current_phase=0
    phase_name=""
    state="UNKNOWN"
    goal=""

    if [[ -f "$colony_state_file" ]]; then
      current_phase=$(jq -r '.current_phase // 0' "$colony_state_file" 2>/dev/null || echo "0")  # SUPPRESS:OK -- read-default: file may not exist yet
      state=$(jq -r '.state // "UNKNOWN"' "$colony_state_file" 2>/dev/null || echo "UNKNOWN")  # SUPPRESS:OK -- read-default: file may not exist yet
      goal=$(jq -r '.goal // ""' "$colony_state_file" 2>/dev/null || echo "")  # SUPPRESS:OK -- read-default: file may not exist yet
    fi

    # Get memory health metrics
    memory_health=$(bash "$0" memory-metrics 2>/dev/null || echo '{}')  # SUPPRESS:OK -- read-default: subcommand may fail
    wisdom_count=$(echo "$memory_health" | jq -r '.wisdom.total // 0')
    pending_promotions=$(echo "$memory_health" | jq -r '.pending.total // 0')
    recent_failures=$(echo "$memory_health" | jq -r '.recent_failures.count // 0')

    # Get recent decisions (last 5)
    recent_decisions="[]"
    if [[ -f "$colony_state_file" ]]; then
      # SUPPRESS:OK -- cross-platform: macOS vs Linux date/stat flags
      recent_decisions=$(jq -r '[.memory.decisions[]?] | reverse | [.[:5][]] | if . == [] then [] else . end' "$colony_state_file" 2>/dev/null || echo "[]")
    fi

    # Get recent events (last 10)
    recent_events="[]"
    if [[ -f "$colony_state_file" ]]; then
      # SUPPRESS:OK -- cross-platform: macOS vs Linux date/stat flags
      recent_events=$(jq -r '[.events[]?] | reverse | [.[:10][] | {timestamp, type, worker, details}]' "$colony_state_file" 2>/dev/null || echo "[]")
    fi

    # Get data safety stats (best-effort)
    ds_stats=$(bash "$0" data-safety-stats 2>/dev/null | jq -r '.result // {}' 2>/dev/null || echo '{}')  # SUPPRESS:OK -- read-default: subcommand may fail

    # Build dashboard JSON
    result=$(jq -n \
      --argjson phase "$current_phase" \
      --arg state "$state" \
      --arg goal "$goal" \
      --argjson wisdom "$wisdom_count" \
      --argjson pending "$pending_promotions" \
      --argjson failures "$recent_failures" \
      --argjson decisions "$recent_decisions" \
      --argjson events "$recent_events" \
      --argjson data_safety "$ds_stats" \
      '{
        "current": {
          "phase": $phase,
          "phase_name": $goal,
          "state": $state,
          "goal": $goal
        },
        "memory_health": {
          "wisdom_count": $wisdom,
          "pending_promotions": $pending,
          "recent_failures": $failures
        },
        "data_safety": $data_safety,
        "recent": {
          "decisions": $decisions,
          "events": $events
        },
        "drill_down": {
          "command": "/ant:memory-details",
          "available": true
        }
      }')

    echo "$result"
    exit 0
    ;;

  colony-vital-signs)
    # Compute colony health metrics from existing data files
    # Usage: colony-vital-signs
    # Returns: JSON with build_velocity, error_rate, signal_health, memory_pressure,
    #          colony_age_hours, and overall_health (0-100)
    _colony_vital_signs
    ;;

  data-safety-stats)
    # Read data safety statistics from safety-stats.json
    # Usage: data-safety-stats
    # Returns: JSON with safety event counts, or defaults if no stats file exists
    _ds_stats_file="$COLONY_DATA_DIR/safety-stats.json"
    if [[ -f "$_ds_stats_file" ]]; then
      _ds_stale=$(jq -r '.stale_locks_cleaned // 0' "$_ds_stats_file" 2>/dev/null || echo 0)
      _ds_rejects=$(jq -r '.json_validation_rejects // 0' "$_ds_stats_file" 2>/dev/null || echo 0)
      _ds_updated=$(jq -r '.last_updated // "unknown"' "$_ds_stats_file" 2>/dev/null || echo "unknown")
      _ds_total=$(( _ds_stale + _ds_rejects ))
      json_ok "$(jq -n \
        --argjson stale_locks_cleaned "$_ds_stale" \
        --argjson json_validation_rejects "$_ds_rejects" \
        --argjson total_events "$_ds_total" \
        --arg last_updated "$_ds_updated" \
        '{stale_locks_cleaned: $stale_locks_cleaned, json_validation_rejects: $json_validation_rejects, total_events: $total_events, last_updated: $last_updated}')"
    else
      json_ok '{"stale_locks_cleaned":0,"json_validation_rejects":0,"total_events":0,"last_updated":null}'
    fi
    exit 0
    ;;

  # colony-vital-signs handled above (dispatches to _colony_vital_signs in state-api.sh)

  suggest-analyze) _suggest_analyze "$@" ;;
  suggest-record) _suggest_record "$@" ;;
  suggest-check) _suggest_check "$@" ;;
  suggest-clear) _suggest_clear "$@" ;;
  suggest-approve) _suggest_approve "$@" ;;
  suggest-quick-dismiss) _suggest_quick_dismiss "$@" ;;

  data-clean)
    # Scan and remove test/synthetic artifacts from colony data files.
    # Usage: data-clean [--dry-run] [--confirm] [--json]
    # --dry-run (default): scan and report only
    # --confirm: actually remove artifacts
    # --json: output as JSON instead of human-readable text

    _dc_confirm=false
    _dc_json=false
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --confirm) _dc_confirm=true; shift ;;
        --dry-run) shift ;;
        --json) _dc_json=true; shift ;;
        *) shift ;;
      esac
    done

    # Test patterns for artifact detection
    _dc_pheromone_content_pat='test signal|demo focus|sanity signal|test area|test pheromone'
    _dc_pheromone_id_pat='^(test_|demo_)'
    _dc_queen_line_pat='test_pattern_|test_decree_|test_evolution_|Test Pattern:|TEST DECREE:|Test Evolution:'
    _dc_obs_colony_pat='^(test-colony|test|different-colony|alpha-colony|beta-colony|gamma-colony|c1|c2)$'
    _dc_spawn_name_pat='TestAnt|Test-Worker|test-worker|Bolt-99'
    _dc_constraint_pat='test area|demo|sanity'

    # Counts
    _dc_phero_count=0
    _dc_queen_count=0
    _dc_obs_count=0
    _dc_midden_count=0
    _dc_spawn_count=0
    _dc_constraint_count=0

    # --- 1. Pheromones ---
    _dc_phero_file="$COLONY_DATA_DIR/pheromones.json"
    if [[ -f "$_dc_phero_file" ]] && jq -e . "$_dc_phero_file" >/dev/null 2>&1; then  # SUPPRESS:OK -- validation: testing JSON validity
      _dc_phero_count=$(jq --arg cpat "$_dc_pheromone_content_pat" --arg ipat "$_dc_pheromone_id_pat" '
        [.signals // [] | .[] | select(
          (.content.text // .content // "" | tostring | test($cpat; "i")) or
          (.id // "" | test($ipat))
        )] | length
      ' "$_dc_phero_file" 2>/dev/null || echo 0)  # SUPPRESS:OK -- read-default: file may not exist yet
    fi

    # --- 2. QUEEN.md ---
    _dc_queen_file="$AETHER_ROOT/.aether/QUEEN.md"
    if [[ -f "$_dc_queen_file" ]]; then
      # SUPPRESS:OK -- existence-test: grep returns 1 when no matches
      _dc_queen_count=$(grep -ciE "$_dc_queen_line_pat" "$_dc_queen_file" 2>/dev/null) || _dc_queen_count=0
    fi

    # --- 3. Learning observations ---
    _dc_obs_file="$COLONY_DATA_DIR/learning-observations.json"
    if [[ -f "$_dc_obs_file" ]] && jq -e . "$_dc_obs_file" >/dev/null 2>&1; then  # SUPPRESS:OK -- validation: testing JSON validity
      _dc_obs_count=$(jq --arg cpat "$_dc_obs_colony_pat" '
        [.observations // [] | .[] | select(.colony_id // "" | test($cpat))] | length
      ' "$_dc_obs_file" 2>/dev/null || echo 0)  # SUPPRESS:OK -- read-default: file may not exist yet
    fi

    # --- 4. Midden ---
    _dc_midden_file="$COLONY_DATA_DIR/midden/midden.json"
    _dc_midden_signal_count=0
    _dc_midden_entry_count=0
    if [[ -f "$_dc_midden_file" ]] && jq -e . "$_dc_midden_file" >/dev/null 2>&1; then  # SUPPRESS:OK -- validation: testing JSON validity
      _dc_midden_signal_count=$(jq --arg cpat "$_dc_pheromone_content_pat" '
        [.signals // [] | .[] | select(
          (.content.text // .content // "" | tostring | test($cpat; "i"))
        )] | length
      ' "$_dc_midden_file" 2>/dev/null || echo 0)  # SUPPRESS:OK -- read-default: file may not exist yet
      _dc_midden_entry_count=$(jq --arg cpat "$_dc_obs_colony_pat" '
        [.entries // .failures // [] | .[] | select(.colony_id // "" | test($cpat))] | length
      ' "$_dc_midden_file" 2>/dev/null || echo 0)  # SUPPRESS:OK -- read-default: file may not exist yet
      _dc_midden_count=$(( _dc_midden_signal_count + _dc_midden_entry_count ))
    fi

    # --- 5. Spawn tree ---
    _dc_spawn_file="$COLONY_DATA_DIR/spawn-tree.txt"
    if [[ -f "$_dc_spawn_file" ]]; then
      # SUPPRESS:OK -- existence-test: grep returns 1 when no matches
      _dc_spawn_count=$(grep -cE "$_dc_spawn_name_pat" "$_dc_spawn_file" 2>/dev/null) || _dc_spawn_count=0
    fi

    # --- 6. Constraints ---
    _dc_constraint_file="$COLONY_DATA_DIR/constraints.json"
    if [[ -f "$_dc_constraint_file" ]] && jq -e . "$_dc_constraint_file" >/dev/null 2>&1; then  # SUPPRESS:OK -- validation: testing JSON validity
      _dc_constraint_count=$(jq --arg cpat "$_dc_constraint_pat" '
        [.focus // [] | .[] | select(
          (.content // .area // "" | tostring | test($cpat; "i"))
        )] | length
      ' "$_dc_constraint_file" 2>/dev/null || echo 0)  # SUPPRESS:OK -- read-default: file may not exist yet
    fi

    _dc_total=$(( _dc_phero_count + _dc_queen_count + _dc_obs_count + _dc_midden_count + _dc_spawn_count + _dc_constraint_count ))

    # --- Dry-run output ---
    if [[ "$_dc_confirm" == "false" ]]; then
      if [[ "$_dc_json" == "true" ]]; then
        json_ok "$(jq -n \
          --argjson phero "$_dc_phero_count" \
          --argjson queen "$_dc_queen_count" \
          --argjson obs "$_dc_obs_count" \
          --argjson midden "$_dc_midden_count" \
          --argjson spawn "$_dc_spawn_count" \
          --argjson constraints "$_dc_constraint_count" \
          --argjson total "$_dc_total" \
          '{mode:"dry-run", pheromones:$phero, queen:$queen, observations:$obs, midden:$midden, spawn_tree:$spawn, constraints:$constraints, total:$total}')"
      else
        cat <<DRYRUN_EOF
Data Clean — Artifact Scan
==========================

pheromones.json:
  Found: $_dc_phero_count test signals

QUEEN.md:
  Found: $_dc_queen_count test entries

learning-observations.json:
  Found: $_dc_obs_count test observations

midden.json:
  Found: $_dc_midden_count test entries

spawn-tree.txt:
  Found: $_dc_spawn_count test worker lines

constraints.json:
  Found: $_dc_constraint_count test focus entries

Total artifacts: $_dc_total
DRYRUN_EOF
        if [[ "$_dc_total" -gt 0 ]]; then
          echo ""
          echo "Run with --confirm to remove these artifacts."
        else
          echo ""
          echo "Colony data is clean. No artifacts found."
        fi
      fi
      exit 0
    fi

    # --- Confirm mode: actually clean ---
    _dc_removed_phero=0
    _dc_removed_queen=0
    _dc_removed_obs=0
    _dc_removed_midden=0
    _dc_removed_spawn=0
    _dc_removed_constraints=0

    # 1. Clean pheromones.json
    if [[ "$_dc_phero_count" -gt 0 ]] && [[ -f "$_dc_phero_file" ]]; then
      _dc_removed_phero=$_dc_phero_count
      _dc_cleaned=$(jq --arg cpat "$_dc_pheromone_content_pat" --arg ipat "$_dc_pheromone_id_pat" '
        .signals = [.signals // [] | .[] | select(
          ((.content.text // .content // "" | tostring | test($cpat; "i")) or
           (.id // "" | test($ipat))) | not
        )]
      ' "$_dc_phero_file") || {
        _aether_log_error "Could not process pheromone data for cleaning"
      }
      if [[ -n "$_dc_cleaned" && "$_dc_cleaned" != "null" ]]; then
        atomic_write "$_dc_phero_file" "$_dc_cleaned" || _aether_log_error "Could not save cleaned pheromone data"
      else
        _aether_log_error "Pheromone cleaning produced empty result -- not overwriting"
      fi
    fi

    # 2. Clean QUEEN.md
    if [[ "$_dc_queen_count" -gt 0 ]] && [[ -f "$_dc_queen_file" ]]; then
      _dc_removed_queen=$_dc_queen_count
      # SUPPRESS:OK -- read-default: grep returns 1 when all lines match (empty result is valid)
      _dc_cleaned=$(grep -viE "$_dc_queen_line_pat" "$_dc_queen_file" || true)
      atomic_write "$_dc_queen_file" "$_dc_cleaned" || _aether_log_error "Could not save cleaned QUEEN data"
    fi

    # 3. Clean learning-observations.json
    if [[ "$_dc_obs_count" -gt 0 ]] && [[ -f "$_dc_obs_file" ]]; then
      _dc_removed_obs=$_dc_obs_count
      _dc_cleaned=$(jq --arg cpat "$_dc_obs_colony_pat" '
        .observations = [.observations // [] | .[] | select((.colony_id // "" | test($cpat)) | not)]
      ' "$_dc_obs_file") || {
        _aether_log_error "Could not process observations data for cleaning"
      }
      if [[ -n "$_dc_cleaned" && "$_dc_cleaned" != "null" ]]; then
        atomic_write "$_dc_obs_file" "$_dc_cleaned" || _aether_log_error "Could not save cleaned observations data"
      else
        _aether_log_error "Observations cleaning produced empty result -- not overwriting"
      fi
    fi

    # 4. Clean midden.json
    if [[ "$_dc_midden_count" -gt 0 ]] && [[ -f "$_dc_midden_file" ]]; then
      _dc_removed_midden=$_dc_midden_count
      _dc_cleaned=$(jq --arg cpat "$_dc_pheromone_content_pat" --arg ecpat "$_dc_obs_colony_pat" '
        .signals = [.signals // [] | .[] | select(
          ((.content.text // .content // "" | tostring | test($cpat; "i"))) | not
        )] |
        .entries = [.entries // .failures // [] | .[] | select((.colony_id // "" | test($ecpat)) | not)]
      ' "$_dc_midden_file") || {
        _aether_log_error "Could not process midden data for cleaning"
      }
      if [[ -n "$_dc_cleaned" && "$_dc_cleaned" != "null" ]]; then
        atomic_write "$_dc_midden_file" "$_dc_cleaned" || _aether_log_error "Could not save cleaned midden data"
      else
        _aether_log_error "Midden cleaning produced empty result -- not overwriting"
      fi
    fi

    # 5. Clean spawn-tree.txt
    if [[ "$_dc_spawn_count" -gt 0 ]] && [[ -f "$_dc_spawn_file" ]]; then
      _dc_removed_spawn=$_dc_spawn_count
      # SUPPRESS:OK -- read-default: grep returns 1 when all lines match (empty result is valid)
      _dc_cleaned=$(grep -vE "$_dc_spawn_name_pat" "$_dc_spawn_file" || true)
      atomic_write "$_dc_spawn_file" "$_dc_cleaned" || _aether_log_error "Could not save cleaned spawn tree"
    fi

    # 6. Clean constraints.json
    if [[ "$_dc_constraint_count" -gt 0 ]] && [[ -f "$_dc_constraint_file" ]]; then
      _dc_removed_constraints=$_dc_constraint_count
      _dc_cleaned=$(jq --arg cpat "$_dc_constraint_pat" '
        .focus = [.focus // [] | .[] | select(
          ((.content // .area // "" | tostring | test($cpat; "i"))) | not
        )]
      ' "$_dc_constraint_file") || {
        _aether_log_error "Could not process constraints data for cleaning"
      }
      if [[ -n "$_dc_cleaned" && "$_dc_cleaned" != "null" ]]; then
        atomic_write "$_dc_constraint_file" "$_dc_cleaned" || _aether_log_error "Could not save cleaned constraints data"
      else
        _aether_log_error "Constraints cleaning produced empty result -- not overwriting"
      fi
    fi

    _dc_total_removed=$(( _dc_removed_phero + _dc_removed_queen + _dc_removed_obs + _dc_removed_midden + _dc_removed_spawn + _dc_removed_constraints ))

    json_ok "$(jq -n \
      --argjson phero "$_dc_removed_phero" \
      --argjson queen "$_dc_removed_queen" \
      --argjson obs "$_dc_removed_obs" \
      --argjson midden "$_dc_removed_midden" \
      --argjson spawn "$_dc_removed_spawn" \
      --argjson constraints "$_dc_removed_constraints" \
      --argjson total "$_dc_total_removed" \
      '{ok:true, removed:{pheromones:$phero, queen:$queen, observations:$obs, midden:$midden, spawn_tree:$spawn, constraints:$constraints}, total:$total}')"
    ;;

  # --- Autopilot State Tracking ---
  # Tracks /ant:run autopilot sessions in run-state.json (separate from COLONY_STATE.json)
  # Optional — colonies without /ant:run are unaffected

  autopilot-init)
    # Initialize autopilot run state
    # Usage: autopilot-init --total-phases N --start-phase N [--max-phases N]
    _ap_total_phases=""
    _ap_start_phase=""
    _ap_max_phases="null"

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --total-phases) _ap_total_phases="$2"; shift 2 ;;
        --start-phase) _ap_start_phase="$2"; shift 2 ;;
        --max-phases) _ap_max_phases="$2"; shift 2 ;;
        *) shift ;;
      esac
    done

    if [[ -z "$_ap_total_phases" || -z "$_ap_start_phase" ]]; then
      json_err "$E_VALIDATION_FAILED" "autopilot-init requires --total-phases and --start-phase"
    fi

    _ap_now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    _ap_state_file="$COLONY_DATA_DIR/run-state.json"

    _ap_state=$(jq -n \
      --arg version "1.0" \
      --arg status "running" \
      --arg started_at "$_ap_now" \
      --arg last_updated "$_ap_now" \
      --argjson total_phases "$_ap_total_phases" \
      --argjson start_phase "$_ap_start_phase" \
      --argjson current_phase "$_ap_start_phase" \
      --argjson max_phases "$_ap_max_phases" \
      '{
        version: $version,
        status: $status,
        started_at: $started_at,
        last_updated: $last_updated,
        phases_completed_in_run: 0,
        total_phases: $total_phases,
        start_phase: $start_phase,
        current_phase: $current_phase,
        max_phases: (if $max_phases == null then null else $max_phases end),
        pause_reason: null,
        last_action: null,
        total_auto_advanced: 0,
        phase_results: []
      }')

    atomic_write "$_ap_state_file" "$_ap_state"
    json_ok '{"created":"run-state.json"}'
    ;;

  autopilot-update)
    # Update autopilot run state after a phase action
    # Usage: autopilot-update --action build|continue|advance --phase N [--result success|failure]
    _ap_action=""
    _ap_phase=""
    _ap_result=""

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --action) _ap_action="$2"; shift 2 ;;
        --phase) _ap_phase="$2"; shift 2 ;;
        --result) _ap_result="$2"; shift 2 ;;
        *) shift ;;
      esac
    done

    _ap_state_file="$COLONY_DATA_DIR/run-state.json"

    if [[ ! -f "$_ap_state_file" ]]; then
      json_err "$E_FILE_NOT_FOUND" "run-state.json not found — autopilot not active"
    fi

    if [[ -z "$_ap_action" || -z "$_ap_phase" ]]; then
      json_err "$E_VALIDATION_FAILED" "autopilot-update requires --action and --phase"
    fi

    _ap_now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Build phase result entry if --result provided
    _ap_result_entry="null"
    if [[ -n "$_ap_result" ]]; then
      _ap_result_entry=$(jq -n \
        --argjson phase "$_ap_phase" \
        --arg action "$_ap_action" \
        --arg result "$_ap_result" \
        --arg timestamp "$_ap_now" \
        '{phase: $phase, action: $action, result: $result, timestamp: $timestamp}')
    fi

    # Update state atomically
    _ap_updated=$(jq \
      --arg last_updated "$_ap_now" \
      --arg last_action "$_ap_action" \
      --argjson current_phase "$_ap_phase" \
      --argjson result_entry "$_ap_result_entry" \
      --arg action "$_ap_action" \
      '
      .last_updated = $last_updated |
      .last_action = $last_action |
      .current_phase = $current_phase |
      (if $action == "advance" then .phases_completed_in_run += 1 else . end) |
      (if $action == "advance" then .total_auto_advanced += 1 else . end) |
      (if $result_entry != null then .phase_results += [$result_entry] else . end)
      ' "$_ap_state_file")

    atomic_write "$_ap_state_file" "$_ap_updated"
    json_ok '{"updated":true}'
    ;;

  autopilot-status)
    # Return current autopilot state
    # Usage: autopilot-status
    _ap_state_file="$COLONY_DATA_DIR/run-state.json"

    if [[ ! -f "$_ap_state_file" ]]; then
      json_ok '{"status":"not_active"}'
      exit 0
    fi

    json_ok "$(cat "$_ap_state_file")"
    ;;

  autopilot-stop)
    # Stop or complete an autopilot run
    # Usage: autopilot-stop --reason "why" [--status stopped|completed]
    _ap_reason=""
    _ap_stop_status="stopped"

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --reason) _ap_reason="$2"; shift 2 ;;
        --status) _ap_stop_status="$2"; shift 2 ;;
        *) shift ;;
      esac
    done

    _ap_state_file="$COLONY_DATA_DIR/run-state.json"

    if [[ ! -f "$_ap_state_file" ]]; then
      json_err "$E_FILE_NOT_FOUND" "run-state.json not found — autopilot not active"
    fi

    if [[ -z "$_ap_reason" ]]; then
      json_err "$E_VALIDATION_FAILED" "autopilot-stop requires --reason"
    fi

    _ap_now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    _ap_updated=$(jq \
      --arg status "$_ap_stop_status" \
      --arg reason "$_ap_reason" \
      --arg last_updated "$_ap_now" \
      '
      .status = $status |
      .pause_reason = $reason |
      .last_updated = $last_updated
      ' "$_ap_state_file")

    atomic_write "$_ap_state_file" "$_ap_updated"
    json_ok "{\"status\":\"$_ap_stop_status\"}"
    ;;

  autopilot-check-replan)
    # Check if a replan trigger should fire based on completed phases
    # Usage: autopilot-check-replan [--interval N]
    # Returns: {should_replan: bool, reason: string, learnings_since_last: number}
    _ap_interval=2

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --interval) _ap_interval="$2"; shift 2 ;;
        *) shift ;;
      esac
    done

    if [[ "$_ap_interval" -le 0 ]]; then
      json_err "$E_VALIDATION_FAILED" "autopilot-check-replan --interval must be > 0 (got $_ap_interval)"
    fi

    _ap_state_file="$COLONY_DATA_DIR/run-state.json"

    if [[ ! -f "$_ap_state_file" ]]; then
      json_err "$E_FILE_NOT_FOUND" "run-state.json not found — autopilot not active"
    fi

    # Read total_auto_advanced from run state (actual phase completions)
    _ap_auto_advanced=$(jq -r '.total_auto_advanced // 0' "$_ap_state_file")

    # MIGRATE: direct COLONY_STATE.json access -- use _state_read_field instead
    # Count learnings from COLONY_STATE.json
    _ap_colony_file="$DATA_DIR/COLONY_STATE.json"
    _ap_learnings_count=0
    if [[ -f "$_ap_colony_file" ]]; then
      # SUPPRESS:OK -- read-default: query may return empty
      _ap_learnings_count=$(jq '[.memory.phase_learnings[]?.learnings[]? // empty] | length' "$_ap_colony_file" 2>/dev/null || echo "0")
    fi

    # Check if replan should trigger: auto_advanced > 0 and is a multiple of interval
    _ap_should_replan="false"
    _ap_reason="No replan needed"

    if [[ "$_ap_auto_advanced" -gt 0 ]] && [[ $(( _ap_auto_advanced % _ap_interval )) -eq 0 ]]; then
      _ap_should_replan="true"
      _ap_reason="$_ap_auto_advanced phases auto-completed (replan interval: every $_ap_interval). $_ap_learnings_count learnings accumulated — consider /ant:plan to regenerate."
    fi

    json_ok "$(jq -n \
      --argjson should_replan "$_ap_should_replan" \
      --arg reason "$_ap_reason" \
      --argjson learnings_since_last "$_ap_learnings_count" \
      '{
        should_replan: $should_replan,
        reason: $reason,
        learnings_since_last: $learnings_since_last
      }')"
    ;;

  # ── Skills Engine ──────────────────────────────────────────────────────────
  skill-parse-frontmatter)
    _skill_parse_frontmatter "$1"
    ;;
  skill-index)
    _skill_build_index "${1:-}"
    ;;
  skill-index-read)
    _deprecation_warning "skill-index-read"
    _skill_read_index "${1:-}"
    ;;
  skill-detect)
    _skill_detect_codebase "${1:-.}" "${2:-}"
    ;;
  skill-match)
    _skill_match "$1" "${2:-}" "${3:-}"
    ;;
  skill-inject)
    _skill_inject "$1"
    ;;
  skill-list)
    _skill_list "${1:-}"
    ;;
  skill-manifest-read)
    _deprecation_warning "skill-manifest-read"
    _skill_manifest_read "$1"
    ;;
  skill-cache-rebuild)
    _sk_rebuild_dir="${1:-${AETHER_SKILLS_DIR:-$HOME/.aether/skills}}"
    rm -f "$_sk_rebuild_dir/.index.json"
    _skill_build_index "$_sk_rebuild_dir"
    ;;
  skill-diff)
    _skill_diff "$1" "${2:-}"
    ;;
  skill-is-user-created)
    _deprecation_warning "skill-is-user-created"
    _skill_is_user_created "$1" "$2"
    ;;

  # ── Smart Init ─────────────────────────────────────────────────────────────
  init-research)
    _scan_init_research "$@"
    ;;

  # ── Emoji Audit ─────────────────────────────────────────────────────────────
  emoji-audit)
    _emoji_audit_main "${1:-$(pwd)}"
    ;;

  *)
    json_err "$E_VALIDATION_FAILED" "Unknown command: $cmd"
    ;;
esac
