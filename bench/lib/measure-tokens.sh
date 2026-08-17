#!/usr/bin/env bash
# measure-tokens.sh — harness-side token measurement from a Claude Code
# session transcript, never from either system's own self-report.
#
# Sourceable library AND a standalone command: `measure-tokens.sh HOME_DIR`
# prints the JSON usage object for that isolated HOME's transcripts on
# stdout.
#
# On-disk transcript shape (confirmed against a real Claude Code transcript,
# not assumed from the raw-stdout shape pkg/codex/usage.go parses): each
# assistant turn is one JSONL line with "type":"assistant" and a nested
# `.message.usage` object carrying `input_tokens`, `output_tokens`,
# `cache_creation_input_tokens`, `cache_read_input_tokens`. These are the
# on-disk field names — they are NOT the same as the WorkerUsage Go struct's
# field names (InputTokens, CachedInputTokens, ...), so this script maps them
# explicitly rather than assuming the Go names appear in the file.
#
# UNLIKE pkg/codex/usage.go's raw-stdout parser (which takes the LAST event
# because both providers report a cumulative running total over one
# dispatch's stdout), a Claude Code session transcript records one usage
# object PER ASSISTANT TURN, and a run's real cost is the SUM across every
# turn, not the last one. Do not "fix" this into last-one-wins — that would
# undercount every session with more than one assistant turn.
#
# Billed-total arithmetic mirrors pkg/codex/usage.go's billedTotal() exactly:
# total = input_tokens + cache_read_input_tokens + cache_creation_input_tokens
#       + output_tokens
# because Anthropic reports the three input-side counts as DISJOINT, not
# overlapping. This repo already shipped a 186x undercount from summing only
# input + output; that specific mistake (see pkg/codex/usage.go's comment on
# billedTotal) must never be reproduced here.
set -euo pipefail

# measure_tokens HOME_DIR
#
# Emits one JSON object on stdout: total_tokens, total_input_tokens,
# model_calls, models, source, transcript_count, transcript_paths. Nothing
# else goes to stdout — diagnostics go to stderr.
measure_tokens() {
  local home_dir="$1"

  if [ -z "$home_dir" ]; then
    echo "measure_tokens: HOME_DIR argument is required" >&2
    return 1
  fi

  # Defensive check: refuse a HOME_DIR that resolves to the operator's real
  # home directory. Measuring the real home would pool the operator's own
  # unrelated Claude Code sessions into a benchmark lane's token figure,
  # silently making every result meaningless.
  local real_home_env="${REAL_HOME:-}"
  local real_home_node
  real_home_node="$(node -e 'process.stdout.write(require("os").homedir())' 2>/dev/null || true)"

  local home_resolved
  home_resolved="$(cd "$home_dir" 2>/dev/null && pwd || echo "$home_dir")"

  if [ -n "$real_home_env" ]; then
    local real_home_env_resolved
    real_home_env_resolved="$(cd "$real_home_env" 2>/dev/null && pwd || echo "$real_home_env")"
    if [ "$home_resolved" = "$real_home_env_resolved" ]; then
      echo "measure_tokens: refusing to measure the operator's real home (REAL_HOME=$real_home_env) — this would pool the operator's own sessions into a lane's figure" >&2
      return 1
    fi
  fi

  if [ -n "$real_home_node" ]; then
    local real_home_node_resolved
    real_home_node_resolved="$(cd "$real_home_node" 2>/dev/null && pwd || echo "$real_home_node")"
    if [ "$home_resolved" = "$real_home_node_resolved" ]; then
      echo "measure_tokens: refusing to measure the operator's real home (os.homedir()=$real_home_node) — this would pool the operator's own sessions into a lane's figure" >&2
      return 1
    fi
  fi

  # Glob rather than hand-construct the encoded project-directory name — the
  # cwd-to-directory-name encoding is not a documented stable contract in
  # this repo (see 186-PATTERNS.md).
  local -a transcripts=()
  local f
  for f in "$home_dir"/.claude/projects/*/*.jsonl; do
    [ -e "$f" ] && transcripts+=("$f")
  done

  if [ "${#transcripts[@]}" -eq 0 ]; then
    echo "measure_tokens: no transcripts found under $home_dir/.claude/projects/*/*.jsonl — a run that produced no transcript must never silently report zero tokens" >&2
    return 1
  fi

  # Concatenate every matched transcript's JSONL lines into one `jq -s`
  # input (a JSON array of every line's parsed object, across all matched
  # files), then compute the aggregate in one filter.
  {
    for f in "${transcripts[@]}"; do
      cat "$f"
    done
  } | jq -s \
    --argjson transcript_count "${#transcripts[@]}" \
    --arg transcript_paths "$(printf '%s\n' "${transcripts[@]}" | jq -R . | jq -s -c .)" \
    '
    [ .[] | select(.type == "assistant") | .message.usage // empty ] as $usages
    |
    ($usages | map(.input_tokens // 0) | add // 0) as $input_tokens
    |
    ($usages | map(.cache_read_input_tokens // 0) | add // 0) as $cache_read_tokens
    |
    ($usages | map(.cache_creation_input_tokens // 0) | add // 0) as $cache_creation_tokens
    |
    ($usages | map(.output_tokens // 0) | add // 0) as $output_tokens
    |
    # Disjoint-field sum — mirrors pkg/codex/usage.go billedTotal() exactly.
    # Never collapse this to `input + output` alone: that omission is what
    # produced the documented 186x undercount this repo already shipped once.
    ($input_tokens + $cache_read_tokens + $cache_creation_tokens + $output_tokens) as $total_tokens
    |
    ($input_tokens + $cache_read_tokens + $cache_creation_tokens) as $total_input_tokens
    |
    ($usages | length) as $model_calls
    |
    ( [ .[] | select(.type == "assistant") | .message.model // empty ] | unique | join(",") ) as $models
    |
    {
      total_tokens: $total_tokens,
      total_input_tokens: $total_input_tokens,
      model_calls: $model_calls,
      models: $models,
      source: "session-transcript",
      transcript_count: $transcript_count,
      transcript_paths: ($transcript_paths | fromjson)
    }
    '
}

# Allow standalone invocation: `measure-tokens.sh HOME_DIR`
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
  if [ $# -lt 1 ]; then
    echo "usage: measure-tokens.sh HOME_DIR" >&2
    exit 1
  fi
  measure_tokens "$1"
fi
