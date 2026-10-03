#!/usr/bin/env bash
# build-messy-practice-project.sh — 207-01-PLAN.md Task 1, expanded to all
# nine traps by 207-02-PLAN.md Task 1 (UED-07).
#
# Builds a real, Aether-initialised "messy" practice project in a
# destination directory, seeded with every trap declared in
# cmd/testdata/journey/traps.json -- the one trap list this script AND
# cmd/journey_traps.go both read, so the two can never disagree about
# which traps exist.
#
# ISOLATION RULE, followed throughout this script: nothing here writes to
# the caller's home folder or the real ~/.aether/ hub. The hub is isolated
# via AETHER_HUB_DIR, pointed inside the destination directory. Every
# aether command runs with HOME pointed at a throwaway home folder inside
# the destination directory too (in_isolated_home), because `aether
# install` and `aether update --force` write the Claude, OpenCode and Codex
# command files into whatever HOME they are given -- and `aether install`
# never rebuilds a binary here (--skip-build-binary): the binary under test
# is already built. Before this, a `go test ./...` run replaced the owner's
# real ~/.local/bin/aether (WINDOWS.md entry 78). The caller's HOME is
# still left alone for everything else (git, jq), and for the chats a
# journey later drives, whose sign-in lives there.
#
# GIT DISCIPLINE, followed throughout this script: `git add -A` / `git add
# .` is NEVER used past the very first (pre-`aether init`) commit. Every
# later commit stages exactly the named path(s) a trap's own construction
# needs. `aether init`/`aether update --force` do not write a project
# .gitignore, so `.aether/` (and everything under it, including every trap
# this script seeds under `.aether/data/`) is untracked by construction --
# never committed, because it is never `git add`ed.
#
# Safely re-runnable: running this script a second time against the same
# destination re-applies and re-verifies every declared trap without
# deleting anything it did not create. Each trap's own apply function
# guards itself against being re-applied (checks what it already built
# before building it again), so a second run neither duplicates nor grows
# unbounded state. A destination that already contains files but carries no
# marker (.journey-practice-project.json) from a prior run of this script
# is refused BY NAME -- this script never silently adopts a directory
# something else built.
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

# cmd/survey_staleness.go's surveyStaleCommitThreshold. Duplicated here as a
# plain integer (this script has no Go import mechanism); if that constant
# ever changes, this trap still works correctly as long as this number stays
# >= the real threshold -- it only needs to be "enough", not exact.
OUT_OF_DATE_CODE_MAP_CHURN_COMMITS=27

# --- Trap construction functions -----------------------------------------
#
# One function per declared trap id. A declared id with no matching
# function here is a "cannot build" case, handled explicitly below -- the
# manifest and this script can never silently drift apart.

# apply_trap_shortcut_to_a_folder creates an uncommitted symbolic link
# ("shortcut-to-src") pointing at the real "src" directory. Never
# `git add`ed -- it survives every run as an unsaved change, which is
# exactly what the pause trap it exercises (blocker #5, b74d25d8) needs.
apply_trap_shortcut_to_a_folder() {
  local repo="$1"
  [ -L "$repo/shortcut-to-src" ] && return 0
  [ -d "$repo/src" ] || fail "shortcut-to-a-folder trap needs $repo/src to already exist"
  (cd "$repo" && ln -s src shortcut-to-src)
}

# apply_trap_shortcut_loop creates two symbolic links that point at each
# other. Never committed -- a robustness net, not tied to unsaved-changes
# specifically, but harmless left dirty either way.
apply_trap_shortcut_loop() {
  local repo="$1"
  if [ -L "$repo/loop-a" ] && [ -L "$repo/loop-b" ]; then
    return 0
  fi
  (cd "$repo" && ln -sf loop-b loop-a)
  (cd "$repo" && ln -sf loop-a loop-b)
}

# apply_trap_case_only_archive_collision writes two real files in DIFFERENT
# directories whose archive-relative names collide only by letter case --
# .aether/HANDOFF.md and .aether/data/handoff.md, exactly the two paths
# named in commit 57e563ba's own message. Deliberately NOT two siblings in
# one directory: a case-insensitive volume (the macOS default) would alias
# them and the trap would silently not exist. Both paths are already inside
# .aether/, which is never `git add`ed by this script, so no explicit
# uncommitted handling is needed here.
apply_trap_case_only_archive_collision() {
  local repo="$1"
  mkdir -p "$repo/.aether/data"
  [ -f "$repo/.aether/HANDOFF.md" ] || printf 'journey trap: case-only-archive-collision (upper path)\n' > "$repo/.aether/HANDOFF.md"
  [ -f "$repo/.aether/data/handoff.md" ] || printf 'journey trap: case-only-archive-collision (lower path)\n' > "$repo/.aether/data/handoff.md"
}

