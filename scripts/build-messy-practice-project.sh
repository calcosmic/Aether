#!/usr/bin/env bash
# build-messy-practice-project.sh — 207-01-PLAN.md Task 1 (UED-07).
#
# Builds a real, Aether-initialised "messy" practice project in a
# destination directory, seeded with the traps declared in
# cmd/testdata/journey/traps.json -- the one trap list this script AND
# cmd/journey_traps.go both read, so the two can never disagree about
# which traps exist.
#
# ISOLATION RULE, followed throughout this script: nothing here writes to
# the real $HOME/.claude/ or the real ~/.aether/ hub. The hub is isolated
# via AETHER_HUB_DIR, pointed inside the destination directory. HOME itself
# is deliberately left ALONE (never overridden) -- the same rule
# scripts/proof-screens-reach-the-owner.sh already follows -- because
# nothing in this script needs to read or write through it.
#
# Safely re-runnable: running this script a second time against the same
# destination re-applies and re-verifies every declared trap without
# deleting anything it did not create. A destination that already contains
# files but carries no marker (.journey-practice-project.json) from a
# prior run of this script is refused BY NAME -- this script never
# silently adopts a directory something else built.
#
# Exits non-zero on the first failure, naming what failed. Requires: go,
# git, jq.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

fail() { echo "BUILD FAIL: $*" >&2; exit 1; }
step() { echo; echo "==> $*"; }

for tool in go git jq; do
  command -v "$tool" >/dev/null 2>&1 || fail "required tool missing: $tool"
done

# --- Trap construction functions -----------------------------------------
#
# One function per declared trap id. A declared id with no matching
# function here is a "cannot build" case, handled explicitly below -- the
# manifest and this script can never silently drift apart.

# apply_trap_leftover_junk_data seeds .aether/data/midden.json (the
# colony's failure log) with at least two unacknowledged entries, in the
# exact shape pkg/colony.MiddenFile / pkg/colony.MiddenEntry require
# (derived from pkg/colony/midden.go, read before writing this function --
# never a plausible-looking guess). Idempotent: merges by id, never
# duplicates or overwrites an existing entry.
apply_trap_leftover_junk_data() {
  local repo="$1"
  local midden="$repo/.aether/data/midden.json"
  mkdir -p "$repo/.aether/data"
  if [ ! -f "$midden" ]; then
    echo '{"version":"1","signals":[],"entries":[]}' > "$midden"
  fi
  jq '
    (.entries // []) as $entries
    | ($entries | map(.id)) as $have
    | ["journey-trap-leftover-junk-data-1","journey-trap-leftover-junk-data-2"] as $want
    | .entries = ($entries + ([$want[] | select(. as $id | ($have | index($id)) == null)]
        | map({
            id: .,
            timestamp: "2026-09-22T00:00:00Z",
            category: "journey-trap",
            source: "build-messy-practice-project.sh",
            message: "seeded leftover junk data trap (UED-07, blocker #6)",
            reviewed: false,
            acknowledged: false
          })))
  ' "$midden" > "$midden.tmp" && mv "$midden.tmp" "$midden"
}

# --- Argument parsing ------------------------------------------------------

DEST=""
AETHER_BIN_OVERRIDE=""
while [ $# -gt 0 ]; do
  case "$1" in
    --aether-bin)
      AETHER_BIN_OVERRIDE="${2:-}"
      [ -n "$AETHER_BIN_OVERRIDE" ] || fail "--aether-bin requires a path argument"
      shift 2
      ;;
    -*)
      fail "unknown flag: $1"
      ;;
    *)
      if [ -z "$DEST" ]; then
        DEST="$1"
        shift
      else
        fail "unexpected extra argument: $1"
      fi
      ;;
  esac
done

[ -n "$DEST" ] || fail "usage: $0 <destination-directory> [--aether-bin <path>]"

MARKER_SCHEMA_VERSION="journey-practice-project/v1"

if [ -d "$DEST" ] && [ -n "$(ls -A "$DEST" 2>/dev/null || true)" ] && [ ! -f "$DEST/.journey-practice-project.json" ]; then
  fail "destination $DEST already contains files and carries no .journey-practice-project.json from a prior run of this script -- refusing to build into a directory this script did not create"
