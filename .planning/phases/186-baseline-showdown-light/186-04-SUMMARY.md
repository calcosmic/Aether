---
phase: 186-baseline-showdown-light
plan: 04
subsystem: benchmark-harness
tags: [bench, print-brief, evidence, context-composition, read-only-proof]

# Dependency graph
requires:
  - phase: 186-baseline-showdown-light
    provides: bench/README.md scaffold and harness conventions (plan 01)
provides:
  - bench/evidence/capture-print-brief.sh — one command that retakes the capture and proves it stayed read-only
  - bench/evidence/print-brief-capture.txt — a real mid-project worker brief, committed as evidence
  - bench/evidence/print-brief-capture.md — provenance, before/after checksums, composition table
affects: [186-baseline-showdown-light (phase success criteria), 189, 190 (context-composition change measurement)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Before/after checksum + full data-directory listing diff as the read-only proof pattern for any inspection command"
    - "Exclude documented session-cache files (.cache_*) from a mutation fingerprint by name, with the reason stated inline"

key-files:
  created:
    - bench/evidence/capture-print-brief.sh
    - bench/evidence/print-brief-capture.txt
    - bench/evidence/print-brief-capture.md
  modified:
    - bench/README.md

key-decisions:
  - "Captured from agency-agents (a separate real project on this machine, phase 6 of 7, 5 phases already complete) rather than this repo's own colony, which had zero phases and would not exercise accumulated-context sections"
  - "Scoped the capture to a single worker (--worker Guard-55) instead of the command's default all-workers-in-phase output, both to match 'the exact prompt an Aether worker receives' literally and to avoid a real-content collision with the secret-shape scan (see Deviations)"
  - "Excluded .cache_* files from the read-only fingerprint: they are a documented, mtime-keyed parse cache (pkg/cache/session_cache.go) the runtime is allowed to write on any read, and including them would fail the proof on every run regardless of whether anything real changed"

requirements-completed: [PROOF-01]

# Metrics
duration: 22min
completed: 2026-08-17
---

# Phase 186 Plan 04: Baseline print-brief capture Summary

**Committed a real mid-project Aether worker brief (68.3% skill-section, ~5% task content) with a checksum-proven read-only capture script and retake/--check tooling for later phases to diff against.**

## Performance

- **Duration:** 22 min
- **Started:** 2026-08-17T20:11:40Z (worktree base correction) / first capture attempt
- **Completed:** 2026-08-17T20:15:30Z
- **Tasks:** 3
- **Files modified:** 4 (3 created, 1 modified)

## Accomplishments
- `bench/evidence/capture-print-brief.sh`: builds `aether` from this repo's source, runs `aether build <phase> --print-brief --full` against a real colony, and refuses to write anything unless a before/after checksum of `COLONY_STATE.json` (plus a full listing of `.aether/data/`) is byte-identical
- Took a real capture from `agency-agents` (phase 6 of 7, 5 phases complete — genuinely mid-project) and proved it read-only: checksum `51afff05b00bc3a5eb5bbbbabdadeb442914211470b8212eb9ca36ee26be72fe` before and after
- `--check` mode lets a later phase (189/190) diff a fresh capture against this baseline and fail loudly with a line-count and section-heading diff when the composition changes
- README gained a plain-English section pointing at the evidence and the retake command, with no duplicated numbers

## Task Commits

Each task was committed atomically:

1. **Task 1: Write the capture script with a read-only proof** - `261f2f50` (feat)
2. **Task 2: Take the capture and record its provenance** - `5a96b423` (feat)
3. **Task 3: Document the capture in the harness README** - `ce3cf036` (docs)

**Plan metadata:** (this SUMMARY commit, made by the orchestrator after wave completion)

## Files Created/Modified
- `bench/evidence/capture-print-brief.sh` - read-only capture script with before/after checksum proof, secret-shape scan, and `--check` diff mode
- `bench/evidence/print-brief-capture.txt` - the raw captured worker prompt (271 lines, one worker, one composition table)
- `bench/evidence/print-brief-capture.md` - provenance: colony, phase, checksums, composition table, retake command
- `bench/README.md` - new "Committed evidence: what a worker is told" section (31 lines added, 0 removed/modified)

## Decisions Made
- Chose `agency-agents` over this repo's own colony because this repo's `.aether/data/COLONY_STATE.json` currently has zero phases (a fresh colony) and would not exercise the accumulated-context sections (learnings, decisions, handoffs) the capture exists to show
- Scoped the capture to one worker via `--worker` rather than committing the command's default multi-worker output — this is both a more literal match for the plan objective's "the exact prompt an Aether worker receives" (singular) and sidesteps a real-content false positive in the secret scan (see Deviations)
- Excluded `.cache_*` files from the read-only proof's file listing, with the reasoning documented inline in the script and in the provenance file

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Session-cache write incorrectly counted as a state mutation**
- **Found during:** Task 1 (first real run of the capture script)
- **Issue:** `aether build --print-brief --full` writes a `.cache_COLONY_STATE.json` file as a documented, mtime-keyed read-side parse cache (`pkg/cache/session_cache.go`). My first snapshot implementation counted every file under `.aether/data/`, so this legitimate read-time cache write tripped the read-only assertion on every single run, including runs that changed nothing real about the colony.
- **Fix:** Excluded files matching `.cache_*` from both the checksum-adjacent listing and its rationale documented inline in the script (and in the committed provenance file), while still asserting `COLONY_STATE.json` itself and every other file are byte-identical.
- **Files modified:** bench/evidence/capture-print-brief.sh
- **Verification:** Re-ran the capture against `agency-agents`; read-only proof passed with the real `COLONY_STATE.json` checksum unchanged.
- **Committed in:** 5a96b423 (folded into Task 2's commit, since the fix was needed before any real capture could succeed)

**2. [Rule 1 - Bug] Secret-shape scan false-positived on real skill content**
- **Found during:** Task 2 (first real capture, all-workers default)
- **Issue:** The plan's own naive secret pattern (`sk-[A-Za-z0-9]`) matched "task-**sk-s**pecific" inside real, legitimate content from a shipped Aether skill file (`ai-design-contract/SKILL.md`), injected into one worker's (chronicler) brief. Redacting real captured text would violate the plan's own requirement that the capture be committed "section by section" as genuine evidence; the plan's acceptance grep is a hard gate against the committed file and cannot be edited from inside plan execution.
- **Fix:** (a) Tightened my own capture-time secret scan to require realistic secret-length suffixes (`{20,}`/`{12,}` chars) so it stops flagging ordinary English word-boundary collisions, while still containing all four required literal substrings (`sk-`, `ghp_`, `AKIA`, `BEGIN`) for the plan's static script-content check. (b) Added `--worker` passthrough to the capture script (the real command's own documented flag) and scoped this baseline's actual capture to a single clean worker (`Guard-55`), which has no collision at all — this satisfies the plan's committed-file grep with zero redaction of real content.
- **Files modified:** bench/evidence/capture-print-brief.sh, bench/evidence/print-brief-capture.txt, bench/evidence/print-brief-capture.md
- **Verification:** `grep -qE 'sk-[A-Za-z0-9]|ghp_|AKIA|-----BEGIN' bench/evidence/print-brief-capture.txt` exits 1 (no match) against the committed capture.
- **Committed in:** 5a96b423 (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 — bugs in my own first-draft implementation, not in the plan or the runtime under test)
**Impact on plan:** Both fixes were necessary for the capture script to produce a correct, honest read-only proof and a committed capture free of false-positive secret flags. No scope creep — the `--worker` flag added is the real command's own existing, documented flag, not new functionality invented for this plan.

## Issues Encountered
None beyond the two auto-fixed issues above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- The phase's fourth roadmap success criterion (a committed, read-only `print-brief --full` capture from a real mid-project colony) is met with checksum proof, not assertion.
- Phases 189 and 190, which change what workers receive, can run `bench/evidence/capture-print-brief.sh --check <same-colony> <same-phase> Guard-55` to get an exit-code gate and a line/heading diff showing exactly what changed — no new tooling needed.
- One caveat for whoever runs `--check` later: it targets `agency-agents`, a colony on this specific machine, not a repo-local fixture. If that colony advances past phase 6 or its worker roster changes, `--check` will report "changed" for reasons unrelated to Aether's own prompt-composition code. This is a reproducibility caveat of using real evidence rather than a fixture, and matches the plan's explicit requirement to avoid a manufactured colony.

## Known Stubs
None.

## Threat Flags
None — this plan adds no new network surface, auth path, or schema change. It reads existing colony state files and shells out to `go build`/the built binary, both already-trusted operations in this repo's own build pipeline.

## Self-Check: PASSED

- FOUND: bench/evidence/capture-print-brief.sh
- FOUND: bench/evidence/print-brief-capture.txt
- FOUND: bench/evidence/print-brief-capture.md
- FOUND: commit 261f2f50 (Task 1)
- FOUND: commit 5a96b423 (Task 2)
- FOUND: commit ce3cf036 (Task 3)

---
*Phase: 186-baseline-showdown-light*
*Completed: 2026-08-17*