# apply_trap_nested_project builds a subdirectory that is its own git
# repository, with its own committed file. Never `git add`ed from the
# outer repository -- git reports a directory containing its own .git as
# one untracked entry, never descending into it.
apply_trap_nested_project() {
  local repo="$1"
  local nested="$repo/nested-project"
  [ -d "$nested/.git" ] && return 0
  mkdir -p "$nested"
  git init -q "$nested"
  git -C "$nested" config user.email "journey@example.com"
  git -C "$nested" config user.name "journey"
  printf 'A project nested inside the messy practice project (journey trap: nested-project).\n' > "$nested/nested.txt"
  git -C "$nested" add nested.txt
  git -C "$nested" commit -q -m "nested project initial commit"
}

# apply_trap_long_folder_names writes a real source file reachable only
# through a directory chain long enough that its "known-public-path-<path>"
# canonical lineage (colony.CanonicalSpecItemID) exceeds 40 characters --
# committed, since it is meant to be a genuine, permanent part of the
# project's own tracked source tree (the fix it exercises, specPublicPathLineage,
# is not one of the six 2026-09-21 blockers -- a bonus regression check).
apply_trap_long_folder_names() {
  local repo="$1"
  local rel="src/deeply/nested/directory/chain/intentionally/long/enough/to/exceed/the/forty/character/canonical/lineage/cap/file.go"
  [ -f "$repo/$rel" ] && return 0
  mkdir -p "$repo/$(dirname "$rel")"
  printf 'package cap\n\nfunc Marker() {}\n' > "$repo/$rel"
  git -C "$repo" add "$rel"
  git -C "$repo" commit -q -m "journey trap: long-folder-names" || fail "committing the long-folder-names trap file failed"
}

# apply_trap_out_of_date_code_map seeds a genuinely valid, digest-correct
# territory snapshot (via the aether binary's own hidden
# journey-seed-stale-survey command -- the runtime's own writer, not a
# hand-typed JSON literal, since the snapshot's digest is self-authenticating)
# pinned to the repository's revision BEFORE a batch of churn commits, so
# classifySurveyFreshness's own commit-count check genuinely classifies it
# stale (surveyStaleCommitThreshold commits or more since the pinned
# revision) -- never a fabricated digest mismatch.
apply_trap_out_of_date_code_map() {
  local repo="$1"
  local bin="$2"
  local snapshot="$repo/.aether/data/survey/territory-snapshot.json"
  [ -f "$snapshot" ] && return 0
  in_isolated_home "$bin" journey-seed-stale-survey "$repo" >/dev/null || fail "journey-seed-stale-survey failed"
  local i
  for i in $(seq 1 "$OUT_OF_DATE_CODE_MAP_CHURN_COMMITS"); do
    printf 'churn %s\n' "$i" >> "$repo/CHURN.md"
    git -C "$repo" add CHURN.md
    git -C "$repo" commit -q -m "journey trap: out-of-date-code-map churn commit $i" \
      || fail "out-of-date-code-map churn commit $i failed"
  done
}

# apply_trap_specification_corrected_mid_planning parks a planning run
# (stage=route_running, genuinely awaiting a worker) bound to an approved
# specification revision, then corrects and approves a successor -- via the
# aether binary's own hidden journey-seed-superseded-plan command, which
# calls the exact same specification writers `aether spec`/`aether discuss`
# use. The parked run's own stage-state.json is self-authenticating (it
# carries a real approval receipt hash); a hand-typed file could not pass
# that check, so this trap is never constructed by typing JSON.
apply_trap_specification_corrected_mid_planning() {
  local repo="$1"
  local bin="$2"
  local stage_state="$repo/.aether/data/planning/journey-trap-run/stage-state.json"
  [ -f "$stage_state" ] && return 0
  in_isolated_home "$bin" journey-seed-superseded-plan "$repo" >/dev/null || fail "journey-seed-superseded-plan failed"
}