fi

mkdir -p "$DEST"
DEST="$(cd "$DEST" && pwd)"
MARKER="$DEST/.journey-practice-project.json"

TRAPS_JSON="$ROOT/cmd/testdata/journey/traps.json"
[ -f "$TRAPS_JSON" ] || fail "trap manifest missing: $TRAPS_JSON"
TRAP_IDS="$(jq -r '.traps[].id' "$TRAPS_JSON")"
[ -n "$TRAP_IDS" ] || fail "trap manifest $TRAPS_JSON declares no traps"

# --- Resolve the aether binary under test ---------------------------------

step "resolving the aether binary under test"
if [ -n "$AETHER_BIN_OVERRIDE" ]; then
  BIN="$AETHER_BIN_OVERRIDE"
  [ -x "$BIN" ] || fail "--aether-bin path is not executable: $BIN"
else
  BIN="$DEST/.journey-bin/aether"
  if [ ! -x "$BIN" ]; then
    mkdir -p "$DEST/.journey-bin"
    (cd "$ROOT" && go build -o "$BIN" ./cmd/aether) || fail "go build failed"
  fi
fi

# Isolate the hub only. HOME is intentionally untouched -- see the
# ISOLATION RULE above.
export AETHER_HUB_DIR="$DEST/.journey-hub"
mkdir -p "$AETHER_HUB_DIR"

step "installing package into isolated hub"
(cd "$ROOT" && "$BIN" install --package-dir "$ROOT" >/dev/null) || fail "aether install failed"

# --- Construct the practice project ----------------------------------------

REPO="$DEST/repo"
if [ ! -d "$REPO/.git" ]; then
  step "constructing the practice project"
  mkdir -p "$REPO"
  git init -q "$REPO"
  git -C "$REPO" config user.email "journey@example.com"
  git -C "$REPO" config user.name "journey"

  cat > "$REPO/README.md" <<'EOF'
# Messy Practice Project

A scratch project built by scripts/build-messy-practice-project.sh to
exercise Aether's full daily lifecycle against a real, imperfect codebase.
Nothing in this project is meant to ship.
EOF

  mkdir -p "$REPO/src"
  cat > "$REPO/src/main.go" <<'EOF'
package main

func main() {}
EOF
  cat > "$REPO/src/util.go" <<'EOF'
package main

func noop() {}
EOF

  git -C "$REPO" add -A
  git -C "$REPO" commit -q -m "initial commit" || fail "initial commit in the practice project failed"

  step "aether init in the practice project"
  (cd "$REPO" && "$BIN" init "Practice the daily lifecycle on a messy real project" >/dev/null 2>&1) \
    || fail "aether init failed in the practice project"

  step "aether update --force in the practice project (the install path under test)"
  (cd "$REPO" && "$BIN" update --force >/dev/null) || fail "aether update --force failed in the practice project"
else
  step "practice project already exists at $REPO -- re-applying and re-verifying traps only"
fi

# --- Apply every declared trap ---------------------------------------------

step "applying declared traps"
APPLIED_IDS=""
while IFS= read -r trap_id; do
  [ -z "$trap_id" ] && continue
  case "$trap_id" in
    leftover-junk-data)
      apply_trap_leftover_junk_data "$REPO"
      ;;
    *)
      fail "declared trap id has no known construction in this script: $trap_id"
      ;;
  esac
  APPLIED_IDS="${APPLIED_IDS}${APPLIED_IDS:+ }${trap_id}"
done <<<"$TRAP_IDS"

step "verifying declared traps landed"
jq -e '.entries != null and (.entries | map(select(.acknowledged == false)) | length) >= 2' \
  "$REPO/.aether/data/midden.json" >/dev/null \
  || fail "leftover-junk-data trap did not land -- .aether/data/midden.json has fewer than two unacknowledged entries"

# --- Write the marker and finish -------------------------------------------

jq -n --arg schema "$MARKER_SCHEMA_VERSION" --arg ids "$APPLIED_IDS" \
  '{schema_version: $schema, trap_ids: ($ids | split(" "))}' > "$MARKER"

echo
echo "BUILD PASS: practice project at $REPO (traps applied: $APPLIED_IDS)"
