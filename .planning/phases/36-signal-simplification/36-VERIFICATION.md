---
phase: 36-signal-simplification
verified: 2026-02-06T19:30:00Z
status: passed
score: 4/4 success criteria verified
re_verification:
  previous_status: gaps_found
  previous_score: 2/4
  gaps_closed:
    - "runtime/aether-utils.sh decay commands removed (now 317 lines, matches .aether/aether-utils.sh)"
    - "runtime/workers/*.md files deleted (glob returns no files)"
    - "commands/ant/plan.md updated to use TTL-based filtering"
    - "commands/ant/organize.md updated to use TTL-based filtering"
    - "commands/ant/colonize.md updated to use TTL-based filtering"
  gaps_remaining: []
  regressions: []
---

# Phase 36: Signal Simplification Verification Report

**Phase Goal:** Pheromone system uses simple TTL instead of exponential decay
**Verified:** 2026-02-06T19:30:00Z
**Status:** passed
**Re-verification:** Yes -- after gap closure (plans 36-04 and 36-05)

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Signals use expires_at timestamp instead of half-life math | VERIFIED | focus.md:59, redirect.md:59, feedback.md:59 all use `expires_at` field; validate-state pheromones checks for expires_at (lines 79-85) |
| 2 | Priority field (high/normal/low) replaces sensitivity matrix calculations | VERIFIED | focus.md uses priority: normal, redirect.md uses priority: high, feedback.md uses priority: low; signal consumers filter by priority not sensitivity |
| 3 | Expired signals filtered on read (no cleanup command needed) | VERIFIED | build.md:73-82, continue.md:443-456, status.md:110-118, plan.md:31-36, organize.md:27-32, colonize.md:29-35 all use TTL-based filtering |
| 4 | All pheromone math removed from aether-utils.sh | VERIFIED | Both runtime/aether-utils.sh and .aether/aether-utils.sh are 317 lines, identical, with no decay/effective/batch/cleanup commands |

**Score:** 4/4 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `runtime/aether-utils.sh` | Decay commands removed | VERIFIED | 317 lines, help lists 14 commands without decay/effective/batch/cleanup |
| `.aether/aether-utils.sh` | Decay commands removed | VERIFIED | 317 lines, identical to runtime version |
| `commands/ant/focus.md` | TTL-based schema | VERIFIED | Uses expires_at, priority: normal, --ttl flag parsing |
| `commands/ant/redirect.md` | TTL-based schema | VERIFIED | Uses expires_at, priority: high, --ttl flag parsing |
| `commands/ant/feedback.md` | TTL-based schema | VERIFIED | Uses expires_at, priority: low, --ttl flag parsing |
| `commands/ant/build.md` | TTL filtering | VERIFIED | Step 3 filters by expires_at, displays priority |
| `commands/ant/continue.md` | TTL filtering | VERIFIED | Step 5 filters phase_end signals on advance |
| `commands/ant/status.md` | TTL filtering | VERIFIED | Filters and displays with priority grouping |
| `commands/ant/pause-colony.md` | paused_at tracking | VERIFIED | Records paused_at timestamp |
| `commands/ant/resume-colony.md` | TTL extension | VERIFIED | Extends wall-clock TTLs by pause duration |
| `commands/ant/plan.md` | TTL filtering | VERIFIED | Lines 31-36 use TTL-based filtering, no pheromone-batch |
| `commands/ant/organize.md` | TTL filtering | VERIFIED | Lines 27-32 use TTL-based filtering, no pheromone-batch |
| `commands/ant/colonize.md` | TTL filtering | VERIFIED | Lines 29-35 use TTL-based filtering, no pheromone-batch |
| `.aether/docs/pheromones.md` | TTL documentation | VERIFIED | 206 lines, documents TTL system, priority levels, --ttl flag |
| `.aether/workers.md` | Priority-based guidance | VERIFIED | 172 lines, no sensitivity matrices, lists signal types per role |
| `runtime/workers/*.md` | Deleted | VERIFIED | Glob returns "No files found" |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| focus.md | pheromones.json | Write tool | WIRED | Writes expires_at + priority |
| redirect.md | pheromones.json | Write tool | WIRED | Writes expires_at + priority |
| feedback.md | pheromones.json | Write tool | WIRED | Writes expires_at + priority |
| build.md | pheromones.json | Read + filter | WIRED | TTL filtering in Step 3 |
| continue.md | pheromones.json | Read + filter | WIRED | TTL filtering in Step 5 |
| status.md | pheromones.json | Read + filter | WIRED | TTL filtering with priority grouping |
| plan.md | pheromones.json | Read + filter | WIRED | TTL filtering at lines 31-36 |
| organize.md | pheromones.json | Read + filter | WIRED | TTL filtering at lines 27-32 |
| colonize.md | pheromones.json | Read + filter | WIRED | TTL filtering at lines 29-35 |
| validate-state pheromones | pheromones.json | jq | WIRED | Checks for id, type, content, priority, created_at, expires_at |

### Requirements Coverage

| Requirement | Status | Notes |
|-------------|--------|-------|
| SIMP-03: Replace pheromone exponential decay with simple TTL | SATISFIED | All signal commands use TTL schema, all consumers filter by TTL |

### Anti-Patterns Found

None. All decay-related code has been removed from active command and utility files.

### Known Exceptions (Acceptable)

| File | Pattern | Severity | Why Acceptable |
|------|---------|----------|----------------|
| `~/.aether/workers/*.md` | pheromone-effective calls | INFO | User's installed copy from previous init; will be updated on next `/ant:init` |
| `.claude/commands/ant/plan.md` | pheromone-batch reference | INFO | Archive copy in .claude directory; not actively used |
| `.planning/**/*.md` | References to old system | INFO | Documentation/planning files documenting the migration |

### Human Verification Required

None for this phase -- all verification can be done programmatically.

## Gaps Summary

**All gaps from previous verification have been closed:**

1. **runtime/aether-utils.sh** -- Now 317 lines, identical to .aether/aether-utils.sh. No decay commands.

2. **runtime/workers/*.md** -- All 6 files deleted. Workers now use consolidated `.aether/workers.md`.

3. **commands/ant/{plan,organize,colonize}.md** -- All updated with TTL-based filtering. No pheromone-batch calls.

## Note on User-Installed Files

The worker spec files in `~/.aether/workers/` (user's home directory) still contain the old sensitivity matrix and pheromone-effective patterns. These were installed by a previous `/ant:init` run and will be updated when the user next initializes a colony.

This is expected behavior -- the repo source is correct, but user installations retain old versions until re-initialized. This is NOT a phase 36 gap; it's a deployment consideration.

---

*Verified: 2026-02-06T19:30:00Z*
*Verifier: Claude (cds-verifier)*
*Re-verification: Gaps from previous verification closed*
