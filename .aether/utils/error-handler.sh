#!/bin/bash
# Aether Colony Error Handler Module
# Structured JSON error handling for bash utilities
#
# Usage: source "$SCRIPT_DIR/utils/error-handler.sh"
#
# Provides consistent error format between Node.js CLI and bash utilities

# --- Error Code Constants (matching Node.js error codes) ---
E_UNKNOWN="E_UNKNOWN"
E_HUB_NOT_FOUND="E_HUB_NOT_FOUND"
E_REPO_NOT_INITIALIZED="E_REPO_NOT_INITIALIZED"
E_FILE_NOT_FOUND="E_FILE_NOT_FOUND"
E_JSON_INVALID="E_JSON_INVALID"
E_LOCK_FAILED="E_LOCK_FAILED"
E_GIT_ERROR="E_GIT_ERROR"
E_VALIDATION_FAILED="E_VALIDATION_FAILED"
E_FEATURE_UNAVAILABLE="E_FEATURE_UNAVAILABLE"
E_BASH_ERROR="E_BASH_ERROR"

# --- Recovery Suggestion Functions (internal, prefixed with _) ---
_recovery_hub_not_found() { echo '"Run: aether install"'; }
_recovery_repo_not_init() { echo '"Run /ant:init in this repo first"'; }
_recovery_file_not_found() { echo '"Check file path and permissions"'; }
_recovery_json_invalid() { echo '"Validate JSON syntax"'; }
_recovery_lock_failed() { echo '"Wait for other operations to complete"'; }
_recovery_git_error() { echo '"Check git status and resolve conflicts"'; }
_recovery_default() { echo 'null'; }

# Get recovery suggestion based on error code
_get_recovery() {
  local code="$1"
  case "$code" in
    "$E_HUB_NOT_FOUND") _recovery_hub_not_found ;;
    "$E_REPO_NOT_INITIALIZED") _recovery_repo_not_init ;;
    "$E_FILE_NOT_FOUND") _recovery_file_not_found ;;
    "$E_JSON_INVALID") _recovery_json_invalid ;;
    "$E_LOCK_FAILED") _recovery_lock_failed ;;
    "$E_GIT_ERROR") _recovery_git_error ;;
    *) _recovery_default ;;
  esac
}

# --- Enhanced json_err function ---
# Signature: json_err [code] [message] [details] [recovery]
# All parameters optional with sensible defaults
json_err() {
  local code="${1:-$E_UNKNOWN}"
  local message="${2:-An unknown error occurred}"
  local details="${3:-null}"
  local recovery="${4:-}"
  local timestamp
  timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

  # Get recovery suggestion if not provided
  if [[ -z "$recovery" ]]; then
    recovery=$(_get_recovery "$code")
  else
    # Escape and quote the recovery string
    recovery=$(echo "$recovery" | sed 's/"/\\"/g' | tr '\n' ' ')
    recovery="\"$recovery\""
  fi

  # Escape message for JSON
  local escaped_message
  escaped_message=$(echo "$message" | sed 's/"/\\"/g' | tr '\n' ' ')

  # Build details JSON
  local details_json
  if [[ "$details" == "null" || -z "$details" ]]; then
    details_json="null"
  else
    details_json="$details"
  fi

  # Output structured JSON to stderr
  printf '{"ok":false,"error":{"code":"%s","message":"%s","details":%s,"recovery":%s,"timestamp":"%s"}}\n' \
    "$code" "$escaped_message" "$details_json" "$recovery" "$timestamp" >&2

  # Log to activity.log (best effort)
  if [[ -n "${DATA_DIR:-}" ]]; then
    echo "[$timestamp] ERROR $code: $escaped_message" >> "$DATA_DIR/activity.log" 2>/dev/null || true
  fi

  exit 1
}

# --- json_warn function for non-fatal warnings ---
# Signature: json_warn [code] [message]
json_warn() {
  local code="${1:-W_UNKNOWN}"
  local message="${2:-Warning}"
  local timestamp
  timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

  # Escape message for JSON
  local escaped_message
  escaped_message=$(echo "$message" | sed 's/"/\\"/g' | tr '\n' ' ')

  # Output warning JSON to stdout (not stderr - this is non-fatal)
  printf '{"ok":true,"warning":{"code":"%s","message":"%s","timestamp":"%s"}}\n' \
    "$code" "$escaped_message" "$timestamp"

  # Log to activity.log (best effort)
  if [[ -n "${DATA_DIR:-}" ]]; then
    echo "[$timestamp] WARN $code: $escaped_message" >> "$DATA_DIR/activity.log" 2>/dev/null || true
  fi
}

