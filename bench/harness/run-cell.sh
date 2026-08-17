#!/usr/bin/env bash
# run-cell.sh — runs exactly one lane x one category of the Aether-vs-GSD
# benchmark: fresh clone, isolated HOME, run, measure, judge, record.
#
# Usage: run-cell.sh LANE CATEGORY OUTPUT_DIR
#   LANE        — one of: gsd, aether-interactive, aether-autopilot
#   CATEGORY    — one of: 01-bug-fix, 02-brownfield-feature,
#                 03-interrupted-execution, 04-fresh-repo-lifecycle
#   OUTPUT_DIR  — a dated results directory (e.g. bench/results/2026-08-17)
#
# Requires BENCH_MODEL to be set — an unpinned model silently breaks the
# harness's model-parity claim, so this script refuses to run without it.
#
# Structure follows scripts/smoke-daily-driver.sh: strict mode, ROOT, a
# mktemp -d WORK dir, trap cleanup with chmod -R u+w before rm -rf, tool
# preflight, numbered step stages.
#
# The cell runner never computes a metric from anything either system said
# about itself, with the single exception of the claimed-success flag —
# which exists precisely to be contradicted by the acceptance script.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
WORK="$(mktemp -d)"
trap 'chmod -R u+w "$WORK" 2>/dev/null || true; rm -rf "$WORK"' EXIT

fail() { echo "RUN-CELL FAIL: $*" >&2; exit 1; }
step() { echo; echo "==> $*"; }

for tool in go node npm jq git; do
  command -v "$tool" >/dev/null || fail "required tool missing: $tool"
done

# --- Stage 1: validate arguments ---------------------------------------
LANE="${1:-}"
CATEGORY="${2:-}"
OUTPUT_DIR="${3:-}"

case "$LANE" in
  gsd|aether-interactive|aether-autopilot) ;;
  *) fail "unknown lane: '$LANE' — must be one of gsd, aether-interactive, aether-autopilot" ;;
esac

case "$CATEGORY" in
  01-bug-fix|02-brownfield-feature|03-interrupted-execution|04-fresh-repo-lifecycle) ;;
  *) fail "unknown category: '$CATEGORY' — must be one of 01-bug-fix, 02-brownfield-feature, 03-interrupted-execution, 04-fresh-repo-lifecycle" ;;
esac

[ -n "$OUTPUT_DIR" ] || fail "OUTPUT_DIR argument is required"

[ -n "${BENCH_MODEL:-}" ] || fail "BENCH_MODEL is not set — an unpinned model silently breaks the parity claim between lanes"
MODEL="$BENCH_MODEL"

REAL_HOME="${REAL_HOME:-$HOME}"
export REAL_HOME

RUN_ID="${LANE}__${CATEGORY}"
CELL_OUT_DIR="$OUTPUT_DIR/$RUN_ID"
mkdir -p "$CELL_OUT_DIR"

# shellcheck source=bench/lib/measure-tokens.sh
source "$ROOT/bench/lib/measure-tokens.sh"
# shellcheck source=bench/lib/git-cleanliness.sh
source "$ROOT/bench/lib/git-cleanliness.sh"
# shellcheck source=bench/lib/operator-log.sh
source "$ROOT/bench/lib/operator-log.sh"

# --- Stage 2: fresh clone of the category's substrate at its pinned SHA -
SUBSTRATE="$WORK/substrate"
STARTING_BRANCH=""
STARTING_SHA=""

case "$CATEGORY" in
  01-bug-fix|03-interrupted-execution)
    step "fresh clone: gorilla/mux at pinned SHA"
    git clone https://github.com/gorilla/mux.git "$SUBSTRATE" >/dev/null 2>&1 \
      || fail "clone of gorilla/mux failed"
    (cd "$SUBSTRATE" && git checkout db9d1d0073d27a0a2d9a8c1bc52aa0af4374d265 >/dev/null 2>&1) \
      || fail "checkout of pinned gorilla/mux SHA failed"
    ;;
  02-brownfield-feature)
    step "fresh clone: colinhacks/zod at pinned SHA"
    git clone https://github.com/colinhacks/zod.git "$SUBSTRATE" >/dev/null 2>&1 \
      || fail "clone of colinhacks/zod failed"
    (cd "$SUBSTRATE" && git checkout ca42965df46b2f7e2747db29c40a26bcb32a51d5 >/dev/null 2>&1) \
      || fail "checkout of pinned zod SHA failed"
    ;;
  04-fresh-repo-lifecycle)
    step "creating an empty git repository (no substrate — this category starts from nothing)"
    mkdir -p "$SUBSTRATE"
    (cd "$SUBSTRATE" && git init -q) || fail "git init for fresh-repo-lifecycle failed"
    ;;
