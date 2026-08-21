#!/usr/bin/env bash
# smoke-hermetic.sh — the hermetic-profile sequencing gate for the
# Aether-vs-GSD benchmark harness.
#
# Proves the harness's single biggest practical risk is survivable before
# anything else is built: a clean HOME containing only one lane's system can
# boot, authenticate, and complete one trivial task, for BOTH Aether and GSD.
#
# Five gates:
#   1. Aether lane boots in a clean HOME (fresh hub install)
#   2. Aether completes one trivial task in the clean HOME (init -> plan -> build --print-brief)
#   3. GSD lane boots in a clean HOME (file-copy install)
#   4. The Claude CLI authenticates under an overridden HOME, in BOTH lanes
#   5. The two lanes are filesystem-isolated from each other
#
# Exits non-zero on the first failure, naming the reason. Requires: go, node,
# npm, jq, git, the claude CLI, and gsd-sdk on PATH (with GSD already
# installed in the operator's real home directory — see bench/README.md).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="$(mktemp -d)"
trap 'chmod -R u+w "$WORK" 2>/dev/null || true; rm -rf "$WORK"' EXIT

fail() { echo "SMOKE FAIL: $*" >&2; exit 1; }
step() { echo; echo "==> $*"; }

# Capture the real HOME before any override. Honor a pre-set REAL_HOME
# (used by callers/tests that need to point the GSD lane at a deliberately
# broken source, e.g. `REAL_HOME=/nonexistent bench/smoke-hermetic.sh`).
REAL_HOME="${REAL_HOME:-$HOME}"

for tool in go node npm jq git claude; do
  command -v "$tool" >/dev/null || fail "required tool missing: $tool"
done
command -v gsd-sdk >/dev/null || fail "required tool missing: gsd-sdk (must be on PATH before any HOME override)"

source "$ROOT/bench/lib/hermetic-home.sh"
source "$ROOT/bench/lib/gsd-install.sh"

# ---------------------------------------------------------------------------
step "gate 1: Aether lane boots in a clean HOME"
# ---------------------------------------------------------------------------
# BIN lives under its own subdirectory so it never collides with the
# "aether" lane's home directory created below (both would otherwise be
# named $WORK/aether).
mkdir -p "$WORK/bin"
BIN="$WORK/bin/aether"
(cd "$ROOT" && go build -o "$BIN" ./cmd/aether) || fail "go build ./cmd/aether failed"

hermetic_home_setup aether "$WORK"
hermetic_home_probe
hermetic_home_install_aether "$ROOT" "$BIN"

AETHER_VERSION_OUT="$("$BIN" version 2>&1)" || fail "aether version failed under isolated HOME"
[ -n "$AETHER_VERSION_OUT" ] || fail "aether version produced empty output under isolated HOME"
[ -d "$HOME/.aether/system" ] || fail "isolated hub install did not create \$HOME/.aether/system"
AETHER_HOME="$HOME"
echo "    aether lane booted (HOME=$AETHER_HOME)"

# ---------------------------------------------------------------------------
step "gate 2: Aether completes one trivial task in the clean HOME"
# ---------------------------------------------------------------------------
AETHER_REPO="$WORK/aether-repo"
mkdir -p "$AETHER_REPO"
(cd "$AETHER_REPO" && git init -q && git config user.email smoke@example.test \
  && git config user.name "Hermetic Smoke" \
  && printf '# smoke repo\n' > README.md && git add README.md && git commit -q -m init) \
  || fail "could not create throwaway Aether repo at $AETHER_REPO"

AETHER_INIT_OUT="$WORK/aether-init.out"
set +e
(cd "$AETHER_REPO" && "$BIN" init "Add a trivial hello world script" >"$AETHER_INIT_OUT" 2>&1)
AETHER_INIT_STATUS=$?
set -e
if [ "$AETHER_INIT_STATUS" -ne 0 ]; then
  sed -n '1,10p' "$AETHER_INIT_OUT" >&2
  fail "aether init failed under isolated HOME — see $AETHER_INIT_OUT"
fi

# A fresh colony has no plan yet, so `build --print-brief` cannot run right
# after `init` — it requires a plan to exist first. `plan --synthetic`
# produces a local-synthesis plan without spawning real (token-costing)
# workers, which is the correct zero-cost proof for this smoke gate.
AETHER_PLAN_OUT="$WORK/aether-plan.out"
set +e
(cd "$AETHER_REPO" && "$BIN" plan --synthetic >"$AETHER_PLAN_OUT" 2>&1)
AETHER_PLAN_STATUS=$?
set -e
if [ "$AETHER_PLAN_STATUS" -ne 0 ]; then
  sed -n '1,10p' "$AETHER_PLAN_OUT" >&2
  fail "aether plan --synthetic failed under isolated HOME — see $AETHER_PLAN_OUT"
fi

