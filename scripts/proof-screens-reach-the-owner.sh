#!/usr/bin/env bash
# proof-screens-reach-the-owner.sh — Phase 206 plan 02's real-chat proof.
#
# Proves, in one real `claude -p` chat, in a scratch project with the hooks
# genuinely installed by the real `aether install` + `aether update --force`
# path, that:
#
#   1. That install path really registers both the direct screen-delivery
#      route (plan 01) and the status line (this plan's Task 1).
#   2. A real Bash tool call the chat makes really runs Aether's own screen
#      command.
#   3. Aether's screen reaches the owner as a Claude Code informational
#      message carrying every banner line the command drew — not merely as
#      the Bash tool's own raw stdout echoed back into the transcript, which
#      would happen regardless of whether the direct route works at all.
#   4. The status line prints.
#
# ISOLATION RULE, followed throughout this script: nothing here writes to the
# real $HOME/.claude/ or the real ~/.aether/ hub. Every hook this proof
# exercises comes from the scratch project's own settings file, installed
# into an isolated hub via AETHER_HUB_DIR. HOME itself is deliberately left
# ALONE (not overridden) because the Claude Code CLI keeps its sign-in there
# and an isolated HOME cannot authenticate — nothing else in this script
# reads or writes through HOME.
#
# The scratch binary this script builds is put first on PATH ONLY around the
# `claude` invocation in gate 3, so the hooks the chat's own Bash calls
# trigger resolve to this build rather than any binary already on the
# developer's PATH.
#
# Exits non-zero on the first failure. Requires: go, git, jq, claude.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

fail() { echo "PROOF FAIL: $*" >&2; exit 1; }
step() { echo; echo "==> $*"; }

for tool in go git jq claude; do
  command -v "$tool" >/dev/null 2>&1 || fail "required tool missing: $tool"
done

BIN="$WORK/aether"
step "building aether from source"
(cd "$ROOT" && go build -o "$BIN" ./cmd/aether) || fail "go build failed"

# Isolate the hub only. HOME is intentionally untouched -- see the ISOLATION
# RULE above.
export AETHER_HUB_DIR="$WORK/hub"
mkdir -p "$AETHER_HUB_DIR"

step "installing package into isolated hub"
(cd "$ROOT" && "$BIN" install --package-dir "$ROOT" >/dev/null) || fail "aether install failed"

REPO="$WORK/repo"
mkdir -p "$REPO"
git init -q "$REPO"
git -C "$REPO" config user.email "proof@example.com"
git -C "$REPO" config user.name "proof"

step "aether init in the scratch project"
(cd "$REPO" && "$BIN" init "Prove Aether's screens reach the owner" >/dev/null 2>&1) \
  || fail "aether init failed in the scratch project"

step "aether update --force in the scratch project (the install path under test)"
(cd "$REPO" && "$BIN" update --force >/dev/null) || fail "aether update --force failed"

SETTINGS="$REPO/.claude/settings.json"

step "gate 1: the install really registered both routes"
[ -f "$SETTINGS" ] || fail "no .claude/settings.json after aether update --force"
jq -e '[(.hooks.PostToolUse // [])[].hooks[].command] | index("aether hook-post-tool-use")' "$SETTINGS" >/dev/null \
  || fail "aether hook-post-tool-use is not registered under PostToolUse -- run aether update --force"
jq -e '.statusLine.command == "aether status-line"' "$SETTINGS" >/dev/null \
  || fail "aether status-line is not registered as the statusLine command -- run aether update --force"
echo "    both routes registered in $SETTINGS"

step "gate 2: work out the expected banner lines"
STATUS_OUT="$WORK/status.out"
(cd "$REPO" && AETHER_OUTPUT_MODE=visual "$BIN" status >"$STATUS_OUT" 2>&1) || true
BANNERS="$WORK/banners.txt"
grep '━' "$STATUS_OUT" >"$BANNERS" || true
[ -s "$BANNERS" ] || fail "aether status drew no banner lines at all -- the rest of this proof would assert nothing"
echo "    $(wc -l <"$BANNERS" | tr -d ' ') banner line(s) captured from aether status"

step "gate 3: the real chat"
PROMPT='Run the exact command `AETHER_OUTPUT_MODE=visual aether status` once, then stop. Add no commentary.'
ARGS=(-p "$PROMPT" --output-format stream-json --verbose --allowedTools Bash --settings "$SETTINGS" --max-turns 6)
CAPS=("max-turns=6")
if claude --help 2>&1 | grep -q -- '--max-budget-usd'; then
  ARGS+=(--max-budget-usd 1.00)
  CAPS+=("max-budget-usd=1.00")
fi
CAPS+=("wall-clock=300s (timeout 300)")
echo "    caps in force: ${CAPS[*]}"

CHAT_OUT="$WORK/chat.jsonl"
CHAT_ERR="$WORK/chat.err"
run_chat() {
  (cd "$REPO" && PATH="$(dirname "$BIN"):$PATH" timeout 300 claude "${ARGS[@]}" >"$CHAT_OUT" 2>"$CHAT_ERR")
}

RETRIED="no"
set +e
run_chat
CHAT_STATUS=$?
set -e
if [ "$CHAT_STATUS" -ne 0 ]; then
  if grep -qiE 'rate.?limit|overloaded|timed?.?out' "$CHAT_ERR" "$CHAT_OUT" 2>/dev/null; then
    echo "    transient failure (exit $CHAT_STATUS) -- retrying once, as instructed; not counted as a real failure unless it recurs"
    RETRIED="yes"
    set +e
    run_chat
    CHAT_STATUS=$?
    set -e
  fi
