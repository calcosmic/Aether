#!/usr/bin/env bash
# capture-print-brief.sh — takes a committed, read-only snapshot of exactly
# what one of Aether's helper agents ("workers") is told before it starts
# work on a real, in-progress project ("colony").
#
# What it proves, not just what it prints: the inspection command being
# captured (`aether build <phase> --print-brief --full`) is documented as
# read-only, but this repo has a history of --dry-run flags that mutated
# state anyway. So this script fingerprints the target colony's saved state
# before and after the capture and refuses to write anything if the
# fingerprint changed even by one byte.
#
# Usage:
#   capture-print-brief.sh <colony-repo> <phase-number> [worker-name]
#   capture-print-brief.sh --check <colony-repo> <phase-number> [worker-name]
#
# <colony-repo> must be a real, already-in-progress project on this machine
# (it must already have .aether/data/COLONY_STATE.json). This script never
# creates or advances a colony — a colony it manufactured would be a
# fixture, not evidence of what a real worker sees.
#
# [worker-name] is optional and passes straight through to the real
# command's own --worker flag, scoping the capture to one named worker's
# brief instead of every worker planned for the phase.
#
# Exit codes: 0 on success (or, in --check mode, on "unchanged"); non-zero
# with a named reason on any failure, including "the composition changed"
# in --check mode.
#
# Requires: go, shasum, git.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
EVIDENCE_DIR="$ROOT/bench/evidence"
CAPTURE_FILE="$EVIDENCE_DIR/print-brief-capture.txt"

fail() { echo "CAPTURE FAIL: $*" >&2; exit 1; }
step() { echo; echo "==> $*"; }

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
CHECK_MODE=false
if [ "${1:-}" = "--check" ]; then
  CHECK_MODE=true
  shift
fi

COLONY_REPO="${1:-}"
PHASE_NUM="${2:-}"
WORKER_NAME="${3:-}"

[ -n "$COLONY_REPO" ] || fail "usage: capture-print-brief.sh [--check] <colony-repo> <phase-number> [worker-name]"
[ -n "$PHASE_NUM" ] || fail "usage: capture-print-brief.sh [--check] <colony-repo> <phase-number> [worker-name]"

case "$PHASE_NUM" in
  ''|*[!0-9]*) fail "phase number must be a positive integer, got '$PHASE_NUM'" ;;
esac

COLONY_REPO="$(cd "$COLONY_REPO" 2>/dev/null && pwd)" || fail "colony repo path does not exist: ${1:-}"
STATE_FILE="$COLONY_REPO/.aether/data/COLONY_STATE.json"

[ -f "$STATE_FILE" ] || fail "no colony found at $COLONY_REPO — expected $STATE_FILE. Point this at a repo with a real, already-initialized Aether project."

for tool in go shasum git; do
  command -v "$tool" >/dev/null || fail "required tool missing: $tool"
done

WORK="$(mktemp -d)"
trap 'chmod -R u+w "$WORK" 2>/dev/null || true; rm -rf "$WORK"' EXIT

# Pin Go caches to the real ones so this build reuses the module cache.
export GOPATH="$(go env GOPATH)"
export GOMODCACHE="$(go env GOMODCACHE)"
export GOCACHE="$(go env GOCACHE)"

# ---------------------------------------------------------------------------
# Before/after fingerprint of the colony's saved state
# ---------------------------------------------------------------------------
snapshot_colony_state() {
  local out="$1"
  {
    shasum -a 256 "$STATE_FILE"
    # .cache_* files are excluded on purpose: they are a documented,
    # read-side parse cache (pkg/cache/session_cache.go), keyed by the
    # source file's own mtime, that the runtime is allowed to write on any
    # read. They never change what the colony believes — they only avoid
    # re-parsing JSON that has not changed. Counting them as "colony state"
    # would fail this proof on every run, including runs that changed
    # nothing real.
    { find "$COLONY_REPO/.aether/data" -type f ! -name '.cache_*' -exec stat -f '%N %m %z' {} \; 2>/dev/null \
      || find "$COLONY_REPO/.aether/data" -type f ! -name '.cache_*' -printf '%p %T@ %s\n' 2>/dev/null ; }
  } | sort > "$out"
}

step "recording before-snapshot of colony state ($STATE_FILE)"
BEFORE_SNAPSHOT="$WORK/before.snapshot"
snapshot_colony_state "$BEFORE_SNAPSHOT"
BEFORE_CHECKSUM="$(shasum -a 256 "$STATE_FILE" | awk '{print $1}')"

# ---------------------------------------------------------------------------
# Build the binary from this repo's current source — the capture is of the
# code under test, not whatever happens to be installed on this machine.
# ---------------------------------------------------------------------------
step "building aether from source"
BIN="$WORK/aether"
(cd "$ROOT" && go build -o "$BIN" ./cmd/aether) || fail "go build failed"
BINARY_VERSION="$("$BIN" version 2>&1 | head -1 || true)"
echo "    binary version: $BINARY_VERSION"

# ---------------------------------------------------------------------------
# Run the real, read-only inspector command from the colony repo.
# ---------------------------------------------------------------------------
WORKER_ARGS=()
if [ -n "$WORKER_NAME" ]; then
  WORKER_ARGS=(--worker "$WORKER_NAME")
fi

step "running aether build $PHASE_NUM --print-brief --full${WORKER_NAME:+ --worker $WORKER_NAME}"
CAPTURE_OUT="$WORK/capture.out"
CAPTURE_ERR="$WORK/capture.err"
set +e
(cd "$COLONY_REPO" && "$BIN" build "$PHASE_NUM" --print-brief --full "${WORKER_ARGS[@]}" >"$CAPTURE_OUT" 2>"$CAPTURE_ERR")
CAPTURE_STATUS=$?
set -e