# apply_trap_leftover_junk_data seeds .aether/data/midden.json (the
# colony's failure log) with at least two unacknowledged entries, in the
# exact shape pkg/colony.MiddenFile / pkg/colony.MiddenEntry require
# (derived from pkg/colony/midden.go, read before writing this function --
# never a plausible-looking guess), and .aether/data/pheromones.json (the
# colony's steering-signal log) with stray signals still marked active well
# past their own expiry -- leftover junk in both logs, per 207-02-PLAN.md
# Task 1's extension of this trap. Idempotent: merges by id, never
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

  local pheromones="$repo/.aether/data/pheromones.json"
  if [ ! -f "$pheromones" ]; then
    echo '{"signals":[]}' > "$pheromones"
  fi
  local text="journey trap: leftover junk data -- a stray signal nobody cleaned up"
  local content_hash
  content_hash="sha256:$(printf '%s' "$text" | shasum -a 256 | cut -d' ' -f1)"
  jq --arg hash "$content_hash" --arg text "$text" '
    (.signals // []) as $signals
    | ($signals | map(.id)) as $have
    | ["journey-trap-pheromone-1","journey-trap-pheromone-2"] as $want
    | .signals = ($signals + ([$want[] | select(. as $id | ($have | index($id)) == null)]
        | map({
            id: .,
            type: "FEEDBACK",
            priority: "low",
            source: "build-messy-practice-project.sh",
            created_at: "2020-01-01T00:00:00Z",
            expires_at: "2020-01-31T00:00:00Z",
            active: true,
            strength: 1,
            reason: "journey trap: leftover junk data",
            content: {text: $text},
            content_hash: $hash
          })))
  ' "$pheromones" > "$pheromones.tmp" && mv "$pheromones.tmp" "$pheromones"
}

# apply_trap_unsaved_changes leaves an uncommitted edit to a tracked file
# (README.md) and at least one untracked file (SCRATCH-NOTES.md) in place.
# Never committed -- that is the entire point of this trap.
apply_trap_unsaved_changes() {
  local repo="$1"
  local marker_line="<!-- journey trap: unsaved-changes -->"
  grep -qF "$marker_line" "$repo/README.md" 2>/dev/null || printf '\n%s\n' "$marker_line" >> "$repo/README.md"
  [ -f "$repo/SCRATCH-NOTES.md" ] || printf 'journey trap: unsaved-changes (untracked file)\n' > "$repo/SCRATCH-NOTES.md"
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
    # -tags=journey: cmd/journey_seed.go (the hidden journey-seed-* trap
    # constructors this script calls below) is excluded from the default,
    # released build (CR-01, 207-REVIEW.md) and only compiles in under this
    # tag.
    (cd "$ROOT" && go build -tags=journey -o "$BIN" ./cmd/aether) || fail "go build failed"
  fi
fi

# Isolate the hub, and the home folder every aether command sees -- see the
# ISOLATION RULE above.
export AETHER_HUB_DIR="$DEST/.journey-hub"
mkdir -p "$AETHER_HUB_DIR"
ISOLATED_HOME="$DEST/.journey-home"
mkdir -p "$ISOLATED_HOME"

# in_isolated_home runs one command with HOME pointed at ISOLATED_HOME.
in_isolated_home() { HOME="$ISOLATED_HOME" "$@"; }

step "installing package into isolated hub"
(cd "$ROOT" && in_isolated_home "$BIN" install --package-dir "$ROOT" --home-dir "$ISOLATED_HOME" --skip-build-binary >/dev/null) \
  || fail "aether install failed"

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
  (cd "$REPO" && in_isolated_home "$BIN" init "Practice the daily lifecycle on a messy real project" >/dev/null 2>&1) \
    || fail "aether init failed in the practice project"

  step "aether update --force in the practice project (the install path under test)"
  (cd "$REPO" && in_isolated_home "$BIN" update --force >/dev/null) || fail "aether update --force failed in the practice project"

  # Mark $REPO itself (not just $DEST) as a genuine practice project. The
  # hidden journey-seed-* commands operate directly on $REPO (their own
  # <repo-root> argument), and their own runtime guard
  # (requireJourneyPracticeProjectMarker, cmd/journey_seed.go) refuses to run
  # against any directory lacking this exact marker file -- CR-01
  # (207-REVIEW.md). Written here, before the trap-application step below
  # calls either hidden command, and only when $REPO is genuinely being
  # constructed fresh by this script.
  jq -n --arg schema "$MARKER_SCHEMA_VERSION" \
    '{schema_version: $schema, trap_ids: []}' > "$REPO/.journey-practice-project.json"
else
  step "practice project already exists at $REPO -- re-applying and re-verifying traps only"
fi

# --- Apply every declared trap ---------------------------------------------
#
# Applied in declared file order EXCEPT that git-committing traps
# (long-folder-names, out-of-date-code-map's churn commits) always run
# before the traps that must survive as unsaved changes -- not because
# ordering can leak an uncommitted trap into a commit (every commit this
# script makes stages an explicit, named path, never `-A`/`.`), but so the
# marker file and this section's own log read in an intuitive build order.
# The declared order in traps.json remains the canonical order for trap
# IDENTITY and for the Go test's own failure-reporting order.

step "applying declared traps"
APPLIED_IDS=""
while IFS= read -r trap_id; do
  [ -z "$trap_id" ] && continue
  case "$trap_id" in
    shortcut-to-a-folder)
      apply_trap_shortcut_to_a_folder "$REPO"
      ;;
    shortcut-loop)
      apply_trap_shortcut_loop "$REPO"
      ;;
    case-only-archive-collision)
      apply_trap_case_only_archive_collision "$REPO"
      ;;
    nested-project)
      apply_trap_nested_project "$REPO"
      ;;
    long-folder-names)
      apply_trap_long_folder_names "$REPO"
      ;;
    out-of-date-code-map)
      apply_trap_out_of_date_code_map "$REPO" "$BIN"
      ;;
    specification-corrected-mid-planning)
      apply_trap_specification_corrected_mid_planning "$REPO" "$BIN"
      ;;
    leftover-junk-data)
      apply_trap_leftover_junk_data "$REPO"
      ;;
    unsaved-changes)
      apply_trap_unsaved_changes "$REPO"
      ;;
    *)
      fail "declared trap id has no known construction in this script: $trap_id"
      ;;
  esac
  APPLIED_IDS="${APPLIED_IDS}${APPLIED_IDS:+ }${trap_id}"
