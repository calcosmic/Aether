#!/usr/bin/env bash
# Pheromone utility functions -- extracted from aether-utils.sh
# Provides: _pheromone_export_eternal, _pheromone_write, _pheromone_count, _pheromone_display,
#           _pheromone_read, _pheromone_prime, _colony_prime, _pheromone_expire,
#           _eternal_init, _eternal_store, _pheromone_export_xml, _pheromone_import_xml,
#           _pheromone_validate_xml
# Note: colony-prime is the most complex function (~706 lines). Moved verbatim.
#       Calls hive-read via subprocess (safe). eternal-init and eternal-store are
#       tightly coupled to pheromone-expire.
#       Uses SCRIPT_DIR, AETHER_ROOT, DATA_DIR, HOME from main file preamble.


# ============================================================================
# _pheromone_export_eternal
# Export pheromones to eternal XML format (distinct from xml-utils.sh pheromone-export function)
# ============================================================================
_pheromone_export_eternal() {
_deprecation_warning "pheromone-export-eternal"
# Export pheromones to eternal XML format (distinct from xml-utils.sh pheromone-export function)
# Usage: pheromone-export-eternal [input_json] [output_xml]
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
  json_err "$E_DEPENDENCY_MISSING" "xml-utils.sh not available. Try: run aether update to restore utility scripts."
fi
}