# --- error_handler function for trap ERR ---
# Captures: line number, command, exit code
# Usage: trap 'error_handler ${LINENO} "$BASH_COMMAND" $?' ERR
error_handler() {
  local line_num="${1:-unknown}"
  local command="${2:-unknown}"
  local exit_code="${3:-1}"
  local timestamp
  timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

  # Escape command for JSON
  local escaped_command
  escaped_command=$(echo "$command" | sed 's/"/\\"/g' | tr '\n' ' ')

  # Build details JSON
  local details
  details="{\"line\":$line_num,\"command\":\"$escaped_command\",\"exit_code\":$exit_code}"

  # Output structured JSON to stderr
  printf '{"ok":false,"error":{"code":"%s","message":"Bash command failed","details":%s,"recovery":%s,"timestamp":"%s"}}\n' \
    "$E_BASH_ERROR" "$details" "$(_recovery_default)" "$timestamp" >&2

  # Log to activity.log (best effort)
  if [[ -n "${DATA_DIR:-}" ]]; then
    echo "[$timestamp] ERROR $E_BASH_ERROR: Command failed at line $line_num (exit $exit_code)" >> "$DATA_DIR/activity.log" 2>/dev/null || true
  fi

  exit 1
}

# --- Feature flag functions for graceful degradation ---
# Using simple variables for bash 3.2+ compatibility (no associative arrays)

# Track disabled features as colon-separated list: "feature1:reason1|feature2:reason2"
_FEATURES_DISABLED=""

# Enable a feature (remove from disabled list if present)
feature_enable() {
  local name="$1"
  # Remove from disabled list if present
  _FEATURES_DISABLED=$(echo "$_FEATURES_DISABLED" | sed "s/:$name:[^|]*//g" | sed 's/^|//;s/|$//')
}

# Disable a feature with reason
feature_disable() {
  local name="$1"
  local reason="${2:-disabled}"
  # Remove existing entry if present, then add new
  _FEATURES_DISABLED=$(echo "$_FEATURES_DISABLED" | sed "s/:$name:[^|]*//g")
  if [[ -z "$_FEATURES_DISABLED" ]]; then
    _FEATURES_DISABLED=":$name:$reason"
  else
    _FEATURES_DISABLED="${_FEATURES_DISABLED}|:$name:$reason"
  fi
}

# Check if feature is enabled (returns 0 if enabled, 1 if disabled)
feature_enabled() {
  local name="$1"
  if echo "$_FEATURES_DISABLED" | grep -q ":$name:"; then
    return 1
  fi
  return 0
}

# Get reason for feature being disabled
_feature_reason() {
  local name="$1"
  echo "$_FEATURES_DISABLED" | grep -o ":$name:[^|]*" | sed "s/:$name://" || echo "unknown"
}

# Log degradation warning
feature_log_degradation() {
  local name="$1"
  local reason="${2:-}"
  if [[ -z "$reason" ]]; then
    reason=$(_feature_reason "$name")
  fi
  json_warn "W_DEGRADED" "Feature '$name' is disabled: $reason"
}

# --- Export all functions and variables ---
export -f json_err json_warn error_handler
export -f feature_enable feature_disable feature_enabled feature_log_degradation
export -f _get_recovery _recovery_hub_not_found _recovery_repo_not_init
export -f _recovery_file_not_found _recovery_json_invalid _recovery_lock_failed
export -f _recovery_git_error _recovery_default _feature_reason
export E_UNKNOWN E_HUB_NOT_FOUND E_REPO_NOT_INITIALIZED E_FILE_NOT_FOUND
export E_JSON_INVALID E_LOCK_FAILED E_GIT_ERROR E_VALIDATION_FAILED
export E_FEATURE_UNAVAILABLE E_BASH_ERROR
export _FEATURES_DISABLED