esac

STARTING_BRANCH="$(cd "$SUBSTRATE" && git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")"
STARTING_SHA="$(cd "$SUBSTRATE" && git rev-parse HEAD 2>/dev/null || echo "")"
echo "substrate starting_branch=$STARTING_BRANCH starting_sha=$STARTING_SHA"

# --- Stage 3: apply the category's Setup (seeded defect + failing test) -
case "$CATEGORY" in
  01-bug-fix|03-interrupted-execution)
    step "applying seeded defect + seeded failing test to mux.go / mux_test.go"
    python3 - "$SUBSTRATE/mux.go" <<'PYEOF' || fail "seeding the defect in mux.go failed"
import sys
path = sys.argv[1]
with open(path) as f:
    content = f.read()
old = "if p[len(p)-1] == '/' && np != \"/\" {"
new = "if p[0] == '/' && np != \"/\" {"
if old not in content:
    raise SystemExit("expected defect target line not found in mux.go")
content = content.replace(old, new, 1)
with open(path, "w") as f:
    f.write(content)
PYEOF
    cat >> "$SUBSTRATE/mux_test.go" <<'GOEOF'

// TestCleanPathPreservesNonSlashInput verifies that cleanPath does not append
// a trailing slash to a path that did not have one to begin with.
func TestCleanPathPreservesNonSlashInput(t *testing.T) {
	got := cleanPath("/foo/bar")
	want := "/foo/bar"
	if got != want {
		t.Fatalf("cleanPath(%q) = %q, want %q (a path with no trailing slash must not gain one)", "/foo/bar", got, want)
	}
}
GOEOF
    ;;
  02-brownfield-feature|04-fresh-repo-lifecycle)
    step "no Setup for this category — starting unmodified"
    ;;
esac

# --- Stage 4: lane_prepare + lane_force_model ---------------------------
# shellcheck source=bench/harness/run-lane.sh
source "$ROOT/bench/harness/run-lane.sh"

BIN="$WORK/bin/aether"
if [ "$LANE" = "aether-interactive" ] || [ "$LANE" = "aether-autopilot" ]; then
  step "building aether from source"
  mkdir -p "$WORK/bin"
  (cd "$ROOT" && go build -o "$BIN" ./cmd/aether) || fail "go build ./cmd/aether failed"
  export BENCH_AETHER_BIN="$BIN"
fi

step "lane_prepare: isolating HOME and installing $LANE"
lane_prepare "$LANE" "$WORK"

step "lane_force_model: pinning model=$MODEL for $LANE"
MODEL_DEVIATION_LINE=""
FORCE_MODEL_OUT="$(lane_force_model "$LANE" "$MODEL")"
echo "$FORCE_MODEL_OUT"
if printf '%s\n' "$FORCE_MODEL_OUT" | grep -q '^MODEL-DEVIATION'; then
  MODEL_DEVIATION_LINE="$(printf '%s\n' "$FORCE_MODEL_OUT" | grep '^MODEL-DEVIATION' | head -1)"
fi

# --- Stage 5: operator log start + print permitted inputs ---------------
OPERATOR_LOG="$WORK/operator-log.jsonl"
export OPERATOR_LOG
: > "$OPERATOR_LOG"

SUBSTRATE_REPO_LABEL="empty-repo"
case "$CATEGORY" in
  01-bug-fix|03-interrupted-execution) SUBSTRATE_REPO_LABEL="gorilla/mux@$STARTING_SHA" ;;
  02-brownfield-feature) SUBSTRATE_REPO_LABEL="colinhacks/zod@$STARTING_SHA" ;;
  04-fresh-repo-lifecycle) SUBSTRATE_REPO_LABEL="empty-repo" ;;
esac

oplog_start "$RUN_ID" "$LANE" "$CATEGORY" "$SUBSTRATE_REPO_LABEL" "$MODEL"

step "permitted operator inputs for this cell (see bench/harness/permitted-inputs.md for the full list)"
PERMITTED_INPUTS_FILE="$ROOT/bench/harness/permitted-inputs.md"
if [ -f "$PERMITTED_INPUTS_FILE" ]; then
  awk -v lane="$LANE" -v category="$CATEGORY" '
    $0 ~ "^## " lane " . " category "$" || $0 ~ "^## " lane "-" category "$" { in_section = 1; print; next }
    in_section && /^## / { in_section = 0 }
    in_section { print }
  ' "$PERMITTED_INPUTS_FILE"
