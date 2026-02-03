#!/bin/bash
# Aether Colony Utility Layer
# Single entry point for deterministic colony operations
#
# Usage: bash .aether/aether-utils.sh <subcommand> [args...]
#
# All subcommands output JSON to stdout.
# Non-zero exit on error with JSON error message to stderr.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$SCRIPT_DIR/.." && pwd 2>/dev/null || echo "$SCRIPT_DIR")"
DATA_DIR="$SCRIPT_DIR/data"

# Initialize lock state before sourcing (file-lock.sh trap needs these)
LOCK_ACQUIRED=${LOCK_ACQUIRED:-false}
CURRENT_LOCK=${CURRENT_LOCK:-""}

# Source shared infrastructure
source "$SCRIPT_DIR/utils/file-lock.sh"
source "$SCRIPT_DIR/utils/atomic-write.sh"

# --- JSON output helpers ---
# Success: JSON to stdout, exit 0
json_ok() { printf '{"ok":true,"result":%s}\n' "$1"; }

# Error: JSON to stderr, exit 1
json_err() { printf '{"ok":false,"error":"%s"}\n' "$1" >&2; exit 1; }

# --- Subcommand dispatch ---
cmd="${1:-help}"
shift 2>/dev/null || true

case "$cmd" in
  help)
    cat <<'EOF'
{"ok":true,"commands":["help","version","pheromone-decay","pheromone-effective","pheromone-batch","pheromone-cleanup","pheromone-combine","validate-state","memory-token-count","memory-compress","memory-search","error-add","error-pattern-check","error-summary","error-dedup"],"description":"Aether Colony Utility Layer — deterministic ops for the ant colony"}
EOF
    ;;
  version)
    json_ok '"0.1.0"'
    ;;
  pheromone-decay)
    [[ $# -ge 3 ]] || json_err "Usage: pheromone-decay <strength> <elapsed_seconds> <half_life>"
    json_ok "$(jq -n --arg s "$1" --arg e "$2" --arg h "$3" \
      '{strength: (($s|tonumber) * ((-0.693147180559945 * ($e|tonumber) / ($h|tonumber)) | exp) | . * 1000000 | round / 1000000)}')"
    ;;
  pheromone-effective)
    [[ $# -ge 2 ]] || json_err "Usage: pheromone-effective <sensitivity> <strength>"
    json_ok "$(jq -n --arg sens "$1" --arg str "$2" \
      '{effective_signal: (($sens|tonumber) * ($str|tonumber) | . * 1000000 | round / 1000000)}')"
    ;;
  pheromone-batch)
    [[ -f "$DATA_DIR/pheromones.json" ]] || json_err "pheromones.json not found"
    now=$(date -u +%s)
    json_ok "$(jq --arg now "$now" '.signals | map(. + {
      current_strength: (
        if .half_life_seconds == null then .strength
        else .strength * ((-0.693147180559945 * (($now|tonumber) - (.created_at | sub("\\.[0-9]+Z$";"Z") | fromdate)) / .half_life_seconds) | exp)
        end | . * 1000 | round / 1000)
    })' "$DATA_DIR/pheromones.json")" || json_err "Failed to read pheromones.json"
    ;;
  pheromone-cleanup)
    [[ -f "$DATA_DIR/pheromones.json" ]] || json_err "pheromones.json not found"
    now=$(date -u +%s)
    before=$(jq '.signals | length' "$DATA_DIR/pheromones.json")
    result=$(jq --arg now "$now" '.signals |= map(select(
      .half_life_seconds == null or
      (.strength * ((-0.693147180559945 * (($now|tonumber) - (.created_at | sub("\\.[0-9]+Z$";"Z") | fromdate)) / .half_life_seconds) | exp)) >= 0.05
    ))' "$DATA_DIR/pheromones.json") || json_err "Failed to process pheromones.json"
    atomic_write "$DATA_DIR/pheromones.json" "$result"
    after=$(echo "$result" | jq '.signals | length')
    json_ok "{\"removed\":$((before - after)),\"remaining\":$after}"
    ;;
  pheromone-combine)
    [[ $# -ge 2 ]] || json_err "Usage: pheromone-combine <signal1_strength> <signal2_strength>"
    json_ok "$(jq -n --arg s1 "$1" --arg s2 "$2" '{
      net_effect: ((($s1|tonumber) - ($s2|tonumber)) | if . < 0 then 0 else . end | . * 1000 | round / 1000),
      dominant: (if ($s1|tonumber) >= ($s2|tonumber) then "signal1" else "signal2" end),
      ratio: (if ($s2|tonumber) == 0 then null else (($s1|tonumber) / ($s2|tonumber)) | . * 1000 | round / 1000 end)
    }')"
    ;;
  validate-state)
    case "${1:-}" in
      colony)
        [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "COLONY_STATE.json not found"
        json_ok "$(jq '
          def chk(f;t): if has(f) then (if (.[f]|type) as $a | t | any(. == $a) then "pass" else "fail: \(f) is \(.[f]|type), expected \(t|join("|"))" end) else "fail: missing \(f)" end;
          {file:"COLONY_STATE.json", checks:[
            chk("goal";["null","string"]),
            chk("state";["string"]),
            chk("current_phase";["number"]),
            chk("workers";["object"]),
            chk("spawn_outcomes";["object"])
          ]} | . + {pass: ([.checks[] | select(. == "pass")] | length) == (.checks | length)}
        ' "$DATA_DIR/COLONY_STATE.json")"
        ;;
      pheromones)
        [[ -f "$DATA_DIR/pheromones.json" ]] || json_err "pheromones.json not found"
        json_ok "$(jq '
          def arr(f): if has(f) and (.[f]|type) == "array" then "pass" else "fail: \(f) not array" end;
          def sig: ["id","type","content","strength","created_at"] as $req | [. as $s | $req[] | select($s[.] == null)] |
            if length == 0 then "pass" else "fail: signal missing \(join(","))" end;
          {file:"pheromones.json", checks:[
            arr("signals"),
            (.signals | if length == 0 then "pass" else [.[] | sig] | map(select(. != "pass")) |
              if length == 0 then "pass" else .[0] end end)
          ]} | . + {pass: ([.checks[] | select(. == "pass")] | length) == (.checks | length)}
        ' "$DATA_DIR/pheromones.json")"
        ;;
      errors)
        [[ -f "$DATA_DIR/errors.json" ]] || json_err "errors.json not found"
        json_ok "$(jq '
          def arr(f): if has(f) and (.[f]|type) == "array" then "pass" else "fail: \(f) not array" end;
          def erchk: ["id","category","severity","description","timestamp"] as $req | [. as $e | $req[] | select($e[.] == null)] |
            if length == 0 then "pass" else "fail: error missing \(join(","))" end;
          {file:"errors.json", checks:[
            arr("errors"), arr("flagged_patterns"),
            (.errors | if length == 0 then "pass" else [.[] | erchk] | map(select(. != "pass")) |
              if length == 0 then "pass" else .[0] end end)
          ]} | . + {pass: ([.checks[] | select(. == "pass")] | length) == (.checks | length)}
        ' "$DATA_DIR/errors.json")"
        ;;
      memory)
        [[ -f "$DATA_DIR/memory.json" ]] || json_err "memory.json not found"
        json_ok "$(jq '
          def arr(f): if has(f) and (.[f]|type) == "array" then "pass" else "fail: \(f) not array" end;
          {file:"memory.json", checks:[arr("phase_learnings"), arr("decisions"), arr("patterns")]}
          | . + {pass: ([.checks[] | select(. == "pass")] | length) == (.checks | length)}
        ' "$DATA_DIR/memory.json")"
        ;;
      events)
        [[ -f "$DATA_DIR/events.json" ]] || json_err "events.json not found"
        json_ok "$(jq '
          def arr(f): if has(f) and (.[f]|type) == "array" then "pass" else "fail: \(f) not array" end;
          def evchk: ["id","type","source","content","timestamp"] as $req | [. as $e | $req[] | select($e[.] == null)] |
            if length == 0 then "pass" else "fail: event missing \(join(","))" end;
          {file:"events.json", checks:[
            arr("events"),
            (.events | if length == 0 then "pass" else [.[] | evchk] | map(select(. != "pass")) |
              if length == 0 then "pass" else .[0] end end)
          ]} | . + {pass: ([.checks[] | select(. == "pass")] | length) == (.checks | length)}
        ' "$DATA_DIR/events.json")"
        ;;
      all)
        results=()
        for target in colony pheromones errors memory events; do
          results+=("$(bash "$SCRIPT_DIR/aether-utils.sh" validate-state "$target" 2>/dev/null || echo '{"ok":false}')")
        done
        combined=$(printf '%s\n' "${results[@]}" | jq -s '[.[] | .result // {file:"unknown",pass:false}]')
        all_pass=$(echo "$combined" | jq 'all(.pass)')
        json_ok "{\"pass\":$all_pass,\"files\":$combined}"
        ;;
      *)
        json_err "Usage: validate-state colony|pheromones|errors|memory|events|all"
        ;;
    esac
    ;;
  memory-token-count)
    [[ -f "$DATA_DIR/memory.json" ]] || json_err "memory.json not found"
    json_ok "$(jq '{tokens: ([.. | strings] | join(" ") | split(" ") | length | . * 1.3 | floor)}' "$DATA_DIR/memory.json")"
    ;;
  memory-compress)
    [[ -f "$DATA_DIR/memory.json" ]] || json_err "memory.json not found"
    threshold="${1:-10000}"
    result=$(jq --arg th "$threshold" '
      .phase_learnings |= (if length > 20 then .[-20:] else . end) |
      .decisions |= (if length > 30 then .[-30:] else . end) |
      . as $trimmed |
      ([.. | strings] | join(" ") | split(" ") | length | . * 1.3 | floor) as $tokens |
      if $tokens > ($th|tonumber) then
        .phase_learnings |= (if length > 10 then .[-10:] else . end) |
        .decisions |= (if length > 15 then .[-15:] else . end)
      else . end
    ' "$DATA_DIR/memory.json") || json_err "Failed to process memory.json"
    atomic_write "$DATA_DIR/memory.json" "$result"
    tokens=$(echo "$result" | jq '[.. | strings] | join(" ") | split(" ") | length | . * 1.3 | floor')
    json_ok "{\"compressed\":true,\"tokens\":$tokens}"
    ;;
  memory-search)
    [[ $# -ge 1 ]] || json_err "Usage: memory-search <keyword>"
    [[ -f "$DATA_DIR/memory.json" ]] || json_err "memory.json not found"
    json_ok "$(jq --arg kw "$1" '
      ($kw | ascii_downcase) as $k |
      {
        phase_learnings: [.phase_learnings[]? | select(tostring | ascii_downcase | contains($k))],
        decisions: [.decisions[]? | select(tostring | ascii_downcase | contains($k))],
        patterns: [.patterns[]? | select(tostring | ascii_downcase | contains($k))]
      }
    ' "$DATA_DIR/memory.json")"
    ;;
  error-add)
    [[ $# -ge 3 ]] || json_err "Usage: error-add <category> <severity> <description>"
    [[ -f "$DATA_DIR/errors.json" ]] || json_err "errors.json not found"
    id="err_$(date -u +%s)_$(head -c 2 /dev/urandom | od -An -tx1 | tr -d ' ')"
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    updated=$(jq --arg id "$id" --arg cat "$1" --arg sev "$2" --arg desc "$3" --arg ts "$ts" '
      .errors += [{id:$id, category:$cat, severity:$sev, description:$desc, root_cause:null, phase:null, task_id:null, timestamp:$ts}] |
      if (.errors|length) > 50 then .errors = .errors[-50:] else . end
    ' "$DATA_DIR/errors.json") || json_err "Failed to update errors.json"
    atomic_write "$DATA_DIR/errors.json" "$updated"
    json_ok "\"$id\""
    ;;
  error-pattern-check)
    [[ -f "$DATA_DIR/errors.json" ]] || json_err "errors.json not found"
    json_ok "$(jq '
      .errors | group_by(.category) | map(select(length >= 3) |
        {category: .[0].category, count: length,
         first_seen: (sort_by(.timestamp) | first.timestamp),
         last_seen: (sort_by(.timestamp) | last.timestamp)})
    ' "$DATA_DIR/errors.json")"
    ;;
  error-summary)
    [[ -f "$DATA_DIR/errors.json" ]] || json_err "errors.json not found"
    json_ok "$(jq '{
      total: (.errors | length),
      by_category: (.errors | group_by(.category) | map({key: .[0].category, value: length}) | from_entries),
      by_severity: (.errors | group_by(.severity) | map({key: .[0].severity, value: length}) | from_entries)
    }' "$DATA_DIR/errors.json")"
    ;;
  error-dedup)
    [[ -f "$DATA_DIR/errors.json" ]] || json_err "errors.json not found"
    before=$(jq '.errors | length' "$DATA_DIR/errors.json")
    result=$(jq '
      .errors |= (group_by(.category + "|" + .description) | map(
        sort_by(.timestamp) | . as $grp |
        [$grp[0]] + [$grp[range(1;length)] | select(
          (.timestamp | sub("\\.[0-9]+Z$";"Z") | fromdate) -
          ($grp[0].timestamp | sub("\\.[0-9]+Z$";"Z") | fromdate) > 60
        )]
      ) | flatten)
    ' "$DATA_DIR/errors.json") || json_err "Failed to process errors.json"
    atomic_write "$DATA_DIR/errors.json" "$result"
    after=$(echo "$result" | jq '.errors | length')
    json_ok "{\"removed\":$((before - after)),\"remaining\":$after}"
    ;;
  *)
    json_err "Unknown command: $cmd"
    ;;
esac
