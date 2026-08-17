#!/usr/bin/env bash
# hermetic-home.sh — shared isolated-HOME setup for the Aether-vs-GSD benchmark harness.
#
# Sourceable library (not directly executable). Extracted from the existing
# scripts/smoke-daily-driver.sh isolated-HOME pattern so every lane runner
# (Aether interactive, Aether autopilot, GSD) builds its clean HOME the same
# way instead of reinventing it three times.
#
# Exposes:
#   hermetic_home_setup LANE_NAME BASE_DIR   — create + export an isolated HOME
#   hermetic_home_probe                       — detect getpwuid/$HOME divergence
#   hermetic_home_install_aether ROOT BIN     — fresh Aether hub install into it
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

fail() { echo "SMOKE FAIL: $*" >&2; exit 1; }
step() { echo; echo "==> $*"; }

# hermetic_home_setup LANE_NAME BASE_DIR
#
# Creates $BASE_DIR/$LANE_NAME/home, exports HOME to it, and writes an
# identical minimal global CLAUDE.md into every lane so no lane gets a
# prompting advantage.
hermetic_home_setup() {
  local lane_name="$1"
  local base_dir="$2"

  # Pin Go caches to the real ones BEFORE overriding HOME, so tool
  # invocations under the isolated HOME reuse the module cache instead of
  # re-downloading into a throwaway (and read-only-flavored) GOPATH. This
  # ordering is load-bearing: isolating HOME also isolates GOPATH/GOCACHE
  # by default, which would force a full module re-download per lane and
  # break the "reproducible in one documented command, under a week" goal.
  export GOPATH
  GOPATH="$(go env GOPATH)"
  export GOMODCACHE
  GOMODCACHE="$(go env GOMODCACHE)"
  export GOCACHE
  GOCACHE="$(go env GOCACHE)"

  local lane_home="$base_dir/$lane_name/home"
  mkdir -p "$lane_home"
  export HOME="$lane_home"
  mkdir -p "$HOME/.claude" "$HOME/.config"

  # Identical minimal global CLAUDE.md for every lane — same bytes, no
  # mention of Aether, GSD, colonies, phases, or benchmarks, so neither
  # system gets a prompting advantage from its own instructions file.
  cat > "$HOME/.claude/CLAUDE.md" <<'EOF'
# Global Instructions

Follow the user's instructions precisely and verify your work before reporting completion.
EOF

  echo "hermetic-home lane=$lane_name home=$HOME"
}

# hermetic_home_probe
#
# Detects the getpwuid divergence risk: some Node/libc home-directory
# resolution paths consult getpwuid() instead of $HOME, which would silently
# defeat the isolation this harness depends on. Prints a loud warning and
# sets HERMETIC_HOME_GETPWUID_DIVERGES=1 on disagreement, but never exits —
# the caller's gates decide whether the divergence is fatal.
hermetic_home_probe() {
  local node_home
  node_home="$(node -e 'process.stdout.write(require("os").homedir())')"
  if [ "$node_home" != "$HOME" ]; then
    echo "WARNING: getpwuid/os.homedir() divergence detected — os.homedir()=[$node_home] but \$HOME=[$HOME]" >&2
    echo "WARNING: tools that resolve home via getpwuid() instead of \$HOME may leak into the real home directory" >&2
    export HERMETIC_HOME_GETPWUID_DIVERGES=1
  else
    echo "hermetic-home-probe: os.homedir() agrees with \$HOME ($HOME)"
  fi
}

# hermetic_home_install_aether ROOT BIN
#
# Installs a fresh Aether hub into the current (already-isolated) HOME by
# pointing AETHER_HUB_DIR at $HOME/.aether and running the built binary's
# package-dir install.
hermetic_home_install_aether() {
  local root="$1"
  local bin="$2"

  export AETHER_HUB_DIR="$HOME/.aether"

  step "installing Aether package into isolated hub ($AETHER_HUB_DIR)"
  (cd "$root" && "$bin" install --package-dir "$root" >/dev/null) \
    || fail "aether install --package-dir failed for isolated HOME=$HOME"
}
