#!/usr/bin/env bash
# run-lane.sh — everything lane-specific for the Aether-vs-GSD benchmark
# harness, kept out of run-cell.sh so the cell runner never branches on
# `if lane == gsd` in its measurement or judging code.
#
# Sourceable library (not directly executable). Sources bench/lib/hermetic-home.sh
# and bench/lib/gsd-install.sh, so the caller must have already resolved ROOT
# and BIN (the built aether binary path) before sourcing this file — see
# run-cell.sh for the exact sourcing order.
#
# Exposes:
#   lane_prepare LANE WORK              — isolated HOME setup + that lane's install
#   lane_invocation LANE CATEGORY       — prints the entry command sequence for a cell
#   lane_working_state_paths LANE       — prints the paths that lane legitimately creates
#   lane_force_model LANE MODEL         — pins the model for that lane, or records a deviation
set -euo pipefail

RUN_LANE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

_lane_fail() { echo "RUN-LANE FAIL: $*" >&2; exit 1; }

_lane_require_known() {
  local lane="$1"
  case "$lane" in
    gsd|aether-interactive|aether-autopilot) return 0 ;;
    *) _lane_fail "unknown lane: '$lane' — must be one of gsd, aether-interactive, aether-autopilot" ;;
  esac
}

# lane_prepare LANE WORK
#
# Creates the lane's isolated HOME via hermetic_home_setup, then installs
# that lane's system only. Fails on an unknown lane rather than defaulting.
lane_prepare() {
  local lane="$1"
  local work="$2"

  _lane_require_known "$lane"

  # shellcheck source=bench/lib/hermetic-home.sh
  source "$RUN_LANE_ROOT/bench/lib/hermetic-home.sh"

  hermetic_home_setup "$lane" "$work"
  hermetic_home_probe || true

  case "$lane" in
    aether-interactive|aether-autopilot)
      local bin="${BENCH_AETHER_BIN:-}"
      [ -n "$bin" ] || _lane_fail "lane_prepare: BENCH_AETHER_BIN is not set — the cell runner must build/locate the aether binary before calling lane_prepare for an Aether lane"
      hermetic_home_install_aether "$RUN_LANE_ROOT" "$bin"
      ;;
    gsd)
      # shellcheck source=bench/lib/gsd-install.sh
      source "$RUN_LANE_ROOT/bench/lib/gsd-install.sh"
      gsd_install_into_home "${REAL_HOME:-}"
      ;;
  esac
}

# lane_invocation LANE CATEGORY
#
# Prints the entry command sequence for that cell, read from the task spec's
# "## Per-lane invocation" section rather than duplicated here — the spec is
# the reviewed artifact, and duplicating it would let the runner and the
# spec drift.
lane_invocation() {
  local lane="$1"
  local category="$2"

  _lane_require_known "$lane"

  local spec="$RUN_LANE_ROOT/bench/tasks/${category}.md"
  [ -f "$spec" ] || _lane_fail "lane_invocation: no task spec for category '$category' at $spec"

  local lane_label
  case "$lane" in
    gsd) lane_label="GSD" ;;
    aether-interactive) lane_label="Aether interactive" ;;
    aether-autopilot) lane_label="Aether autopilot" ;;
  esac

  # Extract the "## Per-lane invocation" section, then the one bullet whose
  # bolded lead matches this lane's label — printed as-is from the spec, not
  # re-derived, so the runner can never drift from the reviewed document.
  awk -v label="$lane_label" '
    /^## Per-lane invocation/ { in_section = 1; next }
    in_section && /^## / { in_section = 0 }
    in_section && $0 ~ "\\*\\*" label ":\\*\\*" { print; found = 1; next }
    in_section && found && /^  / { print; next }
    in_section && found && /^- \*\*/ { found = 0 }
  ' "$spec"
}

# lane_working_state_paths LANE
#
# Prints the paths that lane legitimately creates as normal operation, for
# the cleanliness checker (bench/lib/git-cleanliness.sh's lane block).
lane_working_state_paths() {
  local lane="$1"

  _lane_require_known "$lane"

  case "$lane" in
    aether-interactive|aether-autopilot)
      printf '%s\n' ".aether/**"
      ;;
    gsd)
      printf '%s\n' ".planning/**" ".claude/**"
      ;;
  esac
}

# lane_force_model LANE MODEL
#
# Applies the pinned model to that lane. For the Aether lanes this means
# setting agent frontmatter to "inherit" via configuration in the isolated
# HOME — never by editing this repository's source. If the lane cannot be
# forced to the pinned model by configuration, prints a MODEL-DEVIATION line
# and returns 0 — CONTEXT.md requires the deviation be recorded, not that
# the run be abandoned.
lane_force_model() {
  local lane="$1"
  local model="$2"

  _lane_require_known "$lane"
  [ -n "$model" ] || _lane_fail "lane_force_model: MODEL argument is required"

  case "$lane" in
    aether-interactive|aether-autopilot)
      # Aether agent frontmatter is forced to "inherit" via a config file in
      # the isolated HOME's hub, never by editing this repo's agent sources.
      # The hub config's model-override key is read by the runtime ahead of
      # any per-agent frontmatter model pin.
      local hub_dir="${AETHER_HUB_DIR:-$HOME/.aether}"
      if [ -d "$hub_dir" ]; then
        mkdir -p "$hub_dir"
        local config_file="$hub_dir/benchmark-model-override.json"
        printf '{"agent_model_override": "inherit", "orchestrator_model": "%s"}\n' "$model" > "$config_file"
        export ANTHROPIC_MODEL="$model"
        echo "lane-force-model lane=$lane model=$model mechanism=hub-config file=$config_file"
      else
        echo "MODEL-DEVIATION lane=$lane reason=hub-dir-not-yet-installed-cannot-write-override-config"
      fi
      ;;
    gsd)
      # GSD's model selection is driven by the operator's Claude Code
      # settings/CLI flags, not by a per-agent frontmatter field this
      # harness can override from inside the isolated HOME.
      export ANTHROPIC_MODEL="$model"
      echo "lane-force-model lane=$lane model=$model mechanism=env-ANTHROPIC_MODEL"
      ;;
  esac
}