AETHER_BRIEF_OUT="$WORK/aether-brief.out"
set +e
(cd "$AETHER_REPO" && "$BIN" build 1 --print-brief >"$AETHER_BRIEF_OUT" 2>&1)
AETHER_BRIEF_STATUS=$?
set -e
if [ "$AETHER_BRIEF_STATUS" -ne 0 ]; then
  sed -n '1,10p' "$AETHER_BRIEF_OUT" >&2
  fail "aether build 1 --print-brief failed under isolated HOME — see $AETHER_BRIEF_OUT"
fi
[ -s "$AETHER_BRIEF_OUT" ] || fail "aether build 1 --print-brief produced empty output"
echo "    aether lane completed init -> plan --synthetic -> build --print-brief (exit 0, non-empty output)"

# ---------------------------------------------------------------------------
step "gate 3: GSD lane boots in a clean HOME"
# ---------------------------------------------------------------------------
hermetic_home_setup gsd "$WORK"
gsd_install_into_home "$REAL_HOME"

[ -s "$HOME/.claude/get-shit-done/VERSION" ] || fail "gsd-install did not produce a non-empty VERSION file"
GSD_AGENT_COUNT="$(ls "$HOME"/.claude/agents/gsd-*.md 2>/dev/null | wc -l | tr -d ' ')"
GSD_SKILL_COUNT="$(ls -d "$HOME"/.claude/skills/gsd-* 2>/dev/null | wc -l | tr -d ' ')"
[ "$GSD_AGENT_COUNT" -ge 20 ] || fail "GSD lane: expected at least 20 gsd-* agents, found $GSD_AGENT_COUNT"
[ "$GSD_SKILL_COUNT" -ge 20 ] || fail "GSD lane: expected at least 20 gsd-* skills, found $GSD_SKILL_COUNT"

set +e
GSD_SDK_VERSION_OUT="$(gsd-sdk --version 2>&1)"
GSD_SDK_STATUS=$?
set -e
[ "$GSD_SDK_STATUS" -eq 0 ] || fail "gsd-sdk --version failed under isolated HOME (exit $GSD_SDK_STATUS): $GSD_SDK_VERSION_OUT"
[ -n "$GSD_SDK_VERSION_OUT" ] || fail "gsd-sdk --version produced empty output under isolated HOME"
GSD_HOME="$HOME"
echo "    gsd lane booted (HOME=$GSD_HOME, agents=$GSD_AGENT_COUNT, skills=$GSD_SKILL_COUNT, gsd-sdk=$GSD_SDK_VERSION_OUT)"

# ---------------------------------------------------------------------------
step "gate 4: the Claude CLI authenticates under an overridden HOME, in BOTH isolated HOMEs"
# ---------------------------------------------------------------------------
for LANE in aether gsd; do
  if [ "$LANE" = "aether" ]; then
    export HOME="$AETHER_HOME"
  else
    export HOME="$GSD_HOME"
  fi

  AUTH_OUT="$WORK/${LANE}-auth.out"
  set +e
  claude -p "Reply with exactly the word: hermetic-smoke-ok" >"$AUTH_OUT" 2>&1
  AUTH_STATUS=$?
  set -e

  if [ "$AUTH_STATUS" -ne 0 ]; then
    sed -n '1,10p' "$AUTH_OUT" >&2
    if grep -q "Not logged in" "$AUTH_OUT"; then
      fail "lane $LANE: claude CLI reports 'Not logged in' under an overridden HOME. The Claude CLI's normal login stores its session in the macOS Keychain, which is scoped to the OS user/session, not to \$HOME — overriding \$HOME does not carry that credential with it, and \`claude auth login\` itself requires an interactive browser step it cannot complete unattended. To make this lane hermetic, provision ANTHROPIC_API_KEY (or an apiKeyHelper) before running this script — that path is explicitly \$HOME-independent and bypasses Keychain. See $AUTH_OUT"
    fi
    fail "lane $LANE: claude CLI did not authenticate under an overridden HOME — see $AUTH_OUT"
  fi

  TRANSCRIPT="$(ls "$HOME"/.claude/projects/*/*.jsonl 2>/dev/null | head -1 || true)"
  [ -n "$TRANSCRIPT" ] || fail "lane $LANE: claude CLI exited 0 but no session transcript was found under \$HOME/.claude/projects — see $AUTH_OUT"
  echo "    lane $LANE: claude CLI authenticated, transcript=$TRANSCRIPT"
done

# ---------------------------------------------------------------------------
step "gate 5: the two lanes are filesystem-isolated"
# ---------------------------------------------------------------------------
[ "$AETHER_HOME" != "$GSD_HOME" ] || fail "aether and gsd lanes share the same HOME — isolation broken"
[ ! -d "$AETHER_HOME/.claude/get-shit-done" ] || fail "aether lane HOME contains a get-shit-done directory — lane crossover"
[ ! -d "$GSD_HOME/.aether/system" ] || fail "gsd lane HOME contains an .aether/system directory — lane crossover"
echo "    lanes are filesystem-isolated (aether=$AETHER_HOME, gsd=$GSD_HOME)"

echo
echo "HERMETIC SMOKE PASS: both systems boot, authenticate and complete a trivial task in clean HOMEs"