done <<<"$TRAP_IDS"

step "verifying declared traps landed"
test -L "$REPO/shortcut-to-src" || fail "shortcut-to-a-folder trap did not land -- shortcut-to-src is not a symbolic link"
test -L "$REPO/loop-a" && test -L "$REPO/loop-b" || fail "shortcut-loop trap did not land"
test -f "$REPO/.aether/HANDOFF.md" && test -f "$REPO/.aether/data/handoff.md" \
  || fail "case-only-archive-collision trap did not land"
test -d "$REPO/nested-project/.git" || fail "nested-project trap did not land"
test -f "$REPO/src/deeply/nested/directory/chain/intentionally/long/enough/to/exceed/the/forty/character/canonical/lineage/cap/file.go" \
  || fail "long-folder-names trap did not land"
test -f "$REPO/.aether/data/survey/territory-snapshot.json" || fail "out-of-date-code-map trap did not land"
test -f "$REPO/.aether/data/planning/journey-trap-run/stage-state.json" \
  || fail "specification-corrected-mid-planning trap did not land"
jq -e '.entries != null and (.entries | map(select(.acknowledged == false)) | length) >= 2' \
  "$REPO/.aether/data/midden.json" >/dev/null \
  || fail "leftover-junk-data trap did not land -- .aether/data/midden.json has fewer than two unacknowledged entries"
jq -e '.signals != null and (.signals | map(select(.active == true)) | length) >= 1' \
  "$REPO/.aether/data/pheromones.json" >/dev/null \
  || fail "leftover-junk-data trap did not land -- .aether/data/pheromones.json has no stray active signal"
git -C "$REPO" status --porcelain | grep -q . || fail "no unsaved changes left -- unsaved-changes trap did not land"

# --- Write the marker and finish -------------------------------------------

jq -n --arg schema "$MARKER_SCHEMA_VERSION" --arg ids "$APPLIED_IDS" \
  '{schema_version: $schema, trap_ids: ($ids | split(" "))}' > "$MARKER"

echo
echo "BUILD PASS: practice project at $REPO (traps applied: $APPLIED_IDS)"
