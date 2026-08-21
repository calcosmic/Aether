#!/usr/bin/env bash
# selftest.sh — proves the three measurement libraries (measure-tokens,
# git-cleanliness, operator-log) compute the right answers on committed
# fixtures, against hand-computed literal expected values.
#
# Hand-computed arithmetic for bench/lib/fixtures/transcript-sample.jsonl
# (NOT derived by re-running measure-tokens.sh's own formula — that is the
# exact defect shape that let a 186x token undercount ship green in this
# repo once, because its unit test asserted the same wrong arithmetic the
# parser used):
#
#   Turn 1 — pkg/codex/usage.go's documented Anthropic worked example:
#     input=50, cache_read=100000, cache_creation=2000, output=500
#     total    = 50 + 100000 + 2000 + 500  = 102550
#     input-side = 50 + 100000 + 2000      = 102050
#
#   Turn 2 — small round numbers:
#     input=10, cache_read=20, cache_creation=30, output=40
#     total    = 10 + 20 + 30 + 40 = 100
#     input-side = 10 + 20 + 30    = 60
#
#   Fixture overall (sum across both turns, because a session transcript's
#   real cost is every turn summed, not the last one — see measure-tokens.sh):
#     total_tokens       = 102550 + 100 = 102650
#     total_input_tokens = 102050 + 60  = 102110
#
# Exits 0 and prints "BENCH SELFTEST PASS" on success. On any gate failure,
# prints the expected and actual values before exiting non-zero.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
LIB="$ROOT/bench/lib"

WORK="$(mktemp -d)"
trap 'chmod -R u+w "$WORK" 2>/dev/null || true; rm -rf "$WORK"' EXIT

# shellcheck source=/dev/null
source "$LIB/measure-tokens.sh"
# shellcheck source=/dev/null
source "$LIB/git-cleanliness.sh"
# shellcheck source=/dev/null
source "$LIB/operator-log.sh"

# Defined AFTER sourcing the three libraries above: each of them defines its
# own `fail()`/`step()` pair with the same names (a repo-wide convention —
# see scripts/smoke-daily-driver.sh), and sourcing silently overwrites
# whichever definition came first. Define selftest's own version last so its
# "SELFTEST FAIL:" prefix is the one that actually runs, not a borrowed one
# from whichever library happened to source last.
fail() { echo "SELFTEST FAIL: $*" >&2; exit 1; }
step() { echo; echo "==> $*"; }

# --- gate 1: token arithmetic --------------------------------------------
step "gate 1: token arithmetic against hand-computed literals"

EXPECTED_TOTAL_TOKENS=102650
EXPECTED_TOTAL_INPUT_TOKENS=102110
EXPECTED_MODEL_CALLS=2

FIXTURE_HOME="$WORK/gate1-home"
mkdir -p "$FIXTURE_HOME/.claude/projects/-fixture-repo"
cp "$LIB/fixtures/transcript-sample.jsonl" "$FIXTURE_HOME/.claude/projects/-fixture-repo/session.jsonl"

GATE1_JSON="$(measure_tokens "$FIXTURE_HOME")" || fail "measure_tokens failed against the fixture HOME"

ACTUAL_TOTAL_TOKENS="$(jq -r '.total_tokens' <<<"$GATE1_JSON")"
ACTUAL_TOTAL_INPUT_TOKENS="$(jq -r '.total_input_tokens' <<<"$GATE1_JSON")"
ACTUAL_MODEL_CALLS="$(jq -r '.model_calls' <<<"$GATE1_JSON")"
ACTUAL_SOURCE="$(jq -r '.source' <<<"$GATE1_JSON")"
ACTUAL_INPUT_TOKENS="$(jq -r '.total_tokens - .total_input_tokens' <<<"$GATE1_JSON" 2>/dev/null || echo "")"

[ "$ACTUAL_TOTAL_TOKENS" = "$EXPECTED_TOTAL_TOKENS" ] \
  || fail "gate 1: total_tokens expected=$EXPECTED_TOTAL_TOKENS actual=$ACTUAL_TOTAL_TOKENS"