# ============================================================================
# _pheromone_write
# Write a pheromone signal to pheromones.json
# ============================================================================
_pheromone_write() {
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

# Sanitize and bound input content to reduce injection risk in prompt contexts.

# Check for XML tag injection BEFORE escaping angle brackets.
# Content is injected into worker prompts via colony-prime, so raw XML
# structural tags could break prompt boundaries.
if echo "$pw_content" | grep -Eiq '<[[:space:]]*/?(system|prompt|instructions|system-reminder|assistant|user|human)'; then
  json_err "$E_VALIDATION_FAILED" "Pheromone content rejected: XML tag injection pattern detected"
fi

pw_content="${pw_content//</&lt;}"
pw_content="${pw_content//>/&gt;}"
pw_content="${pw_content:0:500}"
if echo "$pw_content" | grep -Eiq '(\$\(|`|(^|[[:space:]])curl([[:space:]]|$)|(^|[[:space:]])wget([[:space:]]|$)|(^|[[:space:]])rm([[:space:]]|$))'; then
  json_err "$E_VALIDATION_FAILED" "Pheromone content rejected: potential injection pattern"
fi

# Check for prompt injection text patterns. These phrases attempt to
# override LLM instructions when the content is injected into prompts.
if echo "$pw_content" | grep -Eiq '(ignore\s+(all\s+)?(previous\s+|prior\s+|above\s+)?instructions|disregard\s+(above|previous|all)|you are now |new instructions:|system prompt)'; then
  json_err "$E_VALIDATION_FAILED" "Pheromone content rejected: prompt injection pattern detected"
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

if ! [[ "$pw_strength" =~ ^(0(\.[0-9]+)?|1(\.0+)?)$ ]]; then
  json_err "$E_VALIDATION_FAILED" "Strength must be a number between 0.0 and 1.0" "{\"provided\":\"$pw_strength\"}"
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
pw_rand=$(( RANDOM % 10000 ))
pw_type_lower=$(echo "$pw_type" | tr '[:upper:]' '[:lower:]')
pw_id="sig_${pw_type_lower}_${pw_epoch}_${pw_rand}"
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
    # SUPPRESS:OK -- cross-platform: macOS date-from-epoch syntax
    # SUPPRESS:OK -- cross-platform: macOS vs Linux date/stat flags
    pw_expires=$(date -u -r "$pw_expires_epoch" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || \
                 date -u -d "@$pw_expires_epoch" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || \
                 echo "phase_end")
  else
    pw_expires="phase_end"
  fi
fi

pw_file="$COLONY_DATA_DIR/pheromones.json"

pw_lock_held=false
if type acquire_lock &>/dev/null; then
  acquire_lock "$pw_file" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on pheromones.json"
  pw_lock_held=true
  # Trap ensures lock release on unexpected exit (json_err calls exit 1)
  trap 'release_lock 2>/dev/null || true' EXIT  # SUPPRESS:OK -- cleanup: lock may not be held
fi

# Initialize pheromones.json if missing
if [[ ! -f "$pw_file" ]]; then
  pw_colony_id="aether-dev"
  # MIGRATE: direct COLONY_STATE.json access -- use _state_read_field instead
  if [[ -f "$DATA_DIR/COLONY_STATE.json" ]]; then
    # SUPPRESS:OK -- read-default: query may return empty
    pw_colony_id=$(jq -r '.session_id // "aether-dev"' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null || echo "aether-dev")
  fi
  pw_init_content=$(printf '{\n  "version": "1.0.0",\n  "colony_id": "%s",\n  "generated_at": "%s",\n  "signals": []\n}\n' \
    "$pw_colony_id" "$pw_created")
  atomic_write "$pw_file" "$pw_init_content" || {
    _aether_log_error "Could not initialize pheromones file"
    json_err "$E_UNKNOWN" "Failed to create pheromones file"
  }
fi

# Compute SHA-256 content hash for deduplication
pw_hash=$(echo -n "$pw_content" | shasum -a 256 | cut -d' ' -f1)

# Check for existing active signal with same type and content_hash
# SUPPRESS:OK -- read-default: file may not exist yet
pw_existing_count=$(jq \
  --arg type "$pw_type" \
  --arg hash "$pw_hash" \
  '[.signals[] | select(.active == true and .type == $type and .content_hash == $hash)] | length' \
  "$pw_file" 2>/dev/null || echo "0")  # SUPPRESS:OK -- read-default: returns fallback on failure

pw_action="created"

if [[ "$pw_existing_count" -gt 0 ]]; then
  # Reinforce existing signal: update strength to max, reset created_at, increment reinforcement_count
  pw_action="reinforced"

  # Get the reinforced signal's ID for output (before modification)
  # SUPPRESS:OK -- read-default: file may not exist yet
  pw_id=$(jq -r \
    --arg type "$pw_type" \
    --arg hash "$pw_hash" \
    '[.signals[] | select(.active == true and .type == $type and .content_hash == $hash)][0].id' \
    "$pw_file" 2>/dev/null || echo "$pw_id")  # SUPPRESS:OK -- read-default: returns fallback on failure

  pw_updated=$(jq \
    --arg type "$pw_type" \
    --arg hash "$pw_hash" \
    --argjson new_strength "$pw_strength" \
    --arg new_created "$pw_created" \
    '
    .signals = [.signals[] |
      if (.active == true and .type == $type and .content_hash == $hash) then
        .strength = ([.strength, $new_strength] | max) |
        .created_at = $new_created |
        .reinforcement_count = ((.reinforcement_count // 0) + 1)
      else
        .
      end
    ]
    ' "$pw_file" 2>/dev/null)  # SUPPRESS:OK -- read-default: file may not exist yet

  if [[ -z "$pw_updated" ]]; then
    [[ "$pw_lock_held" == "true" ]] && release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
    json_err "${E_JSON_INVALID:-E_JSON_INVALID}" "Failed to reinforce signal in pheromones.json — jq parse error"
  fi
else
  # Build new signal object with content_hash and append
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
    --arg content_hash "$pw_hash" \
    --argjson reinforcement_count 0 \
    '{id: $id, type: $type, priority: $priority, source: $source, created_at: $created_at, expires_at: $expires_at, active: $active, strength: ($strength | tonumber), reason: $reason, content: {text: $content}, content_hash: $content_hash, reinforcement_count: $reinforcement_count}')

  pw_updated=$(jq --argjson sig "$pw_signal" '.signals += [$sig]' "$pw_file") || {
    _aether_log_error "Could not append signal to pheromones.json"
  }
  if [[ -z "$pw_updated" || "$pw_updated" == "null" ]]; then
    [[ "$pw_lock_held" == "true" ]] && release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
    json_err "${E_JSON_INVALID:-E_JSON_INVALID}" "Failed to update pheromones.json — jq parse error"
  fi
fi

atomic_write "$pw_file" "$pw_updated" || {
  [[ "$pw_lock_held" == "true" ]] && release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
  json_err "$E_JSON_INVALID" "Failed to write pheromones.json"
}
[[ "$pw_lock_held" == "true" ]] && { release_lock 2>/dev/null || true; trap - EXIT; }  # SUPPRESS:OK -- cleanup: lock may not be held

# Backward compatibility: also write to constraints.json
pw_cfile="$COLONY_DATA_DIR/constraints.json"
if [[ "$pw_type" == "FOCUS" ]]; then
  if [[ ! -f "$pw_cfile" ]]; then
    atomic_write "$pw_cfile" '{"version":"1.0","focus":[],"constraints":[]}' || _aether_log_error "Could not initialize constraints file"
  fi
  pw_cfile_updated=$(jq --arg txt "$pw_content" '
    .focus += [$txt] |
    if (.focus | length) > 5 then .focus = .focus[-5:] else . end
  ' "$pw_cfile" 2>/dev/null)  # SUPPRESS:OK -- read-default: file may not exist yet
  if [[ -n "$pw_cfile_updated" ]]; then
    atomic_write "$pw_cfile" "$pw_cfile_updated" || _aether_log_error "Could not save focus constraint"
  fi
elif [[ "$pw_type" == "REDIRECT" ]]; then
  if [[ ! -f "$pw_cfile" ]]; then
    atomic_write "$pw_cfile" '{"version":"1.0","focus":[],"constraints":[]}' || _aether_log_error "Could not initialize constraints file"
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
  ' "$pw_cfile" 2>/dev/null)  # SUPPRESS:OK -- read-default: file may not exist yet
  if [[ -n "$pw_cfile_updated" ]]; then
    atomic_write "$pw_cfile" "$pw_cfile_updated" || _aether_log_error "Could not save redirect constraint"
  fi
fi

# Get active signal count
# SUPPRESS:OK -- read-default: query may return empty
pw_active_count=$(jq '[.signals[] | select(.active == true)] | length' "$pw_file" 2>/dev/null || echo "0")

json_ok "$(jq -n --arg signal_id "$pw_id" --arg type "$pw_type" --arg action "$pw_action" --argjson active_count "$pw_active_count" '{signal_id: $signal_id, type: $type, action: $action, active_count: $active_count}')"
}

# ============================================================================
# _pheromone_count
# Count active pheromone signals by type
# ============================================================================
_pheromone_count() {
# Count active pheromone signals by type
# Usage: pheromone-count
# Returns: JSON with per-type counts

pc_file="$COLONY_DATA_DIR/pheromones.json"

if [[ ! -f "$pc_file" ]]; then
  json_ok '{"focus":0,"redirect":0,"feedback":0,"total":0}'
else
  pc_result=$(jq -c '{
    focus:    ([.signals[] | select(.active == true and .type == "FOCUS")]    | length),
    redirect: ([.signals[] | select(.active == true and .type == "REDIRECT")] | length),
    feedback: ([.signals[] | select(.active == true and .type == "FEEDBACK")] | length),
    total:    ([.signals[] | select(.active == true)]                          | length)
  }' "$pc_file" 2>/dev/null)  # SUPPRESS:OK -- read-default: operation may fail
  if [[ -z "$pc_result" ]]; then
    json_ok '{"focus":0,"redirect":0,"feedback":0,"total":0}'
  else
    json_ok "$pc_result"
  fi
fi
}

# ============================================================================
# _pheromone_display
# Display active pheromones in formatted table
# ============================================================================
_pheromone_display() {
# Display active pheromones in formatted table
# Usage: pheromone-display [type]
#   type: Optional filter (focus/redirect/feedback) or 'all' (default: all)
# Returns: Formatted table string (human-readable)

pd_file="$COLONY_DATA_DIR/pheromones.json"
pd_type="${1:-all}"
pd_now_iso=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

if [[ ! -f "$pd_file" ]]; then
  echo "No pheromones active. Colony has no signals."
  echo ""
  echo "Inject signals with:"
  echo "  /ant:focus \"area\"    - Guide attention"
  echo "  /ant:redirect \"avoid\" - Set hard constraint"
  echo "  /ant:feedback \"note\"  - Provide guidance"
  exit 0
fi

# Get signals with decay calculation (same as pheromone-read)
pd_signals=$(jq -c \
  --arg now_iso "$pd_now_iso" \
  --arg type_filter "$pd_type" \
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

  (to_epoch($now_iso)) as $now |
  .signals | map(
    (to_epoch(.created_at)) as $created_epoch |
    (if $created_epoch != null then ($now - $created_epoch) / 86400 else 0 end) as $elapsed_days |
    (decay_days(.type)) as $dd |
    ((.strength // 0.8) * (1 - ($elapsed_days / $dd))) as $eff_raw |
    (if $eff_raw < 0 then 0 else $eff_raw end) as $eff |
    {
      id: .id,
      type: .type,
      content: .content,
      strength: (.strength // 0.8),
      effective_strength: $eff,
      elapsed_days: $elapsed_days,
      remaining_days: ($dd - $elapsed_days),
      created_at: .created_at,
      active: (.active != false and $eff >= 0.1)
    }
  )
  | map(select(.active == true))
  | map(select(if $type_filter == "all" or $type_filter == "" then true else (.type | ascii_downcase) == ($type_filter | ascii_downcase) end))
  | sort_by(-.effective_strength)
  ' "$pd_file" 2>/dev/null)  # SUPPRESS:OK -- read-default: file may not exist yet

if [[ -z "$pd_signals" || "$pd_signals" == "[]" ]]; then
  echo "No active pheromones found."
  if [[ "$pd_type" != "all" ]]; then
    echo "Filter: $pd_type"
  fi
  exit 0
fi

# Count by type
pd_focus=$(echo "$pd_signals" | jq '[.[] | select(.type == "FOCUS")] | length')
pd_redirect=$(echo "$pd_signals" | jq '[.[] | select(.type == "REDIRECT")] | length')
pd_feedback=$(echo "$pd_signals" | jq '[.[] | select(.type == "FEEDBACK")] | length')
pd_total=$(echo "$pd_signals" | jq 'length')

# Display header
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "   A C T I V E   P H E R O M O N E S"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Display FOCUS signals
if [[ "$pd_focus" -gt 0 && ("$pd_type" == "all" || "$pd_type" == "focus") ]]; then
  echo "🎯 FOCUS (Pay attention here)"
  echo "$pd_signals" | jq -r '.[] | select(.type == "FOCUS") | "   \n   [\(.effective_strength * 100 | floor)%] \"\(.content.text // .content // "no content")\"\n      └── \(.elapsed_days | floor)d ago, \(.remaining_days | floor)d remaining"' | head -20
  echo ""
fi

# Display REDIRECT signals
if [[ "$pd_redirect" -gt 0 && ("$pd_type" == "all" || "$pd_type" == "redirect") ]]; then
  echo "🚫 REDIRECT (Hard constraints - DO NOT do this)"
  echo "$pd_signals" | jq -r '.[] | select(.type == "REDIRECT") | "   \n   [\(.effective_strength * 100 | floor)%] \"\(.content.text // .content // "no content")\"\n      └── \(.elapsed_days | floor)d ago, \(.remaining_days | floor)d remaining"' | head -20
  echo ""
fi

# Display FEEDBACK signals
if [[ "$pd_feedback" -gt 0 && ("$pd_type" == "all" || "$pd_type" == "feedback") ]]; then
  echo "💬 FEEDBACK (Guidance to consider)"
  echo "$pd_signals" | jq -r '.[] | select(.type == "FEEDBACK") | "   \n   [\(.effective_strength * 100 | floor)%] \"\(.content.text // .content // "no content")\"\n      └── \(.elapsed_days | floor)d ago, \(.remaining_days | floor)d remaining"' | head -20
  echo ""
fi

# Display footer
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "$pd_total signal(s) active | Decay: FOCUS 30d, REDIRECT 60d, FEEDBACK 90d"
}

# ============================================================================
# _pheromone_read
# Read pheromones from colony data with decay calculation
# ============================================================================
_pheromone_read() {
# Read pheromones from colony data with decay calculation
# Usage: pheromone-read [type]
#   type: Filter by pheromone type (focus, redirect, feedback) or 'all' (default: all)
# Returns: JSON object with pheromones array including effective_strength

pher_type="${1:-all}"
pher_file="$COLONY_DATA_DIR/pheromones.json"

# Check if file exists
if [[ ! -f "$pher_file" ]]; then
  json_err "$E_FILE_NOT_FOUND" "Pheromones file not found. Run /ant:colonize first to initialize the colony."
fi

# Get current time as ISO for consistent epoch conversion
pher_now_iso=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Apply decay and expiry at read time
# Decay rates: FOCUS=30d, REDIRECT=60d, FEEDBACK/PATTERN=90d
# effective_strength = original_strength * (1 - elapsed_days / decay_days)
# If effective_strength < 0.1, mark inactive
# Also check expires_at: if not "phase_end" and past expiry, mark inactive
pher_type_upper=$(echo "$pher_type" | tr '[:lower:]' '[:upper:]')

pher_result=$(jq -c \
  --arg now_iso "$pher_now_iso" \
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

  (to_epoch($now_iso)) as $now |
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
      active: (if $deactivate then false elif .active == false then false else true end)
    }
  ) |
  map(select(.active == true)) |
  if $type_filter != "ALL" then
    map(select(.type == $type_filter))
  else
    .
  end
  ' "$pher_file" 2>/dev/null)  # SUPPRESS:OK -- read-default: file may not exist yet

if [[ -z "$pher_result" || "$pher_result" == "null" ]]; then
  json_ok '{"version":"1.0.0","signals":[]}'
else
  pher_version=$(jq -r '.version // "1.0.0"' "$pher_file" 2>/dev/null || echo "1.0.0")  # SUPPRESS:OK -- read-default: file may not exist yet
  pher_colony=$(jq -r '.colony_id // "unknown"' "$pher_file" 2>/dev/null || echo "unknown")  # SUPPRESS:OK -- read-default: file may not exist yet
  json_ok "$(jq -n --arg version "$pher_version" --arg colony_id "$pher_colony" --argjson signals "$pher_result" '{version: $version, colony_id: $colony_id, signals: $signals}')"
fi
}

# ============================================================================
# _pheromone_prime
# Combine active pheromone signals and learned instincts into a prompt-ready block
# ============================================================================
_pheromone_prime() {
# Combine active pheromone signals and learned instincts into a prompt-ready block
# Usage: pheromone-prime [--compact] [--max-signals N] [--max-instincts N]
# Returns: JSON with signal_count, instinct_count, prompt_section, log_line

pp_compact=false
pp_max_signals=0
pp_max_instincts=5
while [[ $# -gt 0 ]]; do
  case "$1" in
    --compact) pp_compact=true ;;
    --max-signals) shift; pp_max_signals="${1:-8}" ;;
    --max-instincts) shift; pp_max_instincts="${1:-3}" ;;
  esac
  shift
done
[[ "$pp_max_signals" =~ ^[0-9]+$ ]] || pp_max_signals=8
[[ "$pp_max_instincts" =~ ^[0-9]+$ ]] || pp_max_instincts=3
[[ "$pp_max_signals" -lt 1 ]] && pp_max_signals=8
[[ "$pp_max_instincts" -lt 1 ]] && pp_max_instincts=3

pp_pher_file="$COLONY_DATA_DIR/pheromones.json"
# MIGRATE: direct COLONY_STATE.json access -- use _state_read_field instead
pp_state_file="$DATA_DIR/COLONY_STATE.json"
pp_now_iso=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Read active signals (same decay logic as pheromone-read)
pp_signals="[]"
if [[ -f "$pp_pher_file" ]]; then
  pp_signals=$(jq -c \
    --arg now_iso "$pp_now_iso" \
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

    (to_epoch($now_iso)) as $now |
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
        active: (if $deactivate then false elif .active == false then false else true end)
      }
    ) |
    map(select(.active == true))
    ' "$pp_pher_file" 2>/dev/null || echo "[]")  # SUPPRESS:OK -- read-default: file may not exist yet
fi

if [[ -z "$pp_signals" || "$pp_signals" == "null" ]]; then
  pp_signals="[]"
fi

if [[ "$pp_compact" == "true" ]]; then
  pp_signals=$(echo "$pp_signals" | jq -c --argjson max "$pp_max_signals" '
    map(. + {priority: (if .type == "REDIRECT" then 1 elif .type == "FOCUS" then 2 elif .type == "FEEDBACK" then 3 elif .type == "POSITION" then 4 else 5 end)})
    | sort_by(.priority, -(.effective_strength // 0))
    | .[:$max]
    | map(del(.priority))
  ' 2>/dev/null || echo "[]")  # SUPPRESS:OK -- read-default: returns fallback on failure
fi

# Read instincts (confidence >= 0.5, not disproven)
pp_instincts="[]"
if [[ -f "$pp_state_file" ]]; then
  pp_instincts=$(jq -c \
    --argjson max "$pp_max_instincts" \
    '
    (.memory.instincts // [])
    | map(select(
        (.confidence // 0) >= 0.5
        and (.status // "hypothesis") != "disproven"
      ))
    | sort_by(-.confidence)
    | .[:$max]
    ' "$pp_state_file" 2>/dev/null || echo "[]")  # SUPPRESS:OK -- read-default: file may not exist yet
fi

if [[ -z "$pp_instincts" || "$pp_instincts" == "null" ]]; then
  pp_instincts="[]"
fi

pp_signal_count=$(echo "$pp_signals" | jq 'length' 2>/dev/null || echo "0")  # SUPPRESS:OK -- read-default: file may not exist yet
pp_instinct_count=$(echo "$pp_instincts" | jq 'length' 2>/dev/null || echo "0")  # SUPPRESS:OK -- read-default: file may not exist yet

# Build prompt section
if [[ "$pp_signal_count" -eq 0 && "$pp_instinct_count" -eq 0 ]]; then
  pp_section=""
  pp_log_line="Primed: 0 signals, 0 instincts"
else
  if [[ "$pp_compact" == "true" ]]; then
    pp_section="--- COMPACT SIGNALS ---"$'\n'
  else
    pp_section="--- ACTIVE SIGNALS (Colony Guidance) ---"$'\n'
  fi

  # FOCUS signals
  # SUPPRESS:OK -- read-default: file may not exist yet
  pp_focus=$(echo "$pp_signals" | jq -r 'map(select(.type == "FOCUS")) | .[] | "[" + ((.effective_strength * 10 | round) / 10 | tostring) + "] " + (.content.text // (if (.content | type) == "string" then .content else "" end))' 2>/dev/null || echo "")
  if [[ -n "$pp_focus" ]]; then
    pp_section+=$'\n'"FOCUS (Pay attention to):"$'\n'"$pp_focus"$'\n'
  fi

  # REDIRECT signals
  # SUPPRESS:OK -- read-default: file may not exist yet
  pp_redirect=$(echo "$pp_signals" | jq -r 'map(select(.type == "REDIRECT")) | .[] | "[" + ((.effective_strength * 10 | round) / 10 | tostring) + "] " + (.content.text // (if (.content | type) == "string" then .content else "" end))' 2>/dev/null || echo "")
  if [[ -n "$pp_redirect" ]]; then
    pp_section+=$'\n'"REDIRECT (HARD CONSTRAINTS - MUST follow):"$'\n'"$pp_redirect"$'\n'
  fi

  # FEEDBACK signals
  # SUPPRESS:OK -- read-default: file may not exist yet
  pp_feedback=$(echo "$pp_signals" | jq -r 'map(select(.type == "FEEDBACK")) | .[] | "[" + ((.effective_strength * 10 | round) / 10 | tostring) + "] " + (.content.text // (if (.content | type) == "string" then .content else "" end))' 2>/dev/null || echo "")
  if [[ -n "$pp_feedback" ]]; then
    pp_section+=$'\n'"FEEDBACK (Flexible guidance):"$'\n'"$pp_feedback"$'\n'
  fi

  # POSITION signals
  # SUPPRESS:OK -- read-default: file may not exist yet
  pp_position=$(echo "$pp_signals" | jq -r 'map(select(.type == "POSITION")) | .[] | "[" + ((.effective_strength * 10 | round) / 10 | tostring) + "] " + (.content.text // (if (.content | type) == "string" then .content else "" end))' 2>/dev/null || echo "")
  if [[ -n "$pp_position" ]]; then
    pp_section+=$'\n'"POSITION (Where work last progressed):"$'\n'"$pp_position"$'\n'
  fi

  # Instincts section (domain-grouped)
  if [[ "$pp_instinct_count" -gt 0 ]]; then
    if [[ "$pp_compact" == "true" ]]; then
      pp_section+=$'\n'"--- INSTINCTS (Learned Behaviors) ---"$'\n'
    else
      pp_section+=$'\n'"--- INSTINCTS (Learned Behaviors) ---"$'\n'
      pp_section+="Weight by confidence - higher = stronger guidance:"$'\n'
    fi

    # Group instincts by domain per user decision
    pp_instinct_lines=$(echo "$pp_instincts" | jq -r '
      group_by(.domain // "general")
      | map({
          domain: (.[0].domain // "general"),
          items: [.[] | "  [" + ((.confidence * 10 | round) / 10 | tostring) + "] When " + (.trigger | if test("^[Ww]hen ") then sub("^[Ww]hen "; "") else . end) + " -> " + .action]
        })
      | sort_by(.domain)
      | .[]
      | "\n" + (.domain | ascii_upcase | .[0:1]) + (.domain | .[1:]) + ":" + "\n" + (.items | join("\n"))
    ' 2>/dev/null || echo "")  # SUPPRESS:OK -- read-default: returns fallback on failure

    if [[ -n "$pp_instinct_lines" ]]; then
      pp_section+="$pp_instinct_lines"$'\n'
    fi
  fi

  pp_section+=$'\n'"--- END COLONY CONTEXT ---"

  pp_log_line="Primed: ${pp_signal_count} signals, ${pp_instinct_count} instincts"
fi

# Escape section for JSON embedding (use printf to avoid appending extra newline)
pp_section_json=$(printf '%s' "$pp_section" | jq -Rs '.' 2>/dev/null || echo '""')  # SUPPRESS:OK -- read-default: returns fallback if missing
# SUPPRESS:OK -- read-default: text escaping returns fallback on empty input
pp_log_json=$(printf '%s' "$pp_log_line" | jq -Rs '.' 2>/dev/null || echo '"Primed: 0 signals, 0 instincts"')

json_ok "$(jq -n --argjson signal_count "$pp_signal_count" --argjson instinct_count "$pp_instinct_count" --argjson prompt_section "$pp_section_json" --argjson log_line "$pp_log_json" '{signal_count: $signal_count, instinct_count: $instinct_count, prompt_section: $prompt_section, log_line: $log_line}')"
}

# ============================================================================
# _budget_enforce
# Shared budget enforcement for colony-prime and pr-context.
# Trims sections in priority order when assembled prompt exceeds character budget.
# Usage: _budget_enforce "<prefix>"
#   prefix: variable prefix ("cp_" for colony-prime, "pc_" for pr-context)
# Reads/writes via indirect access:
#   {prefix}max_chars, {prefix}budget_len, {prefix}final_prompt,
#   {prefix}sec_rolling, {prefix}sec_learnings, {prefix}sec_decisions,
#   {prefix}sec_hive, {prefix}sec_capsule, {prefix}sec_user_prefs,
#   {prefix}sec_queen_global, {prefix}sec_queen_local, {prefix}sec_signals,
#   {prefix}sec_blockers, {prefix}budget_trimmed_list
# Trim order: rolling > learnings > decisions > hive > capsule > user_prefs >
#   queen_global > queen_local > signals (preserves REDIRECTs). NEVER trims blockers.
# ============================================================================
_budget_enforce() {
  local _be_prefix="${1:-cp_}"

  # Assemble final_prompt from sections
  eval "local _be_sec_queen_global=\"\${${_be_prefix}sec_queen_global}\""
  eval "local _be_sec_queen_local=\"\${${_be_prefix}sec_queen_local}\""
  eval "local _be_sec_user_prefs=\"\${${_be_prefix}sec_user_prefs}\""
  eval "local _be_sec_hive=\"\${${_be_prefix}sec_hive}\""
  eval "local _be_sec_capsule=\"\${${_be_prefix}sec_capsule}\""
  eval "local _be_sec_learnings=\"\${${_be_prefix}sec_learnings}\""
  eval "local _be_sec_decisions=\"\${${_be_prefix}sec_decisions}\""
  eval "local _be_sec_blockers=\"\${${_be_prefix}sec_blockers}\""
  eval "local _be_sec_rolling=\"\${${_be_prefix}sec_rolling}\""
  eval "local _be_sec_signals=\"\${${_be_prefix}sec_signals}\""

  eval "local _be_max_chars=\${${_be_prefix}max_chars}"

  # Assemble all sections in order
  local _be_final_prompt="$_be_sec_queen_global$_be_sec_queen_local$_be_sec_user_prefs$_be_sec_hive$_be_sec_capsule$_be_sec_learnings$_be_sec_decisions$_be_sec_blockers$_be_sec_rolling$_be_sec_signals"

  local _be_budget_len=${#_be_final_prompt}
  local _be_trimmed_list=""

  if [[ "$_be_budget_len" -gt "$_be_max_chars" ]]; then
    # Over budget -- trim sections in priority order (first = trimmed first)

    # 1. Trim rolling-summary
    if [[ "$_be_budget_len" -gt "$_be_max_chars" && -n "$_be_sec_rolling" ]]; then
      _be_sec_rolling=""
      _be_trimmed_list="rolling-summary"
      _be_final_prompt="$_be_sec_queen_global$_be_sec_queen_local$_be_sec_user_prefs$_be_sec_hive$_be_sec_capsule$_be_sec_learnings$_be_sec_decisions$_be_sec_blockers$_be_sec_rolling$_be_sec_signals"
      _be_budget_len=${#_be_final_prompt}
    fi

    # 2. Trim phase-learnings
    if [[ "$_be_budget_len" -gt "$_be_max_chars" && -n "$_be_sec_learnings" ]]; then
      _be_sec_learnings=""
      _be_trimmed_list="${_be_trimmed_list:+$_be_trimmed_list,}phase-learnings"
      _be_final_prompt="$_be_sec_queen_global$_be_sec_queen_local$_be_sec_user_prefs$_be_sec_hive$_be_sec_capsule$_be_sec_learnings$_be_sec_decisions$_be_sec_blockers$_be_sec_rolling$_be_sec_signals"
      _be_budget_len=${#_be_final_prompt}
    fi

    # 3. Trim key-decisions
    if [[ "$_be_budget_len" -gt "$_be_max_chars" && -n "$_be_sec_decisions" ]]; then
      _be_sec_decisions=""
      _be_trimmed_list="${_be_trimmed_list:+$_be_trimmed_list,}key-decisions"
      _be_final_prompt="$_be_sec_queen_global$_be_sec_queen_local$_be_sec_user_prefs$_be_sec_hive$_be_sec_capsule$_be_sec_learnings$_be_sec_decisions$_be_sec_blockers$_be_sec_rolling$_be_sec_signals"
      _be_budget_len=${#_be_final_prompt}
    fi

    # 4. Trim hive-wisdom
    if [[ "$_be_budget_len" -gt "$_be_max_chars" && -n "$_be_sec_hive" ]]; then
      _be_sec_hive=""
      _be_trimmed_list="${_be_trimmed_list:+$_be_trimmed_list,}hive-wisdom"
      _be_final_prompt="$_be_sec_queen_global$_be_sec_queen_local$_be_sec_user_prefs$_be_sec_hive$_be_sec_capsule$_be_sec_learnings$_be_sec_decisions$_be_sec_blockers$_be_sec_rolling$_be_sec_signals"
      _be_budget_len=${#_be_final_prompt}
    fi

    # 5. Trim context-capsule
    if [[ "$_be_budget_len" -gt "$_be_max_chars" && -n "$_be_sec_capsule" ]]; then
      _be_sec_capsule=""
      _be_trimmed_list="${_be_trimmed_list:+$_be_trimmed_list,}context-capsule"
      _be_final_prompt="$_be_sec_queen_global$_be_sec_queen_local$_be_sec_user_prefs$_be_sec_hive$_be_sec_capsule$_be_sec_learnings$_be_sec_decisions$_be_sec_blockers$_be_sec_rolling$_be_sec_signals"
      _be_budget_len=${#_be_final_prompt}
    fi

    # 6. Trim user-prefs
    if [[ "$_be_budget_len" -gt "$_be_max_chars" && -n "$_be_sec_user_prefs" ]]; then
      _be_sec_user_prefs=""
      _be_trimmed_list="${_be_trimmed_list:+$_be_trimmed_list,}user-prefs"
      _be_final_prompt="$_be_sec_queen_global$_be_sec_queen_local$_be_sec_user_prefs$_be_sec_hive$_be_sec_capsule$_be_sec_learnings$_be_sec_decisions$_be_sec_blockers$_be_sec_rolling$_be_sec_signals"
      _be_budget_len=${#_be_final_prompt}
    fi

    # 7. Trim queen-wisdom-global (trim global before local -- local is more relevant)
    if [[ "$_be_budget_len" -gt "$_be_max_chars" && -n "$_be_sec_queen_global" ]]; then
      _be_sec_queen_global=""
      _be_trimmed_list="${_be_trimmed_list:+$_be_trimmed_list,}queen-wisdom-global"
      _be_final_prompt="$_be_sec_queen_global$_be_sec_queen_local$_be_sec_user_prefs$_be_sec_hive$_be_sec_capsule$_be_sec_learnings$_be_sec_decisions$_be_sec_blockers$_be_sec_rolling$_be_sec_signals"
      _be_budget_len=${#_be_final_prompt}
    fi

    # 8. Trim queen-wisdom-local
    if [[ "$_be_budget_len" -gt "$_be_max_chars" && -n "$_be_sec_queen_local" ]]; then
      _be_sec_queen_local=""
      _be_trimmed_list="${_be_trimmed_list:+$_be_trimmed_list,}queen-wisdom-local"
      _be_final_prompt="$_be_sec_queen_global$_be_sec_queen_local$_be_sec_user_prefs$_be_sec_hive$_be_sec_capsule$_be_sec_learnings$_be_sec_decisions$_be_sec_blockers$_be_sec_rolling$_be_sec_signals"
      _be_budget_len=${#_be_final_prompt}
    fi

    # 9. Trim pheromone-signals (preserve REDIRECTs)
    if [[ "$_be_budget_len" -gt "$_be_max_chars" && -n "$_be_sec_signals" ]]; then
      # Extract REDIRECT lines and preserve them
      local _be_redirect_preserved=""
      if [[ "$_be_sec_signals" == *"REDIRECT (HARD CONSTRAINTS"* ]]; then
        local _be_redirect_lines=""
        local _be_in_redirect=false
        local _be_rl
        while IFS= read -r _be_rl; do
          if [[ "$_be_rl" == *"REDIRECT (HARD CONSTRAINTS"* ]]; then
            _be_in_redirect=true
            _be_redirect_lines+="$_be_rl"$'\n'
          elif [[ "$_be_in_redirect" == "true" ]]; then
            if [[ "$_be_rl" == "FOCUS "* ]] || [[ "$_be_rl" == "FEEDBACK "* ]] || \
                 [[ "$_be_rl" == "POSITION "* ]] || [[ "$_be_rl" == "--- "* ]]; then
              _be_in_redirect=false
            else
              _be_redirect_lines+="$_be_rl"$'\n'
            fi
          fi
        done <<< "$_be_sec_signals"
        if [[ -n "$_be_redirect_lines" ]]; then
          _be_redirect_preserved=$'\n'"--- ACTIVE SIGNALS (Colony Guidance) ---"$'\n'
          _be_redirect_preserved+=$'\n'"$_be_redirect_lines"
          _be_redirect_preserved+=$'\n'"--- END COLONY CONTEXT ---"
        fi
      fi
      _be_sec_signals="$_be_redirect_preserved"
      _be_trimmed_list="${_be_trimmed_list:+$_be_trimmed_list,}pheromone-signals"
      _be_final_prompt="$_be_sec_queen_global$_be_sec_queen_local$_be_sec_user_prefs$_be_sec_hive$_be_sec_capsule$_be_sec_learnings$_be_sec_decisions$_be_sec_blockers$_be_sec_rolling$_be_sec_signals"
      _be_budget_len=${#_be_final_prompt}
    fi
  fi

  # Write back to caller's variables
  eval "${_be_prefix}sec_queen_global=\"\$_be_sec_queen_global\""
  eval "${_be_prefix}sec_queen_local=\"\$_be_sec_queen_local\""
  eval "${_be_prefix}sec_user_prefs=\"\$_be_sec_user_prefs\""
  eval "${_be_prefix}sec_hive=\"\$_be_sec_hive\""
  eval "${_be_prefix}sec_capsule=\"\$_be_sec_capsule\""
  eval "${_be_prefix}sec_learnings=\"\$_be_sec_learnings\""
  eval "${_be_prefix}sec_decisions=\"\$_be_sec_decisions\""
  eval "${_be_prefix}sec_blockers=\"\$_be_sec_blockers\""
  eval "${_be_prefix}sec_rolling=\"\$_be_sec_rolling\""
  eval "${_be_prefix}sec_signals=\"\$_be_sec_signals\""
  eval "${_be_prefix}final_prompt=\"\$_be_final_prompt\""
  eval "${_be_prefix}budget_len=\"\$_be_budget_len\""
  eval "${_be_prefix}budget_trimmed_list=\"\$_be_trimmed_list\""
}

# ============================================================================
# _colony_prime
# Unified colony priming: combines wisdom (QUEEN.md) + signals + instincts into single output
# ============================================================================
_colony_prime() {
# Unified colony priming: combines wisdom (QUEEN.md) + signals + instincts into single output
# Usage: colony-prime [--compact]
# Returns: JSON with wisdom, signals, prompt_section
# Error handling: QUEEN.md missing = FAIL HARD; pheromones.json missing = warn but continue

cp_compact=false
if [[ "${1:-}" == "--compact" ]]; then
  cp_compact=true
fi

# Total character budget for cp_final_prompt
cp_max_chars=8000
if [[ "$cp_compact" == "true" ]]; then
  cp_max_chars=4000
fi

cp_global_queen="$HOME/.aether/QUEEN.md"
cp_local_queen="$AETHER_ROOT/.aether/QUEEN.md"

# Track if we have any QUEEN.md
cp_has_global=false
cp_has_local=false
cp_wisdom_json='{}'

# Initialize empty wisdom objects (used if file doesn't exist) -- v2 keys
cp_global_wisdom='{"user_prefs":"","codebase_patterns":"","build_learnings":"","instincts":""}'
cp_local_wisdom='{"user_prefs":"","codebase_patterns":"","build_learnings":"","instincts":""}'

# Helper to filter wisdom entries, keeping only actual entries and phase headers
# Strips description paragraphs, placeholder text, and boilerplate
# Returns only lines starting with "- " (entries) or "### " (phase headers)
_filter_wisdom_entries() {
  local raw="$1"
  if [[ -z "$raw" || "$raw" == "null" ]]; then
    echo ""
    return
  fi
  echo "$raw" | grep -E '^(- |### )' || echo ""  # SUPPRESS:OK -- grep returns 1 on no matches
}

# Helper to extract wisdom sections from a QUEEN.md file
# Uses line number approach to avoid macOS awk range issues
# Supports both v2 (4-section) and v1 (6-emoji-section) formats
_extract_wisdom() {
  local queen_file="$1"

  # Format detection: check for v2 header "## Build Learnings"
  if grep -q '^## Build Learnings$' "$queen_file" 2>/dev/null; then
    # === V2 FORMAT (4 clean sections) ===
    local uprefs_line=$(awk '/^## User Preferences$/ {print NR; exit}' "$queen_file")
    local cpat_line=$(awk '/^## Codebase Patterns$/ {print NR; exit}' "$queen_file")
    local blearn_line=$(awk '/^## Build Learnings$/ {print NR; exit}' "$queen_file")
    local inst_line=$(awk '/^## Instincts$/ {print NR; exit}' "$queen_file")
    local evo_line=$(awk '/^## Evolution Log$/ {print NR; exit}' "$queen_file")

    local user_prefs codebase_patterns build_learnings instincts

    local uprefs_end="${cpat_line:-${blearn_line:-${inst_line:-${evo_line:-999999}}}}"
    if [[ -n "$uprefs_line" ]]; then
      # SUPPRESS:OK -- read-default: text escaping returns fallback on empty input
      user_prefs=$(awk -v s="$uprefs_line" -v e="$uprefs_end" 'NR > s && NR < e {print}' "$queen_file" | sed '/^$/d' | sed '/^---$/d' | jq -Rs '.' 2>/dev/null || echo '""')
    else user_prefs='""'; fi

    local cpat_end="${blearn_line:-${inst_line:-${evo_line:-999999}}}"
    if [[ -n "$cpat_line" ]]; then
      # SUPPRESS:OK -- read-default: text escaping returns fallback on empty input
      codebase_patterns=$(awk -v s="$cpat_line" -v e="$cpat_end" 'NR > s && NR < e {print}' "$queen_file" | sed '/^$/d' | sed '/^---$/d' | jq -Rs '.' 2>/dev/null || echo '""')
    else codebase_patterns='""'; fi

    local blearn_end="${inst_line:-${evo_line:-999999}}"
    if [[ -n "$blearn_line" ]]; then
      # SUPPRESS:OK -- read-default: text escaping returns fallback on empty input
      build_learnings=$(awk -v s="$blearn_line" -v e="$blearn_end" 'NR > s && NR < e {print}' "$queen_file" | sed '/^$/d' | sed '/^---$/d' | jq -Rs '.' 2>/dev/null || echo '""')
    else build_learnings='""'; fi

    if [[ -n "$inst_line" ]]; then
      # SUPPRESS:OK -- read-default: text escaping returns fallback on empty input
      instincts=$(awk -v s="$inst_line" -v e="${evo_line:-999999}" 'NR > s && NR < e {print}' "$queen_file" | sed '/^$/d' | sed '/^---$/d' | jq -Rs '.' 2>/dev/null || echo '""')
    else instincts='""'; fi

    user_prefs=${user_prefs:-'""'}
    codebase_patterns=${codebase_patterns:-'""'}
    build_learnings=${build_learnings:-'""'}
    instincts=${instincts:-'""'}

    echo "{\"user_prefs\":$user_prefs,\"codebase_patterns\":$codebase_patterns,\"build_learnings\":$build_learnings,\"instincts\":$instincts}"

  else
    # === V1 FORMAT (6 emoji sections, mapped to v2 keys) ===
    local p_line=$(awk '/^## ..? ?Philosophies$/ {print NR; exit}' "$queen_file")
    local pat_line=$(awk '/^## ..? ?Patterns$/ {print NR; exit}' "$queen_file")
    local red_line=$(awk '/^## ..? ?Redirects$/ {print NR; exit}' "$queen_file")
    local stack_line=$(awk '/^## ..? ?Stack Wisdom$/ {print NR; exit}' "$queen_file")
    local dec_line=$(awk '/^## ..? ?Decrees$/ {print NR; exit}' "$queen_file")
    local prefs_line=$(awk '/^## ..? ?User Preferences$/ {print NR; exit}' "$queen_file")
    local evo_line=$(awk '/^## ..? ?Evolution Log$/ {print NR; exit}' "$queen_file")

    local philosophies patterns redirects stack_wisdom decrees user_prefs

    if [[ -n "$p_line" && -n "$pat_line" ]]; then
      # SUPPRESS:OK -- read-default: text escaping returns fallback on empty input
      philosophies=$(awk -v s="$p_line" -v e="$pat_line" 'NR > s && NR < e {print}' "$queen_file" | sed '/^$/d' | jq -Rs '.' 2>/dev/null || echo '""')
    else philosophies='""'; fi
    if [[ -n "$pat_line" && -n "$red_line" ]]; then
      # SUPPRESS:OK -- read-default: text escaping returns fallback on empty input
      patterns=$(awk -v s="$pat_line" -v e="$red_line" 'NR > s && NR < e {print}' "$queen_file" | sed '/^$/d' | jq -Rs '.' 2>/dev/null || echo '""')
    else patterns='""'; fi
    if [[ -n "$red_line" && -n "$stack_line" ]]; then
      # SUPPRESS:OK -- read-default: text escaping returns fallback on empty input
      redirects=$(awk -v s="$red_line" -v e="$stack_line" 'NR > s && NR < e {print}' "$queen_file" | sed '/^$/d' | jq -Rs '.' 2>/dev/null || echo '""')
    else redirects='""'; fi
    if [[ -n "$stack_line" && -n "$dec_line" ]]; then
      # SUPPRESS:OK -- read-default: text escaping returns fallback on empty input
      stack_wisdom=$(awk -v s="$stack_line" -v e="$dec_line" 'NR > s && NR < e {print}' "$queen_file" | sed '/^$/d' | jq -Rs '.' 2>/dev/null || echo '""')
    else stack_wisdom='""'; fi

    local dec_end="${prefs_line:-${evo_line:-999999}}"
    if [[ -n "$dec_line" ]]; then
      # SUPPRESS:OK -- read-default: text escaping returns fallback on empty input
      decrees=$(awk -v s="$dec_line" -v e="$dec_end" 'NR > s && NR < e {print}' "$queen_file" | sed '/^$/d' | jq -Rs '.' 2>/dev/null || echo '""')
    else decrees='""'; fi

    if [[ -n "$prefs_line" ]]; then
      # SUPPRESS:OK -- read-default: text escaping returns fallback on empty input
      user_prefs=$(awk -v s="$prefs_line" -v e="${evo_line:-999999}" 'NR > s && NR < e {print}' "$queen_file" | sed '/^$/d' | jq -Rs '.' 2>/dev/null || echo '""')
    else user_prefs='""'; fi

    philosophies=${philosophies:-'""'}
    patterns=${patterns:-'""'}
    redirects=${redirects:-'""'}
    stack_wisdom=${stack_wisdom:-'""'}
    decrees=${decrees:-'""'}
    user_prefs=${user_prefs:-'""'}

    # Map v1 -> v2: combine old sections into new keys
    local combined_codebase
    combined_codebase=$(jq -n \
      --arg phil "$philosophies" \
      --arg pat "$patterns" \
      --arg red "$redirects" \
      --arg stack "$stack_wisdom" \
      '[$phil, $pat, $red, $stack] | map(select(. != "" and . != null)) | join("\n")' 2>/dev/null || echo '""')

    local combined_uprefs
    combined_uprefs=$(jq -n \
      --arg dec "$decrees" \
      --arg up "$user_prefs" \
      '[$dec, $up] | map(select(. != "" and . != null)) | join("\n")' 2>/dev/null || echo '""')

    echo "{\"user_prefs\":$combined_uprefs,\"codebase_patterns\":$combined_codebase,\"build_learnings\":\"\",\"instincts\":\"\"}"
  fi
}

# Detect if global and local QUEEN.md point to the same file (e.g., HOME == AETHER_ROOT in tests)
# In that case, treat as local only to avoid double-loading the same content
cp_same_queen=false
if [[ -f "$cp_global_queen" && -f "$cp_local_queen" ]]; then
  cp_global_real=$(cd "$(dirname "$cp_global_queen")" && pwd)/$(basename "$cp_global_queen") 2>/dev/null || true  # SUPPRESS:OK -- read-default: path resolution
  cp_local_real=$(cd "$(dirname "$cp_local_queen")" && pwd)/$(basename "$cp_local_queen") 2>/dev/null || true  # SUPPRESS:OK -- read-default: path resolution
  if [[ "$cp_global_real" == "$cp_local_real" ]]; then
    cp_same_queen=true
  fi
fi

# Load global QUEEN.md first (~/.aether/QUEEN.md)
# Skip if same file as local (will be loaded as local instead)
if [[ -f "$cp_global_queen" && "$cp_same_queen" == "false" ]]; then
  cp_has_global=true
  # Auto-migrate global QUEEN.md from v1 to v2 if needed (Phase 20)
  if ! grep -q '^## Build Learnings$' "$cp_global_queen" 2>/dev/null; then  # SUPPRESS:OK -- existence-test: format detection
    "$SCRIPT_DIR/aether-utils.sh" queen-migrate --target hub 2>/dev/null || true  # SUPPRESS:OK -- cleanup: migration is best-effort
  fi
  cp_global_wisdom=$(_extract_wisdom "$cp_global_queen" "g")
fi

# Load local QUEEN.md second (.aether/QUEEN.md)
if [[ -f "$cp_local_queen" ]]; then
  cp_has_local=true
  # Auto-migrate local QUEEN.md if same as global and was v1 (edge case: HOME == AETHER_ROOT)
  if [[ "$cp_same_queen" == "true" ]] && ! grep -q '^## Build Learnings$' "$cp_local_queen" 2>/dev/null; then  # SUPPRESS:OK -- existence-test: format detection
    "$SCRIPT_DIR/aether-utils.sh" queen-migrate --target local 2>/dev/null || true  # SUPPRESS:OK -- cleanup: migration is best-effort
  fi
  cp_local_wisdom=$(_extract_wisdom "$cp_local_queen" "l")
fi

# FAIL HARD if no QUEEN.md found at all
if [[ "$cp_has_global" == "false" && "$cp_has_local" == "false" ]]; then
  json_err "$E_FILE_NOT_FOUND" \
    "QUEEN.md not found in either ~/.aether/QUEEN.md or .aether/QUEEN.md. Run /ant:init to create a colony." \
    '{"global_path":"~/.aether/QUEEN.md","local_path":".aether/QUEEN.md"}'
  exit 1
fi

# Process global and local wisdom independently (Phase 20: split sections)
# --- GLOBAL wisdom extraction ---
cp_global_codebase_raw=$(echo "$cp_global_wisdom" | jq -r '.codebase_patterns // ""' 2>/dev/null)  # SUPPRESS:OK -- read-default: may be empty
cp_global_instincts_raw=$(echo "$cp_global_wisdom" | jq -r '.instincts // ""' 2>/dev/null)  # SUPPRESS:OK -- read-default: may be empty
cp_global_prefs_raw=$(echo "$cp_global_wisdom" | jq -r '.user_prefs // ""' 2>/dev/null)  # SUPPRESS:OK -- read-default: may be empty

# --- LOCAL wisdom extraction ---
cp_local_codebase_raw=$(echo "$cp_local_wisdom" | jq -r '.codebase_patterns // ""' 2>/dev/null)  # SUPPRESS:OK -- read-default: may be empty
cp_local_learnings_raw=$(echo "$cp_local_wisdom" | jq -r '.build_learnings // ""' 2>/dev/null)  # SUPPRESS:OK -- read-default: may be empty
cp_local_instincts_raw=$(echo "$cp_local_wisdom" | jq -r '.instincts // ""' 2>/dev/null)  # SUPPRESS:OK -- read-default: may be empty
cp_local_prefs_raw=$(echo "$cp_local_wisdom" | jq -r '.user_prefs // ""' 2>/dev/null)  # SUPPRESS:OK -- read-default: may be empty

# --- Filter entries independently ---
cp_global_codebase=$(_filter_wisdom_entries "$cp_global_codebase_raw")
cp_global_instincts=$(_filter_wisdom_entries "$cp_global_instincts_raw")
cp_local_codebase=$(_filter_wisdom_entries "$cp_local_codebase_raw")
cp_local_learnings=$(_filter_wisdom_entries "$cp_local_learnings_raw")
cp_local_instincts=$(_filter_wisdom_entries "$cp_local_instincts_raw")

# Get metadata from local QUEEN.md if exists, otherwise global
cp_metadata='{"version":"unknown","last_evolved":null,"source":"none"}'
if [[ "$cp_has_local" == "true" ]]; then
  cp_metadata=$(sed -n '/<!-- METADATA/,/-->/p' "$cp_local_queen" | sed '1d;$d' | tr -d '\n' | sed 's/^[[:space:]]*//')
  if [[ -n "$cp_metadata" ]] && echo "$cp_metadata" | jq -e . >/dev/null 2>&1; then  # SUPPRESS:OK -- validation: testing JSON validity
    # SUPPRESS:OK -- read-default: returns fallback on failure
    cp_metadata=$(echo "$cp_metadata" | jq '. + {"source":"local"}' 2>/dev/null || echo "$cp_metadata")
  else
    cp_metadata='{"version":"unknown","last_evolved":null,"source":"local","note":"malformed"}'
  fi
elif [[ "$cp_has_global" == "true" ]]; then
  cp_metadata=$(sed -n '/<!-- METADATA/,/-->/p' "$cp_global_queen" | sed '1d;$d' | tr -d '\n' | sed 's/^[[:space:]]*//')
  if [[ -n "$cp_metadata" ]] && echo "$cp_metadata" | jq -e . >/dev/null 2>&1; then  # SUPPRESS:OK -- validation: testing JSON validity
    # SUPPRESS:OK -- read-default: returns fallback on failure
    cp_metadata=$(echo "$cp_metadata" | jq '. + {"source":"global"}' 2>/dev/null || echo "$cp_metadata")
  else
    cp_metadata='{"version":"unknown","last_evolved":null,"source":"global","note":"malformed"}'
  fi
fi

# Now get signals + instincts via pheromone-prime
# Trap error: if pheromones.json missing, warn but continue
# Call pheromone-prime by re-invoking the script (it's a case branch, not a function)
cp_signals_json='{"signal_count":0,"instinct_count":0,"prompt_section":"","log_line":"Primed: no pheromones (file missing)"}'
cp_pher_warn=""
if [[ -f "$COLONY_DATA_DIR/pheromones.json" ]]; then
  if [[ "$cp_compact" == "true" ]]; then
    # SUPPRESS:OK -- read-default: subcommand call returns fallback on failure
    cp_signals_raw=$("$SCRIPT_DIR/aether-utils.sh" pheromone-prime --compact --max-signals 8 --max-instincts 3 2>/dev/null) || cp_signals_raw=""
  else
    cp_signals_raw=$("$SCRIPT_DIR/aether-utils.sh" pheromone-prime 2>/dev/null) || cp_signals_raw=""  # SUPPRESS:OK -- read-default: subcommand may fail
  fi
  # SUPPRESS:OK -- read-default: query may return empty
  cp_signals_json=$(echo "$cp_signals_raw" | jq -c '.result // {"signal_count":0,"instinct_count":0,"prompt_section":"","log_line":"Primed: 0 signals, 0 instincts"}' 2>/dev/null || echo '{"signal_count":0,"instinct_count":0,"prompt_section":"","log_line":"Primed: 0 signals, 0 instincts"}')
else
  cp_pher_warn="WARNING: pheromones.json not found - continuing without signals"
fi

# Extract components from pheromone-prime output
cp_signal_count=$(echo "$cp_signals_json" | jq -r '.signal_count // 0' 2>/dev/null || echo "0")  # SUPPRESS:OK -- read-default: file may not exist yet
cp_instinct_count=$(echo "$cp_signals_json" | jq -r '.instinct_count // 0' 2>/dev/null || echo "0")  # SUPPRESS:OK -- read-default: file may not exist yet
cp_prompt_section=$(echo "$cp_signals_json" | jq -r '.prompt_section // ""' 2>/dev/null || echo "")  # SUPPRESS:OK -- read-default: file may not exist yet
# SUPPRESS:OK -- read-default: query may return empty
cp_log_line=$(echo "$cp_signals_json" | jq -r '.log_line // "Primed: 0 signals, 0 instincts"' 2>/dev/null || echo "Primed: 0 signals, 0 instincts")

# Append warning if pheromones missing
if [[ -n "$cp_pher_warn" ]]; then
  cp_log_line="$cp_log_line; $cp_pher_warn"
fi

# Build prompt_section that combines wisdom + signals
# Each section is stored separately for budget enforcement
cp_final_prompt=""
cp_sec_queen_global=""
cp_sec_queen_local=""
cp_sec_user_prefs=""
cp_sec_hive=""
cp_sec_capsule=""
cp_sec_learnings=""
cp_sec_decisions=""
cp_sec_blockers=""
cp_sec_rolling=""
cp_sec_signals=""

# Build GLOBAL QUEEN WISDOM section (only if real filtered content exists)
if [[ -n "$cp_global_codebase" || -n "$cp_global_instincts" ]]; then
  cp_sec_queen_global+="--- QUEEN WISDOM (Global -- All Colonies) ---"$'\n'

  if [[ -n "$cp_global_codebase" ]]; then
    cp_sec_queen_global+=$'\n'"Codebase Patterns:"$'\n'"$cp_global_codebase"$'\n'
  fi
  if [[ -n "$cp_global_instincts" ]]; then
    cp_sec_queen_global+=$'\n'"Instincts:"$'\n'"$cp_global_instincts"$'\n'
  fi

  cp_sec_queen_global+=$'\n'"--- END QUEEN WISDOM (Global) ---"$'\n'
fi

# Build LOCAL (Colony-Specific) QUEEN WISDOM section
if [[ -n "$cp_local_codebase" || -n "$cp_local_learnings" || -n "$cp_local_instincts" ]]; then
  cp_sec_queen_local+="--- QUEEN WISDOM (Colony-Specific) ---"$'\n'

  if [[ -n "$cp_local_codebase" ]]; then
    cp_sec_queen_local+=$'\n'"Codebase Patterns:"$'\n'"$cp_local_codebase"$'\n'
  fi
  if [[ -n "$cp_local_learnings" ]]; then
    cp_sec_queen_local+=$'\n'"Build Learnings:"$'\n'"$cp_local_learnings"$'\n'
  fi
  if [[ -n "$cp_local_instincts" ]]; then
    cp_sec_queen_local+=$'\n'"Instincts:"$'\n'"$cp_local_instincts"$'\n'
  fi

  cp_sec_queen_local+=$'\n'"--- END QUEEN WISDOM (Colony-Specific) ---"$'\n'
fi

# Build USER PREFERENCES section with source labels (Phase 20)
cp_sec_user_prefs=""
cp_user_prefs_count=0

# Label global prefs with [global] prefix
cp_global_prefs_labeled=""
if [[ -n "$cp_global_prefs_raw" && "$cp_global_prefs_raw" != "null" ]]; then
  cp_global_prefs_labeled=$(echo "$cp_global_prefs_raw" | grep '^- ' | sed 's/^- /- [global] /' || true)  # SUPPRESS:OK -- grep returns 1 on no matches
fi

# Label local prefs with [local] prefix
cp_local_prefs_labeled=""
if [[ -n "$cp_local_prefs_raw" && "$cp_local_prefs_raw" != "null" ]]; then
  cp_local_prefs_labeled=$(echo "$cp_local_prefs_raw" | grep '^- ' | sed 's/^- /- [local] /' || true)  # SUPPRESS:OK -- grep returns 1 on no matches
fi

# Combine labeled prefs
cp_all_prefs=""
[[ -n "$cp_global_prefs_labeled" ]] && cp_all_prefs+="$cp_global_prefs_labeled"$'\n'
[[ -n "$cp_local_prefs_labeled" ]] && cp_all_prefs+="$cp_local_prefs_labeled"$'\n'

if [[ -n "$cp_all_prefs" ]]; then
  cp_user_prefs_count=$(echo "$cp_all_prefs" | grep -c '^- ' || echo "0")  # SUPPRESS:OK -- read-default: grep returns 1 when no matches
  if [[ "$cp_user_prefs_count" -gt 0 ]]; then
    cp_sec_user_prefs=$'\n'"--- USER PREFERENCES ---"$'\n'
    cp_sec_user_prefs+="$cp_all_prefs"
    cp_sec_user_prefs+="--- END USER PREFERENCES ---"$'\n'
    cp_log_line="$cp_log_line, $cp_user_prefs_count user_prefs"
  fi
fi

# === Hive-wisdom injection (HIVE-01) ===
# Primary: use hive-read with domain tags from registry for scoped wisdom
# Fallback: read high_value_signals from ~/.aether/eternal/memory.json
cp_hive_count=0
cp_sec_hive=""
cp_hive_source=""

cp_max_hive=5
if [[ "$cp_compact" == "true" ]]; then
  cp_max_hive=3
fi

# Get domain tags for current repo from registry
cp_repo_path="${AETHER_ROOT:-$(pwd)}"
# SUPPRESS:OK -- read-default: file may not exist yet
cp_domain_tags=$(jq -r --arg repo "$cp_repo_path" \
  '[.repos[] | select(.path == $repo) | .domain_tags // []] | .[0] // [] | join(",")' \
  "$HOME/.aether/registry.json" 2>/dev/null || echo "")  # SUPPRESS:OK -- read-default: returns fallback on failure

# Try hive-read first (domain-scoped retrieval from ~/.aether/hive/wisdom.json)
cp_hive_result=""
if [[ -n "$cp_domain_tags" ]]; then
  # SUPPRESS:OK -- read-default: subcommand call returns fallback on failure
  cp_hive_result=$(bash "$SCRIPT_DIR/aether-utils.sh" hive-read --domain "$cp_domain_tags" --limit "$cp_max_hive" --format text 2>/dev/null) || cp_hive_result=""
else
  # SUPPRESS:OK -- read-default: subcommand call returns fallback on failure
  cp_hive_result=$(bash "$SCRIPT_DIR/aether-utils.sh" hive-read --limit "$cp_max_hive" --format text 2>/dev/null) || cp_hive_result=""
fi

cp_hive_matched=0
if [[ -n "$cp_hive_result" ]]; then
  # SUPPRESS:OK -- read-default: query may return empty
  cp_hive_matched=$(echo "$cp_hive_result" | jq -r '.result.total_matched // 0' 2>/dev/null || echo "0")
fi

if [[ "$cp_hive_matched" -gt 0 ]]; then
  # Use hive-read text output
  cp_hive_text=$(echo "$cp_hive_result" | jq -r '.result.text // ""' 2>/dev/null || echo "")  # SUPPRESS:OK -- read-default: file may not exist yet
  cp_hive_count="$cp_hive_matched"
  if [[ "$cp_hive_count" -gt "$cp_max_hive" ]]; then
    cp_hive_count="$cp_max_hive"
  fi
  cp_hive_source="hive"

  # Build header with domain info
  if [[ -n "$cp_domain_tags" ]]; then
    cp_domain_display=$(echo "$cp_domain_tags" | tr ',' ', ')
    cp_hive_section="--- HIVE WISDOM (Domain: $cp_domain_display) ---"$'\n'
  else
    cp_hive_section="--- HIVE WISDOM (All Domains) ---"$'\n'
  fi

  # Add hive-read text lines
  if [[ -n "$cp_hive_text" && "$cp_hive_text" != "(no wisdom entries)" ]]; then
    while IFS= read -r cp_hive_line; do
      [[ -n "$cp_hive_line" ]] && cp_hive_section+="- $cp_hive_line"$'\n'
    done <<< "$cp_hive_text"
  fi

  cp_hive_section+="--- END HIVE WISDOM ---"
  cp_sec_hive=$'\n'"$cp_hive_section"$'\n'
  cp_log_line="$cp_log_line, $cp_hive_count hive"
else
  # Fallback: read from eternal memory (legacy)
  cp_hive_file="$HOME/.aether/eternal/memory.json"
  if [[ -f "$cp_hive_file" ]]; then
    cp_hive_signals=$(jq -r \
      --argjson max "$cp_max_hive" \
      '
      .high_value_signals // []
      | .[:$max]
      ' "$cp_hive_file" 2>/dev/null || echo "[]")  # SUPPRESS:OK -- read-default: file may not exist yet

    cp_hive_count=$(echo "$cp_hive_signals" | jq 'length' 2>/dev/null || echo "0")  # SUPPRESS:OK -- read-default: file may not exist yet

    if [[ "$cp_hive_count" -gt 0 ]]; then
      cp_hive_section="--- HIVE WISDOM (Cross-Colony Patterns) ---"$'\n'
      cp_hive_source="eternal"

      cp_hive_lines=$(echo "$cp_hive_signals" | jq -r '
        .[] | "[" + (.type // "UNKNOWN") + " | " + ((.strength // 0) | tostring) + "] " + (.content // "")
      ' 2>/dev/null || echo "")  # SUPPRESS:OK -- read-default: returns fallback on failure

      if [[ -n "$cp_hive_lines" ]]; then
        while IFS= read -r cp_hive_line; do
          [[ -n "$cp_hive_line" ]] && cp_hive_section+="- $cp_hive_line"$'\n'
        done <<< "$cp_hive_lines"
      fi

      cp_hive_section+="--- END HIVE WISDOM ---"

      cp_sec_hive=$'\n'"$cp_hive_section"$'\n'
      cp_log_line="$cp_log_line, $cp_hive_count hive"
    fi
  fi
fi
# === END hive-wisdom injection ===

# Add compact context capsule for low-token continuity
cp_capsule_prompt=""
# SUPPRESS:OK -- read-default: subcommand call returns fallback on failure
cp_capsule_raw=$("$SCRIPT_DIR/aether-utils.sh" context-capsule --compact --json 2>/dev/null) || cp_capsule_raw=""
# SUPPRESS:OK -- read-default: query may return empty
cp_capsule_prompt=$(echo "$cp_capsule_raw" | jq -r '.result.prompt_section // ""' 2>/dev/null || echo "")
if [[ -n "$cp_capsule_prompt" ]]; then
  cp_sec_capsule=$'\n'"$cp_capsule_prompt"$'\n'
fi

# === Phase learnings injection ===
# MIGRATE: direct COLONY_STATE.json access -- use _state_read_field instead
# Extract validated learnings from previous phases in COLONY_STATE.json
# and format as actionable guidance for builders
cp_current_phase=$(jq -r '.current_phase // 0' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null || echo "0")  # SUPPRESS:OK -- read-default: file may not exist yet

cp_max_learnings=15
if [[ "$cp_compact" == "true" ]]; then
  cp_max_learnings=5
fi

cp_learning_claims=$(jq -r \
  --argjson current "$cp_current_phase" \
  --argjson max "$cp_max_learnings" \
  '
  [
    (.memory.phase_learnings // [])[]
    | select((.phase | type) == "string" or ((.phase | tonumber) < $current))
    | .phase as $p | .phase_name as $pn
    | .learnings[]
    | select(.status == "validated")
    | {phase: $p, phase_name: $pn, claim: .claim}
  ]
  | unique_by(.claim)
  | .[:$max]
  ' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null || echo "[]")  # SUPPRESS:OK -- read-default: file may not exist yet

cp_learning_count=$(echo "$cp_learning_claims" | jq 'length' 2>/dev/null || echo "0")  # SUPPRESS:OK -- read-default: file may not exist yet

if [[ "$cp_learning_count" -gt 0 ]]; then
  cp_learning_section="--- PHASE LEARNINGS (Previous Phase Insights) ---"

  cp_learning_lines=$(echo "$cp_learning_claims" | jq -r '
    group_by(.phase)
    | map({
        phase: .[0].phase,
        phase_name: .[0].phase_name,
        claims: [.[].claim]
      })
    | sort_by(if .phase == "inherited" then -1 else (.phase | tonumber) end)
    | .[]
    | "\n"
      + (if .phase == "inherited" then "Inherited"
         elif .phase_name != "" then "Phase " + (.phase | tostring) + " (" + .phase_name + ")"
         else "Phase " + (.phase | tostring)
         end)
      + ":"
      + "\n" + (.claims | map("  - " + .) | join("\n"))
  ' 2>/dev/null || echo "")  # SUPPRESS:OK -- read-default: returns fallback on failure

  if [[ -n "$cp_learning_lines" ]]; then
    cp_learning_section+="$cp_learning_lines"$'\n'
  fi

  cp_learning_section+=$'\n'"--- END PHASE LEARNINGS ---"

  cp_sec_learnings=$'\n'"$cp_learning_section"$'\n'

  cp_log_line="$cp_log_line, $cp_learning_count learnings"
fi
# === End phase learnings injection ===

# === CONTEXT.md decision injection (CTX-01) ===
# Extract key decisions from CONTEXT.md "Recent Decisions" table
# and inject as actionable context for builders
cp_ctx_file="$AETHER_ROOT/.aether/CONTEXT.md"
cp_decision_count=0

cp_decisions=""
if [[ -f "$cp_ctx_file" ]]; then
  cp_decisions=$(awk '
    /^## .*Recent Decisions/ { in_section=1; next }
    in_section && /^\| Date / { next }
    in_section && /^\|[-]+/ { next }
    in_section && /^---/ { exit }
    in_section && /^\| [0-9]{4}-[0-9]{2}/ {
      split($0, fields, "|")
      decision = fields[3]
      rationale = fields[4]
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", decision)
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", rationale)
      if (decision != "") {
        if (rationale != "" && rationale != "-") {
          print decision " (" rationale ")"
        } else {
          print decision
        }
      }
    }
  ' "$cp_ctx_file" 2>/dev/null || echo "")  # SUPPRESS:OK -- read-default: file may not exist yet
fi

cp_max_decisions=5
if [[ "$cp_compact" == "true" ]]; then
  cp_max_decisions=3
fi

if [[ -n "$cp_decisions" ]]; then
  cp_trimmed_decisions=$(echo "$cp_decisions" | tail -n "$cp_max_decisions")
  cp_decision_count=$(echo "$cp_trimmed_decisions" | grep -c '.' || echo "0")  # SUPPRESS:OK -- read-default: grep returns 1 when no matches

  if [[ "$cp_decision_count" -gt 0 ]]; then
    cp_decision_section="--- KEY DECISIONS (Active Decisions) ---"$'\n'
    while IFS= read -r cp_dec_line; do
      [[ -n "$cp_dec_line" ]] && cp_decision_section+="- $cp_dec_line"$'\n'
    done <<< "$cp_trimmed_decisions"
    cp_decision_section+="--- END KEY DECISIONS ---"

    cp_sec_decisions=$'\n'"$cp_decision_section"$'\n'
    cp_log_line="$cp_log_line, $cp_decision_count decisions"
  fi
fi
# === END CONTEXT.md decision injection ===

# === Blocker flag injection (CTX-02) ===
# Extract unresolved blocker flags for the current phase from flags.json
# and inject as REDIRECT-priority warnings distinct from user pheromones
cp_flags_file="$COLONY_DATA_DIR/flags.json"
cp_blocker_count=0

cp_blockers=""
if [[ -f "$cp_flags_file" ]]; then
  cp_blockers=$(jq -r \
    --argjson phase "$cp_current_phase" \
    '
    .flags
    | map(select(
        .type == "blocker"
        and .resolved_at == null
        and (.phase == $phase or .phase == null)
      ))
    | map("[source: " + (.source // "unknown") + "] " + .title + "\n  " + (.description // ""))
    | .[]
    ' "$cp_flags_file" 2>/dev/null || echo "")  # SUPPRESS:OK -- read-default: file may not exist yet
fi

cp_max_blockers=3
if [[ "$cp_compact" == "true" ]]; then
  cp_max_blockers=2
fi

if [[ -n "$cp_blockers" ]]; then
  cp_blocker_count=$(echo "$cp_blockers" | grep -c '^\[source:' || echo "0")  # SUPPRESS:OK -- read-default: grep returns 1 when no matches

  if [[ "$cp_blocker_count" -gt 0 ]]; then
    cp_blocker_section="--- BLOCKER WARNINGS (Unresolved Build Blockers) ---"$'\n'
    cp_blocker_section+="These are critical issues that MUST be addressed. Treat as REDIRECT-priority."$'\n'

    cp_blocker_idx=0
    while IFS= read -r cp_blk_line; do
      if [[ "$cp_blocker_idx" -ge "$cp_max_blockers" ]]; then break; fi
      if [[ "$cp_blk_line" == \[source:* ]]; then
        ((cp_blocker_idx++)) || true  # SUPPRESS:OK -- cleanup: arithmetic overflow is safe
        if [[ "$cp_blocker_idx" -gt "$cp_max_blockers" ]]; then break; fi
      fi
      [[ -n "$cp_blk_line" ]] && cp_blocker_section+="$cp_blk_line"$'\n'
    done <<< "$cp_blockers"

    cp_blocker_section+="--- END BLOCKER WARNINGS ---"

    cp_sec_blockers=$'\n'"$cp_blocker_section"$'\n'
    cp_log_line="$cp_log_line, $cp_blocker_count blockers"
  fi
fi
# === END blocker flag injection ===

# === Rolling-summary injection (MEM-02) ===
# Read last 5 entries directly (not via context-capsule which truncates)
cp_roll_count=5
cp_roll_entries=""
if [[ -f "$COLONY_DATA_DIR/rolling-summary.log" ]]; then
  # SUPPRESS:OK -- read-default: file may not exist
  # SUPPRESS:OK -- read-default: file may not exist yet
  cp_roll_entries=$(tail -n "$cp_roll_count" "$COLONY_DATA_DIR/rolling-summary.log" 2>/dev/null | \
    awk -F'|' 'NF >= 4 {printf "- [%s] %s: %s\n", $1, $2, $4}')
fi

if [[ -n "$cp_roll_entries" ]]; then
  cp_sec_rolling=$'\n'"--- RECENT ACTIVITY (Colony Narrative) ---"$'\n'
  cp_sec_rolling+="$cp_roll_entries"$'\n'
  cp_sec_rolling+="--- END RECENT ACTIVITY ---"$'\n'

  cp_roll_actual=$(echo "$cp_roll_entries" | grep -c '.' || echo "0")  # SUPPRESS:OK -- read-default: grep returns 1 when no matches
  cp_log_line="$cp_log_line, $cp_roll_actual activity entries"
fi
# === END rolling-summary injection ===

# Add pheromone signals section
if [[ -n "$cp_prompt_section" && "$cp_prompt_section" != "null" ]]; then
  cp_sec_signals=$'\n'"$cp_prompt_section"
fi

# === Budget enforcement (BUDGET-01) ===
# Assemble cp_final_prompt from sections, respecting cp_max_chars budget.
# Truncation priority (trim first to last):
#   rolling-summary > phase-learnings > key-decisions > hive-wisdom >
#   context-capsule > user-prefs > queen-wisdom-global > queen-wisdom-local > pheromone-signals (NEVER trim REDIRECTs)
# Blockers are always kept (REDIRECT-priority).

_budget_enforce "cp_"

# Append truncation note to log line (post _budget_enforce)
if [[ -n "${cp_budget_trimmed_list:-}" ]]; then
  cp_log_line="$cp_log_line, truncated: $cp_budget_trimmed_list (budget: ${cp_max_chars})"
fi
# === END Budget enforcement ===

# === Budget trimming notification (REL-06) ===
cp_trimmed_notice=""
cp_trimmed_high_priority=false

if [[ -n "${cp_budget_trimmed_list:-}" ]]; then
  cp_trimmed_sections=$(echo "$cp_budget_trimmed_list" | tr ',' ', ')

  if [[ "$cp_budget_trimmed_list" == *"key-decisions"* ]] || \
     [[ "$cp_budget_trimmed_list" == *"pheromone-signals"* ]]; then
    cp_trimmed_high_priority=true
    cp_trimmed_notice="[!trimmed] Context exceeded ${cp_max_chars}-char budget. Dropped: ${cp_trimmed_sections}. HIGH-PRIORITY items were trimmed -- key decisions or redirect signals may be missing."
    echo "[!trimmed] Colony context exceeded budget. High-priority sections dropped: $cp_trimmed_sections" >&2
  else
    cp_trimmed_notice="[trimmed] Context exceeded ${cp_max_chars}-char budget. Dropped: ${cp_trimmed_sections}."
    echo "[trimmed] Colony context exceeded budget. Dropped: $cp_trimmed_sections" >&2
  fi
fi
# === END Budget trimming notification ===

# Escape for JSON
cp_prompt_json=$(printf '%s' "$cp_final_prompt" | jq -Rs '.' 2>/dev/null || echo '""')  # SUPPRESS:OK -- read-default: returns fallback if missing
# SUPPRESS:OK -- read-default: text escaping returns fallback on empty input
cp_log_json=$(printf '%s' "$cp_log_line" | jq -Rs '.' 2>/dev/null || echo '"Primed: 0 signals, 0 instincts"')

# Build final unified output (Phase 20: split global/local wisdom)
cp_result=$(jq -n \
  --argjson meta "$cp_metadata" \
  --argjson wisdom_global "$cp_global_wisdom" \
  --argjson wisdom_local "$cp_local_wisdom" \
  --argjson signals "$cp_signals_json" \
  --arg prompt "$cp_final_prompt" \
  --arg prompt_json "$cp_prompt_json" \
  --arg log "$cp_log_line" \
  --arg log_json "$cp_log_json" \
  --arg trimmed_notice "$cp_trimmed_notice" \
  --argjson trimmed_high_priority "${cp_trimmed_high_priority:-false}" \
  '{
    metadata: $meta,
    wisdom: { global: $wisdom_global, local: $wisdom_local },
    signals: {
      signal_count: ($signals.signal_count // 0),
      instinct_count: ($signals.instinct_count // 0),
      active_signals: ($signals.prompt_section // "")
    },
    prompt_section: $prompt,
    log_line: $log,
    trimmed_notice: $trimmed_notice,
    trimmed_high_priority: $trimmed_high_priority
  }')

# Validate result
if [[ -z "$cp_result" ]] || ! echo "$cp_result" | jq -e . >/dev/null 2>&1; then  # SUPPRESS:OK -- validation: testing JSON validity
  json_err "$E_JSON_INVALID" \
    "Couldn't assemble colony-prime output" \
    '{"error":"assembly_failed"}'
fi

json_ok "$cp_result"
}

# ============================================================================
# _cache_read / _cache_write
# Cache helpers for pr-context -- TTL-based with mtime validation
# ============================================================================
_cache_read() {
  local _cr_name="$1"
  local _cr_path="$2"
  local _cr_ttl="$3"
  local _cr_cache_file="${COLONY_DATA_DIR:-$DATA_DIR}/pr-context-cache.json"

  if [[ ! -f "$_cr_cache_file" ]]; then
    echo "null"
    return 0
  fi

  # Get source file mtime
  local _cr_mtime
  _cr_mtime=$(stat -f "%m" "$_cr_path" 2>/dev/null || stat -c "%Y" "$_cr_path" 2>/dev/null || echo "0")

  # Get cached entry
  local _cr_entry
  _cr_entry=$(jq -r --arg name "$_cr_name" '.[$name] // null' "$_cr_cache_file" 2>/dev/null)

  if [[ "$_cr_entry" == "null" || -z "$_cr_entry" ]]; then
    echo "null"
    return 0
  fi

  # Check mtime match
  local _cr_cached_mtime
  _cr_cached_mtime=$(echo "$_cr_entry" | jq -r '.mtime // 0' 2>/dev/null)
  if [[ "$_cr_cached_mtime" != "$_cr_mtime" ]]; then
    echo "null"
    return 0
  fi

  # Check TTL
  local _cr_cached_at
  _cr_cached_at=$(echo "$_cr_entry" | jq -r '.cached_at // 0' 2>/dev/null)
  local _cr_now
  _cr_now=$(date +%s)
  local _cr_age=$(( _cr_now - _cr_cached_at ))
  if [[ "$_cr_age" -gt "$_cr_ttl" ]]; then
    echo "null"
    return 0
  fi

  # Cache hit -- return the data
  echo "$_cr_entry" | jq -r '.data'
}

_cache_write() {
  local _cw_name="$1"
  local _cw_path="$2"
  local _cw_data="$3"
  local _cw_cache_file="${COLONY_DATA_DIR:-$DATA_DIR}/pr-context-cache.json"

  # Ensure directory exists
  mkdir -p "$(dirname "$_cw_cache_file")" 2>/dev/null || true

  # Get source file mtime
  local _cw_mtime
  _cw_mtime=$(stat -f "%m" "$_cw_path" 2>/dev/null || stat -c "%Y" "$_cw_path" 2>/dev/null || echo "0")

  local _cw_now
  _cw_now=$(date +%s)

  # Build new entry
  local _cw_entry
  _cw_entry=$(jq -n \
    --arg path "$_cw_path" \
    --arg mtime "$_cw_mtime" \
    --argjson cached_at "$_cw_now" \
    --argjson data "$_cw_data" \
    '{path: $path, mtime: $mtime, cached_at: $cached_at, data: $data}')

  # Merge into cache file
  if [[ ! -f "$_cw_cache_file" ]]; then
    echo "{}" > "$_cw_cache_file"
  fi

  # Use atomic write pattern
  local _cw_tmp="${_cw_cache_file}.tmp.$$"
  jq --arg name "$_cw_name" --argjson entry "$_cw_entry" \
    '.[$name] = $entry' "$_cw_cache_file" > "$_cw_tmp" 2>/dev/null && \
    mv "$_cw_tmp" "$_cw_cache_file" 2>/dev/null || \
    rm -f "$_cw_tmp" 2>/dev/null
}

# ============================================================================
# _pr_context
# Generate CI-ready colony context as structured JSON.
# Soft-fails on every missing source. Uses cache for stable sources.
# ============================================================================
_pr_context() {
pc_compact=false
pc_branch=""
pc_ci_run_id=""

# Parse flags
while [[ $# -gt 0 ]]; do
  case "$1" in
    --compact) pc_compact=true; shift ;;
    --branch) pc_branch="${2:-}"; shift 2 ;;
    --ci-run-id) pc_ci_run_id="${2:-}"; shift 2 ;;
    *) shift ;;
  esac
done

# Defaults
pc_max_chars=6000
if [[ "$pc_compact" == "true" ]]; then
  pc_max_chars=3000
fi

if [[ -z "$pc_branch" ]]; then
  pc_branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
fi

pc_generated_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
pc_warnings=()
pc_fallbacks=()
pc_cache_status='{}'

# === Section: queen (global) ===
pc_queen_global_file="$HOME/.aether/QUEEN.md"
pc_queen_global_data="{}"
if [[ -f "$pc_queen_global_file" ]]; then
  pc_queen_global_cached=$(_cache_read "queen_global" "$pc_queen_global_file" 3600)
  if [[ "$pc_queen_global_cached" != "null" && -n "$pc_queen_global_cached" ]]; then
    pc_queen_global_data="$pc_queen_global_cached"
    pc_cache_status=$(echo "$pc_cache_status" | jq --arg s "cached" '.queen_global = $s' 2>/dev/null || echo '{}')
  else
    pc_queen_global_data=$(_extract_wisdom "$pc_queen_global_file" 2>/dev/null || echo '{}')
    if [[ -z "$pc_queen_global_data" || "$pc_queen_global_data" == "null" ]]; then
      pc_queen_global_data="{}"
    fi
    _cache_write "queen_global" "$pc_queen_global_file" "$pc_queen_global_data" 2>/dev/null || true
    pc_cache_status=$(echo "$pc_cache_status" | jq --arg s "fresh" '.queen_global = $s' 2>/dev/null || echo '{}')
  fi
else
  pc_fallbacks+=("queen_global: no file found")
  pc_cache_status=$(echo "$pc_cache_status" | jq --arg s "missing" '.queen_global = $s' 2>/dev/null || echo '{}')
fi

# === Section: queen (local) ===
pc_queen_local_file="$AETHER_ROOT/.aether/QUEEN.md"
pc_queen_local_data="{}"
if [[ -f "$pc_queen_local_file" ]]; then
  pc_queen_local_cached=$(_cache_read "queen_local" "$pc_queen_local_file" 3600)
  if [[ "$pc_queen_local_cached" != "null" && -n "$pc_queen_local_cached" ]]; then
    pc_queen_local_data="$pc_queen_local_cached"
    pc_cache_status=$(echo "$pc_cache_status" | jq --arg s "cached" '.queen_local = $s' 2>/dev/null || echo '{}')
  else
    pc_queen_local_data=$(_extract_wisdom "$pc_queen_local_file" 2>/dev/null || echo '{}')
    if [[ -z "$pc_queen_local_data" || "$pc_queen_local_data" == "null" ]]; then
      pc_queen_local_data="{}"
    fi
    _cache_write "queen_local" "$pc_queen_local_file" "$pc_queen_local_data" 2>/dev/null || true
    pc_cache_status=$(echo "$pc_cache_status" | jq --arg s "fresh" '.queen_local = $s' 2>/dev/null || echo '{}')
  fi
else
  pc_fallbacks+=("queen_local: no file found")
  pc_cache_status=$(echo "$pc_cache_status" | jq --arg s "missing" '.queen_local = $s' 2>/dev/null || echo '{}')
fi

# === Section: user_preferences ===
pc_user_prefs='[]'
# Extract from global queen wisdom
pc_up_raw=$(echo "$pc_queen_global_data" | jq -r '.user_prefs // ""' 2>/dev/null)
if [[ -n "$pc_up_raw" && "$pc_up_raw" != "null" ]]; then
  pc_user_prefs=$(echo "$pc_up_raw" | jq -R -s 'split("\n") | map(select(length > 0))' 2>/dev/null || echo '[]')
fi
pc_up_local=$(echo "$pc_queen_local_data" | jq -r '.user_prefs // ""' 2>/dev/null)
if [[ -n "$pc_up_local" && "$pc_up_local" != "null" ]]; then
  pc_user_prefs_local=$(echo "$pc_up_local" | jq -R -s 'split("\n") | map(select(length > 0))' 2>/dev/null || echo '[]')
  pc_user_prefs=$(echo "$pc_user_prefs" | jq --argjson local "$pc_user_prefs_local" '. + $local' 2>/dev/null || echo "$pc_user_prefs")
fi

# === Section: signals ===
pc_pher_file="${COLONY_DATA_DIR:-$DATA_DIR}/pheromones.json"
pc_state_file="$DATA_DIR/COLONY_STATE.json"
pc_signals_count=0
pc_redirects='[]'
pc_focus='[]'
pc_feedback='[]'
pc_instincts='[]'

if [[ -f "$pc_pher_file" ]]; then
  # Read and classify signals
  pc_signals_json=$(jq -r '.signals // []' "$pc_pher_file" 2>/dev/null || echo '[]')
  if [[ -n "$pc_signals_json" && "$pc_signals_json" != "null" ]]; then
    pc_signals_count=$(echo "$pc_signals_json" | jq 'length' 2>/dev/null || echo 0)
    pc_redirects=$(echo "$pc_signals_json" | jq '[.[] | select(.type == "REDIRECT")]' 2>/dev/null || echo '[]')
    pc_focus=$(echo "$pc_signals_json" | jq '[.[] | select(.type == "FOCUS")]' 2>/dev/null || echo '[]')
    pc_feedback=$(echo "$pc_signals_json" | jq '[.[] | select(.type == "FEEDBACK")]' 2>/dev/null || echo '[]')
  fi
else
  pc_fallbacks+=("pheromones: no active signals")
fi

# Read instincts from COLONY_STATE.json
if [[ -f "$pc_state_file" ]]; then
  pc_instincts=$(jq -r '.memory.instincts // []' "$pc_state_file" 2>/dev/null || echo '[]')
  if [[ -z "$pc_instincts" || "$pc_instincts" == "null" ]]; then
    pc_instincts='[]'
  fi
fi

# === Section: hive_wisdom ===
pc_hive_data='[]'
pc_hive_source="empty"
pc_hive_file="$HOME/.aether/hive/wisdom.json"
pc_eternal_file="$HOME/.aether/eternal/memory.json"

# Try hive first (via subcommand invocation for proper domain scoping)
pc_hive_raw=$(bash "$SCRIPT_DIR/aether-utils.sh" hive-read --limit 5 --min-confidence 0.5 --format json 2>/dev/null || echo '')
if [[ -n "$pc_hive_raw" ]]; then
  pc_hive_entries=$(echo "$pc_hive_raw" | jq -r '.result.entries // []' 2>/dev/null)
  if [[ -n "$pc_hive_entries" && "$pc_hive_entries" != "null" && "$pc_hive_entries" != "[]" ]]; then
    pc_hive_data="$pc_hive_entries"
    pc_hive_source="hive"
    pc_cache_status=$(echo "$pc_cache_status" | jq --arg s "fresh" '.hive = $s' 2>/dev/null || echo '{}')
  fi
fi

# Fallback to eternal memory
if [[ "$pc_hive_source" == "empty" && -f "$pc_eternal_file" ]]; then
  pc_eternal_cached=$(_cache_read "eternal" "$pc_eternal_file" 7200)
  if [[ "$pc_eternal_cached" != "null" && -n "$pc_eternal_cached" ]]; then
    pc_hive_data="$pc_eternal_cached"
    pc_hive_source="eternal"
    pc_cache_status=$(echo "$pc_cache_status" | jq --arg s "cached" '.hive = $s' 2>/dev/null || echo '{}')
  else
    pc_eternal_raw=$(jq -r '.entries // []' "$pc_eternal_file" 2>/dev/null || echo '[]')
    if [[ -n "$pc_eternal_raw" && "$pc_eternal_raw" != "null" && "$pc_eternal_raw" != "[]" ]]; then
      pc_hive_data="$pc_eternal_raw"
      pc_hive_source="eternal"
      _cache_write "eternal" "$pc_eternal_file" "$pc_eternal_raw" 2>/dev/null || true
      pc_cache_status=$(echo "$pc_cache_status" | jq --arg s "fresh" '.hive = $s' 2>/dev/null || echo '{}')
    fi
  fi
fi

if [[ "$pc_hive_source" == "empty" ]]; then
  pc_fallbacks+=("hive_wisdom: no hive or eternal data")
  pc_cache_status=$(echo "$pc_cache_status" | jq --arg s "missing" '.hive = $s' 2>/dev/null || echo '{}')
fi

# === Section: colony_state ===
pc_cs_exists=false
pc_cs_goal="No goal set"
pc_cs_state="UNKNOWN"
pc_cs_current_phase=0
pc_cs_total_phases=0
pc_cs_phase_name=""
if [[ -f "$pc_state_file" ]]; then
  pc_cs_parsed=$(jq -r '{goal: .goal, state: .state, current_phase: .current_phase, total_phases: (.total_phases // (.plan.phases | length // 0)), phase_name: .phase_name}' "$pc_state_file" 2>/dev/null)
  if [[ -n "$pc_cs_parsed" && "$pc_cs_parsed" != "null" ]]; then
    pc_cs_exists=true
    pc_cs_goal=$(echo "$pc_cs_parsed" | jq -r '.goal // "No goal set"' 2>/dev/null)
    pc_cs_state=$(echo "$pc_cs_parsed" | jq -r '.state // "UNKNOWN"' 2>/dev/null)
    pc_cs_current_phase=$(echo "$pc_cs_parsed" | jq -r '.current_phase // 0' 2>/dev/null)
    pc_cs_total_phases=$(echo "$pc_cs_parsed" | jq -r '.total_phases // 0' 2>/dev/null)
    pc_cs_phase_name=$(echo "$pc_cs_parsed" | jq -r '.phase_name // ""' 2>/dev/null)
  else
    pc_fallbacks+=("colony_state: COLONY_STATE.json corrupt")
  fi
else
  pc_fallbacks+=("colony_state: COLONY_STATE.json missing")
fi

# === Section: blockers ===
pc_flags_file="${COLONY_DATA_DIR:-$DATA_DIR}/flags.json"
pc_blockers_count=0
pc_blockers_items='[]'
if [[ -f "$pc_flags_file" ]]; then
  pc_blockers_items=$(jq -r '[.flags // [] | .[] | select((.resolved // false) != true and ((.type // "") == "blocker" or (.severity // "") == "CRITICAL"))]' "$pc_flags_file" 2>/dev/null || echo '[]')
  pc_blockers_count=$(echo "$pc_blockers_items" | jq 'length' 2>/dev/null || echo 0)
else
  pc_fallbacks+=("blockers: flags.json missing")
fi

# === Section: decisions ===
pc_decisions_count=0
pc_decisions_items='[]'
pc_context_file="$AETHER_ROOT/.aether/CONTEXT.md"
if [[ -f "$pc_context_file" ]]; then
  # Extract decisions from CONTEXT.md if present
  pc_decisions_items=$(jq -r '.memory.decisions // []' "$pc_state_file" 2>/dev/null || echo '[]')
  if [[ -n "$pc_decisions_items" && "$pc_decisions_items" != "null" && "$pc_decisions_items" != "[]" ]]; then
    pc_decisions_count=$(echo "$pc_decisions_items" | jq 'length' 2>/dev/null || echo 0)
  fi
elif [[ -f "$pc_state_file" ]]; then
  pc_decisions_items=$(jq -r '.memory.decisions // []' "$pc_state_file" 2>/dev/null || echo '[]')
  if [[ -n "$pc_decisions_items" && "$pc_decisions_items" != "null" ]]; then
    pc_decisions_count=$(echo "$pc_decisions_items" | jq 'length' 2>/dev/null || echo 0)
  fi
fi

# === Section: rolling_summary ===
pc_roll_file="${COLONY_DATA_DIR:-$DATA_DIR}/rolling-summary.log"
pc_rolling=""
if [[ -f "$pc_roll_file" ]]; then
  pc_rolling=$(tail -n 20 "$pc_roll_file" 2>/dev/null | head -20 || echo "")
fi

# === Section: midden ===
pc_midden_file="${COLONY_DATA_DIR:-$DATA_DIR}/midden/midden.json"
pc_midden_count=0
pc_midden_entries='[]'
pc_midden_cross_pr='{}'

if [[ -f "$pc_midden_file" ]]; then
  # Bound: entries from last 7 days, cap at 10
  local now_epoch
  now_epoch=$(date +%s)
  local seven_days_ago=$(( now_epoch - 604800 ))
  pc_midden_entries=$(jq -r --argjson cutoff "$seven_days_ago" --argjson max 10 '
    [.entries // [] | .[] |
      # Parse occurred_at to epoch (best-effort)
      (.occurred_at // .timestamp // "") as $ts |
      ($ts | split("T")) as $parts |
      if ($parts | length) > 1 then
        ($parts[0] | split("-")) as $d |
        ($parts[1] | rtrimstr("Z") | split(":")) as $t |
        (($d[0] // "0" | tonumber) - 1970) * 365 * 86400 +
        (($d[1] // "0" | tonumber) - 1) * 30 * 86400 +
        (($d[2] // "0" | tonumber) - 1) * 86400 +
        (($t[0] // "0" | tonumber) * 3600) +
        (($t[1] // "0" | tonumber) * 60) +
        (($t[2] // "0" | rtrimstr("Z") | tonumber) // 0) as $epoch |
        . + {_epoch: $epoch}
      else . + {_epoch: 0} end
    ] | sort_by(-._epoch) | .[:$max] |
    map(del(._epoch) | .description = ((.description // "")[0:160]))
  ' "$pc_midden_file" 2>/dev/null || echo '[]')
  pc_midden_count=$(echo "$pc_midden_entries" | jq 'length' 2>/dev/null || echo 0)
else
  pc_fallbacks+=("midden: midden.json missing")
fi

# === Section: context_capsule ===
pc_capsule_data='{}'
pc_capsule_raw=$(bash "$SCRIPT_DIR/aether-utils.sh" context-capsule --json 2>/dev/null || echo '')
if [[ -n "$pc_capsule_raw" ]]; then
  pc_capsule_data=$(echo "$pc_capsule_raw" | jq -r '.result // . // {}' 2>/dev/null || echo '{}')
fi

# === Section: phase_learnings ===
pc_learnings=""
if [[ -f "$pc_state_file" ]]; then
  pc_learnings=$(jq -r '.memory.phase_learnings // [] | map(if type == "object" then (.summary // .description // tostring) else tostring end) | .[]' "$pc_state_file" 2>/dev/null | head -20 || echo "")
fi

# === Build prompt_section (text version) ===
pc_sec_queen_global=""
pc_sec_queen_local=""
pc_sec_user_prefs=""
pc_sec_hive=""
pc_sec_capsule=""
pc_sec_learnings=""
pc_sec_decisions=""
pc_sec_blockers=""
pc_sec_rolling=""
pc_sec_signals=""

# QUEEN global section
local _pc_qg_raw=""
if [[ -f "$pc_queen_global_file" ]]; then
  _pc_qg_raw=$(echo "$pc_queen_global_data" | jq -r 'to_entries | map("\(.key): \(.value)") | .[]' 2>/dev/null)
fi
if [[ -n "$_pc_qg_raw" ]]; then
  pc_sec_queen_global=$'\n'"--- QUEEN WISDOM (Global) ---"$'\n'"$_pc_qg_raw"$'\n'
fi

# QUEEN local section
local _pc_ql_raw=""
if [[ -f "$pc_queen_local_file" ]]; then
  _pc_ql_raw=$(echo "$pc_queen_local_data" | jq -r 'to_entries | map("\(.key): \(.value)") | .[]' 2>/dev/null)
fi
if [[ -n "$_pc_ql_raw" ]]; then
  pc_sec_queen_local=$'\n'"--- QUEEN WISDOM (Local) ---"$'\n'"$_pc_ql_raw"$'\n'
fi

# User preferences
if [[ "$(echo "$pc_user_prefs" | jq 'length' 2>/dev/null)" -gt 0 ]]; then
  pc_sec_user_prefs=$'\n'"--- USER PREFERENCES ---"$'\n'
  pc_sec_user_prefs+=$(echo "$pc_user_prefs" | jq -r '.[]' 2>/dev/null | while IFS= read -r line; do echo "- $line"; done)
  pc_sec_user_prefs+=$'\n'
fi

# Signals section
if [[ "$pc_signals_count" -gt 0 ]]; then
  pc_sec_signals=$'\n'"--- ACTIVE SIGNALS (Colony Guidance) ---"$'\n'
  local _pc_redirects_text=""
  _pc_redirects_text=$(echo "$pc_redirects" | jq -r '.[] | "REDIRECT (HARD CONSTRAINT): " + (.content.text // (.content | if type == "string" then . else "" end))' 2>/dev/null)
  if [[ -n "$_pc_redirects_text" ]]; then
    pc_sec_signals+=$'\n'"REDIRECT (HARD CONSTRAINTS):"$'\n'
    while IFS= read -r line; do [[ -n "$line" ]] && pc_sec_signals+="- $line"$'\n'; done <<< "$_pc_redirects_text"
  fi
  local _pc_focus_text=""
  _pc_focus_text=$(echo "$pc_focus" | jq -r '.[] | "FOCUS: " + (.content.text // (.content | if type == "string" then . else "" end))' 2>/dev/null)
  if [[ -n "$_pc_focus_text" ]]; then
    pc_sec_signals+=$'\n'"FOCUS (Active Guidance):"$'\n'
    while IFS= read -r line; do [[ -n "$line" ]] && pc_sec_signals+="- $line"$'\n'; done <<< "$_pc_focus_text"
  fi
  local _pc_feedback_text=""
  _pc_feedback_text=$(echo "$pc_feedback" | jq -r '.[] | "FEEDBACK: " + (.content.text // (.content | if type == "string" then . else "" end))' 2>/dev/null)
  if [[ -n "$_pc_feedback_text" ]]; then
    pc_sec_signals+=$'\n'"FEEDBACK (Adjustments):"$'\n'
    while IFS= read -r line; do [[ -n "$line" ]] && pc_sec_signals+="- $line"$'\n'; done <<< "$_pc_feedback_text"
  fi
  pc_sec_signals+=$'\n'"--- END SIGNALS ---"$'\n'
fi

# Hive wisdom
local _pc_hive_count=0
_pc_hive_count=$(echo "$pc_hive_data" | jq 'length' 2>/dev/null || echo 0)
if [[ "$_pc_hive_count" -gt 0 ]]; then
  pc_sec_hive=$'\n'"--- HIVE WISDOM (Cross-Colony Patterns) ---"$'\n'
  pc_sec_hive+=$(echo "$pc_hive_data" | jq -r '.[] | "- " + (.wisdom // .text // (. | tostring))' 2>/dev/null | head -10)
  pc_sec_hive+=$'\n'
fi

# Context capsule
local _pc_capsule_text=""
_pc_capsule_text=$(echo "$pc_capsule_data" | jq -r '.prompt_section // ""' 2>/dev/null)
if [[ -n "$_pc_capsule_text" ]]; then
  pc_sec_capsule=$'\n'"$_pc_capsule_text"$'\n'
fi

# Phase learnings
if [[ -n "$pc_learnings" ]]; then
  pc_sec_learnings=$'\n'"--- PHASE LEARNINGS ---"$'\n'"$pc_learnings"$'\n'
fi

# Decisions
if [[ "$pc_decisions_count" -gt 0 ]]; then
  pc_sec_decisions=$'\n'"--- KEY DECISIONS ---"$'\n'
  pc_sec_decisions+=$(echo "$pc_decisions_items" | jq -r '.[] | if type == "object" then "- " + (.decision // .summary // .description // tostring) else "- " + tostring end' 2>/dev/null | head -10)
  pc_sec_decisions+=$'\n'
fi

# Blockers
if [[ "$pc_blockers_count" -gt 0 ]]; then
  pc_sec_blockers=$'\n'"--- BLOCKERS (CRITICAL) ---"$'\n'
  pc_sec_blockers+=$(echo "$pc_blockers_items" | jq -r '.[] | "- " + (.title // .description // tostring)' 2>/dev/null | head -10)
  pc_sec_blockers+=$'\n'
fi

# Rolling summary
if [[ -n "$pc_rolling" ]]; then
  pc_sec_rolling=$'\n'"--- ROLLING SUMMARY ---"$'\n'"$pc_rolling"$'\n'
fi

# === Budget enforcement ===
_budget_enforce "pc_"

# Trim notification
local pc_trimmed_sections=""
if [[ -n "${pc_budget_trimmed_list:-}" ]]; then
  pc_trimmed_sections=$(echo "$pc_budget_trimmed_list" | tr ',' ', ')
fi

# === Build output JSON ===
# Build fallbacks JSON array
local pc_fallbacks_json='[]'
local fb
for fb in ${pc_fallbacks[@]+"${pc_fallbacks[@]}"}; do
  pc_fallbacks_json=$(echo "$pc_fallbacks_json" | jq --arg f "$fb" '. + [$f]' 2>/dev/null || echo '[]')
done

# Build warnings JSON array
local pc_warnings_json='[]'
local w
for w in ${pc_warnings[@]+"${pc_warnings[@]}"}; do
  pc_warnings_json=$(echo "$pc_warnings_json" | jq --arg f "$w" '. + [$f]' 2>/dev/null || echo '[]')
done

# Trimmed sections JSON array
local pc_trimmed_json='[]'
if [[ -n "${pc_budget_trimmed_list:-}" ]]; then
  pc_trimmed_json=$(echo "$pc_budget_trimmed_list" | jq -R 'split(",")' 2>/dev/null || echo '[]')
fi

# Escape prompt_section for JSON
local pc_prompt_json
pc_prompt_json=$(printf '%s' "$pc_final_prompt" | jq -Rs '.' 2>/dev/null || echo '""')

# Colony state JSON
local pc_colony_state_json
pc_colony_state_json=$(jq -n \
  --argjson exists "$pc_cs_exists" \
  --arg goal "$pc_cs_goal" \
  --arg state "$pc_cs_state" \
  --argjson current_phase "$pc_cs_current_phase" \
  --argjson total_phases "$pc_cs_total_phases" \
  --arg phase_name "$pc_cs_phase_name" \
  '{exists: $exists, goal: $goal, state: $state, current_phase: $current_phase, total_phases: $total_phases, phase_name: $phase_name}')

# Build result
local pc_result
pc_result=$(jq -n \
  --arg schema "pr-context-v1" \
  --arg generated_at "$pc_generated_at" \
  --arg branch "$pc_branch" \
  --argjson cache_status "$pc_cache_status" \
  --argjson queen_global "$pc_queen_global_data" \
  --argjson queen_local "$pc_queen_local_data" \
  --argjson combined_prefs "$pc_user_prefs" \
  --argjson signals_count "$pc_signals_count" \
  --argjson redirects "$pc_redirects" \
  --argjson focus "$pc_focus" \
  --argjson feedback "$pc_feedback" \
  --argjson instincts "$pc_instincts" \
  --arg hive_source "$pc_hive_source" \
  --argjson hive_count "$(echo "$pc_hive_data" | jq 'length' 2>/dev/null || echo 0)" \
  --argjson hive_entries "$pc_hive_data" \
  --argjson colony_state "$pc_colony_state_json" \
  --argjson blockers_count "$pc_blockers_count" \
  --argjson blockers_items "$pc_blockers_items" \
  --argjson decisions_count "$pc_decisions_count" \
  --argjson decisions_items "$pc_decisions_items" \
  --argjson midden_count "$pc_midden_count" \
  --argjson midden_entries "$pc_midden_entries" \
  --argjson midden_cross_pr "$pc_midden_cross_pr" \
  --argjson prompt_section "$pc_prompt_json" \
  --argjson char_count "${#pc_final_prompt}" \
  --argjson budget "$pc_max_chars" \
  --argjson trimmed_sections "$pc_trimmed_json" \
  --argjson warnings "$pc_warnings_json" \
  --argjson fallbacks_used "$pc_fallbacks_json" \
  '{
    schema: $schema,
    generated_at: $generated_at,
    branch: $branch,
    cache_status: $cache_status,
    queen: {global: $queen_global, local: $queen_local, combined_prefs: $combined_prefs},
    signals: {count: $signals_count, redirects: $redirects, focus: $focus, feedback: $feedback, instincts: $instincts},
    hive: {source: $hive_source, count: $hive_count, entries: $hive_entries},
    colony_state: $colony_state,
    blockers: {count: $blockers_count, items: $blockers_items},
    decisions: {count: $decisions_count, items: $decisions_items},
    midden: {count: $midden_count, entries: $midden_entries, cross_pr_analysis: $midden_cross_pr},
    prompt_section: $prompt_section,
    char_count: $char_count,
    budget: $budget,
    trimmed_sections: $trimmed_sections,
    warnings: $warnings,
    fallbacks_used: $fallbacks_used
  }')

# Validate result
if [[ -z "$pc_result" ]] || ! echo "$pc_result" | jq -e . >/dev/null 2>&1; then
  json_err "$E_JSON_INVALID" \
    "Couldn't assemble pr-context output" \
    '{"error":"assembly_failed"}'
fi

json_ok "$pc_result"
}

# ============================================================================
# _pheromone_expire
# Archive expired pheromone signals to midden
# ============================================================================
_pheromone_expire() {
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

phe_pheromones_file="$COLONY_DATA_DIR/pheromones.json"
phe_midden_dir="$COLONY_DATA_DIR/midden"
phe_midden_file="$phe_midden_dir/midden.json"

# Handle missing pheromones.json gracefully
if [[ ! -f "$phe_pheromones_file" ]]; then
  json_ok '{"expired_count":0,"remaining_active":0,"midden_total":0}'
  exit 0
fi

# Ensure midden directory and file exist
mkdir -p "$phe_midden_dir"
if [[ ! -f "$phe_midden_file" ]]; then
  atomic_write "$phe_midden_file" '{"version":"1.0.0","archived_at_count":0,"signals":[]}' || {
    _aether_log_error "Could not initialize midden archive file"
    json_err "$E_UNKNOWN" "Failed to create midden archive file"
  }
fi

phe_now_iso=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
phe_archived_at="$phe_now_iso"

# MIGRATE: direct COLONY_STATE.json access -- use _state_read_field instead
# Compute pause_duration from COLONY_STATE.json (pause-aware TTL)
phe_pause_duration=0
if [[ -f "$DATA_DIR/COLONY_STATE.json" ]]; then
  phe_paused_at=$(jq -r '.paused_at // empty' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null || true)  # SUPPRESS:OK -- read-default: file may not exist yet
  phe_resumed_at=$(jq -r '.resumed_at // empty' "$DATA_DIR/COLONY_STATE.json" 2>/dev/null || true)  # SUPPRESS:OK -- read-default: file may not exist yet
  if [[ -n "$phe_paused_at" && -n "$phe_resumed_at" ]]; then
    # SUPPRESS:OK -- cross-platform: macOS vs Linux date/stat flags
    phe_paused_epoch=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "$phe_paused_at" +%s 2>/dev/null || date -d "$phe_paused_at" +%s 2>/dev/null || echo 0)
    # SUPPRESS:OK -- cross-platform: macOS vs Linux date/stat flags
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
  # SUPPRESS:OK -- read-default: query may return empty
  phe_expired_ids=$(jq -r '.signals[] | select(.active == true and .expires_at == "phase_end") | .id' "$phe_pheromones_file" 2>/dev/null || true)
else
  # Expire time-based expired signals (pause-aware) AND decay-expired signals
  phe_expired_ids=$(jq -r --arg now_iso "$phe_now_iso" --argjson pause_secs "$phe_pause_duration" '
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
    (to_epoch($now_iso)) as $now |
    .signals[] |
    select(.active == true) |
    select(
      (.expires_at != "phase_end" and .expires_at != null and .expires_at != "") and
      (
        (to_epoch(.expires_at)) + $pause_secs <= $now
      )
    ) |
    .id
  ' "$phe_pheromones_file" 2>/dev/null || true)  # SUPPRESS:OK -- read-default: file may not exist yet
fi

# Count expired signals
phe_expired_count=0
if [[ -n "$phe_expired_ids" ]]; then
  phe_expired_count=$(echo "$phe_expired_ids" | grep -c . 2>/dev/null || echo 0)  # SUPPRESS:OK -- read-default: count defaults to 0 if file missing
fi

# If nothing to expire, return counts
if [[ "$phe_expired_count" -eq 0 ]]; then
  # SUPPRESS:OK -- read-default: query may return empty
  phe_remaining=$(jq '[.signals[] | select(.active == true)] | length' "$phe_pheromones_file" 2>/dev/null || echo 0)
  phe_midden_total=$(jq '.signals | length' "$phe_midden_file" 2>/dev/null || echo 0)  # SUPPRESS:OK -- read-default: file may not exist yet
  json_ok "{\"expired_count\":0,\"remaining_active\":$phe_remaining,\"midden_total\":$phe_midden_total}"
  exit 0
fi

# Build jq args for IDs to expire
phe_id_array=$(echo "$phe_expired_ids" | jq -R . | jq -s . 2>/dev/null || echo '[]')  # SUPPRESS:OK -- read-default: returns fallback if missing

# Extract expired signal objects (with archived_at added)
phe_expired_objects=$(jq --argjson ids "$phe_id_array" --arg archived_at "$phe_archived_at" '
  [.signals[] | select(.id as $id | $ids | any(. == $id)) | . + {"archived_at": $archived_at, "active": false}]
' "$phe_pheromones_file" 2>/dev/null || echo '[]')  # SUPPRESS:OK -- read-default: file may not exist yet

# Promote high-value expired signals to eternal memory before archival.
# Use decayed effective_strength (not raw .strength) for promotion threshold.
phe_eternal_promoted=0
while IFS= read -r phe_signal; do
  [[ -z "$phe_signal" ]] && continue
  phe_strength_int=$(echo "$phe_signal" | jq -r --arg now_iso "$phe_now_iso" '
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
      if t == "FOCUS" then 30
      elif t == "REDIRECT" then 60
      else 90
      end;
    (to_epoch($now_iso)) as $now |
    (to_epoch(.created_at)) as $created |
    (if $created != null then ($now - $created) / 86400 else 0 end) as $elapsed |
    (decay_days(.type // "FEEDBACK")) as $dd |
    ((.strength // 0) * (1 - ($elapsed / $dd))) as $eff_raw |
    (if $eff_raw < 0 then 0 else $eff_raw end) as $eff |
    (($eff * 100) | floor)
  ' 2>/dev/null || echo "0")  # SUPPRESS:OK -- read-default: returns fallback on failure
  if [[ "$phe_strength_int" -gt 80 ]]; then
    phe_text=$(sanitize_read_value "$(echo "$phe_signal" | jq -r '.content.text // ""' 2>/dev/null || echo "")")  # SUPPRESS:OK -- read-default: file may not exist yet
    phe_type=$(echo "$phe_signal" | jq -r '.type // "UNKNOWN"' 2>/dev/null || echo "UNKNOWN")  # SUPPRESS:OK -- read-default: file may not exist yet
    phe_source=$(echo "$phe_signal" | jq -r '.source // "unknown"' 2>/dev/null || echo "unknown")  # SUPPRESS:OK -- read-default: file may not exist yet
    phe_id=$(echo "$phe_signal" | jq -r '.id // ""' 2>/dev/null || echo "")  # SUPPRESS:OK -- read-default: file may not exist yet
    if [[ -n "$phe_text" ]]; then
      # SUPPRESS:OK -- cleanup: side-effect is best-effort
      if bash "$0" eternal-store "$phe_text" --type "$phe_type" --source "$phe_source" --strength "$(echo "$phe_signal" | jq -r '.strength // 0')" --signal-id "$phe_id" --reason "promoted_on_expire" >/dev/null 2>&1; then
        phe_eternal_promoted=$((phe_eternal_promoted + 1))
      fi
    fi
  fi
done < <(echo "$phe_expired_objects" | jq -c '.[]' 2>/dev/null || true)  # SUPPRESS:OK -- read-default: returns fallback if missing

# Update pheromones.json: set active=false for expired signals (do NOT remove them)
local phe_updated_pheromones
phe_updated_pheromones=$(jq --argjson ids "$phe_id_array" '
  .signals = [.signals[] | if (.id as $id | $ids | any(. == $id)) then .active = false else . end]
' "$phe_pheromones_file") || {
  _aether_log_error "Could not process pheromone expiration update"
}

if [[ -n "$phe_updated_pheromones" && "$phe_updated_pheromones" != "null" ]]; then
  phe_lock_held=false
  if type acquire_lock &>/dev/null; then
    acquire_lock "$phe_pheromones_file" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on pheromones.json"
    phe_lock_held=true
    trap 'release_lock 2>/dev/null || true' EXIT  # SUPPRESS:OK -- cleanup: lock may not be held
  fi
  atomic_write "$phe_pheromones_file" "$phe_updated_pheromones" || {
    [[ "$phe_lock_held" == "true" ]] && release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
    json_err "$E_JSON_INVALID" "Failed to write pheromones.json"
  }
  [[ "$phe_lock_held" == "true" ]] && { release_lock 2>/dev/null || true; trap - EXIT; }  # SUPPRESS:OK -- cleanup: lock may not be held
fi

# Append expired signals to midden.json
local phe_midden_updated
phe_midden_updated=$(jq --argjson new_signals "$phe_expired_objects" '
  .signals += $new_signals |
  .archived_at_count = (.signals | length)
' "$phe_midden_file") || {
  _aether_log_error "Could not process midden archival update"
}

if [[ -n "$phe_midden_updated" && "$phe_midden_updated" != "null" ]]; then
  phe_midden_lock_held=false
  if type acquire_lock &>/dev/null; then
    acquire_lock "$phe_midden_file" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on midden.json"
    phe_midden_lock_held=true
    trap 'release_lock 2>/dev/null || true' EXIT  # SUPPRESS:OK -- cleanup: lock may not be held
  fi
  atomic_write "$phe_midden_file" "$phe_midden_updated" || {
    [[ "$phe_midden_lock_held" == "true" ]] && release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
    json_err "$E_JSON_INVALID" "Failed to write midden.json"
  }
  [[ "$phe_midden_lock_held" == "true" ]] && { release_lock 2>/dev/null || true; trap - EXIT; }  # SUPPRESS:OK -- cleanup: lock may not be held
fi

# SUPPRESS:OK -- read-default: query may return empty
phe_remaining_active=$(jq '[.signals[] | select(.active == true)] | length' "$phe_pheromones_file" 2>/dev/null || echo 0)
phe_midden_total=$(jq '.signals | length' "$phe_midden_file" 2>/dev/null || echo 0)  # SUPPRESS:OK -- read-default: file may not exist yet

json_ok "{\"expired_count\":$phe_expired_count,\"remaining_active\":$phe_remaining_active,\"midden_total\":$phe_midden_total,\"eternal_promoted\":$phe_eternal_promoted}"
}

# ============================================================================
# _eternal_init
# Initialize the ~/.aether/eternal/ directory and memory.json schema
# ============================================================================
_eternal_init() {
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
  local ei_init_content
  ei_init_content=$(printf '%s\n' "{
  \"version\": \"1.0.0\",
  \"created_at\": \"$ei_created_at\",
  \"colonies\": [],
  \"high_value_signals\": [],
  \"cross_session_patterns\": []
}")
  atomic_write "$ei_memory_file" "$ei_init_content" || {
    _aether_log_error "Could not initialize eternal memory file"
    json_err "$E_UNKNOWN" "Failed to create eternal memory file"
  }
fi

json_ok "$(jq -n --arg dir "$ei_eternal_dir" --argjson already_existed "$ei_already_existed" '{dir: $dir, initialized: true, already_existed: $already_existed}')"
}

# ============================================================================
# _eternal_store
# Store a high-value signal in eternal memory.
# ============================================================================
_eternal_store() {
# Store a high-value signal in eternal memory.
# Usage: eternal-store <content> [--type TYPE] [--source SOURCE] [--strength N] [--signal-id ID] [--reason TEXT] [--created-at ISO8601] [--archived-at ISO8601]
es_content="${1:-}"
[[ -z "$es_content" ]] && json_err "$E_VALIDATION_FAILED" "Usage: eternal-store <content> [--type TYPE] [--source SOURCE] [--strength N] [--signal-id ID] [--reason TEXT] [--created-at ISO8601] [--archived-at ISO8601]" '{"missing":"content"}'

es_type="UNKNOWN"
es_source="unknown"
es_strength="0.0"
es_signal_id=""
es_reason="manual_store"
es_created_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
es_archived_at="$es_created_at"

shift
while [[ $# -gt 0 ]]; do
  case "$1" in
    --type) es_type="${2:-UNKNOWN}"; shift 2 ;;
    --source) es_source="${2:-unknown}"; shift 2 ;;
    --strength) es_strength="${2:-0.0}"; shift 2 ;;
    --signal-id) es_signal_id="${2:-}"; shift 2 ;;
    --reason) es_reason="${2:-manual_store}"; shift 2 ;;
    --created-at) es_created_at="${2:-$es_created_at}"; shift 2 ;;
    --archived-at) es_archived_at="${2:-$es_archived_at}"; shift 2 ;;
    *) shift ;;
  esac
done

if ! [[ "$es_strength" =~ ^[0-9]+(\.[0-9]+)?$ ]]; then
  json_err "$E_VALIDATION_FAILED" "Strength must be numeric" "{\"provided\":\"$es_strength\"}"
fi

# SUPPRESS:OK -- cleanup: side-effect is best-effort
bash "$0" eternal-init >/dev/null 2>&1 || json_err "$E_FILE_NOT_FOUND" "Unable to initialize eternal memory"

es_memory_file="$HOME/.aether/eternal/memory.json"
[[ -f "$es_memory_file" ]] || json_err "$E_FILE_NOT_FOUND" "Eternal memory file not found"

if ! jq -e . "$es_memory_file" >/dev/null 2>&1; then  # SUPPRESS:OK -- validation: testing JSON validity
  json_err "$E_JSON_INVALID" "Eternal memory JSON is invalid"
fi

es_entry=$(jq -n \
  --arg content "$es_content" \
  --arg type "$es_type" \
  --arg source "$es_source" \
  --arg signal_id "$es_signal_id" \
  --arg reason "$es_reason" \
  --arg created_at "$es_created_at" \
  --arg archived_at "$es_archived_at" \
  --argjson strength "$es_strength" \
  '{
    content: $content,
    type: $type,
    source: $source,
    signal_id: $signal_id,
    reason: $reason,
    strength: $strength,
    created_at: $created_at,
    archived_at: $archived_at
  }')

es_lock_held=false
if type acquire_lock &>/dev/null; then
  acquire_lock "$es_memory_file" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on eternal memory"
  es_lock_held=true
  # Trap ensures lock release on unexpected exit (json_err calls exit 1)
  trap 'release_lock 2>/dev/null || true' EXIT  # SUPPRESS:OK -- cleanup: lock may not be held
fi

es_updated=$(jq --argjson entry "$es_entry" '
  .high_value_signals = ((.high_value_signals // []) + [$entry]) |
  if (.high_value_signals | length) > 500 then .high_value_signals = .high_value_signals[-500:] else . end |
  .last_updated = $entry.archived_at
' "$es_memory_file" 2>/dev/null) || {  # SUPPRESS:OK -- read-default: file may not exist yet
  [[ "$es_lock_held" == "true" ]] && release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
  json_err "$E_JSON_INVALID" "Failed to update eternal memory"
}

atomic_write "$es_memory_file" "$es_updated" || {
  [[ "$es_lock_held" == "true" ]] && release_lock 2>/dev/null || true  # SUPPRESS:OK -- cleanup: lock may not be held
  json_err "$E_JSON_INVALID" "Failed to write eternal memory"
}

[[ "$es_lock_held" == "true" ]] && { release_lock 2>/dev/null || true; trap - EXIT; }  # SUPPRESS:OK -- cleanup: lock may not be held
json_ok "$(jq -n --arg signal_id "$es_signal_id" --arg type "$es_type" '{stored: true, signal_id: $signal_id, type: $type}')"
}

# ============================================================================
# _pheromone_export_xml
# Export pheromones.json to XML format
# ============================================================================
_pheromone_export_xml() {
# Export pheromones.json to XML format
# Usage: pheromone-export-xml [output_file]
# Default output: .aether/exchange/pheromones.xml

pex_output="${1:-$SCRIPT_DIR/exchange/pheromones.xml}"
pex_pheromones="$COLONY_DATA_DIR/pheromones.json"

# Graceful degradation: check for xmllint
if ! command -v xmllint >/dev/null 2>&1; then
  json_err "$E_FEATURE_UNAVAILABLE" "xmllint is not installed. Try: xcode-select --install on macOS."
fi

# Check pheromones.json exists
if [[ ! -f "$pex_pheromones" ]]; then
  json_err "$E_FILE_NOT_FOUND" "Couldn't find pheromones.json. Try: run /ant:init first."
fi

# Ensure output directory exists
mkdir -p "$(dirname "$pex_output")"

# Source the exchange script
source "$SCRIPT_DIR/exchange/pheromone-xml.sh"

# Call the export function
xml-pheromone-export "$pex_pheromones" "$pex_output"
}

# ============================================================================
# _pheromone_import_xml
# Import pheromone signals from XML into pheromones.json
# ============================================================================
_pheromone_import_xml() {
# Import pheromone signals from XML into pheromones.json
# Usage: pheromone-import-xml <xml_file> [colony_prefix]
# When colony_prefix is provided, imported signal IDs are tagged with "${prefix}:" before merge

pix_xml="${1:-}"
pix_colony_prefix="${2:-}"
pix_pheromones="$COLONY_DATA_DIR/pheromones.json"

if [[ -z "$pix_xml" ]]; then
  json_err "$E_VALIDATION_FAILED" "Missing XML file argument. Try: pheromone-import-xml <xml_file> [colony_prefix]."
fi

if [[ ! -f "$pix_xml" ]]; then
  json_err "$E_FILE_NOT_FOUND" "XML file not found: $pix_xml. Try: check the file path."
fi

# Graceful degradation: check for xmllint
if ! command -v xmllint >/dev/null 2>&1; then
  json_err "$E_FEATURE_UNAVAILABLE" "xmllint is not installed. Try: xcode-select --install on macOS."
fi

# Source the exchange script
source "$SCRIPT_DIR/exchange/pheromone-xml.sh"

# Import XML to get JSON signals
pix_imported=$(xml-pheromone-import "$pix_xml")

# Extract actual signal array from result.json | fromjson | .signals
# (result.signals is an integer count — must unpack result.json to get the array)
# SUPPRESS:OK -- read-default: query may return empty
pix_raw_signals=$(echo "$pix_imported" | jq -r '.result.json // "{}"' | jq -c '.signals // []' 2>/dev/null || echo '[]')

# Apply colony prefix to imported signal IDs (when provided)
# This prevents ID collisions and tags signals with their source colony
if [[ -n "$pix_colony_prefix" ]]; then
  # SUPPRESS:OK -- read-default: returns fallback on failure
  pix_prefixed_signals=$(echo "$pix_raw_signals" | jq --arg prefix "$pix_colony_prefix" '[.[] | .id = ($prefix + ":" + .id)]' 2>/dev/null || echo '[]')
else
  pix_prefixed_signals="$pix_raw_signals"
fi

# If pheromones.json exists, merge; otherwise create
if [[ -f "$pix_pheromones" ]]; then
  # Merge: imported signals first, existing signals last
  # map(last) keeps current colony's version on ID collision — current colony always wins
  pix_merged=$(jq -s --argjson new_signals "$pix_prefixed_signals" '
    .[0] as $existing |
    {
      signals: ([$new_signals[], $existing.signals[]] | group_by(.id) | map(last)),
      version: $existing.version,
      colony_id: $existing.colony_id
    }
  ' "$pix_pheromones" 2>/dev/null)  # SUPPRESS:OK -- read-default: file may not exist yet

  if [[ -n "$pix_merged" ]]; then
    printf '%s\n' "$pix_merged" > "$pix_pheromones"
  fi
fi

pix_count=$(echo "$pix_raw_signals" | jq 'length' 2>/dev/null || echo 0)  # SUPPRESS:OK -- read-default: file may not exist yet
json_ok "$(jq -n --argjson signal_count "$pix_count" --arg source "$pix_xml" '{imported: true, signal_count: $signal_count, source: $source}')"
}

# ============================================================================
# _pheromone_validate_xml
# Validate pheromone XML against XSD schema
# ============================================================================
_pheromone_validate_xml() {
# Validate pheromone XML against XSD schema
# Usage: pheromone-validate-xml <xml_file>

pvx_xml="${1:-}"
pvx_xsd="$SCRIPT_DIR/schemas/pheromone.xsd"

if [[ -z "$pvx_xml" ]]; then
  json_err "$E_VALIDATION_FAILED" "Missing XML file argument. Try: pheromone-validate-xml <xml_file>."
fi

if [[ ! -f "$pvx_xml" ]]; then
  json_err "$E_FILE_NOT_FOUND" "XML file not found: $pvx_xml. Try: check the file path."
fi

# Graceful degradation: check for xmllint
if ! command -v xmllint >/dev/null 2>&1; then
  json_err "$E_FEATURE_UNAVAILABLE" "xmllint is not installed. Try: xcode-select --install on macOS."
fi

# Source the exchange script
source "$SCRIPT_DIR/exchange/pheromone-xml.sh"

# Call validate function
xml-pheromone-validate "$pvx_xml" "$pvx_xsd"
}

