#!/usr/bin/env bash
# run.sh — the one documented command that reproduces a benchmark run.
#
# Usage:
#   bench/run.sh --lane LANE --category CATEGORY   # run a single cell
#   bench/run.sh --dry-run                          # enumerate the full
#                                                     # twelve-cell matrix
#                                                     # and execute nothing
#
# LANE     — one of: gsd, aether-interactive, aether-autopilot
# CATEGORY — one of: 01-bug-fix, 02-brownfield-feature,
#            03-interrupted-execution, 04-fresh-repo-lifecycle
#
# Requires BENCH_MODEL to be set for anything other than --dry-run.
#
# Refuses to run any real cell if bench/acceptance/verify-predates-runs.sh
# exits non-zero — that is the point at which the acceptance-ordering
# discipline becomes enforced rather than documented. A run that would
# invalidate the ordering claim is blocked before it happens, not detected
# afterwards.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() { echo "RUN FAIL: $*" >&2; exit 1; }

LANES=(gsd aether-interactive aether-autopilot)
CATEGORIES=(01-bug-fix 02-brownfield-feature 03-interrupted-execution 04-fresh-repo-lifecycle)

# --- substrate metadata for the dry-run enumeration ------------------------
_substrate_repo() {
  case "$1" in
    01-bug-fix|03-interrupted-execution) echo "gorilla/mux" ;;
    02-brownfield-feature) echo "colinhacks/zod" ;;
    04-fresh-repo-lifecycle) echo "(empty repo — no substrate SHA)" ;;
  esac
}
_substrate_sha() {
  case "$1" in
    01-bug-fix|03-interrupted-execution) echo "db9d1d0073d27a0a2d9a8c1bc52aa0af4374d265" ;;
    02-brownfield-feature) echo "ca42965df46b2f7e2747db29c40a26bcb32a51d5" ;;
    04-fresh-repo-lifecycle) echo "" ;;
  esac
}

DRY_RUN=0
LANE=""
CATEGORY=""

while [ $# -gt 0 ]; do
  case "$1" in
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    --lane)
      LANE="${2:-}"
      shift 2
      ;;
    --category)
      CATEGORY="${2:-}"
      shift 2
      ;;
    *)
      fail "unrecognized argument: $1 (usage: bench/run.sh --dry-run | bench/run.sh --lane LANE --category CATEGORY)"
      ;;
  esac
done

# --- tool preflight: standard list plus claude and (for the GSD lane) gsd-sdk
for tool in go node npm jq git claude; do
  command -v "$tool" >/dev/null 2>&1 || fail "required tool missing: $tool"
done
if [ "$LANE" = "gsd" ] || [ "$DRY_RUN" -eq 1 ]; then
  command -v gsd-sdk >/dev/null 2>&1 || {
    if [ "$DRY_RUN" -eq 1 ]; then
      echo "NOTE: gsd-sdk not found on PATH — required before running the gsd lane for real" >&2
    else
      fail "required tool missing: gsd-sdk (needed for the gsd lane)"
    fi
  }
fi

# --- --dry-run: enumerate the full twelve-cell matrix, execute nothing -----
if [ "$DRY_RUN" -eq 1 ]; then
  echo "BENCH DRY RUN — enumerating all 12 cells. Nothing below is executed."
  echo
  for lane in "${LANES[@]}"; do
    for category in "${CATEGORIES[@]}"; do
      repo="$(_substrate_repo "$category")"
      sha="$(_substrate_sha "$category")"
      acceptance="bench/acceptance/${category}.sh"
      echo "lane=$lane category=$category repo=$repo sha=${sha:-N/A} acceptance=$acceptance"
      echo "  command: bench/harness/run-cell.sh $lane $category bench/results/\$(date -u +%Y-%m-%d)"
    done
  done
  echo
  echo "DRY RUN COMPLETE: 12 cells enumerated, 0 executed."
  exit 0
fi

# --- real run: validate lane/category, require BENCH_MODEL -----------------
[ -n "$LANE" ] || fail "--lane is required (or pass --dry-run)"
[ -n "$CATEGORY" ] || fail "--category is required (or pass --dry-run)"

case "$LANE" in
  gsd|aether-interactive|aether-autopilot) ;;
  *) fail "unknown lane: '$LANE' — must be one of ${LANES[*]}" ;;
esac

case "$CATEGORY" in
  01-bug-fix|02-brownfield-feature|03-interrupted-execution|04-fresh-repo-lifecycle) ;;
  *) fail "unknown category: '$CATEGORY' — must be one of ${CATEGORIES[*]}" ;;
esac

[ -n "${BENCH_MODEL:-}" ] || fail "BENCH_MODEL is not set — an unpinned model silently breaks the parity claim between lanes"

# --- refuse to run if the acceptance-ordering claim would be broken --------
echo "==> checking acceptance-ordering discipline (bench/acceptance/verify-predates-runs.sh)"
if ! bench/acceptance/verify-predates-runs.sh; then
  fail "bench/acceptance/verify-predates-runs.sh failed — running this cell now would risk invalidating the acceptance-predates-results claim. Fix the ordering violation before running any cell."
fi

RESULTS_DIR="bench/results/$(date -u +%Y-%m-%d)"
mkdir -p "$RESULTS_DIR"

echo "==> running lane=$LANE category=$CATEGORY -> $RESULTS_DIR"
exec bench/harness/run-cell.sh "$LANE" "$CATEGORY" "$RESULTS_DIR"