[ "$ACTUAL_TOTAL_INPUT_TOKENS" = "$EXPECTED_TOTAL_INPUT_TOKENS" ] \
  || fail "gate 1: total_input_tokens expected=$EXPECTED_TOTAL_INPUT_TOKENS actual=$ACTUAL_TOTAL_INPUT_TOKENS"
[ "$ACTUAL_MODEL_CALLS" = "$EXPECTED_MODEL_CALLS" ] \
  || fail "gate 1: model_calls expected=$EXPECTED_MODEL_CALLS actual=$ACTUAL_MODEL_CALLS"
[ "$ACTUAL_SOURCE" = "session-transcript" ] \
  || fail "gate 1: source expected=session-transcript actual=$ACTUAL_SOURCE"

# Direct tripwire against the undercount shape returning: total_tokens must
# be strictly greater than input_tokens + output_tokens for this fixture
# (100050 = 50+100000... no: fixture's raw input_tokens+output_tokens sum,
# computed independently here, not via the parser).
RAW_INPUT_PLUS_OUTPUT=$(( (50 + 10) + (500 + 40) ))  # = 600, far below 102650
[ "$ACTUAL_TOTAL_TOKENS" -gt "$RAW_INPUT_PLUS_OUTPUT" ] \
  || fail "gate 1 tripwire: total_tokens ($ACTUAL_TOTAL_TOKENS) is not greater than input+output alone ($RAW_INPUT_PLUS_OUTPUT) — the 186x undercount shape may have returned"

echo "    gate 1 OK: total_tokens=$ACTUAL_TOTAL_TOKENS total_input_tokens=$ACTUAL_TOTAL_INPUT_TOKENS model_calls=$ACTUAL_MODEL_CALLS source=$ACTUAL_SOURCE"

# --- gate 2: no-transcript refusal ----------------------------------------
step "gate 2: no-transcript refusal"

EMPTY_HOME="$WORK/gate2-empty-home"
mkdir -p "$EMPTY_HOME"

if measure_tokens "$EMPTY_HOME" >/dev/null 2>"$WORK/gate2.err"; then
  fail "gate 2: measure_tokens against an empty HOME was expected to exit non-zero but exited 0"
fi
echo "    gate 2 OK: $(cat "$WORK/gate2.err")"

# --- gate 3: git cleanliness ----------------------------------------------
step "gate 3: git cleanliness against a constructed fixture repo"

FIXTURE_REPO="$WORK/gate3-repo"
mkdir -p "$FIXTURE_REPO"
(
  cd "$FIXTURE_REPO"
  git init -q -b main
  git config user.email "selftest@bench.local"
  git config user.name "bench-selftest"
  echo "base" > base.txt
  git add base.txt
  git commit -q -m "base commit"
)
BASE_SHA="$(cd "$FIXTURE_REPO" && git rev-parse HEAD)"

(
  cd "$FIXTURE_REPO"
  git checkout -q -b orphan-branch
  echo "orphan work" > orphan.txt
  git add orphan.txt
  git commit -q -m "orphan commit"
  git checkout -q main
  echo "allowlisted change" > base.txt
  git add base.txt
  git commit -q -m "allowlisted change"
  echo "junk" > untracked-not-allowed.txt
)