else
  echo "(bench/harness/permitted-inputs.md not found — nothing to print yet)"
fi

# --- Stage 6: category 03 only — background SIGKILL watcher -------------
WATCHER_PID=""
if [ "$CATEGORY" = "03-interrupted-execution" ]; then
  step "starting the interrupted-execution watcher (first file write + 120s -> SIGKILL to the run's process group)"
  MARKER="$WORK/.watch-marker"
  touch "$MARKER"
  (
    # Kill rule, identical for all three lanes: the moment of the first file
    # write inside the substrate clone starts a 120-second timer; when that
    # timer expires, SIGKILL is sent to the run's entire process group so no
    # child process survives to keep working unsupervised.
    while true; do
      FIRST_WRITE="$(find "$SUBSTRATE" -newer "$MARKER" -type f -not -path '*/.git/*' 2>/dev/null | head -1)"
      if [ -n "$FIRST_WRITE" ]; then
        oplog_event first_file_write "{\"path\":\"$FIRST_WRITE\"}" || true
        sleep 120
        oplog_event sigkill_sent '{}' || true
        if [ -n "${RUN_PGID:-}" ]; then
          kill -SIGKILL -- "-${RUN_PGID}" 2>/dev/null || true
        fi
        break
      fi
      sleep 1
    done
  ) &
  WATCHER_PID=$!
fi

# --- Stage 7: hand control to the operator -------------------------------
step "operator turn: run the commands below for lane=$LANE category=$CATEGORY"
lane_invocation "$LANE" "$CATEGORY"

oplog_wait_start
echo
echo "Press ENTER once the operator has finished (or been interrupted per the kill rule above) to resume the harness."
if [ -t 0 ]; then
  read -r _ || true
else
  echo "(non-interactive stdin — skipping operator wait; this is expected under --dry-run or automated exercise)"
fi
oplog_wait_end

if [ -n "$WATCHER_PID" ]; then
  kill "$WATCHER_PID" 2>/dev/null || true
  wait "$WATCHER_PID" 2>/dev/null || true
fi

oplog_end "completed"

# --- Stage 8: measurement, all by script ---------------------------------
step "measuring tokens (harness-side, from the isolated HOME's session transcripts)"
TOKENS_JSON="$WORK/tokens.json"
if measure_tokens "$HOME" > "$TOKENS_JSON" 2>"$WORK/tokens.err"; then
  echo "    tokens measured OK"
else
  echo "    WARNING: measure_tokens failed — $(cat "$WORK/tokens.err")" >&2
  echo '{"total_tokens":null,"total_input_tokens":null,"model_calls":null,"models":"","source":"unavailable"}' > "$TOKENS_JSON"
fi

step "measuring git cleanliness against the substrate clone"
ALLOWLIST="$ROOT/bench/tasks/allowlists/${CATEGORY}.txt"
CLEANLINESS_JSON="$WORK/cleanliness.json"
if [ -f "$ALLOWLIST" ] && [ -n "$STARTING_SHA" ]; then
  git_cleanliness "$SUBSTRATE" "$STARTING_BRANCH" "$STARTING_SHA" "$ALLOWLIST" > "$CLEANLINESS_JSON" \
    || fail "git_cleanliness failed for $SUBSTRATE"
else
  echo '{"orphan_branches":null,"cleanliness_score":null}' > "$CLEANLINESS_JSON"
fi

step "summarizing the operator log"
OPLOG_SUMMARY_JSON="$WORK/oplog-summary.json"
oplog_summary "$OPERATOR_LOG" > "$OPLOG_SUMMARY_JSON" || fail "oplog_summary failed"

# --- Stage 9: judgement against a FRESH clone of the resulting state -----
step "judging: fresh clone of the substrate's resulting state, run against the acceptance script"
JUDGE_DIR="$WORK/judge"
if [ "$CATEGORY" = "04-fresh-repo-lifecycle" ]; then
  cp -R "$SUBSTRATE" "$JUDGE_DIR"
  export AETHER_REPO="$ROOT"
else
  git clone "$SUBSTRATE" "$JUDGE_DIR" >/dev/null 2>&1 || fail "judge clone of $SUBSTRATE failed"
fi

ACCEPTANCE_SCRIPT="$ROOT/bench/acceptance/${CATEGORY}.sh"
[ -x "$ACCEPTANCE_SCRIPT" ] || fail "acceptance script not found or not executable: $ACCEPTANCE_SCRIPT"