if [ "$CAPTURE_STATUS" -ne 0 ]; then
  sed -n '1,20p' "$CAPTURE_ERR" >&2
  fail "aether build $PHASE_NUM --print-brief --full exited $CAPTURE_STATUS against $COLONY_REPO — point this at a real colony with that phase number in range"
fi

[ -s "$CAPTURE_OUT" ] || fail "print-brief produced no output"

# ---------------------------------------------------------------------------
# After-snapshot and read-only assertion
# ---------------------------------------------------------------------------
step "recording after-snapshot of colony state"
AFTER_SNAPSHOT="$WORK/after.snapshot"
snapshot_colony_state "$AFTER_SNAPSHOT"
AFTER_CHECKSUM="$(shasum -a 256 "$STATE_FILE" | awk '{print $1}')"

if [ "$BEFORE_CHECKSUM" != "$AFTER_CHECKSUM" ] || ! diff -q "$BEFORE_SNAPSHOT" "$AFTER_SNAPSHOT" >/dev/null 2>&1; then
  diff "$BEFORE_SNAPSHOT" "$AFTER_SNAPSHOT" >&2 || true
  fail "print-brief mutated colony state — the inspection command is not read-only"
fi
echo "    read-only proof OK: COLONY_STATE.json checksum unchanged ($BEFORE_CHECKSUM)"
echo "    read-only proof OK: .aether/data/ listing unchanged"

# ---------------------------------------------------------------------------
# Secret-shape scan — the capture is going into git, so refuse to write it
# if anything looks like a live credential.
# ---------------------------------------------------------------------------
step "scanning capture for secret-shaped strings before writing"
SECRET_PATTERN='\bsk-[A-Za-z0-9]{20,}|ghp_[A-Za-z0-9]{20,}|AKIA[A-Z0-9]{12,}|-----BEGIN'
if grep -nE "$SECRET_PATTERN" "$CAPTURE_OUT" >"$WORK/secret-hits.txt" 2>/dev/null; then
  cat "$WORK/secret-hits.txt" >&2
  fail "capture contains secret-shaped strings at the line numbers above — refusing to commit them"
fi
echo "    no secret-shaped strings found"

# ---------------------------------------------------------------------------
# --check mode: compare against the committed capture, never overwrite it.
#
# One line in the raw prompt is expected to differ on every run no matter
# what: "- Started: <RFC3339 timestamp>" — the wall-clock moment the brief
# was assembled. That is not a composition change, so both sides of the
# comparison have that one volatile line normalized out before diffing.
# ---------------------------------------------------------------------------
if [ "$CHECK_MODE" = true ]; then
  [ -f "$CAPTURE_FILE" ] || fail "--check requires an existing committed capture at $CAPTURE_FILE — run without --check first"

  NORMALIZE_STARTED='s/^- Started: .*/- Started: <normalized>/'
  OLD_NORMALIZED="$WORK/old-normalized.txt"
  NEW_NORMALIZED="$WORK/new-normalized.txt"
  sed -E "$NORMALIZE_STARTED" "$CAPTURE_FILE" > "$OLD_NORMALIZED"
  sed -E "$NORMALIZE_STARTED" "$CAPTURE_OUT" > "$NEW_NORMALIZED"

  if diff -q "$OLD_NORMALIZED" "$NEW_NORMALIZED" >/dev/null 2>&1; then
    echo
    echo "CAPTURE CHECK: unchanged ($(wc -l < "$CAPTURE_FILE" | tr -d ' ') lines, identical to committed capture apart from the capture timestamp)"
    exit 0
  fi

  OLD_LINES="$(wc -l < "$CAPTURE_FILE" | tr -d ' ')"
  NEW_LINES="$(wc -l < "$CAPTURE_OUT" | tr -d ' ')"
  OLD_HEADINGS="$WORK/old-headings.txt"
  NEW_HEADINGS="$WORK/new-headings.txt"
  grep -E '^  [A-Za-z].*[0-9]+ +[0-9.]+%' "$CAPTURE_FILE" | sed -E 's/^  ([A-Za-z][^ ].*[A-Za-z0-9)])[ ]+[0-9].*$/\1/' | sed -E 's/ +$//' > "$OLD_HEADINGS" 2>/dev/null || true
  grep -E '^  [A-Za-z].*[0-9]+ +[0-9.]+%' "$CAPTURE_OUT" | sed -E 's/^  ([A-Za-z][^ ].*[A-Za-z0-9)])[ ]+[0-9].*$/\1/' | sed -E 's/ +$//' > "$NEW_HEADINGS" 2>/dev/null || true

  echo >&2
  echo "CAPTURE CHECK: composition changed" >&2
  echo "  committed capture: $OLD_LINES lines" >&2
  echo "  fresh capture:      $NEW_LINES lines" >&2
  echo "  section headings that differ:" >&2
  diff "$OLD_HEADINGS" "$NEW_HEADINGS" >&2 || true
  fail "print-brief composition changed since the committed baseline — see diff above"
fi

# ---------------------------------------------------------------------------
# Write the capture (only in non-check mode, only after every gate passed).
# ---------------------------------------------------------------------------
mkdir -p "$EVIDENCE_DIR"
cp "$CAPTURE_OUT" "$CAPTURE_FILE"
echo
echo "CAPTURE PASS: wrote $CAPTURE_FILE ($(wc -l < "$CAPTURE_FILE" | tr -d ' ') lines)"
