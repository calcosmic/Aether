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
AETHER_ROOT="$(cd "$SCRIPT_DIR/.." && pwd 2>/dev/null || echo "$SCRIPT_DIR")"
DATA_DIR="$AETHER_ROOT/.aether/data"

# Initialize lock state before sourcing (file-lock.sh trap needs these)
LOCK_ACQUIRED=${LOCK_ACQUIRED:-false}
CURRENT_LOCK=${CURRENT_LOCK:-""}

# Source shared infrastructure if available
[[ -f "$SCRIPT_DIR/utils/file-lock.sh" ]] && source "$SCRIPT_DIR/utils/file-lock.sh"
[[ -f "$SCRIPT_DIR/utils/atomic-write.sh" ]] && source "$SCRIPT_DIR/utils/atomic-write.sh"
[[ -f "$SCRIPT_DIR/utils/error-handler.sh" ]] && source "$SCRIPT_DIR/utils/error-handler.sh"
[[ -f "$SCRIPT_DIR/utils/chamber-utils.sh" ]] && source "$SCRIPT_DIR/utils/chamber-utils.sh"
[[ -f "$SCRIPT_DIR/utils/xml-utils.sh" ]] && source "$SCRIPT_DIR/utils/xml-utils.sh"

# Fallback error constants if error-handler.sh wasn't sourced
# This prevents "unbound variable" errors in older installations
: "${E_UNKNOWN:=E_UNKNOWN}"
: "${E_HUB_NOT_FOUND:=E_HUB_NOT_FOUND}"
: "${E_REPO_NOT_INITIALIZED:=E_REPO_NOT_INITIALIZED}"
: "${E_FILE_NOT_FOUND:=E_FILE_NOT_FOUND}"
: "${E_JSON_INVALID:=E_JSON_INVALID}"
: "${E_LOCK_FAILED:=E_LOCK_FAILED}"
: "${E_GIT_ERROR:=E_GIT_ERROR}"
: "${E_VALIDATION_FAILED:=E_VALIDATION_FAILED}"
: "${E_FEATURE_UNAVAILABLE:=E_FEATURE_UNAVAILABLE}"
: "${E_BASH_ERROR:=E_BASH_ERROR}"

# Feature detection for graceful degradation
# These checks run silently - failures are logged but don't block operation
if type feature_disable &>/dev/null; then
  # Check if DATA_DIR is writable for activity logging
  [[ -w "$DATA_DIR" ]] 2>/dev/null || feature_disable "activity_log" "DATA_DIR not writable"

  # Check if git is available for git integration
  command -v git &>/dev/null || feature_disable "git_integration" "git not installed"

  # Check if jq is available for JSON processing
  command -v jq &>/dev/null || feature_disable "json_processing" "jq not installed"

  # Check if lock utilities are available
  [[ -f "$SCRIPT_DIR/utils/file-lock.sh" ]] || feature_disable "file_locking" "lock utilities not available"
fi