ACCEPTANCE_JSON="$WORK/acceptance.json"
export ACCEPTANCE_JSON
ACCEPTANCE_STATUS=0
ACCEPTANCE_OUT="$("$ACCEPTANCE_SCRIPT" "$JUDGE_DIR" 2>&1)" || ACCEPTANCE_STATUS=$?
echo "$ACCEPTANCE_OUT"
[ -f "$ACCEPTANCE_JSON" ] || echo "{}" > "$ACCEPTANCE_JSON"

# --- Stage 10: hallucinated-completion detection --------------------------
# Both conditions must hold: the system claimed success (from a terminal
# transcript the operator saved per the runbook) AND the acceptance script
# failed. If the claim file is absent, record "unknown" rather than
# "false" — an absent input must never be silently favourable.
CLAIM_FILE="$WORK/claimed-success.txt"
HALLUCINATION_FLAG="unknown"
if [ -f "$CLAIM_FILE" ]; then
  CLAIMED_SUCCESS_TEXT="$(cat "$CLAIM_FILE")"
  if [ "$ACCEPTANCE_STATUS" -ne 0 ] && printf '%s' "$CLAIMED_SUCCESS_TEXT" | grep -qi 'success\|complete\|done'; then
    HALLUCINATION_FLAG="true"
  else
    HALLUCINATION_FLAG="false"
  fi
else
  HALLUCINATION_FLAG="unknown"
fi

# --- Stage 11: write run.json and copy raw evidence before the trap fires -
step "writing run.json and copying raw evidence to $CELL_OUT_DIR"

TRANSCRIPT_DEST="$CELL_OUT_DIR/transcripts"
mkdir -p "$TRANSCRIPT_DEST"
if [ -d "$HOME/.claude/projects" ]; then
  cp -R "$HOME/.claude/projects" "$TRANSCRIPT_DEST/" 2>/dev/null || true
fi
cp "$OPERATOR_LOG" "$CELL_OUT_DIR/operator-log.jsonl" 2>/dev/null || true
cp "$ACCEPTANCE_JSON" "$CELL_OUT_DIR/acceptance.json" 2>/dev/null || true
[ -f "$CLAIM_FILE" ] && cp "$CLAIM_FILE" "$CELL_OUT_DIR/claimed-success.txt" 2>/dev/null || true

RUN_JSON="$CELL_OUT_DIR/run.json"
jq -n \
  --arg lane "$LANE" \
  --arg category "$CATEGORY" \
  --arg repo "$SUBSTRATE_REPO_LABEL" \
  --arg starting_sha "$STARTING_SHA" \
  --arg model "$MODEL" \
  --arg model_deviation "$MODEL_DEVIATION_LINE" \
  --arg parallel_mode "in-repo" \
  --arg parallel_mode_note "worktree mode is disqualified until Phase 187" \
  --argjson acceptance_status "$ACCEPTANCE_STATUS" \
  --argjson acceptance "$(cat "$ACCEPTANCE_JSON")" \
  --argjson tokens "$(cat "$TOKENS_JSON")" \
  --argjson cleanliness "$(cat "$CLEANLINESS_JSON")" \
  --argjson oplog_summary "$(cat "$OPLOG_SUMMARY_JSON")" \
  --arg hallucinated_completion "$HALLUCINATION_FLAG" \
  --arg transcripts_dir "transcripts" \
  --arg operator_log_path "operator-log.jsonl" \
  --arg acceptance_json_path "acceptance.json" \
  '{
    lane: $lane,
    category: $category,
    repo: $repo,
    starting_sha: $starting_sha,
    model: $model,
    model_deviation: (if $model_deviation == "" then null else $model_deviation end),
    parallel_mode: $parallel_mode,
    parallel_mode_note: $parallel_mode_note,
    acceptance_pass: ($acceptance_status == 0),
    acceptance_status: $acceptance_status,
    acceptance: $acceptance,
    tokens: $tokens,
    cleanliness: $cleanliness,
    operator: $oplog_summary,
    hallucinated_completion: $hallucinated_completion,
    evidence: {
      transcripts_dir: $transcripts_dir,
      operator_log_path: $operator_log_path,
      acceptance_json_path: $acceptance_json_path
    }
  }' > "$RUN_JSON"

echo
echo "RUN-CELL COMPLETE: $RUN_ID -> $RUN_JSON (acceptance_pass=$([ "$ACCEPTANCE_STATUS" -eq 0 ] && echo true || echo false))"