ALLOWLIST="$WORK/gate3-allowlist.txt"
cat > "$ALLOWLIST" <<'EOF'
base.txt
# lane working state — not counted as unnecessary
.aether/data/**
EOF

GATE3_JSON="$(git_cleanliness "$FIXTURE_REPO" main "$BASE_SHA" "$ALLOWLIST")" || fail "git_cleanliness failed against the fixture repo"

ACTUAL_ORPHAN_BRANCHES="$(jq -r '.orphan_branches' <<<"$GATE3_JSON")"
ACTUAL_UNNECESSARY_MODS="$(jq -r '.unnecessary_modifications' <<<"$GATE3_JSON")"
ACTUAL_CLEANLINESS_SCORE="$(jq -r '.cleanliness_score' <<<"$GATE3_JSON")"

[ "$ACTUAL_ORPHAN_BRANCHES" = "1" ] \
  || fail "gate 3: orphan_branches expected=1 actual=$ACTUAL_ORPHAN_BRANCHES"
[ "$ACTUAL_UNNECESSARY_MODS" -ge "1" ] \
  || fail "gate 3: unnecessary_modifications expected>=1 actual=$ACTUAL_UNNECESSARY_MODS"
[ "$ACTUAL_CLEANLINESS_SCORE" -lt "100" ] \
  || fail "gate 3: cleanliness_score expected<100 actual=$ACTUAL_CLEANLINESS_SCORE"
[ "$ACTUAL_CLEANLINESS_SCORE" -gt "0" ] \
  || fail "gate 3: cleanliness_score expected>0 actual=$ACTUAL_CLEANLINESS_SCORE"

echo "    gate 3 OK: orphan_branches=$ACTUAL_ORPHAN_BRANCHES unnecessary_modifications=$ACTUAL_UNNECESSARY_MODS cleanliness_score=$ACTUAL_CLEANLINESS_SCORE"

# --- gate 4: operator log --------------------------------------------------
step "gate 4: operator log — interventions and derived autonomy"

OPLOG_A="$WORK/gate4-a.jsonl"
(
  export OPERATOR_LOG="$OPLOG_A"
  oplog_start selftest-run-a aether-interactive small-bug fixture-repo test-model
  oplog_input scripted "approve step 1"
  oplog_wait_start
  oplog_wait_end
  oplog_input scripted "approve step 2"
  oplog_input unscripted "manual fix"
  oplog_end success
) || fail "gate 4: synthetic sequence A (with one unscripted input) failed to log"

GATE4A_JSON="$(oplog_summary "$OPLOG_A")" || fail "gate 4: oplog_summary failed on sequence A"
ACTUAL_SCRIPTED="$(jq -r '.scripted_inputs' <<<"$GATE4A_JSON")"
ACTUAL_UNSCRIPTED="$(jq -r '.unscripted_inputs' <<<"$GATE4A_JSON")"
ACTUAL_AUTONOMOUS_A="$(jq -r '.autonomous' <<<"$GATE4A_JSON")"

[ "$ACTUAL_SCRIPTED" = "2" ] \
  || fail "gate 4: scripted_inputs expected=2 actual=$ACTUAL_SCRIPTED"
[ "$ACTUAL_UNSCRIPTED" = "1" ] \
  || fail "gate 4: unscripted_inputs expected=1 actual=$ACTUAL_UNSCRIPTED"
[ "$ACTUAL_AUTONOMOUS_A" = "false" ] \
  || fail "gate 4: autonomous expected=false actual=$ACTUAL_AUTONOMOUS_A"

OPLOG_B="$WORK/gate4-b.jsonl"
(
  export OPERATOR_LOG="$OPLOG_B"
  oplog_start selftest-run-b aether-interactive small-bug fixture-repo test-model
  oplog_input scripted "approve step 1"
  oplog_end success
) || fail "gate 4: synthetic sequence B (zero unscripted) failed to log"

GATE4B_JSON="$(oplog_summary "$OPLOG_B")" || fail "gate 4: oplog_summary failed on sequence B"
ACTUAL_AUTONOMOUS_B="$(jq -r '.autonomous' <<<"$GATE4B_JSON")"

[ "$ACTUAL_AUTONOMOUS_B" = "true" ] \
  || fail "gate 4: sequence B autonomous expected=true actual=$ACTUAL_AUTONOMOUS_B"

echo "    gate 4 OK: sequence A scripted=$ACTUAL_SCRIPTED unscripted=$ACTUAL_UNSCRIPTED autonomous=$ACTUAL_AUTONOMOUS_A; sequence B autonomous=$ACTUAL_AUTONOMOUS_B"

# --- gate 5: real-home refusal ---------------------------------------------
step "gate 5: measure_tokens refuses the operator's real home"

if measure_tokens "$HOME" >/dev/null 2>"$WORK/gate5.err"; then
  fail "gate 5: measure_tokens against \$HOME was expected to exit non-zero but exited 0"
fi
echo "    gate 5 OK: $(cat "$WORK/gate5.err")"

echo
echo "BENCH SELFTEST PASS"