# Fallback atomic_write if not sourced (uses temp file + mv for true atomicity)
if ! type atomic_write &>/dev/null; then
  atomic_write() {
    local target="$1"
    local content="$2"
    local temp
    temp=$(mktemp)
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
  # Fallback: simple error format for backward compatibility
  json_err() {
    local message="${2:-$1}"
    printf '{"ok":false,"error":"%s"}\n' "$message" >&2
    exit 1
  }
fi

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
    *) echo "🐜" ;;
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

  ensure_context_dir() {
    local dir
    dir=$(dirname "$ctx_file")
    [[ -d "$dir" ]] || mkdir -p "$dir"
  }

  read_colony_state() {
    local state_file="${AETHER_ROOT:-.}/.aether/data/COLONY_STATE.json"
    if [[ -f "$state_file" ]]; then
      current_phase=$(jq -r '.current_phase // "unknown"' "$state_file" 2>/dev/null)
      milestone=$(jq -r '.milestone // "unknown"' "$state_file" 2>/dev/null)
      goal=$(jq -r '.goal // ""' "$state_file" 2>/dev/null)
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
| In the Aether repo, \`.aether/\` IS the source of truth — \`runtime/\` is auto-populated on publish | CLAUDE.md | Permanent |
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
      json_ok "{\"updated\":true,\"action\":\"init\",\"file\":\"$ctx_file\"}"
      ;;

    update-phase)
      local new_phase="${2:-}"
      local new_phase_name="${3:-}"
      local safe_clear="${4:-NO}"
      local safe_reason="${5:-Phase in progress}"

      [[ -f "$ctx_file" ]] || { json_err "CONTEXT.md not found. Run context-update init first."; }

      sed -i.bak "s/| \*\*Last Updated\*\* | .*/| **Last Updated** | $ctx_ts |/" "$ctx_file" && rm -f "$ctx_file.bak"
      sed -i.bak "s/| \*\*Current Phase\*\* | .*/| **Current Phase** | $new_phase |/" "$ctx_file" && rm -f "$ctx_file.bak"
      sed -i.bak "s/| \*\*Phase Name\*\* | .*/| **Phase Name** | $new_phase_name |/" "$ctx_file" && rm -f "$ctx_file.bak"
      sed -i.bak "s/| \*\*Safe to Clear?\*\* | .*/| **Safe to Clear?** | $safe_clear — $safe_reason |/" "$ctx_file" && rm -f "$ctx_file.bak"

      json_ok "{\"updated\":true,\"action\":\"update-phase\",\"phase\":$new_phase}"
      ;;

    activity)
      local cmd="${2:-}"
      local result="${3:-}"
      local files_changed="${4:-—}"

      [[ -f "$ctx_file" ]] || { json_err "CONTEXT.md not found"; }

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
      json_ok "{\"updated\":true,\"action\":\"activity\",\"command\":\"$cmd\"}"
      ;;

    safe-to-clear)
      local safe="${2:-NO}"
      local reason="${3:-Unknown state}"

      [[ -f "$ctx_file" ]] || { json_err "CONTEXT.md not found"; }

      sed -i.bak "s/| \*\*Last Updated\*\* | .*/| **Last Updated** | $ctx_ts |/" "$ctx_file" && rm -f "$ctx_file.bak"
      sed -i.bak "s/| \*\*Safe to Clear?\*\* | .*/| **Safe to Clear?** | $safe — $reason |/" "$ctx_file" && rm -f "$ctx_file.bak"

      json_ok "{\"updated\":true,\"action\":\"safe-to-clear\",\"safe\":\"$safe\"}"
      ;;

    constraint)
      local c_type="${2:-}"
      local c_message="${3:-}"
      local c_source="${4:-User}"

      [[ -f "$ctx_file" ]] || { json_err "CONTEXT.md not found"; }

      sed -i.bak "s/| \*\*Last Updated\*\* | .*/| **Last Updated** | $ctx_ts |/" "$ctx_file" && rm -f "$ctx_file.bak"

      if [[ "$c_type" == "redirect" ]]; then
        sed -i.bak "/^## ⚠️ Active Constraints/,/^## /{ /^| Constraint |/a\\
| $c_message | $c_source | $ctx_ts |
}" "$ctx_file" && rm -f "$ctx_file.bak"
      elif [[ "$c_type" == "focus" ]]; then
        sed -i.bak "/^## 💭 Active Pheromones/,/^## /{ /^| Signal |/a\\
| FOCUS | $c_message | normal |
}" "$ctx_file" && rm -f "$ctx_file.bak"
      fi

      json_ok "{\"updated\":true,\"action\":\"constraint\",\"type\":\"$c_type\"}"
      ;;

    decision)
      local decision="${2:-}"
      local rationale="${3:-}"
      local made_by="${4:-Colony}"

      [[ -f "$ctx_file" ]] || { json_err "CONTEXT.md not found"; }

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
      json_ok "{\"updated\":true,\"action\":\"decision\"}"
      ;;

    build-start)
      local phase_id="${2:-}"
      local worker_count="${3:-0}"
      local tasks_count="${4:-0}"

      [[ -f "$ctx_file" ]] || { json_err "CONTEXT.md not found"; }

      sed -i.bak "s/| \*\*Last Updated\*\* | .*/| **Last Updated** | $ctx_ts |/" "$ctx_file" && rm -f "$ctx_file.bak"
      sed -i.bak "s/## 📍 What's In Progress/## 📍 What's In Progress\n\n**Phase $phase_id Build IN PROGRESS**\n- Workers: $worker_count | Tasks: $tasks_count\n- Started: $ctx_ts/" "$ctx_file" && rm -f "$ctx_file.bak"
      sed -i.bak "s/| \*\*Safe to Clear?\*\* | .*/| **Safe to Clear?** | ⚠️ NO — Build in progress |/" "$ctx_file" && rm -f "$ctx_file.bak"

      json_ok "{\"updated\":true,\"action\":\"build-start\",\"workers\":$worker_count}"
      ;;

    worker-spawn)
      local ant_name="${2:-}"
      local caste="${3:-}"
      local task="${4:-}"

      [[ -f "$ctx_file" ]] || { json_err "CONTEXT.md not found"; }

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

      json_ok "{\"updated\":true,\"action\":\"worker-spawn\",\"ant\":\"$ant_name\"}"
      ;;

    worker-complete)
      local ant_name="${2:-}"
      local status="${3:-completed}"

      [[ -f "$ctx_file" ]] || { json_err "CONTEXT.md not found"; }

      sed -i.bak "s/- .*$ant_name .*$/- $ant_name: $status (updated $ctx_ts)/" "$ctx_file" && rm -f "$ctx_file.bak"

      json_ok "{\"updated\":true,\"action\":\"worker-complete\",\"ant\":\"$ant_name\"}"
      ;;

    build-progress)
      local completed="${2:-0}"
      local total="${3:-1}"
      local percentage=$(( completed * 100 / total ))

      [[ -f "$ctx_file" ]] || { json_err "CONTEXT.md not found"; }

      sed -i.bak "s/Build IN PROGRESS/Build IN PROGRESS ($percentage% complete)/" "$ctx_file" && rm -f "$ctx_file.bak"

      json_ok "{\"updated\":true,\"action\":\"build-progress\",\"percent\":$percentage}"
      ;;

    build-complete)
      local status="${2:-completed}"
      local result="${3:-success}"

      [[ -f "$ctx_file" ]] || { json_err "CONTEXT.md not found"; }

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

      json_ok "{\"updated\":true,\"action\":\"build-complete\",\"status\":\"$status\"}"
      ;;

    *)
      json_err "$E_VALIDATION_FAILED" "Unknown context action: '$ctx_action'. Suggestion: Use one of: init, update-phase, activity, constraint, decision, safe-to-clear, build-start, worker-spawn, worker-complete, build-progress, build-complete"
      ;;
  esac
}

# --- Subcommand dispatch ---
cmd="${1:-help}"
shift 2>/dev/null || true

case "$cmd" in
  help)
    cat <<'EOF'
{"ok":true,"commands":["help","version","validate-state","load-state","unload-state","error-add","error-pattern-check","error-summary","activity-log","activity-log-init","activity-log-read","learning-promote","learning-inject","generate-ant-name","spawn-log","spawn-complete","spawn-can-spawn","spawn-get-depth","spawn-tree-load","spawn-tree-active","spawn-tree-depth","update-progress","check-antipattern","error-flag-pattern","signature-scan","signature-match","flag-add","flag-check-blockers","flag-resolve","flag-acknowledge","flag-list","flag-auto-resolve","autofix-checkpoint","autofix-rollback","spawn-can-spawn-swarm","swarm-findings-init","swarm-findings-add","swarm-findings-read","swarm-solution-set","swarm-cleanup","swarm-activity-log","swarm-display-init","swarm-display-update","swarm-display-get","swarm-display-text","swarm-timing-start","swarm-timing-get","swarm-timing-eta","view-state-init","view-state-get","view-state-set","view-state-toggle","view-state-expand","view-state-collapse","grave-add","grave-check","generate-commit-message","version-check","registry-add","bootstrap-system","model-profile","model-get","model-list","chamber-create","chamber-verify","chamber-list","milestone-detect","queen-init","queen-read","queen-promote","survey-load","survey-verify","pheromone-export","pheromone-write","pheromone-count","pheromone-read","instinct-read","pheromone-prime","pheromone-expire","eternal-init","pheromone-export-xml","pheromone-import-xml","pheromone-validate-xml","wisdom-export-xml","wisdom-import-xml","registry-export-xml","registry-import-xml"],"description":"Aether Colony Utility Layer — deterministic ops for the ant colony"}
EOF
    ;;
  version)
    json_ok '"1.0.0"'
    ;;
  validate-state)
    case "${1:-}" in
      colony)
        [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found" '{"file":"COLONY_STATE.json"}'
        json_ok "$(jq '
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
          ]} | . + {pass: ([.checks[] | select(. == "pass")] | length) == (.checks | length)}
        ' "$DATA_DIR/COLONY_STATE.json")"
        ;;
      constraints)
        [[ -f "$DATA_DIR/constraints.json" ]] || json_err "$E_FILE_NOT_FOUND" "constraints.json not found" '{"file":"constraints.json"}'
        json_ok "$(jq '
          def arr(f): if has(f) and (.[f]|type) == "array" then "pass" else "fail: \(f) not array" end;
          {file:"constraints.json", checks:[
            arr("focus"),
            arr("constraints")
          ]} | . + {pass: ([.checks[] | select(. == "pass")] | length) == (.checks | length)}
        ' "$DATA_DIR/constraints.json")"
        ;;
      all)
        results=()
        for target in colony constraints; do
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
  error-add)
    [[ $# -ge 3 ]] || json_err "$E_VALIDATION_FAILED" "Usage: error-add <category> <severity> <description> [phase]"
    [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found" '{"file":"COLONY_STATE.json"}'
    id="err_$(date -u +%s)_$(head -c 2 /dev/urandom | od -An -tx1 | tr -d ' ')"
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    phase_val="${4:-null}"
    if [[ "$phase_val" =~ ^[0-9]+$ ]]; then
      phase_jq="$phase_val"
    else
      phase_jq="null"
    fi
    updated=$(jq --arg id "$id" --arg cat "$1" --arg sev "$2" --arg desc "$3" --argjson phase "$phase_jq" --arg ts "$ts" '
      .errors.records += [{id:$id, category:$cat, severity:$sev, description:$desc, root_cause:null, phase:$phase, task_id:null, timestamp:$ts}] |
      if (.errors.records|length) > 50 then .errors.records = .errors.records[-50:] else . end
    ' "$DATA_DIR/COLONY_STATE.json") || json_err "$E_JSON_INVALID" "Failed to update COLONY_STATE.json"
    atomic_write "$DATA_DIR/COLONY_STATE.json" "$updated"
    json_ok "\"$id\""
    ;;
  error-pattern-check)
    [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found" '{"file":"COLONY_STATE.json"}'
    json_ok "$(jq '
      .errors.records | group_by(.category) | map(select(length >= 3) |
        {category: .[0].category, count: length,
         first_seen: (sort_by(.timestamp) | first.timestamp),
         last_seen: (sort_by(.timestamp) | last.timestamp)})
    ' "$DATA_DIR/COLONY_STATE.json")"
    ;;
  error-summary)
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

    log_file="$DATA_DIR/activity.log"
    mkdir -p "$DATA_DIR"
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
    log_file="$DATA_DIR/activity.log"
    mkdir -p "$DATA_DIR"
    archive_file="$DATA_DIR/activity-phase-${phase_num}.log"
    # Copy current log to per-phase archive (preserve combined log intact)
    if [ -f "$log_file" ] && [ -s "$log_file" ]; then
      # Handle retry scenario: don't overwrite existing archive
      if [ -f "$archive_file" ]; then
        archive_file="$DATA_DIR/activity-phase-${phase_num}-$(date -u +%s).log"
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

    log_file="$DATA_DIR/activity.log"
    [[ -f "$log_file" ]] || json_err "$E_FILE_NOT_FOUND" "activity.log not found" '{"file":"activity.log"}'
    if [ -n "$caste_filter" ]; then
      content=$(grep "$caste_filter" "$log_file" | tail -20)
    else
      content=$(cat "$log_file")
    fi
    json_ok "$(echo "$content" | jq -Rs '.')"
    ;;
  learning-promote)
    [[ $# -ge 3 ]] || json_err "$E_VALIDATION_FAILED" "Usage: learning-promote <content> <source_project> <source_phase> [tags]"
    content="$1"
    source_project="$2"
    source_phase="$3"
    tags="${4:-}"

    mkdir -p "$DATA_DIR"
    global_file="$DATA_DIR/learnings.json"

    if [[ ! -f "$global_file" ]]; then
      echo '{"learnings":[],"version":1}' > "$global_file"
    fi

    id="global_$(date -u +%s)_$(head -c 2 /dev/urandom | od -An -tx1 | tr -d ' ')"
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    if [[ -n "$tags" ]]; then
      tags_json=$(echo "$tags" | jq -R 'split(",")')
    else
      tags_json="[]"
    fi

    current_count=$(jq '.learnings | length' "$global_file")
    if [[ $current_count -ge 50 ]]; then
      json_ok "{\"promoted\":false,\"reason\":\"cap_reached\",\"current_count\":$current_count,\"cap\":50}"
      exit 0
    fi

    updated=$(jq --arg id "$id" --arg content "$content" --arg sp "$source_project" \
      --arg phase "$source_phase" --argjson tags "$tags_json" --arg ts "$ts" '
      .learnings += [{
        id: $id,
        content: $content,
        source_project: $sp,
        source_phase: $phase,
        tags: $tags,
        promoted_at: $ts
      }]
    ' "$global_file") || json_err "$E_JSON_INVALID" "Failed to update learnings.json"

    echo "$updated" > "$global_file"
    json_ok "{\"promoted\":true,\"id\":\"$id\",\"count\":$((current_count + 1)),\"cap\":50}"
    ;;
  learning-inject)
    [[ $# -ge 1 ]] || json_err "$E_VALIDATION_FAILED" "Usage: learning-inject <tech_keywords_csv>"
    keywords="$1"

    global_file="$DATA_DIR/learnings.json"

    if [[ ! -f "$global_file" ]]; then
      json_ok '{"learnings":[],"count":0}'
      exit 0
    fi

    json_ok "$(jq --arg kw "$keywords" '
      ($kw | split(",") | map(ascii_downcase | ltrimstr(" ") | rtrimstr(" "))) as $keywords |
      .learnings | map(
        select(
          .tags as $tags |
          ($keywords | any(. as $k | $tags | any(ascii_downcase | contains($k))))
        )
      ) | {learnings: ., count: length}
    ' "$global_file")"
    ;;
  spawn-log)
    # Usage: spawn-log <parent_id> <child_caste> <child_name> <task_summary> [model] [status]
    parent_id="${1:-}"
    child_caste="${2:-}"
    child_name="${3:-}"
    task_summary="${4:-}"
    model="${5:-default}"
    status="${6:-spawned}"
    [[ -z "$parent_id" || -z "$child_caste" || -z "$task_summary" ]] && json_err "$E_VALIDATION_FAILED" "Usage: spawn-log <parent_id> <child_caste> <child_name> <task_summary> [model] [status]"
    mkdir -p "$DATA_DIR"
    ts=$(date -u +"%H:%M:%S")
    ts_full=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    emoji=$(get_caste_emoji "$child_caste")
    parent_emoji=$(get_caste_emoji "$parent_id")
    # Log to activity log with spawn format, emojis, and model info
    echo "[$ts] ⚡ SPAWN $parent_emoji $parent_id -> $emoji $child_name ($child_caste): $task_summary [model: $model]" >> "$DATA_DIR/activity.log"
    # Log to spawn tree file for visualization (NEW FORMAT: includes model field)
    echo "$ts_full|$parent_id|$child_caste|$child_name|$task_summary|$model|$status" >> "$DATA_DIR/spawn-tree.txt"
    # Return emoji-formatted result for display
    json_ok "\"⚡ $emoji $child_name spawned\""
    ;;
  spawn-complete)
    # Usage: spawn-complete <ant_name> <status> [summary]
    ant_name="${1:-}"
    status="${2:-completed}"
    summary="${3:-}"
    [[ -z "$ant_name" ]] && json_err "$E_VALIDATION_FAILED" "Usage: spawn-complete <ant_name> <status> [summary]"
    mkdir -p "$DATA_DIR"
    ts=$(date -u +"%H:%M:%S")
    ts_full=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    emoji=$(get_caste_emoji "$ant_name")
    status_icon="✅"
    [[ "$status" == "failed" ]] && status_icon="❌"
    [[ "$status" == "blocked" ]] && status_icon="🚫"
    echo "[$ts] $status_icon $emoji $ant_name: $status${summary:+ - $summary}" >> "$DATA_DIR/activity.log"
    # Update spawn tree
    echo "$ts_full|$ant_name|$status|$summary" >> "$DATA_DIR/spawn-tree.txt"
    # Return emoji-formatted result for display
    json_ok "\"$status_icon $emoji $ant_name: ${summary:-$status}\""
    ;;
  spawn-can-spawn)
    # Check if spawning is allowed at given depth
    # Usage: spawn-can-spawn <depth>
    # Returns: {can_spawn: bool, depth: N, max_spawns: N, current_total: N}
    depth="${1:-1}"

    # Depth limits: 1→4 spawns, 2→2 spawns, 3+→0 spawns
    if [[ $depth -eq 1 ]]; then
      max_for_depth=4
    elif [[ $depth -eq 2 ]]; then
      max_for_depth=2
    else
      max_for_depth=0
    fi

    # Count current spawns in this session (from spawn-tree.txt)
    current=0
    if [[ -f "$DATA_DIR/spawn-tree.txt" ]]; then
      current=$(grep -c "|spawned$" "$DATA_DIR/spawn-tree.txt" 2>/dev/null || echo 0)
    fi

    # Global cap of 10 workers per phase
    global_cap=10

    # Can spawn if: depth < 3 AND under global cap
    if [[ $depth -lt 3 && $current -lt $global_cap ]]; then
      can="true"
    else
      can="false"
    fi

    json_ok "{\"can_spawn\":$can,\"depth\":$depth,\"max_spawns\":$max_for_depth,\"current_total\":$current,\"global_cap\":$global_cap}"
    ;;
  spawn-get-depth)
    # Return depth for a given ant name by tracing spawn tree
    # Usage: spawn-get-depth <ant_name>
    # Queen = depth 0, Queen's spawns = depth 1, their spawns = depth 2, etc.
    ant_name="${1:-Queen}"

    if [[ "$ant_name" == "Queen" ]]; then
      json_ok '{"ant":"Queen","depth":0}'
      exit 0
    fi

    # Check if spawn tree exists
    if [[ ! -f "$DATA_DIR/spawn-tree.txt" ]]; then
      json_ok "{\"ant\":\"$ant_name\",\"depth\":1,\"found\":false}"
      exit 0
    fi

    # Check if ant exists in spawn tree (gracefully handle missing ants)
    if ! grep -q "|$ant_name|" "$DATA_DIR/spawn-tree.txt" 2>/dev/null; then
      json_ok "{\"ant\":\"$ant_name\",\"depth\":1,\"found\":false}"
      exit 0
    fi

    # Find the spawn record for this ant and trace parents
    depth=1
    current_ant="$ant_name"

    # Find who spawned this ant (look for lines with |spawned)
    while true; do
      # Format: timestamp|parent|caste|child_name|task|spawned
      parent=$(grep "|$current_ant|" "$DATA_DIR/spawn-tree.txt" 2>/dev/null | grep "|spawned$" | head -1 | cut -d'|' -f2 || echo "")

      if [[ -z "$parent" || "$parent" == "Queen" ]]; then
        break
      fi

      depth=$((depth + 1))
      current_ant="$parent"

      # Safety limit
      if [[ $depth -gt 5 ]]; then
        break
      fi
    done

    json_ok "{\"ant\":\"$ant_name\",\"depth\":$depth,\"found\":true}"
    ;;
  update-progress)
    # Usage: update-progress <percent> <message> [phase] [total_phases]
    percent="${1:-0}"
    message="${2:-Working...}"
    phase="${3:-1}"
    total="${4:-1}"
    mkdir -p "$DATA_DIR"

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
    cat > "$DATA_DIR/watch-progress.txt" << EOF
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

    patterns_file="$DATA_DIR/error-patterns.json"
    mkdir -p "$DATA_DIR"

    if [[ ! -f "$patterns_file" ]]; then
      echo '{"patterns":[],"version":1}' > "$patterns_file"
    fi

    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    project_name=$(basename "$PWD")

    # Check if pattern already exists
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
      echo "$updated" > "$patterns_file"
      count=$(echo "$updated" | jq --arg name "$pattern_name" '.patterns[] | select(.name == $name) | .occurrences')
      json_ok "{\"updated\":true,\"pattern\":\"$pattern_name\",\"occurrences\":$count}"
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
      echo "$updated" > "$patterns_file"
      json_ok "{\"created\":true,\"pattern\":\"$pattern_name\"}"
    fi
    ;;
  error-patterns-check)
    # Check for known error patterns in a file or codebase
    # Returns patterns that should be avoided
    global_file="$DATA_DIR/error-patterns.json"

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
        if grep -n "didSet" "$file_path" 2>/dev/null | grep -q "self\."; then
          line=$(grep -n "didSet" "$file_path" | grep "self\." | head -1 | cut -d: -f1)
          criticals+=("{\"pattern\":\"didSet-recursion\",\"file\":\"$file_path\",\"line\":$line,\"message\":\"Potential didSet infinite recursion - self assignment in didSet\"}")
        fi
        ;;
      ts|tsx|js|jsx)
        # TypeScript any type check
        if grep -nE '\bany\b' "$file_path" 2>/dev/null | grep -qv "//.*any"; then
          count=$(grep -cE '\bany\b' "$file_path" 2>/dev/null || echo "0")
          warnings+=("{\"pattern\":\"typescript-any\",\"file\":\"$file_path\",\"count\":$count,\"message\":\"Found $count uses of 'any' type\"}")
        fi
        # Console.log in production code (not in test files)
        if [[ ! "$file_path" =~ \.test\. && ! "$file_path" =~ \.spec\. ]]; then
          if grep -n "console\.log" "$file_path" 2>/dev/null | grep -qv "//"; then
            count=$(grep -c "console\.log" "$file_path" 2>/dev/null || echo "0")
            warnings+=("{\"pattern\":\"console-log\",\"file\":\"$file_path\",\"count\":$count,\"message\":\"Found $count console.log statements\"}")
          fi
        fi
        ;;
      py)
        # Python bare except
        if grep -n "except:" "$file_path" 2>/dev/null | grep -qv "#"; then
          line=$(grep -n "except:" "$file_path" | head -1 | cut -d: -f1)
          warnings+=("{\"pattern\":\"bare-except\",\"file\":\"$file_path\",\"line\":$line,\"message\":\"Bare except clause - specify exception type\"}")
        fi
        ;;
    esac

    # Common patterns across all languages
    # Exposed secrets check (critical)
    if grep -nE "(api_key|apikey|secret|password|token)\s*=\s*['\"][^'\"]+['\"]" "$file_path" 2>/dev/null | grep -qvi "example\|test\|mock\|fake"; then
      line=$(grep -nE "(api_key|apikey|secret|password|token)\s*=\s*['\"]" "$file_path" | head -1 | cut -d: -f1)
      criticals+=("{\"pattern\":\"exposed-secret\",\"file\":\"$file_path\",\"line\":${line:-0},\"message\":\"Potential hardcoded secret or credential\"}")
    fi

    # TODO/FIXME check (warning)
    if grep -nE "(TODO|FIXME|XXX|HACK)" "$file_path" 2>/dev/null | head -1 | grep -q .; then
      count=$(grep -cE "(TODO|FIXME|XXX|HACK)" "$file_path" 2>/dev/null || echo "0")
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
    signatures_file="$DATA_DIR/signatures.json"
    if [[ ! -f "$signatures_file" ]]; then
      json_ok '{"found":false,"signature":null}'
      exit 0
    fi

    # Extract signature details using jq
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
    if grep -q -- "$pattern_string" "$target_file" 2>/dev/null; then
      # Match found - return signature details with match info
      match_count=$(grep -c -- "$pattern_string" "$target_file" 2>/dev/null || echo "1")
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
    signatures_file="$DATA_DIR/signatures.json"
    [[ ! -f "$signatures_file" ]] && json_err "$E_FILE_NOT_FOUND" "Signatures file not found"

    # Read high-confidence signatures (confidence >= 0.7) using jq -c for compact single-line output
    high_conf_signatures=$(jq -c '.signatures[] | select(.confidence_threshold >= 0.7)' "$signatures_file" 2>/dev/null)

    # Check if any high-confidence signatures exist
    sig_count=$(echo "$high_conf_signatures" | grep -c '{' || echo 0)
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
      done < <(find "$target_dir" -type f -name "$file_pattern" -print0 2>/dev/null || true)
    else
      # Default: match common code file types
      while IFS= read -r -d '' file; do
        files+=("$file")
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
        if grep -q -- "$sig_pattern" "$file" 2>/dev/null; then
          match_count=$(grep -c -- "$sig_pattern" "$file" 2>/dev/null || echo "1")

          # Add to results
          matches_for_file=$(echo "$matches_for_file" | jq --arg n "$sig_name" --arg d "$sig_desc" --argjson c "$sig_conf" --argjson m "$match_count" \
            '. += [{"name":$n,"description":$d,"confidence_threshold":$c,"match_count":$m}]')
        fi
      done < <(echo "$high_conf_signatures" | jq -c '.' 2>/dev/null || true)

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
  flag-add)
    # Add a project-specific flag (blocker, issue, or note)
    # Usage: flag-add <type> <title> <description> [source] [phase]
    # Types: blocker (critical, blocks advancement), issue (high, warning), note (low, info)
    type="${1:-issue}"
    title="${2:-}"
    desc="${3:-}"
    source="${4:-manual}"
    phase="${5:-null}"
    [[ -z "$title" ]] && json_err "$E_VALIDATION_FAILED" "Usage: flag-add <type> <title> <description> [source] [phase]"

    mkdir -p "$DATA_DIR"
    flags_file="$DATA_DIR/flags.json"

    if [[ ! -f "$flags_file" ]]; then
      echo '{"version":1,"flags":[]}' > "$flags_file"
    fi

    id="flag_$(date -u +%s)_$(head -c 2 /dev/urandom | od -An -tx1 | tr -d ' ')"
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Acquire lock for atomic flag update (degrade gracefully if locking unavailable)
    lock_acquired=false
    if type feature_enabled &>/dev/null && ! feature_enabled "file_locking"; then
      json_warn "W_DEGRADED" "File locking disabled - proceeding without lock: $(type _feature_reason &>/dev/null && _feature_reason file_locking || echo 'unknown')"
    else
      acquire_lock "$flags_file" || {
        if type json_err &>/dev/null; then
          json_err "$E_LOCK_FAILED" "Failed to acquire lock on flags.json"
        else
          echo '{"ok":false,"error":"Failed to acquire lock on flags.json"}' >&2
          exit 1
        fi
      }
      lock_acquired=true
      # Ensure lock is always released on exit (BUG-002 fix)
      trap 'release_lock "$flags_file" 2>/dev/null || true' EXIT
    fi

    # Map type to severity
    case "$type" in
      blocker)  severity="critical" ;;
      issue)    severity="high" ;;
      note)     severity="low" ;;
      *)        severity="medium" ;;
    esac

    # Handle phase as number or null
    if [[ "$phase" =~ ^[0-9]+$ ]]; then
      phase_jq="$phase"
    else
      phase_jq="null"
    fi

    updated=$(jq --arg id "$id" --arg type "$type" --arg sev "$severity" \
      --arg title "$title" --arg desc "$desc" --arg source "$source" \
      --argjson phase "$phase_jq" --arg ts "$ts" '
      .flags += [{
        id: $id,
        type: $type,
        severity: $sev,
        title: $title,
        description: $desc,
        source: $source,
        phase: $phase,
        created_at: $ts,
        acknowledged_at: null,
        resolved_at: null,
        resolution: null,
        auto_resolve_on: (if $type == "blocker" and ($source | test("chaos") | not) then "build_pass" else null end)
      }]
    ' "$flags_file") || { json_err "$E_JSON_INVALID" "Failed to add flag"; }

    atomic_write "$flags_file" "$updated"
    # Lock released by trap on exit (BUG-002 fix)
    trap - EXIT
    json_ok "{\"id\":\"$id\",\"type\":\"$type\",\"severity\":\"$severity\"}"
    ;;
  flag-check-blockers)
    # Count unresolved blockers for the current phase
    # Usage: flag-check-blockers [phase]
    phase="${1:-}"
    flags_file="$DATA_DIR/flags.json"

    if [[ ! -f "$flags_file" ]]; then
      json_ok '{"blockers":0,"issues":0,"notes":0}'
      exit 0
    fi

    if [[ -n "$phase" && "$phase" =~ ^[0-9]+$ ]]; then
      # Filter by phase
      result=$(jq --argjson phase "$phase" '{
        blockers: [.flags[] | select(.type == "blocker" and .resolved_at == null and (.phase == $phase or .phase == null))] | length,
        issues: [.flags[] | select(.type == "issue" and .resolved_at == null and (.phase == $phase or .phase == null))] | length,
        notes: [.flags[] | select(.type == "note" and .resolved_at == null and (.phase == $phase or .phase == null))] | length
      }' "$flags_file")
    else
      # All unresolved
      result=$(jq '{
        blockers: [.flags[] | select(.type == "blocker" and .resolved_at == null)] | length,
        issues: [.flags[] | select(.type == "issue" and .resolved_at == null)] | length,
        notes: [.flags[] | select(.type == "note" and .resolved_at == null)] | length
      }' "$flags_file")
    fi

    json_ok "$result"
    ;;
  flag-resolve)
    # Resolve a flag with optional resolution message
    # Usage: flag-resolve <flag_id> [resolution_message]
    flag_id="${1:-}"
    resolution="${2:-Resolved}"
    [[ -z "$flag_id" ]] && json_err "$E_VALIDATION_FAILED" "Usage: flag-resolve <flag_id> [resolution_message]"

    flags_file="$DATA_DIR/flags.json"
    [[ ! -f "$flags_file" ]] && json_err "$E_FILE_NOT_FOUND" "No flags file found"

    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Acquire lock for atomic flag update (degrade gracefully if locking unavailable)
    if type feature_enabled &>/dev/null && ! feature_enabled "file_locking"; then
      json_warn "W_DEGRADED" "File locking disabled - proceeding without lock"
    else
      acquire_lock "$flags_file" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on flags.json"
    fi

    updated=$(jq --arg id "$flag_id" --arg res "$resolution" --arg ts "$ts" '
      .flags = [.flags[] | if .id == $id then
        .resolved_at = $ts |
        .resolution = $res
      else . end]
    ' "$flags_file") || {
      if type feature_enabled &>/dev/null && feature_enabled "file_locking"; then
        release_lock "$flags_file"
      fi
      json_err "$E_JSON_INVALID" "Failed to resolve flag"
    }

    atomic_write "$flags_file" "$updated"
    release_lock "$flags_file"
    json_ok "{\"resolved\":\"$flag_id\"}"
    ;;
  flag-acknowledge)
    # Acknowledge a flag (issue continues but noted)
    # Usage: flag-acknowledge <flag_id>
    flag_id="${1:-}"
    [[ -z "$flag_id" ]] && json_err "$E_VALIDATION_FAILED" "Usage: flag-acknowledge <flag_id>"

    flags_file="$DATA_DIR/flags.json"
    [[ ! -f "$flags_file" ]] && json_err "$E_FILE_NOT_FOUND" "No flags file found"

    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Acquire lock for atomic flag update (degrade gracefully if locking unavailable)
    if type feature_enabled &>/dev/null && ! feature_enabled "file_locking"; then
      json_warn "W_DEGRADED" "File locking disabled - proceeding without lock"
    else
      acquire_lock "$flags_file" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on flags.json"
    fi

    updated=$(jq --arg id "$flag_id" --arg ts "$ts" '
      .flags = [.flags[] | if .id == $id then
        .acknowledged_at = $ts
      else . end]
    ' "$flags_file") || {
      if type feature_enabled &>/dev/null && feature_enabled "file_locking"; then
        release_lock "$flags_file"
      fi
      json_err "$E_JSON_INVALID" "Failed to acknowledge flag"
    }

    atomic_write "$flags_file" "$updated"
    release_lock "$flags_file"
    json_ok "{\"acknowledged\":\"$flag_id\"}"
    ;;
  flag-list)
    # List flags, optionally filtered
    # Usage: flag-list [--all] [--type blocker|issue|note] [--phase N]
    flags_file="$DATA_DIR/flags.json"
    show_all="false"
    filter_type=""
    filter_phase=""

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --all) show_all="true"; shift ;;
        --type) filter_type="$2"; shift 2 ;;
        --phase) filter_phase="$2"; shift 2 ;;
        *) shift ;;
      esac
    done

    if [[ ! -f "$flags_file" ]]; then
      json_ok '{"flags":[],"count":0}'
      exit 0
    fi

    # Build jq filter
    jq_filter='.flags'

    if [[ "$show_all" != "true" ]]; then
      jq_filter+=' | [.[] | select(.resolved_at == null)]'
    fi

    if [[ -n "$filter_type" ]]; then
      jq_filter+=" | [.[] | select(.type == \"$filter_type\")]"
    fi

    if [[ -n "$filter_phase" && "$filter_phase" =~ ^[0-9]+$ ]]; then
      jq_filter+=" | [.[] | select(.phase == $filter_phase or .phase == null)]"
    fi

    result=$(jq "{flags: ($jq_filter), count: ($jq_filter | length)}" "$flags_file")
    json_ok "$result"
    ;;
  flag-auto-resolve)
    # Auto-resolve flags based on trigger (e.g., build_pass)
    # Usage: flag-auto-resolve <trigger>
    trigger="${1:-build_pass}"
    flags_file="$DATA_DIR/flags.json"

    if [[ ! -f "$flags_file" ]]; then json_ok '{"resolved":0}'; exit 0; fi

    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Acquire lock for atomic flag update (degrade gracefully if locking unavailable)
    lock_acquired=false
    if type feature_enabled &>/dev/null && ! feature_enabled "file_locking"; then
      json_warn "W_DEGRADED" "File locking disabled - proceeding without lock"
    else
      acquire_lock "$flags_file" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on flags.json"
      lock_acquired=true
      # Ensure lock is always released on exit (BUG-005/BUG-011 fix)
      trap 'release_lock "$flags_file" 2>/dev/null || true' EXIT
    fi

    # Count how many will be resolved
    count=$(jq --arg trigger "$trigger" '
      [.flags[] | select(.auto_resolve_on == $trigger and .resolved_at == null)] | length
    ' "$flags_file") || {
      json_err "$E_JSON_INVALID" "Failed to count flags for auto-resolve"
    }

    # Resolve them
    updated=$(jq --arg trigger "$trigger" --arg ts "$ts" '
      .flags = [.flags[] | if .auto_resolve_on == $trigger and .resolved_at == null then
        .resolved_at = $ts |
        .resolution = "Auto-resolved on " + $trigger
      else . end]
    ' "$flags_file") || {
      json_err "$E_JSON_INVALID" "Failed to auto-resolve flags"
    }

    atomic_write "$flags_file" "$updated"
    # Lock released by trap on exit (BUG-005/BUG-011 fix)
    if [[ "$lock_acquired" == "true" ]]; then
      trap - EXIT
    fi
    json_ok "{\"resolved\":$count,\"trigger\":\"$trigger\"}"
    ;;
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
    json_ok "\"$name\""
    ;;

  # ============================================
  # SWARM UTILITIES (ant:swarm support)
  # ============================================

  autofix-checkpoint)
    # Create checkpoint before applying auto-fix
    # Usage: autofix-checkpoint [label]
    # Returns: {type: "stash"|"commit"|"none", ref: "..."}
    # IMPORTANT: Only stash Aether-related files, never touch user work
    if git rev-parse --git-dir >/dev/null 2>&1; then
      # Check if there are changes to Aether-managed files only
      # Target directories that Aether is allowed to modify
      target_dirs=".aether .claude/commands/ant .claude/commands/st .opencode runtime bin"
      has_changes=false

      for dir in $target_dirs; do
        if [[ -d "$dir" ]] && [[ -n "$(git status --porcelain "$dir" 2>/dev/null)" ]]; then
          has_changes=true
          break
        fi
      done

      if [[ "$has_changes" == "true" ]]; then
        label="${1:-autofix-$(date +%s)}"
        stash_name="aether-checkpoint: $label"
        # Only stash Aether-managed directories, never touch user files
        if git stash push -m "$stash_name" -- $target_dirs >/dev/null 2>&1; then
          json_ok "{\"type\":\"stash\",\"ref\":\"$stash_name\"}"
        else
          # Stash failed (possibly due to conflicts), record commit hash
          hash=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
          json_ok "{\"type\":\"commit\",\"ref\":\"$hash\"}"
        fi
      else
        # No changes in Aether-managed directories, just record commit hash
        hash=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
        json_ok "{\"type\":\"commit\",\"ref\":\"$hash\"}"
      fi
    else
      json_ok '{"type":"none","ref":null}'
    fi
    ;;

  autofix-rollback)
    # Rollback from checkpoint if fix failed
    # Usage: autofix-rollback <type> <ref>
    # Returns: {rolled_back: bool, method: "stash"|"reset"|"none"}
    ref_type="${1:-none}"
    ref="${2:-}"

    case "$ref_type" in
      stash)
        # Find and pop the stash
        stash_ref=$(git stash list 2>/dev/null | grep "$ref" | head -1 | cut -d: -f1 || echo "")
        if [[ -n "$stash_ref" ]]; then
          if git stash pop "$stash_ref" >/dev/null 2>&1; then
            json_ok '{"rolled_back":true,"method":"stash"}'
          else
            json_ok '{"rolled_back":false,"method":"stash","error":"stash pop failed"}'
          fi
        else
          json_ok '{"rolled_back":false,"method":"stash","error":"stash not found"}'
        fi
        ;;
      commit)
        # Reset to the commit
        if [[ -n "$ref" && "$ref" != "unknown" ]]; then
          if git reset --hard "$ref" >/dev/null 2>&1; then
            json_ok '{"rolled_back":true,"method":"reset"}'
          else
            json_ok '{"rolled_back":false,"method":"reset","error":"reset failed"}'
          fi
        else
          json_ok '{"rolled_back":false,"method":"reset","error":"invalid ref"}'
        fi
        ;;
      none|*)
        json_ok '{"rolled_back":false,"method":"none"}'
        ;;
    esac
    ;;

  spawn-can-spawn-swarm)
    # Check if swarm can spawn more scouts (separate from phase workers)
    # Usage: spawn-can-spawn-swarm <swarm_id>
    # Swarm has its own cap of 6 (4 scouts + 2 sub-scouts max)
    swarm_id="${1:-swarm}"
    swarm_cap=6

    current=0
    if [[ -f "$DATA_DIR/spawn-tree.txt" ]]; then
      current=$(grep -c "|swarm:$swarm_id$" "$DATA_DIR/spawn-tree.txt" 2>/dev/null) || current=0
    fi

    if [[ $current -lt $swarm_cap ]]; then
      can="true"
      remaining=$((swarm_cap - current))
    else
      can="false"
      remaining=0
    fi

    json_ok "{\"can_spawn\":$can,\"current\":$current,\"cap\":$swarm_cap,\"remaining\":$remaining,\"swarm_id\":\"$swarm_id\"}"
    ;;

  swarm-findings-init)
    # Initialize swarm findings file
    # Usage: swarm-findings-init <swarm_id>
    swarm_id="${1:-swarm-$(date +%s)}"
    findings_file="$DATA_DIR/swarm-findings-$swarm_id.json"

    mkdir -p "$DATA_DIR"
    cat > "$findings_file" <<EOF
{
  "swarm_id": "$swarm_id",
  "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "status": "active",
  "findings": [],
  "solution": null
}
EOF
    json_ok "{\"swarm_id\":\"$swarm_id\",\"file\":\"$findings_file\"}"
    ;;

  swarm-findings-add)
    # Add a finding from a scout
    # Usage: swarm-findings-add <swarm_id> <scout_type> <confidence> <finding_json>
    swarm_id="${1:-}"
    scout_type="${2:-}"
    confidence="${3:-0.5}"
    finding="${4:-}"

    [[ -z "$swarm_id" || -z "$scout_type" || -z "$finding" ]] && json_err "$E_VALIDATION_FAILED" "Usage: swarm-findings-add <swarm_id> <scout_type> <confidence> <finding_json>"

    findings_file="$DATA_DIR/swarm-findings-$swarm_id.json"
    [[ ! -f "$findings_file" ]] && json_err "$E_FILE_NOT_FOUND" "Swarm findings file not found: $swarm_id"

    ts=$(date -u +%Y-%m-%dT%H:%M:%SZ)

    # Add finding to array
    updated=$(jq --arg scout "$scout_type" --arg conf "$confidence" --arg ts "$ts" --argjson finding "$finding" '
      .findings += [{
        "scout": $scout,
        "confidence": ($conf | tonumber),
        "timestamp": $ts,
        "finding": $finding
      }]
    ' "$findings_file")

    echo "$updated" > "$findings_file"
    count=$(echo "$updated" | jq '.findings | length')
    json_ok "{\"added\":true,\"scout\":\"$scout_type\",\"total_findings\":$count}"
    ;;

  swarm-findings-read)
    # Read all findings for a swarm
    # Usage: swarm-findings-read <swarm_id>
    swarm_id="${1:-}"
    [[ -z "$swarm_id" ]] && json_err "$E_VALIDATION_FAILED" "Usage: swarm-findings-read <swarm_id>"

    findings_file="$DATA_DIR/swarm-findings-$swarm_id.json"
    [[ ! -f "$findings_file" ]] && json_err "$E_FILE_NOT_FOUND" "Swarm findings file not found: $swarm_id"

    json_ok "$(cat "$findings_file")"
    ;;

  swarm-solution-set)
    # Set the chosen solution for a swarm
    # Usage: swarm-solution-set <swarm_id> <solution_json>
    swarm_id="${1:-}"
    solution="${2:-}"

    [[ -z "$swarm_id" || -z "$solution" ]] && json_err "$E_VALIDATION_FAILED" "Usage: swarm-solution-set <swarm_id> <solution_json>"

    findings_file="$DATA_DIR/swarm-findings-$swarm_id.json"
    [[ ! -f "$findings_file" ]] && json_err "$E_FILE_NOT_FOUND" "Swarm findings file not found: $swarm_id"

    ts=$(date -u +%Y-%m-%dT%H:%M:%SZ)

    updated=$(jq --argjson solution "$solution" --arg ts "$ts" '
      .solution = $solution |
      .status = "resolved" |
      .resolved_at = $ts
    ' "$findings_file")

    echo "$updated" > "$findings_file"
    json_ok "{\"solution_set\":true,\"swarm_id\":\"$swarm_id\"}"
    ;;

  swarm-cleanup)
    # Clean up swarm files after completion
    # Usage: swarm-cleanup <swarm_id> [--archive]
    swarm_id="${1:-}"
    archive="${2:-}"

    [[ -z "$swarm_id" ]] && json_err "$E_VALIDATION_FAILED" "Usage: swarm-cleanup <swarm_id> [--archive]"

    findings_file="$DATA_DIR/swarm-findings-$swarm_id.json"

    if [[ -f "$findings_file" ]]; then
      if [[ "$archive" == "--archive" ]]; then
        mkdir -p "$DATA_DIR/swarm-archive"
        mv "$findings_file" "$DATA_DIR/swarm-archive/"
        json_ok "{\"archived\":true,\"swarm_id\":\"$swarm_id\"}"
      else
        rm -f "$findings_file"
        json_ok "{\"deleted\":true,\"swarm_id\":\"$swarm_id\"}"
      fi
    else
      json_ok "{\"not_found\":true,\"swarm_id\":\"$swarm_id\"}"
    fi
    ;;

  grave-add)
    # Record a grave marker when a builder fails at a file
    # Usage: grave-add <file> <ant_name> <task_id> <phase> <failure_summary> [function] [line]
    [[ $# -ge 5 ]] || json_err "$E_VALIDATION_FAILED" "Usage: grave-add <file> <ant_name> <task_id> <phase> <failure_summary> [function] [line]"
    [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found" '{"file":"COLONY_STATE.json"}'
    file="$1"
    ant_name="$2"
    task_id="$3"
    phase="$4"
    failure_summary="$5"
    func="${6:-null}"
    line="${7:-null}"
    id="grave_$(date -u +%s)_$(head -c 2 /dev/urandom | od -An -tx1 | tr -d ' ')"
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    if [[ "$phase" =~ ^[0-9]+$ ]]; then
      phase_jq="$phase"
    else
      phase_jq="null"
    fi
    if [[ "$func" == "null" ]]; then
      func_jq="null"
    else
      func_jq="\"$func\""
    fi
    if [[ "$line" =~ ^[0-9]+$ ]]; then
      line_jq="$line"
    else
      line_jq="null"
    fi
    updated=$(jq --arg id "$id" --arg file "$file" --arg ant "$ant_name" --arg tid "$task_id" \
      --argjson phase "$phase_jq" --arg summary "$failure_summary" \
      --argjson func "$func_jq" --argjson line "$line_jq" --arg ts "$ts" '
      (.graveyards // []) as $graves |
      . + {graveyards: ($graves + [{
        id: $id,
        file: $file,
        ant_name: $ant,
        task_id: $tid,
        phase: $phase,
        failure_summary: $summary,
        function: $func,
        line: $line,
        timestamp: $ts
      }])} |
      if (.graveyards | length) > 30 then .graveyards = .graveyards[-30:] else . end
    ' "$DATA_DIR/COLONY_STATE.json") || json_err "$E_JSON_INVALID" "Failed to update COLONY_STATE.json"
    atomic_write "$DATA_DIR/COLONY_STATE.json" "$updated"
    json_ok "\"$id\""
    ;;

  grave-check)
    # Query for grave markers near a file path
    # Usage: grave-check <file_path>
    # Read-only, never modifies state
    [[ $# -ge 1 ]] || json_err "$E_VALIDATION_FAILED" "Usage: grave-check <file_path>"
    [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found" '{"file":"COLONY_STATE.json"}'
    check_file="$1"
    check_dir=$(dirname "$check_file")
    json_ok "$(jq --arg file "$check_file" --arg dir "$check_dir" '
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
    ' "$DATA_DIR/COLONY_STATE.json")"
    ;;

  # ============================================
  # GIT COMMIT UTILITIES
  # ============================================

  generate-commit-message)
    # Generate an intelligent commit message from colony context
    # Usage: generate-commit-message <type> <phase_id> <phase_name> [summary|ai_description] [plan_num]
    # Types: "milestone" | "pause" | "fix" | "contextual"
    # Returns: {"message": "...", "body": "...", "files_changed": N, ...}

    msg_type="${1:-milestone}"
    phase_id="${2:-0}"
    phase_name="${3:-unknown}"
    summary="${4:-}"        # For milestone/fix types, or ai_description for contextual type
    plan_num="${5:-01}"     # Optional: plan number for contextual type (e.g., "01")

    # Count changed files
    files_changed=0
    if git rev-parse --git-dir >/dev/null 2>&1; then
      files_changed=$(git diff --stat --cached HEAD 2>/dev/null | tail -1 | grep -oE '[0-9]+ file' | grep -oE '[0-9]+' || echo "0")
      if [[ "$files_changed" == "0" ]]; then
        files_changed=$(git status --porcelain 2>/dev/null | wc -l | tr -d ' ')
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
        json_ok "{\"message\":\"$message\",\"body\":\"$body\",\"files_changed\":$files_changed,\"subsystem\":\"$subsystem\",\"scope\":\"${phase_id}.${plan_num}\"}"
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

    json_ok "{\"message\":\"$message\",\"body\":\"$body\",\"files_changed\":$files_changed}"
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

    local_ver=$(jq -r '.version // "unknown"' "$local_version_file" 2>/dev/null || echo "unknown")
    hub_ver=$(jq -r '.version // "unknown"' "$hub_version_file" 2>/dev/null || echo "unknown")

    if [[ "$local_ver" == "$hub_ver" ]]; then
      json_ok '""'
    else
      printf -v msg 'Update available: %s to %s (run /ant:update)' "$local_ver" "$hub_ver"
      json_ok "$msg"
    fi
    ;;

  registry-add)
    # Add or update a repo entry in ~/.aether/registry.json
    # Usage: registry-add <repo_path> <version>
    repo_path="${1:-}"
    repo_version="${2:-}"
    [[ -z "$repo_path" || -z "$repo_version" ]] && json_err "$E_VALIDATION_FAILED" "Usage: registry-add <repo_path> <version>"

    registry_file="$HOME/.aether/registry.json"
    mkdir -p "$HOME/.aether"

    if [[ ! -f "$registry_file" ]]; then
      echo '{"schema_version":1,"repos":[]}' > "$registry_file"
    fi

    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Check if repo already exists in registry
    existing=$(jq --arg path "$repo_path" '.repos[] | select(.path == $path)' "$registry_file" 2>/dev/null)

    if [[ -n "$existing" ]]; then
      # Update existing entry
      updated=$(jq --arg path "$repo_path" --arg ver "$repo_version" --arg ts "$ts" '
        .repos = [.repos[] | if .path == $path then
          .version = $ver |
          .updated_at = $ts
        else . end]
      ' "$registry_file") || json_err "$E_JSON_INVALID" "Failed to update registry"
    else
      # Add new entry
      updated=$(jq --arg path "$repo_path" --arg ver "$repo_version" --arg ts "$ts" '
        .repos += [{
          "path": $path,
          "version": $ver,
          "registered_at": $ts,
          "updated_at": $ts
        }]
      ' "$registry_file") || json_err "$E_JSON_INVALID" "Failed to update registry"
    fi

    echo "$updated" > "$registry_file"
    json_ok "{\"registered\":true,\"path\":\"$repo_path\",\"version\":\"$repo_version\"}"
    ;;

  bootstrap-system)
    # Copy system files from ~/.aether/system/ into local .aether/
    # Uses explicit allowlist — never touches colony data
    hub_system="$HOME/.aether/system"
    local_aether="$AETHER_ROOT/.aether"

    [[ ! -d "$hub_system" ]] && json_err "$E_HUB_NOT_FOUND" "Hub system directory not found: $hub_system"

    # Allowlist of system files to copy (relative to system/)
    allowlist=(
      "aether-utils.sh"
      "coding-standards.md"
      "debugging.md"
      "DISCIPLINES.md"
      "learning.md"
      "planning.md"
      "QUEEN_ANT_ARCHITECTURE.md"
      "tdd.md"
      "verification-loop.md"
      "verification.md"
      "workers.md"
      "docs/constraints.md"
      "docs/pathogen-schema-example.json"
      "docs/pathogen-schema.md"
      "docs/pheromones.md"
      "docs/progressive-disclosure.md"
      "utils/atomic-write.sh"
      "utils/colorize-log.sh"
      "utils/file-lock.sh"
      "utils/watch-spawn-tree.sh"
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
    source "$SCRIPT_DIR/utils/state-loader.sh" 2>/dev/null || {
      json_err "$E_FILE_NOT_FOUND" "state-loader.sh not found"
      exit 1
    }
    load_colony_state
    if [[ $? -eq 0 ]]; then
      # Output success with handoff info if detected
      if [[ "$HANDOFF_DETECTED" == "true" ]]; then
        json_ok "{\"loaded\":true,\"handoff_detected\":true,\"handoff_summary\":\"$(get_handoff_summary)\"}"
      else
        json_ok '{"loaded":true}'
      fi
    fi
    # Note: load_colony_state handles its own error output
    ;;

  unload-state)
    source "$SCRIPT_DIR/utils/state-loader.sh" 2>/dev/null || {
      json_err "$E_FILE_NOT_FOUND" "state-loader.sh not found"
      exit 1
    }
    unload_colony_state
    json_ok '{"unloaded":true}'
    ;;

  spawn-tree-load)
    source "$SCRIPT_DIR/utils/spawn-tree.sh" 2>/dev/null || {
      json_err "$E_FILE_NOT_FOUND" "spawn-tree.sh not found"
      exit 1
    }
    tree_json=$(reconstruct_tree_json)
    json_ok "$tree_json"
    ;;

  spawn-tree-active)
    source "$SCRIPT_DIR/utils/spawn-tree.sh" 2>/dev/null || {
      json_err "$E_FILE_NOT_FOUND" "spawn-tree.sh not found"
      exit 1
    }
    active=$(get_active_spawns)
    json_ok "$active"
    ;;

  spawn-tree-depth)
    ant_name="${1:-}"
    [[ -z "$ant_name" ]] && json_err "$E_VALIDATION_FAILED" "Usage: spawn-tree-depth <ant_name>"
    source "$SCRIPT_DIR/utils/spawn-tree.sh" 2>/dev/null || {
      json_err "$E_FILE_NOT_FOUND" "spawn-tree.sh not found"
      exit 1
    }
    depth=$(get_spawn_depth "$ant_name")
    json_ok "$depth"
    ;;

  # --- Model Profile Commands ---
  model-profile)
    action="${1:-get}"
    case "$action" in
      get)
        caste="${2:-}"
        [[ -z "$caste" ]] && json_err "$E_VALIDATION_FAILED" "Usage: model-profile get <caste>"

        profile_file="$AETHER_ROOT/.aether/model-profiles.yaml"
        if [[ ! -f "$profile_file" ]]; then
          json_ok '{"model":"kimi-k2.5","source":"default","caste":"'$caste'"}'
          exit 0
        fi

        # Extract model for caste using awk (bash-compatible YAML parsing)
        model=$(awk '/^worker_models:/{found=1; next} found && /^[^ ]/{exit} found && /^  '$caste':/{print $2; exit}' "$profile_file" 2>/dev/null)

        [[ -z "$model" ]] && model="kimi-k2.5"
        json_ok '{"model":"'$model'","source":"profile","caste":"'$caste'"}'
        ;;

      list)
        profile_file="$AETHER_ROOT/.aether/model-profiles.yaml"
        if [[ ! -f "$profile_file" ]]; then
          json_ok '{"models":{},"source":"default"}'
          exit 0
        fi

        # Extract all caste:model pairs as JSON
        # Lines look like: "  prime: glm-5           # Complex coordination..."
        models=$(awk '/^worker_models:/{found=1; next} found && /^[^ ]/{exit} found && /^  [a-z_]+:/{gsub(/:/,""); printf "\"%s\":\"%s\",", $1, $2}' "$profile_file" 2>/dev/null)
        # Remove trailing comma
        models="${models%,}"

        json_ok '{"models":{'$models'},"source":"profile"}'
        ;;

      verify)
        profile_file="$AETHER_ROOT/.aether/model-profiles.yaml"
        [[ ! -f "$profile_file" ]] && json_err "$E_FILE_NOT_FOUND" "Profile not found" '{"file":"model-profiles.yaml"}'

        # Check proxy health
        proxy_health=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:4000/health 2>/dev/null || echo "000")
        proxy_status=$([[ "$proxy_health" == "200" ]] && echo "healthy" || echo "unhealthy")

        # Count castes
        caste_count=$(awk '/^worker_models:/{found=1; next} found && /^[^ ]/{exit} found && /^  [a-z_]+:/{count++} END{print count+0}' "$profile_file" 2>/dev/null)

        json_ok '{"profile_exists":true,"caste_count":'$caste_count',"proxy_status":"'$proxy_status'","proxy_endpoint":"http://localhost:4000"}'
        ;;

      select)
        # Usage: model-profile select <caste> <task_description> [cli_override]
        # Returns: JSON with model and source
        caste="$2"
        task_description="$3"
        cli_override="${4:-}"

        [[ -z "$caste" ]] && json_err "$E_VALIDATION_FAILED" "Usage: model-profile select <caste> <task_description> [cli_override]"

        # Create a temporary Node.js script to call the library
        node_script=$(cat << 'NODESCRIPT'
const { loadModelProfiles, selectModelForTask } = require('./bin/lib/model-profiles');
const caste = process.argv[2];
const taskDescription = process.argv[3];
const cliOverride = process.argv[4] || null;

try {
  const profiles = loadModelProfiles('.');
  const result = selectModelForTask(profiles, caste, taskDescription, cliOverride);
  console.log(JSON.stringify({ ok: true, result }));
} catch (error) {
  console.log(JSON.stringify({ ok: false, error: error.message }));
  process.exit(1);
}
NODESCRIPT
)

        result=$(echo "$node_script" | node - "$caste" "$task_description" "$cli_override")
        echo "$result"
        ;;

      validate)
        # Usage: model-profile validate <model_name>
        # Returns: JSON with valid boolean
        model_name="$2"

        [[ -z "$model_name" ]] && json_err "$E_VALIDATION_FAILED" "Usage: model-profile validate <model_name>"

        node_script=$(cat << 'NODESCRIPT'
const { loadModelProfiles, validateModel } = require('./bin/lib/model-profiles');
const modelName = process.argv[2];

try {
  const profiles = loadModelProfiles('.');
  const validation = validateModel(profiles, modelName);
  console.log(JSON.stringify({ ok: true, result: validation }));
} catch (error) {
  console.log(JSON.stringify({ ok: false, error: error.message }));
}
NODESCRIPT
)

        result=$(echo "$node_script" | node - "$model_name")
        echo "$result"
        ;;

      *)
        echo "Usage: model-profile <command> [args]"
        echo ""
        echo "Commands:"
        echo "  get <caste>                    Get model for caste"
        echo "  set <caste> <model>            Set user override"
        echo "  reset <caste>                  Reset user override"
        echo "  list                           List all assignments"
        echo "  select <caste> <task> [model]  Select model with task routing"
        echo "  validate <model>               Validate model name"
        json_err "$E_VALIDATION_FAILED" "Usage: model-profile get <caste>|list|verify|select|validate"
        ;;
    esac
    ;;

  model-get)
    # Shortcut: model-get <caste>
    caste="${1:-}"
    [[ -z "$caste" ]] && json_err "$E_VALIDATION_FAILED" "Usage: model-get <caste>"

    # Delegate to model-profile get
    exec bash "$0" model-profile get "$caste"
    ;;

  model-list)
    # Shortcut: list all models
    exec bash "$0" model-profile list
    ;;

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
    # Detect colony milestone from state
    # Usage: milestone-detect
    # Returns: {ok: true, milestone: "...", version: "...", phases_completed: N, total_phases: N, progress_percent: N}

    [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found" '{"file":"COLONY_STATE.json"}'

    # Extract and compute milestone data using jq
    result=$(jq '
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
    ' "$DATA_DIR/COLONY_STATE.json")

    echo "$result"
    ;;

  # ============================================
  # SWARM ACTIVITY TRACKING (colony visualization)
  # ============================================

  swarm-activity-log)
    # Log an activity entry for swarm visualization
    # Usage: swarm-activity-log <ant_name> <action> <details>
    ant_name="${1:-}"
    action="${2:-}"
    details="${3:-}"
    [[ -z "$ant_name" || -z "$action" || -z "$details" ]] && json_err "$E_VALIDATION_FAILED" "Usage: swarm-activity-log <ant_name> <action> <details>"

    mkdir -p "$DATA_DIR"
    log_file="$DATA_DIR/swarm-activity.log"
    ts=$(date -u +"%H:%M:%S")
    echo "[$ts] $ant_name: $action $details" >> "$log_file"
    json_ok '"logged"'
    ;;

  swarm-display-init)
    # Initialize swarm display state file
    # Usage: swarm-display-init <swarm_id>
    swarm_id="${1:-swarm-$(date +%s)}"
    mkdir -p "$DATA_DIR"

    display_file="$DATA_DIR/swarm-display.json"
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    atomic_write "$display_file" "{
  \"swarm_id\": \"$swarm_id\",
  \"timestamp\": \"$ts\",
  \"active_ants\": [],
  \"summary\": { \"total_active\": 0, \"by_caste\": {}, \"by_zone\": {} },
  \"chambers\": {
    \"fungus_garden\": {\"activity\": 0, \"icon\": \"🍄\"},
    \"nursery\": {\"activity\": 0, \"icon\": \"🥚\"},
    \"refuse_pile\": {\"activity\": 0, \"icon\": \"🗑️\"},
    \"throne_room\": {\"activity\": 0, \"icon\": \"👑\"},
    \"foraging_trail\": {\"activity\": 0, \"icon\": \"🌿\"}
  }
}"
    json_ok "{\"swarm_id\":\"$swarm_id\",\"initialized\":true}"
    ;;

  swarm-display-update)
    # Update ant activity in swarm display
    # Usage: swarm-display-update <ant_name> <caste> <ant_status> <task> [parent] [tools_json] [tokens] [chamber] [progress]
    ant_name="${1:-}"
    caste="${2:-}"
    ant_status="${3:-}"
    task="${4:-}"
    parent="${5:-}"
    tools_json="${6:-}"
    [[ -z "$tools_json" ]] && tools_json="{}"
    tokens="${7:-0}"
    chamber="${8:-}"
    progress="${9:-0}"

    [[ -z "$ant_name" || -z "$caste" || -z "$ant_status" ]] && json_err "$E_VALIDATION_FAILED" "Usage: swarm-display-update <ant_name> <caste> <ant_status> <task> [parent] [tools_json] [tokens] [chamber] [progress]"

    display_file="$DATA_DIR/swarm-display.json"

    # Initialize if doesn't exist
    if [[ ! -f "$display_file" ]]; then
      bash "$0" swarm-display-init "default-swarm" >/dev/null 2>&1
    fi

    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Read current display and update using jq
    updated=$(jq --arg ant "$ant_name" --arg caste "$caste" --arg ant_status "$ant_status" \
      --arg task "$task" --arg parent "$parent" --argjson tools "$tools_json" \
      --argjson tokens "$tokens" --arg ts "$ts" --arg chamber "$chamber" --argjson progress "$progress" '
      # Find existing ant or create new entry
      (.active_ants | map(select(.name == $ant)) | length) as $exists |
      # Get old chamber if ant exists
      (if $exists > 0 then
        (.active_ants[] | select(.name == $ant) | .chamber // "")
      else
        ""
      end) as $old_chamber |
      # Determine new chamber
      (if $chamber != "" then $chamber else $old_chamber end) as $new_chamber |
      if $exists > 0 then
        # Update existing ant
        .active_ants = [.active_ants[] | if .name == $ant then
          . + {
            caste: $caste,
            status: $ant_status,
            task: $task,
            parent: (if $parent != "" then $parent else .parent end),
            tools: (if $tools != {} then $tools else .tools end),
            tokens: (.tokens + $tokens),
            chamber: (if $chamber != "" then $chamber else (.chamber // null) end),
            progress: (if $progress > 0 then $progress else (.progress // 0) end),
            updated_at: $ts
          }
        else . end]
      else
        # Add new ant
        .active_ants += [{
          name: $ant,
          caste: $caste,
          status: $ant_status,
          task: $task,
          parent: (if $parent != "" then $parent else null end),
          tools: (if $tools != {} then $tools else {read:0,grep:0,edit:0,bash:0} end),
          tokens: $tokens,
          chamber: (if $chamber != "" then $chamber else null end),
          progress: $progress,
          started_at: $ts,
          updated_at: $ts
        }]
      end |
      # Recalculate summary
      .summary.total_active = (.active_ants | length) |
      .summary.by_caste = (.active_ants | group_by(.caste) | map({key: .[0].caste, value: length}) | from_entries) |
      .summary.by_zone = (.active_ants | group_by(.status) | map({key: .[0].status, value: length}) | from_entries) |
      # Update chamber activity counts
      # Decrement old chamber if changed
      (if $old_chamber != "" and $old_chamber != $new_chamber and has("chambers") and (.chambers | has($old_chamber)) then
        .chambers[$old_chamber].activity = [(.chambers[$old_chamber].activity // 1) - 1, 0] | max
      else
        .
      end) |
      # Increment new chamber
      (if $new_chamber != "" and has("chambers") and (.chambers | has($new_chamber)) then
        .chambers[$new_chamber].activity = (.chambers[$new_chamber].activity // 0) + 1
      else
        .
      end)
    ' "$display_file") || json_err "$E_JSON_INVALID" "Failed to update swarm display"

    atomic_write "$display_file" "$updated"

    # Get emoji for response
    emoji=$(get_caste_emoji "$caste")
    json_ok "{\"updated\":true,\"ant\":\"$ant_name\",\"caste\":\"$caste\",\"emoji\":\"$emoji\",\"chamber\":\"$chamber\",\"progress\":$progress}"
    ;;

  swarm-display-get)
    # Get current swarm display state
    # Usage: swarm-display-get
    display_file="$DATA_DIR/swarm-display.json"

    if [[ ! -f "$display_file" ]]; then
      json_ok '{"swarm_id":null,"active_ants":[],"summary":{"total_active":0,"by_caste":{},"by_zone":{}},"chambers":{}}'
    else
      json_ok "$(cat "$display_file")"
    fi
    ;;

  swarm-display-render)
    # Render the swarm display to terminal
    # Usage: swarm-display-render [swarm_id]
    swarm_id="${1:-default-swarm}"

    display_script="$SCRIPT_DIR/utils/swarm-display.sh"

    if [[ -f "$display_script" ]]; then
      # Execute the display script
      bash "$display_script" "$swarm_id" 2>/dev/null || true
      json_ok '{"rendered":true}'
    else
      json_err "$E_FILE_NOT_FOUND" "Display script not found: $display_script"
    fi
    ;;

  swarm-display-inline)
    # Inline swarm display for Claude Code (no loop, no clear)
    # Usage: swarm-display-inline [swarm_id]
    swarm_id="${1:-default-swarm}"
    display_file="$DATA_DIR/swarm-display.json"

    # ANSI colors
    BLUE='\033[34m'
    GREEN='\033[32m'
    YELLOW='\033[33m'
    RED='\033[31m'
    MAGENTA='\033[35m'
    BOLD='\033[1m'
    DIM='\033[2m'
    RESET='\033[0m'

    # Caste colors
    get_caste_color() {
      case "$1" in
        builder) echo "$BLUE" ;;
        watcher) echo "$GREEN" ;;
        scout) echo "$YELLOW" ;;
        chaos) echo "$RED" ;;
        prime) echo "$MAGENTA" ;;
        oracle) echo "$MAGENTA" ;;
        route_setter) echo "$MAGENTA" ;;
        *) echo "$RESET" ;;
      esac
    }

    # Caste emojis with ant
    get_caste_emoji() {
      case "$1" in
        builder) echo "🔨🐜" ;;
        watcher) echo "👁️🐜" ;;
        scout) echo "🔍🐜" ;;
        chaos) echo "🎲🐜" ;;
        prime) echo "👑🐜" ;;
        oracle) echo "🔮🐜" ;;
        route_setter) echo "🧭🐜" ;;
        archaeologist) echo "🏺🐜" ;;
        chronicler) echo "📝🐜" ;;
        gatekeeper) echo "📦🐜" ;;
        guardian) echo "🛡️🐜" ;;
        includer) echo "♿🐜" ;;
        keeper) echo "📚🐜" ;;
        measurer) echo "⚡🐜" ;;
        probe) echo "🧪🐜" ;;
        sage) echo "📜🐜" ;;
        tracker) echo "🐛🐜" ;;
        weaver) echo "🔄🐜" ;;
        colonizer) echo "🌱🐜" ;;
        dreamer) echo "💭🐜" ;;
        *) echo "🐜" ;;
      esac
    }

    # Status phrases
    get_status_phrase() {
      case "$1" in
        builder) echo "excavating..." ;;
        watcher) echo "observing..." ;;
        scout) echo "exploring..." ;;
        chaos) echo "testing..." ;;
        prime) echo "coordinating..." ;;
        oracle) echo "researching..." ;;
        route_setter) echo "planning..." ;;
        *) echo "working..." ;;
      esac
    }

    # Excavation phrase based on progress
    get_excavation_phrase() {
      local progress="${1:-0}"
      if [[ "$progress" -lt 25 ]]; then
        echo "🚧 Starting excavation..."
      elif [[ "$progress" -lt 50 ]]; then
        echo "⛏️  Digging deeper..."
      elif [[ "$progress" -lt 75 ]]; then
        echo "🪨 Moving earth..."
      elif [[ "$progress" -lt 100 ]]; then
        echo "🏗️  Almost there..."
      else
        echo "✅ Excavation complete!"
      fi
    }

    # Format tools: "📖5 🔍3 ✏️2 ⚡1"
    format_tools() {
      local read="${1:-0}"
      local grep="${2:-0}"
      local edit="${3:-0}"
      local bash="${4:-0}"
      local result=""
      [[ "$read" -gt 0 ]] && result="${result}📖${read} "
      [[ "$grep" -gt 0 ]] && result="${result}🔍${grep} "
      [[ "$edit" -gt 0 ]] && result="${result}✏️${edit} "
      [[ "$bash" -gt 0 ]] && result="${result}⚡${bash}"
      echo "$result"
    }

    # Render progress bar (green when working)
    render_progress_bar() {
      local percent="${1:-0}"
      local width="${2:-20}"
      [[ "$percent" -lt 0 ]] && percent=0
      [[ "$percent" -gt 100 ]] && percent=100
      local filled=$((percent * width / 100))
      local empty=$((width - filled))
      local bar=""
      for ((i=0; i<filled; i++)); do bar+="█"; done
      for ((i=0; i<empty; i++)); do bar+="░"; done
      echo -e "${GREEN}[$bar]${RESET} ${percent}%"
    }

    # Format duration
    format_duration() {
      local seconds="${1:-0}"
      if [[ "$seconds" -lt 60 ]]; then
        echo "${seconds}s"
      else
        local mins=$((seconds / 60))
        local secs=$((seconds % 60))
        echo "${mins}m${secs}s"
      fi
    }

    # Check for display file
    if [[ ! -f "$display_file" ]]; then
      echo -e "${DIM}🐜 No active swarm data${RESET}"
      json_ok '{"displayed":false,"reason":"no_data"}'
      exit 0
    fi

    # Check for jq
    if ! command -v jq >/dev/null 2>&1; then
      echo -e "${DIM}🐜 Swarm active (jq not available for details)${RESET}"
      json_ok '{"displayed":true,"warning":"jq_missing"}'
      exit 0
    fi

    # Read swarm data
    total_active=$(jq -r '.summary.total_active // 0' "$display_file" 2>/dev/null || echo "0")

    if [[ "$total_active" -eq 0 ]]; then
      echo -e "${DIM}🐜 Colony idle${RESET}"
      json_ok '{"displayed":true,"ants":0}'
      exit 0
    fi

    # Render header with ant logo
    echo ""
    cat << 'ANTLOGO'


                                      ▁▐▖      ▁
                            ▗▇▇███▆▇▃▅████▆▇▆▅▟██▛▇
                             ▝▜▅▛██████████████▜▅██
                          ▁▂▀▇▆██▙▜██████████▛▟███▛▁▃▁
                         ▕▂▁▉▅████▙▞██████▜█▚▟████▅▊ ▐
                        ▗▁▐█▀▜████▛▃▝▁████▍▘▟▜████▛▀█▂ ▖
                    ▁▎▝█▁▝▍▆▜████▊▐▀▏▀▍▂▂▝▀▕▀▌█████▀▅▐▚ █▏▁▁
                      ▂▚▃▇▙█▟████▛▏ ▝▜▐▛▀▍▛▘ ▕█████▆▊▐▂▃▞▂▔
                       ▚▔█▛██████▙▟▍▜▍▜▃▃▖▟▛▐██████▛▛▜▔▔▞
                        ▋▖▍▊▖██████▇▃▁▝██▘▝▃████▜█▜ ▋▐▐▗
                        ▍▌▇█▅▂▜██████████████████▉▃▄▋▖  ▝
                      ▁▎▍▁▜▟███▀▀▜████████████▛▀▀███▆▂  ▁▁
                     ██ ▆▇▌▁▕▚▅▆███▛████████▜███▆▄▞▁▁▐▅▎ █▉
                     ▆█████▛▃▟█▀████████████████▛█▙▙▜▉▟▛▜█▌▗
                     ▅▆▋ ▁▁▁▔▕▁▁▁▇█████▛▀▀▀▁▜▇▇▁▁▁▁▁▁▁▁ ▐▊▗
                   ▗▆▃▃▃▔███▖▔██▀▀▝▀██▀▍█▛▁▐█▏█▛▀▀▏█▛▀▜█▆▃▃▆▖
                   ▝▗▖  ▟█▟█▙ █▛▀▘  █▊ ▕█▛▀▜█▏█▛▀▘ █▋▆█▛  ▗▖
                   ▘ ▘ ▟▛  ▝▀▘▀▀▀▀▘ ▀▀▂▂█▙▂▐▀▏▀▀▀▀▘▀▘ ▝▀▅▂▝ ▕▏
                    ▕▕  ▃▗▄▔▗▄▄▗▗▗▔▄▄▄▄▗▄▄▗▔▃▃▃▗▄▂▄▃▗▄▂▖▖ ▏▁
                    ▝▘▏ ▔▔   ▁▔▁▔▔▁▔▔▔▔▔▔▔▁▁ ▔▔   ▔▔▔▔
                             ▀ ▀▝▘▀▀▔▘▘▀▝▕▀▀▝▝▀▔▀ ▀▔▘
                            ▘ ▗▅▁▝▚▃▀▆▟██▙▆▝▃ ▘ ▁▗▌
                               ▔▀▔▝ ▔▀▟▜▛▛▀▔    ▀


ANTLOGO
    echo -e "${BOLD}AETHER COLONY :: Colony Activity${RESET}"
    echo -e "${DIM}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
    echo ""

    # Render each active ant (limit to 5)
    jq -r '.active_ants[0:5][] | "\(.name)|\(.caste)|\(.status // "")|\(.task // "")|\(.tools.read // 0)|\(.tools.grep // 0)|\(.tools.edit // 0)|\(.tools.bash // 0)|\(.tokens // 0)|\(.started_at // "")|\(.parent // "Queen")|\(.progress // 0)"' "$display_file" 2>/dev/null | while IFS='|' read -r ant_name ant_caste ant_status ant_task read_ct grep_ct edit_ct bash_ct tokens started_at parent progress; do
      color=$(get_caste_color "$ant_caste")
      emoji=$(get_caste_emoji "$ant_caste")
      phrase=$(get_status_phrase "$ant_caste")

      # Format tools
      tools_str=$(format_tools "$read_ct" "$grep_ct" "$edit_ct" "$bash_ct")

      # Truncate task if too long
      display_task="$ant_task"
      [[ ${#display_task} -gt 35 ]] && display_task="${display_task:0:32}..."

      # Calculate elapsed time
      elapsed_str=""
      started_ts="${started_at:-}"
      if [[ -n "$started_ts" ]] && [[ "$started_ts" != "null" ]]; then
        started_ts=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "$started_ts" +%s 2>/dev/null)
        if [[ -z "$started_ts" ]] || [[ "$started_ts" == "null" ]]; then
          started_ts=$(date -d "$started_ts" +%s 2>/dev/null) || started_ts=0
        fi
        now_ts=$(date +%s)
        elapsed=0
        if [[ -n "$started_ts" ]] && [[ "$started_ts" -gt 0 ]] 2>/dev/null; then
          elapsed=$((now_ts - started_ts))
        fi
        if [[ ${elapsed:-0} -gt 0 ]]; then
          elapsed_str="($(format_duration $elapsed))"
        fi
      fi

      # Token indicator
      token_str=""
      if [[ -n "$tokens" ]] && [[ "$tokens" -gt 0 ]]; then
        token_str="🍯${tokens}"
      fi

      # Output ant line: "🐜 Builder: excavating... Implement auth 📖5 🔍3 (2m3s) 🍯1250"
      echo -e "${color}${emoji} ${BOLD}${ant_name}${RESET}${color}: ${phrase}${RESET} ${display_task}"
      echo -e "   ${tools_str} ${DIM}${elapsed_str}${RESET} ${token_str}"

      # Show progress bar if progress > 0
      if [[ -n "$progress" ]] && [[ "$progress" -gt 0 ]]; then
        progress_bar=$(render_progress_bar "$progress" 15)
        excavation_phrase=$(get_excavation_phrase "$progress")
        echo -e "   ${DIM}${progress_bar}${RESET}"
        echo -e "   ${DIM}${excavation_phrase}${RESET}"
      fi

      echo ""
    done

    # Chamber activity map
    echo -e "${DIM}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
    echo ""
    echo -e "${BOLD}Chamber Activity:${RESET}"

    # Show active chambers with fire intensity
    has_chamber_activity=0
    jq -r '.chambers | to_entries[] | "\(.key)|\(.value.activity)|\(.value.icon)"' "$display_file" 2>/dev/null | \
    while IFS='|' read -r chamber activity icon; do
      if [[ -n "$activity" ]] && [[ "$activity" -gt 0 ]]; then
        has_chamber_activity=1
        if [[ "$activity" -ge 5 ]]; then
          fires="🔥🔥🔥"
        elif [[ "$activity" -ge 3 ]]; then
          fires="🔥🔥"
        else
          fires="🔥"
        fi
        chamber_name="${chamber//_/ }"
        echo -e "  ${icon} ${chamber_name} ${fires} (${activity} ants)"
      fi
    done

    if [[ "$has_chamber_activity" -eq 0 ]]; then
      echo -e "${DIM}  (no chamber activity)${RESET}"
    fi

    # Summary
    echo ""
    echo -e "${DIM}${total_active} forager$([[ "$total_active" -eq 1 ]] || echo "s") excavating...${RESET}"

    json_ok "{\"displayed\":true,\"ants\":$total_active}"
    ;;

  swarm-display-text)
    # Plain-text swarm display for Claude conversation (no ANSI codes)
    # Usage: swarm-display-text [swarm_id]
    swarm_id="${1:-default-swarm}"
    display_file="$DATA_DIR/swarm-display.json"

    # Check for display file
    if [[ ! -f "$display_file" ]]; then
      echo "🐜 Colony idle"
      json_ok '{"displayed":false,"reason":"no_data"}'
      exit 0
    fi

    # Check for jq
    if ! command -v jq >/dev/null 2>&1; then
      echo "🐜 Swarm active (details unavailable)"
      json_ok '{"displayed":true,"warning":"jq_missing"}'
      exit 0
    fi

    # Read swarm data — handle both flat total_active and nested .summary.total_active
    total_active=$(jq -r '(.total_active // .summary.total_active // 0)' "$display_file" 2>/dev/null || echo "0")

    if [[ "$total_active" -eq 0 ]]; then
      echo "🐜 Colony idle"
      json_ok '{"displayed":true,"ants":0}'
      exit 0
    fi

    # Compact header
    echo "🐜 COLONY ACTIVITY"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    # Caste emoji lookup
    get_emoji() {
      case "$1" in
        builder)       echo "🔨🐜" ;;
        watcher)       echo "👁️🐜" ;;
        scout)         echo "🔍🐜" ;;
        chaos)         echo "🎲🐜" ;;
        prime)         echo "👑🐜" ;;
        oracle)        echo "🔮🐜" ;;
        route_setter)  echo "🧭🐜" ;;
        archaeologist) echo "🏺🐜" ;;
        surveyor)      echo "📊🐜" ;;
        *)             echo "🐜" ;;
      esac
    }

    # Format tool counts (only non-zero)
    format_tools_text() {
      local r="${1:-0}" g="${2:-0}" e="${3:-0}" b="${4:-0}"
      local result=""
      [[ "$r" -gt 0 ]] && result="${result}📖${r} "
      [[ "$g" -gt 0 ]] && result="${result}🔍${g} "
      [[ "$e" -gt 0 ]] && result="${result}✏️${e} "
      [[ "$b" -gt 0 ]] && result="${result}⚡${b}"
      echo "$result"
    }

    # Progress bar using block characters (no ANSI)
    render_bar_text() {
      local pct="${1:-0}" w="${2:-10}"
      [[ "$pct" -lt 0 ]] && pct=0
      [[ "$pct" -gt 100 ]] && pct=100
      local filled=$((pct * w / 100))
      local empty=$((w - filled))
      local bar=""
      for ((i=0; i<filled; i++)); do bar+="█"; done
      for ((i=0; i<empty; i++)); do bar+="░"; done
      echo "[$bar] ${pct}%"
    }

    # Render each ant (max 5)
    jq -r '.active_ants[0:5][] | "\(.name)|\(.caste)|\(.task // "")|\(.tools.read // 0)|\(.tools.grep // 0)|\(.tools.edit // 0)|\(.tools.bash // 0)|\(.progress // 0)"' "$display_file" 2>/dev/null | while IFS='|' read -r name caste task r g e b progress; do
      emoji=$(get_emoji "$caste")
      tools=$(format_tools_text "$r" "$g" "$e" "$b")
      bar=$(render_bar_text "${progress:-0}" 10)

      # Truncate task to 25 chars
      [[ ${#task} -gt 25 ]] && task="${task:0:22}..."

      echo "${emoji} ${name} ${bar} ${task}"
      [[ -n "$tools" ]] && echo "   ${tools}"
      echo ""
    done

    # Overflow indicator
    if [[ "$total_active" -gt 5 ]]; then
      echo "   +$((total_active - 5)) more ants..."
      echo ""
    fi

    # Footer
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "${total_active} ants active"

    json_ok "{\"displayed\":true,\"ants\":$total_active}"
    ;;

  swarm-timing-start)
    # Record start time for an ant
    # Usage: swarm-timing-start <ant_name>
    ant_name="${1:-}"
    [[ -z "$ant_name" ]] && json_err "$E_VALIDATION_FAILED" "Usage: swarm-timing-start <ant_name>"

    mkdir -p "$DATA_DIR"
    timing_file="$DATA_DIR/timing.log"
    ts=$(date +%s)
    ts_iso=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Remove any existing entry for this ant and append new one
    if [[ -f "$timing_file" ]]; then
      grep -v "^$ant_name|" "$timing_file" > "${timing_file}.tmp" 2>/dev/null || true
      mv "${timing_file}.tmp" "$timing_file"
    fi
    echo "$ant_name|$ts|$ts_iso" >> "$timing_file"

    json_ok "{\"ant\":\"$ant_name\",\"started_at\":\"$ts_iso\",\"timestamp\":$ts}"
    ;;

  swarm-timing-get)
    # Get elapsed time for an ant
    # Usage: swarm-timing-get <ant_name>
    ant_name="${1:-}"
    [[ -z "$ant_name" ]] && json_err "$E_VALIDATION_FAILED" "Usage: swarm-timing-get <ant_name>"

    timing_file="$DATA_DIR/timing.log"

    if [[ ! -f "$timing_file" ]] || ! grep -q "^$ant_name|" "$timing_file" 2>/dev/null; then
      json_ok "{\"ant\":\"$ant_name\",\"started_at\":null,\"elapsed_seconds\":0,\"elapsed_formatted\":\"00:00\"}"
      exit 0
    fi

    # Read start time
    start_line=$(grep "^$ant_name|" "$timing_file" | tail -1)
    start_ts=$(echo "$start_line" | cut -d'|' -f2)
    start_iso=$(echo "$start_line" | cut -d'|' -f3)

    now=$(date +%s)
    elapsed=$((now - start_ts))

    # Format as MM:SS
    mins=$((elapsed / 60))
    secs=$((elapsed % 60))
    formatted=$(printf "%02d:%02d" $mins $secs)

    json_ok "{\"ant\":\"$ant_name\",\"started_at\":\"$start_iso\",\"elapsed_seconds\":$elapsed,\"elapsed_formatted\":\"$formatted\"}"
    ;;

  swarm-timing-eta)
    # Calculate ETA based on progress percentage
    # Usage: swarm-timing-eta <ant_name> <percent_complete>
    ant_name="${1:-}"
    percent="${2:-0}"
    [[ -z "$ant_name" ]] && json_err "$E_VALIDATION_FAILED" "Usage: swarm-timing-eta <ant_name> <percent_complete>"

    # Validate percent is a number
    if ! [[ "$percent" =~ ^[0-9]+$ ]]; then
      percent=0
    fi

    # Clamp percent to 0-100
    if [[ $percent -lt 0 ]]; then
      percent=0
    elif [[ $percent -gt 100 ]]; then
      percent=100
    fi

    timing_file="$DATA_DIR/timing.log"

    if [[ ! -f "$timing_file" ]] || ! grep -q "^$ant_name|" "$timing_file" 2>/dev/null; then
      json_ok "{\"ant\":\"$ant_name\",\"percent\":$percent,\"eta_seconds\":null,\"eta_formatted\":\"--:--\"}"
      exit 0
    fi

    # Read start time
    start_ts=$(grep "^$ant_name|" "$timing_file" | tail -1 | cut -d'|' -f2)
    now=$(date +%s)
    elapsed=$((now - start_ts))

    # Calculate ETA
    if [[ $percent -le 0 ]]; then
      eta_seconds=null
      eta_formatted="--:--"
    elif [[ $percent -ge 100 ]]; then
      eta_seconds=0
      eta_formatted="00:00"
    else
      # ETA = (elapsed / percent) * (100 - percent)
      eta_seconds=$(( (elapsed * (100 - percent)) / percent ))
      mins=$((eta_seconds / 60))
      secs=$((eta_seconds % 60))
      eta_formatted=$(printf "%02d:%02d" $mins $secs)
    fi

    json_ok "{\"ant\":\"$ant_name\",\"percent\":$percent,\"eta_seconds\":$eta_seconds,\"eta_formatted\":\"$eta_formatted\"}"
    ;;

  # ============================================
  # VIEW STATE MANAGEMENT (collapsible views)
  # ============================================

  view-state-init)
    # Initialize view state file with default structure
    # Usage: view-state-init
    mkdir -p "$DATA_DIR"
    view_state_file="$DATA_DIR/view-state.json"

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
    view_state_file="$DATA_DIR/view-state.json"

    if [[ ! -f "$view_state_file" ]]; then
      # Auto-initialize if not exists
      bash "$0" view-state-init >/dev/null 2>&1
    fi

    if [[ -z "$view_name" ]]; then
      # Return entire state
      json_ok "$(cat "$view_state_file")"
    elif [[ -z "$key" ]]; then
      # Return specific view
      json_ok "$(jq ".${view_name} // {}" "$view_state_file")"
    else
      # Return specific key from view
      json_ok "$(jq ".${view_name}.${key} // null" "$view_state_file")"
    fi
    ;;

  view-state-set)
    # Set a specific key in a view
    # Usage: view-state-set <view_name> <key> <value>
    view_name="${1:-}"
    key="${2:-}"
    value="${3:-}"
    [[ -z "$view_name" || -z "$key" ]] && json_err "$E_VALIDATION_FAILED" "Usage: view-state-set <view_name> <key> <value>"

    view_state_file="$DATA_DIR/view-state.json"

    if [[ ! -f "$view_state_file" ]]; then
      bash "$0" view-state-init >/dev/null 2>&1
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
    json_ok "$(echo "$updated" | jq ".${view_name}")"
    ;;

  view-state-toggle)
    # Toggle item between expanded and collapsed
    # Usage: view-state-toggle <view_name> <item>
    view_name="${1:-}"
    item="${2:-}"
    [[ -z "$view_name" || -z "$item" ]] && json_err "$E_VALIDATION_FAILED" "Usage: view-state-toggle <view_name> <item>"

    view_state_file="$DATA_DIR/view-state.json"

    if [[ ! -f "$view_state_file" ]]; then
      bash "$0" view-state-init >/dev/null 2>&1
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
    json_ok "{\"item\":\"$item\",\"state\":\"$new_state\",\"view\":\"$view_name\"}"
    ;;

  view-state-expand)
    # Explicitly expand an item
    # Usage: view-state-expand <view_name> <item>
    view_name="${1:-}"
    item="${2:-}"
    [[ -z "$view_name" || -z "$item" ]] && json_err "$E_VALIDATION_FAILED" "Usage: view-state-expand <view_name> <item>"

    view_state_file="$DATA_DIR/view-state.json"

    if [[ ! -f "$view_state_file" ]]; then
      bash "$0" view-state-init >/dev/null 2>&1
    fi

    updated=$(jq --arg view "$view_name" --arg item "$item" '
      .[$view].collapsed -= [$item] |
      .[$view].expanded += [$item]
    ' "$view_state_file") || json_err "$E_JSON_INVALID" "Failed to update view state"

    atomic_write "$view_state_file" "$updated"
    json_ok "{\"item\":\"$item\",\"state\":\"expanded\",\"view\":\"$view_name\"}"
    ;;

  view-state-collapse)
    # Explicitly collapse an item
    # Usage: view-state-collapse <view_name> <item>
    view_name="${1:-}"
    item="${2:-}"
    [[ -z "$view_name" || -z "$item" ]] && json_err "$E_VALIDATION_FAILED" "Usage: view-state-collapse <view_name> <item>"

    view_state_file="$DATA_DIR/view-state.json"

    if [[ ! -f "$view_state_file" ]]; then
      bash "$0" view-state-init >/dev/null 2>&1
    fi

    updated=$(jq --arg view "$view_name" --arg item "$item" '
      .[$view].expanded -= [$item] |
      .[$view].collapsed += [$item]
    ' "$view_state_file") || json_err "$E_JSON_INVALID" "Failed to update view state"

    atomic_write "$view_state_file" "$updated"
    json_ok "{\"item\":\"$item\",\"state\":\"collapsed\",\"view\":\"$view_name\"}"
    ;;

  queen-init)
    # Initialize QUEEN.md from template
    # Creates .aether/QUEEN.md from template if missing
    queen_file="$AETHER_ROOT/.aether/docs/QUEEN.md"

    # Check multiple locations for template
    # Order: dev (runtime/) -> npm install (hub) -> legacy
    template_file=""
    for path in \
      "$AETHER_ROOT/runtime/templates/QUEEN.md.template" \
      "$HOME/.aether/templates/QUEEN.md.template" \
      "$AETHER_ROOT/.aether/templates/QUEEN.md.template"; do
      if [[ -f "$path" ]]; then
        template_file="$path"
        break
      fi
    done

    # Ensure docs directory exists
    mkdir -p "$AETHER_ROOT/.aether/docs"

    # Check if QUEEN.md already exists and has content
    if [[ -f "$queen_file" ]] && [[ -s "$queen_file" ]]; then
      json_ok '{"created":false,"path":".aether/docs/QUEEN.md","reason":"already_exists"}'
      exit 0
    fi

    # Check if template was found
    if [[ -z "$template_file" ]]; then
      json_err "$E_FILE_NOT_FOUND" "Template not found" '{"templates_checked":["runtime/templates/QUEEN.md.template","~/.aether/templates/QUEEN.md.template",".aether/templates/QUEEN.md.template"]}'
      exit 1
    fi

    # Create QUEEN.md from template with timestamp substitution
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    sed -e "s/{TIMESTAMP}/$timestamp/g" "$template_file" > "$queen_file"

    if [[ -f "$queen_file" ]]; then
      json_ok "{\"created\":true,\"path\":\".aether/docs/QUEEN.md\",\"source\":\"$template_file\"}"
    else
      json_err "$E_FILE_NOT_FOUND" "Failed to create QUEEN.md" '{"path":".aether/docs/QUEEN.md"}'
      exit 1
    fi
    ;;

  queen-read)
    # Read QUEEN.md and return wisdom as JSON for worker priming
    # Extracts METADATA block and sections for colony guidance
    queen_file="$AETHER_ROOT/.aether/docs/QUEEN.md"

    # Check if QUEEN.md exists
    if [[ ! -f "$queen_file" ]]; then
      json_err "$E_FILE_NOT_FOUND" "QUEEN.md not found" '{"path":".aether/docs/QUEEN.md"}'
      exit 1
    fi

    # Extract METADATA JSON block (between <!-- METADATA and -->)
    metadata=$(sed -n '/<!-- METADATA/,/-->/p' "$queen_file" | sed '1d;$d' | tr -d '\n' | sed 's/^[[:space:]]*//')

    # If no metadata found, return empty structure
    if [[ -z "$metadata" ]]; then
      metadata='{"version":"unknown","last_evolved":null,"colonies_contributed":[],"promotion_thresholds":{},"stats":{}}'
    fi

    # Extract sections content for worker priming
    # Use awk to parse markdown sections - remove header line and trailing section header
    philosophies=$(awk '/^## 📜 Philosophies$/,/^## /' "$queen_file" | tail -n +2 | sed '$d' | sed '/^$/d' | jq -Rs '.')
    patterns=$(awk '/^## 🧭 Patterns$/,/^## /' "$queen_file" | tail -n +2 | sed '$d' | sed '/^$/d' | jq -Rs '.')
    redirects=$(awk '/^## ⚠️ Redirects$/,/^## /' "$queen_file" | tail -n +2 | sed '$d' | sed '/^$/d' | jq -Rs '.')
    stack_wisdom=$(awk '/^## 🔧 Stack Wisdom$/,/^## /' "$queen_file" | tail -n +2 | sed '$d' | sed '/^$/d' | jq -Rs '.')
    decrees=$(awk '/^## 🏛️ Decrees$/,/^## /' "$queen_file" | tail -n +2 | sed '$d' | sed '/^$/d' | jq -Rs '.')

    # Build JSON output
    result=$(jq -n \
      --argjson meta "$metadata" \
      --arg philosophies "$philosophies" \
      --arg patterns "$patterns" \
      --arg redirects "$redirects" \
      --arg stack_wisdom "$stack_wisdom" \
      --arg decrees "$decrees" \
      '{
        metadata: $meta,
        wisdom: {
          philosophies: $philosophies,
          patterns: $patterns,
          redirects: $redirects,
          stack_wisdom: $stack_wisdom,
          decrees: $decrees
        },
        priming: {
          has_philosophies: ($philosophies | length) > 0 and $philosophies != "*No philosophies recorded yet.*\n",
          has_patterns: ($patterns | length) > 0 and $patterns != "*No patterns recorded yet.*\n",
          has_redirects: ($redirects | length) > 0 and $redirects != "*No redirects recorded yet.*\n",
          has_stack_wisdom: ($stack_wisdom | length) > 0 and $stack_wisdom != "*No stack wisdom recorded yet.*\n",
          has_decrees: ($decrees | length) > 0 and $decrees != "*No decrees recorded yet.*\n"
        }
      }')

    json_ok "$result"
    ;;

  queen-promote)
    # Promote a learning to QUEEN.md wisdom
    # Usage: queen-promote <type> <content> <colony_name>
    # Types: philosophy, pattern, redirect, stack, decree
    wisdom_type="${1:-}"
    content="${2:-}"
    colony_name="${3:-}"

    # Validate required arguments
    [[ -z "$wisdom_type" ]] && json_err "$E_VALIDATION_FAILED" "Usage: queen-promote <type> <content> <colony_name>" '{"missing":"type"}'
    [[ -z "$content" ]] && json_err "$E_VALIDATION_FAILED" "Usage: queen-promote <type> <content> <colony_name>" '{"missing":"content"}'
    [[ -z "$colony_name" ]] && json_err "$E_VALIDATION_FAILED" "Usage: queen-promote <type> <content> <colony_name>" '{"missing":"colony_name"}'

    # Validate type
    valid_types=("philosophy" "pattern" "redirect" "stack" "decree")
    type_valid=false
    for vt in "${valid_types[@]}"; do
      [[ "$wisdom_type" == "$vt" ]] && type_valid=true && break
    done
    [[ "$type_valid" == "false" ]] && json_err "$E_VALIDATION_FAILED" "Invalid type: $wisdom_type" '{"valid_types":["philosophy","pattern","redirect","stack","decree"]}'

    queen_file="$AETHER_ROOT/.aether/docs/QUEEN.md"

    # Check if QUEEN.md exists
    if [[ ! -f "$queen_file" ]]; then
      json_err "$E_FILE_NOT_FOUND" "QUEEN.md not found" '{"path":".aether/docs/QUEEN.md"}'
      exit 1
    fi

    # Extract METADATA to get promotion thresholds
    metadata=$(sed -n '/<!-- METADATA/,/-->/p' "$queen_file" | sed '1d;$d' | tr -d '\n' | sed 's/^[[:space:]]*//')

    # Get threshold for this type (default: philosophy=5, pattern=3, redirect=2, stack=1, decree=0)
    threshold=$(echo "$metadata" | jq -r ".promotion_thresholds.${wisdom_type} // null")
    if [[ "$threshold" == "null" ]]; then
      case "$wisdom_type" in
        philosophy) threshold=5 ;;
        pattern) threshold=3 ;;
        redirect) threshold=2 ;;
        stack) threshold=1 ;;
        decree) threshold=0 ;;
        *) threshold=1 ;;
      esac
    fi

    # For decrees, always promote immediately (threshold 0)
    # For other types, we assume validation count is passed or threshold is met
    # In a real implementation, this would check a validation counter
    # For now, we append if threshold allows (decrees always, others need external validation)

    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Map type to section header and emoji
    case "$wisdom_type" in
      philosophy) section_header="## 📜 Philosophies" ;;
      pattern) section_header="## 🧭 Patterns" ;;
      redirect) section_header="## ⚠️ Redirects" ;;
      stack) section_header="## 🔧 Stack Wisdom" ;;
      decree) section_header="## 🏛️ Decrees" ;;
    esac

    # Build the new entry
    entry="- **${colony_name}** (${ts}): ${content}"

    # Create temp file for atomic write
    tmp_file="${queen_file}.tmp.$$"

    # Find line numbers for section boundaries
    section_line=$(grep -n "^${section_header}$" "$queen_file" | head -1 | cut -d: -f1)
    next_section_line=$(tail -n +$((section_line + 1)) "$queen_file" | grep -n "^## " | head -1 | cut -d: -f1)
    if [[ -n "$next_section_line" ]]; then
      section_end=$((section_line + next_section_line - 1))
    else
      section_end=$(wc -l < "$queen_file")
    fi

    # Check if section has placeholder (grep returns 1 when no matches, handle with || true)
    has_placeholder=$(sed -n "${section_line},${section_end}p" "$queen_file" | grep -c "No.*recorded yet" || true)
    has_placeholder=${has_placeholder:-0}

    if [[ "$has_placeholder" -gt 0 ]]; then
      # Replace placeholder with entry - only within the target section
      # Find the specific line number of the placeholder within the section
      placeholder_line=$(sed -n "${section_line},${section_end}p" "$queen_file" | grep -n "^\\*No .* recorded yet" | head -1 | cut -d: -f1)
      if [[ -n "$placeholder_line" ]]; then
        actual_line=$((section_line + placeholder_line - 1))
        sed "${actual_line}c\\
${entry}" "$queen_file" > "$tmp_file"
      else
        # Fallback: insert after section header
        sed "${section_line}a\\
${entry}" "$queen_file" > "$tmp_file"
      fi
    else
      # Insert entry after the description paragraph (after the second empty line in section)
      # The structure is: header, blank, description, blank, [entries...]
      # We want to insert after the blank line following the description
      empty_lines=$(sed -n "$((section_line + 1)),${section_end}p" "$queen_file" | grep -n "^$" | cut -d: -f1)
      # Get the second empty line (after description)
      insert_line=$(echo "$empty_lines" | sed -n '2p')
      if [[ -n "$insert_line" ]]; then
        insert_line=$((section_line + insert_line))
      else
        # Fallback: use first empty line
        insert_line=$(echo "$empty_lines" | head -1)
        if [[ -n "$insert_line" ]]; then
          insert_line=$((section_line + insert_line))
        else
          insert_line=$((section_line + 1))
        fi
      fi
      # Insert the entry after the found line
      sed "${insert_line}a\\
${entry}" "$queen_file" > "$tmp_file"
    fi

    # Update Evolution Log in temp file
    ev_entry="| ${ts} | ${colony_name} | promoted_${wisdom_type} | Added: ${content:0:50}... |"
    # Find the line after the separator in Evolution Log table
    ev_separator=$(grep -n "^|------|" "$tmp_file" | tail -1 | cut -d: -f1)

    # Use awk for cross-platform insertion
    awk -v line="$ev_separator" -v entry="$ev_entry" 'NR==line{print; print entry; next}1' "$tmp_file" > "${tmp_file}.ev" && mv "${tmp_file}.ev" "$tmp_file"

    # Update METADATA stats in temp file
    # Map wisdom_type to stat key (irregular plurals handled)
    case "$wisdom_type" in
      stack) stat_key="total_stack_entries" ;;
      philosophy) stat_key="total_philosophies" ;;
      *) stat_key="total_${wisdom_type}s" ;;
    esac
    # Read current count from temp file (which has the latest state)
    current_count=$(grep "\"${stat_key}\":" "$tmp_file" 2>/dev/null | grep -o '[0-9]*' | head -1 || true)
    current_count=${current_count:-0}
    new_count=$((current_count + 1))

    # Update last_evolved using awk
    awk -v ts="$ts" '/"last_evolved":/ { gsub(/"last_evolved": "[^"]*"/, "\"last_evolved\": \"" ts "\""); } {print}' "$tmp_file" > "${tmp_file}.meta" && mv "${tmp_file}.meta" "$tmp_file"

    # Update stats count using awk
    awk -v type="$stat_key" -v count="$new_count" '{
      gsub("\"" type "\": [0-9]*", "\"" type "\": " count)
      print
    }' "$tmp_file" > "${tmp_file}.stats" && mv "${tmp_file}.stats" "$tmp_file"

    # Add colony to colonies_contributed if not present
    if ! grep -q "\"${colony_name}\"" "$tmp_file"; then
      # Add to colonies_contributed array using awk - handle empty and non-empty arrays
      awk -v colony="$colony_name" '
        /"colonies_contributed": \[\]/ {
          gsub(/"colonies_contributed": \[\]/, "\"colonies_contributed\": [\"" colony "\"]")
          print
          next
        }
        /"colonies_contributed": \[/ && !/\]/ {
          # Multi-line array, add at next closing bracket
          print
          next
        }
        /"colonies_contributed": \[/ {
          # Single-line array with elements
          gsub(/\]$/, "\"" colony "\", ]")
          print
          next
        }
        { print }
      ' "$tmp_file" > "${tmp_file}.col" && mv "${tmp_file}.col" "$tmp_file"
    fi

    # Atomic move
    mv "$tmp_file" "$queen_file"

    json_ok "{\"promoted\":true,\"type\":\"$wisdom_type\",\"colony\":\"$colony_name\",\"timestamp\":\"$ts\",\"threshold\":$threshold,\"new_count\":$new_count}"
    ;;

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

    json_ok "{\"ok\":true,\"docs\":\"$docs\",\"dir\":\"$survey_dir\"}"
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

    json_ok "{\"ok\":true,\"counts\":\"$counts\"}"
    ;;

  checkpoint-check)
    allowlist_file="$DATA_DIR/checkpoint-allowlist.json"

    if [[ ! -f "$allowlist_file" ]]; then
      json_err "$E_FILE_NOT_FOUND" "Allowlist not found" "{\"path\":\"$allowlist_file\"}"
    fi

    # Get dirty files from git (staged or unstaged)
    dirty_files=$(git status --porcelain 2>/dev/null | awk '{print $2}' || true)

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
      elif [[ "$file" == runtime/* ]]; then
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
      result=$(jq -n \
        --argjson system "$(jq -R . < "$system_files_tmp" 2>/dev/null | jq -s .)" \
        --argjson user "$(jq -R . < "$user_files_tmp" 2>/dev/null | jq -s .)" \
        '{ok: true, system_files: $system, user_files: $user, has_user_files: ($user | length > 0)}')
    else
      # Fallback without jq - simple output
      system_count=$(wc -l < "$system_files_tmp" 2>/dev/null | tr -d ' ' || echo "0")
      user_count=$(wc -l < "$user_files_tmp" 2>/dev/null | tr -d ' ' || echo "0")
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
    # Or: eval "$(bash .aether/aether-utils.sh normalize-args)"
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
      # Preserve arguments with spaces by quoting
      for arg in "$@"; do
        if [[ "$arg" == *" "* ]] || [[ "$arg" == *"\t"* ]] || [[ "$arg" == *"\n"* ]]; then
          # Quote arguments containing whitespace
          normalized="$normalized \"$arg\""
        else
          normalized="$normalized $arg"
        fi
      done
      # Trim leading space
      normalized="${normalized# }"
    fi

    # Output normalized arguments
    echo "$normalized"
    exit 0
    ;;

  # Backward compatibility wrappers for session commands
  survey-verify-fresh)
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

  session-verify-fresh)
    # Generic session freshness verification
    # Usage: bash .aether/aether-utils.sh session-verify-fresh --command <name> [--force] <session_start_unixtime>
    # Returns: JSON with pass/fail status and file details

    # Parse arguments
    command_name=""
    force_mode=""
    session_start_time=""

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --command) command_name="$2"; shift 2 ;;
        --force) force_mode="--force"; shift ;;
        *) session_start_time="$1"; shift ;;
      esac
    done

    # Validate command name
    [[ -z "$command_name" ]] && json_err "$E_VALIDATION_FAILED" "Usage: session-verify-fresh --command <name> [--force] <session_start>"

    # Map command to directory and files (using env var override pattern)
    case "$command_name" in
      survey)
        session_dir="${SURVEY_DIR:-.aether/data/survey}"
        required_docs="PROVISIONS.md TRAILS.md BLUEPRINT.md CHAMBERS.md DISCIPLINES.md SENTINEL-PROTOCOLS.md PATHOGENS.md"
        ;;
      oracle)
        session_dir="${ORACLE_DIR:-.aether/oracle}"
        required_docs="progress.md research.json"
        ;;
      watch)
        session_dir="${WATCH_DIR:-.aether/data}"
        required_docs="watch-status.txt watch-progress.txt"
        ;;
      swarm)
        session_dir="${SWARM_DIR:-.aether/data/swarm}"
        required_docs="findings.json"
        ;;
      init)
        session_dir="${INIT_DIR:-.aether/data}"
        required_docs="COLONY_STATE.json constraints.json"
        ;;
      seal|entomb)
        session_dir="${ARCHIVE_DIR:-.aether/data/archive}"
        required_docs="manifest.json"
        ;;
      *)
        json_err "$E_VALIDATION_FAILED" "Unknown command: $command_name" '{"commands":["survey","oracle","watch","swarm","init","seal","entomb"]}'
        ;;
    esac

    # Initialize result arrays
    fresh_docs=""
    stale_docs=""
    missing_docs=""
    total_lines=0

    for doc in $required_docs; do
      doc_path="$session_dir/$doc"

      if [[ ! -f "$doc_path" ]]; then
        missing_docs="${missing_docs:+$missing_docs }$doc"
        continue
      fi

      # Get line count
      lines=$(wc -l < "$doc_path" 2>/dev/null | tr -d ' ' || echo "0")
      total_lines=$((total_lines + lines))

      # In force mode, accept any existing file
      if [[ "$force_mode" == "--force" ]]; then
        fresh_docs="${fresh_docs:+$fresh_docs }$doc"
        continue
      fi

      # Check timestamp if session_start_time provided
      if [[ -n "$session_start_time" ]]; then
        # Cross-platform stat: macOS uses -f %m, Linux uses -c %Y
        file_mtime=$(stat -f %m "$doc_path" 2>/dev/null || stat -c %Y "$doc_path" 2>/dev/null || echo "0")

        if [[ "$file_mtime" -ge "$session_start_time" ]]; then
          fresh_docs="${fresh_docs:+$fresh_docs }$doc"
        else
          stale_docs="${stale_docs:+$stale_docs }$doc"
        fi
      else
        # No start time provided - accept existing file (backward compatible)
        fresh_docs="${fresh_docs:+$fresh_docs }$doc"
      fi
    done

    # Determine pass/fail
    # pass = true if: no stale files (fresh files can coexist with missing files)
    # missing files are ok - they will be created during the session
    pass=false
    if [[ "$force_mode" == "--force" ]] || [[ -z "$stale_docs" ]]; then
      pass=true
    fi

    # Build JSON response
    fresh_json=""
    for item in $fresh_docs; do fresh_json="$fresh_json\"$item\","; done
    fresh_json="[${fresh_json%,}]"

    stale_json=""
    for item in $stale_docs; do stale_json="$stale_json\"$item\","; done
    stale_json="[${stale_json%,}]"

    missing_json=""
    for item in $missing_docs; do missing_json="$missing_json\"$item\","; done
    missing_json="[${missing_json%,}]"

    echo "{\"ok\":$pass,\"command\":\"$command_name\",\"fresh\":$fresh_json,\"stale\":$stale_json,\"missing\":$missing_json,\"total_lines\":$total_lines}"
    exit 0
    ;;

  session-clear)
    # Generic session file clearing
    # Usage: bash .aether/aether-utils.sh session-clear --command <name> [--dry-run]

    # Parse arguments
    command_name=""
    dry_run=""

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --command) command_name="$2"; shift 2 ;;
        --dry-run) dry_run="--dry-run"; shift ;;
        *) shift ;;
      esac
    done

    [[ -z "$command_name" ]] && json_err "$E_VALIDATION_FAILED" "Usage: session-clear --command <name> [--dry-run]"

    # Map command to directory and files
    case "$command_name" in
      survey)
        session_dir="${SURVEY_DIR:-.aether/data/survey}"
        files="PROVISIONS.md TRAILS.md BLUEPRINT.md CHAMBERS.md DISCIPLINES.md SENTINEL-PROTOCOLS.md PATHOGENS.md"
        ;;
      oracle)
        session_dir="${ORACLE_DIR:-.aether/oracle}"
        files="progress.md research.json .stop"
        # Also clear discoveries subdirectory
        subdir_files="discoveries/*"
        ;;
      watch)
        session_dir="${WATCH_DIR:-.aether/data}"
        files="watch-status.txt watch-progress.txt"
        ;;
      swarm)
        session_dir="${SWARM_DIR:-.aether/data/swarm}"
        files="findings.json display.json timing.json"
        ;;
      init)
        # Init clear is destructive - blocked for auto-clear
        json_err "$E_VALIDATION_FAILED" "Command 'init' is protected and cannot be auto-cleared. Use manual removal of COLONY_STATE.json if absolutely necessary."
        ;;
      seal|entomb)
        # Archive operations should never be auto-cleared
        json_err "$E_VALIDATION_FAILED" "Command '$command_name' is protected and cannot be auto-cleared. Archives and chambers must be managed manually."
        ;;
      *)
        json_err "$E_VALIDATION_FAILED" "Unknown command: $command_name"
        ;;
    esac

    cleared=""
    errors=""

    if [[ -d "$session_dir" && -n "$files" ]]; then
      for doc in $files; do
        doc_path="$session_dir/$doc"
        if [[ -f "$doc_path" ]]; then
          if [[ "$dry_run" == "--dry-run" ]]; then
            cleared="$cleared $doc"
          else
            if rm -f "$doc_path" 2>/dev/null; then
              cleared="$cleared $doc"
            else
              errors="$errors $doc"
            fi
          fi
        fi
      done

      # Handle oracle discoveries subdirectory
      if [[ "$command_name" == "oracle" && -d "$session_dir/discoveries" ]]; then
        if [[ "$dry_run" == "--dry-run" ]]; then
          cleared="$cleared discoveries/"
        else
          rm -rf "$session_dir/discoveries" 2>/dev/null && cleared="$cleared discoveries/" || errors="$errors discoveries/"
        fi
      fi
    fi

    json_ok "{\"command\":\"$command_name\",\"cleared\":\"${cleared// /}\",\"errors\":\"${errors// /}\",\"dry_run\":$([[ "$dry_run" == "--dry-run" ]] && echo "true" || echo "false")}"
    ;;

  pheromone-export)
    # Export pheromones to eternal XML format
    # Usage: pheromone-export [input_json] [output_xml]
    #   input_json: Path to pheromones.json (default: .aether/data/pheromones.json)
    #   output_xml: Path to output XML (default: ~/.aether/eternal/pheromones.xml)

    input_json="${1:-.aether/data/pheromones.json}"
    output_xml="${2:-$HOME/.aether/eternal/pheromones.xml}"
    schema_file="${3:-$SCRIPT_DIR/schemas/pheromone.xsd}"

    # Ensure xml-utils.sh is sourced
    if ! type pheromone-export &>/dev/null; then
      [[ -f "$SCRIPT_DIR/utils/xml-utils.sh" ]] && source "$SCRIPT_DIR/utils/xml-utils.sh"
    fi

    if type pheromone-export &>/dev/null; then
      pheromone-export "$input_json" "$output_xml" "$schema_file"
    else
      json_err "$E_DEPENDENCY_MISSING" "xml-utils.sh not available for pheromone export"
    fi
    ;;

  pheromone-write)
    # Write a pheromone signal to pheromones.json
    # Usage: pheromone-write <type> <content> [--strength N] [--ttl TTL] [--source SOURCE] [--reason REASON]
    #   type:       FOCUS, REDIRECT, or FEEDBACK
    #   content:    signal text (required, max 500 chars)
    #   --strength: 0.0-1.0 (defaults: REDIRECT=0.9, FOCUS=0.8, FEEDBACK=0.7)
    #   --ttl:      phase_end (default), 2h, 1d, 7d, 30d, etc.
    #   --source:   user (default), worker:builder, system
    #   --reason:   human-readable explanation

    pw_type="${1:-}"
    pw_content="${2:-}"

    # Validate type
    if [[ -z "$pw_type" ]]; then
      json_err "$E_VALIDATION_FAILED" "pheromone-write requires <type> argument (FOCUS, REDIRECT, or FEEDBACK)"
    fi

    pw_type=$(echo "$pw_type" | tr '[:lower:]' '[:upper:]')
    case "$pw_type" in
      FOCUS|REDIRECT|FEEDBACK) ;;
      *) json_err "$E_VALIDATION_FAILED" "Invalid pheromone type: $pw_type. Must be FOCUS, REDIRECT, or FEEDBACK" ;;
    esac

    if [[ -z "$pw_content" ]]; then
      json_err "$E_VALIDATION_FAILED" "pheromone-write requires <content> argument"
    fi

    # Parse optional flags from remaining args (after type and content)
    pw_strength=""
    pw_ttl="phase_end"
    pw_source="user"
    pw_reason=""

    shift 2  # shift past type and content
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --strength) pw_strength="$2"; shift 2 ;;
        --ttl)      pw_ttl="$2"; shift 2 ;;
        --source)   pw_source="$2"; shift 2 ;;
        --reason)   pw_reason="$2"; shift 2 ;;
        *) shift ;;
      esac
    done

    # Apply default strength by type
    if [[ -z "$pw_strength" ]]; then
      case "$pw_type" in
        REDIRECT) pw_strength="0.9" ;;
        FOCUS)    pw_strength="0.8" ;;
        FEEDBACK) pw_strength="0.7" ;;
      esac
    fi

    # Apply default reason by type
    if [[ -z "$pw_reason" ]]; then
      pw_type_lower_r=$(echo "$pw_type" | tr '[:upper:]' '[:lower:]')
      pw_reason="User emitted via /ant:${pw_type_lower_r}"
    fi

    # Set priority by type
    case "$pw_type" in
      REDIRECT) pw_priority="high" ;;
      FOCUS)    pw_priority="normal" ;;
      FEEDBACK) pw_priority="low" ;;
    esac

    # Generate ID and timestamps
    pw_epoch=$(date +%s)
    pw_epoch_ms="${pw_epoch}000"
    pw_type_lower=$(echo "$pw_type" | tr '[:upper:]' '[:lower:]')
    pw_id="sig_${pw_type_lower}_${pw_epoch_ms}"
    pw_created=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Compute expires_at from TTL
    if [[ "$pw_ttl" == "phase_end" ]]; then
      pw_expires="phase_end"
    else
      pw_ttl_secs=0
      if [[ "$pw_ttl" =~ ^([0-9]+)m$ ]]; then
        pw_ttl_secs=$(( ${BASH_REMATCH[1]} * 60 ))
      elif [[ "$pw_ttl" =~ ^([0-9]+)h$ ]]; then
        pw_ttl_secs=$(( ${BASH_REMATCH[1]} * 3600 ))
      elif [[ "$pw_ttl" =~ ^([0-9]+)d$ ]]; then
        pw_ttl_secs=$(( ${BASH_REMATCH[1]} * 86400 ))
      fi
      if [[ $pw_ttl_secs -gt 0 ]]; then
        pw_expires_epoch=$(( pw_epoch + pw_ttl_secs ))
        pw_expires=$(date -u -r "$pw_expires_epoch" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || \
                     date -u -d "@$pw_expires_epoch" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || \
                     echo "phase_end")
      else
        pw_expires="phase_end"
      fi
    fi

    pw_file="$DATA_DIR/pheromones.json"

    # Initialize pheromones.json if missing
    if [[ ! -f "$pw_file" ]]; then
      pw_colony_id="aether-dev"
      if [[ -f "$DATA_DIR/COLONY_STATE.json" ]]; then
        pw_colony_id=$(jq -r '.session_id // "aether-dev"' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null || echo "aether-dev")
      fi
      printf '{\n  "version": "1.0.0",\n  "colony_id": "%s",\n  "generated_at": "%s",\n  "signals": []\n}\n' \
        "$pw_colony_id" "$pw_created" > "$pw_file"
    fi

    # Build signal object and append to pheromones.json
    pw_signal=$(jq -n \
      --arg id "$pw_id" \
      --arg type "$pw_type" \
      --arg priority "$pw_priority" \
      --arg source "$pw_source" \
      --arg created_at "$pw_created" \
      --arg expires_at "$pw_expires" \
      --argjson active true \
      --argjson strength "$pw_strength" \
      --arg reason "$pw_reason" \
      --arg content "$pw_content" \
      '{id: $id, type: $type, priority: $priority, source: $source, created_at: $created_at, expires_at: $expires_at, active: $active, strength: ($strength | tonumber), reason: $reason, content: {text: $content}}')

    pw_updated=$(jq --argjson sig "$pw_signal" '.signals += [$sig]' "$pw_file" 2>/dev/null)
    if [[ -z "$pw_updated" ]]; then
      json_err "${E_JSON_INVALID:-E_JSON_INVALID}" "Failed to update pheromones.json — jq parse error"
    fi
    echo "$pw_updated" > "$pw_file"

    # Backward compatibility: also write to constraints.json
    pw_cfile="$DATA_DIR/constraints.json"
    if [[ "$pw_type" == "FOCUS" ]]; then
      if [[ ! -f "$pw_cfile" ]]; then
        echo '{"version":"1.0","focus":[],"constraints":[]}' > "$pw_cfile"
      fi
      pw_cfile_updated=$(jq --arg txt "$pw_content" '
        .focus += [$txt] |
        if (.focus | length) > 5 then .focus = .focus[-5:] else . end
      ' "$pw_cfile" 2>/dev/null)
      [[ -n "$pw_cfile_updated" ]] && echo "$pw_cfile_updated" > "$pw_cfile"
    elif [[ "$pw_type" == "REDIRECT" ]]; then
      if [[ ! -f "$pw_cfile" ]]; then
        echo '{"version":"1.0","focus":[],"constraints":[]}' > "$pw_cfile"
      fi
      pw_constraint=$(jq -n \
        --arg id "c_${pw_epoch}" \
        --arg content "$pw_content" \
        --arg source "user:redirect" \
        --arg created_at "$pw_created" \
        '{id: $id, type: "AVOID", content: $content, source: $source, created_at: $created_at}')
      pw_cfile_updated=$(jq --argjson c "$pw_constraint" '
        .constraints += [$c] |
        if (.constraints | length) > 10 then .constraints = .constraints[-10:] else . end
      ' "$pw_cfile" 2>/dev/null)
      [[ -n "$pw_cfile_updated" ]] && echo "$pw_cfile_updated" > "$pw_cfile"
    fi

    # Get active signal count
    pw_active_count=$(jq '[.signals[] | select(.active == true)] | length' "$pw_file" 2>/dev/null || echo "0")

    json_ok "{\"signal_id\":\"$pw_id\",\"type\":\"$pw_type\",\"active_count\":$pw_active_count}"
    ;;

  pheromone-count)
    # Count active pheromone signals by type
    # Usage: pheromone-count
    # Returns: JSON with per-type counts

    pc_file="$DATA_DIR/pheromones.json"

    if [[ ! -f "$pc_file" ]]; then
      json_ok '{"focus":0,"redirect":0,"feedback":0,"total":0}'
    else
      pc_result=$(jq -c '{
        focus:    ([.signals[] | select(.active == true and .type == "FOCUS")]    | length),
        redirect: ([.signals[] | select(.active == true and .type == "REDIRECT")] | length),
        feedback: ([.signals[] | select(.active == true and .type == "FEEDBACK")] | length),
        total:    ([.signals[] | select(.active == true)]                          | length)
      }' "$pc_file" 2>/dev/null)
      if [[ -z "$pc_result" ]]; then
        json_ok '{"focus":0,"redirect":0,"feedback":0,"total":0}'
      else
        json_ok "$pc_result"
      fi
    fi
    ;;

  pheromone-read)
    # Read pheromones from colony data with decay calculation
    # Usage: pheromone-read [type]
    #   type: Filter by pheromone type (focus, redirect, feedback) or 'all' (default: all)
    # Returns: JSON object with pheromones array including effective_strength

    pher_type="${1:-all}"
    pher_file="$DATA_DIR/pheromones.json"

    # Check if file exists
    if [[ ! -f "$pher_file" ]]; then
      json_err "$E_FILE_NOT_FOUND" "Pheromones file not found. Run /ant:colonize first to initialize the colony."
    fi

    # Get current epoch for decay calculation
    pher_now=$(date +%s)

    # Apply decay and expiry at read time
    # Decay rates: FOCUS=30d, REDIRECT=60d, FEEDBACK/PATTERN=90d
    # effective_strength = original_strength * (1 - elapsed_days / decay_days)
    # If effective_strength < 0.1, mark inactive
    # Also check expires_at: if not "phase_end" and past expiry, mark inactive
    pher_type_upper=$(echo "$pher_type" | tr '[:lower:]' '[:upper:]')

    pher_result=$(jq -c \
      --argjson now "$pher_now" \
      --arg type_filter "$pher_type_upper" \
      '
      # Rough ISO-8601 to epoch: accumulate years*365d + month*30d + days + time
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
        if t == "FOCUS"    then 30
        elif t == "REDIRECT" then 60
        else 90
        end;

      .signals | map(
        (to_epoch(.created_at)) as $created_epoch |
        (if $created_epoch != null then ($now - $created_epoch) / 86400 else 0 end) as $elapsed_days |
        (decay_days(.type)) as $dd |
        ((.strength // 0.8) * (1 - ($elapsed_days / $dd))) as $eff_raw |
        (if $eff_raw < 0 then 0 else $eff_raw end) as $eff |
        (to_epoch(.expires_at)) as $exp_epoch |
        ($exp_epoch != null and $exp_epoch <= $now) as $expired |
        ($eff < 0.1 or $expired) as $deactivate |
        . + {
          effective_strength: (($eff * 100 | round) / 100),
          active: (if $deactivate then false else (.active // true) end)
        }
      ) |
      map(select(.active == true)) |
      if $type_filter != "ALL" then
        map(select(.type == $type_filter))
      else
        .
      end
      ' "$pher_file" 2>/dev/null)

    if [[ -z "$pher_result" || "$pher_result" == "null" ]]; then
      json_ok '{"version":"1.0.0","signals":[]}'
    else
      pher_version=$(jq -r '.version // "1.0.0"' "$pher_file" 2>/dev/null || echo "1.0.0")
      pher_colony=$(jq -r '.colony_id // "unknown"' "$pher_file" 2>/dev/null || echo "unknown")
      json_ok "{\"version\":\"$pher_version\",\"colony_id\":\"$pher_colony\",\"signals\":$pher_result}"
    fi
    ;;

  instinct-read)
    # Read learned instincts from COLONY_STATE.json memory
    # Usage: instinct-read [--min-confidence N] [--max N] [--domain DOMAIN]
    # Returns: JSON with filtered, confidence-sorted instincts

    ir_min_confidence="0.5"
    ir_max="5"
    ir_domain=""

    # Parse flags from positional args
    ir_shift=1
    while [[ $ir_shift -le $# ]]; do
      eval "ir_arg=\${$ir_shift}"
      ir_shift=$((ir_shift + 1))
      case "$ir_arg" in
        --min-confidence)
          eval "ir_min_confidence=\${$ir_shift}"
          ir_shift=$((ir_shift + 1))
          ;;
        --max)
          eval "ir_max=\${$ir_shift}"
          ir_shift=$((ir_shift + 1))
          ;;
        --domain)
          eval "ir_domain=\${$ir_shift}"
          ir_shift=$((ir_shift + 1))
          ;;
      esac
    done

    ir_state_file="$DATA_DIR/COLONY_STATE.json"

    if [[ ! -f "$ir_state_file" ]]; then
      json_err "$E_FILE_NOT_FOUND" "COLONY_STATE.json not found. Run /ant:init first."
    fi

    # Check if memory.instincts exists
    ir_has_instincts=$(jq 'if .memory.instincts then "yes" else "no" end' "$ir_state_file" 2>/dev/null || echo "no")
    if [[ "$ir_has_instincts" != '"yes"' ]]; then
      json_ok '{"instincts":[],"total":0,"filtered":0}'
    fi

    ir_result=$(jq -c \
      --argjson min_conf "$ir_min_confidence" \
      --argjson max_count "$ir_max" \
      --arg domain_filter "$ir_domain" \
      '
      (.memory.instincts // []) as $all |
      ($all | length) as $total |
      $all
      | map(select(
          (.confidence // 0) >= $min_conf
          and (.status // "hypothesis") != "disproven"
          and (if $domain_filter != "" then (.domain // "") == $domain_filter else true end)
        ))
      | sort_by(-.confidence)
      | .[:$max_count]
      | {
          instincts: .,
          total: $total,
          filtered: (. | length)
        }
      ' "$ir_state_file" 2>/dev/null)

    if [[ -z "$ir_result" || "$ir_result" == "null" ]]; then
      json_ok '{"instincts":[],"total":0,"filtered":0}'
    else
      json_ok "$ir_result"
    fi
    ;;

  pheromone-prime)
    # Combine active pheromone signals and learned instincts into a prompt-ready block
    # Usage: pheromone-prime
    # Returns: JSON with signal_count, instinct_count, prompt_section, log_line

    pp_pher_file="$DATA_DIR/pheromones.json"
    pp_state_file="$DATA_DIR/COLONY_STATE.json"
    pp_now=$(date +%s)

    # Read active signals (same decay logic as pheromone-read)
    pp_signals="[]"
    if [[ -f "$pp_pher_file" ]]; then
      pp_signals=$(jq -c \
        --argjson now "$pp_now" \
        '
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
          if t == "FOCUS"    then 30
          elif t == "REDIRECT" then 60
          else 90
          end;

        .signals | map(
          (to_epoch(.created_at)) as $created_epoch |
          (if $created_epoch != null then ($now - $created_epoch) / 86400 else 0 end) as $elapsed_days |
          (decay_days(.type)) as $dd |
          ((.strength // 0.8) * (1 - ($elapsed_days / $dd))) as $eff_raw |
          (if $eff_raw < 0 then 0 else $eff_raw end) as $eff |
          (to_epoch(.expires_at)) as $exp_epoch |
          ($exp_epoch != null and $exp_epoch <= $now) as $expired |
          ($eff < 0.1 or $expired) as $deactivate |
          . + {
            effective_strength: (($eff * 100 | round) / 100),
            active: (if $deactivate then false else (.active // true) end)
          }
        ) |
        map(select(.active == true))
        ' "$pp_pher_file" 2>/dev/null || echo "[]")
    fi

    if [[ -z "$pp_signals" || "$pp_signals" == "null" ]]; then
      pp_signals="[]"
    fi

    # Read instincts (confidence >= 0.5, not disproven, max 5)
    pp_instincts="[]"
    if [[ -f "$pp_state_file" ]]; then
      pp_instincts=$(jq -c \
        '
        (.memory.instincts // [])
        | map(select(
            (.confidence // 0) >= 0.5
            and (.status // "hypothesis") != "disproven"
          ))
        | sort_by(-.confidence)
        | .[:5]
        ' "$pp_state_file" 2>/dev/null || echo "[]")
    fi

    if [[ -z "$pp_instincts" || "$pp_instincts" == "null" ]]; then
      pp_instincts="[]"
    fi

    pp_signal_count=$(echo "$pp_signals" | jq 'length' 2>/dev/null || echo "0")
    pp_instinct_count=$(echo "$pp_instincts" | jq 'length' 2>/dev/null || echo "0")

    # Build prompt section
    if [[ "$pp_signal_count" -eq 0 && "$pp_instinct_count" -eq 0 ]]; then
      pp_section=""
      pp_log_line="Primed: 0 signals, 0 instincts"
    else
      pp_section="--- ACTIVE SIGNALS (Colony Guidance) ---"$'\n'

      # FOCUS signals
      pp_focus=$(echo "$pp_signals" | jq -r 'map(select(.type == "FOCUS")) | .[] | "[" + ((.effective_strength * 10 | round) / 10 | tostring) + "] " + (.content.text // (if (.content | type) == "string" then .content else "" end))' 2>/dev/null || echo "")
      if [[ -n "$pp_focus" ]]; then
        pp_section+=$'\n'"FOCUS (Pay attention to):"$'\n'"$pp_focus"$'\n'
      fi

      # REDIRECT signals
      pp_redirect=$(echo "$pp_signals" | jq -r 'map(select(.type == "REDIRECT")) | .[] | "[" + ((.effective_strength * 10 | round) / 10 | tostring) + "] " + (.content.text // (if (.content | type) == "string" then .content else "" end))' 2>/dev/null || echo "")
      if [[ -n "$pp_redirect" ]]; then
        pp_section+=$'\n'"REDIRECT (HARD CONSTRAINTS - MUST follow):"$'\n'"$pp_redirect"$'\n'
      fi

      # FEEDBACK signals
      pp_feedback=$(echo "$pp_signals" | jq -r 'map(select(.type == "FEEDBACK")) | .[] | "[" + ((.effective_strength * 10 | round) / 10 | tostring) + "] " + (.content.text // (if (.content | type) == "string" then .content else "" end))' 2>/dev/null || echo "")
      if [[ -n "$pp_feedback" ]]; then
        pp_section+=$'\n'"FEEDBACK (Flexible guidance):"$'\n'"$pp_feedback"$'\n'
      fi

      # Instincts section
      if [[ "$pp_instinct_count" -gt 0 ]]; then
        pp_section+=$'\n'"--- INSTINCTS (Learned Behaviors) ---"$'\n'
        pp_section+="Weight by confidence - higher = stronger guidance:"$'\n'
        pp_instinct_lines=$(echo "$pp_instincts" | jq -r '.[] | "[" + ((.confidence * 10 | round) / 10 | tostring) + "] When " + .trigger + " -> " + .action + " (" + (.domain // "general") + ")"' 2>/dev/null || echo "")
        if [[ -n "$pp_instinct_lines" ]]; then
          pp_section+=$'\n'"$pp_instinct_lines"$'\n'
        fi
      fi

      pp_section+=$'\n'"--- END COLONY CONTEXT ---"

      pp_log_line="Primed: ${pp_signal_count} signals, ${pp_instinct_count} instincts"
    fi

    # Escape section for JSON embedding (use printf to avoid appending extra newline)
    pp_section_json=$(printf '%s' "$pp_section" | jq -Rs '.' 2>/dev/null || echo '""')
    pp_log_json=$(printf '%s' "$pp_log_line" | jq -Rs '.' 2>/dev/null || echo '"Primed: 0 signals, 0 instincts"')

    json_ok "{\"signal_count\":$pp_signal_count,\"instinct_count\":$pp_instinct_count,\"prompt_section\":$pp_section_json,\"log_line\":$pp_log_json}"
    ;;

  pheromone-expire)
    # Archive expired pheromone signals to midden
    # Usage: pheromone-expire [--phase-end-only]
    #
    # Two modes:
    #   --phase-end-only  Only expire signals where expires_at == "phase_end"
    #   (no flag)         Expire signals where expires_at is an ISO-8601 timestamp
    #                     <= now, AND signals where effective_strength < 0.1

    phe_phase_end_only="false"
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --phase-end-only) phe_phase_end_only="true"; shift ;;
        *) shift ;;
      esac
    done

    phe_pheromones_file="$DATA_DIR/pheromones.json"
    phe_midden_dir="$DATA_DIR/midden"
    phe_midden_file="$phe_midden_dir/midden.json"

    # Handle missing pheromones.json gracefully
    if [[ ! -f "$phe_pheromones_file" ]]; then
      json_ok '{"expired_count":0,"remaining_active":0,"midden_total":0}'
      exit 0
    fi

    # Ensure midden directory and file exist
    mkdir -p "$phe_midden_dir"
    if [[ ! -f "$phe_midden_file" ]]; then
      printf '%s\n' '{"version":"1.0.0","archived_at_count":0,"signals":[]}' > "$phe_midden_file"
    fi

    phe_now_epoch=$(date +%s)
    phe_archived_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Compute pause_duration from COLONY_STATE.json (pause-aware TTL)
    phe_pause_duration=0
    if [[ -f "$DATA_DIR/COLONY_STATE.json" ]]; then
      phe_paused_at=$(jq -r '.paused_at // empty' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null || true)
      phe_resumed_at=$(jq -r '.resumed_at // empty' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null || true)
      if [[ -n "$phe_paused_at" && -n "$phe_resumed_at" ]]; then
        phe_paused_epoch=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "$phe_paused_at" +%s 2>/dev/null || date -d "$phe_paused_at" +%s 2>/dev/null || echo 0)
        phe_resumed_epoch=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "$phe_resumed_at" +%s 2>/dev/null || date -d "$phe_resumed_at" +%s 2>/dev/null || echo 0)
        if [[ "$phe_resumed_epoch" -gt "$phe_paused_epoch" ]]; then
          phe_pause_duration=$(( phe_resumed_epoch - phe_paused_epoch ))
        fi
      fi
    fi

    # Identify expired signal IDs
    # We'll use jq to find signals to expire, then update in bash
    if [[ "$phe_phase_end_only" == "true" ]]; then
      # Only expire signals where expires_at == "phase_end"
      phe_expired_ids=$(jq -r '.signals[] | select(.active == true and .expires_at == "phase_end") | .id' "$phe_pheromones_file" 2>/dev/null || true)
    else
      # Expire time-based expired signals (pause-aware) AND decay-expired signals
      phe_expired_ids=$(jq -r --argjson now "$phe_now_epoch" --argjson pause_secs "$phe_pause_duration" '
        .signals[] |
        select(.active == true) |
        select(
          (.expires_at != "phase_end" and .expires_at != null and .expires_at != "") and
          (
            # ISO-8601 timestamp expiry (pause-aware: add pause_duration to expires_at before comparing)
            (
              .expires_at |
              # Convert ISO-8601 to approximate epoch via string parsing
              (
                (split("T")[0] | split("-")) as $d |
                (split("T")[1] | split(":")) as $t |
                ($d[0] | tonumber) as $y |
                ($d[1] | tonumber) as $mo |
                ($d[2] | tonumber) as $day |
                ($t[0] | tonumber) as $h |
                ($t[1] | tonumber) as $m |
                (($t[2] // "0") | gsub("[^0-9]";"") | if . == "" then 0 else tonumber end) as $s |
                # Rough epoch: years*365.25*86400 + months*30.44*86400 + day*86400 + time
                (($y - 1970) * 31557600) + (($mo - 1) * 2629800) + (($day - 1) * 86400) + ($h * 3600) + ($m * 60) + $s
              )
            ) + $pause_secs <= $now
          )
        ) |
        .id
      ' "$phe_pheromones_file" 2>/dev/null || true)
    fi

    # Count expired signals
    phe_expired_count=0
    if [[ -n "$phe_expired_ids" ]]; then
      phe_expired_count=$(echo "$phe_expired_ids" | grep -c . 2>/dev/null || echo 0)
    fi

    # If nothing to expire, return counts
    if [[ "$phe_expired_count" -eq 0 ]]; then
      phe_remaining=$(jq '[.signals[] | select(.active == true)] | length' "$phe_pheromones_file" 2>/dev/null || echo 0)
      phe_midden_total=$(jq '.signals | length' "$phe_midden_file" 2>/dev/null || echo 0)
      json_ok "{\"expired_count\":0,\"remaining_active\":$phe_remaining,\"midden_total\":$phe_midden_total}"
      exit 0
    fi

    # Build jq args for IDs to expire
    phe_id_array=$(echo "$phe_expired_ids" | jq -R . | jq -s . 2>/dev/null || echo '[]')

    # Extract expired signal objects (with archived_at added)
    phe_expired_objects=$(jq --argjson ids "$phe_id_array" --arg archived_at "$phe_archived_at" '
      [.signals[] | select(.id as $id | $ids | any(. == $id)) | . + {"archived_at": $archived_at, "active": false}]
    ' "$phe_pheromones_file" 2>/dev/null || echo '[]')

    # Update pheromones.json: set active=false for expired signals (do NOT remove them)
    phe_updated_pheromones=$(jq --argjson ids "$phe_id_array" '
      .signals = [.signals[] | if (.id as $id | $ids | any(. == $id)) then .active = false else . end]
    ' "$phe_pheromones_file" 2>/dev/null)

    if [[ -n "$phe_updated_pheromones" ]]; then
      printf '%s\n' "$phe_updated_pheromones" > "$phe_pheromones_file"
    fi

    # Append expired signals to midden.json
    phe_midden_updated=$(jq --argjson new_signals "$phe_expired_objects" '
      .signals += $new_signals |
      .archived_at_count = (.signals | length)
    ' "$phe_midden_file" 2>/dev/null)

    if [[ -n "$phe_midden_updated" ]]; then
      printf '%s\n' "$phe_midden_updated" > "$phe_midden_file"
    fi

    phe_remaining_active=$(jq '[.signals[] | select(.active == true)] | length' "$phe_pheromones_file" 2>/dev/null || echo 0)
    phe_midden_total=$(jq '.signals | length' "$phe_midden_file" 2>/dev/null || echo 0)

    json_ok "{\"expired_count\":$phe_expired_count,\"remaining_active\":$phe_remaining_active,\"midden_total\":$phe_midden_total}"
    ;;

  eternal-init)
    # Initialize the ~/.aether/eternal/ directory and memory.json schema
    # Usage: eternal-init
    # Idempotent: safe to call multiple times

    ei_eternal_dir="$HOME/.aether/eternal"
    ei_memory_file="$ei_eternal_dir/memory.json"
    ei_already_existed="false"

    mkdir -p "$ei_eternal_dir"

    if [[ -f "$ei_memory_file" ]]; then
      ei_already_existed="true"
    else
      ei_created_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
      printf '%s\n' "{
  \"version\": \"1.0.0\",
  \"created_at\": \"$ei_created_at\",
  \"colonies\": [],
  \"high_value_signals\": [],
  \"cross_session_patterns\": []
}" > "$ei_memory_file"
    fi

    json_ok "{\"dir\":\"$ei_eternal_dir\",\"initialized\":true,\"already_existed\":$ei_already_existed}"
    ;;

  # ============================================================================
  # XML Exchange Commands
  # ============================================================================

  pheromone-export-xml)
    # Export pheromones.json to XML format
    # Usage: pheromone-export-xml [output_file]
    # Default output: .aether/exchange/pheromones.xml

    pex_output="${1:-$SCRIPT_DIR/exchange/pheromones.xml}"
    pex_pheromones="$DATA_DIR/pheromones.json"

    # Graceful degradation: check for xmllint
    if ! command -v xmllint >/dev/null 2>&1; then
      json_err "xmllint not available — XML features require libxml2 (install: xcode-select --install on macOS)"
    fi

    # Check pheromones.json exists
    if [[ ! -f "$pex_pheromones" ]]; then
      json_err "pheromones.json not found at $pex_pheromones"
    fi

    # Ensure output directory exists
    mkdir -p "$(dirname "$pex_output")"

    # Source the exchange script
    source "$SCRIPT_DIR/exchange/pheromone-xml.sh"

    # Call the export function
    xml-pheromone-export "$pex_pheromones" "$pex_output"
    ;;

  pheromone-import-xml)
    # Import pheromone signals from XML into pheromones.json
    # Usage: pheromone-import-xml <xml_file>

    pix_xml="${1:-}"
    pix_pheromones="$DATA_DIR/pheromones.json"

    if [[ -z "$pix_xml" ]]; then
      json_err "Missing XML file argument. Usage: pheromone-import-xml <xml_file>"
    fi

    if [[ ! -f "$pix_xml" ]]; then
      json_err "XML file not found: $pix_xml"
    fi

    # Graceful degradation: check for xmllint
    if ! command -v xmllint >/dev/null 2>&1; then
      json_err "xmllint not available — XML features require libxml2 (install: xcode-select --install on macOS)"
    fi

    # Source the exchange script
    source "$SCRIPT_DIR/exchange/pheromone-xml.sh"

    # Import XML to get JSON signals
    pix_imported=$(xml-pheromone-import "$pix_xml")

    # If pheromones.json exists, merge; otherwise create
    if [[ -f "$pix_pheromones" ]]; then
      # Extract imported signals and merge into existing, deduplicate by id
      pix_merged=$(jq -s '
        .[0] as $existing | .[1] as $imported |
        ($imported.result.signals // []) as $new_signals |
        $existing | .signals = (
          [.signals[], $new_signals[]] |
          group_by(.id) | map(last)
        )
      ' "$pix_pheromones" <(echo "$pix_imported") 2>/dev/null)

      if [[ -n "$pix_merged" ]]; then
        printf '%s\n' "$pix_merged" > "$pix_pheromones"
      fi
    fi

    pix_count=$(echo "$pix_imported" | jq '.result.signals | length' 2>/dev/null || echo 0)
    json_ok "{\"imported\":true,\"signal_count\":$pix_count,\"source\":\"$pix_xml\"}"
    ;;

  pheromone-validate-xml)
    # Validate pheromone XML against XSD schema
    # Usage: pheromone-validate-xml <xml_file>

    pvx_xml="${1:-}"
    pvx_xsd="$SCRIPT_DIR/schemas/pheromone.xsd"

    if [[ -z "$pvx_xml" ]]; then
      json_err "Missing XML file argument. Usage: pheromone-validate-xml <xml_file>"
    fi

    if [[ ! -f "$pvx_xml" ]]; then
      json_err "XML file not found: $pvx_xml"
    fi

    # Graceful degradation: check for xmllint
    if ! command -v xmllint >/dev/null 2>&1; then
      json_err "xmllint not available — XML features require libxml2 (install: xcode-select --install on macOS)"
    fi

    # Source the exchange script
    source "$SCRIPT_DIR/exchange/pheromone-xml.sh"

    # Call validate function
    xml-pheromone-validate "$pvx_xml" "$pvx_xsd"
    ;;

  wisdom-export-xml)
    # Export queen wisdom to XML format
    # Usage: wisdom-export-xml [input_json] [output_xml]
    # Default input: .aether/data/queen-wisdom.json
    # Default output: .aether/exchange/queen-wisdom.xml

    wex_input="${1:-$DATA_DIR/queen-wisdom.json}"
    wex_output="${2:-$SCRIPT_DIR/exchange/queen-wisdom.xml}"

    # Graceful degradation: check for xmllint
    if ! command -v xmllint >/dev/null 2>&1; then
      json_err "xmllint not available — XML features require libxml2 (install: xcode-select --install on macOS)"
    fi

    # Look for wisdom data: check specified file, then COLONY_STATE memory
    if [[ ! -f "$wex_input" ]]; then
      # Try to extract from COLONY_STATE.json memory field
      if [[ -f "$DATA_DIR/COLONY_STATE.json" ]]; then
        wex_memory=$(jq '.memory // {}' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null || echo '{}')
        if [[ "$wex_memory" != "{}" && "$wex_memory" != "null" ]]; then
          # Create minimal wisdom JSON from colony memory
          wex_created_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
          printf '%s\n' "{
  \"version\": \"1.0.0\",
  \"metadata\": {\"created\": \"$wex_created_at\", \"colony_id\": \"$(jq -r '.goal // \"unknown\"' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null)\"},
  \"philosophies\": [],
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
      json_err "Missing XML file argument. Usage: wisdom-import-xml <xml_file> [output_json]"
    fi

    if [[ ! -f "$wix_xml" ]]; then
      json_err "XML file not found: $wix_xml"
    fi

    # Graceful degradation: check for xmllint
    if ! command -v xmllint >/dev/null 2>&1; then
      json_err "xmllint not available — XML features require libxml2 (install: xcode-select --install on macOS)"
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
      json_err "xmllint not available — XML features require libxml2 (install: xcode-select --install on macOS)"
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
            }' "$manifest" 2>/dev/null || true
          done | jq -s '.' 2>/dev/null || echo '[]'
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
      json_err "Missing XML file argument. Usage: registry-import-xml <xml_file> [output_json]"
    fi

    if [[ ! -f "$rix_xml" ]]; then
      json_err "XML file not found: $rix_xml"
    fi

    # Graceful degradation: check for xmllint
    if ! command -v xmllint >/dev/null 2>&1; then
      json_err "xmllint not available — XML features require libxml2 (install: xcode-select --install on macOS)"
    fi

    # Ensure output directory exists
    mkdir -p "$(dirname "$rix_output")"

    # Source the exchange script
    source "$SCRIPT_DIR/exchange/registry-xml.sh"

    # Call the import function
    xml-registry-import "$rix_xml" "$rix_output"
    ;;

  colony-archive-xml)
    # Export combined colony archive XML containing pheromones, wisdom, and registry
    # Usage: colony-archive-xml [output_file]
    # Default output: .aether/exchange/colony-archive.xml
    # Always filters to active-only pheromone signals

    # Graceful degradation: check for xmllint
    if ! command -v xmllint >/dev/null 2>&1; then
      json_err "xmllint not available — XML features require libxml2 (install: xcode-select --install on macOS)"
    fi

    cax_output="${1:-$SCRIPT_DIR/exchange/colony-archive.xml}"
    mkdir -p "$(dirname "$cax_output")"

    # Step 1: Filter active-only pheromone signals to a temp file
    cax_tmp_pheromones=$(mktemp)
    if [[ -f "$DATA_DIR/pheromones.json" ]]; then
      jq '{
        version: .version,
        colony_id: .colony_id,
        generated_at: .generated_at,
        signals: [.signals[] | select(.active == true)]
      }' "$DATA_DIR/pheromones.json" > "$cax_tmp_pheromones" 2>/dev/null
    else
      printf '%s\n' '{"version":"1.0","colony_id":"unknown","generated_at":"","signals":[]}' > "$cax_tmp_pheromones"
    fi

    # Step 2: Export each section to temp XML files
    cax_tmp_dir=$(mktemp -d)

    # Pheromone section (using filtered active-only)
    source "$SCRIPT_DIR/exchange/pheromone-xml.sh"
    xml-pheromone-export "$cax_tmp_pheromones" "$cax_tmp_dir/pheromones.xml" 2>/dev/null || true

    # Wisdom section — reuse wisdom-export-xml fallback logic
    source "$SCRIPT_DIR/exchange/wisdom-xml.sh"
    cax_wisdom_input="$DATA_DIR/queen-wisdom.json"
    if [[ ! -f "$cax_wisdom_input" ]]; then
      # Try extracting from COLONY_STATE.json memory field
      if [[ -f "$DATA_DIR/COLONY_STATE.json" ]]; then
        cax_wex_memory=$(jq '.memory // {}' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null || echo '{}')
        if [[ "$cax_wex_memory" != "{}" && "$cax_wex_memory" != "null" ]]; then
          cax_wex_created_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
          cax_wisdom_input="$cax_tmp_dir/wisdom-input.json"
          printf '%s\n' "{
  \"version\": \"1.0.0\",
  \"metadata\": {\"created\": \"$cax_wex_created_at\", \"colony_id\": \"$(jq -r '.goal // \"unknown\"' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null)\"},
  \"philosophies\": [],
  \"patterns\": $(echo "$cax_wex_memory" | jq '[.instincts // [] | .[] | {"id": (. | @base64), "content": ., "confidence": 0.7, "domain": "general", "source": "colony_memory"}]' 2>/dev/null || echo '[]')
}" > "$cax_wisdom_input"
        fi
      fi
    fi
    if [[ -f "$cax_wisdom_input" ]]; then
      xml-wisdom-export "$cax_wisdom_input" "$cax_tmp_dir/wisdom.xml" 2>/dev/null || true
    fi

    # Registry section — reuse registry-export-xml on-demand generation logic
    source "$SCRIPT_DIR/exchange/registry-xml.sh"
    cax_registry_input="$DATA_DIR/colony-registry.json"
    if [[ ! -f "$cax_registry_input" ]]; then
      cax_rex_chambers_dir="$AETHER_ROOT/.aether/chambers"
      cax_rex_generated_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
      cax_rex_colonies="[]"
      if [[ -d "$cax_rex_chambers_dir" ]]; then
        cax_rex_colonies=$(
          for manifest in "$cax_rex_chambers_dir"/*/manifest.json; do
            [[ -f "$manifest" ]] || continue
            jq -c '{
              id: (.colony_id // .goal // "unknown"),
              name: (.goal // "Unnamed Colony"),
              created_at: (.created_at // "unknown"),
              sealed_at: (.sealed_at // null),
              status: (if .sealed_at then "sealed" else "active" end),
              chamber: input_filename
            }' "$manifest" 2>/dev/null || true
          done | jq -s '.' 2>/dev/null || echo '[]'
        )
      fi
      cax_registry_input="$cax_tmp_dir/registry-input.json"
      printf '%s\n' "{
  \"version\": \"1.0.0\",
  \"generated_at\": \"$cax_rex_generated_at\",
  \"colonies\": $cax_rex_colonies
}" > "$cax_registry_input"
    fi
    xml-registry-export "$cax_registry_input" "$cax_tmp_dir/registry.xml" 2>/dev/null || true

    # Step 3: Build combined XML
    cax_colony_id=$(jq -r '.goal // "unknown"' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null | tr '[:upper:]' '[:lower:]' | tr -cs '[:alnum:]' '-' | sed 's/^-//;s/-$//')
    [[ -z "$cax_colony_id" || "$cax_colony_id" == "unknown" ]] && cax_colony_id="unknown"
    cax_sealed_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    cax_pheromone_count=$(jq '.signals | length' "$cax_tmp_pheromones" 2>/dev/null || echo 0)

    {
      printf '<?xml version="1.0" encoding="UTF-8"?>\n'
      printf '<colony-archive\n'
      printf '    xmlns="http://aether.colony/schemas/archive/1.0"\n'
      printf '    colony_id="%s"\n' "$cax_colony_id"
      printf '    sealed_at="%s"\n' "$cax_sealed_at"
      printf '    version="1.0.0"\n'
      printf '    pheromone_count="%s">\n' "$cax_pheromone_count"

      # Append pheromone section (strip XML declaration)
      if [[ -f "$cax_tmp_dir/pheromones.xml" ]]; then
        sed '1{/^<?xml/d;}' "$cax_tmp_dir/pheromones.xml"
      fi

      # Append wisdom section (strip XML declaration)
      if [[ -f "$cax_tmp_dir/wisdom.xml" ]]; then
        sed '1{/^<?xml/d;}' "$cax_tmp_dir/wisdom.xml"
      fi

      # Append registry section (strip XML declaration)
      if [[ -f "$cax_tmp_dir/registry.xml" ]]; then
        sed '1{/^<?xml/d;}' "$cax_tmp_dir/registry.xml"
      fi

      printf '</colony-archive>\n'
    } > "$cax_output"

    # Step 4: Validate well-formedness
    if xmllint --noout "$cax_output" 2>/dev/null; then
      cax_valid=true
    else
      cax_valid=false
    fi

    # Step 5: Cleanup temp files
    rm -rf "$cax_tmp_dir" "$cax_tmp_pheromones"

    json_ok "{\"path\":\"$cax_output\",\"valid\":$cax_valid,\"colony_id\":\"$cax_colony_id\",\"pheromone_count\":$cax_pheromone_count}"
    ;;

  # ============================================================================
  # Session Continuity Commands
  # ============================================================================

  session-init)
    # Initialize a new session tracking file
    # Usage: session-init [session_id] [goal]
    session_id="${2:-$(date +%s)_$(openssl rand -hex 4 2>/dev/null || echo $$)}"
    goal="${3:-}"

    session_file="$DATA_DIR/session.json"
    baseline=$(git rev-parse HEAD 2>/dev/null || echo "")

    cat > "$session_file" << EOF
{
  "session_id": "$session_id",
  "started_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "last_command": null,
  "last_command_at": null,
  "colony_goal": "$goal",
  "current_phase": 0,
  "current_milestone": "First Mound",
  "suggested_next": "/ant:plan",
  "context_cleared": false,
  "baseline_commit": "$baseline",
  "resumed_at": null,
  "active_todos": [],
  "summary": "Session initialized"
}
EOF
    json_ok "{\"session_id\":\"$session_id\",\"goal\":\"$goal\",\"file\":\"$session_file\"}"
    ;;

  session-update)
    # Update session with latest activity
    # Usage: session-update <command> [suggested_next] [summary]
    cmd_run="${2:-}"
    suggested="${3:-}"
    summary="${4:-}"

    session_file="$DATA_DIR/session.json"

    if [[ ! -f "$session_file" ]]; then
      # Auto-initialize if doesn't exist
      bash "$0" session-init "auto_$(date +%s)" ""
    fi

    # Read current session
    current_session=$(cat "$session_file" 2>/dev/null || echo '{}')

    # Extract current values for preservation
    current_goal=$(echo "$current_session" | jq -r '.colony_goal // empty')
    current_phase=$(echo "$current_session" | jq -r '.current_phase // 0')
    current_milestone=$(echo "$current_session" | jq -r '.current_milestone // "First Mound"')

    # Get top 3 TODOs if TO-DOs.md exists
    todos="[]"
    if [[ -f "TO-DOs.md" ]]; then
      todos=$(grep "^### " TO-DOs.md 2>/dev/null | head -3 | sed 's/^### //' | jq -R . | jq -s .)
    fi

    # Get colony state if exists
    if [[ -f "$DATA_DIR/COLONY_STATE.json" ]]; then
      current_goal=$(jq -r '.goal // empty' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null || echo "$current_goal")
      current_phase=$(jq -r '.current_phase // 0' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null || echo "$current_phase")
      current_milestone=$(jq -r '.milestone // "First Mound"' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null || echo "$current_milestone")
    fi

    # Capture current git HEAD for drift detection
    baseline=$(git rev-parse HEAD 2>/dev/null || echo "")

    # Build updated session
    echo "$current_session" | jq --arg cmd "$cmd_run" \
      --arg ts "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
      --arg suggested "$suggested" \
      --arg summary "$summary" \
      --arg goal "$current_goal" \
      --argjson phase "$current_phase" \
      --arg milestone "$current_milestone" \
      --argjson todos "$todos" \
      --arg baseline "$baseline" \
      '.last_command = $cmd |
       .last_command_at = $ts |
       .suggested_next = $suggested |
       .summary = $summary |
       .colony_goal = $goal |
       .current_phase = $phase |
       .current_milestone = $milestone |
       .active_todos = $todos |
       .baseline_commit = $baseline' > "$session_file"

    json_ok "{\"updated\":true,\"command\":\"$cmd_run\"}"
    ;;

  session-read)
    # Read and return current session state
    session_file="$DATA_DIR/session.json"

    if [[ ! -f "$session_file" ]]; then
      json_ok "{\"exists\":false,\"session\":null}"
      exit 0
    fi

    session_data=$(cat "$session_file" 2>/dev/null || echo '{}')

    # Check if stale (> 24 hours)
    last_cmd_ts="" is_stale="" age_hours=""
    last_cmd_ts=$(echo "$session_data" | jq -r '.last_command_at // .started_at // empty')
    if [[ -n "$last_cmd_ts" ]]; then
      last_epoch=0 now_epoch=0
      last_epoch=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "$last_cmd_ts" +%s 2>/dev/null || echo 0)
      now_epoch=$(date +%s)
      age_hours=$(( (now_epoch - last_epoch) / 3600 ))
      [[ $age_hours -gt 24 ]] && is_stale=true || is_stale=false
    else
      is_stale="false"
      age_hours="unknown"
    fi

    json_ok "{\"exists\":true,\"is_stale\":$is_stale,\"age_hours\":$age_hours,\"session\":$session_data}"
    ;;

  session-is-stale)
    # Check if session is stale (returns JSON with is_stale boolean)
    session_file="$DATA_DIR/session.json"

    if [[ ! -f "$session_file" ]]; then
      json_ok '{"is_stale":true}'
      exit 0
    fi

    last_cmd_ts=$(jq -r '.last_command_at // .started_at // empty' "$session_file" 2>/dev/null)

    if [[ -z "$last_cmd_ts" ]]; then
      json_ok '{"is_stale":true}'
      exit 0
    fi

    last_epoch=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "$last_cmd_ts" +%s 2>/dev/null || echo 0)
    now_epoch=$(date +%s)
    age_hours=$(( (now_epoch - last_epoch) / 3600 ))

    if [[ $age_hours -gt 24 ]]; then
      json_ok '{"is_stale":true}'
    else
      json_ok '{"is_stale":false}'
    fi
    ;;

  session-clear)
    # Mark session as cleared (preserves file but marks context_cleared)
    preserve="${2:-false}"
    session_file="$DATA_DIR/session.json"

    if [[ -f "$session_file" ]]; then
      if [[ "$preserve" == "true" ]]; then
        # Just mark as cleared
        jq '.context_cleared = true' "$session_file" > "$session_file.tmp" && mv "$session_file.tmp" "$session_file"
        json_ok "{\"cleared\":true,\"preserved\":true}"
      else
        # Remove file entirely
        rm -f "$session_file"
        json_ok "{\"cleared\":true,\"preserved\":false}"
      fi
    else
      json_ok "{\"cleared\":false,\"reason\":\"no_session_exists\"}"
    fi
    ;;

  session-mark-resumed)
    # Mark session as resumed
    session_file="$DATA_DIR/session.json"

    if [[ -f "$session_file" ]]; then
      jq --arg ts "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
         '.resumed_at = $ts | .context_cleared = false' "$session_file" > "$session_file.tmp" && mv "$session_file.tmp" "$session_file"
      json_ok "{\"resumed\":true,\"timestamp\":\"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\"}"
    else
      json_err "$E_RESOURCE_NOT_FOUND" "No session to mark as resumed"
    fi
    ;;

  session-summary)
    # Get session summary (human-readable or JSON)
    session_file="$DATA_DIR/session.json"
    json_mode="false"

    # Parse --json flag (command name already shifted by main dispatch)
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --json)
          json_mode="true"
          shift
          ;;
        *)
          shift
          ;;
      esac
    done

    if [[ ! -f "$session_file" ]]; then
      if [[ "$json_mode" == "true" ]]; then
        json_ok '{"exists":false,"goal":null,"phase":0}'
      else
        echo "No active session found."
      fi
      exit 0
    fi

    goal=$(jq -r '.colony_goal // "No goal set"' "$session_file")
    phase=$(jq -r '.current_phase // 0' "$session_file")
    milestone=$(jq -r '.current_milestone // "First Mound"' "$session_file")
    last_cmd=$(jq -r '.last_command // "None"' "$session_file")
    last_at=$(jq -r '.last_command_at // "Unknown"' "$session_file")
    suggested=$(jq -r '.suggested_next // "None"' "$session_file")
    cleared=$(jq -r '.context_cleared // false' "$session_file")

    if [[ "$json_mode" == "true" ]]; then
      # Escape goal for JSON
      goal_escaped=$(echo "$goal" | jq -Rs . | tr -d '\n')
      milestone_escaped=$(echo "$milestone" | jq -Rs . | tr -d '\n')
      last_cmd_escaped=$(echo "$last_cmd" | jq -Rs . | tr -d '\n')
      last_at_escaped=$(echo "$last_at" | jq -Rs . | tr -d '\n')
      suggested_escaped=$(echo "$suggested" | jq -Rs . | tr -d '\n')
      json_ok "{\"exists\":true,\"goal\":$goal_escaped,\"phase\":$phase,\"milestone\":$milestone_escaped,\"last_command\":$last_cmd_escaped,\"last_active\":$last_at_escaped,\"suggested_next\":$suggested_escaped,\"context_cleared\":$cleared}"
    else
      echo "Session Summary"
      echo "=================="
      echo "Goal: $goal"
      [[ "$phase" != "0" ]] && echo "Phase: $phase"
      echo "Milestone: $milestone"
      echo "Last Command: $last_cmd"
      echo "Last Active: $last_at"
      [[ "$suggested" != "None" ]] && echo "Suggested Next: $suggested"
      [[ "$cleared" == "true" ]] && echo "Status: Context was cleared"
    fi
    ;;

  *)
    json_err "$E_VALIDATION_FAILED" "Unknown command: $cmd"
    ;;
esac
