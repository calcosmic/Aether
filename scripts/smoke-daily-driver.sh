#!/usr/bin/env bash
# smoke-daily-driver.sh — the daily-driver regression tripwire.
#
# Builds aether from source, installs it into a fully isolated HOME/hub,
# then proves the four things that must never break for a downstream user:
#
#   1. `aether update --force` in a repo with foreign (GSD) Claude settings
#      preserves every foreign hook/permission while installing Aether's.
#   2. A second forced update leaves settings.json byte-identical (idempotent).
#   3. `aether host plan --dry-run` runs the TS host without any
#      ERR_MODULE_NOT_FOUND — dependencies self-provision on fresh installs.
#   4. `aether build 1 --plan-only` emits a parseable dispatch_manifest
#      envelope directly from Go (the wrapper's manifest path).
#
# Exits non-zero on the first failure. Requires: go, node >= 20, npm, jq, git.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="$(mktemp -d)"
trap 'chmod -R u+w "$WORK" 2>/dev/null || true; rm -rf "$WORK"' EXIT

# Pin Go caches to the real ones BEFORE overriding HOME, so tool invocations
# under the isolated HOME reuse the module cache instead of re-downloading
# into a throwaway (and read-only-flavored) GOPATH.
export GOPATH="$(go env GOPATH)"
export GOMODCACHE="$(go env GOMODCACHE)"
export GOCACHE="$(go env GOCACHE)"

fail() { echo "SMOKE FAIL: $*" >&2; exit 1; }
step() { echo; echo "==> $*"; }

for tool in go node npm jq git; do
  command -v "$tool" >/dev/null || fail "required tool missing: $tool"
done

BIN="$WORK/aether"
step "building aether from source"
(cd "$ROOT" && go build -o "$BIN" ./cmd/aether)

# Fully isolate: nothing touches the real ~/.claude or ~/.aether.
export HOME="$WORK/home"
export AETHER_HUB_DIR="$HOME/.aether"
mkdir -p "$HOME"

step "installing package into isolated hub"
(cd "$ROOT" && "$BIN" install --package-dir "$ROOT" >/dev/null) || fail "aether install failed"

REPO="$WORK/repo"
mkdir -p "$REPO/.claude"
git -C "$WORK" >/dev/null 2>&1 || true
git init -q "$REPO"

step "planting foreign GSD settings in downstream repo"
cat > "$REPO/.claude/settings.json" <<'EOF'
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {"type": "command", "command": "gsd-sdk hook pre-bash", "timeout": 30}
        ]
      }
    ]
  },
  "permissions": {
    "allow": ["Bash(npm test:*)"]
  },
  "gsdMarker": "do-not-touch"
}
EOF

step "gate 1: aether update --force preserves foreign settings"
(cd "$REPO" && "$BIN" update --force >/dev/null) || fail "aether update --force failed"
SETTINGS="$REPO/.claude/settings.json"
jq -e '.gsdMarker == "do-not-touch"' "$SETTINGS" >/dev/null \
  || fail "custom top-level key lost by update --force"
jq -e '.permissions.allow | index("Bash(npm test:*)")' "$SETTINGS" >/dev/null \
  || fail "foreign permission lost by update --force"
jq -e '[.hooks.PreToolUse[].hooks[].command] | index("gsd-sdk hook pre-bash")' "$SETTINGS" >/dev/null \
  || fail "foreign GSD hook lost by update --force"
jq -e '[.hooks.PreToolUse[].hooks[].command] | index("aether hook-pre-tool-use")' "$SETTINGS" >/dev/null \
  || fail "aether hook not installed by update --force"
echo "    settings merge OK"

step "gate 2: repeated update --force is idempotent"
BEFORE="$(shasum -a 256 "$SETTINGS")"
(cd "$REPO" && "$BIN" update --force >/dev/null) || fail "second aether update --force failed"
AFTER="$(shasum -a 256 "$SETTINGS")"
[ "$BEFORE" = "$AFTER" ] || fail "second update rewrote settings.json"
echo "    idempotency OK"

step "gate 3: TS host runs without missing-dependency errors"
HOST_OUT="$WORK/host-plan.out"
set +e
(cd "$REPO" && "$BIN" host plan --dry-run >"$HOST_OUT" 2>&1)
HOST_STATUS=$?
set -e
if grep -q "ERR_MODULE_NOT_FOUND\|Cannot find package" "$HOST_OUT"; then
  sed -n '1,10p' "$HOST_OUT" >&2
  fail "TS host died on missing dependencies (P0-04 regression)"
fi
if grep -q "ts host not found\|TS host dependencies unavailable" "$HOST_OUT"; then
  sed -n '1,10p' "$HOST_OUT" >&2
  fail "TS host could not be provisioned"
fi
echo "    host executed (exit $HOST_STATUS, no dependency errors)"

step "gate 4: direct Go plan-only build emits a dispatch manifest"
cat > "$REPO/.aether/data/COLONY_STATE.json" <<'EOF'
{
  "version": "3.0",
  "goal": "Smoke: prove the daily-driver build path",
  "state": "READY",
  "current_phase": 1,
  "plan": {
    "phases": [
      {
        "id": 1,
        "name": "Smoke Phase",
        "description": "Single-task smoke phase",
        "status": "ready",
        "tasks": [
          {
            "id": "task-1",
            "description": "Touch a smoke file",
            "type": "implementation",
            "status": "pending"
          }
        ]
      }
    ]
  },
  "memory": {"phase_learnings": [], "decisions": [], "instincts": []},
  "errors": {"records": [], "flagged_patterns": []},
  "signals": [],
  "graveyards": [],
  "events": []
}
EOF
BUILD_OUT="$WORK/build-plan.json"
(cd "$REPO" && "$BIN" build 1 --plan-only >"$BUILD_OUT" 2>"$WORK/build-plan.err") \
  || { cat "$WORK/build-plan.err" >&2; fail "aether build 1 --plan-only failed"; }
jq -e '.ok == true and (.result.dispatch_manifest.dispatches | length) >= 1' "$BUILD_OUT" >/dev/null \
  || { head -c 400 "$BUILD_OUT" >&2; fail "plan-only build did not emit a parseable dispatch_manifest"; }
echo "    dispatch manifest OK ($(jq '.result.dispatch_manifest.dispatches | length' "$BUILD_OUT") dispatches)"

echo
echo "SMOKE PASS: all daily-driver gates green"