fi
if [ "$CHAT_STATUS" -ne 0 ]; then
  tail -c 2000 "$CHAT_ERR" >&2
  fail "claude -p failed (exit $CHAT_STATUS) -- if this names a reason outside Aether (not signed in, an unknown flag), that is reported here rather than weakened into a passing gate"
fi

step "gate 4: assert on what actually happened"
BASH_CALLS=$(jq -c 'select(.message.content != null) | .message.content[] | select(.type=="tool_use" and .name=="Bash")' "$CHAT_OUT" 2>/dev/null | wc -l | tr -d ' ')
[ "$BASH_CALLS" -ge 1 ] || fail "the chat never ran a Bash tool call -- it never ran the command at all, which is a different failure from a delivery failure and must not be reported as one"
echo "    the chat ran $BASH_CALLS Bash tool call(s)"

# Primary: the observed real shape of Claude Code's hook-derived
# informational message -- a top-level "system" stream event whose subtype
# names it, carrying the delivered text in its own message field. Verified
# empirically against a real run of this script (2026-09-21): each such
# event decodes as {"type":"system","subtype":"informational","message":
# "PostToolUse:Bash says: <line>\n...", "level":"notice", ...}.
jq -r 'select(.type=="system") | select(.subtype=="informational" or .subtype=="info") | (.message // .systemMessage // empty)' \
  "$CHAT_OUT" >"$WORK/delivered-message.txt" 2>/dev/null || true

if [ -s "$WORK/delivered-message.txt" ]; then
  DELIVERY_SOURCE="the targeted system/informational message field"
else
  # Fallback: do not assume the exact envelope shape survives a future
  # Claude Code release. Take every string leaf anywhere in the stream
  # EXCEPT inside a tool_result content block -- the Bash tool's own
  # stdout, echoed back into the transcript, which will ALWAYS contain the
  # banner text regardless of whether the direct-delivery route works at
  # all, and so must never be allowed to satisfy this assertion on its own.
  jq -r '.. | strings' "$CHAT_OUT" >"$WORK/all-strings.txt" 2>/dev/null || true
  jq -r '.. | objects | select(.type=="tool_result") | .content | if type=="string" then . else (.. | strings) end' \
    "$CHAT_OUT" >"$WORK/tool-result-strings.txt" 2>/dev/null || true
  grep -vFxf "$WORK/tool-result-strings.txt" "$WORK/all-strings.txt" >"$WORK/delivered-message.txt" 2>/dev/null || true
  DELIVERY_SOURCE="a fallback scan (the known system/informational shape was not found; excluded only the Bash tool's own echoed output)"
fi

MISSING=""
while IFS= read -r banner_line; do
  [ -z "$banner_line" ] && continue
  if ! grep -qF -- "$banner_line" "$WORK/delivered-message.txt" 2>/dev/null; then
    MISSING="${MISSING}  - ${banner_line}\n"
  fi
done <"$BANNERS"

if [ -n "$MISSING" ]; then
  printf 'PROOF FAIL: the following banner line(s) never reached the owner as an informational message (only as the Bash tool'"'"'s own output, if anywhere):\n%b' "$MISSING" >&2
  fail "the screen did not reach the owner as an informational message"
fi
echo "    every captured banner line reached the owner as an informational message, distinct from the Bash tool's own output"
echo "    delivery evidence source: $DELIVERY_SOURCE"

# Display refinement only (never changes the pass/fail decision above, which
# already ran against the unfiltered delivered-message.txt): every hook-
# relayed line carries a stable "<hook name> says: " prefix (Aether's own
# `directScreenDelivery` composes it, and it is what the owner actually sees
# on screen), so filtering the fallback's noisier full-stream dump down to
# just those lines is a pure readability improvement when the targeted
# selector above did not match. If nothing matches this pattern either, the
# full (noisier but still verbatim) dump is shown rather than showing nothing.
DISPLAY_FILE="$WORK/delivered-message.txt"
CLEAN_DELIVERY="$WORK/delivered-message-clean.txt"
if grep -F ' says: ' "$WORK/delivered-message.txt" >"$CLEAN_DELIVERY" 2>/dev/null && [ -s "$CLEAN_DELIVERY" ]; then
  DISPLAY_FILE="$CLEAN_DELIVERY"
fi

echo
echo "=== The delivered message (this is what the owner will be asked to look at) ==="
cat "$DISPLAY_FILE"
echo "=== end of delivered message ==="

step "gate 5: the status line"
STATUS_LINE_OUT="$WORK/status-line.out"
(cd "$REPO" && "$BIN" status-line >"$STATUS_LINE_OUT") || fail "aether status-line failed"
STATUS_LINE_COUNT=$(grep -c . "$STATUS_LINE_OUT" || true)
[ "$STATUS_LINE_COUNT" -eq 1 ] || fail "aether status-line printed $STATUS_LINE_COUNT non-empty line(s), want exactly 1"
echo "    status line: $(cat "$STATUS_LINE_OUT")"

echo
echo "=== SUMMARY ==="
echo "PROOF PASS: all gates green"
echo "caps in force for the chat run: ${CAPS[*]}"
echo "transient failure retried: $RETRIED"
echo "banner lines proven delivered: $(wc -l <"$BANNERS" | tr -d ' ')"
echo "status line: $(cat "$STATUS_LINE_OUT")"
